# Storage Benchmark Report

**Generated:** 2026-02-19T13:35:59+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** horse (won 25/40 benchmarks, 62%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | horse | 25 | 62% |
| 2 | zebra | 15 | 38% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | horse | 9.4 GB/s | 2.7x vs zebra |
| Small Write (1KB) | zebra | 1.9 GB/s | +38% vs horse |
| Large Read (10MB) | zebra | 167.2 GB/s | 10.2x vs horse |
| Large Write (10MB) | horse | 979.0 MB/s | +72% vs zebra |
| Delete | zebra | 3.0M ops/s | +52% vs horse |
| Stat | horse | 15.4M ops/s | 2.2x vs zebra |
| List (100 objects) | horse | 135.8K ops/s | 5527.0x vs zebra |
| Copy | zebra | 1.5 GB/s | 2.3x vs horse |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **horse** | 979 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **zebra** | 167211 MB/s | Best for streaming, CDN |
| Small File Operations | **horse** | 5524599 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **zebra** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| horse | 979.0 | 16441.6 | 215.4us | 781.0us |
| zebra | 567.7 | 167211.4 | 308.3us | 167ns |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| horse | 1392309 | 9656889 | 500ns | 83ns |
| zebra | 1916755 | 3537451 | 333ns | 125ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| horse | 15430518 | 135835 | 1967132 |
| zebra | 7014589 | 25 | 2995375 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 792.02 | 111.69 | 24.38 |
| zebra | 1013.88 | 484.53 | 78.93 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 5075.26 | 2597.13 | 3068.01 |
| zebra | 6358.26 | 3637.77 | 2406.43 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 9.5us | 11.6us | 76.9us | 648.8us |
| zebra | 13.4us | 19.4us | 217.2us | 1.2ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 2.8us | 4.6us | 18.2us | 185.0us |
| zebra | 210.9ms | 171.8ms | 170.2ms | 177.7ms |

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
| zebra | 1524.27 MB/s | 458ns | 958ns | 3.8us | 0 |
| horse | 661.14 MB/s | 1.0us | 1.9us | 18.8us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1524.27 MB/s
horse        █████████████ 661.14 MB/s
```

**Latency (P50)**
```
zebra        █████████████ 458ns
horse        ██████████████████████████████ 1.0us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 2995375 ops/s | 333ns | 542ns | 792ns | 0 |
| horse | 1967132 ops/s | 417ns | 833ns | 1.2us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 2995375 ops/s
horse        ███████████████████ 1967132 ops/s
```

**Latency (P50)**
```
zebra        ███████████████████████ 333ns
horse        ██████████████████████████████ 417ns
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 130.41 MB/s | 542ns | 1.2us | 2.0us | 0 |
| zebra | 52.20 MB/s | 625ns | 2.2us | 53.1us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 130.41 MB/s
zebra        ████████████ 52.20 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████ 542ns
zebra        ██████████████████████████████ 625ns
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1094455 ops/s | 500ns | 916ns | 2.2us | 0 |
| zebra | 1035824 ops/s | 417ns | 2.2us | 8.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1094455 ops/s
zebra        ████████████████████████████ 1035824 ops/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 500ns
zebra        █████████████████████████ 417ns
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 104.78 MB/s | 708ns | 1.1us | 2.0us | 0 |
| zebra | 56.90 MB/s | 833ns | 2.4us | 12.9us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 104.78 MB/s
zebra        ████████████████ 56.90 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████ 708ns
zebra        ██████████████████████████████ 833ns
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 135835 ops/s | 6.5us | 12.1us | 28.0us | 0 |
| zebra | 25 ops/s | 38.8ms | 46.1ms | 46.1ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 135835 ops/s
zebra        █ 25 ops/s
```

**Latency (P50)**
```
horse        █ 6.5us
zebra        ██████████████████████████████ 38.8ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 39.75 MB/s | 19.0us | 214.5us | 10.7ms | 0 |
| horse | 9.44 MB/s | 8.0us | 611.4us | 69.6ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 39.75 MB/s
horse        ███████ 9.44 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 19.0us
horse        ████████████ 8.0us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 229.08 MB/s | 13.2us | 197.8us | 833.7us | 0 |
| horse | 130.81 MB/s | 9.4us | 223.4us | 866.9us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 229.08 MB/s
horse        █████████████████ 130.81 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 13.2us
horse        █████████████████████ 9.4us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 12.84 MB/s | 6.0us | 191.2us | 47.6ms | 0 |
| horse | 3.56 MB/s | 4.5us | 57.3ms | 74.3ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 12.84 MB/s
horse        ████████ 3.56 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 6.0us
horse        ██████████████████████ 4.5us
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 452.78 MB/s | 31.7ms | 34.6ms | 37.2ms | 0 |
| horse | 374.15 MB/s | 37.3ms | 46.7ms | 46.7ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 452.78 MB/s
horse        ████████████████████████ 374.15 MB/s
```

**Latency (P50)**
```
zebra        █████████████████████████ 31.7ms
horse        ██████████████████████████████ 37.3ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 6358.26 MB/s | 125ns | 250ns | 458ns | 0 |
| horse | 5075.26 MB/s | 167ns | 292ns | 542ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 6358.26 MB/s
horse        ███████████████████████ 5075.26 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████ 125ns
horse        ██████████████████████████████ 167ns
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 3637.77 MB/s | 167ns | 417ns | 2.2us | 0 |
| horse | 2597.13 MB/s | 250ns | 667ns | 1.4us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 3637.77 MB/s
horse        █████████████████████ 2597.13 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████ 167ns
horse        ██████████████████████████████ 250ns
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 3068.01 MB/s | 208ns | 583ns | 1.3us | 0 |
| zebra | 2406.43 MB/s | 250ns | 667ns | 2.1us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 3068.01 MB/s
zebra        ███████████████████████ 2406.43 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████ 208ns
zebra        ██████████████████████████████ 250ns
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1013.88 MB/s | 875ns | 1.5us | 2.2us | 0 |
| horse | 792.02 MB/s | 1.1us | 1.7us | 2.5us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1013.88 MB/s
horse        ███████████████████████ 792.02 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████████ 875ns
horse        ██████████████████████████████ 1.1us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 484.53 MB/s | 1.1us | 3.9us | 20.1us | 0 |
| horse | 111.69 MB/s | 2.7us | 23.7us | 66.6us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 484.53 MB/s
horse        ██████ 111.69 MB/s
```

**Latency (P50)**
```
zebra        ███████████ 1.1us
horse        ██████████████████████████████ 2.7us
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 78.93 MB/s | 2.0us | 26.2us | 194.7us | 0 |
| horse | 24.38 MB/s | 3.3us | 185.3us | 436.2us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 78.93 MB/s
horse        █████████ 24.38 MB/s
```

**Latency (P50)**
```
zebra        █████████████████ 2.0us
horse        ██████████████████████████████ 3.3us
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2491986.10 MB/s | 83ns | 125ns | 291ns | 0 |
| zebra | 1514077.51 MB/s | 84ns | 292ns | 625ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2491986.10 MB/s
zebra        ██████████████████ 1514077.51 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2332702.84 MB/s | 84ns | 125ns | 292ns | 0 |
| zebra | 2044965.32 MB/s | 84ns | 125ns | 292ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2332702.84 MB/s
zebra        ██████████████████████████ 2044965.32 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 84ns
zebra        ██████████████████████████████ 84ns
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2027689.87 MB/s | 125ns | 167ns | 333ns | 0 |
| zebra | 1781788.64 MB/s | 125ns | 167ns | 917ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2027689.87 MB/s
zebra        ██████████████████████████ 1781788.64 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 125ns
zebra        ██████████████████████████████ 125ns
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 167211.38 MB/s | 167ns | 614.2us | 1.4ms | 0 |
| horse | 16441.62 MB/s | 781.0us | 1.6ms | 2.9ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 167211.38 MB/s
horse        ██ 16441.62 MB/s
```

**Latency (P50)**
```
zebra        █ 167ns
horse        ██████████████████████████████ 781.0us
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 9430.56 MB/s | 83ns | 125ns | 292ns | 0 |
| zebra | 3454.54 MB/s | 125ns | 292ns | 2.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 9430.56 MB/s
zebra        ██████████ 3454.54 MB/s
```

**Latency (P50)**
```
horse        ███████████████████ 83ns
zebra        ██████████████████████████████ 125ns
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 697486.43 MB/s | 125ns | 250ns | 9.2us | 0 |
| horse | 19650.72 MB/s | 71.8us | 110.7us | 224.0us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 697486.43 MB/s
horse        █ 19650.72 MB/s
```

**Latency (P50)**
```
zebra        █ 125ns
horse        ██████████████████████████████ 71.8us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 632129.47 MB/s | 83ns | 125ns | 292ns | 0 |
| zebra | 513062.51 MB/s | 84ns | 125ns | 1.1us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 632129.47 MB/s
zebra        ████████████████████████ 513062.51 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 705716 ops/s | 1.4us | 1.4us | 1.4us | 0 |
| zebra | 155836 ops/s | 6.4us | 6.4us | 6.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 705716 ops/s
zebra        ██████ 155836 ops/s
```

**Latency (P50)**
```
horse        ██████ 1.4us
zebra        ██████████████████████████████ 6.4us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 242424 ops/s | 4.1us | 4.1us | 4.1us | 0 |
| zebra | 40201 ops/s | 24.9us | 24.9us | 24.9us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 242424 ops/s
zebra        ████ 40201 ops/s
```

**Latency (P50)**
```
horse        ████ 4.1us
zebra        ██████████████████████████████ 24.9us
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 34384 ops/s | 29.1us | 29.1us | 29.1us | 0 |
| zebra | 9401 ops/s | 106.4us | 106.4us | 106.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 34384 ops/s
zebra        ████████ 9401 ops/s
```

**Latency (P50)**
```
horse        ████████ 29.1us
zebra        ██████████████████████████████ 106.4us
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 3523 ops/s | 283.9us | 283.9us | 283.9us | 0 |
| zebra | 1456 ops/s | 686.8us | 686.8us | 686.8us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 3523 ops/s
zebra        ████████████ 1456 ops/s
```

**Latency (P50)**
```
horse        ████████████ 283.9us
zebra        ██████████████████████████████ 686.8us
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 363636 ops/s | 2.8us | 2.8us | 2.8us | 0 |
| zebra | 5 ops/s | 210.9ms | 210.9ms | 210.9ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 363636 ops/s
zebra        █ 5 ops/s
```

**Latency (P50)**
```
horse        █ 2.8us
zebra        ██████████████████████████████ 210.9ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 218150 ops/s | 4.6us | 4.6us | 4.6us | 0 |
| zebra | 6 ops/s | 171.8ms | 171.8ms | 171.8ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 218150 ops/s
zebra        █ 6 ops/s
```

**Latency (P50)**
```
horse        █ 4.6us
zebra        ██████████████████████████████ 171.8ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 54795 ops/s | 18.2us | 18.2us | 18.2us | 0 |
| zebra | 6 ops/s | 170.2ms | 170.2ms | 170.2ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 54795 ops/s
zebra        █ 6 ops/s
```

**Latency (P50)**
```
horse        █ 18.2us
zebra        ██████████████████████████████ 170.2ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5404 ops/s | 185.0us | 185.0us | 185.0us | 0 |
| zebra | 6 ops/s | 177.7ms | 177.7ms | 177.7ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5404 ops/s
zebra        █ 6 ops/s
```

**Latency (P50)**
```
horse        █ 185.0us
zebra        ██████████████████████████████ 177.7ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 25.59 MB/s | 9.5us | 9.5us | 9.5us | 0 |
| zebra | 18.20 MB/s | 13.4us | 13.4us | 13.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 25.59 MB/s
zebra        █████████████████████ 18.20 MB/s
```

**Latency (P50)**
```
horse        █████████████████████ 9.5us
zebra        ██████████████████████████████ 13.4us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 210.76 MB/s | 11.6us | 11.6us | 11.6us | 0 |
| zebra | 126.01 MB/s | 19.4us | 19.4us | 19.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 210.76 MB/s
zebra        █████████████████ 126.01 MB/s
```

**Latency (P50)**
```
horse        █████████████████ 11.6us
zebra        ██████████████████████████████ 19.4us
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 317.58 MB/s | 76.9us | 76.9us | 76.9us | 0 |
| zebra | 112.40 MB/s | 217.2us | 217.2us | 217.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 317.58 MB/s
zebra        ██████████ 112.40 MB/s
```

**Latency (P50)**
```
horse        ██████████ 76.9us
zebra        ██████████████████████████████ 217.2us
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 376.30 MB/s | 648.8us | 648.8us | 648.8us | 0 |
| zebra | 211.84 MB/s | 1.2ms | 1.2ms | 1.2ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 376.30 MB/s
zebra        ████████████████ 211.84 MB/s
```

**Latency (P50)**
```
horse        ████████████████ 648.8us
zebra        ██████████████████████████████ 1.2ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 15430518 ops/s | 42ns | 84ns | 250ns | 0 |
| zebra | 7014589 ops/s | 83ns | 84ns | 1.7us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 15430518 ops/s
zebra        █████████████ 7014589 ops/s
```

**Latency (P50)**
```
horse        ███████████████ 42ns
zebra        ██████████████████████████████ 83ns
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 979.05 MB/s | 215.4us | 39.1ms | 204.8ms | 0 |
| zebra | 567.74 MB/s | 308.3us | 34.0ms | 463.3ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 979.05 MB/s
zebra        █████████████████ 567.74 MB/s
```

**Latency (P50)**
```
horse        ████████████████████ 215.4us
zebra        ██████████████████████████████ 308.3us
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1871.83 MB/s | 333ns | 1.1us | 2.3us | 0 |
| horse | 1359.68 MB/s | 500ns | 1.0us | 1.9us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1871.83 MB/s
horse        █████████████████████ 1359.68 MB/s
```

**Latency (P50)**
```
zebra        ███████████████████ 333ns
horse        ██████████████████████████████ 500ns
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1071.66 MB/s | 19.6us | 25.4us | 40.9ms | 0 |
| zebra | 316.62 MB/s | 23.0us | 94.0us | 230.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1071.66 MB/s
zebra        ████████ 316.62 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████ 19.6us
zebra        ██████████████████████████████ 23.0us
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 2207.18 MB/s | 2.1us | 9.1us | 29.8us | 0 |
| horse | 849.34 MB/s | 1.8us | 2.8us | 3.8us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 2207.18 MB/s
horse        ███████████ 849.34 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 2.1us
horse        █████████████████████████ 1.8us
```

## Recommendations

- **Write-heavy workloads:** zebra
- **Read-heavy workloads:** zebra

---

*Generated by storage benchmark CLI*
