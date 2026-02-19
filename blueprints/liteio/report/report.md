# Storage Benchmark Report

**Generated:** 2026-02-19T07:31:08+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** turtle (won 43/48 benchmarks, 90%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | turtle | 43 | 90% |
| 2 | local | 5 | 10% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | local | 7.0 GB/s | close |
| Small Write (1KB) | turtle | 75.9 MB/s | 34.0x vs local |
| Large Read (100MB) | turtle | 825265.6 GB/s | 55365.0x vs local |
| Large Write (100MB) | local | 1.1 GB/s | 4.1x vs turtle |
| Delete | turtle | 2.3M ops/s | 102.3x vs local |
| Stat | turtle | 12.6M ops/s | 2.4x vs local |
| List (100 objects) | turtle | 92.5K ops/s | 20.1x vs local |
| Copy | turtle | 1.0 GB/s | 577.2x vs local |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **local** | 1135 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **turtle** | 825265554 MB/s | Best for streaming, CDN |
| Small File Operations | **local** | 3602255 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **turtle** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| local | 1135.1 | 14905.9 | 83.5ms | 6.5ms |
| turtle | 273.6 | 825265554.2 | 341.8ms | 125ns |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| local | 2288 | 7202222 | 280.4us | 125ns |
| turtle | 77720 | 7101381 | 500ns | 125ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| local | 5299367 | 4610 | 22601 |
| turtle | 12611206 | 92513 | 2312230 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| local | 2.99 | 0.73 | 0.20 | 0.11 | 0.05 | 0.02 |
| turtle | 26.54 | 65.28 | 1.91 | 25.12 | 0.97 | 1.76 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| local | 1534.85 | 944.35 | 894.08 | 752.00 | 611.21 | 327.36 |
| turtle | 4491.77 | 3468.30 | 3425.86 | 3188.91 | 2775.49 | 2506.44 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| local | 71.9ms | 176.6ms | 548.6ms | 4.33s |
| turtle | 21.2us | 254.0us | 788.0us | 14.0ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| local | 115.5us | 323.3us | 3.2ms | 32.7ms |
| turtle | 8.2us | 24.4us | 210.2us | 3.6ms |

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
| turtle | 1025.53 MB/s | 542ns | 1.2us | 7.2us | 0 |
| local | 1.78 MB/s | 333.7us | 1.5ms | 1.8ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1025.53 MB/s
local        █ 1.78 MB/s
```

**Latency (P50)**
```
turtle       █ 542ns
local        ██████████████████████████████ 333.7us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 2312230 ops/s | 375ns | 542ns | 750ns | 0 |
| local | 22601 ops/s | 43.8us | 54.0us | 69.2us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 2312230 ops/s
local        █ 22601 ops/s
```

**Latency (P50)**
```
turtle       █ 375ns
local        ██████████████████████████████ 43.8us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 80.75 MB/s | 500ns | 1.1us | 2.8us | 0 |
| local | 0.11 MB/s | 329.0us | 4.7ms | 8.1ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 80.75 MB/s
local        █ 0.11 MB/s
```

**Latency (P50)**
```
turtle       █ 500ns
local        ██████████████████████████████ 329.0us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| local | 2678 ops/s | 226.8us | 1.5ms | 2.3ms | 0 |
| turtle | 665 ops/s | 709ns | 11.4us | 28.0us | 0 |

**Throughput**
```
local        ██████████████████████████████ 2678 ops/s
turtle       ███████ 665 ops/s
```

**Latency (P50)**
```
local        ██████████████████████████████ 226.8us
turtle       █ 709ns
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 54.24 MB/s | 708ns | 1.6us | 7.6us | 0 |
| local | 0.16 MB/s | 292.6us | 663.6us | 8.0ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 54.24 MB/s
local        █ 0.16 MB/s
```

**Latency (P50)**
```
turtle       █ 708ns
local        ██████████████████████████████ 292.6us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 92513 ops/s | 9.4us | 16.5us | 34.0us | 0 |
| local | 4610 ops/s | 215.0us | 229.2us | 244.0us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 92513 ops/s
local        █ 4610 ops/s
```

**Latency (P50)**
```
turtle       █ 9.4us
local        ██████████████████████████████ 215.0us
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 1.48 MB/s | 50.2us | 47.4ms | 81.4ms | 0 |
| local | 0.69 MB/s | 606.0us | 70.1ms | 88.5ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1.48 MB/s
local        █████████████ 0.69 MB/s
```

**Latency (P50)**
```
turtle       ██ 50.2us
local        ██████████████████████████████ 606.0us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 10.40 MB/s | 167ns | 8.4ms | 39.3ms | 0 |
| local | 4.42 MB/s | 2.0us | 35.1ms | 54.6ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 10.40 MB/s
local        ████████████ 4.42 MB/s
```

**Latency (P50)**
```
turtle       ██ 167ns
local        ██████████████████████████████ 2.0us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 0.84 MB/s | 12.6ms | 58.8ms | 89.9ms | 0 |
| local | 0.32 MB/s | 48.1ms | 102.5ms | 131.1ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 0.84 MB/s
local        ███████████ 0.32 MB/s
```

**Latency (P50)**
```
turtle       ███████ 12.6ms
local        ██████████████████████████████ 48.1ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 338.45 MB/s | 42.2ms | 57.4ms | 72.6ms | 0 |
| local | 77.02 MB/s | 102.3ms | 277.8ms | 277.8ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 338.45 MB/s
local        ██████ 77.02 MB/s
```

**Latency (P50)**
```
turtle       ████████████ 42.2ms
local        ██████████████████████████████ 102.3ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 4491.77 MB/s | 167ns | 416ns | 1.2us | 0 |
| local | 1534.85 MB/s | 584ns | 834ns | 1.1us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 4491.77 MB/s
local        ██████████ 1534.85 MB/s
```

**Latency (P50)**
```
turtle       ████████ 167ns
local        ██████████████████████████████ 584ns
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 3468.30 MB/s | 167ns | 625ns | 1.7us | 0 |
| local | 944.35 MB/s | 792ns | 2.0us | 4.6us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 3468.30 MB/s
local        ████████ 944.35 MB/s
```

**Latency (P50)**
```
turtle       ██████ 167ns
local        ██████████████████████████████ 792ns
```

### ParallelRead/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 2775.49 MB/s | 208ns | 792ns | 2.4us | 0 |
| local | 611.21 MB/s | 916ns | 3.1us | 15.5us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 2775.49 MB/s
local        ██████ 611.21 MB/s
```

**Latency (P50)**
```
turtle       ██████ 208ns
local        ██████████████████████████████ 916ns
```

### ParallelRead/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 2506.44 MB/s | 208ns | 834ns | 3.0us | 0 |
| local | 327.36 MB/s | 1.1us | 2.9us | 16.0us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 2506.44 MB/s
local        ███ 327.36 MB/s
```

**Latency (P50)**
```
turtle       █████ 208ns
local        ██████████████████████████████ 1.1us
```

### ParallelRead/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 3425.86 MB/s | 167ns | 625ns | 1.5us | 0 |
| local | 894.08 MB/s | 792ns | 2.0us | 7.6us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 3425.86 MB/s
local        ███████ 894.08 MB/s
```

**Latency (P50)**
```
turtle       ██████ 167ns
local        ██████████████████████████████ 792ns
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 3188.91 MB/s | 167ns | 667ns | 1.6us | 0 |
| local | 752.00 MB/s | 833ns | 2.6us | 12.5us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 3188.91 MB/s
local        ███████ 752.00 MB/s
```

**Latency (P50)**
```
turtle       ██████ 167ns
local        ██████████████████████████████ 833ns
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 26.54 MB/s | 583ns | 2.6us | 28.2us | 0 |
| local | 2.99 MB/s | 259.9us | 658.7us | 1.4ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 26.54 MB/s
local        ███ 2.99 MB/s
```

**Latency (P50)**
```
turtle       █ 583ns
local        ██████████████████████████████ 259.9us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 65.28 MB/s | 1.3us | 33.8us | 74.3us | 0 |
| local | 0.73 MB/s | 1.3ms | 2.3ms | 2.8ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 65.28 MB/s
local        █ 0.73 MB/s
```

**Latency (P50)**
```
turtle       █ 1.3us
local        ██████████████████████████████ 1.3ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 0.97 MB/s | 4.8us | 546.5us | 1.6ms | 0 |
| local | 0.05 MB/s | 20.3ms | 39.3ms | 52.9ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 0.97 MB/s
local        █ 0.05 MB/s
```

**Latency (P50)**
```
turtle       █ 4.8us
local        ██████████████████████████████ 20.3ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 1.76 MB/s | 6.0us | 797.2us | 1.6ms | 0 |
| local | 0.02 MB/s | 43.6ms | 75.8ms | 89.8ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1.76 MB/s
local        █ 0.02 MB/s
```

**Latency (P50)**
```
turtle       █ 6.0us
local        ██████████████████████████████ 43.6ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 1.91 MB/s | 1.0us | 296.2us | 3.4ms | 0 |
| local | 0.20 MB/s | 4.6ms | 8.7ms | 13.5ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1.91 MB/s
local        ███ 0.20 MB/s
```

**Latency (P50)**
```
turtle       █ 1.0us
local        ██████████████████████████████ 4.6ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 25.12 MB/s | 3.3us | 167.4us | 352.5us | 0 |
| local | 0.11 MB/s | 9.1ms | 14.9ms | 17.5ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 25.12 MB/s
local        █ 0.11 MB/s
```

**Latency (P50)**
```
turtle       █ 3.3us
local        ██████████████████████████████ 9.1ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 1927906.95 MB/s | 125ns | 250ns | 334ns | 0 |
| local | 23666.66 MB/s | 10.0us | 12.5us | 18.1us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1927906.95 MB/s
local        █ 23666.66 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 10.0us
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 1908363.96 MB/s | 125ns | 250ns | 334ns | 0 |
| local | 11316.70 MB/s | 14.6us | 47.8us | 66.3us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1908363.96 MB/s
local        █ 11316.70 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 14.6us
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 1838166.03 MB/s | 125ns | 250ns | 416ns | 0 |
| local | 10973.58 MB/s | 14.6us | 52.7us | 79.0us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1838166.03 MB/s
local        █ 10973.58 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 14.6us
```

### Read/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 825265554.15 MB/s | 125ns | 167ns | 333ns | 0 |
| local | 14905.91 MB/s | 6.5ms | 7.9ms | 9.3ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 825265554.15 MB/s
local        █ 14905.91 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 6.5ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 77683290.15 MB/s | 125ns | 208ns | 334ns | 0 |
| local | 11184.37 MB/s | 741.0us | 1.6ms | 3.1ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 77683290.15 MB/s
local        █ 11184.37 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 741.0us
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| local | 7033.42 MB/s | 125ns | 208ns | 334ns | 0 |
| turtle | 6934.94 MB/s | 125ns | 167ns | 459ns | 0 |

**Throughput**
```
local        ██████████████████████████████ 7033.42 MB/s
turtle       █████████████████████████████ 6934.94 MB/s
```

**Latency (P50)**
```
local        ██████████████████████████████ 125ns
turtle       ██████████████████████████████ 125ns
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 7907301.28 MB/s | 125ns | 167ns | 333ns | 0 |
| local | 95058.30 MB/s | 10.2us | 12.0us | 15.0us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 7907301.28 MB/s
local        █ 95058.30 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 10.2us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 476298.14 MB/s | 125ns | 208ns | 334ns | 0 |
| local | 22644.16 MB/s | 2.0us | 5.6us | 9.8us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 476298.14 MB/s
local        █ 22644.16 MB/s
```

**Latency (P50)**
```
turtle       █ 125ns
local        ██████████████████████████████ 2.0us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 263783 ops/s | 3.8us | 3.8us | 3.8us | 0 |
| local | 1767 ops/s | 565.8us | 565.8us | 565.8us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 263783 ops/s
local        █ 1767 ops/s
```

**Latency (P50)**
```
turtle       █ 3.8us
local        ██████████████████████████████ 565.8us
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 39868 ops/s | 25.1us | 25.1us | 25.1us | 0 |
| local | 266 ops/s | 3.8ms | 3.8ms | 3.8ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 39868 ops/s
local        █ 266 ops/s
```

**Latency (P50)**
```
turtle       █ 25.1us
local        ██████████████████████████████ 3.8ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 1519 ops/s | 658.1us | 658.1us | 658.1us | 0 |
| local | 20 ops/s | 50.5ms | 50.5ms | 50.5ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 1519 ops/s
local        █ 20 ops/s
```

**Latency (P50)**
```
turtle       █ 658.1us
local        ██████████████████████████████ 50.5ms
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 163 ops/s | 6.1ms | 6.1ms | 6.1ms | 0 |
| local | 2 ops/s | 645.4ms | 645.4ms | 645.4ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 163 ops/s
local        █ 2 ops/s
```

**Latency (P50)**
```
turtle       █ 6.1ms
local        ██████████████████████████████ 645.4ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 122444 ops/s | 8.2us | 8.2us | 8.2us | 0 |
| local | 8658 ops/s | 115.5us | 115.5us | 115.5us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 122444 ops/s
local        ██ 8658 ops/s
```

**Latency (P50)**
```
turtle       ██ 8.2us
local        ██████████████████████████████ 115.5us
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 40957 ops/s | 24.4us | 24.4us | 24.4us | 0 |
| local | 3093 ops/s | 323.3us | 323.3us | 323.3us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 40957 ops/s
local        ██ 3093 ops/s
```

**Latency (P50)**
```
turtle       ██ 24.4us
local        ██████████████████████████████ 323.3us
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 4758 ops/s | 210.2us | 210.2us | 210.2us | 0 |
| local | 309 ops/s | 3.2ms | 3.2ms | 3.2ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 4758 ops/s
local        █ 309 ops/s
```

**Latency (P50)**
```
turtle       █ 210.2us
local        ██████████████████████████████ 3.2ms
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 277 ops/s | 3.6ms | 3.6ms | 3.6ms | 0 |
| local | 31 ops/s | 32.7ms | 32.7ms | 32.7ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 277 ops/s
local        ███ 31 ops/s
```

**Latency (P50)**
```
turtle       ███ 3.6ms
local        ██████████████████████████████ 32.7ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 114.89 MB/s | 21.2us | 21.2us | 21.2us | 0 |
| local | 0.03 MB/s | 71.9ms | 71.9ms | 71.9ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 114.89 MB/s
local        █ 0.03 MB/s
```

**Latency (P50)**
```
turtle       █ 21.2us
local        ██████████████████████████████ 71.9ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 96.13 MB/s | 254.0us | 254.0us | 254.0us | 0 |
| local | 0.14 MB/s | 176.6ms | 176.6ms | 176.6ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 96.13 MB/s
local        █ 0.14 MB/s
```

**Latency (P50)**
```
turtle       █ 254.0us
local        ██████████████████████████████ 176.6ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 309.81 MB/s | 788.0us | 788.0us | 788.0us | 0 |
| local | 0.45 MB/s | 548.6ms | 548.6ms | 548.6ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 309.81 MB/s
local        █ 0.45 MB/s
```

**Latency (P50)**
```
turtle       █ 788.0us
local        ██████████████████████████████ 548.6ms
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 174.94 MB/s | 14.0ms | 14.0ms | 14.0ms | 0 |
| local | 0.56 MB/s | 4.33s | 4.33s | 4.33s | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 174.94 MB/s
local        █ 0.56 MB/s
```

**Latency (P50)**
```
turtle       █ 14.0ms
local        ██████████████████████████████ 4.33s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 12611206 ops/s | 83ns | 125ns | 208ns | 0 |
| local | 5299367 ops/s | 125ns | 334ns | 1.2us | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 12611206 ops/s
local        ████████████ 5299367 ops/s
```

**Latency (P50)**
```
turtle       ███████████████████ 83ns
local        ██████████████████████████████ 125ns
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| local | 1135.09 MB/s | 83.5ms | 107.5ms | 118.5ms | 0 |
| turtle | 273.64 MB/s | 341.8ms | 341.8ms | 341.8ms | 0 |

**Throughput**
```
local        ██████████████████████████████ 1135.09 MB/s
turtle       ███████ 273.64 MB/s
```

**Latency (P50)**
```
local        ███████ 83.5ms
turtle       ██████████████████████████████ 341.8ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| local | 1068.81 MB/s | 6.8ms | 19.1ms | 28.6ms | 0 |
| turtle | 210.23 MB/s | 18.8ms | 114.3ms | 320.1ms | 0 |

**Throughput**
```
local        ██████████████████████████████ 1068.81 MB/s
turtle       █████ 210.23 MB/s
```

**Latency (P50)**
```
local        ██████████ 6.8ms
turtle       ██████████████████████████████ 18.8ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 75.90 MB/s | 500ns | 1.5us | 20.5us | 0 |
| local | 2.23 MB/s | 280.4us | 1.1ms | 1.4ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 75.90 MB/s
local        █ 2.23 MB/s
```

**Latency (P50)**
```
turtle       █ 500ns
local        ██████████████████████████████ 280.4us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| local | 639.80 MB/s | 1.6ms | 2.6ms | 3.1ms | 0 |
| turtle | 550.03 MB/s | 1.2ms | 4.7ms | 8.0ms | 0 |

**Throughput**
```
local        ██████████████████████████████ 639.80 MB/s
turtle       █████████████████████████ 550.03 MB/s
```

**Latency (P50)**
```
local        ██████████████████████████████ 1.6ms
turtle       ██████████████████████ 1.2ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| turtle | 412.50 MB/s | 22.5us | 367.8us | 2.2ms | 0 |
| local | 149.58 MB/s | 313.2us | 916.6us | 1.3ms | 0 |

**Throughput**
```
turtle       ██████████████████████████████ 412.50 MB/s
local        ██████████ 149.58 MB/s
```

**Latency (P50)**
```
turtle       ██ 22.5us
local        ██████████████████████████████ 313.2us
```

## Recommendations

- **Write-heavy workloads:** local
- **Read-heavy workloads:** turtle

---

*Generated by storage benchmark CLI*
