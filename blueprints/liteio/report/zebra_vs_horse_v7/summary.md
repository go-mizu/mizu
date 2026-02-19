# Storage Benchmark Summary

**Generated:** 2026-02-19T14:21:53+07:00

## Overall Winner

**horse** won 26/40 categories (65%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| horse | 26 | 65% |
| zebra | 14 | 35% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **zebra** | 895.4 MB/s | horse | 80.4 MB/s | 11.1x faster |
| Delete | **zebra** | 2.7M ops/s | horse | 2.5M ops/s | ~equal |
| EdgeCase/DeepNested | **zebra** | 101.0 MB/s | horse | 43.0 MB/s | 2.3x faster |
| EdgeCase/EmptyObject | **horse** | 1.3M ops/s | zebra | 681.1K ops/s | +92% |
| EdgeCase/LongKey256 | **horse** | 94.2 MB/s | zebra | 56.9 MB/s | +66% |
| List/100 | **horse** | 134.1K ops/s | zebra | 32.5K ops/s | 4.1x faster |
| MixedWorkload/Balanced_50_50 | **horse** | 18.7 MB/s | zebra | 4.4 MB/s | 4.3x faster |
| MixedWorkload/ReadHeavy_90_10 | **zebra** | 639.9 MB/s | horse | 53.3 MB/s | 12.0x faster |
| MixedWorkload/WriteHeavy_10_90 | **horse** | 2.7 MB/s | zebra | 1.8 MB/s | +48% |
| Multipart/15MB_3Parts | **horse** | 193.5 MB/s | zebra | 29.9 MB/s | 6.5x faster |
| ParallelRead/1KB/C1 | **horse** | 6.2 GB/s | zebra | 4.2 GB/s | +47% |
| ParallelRead/1KB/C10 | **horse** | 5.1 GB/s | zebra | 2.4 GB/s | 2.1x faster |
| ParallelRead/1KB/C50 | **horse** | 4.6 GB/s | zebra | 2.4 GB/s | +91% |
| ParallelWrite/1KB/C1 | **horse** | 994.1 MB/s | zebra | 788.0 MB/s | +26% |
| ParallelWrite/1KB/C10 | **zebra** | 318.7 MB/s | horse | 107.8 MB/s | 3.0x faster |
| ParallelWrite/1KB/C50 | **zebra** | 75.1 MB/s | horse | 43.7 MB/s | +72% |
| RangeRead/End_256KB | **horse** | 2490.3 GB/s | zebra | 1644.7 GB/s | +51% |
| RangeRead/Middle_256KB | **horse** | 2327.5 GB/s | zebra | 1933.8 GB/s | +20% |
| RangeRead/Start_256KB | **horse** | 2450.9 GB/s | zebra | 1479.7 GB/s | +66% |
| Read/10MB | **horse** | 88710.2 GB/s | zebra | 78811.2 GB/s | +13% |
| Read/1KB | **horse** | 8.5 GB/s | zebra | 7.8 GB/s | ~equal |
| Read/1MB | **horse** | 10435.1 GB/s | zebra | 8858.3 GB/s | +18% |
| Read/64KB | **horse** | 653.5 GB/s | zebra | 527.0 GB/s | +24% |
| Scale/Delete/1 | **horse** | 545.3K ops/s | zebra | 69.0K ops/s | 7.9x faster |
| Scale/Delete/10 | **zebra** | 43.3K ops/s | horse | 38.2K ops/s | +13% |
| Scale/Delete/100 | **zebra** | 6.7K ops/s | horse | 5.0K ops/s | +34% |
| Scale/Delete/1000 | **horse** | 723 ops/s | zebra | 564 ops/s | +28% |
| Scale/List/1 | **horse** | 237.6K ops/s | zebra | 1 ops/s | 427718.6x faster |
| Scale/List/10 | **horse** | 151.9K ops/s | zebra | 0 ops/s | 347024.0x faster |
| Scale/List/100 | **horse** | 6.7K ops/s | zebra | 0 ops/s | 24353.0x faster |
| Scale/List/1000 | **horse** | 1.1K ops/s | zebra | 0 ops/s | 4138.8x faster |
| Scale/Write/1 | **zebra** | 33.9 MB/s | horse | 13.8 MB/s | 2.5x faster |
| Scale/Write/10 | **horse** | 109.5 MB/s | zebra | 87.7 MB/s | +25% |
| Scale/Write/100 | **zebra** | 185.7 MB/s | horse | 129.9 MB/s | +43% |
| Scale/Write/1000 | **zebra** | 123.8 MB/s | horse | 105.8 MB/s | +17% |
| Stat | **horse** | 15.7M ops/s | zebra | 12.6M ops/s | +25% |
| Write/10MB | **horse** | 1.2 GB/s | zebra | 306.0 MB/s | 3.8x faster |
| Write/1KB | **zebra** | 1.8 GB/s | horse | 1.4 GB/s | +31% |
| Write/1MB | **zebra** | 1.3 GB/s | horse | 886.6 MB/s | +41% |
| Write/64KB | **zebra** | 2.0 GB/s | horse | 1.5 GB/s | +41% |

## Category Summaries

### Write Operations

**Best for Write:** zebra (won 3/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | horse | 1.2 GB/s | 3.8x faster |
| Write/1KB | zebra | 1.8 GB/s | +31% |
| Write/1MB | zebra | 1.3 GB/s | +41% |
| Write/64KB | zebra | 2.0 GB/s | +41% |

### Read Operations

**Best for Read:** horse (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | horse | 88710.2 GB/s | +13% |
| Read/1KB | horse | 8.5 GB/s | ~equal |
| Read/1MB | horse | 10435.1 GB/s | +18% |
| Read/64KB | horse | 653.5 GB/s | +24% |

### ParallelWrite Operations

**Best for ParallelWrite:** zebra (won 2/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | horse | 994.1 MB/s | +26% |
| ParallelWrite/1KB/C10 | zebra | 318.7 MB/s | 3.0x faster |
| ParallelWrite/1KB/C50 | zebra | 75.1 MB/s | +72% |

### ParallelRead Operations

**Best for ParallelRead:** horse (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | horse | 6.2 GB/s | +47% |
| ParallelRead/1KB/C10 | horse | 5.1 GB/s | 2.1x faster |
| ParallelRead/1KB/C50 | horse | 4.6 GB/s | +91% |

### Delete Operations

**Best for Delete:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | zebra | 2.7M ops/s | ~equal |

### Stat Operations

**Best for Stat:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | horse | 15.7M ops/s | +25% |

### List Operations

**Best for List:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | horse | 134.1K ops/s | 4.1x faster |

### Copy Operations

**Best for Copy:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | zebra | 895.4 MB/s | 11.1x faster |

### Scale Operations

**Best for Scale:** horse (won 7/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | horse | 545.3K ops/s | 7.9x faster |
| Scale/Delete/10 | zebra | 43.3K ops/s | +13% |
| Scale/Delete/100 | zebra | 6.7K ops/s | +34% |
| Scale/Delete/1000 | horse | 723 ops/s | +28% |
| Scale/List/1 | horse | 237.6K ops/s | 427718.6x faster |
| Scale/List/10 | horse | 151.9K ops/s | 347024.0x faster |
| Scale/List/100 | horse | 6.7K ops/s | 24353.0x faster |
| Scale/List/1000 | horse | 1.1K ops/s | 4138.8x faster |
| Scale/Write/1 | zebra | 33.9 MB/s | 2.5x faster |
| Scale/Write/10 | horse | 109.5 MB/s | +25% |
| Scale/Write/100 | zebra | 185.7 MB/s | +43% |
| Scale/Write/1000 | zebra | 123.8 MB/s | +17% |

---

*Generated by storage benchmark CLI*
