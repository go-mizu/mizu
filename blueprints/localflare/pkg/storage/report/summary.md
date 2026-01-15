# Storage Benchmark Summary

**Generated:** 2026-01-15T11:00:18+07:00

## Overall Winner

**liteio_mem** won 21/51 categories (41%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| liteio_mem | 21 | 41% |
| liteio | 11 | 22% |
| rustfs | 9 | 18% |
| seaweedfs | 5 | 10% |
| minio | 5 | 10% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **liteio_mem** | 1.5 MB/s | liteio | 1.3 MB/s | +11% |
| Delete | **liteio_mem** | 5.7K ops/s | liteio | 4.0K ops/s | +41% |
| EdgeCase/DeepNested | **liteio** | 0.1 MB/s | liteio_mem | 0.1 MB/s | +10% |
| EdgeCase/EmptyObject | **seaweedfs** | 2.7K ops/s | liteio_mem | 1.6K ops/s | +71% |
| EdgeCase/LongKey256 | **liteio** | 0.1 MB/s | liteio_mem | 0.1 MB/s | ~equal |
| FileCount/Delete/1 | **liteio** | 4.6K ops/s | liteio_mem | 3.5K ops/s | +31% |
| FileCount/Delete/10 | **liteio_mem** | 502 ops/s | liteio | 476 ops/s | ~equal |
| FileCount/Delete/100 | **liteio_mem** | 45 ops/s | seaweedfs | 30 ops/s | +50% |
| FileCount/Delete/1000 | **liteio_mem** | 5 ops/s | liteio | 4 ops/s | ~equal |
| FileCount/Delete/10000 | **liteio_mem** | 0 ops/s | liteio | 0 ops/s | ~equal |
| FileCount/List/1 | **liteio** | 3.9K ops/s | liteio_mem | 3.4K ops/s | +12% |
| FileCount/List/10 | **liteio_mem** | 3.2K ops/s | liteio | 2.8K ops/s | +15% |
| FileCount/List/100 | **liteio_mem** | 800 ops/s | liteio | 784 ops/s | ~equal |
| FileCount/List/1000 | **liteio_mem** | 161 ops/s | liteio | 147 ops/s | ~equal |
| FileCount/List/10000 | **seaweedfs** | 11 ops/s | minio | 6 ops/s | +73% |
| FileCount/Write/1 | **liteio_mem** | 1.5 MB/s | liteio | 1.4 MB/s | ~equal |
| FileCount/Write/10 | **liteio_mem** | 1.6 MB/s | seaweedfs | 1.4 MB/s | +15% |
| FileCount/Write/100 | **rustfs** | 1.5 MB/s | liteio_mem | 1.4 MB/s | ~equal |
| FileCount/Write/1000 | **rustfs** | 1.5 MB/s | liteio_mem | 1.4 MB/s | +12% |
| FileCount/Write/10000 | **liteio_mem** | 1.7 MB/s | liteio | 1.7 MB/s | ~equal |
| List/100 | **liteio_mem** | 1.2K ops/s | liteio | 1.1K ops/s | ~equal |
| MixedWorkload/Balanced_50_50 | **rustfs** | 6.8 MB/s | seaweedfs | 1.4 MB/s | 4.7x faster |
| MixedWorkload/ReadHeavy_90_10 | **rustfs** | 8.5 MB/s | seaweedfs | 1.6 MB/s | 5.4x faster |
| MixedWorkload/WriteHeavy_10_90 | **rustfs** | 4.4 MB/s | liteio | 1.4 MB/s | 3.1x faster |
| Multipart/15MB_3Parts | **rustfs** | 175.8 MB/s | minio | 162.9 MB/s | ~equal |
| ParallelRead/1KB/C1 | **liteio** | 2.8 MB/s | minio | 2.8 MB/s | ~equal |
| ParallelRead/1KB/C10 | **minio** | 1.1 MB/s | liteio | 0.9 MB/s | +26% |
| ParallelRead/1KB/C100 | **liteio_mem** | 0.3 MB/s | liteio | 0.3 MB/s | +19% |
| ParallelRead/1KB/C200 | **liteio_mem** | 0.6 MB/s | liteio | 0.3 MB/s | +89% |
| ParallelRead/1KB/C25 | **minio** | 0.6 MB/s | liteio_mem | 0.6 MB/s | ~equal |
| ParallelRead/1KB/C50 | **liteio_mem** | 0.5 MB/s | liteio | 0.5 MB/s | ~equal |
| ParallelWrite/1KB/C1 | **liteio_mem** | 1.5 MB/s | rustfs | 1.4 MB/s | ~equal |
| ParallelWrite/1KB/C10 | **rustfs** | 0.5 MB/s | seaweedfs | 0.4 MB/s | +22% |
| ParallelWrite/1KB/C100 | **seaweedfs** | 0.1 MB/s | liteio_mem | 0.1 MB/s | +21% |
| ParallelWrite/1KB/C200 | **seaweedfs** | 0.1 MB/s | liteio | 0.1 MB/s | +30% |
| ParallelWrite/1KB/C25 | **liteio** | 0.2 MB/s | liteio_mem | 0.2 MB/s | ~equal |
| ParallelWrite/1KB/C50 | **seaweedfs** | 0.1 MB/s | minio | 0.1 MB/s | +13% |
| RangeRead/End_256KB | **liteio_mem** | 195.1 MB/s | minio | 184.0 MB/s | ~equal |
| RangeRead/Middle_256KB | **liteio_mem** | 200.4 MB/s | seaweedfs | 167.1 MB/s | +20% |
| RangeRead/Start_256KB | **liteio** | 197.9 MB/s | liteio_mem | 190.8 MB/s | ~equal |
| Read/100MB | **minio** | 324.3 MB/s | rustfs | 297.5 MB/s | ~equal |
| Read/10MB | **minio** | 298.3 MB/s | localstack | 295.8 MB/s | ~equal |
| Read/1KB | **liteio** | 4.8 MB/s | liteio_mem | 4.5 MB/s | ~equal |
| Read/1MB | **liteio_mem** | 271.7 MB/s | liteio | 242.2 MB/s | +12% |
| Read/64KB | **liteio** | 119.2 MB/s | liteio_mem | 108.2 MB/s | +10% |
| Stat | **minio** | 4.4K ops/s | liteio | 4.3K ops/s | ~equal |
| Write/100MB | **liteio** | 193.2 MB/s | seaweedfs | 190.9 MB/s | ~equal |
| Write/10MB | **liteio** | 189.2 MB/s | minio | 183.4 MB/s | ~equal |
| Write/1KB | **liteio_mem** | 1.5 MB/s | rustfs | 1.5 MB/s | ~equal |
| Write/1MB | **rustfs** | 166.5 MB/s | liteio | 154.3 MB/s | ~equal |
| Write/64KB | **rustfs** | 60.2 MB/s | liteio | 56.4 MB/s | ~equal |

## Category Summaries

### Write Operations

**Best for Write:** rustfs (won 2/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | liteio | 193.2 MB/s | ~equal |
| Write/10MB | liteio | 189.2 MB/s | ~equal |
| Write/1KB | liteio_mem | 1.5 MB/s | ~equal |
| Write/1MB | rustfs | 166.5 MB/s | ~equal |
| Write/64KB | rustfs | 60.2 MB/s | ~equal |

### Read Operations

**Best for Read:** minio (won 2/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 324.3 MB/s | ~equal |
| Read/10MB | minio | 298.3 MB/s | ~equal |
| Read/1KB | liteio | 4.8 MB/s | ~equal |
| Read/1MB | liteio_mem | 271.7 MB/s | +12% |
| Read/64KB | liteio | 119.2 MB/s | +10% |

### ParallelWrite Operations

**Best for ParallelWrite:** seaweedfs (won 3/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio_mem | 1.5 MB/s | ~equal |
| ParallelWrite/1KB/C10 | rustfs | 0.5 MB/s | +22% |
| ParallelWrite/1KB/C100 | seaweedfs | 0.1 MB/s | +21% |
| ParallelWrite/1KB/C200 | seaweedfs | 0.1 MB/s | +30% |
| ParallelWrite/1KB/C25 | liteio | 0.2 MB/s | ~equal |
| ParallelWrite/1KB/C50 | seaweedfs | 0.1 MB/s | +13% |

### ParallelRead Operations

**Best for ParallelRead:** liteio_mem (won 3/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 2.8 MB/s | ~equal |
| ParallelRead/1KB/C10 | minio | 1.1 MB/s | +26% |
| ParallelRead/1KB/C100 | liteio_mem | 0.3 MB/s | +19% |
| ParallelRead/1KB/C200 | liteio_mem | 0.6 MB/s | +89% |
| ParallelRead/1KB/C25 | minio | 0.6 MB/s | ~equal |
| ParallelRead/1KB/C50 | liteio_mem | 0.5 MB/s | ~equal |

### Delete Operations

**Best for Delete:** liteio_mem (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio_mem | 5.7K ops/s | +41% |

### Stat Operations

**Best for Stat:** minio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | minio | 4.4K ops/s | ~equal |

### List Operations

**Best for List:** liteio_mem (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio_mem | 1.2K ops/s | ~equal |

### Copy Operations

**Best for Copy:** liteio_mem (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio_mem | 1.5 MB/s | +11% |

### FileCount Operations

**Best for FileCount:** liteio_mem (won 10/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | liteio | 4.6K ops/s | +31% |
| FileCount/Delete/10 | liteio_mem | 502 ops/s | ~equal |
| FileCount/Delete/100 | liteio_mem | 45 ops/s | +50% |
| FileCount/Delete/1000 | liteio_mem | 5 ops/s | ~equal |
| FileCount/Delete/10000 | liteio_mem | 0 ops/s | ~equal |
| FileCount/List/1 | liteio | 3.9K ops/s | +12% |
| FileCount/List/10 | liteio_mem | 3.2K ops/s | +15% |
| FileCount/List/100 | liteio_mem | 800 ops/s | ~equal |
| FileCount/List/1000 | liteio_mem | 161 ops/s | ~equal |
| FileCount/List/10000 | seaweedfs | 11 ops/s | +73% |
| FileCount/Write/1 | liteio_mem | 1.5 MB/s | ~equal |
| FileCount/Write/10 | liteio_mem | 1.6 MB/s | +15% |
| FileCount/Write/100 | rustfs | 1.5 MB/s | ~equal |
| FileCount/Write/1000 | rustfs | 1.5 MB/s | +12% |
| FileCount/Write/10000 | liteio_mem | 1.7 MB/s | ~equal |

---

*Generated by storage benchmark CLI*
