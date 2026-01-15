# Storage Benchmark Summary

**Generated:** 2026-01-15T11:10:27+07:00

## Overall Winner

**rustfs** won 16/51 categories (31%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| rustfs | 16 | 31% |
| liteio_mem | 13 | 25% |
| liteio | 13 | 25% |
| seaweedfs | 5 | 10% |
| minio | 3 | 6% |
| localstack | 1 | 2% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **localstack** | 1.2 MB/s | liteio_mem | 1.2 MB/s | ~equal |
| Delete | **liteio_mem** | 6.4K ops/s | liteio | 5.9K ops/s | ~equal |
| EdgeCase/DeepNested | **rustfs** | 0.1 MB/s | seaweedfs | 0.1 MB/s | ~equal |
| EdgeCase/EmptyObject | **rustfs** | 1.5K ops/s | minio | 1.2K ops/s | +28% |
| EdgeCase/LongKey256 | **rustfs** | 0.1 MB/s | liteio | 0.1 MB/s | +18% |
| FileCount/Delete/1 | **liteio** | 4.5K ops/s | liteio_mem | 3.3K ops/s | +35% |
| FileCount/Delete/10 | **liteio** | 496 ops/s | liteio_mem | 342 ops/s | +45% |
| FileCount/Delete/100 | **liteio_mem** | 47 ops/s | liteio | 45 ops/s | ~equal |
| FileCount/Delete/1000 | **liteio_mem** | 5 ops/s | liteio | 5 ops/s | ~equal |
| FileCount/Delete/10000 | **liteio_mem** | 1 ops/s | liteio | 1 ops/s | ~equal |
| FileCount/List/1 | **liteio** | 3.7K ops/s | liteio_mem | 3.0K ops/s | +23% |
| FileCount/List/10 | **liteio** | 2.9K ops/s | liteio_mem | 2.7K ops/s | ~equal |
| FileCount/List/100 | **liteio** | 1.0K ops/s | seaweedfs | 639 ops/s | +62% |
| FileCount/List/1000 | **liteio** | 175 ops/s | liteio_mem | 152 ops/s | +15% |
| FileCount/List/10000 | **seaweedfs** | 11 ops/s | minio | 6 ops/s | +72% |
| FileCount/Write/1 | **rustfs** | 1.3 MB/s | liteio | 1.3 MB/s | ~equal |
| FileCount/Write/10 | **rustfs** | 1.8 MB/s | seaweedfs | 1.4 MB/s | +33% |
| FileCount/Write/100 | **rustfs** | 1.6 MB/s | seaweedfs | 1.2 MB/s | +31% |
| FileCount/Write/1000 | **rustfs** | 1.8 MB/s | seaweedfs | 1.4 MB/s | +29% |
| FileCount/Write/10000 | **rustfs** | 1.5 MB/s | liteio | 1.5 MB/s | ~equal |
| List/100 | **liteio_mem** | 1.3K ops/s | liteio | 1.2K ops/s | ~equal |
| MixedWorkload/Balanced_50_50 | **seaweedfs** | 1.4 MB/s | liteio | 1.3 MB/s | ~equal |
| MixedWorkload/ReadHeavy_90_10 | **seaweedfs** | 2.4 MB/s | liteio_mem | 1.6 MB/s | +53% |
| MixedWorkload/WriteHeavy_10_90 | **seaweedfs** | 1.1 MB/s | minio | 0.9 MB/s | +25% |
| Multipart/15MB_3Parts | **rustfs** | 176.5 MB/s | minio | 169.5 MB/s | ~equal |
| ParallelRead/1KB/C1 | **liteio** | 3.6 MB/s | liteio_mem | 3.5 MB/s | ~equal |
| ParallelRead/1KB/C10 | **minio** | 1.0 MB/s | rustfs | 1.0 MB/s | ~equal |
| ParallelRead/1KB/C100 | **liteio_mem** | 0.3 MB/s | liteio | 0.3 MB/s | ~equal |
| ParallelRead/1KB/C200 | **liteio** | 0.3 MB/s | liteio_mem | 0.3 MB/s | ~equal |
| ParallelRead/1KB/C25 | **liteio_mem** | 0.8 MB/s | liteio | 0.8 MB/s | ~equal |
| ParallelRead/1KB/C50 | **liteio** | 0.5 MB/s | liteio_mem | 0.4 MB/s | ~equal |
| ParallelWrite/1KB/C1 | **seaweedfs** | 1.5 MB/s | rustfs | 1.4 MB/s | ~equal |
| ParallelWrite/1KB/C10 | **rustfs** | 0.5 MB/s | seaweedfs | 0.4 MB/s | +18% |
| ParallelWrite/1KB/C100 | **rustfs** | 0.1 MB/s | liteio_mem | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C200 | **rustfs** | 0.1 MB/s | seaweedfs | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C25 | **rustfs** | 0.2 MB/s | seaweedfs | 0.2 MB/s | ~equal |
| ParallelWrite/1KB/C50 | **minio** | 0.1 MB/s | liteio_mem | 0.1 MB/s | ~equal |
| RangeRead/End_256KB | **liteio** | 221.7 MB/s | liteio_mem | 207.6 MB/s | ~equal |
| RangeRead/Middle_256KB | **liteio_mem** | 224.1 MB/s | liteio | 211.0 MB/s | ~equal |
| RangeRead/Start_256KB | **liteio_mem** | 233.3 MB/s | liteio | 227.3 MB/s | ~equal |
| Read/100MB | **liteio** | 297.4 MB/s | minio | 297.2 MB/s | ~equal |
| Read/10MB | **minio** | 308.4 MB/s | liteio_mem | 299.8 MB/s | ~equal |
| Read/1KB | **liteio_mem** | 4.5 MB/s | liteio | 4.0 MB/s | +11% |
| Read/1MB | **liteio** | 295.3 MB/s | liteio_mem | 294.4 MB/s | ~equal |
| Read/64KB | **liteio** | 147.0 MB/s | minio | 110.3 MB/s | +33% |
| Stat | **liteio_mem** | 5.4K ops/s | liteio | 5.1K ops/s | ~equal |
| Write/100MB | **rustfs** | 199.0 MB/s | seaweedfs | 194.5 MB/s | ~equal |
| Write/10MB | **rustfs** | 191.3 MB/s | minio | 175.3 MB/s | ~equal |
| Write/1KB | **liteio_mem** | 1.8 MB/s | rustfs | 1.5 MB/s | +26% |
| Write/1MB | **rustfs** | 175.5 MB/s | liteio | 165.4 MB/s | ~equal |
| Write/64KB | **liteio_mem** | 71.9 MB/s | rustfs | 60.0 MB/s | +20% |

## Category Summaries

### Write Operations

**Best for Write:** rustfs (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | rustfs | 199.0 MB/s | ~equal |
| Write/10MB | rustfs | 191.3 MB/s | ~equal |
| Write/1KB | liteio_mem | 1.8 MB/s | +26% |
| Write/1MB | rustfs | 175.5 MB/s | ~equal |
| Write/64KB | liteio_mem | 71.9 MB/s | +20% |

### Read Operations

**Best for Read:** liteio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | liteio | 297.4 MB/s | ~equal |
| Read/10MB | minio | 308.4 MB/s | ~equal |
| Read/1KB | liteio_mem | 4.5 MB/s | +11% |
| Read/1MB | liteio | 295.3 MB/s | ~equal |
| Read/64KB | liteio | 147.0 MB/s | +33% |

### ParallelWrite Operations

**Best for ParallelWrite:** rustfs (won 4/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | seaweedfs | 1.5 MB/s | ~equal |
| ParallelWrite/1KB/C10 | rustfs | 0.5 MB/s | +18% |
| ParallelWrite/1KB/C100 | rustfs | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C200 | rustfs | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C25 | rustfs | 0.2 MB/s | ~equal |
| ParallelWrite/1KB/C50 | minio | 0.1 MB/s | ~equal |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 3/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 3.6 MB/s | ~equal |
| ParallelRead/1KB/C10 | minio | 1.0 MB/s | ~equal |
| ParallelRead/1KB/C100 | liteio_mem | 0.3 MB/s | ~equal |
| ParallelRead/1KB/C200 | liteio | 0.3 MB/s | ~equal |
| ParallelRead/1KB/C25 | liteio_mem | 0.8 MB/s | ~equal |
| ParallelRead/1KB/C50 | liteio | 0.5 MB/s | ~equal |

### Delete Operations

**Best for Delete:** liteio_mem (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio_mem | 6.4K ops/s | ~equal |

### Stat Operations

**Best for Stat:** liteio_mem (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio_mem | 5.4K ops/s | ~equal |

### List Operations

**Best for List:** liteio_mem (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio_mem | 1.3K ops/s | ~equal |

### Copy Operations

**Best for Copy:** localstack (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | localstack | 1.2 MB/s | ~equal |

### FileCount Operations

**Best for FileCount:** liteio (won 6/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | liteio | 4.5K ops/s | +35% |
| FileCount/Delete/10 | liteio | 496 ops/s | +45% |
| FileCount/Delete/100 | liteio_mem | 47 ops/s | ~equal |
| FileCount/Delete/1000 | liteio_mem | 5 ops/s | ~equal |
| FileCount/Delete/10000 | liteio_mem | 1 ops/s | ~equal |
| FileCount/List/1 | liteio | 3.7K ops/s | +23% |
| FileCount/List/10 | liteio | 2.9K ops/s | ~equal |
| FileCount/List/100 | liteio | 1.0K ops/s | +62% |
| FileCount/List/1000 | liteio | 175 ops/s | +15% |
| FileCount/List/10000 | seaweedfs | 11 ops/s | +72% |
| FileCount/Write/1 | rustfs | 1.3 MB/s | ~equal |
| FileCount/Write/10 | rustfs | 1.8 MB/s | +33% |
| FileCount/Write/100 | rustfs | 1.6 MB/s | +31% |
| FileCount/Write/1000 | rustfs | 1.8 MB/s | +29% |
| FileCount/Write/10000 | rustfs | 1.5 MB/s | ~equal |

---

*Generated by storage benchmark CLI*
