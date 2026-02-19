# Storage Benchmark Summary

**Generated:** 2026-02-19T13:12:07+07:00

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
| Copy/1KB | **zebra** | 2.1 GB/s | horse | 583.5 MB/s | 3.6x faster |
| Delete | **zebra** | 5.0M ops/s | horse | 2.5M ops/s | 2.0x faster |
| EdgeCase/DeepNested | **horse** | 129.8 MB/s | zebra | 129.6 MB/s | ~equal |
| EdgeCase/EmptyObject | **horse** | 1.8M ops/s | zebra | 1.3M ops/s | +36% |
| EdgeCase/LongKey256 | **horse** | 100.0 MB/s | zebra | 81.6 MB/s | +23% |
| List/100 | **horse** | 119.2K ops/s | zebra | 20 ops/s | 5993.5x faster |
| MixedWorkload/Balanced_50_50 | **horse** | 10.2 MB/s | zebra | 7.7 MB/s | +32% |
| MixedWorkload/ReadHeavy_90_10 | **horse** | 145.9 MB/s | zebra | 41.8 MB/s | 3.5x faster |
| MixedWorkload/WriteHeavy_10_90 | **horse** | 5.0 MB/s | zebra | 3.2 MB/s | +57% |
| Multipart/15MB_3Parts | **horse** | 339.3 MB/s | zebra | 256.9 MB/s | +32% |
| ParallelRead/1KB/C1 | **zebra** | 6.3 GB/s | horse | 6.2 GB/s | ~equal |
| ParallelRead/1KB/C10 | **horse** | 4.8 GB/s | zebra | 4.0 GB/s | +20% |
| ParallelRead/1KB/C50 | **horse** | 4.1 GB/s | zebra | 2.0 GB/s | 2.1x faster |
| ParallelWrite/1KB/C1 | **zebra** | 1.2 GB/s | horse | 1.0 GB/s | +25% |
| ParallelWrite/1KB/C10 | **zebra** | 683.3 MB/s | horse | 143.1 MB/s | 4.8x faster |
| ParallelWrite/1KB/C50 | **zebra** | 196.1 MB/s | horse | 50.8 MB/s | 3.9x faster |
| RangeRead/End_256KB | **horse** | 2252.4 GB/s | zebra | 1458.9 GB/s | +54% |
| RangeRead/Middle_256KB | **horse** | 2107.8 GB/s | zebra | 1731.2 GB/s | +22% |
| RangeRead/Start_256KB | **horse** | 2156.8 GB/s | zebra | 1684.9 GB/s | +28% |
| Read/10MB | **horse** | 41.7 GB/s | zebra | 31.7 GB/s | +32% |
| Read/1KB | **horse** | 9.6 GB/s | zebra | 8.2 GB/s | +17% |
| Read/1MB | **horse** | 9430.9 GB/s | zebra | 16.3 GB/s | 578.0x faster |
| Read/64KB | **horse** | 644.8 GB/s | zebra | 405.4 GB/s | +59% |
| Scale/Delete/1 | **horse** | 666.7K ops/s | zebra | 275.9K ops/s | 2.4x faster |
| Scale/Delete/10 | **horse** | 203.4K ops/s | zebra | 97.2K ops/s | 2.1x faster |
| Scale/Delete/100 | **horse** | 36.3K ops/s | zebra | 26.7K ops/s | +36% |
| Scale/Delete/1000 | **horse** | 3.9K ops/s | zebra | 3.1K ops/s | +26% |
| Scale/List/1 | **horse** | 320.0K ops/s | zebra | 7 ops/s | 48657.3x faster |
| Scale/List/10 | **horse** | 210.5K ops/s | zebra | 6 ops/s | 33359.2x faster |
| Scale/List/100 | **horse** | 30.8K ops/s | zebra | 7 ops/s | 4392.9x faster |
| Scale/List/1000 | **horse** | 5.8K ops/s | zebra | 7 ops/s | 831.3x faster |
| Scale/Write/1 | **zebra** | 106.6 MB/s | horse | 30.0 MB/s | 3.5x faster |
| Scale/Write/10 | **horse** | 259.3 MB/s | zebra | 244.1 MB/s | ~equal |
| Scale/Write/100 | **zebra** | 494.9 MB/s | horse | 290.9 MB/s | +70% |
| Scale/Write/1000 | **zebra** | 509.9 MB/s | horse | 422.9 MB/s | +21% |
| Stat | **zebra** | 14.6M ops/s | horse | 12.3M ops/s | +19% |
| Write/10MB | **horse** | 888.2 MB/s | zebra | 585.6 MB/s | +52% |
| Write/1KB | **zebra** | 2.5 GB/s | horse | 1.5 GB/s | +69% |
| Write/1MB | **horse** | 606.7 MB/s | zebra | 70.6 MB/s | 8.6x faster |
| Write/64KB | **horse** | 1.0 GB/s | zebra | 320.7 MB/s | 3.3x faster |

## Category Summaries

### Write Operations

**Best for Write:** horse (won 3/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | horse | 888.2 MB/s | +52% |
| Write/1KB | zebra | 2.5 GB/s | +69% |
| Write/1MB | horse | 606.7 MB/s | 8.6x faster |
| Write/64KB | horse | 1.0 GB/s | 3.3x faster |

### Read Operations

**Best for Read:** horse (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | horse | 41.7 GB/s | +32% |
| Read/1KB | horse | 9.6 GB/s | +17% |
| Read/1MB | horse | 9430.9 GB/s | 578.0x faster |
| Read/64KB | horse | 644.8 GB/s | +59% |

### ParallelWrite Operations

**Best for ParallelWrite:** zebra (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | zebra | 1.2 GB/s | +25% |
| ParallelWrite/1KB/C10 | zebra | 683.3 MB/s | 4.8x faster |
| ParallelWrite/1KB/C50 | zebra | 196.1 MB/s | 3.9x faster |

### ParallelRead Operations

**Best for ParallelRead:** horse (won 2/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | zebra | 6.3 GB/s | ~equal |
| ParallelRead/1KB/C10 | horse | 4.8 GB/s | +20% |
| ParallelRead/1KB/C50 | horse | 4.1 GB/s | 2.1x faster |

### Delete Operations

**Best for Delete:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | zebra | 5.0M ops/s | 2.0x faster |

### Stat Operations

**Best for Stat:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | zebra | 14.6M ops/s | +19% |

### List Operations

**Best for List:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | horse | 119.2K ops/s | 5993.5x faster |

### Copy Operations

**Best for Copy:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | zebra | 2.1 GB/s | 3.6x faster |

### Scale Operations

**Best for Scale:** horse (won 9/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | horse | 666.7K ops/s | 2.4x faster |
| Scale/Delete/10 | horse | 203.4K ops/s | 2.1x faster |
| Scale/Delete/100 | horse | 36.3K ops/s | +36% |
| Scale/Delete/1000 | horse | 3.9K ops/s | +26% |
| Scale/List/1 | horse | 320.0K ops/s | 48657.3x faster |
| Scale/List/10 | horse | 210.5K ops/s | 33359.2x faster |
| Scale/List/100 | horse | 30.8K ops/s | 4392.9x faster |
| Scale/List/1000 | horse | 5.8K ops/s | 831.3x faster |
| Scale/Write/1 | zebra | 106.6 MB/s | 3.5x faster |
| Scale/Write/10 | horse | 259.3 MB/s | ~equal |
| Scale/Write/100 | zebra | 494.9 MB/s | +70% |
| Scale/Write/1000 | zebra | 509.9 MB/s | +21% |

---

*Generated by storage benchmark CLI*
