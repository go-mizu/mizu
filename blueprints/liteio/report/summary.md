# Storage Benchmark Summary

**Generated:** 2026-02-18T23:34:32+07:00

## Overall Winner

**liteio** won 38/40 categories (95%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| liteio | 38 | 95% |
| minio | 2 | 5% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **liteio** | 0.3 MB/s | minio | 0.2 MB/s | +68% |
| Delete | **liteio** | 1.1K ops/s | minio | 549 ops/s | 2.0x faster |
| EdgeCase/DeepNested | **liteio** | 0.0 MB/s | minio | 0.0 MB/s | 2.4x faster |
| EdgeCase/EmptyObject | **liteio** | 451 ops/s | minio | 108 ops/s | 4.2x faster |
| EdgeCase/LongKey256 | **liteio** | 0.0 MB/s | minio | 0.0 MB/s | 3.1x faster |
| List/100 | **liteio** | 367 ops/s | minio | 176 ops/s | 2.1x faster |
| MixedWorkload/Balanced_50_50 | **liteio** | 0.2 MB/s | minio | 0.1 MB/s | +90% |
| MixedWorkload/ReadHeavy_90_10 | **liteio** | 0.4 MB/s | minio | 0.2 MB/s | 2.2x faster |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 0.2 MB/s | minio | 0.1 MB/s | 3.7x faster |
| Multipart/15MB_3Parts | **liteio** | 45.5 MB/s | minio | 45.5 MB/s | ~equal |
| ParallelRead/1KB/C1 | **liteio** | 0.9 MB/s | minio | 0.6 MB/s | +35% |
| ParallelRead/1KB/C10 | **liteio** | 0.6 MB/s | minio | 0.3 MB/s | +77% |
| ParallelRead/1KB/C50 | **liteio** | 0.2 MB/s | minio | 0.1 MB/s | 3.1x faster |
| ParallelWrite/1KB/C1 | **liteio** | 0.4 MB/s | minio | 0.3 MB/s | +14% |
| ParallelWrite/1KB/C10 | **liteio** | 0.1 MB/s | minio | 0.1 MB/s | +60% |
| ParallelWrite/1KB/C50 | **liteio** | 0.0 MB/s | minio | 0.0 MB/s | 2.3x faster |
| RangeRead/End_256KB | **liteio** | 61.9 MB/s | minio | 33.5 MB/s | +85% |
| RangeRead/Middle_256KB | **liteio** | 57.3 MB/s | minio | 39.7 MB/s | +45% |
| RangeRead/Start_256KB | **liteio** | 70.2 MB/s | minio | 37.6 MB/s | +87% |
| Read/10MB | **minio** | 109.4 MB/s | liteio | 107.1 MB/s | ~equal |
| Read/1KB | **liteio** | 1.0 MB/s | minio | 0.6 MB/s | +59% |
| Read/1MB | **liteio** | 89.2 MB/s | minio | 58.2 MB/s | +53% |
| Read/64KB | **liteio** | 33.9 MB/s | minio | 26.1 MB/s | +30% |
| Scale/Delete/1 | **liteio** | 1.2K ops/s | minio | 183 ops/s | 6.7x faster |
| Scale/Delete/10 | **liteio** | 142 ops/s | minio | 68 ops/s | 2.1x faster |
| Scale/Delete/100 | **liteio** | 7 ops/s | minio | 7 ops/s | +12% |
| Scale/Delete/1000 | **liteio** | 1 ops/s | minio | 1 ops/s | +71% |
| Scale/List/1 | **liteio** | 1.2K ops/s | minio | 321 ops/s | 3.6x faster |
| Scale/List/10 | **liteio** | 1.1K ops/s | minio | 411 ops/s | 2.6x faster |
| Scale/List/100 | **liteio** | 205 ops/s | minio | 131 ops/s | +57% |
| Scale/List/1000 | **liteio** | 29 ops/s | minio | 20 ops/s | +48% |
| Scale/Write/1 | **liteio** | 0.1 MB/s | minio | 0.1 MB/s | +15% |
| Scale/Write/10 | **liteio** | 0.1 MB/s | minio | 0.1 MB/s | +67% |
| Scale/Write/100 | **liteio** | 0.1 MB/s | minio | 0.1 MB/s | +29% |
| Scale/Write/1000 | **liteio** | 0.1 MB/s | minio | 0.1 MB/s | +61% |
| Stat | **liteio** | 1.2K ops/s | minio | 628 ops/s | +84% |
| Write/10MB | **minio** | 55.0 MB/s | liteio | 54.9 MB/s | ~equal |
| Write/1KB | **liteio** | 0.4 MB/s | minio | 0.3 MB/s | +37% |
| Write/1MB | **liteio** | 43.1 MB/s | minio | 39.7 MB/s | ~equal |
| Write/64KB | **liteio** | 12.5 MB/s | minio | 10.8 MB/s | +16% |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 3/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | minio | 55.0 MB/s | ~equal |
| Write/1KB | liteio | 0.4 MB/s | +37% |
| Write/1MB | liteio | 43.1 MB/s | ~equal |
| Write/64KB | liteio | 12.5 MB/s | +16% |

### Read Operations

**Best for Read:** liteio (won 3/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | minio | 109.4 MB/s | ~equal |
| Read/1KB | liteio | 1.0 MB/s | +59% |
| Read/1MB | liteio | 89.2 MB/s | +53% |
| Read/64KB | liteio | 33.9 MB/s | +30% |

### ParallelWrite Operations

**Best for ParallelWrite:** liteio (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio | 0.4 MB/s | +14% |
| ParallelWrite/1KB/C10 | liteio | 0.1 MB/s | +60% |
| ParallelWrite/1KB/C50 | liteio | 0.0 MB/s | 2.3x faster |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 0.9 MB/s | +35% |
| ParallelRead/1KB/C10 | liteio | 0.6 MB/s | +77% |
| ParallelRead/1KB/C50 | liteio | 0.2 MB/s | 3.1x faster |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 1.1K ops/s | 2.0x faster |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 1.2K ops/s | +84% |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 367 ops/s | 2.1x faster |

### Copy Operations

**Best for Copy:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio | 0.3 MB/s | +68% |

### Scale Operations

**Best for Scale:** liteio (won 12/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | liteio | 1.2K ops/s | 6.7x faster |
| Scale/Delete/10 | liteio | 142 ops/s | 2.1x faster |
| Scale/Delete/100 | liteio | 7 ops/s | +12% |
| Scale/Delete/1000 | liteio | 1 ops/s | +71% |
| Scale/List/1 | liteio | 1.2K ops/s | 3.6x faster |
| Scale/List/10 | liteio | 1.1K ops/s | 2.6x faster |
| Scale/List/100 | liteio | 205 ops/s | +57% |
| Scale/List/1000 | liteio | 29 ops/s | +48% |
| Scale/Write/1 | liteio | 0.1 MB/s | +15% |
| Scale/Write/10 | liteio | 0.1 MB/s | +67% |
| Scale/Write/100 | liteio | 0.1 MB/s | +29% |
| Scale/Write/1000 | liteio | 0.1 MB/s | +61% |

---

*Generated by storage benchmark CLI*
