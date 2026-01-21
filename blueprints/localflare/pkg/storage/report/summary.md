# Storage Benchmark Summary

**Generated:** 2026-01-21T18:54:52+07:00

## Overall Winner

**usagi_s3** won 36/48 categories (75%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| usagi_s3 | 36 | 75% |
| devnull_s3 | 7 | 15% |
| minio | 5 | 10% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **usagi_s3** | 1.4 MB/s | devnull_s3 | 1.4 MB/s | ~equal |
| Delete | **usagi_s3** | 4.2K ops/s | devnull_s3 | 4.1K ops/s | ~equal |
| EdgeCase/DeepNested | **usagi_s3** | 0.2 MB/s | devnull_s3 | 0.2 MB/s | +16% |
| EdgeCase/EmptyObject | **usagi_s3** | 2.0K ops/s | devnull_s3 | 1.8K ops/s | +13% |
| EdgeCase/LongKey256 | **usagi_s3** | 0.2 MB/s | devnull_s3 | 0.2 MB/s | +11% |
| List/100 | **usagi_s3** | 1.0K ops/s | devnull_s3 | 999 ops/s | ~equal |
| MixedWorkload/Balanced_50_50 | **usagi_s3** | 0.5 MB/s | devnull_s3 | 0.5 MB/s | +11% |
| MixedWorkload/ReadHeavy_90_10 | **usagi_s3** | 0.6 MB/s | devnull_s3 | 0.6 MB/s | ~equal |
| MixedWorkload/WriteHeavy_10_90 | **usagi_s3** | 0.4 MB/s | devnull_s3 | 0.4 MB/s | ~equal |
| Multipart/15MB_3Parts | **usagi_s3** | 125.9 MB/s | minio | 112.9 MB/s | +12% |
| ParallelRead/1KB/C1 | **usagi_s3** | 3.9 MB/s | devnull_s3 | 3.5 MB/s | +10% |
| ParallelRead/1KB/C10 | **usagi_s3** | 1.2 MB/s | devnull_s3 | 1.1 MB/s | ~equal |
| ParallelRead/1KB/C100 | **usagi_s3** | 0.2 MB/s | devnull_s3 | 0.2 MB/s | ~equal |
| ParallelRead/1KB/C200 | **usagi_s3** | 0.1 MB/s | devnull_s3 | 0.1 MB/s | ~equal |
| ParallelRead/1KB/C25 | **usagi_s3** | 0.6 MB/s | devnull_s3 | 0.6 MB/s | ~equal |
| ParallelRead/1KB/C50 | **usagi_s3** | 0.4 MB/s | devnull_s3 | 0.3 MB/s | ~equal |
| ParallelWrite/1KB/C1 | **usagi_s3** | 1.6 MB/s | devnull_s3 | 1.4 MB/s | +15% |
| ParallelWrite/1KB/C10 | **usagi_s3** | 0.4 MB/s | devnull_s3 | 0.4 MB/s | ~equal |
| ParallelWrite/1KB/C100 | **devnull_s3** | 0.1 MB/s | usagi_s3 | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C200 | **usagi_s3** | 0.0 MB/s | devnull_s3 | 0.0 MB/s | ~equal |
| ParallelWrite/1KB/C25 | **usagi_s3** | 0.2 MB/s | devnull_s3 | 0.2 MB/s | ~equal |
| ParallelWrite/1KB/C50 | **usagi_s3** | 0.1 MB/s | devnull_s3 | 0.1 MB/s | ~equal |
| RangeRead/End_256KB | **usagi_s3** | 155.0 MB/s | devnull_s3 | 150.8 MB/s | ~equal |
| RangeRead/Middle_256KB | **usagi_s3** | 156.1 MB/s | devnull_s3 | 145.7 MB/s | ~equal |
| RangeRead/Start_256KB | **usagi_s3** | 135.1 MB/s | devnull_s3 | 127.9 MB/s | ~equal |
| Read/100MB | **minio** | 237.4 MB/s | usagi_s3 | 174.0 MB/s | +36% |
| Read/10MB | **minio** | 242.4 MB/s | usagi_s3 | 182.3 MB/s | +33% |
| Read/1KB | **usagi_s3** | 4.5 MB/s | devnull_s3 | 4.0 MB/s | +13% |
| Read/1MB | **minio** | 198.4 MB/s | usagi_s3 | 187.5 MB/s | ~equal |
| Read/64KB | **usagi_s3** | 116.1 MB/s | devnull_s3 | 101.9 MB/s | +14% |
| Scale/Delete/10 | **devnull_s3** | 388 ops/s | usagi_s3 | 355 ops/s | ~equal |
| Scale/Delete/100 | **usagi_s3** | 43 ops/s | devnull_s3 | 41 ops/s | ~equal |
| Scale/Delete/1000 | **devnull_s3** | 4 ops/s | usagi_s3 | 4 ops/s | ~equal |
| Scale/Delete/10000 | **usagi_s3** | 0 ops/s | devnull_s3 | 0 ops/s | ~equal |
| Scale/List/10 | **devnull_s3** | 2.2K ops/s | usagi_s3 | 2.1K ops/s | ~equal |
| Scale/List/100 | **usagi_s3** | 959 ops/s | devnull_s3 | 958 ops/s | ~equal |
| Scale/List/1000 | **usagi_s3** | 174 ops/s | devnull_s3 | 163 ops/s | ~equal |
| Scale/List/10000 | **minio** | 6 ops/s | devnull_s3 | 5 ops/s | +16% |
| Scale/Write/10 | **devnull_s3** | 0.3 MB/s | minio | 0.2 MB/s | +83% |
| Scale/Write/100 | **devnull_s3** | 0.4 MB/s | usagi_s3 | 0.3 MB/s | +38% |
| Scale/Write/1000 | **usagi_s3** | 0.4 MB/s | devnull_s3 | 0.4 MB/s | +10% |
| Scale/Write/10000 | **usagi_s3** | 0.4 MB/s | devnull_s3 | 0.4 MB/s | ~equal |
| Stat | **devnull_s3** | 4.2K ops/s | usagi_s3 | 3.6K ops/s | +16% |
| Write/100MB | **usagi_s3** | 152.3 MB/s | minio | 146.8 MB/s | ~equal |
| Write/10MB | **minio** | 143.8 MB/s | usagi_s3 | 142.5 MB/s | ~equal |
| Write/1KB | **usagi_s3** | 1.8 MB/s | devnull_s3 | 1.5 MB/s | +19% |
| Write/1MB | **usagi_s3** | 144.1 MB/s | devnull_s3 | 112.4 MB/s | +28% |
| Write/64KB | **usagi_s3** | 66.0 MB/s | devnull_s3 | 51.4 MB/s | +28% |

## Category Summaries

### Write Operations

**Best for Write:** usagi_s3 (won 4/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | usagi_s3 | 152.3 MB/s | ~equal |
| Write/10MB | minio | 143.8 MB/s | ~equal |
| Write/1KB | usagi_s3 | 1.8 MB/s | +19% |
| Write/1MB | usagi_s3 | 144.1 MB/s | +28% |
| Write/64KB | usagi_s3 | 66.0 MB/s | +28% |

### Read Operations

**Best for Read:** minio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 237.4 MB/s | +36% |
| Read/10MB | minio | 242.4 MB/s | +33% |
| Read/1KB | usagi_s3 | 4.5 MB/s | +13% |
| Read/1MB | minio | 198.4 MB/s | ~equal |
| Read/64KB | usagi_s3 | 116.1 MB/s | +14% |

### ParallelWrite Operations

**Best for ParallelWrite:** usagi_s3 (won 5/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | usagi_s3 | 1.6 MB/s | +15% |
| ParallelWrite/1KB/C10 | usagi_s3 | 0.4 MB/s | ~equal |
| ParallelWrite/1KB/C100 | devnull_s3 | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C200 | usagi_s3 | 0.0 MB/s | ~equal |
| ParallelWrite/1KB/C25 | usagi_s3 | 0.2 MB/s | ~equal |
| ParallelWrite/1KB/C50 | usagi_s3 | 0.1 MB/s | ~equal |

### ParallelRead Operations

**Best for ParallelRead:** usagi_s3 (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | usagi_s3 | 3.9 MB/s | +10% |
| ParallelRead/1KB/C10 | usagi_s3 | 1.2 MB/s | ~equal |
| ParallelRead/1KB/C100 | usagi_s3 | 0.2 MB/s | ~equal |
| ParallelRead/1KB/C200 | usagi_s3 | 0.1 MB/s | ~equal |
| ParallelRead/1KB/C25 | usagi_s3 | 0.6 MB/s | ~equal |
| ParallelRead/1KB/C50 | usagi_s3 | 0.4 MB/s | ~equal |

### Delete Operations

**Best for Delete:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | usagi_s3 | 4.2K ops/s | ~equal |

### Stat Operations

**Best for Stat:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | devnull_s3 | 4.2K ops/s | +16% |

### List Operations

**Best for List:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | usagi_s3 | 1.0K ops/s | ~equal |

### Copy Operations

**Best for Copy:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | usagi_s3 | 1.4 MB/s | ~equal |

### Scale Operations

**Best for Scale:** usagi_s3 (won 6/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | devnull_s3 | 388 ops/s | ~equal |
| Scale/Delete/100 | usagi_s3 | 43 ops/s | ~equal |
| Scale/Delete/1000 | devnull_s3 | 4 ops/s | ~equal |
| Scale/Delete/10000 | usagi_s3 | 0 ops/s | ~equal |
| Scale/List/10 | devnull_s3 | 2.2K ops/s | ~equal |
| Scale/List/100 | usagi_s3 | 959 ops/s | ~equal |
| Scale/List/1000 | usagi_s3 | 174 ops/s | ~equal |
| Scale/List/10000 | minio | 6 ops/s | +16% |
| Scale/Write/10 | devnull_s3 | 0.3 MB/s | +83% |
| Scale/Write/100 | devnull_s3 | 0.4 MB/s | +38% |
| Scale/Write/1000 | usagi_s3 | 0.4 MB/s | +10% |
| Scale/Write/10000 | usagi_s3 | 0.4 MB/s | ~equal |

---

*Generated by storage benchmark CLI*
