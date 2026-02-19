# Storage Benchmark Report

**Generated:** 2026-02-19T14:21:53+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** horse (won 26/40 benchmarks, 65%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | horse | 26 | 65% |
| 2 | zebra | 14 | 35% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | horse | 8.5 GB/s | close |
| Small Write (1KB) | zebra | 1.8 GB/s | +31% vs horse |
| Large Read (10MB) | horse | 88710.2 GB/s | +13% vs zebra |
| Large Write (10MB) | horse | 1.2 GB/s | 3.8x vs zebra |
| Delete | zebra | 2.7M ops/s | close |
| Stat | horse | 15.7M ops/s | +25% vs zebra |
| List (100 objects) | horse | 134.1K ops/s | 4.1x vs zebra |
| Copy | zebra | 895.4 MB/s | 11.1x vs horse |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **horse** | 1165 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **horse** | 88710197 MB/s | Best for streaming, CDN |
| Small File Operations | **horse** | 5064874 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **horse** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| horse | 1165.0 | 88710197.2 | 267.5us | 125ns |
| zebra | 306.0 | 78811175.3 | 260.0us | 125ns |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| horse | 1382942 | 8746806 | 500ns | 84ns |
| zebra | 1816993 | 8010458 | 375ns | 84ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| horse | 15733682 | 134126 | 2480573 |
| zebra | 12582137 | 32506 | 2691038 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 994.12 | 107.82 | 43.69 |
| zebra | 788.02 | 318.66 | 75.10 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 6216.06 | 5115.70 | 4606.75 |
| zebra | 4236.30 | 2418.30 | 2413.58 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 17.7us | 22.3us | 188.0us | 2.3ms |
| zebra | 7.2us | 27.8us | 131.5us | 2.0ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 4.2us | 6.6us | 148.9us | 909.2us |
| zebra | 1.80s | 2.28s | 3.63s | 3.76s |

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

- **horse** (40 benchmarks)
- **zebra** (40 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 895.43 MB/s | 625ns | 1.8us | 9.4us | 0 |
| horse | 80.39 MB/s | 916ns | 1.6us | 3.7us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 895.43 MB/s
horse        ██ 80.39 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████ 625ns
horse        ██████████████████████████████ 916ns
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 2691038 ops/s | 333ns | 583ns | 875ns | 0 |
| horse | 2480573 ops/s | 375ns | 583ns | 709ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 2691038 ops/s
horse        ███████████████████████████ 2480573 ops/s
```

**Latency (P50)**
```
zebra        ██████████████████████████ 333ns
horse        ██████████████████████████████ 375ns
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 101.05 MB/s | 584ns | 1.5us | 5.3us | 0 |
| horse | 43.03 MB/s | 916ns | 2.3us | 6.7us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 101.05 MB/s
horse        ████████████ 43.03 MB/s
```

**Latency (P50)**
```
zebra        ███████████████████ 584ns
horse        ██████████████████████████████ 916ns
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1306478 ops/s | 500ns | 1.2us | 2.3us | 0 |
| zebra | 681061 ops/s | 667ns | 2.6us | 9.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1306478 ops/s
zebra        ███████████████ 681061 ops/s
```

**Latency (P50)**
```
horse        ██████████████████████ 500ns
zebra        ██████████████████████████████ 667ns
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 94.18 MB/s | 750ns | 1.3us | 2.2us | 0 |
| zebra | 56.90 MB/s | 916ns | 2.5us | 10.8us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 94.18 MB/s
zebra        ██████████████████ 56.90 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████ 750ns
zebra        ██████████████████████████████ 916ns
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 134126 ops/s | 6.7us | 11.8us | 18.7us | 0 |
| zebra | 32506 ops/s | 18.9us | 67.6us | 182.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 134126 ops/s
zebra        ███████ 32506 ops/s
```

**Latency (P50)**
```
horse        ██████████ 6.7us
zebra        ██████████████████████████████ 18.9us
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 18.73 MB/s | 5.6us | 915.3us | 3.9ms | 0 |
| zebra | 4.37 MB/s | 2.1us | 196.6us | 4.9ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 18.73 MB/s
zebra        ██████ 4.37 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 5.6us
zebra        ███████████ 2.1us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 639.88 MB/s | 375ns | 8.0us | 28.1us | 0 |
| horse | 53.33 MB/s | 5.3us | 1.1ms | 3.3ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 639.88 MB/s
horse        ██ 53.33 MB/s
```

**Latency (P50)**
```
zebra        ██ 375ns
horse        ██████████████████████████████ 5.3us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2.71 MB/s | 4.9us | 2.6ms | 161.7ms | 0 |
| zebra | 1.84 MB/s | 3.4us | 237.2us | 272.0ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2.71 MB/s
zebra        ████████████████████ 1.84 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 4.9us
zebra        ████████████████████ 3.4us
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 193.51 MB/s | 72.8ms | 95.8ms | 95.8ms | 0 |
| zebra | 29.90 MB/s | 310.3ms | 310.3ms | 310.3ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 193.51 MB/s
zebra        ████ 29.90 MB/s
```

**Latency (P50)**
```
horse        ███████ 72.8ms
zebra        ██████████████████████████████ 310.3ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 6216.06 MB/s | 125ns | 250ns | 500ns | 0 |
| zebra | 4236.30 MB/s | 125ns | 375ns | 1.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 6216.06 MB/s
zebra        ████████████████████ 4236.30 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 125ns
zebra        ██████████████████████████████ 125ns
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5115.70 MB/s | 125ns | 333ns | 959ns | 0 |
| zebra | 2418.30 MB/s | 167ns | 583ns | 3.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5115.70 MB/s
zebra        ██████████████ 2418.30 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████ 125ns
zebra        ██████████████████████████████ 167ns
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 4606.75 MB/s | 125ns | 417ns | 959ns | 0 |
| zebra | 2413.58 MB/s | 292ns | 625ns | 1.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 4606.75 MB/s
zebra        ███████████████ 2413.58 MB/s
```

**Latency (P50)**
```
horse        ████████████ 125ns
zebra        ██████████████████████████████ 292ns
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 994.12 MB/s | 875ns | 1.4us | 1.8us | 0 |
| zebra | 788.02 MB/s | 958ns | 2.5us | 4.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 994.12 MB/s
zebra        ███████████████████████ 788.02 MB/s
```

**Latency (P50)**
```
horse        ███████████████████████████ 875ns
zebra        ██████████████████████████████ 958ns
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 318.66 MB/s | 1.5us | 4.0us | 20.3us | 0 |
| horse | 107.82 MB/s | 1.5us | 19.8us | 64.0us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 318.66 MB/s
horse        ██████████ 107.82 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 1.5us
horse        █████████████████████████████ 1.5us
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 75.10 MB/s | 2.0us | 6.7us | 137.1us | 0 |
| horse | 43.69 MB/s | 2.8us | 93.5us | 185.0us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 75.10 MB/s
horse        █████████████████ 43.69 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████ 2.0us
horse        ██████████████████████████████ 2.8us
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2490290.98 MB/s | 83ns | 125ns | 292ns | 0 |
| zebra | 1644687.09 MB/s | 84ns | 208ns | 1.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2490290.98 MB/s
zebra        ███████████████████ 1644687.09 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2327537.94 MB/s | 83ns | 167ns | 292ns | 0 |
| zebra | 1933788.02 MB/s | 84ns | 167ns | 1.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2327537.94 MB/s
zebra        ████████████████████████ 1933788.02 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2450908.60 MB/s | 83ns | 125ns | 291ns | 0 |
| zebra | 1479740.09 MB/s | 125ns | 250ns | 709ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2450908.60 MB/s
zebra        ██████████████████ 1479740.09 MB/s
```

**Latency (P50)**
```
horse        ███████████████████ 83ns
zebra        ██████████████████████████████ 125ns
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 88710197.24 MB/s | 125ns | 167ns | 292ns | 0 |
| zebra | 78811175.30 MB/s | 125ns | 167ns | 625ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 88710197.24 MB/s
zebra        ██████████████████████████ 78811175.30 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 125ns
zebra        ██████████████████████████████ 125ns
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 8541.80 MB/s | 84ns | 166ns | 333ns | 0 |
| zebra | 7822.71 MB/s | 84ns | 125ns | 500ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 8541.80 MB/s
zebra        ███████████████████████████ 7822.71 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 84ns
zebra        ██████████████████████████████ 84ns
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 10435143.74 MB/s | 83ns | 125ns | 250ns | 0 |
| zebra | 8858319.51 MB/s | 84ns | 167ns | 333ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 10435143.74 MB/s
zebra        █████████████████████████ 8858319.51 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 653479.94 MB/s | 83ns | 125ns | 250ns | 0 |
| zebra | 527005.01 MB/s | 125ns | 167ns | 333ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 653479.94 MB/s
zebra        ████████████████████████ 527005.01 MB/s
```

**Latency (P50)**
```
horse        ███████████████████ 83ns
zebra        ██████████████████████████████ 125ns
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 545256 ops/s | 1.8us | 1.8us | 1.8us | 0 |
| zebra | 68966 ops/s | 14.5us | 14.5us | 14.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 545256 ops/s
zebra        ███ 68966 ops/s
```

**Latency (P50)**
```
horse        ███ 1.8us
zebra        ██████████████████████████████ 14.5us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 43320 ops/s | 23.1us | 23.1us | 23.1us | 0 |
| horse | 38218 ops/s | 26.2us | 26.2us | 26.2us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 43320 ops/s
horse        ██████████████████████████ 38218 ops/s
```

**Latency (P50)**
```
zebra        ██████████████████████████ 23.1us
horse        ██████████████████████████████ 26.2us
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 6665 ops/s | 150.0us | 150.0us | 150.0us | 0 |
| horse | 4982 ops/s | 200.7us | 200.7us | 200.7us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 6665 ops/s
horse        ██████████████████████ 4982 ops/s
```

**Latency (P50)**
```
zebra        ██████████████████████ 150.0us
horse        ██████████████████████████████ 200.7us
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 723 ops/s | 1.4ms | 1.4ms | 1.4ms | 0 |
| zebra | 564 ops/s | 1.8ms | 1.8ms | 1.8ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 723 ops/s
zebra        ███████████████████████ 564 ops/s
```

**Latency (P50)**
```
horse        ███████████████████████ 1.4ms
zebra        ██████████████████████████████ 1.8ms
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 237586 ops/s | 4.2us | 4.2us | 4.2us | 0 |
| zebra | 1 ops/s | 1.80s | 1.80s | 1.80s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 237586 ops/s
zebra        █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 4.2us
zebra        ██████████████████████████████ 1.80s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 151883 ops/s | 6.6us | 6.6us | 6.6us | 0 |
| zebra | 0 ops/s | 2.28s | 2.28s | 2.28s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 151883 ops/s
zebra        █ 0 ops/s
```

**Latency (P50)**
```
horse        █ 6.6us
zebra        ██████████████████████████████ 2.28s
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 6717 ops/s | 148.9us | 148.9us | 148.9us | 0 |
| zebra | 0 ops/s | 3.63s | 3.63s | 3.63s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 6717 ops/s
zebra        █ 0 ops/s
```

**Latency (P50)**
```
horse        █ 148.9us
zebra        ██████████████████████████████ 3.63s
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1100 ops/s | 909.2us | 909.2us | 909.2us | 0 |
| zebra | 0 ops/s | 3.76s | 3.76s | 3.76s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1100 ops/s
zebra        █ 0 ops/s
```

**Latency (P50)**
```
horse        █ 909.2us
zebra        ██████████████████████████████ 3.76s
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 33.87 MB/s | 7.2us | 7.2us | 7.2us | 0 |
| horse | 13.79 MB/s | 17.7us | 17.7us | 17.7us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 33.87 MB/s
horse        ████████████ 13.79 MB/s
```

**Latency (P50)**
```
zebra        ████████████ 7.2us
horse        ██████████████████████████████ 17.7us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 109.52 MB/s | 22.3us | 22.3us | 22.3us | 0 |
| zebra | 87.72 MB/s | 27.8us | 27.8us | 27.8us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 109.52 MB/s
zebra        ████████████████████████ 87.72 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████ 22.3us
zebra        ██████████████████████████████ 27.8us
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 185.72 MB/s | 131.5us | 131.5us | 131.5us | 0 |
| horse | 129.86 MB/s | 188.0us | 188.0us | 188.0us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 185.72 MB/s
horse        ████████████████████ 129.86 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████ 131.5us
horse        ██████████████████████████████ 188.0us
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 123.85 MB/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| horse | 105.81 MB/s | 2.3ms | 2.3ms | 2.3ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 123.85 MB/s
horse        █████████████████████████ 105.81 MB/s
```

**Latency (P50)**
```
zebra        █████████████████████████ 2.0ms
horse        ██████████████████████████████ 2.3ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 15733682 ops/s | 42ns | 84ns | 208ns | 0 |
| zebra | 12582137 ops/s | 42ns | 125ns | 209ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 15733682 ops/s
zebra        ███████████████████████ 12582137 ops/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 42ns
zebra        ██████████████████████████████ 42ns
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1165.00 MB/s | 267.5us | 55.4ms | 75.1ms | 0 |
| zebra | 306.00 MB/s | 260.0us | 884.5us | 866.1ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1165.00 MB/s
zebra        ███████ 306.00 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 267.5us
zebra        █████████████████████████████ 260.0us
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1774.41 MB/s | 375ns | 833ns | 2.8us | 0 |
| horse | 1350.53 MB/s | 500ns | 1.0us | 1.8us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1774.41 MB/s
horse        ██████████████████████ 1350.53 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████ 375ns
horse        ██████████████████████████████ 500ns
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1250.37 MB/s | 20.6us | 32.7us | 98.8us | 0 |
| horse | 886.63 MB/s | 20.0us | 30.2us | 41.1ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1250.37 MB/s
horse        █████████████████████ 886.63 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 20.6us
horse        █████████████████████████████ 20.0us
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 2045.90 MB/s | 2.2us | 9.0us | 41.3us | 0 |
| horse | 1452.86 MB/s | 2.0us | 3.7us | 6.7us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 2045.90 MB/s
horse        █████████████████████ 1452.86 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 2.2us
horse        ███████████████████████████ 2.0us
```

## Recommendations

- **Write-heavy workloads:** zebra
- **Read-heavy workloads:** horse

---

*Generated by storage benchmark CLI*
