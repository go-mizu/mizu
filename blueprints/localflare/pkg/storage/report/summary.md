# Storage Benchmark Summary

**Generated:** 2026-01-15T11:42:25+07:00

## Overall Winner

**liteio_mem** won 22/51 categories (43%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| liteio_mem | 22 | 43% |
| seaweedfs | 12 | 24% |
| liteio | 9 | 18% |
| rustfs | 5 | 10% |
| minio | 3 | 6% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **liteio_mem** | 1.6 MB/s | liteio | 1.5 MB/s | ~equal |
| Delete | **liteio** | 6.3K ops/s | liteio_mem | 5.8K ops/s | ~equal |
| EdgeCase/DeepNested | **liteio_mem** | 0.2 MB/s | rustfs | 0.2 MB/s | ~equal |
| EdgeCase/EmptyObject | **seaweedfs** | 2.1K ops/s | liteio_mem | 1.7K ops/s | +26% |
| EdgeCase/LongKey256 | **liteio_mem** | 0.2 MB/s | seaweedfs | 0.1 MB/s | +29% |
| FileCount/Delete/1 | **liteio** | 2.3K ops/s | seaweedfs | 2.2K ops/s | ~equal |
| FileCount/Delete/10 | **liteio_mem** | 490 ops/s | liteio | 391 ops/s | +25% |
| FileCount/Delete/100 | **liteio_mem** | 52 ops/s | liteio | 49 ops/s | ~equal |
| FileCount/Delete/1000 | **liteio** | 5 ops/s | liteio_mem | 5 ops/s | ~equal |
| FileCount/Delete/10000 | **liteio_mem** | 1 ops/s | liteio | 1 ops/s | ~equal |
| FileCount/List/1 | **liteio_mem** | 4.0K ops/s | liteio | 1.9K ops/s | 2.2x faster |
| FileCount/List/10 | **liteio_mem** | 3.1K ops/s | liteio | 1.9K ops/s | +61% |
| FileCount/List/100 | **liteio_mem** | 1.2K ops/s | liteio | 1.1K ops/s | +14% |
| FileCount/List/1000 | **liteio** | 167 ops/s | liteio_mem | 117 ops/s | +44% |
| FileCount/List/10000 | **seaweedfs** | 8 ops/s | minio | 5 ops/s | +47% |
| FileCount/Write/1 | **liteio_mem** | 1.6 MB/s | rustfs | 1.4 MB/s | +11% |
| FileCount/Write/10 | **liteio_mem** | 1.7 MB/s | rustfs | 1.4 MB/s | +18% |
| FileCount/Write/100 | **liteio_mem** | 1.5 MB/s | rustfs | 1.4 MB/s | ~equal |
| FileCount/Write/1000 | **liteio_mem** | 1.5 MB/s | rustfs | 1.4 MB/s | ~equal |
| FileCount/Write/10000 | **liteio_mem** | 1.8 MB/s | rustfs | 1.4 MB/s | +32% |
| List/100 | **liteio** | 1.3K ops/s | liteio_mem | 1.2K ops/s | ~equal |
| MixedWorkload/Balanced_50_50 | **seaweedfs** | 2.3 MB/s | rustfs | 1.6 MB/s | +44% |
| MixedWorkload/ReadHeavy_90_10 | **seaweedfs** | 3.7 MB/s | rustfs | 2.0 MB/s | +84% |
| MixedWorkload/WriteHeavy_10_90 | **seaweedfs** | 1.9 MB/s | rustfs | 1.5 MB/s | +28% |
| Multipart/15MB_3Parts | **rustfs** | 173.8 MB/s | localstack | 127.6 MB/s | +36% |
| ParallelRead/1KB/C1 | **liteio** | 4.0 MB/s | liteio_mem | 3.8 MB/s | ~equal |
| ParallelRead/1KB/C10 | **minio** | 1.2 MB/s | liteio | 1.1 MB/s | +14% |
| ParallelRead/1KB/C100 | **seaweedfs** | 0.5 MB/s | rustfs | 0.4 MB/s | +19% |
| ParallelRead/1KB/C200 | **seaweedfs** | 0.3 MB/s | rustfs | 0.3 MB/s | ~equal |
| ParallelRead/1KB/C25 | **liteio** | 0.8 MB/s | liteio_mem | 0.8 MB/s | ~equal |
| ParallelRead/1KB/C50 | **liteio_mem** | 0.5 MB/s | seaweedfs | 0.5 MB/s | ~equal |
| ParallelWrite/1KB/C1 | **liteio_mem** | 1.5 MB/s | liteio | 1.4 MB/s | ~equal |
| ParallelWrite/1KB/C10 | **seaweedfs** | 0.4 MB/s | liteio_mem | 0.4 MB/s | ~equal |
| ParallelWrite/1KB/C100 | **rustfs** | 0.2 MB/s | seaweedfs | 0.1 MB/s | +53% |
| ParallelWrite/1KB/C200 | **seaweedfs** | 0.1 MB/s | rustfs | 0.1 MB/s | +20% |
| ParallelWrite/1KB/C25 | **seaweedfs** | 0.3 MB/s | liteio | 0.2 MB/s | +12% |
| ParallelWrite/1KB/C50 | **seaweedfs** | 0.2 MB/s | liteio_mem | 0.1 MB/s | +65% |
| RangeRead/End_256KB | **liteio_mem** | 236.6 MB/s | liteio | 233.7 MB/s | ~equal |
| RangeRead/Middle_256KB | **liteio_mem** | 221.3 MB/s | seaweedfs | 192.6 MB/s | +15% |
| RangeRead/Start_256KB | **liteio_mem** | 235.3 MB/s | liteio | 207.6 MB/s | +13% |
| Read/100MB | **minio** | 330.7 MB/s | localstack | 316.3 MB/s | ~equal |
| Read/10MB | **minio** | 316.9 MB/s | localstack | 313.4 MB/s | ~equal |
| Read/1KB | **liteio** | 5.0 MB/s | liteio_mem | 4.3 MB/s | +18% |
| Read/1MB | **liteio_mem** | 273.2 MB/s | seaweedfs | 259.4 MB/s | ~equal |
| Read/64KB | **liteio** | 150.4 MB/s | liteio_mem | 131.9 MB/s | +14% |
| Stat | **liteio_mem** | 4.2K ops/s | liteio | 4.1K ops/s | ~equal |
| Write/100MB | **seaweedfs** | 194.6 MB/s | rustfs | 182.3 MB/s | ~equal |
| Write/10MB | **rustfs** | 194.9 MB/s | minio | 152.5 MB/s | +28% |
| Write/1KB | **rustfs** | 1.5 MB/s | seaweedfs | 1.4 MB/s | ~equal |
| Write/1MB | **rustfs** | 152.8 MB/s | liteio_mem | 139.4 MB/s | ~equal |
| Write/64KB | **liteio_mem** | 71.8 MB/s | rustfs | 60.5 MB/s | +19% |

## Category Summaries

### Write Operations

**Best for Write:** rustfs (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | seaweedfs | 194.6 MB/s | ~equal |
| Write/10MB | rustfs | 194.9 MB/s | +28% |
| Write/1KB | rustfs | 1.5 MB/s | ~equal |
| Write/1MB | rustfs | 152.8 MB/s | ~equal |
| Write/64KB | liteio_mem | 71.8 MB/s | +19% |

### Read Operations

**Best for Read:** minio (won 2/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 330.7 MB/s | ~equal |
| Read/10MB | minio | 316.9 MB/s | ~equal |
| Read/1KB | liteio | 5.0 MB/s | +18% |
| Read/1MB | liteio_mem | 273.2 MB/s | ~equal |
| Read/64KB | liteio | 150.4 MB/s | +14% |

### ParallelWrite Operations

**Best for ParallelWrite:** seaweedfs (won 4/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio_mem | 1.5 MB/s | ~equal |
| ParallelWrite/1KB/C10 | seaweedfs | 0.4 MB/s | ~equal |
| ParallelWrite/1KB/C100 | rustfs | 0.2 MB/s | +53% |
| ParallelWrite/1KB/C200 | seaweedfs | 0.1 MB/s | +20% |
| ParallelWrite/1KB/C25 | seaweedfs | 0.3 MB/s | +12% |
| ParallelWrite/1KB/C50 | seaweedfs | 0.2 MB/s | +65% |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 2/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 4.0 MB/s | ~equal |
| ParallelRead/1KB/C10 | minio | 1.2 MB/s | +14% |
| ParallelRead/1KB/C100 | seaweedfs | 0.5 MB/s | +19% |
| ParallelRead/1KB/C200 | seaweedfs | 0.3 MB/s | ~equal |
| ParallelRead/1KB/C25 | liteio | 0.8 MB/s | ~equal |
| ParallelRead/1KB/C50 | liteio_mem | 0.5 MB/s | ~equal |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 6.3K ops/s | ~equal |

### Stat Operations

**Best for Stat:** liteio_mem (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio_mem | 4.2K ops/s | ~equal |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 1.3K ops/s | ~equal |

### Copy Operations

**Best for Copy:** liteio_mem (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio_mem | 1.6 MB/s | ~equal |

### FileCount Operations

**Best for FileCount:** liteio_mem (won 11/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | liteio | 2.3K ops/s | ~equal |
| FileCount/Delete/10 | liteio_mem | 490 ops/s | +25% |
| FileCount/Delete/100 | liteio_mem | 52 ops/s | ~equal |
| FileCount/Delete/1000 | liteio | 5 ops/s | ~equal |
| FileCount/Delete/10000 | liteio_mem | 1 ops/s | ~equal |
| FileCount/List/1 | liteio_mem | 4.0K ops/s | 2.2x faster |
| FileCount/List/10 | liteio_mem | 3.1K ops/s | +61% |
| FileCount/List/100 | liteio_mem | 1.2K ops/s | +14% |
| FileCount/List/1000 | liteio | 167 ops/s | +44% |
| FileCount/List/10000 | seaweedfs | 8 ops/s | +47% |
| FileCount/Write/1 | liteio_mem | 1.6 MB/s | +11% |
| FileCount/Write/10 | liteio_mem | 1.7 MB/s | +18% |
| FileCount/Write/100 | liteio_mem | 1.5 MB/s | ~equal |
| FileCount/Write/1000 | liteio_mem | 1.5 MB/s | ~equal |
| FileCount/Write/10000 | liteio_mem | 1.8 MB/s | +32% |

---

*Generated by storage benchmark CLI*
