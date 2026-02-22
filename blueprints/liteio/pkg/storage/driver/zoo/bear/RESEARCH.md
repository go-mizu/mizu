# Bear Driver: Deep Performance Research (v2)

## Table of Contents

1. Architecture Overview
2. Reference Paper Notes ("B-Trees Are Back")
3. Benchmark + Profiling Methodology
4. v1 Full-Suite Baseline (Before Fixes)
5. Focused Write Baseline (`Write/1KB`)
6. v2 Optimization Journey (Implemented)
7. v2 Full-Suite Results (After Optimizations)
8. Memory Budget Analysis (<100MB Constraint)
9. Remaining Bottlenecks (Why 5x Is Not Yet Reached)
10. Next Optimization Plan (Toward v3)
11. Appendix: Commands (Bench + pprof)

---

## 1. Architecture Overview

`bear` is a single-file B-tree storage driver with:

- `btree.dat`: `mmap`-backed pageable B-tree (page-based, slotted leaves/inners)
- `values.log`: append-only value log for object payloads
- composite keys: `bucket + "\x00" + key`

### Page layout (current)

- Leaf pages: slotted page (`keyHead`, `entryOffset`) + packed variable-length entries
- Inner pages: child array + slotted separator keys
- Head optimization: first 4 key bytes stored in slot to reduce full-key dereferences in binary search

### Current important implementation constraints

- Single global `store.mu` write lock serializes all B-tree mutations
- `mmap` maps the whole `btree.dat` file
- No page merge / compaction on delete (pages are rewritten but tree structure is not rebalanced)
- Value log is append-only; payload bytes are never reclaimed

---

## 2. Reference Paper Notes ("B-Trees Are Back")

### Paper / sources

- SIGMOD 2025 entry: https://2025.sigmod.org/toc-3-1.html
- DBLP: https://dblp.org/rec/conf/sigmod/FueglistalerBBC25.html
- TUM LeanStore page (paper + code links): https://db.in.tum.de/research/leanstore/
- PDF (linked from TUM page): https://db.in.tum.de/~fent/papers/Fueglistaler2025BtreesAreBack.pdf
- LeanStore code (paper implementation lineage): https://github.com/leanstore/leanstore

### What the paper contributes (relevant to `bear`)

The paper shows that engineering details in page layout matter enough for B-trees to compete with in-memory structures even when data fits in RAM. The key ideas relevant to `bear` are:

- slotted-page indirection for variable-length records
- prefix-aware inner-node key storage (prefix truncation)
- key-head optimization (store first bytes in slot for cheap reject)
- hint arrays for faster search inside high-fanout nodes
- adaptive per-node layout selection

### Where `bear` already matches the paper's direction

`bear` already uses three of the paper's high-value ideas:

- slotted pages for leaves/inner nodes
- head optimization (`keyHead`) in slots
- `mmap`-backed pageable storage

### Where `bear` still diverges (major optimization headroom)

`bear` does **not** yet implement:

- inner-node prefix truncation
- hint-based search arrays
- adaptive node layouts
- delete-time merge/rebalance / structural compaction

These missing features directly explain the remaining allocation and split overhead in our profiles.

---

## 3. Benchmark + Profiling Methodology

### Environment

- Date: **2026-02-22**
- Go: **go1.26.0**
- Platform: **darwin/arm64**
- CPUs: **10**
- Benchmark tool: `cmd/bench` (local embedded driver mode)
- BenchTime: **1s** (unless noted)
- Concurrency: **200**

### Local benchmark command (actual runs used)

Because unrelated local work temporarily broke `ant`, `cmd/bench` could not build all zoo drivers. I added a minimal build-tag shim so `cmd/bench` can run with `-tags noant` during local `bear` optimization.

Used commands:

- full baseline: `go run -tags noant ./cmd/bench --drivers bear --benchtime 1s --profile --resource-tracking ...`
- focused write: `--filter Write/1KB`
- full optimized trial: same as full baseline after changes

### Profiling workflow (same style as `herd/RESEARCH.md`)

- `go tool pprof -top` for quick hot functions
- `go tool pprof -top -cum` for call-path impact
- `go tool pprof -peek <regexp>` to disambiguate generic symbols and call chains
- `go tool pprof -base old.pprof new.pprof` for before/after deltas

