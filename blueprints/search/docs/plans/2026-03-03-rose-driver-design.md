# Rose FTS Driver — Design Document
**Date:** 2026-03-03
**Status:** Approved
**Spec:** spec/0646_rose_driver.md

## What We're Building

A pure-Go, from-scratch full-text search engine implementing `pkg/index.Engine`,
registered as `"rose"`.  No CGO, no external services, no new dependencies beyond what
is already in `go.mod`.

## Approach

**Approach B: VByte Delta + Block-Max WAND + Quantised BM25+** (see spec for A and C).

## Key Components

1. **Analyzer** — Unicode tokenise → lowercase → 127-word English stopword filter →
   Snowball English stem (kljensen/snowball).
2. **In-memory buffer** — `map[term][]memPosting`; flushes at 64 MB to a `.seg` file.
3. **Segment format** — sorted term dictionary + VByte-delta-encoded 128-posting blocks,
   each with a `BlockMaxImpact` uint8 header.
4. **BM25+ scoring** — Robertson variant with delta=1.0; quantised to uint8 per posting.
5. **Block-Max WAND** — top-k retrieval with per-block upper-bound skipping;
   80–95% posting reduction on selective queries (ref: Grand et al. ECIR 2020).
6. **LSM tiered merge** — background goroutine; tier triggers at ≥4 segments;
   N-way sorted merge preserves immutability of existing segments.
7. **Doc store** — append-only, stores first 512 bytes per doc for snippet generation.

## Files

```
pkg/index/driver/flower/rose/
  RESEARCH.md    deep technical notes + paper summaries
  index.go       Engine entry point
  analyzer.go    text analysis
  segment.go     segment flush + read
  postings.go    VByte codec + block packing
  bm25.go        BM25+ + quantisation
  wand.go        Block-Max WAND cursor + top-k loop
  merge.go       LSM tiered merge
  docstore.go    doc store + snippet
  rose_test.go   tests
```

## Success Criteria

- `go test ./pkg/index/driver/flower/rose/...` passes
- Index throughput > 10,000 docs/s on 173,720-doc CC corpus (4× bleve)
- Search avg latency < 10ms on 10-query standard set
- Disk usage ≤ 512 MB on 173,720-doc corpus
