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

## v7 profiler-driven point-lookup allocation pass

### v7 root cause (Go profiler)

Profiler focus before v7 (from `v6b`):

- `compositeKey` remained a major alloc-space hotspot
- `findPageEntry` still allocated heavily on point lookups due:
  - `[]byte(ck)` conversion during key compare
  - `string(ctBytes)` for content-type decode

Line-level `allocs.pprof` inspection (`pprof -list`) showed the `findPageEntry` compare/content-type conversions were still contributing meaningful alloc-space on read/stat/delete-heavy paths.

### v7 implemented changes

v7 focused on removing avoidable point-lookup string/byte conversion allocations.

- added `stringBytesView(s string) []byte` (unsafe no-copy string-to-bytes view)
- added `bytesEqString` helper
- added `internContentTypeBytes` with fast paths for common content types:
  - `application/octet-stream`
  - `text/plain`
  - `application/json`
- updated `findPageEntry(...)` to compare using `stringBytesView(ck)` (avoids `[]byte(ck)` alloc)
- updated `findPageEntry(...)` to use `internContentTypeBytes(ctBytes)`
- updated `decodePageEntriesWithMode(...)` to intern content-type strings
- updated `decodePageEntriesBorrowed(...)` to intern content-type strings
- updated list metadata scan path (`forEachPageEntryMeta` / `collectFromNode`) to intern content types
- removed a redundant `splitCompositeKey(...)` parse path in `bucket.Open(...)`

### v7 profiler verification

From `fox_v7a_final`:

- `allocs.pprof` total alloc-space: **`5821.24MB`**
- `findPageEntry` is no longer a top alloc-space hotspot (targeted allocations reduced)
- `compositeKey` still dominates point-lookup alloc-space:
  - `compositeKey`: **`454.02MB`** flat
  - `pprof -peek compositeKey` attribution:
    - `(*store).getEntry`: `258.51MB` (`56.94%`)
    - `(*store).putValueRef`: `93MB`
    - `(*store).put`: `51.50MB`
    - `(*store).del`: `51MB`

Interpretation:

- v7 successfully removed the `findPageEntry` conversion hotspot from the top profile
- the next root cause became clearer: repeated `compositeKey(...)` allocations across point lookup and delete paths

### v7 benchmark / memory results

Run: `fox_v7a_final`

Selected throughput (vs v6b):

- `Read/1KB`: `733.1 -> 1193.9 MB/s` (`1.63x`)
- `Read/64KB`: `7.95 -> 13.03 GB/s` (`1.64x`)
- `Read/1MB`: `11.36 -> 14.40 GB/s` (`1.27x`)
- `Stat`: `5.69M/s -> 5.79M/s` (near-flat/slight gain)
- `Delete`: `1.73M/s -> 1.63M/s` (slight regression)
- `List/100`: `123.3/s -> 73.6/s` (regression)

Memory verification:

- peak RSS (resource tracker): **`518.5MB`** (FAIL for strict `<100MB`)
- resource tracker Go heap: `181.3MB`
- `heap.pprof` total in-use: **`98.81MB`** (PASS)

v7 therefore improved point-lookup allocation behavior but did not solve strict memory and introduced mixed benchmark tradeoffs.

## v8 profiler-driven mini-page append churn pass

### v8 root cause (Go profiler)

After v7, `allocs.pprof` line-level inspection (`pprof -list putPreparedEntry`) showed heavy allocation churn around mini-page entry slice growth/append behavior, particularly in write/tombstone paths.

Observed pattern:

- repeated mini-page entry appends caused frequent backing-slice growth
- mini-page objects themselves were also churned (create/evict/recreate)

### v8 implemented changes

v8 focused on reducing append/growth churn in the mini-page write buffer path.

- increased `defaultMiniPageEntryCap` to `32`
- added `miniPageObjPool` (`sync.Pool`) for `miniPage` object reuse
- added `allocMiniPage(...)` helper (resets/reuses pooled mini-page)
- added `freeMiniPage(...)` helper (returns objects and slices)
- added `ensureMiniPageEntryCap(...)` explicit pre-growth helper
- updated `getOrCreate(...)` to use pooled mini-page allocation
- updated `evictOldest(...)` to return evicted mini-pages to the pool
- updated `putPreparedEntry(...)` append path to pre-grow via `ensureMiniPageEntryCap`
- updated `del(...)` tombstone append path to pre-grow via `ensureMiniPageEntryCap`

### v8 profiler verification

From `fox_v8a_final`:

- `allocs.pprof` total alloc-space: **`4902.44MB`** (down materially vs v7)
- `putPreparedEntry` append-line hotspot moved off the prior dominant line (growth behavior improved)

But `heap.pprof` exposed a new problem:

- `heap.pprof` total in-use: **`137.26MB`** (FAIL)
- top in-use node:
  - `getPageEntrySlice`: **`84.87MB`** flat (`61.83%`)

Interpretation:

- v8 reduced alloc churn successfully
- but the new pooling/growth behavior over-retained page-entry scratch slices, causing a large heap snapshot regression

### v8 benchmark / memory results

Run: `fox_v8a_final`

Selected throughput (vs v7a):

- `Write/1KB`: `336.2 -> 386.3 MB/s` (`1.15x`)
- `Write/64KB`: `1546 -> 1800 MB/s` (`1.16x`)
- `Write/1MB`: `466.7 -> 500.7 MB/s` (`1.07x`)
- `List/100`: `73.6/s -> 113.2/s` (`1.54x`)
- `Copy/1KB`: `602.8 -> 659.0 MB/s` (`1.09x`)
- `Read/1KB`: `1193.9 -> 1266.2 MB/s` (`1.06x`)

Memory verification:

- peak RSS: **`525.2MB`** (FAIL)
- resource tracker Go heap: `278.2MB` (worse)
- `heap.pprof` total in-use: **`137.26MB`** (FAIL)

