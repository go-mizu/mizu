# Storage Benchmark Report

**Generated:** 2026-02-19T14:01:57+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** horse (won 29/40 benchmarks, 72%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | horse | 29 | 72% |
| 2 | zebra | 11 | 28% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | horse | 8.9 GB/s | +50% vs zebra |
| Small Write (1KB) | zebra | 1.9 GB/s | +34% vs horse |
| Large Read (10MB) | zebra | 37.1 GB/s | 3.5x vs horse |
| Large Write (10MB) | horse | 1.1 GB/s | 19.6x vs zebra |
| Delete | zebra | 2.9M ops/s | +14% vs horse |
| Stat | horse | 14.9M ops/s | +30% vs zebra |
| List (100 objects) | horse | 125.7K ops/s | +69% vs zebra |
| Copy | zebra | 1.1 GB/s | +73% vs horse |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **horse** | 1102 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **zebra** | 37093 MB/s | Best for streaming, CDN |
| Small File Operations | **horse** | 5251824 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **horse** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| horse | 1102.1 | 10651.3 | 235.2us | 1.1ms |
| zebra | 56.4 | 37093.5 | 289.7us | 708ns |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| horse | 1429975 | 9073673 | 500ns | 83ns |
| zebra | 1923209 | 6055098 | 375ns | 125ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| horse | 14926617 | 125720 | 2543315 |
| zebra | 11462999 | 74301 | 2910967 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 971.17 | 137.84 | 37.82 |
| zebra | 932.84 | 408.70 | 197.40 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 6772.71 | 3526.40 | 2989.16 |
| zebra | 6271.87 | 3067.44 | 1337.00 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 55.7us | 12.7us | 298.5us | 1.0ms |
| zebra | 8.2us | 39.4us | 802.0us | 3.2ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 3.4us | 4.6us | 25.5us | 196.4us |
| zebra | 1.51s | 1.48s | 1.37s | 1.41s |

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
| zebra | 1087.79 MB/s | 541ns | 958ns | 5.9us | 0 |
| horse | 627.84 MB/s | 959ns | 1.7us | 6.6us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1087.79 MB/s
horse        █████████████████ 627.84 MB/s
```

**Latency (P50)**
```
zebra        ████████████████ 541ns
horse        ██████████████████████████████ 959ns
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 2910967 ops/s | 333ns | 500ns | 667ns | 0 |
| horse | 2543315 ops/s | 375ns | 542ns | 667ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 2910967 ops/s
horse        ██████████████████████████ 2543315 ops/s
```

**Latency (P50)**
```
zebra        ██████████████████████████ 333ns
horse        ██████████████████████████████ 375ns
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 118.00 MB/s | 542ns | 1.0us | 2.2us | 0 |
| zebra | 114.44 MB/s | 500ns | 1.4us | 3.1us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 118.00 MB/s
zebra        █████████████████████████████ 114.44 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 542ns
zebra        ███████████████████████████ 500ns
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1514675 ops/s | 500ns | 959ns | 2.2us | 0 |
| zebra | 1152914 ops/s | 417ns | 2.0us | 6.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1514675 ops/s
zebra        ██████████████████████ 1152914 ops/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 500ns
zebra        █████████████████████████ 417ns
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 91.45 MB/s | 708ns | 1.7us | 4.2us | 0 |
| horse | 4.11 MB/s | 750ns | 1.5us | 2.6us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 91.45 MB/s
horse        █ 4.11 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████████████ 708ns
horse        ██████████████████████████████ 750ns
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 125720 ops/s | 7.0us | 12.8us | 28.2us | 0 |
| zebra | 74301 ops/s | 12.9us | 16.4us | 28.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 125720 ops/s
zebra        █████████████████ 74301 ops/s
```

**Latency (P50)**
```
horse        ████████████████ 7.0us
zebra        ██████████████████████████████ 12.9us
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 26.19 MB/s | 10.3us | 4.1ms | 10.0ms | 0 |
| horse | 7.28 MB/s | 8.1us | 2.8ms | 93.4ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 26.19 MB/s
horse        ████████ 7.28 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 10.3us
horse        ███████████████████████ 8.1us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 126.26 MB/s | 8.7us | 363.3us | 2.8ms | 0 |
| zebra | 62.75 MB/s | 5.8us | 567.0us | 5.7ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 126.26 MB/s
zebra        ██████████████ 62.75 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 8.7us
zebra        ████████████████████ 5.8us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 7.01 MB/s | 6.1us | 4.4ms | 38.8ms | 0 |
| horse | 2.64 MB/s | 6.5us | 5.3ms | 147.0ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 7.01 MB/s
horse        ███████████ 2.64 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████████████ 6.1us
horse        ██████████████████████████████ 6.5us
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 210.16 MB/s | 66.0ms | 115.3ms | 115.3ms | 0 |
| zebra | 41.91 MB/s | 510.1ms | 510.1ms | 510.1ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 210.16 MB/s
zebra        █████ 41.91 MB/s
```

**Latency (P50)**
```
horse        ███ 66.0ms
zebra        ██████████████████████████████ 510.1ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 6772.71 MB/s | 125ns | 209ns | 417ns | 0 |
| zebra | 6271.87 MB/s | 125ns | 208ns | 1.1us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 6772.71 MB/s
zebra        ███████████████████████████ 6271.87 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 125ns
zebra        ██████████████████████████████ 125ns
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 3526.40 MB/s | 208ns | 500ns | 1.1us | 0 |
| zebra | 3067.44 MB/s | 208ns | 541ns | 2.6us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 3526.40 MB/s
zebra        ██████████████████████████ 3067.44 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 208ns
zebra        ██████████████████████████████ 208ns
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2989.16 MB/s | 209ns | 584ns | 1.4us | 0 |
| zebra | 1337.00 MB/s | 250ns | 834ns | 4.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2989.16 MB/s
zebra        █████████████ 1337.00 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████ 209ns
zebra        ██████████████████████████████ 250ns
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 971.17 MB/s | 916ns | 1.4us | 2.2us | 0 |
| zebra | 932.84 MB/s | 958ns | 1.5us | 2.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 971.17 MB/s
zebra        ████████████████████████████ 932.84 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████████ 916ns
zebra        ██████████████████████████████ 958ns
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 408.70 MB/s | 1.4us | 5.0us | 17.8us | 0 |
| horse | 137.84 MB/s | 2.0us | 25.3us | 69.6us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 408.70 MB/s
horse        ██████████ 137.84 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████ 1.4us
horse        ██████████████████████████████ 2.0us
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 197.40 MB/s | 2.1us | 8.5us | 56.2us | 0 |
| horse | 37.82 MB/s | 3.2us | 104.1us | 196.8us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 197.40 MB/s
horse        █████ 37.82 MB/s
```

**Latency (P50)**
```
zebra        ███████████████████ 2.1us
horse        ██████████████████████████████ 3.2us
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2480004.61 MB/s | 83ns | 125ns | 292ns | 0 |
| zebra | 2393636.40 MB/s | 84ns | 125ns | 291ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2480004.61 MB/s
zebra        ████████████████████████████ 2393636.40 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2241076.15 MB/s | 84ns | 167ns | 292ns | 0 |
| zebra | 2071576.80 MB/s | 84ns | 125ns | 292ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2241076.15 MB/s
zebra        ███████████████████████████ 2071576.80 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 84ns
zebra        ██████████████████████████████ 84ns
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2103128.20 MB/s | 125ns | 167ns | 333ns | 0 |
| zebra | 1785232.97 MB/s | 125ns | 208ns | 375ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2103128.20 MB/s
zebra        █████████████████████████ 1785232.97 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 125ns
zebra        ██████████████████████████████ 125ns
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 37093.45 MB/s | 708ns | 1.4ms | 1.8ms | 0 |
| horse | 10651.27 MB/s | 1.1ms | 2.5ms | 4.7ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 37093.45 MB/s
horse        ████████ 10651.27 MB/s
```

**Latency (P50)**
```
zebra        █ 708ns
horse        ██████████████████████████████ 1.1ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 8861.01 MB/s | 83ns | 166ns | 334ns | 0 |
| zebra | 5913.18 MB/s | 125ns | 250ns | 1.1us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 8861.01 MB/s
zebra        ████████████████████ 5913.18 MB/s
```

**Latency (P50)**
```
horse        ███████████████████ 83ns
zebra        ██████████████████████████████ 125ns
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 5818662.54 MB/s | 125ns | 208ns | 1.2us | 0 |
| horse | 35163.91 MB/s | 375ns | 97.6us | 173.0us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 5818662.54 MB/s
horse        █ 35163.91 MB/s
```

**Latency (P50)**
```
zebra        ██████████ 125ns
horse        ██████████████████████████████ 375ns
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 627756.49 MB/s | 83ns | 125ns | 250ns | 0 |
| zebra | 465279.83 MB/s | 125ns | 167ns | 708ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 627756.49 MB/s
zebra        ██████████████████████ 465279.83 MB/s
```

**Latency (P50)**
```
horse        ███████████████████ 83ns
zebra        ██████████████████████████████ 125ns
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 705716 ops/s | 1.4us | 1.4us | 1.4us | 0 |
| zebra | 152858 ops/s | 6.5us | 6.5us | 6.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 705716 ops/s
zebra        ██████ 152858 ops/s
```

**Latency (P50)**
```
horse        ██████ 1.4us
zebra        ██████████████████████████████ 6.5us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 90220 ops/s | 11.1us | 11.1us | 11.1us | 0 |
| zebra | 24768 ops/s | 40.4us | 40.4us | 40.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 90220 ops/s
zebra        ████████ 24768 ops/s
```

**Latency (P50)**
```
horse        ████████ 11.1us
zebra        ██████████████████████████████ 40.4us
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 35088 ops/s | 28.5us | 28.5us | 28.5us | 0 |
| zebra | 8270 ops/s | 120.9us | 120.9us | 120.9us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 35088 ops/s
zebra        ███████ 8270 ops/s
```

**Latency (P50)**
```
horse        ███████ 28.5us
zebra        ██████████████████████████████ 120.9us
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 3684 ops/s | 271.5us | 271.5us | 271.5us | 0 |
| zebra | 1300 ops/s | 769.1us | 769.1us | 769.1us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 3684 ops/s
zebra        ██████████ 1300 ops/s
```

**Latency (P50)**
```
horse        ██████████ 271.5us
zebra        ██████████████████████████████ 769.1us
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 292740 ops/s | 3.4us | 3.4us | 3.4us | 0 |
| zebra | 1 ops/s | 1.51s | 1.51s | 1.51s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 292740 ops/s
zebra        █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 3.4us
zebra        ██████████████████████████████ 1.51s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 216216 ops/s | 4.6us | 4.6us | 4.6us | 0 |
| zebra | 1 ops/s | 1.48s | 1.48s | 1.48s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 216216 ops/s
zebra        █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 4.6us
zebra        ██████████████████████████████ 1.48s
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 39151 ops/s | 25.5us | 25.5us | 25.5us | 0 |
| zebra | 1 ops/s | 1.37s | 1.37s | 1.37s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 39151 ops/s
zebra        █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 25.5us
zebra        ██████████████████████████████ 1.37s
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5091 ops/s | 196.4us | 196.4us | 196.4us | 0 |
| zebra | 1 ops/s | 1.41s | 1.41s | 1.41s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5091 ops/s
zebra        █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 196.4us
zebra        ██████████████████████████████ 1.41s
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 29.59 MB/s | 8.2us | 8.2us | 8.2us | 0 |
| horse | 4.38 MB/s | 55.7us | 55.7us | 55.7us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 29.59 MB/s
horse        ████ 4.38 MB/s
```

**Latency (P50)**
```
zebra        ████ 8.2us
horse        ██████████████████████████████ 55.7us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 192.74 MB/s | 12.7us | 12.7us | 12.7us | 0 |
| zebra | 61.94 MB/s | 39.4us | 39.4us | 39.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 192.74 MB/s
zebra        █████████ 61.94 MB/s
```

**Latency (P50)**
```
horse        █████████ 12.7us
zebra        ██████████████████████████████ 39.4us
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 81.79 MB/s | 298.5us | 298.5us | 298.5us | 0 |
| zebra | 30.44 MB/s | 802.0us | 802.0us | 802.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 81.79 MB/s
zebra        ███████████ 30.44 MB/s
```

**Latency (P50)**
```
horse        ███████████ 298.5us
zebra        ██████████████████████████████ 802.0us
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 236.64 MB/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| zebra | 76.99 MB/s | 3.2ms | 3.2ms | 3.2ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 236.64 MB/s
zebra        █████████ 76.99 MB/s
```

**Latency (P50)**
```
horse        █████████ 1.0ms
zebra        ██████████████████████████████ 3.2ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 14926617 ops/s | 42ns | 84ns | 250ns | 0 |
| zebra | 11462999 ops/s | 42ns | 84ns | 292ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 14926617 ops/s
zebra        ███████████████████████ 11462999 ops/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 42ns
zebra        ██████████████████████████████ 42ns
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1102.07 MB/s | 235.2us | 46.5ms | 80.4ms | 0 |
| zebra | 56.37 MB/s | 289.7us | 289.7us | 289.7us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1102.07 MB/s
zebra        █ 56.37 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████ 235.2us
zebra        ██████████████████████████████ 289.7us
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1878.13 MB/s | 375ns | 709ns | 2.5us | 0 |
| horse | 1396.46 MB/s | 500ns | 1.0us | 1.9us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1878.13 MB/s
horse        ██████████████████████ 1396.46 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████ 375ns
horse        ██████████████████████████████ 500ns
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1146.12 MB/s | 19.4us | 32.5us | 47.9ms | 0 |
| zebra | 346.56 MB/s | 23.1us | 64.4us | 186.8us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1146.12 MB/s
zebra        █████████ 346.56 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████ 19.4us
zebra        ██████████████████████████████ 23.1us
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1495.22 MB/s | 1.8us | 2.8us | 4.2us | 0 |
| zebra | 1151.60 MB/s | 1.9us | 5.6us | 11.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1495.22 MB/s
zebra        ███████████████████████ 1151.60 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████████ 1.8us
zebra        ██████████████████████████████ 1.9us
```

## Recommendations

- **Write-heavy workloads:** zebra
- **Read-heavy workloads:** zebra

---

*Generated by storage benchmark CLI*
