# Storage Benchmark Summary

**Generated:** 2026-01-21T12:14:39+07:00

## Overall Winner

**rabbit** won 42/43 categories (98%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| rabbit | 42 | 98% |
| minio | 1 | 2% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **rabbit** | 4.6 MB/s | minio | 0.5 MB/s | 10.2x faster |
| Delete | **rabbit** | 11.7K ops/s | minio | 1.3K ops/s | 9.1x faster |
| EdgeCase/DeepNested | **rabbit** | 0.6 MB/s | minio | 0.0 MB/s | 14.4x faster |
| EdgeCase/EmptyObject | **rabbit** | 3.7K ops/s | minio | 410 ops/s | 9.1x faster |
| EdgeCase/LongKey256 | **rabbit** | 0.5 MB/s | minio | 0.0 MB/s | 11.8x faster |
| FileCount/Delete/1 | **rabbit** | 4.9K ops/s | minio | 1.0K ops/s | 4.7x faster |
| FileCount/Delete/10 | **rabbit** | 1.4K ops/s | minio | 131 ops/s | 10.9x faster |
| FileCount/Delete/100 | **rabbit** | 128 ops/s | minio | 13 ops/s | 9.9x faster |
| FileCount/Delete/1000 | **rabbit** | 9 ops/s | minio | 1 ops/s | 7.9x faster |
| FileCount/Delete/10000 | **rabbit** | 1 ops/s | minio | 0 ops/s | 7.3x faster |
| FileCount/List/1 | **rabbit** | 7.6K ops/s | minio | 985 ops/s | 7.7x faster |
| FileCount/List/10 | **rabbit** | 8.2K ops/s | minio | 689 ops/s | 11.9x faster |
| FileCount/List/100 | **rabbit** | 1.2K ops/s | minio | 255 ops/s | 4.8x faster |
| FileCount/List/1000 | **rabbit** | 128 ops/s | minio | 37 ops/s | 3.4x faster |
| FileCount/List/10000 | **rabbit** | 13 ops/s | minio | 3 ops/s | 3.9x faster |
| FileCount/Write/1 | **rabbit** | 2.2 MB/s | minio | 0.4 MB/s | 5.5x faster |
| FileCount/Write/10 | **rabbit** | 6.3 MB/s | minio | 0.4 MB/s | 14.0x faster |
| FileCount/Write/100 | **rabbit** | 7.5 MB/s | minio | 0.4 MB/s | 17.0x faster |
| FileCount/Write/1000 | **rabbit** | 5.2 MB/s | minio | 0.5 MB/s | 11.4x faster |
| FileCount/Write/10000 | **rabbit** | 4.6 MB/s | minio | 0.4 MB/s | 10.8x faster |
| List/100 | **rabbit** | 2.3K ops/s | minio | 264 ops/s | 8.7x faster |
| MixedWorkload/Balanced_50_50 | **rabbit** | 0.2 MB/s | minio | 0.2 MB/s | +46% |
| MixedWorkload/ReadHeavy_90_10 | **rabbit** | 2.2 MB/s | minio | 0.3 MB/s | 8.7x faster |
| MixedWorkload/WriteHeavy_10_90 | **minio** | 0.2 MB/s | rabbit | 0.2 MB/s | +45% |
| Multipart/15MB_3Parts | **rabbit** | 241.5 MB/s | minio | 56.4 MB/s | 4.3x faster |
| ParallelRead/1KB/C1 | **rabbit** | 475.1 MB/s | minio | 1.1 MB/s | 420.4x faster |
| ParallelRead/1KB/C10 | **rabbit** | 305.0 MB/s | minio | 0.5 MB/s | 677.2x faster |
| ParallelRead/1KB/C50 | **rabbit** | 313.7 MB/s | minio | 0.1 MB/s | 2369.8x faster |
| ParallelWrite/1KB/C1 | **rabbit** | 4.0 MB/s | minio | 0.5 MB/s | 8.0x faster |
| ParallelWrite/1KB/C10 | **rabbit** | 0.8 MB/s | minio | 0.1 MB/s | 6.2x faster |
| ParallelWrite/1KB/C50 | **rabbit** | 0.1 MB/s | minio | 0.0 MB/s | 4.6x faster |
| RangeRead/End_256KB | **rabbit** | 4.4 GB/s | minio | 55.0 MB/s | 80.5x faster |
| RangeRead/Middle_256KB | **rabbit** | 4.3 GB/s | minio | 52.2 MB/s | 82.3x faster |
| RangeRead/Start_256KB | **rabbit** | 3.6 GB/s | minio | 51.0 MB/s | 71.3x faster |
| Read/10MB | **rabbit** | 1.6 GB/s | minio | 93.6 MB/s | 17.3x faster |
| Read/1KB | **rabbit** | 910.7 MB/s | minio | 1.2 MB/s | 766.4x faster |
| Read/1MB | **rabbit** | 3.8 GB/s | minio | 71.5 MB/s | 53.3x faster |
| Read/64KB | **rabbit** | 8.9 GB/s | minio | 41.7 MB/s | 212.2x faster |
| Stat | **rabbit** | 684.2K ops/s | minio | 1.3K ops/s | 545.9x faster |
| Write/10MB | **rabbit** | 1.1 GB/s | minio | 53.8 MB/s | 19.7x faster |
| Write/1KB | **rabbit** | 5.8 MB/s | minio | 0.5 MB/s | 10.8x faster |
| Write/1MB | **rabbit** | 987.0 MB/s | minio | 44.8 MB/s | 22.0x faster |
| Write/64KB | **rabbit** | 213.3 MB/s | minio | 13.8 MB/s | 15.5x faster |

## Category Summaries

### Write Operations

**Best for Write:** rabbit (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | rabbit | 1.1 GB/s | 19.7x faster |
| Write/1KB | rabbit | 5.8 MB/s | 10.8x faster |
| Write/1MB | rabbit | 987.0 MB/s | 22.0x faster |
| Write/64KB | rabbit | 213.3 MB/s | 15.5x faster |

### Read Operations

**Best for Read:** rabbit (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | rabbit | 1.6 GB/s | 17.3x faster |
| Read/1KB | rabbit | 910.7 MB/s | 766.4x faster |
| Read/1MB | rabbit | 3.8 GB/s | 53.3x faster |
| Read/64KB | rabbit | 8.9 GB/s | 212.2x faster |

### ParallelWrite Operations

**Best for ParallelWrite:** rabbit (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | rabbit | 4.0 MB/s | 8.0x faster |
| ParallelWrite/1KB/C10 | rabbit | 0.8 MB/s | 6.2x faster |
| ParallelWrite/1KB/C50 | rabbit | 0.1 MB/s | 4.6x faster |

### ParallelRead Operations

**Best for ParallelRead:** rabbit (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | rabbit | 475.1 MB/s | 420.4x faster |
| ParallelRead/1KB/C10 | rabbit | 305.0 MB/s | 677.2x faster |
| ParallelRead/1KB/C50 | rabbit | 313.7 MB/s | 2369.8x faster |

### Delete Operations

**Best for Delete:** rabbit (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | rabbit | 11.7K ops/s | 9.1x faster |

### Stat Operations

**Best for Stat:** rabbit (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | rabbit | 684.2K ops/s | 545.9x faster |

### List Operations

**Best for List:** rabbit (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | rabbit | 2.3K ops/s | 8.7x faster |

### Copy Operations

**Best for Copy:** rabbit (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | rabbit | 4.6 MB/s | 10.2x faster |

### FileCount Operations

**Best for FileCount:** rabbit (won 15/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | rabbit | 4.9K ops/s | 4.7x faster |
| FileCount/Delete/10 | rabbit | 1.4K ops/s | 10.9x faster |
| FileCount/Delete/100 | rabbit | 128 ops/s | 9.9x faster |
| FileCount/Delete/1000 | rabbit | 9 ops/s | 7.9x faster |
| FileCount/Delete/10000 | rabbit | 1 ops/s | 7.3x faster |
| FileCount/List/1 | rabbit | 7.6K ops/s | 7.7x faster |
| FileCount/List/10 | rabbit | 8.2K ops/s | 11.9x faster |
| FileCount/List/100 | rabbit | 1.2K ops/s | 4.8x faster |
| FileCount/List/1000 | rabbit | 128 ops/s | 3.4x faster |
| FileCount/List/10000 | rabbit | 13 ops/s | 3.9x faster |
| FileCount/Write/1 | rabbit | 2.2 MB/s | 5.5x faster |
| FileCount/Write/10 | rabbit | 6.3 MB/s | 14.0x faster |
| FileCount/Write/100 | rabbit | 7.5 MB/s | 17.0x faster |
| FileCount/Write/1000 | rabbit | 5.2 MB/s | 11.4x faster |
| FileCount/Write/10000 | rabbit | 4.6 MB/s | 10.8x faster |

---

*Generated by storage benchmark CLI*
