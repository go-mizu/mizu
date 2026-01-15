# Storage Benchmark Report

**Generated:** 2026-01-16T01:17:28+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio (won 45/51 benchmarks, 88%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio | 45 | 88% |
| 2 | minio | 4 | 8% |
| 3 | rustfs | 2 | 4% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 4.6 MB/s | +61% vs rustfs |
| Small Write (1KB) | liteio | 1.5 MB/s | +48% vs rustfs |
| Large Read (100MB) | minio | 290.9 MB/s | close |
| Large Write (100MB) | liteio | 198.3 MB/s | close |
| Delete | minio | 3.2K ops/s | close |
| Stat | liteio | 5.3K ops/s | +25% vs minio |
| List (100 objects) | liteio | 1.3K ops/s | 2.2x vs minio |
| Copy | liteio | 1.3 MB/s | +33% vs minio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **liteio** | 198 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 291 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 3089 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 198.3 | 236.5 | 499.6ms | 416.6ms |
| minio | 165.8 | 290.9 | 618.9ms | 332.8ms |
| rustfs | 184.1 | 283.5 | 544.2ms | 349.2ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 1505 | 4672 | 539.1us | 199.5us |
| minio | 766 | 2502 | 1.2ms | 377.0us |
| rustfs | 1020 | 2908 | 830.8us | 337.0us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 5292 | 1259 | 3191 |
| minio | 4219 | 581 | 3249 |
| rustfs | 3266 | 169 | 1356 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 1.92 | 0.38 | 0.46 | 0.45 | 0.54 | 0.40 |
| minio | 1.08 | 0.31 | 0.18 | 0.20 | 0.25 | 0.17 |
| rustfs | 1.24 | 0.25 | 0.16 | 0.23 | 0.24 | 0.30 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 3.36 | 1.83 | 0.49 | 0.74 | 1.83 | 1.58 |
| minio | 2.99 | 1.00 | 0.56 | 0.74 | 0.78 | 0.71 |
| rustfs | 1.84 | 0.83 | 0.63 | 0.58 | 0.71 | 0.62 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 400.2us | 2.7ms | 25.6ms | 213.0ms | 2.29s |
| minio | 1.1ms | 9.5ms | 91.6ms | 893.7ms | 11.02s |
| rustfs | 973.1us | 7.3ms | 70.4ms | 725.2ms | 7.43s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 270.9us | 349.3us | 898.4us | 5.7ms | 198.6ms |
| minio | 444.1us | 940.2us | 1.7ms | 12.1ms | 170.4ms |
| rustfs | 967.1us | 1.4ms | 6.8ms | 54.4ms | 818.8ms |

*\* indicates errors occurred*

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 20 |
| Warmup | 10 |
| Concurrency | 200 |
| Timeout | 30s |

## Drivers Tested

- **liteio** (51 benchmarks)
- **minio** (51 benchmarks)
- **rustfs** (51 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.34 MB/s | 621.4us | 1.8ms | 1.8ms | 0 |
| minio | 1.01 MB/s | 890.4us | 1.2ms | 1.2ms | 0 |
| rustfs | 0.47 MB/s | 1.7ms | 3.9ms | 3.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.34 MB/s
minio        ██████████████████████ 1.01 MB/s
rustfs       ██████████ 0.47 MB/s
```

**Latency (P50)**
```
liteio       ██████████ 621.4us
minio        ███████████████ 890.4us
rustfs       ██████████████████████████████ 1.7ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 3249 ops/s | 293.2us | 357.4us | 357.4us | 0 |
| liteio | 3191 ops/s | 255.5us | 538.4us | 538.4us | 0 |
| rustfs | 1356 ops/s | 769.5us | 856.8us | 856.8us | 0 |

**Throughput**
```
minio        ██████████████████████████████ 3249 ops/s
liteio       █████████████████████████████ 3191 ops/s
rustfs       ████████████ 1356 ops/s
```

**Latency (P50)**
```
minio        ███████████ 293.2us
liteio       █████████ 255.5us
rustfs       ██████████████████████████████ 769.5us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.30 MB/s | 315.7us | 348.8us | 348.8us | 0 |
| rustfs | 0.12 MB/s | 693.2us | 896.6us | 896.6us | 0 |
| minio | 0.09 MB/s | 964.2us | 1.2ms | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.30 MB/s
rustfs       ████████████ 0.12 MB/s
minio        ████████ 0.09 MB/s
```

**Latency (P50)**
```
liteio       █████████ 315.7us
rustfs       █████████████████████ 693.2us
minio        ██████████████████████████████ 964.2us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2430 ops/s | 394.1us | 457.0us | 457.0us | 0 |
| minio | 1067 ops/s | 933.3us | 1.0ms | 1.0ms | 0 |
| rustfs | 1034 ops/s | 711.1us | 1.6ms | 1.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2430 ops/s
minio        █████████████ 1067 ops/s
rustfs       ████████████ 1034 ops/s
```

**Latency (P50)**
```
liteio       ████████████ 394.1us
minio        ██████████████████████████████ 933.3us
rustfs       ██████████████████████ 711.1us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.28 MB/s | 327.5us | 388.7us | 388.7us | 0 |
| rustfs | 0.13 MB/s | 709.3us | 776.5us | 776.5us | 0 |
| minio | 0.10 MB/s | 941.9us | 1.0ms | 1.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.28 MB/s
rustfs       █████████████ 0.13 MB/s
minio        ██████████ 0.10 MB/s
```

**Latency (P50)**
```
liteio       ██████████ 327.5us
rustfs       ██████████████████████ 709.3us
minio        ██████████████████████████████ 941.9us
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4188 ops/s | 238.8us | 238.8us | 238.8us | 0 |
| minio | 2395 ops/s | 417.6us | 417.6us | 417.6us | 0 |
| rustfs | 1171 ops/s | 854.2us | 854.2us | 854.2us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4188 ops/s
minio        █████████████████ 2395 ops/s
rustfs       ████████ 1171 ops/s
```

**Latency (P50)**
```
liteio       ████████ 238.8us
minio        ██████████████ 417.6us
rustfs       ██████████████████████████████ 854.2us
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 456 ops/s | 2.2ms | 2.2ms | 2.2ms | 0 |
| minio | 225 ops/s | 4.5ms | 4.5ms | 4.5ms | 0 |
| rustfs | 110 ops/s | 9.1ms | 9.1ms | 9.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 456 ops/s
minio        ██████████████ 225 ops/s
rustfs       ███████ 110 ops/s
```

**Latency (P50)**
```
liteio       ███████ 2.2ms
minio        ██████████████ 4.5ms
rustfs       ██████████████████████████████ 9.1ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 52 ops/s | 19.2ms | 19.2ms | 19.2ms | 0 |
| minio | 28 ops/s | 35.9ms | 35.9ms | 35.9ms | 0 |
| rustfs | 11 ops/s | 89.4ms | 89.4ms | 89.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 52 ops/s
minio        ████████████████ 28 ops/s
rustfs       ██████ 11 ops/s
```

**Latency (P50)**
```
liteio       ██████ 19.2ms
minio        ████████████ 35.9ms
rustfs       ██████████████████████████████ 89.4ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5 ops/s | 182.5ms | 182.5ms | 182.5ms | 0 |
| minio | 3 ops/s | 356.2ms | 356.2ms | 356.2ms | 0 |
| rustfs | 1 ops/s | 859.5ms | 859.5ms | 859.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5 ops/s
minio        ███████████████ 3 ops/s
rustfs       ██████ 1 ops/s
```

**Latency (P50)**
```
liteio       ██████ 182.5ms
minio        ████████████ 356.2ms
rustfs       ██████████████████████████████ 859.5ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1 ops/s | 1.99s | 1.99s | 1.99s | 0 |
| minio | 0 ops/s | 4.33s | 4.33s | 4.33s | 0 |
| rustfs | 0 ops/s | 8.98s | 8.98s | 8.98s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1 ops/s
minio        █████████████ 0 ops/s
rustfs       ██████ 0 ops/s
```

**Latency (P50)**
```
liteio       ██████ 1.99s
minio        ██████████████ 4.33s
rustfs       ██████████████████████████████ 8.98s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3691 ops/s | 270.9us | 270.9us | 270.9us | 0 |
| minio | 2252 ops/s | 444.1us | 444.1us | 444.1us | 0 |
| rustfs | 1034 ops/s | 967.1us | 967.1us | 967.1us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3691 ops/s
minio        ██████████████████ 2252 ops/s
rustfs       ████████ 1034 ops/s
```

**Latency (P50)**
```
liteio       ████████ 270.9us
minio        █████████████ 444.1us
rustfs       ██████████████████████████████ 967.1us
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2863 ops/s | 349.3us | 349.3us | 349.3us | 0 |
| minio | 1064 ops/s | 940.2us | 940.2us | 940.2us | 0 |
| rustfs | 704 ops/s | 1.4ms | 1.4ms | 1.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2863 ops/s
minio        ███████████ 1064 ops/s
rustfs       ███████ 704 ops/s
```

**Latency (P50)**
```
liteio       ███████ 349.3us
minio        ███████████████████ 940.2us
rustfs       ██████████████████████████████ 1.4ms
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1113 ops/s | 898.4us | 898.4us | 898.4us | 0 |
| minio | 577 ops/s | 1.7ms | 1.7ms | 1.7ms | 0 |
| rustfs | 146 ops/s | 6.8ms | 6.8ms | 6.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1113 ops/s
minio        ███████████████ 577 ops/s
rustfs       ███ 146 ops/s
```

**Latency (P50)**
```
liteio       ███ 898.4us
minio        ███████ 1.7ms
rustfs       ██████████████████████████████ 6.8ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 177 ops/s | 5.7ms | 5.7ms | 5.7ms | 0 |
| minio | 82 ops/s | 12.1ms | 12.1ms | 12.1ms | 0 |
| rustfs | 18 ops/s | 54.4ms | 54.4ms | 54.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 177 ops/s
minio        █████████████ 82 ops/s
rustfs       ███ 18 ops/s
```

**Latency (P50)**
```
liteio       ███ 5.7ms
minio        ██████ 12.1ms
rustfs       ██████████████████████████████ 54.4ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 6 ops/s | 170.4ms | 170.4ms | 170.4ms | 0 |
| liteio | 5 ops/s | 198.6ms | 198.6ms | 198.6ms | 0 |
| rustfs | 1 ops/s | 818.8ms | 818.8ms | 818.8ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 6 ops/s
liteio       █████████████████████████ 5 ops/s
rustfs       ██████ 1 ops/s
```

**Latency (P50)**
```
minio        ██████ 170.4ms
liteio       ███████ 198.6ms
rustfs       ██████████████████████████████ 818.8ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.44 MB/s | 400.2us | 400.2us | 400.2us | 0 |
| rustfs | 1.00 MB/s | 973.1us | 973.1us | 973.1us | 0 |
| minio | 0.86 MB/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.44 MB/s
rustfs       ████████████ 1.00 MB/s
minio        ██████████ 0.86 MB/s
```

**Latency (P50)**
```
liteio       ██████████ 400.2us
rustfs       █████████████████████████ 973.1us
minio        ██████████████████████████████ 1.1ms
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.56 MB/s | 2.7ms | 2.7ms | 2.7ms | 0 |
| rustfs | 1.35 MB/s | 7.3ms | 7.3ms | 7.3ms | 0 |
| minio | 1.03 MB/s | 9.5ms | 9.5ms | 9.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.56 MB/s
rustfs       ███████████ 1.35 MB/s
minio        ████████ 1.03 MB/s
```

**Latency (P50)**
```
liteio       ████████ 2.7ms
rustfs       ███████████████████████ 7.3ms
minio        ██████████████████████████████ 9.5ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.81 MB/s | 25.6ms | 25.6ms | 25.6ms | 0 |
| rustfs | 1.39 MB/s | 70.4ms | 70.4ms | 70.4ms | 0 |
| minio | 1.07 MB/s | 91.6ms | 91.6ms | 91.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.81 MB/s
rustfs       ██████████ 1.39 MB/s
minio        ████████ 1.07 MB/s
```

**Latency (P50)**
```
liteio       ████████ 25.6ms
rustfs       ███████████████████████ 70.4ms
minio        ██████████████████████████████ 91.6ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.59 MB/s | 213.0ms | 213.0ms | 213.0ms | 0 |
| rustfs | 1.35 MB/s | 725.2ms | 725.2ms | 725.2ms | 0 |
| minio | 1.09 MB/s | 893.7ms | 893.7ms | 893.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.59 MB/s
rustfs       ████████ 1.35 MB/s
minio        ███████ 1.09 MB/s
```

**Latency (P50)**
```
liteio       ███████ 213.0ms
rustfs       ████████████████████████ 725.2ms
minio        ██████████████████████████████ 893.7ms
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.27 MB/s | 2.29s | 2.29s | 2.29s | 0 |
| rustfs | 1.31 MB/s | 7.43s | 7.43s | 7.43s | 0 |
| minio | 0.89 MB/s | 11.02s | 11.02s | 11.02s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.27 MB/s
rustfs       █████████ 1.31 MB/s
minio        ██████ 0.89 MB/s
```

**Latency (P50)**
```
liteio       ██████ 2.29s
rustfs       ████████████████████ 7.43s
minio        ██████████████████████████████ 11.02s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1259 ops/s | 767.2us | 936.8us | 936.8us | 0 |
| minio | 581 ops/s | 1.7ms | 1.9ms | 1.9ms | 0 |
| rustfs | 169 ops/s | 6.0ms | 6.5ms | 6.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1259 ops/s
minio        █████████████ 581 ops/s
rustfs       ████ 169 ops/s
```

**Latency (P50)**
```
liteio       ███ 767.2us
minio        ████████ 1.7ms
rustfs       ██████████████████████████████ 6.0ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 11.01 MB/s | 1.4ms | 2.0ms | 2.0ms | 0 |
| minio | 8.82 MB/s | 1.8ms | 2.3ms | 2.3ms | 0 |
| rustfs | 6.78 MB/s | 2.4ms | 2.8ms | 2.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 11.01 MB/s
minio        ████████████████████████ 8.82 MB/s
rustfs       ██████████████████ 6.78 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 1.4ms
minio        █████████████████████ 1.8ms
rustfs       ██████████████████████████████ 2.4ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 7.26 MB/s | 2.2ms | 2.3ms | 2.3ms | 0 |
| minio | 6.77 MB/s | 2.5ms | 2.6ms | 2.6ms | 0 |
| rustfs | 4.41 MB/s | 3.6ms | 3.8ms | 3.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 7.26 MB/s
minio        ███████████████████████████ 6.77 MB/s
rustfs       ██████████████████ 4.41 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 2.2ms
minio        ████████████████████ 2.5ms
rustfs       ██████████████████████████████ 3.6ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 7.52 MB/s | 2.1ms | 2.5ms | 2.5ms | 0 |
| rustfs | 5.02 MB/s | 3.3ms | 3.7ms | 3.7ms | 0 |
| minio | 4.81 MB/s | 3.6ms | 4.9ms | 4.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 7.52 MB/s
rustfs       ████████████████████ 5.02 MB/s
minio        ███████████████████ 4.81 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 2.1ms
rustfs       ███████████████████████████ 3.3ms
minio        ██████████████████████████████ 3.6ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 159.97 MB/s | 95.3ms | 100.7ms | 100.7ms | 0 |
| minio | 152.19 MB/s | 91.0ms | 101.7ms | 101.7ms | 0 |
| liteio | 133.41 MB/s | 101.1ms | 105.6ms | 105.6ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 159.97 MB/s
minio        ████████████████████████████ 152.19 MB/s
liteio       █████████████████████████ 133.41 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████████████ 95.3ms
minio        ███████████████████████████ 91.0ms
liteio       ██████████████████████████████ 101.1ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 3.36 MB/s | 290.8us | 336.2us | 281.5us | 336.2us | 336.2us | 0 |
| minio | 2.99 MB/s | 326.5us | 378.6us | 316.1us | 378.8us | 378.8us | 0 |
| rustfs | 1.84 MB/s | 530.4us | 615.3us | 503.8us | 615.5us | 615.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.36 MB/s
minio        ██████████████████████████ 2.99 MB/s
rustfs       ████████████████ 1.84 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 281.5us
minio        ██████████████████ 316.1us
rustfs       ██████████████████████████████ 503.8us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.83 MB/s | 534.6us | 851.4us | 450.2us | 851.6us | 851.6us | 0 |
| minio | 1.00 MB/s | 976.2us | 1.5ms | 809.0us | 1.5ms | 1.5ms | 0 |
| rustfs | 0.83 MB/s | 1.2ms | 1.5ms | 1.1ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.83 MB/s
minio        ████████████████ 1.00 MB/s
rustfs       █████████████ 0.83 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 450.2us
minio        █████████████████████ 809.0us
rustfs       ██████████████████████████████ 1.1ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.83 MB/s | 532.2us | 628.2us | 524.6us | 628.9us | 628.9us | 0 |
| minio | 0.78 MB/s | 1.2ms | 1.5ms | 1.2ms | 1.5ms | 1.5ms | 0 |
| rustfs | 0.71 MB/s | 1.4ms | 1.6ms | 1.4ms | 1.6ms | 1.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.83 MB/s
minio        ████████████ 0.78 MB/s
rustfs       ███████████ 0.71 MB/s
```

**Latency (P50)**
```
liteio       ███████████ 524.6us
minio        ██████████████████████████ 1.2ms
rustfs       ██████████████████████████████ 1.4ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.58 MB/s | 616.4us | 978.8us | 559.5us | 978.9us | 978.9us | 0 |
| minio | 0.71 MB/s | 1.4ms | 1.6ms | 1.4ms | 1.6ms | 1.6ms | 0 |
| rustfs | 0.62 MB/s | 1.6ms | 1.9ms | 1.6ms | 1.9ms | 1.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.58 MB/s
minio        █████████████ 0.71 MB/s
rustfs       ███████████ 0.62 MB/s
```

**Latency (P50)**
```
liteio       ██████████ 559.5us
minio        ██████████████████████████ 1.4ms
rustfs       ██████████████████████████████ 1.6ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rustfs | 0.63 MB/s | 1.5ms | 1.9ms | 1.6ms | 1.9ms | 1.9ms | 0 |
| minio | 0.56 MB/s | 1.7ms | 2.0ms | 1.8ms | 2.0ms | 2.0ms | 0 |
| liteio | 0.49 MB/s | 2.0ms | 2.4ms | 2.0ms | 2.4ms | 2.4ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.63 MB/s
minio        ██████████████████████████ 0.56 MB/s
liteio       ███████████████████████ 0.49 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████████ 1.6ms
minio        ████████████████████████████ 1.8ms
liteio       ██████████████████████████████ 2.0ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.74 MB/s | 1.3ms | 1.6ms | 1.3ms | 1.6ms | 1.6ms | 0 |
| minio | 0.74 MB/s | 1.3ms | 1.5ms | 1.3ms | 1.5ms | 1.5ms | 0 |
| rustfs | 0.58 MB/s | 1.7ms | 2.1ms | 1.7ms | 2.1ms | 2.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.74 MB/s
minio        █████████████████████████████ 0.74 MB/s
rustfs       ███████████████████████ 0.58 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████ 1.3ms
minio        ██████████████████████ 1.3ms
rustfs       ██████████████████████████████ 1.7ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.92 MB/s | 479.7us | 660.4us | 660.4us | 0 |
| rustfs | 1.24 MB/s | 767.2us | 840.5us | 840.5us | 0 |
| minio | 1.08 MB/s | 885.0us | 1.0ms | 1.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.92 MB/s
rustfs       ███████████████████ 1.24 MB/s
minio        ████████████████ 1.08 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 479.7us
rustfs       ██████████████████████████ 767.2us
minio        ██████████████████████████████ 885.0us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.38 MB/s | 2.5ms | 4.4ms | 4.4ms | 0 |
| minio | 0.31 MB/s | 2.9ms | 4.9ms | 4.9ms | 0 |
| rustfs | 0.25 MB/s | 3.5ms | 6.8ms | 6.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.38 MB/s
minio        ████████████████████████ 0.31 MB/s
rustfs       ███████████████████ 0.25 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 2.5ms
minio        ████████████████████████ 2.9ms
rustfs       ██████████████████████████████ 3.5ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.54 MB/s | 1.8ms | 2.1ms | 2.1ms | 0 |
| minio | 0.25 MB/s | 3.7ms | 5.3ms | 5.3ms | 0 |
| rustfs | 0.24 MB/s | 4.3ms | 5.0ms | 5.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.54 MB/s
minio        █████████████ 0.25 MB/s
rustfs       █████████████ 0.24 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 1.8ms
minio        █████████████████████████ 3.7ms
rustfs       ██████████████████████████████ 4.3ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.40 MB/s | 2.3ms | 3.2ms | 3.2ms | 0 |
| rustfs | 0.30 MB/s | 3.0ms | 3.9ms | 3.9ms | 0 |
| minio | 0.17 MB/s | 5.9ms | 6.7ms | 6.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.40 MB/s
rustfs       ██████████████████████ 0.30 MB/s
minio        ████████████ 0.17 MB/s
```

**Latency (P50)**
```
liteio       ███████████ 2.3ms
rustfs       ███████████████ 3.0ms
minio        ██████████████████████████████ 5.9ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.46 MB/s | 1.6ms | 3.3ms | 3.3ms | 0 |
| minio | 0.18 MB/s | 5.4ms | 6.5ms | 6.5ms | 0 |
| rustfs | 0.16 MB/s | 6.5ms | 7.3ms | 7.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.46 MB/s
minio        ███████████ 0.18 MB/s
rustfs       ██████████ 0.16 MB/s
```

**Latency (P50)**
```
liteio       ███████ 1.6ms
minio        ████████████████████████ 5.4ms
rustfs       ██████████████████████████████ 6.5ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.45 MB/s | 2.0ms | 2.7ms | 2.7ms | 0 |
| rustfs | 0.23 MB/s | 4.0ms | 5.2ms | 5.2ms | 0 |
| minio | 0.20 MB/s | 4.7ms | 6.7ms | 6.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.45 MB/s
rustfs       ███████████████ 0.23 MB/s
minio        █████████████ 0.20 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 2.0ms
rustfs       █████████████████████████ 4.0ms
minio        ██████████████████████████████ 4.7ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 235.93 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |
| minio | 147.02 MB/s | 1.5ms | 1.8ms | 1.8ms | 0 |
| rustfs | 113.48 MB/s | 2.0ms | 3.0ms | 3.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 235.93 MB/s
minio        ██████████████████ 147.02 MB/s
rustfs       ██████████████ 113.48 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.0ms
minio        ██████████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.0ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 218.34 MB/s | 1.1ms | 1.5ms | 1.5ms | 0 |
| minio | 163.78 MB/s | 1.5ms | 1.7ms | 1.7ms | 0 |
| rustfs | 122.30 MB/s | 2.0ms | 2.4ms | 2.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 218.34 MB/s
minio        ██████████████████████ 163.78 MB/s
rustfs       ████████████████ 122.30 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.1ms
minio        ██████████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.0ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 181.84 MB/s | 1.1ms | 2.4ms | 2.4ms | 0 |
| minio | 147.02 MB/s | 1.6ms | 2.1ms | 2.1ms | 0 |
| rustfs | 111.95 MB/s | 2.1ms | 2.8ms | 2.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 181.84 MB/s
minio        ████████████████████████ 147.02 MB/s
rustfs       ██████████████████ 111.95 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 1.1ms
minio        ███████████████████████ 1.6ms
rustfs       ██████████████████████████████ 2.1ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 290.90 MB/s | 1.3ms | 1.1ms | 332.8ms | 346.0ms | 346.0ms | 0 |
| rustfs | 283.48 MB/s | 2.2ms | 2.2ms | 349.2ms | 360.8ms | 360.8ms | 0 |
| liteio | 236.47 MB/s | 4.2ms | 4.6ms | 416.6ms | 426.5ms | 426.5ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 290.90 MB/s
rustfs       █████████████████████████████ 283.48 MB/s
liteio       ████████████████████████ 236.47 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████ 332.8ms
rustfs       █████████████████████████ 349.2ms
liteio       ██████████████████████████████ 416.6ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 279.42 MB/s | 1.4ms | 1.7ms | 35.5ms | 38.0ms | 38.0ms | 0 |
| liteio | 252.86 MB/s | 2.3ms | 3.0ms | 38.5ms | 42.8ms | 42.8ms | 0 |
| rustfs | 247.55 MB/s | 6.8ms | 7.9ms | 39.5ms | 43.6ms | 43.6ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 279.42 MB/s
liteio       ███████████████████████████ 252.86 MB/s
rustfs       ██████████████████████████ 247.55 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 35.5ms
liteio       █████████████████████████████ 38.5ms
rustfs       ██████████████████████████████ 39.5ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.56 MB/s | 213.9us | 275.8us | 199.5us | 275.9us | 275.9us | 0 |
| rustfs | 2.84 MB/s | 343.8us | 399.1us | 337.0us | 399.1us | 399.1us | 0 |
| minio | 2.44 MB/s | 399.6us | 449.2us | 377.0us | 449.3us | 449.3us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.56 MB/s
rustfs       ██████████████████ 2.84 MB/s
minio        ████████████████ 2.44 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 199.5us
rustfs       ██████████████████████████ 337.0us
minio        ██████████████████████████████ 377.0us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 253.84 MB/s | 505.0us | 1.0ms | 3.8ms | 5.0ms | 5.0ms | 0 |
| minio | 218.78 MB/s | 1.4ms | 1.6ms | 4.2ms | 4.7ms | 4.7ms | 0 |
| rustfs | 203.55 MB/s | 1.9ms | 2.1ms | 4.8ms | 5.3ms | 5.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 253.84 MB/s
minio        █████████████████████████ 218.78 MB/s
rustfs       ████████████████████████ 203.55 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 3.8ms
minio        ██████████████████████████ 4.2ms
rustfs       ██████████████████████████████ 4.8ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 138.58 MB/s | 291.5us | 436.6us | 432.1us | 552.0us | 621.5us | 0 |
| rustfs | 84.24 MB/s | 636.0us | 770.6us | 727.8us | 849.2us | 929.8us | 0 |
| minio | 72.46 MB/s | 639.7us | 1.8ms | 644.2us | 1.9ms | 2.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 138.58 MB/s
rustfs       ██████████████████ 84.24 MB/s
minio        ███████████████ 72.46 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 432.1us
rustfs       ██████████████████████████████ 727.8us
minio        ██████████████████████████ 644.2us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5292 ops/s | 166.1us | 278.3us | 278.3us | 0 |
| minio | 4219 ops/s | 238.8us | 261.4us | 261.4us | 0 |
| rustfs | 3266 ops/s | 308.6us | 343.9us | 343.9us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5292 ops/s
minio        ███████████████████████ 4219 ops/s
rustfs       ██████████████████ 3266 ops/s
```

**Latency (P50)**
```
liteio       ████████████████ 166.1us
minio        ███████████████████████ 238.8us
rustfs       ██████████████████████████████ 308.6us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 198.30 MB/s | 499.6ms | 510.7ms | 510.7ms | 0 |
| rustfs | 184.13 MB/s | 544.2ms | 550.7ms | 550.7ms | 0 |
| minio | 165.84 MB/s | 618.9ms | 620.9ms | 620.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 198.30 MB/s
rustfs       ███████████████████████████ 184.13 MB/s
minio        █████████████████████████ 165.84 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████ 499.6ms
rustfs       ██████████████████████████ 544.2ms
minio        ██████████████████████████████ 618.9ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 197.34 MB/s | 49.9ms | 54.2ms | 54.2ms | 0 |
| minio | 158.62 MB/s | 59.8ms | 72.2ms | 72.2ms | 0 |
| rustfs | 153.79 MB/s | 62.8ms | 79.3ms | 79.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 197.34 MB/s
minio        ████████████████████████ 158.62 MB/s
rustfs       ███████████████████████ 153.79 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 49.9ms
minio        ████████████████████████████ 59.8ms
rustfs       ██████████████████████████████ 62.8ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.47 MB/s | 539.1us | 1.0ms | 1.0ms | 0 |
| rustfs | 1.00 MB/s | 830.8us | 1.8ms | 1.8ms | 0 |
| minio | 0.75 MB/s | 1.2ms | 1.6ms | 1.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.47 MB/s
rustfs       ████████████████████ 1.00 MB/s
minio        ███████████████ 0.75 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 539.1us
rustfs       █████████████████████ 830.8us
minio        ██████████████████████████████ 1.2ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 176.61 MB/s | 5.3ms | 8.8ms | 8.8ms | 0 |
| rustfs | 144.34 MB/s | 6.0ms | 10.7ms | 10.7ms | 0 |
| minio | 139.72 MB/s | 6.9ms | 7.6ms | 7.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 176.61 MB/s
rustfs       ████████████████████████ 144.34 MB/s
minio        ███████████████████████ 139.72 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████ 5.3ms
rustfs       ██████████████████████████ 6.0ms
minio        ██████████████████████████████ 6.9ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 109.58 MB/s | 552.8us | 703.8us | 795.4us | 0 |
| rustfs | 43.81 MB/s | 1.2ms | 2.7ms | 3.3ms | 0 |
| minio | 38.86 MB/s | 1.5ms | 2.2ms | 2.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 109.58 MB/s
rustfs       ███████████ 43.81 MB/s
minio        ██████████ 38.86 MB/s
```

**Latency (P50)**
```
liteio       ███████████ 552.8us
rustfs       ███████████████████████ 1.2ms
minio        ██████████████████████████████ 1.5ms
```

## Recommendations

- **Write-heavy workloads:** liteio
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
