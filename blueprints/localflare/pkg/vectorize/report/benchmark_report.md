# Vectorize Driver Benchmark Report

**Generated:** 2026-01-14T20:08:08+07:00

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
| hnsw | ✅ | 0 | 3749 | 63 | 88 | 15482.8 | 0 |
| mizu_vector_acorn | ✅ | 0 | 2900198 | 919 | 2758 | 931.3 | 0 |
| mizu_vector_hnsw | ✅ | 0 | 1446018 | 501 | 697 | 1951.9 | 0 |
| mizu_vector_ivf | ✅ | 0 | 1979039 | 170 | 224 | 5732.5 | 0 |
| mizu_vector_nsg | ✅ | 0 | 3067285 | 814 | 1014 | 1208.0 | 0 |
| mizu_vector_scann | ✅ | 0 | 3149113 | 202 | 888 | 4084.8 | 0 |

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
| hnsw | 15.71 | 23.98 | 96.2 | 1647 |
| mizu_vector_acorn | 4.39 | 5.28 | 39.9 | 460 |
| mizu_vector_hnsw | 4.10 | 5.04 | 39.5 | 430 |
| mizu_vector_ivf | 19.40 | 20.59 | 40.1 | 2034 |
| mizu_vector_nsg | 2.59 | 2.27 | 29.9 | 271 |
| mizu_vector_scann | 1.94 | 2.41 | 30.1 | 203 |

## Detailed Results

### hnsw

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 374953.1/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 92310.5/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3478260.9/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2790178.6/s | 0 |
| insert | 100 | 26.67 | 26.59 | 29.90 | 31.66 | 3749.2/s | 0 |
| search | 1000 | 0.06 | 0.06 | 0.08 | 0.09 | 15482.8/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 721115.7/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2301019.4/s | 0 |

### mizu_vector_acorn

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 571428.6/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 106666.7/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2857959.4/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3001200.5/s | 0 |
| insert | 100 | 0.03 | 0.03 | 0.06 | 0.10 | 2900197.5/s | 0 |
| search | 1000 | 1.07 | 0.92 | 1.85 | 2.76 | 931.3/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 525853.6/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.01 | 0.01 | 2332252.7/s | 0 |

### mizu_vector_hnsw

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 533333.3/s | 0 |
| create_index | 1 | 0.03 | 0.03 | 0.03 | 0.03 | 35036.1/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 1677852.3/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3077870.1/s | 0 |
| insert | 100 | 0.07 | 0.07 | 0.11 | 0.14 | 1446017.6/s | 0 |
| search | 1000 | 0.51 | 0.50 | 0.62 | 0.70 | 1951.9/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 951248.5/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3187352.6/s | 0 |

### mizu_vector_ivf

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 666666.7/s | 0 |
| create_index | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 800000.0/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2961208.2/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2637130.8/s | 0 |
| insert | 100 | 0.05 | 0.04 | 0.10 | 0.18 | 1979038.8/s | 0 |
| search | 1000 | 0.17 | 0.17 | 0.20 | 0.22 | 5732.5/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.01 | 0.01 | 302497.7/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2646062.7/s | 0 |

### mizu_vector_nsg

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 452898.6/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 86956.5/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2499375.2/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2400960.4/s | 0 |
| insert | 100 | 0.03 | 0.03 | 0.06 | 0.10 | 3067285.2/s | 0 |
| search | 1000 | 0.83 | 0.81 | 0.95 | 1.01 | 1208.0/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 834167.5/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3889688.4/s | 0 |

### mizu_vector_scann

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 521920.7/s | 0 |
| create_index | 1 | 0.03 | 0.03 | 0.03 | 0.03 | 34237.2/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2925687.5/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3038590.1/s | 0 |
| insert | 100 | 0.03 | 0.03 | 0.06 | 0.12 | 3149113.4/s | 0 |
| search | 1000 | 0.24 | 0.20 | 0.47 | 0.89 | 4084.8/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 812994.9/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 4469273.7/s | 0 |

## Performance Comparison

### Insert Throughput (vectors/second)

```
hnsw            | 3749.2
mizu_vector_acorn |████████████████████████████████████ 2900197.5
mizu_vector_hnsw |██████████████████ 1446017.6
mizu_vector_ivf |█████████████████████████ 1979038.8
mizu_vector_nsg |██████████████████████████████████████ 3067285.2
mizu_vector_scann |████████████████████████████████████████ 3149113.4
```

### Search Latency p50 (microseconds, lower is better)

```
hnsw            |██ 63.0
mizu_vector_acorn |████████████████████████████████████████ 919.0
mizu_vector_hnsw |█████████████████████ 501.0
mizu_vector_ivf |███████ 170.0
mizu_vector_nsg |███████████████████████████████████ 814.0
mizu_vector_scann |████████ 202.0
```

### Search QPS (queries per second)

```
hnsw            |████████████████████████████████████████ 15482.8
mizu_vector_acorn |██ 931.3
mizu_vector_hnsw |█████ 1951.9
mizu_vector_ivf |██████████████ 5732.5
mizu_vector_nsg |███ 1208.0
mizu_vector_scann |██████████ 4084.8
```

---
*Generated by pkg/vectorize/bench*
