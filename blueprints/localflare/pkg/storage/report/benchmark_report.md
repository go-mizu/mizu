# Storage Benchmark Report

**Generated:** 2026-01-15T00:10:12+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Quick Comparison

| Driver | Write (MB/s) | Read (MB/s) | Errors |
|--------|-------------|-------------|--------|
| liteio | 153.52 | 283.90 | 0 |
| liteio_mem | 156.46 | 290.08 | 0 |

### Key Findings

- **Best Write Performance**: liteio_mem (156.46 MB/s)
- **Best Read Performance**: liteio_mem (290.08 MB/s)

### Resource Insights

- **Highest Memory**: liteio (381.9 MB)
- **Lowest Memory**: liteio_mem (10.1 MB)

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 20 |
| Warmup | 5 |
| Concurrency | 10 |
| Timeout | 30s |

## Drivers Tested

- liteio (22 benchmarks)
- liteio_mem (22 benchmarks)

## Performance Comparison

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.46 MB/s | 613.5us | 806.5us | 806.5us | 0 |
| liteio | 1.04 MB/s | 822.2us | 1.8ms | 1.8ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.46 MB/s
  liteio       ████████████████████████████ 1.04 MB/s
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5545 ops/s | 172.2us | 214.4us | 214.4us | 0 |
| liteio_mem | 5116 ops/s | 198.3us | 220.3us | 220.3us | 0 |

```
  liteio       ████████████████████████████████████████ 5545 ops/s
  liteio_mem   ████████████████████████████████████ 5116 ops/s
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.13 MB/s | 655.9us | 781.2us | 781.2us | 0 |
| liteio | 0.09 MB/s | 668.9us | 1.9ms | 1.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.13 MB/s
  liteio       ██████████████████████████ 0.09 MB/s
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1082 ops/s | 678.8us | 1.3ms | 1.3ms | 0 |
| liteio | 966 ops/s | 840.5us | 1.4ms | 1.4ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1082 ops/s
  liteio       ███████████████████████████████████ 966 ops/s
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.13 MB/s | 708.0us | 793.5us | 793.5us | 0 |
| liteio | 0.08 MB/s | 1.0ms | 1.3ms | 1.3ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.13 MB/s
  liteio       ███████████████████████ 0.08 MB/s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1259 ops/s | 724.1us | 1.1ms | 1.1ms | 0 |
| liteio | 1058 ops/s | 771.0us | 1.3ms | 1.3ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1259 ops/s
  liteio       █████████████████████████████████ 1058 ops/s
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 16.67 MB/s | 978.8us | 1.2ms | 1.2ms | 0 |
| liteio | 9.11 MB/s | 1.6ms | 2.6ms | 2.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 16.67 MB/s
  liteio       █████████████████████ 9.11 MB/s
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 10.48 MB/s | 1.6ms | 1.8ms | 1.8ms | 0 |
| liteio | 9.26 MB/s | 1.5ms | 3.1ms | 3.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 10.48 MB/s
  liteio       ███████████████████████████████████ 9.26 MB/s
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 6.06 MB/s | 1.7ms | 4.4ms | 4.4ms | 0 |
| liteio | 4.90 MB/s | 3.3ms | 5.4ms | 5.4ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 6.06 MB/s
  liteio       ████████████████████████████████ 4.90 MB/s
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 147.68 MB/s | 101.6ms | 102.5ms | 102.5ms | 0 |
| liteio | 135.94 MB/s | 106.5ms | 113.5ms | 113.5ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 147.68 MB/s
  liteio       ████████████████████████████████████ 135.94 MB/s
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.72 MB/s | 1.1ms | 2.9ms | 2.9ms | 0 |
| liteio_mem | 0.66 MB/s | 1.5ms | 2.2ms | 2.2ms | 0 |

```
  liteio       ████████████████████████████████████████ 0.72 MB/s
  liteio_mem   ████████████████████████████████████ 0.66 MB/s
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.23 MB/s | 3.9ms | 8.1ms | 8.1ms | 0 |
| liteio_mem | 0.20 MB/s | 3.5ms | 7.9ms | 7.9ms | 0 |

