# Storage Benchmark Summary

**Generated:** 2026-02-19T14:13:45+07:00

## Overall Winner

**horse** won 31/40 categories (78%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| horse | 31 | 78% |
| zebra | 9 | 22% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **zebra** | 913.7 MB/s | horse | 149.6 MB/s | 6.1x faster |
| Delete | **zebra** | 2.8M ops/s | horse | 2.4M ops/s | +13% |
| EdgeCase/DeepNested | **horse** | 78.3 MB/s | zebra | 55.7 MB/s | +41% |
| EdgeCase/EmptyObject | **horse** | 1.5M ops/s | zebra | 714.1K ops/s | 2.1x faster |
| EdgeCase/LongKey256 | **horse** | 87.9 MB/s | zebra | 27.8 MB/s | 3.2x faster |
| List/100 | **horse** | 126.8K ops/s | zebra | 46 ops/s | 2739.9x faster |
| MixedWorkload/Balanced_50_50 | **zebra** | 177.4 MB/s | horse | 19.5 MB/s | 9.1x faster |
| MixedWorkload/ReadHeavy_90_10 | **zebra** | 147.4 MB/s | horse | 81.3 MB/s | +81% |
| MixedWorkload/WriteHeavy_10_90 | **zebra** | 21.8 MB/s | horse | 2.8 MB/s | 7.7x faster |
| Multipart/15MB_3Parts | **horse** | 106.2 MB/s | zebra | 84.1 MB/s | +26% |
| ParallelRead/1KB/C1 | **horse** | 6.4 GB/s | zebra | 5.7 GB/s | +11% |
| ParallelRead/1KB/C10 | **horse** | 5.1 GB/s | zebra | 2.5 GB/s | 2.1x faster |
| ParallelRead/1KB/C50 | **horse** | 4.3 GB/s | zebra | 2.3 GB/s | +86% |
| ParallelWrite/1KB/C1 | **horse** | 968.2 MB/s | zebra | 851.4 MB/s | +14% |
| ParallelWrite/1KB/C10 | **zebra** | 310.2 MB/s | horse | 34.7 MB/s | 8.9x faster |
| ParallelWrite/1KB/C50 | **zebra** | 129.0 MB/s | horse | 38.0 MB/s | 3.4x faster |
| RangeRead/End_256KB | **horse** | 2431.5 GB/s | zebra | 869.1 GB/s | 2.8x faster |
| RangeRead/Middle_256KB | **horse** | 2352.8 GB/s | zebra | 887.9 GB/s | 2.6x faster |
| RangeRead/Start_256KB | **horse** | 2340.6 GB/s | zebra | 589.5 GB/s | 4.0x faster |
| Read/10MB | **horse** | 94441.6 GB/s | zebra | 51205.0 GB/s | +84% |
| Read/1KB | **horse** | 9.3 GB/s | zebra | 5.5 GB/s | +68% |
| Read/1MB | **horse** | 10270.8 GB/s | zebra | 4141.3 GB/s | 2.5x faster |
| Read/64KB | **horse** | 632.6 GB/s | zebra | 373.5 GB/s | +69% |
| Scale/Delete/1 | **horse** | 648.5K ops/s | zebra | 114.8K ops/s | 5.6x faster |
| Scale/Delete/10 | **horse** | 252.7K ops/s | zebra | 35.8K ops/s | 7.1x faster |
| Scale/Delete/100 | **horse** | 39.9K ops/s | zebra | 7.2K ops/s | 5.5x faster |
| Scale/Delete/1000 | **horse** | 3.8K ops/s | zebra | 645 ops/s | 5.9x faster |
| Scale/List/1 | **horse** | 342.8K ops/s | zebra | 4 ops/s | 83650.5x faster |
| Scale/List/10 | **horse** | 224.3K ops/s | zebra | 5 ops/s | 46677.0x faster |
| Scale/List/100 | **horse** | 28.3K ops/s | zebra | 7 ops/s | 4310.7x faster |
| Scale/List/1000 | **horse** | 4.4K ops/s | zebra | 6 ops/s | 755.7x faster |
| Scale/Write/1 | **zebra** | 36.2 MB/s | horse | 26.4 MB/s | +37% |
| Scale/Write/10 | **horse** | 206.3 MB/s | zebra | 70.4 MB/s | 2.9x faster |
| Scale/Write/100 | **horse** | 167.7 MB/s | zebra | 112.9 MB/s | +49% |
| Scale/Write/1000 | **horse** | 191.9 MB/s | zebra | 162.5 MB/s | +18% |
| Stat | **horse** | 17.2M ops/s | zebra | 11.0M ops/s | +56% |
| Write/10MB | **horse** | 414.2 MB/s | zebra | 258.0 MB/s | +61% |
| Write/1KB | **zebra** | 1.6 GB/s | horse | 1.4 GB/s | +12% |
| Write/1MB | **horse** | 769.7 MB/s | zebra | 322.8 MB/s | 2.4x faster |
| Write/64KB | **horse** | 1.5 GB/s | zebra | 523.6 MB/s | 2.9x faster |

## Category Summaries

### Write Operations

**Best for Write:** horse (won 3/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | horse | 414.2 MB/s | +61% |
| Write/1KB | zebra | 1.6 GB/s | +12% |
| Write/1MB | horse | 769.7 MB/s | 2.4x faster |
| Write/64KB | horse | 1.5 GB/s | 2.9x faster |

### Read Operations

**Best for Read:** horse (won 4/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | horse | 94441.6 GB/s | +84% |
| Read/1KB | horse | 9.3 GB/s | +68% |
| Read/1MB | horse | 10270.8 GB/s | 2.5x faster |
| Read/64KB | horse | 632.6 GB/s | +69% |

### ParallelWrite Operations

**Best for ParallelWrite:** zebra (won 2/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | horse | 968.2 MB/s | +14% |
| ParallelWrite/1KB/C10 | zebra | 310.2 MB/s | 8.9x faster |
| ParallelWrite/1KB/C50 | zebra | 129.0 MB/s | 3.4x faster |

### ParallelRead Operations

**Best for ParallelRead:** horse (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | horse | 6.4 GB/s | +11% |
| ParallelRead/1KB/C10 | horse | 5.1 GB/s | 2.1x faster |
| ParallelRead/1KB/C50 | horse | 4.3 GB/s | +86% |

### Delete Operations

**Best for Delete:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | zebra | 2.8M ops/s | +13% |

### Stat Operations

**Best for Stat:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | horse | 17.2M ops/s | +56% |

### List Operations

**Best for List:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | horse | 126.8K ops/s | 2739.9x faster |

### Copy Operations

**Best for Copy:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | zebra | 913.7 MB/s | 6.1x faster |

### Scale Operations

**Best for Scale:** horse (won 11/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | horse | 648.5K ops/s | 5.6x faster |
| Scale/Delete/10 | horse | 252.7K ops/s | 7.1x faster |
| Scale/Delete/100 | horse | 39.9K ops/s | 5.5x faster |
| Scale/Delete/1000 | horse | 3.8K ops/s | 5.9x faster |
| Scale/List/1 | horse | 342.8K ops/s | 83650.5x faster |
| Scale/List/10 | horse | 224.3K ops/s | 46677.0x faster |
| Scale/List/100 | horse | 28.3K ops/s | 4310.7x faster |
| Scale/List/1000 | horse | 4.4K ops/s | 755.7x faster |
| Scale/Write/1 | zebra | 36.2 MB/s | +37% |
| Scale/Write/10 | horse | 206.3 MB/s | 2.9x faster |
| Scale/Write/100 | horse | 167.7 MB/s | +49% |
| Scale/Write/1000 | horse | 191.9 MB/s | +18% |

---

*Generated by storage benchmark CLI*
