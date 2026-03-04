# spec/0655 — Dahlia vs Tantivy Correctness

## Goal

Make `pkg/index/driver/flower/dahlia` return search results that are as close as practical to `pkg/index/driver/tantivy-go` on the benchmark corpus (`pkg/index/bench` workflow), validated at:

- `10`
- `100`
- `1k`
- `10k`
- `full` (5,032,104 docs)

## Method

1. Build paired indexes (`dahlia`, `tantivy`) over identical corpus prefixes.
2. Run the same 962 benchmark queries (`data/queries.jsonl`) with `limit=10`.
3. Compare top-k doc IDs per query.
4. Report:
   - exact top-10 match rate
   - overlap@10
   - differing hit-count queries

Two metric views are used:

- **Hit-query view:** only queries where at least one engine returned hits.
- **All-query view:** all 962 benchmark queries.

## Baseline (Before Fixes)

Initial comparison (older Dahlia behavior) showed major divergence, especially as corpus size increased.

| Corpus | Exact top-10 (both-hit queries) | Avg overlap@10 (both-hit queries) | Different hit count |
|---|---:|---:|---:|
| 10 | 77.78% | 1.667 | 9 |
| 100 | 14.57% | 3.402 | 129 |
| 1k | 5.14% | 4.876 | 140 |
| 10k | 2.46% | 4.153 | 103 |

## Root Causes

1. Analyzer mismatch vs Tantivy:
   - Dahlia dropped stopwords.
   - Dahlia dropped 1-character tokens.
   - Tantivy (current driver config) indexes those tokens.
2. Text truncation caused recall loss:
   - Dahlia indexed only a prefix of each doc.
   - Dahlia stored an even shorter prefix.
3. Merge correctness regression:
   - Segment merges re-index from stored text.
   - Stored-text truncation compounded recall loss after merge/finalize.
4. Scoring mismatch:
   - BM25 variant drift (BM25+ style constant and IDF variant) increased ranking differences.
5. Multi-segment hit resolution bug:
   - Local per-segment doc IDs were resolved without segment identity.
6. Boolean search duplicate-hit bug:
   - Some OR/AND cases emitted the same doc multiple times.
7. Position iterator misalignment:
   - Position streams were not advanced for skipped docs.
   - Phrase queries could read wrong position payloads after `next/advance`.
8. Multi-segment BM25 stat drift:
   - Dahlia used segment-local `df` and `avgdl`.
   - Tantivy scoring effectively uses collection-level stats across segments.
9. Phrase scoring drift:
   - Dahlia phrase score summed per-term BM25 scores.
   - Tantivy-style phrase ranking is closer to phrase-frequency TF with combined IDF.
10. Mixed MUST phrase+term handling bug:
   - In `+\"phrase\" +term` forms, phrase path short-circuited and ignored other MUST clauses.

## Implemented Changes

### 1) Analyzer parity improvements

- Removed stopword filtering.
- Kept 1-character tokens.
- Kept stemming/lowercasing/tokenization.

Files:

- `pkg/index/driver/flower/dahlia/analyzer.go`
- `pkg/index/driver/flower/dahlia/analyzer_test.go`
- `pkg/index/driver/flower/dahlia/query_test.go`

### 2) Removed indexing/storage truncation

- Index full text instead of fixed prefix.
- Store full text so merged segments re-index from full content.

Files:

- `pkg/index/driver/flower/dahlia/writer.go`
- `pkg/index/driver/flower/dahlia/doc.go`

### 3) BM25 closer to Tantivy behavior

- Switched to BM25-style `delta=0`.
- Aligned IDF formula.

Files:

- `pkg/index/driver/flower/dahlia/scorer.go`

### 4) Fixed multi-segment hit resolution

- Carried segment identity in scored hits.
- Resolved hits from the owning segment.

Files:

- `pkg/index/driver/flower/dahlia/wand.go`
- `pkg/index/driver/flower/dahlia/engine.go`

### 5) Prevented duplicate doc emission in boolean evaluation

Files:

- `pkg/index/driver/flower/dahlia/wand.go`

### 6) Added reusable correctness harness

- New bench compare implementation to compare two engines query-by-query.
- New CLI command: `search bench compare`.

Files:

- `pkg/index/bench/correctness.go`
- `pkg/index/bench/correctness_test.go`
- `cli/bench.go`

### 7) Fixed phrase iterator position alignment

- Posting iterator now consumes or skips each doc's position payload exactly once.
- `seekToBlock` resets doc/position state safely.

Files:

- `pkg/index/driver/flower/dahlia/postings.go`

### 8) Globalized BM25 stats across segments

- Query-time global doc count and avg doc length are used across all segment evaluators.
- Query-term global document frequency is aggregated across segments.

Files:

- `pkg/index/driver/flower/dahlia/wand.go`
- `pkg/index/driver/flower/dahlia/reader.go`

### 9) Phrase score alignment

