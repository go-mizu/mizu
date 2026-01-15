# Storage Benchmark Summary

**Generated:** 2026-01-15T11:35:12+07:00

## Overall Winner

**liteio** won 33/43 categories (77%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| liteio | 33 | 77% |
| minio | 10 | 23% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **minio** | 1.3 MB/s | liteio | 0.4 MB/s | 3.6x faster |
| Delete | **liteio** | 5.3K ops/s | minio | 3.2K ops/s | +62% |
| EdgeCase/DeepNested | **minio** | 0.1 MB/s | liteio | 0.1 MB/s | +24% |
| EdgeCase/EmptyObject | **minio** | 1.4K ops/s | liteio | 937 ops/s | +53% |
| EdgeCase/LongKey256 | **liteio** | 0.2 MB/s | minio | 0.1 MB/s | 2.8x faster |
| FileCount/Delete/1 | **liteio** | 4.7K ops/s | minio | 2.7K ops/s | +74% |
| FileCount/Delete/10 | **liteio** | 435 ops/s | minio | 308 ops/s | +41% |
| FileCount/Delete/100 | **liteio** | 46 ops/s | minio | 28 ops/s | +66% |
| FileCount/Delete/1000 | **liteio** | 5 ops/s | minio | 3 ops/s | +82% |
| FileCount/Delete/10000 | **liteio** | 1 ops/s | minio | 0 ops/s | +89% |
| FileCount/List/1 | **liteio** | 3.6K ops/s | minio | 2.6K ops/s | +40% |
| FileCount/List/10 | **liteio** | 2.7K ops/s | minio | 1.7K ops/s | +57% |
| FileCount/List/100 | **liteio** | 1.0K ops/s | minio | 549 ops/s | +87% |
| FileCount/List/1000 | **liteio** | 193 ops/s | minio | 64 ops/s | 3.0x faster |
| FileCount/List/10000 | **minio** | 6 ops/s | liteio | 5 ops/s | +15% |
| FileCount/Write/1 | **liteio** | 1.7 MB/s | minio | 1.5 MB/s | +16% |
| FileCount/Write/10 | **liteio** | 2.1 MB/s | minio | 1.4 MB/s | +48% |
| FileCount/Write/100 | **liteio** | 2.0 MB/s | minio | 1.3 MB/s | +46% |
| FileCount/Write/1000 | **liteio** | 2.0 MB/s | minio | 1.1 MB/s | +75% |
| FileCount/Write/10000 | **liteio** | 1.9 MB/s | minio | 1.2 MB/s | +65% |
| List/100 | **liteio** | 1.3K ops/s | minio | 520 ops/s | 2.6x faster |
| MixedWorkload/Balanced_50_50 | **minio** | 8.8 MB/s | liteio | 6.5 MB/s | +35% |
| MixedWorkload/ReadHeavy_90_10 | **liteio** | 6.6 MB/s | minio | 6.6 MB/s | ~equal |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 5.1 MB/s | minio | 4.1 MB/s | +26% |
| Multipart/15MB_3Parts | **minio** | 159.0 MB/s | liteio | 150.6 MB/s | ~equal |
| ParallelRead/1KB/C1 | **liteio** | 4.7 MB/s | minio | 2.9 MB/s | +63% |
| ParallelRead/1KB/C10 | **minio** | 0.8 MB/s | liteio | 0.5 MB/s | +75% |
| ParallelRead/1KB/C50 | **liteio** | 1.0 MB/s | minio | 0.5 MB/s | +97% |
| ParallelWrite/1KB/C1 | **liteio** | 1.7 MB/s | minio | 1.4 MB/s | +23% |
| ParallelWrite/1KB/C10 | **liteio** | 0.4 MB/s | minio | 0.3 MB/s | +47% |
| ParallelWrite/1KB/C50 | **liteio** | 0.2 MB/s | minio | 0.2 MB/s | +24% |
| RangeRead/End_256KB | **liteio** | 184.9 MB/s | minio | 166.5 MB/s | +11% |
| RangeRead/Middle_256KB | **liteio** | 140.8 MB/s | minio | 96.2 MB/s | +46% |
| RangeRead/Start_256KB | **minio** | 131.4 MB/s | liteio | 126.9 MB/s | ~equal |
| Read/10MB | **minio** | 298.3 MB/s | liteio | 291.3 MB/s | ~equal |
| Read/1KB | **liteio** | 5.2 MB/s | minio | 3.3 MB/s | +58% |
| Read/1MB | **liteio** | 279.2 MB/s | minio | 243.2 MB/s | +15% |
| Read/64KB | **liteio** | 137.4 MB/s | minio | 95.5 MB/s | +44% |
| Stat | **liteio** | 5.7K ops/s | minio | 4.0K ops/s | +43% |
| Write/10MB | **liteio** | 187.0 MB/s | minio | 161.7 MB/s | +16% |
| Write/1KB | **liteio** | 1.4 MB/s | minio | 1.3 MB/s | ~equal |
| Write/1MB | **liteio** | 160.4 MB/s | minio | 138.3 MB/s | +16% |
| Write/64KB | **minio** | 59.5 MB/s | liteio | 59.2 MB/s | ~equal |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 3/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | liteio | 187.0 MB/s | +16% |
| Write/1KB | liteio | 1.4 MB/s | ~equal |
| Write/1MB | liteio | 160.4 MB/s | +16% |
| Write/64KB | minio | 59.5 MB/s | ~equal |

### Read Operations

**Best for Read:** liteio (won 3/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | minio | 298.3 MB/s | ~equal |
| Read/1KB | liteio | 5.2 MB/s | +58% |
| Read/1MB | liteio | 279.2 MB/s | +15% |
| Read/64KB | liteio | 137.4 MB/s | +44% |

### ParallelWrite Operations

**Best for ParallelWrite:** liteio (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio | 1.7 MB/s | +23% |
| ParallelWrite/1KB/C10 | liteio | 0.4 MB/s | +47% |
| ParallelWrite/1KB/C50 | liteio | 0.2 MB/s | +24% |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 2/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 4.7 MB/s | +63% |
| ParallelRead/1KB/C10 | minio | 0.8 MB/s | +75% |
| ParallelRead/1KB/C50 | liteio | 1.0 MB/s | +97% |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 5.3K ops/s | +62% |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 5.7K ops/s | +43% |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 1.3K ops/s | 2.6x faster |

### Copy Operations

**Best for Copy:** minio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | minio | 1.3 MB/s | 3.6x faster |

### FileCount Operations

**Best for FileCount:** liteio (won 14/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | liteio | 4.7K ops/s | +74% |
| FileCount/Delete/10 | liteio | 435 ops/s | +41% |
| FileCount/Delete/100 | liteio | 46 ops/s | +66% |
| FileCount/Delete/1000 | liteio | 5 ops/s | +82% |
| FileCount/Delete/10000 | liteio | 1 ops/s | +89% |
| FileCount/List/1 | liteio | 3.6K ops/s | +40% |
| FileCount/List/10 | liteio | 2.7K ops/s | +57% |
| FileCount/List/100 | liteio | 1.0K ops/s | +87% |
| FileCount/List/1000 | liteio | 193 ops/s | 3.0x faster |
| FileCount/List/10000 | minio | 6 ops/s | +15% |
| FileCount/Write/1 | liteio | 1.7 MB/s | +16% |
| FileCount/Write/10 | liteio | 2.1 MB/s | +48% |
| FileCount/Write/100 | liteio | 2.0 MB/s | +46% |
| FileCount/Write/1000 | liteio | 2.0 MB/s | +75% |
| FileCount/Write/10000 | liteio | 1.9 MB/s | +65% |

---

*Generated by storage benchmark CLI*