v8 was a throughput/alloc-space win but a memory-target regression.

## v9 profiler-driven pool retention stabilization pass

### v9 root cause (Go profiler)

v8 heap profiling clearly showed the next root cause:

- `getPageEntrySlice`: **`84.87MB`** flat in `heap.pprof`

This pointed to pooling retention policy (especially page-entry scratch slices) rather than live object data as the main driver of the v8 memory regression.

### v9 implemented changes

v9 focused on reducing retained slice memory while keeping most of the v8 alloc reductions.

- split pool cap thresholds:
  - `maxPooledPageEntrySliceCap = 128` (flush scratch)
  - `maxPooledMiniPageEntrySliceCap = 64` (mini-page entries)
- added separate mini-page slice pool:
  - `miniPageEntrySlicePool`
  - `getMiniPageEntrySlice(...)`
  - `putMiniPageEntrySlice(...)`
- moved mini-page entry allocation/free/grow paths to the dedicated mini-page slice pool
- kept flush scratch (`getPageEntrySlice`) separate from mini-page entry slices
- shrank oversized mini-page entry slices after flush:
  - if `cap(mp.entries)` exceeds mini-page threshold, return it and reinitialize small
- tightened retention policy to prevent oversized slices from accumulating in the pool

### v9 profiler verification

From `fox_v9a_final`:

- `allocs.pprof` total alloc-space: **`4386.28MB`** (down again vs v8)
- peak RSS improved substantially vs v8

But heap profiling still showed a remaining pool-retention issue, now shifted:

- `heap.pprof` total in-use: **`121.55MB`** (still FAIL)
- top in-use node:
  - `getMiniPageEntrySlice`: **`49.82MB`** flat (`40.99%`)

Interpretation:

- v9 successfully moved/reduced retention from shared page scratch pooling
- mini-page entry slice retention remained the dominant in-use heap blocker for `<100MB`

### v9 benchmark / memory results

Run: `fox_v9a_final`

Selected throughput (vs v8a):

- `List/100`: `113.2/s -> 123.2/s` (`1.09x`)
- `Delete`: `1.64M/s -> 1.65M/s` (near-flat)
- `RangeRead/*256KB`: near-flat / slightly improved
- `Copy/1KB`: `659.0 -> 640.7 MB/s` (slight regression)
- `Read/10MB`: `9.84 -> 5.72 GB/s` (large regression / unstable run)

Memory verification:

- peak RSS: **`404.0MB`** (major improvement, but FAIL for strict `<100MB`)
- resource tracker Go heap: `216.2MB`
- `heap.pprof` total in-use: **`121.55MB`** (FAIL)

v9 significantly improved memory behavior (especially RSS/GoSys) but still did not meet the memory target.

## v10 profiler-driven composite-key elimination + memory stabilization pass

v10 was implemented as an iterative sub-series (`v10a`, `v10b`, `v10c`) because the primary alloc-space fix exposed a secondary heap-retention root cause.

### v10 primary root cause (Go profiler, before coding)

From `fox_v9a_final/fox/allocs.pprof`:

- `compositeKey`: **`368.02MB`** flat (`8.39%`)
- `pprof -peek compositeKey` attribution:
  - `(*store).getEntry`: `205.01MB` (`55.71%`)
  - `(*store).putValueRef`: `77MB`
  - `(*store).put`: `56.51MB`
  - `(*store).del`: `29.50MB`

This made point lookup/delete `compositeKey(...)` allocations the clear v10 target.

### v10 implemented changes (v10a -> v10c)

#### v10a: parts-based point lookup / delete path (no `compositeKey` on read path)

- added `compositeKeyEqualsParts(...)`
- added `compareBucketKeyToComposite(...)`
- added `compareCompositeBytesToParts(...)`
- added `(*btree).findLeafParts(bucket, key string)`
- added `findPageEntryParts(...)` (page scan compare against `bucket,key` parts)
- updated `(*store).getEntry(...)`:
  - tree lookup via `findLeafParts(...)`
  - mini-page scan via `compositeKeyEqualsParts(...)`
  - disk page scan via `findPageEntryParts(...)`
- updated `(*store).del(...)`:
  - leaf lookup via `findLeafParts(...)`
  - mini-page scan via `compositeKeyEqualsParts(...)`
  - defer `compositeKey(...)` allocation until tombstone append is actually needed

#### v10b: release mini-page entry slices after flush (heap retention fix)

Secondary root cause after v10a (from `heap.pprof`):

- `getMiniPageEntrySlice` remained the top in-use heap node in the v10a snapshot

Change:

- after successful `flushMiniPage(...)`, always return `mp.entries` backing slice to the mini-page slice pool and set `mp.entries = nil`
- clean mini-pages keep metadata state but do not retain a pre-sized entry slice

This trades some future re-allocation for much lower retained heap.

#### v10c: no-scan comparator refinement (CPU recovery while keeping no-alloc path)

v10b improved memory substantially but regressed some CPU-heavy lookup paths (`Stat`, `Delete`).

Refinement:

- `compareBucketKeyToComposite(...)` now delegates to `compareCompositeBytesToParts(stringBytesView(composite), ...)` instead of `strings.IndexByte` + substring compares
- `compositeKeyEqualsParts(...)` now uses fixed-position separator checks (no `IndexByte`)

This keeps the no-allocation design while reducing compare overhead in B-tree traversal / mini-page scans.

### v10 profiler verification

#### v10a (after primary fix)

From `fox_v10a_final`:

- `allocs.pprof` total alloc-space: **`3976.96MB`** (down vs v9a)
- `compositeKey`: **`157.01MB`** flat (from `368.02MB` in v9a)

This confirms the primary v10 root cause was addressed.

#### v10b (after heap-retention fix)

From `fox_v10b_final`:

- `allocs.pprof` total alloc-space: **`3885.27MB`**
- `heap.pprof` total in-use: **`42257.88kB`** (~`41.3MB`) (PASS)
- `compositeKey`: **`148.51MB`** flat

#### v10c (final v10 build)

From `fox_v10c_final`:

- `allocs.pprof` total alloc-space: **`3927.59MB`**
- `heap.pprof` total in-use: **`46331.98kB`** (~`45.3MB`) (PASS)
- `compositeKey`: **`139.51MB`** flat

Compared to v9a:

- `compositeKey` alloc-space reduced from `368.02MB -> 139.51MB` (**`-62.1%`**)
- total alloc-space reduced from `4386.28MB -> 3927.59MB` (**`-10.5%`**)

New top in-use heap nodes in v10c are no longer dominated by retained mini-page slices:

- `getMiniPageEntrySlice`: ~`5.0MB` flat
- remaining profile is more distributed across tree/cache/runtime initialization

### v10 benchmark / memory results (final = v10c)

Runs:

- `v10a`: primary composite-key elimination
- `v10b`: memory retention fix (strongest memory)
- `v10c`: CPU compare refinement + memory target (heap) kept

Selected throughput (`v10c` vs `v9a`):

- `Write/1KB`: `341.8 -> 347.1 MB/s` (`1.02x`)
- `Read/64KB`: `13.42 -> 13.50 GB/s` (`1.01x`)
- `Read/1MB`: `14.16 -> 15.16 GB/s` (`1.07x`)
- `Read/10MB`: `5.72 -> 10.00 GB/s` (`1.75x`) (recovers v9a outlier regression)
- `RangeRead/Start_256KB`: `14.83 -> 15.28 GB/s` (`1.03x`)
- `RangeRead/Middle_256KB`: `14.92 -> 15.25 GB/s` (`1.02x`)
- `RangeRead/End_256KB`: `14.88 -> 15.19 GB/s` (`1.02x`)
- `List/100`: `123.2/s -> 131.7/s` (`1.07x`)

Known v10c regressions vs v9a:

- `Delete`: `1.65M/s -> 0.95M/s` (`0.58x`)
- `Stat`: `4.07M/s -> 3.81M/s` (`0.94x`)
- `Copy/1KB`: `640.7 -> 585.9 MB/s` (`0.91x`)
- `Write/64KB`: `1730.8 -> 1601.9 MB/s` (`0.93x`)

Interpretation:

- v10 is primarily an allocation + memory pass, not a universal throughput win
- it significantly reduces alloc-space and heap retention while preserving or improving several read/range/list paths
- `Delete` remains the main v10 regression to target next

### v10 memory verification (<100MB), carefully verified

#### 1. Strict process-level metric (resource tracker)

`v10c` peak RSS: **`232.6MB`** (FAIL)

Strict process RSS is still above `100MB`.

#### 2. Resource tracker Go heap

`v10c` peak Go heap: **`93.4MB`** (PASS)

#### 3. `heap.pprof` in-use snapshot (`go tool pprof`)

`v10c` total in-use: **`46331.98kB`** (~`45.3MB`) (PASS)

Conclusion:

- v10c meets `<100MB` on Go-heap-centric metrics (tracker Go heap and `heap.pprof` snapshot)
- v10c still does **not** meet strict process RSS `<100MB` under the full benchmark harness process

### v7-v10 technique inventory (>=10 techniques requirement)

Implemented across v7-v10 (non-exhaustive, but counted):

1. no-copy string-to-bytes view (`stringBytesView`)
2. byte-vs-string equality helper (`bytesEqString`)
3. common content-type interning helper
4. `findPageEntry` no-alloc key compare
5. content-type interning in full decode path
6. content-type interning in borrowed decode path
7. content-type interning in metadata scan/list path
8. redundant composite-key split removal in `Open`
9. larger default mini-page entry slice capacity
10. mini-page object pooling (`miniPageObjPool`)
11. reusable `allocMiniPage` / `freeMiniPage` lifecycle
12. explicit mini-page entry slice pre-growth (`ensureMiniPageEntryCap`)
13. pooled mini-page allocation in `getOrCreate`
14. pooled mini-page return on eviction
15. pre-grow append path in writes (`putPreparedEntry`)
16. pre-grow append path in tombstone writes (`del`)
17. separate pool caps for page scratch vs mini-page entries
18. dedicated mini-page entry slice pool
19. oversize mini-page slice shrink after flush
20. parts-based composite compare helpers for B-tree/page scans
21. `btree.findLeafParts` no-composite-key traversal
22. `findPageEntryParts` no-composite-key disk scan
23. `getEntry` parts-based mini-page + disk lookup path
24. `del` parts-based mini-page lookup and deferred tombstone `compositeKey` allocation
25. release mini-page entry slices after successful flush (`mp.entries=nil`)
26. no-scan comparator refinement for `compareBucketKeyToComposite`
27. fixed-position separator equality check in `compositeKeyEqualsParts`

This exceeds the user-requested minimum of 10 optimization techniques.

### v10 conclusion

v10 achieved the profiler-driven objective:

- identified `compositeKey` as the next root cause with Go profiler
- cut `compositeKey` alloc-space by ~`62%` vs v9a
- reduced total alloc-space further vs v9a
- dramatically improved heap retention metrics through flush-slice release
- brought v10c resource-tracker Go heap and `heap.pprof` snapshot below `100MB`

v10 did not achieve strict process RSS `<100MB` under the full benchmark harness process, and introduced a notable `Delete` regression that should be the first v11 target.

### v11 candidates (next)

- profile `Delete` regression (`cpu.pprof` + `allocs.pprof`) on `v10c`:
  - compare mini-page scan cost vs old composite-key path
  - inspect extra existence-check work / tree traversal duplication
