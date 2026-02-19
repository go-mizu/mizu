# Storage Benchmark Summary

**Generated:** 0001-01-01T00:00:00Z

## Overall Winner

**herd_cluster** won 35/40 categories (88%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| herd_cluster | 35 | 88% |
| seaweedfs_cluster | 4 | 10% |
| minio_cluster | 1 | 2% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **herd_cluster** | 8.2 MB/s | seaweedfs_cluster | 1.9 MB/s | 4.4x faster |
| Delete | **herd_cluster** | 11.5K ops/s | seaweedfs_cluster | 7.4K ops/s | +55% |
| EdgeCase/DeepNested | **herd_cluster** | 1.0 MB/s | seaweedfs_cluster | 0.3 MB/s | 3.5x faster |
| EdgeCase/EmptyObject | **herd_cluster** | 10.2K ops/s | seaweedfs_cluster | 6.3K ops/s | +62% |
| EdgeCase/LongKey256 | **herd_cluster** | 0.9 MB/s | seaweedfs_cluster | 0.3 MB/s | 3.6x faster |
| List/100 | **herd_cluster** | 2.0K ops/s | seaweedfs_cluster | 1.1K ops/s | +85% |
| MixedWorkload/Balanced_50_50 | **herd_cluster** | 13.4 MB/s | seaweedfs_cluster | 5.1 MB/s | 2.6x faster |
| MixedWorkload/ReadHeavy_90_10 | **herd_cluster** | 17.1 MB/s | seaweedfs_cluster | 6.9 MB/s | 2.5x faster |
| MixedWorkload/WriteHeavy_10_90 | **herd_cluster** | 11.4 MB/s | seaweedfs_cluster | 4.3 MB/s | 2.7x faster |
| Multipart/15MB_3Parts | **herd_cluster** | 368.9 MB/s | minio_cluster | 287.3 MB/s | +28% |
| ParallelRead/1KB/C1 | **herd_cluster** | 9.4 MB/s | seaweedfs_cluster | 4.6 MB/s | 2.0x faster |
| ParallelRead/1KB/C10 | **herd_cluster** | 4.0 MB/s | seaweedfs_cluster | 1.8 MB/s | 2.3x faster |
| ParallelRead/1KB/C50 | **herd_cluster** | 1.2 MB/s | seaweedfs_cluster | 0.5 MB/s | 2.3x faster |
| ParallelWrite/1KB/C1 | **herd_cluster** | 7.3 MB/s | seaweedfs_cluster | 2.6 MB/s | 2.8x faster |
| ParallelWrite/1KB/C10 | **herd_cluster** | 2.9 MB/s | seaweedfs_cluster | 1.0 MB/s | 2.9x faster |
| ParallelWrite/1KB/C50 | **herd_cluster** | 0.9 MB/s | seaweedfs_cluster | 0.3 MB/s | 3.1x faster |
| RangeRead/End_256KB | **seaweedfs_cluster** | 1.0 GB/s | herd_cluster | 581.8 MB/s | +76% |
| RangeRead/Middle_256KB | **seaweedfs_cluster** | 970.8 MB/s | herd_cluster | 582.8 MB/s | +67% |
| RangeRead/Start_256KB | **seaweedfs_cluster** | 1.0 GB/s | herd_cluster | 480.3 MB/s | 2.1x faster |
| Read/10MB | **minio_cluster** | 2.8 GB/s | herd_cluster | 2.3 GB/s | +22% |
| Read/1KB | **herd_cluster** | 14.0 MB/s | seaweedfs_cluster | 5.3 MB/s | 2.6x faster |
| Read/1MB | **seaweedfs_cluster** | 2.1 GB/s | herd_cluster | 2.0 GB/s | ~equal |
| Read/64KB | **herd_cluster** | 860.1 MB/s | seaweedfs_cluster | 294.6 MB/s | 2.9x faster |
| Scale/Delete/1 | **herd_cluster** | 9.3K ops/s | seaweedfs_cluster | 6.3K ops/s | +46% |
| Scale/Delete/10 | **herd_cluster** | 1.1K ops/s | seaweedfs_cluster | 677 ops/s | +60% |
| Scale/Delete/100 | **herd_cluster** | 117 ops/s | seaweedfs_cluster | 73 ops/s | +61% |
| Scale/Delete/1000 | **herd_cluster** | 11 ops/s | seaweedfs_cluster | 7 ops/s | +55% |
| Scale/List/1 | **herd_cluster** | 4.4K ops/s | seaweedfs_cluster | 2.7K ops/s | +63% |
| Scale/List/10 | **herd_cluster** | 4.4K ops/s | seaweedfs_cluster | 2.6K ops/s | +70% |
| Scale/List/100 | **herd_cluster** | 1.8K ops/s | seaweedfs_cluster | 981 ops/s | +80% |
| Scale/List/1000 | **herd_cluster** | 255 ops/s | seaweedfs_cluster | 161 ops/s | +59% |
| Scale/Write/1 | **herd_cluster** | 2.1 MB/s | seaweedfs_cluster | 0.7 MB/s | 3.2x faster |
| Scale/Write/10 | **herd_cluster** | 2.4 MB/s | seaweedfs_cluster | 0.7 MB/s | 3.2x faster |
| Scale/Write/100 | **herd_cluster** | 2.4 MB/s | seaweedfs_cluster | 0.7 MB/s | 3.2x faster |
| Scale/Write/1000 | **herd_cluster** | 2.5 MB/s | seaweedfs_cluster | 0.7 MB/s | 3.4x faster |
| Stat | **herd_cluster** | 11.4K ops/s | seaweedfs_cluster | 8.0K ops/s | +43% |
| Write/10MB | **herd_cluster** | 868.2 MB/s | minio_cluster | 338.7 MB/s | 2.6x faster |
| Write/1KB | **herd_cluster** | 9.6 MB/s | seaweedfs_cluster | 2.7 MB/s | 3.5x faster |
| Write/1MB | **herd_cluster** | 875.2 MB/s | seaweedfs_cluster | 225.5 MB/s | 3.9x faster |
| Write/64KB | **herd_cluster** | 374.5 MB/s | seaweedfs_cluster | 101.1 MB/s | 3.7x faster |

## Category Summaries

### Write Operations

**Best for Write:** herd_cluster (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | herd_cluster | 868.2 MB/s | 2.6x faster |
| Write/1KB | herd_cluster | 9.6 MB/s | 3.5x faster |
| Write/1MB | herd_cluster | 875.2 MB/s | 3.9x faster |
| Write/64KB | herd_cluster | 374.5 MB/s | 3.7x faster |

### Read Operations

**Best for Read:** herd_cluster (won 2/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | minio_cluster | 2.8 GB/s | +22% |
| Read/1KB | herd_cluster | 14.0 MB/s | 2.6x faster |
| Read/1MB | seaweedfs_cluster | 2.1 GB/s | ~equal |
| Read/64KB | herd_cluster | 860.1 MB/s | 2.9x faster |

### ParallelWrite Operations

**Best for ParallelWrite:** herd_cluster (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | herd_cluster | 7.3 MB/s | 2.8x faster |
| ParallelWrite/1KB/C10 | herd_cluster | 2.9 MB/s | 2.9x faster |
| ParallelWrite/1KB/C50 | herd_cluster | 0.9 MB/s | 3.1x faster |

### ParallelRead Operations

**Best for ParallelRead:** herd_cluster (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | herd_cluster | 9.4 MB/s | 2.0x faster |
| ParallelRead/1KB/C10 | herd_cluster | 4.0 MB/s | 2.3x faster |
| ParallelRead/1KB/C50 | herd_cluster | 1.2 MB/s | 2.3x faster |

### Delete Operations

**Best for Delete:** herd_cluster (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | herd_cluster | 11.5K ops/s | +55% |

### Stat Operations

**Best for Stat:** herd_cluster (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | herd_cluster | 11.4K ops/s | +43% |

### List Operations

**Best for List:** herd_cluster (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | herd_cluster | 2.0K ops/s | +85% |

### Copy Operations

**Best for Copy:** herd_cluster (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | herd_cluster | 8.2 MB/s | 4.4x faster |

### Scale Operations

**Best for Scale:** herd_cluster (won 12/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | herd_cluster | 9.3K ops/s | +46% |
| Scale/Delete/10 | herd_cluster | 1.1K ops/s | +60% |
| Scale/Delete/100 | herd_cluster | 117 ops/s | +61% |
| Scale/Delete/1000 | herd_cluster | 11 ops/s | +55% |
| Scale/List/1 | herd_cluster | 4.4K ops/s | +63% |
| Scale/List/10 | herd_cluster | 4.4K ops/s | +70% |
| Scale/List/100 | herd_cluster | 1.8K ops/s | +80% |
| Scale/List/1000 | herd_cluster | 255 ops/s | +59% |
| Scale/Write/1 | herd_cluster | 2.1 MB/s | 3.2x faster |
| Scale/Write/10 | herd_cluster | 2.4 MB/s | 3.2x faster |
| Scale/Write/100 | herd_cluster | 2.4 MB/s | 3.2x faster |
| Scale/Write/1000 | herd_cluster | 2.5 MB/s | 3.4x faster |

---

*Generated by storage benchmark CLI*
