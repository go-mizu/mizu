# Storage Benchmark Report

**Generated:** 2026-01-21T17:46:37+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** usagi_s3 (won 40/48 benchmarks, 83%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | usagi_s3 | 40 | 83% |
| 2 | minio | 8 | 17% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | usagi_s3 | 3.8 MB/s | +39% vs minio |
| Small Write (1KB) | usagi_s3 | 1.8 MB/s | +36% vs minio |
| Large Read (100MB) | minio | 187.9 MB/s | close |
| Large Write (100MB) | minio | 148.5 MB/s | +10% vs usagi_s3 |
| Delete | usagi_s3 | 3.9K ops/s | +65% vs minio |
| Stat | usagi_s3 | 3.7K ops/s | +20% vs minio |
| List (100 objects) | usagi_s3 | 886 ops/s | +79% vs minio |
| Copy | usagi_s3 | 1.2 MB/s | +44% vs minio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **minio** | 149 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 188 MB/s | Best for streaming, CDN |
| Small File Operations | **usagi_s3** | 2875 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **usagi_s3** | - | Best for multi-user apps |
| Memory Constrained | **minio** | 813 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| minio | 148.5 | 187.9 | 679.7ms | 528.0ms |
| usagi_s3 | 135.0 | 171.6 | 741.0ms | 590.4ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| minio | 1350 | 2816 | 705.9us | 328.3us |
| usagi_s3 | 1836 | 3914 | 505.5us | 237.0us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| minio | 3112 | 496 | 2337 |
| usagi_s3 | 3738 | 886 | 3855 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| minio | 1.03 | 0.22 | 0.14 | 0.05 | 0.02 | 0.01 |
| usagi_s3 | 1.17 | 0.36 | 0.12 | 0.06 | 0.03 | 0.02 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| minio | 2.33 | 0.68 | 0.35 | 0.19 | 0.10 | 0.05 |
| usagi_s3 | 3.22 | 1.02 | 0.52 | 0.31 | 0.18 | 0.08 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| minio | 11.0ms | 103.3ms | 1.12s | 10.87s |
| usagi_s3 | 7.6ms | 66.3ms | 735.9ms | 6.15s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| minio | 824.2us | 2.5ms | 15.7ms | 187.3ms |
| usagi_s3 | 465.9us | 1.0ms | 7.8ms | 228.6ms |

*\* indicates errors occurred*

### Skipped Benchmarks

Some benchmarks were skipped due to driver limitations:

- **minio**: 2 skipped
  - Scale/100000 (requires longer timeout)
  - Scale/1000000 (requires longer timeout)
- **usagi_s3**: 2 skipped
  - Scale/100000 (requires longer timeout)
  - Scale/1000000 (requires longer timeout)

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| minio | 812.6 MB | 3.5% |
| usagi_s3 | 1811.5 MB | 3.6% |

---

## Configuration

| Parameter | Value |
|-----------|-------|
| BenchTime | 1s |
| MinIterations | 3 |
| Warmup | 10 |
| Concurrency | 200 |
| Timeout | 30s |

## Drivers Tested

- **minio** (48 benchmarks)
- **usagi_s3** (48 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.19 MB/s | 697.8us | 1.5ms | 2.0ms | 0 |
| minio | 0.83 MB/s | 1.1ms | 1.9ms | 2.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.19 MB/s
minio        ████████████████████ 0.83 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 697.8us
minio        ██████████████████████████████ 1.1ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 3855 ops/s | 239.2us | 378.1us | 610.5us | 0 |
| minio | 2337 ops/s | 397.0us | 599.0us | 918.2us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 3855 ops/s
minio        ██████████████████ 2337 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 239.2us
minio        ██████████████████████████████ 397.0us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.14 MB/s | 634.0us | 1.1ms | 1.4ms | 0 |
| minio | 0.08 MB/s | 1.1ms | 1.5ms | 2.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.14 MB/s
minio        █████████████████ 0.08 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████ 634.0us
minio        ██████████████████████████████ 1.1ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1598 ops/s | 566.7us | 972.7us | 1.2ms | 0 |
| minio | 896 ops/s | 1.1ms | 1.6ms | 2.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1598 ops/s
minio        ████████████████ 896 ops/s
```

**Latency (P50)**
```
usagi_s3     ████████████████ 566.7us
minio        ██████████████████████████████ 1.1ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.13 MB/s | 662.2us | 1.2ms | 1.7ms | 0 |
| minio | 0.08 MB/s | 1.0ms | 2.0ms | 3.0ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.13 MB/s
minio        ██████████████████ 0.08 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 662.2us
minio        ██████████████████████████████ 1.0ms
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 886 ops/s | 1.1ms | 1.5ms | 1.7ms | 0 |
| minio | 496 ops/s | 1.9ms | 2.7ms | 3.9ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 886 ops/s
minio        ████████████████ 496 ops/s
```

**Latency (P50)**
```
usagi_s3     █████████████████ 1.1ms
minio        ██████████████████████████████ 1.9ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.31 MB/s | 9.8ms | 160.3ms | 201.4ms | 0 |
| minio | 0.24 MB/s | 33.3ms | 173.3ms | 262.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.31 MB/s
minio        ███████████████████████ 0.24 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████ 9.8ms
minio        ██████████████████████████████ 33.3ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.60 MB/s | 23.2ms | 52.8ms | 71.1ms | 0 |
| minio | 0.45 MB/s | 22.3ms | 131.2ms | 235.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.60 MB/s
minio        ██████████████████████ 0.45 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████████████ 23.2ms
minio        ████████████████████████████ 22.3ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.19 MB/s | 109.8ms | 128.9ms | 132.0ms | 0 |
| minio | 0.16 MB/s | 78.8ms | 233.3ms | 456.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.19 MB/s
minio        ████████████████████████ 0.16 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████████████ 109.8ms
minio        █████████████████████ 78.8ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 121.91 MB/s | 119.4ms | 134.3ms | 134.3ms | 0 |
| usagi_s3 | 107.81 MB/s | 132.9ms | 149.0ms | 149.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 121.91 MB/s
usagi_s3     ██████████████████████████ 107.81 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 119.4ms
usagi_s3     ██████████████████████████████ 132.9ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 3.22 MB/s | 303.2us | 450.2us | 280.3us | 450.2us | 677.6us | 0 |
| minio | 2.33 MB/s | 419.4us | 547.5us | 395.3us | 547.9us | 760.7us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 3.22 MB/s
minio        █████████████████████ 2.33 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████ 280.3us
minio        ██████████████████████████████ 395.3us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 1.02 MB/s | 957.4us | 1.5ms | 899.0us | 1.5ms | 1.9ms | 0 |
| minio | 0.68 MB/s | 1.4ms | 2.4ms | 1.3ms | 2.4ms | 3.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.02 MB/s
minio        ███████████████████ 0.68 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 899.0us
minio        ██████████████████████████████ 1.3ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.18 MB/s | 5.4ms | 9.6ms | 5.4ms | 9.6ms | 13.1ms | 0 |
| minio | 0.10 MB/s | 10.2ms | 20.7ms | 9.1ms | 20.7ms | 31.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.18 MB/s
minio        ███████████████ 0.10 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████ 5.4ms
minio        ██████████████████████████████ 9.1ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.08 MB/s | 11.8ms | 17.3ms | 11.9ms | 17.3ms | 20.8ms | 0 |
| minio | 0.05 MB/s | 19.9ms | 41.0ms | 18.3ms | 41.0ms | 63.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.08 MB/s
minio        █████████████████ 0.05 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 11.9ms
minio        ██████████████████████████████ 18.3ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.52 MB/s | 1.9ms | 3.1ms | 1.8ms | 3.1ms | 4.2ms | 0 |
| minio | 0.35 MB/s | 2.8ms | 4.6ms | 2.6ms | 4.6ms | 6.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.52 MB/s
minio        ███████████████████ 0.35 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 1.8ms
minio        ██████████████████████████████ 2.6ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.31 MB/s | 3.1ms | 5.2ms | 3.0ms | 5.2ms | 6.5ms | 0 |
| minio | 0.19 MB/s | 5.1ms | 10.0ms | 4.5ms | 10.0ms | 16.0ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.31 MB/s
minio        ██████████████████ 0.19 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 3.0ms
minio        ██████████████████████████████ 4.5ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.17 MB/s | 770.2us | 1.1ms | 2.0ms | 0 |
| minio | 1.03 MB/s | 884.4us | 1.3ms | 1.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.17 MB/s
minio        ██████████████████████████ 1.03 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████████ 770.2us
minio        ██████████████████████████████ 884.4us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.36 MB/s | 2.5ms | 4.0ms | 7.6ms | 0 |
| minio | 0.22 MB/s | 4.2ms | 7.3ms | 9.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.36 MB/s
minio        ██████████████████ 0.22 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 2.5ms
minio        ██████████████████████████████ 4.2ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.03 MB/s | 33.3ms | 50.2ms | 60.1ms | 0 |
| minio | 0.02 MB/s | 35.9ms | 101.2ms | 133.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.03 MB/s
minio        █████████████████████ 0.02 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████████████ 33.3ms
minio        ██████████████████████████████ 35.9ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.02 MB/s | 64.5ms | 83.1ms | 97.1ms | 0 |
| minio | 0.01 MB/s | 75.7ms | 168.4ms | 246.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.02 MB/s
minio        ███████████████████ 0.01 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████████ 64.5ms
minio        ██████████████████████████████ 75.7ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 0.14 MB/s | 6.8ms | 11.0ms | 14.0ms | 0 |
| usagi_s3 | 0.12 MB/s | 6.6ms | 16.8ms | 24.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.14 MB/s
usagi_s3     ███████████████████████████ 0.12 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████████ 6.8ms
usagi_s3     █████████████████████████████ 6.6ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.06 MB/s | 17.2ms | 26.7ms | 36.4ms | 0 |
| minio | 0.05 MB/s | 19.1ms | 37.4ms | 47.3ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.06 MB/s
minio        ███████████████████████ 0.05 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████████████ 17.2ms
minio        ██████████████████████████████ 19.1ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 148.64 MB/s | 1.6ms | 2.1ms | 2.3ms | 0 |
| minio | 116.73 MB/s | 2.1ms | 2.6ms | 3.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 148.64 MB/s
minio        ███████████████████████ 116.73 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████████ 1.6ms
minio        ██████████████████████████████ 2.1ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 142.70 MB/s | 1.7ms | 2.3ms | 2.7ms | 0 |
| minio | 115.99 MB/s | 2.1ms | 2.7ms | 3.3ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 142.70 MB/s
minio        ████████████████████████ 115.99 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████ 1.7ms
minio        ██████████████████████████████ 2.1ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 132.79 MB/s | 1.8ms | 2.5ms | 2.9ms | 0 |
| minio | 94.09 MB/s | 2.4ms | 3.8ms | 6.4ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 132.79 MB/s
minio        █████████████████████ 94.09 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████ 1.8ms
minio        ██████████████████████████████ 2.4ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 187.86 MB/s | 1.2ms | 1.2ms | 528.0ms | 528.0ms | 528.0ms | 0 |
| usagi_s3 | 171.64 MB/s | 4.1ms | 4.7ms | 590.4ms | 590.4ms | 590.4ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 187.86 MB/s
usagi_s3     ███████████████████████████ 171.64 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 528.0ms
usagi_s3     ██████████████████████████████ 590.4ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 172.76 MB/s | 1.4ms | 1.6ms | 55.2ms | 66.5ms | 71.5ms | 0 |
| usagi_s3 | 162.09 MB/s | 2.4ms | 3.5ms | 62.0ms | 68.3ms | 69.4ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 172.76 MB/s
usagi_s3     ████████████████████████████ 162.09 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 55.2ms
usagi_s3     ██████████████████████████████ 62.0ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 3.82 MB/s | 255.4us | 375.7us | 237.0us | 375.8us | 560.8us | 0 |
| minio | 2.75 MB/s | 355.0us | 499.0us | 328.3us | 499.0us | 720.7us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 3.82 MB/s
minio        █████████████████████ 2.75 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████ 237.0us
minio        ██████████████████████████████ 328.3us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 167.62 MB/s | 552.3us | 902.1us | 5.9ms | 7.0ms | 8.2ms | 0 |
| minio | 162.10 MB/s | 1.3ms | 1.8ms | 6.0ms | 7.3ms | 8.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 167.62 MB/s
minio        █████████████████████████████ 162.10 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████████████ 5.9ms
minio        ██████████████████████████████ 6.0ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 91.82 MB/s | 354.9us | 681.6us | 625.8us | 1.0ms | 1.3ms | 0 |
| minio | 76.84 MB/s | 600.3us | 1.4ms | 642.6us | 1.7ms | 4.0ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 91.82 MB/s
minio        █████████████████████████ 76.84 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████████████ 625.8us
minio        ██████████████████████████████ 642.6us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 368 ops/s | 2.7ms | 2.7ms | 2.7ms | 0 |
| minio | 229 ops/s | 4.4ms | 4.4ms | 4.4ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 368 ops/s
minio        ██████████████████ 229 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 2.7ms
minio        ██████████████████████████████ 4.4ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 38 ops/s | 26.2ms | 26.2ms | 26.2ms | 0 |
| minio | 22 ops/s | 44.6ms | 44.6ms | 44.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 38 ops/s
minio        █████████████████ 22 ops/s
```

**Latency (P50)**
```
usagi_s3     █████████████████ 26.2ms
minio        ██████████████████████████████ 44.6ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 3 ops/s | 291.1ms | 291.1ms | 291.1ms | 0 |
| minio | 2 ops/s | 446.5ms | 446.5ms | 446.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 3 ops/s
minio        ███████████████████ 2 ops/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 291.1ms
minio        ██████████████████████████████ 446.5ms
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0 ops/s | 2.65s | 2.65s | 2.65s | 0 |
| minio | 0 ops/s | 4.53s | 4.53s | 4.53s | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0 ops/s
minio        █████████████████ 0 ops/s
```

**Latency (P50)**
```
usagi_s3     █████████████████ 2.65s
minio        ██████████████████████████████ 4.53s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 2146 ops/s | 465.9us | 465.9us | 465.9us | 0 |
| minio | 1213 ops/s | 824.2us | 824.2us | 824.2us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 2146 ops/s
minio        ████████████████ 1213 ops/s
```

**Latency (P50)**
```
usagi_s3     ████████████████ 465.9us
minio        ██████████████████████████████ 824.2us
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 961 ops/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| minio | 405 ops/s | 2.5ms | 2.5ms | 2.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 961 ops/s
minio        ████████████ 405 ops/s
```

**Latency (P50)**
```
usagi_s3     ████████████ 1.0ms
minio        ██████████████████████████████ 2.5ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 129 ops/s | 7.8ms | 7.8ms | 7.8ms | 0 |
| minio | 64 ops/s | 15.7ms | 15.7ms | 15.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 129 ops/s
minio        ██████████████ 64 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████ 7.8ms
minio        ██████████████████████████████ 15.7ms
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 5 ops/s | 187.3ms | 187.3ms | 187.3ms | 0 |
| usagi_s3 | 4 ops/s | 228.6ms | 228.6ms | 228.6ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 5 ops/s
usagi_s3     ████████████████████████ 4 ops/s
```

**Latency (P50)**
```
minio        ████████████████████████ 187.3ms
usagi_s3     ██████████████████████████████ 228.6ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.32 MB/s | 7.6ms | 7.6ms | 7.6ms | 0 |
| minio | 0.22 MB/s | 11.0ms | 11.0ms | 11.0ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.32 MB/s
minio        ████████████████████ 0.22 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 7.6ms
minio        ██████████████████████████████ 11.0ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.37 MB/s | 66.3ms | 66.3ms | 66.3ms | 0 |
| minio | 0.24 MB/s | 103.3ms | 103.3ms | 103.3ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.37 MB/s
minio        ███████████████████ 0.24 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 66.3ms
minio        ██████████████████████████████ 103.3ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.33 MB/s | 735.9ms | 735.9ms | 735.9ms | 0 |
| minio | 0.22 MB/s | 1.12s | 1.12s | 1.12s | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.33 MB/s
minio        ███████████████████ 0.22 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 735.9ms
minio        ██████████████████████████████ 1.12s
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.40 MB/s | 6.15s | 6.15s | 6.15s | 0 |
| minio | 0.22 MB/s | 10.87s | 10.87s | 10.87s | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.40 MB/s
minio        ████████████████ 0.22 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████ 6.15s
minio        ██████████████████████████████ 10.87s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 3738 ops/s | 245.0us | 423.4us | 597.5us | 0 |
| minio | 3112 ops/s | 294.3us | 460.4us | 844.5us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 3738 ops/s
minio        ████████████████████████ 3112 ops/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████ 245.0us
minio        ██████████████████████████████ 294.3us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 148.53 MB/s | 679.7ms | 679.7ms | 679.7ms | 0 |
| usagi_s3 | 135.01 MB/s | 741.0ms | 741.0ms | 741.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 148.53 MB/s
usagi_s3     ███████████████████████████ 135.01 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 679.7ms
usagi_s3     ██████████████████████████████ 741.0ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 138.55 MB/s | 70.0ms | 86.8ms | 86.8ms | 0 |
| usagi_s3 | 127.78 MB/s | 76.8ms | 87.1ms | 87.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 138.55 MB/s
usagi_s3     ███████████████████████████ 127.78 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 70.0ms
usagi_s3     ██████████████████████████████ 76.8ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.79 MB/s | 505.5us | 808.1us | 1.1ms | 0 |
| minio | 1.32 MB/s | 705.9us | 981.4us | 1.3ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.79 MB/s
minio        ██████████████████████ 1.32 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████ 505.5us
minio        ██████████████████████████████ 705.9us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 113.30 MB/s | 8.4ms | 11.2ms | 12.0ms | 0 |
| usagi_s3 | 108.62 MB/s | 8.7ms | 12.3ms | 14.7ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 113.30 MB/s
usagi_s3     ████████████████████████████ 108.62 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████████ 8.4ms
usagi_s3     ██████████████████████████████ 8.7ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 56.20 MB/s | 929.6us | 1.9ms | 4.5ms | 0 |
| minio | 21.27 MB/s | 2.6ms | 4.8ms | 12.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 56.20 MB/s
minio        ███████████ 21.27 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████ 929.6us
minio        ██████████████████████████████ 2.6ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| minio | 812.8MiB / 7.653GiB | 812.8 MB | - | 3.5% | 1976.0 MB | 4.66MB / 2.2GB |
| usagi_s3 | 1.769GiB / 7.653GiB | 1811.5 MB | - | 3.6% | 6840.3 MB | 1.25MB / 2.3GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** minio
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
