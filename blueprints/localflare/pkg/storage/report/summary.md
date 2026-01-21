# Storage Benchmark Summary

**Generated:** 2026-01-21T17:00:41+07:00

## Overall Winner

**usagi_s3** won 26/39 categories (67%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| usagi_s3 | 26 | 67% |
| minio | 13 | 33% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **usagi_s3** | 1.2 MB/s | minio | 1.0 MB/s | +22% |
| Delete | **usagi_s3** | 4.1K ops/s | minio | 2.7K ops/s | +53% |
| EdgeCase/DeepNested | **usagi_s3** | 0.1 MB/s | minio | 0.1 MB/s | +45% |
| EdgeCase/EmptyObject | **usagi_s3** | 1.7K ops/s | minio | 1.0K ops/s | +64% |
| EdgeCase/LongKey256 | **usagi_s3** | 0.1 MB/s | minio | 0.1 MB/s | +35% |
| List/100 | **minio** | 562 ops/s | usagi_s3 | 305 ops/s | +84% |
| MixedWorkload/Balanced_50_50 | **usagi_s3** | 0.5 MB/s | minio | 0.3 MB/s | +84% |
| MixedWorkload/ReadHeavy_90_10 | **usagi_s3** | 0.6 MB/s | minio | 0.5 MB/s | +22% |
| MixedWorkload/WriteHeavy_10_90 | **usagi_s3** | 0.4 MB/s | minio | 0.2 MB/s | 2.3x faster |
| Multipart/15MB_3Parts | **minio** | 132.2 MB/s | usagi_s3 | 125.6 MB/s | ~equal |
| ParallelRead/1KB/C1 | **usagi_s3** | 3.6 MB/s | minio | 2.6 MB/s | +38% |
| ParallelRead/1KB/C10 | **usagi_s3** | 1.3 MB/s | minio | 0.9 MB/s | +48% |
| ParallelRead/1KB/C100 | **usagi_s3** | 0.2 MB/s | minio | 0.1 MB/s | +39% |
| ParallelRead/1KB/C200 | **usagi_s3** | 0.1 MB/s | minio | 0.1 MB/s | +41% |
| ParallelRead/1KB/C25 | **usagi_s3** | 0.6 MB/s | minio | 0.4 MB/s | +49% |
| ParallelRead/1KB/C50 | **usagi_s3** | 0.4 MB/s | minio | 0.2 MB/s | +52% |
| ParallelWrite/1KB/C1 | **usagi_s3** | 1.2 MB/s | minio | 1.2 MB/s | ~equal |
| ParallelWrite/1KB/C10 | **usagi_s3** | 0.4 MB/s | minio | 0.3 MB/s | +59% |
| ParallelWrite/1KB/C100 | **usagi_s3** | 0.0 MB/s | minio | 0.0 MB/s | +25% |
| ParallelWrite/1KB/C200 | **usagi_s3** | 0.0 MB/s | minio | 0.0 MB/s | +34% |
| ParallelWrite/1KB/C25 | **usagi_s3** | 0.1 MB/s | minio | 0.1 MB/s | +30% |
| ParallelWrite/1KB/C50 | **usagi_s3** | 0.1 MB/s | minio | 0.1 MB/s | +38% |
| RangeRead/End_256KB | **usagi_s3** | 167.2 MB/s | minio | 143.5 MB/s | +17% |
| RangeRead/Middle_256KB | **usagi_s3** | 168.4 MB/s | minio | 144.5 MB/s | +17% |
| RangeRead/Start_256KB | **usagi_s3** | 144.2 MB/s | minio | 123.2 MB/s | +17% |
| Read/100MB | **minio** | 250.8 MB/s | usagi_s3 | 180.9 MB/s | +39% |
| Read/10MB | **minio** | 252.5 MB/s | usagi_s3 | 181.8 MB/s | +39% |
| Read/1KB | **minio** | 2.9 MB/s | usagi_s3 | 2.2 MB/s | +32% |
| Read/1MB | **minio** | 206.0 MB/s | usagi_s3 | 185.5 MB/s | +11% |
| Read/64KB | **minio** | 97.2 MB/s | usagi_s3 | 73.4 MB/s | +32% |
| Scale/Delete/10 | **usagi_s3** | 340 ops/s | minio | 253 ops/s | +34% |
| Scale/List/10 | **usagi_s3** | 2.0K ops/s | minio | 1.2K ops/s | +63% |
| Scale/Write/10 | **usagi_s3** | 1.1 MB/s | minio | 1.0 MB/s | +15% |
| Stat | **minio** | 3.6K ops/s | usagi_s3 | 1.2K ops/s | 3.0x faster |
| Write/100MB | **minio** | 155.3 MB/s | usagi_s3 | 140.9 MB/s | +10% |
| Write/10MB | **minio** | 156.5 MB/s | usagi_s3 | 107.1 MB/s | +46% |
| Write/1KB | **minio** | 1.1 MB/s | usagi_s3 | 0.6 MB/s | +98% |
| Write/1MB | **minio** | 122.6 MB/s | usagi_s3 | 98.3 MB/s | +25% |
| Write/64KB | **minio** | 45.8 MB/s | usagi_s3 | 26.6 MB/s | +72% |

## Category Summaries

### Write Operations

**Best for Write:** minio (won 5/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | minio | 155.3 MB/s | +10% |
| Write/10MB | minio | 156.5 MB/s | +46% |
| Write/1KB | minio | 1.1 MB/s | +98% |
| Write/1MB | minio | 122.6 MB/s | +25% |
| Write/64KB | minio | 45.8 MB/s | +72% |

### Read Operations

**Best for Read:** minio (won 5/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 250.8 MB/s | +39% |
| Read/10MB | minio | 252.5 MB/s | +39% |
| Read/1KB | minio | 2.9 MB/s | +32% |
| Read/1MB | minio | 206.0 MB/s | +11% |
| Read/64KB | minio | 97.2 MB/s | +32% |

### ParallelWrite Operations

**Best for ParallelWrite:** usagi_s3 (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | usagi_s3 | 1.2 MB/s | ~equal |
| ParallelWrite/1KB/C10 | usagi_s3 | 0.4 MB/s | +59% |
| ParallelWrite/1KB/C100 | usagi_s3 | 0.0 MB/s | +25% |
| ParallelWrite/1KB/C200 | usagi_s3 | 0.0 MB/s | +34% |
| ParallelWrite/1KB/C25 | usagi_s3 | 0.1 MB/s | +30% |
| ParallelWrite/1KB/C50 | usagi_s3 | 0.1 MB/s | +38% |

### ParallelRead Operations

**Best for ParallelRead:** usagi_s3 (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | usagi_s3 | 3.6 MB/s | +38% |
| ParallelRead/1KB/C10 | usagi_s3 | 1.3 MB/s | +48% |
| ParallelRead/1KB/C100 | usagi_s3 | 0.2 MB/s | +39% |
| ParallelRead/1KB/C200 | usagi_s3 | 0.1 MB/s | +41% |
| ParallelRead/1KB/C25 | usagi_s3 | 0.6 MB/s | +49% |
| ParallelRead/1KB/C50 | usagi_s3 | 0.4 MB/s | +52% |

### Delete Operations

**Best for Delete:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | usagi_s3 | 4.1K ops/s | +53% |

### Stat Operations

**Best for Stat:** minio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | minio | 3.6K ops/s | 3.0x faster |

### List Operations

**Best for List:** minio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | minio | 562 ops/s | +84% |

### Copy Operations

**Best for Copy:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | usagi_s3 | 1.2 MB/s | +22% |

### Scale Operations

**Best for Scale:** usagi_s3 (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | usagi_s3 | 340 ops/s | +34% |
| Scale/List/10 | usagi_s3 | 2.0K ops/s | +63% |
| Scale/Write/10 | usagi_s3 | 1.1 MB/s | +15% |

---

*Generated by storage benchmark CLI*
