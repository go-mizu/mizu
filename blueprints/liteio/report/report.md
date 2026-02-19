# Storage Benchmark Report

**Generated:** 2026-02-19T03:49:39+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** turtle (won 44/48 benchmarks, 92%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | turtle | 44 | 92% |
| 2 | local | 4 | 8% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | local | 7.4 GB/s | close |
| Small Write (1KB) | turtle | 147.8 MB/s | 55.8x vs local |
| Large Read (100MB) | turtle | 781684.1 GB/s | 46819.9x vs local |
| Large Write (100MB) | local | 2.8 GB/s | 4.2x vs turtle |
| Delete | turtle | 133.0K ops/s | 5.5x vs local |
| Stat | turtle | 12.5M ops/s | 2.3x vs local |
| List (100 objects) | turtle | 89.2K ops/s | 19.3x vs local |
| Copy | turtle | 29.9 MB/s | 12.4x vs local |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **local** | 2835 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **turtle** | 781684120 MB/s | Best for streaming, CDN |
| Small File Operations | **local** | 3787436 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **turtle** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| local | 2835.1 | 16695.6 | 34.2ms | 6.0ms |
| turtle | 667.4 | 781684120.0 | 166.2ms | 125ns |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| local | 2714 | 7572159 | 284.4us | 125ns |
| turtle | 151386 | 7054685 | 458ns | 125ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| local | 5552269 | 4629 | 24084 |
| turtle | 12515612 | 89232 | 133004 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| local | 2.13 | 0.73 | 0.21 | 0.09 | 0.05 | 0.02 |
| turtle | 272.20 | 6.48 | 68.89 | 1.68 | 9.62 | 6.76 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| local | 1533.90 | 986.23 | 902.10 | 756.09 | 636.71 | 580.11 |
| turtle | 4649.05 | 3511.70 | 3426.36 | 3062.09 | 2883.45 | 2616.17 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| local | 3.4ms | 37.3ms | 377.1ms | 4.11s |
| turtle | 21.5us | 123.2us | 1.4ms | 14.6ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| local | 100.6us | 329.4us | 3.2ms | 28.4ms |
| turtle | 6.5us | 24.5us | 265.4us | 3.7ms |

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

- **local** (48 benchmarks)
- **turtle** (48 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 29.95 MB/s | 542ns | 1.7us | 10.8us | 0 |
| local | 2.42 MB/s | 360.3us | 523.6us | 1.6ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 29.95 MB/s
local        ██ 2.42 MB/s
```

**Latency (P50)**
```
turtle       █ 542ns
local        ██████████████████████████████ 360.3us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 133004 ops/s | 375ns | 542ns | 750ns | 0 |
| local | 24084 ops/s | 41.2us | 50.0us | 54.8us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 133004 ops/s
local        █████ 24084 ops/s
```

**Latency (P50)**
```
turtle       █ 375ns
local        ██████████████████████████████ 41.2us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 85.86 MB/s | 542ns | 1.0us | 4.4us | 0 |
| local | 0.22 MB/s | 381.1us | 630.6us | 1.8ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 85.86 MB/s
local        █ 0.22 MB/s
```

**Latency (P50)**
```
turtle       █ 542ns
local        ██████████████████████████████ 381.1us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 1872177 ops/s | 417ns | 708ns | 1.4us | 0 |
| local | 2474 ops/s | 274.0us | 1.4ms | 2.3ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1872177 ops/s
local        █ 2474 ops/s
```

**Latency (P50)**
```
turtle       █ 417ns
local        ██████████████████████████████ 274.0us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 5.14 MB/s | 750ns | 1.6us | 4.4us | 0 |
| local | 0.24 MB/s | 366.3us | 532.0us | 1.6ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 5.14 MB/s
local        █ 0.24 MB/s
```

**Latency (P50)**
```
turtle       █ 750ns
local        ██████████████████████████████ 366.3us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 89232 ops/s | 10.0us | 16.7us | 33.7us | 0 |
| local | 4629 ops/s | 214.8us | 226.2us | 233.4us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 89232 ops/s
local        █ 4629 ops/s
```

**Latency (P50)**
```
turtle       █ 10.0us
local        ██████████████████████████████ 214.8us
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 1.35 MB/s | 76.6us | 52.1ms | 88.4ms | 0 |
| local | 0.52 MB/s | 74.9us | 94.5ms | 128.4ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1.35 MB/s
local        ███████████ 0.52 MB/s
```

**Latency (P50)**
```
turtle       ██████████████████████████████ 76.6us
local        █████████████████████████████ 74.9us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 7.36 MB/s | 208ns | 14.3ms | 50.2ms | 0 |
| local | 2.78 MB/s | 2.5us | 51.4ms | 94.9ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 7.36 MB/s
local        ███████████ 2.78 MB/s
```

**Latency (P50)**
```
turtle       ██ 208ns
local        ██████████████████████████████ 2.5us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 0.77 MB/s | 14.5ms | 61.8ms | 96.0ms | 0 |
| local | 0.27 MB/s | 55.2ms | 115.6ms | 132.2ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 0.77 MB/s
local        ██████████ 0.27 MB/s
```

**Latency (P50)**
```
turtle       ███████ 14.5ms
local        ██████████████████████████████ 55.2ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 400.06 MB/s | 34.8ms | 52.6ms | 67.2ms | 0 |
| local | 242.44 MB/s | 49.6ms | 75.6ms | 76.9ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 400.06 MB/s
local        ██████████████████ 242.44 MB/s
```

**Latency (P50)**
```
turtle       █████████████████████ 34.8ms
local        ██████████████████████████████ 49.6ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 4649.05 MB/s | 167ns | 375ns | 1.2us | 0 |
| local | 1533.90 MB/s | 584ns | 833ns | 1.1us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 4649.05 MB/s
local        █████████ 1533.90 MB/s
```

**Latency (P50)**
```
turtle       ████████ 167ns
local        ██████████████████████████████ 584ns
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 3511.70 MB/s | 167ns | 542ns | 1.5us | 0 |
| local | 986.23 MB/s | 750ns | 1.7us | 4.5us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 3511.70 MB/s
local        ████████ 986.23 MB/s
```

**Latency (P50)**
```
turtle       ██████ 167ns
local        ██████████████████████████████ 750ns
```

### ParallelRead/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 2883.45 MB/s | 208ns | 667ns | 1.8us | 0 |
| local | 636.71 MB/s | 834ns | 3.0us | 16.0us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 2883.45 MB/s
local        ██████ 636.71 MB/s
```

**Latency (P50)**
```
turtle       ███████ 208ns
local        ██████████████████████████████ 834ns
```

### ParallelRead/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 2616.17 MB/s | 208ns | 792ns | 2.8us | 0 |
| local | 580.11 MB/s | 834ns | 3.0us | 18.2us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 2616.17 MB/s
local        ██████ 580.11 MB/s
```

**Latency (P50)**
```
turtle       ███████ 208ns
local        ██████████████████████████████ 834ns
```

### ParallelRead/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 3426.36 MB/s | 208ns | 583ns | 1.5us | 0 |
| local | 902.10 MB/s | 791ns | 2.0us | 8.2us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 3426.36 MB/s
local        ███████ 902.10 MB/s
```

**Latency (P50)**
```
turtle       ███████ 208ns
local        ██████████████████████████████ 791ns
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 3062.09 MB/s | 208ns | 666ns | 1.7us | 0 |
| local | 756.09 MB/s | 792ns | 2.5us | 12.9us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 3062.09 MB/s
local        ███████ 756.09 MB/s
```

**Latency (P50)**
```
turtle       ███████ 208ns
local        ██████████████████████████████ 792ns
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 272.20 MB/s | 542ns | 2.7us | 49.7us | 0 |
| local | 2.13 MB/s | 292.6us | 1.2ms | 2.3ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 272.20 MB/s
local        █ 2.13 MB/s
```

**Latency (P50)**
```
turtle       █ 542ns
local        ██████████████████████████████ 292.6us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 6.48 MB/s | 1.4us | 33.8us | 78.5us | 0 |
| local | 0.73 MB/s | 1.3ms | 2.2ms | 2.8ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 6.48 MB/s
local        ███ 0.73 MB/s
```

**Latency (P50)**
```
turtle       █ 1.4us
local        ██████████████████████████████ 1.3ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 9.62 MB/s | 4.2us | 515.9us | 1.5ms | 0 |
| local | 0.05 MB/s | 21.8ms | 34.4ms | 38.9ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 9.62 MB/s
local        █ 0.05 MB/s
```

**Latency (P50)**
```
turtle       █ 4.2us
local        ██████████████████████████████ 21.8ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 6.76 MB/s | 5.4us | 655.3us | 1.3ms | 0 |
| local | 0.02 MB/s | 44.2ms | 67.4ms | 77.1ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 6.76 MB/s
local        █ 0.02 MB/s
```

**Latency (P50)**
```
turtle       █ 5.4us
local        ██████████████████████████████ 44.2ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 68.89 MB/s | 2.2us | 61.5us | 114.8us | 0 |
| local | 0.21 MB/s | 4.6ms | 7.4ms | 8.7ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 68.89 MB/s
local        █ 0.21 MB/s
```

**Latency (P50)**
```
turtle       █ 2.2us
local        ██████████████████████████████ 4.6ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 1.68 MB/s | 2.9us | 239.2us | 650.0us | 0 |
| local | 0.09 MB/s | 11.0ms | 17.4ms | 19.9ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1.68 MB/s
local        █ 0.09 MB/s
```

**Latency (P50)**
```
turtle       █ 2.9us
local        ██████████████████████████████ 11.0ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 1948081.56 MB/s | 125ns | 250ns | 333ns | 0 |
| local | 16115.87 MB/s | 15.4us | 19.6us | 24.9us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1948081.56 MB/s
local        █ 16115.87 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 15.4us
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 1928906.62 MB/s | 125ns | 250ns | 334ns | 0 |
| local | 16307.41 MB/s | 15.2us | 19.3us | 24.0us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1928906.62 MB/s
local        █ 16307.41 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 15.2us
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 1842515.15 MB/s | 125ns | 209ns | 375ns | 0 |
| local | 16310.45 MB/s | 15.3us | 19.4us | 24.0us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1842515.15 MB/s
local        █ 16310.45 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 15.3us
```

### Read/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 781684119.98 MB/s | 125ns | 167ns | 333ns | 0 |
| local | 16695.55 MB/s | 6.0ms | 6.5ms | 6.8ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 781684119.98 MB/s
local        █ 16695.55 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 6.0ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 77141479.52 MB/s | 125ns | 167ns | 333ns | 0 |
| local | 13148.92 MB/s | 708.1us | 1.1ms | 1.2ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 77141479.52 MB/s
local        █ 13148.92 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 708.1us
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| local | 7394.69 MB/s | 125ns | 208ns | 333ns | 0 |
| turtle | 6889.34 MB/s | 125ns | 208ns | 459ns | 0 |

**Throughput**
```
local        ██████████████████████████████ 7394.69 MB/s
turtle       ███████████████████████████ 6889.34 MB/s
```

**Latency (P50)**
```
local        ██████████████████████████████ 125ns
turtle       ██████████████████████████████ 125ns
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 7574532.06 MB/s | 125ns | 208ns | 334ns | 0 |
| local | 96100.21 MB/s | 10.1us | 12.3us | 14.0us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 7574532.06 MB/s
local        █ 96100.21 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 10.1us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 468106.72 MB/s | 125ns | 208ns | 334ns | 0 |
| local | 29616.59 MB/s | 1.9us | 2.5us | 6.0us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 468106.72 MB/s
local        █ 29616.59 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 1.9us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 263783 ops/s | 3.8us | 3.8us | 3.8us | 0 |
| local | 2223 ops/s | 449.8us | 449.8us | 449.8us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 263783 ops/s
local        █ 2223 ops/s
```

**Latency (P50)**
```
turtle       █ 3.8us
local        ██████████████████████████████ 449.8us
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 36584 ops/s | 27.3us | 27.3us | 27.3us | 0 |
| local | 266 ops/s | 3.8ms | 3.8ms | 3.8ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 36584 ops/s
local        █ 266 ops/s
```

**Latency (P50)**
```
turtle       █ 27.3us
local        ██████████████████████████████ 3.8ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 3567 ops/s | 280.4us | 280.4us | 280.4us | 0 |
| local | 19 ops/s | 52.8ms | 52.8ms | 52.8ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 3567 ops/s
local        █ 19 ops/s
```

**Latency (P50)**
```
turtle       █ 280.4us
local        ██████████████████████████████ 52.8ms
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 271 ops/s | 3.7ms | 3.7ms | 3.7ms | 0 |
| local | 2 ops/s | 536.2ms | 536.2ms | 536.2ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 271 ops/s
local        █ 2 ops/s
```

**Latency (P50)**
```
turtle       █ 3.7ms
local        ██████████████████████████████ 536.2ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 153846 ops/s | 6.5us | 6.5us | 6.5us | 0 |
| local | 9938 ops/s | 100.6us | 100.6us | 100.6us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 153846 ops/s
local        █ 9938 ops/s
```

**Latency (P50)**
```
turtle       █ 6.5us
local        ██████████████████████████████ 100.6us
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 40885 ops/s | 24.5us | 24.5us | 24.5us | 0 |
| local | 3036 ops/s | 329.4us | 329.4us | 329.4us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 40885 ops/s
local        ██ 3036 ops/s
```

**Latency (P50)**
```
turtle       ██ 24.5us
local        ██████████████████████████████ 329.4us
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 3768 ops/s | 265.4us | 265.4us | 265.4us | 0 |
| local | 314 ops/s | 3.2ms | 3.2ms | 3.2ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 3768 ops/s
local        ██ 314 ops/s
```

**Latency (P50)**
```
turtle       ██ 265.4us
local        ██████████████████████████████ 3.2ms
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 270 ops/s | 3.7ms | 3.7ms | 3.7ms | 0 |
| local | 35 ops/s | 28.4ms | 28.4ms | 28.4ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 270 ops/s
local        ███ 35 ops/s
```

**Latency (P50)**
```
turtle       ███ 3.7ms
local        ██████████████████████████████ 28.4ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 113.55 MB/s | 21.5us | 21.5us | 21.5us | 0 |
| local | 0.73 MB/s | 3.4ms | 3.4ms | 3.4ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 113.55 MB/s
local        █ 0.73 MB/s
```

**Latency (P50)**
```
turtle       █ 21.5us
local        ██████████████████████████████ 3.4ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 198.09 MB/s | 123.2us | 123.2us | 123.2us | 0 |
| local | 0.65 MB/s | 37.3ms | 37.3ms | 37.3ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 198.09 MB/s
local        █ 0.65 MB/s
```

**Latency (P50)**
```
turtle       █ 123.2us
local        ██████████████████████████████ 37.3ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 176.19 MB/s | 1.4ms | 1.4ms | 1.4ms | 0 |
| local | 0.65 MB/s | 377.1ms | 377.1ms | 377.1ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 176.19 MB/s
local        █ 0.65 MB/s
```

**Latency (P50)**
```
turtle       █ 1.4ms
local        ██████████████████████████████ 377.1ms
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 167.74 MB/s | 14.6ms | 14.6ms | 14.6ms | 0 |
| local | 0.59 MB/s | 4.11s | 4.11s | 4.11s | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 167.74 MB/s
local        █ 0.59 MB/s
```

**Latency (P50)**
```
turtle       █ 14.6ms
local        ██████████████████████████████ 4.11s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 12515612 ops/s | 83ns | 125ns | 208ns | 0 |
| local | 5552269 ops/s | 125ns | 333ns | 1.0us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 12515612 ops/s
local        █████████████ 5552269 ops/s
```

**Latency (P50)**
```
turtle       ███████████████████ 83ns
local        ██████████████████████████████ 125ns
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| local | 2835.12 MB/s | 34.2ms | 44.3ms | 59.4ms | 0 |
| turtle | 667.40 MB/s | 166.2ms | 184.0ms | 184.0ms | 0 |

**Throughput**
```
local        ██████████████████████████████ 2835.12 MB/s
turtle       ███████ 667.40 MB/s
```

**Latency (P50)**
```
local        ██████ 34.2ms
turtle       ██████████████████████████████ 166.2ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| local | 2733.91 MB/s | 3.5ms | 4.5ms | 6.6ms | 0 |
| turtle | 759.55 MB/s | 12.0ms | 19.5ms | 27.3ms | 0 |

**Throughput**
```
local        ██████████████████████████████ 2733.91 MB/s
turtle       ████████ 759.55 MB/s
```

**Latency (P50)**
```
local        ████████ 3.5ms
turtle       ██████████████████████████████ 12.0ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 147.84 MB/s | 458ns | 1.1us | 4.6us | 0 |
| local | 2.65 MB/s | 284.4us | 990.5us | 1.3ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 147.84 MB/s
local        █ 2.65 MB/s
```

**Latency (P50)**
```
turtle       █ 458ns
local        ██████████████████████████████ 284.4us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| local | 1528.09 MB/s | 652.6us | 791.2us | 1.0ms | 0 |
| turtle | 647.55 MB/s | 935.1us | 4.6ms | 5.8ms | 0 |

**Throughput**
```
local        ██████████████████████████████ 1528.09 MB/s
turtle       ████████████ 647.55 MB/s
```

**Latency (P50)**
```
local        ████████████████████ 652.6us
turtle       ██████████████████████████████ 935.1us
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 576.70 MB/s | 22.4us | 355.3us | 917.4us | 0 |
| local | 148.99 MB/s | 332.4us | 897.0us | 1.3ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 576.70 MB/s
local        ███████ 148.99 MB/s
```

**Latency (P50)**
```
turtle       ██ 22.4us
local        ██████████████████████████████ 332.4us
```

## Recommendations

- **Write-heavy workloads:** local
- **Read-heavy workloads:** turtle

---

*Generated by storage benchmark CLI*
