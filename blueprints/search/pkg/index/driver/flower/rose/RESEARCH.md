# Rose — Deep Technical Research Notes

This file documents every algorithm, data structure, and paper used in Rose's
implementation.  It is the authoritative reference for anyone reading or modifying
the source code.

---

## 1. Text Analysis

### 1.1 Unicode Tokenisation

**Approach:** Split on `[\P{L}\P{N}]+` — any run of characters that are neither Unicode
letters (`\p{L}`) nor Unicode decimal digits (`\p{N}`).  This correctly handles accented
letters (é, ü, ñ), CJK ideographs, and emoji without any language-specific rules.

**Implementation in Go:**

```go
import "unicode"

func tokenise(text []byte) []string {
    var tokens []string
    inToken := false
    start := 0
    for i, r := range string(text) {
        isWordChar := unicode.IsLetter(r) || unicode.IsDigit(r)
        if isWordChar && !inToken {
            start, inToken = i, true
        } else if !isWordChar && inToken {
            tok := string(text[start:i])
            if len(tok) >= 2 && len(tok) <= 64 {
                tokens = append(tokens, tok)
            }
            inToken = false
        }
    }
    if inToken {
        tok := string(text[start:])
        if len(tok) >= 2 && len(tok) <= 64 {
            tokens = append(tokens, tok)
        }
    }
    return tokens
}
```

**Token length bounds:** min=2 (skip single-char tokens like "a", "I"), max=64 (skip
pathological URLs/hashes that pollute the vocabulary and compress poorly).

### 1.2 Stopword Filter

127-word English stopword list, identical to Lucene's `EnglishAnalyzer` default set.
Stored as a `map[string]struct{}` for O(1) lookup.  Applied after lowercasing and before
stemming (so the stemmer never sees high-frequency stopwords).

Common stopwords included: a, an, the, and, or, but, in, on, at, to, for, of, with,
by, from, as, is, was, are, were, be, been, being, have, has, had, do, does, did, will,
would, could, should, may, might, shall, can, need, dare, ought, used, am, it, its, he,
she, they, we, you, i, me, him, her, them, us, my, your, his, their, our, this, that,
these, those, what, which, who, whom, whose, when, where, why, how, all, both, each,
few, more, most, other, some, such, no, nor, not, only, own, same, so, than, too, very,
just, because, if, then, else, while, about, against, between, into, through, during,
before, after, above, below, up, down, out, off, over, under, again, further, once, etc.

### 1.3 Snowball English Stemmer

**Library:** `github.com/kljensen/snowball` (Porter2 / English algorithm).
**Algorithm:** Porter2 (aka Snowball English), a series of suffix-stripping rules.

Key stemming examples relevant to the CC corpus:
- "learning" → "learn", "learned" → "learn"
- "running" → "run", "algorithms" → "algorithm"
- "neural" → "neural" (no change), "networks" → "network"
- "climate" → "climat" (note: stemmer, not lemmatiser)

**Usage:**
```go
import "github.com/kljensen/snowball/english"

stemmed := english.Stem(word, false)  // false = don't lowercase (already done)
```

The stemmer is applied per-token after stopword removal, so it only processes
vocabulary words.

---

## 2. Inverted Index Data Structures

### 2.1 In-Memory Posting Map

During indexing, Rose accumulates postings in a `map[string][]memPosting`:

```go
type memPosting struct {
    docIdx uint32
    tf     uint16   // raw term frequency in this document
}

type memDoc struct {
    externalID string
    docLen     uint32   // total number of tokens in document
    text       []byte   // first 512 bytes for snippet store
}
```

Memory estimation: each `memPosting` = 6 bytes; each map entry overhead ≈ 100 bytes
(Go map bucket overhead).  A 173K-doc corpus with avg 80 unique terms/doc ≈
~100 MB of posting data, safely below the 64 MB flush threshold which triggers multiple
sequential segments.

**Flush threshold** = 64 MB: chosen to stay below typical L3 cache sizes while
producing segments large enough to amortise file I/O overhead.  Configurable via
`Engine.FlushThresholdMB`.

### 2.2 VByte Delta Encoding

