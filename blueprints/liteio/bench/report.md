# Storage Benchmark Report

**Generated:** 2026-02-20T15:11:27+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** herd (won 14/40 benchmarks, 35%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | herd | 14 | 35% |
| 2 | zebra | 9 | 22% |
| 3 | pony | 9 | 22% |
| 4 | horse | 5 | 12% |
| 5 | usagi | 3 | 8% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | horse | 5.5 GB/s | close |
| Small Write (1KB) | zebra | 1.3 GB/s | close |
| Large Read (10MB) | zebra | 54.5 GB/s | close |
| Large Write (10MB) | horse | 2.8 GB/s | +84% vs rabbit |
| Delete | zebra | 3.0M ops/s | close |
| Stat | usagi | 20.4M ops/s | +22% vs horse |
| List (100 objects) | usagi | 195.2K ops/s | +46% vs horse |
| Copy | pony | 1.6 GB/s | +95% vs herd |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **horse** | 2790 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **zebra** | 54453 MB/s | Best for streaming, CDN |
| Small File Operations | **horse** | 3492487 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **zebra** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| bee3 | 324.8 | 8111.8 | 21.0ms | 1.1ms |
| herd | 682.9 | 9962.9 | 4.6ms | 987.3us |
| horse | 2790.4 | 11544.9 | 293.4us | 1.0ms |
| pony | 1141.4 | 51021.6 | 8.1ms | 184.1us |
| rabbit | 1514.6 | 10082.0 | 6.4ms | 980.5us |
| usagi | 1509.5 | 10670.3 | 4.4ms | 914.1us |
| zebra | 124.4 | 54453.2 | 202.1us | 182.5us |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| bee3 | 139154 | 2490560 | 4.4us | 334ns |
| herd | 1219103 | 5276867 | 500ns | 125ns |
| horse | 1325454 | 5659521 | 500ns | 125ns |
| pony | 1277603 | 5320792 | 417ns | 125ns |
| rabbit | 19626 | 5025771 | 45.1us | 167ns |
| usagi | 271560 | 3586600 | 2.2us | 208ns |
| zebra | 1373109 | 5075020 | 375ns | 125ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| bee3 | 3498763 | 586 | 193675 |
| herd | 10707705 | 77792 | 2536405 |
| horse | 16690392 | 133842 | 2565398 |
| pony | 13892447 | 87246 | 2937666 |
| rabbit | 4536211 | 5284 | 29116 |
| usagi | 20366216 | 195187 | 636837 |
| zebra | 14500512 | 67874 | 2974469 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| bee3 | 131.78 | 25.39 | 6.55 |
| herd | 762.69 | 278.12 | 178.04 |
| horse | 1000.39 | 144.09 | 36.10 |
| pony | 516.77 | 76.77 | 13.89 |
| rabbit | 19.86 | 2.65 | 0.78 |
| usagi | 250.52 | 45.18 | 12.25 |
| zebra | 1094.60 | 492.23 | 138.76 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| bee3 | 1372.67 | 876.83 | 937.42 |
| herd | 3772.69 | 2232.82 | 2734.20 |
| horse | 4485.54 | 2967.57 | 2237.00 |
| pony | 1808.16 | 1240.49 | 881.59 |
| rabbit | 1822.92 | 655.21 | 669.08 |
| usagi | 1966.13 | 145.76 | 28.40 |
| zebra | 4945.35 | 3031.16 | 2257.24 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| bee3 | 9.4us | 82.4us | 504.3us | 5.7ms |
| herd | 18.5us | 45.5us | 195.4us | 722.0us |
| horse | 19.3us | 25.6us | 188.5us | 1.2ms |
| pony | 4.7us | 14.0us | 84.9us | 1.3ms |
| rabbit | 134.2us | 485.5us | 4.6ms | 45.2ms |
| usagi | 11.7us | 60.7us | 303.4us | 2.8ms |
| zebra | 8.6us | 36.4us | 128.9us | 974.1us |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| bee3 | 16.0ms | 16.1ms | 15.8ms | 16.8ms |
| herd | 11.8us | 9.8us | 40.9us | 282.0us |
| horse | 4.8us | 20.0us | 60.6us | 408.4us |
| pony | 1.54s | 1.53s | 1.56s | 1.53s |
| rabbit | 34.8us | 37.0us | 226.2us | 2.2ms |
| usagi | 558.1ms | 625.1ms | 601.7ms | 703.7ms |
| zebra | 1.57s | 56.1us | 78.7us | 273.8us |

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

- **bee3** (40 benchmarks)
- **herd** (40 benchmarks)
- **horse** (40 benchmarks)
- **pony** (40 benchmarks)
- **rabbit** (40 benchmarks)
- **usagi** (40 benchmarks)
- **zebra** (40 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 1647.40 MB/s | 500ns | 875ns | 2.1us | 0 |
| herd | 845.71 MB/s | 708ns | 2.0us | 4.0us | 0 |
| horse | 824.09 MB/s | 959ns | 1.5us | 3.3us | 0 |
| zebra | 546.67 MB/s | 625ns | 2.8us | 27.3us | 0 |
| usagi | 301.86 MB/s | 2.5us | 4.6us | 8.9us | 0 |
| rabbit | 13.98 MB/s | 55.5us | 154.6us | 196.1us | 0 |
| bee3 | 13.40 MB/s | 9.5us | 74.7us | 258.9us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 1647.40 MB/s
herd         ███████████████ 845.71 MB/s
horse        ███████████████ 824.09 MB/s
zebra        █████████ 546.67 MB/s
usagi        █████ 301.86 MB/s
rabbit       █ 13.98 MB/s
bee3         █ 13.40 MB/s
```

**Latency (P50)**
```
pony         █ 500ns
herd         █ 708ns
horse        █ 959ns
zebra        █ 625ns
usagi        █ 2.5us
rabbit       ██████████████████████████████ 55.5us
bee3         █████ 9.5us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 2974469 ops/s | 333ns | 500ns | 667ns | 0 |
| pony | 2937666 ops/s | 292ns | 500ns | 750ns | 0 |
| horse | 2565398 ops/s | 375ns | 542ns | 709ns | 0 |
| herd | 2536405 ops/s | 375ns | 542ns | 708ns | 0 |
| usagi | 636837 ops/s | 1.5us | 1.8us | 2.4us | 0 |
| bee3 | 193675 ops/s | 3.5us | 13.0us | 27.8us | 0 |
| rabbit | 29116 ops/s | 33.8us | 43.0us | 48.4us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 2974469 ops/s
pony         █████████████████████████████ 2937666 ops/s
horse        █████████████████████████ 2565398 ops/s
herd         █████████████████████████ 2536405 ops/s
usagi        ██████ 636837 ops/s
bee3         █ 193675 ops/s
rabbit       █ 29116 ops/s
```

**Latency (P50)**
```
zebra        █ 333ns
pony         █ 292ns
horse        █ 375ns
herd         █ 375ns
usagi        █ 1.5us
bee3         ███ 3.5us
rabbit       ██████████████████████████████ 33.8us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 178.90 MB/s | 500ns | 791ns | 1.1us | 0 |
| horse | 148.01 MB/s | 500ns | 834ns | 2.0us | 0 |
| zebra | 131.71 MB/s | 459ns | 917ns | 2.2us | 0 |
| herd | 127.71 MB/s | 542ns | 1.5us | 2.3us | 0 |
| usagi | 39.45 MB/s | 2.0us | 2.8us | 5.0us | 0 |
| bee3 | 17.95 MB/s | 3.7us | 12.2us | 27.6us | 0 |
| rabbit | 1.75 MB/s | 46.9us | 72.2us | 132.9us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 178.90 MB/s
horse        ████████████████████████ 148.01 MB/s
zebra        ██████████████████████ 131.71 MB/s
herd         █████████████████████ 127.71 MB/s
usagi        ██████ 39.45 MB/s
bee3         ███ 17.95 MB/s
rabbit       █ 1.75 MB/s
```

**Latency (P50)**
```
pony         █ 500ns
horse        █ 500ns
zebra        █ 459ns
herd         █ 542ns
usagi        █ 2.0us
bee3         ██ 3.7us
rabbit       ██████████████████████████████ 46.9us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 1632824 ops/s | 416ns | 709ns | 1.6us | 0 |
| horse | 1134052 ops/s | 500ns | 958ns | 2.1us | 0 |
| herd | 1004770 ops/s | 500ns | 1.3us | 5.2us | 0 |
| zebra | 677116 ops/s | 542ns | 2.8us | 14.4us | 0 |
| usagi | 511241 ops/s | 1.6us | 2.2us | 4.8us | 0 |
| bee3 | 109445 ops/s | 3.5us | 11.9us | 26.6us | 0 |
| rabbit | 29099 ops/s | 32.7us | 41.2us | 45.5us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 1632824 ops/s
horse        ████████████████████ 1134052 ops/s
herd         ██████████████████ 1004770 ops/s
zebra        ████████████ 677116 ops/s
usagi        █████████ 511241 ops/s
bee3         ██ 109445 ops/s
rabbit       █ 29099 ops/s
```

**Latency (P50)**
```
pony         █ 416ns
horse        █ 500ns
herd         █ 500ns
zebra        █ 542ns
usagi        █ 1.6us
bee3         ███ 3.5us
rabbit       ██████████████████████████████ 32.7us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 125.41 MB/s | 666ns | 1.1us | 2.3us | 0 |
| horse | 108.51 MB/s | 667ns | 1.1us | 2.0us | 0 |
| zebra | 100.00 MB/s | 667ns | 1.6us | 5.5us | 0 |
| herd | 70.80 MB/s | 959ns | 2.2us | 4.8us | 0 |
| usagi | 34.64 MB/s | 2.3us | 3.2us | 6.0us | 0 |
| bee3 | 4.30 MB/s | 4.5us | 15.3us | 34.6us | 0 |
| rabbit | 1.67 MB/s | 51.8us | 62.0us | 128.4us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 125.41 MB/s
horse        █████████████████████████ 108.51 MB/s
zebra        ███████████████████████ 100.00 MB/s
herd         ████████████████ 70.80 MB/s
usagi        ████████ 34.64 MB/s
bee3         █ 4.30 MB/s
rabbit       █ 1.67 MB/s
```

**Latency (P50)**
```
pony         █ 666ns
horse        █ 667ns
zebra        █ 667ns
herd         █ 959ns
usagi        █ 2.3us
bee3         ██ 4.5us
rabbit       ██████████████████████████████ 51.8us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 195187 ops/s | 4.5us | 7.3us | 13.4us | 0 |
| horse | 133842 ops/s | 6.5us | 12.6us | 26.2us | 0 |
| pony | 87246 ops/s | 9.8us | 17.6us | 38.5us | 0 |
| herd | 77792 ops/s | 11.6us | 17.8us | 26.3us | 0 |
| zebra | 67874 ops/s | 13.3us | 19.5us | 33.2us | 0 |
| rabbit | 5284 ops/s | 182.6us | 201.8us | 389.0us | 0 |
| bee3 | 586 ops/s | 1.5ms | 2.5ms | 3.6ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 195187 ops/s
horse        ████████████████████ 133842 ops/s
pony         █████████████ 87246 ops/s
herd         ███████████ 77792 ops/s
zebra        ██████████ 67874 ops/s
rabbit       █ 5284 ops/s
bee3         █ 586 ops/s
```

**Latency (P50)**
```
usagi        █ 4.5us
horse        █ 6.5us
pony         █ 9.8us
herd         █ 11.6us
zebra        █ 13.3us
rabbit       ███ 182.6us
bee3         ██████████████████████████████ 1.5ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 135.53 MB/s | 2.8us | 14.4us | 878.2us | 0 |
| rabbit | 73.20 MB/s | 63.9us | 743.0us | 1.3ms | 0 |
| zebra | 22.19 MB/s | 2.2us | 70.8us | 565.6us | 0 |
| horse | 17.89 MB/s | 6.5us | 518.5us | 43.7ms | 0 |
| pony | 16.17 MB/s | 10.5us | 4.5ms | 6.4ms | 0 |
| usagi | 13.33 MB/s | 901.1us | 3.3ms | 14.6ms | 0 |
| bee3 | 4.54 MB/s | 18.4us | 19.3ms | 43.4ms | 0 |

**Throughput**
```
herd         ██████████████████████████████ 135.53 MB/s
rabbit       ████████████████ 73.20 MB/s
zebra        ████ 22.19 MB/s
horse        ███ 17.89 MB/s
pony         ███ 16.17 MB/s
usagi        ██ 13.33 MB/s
bee3         █ 4.54 MB/s
```

**Latency (P50)**
```
herd         █ 2.8us
rabbit       ██ 63.9us
zebra        █ 2.2us
horse        █ 6.5us
pony         █ 10.5us
usagi        ██████████████████████████████ 901.1us
bee3         █ 18.4us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 5240.55 MB/s | 709ns | 9.9us | 31.8us | 0 |
| herd | 3383.15 MB/s | 1.1us | 5.3us | 17.2us | 0 |
| horse | 194.06 MB/s | 7.2us | 99.5us | 436.2us | 0 |
| rabbit | 110.13 MB/s | 917ns | 367.2us | 844.9us | 0 |
| pony | 81.60 MB/s | 1.6us | 711.0us | 1.2ms | 0 |
| usagi | 19.34 MB/s | 470.3us | 2.1ms | 7.2ms | 0 |
| bee3 | 3.07 MB/s | 2.3us | 1.8ms | 195.9ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 5240.55 MB/s
herd         ███████████████████ 3383.15 MB/s
horse        █ 194.06 MB/s
rabbit       █ 110.13 MB/s
pony         █ 81.60 MB/s
usagi        █ 19.34 MB/s
bee3         █ 3.07 MB/s
```

**Latency (P50)**
```
zebra        █ 709ns
herd         █ 1.1us
horse        █ 7.2us
rabbit       █ 917ns
pony         █ 1.6us
usagi        ██████████████████████████████ 470.3us
bee3         █ 2.3us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 25.10 MB/s | 3.7us | 188.2us | 20.4ms | 0 |
| usagi | 8.11 MB/s | 1.1ms | 5.5ms | 23.0ms | 0 |
| zebra | 7.45 MB/s | 4.0us | 168.7us | 66.7ms | 0 |
| pony | 6.42 MB/s | 2.4ms | 8.3ms | 12.7ms | 0 |
| horse | 4.93 MB/s | 5.8us | 38.5ms | 71.6ms | 0 |
| bee3 | 3.54 MB/s | 475.5us | 20.5ms | 30.4ms | 0 |
| rabbit | 2.51 MB/s | 387.7us | 53.2ms | 86.0ms | 0 |

**Throughput**
```
herd         ██████████████████████████████ 25.10 MB/s
usagi        █████████ 8.11 MB/s
zebra        ████████ 7.45 MB/s
pony         ███████ 6.42 MB/s
horse        █████ 4.93 MB/s
bee3         ████ 3.54 MB/s
rabbit       ███ 2.51 MB/s
```

**Latency (P50)**
```
herd         █ 3.7us
usagi        █████████████ 1.1ms
zebra        █ 4.0us
pony         ██████████████████████████████ 2.4ms
horse        █ 5.8us
bee3         ██████ 475.5us
rabbit       ████ 387.7us
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 933.82 MB/s | 12.7ms | 24.3ms | 47.3ms | 0 |
| pony | 416.02 MB/s | 33.9ms | 43.5ms | 43.6ms | 0 |
| rabbit | 370.67 MB/s | 36.2ms | 91.6ms | 91.6ms | 0 |
| herd | 360.81 MB/s | 31.7ms | 93.7ms | 103.6ms | 0 |
| horse | 292.78 MB/s | 51.1ms | 54.8ms | 54.8ms | 0 |
| bee3 | 171.68 MB/s | 72.1ms | 113.4ms | 113.4ms | 0 |
| zebra | 83.63 MB/s | 137.8ms | 137.8ms | 137.8ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 933.82 MB/s
pony         █████████████ 416.02 MB/s
rabbit       ███████████ 370.67 MB/s
herd         ███████████ 360.81 MB/s
horse        █████████ 292.78 MB/s
bee3         █████ 171.68 MB/s
zebra        ██ 83.63 MB/s
```

**Latency (P50)**
```
usagi        ██ 12.7ms
pony         ███████ 33.9ms
rabbit       ███████ 36.2ms
herd         ██████ 31.7ms
horse        ███████████ 51.1ms
bee3         ███████████████ 72.1ms
zebra        ██████████████████████████████ 137.8ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 4945.35 MB/s | 167ns | 292ns | 500ns | 0 |
| horse | 4485.54 MB/s | 208ns | 292ns | 542ns | 0 |
| herd | 3772.69 MB/s | 208ns | 375ns | 1.5us | 0 |
| usagi | 1966.13 MB/s | 292ns | 1.3us | 1.7us | 0 |
| rabbit | 1822.92 MB/s | 500ns | 667ns | 875ns | 0 |
| pony | 1808.16 MB/s | 500ns | 708ns | 916ns | 0 |
| bee3 | 1372.67 MB/s | 666ns | 917ns | 1.6us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 4945.35 MB/s
horse        ███████████████████████████ 4485.54 MB/s
herd         ██████████████████████ 3772.69 MB/s
usagi        ███████████ 1966.13 MB/s
rabbit       ███████████ 1822.92 MB/s
pony         ██████████ 1808.16 MB/s
bee3         ████████ 1372.67 MB/s
```

**Latency (P50)**
```
zebra        ███████ 167ns
horse        █████████ 208ns
herd         █████████ 208ns
usagi        █████████████ 292ns
rabbit       ██████████████████████ 500ns
pony         ██████████████████████ 500ns
bee3         ██████████████████████████████ 666ns
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 3031.16 MB/s | 167ns | 666ns | 2.0us | 0 |
| horse | 2967.57 MB/s | 208ns | 625ns | 1.3us | 0 |
| herd | 2232.82 MB/s | 250ns | 833ns | 3.3us | 0 |
| pony | 1240.49 MB/s | 625ns | 1.4us | 3.3us | 0 |
| bee3 | 876.83 MB/s | 792ns | 2.1us | 9.5us | 0 |
| rabbit | 655.21 MB/s | 1.2us | 1.6us | 2.9us | 0 |
| usagi | 145.76 MB/s | 542ns | 36.2us | 70.2us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 3031.16 MB/s
horse        █████████████████████████████ 2967.57 MB/s
herd         ██████████████████████ 2232.82 MB/s
pony         ████████████ 1240.49 MB/s
bee3         ████████ 876.83 MB/s
rabbit       ██████ 655.21 MB/s
usagi        █ 145.76 MB/s
```

**Latency (P50)**
```
zebra        ████ 167ns
horse        █████ 208ns
herd         ██████ 250ns
pony         ████████████████ 625ns
bee3         ████████████████████ 792ns
rabbit       ██████████████████████████████ 1.2us
usagi        █████████████ 542ns
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 2734.20 MB/s | 209ns | 708ns | 1.9us | 0 |
| zebra | 2257.24 MB/s | 209ns | 709ns | 3.1us | 0 |
| horse | 2237.00 MB/s | 250ns | 708ns | 1.7us | 0 |
| bee3 | 937.42 MB/s | 625ns | 2.1us | 9.7us | 0 |
| pony | 881.59 MB/s | 666ns | 2.0us | 11.0us | 0 |
| rabbit | 669.08 MB/s | 1.2us | 1.8us | 7.6us | 0 |
| usagi | 28.40 MB/s | 542ns | 194.5us | 365.8us | 0 |

**Throughput**
```
herd         ██████████████████████████████ 2734.20 MB/s
zebra        ████████████████████████ 2257.24 MB/s
horse        ████████████████████████ 2237.00 MB/s
bee3         ██████████ 937.42 MB/s
pony         █████████ 881.59 MB/s
rabbit       ███████ 669.08 MB/s
usagi        █ 28.40 MB/s
```

**Latency (P50)**
```
herd         █████ 209ns
zebra        █████ 209ns
horse        ██████ 250ns
bee3         ████████████████ 625ns
pony         █████████████████ 666ns
rabbit       ██████████████████████████████ 1.2us
usagi        █████████████ 542ns
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1094.60 MB/s | 792ns | 1.3us | 1.7us | 0 |
| horse | 1000.39 MB/s | 875ns | 1.3us | 1.8us | 0 |
| herd | 762.69 MB/s | 959ns | 2.3us | 5.0us | 0 |
| pony | 516.77 MB/s | 833ns | 1.1us | 1.8us | 0 |
| usagi | 250.52 MB/s | 3.2us | 4.9us | 10.3us | 0 |
| bee3 | 131.78 MB/s | 5.0us | 16.8us | 38.5us | 0 |
| rabbit | 19.86 MB/s | 43.8us | 52.0us | 58.2us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1094.60 MB/s
horse        ███████████████████████████ 1000.39 MB/s
herd         ████████████████████ 762.69 MB/s
pony         ██████████████ 516.77 MB/s
usagi        ██████ 250.52 MB/s
bee3         ███ 131.78 MB/s
rabbit       █ 19.86 MB/s
```

**Latency (P50)**
```
zebra        █ 792ns
horse        █ 875ns
herd         █ 959ns
pony         █ 833ns
usagi        ██ 3.2us
bee3         ███ 5.0us
rabbit       ██████████████████████████████ 43.8us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 492.23 MB/s | 1.0us | 4.0us | 18.3us | 0 |
| herd | 278.12 MB/s | 1.6us | 3.8us | 39.2us | 0 |
| horse | 144.09 MB/s | 1.6us | 20.1us | 55.1us | 0 |
| pony | 76.77 MB/s | 1.2us | 13.4us | 33.3us | 0 |
| usagi | 45.18 MB/s | 9.8us | 53.3us | 104.4us | 0 |
| bee3 | 25.39 MB/s | 6.6us | 155.4us | 319.8us | 0 |
| rabbit | 2.65 MB/s | 276.9us | 766.1us | 1.7ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 492.23 MB/s
herd         ████████████████ 278.12 MB/s
horse        ████████ 144.09 MB/s
pony         ████ 76.77 MB/s
usagi        ██ 45.18 MB/s
bee3         █ 25.39 MB/s
rabbit       █ 2.65 MB/s
```

**Latency (P50)**
```
zebra        █ 1.0us
herd         █ 1.6us
horse        █ 1.6us
pony         █ 1.2us
usagi        █ 9.8us
bee3         █ 6.6us
rabbit       ██████████████████████████████ 276.9us
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 178.04 MB/s | 1.5us | 4.9us | 77.4us | 0 |
| zebra | 138.76 MB/s | 1.8us | 17.1us | 131.4us | 0 |
| horse | 36.10 MB/s | 2.8us | 120.0us | 271.8us | 0 |
| pony | 13.89 MB/s | 2.8us | 112.3us | 3.4ms | 0 |
| usagi | 12.25 MB/s | 24.5us | 253.1us | 548.6us | 0 |
| bee3 | 6.55 MB/s | 15.1us | 657.5us | 1.2ms | 0 |
| rabbit | 0.78 MB/s | 223.4us | 5.1ms | 23.9ms | 0 |

**Throughput**
```
herd         ██████████████████████████████ 178.04 MB/s
zebra        ███████████████████████ 138.76 MB/s
horse        ██████ 36.10 MB/s
pony         ██ 13.89 MB/s
usagi        ██ 12.25 MB/s
bee3         █ 6.55 MB/s
rabbit       █ 0.78 MB/s
```

**Latency (P50)**
```
herd         █ 1.5us
zebra        █ 1.8us
horse        █ 2.8us
pony         █ 2.8us
usagi        ███ 24.5us
bee3         ██ 15.1us
rabbit       ██████████████████████████████ 223.4us
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 58427.28 MB/s | 4.2us | 5.2us | 8.2us | 0 |
| pony | 54420.63 MB/s | 4.5us | 5.6us | 6.0us | 0 |
| zebra | 53315.67 MB/s | 4.5us | 5.5us | 8.3us | 0 |
| horse | 46641.03 MB/s | 4.4us | 8.4us | 14.0us | 0 |
| usagi | 15777.08 MB/s | 15.3us | 18.2us | 22.0us | 0 |
| rabbit | 7153.52 MB/s | 25.9us | 70.3us | 94.4us | 0 |
| bee3 | 2773.79 MB/s | 57.7us | 141.5us | 289.3us | 0 |

**Throughput**
```
herd         ██████████████████████████████ 58427.28 MB/s
pony         ███████████████████████████ 54420.63 MB/s
zebra        ███████████████████████████ 53315.67 MB/s
horse        ███████████████████████ 46641.03 MB/s
usagi        ████████ 15777.08 MB/s
rabbit       ███ 7153.52 MB/s
bee3         █ 2773.79 MB/s
```

**Latency (P50)**
```
herd         ██ 4.2us
pony         ██ 4.5us
zebra        ██ 4.5us
horse        ██ 4.4us
usagi        ███████ 15.3us
rabbit       █████████████ 25.9us
bee3         ██████████████████████████████ 57.7us
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 58533.88 MB/s | 4.1us | 5.0us | 5.7us | 0 |
| herd | 57322.92 MB/s | 4.2us | 5.2us | 5.8us | 0 |
| zebra | 55673.12 MB/s | 4.2us | 4.9us | 8.0us | 0 |
| horse | 38479.51 MB/s | 4.5us | 8.7us | 16.2us | 0 |
| usagi | 15903.84 MB/s | 15.2us | 18.1us | 19.4us | 0 |
| rabbit | 6312.09 MB/s | 30.6us | 73.3us | 100.3us | 0 |
| bee3 | 2296.83 MB/s | 92.0us | 236.1us | 443.4us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 58533.88 MB/s
herd         █████████████████████████████ 57322.92 MB/s
zebra        ████████████████████████████ 55673.12 MB/s
horse        ███████████████████ 38479.51 MB/s
usagi        ████████ 15903.84 MB/s
rabbit       ███ 6312.09 MB/s
bee3         █ 2296.83 MB/s
```

**Latency (P50)**
```
pony         █ 4.1us
herd         █ 4.2us
zebra        █ 4.2us
horse        █ 4.5us
usagi        ████ 15.2us
rabbit       █████████ 30.6us
bee3         ██████████████████████████████ 92.0us
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 55626.89 MB/s | 4.3us | 5.3us | 6.4us | 0 |
| pony | 52100.56 MB/s | 4.7us | 5.5us | 5.9us | 0 |
| zebra | 50861.54 MB/s | 4.5us | 5.5us | 8.3us | 0 |
| horse | 41518.36 MB/s | 4.5us | 8.7us | 18.5us | 0 |
| usagi | 15481.13 MB/s | 15.3us | 19.2us | 21.6us | 0 |
| rabbit | 5497.74 MB/s | 34.0us | 85.4us | 130.2us | 0 |
| bee3 | 1899.48 MB/s | 88.6us | 278.6us | 1.2ms | 0 |

**Throughput**
```
herd         ██████████████████████████████ 55626.89 MB/s
pony         ████████████████████████████ 52100.56 MB/s
zebra        ███████████████████████████ 50861.54 MB/s
horse        ██████████████████████ 41518.36 MB/s
usagi        ████████ 15481.13 MB/s
rabbit       ██ 5497.74 MB/s
bee3         █ 1899.48 MB/s
```

**Latency (P50)**
```
herd         █ 4.3us
pony         █ 4.7us
zebra        █ 4.5us
horse        █ 4.5us
usagi        █████ 15.3us
rabbit       ███████████ 34.0us
bee3         ██████████████████████████████ 88.6us
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 54453.19 MB/s | 182.5us | 199.0us | 218.1us | 0 |
| pony | 51021.60 MB/s | 184.1us | 211.9us | 774.2us | 0 |
| horse | 11544.93 MB/s | 1.0ms | 1.8ms | 3.5ms | 0 |
| usagi | 10670.35 MB/s | 914.1us | 1.1ms | 1.4ms | 0 |
| rabbit | 10082.02 MB/s | 980.5us | 1.0ms | 1.1ms | 0 |
| herd | 9962.93 MB/s | 987.3us | 1.0ms | 1.6ms | 0 |
| bee3 | 8111.76 MB/s | 1.1ms | 1.6ms | 2.7ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 54453.19 MB/s
pony         ████████████████████████████ 51021.60 MB/s
horse        ██████ 11544.93 MB/s
usagi        █████ 10670.35 MB/s
rabbit       █████ 10082.02 MB/s
herd         █████ 9962.93 MB/s
bee3         ████ 8111.76 MB/s
```

**Latency (P50)**
```
zebra        ████ 182.5us
pony         ████ 184.1us
horse        ███████████████████████████ 1.0ms
usagi        ███████████████████████ 914.1us
rabbit       █████████████████████████ 980.5us
herd         █████████████████████████ 987.3us
bee3         ██████████████████████████████ 1.1ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5526.88 MB/s | 125ns | 208ns | 542ns | 0 |
| pony | 5196.09 MB/s | 125ns | 208ns | 459ns | 0 |
| herd | 5153.19 MB/s | 125ns | 250ns | 667ns | 0 |
| zebra | 4956.07 MB/s | 125ns | 250ns | 1.2us | 0 |
| rabbit | 4907.98 MB/s | 167ns | 250ns | 1.1us | 0 |
| usagi | 3502.54 MB/s | 208ns | 459ns | 1.2us | 0 |
| bee3 | 2432.19 MB/s | 334ns | 583ns | 1.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5526.88 MB/s
pony         ████████████████████████████ 5196.09 MB/s
herd         ███████████████████████████ 5153.19 MB/s
zebra        ██████████████████████████ 4956.07 MB/s
rabbit       ██████████████████████████ 4907.98 MB/s
usagi        ███████████████████ 3502.54 MB/s
bee3         █████████████ 2432.19 MB/s
```

**Latency (P50)**
```
horse        ███████████ 125ns
pony         ███████████ 125ns
herd         ███████████ 125ns
zebra        ███████████ 125ns
rabbit       ███████████████ 167ns
usagi        ██████████████████ 208ns
bee3         ██████████████████████████████ 334ns
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 56489.20 MB/s | 17.3us | 20.1us | 22.2us | 0 |
| horse | 55471.51 MB/s | 17.6us | 20.5us | 22.2us | 0 |
| herd | 55181.86 MB/s | 17.6us | 21.3us | 23.9us | 0 |
| zebra | 55142.11 MB/s | 17.5us | 21.0us | 25.5us | 0 |
| usagi | 15620.81 MB/s | 62.4us | 71.0us | 86.4us | 0 |
| rabbit | 11882.81 MB/s | 77.6us | 98.5us | 175.9us | 0 |
| bee3 | 10106.21 MB/s | 82.8us | 153.5us | 256.8us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 56489.20 MB/s
horse        █████████████████████████████ 55471.51 MB/s
herd         █████████████████████████████ 55181.86 MB/s
zebra        █████████████████████████████ 55142.11 MB/s
usagi        ████████ 15620.81 MB/s
rabbit       ██████ 11882.81 MB/s
bee3         █████ 10106.21 MB/s
```

**Latency (P50)**
```
pony         ██████ 17.3us
horse        ██████ 17.6us
herd         ██████ 17.6us
zebra        ██████ 17.5us
usagi        ██████████████████████ 62.4us
rabbit       ████████████████████████████ 77.6us
bee3         ██████████████████████████████ 82.8us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 51491.41 MB/s | 1.2us | 1.5us | 1.5us | 0 |
| pony | 51035.21 MB/s | 1.2us | 1.5us | 1.6us | 0 |
| rabbit | 50089.66 MB/s | 1.2us | 1.5us | 2.5us | 0 |
| herd | 49009.12 MB/s | 1.2us | 1.5us | 2.6us | 0 |
| zebra | 47982.45 MB/s | 1.2us | 1.5us | 3.7us | 0 |
| bee3 | 42898.85 MB/s | 1.3us | 1.8us | 3.8us | 0 |
| usagi | 12572.86 MB/s | 3.5us | 9.2us | 22.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 51491.41 MB/s
pony         █████████████████████████████ 51035.21 MB/s
rabbit       █████████████████████████████ 50089.66 MB/s
herd         ████████████████████████████ 49009.12 MB/s
zebra        ███████████████████████████ 47982.45 MB/s
bee3         ████████████████████████ 42898.85 MB/s
usagi        ███████ 12572.86 MB/s
```

**Latency (P50)**
```
horse        ██████████ 1.2us
pony         ██████████ 1.2us
rabbit       ██████████ 1.2us
herd         ██████████ 1.2us
zebra        ██████████ 1.2us
bee3         ███████████ 1.3us
usagi        ██████████████████████████████ 3.5us
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 342936 ops/s | 2.9us | 2.9us | 2.9us | 0 |
| horse | 338066 ops/s | 3.0us | 3.0us | 3.0us | 0 |
| pony | 195122 ops/s | 5.1us | 5.1us | 5.1us | 0 |
| zebra | 141163 ops/s | 7.1us | 7.1us | 7.1us | 0 |
| bee3 | 53932 ops/s | 18.5us | 18.5us | 18.5us | 0 |
| usagi | 30227 ops/s | 33.1us | 33.1us | 33.1us | 0 |
| rabbit | 20067 ops/s | 49.8us | 49.8us | 49.8us | 0 |

**Throughput**
```
herd         ██████████████████████████████ 342936 ops/s
horse        █████████████████████████████ 338066 ops/s
pony         █████████████████ 195122 ops/s
zebra        ████████████ 141163 ops/s
bee3         ████ 53932 ops/s
usagi        ██ 30227 ops/s
rabbit       █ 20067 ops/s
```

**Latency (P50)**
```
herd         █ 2.9us
horse        █ 3.0us
pony         ███ 5.1us
zebra        ████ 7.1us
bee3         ███████████ 18.5us
usagi        ███████████████████ 33.1us
rabbit       ██████████████████████████████ 49.8us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 95611 ops/s | 10.5us | 10.5us | 10.5us | 0 |
| pony | 86640 ops/s | 11.5us | 11.5us | 11.5us | 0 |
| zebra | 80000 ops/s | 12.5us | 12.5us | 12.5us | 0 |
| horse | 75002 ops/s | 13.3us | 13.3us | 13.3us | 0 |
| bee3 | 12896 ops/s | 77.5us | 77.5us | 77.5us | 0 |
| usagi | 10591 ops/s | 94.4us | 94.4us | 94.4us | 0 |
| rabbit | 3961 ops/s | 252.5us | 252.5us | 252.5us | 0 |

**Throughput**
```
herd         ██████████████████████████████ 95611 ops/s
pony         ███████████████████████████ 86640 ops/s
zebra        █████████████████████████ 80000 ops/s
horse        ███████████████████████ 75002 ops/s
bee3         ████ 12896 ops/s
usagi        ███ 10591 ops/s
rabbit       █ 3961 ops/s
```

**Latency (P50)**
```
herd         █ 10.5us
pony         █ 11.5us
zebra        █ 12.5us
horse        █ 13.3us
bee3         █████████ 77.5us
usagi        ███████████ 94.4us
rabbit       ██████████████████████████████ 252.5us
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 42935 ops/s | 23.3us | 23.3us | 23.3us | 0 |
| zebra | 26548 ops/s | 37.7us | 37.7us | 37.7us | 0 |
| horse | 9427 ops/s | 106.1us | 106.1us | 106.1us | 0 |
| pony | 7282 ops/s | 137.3us | 137.3us | 137.3us | 0 |
| usagi | 4208 ops/s | 237.6us | 237.6us | 237.6us | 0 |
| bee3 | 2082 ops/s | 480.2us | 480.2us | 480.2us | 0 |
| rabbit | 398 ops/s | 2.5ms | 2.5ms | 2.5ms | 0 |

**Throughput**
```
herd         ██████████████████████████████ 42935 ops/s
zebra        ██████████████████ 26548 ops/s
horse        ██████ 9427 ops/s
pony         █████ 7282 ops/s
usagi        ██ 4208 ops/s
bee3         █ 2082 ops/s
rabbit       █ 398 ops/s
```

**Latency (P50)**
```
herd         █ 23.3us
zebra        █ 37.7us
horse        █ 106.1us
pony         █ 137.3us
usagi        ██ 237.6us
bee3         █████ 480.2us
rabbit       ██████████████████████████████ 2.5ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 3432 ops/s | 291.3us | 291.3us | 291.3us | 0 |
| zebra | 2973 ops/s | 336.4us | 336.4us | 336.4us | 0 |
| pony | 1940 ops/s | 515.4us | 515.4us | 515.4us | 0 |
| horse | 1654 ops/s | 604.5us | 604.5us | 604.5us | 0 |
| usagi | 530 ops/s | 1.9ms | 1.9ms | 1.9ms | 0 |
| bee3 | 205 ops/s | 4.9ms | 4.9ms | 4.9ms | 0 |
| rabbit | 31 ops/s | 32.2ms | 32.2ms | 32.2ms | 0 |

**Throughput**
```
herd         ██████████████████████████████ 3432 ops/s
zebra        █████████████████████████ 2973 ops/s
pony         ████████████████ 1940 ops/s
horse        ██████████████ 1654 ops/s
usagi        ████ 530 ops/s
bee3         █ 205 ops/s
rabbit       █ 31 ops/s
```

**Latency (P50)**
```
herd         █ 291.3us
zebra        █ 336.4us
pony         █ 515.4us
horse        █ 604.5us
usagi        █ 1.9ms
bee3         ████ 4.9ms
rabbit       ██████████████████████████████ 32.2ms
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 210526 ops/s | 4.8us | 4.8us | 4.8us | 0 |
| herd | 84509 ops/s | 11.8us | 11.8us | 11.8us | 0 |
| rabbit | 28777 ops/s | 34.8us | 34.8us | 34.8us | 0 |
| bee3 | 62 ops/s | 16.0ms | 16.0ms | 16.0ms | 0 |
| usagi | 2 ops/s | 558.1ms | 558.1ms | 558.1ms | 0 |
| pony | 1 ops/s | 1.54s | 1.54s | 1.54s | 0 |
| zebra | 1 ops/s | 1.57s | 1.57s | 1.57s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 210526 ops/s
herd         ████████████ 84509 ops/s
rabbit       ████ 28777 ops/s
bee3         █ 62 ops/s
usagi        █ 2 ops/s
pony         █ 1 ops/s
zebra        █ 1 ops/s
```

**Latency (P50)**
```
horse        █ 4.8us
herd         █ 11.8us
rabbit       █ 34.8us
bee3         █ 16.0ms
usagi        ██████████ 558.1ms
pony         █████████████████████████████ 1.54s
zebra        ██████████████████████████████ 1.57s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 101698 ops/s | 9.8us | 9.8us | 9.8us | 0 |
| horse | 50105 ops/s | 20.0us | 20.0us | 20.0us | 0 |
| rabbit | 26997 ops/s | 37.0us | 37.0us | 37.0us | 0 |
| zebra | 17831 ops/s | 56.1us | 56.1us | 56.1us | 0 |
| bee3 | 62 ops/s | 16.1ms | 16.1ms | 16.1ms | 0 |
| usagi | 2 ops/s | 625.1ms | 625.1ms | 625.1ms | 0 |
| pony | 1 ops/s | 1.53s | 1.53s | 1.53s | 0 |

**Throughput**
```
herd         ██████████████████████████████ 101698 ops/s
horse        ██████████████ 50105 ops/s
rabbit       ███████ 26997 ops/s
zebra        █████ 17831 ops/s
bee3         █ 62 ops/s
usagi        █ 2 ops/s
pony         █ 1 ops/s
```

**Latency (P50)**
```
herd         █ 9.8us
horse        █ 20.0us
rabbit       █ 37.0us
zebra        █ 56.1us
bee3         █ 16.1ms
usagi        ████████████ 625.1ms
pony         ██████████████████████████████ 1.53s
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 24440 ops/s | 40.9us | 40.9us | 40.9us | 0 |
| horse | 16506 ops/s | 60.6us | 60.6us | 60.6us | 0 |
| zebra | 12712 ops/s | 78.7us | 78.7us | 78.7us | 0 |
| rabbit | 4421 ops/s | 226.2us | 226.2us | 226.2us | 0 |
| bee3 | 63 ops/s | 15.8ms | 15.8ms | 15.8ms | 0 |
| usagi | 2 ops/s | 601.7ms | 601.7ms | 601.7ms | 0 |
| pony | 1 ops/s | 1.56s | 1.56s | 1.56s | 0 |

**Throughput**
```
herd         ██████████████████████████████ 24440 ops/s
horse        ████████████████████ 16506 ops/s
zebra        ███████████████ 12712 ops/s
rabbit       █████ 4421 ops/s
bee3         █ 63 ops/s
usagi        █ 2 ops/s
pony         █ 1 ops/s
```

**Latency (P50)**
```
herd         █ 40.9us
horse        █ 60.6us
zebra        █ 78.7us
rabbit       █ 226.2us
bee3         █ 15.8ms
usagi        ███████████ 601.7ms
pony         ██████████████████████████████ 1.56s
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 3653 ops/s | 273.8us | 273.8us | 273.8us | 0 |
| herd | 3546 ops/s | 282.0us | 282.0us | 282.0us | 0 |
| horse | 2448 ops/s | 408.4us | 408.4us | 408.4us | 0 |
| rabbit | 447 ops/s | 2.2ms | 2.2ms | 2.2ms | 0 |
| bee3 | 59 ops/s | 16.8ms | 16.8ms | 16.8ms | 0 |
| usagi | 1 ops/s | 703.7ms | 703.7ms | 703.7ms | 0 |
| pony | 1 ops/s | 1.53s | 1.53s | 1.53s | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 3653 ops/s
herd         █████████████████████████████ 3546 ops/s
horse        ████████████████████ 2448 ops/s
rabbit       ███ 447 ops/s
bee3         █ 59 ops/s
usagi        █ 1 ops/s
pony         █ 1 ops/s
```

**Latency (P50)**
```
zebra        █ 273.8us
herd         █ 282.0us
horse        █ 408.4us
rabbit       █ 2.2ms
bee3         █ 16.8ms
usagi        █████████████ 703.7ms
pony         ██████████████████████████████ 1.53s
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 52.32 MB/s | 4.7us | 4.7us | 4.7us | 0 |
| zebra | 28.44 MB/s | 8.6us | 8.6us | 8.6us | 0 |
| bee3 | 25.93 MB/s | 9.4us | 9.4us | 9.4us | 0 |
| usagi | 20.93 MB/s | 11.7us | 11.7us | 11.7us | 0 |
| herd | 13.20 MB/s | 18.5us | 18.5us | 18.5us | 0 |
| horse | 12.63 MB/s | 19.3us | 19.3us | 19.3us | 0 |
| rabbit | 1.82 MB/s | 134.2us | 134.2us | 134.2us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 52.32 MB/s
zebra        ████████████████ 28.44 MB/s
bee3         ██████████████ 25.93 MB/s
usagi        ███████████ 20.93 MB/s
herd         ███████ 13.20 MB/s
horse        ███████ 12.63 MB/s
rabbit       █ 1.82 MB/s
```

**Latency (P50)**
```
pony         █ 4.7us
zebra        █ 8.6us
bee3         ██ 9.4us
usagi        ██ 11.7us
herd         ████ 18.5us
horse        ████ 19.3us
rabbit       ██████████████████████████████ 134.2us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 174.91 MB/s | 14.0us | 14.0us | 14.0us | 0 |
| horse | 95.43 MB/s | 25.6us | 25.6us | 25.6us | 0 |
| zebra | 67.04 MB/s | 36.4us | 36.4us | 36.4us | 0 |
| herd | 53.66 MB/s | 45.5us | 45.5us | 45.5us | 0 |
| usagi | 40.24 MB/s | 60.7us | 60.7us | 60.7us | 0 |
| bee3 | 29.64 MB/s | 82.4us | 82.4us | 82.4us | 0 |
| rabbit | 5.03 MB/s | 485.5us | 485.5us | 485.5us | 0 |

**Throughput**
```
pony         ██████████████████████████████ 174.91 MB/s
horse        ████████████████ 95.43 MB/s
zebra        ███████████ 67.04 MB/s
herd         █████████ 53.66 MB/s
usagi        ██████ 40.24 MB/s
bee3         █████ 29.64 MB/s
rabbit       █ 5.03 MB/s
```

**Latency (P50)**
```
pony         █ 14.0us
horse        █ 25.6us
zebra        ██ 36.4us
herd         ██ 45.5us
usagi        ███ 60.7us
bee3         █████ 82.4us
rabbit       ██████████████████████████████ 485.5us
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| pony | 287.51 MB/s | 84.9us | 84.9us | 84.9us | 0 |
| zebra | 189.44 MB/s | 128.9us | 128.9us | 128.9us | 0 |
| horse | 129.49 MB/s | 188.5us | 188.5us | 188.5us | 0 |
| herd | 124.93 MB/s | 195.4us | 195.4us | 195.4us | 0 |
| usagi | 80.47 MB/s | 303.4us | 303.4us | 303.4us | 0 |
| bee3 | 48.41 MB/s | 504.3us | 504.3us | 504.3us | 0 |
| rabbit | 5.27 MB/s | 4.6ms | 4.6ms | 4.6ms | 0 |

**Throughput**
```
pony         ██████████████████████████████ 287.51 MB/s
zebra        ███████████████████ 189.44 MB/s
horse        █████████████ 129.49 MB/s
herd         █████████████ 124.93 MB/s
usagi        ████████ 80.47 MB/s
bee3         █████ 48.41 MB/s
rabbit       █ 5.27 MB/s
```

**Latency (P50)**
```
pony         █ 84.9us
zebra        █ 128.9us
horse        █ 188.5us
herd         █ 195.4us
usagi        █ 303.4us
bee3         ███ 504.3us
rabbit       ██████████████████████████████ 4.6ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 338.14 MB/s | 722.0us | 722.0us | 722.0us | 0 |
| zebra | 250.63 MB/s | 974.1us | 974.1us | 974.1us | 0 |
| horse | 198.22 MB/s | 1.2ms | 1.2ms | 1.2ms | 0 |
| pony | 192.34 MB/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| usagi | 88.37 MB/s | 2.8ms | 2.8ms | 2.8ms | 0 |
| bee3 | 42.50 MB/s | 5.7ms | 5.7ms | 5.7ms | 0 |
| rabbit | 5.40 MB/s | 45.2ms | 45.2ms | 45.2ms | 0 |

**Throughput**
```
herd         ██████████████████████████████ 338.14 MB/s
zebra        ██████████████████████ 250.63 MB/s
horse        █████████████████ 198.22 MB/s
pony         █████████████████ 192.34 MB/s
usagi        ███████ 88.37 MB/s
bee3         ███ 42.50 MB/s
rabbit       █ 5.40 MB/s
```

**Latency (P50)**
```
herd         █ 722.0us
zebra        █ 974.1us
horse        █ 1.2ms
pony         █ 1.3ms
usagi        █ 2.8ms
bee3         ███ 5.7ms
rabbit       ██████████████████████████████ 45.2ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 20366216 ops/s | 42ns | 83ns | 209ns | 0 |
| horse | 16690392 ops/s | 42ns | 84ns | 208ns | 0 |
| zebra | 14500512 ops/s | 42ns | 84ns | 209ns | 0 |
| pony | 13892447 ops/s | 42ns | 84ns | 250ns | 0 |
| herd | 10707705 ops/s | 42ns | 166ns | 375ns | 0 |
| rabbit | 4536211 ops/s | 166ns | 417ns | 1.4us | 0 |
| bee3 | 3498763 ops/s | 208ns | 500ns | 1.6us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 20366216 ops/s
horse        ████████████████████████ 16690392 ops/s
zebra        █████████████████████ 14500512 ops/s
pony         ████████████████████ 13892447 ops/s
herd         ███████████████ 10707705 ops/s
rabbit       ██████ 4536211 ops/s
bee3         █████ 3498763 ops/s
```

**Latency (P50)**
```
usagi        ██████ 42ns
horse        ██████ 42ns
zebra        ██████ 42ns
pony         ██████ 42ns
herd         ██████ 42ns
rabbit       ███████████████████████ 166ns
bee3         ██████████████████████████████ 208ns
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2790.36 MB/s | 293.4us | 18.2ms | 36.1ms | 0 |
| rabbit | 1514.59 MB/s | 6.4ms | 8.6ms | 15.7ms | 0 |
| usagi | 1509.45 MB/s | 4.4ms | 14.0ms | 17.8ms | 0 |
| pony | 1141.41 MB/s | 8.1ms | 11.3ms | 13.3ms | 0 |
| herd | 682.86 MB/s | 4.6ms | 72.3ms | 76.5ms | 0 |
| bee3 | 324.85 MB/s | 21.0ms | 83.0ms | 99.1ms | 0 |
| zebra | 124.44 MB/s | 202.1us | 139.5ms | 313.3ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2790.36 MB/s
rabbit       ████████████████ 1514.59 MB/s
usagi        ████████████████ 1509.45 MB/s
pony         ████████████ 1141.41 MB/s
herd         ███████ 682.86 MB/s
bee3         ███ 324.85 MB/s
zebra        █ 124.44 MB/s
```

**Latency (P50)**
```
horse        █ 293.4us
rabbit       █████████ 6.4ms
usagi        ██████ 4.4ms
pony         ███████████ 8.1ms
herd         ██████ 4.6ms
bee3         ██████████████████████████████ 21.0ms
zebra        █ 202.1us
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1340.93 MB/s | 375ns | 1.8us | 5.6us | 0 |
| horse | 1294.39 MB/s | 500ns | 959ns | 2.8us | 0 |
| pony | 1247.66 MB/s | 417ns | 834ns | 1.5us | 0 |
| herd | 1190.53 MB/s | 500ns | 1.6us | 3.9us | 0 |
| usagi | 265.20 MB/s | 2.2us | 4.8us | 9.2us | 0 |
| bee3 | 135.89 MB/s | 4.4us | 15.6us | 35.7us | 0 |
| rabbit | 19.17 MB/s | 45.1us | 63.8us | 153.9us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1340.93 MB/s
horse        ████████████████████████████ 1294.39 MB/s
pony         ███████████████████████████ 1247.66 MB/s
herd         ██████████████████████████ 1190.53 MB/s
usagi        █████ 265.20 MB/s
bee3         ███ 135.89 MB/s
rabbit       █ 19.17 MB/s
```

**Latency (P50)**
```
zebra        █ 375ns
horse        █ 500ns
pony         █ 417ns
herd         █ 500ns
usagi        █ 2.2us
bee3         ██ 4.4us
rabbit       ██████████████████████████████ 45.1us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2065.65 MB/s | 23.9us | 173.4us | 15.0ms | 0 |
| rabbit | 1807.10 MB/s | 314.8us | 1.3ms | 2.0ms | 0 |
| herd | 1720.01 MB/s | 34.0us | 992.8us | 10.8ms | 0 |
| usagi | 1403.80 MB/s | 221.3us | 1.3ms | 9.0ms | 0 |
| pony | 1127.79 MB/s | 21.0us | 2.7ms | 4.1ms | 0 |
| bee3 | 931.40 MB/s | 604.6us | 2.5ms | 9.6ms | 0 |
| zebra | 837.58 MB/s | 20.6us | 124.4us | 850.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2065.65 MB/s
rabbit       ██████████████████████████ 1807.10 MB/s
herd         ████████████████████████ 1720.01 MB/s
usagi        ████████████████████ 1403.80 MB/s
pony         ████████████████ 1127.79 MB/s
bee3         █████████████ 931.40 MB/s
zebra        ████████████ 837.58 MB/s
```

**Latency (P50)**
```
horse        █ 23.9us
rabbit       ███████████████ 314.8us
herd         █ 34.0us
usagi        ██████████ 221.3us
pony         █ 21.0us
bee3         ██████████████████████████████ 604.6us
zebra        █ 20.6us
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd | 9387.01 MB/s | 2.4us | 13.0us | 36.2us | 0 |
| zebra | 4068.86 MB/s | 2.2us | 9.4us | 36.8us | 0 |
| pony | 2446.27 MB/s | 1.4us | 2.2us | 176.9us | 0 |
| horse | 2092.08 MB/s | 2.0us | 3.5us | 10.5us | 0 |
| usagi | 1119.72 MB/s | 19.5us | 46.8us | 398.2us | 0 |
| bee3 | 1010.11 MB/s | 37.7us | 96.2us | 648.5us | 0 |
| rabbit | 755.27 MB/s | 60.9us | 97.6us | 511.1us | 0 |

**Throughput**
```
herd         ██████████████████████████████ 9387.01 MB/s
zebra        █████████████ 4068.86 MB/s
pony         ███████ 2446.27 MB/s
horse        ██████ 2092.08 MB/s
usagi        ███ 1119.72 MB/s
bee3         ███ 1010.11 MB/s
rabbit       ██ 755.27 MB/s
```

**Latency (P50)**
```
herd         █ 2.4us
zebra        █ 2.2us
pony         █ 1.4us
horse        █ 2.0us
usagi        █████████ 19.5us
bee3         ██████████████████ 37.7us
rabbit       ██████████████████████████████ 60.9us
```

## Runtime Resource Usage

| Driver | Peak RSS | Go Heap | Go Sys | Disk Usage | GC Cycles |
|--------|----------|---------|--------|------------|----------|
| bee3 | 6317.6 MB | 7807.9 MB | 8923.8 MB | 8698.7 MB | 420 |
| herd | 7072.1 MB | 8138.5 MB | 8991.1 MB | 16384.0 MB | 433 |
| horse | 4100.2 MB | 2701.0 MB | 4801.9 MB | 32768.0 MB | 299 |
| pony | 4100.2 MB | 1898.9 MB | 4807.1 MB | 17482.6 MB | 391 |
| rabbit | 3545.5 MB | 1038.7 MB | 4800.7 MB | 4724.1 MB | 37 |
| usagi | 4100.2 MB | 2701.0 MB | 4801.6 MB | 8736.1 MB | 257 |
| zebra | 7232.9 MB | 8666.6 MB | 9428.9 MB | 16384.0 MB | 441 |

> **Note:** Peak RSS = process peak resident set size (includes mmap). Go Heap/Sys = Go runtime allocations. Disk = data directory size after benchmark.

## Recommendations

- **Write-heavy workloads:** herd
- **Read-heavy workloads:** pony

---

*Generated by storage benchmark CLI*