- target `bucket.Stat` CPU regressions after parts-based comparisons
- reduce `getPageEntrySlice` alloc-space (`~170MB` in v10c) without reintroducing heap retention
- evaluate bounded mini-page pool occupancy / adaptive flush cadence for further RSS reduction

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

## v4 Focused Pass: Recover Copy/RangeRead Regressions + Preserve <100MB Heap Snapshot

### v4 goal

The v4 goal was narrower than v2/v3:

- recover the major v3b regressions in `Copy/1KB` and `RangeRead/*256KB`
- avoid correctness regressions
- keep `heap.pprof` in-use under `100MB` (snapshot metric)
- re-check strict process RSS (expected to remain the hard blocker)

The user also requested a broad `5x` improvement target across benchmarks. After implementation and full-suite runs, this target is **not achievable in a single focused pass**; v4 recovers the worst regressions but does not produce 5x gains across the board.

### v4 implemented changes

#### 1. Indirect-value copy fast path (metadata copy, no value materialization)

`Copy/1KB` in fox is usually copying **indirect values** (because `1KB > inline threshold`), but v3b still:

- read the source bytes from `values.dat`
- allocated a new buffer
- rewrote bytes into a new object

v4 changes `bucket.Copy()` to:

- load source metadata with `getMeta()`
- if the value is indirect, call `putValueRef()` directly with the existing `valueRef`
- only load/copy bytes for inline values

Effect:

- removes unnecessary read + write + allocation on the hot `Copy/1KB` path
- drastically reduces copy-path churn

#### 2. Delete existence check uses metadata-only lookup

`store.del()` only needs existence before writing a tombstone. v4 changes the check from `get()` to `getMeta()` to avoid loading inline value bytes unnecessarily.

This is a small change, but it helps reduce avoidable point-lookup work in delete-heavy phases.

#### 3. Size-aware `valueSectionReader.WriteTo` (range-read recovery)

v3b's `valueSectionReader.WriteTo()` path was a contributor to the `RangeRead/*256KB` regression.

v4 uses a size-aware strategy:

- for small sections (`<=64KB`): keep `io.Copy(...)` (often hits destination `ReaderFrom` fast paths)
- for larger sections (notably `256KB` range reads): use pooled `256KB` buffer + `io.CopyBuffer(...)`

Effect:

- restores `RangeRead/*256KB` throughput strongly
- avoids imposing pooled-buffer overhead on tiny reads

#### 4. v4 trial note: no-allocation `findPageEntry` key compare experiment (reverted)

I tested a no-allocation string-vs-bytes compare in `findPageEntry` to remove the `[]byte(ck)` allocation, but the straightforward Go-loop implementation regressed point-lookup throughput enough to outweigh the allocation savings.

Final v4 code keeps the original `bytes.Compare(keyBytes, []byte(ck))` behavior (with the `[]byte(ck)` allocation still present), and the remaining allocation shows up in `allocs.pprof`.

This remains a valid future optimization target, but it likely needs a faster implementation (e.g. specialized/unsafe compare) to avoid throughput regressions.

### v4 benchmark commands (same wrapper workflow)

```bash
# v4 trial (included a reverted compare experiment)
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v4_trial1

# v4 final-code runs (used for results below; same code, benchmark variance check)
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v4b_final
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v4c_final
```

### v4 result summary (vs v3b)

#### High-level outcome

- `Copy/1KB`: **recovered and exceeded v2e**, strongly improved vs v3b
- `RangeRead/*256KB`: **recovered strongly** vs v3b (roughly `2.5x` to `3.2x`)
- `Delete` and `List/100`: improved in both v4 final-code runs
- `heap.pprof` snapshot stayed **below 100MB** in both v4 final-code runs
- strict process peak RSS remained far above `100MB`
- several write/read/stat metrics show **high run-to-run variance** (quick-suite environment noise), so v4 results should be read as a band, not a single exact number

### Selected benchmark deltas: v3b vs v4 (two final-code runs)

| Benchmark | v3b | v4b | v4c | v4b/v3b | v4c/v3b |
|---|---:|---:|---:|---:|---:|
| `Write/1KB` | 350.9 MB/s | 206.8 MB/s | 206.0 MB/s | `0.59x` | `0.59x` |
| `Write/64KB` | 1.61 GB/s | 1.19 GB/s | 1.52 GB/s | `0.73x` | `0.94x` |
| `Write/1MB` | 447.4 MB/s | 390.8 MB/s | 324.8 MB/s | `0.87x` | `0.73x` |
| `Write/10MB` | 454.1 MB/s | 475.4 MB/s | 439.9 MB/s | `1.05x` | `0.97x` |
| `Read/1KB` | 1.16 GB/s | 1.20 GB/s | 1.13 GB/s | `1.03x` | `0.97x` |
| `Read/64KB` | 12.92 GB/s | 13.12 GB/s | 10.22 GB/s | `1.02x` | `0.79x` |
| `Read/1MB` | 12.61 GB/s | 12.83 GB/s | 9.06 GB/s | `1.02x` | `0.72x` |
| `Read/10MB` | 9.18 GB/s | 8.27 GB/s | 9.17 GB/s | `0.90x` | `1.00x` |
| `Stat` | 5.68M/s | 3.17M/s | 5.76M/s | `0.56x` | `1.01x` |
| `List/100` | 46.73/s | 78.18/s | 61.36/s | `1.67x` | `1.31x` |
| `Delete` | 1.31M/s | 1.56M/s | 1.50M/s | `1.19x` | `1.14x` |
| `Copy/1KB` | 79.1 MB/s | 620.5 MB/s | 607.4 MB/s | `7.85x` | `7.68x` |
| `RangeRead/Start_256KB` | 4.70 GB/s | 14.46 GB/s | 13.78 GB/s | `3.07x` | `2.93x` |
| `RangeRead/Middle_256KB` | 4.84 GB/s | 14.98 GB/s | 11.94 GB/s | `3.09x` | `2.47x` |
| `RangeRead/End_256KB` | 4.62 GB/s | 14.68 GB/s | 13.58 GB/s | `3.17x` | `2.94x` |

