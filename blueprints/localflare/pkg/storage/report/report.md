# Storage Benchmark Report

**Generated:** 2026-01-15T23:56:53+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio (won 39/51 benchmarks, 76%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio | 39 | 76% |
| 2 | rustfs | 7 | 14% |
| 3 | minio | 5 | 10% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 4.5 MB/s | +79% vs minio |
| Small Write (1KB) | liteio | 2.3 MB/s | +95% vs rustfs |
| Large Read (10MB) | minio | 314.2 MB/s | +13% vs rustfs |
| Large Write (10MB) | liteio | 210.4 MB/s | +24% vs rustfs |
| Delete | liteio | 5.1K ops/s | +85% vs minio |
| Stat | liteio | 5.3K ops/s | +42% vs minio |
| List (100 objects) | liteio | 1.2K ops/s | +77% vs minio |
| Copy | liteio | 2.2 MB/s | 2.3x vs minio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **liteio** | 197 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 329 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 3500 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |
| Memory Constrained | **liteio** | 170 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 196.6 | 209.4 | 507.6ms | 453.5ms |
| minio | 154.3 | 329.4 | 649.8ms | 299.6ms |
| rustfs | 166.5 | 322.8 | 622.7ms | 309.2ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 2398 | 4602 | 346.8us | 212.7us |
| minio | 758 | 2565 | 1.2ms | 373.9us |
| rustfs | 1228 | 2129 | 742.9us | 478.2us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 5271 | 1177 | 5084 |
| minio | 3712 | 664 | 2742 |
| rustfs | 3534 | 151 | 1225 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 2.87 | 0.75 | 0.32 | 0.18 | 0.07 | 0.12 |
| minio | 0.85 | 0.42 | 0.19 | 0.12 | 0.06 | 0.08 |
| rustfs | 1.15 | 0.52 | 0.25 | 0.15 | 0.12 | 0.13 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 4.36 | 1.47 | 0.80 | 0.52 | 0.42 | 0.38 |
| minio | 2.14 | 1.38 | 0.78 | 0.45 | 0.41 | 0.43 |
| rustfs | 1.65 | 1.13 | 0.72 | 0.49 | 0.43 | 0.43 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 411.6us | 2.5ms | 23.5ms | 245.2ms | 2.41s |
| minio | 1.7ms | 10.2ms | 111.1ms | 1.00s | 9.51s |
| rustfs | 1.3ms | 7.7ms | 75.2ms | 727.3ms | 7.18s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 367.7us | 339.7us | 948.2us | 5.4ms | 199.4ms |
| minio | 645.7us | 630.6us | 2.8ms | 19.0ms | 186.9ms |
| rustfs | 1.0ms | 1.5ms | 7.2ms | 62.4ms | 756.0ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 169.5 MB | 2.4% |
| minio | 428.7 MB | 0.0% |
| rustfs | 659.3 MB | 0.1% |

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
| liteio | 2.20 MB/s | 328.1us | 827.9us | 1.7ms | 0 |
| minio | 0.95 MB/s | 959.2us | 1.3ms | 1.8ms | 0 |
| rustfs | 0.85 MB/s | 994.6us | 2.3ms | 3.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.20 MB/s
minio        ████████████ 0.95 MB/s
rustfs       ███████████ 0.85 MB/s
```

**Latency (P50)**
```
liteio       █████████ 328.1us
minio        ████████████████████████████ 959.2us
rustfs       ██████████████████████████████ 994.6us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5084 ops/s | 189.6us | 258.0us | 295.6us | 0 |
| minio | 2742 ops/s | 347.5us | 461.6us | 632.4us | 0 |
| rustfs | 1225 ops/s | 818.9us | 892.1us | 971.1us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5084 ops/s
minio        ████████████████ 2742 ops/s
rustfs       ███████ 1225 ops/s
```

**Latency (P50)**
```
liteio       ██████ 189.6us
minio        ████████████ 347.5us
rustfs       ██████████████████████████████ 818.9us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.38 MB/s | 238.8us | 339.7us | 348.5us | 0 |
| rustfs | 0.12 MB/s | 715.7us | 794.9us | 1.1ms | 0 |
| minio | 0.09 MB/s | 990.0us | 1.3ms | 2.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.38 MB/s
rustfs       █████████ 0.12 MB/s
minio        ██████ 0.09 MB/s
```

**Latency (P50)**
```
liteio       ███████ 238.8us
rustfs       █████████████████████ 715.7us
minio        ██████████████████████████████ 990.0us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1319 ops/s | 732.4us | 889.0us | 969.9us | 0 |
| liteio | 1001 ops/s | 446.6us | 2.3ms | 3.4ms | 0 |
| minio | 780 ops/s | 1.0ms | 2.5ms | 2.6ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1319 ops/s
liteio       ██████████████████████ 1001 ops/s
minio        █████████████████ 780 ops/s
```

**Latency (P50)**
```
rustfs       ████████████████████ 732.4us
liteio       ████████████ 446.6us
minio        ██████████████████████████████ 1.0ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.41 MB/s | 227.8us | 266.4us | 286.1us | 0 |
| rustfs | 0.13 MB/s | 710.5us | 798.7us | 818.2us | 0 |
| minio | 0.09 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.41 MB/s
rustfs       █████████ 0.13 MB/s
minio        ██████ 0.09 MB/s
```

**Latency (P50)**
```
liteio       ██████ 227.8us
rustfs       █████████████████████ 710.5us
minio        ██████████████████████████████ 1.0ms
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4854 ops/s | 206.0us | 206.0us | 206.0us | 0 |
| minio | 2099 ops/s | 476.3us | 476.3us | 476.3us | 0 |
| rustfs | 1164 ops/s | 858.9us | 858.9us | 858.9us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4854 ops/s
minio        ████████████ 2099 ops/s
rustfs       ███████ 1164 ops/s
```

**Latency (P50)**
```
liteio       ███████ 206.0us
minio        ████████████████ 476.3us
rustfs       ██████████████████████████████ 858.9us
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 491 ops/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| minio | 238 ops/s | 4.2ms | 4.2ms | 4.2ms | 0 |
| rustfs | 114 ops/s | 8.7ms | 8.7ms | 8.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 491 ops/s
minio        ██████████████ 238 ops/s
rustfs       ██████ 114 ops/s
```

**Latency (P50)**
```
liteio       ██████ 2.0ms
minio        ██████████████ 4.2ms
rustfs       ██████████████████████████████ 8.7ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 54 ops/s | 18.4ms | 18.4ms | 18.4ms | 0 |
| minio | 23 ops/s | 42.7ms | 42.7ms | 42.7ms | 0 |
| rustfs | 12 ops/s | 84.8ms | 84.8ms | 84.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 54 ops/s
minio        ████████████ 23 ops/s
rustfs       ██████ 12 ops/s
```

**Latency (P50)**
```
liteio       ██████ 18.4ms
minio        ███████████████ 42.7ms
rustfs       ██████████████████████████████ 84.8ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5 ops/s | 199.2ms | 199.2ms | 199.2ms | 0 |
| minio | 3 ops/s | 394.1ms | 394.1ms | 394.1ms | 0 |
| rustfs | 1 ops/s | 856.5ms | 856.5ms | 856.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5 ops/s
minio        ███████████████ 3 ops/s
rustfs       ██████ 1 ops/s
```

**Latency (P50)**
```
liteio       ██████ 199.2ms
minio        █████████████ 394.1ms
rustfs       ██████████████████████████████ 856.5ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0 ops/s | 2.04s | 2.04s | 2.04s | 0 |
| minio | 0 ops/s | 3.93s | 3.93s | 3.93s | 0 |
| rustfs | 0 ops/s | 8.53s | 8.53s | 8.53s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0 ops/s
minio        ███████████████ 0 ops/s
rustfs       ███████ 0 ops/s
```

**Latency (P50)**
```
liteio       ███████ 2.04s
minio        █████████████ 3.93s
rustfs       ██████████████████████████████ 8.53s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2720 ops/s | 367.7us | 367.7us | 367.7us | 0 |
| minio | 1549 ops/s | 645.7us | 645.7us | 645.7us | 0 |
| rustfs | 995 ops/s | 1.0ms | 1.0ms | 1.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2720 ops/s
minio        █████████████████ 1549 ops/s
rustfs       ██████████ 995 ops/s
```

**Latency (P50)**
```
liteio       ██████████ 367.7us
minio        ███████████████████ 645.7us
rustfs       ██████████████████████████████ 1.0ms
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2944 ops/s | 339.7us | 339.7us | 339.7us | 0 |
| minio | 1586 ops/s | 630.6us | 630.6us | 630.6us | 0 |
| rustfs | 675 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2944 ops/s
minio        ████████████████ 1586 ops/s
rustfs       ██████ 675 ops/s
```

**Latency (P50)**
```
liteio       ██████ 339.7us
minio        ████████████ 630.6us
rustfs       ██████████████████████████████ 1.5ms
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1055 ops/s | 948.2us | 948.2us | 948.2us | 0 |
| minio | 354 ops/s | 2.8ms | 2.8ms | 2.8ms | 0 |
| rustfs | 138 ops/s | 7.2ms | 7.2ms | 7.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1055 ops/s
minio        ██████████ 354 ops/s
rustfs       ███ 138 ops/s
```

**Latency (P50)**
```
liteio       ███ 948.2us
minio        ███████████ 2.8ms
rustfs       ██████████████████████████████ 7.2ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 186 ops/s | 5.4ms | 5.4ms | 5.4ms | 0 |
| minio | 53 ops/s | 19.0ms | 19.0ms | 19.0ms | 0 |
| rustfs | 16 ops/s | 62.4ms | 62.4ms | 62.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 186 ops/s
minio        ████████ 53 ops/s
rustfs       ██ 16 ops/s
```

**Latency (P50)**
```
liteio       ██ 5.4ms
minio        █████████ 19.0ms
rustfs       ██████████████████████████████ 62.4ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 5 ops/s | 186.9ms | 186.9ms | 186.9ms | 0 |
| liteio | 5 ops/s | 199.4ms | 199.4ms | 199.4ms | 0 |
| rustfs | 1 ops/s | 756.0ms | 756.0ms | 756.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 5 ops/s
liteio       ████████████████████████████ 5 ops/s
rustfs       ███████ 1 ops/s
```

**Latency (P50)**
```
minio        ███████ 186.9ms
liteio       ███████ 199.4ms
rustfs       ██████████████████████████████ 756.0ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.37 MB/s | 411.6us | 411.6us | 411.6us | 0 |
| rustfs | 0.78 MB/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| minio | 0.57 MB/s | 1.7ms | 1.7ms | 1.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.37 MB/s
rustfs       █████████ 0.78 MB/s
minio        ███████ 0.57 MB/s
```

**Latency (P50)**
```
liteio       ███████ 411.6us
rustfs       ██████████████████████ 1.3ms
minio        ██████████████████████████████ 1.7ms
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.95 MB/s | 2.5ms | 2.5ms | 2.5ms | 0 |
| rustfs | 1.27 MB/s | 7.7ms | 7.7ms | 7.7ms | 0 |
| minio | 0.96 MB/s | 10.2ms | 10.2ms | 10.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.95 MB/s
rustfs       █████████ 1.27 MB/s
minio        ███████ 0.96 MB/s
```

**Latency (P50)**
```
liteio       ███████ 2.5ms
rustfs       ██████████████████████ 7.7ms
minio        ██████████████████████████████ 10.2ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.15 MB/s | 23.5ms | 23.5ms | 23.5ms | 0 |
| rustfs | 1.30 MB/s | 75.2ms | 75.2ms | 75.2ms | 0 |
| minio | 0.88 MB/s | 111.1ms | 111.1ms | 111.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.15 MB/s
rustfs       █████████ 1.30 MB/s
minio        ██████ 0.88 MB/s
```

**Latency (P50)**
```
liteio       ██████ 23.5ms
rustfs       ████████████████████ 75.2ms
minio        ██████████████████████████████ 111.1ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.98 MB/s | 245.2ms | 245.2ms | 245.2ms | 0 |
| rustfs | 1.34 MB/s | 727.3ms | 727.3ms | 727.3ms | 0 |
| minio | 0.98 MB/s | 1.00s | 1.00s | 1.00s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.98 MB/s
rustfs       ██████████ 1.34 MB/s
minio        ███████ 0.98 MB/s
```

**Latency (P50)**
```
liteio       ███████ 245.2ms
rustfs       █████████████████████ 727.3ms
minio        ██████████████████████████████ 1.00s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.05 MB/s | 2.41s | 2.41s | 2.41s | 0 |
| rustfs | 1.36 MB/s | 7.18s | 7.18s | 7.18s | 0 |
| minio | 1.03 MB/s | 9.51s | 9.51s | 9.51s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.05 MB/s
rustfs       ██████████ 1.36 MB/s
minio        ███████ 1.03 MB/s
```

**Latency (P50)**
```
liteio       ███████ 2.41s
rustfs       ██████████████████████ 7.18s
minio        ██████████████████████████████ 9.51s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1177 ops/s | 815.6us | 1.1ms | 1.3ms | 0 |
| minio | 664 ops/s | 1.5ms | 1.7ms | 1.8ms | 0 |
| rustfs | 151 ops/s | 6.7ms | 7.1ms | 7.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1177 ops/s
minio        ████████████████ 664 ops/s
rustfs       ███ 151 ops/s
```

**Latency (P50)**
```
liteio       ███ 815.6us
minio        ██████ 1.5ms
rustfs       ██████████████████████████████ 6.7ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 2.20 MB/s | 7.6ms | 11.4ms | 12.2ms | 0 |
| liteio | 2.09 MB/s | 8.1ms | 9.4ms | 9.8ms | 0 |
| minio | 2.09 MB/s | 7.1ms | 13.8ms | 14.3ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 2.20 MB/s
liteio       ████████████████████████████ 2.09 MB/s
minio        ████████████████████████████ 2.09 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████████████ 7.6ms
liteio       ██████████████████████████████ 8.1ms
minio        ██████████████████████████ 7.1ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 3.39 MB/s | 4.3ms | 8.2ms | 8.3ms | 0 |
| rustfs | 2.52 MB/s | 6.3ms | 8.5ms | 8.7ms | 0 |
| liteio | 2.08 MB/s | 7.6ms | 9.5ms | 10.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 3.39 MB/s
rustfs       ██████████████████████ 2.52 MB/s
liteio       ██████████████████ 2.08 MB/s
```

**Latency (P50)**
```
minio        █████████████████ 4.3ms
rustfs       █████████████████████████ 6.3ms
liteio       ██████████████████████████████ 7.6ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.30 MB/s | 15.8ms | 16.3ms | 17.3ms | 0 |
| minio | 1.22 MB/s | 13.1ms | 19.2ms | 20.1ms | 0 |
| liteio | 0.93 MB/s | 18.5ms | 21.1ms | 21.9ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.30 MB/s
minio        ████████████████████████████ 1.22 MB/s
liteio       █████████████████████ 0.93 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████████████ 15.8ms
minio        █████████████████████ 13.1ms
liteio       ██████████████████████████████ 18.5ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 145.55 MB/s | 103.4ms | 114.7ms | 114.7ms | 0 |
| rustfs | 125.94 MB/s | 97.5ms | 203.0ms | 203.0ms | 0 |
| liteio | 105.80 MB/s | 131.0ms | 184.1ms | 184.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 145.55 MB/s
rustfs       █████████████████████████ 125.94 MB/s
liteio       █████████████████████ 105.80 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████ 103.4ms
rustfs       ██████████████████████ 97.5ms
liteio       ██████████████████████████████ 131.0ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.36 MB/s | 223.8us | 270.2us | 217.5us | 270.3us | 297.5us | 0 |
| minio | 2.14 MB/s | 455.7us | 546.0us | 433.1us | 546.2us | 712.0us | 0 |
| rustfs | 1.65 MB/s | 592.6us | 674.5us | 578.4us | 674.6us | 728.9us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.36 MB/s
minio        ██████████████ 2.14 MB/s
rustfs       ███████████ 1.65 MB/s
```

**Latency (P50)**
```
liteio       ███████████ 217.5us
minio        ██████████████████████ 433.1us
rustfs       ██████████████████████████████ 578.4us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.47 MB/s | 665.8us | 963.6us | 652.8us | 963.7us | 1.0ms | 0 |
| minio | 1.38 MB/s | 705.4us | 933.2us | 667.2us | 933.3us | 1.0ms | 0 |
| rustfs | 1.13 MB/s | 865.9us | 1.1ms | 841.2us | 1.1ms | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.47 MB/s
minio        ████████████████████████████ 1.38 MB/s
rustfs       ███████████████████████ 1.13 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 652.8us
minio        ███████████████████████ 667.2us
rustfs       ██████████████████████████████ 841.2us
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rustfs | 0.43 MB/s | 2.3ms | 3.8ms | 2.0ms | 3.8ms | 4.3ms | 0 |
| liteio | 0.42 MB/s | 2.3ms | 2.8ms | 2.2ms | 2.8ms | 3.2ms | 0 |
| minio | 0.41 MB/s | 2.4ms | 3.5ms | 2.4ms | 3.5ms | 3.9ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.43 MB/s
liteio       █████████████████████████████ 0.42 MB/s
minio        ████████████████████████████ 0.41 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████████ 2.0ms
liteio       ███████████████████████████ 2.2ms
minio        ██████████████████████████████ 2.4ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rustfs | 0.43 MB/s | 2.2ms | 3.2ms | 2.1ms | 3.2ms | 3.5ms | 0 |
| minio | 0.43 MB/s | 2.3ms | 3.3ms | 2.4ms | 3.3ms | 4.0ms | 0 |
| liteio | 0.38 MB/s | 2.5ms | 3.3ms | 2.6ms | 3.3ms | 4.1ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.43 MB/s
minio        █████████████████████████████ 0.43 MB/s
liteio       ██████████████████████████ 0.38 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████████████ 2.1ms
minio        ███████████████████████████ 2.4ms
liteio       ██████████████████████████████ 2.6ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.80 MB/s | 1.2ms | 1.7ms | 1.2ms | 1.7ms | 1.8ms | 0 |
| minio | 0.78 MB/s | 1.2ms | 1.7ms | 1.3ms | 1.7ms | 2.1ms | 0 |
| rustfs | 0.72 MB/s | 1.4ms | 1.7ms | 1.4ms | 1.7ms | 1.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.80 MB/s
minio        █████████████████████████████ 0.78 MB/s
rustfs       ███████████████████████████ 0.72 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████████ 1.2ms
minio        ███████████████████████████ 1.3ms
rustfs       ██████████████████████████████ 1.4ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.52 MB/s | 1.9ms | 3.0ms | 1.7ms | 3.0ms | 3.3ms | 0 |
| rustfs | 0.49 MB/s | 2.0ms | 2.8ms | 2.0ms | 2.8ms | 2.9ms | 0 |
| minio | 0.45 MB/s | 2.2ms | 3.3ms | 2.1ms | 3.3ms | 3.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.52 MB/s
rustfs       ████████████████████████████ 0.49 MB/s
minio        ██████████████████████████ 0.45 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████ 1.7ms
rustfs       ████████████████████████████ 2.0ms
minio        ██████████████████████████████ 2.1ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.87 MB/s | 334.4us | 393.0us | 429.2us | 0 |
| rustfs | 1.15 MB/s | 847.0us | 956.2us | 1.2ms | 0 |
| minio | 0.85 MB/s | 1.1ms | 1.4ms | 1.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.87 MB/s
rustfs       ████████████ 1.15 MB/s
minio        ████████ 0.85 MB/s
```

**Latency (P50)**
```
liteio       █████████ 334.4us
rustfs       ███████████████████████ 847.0us
minio        ██████████████████████████████ 1.1ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.75 MB/s | 1.1ms | 2.6ms | 3.7ms | 0 |
| rustfs | 0.52 MB/s | 1.8ms | 3.2ms | 3.6ms | 0 |
| minio | 0.42 MB/s | 2.2ms | 3.8ms | 4.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.75 MB/s
rustfs       ████████████████████ 0.52 MB/s
minio        ████████████████ 0.42 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.1ms
rustfs       ████████████████████████ 1.8ms
minio        ██████████████████████████████ 2.2ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.12 MB/s | 9.7ms | 12.1ms | 13.9ms | 0 |
| liteio | 0.07 MB/s | 12.6ms | 16.4ms | 17.5ms | 0 |
| minio | 0.06 MB/s | 16.5ms | 21.4ms | 22.1ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.12 MB/s
liteio       ██████████████████ 0.07 MB/s
minio        ███████████████ 0.06 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████ 9.7ms
liteio       ███████████████████████ 12.6ms
minio        ██████████████████████████████ 16.5ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.13 MB/s | 8.3ms | 11.6ms | 12.0ms | 0 |
| liteio | 0.12 MB/s | 7.0ms | 13.8ms | 14.8ms | 0 |
| minio | 0.08 MB/s | 11.7ms | 18.6ms | 19.1ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.13 MB/s
liteio       ███████████████████████████ 0.12 MB/s
minio        ███████████████████ 0.08 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████████ 8.3ms
liteio       ██████████████████ 7.0ms
minio        ██████████████████████████████ 11.7ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.32 MB/s | 2.5ms | 6.2ms | 6.3ms | 0 |
| rustfs | 0.25 MB/s | 3.7ms | 6.6ms | 8.2ms | 0 |
| minio | 0.19 MB/s | 4.6ms | 7.8ms | 9.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.32 MB/s
rustfs       ███████████████████████ 0.25 MB/s
minio        █████████████████ 0.19 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 2.5ms
rustfs       ███████████████████████ 3.7ms
minio        ██████████████████████████████ 4.6ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.18 MB/s | 4.5ms | 11.2ms | 12.8ms | 0 |
| rustfs | 0.15 MB/s | 6.3ms | 12.3ms | 13.2ms | 0 |
| minio | 0.12 MB/s | 6.3ms | 18.3ms | 18.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.18 MB/s
rustfs       ████████████████████████ 0.15 MB/s
minio        ███████████████████ 0.12 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 4.5ms
rustfs       ██████████████████████████████ 6.3ms
minio        █████████████████████████████ 6.3ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 189.22 MB/s | 1.2ms | 1.9ms | 2.9ms | 0 |
| minio | 162.61 MB/s | 1.5ms | 1.8ms | 2.3ms | 0 |
| rustfs | 123.38 MB/s | 2.0ms | 2.5ms | 3.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 189.22 MB/s
minio        █████████████████████████ 162.61 MB/s
rustfs       ███████████████████ 123.38 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 1.2ms
minio        ██████████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.0ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 206.76 MB/s | 1.2ms | 1.4ms | 1.5ms | 0 |
| minio | 165.72 MB/s | 1.4ms | 1.7ms | 3.7ms | 0 |
| rustfs | 114.33 MB/s | 1.9ms | 2.1ms | 11.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 206.76 MB/s
minio        ████████████████████████ 165.72 MB/s
rustfs       ████████████████ 114.33 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 1.2ms
minio        ██████████████████████ 1.4ms
rustfs       ██████████████████████████████ 1.9ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 205.84 MB/s | 1.1ms | 1.4ms | 3.0ms | 0 |
| minio | 140.43 MB/s | 1.6ms | 1.8ms | 2.3ms | 0 |
| rustfs | 112.40 MB/s | 1.9ms | 4.2ms | 6.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 205.84 MB/s
minio        ████████████████████ 140.43 MB/s
rustfs       ████████████████ 112.40 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 1.1ms
minio        █████████████████████████ 1.6ms
rustfs       ██████████████████████████████ 1.9ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 329.43 MB/s | 1.5ms | 1.5ms | 299.6ms | 306.4ms | 306.4ms | 0 |
| rustfs | 322.85 MB/s | 1.9ms | 2.0ms | 309.2ms | 311.4ms | 311.4ms | 0 |
| liteio | 209.38 MB/s | 4.3ms | 5.3ms | 453.5ms | 487.0ms | 487.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 329.43 MB/s
rustfs       █████████████████████████████ 322.85 MB/s
liteio       ███████████████████ 209.38 MB/s
```

**Latency (P50)**
```
minio        ███████████████████ 299.6ms
rustfs       ████████████████████ 309.2ms
liteio       ██████████████████████████████ 453.5ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 314.23 MB/s | 1.2ms | 1.5ms | 31.8ms | 32.3ms | 32.3ms | 0 |
| rustfs | 277.28 MB/s | 5.2ms | 5.6ms | 35.8ms | 36.7ms | 36.7ms | 0 |
| liteio | 235.88 MB/s | 2.7ms | 3.5ms | 41.8ms | 45.6ms | 45.6ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 314.23 MB/s
rustfs       ██████████████████████████ 277.28 MB/s
liteio       ██████████████████████ 235.88 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████ 31.8ms
rustfs       █████████████████████████ 35.8ms
liteio       ██████████████████████████████ 41.8ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.49 MB/s | 217.2us | 254.1us | 212.7us | 254.2us | 302.5us | 0 |
| minio | 2.50 MB/s | 389.6us | 508.9us | 373.9us | 509.1us | 637.8us | 0 |
| rustfs | 2.08 MB/s | 469.5us | 506.6us | 478.2us | 506.8us | 541.7us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.49 MB/s
minio        ████████████████ 2.50 MB/s
rustfs       █████████████ 2.08 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 212.7us
minio        ███████████████████████ 373.9us
rustfs       ██████████████████████████████ 478.2us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 263.25 MB/s | 458.9us | 469.6us | 3.7ms | 4.1ms | 4.1ms | 0 |
| minio | 233.73 MB/s | 1.1ms | 1.3ms | 4.2ms | 4.5ms | 4.5ms | 0 |
| rustfs | 110.29 MB/s | 6.1ms | 4.3ms | 4.4ms | 7.3ms | 7.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 263.25 MB/s
minio        ██████████████████████████ 233.73 MB/s
rustfs       ████████████ 110.29 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████████ 3.7ms
minio        ████████████████████████████ 4.2ms
rustfs       ██████████████████████████████ 4.4ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 123.27 MB/s | 284.6us | 360.7us | 462.5us | 622.0us | 669.7us | 0 |
| minio | 99.49 MB/s | 466.4us | 626.5us | 589.6us | 802.3us | 1.1ms | 0 |
| rustfs | 83.43 MB/s | 636.2us | 648.6us | 638.7us | 753.8us | 2.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 123.27 MB/s
minio        ████████████████████████ 99.49 MB/s
rustfs       ████████████████████ 83.43 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 462.5us
minio        ███████████████████████████ 589.6us
rustfs       ██████████████████████████████ 638.7us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5271 ops/s | 189.2us | 249.5us | 275.8us | 0 |
| minio | 3712 ops/s | 213.2us | 295.5us | 604.9us | 0 |
| rustfs | 3534 ops/s | 289.5us | 317.2us | 324.4us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5271 ops/s
minio        █████████████████████ 3712 ops/s
rustfs       ████████████████████ 3534 ops/s
```

**Latency (P50)**
```
liteio       ███████████████████ 189.2us
minio        ██████████████████████ 213.2us
rustfs       ██████████████████████████████ 289.5us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 196.63 MB/s | 507.6ms | 509.6ms | 509.6ms | 0 |
| rustfs | 166.45 MB/s | 622.7ms | 622.9ms | 622.9ms | 0 |
| minio | 154.26 MB/s | 649.8ms | 651.8ms | 651.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 196.63 MB/s
rustfs       █████████████████████████ 166.45 MB/s
minio        ███████████████████████ 154.26 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 507.6ms
rustfs       ████████████████████████████ 622.7ms
minio        ██████████████████████████████ 649.8ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 210.39 MB/s | 48.2ms | 49.5ms | 49.5ms | 0 |
| rustfs | 170.02 MB/s | 61.1ms | 63.4ms | 63.4ms | 0 |
| minio | 141.90 MB/s | 70.3ms | 73.5ms | 73.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 210.39 MB/s
rustfs       ████████████████████████ 170.02 MB/s
minio        ████████████████████ 141.90 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 48.2ms
rustfs       ██████████████████████████ 61.1ms
minio        ██████████████████████████████ 70.3ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.34 MB/s | 346.8us | 677.1us | 1.1ms | 0 |
| rustfs | 1.20 MB/s | 742.9us | 1.6ms | 2.0ms | 0 |
| minio | 0.74 MB/s | 1.2ms | 2.1ms | 3.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.34 MB/s
rustfs       ███████████████ 1.20 MB/s
minio        █████████ 0.74 MB/s
```

**Latency (P50)**
```
liteio       ████████ 346.8us
rustfs       ██████████████████ 742.9us
minio        ██████████████████████████████ 1.2ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 202.79 MB/s | 4.9ms | 5.1ms | 5.1ms | 0 |
| rustfs | 144.51 MB/s | 6.6ms | 8.9ms | 8.9ms | 0 |
| minio | 99.62 MB/s | 9.5ms | 12.1ms | 12.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 202.79 MB/s
rustfs       █████████████████████ 144.51 MB/s
minio        ██████████████ 99.62 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 4.9ms
rustfs       ████████████████████ 6.6ms
minio        ██████████████████████████████ 9.5ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 104.60 MB/s | 500.5us | 689.8us | 1.1ms | 0 |
| rustfs | 55.59 MB/s | 1.0ms | 1.5ms | 1.6ms | 0 |
| minio | 21.91 MB/s | 2.8ms | 3.5ms | 4.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 104.60 MB/s
rustfs       ███████████████ 55.59 MB/s
minio        ██████ 21.91 MB/s
```

**Latency (P50)**
```
liteio       █████ 500.5us
rustfs       ███████████ 1.0ms
minio        ██████████████████████████████ 2.8ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 169.5MiB / 7.653GiB | 169.5 MB | - | 2.4% | (no data) | 139MB / 2.34GB |
| minio | 428.2MiB / 7.653GiB | 428.2 MB | - | 0.0% | 1924.1 MB | 190MB / 7.9GB |
| rustfs | 660.1MiB / 7.653GiB | 660.1 MB | - | 0.1% | 1923.1 MB | 178MB / 6.31GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** liteio
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
