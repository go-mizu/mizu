# Storage Benchmark Report

## System Info

| Property | Value |
|----------|-------|
| Timestamp | 2026-02-21T08:00:01+07:00 |
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
| 1 | **herd** |       |       | 0/0 | 6.7 GB | 16.0 GB | OK |

> **Overall Leader: herd** -- won 0/0 benchmarks with combined write+read score of 0%.

## Performance Matrix

All drivers x key operations. Values show throughput (ops/s or MB/s). **Bold** = best in column.

| Driver | W/1KB | W/64KB | W/1MB | W/10MB | W/100MB | R/1KB | R/64KB | R/1MB | R/10MB | R/100MB | Stat | Delete | List/100 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| herd | **1.1 GB/s** | **4.0 GB/s** | **962.3 MB/s** | **218.7 MB/s** | **321.3 MB/s** | **5.0 GB/s** | **49.1 GB/s** | **55.5 GB/s** | **10.0 GB/s** | **9.9 GB/s** | **9.0M/s** | **2.3M/s** | **42.5K/s** |

## Write Performance Deep Dive

### Write Throughput & Latency

| Driver | 1KB ops/s | 1KB MB/s | 1KB P50 | 1KB P99 | 64KB ops/s | 64KB MB/s | 64KB P50 | 64KB P99 | 1MB ops/s | 1MB MB/s | 1MB P50 | 1MB P99 | 10MB ops/s | 10MB MB/s | 10MB P50 | 10MB P99 | 100MB ops/s | 100MB MB/s | 100MB P50 | 100MB P99 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| herd | 1.1M/s | 1.1 GB/s | 542ns | 3.4us | 63.9K/s | 4.0 GB/s | 2.9us | 197.1us | 962/s | 962.3 MB/s | 32.0us | 15.9ms | 22/s | 218.7 MB/s | 66.3ms | 131.4ms | 3/s | 321.3 MB/s | 308.0ms | 481.5ms |

## Read Performance Deep Dive

### Read Throughput & Latency

| Driver | 1KB ops/s | 1KB MB/s | 1KB P50 | 1KB P99 | 64KB ops/s | 64KB MB/s | 64KB P50 | 64KB P99 | 1MB ops/s | 1MB MB/s | 1MB P50 | 1MB P99 | 10MB ops/s | 10MB MB/s | 10MB P50 | 10MB P99 | 100MB ops/s | 100MB MB/s | 100MB P50 | 100MB P99 |
|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|
| herd | 5.1M/s | 5.0 GB/s | 125ns | 1.6us | 786.2K/s | 49.1 GB/s | 1.2us | 2.5us | 55.5K/s | 55.5 GB/s | 17.3us | 30.3us | 997/s | 10.0 GB/s | 990.0us | 1.1ms | 99/s | 9.9 GB/s | 9.9ms | 11.6ms |

## Parallel Scalability

### Parallel Write Scalability

| Driver | C1 | C10 | C25 | C50 | C100 | Scaling |
|--------|--------|--------|--------|--------|--------|---------|
| herd | 609.5 MB/s | 227.6 MB/s | 99.7 MB/s | 97.4 MB/s | 9.7 MB/s | 0% |

> Scaling = (throughput at C100 / throughput at C1) / (100/1). 100% = perfect linear scaling.

### Parallel Read Scalability

| Driver | C1 | C10 | C25 | C50 | C100 | Scaling |
|--------|--------|--------|--------|--------|--------|---------|
| herd | 3.6 GB/s | 1.6 GB/s | 1.3 GB/s | 2.7 GB/s | - | - |

## Latency Analysis

*No latency data available.*

## Resource Efficiency

### Runtime Resources

| Driver | Peak RSS | Go Heap | Disk Used | GC Cycles | Total Ops | Efficiency |
|--------|----------|---------|-----------|-----------|-----------|------------|
| herd | 6.7 GB | 30.3 GB | 16.0 GB | 24 | 81.0M/s | 11.8K ops/MB |

> Peak RSS = process-level resident memory. Go Heap = Go runtime heap. Efficiency = total iterations / peak RSS.

## Profiling

### Profile: herd

**Runtime Memory Stats:**

| Metric | Value |
|--------|-------|
| Heap In Use | 24028.6 MB |
| Heap Alloc | 23718.0 MB |
| Heap Sys | 31454.7 MB |
| Heap Objects | 75649988 |
| Total Alloc | 65238.1 MB |
| Stack In Use | 1.3 MB |
| GC Cycles | 24 |
| GC Pause Total | 7.8 ms |
| Goroutines | 26 |

**CPU Profile (top consumers):**

