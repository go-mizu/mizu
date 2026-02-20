# Storage Benchmark Summary

**Generated:** 2026-02-20T14:49:03+07:00

## Overall Winner

**herd_s3** won 27/48 categories (56%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| herd_s3 | 27 | 56% |
| liteio | 21 | 44% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **herd_s3** | 4.5 MB/s | liteio | 1.4 MB/s | 3.2x faster |
| Delete | **herd_s3** | 5.1K ops/s | liteio | 4.9K ops/s | ~equal |
| EdgeCase/DeepNested | **herd_s3** | 0.4 MB/s | liteio | 0.1 MB/s | 2.4x faster |
| EdgeCase/EmptyObject | **herd_s3** | 3.5K ops/s | liteio | 1.7K ops/s | 2.0x faster |
| EdgeCase/LongKey256 | **herd_s3** | 0.3 MB/s | liteio | 0.1 MB/s | 2.0x faster |
| List/100 | **herd_s3** | 1.4K ops/s | liteio | 979 ops/s | +47% |
| MixedWorkload/Balanced_50_50 | **liteio** | 0.6 MB/s | herd_s3 | 0.6 MB/s | ~equal |
| MixedWorkload/ReadHeavy_90_10 | **herd_s3** | 0.6 MB/s | liteio | 0.5 MB/s | +14% |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 0.6 MB/s | herd_s3 | 0.5 MB/s | +33% |
| Multipart/15MB_3Parts | **liteio** | 135.1 MB/s | herd_s3 | 103.5 MB/s | +31% |
| ParallelRead/1KB/C1 | **liteio** | 4.1 MB/s | herd_s3 | 3.8 MB/s | ~equal |
| ParallelRead/1KB/C10 | **liteio** | 1.2 MB/s | herd_s3 | 1.1 MB/s | +13% |
| ParallelRead/1KB/C100 | **herd_s3** | 0.2 MB/s | liteio | 0.2 MB/s | ~equal |
| ParallelRead/1KB/C200 | **liteio** | 0.1 MB/s | herd_s3 | 0.1 MB/s | +13% |
| ParallelRead/1KB/C25 | **liteio** | 0.6 MB/s | herd_s3 | 0.6 MB/s | +14% |
| ParallelRead/1KB/C50 | **liteio** | 0.3 MB/s | herd_s3 | 0.3 MB/s | +12% |
| ParallelWrite/1KB/C1 | **herd_s3** | 3.4 MB/s | liteio | 1.4 MB/s | 2.4x faster |
| ParallelWrite/1KB/C10 | **herd_s3** | 1.0 MB/s | liteio | 0.5 MB/s | 2.0x faster |
| ParallelWrite/1KB/C100 | **herd_s3** | 0.1 MB/s | liteio | 0.1 MB/s | +27% |
| ParallelWrite/1KB/C200 | **herd_s3** | 0.1 MB/s | liteio | 0.0 MB/s | +39% |
| ParallelWrite/1KB/C25 | **herd_s3** | 0.5 MB/s | liteio | 0.3 MB/s | +88% |
| ParallelWrite/1KB/C50 | **herd_s3** | 0.2 MB/s | liteio | 0.1 MB/s | +93% |
| RangeRead/End_256KB | **liteio** | 154.5 MB/s | herd_s3 | 151.5 MB/s | ~equal |
| RangeRead/Middle_256KB | **liteio** | 149.2 MB/s | herd_s3 | 148.5 MB/s | ~equal |
| RangeRead/Start_256KB | **liteio** | 142.0 MB/s | herd_s3 | 134.5 MB/s | ~equal |
| Read/100MB | **liteio** | 183.0 MB/s | minio | 170.5 MB/s | ~equal |
| Read/10MB | **liteio** | 203.3 MB/s | herd_s3 | 183.1 MB/s | +11% |
| Read/1KB | **liteio** | 4.5 MB/s | herd_s3 | 3.5 MB/s | +29% |
| Read/1MB | **liteio** | 177.1 MB/s | herd_s3 | 168.1 MB/s | ~equal |
| Read/64KB | **liteio** | 117.2 MB/s | herd_s3 | 101.8 MB/s | +15% |
| Scale/Delete/10 | **liteio** | 517 ops/s | herd_s3 | 456 ops/s | +14% |
| Scale/Delete/100 | **liteio** | 52 ops/s | herd_s3 | 46 ops/s | +11% |
| Scale/Delete/1000 | **herd_s3** | 5 ops/s | liteio | 5 ops/s | ~equal |
| Scale/Delete/10000 | **liteio** | 0 ops/s | herd_s3 | 0 ops/s | +17% |
| Scale/List/10 | **liteio** | 3.1K ops/s | herd_s3 | 2.7K ops/s | +15% |
| Scale/List/100 | **liteio** | 1.2K ops/s | herd_s3 | 1.0K ops/s | +16% |
| Scale/List/1000 | **herd_s3** | 188 ops/s | liteio | 166 ops/s | +13% |
| Scale/List/10000 | **herd_s3** | 10 ops/s | minio | 5 ops/s | +78% |
| Scale/Write/10 | **herd_s3** | 1.0 MB/s | liteio | 0.4 MB/s | 2.4x faster |
| Scale/Write/100 | **herd_s3** | 1.0 MB/s | liteio | 0.4 MB/s | 2.8x faster |
| Scale/Write/1000 | **herd_s3** | 1.1 MB/s | liteio | 0.4 MB/s | 2.6x faster |
| Scale/Write/10000 | **herd_s3** | 1.1 MB/s | liteio | 0.4 MB/s | 3.1x faster |
| Stat | **herd_s3** | 4.5K ops/s | liteio | 3.7K ops/s | +24% |
| Write/100MB | **herd_s3** | 172.2 MB/s | liteio | 125.2 MB/s | +38% |
| Write/10MB | **herd_s3** | 161.6 MB/s | minio | 153.6 MB/s | ~equal |
| Write/1KB | **herd_s3** | 4.4 MB/s | liteio | 1.5 MB/s | 3.0x faster |
| Write/1MB | **herd_s3** | 189.1 MB/s | liteio | 128.5 MB/s | +47% |
| Write/64KB | **herd_s3** | 129.1 MB/s | liteio | 65.5 MB/s | +97% |

## Category Summaries

### Write Operations

**Best for Write:** herd_s3 (won 5/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | herd_s3 | 172.2 MB/s | +38% |
| Write/10MB | herd_s3 | 161.6 MB/s | ~equal |
| Write/1KB | herd_s3 | 4.4 MB/s | 3.0x faster |
| Write/1MB | herd_s3 | 189.1 MB/s | +47% |
| Write/64KB | herd_s3 | 129.1 MB/s | +97% |

### Read Operations

**Best for Read:** liteio (won 5/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | liteio | 183.0 MB/s | ~equal |
| Read/10MB | liteio | 203.3 MB/s | +11% |
| Read/1KB | liteio | 4.5 MB/s | +29% |
| Read/1MB | liteio | 177.1 MB/s | ~equal |
| Read/64KB | liteio | 117.2 MB/s | +15% |

### ParallelWrite Operations

**Best for ParallelWrite:** herd_s3 (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | herd_s3 | 3.4 MB/s | 2.4x faster |
| ParallelWrite/1KB/C10 | herd_s3 | 1.0 MB/s | 2.0x faster |
| ParallelWrite/1KB/C100 | herd_s3 | 0.1 MB/s | +27% |
| ParallelWrite/1KB/C200 | herd_s3 | 0.1 MB/s | +39% |
| ParallelWrite/1KB/C25 | herd_s3 | 0.5 MB/s | +88% |
| ParallelWrite/1KB/C50 | herd_s3 | 0.2 MB/s | +93% |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 5/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 4.1 MB/s | ~equal |
| ParallelRead/1KB/C10 | liteio | 1.2 MB/s | +13% |
| ParallelRead/1KB/C100 | herd_s3 | 0.2 MB/s | ~equal |
| ParallelRead/1KB/C200 | liteio | 0.1 MB/s | +13% |
| ParallelRead/1KB/C25 | liteio | 0.6 MB/s | +14% |
| ParallelRead/1KB/C50 | liteio | 0.3 MB/s | +12% |

### Delete Operations

**Best for Delete:** herd_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | herd_s3 | 5.1K ops/s | ~equal |

### Stat Operations

**Best for Stat:** herd_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | herd_s3 | 4.5K ops/s | +24% |

### List Operations

**Best for List:** herd_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | herd_s3 | 1.4K ops/s | +47% |

### Copy Operations

**Best for Copy:** herd_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | herd_s3 | 4.5 MB/s | 3.2x faster |

### Scale Operations

**Best for Scale:** herd_s3 (won 7/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | liteio | 517 ops/s | +14% |
| Scale/Delete/100 | liteio | 52 ops/s | +11% |
| Scale/Delete/1000 | herd_s3 | 5 ops/s | ~equal |
| Scale/Delete/10000 | liteio | 0 ops/s | +17% |
| Scale/List/10 | liteio | 3.1K ops/s | +15% |
| Scale/List/100 | liteio | 1.2K ops/s | +16% |
| Scale/List/1000 | herd_s3 | 188 ops/s | +13% |
| Scale/List/10000 | herd_s3 | 10 ops/s | +78% |
| Scale/Write/10 | herd_s3 | 1.0 MB/s | 2.4x faster |
| Scale/Write/100 | herd_s3 | 1.0 MB/s | 2.8x faster |
| Scale/Write/1000 | herd_s3 | 1.1 MB/s | 2.6x faster |
| Scale/Write/10000 | herd_s3 | 1.1 MB/s | 3.1x faster |

---

*Generated by storage benchmark CLI*