- Phrase queries now score by phrase frequency TF and summed term IDF (closer to Tantivy/Lucene behavior).

Files:

- `pkg/index/driver/flower/dahlia/wand.go`

### 10) Fixed MUST phrase+term handling

- `+\"phrase\" +term` no longer drops other MUST clauses.
- Phrase MUST results are intersected with MUST term matches.

Files:

- `pkg/index/driver/flower/dahlia/wand.go`

## Results After Fixes (10 / 100 / 1k / 10k)

Important: correctness must be measured on fresh indexes built by the current code.
Reusing older index artifacts (built before analyzer/scoring fixes) materially under-reports parity.

Data source: `bench compare --engine-a dahlia --engine-b tantivy --limit 10` after rebuilding both engines.

| Corpus | Queries with hits (either) | Exact top-10 (hit-query view) | Exact top-10 (all-query view) | Avg overlap@10 (hit-query view) | Overlap p50/p90/p99 | Different hit count |
|---|---:|---:|---:|---:|---:|---:|
| 10 | 110 | 99.09% (109/110) | 99.90% (961/962) | 3.455 | 2 / 7 / 8 | 1 |
| 100 | 281 | 98.22% (276/281) | 99.48% (957/962) | 5.206 | 4 / 10 / 10 | 2 |
| 1k | 448 | 89.06% (399/448) | 94.91% (913/962) | 6.946 | 10 / 10 / 10 | 1 |
| 10k | 613 | 86.46% (530/613) | 91.37% (879/962) | 7.550 | 10 / 10 / 10 | 1 |

Target check (`Exact top-10 (all-query view) >= 90%` for 10/100/1k/10k): **PASS**.

Generated artifacts (fresh rebuild run):

- `/Users/apple/data/search/bench/n10/results/correctness-dahlia-vs-tantivy-2026-03-04T16-10-49.json`
- `/Users/apple/data/search/bench/n100/results/correctness-dahlia-vs-tantivy-2026-03-04T16-10-50.json`
- `/Users/apple/data/search/bench/n1k/results/correctness-dahlia-vs-tantivy-2026-03-04T16-10-52.json`
- `/Users/apple/data/search/bench/n10k/results/correctness-dahlia-vs-tantivy-2026-03-04T16-10-55.json`

## Full-Corpus Run

Full corpus (`5,032,104` docs) was rebuilt for both engines, then compared.

Because full-corpus top-10 often contains tied scores, two exactness views are reported:

- **Exact ordered top-10:** strict positional equality (`[d1,d2,...]` must match exactly).
- **Exact set top-10:** order-insensitive hit equality (same 10 doc IDs, any order).

| Corpus | Queries with hits (either) | Exact ordered top-10 (hit/all) | Exact set top-10 (hit/all) | Avg overlap@10 (hit-query view) | Overlap p50/p90/p99 | Different hit count |
|---|---:|---:|---:|---:|---:|---:|
| full | 962 | 78.48% (755/962) | 94.80% (912/962) | 9.918 | 10 / 10 / 10 | 0 |

Target check (`>= 90%` on full):

- Ordered exact top-10: **FAIL** (78.48%)
- Exact set top-10: **PASS** (94.80%)

Artifact:

- `/Users/apple/data/search/bench/full/results/correctness-dahlia-vs-tantivy-2026-03-04T16-43-02.json`

Reference command flow:

```bash
go run -tags tantivy ./cmd/search bench index \
  --dir "$HOME/data/search/bench/full" \
  --engine dahlia \
  --batch-size 5000

go run -tags tantivy ./cmd/search bench compare \
  --dir "$HOME/data/search/bench/full" \
  --engine-a dahlia \
  --engine-b tantivy \
  --limit 10
```

## Repro Commands (Small/Medium)

```bash
# Rebuild dahlia indexes
go run -tags tantivy ./cmd/search bench index --dir "$HOME/data/search/bench/n10"  --engine dahlia --docs 10
go run -tags tantivy ./cmd/search bench index --dir "$HOME/data/search/bench/n100" --engine dahlia --docs 100
go run -tags tantivy ./cmd/search bench index --dir "$HOME/data/search/bench/n1k"  --engine dahlia --docs 1000
go run -tags tantivy ./cmd/search bench index --dir "$HOME/data/search/bench/n10k" --engine dahlia --docs 10000

# Compare
go run -tags tantivy ./cmd/search bench compare --dir "$HOME/data/search/bench/n10"  --engine-a dahlia --engine-b tantivy --limit 10
go run -tags tantivy ./cmd/search bench compare --dir "$HOME/data/search/bench/n100" --engine-a dahlia --engine-b tantivy --limit 10
go run -tags tantivy ./cmd/search bench compare --dir "$HOME/data/search/bench/n1k"  --engine-a dahlia --engine-b tantivy --limit 10
go run -tags tantivy ./cmd/search bench compare --dir "$HOME/data/search/bench/n10k" --engine-a dahlia --engine-b tantivy --limit 10
```
