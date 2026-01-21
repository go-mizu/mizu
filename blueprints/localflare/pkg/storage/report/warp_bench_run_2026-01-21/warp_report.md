# Warp S3 Benchmark Report

**Generated**: 2026-01-21T21:27:50+07:00

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
| Output dir | ./pkg/storage/report/warp_bench_run_2026-01-21 |
| No clear | true |
| Prefix |  |
| Lookup style | path |
| Disable SHA256 | true |
| Autoterm | true |
| Autoterm duration | 15s |
| Autoterm pct | 7.50 |

## Environment

| Item | Value |
|------|-------|
| Go version | go1.25.5 |
| OS/Arch | darwin/arm64 |
| Warp version | warp version (dev) - (dev) |
| Warp path | /Users/apple/bin/warp |
| Warp work dir | /Users/apple/Library/Caches/mizu/warp_bench/run-542596052 |
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
| devnull_s3 | 170.63 | **210.20** | **0.00** | **0.00** | 156.84 |
| minio | **205.28** | 206.84 | **0.00** | **0.00** | 153.33 |
| usagi_s3 | 203.22 | 199.18 | **0.00** | **0.00** | **159.80** |

## Winners by Operation (Avg Throughput)

| Operation | Winner | Avg MB/s | Margin vs #2 |
|-----------|--------|----------|--------------|
| PUT | minio | 205.28 | +1.0% |
| GET | devnull_s3 | 210.20 | +1.6% |
| STAT | devnull_s3 | 0.00 | - |
| LIST | devnull_s3 | 0.00 | - |
| MIXED | usagi_s3 | 159.80 | +1.9% |

## Detailed Results

### PUT Operations

#### Object Size: 1MiB

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **minio** | 205.28 | 0.0% | 205.28 | 95.70 | 66.10 | 378.60 | 0 |
| usagi_s3 | 203.22 | -1.0% | 203.22 | 101.80 | 60.00 | 459.90 | 0 |
| devnull_s3 | 170.63 | -16.9% | 170.63 | 135.00 | 53.60 | 4032.60 | 0 |


### GET Operations

#### Object Size: 1MiB

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **devnull_s3** | 210.20 | 0.0% | 210.20 | 95.80 | 95.50 | 132.50 | 0 |
| minio | 206.84 | -1.6% | 206.84 | 93.40 | 95.20 | 124.10 | 0 |
| usagi_s3 | 199.18 | -5.2% | 199.18 | 104.00 | 98.40 | 400.10 | 0 |


### STAT Operations

#### Object Size: 1MiB

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **minio** | 0.00 | - | 14230.86 | 1.40 | 1.30 | 3.70 | 0 |
| **usagi_s3** | 0.00 | - | 21630.67 | 0.90 | 0.90 | 2.10 | 0 |
| **devnull_s3** | 0.00 | - | 20224.12 | 1.10 | 1.00 | 3.10 | 0 |


### LIST Operations

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **minio** | 0.00 | - | 127102.22 | 8.20 | 6.50 | 27.60 | 0 |
| **usagi_s3** | 0.00 | - | 238581.71 | 4.10 | 3.80 | 10.70 | 0 |
| **devnull_s3** | 0.00 | - | 235169.34 | 5.50 | 4.60 | 18.20 | 0 |


### MIXED Operations

#### Object Size: 1MiB

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **usagi_s3** | 159.80 | 0.0% | 159.80 | 15.70 | 18.70 | 29.80 | 0 |
| devnull_s3 | 156.84 | -1.9% | 156.84 | 15.80 | 15.90 | 26.80 | 0 |
| minio | 153.33 | -4.0% | 153.33 | 13.90 | 13.60 | 23.80 | 0 |


