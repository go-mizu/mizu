# Storage Benchmark Summary

**Generated:** 2026-02-19T14:08:30+07:00

## Overall Winner

**horse** won 28/40 categories (70%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| horse | 28 | 70% |
| zebra | 12 | 30% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **zebra** | 832.4 MB/s | horse | 702.7 MB/s | +18% |
| Delete | **zebra** | 2.7M ops/s | horse | 2.5M ops/s | ~equal |
| EdgeCase/DeepNested | **zebra** | 105.5 MB/s | horse | 91.7 MB/s | +15% |
| EdgeCase/EmptyObject | **horse** | 1.8M ops/s | zebra | 1.4M ops/s | +31% |
| EdgeCase/LongKey256 | **zebra** | 99.7 MB/s | horse | 85.8 MB/s | +16% |
| List/100 | **horse** | 135.6K ops/s | zebra | 64.4K ops/s | 2.1x faster |
| MixedWorkload/Balanced_50_50 | **zebra** | 94.8 MB/s | horse | 2.0 MB/s | 46.6x faster |
| MixedWorkload/ReadHeavy_90_10 | **zebra** | 329.8 MB/s | horse | 91.8 MB/s | 3.6x faster |
| MixedWorkload/WriteHeavy_10_90 | **zebra** | 3.3 MB/s | horse | 0.8 MB/s | 4.0x faster |
| Multipart/15MB_3Parts | **horse** | 176.9 MB/s | zebra | 47.4 MB/s | 3.7x faster |
| ParallelRead/1KB/C1 | **zebra** | 6.7 GB/s | horse | 6.6 GB/s | ~equal |
| ParallelRead/1KB/C10 | **horse** | 5.2 GB/s | zebra | 3.9 GB/s | +32% |
| ParallelRead/1KB/C50 | **zebra** | 3.9 GB/s | horse | 3.0 GB/s | +30% |
| ParallelWrite/1KB/C1 | **horse** | 997.8 MB/s | zebra | 917.3 MB/s | ~equal |
| ParallelWrite/1KB/C10 | **zebra** | 494.0 MB/s | horse | 105.2 MB/s | 4.7x faster |
| ParallelWrite/1KB/C50 | **zebra** | 190.1 MB/s | horse | 23.9 MB/s | 8.0x faster |
| RangeRead/End_256KB | **horse** | 2448.7 GB/s | zebra | 2274.3 GB/s | ~equal |
| RangeRead/Middle_256KB | **horse** | 2507.7 GB/s | zebra | 1968.8 GB/s | +27% |
| RangeRead/Start_256KB | **horse** | 2260.5 GB/s | zebra | 2072.7 GB/s | ~equal |
| Read/10MB | **horse** | 90199.7 GB/s | zebra | 52783.3 GB/s | +71% |
| Read/1KB | **horse** | 8.6 GB/s | zebra | 8.0 GB/s | ~equal |
| Read/1MB | **horse** | 9730.1 GB/s | zebra | 8180.0 GB/s | +19% |
| Read/64KB | **horse** | 651.8 GB/s | zebra | 523.0 GB/s | +25% |
| Scale/Delete/1 | **horse** | 750.2K ops/s | zebra | 45.6K ops/s | 16.4x faster |
| Scale/Delete/10 | **horse** | 230.8K ops/s | zebra | 44.9K ops/s | 5.1x faster |
| Scale/Delete/100 | **horse** | 40.7K ops/s | zebra | 9.8K ops/s | 4.2x faster |
| Scale/Delete/1000 | **horse** | 3.8K ops/s | zebra | 997 ops/s | 3.8x faster |
| Scale/List/1 | **horse** | 303.9K ops/s | zebra | 1 ops/s | 587854.9x faster |
| Scale/List/10 | **horse** | 235.3K ops/s | zebra | 1 ops/s | 349898.3x faster |
| Scale/List/100 | **horse** | 57.8K ops/s | zebra | 1 ops/s | 85627.4x faster |
| Scale/List/1000 | **horse** | 6.0K ops/s | zebra | 1 ops/s | 8829.9x faster |
| Scale/Write/1 | **horse** | 41.3 MB/s | zebra | 25.9 MB/s | +59% |
| Scale/Write/10 | **horse** | 300.5 MB/s | zebra | 101.4 MB/s | 3.0x faster |
| Scale/Write/100 | **horse** | 330.1 MB/s | zebra | 153.9 MB/s | 2.1x faster |
| Scale/Write/1000 | **horse** | 384.5 MB/s | zebra | 265.2 MB/s | +45% |
| Stat | **horse** | 17.1M ops/s | zebra | 10.5M ops/s | +63% |
| Write/10MB | **horse** | 544.6 MB/s | zebra | 315.3 MB/s | +73% |
| Write/1KB | **zebra** | 1.8 GB/s | horse | 1.4 GB/s | +33% |
| Write/1MB | **horse** | 1.1 GB/s | zebra | 497.3 MB/s | 2.1x faster |
| Write/64KB | **horse** | 1.5 GB/s | zebra | 367.0 MB/s | 4.0x faster |

## Category Summaries

### Write Operations

**Best for Write:** horse (won 3/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | horse | 544.6 MB/s | +73% |
| Write/1KB | zebra | 1.8 GB/s | +33% |
| Write/1MB | horse | 1.1 GB/s | 2.1x faster |
| Write/64KB | horse | 1.5 GB/s | 4.0x faster |

### Read Operations

**Best for Read:** horse (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | horse | 90199.7 GB/s | +71% |
| Read/1KB | horse | 8.6 GB/s | ~equal |
| Read/1MB | horse | 9730.1 GB/s | +19% |
| Read/64KB | horse | 651.8 GB/s | +25% |

### ParallelWrite Operations

**Best for ParallelWrite:** zebra (won 2/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | horse | 997.8 MB/s | ~equal |
| ParallelWrite/1KB/C10 | zebra | 494.0 MB/s | 4.7x faster |
| ParallelWrite/1KB/C50 | zebra | 190.1 MB/s | 8.0x faster |

### ParallelRead Operations

**Best for ParallelRead:** zebra (won 2/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | zebra | 6.7 GB/s | ~equal |
| ParallelRead/1KB/C10 | horse | 5.2 GB/s | +32% |
| ParallelRead/1KB/C50 | zebra | 3.9 GB/s | +30% |

### Delete Operations

**Best for Delete:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | zebra | 2.7M ops/s | ~equal |

### Stat Operations

**Best for Stat:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | horse | 17.1M ops/s | +63% |

### List Operations

**Best for List:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | horse | 135.6K ops/s | 2.1x faster |

### Copy Operations

**Best for Copy:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | zebra | 832.4 MB/s | +18% |

### Scale Operations

**Best for Scale:** horse (won 12/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | horse | 750.2K ops/s | 16.4x faster |
| Scale/Delete/10 | horse | 230.8K ops/s | 5.1x faster |
| Scale/Delete/100 | horse | 40.7K ops/s | 4.2x faster |
| Scale/Delete/1000 | horse | 3.8K ops/s | 3.8x faster |
| Scale/List/1 | horse | 303.9K ops/s | 587854.9x faster |
| Scale/List/10 | horse | 235.3K ops/s | 349898.3x faster |
| Scale/List/100 | horse | 57.8K ops/s | 85627.4x faster |
| Scale/List/1000 | horse | 6.0K ops/s | 8829.9x faster |
| Scale/Write/1 | horse | 41.3 MB/s | +59% |
| Scale/Write/10 | horse | 300.5 MB/s | 3.0x faster |
| Scale/Write/100 | horse | 330.1 MB/s | 2.1x faster |
| Scale/Write/1000 | horse | 384.5 MB/s | +45% |

---

*Generated by storage benchmark CLI*
