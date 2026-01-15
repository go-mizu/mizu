# Storage Benchmark Report

**Generated:** 2026-01-16T00:20:15+07:00

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
| Small Read (1KB) | liteio | 4.7 MB/s | +73% vs minio |
| Small Write (1KB) | liteio | 1.5 MB/s | +16% vs rustfs |
| Large Read (100MB) | minio | 271.3 MB/s | +15% vs liteio |
| Large Write (100MB) | liteio | 200.0 MB/s | +34% vs minio |
| Delete | liteio | 5.0K ops/s | +68% vs minio |
| Stat | liteio | 5.7K ops/s | +54% vs rustfs |
| List (100 objects) | liteio | 1.3K ops/s | 2.5x vs minio |
| Copy | liteio | 2.7 MB/s | 3.0x vs rustfs |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **liteio** | 200 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 271 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 3171 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 200.0 | 235.2 | 493.9ms | 424.8ms |
| minio | 149.0 | 271.3 | 673.1ms | 368.8ms |
| rustfs | 108.8 | 156.6 | 929.9ms | 392.9ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 1525 | 4817 | 514.6us | 204.1us |
| minio | 734 | 2789 | 1.2ms | 345.5us |
| rustfs | 1309 | 2142 | 742.4us | 456.0us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 5656 | 1320 | 4991 |
| minio | 3103 | 529 | 2968 |
| rustfs | 3670 | 185 | 1428 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 2.91 | 0.54 | 0.45 | 0.24 | 0.20 | 0.23 |
| minio | 0.63 | 0.18 | 0.11 | 0.07 | 0.08 | 0.09 |
| rustfs | 1.37 | 0.35 | 0.18 | 0.10 | 0.12 | 0.11 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 3.83 | 1.28 | 0.91 | 0.52 | 0.64 | 0.54 |
| minio | 2.76 | 1.12 | 0.41 | 0.37 | 0.34 | 0.34 |
| rustfs | 1.99 | 0.77 | 0.41 | 0.24 | 0.27 | 0.30 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 444.1us | 2.5ms | 23.7ms | 216.2ms | 2.17s |
| minio | 1.6ms | 11.9ms | 102.1ms | 1.03s | 10.69s |
| rustfs | 818.4us | 5.8ms | 68.3ms | 709.8ms | 7.49s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 304.2us | 321.5us | 821.2us | 4.9ms | 172.1ms |
| minio | 531.5us | 562.2us | 1.7ms | 13.6ms | 159.0ms |
| rustfs | 839.6us | 1.1ms | 7.4ms | 58.5ms | 719.7ms |

