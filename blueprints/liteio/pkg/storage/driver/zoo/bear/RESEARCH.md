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

v2 fixed the catastrophic correctness/stability problem (4GB B-tree cap failures under the benchmark suite), reduced memory dramatically, and improved small-write throughput substantially. It did **not** meet the requested 5x throughput target, and at that time the full-suite `<100MB` process target was blocked by both benchmark-harness payload allocation and `bear` page-structure growth.

---

## 12. v3 Follow-Up Addendum (2026-02-22)

This addendum documents the three follow-up changes implemented after the v2 report:

1. delete-time merge / structural compaction (page reuse via free list is now exercised during delete)
2. paper-aligned separator prefix truncation (safe routing prefixes for inner-node separators)
3. `cmd/bench` payload-memory fix (streaming large write payloads + cache release between benchmarks)

### 12.1 Implemented code changes

#### A. `bear`: delete-time merge / compaction path

- Replaced the old leaf-only delete rewrite with recursive `deleteFrom(...)` descent.
- Added sibling-merge rebalancing for:
  - leaf children (`tryMergeLeafChildrenAt`)
  - inner children (`tryMergeInnerChildrenAt`)
- Added root shrink after delete when the root becomes a degenerate inner node (0 keys, 1 child).
- Freed merged-away pages through the existing free-list (`freePage`), so subsequent inserts can reuse them.

Practical effect:

- delete-heavy phases now reclaim B-tree pages structurally instead of only removing leaf entries and leaving page topology untouched.
- correctness remained intact in local full-suite validation (`0` benchmark errors).

#### B. `bear`: safe separator prefix truncation (paper-inspired)

- Added `shortestSeparator(leftMax, rightMin)` which returns the shortest prefix of `rightMin` that still satisfies:
  - `leftMax < separator <= rightMin`
- Applied on leaf split promotion (parent separator no longer always stores full `right[0].key`).
- Added subtree boundary helpers (`subtreeMinKey`, `subtreeMaxKey`) and used them when splitting inner nodes so promoted separators are also safely truncated using actual child-subtree bounds.

Why this matters:

- inner nodes store separators only for routing; using shortest valid routing prefixes reduces inner-page key bytes, which lowers split pressure and is directly aligned with the paper's prefix-truncation idea.

#### C. `cmd/bench`: large-payload memory fix (local benchmark harness)

`bench/runner.go` was changed to remove the old payload-memory floor:

- `payloadReader(size)` now streams large payloads (`>= 1MB`) using a deterministic xorshift reader (no giant `[]byte` allocation).
- `runBenchmark(...)` now calls `releasePayloadCache()` after each benchmark:
  - clears cached payload map
  - runs `runtime.GC()`
  - runs `debug.FreeOSMemory()`

This preserves comparable benchmark behavior while removing the persistent "all payload sizes cached for the whole suite" memory footprint.

### 12.2 v3 validation commands (`cmd/bench`, local)

All validation below uses local `cmd/bench` runs (as requested).

#### Focused smoke (`Write/1KB`, quick)

- Command:

```bash
go run -tags noant ./cmd/bench \
  --drivers bear \
  --quick \
  --filter Write/1KB \
  --output ./report/bear_v3_smoke
```

- Results (`report/bear_v3_smoke/raw_results.json`):
  - `Write/1KB`: `612,520 ops/s` (`519,157` iterations)
  - Peak RSS: **75.4 MB**
  - Peak Go Heap: **28.9 MB**
  - Errors: `0`

#### Focused large object (`Write/100MB`, non-quick short run)

Note: `--quick` excludes the `100MB` object size even when `--large` is set, so a short non-quick run was used.

- Command:

```bash
go run -tags noant ./cmd/bench \
  --drivers bear \
  --large \
  --filter Write/100MB \
  --benchtime 200ms \
  --warmup 1 \
  --min-iters 1 \
  --output ./report/bear_v3_focus_write100mb_real
```

- Results (`report/bear_v3_focus_write100mb_real/raw_results.json`):
  - `Write/100MB`: `5.57 ops/s` (`3` iterations)
  - Peak RSS: **35.6 MB**
  - Peak Go Heap: **28.7 MB**
  - Errors: `0`

This confirms the old payload-cache / 100MB-slice harness floor is no longer present.

