# Storage Benchmark Report

**Generated:** 2026-02-19T14:31:17+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** horse (won 27/40 benchmarks, 68%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | horse | 27 | 68% |
| 2 | zebra | 13 | 32% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | horse | 8.8 GB/s | close |
| Small Write (1KB) | zebra | 1.8 GB/s | +33% vs horse |
| Large Read (10MB) | horse | 82095.6 GB/s | close |
| Large Write (10MB) | horse | 708.2 MB/s | +30% vs zebra |
| Delete | zebra | 3.0M ops/s | +20% vs horse |
| Stat | horse | 17.3M ops/s | +21% vs zebra |
| List (100 objects) | horse | 135.9K ops/s | +76% vs zebra |
| Copy | zebra | 919.6 MB/s | +60% vs horse |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **horse** | 708 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **horse** | 82095562 MB/s | Best for streaming, CDN |
| Small File Operations | **zebra** | 5316963 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **zebra** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| horse | 708.2 | 82095561.9 | 240.6us | 84ns |
| zebra | 542.9 | 75615015.9 | 255.3us | 84ns |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| horse | 1390468 | 8994938 | 500ns | 83ns |
| zebra | 1844886 | 8789040 | 375ns | 83ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| horse | 17262894 | 135882 | 2515508 |
| zebra | 14322616 | 76992 | 3006894 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 1009.20 | 161.18 | 43.73 |
| zebra | 1066.13 | 507.42 | 161.90 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 6937.33 | 5286.24 | 4702.55 |
| zebra | 6166.42 | 5090.73 | 2175.29 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 9.0us | 11.7us | 84.5us | 847.3us |
| zebra | 5.7us | 26.8us | 296.5us | 5.9ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 2.8us | 4.3us | 22.0us | 197.5us |
| zebra | 1.56s | 30.9us | 41.8us | 347.5us |

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
| zebra | 919.60 MB/s | 542ns | 1.3us | 6.2us | 0 |
| horse | 573.63 MB/s | 1.0us | 1.6us | 6.2us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 919.60 MB/s
horse        ██████████████████ 573.63 MB/s
```

**Latency (P50)**
```
zebra        ████████████████ 542ns
horse        ██████████████████████████████ 1.0us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 3006894 ops/s | 333ns | 500ns | 666ns | 0 |
| horse | 2515508 ops/s | 375ns | 542ns | 708ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 3006894 ops/s
horse        █████████████████████████ 2515508 ops/s
```

**Latency (P50)**
```
zebra        ██████████████████████████ 333ns
horse        ██████████████████████████████ 375ns
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 119.49 MB/s | 500ns | 917ns | 1.8us | 0 |
| horse | 117.97 MB/s | 542ns | 1.2us | 2.1us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 119.49 MB/s
horse        █████████████████████████████ 117.97 MB/s
```

**Latency (P50)**
```
zebra        ███████████████████████████ 500ns
horse        ██████████████████████████████ 542ns
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1467736 ops/s | 459ns | 1.1us | 2.5us | 0 |
| zebra | 910179 ops/s | 500ns | 1.9us | 7.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1467736 ops/s
zebra        ██████████████████ 910179 ops/s
```

**Latency (P50)**
```
horse        ███████████████████████████ 459ns
zebra        ██████████████████████████████ 500ns
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 121.83 MB/s | 666ns | 1.2us | 3.0us | 0 |
| horse | 100.59 MB/s | 709ns | 1.2us | 2.1us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 121.83 MB/s
horse        ████████████████████████ 100.59 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████████████ 666ns
horse        ██████████████████████████████ 709ns
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 135882 ops/s | 6.4us | 12.1us | 26.3us | 0 |
| zebra | 76992 ops/s | 12.2us | 16.0us | 20.7us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 135882 ops/s
zebra        ████████████████ 76992 ops/s
```

**Latency (P50)**
```
horse        ███████████████ 6.4us
zebra        ██████████████████████████████ 12.2us
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5.39 MB/s | 7.0us | 3.3ms | 121.7ms | 0 |
| zebra | 4.91 MB/s | 1.7us | 13.0us | 311.6us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5.39 MB/s
zebra        ███████████████████████████ 4.91 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 7.0us
zebra        ███████ 1.7us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 129.61 MB/s | 250ns | 5.8us | 25.9us | 0 |
| horse | 100.88 MB/s | 7.0us | 298.8us | 2.7ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 129.61 MB/s
horse        ███████████████████████ 100.88 MB/s
```

**Latency (P50)**
```
zebra        █ 250ns
horse        ██████████████████████████████ 7.0us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1.86 MB/s | 5.5us | 5.9ms | 198.3ms | 0 |
| zebra | 0.48 MB/s | 2.4us | 17.3us | 1.65s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1.86 MB/s
zebra        ███████ 0.48 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 5.5us
zebra        █████████████ 2.4us
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 176.71 MB/s | 53.8ms | 121.9ms | 121.9ms | 0 |
| horse | 161.20 MB/s | 85.1ms | 97.6ms | 97.6ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 176.71 MB/s
horse        ███████████████████████████ 161.20 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████ 53.8ms
horse        ██████████████████████████████ 85.1ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 6937.33 MB/s | 125ns | 209ns | 417ns | 0 |
| zebra | 6166.42 MB/s | 125ns | 209ns | 1.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 6937.33 MB/s
zebra        ██████████████████████████ 6166.42 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 125ns
zebra        ██████████████████████████████ 125ns
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5286.24 MB/s | 125ns | 375ns | 959ns | 0 |
| zebra | 5090.73 MB/s | 125ns | 291ns | 1.7us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5286.24 MB/s
zebra        ████████████████████████████ 5090.73 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 125ns
zebra        ██████████████████████████████ 125ns
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 4702.55 MB/s | 125ns | 416ns | 1.0us | 0 |
| zebra | 2175.29 MB/s | 167ns | 584ns | 2.6us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 4702.55 MB/s
zebra        █████████████ 2175.29 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████ 125ns
zebra        ██████████████████████████████ 167ns
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1066.13 MB/s | 833ns | 1.3us | 1.7us | 0 |
| horse | 1009.20 MB/s | 875ns | 1.3us | 1.8us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1066.13 MB/s
horse        ████████████████████████████ 1009.20 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████████████ 833ns
horse        ██████████████████████████████ 875ns
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 507.42 MB/s | 1.0us | 3.7us | 13.3us | 0 |
| horse | 161.18 MB/s | 1.5us | 20.7us | 58.8us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 507.42 MB/s
horse        █████████ 161.18 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████ 1.0us
horse        ██████████████████████████████ 1.5us
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 161.90 MB/s | 1.8us | 14.5us | 109.6us | 0 |
| horse | 43.73 MB/s | 2.4us | 92.3us | 211.4us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 161.90 MB/s
horse        ████████ 43.73 MB/s
```

**Latency (P50)**
```
zebra        ███████████████████████ 1.8us
horse        ██████████████████████████████ 2.4us
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2541881.33 MB/s | 83ns | 125ns | 291ns | 0 |
| zebra | 1977959.99 MB/s | 84ns | 250ns | 375ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2541881.33 MB/s
zebra        ███████████████████████ 1977959.99 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2421336.02 MB/s | 83ns | 167ns | 292ns | 0 |
| zebra | 2321531.46 MB/s | 84ns | 125ns | 292ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2421336.02 MB/s
zebra        ████████████████████████████ 2321531.46 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2480279.91 MB/s | 83ns | 125ns | 250ns | 0 |
| zebra | 1636456.64 MB/s | 84ns | 125ns | 1.1us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2480279.91 MB/s
zebra        ███████████████████ 1636456.64 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 82095561.92 MB/s | 84ns | 167ns | 292ns | 0 |
| zebra | 75615015.86 MB/s | 84ns | 167ns | 1.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 82095561.92 MB/s
zebra        ███████████████████████████ 75615015.86 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 84ns
zebra        ██████████████████████████████ 84ns
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 8784.12 MB/s | 83ns | 166ns | 333ns | 0 |
| zebra | 8583.05 MB/s | 83ns | 125ns | 583ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 8784.12 MB/s
zebra        █████████████████████████████ 8583.05 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 83ns
zebra        ██████████████████████████████ 83ns
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 9734336.74 MB/s | 84ns | 125ns | 292ns | 0 |
| zebra | 8258894.81 MB/s | 84ns | 125ns | 1.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 9734336.74 MB/s
zebra        █████████████████████████ 8258894.81 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 84ns
zebra        ██████████████████████████████ 84ns
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 629394.17 MB/s | 83ns | 125ns | 250ns | 0 |
| zebra | 538121.43 MB/s | 84ns | 125ns | 1.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 629394.17 MB/s
zebra        █████████████████████████ 538121.43 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 585480 ops/s | 1.7us | 1.7us | 1.7us | 0 |
| zebra | 170213 ops/s | 5.9us | 5.9us | 5.9us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 585480 ops/s
zebra        ████████ 170213 ops/s
```

**Latency (P50)**
```
horse        ████████ 1.7us
zebra        ██████████████████████████████ 5.9us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 328731 ops/s | 3.0us | 3.0us | 3.0us | 0 |
| horse | 230787 ops/s | 4.3us | 4.3us | 4.3us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 328731 ops/s
horse        █████████████████████ 230787 ops/s
```

**Latency (P50)**
```
zebra        █████████████████████ 3.0us
horse        ██████████████████████████████ 4.3us
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 39473 ops/s | 25.3us | 25.3us | 25.3us | 0 |
| zebra | 33850 ops/s | 29.5us | 29.5us | 29.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 39473 ops/s
zebra        █████████████████████████ 33850 ops/s
```

**Latency (P50)**
```
horse        █████████████████████████ 25.3us
zebra        ██████████████████████████████ 29.5us
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 3641 ops/s | 274.6us | 274.6us | 274.6us | 0 |
| zebra | 2100 ops/s | 476.2us | 476.2us | 476.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 3641 ops/s
zebra        █████████████████ 2100 ops/s
```

**Latency (P50)**
```
horse        █████████████████ 274.6us
zebra        ██████████████████████████████ 476.2us
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 358295 ops/s | 2.8us | 2.8us | 2.8us | 0 |
| zebra | 1 ops/s | 1.56s | 1.56s | 1.56s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 358295 ops/s
zebra        █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 2.8us
zebra        ██████████████████████████████ 1.56s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 230734 ops/s | 4.3us | 4.3us | 4.3us | 0 |
| zebra | 32346 ops/s | 30.9us | 30.9us | 30.9us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 230734 ops/s
zebra        ████ 32346 ops/s
```

**Latency (P50)**
```
horse        ████ 4.3us
zebra        ██████████████████████████████ 30.9us
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 45541 ops/s | 22.0us | 22.0us | 22.0us | 0 |
| zebra | 23905 ops/s | 41.8us | 41.8us | 41.8us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 45541 ops/s
zebra        ███████████████ 23905 ops/s
```

**Latency (P50)**
```
horse        ███████████████ 22.0us
zebra        ██████████████████████████████ 41.8us
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5062 ops/s | 197.5us | 197.5us | 197.5us | 0 |
| zebra | 2877 ops/s | 347.5us | 347.5us | 347.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5062 ops/s
zebra        █████████████████ 2877 ops/s
```

**Latency (P50)**
```
horse        █████████████████ 197.5us
zebra        ██████████████████████████████ 347.5us
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 42.77 MB/s | 5.7us | 5.7us | 5.7us | 0 |
| horse | 27.25 MB/s | 9.0us | 9.0us | 9.0us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 42.77 MB/s
horse        ███████████████████ 27.25 MB/s
```

**Latency (P50)**
```
zebra        ███████████████████ 5.7us
horse        ██████████████████████████████ 9.0us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 208.52 MB/s | 11.7us | 11.7us | 11.7us | 0 |
| zebra | 90.99 MB/s | 26.8us | 26.8us | 26.8us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 208.52 MB/s
zebra        █████████████ 90.99 MB/s
```

**Latency (P50)**
```
horse        █████████████ 11.7us
zebra        ██████████████████████████████ 26.8us
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 288.92 MB/s | 84.5us | 84.5us | 84.5us | 0 |
| zebra | 82.35 MB/s | 296.5us | 296.5us | 296.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 288.92 MB/s
zebra        ████████ 82.35 MB/s
```

**Latency (P50)**
```
horse        ████████ 84.5us
zebra        ██████████████████████████████ 296.5us
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 288.13 MB/s | 847.3us | 847.3us | 847.3us | 0 |
| zebra | 41.57 MB/s | 5.9ms | 5.9ms | 5.9ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 288.13 MB/s
zebra        ████ 41.57 MB/s
```

**Latency (P50)**
```
horse        ████ 847.3us
zebra        ██████████████████████████████ 5.9ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 17262894 ops/s | 42ns | 84ns | 208ns | 0 |
| zebra | 14322616 ops/s | 42ns | 84ns | 333ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 17262894 ops/s
zebra        ████████████████████████ 14322616 ops/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 42ns
zebra        ██████████████████████████████ 42ns
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 708.25 MB/s | 240.6us | 54.1ms | 269.2ms | 0 |
| zebra | 542.94 MB/s | 255.3us | 37.4ms | 414.5ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 708.25 MB/s
zebra        ██████████████████████ 542.94 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████████ 240.6us
zebra        ██████████████████████████████ 255.3us
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1801.65 MB/s | 375ns | 750ns | 2.0us | 0 |
| horse | 1357.88 MB/s | 500ns | 1.1us | 2.0us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1801.65 MB/s
horse        ██████████████████████ 1357.88 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████ 375ns
horse        ██████████████████████████████ 500ns
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 685.51 MB/s | 20.5us | 25.3us | 39.0ms | 0 |
| zebra | 101.80 MB/s | 18.5us | 28.8us | 408.3ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 685.51 MB/s
zebra        ████ 101.80 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 20.5us
zebra        ███████████████████████████ 18.5us
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 2533.04 MB/s | 1.9us | 4.9us | 10.2us | 0 |
| horse | 1268.96 MB/s | 1.9us | 3.4us | 6.7us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 2533.04 MB/s
horse        ███████████████ 1268.96 MB/s
```

**Latency (P50)**
```
zebra        █████████████████████████████ 1.9us
horse        ██████████████████████████████ 1.9us
```

## Recommendations

- **Write-heavy workloads:** zebra
- **Read-heavy workloads:** horse

---

*Generated by storage benchmark CLI*
