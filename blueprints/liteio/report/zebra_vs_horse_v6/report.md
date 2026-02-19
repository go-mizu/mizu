# Storage Benchmark Report

**Generated:** 2026-02-19T14:18:02+07:00

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
| Small Read (1KB) | zebra | 8.4 GB/s | +39% vs horse |
| Small Write (1KB) | zebra | 1.7 GB/s | 2.3x vs horse |
| Large Read (10MB) | horse | 66648.1 GB/s | close |
| Large Write (10MB) | horse | 1.4 GB/s | 3.6x vs zebra |
| Delete | zebra | 2.8M ops/s | +17% vs horse |
| Stat | zebra | 13.0M ops/s | +20% vs horse |
| List (100 objects) | horse | 74.2K ops/s | 1282.8x vs zebra |
| Copy | zebra | 1.3 GB/s | +78% vs horse |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **horse** | 1432 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **horse** | 66648103 MB/s | Best for streaming, CDN |
| Small File Operations | **zebra** | 5205383 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **horse** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| horse | 1431.5 | 66648102.9 | 341.0us | 125ns |
| zebra | 403.1 | 65036712.4 | 247.8us | 125ns |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| horse | 770543 | 6225050 | 792ns | 125ns |
| zebra | 1772685 | 8638081 | 375ns | 125ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| horse | 10757388 | 74247 | 2439201 |
| zebra | 12953737 | 58 | 2844479 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 1030.19 | 132.52 | 39.19 |
| zebra | 860.77 | 420.82 | 169.69 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 6464.58 | 3739.90 | 3179.85 |
| zebra | 5179.31 | 2952.32 | 2848.57 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 9.5us | 11.6us | 73.9us | 605.4us |
| zebra | 6.6us | 18.3us | 140.2us | 866.4us |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 2.9us | 3.9us | 17.7us | 181.8us |
| zebra | 123.5ms | 120.5ms | 114.0ms | 113.8ms |

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
| zebra | 1307.27 MB/s | 542ns | 959ns | 1.6us | 0 |
| horse | 733.58 MB/s | 958ns | 1.5us | 3.3us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1307.27 MB/s
horse        ████████████████ 733.58 MB/s
```

**Latency (P50)**
```
zebra        ████████████████ 542ns
horse        ██████████████████████████████ 958ns
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 2844479 ops/s | 333ns | 541ns | 791ns | 0 |
| horse | 2439201 ops/s | 375ns | 583ns | 750ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 2844479 ops/s
horse        █████████████████████████ 2439201 ops/s
```

**Latency (P50)**
```
zebra        ██████████████████████████ 333ns
horse        ██████████████████████████████ 375ns
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 151.59 MB/s | 459ns | 875ns | 1.6us | 0 |
| horse | 132.62 MB/s | 500ns | 1.3us | 2.1us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 151.59 MB/s
horse        ██████████████████████████ 132.62 MB/s
```

**Latency (P50)**
```
zebra        ███████████████████████████ 459ns
horse        ██████████████████████████████ 500ns
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1489439 ops/s | 459ns | 834ns | 1.4us | 0 |
| zebra | 1414703 ops/s | 416ns | 1.3us | 4.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1489439 ops/s
zebra        ████████████████████████████ 1414703 ops/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 459ns
zebra        ███████████████████████████ 416ns
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 107.25 MB/s | 667ns | 1.4us | 4.4us | 0 |
| horse | 106.63 MB/s | 708ns | 1.3us | 2.1us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 107.25 MB/s
horse        █████████████████████████████ 106.63 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████████████ 667ns
horse        ██████████████████████████████ 708ns
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 74247 ops/s | 11.8us | 19.2us | 35.3us | 0 |
| zebra | 58 ops/s | 17.2ms | 19.2ms | 19.4ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 74247 ops/s
zebra        █ 58 ops/s
```

**Latency (P50)**
```
horse        █ 11.8us
zebra        ██████████████████████████████ 17.2ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 21.06 MB/s | 9.8us | 3.3ms | 8.7ms | 0 |
| zebra | 5.31 MB/s | 2.1us | 54.5us | 813.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 21.06 MB/s
zebra        ███████ 5.31 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 9.8us
zebra        ██████ 2.1us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 7175.95 MB/s | 292ns | 6.2us | 27.0us | 0 |
| horse | 6.90 MB/s | 6.2us | 1.3ms | 4.2ms | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 7175.95 MB/s
horse        █ 6.90 MB/s
```

**Latency (P50)**
```
zebra        █ 292ns
horse        ██████████████████████████████ 6.2us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 4.58 MB/s | 4.8us | 2.9ms | 59.1ms | 0 |
| zebra | 1.21 MB/s | 3.7us | 239.4us | 652.5ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 4.58 MB/s
zebra        ███████ 1.21 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 4.8us
zebra        ██████████████████████ 3.7us
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 231.16 MB/s | 64.2ms | 75.2ms | 75.2ms | 0 |
| zebra | 181.93 MB/s | 47.2ms | 101.4ms | 101.4ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 231.16 MB/s
zebra        ███████████████████████ 181.93 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 64.2ms
zebra        ██████████████████████ 47.2ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 6464.58 MB/s | 125ns | 209ns | 500ns | 0 |
| zebra | 5179.31 MB/s | 125ns | 250ns | 750ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 6464.58 MB/s
zebra        ████████████████████████ 5179.31 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 125ns
zebra        ██████████████████████████████ 125ns
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 3739.90 MB/s | 208ns | 500ns | 1.1us | 0 |
| zebra | 2952.32 MB/s | 250ns | 625ns | 1.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 3739.90 MB/s
zebra        ███████████████████████ 2952.32 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████ 208ns
zebra        ██████████████████████████████ 250ns
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 3179.85 MB/s | 208ns | 583ns | 1.3us | 0 |
| zebra | 2848.57 MB/s | 250ns | 625ns | 1.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 3179.85 MB/s
zebra        ██████████████████████████ 2848.57 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████ 208ns
zebra        ██████████████████████████████ 250ns
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1030.19 MB/s | 834ns | 1.3us | 1.7us | 0 |
| zebra | 860.77 MB/s | 917ns | 2.4us | 3.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1030.19 MB/s
zebra        █████████████████████████ 860.77 MB/s
```

**Latency (P50)**
```
horse        ███████████████████████████ 834ns
zebra        ██████████████████████████████ 917ns
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 420.82 MB/s | 1.6us | 4.3us | 18.1us | 0 |
| horse | 132.52 MB/s | 1.8us | 22.5us | 63.8us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 420.82 MB/s
horse        █████████ 132.52 MB/s
```

**Latency (P50)**
```
zebra        █████████████████████████ 1.6us
horse        ██████████████████████████████ 1.8us
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 169.69 MB/s | 2.1us | 6.5us | 51.9us | 0 |
| horse | 39.19 MB/s | 3.0us | 101.0us | 214.1us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 169.69 MB/s
horse        ██████ 39.19 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████ 2.1us
horse        ██████████████████████████████ 3.0us
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2418013.94 MB/s | 83ns | 125ns | 292ns | 0 |
| zebra | 2083740.20 MB/s | 84ns | 166ns | 875ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2418013.94 MB/s
zebra        █████████████████████████ 2083740.20 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2362144.74 MB/s | 84ns | 167ns | 292ns | 0 |
| zebra | 2131509.61 MB/s | 84ns | 125ns | 292ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2362144.74 MB/s
zebra        ███████████████████████████ 2131509.61 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 84ns
zebra        ██████████████████████████████ 84ns
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2155352.79 MB/s | 84ns | 166ns | 292ns | 0 |
| zebra | 1821635.16 MB/s | 125ns | 167ns | 334ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2155352.79 MB/s
zebra        █████████████████████████ 1821635.16 MB/s
```

**Latency (P50)**
```
horse        ████████████████████ 84ns
zebra        ██████████████████████████████ 125ns
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 66648102.86 MB/s | 125ns | 209ns | 417ns | 0 |
| zebra | 65036712.37 MB/s | 125ns | 208ns | 1.7us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 66648102.86 MB/s
zebra        █████████████████████████████ 65036712.37 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 125ns
zebra        ██████████████████████████████ 125ns
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 8435.63 MB/s | 125ns | 166ns | 333ns | 0 |
| horse | 6079.15 MB/s | 125ns | 250ns | 458ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 8435.63 MB/s
horse        █████████████████████ 6079.15 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 125ns
horse        ██████████████████████████████ 125ns
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 7106658.98 MB/s | 125ns | 208ns | 375ns | 0 |
| zebra | 6703423.45 MB/s | 125ns | 167ns | 750ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 7106658.98 MB/s
zebra        ████████████████████████████ 6703423.45 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 125ns
zebra        ██████████████████████████████ 125ns
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 446464.12 MB/s | 125ns | 167ns | 1.1us | 0 |
| horse | 443827.47 MB/s | 125ns | 208ns | 334ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 446464.12 MB/s
horse        █████████████████████████████ 443827.47 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 125ns
horse        ██████████████████████████████ 125ns
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 685401 ops/s | 1.5us | 1.5us | 1.5us | 0 |
| zebra | 216216 ops/s | 4.6us | 4.6us | 4.6us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 685401 ops/s
zebra        █████████ 216216 ops/s
```

**Latency (P50)**
```
horse        █████████ 1.5us
zebra        ██████████████████████████████ 4.6us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 242424 ops/s | 4.1us | 4.1us | 4.1us | 0 |
| zebra | 65574 ops/s | 15.2us | 15.2us | 15.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 242424 ops/s
zebra        ████████ 65574 ops/s
```

**Latency (P50)**
```
horse        ████████ 4.1us
zebra        ██████████████████████████████ 15.2us
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 40816 ops/s | 24.5us | 24.5us | 24.5us | 0 |
| zebra | 11310 ops/s | 88.4us | 88.4us | 88.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 40816 ops/s
zebra        ████████ 11310 ops/s
```

**Latency (P50)**
```
horse        ████████ 24.5us
zebra        ██████████████████████████████ 88.4us
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 3833 ops/s | 260.9us | 260.9us | 260.9us | 0 |
| zebra | 1421 ops/s | 703.7us | 703.7us | 703.7us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 3833 ops/s
zebra        ███████████ 1421 ops/s
```

**Latency (P50)**
```
horse        ███████████ 260.9us
zebra        ██████████████████████████████ 703.7us
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 342818 ops/s | 2.9us | 2.9us | 2.9us | 0 |
| zebra | 8 ops/s | 123.5ms | 123.5ms | 123.5ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 342818 ops/s
zebra        █ 8 ops/s
```

**Latency (P50)**
```
horse        █ 2.9us
zebra        ██████████████████████████████ 123.5ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 258065 ops/s | 3.9us | 3.9us | 3.9us | 0 |
| zebra | 8 ops/s | 120.5ms | 120.5ms | 120.5ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 258065 ops/s
zebra        █ 8 ops/s
```

**Latency (P50)**
```
horse        █ 3.9us
zebra        ██████████████████████████████ 120.5ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 56603 ops/s | 17.7us | 17.7us | 17.7us | 0 |
| zebra | 9 ops/s | 114.0ms | 114.0ms | 114.0ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 56603 ops/s
zebra        █ 9 ops/s
```

**Latency (P50)**
```
horse        █ 17.7us
zebra        ██████████████████████████████ 114.0ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5501 ops/s | 181.8us | 181.8us | 181.8us | 0 |
| zebra | 9 ops/s | 113.8ms | 113.8ms | 113.8ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5501 ops/s
zebra        █ 9 ops/s
```

**Latency (P50)**
```
horse        █ 181.8us
zebra        ██████████████████████████████ 113.8ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 37.09 MB/s | 6.6us | 6.6us | 6.6us | 0 |
| horse | 25.59 MB/s | 9.5us | 9.5us | 9.5us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 37.09 MB/s
horse        ████████████████████ 25.59 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████ 6.6us
horse        ██████████████████████████████ 9.5us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 210.01 MB/s | 11.6us | 11.6us | 11.6us | 0 |
| zebra | 133.16 MB/s | 18.3us | 18.3us | 18.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 210.01 MB/s
zebra        ███████████████████ 133.16 MB/s
```

**Latency (P50)**
```
horse        ███████████████████ 11.6us
zebra        ██████████████████████████████ 18.3us
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 330.48 MB/s | 73.9us | 73.9us | 73.9us | 0 |
| zebra | 174.13 MB/s | 140.2us | 140.2us | 140.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 330.48 MB/s
zebra        ███████████████ 174.13 MB/s
```

**Latency (P50)**
```
horse        ███████████████ 73.9us
zebra        ██████████████████████████████ 140.2us
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 403.26 MB/s | 605.4us | 605.4us | 605.4us | 0 |
| zebra | 281.78 MB/s | 866.4us | 866.4us | 866.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 403.26 MB/s
zebra        ████████████████████ 281.78 MB/s
```

**Latency (P50)**
```
horse        ████████████████████ 605.4us
zebra        ██████████████████████████████ 866.4us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 12953737 ops/s | 42ns | 84ns | 459ns | 0 |
| horse | 10757388 ops/s | 83ns | 125ns | 250ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 12953737 ops/s
horse        ████████████████████████ 10757388 ops/s
```

**Latency (P50)**
```
zebra        ███████████████ 42ns
horse        ██████████████████████████████ 83ns
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1431.53 MB/s | 341.0us | 38.6ms | 56.0ms | 0 |
| zebra | 403.05 MB/s | 247.8us | 381.3us | 423.1ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1431.53 MB/s
zebra        ████████ 403.05 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 341.0us
zebra        █████████████████████ 247.8us
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1731.14 MB/s | 375ns | 834ns | 2.2us | 0 |
| horse | 752.48 MB/s | 792ns | 1.8us | 4.9us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1731.14 MB/s
horse        █████████████ 752.48 MB/s
```

**Latency (P50)**
```
zebra        ██████████████ 375ns
horse        ██████████████████████████████ 792ns
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 861.31 MB/s | 33.2us | 166.5us | 26.4ms | 0 |
| zebra | 830.12 MB/s | 21.9us | 27.8us | 39.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 861.31 MB/s
zebra        ████████████████████████████ 830.12 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 33.2us
zebra        ███████████████████ 21.9us
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1076.98 MB/s | 2.0us | 7.3us | 18.1us | 0 |
| horse | 931.42 MB/s | 5.1us | 11.9us | 35.2us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1076.98 MB/s
horse        █████████████████████████ 931.42 MB/s
```

**Latency (P50)**
```
zebra        ████████████ 2.0us
horse        ██████████████████████████████ 5.1us
```

## Recommendations

- **Write-heavy workloads:** zebra
- **Read-heavy workloads:** horse

---

*Generated by storage benchmark CLI*
