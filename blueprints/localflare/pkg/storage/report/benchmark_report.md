# Storage Benchmark Report

**Generated:** 2026-01-15T00:25:13+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Throughput Summary (MB/s)

| Driver | Write | Read |
|--------|-------|------|
| liteio | 164.67 | 294.45 |
| liteio_mem | 166.71 | 292.97 |
| localstack | 132.80 | 241.03 |
| minio | 140.97 | 260.04 |
| rustfs | 164.28 | 217.10 |
| seaweedfs | 124.47 | 236.05 |

### Latency Summary

| Driver | Write P50 | Write P99 | Read P50 | Read P99 |
|--------|-----------|-----------|----------|----------|
| liteio | 5.9ms | 6.7ms | 3.4ms | 3.6ms |
| liteio_mem | 5.9ms | 6.4ms | 3.4ms | 3.6ms |
| localstack | 7.3ms | 8.4ms | 4.0ms | 5.2ms |
| minio | 7.0ms | 8.2ms | 3.8ms | 4.0ms |
| rustfs | 5.9ms | 8.3ms | 4.5ms | 5.0ms |
| seaweedfs | 7.9ms | 9.2ms | 4.1ms | 6.0ms |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C5 | C10 |
|--------|------|------|------|
| liteio | 1.40 | 0.57 | 0.37 |
| liteio_mem | 1.35 | 0.51 | 0.32 |
| localstack | 1.26 | 0.35 | 0.19 |
| minio | 1.02 | 0.56 | 0.33 |
| rustfs | 1.21 | 0.86 | 0.39 |
| seaweedfs | 1.45 | 0.66 | 0.42 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C5 | C10 |
|--------|------|------|------|
| liteio | 4.92 | 1.97 | 0.90 |
| liteio_mem | 4.60 | 1.69 | 1.02 |
| localstack | 1.27 | 0.34 | 0.19 |
| minio | 2.23 | 1.76 | 1.08 |
| rustfs | 1.76 | 1.38 | 0.91 |
| seaweedfs | 1.87 | 1.35 | 0.68 |

*\* indicates errors occurred*

### Key Findings

- **Best Write Throughput**: liteio_mem (166.71 MB/s)
- **Best Read Throughput**: liteio (294.45 MB/s)
- **Lowest Write Latency**: rustfs (5.9ms P50)

### Resource Insights

- **Highest Memory**: minio (407.5 MB)
- **Lowest Memory**: liteio (9.6 MB)

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 30 |
| Warmup | 10 |
| Concurrency | 10 |
| Timeout | 30s |

## Drivers Tested

- liteio (26 benchmarks)
- liteio_mem (26 benchmarks)
- localstack (26 benchmarks)
- minio (26 benchmarks)
- rustfs (26 benchmarks)
- seaweedfs (26 benchmarks)

## Performance Comparison

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.60 MB/s | 581.8us | 688.5us | 838.0us | 0 |
| liteio_mem | 1.47 MB/s | 592.5us | 900.5us | 1.1ms | 0 |
| localstack | 1.35 MB/s | 689.2us | 891.2us | 935.1us | 0 |
| minio | 1.07 MB/s | 914.3us | 1.1ms | 1.1ms | 0 |
| rustfs | 1.02 MB/s | 941.5us | 1.1ms | 1.1ms | 0 |
| seaweedfs | 0.93 MB/s | 1.0ms | 1.3ms | 1.4ms | 0 |