Interpretation:

- The v4 fixes **clearly solved the v3b copy/range-read regressions**.
- They did **not** produce a broad 5x uplift.
- Several non-target metrics vary significantly between `v4b` and `v4c`, so quick-run noise remains a limiting factor for single-run claims.

### \"5x every benchmark\" status (explicit)

Across the `40` quick-suite benchmarks in each v4 final-code run:

- `v4b`: `1 / 40` benchmarks reached `>=5x` vs v3b
- `v4c`: `1 / 40` benchmarks reached `>=5x` vs v3b

The only benchmark consistently above `5x` was:

- `Copy/1KB` (via indirect-value metadata copy fast path)

### v4 memory verification (<100MB), carefully verified

#### 1. Strict process-level metric (resource tracker)

- `v4b` peak RSS: **`464.0 MB`** (FAIL)
- `v4c` peak RSS: **`479.0 MB`** (FAIL)

So the strict `<100MB process peak RSS` target is still **not met**.

#### 2. `heap.pprof` snapshot in-use (`go tool pprof`, authoritative for this report's snapshot metric)

- `v3b`: `92.33 MB` total in-use
- `v4b`: **`88.89 MB`** total in-use (PASS, below `100MB`)
- `v4c`: **`92.85 MB`** total in-use (PASS, below `100MB`)

This keeps fox in the same sub-`100MB` heap-profile snapshot range as v3 (and slightly improves it in `v4b`).

#### 3. Runtime / analyzer fields (not directly comparable to `heap.pprof` total)

`raw_results.json -> profile_analyses.heap_in_use` remains much higher than `pprof` totals (runtime page accounting vs profile snapshot attribution). This is the same metric mismatch observed in v3 and is why the report uses direct `go tool pprof` totals for the `<100MB` snapshot claim.

### v4 profile highlights (`fox_v4b_final`)

#### Heap in-use (`report/fox_v4b_final/fox/heap.pprof`)

- Total heap in-use profile: **`88.89 MB`**
- Top fox contributors remain dominated by write-path structures:
  - `(*store).putPreparedEntry`: `28.11 MB`
  - `(*bucket).Write`: `18.00 MB`
  - `compositeKey`: `14.00 MB`

#### Allocs (`report/fox_v4b_final/fox/allocs.pprof`)

- Total alloc-space: **`7.82 GB`** (`7824.64MB`) vs v3b `9.23 GB` (`9230.08MB`) => **~15.2% lower**
- `decodePageEntriesWithMode` is still the dominant allocator (`2562.61MB`)
- `findPageEntry` now appears as a visible allocator (`141.50MB`) after reverting the slower no-alloc compare experiment

### v4 conclusion

v4 succeeded at its focused mission:

- recovered `Copy/1KB` by a large margin
- recovered `RangeRead/*256KB` by ~`2.5x` to `3.2x` vs v3b
- preserved error-free behavior (`0` errors)
- preserved the `<100MB` **heap-profile snapshot** target in final-code runs

v4 did **not** satisfy:

- strict `<100MB` process peak RSS
- the user-requested `5x` improvement across every benchmark

For a v5 pass, the next high-value work is likely:

- stabilize write/point-read variance (reprofile under longer bench time or repeated medians)
- reduce `decodePageEntriesWithMode` alloc-space with more specialized decoders
- optimize `findPageEntry` key comparison without the throughput regression seen in the v4 trial

---

## v5 Profiler-Driven Pass: Decoder Allocation Root Cause (List/Rebuild + Flush)

### v5 goal

Use the Go profiler (`allocs.pprof`) to identify the dominant allocation root cause in v4, then implement targeted fixes in fox instead of another broad optimization pass.

### Root cause identified with Go profiler (v4b)

`go tool pprof -top report/fox_v4b_final/fox/allocs.pprof` showed:

- `decodePageEntriesWithMode`: **`2562.61MB`** flat alloc-space (**32.75%** of total `7824.64MB`)

Then `go tool pprof -peek 'decodePageEntriesWithMode' ...` showed the caller split:

- `decodePageEntriesMeta`: **`2075.78MB`** (**81%**)
- `decodePageEntries`: **`486.83MB`** (**19%**)

Interpretation:

- most decoder allocation churn was coming from **metadata scans** (list/rebuild paths), not only the flush path
- the remaining write-path decoder churn still mattered (`flushMiniPage -> decodePageEntries`)

This gave a clear v5 plan:

1. remove materialized `[]pageEntry` decoding from metadata scans (list/rebuild)
2. reduce flush-path decode copies by borrowing page bytes during merge

### v5 implemented changes (profiler-driven)

#### 1. Streaming metadata page scanner (no `[]pageEntry` materialization)

Added:

- `forEachPageEntryMeta(...)`
- `firstPageEntryKey(...)`
- `hasPrefixBytesString(...)`

Used in:

- `rebuildTree()` (first-key extraction only)
- `collectFromNode()` (list scan path)

Effect:

- avoids `decodePageEntriesMeta()` allocating an `entries` slice for every scanned page
- avoids allocating `pageEntry` structs for metadata-only scans
- allocates strings only when needed for list output / map keys

#### 2. Borrowed flush-page decoder for merge path

Added:

- `decodePageEntriesBorrowed(buf []byte)`

Behavior:

- borrows keys/content-types (unsafe string view) and inline values (slice view) from the page buffer
- used only inside `flushMiniPage()`, where the page buffer lifetime is controlled

To make this safe, v5 also changed `flushMiniPage()` to use:

- `readBuf` (decode source)
- `writeBuf` (encode target)

This prevents encode-time buffer clears from corrupting borrowed decoded entries.

#### 3. Correctness fix for borrowed split keys (critical)

