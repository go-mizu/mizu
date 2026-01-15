# Storage Benchmark Report

**Generated:** 2026-01-15T11:10:27+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** rustfs (won 16/51 benchmarks, 31%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | rustfs | 16 | 31% |
| 2 | liteio | 13 | 25% |
| 3 | liteio_mem | 13 | 25% |
| 4 | seaweedfs | 5 | 10% |
| 5 | minio | 3 | 6% |
| 6 | localstack | 1 | 2% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio_mem | 4.5 MB/s | +11% vs liteio |
| Small Write (1KB) | liteio_mem | 1.8 MB/s | +26% vs rustfs |
| Large Read (10MB) | minio | 308.4 MB/s | close |
| Large Write (10MB) | rustfs | 191.3 MB/s | close |
| Delete | liteio_mem | 6.4K ops/s | close |
| Stat | liteio_mem | 5.4K ops/s | close |
| List (100 objects) | liteio_mem | 1.3K ops/s | close |
| Copy | localstack | 1.2 MB/s | close |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **rustfs** | 199 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **liteio** | 297 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio_mem** | 3232 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **rustfs** | - | Best for multi-user apps |
| Memory Constrained | **liteio_mem** | 68 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 154.2 | 297.4 | 617.5ms | 346.2ms |
| liteio_mem | 170.5 | 276.8 | 587.2ms | 362.2ms |
| localstack | 150.0 | 270.9 | 662.0ms | 367.5ms |
| minio | 164.5 | 297.2 | 611.3ms | 336.1ms |
| rustfs | 199.0 | 292.8 | 488.5ms | 340.8ms |
| seaweedfs | 194.5 | 277.6 | 504.2ms | 359.2ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 1101 | 4142 | 737.0us | 212.7us |
| liteio_mem | 1870 | 4595 | 482.8us | 216.9us |
| localstack | 1348 | 1452 | 711.2us | 668.9us |
| minio | 1175 | 3043 | 813.4us | 322.5us |
| rustfs | 1489 | 2191 | 652.3us | 447.6us |
| seaweedfs | 1111 | 2173 | 781.4us | 455.4us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 5085 | 1233 | 5888 |
| liteio_mem | 5405 | 1281 | 6398 |
| localstack | 1494 | 378 | 1498 |
| minio | 4015 | 631 | 3007 |
| rustfs | 3526 | 165 | 1279 |
| seaweedfs | 3531 | 626 | 4193 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 1.17 | 0.36 | 0.19 | 0.10 | 0.05 | 0.07 |
| liteio_mem | 1.10 | 0.32 | 0.20 | 0.10 | 0.07 | 0.08 |
| localstack | 1.09 | 0.18 | 0.07 | 0.03 | 0.02 | 0.02 |
| minio | 1.08 | 0.37 | 0.16 | 0.11 | 0.05 | 0.06 |
| rustfs | 1.42 | 0.52 | 0.23 | 0.08 | 0.07 | 0.09 |
| seaweedfs | 1.46 | 0.44 | 0.23 | 0.08 | 0.07 | 0.09 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 3.57 | 0.84 | 0.78 | 0.46 | 0.31 | 0.28 |
| liteio_mem | 3.54 | 0.91 | 0.84 | 0.42 | 0.33 | 0.28 |
| localstack | 1.13 | 0.17 | 0.07 | 0.03 | 0.02 | 0.01 |
| minio | 2.87 | 1.00 | 0.55 | 0.30 | 0.20 | 0.17 |
| rustfs | 1.58 | 1.00 | 0.54 | 0.25 | 0.17 | 0.17 |
| seaweedfs | 2.65 | 0.89 | 0.44 | 0.15 | 0.21 | 0.20 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 774.1us | 7.8ms | 81.6ms | 879.8ms | 6.53s |
| liteio_mem | 1.5ms | 14.6ms | 174.6ms | 1.63s | 6.94s |
| localstack | 870.5us | 8.8ms | 95.6ms | 802.8ms | 8.14s |
| minio | 1.0ms | 12.2ms | 94.0ms | 923.7ms | 8.54s |
| rustfs | 757.2us | 5.3ms | 61.7ms | 541.6ms | 6.45s |
| seaweedfs | 823.2us | 7.0ms | 80.5ms | 696.0ms | 7.05s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 273.8us | 348.7us | 964.5us | 5.7ms | 182.8ms |
| liteio_mem | 337.5us | 373.2us | 1.6ms | 6.6ms | 184.0ms |
| localstack | 1.1ms | 3.8ms | 3.9ms | 19.9ms | 290.0ms |
| minio | 446.2us | 695.3us | 1.8ms | 13.2ms | 162.3ms |
| rustfs | 1.1ms | 1.6ms | 5.5ms | 39.4ms | 747.7ms |
| seaweedfs | 752.5us | 753.0us | 1.6ms | 11.0ms | 94.3ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 79.2 MB | 0.0% |
| liteio_mem | 68.3 MB | 0.0% |
| localstack | 364.6 MB | 0.1% |
| minio | 389.9 MB | 0.0% |
| rustfs | 500.7 MB | 0.1% |
| seaweedfs | 106.8 MB | 0.0% |

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
- **liteio_mem** (51 benchmarks)
- **localstack** (51 benchmarks)
- **minio** (51 benchmarks)
- **rustfs** (51 benchmarks)
- **seaweedfs** (51 benchmarks)

*Reference baseline: devnull (excluded from comparisons)*

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| localstack | 1.25 MB/s | 743.2us | 1.1ms | 1.3ms | 0 |
| liteio_mem | 1.23 MB/s | 604.1us | 1.7ms | 2.0ms | 0 |
| liteio | 1.18 MB/s | 613.2us | 1.7ms | 1.8ms | 0 |
| minio | 1.11 MB/s | 837.9us | 1.1ms | 1.5ms | 0 |
| rustfs | 1.10 MB/s | 891.8us | 1.1ms | 1.3ms | 0 |
| seaweedfs | 0.25 MB/s | 1.1ms | 1.6ms | 2.2ms | 0 |

**Throughput**
```
localstack   ██████████████████████████████ 1.25 MB/s
liteio_mem   █████████████████████████████ 1.23 MB/s
liteio       ████████████████████████████ 1.18 MB/s
minio        ██████████████████████████ 1.11 MB/s
rustfs       ██████████████████████████ 1.10 MB/s
seaweedfs    █████ 0.25 MB/s
```

**Latency (P50)**
```
localstack   ████████████████████ 743.2us
liteio_mem   ████████████████ 604.1us
liteio       █████████████████ 613.2us
minio        ███████████████████████ 837.9us
rustfs       █████████████████████████ 891.8us
seaweedfs    ██████████████████████████████ 1.1ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 6398 ops/s | 150.3us | 189.8us | 230.8us | 0 |
| liteio | 5888 ops/s | 155.2us | 206.3us | 316.0us | 0 |
| seaweedfs | 4193 ops/s | 233.6us | 278.9us | 325.6us | 0 |
| minio | 3007 ops/s | 328.9us | 365.5us | 450.8us | 0 |
| localstack | 1498 ops/s | 659.4us | 713.6us | 793.0us | 0 |
| rustfs | 1279 ops/s | 775.9us | 855.7us | 1.0ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 6398 ops/s
liteio       ███████████████████████████ 5888 ops/s
seaweedfs    ███████████████████ 4193 ops/s
minio        ██████████████ 3007 ops/s
localstack   ███████ 1498 ops/s
rustfs       █████ 1279 ops/s
```

**Latency (P50)**
```
liteio_mem   █████ 150.3us
liteio       █████ 155.2us
seaweedfs    █████████ 233.6us
minio        ████████████ 328.9us
localstack   █████████████████████████ 659.4us
rustfs       ██████████████████████████████ 775.9us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.14 MB/s | 639.9us | 1.2ms | 1.4ms | 0 |
| seaweedfs | 0.13 MB/s | 661.1us | 1.1ms | 1.1ms | 0 |
| liteio | 0.13 MB/s | 618.6us | 1.6ms | 1.8ms | 0 |
| localstack | 0.11 MB/s | 782.4us | 1.4ms | 1.7ms | 0 |
| minio | 0.10 MB/s | 890.4us | 1.1ms | 1.3ms | 0 |
| liteio_mem | 0.07 MB/s | 1.5ms | 1.7ms | 1.8ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.14 MB/s
seaweedfs    ████████████████████████████ 0.13 MB/s
liteio       ████████████████████████████ 0.13 MB/s
localstack   ██████████████████████ 0.11 MB/s
minio        ██████████████████████ 0.10 MB/s
liteio_mem   ██████████████ 0.07 MB/s
```

**Latency (P50)**
```
rustfs       █████████████ 639.9us
seaweedfs    █████████████ 661.1us
liteio       ████████████ 618.6us
localstack   ████████████████ 782.4us
minio        ██████████████████ 890.4us
liteio_mem   ██████████████████████████████ 1.5ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1549 ops/s | 634.3us | 703.7us | 776.0us | 0 |
| minio | 1208 ops/s | 784.4us | 1.1ms | 1.1ms | 0 |
| seaweedfs | 1183 ops/s | 606.3us | 1.9ms | 2.2ms | 0 |
| liteio | 1020 ops/s | 600.3us | 1.7ms | 1.8ms | 0 |
| localstack | 892 ops/s | 1.0ms | 1.5ms | 1.8ms | 0 |
| liteio_mem | 477 ops/s | 2.1ms | 2.3ms | 2.3ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1549 ops/s
minio        ███████████████████████ 1208 ops/s
seaweedfs    ██████████████████████ 1183 ops/s
liteio       ███████████████████ 1020 ops/s
localstack   █████████████████ 892 ops/s
liteio_mem   █████████ 477 ops/s
```

**Latency (P50)**
```
rustfs       █████████ 634.3us
minio        ███████████ 784.4us
seaweedfs    ████████ 606.3us
liteio       ████████ 600.3us
localstack   ██████████████ 1.0ms
liteio_mem   ██████████████████████████████ 2.1ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.14 MB/s | 616.9us | 998.5us | 1.7ms | 0 |
| liteio | 0.12 MB/s | 620.7us | 1.7ms | 1.8ms | 0 |
| seaweedfs | 0.10 MB/s | 761.8us | 1.5ms | 2.9ms | 0 |
| localstack | 0.09 MB/s | 875.5us | 1.6ms | 1.7ms | 0 |
| minio | 0.07 MB/s | 1.4ms | 1.6ms | 1.9ms | 0 |
| liteio_mem | 0.04 MB/s | 1.6ms | 5.5ms | 5.9ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.14 MB/s
liteio       █████████████████████████ 0.12 MB/s
seaweedfs    █████████████████████ 0.10 MB/s
localstack   ████████████████████ 0.09 MB/s
minio        ██████████████ 0.07 MB/s
liteio_mem   █████████ 0.04 MB/s
```

**Latency (P50)**
```
rustfs       ███████████ 616.9us
liteio       ███████████ 620.7us
seaweedfs    ██████████████ 761.8us
localstack   ████████████████ 875.5us
minio        █████████████████████████ 1.4ms
liteio_mem   ██████████████████████████████ 1.6ms
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4468 ops/s | 223.8us | 223.8us | 223.8us | 0 |
| liteio_mem | 3313 ops/s | 301.9us | 301.9us | 301.9us | 0 |
| seaweedfs | 2460 ops/s | 406.5us | 406.5us | 406.5us | 0 |
| minio | 2321 ops/s | 430.8us | 430.8us | 430.8us | 0 |
| localstack | 1408 ops/s | 710.4us | 710.4us | 710.4us | 0 |
| rustfs | 1384 ops/s | 722.5us | 722.5us | 722.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4468 ops/s
liteio_mem   ██████████████████████ 3313 ops/s
seaweedfs    ████████████████ 2460 ops/s
minio        ███████████████ 2321 ops/s
localstack   █████████ 1408 ops/s
rustfs       █████████ 1384 ops/s
```

**Latency (P50)**
```
liteio       █████████ 223.8us
liteio_mem   ████████████ 301.9us
seaweedfs    ████████████████ 406.5us
minio        █████████████████ 430.8us
localstack   █████████████████████████████ 710.4us
rustfs       ██████████████████████████████ 722.5us
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 496 ops/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| liteio_mem | 342 ops/s | 2.9ms | 2.9ms | 2.9ms | 0 |
| seaweedfs | 297 ops/s | 3.4ms | 3.4ms | 3.4ms | 0 |
| minio | 283 ops/s | 3.5ms | 3.5ms | 3.5ms | 0 |
| localstack | 152 ops/s | 6.6ms | 6.6ms | 6.6ms | 0 |
| rustfs | 147 ops/s | 6.8ms | 6.8ms | 6.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 496 ops/s
liteio_mem   ████████████████████ 342 ops/s
seaweedfs    █████████████████ 297 ops/s
minio        █████████████████ 283 ops/s
localstack   █████████ 152 ops/s
rustfs       ████████ 147 ops/s
```

**Latency (P50)**
```
liteio       ████████ 2.0ms
liteio_mem   ████████████ 2.9ms
seaweedfs    ██████████████ 3.4ms
minio        ███████████████ 3.5ms
localstack   █████████████████████████████ 6.6ms
rustfs       ██████████████████████████████ 6.8ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 47 ops/s | 21.2ms | 21.2ms | 21.2ms | 0 |
| liteio | 45 ops/s | 22.0ms | 22.0ms | 22.0ms | 0 |
| seaweedfs | 32 ops/s | 31.7ms | 31.7ms | 31.7ms | 0 |
| minio | 28 ops/s | 35.5ms | 35.5ms | 35.5ms | 0 |
| localstack | 15 ops/s | 65.1ms | 65.1ms | 65.1ms | 0 |
| rustfs | 13 ops/s | 77.2ms | 77.2ms | 77.2ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 47 ops/s
liteio       ████████████████████████████ 45 ops/s
seaweedfs    ████████████████████ 32 ops/s
minio        █████████████████ 28 ops/s
localstack   █████████ 15 ops/s
rustfs       ████████ 13 ops/s
```

**Latency (P50)**
```
liteio_mem   ████████ 21.2ms
liteio       ████████ 22.0ms
seaweedfs    ████████████ 31.7ms
minio        █████████████ 35.5ms
localstack   █████████████████████████ 65.1ms
rustfs       ██████████████████████████████ 77.2ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 5 ops/s | 186.0ms | 186.0ms | 186.0ms | 0 |
| liteio | 5 ops/s | 188.7ms | 188.7ms | 188.7ms | 0 |
| seaweedfs | 3 ops/s | 314.5ms | 314.5ms | 314.5ms | 0 |
| minio | 3 ops/s | 371.7ms | 371.7ms | 371.7ms | 0 |
| rustfs | 2 ops/s | 632.2ms | 632.2ms | 632.2ms | 0 |
| localstack | 1 ops/s | 669.3ms | 669.3ms | 669.3ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 5 ops/s
liteio       █████████████████████████████ 5 ops/s
seaweedfs    █████████████████ 3 ops/s
minio        ███████████████ 3 ops/s
rustfs       ████████ 2 ops/s
localstack   ████████ 1 ops/s
```

**Latency (P50)**
```
liteio_mem   ████████ 186.0ms
liteio       ████████ 188.7ms
seaweedfs    ██████████████ 314.5ms
minio        ████████████████ 371.7ms
rustfs       ████████████████████████████ 632.2ms
localstack   ██████████████████████████████ 669.3ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1 ops/s | 1.87s | 1.87s | 1.87s | 0 |
| liteio | 1 ops/s | 1.88s | 1.88s | 1.88s | 0 |
| seaweedfs | 0 ops/s | 3.21s | 3.21s | 3.21s | 0 |
| minio | 0 ops/s | 3.62s | 3.62s | 3.62s | 0 |
| localstack | 0 ops/s | 6.28s | 6.28s | 6.28s | 0 |
| rustfs | 0 ops/s | 8.14s | 8.14s | 8.14s | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1 ops/s
liteio       █████████████████████████████ 1 ops/s
seaweedfs    █████████████████ 0 ops/s
minio        ███████████████ 0 ops/s
localstack   ████████ 0 ops/s
rustfs       ██████ 0 ops/s
```

**Latency (P50)**
```
liteio_mem   ██████ 1.87s
liteio       ██████ 1.88s
seaweedfs    ███████████ 3.21s
minio        █████████████ 3.62s
localstack   ███████████████████████ 6.28s
rustfs       ██████████████████████████████ 8.14s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3652 ops/s | 273.8us | 273.8us | 273.8us | 0 |
| liteio_mem | 2963 ops/s | 337.5us | 337.5us | 337.5us | 0 |
| minio | 2241 ops/s | 446.2us | 446.2us | 446.2us | 0 |
| seaweedfs | 1329 ops/s | 752.5us | 752.5us | 752.5us | 0 |
| rustfs | 904 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| localstack | 899 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3652 ops/s
liteio_mem   ████████████████████████ 2963 ops/s
minio        ██████████████████ 2241 ops/s
seaweedfs    ██████████ 1329 ops/s
rustfs       ███████ 904 ops/s
localstack   ███████ 899 ops/s
```

**Latency (P50)**
```
liteio       ███████ 273.8us
liteio_mem   █████████ 337.5us
minio        ████████████ 446.2us
seaweedfs    ████████████████████ 752.5us
rustfs       █████████████████████████████ 1.1ms
localstack   ██████████████████████████████ 1.1ms
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2868 ops/s | 348.7us | 348.7us | 348.7us | 0 |
| liteio_mem | 2679 ops/s | 373.2us | 373.2us | 373.2us | 0 |
| minio | 1438 ops/s | 695.3us | 695.3us | 695.3us | 0 |
| seaweedfs | 1328 ops/s | 753.0us | 753.0us | 753.0us | 0 |
| rustfs | 625 ops/s | 1.6ms | 1.6ms | 1.6ms | 0 |
| localstack | 262 ops/s | 3.8ms | 3.8ms | 3.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2868 ops/s
liteio_mem   ████████████████████████████ 2679 ops/s
minio        ███████████████ 1438 ops/s
seaweedfs    █████████████ 1328 ops/s
rustfs       ██████ 625 ops/s
localstack   ██ 262 ops/s
```

**Latency (P50)**
```
liteio       ██ 348.7us
liteio_mem   ██ 373.2us
minio        █████ 695.3us
seaweedfs    █████ 753.0us
rustfs       ████████████ 1.6ms
localstack   ██████████████████████████████ 3.8ms
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1037 ops/s | 964.5us | 964.5us | 964.5us | 0 |
| seaweedfs | 639 ops/s | 1.6ms | 1.6ms | 1.6ms | 0 |
| liteio_mem | 635 ops/s | 1.6ms | 1.6ms | 1.6ms | 0 |
| minio | 554 ops/s | 1.8ms | 1.8ms | 1.8ms | 0 |
| localstack | 257 ops/s | 3.9ms | 3.9ms | 3.9ms | 0 |
| rustfs | 183 ops/s | 5.5ms | 5.5ms | 5.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1037 ops/s
seaweedfs    ██████████████████ 639 ops/s
liteio_mem   ██████████████████ 635 ops/s
minio        ████████████████ 554 ops/s
localstack   ███████ 257 ops/s
rustfs       █████ 183 ops/s
```

**Latency (P50)**
```
liteio       █████ 964.5us
seaweedfs    ████████ 1.6ms
liteio_mem   ████████ 1.6ms
minio        █████████ 1.8ms
localstack   █████████████████████ 3.9ms
rustfs       ██████████████████████████████ 5.5ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 175 ops/s | 5.7ms | 5.7ms | 5.7ms | 0 |
| liteio_mem | 152 ops/s | 6.6ms | 6.6ms | 6.6ms | 0 |
| seaweedfs | 91 ops/s | 11.0ms | 11.0ms | 11.0ms | 0 |
| minio | 76 ops/s | 13.2ms | 13.2ms | 13.2ms | 0 |
| localstack | 50 ops/s | 19.9ms | 19.9ms | 19.9ms | 0 |
| rustfs | 25 ops/s | 39.4ms | 39.4ms | 39.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 175 ops/s
liteio_mem   ██████████████████████████ 152 ops/s
seaweedfs    ███████████████ 91 ops/s
minio        ████████████ 76 ops/s
localstack   ████████ 50 ops/s
rustfs       ████ 25 ops/s
```

**Latency (P50)**
```
liteio       ████ 5.7ms
liteio_mem   █████ 6.6ms
seaweedfs    ████████ 11.0ms
minio        ██████████ 13.2ms
localstack   ███████████████ 19.9ms
rustfs       ██████████████████████████████ 39.4ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 11 ops/s | 94.3ms | 94.3ms | 94.3ms | 0 |
| minio | 6 ops/s | 162.3ms | 162.3ms | 162.3ms | 0 |
| liteio | 5 ops/s | 182.8ms | 182.8ms | 182.8ms | 0 |
| liteio_mem | 5 ops/s | 184.0ms | 184.0ms | 184.0ms | 0 |
| localstack | 3 ops/s | 290.0ms | 290.0ms | 290.0ms | 0 |
| rustfs | 1 ops/s | 747.7ms | 747.7ms | 747.7ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 11 ops/s
minio        █████████████████ 6 ops/s
liteio       ███████████████ 5 ops/s
liteio_mem   ███████████████ 5 ops/s
localstack   █████████ 3 ops/s
rustfs       ███ 1 ops/s
```

**Latency (P50)**
```
seaweedfs    ███ 94.3ms
minio        ██████ 162.3ms
liteio       ███████ 182.8ms
liteio_mem   ███████ 184.0ms
localstack   ███████████ 290.0ms
rustfs       ██████████████████████████████ 747.7ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.29 MB/s | 757.2us | 757.2us | 757.2us | 0 |
| liteio | 1.26 MB/s | 774.1us | 774.1us | 774.1us | 0 |
| seaweedfs | 1.19 MB/s | 823.2us | 823.2us | 823.2us | 0 |
| localstack | 1.12 MB/s | 870.5us | 870.5us | 870.5us | 0 |
| minio | 0.94 MB/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| liteio_mem | 0.67 MB/s | 1.5ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.29 MB/s
liteio       █████████████████████████████ 1.26 MB/s
seaweedfs    ███████████████████████████ 1.19 MB/s
localstack   ██████████████████████████ 1.12 MB/s
minio        █████████████████████ 0.94 MB/s
liteio_mem   ███████████████ 0.67 MB/s
```

**Latency (P50)**
```
rustfs       ███████████████ 757.2us
liteio       ███████████████ 774.1us
seaweedfs    ████████████████ 823.2us
localstack   █████████████████ 870.5us
minio        █████████████████████ 1.0ms
liteio_mem   ██████████████████████████████ 1.5ms
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.84 MB/s | 5.3ms | 5.3ms | 5.3ms | 0 |
| seaweedfs | 1.39 MB/s | 7.0ms | 7.0ms | 7.0ms | 0 |
| liteio | 1.25 MB/s | 7.8ms | 7.8ms | 7.8ms | 0 |
| localstack | 1.11 MB/s | 8.8ms | 8.8ms | 8.8ms | 0 |
| minio | 0.80 MB/s | 12.2ms | 12.2ms | 12.2ms | 0 |
| liteio_mem | 0.67 MB/s | 14.6ms | 14.6ms | 14.6ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.84 MB/s
seaweedfs    ██████████████████████ 1.39 MB/s
liteio       ████████████████████ 1.25 MB/s
localstack   ██████████████████ 1.11 MB/s
minio        █████████████ 0.80 MB/s
liteio_mem   ██████████ 0.67 MB/s
```

**Latency (P50)**
```
rustfs       ██████████ 5.3ms
seaweedfs    ██████████████ 7.0ms
liteio       ████████████████ 7.8ms
localstack   ██████████████████ 8.8ms
minio        █████████████████████████ 12.2ms
liteio_mem   ██████████████████████████████ 14.6ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.58 MB/s | 61.7ms | 61.7ms | 61.7ms | 0 |
| seaweedfs | 1.21 MB/s | 80.5ms | 80.5ms | 80.5ms | 0 |
| liteio | 1.20 MB/s | 81.6ms | 81.6ms | 81.6ms | 0 |
| minio | 1.04 MB/s | 94.0ms | 94.0ms | 94.0ms | 0 |
| localstack | 1.02 MB/s | 95.6ms | 95.6ms | 95.6ms | 0 |
| liteio_mem | 0.56 MB/s | 174.6ms | 174.6ms | 174.6ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.58 MB/s
seaweedfs    ██████████████████████ 1.21 MB/s
liteio       ██████████████████████ 1.20 MB/s
minio        ███████████████████ 1.04 MB/s
localstack   ███████████████████ 1.02 MB/s
liteio_mem   ██████████ 0.56 MB/s
```

**Latency (P50)**
```
rustfs       ██████████ 61.7ms
seaweedfs    █████████████ 80.5ms
liteio       ██████████████ 81.6ms
minio        ████████████████ 94.0ms
localstack   ████████████████ 95.6ms
liteio_mem   ██████████████████████████████ 174.6ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.80 MB/s | 541.6ms | 541.6ms | 541.6ms | 0 |
| seaweedfs | 1.40 MB/s | 696.0ms | 696.0ms | 696.0ms | 0 |
| localstack | 1.22 MB/s | 802.8ms | 802.8ms | 802.8ms | 0 |
| liteio | 1.11 MB/s | 879.8ms | 879.8ms | 879.8ms | 0 |
| minio | 1.06 MB/s | 923.7ms | 923.7ms | 923.7ms | 0 |
| liteio_mem | 0.60 MB/s | 1.63s | 1.63s | 1.63s | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.80 MB/s
seaweedfs    ███████████████████████ 1.40 MB/s
localstack   ████████████████████ 1.22 MB/s
liteio       ██████████████████ 1.11 MB/s
minio        █████████████████ 1.06 MB/s
liteio_mem   █████████ 0.60 MB/s
```

**Latency (P50)**
```
rustfs       █████████ 541.6ms
seaweedfs    ████████████ 696.0ms
localstack   ██████████████ 802.8ms
liteio       ████████████████ 879.8ms
minio        ████████████████ 923.7ms
liteio_mem   ██████████████████████████████ 1.63s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.51 MB/s | 6.45s | 6.45s | 6.45s | 0 |
| liteio | 1.49 MB/s | 6.53s | 6.53s | 6.53s | 0 |
| liteio_mem | 1.41 MB/s | 6.94s | 6.94s | 6.94s | 0 |
| seaweedfs | 1.39 MB/s | 7.05s | 7.05s | 7.05s | 0 |
| localstack | 1.20 MB/s | 8.14s | 8.14s | 8.14s | 0 |
| minio | 1.14 MB/s | 8.54s | 8.54s | 8.54s | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.51 MB/s
liteio       █████████████████████████████ 1.49 MB/s
liteio_mem   ███████████████████████████ 1.41 MB/s
seaweedfs    ███████████████████████████ 1.39 MB/s
localstack   ███████████████████████ 1.20 MB/s
minio        ██████████████████████ 1.14 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████████ 6.45s
liteio       ██████████████████████ 6.53s
liteio_mem   ████████████████████████ 6.94s
seaweedfs    ████████████████████████ 7.05s
localstack   ████████████████████████████ 8.14s
minio        ██████████████████████████████ 8.54s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1281 ops/s | 727.2us | 1.2ms | 1.4ms | 0 |
| liteio | 1233 ops/s | 762.9us | 1.2ms | 1.4ms | 0 |
| minio | 631 ops/s | 1.5ms | 1.7ms | 4.2ms | 0 |
| seaweedfs | 626 ops/s | 1.5ms | 2.2ms | 4.1ms | 0 |
| localstack | 378 ops/s | 2.5ms | 3.5ms | 3.7ms | 0 |
| rustfs | 165 ops/s | 6.0ms | 6.7ms | 6.9ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1281 ops/s
liteio       ████████████████████████████ 1233 ops/s
minio        ██████████████ 631 ops/s
seaweedfs    ██████████████ 626 ops/s
localstack   ████████ 378 ops/s
rustfs       ███ 165 ops/s
```

**Latency (P50)**
```
liteio_mem   ███ 727.2us
liteio       ███ 762.9us
minio        ███████ 1.5ms
seaweedfs    ███████ 1.5ms
localstack   ████████████ 2.5ms
rustfs       ██████████████████████████████ 6.0ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.36 MB/s | 11.7ms | 13.3ms | 13.9ms | 0 |
| liteio | 1.34 MB/s | 10.8ms | 17.2ms | 17.6ms | 0 |
| rustfs | 1.30 MB/s | 11.6ms | 16.2ms | 16.5ms | 0 |
| liteio_mem | 1.17 MB/s | 12.0ms | 20.8ms | 20.9ms | 0 |
| minio | 1.13 MB/s | 13.1ms | 20.6ms | 21.2ms | 0 |
| localstack | 0.31 MB/s | 54.7ms | 56.2ms | 57.5ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 1.36 MB/s
liteio       █████████████████████████████ 1.34 MB/s
rustfs       ████████████████████████████ 1.30 MB/s
liteio_mem   █████████████████████████ 1.17 MB/s
minio        ████████████████████████ 1.13 MB/s
localstack   ██████ 0.31 MB/s
```

**Latency (P50)**
```
seaweedfs    ██████ 11.7ms
liteio       █████ 10.8ms
rustfs       ██████ 11.6ms
liteio_mem   ██████ 12.0ms
minio        ███████ 13.1ms
localstack   ██████████████████████████████ 54.7ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 2.40 MB/s | 6.9ms | 9.7ms | 10.1ms | 0 |
| liteio_mem | 1.57 MB/s | 9.9ms | 12.4ms | 12.5ms | 0 |
| liteio | 1.51 MB/s | 10.6ms | 13.4ms | 13.8ms | 0 |
| minio | 1.24 MB/s | 13.0ms | 14.8ms | 17.0ms | 0 |
| rustfs | 1.06 MB/s | 15.5ms | 16.0ms | 16.3ms | 0 |
| localstack | 0.29 MB/s | 54.3ms | 59.6ms | 59.7ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 2.40 MB/s
liteio_mem   ███████████████████ 1.57 MB/s
liteio       ██████████████████ 1.51 MB/s
minio        ███████████████ 1.24 MB/s
rustfs       █████████████ 1.06 MB/s
localstack   ███ 0.29 MB/s
```

**Latency (P50)**
```
seaweedfs    ███ 6.9ms
liteio_mem   █████ 9.9ms
liteio       █████ 10.6ms
minio        ███████ 13.0ms
rustfs       ████████ 15.5ms
localstack   ██████████████████████████████ 54.3ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.14 MB/s | 14.1ms | 16.4ms | 16.6ms | 0 |
| minio | 0.91 MB/s | 17.1ms | 24.1ms | 24.4ms | 0 |
| liteio | 0.77 MB/s | 22.6ms | 25.3ms | 25.7ms | 0 |
| rustfs | 0.73 MB/s | 23.9ms | 25.5ms | 26.0ms | 0 |
| liteio_mem | 0.70 MB/s | 25.0ms | 28.4ms | 28.7ms | 0 |
| localstack | 0.28 MB/s | 58.7ms | 59.9ms | 61.3ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 1.14 MB/s
minio        ███████████████████████ 0.91 MB/s
liteio       ████████████████████ 0.77 MB/s
rustfs       ███████████████████ 0.73 MB/s
liteio_mem   ██████████████████ 0.70 MB/s
localstack   ███████ 0.28 MB/s
```

**Latency (P50)**
```
seaweedfs    ███████ 14.1ms
minio        ████████ 17.1ms
liteio       ███████████ 22.6ms
rustfs       ████████████ 23.9ms
liteio_mem   ████████████ 25.0ms
localstack   ██████████████████████████████ 58.7ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 176.45 MB/s | 84.3ms | 95.0ms | 95.0ms | 0 |
| minio | 169.46 MB/s | 87.2ms | 99.6ms | 99.6ms | 0 |
| seaweedfs | 128.07 MB/s | 114.9ms | 126.2ms | 126.2ms | 0 |
| localstack | 126.87 MB/s | 117.2ms | 126.5ms | 126.5ms | 0 |
| liteio | 121.82 MB/s | 117.0ms | 155.6ms | 155.6ms | 0 |
| liteio_mem | 106.46 MB/s | 114.0ms | 249.9ms | 249.9ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 176.45 MB/s
minio        ████████████████████████████ 169.46 MB/s
seaweedfs    █████████████████████ 128.07 MB/s
localstack   █████████████████████ 126.87 MB/s
liteio       ████████████████████ 121.82 MB/s
liteio_mem   ██████████████████ 106.46 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████████ 84.3ms
minio        ██████████████████████ 87.2ms
seaweedfs    █████████████████████████████ 114.9ms
localstack   ██████████████████████████████ 117.2ms
liteio       █████████████████████████████ 117.0ms
liteio_mem   █████████████████████████████ 114.0ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 3.57 MB/s | 273.4us | 376.8us | 240.2us | 376.9us | 772.8us | 0 |
| liteio_mem | 3.54 MB/s | 274.0us | 431.0us | 230.7us | 431.1us | 1.3ms | 0 |
| minio | 2.87 MB/s | 339.7us | 377.5us | 339.4us | 377.5us | 384.0us | 0 |
| seaweedfs | 2.65 MB/s | 368.5us | 430.5us | 357.5us | 430.5us | 511.2us | 0 |
| rustfs | 1.58 MB/s | 618.0us | 1.0ms | 538.5us | 1.0ms | 1.2ms | 0 |
| localstack | 1.13 MB/s | 865.9us | 1.0ms | 867.2us | 1.0ms | 1.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.57 MB/s
liteio_mem   █████████████████████████████ 3.54 MB/s
minio        ████████████████████████ 2.87 MB/s
seaweedfs    ██████████████████████ 2.65 MB/s
rustfs       █████████████ 1.58 MB/s
localstack   █████████ 1.13 MB/s
```

**Latency (P50)**
```
liteio       ████████ 240.2us
liteio_mem   ███████ 230.7us
minio        ███████████ 339.4us
seaweedfs    ████████████ 357.5us
rustfs       ██████████████████ 538.5us
localstack   ██████████████████████████████ 867.2us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 1.00 MB/s | 976.0us | 1.4ms | 933.4us | 1.4ms | 1.7ms | 0 |
| rustfs | 1.00 MB/s | 978.3us | 1.4ms | 925.6us | 1.4ms | 1.6ms | 0 |
| liteio_mem | 0.91 MB/s | 1.0ms | 3.1ms | 739.7us | 3.1ms | 3.6ms | 0 |
| seaweedfs | 0.89 MB/s | 1.1ms | 1.8ms | 1.0ms | 1.8ms | 1.9ms | 0 |
| liteio | 0.84 MB/s | 1.2ms | 5.3ms | 732.4us | 5.3ms | 6.4ms | 0 |
| localstack | 0.17 MB/s | 5.6ms | 8.7ms | 5.2ms | 8.7ms | 10.6ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 1.00 MB/s
rustfs       █████████████████████████████ 1.00 MB/s
liteio_mem   ███████████████████████████ 0.91 MB/s
seaweedfs    ██████████████████████████ 0.89 MB/s
liteio       █████████████████████████ 0.84 MB/s
localstack   █████ 0.17 MB/s
```

**Latency (P50)**
```
minio        █████ 933.4us
rustfs       █████ 925.6us
liteio_mem   ████ 739.7us
seaweedfs    ██████ 1.0ms
liteio       ████ 732.4us
localstack   ██████████████████████████████ 5.2ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.33 MB/s | 2.9ms | 4.0ms | 3.1ms | 4.0ms | 4.4ms | 0 |
| liteio | 0.31 MB/s | 3.2ms | 4.2ms | 3.2ms | 4.2ms | 4.4ms | 0 |
| seaweedfs | 0.21 MB/s | 4.7ms | 6.2ms | 5.0ms | 6.2ms | 6.3ms | 0 |
| minio | 0.20 MB/s | 4.8ms | 6.4ms | 5.2ms | 6.4ms | 6.7ms | 0 |
| rustfs | 0.17 MB/s | 5.6ms | 7.9ms | 6.0ms | 7.9ms | 8.1ms | 0 |
| localstack | 0.02 MB/s | 48.6ms | 58.6ms | 56.8ms | 58.6ms | 58.8ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 0.33 MB/s
liteio       ███████████████████████████ 0.31 MB/s
seaweedfs    ██████████████████ 0.21 MB/s
minio        ██████████████████ 0.20 MB/s
rustfs       ███████████████ 0.17 MB/s
localstack   █ 0.02 MB/s
```

**Latency (P50)**
```
liteio_mem   █ 3.1ms
liteio       █ 3.2ms
seaweedfs    ██ 5.0ms
minio        ██ 5.2ms
rustfs       ███ 6.0ms
localstack   ██████████████████████████████ 56.8ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.28 MB/s | 3.5ms | 4.7ms | 3.5ms | 4.8ms | 5.0ms | 0 |
| liteio_mem | 0.28 MB/s | 3.5ms | 4.9ms | 3.5ms | 4.9ms | 5.1ms | 0 |
| seaweedfs | 0.20 MB/s | 4.9ms | 5.9ms | 5.0ms | 5.9ms | 5.9ms | 0 |
| rustfs | 0.17 MB/s | 5.6ms | 7.6ms | 5.8ms | 7.6ms | 7.9ms | 0 |
| minio | 0.17 MB/s | 5.8ms | 7.1ms | 6.2ms | 7.1ms | 7.3ms | 0 |
| localstack | 0.01 MB/s | 115.8ms | 128.0ms | 125.2ms | 128.0ms | 129.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.28 MB/s
liteio_mem   █████████████████████████████ 0.28 MB/s
seaweedfs    █████████████████████ 0.20 MB/s
rustfs       ██████████████████ 0.17 MB/s
minio        █████████████████ 0.17 MB/s
localstack   █ 0.01 MB/s
```

**Latency (P50)**
```
liteio       █ 3.5ms
liteio_mem   █ 3.5ms
seaweedfs    █ 5.0ms
rustfs       █ 5.8ms
minio        █ 6.2ms
localstack   ██████████████████████████████ 125.2ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.84 MB/s | 1.2ms | 1.9ms | 1.1ms | 1.9ms | 2.0ms | 0 |
| liteio | 0.78 MB/s | 1.2ms | 1.8ms | 1.3ms | 1.8ms | 1.9ms | 0 |
| minio | 0.55 MB/s | 1.8ms | 2.7ms | 1.7ms | 2.7ms | 2.9ms | 0 |
| rustfs | 0.54 MB/s | 1.8ms | 2.4ms | 1.8ms | 2.4ms | 2.5ms | 0 |
| seaweedfs | 0.44 MB/s | 2.2ms | 3.1ms | 2.2ms | 3.1ms | 3.7ms | 0 |
| localstack | 0.07 MB/s | 13.3ms | 21.5ms | 12.9ms | 21.5ms | 26.0ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 0.84 MB/s
liteio       ███████████████████████████ 0.78 MB/s
minio        ███████████████████ 0.55 MB/s
rustfs       ███████████████████ 0.54 MB/s
seaweedfs    ███████████████ 0.44 MB/s
localstack   ██ 0.07 MB/s
```

**Latency (P50)**
```
liteio_mem   ██ 1.1ms
liteio       ██ 1.3ms
minio        ███ 1.7ms
rustfs       ████ 1.8ms
seaweedfs    █████ 2.2ms
localstack   ██████████████████████████████ 12.9ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.46 MB/s | 2.1ms | 2.9ms | 2.2ms | 2.9ms | 3.4ms | 0 |
| liteio_mem | 0.42 MB/s | 2.3ms | 3.0ms | 2.3ms | 3.0ms | 3.3ms | 0 |
| minio | 0.30 MB/s | 3.3ms | 4.7ms | 3.4ms | 4.7ms | 4.8ms | 0 |
| rustfs | 0.25 MB/s | 3.8ms | 5.3ms | 4.0ms | 5.3ms | 6.9ms | 0 |
| seaweedfs | 0.15 MB/s | 6.4ms | 10.4ms | 5.1ms | 10.4ms | 11.5ms | 0 |
| localstack | 0.03 MB/s | 28.9ms | 32.3ms | 28.0ms | 32.3ms | 32.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.46 MB/s
liteio_mem   ███████████████████████████ 0.42 MB/s
minio        ███████████████████ 0.30 MB/s
rustfs       ████████████████ 0.25 MB/s
seaweedfs    █████████ 0.15 MB/s
localstack   ██ 0.03 MB/s
```

**Latency (P50)**
```
liteio       ██ 2.2ms
liteio_mem   ██ 2.3ms
minio        ███ 3.4ms
rustfs       ████ 4.0ms
seaweedfs    █████ 5.1ms
localstack   ██████████████████████████████ 28.0ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.46 MB/s | 642.1us | 863.8us | 1.0ms | 0 |
| rustfs | 1.42 MB/s | 680.5us | 797.2us | 1.1ms | 0 |
| liteio | 1.17 MB/s | 611.2us | 1.7ms | 2.0ms | 0 |
| liteio_mem | 1.10 MB/s | 664.8us | 1.6ms | 1.7ms | 0 |
| localstack | 1.09 MB/s | 896.5us | 1.1ms | 1.3ms | 0 |
| minio | 1.08 MB/s | 853.8us | 1.3ms | 1.4ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 1.46 MB/s
rustfs       █████████████████████████████ 1.42 MB/s
liteio       ███████████████████████ 1.17 MB/s
liteio_mem   ██████████████████████ 1.10 MB/s
localstack   ██████████████████████ 1.09 MB/s
minio        ██████████████████████ 1.08 MB/s
```

**Latency (P50)**
```
seaweedfs    █████████████████████ 642.1us
rustfs       ██████████████████████ 680.5us
liteio       ████████████████████ 611.2us
liteio_mem   ██████████████████████ 664.8us
localstack   ██████████████████████████████ 896.5us
minio        ████████████████████████████ 853.8us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.52 MB/s | 1.7ms | 3.5ms | 5.1ms | 0 |
| seaweedfs | 0.44 MB/s | 2.0ms | 4.1ms | 4.8ms | 0 |
| minio | 0.37 MB/s | 2.5ms | 4.3ms | 5.1ms | 0 |
| liteio | 0.36 MB/s | 2.6ms | 4.1ms | 4.6ms | 0 |
| liteio_mem | 0.32 MB/s | 2.6ms | 6.9ms | 7.2ms | 0 |
| localstack | 0.18 MB/s | 5.2ms | 9.0ms | 9.6ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.52 MB/s
seaweedfs    █████████████████████████ 0.44 MB/s
minio        █████████████████████ 0.37 MB/s
liteio       █████████████████████ 0.36 MB/s
liteio_mem   ██████████████████ 0.32 MB/s
localstack   ██████████ 0.18 MB/s
```

**Latency (P50)**
```
rustfs       █████████ 1.7ms
seaweedfs    ███████████ 2.0ms
minio        ██████████████ 2.5ms
liteio       ███████████████ 2.6ms
liteio_mem   ███████████████ 2.6ms
localstack   ██████████████████████████████ 5.2ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.07 MB/s | 13.6ms | 16.9ms | 30.8ms | 0 |
| liteio_mem | 0.07 MB/s | 13.3ms | 25.1ms | 26.3ms | 0 |
| seaweedfs | 0.07 MB/s | 16.2ms | 19.4ms | 19.7ms | 0 |
| minio | 0.05 MB/s | 17.9ms | 24.9ms | 25.2ms | 0 |
| liteio | 0.05 MB/s | 18.4ms | 27.0ms | 27.2ms | 0 |
| localstack | 0.02 MB/s | 38.8ms | 55.6ms | 57.0ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.07 MB/s
liteio_mem   █████████████████████████████ 0.07 MB/s
seaweedfs    ██████████████████████████ 0.07 MB/s
minio        ██████████████████████ 0.05 MB/s
liteio       ████████████████████ 0.05 MB/s
localstack   █████████ 0.02 MB/s
```

**Latency (P50)**
```
rustfs       ██████████ 13.6ms
liteio_mem   ██████████ 13.3ms
seaweedfs    ████████████ 16.2ms
minio        █████████████ 17.9ms
liteio       ██████████████ 18.4ms
localstack   ██████████████████████████████ 38.8ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.09 MB/s | 11.3ms | 14.0ms | 14.3ms | 0 |
| seaweedfs | 0.09 MB/s | 11.4ms | 13.0ms | 13.6ms | 0 |
| liteio_mem | 0.08 MB/s | 11.4ms | 18.2ms | 19.2ms | 0 |
| liteio | 0.07 MB/s | 13.7ms | 19.9ms | 20.8ms | 0 |
| minio | 0.06 MB/s | 13.8ms | 23.2ms | 25.6ms | 0 |
| localstack | 0.02 MB/s | 48.6ms | 51.9ms | 52.5ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.09 MB/s
seaweedfs    █████████████████████████████ 0.09 MB/s
liteio_mem   ███████████████████████████ 0.08 MB/s
liteio       ████████████████████████ 0.07 MB/s
minio        █████████████████████ 0.06 MB/s
localstack   ███████ 0.02 MB/s
```

**Latency (P50)**
```
rustfs       ██████ 11.3ms
seaweedfs    ███████ 11.4ms
liteio_mem   ███████ 11.4ms
liteio       ████████ 13.7ms
minio        ████████ 13.8ms
localstack   ██████████████████████████████ 48.6ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.23 MB/s | 3.6ms | 8.1ms | 9.8ms | 0 |
| seaweedfs | 0.23 MB/s | 4.1ms | 6.2ms | 6.5ms | 0 |
| liteio_mem | 0.20 MB/s | 4.6ms | 6.7ms | 7.7ms | 0 |
| liteio | 0.19 MB/s | 5.4ms | 7.8ms | 9.4ms | 0 |
| minio | 0.16 MB/s | 5.7ms | 10.2ms | 11.4ms | 0 |
| localstack | 0.07 MB/s | 12.3ms | 22.7ms | 31.8ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.23 MB/s
seaweedfs    █████████████████████████████ 0.23 MB/s
liteio_mem   ██████████████████████████ 0.20 MB/s
liteio       ███████████████████████ 0.19 MB/s
minio        ████████████████████ 0.16 MB/s
localstack   █████████ 0.07 MB/s
```

**Latency (P50)**
```
rustfs       ████████ 3.6ms
seaweedfs    ██████████ 4.1ms
liteio_mem   ███████████ 4.6ms
liteio       █████████████ 5.4ms
minio        █████████████ 5.7ms
localstack   ██████████████████████████████ 12.3ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 0.11 MB/s | 8.2ms | 17.1ms | 19.7ms | 0 |
| liteio_mem | 0.10 MB/s | 8.3ms | 18.8ms | 20.1ms | 0 |
| liteio | 0.10 MB/s | 8.5ms | 16.9ms | 21.7ms | 0 |
| seaweedfs | 0.08 MB/s | 10.1ms | 19.2ms | 19.4ms | 0 |
| rustfs | 0.08 MB/s | 9.3ms | 28.3ms | 28.9ms | 0 |
| localstack | 0.03 MB/s | 26.8ms | 65.7ms | 65.9ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.11 MB/s
liteio_mem   ████████████████████████████ 0.10 MB/s
liteio       ███████████████████████████ 0.10 MB/s
seaweedfs    ███████████████████████ 0.08 MB/s
rustfs       ██████████████████████ 0.08 MB/s
localstack   ████████ 0.03 MB/s
```

**Latency (P50)**
```
minio        █████████ 8.2ms
liteio_mem   █████████ 8.3ms
liteio       █████████ 8.5ms
seaweedfs    ███████████ 10.1ms
rustfs       ██████████ 9.3ms
localstack   ██████████████████████████████ 26.8ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 221.68 MB/s | 1.1ms | 1.5ms | 1.5ms | 0 |
| liteio_mem | 207.65 MB/s | 1.1ms | 1.6ms | 2.3ms | 0 |
| minio | 178.79 MB/s | 1.4ms | 1.8ms | 1.9ms | 0 |
| seaweedfs | 178.20 MB/s | 1.3ms | 2.0ms | 2.2ms | 0 |
| localstack | 161.45 MB/s | 1.5ms | 1.7ms | 2.1ms | 0 |
| rustfs | 108.81 MB/s | 2.0ms | 2.5ms | 6.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 221.68 MB/s
liteio_mem   ████████████████████████████ 207.65 MB/s
minio        ████████████████████████ 178.79 MB/s
seaweedfs    ████████████████████████ 178.20 MB/s
localstack   █████████████████████ 161.45 MB/s
rustfs       ██████████████ 108.81 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.1ms
liteio_mem   ████████████████ 1.1ms
minio        ████████████████████ 1.4ms
seaweedfs    ███████████████████ 1.3ms
localstack   ██████████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.0ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 224.12 MB/s | 1.1ms | 1.3ms | 1.7ms | 0 |
| liteio | 210.98 MB/s | 1.1ms | 1.5ms | 1.8ms | 0 |
| seaweedfs | 191.10 MB/s | 1.3ms | 1.6ms | 1.7ms | 0 |
| localstack | 163.26 MB/s | 1.5ms | 1.7ms | 2.0ms | 0 |
| minio | 154.73 MB/s | 1.5ms | 2.0ms | 2.4ms | 0 |
| rustfs | 121.81 MB/s | 1.9ms | 2.5ms | 5.0ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 224.12 MB/s
liteio       ████████████████████████████ 210.98 MB/s
seaweedfs    █████████████████████████ 191.10 MB/s
localstack   █████████████████████ 163.26 MB/s
minio        ████████████████████ 154.73 MB/s
rustfs       ████████████████ 121.81 MB/s
```

**Latency (P50)**
```
liteio_mem   ████████████████ 1.1ms
liteio       █████████████████ 1.1ms
seaweedfs    ███████████████████ 1.3ms
localstack   ███████████████████████ 1.5ms
minio        ███████████████████████ 1.5ms
rustfs       ██████████████████████████████ 1.9ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 233.31 MB/s | 1.0ms | 1.3ms | 1.6ms | 0 |
| liteio | 227.29 MB/s | 1.1ms | 1.2ms | 1.4ms | 0 |
| seaweedfs | 196.06 MB/s | 1.2ms | 1.6ms | 1.8ms | 0 |
| minio | 162.37 MB/s | 1.5ms | 1.8ms | 1.9ms | 0 |
| localstack | 154.47 MB/s | 1.6ms | 2.1ms | 2.2ms | 0 |
| rustfs | 117.56 MB/s | 2.0ms | 3.0ms | 3.6ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 233.31 MB/s
liteio       █████████████████████████████ 227.29 MB/s
seaweedfs    █████████████████████████ 196.06 MB/s
minio        ████████████████████ 162.37 MB/s
localstack   ███████████████████ 154.47 MB/s
rustfs       ███████████████ 117.56 MB/s
```

**Latency (P50)**
```
liteio_mem   ███████████████ 1.0ms
liteio       ████████████████ 1.1ms
seaweedfs    ██████████████████ 1.2ms
minio        ███████████████████████ 1.5ms
localstack   ███████████████████████ 1.6ms
rustfs       ██████████████████████████████ 2.0ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 297.43 MB/s | 672.5us | 811.1us | 346.2ms | 346.8ms | 346.8ms | 0 |
| minio | 297.23 MB/s | 1.1ms | 1.1ms | 336.1ms | 340.9ms | 340.9ms | 0 |
| rustfs | 292.78 MB/s | 2.1ms | 2.2ms | 340.8ms | 344.4ms | 344.4ms | 0 |
| seaweedfs | 277.64 MB/s | 2.7ms | 2.9ms | 359.2ms | 363.0ms | 363.0ms | 0 |
| liteio_mem | 276.81 MB/s | 673.4us | 814.0us | 362.2ms | 365.2ms | 365.2ms | 0 |
| localstack | 270.93 MB/s | 1.8ms | 1.8ms | 367.5ms | 370.7ms | 370.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 297.43 MB/s
minio        █████████████████████████████ 297.23 MB/s
rustfs       █████████████████████████████ 292.78 MB/s
seaweedfs    ████████████████████████████ 277.64 MB/s
liteio_mem   ███████████████████████████ 276.81 MB/s
localstack   ███████████████████████████ 270.93 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████████ 346.2ms
minio        ███████████████████████████ 336.1ms
rustfs       ███████████████████████████ 340.8ms
seaweedfs    █████████████████████████████ 359.2ms
liteio_mem   █████████████████████████████ 362.2ms
localstack   ██████████████████████████████ 367.5ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 308.43 MB/s | 1.0ms | 1.3ms | 31.6ms | 34.3ms | 34.3ms | 0 |
| liteio_mem | 299.77 MB/s | 465.6us | 537.3us | 33.2ms | 34.2ms | 34.2ms | 0 |
| localstack | 292.74 MB/s | 1.5ms | 1.6ms | 34.2ms | 35.4ms | 35.4ms | 0 |
| liteio | 287.63 MB/s | 569.3us | 730.8us | 32.6ms | 35.9ms | 35.9ms | 0 |
| seaweedfs | 284.16 MB/s | 2.1ms | 2.7ms | 34.9ms | 36.5ms | 36.5ms | 0 |
| rustfs | 259.44 MB/s | 5.7ms | 7.7ms | 37.9ms | 41.0ms | 41.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 308.43 MB/s
liteio_mem   █████████████████████████████ 299.77 MB/s
localstack   ████████████████████████████ 292.74 MB/s
liteio       ███████████████████████████ 287.63 MB/s
seaweedfs    ███████████████████████████ 284.16 MB/s
rustfs       █████████████████████████ 259.44 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████ 31.6ms
liteio_mem   ██████████████████████████ 33.2ms
localstack   ███████████████████████████ 34.2ms
liteio       █████████████████████████ 32.6ms
seaweedfs    ███████████████████████████ 34.9ms
rustfs       ██████████████████████████████ 37.9ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 4.49 MB/s | 217.4us | 248.4us | 216.9us | 248.4us | 263.0us | 0 |
| liteio | 4.05 MB/s | 241.3us | 254.5us | 212.7us | 254.5us | 1.1ms | 0 |
| minio | 2.97 MB/s | 328.6us | 369.5us | 322.5us | 369.6us | 462.4us | 0 |
| rustfs | 2.14 MB/s | 456.2us | 508.7us | 447.6us | 508.8us | 625.2us | 0 |
| seaweedfs | 2.12 MB/s | 459.9us | 527.7us | 455.4us | 528.2us | 546.4us | 0 |
| localstack | 1.42 MB/s | 688.3us | 757.7us | 668.9us | 758.3us | 1.1ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 4.49 MB/s
liteio       ███████████████████████████ 4.05 MB/s
minio        ███████████████████ 2.97 MB/s
rustfs       ██████████████ 2.14 MB/s
seaweedfs    ██████████████ 2.12 MB/s
localstack   █████████ 1.42 MB/s
```

**Latency (P50)**
```
liteio_mem   █████████ 216.9us
liteio       █████████ 212.7us
minio        ██████████████ 322.5us
rustfs       ████████████████████ 447.6us
seaweedfs    ████████████████████ 455.4us
localstack   ██████████████████████████████ 668.9us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 295.35 MB/s | 308.6us | 482.9us | 3.4ms | 3.5ms | 3.5ms | 0 |
| liteio_mem | 294.36 MB/s | 292.5us | 372.7us | 3.4ms | 3.6ms | 3.6ms | 0 |
| localstack | 259.76 MB/s | 884.4us | 964.5us | 3.8ms | 4.0ms | 4.0ms | 0 |
| seaweedfs | 254.70 MB/s | 910.9us | 1.1ms | 3.9ms | 4.2ms | 4.2ms | 0 |
| minio | 227.35 MB/s | 946.4us | 1.3ms | 4.1ms | 6.4ms | 6.4ms | 0 |
| rustfs | 192.70 MB/s | 2.0ms | 4.2ms | 4.7ms | 7.4ms | 7.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 295.35 MB/s
liteio_mem   █████████████████████████████ 294.36 MB/s
localstack   ██████████████████████████ 259.76 MB/s
seaweedfs    █████████████████████████ 254.70 MB/s
minio        ███████████████████████ 227.35 MB/s
rustfs       ███████████████████ 192.70 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 3.4ms
liteio_mem   █████████████████████ 3.4ms
localstack   ████████████████████████ 3.8ms
seaweedfs    █████████████████████████ 3.9ms
minio        ██████████████████████████ 4.1ms
rustfs       ██████████████████████████████ 4.7ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 146.97 MB/s | 220.7us | 292.2us | 411.5us | 489.1us | 491.0us | 0 |
| minio | 110.34 MB/s | 362.4us | 752.9us | 535.0us | 946.6us | 993.4us | 0 |
| liteio_mem | 102.75 MB/s | 384.7us | 728.6us | 553.1us | 952.2us | 1.3ms | 0 |
| rustfs | 94.54 MB/s | 537.3us | 596.6us | 605.3us | 741.9us | 772.8us | 0 |
| seaweedfs | 88.11 MB/s | 548.5us | 629.0us | 704.4us | 777.8us | 795.4us | 0 |
| localstack | 66.67 MB/s | 836.2us | 1.1ms | 874.9us | 1.2ms | 1.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 146.97 MB/s
minio        ██████████████████████ 110.34 MB/s
liteio_mem   ████████████████████ 102.75 MB/s
rustfs       ███████████████████ 94.54 MB/s
seaweedfs    █████████████████ 88.11 MB/s
localstack   █████████████ 66.67 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 411.5us
minio        ██████████████████ 535.0us
liteio_mem   ██████████████████ 553.1us
rustfs       ████████████████████ 605.3us
seaweedfs    ████████████████████████ 704.4us
localstack   ██████████████████████████████ 874.9us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 5405 ops/s | 176.3us | 239.0us | 262.2us | 0 |
| liteio | 5085 ops/s | 183.1us | 279.5us | 318.3us | 0 |
| minio | 4015 ops/s | 242.0us | 289.3us | 352.0us | 0 |
| seaweedfs | 3531 ops/s | 272.5us | 351.4us | 455.2us | 0 |
| rustfs | 3526 ops/s | 284.4us | 317.1us | 360.4us | 0 |
| localstack | 1494 ops/s | 638.3us | 834.8us | 1.1ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 5405 ops/s
liteio       ████████████████████████████ 5085 ops/s
minio        ██████████████████████ 4015 ops/s
seaweedfs    ███████████████████ 3531 ops/s
rustfs       ███████████████████ 3526 ops/s
localstack   ████████ 1494 ops/s
```

**Latency (P50)**
```
liteio_mem   ████████ 176.3us
liteio       ████████ 183.1us
minio        ███████████ 242.0us
seaweedfs    ████████████ 272.5us
rustfs       █████████████ 284.4us
localstack   ██████████████████████████████ 638.3us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 198.99 MB/s | 488.5ms | 524.1ms | 524.1ms | 0 |
| seaweedfs | 194.55 MB/s | 504.2ms | 504.8ms | 504.8ms | 0 |
| liteio_mem | 170.49 MB/s | 587.2ms | 589.8ms | 589.8ms | 0 |
| minio | 164.52 MB/s | 611.3ms | 644.6ms | 644.6ms | 0 |
| liteio | 154.20 MB/s | 617.5ms | 747.2ms | 747.2ms | 0 |
| localstack | 149.99 MB/s | 662.0ms | 673.6ms | 673.6ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 198.99 MB/s
seaweedfs    █████████████████████████████ 194.55 MB/s
liteio_mem   █████████████████████████ 170.49 MB/s
minio        ████████████████████████ 164.52 MB/s
liteio       ███████████████████████ 154.20 MB/s
localstack   ██████████████████████ 149.99 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████████ 488.5ms
seaweedfs    ██████████████████████ 504.2ms
liteio_mem   ██████████████████████████ 587.2ms
minio        ███████████████████████████ 611.3ms
liteio       ███████████████████████████ 617.5ms
localstack   ██████████████████████████████ 662.0ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 191.35 MB/s | 51.6ms | 61.3ms | 61.3ms | 0 |
| minio | 175.25 MB/s | 54.9ms | 60.2ms | 60.2ms | 0 |
| liteio_mem | 167.63 MB/s | 58.2ms | 64.5ms | 64.5ms | 0 |
| localstack | 147.89 MB/s | 68.6ms | 70.0ms | 70.0ms | 0 |
| seaweedfs | 147.86 MB/s | 65.7ms | 72.9ms | 72.9ms | 0 |
| liteio | 146.14 MB/s | 64.9ms | 82.3ms | 82.3ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 191.35 MB/s
minio        ███████████████████████████ 175.25 MB/s
liteio_mem   ██████████████████████████ 167.63 MB/s
localstack   ███████████████████████ 147.89 MB/s
seaweedfs    ███████████████████████ 147.86 MB/s
liteio       ██████████████████████ 146.14 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████████ 51.6ms
minio        ████████████████████████ 54.9ms
liteio_mem   █████████████████████████ 58.2ms
localstack   ██████████████████████████████ 68.6ms
seaweedfs    ████████████████████████████ 65.7ms
liteio       ████████████████████████████ 64.9ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.83 MB/s | 482.8us | 789.5us | 1.6ms | 0 |
| rustfs | 1.45 MB/s | 652.3us | 766.8us | 1.0ms | 0 |
| localstack | 1.32 MB/s | 711.2us | 891.6us | 1.2ms | 0 |
| minio | 1.15 MB/s | 813.4us | 1.0ms | 1.2ms | 0 |
| seaweedfs | 1.08 MB/s | 781.4us | 1.5ms | 2.0ms | 0 |
| liteio | 1.07 MB/s | 737.0us | 1.6ms | 1.8ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.83 MB/s
rustfs       ███████████████████████ 1.45 MB/s
localstack   █████████████████████ 1.32 MB/s
minio        ██████████████████ 1.15 MB/s
seaweedfs    █████████████████ 1.08 MB/s
liteio       █████████████████ 1.07 MB/s
```

**Latency (P50)**
```
liteio_mem   █████████████████ 482.8us
rustfs       ████████████████████████ 652.3us
localstack   ██████████████████████████ 711.2us
minio        ██████████████████████████████ 813.4us
seaweedfs    ████████████████████████████ 781.4us
liteio       ███████████████████████████ 737.0us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 175.50 MB/s | 5.4ms | 7.4ms | 7.4ms | 0 |
| liteio | 165.38 MB/s | 5.8ms | 7.5ms | 7.5ms | 0 |
| liteio_mem | 148.18 MB/s | 6.2ms | 9.7ms | 9.7ms | 0 |
| localstack | 130.51 MB/s | 7.5ms | 8.2ms | 8.2ms | 0 |
| minio | 129.91 MB/s | 7.6ms | 8.9ms | 8.9ms | 0 |
| seaweedfs | 120.50 MB/s | 8.1ms | 9.6ms | 9.6ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 175.50 MB/s
liteio       ████████████████████████████ 165.38 MB/s
liteio_mem   █████████████████████████ 148.18 MB/s
localstack   ██████████████████████ 130.51 MB/s
minio        ██████████████████████ 129.91 MB/s
seaweedfs    ████████████████████ 120.50 MB/s
```

**Latency (P50)**
```
rustfs       ███████████████████ 5.4ms
liteio       █████████████████████ 5.8ms
liteio_mem   ██████████████████████ 6.2ms
localstack   ███████████████████████████ 7.5ms
minio        ████████████████████████████ 7.6ms
seaweedfs    ██████████████████████████████ 8.1ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 71.90 MB/s | 768.0us | 1.1ms | 2.2ms | 0 |
| rustfs | 60.02 MB/s | 965.0us | 1.2ms | 1.9ms | 0 |
| liteio | 57.50 MB/s | 937.2us | 1.6ms | 2.1ms | 0 |
| localstack | 55.79 MB/s | 1.1ms | 1.4ms | 1.9ms | 0 |
| minio | 54.36 MB/s | 1.1ms | 1.4ms | 1.4ms | 0 |
| seaweedfs | 51.47 MB/s | 1.1ms | 1.4ms | 1.5ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 71.90 MB/s
rustfs       █████████████████████████ 60.02 MB/s
liteio       ███████████████████████ 57.50 MB/s
localstack   ███████████████████████ 55.79 MB/s
minio        ██████████████████████ 54.36 MB/s
seaweedfs    █████████████████████ 51.47 MB/s
```

**Latency (P50)**
```
liteio_mem   ████████████████████ 768.0us
rustfs       █████████████████████████ 965.0us
liteio       ████████████████████████ 937.2us
localstack   ███████████████████████████ 1.1ms
minio        █████████████████████████████ 1.1ms
seaweedfs    ██████████████████████████████ 1.1ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 79.22MiB / 7.653GiB | 79.2 MB | - | 0.0% | 3160.1 MB | 524kB / 2.23GB |
| liteio_mem | 68.31MiB / 7.653GiB | 68.3 MB | - | 0.0% | 3160.1 MB | 8.19kB / 2.23GB |
| localstack | 364.6MiB / 7.653GiB | 364.6 MB | - | 0.1% | 0.0 MB | 39.3MB / 2.03GB |
| minio | 401.2MiB / 7.653GiB | 401.2 MB | - | 0.0% | 3349.5 MB | 54.9MB / 2.01GB |
| rustfs | 518.1MiB / 7.653GiB | 518.1 MB | - | 0.1% | 2489.3 MB | 6.3MB / 1.62GB |
| seaweedfs | 106.6MiB / 7.653GiB | 106.6 MB | - | 0.0% | (no data) | 1.92MB / 0B |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** rustfs
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
