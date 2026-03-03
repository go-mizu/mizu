# spec/0646 — Rose: Research-grade Original Search Engine

**Status:** implementing
**Branch:** index-pane
**Driver name:** `rose`
**Package:** `pkg/index/driver/flower/rose/`

---

## Goal

Implement a pure-Go, from-scratch full-text search engine driver named **Rose** that
incorporates state-of-the-art techniques from 2014–2025 IR research.  Rose targets:

- **Highest indexing throughput** among embedded pure-Go drivers (target: beat bleve's 2,288 docs/s)
- **Lowest search latency** for top-10 queries (target: approach tantivy's 2ms avg)
- **Minimal disk footprint** via VByte delta compression + zstd block compression
- **Streaming ingest** — no corpus-wide RAM requirement; LSM-style segment flush + merge

Rose implements the `pkg/index.Engine` interface identically to every other driver and
self-registers under the name `"rose"` via `init()`.

---

## Research Foundation

### Selected Papers

The following papers directly inform Rose's design.  All three candidate approaches are
documented below; the **implemented** approach is **B**.

#### Compression Codecs

1. **Pibiri & Venturini — "Techniques for Inverted Index Compression"**
   ACM Computing Surveys 53(6), 2021.
   [DOI:10.1145/3415148](https://dl.acm.org/doi/10.1145/3415148) |
   [arXiv:1908.10598](https://arxiv.org/abs/1908.10598)
   Definitive survey of every integer codec used in inverted indexes: unary, gamma, delta,
   Rice, VByte, Masked-VByte, StreamVByte, PForDelta, FastPFOR, Simple8b, Elias-Fano, and
   SIMD variants. Benchmarks each codec on real posting lists and characterises the
   space-time tradeoff. *Foundation for the codec choice in all three approaches.*

2. **Ottaviano & Venturini — "Partitioned Elias-Fano Indexes"**
   SIGIR 2014.
   [DOI:10.1145/2600428.2609615](https://dl.acm.org/doi/10.1145/2600428.2609615) |
   [PDF](http://groups.di.unipi.it/~ottavian/files/elias_fano_sigir14.pdf)
   Partitions each posting list into variable-size blocks, EF-compresses each block
   independently to capture local density variation.  Achieves near-optimal
   bits-per-integer with O(1) random access.  *Basis for Approach A.*

#### Top-k Retrieval Algorithms

3. **Grand, Muir, Ferenczi, Lin — "From MaxScore to Block-Max Wand: The Story of How
   Lucene Significantly Improved Query Evaluation Performance"**
   ECIR 2020 (Reproducibility).
   [DOI:10.1007/978-3-030-45442-5_3](https://link.springer.com/chapter/10.1007/978-3-030-45442-5_3) |
   [PDF](https://cs.uwaterloo.ca/~jimmylin/publications/Grand_etal_ECIR2020_preprint.pdf)
   Traces the 8-year journey from the academic BMW paper to its production deployment in
   Apache Lucene 8.0.  Quantifies the 3–7× query evaluation speedup and documents
   production implementation decisions (block size, score normalization).
   *Direct blueprint for Rose's Block-Max WAND implementation.*

4. **Ding & Suel — "Optimizing Top-k Document Retrieval Strategies for Block-Max Indexes"**
   WSDM 2013.
   [DOI:10.1145/2433396.2433412](https://dl.acm.org/doi/10.1145/2433396.2433412) |
   [PDF](https://research.engineering.nyu.edu/~suel/papers/bmm.pdf)
   Introduces Block-Max MaxScore (BMM): combines per-block max scores with the MaxScore
   skip condition, pruning ~2× more documents than either BMW or MaxScore alone.
   *Defines the block-max storage layout used in Rose's segment format.*

5. **Mallia, Ottaviano, Porciani, Tonellotto, Venturini — "Faster Block-Max WAND with
   Variable-Sized Blocks"**
   SIGIR 2017.
   [DOI:10.1145/3077136.3080780](https://dl.acm.org/doi/10.1145/3077136.3080780)
   Frames block boundary selection as a global optimisation problem; shows variable-size
   blocks outperform uniform blocks by 10–30% in query throughput.  Rose uses fixed-size
   blocks (128 postings) for simplicity, with this paper as the reference for future tuning.

6. **Mallia, Siedlaczek, Mackenzie, Suel — "PISA: Performant Indexes and Search for
   Academia"**
   OSIRRC@SIGIR 2019.
   [PDF](https://ceur-ws.org/Vol-2409/docker08.pdf) |
   [GitHub](https://github.com/pisa-engine/pisa)
   Open-source C++ reference implementation covering OptPFOR, QMX, Elias-Fano, MaxScore,
   WAND, and Block-Max WAND.  Rose's WAND loop structure mirrors PISA's `maxscore_query`.

7. **Mallia, Suel, Tonellotto et al. — "Efficient In-Memory Inverted Indexes: Theory and
   Practice"**
   Tutorial, SIGIR 2025.
   [DOI:10.1145/3726302.3731688](https://dl.acm.org/doi/10.1145/3726302.3731688) |
   [Site](https://pisa-engine.github.io/sigir-2025.html)
   Most comprehensive 2025 overview: compression, DAAT traversal, MaxScore/WAND/BMW
   dynamic pruning, quantised BM25, and the transition to learned sparse retrieval.

#### BM25 Scoring

8. **Kamphuis, de Vries, Boytsov, Lin — "Which BM25 Do You Mean? A Large-Scale
   Reproducibility Study of Scoring Variants"**
   ECIR 2020.
   [DOI:10.1007/978-3-030-45442-5_4](https://link.springer.com/chapter/10.1007/978-3-030-45442-5_4) |
   [PDF](https://cs.uwaterloo.ca/~jimmylin/publications/Kamphuis_etal_ECIR2020_preprint.pdf)
   Documents and disambiguates eight BM25 variants (Lucene, Terrier, Anserini, Indri…).
   Rose implements the **Robertson BM25+** variant (equation below) — the most studied
   formulation with the best theoretical properties.

9. **Lu — "BM25S: Orders of Magnitude Faster Lexical Search via Eager Sparse Scoring"**
   arXiv:2407.03618, July 2024.
   [arXiv](https://arxiv.org/abs/2407.03618) | [GitHub](https://github.com/xhluca/bm25s)
   Pre-computes all BM25(t,d) scores at index time and stores as a sparse matrix; query
   becomes a sparse matrix-vector multiply.  *Basis for Approach C.*  Rose borrows the
   uint8 quantisation scheme from this paper.

#### LSM / Segment Architecture

10. **Survey: "A Survey of LSM-Tree Based Indexes, Data Systems and KV-Stores"**
    arXiv:2402.10460, February 2024.
    [arXiv](https://arxiv.org/abs/2402.10460)
    Covers tiered vs. levelled merge policies, bloom filter sizing, compaction scheduling,
    and write amplification bounds.  Rose's merge policy (tiered: ≥4 segments at the same
    size tier triggers a merge) is the classic LSM tiering strategy described here.

#### Learned Sparse Retrieval (context / future work)

11. **Mallia, Mackenzie, Suel, Tonellotto — "Faster Learned Sparse Retrieval with Guided
    Traversal"**
    SIGIR 2022.
    [DOI:10.1145/3477495.3531774](https://dl.acm.org/doi/10.1145/3477495.3531774) |
    [arXiv:2204.11314](https://arxiv.org/abs/2204.11314)

12. **Mallia, Suel, Tonellotto — "Faster Learned Sparse Retrieval with Block-Max Pruning"**
    SIGIR 2024.
    [DOI:10.1145/3626772.3657906](https://dl.acm.org/doi/10.1145/3626772.3657906) |
    [arXiv:2405.01117](https://arxiv.org/abs/2405.01117)

13. **Bruch, Nardini, Rulli, Venturini — "Seismic: Efficient and Effective Retrieval over
    Learned Sparse Representations"**
    SIGIR 2024 Best Paper.
    [DOI:10.1145/3626772.3657769](https://dl.acm.org/doi/10.1145/3626772.3657769) |
    [arXiv:2404.18812](https://arxiv.org/abs/2404.18812)

    Papers 11–13 describe the trajectory of Block-Max pruning applied to neural sparse
    (SPLADE/DeepImpact) indexes.  Rose's block-max infrastructure is forward-compatible
    with replacing BM25 impact scores with quantised neural scores.

---

## Three Candidate Approaches

### Approach A — Partitioned Elias-Fano + MaxScore (not implemented)

**Papers:** Ottaviano & Venturini SIGIR 2014; PISA; SIGIR 2025 tutorial.

Posting lists encoded with Partitioned Elias-Fano: list split into variable-size blocks,
each independently EF-compressed (pointer array + lower-bits array).  Achieves near-optimal
bits-per-integer.  Top-k via MaxScore: sort lists by whole-list max impact; "non-essential"
lists (whose entire max cannot beat current threshold) are skipped completely without
examining any posting.

**Pros:** Best-in-class compression; O(1) random access per posting (good for AND queries).
**Cons:** EF bit manipulation is complex in pure Go with no SIMD; the bit-level access
advantage disappears without vectorised decode; MaxScore has less pruning power than BMW on
disjunctive OR queries.

---

### Approach B — VByte Delta + Block-Max WAND + Quantised BM25 ← **IMPLEMENTED**

**Papers:** Grand et al. ECIR 2020; Ding & Suel WSDM 2013; Kamphuis et al. ECIR 2020;
arXiv:2402.10460 (LSM); Pibiri & Venturini ACM Surveys 2021.

See §Implementation below.

---

### Approach C — Pre-scored Sparse Matrix / BM25S (not implemented)

**Papers:** Lu arXiv:2407.03618 (BM25S, 2024).

At index time, compute BM25(t, d) for every (term, doc) pair and store in a Compressed
Sparse Row (CSR) matrix, one row per term, values are uint16 scores.  The entire matrix is
zstd-compressed.  Query time = sparse matrix-vector product: look up each query term's row,
accumulate into a docID-keyed score array, take top-k.

**Pros:** Simplest query processor (no cursor management, no heap during traversal);
theoretically up to 500× faster than naïve Python BM25 clients (per the paper).
**Cons:** Index is 4–8× larger than an inverted index (stores float32/uint16 scores per
posting, not just delta docIDs); phrase queries impossible; score accumulator array must fit
in RAM for the full corpus; no streaming ingest — must know the full term vocabulary and
corpus size before writing.

---

## Implementation — Approach B

### 2.1 BM25+ Scoring

Rose implements **Robertson BM25+** (Lüc Büttcher et al. 2010, surveyed in Kamphuis ECIR 2020):

```
IDF(t)       = log( (N - df(t) + 0.5) / (df(t) + 0.5) + 1 )
TF_norm(t,d) = tf(t,d) * (k1 + 1)
               ─────────────────────────────────────────────
               tf(t,d) + k1 * (1 - b + b * (dl / avgdl))
BM25+(t,d)   = IDF(t) * TF_norm(t,d) + delta
```

Constants: `k1 = 1.2`, `b = 0.75`, `delta = 1.0`.

`delta = 1.0` is the defining difference from plain BM25: it lower-bounds every
matching term's contribution, eliminating zero-score matches and slightly improving
retrieval effectiveness on short queries (from the BM25+ paper).

**Quantisation** (borrowed from BM25S, arXiv:2407.03618): at segment flush time,
find `max_score` = max BM25+ across all postings in a given posting list.  Quantise:

```
impact_uint8 = clamp( round(bm25_plus / max_score * 255), 1, 255 )
```

The `clamp(…, 1, …)` guarantees every matching posting has a nonzero impact,
preserving the BM25+ lower-bound property.  `max_score` is stored in the term
dictionary entry so dequantisation is possible at search time.

At query time, scores are accumulated as `uint16` (sum of up to 255 × N uint8 impacts,
where N ≤ query length ≤ ~10) and converted to float64 only for the final heap.

---

### 2.2 Text Analysis Pipeline

```
Raw text bytes
  │
  ▼ Unicode tokenise
    split on [\P{L}\P{N}]+ (anything that is not a Unicode letter or digit)
    discard tokens < 2 bytes or > 64 bytes
  │
  ▼ Lowercase (unicode.ToLower, handles accented chars)
  │
  ▼ Stopword filter
    127-word English list (hardcoded; same set used by Lucene's English analyser)
  │
  ▼ Snowball English stem (kljensen/snowball, already in go.mod)
  │
  ▼ term string → in-memory posting map
```

Term hashing for the in-memory map: `cespare/xxhash/v2` (already in go.mod) for O(1)
map operations; the canonical term string is still stored for segment serialisation.

---

### 2.3 In-memory Indexing Buffer

```go
type memIndex struct {
    mu       sync.Mutex
    postings map[string][]memPosting  // term → sorted []memPosting
    docs     []memDoc                 // docID → {externalID, tf map, docLen}
    sizeEst  int64                    // estimated byte footprint
}

type memPosting struct {
    docIdx uint32 // index into memIndex.docs
    tf     uint16 // raw term frequency
}
```

`sizeEst` is updated on every `Index()` call.  When `sizeEst >= flushThreshold` (default
64 MB), the engine flushes the current buffer to a new `.seg` file atomically, then resets.

Concurrent `Index()` calls: acquire `mu`, append to buffer, release.  The flush itself
holds `mu` only for the swap (reset memIndex pointer), not for the file write.

---

### 2.4 Segment Binary Format

```
Offset  Size   Field
──────────────────────────────────────────────────────
 0       4     Magic = "ROSE" (0x524F5345)
 4       1     Version = 1
 5       4     DocCount (uint32, little-endian)
 9       4     AvgDocLen (float32, little-endian)
13       4     DictSize — number of terms (uint32)
17       8     PostingBase — byte offset of first posting block (uint64)
25      ...    Term Dictionary (sorted lexicographically)
               Each entry:
                 [1] TermLen (uint8)
                 [N] Term bytes (UTF-8)
                 [4] DocFreq (uint32)
                 [1] MaxImpact (uint8) — whole-list max
                 [4] MaxImpactScore (float32) — for dequantisation
                 [8] PostOffset (uint64) — byte offset from PostingBase
                 [4] PostLen (uint32) — compressed size in bytes
...     ...    Posting Lists (from PostingBase)
               Each list = sequence of blocks:
                 [1] BlockMaxImpact (uint8)
                 [1] NumPostings (uint8, 1..128)
                 [N] VByte-encoded docID deltas
                 [N] uint8 impact scores (one per posting, parallel array)
```

**VByte encoding** (standard, from Pibiri & Venturini ACM Surveys 2021, §3.1):

```
encode(v):
  while v >= 128:
    emit byte(v & 0x7F | 0x80)   // continuation bit set
    v >>= 7
  emit byte(v & 0x7F)            // final byte, MSB clear

decode: read bytes until MSB=0; accumulate 7-bit groups
```

Byte-aligned, no bit manipulation, branch-predictable, 1–5 bytes per integer.
A delta of 0–127 costs 1 byte; 128–16383 costs 2 bytes; most docID gaps in a dense
list cost 1–2 bytes.

**Block size = 128 postings**: matches Lucene's default (Grand et al. ECIR 2020) and
aligns with PISA's benchmarked optimum.  The `BlockMaxImpact` header (1 byte) enables
WAND to skip the entire block without decoding any docID.

The segment file is **not** separately zstd-compressed; docID deltas are already compact
via VByte.  For very sparse lists (low df terms) the overhead is minimal.

---

### 2.5 Document Store

`{dir}/rose.docs` is an append-only sequential file:

```
Per document:
  [4] ExternalIDLen (uint32)
  [N] ExternalID bytes (UTF-8) — the caller-supplied DocID string
  [4] TextLen (uint32)
  [M] Text prefix (up to 512 bytes, for snippet generation)
```

A companion in-memory lookup is built at `Open()` time by scanning the file once:
```go
type docEntry struct {
    externalID string
    textOffset int64  // byte offset in rose.docs for snippet read
    textLen    uint32
}
var docTable []docEntry // indexed by internal uint32 docIdx
```

Snippet generation: find the first window of ±30 words around any query term occurrence.

---

### 2.6 Block-Max WAND Algorithm

Based on Grand et al. ECIR 2020 and Ding & Suel WSDM 2013.

```go
// ListCursor wraps one term's posting list across one segment.
type listCursor struct {
    docIDs  []uint32   // decoded docID block (up to 128)
    impacts []uint8    // parallel impact scores
    blockMaxImpact uint8
    wholeListMax   uint8
    pos     int        // position within current block
    // ... segment/term references for advancing to next block
}

func wandTopK(cursors []*listCursor, k int) []hit {
    heap := newMinHeap(k)      // min-heap of (score, docID)
    threshold := uint16(0)

    for {
        // 1. Sort cursors by current docID
        sortByDocID(cursors)

        // 2. Find pivot: scan left to right until cumulative
        //    blockMaxImpact sum >= threshold
        pivot, pivotDoc := findPivot(cursors, threshold)
        if pivot < 0 { break }

        // 3. If all cursors[0..pivot] point to pivotDoc → full evaluation
        if cursors[0].curDocID() == pivotDoc {
            score := evaluateDoc(cursors[:pivot+1], pivotDoc)
            heap.pushOrReplace(score, pivotDoc)
            threshold = heap.minScore()
            advanceAll(cursors[:pivot+1], pivotDoc+1)
        } else {
            // 4. Advance the leftmost cursor to pivotDoc
            cursors[0].advanceTo(pivotDoc)
        }
    }
    return heap.sorted()
}
```

`findPivot` uses `blockMaxImpact` (not whole-list max) for tighter bounds, achieving
the block-max property: when a cursor's current block cannot contribute enough to beat
`threshold`, the entire 128-posting block is skipped.

**Cross-segment search:** One `listCursor` is created per (segment × queryTerm) pair.
All cursors are fed into the same WAND loop.  Results are merged and re-sorted by score.

---

### 2.7 LSM-style Segment Merging

Based on arXiv:2402.10460 (LSM survey, §3.2 "Tiered merge policy").

**Tier definition:** segment is in tier `t` if `flushThreshold * 4^t ≤ segSize < flushThreshold * 4^(t+1)`.

**Trigger:** background goroutine wakes every 5 s.  If any tier has ≥ 4 segments → merge.

**Merge procedure:**
1. Open term dictionary iterators for all input segments (already sorted).
2. N-way merge: advance the iterator with the smallest current term.
3. For each unique term: concatenate posting lists from all segments, re-assign global
   docIDs (offset by each segment's doc base), re-block into 128-posting blocks, recompute
   `BlockMaxImpact`.
4. Corpus stats (N, avgdl) are recomputed from the merged doc table; BM25+ IDF scores
   are recomputed and re-quantised so `MaxImpact` reflects the merged corpus.
5. Write merged segment to `tmp_{uuid}.seg` → fsync → rename to `rose_{seq:05d}.seg`.
6. Delete input segment files; update `rose.meta`.

**Atomicity:** `rose.meta` is rewritten atomically (write to `.tmp` + rename) after
each flush or merge, so a crash leaves a consistent set of segments.

---

### 2.8 Metadata File

`{dir}/rose.meta` — JSON for human inspectability:

```json
{
  "version": 1,
  "doc_count": 173720,
  "avg_doc_len": 412.7,
  "segments": ["rose_00000.seg", "rose_00001.seg"],
  "next_seg_seq": 2
}
```

---

### 2.9 Concurrency Model

```
Index() calls ──► memIndex (mutex-protected write)
                        │
                        │ sizeEst >= 64 MB
                        ▼
                   flushWorker goroutine ──► rose_NNNNN.seg
                        │
                        │ ticker 5s
                        ▼
                   mergeWorker goroutine ──► merged rose_MMMMM.seg

Search() ──► reads all current .seg files (shared, immutable once written)
             opens cursors, runs WAND, returns results
```

`Search()` and `Index()` can run concurrently.  A search sees all segments whose files
exist at the moment `Open()` is called on them; newly flushed segments become visible
only after `rose.meta` is rewritten.

---

## File Layout

```
pkg/index/driver/flower/rose/
├── index.go        Engine struct: Open/Close/Stats/Index/Search; init() registers "rose"
├── analyzer.go     unicodeTokenize → lowercase → stopwordFilter → snowballStem
├── segment.go      Segment: flush memIndex → .seg; readSegment; term lookup
├── postings.go     vbyteEncode / vbyteDecode; packBlock / unpackBlock
├── bm25.go         bm25Plus(tf,df,dl,avgdl,N) float64; quantise/dequantise
├── wand.go         listCursor; wandTopK; findPivot; evaluateDoc
├── merge.go        tierOf; mergeSegments; background merge loop
├── docstore.go     appendDoc; loadDocTable; snippetFor(docIdx, queryTerms)
└── rose_test.go    roundtrip, stats, empty search, multi-segment, merge
```

---

## Build

No build tags required.  No CGO.  No new dependencies (uses only `kljensen/snowball`,
`cespare/xxhash/v2`, and `klauspost/compress` already present in `go.mod`).

```bash
# Run Rose tests
go test ./pkg/index/driver/flower/rose/...

# Benchmark against all embedded drivers
go test -bench=. -benchtime=30s ./pkg/index/driver/flower/rose/...

# Index CC-MAIN-2026-08 corpus
search cc fts index --engine rose
```

---

## Benchmark Plan

Dataset: CC-MAIN-2026-08, 173,720 docs, `--source bin`, Apple M-series Mac (ARM64).

### Index Benchmark

| Engine | Time (s) | Docs/s | Peak RSS (MB) | Disk (MB) |
|--------|----------|--------|--------------|-----------|
| devnull | 0.2 | 787,840 | 0 | 0 |
| sqlite | 0.8 | 212,491 | 314 | 1,126 |
| tantivy (CGO) | 34.8 | 4,998 | 327 | 764 |
| bleve | 75.9 | 2,288 | 3,690 | 3,584 |
| duckdb | 100.8 | 1,724 | 323 | 1,536 |
| **rose** | — | — | — | — |

Target: >10,000 docs/s (4× bleve); disk ≤ 512 MB (2× tantivy).

### Search Benchmark

10 queries, warm run, limit=10. Apple M-series Mac (ARM64), 173,720-doc index.

| Engine | Avg ms | P95 ms | Notes |
|--------|--------|--------|-------|
| tantivy (CGO) | 2 | 2 | BM25, sub-ms after warm-up |
| bleve | 31 | 52 | BM25, returns total hit count |
| sqlite | 85 | 287 | FTS5 BM25 |
| duckdb | 218 | 244 | tfidf; full-scan |
| **rose** | — | — | BM25+, Block-Max WAND |

Target: <10ms avg; <30ms P95 on the 10-query standard set.

Queries (same set as spec/0644):
1. `machine learning`
2. `climate change`
3. `artificial intelligence`
4. `United States`
5. `open source software`
6. `COVID-19 pandemic`
7. `data privacy`
8. `renewable energy`
9. `blockchain technology`
10. `neural network`

---

## Implementation Order

1. `postings.go` — VByte encode/decode; block pack/unpack (unit-tested in isolation)
2. `bm25.go` — BM25+ formula; uint8 quantisation; test against known values
3. `analyzer.go` — tokeniser + stopwords + Snowball stem; test with English sentences
4. `docstore.go` — append/read doc store; snippet extraction
5. `segment.go` — flush memIndex → .seg; readSegment; binary format
6. `wand.go` — listCursor; wandTopK across one segment; unit test with synthetic data
7. `merge.go` — N-way merge; tiered policy; background goroutine
8. `index.go` — Engine: Open/Close/Stats/Index/Search; init() registration
9. `rose_test.go` — roundtrip; multi-segment; merge; empty; stats
10. Benchmark on 173,720-doc corpus; fill tables above