```
  liteio       ████████████████████████████████████████ 1.60 MB/s
  liteio_mem   ████████████████████████████████████ 1.47 MB/s
  localstack   █████████████████████████████████ 1.35 MB/s
  minio        ██████████████████████████ 1.07 MB/s
  rustfs       █████████████████████████ 1.02 MB/s
  seaweedfs    ███████████████████████ 0.93 MB/s
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 6517 ops/s | 148.5us | 186.7us | 189.0us | 0 |
| liteio | 6485 ops/s | 144.0us | 181.6us | 187.0us | 0 |
| seaweedfs | 3092 ops/s | 305.4us | 344.3us | 377.2us | 0 |
| minio | 2992 ops/s | 326.4us | 366.5us | 423.1us | 0 |
| localstack | 1703 ops/s | 570.2us | 652.2us | 665.8us | 0 |
| rustfs | 1149 ops/s | 816.4us | 1.1ms | 1.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 6517 ops/s
  liteio       ███████████████████████████████████████ 6485 ops/s
  seaweedfs    ██████████████████ 3092 ops/s
  minio        ██████████████████ 2992 ops/s
  localstack   ██████████ 1703 ops/s
  rustfs       ███████ 1149 ops/s
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.15 MB/s | 590.8us | 687.0us | 687.0us | 0 |
| seaweedfs | 0.14 MB/s | 632.1us | 862.1us | 862.1us | 0 |
| liteio | 0.14 MB/s | 647.7us | 858.0us | 858.0us | 0 |
| localstack | 0.14 MB/s | 700.5us | 799.0us | 799.0us | 0 |
| liteio_mem | 0.13 MB/s | 669.5us | 877.3us | 877.3us | 0 |
| minio | 0.11 MB/s | 894.9us | 1.0ms | 1.0ms | 0 |

```
  rustfs       ████████████████████████████████████████ 0.15 MB/s
  seaweedfs    █████████████████████████████████████ 0.14 MB/s
  liteio       ████████████████████████████████████ 0.14 MB/s
  localstack   ████████████████████████████████████ 0.14 MB/s
  liteio_mem   █████████████████████████████████ 0.13 MB/s
  minio        ███████████████████████████ 0.11 MB/s
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 2280 ops/s | 417.4us | 558.1us | 558.1us | 0 |
| localstack | 1256 ops/s | 763.1us | 930.6us | 930.6us | 0 |
| rustfs | 1255 ops/s | 766.6us | 883.8us | 883.8us | 0 |
| liteio_mem | 1190 ops/s | 664.5us | 1.6ms | 1.6ms | 0 |
| liteio | 1141 ops/s | 641.7us | 1.1ms | 1.1ms | 0 |
| minio | 1091 ops/s | 874.0us | 1.0ms | 1.0ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 2280 ops/s
  localstack   ██████████████████████ 1256 ops/s
  rustfs       ██████████████████████ 1255 ops/s
  liteio_mem   ████████████████████ 1190 ops/s
  liteio       ████████████████████ 1141 ops/s
  minio        ███████████████████ 1091 ops/s
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.15 MB/s | 543.3us | 879.4us | 879.4us | 0 |
| liteio | 0.14 MB/s | 666.6us | 749.4us | 749.4us | 0 |
| liteio_mem | 0.14 MB/s | 671.8us | 800.3us | 800.3us | 0 |
| localstack | 0.13 MB/s | 724.7us | 844.2us | 844.2us | 0 |
| seaweedfs | 0.12 MB/s | 765.8us | 1.0ms | 1.0ms | 0 |
| minio | 0.10 MB/s | 891.2us | 996.8us | 996.8us | 0 |

```
  rustfs       ████████████████████████████████████████ 0.15 MB/s
  liteio       █████████████████████████████████████ 0.14 MB/s
  liteio_mem   ████████████████████████████████████ 0.14 MB/s
  localstack   █████████████████████████████████ 0.13 MB/s
  seaweedfs    ██████████████████████████████ 0.12 MB/s
  minio        ███████████████████████████ 0.10 MB/s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1293 ops/s | 743.4us | 934.0us | 1.0ms | 0 |
| liteio_mem | 1208 ops/s | 718.8us | 998.2us | 1.5ms | 0 |
| seaweedfs | 673 ops/s | 1.4ms | 1.7ms | 1.7ms | 0 |
| minio | 444 ops/s | 2.2ms | 2.6ms | 2.7ms | 0 |
| localstack | 380 ops/s | 2.5ms | 3.2ms | 3.6ms | 0 |
| rustfs | 156 ops/s | 6.5ms | 7.0ms | 7.1ms | 0 |

```
  liteio       ████████████████████████████████████████ 1293 ops/s
  liteio_mem   █████████████████████████████████████ 1208 ops/s
  seaweedfs    ████████████████████ 673 ops/s
  minio        █████████████ 444 ops/s
  localstack   ███████████ 380 ops/s
  rustfs       ████ 156 ops/s
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 14.24 MB/s | 1.1ms | 1.5ms | 1.5ms | 0 |
| minio | 12.42 MB/s | 1.2ms | 1.8ms | 2.3ms | 0 |
| liteio_mem | 11.72 MB/s | 1.3ms | 2.0ms | 2.1ms | 0 |
| liteio | 11.41 MB/s | 1.3ms | 1.9ms | 2.0ms | 0 |
| seaweedfs | 9.38 MB/s | 1.6ms | 2.0ms | 2.1ms | 0 |
| localstack | 2.64 MB/s | 6.2ms | 7.8ms | 9.6ms | 0 |

