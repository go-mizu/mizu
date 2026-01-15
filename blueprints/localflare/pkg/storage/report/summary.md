# Storage Benchmark Summary

**Generated:** 2026-01-16T01:11:15+07:00

## Overall Winner

**liteio** won 43/51 categories (84%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| liteio | 43 | 84% |
| rustfs | 6 | 12% |
| minio | 2 | 4% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **liteio** | 2.0 MB/s | rustfs | 0.9 MB/s | 2.3x faster |
| Delete | **liteio** | 5.5K ops/s | minio | 1.9K ops/s | 2.9x faster |
| EdgeCase/DeepNested | **liteio** | 0.5 MB/s | rustfs | 0.1 MB/s | 4.1x faster |
| EdgeCase/EmptyObject | **liteio** | 2.9K ops/s | rustfs | 1.1K ops/s | 2.5x faster |
| EdgeCase/LongKey256 | **liteio** | 0.3 MB/s | rustfs | 0.1 MB/s | 3.0x faster |
| FileCount/Delete/1 | **liteio** | 5.2K ops/s | minio | 2.4K ops/s | 2.2x faster |
| FileCount/Delete/10 | **liteio** | 605 ops/s | minio | 215 ops/s | 2.8x faster |
| FileCount/Delete/100 | **liteio** | 62 ops/s | minio | 20 ops/s | 3.1x faster |
| FileCount/Delete/1000 | **liteio** | 6 ops/s | minio | 3 ops/s | 2.2x faster |
| FileCount/Delete/10000 | **liteio** | 1 ops/s | minio | 0 ops/s | 2.5x faster |
| FileCount/List/1 | **liteio** | 2.6K ops/s | minio | 1.0K ops/s | 2.5x faster |
| FileCount/List/10 | **liteio** | 3.4K ops/s | minio | 1.4K ops/s | 2.4x faster |
| FileCount/List/100 | **liteio** | 1.2K ops/s | minio | 330 ops/s | 3.8x faster |
| FileCount/List/1000 | **liteio** | 206 ops/s | minio | 71 ops/s | 2.9x faster |
| FileCount/List/10000 | **liteio** | 6 ops/s | minio | 5 ops/s | +11% |
| FileCount/Write/1 | **liteio** | 1.4 MB/s | rustfs | 0.7 MB/s | 2.0x faster |
| FileCount/Write/10 | **liteio** | 4.4 MB/s | rustfs | 1.1 MB/s | 3.9x faster |
| FileCount/Write/100 | **liteio** | 5.2 MB/s | minio | 1.0 MB/s | 5.1x faster |
| FileCount/Write/1000 | **liteio** | 5.2 MB/s | rustfs | 1.1 MB/s | 4.6x faster |
| FileCount/Write/10000 | **liteio** | 4.9 MB/s | rustfs | 1.1 MB/s | 4.3x faster |
| List/100 | **liteio** | 1.2K ops/s | minio | 618 ops/s | 2.0x faster |
| MixedWorkload/Balanced_50_50 | **rustfs** | 2.4 MB/s | minio | 2.2 MB/s | ~equal |
| MixedWorkload/ReadHeavy_90_10 | **minio** | 3.2 MB/s | rustfs | 1.3 MB/s | 2.5x faster |
| MixedWorkload/WriteHeavy_10_90 | **rustfs** | 1.3 MB/s | minio | 0.9 MB/s | +45% |
| Multipart/15MB_3Parts | **liteio** | 152.1 MB/s | rustfs | 144.4 MB/s | ~equal |
| ParallelRead/1KB/C1 | **liteio** | 4.5 MB/s | minio | 1.4 MB/s | 3.1x faster |
| ParallelRead/1KB/C10 | **liteio** | 1.7 MB/s | rustfs | 1.1 MB/s | +62% |
| ParallelRead/1KB/C100 | **rustfs** | 0.6 MB/s | minio | 0.5 MB/s | +25% |
| ParallelRead/1KB/C200 | **liteio** | 0.4 MB/s | rustfs | 0.4 MB/s | ~equal |
| ParallelRead/1KB/C25 | **liteio** | 0.9 MB/s | minio | 0.8 MB/s | +14% |
| ParallelRead/1KB/C50 | **liteio** | 0.5 MB/s | minio | 0.4 MB/s | +29% |
| ParallelWrite/1KB/C1 | **liteio** | 2.7 MB/s | minio | 0.8 MB/s | 3.5x faster |
| ParallelWrite/1KB/C10 | **liteio** | 0.8 MB/s | rustfs | 0.5 MB/s | +71% |
| ParallelWrite/1KB/C100 | **liteio** | 0.4 MB/s | rustfs | 0.1 MB/s | 4.8x faster |
| ParallelWrite/1KB/C200 | **liteio** | 0.1 MB/s | minio | 0.1 MB/s | +25% |
| ParallelWrite/1KB/C25 | **liteio** | 0.4 MB/s | rustfs | 0.2 MB/s | +60% |
| ParallelWrite/1KB/C50 | **rustfs** | 0.2 MB/s | liteio | 0.2 MB/s | +12% |
| RangeRead/End_256KB | **liteio** | 253.2 MB/s | minio | 124.2 MB/s | 2.0x faster |
| RangeRead/Middle_256KB | **liteio** | 251.5 MB/s | minio | 175.5 MB/s | +43% |
| RangeRead/Start_256KB | **liteio** | 234.7 MB/s | minio | 148.1 MB/s | +58% |
| Read/100MB | **rustfs** | 330.7 MB/s | minio | 326.2 MB/s | ~equal |
| Read/10MB | **minio** | 316.2 MB/s | liteio | 302.2 MB/s | ~equal |
| Read/1KB | **liteio** | 4.0 MB/s | rustfs | 1.9 MB/s | 2.1x faster |
| Read/1MB | **liteio** | 300.1 MB/s | minio | 236.8 MB/s | +27% |
| Read/64KB | **liteio** | 146.6 MB/s | minio | 99.5 MB/s | +47% |
| Stat | **liteio** | 5.1K ops/s | minio | 3.6K ops/s | +41% |
| Write/100MB | **liteio** | 201.8 MB/s | rustfs | 168.3 MB/s | +20% |
| Write/10MB | **liteio** | 201.6 MB/s | minio | 155.7 MB/s | +29% |
| Write/1KB | **rustfs** | 1.1 MB/s | liteio | 1.0 MB/s | +11% |
| Write/1MB | **liteio** | 148.8 MB/s | rustfs | 122.8 MB/s | +21% |
| Write/64KB | **liteio** | 126.3 MB/s | rustfs | 52.9 MB/s | 2.4x faster |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 4/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | liteio | 201.8 MB/s | +20% |
| Write/10MB | liteio | 201.6 MB/s | +29% |
| Write/1KB | rustfs | 1.1 MB/s | +11% |
| Write/1MB | liteio | 148.8 MB/s | +21% |
| Write/64KB | liteio | 126.3 MB/s | 2.4x faster |

### Read Operations

**Best for Read:** liteio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | rustfs | 330.7 MB/s | ~equal |
| Read/10MB | minio | 316.2 MB/s | ~equal |
| Read/1KB | liteio | 4.0 MB/s | 2.1x faster |
| Read/1MB | liteio | 300.1 MB/s | +27% |
| Read/64KB | liteio | 146.6 MB/s | +47% |

### ParallelWrite Operations

**Best for ParallelWrite:** liteio (won 5/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio | 2.7 MB/s | 3.5x faster |
| ParallelWrite/1KB/C10 | liteio | 0.8 MB/s | +71% |
| ParallelWrite/1KB/C100 | liteio | 0.4 MB/s | 4.8x faster |
| ParallelWrite/1KB/C200 | liteio | 0.1 MB/s | +25% |
| ParallelWrite/1KB/C25 | liteio | 0.4 MB/s | +60% |
| ParallelWrite/1KB/C50 | rustfs | 0.2 MB/s | +12% |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 5/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 4.5 MB/s | 3.1x faster |
| ParallelRead/1KB/C10 | liteio | 1.7 MB/s | +62% |
| ParallelRead/1KB/C100 | rustfs | 0.6 MB/s | +25% |
| ParallelRead/1KB/C200 | liteio | 0.4 MB/s | ~equal |
| ParallelRead/1KB/C25 | liteio | 0.9 MB/s | +14% |
| ParallelRead/1KB/C50 | liteio | 0.5 MB/s | +29% |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 5.5K ops/s | 2.9x faster |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 5.1K ops/s | +41% |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 1.2K ops/s | 2.0x faster |

### Copy Operations

**Best for Copy:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio | 2.0 MB/s | 2.3x faster |

### FileCount Operations

**Best for FileCount:** liteio (won 15/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | liteio | 5.2K ops/s | 2.2x faster |
| FileCount/Delete/10 | liteio | 605 ops/s | 2.8x faster |
| FileCount/Delete/100 | liteio | 62 ops/s | 3.1x faster |
| FileCount/Delete/1000 | liteio | 6 ops/s | 2.2x faster |
| FileCount/Delete/10000 | liteio | 1 ops/s | 2.5x faster |
| FileCount/List/1 | liteio | 2.6K ops/s | 2.5x faster |
| FileCount/List/10 | liteio | 3.4K ops/s | 2.4x faster |
| FileCount/List/100 | liteio | 1.2K ops/s | 3.8x faster |
| FileCount/List/1000 | liteio | 206 ops/s | 2.9x faster |
| FileCount/List/10000 | liteio | 6 ops/s | +11% |
| FileCount/Write/1 | liteio | 1.4 MB/s | 2.0x faster |
| FileCount/Write/10 | liteio | 4.4 MB/s | 3.9x faster |
| FileCount/Write/100 | liteio | 5.2 MB/s | 5.1x faster |
| FileCount/Write/1000 | liteio | 5.2 MB/s | 4.6x faster |
| FileCount/Write/10000 | liteio | 4.9 MB/s | 4.3x faster |

---

*Generated by storage benchmark CLI*
