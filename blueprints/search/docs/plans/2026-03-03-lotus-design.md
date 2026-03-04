# Lotus: tantivy-in-Go — Design Document

**Date:** 2026-03-03
**Branch:** index-pane
**Driver name:** `lotus`
**Package:** `pkg/index/driver/flower/lotus/`
**Status:** approved

---

## Goal

Implement a pure-Go full-text search engine named **Lotus** that deeply reproduces
tantivy's internal architecture: FST term dictionary, BP128 block posting compression,
memory-mapped segment reads, positions for phrase queries, BM25F scoring, and
Block-Max WAND dynamic pruning.

Benchmark target: approach tantivy's 2.7 ms p50 on 5 M Wikipedia docs while keeping
peak RSS < 1 GB (vs rose's 22.4 GB) by virtue of memory-mapped, disk-resident index.

---

## Flower name

**Lotus** — clean-room, from-scratch philosophy; the lotus grows from the mud untouched.

---

## Tantivy ideas we implement

| Tantivy feature | Lotus implementation |
|-----------------|---------------------|
| FST term dictionary (`fst` crate / vellum) | `vellum` (production) + scratch FST (research, build-tagged `lotusfstscratch`) |
| SIMD-BP128 posting compression | Pure-Go BP128: 128-int blocks, 1-byte bit-width header, b×16 bytes data |
| Memory-mapped segments | `golang.org/x/sys/unix.Mmap` (darwin + linux) |
| Skip list for `advance(docID)` | Per-block skip entries stored after posting data in `.doc` |
| Positions file for phrase queries | `.pos` file, BP128-encoded position deltas |
| Stored fields with zstd compression | 512-doc blocks, per-block zstd, block offset index |
| Field norms (doc-length normalization) | `.fnm` u8 array, Lucene SmartNorm encoding |
| Multi-segment architecture + merge | Tiered merge: ≥ 3 segs at same tier → background merge goroutine |
| BM25F scoring | Robertson BM25+ (k1=1.2, b=0.75, delta=1.0), u8 impact quantization |
| Block-Max WAND | Block-max impacts from skip list, min-heap threshold pruning |
| Phrase query | Position adjacency check via positional cursor intersection |
| Boolean query | must / should / must_not with WAND for should terms |
| Delete bitset | `.del` file (future; stub for now) |

---

## On-disk format

### `lotus.meta` (root, JSON)

```json
{
  "version": 1,
  "doc_count": 5032104,
  "avg_doc_len": 412.7,
  "segments": ["a1b2c3d4", "e5f6g7h8"],
  "next_seg_seq": 42
}
```

### Per-segment directory `{uuid}/`

| File | Content |
|------|---------|
| `.tdi` | vellum FST bytes: term → TermInfo packed into uint64 |
| `.doc` | BP128-encoded docID delta blocks + skip index footer |
| `.freq` | BP128-encoded term frequency blocks (parallel with `.doc`) |
| `.pos` | BP128-encoded position delta blocks (parallel with `.doc`) |
| `.store` | Stored fields: zstd-compressed 512-doc blocks + block offset array |
| `.fnm` | Raw u8[docCount] field norms |
| `.del` | Future delete bitset (stub, empty file) |
| `.meta` | JSON: docCount, avgDocLen |

---

## BP128 block posting compression

Tantivy's SIMD-BP128 uses 128-integer blocks with bit-packing:

```
For a block of 128 deltas d[0..127]:
  b = bits_required(max(d[i]))  -- 1..32
  header: 1 byte containing b
  data:   b × 16 bytes (128 integers × b bits, packed)

Total per block: 1 + b×16 bytes (vs VByte: ~140–256 bytes for 128 integers)

Skip index (appended after all posting data):
  for each block i:
    lastDoc[i]   uint32  -- last docID in block i
    docOff[i]    uint32  -- byte offset of block i in .doc
    posOff[i]    uint32  -- byte offset of matching .pos block
  numBlocks      uint32  -- count, stored at file end (4 bytes)
```

In Go without SIMD: still ~3–5× faster than VByte decode due to cache-friendly
packed memory layout and loop-unrollable bit extraction.

---

## FST term dictionary

**TermInfo** packed into vellum's `uint64` output value:

```
bits [0..30]  = doc_freq       (u31, supports up to 2B matching docs)
bit  [31]     = has_positions  (1 = .pos file has data for this term)
bits [32..62] = postingsOffset (u31, byte offset in .doc / .freq / .pos)
bit  [63]     = reserved
```

**Build path:** `TermDictWriter` collects `(term, TermInfo)` pairs in sorted order
during segment flush, then writes the vellum FST in one pass.

**Scratch FST:** `pkg/index/driver/flower/lotus/fst/` contains a pure-Go MADA
(Minimal Acyclic Deterministic Automaton) using Daciuk's incremental suffix-sharing
algorithm. Activated by build tag `lotusfstscratch`. Both implementations satisfy
the `TermDict` interface so callers are unaware of the swap.

---

## Text analysis pipeline

```
Raw text
  → unicode tokenize: split on \P{L}\P{N}+, keep len ∈ [2, 64]
  → lowercase: unicode.ToLower (handles accented chars)
  → stopword filter: 127 English words (Lucene-compatible list)
  → Snowball stem: kljensen/snowball English
  → record (term, position) pairs
```

**Position tracking:** every token records its absolute position (0-indexed) in the
document. Positions are stored per-term in a parallel list alongside docID+freq.

---

## Query model

```go
type Query interface{ query() }

type TermQuery   struct{ Field, Term string }
type PhraseQuery struct{ Field string; Terms []string; Slop int }
type BooleanQuery struct {
    Must    []Query
    Should  []Query
    MustNot []Query
}
```

**Parse rules (bench-compatible):**

| Input shape | Parsed as |
|-------------|-----------|
| `"+a +b"`   | BooleanQuery{Must: [a, b]} |
| `"a b"`     | BooleanQuery{Should: [a, b]} |
| `"-a +b"`   | BooleanQuery{Must: [b], MustNot: [a]} |
| `'"a b c"'` | PhraseQuery{Terms: [a, b, c]} |

---

## Scoring: BM25F

Robertson BM25+ with quantized u8 block-max impacts:

```
IDF(t)       = log((N - df + 0.5) / (df + 0.5) + 1)
TF_norm(t,d) = tf × (k1 + 1) / (tf + k1 × (1 - b + b × dl/avgdl))
BM25+(t,d)   = IDF(t) × TF_norm(t,d) + delta

k1=1.2  b=0.75  delta=1.0

Quantize: impact_u8 = clamp(round(bm25+ / list_max × 255), 1, 255)
Block-max impact stored in skip entry (1 byte per block per term)
```

---

## Block-Max WAND retrieval

Same algorithm as rose's WAND, but skip entries are in-memory from the skip index
(loaded from the mmap'd `.doc` file), so block-skipping is zero-copy.

For phrase queries: after BM25F identifies candidate docIDs, a secondary position
check advances all per-term position cursors and verifies adjacency.

---

## Concurrency model

```
IndexWriter   — single goroutine (no mutex needed for writes)
  │
  └── SegmentWriter.flush() → atomic rename to {uuid}/ → update lotus.meta
                                    ↓
                           mergeWorker goroutine (background)
                           wakes every 5s or on flush signal

IndexReader   — snapshot of lotus.meta at Open() time
  └── Searcher — concurrent-safe (mmap is read-only, shared)
```

---

## File layout

```
pkg/index/driver/flower/lotus/
  RESEARCH.md       ← tantivy internals deep-dive + paper references
  engine.go         ← Engine: Open/Close/Stats/Index/Search; init() "lotus"
  writer.go         ← SegmentWriter: accumulate → flush segment dir
  reader.go         ← SegmentReader: mmap open; term lookup; cursor creation
  bp128.go          ← Pack128/Unpack128/EncodeAll/DecodeAll; skiplist build
  postings.go       ← PostingIterator (docID + freq + pos advance)
  termdict.go       ← TermDictWriter (vellum) + TermDictReader interface
  fst/              ← scratch MADA FST (build tag: lotusfstscratch)
    builder.go
    fst.go
    node.go
  store.go          ← StoredFieldsWriter/Reader (zstd 512-doc blocks)
  fieldnorms.go     ← u8 Lucene-style norm encode/decode
  merge.go          ← N-way segment merge; tiered policy; background goroutine
  query.go          ← Query types; parse(); BooleanQuery/TermQuery/PhraseQuery
  scorer.go         ← BM25F; quantize; WAND evaluator; phrase checker
  analyzer.go       ← tokenize + lowercase + stopword + Snowball stem
  mmap_unix.go      ← mmap/munmap (//go:build linux || darwin)
  mmap_other.go     ← fallback: os.ReadFile (//go:build !linux && !darwin)
```

---

## Benchmark plan

Corpus: English Wikipedia, 5,032,104 docs.
Machine: Apple M-series darwin/arm64, local NVMe.

### Index benchmark

| Engine | Docs/s target | Peak RSS target | Disk target |
|--------|--------------|----------------|-------------|
| lotus  | ≥ 10,000     | ≤ 600 MB       | ≤ 5 GB      |
| tantivy (CGO) | ~11,800 | ~278 MB  | ~7.3 GB     |

### Search benchmark (TOP_10, 962 queries, 10 iter, 30s warmup)

| Engine | p50 target | p95 target |
|--------|-----------|-----------|
| lotus  | ≤ 5 ms    | ≤ 8 ms    |
| tantivy (CGO) | 2.7 ms | 3.4 ms |

---

## Dependencies

| Package | Already in go.mod? | Purpose |
|---------|-------------------|---------|
| `github.com/blevesearch/vellum` | No — add | FST term dictionary |
| `github.com/klauspost/compress/zstd` | Yes | Stored fields compression |
| `github.com/kljensen/snowball` | Yes | Stemming |
| `golang.org/x/sys/unix` | Check | mmap syscalls |

---

## Non-goals

- No HTTP server / external service
- No numeric/date fast fields (future)
- No fuzzy query (future — requires Levenshtein automaton over FST)
- No update/delete support beyond the stub `.del` file
