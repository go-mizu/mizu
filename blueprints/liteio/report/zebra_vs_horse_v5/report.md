# Storage Benchmark Report

**Generated:** 2026-02-19T14:13:45+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** horse (won 31/40 benchmarks, 78%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | horse | 31 | 78% |
| 2 | zebra | 9 | 22% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | horse | 9.3 GB/s | +68% vs zebra |
| Small Write (1KB) | zebra | 1.6 GB/s | +12% vs horse |
| Large Read (10MB) | horse | 94441.6 GB/s | +84% vs zebra |
| Large Write (10MB) | horse | 414.2 MB/s | +61% vs zebra |
| Delete | zebra | 2.8M ops/s | +13% vs horse |
| Stat | horse | 17.2M ops/s | +56% vs zebra |
| List (100 objects) | horse | 126.8K ops/s | 2739.9x vs zebra |
| Copy | zebra | 913.7 MB/s | 6.1x vs horse |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **horse** | 414 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **horse** | 94441585 MB/s | Best for streaming, CDN |
| Small File Operations | **horse** | 5483901 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **horse** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| horse | 414.2 | 94441585.2 | 243.0us | 84ns |
| zebra | 258.0 | 51205015.4 | 271.8us | 125ns |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| horse | 1437673 | 9530129 | 500ns | 84ns |
| zebra | 1605706 | 5668190 | 375ns | 84ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| horse | 17173933 | 126803 | 2449618 |
| zebra | 11042430 | 46 | 2769948 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 968.20 | 34.73 | 37.99 |
| zebra | 851.42 | 310.23 | 129.04 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 6380.89 | 5148.76 | 4302.43 |
| zebra | 5736.60 | 2505.36 | 2313.44 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 9.2us | 11.8us | 145.6us | 1.3ms |
| zebra | 6.8us | 34.7us | 216.3us | 1.5ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 2.9us | 4.5us | 35.4us | 229.6us |
| zebra | 244.0ms | 208.1ms | 152.5ms | 173.5ms |

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
| zebra | 913.75 MB/s | 750ns | 2.2us | 3.6us | 0 |
| horse | 149.55 MB/s | 958ns | 1.5us | 7.6us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 913.75 MB/s
horse        ████ 149.55 MB/s
```

**Latency (P50)**
```
zebra        ███████████████████████ 750ns
horse        ██████████████████████████████ 958ns
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 2769948 ops/s | 333ns | 583ns | 833ns | 0 |
| horse | 2449618 ops/s | 375ns | 583ns | 833ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 2769948 ops/s
horse        ██████████████████████████ 2449618 ops/s
```

**Latency (P50)**
```
zebra        ██████████████████████████ 333ns
horse        ██████████████████████████████ 375ns
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 78.25 MB/s | 584ns | 1.3us | 6.4us | 0 |
| zebra | 55.67 MB/s | 708ns | 3.2us | 20.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 78.25 MB/s
zebra        █████████████████████ 55.67 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████ 584ns
zebra        ██████████████████████████████ 708ns
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1471583 ops/s | 500ns | 1.2us | 2.0us | 0 |
| zebra | 714099 ops/s | 833ns | 1.6us | 3.8us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1471583 ops/s
zebra        ██████████████ 714099 ops/s
```

**Latency (P50)**
```
horse        ██████████████████ 500ns
zebra        ██████████████████████████████ 833ns
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 87.93 MB/s | 750ns | 1.7us | 2.7us | 0 |
| zebra | 27.75 MB/s | 1.3us | 3.9us | 69.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 87.93 MB/s
zebra        █████████ 27.75 MB/s
```

**Latency (P50)**
```
horse        █████████████████ 750ns
zebra        ██████████████████████████████ 1.3us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 126803 ops/s | 6.8us | 12.5us | 28.2us | 0 |
| zebra | 46 ops/s | 21.6ms | 24.1ms | 24.2ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 126803 ops/s
zebra        █ 46 ops/s
```

**Latency (P50)**
```
horse        █ 6.8us
zebra        ██████████████████████████████ 21.6ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 177.43 MB/s | 9.9us | 306.0us | 1.7ms | 0 |
| horse | 19.52 MB/s | 8.1us | 3.1ms | 6.9ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 177.43 MB/s
horse        ███ 19.52 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 9.9us
horse        ████████████████████████ 8.1us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 147.44 MB/s | 8.9us | 276.9us | 1.4ms | 0 |
| horse | 81.28 MB/s | 6.1us | 1.1ms | 3.4ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 147.44 MB/s
horse        ████████████████ 81.28 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 8.9us
horse        ████████████████████ 6.1us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 21.82 MB/s | 4.5us | 526.9us | 13.6ms | 0 |
| horse | 2.84 MB/s | 5.4us | 2.9ms | 137.0ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 21.82 MB/s
horse        ███ 2.84 MB/s
```

**Latency (P50)**
```
zebra        █████████████████████████ 4.5us
horse        ██████████████████████████████ 5.4us
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 106.24 MB/s | 64.5ms | 270.0ms | 270.0ms | 0 |
| zebra | 84.05 MB/s | 121.7ms | 121.7ms | 121.7ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 106.24 MB/s
zebra        ███████████████████████ 84.05 MB/s
```

**Latency (P50)**
```
horse        ███████████████ 64.5ms
zebra        ██████████████████████████████ 121.7ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 6380.89 MB/s | 125ns | 250ns | 459ns | 0 |
| zebra | 5736.60 MB/s | 125ns | 250ns | 1.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 6380.89 MB/s
zebra        ██████████████████████████ 5736.60 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 125ns
zebra        ██████████████████████████████ 125ns
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5148.76 MB/s | 125ns | 375ns | 875ns | 0 |
| zebra | 2505.36 MB/s | 292ns | 542ns | 1.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5148.76 MB/s
zebra        ██████████████ 2505.36 MB/s
```

**Latency (P50)**
```
horse        ████████████ 125ns
zebra        ██████████████████████████████ 292ns
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 4302.43 MB/s | 166ns | 458ns | 1.1us | 0 |
| zebra | 2313.44 MB/s | 292ns | 625ns | 1.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 4302.43 MB/s
zebra        ████████████████ 2313.44 MB/s
```

**Latency (P50)**
```
horse        █████████████████ 166ns
zebra        ██████████████████████████████ 292ns
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 968.20 MB/s | 875ns | 1.4us | 2.1us | 0 |
| zebra | 851.42 MB/s | 958ns | 2.3us | 3.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 968.20 MB/s
zebra        ██████████████████████████ 851.42 MB/s
```

**Latency (P50)**
```
horse        ███████████████████████████ 875ns
zebra        ██████████████████████████████ 958ns
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 310.23 MB/s | 1.5us | 3.5us | 22.2us | 0 |
| horse | 34.73 MB/s | 1.6us | 20.0us | 56.4us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 310.23 MB/s
horse        ███ 34.73 MB/s
```

**Latency (P50)**
```
zebra        █████████████████████████████ 1.5us
horse        ██████████████████████████████ 1.6us
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 129.04 MB/s | 2.0us | 4.1us | 113.0us | 0 |
| horse | 37.99 MB/s | 2.2us | 115.8us | 274.2us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 129.04 MB/s
horse        ████████ 37.99 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████ 2.0us
horse        ██████████████████████████████ 2.2us
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2431476.52 MB/s | 83ns | 125ns | 292ns | 0 |
| zebra | 869096.18 MB/s | 250ns | 292ns | 708ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2431476.52 MB/s
zebra        ██████████ 869096.18 MB/s
```

**Latency (P50)**
```
horse        █████████ 83ns
zebra        ██████████████████████████████ 250ns
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2352804.47 MB/s | 83ns | 167ns | 292ns | 0 |
| zebra | 887902.53 MB/s | 250ns | 292ns | 708ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2352804.47 MB/s
zebra        ███████████ 887902.53 MB/s
```

**Latency (P50)**
```
horse        █████████ 83ns
zebra        ██████████████████████████████ 250ns
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2340597.90 MB/s | 84ns | 125ns | 292ns | 0 |
| zebra | 589530.89 MB/s | 250ns | 334ns | 959ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2340597.90 MB/s
zebra        ███████ 589530.89 MB/s
```

**Latency (P50)**
```
horse        ██████████ 84ns
zebra        ██████████████████████████████ 250ns
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 94441585.18 MB/s | 84ns | 125ns | 334ns | 0 |
| zebra | 51205015.39 MB/s | 125ns | 250ns | 2.1us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 94441585.18 MB/s
zebra        ████████████████ 51205015.39 MB/s
```

**Latency (P50)**
```
horse        ████████████████████ 84ns
zebra        ██████████████████████████████ 125ns
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 9306.77 MB/s | 84ns | 125ns | 292ns | 0 |
| zebra | 5535.34 MB/s | 84ns | 167ns | 1.9us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 9306.77 MB/s
zebra        █████████████████ 5535.34 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 84ns
zebra        ██████████████████████████████ 84ns
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 10270838.59 MB/s | 83ns | 125ns | 292ns | 0 |
| zebra | 4141257.85 MB/s | 125ns | 250ns | 2.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 10270838.59 MB/s
zebra        ████████████ 4141257.85 MB/s
```

**Latency (P50)**
```
horse        ███████████████████ 83ns
zebra        ██████████████████████████████ 125ns
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 632590.11 MB/s | 83ns | 125ns | 291ns | 0 |
| zebra | 373541.10 MB/s | 84ns | 166ns | 1.6us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 632590.11 MB/s
zebra        █████████████████ 373541.10 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 648508 ops/s | 1.5us | 1.5us | 1.5us | 0 |
| zebra | 114837 ops/s | 8.7us | 8.7us | 8.7us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 648508 ops/s
zebra        █████ 114837 ops/s
```

**Latency (P50)**
```
horse        █████ 1.5us
zebra        ██████████████████████████████ 8.7us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 252653 ops/s | 4.0us | 4.0us | 4.0us | 0 |
| zebra | 35820 ops/s | 27.9us | 27.9us | 27.9us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 252653 ops/s
zebra        ████ 35820 ops/s
```

**Latency (P50)**
```
horse        ████ 4.0us
zebra        ██████████████████████████████ 27.9us
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 39933 ops/s | 25.0us | 25.0us | 25.0us | 0 |
| zebra | 7249 ops/s | 138.0us | 138.0us | 138.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 39933 ops/s
zebra        █████ 7249 ops/s
```

**Latency (P50)**
```
horse        █████ 25.0us
zebra        ██████████████████████████████ 138.0us
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 3826 ops/s | 261.4us | 261.4us | 261.4us | 0 |
| zebra | 645 ops/s | 1.6ms | 1.6ms | 1.6ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 3826 ops/s
zebra        █████ 645 ops/s
```

**Latency (P50)**
```
horse        █████ 261.4us
zebra        ██████████████████████████████ 1.6ms
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 342818 ops/s | 2.9us | 2.9us | 2.9us | 0 |
| zebra | 4 ops/s | 244.0ms | 244.0ms | 244.0ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 342818 ops/s
zebra        █ 4 ops/s
```

**Latency (P50)**
```
horse        █ 2.9us
zebra        ██████████████████████████████ 244.0ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 224316 ops/s | 4.5us | 4.5us | 4.5us | 0 |
| zebra | 5 ops/s | 208.1ms | 208.1ms | 208.1ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 224316 ops/s
zebra        █ 5 ops/s
```

**Latency (P50)**
```
horse        █ 4.5us
zebra        ██████████████████████████████ 208.1ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 28269 ops/s | 35.4us | 35.4us | 35.4us | 0 |
| zebra | 7 ops/s | 152.5ms | 152.5ms | 152.5ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 28269 ops/s
zebra        █ 7 ops/s
```

**Latency (P50)**
```
horse        █ 35.4us
zebra        ██████████████████████████████ 152.5ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 4356 ops/s | 229.6us | 229.6us | 229.6us | 0 |
| zebra | 6 ops/s | 173.5ms | 173.5ms | 173.5ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 4356 ops/s
zebra        █ 6 ops/s
```

**Latency (P50)**
```
horse        █ 229.6us
zebra        ██████████████████████████████ 173.5ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 36.17 MB/s | 6.8us | 6.8us | 6.8us | 0 |
| horse | 26.39 MB/s | 9.2us | 9.2us | 9.2us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 36.17 MB/s
horse        █████████████████████ 26.39 MB/s
```

**Latency (P50)**
```
zebra        █████████████████████ 6.8us
horse        ██████████████████████████████ 9.2us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 206.32 MB/s | 11.8us | 11.8us | 11.8us | 0 |
| zebra | 70.42 MB/s | 34.7us | 34.7us | 34.7us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 206.32 MB/s
zebra        ██████████ 70.42 MB/s
```

**Latency (P50)**
```
horse        ██████████ 11.8us
zebra        ██████████████████████████████ 34.7us
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 167.65 MB/s | 145.6us | 145.6us | 145.6us | 0 |
| zebra | 112.85 MB/s | 216.3us | 216.3us | 216.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 167.65 MB/s
zebra        ████████████████████ 112.85 MB/s
```

**Latency (P50)**
```
horse        ████████████████████ 145.6us
zebra        ██████████████████████████████ 216.3us
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 191.92 MB/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| zebra | 162.53 MB/s | 1.5ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 191.92 MB/s
zebra        █████████████████████████ 162.53 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████ 1.3ms
zebra        ██████████████████████████████ 1.5ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 17173933 ops/s | 42ns | 84ns | 167ns | 0 |
| zebra | 11042430 ops/s | 42ns | 84ns | 1.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 17173933 ops/s
zebra        ███████████████████ 11042430 ops/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 42ns
zebra        ██████████████████████████████ 42ns
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 414.16 MB/s | 243.0us | 122.0ms | 365.1ms | 0 |
| zebra | 257.96 MB/s | 271.8us | 1.1ms | 1.21s | 0 |

**Throughput**
```
horse        ██████████████████████████████ 414.16 MB/s
zebra        ██████████████████ 257.96 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████ 243.0us
zebra        ██████████████████████████████ 271.8us
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1568.07 MB/s | 375ns | 959ns | 3.7us | 0 |
| horse | 1403.98 MB/s | 500ns | 1.0us | 1.8us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1568.07 MB/s
horse        ██████████████████████████ 1403.98 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████ 375ns
horse        ██████████████████████████████ 500ns
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 769.70 MB/s | 21.0us | 28.1us | 51.9ms | 0 |
| zebra | 322.82 MB/s | 20.1us | 87.4us | 235.8us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 769.70 MB/s
zebra        ████████████ 322.82 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 21.0us
zebra        ████████████████████████████ 20.1us
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1496.43 MB/s | 2.0us | 4.5us | 7.0us | 0 |
| zebra | 523.61 MB/s | 2.0us | 5.9us | 14.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1496.43 MB/s
zebra        ██████████ 523.61 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████████ 2.0us
zebra        ██████████████████████████████ 2.0us
```

## Recommendations

- **Write-heavy workloads:** zebra
- **Read-heavy workloads:** horse

---

*Generated by storage benchmark CLI*
