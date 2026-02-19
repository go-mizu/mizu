# Storage Benchmark Summary

**Generated:** 2026-02-20T01:08:45+07:00

## Overall Winner

**herd_cluster** won 39/40 categories (98%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| herd_cluster | 39 | 98% |
| minio_cluster | 1 | 2% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **herd_cluster** | 12.5 MB/s | seaweedfs_cluster | 0.8 MB/s | 16.3x faster |
| Delete | **herd_cluster** | 13.7K ops/s | seaweedfs_cluster | 5.6K ops/s | 2.4x faster |
| EdgeCase/DeepNested | **herd_cluster** | 1.2 MB/s | seaweedfs_cluster | 0.2 MB/s | 8.1x faster |
| EdgeCase/EmptyObject | **herd_cluster** | 13.2K ops/s | seaweedfs_cluster | 2.9K ops/s | 4.6x faster |
| EdgeCase/LongKey256 | **herd_cluster** | 1.1 MB/s | seaweedfs_cluster | 0.1 MB/s | 12.9x faster |
| List/100 | **herd_cluster** | 1.9K ops/s | seaweedfs_cluster | 923 ops/s | 2.0x faster |
| MixedWorkload/Balanced_50_50 | **herd_cluster** | 3.7 MB/s | seaweedfs_cluster | 1.1 MB/s | 3.5x faster |
| MixedWorkload/ReadHeavy_90_10 | **herd_cluster** | 4.0 MB/s | seaweedfs_cluster | 1.4 MB/s | 2.9x faster |
| MixedWorkload/WriteHeavy_10_90 | **herd_cluster** | 2.9 MB/s | seaweedfs_cluster | 0.6 MB/s | 5.1x faster |
| Multipart/15MB_3Parts | **minio_cluster** | 280.0 MB/s | seaweedfs_cluster | 180.7 MB/s | +55% |
| ParallelRead/1KB/C1 | **herd_cluster** | 9.0 MB/s | seaweedfs_cluster | 4.2 MB/s | 2.1x faster |
| ParallelRead/1KB/C10 | **herd_cluster** | 3.2 MB/s | seaweedfs_cluster | 1.3 MB/s | 2.5x faster |
| ParallelRead/1KB/C50 | **herd_cluster** | 0.8 MB/s | seaweedfs_cluster | 0.4 MB/s | 2.2x faster |
| ParallelWrite/1KB/C1 | **herd_cluster** | 8.3 MB/s | rustfs_cluster | 0.6 MB/s | 13.2x faster |
| ParallelWrite/1KB/C10 | **herd_cluster** | 3.1 MB/s | seaweedfs_cluster | 0.8 MB/s | 3.7x faster |
| ParallelWrite/1KB/C50 | **herd_cluster** | 0.8 MB/s | seaweedfs_cluster | 0.2 MB/s | 3.7x faster |
| RangeRead/End_256KB | **herd_cluster** | 2.7 GB/s | seaweedfs_cluster | 1.0 GB/s | 2.7x faster |
| RangeRead/Middle_256KB | **herd_cluster** | 2.7 GB/s | seaweedfs_cluster | 961.5 MB/s | 2.9x faster |
| RangeRead/Start_256KB | **herd_cluster** | 2.7 GB/s | seaweedfs_cluster | 979.6 MB/s | 2.7x faster |
| Read/10MB | **herd_cluster** | 4.8 GB/s | minio_cluster | 2.4 GB/s | 2.0x faster |
| Read/1KB | **herd_cluster** | 10.0 MB/s | minio_cluster | 4.8 MB/s | 2.1x faster |
| Read/1MB | **herd_cluster** | 2.2 GB/s | seaweedfs_cluster | 2.0 GB/s | +11% |
| Read/64KB | **herd_cluster** | 535.6 MB/s | seaweedfs_cluster | 303.9 MB/s | +76% |
| Scale/Delete/1 | **herd_cluster** | 12.6K ops/s | seaweedfs_cluster | 3.3K ops/s | 3.9x faster |
| Scale/Delete/10 | **herd_cluster** | 1.4K ops/s | seaweedfs_cluster | 503 ops/s | 2.8x faster |
| Scale/Delete/100 | **herd_cluster** | 130 ops/s | seaweedfs_cluster | 48 ops/s | 2.7x faster |
| Scale/Delete/1000 | **herd_cluster** | 15 ops/s | seaweedfs_cluster | 2 ops/s | 6.2x faster |
| Scale/List/1 | **herd_cluster** | 7.9K ops/s | seaweedfs_cluster | 1.9K ops/s | 4.2x faster |
| Scale/List/10 | **herd_cluster** | 7.3K ops/s | seaweedfs_cluster | 896 ops/s | 8.1x faster |
| Scale/List/100 | **herd_cluster** | 2.1K ops/s | seaweedfs_cluster | 834 ops/s | 2.5x faster |
| Scale/List/1000 | **herd_cluster** | 272 ops/s | seaweedfs_cluster | 122 ops/s | 2.2x faster |
| Scale/Write/1 | **herd_cluster** | 2.7 MB/s | seaweedfs_cluster | 0.4 MB/s | 6.8x faster |
| Scale/Write/10 | **herd_cluster** | 2.9 MB/s | seaweedfs_cluster | 0.4 MB/s | 7.6x faster |
| Scale/Write/100 | **herd_cluster** | 3.2 MB/s | seaweedfs_cluster | 0.5 MB/s | 7.0x faster |
| Scale/Write/1000 | **herd_cluster** | 3.2 MB/s | seaweedfs_cluster | 0.4 MB/s | 7.6x faster |
| Stat | **herd_cluster** | 10.0K ops/s | seaweedfs_cluster | 8.0K ops/s | +25% |
| Write/10MB | **herd_cluster** | 652.0 MB/s | minio_cluster | 330.9 MB/s | +97% |
| Write/1KB | **herd_cluster** | 7.8 MB/s | seaweedfs_cluster | 2.8 MB/s | 2.8x faster |
| Write/1MB | **herd_cluster** | 618.5 MB/s | seaweedfs_cluster | 209.7 MB/s | 2.9x faster |
| Write/64KB | **herd_cluster** | 290.1 MB/s | seaweedfs_cluster | 103.1 MB/s | 2.8x faster |

## Category Summaries

### Write Operations

**Best for Write:** herd_cluster (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | herd_cluster | 652.0 MB/s | +97% |
| Write/1KB | herd_cluster | 7.8 MB/s | 2.8x faster |
| Write/1MB | herd_cluster | 618.5 MB/s | 2.9x faster |
| Write/64KB | herd_cluster | 290.1 MB/s | 2.8x faster |

### Read Operations

**Best for Read:** herd_cluster (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | herd_cluster | 4.8 GB/s | 2.0x faster |
| Read/1KB | herd_cluster | 10.0 MB/s | 2.1x faster |
| Read/1MB | herd_cluster | 2.2 GB/s | +11% |
| Read/64KB | herd_cluster | 535.6 MB/s | +76% |

### ParallelWrite Operations

**Best for ParallelWrite:** herd_cluster (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | herd_cluster | 8.3 MB/s | 13.2x faster |
| ParallelWrite/1KB/C10 | herd_cluster | 3.1 MB/s | 3.7x faster |
| ParallelWrite/1KB/C50 | herd_cluster | 0.8 MB/s | 3.7x faster |

### ParallelRead Operations

**Best for ParallelRead:** herd_cluster (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | herd_cluster | 9.0 MB/s | 2.1x faster |
| ParallelRead/1KB/C10 | herd_cluster | 3.2 MB/s | 2.5x faster |
| ParallelRead/1KB/C50 | herd_cluster | 0.8 MB/s | 2.2x faster |

### Delete Operations

**Best for Delete:** herd_cluster (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | herd_cluster | 13.7K ops/s | 2.4x faster |

### Stat Operations

**Best for Stat:** herd_cluster (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | herd_cluster | 10.0K ops/s | +25% |

### List Operations

**Best for List:** herd_cluster (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | herd_cluster | 1.9K ops/s | 2.0x faster |

### Copy Operations

**Best for Copy:** herd_cluster (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | herd_cluster | 12.5 MB/s | 16.3x faster |

### Scale Operations

**Best for Scale:** herd_cluster (won 12/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | herd_cluster | 12.6K ops/s | 3.9x faster |
| Scale/Delete/10 | herd_cluster | 1.4K ops/s | 2.8x faster |
| Scale/Delete/100 | herd_cluster | 130 ops/s | 2.7x faster |
| Scale/Delete/1000 | herd_cluster | 15 ops/s | 6.2x faster |
| Scale/List/1 | herd_cluster | 7.9K ops/s | 4.2x faster |
| Scale/List/10 | herd_cluster | 7.3K ops/s | 8.1x faster |
| Scale/List/100 | herd_cluster | 2.1K ops/s | 2.5x faster |
| Scale/List/1000 | herd_cluster | 272 ops/s | 2.2x faster |
| Scale/Write/1 | herd_cluster | 2.7 MB/s | 6.8x faster |
| Scale/Write/10 | herd_cluster | 2.9 MB/s | 7.6x faster |
| Scale/Write/100 | herd_cluster | 3.2 MB/s | 7.0x faster |
| Scale/Write/1000 | herd_cluster | 3.2 MB/s | 7.6x faster |

---

*Generated by storage benchmark CLI*
