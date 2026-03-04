# spec/0650 — Lotus: tantivy-in-Go Full-Text Search Engine

**Status:** implementing
**Branch:** index-pane
**Driver name:** `lotus`
**Package:** `pkg/index/driver/flower/lotus/`

---

## Goal

Implement a pure-Go full-text search engine named **Lotus** that faithfully reproduces
tantivy's core architecture:

- **FST term dictionary** (vellum, with scratch FST for comparison)
- **BP128 block posting compression** (128-int bitpacked blocks)
- **Memory-mapped segment reads** (zero-copy, OS-managed page cache)
- **Position-indexed phrase queries** (separate .pos file)
- **BM25F scoring** with Robertson BM25+ (k1=1.2, b=0.75, delta=1.0)
- **Block-Max WAND** dynamic pruning (skip entire 128-doc blocks)
- **Lucene SmallFloat field norms** (1 byte per doc, 256-entry BM25 table)
- **Zstd-compressed stored fields** (16KB blocks with skip index)
- **LSM-style tiered segment merge** (background goroutine)

Target: approach tantivy's 2.7 ms p50 search latency on 5M Wikipedia docs while
keeping peak RSS < 1 GB (vs rose's 22.4 GB) via disk-resident mmap'd index.

---

## Tantivy Ideas Reproduced

| Tantivy component | Lotus implementation |
|-------------------|---------------------|
| `fst` crate (BurntSushi) | `blevesearch/vellum` (production) + scratch FST (build tag `lotusfstscratch`) |
| SIMD-BP128 (`bitpacking` crate) | Pure-Go BP128: 128-int blocks, 1-byte bit-width header, scalar bit extraction |
| Memory-mapped segments | `golang.org/x/sys/unix.Mmap` (darwin + linux), fallback `os.ReadFile` |
| Skip list in `.idx` | Per-block skip entries appended after posting data in `.doc` |
| Positions (`.pos`) | BP128-encoded position deltas, separate file |
| Stored fields (`.store`) | 16KB blocks, zstd compressed, skip index + footer |
| Field norms (`.fieldnorm`) | Lucene `intToByte4` encoding, u8[docCount], 256-entry BM25 table |
| Tiered merge | ≥ 3 segments at same tier → background merge goroutine |
| BM25F scoring | Robertson BM25+ with u8 impact quantization |
| Block-Max WAND | Block-max impacts from skip entries, min-heap threshold pruning |
| Phrase query | Position adjacency check via positional cursor intersection |
| Boolean query | must / should / must_not with WAND for should terms |
| Delete bitset | `.del` file stub (future) |

---

## On-Disk Format

### Root: `lotus.meta` (JSON)

```json
{
  "version": 1,
  "doc_count": 5032104,
  "avg_doc_len": 412.7,
  "segments": ["seg_00000001", "seg_00000002"],
  "next_seg_seq": 3
}
```

### Per-Segment Directory: `seg_{NNNNNNNN}/`

| File | Content |
|------|---------|
| `segment.tdi` | Vellum FST bytes: term → packed termInfo (docFreq + postingsOff + hasPositions) |
| `segment.doc` | BP128-encoded docID delta blocks + skip index footer |
| `segment.freq` | BP128-encoded term frequency blocks (parallel with .doc) |
| `segment.pos` | BP128-encoded position delta blocks |
| `segment.store` | Zstd-compressed stored fields: 16KB blocks + skip index + footer |
| `segment.fnm` | Raw u8[docCount] field norms (Lucene SmallFloat) |
| `segment.meta` | JSON: docCount, avgDocLen |

### BP128 Block Format

```
For a block of 128 deltas d[0..127]:
  numBits = bitsNeeded(max(d))           -- 0..32
  header:  1 byte (numBits)
  data:    numBits × 16 bytes            -- 128 integers × numBits bits, packed

Total per block: 1 + numBits×16 bytes
  vs VByte: ~140–256 bytes for 128 integers

Skip index (appended after all blocks in .doc):
  for each full block i:
    lastDoc[i]     uint32   -- last docID in block
    docByteOff[i]  uint32   -- byte offset of block in .doc
    freqByteOff[i] uint32   -- byte offset of block in .freq
    posByteOff[i]  uint32   -- byte offset of block in .pos
    blockMaxTF[i]  uint32   -- max term freq in block (for Block WAND)
    blockMaxNorm[i] uint8   -- fieldnorm byte of shortest doc
  numBlocks        uint32   -- count, stored at very end
```

### FST TermInfo Packing

```
uint64 value stored in vellum FST:
  bits [0..30]  = docFreq       (u31, max 2B docs)
  bit  [31]     = hasPositions  (1 = .pos has data for this term)
  bits [32..63] = postingsOff   (u32, byte offset in .doc/.freq/.pos)
```

### Stored Fields Format

```
[Block 0 (zstd)] [Block 1 (zstd)] ... [Block N] [Skip Index] [Footer]

Block: zstd-compressed serialized docs
  Per doc: [idLen u32][id bytes][textLen u32][text bytes]

Skip Index: []storeSkipEntry{lastDoc uint32, blockOffset uint32}

Footer (last 8 bytes): skipIndexOffset uint64
```

### Field Norm Encoding

Lucene SmallFloat `intToByte4`:
- Values 0–23: lossless (identity mapping)
- Values ≥ 24: 3-bit mantissa + variable exponent (~12.5% precision)
- Monotonic: larger token count → larger byte value
- At search time: 256-entry precomputed table `K1*(1-B+B*dl/avgdl)`

---

## Text Analysis Pipeline

```
Raw text
  → Unicode tokenize: split on \P{L}\P{N}+, keep len ∈ [2, 64]
  → Lowercase: unicode.ToLower
  → Stopword filter: 127 English words (Lucene-compatible)
  → Snowball stem: kljensen/snowball English
  → Record (term, position) pairs
```

Position = 0-indexed absolute offset of original token (before stopword removal).
Critical for phrase query adjacency checks.

---

## Query Model

```
"+a +b"        → BooleanQuery{must: [a, b]}           -- intersection
"a b"          → BooleanQuery{should: [a, b]}          -- union
"+a -b"        → BooleanQuery{must: [a], mustNot: [b]} -- AND NOT
'"a b c"'      → PhraseQuery{terms: [a, b, c]}         -- exact phrase
```

Query types:
- `termQuery{term string}` — single term lookup
- `phraseQuery{terms []string}` — positional adjacency check
- `booleanQuery{must, should, mustNot []query}` — boolean combination

---

## Scoring: BM25F

Robertson BM25+ (Kamphuis et al. ECIR 2020):

```
IDF(t)       = log((N - df + 0.5) / (df + 0.5) + 1)
TF_norm(t,d) = tf × (k1 + 1) / (tf + k1 × (1 - b + b × dl/avgdl))
BM25+(t,d)   = IDF(t) × TF_norm(t,d) + delta

k1 = 1.2,  b = 0.75,  delta = 1.0
```

Quantization at flush time:
```
impact_u8 = clamp(round(bm25+ / list_max × 255), 1, 255)
```

Block-max impact stored in skip entry (1 byte per block per term).
At query time: WAND uses block-max impacts to skip non-competitive blocks.

---

## Concurrency

```
IndexWriter   — single goroutine, no mutex needed for writes
  └── SegmentWriter.flush() → atomic dir write → update lotus.meta
                                    ↓
                           mergeWorker (background goroutine, tiered)

IndexReader   — snapshot of lotus.meta, immutable segments
  └── Searcher — concurrent-safe via mmap (read-only, shared)
```

---

## File Layout

```
pkg/index/driver/flower/lotus/
  RESEARCH.md       ← tantivy internals deep-dive
  engine.go         ← Engine: Open/Close/Stats/Index/Search; init() "lotus"
  writer.go         ← SegmentWriter: accumulate → flush segment dir
  reader.go         ← SegmentReader: mmap open; term lookup; cursor creation
  bp128.go          ← Pack128/Unpack128 (scalar bitpacking)
  vint.go           ← VInt variable-byte codec (last-block encoding)
  postings.go       ← PostingsWriter/Reader + PostingIterator (BP128 + skip)
  termdict.go       ← TermDictWriter (vellum) + TermDictReader
  store.go          ← StoredFieldsWriter/Reader (zstd 16KB blocks)
  fieldnorms.go     ← Lucene SmallFloat encode/decode + BM25 table
  merge.go          ← N-way segment merge; tiered policy; background loop
  query.go          ← Query types + parser (term/phrase/boolean)
  scorer.go         ← BM25F + quantize + dequantize
  wand.go           ← Block-Max WAND evaluator + phrase checker
  analyzer.go       ← tokenize + lowercase + stopword + Snowball stem
  mmap_unix.go      ← mmap/munmap (linux || darwin)
  mmap_other.go     ← fallback: os.ReadFile (!linux && !darwin)
  fst/              ← scratch FST (Daciuk's MADA, build tag: lotusfstscratch)
    builder.go
    fst.go
    node.go
```

---

## Dependencies

| Package | In go.mod? | Purpose |
|---------|-----------|---------|
| `github.com/blevesearch/vellum` | Yes (indirect) | FST term dictionary |
| `github.com/klauspost/compress/zstd` | Yes | Stored fields compression |
| `github.com/kljensen/snowball` | Yes | Stemming |
| `golang.org/x/sys/unix` | Yes (indirect) | mmap syscalls |

No new dependencies needed — all are already transitively present via bleve/klauspost.

---

## Benchmark Plan

Corpus: English Wikipedia, 5,032,104 docs (~5 M).
Machine: Apple M-series (darwin/arm64), local NVMe.
Commands: `search bench index --engine lotus`, `search bench search --engine lotus`.

### Index Performance

| Engine | Docs | Index time | Rate (docs/s) | Disk | Peak RSS |
|--------|-----:|-----------|--------------|-----:|--------:|
| lotus  | 5,032,104 | — | — | — | — |
| tantivy (CGO) | 5,032,104 | 7m6s | 11,808 | 7.3 GB | 278 MB |
| rose   | 5,032,104 | 6m40s | 12,563 | 3.2 GB | 22.4 GB |

### Search Performance (TOP_10, 962 queries, 10 iter, 30s warmup)

| Engine | p50 | p95 | p99 | Slowest |
|--------|----:|----:|----:|---------|
| lotus  | — | — | — | — |
| tantivy (CGO) | 2.7 ms | 3.4 ms | 3.4 ms | 227.7 ms |
| rose   | 14.4 ms | 16.1 ms | 16.1 ms | 6.8 s |

### Targets

| Metric | Target | Rationale |
|--------|--------|-----------|
| Index throughput | ≥ 10,000 docs/s | Match rose's optimized rate |
| Search p50 | ≤ 5 ms | Within 2× of tantivy |
| Search p95 | ≤ 8 ms | Within 2.5× of tantivy |
| Peak RSS (index) | ≤ 600 MB | 2× tantivy, 37× better than rose |
| Disk size | ≤ 5 GB | Comparable to tantivy |

---

## Implementation Order

1. RESEARCH.md — tantivy internals deep-dive
2. BP128 codec — foundational compression
3. VInt codec — last-block encoding
4. Field norms — Lucene SmallFloat encoding
5. Analyzer — tokenize + stem + positions
6. mmap utilities — platform-specific
7. Term dictionary — vellum FST wrapper
8. Posting lists — BP128 + skip index writer/reader
9. Stored fields — zstd block writer/reader
10. Segment writer — flush orchestrator
11. Segment reader — mmap + cursors
12. BM25F scorer — scoring + quantization
13. Query parser — term/phrase/boolean
14. WAND evaluator — block-max pruning + phrase check
15. Segment merge — N-way tiered
16. Engine — index.Engine impl + registration
17. Scratch FST — Daciuk's MADA (build-tagged)
18. Registration + integration test
19. Spec document
20. Wikipedia benchmark + results