```
  rustfs       ████████████████████████████████████████ 14.24 MB/s
  minio        ██████████████████████████████████ 12.42 MB/s
  liteio_mem   ████████████████████████████████ 11.72 MB/s
  liteio       ████████████████████████████████ 11.41 MB/s
  seaweedfs    ██████████████████████████ 9.38 MB/s
  localstack   ███████ 2.64 MB/s
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 16.94 MB/s | 851.5us | 1.2ms | 1.3ms | 0 |
| rustfs | 12.77 MB/s | 1.2ms | 1.8ms | 1.8ms | 0 |
| liteio_mem | 12.60 MB/s | 1.2ms | 1.9ms | 1.9ms | 0 |
| liteio | 9.66 MB/s | 1.5ms | 3.4ms | 3.7ms | 0 |
| seaweedfs | 9.01 MB/s | 1.6ms | 2.3ms | 2.3ms | 0 |
| localstack | 2.44 MB/s | 5.4ms | 9.2ms | 9.2ms | 0 |

```
  minio        ████████████████████████████████████████ 16.94 MB/s
  rustfs       ██████████████████████████████ 12.77 MB/s
  liteio_mem   █████████████████████████████ 12.60 MB/s
  liteio       ██████████████████████ 9.66 MB/s
  seaweedfs    █████████████████████ 9.01 MB/s
  localstack   █████ 2.44 MB/s
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 6.47 MB/s | 2.3ms | 3.4ms | 3.9ms | 0 |
| rustfs | 6.42 MB/s | 2.6ms | 3.7ms | 3.7ms | 0 |
| liteio_mem | 5.47 MB/s | 2.8ms | 4.3ms | 4.5ms | 0 |
| minio | 5.30 MB/s | 3.2ms | 4.5ms | 4.6ms | 0 |
| liteio | 4.76 MB/s | 3.1ms | 5.1ms | 5.2ms | 0 |
| localstack | 2.86 MB/s | 5.2ms | 8.7ms | 9.2ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 6.47 MB/s
  rustfs       ███████████████████████████████████████ 6.42 MB/s
  liteio_mem   █████████████████████████████████ 5.47 MB/s
  minio        ████████████████████████████████ 5.30 MB/s
  liteio       █████████████████████████████ 4.76 MB/s
  localstack   █████████████████ 2.86 MB/s
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 164.22 MB/s | 90.7ms | 93.9ms | 93.9ms | 0 |
| rustfs | 158.09 MB/s | 95.4ms | 103.9ms | 103.9ms | 0 |
| liteio | 148.03 MB/s | 95.0ms | 111.6ms | 111.6ms | 0 |
| minio | 141.66 MB/s | 105.5ms | 107.8ms | 107.8ms | 0 |
| seaweedfs | 133.28 MB/s | 112.0ms | 114.1ms | 114.1ms | 0 |
| localstack | 128.09 MB/s | 116.1ms | 117.9ms | 117.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 164.22 MB/s
  rustfs       ██████████████████████████████████████ 158.09 MB/s
  liteio       ████████████████████████████████████ 148.03 MB/s
  minio        ██████████████████████████████████ 141.66 MB/s
  seaweedfs    ████████████████████████████████ 133.28 MB/s
  localstack   ███████████████████████████████ 128.09 MB/s
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.92 MB/s | 193.0us | 226.3us | 236.8us | 0 |
| liteio_mem | 4.60 MB/s | 206.6us | 256.2us | 260.2us | 0 |
| minio | 2.23 MB/s | 423.5us | 574.5us | 624.9us | 0 |
| seaweedfs | 1.87 MB/s | 479.5us | 579.1us | 734.6us | 0 |
| rustfs | 1.76 MB/s | 546.9us | 631.5us | 642.2us | 0 |
| localstack | 1.27 MB/s | 740.4us | 894.5us | 936.5us | 0 |