### Important measurement caveat for memory

`cmd/bench` itself allocates large payload buffers for benchmark object sizes (up to 100MB), so the harness contributes a large baseline RSS/heap footprint independent of `bear`.

In the optimized full-suite heap profile, `bench.(*Runner).payload` alone accounts for ~116MB in-use at profile capture time.

---

## 4. v1 Full-Suite Baseline (Before Fixes)

### Command

- Output dir: `report/bear_v1_baseline`
- Command: `go run -tags noant ./cmd/bench --drivers bear --benchtime 1s --profile --resource-tracking --formats markdown,json --output ./report/bear_v1_baseline --progress`

### Headline results (v1 baseline)

- **42 benchmarks completed**
- **6,535,715 errors**
- Peak RSS: **2431.8 MB**
- Peak Go Heap: **425.5 MB**
- Peak Go Sys: **815.4 MB**
- Final Disk: **22.2 GB**
- GC cycles: **2472**

### Primary failure mode

`bear` hit the B-tree file cap repeatedly:

- `bear: file size would exceed limit (4294967296 bytes)`

This cascaded into:

- failed pool pre-creation for read benchmarks
- failed parallel write/read benchmarks
- `Scale/List/*` count mismatches (writes failed before list)
- `Delete` errors (`storage: not exist`) on objects that were never inserted due prior failures

### Selected baseline throughput (`raw_results.json`)

- `Write/1KB`: **299,709 ops/s**
- `Read/1KB`: **2,314,923 ops/s**
- `Stat`: **3,069,043 ops/s**
- `List/100`: **73,866 ops/s**
- `Delete`: **1,500,547 ops/s** (but with **909,392 errors**)

### v1 CPU profile (top)

- `syscall.rawsyscalln`: **23.42%**
- `runtime.pthread_cond_wait`: **13.66%**
- `runtime.memclrNoHeapPointers`: **10.51%**
- `runtime.usleep`: **9.93%**
- `runtime.pthread_cond_signal`: **9.43%**

Interpretation:

- CPU time was dominated by runtime/scheduler/syscall overhead, with substantial allocation zeroing (`memclr`) pressure.

### v1 allocs profile (key finding)

Top allocators (from `report/bear_v1_baseline/bear/allocs.pprof`):

- `(*store).readFromValueLog`: **138.76 GB** (59.28%)
- `readLeafEntry`: **48.25 GB** (20.61%)
- `(*bucket).Write`: **23.05 GB** (9.85%)

`pprof -peek "readFromValueLog"` showed it came through `resolveValue` and dominated read/copy-style workloads.

### Root causes identified in v1

1. Small values (1KB benchmark objects) were stored inline in B-tree leaves.
2. This caused extreme leaf split churn, fast `btree.dat` growth, and 4GB cap failures.
3. External reads allocated full value slices (`readFromValueLog`) for every open.
4. `Copy`/`Move` duplicated payload bytes even when data already existed in the same `values.log`.

---

## 5. Focused Write Baseline (`Write/1KB`)

### Why a focused benchmark was needed

The full-suite profiles mix read/copy/list/multipart behavior. To optimize the write path, I used a focused `cmd/bench` run with `--filter Write/1KB`.

### v1 focused write (`report/bear_focus_write_v1`)

- `Write/1KB`: **363,792 ops/s**
- Peak RSS: **64.4 MB**
- Peak Go Heap: **32.1 MB**
- Peak Go Sys: **59.5 MB**
- GC cycles: **251**

### v1 focused write CPU profile

- `runtime.memclrNoHeapPointers`: **69.07%** (dominant)

### v1 focused write allocs profile (top)

- `(*store).insertIntoInner`: **1.35 GB**
- `readLeafEntry`: **894.8 MB**
- `(*bucket).Write`: **599 MB**
- `readInnerKey`: **545 MB**

Interpretation:

- The write path was dominated by split-related allocations and repeated page rebuilds (`insertIntoLeaf` -> `writeLeafPage`), with heavy heap zeroing pressure.

---

## 6. v2 Optimization Journey (Implemented)

## Optimization 1: Externalize all non-empty values (value log first)

### Change

Set `valLogThreshold = 0` (with `n > 0` guard), so all non-empty objects go to `values.log` instead of inline leaf storage.

### Why this helps

