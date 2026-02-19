# Storage Benchmark Report

**Generated:** 2026-02-19T14:25:45+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** horse (won 29/40 benchmarks, 72%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | horse | 29 | 72% |
| 2 | pony | 11 | 28% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | horse | 3.0 GB/s | close |
| Small Write (1KB) | pony | 802.8 MB/s | +12% vs horse |
| Large Read (10MB) | horse | 24067.3 GB/s | 12998.4x vs pony |
| Large Write (10MB) | pony | 1.1 GB/s | 5.0x vs horse |
| Delete | pony | 2.5M ops/s | +35% vs horse |
| Stat | horse | 5.2M ops/s | +21% vs pony |
| List (100 objects) | horse | 33.5K ops/s | +53% vs pony |
| Copy | pony | 394.5 MB/s | +38% vs horse |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **pony** | 1143 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **horse** | 24067257 MB/s | Best for streaming, CDN |
| Small File Operations | **horse** | 1900250 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **horse** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| horse | 230.5 | 24067257.2 | 412.9us | 291ns |
| pony | 1142.6 | 1851.6 | 8.1ms | 5.4ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| horse | 733981 | 3066519 | 583ns | 125ns |
| pony | 822057 | 2926516 | 458ns | 166ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| horse | 5248375 | 33487 | 1870677 |
| pony | 4334308 | 21923 | 2521893 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 365.53 | 70.15 | 12.57 |
| pony | 88.05 | 76.73 | 27.98 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 1529.33 | 358.91 | 274.31 |
| pony | 483.71 | 237.21 | 100.27 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 43.0us | 21.4us | 333.8us | 1.6ms |
| pony | 7.8us | 38.2us | 511.0us | 7.9ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 10.0us | 8.2us | 67.9us | 769.0us |
| pony | 892.7ms | 796.7ms | 755.6ms | 757.8ms |

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
- **pony** (40 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 394.46 MB/s | 1.4us | 3.0us | 5.7us | 0 |
| horse | 286.07 MB/s | 1.5us | 3.3us | 7.5us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 394.46 MB/s
horse        █████████████████████ 286.07 MB/s
```

**Latency (P50)**
```
pony         ████████████████████████████ 1.4us
horse        ██████████████████████████████ 1.5us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 2521893 ops/s | 375ns | 584ns | 833ns | 0 |
| horse | 1870677 ops/s | 375ns | 916ns | 1.3us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 2521893 ops/s
horse        ██████████████████████ 1870677 ops/s
```

**Latency (P50)**
```
pony         ██████████████████████████████ 375ns
horse        ██████████████████████████████ 375ns
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 44.25 MB/s | 1.0us | 2.0us | 4.0us | 0 |
| pony | 32.72 MB/s | 833ns | 1.9us | 8.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 44.25 MB/s
pony         ██████████████████████ 32.72 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 1.0us
pony         ███████████████████████ 833ns
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 962194 ops/s | 708ns | 1.2us | 2.9us | 0 |
| horse | 488242 ops/s | 958ns | 2.0us | 4.0us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 962194 ops/s
horse        ███████████████ 488242 ops/s
```

**Latency (P50)**
```
pony         ██████████████████████ 708ns
horse        ██████████████████████████████ 958ns
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 35.59 MB/s | 1.3us | 2.6us | 4.8us | 0 |
| pony | 12.62 MB/s | 1.2us | 2.5us | 5.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 35.59 MB/s
pony         ██████████ 12.62 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 1.3us
pony         ███████████████████████████ 1.2us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 33487 ops/s | 15.3us | 32.8us | 67.9us | 0 |
| pony | 21923 ops/s | 24.8us | 50.4us | 134.6us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 33487 ops/s
pony         ███████████████████ 21923 ops/s
```

**Latency (P50)**
```
horse        ██████████████████ 15.3us
pony         ██████████████████████████████ 24.8us
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 24.27 MB/s | 8.6us | 2.4ms | 11.6ms | 0 |
| pony | 12.58 MB/s | 16.2us | 7.2ms | 11.1ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 24.27 MB/s
pony         ███████████████ 12.58 MB/s
```

**Latency (P50)**
```
horse        ███████████████ 8.6us
pony         ██████████████████████████████ 16.2us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 42.73 MB/s | 11.7us | 309.5us | 11.4ms | 0 |
| pony | 7.46 MB/s | 58.8us | 16.9ms | 25.2ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 42.73 MB/s
pony         █████ 7.46 MB/s
```

**Latency (P50)**
```
horse        █████ 11.7us
pony         ██████████████████████████████ 58.8us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 141.36 MB/s | 4.2us | 377.9us | 2.0ms | 0 |
| pony | 2.49 MB/s | 2.7ms | 22.3ms | 65.9ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 141.36 MB/s
pony         █ 2.49 MB/s
```

**Latency (P50)**
```
horse        █ 4.2us
pony         ██████████████████████████████ 2.7ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 200.16 MB/s | 75.5ms | 86.4ms | 86.4ms | 0 |
| horse | 122.61 MB/s | 106.2ms | 119.8ms | 119.8ms | 0 |

**Throughput**
```
pony         ██████████████████████████████ 200.16 MB/s
horse        ██████████████████ 122.61 MB/s
```

**Latency (P50)**
```
pony         █████████████████████ 75.5ms
horse        ██████████████████████████████ 106.2ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1529.33 MB/s | 250ns | 792ns | 2.5us | 0 |
| pony | 483.71 MB/s | 916ns | 1.9us | 3.7us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1529.33 MB/s
pony         █████████ 483.71 MB/s
```

**Latency (P50)**
```
horse        ████████ 250ns
pony         ██████████████████████████████ 916ns
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 358.91 MB/s | 375ns | 1.0us | 2.7us | 0 |
| pony | 237.21 MB/s | 1.6us | 3.3us | 7.9us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 358.91 MB/s
pony         ███████████████████ 237.21 MB/s
```

**Latency (P50)**
```
horse        ██████ 375ns
pony         ██████████████████████████████ 1.6us
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 274.31 MB/s | 417ns | 1.2us | 2.9us | 0 |
| pony | 100.27 MB/s | 1.7us | 3.8us | 24.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 274.31 MB/s
pony         ██████████ 100.27 MB/s
```

**Latency (P50)**
```
horse        ███████ 417ns
pony         ██████████████████████████████ 1.7us
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 365.53 MB/s | 1.6us | 3.6us | 6.2us | 0 |
| pony | 88.05 MB/s | 1.1us | 2.4us | 5.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 365.53 MB/s
pony         ███████ 88.05 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 1.6us
pony         █████████████████████ 1.1us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 76.73 MB/s | 2.1us | 16.7us | 186.5us | 0 |
| horse | 70.15 MB/s | 2.3us | 6.6us | 70.9us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 76.73 MB/s
horse        ███████████████████████████ 70.15 MB/s
```

**Latency (P50)**
```
pony         ███████████████████████████ 2.1us
horse        ██████████████████████████████ 2.3us
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 27.98 MB/s | 2.7us | 144.0us | 684.3us | 0 |
| horse | 12.57 MB/s | 2.4us | 65.5us | 1.4ms | 0 |

**Throughput**
```
pony         ██████████████████████████████ 27.98 MB/s
horse        █████████████ 12.57 MB/s
```

**Latency (P50)**
```
pony         ██████████████████████████████ 2.7us
horse        ███████████████████████████ 2.4us
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 627840.09 MB/s | 167ns | 542ns | 2.7us | 0 |
| pony | 533802.11 MB/s | 208ns | 708ns | 2.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 627840.09 MB/s
pony         █████████████████████████ 533802.11 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████ 167ns
pony         ██████████████████████████████ 208ns
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 552849.64 MB/s | 167ns | 542ns | 2.7us | 0 |
| pony | 505766.20 MB/s | 208ns | 750ns | 2.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 552849.64 MB/s
pony         ███████████████████████████ 505766.20 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████ 167ns
pony         ██████████████████████████████ 208ns
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 585693.07 MB/s | 167ns | 542ns | 2.7us | 0 |
| pony | 512271.39 MB/s | 208ns | 708ns | 2.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 585693.07 MB/s
pony         ██████████████████████████ 512271.39 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████ 167ns
pony         ██████████████████████████████ 208ns
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 24067257.16 MB/s | 291ns | 1.0us | 2.8us | 0 |
| pony | 1851.56 MB/s | 5.4ms | 6.9ms | 7.7ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 24067257.16 MB/s
pony         █ 1851.56 MB/s
```

**Latency (P50)**
```
horse        █ 291ns
pony         ██████████████████████████████ 5.4ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2994.65 MB/s | 125ns | 542ns | 1.6us | 0 |
| pony | 2857.93 MB/s | 166ns | 584ns | 1.7us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2994.65 MB/s
pony         ████████████████████████████ 2857.93 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████ 125ns
pony         ██████████████████████████████ 166ns
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1271409.97 MB/s | 125ns | 583ns | 1.3us | 0 |
| pony | 1945.63 MB/s | 275.5us | 1.7ms | 2.6ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1271409.97 MB/s
pony         █ 1945.63 MB/s
```

**Latency (P50)**
```
horse        █ 125ns
pony         ██████████████████████████████ 275.5us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 189280.36 MB/s | 125ns | 541ns | 1.3us | 0 |
| pony | 8860.84 MB/s | 167ns | 16.9us | 115.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 189280.36 MB/s
pony         █ 8860.84 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████ 125ns
pony         ██████████████████████████████ 167ns
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 338066 ops/s | 3.0us | 3.0us | 3.0us | 0 |
| pony | 71215 ops/s | 14.0us | 14.0us | 14.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 338066 ops/s
pony         ██████ 71215 ops/s
```

**Latency (P50)**
```
horse        ██████ 3.0us
pony         ██████████████████████████████ 14.0us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 65755 ops/s | 15.2us | 15.2us | 15.2us | 0 |
| pony | 55426 ops/s | 18.0us | 18.0us | 18.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 65755 ops/s
pony         █████████████████████████ 55426 ops/s
```

**Latency (P50)**
```
horse        █████████████████████████ 15.2us
pony         ██████████████████████████████ 18.0us
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 10274 ops/s | 97.3us | 97.3us | 97.3us | 0 |
| pony | 6768 ops/s | 147.8us | 147.8us | 147.8us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 10274 ops/s
pony         ███████████████████ 6768 ops/s
```

**Latency (P50)**
```
horse        ███████████████████ 97.3us
pony         ██████████████████████████████ 147.8us
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 1359 ops/s | 736.0us | 736.0us | 736.0us | 0 |
| horse | 812 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |

**Throughput**
```
pony         ██████████████████████████████ 1359 ops/s
horse        █████████████████ 812 ops/s
```

**Latency (P50)**
```
pony         █████████████████ 736.0us
horse        ██████████████████████████████ 1.2ms
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 99592 ops/s | 10.0us | 10.0us | 10.0us | 0 |
| pony | 1 ops/s | 892.7ms | 892.7ms | 892.7ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 99592 ops/s
pony         █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 10.0us
pony         ██████████████████████████████ 892.7ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 121212 ops/s | 8.2us | 8.2us | 8.2us | 0 |
| pony | 1 ops/s | 796.7ms | 796.7ms | 796.7ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 121212 ops/s
pony         █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 8.2us
pony         ██████████████████████████████ 796.7ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 14724 ops/s | 67.9us | 67.9us | 67.9us | 0 |
| pony | 1 ops/s | 755.6ms | 755.6ms | 755.6ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 14724 ops/s
pony         █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 67.9us
pony         ██████████████████████████████ 755.6ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1300 ops/s | 769.0us | 769.0us | 769.0us | 0 |
| pony | 1 ops/s | 757.8ms | 757.8ms | 757.8ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1300 ops/s
pony         █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 769.0us
pony         ██████████████████████████████ 757.8ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 31.33 MB/s | 7.8us | 7.8us | 7.8us | 0 |
| horse | 5.68 MB/s | 43.0us | 43.0us | 43.0us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 31.33 MB/s
horse        █████ 5.68 MB/s
```

**Latency (P50)**
```
pony         █████ 7.8us
horse        ██████████████████████████████ 43.0us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 114.22 MB/s | 21.4us | 21.4us | 21.4us | 0 |
| pony | 63.90 MB/s | 38.2us | 38.2us | 38.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 114.22 MB/s
pony         ████████████████ 63.90 MB/s
```

**Latency (P50)**
```
horse        ████████████████ 21.4us
pony         ██████████████████████████████ 38.2us
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 73.14 MB/s | 333.8us | 333.8us | 333.8us | 0 |
| pony | 47.78 MB/s | 511.0us | 511.0us | 511.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 73.14 MB/s
pony         ███████████████████ 47.78 MB/s
```

**Latency (P50)**
```
horse        ███████████████████ 333.8us
pony         ██████████████████████████████ 511.0us
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 152.57 MB/s | 1.6ms | 1.6ms | 1.6ms | 0 |
| pony | 30.85 MB/s | 7.9ms | 7.9ms | 7.9ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 152.57 MB/s
pony         ██████ 30.85 MB/s
```

**Latency (P50)**
```
horse        ██████ 1.6ms
pony         ██████████████████████████████ 7.9ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5248375 ops/s | 83ns | 167ns | 1.5us | 0 |
| pony | 4334308 ops/s | 84ns | 209ns | 1.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5248375 ops/s
pony         ████████████████████████ 4334308 ops/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
pony         ██████████████████████████████ 84ns
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 1142.55 MB/s | 8.1ms | 12.2ms | 13.3ms | 0 |
| horse | 230.47 MB/s | 412.9us | 235.6ms | 787.4ms | 0 |

**Throughput**
```
pony         ██████████████████████████████ 1142.55 MB/s
horse        ██████ 230.47 MB/s
```

**Latency (P50)**
```
pony         ██████████████████████████████ 8.1ms
horse        █ 412.9us
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 802.79 MB/s | 458ns | 958ns | 1.8us | 0 |
| horse | 716.78 MB/s | 583ns | 1.6us | 4.4us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 802.79 MB/s
horse        ██████████████████████████ 716.78 MB/s
```

**Latency (P50)**
```
pony         ███████████████████████ 458ns
horse        ██████████████████████████████ 583ns
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 605.06 MB/s | 56.2us | 4.2ms | 6.4ms | 0 |
| horse | 551.35 MB/s | 23.0us | 108.2us | 33.3ms | 0 |

**Throughput**
```
pony         ██████████████████████████████ 605.06 MB/s
horse        ███████████████████████████ 551.35 MB/s
```

**Latency (P50)**
```
pony         ██████████████████████████████ 56.2us
horse        ████████████ 23.0us
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2103.04 MB/s | 3.1us | 7.2us | 13.6us | 0 |
| pony | 1703.86 MB/s | 3.6us | 6.4us | 490.8us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2103.04 MB/s
pony         ████████████████████████ 1703.86 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████ 3.1us
pony         ██████████████████████████████ 3.6us
```

## Recommendations

- **Write-heavy workloads:** horse
- **Read-heavy workloads:** horse

---

*Generated by storage benchmark CLI*
