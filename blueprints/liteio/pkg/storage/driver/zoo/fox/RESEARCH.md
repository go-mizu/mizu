# Fox Driver: Deep Performance Research (Bf-Tree-Inspired, Object-Storage Adaptation)

## Table of Contents

1. Architecture Overview
2. Reference Paper Learnings (Bf-Tree, PVLDB 2024)
3. Fox v1 Baseline (Correctness + Profiling)
4. Root Causes Identified
5. Optimization Journey (v2 iterations)
6. Final v2 Results (Benchmark + Profiles)
7. Memory Verification (<100MB target)
8. Remaining Bottlenecks
9. Next Optimization Directions (v3)
10. v3 Focused Pass (Leaf Read Cache + Flush Scratch Reuse)
11. Appendix: Benchmark / pprof Commands
12. References

---

## Architecture Overview

`fox` is a Bf-Tree-inspired local storage driver for LiteIO.

### Core layout (current v2)

- `pages.dat`: fixed-size leaf pages (default `4KB`) containing sorted metadata records
- `values.dat`: append-only value log for spilled payloads (most non-trivial objects)
- in-memory B-tree inner nodes (`btreeNode`) for routing leaf pages
- mini-page pool (LRU) for per-leaf write buffering / merge batching

### Key v2 design choices (object-storage adaptation)

The paper targets KV records (small/medium values). LiteIO benchmark includes `1KB`, `64KB`, `1MB`, `10MB`, and multipart workloads. A literal inline-leaf implementation loses correctness/perf for large objects under `4KB` pages.

v2 therefore uses:

- metadata-in-leaf + value-pointer records (`values.dat`) for large values
- streaming writes into `values.dat` (avoid whole-object heap buffering)
- streaming `Open` via `io.SectionReader` for indirect values
- heap-aware mini-page pool budgeting (not just serialized entry bytes)

This keeps the B-tree/mini-page structure but makes it viable for object-storage workloads.

---

## Reference Paper Learnings (Bf-Tree, PVLDB 2024)

### Paper used

- **Bf-Tree: Cache-Optimized B-Trees for Modern Hardware** (PVLDB 2024)
- Primary source PDF used for this work:
  - <https://www.vldb.org/pvldb/vol17/p3442-yoon.pdf>
- Project page / implementation notes:
  - <https://github.com/XiangpengHao/bf-tree-docs>

### What the paper contributes (relevant to `fox`)

The paper's central idea is to combine:

- B-tree leaf page layout (sorted, good for point/range reads)
- **mini-pages** (small per-leaf in-memory buffers) to absorb writes and cache recent records
- a buffer manager that can flush only affected leafs rather than rewriting large LSM structures

Paper concepts that directly informed the `fox` v2 changes:

1. **Leaf-local write buffering matters more than global buffering**
- Our `miniPagePool` remains per-leaf and flushes by leaf.
- v2 fixed correctness first, then focused on making leaf flushes cheap and safe.

2. **Do not let hot in-memory buffers explode memory footprint**
- Paper mini-pages are small and bounded.
- `fox` v1/v2a used serialized-byte accounting, which badly underestimated Go heap usage.
- v2c+ switched pool budgeting to approximate heap cost per cached entry.

3. **Reads should avoid unnecessary decode/copy work**
- Paper aims to reduce cache misses and unnecessary movement.
- v2e added targeted page lookup parsing (`findPageEntry`) and pooled page buffers for point lookups.

4. **Hardware-aware layout must be adapted to workload shape**
- LiteIO benchmark includes large object payloads far beyond leaf page size.
- `fox` v2 externalizes most values to `values.dat`, keeping leaves metadata-dense and tree fanout high.

### Important mismatch vs paper (intentional)

The Bf-Tree paper is not an object-storage engine storing many `10MB` objects inline in `4KB` leaves. The biggest v2 lesson is that **paper-aligned mini-page buffering must be combined with an external value log** for this benchmark profile.

---

## Fox v1 Baseline (Correctness + Profiling)

### Baseline command (local, quick, fox-only)

Because local `ant` code in this workspace is currently broken, `cmd/bench` was run with the existing `noant` build tag:

```bash
go run -tags noant ./cmd/bench \
  --drivers fox \
  --quick \
  --profile \
  --docker-stats=false \
  --output ./report/fox_v1_baseline
```

### Baseline result summary (`report/fox_v1_baseline`)

- Benchmarks: `40`
- Errors: `3,311,157`
- Peak RSS: `278.9 MB`
- Peak Go Heap: `108.7 MB`
- Peak Go Sys: `305.9 MB`
- GC cycles: `1981`

### Baseline correctness failure (critical)

v1 dropped records for values larger than the `4KB` leaf page size.

Root cause:

- `flushMiniPage()` split path encoded left/right halves but ignored `encodePageEntries(...)=fits=false` for oversized entries.
- Single `64KB+` values could not fit into a leaf page, producing empty pages and `storage: not exist` on read/range-read.

This explains the baseline benchmark behavior:

- `Read/64KB`, `Read/1MB`, `Read/10MB`: `0` throughput with errors
- millions of `not exist` errors across read/range/copy/delete/mixed workloads

### Baseline profile highlights (v1)

From `report/fox_v1_baseline/report.md` and pprof:

#### CPU (v1)

- `syscall.rawsyscalln`: `48.21%` flat
- `(*store).readPage` cumulative dominated CPU (~`45%` cum)

Interpretation:

- repeated page reads + syscall overhead dominated the hot path
- large reads were failing early, so CPU profile mostly reflected metadata/page churn, not useful payload reads

#### Heap / allocs (v1)

Top allocators:

- `(*store).readPage`: `25.98 GB` alloc_space
- `decodePageEntries`: `13.73 GB`
- `(*store).put`: `13.23 GB`
- `(*bucket).Write`: `12.92 GB`
- `encodePageEntries`: `2.60 GB`

Top in-use heap entries:

- `(*btree).insertIntoParent`: `19.89 MB`
- `compositeKey`: `11 MB`

Interpretation:

- page reads/decodes allocated excessively
- tree growth was amplified by broken split behavior
- large values were the correctness cliff

---

## Root Causes Identified

### 1. Large values incompatible with 4KB leaf pages (correctness bug)

- v1 attempted to store all values inline in leaf pages.
- `64KB+` values could not be encoded into `4KB` pages.
- split fallback lost data for oversized entries.

### 2. Mini-page pool memory accounting underestimated real Go heap cost

v1/v2a pool accounting used serialized bytes only. In Go, actual cost per cached entry also includes:

- `pageEntry` struct + slice headers
- string headers + allocated key bytes
- map/list node overhead
- per-mini-page object overhead

Result: pool stayed "under budget" while process heap kept growing.

### 3. Large write path buffered full objects in heap

`bucket.Write()` originally read the entire object into `[]byte` before calling `put`.

For large objects this caused:

- large transient heap spikes
- very high total allocations
- high Go heap sys retention (`GoSys`) even after GC

### 4. Point lookups decoded whole leaf pages

`get()` used `decodePageEntries()` for every lookup, allocating all entry structs/strings/values in the leaf page even when only one key was needed.

This hurt:

- `Open`
- `Stat`
- `CopyPart`
- read-heavy mixed workloads

---

## Optimization Journey (v2 iterations)

## v2a (`report/fox_v2_optimized`): Correctness fixed, memory regressed

### Implemented

- Added `values.dat` value-log spill for large objects (pointer encoded in leaf entries)
- Added indirect value encoding marker (`0xFFFFFFFE`)
- Fixed split logic using size-aware chunking (`splitEntriesByPage`) instead of "split in half and hope"
- Added section-reader based `Open` for indirect values
- Added tests for large value round-trip and split preservation

### Result

- Errors: `0` (from `3,311,157`)
- Peak RSS: **`851.3 MB`** (worse)

Why memory got worse initially:

- `1KB` values still inline (too many leaf splits/tree growth)
- large writes still allocated full buffers in `bucket.Write`
- mini-page pool budget still based on serialized bytes

## v2b (`report/fox_v2b_lowmem`): Low-inline threshold + streaming write path

### Implemented

- Lowered inline threshold (spill `1KB` objects too)
- `bucket.Write` streams large values directly to `values.dat`
- avoids building full object `[]byte` for spill-path writes

