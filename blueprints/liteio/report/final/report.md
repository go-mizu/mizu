# Storage Benchmark Report

## System Info

| Property | Value |
|----------|-------|
| Timestamp | 2026-02-21T08:21:37+07:00 |
| Go Version | go1.26.0 |
| Platform | darwin/arm64 |
| CPUs | 10 |
| BenchTime | 2s |
| Concurrency | 200 |
| Levels | [1 10 25 50 100 200] |
| Object Sizes | 1KB, 64KB, 1MB, 10MB, 100MB |
| Warmup | 10 iterations |
| Timeout | 30s |
| Drivers | 1 |

## Table of Contents

1. [Executive Summary Dashboard](#executive-summary-dashboard)
2. [Performance Matrix](#performance-matrix)
3. [Write Performance Deep Dive](#write-performance-deep-dive)
4. [Read Performance Deep Dive](#read-performance-deep-dive)
5. [Parallel Scalability](#parallel-scalability)
6. [Latency Analysis](#latency-analysis)
7. [Resource Efficiency](#resource-efficiency)
8. [Error & Timeout Summary](#error--timeout-summary)
9. [Recommendations](#recommendations)

## Executive Summary Dashboard

| # | Driver | Write | Read | Wins | Memory | Disk | Status |
|---|--------|-------|------|------|--------|------|--------|
| 1 | **herd** |       |       | 0/0 | 9.8 GB | 64.0 GB | OK |

> **Overall Leader: herd** -- won 0/0 benchmarks with combined write+read score of 0%.

## Performance Matrix

All drivers x key operations. Values show throughput (ops/s or MB/s). **Bold** = best in column.

| Driver | W/1KB | W/64KB | W/1MB | W/10MB | W/100MB | R/1KB | R/64KB | R/1MB | R/10MB | R/100MB | Stat | Delete | Copy/1KB | List/100 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| herd | **1.1 GB/s** | **10.7 GB/s** | **1.4 GB/s** | **426.0 MB/s** | **1.1 GB/s** | **4.5 GB/s** | **49.4 GB/s** | **57.9 GB/s** | **11.0 GB/s** | **10.8 GB/s** | **8.8M/s** | **2.8M/s** | **515.2 MB/s** | **26.8K/s** |

## Write Performance Deep Dive

### Write Throughput & Latency

| Driver | 1KB ops/s | 1KB MB/s | 1KB P50 | 1KB P99 | 64KB ops/s | 64KB MB/s | 64KB P50 | 64KB P99 | 1MB ops/s | 1MB MB/s | 1MB P50 | 1MB P99 | 10MB ops/s | 10MB MB/s | 10MB P50 | 10MB P99 | 100MB ops/s | 100MB MB/s | 100MB P50 | 100MB P99 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| herd | 1.1M/s | 1.1 GB/s | 625ns | 2.0us | 170.7K/s | 10.7 GB/s | 3.1us | 42.9us | 1.4K/s | 1.4 GB/s | 46.5us | 15.6ms | 43/s | 426.0 MB/s | 3.6ms | 93.0ms | 11/s | 1.1 GB/s | 39.6ms | 353.6ms |

## Read Performance Deep Dive

### Read Throughput & Latency

| Driver | 1KB ops/s | 1KB MB/s | 1KB P50 | 1KB P99 | 64KB ops/s | 64KB MB/s | 64KB P50 | 64KB P99 | 1MB ops/s | 1MB MB/s | 1MB P50 | 1MB P99 | 10MB ops/s | 10MB MB/s | 10MB P50 | 10MB P99 | 100MB ops/s | 100MB MB/s | 100MB P50 | 100MB P99 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| herd | 4.6M/s | 4.5 GB/s | 167ns | 625ns | 790.1K/s | 49.4 GB/s | 1.2us | 1.6us | 57.9K/s | 57.9 GB/s | 17.0us | 21.2us | 1.1K/s | 11.0 GB/s | 901.6us | 1.0ms | 108/s | 10.8 GB/s | 9.1ms | 10.6ms |

## Parallel Scalability

### Parallel Write Scalability

| Driver | C1 | C10 | C25 | C50 | C100 | C200 | Scaling |
|--------|--------|--------|--------|--------|--------|--------|---------|
| herd | 677.9 MB/s | 408.1 MB/s | 165.8 MB/s | 25.2 MB/s | 17.3 MB/s | 16.6 MB/s | 0% |

> Scaling = (throughput at C200 / throughput at C1) / (200/1). 100% = perfect linear scaling.

### Parallel Read Scalability

| Driver | C1 | C10 | C25 | C50 | C100 | C200 | Scaling |
|--------|--------|--------|--------|--------|--------|--------|---------|
| herd | 2.9 GB/s | 2.6 GB/s | 2.3 GB/s | 1.8 GB/s | 1.7 GB/s | 1.5 GB/s | 0% |

## Latency Analysis

*No latency data available.*

## Resource Efficiency

### Runtime Resources

| Driver | Peak RSS | Go Heap | Disk Used | GC Cycles | Total Ops | Efficiency |
|--------|----------|---------|-----------|-----------|-----------|------------|
| herd | 9.8 GB | 36.4 GB | 64.0 GB | 21 | 109.6M/s | 10.9K ops/MB |

> Peak RSS = process-level resident memory. Go Heap = Go runtime heap. Efficiency = total iterations / peak RSS.

## Profiling

### Profile: herd

**Runtime Memory Stats:**

| Metric | Value |
|--------|-------|
| Heap In Use | 33307.1 MB |
| Heap Alloc | 33146.9 MB |
| Heap Sys | 57706.4 MB |
| Heap Objects | 80225846 |
| Total Alloc | 127626.9 MB |
| Stack In Use | 1.6 MB |
| GC Cycles | 21 |
| GC Pause Total | 9.4 ms |
| Goroutines | 26 |

**CPU Profile (top consumers):**

```
File: bench
Type: cpu
Time: 2026-02-21 08:19:02 +07
Duration: 144.80s, Total samples = 333.80s (230.52%)
Showing nodes accounting for 243.33s, 72.90% of 333.80s total
Dropped 512 nodes (cum <= 1.67s)
Showing top 10 nodes out of 178
      flat  flat%   sum%        cum   cum%
    68.68s 20.58% 20.58%     68.68s 20.58%  runtime.usleep
    42.26s 12.66% 33.24%     42.31s 12.68%  runtime.pthread_cond_wait
    31.60s  9.47% 42.70%     31.60s  9.47%  runtime.pthread_cond_signal
    30.79s  9.22% 51.93%     30.79s  9.22%  runtime.memclrNoHeapPointers
    22.73s  6.81% 58.74%     22.73s  6.81%  runtime.memmove
    11.86s  3.55% 62.29%     11.86s  3.55%  runtime.madvise
    11.62s  3.48% 65.77%     13.91s  4.17%  runtime.tryDeferToSpanScan
     8.70s  2.61% 68.38%      9.89s  2.96%  runtime.(*mspan).moveInlineMarks
     7.95s  2.38% 70.76%      7.95s  2.38%  syscall.rawsyscalln
     7.14s  2.14% 72.90%      8.29s  2.48%  runtime.mapaccess2_faststr
```

**Heap Profile (top consumers):**

```
File: bench
Type: inuse_space
Time: 2026-02-21 08:21:33 +07
Showing nodes accounting for 33012.88MB, 99.49% of 33182.65MB total
Dropped 46 nodes (cum <= 165.91MB)
Showing top 10 nodes out of 35
      flat  flat%   sum%        cum   cum%
   24576MB 74.06% 74.06%    24576MB 74.06%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*slabArena).alloc
 2070.16MB  6.24% 80.30%  2070.16MB  6.24%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.init.func1
 1688.57MB  5.09% 85.39%  3014.17MB  9.08%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*shardedIndex).put
 1325.61MB  3.99% 89.39%  1325.61MB  3.99%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.compositeKey (inline)
    1024MB  3.09% 92.47%     1024MB  3.09%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.newWriteBuffer (inline)
    1024MB  3.09% 95.56%     1024MB  3.09%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.newSlabArena (inline)
  855.57MB  2.58% 98.14%   855.57MB  2.58%  fmt.Sprintf
  448.99MB  1.35% 99.49%   448.99MB  1.35%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*shardedIndex).list
         0     0% 99.49%  2082.85MB  6.28%  github.com/liteio-dev/liteio/bench.(*Runner).Run
         0     0% 99.49%  1173.51MB  3.54%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkCopy
```

**Allocs Profile (top consumers):**

```
File: bench
Type: alloc_space
Time: 2026-02-21 08:21:33 +07
Showing nodes accounting for 107.93GB, 86.29% of 125.07GB total
Dropped 80 nodes (cum <= 0.63GB)
Showing top 10 nodes out of 62
      flat  flat%   sum%        cum   cum%
   65.88GB 52.67% 52.67%    65.88GB 52.67%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*slabArena).alloc
   11.39GB  9.11% 61.78%    12.90GB 10.31%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*shardedIndex).put
    7.36GB  5.89% 67.66%     7.36GB  5.89%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).Open
    4.93GB  3.94% 71.60%    84.66GB 67.69%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).Write
    4.62GB  3.69% 75.29%     4.62GB  3.69%  github.com/liteio-dev/liteio/bench.(*Collector).RecordWithError
    4.15GB  3.32% 78.61%     4.15GB  3.32%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).Stat
    3.37GB  2.69% 81.30%     3.84GB  3.07%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkParallelRead
    2.13GB  1.70% 83.00%     2.13GB  1.70%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.init.func1
    2.11GB  1.69% 84.69%     2.53GB  2.03%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*shardedIndex).list
       2GB  1.60% 86.29%        2GB  1.60%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.newWriteBuffer
```

**Profile Files:**

```bash
go tool pprof -http=:8080 report/final/herd/cpu.pprof  # CPU
go tool pprof -http=:8080 report/final/herd/heap.pprof  # Heap
go tool pprof -http=:8080 report/final/herd/allocs.pprof  # Allocs
```

## Error & Timeout Summary

No errors or timeouts recorded. All benchmarks completed successfully.

## Recommendations

### Best for Write-Heavy Workloads

> **herd** -- highest average write throughput across all object sizes.

| Rank | Driver | Avg Write Throughput |
|------|--------|---------------------|
| 1 | herd | 2.9 GB/s |

### Best for Read-Heavy Workloads

> **herd** -- highest average read throughput across all object sizes.

| Rank | Driver | Avg Read Throughput |
|------|--------|--------------------|
| 1 | herd | 26.7 GB/s |

### Most Memory Efficient

> **herd** -- lowest peak RSS at 9.8 GB.

### Best Overall


---

*Generated by storage benchmark CLI*
