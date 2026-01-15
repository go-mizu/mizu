# Storage Benchmark Report

**Generated:** 2026-01-16T01:05:58+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio (won 46/51 benchmarks, 90%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio | 46 | 90% |
| 2 | minio | 3 | 6% |
| 3 | rustfs | 2 | 4% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 5.8 MB/s | 2.2x vs minio |
| Small Write (1KB) | liteio | 1.0 MB/s | +11% vs rustfs |
| Large Read (100MB) | minio | 312.1 MB/s | close |
| Large Write (100MB) | liteio | 207.2 MB/s | +24% vs minio |
| Delete | liteio | 4.9K ops/s | +64% vs minio |
| Stat | liteio | 5.1K ops/s | +22% vs minio |
| List (100 objects) | liteio | 1.3K ops/s | 3.2x vs minio |
| Copy | liteio | 1.9 MB/s | 2.0x vs minio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **liteio** | 207 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 312 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 3514 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 207.2 | 287.1 | 485.0ms | 344.1ms |
| minio | 167.2 | 312.1 | 606.3ms | 317.2ms |
| rustfs | 54.7 | 302.5 | 1.36s | 322.4ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 1040 | 5988 | 790.3us | 156.1us |
| minio | 699 | 2711 | 1.3ms | 342.0us |
| rustfs | 936 | 1767 | 893.0us | 513.8us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 5104 | 1331 | 4882 |
| minio | 4178 | 414 | 2983 |
| rustfs | 4126 | 165 | 1204 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 1.99 | 0.73 | 0.99 | 0.82 | 0.88 | 1.03 |
| minio | 0.99 | 0.22 | 0.19 | 0.22 | 0.26 | 0.22 |
| rustfs | 1.18 | 0.26 | 0.25 | 0.24 | 0.22 | 0.24 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 3.81 | 1.84 | 1.47 | 0.98 | 0.87 | 1.75 |
| minio | 2.44 | 1.06 | 0.65 | 0.61 | 0.66 | 0.78 |
| rustfs | 1.52 | 0.71 | 0.50 | 0.54 | 0.77 | 0.59 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 406.6us | 2.4ms | 23.5ms | 209.6ms | 2.11s |
| minio | 1.5ms | 9.9ms | 87.8ms | 922.9ms | 9.29s |
| rustfs | 840.5us | 7.9ms | 74.7ms | 763.3ms | 7.33s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 249.3us | 333.8us | 822.9us | 5.4ms | 184.5ms |
| minio | 454.3us | 600.2us | 1.8ms | 14.0ms | 172.3ms |
| rustfs | 819.2us | 1.5ms | 6.9ms | 58.7ms | 760.3ms |

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
| liteio | 1.91 MB/s | 475.7us | 728.8us | 728.8us | 0 |
| minio | 0.94 MB/s | 985.7us | 1.3ms | 1.3ms | 0 |
| rustfs | 0.93 MB/s | 1.1ms | 1.2ms | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.91 MB/s
minio        ██████████████ 0.94 MB/s
rustfs       ██████████████ 0.93 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 475.7us
minio        ███████████████████████████ 985.7us
rustfs       ██████████████████████████████ 1.1ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4882 ops/s | 207.5us | 235.7us | 235.7us | 0 |
| minio | 2983 ops/s | 326.4us | 378.0us | 378.0us | 0 |
| rustfs | 1204 ops/s | 822.6us | 1.1ms | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4882 ops/s
minio        ██████████████████ 2983 ops/s
rustfs       ███████ 1204 ops/s
```

**Latency (P50)**
```
liteio       ███████ 207.5us
minio        ███████████ 326.4us
rustfs       ██████████████████████████████ 822.6us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.42 MB/s | 216.6us | 252.2us | 252.2us | 0 |
| minio | 0.10 MB/s | 904.5us | 999.8us | 999.8us | 0 |
| rustfs | 0.08 MB/s | 1.1ms | 1.6ms | 1.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.42 MB/s
minio        ██████ 0.10 MB/s
rustfs       █████ 0.08 MB/s
```

**Latency (P50)**
```
liteio       █████ 216.6us
minio        ████████████████████████ 904.5us
rustfs       ██████████████████████████████ 1.1ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3317 ops/s | 284.2us | 337.5us | 337.5us | 0 |
| minio | 1015 ops/s | 931.3us | 1.2ms | 1.2ms | 0 |
| rustfs | 290 ops/s | 2.8ms | 6.0ms | 6.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3317 ops/s
minio        █████████ 1015 ops/s
rustfs       ██ 290 ops/s
```

**Latency (P50)**
```
liteio       ███ 284.2us
minio        ██████████ 931.3us
rustfs       ██████████████████████████████ 2.8ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.39 MB/s | 228.6us | 268.1us | 268.1us | 0 |
| rustfs | 0.11 MB/s | 798.3us | 1.1ms | 1.1ms | 0 |
| minio | 0.09 MB/s | 986.8us | 1.1ms | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.39 MB/s
rustfs       ████████ 0.11 MB/s
minio        ███████ 0.09 MB/s
```

**Latency (P50)**
```
liteio       ██████ 228.6us
rustfs       ████████████████████████ 798.3us
minio        ██████████████████████████████ 986.8us
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4155 ops/s | 240.7us | 240.7us | 240.7us | 0 |
| minio | 2200 ops/s | 454.6us | 454.6us | 454.6us | 0 |
| rustfs | 1524 ops/s | 656.1us | 656.1us | 656.1us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4155 ops/s
minio        ███████████████ 2200 ops/s
rustfs       ███████████ 1524 ops/s
```

**Latency (P50)**
```
liteio       ███████████ 240.7us
minio        ████████████████████ 454.6us
rustfs       ██████████████████████████████ 656.1us
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 430 ops/s | 2.3ms | 2.3ms | 2.3ms | 0 |
| minio | 269 ops/s | 3.7ms | 3.7ms | 3.7ms | 0 |
| rustfs | 123 ops/s | 8.1ms | 8.1ms | 8.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 430 ops/s
minio        ██████████████████ 269 ops/s
rustfs       ████████ 123 ops/s
```

**Latency (P50)**
```
liteio       ████████ 2.3ms
minio        █████████████ 3.7ms
rustfs       ██████████████████████████████ 8.1ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 51 ops/s | 19.8ms | 19.8ms | 19.8ms | 0 |
| minio | 27 ops/s | 36.8ms | 36.8ms | 36.8ms | 0 |
| rustfs | 13 ops/s | 75.1ms | 75.1ms | 75.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 51 ops/s
minio        ████████████████ 27 ops/s
rustfs       ███████ 13 ops/s
```

**Latency (P50)**
```
liteio       ███████ 19.8ms
minio        ██████████████ 36.8ms
rustfs       ██████████████████████████████ 75.1ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6 ops/s | 176.1ms | 176.1ms | 176.1ms | 0 |
| minio | 3 ops/s | 347.9ms | 347.9ms | 347.9ms | 0 |
| rustfs | 1 ops/s | 667.6ms | 667.6ms | 667.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 6 ops/s
minio        ███████████████ 3 ops/s
rustfs       ███████ 1 ops/s
```

**Latency (P50)**
```
liteio       ███████ 176.1ms
minio        ███████████████ 347.9ms
rustfs       ██████████████████████████████ 667.6ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1 ops/s | 1.78s | 1.78s | 1.78s | 0 |
| minio | 0 ops/s | 3.92s | 3.92s | 3.92s | 0 |
| rustfs | 0 ops/s | 8.22s | 8.22s | 8.22s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1 ops/s
minio        █████████████ 0 ops/s
rustfs       ██████ 0 ops/s
```

**Latency (P50)**
```
liteio       ██████ 1.78s
minio        ██████████████ 3.92s
rustfs       ██████████████████████████████ 8.22s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4011 ops/s | 249.3us | 249.3us | 249.3us | 0 |
| minio | 2201 ops/s | 454.3us | 454.3us | 454.3us | 0 |
| rustfs | 1221 ops/s | 819.2us | 819.2us | 819.2us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4011 ops/s
minio        ████████████████ 2201 ops/s
rustfs       █████████ 1221 ops/s
```

**Latency (P50)**
```
liteio       █████████ 249.3us
minio        ████████████████ 454.3us
rustfs       ██████████████████████████████ 819.2us
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2996 ops/s | 333.8us | 333.8us | 333.8us | 0 |
| minio | 1666 ops/s | 600.2us | 600.2us | 600.2us | 0 |
| rustfs | 662 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2996 ops/s
minio        ████████████████ 1666 ops/s
rustfs       ██████ 662 ops/s
```

**Latency (P50)**
```
liteio       ██████ 333.8us
minio        ███████████ 600.2us
rustfs       ██████████████████████████████ 1.5ms
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1215 ops/s | 822.9us | 822.9us | 822.9us | 0 |
| minio | 558 ops/s | 1.8ms | 1.8ms | 1.8ms | 0 |
| rustfs | 145 ops/s | 6.9ms | 6.9ms | 6.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1215 ops/s
minio        █████████████ 558 ops/s
rustfs       ███ 145 ops/s
```

**Latency (P50)**
```
liteio       ███ 822.9us
minio        ███████ 1.8ms
rustfs       ██████████████████████████████ 6.9ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 185 ops/s | 5.4ms | 5.4ms | 5.4ms | 0 |
| minio | 72 ops/s | 14.0ms | 14.0ms | 14.0ms | 0 |
| rustfs | 17 ops/s | 58.7ms | 58.7ms | 58.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 185 ops/s
minio        ███████████ 72 ops/s
rustfs       ██ 17 ops/s
```

**Latency (P50)**
```
liteio       ██ 5.4ms
minio        ███████ 14.0ms
rustfs       ██████████████████████████████ 58.7ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 6 ops/s | 172.3ms | 172.3ms | 172.3ms | 0 |
| liteio | 5 ops/s | 184.5ms | 184.5ms | 184.5ms | 0 |
| rustfs | 1 ops/s | 760.3ms | 760.3ms | 760.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 6 ops/s
liteio       ████████████████████████████ 5 ops/s
rustfs       ██████ 1 ops/s
```

**Latency (P50)**
```
minio        ██████ 172.3ms
liteio       ███████ 184.5ms
rustfs       ██████████████████████████████ 760.3ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.40 MB/s | 406.6us | 406.6us | 406.6us | 0 |
| rustfs | 1.16 MB/s | 840.5us | 840.5us | 840.5us | 0 |
| minio | 0.64 MB/s | 1.5ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.40 MB/s
rustfs       ██████████████ 1.16 MB/s
minio        ████████ 0.64 MB/s
```

**Latency (P50)**
```
liteio       ████████ 406.6us
rustfs       ████████████████ 840.5us
minio        ██████████████████████████████ 1.5ms
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.99 MB/s | 2.4ms | 2.4ms | 2.4ms | 0 |
| rustfs | 1.23 MB/s | 7.9ms | 7.9ms | 7.9ms | 0 |
| minio | 0.98 MB/s | 9.9ms | 9.9ms | 9.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.99 MB/s
rustfs       █████████ 1.23 MB/s
minio        ███████ 0.98 MB/s
```

**Latency (P50)**
```
liteio       ███████ 2.4ms
rustfs       ████████████████████████ 7.9ms
minio        ██████████████████████████████ 9.9ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.16 MB/s | 23.5ms | 23.5ms | 23.5ms | 0 |
| rustfs | 1.31 MB/s | 74.7ms | 74.7ms | 74.7ms | 0 |
| minio | 1.11 MB/s | 87.8ms | 87.8ms | 87.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.16 MB/s
rustfs       █████████ 1.31 MB/s
minio        ████████ 1.11 MB/s
```

**Latency (P50)**
```
liteio       ████████ 23.5ms
rustfs       █████████████████████████ 74.7ms
minio        ██████████████████████████████ 87.8ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.66 MB/s | 209.6ms | 209.6ms | 209.6ms | 0 |
| rustfs | 1.28 MB/s | 763.3ms | 763.3ms | 763.3ms | 0 |
| minio | 1.06 MB/s | 922.9ms | 922.9ms | 922.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.66 MB/s
rustfs       ████████ 1.28 MB/s
minio        ██████ 1.06 MB/s
```

**Latency (P50)**
```
liteio       ██████ 209.6ms
rustfs       ████████████████████████ 763.3ms
minio        ██████████████████████████████ 922.9ms
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.63 MB/s | 2.11s | 2.11s | 2.11s | 0 |
| rustfs | 1.33 MB/s | 7.33s | 7.33s | 7.33s | 0 |
| minio | 1.05 MB/s | 9.29s | 9.29s | 9.29s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.63 MB/s
rustfs       ████████ 1.33 MB/s
minio        ██████ 1.05 MB/s
```

**Latency (P50)**
```
liteio       ██████ 2.11s
rustfs       ███████████████████████ 7.33s
minio        ██████████████████████████████ 9.29s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1331 ops/s | 743.1us | 836.5us | 836.5us | 0 |
| minio | 414 ops/s | 1.9ms | 4.1ms | 4.1ms | 0 |
| rustfs | 165 ops/s | 6.2ms | 6.5ms | 6.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1331 ops/s
minio        █████████ 414 ops/s
rustfs       ███ 165 ops/s
```

**Latency (P50)**
```
liteio       ███ 743.1us
minio        █████████ 1.9ms
rustfs       ██████████████████████████████ 6.2ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 10.22 MB/s | 1.4ms | 2.1ms | 2.1ms | 0 |
| rustfs | 8.84 MB/s | 1.7ms | 2.2ms | 2.2ms | 0 |
| minio | 8.28 MB/s | 1.9ms | 2.4ms | 2.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 10.22 MB/s
rustfs       █████████████████████████ 8.84 MB/s
minio        ████████████████████████ 8.28 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████ 1.4ms
rustfs       ███████████████████████████ 1.7ms
minio        ██████████████████████████████ 1.9ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 7.89 MB/s | 1.9ms | 2.7ms | 2.7ms | 0 |
| liteio | 7.34 MB/s | 2.2ms | 2.4ms | 2.4ms | 0 |
| minio | 5.77 MB/s | 2.7ms | 3.0ms | 3.0ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 7.89 MB/s
liteio       ███████████████████████████ 7.34 MB/s
minio        █████████████████████ 5.77 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████████ 1.9ms
liteio       ████████████████████████ 2.2ms
minio        ██████████████████████████████ 2.7ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6.59 MB/s | 2.4ms | 2.8ms | 2.8ms | 0 |
| minio | 4.76 MB/s | 3.4ms | 4.9ms | 4.9ms | 0 |
| rustfs | 4.68 MB/s | 3.9ms | 4.6ms | 4.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 6.59 MB/s
minio        █████████████████████ 4.76 MB/s
rustfs       █████████████████████ 4.68 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 2.4ms
minio        ██████████████████████████ 3.4ms
rustfs       ██████████████████████████████ 3.9ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 163.74 MB/s | 88.8ms | 94.6ms | 94.6ms | 0 |
| minio | 162.26 MB/s | 90.7ms | 93.8ms | 93.8ms | 0 |
| liteio | 146.83 MB/s | 96.5ms | 98.8ms | 98.8ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 163.74 MB/s
minio        █████████████████████████████ 162.26 MB/s
liteio       ██████████████████████████ 146.83 MB/s
```

**Latency (P50)**
```
rustfs       ███████████████████████████ 88.8ms
minio        ████████████████████████████ 90.7ms
liteio       ██████████████████████████████ 96.5ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 3.81 MB/s | 256.3us | 289.6us | 251.0us | 289.8us | 289.8us | 0 |
| minio | 2.44 MB/s | 399.5us | 517.9us | 371.1us | 518.1us | 518.1us | 0 |
| rustfs | 1.52 MB/s | 642.9us | 735.6us | 646.9us | 736.1us | 736.1us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.81 MB/s
minio        ███████████████████ 2.44 MB/s
rustfs       ███████████ 1.52 MB/s
```

**Latency (P50)**
```
liteio       ███████████ 251.0us
minio        █████████████████ 371.1us
rustfs       ██████████████████████████████ 646.9us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.84 MB/s | 530.3us | 707.9us | 528.6us | 708.0us | 708.0us | 0 |
| minio | 1.06 MB/s | 920.6us | 1.4ms | 828.9us | 1.4ms | 1.4ms | 0 |
| rustfs | 0.71 MB/s | 1.4ms | 2.1ms | 1.3ms | 2.1ms | 2.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.84 MB/s
minio        █████████████████ 1.06 MB/s
rustfs       ███████████ 0.71 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 528.6us
minio        ███████████████████ 828.9us
rustfs       ██████████████████████████████ 1.3ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.87 MB/s | 1.1ms | 1.4ms | 1.1ms | 1.4ms | 1.4ms | 0 |
| rustfs | 0.77 MB/s | 1.3ms | 1.5ms | 1.3ms | 1.5ms | 1.5ms | 0 |
| minio | 0.66 MB/s | 1.5ms | 1.7ms | 1.4ms | 1.7ms | 1.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.87 MB/s
rustfs       ██████████████████████████ 0.77 MB/s
minio        ██████████████████████ 0.66 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████ 1.1ms
rustfs       ██████████████████████████ 1.3ms
minio        ██████████████████████████████ 1.4ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.75 MB/s | 556.4us | 730.3us | 533.3us | 730.5us | 730.5us | 0 |
| minio | 0.78 MB/s | 1.2ms | 1.6ms | 1.2ms | 1.6ms | 1.6ms | 0 |
| rustfs | 0.59 MB/s | 1.7ms | 2.0ms | 1.7ms | 2.0ms | 2.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.75 MB/s
minio        █████████████ 0.78 MB/s
rustfs       ██████████ 0.59 MB/s
```

**Latency (P50)**
```
liteio       █████████ 533.3us
minio        ██████████████████████ 1.2ms
rustfs       ██████████████████████████████ 1.7ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.47 MB/s | 662.9us | 965.8us | 616.7us | 966.0us | 966.0us | 0 |
| minio | 0.65 MB/s | 1.5ms | 1.6ms | 1.5ms | 1.6ms | 1.6ms | 0 |
| rustfs | 0.50 MB/s | 2.0ms | 2.4ms | 2.0ms | 2.4ms | 2.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.47 MB/s
minio        █████████████ 0.65 MB/s
rustfs       ██████████ 0.50 MB/s
```

**Latency (P50)**
```
liteio       █████████ 616.7us
minio        ██████████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.0ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.98 MB/s | 999.3us | 1.2ms | 1.0ms | 1.2ms | 1.2ms | 0 |
| minio | 0.61 MB/s | 1.6ms | 1.9ms | 1.6ms | 1.9ms | 1.9ms | 0 |
| rustfs | 0.54 MB/s | 1.8ms | 2.1ms | 1.9ms | 2.1ms | 2.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.98 MB/s
minio        ██████████████████ 0.61 MB/s
rustfs       ████████████████ 0.54 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 1.0ms
minio        █████████████████████████ 1.6ms
rustfs       ██████████████████████████████ 1.9ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.99 MB/s | 439.5us | 716.2us | 716.2us | 0 |
| rustfs | 1.18 MB/s | 790.5us | 1.0ms | 1.0ms | 0 |
| minio | 0.99 MB/s | 904.9us | 1.4ms | 1.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.99 MB/s
rustfs       █████████████████ 1.18 MB/s
minio        ██████████████ 0.99 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 439.5us
rustfs       ██████████████████████████ 790.5us
minio        ██████████████████████████████ 904.9us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.73 MB/s | 1.0ms | 2.5ms | 2.5ms | 0 |
| rustfs | 0.26 MB/s | 3.9ms | 5.6ms | 5.6ms | 0 |
| minio | 0.22 MB/s | 4.1ms | 6.6ms | 6.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.73 MB/s
rustfs       ██████████ 0.26 MB/s
minio        █████████ 0.22 MB/s
```

**Latency (P50)**
```
liteio       ███████ 1.0ms
rustfs       ████████████████████████████ 3.9ms
minio        ██████████████████████████████ 4.1ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.88 MB/s | 906.2us | 1.6ms | 1.6ms | 0 |
| minio | 0.26 MB/s | 3.9ms | 4.7ms | 4.7ms | 0 |
| rustfs | 0.22 MB/s | 4.5ms | 5.3ms | 5.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.88 MB/s
minio        ████████ 0.26 MB/s
rustfs       ███████ 0.22 MB/s
```

**Latency (P50)**
```
liteio       █████ 906.2us
minio        ██████████████████████████ 3.9ms
rustfs       ██████████████████████████████ 4.5ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.03 MB/s | 909.0us | 1.1ms | 1.1ms | 0 |
| rustfs | 0.24 MB/s | 3.9ms | 4.6ms | 4.6ms | 0 |
| minio | 0.22 MB/s | 3.9ms | 5.7ms | 5.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.03 MB/s
rustfs       ███████ 0.24 MB/s
minio        ██████ 0.22 MB/s
```

**Latency (P50)**
```
liteio       ██████ 909.0us
rustfs       █████████████████████████████ 3.9ms
minio        ██████████████████████████████ 3.9ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.99 MB/s | 970.8us | 1.2ms | 1.2ms | 0 |
| rustfs | 0.25 MB/s | 3.6ms | 5.4ms | 5.4ms | 0 |
| minio | 0.19 MB/s | 5.5ms | 6.4ms | 6.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.99 MB/s
rustfs       ███████ 0.25 MB/s
minio        █████ 0.19 MB/s
```

**Latency (P50)**
```
liteio       █████ 970.8us
rustfs       ███████████████████ 3.6ms
minio        ██████████████████████████████ 5.5ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.82 MB/s | 1.1ms | 1.6ms | 1.6ms | 0 |
| rustfs | 0.24 MB/s | 3.9ms | 5.1ms | 5.1ms | 0 |
| minio | 0.22 MB/s | 4.1ms | 6.2ms | 6.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.82 MB/s
rustfs       ████████ 0.24 MB/s
minio        ████████ 0.22 MB/s
```

**Latency (P50)**
```
liteio       ████████ 1.1ms
rustfs       ████████████████████████████ 3.9ms
minio        ██████████████████████████████ 4.1ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 230.53 MB/s | 1.0ms | 1.5ms | 1.5ms | 0 |
| minio | 166.15 MB/s | 1.5ms | 1.6ms | 1.6ms | 0 |
| rustfs | 109.10 MB/s | 2.1ms | 2.6ms | 2.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 230.53 MB/s
minio        █████████████████████ 166.15 MB/s
rustfs       ██████████████ 109.10 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 1.0ms
minio        ████████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.1ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 241.52 MB/s | 1.0ms | 1.1ms | 1.1ms | 0 |
| minio | 164.85 MB/s | 1.4ms | 1.8ms | 1.8ms | 0 |
| rustfs | 107.98 MB/s | 2.3ms | 2.8ms | 2.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 241.52 MB/s
minio        ████████████████████ 164.85 MB/s
rustfs       █████████████ 107.98 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 1.0ms
minio        ██████████████████ 1.4ms
rustfs       ██████████████████████████████ 2.3ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 165.94 MB/s | 1.1ms | 3.0ms | 3.0ms | 0 |
| minio | 162.74 MB/s | 1.5ms | 1.8ms | 1.8ms | 0 |
| rustfs | 101.26 MB/s | 2.4ms | 3.2ms | 3.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 165.94 MB/s
minio        █████████████████████████████ 162.74 MB/s
rustfs       ██████████████████ 101.26 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 1.1ms
minio        ███████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.4ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 312.06 MB/s | 1.6ms | 1.9ms | 317.2ms | 322.4ms | 322.4ms | 0 |
| rustfs | 302.49 MB/s | 2.1ms | 2.3ms | 322.4ms | 345.7ms | 345.7ms | 0 |
| liteio | 287.10 MB/s | 4.7ms | 5.5ms | 344.1ms | 349.0ms | 349.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 312.06 MB/s
rustfs       █████████████████████████████ 302.49 MB/s
liteio       ███████████████████████████ 287.10 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 317.2ms
rustfs       ████████████████████████████ 322.4ms
liteio       ██████████████████████████████ 344.1ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 303.92 MB/s | 1.5ms | 1.6ms | 32.9ms | 33.4ms | 33.4ms | 0 |
| rustfs | 256.72 MB/s | 7.1ms | 8.3ms | 36.9ms | 40.6ms | 40.6ms | 0 |
| liteio | 164.96 MB/s | 4.6ms | 5.8ms | 57.6ms | 69.1ms | 69.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 303.92 MB/s
rustfs       █████████████████████████ 256.72 MB/s
liteio       ████████████████ 164.96 MB/s
```

**Latency (P50)**
```
minio        █████████████████ 32.9ms
rustfs       ███████████████████ 36.9ms
liteio       ██████████████████████████████ 57.6ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 5.85 MB/s | 166.9us | 198.1us | 156.1us | 198.2us | 198.2us | 0 |
| minio | 2.65 MB/s | 368.8us | 452.3us | 342.0us | 452.4us | 452.4us | 0 |
| rustfs | 1.73 MB/s | 565.7us | 793.0us | 513.8us | 794.0us | 794.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5.85 MB/s
minio        █████████████ 2.65 MB/s
rustfs       ████████ 1.73 MB/s
```

**Latency (P50)**
```
liteio       █████████ 156.1us
minio        ███████████████████ 342.0us
rustfs       ██████████████████████████████ 513.8us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 285.77 MB/s | 405.8us | 408.3us | 3.3ms | 4.0ms | 4.0ms | 0 |
| minio | 227.05 MB/s | 1.3ms | 1.9ms | 4.2ms | 5.2ms | 5.2ms | 0 |
| rustfs | 192.23 MB/s | 2.2ms | 2.4ms | 5.2ms | 5.5ms | 5.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 285.77 MB/s
minio        ███████████████████████ 227.05 MB/s
rustfs       ████████████████████ 192.23 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 3.3ms
minio        ████████████████████████ 4.2ms
rustfs       ██████████████████████████████ 5.2ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 153.66 MB/s | 218.6us | 262.9us | 381.9us | 454.5us | 533.8us | 0 |
| minio | 94.59 MB/s | 454.4us | 594.9us | 643.7us | 793.2us | 851.7us | 0 |
| rustfs | 76.24 MB/s | 706.4us | 820.6us | 803.1us | 929.6us | 948.2us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 153.66 MB/s
minio        ██████████████████ 94.59 MB/s
rustfs       ██████████████ 76.24 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 381.9us
minio        ████████████████████████ 643.7us
rustfs       ██████████████████████████████ 803.1us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5104 ops/s | 201.5us | 219.3us | 219.3us | 0 |
| minio | 4178 ops/s | 238.9us | 307.1us | 307.1us | 0 |
| rustfs | 4126 ops/s | 234.1us | 275.2us | 275.2us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5104 ops/s
minio        ████████████████████████ 4178 ops/s
rustfs       ████████████████████████ 4126 ops/s
```

**Latency (P50)**
```
liteio       █████████████████████████ 201.5us
minio        ██████████████████████████████ 238.9us
rustfs       █████████████████████████████ 234.1us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 207.21 MB/s | 485.0ms | 490.1ms | 490.1ms | 0 |
| minio | 167.23 MB/s | 606.3ms | 614.7ms | 614.7ms | 0 |
| rustfs | 54.74 MB/s | 1.36s | 1.41s | 1.41s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 207.21 MB/s
minio        ████████████████████████ 167.23 MB/s
rustfs       ███████ 54.74 MB/s
```

**Latency (P50)**
```
liteio       ██████████ 485.0ms
minio        █████████████ 606.3ms
rustfs       ██████████████████████████████ 1.36s
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 198.41 MB/s | 48.8ms | 53.6ms | 53.6ms | 0 |
| rustfs | 162.87 MB/s | 60.1ms | 66.4ms | 66.4ms | 0 |
| minio | 159.22 MB/s | 59.7ms | 69.9ms | 69.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 198.41 MB/s
rustfs       ████████████████████████ 162.87 MB/s
minio        ████████████████████████ 159.22 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████ 48.8ms
rustfs       ██████████████████████████████ 60.1ms
minio        █████████████████████████████ 59.7ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.02 MB/s | 790.3us | 1.5ms | 1.5ms | 0 |
| rustfs | 0.91 MB/s | 893.0us | 1.6ms | 1.6ms | 0 |
| minio | 0.68 MB/s | 1.3ms | 1.8ms | 1.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.02 MB/s
rustfs       ███████████████████████████ 0.91 MB/s
minio        ████████████████████ 0.68 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 790.3us
rustfs       ████████████████████ 893.0us
minio        ██████████████████████████████ 1.3ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 190.93 MB/s | 4.9ms | 6.8ms | 6.8ms | 0 |
| rustfs | 144.28 MB/s | 6.7ms | 8.2ms | 8.2ms | 0 |
| minio | 133.70 MB/s | 7.3ms | 8.5ms | 8.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 190.93 MB/s
rustfs       ██████████████████████ 144.28 MB/s
minio        █████████████████████ 133.70 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 4.9ms
rustfs       ███████████████████████████ 6.7ms
minio        ██████████████████████████████ 7.3ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 115.67 MB/s | 520.5us | 732.6us | 750.2us | 0 |
| rustfs | 54.93 MB/s | 1.1ms | 1.4ms | 1.5ms | 0 |
| minio | 36.42 MB/s | 1.6ms | 2.3ms | 2.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 115.67 MB/s
rustfs       ██████████████ 54.93 MB/s
minio        █████████ 36.42 MB/s
```

**Latency (P50)**
```
liteio       █████████ 520.5us
rustfs       ████████████████████ 1.1ms
minio        ██████████████████████████████ 1.6ms
```

## Recommendations

- **Write-heavy workloads:** liteio
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
