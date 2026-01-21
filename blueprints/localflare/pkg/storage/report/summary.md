# Storage Benchmark Summary

**Generated:** 2026-01-21T18:43:40+07:00

## Overall Winner

**devnull_s3** won 33/48 categories (69%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| devnull_s3 | 33 | 69% |
| liteio | 13 | 27% |
| minio | 2 | 4% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **devnull_s3** | 1.4 MB/s | liteio | 1.3 MB/s | ~equal |
| Delete | **devnull_s3** | 4.1K ops/s | liteio | 4.0K ops/s | ~equal |
| EdgeCase/DeepNested | **devnull_s3** | 0.2 MB/s | liteio | 0.2 MB/s | ~equal |
| EdgeCase/EmptyObject | **devnull_s3** | 2.1K ops/s | liteio | 1.8K ops/s | +15% |
| EdgeCase/LongKey256 | **devnull_s3** | 0.2 MB/s | liteio | 0.1 MB/s | +11% |
| List/100 | **devnull_s3** | 952 ops/s | liteio | 828 ops/s | +15% |
| MixedWorkload/Balanced_50_50 | **liteio** | 0.6 MB/s | devnull_s3 | 0.5 MB/s | +11% |
| MixedWorkload/ReadHeavy_90_10 | **liteio** | 0.7 MB/s | devnull_s3 | 0.6 MB/s | +17% |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 0.4 MB/s | devnull_s3 | 0.4 MB/s | +23% |
| Multipart/15MB_3Parts | **minio** | 121.6 MB/s | devnull_s3 | 110.3 MB/s | +10% |
| ParallelRead/1KB/C1 | **devnull_s3** | 3.4 MB/s | liteio | 3.2 MB/s | ~equal |
| ParallelRead/1KB/C10 | **devnull_s3** | 1.1 MB/s | liteio | 1.1 MB/s | ~equal |
| ParallelRead/1KB/C100 | **devnull_s3** | 0.2 MB/s | liteio | 0.2 MB/s | ~equal |
| ParallelRead/1KB/C200 | **devnull_s3** | 0.1 MB/s | liteio | 0.1 MB/s | ~equal |
| ParallelRead/1KB/C25 | **liteio** | 0.6 MB/s | devnull_s3 | 0.6 MB/s | ~equal |
| ParallelRead/1KB/C50 | **devnull_s3** | 0.4 MB/s | liteio | 0.3 MB/s | ~equal |
| ParallelWrite/1KB/C1 | **devnull_s3** | 1.6 MB/s | liteio | 1.3 MB/s | +21% |
| ParallelWrite/1KB/C10 | **devnull_s3** | 0.4 MB/s | liteio | 0.4 MB/s | +11% |
| ParallelWrite/1KB/C100 | **devnull_s3** | 0.1 MB/s | liteio | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C200 | **liteio** | 0.0 MB/s | devnull_s3 | 0.0 MB/s | ~equal |
| ParallelWrite/1KB/C25 | **devnull_s3** | 0.2 MB/s | liteio | 0.2 MB/s | +14% |
| ParallelWrite/1KB/C50 | **devnull_s3** | 0.1 MB/s | liteio | 0.1 MB/s | ~equal |
| RangeRead/End_256KB | **devnull_s3** | 151.2 MB/s | liteio | 145.1 MB/s | ~equal |
| RangeRead/Middle_256KB | **devnull_s3** | 151.1 MB/s | liteio | 144.6 MB/s | ~equal |
| RangeRead/Start_256KB | **devnull_s3** | 133.5 MB/s | liteio | 132.0 MB/s | ~equal |
| Read/100MB | **devnull_s3** | 175.0 MB/s | minio | 160.8 MB/s | ~equal |
| Read/10MB | **devnull_s3** | 184.5 MB/s | minio | 178.1 MB/s | ~equal |
| Read/1KB | **devnull_s3** | 4.1 MB/s | liteio | 4.0 MB/s | ~equal |
| Read/1MB | **devnull_s3** | 173.3 MB/s | minio | 157.4 MB/s | +10% |
| Read/64KB | **devnull_s3** | 97.9 MB/s | liteio | 96.5 MB/s | ~equal |
| Scale/Delete/10 | **liteio** | 417 ops/s | devnull_s3 | 417 ops/s | ~equal |
| Scale/Delete/100 | **liteio** | 41 ops/s | devnull_s3 | 40 ops/s | ~equal |
| Scale/Delete/1000 | **liteio** | 4 ops/s | devnull_s3 | 4 ops/s | ~equal |
| Scale/Delete/10000 | **devnull_s3** | 0 ops/s | liteio | 0 ops/s | ~equal |
| Scale/List/10 | **devnull_s3** | 2.5K ops/s | liteio | 2.3K ops/s | ~equal |
| Scale/List/100 | **devnull_s3** | 902 ops/s | liteio | 897 ops/s | ~equal |
| Scale/List/1000 | **liteio** | 166 ops/s | devnull_s3 | 166 ops/s | ~equal |
| Scale/List/10000 | **minio** | 5 ops/s | devnull_s3 | 5 ops/s | +14% |
| Scale/Write/10 | **liteio** | 0.4 MB/s | devnull_s3 | 0.4 MB/s | ~equal |
| Scale/Write/100 | **liteio** | 0.4 MB/s | devnull_s3 | 0.4 MB/s | ~equal |
| Scale/Write/1000 | **liteio** | 0.4 MB/s | devnull_s3 | 0.4 MB/s | +12% |
| Scale/Write/10000 | **devnull_s3** | 0.4 MB/s | liteio | 0.4 MB/s | ~equal |
| Stat | **devnull_s3** | 4.1K ops/s | liteio | 4.0K ops/s | ~equal |
| Write/100MB | **devnull_s3** | 136.8 MB/s | liteio | 124.1 MB/s | +10% |
| Write/10MB | **liteio** | 136.4 MB/s | devnull_s3 | 127.4 MB/s | ~equal |
| Write/1KB | **devnull_s3** | 1.8 MB/s | liteio | 1.8 MB/s | ~equal |
| Write/1MB | **devnull_s3** | 114.4 MB/s | liteio | 112.6 MB/s | ~equal |
| Write/64KB | **devnull_s3** | 56.9 MB/s | liteio | 56.8 MB/s | ~equal |

## Category Summaries

### Write Operations

**Best for Write:** devnull_s3 (won 4/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | devnull_s3 | 136.8 MB/s | +10% |
| Write/10MB | liteio | 136.4 MB/s | ~equal |
| Write/1KB | devnull_s3 | 1.8 MB/s | ~equal |
| Write/1MB | devnull_s3 | 114.4 MB/s | ~equal |
| Write/64KB | devnull_s3 | 56.9 MB/s | ~equal |

### Read Operations

**Best for Read:** devnull_s3 (won 5/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | devnull_s3 | 175.0 MB/s | ~equal |
| Read/10MB | devnull_s3 | 184.5 MB/s | ~equal |
| Read/1KB | devnull_s3 | 4.1 MB/s | ~equal |
| Read/1MB | devnull_s3 | 173.3 MB/s | +10% |
| Read/64KB | devnull_s3 | 97.9 MB/s | ~equal |

### ParallelWrite Operations

**Best for ParallelWrite:** devnull_s3 (won 5/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | devnull_s3 | 1.6 MB/s | +21% |
| ParallelWrite/1KB/C10 | devnull_s3 | 0.4 MB/s | +11% |
| ParallelWrite/1KB/C100 | devnull_s3 | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C200 | liteio | 0.0 MB/s | ~equal |
| ParallelWrite/1KB/C25 | devnull_s3 | 0.2 MB/s | +14% |
| ParallelWrite/1KB/C50 | devnull_s3 | 0.1 MB/s | ~equal |

### ParallelRead Operations

**Best for ParallelRead:** devnull_s3 (won 5/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | devnull_s3 | 3.4 MB/s | ~equal |
| ParallelRead/1KB/C10 | devnull_s3 | 1.1 MB/s | ~equal |
| ParallelRead/1KB/C100 | devnull_s3 | 0.2 MB/s | ~equal |
| ParallelRead/1KB/C200 | devnull_s3 | 0.1 MB/s | ~equal |
| ParallelRead/1KB/C25 | liteio | 0.6 MB/s | ~equal |
| ParallelRead/1KB/C50 | devnull_s3 | 0.4 MB/s | ~equal |

### Delete Operations

**Best for Delete:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | devnull_s3 | 4.1K ops/s | ~equal |

### Stat Operations

**Best for Stat:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | devnull_s3 | 4.1K ops/s | ~equal |

### List Operations

**Best for List:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | devnull_s3 | 952 ops/s | +15% |

### Copy Operations

**Best for Copy:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | devnull_s3 | 1.4 MB/s | ~equal |

### Scale Operations

**Best for Scale:** liteio (won 7/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | liteio | 417 ops/s | ~equal |
| Scale/Delete/100 | liteio | 41 ops/s | ~equal |
| Scale/Delete/1000 | liteio | 4 ops/s | ~equal |
| Scale/Delete/10000 | devnull_s3 | 0 ops/s | ~equal |
| Scale/List/10 | devnull_s3 | 2.5K ops/s | ~equal |
| Scale/List/100 | devnull_s3 | 902 ops/s | ~equal |
| Scale/List/1000 | liteio | 166 ops/s | ~equal |
| Scale/List/10000 | minio | 5 ops/s | +14% |
| Scale/Write/10 | liteio | 0.4 MB/s | ~equal |
| Scale/Write/100 | liteio | 0.4 MB/s | ~equal |
| Scale/Write/1000 | liteio | 0.4 MB/s | +12% |
| Scale/Write/10000 | devnull_s3 | 0.4 MB/s | ~equal |

---

*Generated by storage benchmark CLI*
