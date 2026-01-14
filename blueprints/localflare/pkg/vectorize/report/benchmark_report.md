# Vectorize Driver Benchmark Report

**Generated:** 2026-01-14T18:20:20+07:00

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
| chromem | ✅ | 0 | 441634 | 3301 | 5357 | 298.4 | 0 |
| hnsw | ✅ | 0 | 3653 | 68 | 121 | 14258.9 | 0 |
| mizu_vector_hnsw | ✅ | 0 | 1870672 | 575 | 1312 | 1641.6 | 0 |
| mizu_vector_ivf | ✅ | 0 | 1510308 | 120 | 282 | 7737.6 | 0 |
| mizu_vector_lsh | ✅ | 0 | 1693649 | 154 | 342 | 6109.6 | 0 |
| mizu_vector_rabitq | ✅ | 0 | 2092728 | 922 | 1228 | 1068.6 | 0 |

## Resource Usage (Docker Containers)

| Driver | Type | Memory (MB) | Memory Limit (MB) | Memory % | CPU % | Disk (MB) |
|--------|------|-------------|-------------------|----------|-------|----------|
| chromem | Embedded | N/A | N/A | N/A | N/A | local |
| hnsw | Embedded | N/A | N/A | N/A | N/A | local |
| mizu_vector_hnsw | Embedded | N/A | N/A | N/A | N/A | local |
| mizu_vector_ivf | Embedded | N/A | N/A | N/A | N/A | local |
| mizu_vector_lsh | Embedded | N/A | N/A | N/A | N/A | local |
| mizu_vector_rabitq | Embedded | N/A | N/A | N/A | N/A | local |

## Detailed Results

### chromem

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.02 | 0.02 | 0.02 | 0.02 | 52061.6/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 66115.7/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3379520.1/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3002101.5/s | 0 |
| insert | 100 | 0.23 | 0.20 | 0.30 | 0.80 | 441634.3/s | 0 |
| search | 1000 | 3.35 | 3.30 | 3.96 | 5.36 | 298.4/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 789714.8/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.01 | 0.01 | 2446543.0/s | 0 |

### hnsw

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 1000000.0/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 105263.2/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2729257.6/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2527805.9/s | 0 |
| insert | 100 | 27.37 | 27.59 | 30.73 | 31.89 | 3653.3/s | 0 |
| search | 1000 | 0.07 | 0.07 | 0.09 | 0.12 | 14258.9/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 850079.9/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3934374.6/s | 0 |

### mizu_vector_hnsw

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 461680.5/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 107135.2/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2962963.0/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2265518.8/s | 0 |
| insert | 100 | 0.05 | 0.04 | 0.16 | 0.29 | 1870672.2/s | 0 |
| search | 1000 | 0.61 | 0.57 | 0.85 | 1.31 | 1641.6/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 1049824.7/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3080904.6/s | 0 |

### mizu_vector_ivf

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 521648.4/s | 0 |
| create_index | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 685401.0/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3000300.0/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2582644.6/s | 0 |
| insert | 100 | 0.07 | 0.04 | 0.17 | 0.28 | 1510307.5/s | 0 |
| search | 1000 | 0.13 | 0.12 | 0.17 | 0.28 | 7737.6/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 885606.2/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.01 | 0.01 | 2966830.8/s | 0 |

### mizu_vector_lsh

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 500000.0/s | 0 |
| create_index | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 857632.9/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2499375.2/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2638522.4/s | 0 |
| insert | 100 | 0.06 | 0.05 | 0.09 | 0.15 | 1693649.0/s | 0 |
| search | 1000 | 0.16 | 0.15 | 0.22 | 0.34 | 6109.6/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.01 | 893822.8/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3839803.4/s | 0 |

### mizu_vector_rabitq

| Operation | Iterations | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | Throughput | Errors |
|-----------|------------|----------|----------|----------|----------|------------|--------|
| connect | 1 | 0.00 | 0.00 | 0.00 | 0.00 | 558035.7/s | 0 |
| create_index | 1 | 0.01 | 0.01 | 0.01 | 0.01 | 102134.6/s | 0 |
| get_index | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 3158559.7/s | 0 |
| list_indexes | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 2727024.8/s | 0 |
| insert | 100 | 0.05 | 0.03 | 0.10 | 0.28 | 2092728.4/s | 0 |
| search | 1000 | 0.94 | 0.92 | 1.00 | 1.23 | 1068.6/s | 0 |
| get_single | 100 | 0.00 | 0.00 | 0.00 | 0.00 | 856560.4/s | 0 |
| get_batch | 10 | 0.00 | 0.00 | 0.00 | 0.00 | 4195862.9/s | 0 |

## Performance Comparison

### Insert Throughput (vectors/second)

```
chromem         |████████ 441634.3
hnsw            | 3653.3
mizu_vector_hnsw |███████████████████████████████████ 1870672.2
mizu_vector_ivf |████████████████████████████ 1510307.5
mizu_vector_lsh |████████████████████████████████ 1693649.0
mizu_vector_rabitq |████████████████████████████████████████ 2092728.4
```

### Search Latency p50 (microseconds, lower is better)

```
chromem         |████████████████████████████████████████ 3301.0
hnsw            | 68.0
mizu_vector_hnsw |██████ 575.0
mizu_vector_ivf |█ 120.0
mizu_vector_lsh |█ 154.0
mizu_vector_rabitq |███████████ 922.0
```

### Search QPS (queries per second)

```
chromem         | 298.4
hnsw            |████████████████████████████████████████ 14258.9
mizu_vector_hnsw |████ 1641.6
mizu_vector_ivf |█████████████████████ 7737.6
mizu_vector_lsh |█████████████████ 6109.6
mizu_vector_rabitq |██ 1068.6
```

---
*Generated by pkg/vectorize/bench*
