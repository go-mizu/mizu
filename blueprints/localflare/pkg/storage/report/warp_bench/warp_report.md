# Warp S3 Benchmark Report

**Generated**: 2026-01-22T12:30:53+07:00

## Configuration

| Parameter | Value |
|-----------|-------|
| Duration per test | 5s |
| Concurrent clients | 1000 |
| Objects | 100 |
| Object sizes | 256B |
| Operations | put, get, stat, list, mixed |
| List objects | 1000 |
| List max keys | 100 |
| Docker cleanup | false |
| Compose dir | ./docker/s3/all |
| Output dir | ./pkg/storage/report/warp_bench |
| No clear | true |
| Prefix |  |
| Lookup style | path |
| Disable SHA256 | true |
| Autoterm | true |
| Autoterm duration | 5s |
| Autoterm pct | 7.50 |
| PTY wrapper | false |
| Progress interval | 0s |

## Environment

| Item | Value |
|------|-------|
| Go version | go1.25.5 |
| OS/Arch | darwin/arm64 |
| Warp version | warp version (dev) - (dev) |
| Warp path | /Users/apple/bin/warp |
| Warp work dir | /Users/apple/Library/Caches/mizu/warp_bench/run-2403155199 |
| Keep work dir | false |

## Drivers

| Driver | Endpoint | Bucket | Status | Notes |
|--------|----------|--------|--------|-------|
| minio | localhost:9000 | test-bucket | benchmarked |  |
| usagi_s3 | localhost:9301 | test-bucket | benchmarked |  |

## Summary

| Driver | PUT (MB/s) | GET (MB/s) | STAT (MB/s) | LIST (MB/s) | MIXED (MB/s) |
|--------|------------|------------|------------|------------|------------|
| minio | 0.00 | 3.05 | **0.00** | **0.00** | **0.00** |
| usagi_s3 | **1.82** | **6.05** | **0.00** | **0.00** | **0.00** |

## Winners by Operation (Avg Throughput)

| Operation | Winner | Avg MB/s | Margin vs #2 |
|-----------|--------|----------|--------------|
| PUT | usagi_s3 | 1.82 | - |
| GET | usagi_s3 | 6.05 | +98.4% |
| STAT | minio | 0.00 | - |
| LIST | minio | 0.00 | - |
| MIXED | minio | 0.00 | - |

## Detailed Results

### PUT Operations

#### Object Size: 256B

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **usagi_s3** | 1.82 | 0.0% | 7455.52 | 141.00 | 122.10 | 600.80 | 0 |
| minio | 0.00 | -100.0% | 0.00 | 0.00 | 0.00 | 0.00 | 0 |


### GET Operations

#### Object Size: 256B

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **usagi_s3** | 6.05 | 0.0% | 24793.11 | 40.70 | 40.00 | 69.80 | 0 |
| minio | 3.05 | -49.6% | 12500.26 | 88.30 | 78.60 | 271.40 | 0 |


### STAT Operations

#### Object Size: 256B

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **minio** | 0.00 | - | 8070.49 | 100.20 | 72.00 | 464.00 | 0 |
| **usagi_s3** | 0.00 | - | 26523.81 | 37.10 | 35.90 | 82.70 | 0 |


### LIST Operations

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **minio** | 0.00 | - | 9747.91 | 97.30 | 101.40 | 215.90 | 0 |
| **usagi_s3** | 0.00 | - | 17358.60 | 63.40 | 55.60 | 170.70 | 0 |


### MIXED Operations

#### Object Size: 256B

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **minio** | 0.00 | - | 0.00 | 0.00 | 0.00 | 0.00 | 1 |
| **usagi_s3** | 0.00 | - | 0.00 | 0.00 | 0.00 | 0.00 | 1 |


