# Storage Benchmark Report

**Generated:** 2026-01-14T23:56:06+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 20 |
| Warmup | 5 |
| Concurrency | 10 |
| Timeout | 30s |

## Drivers Tested

- liteio (11 benchmarks)
- liteio_mem (11 benchmarks)
- localstack (11 benchmarks)
- minio (11 benchmarks)
- rustfs (11 benchmarks)
- seaweedfs (11 benchmarks)

## Performance Comparison

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5449 ops/s | 190.5us | 218.5us | 218.5us | 0 |
| liteio_mem | 5058 ops/s | 202.1us | 237.5us | 237.5us | 0 |
| minio | 1876 ops/s | 451.1us | 667.0us | 667.0us | 0 |
| localstack | 1442 ops/s | 630.1us | 948.2us | 948.2us | 0 |
| seaweedfs | 1423 ops/s | 473.9us | 1.2ms | 1.2ms | 0 |
| rustfs | 198 ops/s | 812.5us | 1.8ms | 1.8ms | 0 |

```
  liteio       ████████████████████████████████████████ 5449 ops/s
  liteio_mem   █████████████████████████████████████ 5058 ops/s
  minio        █████████████ 1876 ops/s
  localstack   ██████████ 1442 ops/s
  seaweedfs    ██████████ 1423 ops/s
  rustfs       █ 198 ops/s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1300 ops/s | 723.4us | 991.6us | 991.6us | 0 |
| liteio_mem | 1158 ops/s | 800.7us | 1.2ms | 1.2ms | 0 |
| seaweedfs | 576 ops/s | 1.7ms | 2.0ms | 2.0ms | 0 |
| minio | 428 ops/s | 2.1ms | 3.1ms | 3.1ms | 0 |
| rustfs | 153 ops/s | 6.4ms | 7.1ms | 7.1ms | 0 |
| localstack | 28 ops/s | 34.5ms | 44.0ms | 44.0ms | 0 |

```
  liteio       ████████████████████████████████████████ 1300 ops/s
  liteio_mem   ███████████████████████████████████ 1158 ops/s
  seaweedfs    █████████████████ 576 ops/s
  minio        █████████████ 428 ops/s
  rustfs       ████ 153 ops/s
  localstack   █ 28 ops/s
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.19 MB/s | 730.9us | 1.2ms | 1.2ms | 0 |
| minio | 0.71 MB/s | 1.2ms | 2.0ms | 2.0ms | 0 |
| liteio | 0.58 MB/s | 1.2ms | 3.1ms | 3.1ms | 0 |
| rustfs | 0.56 MB/s | 1.4ms | 3.1ms | 3.1ms | 0 |
| seaweedfs | 0.51 MB/s | 1.5ms | 2.7ms | 2.7ms | 0 |
| localstack | 0.16 MB/s | 5.1ms | 9.6ms | 9.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.19 MB/s
  minio        ███████████████████████ 0.71 MB/s
  liteio       ███████████████████ 0.58 MB/s
  rustfs       ███████████████████ 0.56 MB/s
  seaweedfs    █████████████████ 0.51 MB/s
  localstack   █████ 0.16 MB/s
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.31 MB/s | 2.3ms | 5.1ms | 5.1ms | 0 |
| liteio | 0.28 MB/s | 2.8ms | 5.7ms | 5.7ms | 0 |
| seaweedfs | 0.23 MB/s | 4.1ms | 6.1ms | 6.1ms | 0 |
| minio | 0.23 MB/s | 3.2ms | 6.6ms | 6.6ms | 0 |
| localstack | 0.16 MB/s | 5.3ms | 10.2ms | 10.2ms | 0 |
| rustfs | 0.15 MB/s | 5.0ms | 10.7ms | 10.7ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.31 MB/s
  liteio       ████████████████████████████████████ 0.28 MB/s
  seaweedfs    ██████████████████████████████ 0.23 MB/s
  minio        █████████████████████████████ 0.23 MB/s
  localstack   ████████████████████ 0.16 MB/s
  rustfs       ███████████████████ 0.15 MB/s
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 4.69 MB/s | 205.4us | 242.8us | 242.8us | 0 |
| liteio | 3.93 MB/s | 229.5us | 344.2us | 344.2us | 0 |
| rustfs | 2.60 MB/s | 363.0us | 431.5us | 431.5us | 0 |
| seaweedfs | 2.31 MB/s | 385.8us | 607.9us | 607.9us | 0 |
| minio | 2.29 MB/s | 351.5us | 415.3us | 415.3us | 0 |
| localstack | 1.15 MB/s | 824.9us | 956.7us | 956.7us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4.69 MB/s
  liteio       █████████████████████████████████ 3.93 MB/s
  rustfs       ██████████████████████ 2.60 MB/s
  seaweedfs    ███████████████████ 2.31 MB/s
  minio        ███████████████████ 2.29 MB/s
  localstack   █████████ 1.15 MB/s
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 289.44 MB/s | 3.4ms | 3.8ms | 3.8ms | 0 |
| liteio_mem | 287.59 MB/s | 3.4ms | 3.7ms | 3.7ms | 0 |
| seaweedfs | 245.31 MB/s | 4.0ms | 4.3ms | 4.3ms | 0 |
| localstack | 239.20 MB/s | 4.1ms | 4.6ms | 4.6ms | 0 |
| minio | 201.29 MB/s | 4.4ms | 7.8ms | 7.8ms | 0 |
| rustfs | 184.98 MB/s | 5.1ms | 6.5ms | 6.5ms | 0 |