### Result

- Errors: `0`
- Peak RSS: `625.7 MB` (improved from `851.3 MB`)
- `List/100`: improved sharply (metadata fanout increased because leaves store pointers)

## v2c (`report/fox_v2c_poolbudget`): Heap-aware mini-page pool budgeting

### Implemented

- Mini-page pool budget now tracks approximate heap cost (`poolCost`), not just serialized bytes
- separated:
  - `miniPage.size` (serialized bytes, flush threshold)
  - `miniPage.poolCost` (heap budget)
- more aggressive eviction under the same DSN `pool_size`

### Result

- Errors: `0`
- Peak RSS: `526.5 MB`

## v2d (`report/fox_v2d_mempush`): Pooled streaming buffer + aggressive pool cost

### Implemented

- pooled `appendValueFromReader` buffer via `sync.Pool`
- sharply reduced alloc churn from spill-path writes
- further increased per-entry heap-cost weighting for mini-page pool eviction

### Result

- Errors: `0`
- Best observed peak RSS in this series: `435.2 MB`
- Heap in-use profile dropped to ~`96.5 MB` total process (near target)

## v2e final (`report/fox_v2e_final`): Point-lookup parser + pooled page buffers

### Implemented

- `findPageEntry()` targeted on-page parser for point lookups (no full-page decode for one key)
- `getEntry(..., loadValue)` / `getMeta()` split for `Stat` metadata-only path
- pooled page buffers (`readPageInto`, `pageBufPool`) for lookup path

### Final code state

- Correctness fixed (`0` errors)
- Throughput materially improved on key workloads
- Total process heap-in-use at profile time reduced to ~`102.9 MB`
- Full-suite process **peak RSS still >100MB** (see memory verification section)

---

## Final v2 Results (Benchmark + Profiles)

### Final benchmark command (current code)

```bash
go run -tags noant ./cmd/bench \
  --drivers fox \
  --quick \
  --profile \
  --docker-stats=false \
  --output ./report/fox_v2e_final
```

### Reliability

- Baseline v1: `3,311,157` errors
- Final v2e: `0` errors

### Selected performance comparison (v1 baseline vs v2e final)

Values below are from `report/fox_v1_baseline/report.md` and `report/fox_v2e_final/report.md`.

| Metric | v1 Baseline | v2e Final | Change | Notes |
|---|---:|---:|---:|---|
| `Write/1KB` | `168.0 MB/s` | `334.8 MB/s` | `1.99x` | faster + correct |
| `Read/1KB` | `559.4 MB/s` | `878.1 MB/s` | `1.57x` | faster + correct |
| `Read/64KB` | `0.00 MB/s` | `5.5 GB/s` | N/A | baseline broken |
| `Read/1MB` | `0.00 MB/s` | `5.8 GB/s` | N/A | baseline broken |
| `Read/10MB` | `0.00 MB/s` | `7.4 GB/s` | N/A | baseline broken |
| `List/100` | `17/s` | `51/s` | `3.0x` | metadata-only leafs help |
| `Delete` | `376.6K/s` | `1.1M/s` | `~2.9x` | no read-miss fallout |
| `Copy/1KB` | `12.5 MB/s` | `232.5 MB/s` | `18.6x` | baseline had many errors |

### Resource summary (v1 baseline vs v2e final)

| Metric | v1 Baseline | v2e Final | Change |
|---|---:|---:|---:|
| Errors | `3,311,157` | `0` | fixed |
| Peak RSS | `278.9 MB` | `504.9 MB` | worse (process peak) |
| Peak Go Heap | `108.7 MB` | `219.5 MB` | higher |
| Peak Go Sys | `305.9 MB` | `507.8 MB` | higher |
| GC cycles | `1981` | `259` | much lower |
| Runtime heap in use (profile-time) | `77.5 MB` | `183.6 MB` | higher |
| Total allocations | `76,435.7 MB` | `16,263.2 MB` | **4.7x lower** |

Interpretation:

- v2 fixed correctness and drastically reduced total allocation churn.
- Peak RSS got worse because the benchmark now successfully executes large reads/writes and the process retains more heap/sys memory over the full run.
- GC cycles dropped sharply (`1981 -> 259`) because v2 removed huge avoidable allocation churn in the write path and point lookup path.

