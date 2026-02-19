# Storage Benchmark Summary

**Generated:** 2026-02-19T13:35:59+07:00

## Overall Winner

**horse** won 25/40 categories (62%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| horse | 25 | 62% |
| zebra | 15 | 38% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **zebra** | 1.5 GB/s | horse | 661.1 MB/s | 2.3x faster |
| Delete | **zebra** | 3.0M ops/s | horse | 2.0M ops/s | +52% |
| EdgeCase/DeepNested | **horse** | 130.4 MB/s | zebra | 52.2 MB/s | 2.5x faster |
| EdgeCase/EmptyObject | **horse** | 1.1M ops/s | zebra | 1.0M ops/s | ~equal |
| EdgeCase/LongKey256 | **horse** | 104.8 MB/s | zebra | 56.9 MB/s | +84% |
| List/100 | **horse** | 135.8K ops/s | zebra | 25 ops/s | 5527.0x faster |
| MixedWorkload/Balanced_50_50 | **zebra** | 39.7 MB/s | horse | 9.4 MB/s | 4.2x faster |
| MixedWorkload/ReadHeavy_90_10 | **zebra** | 229.1 MB/s | horse | 130.8 MB/s | +75% |
| MixedWorkload/WriteHeavy_10_90 | **zebra** | 12.8 MB/s | horse | 3.6 MB/s | 3.6x faster |
| Multipart/15MB_3Parts | **zebra** | 452.8 MB/s | horse | 374.2 MB/s | +21% |
| ParallelRead/1KB/C1 | **zebra** | 6.4 GB/s | horse | 5.1 GB/s | +25% |
| ParallelRead/1KB/C10 | **zebra** | 3.6 GB/s | horse | 2.6 GB/s | +40% |
| ParallelRead/1KB/C50 | **horse** | 3.1 GB/s | zebra | 2.4 GB/s | +27% |
| ParallelWrite/1KB/C1 | **zebra** | 1.0 GB/s | horse | 792.0 MB/s | +28% |
| ParallelWrite/1KB/C10 | **zebra** | 484.5 MB/s | horse | 111.7 MB/s | 4.3x faster |
| ParallelWrite/1KB/C50 | **zebra** | 78.9 MB/s | horse | 24.4 MB/s | 3.2x faster |
| RangeRead/End_256KB | **horse** | 2492.0 GB/s | zebra | 1514.1 GB/s | +65% |
| RangeRead/Middle_256KB | **horse** | 2332.7 GB/s | zebra | 2045.0 GB/s | +14% |
| RangeRead/Start_256KB | **horse** | 2027.7 GB/s | zebra | 1781.8 GB/s | +14% |
| Read/10MB | **zebra** | 167.2 GB/s | horse | 16.4 GB/s | 10.2x faster |
| Read/1KB | **horse** | 9.4 GB/s | zebra | 3.5 GB/s | 2.7x faster |
| Read/1MB | **zebra** | 697.5 GB/s | horse | 19.7 GB/s | 35.5x faster |
| Read/64KB | **horse** | 632.1 GB/s | zebra | 513.1 GB/s | +23% |
| Scale/Delete/1 | **horse** | 705.7K ops/s | zebra | 155.8K ops/s | 4.5x faster |
| Scale/Delete/10 | **horse** | 242.4K ops/s | zebra | 40.2K ops/s | 6.0x faster |
| Scale/Delete/100 | **horse** | 34.4K ops/s | zebra | 9.4K ops/s | 3.7x faster |
| Scale/Delete/1000 | **horse** | 3.5K ops/s | zebra | 1.5K ops/s | 2.4x faster |
| Scale/List/1 | **horse** | 363.6K ops/s | zebra | 5 ops/s | 76678.4x faster |
| Scale/List/10 | **horse** | 218.2K ops/s | zebra | 6 ops/s | 37488.4x faster |
| Scale/List/100 | **horse** | 54.8K ops/s | zebra | 6 ops/s | 9325.9x faster |
| Scale/List/1000 | **horse** | 5.4K ops/s | zebra | 6 ops/s | 960.3x faster |
| Scale/Write/1 | **horse** | 25.6 MB/s | zebra | 18.2 MB/s | +41% |
| Scale/Write/10 | **horse** | 210.8 MB/s | zebra | 126.0 MB/s | +67% |
| Scale/Write/100 | **horse** | 317.6 MB/s | zebra | 112.4 MB/s | 2.8x faster |
| Scale/Write/1000 | **horse** | 376.3 MB/s | zebra | 211.8 MB/s | +78% |
| Stat | **horse** | 15.4M ops/s | zebra | 7.0M ops/s | 2.2x faster |
| Write/10MB | **horse** | 979.0 MB/s | zebra | 567.7 MB/s | +72% |
| Write/1KB | **zebra** | 1.9 GB/s | horse | 1.4 GB/s | +38% |
| Write/1MB | **horse** | 1.1 GB/s | zebra | 316.6 MB/s | 3.4x faster |
| Write/64KB | **zebra** | 2.2 GB/s | horse | 849.3 MB/s | 2.6x faster |

## Category Summaries

### Write Operations

**Best for Write:** horse (won 2/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/10MB | horse | 979.0 MB/s | +72% |
| Write/1KB | zebra | 1.9 GB/s | +38% |
| Write/1MB | horse | 1.1 GB/s | 3.4x faster |
| Write/64KB | zebra | 2.2 GB/s | 2.6x faster |

### Read Operations

**Best for Read:** zebra (won 2/4)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/10MB | zebra | 167.2 GB/s | 10.2x faster |
| Read/1KB | horse | 9.4 GB/s | 2.7x faster |
| Read/1MB | zebra | 697.5 GB/s | 35.5x faster |
| Read/64KB | horse | 632.1 GB/s | +23% |

### ParallelWrite Operations

**Best for ParallelWrite:** zebra (won 3/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | zebra | 1.0 GB/s | +28% |
| ParallelWrite/1KB/C10 | zebra | 484.5 MB/s | 4.3x faster |
| ParallelWrite/1KB/C50 | zebra | 78.9 MB/s | 3.2x faster |

### ParallelRead Operations

**Best for ParallelRead:** zebra (won 2/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | zebra | 6.4 GB/s | +25% |
| ParallelRead/1KB/C10 | zebra | 3.6 GB/s | +40% |
| ParallelRead/1KB/C50 | horse | 3.1 GB/s | +27% |

### Delete Operations

**Best for Delete:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | zebra | 3.0M ops/s | +52% |

### Stat Operations

**Best for Stat:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | horse | 15.4M ops/s | 2.2x faster |

### List Operations

**Best for List:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | horse | 135.8K ops/s | 5527.0x faster |

### Copy Operations

**Best for Copy:** zebra (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | zebra | 1.5 GB/s | 2.3x faster |

### Scale Operations

**Best for Scale:** horse (won 12/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/1 | horse | 705.7K ops/s | 4.5x faster |
| Scale/Delete/10 | horse | 242.4K ops/s | 6.0x faster |
| Scale/Delete/100 | horse | 34.4K ops/s | 3.7x faster |
| Scale/Delete/1000 | horse | 3.5K ops/s | 2.4x faster |
| Scale/List/1 | horse | 363.6K ops/s | 76678.4x faster |
| Scale/List/10 | horse | 218.2K ops/s | 37488.4x faster |
| Scale/List/100 | horse | 54.8K ops/s | 9325.9x faster |
| Scale/List/1000 | horse | 5.4K ops/s | 960.3x faster |
| Scale/Write/1 | horse | 25.6 MB/s | +41% |
| Scale/Write/10 | horse | 210.8 MB/s | +67% |
| Scale/Write/100 | horse | 317.6 MB/s | 2.8x faster |
| Scale/Write/1000 | horse | 376.3 MB/s | +78% |

---

*Generated by storage benchmark CLI*
