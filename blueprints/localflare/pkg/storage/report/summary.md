# Storage Benchmark Summary

**Generated:** 2026-01-15T22:07:16+07:00

## Overall Winner

**liteio** won 35/51 categories (69%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| liteio | 35 | 69% |
| rustfs | 10 | 20% |
| minio | 6 | 12% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **liteio** | 1.3 MB/s | rustfs | 1.0 MB/s | +31% |
| Delete | **liteio** | 6.1K ops/s | minio | 2.6K ops/s | 2.4x faster |
| EdgeCase/DeepNested | **liteio** | 0.1 MB/s | rustfs | 0.1 MB/s | +19% |
| EdgeCase/EmptyObject | **rustfs** | 1.1K ops/s | minio | 1.0K ops/s | ~equal |
| EdgeCase/LongKey256 | **liteio** | 0.1 MB/s | rustfs | 0.1 MB/s | +16% |
| FileCount/Delete/1 | **liteio** | 5.1K ops/s | minio | 1.7K ops/s | 3.1x faster |
| FileCount/Delete/10 | **liteio** | 645 ops/s | minio | 248 ops/s | 2.6x faster |
| FileCount/Delete/100 | **liteio** | 49 ops/s | minio | 24 ops/s | 2.1x faster |
| FileCount/Delete/1000 | **liteio** | 6 ops/s | minio | 3 ops/s | 2.1x faster |
| FileCount/Delete/10000 | **liteio** | 1 ops/s | minio | 0 ops/s | 2.1x faster |
| FileCount/List/1 | **liteio** | 3.1K ops/s | minio | 1.3K ops/s | 2.4x faster |
| FileCount/List/10 | **liteio** | 1.7K ops/s | minio | 1.0K ops/s | +58% |
| FileCount/List/100 | **liteio** | 1.2K ops/s | minio | 372 ops/s | 3.3x faster |
| FileCount/List/1000 | **liteio** | 92 ops/s | minio | 50 ops/s | +83% |
| FileCount/List/10000 | **minio** | 5 ops/s | liteio | 5 ops/s | ~equal |
| FileCount/Write/1 | **rustfs** | 0.9 MB/s | liteio | 0.8 MB/s | +23% |
| FileCount/Write/10 | **liteio** | 1.5 MB/s | rustfs | 1.1 MB/s | +43% |
| FileCount/Write/100 | **liteio** | 1.5 MB/s | rustfs | 1.2 MB/s | +24% |
| FileCount/Write/1000 | **liteio** | 1.5 MB/s | rustfs | 1.3 MB/s | +16% |
| FileCount/Write/10000 | **liteio** | 1.4 MB/s | rustfs | 1.3 MB/s | ~equal |
| List/100 | **liteio** | 1.3K ops/s | minio | 654 ops/s | +99% |
| MixedWorkload/Balanced_50_50 | **minio** | 2.1 MB/s | rustfs | 1.8 MB/s | +16% |
| MixedWorkload/ReadHeavy_90_10 | **rustfs** | 2.6 MB/s | minio | 2.0 MB/s | +25% |
| MixedWorkload/WriteHeavy_10_90 | **rustfs** | 1.1 MB/s | minio | 1.0 MB/s | +12% |
| Multipart/15MB_3Parts | **liteio** | 155.3 MB/s | rustfs | 155.2 MB/s | ~equal |
| ParallelRead/1KB/C1 | **liteio** | 3.1 MB/s | minio | 2.1 MB/s | +47% |
| ParallelRead/1KB/C10 | **minio** | 1.4 MB/s | rustfs | 1.1 MB/s | +27% |
| ParallelRead/1KB/C100 | **rustfs** | 0.4 MB/s | liteio | 0.3 MB/s | ~equal |
| ParallelRead/1KB/C200 | **rustfs** | 0.7 MB/s | minio | 0.6 MB/s | +21% |
| ParallelRead/1KB/C25 | **minio** | 0.8 MB/s | rustfs | 0.7 MB/s | +13% |
| ParallelRead/1KB/C50 | **minio** | 0.5 MB/s | liteio | 0.5 MB/s | ~equal |
| ParallelWrite/1KB/C1 | **liteio** | 1.3 MB/s | rustfs | 1.1 MB/s | +14% |
| ParallelWrite/1KB/C10 | **liteio** | 0.4 MB/s | minio | 0.4 MB/s | ~equal |
| ParallelWrite/1KB/C100 | **rustfs** | 0.1 MB/s | liteio | 0.1 MB/s | +67% |
| ParallelWrite/1KB/C200 | **rustfs** | 0.1 MB/s | liteio | 0.1 MB/s | +56% |
| ParallelWrite/1KB/C25 | **liteio** | 0.2 MB/s | minio | 0.2 MB/s | +19% |
| ParallelWrite/1KB/C50 | **liteio** | 0.2 MB/s | rustfs | 0.1 MB/s | +88% |
| RangeRead/End_256KB | **liteio** | 248.9 MB/s | minio | 158.0 MB/s | +57% |
| RangeRead/Middle_256KB | **liteio** | 241.9 MB/s | minio | 168.7 MB/s | +43% |
| RangeRead/Start_256KB | **liteio** | 241.9 MB/s | minio | 158.7 MB/s | +52% |
| Read/100MB | **minio** | 327.2 MB/s | liteio | 299.6 MB/s | ~equal |
| Read/10MB | **liteio** | 328.8 MB/s | minio | 302.4 MB/s | ~equal |
| Read/1KB | **liteio** | 3.9 MB/s | minio | 2.1 MB/s | +85% |
| Read/1MB | **liteio** | 309.1 MB/s | minio | 248.5 MB/s | +24% |
| Read/64KB | **liteio** | 157.4 MB/s | rustfs | 106.3 MB/s | +48% |
| Stat | **liteio** | 5.0K ops/s | minio | 4.2K ops/s | +20% |
| Write/100MB | **liteio** | 181.9 MB/s | rustfs | 179.9 MB/s | ~equal |
| Write/10MB | **rustfs** | 193.5 MB/s | liteio | 175.4 MB/s | +10% |
| Write/1KB | **liteio** | 1.0 MB/s | minio | 1.0 MB/s | ~equal |
| Write/1MB | **rustfs** | 163.1 MB/s | minio | 128.6 MB/s | +27% |
| Write/64KB | **liteio** | 53.3 MB/s | rustfs | 52.0 MB/s | ~equal |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | liteio | 181.9 MB/s | ~equal |
| Write/10MB | rustfs | 193.5 MB/s | +10% |
| Write/1KB | liteio | 1.0 MB/s | ~equal |
| Write/1MB | rustfs | 163.1 MB/s | +27% |
| Write/64KB | liteio | 53.3 MB/s | ~equal |

### Read Operations

**Best for Read:** liteio (won 4/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 327.2 MB/s | ~equal |
| Read/10MB | liteio | 328.8 MB/s | ~equal |
| Read/1KB | liteio | 3.9 MB/s | +85% |
| Read/1MB | liteio | 309.1 MB/s | +24% |
| Read/64KB | liteio | 157.4 MB/s | +48% |

### ParallelWrite Operations

**Best for ParallelWrite:** liteio (won 4/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio | 1.3 MB/s | +14% |
| ParallelWrite/1KB/C10 | liteio | 0.4 MB/s | ~equal |
| ParallelWrite/1KB/C100 | rustfs | 0.1 MB/s | +67% |
| ParallelWrite/1KB/C200 | rustfs | 0.1 MB/s | +56% |
| ParallelWrite/1KB/C25 | liteio | 0.2 MB/s | +19% |
| ParallelWrite/1KB/C50 | liteio | 0.2 MB/s | +88% |

### ParallelRead Operations

**Best for ParallelRead:** minio (won 3/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 3.1 MB/s | +47% |
| ParallelRead/1KB/C10 | minio | 1.4 MB/s | +27% |
| ParallelRead/1KB/C100 | rustfs | 0.4 MB/s | ~equal |
| ParallelRead/1KB/C200 | rustfs | 0.7 MB/s | +21% |
| ParallelRead/1KB/C25 | minio | 0.8 MB/s | +13% |
| ParallelRead/1KB/C50 | minio | 0.5 MB/s | ~equal |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 6.1K ops/s | 2.4x faster |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 5.0K ops/s | +20% |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 1.3K ops/s | +99% |

### Copy Operations

**Best for Copy:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio | 1.3 MB/s | +31% |

### FileCount Operations

**Best for FileCount:** liteio (won 13/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | liteio | 5.1K ops/s | 3.1x faster |
| FileCount/Delete/10 | liteio | 645 ops/s | 2.6x faster |
| FileCount/Delete/100 | liteio | 49 ops/s | 2.1x faster |
| FileCount/Delete/1000 | liteio | 6 ops/s | 2.1x faster |
| FileCount/Delete/10000 | liteio | 1 ops/s | 2.1x faster |
| FileCount/List/1 | liteio | 3.1K ops/s | 2.4x faster |
| FileCount/List/10 | liteio | 1.7K ops/s | +58% |
| FileCount/List/100 | liteio | 1.2K ops/s | 3.3x faster |
| FileCount/List/1000 | liteio | 92 ops/s | +83% |
| FileCount/List/10000 | minio | 5 ops/s | ~equal |
| FileCount/Write/1 | rustfs | 0.9 MB/s | +23% |
| FileCount/Write/10 | liteio | 1.5 MB/s | +43% |
| FileCount/Write/100 | liteio | 1.5 MB/s | +24% |
| FileCount/Write/1000 | liteio | 1.5 MB/s | +16% |
| FileCount/Write/10000 | liteio | 1.4 MB/s | ~equal |

---

*Generated by storage benchmark CLI*
