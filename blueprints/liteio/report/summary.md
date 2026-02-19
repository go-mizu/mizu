# Storage Benchmark Summary

**Generated:** 2026-02-19T09:43:11+07:00

## Overall Winner

**liteio** won 40/40 categories (100%)

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **liteio** | 15.1 MB/s | garage | 3.8 MB/s | 3.9x faster |
| Delete | **liteio** | 18.7K ops/s | seaweedfs | 7.5K ops/s | 2.5x faster |
| EdgeCase/DeepNested | **liteio** | 1.5 MB/s | garage | 0.4 MB/s | 4.3x faster |
| EdgeCase/EmptyObject | **liteio** | 16.3K ops/s | seaweedfs | 6.5K ops/s | 2.5x faster |
| EdgeCase/LongKey256 | **liteio** | 1.4 MB/s | garage | 0.4 MB/s | 3.9x faster |
| List/100 | **liteio** | 2.5K ops/s | seaweedfs | 1.0K ops/s | 2.5x faster |
| MixedWorkload/Balanced_50_50 | **liteio** | 4.0 MB/s | seaweedfs | 1.2 MB/s | 3.4x faster |
| MixedWorkload/ReadHeavy_90_10 | **liteio** | 4.9 MB/s | seaweedfs | 1.6 MB/s | 3.1x faster |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 4.1 MB/s | seaweedfs | 0.9 MB/s | 4.6x faster |
| Multipart/15MB_3Parts | **liteio** | 371.0 MB/s | minio | 330.7 MB/s | +12% |
| ParallelRead/1KB/C1 | **liteio** | 10.5 MB/s | garage | 8.8 MB/s | +19% |
| ParallelRead/1KB/C10 | **liteio** | 3.7 MB/s | garage | 2.3 MB/s | +61% |
| ParallelRead/1KB/C50 | **liteio** | 1.0 MB/s | garage | 0.6 MB/s | +74% |
| ParallelWrite/1KB/C1 | **liteio** | 9.8 MB/s | garage | 4.1 MB/s | 2.4x faster |
| ParallelWrite/1KB/C10 | **liteio** | 3.8 MB/s | garage | 1.0 MB/s | 3.6x faster |
| ParallelWrite/1KB/C50 | **liteio** | 0.9 MB/s | seaweedfs | 0.2 MB/s | 3.9x faster |
| RangeRead/End_256KB | **liteio** | 2.8 GB/s | seaweedfs | 1.0 GB/s | 2.8x faster |
| RangeRead/Middle_256KB | **liteio** | 2.8 GB/s | seaweedfs | 1.0 GB/s | 2.7x faster |
| RangeRead/Start_256KB | **liteio** | 2.6 GB/s | seaweedfs | 1.0 GB/s | 2.6x faster |
| Read/10MB | **liteio** | 8.3 GB/s | garage | 4.0 GB/s | 2.1x faster |
| Read/1KB | **liteio** | 16.6 MB/s | garage | 12.9 MB/s | +29% |
| Read/1MB | **liteio** | 5.8 GB/s | seaweedfs | 2.1 GB/s | 2.8x faster |
| Read/64KB | **liteio** | 925.9 MB/s | minio | 323.6 MB/s | 2.9x faster |
| Scale/Delete/1 | **liteio** | 15.9K ops/s | seaweedfs | 5.5K ops/s | 2.9x faster |
| Scale/Delete/10 | **liteio** | 1.9K ops/s | seaweedfs | 664 ops/s | 2.8x faster |
| Scale/Delete/100 | **liteio** | 149 ops/s | seaweedfs | 68 ops/s | 2.2x faster |
| Scale/Delete/1000 | **liteio** | 19 ops/s | seaweedfs | 8 ops/s | 2.5x faster |
| Scale/List/1 | **liteio** | 10.0K ops/s | seaweedfs | 2.8K ops/s | 3.6x faster |
| Scale/List/10 | **liteio** | 8.3K ops/s | seaweedfs | 2.2K ops/s | 3.8x faster |
| Scale/List/100 | **liteio** | 2.3K ops/s | seaweedfs | 861 ops/s | 2.7x faster |
| Scale/List/1000 | **liteio** | 280 ops/s | seaweedfs | 161 ops/s | +75% |
| Scale/Write/1 | **liteio** | 3.3 MB/s | garage | 1.2 MB/s | 2.7x faster |
| Scale/Write/10 | **liteio** | 3.9 MB/s | garage | 0.9 MB/s | 4.4x faster |
| Scale/Write/100 | **liteio** | 4.0 MB/s | garage | 0.9 MB/s | 4.5x faster |
| Scale/Write/1000 | **liteio** | 3.9 MB/s | garage | 0.9 MB/s | 4.3x faster |
| Stat | **liteio** | 18.5K ops/s | garage | 13.6K ops/s | +36% |
| Write/10MB | **liteio** | 1.3 GB/s | minio | 396.2 MB/s | 3.4x faster |
| Write/1KB | **liteio** | 15.2 MB/s | garage | 4.6 MB/s | 3.3x faster |
| Write/1MB | **liteio** | 1.3 GB/s | minio | 285.5 MB/s | 4.7x faster |
| Write/64KB | **liteio** | 604.7 MB/s | seaweedfs | 94.7 MB/s | 6.4x faster |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | liteio | 1.3 GB/s | 3.4x faster |
| Write/1KB | liteio | 15.2 MB/s | 3.3x faster |
| Write/1MB | liteio | 1.3 GB/s | 4.7x faster |
| Write/64KB | liteio | 604.7 MB/s | 6.4x faster |

