# Storage Benchmark Report

## System Info

| Property | Value |
|----------|-------|
| Timestamp | 2026-02-21T08:10:40+07:00 |
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
| 1 | **herd** |       |       | 0/0 | 8.7 GB | 56.0 GB | OK |

> **Overall Leader: herd** -- won 0/0 benchmarks with combined write+read score of 0%.

## Performance Matrix

All drivers x key operations. Values show throughput (ops/s or MB/s). **Bold** = best in column.

| Driver | W/1KB | W/64KB | W/1MB | W/10MB | W/100MB | R/1KB | R/64KB | R/1MB | R/10MB | R/100MB | Stat | Delete | Copy/1KB | List/100 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| herd | **1.4 GB/s** | **5.6 GB/s** | **980.6 MB/s** | **338.6 MB/s** | **92.9 MB/s** | **4.6 GB/s** | **48.6 GB/s** | **56.0 GB/s** | **10.5 GB/s** | **10.3 GB/s** | **8.1M/s** | **4.3M/s** | **555.9 MB/s** | **8/s** |

## Write Performance Deep Dive

### Write Throughput & Latency

| Driver | 1KB ops/s | 1KB MB/s | 1KB P50 | 1KB P99 | 64KB ops/s | 64KB MB/s | 64KB P50 | 64KB P99 | 1MB ops/s | 1MB MB/s | 1MB P50 | 1MB P99 | 10MB ops/s | 10MB MB/s | 10MB P50 | 10MB P99 | 100MB ops/s | 100MB MB/s | 100MB P50 | 100MB P99 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| herd | 1.4M/s | 1.4 GB/s | 500ns | 2.1us | 89.1K/s | 5.6 GB/s | 2.5us | 114.8us | 981/s | 980.6 MB/s | 24.3us | 16.3ms | 34/s | 338.6 MB/s | 2.2ms | 177.2ms | 1/s | 92.9 MB/s | 209.0ms | 1.25s |

## Read Performance Deep Dive

### Read Throughput & Latency

| Driver | 1KB ops/s | 1KB MB/s | 1KB P50 | 1KB P99 | 64KB ops/s | 64KB MB/s | 64KB P50 | 64KB P99 | 1MB ops/s | 1MB MB/s | 1MB P50 | 1MB P99 | 10MB ops/s | 10MB MB/s | 10MB P50 | 10MB P99 | 100MB ops/s | 100MB MB/s | 100MB P50 | 100MB P99 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| herd | 4.7M/s | 4.6 GB/s | 167ns | 583ns | 777.1K/s | 48.6 GB/s | 1.2us | 2.4us | 56.0K/s | 56.0 GB/s | 17.5us | 21.7us | 1.1K/s | 10.5 GB/s | 940.0us | 1.1ms | 103/s | 10.3 GB/s | 9.6ms | 12.5ms |

## Parallel Scalability

### Parallel Write Scalability

| Driver | C1 | C10 | C25 | C50 | C100 | C200 | Scaling |
|--------|--------|--------|--------|--------|--------|--------|---------|
| herd | 843.7 MB/s | 287.4 MB/s | 122.0 MB/s | 51.0 MB/s | 9.8 MB/s | 24.1 MB/s | 0% |

> Scaling = (throughput at C200 / throughput at C1) / (200/1). 100% = perfect linear scaling.

### Parallel Read Scalability

| Driver | C1 | C10 | C25 | C50 | C100 | C200 | Scaling |
|--------|--------|--------|--------|--------|--------|--------|---------|
| herd | 3.0 GB/s | 2.5 GB/s | 2.0 GB/s | 1.9 GB/s | 1.8 GB/s | 2.1 GB/s | 0% |

## Latency Analysis

*No latency data available.*

## Resource Efficiency

### Runtime Resources

| Driver | Peak RSS | Go Heap | Disk Used | GC Cycles | Total Ops | Efficiency |
|--------|----------|---------|-----------|-----------|-----------|------------|
| herd | 8.7 GB | 47.4 GB | 56.0 GB | 23 | 116.2M/s | 13.1K ops/MB |

> Peak RSS = process-level resident memory. Go Heap = Go runtime heap. Efficiency = total iterations / peak RSS.

## Profiling

### Profile: herd

**Runtime Memory Stats:**

| Metric | Value |
|--------|-------|
| Heap In Use | 40267.6 MB |
| Heap Alloc | 39885.7 MB |
| Heap Sys | 56190.1 MB |
| Heap Objects | 61044654 |
| Total Alloc | 150029.6 MB |
| Stack In Use | 1.9 MB |
| GC Cycles | 23 |
| GC Pause Total | 8.3 ms |
| Goroutines | 26 |

**CPU Profile (top consumers):**

