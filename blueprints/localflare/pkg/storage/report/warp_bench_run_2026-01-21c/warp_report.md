# Warp S3 Benchmark Report

**Generated**: 2026-01-21T22:30:23+07:00

## Configuration

| Parameter | Value |
|-----------|-------|
| Duration per test | 10s |
| Concurrent clients | 20 |
| Objects | 200 |
| Object sizes | 1MiB |
| Operations | put, get, stat, list, mixed |
| List objects | 1000 |
| List max keys | 100 |
| Docker cleanup | true |
| Compose dir | ./docker/s3/all |
| Output dir | ./pkg/storage/report/warp_bench_run_2026-01-21c |
| No clear | true |
| Prefix |  |
| Lookup style | path |
| Disable SHA256 | true |
| Autoterm | true |
| Autoterm duration | 15s |
| Autoterm pct | 7.50 |
| PTY wrapper | true |
| Progress interval | 0s |

## Environment

| Item | Value |
|------|-------|
| Go version | go1.25.5 |
| OS/Arch | darwin/arm64 |
| Warp version | warp version (dev) - (dev) |
| Warp path | /Users/apple/bin/warp |
| Warp work dir | /Users/apple/Library/Caches/mizu/warp_bench/run-1404567348 |
| Keep work dir | false |

## Drivers

| Driver | Endpoint | Bucket | Status | Notes |
|--------|----------|--------|--------|-------|
| devnull_s3 | localhost:9302 | test-bucket | benchmarked |  |
| minio | localhost:9000 | test-bucket | benchmarked |  |
| usagi_s3 | localhost:9301 | test-bucket | benchmarked |  |

## Summary

| Driver | PUT (MB/s) | GET (MB/s) | STAT (MB/s) | LIST (MB/s) | MIXED (MB/s) |
|--------|------------|------------|------------|------------|------------|
| devnull_s3 | 197.36 | 208.68 | **0.00** | **0.00** | 148.59 |
| minio | 203.45 | **209.46** | **0.00** | **0.00** | 152.31 |
| usagi_s3 | **208.44** | 208.16 | **0.00** | **0.00** | **156.54** |

## Winners by Operation (Avg Throughput)

| Operation | Winner | Avg MB/s | Margin vs #2 |
|-----------|--------|----------|--------------|
| PUT | usagi_s3 | 208.44 | +2.5% |
| GET | minio | 209.46 | +0.4% |
| STAT | devnull_s3 | 0.00 | - |
| LIST | devnull_s3 | 0.00 | - |
| MIXED | usagi_s3 | 156.54 | +2.8% |

## Detailed Results

### PUT Operations

#### Object Size: 1MiB

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **usagi_s3** | 208.44 | 0.0% | 208.44 | 1333.50 | 1315.90 | 1499.70 | 0 |
| minio | 203.45 | -2.4% | 203.45 | 113.50 | 75.10 | 392.60 | 0 |
| devnull_s3 | 197.36 | -5.3% | 197.36 | 113.90 | 58.50 | 1424.00 | 0 |


### GET Operations

#### Object Size: 1MiB

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **minio** | 209.46 | 0.0% | 209.46 | 94.50 | 95.80 | 154.30 | 0 |
| devnull_s3 | 208.68 | -0.4% | 208.68 | 96.40 | 95.70 | 140.60 | 0 |
| usagi_s3 | 208.16 | -0.6% | 208.16 | 98.00 | 97.60 | 136.60 | 0 |


### STAT Operations

#### Object Size: 1MiB

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **minio** | 0.00 | - | 14686.91 | 1.60 | 1.30 | 5.30 | 0 |
| **usagi_s3** | 0.00 | - | 20330.34 | 1.00 | 0.90 | 4.20 | 0 |
| **devnull_s3** | 0.00 | - | 21152.43 | 0.90 | 0.90 | 2.20 | 0 |


### LIST Operations

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **minio** | 0.00 | - | 126143.74 | 8.00 | 6.40 | 25.80 | 0 |
| **usagi_s3** | 0.00 | - | 233484.69 | 4.10 | 3.70 | 10.70 | 0 |
| **devnull_s3** | 0.00 | - | 227995.07 | 4.60 | 4.10 | 12.50 | 0 |


### MIXED Operations

#### Object Size: 1MiB

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **usagi_s3** | 156.54 | 0.0% | 156.54 | 20.30 | 19.60 | 38.80 | 0 |
| minio | 152.31 | -2.7% | 152.31 | 14.50 | 14.60 | 26.10 | 0 |
| devnull_s3 | 148.59 | -5.1% | 148.59 | 18.40 | 17.60 | 42.20 | 0 |


