# Dahlia: Tantivy Architecture Research

## Overview

Tantivy is a Rust full-text search library inspired by Apache Lucene. This document
analyzes tantivy's architecture to guide the Dahlia pure-Go implementation.

## Segment Architecture

Tantivy stores an index as a collection of immutable **segments**. Each segment is a
self-contained mini-index. New documents are buffered in memory and flushed as new
segments. Background merge compacts multiple segments into one.

### Segment Files
Each segment contains:
- **`.term`** — FST-based term dictionary mapping term bytes → term ordinal
- **`.postings`** — Compressed posting lists (doc IDs + term frequencies)
- **`.pos`** — Position data for phrase queries
- **`.store`** — Stored field values (compressed blocks)
- **`.fieldnorm`** — Field length norms for BM25 scoring
- **`.fast`** — Columnar fast fields (not used in Dahlia v1)

### Index Meta (`meta.json`)
Tracks: segment list, doc count, opstamp, schema. Atomic updates via temp+rename.

## Posting List Compression: BP128 (Bitpacking)

Tantivy uses **bitpacking** for posting list compression, processing 128 integers at a time.

### Algorithm
1. Compute required bit-width `b = ceil(log2(max(deltas) + 1))`
2. Pack 128 values into `b * 128 / 8 = 16b` bytes
3. Store bit-width as 1-byte header

### Properties
- Block size: 1 + 16b bytes (1 byte header + data)
- Zero case: 1 byte (all deltas are 0)
- Max case: 513 bytes (32 bits × 128 values)
- Delta encoding: doc IDs stored as deltas from previous

### Tail Handling
When fewer than 128 docs remain, tantivy uses VByte encoding for the "tail".

## Variable-Byte Integer Encoding (VByte/VInt)

Standard variable-byte encoding used for tail values, position deltas, and skip metadata.

### Format
- 7 data bits per byte, MSB = continuation flag
- 1 byte: 0–127
- 2 bytes: 128–16383
- 5 bytes max for uint32

## Term Dictionary: Finite State Transducer (FST)

Tantivy uses an FST (via the `fst` crate) for term→termInfo mapping.

### Properties
- Shared prefix compression (extremely space-efficient for sorted terms)
- O(key_length) lookup time
- Supports range iteration and prefix queries
- Values packed as uint64 (docFreq + postings offset + flags)

### TermInfo Packing
Dahlia packs into uint64:
- Bits [0:30] — docFreq (up to 1 billion)
- Bit [30] — hasPositions flag
- Bits [32:63] — postings offset in .doc file

## Skip Lists for Block-Level Seeking

### Skip Entry Format (Dahlia: 21 bytes)
```
lastDoc:      uint32  (4 bytes) — highest docID in the block
docOff:       uint32  (4 bytes) — byte offset into .doc file
freqOff:      uint32  (4 bytes) — byte offset into .freq file
posOff:       uint32  (4 bytes) — byte offset into .pos file
blockMaxTF:   uint32  (4 bytes) — max term frequency in block (WAND)
blockMaxNorm: uint8   (1 byte)  — shortest-doc norm in block (WAND)
```

Key improvement over lotus: separate offsets for .doc/.freq/.pos files,
enabling independent seeking into each file.

### Tantivy Skip List Usage
- Binary search on skip entries to find target block
- `advance(target)` skips O(log N) blocks instead of scanning O(N) docs
- Block-Max WAND uses `blockMaxTF` + `blockMaxNorm` for upper-bound scoring

## Block-Max WAND (Weighted AND)

### Paper Reference
Ding & Suel, "Faster Top-k Document Retrieval Using Block-Max Indexes" (SIGIR 2011)

### Algorithm
1. Sort posting cursors by current docID (ascending)
2. Compute prefix sum of upper-bound block scores
3. Find "pivot" where cumulative upper-bound ≥ threshold
4. If pivot doc is in all prefix cursors' current blocks, score it fully
5. Otherwise, advance the cursor with lowest upper-bound past pivot
6. **Block skipping**: If a block's upper-bound score can't contribute enough
   to exceed the K-th threshold, skip the entire 128-doc block

### Dahlia's True Block-Max WAND
Lotus defines `blockMaxImpact()` but never uses it for skip decisions.
Dahlia computes per-block upper-bound scores and skips blocks that
cannot contribute to top-K results, yielding significant speedup on
disjunctive (OR) queries.

## BM25+ Scoring

### Formula
```
score(q,d) = Σ_t IDF(t) × [ TF_norm(t,d) + δ ]
```
Where:
- `IDF(t) = ln(1 + (N - df + 0.5) / (df + 0.5))`
- `TF_norm(t,d) = (tf × (k1 + 1)) / (tf + k1 × (1 - b + b × dl/avgdl))`
- `δ = 1.0` (BM25+ additive constant, ensures positive TF contribution)
- `k1 = 1.2`, `b = 0.75` (standard parameters)

### Field Norms (Lucene SmallFloat)
Document lengths are encoded as single bytes using Lucene's SmallFloat encoding:
- Values 0–23: stored losslessly
- Values ≥ 24: 4-bit mantissa + exponent (lossy but monotonic)

BM25 norm component `k1 × (1 - b + b × dl/avgdl)` is precomputed as a 256-entry
lookup table for all possible norm byte values.

## Position Indexing for Phrase Queries

### Storage
- Positions stored as delta-encoded VInt sequences per document
- Each term's positions for a doc: `[numPositions VInt][delta0 VInt][delta1 VInt]...`
- `.pos` file contains all position data, skip entries have posOff for seeking

### Phrase Query Evaluation
1. Intersect posting lists (all terms must appear in doc)
2. Load position lists for each term
3. Check adjacency: `pos[i+1] - pos[i] == 1` for consecutive query terms
4. Generalized: for query term at offset k, check `pos[k] - pos[0] == k`

## Stored Fields: Compressed Block Store

### Format
- Documents serialized as `[idLen:4][id:N][textLen:4][text:M]`
- Grouped into 16KB blocks, each block zstd-compressed
- Skip index at end of file: `[lastDocID:4][blockOffset:8]` per block
- Footer: 8-byte skip index offset

### Retrieval
1. Binary search skip index for target docID's block
2. Decompress single block
3. Linear scan within block for target doc

## Merge Policy: Tiered Merge

### Tantivy's Approach
- Segments grouped by size tier (log-scale)
- Merge triggered when a tier has too many segments (default: 10)
- Merge selects segments within same tier for compaction
- Background thread performs merge, atomically swaps segment list

### Dahlia's Implementation
- Trigger: > maxSegmentsBeforeMerge (default 10) segments
- Select: smallest segments first (up to maxMergeSegments = 10)
- N-way merge: iterate all terms across segments, rebuild postings
- Atomic swap: write new segment, update meta, delete old segments

## References

1. Ding & Suel, "Faster Top-k Document Retrieval Using Block-Max Indexes", SIGIR 2011
2. Lemire & Boytsov, "Decoding billions of integers in milliseconds through vectorized bit packing", Software: Practice and Experience, 2015
3. Robertson & Zaragoza, "The Probabilistic Relevance Framework: BM25 and Beyond", 2009
4. Lv & Zhai, "Lower-Bounding Term Frequency Normalization", CIKM 2011 (BM25+)
5. Lucene SmallFloat: org.apache.lucene.util.SmallFloat
6. Tantivy source: https://github.com/quickwit-oss/tantivy