### Final profile highlights (v2e)

#### CPU (`report/fox_v2e_final/fox/cpu.pprof`)

- `syscall.rawsyscalln`: `68.31%` flat
- `(*store).readPageInto` remains a major cumulative path

Interpretation:

- `fox` is now correctness-safe and much more allocation-efficient, but largely syscall-bound.
- Next major gains require I/O reduction/caching or concurrency redesign, not just more heap tuning.

#### Heap in-use (`report/fox_v2e_final/fox/heap.pprof`)

Top in-use contributors (total profile heap ≈ `102.86 MB`):

- `(*store).putPreparedEntry`: `23.59 MB`
- `(*bucket).Write`: `22 MB`
- `bench.(*Runner).payload`: `16.16 MB` (benchmark harness)
- `compositeKey`: `14.50 MB`
- `(*btree).insertIntoParent`: `9.97 MB`

#### Allocs (`report/fox_v2e_final/fox/allocs.pprof`)

Top alloc-space contributors:

- `decodePageEntriesWithMode`: `4.18 GB`
- `(*store).readPage`: `3.01 GB`
- `mergeEntries`: `1.68 GB`
- `encodePageEntries`: `0.95 GB`
- `io.ReadAll`: `0.88 GB` (benchmark/client side + multipart paths)

This is much improved from earlier v2 runs (and dramatically lower than v1 total allocations), but page decode/merge paths are still the biggest remaining optimization targets.

---

## Memory Verification (<100MB target)

## Requested target

- "Keep total memory under 100MB (verify carefully)"

## Verified numbers (carefully separated)

### A) Full-suite `cmd/bench` process peak RSS (resource tracker)

From `report/fox_v2e_final/report.md`:

- **Peak RSS: `504.9 MB`** (FAIL vs `<100MB` target)

This is process-level peak resident memory across the entire benchmark run and includes:

- benchmark harness allocations (`bench.(*Runner).payload`, etc.)
- imported driver init caches (e.g. local/rabbit caches in the same process)
- transient heap growth retained by the Go runtime (`GoSys`) during the run

### B) Final heap profile in-use (process, profile-time snapshot)

From `report/fox_v2e_final/fox/heap.pprof`:

- **Total heap in-use profile: `102.86 MB`** (near target, slightly above)

This is a snapshot at profile capture time, not peak RSS.

### C) Estimated fox-retained heap (excluding obvious benchmark/import overhead)

Visible non-fox/harness allocations in the final heap profile include roughly:

- `bench.(*Runner).payload`: `16.16 MB`
- `local` driver init/cache entries: ~`9-10 MB`
- `rabbit` dir cache: ~`1 MB`

Subtracting those from `102.86 MB` implies **fox-retained heap is roughly in the `75-85 MB` range** at profile time.

### Conclusion on memory target

- **Full benchmark process peak RSS `<100MB` was not achieved**.
- **Fox driver retained heap at profile time is approximately within the requested envelope**, but that is not the same metric as peak RSS.

If the requirement is strictly **process peak RSS** under the current full `cmd/bench` harness, more intrusive changes are needed (or a benchmark harness mode that isolates driver memory from harness payload caches / imported driver init allocations).

---

## Remaining Bottlenecks

### 1. Syscall-bound read path

CPU profile remains dominated by `pread` syscalls.

Likely improvements:

- small leaf page cache (raw page cache or parsed metadata cache)
- read-through mini-page caching for hot point lookups
- batched fs I/O / larger sequential coalescing where safe

### 2. Leaf merge path still allocates heavily

`mergeEntries` + `encodePageEntries` + `decodePageEntriesWithMode` dominate alloc space.

Likely improvements:

- specialized merge for sorted slices without temporary map when mini-page entries are sorted/deduped
- entry scratch pools for flush path
- list/scan decode variants that avoid content-type decode when unused

### 3. Global store lock limits parallel write scaling

Parallel write scaling is still poor (expected from current coarse-grained locking).

Likely improvements:

- striped trees / partitioned leaf spaces
- per-leaf or per-shard locks for mini-page mutation
- lock-free read-only routing snapshots for inner nodes

### 4. Composite key allocation remains persistent cost

`compositeKey` still shows up in final in-use heap.

Likely improvements:

- per-bucket key namespace / separate bucket routing to avoid `bucket + "\\x00" + key` string creation everywhere
- key interning/prefix dedup for separator keys (careful with memory tradeoffs)

---

## Next Optimization Directions (v3)

1. **Read-through leaf cache (highest ROI likely)**
- Cache raw leaf pages or parsed metadata entries for hot leaves.
- Goal: cut `pread` syscall dominance and `readPage` allocations.

2. **Flush-path allocation reduction**
- Replace `mergeEntries` map+sort with sorted mini-page entries + linear merge.
- Add scratch reuse for page encode/decode during flush.

3. **More precise memory budgeting**
- Separate budgets for:
  - mini-page metadata cache
  - dirty write buffers
  - optional read cache
- Make memory budget explicit in DSN (e.g. `pool_heap_budget=`).

4. **Concurrency refactor**
- Partition by key hash into multiple B-tree instances or stripe locks.
- This is required for meaningful `ParallelWrite` improvement.

5. **Benchmark memory isolation mode (tooling)**
- If process `<100MB` is a hard requirement for comparison, add a `cmd/bench` mode that:
  - disables payload cache reuse, or
  - runs one benchmark case per subprocess, or
  - imports only the target driver

---

## v3 Focused Pass (Leaf Read Cache + Flush Scratch Reuse)

After the v2e pass, the main remaining issue was the large gap between:

- process peak RSS (`~505MB`) and
- profile-time retained heap (much lower)

This suggested a large part of the problem was **allocation churn / heap growth retention**, not only retained state.

### v3 hypothesis

A focused pass on:

1. **bounded leaf read cache** (to cut repeated `pread` and page-buffer allocations)
2. **flush-path scratch reuse** (pooled page buffers + reusable encode buffer)
3. **lower-allocation flush merge** (sorted merge, no map+sort)

should reduce allocation churn and improve read-heavy throughput, while potentially lowering peak RSS.

### v3 implemented changes

#### 1. Bounded raw leaf page cache (LRU)

- Added `leafPageCache` (bounded LRU, default `4MB`)
- Raw page bytes keyed by `leafID`
- Integrated with `readPageInto()` and `writePage()` for cache consistency

#### 2. Flush path scratch reuse

- `flushMiniPage()` now uses pooled page buffers (`pageBufPool`) for page read + encode writeback
- Added `encodePageEntriesInto()` to reuse the same 4KB page buffer across flush/split writes
- Removed per-flush page buffer allocations in this path

#### 3. Lower-allocation merge on flush

- Replaced `mergeEntries` map+sort implementation with:
  - in-place sort of mini-page entries
  - linear merge with sorted disk entries
- Eliminates large temporary map allocations during flush

#### 4. v3 cache polish (critical)

The first v3 attempt (`fox_v3_focused`) showed `leafPageCache.set` as a top allocator because the cache was being populated by:

- point lookups (good)
- **flush reads / list scans / rebuild scans** (bad, cache pollution)

v3b fixed this by:

- adding `readPageIntoNoCache()` for non-point-read paths
- keeping cache usage for point lookups only
- pooling leaf-cache page buffers to reduce cache insert churn

### Benchmark execution note (local workspace issue)

Local `cmd/bench/main.go` is currently broken by unrelated in-progress edits in this workspace, so v3 was run via a tiny temporary wrapper that calls `bench.NewRunner` directly (same quick/profile config, fox-only).

This preserves the same benchmark engine (`bench` package) and output artifacts.

### v3 benchmark commands

```bash
# v3 first pass (leaf cache + flush scratch reuse)
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v3_focused

# v3b polished pass (cache only on point lookups + cache page pooling)
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v3b_final
```

### v3a vs v3b results summary