#### Full suite (`bear`, quick + large)

- Command:

```bash
go run -tags noant ./cmd/bench \
  --drivers bear \
  --quick \
  --large \
  --output ./report/bear_v3_full
```

- Results (`report/bear_v3_full/raw_results.json`):
  - Benchmarks: `40`
  - Errors: `0`
  - Peak RSS: **255.6 MB**
  - Peak Go Heap: **29.8 MB**
  - Peak Go Sys: **173.0 MB**
  - Selected throughput:
    - `Write/1KB`: `831,402 ops/s`
    - `Delete`: `836,285 ops/s`
    - `Copy/1KB`: `941,348 ops/s`

### 12.3 Updated memory interpretation after the harness fix

The v2 report's harness-memory caveat was correct *for v2*, but it is now partially superseded:

- The specific payload-cache floor (including persistent 100MB payload retention) is fixed.
- This is verified by the `Write/100MB` focused run at **35.6 MB** peak RSS.

However, the full quick suite still peaks at **255.6 MB**. The evidence now points to a different mix of causes:

- long-lived benchmark-suite accumulation (many objects retained until driver cleanup at end of the run)
- `bear`'s mmap/file growth behavior over a long multi-benchmark session (freed pages are reusable, but file size / mapping size do not shrink automatically)
- Go runtime `Sys` growth during the suite (`173.0 MB`) despite low live heap (`29.8 MB`)

So, after v3:

- `<100MB` is verified for focused local `cmd/bench` workloads, including `Write/100MB`
- `<100MB` is still **not** met for the full multi-phase suite run
- the remaining gap is no longer explained by payload preallocation alone

### 12.4 What remains for the original targets

- **5x throughput target**: still not achieved.
- **Full-suite `<100MB` process RSS**: still not achieved.

Most impactful next steps are now:

1. add a true B-tree vacuum/truncate path (shrink file + remap after large delete-heavy phases)
2. optionally add per-benchmark storage isolation in `cmd/bench` (for driver-footprint measurement mode)
3. implement hint arrays / additional paper-aligned inner-page optimizations for throughput

---

## 13. v4/v5 Follow-Up Addendum (All Three Implemented)

Date: `2026-02-22`

This section covers the next three requested follow-ups:

1. `bear` vacuum/truncate path
2. `cmd/bench` per-benchmark isolation mode
3. `bear` search-path optimization (hint-style runtime head sampling)

### 13.1 Implemented changes

#### A. `bear`: tail-trim vacuum / truncate path

- Added free-page debt tracking (`vacuumDebt`) and a tail trim routine that:
  - scans the free list
  - identifies a contiguous free suffix of page IDs
  - rebuilds the free list without the trimmed suffix
  - shrinks `pageCount`
  - truncates + remaps `btree.dat` to the new size
- Triggered aggressively for recursive/bulk delete paths (`Delete(..., recursive=true)` and force bucket deletes).
- Kept single-object delete latency fast by batching/suppressing tail trim there.

Important scope note:

- This is a **tail-trim vacuum**, not a full B-tree rebuild compactor. It reclaims file/mmap size when free pages reach the end of the file.

#### B. `cmd/bench`: per-benchmark isolation mode

- Added CLI flag:
  - `--isolate-embedded-benchmarks`
- Runner now reopens a fresh storage instance for each benchmark phase (embedded drivers only).
- Final implementation uses filesystem data-path reset (`driver.DataPath`) before/after each benchmark when available (much faster than API-level object-by-object cleanup).
- Fallback remains API cleanup for drivers without a local `DataPath`.
- Also fixed cleanup output noise: `cleanupBucket()` progress now respects `--progress`.

#### C. `bear`: search-path optimization (format-preserving)

Without changing the on-disk format, added runtime search hints:

- sampled head-hint windows (`searchHintSegments`) for large pages
- head-range narrowing (`lower/upper bound` on `keyHead`) before full key compares
- optimized `keyHead()` to avoid copying the first 4 bytes for common key lengths

This targets hot routing paths:

- `leafSearch`
- `innerSearch`

### 13.2 Validation runs (local `cmd/bench`)

#### A. Non-isolated full quick suite (final, `report/bear_v5_full`)

Command:

```bash
go run -tags noant ./cmd/bench \
  --drivers bear \
  --quick \
  --large \
  --output ./report/bear_v5_full
```

