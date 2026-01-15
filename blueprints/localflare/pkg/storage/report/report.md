# Storage Benchmark Report

**Generated:** 2026-01-15T22:07:16+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio (won 35/51 benchmarks, 69%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio | 35 | 69% |
| 2 | rustfs | 10 | 20% |
| 3 | minio | 6 | 12% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 3.9 MB/s | +85% vs minio |
| Small Write (1KB) | liteio | 1.0 MB/s | close |
| Large Read (10MB) | liteio | 328.8 MB/s | close |
| Large Write (10MB) | rustfs | 193.5 MB/s | +10% vs liteio |
| Delete | liteio | 6.1K ops/s | 2.4x vs minio |
| Stat | liteio | 5.0K ops/s | +20% vs minio |
| List (100 objects) | liteio | 1.3K ops/s | +99% vs minio |
| Copy | liteio | 1.3 MB/s | +31% vs rustfs |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **liteio** | 182 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 327 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 2505 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **minio** | - | Best for multi-user apps |
| Memory Constrained | **liteio** | 45 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 181.9 | 299.6 | 537.4ms | 333.5ms |
| minio | 178.5 | 327.2 | 511.4ms | 305.6ms |
| rustfs | 179.9 | 299.5 | 573.7ms | 317.8ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 1062 | 3948 | 817.8us | 252.8us |
| minio | 1020 | 2131 | 941.3us | 412.6us |
| rustfs | 977 | 1935 | 914.1us | 504.7us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 5032 | 1304 | 6102 |
| minio | 4183 | 654 | 2577 |
| rustfs | 3231 | 145 | 1220 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 1.27 | 0.40 | 0.24 | 0.16 | 0.08 | 0.08 |
| minio | 0.90 | 0.39 | 0.20 | 0.08 | 0.03 | 0.08 |
| rustfs | 1.11 | 0.32 | 0.19 | 0.09 | 0.14 | 0.13 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 3.13 | 1.00 | 0.43 | 0.46 | 0.33 | 0.26 |
| minio | 2.14 | 1.45 | 0.77 | 0.48 | 0.31 | 0.56 |
| rustfs | 1.61 | 1.14 | 0.68 | 0.38 | 0.36 | 0.68 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 1.3ms | 6.5ms | 63.7ms | 631.1ms | 6.95s |
| minio | 1.5ms | 10.2ms | 99.3ms | 954.3ms | 9.75s |
| rustfs | 1.0ms | 9.3ms | 79.0ms | 733.9ms | 7.59s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 320.7us | 603.5us | 812.6us | 10.9ms | 203.3ms |
| minio | 778.5us | 954.5us | 2.7ms | 20.0ms | 191.5ms |
| rustfs | 1.1ms | 1.5ms | 7.2ms | 67.1ms | 779.6ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 44.6 MB | 0.0% |
| minio | 384.6 MB | 4.4% |
| rustfs | 721.9 MB | 0.1% |

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
| liteio | 1.26 MB/s | 629.8us | 1.5ms | 2.4ms | 0 |
| rustfs | 0.96 MB/s | 996.9us | 1.2ms | 1.3ms | 0 |
| minio | 0.81 MB/s | 1.1ms | 1.7ms | 3.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.26 MB/s
rustfs       ██████████████████████ 0.96 MB/s
minio        ███████████████████ 0.81 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 629.8us
rustfs       ██████████████████████████ 996.9us
minio        ██████████████████████████████ 1.1ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6102 ops/s | 148.2us | 184.3us | 195.5us | 0 |
| minio | 2577 ops/s | 360.5us | 454.5us | 987.0us | 0 |
| rustfs | 1220 ops/s | 811.3us | 879.3us | 927.3us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 6102 ops/s
minio        ████████████ 2577 ops/s
rustfs       █████ 1220 ops/s
```

**Latency (P50)**
```
liteio       █████ 148.2us
minio        █████████████ 360.5us
rustfs       ██████████████████████████████ 811.3us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.15 MB/s | 619.4us | 767.0us | 896.9us | 0 |
| rustfs | 0.12 MB/s | 735.0us | 802.0us | 1.3ms | 0 |
| minio | 0.10 MB/s | 922.5us | 1.2ms | 1.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.15 MB/s
rustfs       █████████████████████████ 0.12 MB/s
minio        ███████████████████ 0.10 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 619.4us
rustfs       ███████████████████████ 735.0us
minio        ██████████████████████████████ 922.5us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1082 ops/s | 785.8us | 1.3ms | 2.8ms | 0 |
| minio | 1024 ops/s | 958.0us | 1.2ms | 1.2ms | 0 |
| liteio | 449 ops/s | 781.4us | 3.2ms | 5.4ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1082 ops/s
minio        ████████████████████████████ 1024 ops/s
liteio       ████████████ 449 ops/s
```

**Latency (P50)**
```
rustfs       ████████████████████████ 785.8us
minio        ██████████████████████████████ 958.0us
liteio       ████████████████████████ 781.4us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.15 MB/s | 628.9us | 860.1us | 895.3us | 0 |
| rustfs | 0.13 MB/s | 749.6us | 823.4us | 838.5us | 0 |
| minio | 0.10 MB/s | 952.0us | 1.2ms | 1.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.15 MB/s
rustfs       █████████████████████████ 0.13 MB/s
minio        ███████████████████ 0.10 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 628.9us
rustfs       ███████████████████████ 749.6us
minio        ██████████████████████████████ 952.0us
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5128 ops/s | 195.0us | 195.0us | 195.0us | 0 |
| minio | 1652 ops/s | 605.3us | 605.3us | 605.3us | 0 |
| rustfs | 1064 ops/s | 940.1us | 940.1us | 940.1us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5128 ops/s
minio        █████████ 1652 ops/s
rustfs       ██████ 1064 ops/s
```

**Latency (P50)**
```
liteio       ██████ 195.0us
minio        ███████████████████ 605.3us
rustfs       ██████████████████████████████ 940.1us
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 645 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |
| minio | 248 ops/s | 4.0ms | 4.0ms | 4.0ms | 0 |
| rustfs | 108 ops/s | 9.3ms | 9.3ms | 9.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 645 ops/s
minio        ███████████ 248 ops/s
rustfs       █████ 108 ops/s
```

**Latency (P50)**
```
liteio       █████ 1.5ms
minio        █████████████ 4.0ms
rustfs       ██████████████████████████████ 9.3ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 49 ops/s | 20.3ms | 20.3ms | 20.3ms | 0 |
| minio | 24 ops/s | 42.5ms | 42.5ms | 42.5ms | 0 |
| rustfs | 10 ops/s | 98.5ms | 98.5ms | 98.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 49 ops/s
minio        ██████████████ 24 ops/s
rustfs       ██████ 10 ops/s
```

**Latency (P50)**
```
liteio       ██████ 20.3ms
minio        ████████████ 42.5ms
rustfs       ██████████████████████████████ 98.5ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6 ops/s | 179.1ms | 179.1ms | 179.1ms | 0 |
| minio | 3 ops/s | 384.5ms | 384.5ms | 384.5ms | 0 |
| rustfs | 1 ops/s | 938.4ms | 938.4ms | 938.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 6 ops/s
minio        █████████████ 3 ops/s
rustfs       █████ 1 ops/s
```

**Latency (P50)**
```
liteio       █████ 179.1ms
minio        ████████████ 384.5ms
rustfs       ██████████████████████████████ 938.4ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1 ops/s | 1.81s | 1.81s | 1.81s | 0 |
| minio | 0 ops/s | 3.86s | 3.86s | 3.86s | 0 |
| rustfs | 0 ops/s | 8.65s | 8.65s | 8.65s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1 ops/s
minio        ██████████████ 0 ops/s
rustfs       ██████ 0 ops/s
```

**Latency (P50)**
```
liteio       ██████ 1.81s
minio        █████████████ 3.86s
rustfs       ██████████████████████████████ 8.65s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3119 ops/s | 320.7us | 320.7us | 320.7us | 0 |
| minio | 1284 ops/s | 778.5us | 778.5us | 778.5us | 0 |
| rustfs | 923 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3119 ops/s
minio        ████████████ 1284 ops/s
rustfs       ████████ 923 ops/s
```

**Latency (P50)**
```
liteio       ████████ 320.7us
minio        █████████████████████ 778.5us
rustfs       ██████████████████████████████ 1.1ms
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1657 ops/s | 603.5us | 603.5us | 603.5us | 0 |
| minio | 1048 ops/s | 954.5us | 954.5us | 954.5us | 0 |
| rustfs | 655 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1657 ops/s
minio        ██████████████████ 1048 ops/s
rustfs       ███████████ 655 ops/s
```

**Latency (P50)**
```
liteio       ███████████ 603.5us
minio        ██████████████████ 954.5us
rustfs       ██████████████████████████████ 1.5ms
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1231 ops/s | 812.6us | 812.6us | 812.6us | 0 |
| minio | 372 ops/s | 2.7ms | 2.7ms | 2.7ms | 0 |
| rustfs | 138 ops/s | 7.2ms | 7.2ms | 7.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1231 ops/s
minio        █████████ 372 ops/s
rustfs       ███ 138 ops/s
```

**Latency (P50)**
```
liteio       ███ 812.6us
minio        ███████████ 2.7ms
rustfs       ██████████████████████████████ 7.2ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 92 ops/s | 10.9ms | 10.9ms | 10.9ms | 0 |
| minio | 50 ops/s | 20.0ms | 20.0ms | 20.0ms | 0 |
| rustfs | 15 ops/s | 67.1ms | 67.1ms | 67.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 92 ops/s
minio        ████████████████ 50 ops/s
rustfs       ████ 15 ops/s
```

**Latency (P50)**
```
liteio       ████ 10.9ms
minio        ████████ 20.0ms
rustfs       ██████████████████████████████ 67.1ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 5 ops/s | 191.5ms | 191.5ms | 191.5ms | 0 |
| liteio | 5 ops/s | 203.3ms | 203.3ms | 203.3ms | 0 |
| rustfs | 1 ops/s | 779.6ms | 779.6ms | 779.6ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 5 ops/s
liteio       ████████████████████████████ 5 ops/s
rustfs       ███████ 1 ops/s
```

**Latency (P50)**
```
minio        ███████ 191.5ms
liteio       ███████ 203.3ms
rustfs       ██████████████████████████████ 779.6ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.94 MB/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| liteio | 0.76 MB/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| minio | 0.64 MB/s | 1.5ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.94 MB/s
liteio       ████████████████████████ 0.76 MB/s
minio        ████████████████████ 0.64 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████ 1.0ms
liteio       ████████████████████████ 1.3ms
minio        ██████████████████████████████ 1.5ms
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.51 MB/s | 6.5ms | 6.5ms | 6.5ms | 0 |
| rustfs | 1.05 MB/s | 9.3ms | 9.3ms | 9.3ms | 0 |
| minio | 0.96 MB/s | 10.2ms | 10.2ms | 10.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.51 MB/s
rustfs       ████████████████████ 1.05 MB/s
minio        ███████████████████ 0.96 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 6.5ms
rustfs       ███████████████████████████ 9.3ms
minio        ██████████████████████████████ 10.2ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.53 MB/s | 63.7ms | 63.7ms | 63.7ms | 0 |
| rustfs | 1.24 MB/s | 79.0ms | 79.0ms | 79.0ms | 0 |
| minio | 0.98 MB/s | 99.3ms | 99.3ms | 99.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.53 MB/s
rustfs       ████████████████████████ 1.24 MB/s
minio        ███████████████████ 0.98 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 63.7ms
rustfs       ███████████████████████ 79.0ms
minio        ██████████████████████████████ 99.3ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.55 MB/s | 631.1ms | 631.1ms | 631.1ms | 0 |
| rustfs | 1.33 MB/s | 733.9ms | 733.9ms | 733.9ms | 0 |
| minio | 1.02 MB/s | 954.3ms | 954.3ms | 954.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.55 MB/s
rustfs       █████████████████████████ 1.33 MB/s
minio        ███████████████████ 1.02 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 631.1ms
rustfs       ███████████████████████ 733.9ms
minio        ██████████████████████████████ 954.3ms
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.41 MB/s | 6.95s | 6.95s | 6.95s | 0 |
| rustfs | 1.29 MB/s | 7.59s | 7.59s | 7.59s | 0 |
| minio | 1.00 MB/s | 9.75s | 9.75s | 9.75s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.41 MB/s
rustfs       ███████████████████████████ 1.29 MB/s
minio        █████████████████████ 1.00 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 6.95s
rustfs       ███████████████████████ 7.59s
minio        ██████████████████████████████ 9.75s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1304 ops/s | 716.0us | 811.5us | 2.2ms | 0 |
| minio | 654 ops/s | 1.5ms | 1.8ms | 1.8ms | 0 |
| rustfs | 145 ops/s | 6.7ms | 7.4ms | 11.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1304 ops/s
minio        ███████████████ 654 ops/s
rustfs       ███ 145 ops/s
```

**Latency (P50)**
```
liteio       ███ 716.0us
minio        ██████ 1.5ms
rustfs       ██████████████████████████████ 6.7ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 2.14 MB/s | 5.9ms | 14.7ms | 15.3ms | 0 |
| rustfs | 1.85 MB/s | 8.9ms | 14.5ms | 15.8ms | 0 |
| liteio | 1.60 MB/s | 9.0ms | 15.3ms | 15.9ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 2.14 MB/s
rustfs       █████████████████████████ 1.85 MB/s
liteio       ██████████████████████ 1.60 MB/s
```

**Latency (P50)**
```
minio        ███████████████████ 5.9ms
rustfs       █████████████████████████████ 8.9ms
liteio       ██████████████████████████████ 9.0ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 2.57 MB/s | 6.3ms | 7.3ms | 8.2ms | 0 |
| minio | 2.05 MB/s | 8.6ms | 9.8ms | 10.0ms | 0 |
| liteio | 1.22 MB/s | 13.6ms | 14.6ms | 14.9ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 2.57 MB/s
minio        ███████████████████████ 2.05 MB/s
liteio       ██████████████ 1.22 MB/s
```

**Latency (P50)**
```
rustfs       █████████████ 6.3ms
minio        ██████████████████ 8.6ms
liteio       ██████████████████████████████ 13.6ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.13 MB/s | 16.0ms | 19.4ms | 21.5ms | 0 |
| minio | 1.01 MB/s | 15.8ms | 21.6ms | 22.4ms | 0 |
| liteio | 0.88 MB/s | 17.8ms | 24.4ms | 25.2ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.13 MB/s
minio        ██████████████████████████ 1.01 MB/s
liteio       ███████████████████████ 0.88 MB/s
```

**Latency (P50)**
```
rustfs       ███████████████████████████ 16.0ms
minio        ██████████████████████████ 15.8ms
liteio       ██████████████████████████████ 17.8ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 155.29 MB/s | 93.9ms | 111.6ms | 111.6ms | 0 |
| rustfs | 155.20 MB/s | 93.7ms | 123.6ms | 123.6ms | 0 |
| minio | 153.32 MB/s | 94.1ms | 115.8ms | 115.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 155.29 MB/s
rustfs       █████████████████████████████ 155.20 MB/s
minio        █████████████████████████████ 153.32 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████████████ 93.9ms
rustfs       █████████████████████████████ 93.7ms
minio        ██████████████████████████████ 94.1ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 3.13 MB/s | 310.4us | 383.7us | 279.8us | 384.0us | 790.2us | 0 |
| minio | 2.14 MB/s | 456.4us | 491.8us | 452.3us | 492.2us | 520.2us | 0 |
| rustfs | 1.61 MB/s | 605.3us | 714.8us | 587.9us | 715.0us | 792.9us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.13 MB/s
minio        ████████████████████ 2.14 MB/s
rustfs       ███████████████ 1.61 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 279.8us
minio        ███████████████████████ 452.3us
rustfs       ██████████████████████████████ 587.9us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 1.45 MB/s | 675.2us | 999.1us | 644.5us | 999.4us | 1.2ms | 0 |
| rustfs | 1.14 MB/s | 855.9us | 1.1ms | 827.4us | 1.1ms | 1.3ms | 0 |
| liteio | 1.00 MB/s | 971.8us | 4.0ms | 628.2us | 4.0ms | 4.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 1.45 MB/s
rustfs       ███████████████████████ 1.14 MB/s
liteio       ████████████████████ 1.00 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████ 644.5us
rustfs       ██████████████████████████████ 827.4us
liteio       ██████████████████████ 628.2us
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rustfs | 0.36 MB/s | 2.7ms | 5.7ms | 2.3ms | 5.7ms | 6.6ms | 0 |
| liteio | 0.33 MB/s | 3.0ms | 4.0ms | 3.0ms | 4.0ms | 4.2ms | 0 |
| minio | 0.31 MB/s | 3.2ms | 4.6ms | 3.1ms | 4.6ms | 6.5ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.36 MB/s
liteio       ███████████████████████████ 0.33 MB/s
minio        █████████████████████████ 0.31 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████████ 2.3ms
liteio       █████████████████████████████ 3.0ms
minio        ██████████████████████████████ 3.1ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rustfs | 0.68 MB/s | 1.4ms | 2.6ms | 1.3ms | 2.6ms | 2.9ms | 0 |
| minio | 0.56 MB/s | 1.8ms | 2.7ms | 1.7ms | 2.7ms | 3.1ms | 0 |
| liteio | 0.26 MB/s | 3.8ms | 6.7ms | 3.6ms | 6.7ms | 7.3ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.68 MB/s
minio        ████████████████████████ 0.56 MB/s
liteio       ███████████ 0.26 MB/s
```

**Latency (P50)**
```
rustfs       ███████████ 1.3ms
minio        ██████████████ 1.7ms
liteio       ██████████████████████████████ 3.6ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.77 MB/s | 1.3ms | 1.7ms | 1.3ms | 1.7ms | 2.0ms | 0 |
| rustfs | 0.68 MB/s | 1.4ms | 1.9ms | 1.4ms | 1.9ms | 2.5ms | 0 |
| liteio | 0.43 MB/s | 2.3ms | 6.1ms | 1.2ms | 6.1ms | 6.8ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.77 MB/s
rustfs       ██████████████████████████ 0.68 MB/s
liteio       ████████████████ 0.43 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 1.3ms
rustfs       ██████████████████████████████ 1.4ms
liteio       █████████████████████████ 1.2ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.48 MB/s | 2.0ms | 3.3ms | 1.8ms | 3.3ms | 4.2ms | 0 |
| liteio | 0.46 MB/s | 2.1ms | 3.5ms | 2.0ms | 3.5ms | 3.9ms | 0 |
| rustfs | 0.38 MB/s | 2.6ms | 6.4ms | 2.1ms | 6.4ms | 7.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.48 MB/s
liteio       ████████████████████████████ 0.46 MB/s
rustfs       ███████████████████████ 0.38 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 1.8ms
liteio       █████████████████████████████ 2.0ms
rustfs       ██████████████████████████████ 2.1ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.27 MB/s | 735.8us | 987.0us | 1.1ms | 0 |
| rustfs | 1.11 MB/s | 871.3us | 960.0us | 1.0ms | 0 |
| minio | 0.90 MB/s | 1.0ms | 1.4ms | 1.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.27 MB/s
rustfs       ██████████████████████████ 1.11 MB/s
minio        █████████████████████ 0.90 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 735.8us
rustfs       █████████████████████████ 871.3us
minio        ██████████████████████████████ 1.0ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.40 MB/s | 2.1ms | 6.5ms | 7.2ms | 0 |
| minio | 0.39 MB/s | 2.3ms | 4.3ms | 5.4ms | 0 |
| rustfs | 0.32 MB/s | 2.7ms | 7.2ms | 8.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.40 MB/s
minio        █████████████████████████████ 0.39 MB/s
rustfs       ████████████████████████ 0.32 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 2.1ms
minio        ██████████████████████████ 2.3ms
rustfs       ██████████████████████████████ 2.7ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.14 MB/s | 7.2ms | 11.6ms | 13.4ms | 0 |
| liteio | 0.08 MB/s | 11.1ms | 17.4ms | 18.0ms | 0 |
| minio | 0.03 MB/s | 28.1ms | 47.4ms | 48.8ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.14 MB/s
liteio       █████████████████ 0.08 MB/s
minio        ███████ 0.03 MB/s
```

**Latency (P50)**
```
rustfs       ███████ 7.2ms
liteio       ███████████ 11.1ms
minio        ██████████████████████████████ 28.1ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.13 MB/s | 8.3ms | 10.9ms | 12.3ms | 0 |
| liteio | 0.08 MB/s | 11.5ms | 17.0ms | 18.1ms | 0 |
| minio | 0.08 MB/s | 11.7ms | 20.0ms | 22.6ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.13 MB/s
liteio       ███████████████████ 0.08 MB/s
minio        █████████████████ 0.08 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████████ 8.3ms
liteio       █████████████████████████████ 11.5ms
minio        ██████████████████████████████ 11.7ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.24 MB/s | 3.5ms | 7.6ms | 9.1ms | 0 |
| minio | 0.20 MB/s | 4.4ms | 7.3ms | 8.0ms | 0 |
| rustfs | 0.19 MB/s | 4.1ms | 12.5ms | 14.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.24 MB/s
minio        █████████████████████████ 0.20 MB/s
rustfs       ███████████████████████ 0.19 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████ 3.5ms
minio        ██████████████████████████████ 4.4ms
rustfs       ███████████████████████████ 4.1ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.16 MB/s | 5.5ms | 13.3ms | 13.9ms | 0 |
| rustfs | 0.09 MB/s | 9.8ms | 24.7ms | 25.5ms | 0 |
| minio | 0.08 MB/s | 9.5ms | 26.7ms | 27.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.16 MB/s
rustfs       ███████████████ 0.09 MB/s
minio        ███████████████ 0.08 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 5.5ms
rustfs       ██████████████████████████████ 9.8ms
minio        █████████████████████████████ 9.5ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 248.90 MB/s | 988.4us | 1.1ms | 1.2ms | 0 |
| minio | 158.04 MB/s | 1.6ms | 1.8ms | 2.1ms | 0 |
| rustfs | 104.51 MB/s | 2.2ms | 3.4ms | 4.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 248.90 MB/s
minio        ███████████████████ 158.04 MB/s
rustfs       ████████████ 104.51 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 988.4us
minio        █████████████████████ 1.6ms
rustfs       ██████████████████████████████ 2.2ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 241.93 MB/s | 1.0ms | 1.2ms | 1.3ms | 0 |
| minio | 168.73 MB/s | 1.4ms | 1.7ms | 1.8ms | 0 |
| rustfs | 116.95 MB/s | 2.1ms | 2.5ms | 2.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 241.93 MB/s
minio        ████████████████████ 168.73 MB/s
rustfs       ██████████████ 116.95 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 1.0ms
minio        ████████████████████ 1.4ms
rustfs       ██████████████████████████████ 2.1ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 241.89 MB/s | 1.0ms | 1.1ms | 1.2ms | 0 |
| minio | 158.74 MB/s | 1.4ms | 1.9ms | 3.0ms | 0 |
| rustfs | 121.22 MB/s | 2.0ms | 2.3ms | 3.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 241.89 MB/s
minio        ███████████████████ 158.74 MB/s
rustfs       ███████████████ 121.22 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.0ms
minio        █████████████████████ 1.4ms
rustfs       ██████████████████████████████ 2.0ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 327.23 MB/s | 1.1ms | 1.1ms | 305.6ms | 306.1ms | 306.1ms | 0 |
| liteio | 299.65 MB/s | 2.5ms | 2.5ms | 333.5ms | 333.6ms | 333.6ms | 0 |
| rustfs | 299.52 MB/s | 2.2ms | 2.2ms | 317.8ms | 325.0ms | 325.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 327.23 MB/s
liteio       ███████████████████████████ 299.65 MB/s
rustfs       ███████████████████████████ 299.52 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 305.6ms
liteio       ██████████████████████████████ 333.5ms
rustfs       ████████████████████████████ 317.8ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 328.79 MB/s | 415.9us | 441.2us | 30.3ms | 30.7ms | 30.7ms | 0 |
| minio | 302.44 MB/s | 1.8ms | 1.9ms | 32.1ms | 36.7ms | 36.7ms | 0 |
| rustfs | 290.50 MB/s | 4.4ms | 4.8ms | 34.1ms | 36.1ms | 36.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 328.79 MB/s
minio        ███████████████████████████ 302.44 MB/s
rustfs       ██████████████████████████ 290.50 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████ 30.3ms
minio        ████████████████████████████ 32.1ms
rustfs       ██████████████████████████████ 34.1ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 3.86 MB/s | 253.1us | 300.8us | 252.8us | 301.0us | 311.0us | 0 |
| minio | 2.08 MB/s | 469.0us | 732.5us | 412.6us | 732.8us | 1.0ms | 0 |
| rustfs | 1.89 MB/s | 516.3us | 611.9us | 504.7us | 612.8us | 647.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.86 MB/s
minio        ████████████████ 2.08 MB/s
rustfs       ██████████████ 1.89 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 252.8us
minio        ████████████████████████ 412.6us
rustfs       ██████████████████████████████ 504.7us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 309.13 MB/s | 252.8us | 323.8us | 3.2ms | 3.3ms | 3.3ms | 0 |
| minio | 248.51 MB/s | 940.0us | 1.1ms | 4.0ms | 4.3ms | 4.3ms | 0 |
| rustfs | 237.04 MB/s | 1.2ms | 1.5ms | 4.2ms | 4.5ms | 4.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 309.13 MB/s
minio        ████████████████████████ 248.51 MB/s
rustfs       ███████████████████████ 237.04 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 3.2ms
minio        ████████████████████████████ 4.0ms
rustfs       ██████████████████████████████ 4.2ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 157.42 MB/s | 233.3us | 291.6us | 395.9us | 429.8us | 442.0us | 0 |
| rustfs | 106.33 MB/s | 461.9us | 505.9us | 580.2us | 624.1us | 627.0us | 0 |
| minio | 100.26 MB/s | 435.8us | 473.4us | 622.1us | 660.7us | 674.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 157.42 MB/s
rustfs       ████████████████████ 106.33 MB/s
minio        ███████████████████ 100.26 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 395.9us
rustfs       ███████████████████████████ 580.2us
minio        ██████████████████████████████ 622.1us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5032 ops/s | 180.8us | 318.6us | 377.8us | 0 |
| minio | 4183 ops/s | 239.0us | 281.2us | 293.4us | 0 |
| rustfs | 3231 ops/s | 303.0us | 352.8us | 363.4us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5032 ops/s
minio        ████████████████████████ 4183 ops/s
rustfs       ███████████████████ 3231 ops/s
```

**Latency (P50)**
```
liteio       █████████████████ 180.8us
minio        ███████████████████████ 239.0us
rustfs       ██████████████████████████████ 303.0us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 181.89 MB/s | 537.4ms | 548.0ms | 548.0ms | 0 |
| rustfs | 179.87 MB/s | 573.7ms | 587.5ms | 587.5ms | 0 |
| minio | 178.46 MB/s | 511.4ms | 572.2ms | 572.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 181.89 MB/s
rustfs       █████████████████████████████ 179.87 MB/s
minio        █████████████████████████████ 178.46 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████████ 537.4ms
rustfs       ██████████████████████████████ 573.7ms
minio        ██████████████████████████ 511.4ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 193.46 MB/s | 51.0ms | 54.7ms | 54.7ms | 0 |
| liteio | 175.38 MB/s | 55.6ms | 60.0ms | 60.0ms | 0 |
| minio | 168.51 MB/s | 57.2ms | 67.0ms | 67.0ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 193.46 MB/s
liteio       ███████████████████████████ 175.38 MB/s
minio        ██████████████████████████ 168.51 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████████████ 51.0ms
liteio       █████████████████████████████ 55.6ms
minio        ██████████████████████████████ 57.2ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.04 MB/s | 817.8us | 1.4ms | 2.9ms | 0 |
| minio | 1.00 MB/s | 941.3us | 1.2ms | 1.6ms | 0 |
| rustfs | 0.95 MB/s | 914.1us | 1.5ms | 2.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.04 MB/s
minio        ████████████████████████████ 1.00 MB/s
rustfs       ███████████████████████████ 0.95 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████ 817.8us
minio        ██████████████████████████████ 941.3us
rustfs       █████████████████████████████ 914.1us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 163.06 MB/s | 5.9ms | 7.2ms | 7.2ms | 0 |
| minio | 128.56 MB/s | 7.4ms | 9.0ms | 9.0ms | 0 |
| liteio | 126.18 MB/s | 6.9ms | 10.6ms | 10.6ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 163.06 MB/s
minio        ███████████████████████ 128.56 MB/s
liteio       ███████████████████████ 126.18 MB/s
```

**Latency (P50)**
```
rustfs       ███████████████████████ 5.9ms
minio        ██████████████████████████████ 7.4ms
liteio       ███████████████████████████ 6.9ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 53.32 MB/s | 1.1ms | 1.7ms | 1.8ms | 0 |
| rustfs | 52.05 MB/s | 1.2ms | 1.3ms | 1.4ms | 0 |
| minio | 45.26 MB/s | 1.3ms | 1.7ms | 1.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 53.32 MB/s
rustfs       █████████████████████████████ 52.05 MB/s
minio        █████████████████████████ 45.26 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████████ 1.1ms
rustfs       ███████████████████████████ 1.2ms
minio        ██████████████████████████████ 1.3ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 44.6MiB / 7.653GiB | 44.6 MB | - | 0.0% | (no data) | 512kB / 6.68GB |
| minio | 385.3MiB / 7.653GiB | 385.3 MB | - | 4.4% | 1924.1 MB | 1.89MB / 1.92GB |
| rustfs | 721.6MiB / 7.653GiB | 721.6 MB | - | 0.1% | 1923.1 MB | 15.6MB / 1.69GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** rustfs
- **Read-heavy workloads:** liteio

---

*Generated by storage benchmark CLI*
