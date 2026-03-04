# spec/0656 — Dahlia Low-RSS Research and Optimization

## Goal

Understand why Dahlia indexing RSS gets high on large corpus runs (observed around 9.7 GB on full run), study Tantivy behavior, and implement concrete memory-control changes in `pkg/index/driver/flower/dahlia`.

## Problem Statement

Observed full indexing state (user report):

- docs indexed: `~5.03M`
- throughput: `~4.45k docs/s`
- RSS: `~9.7 GB`

Two questions had to be answered:

1. Why memory climbs so much.
2. Which Tantivy patterns we should copy to stabilize Dahlia.

## Tantivy Findings (from local `tantivy-go` Rust sources)

### 1) Explicit writer memory budget

`tantivy-go` creates writer with a fixed budget:

- `DOCUMENT_BUDGET_BYTES: 50_000_000`
- `index.writer(DOCUMENT_BUDGET_BYTES)`

References:

- `$GOPATH/pkg/mod/github.com/anyproto/tantivy-go@v1.0.6/rust/src/tantivy_util/util.rs:9`
- `$GOPATH/pkg/mod/github.com/anyproto/tantivy-go@v1.0.6/rust/src/c_util/util.rs:196`

Implication: Tantivy enforces bounded in-memory indexing work and applies backpressure when that budget is saturated.

### 2) Merge lifecycle is explicitly waited/drained

Tantivy context teardown calls `wait_merging_threads()`.

Reference:

- `$GOPATH/pkg/mod/github.com/anyproto/tantivy-go@v1.0.6/rust/src/lib.rs:750-762`

Implication: merge threads are treated as first-class resource owners and not left unbounded.

## Dahlia Root Causes

### 1) Flush fan-out could create too much in-flight memory

Before this change, Dahlia could launch many background flushes while continuing ingestion. Each flush has temporary allocations (term sorting + postings buffers + file buffers), so many concurrent flushes amplify RSS.

### 2) Background merge policy was too aggressive for current merge implementation

Dahlia merge currently re-reads docs and re-indexes them into a fresh in-memory `segmentWriter` (`mergeSegments`), which is expensive in both CPU and memory.

Aggressive merge triggering (small segment threshold, high fan-in) caused long merge phases and memory plateaus/spikes.

### 3) Force-merge could create a huge one-shot working set

`Finalize` previously merged all segments in one pass. With doc re-index merge, this can produce very large temporary memory pressure.

## Implemented Changes

### 1) Bounded background flush concurrency (backpressure)

File:

- `pkg/index/driver/flower/dahlia/engine.go`

Change:

- Added `flushSlots chan struct{}` and `maxFlushWorkers=1`.
- `Index()` now blocks on available flush slot before spawning a background flush.

Effect:

- Caps concurrent flush writers to 1, reducing temporary memory fan-out.

### 2) Conservative background merge policy

Files:

- `pkg/index/driver/flower/dahlia/doc.go`
- `pkg/index/driver/flower/dahlia/merge.go`

Changes:

- `maxSegBeforeMerge`: `10 -> 128`
- `maxMergeSegments`: `10 -> 4`
- Added `maxBgMergeDocs=12_000` hard doc budget per background merge.
- Added `pickMergeIndices(...)` to select smallest segments under doc budget.

Effect:

- Background merge is less frequent and each merge has bounded working set.
- Fewer long pauses and fewer memory spikes from large multi-segment merge jobs.

### 3) Budgeted staged force-merge

File:

- `pkg/index/driver/flower/dahlia/merge.go`

Changes:

- Replaced one-shot force merge with iterative merge rounds.
- Added `maxFMMergeDocs=30_000` cap per merge step.
- Keeps merge progress guaranteed with fallback to 2 smallest segments.

Effect:

- Finalize memory profile is bounded per step rather than all-at-once.

### 4) New tests for merge candidate selection logic

File:

- `pkg/index/driver/flower/dahlia/merge_test.go`

Added:

- `TestPickMergeIndicesRespectsBudget`
- `TestPickMergeIndicesForceProgress`

Also verified:

```bash
go test ./pkg/index/driver/flower/dahlia
go test ./cmd/bench ./cli
```

## Benchmark Results

Benchmark command pattern (same for before/after binaries):

```bash
/usr/bin/time -l /tmp/bench-<old|new> index \
  --dir /tmp/bench-rss-study \
  --engine dahlia \
  --docs <N> \
  --batch-size 5000 \
  --no-finalize
```

### 500k docs (`--no-finalize`)

| Metric | Before | After | Delta |
|---|---:|---:|---:|
| Docs indexed | 494,458 | 499,619 | +5,161 |
| Elapsed | 48.6s | 48.8s | +0.2s |
| Avg rate | 10,280 docs/s | 10,237 docs/s | -0.42% |
| Bench peak RSS | 2,157 MB | 2,088 MB | -3.20% |
| `/usr/bin/time` max resident set | 2,297,610,240 B | 2,151,301,120 B | -6.37% |

Logs:

- `/tmp/bench_rss_before.log`
- `/tmp/bench_rss_after_v2.log`

### 1M docs (`--no-finalize`)

| Metric | Before | After | Delta |
|---|---:|---:|---:|
| Docs indexed | 995,271 | 999,305 | +4,034 |
| Elapsed | 1m58.6s | 1m26.5s | -27.1% |
| Avg rate | 8,434 docs/s | 11,555 docs/s | +37.0% |
| Bench peak RSS | 3,796 MB | 3,658 MB | -3.64% |
| `/usr/bin/time` max resident set | 3,933,536,256 B | 3,768,500,224 B | -4.20% |

Logs:

- `/tmp/bench_rss_before_1m.log`
- `/tmp/bench_rss_after_v2_1m.log`

## Interpretation

1. Memory improved consistently (both Go-side peak and OS max resident).
2. 500k throughput is neutral.
3. 1M throughput improved strongly because previous aggressive merge policy caused long merge stalls.
4. Full-run RSS is expected to remain non-trivial due to:
   - large live segment set,
   - mmap/store access footprint,
   - Go allocator retaining arenas,
   - merge still being doc-reindex-based (not postings-level streaming merge).

## What This Solves vs. What Remains

### Solved now

- Unbounded in-flight flush memory.
- Over-aggressive background merge fan-in and cadence.
- One-shot force-merge working-set explosion.

### Remaining hard problem

Dahlia merge path still rebuilds from full docs (`getDoc -> addDoc`) instead of merging postings/term dictionaries directly. That architecture is the main long-term lever for further RSS and CPU reduction at full corpus size.

## Next Steps (recommended)

1. Implement postings-level segment merge (no re-tokenization, no full doc round-trip).
2. Add runtime metrics split:
   - Go allocator memory (`runtime.MemStats`)
   - OS RSS (`/proc`/platform syscall)
3. Run full 5.03M after this patch and append exact before/after numbers to this spec.

## Changed Files

- `pkg/index/driver/flower/dahlia/doc.go`
- `pkg/index/driver/flower/dahlia/engine.go`
- `pkg/index/driver/flower/dahlia/merge.go`
- `pkg/index/driver/flower/dahlia/merge_test.go`
- `spec/0656_low_rss.md`
