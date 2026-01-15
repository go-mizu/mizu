# Storage Benchmark Report

**Generated:** 2026-01-15T11:23:18+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 4.4 MB/s |  |
| Small Write (1KB) | liteio | 1.2 MB/s |  |
| Large Read (10MB) | liteio | 296.0 MB/s |  |
| Large Write (10MB) | liteio | 182.4 MB/s |  |
| Delete | liteio | 6.4K ops/s |  |
| Stat | liteio | 5.4K ops/s |  |
| List (100 objects) | liteio | 1.4K ops/s |  |
| Copy | liteio | 2.1 MB/s |  |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **liteio** | 163 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **liteio** | 290 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 2871 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |
| Memory Constrained | **liteio** | 59 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 162.6 | 289.9 | 621.5ms | 344.7ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 1200 | 4541 | 753.2us | 198.3us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 5391 | 1384 | 6406 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 1.93 | 0.41 | 0.26 | 0.13 | 0.19 | 0.20 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 4.38 | 0.83 | 0.83 | 0.56 | 0.49 | 0.51 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 616.1us | 5.2ms | 47.7ms | 554.6ms | 5.14s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 297.4us | 379.5us | 811.6us | 5.0ms | 190.4ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 58.9 MB | 0.0% |

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 50 |
| Warmup | 10 |
| Concurrency | 200 |
| Timeout | 30s |

## Drivers Tested

- **liteio** (51 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.12 MB/s | 427.7us | 541.4us | 547.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.12 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 427.7us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6406 ops/s | 152.9us | 176.9us | 179.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 6406 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 152.9us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.15 MB/s | 556.0us | 984.8us | 1.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.15 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 556.0us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1840 ops/s | 451.5us | 717.8us | 862.2us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1840 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 451.5us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.17 MB/s | 530.4us | 772.0us | 786.3us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.17 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 530.4us
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4348 ops/s | 230.0us | 230.0us | 230.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4348 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 230.0us
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 407 ops/s | 2.5ms | 2.5ms | 2.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 407 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 2.5ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 52 ops/s | 19.2ms | 19.2ms | 19.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 52 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 19.2ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5 ops/s | 190.1ms | 190.1ms | 190.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 190.1ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1 ops/s | 1.94s | 1.94s | 1.94s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 1.94s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3363 ops/s | 297.4us | 297.4us | 297.4us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3363 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 297.4us
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2635 ops/s | 379.5us | 379.5us | 379.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2635 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 379.5us
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1232 ops/s | 811.6us | 811.6us | 811.6us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1232 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 811.6us
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 199 ops/s | 5.0ms | 5.0ms | 5.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 199 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 5.0ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5 ops/s | 190.4ms | 190.4ms | 190.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 190.4ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.59 MB/s | 616.1us | 616.1us | 616.1us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.59 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 616.1us
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.87 MB/s | 5.2ms | 5.2ms | 5.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.87 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 5.2ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.05 MB/s | 47.7ms | 47.7ms | 47.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.05 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 47.7ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.76 MB/s | 554.6ms | 554.6ms | 554.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.76 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 554.6ms
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.90 MB/s | 5.14s | 5.14s | 5.14s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.90 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 5.14s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1384 ops/s | 692.1us | 852.7us | 906.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1384 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 692.1us
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.74 MB/s | 3.3ms | 4.4ms | 4.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.74 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 3.3ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.76 MB/s | 4.3ms | 4.6ms | 4.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.76 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 4.3ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.55 MB/s | 6.3ms | 8.9ms | 9.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.55 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 6.3ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 114.74 MB/s | 123.1ms | 133.9ms | 133.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 114.74 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 123.1ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.38 MB/s | 219.8us | 273.2us | 195.6us | 279.6us | 530.1us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.38 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 195.6us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.83 MB/s | 1.2ms | 2.3ms | 946.8us | 2.3ms | 2.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.83 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 946.8us
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.49 MB/s | 2.0ms | 2.3ms | 2.0ms | 2.3ms | 2.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.49 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 2.0ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.51 MB/s | 1.9ms | 2.3ms | 1.9ms | 2.3ms | 2.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.51 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 1.9ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.83 MB/s | 1.2ms | 1.7ms | 1.1ms | 1.7ms | 1.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.83 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 1.1ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.56 MB/s | 1.7ms | 2.3ms | 1.7ms | 2.4ms | 2.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.56 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 1.7ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.93 MB/s | 473.4us | 722.0us | 855.6us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.93 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 473.4us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.41 MB/s | 1.8ms | 4.7ms | 5.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.41 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 1.8ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.19 MB/s | 5.4ms | 7.6ms | 7.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.19 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 5.4ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.20 MB/s | 4.9ms | 7.3ms | 7.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.20 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 4.9ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.26 MB/s | 3.3ms | 5.9ms | 6.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.26 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 3.3ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.13 MB/s | 7.0ms | 13.0ms | 13.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.13 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 7.0ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 248.52 MB/s | 988.8us | 1.1ms | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 248.52 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 988.8us
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 243.78 MB/s | 991.9us | 1.2ms | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 243.78 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 991.9us
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 239.26 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 239.26 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 1.0ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 289.89 MB/s | 495.9us | 498.8us | 344.7ms | 353.0ms | 353.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 289.89 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 344.7ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 296.00 MB/s | 743.9us | 1.1ms | 31.8ms | 38.6ms | 38.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 296.00 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 31.8ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.43 MB/s | 218.7us | 328.3us | 198.3us | 328.4us | 359.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.43 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 198.3us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 238.80 MB/s | 659.8us | 946.0us | 3.9ms | 6.1ms | 6.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 238.80 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 3.9ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 107.22 MB/s | 330.7us | 671.3us | 528.2us | 835.8us | 955.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 107.22 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 528.2us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5391 ops/s | 161.1us | 234.4us | 306.6us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5391 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 161.1us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 162.56 MB/s | 621.5ms | 624.8ms | 624.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 162.56 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 621.5ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 182.44 MB/s | 53.7ms | 61.2ms | 61.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 182.44 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 53.7ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.17 MB/s | 753.2us | 1.2ms | 1.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.17 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 753.2us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 163.45 MB/s | 5.8ms | 8.1ms | 8.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 163.45 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 5.8ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 59.21 MB/s | 952.2us | 1.6ms | 2.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 59.21 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 952.2us
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 58.9MiB / 7.653GiB | 58.9 MB | - | 0.0% | 3686.4 MB | 2.18MB / 1.91GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** liteio
- **Read-heavy workloads:** liteio

---

*Generated by storage benchmark CLI*
