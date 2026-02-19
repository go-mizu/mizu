# Storage Benchmark Report

**Generated:** 2026-02-19T13:12:07+07:00

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
| Small Read (1KB) | horse | 9.6 GB/s | +17% vs zebra |
| Small Write (1KB) | zebra | 2.5 GB/s | +69% vs horse |
| Large Read (10MB) | horse | 41.7 GB/s | +32% vs zebra |
| Large Write (10MB) | horse | 888.2 MB/s | +52% vs zebra |
| Delete | zebra | 5.0M ops/s | 2.0x vs horse |
| Stat | zebra | 14.6M ops/s | +19% vs horse |
| List (100 objects) | horse | 119.2K ops/s | 5993.5x vs zebra |
| Copy | zebra | 2.1 GB/s | 3.6x vs horse |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **horse** | 888 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **horse** | 41706 MB/s | Best for streaming, CDN |
| Small File Operations | **horse** | 5662949 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **horse** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| horse | 888.2 | 41706.5 | 224.7us | 125ns |
| zebra | 585.6 | 31661.0 | 260.1us | 1.0us |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| horse | 1515291 | 9810607 | 459ns | 83ns |
| zebra | 2559666 | 8358184 | 250ns | 84ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| horse | 12293128 | 119184 | 2452982 |
| zebra | 14612906 | 20 | 4995359 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 1000.62 | 143.13 | 50.84 |
| zebra | 1248.19 | 683.29 | 196.08 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| horse | 6239.26 | 4777.23 | 4084.99 |
| zebra | 6335.37 | 3967.93 | 1965.24 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 8.1us | 9.4us | 83.9us | 577.3us |
| zebra | 2.3us | 10.0us | 49.3us | 478.8us |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| horse | 3.1us | 4.8us | 32.5us | 172.8us |
| zebra | 152.1ms | 158.5ms | 142.8ms | 143.6ms |

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
| zebra | 2115.20 MB/s | 333ns | 708ns | 3.2us | 0 |
| horse | 583.49 MB/s | 1.0us | 1.6us | 6.5us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 2115.20 MB/s
horse        ████████ 583.49 MB/s
```

**Latency (P50)**
```
zebra        █████████ 333ns
horse        ██████████████████████████████ 1.0us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 4995359 ops/s | 167ns | 333ns | 500ns | 0 |
| horse | 2452982 ops/s | 375ns | 583ns | 792ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 4995359 ops/s
horse        ██████████████ 2452982 ops/s
```

**Latency (P50)**
```
zebra        █████████████ 167ns
horse        ██████████████████████████████ 375ns
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 129.76 MB/s | 542ns | 1.1us | 1.8us | 0 |
| zebra | 129.62 MB/s | 291ns | 1.1us | 9.8us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 129.76 MB/s
zebra        █████████████████████████████ 129.62 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 542ns
zebra        ████████████████ 291ns
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1828837 ops/s | 458ns | 750ns | 1.0us | 0 |
| zebra | 1348812 ops/s | 250ns | 1.8us | 10.9us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1828837 ops/s
zebra        ██████████████████████ 1348812 ops/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 458ns
zebra        ████████████████ 250ns
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 100.04 MB/s | 750ns | 1.2us | 2.2us | 0 |
| zebra | 81.59 MB/s | 500ns | 2.3us | 17.1us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 100.04 MB/s
zebra        ████████████████████████ 81.59 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 750ns
zebra        ████████████████████ 500ns
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 119184 ops/s | 7.5us | 13.1us | 27.7us | 0 |
| zebra | 20 ops/s | 50.1ms | 50.9ms | 50.9ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 119184 ops/s
zebra        █ 20 ops/s
```

**Latency (P50)**
```
horse        █ 7.5us
zebra        ██████████████████████████████ 50.1ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 10.15 MB/s | 5.8us | 742.5us | 56.8ms | 0 |
| zebra | 7.67 MB/s | 2.5us | 206.5us | 81.0ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 10.15 MB/s
zebra        ██████████████████████ 7.67 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 5.8us
zebra        █████████████ 2.5us
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 145.89 MB/s | 6.9us | 484.1us | 1.4ms | 0 |
| zebra | 41.81 MB/s | 5.8us | 108.6us | 2.3ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 145.89 MB/s
zebra        ████████ 41.81 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 6.9us
zebra        █████████████████████████ 5.8us
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 4.96 MB/s | 4.8us | 22.0ms | 76.8ms | 0 |
| zebra | 3.15 MB/s | 1.9us | 41.4us | 224.3ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 4.96 MB/s
zebra        ███████████████████ 3.15 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 4.8us
zebra        ███████████ 1.9us
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 339.28 MB/s | 38.9ms | 68.0ms | 68.0ms | 0 |
| zebra | 256.88 MB/s | 55.3ms | 69.3ms | 69.3ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 339.28 MB/s
zebra        ██████████████████████ 256.88 MB/s
```

**Latency (P50)**
```
horse        █████████████████████ 38.9ms
zebra        ██████████████████████████████ 55.3ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 6335.37 MB/s | 125ns | 209ns | 542ns | 0 |
| horse | 6239.26 MB/s | 125ns | 250ns | 458ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 6335.37 MB/s
horse        █████████████████████████████ 6239.26 MB/s
```

**Latency (P50)**
```
zebra        ██████████████████████████████ 125ns
horse        ██████████████████████████████ 125ns
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 4777.23 MB/s | 166ns | 334ns | 916ns | 0 |
| zebra | 3967.93 MB/s | 167ns | 375ns | 2.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 4777.23 MB/s
zebra        ████████████████████████ 3967.93 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 166ns
zebra        ██████████████████████████████ 167ns
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 4084.99 MB/s | 166ns | 459ns | 1.2us | 0 |
| zebra | 1965.24 MB/s | 208ns | 708ns | 3.6us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 4084.99 MB/s
zebra        ██████████████ 1965.24 MB/s
```

**Latency (P50)**
```
horse        ███████████████████████ 166ns
zebra        ██████████████████████████████ 208ns
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 1248.19 MB/s | 708ns | 1.2us | 1.5us | 0 |
| horse | 1000.62 MB/s | 875ns | 1.4us | 1.8us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 1248.19 MB/s
horse        ████████████████████████ 1000.62 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████████ 708ns
horse        ██████████████████████████████ 875ns
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 683.29 MB/s | 833ns | 3.2us | 10.2us | 0 |
| horse | 143.13 MB/s | 1.6us | 20.5us | 56.7us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 683.29 MB/s
horse        ██████ 143.13 MB/s
```

**Latency (P50)**
```
zebra        ███████████████ 833ns
horse        ██████████████████████████████ 1.6us
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 196.08 MB/s | 1.4us | 12.7us | 78.6us | 0 |
| horse | 50.84 MB/s | 2.4us | 83.5us | 163.7us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 196.08 MB/s
horse        ███████ 50.84 MB/s
```

**Latency (P50)**
```
zebra        █████████████████ 1.4us
horse        ██████████████████████████████ 2.4us
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2252448.39 MB/s | 84ns | 167ns | 292ns | 0 |
| zebra | 1458949.32 MB/s | 84ns | 291ns | 1.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2252448.39 MB/s
zebra        ███████████████████ 1458949.32 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 84ns
zebra        ██████████████████████████████ 84ns
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2107825.69 MB/s | 84ns | 208ns | 333ns | 0 |
| zebra | 1731159.76 MB/s | 84ns | 125ns | 416ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2107825.69 MB/s
zebra        ████████████████████████ 1731159.76 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 84ns
zebra        ██████████████████████████████ 84ns
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 2156755.67 MB/s | 84ns | 167ns | 292ns | 0 |
| zebra | 1684912.34 MB/s | 84ns | 125ns | 708ns | 0 |