```
  liteio       ████████████████████████████████████████ 4.92 MB/s
  liteio_mem   █████████████████████████████████████ 4.60 MB/s
  minio        ██████████████████ 2.23 MB/s
  seaweedfs    ███████████████ 1.87 MB/s
  rustfs       ██████████████ 1.76 MB/s
  localstack   ██████████ 1.27 MB/s
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 1.08 MB/s | 906.1us | 1.2ms | 1.3ms | 0 |
| liteio_mem | 1.02 MB/s | 903.9us | 1.5ms | 1.5ms | 0 |
| rustfs | 0.91 MB/s | 1.1ms | 1.3ms | 1.3ms | 0 |
| liteio | 0.90 MB/s | 995.2us | 1.8ms | 1.8ms | 0 |
| seaweedfs | 0.68 MB/s | 1.4ms | 1.8ms | 2.1ms | 0 |
| localstack | 0.19 MB/s | 5.2ms | 7.5ms | 7.9ms | 0 |

```
  minio        ████████████████████████████████████████ 1.08 MB/s
  liteio_mem   █████████████████████████████████████ 1.02 MB/s
  rustfs       █████████████████████████████████ 0.91 MB/s
  liteio       █████████████████████████████████ 0.90 MB/s
  seaweedfs    █████████████████████████ 0.68 MB/s
  localstack   ██████ 0.19 MB/s
```

### ParallelRead/1KB/C5

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.97 MB/s | 444.2us | 686.6us | 732.6us | 0 |
| minio | 1.76 MB/s | 552.4us | 775.0us | 795.0us | 0 |
| liteio_mem | 1.69 MB/s | 557.8us | 891.9us | 919.8us | 0 |
| rustfs | 1.38 MB/s | 701.5us | 938.8us | 1.0ms | 0 |
| seaweedfs | 1.35 MB/s | 686.5us | 986.4us | 1.1ms | 0 |
| localstack | 0.34 MB/s | 2.5ms | 4.0ms | 4.1ms | 0 |

```
  liteio       ████████████████████████████████████████ 1.97 MB/s
  minio        ███████████████████████████████████ 1.76 MB/s
  liteio_mem   ██████████████████████████████████ 1.69 MB/s
  rustfs       ████████████████████████████ 1.38 MB/s
  seaweedfs    ███████████████████████████ 1.35 MB/s
  localstack   ██████ 0.34 MB/s
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.45 MB/s | 660.0us | 750.9us | 771.7us | 0 |
| liteio | 1.40 MB/s | 649.0us | 928.7us | 934.0us | 0 |
| liteio_mem | 1.35 MB/s | 666.4us | 942.8us | 1.3ms | 0 |
| localstack | 1.26 MB/s | 746.8us | 911.1us | 922.0us | 0 |
| rustfs | 1.21 MB/s | 798.2us | 895.1us | 973.2us | 0 |
| minio | 1.02 MB/s | 905.2us | 1.1ms | 1.2ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 1.45 MB/s
  liteio       ██████████████████████████████████████ 1.40 MB/s
  liteio_mem   █████████████████████████████████████ 1.35 MB/s
  localstack   ██████████████████████████████████ 1.26 MB/s
  rustfs       █████████████████████████████████ 1.21 MB/s
  minio        ████████████████████████████ 1.02 MB/s
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.42 MB/s | 2.2ms | 3.4ms | 3.5ms | 0 |
| rustfs | 0.39 MB/s | 2.4ms | 3.6ms | 3.7ms | 0 |
| liteio | 0.37 MB/s | 2.3ms | 4.5ms | 4.6ms | 0 |
| minio | 0.33 MB/s | 2.7ms | 4.6ms | 4.7ms | 0 |
| liteio_mem | 0.32 MB/s | 3.0ms | 3.9ms | 4.3ms | 0 |
| localstack | 0.19 MB/s | 4.3ms | 11.1ms | 12.1ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.42 MB/s
  rustfs       █████████████████████████████████████ 0.39 MB/s
  liteio       ██████████████████████████████████ 0.37 MB/s
  minio        ███████████████████████████████ 0.33 MB/s
  liteio_mem   ██████████████████████████████ 0.32 MB/s
  localstack   █████████████████ 0.19 MB/s
