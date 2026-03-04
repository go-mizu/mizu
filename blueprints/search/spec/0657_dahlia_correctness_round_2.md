# spec/0657 — Dahlia Correctness Round 2 (vs Tantivy CGO)

## Goal

Bring `pkg/index/driver/flower/dahlia` search results as close as practical to `tantivy` (CGO, `pkg/index/driver/tantivy-go`) on `pkg/index/bench/corpus`, validated progressively on:

- 10 docs
- 100 docs
- 1k docs
- 10k docs
- full (5,032,104 docs)

All comparisons are top-10 doc-id parity using `bench compare`.

## Setup

Reference engine for this round is **Tantivy CGO** (`engine=tantivy`, with `-tags tantivy`).

Commands used:

```bash
go run -tags tantivy ./cmd/bench index --dir "$HOME/data/search/bench/n10" --engine tantivy --docs 10 --batch-size 1000
go run -tags tantivy ./cmd/bench index --dir "$HOME/data/search/bench/n10" --engine dahlia --docs 10 --batch-size 1000
go run -tags tantivy ./cmd/bench compare --dir "$HOME/data/search/bench/n10" --engine-a dahlia --engine-b tantivy --output "$HOME/data/search/bench/n10/results/0657_n10_dahlia_vs_tantivy.json"

# repeat for n100/n1k/n10k (docs=100/1000/10000)

# full compare (using existing full indexes)
go run -tags tantivy ./cmd/bench compare \
  --dir "$HOME/data/search/bench/full" \
  --engine-a dahlia \
  --engine-b tantivy \
  --output "$HOME/data/search/bench/full/results/0657_full_dahlia_vs_tantivy.json"
```

## Round-2 Baseline (Before New Fixes)

### Small/medium sets

| Corpus | Hit queries (either) | Exact top-10 (hit view) | Exact top-10 (all view) | Avg overlap@10 (hit view) | p50/p90/p99 | Different hit count |
|---|---:|---:|---:|---:|---:|---:|
| 10 | 110 | 90/110 (81.82%) | 942/962 (97.92%) | 3.455 | 2 / 7 / 8 | 1 |
| 100 | 281 | 183/281 (65.12%) | 864/962 (89.81%) | 4.954 | 4 / 10 / 10 | 2 |
| 1k | 448 | 250/448 (55.80%) | 764/962 (79.42%) | 6.567 | 8 / 10 / 10 | 1 |
| 10k | 613 | 249/613 (40.62%) | 598/962 (62.16%) | 6.871 | 9 / 10 / 10 | 1 |

Artifacts:

- `/Users/apple/data/search/bench/n10/results/0657_n10_dahlia_vs_tantivy.json`
- `/Users/apple/data/search/bench/n100/results/0657_n100_dahlia_vs_tantivy.json`
- `/Users/apple/data/search/bench/n1k/results/0657_n1k_dahlia_vs_tantivy.json`
- `/Users/apple/data/search/bench/n10k/results/0657_n10k_dahlia_vs_tantivy.json`

### Full set

| Corpus | Hit queries (either) | Exact top-10 (all/hit) | Avg overlap@10 | p50/p90/p99 | Different hit count |
|---|---:|---:|---:|---:|---:|
| full | 962 | 112/962 (11.64%) | 7.787 | 9 / 10 / 10 | 0 |

Artifact:

- `/Users/apple/data/search/bench/full/results/0657_full_dahlia_vs_tantivy.json`

## Diagnosis

Query-type breakout on full baseline showed the issue was concentrated in **plain multi-term queries**:

- `must`: avg overlap 9.53
- `phrase`: avg overlap 9.68
- `plain`: avg overlap **3.84**

This pointed to disjunction (`should`) behavior rather than general analyzer/index mismatch.

### Root cause 1: incorrect OR scoring path in Dahlia

`searchBooleanShould` used a WAND-like loop that could score candidates with only a subset of clause contributions while other clause cursors remained behind. This distorts ranking for plain OR queries and diverges from Tantivy’s additive BM25 disjunction behavior.

### Root cause 2: token loss in query parsing

`parseQuery` analyzed each token but only kept the first analyzed term (`terms[0]`). For punctuation-splitting tokens (e.g. `wi-fi`) this dropped terms and reduced parity.

## Implemented Changes

### 1) Replace fragile WAND OR path with exact disjunction accumulation

File:

- `pkg/index/driver/flower/dahlia/wand.go`

Change summary:

- `searchBooleanShould` now performs exact union scoring by iterating doc-at-a-time on the minimum current docID.
- Per-doc score is full additive BM25 sum across all matching term clauses.
- Non-term `should` clauses are now actually merged into doc scores (previously evaluated then discarded).
- Keeps top-K via heap, preserving correctness first.

### 2) Keep all analyzed terms per query token

File:

- `pkg/index/driver/flower/dahlia/query.go`

Change summary:

- For non-quoted tokens, all analyzed terms are appended as clauses (`must/should/mustNot`) instead of dropping all but first.

### 3) Regression tests

Files:

- `pkg/index/driver/flower/dahlia/wand_test.go`
- `pkg/index/driver/flower/dahlia/query_test.go`

Added:

- `TestWandBooleanShouldAccumulatesTermScores`: ensures a doc matching both terms ranks above one matching only one term.
- `TestParseQueryExpandsAnalyzedTermsPerToken`: ensures split token (`wi-fi`) becomes two clauses.

## Results After Fixes

### Small/medium sets (rerun)

| Corpus | Exact top-10 (hit view) | Exact top-10 (all view) | Avg overlap@10 (hit view) |
|---|---:|---:|---:|
| 10 | 90/110 (81.82%) | 942/962 (97.92%) | 3.455 |
| 100 | 186/281 (66.19%) | 867/962 (90.12%) | 5.064 |
| 1k | 276/448 (61.61%) | 790/962 (82.12%) | 6.835 |
| 10k | 274/613 (44.70%) | 623/962 (64.76%) | 7.357 |

Artifacts:

- `/Users/apple/data/search/bench/n10/results/0657b_n10_dahlia_vs_tantivy.json`
- `/Users/apple/data/search/bench/n100/results/0657b_n100_dahlia_vs_tantivy.json`
- `/Users/apple/data/search/bench/n1k/results/0657b_n1k_dahlia_vs_tantivy.json`
- `/Users/apple/data/search/bench/n10k/results/0657b_n10k_dahlia_vs_tantivy.json`

### Full set (rerun)

| Corpus | Exact top-10 (all/hit) | Avg overlap@10 | p50/p90/p99 | Different hit count |
|---|---:|---:|---:|---:|
| full | 152/962 (15.80%) | 9.561 | 10 / 10 / 10 | 0 |

Artifact:

- `/Users/apple/data/search/bench/full/results/0657b_full_dahlia_vs_tantivy.json`

## Key Outcome

Main correctness gap from round 2 was plain OR ranking drift. After fixing OR accumulation and token expansion:

- full avg overlap@10 improved **7.787 -> 9.561** (+1.774)
- full exact top-10 improved **11.64% -> 15.80%**
- plain-query full overlap moved from major outlier to near parity (now ~9.50 average overlap)

Hit-count parity remained stable (`different_hit_count = 0` on full).

## Notes

- This round targets **correctness parity**, not speed. Exact OR accumulation is intentionally correctness-oriented.
- Existing `cmd/bench` compare/report tooling was used end-to-end to produce all artifacts above.
