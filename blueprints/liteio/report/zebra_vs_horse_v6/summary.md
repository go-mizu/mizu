# Storage Benchmark Summary

**Generated:** 2026-02-19T14:18:02+07:00

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
| Copy/1KB | **zebra** | 1.3 GB/s | horse | 733.6 MB/s | +78% |
| Delete | **zebra** | 2.8M ops/s | horse | 2.4M ops/s | +17% |
| EdgeCase/DeepNested | **zebra** | 151.6 MB/s | horse | 132.6 MB/s | +14% |
| EdgeCase/EmptyObject | **horse** | 1.5M ops/s | zebra | 1.4M ops/s | ~equal |
| EdgeCase/LongKey256 | **zebra** | 107.2 MB/s | horse | 106.6 MB/s | ~equal |
| List/100 | **horse** | 74.2K ops/s | zebra | 58 ops/s | 1282.8x faster |
| MixedWorkload/Balanced_50_50 | **horse** | 21.1 MB/s | zebra | 5.3 MB/s | 4.0x faster |
| MixedWorkload/ReadHeavy_90_10 | **zebra** | 7.2 GB/s | horse | 6.9 MB/s | 1039.5x faster |
| MixedWorkload/WriteHeavy_10_90 | **horse** | 4.6 MB/s | zebra | 1.2 MB/s | 3.8x faster |
| Multipart/15MB_3Parts | **horse** | 231.2 MB/s | zebra | 181.9 MB/s | +27% |
| ParallelRead/1KB/C1 | **horse** | 6.5 GB/s | zebra | 5.2 GB/s | +25% |
| ParallelRead/1KB/C10 | **horse** | 3.7 GB/s | zebra | 3.0 GB/s | +27% |
| ParallelRead/1KB/C50 | **horse** | 3.2 GB/s | zebra | 2.8 GB/s | +12% |
| ParallelWrite/1KB/C1 | **horse** | 1.0 GB/s | zebra | 860.8 MB/s | +20% |
| ParallelWrite/1KB/C10 | **zebra** | 420.8 MB/s | horse | 132.5 MB/s | 3.2x faster |
| ParallelWrite/1KB/C50 | **zebra** | 169.7 MB/s | horse | 39.2 MB/s | 4.3x faster |
| RangeRead/End_256KB | **horse** | 2418.0 GB/s | zebra | 2083.7 GB/s | +16% |
| RangeRead/Middle_256KB | **horse** | 2362.1 GB/s | zebra | 2131.5 GB/s | +11% |
| RangeRead/Start_256KB | **horse** | 2155.4 GB/s | zebra | 1821.6 GB/s | +18% |
| Read/10MB | **horse** | 66648.1 GB/s | zebra | 65036.7 GB/s | ~equal |
| Read/1KB | **zebra** | 8.4 GB/s | horse | 6.1 GB/s | +39% |
| Read/1MB | **horse** | 7106.7 GB/s | zebra | 6703.4 GB/s | ~equal |
| Read/64KB | **zebra** | 446.5 GB/s | horse | 443.8 GB/s | ~equal |
| Scale/Delete/1 | **horse** | 685.4K ops/s | zebra | 216.2K ops/s | 3.2x faster |
| Scale/Delete/10 | **horse** | 242.4K ops/s | zebra | 65.6K ops/s | 3.7x faster |
| Scale/Delete/100 | **horse** | 40.8K ops/s | zebra | 11.3K ops/s | 3.6x faster |
| Scale/Delete/1000 | **horse** | 3.8K ops/s | zebra | 1.4K ops/s | 2.7x faster |
| Scale/List/1 | **horse** | 342.8K ops/s | zebra | 8 ops/s | 42324.7x faster |
| Scale/List/10 | **horse** | 258.1K ops/s | zebra | 8 ops/s | 31086.7x faster |
| Scale/List/100 | **horse** | 56.6K ops/s | zebra | 9 ops/s | 6454.5x faster |
| Scale/List/1000 | **horse** | 5.5K ops/s | zebra | 9 ops/s | 626.2x faster |
| Scale/Write/1 | **zebra** | 37.1 MB/s | horse | 25.6 MB/s | +45% |
| Scale/Write/10 | **horse** | 210.0 MB/s | zebra | 133.2 MB/s | +58% |
| Scale/Write/100 | **horse** | 330.5 MB/s | zebra | 174.1 MB/s | +90% |
| Scale/Write/1000 | **horse** | 403.3 MB/s | zebra | 281.8 MB/s | +43% |
| Stat | **zebra** | 13.0M ops/s | horse | 10.8M ops/s | +20% |
| Write/10MB | **horse** | 1.4 GB/s | zebra | 403.1 MB/s | 3.6x faster |
| Write/1KB | **zebra** | 1.7 GB/s | horse | 752.5 MB/s | 2.3x faster |
| Write/1MB | **horse** | 861.3 MB/s | zebra | 830.1 MB/s | ~equal |
| Write/64KB | **zebra** | 1.1 GB/s | horse | 931.4 MB/s | +16% |

