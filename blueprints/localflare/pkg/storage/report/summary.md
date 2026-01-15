# Storage Benchmark Summary

**Generated:** 2026-01-16T01:05:58+07:00

## Overall Winner

**liteio** won 46/51 categories (90%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| liteio | 46 | 90% |
| minio | 3 | 6% |
| rustfs | 2 | 4% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **liteio** | 1.9 MB/s | minio | 0.9 MB/s | 2.0x faster |
| Delete | **liteio** | 4.9K ops/s | minio | 3.0K ops/s | +64% |
| EdgeCase/DeepNested | **liteio** | 0.4 MB/s | minio | 0.1 MB/s | 4.3x faster |
| EdgeCase/EmptyObject | **liteio** | 3.3K ops/s | minio | 1.0K ops/s | 3.3x faster |
| EdgeCase/LongKey256 | **liteio** | 0.4 MB/s | rustfs | 0.1 MB/s | 3.7x faster |
| FileCount/Delete/1 | **liteio** | 4.2K ops/s | minio | 2.2K ops/s | +89% |
| FileCount/Delete/10 | **liteio** | 430 ops/s | minio | 269 ops/s | +60% |
| FileCount/Delete/100 | **liteio** | 51 ops/s | minio | 27 ops/s | +86% |
| FileCount/Delete/1000 | **liteio** | 6 ops/s | minio | 3 ops/s | +98% |
| FileCount/Delete/10000 | **liteio** | 1 ops/s | minio | 0 ops/s | 2.2x faster |
| FileCount/List/1 | **liteio** | 4.0K ops/s | minio | 2.2K ops/s | +82% |
| FileCount/List/10 | **liteio** | 3.0K ops/s | minio | 1.7K ops/s | +80% |
| FileCount/List/100 | **liteio** | 1.2K ops/s | minio | 558 ops/s | 2.2x faster |
| FileCount/List/1000 | **liteio** | 185 ops/s | minio | 72 ops/s | 2.6x faster |
| FileCount/List/10000 | **minio** | 6 ops/s | liteio | 5 ops/s | ~equal |
| FileCount/Write/1 | **liteio** | 2.4 MB/s | rustfs | 1.2 MB/s | 2.1x faster |
| FileCount/Write/10 | **liteio** | 4.0 MB/s | rustfs | 1.2 MB/s | 3.2x faster |
| FileCount/Write/100 | **liteio** | 4.2 MB/s | rustfs | 1.3 MB/s | 3.2x faster |
| FileCount/Write/1000 | **liteio** | 4.7 MB/s | rustfs | 1.3 MB/s | 3.6x faster |
| FileCount/Write/10000 | **liteio** | 4.6 MB/s | rustfs | 1.3 MB/s | 3.5x faster |
| List/100 | **liteio** | 1.3K ops/s | minio | 414 ops/s | 3.2x faster |
| MixedWorkload/Balanced_50_50 | **liteio** | 10.2 MB/s | rustfs | 8.8 MB/s | +16% |
| MixedWorkload/ReadHeavy_90_10 | **rustfs** | 7.9 MB/s | liteio | 7.3 MB/s | ~equal |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 6.6 MB/s | minio | 4.8 MB/s | +39% |
| Multipart/15MB_3Parts | **rustfs** | 163.7 MB/s | minio | 162.3 MB/s | ~equal |
| ParallelRead/1KB/C1 | **liteio** | 3.8 MB/s | minio | 2.4 MB/s | +56% |
| ParallelRead/1KB/C10 | **liteio** | 1.8 MB/s | minio | 1.1 MB/s | +74% |
| ParallelRead/1KB/C100 | **liteio** | 0.9 MB/s | rustfs | 0.8 MB/s | +14% |
| ParallelRead/1KB/C200 | **liteio** | 1.8 MB/s | minio | 0.8 MB/s | 2.2x faster |
| ParallelRead/1KB/C25 | **liteio** | 1.5 MB/s | minio | 0.7 MB/s | 2.3x faster |
| ParallelRead/1KB/C50 | **liteio** | 1.0 MB/s | minio | 0.6 MB/s | +61% |
| ParallelWrite/1KB/C1 | **liteio** | 2.0 MB/s | rustfs | 1.2 MB/s | +69% |
| ParallelWrite/1KB/C10 | **liteio** | 0.7 MB/s | rustfs | 0.3 MB/s | 2.8x faster |
| ParallelWrite/1KB/C100 | **liteio** | 0.9 MB/s | minio | 0.3 MB/s | 3.4x faster |
| ParallelWrite/1KB/C200 | **liteio** | 1.0 MB/s | rustfs | 0.2 MB/s | 4.2x faster |
| ParallelWrite/1KB/C25 | **liteio** | 1.0 MB/s | rustfs | 0.3 MB/s | 4.0x faster |
| ParallelWrite/1KB/C50 | **liteio** | 0.8 MB/s | rustfs | 0.2 MB/s | 3.5x faster |
| RangeRead/End_256KB | **liteio** | 230.5 MB/s | minio | 166.1 MB/s | +39% |
| RangeRead/Middle_256KB | **liteio** | 241.5 MB/s | minio | 164.8 MB/s | +47% |
| RangeRead/Start_256KB | **liteio** | 165.9 MB/s | minio | 162.7 MB/s | ~equal |
| Read/100MB | **minio** | 312.1 MB/s | rustfs | 302.5 MB/s | ~equal |
| Read/10MB | **minio** | 303.9 MB/s | rustfs | 256.7 MB/s | +18% |
| Read/1KB | **liteio** | 5.8 MB/s | minio | 2.6 MB/s | 2.2x faster |
| Read/1MB | **liteio** | 285.8 MB/s | minio | 227.1 MB/s | +26% |
| Read/64KB | **liteio** | 153.7 MB/s | minio | 94.6 MB/s | +62% |
| Stat | **liteio** | 5.1K ops/s | minio | 4.2K ops/s | +22% |
| Write/100MB | **liteio** | 207.2 MB/s | minio | 167.2 MB/s | +24% |
| Write/10MB | **liteio** | 198.4 MB/s | rustfs | 162.9 MB/s | +22% |
| Write/1KB | **liteio** | 1.0 MB/s | rustfs | 0.9 MB/s | +11% |
| Write/1MB | **liteio** | 190.9 MB/s | rustfs | 144.3 MB/s | +32% |
| Write/64KB | **liteio** | 115.7 MB/s | rustfs | 54.9 MB/s | 2.1x faster |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 5/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | liteio | 207.2 MB/s | +24% |
| Write/10MB | liteio | 198.4 MB/s | +22% |
| Write/1KB | liteio | 1.0 MB/s | +11% |
| Write/1MB | liteio | 190.9 MB/s | +32% |
| Write/64KB | liteio | 115.7 MB/s | 2.1x faster |

### Read Operations

**Best for Read:** liteio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 312.1 MB/s | ~equal |
| Read/10MB | minio | 303.9 MB/s | +18% |
| Read/1KB | liteio | 5.8 MB/s | 2.2x faster |
| Read/1MB | liteio | 285.8 MB/s | +26% |
| Read/64KB | liteio | 153.7 MB/s | +62% |

### ParallelWrite Operations

**Best for ParallelWrite:** liteio (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio | 2.0 MB/s | +69% |
| ParallelWrite/1KB/C10 | liteio | 0.7 MB/s | 2.8x faster |
| ParallelWrite/1KB/C100 | liteio | 0.9 MB/s | 3.4x faster |
| ParallelWrite/1KB/C200 | liteio | 1.0 MB/s | 4.2x faster |
| ParallelWrite/1KB/C25 | liteio | 1.0 MB/s | 4.0x faster |
| ParallelWrite/1KB/C50 | liteio | 0.8 MB/s | 3.5x faster |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 3.8 MB/s | +56% |
| ParallelRead/1KB/C10 | liteio | 1.8 MB/s | +74% |
| ParallelRead/1KB/C100 | liteio | 0.9 MB/s | +14% |
| ParallelRead/1KB/C200 | liteio | 1.8 MB/s | 2.2x faster |
| ParallelRead/1KB/C25 | liteio | 1.5 MB/s | 2.3x faster |
| ParallelRead/1KB/C50 | liteio | 1.0 MB/s | +61% |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 4.9K ops/s | +64% |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 5.1K ops/s | +22% |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 1.3K ops/s | 3.2x faster |

### Copy Operations

**Best for Copy:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio | 1.9 MB/s | 2.0x faster |

### FileCount Operations

**Best for FileCount:** liteio (won 14/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | liteio | 4.2K ops/s | +89% |
| FileCount/Delete/10 | liteio | 430 ops/s | +60% |
| FileCount/Delete/100 | liteio | 51 ops/s | +86% |
| FileCount/Delete/1000 | liteio | 6 ops/s | +98% |
| FileCount/Delete/10000 | liteio | 1 ops/s | 2.2x faster |
| FileCount/List/1 | liteio | 4.0K ops/s | +82% |
| FileCount/List/10 | liteio | 3.0K ops/s | +80% |
| FileCount/List/100 | liteio | 1.2K ops/s | 2.2x faster |
| FileCount/List/1000 | liteio | 185 ops/s | 2.6x faster |
| FileCount/List/10000 | minio | 6 ops/s | ~equal |
| FileCount/Write/1 | liteio | 2.4 MB/s | 2.1x faster |
| FileCount/Write/10 | liteio | 4.0 MB/s | 3.2x faster |
| FileCount/Write/100 | liteio | 4.2 MB/s | 3.2x faster |
| FileCount/Write/1000 | liteio | 4.7 MB/s | 3.6x faster |
| FileCount/Write/10000 | liteio | 4.6 MB/s | 3.5x faster |

---

*Generated by storage benchmark CLI*