```
File: bench
Type: cpu
Time: 2026-02-21 07:57:46 +07
Duration: 102.09s, Total samples = 275.45s (269.81%)
Showing nodes accounting for 213.72s, 77.59% of 275.45s total
Dropped 419 nodes (cum <= 1.38s)
Showing top 10 nodes out of 154
      flat  flat%   sum%        cum   cum%
    75.57s 27.44% 27.44%     77.05s 27.97%  runtime.tryDeferToSpanScan
    24.57s  8.92% 36.36%     24.57s  8.92%  runtime.memclrNoHeapPointers
    20.94s  7.60% 43.96%     20.94s  7.60%  runtime.madvise
    18.20s  6.61% 50.56%     20.55s  7.46%  runtime.mapaccess2_faststr
    15.88s  5.77% 56.33%     15.88s  5.77%  runtime.usleep
    14.01s  5.09% 61.42%     14.02s  5.09%  runtime.pthread_cond_wait
    13.55s  4.92% 66.34%     23.03s  8.36%  runtime.mapassign_faststr
    11.39s  4.14% 70.47%     11.39s  4.14%  runtime.pthread_cond_signal
     9.82s  3.57% 74.04%     11.02s  4.00%  runtime.(*mspan).moveInlineMarks
     9.79s  3.55% 77.59%      9.79s  3.55%  internal/runtime/atomic.(*Uint8).Load
```

**Heap Profile (top consumers):**

```
File: bench
Type: inuse_space
Time: 2026-02-21 07:59:57 +07
Showing nodes accounting for 23549.01MB, 99.35% of 23703.33MB total
Dropped 41 nodes (cum <= 118.52MB)
Showing top 10 nodes out of 24
      flat  flat%   sum%        cum   cum%
18435.99MB 77.78% 77.78% 22645.50MB 95.54%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).Write
 1993.87MB  8.41% 86.19%  2779.40MB 11.73%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*shardedIndex).put
 1430.11MB  6.03% 92.22%  1430.11MB  6.03%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.init.func1
  785.53MB  3.31% 95.54%   785.53MB  3.31%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.compositeKey (inline)
     512MB  2.16% 97.70%      512MB  2.16%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.newWriteBuffer (inline)
  391.51MB  1.65% 99.35%   391.51MB  1.65%  fmt.Sprintf
         0     0% 99.35%   541.53MB  2.28%  github.com/liteio-dev/liteio/bench.(*Runner).Run
         0     0% 99.35%   618.92MB  2.61%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkDelete
         0     0% 99.35%   541.53MB  2.28%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkDriverWithTracker
         0     0% 99.35%  4401.62MB 18.57%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkDriverWithTracker.func1
```

**Allocs Profile (top consumers):**

```
File: bench
Type: alloc_space
Time: 2026-02-21 07:59:57 +07
Showing nodes accounting for 57.75GB, 90.53% of 63.79GB total
Dropped 67 nodes (cum <= 0.32GB)
Showing top 10 nodes out of 46
      flat  flat%   sum%        cum   cum%
   27.89GB 43.72% 43.72%    40.41GB 63.35%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).Write
    9.94GB 15.59% 59.31%    10.91GB 17.10%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*shardedIndex).put
    4.25GB  6.67% 65.98%     4.25GB  6.67%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).Open
    4.07GB  6.38% 72.36%     4.07GB  6.38%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).Stat
    3.37GB  5.29% 77.64%     3.37GB  5.29%  github.com/liteio-dev/liteio/bench.(*Collector).RecordWithError
    1.97GB  3.08% 80.72%     2.25GB  3.52%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkParallelRead
    1.93GB  3.02% 83.74%     2.45GB  3.84%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.(*bucket).List
    1.80GB  2.83% 86.57%     2.03GB  3.18%  github.com/liteio-dev/liteio/bench.(*Runner).benchmarkParallelWrite
    1.40GB  2.19% 88.76%     1.40GB  2.19%  github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd.init.func1
    1.13GB  1.77% 90.53%     1.13GB  1.77%  bytes.NewReader
```

**Profile Files:**

```bash
go tool pprof -http=:8080 report/baseline/herd/cpu.pprof  # CPU
go tool pprof -http=:8080 report/baseline/herd/heap.pprof  # Heap
go tool pprof -http=:8080 report/baseline/herd/allocs.pprof  # Allocs
```

## Error & Timeout Summary

No errors or timeouts recorded. All benchmarks completed successfully.

## Recommendations

### Best for Write-Heavy Workloads

> **herd** -- highest average write throughput across all object sizes.

| Rank | Driver | Avg Write Throughput |
|------|--------|---------------------|
| 1 | herd | 1.3 GB/s |

### Best for Read-Heavy Workloads

> **herd** -- highest average read throughput across all object sizes.

| Rank | Driver | Avg Read Throughput |
|------|--------|--------------------|
| 1 | herd | 25.9 GB/s |

### Most Memory Efficient

> **herd** -- lowest peak RSS at 6.7 GB.

### Best Overall


---

*Generated by storage benchmark CLI*
