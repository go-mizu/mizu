# Storage Benchmark Summary

**Generated:** 2026-01-15T22:46:56+07:00

## Overall Winner

**liteio** won 35/51 categories (69%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| liteio | 35 | 69% |
| rustfs | 14 | 27% |
| minio | 2 | 4% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **liteio** | 1.1 MB/s | rustfs | 1.0 MB/s | +10% |
| Delete | **liteio** | 4.3K ops/s | minio | 2.6K ops/s | +67% |
| EdgeCase/DeepNested | **liteio** | 0.1 MB/s | rustfs | 0.1 MB/s | ~equal |
| EdgeCase/EmptyObject | **rustfs** | 1.3K ops/s | minio | 970 ops/s | +31% |
| EdgeCase/LongKey256 | **rustfs** | 0.1 MB/s | liteio | 0.1 MB/s | +13% |
| FileCount/Delete/1 | **liteio** | 4.9K ops/s | minio | 2.1K ops/s | 2.3x faster |
| FileCount/Delete/10 | **liteio** | 578 ops/s | minio | 232 ops/s | 2.5x faster |
| FileCount/Delete/100 | **liteio** | 39 ops/s | minio | 19 ops/s | 2.0x faster |
| FileCount/Delete/1000 | **liteio** | 5 ops/s | minio | 3 ops/s | 2.1x faster |
| FileCount/Delete/10000 | **liteio** | 1 ops/s | minio | 0 ops/s | 2.6x faster |
| FileCount/List/1 | **liteio** | 3.0K ops/s | minio | 1.7K ops/s | +78% |
| FileCount/List/10 | **liteio** | 3.1K ops/s | minio | 1.2K ops/s | 2.5x faster |
| FileCount/List/100 | **liteio** | 775 ops/s | minio | 321 ops/s | 2.4x faster |
| FileCount/List/1000 | **liteio** | 100 ops/s | minio | 64 ops/s | +56% |
| FileCount/List/10000 | **liteio** | 4 ops/s | minio | 4 ops/s | ~equal |
| FileCount/Write/1 | **liteio** | 0.7 MB/s | minio | 0.5 MB/s | +44% |
| FileCount/Write/10 | **rustfs** | 1.2 MB/s | liteio | 1.1 MB/s | ~equal |
| FileCount/Write/100 | **rustfs** | 1.3 MB/s | liteio | 1.2 MB/s | ~equal |
| FileCount/Write/1000 | **rustfs** | 1.3 MB/s | liteio | 1.2 MB/s | ~equal |
| FileCount/Write/10000 | **rustfs** | 1.3 MB/s | liteio | 1.2 MB/s | +10% |
| List/100 | **liteio** | 1.4K ops/s | minio | 573 ops/s | 2.4x faster |
| MixedWorkload/Balanced_50_50 | **liteio** | 2.3 MB/s | rustfs | 1.8 MB/s | +23% |
| MixedWorkload/ReadHeavy_90_10 | **rustfs** | 2.8 MB/s | liteio | 1.9 MB/s | +47% |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 1.4 MB/s | rustfs | 1.1 MB/s | +32% |
| Multipart/15MB_3Parts | **rustfs** | 162.9 MB/s | minio | 150.6 MB/s | ~equal |
| ParallelRead/1KB/C1 | **liteio** | 3.8 MB/s | minio | 2.4 MB/s | +58% |
| ParallelRead/1KB/C10 | **liteio** | 1.7 MB/s | minio | 1.1 MB/s | +51% |
| ParallelRead/1KB/C100 | **liteio** | 0.5 MB/s | rustfs | 0.5 MB/s | ~equal |
| ParallelRead/1KB/C200 | **liteio** | 1.5 MB/s | minio | 0.7 MB/s | 2.2x faster |
| ParallelRead/1KB/C25 | **liteio** | 1.2 MB/s | minio | 0.8 MB/s | +66% |
| ParallelRead/1KB/C50 | **liteio** | 0.8 MB/s | minio | 0.6 MB/s | +46% |
| ParallelWrite/1KB/C1 | **rustfs** | 1.1 MB/s | liteio | 1.0 MB/s | +12% |
| ParallelWrite/1KB/C10 | **rustfs** | 0.5 MB/s | minio | 0.4 MB/s | +24% |
| ParallelWrite/1KB/C100 | **rustfs** | 0.1 MB/s | liteio | 0.1 MB/s | +56% |
| ParallelWrite/1KB/C200 | **rustfs** | 0.1 MB/s | minio | 0.1 MB/s | +36% |
| ParallelWrite/1KB/C25 | **liteio** | 0.2 MB/s | rustfs | 0.2 MB/s | ~equal |
| ParallelWrite/1KB/C50 | **liteio** | 0.2 MB/s | rustfs | 0.1 MB/s | +17% |
| RangeRead/End_256KB | **liteio** | 245.7 MB/s | minio | 138.7 MB/s | +77% |
| RangeRead/Middle_256KB | **liteio** | 249.0 MB/s | minio | 165.0 MB/s | +51% |
| RangeRead/Start_256KB | **liteio** | 237.3 MB/s | minio | 135.1 MB/s | +76% |
| Read/100MB | **minio** | 325.0 MB/s | liteio | 305.0 MB/s | ~equal |
| Read/10MB | **minio** | 289.4 MB/s | rustfs | 287.2 MB/s | ~equal |
| Read/1KB | **liteio** | 3.5 MB/s | minio | 2.8 MB/s | +26% |
| Read/1MB | **liteio** | 286.8 MB/s | minio | 248.3 MB/s | +16% |
| Read/64KB | **liteio** | 130.0 MB/s | minio | 99.9 MB/s | +30% |
| Stat | **liteio** | 5.9K ops/s | minio | 3.8K ops/s | +55% |
| Write/100MB | **rustfs** | 166.5 MB/s | minio | 163.5 MB/s | ~equal |
| Write/10MB | **liteio** | 174.0 MB/s | minio | 159.1 MB/s | ~equal |
| Write/1KB | **rustfs** | 1.3 MB/s | liteio | 1.2 MB/s | ~equal |
| Write/1MB | **liteio** | 147.5 MB/s | minio | 124.2 MB/s | +19% |
| Write/64KB | **liteio** | 62.5 MB/s | rustfs | 56.7 MB/s | +10% |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | rustfs | 166.5 MB/s | ~equal |
| Write/10MB | liteio | 174.0 MB/s | ~equal |
| Write/1KB | rustfs | 1.3 MB/s | ~equal |
| Write/1MB | liteio | 147.5 MB/s | +19% |
| Write/64KB | liteio | 62.5 MB/s | +10% |

### Read Operations

**Best for Read:** liteio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 325.0 MB/s | ~equal |
| Read/10MB | minio | 289.4 MB/s | ~equal |
| Read/1KB | liteio | 3.5 MB/s | +26% |
| Read/1MB | liteio | 286.8 MB/s | +16% |
| Read/64KB | liteio | 130.0 MB/s | +30% |

### ParallelWrite Operations

**Best for ParallelWrite:** rustfs (won 4/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | rustfs | 1.1 MB/s | +12% |
| ParallelWrite/1KB/C10 | rustfs | 0.5 MB/s | +24% |
| ParallelWrite/1KB/C100 | rustfs | 0.1 MB/s | +56% |
| ParallelWrite/1KB/C200 | rustfs | 0.1 MB/s | +36% |
| ParallelWrite/1KB/C25 | liteio | 0.2 MB/s | ~equal |
| ParallelWrite/1KB/C50 | liteio | 0.2 MB/s | +17% |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 3.8 MB/s | +58% |
| ParallelRead/1KB/C10 | liteio | 1.7 MB/s | +51% |
| ParallelRead/1KB/C100 | liteio | 0.5 MB/s | ~equal |
| ParallelRead/1KB/C200 | liteio | 1.5 MB/s | 2.2x faster |
| ParallelRead/1KB/C25 | liteio | 1.2 MB/s | +66% |
| ParallelRead/1KB/C50 | liteio | 0.8 MB/s | +46% |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 4.3K ops/s | +67% |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 5.9K ops/s | +55% |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 1.4K ops/s | 2.4x faster |

### Copy Operations

**Best for Copy:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio | 1.1 MB/s | +10% |

### FileCount Operations

**Best for FileCount:** liteio (won 11/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | liteio | 4.9K ops/s | 2.3x faster |
| FileCount/Delete/10 | liteio | 578 ops/s | 2.5x faster |
| FileCount/Delete/100 | liteio | 39 ops/s | 2.0x faster |
| FileCount/Delete/1000 | liteio | 5 ops/s | 2.1x faster |
| FileCount/Delete/10000 | liteio | 1 ops/s | 2.6x faster |
| FileCount/List/1 | liteio | 3.0K ops/s | +78% |
| FileCount/List/10 | liteio | 3.1K ops/s | 2.5x faster |
| FileCount/List/100 | liteio | 775 ops/s | 2.4x faster |
| FileCount/List/1000 | liteio | 100 ops/s | +56% |
| FileCount/List/10000 | liteio | 4 ops/s | ~equal |
| FileCount/Write/1 | liteio | 0.7 MB/s | +44% |
| FileCount/Write/10 | rustfs | 1.2 MB/s | ~equal |
| FileCount/Write/100 | rustfs | 1.3 MB/s | ~equal |
| FileCount/Write/1000 | rustfs | 1.3 MB/s | ~equal |
| FileCount/Write/10000 | rustfs | 1.3 MB/s | +10% |

---

*Generated by storage benchmark CLI*