### Read Operations

**Best for Read:** liteio (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | liteio | 8.3 GB/s | 2.1x faster |
| Read/1KB | liteio | 16.6 MB/s | +29% |
| Read/1MB | liteio | 5.8 GB/s | 2.8x faster |
| Read/64KB | liteio | 925.9 MB/s | 2.9x faster |

### ParallelWrite Operations

**Best for ParallelWrite:** liteio (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio | 9.8 MB/s | 2.4x faster |
| ParallelWrite/1KB/C10 | liteio | 3.8 MB/s | 3.6x faster |
| ParallelWrite/1KB/C50 | liteio | 0.9 MB/s | 3.9x faster |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 10.5 MB/s | +19% |
| ParallelRead/1KB/C10 | liteio | 3.7 MB/s | +61% |
| ParallelRead/1KB/C50 | liteio | 1.0 MB/s | +74% |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 18.7K ops/s | 2.5x faster |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 18.5K ops/s | +36% |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 2.5K ops/s | 2.5x faster |

### Copy Operations

**Best for Copy:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio | 15.1 MB/s | 3.9x faster |

### Scale Operations

**Best for Scale:** liteio (won 12/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | liteio | 15.9K ops/s | 2.9x faster |
| Scale/Delete/10 | liteio | 1.9K ops/s | 2.8x faster |
| Scale/Delete/100 | liteio | 149 ops/s | 2.2x faster |
| Scale/Delete/1000 | liteio | 19 ops/s | 2.5x faster |
| Scale/List/1 | liteio | 10.0K ops/s | 3.6x faster |
| Scale/List/10 | liteio | 8.3K ops/s | 3.8x faster |
| Scale/List/100 | liteio | 2.3K ops/s | 2.7x faster |
| Scale/List/1000 | liteio | 280 ops/s | +75% |
| Scale/Write/1 | liteio | 3.3 MB/s | 2.7x faster |
| Scale/Write/10 | liteio | 3.9 MB/s | 4.4x faster |
| Scale/Write/100 | liteio | 4.0 MB/s | 4.5x faster |
| Scale/Write/1000 | liteio | 3.9 MB/s | 4.3x faster |

---

*Generated by storage benchmark CLI*