**Throughput**
```
horse        ██████████████████████████████ 2156755.67 MB/s
zebra        ███████████████████████ 1684912.34 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 84ns
zebra        ██████████████████████████████ 84ns
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 41706.49 MB/s | 125ns | 1.0ms | 1.6ms | 0 |
| zebra | 31661.04 MB/s | 1.0us | 1.1ms | 1.3ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 41706.49 MB/s
zebra        ██████████████████████ 31661.04 MB/s
```

**Latency (P50)**
```
horse        ███ 125ns
zebra        ██████████████████████████████ 1.0us
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 9580.67 MB/s | 83ns | 125ns | 292ns | 0 |
| zebra | 8162.29 MB/s | 84ns | 125ns | 1.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 9580.67 MB/s
zebra        █████████████████████████ 8162.29 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 9430924.40 MB/s | 84ns | 166ns | 292ns | 0 |
| zebra | 16317.54 MB/s | 76.3us | 131.5us | 197.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 9430924.40 MB/s
zebra        █ 16317.54 MB/s
```

**Latency (P50)**
```
horse        █ 84ns
zebra        ██████████████████████████████ 76.3us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 644785.77 MB/s | 83ns | 125ns | 292ns | 0 |
| zebra | 405360.31 MB/s | 84ns | 167ns | 1.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 644785.77 MB/s
zebra        ██████████████████ 405360.31 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████████ 83ns
zebra        ██████████████████████████████ 84ns
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 666667 ops/s | 1.5us | 1.5us | 1.5us | 0 |
| zebra | 275862 ops/s | 3.6us | 3.6us | 3.6us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 666667 ops/s
zebra        ████████████ 275862 ops/s
```

