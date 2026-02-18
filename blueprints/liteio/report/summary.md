# Storage Benchmark Summary

**Generated:** 2026-02-18T23:29:57+07:00

## Overall Winner

**liteio** won 40/40 categories (100%)

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **liteio** | 0.9 MB/s | minio | 0.2 MB/s | 4.1x faster |
| Delete | **liteio** | 1.1K ops/s | minio | 579 ops/s | +97% |
| EdgeCase/DeepNested | **liteio** | 0.1 MB/s | minio | 0.0 MB/s | 7.0x faster |
| EdgeCase/EmptyObject | **liteio** | 1.0K ops/s | minio | 224 ops/s | 4.5x faster |
| EdgeCase/LongKey256 | **liteio** | 0.1 MB/s | minio | 0.0 MB/s | 5.2x faster |
| List/100 | **liteio** | 384 ops/s | minio | 196 ops/s | +96% |
| MixedWorkload/Balanced_50_50 | **liteio** | 0.5 MB/s | minio | 0.1 MB/s | 4.2x faster |
| MixedWorkload/ReadHeavy_90_10 | **liteio** | 0.5 MB/s | minio | 0.2 MB/s | 2.3x faster |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 0.3 MB/s | minio | 0.1 MB/s | 4.3x faster |
| Multipart/15MB_3Parts | **liteio** | 58.1 MB/s | minio | 52.3 MB/s | +11% |
| ParallelRead/1KB/C1 | **liteio** | 1.2 MB/s | minio | 0.6 MB/s | 2.1x faster |
| ParallelRead/1KB/C10 | **liteio** | 0.9 MB/s | minio | 0.4 MB/s | 2.4x faster |
| ParallelRead/1KB/C50 | **liteio** | 0.2 MB/s | minio | 0.1 MB/s | 3.3x faster |
| ParallelWrite/1KB/C1 | **liteio** | 0.9 MB/s | minio | 0.2 MB/s | 4.9x faster |
| ParallelWrite/1KB/C10 | **liteio** | 0.7 MB/s | minio | 0.1 MB/s | 10.2x faster |
| ParallelWrite/1KB/C50 | **liteio** | 0.2 MB/s | minio | 0.0 MB/s | 29.0x faster |
| RangeRead/End_256KB | **liteio** | 59.5 MB/s | minio | 38.0 MB/s | +57% |
| RangeRead/Middle_256KB | **liteio** | 60.2 MB/s | minio | 40.8 MB/s | +48% |
| RangeRead/Start_256KB | **liteio** | 73.4 MB/s | minio | 41.7 MB/s | +76% |
| Read/10MB | **liteio** | 114.5 MB/s | minio | 97.3 MB/s | +18% |
| Read/1KB | **liteio** | 1.3 MB/s | minio | 0.6 MB/s | +96% |
| Read/1MB | **liteio** | 76.4 MB/s | minio | 75.0 MB/s | ~equal |
| Read/64KB | **liteio** | 39.7 MB/s | minio | 25.6 MB/s | +55% |
| Scale/Delete/1 | **liteio** | 461 ops/s | minio | 322 ops/s | +43% |
| Scale/Delete/10 | **liteio** | 140 ops/s | minio | 45 ops/s | 3.1x faster |
| Scale/Delete/100 | **liteio** | 6 ops/s | minio | 6 ops/s | +15% |
| Scale/Delete/1000 | **liteio** | 1 ops/s | minio | 1 ops/s | 2.2x faster |
| Scale/List/1 | **liteio** | 577 ops/s | minio | 161 ops/s | 3.6x faster |
| Scale/List/10 | **liteio** | 1.1K ops/s | minio | 263 ops/s | 4.3x faster |
| Scale/List/100 | **liteio** | 438 ops/s | minio | 125 ops/s | 3.5x faster |
| Scale/List/1000 | **liteio** | 77 ops/s | minio | 30 ops/s | 2.5x faster |
| Scale/Write/1 | **liteio** | 0.1 MB/s | minio | 0.0 MB/s | 2.7x faster |
| Scale/Write/10 | **liteio** | 0.3 MB/s | minio | 0.0 MB/s | 7.6x faster |
| Scale/Write/100 | **liteio** | 0.3 MB/s | minio | 0.1 MB/s | 4.4x faster |
| Scale/Write/1000 | **liteio** | 0.3 MB/s | minio | 0.1 MB/s | 5.0x faster |
| Stat | **liteio** | 1.5K ops/s | minio | 850 ops/s | +71% |
| Write/10MB | **liteio** | 63.3 MB/s | minio | 58.1 MB/s | ~equal |
| Write/1KB | **liteio** | 1.4 MB/s | minio | 0.2 MB/s | 8.7x faster |
| Write/1MB | **liteio** | 56.7 MB/s | minio | 32.9 MB/s | +72% |
| Write/64KB | **liteio** | 30.0 MB/s | minio | 8.2 MB/s | 3.7x faster |

## Category Summaries

### Write Operations

**Best for Write:** liteio (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | liteio | 63.3 MB/s | ~equal |
| Write/1KB | liteio | 1.4 MB/s | 8.7x faster |
| Write/1MB | liteio | 56.7 MB/s | +72% |
| Write/64KB | liteio | 30.0 MB/s | 3.7x faster |

### Read Operations

**Best for Read:** liteio (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | liteio | 114.5 MB/s | +18% |
| Read/1KB | liteio | 1.3 MB/s | +96% |
| Read/1MB | liteio | 76.4 MB/s | ~equal |
| Read/64KB | liteio | 39.7 MB/s | +55% |

### ParallelWrite Operations

**Best for ParallelWrite:** liteio (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | liteio | 0.9 MB/s | 4.9x faster |
| ParallelWrite/1KB/C10 | liteio | 0.7 MB/s | 10.2x faster |
| ParallelWrite/1KB/C50 | liteio | 0.2 MB/s | 29.0x faster |

### ParallelRead Operations

**Best for ParallelRead:** liteio (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | liteio | 1.2 MB/s | 2.1x faster |
| ParallelRead/1KB/C10 | liteio | 0.9 MB/s | 2.4x faster |
| ParallelRead/1KB/C50 | liteio | 0.2 MB/s | 3.3x faster |

### Delete Operations

**Best for Delete:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | liteio | 1.1K ops/s | +97% |

### Stat Operations

**Best for Stat:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | liteio | 1.5K ops/s | +71% |

### List Operations

**Best for List:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | liteio | 384 ops/s | +96% |

### Copy Operations

**Best for Copy:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio | 0.9 MB/s | 4.1x faster |

### Scale Operations

**Best for Scale:** liteio (won 12/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | liteio | 461 ops/s | +43% |
| Scale/Delete/10 | liteio | 140 ops/s | 3.1x faster |
| Scale/Delete/100 | liteio | 6 ops/s | +15% |
| Scale/Delete/1000 | liteio | 1 ops/s | 2.2x faster |
| Scale/List/1 | liteio | 577 ops/s | 3.6x faster |
| Scale/List/10 | liteio | 1.1K ops/s | 4.3x faster |
| Scale/List/100 | liteio | 438 ops/s | 3.5x faster |
| Scale/List/1000 | liteio | 77 ops/s | 2.5x faster |
| Scale/Write/1 | liteio | 0.1 MB/s | 2.7x faster |
| Scale/Write/10 | liteio | 0.3 MB/s | 7.6x faster |
| Scale/Write/100 | liteio | 0.3 MB/s | 4.4x faster |
| Scale/Write/1000 | liteio | 0.3 MB/s | 5.0x faster |

---

*Generated by storage benchmark CLI*
