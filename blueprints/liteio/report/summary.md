# Storage Benchmark Summary

**Generated:** 2026-02-18T23:05:36+07:00

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
| Copy/1KB | **liteio** | 0.3 MB/s | minio | 0.3 MB/s | +32% |
| Delete | **liteio** | 1.2K ops/s | minio | 569 ops/s | 2.1x faster |
| EdgeCase/DeepNested | **liteio** | 0.0 MB/s | minio | 0.0 MB/s | 2.2x faster |
| EdgeCase/EmptyObject | **liteio** | 463 ops/s | minio | 197 ops/s | 2.3x faster |
| EdgeCase/LongKey256 | **liteio** | 0.0 MB/s | minio | 0.0 MB/s | 2.1x faster |
| List/100 | **liteio** | 368 ops/s | minio | 184 ops/s | 2.0x faster |
| MixedWorkload/Balanced_50_50 | **liteio** | 0.2 MB/s | minio | 0.1 MB/s | 3.1x faster |
| MixedWorkload/ReadHeavy_90_10 | **liteio** | 0.5 MB/s | minio | 0.2 MB/s | 3.0x faster |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 0.1 MB/s | minio | 0.1 MB/s | +34% |
| Multipart/15MB_3Parts | **liteio** | 46.9 MB/s | minio | 43.9 MB/s | ~equal |
| ParallelRead/1KB/C1 | **liteio** | 0.8 MB/s | minio | 0.6 MB/s | +37% |
| ParallelRead/1KB/C10 | **liteio** | 0.6 MB/s | minio | 0.4 MB/s | +73% |
| ParallelRead/1KB/C50 | **liteio** | 0.2 MB/s | minio | 0.1 MB/s | 2.4x faster |
| ParallelWrite/1KB/C1 | **liteio** | 0.4 MB/s | minio | 0.2 MB/s | +53% |
| ParallelWrite/1KB/C10 | **liteio** | 0.2 MB/s | minio | 0.1 MB/s | 2.2x faster |
| ParallelWrite/1KB/C50 | **liteio** | 0.0 MB/s | minio | 0.0 MB/s | 2.8x faster |
| RangeRead/End_256KB | **liteio** | 58.9 MB/s | minio | 46.1 MB/s | +28% |
| RangeRead/Middle_256KB | **liteio** | 59.5 MB/s | minio | 39.5 MB/s | +51% |
| RangeRead/Start_256KB | **liteio** | 64.9 MB/s | minio | 35.4 MB/s | +83% |
| Read/10MB | **liteio** | 104.4 MB/s | minio | 103.7 MB/s | ~equal |
| Read/1KB | **liteio** | 1.1 MB/s | minio | 0.6 MB/s | +77% |
| Read/1MB | **liteio** | 84.0 MB/s | minio | 72.0 MB/s | +17% |
| Read/64KB | **liteio** | 39.5 MB/s | minio | 22.0 MB/s | +79% |
| Scale/Delete/1 | **liteio** | 1.4K ops/s | minio | 398 ops/s | 3.4x faster |
| Scale/Delete/10 | **liteio** | 108 ops/s | minio | 58 ops/s | +87% |
| Scale/Delete/100 | **liteio** | 8 ops/s | minio | 6 ops/s | +39% |
| Scale/Delete/1000 | **liteio** | 1 ops/s | minio | 1 ops/s | 2.2x faster |
| Scale/List/1 | **liteio** | 1.3K ops/s | minio | 344 ops/s | 3.7x faster |
| Scale/List/10 | **liteio** | 448 ops/s | minio | 439 ops/s | ~equal |
| Scale/List/100 | **liteio** | 366 ops/s | minio | 205 ops/s | +79% |
| Scale/List/1000 | **liteio** | 53 ops/s | minio | 19 ops/s | 2.8x faster |
| Scale/Write/1 | **liteio** | 0.1 MB/s | minio | 0.1 MB/s | +51% |
| Scale/Write/10 | **minio** | 0.1 MB/s | liteio | 0.1 MB/s | +20% |
| Scale/Write/100 | **liteio** | 0.1 MB/s | minio | 0.1 MB/s | ~equal |
| Scale/Write/1000 | **liteio** | 0.1 MB/s | minio | 0.1 MB/s | +56% |
| Stat | **liteio** | 1.1K ops/s | minio | 823 ops/s | +38% |
| Write/10MB | **minio** | 53.7 MB/s | liteio | 53.5 MB/s | ~equal |
| Write/1KB | **liteio** | 0.4 MB/s | minio | 0.2 MB/s | +81% |
| Write/1MB | **liteio** | 46.6 MB/s | minio | 36.7 MB/s | +27% |
| Write/64KB | **liteio** | 15.5 MB/s | minio | 9.6 MB/s | +62% |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 3/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | minio | 53.7 MB/s | ~equal |
| Write/1KB | liteio | 0.4 MB/s | +81% |
| Write/1MB | liteio | 46.6 MB/s | +27% |
| Write/64KB | liteio | 15.5 MB/s | +62% |

### Read Operations

**Best for Read:** liteio (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | liteio | 104.4 MB/s | ~equal |
| Read/1KB | liteio | 1.1 MB/s | +77% |
| Read/1MB | liteio | 84.0 MB/s | +17% |
| Read/64KB | liteio | 39.5 MB/s | +79% |

### ParallelWrite Operations

**Best for ParallelWrite:** liteio (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio | 0.4 MB/s | +53% |
| ParallelWrite/1KB/C10 | liteio | 0.2 MB/s | 2.2x faster |
| ParallelWrite/1KB/C50 | liteio | 0.0 MB/s | 2.8x faster |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 0.8 MB/s | +37% |
| ParallelRead/1KB/C10 | liteio | 0.6 MB/s | +73% |
| ParallelRead/1KB/C50 | liteio | 0.2 MB/s | 2.4x faster |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 1.2K ops/s | 2.1x faster |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 1.1K ops/s | +38% |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 368 ops/s | 2.0x faster |

### Copy Operations

**Best for Copy:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio | 0.3 MB/s | +32% |

### Scale Operations

**Best for Scale:** liteio (won 11/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | liteio | 1.4K ops/s | 3.4x faster |
| Scale/Delete/10 | liteio | 108 ops/s | +87% |
| Scale/Delete/100 | liteio | 8 ops/s | +39% |
| Scale/Delete/1000 | liteio | 1 ops/s | 2.2x faster |
| Scale/List/1 | liteio | 1.3K ops/s | 3.7x faster |
| Scale/List/10 | liteio | 448 ops/s | ~equal |
| Scale/List/100 | liteio | 366 ops/s | +79% |
| Scale/List/1000 | liteio | 53 ops/s | 2.8x faster |
| Scale/Write/1 | liteio | 0.1 MB/s | +51% |
| Scale/Write/10 | minio | 0.1 MB/s | +20% |
| Scale/Write/100 | liteio | 0.1 MB/s | ~equal |
| Scale/Write/1000 | liteio | 0.1 MB/s | +56% |

---

*Generated by storage benchmark CLI*