*\* indicates errors occurred*

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 50 |
| Warmup | 5 |
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
| liteio | 2.73 MB/s | 292.8us | 714.5us | 833.4us | 0 |
| rustfs | 0.91 MB/s | 968.2us | 1.3ms | 1.4ms | 0 |
| minio | 0.87 MB/s | 942.2us | 1.8ms | 2.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.73 MB/s
rustfs       █████████ 0.91 MB/s
minio        █████████ 0.87 MB/s
```

**Latency (P50)**
```
liteio       █████████ 292.8us
rustfs       ██████████████████████████████ 968.2us
minio        █████████████████████████████ 942.2us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4991 ops/s | 195.2us | 226.2us | 245.1us | 0 |
| minio | 2968 ops/s | 324.5us | 416.1us | 447.2us | 0 |
| rustfs | 1428 ops/s | 698.2us | 949.6us | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4991 ops/s
minio        █████████████████ 2968 ops/s
rustfs       ████████ 1428 ops/s
```

**Latency (P50)**
```
liteio       ████████ 195.2us
minio        █████████████ 324.5us
rustfs       ██████████████████████████████ 698.2us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.42 MB/s | 221.4us | 243.2us | 250.6us | 0 |
| rustfs | 0.13 MB/s | 698.4us | 751.0us | 775.8us | 0 |
| minio | 0.08 MB/s | 973.5us | 1.7ms | 2.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.42 MB/s
rustfs       █████████ 0.13 MB/s
minio        █████ 0.08 MB/s
```

**Latency (P50)**
```
liteio       ██████ 221.4us
rustfs       █████████████████████ 698.4us
minio        ██████████████████████████████ 973.5us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1403 ops/s | 706.0us | 796.5us | 798.4us | 0 |
| minio | 918 ops/s | 869.2us | 1.7ms | 1.7ms | 0 |
| liteio | 717 ops/s | 1.5ms | 2.8ms | 3.0ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1403 ops/s
minio        ███████████████████ 918 ops/s
liteio       ███████████████ 717 ops/s
```

**Latency (P50)**
```
rustfs       ██████████████ 706.0us
minio        █████████████████ 869.2us
liteio       ██████████████████████████████ 1.5ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.33 MB/s | 264.0us | 398.9us | 405.6us | 0 |
| rustfs | 0.14 MB/s | 678.2us | 729.6us | 757.9us | 0 |
| minio | 0.09 MB/s | 945.4us | 1.7ms | 1.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.33 MB/s
rustfs       ████████████ 0.14 MB/s
minio        ███████ 0.09 MB/s
```

**Latency (P50)**
```
liteio       ████████ 264.0us
rustfs       █████████████████████ 678.2us
minio        ██████████████████████████████ 945.4us
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5092 ops/s | 196.4us | 196.4us | 196.4us | 0 |
| minio | 2493 ops/s | 401.1us | 401.1us | 401.1us | 0 |
| rustfs | 852 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5092 ops/s
minio        ██████████████ 2493 ops/s
rustfs       █████ 852 ops/s
```

**Latency (P50)**
```
liteio       █████ 196.4us
minio        ██████████ 401.1us
rustfs       ██████████████████████████████ 1.2ms
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 510 ops/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| minio | 282 ops/s | 3.5ms | 3.5ms | 3.5ms | 0 |
| rustfs | 112 ops/s | 8.9ms | 8.9ms | 8.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 510 ops/s
minio        ████████████████ 282 ops/s
rustfs       ██████ 112 ops/s
```

**Latency (P50)**
```
liteio       ██████ 2.0ms
minio        ███████████ 3.5ms
rustfs       ██████████████████████████████ 8.9ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 53 ops/s | 18.9ms | 18.9ms | 18.9ms | 0 |
| minio | 29 ops/s | 34.9ms | 34.9ms | 34.9ms | 0 |
| rustfs | 14 ops/s | 69.4ms | 69.4ms | 69.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 53 ops/s
minio        ████████████████ 29 ops/s
rustfs       ████████ 14 ops/s
```

**Latency (P50)**
```
liteio       ████████ 18.9ms
minio        ███████████████ 34.9ms
rustfs       ██████████████████████████████ 69.4ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6 ops/s | 179.4ms | 179.4ms | 179.4ms | 0 |
| minio | 3 ops/s | 374.9ms | 374.9ms | 374.9ms | 0 |
| rustfs | 1 ops/s | 730.7ms | 730.7ms | 730.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 6 ops/s
minio        ██████████████ 3 ops/s
rustfs       ███████ 1 ops/s
```

**Latency (P50)**
```
liteio       ███████ 179.4ms
minio        ███████████████ 374.9ms
rustfs       ██████████████████████████████ 730.7ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1 ops/s | 1.81s | 1.81s | 1.81s | 0 |
| minio | 0 ops/s | 3.53s | 3.53s | 3.53s | 0 |
| rustfs | 0 ops/s | 8.25s | 8.25s | 8.25s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1 ops/s
minio        ███████████████ 0 ops/s
rustfs       ██████ 0 ops/s
```

**Latency (P50)**
```
liteio       ██████ 1.81s
minio        ████████████ 3.53s
rustfs       ██████████████████████████████ 8.25s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3288 ops/s | 304.2us | 304.2us | 304.2us | 0 |
| minio | 1882 ops/s | 531.5us | 531.5us | 531.5us | 0 |
| rustfs | 1191 ops/s | 839.6us | 839.6us | 839.6us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3288 ops/s
minio        █████████████████ 1882 ops/s
rustfs       ██████████ 1191 ops/s
```

**Latency (P50)**
```
liteio       ██████████ 304.2us
minio        ██████████████████ 531.5us
rustfs       ██████████████████████████████ 839.6us
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3110 ops/s | 321.5us | 321.5us | 321.5us | 0 |
| minio | 1779 ops/s | 562.2us | 562.2us | 562.2us | 0 |
| rustfs | 882 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3110 ops/s
minio        █████████████████ 1779 ops/s
rustfs       ████████ 882 ops/s
```

**Latency (P50)**
```
liteio       ████████ 321.5us
minio        ██████████████ 562.2us
rustfs       ██████████████████████████████ 1.1ms
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1218 ops/s | 821.2us | 821.2us | 821.2us | 0 |
| minio | 572 ops/s | 1.7ms | 1.7ms | 1.7ms | 0 |
| rustfs | 135 ops/s | 7.4ms | 7.4ms | 7.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1218 ops/s
minio        ██████████████ 572 ops/s
rustfs       ███ 135 ops/s
```

**Latency (P50)**
```
liteio       ███ 821.2us
minio        ███████ 1.7ms
rustfs       ██████████████████████████████ 7.4ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 204 ops/s | 4.9ms | 4.9ms | 4.9ms | 0 |
| minio | 74 ops/s | 13.6ms | 13.6ms | 13.6ms | 0 |
| rustfs | 17 ops/s | 58.5ms | 58.5ms | 58.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 204 ops/s
minio        ██████████ 74 ops/s
rustfs       ██ 17 ops/s
```

**Latency (P50)**
```
liteio       ██ 4.9ms
minio        ██████ 13.6ms
rustfs       ██████████████████████████████ 58.5ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 6 ops/s | 159.0ms | 159.0ms | 159.0ms | 0 |
| liteio | 6 ops/s | 172.1ms | 172.1ms | 172.1ms | 0 |
| rustfs | 1 ops/s | 719.7ms | 719.7ms | 719.7ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 6 ops/s
liteio       ███████████████████████████ 6 ops/s
rustfs       ██████ 1 ops/s
```

**Latency (P50)**
```
minio        ██████ 159.0ms
liteio       ███████ 172.1ms
rustfs       ██████████████████████████████ 719.7ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.20 MB/s | 444.1us | 444.1us | 444.1us | 0 |
| rustfs | 1.19 MB/s | 818.4us | 818.4us | 818.4us | 0 |
| minio | 0.61 MB/s | 1.6ms | 1.6ms | 1.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.20 MB/s
rustfs       ████████████████ 1.19 MB/s
minio        ████████ 0.61 MB/s
```

**Latency (P50)**
```
liteio       ████████ 444.1us
rustfs       ███████████████ 818.4us
minio        ██████████████████████████████ 1.6ms
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.94 MB/s | 2.5ms | 2.5ms | 2.5ms | 0 |
| rustfs | 1.67 MB/s | 5.8ms | 5.8ms | 5.8ms | 0 |
| minio | 0.82 MB/s | 11.9ms | 11.9ms | 11.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.94 MB/s
rustfs       ████████████ 1.67 MB/s
minio        ██████ 0.82 MB/s
```

**Latency (P50)**
```
liteio       ██████ 2.5ms
rustfs       ██████████████ 5.8ms
minio        ██████████████████████████████ 11.9ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.11 MB/s | 23.7ms | 23.7ms | 23.7ms | 0 |
| rustfs | 1.43 MB/s | 68.3ms | 68.3ms | 68.3ms | 0 |
| minio | 0.96 MB/s | 102.1ms | 102.1ms | 102.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.11 MB/s
rustfs       ██████████ 1.43 MB/s
minio        ██████ 0.96 MB/s
```

**Latency (P50)**
```
liteio       ██████ 23.7ms
rustfs       ████████████████████ 68.3ms
minio        ██████████████████████████████ 102.1ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.52 MB/s | 216.2ms | 216.2ms | 216.2ms | 0 |
| rustfs | 1.38 MB/s | 709.8ms | 709.8ms | 709.8ms | 0 |
| minio | 0.95 MB/s | 1.03s | 1.03s | 1.03s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.52 MB/s
rustfs       █████████ 1.38 MB/s
minio        ██████ 0.95 MB/s
```

**Latency (P50)**
```
liteio       ██████ 216.2ms
rustfs       ████████████████████ 709.8ms
minio        ██████████████████████████████ 1.03s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.51 MB/s | 2.17s | 2.17s | 2.17s | 0 |
| rustfs | 1.30 MB/s | 7.49s | 7.49s | 7.49s | 0 |
| minio | 0.91 MB/s | 10.69s | 10.69s | 10.69s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.51 MB/s
rustfs       ████████ 1.30 MB/s
minio        ██████ 0.91 MB/s
```

**Latency (P50)**
```
liteio       ██████ 2.17s
rustfs       █████████████████████ 7.49s
minio        ██████████████████████████████ 10.69s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1320 ops/s | 741.9us | 869.7us | 897.0us | 0 |
| minio | 529 ops/s | 1.7ms | 2.6ms | 3.8ms | 0 |
| rustfs | 185 ops/s | 5.1ms | 6.5ms | 6.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1320 ops/s
minio        ████████████ 529 ops/s
rustfs       ████ 185 ops/s
```

**Latency (P50)**
```
liteio       ████ 741.9us
minio        ██████████ 1.7ms
rustfs       ██████████████████████████████ 5.1ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5.87 MB/s | 2.6ms | 3.7ms | 3.7ms | 0 |
| minio | 3.59 MB/s | 4.2ms | 5.6ms | 5.9ms | 0 |
| rustfs | 2.53 MB/s | 6.1ms | 8.6ms | 9.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5.87 MB/s
minio        ██████████████████ 3.59 MB/s
rustfs       ████████████ 2.53 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 2.6ms
minio        ████████████████████ 4.2ms
rustfs       ██████████████████████████████ 6.1ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.72 MB/s | 4.2ms | 4.7ms | 4.8ms | 0 |
| minio | 2.59 MB/s | 6.1ms | 6.5ms | 6.6ms | 0 |
| rustfs | 2.28 MB/s | 7.4ms | 8.9ms | 8.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.72 MB/s
minio        ████████████████████ 2.59 MB/s
rustfs       ██████████████████ 2.28 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 4.2ms
minio        ████████████████████████ 6.1ms
rustfs       ██████████████████████████████ 7.4ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.97 MB/s | 5.5ms | 6.1ms | 6.4ms | 0 |
| minio | 1.30 MB/s | 12.5ms | 16.5ms | 16.8ms | 0 |
| rustfs | 1.16 MB/s | 12.3ms | 17.7ms | 18.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.97 MB/s
minio        █████████████ 1.30 MB/s
rustfs       ███████████ 1.16 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 5.5ms
minio        ██████████████████████████████ 12.5ms
rustfs       █████████████████████████████ 12.3ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 173.50 MB/s | 83.8ms | 95.9ms | 95.9ms | 0 |
| minio | 144.16 MB/s | 102.4ms | 108.5ms | 108.5ms | 0 |
| liteio | 109.83 MB/s | 122.3ms | 164.5ms | 164.5ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 173.50 MB/s
minio        ████████████████████████ 144.16 MB/s
liteio       ██████████████████ 109.83 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████ 83.8ms
minio        █████████████████████████ 102.4ms
liteio       ██████████████████████████████ 122.3ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 3.83 MB/s | 255.1us | 316.3us | 252.4us | 316.5us | 335.0us | 0 |
| minio | 2.76 MB/s | 354.3us | 400.8us | 346.4us | 400.9us | 432.4us | 0 |
| rustfs | 1.99 MB/s | 491.2us | 530.5us | 487.4us | 530.6us | 545.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.83 MB/s
minio        █████████████████████ 2.76 MB/s
rustfs       ███████████████ 1.99 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 252.4us
minio        █████████████████████ 346.4us
rustfs       ██████████████████████████████ 487.4us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.28 MB/s | 760.3us | 1.3ms | 665.4us | 1.3ms | 1.7ms | 0 |
| minio | 1.12 MB/s | 870.1us | 1.3ms | 831.5us | 1.3ms | 1.4ms | 0 |
| rustfs | 0.77 MB/s | 1.3ms | 1.8ms | 1.2ms | 1.8ms | 2.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.28 MB/s
minio        ██████████████████████████ 1.12 MB/s
rustfs       █████████████████ 0.77 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 665.4us
minio        ████████████████████ 831.5us
rustfs       ██████████████████████████████ 1.2ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.64 MB/s | 1.5ms | 2.0ms | 1.6ms | 2.0ms | 2.0ms | 0 |
| minio | 0.34 MB/s | 2.9ms | 3.6ms | 3.2ms | 3.6ms | 3.7ms | 0 |
| rustfs | 0.27 MB/s | 3.6ms | 4.8ms | 3.8ms | 4.8ms | 4.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.64 MB/s
minio        ███████████████ 0.34 MB/s
rustfs       ████████████ 0.27 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 1.6ms
minio        █████████████████████████ 3.2ms
rustfs       ██████████████████████████████ 3.8ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.54 MB/s | 1.8ms | 2.4ms | 1.8ms | 2.4ms | 2.6ms | 0 |
| minio | 0.34 MB/s | 2.9ms | 3.3ms | 2.8ms | 3.3ms | 3.3ms | 0 |
| rustfs | 0.30 MB/s | 3.3ms | 4.4ms | 3.5ms | 4.4ms | 4.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.54 MB/s
minio        ███████████████████ 0.34 MB/s
rustfs       ████████████████ 0.30 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.8ms
minio        ████████████████████████ 2.8ms
rustfs       ██████████████████████████████ 3.5ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.91 MB/s | 1.1ms | 1.4ms | 1.1ms | 1.4ms | 1.5ms | 0 |
| minio | 0.41 MB/s | 2.4ms | 3.2ms | 2.3ms | 3.2ms | 3.4ms | 0 |
| rustfs | 0.41 MB/s | 2.4ms | 3.4ms | 2.2ms | 3.4ms | 5.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.91 MB/s
minio        █████████████ 0.41 MB/s
rustfs       █████████████ 0.41 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 1.1ms
minio        ██████████████████████████████ 2.3ms
rustfs       ███████████████████████████ 2.2ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.52 MB/s | 1.9ms | 2.3ms | 1.9ms | 2.3ms | 2.3ms | 0 |
| minio | 0.37 MB/s | 2.6ms | 3.4ms | 2.8ms | 3.4ms | 3.5ms | 0 |
| rustfs | 0.24 MB/s | 4.0ms | 5.2ms | 3.9ms | 5.2ms | 5.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.52 MB/s
minio        █████████████████████ 0.37 MB/s
rustfs       ██████████████ 0.24 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 1.9ms
minio        █████████████████████ 2.8ms
rustfs       ██████████████████████████████ 3.9ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.91 MB/s | 307.2us | 421.0us | 609.5us | 0 |
| rustfs | 1.37 MB/s | 723.5us | 808.0us | 853.1us | 0 |
| minio | 0.63 MB/s | 1.5ms | 3.0ms | 3.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.91 MB/s
rustfs       ██████████████ 1.37 MB/s
minio        ██████ 0.63 MB/s
```

**Latency (P50)**
```
liteio       ██████ 307.2us
rustfs       ██████████████ 723.5us
minio        ██████████████████████████████ 1.5ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.54 MB/s | 1.9ms | 2.8ms | 3.4ms | 0 |
| rustfs | 0.35 MB/s | 2.5ms | 4.6ms | 5.8ms | 0 |
| minio | 0.18 MB/s | 4.6ms | 12.6ms | 12.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.54 MB/s
rustfs       ███████████████████ 0.35 MB/s
minio        █████████ 0.18 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 1.9ms
rustfs       ████████████████ 2.5ms
minio        ██████████████████████████████ 4.6ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.20 MB/s | 4.3ms | 6.8ms | 7.2ms | 0 |
| rustfs | 0.12 MB/s | 8.9ms | 10.5ms | 10.5ms | 0 |
| minio | 0.08 MB/s | 13.3ms | 16.6ms | 17.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.20 MB/s
rustfs       █████████████████ 0.12 MB/s
minio        ████████████ 0.08 MB/s
```

**Latency (P50)**
```
liteio       █████████ 4.3ms
rustfs       ████████████████████ 8.9ms
minio        ██████████████████████████████ 13.3ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.23 MB/s | 4.0ms | 6.1ms | 6.6ms | 0 |
| rustfs | 0.11 MB/s | 9.2ms | 10.4ms | 10.7ms | 0 |
| minio | 0.09 MB/s | 11.2ms | 14.8ms | 15.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.23 MB/s
rustfs       ██████████████ 0.11 MB/s
minio        ███████████ 0.09 MB/s
```

**Latency (P50)**
```
liteio       ██████████ 4.0ms
rustfs       ████████████████████████ 9.2ms
minio        ██████████████████████████████ 11.2ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.45 MB/s | 1.8ms | 4.5ms | 4.6ms | 0 |
| rustfs | 0.18 MB/s | 4.7ms | 8.8ms | 10.2ms | 0 |
| minio | 0.11 MB/s | 7.8ms | 11.8ms | 12.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.45 MB/s
rustfs       ███████████ 0.18 MB/s
minio        ███████ 0.11 MB/s
```

**Latency (P50)**
```
liteio       ██████ 1.8ms
rustfs       █████████████████ 4.7ms
minio        ██████████████████████████████ 7.8ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.24 MB/s | 3.8ms | 5.9ms | 6.0ms | 0 |
| rustfs | 0.10 MB/s | 10.4ms | 11.3ms | 11.4ms | 0 |
| minio | 0.07 MB/s | 13.1ms | 17.3ms | 17.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.24 MB/s
rustfs       ████████████ 0.10 MB/s
minio        █████████ 0.07 MB/s
```

**Latency (P50)**
```
liteio       ████████ 3.8ms
rustfs       ███████████████████████ 10.4ms
minio        ██████████████████████████████ 13.1ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 235.55 MB/s | 1.1ms | 1.3ms | 1.3ms | 0 |
| minio | 156.77 MB/s | 1.5ms | 2.0ms | 2.8ms | 0 |
| rustfs | 82.95 MB/s | 2.4ms | 4.7ms | 5.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 235.55 MB/s
minio        ███████████████████ 156.77 MB/s
rustfs       ██████████ 82.95 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 1.1ms
minio        ██████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.4ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 235.29 MB/s | 1.0ms | 1.2ms | 1.3ms | 0 |
| minio | 151.07 MB/s | 1.5ms | 2.0ms | 3.1ms | 0 |
| rustfs | 85.15 MB/s | 2.4ms | 3.4ms | 6.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 235.29 MB/s
minio        ███████████████████ 151.07 MB/s
rustfs       ██████████ 85.15 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 1.0ms
minio        ███████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.4ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 206.67 MB/s | 1.0ms | 2.4ms | 2.7ms | 0 |
| minio | 162.59 MB/s | 1.5ms | 2.0ms | 2.2ms | 0 |
| rustfs | 81.80 MB/s | 2.2ms | 5.1ms | 14.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 206.67 MB/s
minio        ███████████████████████ 162.59 MB/s
rustfs       ███████████ 81.80 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 1.0ms
minio        ███████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.2ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 271.30 MB/s | 1.7ms | 1.7ms | 368.8ms | 369.0ms | 369.0ms | 0 |
| liteio | 235.18 MB/s | 3.7ms | 4.1ms | 424.8ms | 426.2ms | 426.2ms | 0 |
| rustfs | 156.58 MB/s | 244.9ms | 4.1ms | 392.9ms | 490.6ms | 490.6ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 271.30 MB/s
liteio       ██████████████████████████ 235.18 MB/s
rustfs       █████████████████ 156.58 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 368.8ms
liteio       ██████████████████████████████ 424.8ms
rustfs       ███████████████████████████ 392.9ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 262.17 MB/s | 1.4ms | 1.7ms | 34.0ms | 41.7ms | 41.7ms | 0 |
| liteio | 229.65 MB/s | 2.8ms | 4.4ms | 40.9ms | 50.3ms | 50.3ms | 0 |
| rustfs | 224.54 MB/s | 8.2ms | 15.3ms | 42.1ms | 51.1ms | 51.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 262.17 MB/s
liteio       ██████████████████████████ 229.65 MB/s
rustfs       █████████████████████████ 224.54 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████ 34.0ms
liteio       █████████████████████████████ 40.9ms
rustfs       ██████████████████████████████ 42.1ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.70 MB/s | 207.5us | 228.2us | 204.1us | 228.2us | 252.4us | 0 |
| minio | 2.72 MB/s | 358.5us | 423.3us | 345.5us | 423.4us | 442.5us | 0 |
| rustfs | 2.09 MB/s | 466.8us | 526.5us | 456.0us | 526.6us | 540.4us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.70 MB/s
minio        █████████████████ 2.72 MB/s
rustfs       █████████████ 2.09 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 204.1us
minio        ██████████████████████ 345.5us
rustfs       ██████████████████████████████ 456.0us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 272.35 MB/s | 387.8us | 590.3us | 3.6ms | 4.2ms | 4.2ms | 0 |
| minio | 221.30 MB/s | 1.3ms | 1.6ms | 4.4ms | 5.2ms | 5.2ms | 0 |
| rustfs | 173.16 MB/s | 2.8ms | 8.4ms | 4.8ms | 11.2ms | 11.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 272.35 MB/s
minio        ████████████████████████ 221.30 MB/s
rustfs       ███████████████████ 173.16 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████ 3.6ms
minio        ███████████████████████████ 4.4ms
rustfs       ██████████████████████████████ 4.8ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 146.99 MB/s | 247.9us | 304.0us | 416.5us | 508.5us | 587.6us | 0 |
| minio | 88.03 MB/s | 521.8us | 788.5us | 653.8us | 965.2us | 1.1ms | 0 |
| rustfs | 74.04 MB/s | 713.2us | 1.3ms | 712.3us | 1.4ms | 2.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 146.99 MB/s
minio        █████████████████ 88.03 MB/s
rustfs       ███████████████ 74.04 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 416.5us
minio        ███████████████████████████ 653.8us
rustfs       ██████████████████████████████ 712.3us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5656 ops/s | 168.4us | 224.3us | 240.5us | 0 |
| rustfs | 3670 ops/s | 251.1us | 396.3us | 429.7us | 0 |
| minio | 3103 ops/s | 317.3us | 380.4us | 392.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5656 ops/s
rustfs       ███████████████████ 3670 ops/s
minio        ████████████████ 3103 ops/s
```

**Latency (P50)**
```
liteio       ███████████████ 168.4us
rustfs       ███████████████████████ 251.1us
minio        ██████████████████████████████ 317.3us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 200.03 MB/s | 493.9ms | 516.6ms | 516.6ms | 0 |
| minio | 149.05 MB/s | 673.1ms | 679.3ms | 679.3ms | 0 |
| rustfs | 108.76 MB/s | 929.9ms | 1.08s | 1.08s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 200.03 MB/s
minio        ██████████████████████ 149.05 MB/s
rustfs       ████████████████ 108.76 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 493.9ms
minio        █████████████████████ 673.1ms
rustfs       ██████████████████████████████ 929.9ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 205.86 MB/s | 48.1ms | 50.6ms | 50.6ms | 0 |
| rustfs | 173.86 MB/s | 57.7ms | 61.2ms | 61.2ms | 0 |
| minio | 148.74 MB/s | 66.1ms | 70.0ms | 70.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 205.86 MB/s
rustfs       █████████████████████████ 173.86 MB/s
minio        █████████████████████ 148.74 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 48.1ms
rustfs       ██████████████████████████ 57.7ms
minio        ██████████████████████████████ 66.1ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.49 MB/s | 514.6us | 925.6us | 2.0ms | 0 |
| rustfs | 1.28 MB/s | 742.4us | 875.1us | 971.0us | 0 |
| minio | 0.72 MB/s | 1.2ms | 1.9ms | 2.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.49 MB/s
rustfs       █████████████████████████ 1.28 MB/s
minio        ██████████████ 0.72 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 514.6us
rustfs       ██████████████████ 742.4us
minio        ██████████████████████████████ 1.2ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 188.82 MB/s | 5.1ms | 6.6ms | 6.6ms | 0 |
| rustfs | 156.09 MB/s | 6.0ms | 7.8ms | 7.8ms | 0 |
| minio | 106.42 MB/s | 9.1ms | 11.4ms | 11.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 188.82 MB/s
rustfs       ████████████████████████ 156.09 MB/s
minio        ████████████████ 106.42 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 5.1ms
rustfs       ███████████████████ 6.0ms
minio        ██████████████████████████████ 9.1ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 126.93 MB/s | 474.6us | 570.6us | 635.5us | 0 |
| rustfs | 45.04 MB/s | 1.2ms | 2.4ms | 3.0ms | 0 |
| minio | 36.81 MB/s | 1.6ms | 2.2ms | 2.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 126.93 MB/s
rustfs       ██████████ 45.04 MB/s
minio        ████████ 36.81 MB/s
```

**Latency (P50)**
```
liteio       ████████ 474.6us
rustfs       █████████████████████ 1.2ms
minio        ██████████████████████████████ 1.6ms
```

## Recommendations

- **Write-heavy workloads:** liteio
- **Read-heavy workloads:** liteio

---

*Generated by storage benchmark CLI*