```
  liteio       ████████████████████████████████████████ 289.44 MB/s
  liteio_mem   ███████████████████████████████████████ 287.59 MB/s
  seaweedfs    █████████████████████████████████ 245.31 MB/s
  localstack   █████████████████████████████████ 239.20 MB/s
  minio        ███████████████████████████ 201.29 MB/s
  rustfs       █████████████████████████ 184.98 MB/s
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 159.24 MB/s | 379.6us | 448.0us | 448.0us | 0 |
| liteio_mem | 139.62 MB/s | 449.1us | 495.3us | 495.3us | 0 |
| minio | 98.72 MB/s | 613.8us | 731.2us | 731.2us | 0 |
| rustfs | 91.66 MB/s | 622.5us | 949.9us | 949.9us | 0 |
| seaweedfs | 70.77 MB/s | 673.1us | 1.2ms | 1.2ms | 0 |
| localstack | 57.38 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |

```
  liteio       ████████████████████████████████████████ 159.24 MB/s
  liteio_mem   ███████████████████████████████████ 139.62 MB/s
  minio        ████████████████████████ 98.72 MB/s
  rustfs       ███████████████████████ 91.66 MB/s
  seaweedfs    █████████████████ 70.77 MB/s
  localstack   ██████████████ 57.38 MB/s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 5958 ops/s | 163.6us | 187.2us | 187.2us | 0 |
| liteio | 5091 ops/s | 188.0us | 245.1us | 245.1us | 0 |
| minio | 3360 ops/s | 293.6us | 358.8us | 358.8us | 0 |
| seaweedfs | 2896 ops/s | 333.0us | 408.9us | 408.9us | 0 |
| rustfs | 2563 ops/s | 347.8us | 550.5us | 550.5us | 0 |
| localstack | 1277 ops/s | 722.6us | 869.5us | 869.5us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 5958 ops/s
  liteio       ██████████████████████████████████ 5091 ops/s
  minio        ██████████████████████ 3360 ops/s
  seaweedfs    ███████████████████ 2896 ops/s
  rustfs       █████████████████ 2563 ops/s
  localstack   ████████ 1277 ops/s
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.14 MB/s | 716.5us | 1.2ms | 1.2ms | 0 |
| liteio_mem | 1.10 MB/s | 848.5us | 1.1ms | 1.1ms | 0 |
| seaweedfs | 0.92 MB/s | 951.8us | 1.5ms | 1.5ms | 0 |
| liteio | 0.72 MB/s | 1.3ms | 2.2ms | 2.2ms | 0 |
| localstack | 0.67 MB/s | 1.2ms | 1.5ms | 1.5ms | 0 |
| minio | 0.64 MB/s | 1.2ms | 2.4ms | 2.4ms | 0 |

```
  rustfs       ████████████████████████████████████████ 1.14 MB/s
  liteio_mem   ██████████████████████████████████████ 1.10 MB/s
  seaweedfs    ████████████████████████████████ 0.92 MB/s
  liteio       █████████████████████████ 0.72 MB/s
  localstack   ███████████████████████ 0.67 MB/s
  minio        ██████████████████████ 0.64 MB/s
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 158.68 MB/s | 6.1ms | 7.3ms | 7.3ms | 0 |
| rustfs | 142.12 MB/s | 6.8ms | 9.2ms | 9.2ms | 0 |
| liteio_mem | 139.77 MB/s | 6.7ms | 10.6ms | 10.6ms | 0 |
| minio | 122.20 MB/s | 7.8ms | 10.0ms | 10.0ms | 0 |
| localstack | 114.68 MB/s | 8.6ms | 9.5ms | 9.5ms | 0 |
| seaweedfs | 113.56 MB/s | 8.6ms | 10.4ms | 10.4ms | 0 |

```
  liteio       ████████████████████████████████████████ 158.68 MB/s
  rustfs       ███████████████████████████████████ 142.12 MB/s
  liteio_mem   ███████████████████████████████████ 139.77 MB/s
  minio        ██████████████████████████████ 122.20 MB/s
  localstack   ████████████████████████████ 114.68 MB/s
  seaweedfs    ████████████████████████████ 113.56 MB/s
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 58.04 MB/s | 1.1ms | 1.3ms | 1.3ms | 0 |
| rustfs | 50.06 MB/s | 1.2ms | 1.7ms | 1.7ms | 0 |
| liteio | 45.87 MB/s | 1.1ms | 2.7ms | 2.7ms | 0 |
| seaweedfs | 44.84 MB/s | 1.3ms | 1.9ms | 1.9ms | 0 |
| localstack | 43.41 MB/s | 1.4ms | 1.5ms | 1.5ms | 0 |
| minio | 33.86 MB/s | 1.6ms | 2.6ms | 2.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 58.04 MB/s
  rustfs       ██████████████████████████████████ 50.06 MB/s
  liteio       ███████████████████████████████ 45.87 MB/s
  seaweedfs    ██████████████████████████████ 44.84 MB/s
  localstack   █████████████████████████████ 43.41 MB/s
  minio        ███████████████████████ 33.86 MB/s
```

## Resource Usage

| Driver | Memory | CPU | Disk |
|--------|--------|-----|------|
| liteio | 378.4MiB / 7.653GiB | 0.0% | - |
| liteio_mem | 9.777MiB / 7.653GiB | 0.0% | - |
| localstack | 2.17GiB / 7.653GiB | 0.1% | - |
| minio | 416.8MiB / 7.653GiB | 0.0% | - |
| rustfs | 1.134GiB / 7.653GiB | 0.2% | - |
| seaweedfs | 72.83MiB / 7.653GiB | 0.0% | - |

## Recommendations

- **Best for write-heavy workloads:** liteio
- **Best for read-heavy workloads:** liteio

---

*Report generated by storage benchmark CLI*