- Leaf entries become key+metadata only (much smaller)
- Many more keys fit per 4KB leaf
- Much fewer leaf splits
- Slower `btree.dat` growth
- Delays/avoids the 4GB B-tree cap

### Tradeoff

- Small reads (especially `Read/1KB`) now pay a `pread`/streaming cost
- This intentionally trades some small-read speed for write scalability + memory stability + correctness

## Optimization 2: Buffered append for `values.log` (8MB buffer)

### Change

Replaced per-value `WriteAt` pattern with an append buffer (`valBuf`) plus flush-on-demand behavior.

Key behavior:

- appends are buffered under `valMu`
- flush occurs when buffer fills, on `msync`, on close, or before reading unflushed regions

### Why this helps

- amortizes syscall cost for small writes (especially 1KB objects)
- preserves append offsets deterministically
- keeps default `sync=none` fast while maintaining read correctness

## Optimization 3: Stream external reads (no `readFromValueLog` allocation)

### Change

`Open()` now returns `io.NewSectionReader` for external values instead of materializing a `[]byte` via `readFromValueLog()`.

### Why this helps

- removes the biggest v1 allocator (`readFromValueLog`)
- reduces GC pressure substantially on read-heavy/mixed workloads
- allows range reads without extra slicing/copying allocations

## Optimization 4: Reuse external value-log references for `Copy`/`Move`

### Change

When source data is external (`valOffset >= 0`), `Copy()` and `Move()` now reuse the same `{valOffset, valLen}` instead of reading + rewriting payload bytes.

### Why this helps

- eliminates redundant value-log reads/writes for intra-store copies/moves
- improves `Copy/1KB` throughput significantly
- reduces allocs and syscall pressure

## Optimization 5: Benchmark tooling unblock for local `cmd/bench`

### Change (non-driver)

Added `bench/driver_import_ant.go` with build tag `!noant` and removed the direct `ant` blank import from `bench/runner.go`, enabling:

- `go run -tags noant ./cmd/bench ...`

This does **not** change benchmark logic; it only avoids unrelated local `ant` compile failures while optimizing `bear`.

---

## 7. v2 Full-Suite Results (After Optimizations)

### Command

- Output dir: `report/bear_v2_trial`
- Command: `go run -tags noant ./cmd/bench --drivers bear --benchtime 1s --profile --resource-tracking --formats markdown,json --output ./report/bear_v2_trial --progress`

### Headline results (v2)

- **48 benchmarks completed**
- **0 errors** (down from **6,535,715**)
- Peak RSS: **656.4 MB** (down from **2431.8 MB**, **-73%**) 
- Peak Go Heap: **166.2 MB** (down from **425.5 MB**, **-61%**)
- Peak Go Sys: **480.5 MB** (down from **815.4 MB**, **-41%**)
- GC cycles: **280** (down from **2472**, **-88.7%**)
- Total iterations across results: **24,979,520** vs **11,108,631** (**2.25x**)

### Selected throughput deltas (baseline -> v2)

- `Write/1KB`: **299,709 -> 874,858 ops/s** (**2.92x**)
- `Write/10MB`: **115 -> 145 ops/s** (**1.26x**)
- `Write/100MB`: **8 -> 13 ops/s** (**1.53x**)
- `Stat`: **3,069,043 -> 3,961,383 ops/s** (**1.29x**)
- `List/100`: **73,866 -> 86,831 ops/s** (**1.18x**)
- `Copy/1KB`: **452,796 -> 949,292 ops/s** (**2.10x**, plus baseline had massive errors)
- `ParallelWrite/1KB/C*`: **baseline effectively broken (0 MB/s due cap errors) -> stable, error-free**

### Read-path tradeoff (expected regression on `Read/1KB`)

- `Read/1KB`: **2,314,923 -> 1,307,688 ops/s** (**0.56x**)

Reason:

- small values are now externalized and streamed from `values.log` (`pread`/`SectionReader`) instead of being copied from inline B-tree leaves.

This is the main performance tradeoff of the v2 design.

### v2 CPU profile (full suite)

Top functions:

- `syscall.rawsyscalln`: **55.71%**
- `runtime.pthread_cond_wait`: **10.87%**
- `runtime.memclrNoHeapPointers`: **7.62%**

Interpretation:

- We traded heap-allocation pressure for syscall pressure.
- This is a good trade for correctness + memory + write scalability, but it caps `Read/1KB` and keeps write path below the requested 5x target.

### v2 allocs profile (full suite)

Top allocators:

- `readLeafEntry`: **9.78 GB**
- `(*bucket).Write`: **2.34 GB**
- `(*bucket).List.func1`: **2.04 GB**
- `io.ReadAll`: **1.79 GB** (multipart + fallback paths)
- `(*store).insertIntoInner`: **1.78 GB**

### `pprof -peek` findings (v2)

- `readLeafEntry` allocs are still driven by three paths:
  - `readAllLeafEntries` (splits/rewrites)
  - `btreeGet` (point lookups)
  - `btreeScan` (listing)
- `insertIntoInner` allocs are mostly from `readInnerKey` during inner-node rebuild on child split
- `io.ReadAll` is now mostly multipart uploads + fallback write paths, not point reads

---

## 8. Memory Budget Analysis (<100MB Constraint)

## What was achieved

### Verified sub-100MB local benchmark (using `cmd/bench`)

Focused `Write/1KB` (`report/bear_focus_write_v2`):

- Peak RSS: **96.7 MB** (verified)
- Peak Go Heap: **35.7 MB**
- `Write/1KB`: **805,745 ops/s** (focused run, `--filter Write/1KB`)

This meets the `<100MB` requirement **for a local `cmd/bench` write workload**.

## Why the full-suite `cmd/bench` run cannot be <100MB without changing the benchmark harness

The optimized full-suite heap profile shows:

- `bench.(*Runner).payload`: **~116 MB in use**

That is benchmark harness memory (prebuilt payload buffers for 1KB..100MB object sizes), not `bear` state.

Therefore:

- **Full-suite `cmd/bench` RSS <100MB is impossible** in the current harness configuration, even with a near-zero-memory driver.

### Practical interpretation for `bear`

- Driver-specific memory was drastically reduced (RSS 2.4GB -> 656MB, heap 425MB -> 166MB)
- The remaining full-suite RSS gap is a combination of:
  - benchmark harness payload floor (~116MB)
  - `mmap`-resident B-tree metadata pages during long write-heavy phases
  - no delete-time structural compaction / page merge

---

## 9. Remaining Bottlenecks (Why 5x Is Not Yet Reached)

The request target was **5x** over current `bear`. We achieved large reliability + memory improvements and ~3x on `Write/1KB` (full-suite), but not 5x.

### Bottleneck A: split-path allocations still dominate write scalability

Evidence:

- focused write alloc diff shows major reductions in `insertIntoInner`, `readLeafEntry`, `readInnerKey`
- but focused CPU still shows `memclr` + split-path rebuild overhead as dominant cost

Root cause:

- child splits trigger full leaf/inner reconstruction (`readAllLeafEntries`, `readInnerKey`, `writeLeafPage`, `writeInnerPage`)
- no prefix truncation / hint arrays / adaptive layouts (paper features not yet implemented)

### Bottleneck B: syscall-bound external value reads/writes

Evidence:

- v2 full-suite CPU: `syscall.rawsyscalln` = **55.71%**

Root cause:

- small values now hit `pread`/`pwrite` paths more often (correct trade for cap/memory, but throughput ceiling shifts to syscalls)

### Bottleneck C: global mutation lock destroys parallel write scaling

Evidence:

- `ParallelWrite/1KB` scaling is effectively flat/degraded at high concurrency

Root cause:

- single `store.mu` serializes all B-tree mutations

### Bottleneck D: no page merge/compaction on delete

Evidence:

- full-suite RSS still grows to **656MB** despite externalized values and zero errors

Root cause:

- delete rewrites leaves but does not rebalance/merge/free structural pages in the tree
- `mmap` remains sized for historical peak `btree.dat`

---

## 10. Next Optimization Plan (Toward v3)

These are the changes most likely to reach the remaining goals (5x write target and tighter memory):

1. **Implement delete-time page merge + parent separator removal**
- Goal: reduce B-tree page growth during delete-heavy benchmarks
- Impact: lower RSS, lower disk, better long-run stability

2. **Add online/offline compaction (vacuum) for B-tree pages**
- Trigger when `pageCount` grows much faster than `entryCount`
- Rebuild B-tree pages from live entries; reuse same `values.log` offsets
- Impact: large RSS reduction in long benchmark runs