**Latency (P50)**
```
horse        ████████████ 1.5us
zebra        ██████████████████████████████ 3.6us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 203376 ops/s | 4.9us | 4.9us | 4.9us | 0 |
| zebra | 97163 ops/s | 10.3us | 10.3us | 10.3us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 203376 ops/s
zebra        ██████████████ 97163 ops/s
```

**Latency (P50)**
```
horse        ██████████████ 4.9us
zebra        ██████████████████████████████ 10.3us
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 36253 ops/s | 27.6us | 27.6us | 27.6us | 0 |
| zebra | 26727 ops/s | 37.4us | 37.4us | 37.4us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 36253 ops/s
zebra        ██████████████████████ 26727 ops/s
```

**Latency (P50)**
```
horse        ██████████████████████ 27.6us
zebra        ██████████████████████████████ 37.4us
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 3927 ops/s | 254.7us | 254.7us | 254.7us | 0 |
| zebra | 3120 ops/s | 320.5us | 320.5us | 320.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 3927 ops/s
zebra        ███████████████████████ 3120 ops/s
```

**Latency (P50)**
```
horse        ███████████████████████ 254.7us
zebra        ██████████████████████████████ 320.5us
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 320000 ops/s | 3.1us | 3.1us | 3.1us | 0 |
| zebra | 7 ops/s | 152.1ms | 152.1ms | 152.1ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 320000 ops/s
zebra        █ 7 ops/s
```

**Latency (P50)**
```
horse        █ 3.1us
zebra        ██████████████████████████████ 152.1ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 210526 ops/s | 4.8us | 4.8us | 4.8us | 0 |
| zebra | 6 ops/s | 158.5ms | 158.5ms | 158.5ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 210526 ops/s
zebra        █ 6 ops/s
```

**Latency (P50)**
```
horse        █ 4.8us
zebra        ██████████████████████████████ 158.5ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 30769 ops/s | 32.5us | 32.5us | 32.5us | 0 |
| zebra | 7 ops/s | 142.8ms | 142.8ms | 142.8ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 30769 ops/s
zebra        █ 7 ops/s
```

