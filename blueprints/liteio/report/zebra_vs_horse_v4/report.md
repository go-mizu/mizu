# Storage Benchmark Report

**Generated:** 2026-02-19T14:08:30+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** horse (won 28/40 benchmarks, 70%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | horse | 28 | 70% |
| 2 | zebra | 12 | 30% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | horse | 8.6 GB/s | close |
| Small Write (1KB) | zebra | 1.8 GB/s | +33% vs horse |
| Large Read (10MB) | horse | 90199.7 GB/s | +71% vs zebra |
| Large Write (10MB) | horse | 544.6 MB/s | +73% vs zebra |
| Delete | zebra | 2.7M ops/s | close |
| Stat | horse | 17.1M ops/s | +63% vs zebra |
| List (100 objects) | horse | 135.6K ops/s | 2.1x vs zebra |
| Copy | zebra | 832.4 MB/s | +18% vs horse |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **horse** | 545 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **horse** | 90199741 MB/s | Best for streaming, CDN |
| Small File Operations | **horse** | 5090197 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **horse** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| horse | 544.6 | 90199741.4 | 230.8us | 84ns |
| zebra | 315.3 | 52783284.9 | 235.8us | 84ns |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| horse | 1411934 | 8768459 | 459ns | 83ns |
| zebra | 1874262 | 8197952 | 375ns | 84ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| horse | 17064739 | 135560 | 2480027 |
| zebra | 10453782 | 64433 | 2684473 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 997.79 | 105.19 | 23.86 |
| zebra | 917.33 | 494.03 | 190.07 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 6593.51 | 5184.63 | 3016.35 |
| zebra | 6671.73 | 3926.75 | 3922.63 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 5.9us | 8.1us | 74.0us | 635.0us |
| zebra | 9.4us | 24.1us | 158.7us | 920.7us |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 3.3us | 4.2us | 17.3us | 165.3us |
| zebra | 1.93s | 1.49s | 1.48s | 1.46s |

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
| zebra | 832.41 MB/s | 583ns | 1.4us | 8.2us | 0 |
| horse | 702.67 MB/s | 959ns | 1.5us | 6.9us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 832.41 MB/s
horse        █████████████████████████ 702.67 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████ 583ns
horse        ██████████████████████████████ 959ns
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 2684473 ops/s | 333ns | 584ns | 958ns | 0 |
| horse | 2480027 ops/s | 375ns | 542ns | 750ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 2684473 ops/s
horse        ███████████████████████████ 2480027 ops/s
```

**Latency (P50)**
```
zebra        ██████████████████████████ 333ns
horse        ██████████████████████████████ 375ns
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 105.53 MB/s | 542ns | 1.5us | 4.3us | 0 |
| horse | 91.67 MB/s | 541ns | 1.0us | 1.8us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 105.53 MB/s
horse        ██████████████████████████ 91.67 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 542ns
horse        █████████████████████████████ 541ns
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1826135 ops/s | 458ns | 708ns | 958ns | 0 |
| zebra | 1389110 ops/s | 417ns | 1.8us | 5.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1826135 ops/s
zebra        ██████████████████████ 1389110 ops/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 458ns
zebra        ███████████████████████████ 417ns
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 99.68 MB/s | 708ns | 1.8us | 4.0us | 0 |
| horse | 85.85 MB/s | 708ns | 1.1us | 1.8us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 99.68 MB/s
horse        █████████████████████████ 85.85 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 708ns
horse        ██████████████████████████████ 708ns
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 135560 ops/s | 6.5us | 11.8us | 25.9us | 0 |
| zebra | 64433 ops/s | 13.8us | 21.2us | 50.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 135560 ops/s
zebra        ██████████████ 64433 ops/s
```

**Latency (P50)**
```
horse        ██████████████ 6.5us
zebra        ██████████████████████████████ 13.8us
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 94.78 MB/s | 9.5us | 212.9us | 2.5ms | 0 |
| horse | 2.03 MB/s | 6.5us | 2.7ms | 95.9ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 94.78 MB/s
horse        █ 2.03 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 9.5us
horse        ████████████████████ 6.5us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 329.76 MB/s | 10.7us | 169.7us | 656.2us | 0 |
| horse | 91.84 MB/s | 7.2us | 253.5us | 2.6ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 329.76 MB/s
horse        ████████ 91.84 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 10.7us
horse        ████████████████████ 7.2us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 3.29 MB/s | 4.0us | 1.1ms | 117.1ms | 0 |
| horse | 0.83 MB/s | 3.2us | 11.3ms | 697.9ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 3.29 MB/s
horse        ███████ 0.83 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 4.0us
horse        ████████████████████████ 3.2us
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 176.91 MB/s | 81.9ms | 97.0ms | 97.0ms | 0 |
| zebra | 47.39 MB/s | 407.0ms | 407.0ms | 407.0ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 176.91 MB/s
zebra        ████████ 47.39 MB/s
```

**Latency (P50)**
```
horse        ██████ 81.9ms
zebra        ██████████████████████████████ 407.0ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 6671.73 MB/s | 125ns | 209ns | 459ns | 0 |
| horse | 6593.51 MB/s | 125ns | 250ns | 459ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 6671.73 MB/s
horse        █████████████████████████████ 6593.51 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 125ns
horse        ██████████████████████████████ 125ns
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5184.63 MB/s | 125ns | 375ns | 917ns | 0 |
| zebra | 3926.75 MB/s | 166ns | 459ns | 1.9us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5184.63 MB/s
zebra        ██████████████████████ 3926.75 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████ 125ns
zebra        ██████████████████████████████ 166ns
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 3922.63 MB/s | 167ns | 500ns | 1.1us | 0 |
| horse | 3016.35 MB/s | 125ns | 375ns | 1.0us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 3922.63 MB/s
horse        ███████████████████████ 3016.35 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 167ns
horse        ██████████████████████ 125ns
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 997.79 MB/s | 875ns | 1.3us | 1.8us | 0 |
| zebra | 917.33 MB/s | 958ns | 1.5us | 2.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 997.79 MB/s
zebra        ███████████████████████████ 917.33 MB/s
```

**Latency (P50)**
```
horse        ███████████████████████████ 875ns
zebra        ██████████████████████████████ 958ns
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 494.03 MB/s | 1.2us | 4.1us | 15.5us | 0 |
| horse | 105.19 MB/s | 1.5us | 18.0us | 52.3us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 494.03 MB/s
horse        ██████ 105.19 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████ 1.2us
horse        ██████████████████████████████ 1.5us
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 190.07 MB/s | 2.0us | 7.9us | 84.2us | 0 |
| horse | 23.86 MB/s | 2.0us | 90.1us | 218.1us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 190.07 MB/s
horse        ███ 23.86 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 2.0us
horse        █████████████████████████████ 2.0us
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2448724.97 MB/s | 83ns | 166ns | 292ns | 0 |
| zebra | 2274255.51 MB/s | 84ns | 125ns | 291ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2448724.97 MB/s
zebra        ███████████████████████████ 2274255.51 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2507706.05 MB/s | 83ns | 166ns | 292ns | 0 |
| zebra | 1968800.24 MB/s | 125ns | 167ns | 417ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2507706.05 MB/s
zebra        ███████████████████████ 1968800.24 MB/s
```

**Latency (P50)**
```
horse        ███████████████████ 83ns
zebra        ██████████████████████████████ 125ns
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2260484.33 MB/s | 83ns | 167ns | 292ns | 0 |
| zebra | 2072739.50 MB/s | 125ns | 167ns | 292ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2260484.33 MB/s
zebra        ███████████████████████████ 2072739.50 MB/s
```

**Latency (P50)**
```
horse        ███████████████████ 83ns
zebra        ██████████████████████████████ 125ns
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 90199741.39 MB/s | 84ns | 125ns | 291ns | 0 |
| zebra | 52783284.93 MB/s | 84ns | 167ns | 1.1us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 90199741.39 MB/s
zebra        █████████████████ 52783284.93 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 84ns
zebra        ██████████████████████████████ 84ns
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 8562.95 MB/s | 83ns | 208ns | 333ns | 0 |
| zebra | 8005.81 MB/s | 84ns | 125ns | 1.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 8562.95 MB/s
zebra        ████████████████████████████ 8005.81 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 9730090.97 MB/s | 84ns | 125ns | 291ns | 0 |
| zebra | 8180045.55 MB/s | 84ns | 125ns | 958ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 9730090.97 MB/s
zebra        █████████████████████████ 8180045.55 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 84ns
zebra        ██████████████████████████████ 84ns
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 651821.13 MB/s | 83ns | 125ns | 250ns | 0 |
| zebra | 522984.84 MB/s | 84ns | 125ns | 1.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 651821.13 MB/s
zebra        ████████████████████████ 522984.84 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 750188 ops/s | 1.3us | 1.3us | 1.3us | 0 |
| zebra | 45629 ops/s | 21.9us | 21.9us | 21.9us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 750188 ops/s
zebra        █ 45629 ops/s
```

**Latency (P50)**
```
horse        █ 1.3us
zebra        ██████████████████████████████ 21.9us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 230787 ops/s | 4.3us | 4.3us | 4.3us | 0 |
| zebra | 44859 ops/s | 22.3us | 22.3us | 22.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 230787 ops/s
zebra        █████ 44859 ops/s
```

**Latency (P50)**
```
horse        █████ 4.3us
zebra        ██████████████████████████████ 22.3us
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 40748 ops/s | 24.5us | 24.5us | 24.5us | 0 |
| zebra | 9788 ops/s | 102.2us | 102.2us | 102.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 40748 ops/s
zebra        ███████ 9788 ops/s
```

**Latency (P50)**
```
horse        ███████ 24.5us
zebra        ██████████████████████████████ 102.2us
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 3778 ops/s | 264.7us | 264.7us | 264.7us | 0 |
| zebra | 997 ops/s | 1.0ms | 1.0ms | 1.0ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 3778 ops/s
zebra        ███████ 997 ops/s
```

**Latency (P50)**
```
horse        ███████ 264.7us
zebra        ██████████████████████████████ 1.0ms
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 303859 ops/s | 3.3us | 3.3us | 3.3us | 0 |
| zebra | 1 ops/s | 1.93s | 1.93s | 1.93s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 303859 ops/s
zebra        █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 3.3us
zebra        ██████████████████████████████ 1.93s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 235294 ops/s | 4.2us | 4.2us | 4.2us | 0 |
| zebra | 1 ops/s | 1.49s | 1.49s | 1.49s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 235294 ops/s
zebra        █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 4.2us
zebra        ██████████████████████████████ 1.49s
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 57830 ops/s | 17.3us | 17.3us | 17.3us | 0 |
| zebra | 1 ops/s | 1.48s | 1.48s | 1.48s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 57830 ops/s
zebra        █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 17.3us
zebra        ██████████████████████████████ 1.48s
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 6050 ops/s | 165.3us | 165.3us | 165.3us | 0 |
| zebra | 1 ops/s | 1.46s | 1.46s | 1.46s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 6050 ops/s
zebra        █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 165.3us
zebra        ██████████████████████████████ 1.46s
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 41.26 MB/s | 5.9us | 5.9us | 5.9us | 0 |
| zebra | 25.93 MB/s | 9.4us | 9.4us | 9.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 41.26 MB/s
zebra        ██████████████████ 25.93 MB/s
```

**Latency (P50)**
```
horse        ██████████████████ 5.9us
zebra        ██████████████████████████████ 9.4us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 300.48 MB/s | 8.1us | 8.1us | 8.1us | 0 |
| zebra | 101.37 MB/s | 24.1us | 24.1us | 24.1us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 300.48 MB/s
zebra        ██████████ 101.37 MB/s
```

**Latency (P50)**
```
horse        ██████████ 8.1us
zebra        ██████████████████████████████ 24.1us
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 330.11 MB/s | 74.0us | 74.0us | 74.0us | 0 |
| zebra | 153.87 MB/s | 158.7us | 158.7us | 158.7us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 330.11 MB/s
zebra        █████████████ 153.87 MB/s
```

**Latency (P50)**
```
horse        █████████████ 74.0us
zebra        ██████████████████████████████ 158.7us
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 384.50 MB/s | 635.0us | 635.0us | 635.0us | 0 |
| zebra | 265.17 MB/s | 920.7us | 920.7us | 920.7us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 384.50 MB/s
zebra        ████████████████████ 265.17 MB/s
```

**Latency (P50)**
```
horse        ████████████████████ 635.0us
zebra        ██████████████████████████████ 920.7us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 17064739 ops/s | 42ns | 84ns | 208ns | 0 |
| zebra | 10453782 ops/s | 42ns | 125ns | 458ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 17064739 ops/s
zebra        ██████████████████ 10453782 ops/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 42ns
zebra        ██████████████████████████████ 42ns
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 544.61 MB/s | 230.8us | 95.5ms | 191.5ms | 0 |
| zebra | 315.29 MB/s | 235.8us | 75.7ms | 453.6ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 544.61 MB/s
zebra        █████████████████ 315.29 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 230.8us
zebra        ██████████████████████████████ 235.8us
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1830.33 MB/s | 375ns | 875ns | 2.0us | 0 |
| horse | 1378.84 MB/s | 459ns | 1.2us | 2.1us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1830.33 MB/s
horse        ██████████████████████ 1378.84 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████████ 375ns
horse        ██████████████████████████████ 459ns
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1057.97 MB/s | 19.7us | 25.9us | 37.0ms | 0 |
| zebra | 497.34 MB/s | 20.9us | 109.4us | 326.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1057.97 MB/s
zebra        ██████████████ 497.34 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████████ 19.7us
zebra        ██████████████████████████████ 20.9us
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1467.86 MB/s | 1.9us | 3.0us | 4.8us | 0 |
| zebra | 366.99 MB/s | 1.9us | 4.5us | 11.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1467.86 MB/s
zebra        ███████ 366.99 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 1.9us
zebra        █████████████████████████████ 1.9us
```

## Recommendations

- **Write-heavy workloads:** zebra
- **Read-heavy workloads:** horse

---

*Generated by storage benchmark CLI*
