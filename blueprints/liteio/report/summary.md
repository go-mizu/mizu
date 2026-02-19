# Storage Benchmark Summary

**Generated:** 2026-02-19T07:31:08+07:00

## Overall Winner

**turtle** won 43/48 categories (90%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| turtle | 43 | 90% |
| local | 5 | 10% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **turtle** | 1.0 GB/s | local | 1.8 MB/s | 577.2x faster |
| Delete | **turtle** | 2.3M ops/s | local | 22.6K ops/s | 102.3x faster |
| EdgeCase/DeepNested | **turtle** | 80.7 MB/s | local | 0.1 MB/s | 707.3x faster |
| EdgeCase/EmptyObject | **local** | 2.7K ops/s | turtle | 665 ops/s | 4.0x faster |
| EdgeCase/LongKey256 | **turtle** | 54.2 MB/s | local | 0.2 MB/s | 334.0x faster |
| List/100 | **turtle** | 92.5K ops/s | local | 4.6K ops/s | 20.1x faster |
| MixedWorkload/Balanced_50_50 | **turtle** | 1.5 MB/s | local | 0.7 MB/s | 2.2x faster |
| MixedWorkload/ReadHeavy_90_10 | **turtle** | 10.4 MB/s | local | 4.4 MB/s | 2.4x faster |
| MixedWorkload/WriteHeavy_10_90 | **turtle** | 0.8 MB/s | local | 0.3 MB/s | 2.6x faster |
| Multipart/15MB_3Parts | **turtle** | 338.4 MB/s | local | 77.0 MB/s | 4.4x faster |
| ParallelRead/1KB/C1 | **turtle** | 4.5 GB/s | local | 1.5 GB/s | 2.9x faster |
| ParallelRead/1KB/C10 | **turtle** | 3.5 GB/s | local | 944.3 MB/s | 3.7x faster |
| ParallelRead/1KB/C100 | **turtle** | 2.8 GB/s | local | 611.2 MB/s | 4.5x faster |
| ParallelRead/1KB/C200 | **turtle** | 2.5 GB/s | local | 327.4 MB/s | 7.7x faster |
| ParallelRead/1KB/C25 | **turtle** | 3.4 GB/s | local | 894.1 MB/s | 3.8x faster |
| ParallelRead/1KB/C50 | **turtle** | 3.2 GB/s | local | 752.0 MB/s | 4.2x faster |
| ParallelWrite/1KB/C1 | **turtle** | 26.5 MB/s | local | 3.0 MB/s | 8.9x faster |
| ParallelWrite/1KB/C10 | **turtle** | 65.3 MB/s | local | 0.7 MB/s | 89.6x faster |
| ParallelWrite/1KB/C100 | **turtle** | 1.0 MB/s | local | 0.0 MB/s | 21.3x faster |
| ParallelWrite/1KB/C200 | **turtle** | 1.8 MB/s | local | 0.0 MB/s | 80.5x faster |
| ParallelWrite/1KB/C25 | **turtle** | 1.9 MB/s | local | 0.2 MB/s | 9.5x faster |
| ParallelWrite/1KB/C50 | **turtle** | 25.1 MB/s | local | 0.1 MB/s | 236.3x faster |
| RangeRead/End_256KB | **turtle** | 1927.9 GB/s | local | 23.7 GB/s | 81.5x faster |
| RangeRead/Middle_256KB | **turtle** | 1908.4 GB/s | local | 11.3 GB/s | 168.6x faster |
| RangeRead/Start_256KB | **turtle** | 1838.2 GB/s | local | 11.0 GB/s | 167.5x faster |
| Read/100MB | **turtle** | 825265.6 GB/s | local | 14.9 GB/s | 55365.0x faster |
| Read/10MB | **turtle** | 77683.3 GB/s | local | 11.2 GB/s | 6945.7x faster |
| Read/1KB | **local** | 7.0 GB/s | turtle | 6.9 GB/s | ~equal |
| Read/1MB | **turtle** | 7907.3 GB/s | local | 95.1 GB/s | 83.2x faster |
| Read/64KB | **turtle** | 476.3 GB/s | local | 22.6 GB/s | 21.0x faster |
| Scale/Delete/10 | **turtle** | 263.8K ops/s | local | 1.8K ops/s | 149.3x faster |
| Scale/Delete/100 | **turtle** | 39.9K ops/s | local | 266 ops/s | 149.8x faster |
| Scale/Delete/1000 | **turtle** | 1.5K ops/s | local | 20 ops/s | 76.8x faster |
| Scale/Delete/10000 | **turtle** | 163 ops/s | local | 2 ops/s | 105.5x faster |
| Scale/List/10 | **turtle** | 122.4K ops/s | local | 8.7K ops/s | 14.1x faster |
| Scale/List/100 | **turtle** | 41.0K ops/s | local | 3.1K ops/s | 13.2x faster |
| Scale/List/1000 | **turtle** | 4.8K ops/s | local | 309 ops/s | 15.4x faster |
| Scale/List/10000 | **turtle** | 277 ops/s | local | 31 ops/s | 9.1x faster |
| Scale/Write/10 | **turtle** | 114.9 MB/s | local | 0.0 MB/s | 3382.4x faster |
| Scale/Write/100 | **turtle** | 96.1 MB/s | local | 0.1 MB/s | 695.2x faster |
| Scale/Write/1000 | **turtle** | 309.8 MB/s | local | 0.4 MB/s | 696.1x faster |
| Scale/Write/10000 | **turtle** | 174.9 MB/s | local | 0.6 MB/s | 310.0x faster |
| Stat | **turtle** | 12.6M ops/s | local | 5.3M ops/s | 2.4x faster |
| Write/100MB | **local** | 1.1 GB/s | turtle | 273.6 MB/s | 4.1x faster |
| Write/10MB | **local** | 1.1 GB/s | turtle | 210.2 MB/s | 5.1x faster |
| Write/1KB | **turtle** | 75.9 MB/s | local | 2.2 MB/s | 34.0x faster |
| Write/1MB | **local** | 639.8 MB/s | turtle | 550.0 MB/s | +16% |
| Write/64KB | **turtle** | 412.5 MB/s | local | 149.6 MB/s | 2.8x faster |

## Category Summaries

### Write Operations

**Best for Write:** local (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | local | 1.1 GB/s | 4.1x faster |
| Write/10MB | local | 1.1 GB/s | 5.1x faster |
| Write/1KB | turtle | 75.9 MB/s | 34.0x faster |
| Write/1MB | local | 639.8 MB/s | +16% |
| Write/64KB | turtle | 412.5 MB/s | 2.8x faster |

### Read Operations

**Best for Read:** turtle (won 4/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | turtle | 825265.6 GB/s | 55365.0x faster |
| Read/10MB | turtle | 77683.3 GB/s | 6945.7x faster |
| Read/1KB | local | 7.0 GB/s | ~equal |
| Read/1MB | turtle | 7907.3 GB/s | 83.2x faster |
| Read/64KB | turtle | 476.3 GB/s | 21.0x faster |

### ParallelWrite Operations

**Best for ParallelWrite:** turtle (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | turtle | 26.5 MB/s | 8.9x faster |
| ParallelWrite/1KB/C10 | turtle | 65.3 MB/s | 89.6x faster |
| ParallelWrite/1KB/C100 | turtle | 1.0 MB/s | 21.3x faster |
| ParallelWrite/1KB/C200 | turtle | 1.8 MB/s | 80.5x faster |
| ParallelWrite/1KB/C25 | turtle | 1.9 MB/s | 9.5x faster |
| ParallelWrite/1KB/C50 | turtle | 25.1 MB/s | 236.3x faster |

### ParallelRead Operations

**Best for ParallelRead:** turtle (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | turtle | 4.5 GB/s | 2.9x faster |
| ParallelRead/1KB/C10 | turtle | 3.5 GB/s | 3.7x faster |
| ParallelRead/1KB/C100 | turtle | 2.8 GB/s | 4.5x faster |
| ParallelRead/1KB/C200 | turtle | 2.5 GB/s | 7.7x faster |
| ParallelRead/1KB/C25 | turtle | 3.4 GB/s | 3.8x faster |
| ParallelRead/1KB/C50 | turtle | 3.2 GB/s | 4.2x faster |

### Delete Operations

**Best for Delete:** turtle (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | turtle | 2.3M ops/s | 102.3x faster |

### Stat Operations

**Best for Stat:** turtle (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | turtle | 12.6M ops/s | 2.4x faster |

### List Operations

**Best for List:** turtle (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | turtle | 92.5K ops/s | 20.1x faster |

### Copy Operations

**Best for Copy:** turtle (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | turtle | 1.0 GB/s | 577.2x faster |

### Scale Operations

**Best for Scale:** turtle (won 12/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | turtle | 263.8K ops/s | 149.3x faster |
| Scale/Delete/100 | turtle | 39.9K ops/s | 149.8x faster |
| Scale/Delete/1000 | turtle | 1.5K ops/s | 76.8x faster |
| Scale/Delete/10000 | turtle | 163 ops/s | 105.5x faster |
| Scale/List/10 | turtle | 122.4K ops/s | 14.1x faster |
| Scale/List/100 | turtle | 41.0K ops/s | 13.2x faster |
| Scale/List/1000 | turtle | 4.8K ops/s | 15.4x faster |
| Scale/List/10000 | turtle | 277 ops/s | 9.1x faster |
| Scale/Write/10 | turtle | 114.9 MB/s | 3382.4x faster |
| Scale/Write/100 | turtle | 96.1 MB/s | 695.2x faster |
| Scale/Write/1000 | turtle | 309.8 MB/s | 696.1x faster |
| Scale/Write/10000 | turtle | 174.9 MB/s | 310.0x faster |

---

*Generated by storage benchmark CLI*
