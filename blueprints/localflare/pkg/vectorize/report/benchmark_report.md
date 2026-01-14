# Vectorize Driver Benchmark Report

**Generated:** 2026-01-14T19:47:46+07:00

## Configuration

| Parameter | Value |
|-----------|-------|
| Dimensions | 384 |
| Dataset Size | 10000 |
| Batch Size | 100 |
| Search Iterations | 1000 |
| TopK | 10 |

## Summary

| Driver | Status | Connect (ms) | Insert (vec/s) | Search p50 (μs) | Search p99 (μs) | QPS | Errors |
|--------|--------|--------------|----------------|-----------------|-----------------|-----|--------|
| hnsw | ✅ | 0 | 3760 | 70 | 100 | 13953.8 | 0 |
| mizu_vector_acorn | ✅ | 0 | 3027777 | 678 | 942 | 1441.5 | 0 |
| mizu_vector_hnsw | ✅ | 0 | 2248475 | 490 | 655 | 1991.2 | 0 |
| mizu_vector_ivf | ✅ | 0 | 2143314 | 199 | 234 | 4988.1 | 0 |
| mizu_vector_nsg | ✅ | 0 | 2760779 | 848 | 1336 | 1136.0 | 0 |
| mizu_vector_scann | ✅ | 0 | 3105470 | 201 | 390 | 4672.0 | 0 |

## Resource Usage (Docker Containers)

| Driver | Type | Memory (MB) | Memory Limit (MB) | Memory % | CPU % | Disk (MB) |
|--------|------|-------------|-------------------|----------|-------|----------|
| hnsw | Embedded | - | - | - | - | - |
| mizu_vector_acorn | Embedded | - | - | - | - | - |
| mizu_vector_hnsw | Embedded | - | - | - | - | - |
| mizu_vector_ivf | Embedded | - | - | - | - | - |
| mizu_vector_nsg | Embedded | - | - | - | - | - |
| mizu_vector_scann | Embedded | - | - | - | - | - |

## Memory Usage (Embedded Drivers)

| Driver | Heap Alloc (MB) | Heap InUse (MB) | Heap Objects (K) | Bytes/Vector |
|--------|-----------------|-----------------|------------------|-------------|
| hnsw | 33.18 | 43.84 | 107.4 | 3480 |
| mizu_vector_acorn | 19.40 | 20.48 | 40.1 | 2034 |
| mizu_vector_hnsw | 19.41 | 20.45 | 40.1 | 2035 |
| mizu_vector_ivf | 19.40 | 20.52 | 40.1 | 2034 |
| mizu_vector_nsg | 19.40 | 20.55 | 40.1 | 2034 |
| mizu_vector_scann | 19.40 | 20.48 | 40.1 | 2034 |

## Detailed Results

### hnsw

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 857632.9/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 88888.9/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3242542.2/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2858776.4/s | 0 |
| insert | 100 | 26.59 | 26.90 | 29.48 | 31.08 | 3760.2/s | 0 |
| search | 1000 | 0.07 | 0.07 | 0.09 | 0.10 | 13953.8/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 971609.6/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.01 | 0.01 | 2580711.8/s | 0 |

### mizu_vector_acorn

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 470588.2/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 78690.6/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 1311647.4/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2162162.2/s | 0 |
| insert | 100 | 0.03 | 0.03 | 0.06 | 0.18 | 3027777.1/s | 0 |
| search | 1000 | 0.69 | 0.68 | 0.81 | 0.94 | 1441.5/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 825320.8/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2409522.4/s | 0 |

### mizu_vector_hnsw

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 444444.4/s | 0 |
| create_index | 1 | 0.03 | 0.03 | 0.03 | 0.03 | 33945.5/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2858776.4/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2666666.7/s | 0 |
| insert | 100 | 0.04 | 0.04 | 0.07 | 0.13 | 2248474.9/s | 0 |
| search | 1000 | 0.50 | 0.49 | 0.60 | 0.66 | 1991.2/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 862693.7/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 4240342.6/s | 0 |

### mizu_vector_ivf

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 571428.6/s | 0 |
| create_index | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 615384.6/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2759381.9/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2285714.3/s | 0 |
| insert | 100 | 0.05 | 0.04 | 0.08 | 0.19 | 2143314.4/s | 0 |
| search | 1000 | 0.20 | 0.20 | 0.22 | 0.23 | 4988.1/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 617604.2/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2663683.3/s | 0 |

### mizu_vector_nsg

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 479846.4/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 86021.5/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2962085.3/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2397506.6/s | 0 |
| insert | 100 | 0.04 | 0.03 | 0.06 | 0.15 | 2760779.0/s | 0 |
| search | 1000 | 0.88 | 0.85 | 1.12 | 1.34 | 1136.0/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 835917.7/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2952116.7/s | 0 |

### mizu_vector_scann

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 347826.1/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 85106.4/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2891845.0/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2399808.0/s | 0 |
| insert | 100 | 0.03 | 0.02 | 0.06 | 0.10 | 3105470.5/s | 0 |
| search | 1000 | 0.21 | 0.20 | 0.28 | 0.39 | 4672.0/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 842396.1/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2414467.5/s | 0 |

## Performance Comparison

### Insert Throughput (vectors/second)

```
hnsw            | 3760.2
mizu_vector_acorn |██████████████████████████████████████ 3027777.1
mizu_vector_hnsw |████████████████████████████ 2248474.9
mizu_vector_ivf |███████████████████████████ 2143314.4
mizu_vector_nsg |███████████████████████████████████ 2760779.0
mizu_vector_scann |████████████████████████████████████████ 3105470.5
```

### Search Latency p50 (microseconds, lower is better)

```
hnsw            |███ 70.0
mizu_vector_acorn |███████████████████████████████ 678.0
mizu_vector_hnsw |███████████████████████ 490.0
mizu_vector_ivf |█████████ 199.0
mizu_vector_nsg |████████████████████████████████████████ 848.0
mizu_vector_scann |█████████ 201.0
```

### Search QPS (queries per second)

```
hnsw            |████████████████████████████████████████ 13953.8
mizu_vector_acorn |████ 1441.5
mizu_vector_hnsw |█████ 1991.2
mizu_vector_ivf |██████████████ 4988.1
mizu_vector_nsg |███ 1136.0
mizu_vector_scann |█████████████ 4672.0
```

---
*Generated by pkg/vectorize/bench*
