# Storage Benchmark Summary

**Generated:** 2026-01-16T00:20:15+07:00

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
| Copy/1KB | **liteio** | 2.7 MB/s | rustfs | 0.9 MB/s | 3.0x faster |
| Delete | **liteio** | 5.0K ops/s | minio | 3.0K ops/s | +68% |
| EdgeCase/DeepNested | **liteio** | 0.4 MB/s | rustfs | 0.1 MB/s | 3.2x faster |
| EdgeCase/EmptyObject | **rustfs** | 1.4K ops/s | minio | 918 ops/s | +53% |
| EdgeCase/LongKey256 | **liteio** | 0.3 MB/s | rustfs | 0.1 MB/s | 2.3x faster |
| FileCount/Delete/1 | **liteio** | 5.1K ops/s | minio | 2.5K ops/s | 2.0x faster |
| FileCount/Delete/10 | **liteio** | 510 ops/s | minio | 282 ops/s | +81% |
| FileCount/Delete/100 | **liteio** | 53 ops/s | minio | 29 ops/s | +85% |
| FileCount/Delete/1000 | **liteio** | 6 ops/s | minio | 3 ops/s | 2.1x faster |
| FileCount/Delete/10000 | **liteio** | 1 ops/s | minio | 0 ops/s | +95% |
| FileCount/List/1 | **liteio** | 3.3K ops/s | minio | 1.9K ops/s | +75% |
| FileCount/List/10 | **liteio** | 3.1K ops/s | minio | 1.8K ops/s | +75% |
| FileCount/List/100 | **liteio** | 1.2K ops/s | minio | 572 ops/s | 2.1x faster |
| FileCount/List/1000 | **liteio** | 204 ops/s | minio | 74 ops/s | 2.8x faster |
| FileCount/List/10000 | **minio** | 6 ops/s | liteio | 6 ops/s | ~equal |
| FileCount/Write/1 | **liteio** | 2.2 MB/s | rustfs | 1.2 MB/s | +84% |
| FileCount/Write/10 | **liteio** | 3.9 MB/s | rustfs | 1.7 MB/s | 2.4x faster |
| FileCount/Write/100 | **liteio** | 4.1 MB/s | rustfs | 1.4 MB/s | 2.9x faster |
| FileCount/Write/1000 | **liteio** | 4.5 MB/s | rustfs | 1.4 MB/s | 3.3x faster |
| FileCount/Write/10000 | **liteio** | 4.5 MB/s | rustfs | 1.3 MB/s | 3.5x faster |
| List/100 | **liteio** | 1.3K ops/s | minio | 529 ops/s | 2.5x faster |
| MixedWorkload/Balanced_50_50 | **liteio** | 5.9 MB/s | minio | 3.6 MB/s | +63% |
| MixedWorkload/ReadHeavy_90_10 | **liteio** | 3.7 MB/s | minio | 2.6 MB/s | +44% |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 3.0 MB/s | minio | 1.3 MB/s | 2.3x faster |
| Multipart/15MB_3Parts | **rustfs** | 173.5 MB/s | minio | 144.2 MB/s | +20% |
| ParallelRead/1KB/C1 | **liteio** | 3.8 MB/s | minio | 2.8 MB/s | +39% |
| ParallelRead/1KB/C10 | **liteio** | 1.3 MB/s | minio | 1.1 MB/s | +14% |
| ParallelRead/1KB/C100 | **liteio** | 0.6 MB/s | minio | 0.3 MB/s | +88% |
| ParallelRead/1KB/C200 | **liteio** | 0.5 MB/s | minio | 0.3 MB/s | +57% |
| ParallelRead/1KB/C25 | **liteio** | 0.9 MB/s | minio | 0.4 MB/s | 2.2x faster |
| ParallelRead/1KB/C50 | **liteio** | 0.5 MB/s | minio | 0.4 MB/s | +40% |
| ParallelWrite/1KB/C1 | **liteio** | 2.9 MB/s | rustfs | 1.4 MB/s | 2.1x faster |
| ParallelWrite/1KB/C10 | **liteio** | 0.5 MB/s | rustfs | 0.3 MB/s | +55% |
| ParallelWrite/1KB/C100 | **liteio** | 0.2 MB/s | rustfs | 0.1 MB/s | +69% |
| ParallelWrite/1KB/C200 | **liteio** | 0.2 MB/s | rustfs | 0.1 MB/s | 2.0x faster |
| ParallelWrite/1KB/C25 | **liteio** | 0.4 MB/s | rustfs | 0.2 MB/s | 2.5x faster |
| ParallelWrite/1KB/C50 | **liteio** | 0.2 MB/s | rustfs | 0.1 MB/s | 2.4x faster |
| RangeRead/End_256KB | **liteio** | 235.5 MB/s | minio | 156.8 MB/s | +50% |
| RangeRead/Middle_256KB | **liteio** | 235.3 MB/s | minio | 151.1 MB/s | +56% |
| RangeRead/Start_256KB | **liteio** | 206.7 MB/s | minio | 162.6 MB/s | +27% |
| Read/100MB | **minio** | 271.3 MB/s | liteio | 235.2 MB/s | +15% |
| Read/10MB | **minio** | 262.2 MB/s | liteio | 229.7 MB/s | +14% |
| Read/1KB | **liteio** | 4.7 MB/s | minio | 2.7 MB/s | +73% |
| Read/1MB | **liteio** | 272.4 MB/s | minio | 221.3 MB/s | +23% |
| Read/64KB | **liteio** | 147.0 MB/s | minio | 88.0 MB/s | +67% |
| Stat | **liteio** | 5.7K ops/s | rustfs | 3.7K ops/s | +54% |
| Write/100MB | **liteio** | 200.0 MB/s | minio | 149.0 MB/s | +34% |
| Write/10MB | **liteio** | 205.9 MB/s | rustfs | 173.9 MB/s | +18% |
| Write/1KB | **liteio** | 1.5 MB/s | rustfs | 1.3 MB/s | +16% |
| Write/1MB | **liteio** | 188.8 MB/s | rustfs | 156.1 MB/s | +21% |
| Write/64KB | **liteio** | 126.9 MB/s | rustfs | 45.0 MB/s | 2.8x faster |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 5/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | liteio | 200.0 MB/s | +34% |
| Write/10MB | liteio | 205.9 MB/s | +18% |
| Write/1KB | liteio | 1.5 MB/s | +16% |
| Write/1MB | liteio | 188.8 MB/s | +21% |
| Write/64KB | liteio | 126.9 MB/s | 2.8x faster |

