# Storage Benchmark Summary

**Generated:** 2026-02-19T14:31:17+07:00

## Overall Winner

**horse** won 27/40 categories (68%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| horse | 27 | 68% |
| zebra | 13 | 32% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **zebra** | 919.6 MB/s | horse | 573.6 MB/s | +60% |
| Delete | **zebra** | 3.0M ops/s | horse | 2.5M ops/s | +20% |
| EdgeCase/DeepNested | **zebra** | 119.5 MB/s | horse | 118.0 MB/s | ~equal |
| EdgeCase/EmptyObject | **horse** | 1.5M ops/s | zebra | 910.2K ops/s | +61% |
| EdgeCase/LongKey256 | **zebra** | 121.8 MB/s | horse | 100.6 MB/s | +21% |
| List/100 | **horse** | 135.9K ops/s | zebra | 77.0K ops/s | +76% |
| MixedWorkload/Balanced_50_50 | **horse** | 5.4 MB/s | zebra | 4.9 MB/s | ~equal |
| MixedWorkload/ReadHeavy_90_10 | **zebra** | 129.6 MB/s | horse | 100.9 MB/s | +28% |
| MixedWorkload/WriteHeavy_10_90 | **horse** | 1.9 MB/s | zebra | 0.5 MB/s | 3.9x faster |
| Multipart/15MB_3Parts | **zebra** | 176.7 MB/s | horse | 161.2 MB/s | ~equal |
| ParallelRead/1KB/C1 | **horse** | 6.9 GB/s | zebra | 6.2 GB/s | +13% |
| ParallelRead/1KB/C10 | **horse** | 5.3 GB/s | zebra | 5.1 GB/s | ~equal |
| ParallelRead/1KB/C50 | **horse** | 4.7 GB/s | zebra | 2.2 GB/s | 2.2x faster |
| ParallelWrite/1KB/C1 | **zebra** | 1.1 GB/s | horse | 1.0 GB/s | ~equal |
| ParallelWrite/1KB/C10 | **zebra** | 507.4 MB/s | horse | 161.2 MB/s | 3.1x faster |
| ParallelWrite/1KB/C50 | **zebra** | 161.9 MB/s | horse | 43.7 MB/s | 3.7x faster |
| RangeRead/End_256KB | **horse** | 2541.9 GB/s | zebra | 1978.0 GB/s | +29% |
| RangeRead/Middle_256KB | **horse** | 2421.3 GB/s | zebra | 2321.5 GB/s | ~equal |
| RangeRead/Start_256KB | **horse** | 2480.3 GB/s | zebra | 1636.5 GB/s | +52% |
| Read/10MB | **horse** | 82095.6 GB/s | zebra | 75615.0 GB/s | ~equal |
| Read/1KB | **horse** | 8.8 GB/s | zebra | 8.6 GB/s | ~equal |
| Read/1MB | **horse** | 9734.3 GB/s | zebra | 8258.9 GB/s | +18% |
| Read/64KB | **horse** | 629.4 GB/s | zebra | 538.1 GB/s | +17% |
| Scale/Delete/1 | **horse** | 585.5K ops/s | zebra | 170.2K ops/s | 3.4x faster |
| Scale/Delete/10 | **zebra** | 328.7K ops/s | horse | 230.8K ops/s | +42% |
| Scale/Delete/100 | **horse** | 39.5K ops/s | zebra | 33.9K ops/s | +17% |
| Scale/Delete/1000 | **horse** | 3.6K ops/s | zebra | 2.1K ops/s | +73% |
| Scale/List/1 | **horse** | 358.3K ops/s | zebra | 1 ops/s | 558270.8x faster |
| Scale/List/10 | **horse** | 230.7K ops/s | zebra | 32.3K ops/s | 7.1x faster |
| Scale/List/100 | **horse** | 45.5K ops/s | zebra | 23.9K ops/s | +91% |
| Scale/List/1000 | **horse** | 5.1K ops/s | zebra | 2.9K ops/s | +76% |
| Scale/Write/1 | **zebra** | 42.8 MB/s | horse | 27.3 MB/s | +57% |
| Scale/Write/10 | **horse** | 208.5 MB/s | zebra | 91.0 MB/s | 2.3x faster |
| Scale/Write/100 | **horse** | 288.9 MB/s | zebra | 82.4 MB/s | 3.5x faster |
| Scale/Write/1000 | **horse** | 288.1 MB/s | zebra | 41.6 MB/s | 6.9x faster |
| Stat | **horse** | 17.3M ops/s | zebra | 14.3M ops/s | +21% |
| Write/10MB | **horse** | 708.2 MB/s | zebra | 542.9 MB/s | +30% |
| Write/1KB | **zebra** | 1.8 GB/s | horse | 1.4 GB/s | +33% |
| Write/1MB | **horse** | 685.5 MB/s | zebra | 101.8 MB/s | 6.7x faster |
| Write/64KB | **zebra** | 2.5 GB/s | horse | 1.3 GB/s | +100% |

## Category Summaries

### Write Operations

**Best for Write:** horse (won 2/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | horse | 708.2 MB/s | +30% |
| Write/1KB | zebra | 1.8 GB/s | +33% |
| Write/1MB | horse | 685.5 MB/s | 6.7x faster |
| Write/64KB | zebra | 2.5 GB/s | +100% |

### Read Operations

**Best for Read:** horse (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | horse | 82095.6 GB/s | ~equal |
| Read/1KB | horse | 8.8 GB/s | ~equal |
| Read/1MB | horse | 9734.3 GB/s | +18% |
| Read/64KB | horse | 629.4 GB/s | +17% |

### ParallelWrite Operations

**Best for ParallelWrite:** zebra (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | zebra | 1.1 GB/s | ~equal |
| ParallelWrite/1KB/C10 | zebra | 507.4 MB/s | 3.1x faster |
| ParallelWrite/1KB/C50 | zebra | 161.9 MB/s | 3.7x faster |

### ParallelRead Operations

**Best for ParallelRead:** horse (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | horse | 6.9 GB/s | +13% |
| ParallelRead/1KB/C10 | horse | 5.3 GB/s | ~equal |
| ParallelRead/1KB/C50 | horse | 4.7 GB/s | 2.2x faster |

### Delete Operations

**Best for Delete:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | zebra | 3.0M ops/s | +20% |

### Stat Operations

**Best for Stat:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | horse | 17.3M ops/s | +21% |

### List Operations

**Best for List:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | horse | 135.9K ops/s | +76% |

### Copy Operations

**Best for Copy:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | zebra | 919.6 MB/s | +60% |

### Scale Operations

**Best for Scale:** horse (won 10/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | horse | 585.5K ops/s | 3.4x faster |
| Scale/Delete/10 | zebra | 328.7K ops/s | +42% |
| Scale/Delete/100 | horse | 39.5K ops/s | +17% |
| Scale/Delete/1000 | horse | 3.6K ops/s | +73% |
| Scale/List/1 | horse | 358.3K ops/s | 558270.8x faster |
| Scale/List/10 | horse | 230.7K ops/s | 7.1x faster |
| Scale/List/100 | horse | 45.5K ops/s | +91% |
| Scale/List/1000 | horse | 5.1K ops/s | +76% |
| Scale/Write/1 | zebra | 42.8 MB/s | +57% |
| Scale/Write/10 | horse | 208.5 MB/s | 2.3x faster |
| Scale/Write/100 | horse | 288.9 MB/s | 3.5x faster |
| Scale/Write/1000 | horse | 288.1 MB/s | 6.9x faster |

---

*Generated by storage benchmark CLI*
