# Storage Benchmark Summary

**Generated:** 2026-02-19T09:16:39+07:00

## Overall Winner

**horse** won 39/40 categories (98%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| horse | 39 | 98% |
| minio | 1 | 2% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **horse** | 1.1 GB/s | minio | 1.9 MB/s | 567.8x faster |
| Delete | **horse** | 2.6M ops/s | minio | 2.3K ops/s | 1136.9x faster |
| EdgeCase/DeepNested | **horse** | 147.7 MB/s | rustfs | 0.2 MB/s | 748.3x faster |
| EdgeCase/EmptyObject | **horse** | 1.7M ops/s | minio | 2.2K ops/s | 743.7x faster |
| EdgeCase/LongKey256 | **horse** | 105.6 MB/s | rustfs | 0.2 MB/s | 560.4x faster |
| List/100 | **horse** | 138.9K ops/s | minio | 431 ops/s | 322.0x faster |
| MixedWorkload/Balanced_50_50 | **horse** | 12.4 MB/s | minio | 0.8 MB/s | 15.5x faster |
| MixedWorkload/ReadHeavy_90_10 | **horse** | 98.1 MB/s | minio | 1.7 MB/s | 59.1x faster |
| MixedWorkload/WriteHeavy_10_90 | **horse** | 6.2 MB/s | minio | 0.5 MB/s | 12.2x faster |
| Multipart/15MB_3Parts | **minio** | 346.2 MB/s | rustfs | 312.5 MB/s | +11% |
| ParallelRead/1KB/C1 | **horse** | 6.5 GB/s | minio | 4.5 MB/s | 1459.9x faster |
| ParallelRead/1KB/C10 | **horse** | 5.4 GB/s | minio | 1.5 MB/s | 3499.8x faster |
| ParallelRead/1KB/C50 | **horse** | 4.7 GB/s | minio | 0.5 MB/s | 9583.1x faster |
| ParallelWrite/1KB/C1 | **horse** | 1.0 GB/s | minio | 2.0 MB/s | 515.1x faster |
| ParallelWrite/1KB/C10 | **horse** | 176.3 MB/s | minio | 0.5 MB/s | 330.9x faster |
| ParallelWrite/1KB/C50 | **horse** | 41.5 MB/s | minio | 0.1 MB/s | 363.9x faster |
| RangeRead/End_256KB | **horse** | 2424.0 GB/s | minio | 546.0 MB/s | 4439.9x faster |
| RangeRead/Middle_256KB | **horse** | 2530.5 GB/s | minio | 547.2 MB/s | 4624.0x faster |
| RangeRead/Start_256KB | **horse** | 2369.5 GB/s | minio | 620.2 MB/s | 3820.6x faster |
| Read/10MB | **horse** | 96848.4 GB/s | minio | 3.2 GB/s | 30076.6x faster |
| Read/1KB | **horse** | 8.7 GB/s | minio | 5.4 MB/s | 1617.1x faster |
| Read/1MB | **horse** | 9454.7 GB/s | minio | 1.6 GB/s | 5939.2x faster |
| Read/64KB | **horse** | 622.0 GB/s | minio | 324.5 MB/s | 1916.7x faster |
| Scale/Delete/1 | **horse** | 666.7K ops/s | rustfs | 1.6K ops/s | 415.2x faster |
| Scale/Delete/10 | **horse** | 247.4K ops/s | minio | 225 ops/s | 1100.1x faster |
| Scale/Delete/100 | **horse** | 40.3K ops/s | minio | 23 ops/s | 1719.3x faster |
| Scale/Delete/1000 | **horse** | 3.7K ops/s | minio | 2 ops/s | 1680.7x faster |
| Scale/List/1 | **horse** | 269.7K ops/s | minio | 2.1K ops/s | 129.2x faster |
| Scale/List/10 | **horse** | 263.7K ops/s | minio | 1.3K ops/s | 206.9x faster |
| Scale/List/100 | **horse** | 63.5K ops/s | minio | 298 ops/s | 213.0x faster |
| Scale/List/1000 | **horse** | 5.8K ops/s | minio | 39 ops/s | 150.2x faster |
| Scale/Write/1 | **horse** | 28.0 MB/s | minio | 0.5 MB/s | 58.6x faster |
| Scale/Write/10 | **horse** | 230.7 MB/s | rustfs | 0.5 MB/s | 465.4x faster |
| Scale/Write/100 | **horse** | 324.4 MB/s | minio | 0.5 MB/s | 619.0x faster |
| Scale/Write/1000 | **horse** | 413.4 MB/s | minio | 0.5 MB/s | 784.5x faster |
| Stat | **horse** | 17.0M ops/s | minio | 6.3K ops/s | 2683.8x faster |
| Write/10MB | **horse** | 1.5 GB/s | minio | 408.8 MB/s | 3.7x faster |
| Write/1KB | **horse** | 1.4 GB/s | rustfs | 2.1 MB/s | 674.3x faster |
| Write/1MB | **horse** | 1.3 GB/s | minio | 292.1 MB/s | 4.6x faster |
| Write/64KB | **horse** | 1.4 GB/s | minio | 93.2 MB/s | 15.0x faster |

## Category Summaries

### Write Operations

**Best for Write:** horse (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | horse | 1.5 GB/s | 3.7x faster |
| Write/1KB | horse | 1.4 GB/s | 674.3x faster |
| Write/1MB | horse | 1.3 GB/s | 4.6x faster |
| Write/64KB | horse | 1.4 GB/s | 15.0x faster |

### Read Operations

**Best for Read:** horse (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | horse | 96848.4 GB/s | 30076.6x faster |
| Read/1KB | horse | 8.7 GB/s | 1617.1x faster |
| Read/1MB | horse | 9454.7 GB/s | 5939.2x faster |
| Read/64KB | horse | 622.0 GB/s | 1916.7x faster |

### ParallelWrite Operations

**Best for ParallelWrite:** horse (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | horse | 1.0 GB/s | 515.1x faster |
| ParallelWrite/1KB/C10 | horse | 176.3 MB/s | 330.9x faster |
| ParallelWrite/1KB/C50 | horse | 41.5 MB/s | 363.9x faster |

### ParallelRead Operations

**Best for ParallelRead:** horse (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | horse | 6.5 GB/s | 1459.9x faster |
| ParallelRead/1KB/C10 | horse | 5.4 GB/s | 3499.8x faster |
| ParallelRead/1KB/C50 | horse | 4.7 GB/s | 9583.1x faster |

### Delete Operations

**Best for Delete:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | horse | 2.6M ops/s | 1136.9x faster |

### Stat Operations

**Best for Stat:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | horse | 17.0M ops/s | 2683.8x faster |

### List Operations

**Best for List:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | horse | 138.9K ops/s | 322.0x faster |

### Copy Operations

**Best for Copy:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | horse | 1.1 GB/s | 567.8x faster |

### Scale Operations

**Best for Scale:** horse (won 12/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | horse | 666.7K ops/s | 415.2x faster |
| Scale/Delete/10 | horse | 247.4K ops/s | 1100.1x faster |
| Scale/Delete/100 | horse | 40.3K ops/s | 1719.3x faster |
| Scale/Delete/1000 | horse | 3.7K ops/s | 1680.7x faster |
| Scale/List/1 | horse | 269.7K ops/s | 129.2x faster |
| Scale/List/10 | horse | 263.7K ops/s | 206.9x faster |
| Scale/List/100 | horse | 63.5K ops/s | 213.0x faster |
| Scale/List/1000 | horse | 5.8K ops/s | 150.2x faster |
| Scale/Write/1 | horse | 28.0 MB/s | 58.6x faster |
| Scale/Write/10 | horse | 230.7 MB/s | 465.4x faster |
| Scale/Write/100 | horse | 324.4 MB/s | 619.0x faster |
| Scale/Write/1000 | horse | 413.4 MB/s | 784.5x faster |

---

*Generated by storage benchmark CLI*
