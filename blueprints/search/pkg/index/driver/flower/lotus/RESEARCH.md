# Lotus RESEARCH.md — Tantivy Architecture Reference

This document captures the key architectural ideas from tantivy (Rust full-text search
engine) that Lotus reproduces in pure Go. Each section contains enough detail to
implement the component from scratch.

---

## 1. Segment File Format

A tantivy index is a collection of **segments**, each identified by a UUID. Each
segment is self-contained with its own term dictionary, posting lists, and supporting
structures.

| Extension | Purpose |
|-----------|---------|
| `.term` | Term dictionary (FST + TermInfoStore) |
| `.idx` | Posting lists (doc IDs + term frequencies) |
| `.pos` | Term position data (for phrase queries) |
| `.fieldnorm` | Field norms (1 byte per doc per field, for BM25) |
| `.store` | Stored document fields (LZ4/Zstd compressed blocks) |
| `.fast` | Fast fields (column-oriented, bitpacked) |
| `.del` | Alive bitset / deletion tombstone |

`meta.json` at the index root stores the schema and committed segment list.

Within a segment, documents receive compact sequential `DocId` values `[0, max_doc)`.
This dense numbering is critical for all compact encoding schemes.

**Lotus adaptation:** We use a similar per-segment directory structure with extensions
`.tdi` (FST), `.doc` (docID deltas), `.freq` (term frequencies), `.pos` (positions),
`.store` (stored fields), `.fnm` (field norms), `.meta` (segment metadata).

---

## 2. BP128 Bitpacking (the `bitpacking` / `tantivy-bitpacker` crates)

Tantivy's posting list compression is built on Daniel Lemire's **simdcomp** library.
The core idea is **block-based bitpacking of 128 integers**.

### Algorithm

1. **Delta encoding**: For sorted sequences (doc IDs), replace each value with the
   difference from the previous: `[1, 3, 7, 8]` → `[1, 2, 4, 1]`.

2. **Bit-width determination**: Find `max(block)`, compute
   `num_bits = ceil(log2(max + 1))`. This is bits needed per integer.

3. **Bitpacking**: Pack all 128 integers using exactly `num_bits` bits each. The
   bit-width is stored as a 1-byte header.

4. **Output size**: `1 + num_bits × 16` bytes per 128-integer block.

### SIMD Layout (BitPacker4x)

SSE3: processes 4×u32 per instruction. Instead of packing sequentially, it interleaves:
positions `[0,1,2,3]` packed together, then `[4,5,6,7]`, etc.

- BitPacker4x: 128-bit SSE3, 4×u32/instruction
- BitPacker8x: 256-bit AVX2, 8×u32/instruction
- Scalar fallback: identical logic on single u32s (ARM, WASM, older x86)

### Performance

- Scalar: 1.48 billion integers/sec
- SSE3: 6 billion integers/sec

### Lotus adaptation

Pure-Go scalar implementation. For each of 128 integers, extract `num_bits` bits at
bit offset `i × num_bits` using shift-and-mask on uint32/uint64 words. No SIMD, but
still ~3–5× faster than VByte due to cache-friendly packed layout.

