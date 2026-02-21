# Storage Benchmark Report

## System Info

| Property | Value |
|----------|-------|
| Timestamp | 2026-02-21T08:17:00+07:00 |
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
| 1 | **herd** |       |       | 0/0 | 8.9 GB | 64.0 GB | OK |

> **Overall Leader: herd** -- won 0/0 benchmarks with combined write+read score of 0%.

## Performance Matrix

All drivers x key operations. Values show throughput (ops/s or MB/s). **Bold** = best in column.

| Driver | W/1KB | W/64KB | W/1MB | W/10MB | W/100MB | R/1KB | R/64KB | R/1MB | R/10MB | R/100MB | Stat | Delete | Copy/1KB | List/100 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| herd | **1.1 GB/s** | **6.8 GB/s** | **1.3 GB/s** | **165.5 MB/s** | **220.2 MB/s** | **4.3 GB/s** | **47.5 GB/s** | **56.3 GB/s** | **10.0 GB/s** | **10.4 GB/s** | **7.9M/s** | **2.8M/s** | **277.4 MB/s** | **13/s** |

## Write Performance Deep Dive

### Write Throughput & Latency

| Driver | 1KB ops/s | 1KB MB/s | 1KB P50 | 1KB P99 | 64KB ops/s | 64KB MB/s | 64KB P50 | 64KB P99 | 1MB ops/s | 1MB MB/s | 1MB P50 | 1MB P99 | 10MB ops/s | 10MB MB/s | 10MB P50 | 10MB P99 | 100MB ops/s | 100MB MB/s | 100MB P50 | 100MB P99 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| herd | 1.1M/s | 1.1 GB/s | 625ns | 2.1us | 109.5K/s | 6.8 GB/s | 3.8us | 105.0us | 1.3K/s | 1.3 GB/s | 43.3us | 14.9ms | 17/s | 165.5 MB/s | 67.2ms | 206.4ms | 2/s | 220.2 MB/s | 424.7ms | 611.2ms |

## Read Performance Deep Dive

### Read Throughput & Latency

| Driver | 1KB ops/s | 1KB MB/s | 1KB P50 | 1KB P99 | 64KB ops/s | 64KB MB/s | 64KB P50 | 64KB P99 | 1MB ops/s | 1MB MB/s | 1MB P50 | 1MB P99 | 10MB ops/s | 10MB MB/s | 10MB P50 | 10MB P99 | 100MB ops/s | 100MB MB/s | 100MB P50 | 100MB P99 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| herd | 4.4M/s | 4.3 GB/s | 208ns | 1.3us | 760.1K/s | 47.5 GB/s | 1.2us | 2.5us | 56.3K/s | 56.3 GB/s | 17.3us | 21.5us | 996/s | 10.0 GB/s | 970.7us | 2.1ms | 104/s | 10.4 GB/s | 9.5ms | 10.1ms |

## Parallel Scalability

### Parallel Write Scalability

| Driver | C1 | C10 | C25 | C50 | C100 | C200 | Scaling |
|--------|--------|--------|--------|--------|--------|--------|---------|
| herd | 746.9 MB/s | 332.4 MB/s | 97.2 MB/s | 27.3 MB/s | 54.3 MB/s | 16.2 MB/s | 0% |

> Scaling = (throughput at C200 / throughput at C1) / (200/1). 100% = perfect linear scaling.

### Parallel Read Scalability

| Driver | C1 | C10 | C25 | C50 | C100 | C200 | Scaling |
|--------|--------|--------|--------|--------|--------|--------|---------|
| herd | 2.8 GB/s | 2.7 GB/s | 2.1 GB/s | 1.9 GB/s | 2.0 GB/s | 1.9 GB/s | 0% |

## Latency Analysis

*No latency data available.*

## Resource Efficiency

### Runtime Resources

| Driver | Peak RSS | Go Heap | Disk Used | GC Cycles | Total Ops | Efficiency |
|--------|----------|---------|-----------|-----------|-----------|------------|
| herd | 8.9 GB | 37.1 GB | 64.0 GB | 19 | 108.5M/s | 11.9K ops/MB |

> Peak RSS = process-level resident memory. Go Heap = Go runtime heap. Efficiency = total iterations / peak RSS.

## Profiling

### Profile: herd

**Runtime Memory Stats:**

| Metric | Value |
|--------|-------|
| Heap In Use | 30687.1 MB |
| Heap Alloc | 30425.2 MB |
| Heap Sys | 44778.3 MB |
| Heap Objects | 73046246 |
| Total Alloc | 113418.6 MB |
| Stack In Use | 1.7 MB |
| GC Cycles | 19 |
| GC Pause Total | 5.3 ms |
| Goroutines | 26 |

**CPU Profile (top consumers):**

