# Storage Benchmark Report

**Generated:** 2026-01-15T23:51:58+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |

### Large File Performance (1MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 0.0 | 0.0 | 0ns | 0ns |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 0 | 0 | 0ns | 0ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 0 | 0 | 0 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 1.97 | 0.78 | 0.32 | 0.14 | 0.07 | 0.04 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 4.25 | 1.58 | 0.88 | 0.45 | 0.26 | 0.15 |

*\* indicates errors occurred*

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 200 |
| Warmup | 10 |
| Concurrency | 200 |
| Timeout | 30s |

## Drivers Tested

- **liteio** (12 benchmarks)

## Detailed Results

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.25 MB/s | 229.7us | 327.5us | 210.4us | 327.6us | 511.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.25 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 210.4us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.58 MB/s | 616.1us | 996.6us | 592.3us | 996.7us | 1.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.58 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 592.3us
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.26 MB/s | 3.8ms | 5.5ms | 3.4ms | 5.5ms | 5.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.26 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 3.4ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.15 MB/s | 6.4ms | 8.5ms | 6.3ms | 8.5ms | 8.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.15 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 6.3ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.88 MB/s | 1.1ms | 1.8ms | 1.1ms | 1.8ms | 2.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.88 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 1.1ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.45 MB/s | 2.2ms | 3.5ms | 1.9ms | 3.5ms | 3.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.45 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 1.9ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.97 MB/s | 405.5us | 1.0ms | 1.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.97 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 405.5us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.78 MB/s | 1.2ms | 2.2ms | 2.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.78 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 1.2ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.07 MB/s | 10.9ms | 32.3ms | 33.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.07 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 10.9ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.04 MB/s | 25.4ms | 33.2ms | 33.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.04 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 25.4ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.32 MB/s | 2.7ms | 6.5ms | 7.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.32 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 2.7ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.14 MB/s | 5.7ms | 14.1ms | 21.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.14 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 5.7ms
```

## Recommendations


---

*Generated by storage benchmark CLI*
