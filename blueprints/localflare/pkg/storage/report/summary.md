# Storage Benchmark Summary

**Generated:** 2026-01-21T18:19:38+07:00

## Overall Winner

**devnull_s3** won 30/48 categories (62%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| devnull_s3 | 30 | 62% |
| usagi_s3 | 13 | 27% |
| minio | 5 | 10% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **devnull_s3** | 1.1 MB/s | usagi_s3 | 0.8 MB/s | +44% |
| Delete | **devnull_s3** | 3.4K ops/s | usagi_s3 | 3.4K ops/s | ~equal |
| EdgeCase/DeepNested | **devnull_s3** | 0.1 MB/s | usagi_s3 | 0.1 MB/s | +23% |
| EdgeCase/EmptyObject | **devnull_s3** | 1.4K ops/s | usagi_s3 | 1.2K ops/s | +22% |
| EdgeCase/LongKey256 | **devnull_s3** | 0.2 MB/s | usagi_s3 | 0.1 MB/s | +31% |
| List/100 | **usagi_s3** | 850 ops/s | devnull_s3 | 844 ops/s | ~equal |
| MixedWorkload/Balanced_50_50 | **usagi_s3** | 0.4 MB/s | devnull_s3 | 0.3 MB/s | ~equal |
| MixedWorkload/ReadHeavy_90_10 | **devnull_s3** | 0.6 MB/s | usagi_s3 | 0.5 MB/s | ~equal |
| MixedWorkload/WriteHeavy_10_90 | **usagi_s3** | 0.2 MB/s | devnull_s3 | 0.2 MB/s | +33% |
| Multipart/15MB_3Parts | **devnull_s3** | 106.6 MB/s | usagi_s3 | 99.6 MB/s | ~equal |
| ParallelRead/1KB/C1 | **devnull_s3** | 3.2 MB/s | usagi_s3 | 3.0 MB/s | ~equal |
| ParallelRead/1KB/C10 | **devnull_s3** | 1.0 MB/s | usagi_s3 | 0.9 MB/s | ~equal |
| ParallelRead/1KB/C100 | **devnull_s3** | 0.2 MB/s | usagi_s3 | 0.2 MB/s | +10% |
| ParallelRead/1KB/C200 | **devnull_s3** | 0.1 MB/s | usagi_s3 | 0.1 MB/s | ~equal |
| ParallelRead/1KB/C25 | **devnull_s3** | 0.5 MB/s | usagi_s3 | 0.5 MB/s | ~equal |
| ParallelRead/1KB/C50 | **devnull_s3** | 0.3 MB/s | usagi_s3 | 0.3 MB/s | ~equal |
| ParallelWrite/1KB/C1 | **devnull_s3** | 1.3 MB/s | usagi_s3 | 1.2 MB/s | ~equal |
| ParallelWrite/1KB/C10 | **devnull_s3** | 0.4 MB/s | usagi_s3 | 0.3 MB/s | +34% |
| ParallelWrite/1KB/C100 | **usagi_s3** | 0.0 MB/s | devnull_s3 | 0.0 MB/s | +45% |
| ParallelWrite/1KB/C200 | **usagi_s3** | 0.0 MB/s | devnull_s3 | 0.0 MB/s | +17% |
| ParallelWrite/1KB/C25 | **devnull_s3** | 0.1 MB/s | usagi_s3 | 0.1 MB/s | +25% |
| ParallelWrite/1KB/C50 | **usagi_s3** | 0.1 MB/s | devnull_s3 | 0.1 MB/s | ~equal |
| RangeRead/End_256KB | **devnull_s3** | 144.9 MB/s | usagi_s3 | 136.2 MB/s | ~equal |
| RangeRead/Middle_256KB | **devnull_s3** | 140.2 MB/s | usagi_s3 | 130.0 MB/s | ~equal |
| RangeRead/Start_256KB | **devnull_s3** | 125.0 MB/s | usagi_s3 | 124.8 MB/s | ~equal |
| Read/100MB | **minio** | 176.7 MB/s | usagi_s3 | 160.8 MB/s | ~equal |
| Read/10MB | **minio** | 175.7 MB/s | devnull_s3 | 159.1 MB/s | +10% |
| Read/1KB | **usagi_s3** | 3.9 MB/s | devnull_s3 | 3.4 MB/s | +16% |
| Read/1MB | **devnull_s3** | 161.3 MB/s | minio | 153.3 MB/s | ~equal |
| Read/64KB | **devnull_s3** | 95.4 MB/s | usagi_s3 | 89.7 MB/s | ~equal |
| Scale/Delete/10 | **usagi_s3** | 385 ops/s | devnull_s3 | 379 ops/s | ~equal |
| Scale/Delete/100 | **devnull_s3** | 37 ops/s | usagi_s3 | 36 ops/s | ~equal |
| Scale/Delete/1000 | **usagi_s3** | 4 ops/s | devnull_s3 | 4 ops/s | ~equal |
| Scale/Delete/10000 | **devnull_s3** | 0 ops/s | usagi_s3 | 0 ops/s | +11% |
| Scale/List/10 | **usagi_s3** | 1.8K ops/s | devnull_s3 | 1.5K ops/s | +24% |
| Scale/List/100 | **usagi_s3** | 932 ops/s | devnull_s3 | 686 ops/s | +36% |
| Scale/List/1000 | **devnull_s3** | 148 ops/s | usagi_s3 | 141 ops/s | ~equal |
| Scale/List/10000 | **devnull_s3** | 4 ops/s | usagi_s3 | 3 ops/s | ~equal |
| Scale/Write/10 | **devnull_s3** | 0.4 MB/s | usagi_s3 | 0.3 MB/s | +42% |
| Scale/Write/100 | **devnull_s3** | 0.4 MB/s | usagi_s3 | 0.3 MB/s | +16% |
| Scale/Write/1000 | **usagi_s3** | 0.3 MB/s | devnull_s3 | 0.3 MB/s | ~equal |
| Scale/Write/10000 | **devnull_s3** | 0.4 MB/s | usagi_s3 | 0.4 MB/s | ~equal |
| Stat | **devnull_s3** | 4.0K ops/s | usagi_s3 | 3.5K ops/s | +14% |
| Write/100MB | **minio** | 125.2 MB/s | usagi_s3 | 123.4 MB/s | ~equal |
| Write/10MB | **minio** | 142.7 MB/s | usagi_s3 | 126.7 MB/s | +13% |
| Write/1KB | **usagi_s3** | 1.6 MB/s | devnull_s3 | 1.6 MB/s | ~equal |
| Write/1MB | **minio** | 117.6 MB/s | devnull_s3 | 110.1 MB/s | ~equal |
| Write/64KB | **devnull_s3** | 64.5 MB/s | usagi_s3 | 52.5 MB/s | +23% |

## Category Summaries

### Write Operations

**Best for Write:** minio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | minio | 125.2 MB/s | ~equal |
| Write/10MB | minio | 142.7 MB/s | +13% |
| Write/1KB | usagi_s3 | 1.6 MB/s | ~equal |
| Write/1MB | minio | 117.6 MB/s | ~equal |
| Write/64KB | devnull_s3 | 64.5 MB/s | +23% |

### Read Operations

**Best for Read:** minio (won 2/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 176.7 MB/s | ~equal |
| Read/10MB | minio | 175.7 MB/s | +10% |
| Read/1KB | usagi_s3 | 3.9 MB/s | +16% |
| Read/1MB | devnull_s3 | 161.3 MB/s | ~equal |
| Read/64KB | devnull_s3 | 95.4 MB/s | ~equal |

### ParallelWrite Operations

**Best for ParallelWrite:** devnull_s3 (won 3/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | devnull_s3 | 1.3 MB/s | ~equal |
| ParallelWrite/1KB/C10 | devnull_s3 | 0.4 MB/s | +34% |
| ParallelWrite/1KB/C100 | usagi_s3 | 0.0 MB/s | +45% |
| ParallelWrite/1KB/C200 | usagi_s3 | 0.0 MB/s | +17% |
| ParallelWrite/1KB/C25 | devnull_s3 | 0.1 MB/s | +25% |
| ParallelWrite/1KB/C50 | usagi_s3 | 0.1 MB/s | ~equal |

### ParallelRead Operations

**Best for ParallelRead:** devnull_s3 (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | devnull_s3 | 3.2 MB/s | ~equal |
| ParallelRead/1KB/C10 | devnull_s3 | 1.0 MB/s | ~equal |
| ParallelRead/1KB/C100 | devnull_s3 | 0.2 MB/s | +10% |
| ParallelRead/1KB/C200 | devnull_s3 | 0.1 MB/s | ~equal |
| ParallelRead/1KB/C25 | devnull_s3 | 0.5 MB/s | ~equal |
| ParallelRead/1KB/C50 | devnull_s3 | 0.3 MB/s | ~equal |

### Delete Operations

**Best for Delete:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | devnull_s3 | 3.4K ops/s | ~equal |

### Stat Operations

**Best for Stat:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | devnull_s3 | 4.0K ops/s | +14% |

### List Operations

**Best for List:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | usagi_s3 | 850 ops/s | ~equal |

### Copy Operations

**Best for Copy:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | devnull_s3 | 1.1 MB/s | +44% |

### Scale Operations

**Best for Scale:** devnull_s3 (won 7/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | usagi_s3 | 385 ops/s | ~equal |
| Scale/Delete/100 | devnull_s3 | 37 ops/s | ~equal |
| Scale/Delete/1000 | usagi_s3 | 4 ops/s | ~equal |
| Scale/Delete/10000 | devnull_s3 | 0 ops/s | +11% |
| Scale/List/10 | usagi_s3 | 1.8K ops/s | +24% |
| Scale/List/100 | usagi_s3 | 932 ops/s | +36% |
| Scale/List/1000 | devnull_s3 | 148 ops/s | ~equal |
| Scale/List/10000 | devnull_s3 | 4 ops/s | ~equal |
| Scale/Write/10 | devnull_s3 | 0.4 MB/s | +42% |
| Scale/Write/100 | devnull_s3 | 0.4 MB/s | +16% |
| Scale/Write/1000 | usagi_s3 | 0.3 MB/s | ~equal |
| Scale/Write/10000 | devnull_s3 | 0.4 MB/s | ~equal |

---

*Generated by storage benchmark CLI*