**Reference:** Pibiri & Venturini, ACM Computing Surveys 53(6) 2021, §3.1.
[arXiv:1908.10598](https://arxiv.org/abs/1908.10598)

VByte (Variable Byte) is the simplest practical integer codec.  Integers are encoded
7 bits at a time, LSB first, with the MSB of each byte used as a continuation flag.

```
Bit layout per byte:  [cont_bit | b6 | b5 | b4 | b3 | b2 | b1 | b0]
  cont_bit = 1 → more bytes follow
  cont_bit = 0 → last byte of this integer
```

**Encoding ranges:**
| Integer range      | Bytes | Bit usage |
|--------------------|-------|-----------|
| 0 – 127            | 1     | 7/8 bits  |
| 128 – 16,383       | 2     | 14/16 bits|
| 16,384 – 2,097,151 | 3     | 21/24 bits|
| 2,097,152 – 268M   | 4     | 28/32 bits|

**Delta encoding:** Instead of storing absolute docIDs, store `delta[i] = docID[i] - docID[i-1]`
(first delta = docID[0]).  For a sorted posting list, deltas are always non-negative.
In a typical posting list, docIDs are spread across a 173K-doc corpus, giving average
deltas of ~173000/df.  For a term with df=1000, avg delta ≈ 173, encoding in 2 bytes each.
For df=10000, avg delta ≈ 17, encoding in 1 byte each.

**Go implementation:**

```go
func vbyteEncode(buf []byte, v uint32) []byte {
    for v >= 0x80 {
        buf = append(buf, byte(v&0x7F)|0x80)
        v >>= 7
    }
    return append(buf, byte(v))
}

func vbyteDecode(buf []byte, pos int) (uint32, int) {
    var v uint32
    var shift uint
    for {
        b := buf[pos]; pos++
        v |= uint32(b&0x7F) << shift
        if b < 0x80 { break }
        shift += 7
    }
    return v, pos
}
```

**Why not PForDelta or Elias-Fano?**
- PForDelta (FastPFOR variant) achieves ~0.5–1.0 bits/int advantage over VByte but
  requires batches of 128 integers and bitwise operations that are slower in pure Go
  without SIMD.  Reference: Pibiri & Venturini §3.3.
- Elias-Fano achieves optimal compression for sorted sequences but requires bit-level
  manipulation (pointer array + lower-bits array) that is complex to implement correctly
  without C/assembly.  Reference: Ottaviano & Venturini SIGIR 2014.
- VByte's byte-alignment means the Go decoder is branch-predictable and generates
  efficient machine code.  In benchmarks on the Go runtime, VByte decoding throughput
  is ~300–500 M integers/s vs ~100–200 M integers/s for bit-level codecs in pure Go.

### 2.3 Block Structure (128 postings per block)

**Reference:** Grand et al. ECIR 2020, §3.1 "Block-Max Indexes".

Each posting list is partitioned into fixed-size blocks of **128 postings**.  The block
header stores the maximum quantised impact score in that block (`BlockMaxImpact`).

```
Block layout:
  [1 byte]  BlockMaxImpact  — max uint8 impact in this block
  [1 byte]  NumPostings     — actual count (1..128; last block may be < 128)
  [N bytes] DocID deltas    — VByte encoded; delta from start of THIS block
                              (first delta = docID[0] - blockBase, where blockBase
                               is the last docID of the previous block, or 0)
  [N bytes] Impact scores   — uint8 array, one per posting (parallel to docIDs)
```

Note: docID deltas within a block are relative to the **last docID of the previous block**
(not relative to the previous delta within the block).  This allows random access to any
block: seek to block k, reset delta accumulator to the known "block base" docID stored
in the block index (see segment format).

**Block index** (stored in the term dict entry): an array of `(blockBase uint32,
fileOffset uint64)` pairs, one per block.  This allows O(log B) binary search to find
the block containing a target docID, enabling `advanceTo(targetDocID)` without scanning
all prior blocks.

**Why 128?** Matches Lucene's default (from the BMW paper's §4 analysis) and PISA's
benchmarked optimum.  Smaller blocks (e.g. 32) give tighter bounds but more overhead;
larger blocks (e.g. 512) reduce overhead but make WAND bounds looser.

---

## 3. BM25+ Scoring and Quantisation

### 3.1 BM25+ Formula

**References:**
- Robertson & Zaragoza, "The Probabilistic Relevance Framework: BM25 and Beyond",
  Foundations and Trends in Information Retrieval 3(4), 2009.
- Liang & Croft, "A Comparison of Collection Selection Algorithms for Distributed
  Information Retrieval", ACM TOIS 2012 (introduces BM25+ delta term).
