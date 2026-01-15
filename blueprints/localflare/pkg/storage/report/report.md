# Storage Benchmark Report

**Generated:** 2026-01-15T23:34:12+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio (won 12/14 benchmarks, 86%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio | 12 | 86% |
| 2 | minio | 2 | 14% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 4.3 MB/s | +65% vs minio |
| Large Read (10MB) | minio | 284.5 MB/s | close |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Downloads (100MB) | **minio** | 294 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 2221 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 0.0 | 258.6 | 0ns | 386.0ms |
| minio | 0.0 | 294.1 | 0ns | 339.9ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 0 | 4442 | 0ns | 213.9us |
| minio | 0 | 2692 | 0ns | 357.7us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 0 | 0 | 0 |
| minio | 0 | 0 | 0 |

### Concurrency Performance

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 4.65 | 1.11 | 0.56 | 0.31 | 0.18 | 0.34 |
| minio | 2.66 | 0.74 | 0.44 | 0.21 | 0.13 | 0.20 |

*\* indicates errors occurred*

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 100 |
| Warmup | 10 |
| Concurrency | 200 |
| Timeout | 30s |

## Drivers Tested

- **liteio** (14 benchmarks)
- **minio** (14 benchmarks)

## Detailed Results

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.65 MB/s | 209.8us | 267.5us | 197.1us | 267.5us | 346.5us | 0 |
| minio | 2.66 MB/s | 367.2us | 450.0us | 357.2us | 450.1us | 533.3us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.65 MB/s
minio        █████████████████ 2.66 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 197.1us
minio        ██████████████████████████████ 357.2us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.11 MB/s | 880.7us | 2.4ms | 725.9us | 2.4ms | 3.0ms | 0 |
| minio | 0.74 MB/s | 1.3ms | 2.4ms | 1.2ms | 2.4ms | 3.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.11 MB/s
minio        ████████████████████ 0.74 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 725.9us
minio        ██████████████████████████████ 1.2ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.18 MB/s | 5.3ms | 8.0ms | 3.8ms | 8.0ms | 8.1ms | 0 |
| minio | 0.13 MB/s | 7.5ms | 10.1ms | 6.3ms | 10.1ms | 10.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.18 MB/s
minio        █████████████████████ 0.13 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 3.8ms
minio        ██████████████████████████████ 6.3ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.34 MB/s | 2.9ms | 3.8ms | 3.0ms | 3.8ms | 3.9ms | 0 |
| minio | 0.20 MB/s | 4.9ms | 6.4ms | 5.3ms | 6.4ms | 6.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.34 MB/s
minio        █████████████████ 0.20 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 3.0ms
minio        ██████████████████████████████ 5.3ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.56 MB/s | 1.8ms | 3.6ms | 1.5ms | 3.6ms | 4.1ms | 0 |
| minio | 0.44 MB/s | 2.2ms | 4.4ms | 1.9ms | 4.4ms | 6.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.56 MB/s
minio        ███████████████████████ 0.44 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████ 1.5ms
minio        ██████████████████████████████ 1.9ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.31 MB/s | 3.1ms | 6.5ms | 2.3ms | 6.5ms | 6.7ms | 0 |
| minio | 0.21 MB/s | 4.6ms | 9.8ms | 3.8ms | 9.8ms | 10.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.31 MB/s
minio        ████████████████████ 0.21 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 2.3ms
minio        ██████████████████████████████ 3.8ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 221.64 MB/s | 1.1ms | 1.3ms | 1.7ms | 0 |
| minio | 168.90 MB/s | 1.4ms | 1.8ms | 2.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 221.64 MB/s
minio        ██████████████████████ 168.90 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████ 1.1ms
minio        ██████████████████████████████ 1.4ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 233.60 MB/s | 1.1ms | 1.2ms | 1.3ms | 0 |
| minio | 150.01 MB/s | 1.6ms | 2.3ms | 3.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 233.60 MB/s
minio        ███████████████████ 150.01 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 1.1ms
minio        ██████████████████████████████ 1.6ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 207.64 MB/s | 1.1ms | 1.4ms | 3.4ms | 0 |
| minio | 131.92 MB/s | 1.7ms | 2.7ms | 4.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 207.64 MB/s
minio        ███████████████████ 131.92 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 1.1ms
minio        ██████████████████████████████ 1.7ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 294.15 MB/s | 1.3ms | 1.3ms | 339.9ms | 345.7ms | 345.7ms | 0 |
| liteio | 258.60 MB/s | 4.0ms | 4.6ms | 386.0ms | 397.0ms | 397.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 294.15 MB/s
liteio       ██████████████████████████ 258.60 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 339.9ms
liteio       ██████████████████████████████ 386.0ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 284.54 MB/s | 1.3ms | 1.5ms | 33.8ms | 37.9ms | 37.9ms | 0 |
| liteio | 261.78 MB/s | 4.1ms | 4.5ms | 37.9ms | 41.4ms | 41.4ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 284.54 MB/s
liteio       ███████████████████████████ 261.78 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 33.8ms
liteio       ██████████████████████████████ 37.9ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.34 MB/s | 225.0us | 321.0us | 213.9us | 321.1us | 398.0us | 0 |
| minio | 2.63 MB/s | 371.3us | 442.7us | 357.7us | 442.8us | 502.2us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.34 MB/s
minio        ██████████████████ 2.63 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 213.9us
minio        ██████████████████████████████ 357.7us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 292.67 MB/s | 345.8us | 531.9us | 3.3ms | 4.0ms | 4.0ms | 0 |
| minio | 242.19 MB/s | 1.0ms | 1.2ms | 4.1ms | 4.4ms | 4.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 292.67 MB/s
minio        ████████████████████████ 242.19 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████ 3.3ms
minio        ██████████████████████████████ 4.1ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 120.75 MB/s | 344.0us | 808.9us | 443.7us | 1.1ms | 1.5ms | 0 |
| minio | 98.15 MB/s | 457.6us | 626.8us | 611.0us | 821.9us | 866.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 120.75 MB/s
minio        ████████████████████████ 98.15 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 443.7us
minio        ██████████████████████████████ 611.0us
```

## Recommendations

- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
