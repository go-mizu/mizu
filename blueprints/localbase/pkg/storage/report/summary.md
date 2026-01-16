# Storage Benchmark Summary

**Generated:** 2026-01-16T01:42:32+07:00

## Overall Winner

**liteio** won 33/51 categories (65%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| liteio | 33 | 65% |
| minio | 14 | 27% |
| rustfs | 4 | 8% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **minio** | 1.0 MB/s | rustfs | 0.9 MB/s | +11% |
| Delete | **liteio** | 3.1K ops/s | minio | 2.7K ops/s | +16% |
| EdgeCase/DeepNested | **liteio** | 0.3 MB/s | rustfs | 0.1 MB/s | 2.3x faster |
| EdgeCase/EmptyObject | **liteio** | 3.0K ops/s | rustfs | 1.2K ops/s | 2.6x faster |
| EdgeCase/LongKey256 | **liteio** | 0.3 MB/s | rustfs | 0.1 MB/s | 2.3x faster |
| FileCount/Delete/1 | **liteio** | 3.3K ops/s | minio | 1.3K ops/s | 2.5x faster |
| FileCount/Delete/10 | **liteio** | 487 ops/s | minio | 180 ops/s | 2.7x faster |
| FileCount/Delete/100 | **liteio** | 40 ops/s | minio | 25 ops/s | +58% |
| FileCount/Delete/1000 | **liteio** | 3 ops/s | minio | 2 ops/s | 2.1x faster |
| FileCount/Delete/10000 | **liteio** | 0 ops/s | minio | 0 ops/s | +41% |
| FileCount/List/1 | **liteio** | 3.7K ops/s | minio | 1.0K ops/s | 3.6x faster |
| FileCount/List/10 | **liteio** | 2.5K ops/s | minio | 868 ops/s | 2.9x faster |
| FileCount/List/100 | **liteio** | 916 ops/s | minio | 477 ops/s | +92% |
| FileCount/List/1000 | **liteio** | 123 ops/s | minio | 72 ops/s | +71% |
| FileCount/List/10000 | **minio** | 6 ops/s | liteio | 4 ops/s | +38% |
| FileCount/Write/1 | **liteio** | 1.9 MB/s | rustfs | 0.6 MB/s | 2.9x faster |
| FileCount/Write/10 | **liteio** | 3.5 MB/s | rustfs | 0.9 MB/s | 4.0x faster |
| FileCount/Write/100 | **liteio** | 3.6 MB/s | rustfs | 1.3 MB/s | 2.8x faster |
| FileCount/Write/1000 | **liteio** | 3.3 MB/s | rustfs | 1.3 MB/s | 2.6x faster |
| FileCount/Write/10000 | **liteio** | 2.9 MB/s | rustfs | 1.0 MB/s | 3.0x faster |
| List/100 | **liteio** | 757 ops/s | minio | 582 ops/s | +30% |
| MixedWorkload/Balanced_50_50 | **liteio** | 0.4 MB/s | minio | 0.4 MB/s | ~equal |
| MixedWorkload/ReadHeavy_90_10 | **minio** | 0.6 MB/s | liteio | 0.6 MB/s | ~equal |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 0.3 MB/s | minio | 0.2 MB/s | +27% |
| Multipart/15MB_3Parts | **rustfs** | 135.0 MB/s | minio | 128.1 MB/s | ~equal |
| ParallelRead/1KB/C1 | **minio** | 2.7 MB/s | rustfs | 1.8 MB/s | +51% |
| ParallelRead/1KB/C10 | **minio** | 1.0 MB/s | liteio | 0.9 MB/s | ~equal |
| ParallelRead/1KB/C100 | **liteio** | 0.2 MB/s | minio | 0.1 MB/s | +10% |
| ParallelRead/1KB/C200 | **liteio** | 0.1 MB/s | minio | 0.1 MB/s | +12% |
| ParallelRead/1KB/C25 | **liteio** | 0.5 MB/s | minio | 0.5 MB/s | ~equal |
| ParallelRead/1KB/C50 | **minio** | 0.3 MB/s | rustfs | 0.2 MB/s | +60% |
| ParallelWrite/1KB/C1 | **liteio** | 1.9 MB/s | rustfs | 1.1 MB/s | +71% |
| ParallelWrite/1KB/C10 | **liteio** | 0.4 MB/s | rustfs | 0.3 MB/s | +20% |
| ParallelWrite/1KB/C100 | **liteio** | 0.0 MB/s | rustfs | 0.0 MB/s | +33% |
| ParallelWrite/1KB/C200 | **rustfs** | 0.0 MB/s | minio | 0.0 MB/s | ~equal |
| ParallelWrite/1KB/C25 | **liteio** | 0.2 MB/s | rustfs | 0.1 MB/s | +84% |
| ParallelWrite/1KB/C50 | **liteio** | 0.1 MB/s | rustfs | 0.1 MB/s | +65% |
| RangeRead/End_256KB | **minio** | 161.0 MB/s | liteio | 127.7 MB/s | +26% |
| RangeRead/Middle_256KB | **minio** | 161.4 MB/s | liteio | 126.2 MB/s | +28% |
| RangeRead/Start_256KB | **minio** | 155.1 MB/s | liteio | 131.1 MB/s | +18% |
| Read/100MB | **minio** | 256.2 MB/s | rustfs | 173.7 MB/s | +47% |
| Read/10MB | **minio** | 256.5 MB/s | rustfs | 201.5 MB/s | +27% |
| Read/1KB | **liteio** | 4.1 MB/s | minio | 2.9 MB/s | +39% |
| Read/1MB | **minio** | 227.1 MB/s | rustfs | 184.2 MB/s | +23% |
| Read/64KB | **minio** | 115.7 MB/s | liteio | 98.8 MB/s | +17% |
| Stat | **liteio** | 4.0K ops/s | minio | 3.7K ops/s | ~equal |
| Write/100MB | **minio** | 159.1 MB/s | liteio | 140.5 MB/s | +13% |
| Write/10MB | **rustfs** | 163.2 MB/s | minio | 161.1 MB/s | ~equal |
| Write/1KB | **liteio** | 3.2 MB/s | rustfs | 1.3 MB/s | 2.6x faster |
| Write/1MB | **rustfs** | 147.6 MB/s | minio | 125.3 MB/s | +18% |
| Write/64KB | **liteio** | 87.5 MB/s | rustfs | 56.5 MB/s | +55% |

## Category Summaries

### Write Operations

**Best for Write:** rustfs (won 2/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | minio | 159.1 MB/s | +13% |
| Write/10MB | rustfs | 163.2 MB/s | ~equal |
| Write/1KB | liteio | 3.2 MB/s | 2.6x faster |
| Write/1MB | rustfs | 147.6 MB/s | +18% |
| Write/64KB | liteio | 87.5 MB/s | +55% |

### Read Operations

**Best for Read:** minio (won 4/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 256.2 MB/s | +47% |
| Read/10MB | minio | 256.5 MB/s | +27% |
| Read/1KB | liteio | 4.1 MB/s | +39% |
| Read/1MB | minio | 227.1 MB/s | +23% |
| Read/64KB | minio | 115.7 MB/s | +17% |

### ParallelWrite Operations

**Best for ParallelWrite:** liteio (won 5/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio | 1.9 MB/s | +71% |
| ParallelWrite/1KB/C10 | liteio | 0.4 MB/s | +20% |
| ParallelWrite/1KB/C100 | liteio | 0.0 MB/s | +33% |
| ParallelWrite/1KB/C200 | rustfs | 0.0 MB/s | ~equal |
| ParallelWrite/1KB/C25 | liteio | 0.2 MB/s | +84% |
| ParallelWrite/1KB/C50 | liteio | 0.1 MB/s | +65% |

### ParallelRead Operations

**Best for ParallelRead:** minio (won 3/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | minio | 2.7 MB/s | +51% |
| ParallelRead/1KB/C10 | minio | 1.0 MB/s | ~equal |
| ParallelRead/1KB/C100 | liteio | 0.2 MB/s | +10% |
| ParallelRead/1KB/C200 | liteio | 0.1 MB/s | +12% |
| ParallelRead/1KB/C25 | liteio | 0.5 MB/s | ~equal |
| ParallelRead/1KB/C50 | minio | 0.3 MB/s | +60% |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 3.1K ops/s | +16% |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 4.0K ops/s | ~equal |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 757 ops/s | +30% |

### Copy Operations

**Best for Copy:** minio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | minio | 1.0 MB/s | +11% |

### FileCount Operations

**Best for FileCount:** liteio (won 14/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | liteio | 3.3K ops/s | 2.5x faster |
| FileCount/Delete/10 | liteio | 487 ops/s | 2.7x faster |
| FileCount/Delete/100 | liteio | 40 ops/s | +58% |
| FileCount/Delete/1000 | liteio | 3 ops/s | 2.1x faster |
| FileCount/Delete/10000 | liteio | 0 ops/s | +41% |
| FileCount/List/1 | liteio | 3.7K ops/s | 3.6x faster |
| FileCount/List/10 | liteio | 2.5K ops/s | 2.9x faster |
| FileCount/List/100 | liteio | 916 ops/s | +92% |
| FileCount/List/1000 | liteio | 123 ops/s | +71% |
| FileCount/List/10000 | minio | 6 ops/s | +38% |
| FileCount/Write/1 | liteio | 1.9 MB/s | 2.9x faster |
| FileCount/Write/10 | liteio | 3.5 MB/s | 4.0x faster |
| FileCount/Write/100 | liteio | 3.6 MB/s | 2.8x faster |
| FileCount/Write/1000 | liteio | 3.3 MB/s | 2.6x faster |
| FileCount/Write/10000 | liteio | 2.9 MB/s | 3.0x faster |

---

*Generated by storage benchmark CLI*