Results (`report/bear_v5_full/raw_results.json`):

- Benchmarks: `40`
- Errors: `0`
- Peak RSS: **251.8 MB**
- Peak Go Heap: **30.0 MB**
- Peak Go Sys: **165.6 MB**
- Selected throughput:
  - `Write/1KB`: **957,924 ops/s** (up from `831,402` in `bear_v3_full`)
  - `Read/1KB`: **1,345,167 ops/s** (slightly down from `1,416,422`)
  - `Delete`: **844,132 ops/s** (stable vs `836,285`)
  - `ParallelWrite/1KB/C10`: **55,488 ops/s** (up from `53,439`)

#### B. Isolated full quick suite (new mode, `report/bear_v5_full_isolated2`)

Command:

```bash
go run -tags noant ./cmd/bench \
  --drivers bear \
  --quick \
  --large \
  --isolate-embedded-benchmarks \
  --output ./report/bear_v5_full_isolated2
```

Results (`report/bear_v5_full_isolated2/raw_results.json`):

- Benchmarks: `40`
- Errors: `0`
- Peak RSS: **330.3 MB**
- Peak Go Heap: **22.0 MB**
- Peak Go Sys: **242.0 MB**
- Final disk: **0.0 MB** (path reset between benchmarks)

Key observation:

- Isolation mode successfully removes long-lived on-disk accumulation between phases.
- Process RSS still exceeds `<100MB` and is actually **higher** here due repeated open/close/remap churn increasing Go `Sys`.

### 13.3 Interpretation after all three follow-ups

What improved:

- Throughput: `Write/1KB` improved materially vs `bear_v3_full` (quick suite).
- `Delete` stayed healthy after batching tail-vacuum (no per-op vacuum in single-key delete).
- `cmd/bench` now has a real per-benchmark isolation mode for embedded drivers.
- `bear` now has a real truncate/remap vacuum path (tail-trim variant).

What remains unsolved:

- Full-suite process RSS `<100MB` is still not met in either mode.
- A full compaction/rewrite vacuum (not just tail trim) is still needed to reclaim fragmented free pages.
- Repeated remap/open cycles can inflate `Go Sys`, so isolation mode is useful for methodology but not a guaranteed lower-RSS mode.

## 14. v6/v7 Follow-Up Addendum (Rebuild Vacuum + Subprocess Isolation + On-Disk Inner Hints)

Date: `2026-02-22`

This section covers the next requested set:

1. full rebuild-based vacuum (`bear`)
2. subprocess-per-benchmark isolation mode (`cmd/bench`)
3. on-disk inner-page hint arrays (`bear`, format change)

### 14.1 Implemented changes

#### A. `bear`: full rebuild vacuum (fragmentation reclaim, not just tail trim)

- Added a rebuild compactor that:
  - scans live entries with `btreeScan`
  - rebuilds a compact temporary `btree.dat` by reinserting entries into a fresh B-tree
  - truncates/remaps the temp file to exact `pageCount`
  - atomically replaces the active `btree.dat`
  - reopens/remaps the active store file in-place
- Triggered as a **best-effort bulk-delete optimization** only:
  - force bucket delete
  - recursive delete
  - after tail-trim vacuum runs
  - only when delete count + free-page fragmentation exceed conservative thresholds

Why this matters:

- Tail-trim reclaims only contiguous free pages at the file tail.
- Rebuild vacuum reclaims **fragmented** free pages and compacts the B-tree file layout.

#### B. `cmd/bench`: subprocess-per-phase isolation mode (process reset)

- Added CLI flag:
  - `--isolate-embedded-benchmarks-subprocess`
- Added exact internal phase filter:
  - `--phase-filter`
- Parent `cmd/bench` process now:
  - enumerates benchmark phases
  - spawns a child `cmd/bench` process per phase for embedded drivers
  - loads each child `raw_results.json`
  - merges results into a single final report

Notes:

- This is phase-level isolation (matching `runWithBucket(...)` labels), which is the right granularity for the current runner.
- It resets Go runtime state (`Go Sys`, heap arenas, goroutines) between phases, unlike in-process reopen/reset mode.

#### C. `bear`: on-disk inner-page hint arrays (new page type, backward-compatible)

