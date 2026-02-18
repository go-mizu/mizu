# Storage Benchmark Summary

**Generated:** 2026-02-19T00:46:44+07:00

## Overall Winner

**devnull_s3** won 33/48 categories (69%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| devnull_s3 | 33 | 69% |
| liteio | 15 | 31% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **devnull_s3** | 15.5 MB/s | liteio | 2.4 MB/s | 6.6x faster |
| Delete | **devnull_s3** | 18.8K ops/s | liteio | 9.5K ops/s | +98% |
| EdgeCase/DeepNested | **devnull_s3** | 1.5 MB/s | liteio | 0.2 MB/s | 6.3x faster |
| EdgeCase/EmptyObject | **devnull_s3** | 16.1K ops/s | liteio | 3.3K ops/s | 4.9x faster |
| EdgeCase/LongKey256 | **devnull_s3** | 1.4 MB/s | liteio | 0.2 MB/s | 5.8x faster |
| List/100 | **devnull_s3** | 1.4K ops/s | liteio | 1.2K ops/s | +21% |
| MixedWorkload/Balanced_50_50 | **devnull_s3** | 4.2 MB/s | liteio | 0.5 MB/s | 7.9x faster |
| MixedWorkload/ReadHeavy_90_10 | **devnull_s3** | 4.8 MB/s | liteio | 2.7 MB/s | +76% |
| MixedWorkload/WriteHeavy_10_90 | **devnull_s3** | 4.1 MB/s | liteio | 0.3 MB/s | 15.5x faster |
| Multipart/15MB_3Parts | **devnull_s3** | 1.3 GB/s | liteio | 214.8 MB/s | 5.9x faster |
| ParallelRead/1KB/C1 | **devnull_s3** | 10.2 MB/s | liteio | 9.6 MB/s | ~equal |
| ParallelRead/1KB/C10 | **liteio** | 4.3 MB/s | devnull_s3 | 0.0 MB/s |  |
| ParallelRead/1KB/C100 | **liteio** | 0.7 MB/s | devnull_s3 | 0.0 MB/s |  |
| ParallelRead/1KB/C200 | **liteio** | 0.3 MB/s | devnull_s3 | 0.0 MB/s |  |
| ParallelRead/1KB/C25 | **liteio** | 1.7 MB/s | devnull_s3 | 0.0 MB/s |  |
| ParallelRead/1KB/C50 | **liteio** | 1.0 MB/s | devnull_s3 | 0.0 MB/s |  |
| ParallelWrite/1KB/C1 | **devnull_s3** | 9.8 MB/s | liteio | 2.0 MB/s | 5.0x faster |
| ParallelWrite/1KB/C10 | **devnull_s3** | 4.1 MB/s | liteio | 0.5 MB/s | 8.8x faster |
| ParallelWrite/1KB/C100 | **devnull_s3** | 0.5 MB/s | liteio | 0.0 MB/s | 13.0x faster |
| ParallelWrite/1KB/C200 | **devnull_s3** | 0.1 MB/s | liteio | 0.0 MB/s | 4.3x faster |
| ParallelWrite/1KB/C25 | **devnull_s3** | 1.9 MB/s | liteio | 0.2 MB/s | 9.3x faster |
| ParallelWrite/1KB/C50 | **devnull_s3** | 0.7 MB/s | liteio | 0.1 MB/s | 7.2x faster |
| RangeRead/End_256KB | **liteio** | 2.2 GB/s | devnull_s3 | 0.0 MB/s |  |
| RangeRead/Middle_256KB | **liteio** | 2.2 GB/s | devnull_s3 | 0.0 MB/s |  |
| RangeRead/Start_256KB | **liteio** | 2.2 GB/s | devnull_s3 | 0.0 MB/s |  |
| Read/100MB | **devnull_s3** | 8.6 GB/s | liteio | 2.2 GB/s | 3.9x faster |
| Read/10MB | **devnull_s3** | 5.9 GB/s | liteio | 2.3 GB/s | 2.5x faster |
| Read/1KB | **devnull_s3** | 16.9 MB/s | liteio | 14.8 MB/s | +14% |
| Read/1MB | **devnull_s3** | 5.7 GB/s | liteio | 2.8 GB/s | 2.0x faster |
| Read/64KB | **devnull_s3** | 972.2 MB/s | liteio | 907.8 MB/s | ~equal |
| Scale/Delete/10 | **liteio** | 894 ops/s | devnull_s3 | 814 ops/s | ~equal |
| Scale/Delete/100 | **liteio** | 99 ops/s | devnull_s3 | 45 ops/s | 2.2x faster |
| Scale/Delete/1000 | **liteio** | 10 ops/s | devnull_s3 | 7 ops/s | +38% |
| Scale/Delete/10000 | **devnull_s3** | 2 ops/s | liteio | 1 ops/s | +76% |
| Scale/List/10 | **liteio** | 4.6K ops/s | devnull_s3 | 0 ops/s |  |
| Scale/List/100 | **liteio** | 1.3K ops/s | devnull_s3 | 0 ops/s |  |
| Scale/List/1000 | **liteio** | 146 ops/s | devnull_s3 | 0 ops/s |  |
| Scale/List/10000 | **liteio** | 3 ops/s | devnull_s3 | 0 ops/s |  |
| Scale/Write/10 | **devnull_s3** | 1.6 MB/s | liteio | 0.7 MB/s | 2.4x faster |
| Scale/Write/100 | **devnull_s3** | 1.2 MB/s | liteio | 0.7 MB/s | +83% |
| Scale/Write/1000 | **devnull_s3** | 1.3 MB/s | liteio | 0.7 MB/s | +88% |
| Scale/Write/10000 | **devnull_s3** | 2.0 MB/s | liteio | 0.6 MB/s | 3.1x faster |
| Stat | **devnull_s3** | 18.1K ops/s | liteio | 10.2K ops/s | +77% |
| Write/100MB | **devnull_s3** | 1.5 GB/s | liteio | 850.3 MB/s | +80% |
| Write/10MB | **devnull_s3** | 1.3 GB/s | liteio | 955.6 MB/s | +36% |
| Write/1KB | **devnull_s3** | 15.5 MB/s | liteio | 0.9 MB/s | 16.7x faster |
| Write/1MB | **devnull_s3** | 1.2 GB/s | liteio | 706.1 MB/s | +75% |
| Write/64KB | **devnull_s3** | 566.6 MB/s | liteio | 150.6 MB/s | 3.8x faster |

## Category Summaries

### Write Operations

**Best for Write:** devnull_s3 (won 5/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | devnull_s3 | 1.5 GB/s | +80% |
| Write/10MB | devnull_s3 | 1.3 GB/s | +36% |
| Write/1KB | devnull_s3 | 15.5 MB/s | 16.7x faster |
| Write/1MB | devnull_s3 | 1.2 GB/s | +75% |
| Write/64KB | devnull_s3 | 566.6 MB/s | 3.8x faster |

### Read Operations

**Best for Read:** devnull_s3 (won 5/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | devnull_s3 | 8.6 GB/s | 3.9x faster |
| Read/10MB | devnull_s3 | 5.9 GB/s | 2.5x faster |
| Read/1KB | devnull_s3 | 16.9 MB/s | +14% |
| Read/1MB | devnull_s3 | 5.7 GB/s | 2.0x faster |
| Read/64KB | devnull_s3 | 972.2 MB/s | ~equal |

### ParallelWrite Operations

**Best for ParallelWrite:** devnull_s3 (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | devnull_s3 | 9.8 MB/s | 5.0x faster |
| ParallelWrite/1KB/C10 | devnull_s3 | 4.1 MB/s | 8.8x faster |
| ParallelWrite/1KB/C100 | devnull_s3 | 0.5 MB/s | 13.0x faster |
| ParallelWrite/1KB/C200 | devnull_s3 | 0.1 MB/s | 4.3x faster |
| ParallelWrite/1KB/C25 | devnull_s3 | 1.9 MB/s | 9.3x faster |
| ParallelWrite/1KB/C50 | devnull_s3 | 0.7 MB/s | 7.2x faster |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 5/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | devnull_s3 | 10.2 MB/s | ~equal |
| ParallelRead/1KB/C10 | liteio | 4.3 MB/s |  |
| ParallelRead/1KB/C100 | liteio | 0.7 MB/s |  |
| ParallelRead/1KB/C200 | liteio | 0.3 MB/s |  |
| ParallelRead/1KB/C25 | liteio | 1.7 MB/s |  |
| ParallelRead/1KB/C50 | liteio | 1.0 MB/s |  |

### Delete Operations

**Best for Delete:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | devnull_s3 | 18.8K ops/s | +98% |

### Stat Operations

**Best for Stat:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | devnull_s3 | 18.1K ops/s | +77% |

### List Operations

**Best for List:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | devnull_s3 | 1.4K ops/s | +21% |

### Copy Operations

**Best for Copy:** devnull_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | devnull_s3 | 15.5 MB/s | 6.6x faster |

### Scale Operations

**Best for Scale:** liteio (won 7/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | liteio | 894 ops/s | ~equal |
| Scale/Delete/100 | liteio | 99 ops/s | 2.2x faster |
| Scale/Delete/1000 | liteio | 10 ops/s | +38% |
| Scale/Delete/10000 | devnull_s3 | 2 ops/s | +76% |
| Scale/List/10 | liteio | 4.6K ops/s |  |
| Scale/List/100 | liteio | 1.3K ops/s |  |
| Scale/List/1000 | liteio | 146 ops/s |  |
| Scale/List/10000 | liteio | 3 ops/s |  |
| Scale/Write/10 | devnull_s3 | 1.6 MB/s | 2.4x faster |
| Scale/Write/100 | devnull_s3 | 1.2 MB/s | +83% |
| Scale/Write/1000 | devnull_s3 | 1.3 MB/s | +88% |
| Scale/Write/10000 | devnull_s3 | 2.0 MB/s | 3.1x faster |

---

*Generated by storage benchmark CLI*