| Metric | v3a (`fox_v3_focused`) | v3b (`fox_v3b_final`) | Delta |
|---|---:|---:|---:|
| Errors | `0` | `0` | stable |
| Peak RSS | `497.9 MB` | `438.8 MB` | **-11.9%** |
| Peak Go Heap | `203.9 MB` | `181.4 MB` | **-11.0%** |
| Peak Go Sys | `498.2 MB` | `435.4 MB` | **-12.6%** |
| Runtime `HeapSys` (profile analysis) | `477.7 MB` | `416.8 MB` | **-12.7%** |
| Total allocations | `12.58 GB` | `9.24 GB` | **-26.5%** |

### v2e final vs v3b final (current best)

| Metric | v2e Final | v3b Final | Change |
|---|---:|---:|---:|
| Errors | `0` | `0` | stable |
| Peak RSS | `504.9 MB` | `438.8 MB` | **-13.1%** |
| Peak Go Heap | `219.5 MB` | `181.4 MB` | **-17.4%** |
| Peak Go Sys | `507.8 MB` | `435.4 MB` | **-14.3%** |
| Runtime heap in use (profile analysis) | `183.6 MB` | `175.9 MB` | `-4.2%` |
| Total allocations | `16.26 GB` | `9.24 GB` | **-43.2%** |

### Selected throughput deltas (v2e -> v3b)

#### Improved

- `Write/1KB`: `334.8 MB/s -> 350.9 MB/s` (`+4.8%`)
- `Write/64KB`: `1.45 GB/s -> 1.65 GB/s` (`+13.8%`)
- `Read/1KB`: `878.1 MB/s -> 1.19 GB/s` (`+35.3%`)
- `Read/64KB`: `5.5 GB/s -> 13.2 GB/s` (`+2.4x`)
- `Read/1MB`: `5.8 GB/s -> 12.9 GB/s` (`+2.2x`)
- `Read/10MB`: `7.4 GB/s -> 9.4 GB/s` (`+27.7%`)
- `Stat`: `5.47M/s -> 5.68M/s` (`+3.8%`)
- `Delete`: `1.09M/s -> 1.31M/s` (`+19.9%`)

#### Regressed (needs follow-up)

- `Write/1MB`: `1.35 GB/s -> 447 MB/s`
- `Write/10MB`: `1.47 GB/s -> 454 MB/s`
- `Copy/1KB`: `232.5 MB/s -> 79.1 MB/s`
- `RangeRead/*256KB`: significant regression in v3b
- `List/100`: small regression (`51/s -> 46.7/s`)

Interpretation:

- v3b improved the memory profile and many read-heavy paths.
- Some write/copy/range-read regressions likely reflect interactions with the new cache policy and changed benchmark dynamics; they need targeted follow-up before calling v3 \"net better\" for all workloads.

### v3b profile highlights

#### Heap in-use (`report/fox_v3b_final/fox/heap.pprof`)

- Total heap in-use profile: **`92.33 MB`** (below 100MB)
- Key fox contributors:
  - `(*store).putPreparedEntry`: `31.62 MB`
  - `compositeKey`: `12.50 MB`
  - `(*btree).insertIntoParent`: `10.97 MB`
  - leaf page cache buffers (`newLeafPageCache` pool alloc path): `4.52 MB`

#### Allocs (`report/fox_v3b_final/fox/allocs.pprof`)

- Total alloc-space: **`9.01 GB`** (down from `16.01 GB` in v2e final)
- `leafPageCache.set` is no longer a top allocator after v3b cache-polish
- Remaining top allocator is still `decodePageEntriesWithMode` (`3.76 GB`)

### Updated memory verification (v3b)

Strict process-level metric (resource tracker):

- **Peak RSS = `438.8 MB`** (still FAIL vs `<100MB` target)

Profile-time heap metrics:

- `heap.pprof` total in-use: **`92.33 MB`** (PASS vs `<100MB`, but snapshot metric)
- `runtime.MemStats.HeapInUse` in profile analysis: `175.9 MB` (runtime page accounting)

This reinforces the earlier conclusion:

- we can get fox-retained/profiled heap near or below `100MB`,
- but full-process peak RSS under the current benchmark harness remains well above `100MB`.

### v3 conclusion

The focused v3 pass succeeded at what it targeted:

- lower allocation churn (especially after v3b cache-polish)
- lower peak RSS / GoSys / GoHeap vs v2e
- faster point/read-heavy paths