**Sources:**
- [Of bitpacking with or without SSE3 — fulmicoton](https://fulmicoton.com/posts/bitpacking/)
- [bitpacking crate — GitHub](https://github.com/tantivy-search/bitpacking)
- [tantivy-bitpacker docs](https://docs.rs/tantivy-bitpacker/0.3.0/tantivy_bitpacker/)

---

## 3. FST Term Dictionary

The term dictionary has **two components** in the `.term` file:

### Component 1: FST (Finite State Transducer)

Built using BurntSushi's `fst` crate (Go equivalent: `blevesearch/vellum`). Maps each
term (`[]byte`) to a `uint64` value — either a term ordinal or an offset into the
TermInfoStore.

FSTs share both prefixes AND suffixes (unlike tries). Key advantage: can definitively
determine a term does NOT exist (no false positives). Typical size: ~3 bytes/term on
Wikipedia.

### Component 2: TermInfoStore

Conceptually a vector of `TermInfo` structs ordered by term ordinal, with delta
compression and block-based encoding:

```
TermInfo {
    doc_freq: u32,                    // docs containing the term
    postings_range: Range<usize>,     // byte range in .idx
    positions_range: Range<usize>,    // byte range in .pos
}
```

**Block layout:**
1. **Metadata record** (fixed-size, per block): offset, ref TermInfo, bitwidths
2. **Data block** (variable): delta-encoded TermInfos, bitpacked per field

**Lookup:** FST → ordinal → `block_index = ordinal / block_size` → read metadata →
unpack delta → add to ref TermInfo.

### Lotus adaptation

We use vellum directly and pack `termInfo` into the uint64 output value:
- bits [0..30] = docFreq
- bit [31] = hasPositions
- bits [32..63] = postingsOffset

For the scratch FST, we implement Daciuk's incremental suffix-sharing algorithm.

**Sources:**
- [Term dictionary — tantivy-doc](https://fulmicoton.gitbooks.io/tantivy-doc/content/term-dictionary.html)
- [Inverted index deep dive — search-benchmark-game wiki](https://github.com/Tony-X/search-benchmark-game/wiki/Inverted-index-deep-dive)

---

## 4. Skip List + Block WAND

The skip list is stored inline within the `.idx` file alongside posting data. It
enables O(log N) seeking within a posting list.

### Skip Entry Format

For each 128-doc block:
- `last_doc_in_block: u32` (LE) — highest docID in this block
- `doc_num_bits: u8` — bit-width of delta-encoded doc IDs
- For WithFreqs: `tf_num_bits: u8` + block WAND data
- For WithPositions: `position_offset: u64`

### Bitwidth encoding

```
encode: bitwidth | ((delta_1 as u8) << 6)
decode: bitwidth = raw & 0x3F, delta_1 = (raw >> 6) & 1
```

### Block WAND

Divides each posting list into 128-doc blocks. Per block, stores a **local maximum
score** — the maximum BM25 score any document in that block could achieve.

Block-max data embedded in skip entries:
- `block_wand_fieldnorm_id: u8` — fieldnorm that produces highest score
- `block_wand_term_freq: u32` — max TF in block

**Query processing:**
1. Maintain min-heap of top-K, K-th score as threshold
2. Per block, compute max possible score from block WAND data
3. If block max < threshold → skip entire 128-doc block
4. Otherwise → decompress and score all docs in block

**Searching within a block:** After decompressing 128 sorted docIDs, use branchless
binary search on `[128]uint32`. LLVM unrolls to ~9.55 CPU cycles → 10% improvement
on intersection queries.

### Lotus adaptation

Skip entries stored after posting data in `.doc` file. Each entry records lastDoc,
byte offsets, max TF, and max fieldnorm for Block-Max WAND scoring.

**Sources:**
- [Block WAND — tantivy issue #2390](https://github.com/quickwit-oss/tantivy/issues/2390)
- [A cool Rust optimization — Quickwit](https://quickwit.io/blog/search-a-sorted-block)

---

## 5. Stored Fields (.store)

Row-oriented storage for `STORED` fields. Used to retrieve original document content
for display in search results.

### Block-Based Architecture

1. Documents serialized (field values as bytes), appended to buffer
2. When buffer exceeds **16 KB**, compress (LZ4 or Zstd) and write as one block
3. Last block flushed on segment finalization

### File Layout

```
[Block 0 (compressed)] [Block 1] ... [Block N] [Skip Index] [Footer]
```

**Footer** (at file end):
- `decompressor`: enum (LZ4/Zstd/None)
- `offset: u64`: byte offset where skip index starts

**Skip Index:** list of `(last_doc_id, byte_offset)` pairs.

### Retrieval

1. Binary search skip index → find block containing target docID
2. Read compressed block from disk
3. Decompress entire block
4. Scan decompressed data for target doc's fields

Performance: relatively slow (decompresses whole 16KB block). Recommended: max ~100
docs fetched per query. LRU cache by block range avoids redundant decompression.

### Lotus adaptation

Same design with Zstd via `klauspost/compress`. 512-doc blocks (slightly larger than
tantivy's 16KB blocks). Skip index at end, footer with skip offset.

**Sources:**
- [Store — tantivy-doc](https://fulmicoton.gitbooks.io/tantivy-doc/content/store.html)
- [tantivy::store module](https://docs.rs/tantivy/latest/tantivy/store/index.html)

---

## 6. Field Norm Encoding (Lucene SmallFloat)

Field norms store the **token count** per document per field. BM25 needs this to
penalize longer documents. Encoded as 1 byte using Lucene's `SmallFloat.intToByte4`.

### Encoding Algorithm

**Constants:** `NUM_FREE_VALUES = 24`, `MAX_INT4 = 231`

```
intToByte4(i):
    if i < 24: return i                    // lossless for 0..23
    return 24 + longToInt4(i - 24)

longToInt4(i):
    numBits = 32 - leadingZeros(i)
    if numBits < 4: return i               // subnormal
    shift = numBits - 4
    encoded = (i >>> shift) & 0x07         // 3-bit mantissa, clear implicit MSB
    encoded |= (shift + 1) << 3           // encode exponent
    return encoded
```

### Key Properties

- Values 0–23 encoded losslessly
- Beyond 24: 3-bit mantissa + variable exponent (~12.5% precision)
- Monotonic: larger norms → larger byte values
- 256 possible byte values → 256 distinct fieldnorm values

### BM25 Integration

Precompute 256-entry table: for each byte value b, compute:
```
table[b] = K1 × (1 - B + B × fieldNormDecode(b) / avg_fieldnorm)
```
Avoids per-document float math during scoring.

### Lotus adaptation

Exact port of Lucene's `intToByte4`/`byte4ToInt`. 256-entry precomputed BM25 norm
table at search time.

**Sources:**
- [SmallFloat.java — Lucene](https://github.com/apache/lucene/blob/main/lucene/core/src/java/org/apache/lucene/util/SmallFloat.java)
- [tantivy bm25.rs](https://github.com/quickwit-oss/tantivy/blob/main/src/query/bm25.rs)

---

## 7. Posting List Binary Format (.idx)

Each term's posting list occupies a contiguous byte range identified by
`TermInfo.postings_range`.

### Full blocks (128 docs)

1. **Doc IDs:** delta-encoded, then bitpacked using BP128. Bit-width recorded in
   skip entry.
2. **Term frequencies:** bitpacked (optionally with minus-1 encoding since TF ≥ 1).

### Last block (1–127 docs)

1. **Doc IDs:** delta-encoded, then VInt (variable-byte) encoded
2. **Term frequencies:** VInt encoded

### Position data (.pos)

Same block-of-128 scheme. Positions are delta-encoded within each document, then
bitpacked in blocks of 128 with VInt for remainder.

### Lotus adaptation

Separate `.doc`, `.freq`, `.pos` files (rather than interleaved in one `.idx`).
BP128 for full blocks, VInt for last partial block. Skip index appended after
posting data in `.doc`.

**Sources:**
- [compression/mod.rs — tantivy](https://docs.rs/tantivy/latest/src/tantivy/postings/compression/mod.rs.html)
- [Of tantivy, a search engine in Rust — fulmicoton](https://fulmicoton.com/posts/behold-tantivy/)

---

## Research Papers

1. **Pibiri & Venturini** — "Techniques for Inverted Index Compression", ACM Surveys 53(6), 2021.
2. **Ottaviano & Venturini** — "Partitioned Elias-Fano Indexes", SIGIR 2014.
3. **Grand, Muir, Ferenczi, Lin** — "From MaxScore to Block-Max Wand", ECIR 2020.
4. **Ding & Suel** — "Optimizing Top-k Document Retrieval Strategies for Block-Max Indexes", WSDM 2013.
5. **Mallia et al.** — "Faster Block-Max WAND with Variable-Sized Blocks", SIGIR 2017.
6. **Mallia et al.** — "PISA: Performant Indexes and Search for Academia", OSIRRC@SIGIR 2019.
7. **Kamphuis et al.** — "Which BM25 Do You Mean?", ECIR 2020.
8. **Lu** — "BM25S: Orders of Magnitude Faster Lexical Search via Eager Sparse Scoring", arXiv:2407.03618, 2024.
9. **LSM Survey** — "A Survey of LSM-Tree Based Indexes", arXiv:2402.10460, 2024.
10. **Mallia et al.** — "Efficient In-Memory Inverted Indexes: Theory and Practice", SIGIR 2025 Tutorial.
