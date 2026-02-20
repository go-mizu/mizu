# Storage Benchmark Summary

**Generated:** 2026-02-20T14:15:20+07:00

## Overall Winner

**herd_s3** won 30/48 categories (62%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| herd_s3 | 30 | 62% |
| liteio | 14 | 29% |
| minio | 4 | 8% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **herd_s3** | 5.3 MB/s | liteio | 1.4 MB/s | 3.7x faster |
| Delete | **herd_s3** | 6.1K ops/s | liteio | 4.2K ops/s | +44% |
| EdgeCase/DeepNested | **herd_s3** | 0.4 MB/s | liteio | 0.1 MB/s | 2.7x faster |
| EdgeCase/EmptyObject | **herd_s3** | 4.6K ops/s | seaweedfs | 3.2K ops/s | +43% |
| EdgeCase/LongKey256 | **herd_s3** | 0.4 MB/s | liteio | 0.2 MB/s | 2.6x faster |
| List/100 | **herd_s3** | 1.7K ops/s | liteio | 1.4K ops/s | +20% |
| MixedWorkload/Balanced_50_50 | **liteio** | 0.6 MB/s | herd_s3 | 0.6 MB/s | ~equal |
| MixedWorkload/ReadHeavy_90_10 | **liteio** | 0.8 MB/s | herd_s3 | 0.7 MB/s | +12% |
| MixedWorkload/WriteHeavy_10_90 | **herd_s3** | 0.6 MB/s | liteio | 0.5 MB/s | ~equal |
| Multipart/15MB_3Parts | **liteio** | 150.5 MB/s | minio | 144.6 MB/s | ~equal |
| ParallelRead/1KB/C1 | **herd_s3** | 4.8 MB/s | liteio | 4.0 MB/s | +19% |
| ParallelRead/1KB/C10 | **herd_s3** | 1.4 MB/s | liteio | 1.3 MB/s | ~equal |
| ParallelRead/1KB/C100 | **herd_s3** | 0.2 MB/s | liteio | 0.2 MB/s | +16% |
| ParallelRead/1KB/C200 | **liteio** | 0.1 MB/s | herd_s3 | 0.1 MB/s | ~equal |
| ParallelRead/1KB/C25 | **liteio** | 0.8 MB/s | herd_s3 | 0.7 MB/s | ~equal |
| ParallelRead/1KB/C50 | **liteio** | 0.4 MB/s | herd_s3 | 0.4 MB/s | +12% |
| ParallelWrite/1KB/C1 | **herd_s3** | 4.1 MB/s | seaweedfs | 1.4 MB/s | 3.1x faster |
| ParallelWrite/1KB/C10 | **herd_s3** | 1.1 MB/s | liteio | 0.5 MB/s | 2.2x faster |
| ParallelWrite/1KB/C100 | **herd_s3** | 0.2 MB/s | liteio | 0.1 MB/s | +38% |
| ParallelWrite/1KB/C200 | **herd_s3** | 0.1 MB/s | liteio | 0.1 MB/s | +36% |
| ParallelWrite/1KB/C25 | **herd_s3** | 0.6 MB/s | liteio | 0.3 MB/s | 2.1x faster |
| ParallelWrite/1KB/C50 | **herd_s3** | 0.3 MB/s | liteio | 0.2 MB/s | +55% |
| RangeRead/End_256KB | **minio** | 183.3 MB/s | liteio | 182.0 MB/s | ~equal |
| RangeRead/Middle_256KB | **liteio** | 189.1 MB/s | minio | 174.3 MB/s | ~equal |
| RangeRead/Start_256KB | **liteio** | 170.4 MB/s | herd_s3 | 164.6 MB/s | ~equal |
| Read/100MB | **minio** | 314.6 MB/s | herd_s3 | 244.6 MB/s | +29% |
| Read/10MB | **minio** | 313.9 MB/s | rustfs | 262.9 MB/s | +19% |
| Read/1KB | **liteio** | 5.4 MB/s | herd_s3 | 5.2 MB/s | ~equal |
| Read/1MB | **minio** | 263.0 MB/s | liteio | 230.2 MB/s | +14% |
| Read/64KB | **herd_s3** | 144.9 MB/s | liteio | 128.3 MB/s | +13% |
| Scale/Delete/10 | **liteio** | 504 ops/s | herd_s3 | 481 ops/s | ~equal |
| Scale/Delete/100 | **herd_s3** | 46 ops/s | liteio | 36 ops/s | +29% |
| Scale/Delete/1000 | **liteio** | 6 ops/s | herd_s3 | 5 ops/s | +23% |
| Scale/Delete/10000 | **liteio** | 1 ops/s | herd_s3 | 1 ops/s | ~equal |
| Scale/List/10 | **herd_s3** | 3.1K ops/s | liteio | 2.9K ops/s | ~equal |
| Scale/List/100 | **liteio** | 1.2K ops/s | herd_s3 | 1.2K ops/s | ~equal |
| Scale/List/1000 | **herd_s3** | 202 ops/s | liteio | 181 ops/s | +12% |
| Scale/List/10000 | **herd_s3** | 11 ops/s | seaweedfs | 9 ops/s | +21% |
| Scale/Write/10 | **herd_s3** | 1.1 MB/s | seaweedfs | 0.4 MB/s | 2.8x faster |
| Scale/Write/100 | **herd_s3** | 1.1 MB/s | liteio | 0.4 MB/s | 2.6x faster |
| Scale/Write/1000 | **herd_s3** | 1.1 MB/s | liteio | 0.4 MB/s | 2.7x faster |
| Scale/Write/10000 | **herd_s3** | 1.1 MB/s | seaweedfs | 0.4 MB/s | 2.9x faster |
| Stat | **herd_s3** | 5.8K ops/s | liteio | 4.3K ops/s | +34% |
| Write/100MB | **herd_s3** | 173.4 MB/s | seaweedfs | 169.9 MB/s | ~equal |
| Write/10MB | **liteio** | 181.8 MB/s | herd_s3 | 170.2 MB/s | ~equal |
| Write/1KB | **herd_s3** | 4.8 MB/s | liteio | 1.6 MB/s | 3.1x faster |
| Write/1MB | **herd_s3** | 182.0 MB/s | liteio | 154.4 MB/s | +18% |
| Write/64KB | **herd_s3** | 139.9 MB/s | liteio | 66.3 MB/s | 2.1x faster |

## Category Summaries

### Write Operations

**Best for Write:** herd_s3 (won 4/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | herd_s3 | 173.4 MB/s | ~equal |
| Write/10MB | liteio | 181.8 MB/s | ~equal |
| Write/1KB | herd_s3 | 4.8 MB/s | 3.1x faster |
| Write/1MB | herd_s3 | 182.0 MB/s | +18% |
| Write/64KB | herd_s3 | 139.9 MB/s | 2.1x faster |

### Read Operations

**Best for Read:** minio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 314.6 MB/s | +29% |
| Read/10MB | minio | 313.9 MB/s | +19% |
| Read/1KB | liteio | 5.4 MB/s | ~equal |
| Read/1MB | minio | 263.0 MB/s | +14% |
| Read/64KB | herd_s3 | 144.9 MB/s | +13% |

### ParallelWrite Operations

**Best for ParallelWrite:** herd_s3 (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | herd_s3 | 4.1 MB/s | 3.1x faster |
| ParallelWrite/1KB/C10 | herd_s3 | 1.1 MB/s | 2.2x faster |
| ParallelWrite/1KB/C100 | herd_s3 | 0.2 MB/s | +38% |
| ParallelWrite/1KB/C200 | herd_s3 | 0.1 MB/s | +36% |
| ParallelWrite/1KB/C25 | herd_s3 | 0.6 MB/s | 2.1x faster |
| ParallelWrite/1KB/C50 | herd_s3 | 0.3 MB/s | +55% |

### ParallelRead Operations

**Best for ParallelRead:** herd_s3 (won 3/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | herd_s3 | 4.8 MB/s | +19% |
| ParallelRead/1KB/C10 | herd_s3 | 1.4 MB/s | ~equal |
| ParallelRead/1KB/C100 | herd_s3 | 0.2 MB/s | +16% |
| ParallelRead/1KB/C200 | liteio | 0.1 MB/s | ~equal |
| ParallelRead/1KB/C25 | liteio | 0.8 MB/s | ~equal |
| ParallelRead/1KB/C50 | liteio | 0.4 MB/s | +12% |

### Delete Operations

**Best for Delete:** herd_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | herd_s3 | 6.1K ops/s | +44% |

### Stat Operations

**Best for Stat:** herd_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | herd_s3 | 5.8K ops/s | +34% |

### List Operations

**Best for List:** herd_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | herd_s3 | 1.7K ops/s | +20% |

### Copy Operations

**Best for Copy:** herd_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | herd_s3 | 5.3 MB/s | 3.7x faster |

### Scale Operations

**Best for Scale:** herd_s3 (won 8/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | liteio | 504 ops/s | ~equal |
| Scale/Delete/100 | herd_s3 | 46 ops/s | +29% |
| Scale/Delete/1000 | liteio | 6 ops/s | +23% |
| Scale/Delete/10000 | liteio | 1 ops/s | ~equal |
| Scale/List/10 | herd_s3 | 3.1K ops/s | ~equal |
| Scale/List/100 | liteio | 1.2K ops/s | ~equal |
| Scale/List/1000 | herd_s3 | 202 ops/s | +12% |
| Scale/List/10000 | herd_s3 | 11 ops/s | +21% |
| Scale/Write/10 | herd_s3 | 1.1 MB/s | 2.8x faster |
| Scale/Write/100 | herd_s3 | 1.1 MB/s | 2.6x faster |
| Scale/Write/1000 | herd_s3 | 1.1 MB/s | 2.7x faster |
| Scale/Write/10000 | herd_s3 | 1.1 MB/s | 2.9x faster |

---

*Generated by storage benchmark CLI*