**Latency (P50)**
```
horse        █ 32.5us
zebra        ██████████████████████████████ 142.8ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 5787 ops/s | 172.8us | 172.8us | 172.8us | 0 |
| zebra | 7 ops/s | 143.6ms | 143.6ms | 143.6ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 5787 ops/s
zebra        █ 7 ops/s
```

**Latency (P50)**
```
horse        █ 172.8us
zebra        ██████████████████████████████ 143.6ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 106.57 MB/s | 2.3us | 2.3us | 2.3us | 0 |
| horse | 30.05 MB/s | 8.1us | 8.1us | 8.1us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 106.57 MB/s
horse        ████████ 30.05 MB/s
```

**Latency (P50)**
```
zebra        ████████ 2.3us
horse        ██████████████████████████████ 8.1us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 259.26 MB/s | 9.4us | 9.4us | 9.4us | 0 |
| zebra | 244.14 MB/s | 10.0us | 10.0us | 10.0us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 259.26 MB/s
zebra        ████████████████████████████ 244.14 MB/s
```

**Latency (P50)**
```
horse        ████████████████████████████ 9.4us
zebra        ██████████████████████████████ 10.0us
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 494.88 MB/s | 49.3us | 49.3us | 49.3us | 0 |
| horse | 290.93 MB/s | 83.9us | 83.9us | 83.9us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 494.88 MB/s
horse        █████████████████ 290.93 MB/s
```

**Latency (P50)**
```
zebra        █████████████████ 49.3us
horse        ██████████████████████████████ 83.9us
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 509.91 MB/s | 478.8us | 478.8us | 478.8us | 0 |
| horse | 422.88 MB/s | 577.3us | 577.3us | 577.3us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 509.91 MB/s
horse        ████████████████████████ 422.88 MB/s
```

**Latency (P50)**
```
zebra        ████████████████████████ 478.8us
horse        ██████████████████████████████ 577.3us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 14612906 ops/s | 42ns | 84ns | 208ns | 0 |
| horse | 12293128 ops/s | 83ns | 125ns | 291ns | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 14612906 ops/s
horse        █████████████████████████ 12293128 ops/s
```

**Latency (P50)**
```
zebra        ███████████████ 42ns
horse        ██████████████████████████████ 83ns
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 888.17 MB/s | 224.7us | 55.1ms | 93.7ms | 0 |
| zebra | 585.63 MB/s | 260.1us | 102.5ms | 242.6ms | 0 |

**Throughput**
```
horse        ██████████████████████████████ 888.17 MB/s
zebra        ███████████████████ 585.63 MB/s
```

**Latency (P50)**
```
horse        █████████████████████████ 224.7us
zebra        ██████████████████████████████ 260.1us
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| zebra | 2499.67 MB/s | 250ns | 834ns | 1.8us | 0 |
| horse | 1479.78 MB/s | 459ns | 917ns | 1.7us | 0 |

**Throughput**
```
zebra        ██████████████████████████████ 2499.67 MB/s
horse        █████████████████ 1479.78 MB/s
```

**Latency (P50)**
```
zebra        ████████████████ 250ns
horse        ██████████████████████████████ 459ns
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 606.68 MB/s | 19.8us | 27.2us | 60.4ms | 0 |
| zebra | 70.63 MB/s | 19.1us | 43.0us | 326.2us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 606.68 MB/s
zebra        ███ 70.63 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 19.8us
zebra        █████████████████████████████ 19.1us
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| horse | 1047.13 MB/s | 1.9us | 2.9us | 37.3us | 0 |
| zebra | 320.73 MB/s | 1.8us | 3.7us | 10.5us | 0 |

**Throughput**
```
horse        ██████████████████████████████ 1047.13 MB/s
zebra        █████████ 320.73 MB/s
```

**Latency (P50)**
```
horse        ██████████████████████████████ 1.9us
zebra        ███████████████████████████ 1.8us
```

## Recommendations

- **Write-heavy workloads:** zebra
- **Read-heavy workloads:** horse

---

*Generated by storage benchmark CLI*
