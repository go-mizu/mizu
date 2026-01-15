# Storage Benchmark Summary

**Generated:** 2026-01-15T23:56:53+07:00

## Overall Winner

**liteio** won 39/51 categories (76%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| liteio | 39 | 76% |
| rustfs | 7 | 14% |
| minio | 5 | 10% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **liteio** | 2.2 MB/s | minio | 0.9 MB/s | 2.3x faster |
| Delete | **liteio** | 5.1K ops/s | minio | 2.7K ops/s | +85% |
| EdgeCase/DeepNested | **liteio** | 0.4 MB/s | rustfs | 0.1 MB/s | 3.1x faster |
| EdgeCase/EmptyObject | **rustfs** | 1.3K ops/s | liteio | 1.0K ops/s | +32% |
| EdgeCase/LongKey256 | **liteio** | 0.4 MB/s | rustfs | 0.1 MB/s | 3.1x faster |
| FileCount/Delete/1 | **liteio** | 4.9K ops/s | minio | 2.1K ops/s | 2.3x faster |
| FileCount/Delete/10 | **liteio** | 491 ops/s | minio | 238 ops/s | 2.1x faster |
| FileCount/Delete/100 | **liteio** | 54 ops/s | minio | 23 ops/s | 2.3x faster |
| FileCount/Delete/1000 | **liteio** | 5 ops/s | minio | 3 ops/s | +98% |
| FileCount/Delete/10000 | **liteio** | 0 ops/s | minio | 0 ops/s | +92% |
| FileCount/List/1 | **liteio** | 2.7K ops/s | minio | 1.5K ops/s | +76% |
| FileCount/List/10 | **liteio** | 2.9K ops/s | minio | 1.6K ops/s | +86% |
| FileCount/List/100 | **liteio** | 1.1K ops/s | minio | 354 ops/s | 3.0x faster |
| FileCount/List/1000 | **liteio** | 186 ops/s | minio | 53 ops/s | 3.5x faster |
| FileCount/List/10000 | **minio** | 5 ops/s | liteio | 5 ops/s | ~equal |
| FileCount/Write/1 | **liteio** | 2.4 MB/s | rustfs | 0.8 MB/s | 3.1x faster |
| FileCount/Write/10 | **liteio** | 3.9 MB/s | rustfs | 1.3 MB/s | 3.1x faster |
| FileCount/Write/100 | **liteio** | 4.2 MB/s | rustfs | 1.3 MB/s | 3.2x faster |
| FileCount/Write/1000 | **liteio** | 4.0 MB/s | rustfs | 1.3 MB/s | 3.0x faster |
| FileCount/Write/10000 | **liteio** | 4.0 MB/s | rustfs | 1.4 MB/s | 3.0x faster |
| List/100 | **liteio** | 1.2K ops/s | minio | 664 ops/s | +77% |
| MixedWorkload/Balanced_50_50 | **rustfs** | 2.2 MB/s | liteio | 2.1 MB/s | ~equal |
| MixedWorkload/ReadHeavy_90_10 | **minio** | 3.4 MB/s | rustfs | 2.5 MB/s | +34% |
| MixedWorkload/WriteHeavy_10_90 | **rustfs** | 1.3 MB/s | minio | 1.2 MB/s | ~equal |
| Multipart/15MB_3Parts | **minio** | 145.6 MB/s | rustfs | 125.9 MB/s | +16% |
| ParallelRead/1KB/C1 | **liteio** | 4.4 MB/s | minio | 2.1 MB/s | 2.0x faster |
| ParallelRead/1KB/C10 | **liteio** | 1.5 MB/s | minio | 1.4 MB/s | ~equal |
| ParallelRead/1KB/C100 | **rustfs** | 0.4 MB/s | liteio | 0.4 MB/s | ~equal |
| ParallelRead/1KB/C200 | **rustfs** | 0.4 MB/s | minio | 0.4 MB/s | ~equal |
| ParallelRead/1KB/C25 | **liteio** | 0.8 MB/s | minio | 0.8 MB/s | ~equal |
| ParallelRead/1KB/C50 | **liteio** | 0.5 MB/s | rustfs | 0.5 MB/s | ~equal |
| ParallelWrite/1KB/C1 | **liteio** | 2.9 MB/s | rustfs | 1.2 MB/s | 2.5x faster |
| ParallelWrite/1KB/C10 | **liteio** | 0.8 MB/s | rustfs | 0.5 MB/s | +46% |
| ParallelWrite/1KB/C100 | **rustfs** | 0.1 MB/s | liteio | 0.1 MB/s | +63% |
| ParallelWrite/1KB/C200 | **rustfs** | 0.1 MB/s | liteio | 0.1 MB/s | +11% |
| ParallelWrite/1KB/C25 | **liteio** | 0.3 MB/s | rustfs | 0.2 MB/s | +30% |
| ParallelWrite/1KB/C50 | **liteio** | 0.2 MB/s | rustfs | 0.1 MB/s | +23% |
| RangeRead/End_256KB | **liteio** | 189.2 MB/s | minio | 162.6 MB/s | +16% |
| RangeRead/Middle_256KB | **liteio** | 206.8 MB/s | minio | 165.7 MB/s | +25% |
| RangeRead/Start_256KB | **liteio** | 205.8 MB/s | minio | 140.4 MB/s | +47% |
| Read/100MB | **minio** | 329.4 MB/s | rustfs | 322.8 MB/s | ~equal |
| Read/10MB | **minio** | 314.2 MB/s | rustfs | 277.3 MB/s | +13% |
| Read/1KB | **liteio** | 4.5 MB/s | minio | 2.5 MB/s | +79% |
| Read/1MB | **liteio** | 263.3 MB/s | minio | 233.7 MB/s | +13% |
| Read/64KB | **liteio** | 123.3 MB/s | minio | 99.5 MB/s | +24% |
| Stat | **liteio** | 5.3K ops/s | minio | 3.7K ops/s | +42% |
| Write/100MB | **liteio** | 196.6 MB/s | rustfs | 166.5 MB/s | +18% |
| Write/10MB | **liteio** | 210.4 MB/s | rustfs | 170.0 MB/s | +24% |
| Write/1KB | **liteio** | 2.3 MB/s | rustfs | 1.2 MB/s | +95% |
| Write/1MB | **liteio** | 202.8 MB/s | rustfs | 144.5 MB/s | +40% |
| Write/64KB | **liteio** | 104.6 MB/s | rustfs | 55.6 MB/s | +88% |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 5/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | liteio | 196.6 MB/s | +18% |
| Write/10MB | liteio | 210.4 MB/s | +24% |
| Write/1KB | liteio | 2.3 MB/s | +95% |
| Write/1MB | liteio | 202.8 MB/s | +40% |
| Write/64KB | liteio | 104.6 MB/s | +88% |

### Read Operations

**Best for Read:** liteio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 329.4 MB/s | ~equal |
| Read/10MB | minio | 314.2 MB/s | +13% |
| Read/1KB | liteio | 4.5 MB/s | +79% |
| Read/1MB | liteio | 263.3 MB/s | +13% |
| Read/64KB | liteio | 123.3 MB/s | +24% |

### ParallelWrite Operations

**Best for ParallelWrite:** liteio (won 4/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio | 2.9 MB/s | 2.5x faster |
| ParallelWrite/1KB/C10 | liteio | 0.8 MB/s | +46% |
| ParallelWrite/1KB/C100 | rustfs | 0.1 MB/s | +63% |
| ParallelWrite/1KB/C200 | rustfs | 0.1 MB/s | +11% |
| ParallelWrite/1KB/C25 | liteio | 0.3 MB/s | +30% |
| ParallelWrite/1KB/C50 | liteio | 0.2 MB/s | +23% |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 4/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 4.4 MB/s | 2.0x faster |
| ParallelRead/1KB/C10 | liteio | 1.5 MB/s | ~equal |
| ParallelRead/1KB/C100 | rustfs | 0.4 MB/s | ~equal |
| ParallelRead/1KB/C200 | rustfs | 0.4 MB/s | ~equal |
| ParallelRead/1KB/C25 | liteio | 0.8 MB/s | ~equal |
| ParallelRead/1KB/C50 | liteio | 0.5 MB/s | ~equal |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 5.1K ops/s | +85% |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 5.3K ops/s | +42% |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 1.2K ops/s | +77% |

### Copy Operations

**Best for Copy:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio | 2.2 MB/s | 2.3x faster |

### FileCount Operations

**Best for FileCount:** liteio (won 14/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | liteio | 4.9K ops/s | 2.3x faster |
| FileCount/Delete/10 | liteio | 491 ops/s | 2.1x faster |
| FileCount/Delete/100 | liteio | 54 ops/s | 2.3x faster |
| FileCount/Delete/1000 | liteio | 5 ops/s | +98% |
| FileCount/Delete/10000 | liteio | 0 ops/s | +92% |
| FileCount/List/1 | liteio | 2.7K ops/s | +76% |
| FileCount/List/10 | liteio | 2.9K ops/s | +86% |
| FileCount/List/100 | liteio | 1.1K ops/s | 3.0x faster |
| FileCount/List/1000 | liteio | 186 ops/s | 3.5x faster |
| FileCount/List/10000 | minio | 5 ops/s | ~equal |
| FileCount/Write/1 | liteio | 2.4 MB/s | 3.1x faster |
| FileCount/Write/10 | liteio | 3.9 MB/s | 3.1x faster |
| FileCount/Write/100 | liteio | 4.2 MB/s | 3.2x faster |
| FileCount/Write/1000 | liteio | 4.0 MB/s | 3.0x faster |
| FileCount/Write/10000 | liteio | 4.0 MB/s | 3.0x faster |

---

*Generated by storage benchmark CLI*
