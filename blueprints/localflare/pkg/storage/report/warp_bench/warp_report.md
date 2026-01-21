# Warp S3 Benchmark Report

**Generated**: 2026-01-21T11:36:20+07:00

## Configuration

| Parameter | Value |
|-----------|-------|
| Duration per test | 5s |
| Concurrent clients | 16 |
| Objects | 100 |
| Object sizes | 10MiB |
| Operations | delete |
| Docker cleanup | false |
| Compose dir | ./docker/s3/all |
| Output dir | ./pkg/storage/report/warp_bench |

## Environment

| Item | Value |
|------|-------|
| Go version | go1.25.5 |
| OS/Arch | darwin/arm64 |
| Warp version | warp version (dev) - (dev) |
| Warp path | /Users/apple/bin/warp |
| Warp work dir | /Users/apple/Library/Caches/mizu/warp_bench/run-3174482749 |
| Keep work dir | false |

## Drivers

| Driver | Endpoint | Bucket | Status | Notes |
|--------|----------|--------|--------|-------|
| minio | localhost:9000 | test-bucket | benchmarked |  |

## Summary

| Driver | DELETE (MB/s) |
|--------|------------|
| minio | **0.00** |

## Winners by Operation (Avg Throughput)

| Operation | Winner | Avg MB/s | Margin vs #2 |
|-----------|--------|----------|--------------|
| DELETE | minio | 0.00 | - |

## Detailed Results

### DELETE Operations

#### Object Size: 10MiB

| Driver | Throughput (MB/s) | Δ vs best | Ops/s | Avg (ms) | P50 (ms) | P99 (ms) | Errors |
|--------|-------------------|-----------|-------|----------|----------|----------|--------|
| **minio** | 0.00 | - | 0.00 | 0.00 | 0.00 | 0.00 | 0 |