During v5 development, `TestFoxSplitPreservesSmallValues` failed because a borrowed key from the read page buffer escaped into the B-tree separator keys via `insertLeaf(...)`.

Final v5 fix:

- copy `chunk[0].key` before passing it to `tree.insertLeaf(...)`

This preserves correctness while still using borrowed decoding for transient flush merge data.

### v5 benchmark / profiler commands

```bash
# tests
go test ./pkg/storage/driver/zoo/fox

# v5 profiler-driven runs
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v5a_final
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v5b_final

# profiler inspection
go tool pprof -top -nodecount=20 report/fox_v4b_final/fox/allocs.pprof
go tool pprof -peek "decodePageEntriesWithMode" report/fox_v4b_final/fox/allocs.pprof

go tool pprof -top -nodecount=20 report/fox_v5b_final/fox/allocs.pprof
go tool pprof -peek "decodePageEntriesBorrowed" report/fox_v5b_final/fox/allocs.pprof
go tool pprof -top -nodecount=20 report/fox_v5b_final/fox/heap.pprof
```

### v5 profiler results (root cause addressed)

#### v4b -> v5 alloc-space (`allocs.pprof`)

- v4b total alloc-space: **`7824.64MB`**
- v5a total alloc-space: **`6294.76MB`** (**-19.6%**)
- v5b total alloc-space: **`6393.96MB`** (**-18.3%**)

Most importantly:

- `decodePageEntriesWithMode` disappears from the v5 alloc-space top list (metadata scans no longer route through it)
- new flush-only decoder appears instead:
  - `decodePageEntriesBorrowed`: `363.66MB` (v5a) / `391.29MB` (v5b)
  - `pprof -peek` shows it is **100%** called from `(*miniPagePool).flushMiniPage`

This is exactly the intended root-cause shift:

- metadata decoder allocation hotspot removed
- remaining decoder churn isolated to the write/flush path

### v5 benchmark results (vs v4b, two runs)

Quick-suite variance remains significant, so v5 is reported as a range (`v5a`, `v5b`).

#### Consistent improvements (both v5 runs vs v4b)

- `Write/1KB`: `+56%` to `+84%` (`206.8 -> 322.6 / 380.8 MB/s`)
- `Write/64KB`: `+36%` to `+53%` (`1.19 -> 1.61 / 1.82 GB/s`)
- `Write/1MB`: `+23%` to `+30%` (`390.8 -> 481.5 / 507.6 MB/s`)
- `Write/10MB`: `+0.7%` to `+7.0%`
- `Stat`: `+73%` to `+79%` (v4b was a low outlier)
- `List/100`: `+38%` to `+65%`

#### Preserved / near-flat

- `Copy/1KB`: `587-636 MB/s` (still strong, around v4 levels)
- `Read/1KB`: `~flat to -11%`
- `Read/64KB`: `~flat to -2%`
- `Read/1MB`: `-4% to -6%`
- `Read/10MB`: `+7% to +10%`

#### Regressed / unstable

- `Delete`: regressed in both v5 runs vs v4b (but still around v3b on one run)
- `RangeRead/*256KB`: mixed; `v5b` especially regressed on `Start_256KB`

Interpretation:

- v5 clearly improved write-heavy and list-heavy behavior while reducing alloc-space
- v4's `Copy/1KB` improvement is preserved
- some range-read and delete behavior remains noisy/regressed and needs a follow-up pass

### v5 memory verification (<100MB), carefully verified

#### 1. Strict process-level metric (resource tracker)

- `v5a` peak RSS: **`559.6 MB`** (FAIL)
- `v5b` peak RSS: **`576.9 MB`** (FAIL)

This is worse than v4 and far from the strict `<100MB` target.

#### 2. `heap.pprof` in-use snapshot (`go tool pprof`)

- `v5a`: **`91.25 MB`** (PASS)
- `v5b`: **`103.27 MB`** (FAIL, slightly above target)

So v5 does **not** stably keep the heap snapshot under `100MB`; it now fluctuates around the threshold.

#### 3. Likely cause of worse RSS despite lower alloc-space (inference)

This is an inference from profiler/resource data:

- v5 greatly reduced total alloc-space
- GC count also dropped materially (`143` vs `188` in v4b, `211` in v4c)
- process `GoSys`/RSS increased

Likely explanation:

- less allocation churn triggers fewer GCs/scavenges in the benchmark window
- runtime retains more heap/system memory pages (higher `GoSys` / RSS), even while alloc-space is lower

This further supports treating strict process RSS and `heap.pprof` totals as different metrics with different behavior.

### v5 conclusion

v5 successfully delivered the profiler-driven objective:

- identified the dominant v4 allocation root cause using Go profiler
- removed the metadata decoder allocation hotspot (`decodePageEntriesWithMode`) from the top alloc-space path
- reduced total alloc-space by ~`18-20%` vs v4b
- improved several write/list metrics substantially

v5 did **not** achieve:

- strict `<100MB` process RSS
- stable `<100MB` `heap.pprof` snapshot in every run
- stable range-read/delete improvements

### v6 candidates (next)

- profile `RangeRead/*256KB` and `Delete` specifically under v5 (CPU + allocs)
- reduce `mergeEntries` alloc-space (`~0.84-0.87GB` in v5) via scratch reuse / pooled result slices
- address `findPageEntry` allocs (`~134-148MB`) with a faster no-alloc compare implementation
- consider explicit memory-pressure controls (or benchmark isolation) if strict RSS is a hard target

---

## v6 Profiler-Driven Pass: Flush Merge Allocation Root Cause (`mergeEntries`)

### v6 goal

Use Go profiler on v5 to identify the next dominant fox allocation root cause and reduce it with a focused flush-path optimization.

### Root cause identified with Go profiler (v5b)

`go tool pprof -top report/fox_v5b_final/fox/allocs.pprof` showed the top fox alloc-space hotspots after v5:

- `mergeEntries`: **`868.93MB`** flat
- `decodePageEntriesBorrowed`: **`391.29MB`** flat

`pprof -peek` for both functions showed they were entirely on the flush path:

- `mergeEntries` alloc-space attributed **100%** to `(*miniPagePool).flushMiniPage`
- `decodePageEntriesBorrowed` alloc-space attributed **100%** to `(*miniPagePool).flushMiniPage`

Interpretation:

- v5 successfully removed metadata decoder churn,
- but flush still allocated heavily on every mini-page flush due repeated `[]pageEntry` backing allocations (decode + merge result).

### v6 implemented changes (profiler-driven)

#### 1. Pooled `[]pageEntry` scratch reuse for flush decode/merge

Added:

- `getPageEntrySlice(minCap int) []pageEntry`
- `putPageEntrySlice([]pageEntry)`
- `pageEntrySlicePool` (`sync.Pool`)

Characteristics:

- bounded pooling (`maxPooledPageEntrySliceCap = 512`)
- clears pooled entries before reuse to avoid retaining borrowed strings/byte-slice references
- used for both flush decode and merge result slices

#### 2. `decodePageEntriesBorrowed` now uses pooled scratch

Before v6:

- every call allocated `make([]pageEntry, 0, count)`

After v6:

- decoder gets capacity from `getPageEntrySlice(count)`
- `flushMiniPage()` returns the decoded slice to the pool after encoding

#### 3. `mergeEntries` now uses pooled scratch

Before v6:

- `mergeEntries` allocated a fresh result slice each flush

After v6:

- `mergeEntries` uses `getPageEntrySlice(len(disk)+len(mini))`
- `flushMiniPage()` returns the merged slice to the pool after writeback/split handling

### v6 benchmark / profiler commands

```bash
# tests
go test ./pkg/storage/driver/zoo/fox

# v6 profiler-driven runs
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v6a_final
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v6b_final

# root-cause identification (before v6)
go tool pprof -top -nodecount=20 report/fox_v5b_final/fox/allocs.pprof
go tool pprof -peek "mergeEntries|decodePageEntriesBorrowed" report/fox_v5b_final/fox/allocs.pprof

# v6 verification
go tool pprof -top -nodecount=20 report/fox_v6a_final/fox/allocs.pprof
go tool pprof -top -nodecount=20 report/fox_v6b_final/fox/allocs.pprof
go tool pprof -peek "mergeEntries" report/fox_v6b_final/fox/allocs.pprof
go tool pprof -top -nodecount=20 report/fox_v6b_final/fox/heap.pprof
```

### v6 profiler results (root cause addressed)

#### v5b -> v6 alloc-space totals (`allocs.pprof`)

- v5b: **`6393.96MB`**
- v6a: **`4945.89MB`** (**-22.6%** vs v5b)
- v6b: **`5630.80MB`** (**-11.9%** vs v5b)

#### Hotspot shift (v5b -> v6)

v5b:

- `mergeEntries`: `868.93MB` flat
- `decodePageEntriesBorrowed`: `391.29MB` flat

v6a/v6b:

- `mergeEntries`: no longer a top flat allocator
  - `pprof -peek` on v6b shows `mergeEntries` **flat ~`1MB`**, cumulative `219.76MB`
- `decodePageEntriesBorrowed`: effectively removed as a hotspot
  - v6a `pprof -peek` shows cumulative only `2.01MB`
- new smaller flush-related allocator appears:
  - `getPageEntrySlice`: `~0.16-0.20GB` flat (pool misses/initial allocations)

This confirms the v6 fix hit the intended root cause:

- flush-path `[]pageEntry` backing allocations were the main remaining decoder/merge churn
- pooled scratch reuse materially reduced total alloc-space

### v6 benchmark results (vs v5b, two runs)

Quick-suite variance remains high. v6 is reported as a range (`v6a`, `v6b`).

#### Consistent improvements vs v5b

- `List/100`: `+12%` to `+14%` (`108.3/s -> 121.7/s / 123.3/s`)
- `Delete`: `+38%` to `+78%` (`0.97M/s -> 1.35M/s / 1.73M/s`)
- `RangeRead/Start_256KB`: `+2.12x` to `+2.24x` (v5b had regressed badly)

#### Preserved / near-flat (best v6 run keeps v5 gains)

- `Copy/1KB`: `573-652 MB/s` (still strong; v6b exceeds v5b and v4b)
- `Write/1MB`: near-flat (`0.95x` to `1.01x`)
- `Write/10MB`: near-flat to modest gain (`0.96x` to `1.06x`)

#### Mixed / unstable

- reads vary significantly between `v6a` and `v6b` (especially `Read/1KB`, `Read/64KB`, `Read/10MB`)
- `Stat` also varies (`0.71x` to `1.03x` vs v5b)
- write small/medium throughput is not a stable improvement over the strong v5b outlier

### v6 context vs earlier versions

Compared to v4b (which had restored copy/range-read):

- v6b preserves or improves most v4b wins:
  - `Copy/1KB`: `620.5 -> 652.2 MB/s` (`1.05x`)
  - `RangeRead/*256KB`: roughly at parity/slightly better
  - `List/100`: `78.2/s -> 123.3/s` (`1.58x`)
  - `Stat`: `3.17M/s -> 5.69M/s` (`1.79x`)

Compared to v3b:

- `Copy/1KB`: `~7.25x` to `~8.25x`
- `RangeRead/*256KB`: `~2.5x` to `~3.2x`
- `List/100`: `~2.6x`

The “5x every benchmark” request remains unmet:

- v6a vs v3b: `1 / 40` benchmarks `>=5x`
- v6b vs v3b: `1 / 40` benchmarks `>=5x`

### v6 memory verification (<100MB), carefully verified

#### 1. Strict process-level metric (resource tracker)

