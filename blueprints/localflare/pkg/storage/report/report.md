# Storage Benchmark Report

**Generated:** 2026-01-15T22:46:56+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio (won 35/51 benchmarks, 69%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio | 35 | 69% |
| 2 | rustfs | 14 | 27% |
| 3 | minio | 2 | 4% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 3.5 MB/s | +26% vs minio |
| Small Write (1KB) | rustfs | 1.3 MB/s | close |
| Large Read (10MB) | minio | 289.4 MB/s | close |
| Large Write (10MB) | liteio | 174.0 MB/s | close |
| Delete | liteio | 4.3K ops/s | +67% vs minio |
| Stat | liteio | 5.9K ops/s | +55% vs minio |
| List (100 objects) | liteio | 1.4K ops/s | 2.4x vs minio |
| Copy | liteio | 1.1 MB/s | +10% vs rustfs |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **rustfs** | 167 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 325 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 2423 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |
| Memory Constrained | **liteio** | 103 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 155.5 | 305.0 | 648.4ms | 329.0ms |
| minio | 163.5 | 325.0 | 635.3ms | 303.1ms |
| rustfs | 166.5 | 282.0 | 617.6ms | 339.2ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 1253 | 3593 | 761.4us | 273.3us |
| minio | 968 | 2857 | 1.0ms | 346.6us |
| rustfs | 1314 | 1954 | 748.7us | 505.6us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 5938 | 1395 | 4345 |
| minio | 3821 | 573 | 2606 |
| rustfs | 3300 | 143 | 1211 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 0.98 | 0.37 | 0.24 | 0.16 | 0.10 | 0.07 |
| minio | 0.87 | 0.38 | 0.17 | 0.11 | 0.08 | 0.10 |
| rustfs | 1.10 | 0.47 | 0.23 | 0.13 | 0.15 | 0.13 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 3.83 | 1.72 | 1.24 | 0.83 | 0.52 | 1.45 |
| minio | 2.42 | 1.14 | 0.75 | 0.57 | 0.38 | 0.66 |
| rustfs | 1.62 | 1.10 | 0.68 | 0.40 | 0.47 | 0.44 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 1.4ms | 8.9ms | 80.1ms | 813.1ms | 8.43s |
| minio | 2.0ms | 9.5ms | 95.7ms | 1.01s | 10.69s |
| rustfs | 2.6ms | 8.4ms | 77.2ms | 741.1ms | 7.66s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 331.0us | 318.0us | 1.3ms | 10.0ms | 226.3ms |
| minio | 589.7us | 801.8us | 3.1ms | 15.5ms | 227.2ms |
| rustfs | 1.1ms | 1.6ms | 8.1ms | 60.4ms | 771.9ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 103.2 MB | 0.0% |
| minio | 447.3 MB | 1.7% |
| rustfs | 563.6 MB | 1.6% |

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
| liteio | 1.06 MB/s | 810.7us | 1.7ms | 2.0ms | 0 |
| rustfs | 0.97 MB/s | 1.0ms | 1.1ms | 1.5ms | 0 |
| minio | 0.92 MB/s | 981.7us | 1.7ms | 2.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.06 MB/s
rustfs       ███████████████████████████ 0.97 MB/s
minio        █████████████████████████ 0.92 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████ 810.7us
rustfs       ██████████████████████████████ 1.0ms
minio        █████████████████████████████ 981.7us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4345 ops/s | 226.7us | 276.3us | 285.5us | 0 |
| minio | 2606 ops/s | 374.4us | 453.5us | 539.6us | 0 |
| rustfs | 1211 ops/s | 819.2us | 885.8us | 925.1us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4345 ops/s
minio        █████████████████ 2606 ops/s
rustfs       ████████ 1211 ops/s
```

**Latency (P50)**
```
liteio       ████████ 226.7us
minio        █████████████ 374.4us
rustfs       ██████████████████████████████ 819.2us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.11 MB/s | 604.9us | 1.7ms | 1.8ms | 0 |
| rustfs | 0.11 MB/s | 767.8us | 1.4ms | 1.7ms | 0 |
| minio | 0.10 MB/s | 944.5us | 1.1ms | 1.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.11 MB/s
rustfs       ████████████████████████████ 0.11 MB/s
minio        █████████████████████████ 0.10 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 604.9us
rustfs       ████████████████████████ 767.8us
minio        ██████████████████████████████ 944.5us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1274 ops/s | 748.8us | 884.5us | 1.2ms | 0 |
| minio | 970 ops/s | 991.1us | 1.2ms | 1.5ms | 0 |
| liteio | 499 ops/s | 772.8us | 3.9ms | 6.5ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1274 ops/s
minio        ██████████████████████ 970 ops/s
liteio       ███████████ 499 ops/s
```

**Latency (P50)**
```
rustfs       ██████████████████████ 748.8us
minio        ██████████████████████████████ 991.1us
liteio       ███████████████████████ 772.8us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.13 MB/s | 727.0us | 834.1us | 841.1us | 0 |
| liteio | 0.11 MB/s | 634.1us | 1.6ms | 2.2ms | 0 |
| minio | 0.09 MB/s | 981.3us | 1.4ms | 1.7ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.13 MB/s
liteio       ██████████████████████████ 0.11 MB/s
minio        █████████████████████ 0.09 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████████ 727.0us
liteio       ███████████████████ 634.1us
minio        ██████████████████████████████ 981.3us
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4862 ops/s | 205.7us | 205.7us | 205.7us | 0 |
| minio | 2117 ops/s | 472.5us | 472.5us | 472.5us | 0 |
| rustfs | 1043 ops/s | 958.8us | 958.8us | 958.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4862 ops/s
minio        █████████████ 2117 ops/s
rustfs       ██████ 1043 ops/s
```

**Latency (P50)**
```
liteio       ██████ 205.7us
minio        ██████████████ 472.5us
rustfs       ██████████████████████████████ 958.8us
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 578 ops/s | 1.7ms | 1.7ms | 1.7ms | 0 |
| minio | 232 ops/s | 4.3ms | 4.3ms | 4.3ms | 0 |
| rustfs | 113 ops/s | 8.8ms | 8.8ms | 8.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 578 ops/s
minio        ████████████ 232 ops/s
rustfs       █████ 113 ops/s
```

**Latency (P50)**
```
liteio       █████ 1.7ms
minio        ██████████████ 4.3ms
rustfs       ██████████████████████████████ 8.8ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 39 ops/s | 25.8ms | 25.8ms | 25.8ms | 0 |
| minio | 19 ops/s | 52.0ms | 52.0ms | 52.0ms | 0 |
| rustfs | 11 ops/s | 88.4ms | 88.4ms | 88.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 39 ops/s
minio        ██████████████ 19 ops/s
rustfs       ████████ 11 ops/s
```

**Latency (P50)**
```
liteio       ████████ 25.8ms
minio        █████████████████ 52.0ms
rustfs       ██████████████████████████████ 88.4ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5 ops/s | 182.0ms | 182.0ms | 182.0ms | 0 |
| minio | 3 ops/s | 385.4ms | 385.4ms | 385.4ms | 0 |
| rustfs | 1 ops/s | 804.5ms | 804.5ms | 804.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5 ops/s
minio        ██████████████ 3 ops/s
rustfs       ██████ 1 ops/s
```

**Latency (P50)**
```
liteio       ██████ 182.0ms
minio        ██████████████ 385.4ms
rustfs       ██████████████████████████████ 804.5ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1 ops/s | 1.79s | 1.79s | 1.79s | 0 |
| minio | 0 ops/s | 4.60s | 4.60s | 4.60s | 0 |
| rustfs | 0 ops/s | 8.77s | 8.77s | 8.77s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1 ops/s
minio        ███████████ 0 ops/s
rustfs       ██████ 0 ops/s
```

**Latency (P50)**
```
liteio       ██████ 1.79s
minio        ███████████████ 4.60s
rustfs       ██████████████████████████████ 8.77s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3022 ops/s | 331.0us | 331.0us | 331.0us | 0 |
| minio | 1696 ops/s | 589.7us | 589.7us | 589.7us | 0 |
| rustfs | 887 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3022 ops/s
minio        ████████████████ 1696 ops/s
rustfs       ████████ 887 ops/s
```

**Latency (P50)**
```
liteio       ████████ 331.0us
minio        ███████████████ 589.7us
rustfs       ██████████████████████████████ 1.1ms
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3145 ops/s | 318.0us | 318.0us | 318.0us | 0 |
| minio | 1247 ops/s | 801.8us | 801.8us | 801.8us | 0 |
| rustfs | 628 ops/s | 1.6ms | 1.6ms | 1.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3145 ops/s
minio        ███████████ 1247 ops/s
rustfs       █████ 628 ops/s
```

**Latency (P50)**
```
liteio       █████ 318.0us
minio        ███████████████ 801.8us
rustfs       ██████████████████████████████ 1.6ms
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 775 ops/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| minio | 321 ops/s | 3.1ms | 3.1ms | 3.1ms | 0 |
| rustfs | 124 ops/s | 8.1ms | 8.1ms | 8.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 775 ops/s
minio        ████████████ 321 ops/s
rustfs       ████ 124 ops/s
```

**Latency (P50)**
```
liteio       ████ 1.3ms
minio        ███████████ 3.1ms
rustfs       ██████████████████████████████ 8.1ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 100 ops/s | 10.0ms | 10.0ms | 10.0ms | 0 |
| minio | 64 ops/s | 15.5ms | 15.5ms | 15.5ms | 0 |
| rustfs | 17 ops/s | 60.4ms | 60.4ms | 60.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 100 ops/s
minio        ███████████████████ 64 ops/s
rustfs       ████ 17 ops/s
```

**Latency (P50)**
```
liteio       ████ 10.0ms
minio        ███████ 15.5ms
rustfs       ██████████████████████████████ 60.4ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4 ops/s | 226.3ms | 226.3ms | 226.3ms | 0 |
| minio | 4 ops/s | 227.2ms | 227.2ms | 227.2ms | 0 |
| rustfs | 1 ops/s | 771.9ms | 771.9ms | 771.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4 ops/s
minio        █████████████████████████████ 4 ops/s
rustfs       ████████ 1 ops/s
```

**Latency (P50)**
```
liteio       ████████ 226.3ms
minio        ████████ 227.2ms
rustfs       ██████████████████████████████ 771.9ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.71 MB/s | 1.4ms | 1.4ms | 1.4ms | 0 |
| minio | 0.50 MB/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| rustfs | 0.37 MB/s | 2.6ms | 2.6ms | 2.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.71 MB/s
minio        ████████████████████ 0.50 MB/s
rustfs       ███████████████ 0.37 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.4ms
minio        ██████████████████████ 2.0ms
rustfs       ██████████████████████████████ 2.6ms
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.16 MB/s | 8.4ms | 8.4ms | 8.4ms | 0 |
| liteio | 1.10 MB/s | 8.9ms | 8.9ms | 8.9ms | 0 |
| minio | 1.03 MB/s | 9.5ms | 9.5ms | 9.5ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.16 MB/s
liteio       ████████████████████████████ 1.10 MB/s
minio        ██████████████████████████ 1.03 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████████████ 8.4ms
liteio       ████████████████████████████ 8.9ms
minio        ██████████████████████████████ 9.5ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.26 MB/s | 77.2ms | 77.2ms | 77.2ms | 0 |
| liteio | 1.22 MB/s | 80.1ms | 80.1ms | 80.1ms | 0 |
| minio | 1.02 MB/s | 95.7ms | 95.7ms | 95.7ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.26 MB/s
liteio       ████████████████████████████ 1.22 MB/s
minio        ████████████████████████ 1.02 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████████ 77.2ms
liteio       █████████████████████████ 80.1ms
minio        ██████████████████████████████ 95.7ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.32 MB/s | 741.1ms | 741.1ms | 741.1ms | 0 |
| liteio | 1.20 MB/s | 813.1ms | 813.1ms | 813.1ms | 0 |
| minio | 0.97 MB/s | 1.01s | 1.01s | 1.01s | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.32 MB/s
liteio       ███████████████████████████ 1.20 MB/s
minio        ██████████████████████ 0.97 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████████ 741.1ms
liteio       ████████████████████████ 813.1ms
minio        ██████████████████████████████ 1.01s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.27 MB/s | 7.66s | 7.66s | 7.66s | 0 |
| liteio | 1.16 MB/s | 8.43s | 8.43s | 8.43s | 0 |
| minio | 0.91 MB/s | 10.69s | 10.69s | 10.69s | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.27 MB/s
liteio       ███████████████████████████ 1.16 MB/s
minio        █████████████████████ 0.91 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████████ 7.66s
liteio       ███████████████████████ 8.43s
minio        ██████████████████████████████ 10.69s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1395 ops/s | 714.8us | 777.6us | 814.2us | 0 |
| minio | 573 ops/s | 1.7ms | 2.2ms | 2.8ms | 0 |
| rustfs | 143 ops/s | 6.8ms | 8.7ms | 10.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1395 ops/s
minio        ████████████ 573 ops/s
rustfs       ███ 143 ops/s
```

**Latency (P50)**
```
liteio       ███ 714.8us
minio        ███████ 1.7ms
rustfs       ██████████████████████████████ 6.8ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.25 MB/s | 7.1ms | 13.9ms | 14.4ms | 0 |
| rustfs | 1.84 MB/s | 8.1ms | 13.7ms | 14.1ms | 0 |
| minio | 1.24 MB/s | 12.1ms | 19.5ms | 19.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.25 MB/s
rustfs       ████████████████████████ 1.84 MB/s
minio        ████████████████ 1.24 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 7.1ms
rustfs       ████████████████████ 8.1ms
minio        ██████████████████████████████ 12.1ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 2.80 MB/s | 5.6ms | 6.8ms | 8.2ms | 0 |
| liteio | 1.90 MB/s | 8.5ms | 9.3ms | 9.4ms | 0 |
| minio | 1.36 MB/s | 11.6ms | 13.2ms | 13.4ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 2.80 MB/s
liteio       ████████████████████ 1.90 MB/s
minio        ██████████████ 1.36 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████ 5.6ms
liteio       ██████████████████████ 8.5ms
minio        ██████████████████████████████ 11.6ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.41 MB/s | 10.4ms | 17.9ms | 18.1ms | 0 |
| rustfs | 1.07 MB/s | 17.6ms | 20.7ms | 21.0ms | 0 |
| minio | 0.49 MB/s | 33.8ms | 40.7ms | 41.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.41 MB/s
rustfs       ██████████████████████ 1.07 MB/s
minio        ██████████ 0.49 MB/s
```

**Latency (P50)**
```
liteio       █████████ 10.4ms
rustfs       ███████████████ 17.6ms
minio        ██████████████████████████████ 33.8ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 162.89 MB/s | 95.2ms | 113.8ms | 113.8ms | 0 |
| minio | 150.56 MB/s | 94.6ms | 119.0ms | 119.0ms | 0 |
| liteio | 145.60 MB/s | 100.9ms | 117.1ms | 117.1ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 162.89 MB/s
minio        ███████████████████████████ 150.56 MB/s
liteio       ██████████████████████████ 145.60 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████████████ 95.2ms
minio        ████████████████████████████ 94.6ms
liteio       ██████████████████████████████ 100.9ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 3.83 MB/s | 254.9us | 358.8us | 232.7us | 358.9us | 410.1us | 0 |
| minio | 2.42 MB/s | 403.5us | 477.7us | 402.6us | 477.9us | 504.6us | 0 |
| rustfs | 1.62 MB/s | 601.6us | 673.1us | 594.6us | 673.4us | 697.1us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.83 MB/s
minio        ██████████████████ 2.42 MB/s
rustfs       ████████████ 1.62 MB/s
```

**Latency (P50)**
```
liteio       ███████████ 232.7us
minio        ████████████████████ 402.6us
rustfs       ██████████████████████████████ 594.6us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.72 MB/s | 567.1us | 853.0us | 534.7us | 853.2us | 947.5us | 0 |
| minio | 1.14 MB/s | 857.5us | 1.4ms | 794.1us | 1.4ms | 1.6ms | 0 |
| rustfs | 1.10 MB/s | 890.1us | 1.3ms | 823.9us | 1.3ms | 1.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.72 MB/s
minio        ███████████████████ 1.14 MB/s
rustfs       ███████████████████ 1.10 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 534.7us
minio        ████████████████████████████ 794.1us
rustfs       ██████████████████████████████ 823.9us
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.52 MB/s | 1.9ms | 3.6ms | 1.6ms | 3.6ms | 3.7ms | 0 |
| rustfs | 0.47 MB/s | 2.1ms | 5.4ms | 1.7ms | 5.4ms | 5.9ms | 0 |
| minio | 0.38 MB/s | 2.6ms | 4.3ms | 2.5ms | 4.3ms | 4.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.52 MB/s
rustfs       ███████████████████████████ 0.47 MB/s
minio        █████████████████████ 0.38 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 1.6ms
rustfs       ████████████████████ 1.7ms
minio        ██████████████████████████████ 2.5ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.45 MB/s | 672.8us | 1.6ms | 484.9us | 1.6ms | 2.5ms | 0 |
| minio | 0.66 MB/s | 1.5ms | 3.9ms | 1.0ms | 3.9ms | 5.9ms | 0 |
| rustfs | 0.44 MB/s | 2.2ms | 5.3ms | 1.8ms | 5.3ms | 5.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.45 MB/s
minio        █████████████ 0.66 MB/s
rustfs       █████████ 0.44 MB/s
```

**Latency (P50)**
```
liteio       ████████ 484.9us
minio        █████████████████ 1.0ms
rustfs       ██████████████████████████████ 1.8ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.24 MB/s | 784.9us | 1.5ms | 689.3us | 1.5ms | 1.6ms | 0 |
| minio | 0.75 MB/s | 1.3ms | 2.0ms | 1.3ms | 2.0ms | 2.4ms | 0 |
| rustfs | 0.68 MB/s | 1.4ms | 2.1ms | 1.4ms | 2.1ms | 2.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.24 MB/s
minio        ██████████████████ 0.75 MB/s
rustfs       ████████████████ 0.68 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 689.3us
minio        ███████████████████████████ 1.3ms
rustfs       ██████████████████████████████ 1.4ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.83 MB/s | 1.2ms | 1.9ms | 1.2ms | 1.9ms | 2.1ms | 0 |
| minio | 0.57 MB/s | 1.7ms | 2.7ms | 1.7ms | 2.7ms | 3.1ms | 0 |
| rustfs | 0.40 MB/s | 2.5ms | 4.5ms | 2.2ms | 4.5ms | 5.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.83 MB/s
minio        ████████████████████ 0.57 MB/s
rustfs       ██████████████ 0.40 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 1.2ms
minio        ███████████████████████ 1.7ms
rustfs       ██████████████████████████████ 2.2ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.10 MB/s | 883.0us | 962.6us | 1.1ms | 0 |
| liteio | 0.98 MB/s | 820.5us | 1.8ms | 2.2ms | 0 |
| minio | 0.87 MB/s | 1.1ms | 1.3ms | 1.5ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.10 MB/s
liteio       ██████████████████████████ 0.98 MB/s
minio        ███████████████████████ 0.87 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████████ 883.0us
liteio       ██████████████████████ 820.5us
minio        ██████████████████████████████ 1.1ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.47 MB/s | 1.9ms | 3.4ms | 4.0ms | 0 |
| minio | 0.38 MB/s | 2.4ms | 4.3ms | 4.5ms | 0 |
| liteio | 0.37 MB/s | 2.5ms | 5.0ms | 6.1ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.47 MB/s
minio        ████████████████████████ 0.38 MB/s
liteio       ███████████████████████ 0.37 MB/s
```

**Latency (P50)**
```
rustfs       ███████████████████████ 1.9ms
minio        █████████████████████████████ 2.4ms
liteio       ██████████████████████████████ 2.5ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.15 MB/s | 7.1ms | 10.7ms | 11.6ms | 0 |
| liteio | 0.10 MB/s | 10.9ms | 16.4ms | 16.6ms | 0 |
| minio | 0.08 MB/s | 12.1ms | 18.2ms | 19.3ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.15 MB/s
liteio       ███████████████████ 0.10 MB/s
minio        ████████████████ 0.08 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████ 7.1ms
liteio       ██████████████████████████ 10.9ms
minio        ██████████████████████████████ 12.1ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.13 MB/s | 7.3ms | 11.9ms | 13.7ms | 0 |
| minio | 0.10 MB/s | 10.1ms | 16.6ms | 17.9ms | 0 |
| liteio | 0.07 MB/s | 12.9ms | 24.5ms | 25.2ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.13 MB/s
minio        ██████████████████████ 0.10 MB/s
liteio       ████████████████ 0.07 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████ 7.3ms
minio        ███████████████████████ 10.1ms
liteio       ██████████████████████████████ 12.9ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.24 MB/s | 3.6ms | 6.9ms | 7.4ms | 0 |
| rustfs | 0.23 MB/s | 4.1ms | 6.5ms | 7.0ms | 0 |
| minio | 0.17 MB/s | 5.5ms | 8.6ms | 10.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.24 MB/s
rustfs       ████████████████████████████ 0.23 MB/s
minio        █████████████████████ 0.17 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 3.6ms
rustfs       ██████████████████████ 4.1ms
minio        ██████████████████████████████ 5.5ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.16 MB/s | 5.6ms | 10.1ms | 11.9ms | 0 |
| rustfs | 0.13 MB/s | 5.5ms | 12.2ms | 14.1ms | 0 |
| minio | 0.11 MB/s | 8.1ms | 19.0ms | 21.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.16 MB/s
rustfs       █████████████████████████ 0.13 MB/s
minio        ████████████████████ 0.11 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 5.6ms
rustfs       ████████████████████ 5.5ms
minio        ██████████████████████████████ 8.1ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 245.70 MB/s | 1.0ms | 1.1ms | 1.1ms | 0 |
| minio | 138.70 MB/s | 1.6ms | 2.9ms | 4.1ms | 0 |
| rustfs | 112.37 MB/s | 2.1ms | 2.7ms | 5.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 245.70 MB/s
minio        ████████████████ 138.70 MB/s
rustfs       █████████████ 112.37 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 1.0ms
minio        ██████████████████████ 1.6ms
rustfs       ██████████████████████████████ 2.1ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 249.04 MB/s | 1.0ms | 1.1ms | 1.2ms | 0 |
| minio | 164.99 MB/s | 1.4ms | 1.8ms | 1.9ms | 0 |
| rustfs | 124.57 MB/s | 2.0ms | 2.3ms | 2.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 249.04 MB/s
minio        ███████████████████ 164.99 MB/s
rustfs       ███████████████ 124.57 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.0ms
minio        █████████████████████ 1.4ms
rustfs       ██████████████████████████████ 2.0ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 237.28 MB/s | 1.0ms | 1.2ms | 1.5ms | 0 |
| minio | 135.15 MB/s | 1.7ms | 3.0ms | 3.6ms | 0 |
| rustfs | 109.51 MB/s | 2.0ms | 2.4ms | 4.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 237.28 MB/s
minio        █████████████████ 135.15 MB/s
rustfs       █████████████ 109.51 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.0ms
minio        █████████████████████████ 1.7ms
rustfs       ██████████████████████████████ 2.0ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 324.99 MB/s | 1.3ms | 1.2ms | 303.1ms | 310.4ms | 310.4ms | 0 |
| liteio | 305.02 MB/s | 2.2ms | 2.5ms | 329.0ms | 329.4ms | 329.4ms | 0 |
| rustfs | 282.05 MB/s | 2.2ms | 2.6ms | 339.2ms | 341.6ms | 341.6ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 324.99 MB/s
liteio       ████████████████████████████ 305.02 MB/s
rustfs       ██████████████████████████ 282.05 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 303.1ms
liteio       █████████████████████████████ 329.0ms
rustfs       ██████████████████████████████ 339.2ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 289.40 MB/s | 1.0ms | 1.2ms | 31.7ms | 35.8ms | 35.8ms | 0 |
| rustfs | 287.18 MB/s | 4.7ms | 5.2ms | 34.7ms | 35.3ms | 35.3ms | 0 |
| liteio | 270.42 MB/s | 759.1us | 1.1ms | 34.3ms | 44.5ms | 44.5ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 289.40 MB/s
rustfs       █████████████████████████████ 287.18 MB/s
liteio       ████████████████████████████ 270.42 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 31.7ms
rustfs       ██████████████████████████████ 34.7ms
liteio       █████████████████████████████ 34.3ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 3.51 MB/s | 278.1us | 333.4us | 273.3us | 333.7us | 377.2us | 0 |
| minio | 2.79 MB/s | 349.9us | 406.0us | 346.6us | 406.0us | 412.1us | 0 |
| rustfs | 1.91 MB/s | 511.5us | 563.3us | 505.6us | 563.6us | 596.2us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.51 MB/s
minio        ███████████████████████ 2.79 MB/s
rustfs       ████████████████ 1.91 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 273.3us
minio        ████████████████████ 346.6us
rustfs       ██████████████████████████████ 505.6us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 286.83 MB/s | 336.5us | 417.8us | 3.4ms | 4.0ms | 4.0ms | 0 |
| minio | 248.32 MB/s | 999.4us | 1.1ms | 4.0ms | 4.2ms | 4.2ms | 0 |
| rustfs | 214.51 MB/s | 1.7ms | 2.5ms | 4.6ms | 5.5ms | 5.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 286.83 MB/s
minio        █████████████████████████ 248.32 MB/s
rustfs       ██████████████████████ 214.51 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████ 3.4ms
minio        ██████████████████████████ 4.0ms
rustfs       ██████████████████████████████ 4.6ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 129.98 MB/s | 290.6us | 315.0us | 456.3us | 520.2us | 526.0us | 0 |
| minio | 99.86 MB/s | 427.8us | 471.4us | 623.3us | 681.7us | 687.0us | 0 |
| rustfs | 82.45 MB/s | 657.6us | 702.5us | 754.6us | 790.3us | 793.2us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 129.98 MB/s
minio        ███████████████████████ 99.86 MB/s
rustfs       ███████████████████ 82.45 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 456.3us
minio        ████████████████████████ 623.3us
rustfs       ██████████████████████████████ 754.6us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5938 ops/s | 157.2us | 245.0us | 255.3us | 0 |
| minio | 3821 ops/s | 256.0us | 305.8us | 323.4us | 0 |
| rustfs | 3300 ops/s | 298.9us | 325.9us | 330.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5938 ops/s
minio        ███████████████████ 3821 ops/s
rustfs       ████████████████ 3300 ops/s
```

**Latency (P50)**
```
liteio       ███████████████ 157.2us
minio        █████████████████████████ 256.0us
rustfs       ██████████████████████████████ 298.9us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 166.50 MB/s | 617.6ms | 631.8ms | 631.8ms | 0 |
| minio | 163.53 MB/s | 635.3ms | 650.3ms | 650.3ms | 0 |
| liteio | 155.52 MB/s | 648.4ms | 671.7ms | 671.7ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 166.50 MB/s
minio        █████████████████████████████ 163.53 MB/s
liteio       ████████████████████████████ 155.52 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████████████ 617.6ms
minio        █████████████████████████████ 635.3ms
liteio       ██████████████████████████████ 648.4ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 174.05 MB/s | 55.0ms | 63.3ms | 63.3ms | 0 |
| minio | 159.12 MB/s | 60.2ms | 68.9ms | 68.9ms | 0 |
| rustfs | 131.51 MB/s | 62.3ms | 67.1ms | 67.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 174.05 MB/s
minio        ███████████████████████████ 159.12 MB/s
rustfs       ██████████████████████ 131.51 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████ 55.0ms
minio        ████████████████████████████ 60.2ms
rustfs       ██████████████████████████████ 62.3ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.28 MB/s | 748.7us | 852.3us | 946.3us | 0 |
| liteio | 1.22 MB/s | 761.4us | 986.8us | 1.2ms | 0 |
| minio | 0.94 MB/s | 1.0ms | 1.2ms | 1.5ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.28 MB/s
liteio       ████████████████████████████ 1.22 MB/s
minio        ██████████████████████ 0.94 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████████ 748.7us
liteio       ██████████████████████ 761.4us
minio        ██████████████████████████████ 1.0ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 147.47 MB/s | 6.2ms | 8.9ms | 8.9ms | 0 |
| minio | 124.23 MB/s | 7.6ms | 10.0ms | 10.0ms | 0 |
| rustfs | 35.12 MB/s | 7.3ms | 11.2ms | 11.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 147.47 MB/s
minio        █████████████████████████ 124.23 MB/s
rustfs       ███████ 35.12 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████ 6.2ms
minio        ██████████████████████████████ 7.6ms
rustfs       ████████████████████████████ 7.3ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 62.54 MB/s | 988.0us | 1.1ms | 1.2ms | 0 |
| rustfs | 56.70 MB/s | 1.1ms | 1.4ms | 1.5ms | 0 |
| minio | 35.26 MB/s | 1.6ms | 2.2ms | 4.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 62.54 MB/s
rustfs       ███████████████████████████ 56.70 MB/s
minio        ████████████████ 35.26 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 988.0us
rustfs       ███████████████████ 1.1ms
minio        ██████████████████████████████ 1.6ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 103MiB / 7.653GiB | 103.0 MB | - | 0.0% | (no data) | 643kB / 2.23GB |
| minio | 447.5MiB / 7.653GiB | 447.5 MB | - | 1.7% | 1924.1 MB | 60.7MB / 1.95GB |
| rustfs | 563.4MiB / 7.653GiB | 563.4 MB | - | 1.6% | 1923.1 MB | 31.4MB / 1.86GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** liteio
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
