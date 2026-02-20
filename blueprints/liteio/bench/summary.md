# Storage Benchmark Summary

**Generated:** 2026-02-20T15:11:27+07:00

## Overall Winner

**herd** won 14/40 categories (35%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| herd | 14 | 35% |
| pony | 9 | 22% |
| zebra | 9 | 22% |
| horse | 5 | 12% |
| usagi | 3 | 8% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **pony** | 1.6 GB/s | herd | 845.7 MB/s | +95% |
| Delete | **zebra** | 3.0M ops/s | pony | 2.9M ops/s | ~equal |
| EdgeCase/DeepNested | **pony** | 178.9 MB/s | horse | 148.0 MB/s | +21% |
| EdgeCase/EmptyObject | **pony** | 1.6M ops/s | horse | 1.1M ops/s | +44% |
| EdgeCase/LongKey256 | **pony** | 125.4 MB/s | horse | 108.5 MB/s | +16% |
| List/100 | **usagi** | 195.2K ops/s | horse | 133.8K ops/s | +46% |
| MixedWorkload/Balanced_50_50 | **herd** | 135.5 MB/s | rabbit | 73.2 MB/s | +85% |
| MixedWorkload/ReadHeavy_90_10 | **zebra** | 5.2 GB/s | herd | 3.4 GB/s | +55% |
| MixedWorkload/WriteHeavy_10_90 | **herd** | 25.1 MB/s | usagi | 8.1 MB/s | 3.1x faster |
| Multipart/15MB_3Parts | **usagi** | 933.8 MB/s | pony | 416.0 MB/s | 2.2x faster |
| ParallelRead/1KB/C1 | **zebra** | 4.9 GB/s | horse | 4.5 GB/s | +10% |
| ParallelRead/1KB/C10 | **zebra** | 3.0 GB/s | horse | 3.0 GB/s | ~equal |
| ParallelRead/1KB/C50 | **herd** | 2.7 GB/s | zebra | 2.3 GB/s | +21% |
| ParallelWrite/1KB/C1 | **zebra** | 1.1 GB/s | horse | 1.0 GB/s | ~equal |
| ParallelWrite/1KB/C10 | **zebra** | 492.2 MB/s | herd | 278.1 MB/s | +77% |
| ParallelWrite/1KB/C50 | **herd** | 178.0 MB/s | zebra | 138.8 MB/s | +28% |
| RangeRead/End_256KB | **herd** | 58.4 GB/s | pony | 54.4 GB/s | ~equal |
| RangeRead/Middle_256KB | **pony** | 58.5 GB/s | herd | 57.3 GB/s | ~equal |
| RangeRead/Start_256KB | **herd** | 55.6 GB/s | pony | 52.1 GB/s | ~equal |
| Read/10MB | **zebra** | 54.5 GB/s | pony | 51.0 GB/s | ~equal |
| Read/1KB | **horse** | 5.5 GB/s | pony | 5.2 GB/s | ~equal |
| Read/1MB | **pony** | 56.5 GB/s | horse | 55.5 GB/s | ~equal |
| Read/64KB | **horse** | 51.5 GB/s | pony | 51.0 GB/s | ~equal |
| Scale/Delete/1 | **herd** | 342.9K ops/s | horse | 338.1K ops/s | ~equal |
| Scale/Delete/10 | **herd** | 95.6K ops/s | pony | 86.6K ops/s | +10% |
| Scale/Delete/100 | **herd** | 42.9K ops/s | zebra | 26.5K ops/s | +62% |
| Scale/Delete/1000 | **herd** | 3.4K ops/s | zebra | 3.0K ops/s | +15% |
| Scale/List/1 | **horse** | 210.5K ops/s | herd | 84.5K ops/s | 2.5x faster |
| Scale/List/10 | **herd** | 101.7K ops/s | horse | 50.1K ops/s | 2.0x faster |
| Scale/List/100 | **herd** | 24.4K ops/s | horse | 16.5K ops/s | +48% |
| Scale/List/1000 | **zebra** | 3.7K ops/s | herd | 3.5K ops/s | ~equal |
| Scale/Write/1 | **pony** | 52.3 MB/s | zebra | 28.4 MB/s | +84% |
| Scale/Write/10 | **pony** | 174.9 MB/s | horse | 95.4 MB/s | +83% |
| Scale/Write/100 | **pony** | 287.5 MB/s | zebra | 189.4 MB/s | +52% |
| Scale/Write/1000 | **herd** | 338.1 MB/s | zebra | 250.6 MB/s | +35% |
| Stat | **usagi** | 20.4M ops/s | horse | 16.7M ops/s | +22% |
| Write/10MB | **horse** | 2.8 GB/s | rabbit | 1.5 GB/s | +84% |
| Write/1KB | **zebra** | 1.3 GB/s | horse | 1.3 GB/s | ~equal |
| Write/1MB | **horse** | 2.1 GB/s | rabbit | 1.8 GB/s | +14% |
| Write/64KB | **herd** | 9.4 GB/s | zebra | 4.1 GB/s | 2.3x faster |

## Category Summaries

### Write Operations

**Best for Write:** horse (won 2/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | horse | 2.8 GB/s | +84% |
| Write/1KB | zebra | 1.3 GB/s | ~equal |
| Write/1MB | horse | 2.1 GB/s | +14% |
| Write/64KB | herd | 9.4 GB/s | 2.3x faster |

### Read Operations

**Best for Read:** horse (won 2/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | zebra | 54.5 GB/s | ~equal |
| Read/1KB | horse | 5.5 GB/s | ~equal |
| Read/1MB | pony | 56.5 GB/s | ~equal |
| Read/64KB | horse | 51.5 GB/s | ~equal |

### ParallelWrite Operations

**Best for ParallelWrite:** zebra (won 2/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | zebra | 1.1 GB/s | ~equal |
| ParallelWrite/1KB/C10 | zebra | 492.2 MB/s | +77% |
| ParallelWrite/1KB/C50 | herd | 178.0 MB/s | +28% |

### ParallelRead Operations

**Best for ParallelRead:** zebra (won 2/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | zebra | 4.9 GB/s | +10% |
| ParallelRead/1KB/C10 | zebra | 3.0 GB/s | ~equal |
| ParallelRead/1KB/C50 | herd | 2.7 GB/s | +21% |

### Delete Operations

**Best for Delete:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | zebra | 3.0M ops/s | ~equal |

### Stat Operations

**Best for Stat:** usagi (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | usagi | 20.4M ops/s | +22% |

### List Operations

**Best for List:** usagi (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | usagi | 195.2K ops/s | +46% |

### Copy Operations

**Best for Copy:** pony (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | pony | 1.6 GB/s | +95% |

### Scale Operations

**Best for Scale:** herd (won 7/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | herd | 342.9K ops/s | ~equal |
| Scale/Delete/10 | herd | 95.6K ops/s | +10% |
| Scale/Delete/100 | herd | 42.9K ops/s | +62% |
| Scale/Delete/1000 | herd | 3.4K ops/s | +15% |
| Scale/List/1 | horse | 210.5K ops/s | 2.5x faster |
| Scale/List/10 | herd | 101.7K ops/s | 2.0x faster |
| Scale/List/100 | herd | 24.4K ops/s | +48% |
| Scale/List/1000 | zebra | 3.7K ops/s | ~equal |
| Scale/Write/1 | pony | 52.3 MB/s | +84% |
| Scale/Write/10 | pony | 174.9 MB/s | +83% |
| Scale/Write/100 | pony | 287.5 MB/s | +52% |
| Scale/Write/1000 | herd | 338.1 MB/s | +35% |

## Runtime Resource Usage

| Driver | Peak RSS | Go Heap | Go Sys | Disk Usage | GC Cycles |
|--------|----------|---------|--------|------------|----------|
| bee3 | 6317.6 MB | 7807.9 MB | 8923.8 MB | 8698.7 MB | 420 |
| herd | 7072.1 MB | 8138.5 MB | 8991.1 MB | 16384.0 MB | 433 |
| horse | 4100.2 MB | 2701.0 MB | 4801.9 MB | 32768.0 MB | 299 |
| pony | 4100.2 MB | 1898.9 MB | 4807.1 MB | 17482.6 MB | 391 |
| rabbit | 3545.5 MB | 1038.7 MB | 4800.7 MB | 4724.1 MB | 37 |
| usagi | 4100.2 MB | 2701.0 MB | 4801.6 MB | 8736.1 MB | 257 |
| zebra | 7232.9 MB | 8666.6 MB | 9428.9 MB | 16384.0 MB | 441 |

---

*Generated by storage benchmark CLI*
