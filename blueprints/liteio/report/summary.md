# Storage Benchmark Summary

**Generated:** 2026-02-19T14:25:45+07:00

## Overall Winner

**horse** won 29/40 categories (72%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| horse | 29 | 72% |
| pony | 11 | 28% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **pony** | 394.5 MB/s | horse | 286.1 MB/s | +38% |
| Delete | **pony** | 2.5M ops/s | horse | 1.9M ops/s | +35% |
| EdgeCase/DeepNested | **horse** | 44.2 MB/s | pony | 32.7 MB/s | +35% |
| EdgeCase/EmptyObject | **pony** | 962.2K ops/s | horse | 488.2K ops/s | +97% |
| EdgeCase/LongKey256 | **horse** | 35.6 MB/s | pony | 12.6 MB/s | 2.8x faster |
| List/100 | **horse** | 33.5K ops/s | pony | 21.9K ops/s | +53% |
| MixedWorkload/Balanced_50_50 | **horse** | 24.3 MB/s | pony | 12.6 MB/s | +93% |
| MixedWorkload/ReadHeavy_90_10 | **horse** | 42.7 MB/s | pony | 7.5 MB/s | 5.7x faster |
| MixedWorkload/WriteHeavy_10_90 | **horse** | 141.4 MB/s | pony | 2.5 MB/s | 56.9x faster |
| Multipart/15MB_3Parts | **pony** | 200.2 MB/s | horse | 122.6 MB/s | +63% |
| ParallelRead/1KB/C1 | **horse** | 1.5 GB/s | pony | 483.7 MB/s | 3.2x faster |
| ParallelRead/1KB/C10 | **horse** | 358.9 MB/s | pony | 237.2 MB/s | +51% |
| ParallelRead/1KB/C50 | **horse** | 274.3 MB/s | pony | 100.3 MB/s | 2.7x faster |
| ParallelWrite/1KB/C1 | **horse** | 365.5 MB/s | pony | 88.0 MB/s | 4.2x faster |
| ParallelWrite/1KB/C10 | **pony** | 76.7 MB/s | horse | 70.2 MB/s | ~equal |
| ParallelWrite/1KB/C50 | **pony** | 28.0 MB/s | horse | 12.6 MB/s | 2.2x faster |
| RangeRead/End_256KB | **horse** | 627.8 GB/s | pony | 533.8 GB/s | +18% |
| RangeRead/Middle_256KB | **horse** | 552.8 GB/s | pony | 505.8 GB/s | ~equal |
| RangeRead/Start_256KB | **horse** | 585.7 GB/s | pony | 512.3 GB/s | +14% |
| Read/10MB | **horse** | 24067.3 GB/s | pony | 1.9 GB/s | 12998.4x faster |
| Read/1KB | **horse** | 3.0 GB/s | pony | 2.9 GB/s | ~equal |
| Read/1MB | **horse** | 1271.4 GB/s | pony | 1.9 GB/s | 653.5x faster |
| Read/64KB | **horse** | 189.3 GB/s | pony | 8.9 GB/s | 21.4x faster |
| Scale/Delete/1 | **horse** | 338.1K ops/s | pony | 71.2K ops/s | 4.7x faster |
| Scale/Delete/10 | **horse** | 65.8K ops/s | pony | 55.4K ops/s | +19% |
| Scale/Delete/100 | **horse** | 10.3K ops/s | pony | 6.8K ops/s | +52% |
| Scale/Delete/1000 | **pony** | 1.4K ops/s | horse | 812 ops/s | +67% |
| Scale/List/1 | **horse** | 99.6K ops/s | pony | 1 ops/s | 88901.4x faster |
| Scale/List/10 | **horse** | 121.2K ops/s | pony | 1 ops/s | 96573.2x faster |
| Scale/List/100 | **horse** | 14.7K ops/s | pony | 1 ops/s | 11125.4x faster |
| Scale/List/1000 | **horse** | 1.3K ops/s | pony | 1 ops/s | 985.4x faster |
| Scale/Write/1 | **pony** | 31.3 MB/s | horse | 5.7 MB/s | 5.5x faster |
| Scale/Write/10 | **horse** | 114.2 MB/s | pony | 63.9 MB/s | +79% |
| Scale/Write/100 | **horse** | 73.1 MB/s | pony | 47.8 MB/s | +53% |
| Scale/Write/1000 | **horse** | 152.6 MB/s | pony | 30.9 MB/s | 4.9x faster |
| Stat | **horse** | 5.2M ops/s | pony | 4.3M ops/s | +21% |
| Write/10MB | **pony** | 1.1 GB/s | horse | 230.5 MB/s | 5.0x faster |
| Write/1KB | **pony** | 802.8 MB/s | horse | 716.8 MB/s | +12% |
| Write/1MB | **pony** | 605.1 MB/s | horse | 551.3 MB/s | ~equal |
| Write/64KB | **horse** | 2.1 GB/s | pony | 1.7 GB/s | +23% |

## Category Summaries

### Write Operations

**Best for Write:** pony (won 3/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | pony | 1.1 GB/s | 5.0x faster |
| Write/1KB | pony | 802.8 MB/s | +12% |
| Write/1MB | pony | 605.1 MB/s | ~equal |
| Write/64KB | horse | 2.1 GB/s | +23% |

### Read Operations

**Best for Read:** horse (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | horse | 24067.3 GB/s | 12998.4x faster |
| Read/1KB | horse | 3.0 GB/s | ~equal |
| Read/1MB | horse | 1271.4 GB/s | 653.5x faster |
| Read/64KB | horse | 189.3 GB/s | 21.4x faster |

### ParallelWrite Operations

**Best for ParallelWrite:** pony (won 2/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | horse | 365.5 MB/s | 4.2x faster |
| ParallelWrite/1KB/C10 | pony | 76.7 MB/s | ~equal |
| ParallelWrite/1KB/C50 | pony | 28.0 MB/s | 2.2x faster |

### ParallelRead Operations

**Best for ParallelRead:** horse (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | horse | 1.5 GB/s | 3.2x faster |
| ParallelRead/1KB/C10 | horse | 358.9 MB/s | +51% |
| ParallelRead/1KB/C50 | horse | 274.3 MB/s | 2.7x faster |

### Delete Operations

**Best for Delete:** pony (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | pony | 2.5M ops/s | +35% |

### Stat Operations

**Best for Stat:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | horse | 5.2M ops/s | +21% |

### List Operations

**Best for List:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | horse | 33.5K ops/s | +53% |

### Copy Operations

**Best for Copy:** pony (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | pony | 394.5 MB/s | +38% |

### Scale Operations

**Best for Scale:** horse (won 10/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | horse | 338.1K ops/s | 4.7x faster |
| Scale/Delete/10 | horse | 65.8K ops/s | +19% |
| Scale/Delete/100 | horse | 10.3K ops/s | +52% |
| Scale/Delete/1000 | pony | 1.4K ops/s | +67% |
| Scale/List/1 | horse | 99.6K ops/s | 88901.4x faster |
| Scale/List/10 | horse | 121.2K ops/s | 96573.2x faster |
| Scale/List/100 | horse | 14.7K ops/s | 11125.4x faster |
| Scale/List/1000 | horse | 1.3K ops/s | 985.4x faster |
| Scale/Write/1 | pony | 31.3 MB/s | 5.5x faster |
| Scale/Write/10 | horse | 114.2 MB/s | +79% |
| Scale/Write/100 | horse | 73.1 MB/s | +53% |
| Scale/Write/1000 | horse | 152.6 MB/s | 4.9x faster |

---

*Generated by storage benchmark CLI*
