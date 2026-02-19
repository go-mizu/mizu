# Storage Benchmark Report

**Generated:** 2026-02-19T09:16:39+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** horse (won 39/40 benchmarks, 98%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | horse | 39 | 98% |
| 2 | minio | 1 | 2% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | horse | 8.7 GB/s | 1617.1x vs minio |
| Small Write (1KB) | horse | 1.4 GB/s | 674.3x vs rustfs |
| Large Read (10MB) | horse | 96848.4 GB/s | 30076.6x vs minio |
| Large Write (10MB) | horse | 1.5 GB/s | 3.7x vs minio |
| Delete | horse | 2.6M ops/s | 1136.9x vs minio |
| Stat | horse | 17.0M ops/s | 2683.8x vs minio |
| List (100 objects) | horse | 138.9K ops/s | 322.0x vs minio |
| Copy | horse | 1.1 GB/s | 567.8x vs minio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **horse** | 1493 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **horse** | 96848378 MB/s | Best for streaming, CDN |
| Small File Operations | **horse** | 5196588 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **horse** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| horse | 1492.6 | 96848378.4 | 240.3us | 84ns |
| minio | 408.8 | 3220.1 | 24.4ms | 3.0ms |
| rustfs | 349.0 | 1791.4 | 28.1ms | 5.5ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| horse | 1451951 | 8941225 | 459ns | 83ns |
| minio | 2145 | 5529 | 450.9us | 178.7us |
| rustfs | 2153 | 4590 | 448.4us | 212.3us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| horse | 16965122 | 138925 | 2597673 |
| minio | 6321 | 431 | 2285 |
| rustfs | 5969 | 314 | 1802 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 1017.30 | 176.26 | 41.47 |
| minio | 1.97 | 0.53 | 0.11 |
| rustfs | 1.90 | 0.40 | 0.08 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 6501.12 | 5414.61 | 4748.65 |
| minio | 4.45 | 1.55 | 0.50 |
| rustfs | 3.84 | 0.46 | 0.10 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 8.7us | 10.6us | 75.2us | 590.5us |
| minio | 510.0us | 5.4ms | 46.6ms | 463.3ms |
| rustfs | 561.9us | 4.9ms | 48.5ms | 470.8ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 3.7us | 3.8us | 15.8us | 172.3us |
| minio | 479.2us | 784.4us | 3.4ms | 25.9ms |
| rustfs | 851.3us | 1.4ms | 4.8ms | 36.5ms |

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
- **minio** (40 benchmarks)
- **rustfs** (40 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1088.73 MB/s | 583ns | 1.0us | 1.9us | 0 |
| minio | 1.92 MB/s | 493.4us | 616.7us | 686.8us | 0 |
| rustfs | 1.55 MB/s | 528.0us | 677.5us | 2.1ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1088.73 MB/s
minio        █ 1.92 MB/s
rustfs       █ 1.55 MB/s
```

**Latency (P50)**
```
horse        █ 583ns
minio        ████████████████████████████ 493.4us
rustfs       ██████████████████████████████ 528.0us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2597673 ops/s | 375ns | 541ns | 708ns | 0 |
| minio | 2285 ops/s | 429.5us | 497.8us | 532.5us | 0 |
| rustfs | 1802 ops/s | 544.2us | 627.0us | 740.7us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2597673 ops/s
minio        █ 2285 ops/s
rustfs       █ 1802 ops/s
```

**Latency (P50)**
```
horse        █ 375ns
minio        ███████████████████████ 429.5us
rustfs       ██████████████████████████████ 544.2us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 147.75 MB/s | 500ns | 834ns | 1.2us | 0 |
| rustfs | 0.20 MB/s | 470.1us | 566.0us | 709.3us | 0 |
| minio | 0.19 MB/s | 473.3us | 584.1us | 624.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 147.75 MB/s
rustfs       █ 0.20 MB/s
minio        █ 0.19 MB/s
```

**Latency (P50)**
```
horse        █ 500ns
rustfs       █████████████████████████████ 470.1us
minio        ██████████████████████████████ 473.3us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1655715 ops/s | 458ns | 875ns | 1.5us | 0 |
| minio | 2226 ops/s | 434.4us | 530.4us | 573.0us | 0 |
| rustfs | 2089 ops/s | 468.5us | 562.7us | 637.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1655715 ops/s
minio        █ 2226 ops/s
rustfs       █ 2089 ops/s
```

**Latency (P50)**
```
horse        █ 458ns
minio        ███████████████████████████ 434.4us
rustfs       ██████████████████████████████ 468.5us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 105.61 MB/s | 667ns | 1.5us | 2.5us | 0 |
| rustfs | 0.19 MB/s | 481.5us | 577.2us | 688.0us | 0 |
| minio | 0.17 MB/s | 518.5us | 607.4us | 691.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 105.61 MB/s
rustfs       █ 0.19 MB/s
minio        █ 0.17 MB/s
```

**Latency (P50)**
```
horse        █ 667ns
rustfs       ███████████████████████████ 481.5us
minio        ██████████████████████████████ 518.5us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 138925 ops/s | 6.5us | 11.8us | 14.8us | 0 |
| minio | 431 ops/s | 2.3ms | 2.4ms | 2.7ms | 0 |
| rustfs | 314 ops/s | 3.1ms | 3.5ms | 3.8ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 138925 ops/s
minio        █ 431 ops/s
rustfs       █ 314 ops/s
```

**Latency (P50)**
```
horse        █ 6.5us
minio        ██████████████████████ 2.3ms
rustfs       ██████████████████████████████ 3.1ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 12.36 MB/s | 1.9us | 146.8us | 53.5ms | 0 |
| minio | 0.80 MB/s | 12.3ms | 61.0ms | 89.5ms | 0 |
| rustfs | 0.36 MB/s | 37.8ms | 61.7ms | 66.3ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 12.36 MB/s
minio        █ 0.80 MB/s
rustfs       █ 0.36 MB/s
```

**Latency (P50)**
```
horse        █ 1.9us
minio        █████████ 12.3ms
rustfs       ██████████████████████████████ 37.8ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 98.13 MB/s | 209ns | 3.6us | 19.2us | 0 |
| minio | 1.66 MB/s | 7.7ms | 23.4ms | 42.4ms | 0 |
| rustfs | 0.38 MB/s | 40.3ms | 70.8ms | 75.6ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 98.13 MB/s
minio        █ 1.66 MB/s
rustfs       █ 0.38 MB/s
```

**Latency (P50)**
```
horse        █ 209ns
minio        █████ 7.7ms
rustfs       ██████████████████████████████ 40.3ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 6.25 MB/s | 4.5us | 1.7ms | 58.9ms | 0 |
| minio | 0.51 MB/s | 26.8ms | 74.5ms | 115.0ms | 0 |
| rustfs | 0.34 MB/s | 49.3ms | 57.9ms | 69.6ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 6.25 MB/s
minio        ██ 0.51 MB/s
rustfs       █ 0.34 MB/s
```

**Latency (P50)**
```
horse        █ 4.5us
minio        ████████████████ 26.8ms
rustfs       ██████████████████████████████ 49.3ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 346.23 MB/s | 42.8ms | 43.5ms | 43.5ms | 0 |
| rustfs | 312.46 MB/s | 47.3ms | 50.4ms | 50.4ms | 0 |
| horse | 4.01 MB/s | 41.0ms | 67.4ms | 67.4ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 346.23 MB/s
rustfs       ███████████████████████████ 312.46 MB/s
horse        █ 4.01 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 42.8ms
rustfs       ██████████████████████████████ 47.3ms
horse        █████████████████████████ 41.0ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 6501.12 MB/s | 125ns | 250ns | 459ns | 0 |
| minio | 4.45 MB/s | 216.5us | 240.7us | 281.1us | 0 |
| rustfs | 3.84 MB/s | 248.1us | 296.1us | 330.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 6501.12 MB/s
minio        █ 4.45 MB/s
rustfs       █ 3.84 MB/s
```

**Latency (P50)**
```
horse        █ 125ns
minio        ██████████████████████████ 216.5us
rustfs       ██████████████████████████████ 248.1us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5414.61 MB/s | 125ns | 375ns | 875ns | 0 |
| minio | 1.55 MB/s | 427.7us | 1.4ms | 2.8ms | 0 |
| rustfs | 0.46 MB/s | 2.1ms | 3.1ms | 3.6ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5414.61 MB/s
minio        █ 1.55 MB/s
rustfs       █ 0.46 MB/s
```

**Latency (P50)**
```
horse        █ 125ns
minio        ██████ 427.7us
rustfs       ██████████████████████████████ 2.1ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 4748.65 MB/s | 125ns | 417ns | 875ns | 0 |
| minio | 0.50 MB/s | 1.4ms | 5.2ms | 10.8ms | 0 |
| rustfs | 0.10 MB/s | 10.1ms | 12.1ms | 13.3ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 4748.65 MB/s
minio        █ 0.50 MB/s
rustfs       █ 0.10 MB/s
```

**Latency (P50)**
```
horse        █ 125ns
minio        ████ 1.4ms
rustfs       ██████████████████████████████ 10.1ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1017.30 MB/s | 875ns | 1.3us | 1.8us | 0 |
| minio | 1.97 MB/s | 482.0us | 571.4us | 646.4us | 0 |
| rustfs | 1.90 MB/s | 498.5us | 607.0us | 786.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1017.30 MB/s
minio        █ 1.97 MB/s
rustfs       █ 1.90 MB/s
```

**Latency (P50)**
```
horse        █ 875ns
minio        █████████████████████████████ 482.0us
rustfs       ██████████████████████████████ 498.5us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 176.26 MB/s | 1.5us | 16.3us | 52.3us | 0 |
| minio | 0.53 MB/s | 1.6ms | 3.3ms | 7.9ms | 0 |
| rustfs | 0.40 MB/s | 2.4ms | 3.6ms | 5.1ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 176.26 MB/s
minio        █ 0.53 MB/s
rustfs       █ 0.40 MB/s
```

**Latency (P50)**
```
horse        █ 1.5us
minio        ████████████████████ 1.6ms
rustfs       ██████████████████████████████ 2.4ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 41.47 MB/s | 2.1us | 84.8us | 184.2us | 0 |
| minio | 0.11 MB/s | 6.2ms | 22.6ms | 40.9ms | 0 |
| rustfs | 0.08 MB/s | 11.6ms | 15.2ms | 32.3ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 41.47 MB/s
minio        █ 0.11 MB/s
rustfs       █ 0.08 MB/s
```

**Latency (P50)**
```
horse        █ 2.1us
minio        ████████████████ 6.2ms
rustfs       ██████████████████████████████ 11.6ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2423976.66 MB/s | 83ns | 166ns | 292ns | 0 |
| minio | 545.95 MB/s | 444.2us | 501.5us | 829.7us | 0 |
| rustfs | 403.97 MB/s | 612.0us | 697.5us | 769.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2423976.66 MB/s
minio        █ 545.95 MB/s
rustfs       █ 403.97 MB/s
```

**Latency (P50)**
```
horse        █ 83ns
minio        █████████████████████ 444.2us
rustfs       ██████████████████████████████ 612.0us
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2530466.39 MB/s | 83ns | 125ns | 292ns | 0 |
| minio | 547.24 MB/s | 443.0us | 498.9us | 715.9us | 0 |
| rustfs | 404.75 MB/s | 610.9us | 706.0us | 773.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2530466.39 MB/s
minio        █ 547.24 MB/s
rustfs       █ 404.75 MB/s
```

**Latency (P50)**
```
horse        █ 83ns
minio        █████████████████████ 443.0us
rustfs       ██████████████████████████████ 610.9us
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2369522.00 MB/s | 83ns | 167ns | 333ns | 0 |
| minio | 620.19 MB/s | 380.7us | 460.8us | 899.6us | 0 |
| rustfs | 409.76 MB/s | 605.4us | 686.2us | 733.6us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2369522.00 MB/s
minio        █ 620.19 MB/s
rustfs       █ 409.76 MB/s
```

**Latency (P50)**
```
horse        █ 83ns
minio        ██████████████████ 380.7us
rustfs       ██████████████████████████████ 605.4us
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 96848378.42 MB/s | 84ns | 125ns | 291ns | 0 |
| minio | 3220.06 MB/s | 3.0ms | 3.5ms | 3.8ms | 0 |
| rustfs | 1791.39 MB/s | 5.5ms | 6.3ms | 6.5ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 96848378.42 MB/s
minio        █ 3220.06 MB/s
rustfs       █ 1791.39 MB/s
```

**Latency (P50)**
```
horse        █ 84ns
minio        ████████████████ 3.0ms
rustfs       ██████████████████████████████ 5.5ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 8731.66 MB/s | 83ns | 125ns | 375ns | 0 |
| minio | 5.40 MB/s | 178.7us | 207.8us | 230.2us | 0 |
| rustfs | 4.48 MB/s | 212.3us | 254.5us | 286.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 8731.66 MB/s
minio        █ 5.40 MB/s
rustfs       █ 4.48 MB/s
```

**Latency (P50)**
```
horse        █ 83ns
minio        █████████████████████████ 178.7us
rustfs       ██████████████████████████████ 212.3us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 9454688.07 MB/s | 84ns | 125ns | 292ns | 0 |
| minio | 1591.92 MB/s | 624.6us | 737.5us | 1.1ms | 0 |
| rustfs | 1147.17 MB/s | 855.1us | 1.0ms | 1.1ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 9454688.07 MB/s
minio        █ 1591.92 MB/s
rustfs       █ 1147.17 MB/s
```

**Latency (P50)**
```
horse        █ 84ns
minio        █████████████████████ 624.6us
rustfs       ██████████████████████████████ 855.1us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 621999.64 MB/s | 83ns | 125ns | 292ns | 0 |
| minio | 324.52 MB/s | 187.1us | 211.2us | 275.0us | 0 |
| rustfs | 252.19 MB/s | 243.5us | 276.8us | 310.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 621999.64 MB/s
minio        █ 324.52 MB/s
rustfs       █ 252.19 MB/s
```

**Latency (P50)**
```
horse        █ 83ns
minio        ███████████████████████ 187.1us
rustfs       ██████████████████████████████ 243.5us
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 666667 ops/s | 1.5us | 1.5us | 1.5us | 0 |
| rustfs | 1606 ops/s | 622.8us | 622.8us | 622.8us | 0 |
| minio | 513 ops/s | 1.9ms | 1.9ms | 1.9ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 666667 ops/s
rustfs       █ 1606 ops/s
minio        █ 513 ops/s
```

**Latency (P50)**
```
horse        █ 1.5us
rustfs       █████████ 622.8us
minio        ██████████████████████████████ 1.9ms
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 247402 ops/s | 4.0us | 4.0us | 4.0us | 0 |
| minio | 225 ops/s | 4.4ms | 4.4ms | 4.4ms | 0 |
| rustfs | 168 ops/s | 5.9ms | 5.9ms | 5.9ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 247402 ops/s
minio        █ 225 ops/s
rustfs       █ 168 ops/s
```

**Latency (P50)**
```
horse        █ 4.0us
minio        ██████████████████████ 4.4ms
rustfs       ██████████████████████████████ 5.9ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 40336 ops/s | 24.8us | 24.8us | 24.8us | 0 |
| minio | 23 ops/s | 42.6ms | 42.6ms | 42.6ms | 0 |
| rustfs | 17 ops/s | 60.4ms | 60.4ms | 60.4ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 40336 ops/s
minio        █ 23 ops/s
rustfs       █ 17 ops/s
```

**Latency (P50)**
```
horse        █ 24.8us
minio        █████████████████████ 42.6ms
rustfs       ██████████████████████████████ 60.4ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 3709 ops/s | 269.6us | 269.6us | 269.6us | 0 |
| minio | 2 ops/s | 453.2ms | 453.2ms | 453.2ms | 0 |
| rustfs | 2 ops/s | 620.0ms | 620.0ms | 620.0ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 3709 ops/s
minio        █ 2 ops/s
rustfs       █ 2 ops/s
```

**Latency (P50)**
```
horse        █ 269.6us
minio        █████████████████████ 453.2ms
rustfs       ██████████████████████████████ 620.0ms
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 269687 ops/s | 3.7us | 3.7us | 3.7us | 0 |
| minio | 2087 ops/s | 479.2us | 479.2us | 479.2us | 0 |
| rustfs | 1175 ops/s | 851.3us | 851.3us | 851.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 269687 ops/s
minio        █ 2087 ops/s
rustfs       █ 1175 ops/s
```

**Latency (P50)**
```
horse        █ 3.7us
minio        ████████████████ 479.2us
rustfs       ██████████████████████████████ 851.3us
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 263713 ops/s | 3.8us | 3.8us | 3.8us | 0 |
| minio | 1275 ops/s | 784.4us | 784.4us | 784.4us | 0 |
| rustfs | 732 ops/s | 1.4ms | 1.4ms | 1.4ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 263713 ops/s
minio        █ 1275 ops/s
rustfs       █ 732 ops/s
```

**Latency (P50)**
```
horse        █ 3.8us
minio        █████████████████ 784.4us
rustfs       ██████████████████████████████ 1.4ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 63492 ops/s | 15.8us | 15.8us | 15.8us | 0 |
| minio | 298 ops/s | 3.4ms | 3.4ms | 3.4ms | 0 |
| rustfs | 207 ops/s | 4.8ms | 4.8ms | 4.8ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 63492 ops/s
minio        █ 298 ops/s
rustfs       █ 207 ops/s
```

**Latency (P50)**
```
horse        █ 15.8us
minio        ████████████████████ 3.4ms
rustfs       ██████████████████████████████ 4.8ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5803 ops/s | 172.3us | 172.3us | 172.3us | 0 |
| minio | 39 ops/s | 25.9ms | 25.9ms | 25.9ms | 0 |
| rustfs | 27 ops/s | 36.5ms | 36.5ms | 36.5ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5803 ops/s
minio        █ 39 ops/s
rustfs       █ 27 ops/s
```

**Latency (P50)**
```
horse        █ 172.3us
minio        █████████████████████ 25.9ms
rustfs       ██████████████████████████████ 36.5ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 28.04 MB/s | 8.7us | 8.7us | 8.7us | 0 |
| minio | 0.48 MB/s | 510.0us | 510.0us | 510.0us | 0 |
| rustfs | 0.43 MB/s | 561.9us | 561.9us | 561.9us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 28.04 MB/s
minio        █ 0.48 MB/s
rustfs       █ 0.43 MB/s
```

**Latency (P50)**
```
horse        █ 8.7us
minio        ███████████████████████████ 510.0us
rustfs       ██████████████████████████████ 561.9us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 230.67 MB/s | 10.6us | 10.6us | 10.6us | 0 |
| rustfs | 0.50 MB/s | 4.9ms | 4.9ms | 4.9ms | 0 |
| minio | 0.46 MB/s | 5.4ms | 5.4ms | 5.4ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 230.67 MB/s
rustfs       █ 0.50 MB/s
minio        █ 0.46 MB/s
```

**Latency (P50)**
```
horse        █ 10.6us
rustfs       ███████████████████████████ 4.9ms
minio        ██████████████████████████████ 5.4ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 324.44 MB/s | 75.2us | 75.2us | 75.2us | 0 |
| minio | 0.52 MB/s | 46.6ms | 46.6ms | 46.6ms | 0 |
| rustfs | 0.50 MB/s | 48.5ms | 48.5ms | 48.5ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 324.44 MB/s
minio        █ 0.52 MB/s
rustfs       █ 0.50 MB/s
```

**Latency (P50)**
```
horse        █ 75.2us
minio        ████████████████████████████ 46.6ms
rustfs       ██████████████████████████████ 48.5ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 413.42 MB/s | 590.5us | 590.5us | 590.5us | 0 |
| minio | 0.53 MB/s | 463.3ms | 463.3ms | 463.3ms | 0 |
| rustfs | 0.52 MB/s | 470.8ms | 470.8ms | 470.8ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 413.42 MB/s
minio        █ 0.53 MB/s
rustfs       █ 0.52 MB/s
```

**Latency (P50)**
```
horse        █ 590.5us
minio        █████████████████████████████ 463.3ms
rustfs       ██████████████████████████████ 470.8ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 16965122 ops/s | 42ns | 84ns | 208ns | 0 |
| minio | 6321 ops/s | 154.0us | 182.1us | 257.9us | 0 |
| rustfs | 5969 ops/s | 162.9us | 199.6us | 237.1us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 16965122 ops/s
minio        █ 6321 ops/s
rustfs       █ 5969 ops/s
```

**Latency (P50)**
```
horse        █ 42ns
minio        ████████████████████████████ 154.0us
rustfs       ██████████████████████████████ 162.9us
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1492.63 MB/s | 240.3us | 41.2ms | 45.0ms | 0 |
| minio | 408.83 MB/s | 24.4ms | 25.0ms | 25.1ms | 0 |
| rustfs | 349.03 MB/s | 28.1ms | 29.1ms | 29.1ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1492.63 MB/s
minio        ████████ 408.83 MB/s
rustfs       ███████ 349.03 MB/s
```

**Latency (P50)**
```
horse        █ 240.3us
minio        ██████████████████████████ 24.4ms
rustfs       ██████████████████████████████ 28.1ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1417.92 MB/s | 459ns | 959ns | 1.9us | 0 |
| rustfs | 2.10 MB/s | 448.4us | 551.3us | 630.5us | 0 |
| minio | 2.09 MB/s | 450.9us | 555.3us | 612.8us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1417.92 MB/s
rustfs       █ 2.10 MB/s
minio        █ 2.09 MB/s
```

**Latency (P50)**
```
horse        █ 459ns
rustfs       █████████████████████████████ 448.4us
minio        ██████████████████████████████ 450.9us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1341.49 MB/s | 20.1us | 26.5us | 34.1ms | 0 |
| minio | 292.08 MB/s | 3.4ms | 3.8ms | 4.1ms | 0 |
| rustfs | 267.25 MB/s | 3.7ms | 4.1ms | 4.3ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1341.49 MB/s
minio        ██████ 292.08 MB/s
rustfs       █████ 267.25 MB/s
```

**Latency (P50)**
```
horse        █ 20.1us
minio        ███████████████████████████ 3.4ms
rustfs       ██████████████████████████████ 3.7ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1400.83 MB/s | 1.9us | 3.3us | 4.6us | 0 |
| minio | 93.23 MB/s | 642.1us | 753.9us | 1.1ms | 0 |
| rustfs | 90.61 MB/s | 642.2us | 791.5us | 1.0ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1400.83 MB/s
minio        █ 93.23 MB/s
rustfs       █ 90.61 MB/s
```

**Latency (P50)**
```
horse        █ 1.9us
minio        █████████████████████████████ 642.1us
rustfs       ██████████████████████████████ 642.2us
```

## Recommendations

- **Write-heavy workloads:** horse
- **Read-heavy workloads:** horse

---

*Generated by storage benchmark CLI*