```
File: bench
Type: cpu
Time: 2026-02-21 08:14:39 +07
Duration: 130.25s, Total samples = 311.51s (239.16%)
Showing nodes accounting for 236.78s, 76.01% of 311.51s total
Dropped 466 nodes (cum <= 1.56s)
Showing top 10 nodes out of 174
      flat  flat%   sum%        cum   cum%
    66.91s 21.48% 21.48%     66.91s 21.48%  runtime.usleep
    42.19s 13.54% 35.02%     42.19s 13.54%  runtime.pthread_cond_wait
    31.10s  9.98% 45.01%     31.12s  9.99%  runtime.pthread_cond_signal
    24.55s  7.88% 52.89%     24.55s  7.88%  runtime.memclrNoHeapPointers
    22.88s  7.34% 60.23%     22.88s  7.34%  runtime.memmove
    15.93s  5.11% 65.35%     15.93s  5.11%  runtime.madvise
    10.07s  3.23% 68.58%     11.78s  3.78%  runtime.tryDeferToSpanScan
     7.88s  2.53% 71.11%      9.03s  2.90%  runtime.mapaccess2_faststr
     7.84s  2.52% 73.63%     14.90s  4.78%  runtime.mapassign_faststr
     7.43s  2.39% 76.01%      7.50s  2.41%  runtime.(*spanInlineMarkBits).init
```

**Heap Profile (top consumers):**

```
File: bench
Type: inuse_space
Time: 2026-02-21 08:16:56 +07
Showing nodes accounting for 30092.65MB, 99.10% of 30366.35MB total
Dropped 48 nodes (cum <= 151.83MB)
Showing top 10 nodes out of 32
      flat  flat%   sum%        cum   cum%
   22528MB 74.19% 74.19%    22528MB 74.19%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*slabArena).alloc
 1787.64MB  5.89% 80.07%  1787.64MB  5.89%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.init.func1
 1717.84MB  5.66% 85.73%  2917.44MB  9.61%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*shardedIndex).put
 1199.60MB  3.95% 89.68%  1199.60MB  3.95%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.compositeKey (inline)
    1024MB  3.37% 93.05%     1024MB  3.37%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.newWriteBuffer (inline)
    1024MB  3.37% 96.43%     1024MB  3.37%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.newSlabArena (inline)
  811.57MB  2.67% 99.10%   811.57MB  2.67%  fmt.Sprintf
         0     0% 99.10%  2081.97MB  6.86%  github.com/liteio-dev/liteio/bench.(*Runner).Run
         0     0% 99.10%  1349.12MB  4.44%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkCopy
         0     0% 99.10%  7601.03MB 25.03%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkDelete
```

**Allocs Profile (top consumers):**

```
File: bench
Type: alloc_space
Time: 2026-02-21 08:16:56 +07
Showing nodes accounting for 99.23GB, 89.62% of 110.72GB total
Dropped 71 nodes (cum <= 0.55GB)
Showing top 10 nodes out of 54
      flat  flat%   sum%        cum   cum%
   58.19GB 52.55% 52.55%    58.19GB 52.55%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*slabArena).alloc
   10.69GB  9.65% 62.21%    12.08GB 10.91%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*shardedIndex).put
    7.84GB  7.08% 69.29%     7.84GB  7.08%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).Open
    4.58GB  4.13% 73.42%    75.33GB 68.04%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).Write
    4.45GB  4.02% 77.44%     4.45GB  4.02%  github.com/liteio-dev/liteio/bench.(*Collector).RecordWithError
    3.79GB  3.42% 80.86%     4.33GB  3.91%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkParallelRead
    3.67GB  3.32% 84.17%     3.67GB  3.32%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).Stat
    2.03GB  1.84% 86.01%     2.04GB  1.85%  github.com/liteio-dev/liteio/bench.(*Runner).copyToDiscard
       2GB  1.81% 87.82%        2GB  1.81%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.newWriteBuffer
       2GB  1.81% 89.62%        2GB  1.81%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.newSlabArena
```

**Profile Files:**

```bash
go tool pprof -http=:8080 report/optimized_v2/herd/cpu.pprof  # CPU
go tool pprof -http=:8080 report/optimized_v2/herd/heap.pprof  # Heap
go tool pprof -http=:8080 report/optimized_v2/herd/allocs.pprof  # Allocs
```

## Error & Timeout Summary

No errors or timeouts recorded. All benchmarks completed successfully.

## Recommendations

### Best for Write-Heavy Workloads

> **herd** -- highest average write throughput across all object sizes.

| Rank | Driver | Avg Write Throughput |
|------|--------|---------------------|
| 1 | herd | 1.9 GB/s |

### Best for Read-Heavy Workloads

> **herd** -- highest average read throughput across all object sizes.

| Rank | Driver | Avg Read Throughput |
|------|--------|--------------------|
| 1 | herd | 25.7 GB/s |

### Most Memory Efficient

> **herd** -- lowest peak RSS at 8.9 GB.

### Best Overall


---

*Generated by storage benchmark CLI*
