# Storage Benchmark Summary

**Generated:** 2026-01-16T01:17:28+07:00

## Overall Winner

**liteio** won 45/51 categories (88%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| liteio | 45 | 88% |
| minio | 4 | 8% |
| rustfs | 2 | 4% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **liteio** | 1.3 MB/s | minio | 1.0 MB/s | +33% |
| Delete | **minio** | 3.2K ops/s | liteio | 3.2K ops/s | ~equal |
| EdgeCase/DeepNested | **liteio** | 0.3 MB/s | rustfs | 0.1 MB/s | 2.5x faster |
| EdgeCase/EmptyObject | **liteio** | 2.4K ops/s | minio | 1.1K ops/s | 2.3x faster |
| EdgeCase/LongKey256 | **liteio** | 0.3 MB/s | rustfs | 0.1 MB/s | 2.2x faster |
| FileCount/Delete/1 | **liteio** | 4.2K ops/s | minio | 2.4K ops/s | +75% |
| FileCount/Delete/10 | **liteio** | 456 ops/s | minio | 225 ops/s | 2.0x faster |
| FileCount/Delete/100 | **liteio** | 52 ops/s | minio | 28 ops/s | +87% |
| FileCount/Delete/1000 | **liteio** | 5 ops/s | minio | 3 ops/s | +95% |
| FileCount/Delete/10000 | **liteio** | 1 ops/s | minio | 0 ops/s | 2.2x faster |
| FileCount/List/1 | **liteio** | 3.7K ops/s | minio | 2.3K ops/s | +64% |
| FileCount/List/10 | **liteio** | 2.9K ops/s | minio | 1.1K ops/s | 2.7x faster |
| FileCount/List/100 | **liteio** | 1.1K ops/s | minio | 577 ops/s | +93% |
| FileCount/List/1000 | **liteio** | 177 ops/s | minio | 82 ops/s | 2.1x faster |
| FileCount/List/10000 | **minio** | 6 ops/s | liteio | 5 ops/s | +17% |
| FileCount/Write/1 | **liteio** | 2.4 MB/s | rustfs | 1.0 MB/s | 2.4x faster |
| FileCount/Write/10 | **liteio** | 3.6 MB/s | rustfs | 1.3 MB/s | 2.6x faster |
| FileCount/Write/100 | **liteio** | 3.8 MB/s | rustfs | 1.4 MB/s | 2.7x faster |
| FileCount/Write/1000 | **liteio** | 4.6 MB/s | rustfs | 1.3 MB/s | 3.4x faster |
| FileCount/Write/10000 | **liteio** | 4.3 MB/s | rustfs | 1.3 MB/s | 3.3x faster |
| List/100 | **liteio** | 1.3K ops/s | minio | 581 ops/s | 2.2x faster |
| MixedWorkload/Balanced_50_50 | **liteio** | 11.0 MB/s | minio | 8.8 MB/s | +25% |
| MixedWorkload/ReadHeavy_90_10 | **liteio** | 7.3 MB/s | minio | 6.8 MB/s | ~equal |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 7.5 MB/s | rustfs | 5.0 MB/s | +50% |
| Multipart/15MB_3Parts | **rustfs** | 160.0 MB/s | minio | 152.2 MB/s | ~equal |
| ParallelRead/1KB/C1 | **liteio** | 3.4 MB/s | minio | 3.0 MB/s | +12% |
| ParallelRead/1KB/C10 | **liteio** | 1.8 MB/s | minio | 1.0 MB/s | +83% |
| ParallelRead/1KB/C100 | **liteio** | 1.8 MB/s | minio | 0.8 MB/s | 2.3x faster |
| ParallelRead/1KB/C200 | **liteio** | 1.6 MB/s | minio | 0.7 MB/s | 2.2x faster |
| ParallelRead/1KB/C25 | **rustfs** | 0.6 MB/s | minio | 0.6 MB/s | +13% |
| ParallelRead/1KB/C50 | **liteio** | 0.7 MB/s | minio | 0.7 MB/s | ~equal |
| ParallelWrite/1KB/C1 | **liteio** | 1.9 MB/s | rustfs | 1.2 MB/s | +54% |
| ParallelWrite/1KB/C10 | **liteio** | 0.4 MB/s | minio | 0.3 MB/s | +23% |
| ParallelWrite/1KB/C100 | **liteio** | 0.5 MB/s | minio | 0.2 MB/s | 2.2x faster |
| ParallelWrite/1KB/C200 | **liteio** | 0.4 MB/s | rustfs | 0.3 MB/s | +32% |
| ParallelWrite/1KB/C25 | **liteio** | 0.5 MB/s | minio | 0.2 MB/s | 2.6x faster |
| ParallelWrite/1KB/C50 | **liteio** | 0.5 MB/s | rustfs | 0.2 MB/s | +94% |
| RangeRead/End_256KB | **liteio** | 235.9 MB/s | minio | 147.0 MB/s | +60% |
| RangeRead/Middle_256KB | **liteio** | 218.3 MB/s | minio | 163.8 MB/s | +33% |
| RangeRead/Start_256KB | **liteio** | 181.8 MB/s | minio | 147.0 MB/s | +24% |
| Read/100MB | **minio** | 290.9 MB/s | rustfs | 283.5 MB/s | ~equal |
| Read/10MB | **minio** | 279.4 MB/s | liteio | 252.9 MB/s | +11% |
| Read/1KB | **liteio** | 4.6 MB/s | rustfs | 2.8 MB/s | +61% |
| Read/1MB | **liteio** | 253.8 MB/s | minio | 218.8 MB/s | +16% |
| Read/64KB | **liteio** | 138.6 MB/s | rustfs | 84.2 MB/s | +65% |
| Stat | **liteio** | 5.3K ops/s | minio | 4.2K ops/s | +25% |
| Write/100MB | **liteio** | 198.3 MB/s | rustfs | 184.1 MB/s | ~equal |
| Write/10MB | **liteio** | 197.3 MB/s | minio | 158.6 MB/s | +24% |
| Write/1KB | **liteio** | 1.5 MB/s | rustfs | 1.0 MB/s | +48% |
| Write/1MB | **liteio** | 176.6 MB/s | rustfs | 144.3 MB/s | +22% |
| Write/64KB | **liteio** | 109.6 MB/s | rustfs | 43.8 MB/s | 2.5x faster |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 5/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | liteio | 198.3 MB/s | ~equal |
| Write/10MB | liteio | 197.3 MB/s | +24% |
| Write/1KB | liteio | 1.5 MB/s | +48% |
| Write/1MB | liteio | 176.6 MB/s | +22% |
| Write/64KB | liteio | 109.6 MB/s | 2.5x faster |

### Read Operations

**Best for Read:** liteio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 290.9 MB/s | ~equal |
| Read/10MB | minio | 279.4 MB/s | +11% |
| Read/1KB | liteio | 4.6 MB/s | +61% |
| Read/1MB | liteio | 253.8 MB/s | +16% |
| Read/64KB | liteio | 138.6 MB/s | +65% |

### ParallelWrite Operations

**Best for ParallelWrite:** liteio (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio | 1.9 MB/s | +54% |
| ParallelWrite/1KB/C10 | liteio | 0.4 MB/s | +23% |
| ParallelWrite/1KB/C100 | liteio | 0.5 MB/s | 2.2x faster |
| ParallelWrite/1KB/C200 | liteio | 0.4 MB/s | +32% |
| ParallelWrite/1KB/C25 | liteio | 0.5 MB/s | 2.6x faster |
| ParallelWrite/1KB/C50 | liteio | 0.5 MB/s | +94% |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 5/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 3.4 MB/s | +12% |
| ParallelRead/1KB/C10 | liteio | 1.8 MB/s | +83% |
| ParallelRead/1KB/C100 | liteio | 1.8 MB/s | 2.3x faster |
| ParallelRead/1KB/C200 | liteio | 1.6 MB/s | 2.2x faster |
| ParallelRead/1KB/C25 | rustfs | 0.6 MB/s | +13% |
| ParallelRead/1KB/C50 | liteio | 0.7 MB/s | ~equal |

### Delete Operations

**Best for Delete:** minio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | minio | 3.2K ops/s | ~equal |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 5.3K ops/s | +25% |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 1.3K ops/s | 2.2x faster |

### Copy Operations

**Best for Copy:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio | 1.3 MB/s | +33% |

### FileCount Operations

**Best for FileCount:** liteio (won 14/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | liteio | 4.2K ops/s | +75% |
| FileCount/Delete/10 | liteio | 456 ops/s | 2.0x faster |
| FileCount/Delete/100 | liteio | 52 ops/s | +87% |
| FileCount/Delete/1000 | liteio | 5 ops/s | +95% |
| FileCount/Delete/10000 | liteio | 1 ops/s | 2.2x faster |
| FileCount/List/1 | liteio | 3.7K ops/s | +64% |
| FileCount/List/10 | liteio | 2.9K ops/s | 2.7x faster |
| FileCount/List/100 | liteio | 1.1K ops/s | +93% |
| FileCount/List/1000 | liteio | 177 ops/s | 2.1x faster |
| FileCount/List/10000 | minio | 6 ops/s | +17% |
| FileCount/Write/1 | liteio | 2.4 MB/s | 2.4x faster |
| FileCount/Write/10 | liteio | 3.6 MB/s | 2.6x faster |
| FileCount/Write/100 | liteio | 3.8 MB/s | 2.7x faster |
| FileCount/Write/1000 | liteio | 4.6 MB/s | 3.4x faster |
| FileCount/Write/10000 | liteio | 4.3 MB/s | 3.3x faster |

---

*Generated by storage benchmark CLI*