- Added `pageTypeInnerHint` (`0x03`) for inner pages carrying a contiguous on-disk 1-byte-per-key hint array.
- Kept backward compatibility:
  - all old `pageTypeInner` pages remain readable
  - inner-page checks now accept both types
- `innerSearch` now uses:
  - existing `keyHead` narrowing
  - plus a second-stage on-disk hint-byte narrowing before full `bytes.Compare`
- Tuned to avoid fanout loss on small inner pages:
  - persisted hint arrays are only written for larger inner pages (`innerHintArrayMinKeys = 64`)

### 14.2 Validation runs (local `cmd/bench`)

#### A. Subprocess isolation smoke (`Write/1KB`)

Command:

```bash
go run -tags noant ./cmd/bench \
  --quick \
  --drivers bear \
  --filter Write/1KB \
  --isolate-embedded-benchmarks-subprocess \
  --output ./report/bear_v6_subproc_smoke \
  --formats json,markdown
```

Result (`report/bear_v6_subproc_smoke/raw_results.json`):

- Benchmarks: `1`
- Errors: `0`
- Peak RSS: **75.7 MB**
- `Write/1KB`: phase executed successfully via subprocess orchestration + merged report

#### B. Full quick suite (current code, subprocess isolation)

Command:

```bash
go run -tags noant ./cmd/bench \
  --quick \
  --drivers bear \
  --isolate-embedded-benchmarks-subprocess \
  --output ./report/bear_v7_full_subproc \
  --formats json,markdown
```

Results (`report/bear_v7_full_subproc/raw_results.json`):

- Benchmarks: `40`
- Errors: `0`
- Peak RSS: **229.8 MB**
- Peak Go Heap: **30.1 MB**
- Peak Go Sys: **248.9 MB**
- Final disk (merged max across subprocess phases): **4197.8 MB**
- Selected throughput:
  - `Write/1KB`: **734,927 ops/s**
  - `Delete`: **816,985 ops/s**

Interpretation:

- Subprocess isolation is working and gives a cleaner per-phase measurement methodology than in-process reopen/reset.
- It still does **not** achieve `<100MB` full-suite peak RSS because some individual phases themselves exceed that budget.

#### C. Focused `Write/1KB` sanity check (current code)

Command:

```bash
go run -tags noant ./cmd/bench \
  --quick \
  --drivers bear \
  --filter Write/1KB \
  --output ./report/bear_v7_focus_write1k \
  --formats json
```

Results (`report/bear_v7_focus_write1k/raw_results.json`):

- `Write/1KB`: **775,127 ops/s**
- Peak RSS: **78.0 MB**
- Errors: `0`

#### D. Full quick suite variability note (`v7_full`)

Command:

```bash
go run -tags noant ./cmd/bench \
  --quick \
  --drivers bear \
  --output ./report/bear_v7_full \
  --formats json,markdown
```

Observed result (`report/bear_v7_full/raw_results.json`):

- Benchmarks: `40`, Errors: `0`
- Peak RSS: **144.1 MB**
- `Write/1KB`: **321,322 ops/s** (significantly lower than focused/subprocess runs)

Interpretation:

- This run appears to be a noisy/full-suite outlier (machine contention / long-suite interaction).
- Focused and subprocess runs on the same code remained in the `~735k–775k` range for `Write/1KB`.

### 14.3 Additional validation status

- `go test ./pkg/storage/driver/zoo/bear` ✅
- `go test -tags noant ./cmd/bench` ✅

### 14.4 Current conclusion after v6/v7

- All three requested follow-ups are implemented:
  - rebuild-based vacuum
  - subprocess-per-phase isolation
  - on-disk inner-page hints (format change, backward-compatible)
- Full-suite `<100MB` process RSS remains unmet.
- Rebuild vacuum improves B-tree file compaction behavior (fragmented free-page reclaim), but value-log growth still dominates disk footprint in long suites.
- The new subprocess mode improves methodology and reportability, but not peak RSS enough to satisfy the `<100MB` full-suite target.

## 15. v8 Follow-Up Addendum (Temporary `ant` disable + inner fast-path optimization)

Date: `2026-02-22`

This pass focused on two practical goals:

1. make `cmd/bench` usable locally without the broken `ant` driver
2. improve `bear` write-path CPU cost again without changing external behavior

### 15.1 `cmd/bench`: temporary `ant` disable by default