```

### ParallelWrite/1KB/C5

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.86 MB/s | 1.1ms | 1.8ms | 1.8ms | 0 |
| seaweedfs | 0.66 MB/s | 1.3ms | 2.7ms | 2.8ms | 0 |
| liteio | 0.57 MB/s | 1.4ms | 3.8ms | 3.8ms | 0 |
| minio | 0.56 MB/s | 1.6ms | 2.4ms | 2.4ms | 0 |
| liteio_mem | 0.51 MB/s | 1.8ms | 2.7ms | 3.4ms | 0 |
| localstack | 0.35 MB/s | 2.7ms | 4.3ms | 5.5ms | 0 |

```
  rustfs       ████████████████████████████████████████ 0.86 MB/s
  seaweedfs    ███████████████████████████████ 0.66 MB/s
  liteio       ██████████████████████████ 0.57 MB/s
  minio        ██████████████████████████ 0.56 MB/s
  liteio_mem   ████████████████████████ 0.51 MB/s
  localstack   ████████████████ 0.35 MB/s
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 250.21 MB/s | 982.9us | 1.1ms | 1.1ms | 0 |
| liteio | 242.70 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |
| seaweedfs | 187.10 MB/s | 1.3ms | 1.6ms | 1.7ms | 0 |
| minio | 164.43 MB/s | 1.5ms | 1.8ms | 1.8ms | 0 |
| localstack | 159.26 MB/s | 1.5ms | 1.7ms | 1.8ms | 0 |
| rustfs | 130.88 MB/s | 1.9ms | 2.1ms | 2.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 250.21 MB/s
  liteio       ██████████████████████████████████████ 242.70 MB/s
  seaweedfs    █████████████████████████████ 187.10 MB/s
  minio        ██████████████████████████ 164.43 MB/s
  localstack   █████████████████████████ 159.26 MB/s
  rustfs       ████████████████████ 130.88 MB/s
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 236.05 MB/s | 991.6us | 1.2ms | 1.2ms | 0 |
| liteio | 221.56 MB/s | 1.1ms | 1.4ms | 1.6ms | 0 |
| minio | 181.23 MB/s | 1.4ms | 1.5ms | 1.5ms | 0 |
| localstack | 160.83 MB/s | 1.5ms | 1.7ms | 1.7ms | 0 |
| seaweedfs | 138.92 MB/s | 1.6ms | 2.9ms | 3.0ms | 0 |
| rustfs | 95.58 MB/s | 2.0ms | 3.8ms | 4.8ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 236.05 MB/s
  liteio       █████████████████████████████████████ 221.56 MB/s
  minio        ██████████████████████████████ 181.23 MB/s
  localstack   ███████████████████████████ 160.83 MB/s
  seaweedfs    ███████████████████████ 138.92 MB/s
  rustfs       ████████████████ 95.58 MB/s
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 248.89 MB/s | 979.5us | 1.1ms | 1.2ms | 0 |
| liteio_mem | 240.07 MB/s | 998.3us | 1.2ms | 1.3ms | 0 |
| minio | 177.63 MB/s | 1.4ms | 1.6ms | 1.6ms | 0 |
| seaweedfs | 165.16 MB/s | 1.5ms | 1.7ms | 1.9ms | 0 |
| localstack | 161.68 MB/s | 1.5ms | 1.8ms | 1.8ms | 0 |
| rustfs | 107.93 MB/s | 1.9ms | 3.2ms | 3.6ms | 0 |