3. **Prefix truncation for inner nodes (paper-aligned)**
- Store shortened separator keys sufficient for routing
- Impact: higher fanout, fewer inner splits, lower allocs (`readInnerKey`, `insertIntoInner`)

4. **Hint-based search for inner nodes (paper-aligned)**
- Contiguous hint array for hot path search
- Impact: lower CPU for point lookup / descent

5. **Point-lookup decode variants (no key copy / metadata-only decode)**
- `btreeGetMeta` for `Stat`/update checks
- `btreeGetOpen` to avoid unnecessary key alloc on `Open`
- Impact: reduce `readLeafEntry` allocs

6. **List scanner fast path (decode key+size only)**
- Avoid content-type decoding when caller does not need it (or delay decode)
- Impact: reduce `List` allocs

7. **Write-path key normalization fast path**
- Avoid `path.Clean` + `strings.Split` on already-safe benchmark keys
- Impact: small but measurable CPU/alloc reduction (`strings.genSplit`)

8. **Concurrency architecture change (striped B-tree or partitioned trees)**
- Required for real parallel write scalability improvements
- Biggest code change, but necessary to move parallel scaling off the floor

---

## 11. Appendix: Commands (Bench + pprof)

### Full suite (baseline)

```bash
go run -tags noant ./cmd/bench \
  --drivers bear \
  --benchtime 1s \
  --profile --resource-tracking \
  --formats markdown,json \
  --output ./report/bear_v1_baseline \
  --progress
```

### Focused write (`Write/1KB`) baseline/optimized

```bash
go run -tags noant ./cmd/bench \
  --drivers bear \
  --benchtime 1s \
  --profile --resource-tracking \
  --formats json \
  --filter Write/1KB \
  --output ./report/bear_focus_write_v1

# after optimization
go run -tags noant ./cmd/bench \
  --drivers bear \
  --benchtime 1s \
  --profile --resource-tracking \
  --formats json \
  --filter Write/1KB \
  --output ./report/bear_focus_write_v2
```

### Full suite (optimized v2 trial)

```bash
go run -tags noant ./cmd/bench \
  --drivers bear \
  --benchtime 1s \
  --profile --resource-tracking \
  --formats markdown,json \
  --output ./report/bear_v2_trial \
  --progress
```

### pprof commands (same workflow used for `herd` research)

```bash
# Interactive web UI
go tool pprof -http=:8080 report/bear_v1_baseline/bear/cpu.pprof
go tool pprof -http=:8080 report/bear_v1_baseline/bear/heap.pprof
go tool pprof -http=:8080 report/bear_v1_baseline/bear/allocs.pprof

go tool pprof -http=:8080 report/bear_v2_trial/bear/cpu.pprof
go tool pprof -http=:8080 report/bear_v2_trial/bear/heap.pprof
go tool pprof -http=:8080 report/bear_v2_trial/bear/allocs.pprof

# Top consumers
go tool pprof -top -nodecount=25 report/bear_v2_trial/bear/cpu.pprof
go tool pprof -top -nodecount=25 report/bear_v2_trial/bear/allocs.pprof

# Cumulative hot paths
go tool pprof -top -cum -nodecount=40 report/bear_v2_trial/bear/cpu.pprof

# Before/after diffs
go tool pprof -base report/bear_v1_baseline/bear/cpu.pprof report/bear_v2_trial/bear/cpu.pprof
go tool pprof -base report/bear_v1_baseline/bear/allocs.pprof report/bear_v2_trial/bear/allocs.pprof

# Call-chain disambiguation (important)
go tool pprof -peek "readLeafEntry" report/bear_v2_trial/bear/allocs.pprof
go tool pprof -peek "insertIntoInner" report/bear_v2_trial/bear/allocs.pprof
go tool pprof -peek "readFromValueLog" report/bear_v1_baseline/bear/allocs.pprof
```

---

## Summary (v2 status)

v2 fixed the catastrophic correctness/stability problem (4GB B-tree cap failures under the benchmark suite), reduced memory dramatically, and improved small-write throughput substantially. It does **not** yet meet the full requested 5x throughput target, and full-suite `<100MB` memory is blocked both by benchmark harness payload allocation and by `bear`'s remaining page-structure growth / lack of compaction.