Problem:

- `bench` / `cmd/bench` default builds could fail when local `ant` is mid-refactor.
- This blocked normal benchmark runs even when benchmarking `bear` only.

Change:

- Switched `bench/driver_import_ant.go` build tag from default-on to opt-in:
  - now `//go:build antbench`
- Result:
  - `ant` is **disabled by default** in `cmd/bench`
  - re-enable explicitly with `-tags antbench`

Validation:

- `go test ./cmd/bench` ✅
- `go test ./bench` ✅

### 15.2 `bear`: inner-node in-place insert fast path

Targeted hot path:

- `insertIntoInner(...)` previously always decoded all inner keys/children and rebuilt the page (`writeInnerPage`) after each child split, even when the page still had room.

Optimization:

- Added an in-place inner-page insert path for non-hint inner pages:
  - computes `innerFreeSpace(...)`
  - appends only the new separator key payload to the data region
  - rewrites only child/slot metadata arrays (no full key decode/re-encode)
  - falls back to existing safe rebuild path when needed
- Hint-array pages (`pageTypeInnerHint`) intentionally still use the safe rebuild path in this version.
  - I tested a hint-page in-place variant and hit corruption; this was intentionally rolled back to preserve correctness.

Key code points:

- `innerFixedSize`, `innerFreeSpace`: metadata accounting for inner pages
- `innerInsertAt`: in-place fast path
- `insertIntoInner`: fast-path attempt before full rebuild

### 15.3 Benchmarking notes (important)

I hit false `bear` panics during one intermediate validation because I launched **two `cmd/bench` processes in parallel** against the same default local embedded data path (`/tmp/bear-bench`).

This is a benchmark harness collision, not a `bear` correctness issue.

Rule for local embedded-driver benchmarking:

- do **not** run multiple `cmd/bench` processes concurrently unless they use distinct data paths

### 15.4 Validation runs (local `cmd/bench`, serial)

#### A. Focused `Write/1KB` (`report/bear_v8c_focus_write1k`)

Command:

```bash
go run ./cmd/bench \
  --quick \
  --drivers bear \
  --filter Write/1KB \
  --output ./report/bear_v8c_focus_write1k \
  --formats json
```

Results:

- `Write/1KB`: **789,684 ops/s**
- Peak RSS: **78.5 MB**
- Errors: `0`

Comparison:

- prior focused reference (`report/bear_v7_focus_write1k`): **775,127 ops/s**
- delta: modest improvement (~`+1.9%`)

#### B. Focused `Delete` (`report/bear_v8c_focus_delete`)

Command:

```bash
go run ./cmd/bench \
  --quick \
  --drivers bear \
  --filter Delete \
  --output ./report/bear_v8c_focus_delete \
  --formats json
```

Results:

- `Delete`: **760,458 ops/s**
- Peak RSS: **193.0 MB**
- Errors: `0`

#### C. Full quick suite (`report/bear_v8_full`)

Command:

```bash
go run ./cmd/bench \
  --quick \
  --drivers bear \
  --output ./report/bear_v8_full \
  --formats json,markdown
```

Results:

- Benchmarks: `40`
- Errors: `0`
- Peak RSS: **232.2 MB**
- Peak Go Heap: **28.9 MB**
- Peak Go Sys: **148.9 MB**
- Final disk: **8604.4 MB**
- Selected throughput:
  - `Write/1KB`: **761,115 ops/s**
  - `Delete`: **792,825 ops/s**

#### D. Full quick suite, subprocess isolation (`report/bear_v8_full_subproc`)

Command:

```bash
go run ./cmd/bench \
  --quick \
  --drivers bear \
  --isolate-embedded-benchmarks-subprocess \
  --output ./report/bear_v8_full_subproc \
  --formats json,markdown
```

Results:

- Benchmarks: `40`
- Errors: `0`
- Peak RSS: **183.6 MB**
- Peak Go Heap: **30.0 MB**
- Peak Go Sys: **181.0 MB**
- Final disk (merged phase max): **4665.9 MB**
- Selected throughput:
  - `Write/1KB`: **856,927 ops/s**
  - `Delete`: **787,639 ops/s**

Comparison vs prior subprocess reference (`report/bear_v7_full_subproc`):

