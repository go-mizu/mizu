# Storage Benchmark Report

**Generated:** 2026-02-19T00:46:44+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** devnull_s3 (won 33/48 benchmarks, 69%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | devnull_s3 | 33 | 69% |
| 2 | liteio | 15 | 31% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | devnull_s3 | 16.9 MB/s | +14% vs liteio |
| Small Write (1KB) | devnull_s3 | 15.5 MB/s | 16.7x vs liteio |
| Large Read (100MB) | devnull_s3 | 8.6 GB/s | 3.9x vs liteio |
| Large Write (100MB) | devnull_s3 | 1.5 GB/s | +80% vs liteio |
| Delete | devnull_s3 | 18.8K ops/s | +98% vs liteio |
| Stat | devnull_s3 | 18.1K ops/s | +77% vs liteio |
| List (100 objects) | devnull_s3 | 1.4K ops/s | +21% vs liteio |
| Copy | devnull_s3 | 15.5 MB/s | 6.6x vs liteio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **devnull_s3** | 1530 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **devnull_s3** | 8627 MB/s | Best for streaming, CDN |
| Small File Operations | **devnull_s3** | 16612 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| devnull_s3 | 1530.3 | 8626.8 | 64.8ms | 11.2ms |
| liteio | 850.3 | 2225.2 | 107.0ms | 45.2ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| devnull_s3 | 15918 | 17307 | 60.6us | 56.5us |
| liteio | 953 | 15199 | 1.3ms | 56.9us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| devnull_s3 | 18061 | 1410 | 18781 |
| liteio | 10229 | 1164 | 9493 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| devnull_s3 | 9.78 | 4.13 | 1.85 | 0.67 | 0.55 | 0.09 |
| liteio | 1.96 | 0.47 | 0.20 | 0.09 | 0.04 | 0.02 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| devnull_s3 | 10.17 | 0.00* | 0.00* | 0.00* | 0.00* | 0.00* |
| liteio | 9.61 | 4.26 | 1.68 | 1.00 | 0.68 | 0.35 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| devnull_s3 | 1.5ms | 20.0ms | 186.7ms | 1.21s |
| liteio | 3.7ms | 36.7ms | 351.5ms | 3.81s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| devnull_s3 | 0ns* | 0ns* | 0ns* | 0ns* |
| liteio | 216.0us | 789.6us | 6.9ms | 322.9ms |

*\* indicates errors occurred*

### Warnings

- **devnull_s3**: 443114 errors during benchmarks

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

- **devnull_s3** (48 benchmarks)
- **liteio** (48 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 15.46 MB/s | 60.8us | 74.0us | 108.2us | 0 |
| liteio | 2.35 MB/s | 353.4us | 796.8us | 1.0ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 15.46 MB/s
liteio       ████ 2.35 MB/s
```

**Latency (P50)**
```
devnull_s3   █████ 60.8us
liteio       ██████████████████████████████ 353.4us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 18781 ops/s | 52.0us | 61.2us | 80.0us | 0 |
| liteio | 9493 ops/s | 100.1us | 130.2us | 259.7us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 18781 ops/s
liteio       ███████████████ 9493 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████████ 52.0us
liteio       ██████████████████████████████ 100.1us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1.48 MB/s | 61.0us | 77.3us | 137.1us | 0 |
| liteio | 0.23 MB/s | 359.1us | 586.0us | 922.1us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.48 MB/s
liteio       ████ 0.23 MB/s
```

**Latency (P50)**
```
devnull_s3   █████ 61.0us
liteio       ██████████████████████████████ 359.1us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 16148 ops/s | 58.7us | 79.1us | 130.0us | 0 |
| liteio | 3327 ops/s | 274.9us | 428.7us | 681.5us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 16148 ops/s
liteio       ██████ 3327 ops/s
```

**Latency (P50)**
```
devnull_s3   ██████ 58.7us
liteio       ██████████████████████████████ 274.9us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1.44 MB/s | 64.3us | 77.4us | 114.5us | 0 |
| liteio | 0.25 MB/s | 370.2us | 497.9us | 595.5us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.44 MB/s
liteio       █████ 0.25 MB/s
```

**Latency (P50)**
```
devnull_s3   █████ 64.3us
liteio       ██████████████████████████████ 370.2us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1410 ops/s | 699.7us | 759.8us | 937.8us | 0 |
| liteio | 1164 ops/s | 710.2us | 1.7ms | 2.5ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1410 ops/s
liteio       ████████████████████████ 1164 ops/s
```

**Latency (P50)**
```
devnull_s3   █████████████████████████████ 699.7us
liteio       ██████████████████████████████ 710.2us
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 4.20 MB/s | 2.6ms | 10.5ms | 20.4ms | 25414 |
| liteio | 0.53 MB/s | 17.3ms | 90.9ms | 122.9ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 4.20 MB/s
liteio       ███ 0.53 MB/s
```

**Latency (P50)**
```
devnull_s3   ████ 2.6ms
liteio       ██████████████████████████████ 17.3ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 4.78 MB/s | 2.4ms | 9.0ms | 16.3ms | 57644 |
| liteio | 2.71 MB/s | 3.7ms | 17.3ms | 35.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 4.78 MB/s
liteio       █████████████████ 2.71 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 2.4ms
liteio       ██████████████████████████████ 3.7ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 4.05 MB/s | 2.6ms | 11.5ms | 23.0ms | 4769 |
| liteio | 0.26 MB/s | 59.2ms | 113.3ms | 140.3ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 4.05 MB/s
liteio       █ 0.26 MB/s
```

**Latency (P50)**
```
devnull_s3   █ 2.6ms
liteio       ██████████████████████████████ 59.2ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1272.23 MB/s | 11.7ms | 13.1ms | 13.7ms | 0 |
| liteio | 214.80 MB/s | 63.4ms | 98.4ms | 104.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1272.23 MB/s
liteio       █████ 214.80 MB/s
```

**Latency (P50)**
```
devnull_s3   █████ 11.7ms
liteio       ██████████████████████████████ 63.4ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 10.17 MB/s | 94.7us | 105.6us | 117.6us | 0 |
| liteio | 9.61 MB/s | 94.6us | 144.4us | 270.0us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 10.17 MB/s
liteio       ████████████████████████████ 9.61 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████████████████ 94.7us
liteio       █████████████████████████████ 94.6us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.26 MB/s | 215.6us | 368.6us | 559.5us | 0 |
| devnull_s3 | 0.00 MB/s | 0ns | 0ns | 0ns | 49871 |

**Throughput**
```
liteio       ██████████████████████████████ 4.26 MB/s
devnull_s3    0.00 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 215.6us
devnull_s3    0ns
```

### ParallelRead/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.68 MB/s | 994.2us | 4.1ms | 7.0ms | 0 |
| devnull_s3 | 0.00 MB/s | 0ns | 0ns | 0ns | 48923 |

**Throughput**
```
liteio       ██████████████████████████████ 0.68 MB/s
devnull_s3    0.00 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 994.2us
devnull_s3    0ns
```

### ParallelRead/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.35 MB/s | 2.1ms | 7.4ms | 13.0ms | 0 |
| devnull_s3 | 0.00 MB/s | 0ns | 0ns | 0ns | 59064 |

**Throughput**
```
liteio       ██████████████████████████████ 0.35 MB/s
devnull_s3    0.00 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 2.1ms
devnull_s3    0ns
```

### ParallelRead/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.68 MB/s | 428.7us | 1.4ms | 2.7ms | 0 |
| devnull_s3 | 0.00 MB/s | 0ns | 0ns | 0ns | 60336 |

**Throughput**
```
liteio       ██████████████████████████████ 1.68 MB/s
devnull_s3    0.00 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 428.7us
devnull_s3    0ns
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.00 MB/s | 684.2us | 2.7ms | 5.0ms | 0 |
| devnull_s3 | 0.00 MB/s | 0ns | 0ns | 0ns | 55248 |

**Throughput**
```
liteio       ██████████████████████████████ 1.00 MB/s
devnull_s3    0.00 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 684.2us
devnull_s3    0ns
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 9.78 MB/s | 98.0us | 110.2us | 138.7us | 0 |
| liteio | 1.96 MB/s | 360.3us | 1.2ms | 2.6ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 9.78 MB/s
liteio       ██████ 1.96 MB/s
```

**Latency (P50)**
```
devnull_s3   ████████ 98.0us
liteio       ██████████████████████████████ 360.3us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 4.13 MB/s | 223.8us | 370.9us | 564.6us | 0 |
| liteio | 0.47 MB/s | 1.6ms | 2.8ms | 4.9ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 4.13 MB/s
liteio       ███ 0.47 MB/s
```

**Latency (P50)**
```
devnull_s3   ████ 223.8us
liteio       ██████████████████████████████ 1.6ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.55 MB/s | 1.2ms | 5.3ms | 8.3ms | 0 |
| liteio | 0.04 MB/s | 22.3ms | 39.6ms | 52.8ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.55 MB/s
liteio       ██ 0.04 MB/s
```

**Latency (P50)**
```
devnull_s3   █ 1.2ms
liteio       ██████████████████████████████ 22.3ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.09 MB/s | 3.6ms | 37.8ms | 143.8ms | 0 |
| liteio | 0.02 MB/s | 43.5ms | 77.8ms | 94.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.09 MB/s
liteio       ██████ 0.02 MB/s
```

**Latency (P50)**
```
devnull_s3   ██ 3.6ms
liteio       ██████████████████████████████ 43.5ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1.85 MB/s | 424.0us | 1.2ms | 1.9ms | 0 |
| liteio | 0.20 MB/s | 4.6ms | 8.6ms | 12.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.85 MB/s
liteio       ███ 0.20 MB/s
```

**Latency (P50)**
```
devnull_s3   ██ 424.0us
liteio       ██████████████████████████████ 4.6ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.67 MB/s | 890.6us | 4.3ms | 9.3ms | 0 |
| liteio | 0.09 MB/s | 10.2ms | 17.6ms | 21.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.67 MB/s
liteio       ████ 0.09 MB/s
```

**Latency (P50)**
```
devnull_s3   ██ 890.6us
liteio       ██████████████████████████████ 10.2ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2215.63 MB/s | 111.3us | 125.7us | 152.0us | 0 |
| devnull_s3 | 0.00 MB/s | 0ns | 0ns | 0ns | 27511 |

**Throughput**
```
liteio       ██████████████████████████████ 2215.63 MB/s
devnull_s3    0.00 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 111.3us
devnull_s3    0ns
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2210.67 MB/s | 111.5us | 126.3us | 155.8us | 0 |
| devnull_s3 | 0.00 MB/s | 0ns | 0ns | 0ns | 27507 |

**Throughput**
```
liteio       ██████████████████████████████ 2210.67 MB/s
devnull_s3    0.00 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 111.5us
devnull_s3    0ns
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2223.40 MB/s | 110.9us | 125.1us | 143.2us | 0 |
| devnull_s3 | 0.00 MB/s | 0ns | 0ns | 0ns | 26823 |

**Throughput**
```
liteio       ██████████████████████████████ 2223.40 MB/s
devnull_s3    0.00 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 110.9us
devnull_s3    0ns
```

### Read/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 8626.77 MB/s | 11.2ms | 13.8ms | 19.5ms | 0 |
| liteio | 2225.22 MB/s | 45.2ms | 51.2ms | 52.8ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 8626.77 MB/s
liteio       ███████ 2225.22 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████ 11.2ms
liteio       ██████████████████████████████ 45.2ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 5928.83 MB/s | 1.3ms | 3.5ms | 4.3ms | 0 |
| liteio | 2331.12 MB/s | 4.1ms | 6.9ms | 10.1ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 5928.83 MB/s
liteio       ███████████ 2331.12 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████ 1.3ms
liteio       ██████████████████████████████ 4.1ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 16.90 MB/s | 56.5us | 66.2us | 80.2us | 0 |
| liteio | 14.84 MB/s | 56.9us | 106.4us | 267.0us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 16.90 MB/s
liteio       ██████████████████████████ 14.84 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████████████████ 56.5us
liteio       ██████████████████████████████ 56.9us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 5708.81 MB/s | 167.1us | 220.8us | 321.3us | 0 |
| liteio | 2796.12 MB/s | 257.2us | 740.4us | 1.1ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 5708.81 MB/s
liteio       ██████████████ 2796.12 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 167.1us
liteio       ██████████████████████████████ 257.2us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 972.16 MB/s | 63.2us | 73.3us | 84.9us | 0 |
| liteio | 907.81 MB/s | 63.5us | 93.7us | 188.8us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 972.16 MB/s
liteio       ████████████████████████████ 907.81 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████████████████ 63.2us
liteio       ██████████████████████████████ 63.5us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 894 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| devnull_s3 | 814 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 894 ops/s
devnull_s3   ███████████████████████████ 814 ops/s
```

**Latency (P50)**
```
liteio       ███████████████████████████ 1.1ms
devnull_s3   ██████████████████████████████ 1.2ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 99 ops/s | 10.1ms | 10.1ms | 10.1ms | 0 |
| devnull_s3 | 45 ops/s | 22.1ms | 22.1ms | 22.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 99 ops/s
devnull_s3   █████████████ 45 ops/s
```

**Latency (P50)**
```
liteio       █████████████ 10.1ms
devnull_s3   ██████████████████████████████ 22.1ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 10 ops/s | 101.2ms | 101.2ms | 101.2ms | 0 |
| devnull_s3 | 7 ops/s | 140.0ms | 140.0ms | 140.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 10 ops/s
devnull_s3   █████████████████████ 7 ops/s
```

**Latency (P50)**
```
liteio       █████████████████████ 101.2ms
devnull_s3   ██████████████████████████████ 140.0ms
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 2 ops/s | 590.1ms | 590.1ms | 590.1ms | 0 |
| liteio | 1 ops/s | 1.04s | 1.04s | 1.04s | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 2 ops/s
liteio       █████████████████ 1 ops/s
```

**Latency (P50)**
```
devnull_s3   █████████████████ 590.1ms
liteio       ██████████████████████████████ 1.04s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4629 ops/s | 216.0us | 216.0us | 216.0us | 0 |
| devnull_s3 | 0 ops/s | 0ns | 0ns | 0ns | 1 |

**Throughput**
```
liteio       ██████████████████████████████ 4629 ops/s
devnull_s3    0 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 216.0us
devnull_s3    0ns
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1266 ops/s | 789.6us | 789.6us | 789.6us | 0 |
| devnull_s3 | 0 ops/s | 0ns | 0ns | 0ns | 1 |

**Throughput**
```
liteio       ██████████████████████████████ 1266 ops/s
devnull_s3    0 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 789.6us
devnull_s3    0ns
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 146 ops/s | 6.9ms | 6.9ms | 6.9ms | 0 |
| devnull_s3 | 0 ops/s | 0ns | 0ns | 0ns | 1 |

**Throughput**
```
liteio       ██████████████████████████████ 146 ops/s
devnull_s3    0 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 6.9ms
devnull_s3    0ns
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3 ops/s | 322.9ms | 322.9ms | 322.9ms | 0 |
| devnull_s3 | 0 ops/s | 0ns | 0ns | 0ns | 1 |

**Throughput**
```
liteio       ██████████████████████████████ 3 ops/s
devnull_s3    0 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 322.9ms
devnull_s3    0ns
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1.60 MB/s | 1.5ms | 1.5ms | 1.5ms | 0 |
| liteio | 0.66 MB/s | 3.7ms | 3.7ms | 3.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.60 MB/s
liteio       ████████████ 0.66 MB/s
```

**Latency (P50)**
```
devnull_s3   ████████████ 1.5ms
liteio       ██████████████████████████████ 3.7ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1.22 MB/s | 20.0ms | 20.0ms | 20.0ms | 0 |
| liteio | 0.67 MB/s | 36.7ms | 36.7ms | 36.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.22 MB/s
liteio       ████████████████ 0.67 MB/s
```

**Latency (P50)**
```
devnull_s3   ████████████████ 20.0ms
liteio       ██████████████████████████████ 36.7ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1.31 MB/s | 186.7ms | 186.7ms | 186.7ms | 0 |
| liteio | 0.69 MB/s | 351.5ms | 351.5ms | 351.5ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.31 MB/s
liteio       ███████████████ 0.69 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████ 186.7ms
liteio       ██████████████████████████████ 351.5ms
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 2.02 MB/s | 1.21s | 1.21s | 1.21s | 0 |
| liteio | 0.64 MB/s | 3.81s | 3.81s | 3.81s | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 2.02 MB/s
liteio       █████████ 0.64 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████ 1.21s
liteio       ██████████████████████████████ 3.81s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 18061 ops/s | 54.3us | 63.3us | 74.6us | 0 |
| liteio | 10229 ops/s | 61.8us | 263.4us | 434.0us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 18061 ops/s
liteio       ████████████████ 10229 ops/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████████████ 54.3us
liteio       ██████████████████████████████ 61.8us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1530.30 MB/s | 64.8ms | 67.4ms | 67.4ms | 0 |
| liteio | 850.33 MB/s | 107.0ms | 166.8ms | 166.8ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1530.30 MB/s
liteio       ████████████████ 850.33 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████ 64.8ms
liteio       ██████████████████████████████ 107.0ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1301.98 MB/s | 7.6ms | 8.8ms | 8.8ms | 0 |
| liteio | 955.64 MB/s | 9.6ms | 14.3ms | 25.6ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1301.98 MB/s
liteio       ██████████████████████ 955.64 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████████ 7.6ms
liteio       ██████████████████████████████ 9.6ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 15.54 MB/s | 60.6us | 74.0us | 118.0us | 0 |
| liteio | 0.93 MB/s | 1.3ms | 1.8ms | 2.1ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 15.54 MB/s
liteio       █ 0.93 MB/s
```

**Latency (P50)**
```
devnull_s3   █ 60.6us
liteio       ██████████████████████████████ 1.3ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1238.09 MB/s | 780.0us | 1.0ms | 1.1ms | 0 |
| liteio | 706.13 MB/s | 1.2ms | 2.2ms | 3.2ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1238.09 MB/s
liteio       █████████████████ 706.13 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 780.0us
liteio       ██████████████████████████████ 1.2ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 566.59 MB/s | 95.4us | 129.5us | 406.8us | 0 |
| liteio | 150.63 MB/s | 391.6us | 568.8us | 804.3us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 566.59 MB/s
liteio       ███████ 150.63 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████ 95.4us
liteio       ██████████████████████████████ 391.6us
```

## Recommendations

- **Write-heavy workloads:** devnull_s3
- **Read-heavy workloads:** devnull_s3

---

*Generated by storage benchmark CLI*