```
  liteio       ████████████████████████████████████████ 0.23 MB/s
  liteio_mem   ██████████████████████████████████ 0.20 MB/s
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 237.59 MB/s | 1.0ms | 1.3ms | 1.3ms | 0 |
| liteio_mem | 219.40 MB/s | 1.1ms | 1.4ms | 1.4ms | 0 |

```
  liteio       ████████████████████████████████████████ 237.59 MB/s
  liteio_mem   ████████████████████████████████████ 219.40 MB/s
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 239.55 MB/s | 1.0ms | 1.1ms | 1.1ms | 0 |
| liteio_mem | 239.13 MB/s | 1.0ms | 1.1ms | 1.1ms | 0 |

```
  liteio       ████████████████████████████████████████ 239.55 MB/s
  liteio_mem   ███████████████████████████████████████ 239.13 MB/s
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 241.07 MB/s | 1.0ms | 1.1ms | 1.1ms | 0 |
| liteio | 224.12 MB/s | 1.0ms | 1.7ms | 1.7ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 241.07 MB/s
  liteio       █████████████████████████████████████ 224.12 MB/s
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 4.51 MB/s | 214.3us | 239.7us | 239.7us | 0 |
| liteio | 4.47 MB/s | 208.1us | 259.8us | 259.8us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4.51 MB/s
  liteio       ███████████████████████████████████████ 4.47 MB/s
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 290.08 MB/s | 3.4ms | 3.6ms | 3.6ms | 0 |
| liteio | 283.90 MB/s | 3.4ms | 4.0ms | 4.0ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 290.08 MB/s
  liteio       ███████████████████████████████████████ 283.90 MB/s
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 147.84 MB/s | 408.5us | 553.8us | 553.8us | 0 |
| liteio | 128.50 MB/s | 400.7us | 507.4us | 507.4us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 147.84 MB/s
  liteio       ██████████████████████████████████ 128.50 MB/s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 6606 ops/s | 141.9us | 191.6us | 191.6us | 0 |
| liteio | 6445 ops/s | 149.5us | 212.7us | 212.7us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 6606 ops/s
  liteio       ███████████████████████████████████████ 6445 ops/s
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.81 MB/s | 1.1ms | 1.9ms | 1.9ms | 0 |
| liteio | 0.71 MB/s | 1.2ms | 1.9ms | 1.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.81 MB/s
  liteio       ███████████████████████████████████ 0.71 MB/s
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 156.46 MB/s | 6.3ms | 6.7ms | 6.7ms | 0 |
| liteio | 153.52 MB/s | 6.3ms | 7.6ms | 7.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 156.46 MB/s
  liteio       ███████████████████████████████████████ 153.52 MB/s
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 51.34 MB/s | 1.1ms | 1.6ms | 1.6ms | 0 |
| liteio_mem | 48.81 MB/s | 1.2ms | 1.7ms | 1.7ms | 0 |

```
  liteio       ████████████████████████████████████████ 51.34 MB/s
  liteio_mem   ██████████████████████████████████████ 48.81 MB/s
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 381.9MiB / 7.653GiB | - | - | 0.0% | (no data) | 2.59GB / 7.1GB |
| liteio_mem | 10.07MiB / 7.653GiB | - | - | 0.7% | (no data) | 8.18MB / 230MB |

### Memory Analysis Note

> **RSS (Resident Set Size)**: Actual application memory usage.
> 
> **Cache**: Linux page cache from filesystem I/O. Disk-based drivers show higher total memory because the OS caches file pages in RAM. This memory is reclaimable and doesn't indicate a memory leak.
> 
> Memory-based drivers (like `liteio_mem`) have minimal cache because data stays in application memory (RSS), not filesystem cache.

## Recommendations

- **Best for write-heavy workloads:** liteio_mem
- **Best for read-heavy workloads:** liteio_mem

---

*Report generated by storage benchmark CLI*
