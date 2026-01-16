# Storage Benchmark Report

**Generated:** 2026-01-16T01:42:32+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio (won 33/51 benchmarks, 65%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio | 33 | 65% |
| 2 | minio | 14 | 27% |
| 3 | rustfs | 4 | 8% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 4.1 MB/s | +39% vs minio |
| Small Write (1KB) | liteio | 3.2 MB/s | 2.6x vs rustfs |
| Large Read (100MB) | minio | 256.2 MB/s | +47% vs rustfs |
| Large Write (100MB) | minio | 159.1 MB/s | +13% vs liteio |
| Delete | liteio | 3.1K ops/s | +16% vs minio |
| Stat | liteio | 4.0K ops/s | close |
| List (100 objects) | liteio | 757 ops/s | +30% vs minio |
| Copy | minio | 1.0 MB/s | +11% vs rustfs |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **minio** | 159 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 256 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 3728 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 140.5 | 139.7 | 708.6ms | 728.0ms |
| minio | 159.1 | 256.2 | 624.6ms | 394.5ms |
| rustfs | 124.9 | 173.7 | 629.7ms | 569.9ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 3302 | 4153 | 276.7us | 218.5us |
| minio | 555 | 2988 | 1.0ms | 326.5us |
| rustfs | 1289 | 2270 | 743.7us | 445.1us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 4042 | 757 | 3089 |
| minio | 3714 | 582 | 2668 |
| rustfs | 2256 | 154 | 1099 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 1.92 | 0.35 | 0.23 | 0.11 | 0.05 | 0.01 |
| minio | 0.87 | 0.25 | 0.12 | 0.03 | 0.03 | 0.02 |
| rustfs | 1.13 | 0.29 | 0.13 | 0.07 | 0.03 | 0.02 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 1.31 | 0.92 | 0.50 | 0.16 | 0.15 | 0.08 |
| minio | 2.74 | 1.01 | 0.49 | 0.28 | 0.14 | 0.07 |
| rustfs | 1.82 | 0.62 | 0.33 | 0.17 | 0.10 | 0.05 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 516.2us | 2.8ms | 27.3ms | 299.6ms | 3.43s |
| minio | 10.9ms | 19.1ms | 111.9ms | 1.01s | 10.74s |
| rustfs | 1.5ms | 11.1ms | 76.8ms | 778.4ms | 10.24s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 272.8us | 401.5us | 1.1ms | 8.1ms | 242.9ms |
| minio | 976.5us | 1.2ms | 2.1ms | 13.9ms | 175.4ms |
| rustfs | 1.2ms | 1.7ms | 7.4ms | 60.3ms | 888.8ms |

*\* indicates errors occurred*

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 100 |
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
| minio | 0.99 MB/s | 927.1us | 1.3ms | 2.0ms | 0 |
| rustfs | 0.89 MB/s | 1.0ms | 1.4ms | 2.0ms | 0 |
| liteio | 0.64 MB/s | 1.1ms | 4.0ms | 8.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.99 MB/s
rustfs       ███████████████████████████ 0.89 MB/s
liteio       ███████████████████ 0.64 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████ 927.1us
rustfs       ███████████████████████████ 1.0ms
liteio       ██████████████████████████████ 1.1ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3089 ops/s | 269.5us | 637.6us | 1.1ms | 0 |
| minio | 2668 ops/s | 352.7us | 469.7us | 793.8us | 0 |
| rustfs | 1099 ops/s | 850.9us | 1.1ms | 2.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3089 ops/s
minio        █████████████████████████ 2668 ops/s
rustfs       ██████████ 1099 ops/s
```

**Latency (P50)**
```
liteio       █████████ 269.5us
minio        ████████████ 352.7us
rustfs       ██████████████████████████████ 850.9us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.30 MB/s | 294.2us | 435.1us | 755.2us | 0 |
| rustfs | 0.13 MB/s | 710.3us | 858.4us | 1.4ms | 0 |
| minio | 0.10 MB/s | 927.3us | 1.2ms | 1.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.30 MB/s
rustfs       ████████████ 0.13 MB/s
minio        █████████ 0.10 MB/s
```

**Latency (P50)**
```
liteio       █████████ 294.2us
rustfs       ██████████████████████ 710.3us
minio        ██████████████████████████████ 927.3us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3026 ops/s | 310.0us | 425.8us | 840.1us | 0 |
| rustfs | 1168 ops/s | 819.3us | 1.0ms | 1.6ms | 0 |
| minio | 985 ops/s | 924.9us | 1.5ms | 2.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3026 ops/s
rustfs       ███████████ 1168 ops/s
minio        █████████ 985 ops/s
```

**Latency (P50)**
```
liteio       ██████████ 310.0us
rustfs       ██████████████████████████ 819.3us
minio        ██████████████████████████████ 924.9us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.29 MB/s | 300.8us | 495.1us | 970.0us | 0 |
| rustfs | 0.12 MB/s | 735.1us | 931.1us | 1.5ms | 0 |
| minio | 0.10 MB/s | 941.9us | 1.2ms | 1.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.29 MB/s
rustfs       ████████████ 0.12 MB/s
minio        ██████████ 0.10 MB/s
```

**Latency (P50)**
```
liteio       █████████ 300.8us
rustfs       ███████████████████████ 735.1us
minio        ██████████████████████████████ 941.9us
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3288 ops/s | 304.2us | 304.2us | 304.2us | 0 |
| minio | 1310 ops/s | 763.5us | 763.5us | 763.5us | 0 |
| rustfs | 1087 ops/s | 919.9us | 919.9us | 919.9us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3288 ops/s
minio        ███████████ 1310 ops/s
rustfs       █████████ 1087 ops/s
```

**Latency (P50)**
```
liteio       █████████ 304.2us
minio        ████████████████████████ 763.5us
rustfs       ██████████████████████████████ 919.9us
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 487 ops/s | 2.1ms | 2.1ms | 2.1ms | 0 |
| minio | 180 ops/s | 5.5ms | 5.5ms | 5.5ms | 0 |
| rustfs | 110 ops/s | 9.1ms | 9.1ms | 9.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 487 ops/s
minio        ███████████ 180 ops/s
rustfs       ██████ 110 ops/s
```

**Latency (P50)**
```
liteio       ██████ 2.1ms
minio        ██████████████████ 5.5ms
rustfs       ██████████████████████████████ 9.1ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 40 ops/s | 25.1ms | 25.1ms | 25.1ms | 0 |
| minio | 25 ops/s | 39.6ms | 39.6ms | 39.6ms | 0 |
| rustfs | 10 ops/s | 97.5ms | 97.5ms | 97.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 40 ops/s
minio        ██████████████████ 25 ops/s
rustfs       ███████ 10 ops/s
```

**Latency (P50)**
```
liteio       ███████ 25.1ms
minio        ████████████ 39.6ms
rustfs       ██████████████████████████████ 97.5ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3 ops/s | 290.7ms | 290.7ms | 290.7ms | 0 |
| minio | 2 ops/s | 622.8ms | 622.8ms | 622.8ms | 0 |
| rustfs | 1 ops/s | 950.1ms | 950.1ms | 950.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3 ops/s
minio        ██████████████ 2 ops/s
rustfs       █████████ 1 ops/s
```

**Latency (P50)**
```
liteio       █████████ 290.7ms
minio        ███████████████████ 622.8ms
rustfs       ██████████████████████████████ 950.1ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0 ops/s | 2.85s | 2.85s | 2.85s | 0 |
| minio | 0 ops/s | 4.01s | 4.01s | 4.01s | 0 |
| rustfs | 0 ops/s | 9.61s | 9.61s | 9.61s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0 ops/s
minio        █████████████████████ 0 ops/s
rustfs       ████████ 0 ops/s
```

**Latency (P50)**
```
liteio       ████████ 2.85s
minio        ████████████ 4.01s
rustfs       ██████████████████████████████ 9.61s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3666 ops/s | 272.8us | 272.8us | 272.8us | 0 |
| minio | 1024 ops/s | 976.5us | 976.5us | 976.5us | 0 |
| rustfs | 853 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3666 ops/s
minio        ████████ 1024 ops/s
rustfs       ██████ 853 ops/s
```

**Latency (P50)**
```
liteio       ██████ 272.8us
minio        ████████████████████████ 976.5us
rustfs       ██████████████████████████████ 1.2ms
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2491 ops/s | 401.5us | 401.5us | 401.5us | 0 |
| minio | 868 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |
| rustfs | 577 ops/s | 1.7ms | 1.7ms | 1.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2491 ops/s
minio        ██████████ 868 ops/s
rustfs       ██████ 577 ops/s
```

**Latency (P50)**
```
liteio       ██████ 401.5us
minio        ███████████████████ 1.2ms
rustfs       ██████████████████████████████ 1.7ms
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 916 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| minio | 477 ops/s | 2.1ms | 2.1ms | 2.1ms | 0 |
| rustfs | 135 ops/s | 7.4ms | 7.4ms | 7.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 916 ops/s
minio        ███████████████ 477 ops/s
rustfs       ████ 135 ops/s
```

**Latency (P50)**
```
liteio       ████ 1.1ms
minio        ████████ 2.1ms
rustfs       ██████████████████████████████ 7.4ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 123 ops/s | 8.1ms | 8.1ms | 8.1ms | 0 |
| minio | 72 ops/s | 13.9ms | 13.9ms | 13.9ms | 0 |
| rustfs | 17 ops/s | 60.3ms | 60.3ms | 60.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 123 ops/s
minio        █████████████████ 72 ops/s
rustfs       ████ 17 ops/s
```

**Latency (P50)**
```
liteio       ████ 8.1ms
minio        ██████ 13.9ms
rustfs       ██████████████████████████████ 60.3ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 6 ops/s | 175.4ms | 175.4ms | 175.4ms | 0 |
| liteio | 4 ops/s | 242.9ms | 242.9ms | 242.9ms | 0 |
| rustfs | 1 ops/s | 888.8ms | 888.8ms | 888.8ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 6 ops/s
liteio       █████████████████████ 4 ops/s
rustfs       █████ 1 ops/s
```

**Latency (P50)**
```
minio        █████ 175.4ms
liteio       ████████ 242.9ms
rustfs       ██████████████████████████████ 888.8ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.89 MB/s | 516.2us | 516.2us | 516.2us | 0 |
| rustfs | 0.65 MB/s | 1.5ms | 1.5ms | 1.5ms | 0 |
| minio | 0.09 MB/s | 10.9ms | 10.9ms | 10.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.89 MB/s
rustfs       ██████████ 0.65 MB/s
minio        █ 0.09 MB/s
```

**Latency (P50)**
```
liteio       █ 516.2us
rustfs       ████ 1.5ms
minio        ██████████████████████████████ 10.9ms
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.54 MB/s | 2.8ms | 2.8ms | 2.8ms | 0 |
| rustfs | 0.88 MB/s | 11.1ms | 11.1ms | 11.1ms | 0 |
| minio | 0.51 MB/s | 19.1ms | 19.1ms | 19.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.54 MB/s
rustfs       ███████ 0.88 MB/s
minio        ████ 0.51 MB/s
```

**Latency (P50)**
```
liteio       ████ 2.8ms
rustfs       █████████████████ 11.1ms
minio        ██████████████████████████████ 19.1ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.58 MB/s | 27.3ms | 27.3ms | 27.3ms | 0 |
| rustfs | 1.27 MB/s | 76.8ms | 76.8ms | 76.8ms | 0 |
| minio | 0.87 MB/s | 111.9ms | 111.9ms | 111.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.58 MB/s
rustfs       ██████████ 1.27 MB/s
minio        ███████ 0.87 MB/s
```

**Latency (P50)**
```
liteio       ███████ 27.3ms
rustfs       ████████████████████ 76.8ms
minio        ██████████████████████████████ 111.9ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.26 MB/s | 299.6ms | 299.6ms | 299.6ms | 0 |
| rustfs | 1.25 MB/s | 778.4ms | 778.4ms | 778.4ms | 0 |
| minio | 0.97 MB/s | 1.01s | 1.01s | 1.01s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.26 MB/s
rustfs       ███████████ 1.25 MB/s
minio        ████████ 0.97 MB/s
```

**Latency (P50)**
```
liteio       ████████ 299.6ms
rustfs       ███████████████████████ 778.4ms
minio        ██████████████████████████████ 1.01s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.85 MB/s | 3.43s | 3.43s | 3.43s | 0 |
| rustfs | 0.95 MB/s | 10.24s | 10.24s | 10.24s | 0 |
| minio | 0.91 MB/s | 10.74s | 10.74s | 10.74s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.85 MB/s
rustfs       ██████████ 0.95 MB/s
minio        █████████ 0.91 MB/s
```

**Latency (P50)**
```
liteio       █████████ 3.43s
rustfs       ████████████████████████████ 10.24s
minio        ██████████████████████████████ 10.74s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 757 ops/s | 1.3ms | 1.8ms | 2.7ms | 0 |
| minio | 582 ops/s | 1.6ms | 2.3ms | 3.2ms | 0 |
| rustfs | 154 ops/s | 6.4ms | 7.4ms | 9.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 757 ops/s
minio        ███████████████████████ 582 ops/s
rustfs       ██████ 154 ops/s
```

**Latency (P50)**
```
liteio       █████ 1.3ms
minio        ███████ 1.6ms
rustfs       ██████████████████████████████ 6.4ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.38 MB/s | 24.7ms | 62.2ms | 649.4ms | 0 |
| minio | 0.36 MB/s | 36.4ms | 90.7ms | 114.9ms | 0 |
| rustfs | 0.31 MB/s | 45.5ms | 81.9ms | 106.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.38 MB/s
minio        ████████████████████████████ 0.36 MB/s
rustfs       ████████████████████████ 0.31 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 24.7ms
minio        ███████████████████████ 36.4ms
rustfs       ██████████████████████████████ 45.5ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 0.58 MB/s | 23.4ms | 57.1ms | 107.6ms | 0 |
| liteio | 0.55 MB/s | 27.9ms | 39.4ms | 42.9ms | 0 |
| rustfs | 0.48 MB/s | 30.6ms | 51.8ms | 65.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.58 MB/s
liteio       ████████████████████████████ 0.55 MB/s
rustfs       ████████████████████████ 0.48 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████ 23.4ms
liteio       ███████████████████████████ 27.9ms
rustfs       ██████████████████████████████ 30.6ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.29 MB/s | 20.1ms | 171.6ms | 652.6ms | 0 |
| minio | 0.23 MB/s | 51.7ms | 147.9ms | 558.5ms | 0 |
| rustfs | 0.21 MB/s | 60.7ms | 113.2ms | 336.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.29 MB/s
minio        ███████████████████████ 0.23 MB/s
rustfs       █████████████████████ 0.21 MB/s
```

**Latency (P50)**
```
liteio       █████████ 20.1ms
minio        █████████████████████████ 51.7ms
rustfs       ██████████████████████████████ 60.7ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 134.97 MB/s | 106.9ms | 126.4ms | 126.4ms | 0 |
| minio | 128.10 MB/s | 110.0ms | 135.0ms | 135.0ms | 0 |
| liteio | 112.33 MB/s | 128.1ms | 139.9ms | 139.9ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 134.97 MB/s
minio        ████████████████████████████ 128.10 MB/s
liteio       ████████████████████████ 112.33 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████████████ 106.9ms
minio        █████████████████████████ 110.0ms
liteio       ██████████████████████████████ 128.1ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 2.74 MB/s | 356.3us | 397.0us | 350.1us | 397.2us | 502.3us | 0 |
| rustfs | 1.82 MB/s | 536.8us | 674.3us | 511.0us | 674.4us | 1.0ms | 0 |
| liteio | 1.31 MB/s | 742.7us | 1.4ms | 618.1us | 1.4ms | 2.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 2.74 MB/s
rustfs       ███████████████████ 1.82 MB/s
liteio       ██████████████ 1.31 MB/s
```

**Latency (P50)**
```
minio        ████████████████ 350.1us
rustfs       ████████████████████████ 511.0us
liteio       ██████████████████████████████ 618.1us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 1.01 MB/s | 963.3us | 1.5ms | 899.8us | 1.6ms | 2.3ms | 0 |
| liteio | 0.92 MB/s | 1.1ms | 1.8ms | 971.7us | 1.8ms | 2.6ms | 0 |
| rustfs | 0.62 MB/s | 1.6ms | 2.7ms | 1.4ms | 2.7ms | 3.9ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 1.01 MB/s
liteio       ███████████████████████████ 0.92 MB/s
rustfs       ██████████████████ 0.62 MB/s
```

**Latency (P50)**
```
minio        ██████████████████ 899.8us
liteio       ████████████████████ 971.7us
rustfs       ██████████████████████████████ 1.4ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.15 MB/s | 6.3ms | 11.0ms | 5.8ms | 11.0ms | 19.1ms | 0 |
| minio | 0.14 MB/s | 7.0ms | 13.3ms | 6.4ms | 13.3ms | 19.3ms | 0 |
| rustfs | 0.10 MB/s | 10.3ms | 15.4ms | 10.1ms | 15.4ms | 18.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.15 MB/s
minio        ███████████████████████████ 0.14 MB/s
rustfs       ██████████████████ 0.10 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 5.8ms
minio        ███████████████████ 6.4ms
rustfs       ██████████████████████████████ 10.1ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.08 MB/s | 12.7ms | 22.3ms | 12.5ms | 22.3ms | 26.7ms | 0 |
| minio | 0.07 MB/s | 14.2ms | 28.3ms | 13.0ms | 28.4ms | 38.3ms | 0 |
| rustfs | 0.05 MB/s | 21.4ms | 37.2ms | 19.9ms | 37.2ms | 48.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.08 MB/s
minio        ██████████████████████████ 0.07 MB/s
rustfs       █████████████████ 0.05 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 12.5ms
minio        ███████████████████ 13.0ms
rustfs       ██████████████████████████████ 19.9ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.50 MB/s | 2.0ms | 3.5ms | 1.8ms | 3.5ms | 4.9ms | 0 |
| minio | 0.49 MB/s | 2.0ms | 3.3ms | 1.9ms | 3.3ms | 4.5ms | 0 |
| rustfs | 0.33 MB/s | 2.9ms | 4.9ms | 2.7ms | 4.9ms | 6.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.50 MB/s
minio        █████████████████████████████ 0.49 MB/s
rustfs       ████████████████████ 0.33 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 1.8ms
minio        █████████████████████ 1.9ms
rustfs       ██████████████████████████████ 2.7ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.28 MB/s | 3.5ms | 6.2ms | 3.3ms | 6.2ms | 8.3ms | 0 |
| rustfs | 0.17 MB/s | 5.7ms | 9.1ms | 5.4ms | 9.1ms | 11.7ms | 0 |
| liteio | 0.16 MB/s | 6.2ms | 15.7ms | 3.9ms | 15.7ms | 35.6ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.28 MB/s
rustfs       ██████████████████ 0.17 MB/s
liteio       █████████████████ 0.16 MB/s
```

**Latency (P50)**
```
minio        ██████████████████ 3.3ms
rustfs       ██████████████████████████████ 5.4ms
liteio       █████████████████████ 3.9ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.92 MB/s | 449.8us | 782.0us | 1.5ms | 0 |
| rustfs | 1.13 MB/s | 835.7us | 1.1ms | 1.5ms | 0 |
| minio | 0.87 MB/s | 934.8us | 1.8ms | 2.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.92 MB/s
rustfs       █████████████████ 1.13 MB/s
minio        █████████████ 0.87 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 449.8us
rustfs       ██████████████████████████ 835.7us
minio        ██████████████████████████████ 934.8us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.35 MB/s | 2.2ms | 5.7ms | 15.9ms | 0 |
| rustfs | 0.29 MB/s | 3.0ms | 5.5ms | 8.5ms | 0 |
| minio | 0.25 MB/s | 3.7ms | 6.9ms | 9.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.35 MB/s
rustfs       █████████████████████████ 0.29 MB/s
minio        █████████████████████ 0.25 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 2.2ms
rustfs       ████████████████████████ 3.0ms
minio        ██████████████████████████████ 3.7ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.05 MB/s | 8.3ms | 75.8ms | 83.4ms | 0 |
| rustfs | 0.03 MB/s | 28.0ms | 41.8ms | 51.9ms | 0 |
| minio | 0.03 MB/s | 27.0ms | 64.8ms | 97.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.05 MB/s
rustfs       ██████████████████████ 0.03 MB/s
minio        █████████████████████ 0.03 MB/s
```

**Latency (P50)**
```
liteio       ████████ 8.3ms
rustfs       ██████████████████████████████ 28.0ms
minio        ████████████████████████████ 27.0ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.02 MB/s | 56.2ms | 73.4ms | 79.5ms | 0 |
| minio | 0.02 MB/s | 44.8ms | 132.3ms | 364.9ms | 0 |
| liteio | 0.01 MB/s | 20.9ms | 331.5ms | 452.3ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.02 MB/s
minio        █████████████████████████████ 0.02 MB/s
liteio       █████████████████████ 0.01 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████████████████ 56.2ms
minio        ███████████████████████ 44.8ms
liteio       ███████████ 20.9ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.23 MB/s | 3.2ms | 11.2ms | 16.8ms | 0 |
| rustfs | 0.13 MB/s | 7.3ms | 12.9ms | 17.4ms | 0 |
| minio | 0.12 MB/s | 7.7ms | 12.8ms | 15.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.23 MB/s
rustfs       ████████████████ 0.13 MB/s
minio        ███████████████ 0.12 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 3.2ms
rustfs       ████████████████████████████ 7.3ms
minio        ██████████████████████████████ 7.7ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.11 MB/s | 4.8ms | 28.6ms | 37.0ms | 0 |
| rustfs | 0.07 MB/s | 14.1ms | 23.8ms | 31.1ms | 0 |
| minio | 0.03 MB/s | 31.7ms | 79.7ms | 93.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.11 MB/s
rustfs       ██████████████████ 0.07 MB/s
minio        ██████ 0.03 MB/s
```

**Latency (P50)**
```
liteio       ████ 4.8ms
rustfs       █████████████ 14.1ms
minio        ██████████████████████████████ 31.7ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 161.02 MB/s | 1.5ms | 1.9ms | 2.4ms | 0 |
| liteio | 127.70 MB/s | 1.9ms | 2.6ms | 3.0ms | 0 |
| rustfs | 72.59 MB/s | 3.0ms | 6.0ms | 7.5ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 161.02 MB/s
liteio       ███████████████████████ 127.70 MB/s
rustfs       █████████████ 72.59 MB/s
```

**Latency (P50)**
```
minio        ██████████████ 1.5ms
liteio       ██████████████████ 1.9ms
rustfs       ██████████████████████████████ 3.0ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 161.36 MB/s | 1.5ms | 1.9ms | 2.3ms | 0 |
| liteio | 126.19 MB/s | 1.8ms | 2.9ms | 3.6ms | 0 |
| rustfs | 79.32 MB/s | 3.0ms | 4.4ms | 5.9ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 161.36 MB/s
liteio       ███████████████████████ 126.19 MB/s
rustfs       ██████████████ 79.32 MB/s
```

**Latency (P50)**
```
minio        ███████████████ 1.5ms
liteio       ██████████████████ 1.8ms
rustfs       ██████████████████████████████ 3.0ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 155.07 MB/s | 1.5ms | 2.2ms | 3.0ms | 0 |
| liteio | 131.09 MB/s | 1.8ms | 2.7ms | 3.4ms | 0 |
| rustfs | 72.56 MB/s | 2.9ms | 4.5ms | 7.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 155.07 MB/s
liteio       █████████████████████████ 131.09 MB/s
rustfs       ██████████████ 72.56 MB/s
```

**Latency (P50)**
```
minio        ███████████████ 1.5ms
liteio       ██████████████████ 1.8ms
rustfs       ██████████████████████████████ 2.9ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 256.18 MB/s | 2.5ms | 2.7ms | 394.5ms | 394.5ms | 394.5ms | 0 |
| rustfs | 173.69 MB/s | 44.5ms | 4.0ms | 569.9ms | 569.9ms | 569.9ms | 0 |
| liteio | 139.74 MB/s | 17.0ms | 20.5ms | 728.0ms | 728.0ms | 728.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 256.18 MB/s
rustfs       ████████████████████ 173.69 MB/s
liteio       ████████████████ 139.74 MB/s
```

**Latency (P50)**
```
minio        ████████████████ 394.5ms
rustfs       ███████████████████████ 569.9ms
liteio       ██████████████████████████████ 728.0ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 256.47 MB/s | 989.8us | 1.2ms | 38.0ms | 45.9ms | 48.3ms | 0 |
| rustfs | 201.52 MB/s | 11.5ms | 17.9ms | 47.9ms | 56.4ms | 60.3ms | 0 |
| liteio | 116.60 MB/s | 16.1ms | 23.7ms | 81.2ms | 106.0ms | 106.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 256.47 MB/s
rustfs       ███████████████████████ 201.52 MB/s
liteio       █████████████ 116.60 MB/s
```

**Latency (P50)**
```
minio        ██████████████ 38.0ms
rustfs       █████████████████ 47.9ms
liteio       ██████████████████████████████ 81.2ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.06 MB/s | 240.7us | 357.2us | 218.5us | 357.3us | 555.7us | 0 |
| minio | 2.92 MB/s | 334.6us | 397.7us | 326.5us | 397.8us | 465.8us | 0 |
| rustfs | 2.22 MB/s | 440.5us | 506.4us | 445.1us | 506.7us | 766.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.06 MB/s
minio        █████████████████████ 2.92 MB/s
rustfs       ████████████████ 2.22 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 218.5us
minio        ██████████████████████ 326.5us
rustfs       ██████████████████████████████ 445.1us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 227.08 MB/s | 965.2us | 1.5ms | 4.2ms | 5.4ms | 7.0ms | 0 |
| rustfs | 184.24 MB/s | 1.9ms | 2.5ms | 5.0ms | 6.8ms | 10.3ms | 0 |
| liteio | 167.84 MB/s | 522.7us | 784.0us | 5.8ms | 7.3ms | 8.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 227.08 MB/s
rustfs       ████████████████████████ 184.24 MB/s
liteio       ██████████████████████ 167.84 MB/s
```

**Latency (P50)**
```
minio        █████████████████████ 4.2ms
rustfs       █████████████████████████ 5.0ms
liteio       ██████████████████████████████ 5.8ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 115.68 MB/s | 367.9us | 461.3us | 525.2us | 618.3us | 796.5us | 0 |
| liteio | 98.76 MB/s | 329.8us | 619.8us | 586.4us | 979.7us | 1.3ms | 0 |
| rustfs | 86.04 MB/s | 600.9us | 694.1us | 708.5us | 809.6us | 1.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 115.68 MB/s
liteio       █████████████████████████ 98.76 MB/s
rustfs       ██████████████████████ 86.04 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████ 525.2us
liteio       ████████████████████████ 586.4us
rustfs       ██████████████████████████████ 708.5us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4042 ops/s | 223.8us | 362.2us | 643.5us | 0 |
| minio | 3714 ops/s | 256.5us | 328.4us | 458.8us | 0 |
| rustfs | 2256 ops/s | 391.5us | 672.9us | 1.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4042 ops/s
minio        ███████████████████████████ 3714 ops/s
rustfs       ████████████████ 2256 ops/s
```

**Latency (P50)**
```
liteio       █████████████████ 223.8us
minio        ███████████████████ 256.5us
rustfs       ██████████████████████████████ 391.5us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 159.05 MB/s | 624.6ms | 624.6ms | 624.6ms | 0 |
| liteio | 140.53 MB/s | 708.6ms | 708.6ms | 708.6ms | 0 |
| rustfs | 124.91 MB/s | 629.7ms | 629.7ms | 629.7ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 159.05 MB/s
liteio       ██████████████████████████ 140.53 MB/s
rustfs       ███████████████████████ 124.91 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 624.6ms
liteio       ██████████████████████████████ 708.6ms
rustfs       ██████████████████████████ 629.7ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 163.19 MB/s | 60.8ms | 70.7ms | 72.8ms | 0 |
| minio | 161.10 MB/s | 56.1ms | 76.6ms | 77.9ms | 0 |
| liteio | 143.15 MB/s | 68.6ms | 79.7ms | 79.7ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 163.19 MB/s
minio        █████████████████████████████ 161.10 MB/s
liteio       ██████████████████████████ 143.15 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████████████ 60.8ms
minio        ████████████████████████ 56.1ms
liteio       ██████████████████████████████ 68.6ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.22 MB/s | 276.7us | 386.6us | 1.0ms | 0 |
| rustfs | 1.26 MB/s | 743.7us | 866.0us | 1.4ms | 0 |
| minio | 0.54 MB/s | 1.0ms | 6.2ms | 11.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.22 MB/s
rustfs       ███████████ 1.26 MB/s
minio        █████ 0.54 MB/s
```

**Latency (P50)**
```
liteio       ████████ 276.7us
rustfs       ██████████████████████ 743.7us
minio        ██████████████████████████████ 1.0ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 147.59 MB/s | 6.5ms | 9.5ms | 11.4ms | 0 |
| minio | 125.27 MB/s | 7.5ms | 10.5ms | 14.7ms | 0 |
| liteio | 96.33 MB/s | 7.5ms | 20.7ms | 27.4ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 147.59 MB/s
minio        █████████████████████████ 125.27 MB/s
liteio       ███████████████████ 96.33 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████████████ 6.5ms
minio        ██████████████████████████████ 7.5ms
liteio       █████████████████████████████ 7.5ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 87.49 MB/s | 679.0us | 945.9us | 1.5ms | 0 |
| rustfs | 56.52 MB/s | 1.0ms | 1.4ms | 1.9ms | 0 |
| minio | 44.04 MB/s | 1.3ms | 1.8ms | 2.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 87.49 MB/s
rustfs       ███████████████████ 56.52 MB/s
minio        ███████████████ 44.04 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 679.0us
rustfs       ███████████████████████ 1.0ms
minio        ██████████████████████████████ 1.3ms
```

## Recommendations

- **Write-heavy workloads:** rustfs
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