It did **not** solve the strict `<100MB process peak RSS` requirement, and it introduced notable regressions in some write/copy/range-read paths that should be addressed in the next pass.

---

## Appendix: Benchmark / pprof Commands

### Benchmark commands used

```bash
# v1 baseline (broken correctness)
go run -tags noant ./cmd/bench --drivers fox --quick --profile --docker-stats=false --output ./report/fox_v1_baseline

# v2 iteration snapshots
go run -tags noant ./cmd/bench --drivers fox --quick --profile --docker-stats=false --output ./report/fox_v2_optimized
go run -tags noant ./cmd/bench --drivers fox --quick --profile --docker-stats=false --output ./report/fox_v2b_lowmem
go run -tags noant ./cmd/bench --drivers fox --quick --profile --docker-stats=false --output ./report/fox_v2c_poolbudget
go run -tags noant ./cmd/bench --drivers fox --quick --profile --docker-stats=false --output ./report/fox_v2d_mempush

# final
go run -tags noant ./cmd/bench --drivers fox --quick --profile --docker-stats=false --output ./report/fox_v2e_final

# v3 focused pass (wrapper, because local cmd/bench is temporarily broken in this workspace)
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v3_focused
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v3b_final
```

### pprof commands (same workflow style as herd)

```bash
# Open profiles

go tool pprof -http=:8080 report/fox_v1_baseline/fox/cpu.pprof
go tool pprof -http=:8080 report/fox_v1_baseline/fox/heap.pprof
go tool pprof -http=:8080 report/fox_v1_baseline/fox/allocs.pprof

go tool pprof -http=:8080 report/fox_v2e_final/fox/cpu.pprof
go tool pprof -http=:8080 report/fox_v2e_final/fox/heap.pprof
go tool pprof -http=:8080 report/fox_v2e_final/fox/allocs.pprof

go tool pprof -http=:8080 report/fox_v3b_final/fox/cpu.pprof
go tool pprof -http=:8080 report/fox_v3b_final/fox/heap.pprof
go tool pprof -http=:8080 report/fox_v3b_final/fox/allocs.pprof

# Text summaries (top / cumulative)

go tool pprof -top -nodecount=20 report/fox_v2e_final/fox/heap.pprof
go tool pprof -top -nodecount=20 report/fox_v2e_final/fox/allocs.pprof
go tool pprof -top -cum -nodecount=15 report/fox_v2e_final/fox/cpu.pprof

go tool pprof -top -nodecount=20 report/fox_v3b_final/fox/heap.pprof
go tool pprof -top -nodecount=20 report/fox_v3b_final/fox/allocs.pprof
go tool pprof -top -cum -nodecount=15 report/fox_v3b_final/fox/cpu.pprof

# Compare baseline vs final

go tool pprof -base report/fox_v1_baseline/fox/cpu.pprof report/fox_v2e_final/fox/cpu.pprof
go tool pprof -base report/fox_v1_baseline/fox/heap.pprof report/fox_v2e_final/fox/heap.pprof

go tool pprof -base report/fox_v1_baseline/fox/allocs.pprof report/fox_v2e_final/fox/allocs.pprof

# Attribution / call-chain inspection (useful for generic labels)

go tool pprof -peek "syscall.rawsyscalln" report/fox_v1_baseline/fox/cpu.pprof

go tool pprof -peek "insertIntoParent" report/fox_v1_baseline/fox/heap.pprof

# Line-level inspection

go tool pprof -list "github.com/liteio-dev/liteio/pkg/storage/driver/zoo/fox.(*store).putPreparedEntry" report/fox_v2e_final/fox/heap.pprof
```

### Unit tests added for regression protection

```bash
go test ./pkg/storage/driver/zoo/fox -run 'TestFox'
```

---

## References

- PVLDB paper PDF: <https://www.vldb.org/pvldb/vol17/p3442-yoon.pdf>
- Bf-Tree docs / project page: <https://github.com/XiangpengHao/bf-tree-docs>
- Bf-Tree paper announcement / metadata page: <https://collaborate.princeton.edu/en/publications/bf-tree-cache-optimized-b-trees-for-modern-hardware>