## Category Summaries

### Write Operations

**Best for Write:** horse (won 2/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | horse | 1.4 GB/s | 3.6x faster |
| Write/1KB | zebra | 1.7 GB/s | 2.3x faster |
| Write/1MB | horse | 861.3 MB/s | ~equal |
| Write/64KB | zebra | 1.1 GB/s | +16% |

### Read Operations

**Best for Read:** horse (won 2/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | horse | 66648.1 GB/s | ~equal |
| Read/1KB | zebra | 8.4 GB/s | +39% |
| Read/1MB | horse | 7106.7 GB/s | ~equal |
| Read/64KB | zebra | 446.5 GB/s | ~equal |

### ParallelWrite Operations

**Best for ParallelWrite:** zebra (won 2/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | horse | 1.0 GB/s | +20% |
| ParallelWrite/1KB/C10 | zebra | 420.8 MB/s | 3.2x faster |
| ParallelWrite/1KB/C50 | zebra | 169.7 MB/s | 4.3x faster |

### ParallelRead Operations

**Best for ParallelRead:** horse (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | horse | 6.5 GB/s | +25% |
| ParallelRead/1KB/C10 | horse | 3.7 GB/s | +27% |
| ParallelRead/1KB/C50 | horse | 3.2 GB/s | +12% |

### Delete Operations

**Best for Delete:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | zebra | 2.8M ops/s | +17% |

### Stat Operations

**Best for Stat:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | zebra | 13.0M ops/s | +20% |

### List Operations

**Best for List:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | horse | 74.2K ops/s | 1282.8x faster |

### Copy Operations

**Best for Copy:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | zebra | 1.3 GB/s | +78% |

### Scale Operations

**Best for Scale:** horse (won 11/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | horse | 685.4K ops/s | 3.2x faster |
| Scale/Delete/10 | horse | 242.4K ops/s | 3.7x faster |
| Scale/Delete/100 | horse | 40.8K ops/s | 3.6x faster |
| Scale/Delete/1000 | horse | 3.8K ops/s | 2.7x faster |
| Scale/List/1 | horse | 342.8K ops/s | 42324.7x faster |
| Scale/List/10 | horse | 258.1K ops/s | 31086.7x faster |
| Scale/List/100 | horse | 56.6K ops/s | 6454.5x faster |
| Scale/List/1000 | horse | 5.5K ops/s | 626.2x faster |
| Scale/Write/1 | zebra | 37.1 MB/s | +45% |
| Scale/Write/10 | horse | 210.0 MB/s | +58% |
| Scale/Write/100 | horse | 330.5 MB/s | +90% |
| Scale/Write/1000 | horse | 403.3 MB/s | +43% |

---

*Generated by storage benchmark CLI*