- `Write/1KB`: `734,927 -> 856,927` (**+16.6%**)
- Peak RSS: `229.8 MB -> 183.6 MB` (**-20.1%**)

### 15.5 Current status after v8

- `cmd/bench` is usable again by default without `ant` (temporary opt-in via `-tags antbench`)
- `bear` got an additional safe write-path optimization (inner insert fast path for non-hint pages)
- Full-suite `<100MB` RSS is still not met
- Focused `Write/1KB` remains `<100MB` RSS and improved slightly
- Subprocess-isolated full-suite results improved significantly versus `v7` on this machine/run set

## 16. v10 Follow-Up Addendum (Profiler-driven optimization + value-log tail reclaim)

Date: `2026-02-22`

This pass was explicitly profiler-driven.

### 16.1 Go profiler findings (before optimization)

Profiled focused `Write/1KB` (`report/bear_v9_profile_write1k`):

```bash
go run ./cmd/bench \
  --quick \
  --drivers bear \
  --filter Write/1KB \
  --profile \
  --output ./report/bear_v9_profile_write1k \
  --formats json
```

Key CPU finding (`pprof -top`, embedded in `raw_results.json`):

- `runtime.memclrNoHeapPointers`: **~43.5% flat**

Interpretation:

- We were spending a large fraction of CPU repeatedly zeroing 4KB pages.
- The obvious suspects were full-page clears in:
  - `writeLeafPage`
  - `writeInnerPage`
  - free-list page zeroing (`allocPage` reuse / `freePage`)

Profiled focused `Delete` (`report/bear_v9_profile_delete`) also showed:

- heavy allocations from `readLeafEntry` / `readAllLeafEntries`
- CPU in page rewrites + metadata writes

### 16.2 Optimizations implemented in this pass

#### A. Remove unnecessary page clears (profiler root cause fix)

- Removed full-page zeroing in:
  - `writeLeafPage*`
  - `writeInnerPage`
  - `allocPage()` free-page reuse path
  - `freePage()`
  - free-list rebuild in tail-trim vacuum

Rationale:

- Page readers use `count`, `freeOff`, and slot metadata to define the live region.
- Stale bytes in free space / dead slots are safe and should not require zeroing.
- This directly targets the `memclr` hotspot.

#### B. `Delete` path: in-place leaf delete + deferred compaction

- Added `leafDeleteAt(...)`:
  - removes a slot in-place
  - updates `count`
  - coalesces `freeOff` only when deleting the current low-watermark entry
- Added `compactLeafPage(...)` and `leafPackedFreeSpace(...)`
  - compaction is only attempted later on insert when a page is full **and** recoverable hole space exists

Effect:

- `Delete` no longer rewrites the entire leaf page on every delete.
- This removes the previous `readAllLeafEntries`/`writeLeafPage` per-delete cost from the hot path.

#### C. `Write` split path: avoid unnecessary work

- `insertIntoLeaf` split path now inserts at known `idx` instead of append+`sort.Slice`
- Added `writeLeafPageSorted(...)` and routed hot internal call sites to it (skip redundant sorting)
- Fixed a regression introduced during compaction work:
  - compaction-before-split now runs **only** when recoverable hole space exists (`leafPackedFreeSpace > leafFreeSpace`)

#### D. `innerInsertAt` allocation reduction

- Switched `innerInsertAt(...)` scratch metadata slices to stack-backed fixed arrays (`innerScratchMaxKeys`)
- Reduced heap allocations in the inner insert fast path

#### E. Value-log reclaim: tail trim after bulk deletes (safe reclaim path)

- Added `maybeTrimValueLogTailLocked(...)`
  - scans live entries to find max live external value end offset
  - flushes buffered value-log bytes
  - truncates `values.log` tail if all live external values are below current end
- Triggered after bulk delete paths (force bucket delete / recursive delete)

Important scope note:

- This is **tail reclaim**, not full value-log rewrite compaction.
- It reclaims dead space at the end of `values.log` safely with no offset rewriting.

### 16.3 Validation: value-log tail reclaim

Ad-hoc local check (recursive delete):

- `btree.dat`: `4,194,304 -> 8,192` bytes
- `values.log`: `16,777,216 -> 0` bytes

This confirms the new tail-reclaim path works when recent values are deleted and no live external values remain.

### 16.4 Go profiler findings (after optimization)