```
File: bench
Type: cpu
Time: 2026-02-21 08:08:02 +07
Duration: 144.32s, Total samples = 318.58s (220.74%)
Showing nodes accounting for 241.33s, 75.75% of 318.58s total
Dropped 487 nodes (cum <= 1.59s)
Showing top 10 nodes out of 177
      flat  flat%   sum%        cum   cum%
    54.42s 17.08% 17.08%     54.42s 17.08%  runtime.usleep
    40.98s 12.86% 29.95%     40.99s 12.87%  runtime.pthread_cond_wait
    31.81s  9.98% 39.93%     31.81s  9.98%  runtime.memclrNoHeapPointers
    30.77s  9.66% 49.59%     30.77s  9.66%  runtime.memmove
    30.07s  9.44% 59.03%     30.10s  9.45%  runtime.pthread_cond_signal
    15.08s  4.73% 63.76%     17.72s  5.56%  runtime.tryDeferToSpanScan
    11.30s  3.55% 67.31%     13.50s  4.24%  runtime.mapaccess2_faststr
    10.96s  3.44% 70.75%     13.21s  4.15%  runtime.(*mspan).moveInlineMarks
     8.70s  2.73% 73.48%      8.78s  2.76%  runtime.(*spanInlineMarkBits).init
     7.24s  2.27% 75.75%      7.24s  2.27%  runtime.madvise
```

**Heap Profile (top consumers):**

```
File: bench
Type: inuse_space
Time: 2026-02-21 08:10:32 +07
Showing nodes accounting for 39537.94MB, 99.32% of 39809.19MB total
Dropped 47 nodes (cum <= 199.05MB)
Showing top 10 nodes out of 29
      flat  flat%   sum%        cum   cum%
   32768MB 82.31% 82.31%    32768MB 82.31%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*slabArena).alloc
 2278.67MB  5.72% 88.04%  2278.67MB  5.72%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.init.func1
 1575.67MB  3.96% 91.99%  2955.26MB  7.42%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*shardedIndex).put
 1379.60MB  3.47% 95.46%  1379.60MB  3.47%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.compositeKey (inline)
    1024MB  2.57% 98.03%     1024MB  2.57%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.newSlabArena (inline)
     512MB  1.29% 99.32%      512MB  1.29%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.newWriteBuffer (inline)
         0     0% 99.32%  1570.84MB  3.95%  github.com/liteio-dev/liteio/bench.(*Runner).Run
         0     0% 99.32%  3374.02MB  8.48%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkCopy
         0     0% 99.32% 10635.56MB 26.72%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkDelete
         0     0% 99.32%  1570.84MB  3.95%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkDriverWithTracker
```

**Allocs Profile (top consumers):**

```
File: bench
Type: alloc_space
Time: 2026-02-21 08:10:32 +07
Showing nodes accounting for 133.60GB, 91.31% of 146.32GB total
Dropped 85 nodes (cum <= 0.73GB)
Showing top 10 nodes out of 54
      flat  flat%   sum%        cum   cum%
   89.06GB 60.87% 60.87%    89.06GB 60.87%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*slabArena).alloc
   12.88GB  8.80% 69.67%    14.54GB  9.94%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*shardedIndex).put
    7.43GB  5.08% 74.75%     7.43GB  5.08%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).Open
    5.71GB  3.90% 78.65%   108.09GB 73.87%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).Write
    4.59GB  3.14% 81.79%     4.59GB  3.14%  github.com/liteio-dev/liteio/bench.(*Collector).RecordWithError
    3.74GB  2.56% 84.35%     3.74GB  2.56%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).Stat
    3.68GB  2.52% 86.86%     4.22GB  2.88%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkParallelRead
    2.52GB  1.72% 88.59%     2.52GB  1.72%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.init.func1
       2GB  1.37% 89.95%        2GB  1.37%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.newSlabArena
    1.98GB  1.35% 91.31%     2.23GB  1.53%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkParallelWrite
```

**Profile Files:**

```bash
go tool pprof -http=:8080 report/optimized/herd/cpu.pprof  # CPU
go tool pprof -http=:8080 report/optimized/herd/heap.pprof  # Heap
go tool pprof -http=:8080 report/optimized/herd/allocs.pprof  # Allocs
```

## Error & Timeout Summary

No errors or timeouts recorded. All benchmarks completed successfully.

## Recommendations

### Best for Write-Heavy Workloads

> **herd** -- highest average write throughput across all object sizes.

| Rank | Driver | Avg Write Throughput |
|------|--------|---------------------|
| 1 | herd | 1.7 GB/s |

### Best for Read-Heavy Workloads

> **herd** -- highest average read throughput across all object sizes.

| Rank | Driver | Avg Read Throughput |
|------|--------|--------------------|
| 1 | herd | 26.0 GB/s |

### Most Memory Efficient

> **herd** -- lowest peak RSS at 8.7 GB.

### Best Overall


---

*Generated by storage benchmark CLI*
