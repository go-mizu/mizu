# Storage Benchmark Summary

**Generated:** 2026-02-19T09:25:59+07:00

## Overall Winner

**liteio** won 40/40 categories (100%)

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **liteio** | 15.6 MB/s | minio | 1.9 MB/s | 8.2x faster |
| Delete | **liteio** | 17.0K ops/s | minio | 2.3K ops/s | 7.4x faster |
| EdgeCase/DeepNested | **liteio** | 1.5 MB/s | rustfs | 0.2 MB/s | 7.8x faster |
| EdgeCase/EmptyObject | **liteio** | 16.3K ops/s | rustfs | 2.1K ops/s | 7.7x faster |
| EdgeCase/LongKey256 | **liteio** | 1.4 MB/s | rustfs | 0.2 MB/s | 7.7x faster |
| List/100 | **liteio** | 2.5K ops/s | minio | 433 ops/s | 5.8x faster |
| MixedWorkload/Balanced_50_50 | **liteio** | 4.5 MB/s | minio | 0.7 MB/s | 6.6x faster |
| MixedWorkload/ReadHeavy_90_10 | **liteio** | 5.3 MB/s | minio | 1.7 MB/s | 3.2x faster |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 4.4 MB/s | minio | 0.5 MB/s | 9.3x faster |
| Multipart/15MB_3Parts | **liteio** | 352.3 MB/s | rustfs | 314.5 MB/s | +12% |
| ParallelRead/1KB/C1 | **liteio** | 9.8 MB/s | minio | 4.4 MB/s | 2.2x faster |
| ParallelRead/1KB/C10 | **liteio** | 4.0 MB/s | minio | 2.4 MB/s | +65% |
| ParallelRead/1KB/C50 | **liteio** | 1.1 MB/s | minio | 0.5 MB/s | 2.1x faster |
| ParallelWrite/1KB/C1 | **liteio** | 9.2 MB/s | minio | 2.0 MB/s | 4.7x faster |
| ParallelWrite/1KB/C10 | **liteio** | 4.0 MB/s | minio | 0.6 MB/s | 7.2x faster |
| ParallelWrite/1KB/C50 | **liteio** | 1.0 MB/s | minio | 0.1 MB/s | 9.8x faster |
| RangeRead/End_256KB | **liteio** | 3.0 GB/s | minio | 551.2 MB/s | 5.5x faster |
| RangeRead/Middle_256KB | **liteio** | 2.8 GB/s | minio | 556.9 MB/s | 5.1x faster |
| RangeRead/Start_256KB | **liteio** | 2.7 GB/s | minio | 627.8 MB/s | 4.3x faster |
| Read/10MB | **liteio** | 8.2 GB/s | minio | 3.3 GB/s | 2.5x faster |
| Read/1KB | **liteio** | 15.2 MB/s | minio | 5.4 MB/s | 2.8x faster |
| Read/1MB | **liteio** | 5.5 GB/s | minio | 1.6 GB/s | 3.4x faster |
| Read/64KB | **liteio** | 824.1 MB/s | minio | 326.6 MB/s | 2.5x faster |
| Scale/Delete/1 | **liteio** | 15.0K ops/s | rustfs | 1.6K ops/s | 9.4x faster |
| Scale/Delete/10 | **liteio** | 1.8K ops/s | minio | 249 ops/s | 7.2x faster |
| Scale/Delete/100 | **liteio** | 191 ops/s | minio | 24 ops/s | 8.1x faster |
| Scale/Delete/1000 | **liteio** | 19 ops/s | minio | 2 ops/s | 8.3x faster |
| Scale/List/1 | **liteio** | 10.4K ops/s | rustfs | 2.1K ops/s | 4.9x faster |
| Scale/List/10 | **liteio** | 8.7K ops/s | minio | 1.5K ops/s | 5.8x faster |
| Scale/List/100 | **liteio** | 2.1K ops/s | minio | 330 ops/s | 6.5x faster |
| Scale/List/1000 | **liteio** | 284 ops/s | minio | 39 ops/s | 7.3x faster |
| Scale/Write/1 | **liteio** | 3.9 MB/s | minio | 0.5 MB/s | 8.0x faster |
| Scale/Write/10 | **liteio** | 3.9 MB/s | minio | 0.5 MB/s | 7.3x faster |
| Scale/Write/100 | **liteio** | 3.9 MB/s | minio | 0.5 MB/s | 7.5x faster |
| Scale/Write/1000 | **liteio** | 3.9 MB/s | minio | 0.5 MB/s | 7.8x faster |
| Stat | **liteio** | 16.6K ops/s | minio | 6.4K ops/s | 2.6x faster |
| Write/10MB | **liteio** | 1.3 GB/s | minio | 395.1 MB/s | 3.4x faster |
| Write/1KB | **liteio** | 13.2 MB/s | minio | 2.1 MB/s | 6.2x faster |
| Write/1MB | **liteio** | 1.3 GB/s | rustfs | 262.5 MB/s | 5.0x faster |
| Write/64KB | **liteio** | 583.7 MB/s | rustfs | 96.9 MB/s | 6.0x faster |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | liteio | 1.3 GB/s | 3.4x faster |
| Write/1KB | liteio | 13.2 MB/s | 6.2x faster |
| Write/1MB | liteio | 1.3 GB/s | 5.0x faster |
| Write/64KB | liteio | 583.7 MB/s | 6.0x faster |

### Read Operations

**Best for Read:** liteio (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | liteio | 8.2 GB/s | 2.5x faster |
| Read/1KB | liteio | 15.2 MB/s | 2.8x faster |
| Read/1MB | liteio | 5.5 GB/s | 3.4x faster |
| Read/64KB | liteio | 824.1 MB/s | 2.5x faster |

### ParallelWrite Operations

**Best for ParallelWrite:** liteio (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio | 9.2 MB/s | 4.7x faster |
| ParallelWrite/1KB/C10 | liteio | 4.0 MB/s | 7.2x faster |
| ParallelWrite/1KB/C50 | liteio | 1.0 MB/s | 9.8x faster |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 9.8 MB/s | 2.2x faster |
| ParallelRead/1KB/C10 | liteio | 4.0 MB/s | +65% |
| ParallelRead/1KB/C50 | liteio | 1.1 MB/s | 2.1x faster |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 17.0K ops/s | 7.4x faster |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 16.6K ops/s | 2.6x faster |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 2.5K ops/s | 5.8x faster |

### Copy Operations

**Best for Copy:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio | 15.6 MB/s | 8.2x faster |

### Scale Operations

**Best for Scale:** liteio (won 12/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | liteio | 15.0K ops/s | 9.4x faster |
| Scale/Delete/10 | liteio | 1.8K ops/s | 7.2x faster |
| Scale/Delete/100 | liteio | 191 ops/s | 8.1x faster |
| Scale/Delete/1000 | liteio | 19 ops/s | 8.3x faster |
| Scale/List/1 | liteio | 10.4K ops/s | 4.9x faster |
| Scale/List/10 | liteio | 8.7K ops/s | 5.8x faster |
| Scale/List/100 | liteio | 2.1K ops/s | 6.5x faster |
| Scale/List/1000 | liteio | 284 ops/s | 7.3x faster |
| Scale/Write/1 | liteio | 3.9 MB/s | 8.0x faster |
| Scale/Write/10 | liteio | 3.9 MB/s | 7.3x faster |
| Scale/Write/100 | liteio | 3.9 MB/s | 7.5x faster |
| Scale/Write/1000 | liteio | 3.9 MB/s | 7.8x faster |

---

*Generated by storage benchmark CLI*
