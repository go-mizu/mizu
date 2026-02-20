# Storage Benchmark Report

**Generated:** 2026-02-20T14:42:39+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** herd_s3 (won 23/48 benchmarks, 48%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | herd_s3 | 23 | 48% |
| 2 | liteio | 15 | 31% |
| 3 | minio | 6 | 12% |
| 4 | seaweedfs | 3 | 6% |
| 5 | rustfs | 1 | 2% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | herd_s3 | 3.5 MB/s | +63% vs minio |
| Small Write (1KB) | herd_s3 | 1.7 MB/s | +31% vs seaweedfs |
| Large Read (100MB) | rustfs | 202.8 MB/s | +14% vs minio |
| Large Write (100MB) | seaweedfs | 157.5 MB/s | close |
| Delete | liteio | 4.2K ops/s | +39% vs herd_s3 |
| Stat | herd_s3 | 3.9K ops/s | +28% vs minio |
| List (100 objects) | herd_s3 | 1.2K ops/s | +31% vs liteio |
| Copy | herd_s3 | 1.8 MB/s | +46% vs liteio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **seaweedfs** | 157 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **rustfs** | 203 MB/s | Best for streaming, CDN |
| Small File Operations | **herd_s3** | 2644 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **herd_s3** | - | Best for multi-user apps |
| Memory Constrained | **seaweedfs** | 152 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| herd_s3 | 117.2 | 147.9 | 867.3ms | 672.3ms |
| liteio | 86.9 | 138.0 | 947.9ms | 730.6ms |
| minio | 152.0 | 177.7 | 667.8ms | 622.8ms |
| rustfs | 119.0 | 202.8 | 846.6ms | 466.4ms |
| seaweedfs | 157.5 | 147.0 | 613.7ms | 660.5ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| herd_s3 | 1725 | 3563 | 507.8us | 242.2us |
| liteio | 957 | 684 | 739.5us | 979.0us |
| minio | 444 | 2188 | 1.2ms | 391.0us |
| rustfs | 716 | 1234 | 1.1ms | 580.2us |
| seaweedfs | 1320 | 1513 | 672.5us | 490.4us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| herd_s3 | 3929 | 1234 | 2986 |
| liteio | 2974 | 941 | 4153 |
| minio | 3067 | 486 | 2157 |
| rustfs | 2141 | 131 | 836 |
| seaweedfs | 1020 | 449 | 2139 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| herd_s3 | 1.49 | 0.75 | 0.28 | 0.14 | 0.14 | 0.05 |
| liteio | 1.25 | 0.41 | 0.24 | 0.13 | 0.09 | 0.05 |
| minio | 0.67 | 0.27 | 0.10 | 0.06 | 0.02 | 0.01 |
| rustfs | 1.03 | 0.22 | 0.09 | 0.04 | 0.02 | 0.01 |
| seaweedfs | 1.00 | 0.33 | 0.17 | 0.10 | 0.04 | 0.03 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| herd_s3 | 1.86 | 0.69 | 0.28 | 0.28 | 0.15 | 0.07 |
| liteio | 3.54 | 0.96 | 0.46 | 0.32 | 0.16 | 0.09 |
| minio | 2.69 | 0.79 | 0.41 | 0.21 | 0.08 | 0.06 |
| rustfs | 1.53 | 0.25 | 0.10 | 0.05 | 0.02 | 0.01 |
| seaweedfs | 1.57 | 0.54 | 0.30 | 0.11 | 0.09 | 0.04 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| herd_s3 | 4.2ms | 46.6ms | 536.3ms | 7.74s |
| liteio | 34.1ms | 173.2ms | 1.44s | 17.56s |
| minio | 9.3ms | 98.9ms | 1.94s | 13.78s |
| rustfs | 9.6ms | 94.5ms | 1.02s | 8.79s |
| seaweedfs | 8.7ms | 83.5ms | 869.0ms | 9.62s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| herd_s3 | 609.3us | 1.3ms | 7.5ms | 171.1ms |
| liteio | 988.3us | 1.5ms | 13.6ms | 386.5ms |
| minio | 935.6us | 2.1ms | 15.3ms | 208.5ms |
| rustfs | 2.2ms | 7.7ms | 99.3ms | 817.1ms |
| seaweedfs | 982.7us | 2.6ms | 14.3ms | 100.5ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| herd_s3 | 1651.7 MB | 4.9% |
| liteio | 847.9 MB | 6.7% |
| minio | 958.0 MB | 10.6% |
| rustfs | 729.2 MB | 0.0% |
| seaweedfs | 152.4 MB | 0.2% |

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
| herd_s3 | 1.76 MB/s | 487.0us | 987.9us | 1.5ms | 0 |
| liteio | 1.21 MB/s | 697.7us | 1.5ms | 2.5ms | 0 |
| rustfs | 0.57 MB/s | 1.1ms | 4.0ms | 6.6ms | 0 |
| seaweedfs | 0.49 MB/s | 1.7ms | 3.6ms | 6.6ms | 0 |
| minio | 0.45 MB/s | 1.3ms | 3.1ms | 8.8ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1.76 MB/s
liteio       ████████████████████ 1.21 MB/s
rustfs       █████████ 0.57 MB/s
seaweedfs    ████████ 0.49 MB/s
minio        ███████ 0.45 MB/s
```

**Latency (P50)**
```
herd_s3      ████████ 487.0us
liteio       ████████████ 697.7us
rustfs       ████████████████████ 1.1ms
seaweedfs    ██████████████████████████████ 1.7ms
minio        ███████████████████████ 1.3ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4153 ops/s | 217.5us | 380.4us | 543.6us | 0 |
| herd_s3 | 2986 ops/s | 284.8us | 600.3us | 1.2ms | 0 |
| minio | 2157 ops/s | 341.8us | 650.2us | 1.3ms | 0 |
| seaweedfs | 2139 ops/s | 411.1us | 793.2us | 1.3ms | 0 |
| rustfs | 836 ops/s | 1.1ms | 1.8ms | 3.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4153 ops/s
herd_s3      █████████████████████ 2986 ops/s
minio        ███████████████ 2157 ops/s
seaweedfs    ███████████████ 2139 ops/s
rustfs       ██████ 836 ops/s
```

**Latency (P50)**
```
liteio       ██████ 217.5us
herd_s3      ███████ 284.8us
minio        █████████ 341.8us
seaweedfs    ███████████ 411.1us
rustfs       ██████████████████████████████ 1.1ms
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.22 MB/s | 371.8us | 747.0us | 1.2ms | 0 |
| seaweedfs | 0.11 MB/s | 799.2us | 1.4ms | 2.5ms | 0 |
| rustfs | 0.07 MB/s | 1.1ms | 2.9ms | 4.1ms | 0 |
| liteio | 0.06 MB/s | 1.5ms | 2.4ms | 3.7ms | 0 |
| minio | 0.06 MB/s | 1.4ms | 3.4ms | 4.4ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.22 MB/s
seaweedfs    ██████████████ 0.11 MB/s
rustfs       ████████ 0.07 MB/s
liteio       ████████ 0.06 MB/s
minio        ███████ 0.06 MB/s
```

**Latency (P50)**
```
herd_s3      ███████ 371.8us
seaweedfs    ████████████████ 799.2us
rustfs       ███████████████████████ 1.1ms
liteio       ██████████████████████████████ 1.5ms
minio        ████████████████████████████ 1.4ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 2118 ops/s | 365.2us | 1.0ms | 1.9ms | 0 |
| seaweedfs | 1614 ops/s | 536.3us | 1.1ms | 1.7ms | 0 |
| liteio | 852 ops/s | 1.1ms | 1.7ms | 2.5ms | 0 |
| minio | 548 ops/s | 1.5ms | 3.4ms | 7.4ms | 0 |
| rustfs | 543 ops/s | 1.4ms | 3.9ms | 7.3ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 2118 ops/s
seaweedfs    ██████████████████████ 1614 ops/s
liteio       ████████████ 852 ops/s
minio        ███████ 548 ops/s
rustfs       ███████ 543 ops/s
```

**Latency (P50)**
```
herd_s3      ███████ 365.2us
seaweedfs    ██████████ 536.3us
liteio       ██████████████████████ 1.1ms
minio        ██████████████████████████████ 1.5ms
rustfs       ███████████████████████████ 1.4ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.20 MB/s | 412.7us | 916.4us | 1.6ms | 0 |
| seaweedfs | 0.08 MB/s | 1.0ms | 1.9ms | 3.6ms | 0 |
| rustfs | 0.08 MB/s | 1.0ms | 2.1ms | 3.2ms | 0 |
| minio | 0.06 MB/s | 1.3ms | 2.8ms | 5.3ms | 0 |
| liteio | 0.05 MB/s | 1.5ms | 3.0ms | 9.3ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.20 MB/s
seaweedfs    ████████████ 0.08 MB/s
rustfs       ████████████ 0.08 MB/s
minio        █████████ 0.06 MB/s
liteio       ███████ 0.05 MB/s
```

**Latency (P50)**
```
herd_s3      ████████ 412.7us
seaweedfs    ████████████████████ 1.0ms
rustfs       ████████████████████ 1.0ms
minio        █████████████████████████ 1.3ms
liteio       ██████████████████████████████ 1.5ms
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1234 ops/s | 766.1us | 1.1ms | 1.5ms | 0 |
| liteio | 941 ops/s | 1.0ms | 1.3ms | 1.7ms | 0 |
| minio | 486 ops/s | 1.9ms | 3.1ms | 5.8ms | 0 |
| seaweedfs | 449 ops/s | 2.0ms | 3.4ms | 5.0ms | 0 |
| rustfs | 131 ops/s | 7.0ms | 10.3ms | 14.3ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1234 ops/s
liteio       ██████████████████████ 941 ops/s
minio        ███████████ 486 ops/s
seaweedfs    ██████████ 449 ops/s
rustfs       ███ 131 ops/s
```

**Latency (P50)**
```
herd_s3      ███ 766.1us
liteio       ████ 1.0ms
minio        ███████ 1.9ms
seaweedfs    ████████ 2.0ms
rustfs       ██████████████████████████████ 7.0ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.47 MB/s | 23.7ms | 62.1ms | 279.6ms | 0 |
| herd_s3 | 0.30 MB/s | 30.5ms | 272.7ms | 330.9ms | 0 |
| seaweedfs | 0.22 MB/s | 46.7ms | 211.3ms | 363.5ms | 0 |
| rustfs | 0.17 MB/s | 84.7ms | 145.0ms | 258.0ms | 0 |
| minio | 0.14 MB/s | 48.0ms | 396.4ms | 903.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.47 MB/s
herd_s3      ███████████████████ 0.30 MB/s
seaweedfs    █████████████ 0.22 MB/s
rustfs       ██████████ 0.17 MB/s
minio        █████████ 0.14 MB/s
```

**Latency (P50)**
```
liteio       ████████ 23.7ms
herd_s3      ██████████ 30.5ms
seaweedfs    ████████████████ 46.7ms
rustfs       ██████████████████████████████ 84.7ms
minio        ████████████████ 48.0ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.56 MB/s | 26.2ms | 42.1ms | 56.0ms | 0 |
| minio | 0.48 MB/s | 26.1ms | 81.2ms | 130.2ms | 0 |
| herd_s3 | 0.44 MB/s | 32.1ms | 62.3ms | 80.6ms | 0 |
| seaweedfs | 0.39 MB/s | 36.0ms | 76.7ms | 102.6ms | 0 |
| rustfs | 0.18 MB/s | 82.8ms | 145.4ms | 172.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.56 MB/s
minio        █████████████████████████ 0.48 MB/s
herd_s3      ███████████████████████ 0.44 MB/s
seaweedfs    █████████████████████ 0.39 MB/s
rustfs       █████████ 0.18 MB/s
```

**Latency (P50)**
```
liteio       █████████ 26.2ms
minio        █████████ 26.1ms
herd_s3      ███████████ 32.1ms
seaweedfs    █████████████ 36.0ms
rustfs       ██████████████████████████████ 82.8ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.37 MB/s | 30.4ms | 66.4ms | 475.5ms | 0 |
| herd_s3 | 0.35 MB/s | 24.4ms | 158.3ms | 626.8ms | 0 |
| seaweedfs | 0.29 MB/s | 36.1ms | 69.8ms | 838.5ms | 0 |
| minio | 0.15 MB/s | 99.7ms | 229.9ms | 307.7ms | 0 |
| rustfs | 0.14 MB/s | 117.6ms | 172.9ms | 194.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.37 MB/s
herd_s3      ████████████████████████████ 0.35 MB/s
seaweedfs    ███████████████████████ 0.29 MB/s
minio        ████████████ 0.15 MB/s
rustfs       ███████████ 0.14 MB/s
```

**Latency (P50)**
```
liteio       ███████ 30.4ms
herd_s3      ██████ 24.4ms
seaweedfs    █████████ 36.1ms
minio        █████████████████████████ 99.7ms
rustfs       ██████████████████████████████ 117.6ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 113.76 MB/s | 129.4ms | 147.4ms | 147.4ms | 0 |
| liteio | 96.03 MB/s | 147.2ms | 172.5ms | 172.5ms | 0 |
| rustfs | 93.17 MB/s | 159.6ms | 203.4ms | 203.4ms | 0 |
| herd_s3 | 68.88 MB/s | 218.1ms | 250.2ms | 250.2ms | 0 |
| seaweedfs | 65.25 MB/s | 218.4ms | 241.8ms | 241.8ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 113.76 MB/s
liteio       █████████████████████████ 96.03 MB/s
rustfs       ████████████████████████ 93.17 MB/s
herd_s3      ██████████████████ 68.88 MB/s
seaweedfs    █████████████████ 65.25 MB/s
```

**Latency (P50)**
```
minio        █████████████████ 129.4ms
liteio       ████████████████████ 147.2ms
rustfs       █████████████████████ 159.6ms
herd_s3      █████████████████████████████ 218.1ms
seaweedfs    ██████████████████████████████ 218.4ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.54 MB/s | 250.6us | 412.1us | 594.0us | 0 |
| minio | 2.69 MB/s | 343.2us | 457.5us | 691.3us | 0 |
| herd_s3 | 1.86 MB/s | 460.6us | 928.6us | 1.3ms | 0 |
| seaweedfs | 1.57 MB/s | 543.8us | 1.0ms | 2.0ms | 0 |
| rustfs | 1.53 MB/s | 596.2us | 804.7us | 1.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.54 MB/s
minio        ██████████████████████ 2.69 MB/s
herd_s3      ███████████████ 1.86 MB/s
seaweedfs    █████████████ 1.57 MB/s
rustfs       ████████████ 1.53 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 250.6us
minio        █████████████████ 343.2us
herd_s3      ███████████████████████ 460.6us
seaweedfs    ███████████████████████████ 543.8us
rustfs       ██████████████████████████████ 596.2us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.96 MB/s | 903.8us | 1.7ms | 2.7ms | 0 |
| minio | 0.79 MB/s | 1.1ms | 2.3ms | 4.1ms | 0 |
| herd_s3 | 0.69 MB/s | 1.1ms | 3.1ms | 5.8ms | 0 |
| seaweedfs | 0.54 MB/s | 1.7ms | 3.0ms | 4.7ms | 0 |
| rustfs | 0.25 MB/s | 3.7ms | 6.3ms | 9.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.96 MB/s
minio        ████████████████████████ 0.79 MB/s
herd_s3      █████████████████████ 0.69 MB/s
seaweedfs    ████████████████ 0.54 MB/s
rustfs       ███████ 0.25 MB/s
```

**Latency (P50)**
```
liteio       ███████ 903.8us
minio        ████████ 1.1ms
herd_s3      █████████ 1.1ms
seaweedfs    █████████████ 1.7ms
rustfs       ██████████████████████████████ 3.7ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.16 MB/s | 5.6ms | 9.9ms | 13.2ms | 0 |
| herd_s3 | 0.15 MB/s | 5.7ms | 11.6ms | 18.9ms | 0 |
| seaweedfs | 0.09 MB/s | 10.2ms | 21.2ms | 35.7ms | 0 |
| minio | 0.08 MB/s | 9.2ms | 30.1ms | 53.4ms | 0 |
| rustfs | 0.02 MB/s | 41.8ms | 59.7ms | 75.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.16 MB/s
herd_s3      ███████████████████████████ 0.15 MB/s
seaweedfs    ███████████████ 0.09 MB/s
minio        ██████████████ 0.08 MB/s
rustfs       ████ 0.02 MB/s
```

**Latency (P50)**
```
liteio       ███ 5.6ms
herd_s3      ████ 5.7ms
seaweedfs    ███████ 10.2ms
minio        ██████ 9.2ms
rustfs       ██████████████████████████████ 41.8ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.09 MB/s | 11.2ms | 17.4ms | 22.8ms | 0 |
| herd_s3 | 0.07 MB/s | 13.8ms | 22.4ms | 37.5ms | 0 |
| minio | 0.06 MB/s | 13.0ms | 35.4ms | 51.0ms | 0 |
| seaweedfs | 0.04 MB/s | 18.8ms | 52.7ms | 86.7ms | 0 |
| rustfs | 0.01 MB/s | 83.4ms | 97.8ms | 106.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.09 MB/s
herd_s3      ████████████████████████ 0.07 MB/s
minio        ██████████████████████ 0.06 MB/s
seaweedfs    ██████████████ 0.04 MB/s
rustfs       ████ 0.01 MB/s
```

**Latency (P50)**
```
liteio       ████ 11.2ms
herd_s3      ████ 13.8ms
minio        ████ 13.0ms
seaweedfs    ██████ 18.8ms
rustfs       ██████████████████████████████ 83.4ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.46 MB/s | 1.9ms | 3.8ms | 7.4ms | 0 |
| minio | 0.41 MB/s | 2.2ms | 4.1ms | 6.7ms | 0 |
| seaweedfs | 0.30 MB/s | 3.0ms | 5.5ms | 8.1ms | 0 |
| herd_s3 | 0.28 MB/s | 2.5ms | 8.9ms | 18.8ms | 0 |
| rustfs | 0.10 MB/s | 9.7ms | 13.9ms | 17.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.46 MB/s
minio        ██████████████████████████ 0.41 MB/s
seaweedfs    ███████████████████ 0.30 MB/s
herd_s3      █████████████████ 0.28 MB/s
rustfs       ██████ 0.10 MB/s
```

**Latency (P50)**
```
liteio       █████ 1.9ms
minio        ██████ 2.2ms
seaweedfs    █████████ 3.0ms
herd_s3      ███████ 2.5ms
rustfs       ██████████████████████████████ 9.7ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.32 MB/s | 2.9ms | 4.7ms | 6.2ms | 0 |
| herd_s3 | 0.28 MB/s | 3.3ms | 5.8ms | 8.2ms | 0 |
| minio | 0.21 MB/s | 3.9ms | 9.9ms | 16.6ms | 0 |
| seaweedfs | 0.11 MB/s | 7.0ms | 20.6ms | 31.1ms | 0 |
| rustfs | 0.05 MB/s | 20.5ms | 26.7ms | 31.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.32 MB/s
herd_s3      █████████████████████████ 0.28 MB/s
minio        ███████████████████ 0.21 MB/s
seaweedfs    ██████████ 0.11 MB/s
rustfs       ████ 0.05 MB/s
```

**Latency (P50)**
```
liteio       ████ 2.9ms
herd_s3      ████ 3.3ms
minio        █████ 3.9ms
seaweedfs    ██████████ 7.0ms
rustfs       ██████████████████████████████ 20.5ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1.49 MB/s | 508.1us | 1.4ms | 2.8ms | 0 |
| liteio | 1.25 MB/s | 735.2us | 1.0ms | 1.7ms | 0 |
| rustfs | 1.03 MB/s | 888.0us | 1.3ms | 2.2ms | 0 |
| seaweedfs | 1.00 MB/s | 871.6us | 1.5ms | 2.8ms | 0 |
| minio | 0.67 MB/s | 947.2us | 1.8ms | 17.5ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1.49 MB/s
liteio       █████████████████████████ 1.25 MB/s
rustfs       ████████████████████ 1.03 MB/s
seaweedfs    ████████████████████ 1.00 MB/s
minio        █████████████ 0.67 MB/s
```

**Latency (P50)**
```
herd_s3      ████████████████ 508.1us
liteio       ███████████████████████ 735.2us
rustfs       ████████████████████████████ 888.0us
seaweedfs    ███████████████████████████ 871.6us
minio        ██████████████████████████████ 947.2us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.75 MB/s | 1.2ms | 2.2ms | 3.6ms | 0 |
| liteio | 0.41 MB/s | 2.3ms | 3.3ms | 4.5ms | 0 |
| seaweedfs | 0.33 MB/s | 2.8ms | 4.6ms | 6.2ms | 0 |
| minio | 0.27 MB/s | 3.5ms | 5.7ms | 8.1ms | 0 |
| rustfs | 0.22 MB/s | 4.0ms | 7.2ms | 11.5ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.75 MB/s
liteio       ████████████████ 0.41 MB/s
seaweedfs    █████████████ 0.33 MB/s
minio        ██████████ 0.27 MB/s
rustfs       ████████ 0.22 MB/s
```

**Latency (P50)**
```
herd_s3      ████████ 1.2ms
liteio       █████████████████ 2.3ms
seaweedfs    ████████████████████ 2.8ms
minio        █████████████████████████ 3.5ms
rustfs       ██████████████████████████████ 4.0ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.14 MB/s | 6.4ms | 13.2ms | 20.0ms | 0 |
| liteio | 0.09 MB/s | 10.5ms | 16.8ms | 23.1ms | 0 |
| seaweedfs | 0.04 MB/s | 22.1ms | 54.7ms | 75.4ms | 0 |
| rustfs | 0.02 MB/s | 44.1ms | 52.6ms | 67.2ms | 0 |
| minio | 0.02 MB/s | 38.7ms | 113.3ms | 187.5ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.14 MB/s
liteio       ███████████████████ 0.09 MB/s
seaweedfs    ████████ 0.04 MB/s
rustfs       ████ 0.02 MB/s
minio        ████ 0.02 MB/s
```

**Latency (P50)**
```
herd_s3      ████ 6.4ms
liteio       ███████ 10.5ms
seaweedfs    ███████████████ 22.1ms
rustfs       ██████████████████████████████ 44.1ms
minio        ██████████████████████████ 38.7ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.05 MB/s | 16.5ms | 39.0ms | 61.8ms | 0 |
| liteio | 0.05 MB/s | 17.3ms | 30.0ms | 56.1ms | 0 |
| seaweedfs | 0.03 MB/s | 30.4ms | 53.1ms | 70.9ms | 0 |
| rustfs | 0.01 MB/s | 90.8ms | 124.0ms | 137.6ms | 0 |
| minio | 0.01 MB/s | 110.0ms | 377.2ms | 490.0ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.05 MB/s
liteio       ████████████████████████████ 0.05 MB/s
seaweedfs    ████████████████ 0.03 MB/s
rustfs       █████ 0.01 MB/s
minio        ███ 0.01 MB/s
```

**Latency (P50)**
```
herd_s3      ████ 16.5ms
liteio       ████ 17.3ms
seaweedfs    ████████ 30.4ms
rustfs       ████████████████████████ 90.8ms
minio        ██████████████████████████████ 110.0ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.28 MB/s | 2.6ms | 8.5ms | 18.9ms | 0 |
| liteio | 0.24 MB/s | 3.8ms | 5.4ms | 7.6ms | 0 |
| seaweedfs | 0.17 MB/s | 5.1ms | 9.7ms | 16.3ms | 0 |
| minio | 0.10 MB/s | 8.8ms | 19.9ms | 27.7ms | 0 |
| rustfs | 0.09 MB/s | 10.2ms | 15.6ms | 21.8ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.28 MB/s
liteio       ██████████████████████████ 0.24 MB/s
seaweedfs    ██████████████████ 0.17 MB/s
minio        ██████████ 0.10 MB/s
rustfs       █████████ 0.09 MB/s
```

**Latency (P50)**
```
herd_s3      ███████ 2.6ms
liteio       ███████████ 3.8ms
seaweedfs    ███████████████ 5.1ms
minio        █████████████████████████ 8.8ms
rustfs       ██████████████████████████████ 10.2ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.14 MB/s | 5.2ms | 17.8ms | 30.1ms | 0 |
| liteio | 0.13 MB/s | 6.7ms | 13.5ms | 20.3ms | 0 |
| seaweedfs | 0.10 MB/s | 9.3ms | 18.1ms | 24.6ms | 0 |
| minio | 0.06 MB/s | 15.3ms | 37.1ms | 49.6ms | 0 |
| rustfs | 0.04 MB/s | 21.8ms | 32.5ms | 45.7ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.14 MB/s
liteio       ███████████████████████████ 0.13 MB/s
seaweedfs    ████████████████████ 0.10 MB/s
minio        ███████████ 0.06 MB/s
rustfs       ████████ 0.04 MB/s
```

**Latency (P50)**
```
herd_s3      ███████ 5.2ms
liteio       █████████ 6.7ms
seaweedfs    ████████████ 9.3ms
minio        ████████████████████ 15.3ms
rustfs       ██████████████████████████████ 21.8ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 143.09 MB/s | 1.7ms | 2.3ms | 2.8ms | 0 |
| herd_s3 | 108.31 MB/s | 2.2ms | 3.1ms | 4.6ms | 0 |
| seaweedfs | 98.25 MB/s | 2.3ms | 4.3ms | 5.7ms | 0 |
| rustfs | 81.34 MB/s | 2.6ms | 4.3ms | 10.1ms | 0 |
| minio | 67.55 MB/s | 2.4ms | 9.1ms | 21.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 143.09 MB/s
herd_s3      ██████████████████████ 108.31 MB/s
seaweedfs    ████████████████████ 98.25 MB/s
rustfs       █████████████████ 81.34 MB/s
minio        ██████████████ 67.55 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 1.7ms
herd_s3      █████████████████████████ 2.2ms
seaweedfs    ██████████████████████████ 2.3ms
rustfs       ██████████████████████████████ 2.6ms
minio        ████████████████████████████ 2.4ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 136.90 MB/s | 1.7ms | 2.5ms | 3.1ms | 0 |
| minio | 133.79 MB/s | 1.7ms | 2.9ms | 4.3ms | 0 |
| herd_s3 | 103.76 MB/s | 2.3ms | 3.5ms | 4.4ms | 0 |
| seaweedfs | 101.50 MB/s | 2.2ms | 4.2ms | 5.6ms | 0 |
| rustfs | 64.83 MB/s | 3.1ms | 6.5ms | 15.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 136.90 MB/s
minio        █████████████████████████████ 133.79 MB/s
herd_s3      ██████████████████████ 103.76 MB/s
seaweedfs    ██████████████████████ 101.50 MB/s
rustfs       ██████████████ 64.83 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 1.7ms
minio        ████████████████ 1.7ms
herd_s3      █████████████████████ 2.3ms
seaweedfs    █████████████████████ 2.2ms
rustfs       ██████████████████████████████ 3.1ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 128.03 MB/s | 1.9ms | 2.6ms | 3.1ms | 0 |
| seaweedfs | 106.67 MB/s | 2.2ms | 3.2ms | 4.6ms | 0 |
| herd_s3 | 98.04 MB/s | 2.5ms | 3.4ms | 4.2ms | 0 |
| rustfs | 75.31 MB/s | 3.0ms | 5.5ms | 7.0ms | 0 |
| minio | 30.56 MB/s | 2.3ms | 38.9ms | 70.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 128.03 MB/s
seaweedfs    ████████████████████████ 106.67 MB/s
herd_s3      ██████████████████████ 98.04 MB/s
rustfs       █████████████████ 75.31 MB/s
minio        ███████ 30.56 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 1.9ms
seaweedfs    ██████████████████████ 2.2ms
herd_s3      ████████████████████████ 2.5ms
rustfs       ██████████████████████████████ 3.0ms
minio        ███████████████████████ 2.3ms
```

### Read/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 202.82 MB/s | 466.4ms | 466.4ms | 466.4ms | 0 |
| minio | 177.73 MB/s | 622.8ms | 622.8ms | 622.8ms | 0 |
| herd_s3 | 147.90 MB/s | 672.3ms | 672.3ms | 672.3ms | 0 |
| seaweedfs | 146.99 MB/s | 660.5ms | 660.5ms | 660.5ms | 0 |
| liteio | 138.05 MB/s | 730.6ms | 730.6ms | 730.6ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 202.82 MB/s
minio        ██████████████████████████ 177.73 MB/s
herd_s3      █████████████████████ 147.90 MB/s
seaweedfs    █████████████████████ 146.99 MB/s
liteio       ████████████████████ 138.05 MB/s
```

**Latency (P50)**
```
rustfs       ███████████████████ 466.4ms
minio        █████████████████████████ 622.8ms
herd_s3      ███████████████████████████ 672.3ms
seaweedfs    ███████████████████████████ 660.5ms
liteio       ██████████████████████████████ 730.6ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 234.85 MB/s | 40.8ms | 49.6ms | 50.9ms | 0 |
| rustfs | 163.68 MB/s | 55.7ms | 89.0ms | 108.5ms | 0 |
| seaweedfs | 155.02 MB/s | 65.4ms | 69.4ms | 76.3ms | 0 |
| herd_s3 | 148.13 MB/s | 60.1ms | 83.6ms | 95.6ms | 0 |
| liteio | 123.40 MB/s | 80.7ms | 90.9ms | 103.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 234.85 MB/s
rustfs       ████████████████████ 163.68 MB/s
seaweedfs    ███████████████████ 155.02 MB/s
herd_s3      ██████████████████ 148.13 MB/s
liteio       ███████████████ 123.40 MB/s
```

**Latency (P50)**
```
minio        ███████████████ 40.8ms
rustfs       ████████████████████ 55.7ms
seaweedfs    ████████████████████████ 65.4ms
herd_s3      ██████████████████████ 60.1ms
liteio       ██████████████████████████████ 80.7ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 3.48 MB/s | 242.2us | 468.8us | 906.8us | 0 |
| minio | 2.14 MB/s | 391.0us | 831.2us | 1.4ms | 0 |
| seaweedfs | 1.48 MB/s | 490.4us | 1.5ms | 3.4ms | 0 |
| rustfs | 1.20 MB/s | 580.2us | 2.0ms | 3.7ms | 0 |
| liteio | 0.67 MB/s | 979.0us | 3.9ms | 8.8ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 3.48 MB/s
minio        ██████████████████ 2.14 MB/s
seaweedfs    ████████████ 1.48 MB/s
rustfs       ██████████ 1.20 MB/s
liteio       █████ 0.67 MB/s
```

**Latency (P50)**
```
herd_s3      ███████ 242.2us
minio        ███████████ 391.0us
seaweedfs    ███████████████ 490.4us
rustfs       █████████████████ 580.2us
liteio       ██████████████████████████████ 979.0us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 218.19 MB/s | 4.4ms | 6.0ms | 7.6ms | 0 |
| herd_s3 | 145.28 MB/s | 6.4ms | 9.3ms | 11.6ms | 0 |
| seaweedfs | 142.50 MB/s | 6.7ms | 9.3ms | 11.5ms | 0 |
| rustfs | 136.65 MB/s | 6.1ms | 13.7ms | 18.6ms | 0 |
| liteio | 119.03 MB/s | 7.7ms | 12.2ms | 18.9ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 218.19 MB/s
herd_s3      ███████████████████ 145.28 MB/s
seaweedfs    ███████████████████ 142.50 MB/s
rustfs       ██████████████████ 136.65 MB/s
liteio       ████████████████ 119.03 MB/s
```

**Latency (P50)**
```
minio        ████████████████ 4.4ms
herd_s3      ████████████████████████ 6.4ms
seaweedfs    █████████████████████████ 6.7ms
rustfs       ███████████████████████ 6.1ms
liteio       ██████████████████████████████ 7.7ms
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 89.01 MB/s | 648.4us | 1.0ms | 1.5ms | 0 |
| herd_s3 | 82.00 MB/s | 705.5us | 1.1ms | 1.5ms | 0 |
| seaweedfs | 45.65 MB/s | 1.1ms | 2.7ms | 3.8ms | 0 |
| liteio | 41.58 MB/s | 1.2ms | 3.4ms | 7.0ms | 0 |
| rustfs | 38.66 MB/s | 1.2ms | 3.8ms | 6.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 89.01 MB/s
herd_s3      ███████████████████████████ 82.00 MB/s
seaweedfs    ███████████████ 45.65 MB/s
liteio       ██████████████ 41.58 MB/s
rustfs       █████████████ 38.66 MB/s
```

**Latency (P50)**
```
minio        ████████████████ 648.4us
herd_s3      █████████████████ 705.5us
seaweedfs    ███████████████████████████ 1.1ms
liteio       ████████████████████████████ 1.2ms
rustfs       ██████████████████████████████ 1.2ms
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 250 ops/s | 4.0ms | 4.0ms | 4.0ms | 0 |
| herd_s3 | 242 ops/s | 4.1ms | 4.1ms | 4.1ms | 0 |
| seaweedfs | 239 ops/s | 4.2ms | 4.2ms | 4.2ms | 0 |
| liteio | 188 ops/s | 5.3ms | 5.3ms | 5.3ms | 0 |
| rustfs | 85 ops/s | 11.8ms | 11.8ms | 11.8ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 250 ops/s
herd_s3      ████████████████████████████ 242 ops/s
seaweedfs    ████████████████████████████ 239 ops/s
liteio       ██████████████████████ 188 ops/s
rustfs       ██████████ 85 ops/s
```

**Latency (P50)**
```
minio        ██████████ 4.0ms
herd_s3      ██████████ 4.1ms
seaweedfs    ██████████ 4.2ms
liteio       █████████████ 5.3ms
rustfs       ██████████████████████████████ 11.8ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 25 ops/s | 39.3ms | 39.3ms | 39.3ms | 0 |
| minio | 24 ops/s | 42.0ms | 42.0ms | 42.0ms | 0 |
| herd_s3 | 23 ops/s | 43.4ms | 43.4ms | 43.4ms | 0 |
| liteio | 18 ops/s | 55.4ms | 55.4ms | 55.4ms | 0 |
| rustfs | 8 ops/s | 128.2ms | 128.2ms | 128.2ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 25 ops/s
minio        ████████████████████████████ 24 ops/s
herd_s3      ███████████████████████████ 23 ops/s
liteio       █████████████████████ 18 ops/s
rustfs       █████████ 8 ops/s
```

**Latency (P50)**
```
seaweedfs    █████████ 39.3ms
minio        █████████ 42.0ms
herd_s3      ██████████ 43.4ms
liteio       ████████████ 55.4ms
rustfs       ██████████████████████████████ 128.2ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 2 ops/s | 412.3ms | 412.3ms | 412.3ms | 0 |
| seaweedfs | 2 ops/s | 428.9ms | 428.9ms | 428.9ms | 0 |
| minio | 2 ops/s | 461.7ms | 461.7ms | 461.7ms | 0 |
| liteio | 1 ops/s | 1.11s | 1.11s | 1.11s | 0 |
| rustfs | 1 ops/s | 1.73s | 1.73s | 1.73s | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 2 ops/s
seaweedfs    ████████████████████████████ 2 ops/s
minio        ██████████████████████████ 2 ops/s
liteio       ███████████ 1 ops/s
rustfs       ███████ 1 ops/s
```

**Latency (P50)**
```
herd_s3      ███████ 412.3ms
seaweedfs    ███████ 428.9ms
minio        ████████ 461.7ms
liteio       ███████████████████ 1.11s
rustfs       ██████████████████████████████ 1.73s
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 0 ops/s | 3.70s | 3.70s | 3.70s | 0 |
| herd_s3 | 0 ops/s | 4.47s | 4.47s | 4.47s | 0 |
| seaweedfs | 0 ops/s | 4.52s | 4.52s | 4.52s | 0 |
| liteio | 0 ops/s | 7.45s | 7.45s | 7.45s | 0 |
| rustfs | 0 ops/s | 12.61s | 12.61s | 12.61s | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0 ops/s
herd_s3      ████████████████████████ 0 ops/s
seaweedfs    ████████████████████████ 0 ops/s
liteio       ██████████████ 0 ops/s
rustfs       ████████ 0 ops/s
```

**Latency (P50)**
```
minio        ████████ 3.70s
herd_s3      ██████████ 4.47s
seaweedfs    ██████████ 4.52s
liteio       █████████████████ 7.45s
rustfs       ██████████████████████████████ 12.61s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1641 ops/s | 609.3us | 609.3us | 609.3us | 0 |
| minio | 1069 ops/s | 935.6us | 935.6us | 935.6us | 0 |
| seaweedfs | 1018 ops/s | 982.7us | 982.7us | 982.7us | 0 |
| liteio | 1012 ops/s | 988.3us | 988.3us | 988.3us | 0 |
| rustfs | 449 ops/s | 2.2ms | 2.2ms | 2.2ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1641 ops/s
minio        ███████████████████ 1069 ops/s
seaweedfs    ██████████████████ 1018 ops/s
liteio       ██████████████████ 1012 ops/s
rustfs       ████████ 449 ops/s
```

**Latency (P50)**
```
herd_s3      ████████ 609.3us
minio        ████████████ 935.6us
seaweedfs    █████████████ 982.7us
liteio       █████████████ 988.3us
rustfs       ██████████████████████████████ 2.2ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 768 ops/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| liteio | 669 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |
| minio | 478 ops/s | 2.1ms | 2.1ms | 2.1ms | 0 |
| seaweedfs | 387 ops/s | 2.6ms | 2.6ms | 2.6ms | 0 |
| rustfs | 129 ops/s | 7.7ms | 7.7ms | 7.7ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 768 ops/s
liteio       ██████████████████████████ 669 ops/s
minio        ██████████████████ 478 ops/s
seaweedfs    ███████████████ 387 ops/s
rustfs       █████ 129 ops/s
```

**Latency (P50)**
```
herd_s3      █████ 1.3ms
liteio       █████ 1.5ms
minio        ████████ 2.1ms
seaweedfs    ██████████ 2.6ms
rustfs       ██████████████████████████████ 7.7ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 133 ops/s | 7.5ms | 7.5ms | 7.5ms | 0 |
| liteio | 73 ops/s | 13.6ms | 13.6ms | 13.6ms | 0 |
| seaweedfs | 70 ops/s | 14.3ms | 14.3ms | 14.3ms | 0 |
| minio | 65 ops/s | 15.3ms | 15.3ms | 15.3ms | 0 |
| rustfs | 10 ops/s | 99.3ms | 99.3ms | 99.3ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 133 ops/s
liteio       ████████████████ 73 ops/s
seaweedfs    ███████████████ 70 ops/s
minio        ██████████████ 65 ops/s
rustfs       ██ 10 ops/s
```

**Latency (P50)**
```
herd_s3      ██ 7.5ms
liteio       ████ 13.6ms
seaweedfs    ████ 14.3ms
minio        ████ 15.3ms
rustfs       ██████████████████████████████ 99.3ms
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 10 ops/s | 100.5ms | 100.5ms | 100.5ms | 0 |
| herd_s3 | 6 ops/s | 171.1ms | 171.1ms | 171.1ms | 0 |
| minio | 5 ops/s | 208.5ms | 208.5ms | 208.5ms | 0 |
| liteio | 3 ops/s | 386.5ms | 386.5ms | 386.5ms | 0 |
| rustfs | 1 ops/s | 817.1ms | 817.1ms | 817.1ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 10 ops/s
herd_s3      █████████████████ 6 ops/s
minio        ██████████████ 5 ops/s
liteio       ███████ 3 ops/s
rustfs       ███ 1 ops/s
```

**Latency (P50)**
```
seaweedfs    ███ 100.5ms
herd_s3      ██████ 171.1ms
minio        ███████ 208.5ms
liteio       ██████████████ 386.5ms
rustfs       ██████████████████████████████ 817.1ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.58 MB/s | 4.2ms | 4.2ms | 4.2ms | 0 |
| seaweedfs | 0.28 MB/s | 8.7ms | 8.7ms | 8.7ms | 0 |
| minio | 0.26 MB/s | 9.3ms | 9.3ms | 9.3ms | 0 |
| rustfs | 0.26 MB/s | 9.6ms | 9.6ms | 9.6ms | 0 |
| liteio | 0.07 MB/s | 34.1ms | 34.1ms | 34.1ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.58 MB/s
seaweedfs    ██████████████ 0.28 MB/s
minio        █████████████ 0.26 MB/s
rustfs       █████████████ 0.26 MB/s
liteio       ███ 0.07 MB/s
```

**Latency (P50)**
```
herd_s3      ███ 4.2ms
seaweedfs    ███████ 8.7ms
minio        ████████ 9.3ms
rustfs       ████████ 9.6ms
liteio       ██████████████████████████████ 34.1ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.52 MB/s | 46.6ms | 46.6ms | 46.6ms | 0 |
| seaweedfs | 0.29 MB/s | 83.5ms | 83.5ms | 83.5ms | 0 |
| rustfs | 0.26 MB/s | 94.5ms | 94.5ms | 94.5ms | 0 |
| minio | 0.25 MB/s | 98.9ms | 98.9ms | 98.9ms | 0 |
| liteio | 0.14 MB/s | 173.2ms | 173.2ms | 173.2ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.52 MB/s
seaweedfs    ████████████████ 0.29 MB/s
rustfs       ██████████████ 0.26 MB/s
minio        ██████████████ 0.25 MB/s
liteio       ████████ 0.14 MB/s
```

**Latency (P50)**
```
herd_s3      ████████ 46.6ms
seaweedfs    ██████████████ 83.5ms
rustfs       ████████████████ 94.5ms
minio        █████████████████ 98.9ms
liteio       ██████████████████████████████ 173.2ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.46 MB/s | 536.3ms | 536.3ms | 536.3ms | 0 |
| seaweedfs | 0.28 MB/s | 869.0ms | 869.0ms | 869.0ms | 0 |
| rustfs | 0.24 MB/s | 1.02s | 1.02s | 1.02s | 0 |
| liteio | 0.17 MB/s | 1.44s | 1.44s | 1.44s | 0 |
| minio | 0.13 MB/s | 1.94s | 1.94s | 1.94s | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.46 MB/s
seaweedfs    ██████████████████ 0.28 MB/s
rustfs       ███████████████ 0.24 MB/s
liteio       ███████████ 0.17 MB/s
minio        ████████ 0.13 MB/s
```

**Latency (P50)**
```
herd_s3      ████████ 536.3ms
seaweedfs    █████████████ 869.0ms
rustfs       ███████████████ 1.02s
liteio       ██████████████████████ 1.44s
minio        ██████████████████████████████ 1.94s
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.32 MB/s | 7.74s | 7.74s | 7.74s | 0 |
| rustfs | 0.28 MB/s | 8.79s | 8.79s | 8.79s | 0 |
| seaweedfs | 0.25 MB/s | 9.62s | 9.62s | 9.62s | 0 |
| minio | 0.18 MB/s | 13.78s | 13.78s | 13.78s | 0 |
| liteio | 0.14 MB/s | 17.56s | 17.56s | 17.56s | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.32 MB/s
rustfs       ██████████████████████████ 0.28 MB/s
seaweedfs    ████████████████████████ 0.25 MB/s
minio        ████████████████ 0.18 MB/s
liteio       █████████████ 0.14 MB/s
```

**Latency (P50)**
```
herd_s3      █████████████ 7.74s
rustfs       ███████████████ 8.79s
seaweedfs    ████████████████ 9.62s
minio        ███████████████████████ 13.78s
liteio       ██████████████████████████████ 17.56s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 3929 ops/s | 226.0us | 414.2us | 596.9us | 0 |
| minio | 3067 ops/s | 284.9us | 534.8us | 898.2us | 0 |
| liteio | 2974 ops/s | 304.8us | 540.0us | 738.5us | 0 |
| rustfs | 2141 ops/s | 418.6us | 674.7us | 1.4ms | 0 |
| seaweedfs | 1020 ops/s | 768.5us | 2.2ms | 3.8ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 3929 ops/s
minio        ███████████████████████ 3067 ops/s
liteio       ██████████████████████ 2974 ops/s
rustfs       ████████████████ 2141 ops/s
seaweedfs    ███████ 1020 ops/s
```

**Latency (P50)**
```
herd_s3      ████████ 226.0us
minio        ███████████ 284.9us
liteio       ███████████ 304.8us
rustfs       ████████████████ 418.6us
seaweedfs    ██████████████████████████████ 768.5us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 157.48 MB/s | 613.7ms | 613.7ms | 613.7ms | 0 |
| minio | 152.05 MB/s | 667.8ms | 667.8ms | 667.8ms | 0 |
| rustfs | 118.98 MB/s | 846.6ms | 846.6ms | 846.6ms | 0 |
| herd_s3 | 117.21 MB/s | 867.3ms | 867.3ms | 867.3ms | 0 |
| liteio | 86.89 MB/s | 947.9ms | 947.9ms | 947.9ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 157.48 MB/s
minio        ████████████████████████████ 152.05 MB/s
rustfs       ██████████████████████ 118.98 MB/s
herd_s3      ██████████████████████ 117.21 MB/s
liteio       ████████████████ 86.89 MB/s
```

**Latency (P50)**
```
seaweedfs    ███████████████████ 613.7ms
minio        █████████████████████ 667.8ms
rustfs       ██████████████████████████ 846.6ms
herd_s3      ███████████████████████████ 867.3ms
liteio       ██████████████████████████████ 947.9ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 125.88 MB/s | 73.9ms | 100.5ms | 122.0ms | 0 |
| seaweedfs | 117.38 MB/s | 79.7ms | 114.5ms | 116.1ms | 0 |
| herd_s3 | 111.52 MB/s | 85.3ms | 98.4ms | 114.5ms | 0 |
| rustfs | 83.93 MB/s | 112.5ms | 143.8ms | 143.8ms | 0 |
| minio | 42.78 MB/s | 66.1ms | 142.0ms | 142.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 125.88 MB/s
seaweedfs    ███████████████████████████ 117.38 MB/s
herd_s3      ██████████████████████████ 111.52 MB/s
rustfs       ████████████████████ 83.93 MB/s
minio        ██████████ 42.78 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 73.9ms
seaweedfs    █████████████████████ 79.7ms
herd_s3      ██████████████████████ 85.3ms
rustfs       ██████████████████████████████ 112.5ms
minio        █████████████████ 66.1ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1.68 MB/s | 507.8us | 1.0ms | 1.6ms | 0 |
| seaweedfs | 1.29 MB/s | 672.5us | 1.2ms | 2.5ms | 0 |
| liteio | 0.93 MB/s | 739.5us | 2.4ms | 4.9ms | 0 |
| rustfs | 0.70 MB/s | 1.1ms | 3.0ms | 4.8ms | 0 |
| minio | 0.43 MB/s | 1.2ms | 6.4ms | 19.4ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1.68 MB/s
seaweedfs    ██████████████████████ 1.29 MB/s
liteio       ████████████████ 0.93 MB/s
rustfs       ████████████ 0.70 MB/s
minio        ███████ 0.43 MB/s
```

**Latency (P50)**
```
herd_s3      ████████████ 507.8us
seaweedfs    █████████████████ 672.5us
liteio       ██████████████████ 739.5us
rustfs       ███████████████████████████ 1.1ms
minio        ██████████████████████████████ 1.2ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 104.61 MB/s | 8.7ms | 15.0ms | 17.3ms | 0 |
| minio | 97.26 MB/s | 8.0ms | 11.4ms | 12.9ms | 0 |
| herd_s3 | 94.25 MB/s | 9.1ms | 17.7ms | 22.1ms | 0 |
| seaweedfs | 89.16 MB/s | 9.8ms | 17.6ms | 29.0ms | 0 |
| rustfs | 59.01 MB/s | 13.2ms | 35.3ms | 46.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 104.61 MB/s
minio        ███████████████████████████ 97.26 MB/s
herd_s3      ███████████████████████████ 94.25 MB/s
seaweedfs    █████████████████████████ 89.16 MB/s
rustfs       ████████████████ 59.01 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 8.7ms
minio        ██████████████████ 8.0ms
herd_s3      ████████████████████ 9.1ms
seaweedfs    ██████████████████████ 9.8ms
rustfs       ██████████████████████████████ 13.2ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 47.03 MB/s | 1.2ms | 2.0ms | 3.2ms | 0 |
| seaweedfs | 45.99 MB/s | 1.3ms | 2.0ms | 2.9ms | 0 |
| minio | 38.52 MB/s | 1.4ms | 2.7ms | 5.0ms | 0 |
| liteio | 35.56 MB/s | 1.4ms | 3.5ms | 5.3ms | 0 |
| rustfs | 34.49 MB/s | 1.5ms | 3.2ms | 5.8ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 47.03 MB/s
seaweedfs    █████████████████████████████ 45.99 MB/s
minio        ████████████████████████ 38.52 MB/s
liteio       ██████████████████████ 35.56 MB/s
rustfs       ██████████████████████ 34.49 MB/s
```

**Latency (P50)**
```
herd_s3      ████████████████████████ 1.2ms
seaweedfs    ████████████████████████ 1.3ms
minio        ████████████████████████████ 1.4ms
liteio       ████████████████████████████ 1.4ms
rustfs       ██████████████████████████████ 1.5ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| herd_s3 | 1.613GiB / 7.653GiB | 1651.7 MB | - | 4.9% | 1812.0 MB | 3.68MB / 1.81GB |
| liteio | 847.9MiB / 7.653GiB | 847.9 MB | - | 6.7% | 1761.3 MB | 1.47MB / 2.87GB |
| minio | 957.1MiB / 7.653GiB | 957.1 MB | - | 10.6% | 1652.0 MB | 65.7MB / 1.75GB |
| rustfs | 729.2MiB / 7.653GiB | 729.2 MB | - | 0.0% | 2321.0 MB | 25.7MB / 2.45GB |
| seaweedfs | 152.4MiB / 7.653GiB | 152.4 MB | - | 0.2% | 3004.0 MB | 1.34MB / 2.21GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** seaweedfs
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
