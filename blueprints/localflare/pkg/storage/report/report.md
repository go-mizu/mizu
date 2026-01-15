# Storage Benchmark Report

**Generated:** 2026-01-15T23:15:04+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio (won 7/12 benchmarks, 58%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio | 7 | 58% |
| 2 | rustfs | 4 | 33% |
| 3 | minio | 1 | 8% |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| High Concurrency (C10) | **minio** | - | Best for multi-user apps |
| Memory Constrained | **liteio** | 33 MB RAM | Best for edge/embedded |

### Large File Performance (1MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 0.0 | 0.0 | 0ns | 0ns |
| minio | 0.0 | 0.0 | 0ns | 0ns |
| rustfs | 0.0 | 0.0 | 0ns | 0ns |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 0 | 0 | 0ns | 0ns |
| minio | 0 | 0 | 0ns | 0ns |
| rustfs | 0 | 0 | 0ns | 0ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 0 | 0 | 0 |
| minio | 0 | 0 | 0 |
| rustfs | 0 | 0 | 0 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 0.74 | 0.40 | 0.18 | 0.07 | 0.15 | 0.13 |
| minio | 0.60 | 0.32 | 0.12 | 0.07 | 0.10 | 0.09 |
| rustfs | 0.32 | 0.34 | 0.17 | 0.08 | 0.20 | 0.16 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 3.21 | 0.79 | 0.91 | 0.39 | 0.47 | 0.37 |
| minio | 2.48 | 0.90 | 0.44 | 0.26 | 0.28 | 0.26 |
| rustfs | 1.94 | 0.76 | 0.54 | 0.65 | 0.22 | 0.33 |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 33.4 MB | 1.0% |
| minio | 208.5 MB | 0.0% |
| rustfs | 147.8 MB | 0.0% |

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 50 |
| Warmup | 10 |
| Concurrency | 200 |
| Timeout | 30s |

## Drivers Tested

- **liteio** (12 benchmarks)
- **minio** (12 benchmarks)
- **rustfs** (12 benchmarks)

## Detailed Results

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 3.21 MB/s | 304.2us | 596.8us | 261.8us | 596.8us | 700.5us | 0 |
| minio | 2.48 MB/s | 393.9us | 438.1us | 351.7us | 438.2us | 623.5us | 0 |
| rustfs | 1.94 MB/s | 503.6us | 543.6us | 476.4us | 543.7us | 680.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.21 MB/s
minio        ███████████████████████ 2.48 MB/s
rustfs       ██████████████████ 1.94 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 261.8us
minio        ██████████████████████ 351.7us
rustfs       ██████████████████████████████ 476.4us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.90 MB/s | 1.1ms | 1.6ms | 1.1ms | 1.6ms | 1.6ms | 0 |
| liteio | 0.79 MB/s | 1.2ms | 1.9ms | 1.1ms | 1.9ms | 2.0ms | 0 |
| rustfs | 0.76 MB/s | 1.3ms | 1.7ms | 1.2ms | 1.7ms | 2.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.90 MB/s
liteio       ██████████████████████████ 0.79 MB/s
rustfs       █████████████████████████ 0.76 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 1.1ms
liteio       ██████████████████████████ 1.1ms
rustfs       ██████████████████████████████ 1.2ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.47 MB/s | 2.1ms | 2.6ms | 2.2ms | 2.6ms | 2.6ms | 0 |
| minio | 0.28 MB/s | 3.5ms | 4.5ms | 3.3ms | 4.5ms | 4.6ms | 0 |
| rustfs | 0.22 MB/s | 4.5ms | 5.2ms | 4.7ms | 5.2ms | 5.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.47 MB/s
minio        ██████████████████ 0.28 MB/s
rustfs       █████████████ 0.22 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 2.2ms
minio        █████████████████████ 3.3ms
rustfs       ██████████████████████████████ 4.7ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.37 MB/s | 2.6ms | 3.3ms | 2.8ms | 3.3ms | 3.5ms | 0 |
| rustfs | 0.33 MB/s | 3.0ms | 3.7ms | 3.2ms | 3.7ms | 3.8ms | 0 |
| minio | 0.26 MB/s | 3.7ms | 4.1ms | 3.7ms | 4.1ms | 4.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.37 MB/s
rustfs       ██████████████████████████ 0.33 MB/s
minio        █████████████████████ 0.26 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████ 2.8ms
rustfs       ██████████████████████████ 3.2ms
minio        ██████████████████████████████ 3.7ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.91 MB/s | 1.1ms | 1.6ms | 1.0ms | 1.6ms | 1.7ms | 0 |
| rustfs | 0.54 MB/s | 1.8ms | 2.3ms | 1.8ms | 2.3ms | 2.7ms | 0 |
| minio | 0.44 MB/s | 2.2ms | 2.7ms | 2.2ms | 2.7ms | 2.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.91 MB/s
rustfs       █████████████████ 0.54 MB/s
minio        ██████████████ 0.44 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 1.0ms
rustfs       ███████████████████████ 1.8ms
minio        ██████████████████████████████ 2.2ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rustfs | 0.65 MB/s | 1.5ms | 2.2ms | 1.4ms | 2.2ms | 2.4ms | 0 |
| liteio | 0.39 MB/s | 2.5ms | 3.3ms | 2.4ms | 3.3ms | 3.3ms | 0 |
| minio | 0.26 MB/s | 3.7ms | 4.7ms | 3.8ms | 4.7ms | 4.8ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.65 MB/s
liteio       ██████████████████ 0.39 MB/s
minio        ████████████ 0.26 MB/s
```

**Latency (P50)**
```
rustfs       ███████████ 1.4ms
liteio       ██████████████████ 2.4ms
minio        ██████████████████████████████ 3.8ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.74 MB/s | 1.2ms | 2.1ms | 2.4ms | 0 |
| minio | 0.60 MB/s | 1.2ms | 3.0ms | 4.3ms | 0 |
| rustfs | 0.32 MB/s | 866.2us | 1.4ms | 1.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.74 MB/s
minio        ████████████████████████ 0.60 MB/s
rustfs       █████████████ 0.32 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████████ 1.2ms
minio        ██████████████████████████████ 1.2ms
rustfs       █████████████████████ 866.2us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.40 MB/s | 2.1ms | 3.9ms | 4.1ms | 0 |
| rustfs | 0.34 MB/s | 2.6ms | 5.1ms | 5.3ms | 0 |
| minio | 0.32 MB/s | 2.8ms | 5.1ms | 5.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.40 MB/s
rustfs       ████████████████████████ 0.34 MB/s
minio        ███████████████████████ 0.32 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 2.1ms
rustfs       ███████████████████████████ 2.6ms
minio        ██████████████████████████████ 2.8ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.20 MB/s | 5.2ms | 6.6ms | 6.8ms | 0 |
| liteio | 0.15 MB/s | 5.4ms | 9.0ms | 9.1ms | 0 |
| minio | 0.10 MB/s | 10.0ms | 13.3ms | 13.4ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.20 MB/s
liteio       ██████████████████████ 0.15 MB/s
minio        ███████████████ 0.10 MB/s
```

**Latency (P50)**
```
rustfs       ███████████████ 5.2ms
liteio       ████████████████ 5.4ms
minio        ██████████████████████████████ 10.0ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.16 MB/s | 6.9ms | 8.1ms | 8.3ms | 0 |
| liteio | 0.13 MB/s | 7.9ms | 10.3ms | 11.0ms | 0 |
| minio | 0.09 MB/s | 11.1ms | 15.3ms | 15.4ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.16 MB/s
liteio       ███████████████████████ 0.13 MB/s
minio        █████████████████ 0.09 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████ 6.9ms
liteio       █████████████████████ 7.9ms
minio        ██████████████████████████████ 11.1ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.18 MB/s | 5.2ms | 8.3ms | 8.5ms | 0 |
| rustfs | 0.17 MB/s | 5.3ms | 10.0ms | 10.5ms | 0 |
| minio | 0.12 MB/s | 7.8ms | 13.4ms | 14.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.18 MB/s
rustfs       ████████████████████████████ 0.17 MB/s
minio        ███████████████████ 0.12 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 5.2ms
rustfs       ████████████████████ 5.3ms
minio        ██████████████████████████████ 7.8ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.08 MB/s | 7.2ms | 23.5ms | 23.6ms | 0 |
| liteio | 0.07 MB/s | 13.7ms | 15.3ms | 15.5ms | 0 |
| minio | 0.07 MB/s | 14.9ms | 18.2ms | 18.4ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.08 MB/s
liteio       ███████████████████████████ 0.07 MB/s
minio        ████████████████████████ 0.07 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████ 7.2ms
liteio       ███████████████████████████ 13.7ms
minio        ██████████████████████████████ 14.9ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 33.42MiB / 7.653GiB | 33.4 MB | - | 1.0% | (no data) | 115kB / 1.72MB |
| minio | 209.2MiB / 7.653GiB | 209.2 MB | - | 0.0% | 0.6 MB | 71.8MB / 1.86MB |
| rustfs | 147.2MiB / 7.653GiB | 147.2 MB | - | 0.0% | 0.6 MB | 12.7MB / 24.6kB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations


---

*Generated by storage benchmark CLI*
