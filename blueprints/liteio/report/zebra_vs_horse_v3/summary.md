# Storage Benchmark Summary

**Generated:** 2026-02-19T14:01:57+07:00

## Overall Winner

**horse** won 29/40 categories (72%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| horse | 29 | 72% |
| zebra | 11 | 28% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **zebra** | 1.1 GB/s | horse | 627.8 MB/s | +73% |
| Delete | **zebra** | 2.9M ops/s | horse | 2.5M ops/s | +14% |
| EdgeCase/DeepNested | **horse** | 118.0 MB/s | zebra | 114.4 MB/s | ~equal |
| EdgeCase/EmptyObject | **horse** | 1.5M ops/s | zebra | 1.2M ops/s | +31% |
| EdgeCase/LongKey256 | **zebra** | 91.5 MB/s | horse | 4.1 MB/s | 22.3x faster |
| List/100 | **horse** | 125.7K ops/s | zebra | 74.3K ops/s | +69% |
| MixedWorkload/Balanced_50_50 | **zebra** | 26.2 MB/s | horse | 7.3 MB/s | 3.6x faster |
| MixedWorkload/ReadHeavy_90_10 | **horse** | 126.3 MB/s | zebra | 62.7 MB/s | 2.0x faster |
| MixedWorkload/WriteHeavy_10_90 | **zebra** | 7.0 MB/s | horse | 2.6 MB/s | 2.7x faster |
| Multipart/15MB_3Parts | **horse** | 210.2 MB/s | zebra | 41.9 MB/s | 5.0x faster |
| ParallelRead/1KB/C1 | **horse** | 6.8 GB/s | zebra | 6.3 GB/s | ~equal |
| ParallelRead/1KB/C10 | **horse** | 3.5 GB/s | zebra | 3.1 GB/s | +15% |
| ParallelRead/1KB/C50 | **horse** | 3.0 GB/s | zebra | 1.3 GB/s | 2.2x faster |
| ParallelWrite/1KB/C1 | **horse** | 971.2 MB/s | zebra | 932.8 MB/s | ~equal |
| ParallelWrite/1KB/C10 | **zebra** | 408.7 MB/s | horse | 137.8 MB/s | 3.0x faster |
| ParallelWrite/1KB/C50 | **zebra** | 197.4 MB/s | horse | 37.8 MB/s | 5.2x faster |
| RangeRead/End_256KB | **horse** | 2480.0 GB/s | zebra | 2393.6 GB/s | ~equal |
| RangeRead/Middle_256KB | **horse** | 2241.1 GB/s | zebra | 2071.6 GB/s | ~equal |
| RangeRead/Start_256KB | **horse** | 2103.1 GB/s | zebra | 1785.2 GB/s | +18% |
| Read/10MB | **zebra** | 37.1 GB/s | horse | 10.7 GB/s | 3.5x faster |
| Read/1KB | **horse** | 8.9 GB/s | zebra | 5.9 GB/s | +50% |
| Read/1MB | **zebra** | 5818.7 GB/s | horse | 35.2 GB/s | 165.5x faster |
| Read/64KB | **horse** | 627.8 GB/s | zebra | 465.3 GB/s | +35% |
| Scale/Delete/1 | **horse** | 705.7K ops/s | zebra | 152.9K ops/s | 4.6x faster |
| Scale/Delete/10 | **horse** | 90.2K ops/s | zebra | 24.8K ops/s | 3.6x faster |
| Scale/Delete/100 | **horse** | 35.1K ops/s | zebra | 8.3K ops/s | 4.2x faster |
| Scale/Delete/1000 | **horse** | 3.7K ops/s | zebra | 1.3K ops/s | 2.8x faster |
| Scale/List/1 | **horse** | 292.7K ops/s | zebra | 1 ops/s | 441880.5x faster |
| Scale/List/10 | **horse** | 216.2K ops/s | zebra | 1 ops/s | 319255.1x faster |
| Scale/List/100 | **horse** | 39.2K ops/s | zebra | 1 ops/s | 53703.1x faster |
| Scale/List/1000 | **horse** | 5.1K ops/s | zebra | 1 ops/s | 7183.4x faster |
| Scale/Write/1 | **zebra** | 29.6 MB/s | horse | 4.4 MB/s | 6.8x faster |
| Scale/Write/10 | **horse** | 192.7 MB/s | zebra | 61.9 MB/s | 3.1x faster |
| Scale/Write/100 | **horse** | 81.8 MB/s | zebra | 30.4 MB/s | 2.7x faster |
| Scale/Write/1000 | **horse** | 236.6 MB/s | zebra | 77.0 MB/s | 3.1x faster |
| Stat | **horse** | 14.9M ops/s | zebra | 11.5M ops/s | +30% |
| Write/10MB | **horse** | 1.1 GB/s | zebra | 56.4 MB/s | 19.6x faster |
| Write/1KB | **zebra** | 1.9 GB/s | horse | 1.4 GB/s | +34% |
| Write/1MB | **horse** | 1.1 GB/s | zebra | 346.6 MB/s | 3.3x faster |
| Write/64KB | **horse** | 1.5 GB/s | zebra | 1.2 GB/s | +30% |

## Category Summaries

### Write Operations

**Best for Write:** horse (won 3/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | horse | 1.1 GB/s | 19.6x faster |
| Write/1KB | zebra | 1.9 GB/s | +34% |
| Write/1MB | horse | 1.1 GB/s | 3.3x faster |
| Write/64KB | horse | 1.5 GB/s | +30% |

### Read Operations

**Best for Read:** zebra (won 2/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | zebra | 37.1 GB/s | 3.5x faster |
| Read/1KB | horse | 8.9 GB/s | +50% |
| Read/1MB | zebra | 5818.7 GB/s | 165.5x faster |
| Read/64KB | horse | 627.8 GB/s | +35% |

### ParallelWrite Operations

**Best for ParallelWrite:** zebra (won 2/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | horse | 971.2 MB/s | ~equal |
| ParallelWrite/1KB/C10 | zebra | 408.7 MB/s | 3.0x faster |
| ParallelWrite/1KB/C50 | zebra | 197.4 MB/s | 5.2x faster |

### ParallelRead Operations

**Best for ParallelRead:** horse (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | horse | 6.8 GB/s | ~equal |
| ParallelRead/1KB/C10 | horse | 3.5 GB/s | +15% |
| ParallelRead/1KB/C50 | horse | 3.0 GB/s | 2.2x faster |

### Delete Operations

**Best for Delete:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | zebra | 2.9M ops/s | +14% |

### Stat Operations

**Best for Stat:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | horse | 14.9M ops/s | +30% |

### List Operations

**Best for List:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | horse | 125.7K ops/s | +69% |

### Copy Operations

**Best for Copy:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | zebra | 1.1 GB/s | +73% |

### Scale Operations

**Best for Scale:** horse (won 11/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | horse | 705.7K ops/s | 4.6x faster |
| Scale/Delete/10 | horse | 90.2K ops/s | 3.6x faster |
| Scale/Delete/100 | horse | 35.1K ops/s | 4.2x faster |
| Scale/Delete/1000 | horse | 3.7K ops/s | 2.8x faster |
| Scale/List/1 | horse | 292.7K ops/s | 441880.5x faster |
| Scale/List/10 | horse | 216.2K ops/s | 319255.1x faster |
| Scale/List/100 | horse | 39.2K ops/s | 53703.1x faster |
| Scale/List/1000 | horse | 5.1K ops/s | 7183.4x faster |
| Scale/Write/1 | zebra | 29.6 MB/s | 6.8x faster |
| Scale/Write/10 | horse | 192.7 MB/s | 3.1x faster |
| Scale/Write/100 | horse | 81.8 MB/s | 2.7x faster |
| Scale/Write/1000 | horse | 236.6 MB/s | 3.1x faster |

---

*Generated by storage benchmark CLI*
