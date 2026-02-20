# Storage Benchmark Report

**Generated:** 2026-02-20T14:15:20+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** herd_s3 (won 30/48 benchmarks, 62%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | herd_s3 | 30 | 62% |
| 2 | liteio | 14 | 29% |
| 3 | minio | 4 | 8% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 5.4 MB/s | close |
| Small Write (1KB) | herd_s3 | 4.8 MB/s | 3.1x vs liteio |
| Large Read (100MB) | minio | 314.6 MB/s | +29% vs herd_s3 |
| Large Write (100MB) | herd_s3 | 173.4 MB/s | close |
| Delete | herd_s3 | 6.1K ops/s | +44% vs liteio |
| Stat | herd_s3 | 5.8K ops/s | +34% vs liteio |
| List (100 objects) | herd_s3 | 1.7K ops/s | +20% vs liteio |
| Copy | herd_s3 | 5.3 MB/s | 3.7x vs liteio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **herd_s3** | 173 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 315 MB/s | Best for streaming, CDN |
| Small File Operations | **herd_s3** | 5107 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **herd_s3** | - | Best for multi-user apps |
| Memory Constrained | **seaweedfs** | 216 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| herd_s3 | 173.4 | 244.6 | 568.0ms | 395.5ms |
| liteio | 152.0 | 234.9 | 573.8ms | 423.2ms |
| minio | 167.9 | 314.6 | 586.2ms | 309.7ms |
| rustfs | 162.8 | 243.8 | 600.2ms | 426.3ms |
| seaweedfs | 169.9 | 242.5 | 594.0ms | 416.4ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| herd_s3 | 4929 | 5284 | 179.6us | 170.0us |
| liteio | 1613 | 5509 | 590.5us | 154.3us |
| minio | 1464 | 3092 | 655.3us | 295.1us |
| rustfs | 1030 | 1869 | 821.2us | 518.5us |
| seaweedfs | 1033 | 2692 | 782.5us | 351.2us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| herd_s3 | 5808 | 1703 | 6096 |
| liteio | 4341 | 1415 | 4239 |
| minio | 4181 | 596 | 677 |
| rustfs | 2429 | 141 | 851 |
| seaweedfs | 3625 | 660 | 2557 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| herd_s3 | 4.15 | 1.10 | 0.60 | 0.33 | 0.17 | 0.08 |
| liteio | 1.08 | 0.49 | 0.28 | 0.21 | 0.12 | 0.06 |
| minio | 0.36 | 0.34 | 0.15 | 0.07 | 0.03 | 0.02 |
| rustfs | 1.13 | 0.23 | 0.10 | 0.05 | 0.03 | 0.01 |
| seaweedfs | 1.36 | 0.43 | 0.23 | 0.13 | 0.07 | 0.04 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| herd_s3 | 4.78 | 1.36 | 0.73 | 0.39 | 0.21 | 0.08 |
| liteio | 4.00 | 1.35 | 0.80 | 0.43 | 0.18 | 0.08 |
| minio | 2.53 | 1.11 | 0.54 | 0.27 | 0.13 | 0.06 |
| rustfs | 1.51 | 0.23 | 0.09 | 0.05 | 0.03 | 0.01 |
| seaweedfs | 1.83 | 0.74 | 0.38 | 0.23 | 0.13 | 0.06 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| herd_s3 | 2.2ms | 22.3ms | 223.5ms | 2.30s |
| liteio | 6.5ms | 58.6ms | 595.9ms | 6.84s |
| minio | 9.3ms | 94.3ms | 1.13s | 9.82s |
| rustfs | 7.9ms | 80.0ms | 839.7ms | 8.44s |
| seaweedfs | 6.3ms | 76.4ms | 652.4ms | 6.58s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| herd_s3 | 325.1us | 859.4us | 4.9ms | 88.5ms |
| liteio | 343.9us | 821.0us | 5.5ms | 192.9ms |
| minio | 1.0ms | 2.0ms | 11.7ms | 184.5ms |
| rustfs | 1.7ms | 6.5ms | 60.9ms | 776.1ms |
| seaweedfs | 596.0us | 1.6ms | 7.7ms | 107.2ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| herd_s3 | 1872.9 MB | 6.7% |
| liteio | 1227.8 MB | 5.4% |
| minio | 1106.9 MB | 0.0% |
| rustfs | 938.9 MB | 1.4% |
| seaweedfs | 215.9 MB | 1.4% |

---

## Configuration

| Parameter | Value |
|-----------|-------|
| BenchTime | 1s |
| MinIterations | 3 |
| Warmup | 10 |
| Concurrency | 200 |
| Timeout | 30s |

## Drivers Tested

- **herd_s3** (48 benchmarks)
- **liteio** (48 benchmarks)
- **minio** (48 benchmarks)
- **rustfs** (48 benchmarks)
- **seaweedfs** (48 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 5.33 MB/s | 167.5us | 267.5us | 376.1us | 0 |
| liteio | 1.44 MB/s | 599.8us | 1.1ms | 2.1ms | 0 |
| minio | 1.02 MB/s | 903.8us | 1.3ms | 1.8ms | 0 |
| seaweedfs | 0.89 MB/s | 943.2us | 1.9ms | 3.1ms | 0 |
| rustfs | 0.82 MB/s | 1.1ms | 1.9ms | 2.8ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 5.33 MB/s
liteio       ████████ 1.44 MB/s
minio        █████ 1.02 MB/s
seaweedfs    █████ 0.89 MB/s
rustfs       ████ 0.82 MB/s
```

**Latency (P50)**
```
herd_s3      ████ 167.5us
liteio       ████████████████ 599.8us
minio        █████████████████████████ 903.8us
seaweedfs    ██████████████████████████ 943.2us
rustfs       ██████████████████████████████ 1.1ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 6096 ops/s | 154.6us | 216.5us | 299.0us | 0 |
| liteio | 4239 ops/s | 197.0us | 427.9us | 748.2us | 0 |
| seaweedfs | 2557 ops/s | 315.2us | 753.2us | 1.6ms | 0 |
| rustfs | 851 ops/s | 1.1ms | 1.6ms | 2.5ms | 0 |
| minio | 677 ops/s | 1.5ms | 2.8ms | 3.7ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 6096 ops/s
liteio       ████████████████████ 4239 ops/s
seaweedfs    ████████████ 2557 ops/s
rustfs       ████ 851 ops/s
minio        ███ 677 ops/s
```

**Latency (P50)**
```
herd_s3      ███ 154.6us
liteio       ███ 197.0us
seaweedfs    ██████ 315.2us
rustfs       ████████████████████ 1.1ms
minio        ██████████████████████████████ 1.5ms
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.40 MB/s | 212.8us | 368.9us | 533.2us | 0 |
| liteio | 0.15 MB/s | 597.0us | 846.2us | 1.4ms | 0 |
| seaweedfs | 0.13 MB/s | 643.4us | 1.4ms | 2.5ms | 0 |
| rustfs | 0.11 MB/s | 814.8us | 1.2ms | 1.8ms | 0 |
| minio | 0.10 MB/s | 850.7us | 1.7ms | 3.1ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.40 MB/s
liteio       ███████████ 0.15 MB/s
seaweedfs    █████████ 0.13 MB/s
rustfs       ████████ 0.11 MB/s
minio        ███████ 0.10 MB/s
```

**Latency (P50)**
```
herd_s3      ███████ 212.8us
liteio       █████████████████████ 597.0us
seaweedfs    ██████████████████████ 643.4us
rustfs       ████████████████████████████ 814.8us
minio        ██████████████████████████████ 850.7us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 4588 ops/s | 190.9us | 336.7us | 581.0us | 0 |
| seaweedfs | 3202 ops/s | 290.8us | 433.0us | 629.5us | 0 |
| liteio | 1848 ops/s | 502.8us | 751.9us | 1.2ms | 0 |
| rustfs | 1229 ops/s | 785.2us | 1.1ms | 1.4ms | 0 |
| minio | 962 ops/s | 897.2us | 1.5ms | 3.5ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 4588 ops/s
seaweedfs    ████████████████████ 3202 ops/s
liteio       ████████████ 1848 ops/s
rustfs       ████████ 1229 ops/s
minio        ██████ 962 ops/s
```

**Latency (P50)**
```
herd_s3      ██████ 190.9us
seaweedfs    █████████ 290.8us
liteio       ████████████████ 502.8us
rustfs       ██████████████████████████ 785.2us
minio        ██████████████████████████████ 897.2us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.39 MB/s | 224.0us | 365.9us | 522.0us | 0 |
| liteio | 0.15 MB/s | 589.0us | 857.0us | 1.4ms | 0 |
| seaweedfs | 0.14 MB/s | 623.2us | 1.0ms | 2.2ms | 0 |
| rustfs | 0.10 MB/s | 830.8us | 1.3ms | 2.5ms | 0 |
| minio | 0.08 MB/s | 939.5us | 1.3ms | 1.6ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.39 MB/s
liteio       ███████████ 0.15 MB/s
seaweedfs    ██████████ 0.14 MB/s
rustfs       ████████ 0.10 MB/s
minio        ██████ 0.08 MB/s
```

**Latency (P50)**
```
herd_s3      ███████ 224.0us
liteio       ██████████████████ 589.0us
seaweedfs    ███████████████████ 623.2us
rustfs       ██████████████████████████ 830.8us
minio        ██████████████████████████████ 939.5us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1703 ops/s | 577.1us | 653.5us | 894.8us | 0 |
| liteio | 1415 ops/s | 659.9us | 983.2us | 1.4ms | 0 |
| seaweedfs | 660 ops/s | 1.4ms | 2.2ms | 4.0ms | 0 |
| minio | 596 ops/s | 1.5ms | 2.5ms | 4.5ms | 0 |
| rustfs | 141 ops/s | 6.9ms | 8.4ms | 10.2ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1703 ops/s
liteio       ████████████████████████ 1415 ops/s
seaweedfs    ███████████ 660 ops/s
minio        ██████████ 596 ops/s
rustfs       ██ 141 ops/s
```

**Latency (P50)**
```
herd_s3      ██ 577.1us
liteio       ██ 659.9us
seaweedfs    █████ 1.4ms
minio        ██████ 1.5ms
rustfs       ██████████████████████████████ 6.9ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.62 MB/s | 18.4ms | 47.2ms | 243.2ms | 0 |
| herd_s3 | 0.58 MB/s | 16.5ms | 39.0ms | 283.0ms | 0 |
| seaweedfs | 0.39 MB/s | 30.6ms | 60.9ms | 275.9ms | 0 |
| minio | 0.33 MB/s | 32.9ms | 124.1ms | 179.3ms | 0 |
| rustfs | 0.19 MB/s | 74.1ms | 123.0ms | 136.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.62 MB/s
herd_s3      ████████████████████████████ 0.58 MB/s
seaweedfs    ██████████████████ 0.39 MB/s
minio        ████████████████ 0.33 MB/s
rustfs       █████████ 0.19 MB/s
```

**Latency (P50)**
```
liteio       ███████ 18.4ms
herd_s3      ██████ 16.5ms
seaweedfs    ████████████ 30.6ms
minio        █████████████ 32.9ms
rustfs       ██████████████████████████████ 74.1ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.76 MB/s | 20.3ms | 28.9ms | 35.2ms | 0 |
| herd_s3 | 0.68 MB/s | 22.5ms | 32.3ms | 35.4ms | 0 |
| minio | 0.54 MB/s | 14.4ms | 125.0ms | 254.5ms | 0 |
| seaweedfs | 0.47 MB/s | 31.0ms | 59.9ms | 72.6ms | 0 |
| rustfs | 0.19 MB/s | 78.0ms | 135.5ms | 159.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.76 MB/s
herd_s3      ██████████████████████████ 0.68 MB/s
minio        █████████████████████ 0.54 MB/s
seaweedfs    ██████████████████ 0.47 MB/s
rustfs       ███████ 0.19 MB/s
```

**Latency (P50)**
```
liteio       ███████ 20.3ms
herd_s3      ████████ 22.5ms
minio        █████ 14.4ms
seaweedfs    ███████████ 31.0ms
rustfs       ██████████████████████████████ 78.0ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.60 MB/s | 12.2ms | 54.7ms | 243.5ms | 0 |
| liteio | 0.54 MB/s | 20.1ms | 57.0ms | 240.2ms | 0 |
| seaweedfs | 0.35 MB/s | 26.6ms | 59.2ms | 662.3ms | 0 |
| minio | 0.23 MB/s | 59.9ms | 151.0ms | 206.9ms | 0 |
| rustfs | 0.19 MB/s | 74.9ms | 114.3ms | 468.1ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.60 MB/s
liteio       ███████████████████████████ 0.54 MB/s
seaweedfs    █████████████████ 0.35 MB/s
minio        ███████████ 0.23 MB/s
rustfs       █████████ 0.19 MB/s
```

**Latency (P50)**
```
herd_s3      ████ 12.2ms
liteio       ████████ 20.1ms
seaweedfs    ██████████ 26.6ms
minio        ███████████████████████ 59.9ms
rustfs       ██████████████████████████████ 74.9ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 150.54 MB/s | 97.5ms | 110.8ms | 110.8ms | 0 |
| minio | 144.62 MB/s | 104.7ms | 115.2ms | 115.2ms | 0 |
| rustfs | 140.58 MB/s | 104.7ms | 112.0ms | 112.0ms | 0 |
| herd_s3 | 125.22 MB/s | 118.3ms | 124.6ms | 124.6ms | 0 |
| seaweedfs | 92.00 MB/s | 148.8ms | 174.3ms | 174.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 150.54 MB/s
minio        ████████████████████████████ 144.62 MB/s
rustfs       ████████████████████████████ 140.58 MB/s
herd_s3      ████████████████████████ 125.22 MB/s
seaweedfs    ██████████████████ 92.00 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 97.5ms
minio        █████████████████████ 104.7ms
rustfs       █████████████████████ 104.7ms
herd_s3      ███████████████████████ 118.3ms
seaweedfs    ██████████████████████████████ 148.8ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 4.78 MB/s | 195.0us | 253.7us | 341.8us | 0 |
| liteio | 4.00 MB/s | 216.8us | 373.3us | 518.7us | 0 |
| minio | 2.53 MB/s | 346.8us | 551.6us | 1.1ms | 0 |
| seaweedfs | 1.83 MB/s | 445.1us | 979.8us | 2.1ms | 0 |
| rustfs | 1.51 MB/s | 584.7us | 926.0us | 1.5ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 4.78 MB/s
liteio       █████████████████████████ 4.00 MB/s
minio        ███████████████ 2.53 MB/s
seaweedfs    ███████████ 1.83 MB/s
rustfs       █████████ 1.51 MB/s
```

**Latency (P50)**
```
herd_s3      ██████████ 195.0us
liteio       ███████████ 216.8us
minio        █████████████████ 346.8us
seaweedfs    ██████████████████████ 445.1us
rustfs       ██████████████████████████████ 584.7us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1.36 MB/s | 688.1us | 1.1ms | 1.6ms | 0 |
| liteio | 1.35 MB/s | 658.0us | 1.3ms | 1.9ms | 0 |
| minio | 1.11 MB/s | 820.4us | 1.4ms | 2.2ms | 0 |
| seaweedfs | 0.74 MB/s | 1.2ms | 2.1ms | 3.3ms | 0 |
| rustfs | 0.23 MB/s | 3.8ms | 7.0ms | 10.6ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1.36 MB/s
liteio       █████████████████████████████ 1.35 MB/s
minio        ████████████████████████ 1.11 MB/s
seaweedfs    ████████████████ 0.74 MB/s
rustfs       █████ 0.23 MB/s
```

**Latency (P50)**
```
herd_s3      █████ 688.1us
liteio       █████ 658.0us
minio        ██████ 820.4us
seaweedfs    █████████ 1.2ms
rustfs       ██████████████████████████████ 3.8ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.21 MB/s | 4.6ms | 7.0ms | 9.7ms | 0 |
| liteio | 0.18 MB/s | 5.1ms | 9.4ms | 14.8ms | 0 |
| seaweedfs | 0.13 MB/s | 7.0ms | 13.0ms | 18.1ms | 0 |
| minio | 0.13 MB/s | 6.5ms | 17.0ms | 26.0ms | 0 |
| rustfs | 0.03 MB/s | 37.6ms | 50.3ms | 58.8ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.21 MB/s
liteio       █████████████████████████ 0.18 MB/s
seaweedfs    ██████████████████ 0.13 MB/s
minio        ██████████████████ 0.13 MB/s
rustfs       ███ 0.03 MB/s
```

**Latency (P50)**
```
herd_s3      ███ 4.6ms
liteio       ████ 5.1ms
seaweedfs    █████ 7.0ms
minio        █████ 6.5ms
rustfs       ██████████████████████████████ 37.6ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.08 MB/s | 10.9ms | 21.1ms | 27.2ms | 0 |
| herd_s3 | 0.08 MB/s | 11.1ms | 21.0ms | 38.5ms | 0 |
| minio | 0.06 MB/s | 13.7ms | 33.5ms | 56.1ms | 0 |
| seaweedfs | 0.06 MB/s | 14.0ms | 28.1ms | 37.8ms | 0 |
| rustfs | 0.01 MB/s | 73.8ms | 93.6ms | 100.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.08 MB/s
herd_s3      ████████████████████████████ 0.08 MB/s
minio        ███████████████████████ 0.06 MB/s
seaweedfs    ██████████████████████ 0.06 MB/s
rustfs       ████ 0.01 MB/s
```

**Latency (P50)**
```
liteio       ████ 10.9ms
herd_s3      ████ 11.1ms
minio        █████ 13.7ms
seaweedfs    █████ 14.0ms
rustfs       ██████████████████████████████ 73.8ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.80 MB/s | 1.2ms | 1.9ms | 2.6ms | 0 |
| herd_s3 | 0.73 MB/s | 1.3ms | 2.1ms | 2.8ms | 0 |
| minio | 0.54 MB/s | 1.7ms | 3.0ms | 4.1ms | 0 |
| seaweedfs | 0.38 MB/s | 2.4ms | 4.4ms | 6.2ms | 0 |
| rustfs | 0.09 MB/s | 9.8ms | 15.7ms | 22.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.80 MB/s
herd_s3      ███████████████████████████ 0.73 MB/s
minio        ████████████████████ 0.54 MB/s
seaweedfs    ██████████████ 0.38 MB/s
rustfs       ███ 0.09 MB/s
```

**Latency (P50)**
```
liteio       ███ 1.2ms
herd_s3      ███ 1.3ms
minio        █████ 1.7ms
seaweedfs    ███████ 2.4ms
rustfs       ██████████████████████████████ 9.8ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.43 MB/s | 2.1ms | 3.9ms | 5.5ms | 0 |
| herd_s3 | 0.39 MB/s | 2.4ms | 3.9ms | 4.9ms | 0 |
| minio | 0.27 MB/s | 3.3ms | 6.5ms | 8.9ms | 0 |
| seaweedfs | 0.23 MB/s | 4.0ms | 6.9ms | 9.2ms | 0 |
| rustfs | 0.05 MB/s | 18.1ms | 29.4ms | 62.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.43 MB/s
herd_s3      ██████████████████████████ 0.39 MB/s
minio        ██████████████████ 0.27 MB/s
seaweedfs    ████████████████ 0.23 MB/s
rustfs       ███ 0.05 MB/s
```

**Latency (P50)**
```
liteio       ███ 2.1ms
herd_s3      ████ 2.4ms
minio        █████ 3.3ms
seaweedfs    ██████ 4.0ms
rustfs       ██████████████████████████████ 18.1ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 4.15 MB/s | 222.2us | 305.5us | 469.3us | 0 |
| seaweedfs | 1.36 MB/s | 644.2us | 1.1ms | 2.2ms | 0 |
| rustfs | 1.13 MB/s | 833.4us | 1.0ms | 1.2ms | 0 |
| liteio | 1.08 MB/s | 809.4us | 1.2ms | 2.3ms | 0 |
| minio | 0.36 MB/s | 2.7ms | 4.2ms | 6.8ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 4.15 MB/s
seaweedfs    █████████ 1.36 MB/s
rustfs       ████████ 1.13 MB/s
liteio       ███████ 1.08 MB/s
minio        ██ 0.36 MB/s
```

**Latency (P50)**
```
herd_s3      ██ 222.2us
seaweedfs    ███████ 644.2us
rustfs       █████████ 833.4us
liteio       ████████ 809.4us
minio        ██████████████████████████████ 2.7ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1.10 MB/s | 818.2us | 1.4ms | 2.4ms | 0 |
| liteio | 0.49 MB/s | 1.9ms | 3.2ms | 4.5ms | 0 |
| seaweedfs | 0.43 MB/s | 2.1ms | 3.6ms | 5.0ms | 0 |
| minio | 0.34 MB/s | 2.5ms | 4.5ms | 8.5ms | 0 |
| rustfs | 0.23 MB/s | 4.1ms | 6.3ms | 8.7ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1.10 MB/s
liteio       █████████████ 0.49 MB/s
seaweedfs    ███████████ 0.43 MB/s
minio        █████████ 0.34 MB/s
rustfs       ██████ 0.23 MB/s
```

**Latency (P50)**
```
herd_s3      ██████ 818.2us
liteio       █████████████ 1.9ms
seaweedfs    ███████████████ 2.1ms
minio        ██████████████████ 2.5ms
rustfs       ██████████████████████████████ 4.1ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.17 MB/s | 5.7ms | 8.4ms | 10.2ms | 0 |
| liteio | 0.12 MB/s | 7.2ms | 12.7ms | 18.9ms | 0 |
| seaweedfs | 0.07 MB/s | 12.5ms | 22.2ms | 32.6ms | 0 |
| minio | 0.03 MB/s | 25.9ms | 57.4ms | 92.4ms | 0 |
| rustfs | 0.03 MB/s | 38.3ms | 45.2ms | 50.1ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.17 MB/s
liteio       █████████████████████ 0.12 MB/s
seaweedfs    ████████████ 0.07 MB/s
minio        ██████ 0.03 MB/s
rustfs       ████ 0.03 MB/s
```

**Latency (P50)**
```
herd_s3      ████ 5.7ms
liteio       █████ 7.2ms
seaweedfs    █████████ 12.5ms
minio        ████████████████████ 25.9ms
rustfs       ██████████████████████████████ 38.3ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.08 MB/s | 12.2ms | 18.5ms | 27.9ms | 0 |
| liteio | 0.06 MB/s | 15.4ms | 29.3ms | 42.7ms | 0 |
| seaweedfs | 0.04 MB/s | 22.7ms | 39.2ms | 46.3ms | 0 |
| minio | 0.02 MB/s | 49.5ms | 150.8ms | 270.5ms | 0 |
| rustfs | 0.01 MB/s | 79.6ms | 114.8ms | 199.4ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.08 MB/s
liteio       ██████████████████████ 0.06 MB/s
seaweedfs    ███████████████ 0.04 MB/s
minio        ██████ 0.02 MB/s
rustfs       ████ 0.01 MB/s
```

**Latency (P50)**
```
herd_s3      ████ 12.2ms
liteio       █████ 15.4ms
seaweedfs    ████████ 22.7ms
minio        ██████████████████ 49.5ms
rustfs       ██████████████████████████████ 79.6ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.60 MB/s | 1.6ms | 2.5ms | 3.3ms | 0 |
| liteio | 0.28 MB/s | 3.1ms | 5.9ms | 10.0ms | 0 |
| seaweedfs | 0.23 MB/s | 3.9ms | 7.2ms | 11.2ms | 0 |
| minio | 0.15 MB/s | 5.6ms | 12.4ms | 16.1ms | 0 |
| rustfs | 0.10 MB/s | 9.5ms | 14.7ms | 27.6ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.60 MB/s
liteio       █████████████ 0.28 MB/s
seaweedfs    ███████████ 0.23 MB/s
minio        ███████ 0.15 MB/s
rustfs       ████ 0.10 MB/s
```

**Latency (P50)**
```
herd_s3      ████ 1.6ms
liteio       █████████ 3.1ms
seaweedfs    ████████████ 3.9ms
minio        █████████████████ 5.6ms
rustfs       ██████████████████████████████ 9.5ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.33 MB/s | 2.8ms | 4.4ms | 6.5ms | 0 |
| liteio | 0.21 MB/s | 4.3ms | 6.7ms | 9.4ms | 0 |
| seaweedfs | 0.13 MB/s | 7.3ms | 12.7ms | 15.4ms | 0 |
| minio | 0.07 MB/s | 11.9ms | 29.1ms | 37.6ms | 0 |
| rustfs | 0.05 MB/s | 19.4ms | 24.2ms | 30.3ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.33 MB/s
liteio       ███████████████████ 0.21 MB/s
seaweedfs    ███████████ 0.13 MB/s
minio        ██████ 0.07 MB/s
rustfs       ████ 0.05 MB/s
```

**Latency (P50)**
```
herd_s3      ████ 2.8ms
liteio       ██████ 4.3ms
seaweedfs    ███████████ 7.3ms
minio        ██████████████████ 11.9ms
rustfs       ██████████████████████████████ 19.4ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 183.33 MB/s | 1.3ms | 1.6ms | 2.2ms | 0 |
| liteio | 182.04 MB/s | 1.3ms | 2.0ms | 2.8ms | 0 |
| seaweedfs | 146.32 MB/s | 1.5ms | 2.5ms | 3.6ms | 0 |
| herd_s3 | 140.49 MB/s | 1.6ms | 3.0ms | 3.5ms | 0 |
| rustfs | 98.94 MB/s | 2.2ms | 4.0ms | 6.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 183.33 MB/s
liteio       █████████████████████████████ 182.04 MB/s
seaweedfs    ███████████████████████ 146.32 MB/s
herd_s3      ██████████████████████ 140.49 MB/s
rustfs       ████████████████ 98.94 MB/s
```

**Latency (P50)**
```
minio        █████████████████ 1.3ms
liteio       █████████████████ 1.3ms
seaweedfs    ████████████████████ 1.5ms
herd_s3      ████████████████████ 1.6ms
rustfs       ██████████████████████████████ 2.2ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 189.06 MB/s | 1.3ms | 1.6ms | 2.0ms | 0 |
| minio | 174.33 MB/s | 1.4ms | 1.9ms | 2.7ms | 0 |
| herd_s3 | 167.46 MB/s | 1.4ms | 2.0ms | 2.8ms | 0 |
| seaweedfs | 148.97 MB/s | 1.5ms | 2.7ms | 3.6ms | 0 |
| rustfs | 90.65 MB/s | 2.4ms | 4.3ms | 8.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 189.06 MB/s
minio        ███████████████████████████ 174.33 MB/s
herd_s3      ██████████████████████████ 167.46 MB/s
seaweedfs    ███████████████████████ 148.97 MB/s
rustfs       ██████████████ 90.65 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.3ms
minio        ████████████████ 1.4ms
herd_s3      █████████████████ 1.4ms
seaweedfs    ██████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.4ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 170.43 MB/s | 1.4ms | 2.1ms | 2.6ms | 0 |
| herd_s3 | 164.58 MB/s | 1.4ms | 2.1ms | 2.8ms | 0 |
| seaweedfs | 160.98 MB/s | 1.5ms | 2.0ms | 2.8ms | 0 |
| minio | 158.78 MB/s | 1.4ms | 2.3ms | 4.4ms | 0 |
| rustfs | 105.18 MB/s | 2.2ms | 3.3ms | 4.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 170.43 MB/s
herd_s3      ████████████████████████████ 164.58 MB/s
seaweedfs    ████████████████████████████ 160.98 MB/s
minio        ███████████████████████████ 158.78 MB/s
rustfs       ██████████████████ 105.18 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 1.4ms
herd_s3      ███████████████████ 1.4ms
seaweedfs    ███████████████████ 1.5ms
minio        ███████████████████ 1.4ms
rustfs       ██████████████████████████████ 2.2ms
```

### Read/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 314.58 MB/s | 309.7ms | 311.9ms | 311.9ms | 0 |
| herd_s3 | 244.62 MB/s | 395.5ms | 395.5ms | 395.5ms | 0 |
| rustfs | 243.80 MB/s | 426.3ms | 426.3ms | 426.3ms | 0 |
| seaweedfs | 242.50 MB/s | 416.4ms | 416.4ms | 416.4ms | 0 |
| liteio | 234.90 MB/s | 423.2ms | 423.2ms | 423.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 314.58 MB/s
herd_s3      ███████████████████████ 244.62 MB/s
rustfs       ███████████████████████ 243.80 MB/s
seaweedfs    ███████████████████████ 242.50 MB/s
liteio       ██████████████████████ 234.90 MB/s
```

**Latency (P50)**
```
minio        █████████████████████ 309.7ms
herd_s3      ███████████████████████████ 395.5ms
rustfs       ██████████████████████████████ 426.3ms
seaweedfs    █████████████████████████████ 416.4ms
liteio       █████████████████████████████ 423.2ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 313.91 MB/s | 30.7ms | 34.4ms | 40.6ms | 0 |
| rustfs | 262.93 MB/s | 37.2ms | 41.2ms | 42.2ms | 0 |
| liteio | 253.03 MB/s | 38.2ms | 45.6ms | 52.5ms | 0 |
| seaweedfs | 242.48 MB/s | 39.8ms | 45.5ms | 54.4ms | 0 |
| herd_s3 | 237.86 MB/s | 40.5ms | 48.4ms | 54.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 313.91 MB/s
rustfs       █████████████████████████ 262.93 MB/s
liteio       ████████████████████████ 253.03 MB/s
seaweedfs    ███████████████████████ 242.48 MB/s
herd_s3      ██████████████████████ 237.86 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████ 30.7ms
rustfs       ███████████████████████████ 37.2ms
liteio       ████████████████████████████ 38.2ms
seaweedfs    █████████████████████████████ 39.8ms
herd_s3      ██████████████████████████████ 40.5ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5.38 MB/s | 154.3us | 313.2us | 524.9us | 0 |
| herd_s3 | 5.16 MB/s | 170.0us | 288.1us | 519.2us | 0 |
| minio | 3.02 MB/s | 295.1us | 427.2us | 933.0us | 0 |
| seaweedfs | 2.63 MB/s | 351.2us | 514.5us | 792.8us | 0 |
| rustfs | 1.83 MB/s | 518.5us | 639.0us | 929.9us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5.38 MB/s
herd_s3      ████████████████████████████ 5.16 MB/s
minio        ████████████████ 3.02 MB/s
seaweedfs    ██████████████ 2.63 MB/s
rustfs       ██████████ 1.83 MB/s
```

**Latency (P50)**
```
liteio       ████████ 154.3us
herd_s3      █████████ 170.0us
minio        █████████████████ 295.1us
seaweedfs    ████████████████████ 351.2us
rustfs       ██████████████████████████████ 518.5us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 262.98 MB/s | 3.7ms | 4.4ms | 6.4ms | 0 |
| liteio | 230.16 MB/s | 4.1ms | 6.0ms | 7.2ms | 0 |
| herd_s3 | 228.26 MB/s | 4.2ms | 5.2ms | 7.2ms | 0 |
| rustfs | 219.36 MB/s | 4.3ms | 6.0ms | 8.7ms | 0 |
| seaweedfs | 205.21 MB/s | 4.5ms | 6.5ms | 9.5ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 262.98 MB/s
liteio       ██████████████████████████ 230.16 MB/s
herd_s3      ██████████████████████████ 228.26 MB/s
rustfs       █████████████████████████ 219.36 MB/s
seaweedfs    ███████████████████████ 205.21 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████ 3.7ms
liteio       ██████████████████████████ 4.1ms
herd_s3      ███████████████████████████ 4.2ms
rustfs       ████████████████████████████ 4.3ms
seaweedfs    ██████████████████████████████ 4.5ms
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 144.92 MB/s | 414.6us | 532.2us | 714.3us | 0 |
| liteio | 128.25 MB/s | 411.7us | 917.6us | 1.4ms | 0 |
| minio | 122.02 MB/s | 488.4us | 602.3us | 986.3us | 0 |
| seaweedfs | 93.80 MB/s | 603.6us | 985.1us | 1.8ms | 0 |
| rustfs | 78.97 MB/s | 761.9us | 942.6us | 1.2ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 144.92 MB/s
liteio       ██████████████████████████ 128.25 MB/s
minio        █████████████████████████ 122.02 MB/s
seaweedfs    ███████████████████ 93.80 MB/s
rustfs       ████████████████ 78.97 MB/s
```

**Latency (P50)**
```
herd_s3      ████████████████ 414.6us
liteio       ████████████████ 411.7us
minio        ███████████████████ 488.4us
seaweedfs    ███████████████████████ 603.6us
rustfs       ██████████████████████████████ 761.9us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 504 ops/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| herd_s3 | 481 ops/s | 2.1ms | 2.1ms | 2.1ms | 0 |
| seaweedfs | 272 ops/s | 3.7ms | 3.7ms | 3.7ms | 0 |
| minio | 228 ops/s | 4.4ms | 4.4ms | 4.4ms | 0 |
| rustfs | 87 ops/s | 11.5ms | 11.5ms | 11.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 504 ops/s
herd_s3      ████████████████████████████ 481 ops/s
seaweedfs    ████████████████ 272 ops/s
minio        █████████████ 228 ops/s
rustfs       █████ 87 ops/s
```

**Latency (P50)**
```
liteio       █████ 2.0ms
herd_s3      █████ 2.1ms
seaweedfs    █████████ 3.7ms
minio        ███████████ 4.4ms
rustfs       ██████████████████████████████ 11.5ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 46 ops/s | 21.6ms | 21.6ms | 21.6ms | 0 |
| liteio | 36 ops/s | 27.9ms | 27.9ms | 27.9ms | 0 |
| seaweedfs | 31 ops/s | 31.9ms | 31.9ms | 31.9ms | 0 |
| minio | 27 ops/s | 37.7ms | 37.7ms | 37.7ms | 0 |
| rustfs | 9 ops/s | 111.5ms | 111.5ms | 111.5ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 46 ops/s
liteio       ███████████████████████ 36 ops/s
seaweedfs    ████████████████████ 31 ops/s
minio        █████████████████ 27 ops/s
rustfs       █████ 9 ops/s
```

**Latency (P50)**
```
herd_s3      █████ 21.6ms
liteio       ███████ 27.9ms
seaweedfs    ████████ 31.9ms
minio        ██████████ 37.7ms
rustfs       ██████████████████████████████ 111.5ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6 ops/s | 170.2ms | 170.2ms | 170.2ms | 0 |
| herd_s3 | 5 ops/s | 209.5ms | 209.5ms | 209.5ms | 0 |
| seaweedfs | 3 ops/s | 309.0ms | 309.0ms | 309.0ms | 0 |
| minio | 3 ops/s | 348.4ms | 348.4ms | 348.4ms | 0 |
| rustfs | 1 ops/s | 1.15s | 1.15s | 1.15s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 6 ops/s
herd_s3      ████████████████████████ 5 ops/s
seaweedfs    ████████████████ 3 ops/s
minio        ██████████████ 3 ops/s
rustfs       ████ 1 ops/s
```

**Latency (P50)**
```
liteio       ████ 170.2ms
herd_s3      █████ 209.5ms
seaweedfs    ████████ 309.0ms
minio        █████████ 348.4ms
rustfs       ██████████████████████████████ 1.15s
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1 ops/s | 1.71s | 1.71s | 1.71s | 0 |
| herd_s3 | 1 ops/s | 1.72s | 1.72s | 1.72s | 0 |
| seaweedfs | 0 ops/s | 3.45s | 3.45s | 3.45s | 0 |
| minio | 0 ops/s | 3.60s | 3.60s | 3.60s | 0 |
| rustfs | 0 ops/s | 11.33s | 11.33s | 11.33s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1 ops/s
herd_s3      █████████████████████████████ 1 ops/s
seaweedfs    ██████████████ 0 ops/s
minio        ██████████████ 0 ops/s
rustfs       ████ 0 ops/s
```

**Latency (P50)**
```
liteio       ████ 1.71s
herd_s3      ████ 1.72s
seaweedfs    █████████ 3.45s
minio        █████████ 3.60s
rustfs       ██████████████████████████████ 11.33s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 3076 ops/s | 325.1us | 325.1us | 325.1us | 0 |
| liteio | 2908 ops/s | 343.9us | 343.9us | 343.9us | 0 |
| seaweedfs | 1678 ops/s | 596.0us | 596.0us | 596.0us | 0 |
| minio | 979 ops/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| rustfs | 605 ops/s | 1.7ms | 1.7ms | 1.7ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 3076 ops/s
liteio       ████████████████████████████ 2908 ops/s
seaweedfs    ████████████████ 1678 ops/s
minio        █████████ 979 ops/s
rustfs       █████ 605 ops/s
```

**Latency (P50)**
```
herd_s3      █████ 325.1us
liteio       ██████ 343.9us
seaweedfs    ██████████ 596.0us
minio        ██████████████████ 1.0ms
rustfs       ██████████████████████████████ 1.7ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1218 ops/s | 821.0us | 821.0us | 821.0us | 0 |
| herd_s3 | 1164 ops/s | 859.4us | 859.4us | 859.4us | 0 |
| seaweedfs | 625 ops/s | 1.6ms | 1.6ms | 1.6ms | 0 |
| minio | 490 ops/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| rustfs | 155 ops/s | 6.5ms | 6.5ms | 6.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1218 ops/s
herd_s3      ████████████████████████████ 1164 ops/s
seaweedfs    ███████████████ 625 ops/s
minio        ████████████ 490 ops/s
rustfs       ███ 155 ops/s
```

**Latency (P50)**
```
liteio       ███ 821.0us
herd_s3      ███ 859.4us
seaweedfs    ███████ 1.6ms
minio        █████████ 2.0ms
rustfs       ██████████████████████████████ 6.5ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 202 ops/s | 4.9ms | 4.9ms | 4.9ms | 0 |
| liteio | 181 ops/s | 5.5ms | 5.5ms | 5.5ms | 0 |
| seaweedfs | 130 ops/s | 7.7ms | 7.7ms | 7.7ms | 0 |
| minio | 85 ops/s | 11.7ms | 11.7ms | 11.7ms | 0 |
| rustfs | 16 ops/s | 60.9ms | 60.9ms | 60.9ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 202 ops/s
liteio       ██████████████████████████ 181 ops/s
seaweedfs    ███████████████████ 130 ops/s
minio        ████████████ 85 ops/s
rustfs       ██ 16 ops/s
```

**Latency (P50)**
```
herd_s3      ██ 4.9ms
liteio       ██ 5.5ms
seaweedfs    ███ 7.7ms
minio        █████ 11.7ms
rustfs       ██████████████████████████████ 60.9ms
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 11 ops/s | 88.5ms | 88.5ms | 88.5ms | 0 |
| seaweedfs | 9 ops/s | 107.2ms | 107.2ms | 107.2ms | 0 |
| minio | 5 ops/s | 184.5ms | 184.5ms | 184.5ms | 0 |
| liteio | 5 ops/s | 192.9ms | 192.9ms | 192.9ms | 0 |
| rustfs | 1 ops/s | 776.1ms | 776.1ms | 776.1ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 11 ops/s
seaweedfs    ████████████████████████ 9 ops/s
minio        ██████████████ 5 ops/s
liteio       █████████████ 5 ops/s
rustfs       ███ 1 ops/s
```

**Latency (P50)**
```
herd_s3      ███ 88.5ms
seaweedfs    ████ 107.2ms
minio        ███████ 184.5ms
liteio       ███████ 192.9ms
rustfs       ██████████████████████████████ 776.1ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1.09 MB/s | 2.2ms | 2.2ms | 2.2ms | 0 |
| seaweedfs | 0.39 MB/s | 6.3ms | 6.3ms | 6.3ms | 0 |
| liteio | 0.37 MB/s | 6.5ms | 6.5ms | 6.5ms | 0 |
| rustfs | 0.31 MB/s | 7.9ms | 7.9ms | 7.9ms | 0 |
| minio | 0.26 MB/s | 9.3ms | 9.3ms | 9.3ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1.09 MB/s
seaweedfs    ██████████ 0.39 MB/s
liteio       ██████████ 0.37 MB/s
rustfs       ████████ 0.31 MB/s
minio        ███████ 0.26 MB/s
```

**Latency (P50)**
```
herd_s3      ███████ 2.2ms
seaweedfs    ████████████████████ 6.3ms
liteio       █████████████████████ 6.5ms
rustfs       █████████████████████████ 7.9ms
minio        ██████████████████████████████ 9.3ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1.10 MB/s | 22.3ms | 22.3ms | 22.3ms | 0 |
| liteio | 0.42 MB/s | 58.6ms | 58.6ms | 58.6ms | 0 |
| seaweedfs | 0.32 MB/s | 76.4ms | 76.4ms | 76.4ms | 0 |
| rustfs | 0.31 MB/s | 80.0ms | 80.0ms | 80.0ms | 0 |
| minio | 0.26 MB/s | 94.3ms | 94.3ms | 94.3ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1.10 MB/s
liteio       ███████████ 0.42 MB/s
seaweedfs    ████████ 0.32 MB/s
rustfs       ████████ 0.31 MB/s
minio        ███████ 0.26 MB/s
```

**Latency (P50)**
```
herd_s3      ███████ 22.3ms
liteio       ██████████████████ 58.6ms
seaweedfs    ████████████████████████ 76.4ms
rustfs       █████████████████████████ 80.0ms
minio        ██████████████████████████████ 94.3ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1.09 MB/s | 223.5ms | 223.5ms | 223.5ms | 0 |
| liteio | 0.41 MB/s | 595.9ms | 595.9ms | 595.9ms | 0 |
| seaweedfs | 0.37 MB/s | 652.4ms | 652.4ms | 652.4ms | 0 |
| rustfs | 0.29 MB/s | 839.7ms | 839.7ms | 839.7ms | 0 |
| minio | 0.22 MB/s | 1.13s | 1.13s | 1.13s | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1.09 MB/s
liteio       ███████████ 0.41 MB/s
seaweedfs    ██████████ 0.37 MB/s
rustfs       ███████ 0.29 MB/s
minio        █████ 0.22 MB/s
```

**Latency (P50)**
```
herd_s3      █████ 223.5ms
liteio       ███████████████ 595.9ms
seaweedfs    █████████████████ 652.4ms
rustfs       ██████████████████████ 839.7ms
minio        ██████████████████████████████ 1.13s
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1.06 MB/s | 2.30s | 2.30s | 2.30s | 0 |
| seaweedfs | 0.37 MB/s | 6.58s | 6.58s | 6.58s | 0 |
| liteio | 0.36 MB/s | 6.84s | 6.84s | 6.84s | 0 |
| rustfs | 0.29 MB/s | 8.44s | 8.44s | 8.44s | 0 |
| minio | 0.25 MB/s | 9.82s | 9.82s | 9.82s | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1.06 MB/s
seaweedfs    ██████████ 0.37 MB/s
liteio       ██████████ 0.36 MB/s
rustfs       ████████ 0.29 MB/s
minio        ███████ 0.25 MB/s
```

**Latency (P50)**
```
herd_s3      ███████ 2.30s
seaweedfs    ████████████████████ 6.58s
liteio       ████████████████████ 6.84s
rustfs       █████████████████████████ 8.44s
minio        ██████████████████████████████ 9.82s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 5808 ops/s | 161.4us | 232.0us | 342.8us | 0 |
| liteio | 4341 ops/s | 189.8us | 441.0us | 760.3us | 0 |
| minio | 4181 ops/s | 229.7us | 282.9us | 388.2us | 0 |
| seaweedfs | 3625 ops/s | 237.6us | 460.8us | 860.8us | 0 |
| rustfs | 2429 ops/s | 393.8us | 519.5us | 774.7us | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 5808 ops/s
liteio       ██████████████████████ 4341 ops/s
minio        █████████████████████ 4181 ops/s
seaweedfs    ██████████████████ 3625 ops/s
rustfs       ████████████ 2429 ops/s
```

**Latency (P50)**
```
herd_s3      ████████████ 161.4us
liteio       ██████████████ 189.8us
minio        █████████████████ 229.7us
seaweedfs    ██████████████████ 237.6us
rustfs       ██████████████████████████████ 393.8us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 173.36 MB/s | 568.0ms | 568.0ms | 568.0ms | 0 |
| seaweedfs | 169.88 MB/s | 594.0ms | 594.0ms | 594.0ms | 0 |
| minio | 167.89 MB/s | 586.2ms | 586.2ms | 586.2ms | 0 |
| rustfs | 162.79 MB/s | 600.2ms | 600.2ms | 600.2ms | 0 |
| liteio | 151.96 MB/s | 573.8ms | 573.8ms | 573.8ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 173.36 MB/s
seaweedfs    █████████████████████████████ 169.88 MB/s
minio        █████████████████████████████ 167.89 MB/s
rustfs       ████████████████████████████ 162.79 MB/s
liteio       ██████████████████████████ 151.96 MB/s
```

**Latency (P50)**
```
herd_s3      ████████████████████████████ 568.0ms
seaweedfs    █████████████████████████████ 594.0ms
minio        █████████████████████████████ 586.2ms
rustfs       ██████████████████████████████ 600.2ms
liteio       ████████████████████████████ 573.8ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 181.83 MB/s | 53.1ms | 67.2ms | 71.1ms | 0 |
| herd_s3 | 170.21 MB/s | 58.1ms | 65.5ms | 66.7ms | 0 |
| minio | 168.11 MB/s | 58.9ms | 65.0ms | 66.1ms | 0 |
| rustfs | 159.86 MB/s | 61.8ms | 69.6ms | 71.8ms | 0 |
| seaweedfs | 129.48 MB/s | 75.9ms | 84.7ms | 86.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 181.83 MB/s
herd_s3      ████████████████████████████ 170.21 MB/s
minio        ███████████████████████████ 168.11 MB/s
rustfs       ██████████████████████████ 159.86 MB/s
seaweedfs    █████████████████████ 129.48 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 53.1ms
herd_s3      ██████████████████████ 58.1ms
minio        ███████████████████████ 58.9ms
rustfs       ████████████████████████ 61.8ms
seaweedfs    ██████████████████████████████ 75.9ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 4.81 MB/s | 179.6us | 304.7us | 640.8us | 0 |
| liteio | 1.58 MB/s | 590.5us | 786.6us | 1.2ms | 0 |
| minio | 1.43 MB/s | 655.3us | 846.7us | 1.1ms | 0 |
| seaweedfs | 1.01 MB/s | 782.5us | 2.0ms | 3.4ms | 0 |
| rustfs | 1.01 MB/s | 821.2us | 1.6ms | 3.8ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 4.81 MB/s
liteio       █████████ 1.58 MB/s
minio        ████████ 1.43 MB/s
seaweedfs    ██████ 1.01 MB/s
rustfs       ██████ 1.01 MB/s
```

**Latency (P50)**
```
herd_s3      ██████ 179.6us
liteio       █████████████████████ 590.5us
minio        ███████████████████████ 655.3us
seaweedfs    ████████████████████████████ 782.5us
rustfs       ██████████████████████████████ 821.2us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 181.99 MB/s | 5.0ms | 9.0ms | 11.6ms | 0 |
| liteio | 154.36 MB/s | 5.8ms | 10.2ms | 13.4ms | 0 |
| minio | 143.25 MB/s | 6.9ms | 9.9ms | 11.1ms | 0 |
| rustfs | 139.65 MB/s | 7.1ms | 9.3ms | 11.5ms | 0 |
| seaweedfs | 103.63 MB/s | 9.3ms | 11.2ms | 13.3ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 181.99 MB/s
liteio       █████████████████████████ 154.36 MB/s
minio        ███████████████████████ 143.25 MB/s
rustfs       ███████████████████████ 139.65 MB/s
seaweedfs    █████████████████ 103.63 MB/s
```

**Latency (P50)**
```
herd_s3      ███████████████ 5.0ms
liteio       ██████████████████ 5.8ms
minio        ██████████████████████ 6.9ms
rustfs       ██████████████████████ 7.1ms
seaweedfs    ██████████████████████████████ 9.3ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 139.86 MB/s | 424.4us | 535.3us | 1.1ms | 0 |
| liteio | 66.32 MB/s | 883.8us | 1.2ms | 1.8ms | 0 |
| minio | 57.21 MB/s | 979.1us | 1.5ms | 2.9ms | 0 |
| rustfs | 55.36 MB/s | 1.1ms | 1.3ms | 1.7ms | 0 |
| seaweedfs | 36.48 MB/s | 1.4ms | 3.5ms | 5.2ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 139.86 MB/s
liteio       ██████████████ 66.32 MB/s
minio        ████████████ 57.21 MB/s
rustfs       ███████████ 55.36 MB/s
seaweedfs    ███████ 36.48 MB/s
```

**Latency (P50)**
```
herd_s3      █████████ 424.4us
liteio       ███████████████████ 883.8us
minio        █████████████████████ 979.1us
rustfs       ███████████████████████ 1.1ms
seaweedfs    ██████████████████████████████ 1.4ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| herd_s3 | 1.829GiB / 7.653GiB | 1872.9 MB | - | 6.7% | 1835.0 MB | 1.09MB / 1.79GB |
| liteio | 1.199GiB / 7.653GiB | 1227.8 MB | - | 5.4% | 2185.2 MB | 238kB / 2.8GB |
| minio | 1.08GiB / 7.653GiB | 1105.9 MB | - | 0.0% | 2228.0 MB | 1.02MB / 2.23GB |
| rustfs | 938.9MiB / 7.653GiB | 938.9 MB | - | 1.4% | 2070.0 MB | 2.59MB / 1.68GB |
| seaweedfs | 215.9MiB / 7.653GiB | 215.9 MB | - | 1.4% | 2467.0 MB | 0B / 1.5GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** herd_s3
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
