# Storage Benchmark Report

**Generated:** 2026-01-16T01:11:15+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio (won 43/51 benchmarks, 84%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio | 43 | 84% |
| 2 | rustfs | 6 | 12% |
| 3 | minio | 2 | 4% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 4.0 MB/s | 2.1x vs rustfs |
| Small Write (1KB) | rustfs | 1.1 MB/s | +11% vs liteio |
| Large Read (100MB) | rustfs | 330.7 MB/s | close |
| Large Write (100MB) | liteio | 201.8 MB/s | +20% vs rustfs |
| Delete | liteio | 5.5K ops/s | 2.9x vs minio |
| Stat | liteio | 5.1K ops/s | +41% vs minio |
| List (100 objects) | liteio | 1.2K ops/s | 2.0x vs minio |
| Copy | liteio | 2.0 MB/s | 2.3x vs rustfs |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **liteio** | 202 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **rustfs** | 331 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 2516 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |
| Memory Constrained | **liteio** | 74 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 201.8 | 305.8 | 489.4ms | 325.3ms |
| minio | 157.7 | 326.2 | 641.7ms | 298.0ms |
| rustfs | 168.3 | 330.7 | 604.9ms | 302.3ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 974 | 4059 | 1.0ms | 242.8us |
| minio | 949 | 1751 | 1.0ms | 562.5us |
| rustfs | 1079 | 1965 | 864.0us | 530.5us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 5134 | 1241 | 5546 |
| minio | 3637 | 618 | 1945 |
| rustfs | 3278 | 132 | 1043 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 2.71 | 0.83 | 0.37 | 0.15 | 0.39 | 0.11 |
| minio | 0.77 | 0.37 | 0.20 | 0.12 | 0.08 | 0.09 |
| rustfs | 0.68 | 0.49 | 0.23 | 0.17 | 0.08 | 0.07 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 4.48 | 1.74 | 0.85 | 0.52 | 0.10 | 0.40 |
| minio | 1.42 | 0.82 | 0.75 | 0.40 | 0.47 | 0.11 |
| rustfs | 1.06 | 1.08 | 0.54 | 0.36 | 0.59 | 0.40 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 684.9us | 2.2ms | 18.9ms | 189.4ms | 1.98s |
| minio | 1.8ms | 10.9ms | 96.1ms | 1.13s | 9.88s |
| rustfs | 1.4ms | 8.6ms | 116.2ms | 866.6ms | 8.53s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 379.9us | 298.0us | 806.9us | 4.9ms | 176.9ms |
| minio | 963.0us | 708.9us | 3.0ms | 14.0ms | 195.7ms |
| rustfs | 1.3ms | 1.8ms | 8.1ms | 66.9ms | 837.5ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 74.0 MB | 1.7% |
| minio | 382.7 MB | 0.1% |
| rustfs | 777.6 MB | 0.1% |

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
| liteio | 2.03 MB/s | 361.1us | 733.8us | 1.5ms | 0 |
| rustfs | 0.88 MB/s | 1.1ms | 1.3ms | 1.6ms | 0 |
| minio | 0.87 MB/s | 1.1ms | 1.3ms | 2.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.03 MB/s
rustfs       ████████████ 0.88 MB/s
minio        ████████████ 0.87 MB/s
```

**Latency (P50)**
```
liteio       █████████ 361.1us
rustfs       ██████████████████████████████ 1.1ms
minio        █████████████████████████████ 1.1ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5546 ops/s | 163.7us | 232.2us | 280.1us | 0 |
| minio | 1945 ops/s | 514.0us | 584.9us | 630.4us | 0 |
| rustfs | 1043 ops/s | 940.9us | 1.1ms | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5546 ops/s
minio        ██████████ 1945 ops/s
rustfs       █████ 1043 ops/s
```

**Latency (P50)**
```
liteio       █████ 163.7us
minio        ████████████████ 514.0us
rustfs       ██████████████████████████████ 940.9us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.50 MB/s | 186.5us | 232.8us | 252.1us | 0 |
| rustfs | 0.12 MB/s | 716.0us | 993.3us | 1.5ms | 0 |
| minio | 0.09 MB/s | 989.8us | 1.2ms | 1.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.50 MB/s
rustfs       ███████ 0.12 MB/s
minio        █████ 0.09 MB/s
```

**Latency (P50)**
```
liteio       █████ 186.5us
rustfs       █████████████████████ 716.0us
minio        ██████████████████████████████ 989.8us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2860 ops/s | 331.1us | 439.8us | 509.3us | 0 |
| rustfs | 1135 ops/s | 869.3us | 1.0ms | 1.1ms | 0 |
| minio | 970 ops/s | 1.0ms | 1.2ms | 1.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2860 ops/s
rustfs       ███████████ 1135 ops/s
minio        ██████████ 970 ops/s
```

**Latency (P50)**
```
liteio       █████████ 331.1us
rustfs       █████████████████████████ 869.3us
minio        ██████████████████████████████ 1.0ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.33 MB/s | 293.0us | 336.5us | 337.8us | 0 |
| rustfs | 0.11 MB/s | 868.4us | 952.3us | 956.5us | 0 |
| minio | 0.09 MB/s | 1.0ms | 1.2ms | 1.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.33 MB/s
rustfs       █████████ 0.11 MB/s
minio        ███████ 0.09 MB/s
```

**Latency (P50)**
```
liteio       ████████ 293.0us
rustfs       ████████████████████████ 868.4us
minio        ██████████████████████████████ 1.0ms
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5171 ops/s | 193.4us | 193.4us | 193.4us | 0 |
| minio | 2373 ops/s | 421.3us | 421.3us | 421.3us | 0 |
| rustfs | 844 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5171 ops/s
minio        █████████████ 2373 ops/s
rustfs       ████ 844 ops/s
```

**Latency (P50)**
```
liteio       ████ 193.4us
minio        ██████████ 421.3us
rustfs       ██████████████████████████████ 1.2ms
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 605 ops/s | 1.7ms | 1.7ms | 1.7ms | 0 |
| minio | 215 ops/s | 4.7ms | 4.7ms | 4.7ms | 0 |
| rustfs | 98 ops/s | 10.2ms | 10.2ms | 10.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 605 ops/s
minio        ██████████ 215 ops/s
rustfs       ████ 98 ops/s
```

**Latency (P50)**
```
liteio       ████ 1.7ms
minio        █████████████ 4.7ms
rustfs       ██████████████████████████████ 10.2ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 62 ops/s | 16.1ms | 16.1ms | 16.1ms | 0 |
| minio | 20 ops/s | 49.6ms | 49.6ms | 49.6ms | 0 |
| rustfs | 10 ops/s | 99.6ms | 99.6ms | 99.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 62 ops/s
minio        █████████ 20 ops/s
rustfs       ████ 10 ops/s
```

**Latency (P50)**
```
liteio       ████ 16.1ms
minio        ██████████████ 49.6ms
rustfs       ██████████████████████████████ 99.6ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6 ops/s | 160.6ms | 160.6ms | 160.6ms | 0 |
| minio | 3 ops/s | 352.8ms | 352.8ms | 352.8ms | 0 |
| rustfs | 1 ops/s | 1.33s | 1.33s | 1.33s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 6 ops/s
minio        █████████████ 3 ops/s
rustfs       ███ 1 ops/s
```

**Latency (P50)**
```
liteio       ███ 160.6ms
minio        ███████ 352.8ms
rustfs       ██████████████████████████████ 1.33s
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1 ops/s | 1.68s | 1.68s | 1.68s | 0 |
| minio | 0 ops/s | 4.19s | 4.19s | 4.19s | 0 |
| rustfs | 0 ops/s | 9.61s | 9.61s | 9.61s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1 ops/s
minio        ████████████ 0 ops/s
rustfs       █████ 0 ops/s
```

**Latency (P50)**
```
liteio       █████ 1.68s
minio        █████████████ 4.19s
rustfs       ██████████████████████████████ 9.61s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2632 ops/s | 379.9us | 379.9us | 379.9us | 0 |
| minio | 1038 ops/s | 963.0us | 963.0us | 963.0us | 0 |
| rustfs | 773 ops/s | 1.3ms | 1.3ms | 1.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2632 ops/s
minio        ███████████ 1038 ops/s
rustfs       ████████ 773 ops/s
```

**Latency (P50)**
```
liteio       ████████ 379.9us
minio        ██████████████████████ 963.0us
rustfs       ██████████████████████████████ 1.3ms
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3356 ops/s | 298.0us | 298.0us | 298.0us | 0 |
| minio | 1411 ops/s | 708.9us | 708.9us | 708.9us | 0 |
| rustfs | 545 ops/s | 1.8ms | 1.8ms | 1.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3356 ops/s
minio        ████████████ 1411 ops/s
rustfs       ████ 545 ops/s
```

**Latency (P50)**
```
liteio       ████ 298.0us
minio        ███████████ 708.9us
rustfs       ██████████████████████████████ 1.8ms
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1239 ops/s | 806.9us | 806.9us | 806.9us | 0 |
| minio | 330 ops/s | 3.0ms | 3.0ms | 3.0ms | 0 |
| rustfs | 124 ops/s | 8.1ms | 8.1ms | 8.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1239 ops/s
minio        ███████ 330 ops/s
rustfs       ██ 124 ops/s
```

**Latency (P50)**
```
liteio       ██ 806.9us
minio        ███████████ 3.0ms
rustfs       ██████████████████████████████ 8.1ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 206 ops/s | 4.9ms | 4.9ms | 4.9ms | 0 |
| minio | 71 ops/s | 14.0ms | 14.0ms | 14.0ms | 0 |
| rustfs | 15 ops/s | 66.9ms | 66.9ms | 66.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 206 ops/s
minio        ██████████ 71 ops/s
rustfs       ██ 15 ops/s
```

**Latency (P50)**
```
liteio       ██ 4.9ms
minio        ██████ 14.0ms
rustfs       ██████████████████████████████ 66.9ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6 ops/s | 176.9ms | 176.9ms | 176.9ms | 0 |
| minio | 5 ops/s | 195.7ms | 195.7ms | 195.7ms | 0 |
| rustfs | 1 ops/s | 837.5ms | 837.5ms | 837.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 6 ops/s
minio        ███████████████████████████ 5 ops/s
rustfs       ██████ 1 ops/s
```

**Latency (P50)**
```
liteio       ██████ 176.9ms
minio        ███████ 195.7ms
rustfs       ██████████████████████████████ 837.5ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.43 MB/s | 684.9us | 684.9us | 684.9us | 0 |
| rustfs | 0.71 MB/s | 1.4ms | 1.4ms | 1.4ms | 0 |
| minio | 0.53 MB/s | 1.8ms | 1.8ms | 1.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.43 MB/s
rustfs       ██████████████ 0.71 MB/s
minio        ███████████ 0.53 MB/s
```

**Latency (P50)**
```
liteio       ███████████ 684.9us
rustfs       ██████████████████████ 1.4ms
minio        ██████████████████████████████ 1.8ms
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.44 MB/s | 2.2ms | 2.2ms | 2.2ms | 0 |
| rustfs | 1.14 MB/s | 8.6ms | 8.6ms | 8.6ms | 0 |
| minio | 0.90 MB/s | 10.9ms | 10.9ms | 10.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.44 MB/s
rustfs       ███████ 1.14 MB/s
minio        ██████ 0.90 MB/s
```

**Latency (P50)**
```
liteio       ██████ 2.2ms
rustfs       ███████████████████████ 8.6ms
minio        ██████████████████████████████ 10.9ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5.17 MB/s | 18.9ms | 18.9ms | 18.9ms | 0 |
| minio | 1.02 MB/s | 96.1ms | 96.1ms | 96.1ms | 0 |
| rustfs | 0.84 MB/s | 116.2ms | 116.2ms | 116.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5.17 MB/s
minio        █████ 1.02 MB/s
rustfs       ████ 0.84 MB/s
```

**Latency (P50)**
```
liteio       ████ 18.9ms
minio        ████████████████████████ 96.1ms
rustfs       ██████████████████████████████ 116.2ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5.16 MB/s | 189.4ms | 189.4ms | 189.4ms | 0 |
| rustfs | 1.13 MB/s | 866.6ms | 866.6ms | 866.6ms | 0 |
| minio | 0.87 MB/s | 1.13s | 1.13s | 1.13s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5.16 MB/s
rustfs       ██████ 1.13 MB/s
minio        █████ 0.87 MB/s
```

**Latency (P50)**
```
liteio       █████ 189.4ms
rustfs       ███████████████████████ 866.6ms
minio        ██████████████████████████████ 1.13s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.92 MB/s | 1.98s | 1.98s | 1.98s | 0 |
| rustfs | 1.15 MB/s | 8.53s | 8.53s | 8.53s | 0 |
| minio | 0.99 MB/s | 9.88s | 9.88s | 9.88s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.92 MB/s
rustfs       ██████ 1.15 MB/s
minio        ██████ 0.99 MB/s
```

**Latency (P50)**
```
liteio       ██████ 1.98s
rustfs       █████████████████████████ 8.53s
minio        ██████████████████████████████ 9.88s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1241 ops/s | 776.6us | 997.6us | 1.1ms | 0 |
| minio | 618 ops/s | 1.5ms | 2.3ms | 2.5ms | 0 |
| rustfs | 132 ops/s | 7.7ms | 8.6ms | 8.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1241 ops/s
minio        ██████████████ 618 ops/s
rustfs       ███ 132 ops/s
```

**Latency (P50)**
```
liteio       ███ 776.6us
minio        █████ 1.5ms
rustfs       ██████████████████████████████ 7.7ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 2.43 MB/s | 6.7ms | 11.2ms | 12.1ms | 0 |
| minio | 2.22 MB/s | 7.1ms | 13.4ms | 14.1ms | 0 |
| liteio | 1.94 MB/s | 9.2ms | 10.9ms | 11.3ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 2.43 MB/s
minio        ███████████████████████████ 2.22 MB/s
liteio       ███████████████████████ 1.94 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████████ 6.7ms
minio        ███████████████████████ 7.1ms
liteio       ██████████████████████████████ 9.2ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 3.17 MB/s | 4.9ms | 7.3ms | 7.9ms | 0 |
| rustfs | 1.28 MB/s | 11.9ms | 16.2ms | 18.9ms | 0 |
| liteio | 1.08 MB/s | 15.0ms | 16.1ms | 16.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 3.17 MB/s
rustfs       ████████████ 1.28 MB/s
liteio       ██████████ 1.08 MB/s
```

**Latency (P50)**
```
minio        █████████ 4.9ms
rustfs       ███████████████████████ 11.9ms
liteio       ██████████████████████████████ 15.0ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.29 MB/s | 13.8ms | 18.1ms | 19.6ms | 0 |
| minio | 0.89 MB/s | 18.1ms | 24.1ms | 25.0ms | 0 |
| liteio | 0.82 MB/s | 19.3ms | 20.7ms | 20.9ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.29 MB/s
minio        ████████████████████ 0.89 MB/s
liteio       ███████████████████ 0.82 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████████ 13.8ms
minio        ████████████████████████████ 18.1ms
liteio       ██████████████████████████████ 19.3ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 152.06 MB/s | 99.3ms | 105.8ms | 105.8ms | 0 |
| rustfs | 144.45 MB/s | 104.7ms | 114.0ms | 114.0ms | 0 |
| minio | 134.60 MB/s | 110.4ms | 119.1ms | 119.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 152.06 MB/s
rustfs       ████████████████████████████ 144.45 MB/s
minio        ██████████████████████████ 134.60 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████ 99.3ms
rustfs       ████████████████████████████ 104.7ms
minio        ██████████████████████████████ 110.4ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.48 MB/s | 217.7us | 272.8us | 209.7us | 273.0us | 289.4us | 0 |
| minio | 1.42 MB/s | 684.8us | 958.9us | 653.2us | 959.3us | 1.0ms | 0 |
| rustfs | 1.06 MB/s | 917.6us | 1.2ms | 915.3us | 1.2ms | 1.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.48 MB/s
minio        █████████ 1.42 MB/s
rustfs       ███████ 1.06 MB/s
```

**Latency (P50)**
```
liteio       ██████ 209.7us
minio        █████████████████████ 653.2us
rustfs       ██████████████████████████████ 915.3us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.74 MB/s | 560.1us | 745.2us | 559.6us | 745.3us | 779.5us | 0 |
| rustfs | 1.08 MB/s | 904.9us | 1.4ms | 839.0us | 1.4ms | 2.3ms | 0 |
| minio | 0.82 MB/s | 1.2ms | 4.1ms | 714.0us | 4.1ms | 4.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.74 MB/s
rustfs       ██████████████████ 1.08 MB/s
minio        ██████████████ 0.82 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 559.6us
rustfs       ██████████████████████████████ 839.0us
minio        █████████████████████████ 714.0us
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rustfs | 0.59 MB/s | 1.7ms | 4.3ms | 1.3ms | 4.3ms | 4.7ms | 0 |
| minio | 0.47 MB/s | 2.1ms | 3.1ms | 2.1ms | 3.1ms | 3.9ms | 0 |
| liteio | 0.10 MB/s | 9.9ms | 18.6ms | 4.0ms | 18.6ms | 18.9ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.59 MB/s
minio        ████████████████████████ 0.47 MB/s
liteio       █████ 0.10 MB/s
```

**Latency (P50)**
```
rustfs       █████████ 1.3ms
minio        ███████████████ 2.1ms
liteio       ██████████████████████████████ 4.0ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.40 MB/s | 2.4ms | 3.1ms | 2.4ms | 3.1ms | 3.4ms | 0 |
| rustfs | 0.40 MB/s | 2.5ms | 5.1ms | 2.0ms | 5.1ms | 6.4ms | 0 |
| minio | 0.11 MB/s | 8.7ms | 10.7ms | 9.1ms | 10.7ms | 10.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.40 MB/s
rustfs       █████████████████████████████ 0.40 MB/s
minio        ████████ 0.11 MB/s
```

**Latency (P50)**
```
liteio       ████████ 2.4ms
rustfs       ██████ 2.0ms
minio        ██████████████████████████████ 9.1ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.85 MB/s | 1.1ms | 2.1ms | 1.1ms | 2.1ms | 2.2ms | 0 |
| minio | 0.75 MB/s | 1.3ms | 2.0ms | 1.2ms | 2.0ms | 2.3ms | 0 |
| rustfs | 0.54 MB/s | 1.8ms | 3.3ms | 1.7ms | 3.3ms | 3.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.85 MB/s
minio        ██████████████████████████ 0.75 MB/s
rustfs       ███████████████████ 0.54 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 1.1ms
minio        ██████████████████████ 1.2ms
rustfs       ██████████████████████████████ 1.7ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.52 MB/s | 1.9ms | 2.7ms | 1.9ms | 2.7ms | 2.8ms | 0 |
| minio | 0.40 MB/s | 2.4ms | 4.8ms | 2.0ms | 4.8ms | 5.6ms | 0 |
| rustfs | 0.36 MB/s | 2.7ms | 4.9ms | 2.4ms | 4.9ms | 5.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.52 MB/s
minio        ███████████████████████ 0.40 MB/s
rustfs       ████████████████████ 0.36 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████ 1.9ms
minio        ████████████████████████ 2.0ms
rustfs       ██████████████████████████████ 2.4ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.71 MB/s | 325.9us | 531.2us | 660.7us | 0 |
| minio | 0.77 MB/s | 1.3ms | 1.5ms | 1.8ms | 0 |
| rustfs | 0.68 MB/s | 1.2ms | 2.2ms | 5.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.71 MB/s
minio        ████████ 0.77 MB/s
rustfs       ███████ 0.68 MB/s
```

**Latency (P50)**
```
liteio       ███████ 325.9us
minio        ██████████████████████████████ 1.3ms
rustfs       █████████████████████████████ 1.2ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.83 MB/s | 939.6us | 3.0ms | 3.3ms | 0 |
| rustfs | 0.49 MB/s | 1.9ms | 3.5ms | 4.3ms | 0 |
| minio | 0.37 MB/s | 2.3ms | 4.6ms | 6.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.83 MB/s
rustfs       █████████████████ 0.49 MB/s
minio        █████████████ 0.37 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 939.6us
rustfs       ████████████████████████ 1.9ms
minio        ██████████████████████████████ 2.3ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.39 MB/s | 2.0ms | 5.3ms | 5.5ms | 0 |
| rustfs | 0.08 MB/s | 14.4ms | 17.4ms | 26.2ms | 0 |
| minio | 0.08 MB/s | 12.6ms | 18.2ms | 18.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.39 MB/s
rustfs       ██████ 0.08 MB/s
minio        ██████ 0.08 MB/s
```

**Latency (P50)**
```
liteio       ████ 2.0ms
rustfs       ██████████████████████████████ 14.4ms
minio        ██████████████████████████ 12.6ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.11 MB/s | 8.4ms | 12.8ms | 13.2ms | 0 |
| minio | 0.09 MB/s | 10.2ms | 16.8ms | 18.6ms | 0 |
| rustfs | 0.07 MB/s | 14.0ms | 18.4ms | 19.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.11 MB/s
minio        ████████████████████████ 0.09 MB/s
rustfs       ██████████████████ 0.07 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 8.4ms
minio        █████████████████████ 10.2ms
rustfs       ██████████████████████████████ 14.0ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.37 MB/s | 2.2ms | 6.2ms | 8.1ms | 0 |
| rustfs | 0.23 MB/s | 4.2ms | 6.5ms | 7.0ms | 0 |
| minio | 0.20 MB/s | 4.9ms | 7.7ms | 10.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.37 MB/s
rustfs       ██████████████████ 0.23 MB/s
minio        ███████████████ 0.20 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 2.2ms
rustfs       █████████████████████████ 4.2ms
minio        ██████████████████████████████ 4.9ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.17 MB/s | 5.7ms | 9.5ms | 10.5ms | 0 |
| liteio | 0.15 MB/s | 6.3ms | 11.8ms | 13.0ms | 0 |
| minio | 0.12 MB/s | 7.2ms | 17.5ms | 18.4ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.17 MB/s
liteio       ██████████████████████████ 0.15 MB/s
minio        ████████████████████ 0.12 MB/s
```

**Latency (P50)**
```
rustfs       ███████████████████████ 5.7ms
liteio       ██████████████████████████ 6.3ms
minio        ██████████████████████████████ 7.2ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 253.17 MB/s | 985.0us | 1.0ms | 1.1ms | 0 |
| minio | 124.19 MB/s | 1.5ms | 2.2ms | 10.2ms | 0 |
| rustfs | 115.56 MB/s | 2.2ms | 2.5ms | 2.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 253.17 MB/s
minio        ██████████████ 124.19 MB/s
rustfs       █████████████ 115.56 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 985.0us
minio        █████████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.2ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 251.50 MB/s | 987.9us | 1.1ms | 1.1ms | 0 |
| minio | 175.48 MB/s | 1.4ms | 1.6ms | 1.7ms | 0 |
| rustfs | 119.32 MB/s | 2.0ms | 2.5ms | 2.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 251.50 MB/s
minio        ████████████████████ 175.48 MB/s
rustfs       ██████████████ 119.32 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 987.9us
minio        ████████████████████ 1.4ms
rustfs       ██████████████████████████████ 2.0ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 234.68 MB/s | 994.3us | 1.1ms | 2.4ms | 0 |
| minio | 148.08 MB/s | 1.6ms | 2.2ms | 2.5ms | 0 |
| rustfs | 120.31 MB/s | 1.9ms | 2.4ms | 5.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 234.68 MB/s
minio        ██████████████████ 148.08 MB/s
rustfs       ███████████████ 120.31 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 994.3us
minio        ████████████████████████ 1.6ms
rustfs       ██████████████████████████████ 1.9ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rustfs | 330.71 MB/s | 1.9ms | 1.9ms | 302.3ms | 303.8ms | 303.8ms | 0 |
| minio | 326.19 MB/s | 927.9us | 978.2us | 298.0ms | 299.1ms | 299.1ms | 0 |
| liteio | 305.79 MB/s | 3.6ms | 4.5ms | 325.3ms | 336.6ms | 336.6ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 330.71 MB/s
minio        █████████████████████████████ 326.19 MB/s
liteio       ███████████████████████████ 305.79 MB/s
```

**Latency (P50)**
```
rustfs       ███████████████████████████ 302.3ms
minio        ███████████████████████████ 298.0ms
liteio       ██████████████████████████████ 325.3ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 316.16 MB/s | 1.6ms | 1.2ms | 31.4ms | 31.9ms | 31.9ms | 0 |
| liteio | 302.20 MB/s | 2.1ms | 2.4ms | 32.3ms | 34.3ms | 34.3ms | 0 |
| rustfs | 285.83 MB/s | 5.2ms | 6.8ms | 34.4ms | 36.7ms | 36.7ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 316.16 MB/s
liteio       ████████████████████████████ 302.20 MB/s
rustfs       ███████████████████████████ 285.83 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 31.4ms
liteio       ████████████████████████████ 32.3ms
rustfs       ██████████████████████████████ 34.4ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 3.96 MB/s | 246.2us | 305.0us | 242.8us | 305.2us | 340.5us | 0 |
| rustfs | 1.92 MB/s | 508.5us | 567.7us | 530.5us | 568.1us | 580.8us | 0 |
| minio | 1.71 MB/s | 570.4us | 705.1us | 562.5us | 705.6us | 743.6us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.96 MB/s
rustfs       ██████████████ 1.92 MB/s
minio        ████████████ 1.71 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 242.8us
rustfs       ████████████████████████████ 530.5us
minio        ██████████████████████████████ 562.5us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 300.07 MB/s | 261.8us | 410.1us | 3.3ms | 3.5ms | 3.5ms | 0 |
| minio | 236.77 MB/s | 1.2ms | 1.2ms | 4.0ms | 4.2ms | 4.2ms | 0 |
| rustfs | 214.46 MB/s | 1.8ms | 2.6ms | 4.5ms | 5.5ms | 5.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 300.07 MB/s
minio        ███████████████████████ 236.77 MB/s
rustfs       █████████████████████ 214.46 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████ 3.3ms
minio        ██████████████████████████ 4.0ms
rustfs       ██████████████████████████████ 4.5ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 146.60 MB/s | 260.2us | 333.0us | 386.6us | 473.1us | 843.6us | 0 |
| minio | 99.46 MB/s | 468.6us | 558.8us | 622.0us | 697.9us | 709.0us | 0 |
| rustfs | 83.17 MB/s | 640.7us | 684.7us | 722.3us | 787.6us | 799.7us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 146.60 MB/s
minio        ████████████████████ 99.46 MB/s
rustfs       █████████████████ 83.17 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 386.6us
minio        █████████████████████████ 622.0us
rustfs       ██████████████████████████████ 722.3us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5134 ops/s | 166.7us | 300.9us | 321.5us | 0 |
| minio | 3637 ops/s | 275.5us | 315.4us | 344.0us | 0 |
| rustfs | 3278 ops/s | 300.2us | 324.4us | 353.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5134 ops/s
minio        █████████████████████ 3637 ops/s
rustfs       ███████████████████ 3278 ops/s
```

**Latency (P50)**
```
liteio       ████████████████ 166.7us
minio        ███████████████████████████ 275.5us
rustfs       ██████████████████████████████ 300.2us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 201.75 MB/s | 489.4ms | 508.5ms | 508.5ms | 0 |
| rustfs | 168.35 MB/s | 604.9ms | 619.9ms | 619.9ms | 0 |
| minio | 157.71 MB/s | 641.7ms | 654.4ms | 654.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 201.75 MB/s
rustfs       █████████████████████████ 168.35 MB/s
minio        ███████████████████████ 157.71 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████ 489.4ms
rustfs       ████████████████████████████ 604.9ms
minio        ██████████████████████████████ 641.7ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 201.57 MB/s | 49.3ms | 54.6ms | 54.6ms | 0 |
| minio | 155.67 MB/s | 59.9ms | 71.4ms | 71.4ms | 0 |
| rustfs | 152.24 MB/s | 67.2ms | 72.2ms | 72.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 201.57 MB/s
minio        ███████████████████████ 155.67 MB/s
rustfs       ██████████████████████ 152.24 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████ 49.3ms
minio        ██████████████████████████ 59.9ms
rustfs       ██████████████████████████████ 67.2ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.05 MB/s | 864.0us | 1.3ms | 1.6ms | 0 |
| liteio | 0.95 MB/s | 1.0ms | 1.6ms | 1.8ms | 0 |
| minio | 0.93 MB/s | 1.0ms | 1.3ms | 1.7ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.05 MB/s
liteio       ███████████████████████████ 0.95 MB/s
minio        ██████████████████████████ 0.93 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████████ 864.0us
liteio       ██████████████████████████████ 1.0ms
minio        █████████████████████████████ 1.0ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 148.83 MB/s | 7.2ms | 7.8ms | 7.8ms | 0 |
| rustfs | 122.80 MB/s | 7.9ms | 9.1ms | 9.1ms | 0 |
| minio | 109.26 MB/s | 8.9ms | 10.6ms | 10.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 148.83 MB/s
rustfs       ████████████████████████ 122.80 MB/s
minio        ██████████████████████ 109.26 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████ 7.2ms
rustfs       ██████████████████████████ 7.9ms
minio        ██████████████████████████████ 8.9ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 126.33 MB/s | 486.9us | 547.5us | 558.9us | 0 |
| rustfs | 52.91 MB/s | 1.1ms | 1.4ms | 1.5ms | 0 |
| minio | 34.60 MB/s | 1.7ms | 2.3ms | 2.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 126.33 MB/s
rustfs       ████████████ 52.91 MB/s
minio        ████████ 34.60 MB/s
```

**Latency (P50)**
```
liteio       ████████ 486.9us
rustfs       ███████████████████ 1.1ms
minio        ██████████████████████████████ 1.7ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 73.88MiB / 7.653GiB | 73.9 MB | - | 1.7% | (no data) | 209MB / 2.96GB |
| minio | 383.5MiB / 7.653GiB | 383.5 MB | - | 0.1% | 1925.1 MB | 1.55GB / 13.6GB |
| rustfs | 777.3MiB / 7.653GiB | 777.3 MB | - | 0.1% | 1923.1 MB | 424MB / 13.4GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** liteio
- **Read-heavy workloads:** rustfs

---

*Generated by storage benchmark CLI*
