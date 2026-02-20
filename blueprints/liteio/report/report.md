# Storage Benchmark Report

**Generated:** 2026-02-20T18:55:44+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | owl | 3.5 GB/s |  |
| Small Write (1KB) | owl | 397.9 MB/s |  |
| Large Read (10MB) | owl | 46.9 GB/s |  |
| Large Write (10MB) | owl | 4.6 GB/s |  |
| Delete | owl | 2.4M ops/s |  |
| Stat | owl | 8.3M ops/s |  |
| List (100 objects) | owl | 419 ops/s |  |
| Copy | owl | 975.0 MB/s |  |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **owl** | 4622 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **owl** | 46901 MB/s | Best for streaming, CDN |
| Small File Operations | **owl** | 1990806 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **owl** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| owl | 4621.8 | 46901.4 | 1.9ms | 207.4us |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| owl | 407463 | 3574150 | 583ns | 166ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| owl | 8339582 | 419 | 2378761 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| owl | 416.43 | 24.51 | 10.73 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| owl | 3495.56 | 1808.40 | 1544.61 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| owl | 10.5us | 25.9us | 134.3us | 1.1ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| owl | 304.7ms | 172.3ms | 145.3ms | 119.9ms |

*\* indicates errors occurred*

---

## Configuration

| Parameter | Value |
|-----------|-------|
| BenchTime | 500ms |
| MinIterations | 3 |
| Warmup | 5 |
| Concurrency | 200 |
| Timeout | 1m0s |

## Drivers Tested

- **owl** (40 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 974.95 MB/s | 584ns | 2.4us | 4.3us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 974.95 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 584ns
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 2378761 ops/s | 209ns | 500ns | 1.4us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 2378761 ops/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 209ns
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 96.59 MB/s | 625ns | 2.4us | 4.2us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 96.59 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 625ns
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 643111 ops/s | 584ns | 2.2us | 14.3us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 643111 ops/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 584ns
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 81.63 MB/s | 833ns | 2.5us | 4.2us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 81.63 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 833ns
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 419 ops/s | 2.7ms | 6.5ms | 8.8ms | 0 |

**Throughput**
```
owl          ██████████████████████████████ 419 ops/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 2.7ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 46.61 MB/s | 11.0us | 1.4ms | 2.2ms | 0 |

**Throughput**
```
owl          ██████████████████████████████ 46.61 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 11.0us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 264.75 MB/s | 1.2us | 263.5us | 1.1ms | 0 |

**Throughput**
```
owl          ██████████████████████████████ 264.75 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 1.2us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 23.06 MB/s | 40.8us | 2.4ms | 3.7ms | 0 |

**Throughput**
```
owl          ██████████████████████████████ 23.06 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 40.8us
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 549.47 MB/s | 27.1ms | 30.9ms | 31.0ms | 0 |

**Throughput**
```
owl          ██████████████████████████████ 549.47 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 27.1ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 3495.56 MB/s | 208ns | 375ns | 1.9us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 3495.56 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 208ns
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 1808.40 MB/s | 292ns | 1.3us | 3.6us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 1808.40 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 292ns
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 1544.61 MB/s | 292ns | 1.5us | 6.6us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 1544.61 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 292ns
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 416.43 MB/s | 708ns | 3.3us | 18.4us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 416.43 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 708ns
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 24.51 MB/s | 3.1us | 219.8us | 541.0us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 24.51 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 3.1us
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 10.73 MB/s | 10.5us | 349.4us | 564.9us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 10.73 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 10.5us
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 57943.28 MB/s | 4.2us | 5.2us | 6.6us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 57943.28 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 4.2us
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 54500.49 MB/s | 4.5us | 4.9us | 6.8us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 54500.49 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 4.5us
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 48273.06 MB/s | 5.0us | 6.0us | 7.5us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 48273.06 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 5.0us
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 46901.41 MB/s | 207.4us | 259.5us | 287.1us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 46901.41 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 207.4us
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 3490.38 MB/s | 166ns | 209ns | 1.6us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 3490.38 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 166ns
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 52489.81 MB/s | 17.2us | 21.0us | 25.7us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 52489.81 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 17.2us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 36042.87 MB/s | 1.2us | 1.4us | 3.0us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 36042.87 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 1.2us
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 80808 ops/s | 12.4us | 12.4us | 12.4us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 80808 ops/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 12.4us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 84211 ops/s | 11.9us | 11.9us | 11.9us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 84211 ops/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 11.9us
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 18547 ops/s | 53.9us | 53.9us | 53.9us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 18547 ops/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 53.9us
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 2693 ops/s | 371.3us | 371.3us | 371.3us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 2693 ops/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 371.3us
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 3 ops/s | 304.7ms | 304.7ms | 304.7ms | 0 |

**Throughput**
```
owl          ██████████████████████████████ 3 ops/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 304.7ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 6 ops/s | 172.3ms | 172.3ms | 172.3ms | 0 |

**Throughput**
```
owl          ██████████████████████████████ 6 ops/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 172.3ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 7 ops/s | 145.3ms | 145.3ms | 145.3ms | 0 |

**Throughput**
```
owl          ██████████████████████████████ 7 ops/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 145.3ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 8 ops/s | 119.9ms | 119.9ms | 119.9ms | 0 |

**Throughput**
```
owl          ██████████████████████████████ 8 ops/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 119.9ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 23.16 MB/s | 10.5us | 10.5us | 10.5us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 23.16 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 10.5us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 94.35 MB/s | 25.9us | 25.9us | 25.9us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 94.35 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 25.9us
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 181.74 MB/s | 134.3us | 134.3us | 134.3us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 181.74 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 134.3us
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 227.20 MB/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
owl          ██████████████████████████████ 227.20 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 1.1ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 8339582 ops/s | 42ns | 84ns | 917ns | 0 |

**Throughput**
```
owl          ██████████████████████████████ 8339582 ops/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 42ns
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 4621.81 MB/s | 1.9ms | 4.3ms | 4.8ms | 0 |

**Throughput**
```
owl          ██████████████████████████████ 4621.81 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 1.9ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 397.91 MB/s | 583ns | 3.4us | 18.2us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 397.91 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 583ns
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 1144.06 MB/s | 275.0us | 2.5ms | 4.4ms | 0 |

**Throughput**
```
owl          ██████████████████████████████ 1144.06 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 275.0us
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| owl | 2707.59 MB/s | 9.5us | 42.2us | 342.6us | 0 |

**Throughput**
```
owl          ██████████████████████████████ 2707.59 MB/s
```

**Latency (P50)**
```
owl          ██████████████████████████████ 9.5us
```

## Runtime Resource Usage

| Driver | Peak RSS | Go Heap | Go Sys | Disk Usage | GC Cycles |
|--------|----------|---------|--------|------------|----------|
| owl | 9108.5 MB | 53270.2 MB | 95700.8 MB | 12970.4 MB | 8 |

> **Note:** Peak RSS = process peak resident set size (includes mmap). Go Heap/Sys = Go runtime allocations. Disk = data directory size after benchmark.

## Recommendations

- **Write-heavy workloads:** owl
- **Read-heavy workloads:** owl

---

*Generated by storage benchmark CLI*
