# Vectorize Driver Benchmark Report

**Generated:** 2026-01-14T19:13:47+07:00

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
| hnsw | ✅ | 0 | 3854 | 63 | 98 | 15360.3 | 0 |
| mizu_vector_acorn | ✅ | 0 | 2877317 | 861 | 2080 | 1056.2 | 0 |
| mizu_vector_hnsw | ✅ | 0 | 976873 | 501 | 800 | 1933.5 | 0 |
| mizu_vector_ivf | ✅ | 0 | 1095964 | 204 | 448 | 4532.7 | 0 |
| mizu_vector_nsg | ✅ | 0 | 3124063 | 440 | 1094 | 2055.3 | 0 |
| mizu_vector_scann | ✅ | 0 | 2033778 | 428 | 1027 | 2263.4 | 0 |

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
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 201653.6/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 74766.4/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 4364906.2/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3380662.6/s | 0 |
| insert | 100 | 25.95 | 26.28 | 28.67 | 29.50 | 3853.6/s | 0 |
| search | 1000 | 0.07 | 0.06 | 0.08 | 0.10 | 15360.3/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 1172497.9/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3889537.1/s | 0 |

### mizu_vector_acorn

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 461467.5/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 114285.7/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 4798464.5/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 4140786.7/s | 0 |
| insert | 100 | 0.03 | 0.03 | 0.06 | 0.10 | 2877317.0/s | 0 |
| search | 1000 | 0.95 | 0.86 | 1.47 | 2.08 | 1056.2/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 1002034.1/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3200000.0/s | 0 |

### mizu_vector_hnsw

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 444444.4/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 70796.5/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3000300.0/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2449179.5/s | 0 |
| insert | 100 | 0.10 | 0.09 | 0.21 | 0.31 | 976872.8/s | 0 |
| search | 1000 | 0.52 | 0.50 | 0.66 | 0.80 | 1933.5/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 1055397.8/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2988911.1/s | 0 |

### mizu_vector_ivf

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 428449.0/s | 0 |
| create_index | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 571428.6/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2068680.2/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 1667222.4/s | 0 |
| insert | 100 | 0.09 | 0.06 | 0.20 | 0.42 | 1095964.4/s | 0 |
| search | 1000 | 0.22 | 0.20 | 0.33 | 0.45 | 4532.7/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.02 | 447505.4/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2739500.9/s | 0 |

### mizu_vector_nsg

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 510725.2/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 104810.8/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 4366812.2/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3995205.8/s | 0 |
| insert | 100 | 0.03 | 0.03 | 0.06 | 0.11 | 3124062.8/s | 0 |
| search | 1000 | 0.49 | 0.44 | 0.69 | 1.09 | 2055.3/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 1013469.0/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2749065.3/s | 0 |

### mizu_vector_scann

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 128336.8/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 110095.8/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 4526935.3/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 4288164.7/s | 0 |
| insert | 100 | 0.05 | 0.03 | 0.15 | 0.24 | 2033777.8/s | 0 |
| search | 1000 | 0.44 | 0.43 | 0.59 | 1.03 | 2263.4/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 968110.4/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3338118.0/s | 0 |

## Performance Comparison

### Insert Throughput (vectors/second)

```
hnsw            | 3853.6
mizu_vector_acorn |████████████████████████████████████ 2877317.0
mizu_vector_hnsw |████████████ 976872.8
mizu_vector_ivf |██████████████ 1095964.4
mizu_vector_nsg |████████████████████████████████████████ 3124062.8
mizu_vector_scann |██████████████████████████ 2033777.8
```

### Search Latency p50 (microseconds, lower is better)

```
hnsw            |██ 63.0
mizu_vector_acorn |████████████████████████████████████████ 861.0
mizu_vector_hnsw |███████████████████████ 501.0
mizu_vector_ivf |█████████ 204.0
mizu_vector_nsg |████████████████████ 440.0
mizu_vector_scann |███████████████████ 428.0
```

### Search QPS (queries per second)

```
hnsw            |████████████████████████████████████████ 15360.3
mizu_vector_acorn |██ 1056.2
mizu_vector_hnsw |█████ 1933.5
mizu_vector_ivf |███████████ 4532.7
mizu_vector_nsg |█████ 2055.3
mizu_vector_scann |█████ 2263.4
```

---
*Generated by pkg/vectorize/bench*