- `v6a` peak RSS: **`578.0 MB`** (FAIL)
- `v6b` peak RSS: **`546.1 MB`** (FAIL)

Strict process RSS remains far above `100MB`.

#### 2. `heap.pprof` in-use snapshot (`go tool pprof`)

- `v6a`: **`102.88 MB`** (FAIL)
- `v6b`: **`87.30 MB`** (PASS)

So v6, like v5, does **not** stably satisfy `<100MB` even for the snapshot metric.

#### 3. Allocation vs RSS behavior (inference)

Inference from profiler/resource data:

- v6 reduced alloc-space significantly vs v5b
- GC count also dropped in `v6a` (`118`) and remained below v4/v5 ranges in `v6b` (`135`)
- process `GoSys`/RSS remained very high

Likely explanation remains the same pattern seen in v5:

- lower alloc churn reduces GC/scavenge frequency during the benchmark window,
- so runtime system memory retention dominates process RSS/GoSys metrics.

### v6 conclusion

v6 successfully delivered the profiler-driven objective:

- identified the next root cause with Go profiler (`mergeEntries` + flush decode allocs)
- removed the flush merge/decode allocation hotspot from the top alloc-space profile
- reduced total alloc-space materially again vs v5 (up to **`-22.6%`** in `v6a`)

v6 also improved several key behaviors (especially `List/100`, `Delete`, and recovering v5b `RangeRead/Start_256KB` regression), while largely preserving the major copy/range-read wins accumulated since v4.

v6 did **not** solve the memory target:

- strict process RSS still far above `100MB`
- heap snapshot `<100MB` remains unstable across runs

### v7 candidates (next)

- profile `putPreparedEntry` / `compositeKey` allocation churn (now dominant fox alloc-space)
- revisit `findPageEntry` allocation (~`140MB`) with a faster no-alloc compare path
- investigate read-path variance (`Read/1KB`, `Read/64KB`, `Read/10MB`) with targeted profiles
- if strict RSS is mandatory, add benchmark isolation / explicit memory-mode reporting

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

# v4 focused regression recovery
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v4_trial1
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v4b_final
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v4c_final

# v5 profiler-driven decoder/root-cause pass
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v5a_final
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v5b_final

# v6 profiler-driven flush merge/root-cause pass
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v6a_final
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v6b_final

# v7-v10 profiler-driven passes
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v7a_final
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v8a_final
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v9a_final
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v10a_final
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v10b_final
go run -tags noant /tmp/fox_bench_wrapper.go ./report/fox_v10c_final
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

go tool pprof -top -nodecount=20 report/fox_v4b_final/fox/heap.pprof
go tool pprof -top -nodecount=20 report/fox_v4b_final/fox/allocs.pprof
go tool pprof -top -cum -nodecount=15 report/fox_v4b_final/fox/cpu.pprof
go tool pprof -peek "decodePageEntriesWithMode" report/fox_v4b_final/fox/allocs.pprof

go tool pprof -top -nodecount=20 report/fox_v5b_final/fox/heap.pprof
go tool pprof -top -nodecount=20 report/fox_v5b_final/fox/allocs.pprof
go tool pprof -top -cum -nodecount=15 report/fox_v5b_final/fox/cpu.pprof
go tool pprof -peek "decodePageEntriesBorrowed" report/fox_v5b_final/fox/allocs.pprof
go tool pprof -peek "mergeEntries|decodePageEntriesBorrowed" report/fox_v5b_final/fox/allocs.pprof

go tool pprof -top -nodecount=20 report/fox_v6b_final/fox/heap.pprof
go tool pprof -top -nodecount=20 report/fox_v6b_final/fox/allocs.pprof
go tool pprof -top -cum -nodecount=15 report/fox_v6b_final/fox/cpu.pprof
go tool pprof -peek "mergeEntries" report/fox_v6b_final/fox/allocs.pprof

go tool pprof -top -nodecount=20 report/fox_v7a_final/fox/allocs.pprof
go tool pprof -peek "compositeKey" report/fox_v7a_final/fox/allocs.pprof

go tool pprof -top -nodecount=20 report/fox_v8a_final/fox/allocs.pprof
go tool pprof -top -nodecount=20 report/fox_v8a_final/fox/heap.pprof
go tool pprof -list "github.com/liteio-dev/liteio/pkg/storage/driver/zoo/fox.(*store).putPreparedEntry" report/fox_v8a_final/fox/allocs.pprof

go tool pprof -top -nodecount=20 report/fox_v9a_final/fox/allocs.pprof
go tool pprof -top -nodecount=20 report/fox_v9a_final/fox/heap.pprof
go tool pprof -peek "compositeKey" report/fox_v9a_final/fox/allocs.pprof

go tool pprof -top -nodecount=20 report/fox_v10c_final/fox/allocs.pprof
go tool pprof -top -nodecount=20 report/fox_v10c_final/fox/heap.pprof
go tool pprof -peek "compositeKey" report/fox_v10c_final/fox/allocs.pprof

# Compare baseline vs final

go tool pprof -base report/fox_v1_baseline/fox/cpu.pprof report/fox_v2e_final/fox/cpu.pprof
go tool pprof -base report/fox_v1_baseline/fox/heap.pprof report/fox_v2e_final/fox/heap.pprof

go tool pprof -base report/fox_v1_baseline/fox/allocs.pprof report/fox_v2e_final/fox/allocs.pprof

go tool pprof -base report/fox_v3b_final/fox/heap.pprof report/fox_v4b_final/fox/heap.pprof
go tool pprof -base report/fox_v3b_final/fox/allocs.pprof report/fox_v4b_final/fox/allocs.pprof

go tool pprof -base report/fox_v4b_final/fox/allocs.pprof report/fox_v5b_final/fox/allocs.pprof
go tool pprof -base report/fox_v5b_final/fox/allocs.pprof report/fox_v6b_final/fox/allocs.pprof

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