### Read Operations

**Best for Read:** liteio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 271.3 MB/s | +15% |
| Read/10MB | minio | 262.2 MB/s | +14% |
| Read/1KB | liteio | 4.7 MB/s | +73% |
| Read/1MB | liteio | 272.4 MB/s | +23% |
| Read/64KB | liteio | 147.0 MB/s | +67% |

### ParallelWrite Operations

**Best for ParallelWrite:** liteio (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio | 2.9 MB/s | 2.1x faster |
| ParallelWrite/1KB/C10 | liteio | 0.5 MB/s | +55% |
| ParallelWrite/1KB/C100 | liteio | 0.2 MB/s | +69% |
| ParallelWrite/1KB/C200 | liteio | 0.2 MB/s | 2.0x faster |
| ParallelWrite/1KB/C25 | liteio | 0.4 MB/s | 2.5x faster |
| ParallelWrite/1KB/C50 | liteio | 0.2 MB/s | 2.4x faster |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 3.8 MB/s | +39% |
| ParallelRead/1KB/C10 | liteio | 1.3 MB/s | +14% |
| ParallelRead/1KB/C100 | liteio | 0.6 MB/s | +88% |
| ParallelRead/1KB/C200 | liteio | 0.5 MB/s | +57% |
| ParallelRead/1KB/C25 | liteio | 0.9 MB/s | 2.2x faster |
| ParallelRead/1KB/C50 | liteio | 0.5 MB/s | +40% |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 5.0K ops/s | +68% |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 5.7K ops/s | +54% |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 1.3K ops/s | 2.5x faster |

### Copy Operations

**Best for Copy:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio | 2.7 MB/s | 3.0x faster |

### FileCount Operations

**Best for FileCount:** liteio (won 14/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | liteio | 5.1K ops/s | 2.0x faster |
| FileCount/Delete/10 | liteio | 510 ops/s | +81% |
| FileCount/Delete/100 | liteio | 53 ops/s | +85% |
| FileCount/Delete/1000 | liteio | 6 ops/s | 2.1x faster |
| FileCount/Delete/10000 | liteio | 1 ops/s | +95% |
| FileCount/List/1 | liteio | 3.3K ops/s | +75% |
| FileCount/List/10 | liteio | 3.1K ops/s | +75% |
| FileCount/List/100 | liteio | 1.2K ops/s | 2.1x faster |
| FileCount/List/1000 | liteio | 204 ops/s | 2.8x faster |
| FileCount/List/10000 | minio | 6 ops/s | ~equal |
| FileCount/Write/1 | liteio | 2.2 MB/s | +84% |
| FileCount/Write/10 | liteio | 3.9 MB/s | 2.4x faster |
| FileCount/Write/100 | liteio | 4.1 MB/s | 2.9x faster |
| FileCount/Write/1000 | liteio | 4.5 MB/s | 3.3x faster |
| FileCount/Write/10000 | liteio | 4.5 MB/s | 3.5x faster |

---

*Generated by storage benchmark CLI*
