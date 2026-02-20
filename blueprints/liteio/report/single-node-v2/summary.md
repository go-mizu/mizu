# Storage Benchmark Summary

**Generated:** 2026-02-20T14:42:39+07:00

## Overall Winner

**herd_s3** won 23/48 categories (48%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| herd_s3 | 23 | 48% |
| liteio | 15 | 31% |
| minio | 6 | 12% |
| seaweedfs | 3 | 6% |
| rustfs | 1 | 2% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **herd_s3** | 1.8 MB/s | liteio | 1.2 MB/s | +46% |
| Delete | **liteio** | 4.2K ops/s | herd_s3 | 3.0K ops/s | +39% |
| EdgeCase/DeepNested | **herd_s3** | 0.2 MB/s | seaweedfs | 0.1 MB/s | 2.1x faster |
| EdgeCase/EmptyObject | **herd_s3** | 2.1K ops/s | seaweedfs | 1.6K ops/s | +31% |
| EdgeCase/LongKey256 | **herd_s3** | 0.2 MB/s | seaweedfs | 0.1 MB/s | 2.4x faster |
| List/100 | **herd_s3** | 1.2K ops/s | liteio | 941 ops/s | +31% |
| MixedWorkload/Balanced_50_50 | **liteio** | 0.5 MB/s | herd_s3 | 0.3 MB/s | +53% |
| MixedWorkload/ReadHeavy_90_10 | **liteio** | 0.6 MB/s | minio | 0.5 MB/s | +17% |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 0.4 MB/s | herd_s3 | 0.4 MB/s | ~equal |
| Multipart/15MB_3Parts | **minio** | 113.8 MB/s | liteio | 96.0 MB/s | +18% |
| ParallelRead/1KB/C1 | **liteio** | 3.5 MB/s | minio | 2.7 MB/s | +31% |
| ParallelRead/1KB/C10 | **liteio** | 1.0 MB/s | minio | 0.8 MB/s | +22% |
| ParallelRead/1KB/C100 | **liteio** | 0.2 MB/s | herd_s3 | 0.2 MB/s | ~equal |
| ParallelRead/1KB/C200 | **liteio** | 0.1 MB/s | herd_s3 | 0.1 MB/s | +25% |
| ParallelRead/1KB/C25 | **liteio** | 0.5 MB/s | minio | 0.4 MB/s | +12% |
| ParallelRead/1KB/C50 | **liteio** | 0.3 MB/s | herd_s3 | 0.3 MB/s | +16% |
| ParallelWrite/1KB/C1 | **herd_s3** | 1.5 MB/s | liteio | 1.2 MB/s | +19% |
| ParallelWrite/1KB/C10 | **herd_s3** | 0.7 MB/s | liteio | 0.4 MB/s | +83% |
| ParallelWrite/1KB/C100 | **herd_s3** | 0.1 MB/s | liteio | 0.1 MB/s | +55% |
| ParallelWrite/1KB/C200 | **herd_s3** | 0.1 MB/s | liteio | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C25 | **herd_s3** | 0.3 MB/s | liteio | 0.2 MB/s | +15% |
| ParallelWrite/1KB/C50 | **herd_s3** | 0.1 MB/s | liteio | 0.1 MB/s | ~equal |
| RangeRead/End_256KB | **liteio** | 143.1 MB/s | herd_s3 | 108.3 MB/s | +32% |
| RangeRead/Middle_256KB | **liteio** | 136.9 MB/s | minio | 133.8 MB/s | ~equal |
| RangeRead/Start_256KB | **liteio** | 128.0 MB/s | seaweedfs | 106.7 MB/s | +20% |
| Read/100MB | **rustfs** | 202.8 MB/s | minio | 177.7 MB/s | +14% |
| Read/10MB | **minio** | 234.8 MB/s | rustfs | 163.7 MB/s | +43% |
| Read/1KB | **herd_s3** | 3.5 MB/s | minio | 2.1 MB/s | +63% |
| Read/1MB | **minio** | 218.2 MB/s | herd_s3 | 145.3 MB/s | +50% |
| Read/64KB | **minio** | 89.0 MB/s | herd_s3 | 82.0 MB/s | ~equal |
| Scale/Delete/10 | **minio** | 250 ops/s | herd_s3 | 242 ops/s | ~equal |
| Scale/Delete/100 | **seaweedfs** | 25 ops/s | minio | 24 ops/s | ~equal |
| Scale/Delete/1000 | **herd_s3** | 2 ops/s | seaweedfs | 2 ops/s | ~equal |
| Scale/Delete/10000 | **minio** | 0 ops/s | herd_s3 | 0 ops/s | +21% |
| Scale/List/10 | **herd_s3** | 1.6K ops/s | minio | 1.1K ops/s | +54% |
| Scale/List/100 | **herd_s3** | 768 ops/s | liteio | 669 ops/s | +15% |
| Scale/List/1000 | **herd_s3** | 133 ops/s | liteio | 73 ops/s | +81% |
| Scale/List/10000 | **seaweedfs** | 10 ops/s | herd_s3 | 6 ops/s | +70% |
| Scale/Write/10 | **herd_s3** | 0.6 MB/s | seaweedfs | 0.3 MB/s | 2.1x faster |
| Scale/Write/100 | **herd_s3** | 0.5 MB/s | seaweedfs | 0.3 MB/s | +79% |
| Scale/Write/1000 | **herd_s3** | 0.5 MB/s | seaweedfs | 0.3 MB/s | +62% |
| Scale/Write/10000 | **herd_s3** | 0.3 MB/s | rustfs | 0.3 MB/s | +14% |
| Stat | **herd_s3** | 3.9K ops/s | minio | 3.1K ops/s | +28% |
| Write/100MB | **seaweedfs** | 157.5 MB/s | minio | 152.0 MB/s | ~equal |
| Write/10MB | **liteio** | 125.9 MB/s | seaweedfs | 117.4 MB/s | ~equal |
| Write/1KB | **herd_s3** | 1.7 MB/s | seaweedfs | 1.3 MB/s | +31% |
| Write/1MB | **liteio** | 104.6 MB/s | minio | 97.3 MB/s | ~equal |
| Write/64KB | **herd_s3** | 47.0 MB/s | seaweedfs | 46.0 MB/s | ~equal |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 2/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | seaweedfs | 157.5 MB/s | ~equal |
| Write/10MB | liteio | 125.9 MB/s | ~equal |
| Write/1KB | herd_s3 | 1.7 MB/s | +31% |
| Write/1MB | liteio | 104.6 MB/s | ~equal |
| Write/64KB | herd_s3 | 47.0 MB/s | ~equal |

### Read Operations

**Best for Read:** minio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | rustfs | 202.8 MB/s | +14% |
| Read/10MB | minio | 234.8 MB/s | +43% |
| Read/1KB | herd_s3 | 3.5 MB/s | +63% |
| Read/1MB | minio | 218.2 MB/s | +50% |
| Read/64KB | minio | 89.0 MB/s | ~equal |

### ParallelWrite Operations

**Best for ParallelWrite:** herd_s3 (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | herd_s3 | 1.5 MB/s | +19% |
| ParallelWrite/1KB/C10 | herd_s3 | 0.7 MB/s | +83% |
| ParallelWrite/1KB/C100 | herd_s3 | 0.1 MB/s | +55% |
| ParallelWrite/1KB/C200 | herd_s3 | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C25 | herd_s3 | 0.3 MB/s | +15% |
| ParallelWrite/1KB/C50 | herd_s3 | 0.1 MB/s | ~equal |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 3.5 MB/s | +31% |
| ParallelRead/1KB/C10 | liteio | 1.0 MB/s | +22% |
| ParallelRead/1KB/C100 | liteio | 0.2 MB/s | ~equal |
| ParallelRead/1KB/C200 | liteio | 0.1 MB/s | +25% |
| ParallelRead/1KB/C25 | liteio | 0.5 MB/s | +12% |
| ParallelRead/1KB/C50 | liteio | 0.3 MB/s | +16% |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 4.2K ops/s | +39% |

### Stat Operations

**Best for Stat:** herd_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | herd_s3 | 3.9K ops/s | +28% |

### List Operations

**Best for List:** herd_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | herd_s3 | 1.2K ops/s | +31% |

### Copy Operations

**Best for Copy:** herd_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | herd_s3 | 1.8 MB/s | +46% |

### Scale Operations

**Best for Scale:** herd_s3 (won 8/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | minio | 250 ops/s | ~equal |
| Scale/Delete/100 | seaweedfs | 25 ops/s | ~equal |
| Scale/Delete/1000 | herd_s3 | 2 ops/s | ~equal |
| Scale/Delete/10000 | minio | 0 ops/s | +21% |
| Scale/List/10 | herd_s3 | 1.6K ops/s | +54% |
| Scale/List/100 | herd_s3 | 768 ops/s | +15% |
| Scale/List/1000 | herd_s3 | 133 ops/s | +81% |
| Scale/List/10000 | seaweedfs | 10 ops/s | +70% |
| Scale/Write/10 | herd_s3 | 0.6 MB/s | 2.1x faster |
| Scale/Write/100 | herd_s3 | 0.5 MB/s | +79% |
| Scale/Write/1000 | herd_s3 | 0.5 MB/s | +62% |
| Scale/Write/10000 | herd_s3 | 0.3 MB/s | +14% |

---

*Generated by storage benchmark CLI*
