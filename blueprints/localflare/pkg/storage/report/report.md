# Storage Benchmark Report

**Generated:** 2026-01-21T12:14:39+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** rabbit (won 42/43 benchmarks, 98%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | rabbit | 42 | 98% |
| 2 | minio | 1 | 2% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | rabbit | 910.7 MB/s | 766.4x vs minio |
| Small Write (1KB) | rabbit | 5.8 MB/s | 10.8x vs minio |
| Large Read (10MB) | rabbit | 1.6 GB/s | 17.3x vs minio |
| Large Write (10MB) | rabbit | 1.1 GB/s | 19.7x vs minio |
| Delete | rabbit | 11.7K ops/s | 9.1x vs minio |
| Stat | rabbit | 684.2K ops/s | 545.9x vs minio |
| List (100 objects) | rabbit | 2.3K ops/s | 8.7x vs minio |
| Copy | rabbit | 4.6 MB/s | 10.2x vs minio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **rabbit** | 1061 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **rabbit** | 1616 MB/s | Best for streaming, CDN |
| Small File Operations | **rabbit** | 469252 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **rabbit** | - | Best for multi-user apps |
| Memory Constrained | **minio** | 438 MB RAM | Best for edge/embedded |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| minio | 53.8 | 93.6 | 193.2ms | 110.7ms |
| rabbit | 1060.9 | 1616.5 | 6.1ms | 5.8ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| minio | 551 | 1217 | 1.7ms | 787.5us |
| rabbit | 5964 | 932540 | 156.0us | 875ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| minio | 1253 | 264 | 1293 |
| rabbit | 684181 | 2289 | 11716 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| minio | 0.50 | 0.13 | 0.03 |
| rabbit | 3.99 | 0.82 | 0.15 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| minio | 1.13 | 0.45 | 0.13 |
| rabbit | 475.10 | 305.00 | 313.68 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| minio | 2.4ms | 21.8ms | 221.8ms | 2.14s | 22.80s |
| rabbit | 441.5us | 1.6ms | 13.0ms | 188.1ms | 2.12s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| minio | 1.0ms | 1.5ms | 3.9ms | 26.8ms | 305.8ms |
| rabbit | 132.2us | 122.4us | 824.8us | 7.8ms | 79.4ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| minio | 437.9 MB | 6.6% |

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

- **minio** (43 benchmarks)
- **rabbit** (43 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 4.62 MB/s | 188.6us | 360.0us | 467.6us | 0 |
| minio | 0.45 MB/s | 1.9ms | 3.7ms | 4.8ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 4.62 MB/s
minio        ██ 0.45 MB/s
```

**Latency (P50)**
```
rabbit       ██ 188.6us
minio        ██████████████████████████████ 1.9ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 11716 ops/s | 79.5us | 107.6us | 128.7us | 0 |
| minio | 1293 ops/s | 732.9us | 990.4us | 1.4ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 11716 ops/s
minio        ███ 1293 ops/s
```

**Latency (P50)**
```
rabbit       ███ 79.5us
minio        ██████████████████████████████ 732.9us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 0.61 MB/s | 135.2us | 231.9us | 315.9us | 0 |
| minio | 0.04 MB/s | 2.0ms | 3.8ms | 5.4ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 0.61 MB/s
minio        ██ 0.04 MB/s
```

**Latency (P50)**
```
rabbit       █ 135.2us
minio        ██████████████████████████████ 2.0ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 3737 ops/s | 248.8us | 476.7us | 815.8us | 0 |
| minio | 410 ops/s | 2.1ms | 4.4ms | 6.2ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 3737 ops/s
minio        ███ 410 ops/s
```

**Latency (P50)**
```
rabbit       ███ 248.8us
minio        ██████████████████████████████ 2.1ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 0.55 MB/s | 169.8us | 246.8us | 280.5us | 0 |
| minio | 0.05 MB/s | 1.9ms | 2.8ms | 4.4ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 0.55 MB/s
minio        ██ 0.05 MB/s
```

**Latency (P50)**
```
rabbit       ██ 169.8us
minio        ██████████████████████████████ 1.9ms
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 4886 ops/s | 204.7us | 204.7us | 204.7us | 0 |
| minio | 1037 ops/s | 964.7us | 964.7us | 964.7us | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 4886 ops/s
minio        ██████ 1037 ops/s
```

**Latency (P50)**
```
rabbit       ██████ 204.7us
minio        ██████████████████████████████ 964.7us
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 1425 ops/s | 701.5us | 701.5us | 701.5us | 0 |
| minio | 131 ops/s | 7.6ms | 7.6ms | 7.6ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 1425 ops/s
minio        ██ 131 ops/s
```

**Latency (P50)**
```
rabbit       ██ 701.5us
minio        ██████████████████████████████ 7.6ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 128 ops/s | 7.8ms | 7.8ms | 7.8ms | 0 |
| minio | 13 ops/s | 76.9ms | 76.9ms | 76.9ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 128 ops/s
minio        ███ 13 ops/s
```

**Latency (P50)**
```
rabbit       ███ 7.8ms
minio        ██████████████████████████████ 76.9ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 9 ops/s | 108.7ms | 108.7ms | 108.7ms | 0 |
| minio | 1 ops/s | 856.7ms | 856.7ms | 856.7ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 9 ops/s
minio        ███ 1 ops/s
```

**Latency (P50)**
```
rabbit       ███ 108.7ms
minio        ██████████████████████████████ 856.7ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 1 ops/s | 1.20s | 1.20s | 1.20s | 0 |
| minio | 0 ops/s | 8.71s | 8.71s | 8.71s | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 1 ops/s
minio        ████ 0 ops/s
```

**Latency (P50)**
```
rabbit       ████ 1.20s
minio        ██████████████████████████████ 8.71s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 7566 ops/s | 132.2us | 132.2us | 132.2us | 0 |
| minio | 985 ops/s | 1.0ms | 1.0ms | 1.0ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 7566 ops/s
minio        ███ 985 ops/s
```

**Latency (P50)**
```
rabbit       ███ 132.2us
minio        ██████████████████████████████ 1.0ms
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 8169 ops/s | 122.4us | 122.4us | 122.4us | 0 |
| minio | 689 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 8169 ops/s
minio        ██ 689 ops/s
```

**Latency (P50)**
```
rabbit       ██ 122.4us
minio        ██████████████████████████████ 1.5ms
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 1212 ops/s | 824.8us | 824.8us | 824.8us | 0 |
| minio | 255 ops/s | 3.9ms | 3.9ms | 3.9ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 1212 ops/s
minio        ██████ 255 ops/s
```

**Latency (P50)**
```
rabbit       ██████ 824.8us
minio        ██████████████████████████████ 3.9ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 128 ops/s | 7.8ms | 7.8ms | 7.8ms | 0 |
| minio | 37 ops/s | 26.8ms | 26.8ms | 26.8ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 128 ops/s
minio        ████████ 37 ops/s
```

**Latency (P50)**
```
rabbit       ████████ 7.8ms
minio        ██████████████████████████████ 26.8ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 13 ops/s | 79.4ms | 79.4ms | 79.4ms | 0 |
| minio | 3 ops/s | 305.8ms | 305.8ms | 305.8ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 13 ops/s
minio        ███████ 3 ops/s
```

**Latency (P50)**
```
rabbit       ███████ 79.4ms
minio        ██████████████████████████████ 305.8ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 2.21 MB/s | 441.5us | 441.5us | 441.5us | 0 |
| minio | 0.40 MB/s | 2.4ms | 2.4ms | 2.4ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 2.21 MB/s
minio        █████ 0.40 MB/s
```

**Latency (P50)**
```
rabbit       █████ 441.5us
minio        ██████████████████████████████ 2.4ms
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 6.28 MB/s | 1.6ms | 1.6ms | 1.6ms | 0 |
| minio | 0.45 MB/s | 21.8ms | 21.8ms | 21.8ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 6.28 MB/s
minio        ██ 0.45 MB/s
```

**Latency (P50)**
```
rabbit       ██ 1.6ms
minio        ██████████████████████████████ 21.8ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 7.50 MB/s | 13.0ms | 13.0ms | 13.0ms | 0 |
| minio | 0.44 MB/s | 221.8ms | 221.8ms | 221.8ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 7.50 MB/s
minio        █ 0.44 MB/s
```

**Latency (P50)**
```
rabbit       █ 13.0ms
minio        ██████████████████████████████ 221.8ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 5.19 MB/s | 188.1ms | 188.1ms | 188.1ms | 0 |
| minio | 0.46 MB/s | 2.14s | 2.14s | 2.14s | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 5.19 MB/s
minio        ██ 0.46 MB/s
```

**Latency (P50)**
```
rabbit       ██ 188.1ms
minio        ██████████████████████████████ 2.14s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 4.61 MB/s | 2.12s | 2.12s | 2.12s | 0 |
| minio | 0.43 MB/s | 22.80s | 22.80s | 22.80s | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 4.61 MB/s
minio        ██ 0.43 MB/s
```

**Latency (P50)**
```
rabbit       ██ 2.12s
minio        ██████████████████████████████ 22.80s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 2289 ops/s | 427.5us | 487.5us | 589.5us | 0 |
| minio | 264 ops/s | 3.7ms | 4.4ms | 5.5ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 2289 ops/s
minio        ███ 264 ops/s
```

**Latency (P50)**
```
rabbit       ███ 427.5us
minio        ██████████████████████████████ 3.7ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 0.24 MB/s | 808.0us | 305.4ms | 405.9ms | 0 |
| minio | 0.16 MB/s | 56.3ms | 298.1ms | 351.8ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 0.24 MB/s
minio        ████████████████████ 0.16 MB/s
```

**Latency (P50)**
```
rabbit       █ 808.0us
minio        ██████████████████████████████ 56.3ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 2.20 MB/s | 4.3us | 15.3ms | 222.7ms | 0 |
| minio | 0.25 MB/s | 48.7ms | 161.4ms | 230.2ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 2.20 MB/s
minio        ███ 0.25 MB/s
```

**Latency (P50)**
```
rabbit       █ 4.3us
minio        ██████████████████████████████ 48.7ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 0.22 MB/s | 70.8ms | 114.7ms | 145.0ms | 0 |
| rabbit | 0.15 MB/s | 72.0ms | 288.4ms | 335.5ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.22 MB/s
rabbit       ████████████████████ 0.15 MB/s
```

**Latency (P50)**
```
minio        █████████████████████████████ 70.8ms
rabbit       ██████████████████████████████ 72.0ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 241.49 MB/s | 58.7ms | 63.2ms | 63.2ms | 0 |
| minio | 56.43 MB/s | 273.1ms | 273.1ms | 273.1ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 241.49 MB/s
minio        ███████ 56.43 MB/s
```

**Latency (P50)**
```
rabbit       ██████ 58.7ms
minio        ██████████████████████████████ 273.1ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 475.10 MB/s | 2.0us | 2.6us | 2.0us | 2.7us | 4.2us | 0 |
| minio | 1.13 MB/s | 863.9us | 1.2ms | 821.4us | 1.2ms | 1.5ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 475.10 MB/s
minio        █ 1.13 MB/s
```

**Latency (P50)**
```
rabbit       █ 2.0us
minio        ██████████████████████████████ 821.4us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 305.00 MB/s | 3.1us | 4.0us | 2.4us | 4.0us | 19.1us | 0 |
| minio | 0.45 MB/s | 2.2ms | 3.2ms | 2.1ms | 3.2ms | 4.3ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 305.00 MB/s
minio        █ 0.45 MB/s
```

**Latency (P50)**
```
rabbit       █ 2.4us
minio        ██████████████████████████████ 2.1ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 313.68 MB/s | 3.0us | 3.7us | 2.3us | 3.8us | 20.8us | 0 |
| minio | 0.13 MB/s | 7.4ms | 13.1ms | 6.7ms | 13.1ms | 20.2ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 313.68 MB/s
minio        █ 0.13 MB/s
```

**Latency (P50)**
```
rabbit       █ 2.3us
minio        ██████████████████████████████ 6.7ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 3.99 MB/s | 199.0us | 436.5us | 671.9us | 0 |
| minio | 0.50 MB/s | 1.9ms | 2.5ms | 3.1ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 3.99 MB/s
minio        ███ 0.50 MB/s
```

**Latency (P50)**
```
rabbit       ███ 199.0us
minio        ██████████████████████████████ 1.9ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 0.82 MB/s | 964.8us | 2.5ms | 4.0ms | 0 |
| minio | 0.13 MB/s | 6.6ms | 10.0ms | 14.5ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 0.82 MB/s
minio        ████ 0.13 MB/s
```

**Latency (P50)**
```
rabbit       ████ 964.8us
minio        ██████████████████████████████ 6.6ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 0.15 MB/s | 3.4ms | 23.1ms | 43.6ms | 0 |
| minio | 0.03 MB/s | 27.1ms | 56.0ms | 70.3ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 0.15 MB/s
minio        ██████ 0.03 MB/s
```

**Latency (P50)**
```
rabbit       ███ 3.4ms
minio        ██████████████████████████████ 27.1ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 4432.68 MB/s | 56.9us | 59.8us | 68.0us | 0 |
| minio | 55.05 MB/s | 4.3ms | 6.1ms | 6.7ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 4432.68 MB/s
minio        █ 55.05 MB/s
```

**Latency (P50)**
```
rabbit       █ 56.9us
minio        ██████████████████████████████ 4.3ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 4297.53 MB/s | 57.0us | 67.5us | 71.2us | 0 |
| minio | 52.22 MB/s | 4.5ms | 6.4ms | 7.8ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 4297.53 MB/s
minio        █ 52.22 MB/s
```

**Latency (P50)**
```
rabbit       █ 57.0us
minio        ██████████████████████████████ 4.5ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 3637.76 MB/s | 67.8us | 71.9us | 82.9us | 0 |
| minio | 50.98 MB/s | 4.7ms | 6.0ms | 9.0ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 3637.76 MB/s
minio        █ 50.98 MB/s
```

**Latency (P50)**
```
rabbit       █ 67.8us
minio        ██████████████████████████████ 4.7ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 1616.48 MB/s | 218.0us | 586.1us | 5.8ms | 9.1ms | 11.6ms | 0 |
| minio | 93.62 MB/s | 3.1ms | 3.6ms | 110.7ms | 111.8ms | 111.8ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 1616.48 MB/s
minio        █ 93.62 MB/s
```

**Latency (P50)**
```
rabbit       █ 5.8ms
minio        ██████████████████████████████ 110.7ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 910.68 MB/s | 1.0us | 1.9us | 875ns | 2.0us | 3.6us | 0 |
| minio | 1.19 MB/s | 821.5us | 1.1ms | 787.5us | 1.1ms | 1.4ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 910.68 MB/s
minio        █ 1.19 MB/s
```

**Latency (P50)**
```
rabbit       █ 875ns
minio        ██████████████████████████████ 787.5us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 3808.22 MB/s | 32.1us | 41.5us | 229.9us | 335.7us | 642.6us | 0 |
| minio | 71.46 MB/s | 3.1ms | 4.5ms | 13.7ms | 16.7ms | 17.0ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 3808.22 MB/s
minio        █ 71.46 MB/s
```

**Latency (P50)**
```
rabbit       █ 229.9us
minio        ██████████████████████████████ 13.7ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 8858.27 MB/s | 4.5us | 22.1us | 3.9us | 23.9us | 49.8us | 0 |
| minio | 41.74 MB/s | 1.0ms | 1.5ms | 1.4ms | 2.0ms | 2.4ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 8858.27 MB/s
minio        █ 41.74 MB/s
```

**Latency (P50)**
```
rabbit       █ 3.9us
minio        ██████████████████████████████ 1.4ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 684181 ops/s | 1.1us | 2.8us | 8.6us | 0 |
| minio | 1253 ops/s | 753.5us | 1.1ms | 1.5ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 684181 ops/s
minio        █ 1253 ops/s
```

**Latency (P50)**
```
rabbit       █ 1.1us
minio        ██████████████████████████████ 753.5us
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 1060.90 MB/s | 6.1ms | 24.4ms | 65.3ms | 0 |
| minio | 53.77 MB/s | 193.2ms | 193.2ms | 193.2ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 1060.90 MB/s
minio        █ 53.77 MB/s
```

**Latency (P50)**
```
rabbit       █ 6.1ms
minio        ██████████████████████████████ 193.2ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 5.82 MB/s | 156.0us | 205.6us | 363.9us | 0 |
| minio | 0.54 MB/s | 1.7ms | 2.4ms | 2.9ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 5.82 MB/s
minio        ██ 0.54 MB/s
```

**Latency (P50)**
```
rabbit       ██ 156.0us
minio        ██████████████████████████████ 1.7ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 987.00 MB/s | 758.8us | 2.0ms | 3.8ms | 0 |
| minio | 44.79 MB/s | 21.4ms | 27.5ms | 28.6ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 987.00 MB/s
minio        █ 44.79 MB/s
```

**Latency (P50)**
```
rabbit       █ 758.8us
minio        ██████████████████████████████ 21.4ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 213.30 MB/s | 235.5us | 501.9us | 1.4ms | 0 |
| minio | 13.78 MB/s | 3.9ms | 7.4ms | 11.0ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 213.30 MB/s
minio        █ 13.78 MB/s
```

**Latency (P50)**
```
rabbit       █ 235.5us
minio        ██████████████████████████████ 3.9ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| minio | 438.3MiB / 7.653GiB | 438.3 MB | - | 6.6% | 3016.0 MB | 69.6kB / 1.53GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** rabbit
- **Read-heavy workloads:** rabbit

---

*Generated by storage benchmark CLI*