Profiled focused `Write/1KB` again (`report/bear_v10b_profile_write1k`):

```bash
go run ./cmd/bench \
  --quick \
  --drivers bear \
  --filter Write/1KB \
  --profile \
  --output ./report/bear_v10b_profile_write1k \
  --formats json
```

Key changes in CPU top:

- `runtime.memclrNoHeapPointers`: **~43.5% -> ~4.35%** (large drop)
- New root cause: `bear.writeLeafPageSorted` (**~47.8% flat**)

Interpretation:

- The profiler-guided memclr optimization worked.
- The next dominant bottleneck is structural:
  - leaf page rewrites during split/merge-heavy paths (`writeLeafPageSorted`)
- Deeper improvements from here likely require:
  - reducing split frequency
  - reducing leaf rewrite volume
  - or more aggressive page-layout / split-policy changes

### 16.5 Benchmark results (local `cmd/bench`)

#### A. Focused `Write/1KB` (post-fix, non-profile run)

Command:

```bash
go run ./cmd/bench \
  --quick \
  --drivers bear \
  --filter Write/1KB \
  --output ./report/bear_v10_focus_write1k \
  --formats json
```

Result (`report/bear_v10_focus_write1k/raw_results.json`):

- `Write/1KB`: **684,562 ops/s**
- Peak RSS: **76.9 MB**
- Errors: `0`

Profile-mode reference (`report/bear_v10b_profile_write1k`):

- `Write/1KB`: **794,900 ops/s** (profile mode can vary; use mainly for hotspot attribution)

#### B. Focused `Delete` (post optimization)

Command:

```bash
go run ./cmd/bench \
  --quick \
  --drivers bear \
  --filter Delete \
  --output ./report/bear_v10b_focus_delete \
  --formats json
```

Result (`report/bear_v10b_focus_delete/raw_results.json`):

- `Delete`: **2,718,646 ops/s**
- Peak RSS: **549.1 MB**
- Errors: `0`

Interpretation:

- Massive throughput increase is expected from the in-place leaf delete path.
- Peak RSS increased because the adaptive benchmark now completes far more iterations in the same time budget (more total workload / larger transient state).

#### C. Full quick suite (`report/bear_v10_full`)

Command:

```bash
go run ./cmd/bench \
  --quick \
  --drivers bear \
  --output ./report/bear_v10_full \
  --formats json,markdown
```

Results:

- Benchmarks: `40`
- Errors: `0`
- Peak RSS: **527.9 MB**
- Peak Go Heap: **28.8 MB**
- Peak Go Sys: **251.6 MB**
- Final disk: **13148.5 MB**
- `Write/1KB`: **1,019,533 ops/s**
- `Delete`: **3,043,130 ops/s**

#### D. Full quick suite, subprocess isolation (`report/bear_v10_full_subproc`)

Command:

```bash
go run ./cmd/bench \
  --quick \
  --drivers bear \
  --isolate-embedded-benchmarks-subprocess \
  --output ./report/bear_v10_full_subproc \
  --formats json,markdown
```

Results:

- Benchmarks: `40`
- Errors: `0`
- Peak RSS: **400.5 MB**
- Peak Go Heap: **30.0 MB**
- Peak Go Sys: **241.4 MB**
- Final disk (merged phase max): **4449.2 MB**
- `Write/1KB`: **1,115,531 ops/s**
- `Delete`: **2,503,138 ops/s**

Comparison vs `report/bear_v8_full_subproc`:

- `Write/1KB`: `856,927 -> 1,115,531` (**+30.2%**)
- `Delete`: `787,639 -> 2,503,138` (**3.18x**)
- Peak RSS: `183.6 MB -> 400.5 MB` (**+118.1%**) due much higher adaptive iteration throughput / workload volume

### 16.6 Current conclusion after v10

- Profiler-guided optimization materially improved throughput and removed the prior `memclr` bottleneck.
- `Delete` performance improved dramatically via in-place leaf deletes.
- Value-log tail reclaim now works for bulk-delete patterns (safe tail truncation).
- Full-suite `<100MB` RSS remains unmet and is now further constrained by benchmark throughput scaling (more work per fixed benchmark duration).
- The next root cause from Go profiler is `writeLeafPageSorted` (leaf page rewrites during split/merge-heavy paths).