```
  liteio       ████████████████████████████████████████ 248.89 MB/s
  liteio_mem   ██████████████████████████████████████ 240.07 MB/s
  minio        ████████████████████████████ 177.63 MB/s
  seaweedfs    ██████████████████████████ 165.16 MB/s
  localstack   █████████████████████████ 161.68 MB/s
  rustfs       █████████████████ 107.93 MB/s
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 4.72 MB/s | 211.0us | 231.5us | 233.5us | 0 |
| liteio | 4.23 MB/s | 225.5us | 262.1us | 267.8us | 0 |
| minio | 3.27 MB/s | 294.7us | 327.2us | 332.7us | 0 |
| seaweedfs | 2.91 MB/s | 333.3us | 362.2us | 363.8us | 0 |
| rustfs | 1.90 MB/s | 512.0us | 612.9us | 617.3us | 0 |
| localstack | 1.42 MB/s | 651.0us | 731.4us | 946.6us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4.72 MB/s
  liteio       ███████████████████████████████████ 4.23 MB/s
  minio        ███████████████████████████ 3.27 MB/s
  seaweedfs    ████████████████████████ 2.91 MB/s
  rustfs       ████████████████ 1.90 MB/s
  localstack   ████████████ 1.42 MB/s
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 294.45 MB/s | 3.4ms | 3.6ms | 3.6ms | 0 |
| liteio_mem | 292.97 MB/s | 3.4ms | 3.5ms | 3.6ms | 0 |
| minio | 260.04 MB/s | 3.8ms | 4.0ms | 4.0ms | 0 |
| localstack | 241.03 MB/s | 4.0ms | 5.1ms | 5.2ms | 0 |
| seaweedfs | 236.05 MB/s | 4.1ms | 4.6ms | 6.0ms | 0 |
| rustfs | 217.10 MB/s | 4.5ms | 4.9ms | 5.0ms | 0 |

```
  liteio       ████████████████████████████████████████ 294.45 MB/s
  liteio_mem   ███████████████████████████████████████ 292.97 MB/s
  minio        ███████████████████████████████████ 260.04 MB/s
  localstack   ████████████████████████████████ 241.03 MB/s
  seaweedfs    ████████████████████████████████ 236.05 MB/s
  rustfs       █████████████████████████████ 217.10 MB/s
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 136.21 MB/s | 443.3us | 549.0us | 577.2us | 0 |
| minio | 117.16 MB/s | 518.6us | 558.3us | 573.1us | 0 |
| liteio | 114.88 MB/s | 439.9us | 1.1ms | 1.2ms | 0 |
| seaweedfs | 88.77 MB/s | 672.5us | 760.9us | 779.3us | 0 |
| rustfs | 83.01 MB/s | 709.2us | 921.5us | 927.8us | 0 |
| localstack | 72.72 MB/s | 836.3us | 884.0us | 929.1us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 136.21 MB/s
  minio        ██████████████████████████████████ 117.16 MB/s
  liteio       █████████████████████████████████ 114.88 MB/s
  seaweedfs    ██████████████████████████ 88.77 MB/s
  rustfs       ████████████████████████ 83.01 MB/s
  localstack   █████████████████████ 72.72 MB/s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 6531 ops/s | 140.1us | 178.0us | 197.9us | 0 |
