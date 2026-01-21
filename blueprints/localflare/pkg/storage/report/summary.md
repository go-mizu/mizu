# Storage Benchmark Summary

**Generated:** 2026-01-21T17:57:03+07:00

## Overall Winner

**devnull_s3** won 25/48 categories (52%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| devnull_s3 | 25 | 52% |
| usagi_s3 | 19 | 40% |
| minio | 4 | 8% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **devnull_s3** | 1.2 MB/s | usagi_s3 | 0.9 MB/s | +33% |
| Delete | **devnull_s3** | 4.2K ops/s | usagi_s3 | 4.0K ops/s | ~equal |
| EdgeCase/DeepNested | **devnull_s3** | 0.1 MB/s | usagi_s3 | 0.1 MB/s | ~equal |
| EdgeCase/EmptyObject | **usagi_s3** | 1.6K ops/s | devnull_s3 | 1.4K ops/s | +11% |
| EdgeCase/LongKey256 | **devnull_s3** | 0.1 MB/s | usagi_s3 | 0.1 MB/s | ~equal |
| List/100 | **devnull_s3** | 960 ops/s | usagi_s3 | 947 ops/s | ~equal |
| MixedWorkload/Balanced_50_50 | **devnull_s3** | 0.4 MB/s | minio | 0.3 MB/s | +33% |
| MixedWorkload/ReadHeavy_90_10 | **devnull_s3** | 0.6 MB/s | usagi_s3 | 0.5 MB/s | +20% |
| MixedWorkload/WriteHeavy_10_90 | **devnull_s3** | 0.2 MB/s | usagi_s3 | 0.2 MB/s | ~equal |
| Multipart/15MB_3Parts | **minio** | 133.0 MB/s | usagi_s3 | 109.1 MB/s | +22% |
| ParallelRead/1KB/C1 | **usagi_s3** | 3.5 MB/s | devnull_s3 | 3.5 MB/s | ~equal |
| ParallelRead/1KB/C10 | **usagi_s3** | 1.1 MB/s | devnull_s3 | 1.1 MB/s | ~equal |
| ParallelRead/1KB/C100 | **devnull_s3** | 0.2 MB/s | usagi_s3 | 0.1 MB/s | +12% |
| ParallelRead/1KB/C200 | **devnull_s3** | 0.1 MB/s | usagi_s3 | 0.1 MB/s | +56% |
| ParallelRead/1KB/C25 | **usagi_s3** | 0.6 MB/s | devnull_s3 | 0.6 MB/s | ~equal |
| ParallelRead/1KB/C50 | **usagi_s3** | 0.3 MB/s | devnull_s3 | 0.3 MB/s | ~equal |
| ParallelWrite/1KB/C1 | **usagi_s3** | 1.4 MB/s | devnull_s3 | 1.4 MB/s | ~equal |
| ParallelWrite/1KB/C10 | **usagi_s3** | 0.4 MB/s | devnull_s3 | 0.4 MB/s | ~equal |
| ParallelWrite/1KB/C100 | **devnull_s3** | 0.1 MB/s | usagi_s3 | 0.0 MB/s | +37% |
| ParallelWrite/1KB/C200 | **devnull_s3** | 0.0 MB/s | usagi_s3 | 0.0 MB/s | +76% |
| ParallelWrite/1KB/C25 | **usagi_s3** | 0.1 MB/s | minio | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C50 | **usagi_s3** | 0.1 MB/s | devnull_s3 | 0.1 MB/s | ~equal |
| RangeRead/End_256KB | **devnull_s3** | 146.9 MB/s | minio | 130.8 MB/s | +12% |
| RangeRead/Middle_256KB | **devnull_s3** | 147.3 MB/s | minio | 129.4 MB/s | +14% |
| RangeRead/Start_256KB | **devnull_s3** | 139.1 MB/s | minio | 107.3 MB/s | +30% |
| Read/100MB | **minio** | 242.3 MB/s | usagi_s3 | 212.0 MB/s | +14% |
| Read/10MB | **minio** | 236.2 MB/s | usagi_s3 | 204.9 MB/s | +15% |
| Read/1KB | **usagi_s3** | 4.3 MB/s | devnull_s3 | 4.3 MB/s | ~equal |
| Read/1MB | **usagi_s3** | 206.7 MB/s | minio | 202.9 MB/s | ~equal |
| Read/64KB | **usagi_s3** | 113.1 MB/s | devnull_s3 | 106.8 MB/s | ~equal |
| Scale/Delete/10 | **devnull_s3** | 409 ops/s | usagi_s3 | 378 ops/s | ~equal |
| Scale/Delete/100 | **usagi_s3** | 37 ops/s | devnull_s3 | 36 ops/s | ~equal |
| Scale/Delete/1000 | **devnull_s3** | 4 ops/s | usagi_s3 | 3 ops/s | ~equal |
| Scale/Delete/10000 | **devnull_s3** | 0 ops/s | usagi_s3 | 0 ops/s | ~equal |
| Scale/List/10 | **devnull_s3** | 2.1K ops/s | usagi_s3 | 1.7K ops/s | +27% |
| Scale/List/100 | **devnull_s3** | 901 ops/s | usagi_s3 | 809 ops/s | +11% |
| Scale/List/1000 | **devnull_s3** | 145 ops/s | usagi_s3 | 114 ops/s | +28% |
| Scale/List/10000 | **minio** | 6 ops/s | devnull_s3 | 4 ops/s | +34% |
| Scale/Write/10 | **devnull_s3** | 0.4 MB/s | usagi_s3 | 0.4 MB/s | ~equal |
| Scale/Write/100 | **usagi_s3** | 0.4 MB/s | devnull_s3 | 0.4 MB/s | ~equal |
| Scale/Write/1000 | **devnull_s3** | 0.4 MB/s | usagi_s3 | 0.3 MB/s | +25% |
| Scale/Write/10000 | **devnull_s3** | 0.4 MB/s | usagi_s3 | 0.4 MB/s | +11% |
| Stat | **usagi_s3** | 4.5K ops/s | devnull_s3 | 4.3K ops/s | ~equal |
| Write/100MB | **usagi_s3** | 158.9 MB/s | minio | 152.6 MB/s | ~equal |
| Write/10MB | **usagi_s3** | 158.1 MB/s | devnull_s3 | 146.5 MB/s | ~equal |
| Write/1KB | **usagi_s3** | 1.7 MB/s | devnull_s3 | 1.7 MB/s | ~equal |
| Write/1MB | **devnull_s3** | 133.7 MB/s | usagi_s3 | 126.9 MB/s | ~equal |
| Write/64KB | **usagi_s3** | 62.8 MB/s | devnull_s3 | 60.1 MB/s | ~equal |

## Category Summaries

### Write Operations

**Best for Write:** usagi_s3 (won 4/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | usagi_s3 | 158.9 MB/s | ~equal |
| Write/10MB | usagi_s3 | 158.1 MB/s | ~equal |
| Write/1KB | usagi_s3 | 1.7 MB/s | ~equal |
| Write/1MB | devnull_s3 | 133.7 MB/s | ~equal |
| Write/64KB | usagi_s3 | 62.8 MB/s | ~equal |

### Read Operations

**Best for Read:** usagi_s3 (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 242.3 MB/s | +14% |
| Read/10MB | minio | 236.2 MB/s | +15% |
| Read/1KB | usagi_s3 | 4.3 MB/s | ~equal |
| Read/1MB | usagi_s3 | 206.7 MB/s | ~equal |
| Read/64KB | usagi_s3 | 113.1 MB/s | ~equal |

### ParallelWrite Operations

**Best for ParallelWrite:** usagi_s3 (won 4/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | usagi_s3 | 1.4 MB/s | ~equal |
| ParallelWrite/1KB/C10 | usagi_s3 | 0.4 MB/s | ~equal |
| ParallelWrite/1KB/C100 | devnull_s3 | 0.1 MB/s | +37% |
| ParallelWrite/1KB/C200 | devnull_s3 | 0.0 MB/s | +76% |
| ParallelWrite/1KB/C25 | usagi_s3 | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C50 | usagi_s3 | 0.1 MB/s | ~equal |

### ParallelRead Operations

**Best for ParallelRead:** usagi_s3 (won 4/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | usagi_s3 | 3.5 MB/s | ~equal |
| ParallelRead/1KB/C10 | usagi_s3 | 1.1 MB/s | ~equal |
| ParallelRead/1KB/C100 | devnull_s3 | 0.2 MB/s | +12% |
| ParallelRead/1KB/C200 | devnull_s3 | 0.1 MB/s | +56% |
| ParallelRead/1KB/C25 | usagi_s3 | 0.6 MB/s | ~equal |
| ParallelRead/1KB/C50 | usagi_s3 | 0.3 MB/s | ~equal |

### Delete Operations

**Best for Delete:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | devnull_s3 | 4.2K ops/s | ~equal |

### Stat Operations

**Best for Stat:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | usagi_s3 | 4.5K ops/s | ~equal |

### List Operations

**Best for List:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | devnull_s3 | 960 ops/s | ~equal |

### Copy Operations

**Best for Copy:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | devnull_s3 | 1.2 MB/s | +33% |

### Scale Operations

**Best for Scale:** devnull_s3 (won 9/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | devnull_s3 | 409 ops/s | ~equal |
| Scale/Delete/100 | usagi_s3 | 37 ops/s | ~equal |
| Scale/Delete/1000 | devnull_s3 | 4 ops/s | ~equal |
| Scale/Delete/10000 | devnull_s3 | 0 ops/s | ~equal |
| Scale/List/10 | devnull_s3 | 2.1K ops/s | +27% |
| Scale/List/100 | devnull_s3 | 901 ops/s | +11% |
| Scale/List/1000 | devnull_s3 | 145 ops/s | +28% |
| Scale/List/10000 | minio | 6 ops/s | +34% |
| Scale/Write/10 | devnull_s3 | 0.4 MB/s | ~equal |
| Scale/Write/100 | usagi_s3 | 0.4 MB/s | ~equal |
| Scale/Write/1000 | devnull_s3 | 0.4 MB/s | +25% |
| Scale/Write/10000 | devnull_s3 | 0.4 MB/s | +11% |

---

*Generated by storage benchmark CLI*
