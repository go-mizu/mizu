# Vectorize Driver Benchmark Report

**Generated:** 2026-01-14T19:30:16+07:00

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
| hnsw | ✅ | 0 | 3734 | 66 | 98 | 14740.7 | 0 |
| mizu_vector_acorn | ✅ | 0 | 2830651 | 433 | 846 | 2156.9 | 0 |
| mizu_vector_hnsw | ✅ | 0 | 1831111 | 319 | 689 | 2960.5 | 0 |
| mizu_vector_ivf | ✅ | 0 | 1421726 | 139 | 227 | 7028.2 | 0 |
| mizu_vector_nsg | ✅ | 0 | 1838389 | 427 | 1058 | 2153.7 | 0 |
| mizu_vector_scann | ✅ | 0 | 2628493 | 225 | 562 | 4080.2 | 0 |

## Resource Usage (Docker Containers)

| Driver | Type | Memory (MB) | Memory Limit (MB) | Memory % | CPU % | Disk (MB) |
|--------|------|-------------|-------------------|----------|-------|----------|
| hnsw | Embedded | N/A | N/A | N/A | N/A | local |
| mizu_vector_acorn | Embedded | N/A | N/A | N/A | N/A | local |
| mizu_vector_hnsw | Embedded | N/A | N/A | N/A | N/A | local |
| mizu_vector_ivf | Embedded | N/A | N/A | N/A | N/A | local |
| mizu_vector_nsg | Embedded | N/A | N/A | N/A | N/A | local |
| mizu_vector_scann | Embedded | N/A | N/A | N/A | N/A | local |

## Detailed Results

### hnsw

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 1142857.1/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 101698.4/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2926543.8/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2637130.8/s | 0 |
| insert | 100 | 26.78 | 27.07 | 30.40 | 31.24 | 3733.6/s | 0 |
| search | 1000 | 0.07 | 0.07 | 0.09 | 0.10 | 14740.7/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 1023541.5/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3846153.8/s | 0 |

### mizu_vector_acorn

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 470588.2/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 105263.2/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3001200.5/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3001200.5/s | 0 |
| insert | 100 | 0.04 | 0.03 | 0.06 | 0.10 | 2830651.2/s | 0 |
| search | 1000 | 0.46 | 0.43 | 0.69 | 0.85 | 2156.9/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 974032.3/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3108969.4/s | 0 |

### mizu_vector_hnsw

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 436300.2/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 86021.5/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2636435.5/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2475247.5/s | 0 |
| insert | 100 | 0.05 | 0.04 | 0.14 | 0.18 | 1831110.8/s | 0 |
| search | 1000 | 0.34 | 0.32 | 0.50 | 0.69 | 2960.5/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 882706.0/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.01 | 0.01 | 2884504.4/s | 0 |

### mizu_vector_ivf

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 500000.0/s | 0 |
| create_index | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 600240.1/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3116235.6/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2581311.3/s | 0 |
| insert | 100 | 0.07 | 0.05 | 0.20 | 0.28 | 1421726.0/s | 0 |
| search | 1000 | 0.14 | 0.14 | 0.17 | 0.23 | 7028.2/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 864565.8/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2675370.5/s | 0 |

### mizu_vector_nsg

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 400000.0/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 109589.0/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3580379.5/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3000300.0/s | 0 |
| insert | 100 | 0.05 | 0.03 | 0.16 | 0.26 | 1838389.1/s | 0 |
| search | 1000 | 0.46 | 0.43 | 0.74 | 1.06 | 2153.7/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 893168.2/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3870867.8/s | 0 |

### mizu_vector_scann

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 521648.4/s | 0 |
| create_index | 1 | 0.04 | 0.04 | 0.04 | 0.04 | 26172.5/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2998500.7/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2448579.8/s | 0 |
| insert | 100 | 0.04 | 0.03 | 0.07 | 0.14 | 2628493.2/s | 0 |
| search | 1000 | 0.24 | 0.23 | 0.38 | 0.56 | 4080.2/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 923727.8/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3582175.1/s | 0 |

## Performance Comparison

### Insert Throughput (vectors/second)

```
hnsw            | 3733.6
mizu_vector_acorn |████████████████████████████████████████ 2830651.2
mizu_vector_hnsw |█████████████████████████ 1831110.8
mizu_vector_ivf |████████████████████ 1421726.0
mizu_vector_nsg |█████████████████████████ 1838389.1
mizu_vector_scann |█████████████████████████████████████ 2628493.2
```

### Search Latency p50 (microseconds, lower is better)

```
hnsw            |██████ 66.0
mizu_vector_acorn |████████████████████████████████████████ 433.0
mizu_vector_hnsw |█████████████████████████████ 319.0
mizu_vector_ivf |████████████ 139.0
mizu_vector_nsg |███████████████████████████████████████ 427.0
mizu_vector_scann |████████████████████ 225.0
```

### Search QPS (queries per second)

```
hnsw            |████████████████████████████████████████ 14740.7
mizu_vector_acorn |█████ 2156.9
mizu_vector_hnsw |████████ 2960.5
mizu_vector_ivf |███████████████████ 7028.2
mizu_vector_nsg |█████ 2153.7
mizu_vector_scann |███████████ 4080.2
```

---
*Generated by pkg/vectorize/bench*
