# Storage Benchmark Report

**Generated:** 2026-01-21T15:27:13+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** usagi (won 35/51 benchmarks, 69%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | usagi | 35 | 69% |
| 2 | rabbit | 16 | 31% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | rabbit | 860.7 MB/s | 10.5x vs usagi |
| Small Write (1KB) | usagi | 66.2 MB/s | 14.9x vs rabbit |
| Large Read (100MB) | rabbit | 2.6 GB/s | +16% vs usagi |
| Large Write (100MB) | rabbit | 1.6 GB/s | 2.8x vs usagi |
| Delete | usagi | 33.4K ops/s | +86% vs rabbit |
| Stat | usagi | 1.5M ops/s | +46% vs rabbit |
| List (100 objects) | usagi | 15.5K ops/s | 5.1x vs rabbit |
| Copy | usagi | 28.8 MB/s | 13.6x vs rabbit |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **rabbit** | 1633 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **rabbit** | 2644 MB/s | Best for streaming, CDN |
| Small File Operations | **rabbit** | 442953 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **rabbit** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| rabbit | 1632.5 | 2643.6 | 69.6ms | 35.9ms |
| usagi | 579.4 | 2269.5 | 168.5ms | 42.8ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| rabbit | 4560 | 881347 | 151.3us | 916ns |
| usagi | 67776 | 84206 | 11.2us | 11.6us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| rabbit | 1004088 | 3055 | 17937 |
| usagi | 1469589 | 15461 | 33389 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| rabbit | 8.30 | 1.16 | 0.39 | 0.19 | 0.07 | 0.02 |
| usagi | 66.74 | 3.42 | 1.60 | 0.92 | 0.56 | 0.27 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| rabbit | 811.96 | 454.85 | 456.44 | 451.51 | 440.96 | 433.54 |
| usagi | 52.50 | 16.80 | 15.00 | 13.65 | 12.82 | 12.14 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| rabbit | 442.1us | 1.5ms | 17.5ms | 115.8ms | 1.33s |
| usagi | 60.9us | 229.5us | 1.3ms | 16.0ms | 148.1ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| rabbit | 68.4us | 127.4us | 706.6us | 5.5ms | 67.4ms |
| usagi | 287.4us | 123.0us | 181.7us | 715.8us | 5.2ms |

*\* indicates errors occurred*

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

- **rabbit** (51 benchmarks)
- **usagi** (51 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 28.79 MB/s | 29.0us | 52.5us | 116.8us | 0 |
| rabbit | 2.11 MB/s | 404.7us | 916.9us | 1.5ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 28.79 MB/s
rabbit       ██ 2.11 MB/s
```

**Latency (P50)**
```
usagi        ██ 29.0us
rabbit       ██████████████████████████████ 404.7us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 33389 ops/s | 28.6us | 50.4us | 82.2us | 0 |
| rabbit | 17937 ops/s | 48.9us | 70.7us | 82.6us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 33389 ops/s
rabbit       ████████████████ 17937 ops/s
```

**Latency (P50)**
```
usagi        █████████████████ 28.6us
rabbit       ██████████████████████████████ 48.9us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 4.42 MB/s | 18.9us | 29.8us | 103.6us | 0 |
| rabbit | 0.51 MB/s | 143.1us | 390.7us | 564.7us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 4.42 MB/s
rabbit       ███ 0.51 MB/s
```

**Latency (P50)**
```
usagi        ███ 18.9us
rabbit       ██████████████████████████████ 143.1us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 80174 ops/s | 10.2us | 17.3us | 62.1us | 0 |
| rabbit | 3533 ops/s | 225.8us | 621.8us | 847.8us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 80174 ops/s
rabbit       █ 3533 ops/s
```

**Latency (P50)**
```
usagi        █ 10.2us
rabbit       ██████████████████████████████ 225.8us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 4.94 MB/s | 17.1us | 26.5us | 73.7us | 0 |
| rabbit | 0.43 MB/s | 197.3us | 453.6us | 631.0us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 4.94 MB/s
rabbit       ██ 0.43 MB/s
```

**Latency (P50)**
```
usagi        ██ 17.1us
rabbit       ██████████████████████████████ 197.3us
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 46243 ops/s | 21.6us | 21.6us | 21.6us | 0 |
| rabbit | 11905 ops/s | 84.0us | 84.0us | 84.0us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 46243 ops/s
rabbit       ███████ 11905 ops/s
```

**Latency (P50)**
```
usagi        ███████ 21.6us
rabbit       ██████████████████████████████ 84.0us
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 10605 ops/s | 94.3us | 94.3us | 94.3us | 0 |
| rabbit | 1786 ops/s | 559.9us | 559.9us | 559.9us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 10605 ops/s
rabbit       █████ 1786 ops/s
```

**Latency (P50)**
```
usagi        █████ 94.3us
rabbit       ██████████████████████████████ 559.9us
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 879 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| rabbit | 148 ops/s | 6.7ms | 6.7ms | 6.7ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 879 ops/s
rabbit       █████ 148 ops/s
```

**Latency (P50)**
```
usagi        █████ 1.1ms
rabbit       ██████████████████████████████ 6.7ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 108 ops/s | 9.2ms | 9.2ms | 9.2ms | 0 |
| rabbit | 13 ops/s | 75.7ms | 75.7ms | 75.7ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 108 ops/s
rabbit       ███ 13 ops/s
```

**Latency (P50)**
```
usagi        ███ 9.2ms
rabbit       ██████████████████████████████ 75.7ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 7 ops/s | 140.4ms | 140.4ms | 140.4ms | 0 |
| rabbit | 1 ops/s | 907.6ms | 907.6ms | 907.6ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 7 ops/s
rabbit       ████ 1 ops/s
```

**Latency (P50)**
```
usagi        ████ 140.4ms
rabbit       ██████████████████████████████ 907.6ms
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 14625 ops/s | 68.4us | 68.4us | 68.4us | 0 |
| usagi | 3479 ops/s | 287.4us | 287.4us | 287.4us | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 14625 ops/s
usagi        ███████ 3479 ops/s
```

**Latency (P50)**
```
rabbit       ███████ 68.4us
usagi        ██████████████████████████████ 287.4us
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 8130 ops/s | 123.0us | 123.0us | 123.0us | 0 |
| rabbit | 7848 ops/s | 127.4us | 127.4us | 127.4us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 8130 ops/s
rabbit       ████████████████████████████ 7848 ops/s
```

**Latency (P50)**
```
usagi        ████████████████████████████ 123.0us
rabbit       ██████████████████████████████ 127.4us
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 5503 ops/s | 181.7us | 181.7us | 181.7us | 0 |
| rabbit | 1415 ops/s | 706.6us | 706.6us | 706.6us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 5503 ops/s
rabbit       ███████ 1415 ops/s
```

**Latency (P50)**
```
usagi        ███████ 181.7us
rabbit       ██████████████████████████████ 706.6us
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 1397 ops/s | 715.8us | 715.8us | 715.8us | 0 |
| rabbit | 183 ops/s | 5.5ms | 5.5ms | 5.5ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 1397 ops/s
rabbit       ███ 183 ops/s
```

**Latency (P50)**
```
usagi        ███ 715.8us
rabbit       ██████████████████████████████ 5.5ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 192 ops/s | 5.2ms | 5.2ms | 5.2ms | 0 |
| rabbit | 15 ops/s | 67.4ms | 67.4ms | 67.4ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 192 ops/s
rabbit       ██ 15 ops/s
```

**Latency (P50)**
```
usagi        ██ 5.2ms
rabbit       ██████████████████████████████ 67.4ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 16.03 MB/s | 60.9us | 60.9us | 60.9us | 0 |
| rabbit | 2.21 MB/s | 442.1us | 442.1us | 442.1us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 16.03 MB/s
rabbit       ████ 2.21 MB/s
```

**Latency (P50)**
```
usagi        ████ 60.9us
rabbit       ██████████████████████████████ 442.1us
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 42.56 MB/s | 229.5us | 229.5us | 229.5us | 0 |
| rabbit | 6.61 MB/s | 1.5ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 42.56 MB/s
rabbit       ████ 6.61 MB/s
```

**Latency (P50)**
```
usagi        ████ 229.5us
rabbit       ██████████████████████████████ 1.5ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 74.61 MB/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| rabbit | 5.57 MB/s | 17.5ms | 17.5ms | 17.5ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 74.61 MB/s
rabbit       ██ 5.57 MB/s
```

**Latency (P50)**
```
usagi        ██ 1.3ms
rabbit       ██████████████████████████████ 17.5ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 60.86 MB/s | 16.0ms | 16.0ms | 16.0ms | 0 |
| rabbit | 8.44 MB/s | 115.8ms | 115.8ms | 115.8ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 60.86 MB/s
rabbit       ████ 8.44 MB/s
```

**Latency (P50)**
```
usagi        ████ 16.0ms
rabbit       ██████████████████████████████ 115.8ms
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 65.94 MB/s | 148.1ms | 148.1ms | 148.1ms | 0 |
| rabbit | 7.35 MB/s | 1.33s | 1.33s | 1.33s | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 65.94 MB/s
rabbit       ███ 7.35 MB/s
```

**Latency (P50)**
```
usagi        ███ 148.1ms
rabbit       ██████████████████████████████ 1.33s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 15461 ops/s | 47.7us | 118.6us | 268.2us | 0 |
| rabbit | 3055 ops/s | 293.6us | 606.9us | 760.8us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 15461 ops/s
rabbit       █████ 3055 ops/s
```

**Latency (P50)**
```
usagi        ████ 47.7us
rabbit       ██████████████████████████████ 293.6us
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 4.05 MB/s | 105.3us | 12.6ms | 40.2ms | 0 |
| rabbit | 0.38 MB/s | 228.3us | 190.0ms | 237.2ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 4.05 MB/s
rabbit       ██ 0.38 MB/s
```

**Latency (P50)**
```
usagi        █████████████ 105.3us
rabbit       ██████████████████████████████ 228.3us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 11.68 MB/s | 50.5us | 9.1ms | 19.6ms | 0 |
| rabbit | 1.33 MB/s | 2.2us | 127.6ms | 191.4ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 11.68 MB/s
rabbit       ███ 1.33 MB/s
```

**Latency (P50)**
```
usagi        ██████████████████████████████ 50.5us
rabbit       █ 2.2us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 2.45 MB/s | 5.2ms | 14.7ms | 49.5ms | 0 |
| rabbit | 0.16 MB/s | 88.9ms | 230.5ms | 284.0ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 2.45 MB/s
rabbit       █ 0.16 MB/s
```

**Latency (P50)**
```
usagi        █ 5.2ms
rabbit       ██████████████████████████████ 88.9ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 408.47 MB/s | 25.7ms | 83.7ms | 86.4ms | 0 |
| rabbit | 241.20 MB/s | 55.8ms | 65.2ms | 68.4ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 408.47 MB/s
rabbit       █████████████████ 241.20 MB/s
```

**Latency (P50)**
```
usagi        █████████████ 25.7ms
rabbit       ██████████████████████████████ 55.8ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 811.96 MB/s | 1.2us | 1.6us | 1.1us | 1.6us | 2.2us | 0 |
| usagi | 52.50 MB/s | 17.8us | 25.6us | 16.8us | 26.8us | 58.3us | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 811.96 MB/s
usagi        █ 52.50 MB/s
```

**Latency (P50)**
```
rabbit       ██ 1.1us
usagi        ██████████████████████████████ 16.8us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 454.85 MB/s | 2.1us | 3.2us | 1.8us | 3.3us | 14.8us | 0 |
| usagi | 16.80 MB/s | 53.7us | 84.4us | 41.9us | 91.8us | 328.7us | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 454.85 MB/s
usagi        █ 16.80 MB/s
```

**Latency (P50)**
```
rabbit       █ 1.8us
usagi        ██████████████████████████████ 41.9us
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 440.96 MB/s | 2.2us | 3.0us | 1.8us | 3.1us | 14.3us | 0 |
| usagi | 12.82 MB/s | 71.6us | 95.2us | 41.6us | 104.8us | 589.8us | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 440.96 MB/s
usagi        █ 12.82 MB/s
```

**Latency (P50)**
```
rabbit       █ 1.8us
usagi        ██████████████████████████████ 41.6us
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 433.54 MB/s | 2.2us | 3.1us | 1.8us | 3.2us | 15.4us | 0 |
| usagi | 12.14 MB/s | 75.1us | 101.8us | 41.9us | 113.7us | 770.8us | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 433.54 MB/s
usagi        █ 12.14 MB/s
```

**Latency (P50)**
```
rabbit       █ 1.8us
usagi        ██████████████████████████████ 41.9us
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 456.44 MB/s | 2.1us | 3.2us | 1.9us | 3.2us | 12.9us | 0 |
| usagi | 15.00 MB/s | 60.5us | 92.9us | 41.0us | 104.0us | 477.0us | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 456.44 MB/s
usagi        █ 15.00 MB/s
```

**Latency (P50)**
```
rabbit       █ 1.9us
usagi        ██████████████████████████████ 41.0us
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 451.51 MB/s | 2.1us | 3.0us | 1.8us | 3.1us | 14.3us | 0 |
| usagi | 13.65 MB/s | 66.2us | 87.7us | 41.2us | 97.5us | 543.0us | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 451.51 MB/s
usagi        █ 13.65 MB/s
```

**Latency (P50)**
```
rabbit       █ 1.8us
usagi        ██████████████████████████████ 41.2us
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 66.74 MB/s | 11.6us | 25.4us | 51.2us | 0 |
| rabbit | 8.30 MB/s | 110.3us | 143.0us | 174.0us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 66.74 MB/s
rabbit       ███ 8.30 MB/s
```

**Latency (P50)**
```
usagi        ███ 11.6us
rabbit       ██████████████████████████████ 110.3us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 3.42 MB/s | 26.9us | 1.0ms | 1.6ms | 0 |
| rabbit | 1.16 MB/s | 637.8us | 1.5ms | 4.7ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 3.42 MB/s
rabbit       ██████████ 1.16 MB/s
```

**Latency (P50)**
```
usagi        █ 26.9us
rabbit       ██████████████████████████████ 637.8us
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 0.56 MB/s | 1.7ms | 3.1ms | 5.1ms | 0 |
| rabbit | 0.07 MB/s | 9.8ms | 44.7ms | 63.5ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 0.56 MB/s
rabbit       ███ 0.07 MB/s
```

**Latency (P50)**
```
usagi        █████ 1.7ms
rabbit       ██████████████████████████████ 9.8ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 0.27 MB/s | 3.6ms | 5.6ms | 9.3ms | 0 |
| rabbit | 0.02 MB/s | 28.1ms | 163.7ms | 221.1ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 0.27 MB/s
rabbit       ██ 0.02 MB/s
```

**Latency (P50)**
```
usagi        ███ 3.6ms
rabbit       ██████████████████████████████ 28.1ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 1.60 MB/s | 283.5us | 1.7ms | 2.3ms | 0 |
| rabbit | 0.39 MB/s | 1.8ms | 6.0ms | 18.3ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 1.60 MB/s
rabbit       ███████ 0.39 MB/s
```

**Latency (P50)**
```
usagi        ████ 283.5us
rabbit       ██████████████████████████████ 1.8ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 0.92 MB/s | 1.1ms | 2.0ms | 2.9ms | 0 |
| rabbit | 0.19 MB/s | 3.3ms | 19.9ms | 28.1ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 0.92 MB/s
rabbit       ██████ 0.19 MB/s
```

**Latency (P50)**
```
usagi        ██████████ 1.1ms
rabbit       ██████████████████████████████ 3.3ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 5836.95 MB/s | 39.2us | 63.2us | 103.9us | 0 |
| usagi | 5206.94 MB/s | 45.0us | 68.3us | 129.6us | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 5836.95 MB/s
usagi        ██████████████████████████ 5206.94 MB/s
```

**Latency (P50)**
```
rabbit       ██████████████████████████ 39.2us
usagi        ██████████████████████████████ 45.0us
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 6387.52 MB/s | 36.1us | 40.8us | 70.8us | 0 |
| usagi | 5080.40 MB/s | 45.0us | 78.0us | 137.4us | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 6387.52 MB/s
usagi        ███████████████████████ 5080.40 MB/s
```

**Latency (P50)**
```
rabbit       ████████████████████████ 36.1us
usagi        ██████████████████████████████ 45.0us
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 6444.80 MB/s | 38.6us | 43.6us | 54.2us | 0 |
| usagi | 5427.81 MB/s | 44.3us | 59.9us | 117.2us | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 6444.80 MB/s
usagi        █████████████████████████ 5427.81 MB/s
```

**Latency (P50)**
```
rabbit       ██████████████████████████ 38.6us
usagi        ██████████████████████████████ 44.3us
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 2643.56 MB/s | 227.1us | 427.0us | 35.9ms | 48.1ms | 50.3ms | 0 |
| usagi | 2269.50 MB/s | 571.7us | 1.4ms | 42.8ms | 58.9ms | 66.2ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 2643.56 MB/s
usagi        █████████████████████████ 2269.50 MB/s
```

**Latency (P50)**
```
rabbit       █████████████████████████ 35.9ms
usagi        ██████████████████████████████ 42.8ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi | 7574.13 MB/s | 26.2us | 42.7us | 1.3ms | 1.4ms | 1.6ms | 0 |
| rabbit | 5669.79 MB/s | 48.8us | 217.6us | 1.4ms | 6.1ms | 8.5ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 7574.13 MB/s
rabbit       ██████████████████████ 5669.79 MB/s
```

**Latency (P50)**
```
usagi        █████████████████████████████ 1.3ms
rabbit       ██████████████████████████████ 1.4ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 860.69 MB/s | 1.1us | 1.6us | 916ns | 1.7us | 3.8us | 0 |
| usagi | 82.23 MB/s | 11.4us | 13.2us | 11.6us | 13.8us | 22.2us | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 860.69 MB/s
usagi        ██ 82.23 MB/s
```

**Latency (P50)**
```
rabbit       ██ 916ns
usagi        ██████████████████████████████ 11.6us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi | 6811.35 MB/s | 16.2us | 24.6us | 141.4us | 179.7us | 258.7us | 0 |
| rabbit | 4587.97 MB/s | 31.7us | 75.5us | 175.8us | 326.5us | 1.0ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 6811.35 MB/s
rabbit       ████████████████████ 4587.97 MB/s
```

**Latency (P50)**
```
usagi        ████████████████████████ 141.4us
rabbit       ██████████████████████████████ 175.8us
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit | 12658.53 MB/s | 2.4us | 4.0us | 3.6us | 7.1us | 27.8us | 0 |
| usagi | 3205.00 MB/s | 12.6us | 14.0us | 18.9us | 21.7us | 35.0us | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 12658.53 MB/s
usagi        ███████ 3205.00 MB/s
```

**Latency (P50)**
```
rabbit       █████ 3.6us
usagi        ██████████████████████████████ 18.9us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 1469589 ops/s | 542ns | 875ns | 2.5us | 0 |
| rabbit | 1004088 ops/s | 708ns | 2.4us | 5.6us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 1469589 ops/s
rabbit       ████████████████████ 1004088 ops/s
```

**Latency (P50)**
```
usagi        ██████████████████████ 542ns
rabbit       ██████████████████████████████ 708ns
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 1632.52 MB/s | 69.6ms | 95.9ms | 95.9ms | 0 |
| usagi | 579.45 MB/s | 168.5ms | 181.6ms | 181.6ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 1632.52 MB/s
usagi        ██████████ 579.45 MB/s
```

**Latency (P50)**
```
rabbit       ████████████ 69.6ms
usagi        ██████████████████████████████ 168.5ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 1615.29 MB/s | 5.1ms | 12.7ms | 14.4ms | 0 |
| usagi | 560.00 MB/s | 8.5ms | 54.4ms | 157.4ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 1615.29 MB/s
usagi        ██████████ 560.00 MB/s
```

**Latency (P50)**
```
rabbit       █████████████████ 5.1ms
usagi        ██████████████████████████████ 8.5ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 66.19 MB/s | 11.2us | 22.7us | 57.5us | 0 |
| rabbit | 4.45 MB/s | 151.3us | 449.0us | 925.3us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 66.19 MB/s
rabbit       ██ 4.45 MB/s
```

**Latency (P50)**
```
usagi        ██ 11.2us
rabbit       ██████████████████████████████ 151.3us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit | 1410.59 MB/s | 507.8us | 1.6ms | 2.3ms | 0 |
| usagi | 1260.89 MB/s | 306.2us | 2.6ms | 8.7ms | 0 |

**Throughput**
```
rabbit       ██████████████████████████████ 1410.59 MB/s
usagi        ██████████████████████████ 1260.89 MB/s
```

**Latency (P50)**
```
rabbit       ██████████████████████████████ 507.8us
usagi        ██████████████████ 306.2us
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 934.28 MB/s | 33.5us | 78.8us | 694.7us | 0 |
| rabbit | 223.81 MB/s | 202.7us | 552.7us | 1.1ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 934.28 MB/s
rabbit       ███████ 223.81 MB/s
```

**Latency (P50)**
```
usagi        ████ 33.5us
rabbit       ██████████████████████████████ 202.7us
```

## Recommendations

- **Write-heavy workloads:** rabbit
- **Read-heavy workloads:** rabbit

---

*Generated by storage benchmark CLI*
