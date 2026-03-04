# spec/0659 — Dahlia Correctness Round 3 (Path to 100% Exact Top-10)

## Ultimate Goal

Reach **100% exact top-10 compatibility** between:

- `pkg/index/driver/flower/dahlia`
- `pkg/index/driver/tantivy-go` (`engine=tantivy`, CGO)

on `pkg/index/bench/corpus` queries (962 queries), validated progressively and on full corpus.

## Baseline for Round 3 (Step 0)

Starting point is the result from round-2 post-fix run:

- artifact: `/Users/apple/data/search/bench/full/results/0657b_full_dahlia_vs_tantivy.json`
- full exact top-10: **152 / 962 (15.80%)**
- full avg overlap@10: **9.561**
- different hit count: **0**

Important decomposition (Step 0):

- total queries: 962
- exact: 152
- overlap@10 = 10: 641
- set-equal top-10 (order-insensitive): 641

Implication: large remaining gap is mostly **ordering/scoring parity**, not recall.

---

## Step 1 — Apply MUST_NOT to should-only booleans + deterministic tie ordering

### Hypothesis

1. `mustNot` was not enforced when query had only `should` clauses (no `must`) in `searchBooleanShould`.
2. Equal-score ordering was non-deterministic (score-only comparison), creating avoidable exact mismatches.

### Code changes

Files:

- `pkg/index/driver/flower/dahlia/wand.go`
- `pkg/index/driver/flower/dahlia/wand_test.go`

Changes:

1. `searchBooleanShould` now builds and enforces `mustNotSet` (term + phrase exclusions).
2. Introduced deterministic score tie behavior in scorer heap/sort path.
3. Added regression test: `TestWandShouldWithMustNot`.

### Step-1 full result

- artifact: `/Users/apple/data/search/bench/full/results/0659_step1_full_dahlia_vs_tantivy.json`
- full exact top-10: **164 / 962 (17.05%)**
- full avg overlap@10: **9.561**
- different hit count: **0**

Delta vs Step 0:

- exact top-10: **+12 queries** (152 -> 164)
- overlap unchanged, indicating this step primarily improved **ordering/boolean exclusion correctness**.

### Step-1 small-set checkpoints

Artifacts:

- `/Users/apple/data/search/bench/n10/results/0659_step1_n10_dahlia_vs_tantivy.json`
- `/Users/apple/data/search/bench/n100/results/0659_step1_n100_dahlia_vs_tantivy.json`
- `/Users/apple/data/search/bench/n1k/results/0659_step1_n1k_dahlia_vs_tantivy.json`
- `/Users/apple/data/search/bench/n10k/results/0659_step1_n10k_dahlia_vs_tantivy.json`

Summary:

- `n10`: exact-hit 99, exact-all 951, avg-overlap 3.455
- `n100`: exact-hit 197, exact-all 878, avg-overlap 5.060
- `n1k`: exact-hit 286, exact-all 800, avg-overlap 6.835
- `n10k`: exact-hit 271, exact-all 620, avg-overlap 7.372

---

## Current Gap to 100%

After Step 1, remaining non-exact queries on full:

- `962 - 164 = 798`

Key signal:

- overlap is already very high (`p50/p90/p99 = 10/10/10`), so the remaining gap is dominated by **fine-grained ranking parity**, not hit-set parity.

## Next Planned Steps

1. Align Dahlia ranking tie-break semantics with Tantivy more precisely (doc-address style behavior).
2. Audit BM25 numeric path against Tantivy internals (idf precision, norm encoding/decoding behavior).
3. Add ranking-focused golden tests from real mismatch queries (full corpus snapshots) to prevent regressions while moving exact@10 upward.

Round 3 will continue with incremental steps, each with:

- implementation
- benchmark rerun
- committed code
- spec update with exact metrics delta