| liteio | 6211 ops/s | 151.1us | 182.7us | 200.0us | 0 |
| minio | 3780 ops/s | 251.2us | 309.1us | 377.1us | 0 |
| seaweedfs | 3584 ops/s | 277.2us | 337.2us | 363.1us | 0 |
| rustfs | 3536 ops/s | 278.6us | 331.2us | 335.2us | 0 |
| localstack | 1594 ops/s | 616.3us | 681.3us | 705.5us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 6531 ops/s
  liteio       ██████████████████████████████████████ 6211 ops/s
  minio        ███████████████████████ 3780 ops/s
  seaweedfs    █████████████████████ 3584 ops/s
  rustfs       █████████████████████ 3536 ops/s
  localstack   █████████ 1594 ops/s
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.38 MB/s | 683.6us | 808.4us | 816.5us | 0 |
| localstack | 1.35 MB/s | 699.0us | 789.2us | 879.7us | 0 |
| liteio | 1.32 MB/s | 733.5us | 858.9us | 861.9us | 0 |
| rustfs | 1.29 MB/s | 738.4us | 892.2us | 892.5us | 0 |
| liteio_mem | 1.06 MB/s | 852.4us | 1.2ms | 1.4ms | 0 |
| minio | 0.89 MB/s | 1.1ms | 1.3ms | 1.4ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 1.38 MB/s
  localstack   ███████████████████████████████████████ 1.35 MB/s
  liteio       ██████████████████████████████████████ 1.32 MB/s
  rustfs       █████████████████████████████████████ 1.29 MB/s
  liteio_mem   ██████████████████████████████ 1.06 MB/s
  minio        █████████████████████████ 0.89 MB/s
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 166.71 MB/s | 5.9ms | 6.2ms | 6.4ms | 0 |
| liteio | 164.67 MB/s | 5.9ms | 6.5ms | 6.7ms | 0 |
| rustfs | 164.28 MB/s | 5.9ms | 7.3ms | 8.3ms | 0 |
| minio | 140.97 MB/s | 7.0ms | 7.5ms | 8.2ms | 0 |
| localstack | 132.80 MB/s | 7.3ms | 8.1ms | 8.4ms | 0 |
| seaweedfs | 124.47 MB/s | 7.9ms | 8.6ms | 9.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 166.71 MB/s
  liteio       ███████████████████████████████████████ 164.67 MB/s
  rustfs       ███████████████████████████████████████ 164.28 MB/s
  minio        █████████████████████████████████ 140.97 MB/s
  localstack   ███████████████████████████████ 132.80 MB/s
  seaweedfs    █████████████████████████████ 124.47 MB/s
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 67.68 MB/s | 884.7us | 1.1ms | 1.2ms | 0 |
| liteio | 65.45 MB/s | 906.4us | 1.1ms | 1.3ms | 0 |
| localstack | 62.21 MB/s | 988.7us | 1.0ms | 1.1ms | 0 |
| seaweedfs | 53.48 MB/s | 1.1ms | 1.5ms | 1.5ms | 0 |
| minio | 51.06 MB/s | 1.2ms | 1.4ms | 1.4ms | 0 |
| rustfs | 48.82 MB/s | 1.2ms | 1.6ms | 1.8ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 67.68 MB/s
  liteio       ██████████████████████████████████████ 65.45 MB/s
  localstack   ████████████████████████████████████ 62.21 MB/s
  seaweedfs    ███████████████████████████████ 53.48 MB/s
  minio        ██████████████████████████████ 51.06 MB/s
  rustfs       ████████████████████████████ 48.82 MB/s
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 9.613MiB / 7.653GiB | 9.6 MB | - | 0.0% | (no data) | 102kB / 271MB |
| liteio_mem | 9.684MiB / 7.653GiB | 9.7 MB | - | 0.0% | (no data) | 139kB / 271MB |
| localstack | 192.2MiB / 7.653GiB | 192.2 MB | - | 0.1% | 0.0 MB | 73.8MB / 61.4kB |
| minio | 408MiB / 7.653GiB | 408.0 MB | - | 2.5% | 290.7 MB | 135MB / 295MB |
| rustfs | 227.6MiB / 7.653GiB | 227.6 MB | - | 0.1% | 290.7 MB | 47.7MB / 117MB |
| seaweedfs | 113.5MiB / 7.653GiB | 113.5 MB | - | 0.0% | (no data) | 2.72MB / 0B |

### Memory Analysis Note

> **RSS (Resident Set Size)**: Actual application memory usage.
> 
> **Cache**: Linux page cache from filesystem I/O. Disk-based drivers show higher total memory because the OS caches file pages in RAM. This memory is reclaimable and doesn't indicate a memory leak.
> 
> Memory-based drivers (like `liteio_mem`) have minimal cache because data stays in application memory (RSS), not filesystem cache.

## Recommendations

- **Best for write-heavy workloads:** liteio_mem
- **Best for read-heavy workloads:** liteio

---

*Report generated by storage benchmark CLI*