- Kamphuis et al., ECIR 2020, §2 — disambiguates 8 BM25 variants; Rose uses "Robertson".
  [PDF](https://cs.uwaterloo.ca/~jimmylin/publications/Kamphuis_etal_ECIR2020_preprint.pdf)

```
IDF(t) = log( (N - df(t) + 0.5) / (df(t) + 0.5) + 1 )

           (k1 + 1) · tf(t,d)
TF(t,d) = ─────────────────────────────────────────────
           tf(t,d) + k1 · (1 - b + b · dl(d) / avgdl)

BM25+(t,d) = IDF(t) · TF(t,d) + δ
```

**Constants:**
- `k1 = 1.2` — TF saturation.  Higher k1 → TF counts more; lower → faster saturation.
  1.2 is the Robertson standard; Lucene uses 1.2.
- `b = 0.75` — document length normalisation.  1.0 = full normalisation; 0.0 = none.
  0.75 is the standard recommended value.
- `δ = 1.0` — BM25+ additive floor.  Prevents zero scores for matching terms, improving
  recall on short queries where some query terms are rare.  Value 1.0 from Liang & Croft.

**Difference from BM25:** The `+ δ` term.  A document containing term t always gets at
least δ · IDF(t) contribution from that term, regardless of document length.

**IDF floor:** When df(t) ≈ N/2 (very common term), the numerator can approach 0.5 and
the log approaches log(1.5) ≈ 0.4.  IDF is never negative in the Robertson formulation
(the `+1` inside the log ensures this), unlike some older formulations.

### 3.2 Quantisation to uint8

**Reference:** Lu, "BM25S: Orders of Magnitude Faster Lexical Search", arXiv:2407.03618,
§3 "Scoring". [arXiv:2407.03618](https://arxiv.org/abs/2407.03618)

At segment flush time, per-term:

```go
func quantise(scores []float64) []uint8 {
    maxScore := 0.0
    for _, s := range scores { if s > maxScore { maxScore = s } }
    if maxScore == 0 { maxScore = 1 }

    out := make([]uint8, len(scores))
    for i, s := range scores {
        q := int(math.Round(s / maxScore * 255))
        if q < 1  { q = 1   }   // BM25+ floor: never zero
        if q > 255 { q = 255 }
        out[i] = uint8(q)
    }
    return out
}
```

`maxScore` is stored in the term dictionary entry (`MaxImpactScore float32`) to allow
exact dequantisation at search time if needed.

**Why uint8?**  Accumulating up to 10 query terms × 255 max impact = 2550 < 65535.
So the per-document accumulator can be `uint16` — pure integer arithmetic in the hot
path, no floating point until the final top-k sort.

**Quantisation loss:** The rounding error per term is at most 0.5/255 ≈ 0.2% of the
term's max impact.  This has negligible effect on ranking quality (BM25+ already
approximate due to IDF smoothing).

### 3.3 Corpus Statistics

During indexing, Rose tracks:
- `N` = total document count (for IDF)
- `sumDocLen` = sum of all document lengths (for avgdl = sumDocLen/N)

These are maintained in the `rose.meta` file and updated atomically after each flush.
When segments are merged, the merged segment recomputes BM25+ IDF scores from the
merged N and avgdl, and re-quantises all impact scores accordingly.

---

## 4. Block-Max WAND Algorithm

### 4.1 Background: DAAT vs. Dynamic Pruning

**Document-At-A-Time (DAAT)** traversal naïvely evaluates every document containing
at least one query term.  For a 10-word query on a 173K-doc corpus, this can mean
scoring hundreds of thousands of documents to find the top 10.

**Dynamic pruning** exploits the fact that we only need the top-k: if an upper bound
on a document's score is less than the k-th score seen so far, we can safely skip it.

### 4.2 WAND (Weak AND)

**Reference:** Broder, Carmel, Herscovici, Soffer, Zien, "Efficient Query Evaluation
using a Two-Level Retrieval Process", CIKM 2003.

WAND maintains a **threshold** θ = the k-th highest score seen so far (min of a size-k
max-heap).  For each candidate document d, it computes an upper bound using per-term
global maxima:

```
UB(d) = Σ  max_impact(t)   for all t where docID(t) ≤ d
```

If `UB(d) < θ`, document d is skipped.  The pivot selection finds the minimum d for
which UB(d) ≥ θ, enabling large jumps in the posting lists.

### 4.3 Block-Max WAND (BMW)

**References:**
- Ding & Suel, WSDM 2013. [DOI:10.1145/2433396.2433412](https://dl.acm.org/doi/10.1145/2433396.2433412)
- Grand et al., ECIR 2020 (Lucene implementation). [PDF](https://cs.uwaterloo.ca/~jimmylin/publications/Grand_etal_ECIR2020_preprint.pdf)

BMW improves on WAND by using **per-block** max impact (`BlockMaxImpact`) instead of
the global per-list maximum.  This gives a tighter upper bound:

```
UB_block(d) = Σ  blockMaxImpact(t, block_containing_d)
```

`blockMaxImpact` ≤ `max_impact` always, so BMW prunes at least as aggressively as WAND,
and in practice prunes 2–5× more documents on typical BM25 index distributions.

**The pivot algorithm (Grand et al. ECIR 2020 §3.2):**

```
1. Sort cursors by current docID: cur[0].docID ≤ cur[1].docID ≤ … ≤ cur[n-1].docID
2. Scan left to right; accumulate blockMaxImpact[i].
   Find pivot p: smallest index where prefix_sum(blockMaxImpact[0..p]) ≥ threshold.
3. pivotDoc = cur[p].docID
4. If cur[0].docID == pivotDoc:
   → All cursors[0..p] are at pivotDoc. Evaluate: score = Σ actual_impact[i]
   → Push (score, pivotDoc) to heap; update threshold = heap.min
   → Advance all cursors[0..p] past pivotDoc
5. Else (cur[0].docID < pivotDoc):
   → Some cursors are behind pivotDoc. Advance cur[0] to pivotDoc via advanceTo().
   advanceTo() uses the block index to skip entire blocks when BlockMaxImpact cannot
   contribute to a score ≥ threshold. This is the inner-loop block skip.
6. Repeat.
```

**advanceTo(targetDocID, threshold):**
```
Binary search in block index for the block containing targetDocID.
While current block's BlockMaxImpact + sumOtherCursorsMax < threshold:
    skip entire block (advance to next block).
Within the first non-skippable block: linear scan docIDs to targetDocID.
```

**Correctness:** BMW is an **exact** top-k algorithm — it returns precisely the same
top-k documents and scores as a full DAAT evaluation.  No approximation.

**Expected speedup:** Grand et al. ECIR 2020 reports 3–7× query evaluation speedup
over full DAAT on standard TREC test collections with BM25.  On selective queries
(high-IDF query terms), the speedup can exceed 10×.

### 4.4 Cross-Segment Query Processing

Rose creates one `listCursor` per (queryTerm × segment) pair.  All cursors are combined
into a single WAND loop.  The segment-local uint32 docIDs are mapped to global uint32
docIDs via `segmentBase + localDocID`.  Results from all cursors are accumulated into
one min-heap of size k, producing a globally correct top-k ranking.

---

## 5. Segment Architecture

### 5.1 Segment Format Details

```
Byte 0–3:   Magic "ROSE" = 0x524F5345 (big-endian for readability)
Byte 4:     Version = 0x01
Byte 5–8:   DocCount (uint32, little-endian) — docs in this segment
Byte 9–12:  AvgDocLen (float32, little-endian) — avg tokens per doc in segment
Byte 13–16: DictSize (uint32) — number of unique terms
Byte 17–24: PostingBase (uint64) — absolute byte offset of first posting block

Term Dictionary (DictSize entries, sorted lexicographically by term):
  [1] TermLen (uint8)
  [N] Term (UTF-8 bytes, max 64)
  [4] DocFreq (uint32)
  [1] MaxImpact (uint8) — whole-list max impact (for WAND global bound)
  [4] MaxImpactScore (float32) — max BM25+ score before quantisation
  [8] PostOffset (uint64) — offset from PostingBase to this list's first block
  [4] PostLen (uint32) — byte length of posting list data
  [4] NumBlocks (uint32) — number of 128-posting blocks
  [NumBlocks × (uint32 + uint64)] BlockIndex:
       [4] BlockBase (uint32) — docID of last posting in PREVIOUS block (or 0)
       [8] BlockFileOff (uint64) — absolute byte offset of this block

Posting Blocks (from PostingBase):
  [1] BlockMaxImpact (uint8)
  [1] NumPostings (uint8, 1..128)
  [N] VByte-encoded deltas (delta from BlockBase, delta-of-delta within block)
  [N] uint8 impact scores
```

**Term dictionary binary search:** At search time, `termLookup(term)` does a binary
search over the sorted term dictionary (O(log V) comparisons, where V = vocabulary size).
For a 173K-doc corpus, V ≈ 500K terms, so ~19 comparisons per lookup.

An alternative would be a hash map (O(1) expected), but binary search requires no
additional data structure, reads sequentially (cache-friendly), and 19 comparisons at
nanosecond speed is negligible compared to the subsequent WAND traversal.

### 5.2 Segment Flush Procedure

```
1. Lock memIndex; snapshot current map + docs; reset memIndex.
2. Release lock (Index() calls can proceed to new memIndex).
3. Sort terms lexicographically.
4. For each term, sort memPostings by docIdx (already insertion-ordered, but verify).
5. Compute BM25+ scores for all (term, doc) pairs using current N and avgdl.
6. Quantise scores to uint8.
7. Pack postings into 128-posting blocks; compute BlockMaxImpact for each block.
8. Write segment to tmp file; build block index in memory.
9. fsync tmp file.
10. Rename to rose_{seq:05d}.seg.
11. Append docs to rose.docs.
12. Rewrite rose.meta atomically (write tmp + rename).
```

### 5.3 Concurrency Safety

- `Index()` acquires a mutex, appends to the active `memIndex`, releases the mutex.
- The flush worker runs in a separate goroutine.  It swaps the memIndex pointer under
  the mutex (O(1)) then performs all I/O without holding the mutex.
- `Search()` reads `rose.meta` at start; opens segment files that exist at that moment.
  Segment files are immutable once written.  New segments become visible after meta
  rewrite.
- Merge: reads from multiple existing segments, writes a new segment, then atomically
  updates meta and deletes old segments.  Search() running concurrently sees either the
  old set or the new set (never a partial view), because segment files are not deleted
  until the meta is updated.

---

## 6. LSM-style Merge

### 6.1 Tiered Merge Policy

**Reference:** "A Survey of LSM-Tree Based Indexes, Data Systems and KV-Stores",
arXiv:2402.10460, §3.2 "Tiered Compaction".
[arXiv:2402.10460](https://arxiv.org/abs/2402.10460)

In the classic LSM tiered policy:
- Level 0 contains the freshest, smallest segments.
- A level is "full" when it has ≥ T segments (Rose: T=4).
- When full, all segments at that level are merged into one segment at level L+1.
- Tier boundaries: flushThreshold × 4^L.

**Write amplification** of tiered policy = O(L × T) where L = number of levels.
For T=4 and a corpus that fits in 3 levels (4^3 = 64 flush segments ≈ 64 × 64 MB = 4 GB),
write amplification ≈ 12×.  This is acceptable for a 173K-doc corpus.

### 6.2 N-Way Merge Algorithm

All input segment term dictionaries are already sorted.  N-way merge:

```go
type termIter struct {
    segment  *Segment
    termIdx  int       // current position in segment.terms
}

// Priority queue ordered by term string (min-heap)
pq := newTermPQ(iters)

for pq.Len() > 0 {
    // Pop the iter with the smallest current term
    minTerm, iters := pq.popEqual()  // pops all iters at the same term

    // Merge posting lists from all iters at this term
    mergedPostings := mergePostingLists(minTerm, iters, globalDocBase)

    // Recompute BM25+ scores with merged N, avgdl
    recompute(mergedPostings, mergedN, mergedAvgdl)

    // Quantise, re-block, write to output segment
    writeTermBlock(out, minTerm, mergedPostings)
}
```

DocID mapping: each input segment has a `segBase uint32` — the first global docID
assigned to docs in that segment.  Local docID `d` in segment `s` maps to global
`segBase[s] + d`.

### 6.3 Merge Triggers and Scheduling

```go
// Background goroutine
func (e *Engine) mergeLoop(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            if segs := e.segmentsInTier(0); len(segs) >= 4 {
                e.mergeSegments(segs)
            }
        case <-ctx.Done():
            return
        }
    }
}
```

Merge is triggered every 5 seconds if any tier has ≥ 4 segments.  During a merge,
`Index()` continues writing to the active memIndex unimpeded.

---

## 7. Document Store and Snippets

### 7.1 Document Store Format

`rose.docs` is a sequential append-only file:

```
Per document (in insertion order, matching internal docIdx):
  [4] ExternalIDLen (uint32, little-endian)
  [N] ExternalID bytes (UTF-8) — caller-supplied DocID string
  [4] TextLen (uint32) — actual stored text length (min(512, full text len))
  [M] Text bytes (UTF-8, up to 512 bytes)
```

At `Open()`, scan `rose.docs` once to build:
```go
type docEntry struct {
    externalID  string
    textOffset  int64  // byte offset in rose.docs where text starts
    textLen     uint32
}
```

### 7.2 Snippet Generation

```go
func snippet(text []byte, queryTerms []string, maxLen int) string {
    // Find first occurrence of any stemmed query term in text
    // Return a ±30-word window around that occurrence
    // Truncate to maxLen characters
}
```

Snippet generation uses the same analysis pipeline (tokenise + stem) applied to the
stored text fragment, matching against stemmed query terms.

---

## 8. Performance Engineering Notes

### 8.1 Hot Path Analysis

The query hot path is `wandTopK`:

```
For each pivot candidate:
  1. Sort N cursors by docID  → O(N log N), N ≤ ~10 (number of query terms × segments)
  2. Scan prefix for pivot    → O(N)
  3. advanceTo() inner loop:
       a. Binary search block index → O(log B), B = num blocks
       b. VByte decode of block → O(128) byte reads
     This is the innermost loop; target: 300M+ int decodes/s
  4. Score evaluation (when all cursors align):
       uint8 accumulation → O(N), N ≤ 10; done in registers
  5. Heap push → O(log k), k ≤ 10
```

VByte decoding is the dominant cost.  Go's compiler generates efficient code for the
VByte loop (no bit manipulation, byte-aligned reads, predictable branch on MSB).

### 8.2 Memory Layout

Decoded block buffers (`[]uint32` for docIDs, `[]uint8` for impacts) are pre-allocated
per cursor and reused across block advances.  This avoids GC pressure in the hot path.

```go
type listCursor struct {
    // ...
    docBuf [128]uint32   // decoded docIDs for current block (on stack if possible)
    impBuf [128]uint8    // decoded impacts for current block
    blockN int           // number of valid entries in docBuf/impBuf
    posInBlock int       // current position within docBuf
}
```

### 8.3 Parallel Indexing

`Engine.Index(ctx, docs []Document)` acquires a mutex only for the append-to-memIndex
operation.  Multiple goroutines calling `Index()` concurrently serialize on this mutex,
but the critical section is just a few map writes — sub-microsecond.  File I/O (flush,
merge) happens entirely outside the mutex.

### 8.4 Expected Throughput Analysis

Indexing bottleneck is analysis (tokenise + stem):
- Snowball stem: ~5 µs/token (from kljensen benchmarks)
- A 412-byte average doc with ~80 unique tokens → ~400 µs of analysis per doc
- At 1 goroutine: ~2,500 docs/s
- At 8 goroutines (M2 Pro cores): ~20,000 docs/s
- Pipeline (caller uses 8 parallel Index() calls via RunPipeline): **target 10–20K docs/s**

Search bottleneck is VByte decode + WAND overhead:
- VByte decode: ~300M int/s in Go
- For a 1000-df term on 173K docs: 1000 postings / 128 = ~8 blocks
- BMW skips ~90% of blocks on selective queries → decodes ~0.8 blocks per term
- For a 2-term query: ~1.6 block decodes = ~205 VByte decodes ≈ 0.7 µs of decode
- Plus sort/pivot overhead: ~2–5 µs per query
- **Target: 2–10ms per query** (including disk reads for cold segments)

---

## 9. Future Extensions

1. **Variable-size blocks** (Mallia et al. SIGIR 2017): Optimise block boundaries
   globally to maximise skip probability.  Estimated +10–30% query throughput.

2. **Elias-Fano compression** for dense posting lists (df > 0.1% of corpus):
   Switch from VByte to PEF for lists where EF gives > 20% size reduction.
   Reference: Ottaviano & Venturini SIGIR 2014.

3. **Neural sparse scoring** (SPLADE/DeepImpact-style):
   Replace BM25+ uint8 impacts with neural sparse model outputs.
   Rose's block-max infrastructure is directly compatible (per BMP, SIGIR 2024).
   Reference: Mallia et al. arXiv:2405.01117.

4. **Document reordering** (Yafay & Altingovde SIGIR 2023): Sort docIDs within
   segments by predicted relevance so WAND encounters high-scoring documents early,
   raising the threshold faster and skipping more blocks.

5. **Concurrent segment search** with `golang.org/x/sync/errgroup`: Search each
   segment in a separate goroutine and merge results.  Currently all segments are
   searched serially by the WAND loop.

---

## 10. References (Full List)

| # | Paper | Venue | Link |
|---|-------|-------|------|
| 1 | Pibiri & Venturini, "Techniques for Inverted Index Compression" | ACM CSUR 2021 | [DOI](https://dl.acm.org/doi/10.1145/3415148) · [arXiv:1908.10598](https://arxiv.org/abs/1908.10598) |
| 2 | Ottaviano & Venturini, "Partitioned Elias-Fano Indexes" | SIGIR 2014 | [DOI](https://dl.acm.org/doi/10.1145/2600428.2609615) · [PDF](http://groups.di.unipi.it/~ottavian/files/elias_fano_sigir14.pdf) |
| 3 | Grand, Muir, Ferenczi, Lin, "From MaxScore to Block-Max Wand" | ECIR 2020 | [DOI](https://link.springer.com/chapter/10.1007/978-3-030-45442-5_3) · [PDF](https://cs.uwaterloo.ca/~jimmylin/publications/Grand_etal_ECIR2020_preprint.pdf) |
| 4 | Ding & Suel, "Optimizing Top-k Document Retrieval for Block-Max" | WSDM 2013 | [DOI](https://dl.acm.org/doi/10.1145/2433396.2433412) · [PDF](https://research.engineering.nyu.edu/~suel/papers/bmm.pdf) |
| 5 | Mallia et al., "Faster Block-Max WAND with Variable-Sized Blocks" | SIGIR 2017 | [DOI](https://dl.acm.org/doi/10.1145/3077136.3080780) |
| 6 | Mallia, Siedlaczek, Mackenzie, Suel, "PISA" | OSIRRC@SIGIR 2019 | [PDF](https://ceur-ws.org/Vol-2409/docker08.pdf) · [GitHub](https://github.com/pisa-engine/pisa) |
| 7 | Mallia, Suel, Tonellotto et al., "Efficient In-Memory Inverted Indexes" | SIGIR 2025 tutorial | [DOI](https://dl.acm.org/doi/10.1145/3726302.3731688) |
| 8 | Kamphuis, de Vries, Boytsov, Lin, "Which BM25 Do You Mean?" | ECIR 2020 | [DOI](https://link.springer.com/chapter/10.1007/978-3-030-45442-5_4) · [PDF](https://cs.uwaterloo.ca/~jimmylin/publications/Kamphuis_etal_ECIR2020_preprint.pdf) |
| 9 | Lu, "BM25S: Orders of Magnitude Faster Lexical Search" | arXiv 2024 | [arXiv:2407.03618](https://arxiv.org/abs/2407.03618) |
| 10 | Survey: "A Survey of LSM-Tree Based Indexes" | arXiv 2024 | [arXiv:2402.10460](https://arxiv.org/abs/2402.10460) |
| 11 | Mallia, Mackenzie, Suel, Tonellotto, "Faster Learned Sparse Retrieval: Guided Traversal" | SIGIR 2022 | [DOI](https://dl.acm.org/doi/10.1145/3477495.3531774) · [arXiv:2204.11314](https://arxiv.org/abs/2204.11314) |
| 12 | Mallia, Suel, Tonellotto, "Faster Learned Sparse Retrieval: Block-Max Pruning" | SIGIR 2024 | [DOI](https://dl.acm.org/doi/10.1145/3626772.3657906) · [arXiv:2405.01117](https://arxiv.org/abs/2405.01117) |
| 13 | Bruch, Nardini, Rulli, Venturini, "Seismic" | SIGIR 2024 Best Paper | [DOI](https://dl.acm.org/doi/10.1145/3626772.3657769) · [arXiv:2404.18812](https://arxiv.org/abs/2404.18812) |
| 14 | Yafay & Altingovde, "Faster Dynamic Pruning via Reordering" | SIGIR 2023 | [DOI](https://dl.acm.org/doi/10.1145/3539618.3591987) |
| 15 | Formal et al., "SPLADE++: Towards Effective and Efficient Sparse Neural IR" | TOIS 2024 | [DOI](https://dl.acm.org/doi/10.1145/3634912) |
| 16 | Broch, Nardini et al., "Dynamic Superblock Pruning" | arXiv 2025 | [arXiv:2504.17045](https://arxiv.org/html/2504.17045v1) |
