# Storage Benchmark Summary

**Generated:** 2026-01-21T15:27:13+07:00

## Overall Winner

**usagi** won 35/51 categories (69%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| usagi | 35 | 69% |
| rabbit | 16 | 31% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **usagi** | 28.8 MB/s | rabbit | 2.1 MB/s | 13.6x faster |
| Delete | **usagi** | 33.4K ops/s | rabbit | 17.9K ops/s | +86% |
| EdgeCase/DeepNested | **usagi** | 4.4 MB/s | rabbit | 0.5 MB/s | 8.7x faster |
| EdgeCase/EmptyObject | **usagi** | 80.2K ops/s | rabbit | 3.5K ops/s | 22.7x faster |
| EdgeCase/LongKey256 | **usagi** | 4.9 MB/s | rabbit | 0.4 MB/s | 11.5x faster |
| FileCount/Delete/1 | **usagi** | 46.2K ops/s | rabbit | 11.9K ops/s | 3.9x faster |
| FileCount/Delete/10 | **usagi** | 10.6K ops/s | rabbit | 1.8K ops/s | 5.9x faster |
| FileCount/Delete/100 | **usagi** | 879 ops/s | rabbit | 148 ops/s | 5.9x faster |
| FileCount/Delete/1000 | **usagi** | 108 ops/s | rabbit | 13 ops/s | 8.2x faster |
| FileCount/Delete/10000 | **usagi** | 7 ops/s | rabbit | 1 ops/s | 6.5x faster |
| FileCount/List/1 | **rabbit** | 14.6K ops/s | usagi | 3.5K ops/s | 4.2x faster |
| FileCount/List/10 | **usagi** | 8.1K ops/s | rabbit | 7.8K ops/s | ~equal |
| FileCount/List/100 | **usagi** | 5.5K ops/s | rabbit | 1.4K ops/s | 3.9x faster |
| FileCount/List/1000 | **usagi** | 1.4K ops/s | rabbit | 183 ops/s | 7.6x faster |
| FileCount/List/10000 | **usagi** | 192 ops/s | rabbit | 15 ops/s | 13.0x faster |
| FileCount/Write/1 | **usagi** | 16.0 MB/s | rabbit | 2.2 MB/s | 7.3x faster |
| FileCount/Write/10 | **usagi** | 42.6 MB/s | rabbit | 6.6 MB/s | 6.4x faster |
| FileCount/Write/100 | **usagi** | 74.6 MB/s | rabbit | 5.6 MB/s | 13.4x faster |
| FileCount/Write/1000 | **usagi** | 60.9 MB/s | rabbit | 8.4 MB/s | 7.2x faster |
| FileCount/Write/10000 | **usagi** | 65.9 MB/s | rabbit | 7.3 MB/s | 9.0x faster |
| List/100 | **usagi** | 15.5K ops/s | rabbit | 3.1K ops/s | 5.1x faster |
| MixedWorkload/Balanced_50_50 | **usagi** | 4.0 MB/s | rabbit | 0.4 MB/s | 10.5x faster |
| MixedWorkload/ReadHeavy_90_10 | **usagi** | 11.7 MB/s | rabbit | 1.3 MB/s | 8.8x faster |
| MixedWorkload/WriteHeavy_10_90 | **usagi** | 2.5 MB/s | rabbit | 0.2 MB/s | 15.1x faster |
| Multipart/15MB_3Parts | **usagi** | 408.5 MB/s | rabbit | 241.2 MB/s | +69% |
| ParallelRead/1KB/C1 | **rabbit** | 812.0 MB/s | usagi | 52.5 MB/s | 15.5x faster |
| ParallelRead/1KB/C10 | **rabbit** | 454.8 MB/s | usagi | 16.8 MB/s | 27.1x faster |
| ParallelRead/1KB/C100 | **rabbit** | 441.0 MB/s | usagi | 12.8 MB/s | 34.4x faster |
| ParallelRead/1KB/C200 | **rabbit** | 433.5 MB/s | usagi | 12.1 MB/s | 35.7x faster |
| ParallelRead/1KB/C25 | **rabbit** | 456.4 MB/s | usagi | 15.0 MB/s | 30.4x faster |
| ParallelRead/1KB/C50 | **rabbit** | 451.5 MB/s | usagi | 13.6 MB/s | 33.1x faster |
| ParallelWrite/1KB/C1 | **usagi** | 66.7 MB/s | rabbit | 8.3 MB/s | 8.0x faster |
| ParallelWrite/1KB/C10 | **usagi** | 3.4 MB/s | rabbit | 1.2 MB/s | 2.9x faster |
| ParallelWrite/1KB/C100 | **usagi** | 0.6 MB/s | rabbit | 0.1 MB/s | 8.4x faster |
| ParallelWrite/1KB/C200 | **usagi** | 0.3 MB/s | rabbit | 0.0 MB/s | 14.5x faster |
| ParallelWrite/1KB/C25 | **usagi** | 1.6 MB/s | rabbit | 0.4 MB/s | 4.1x faster |
| ParallelWrite/1KB/C50 | **usagi** | 0.9 MB/s | rabbit | 0.2 MB/s | 5.0x faster |
| RangeRead/End_256KB | **rabbit** | 5.8 GB/s | usagi | 5.2 GB/s | +12% |
| RangeRead/Middle_256KB | **rabbit** | 6.4 GB/s | usagi | 5.1 GB/s | +26% |
| RangeRead/Start_256KB | **rabbit** | 6.4 GB/s | usagi | 5.4 GB/s | +19% |
| Read/100MB | **rabbit** | 2.6 GB/s | usagi | 2.3 GB/s | +16% |
| Read/10MB | **usagi** | 7.6 GB/s | rabbit | 5.7 GB/s | +34% |
| Read/1KB | **rabbit** | 860.7 MB/s | usagi | 82.2 MB/s | 10.5x faster |
| Read/1MB | **usagi** | 6.8 GB/s | rabbit | 4.6 GB/s | +48% |
| Read/64KB | **rabbit** | 12.7 GB/s | usagi | 3.2 GB/s | 3.9x faster |
| Stat | **usagi** | 1.5M ops/s | rabbit | 1.0M ops/s | +46% |
| Write/100MB | **rabbit** | 1.6 GB/s | usagi | 579.4 MB/s | 2.8x faster |
| Write/10MB | **rabbit** | 1.6 GB/s | usagi | 560.0 MB/s | 2.9x faster |
| Write/1KB | **usagi** | 66.2 MB/s | rabbit | 4.5 MB/s | 14.9x faster |
| Write/1MB | **rabbit** | 1.4 GB/s | usagi | 1.3 GB/s | +12% |
| Write/64KB | **usagi** | 934.3 MB/s | rabbit | 223.8 MB/s | 4.2x faster |

## Category Summaries

### Write Operations

**Best for Write:** rabbit (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | rabbit | 1.6 GB/s | 2.8x faster |
| Write/10MB | rabbit | 1.6 GB/s | 2.9x faster |
| Write/1KB | usagi | 66.2 MB/s | 14.9x faster |
| Write/1MB | rabbit | 1.4 GB/s | +12% |
| Write/64KB | usagi | 934.3 MB/s | 4.2x faster |

### Read Operations

**Best for Read:** rabbit (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | rabbit | 2.6 GB/s | +16% |
| Read/10MB | usagi | 7.6 GB/s | +34% |
| Read/1KB | rabbit | 860.7 MB/s | 10.5x faster |
| Read/1MB | usagi | 6.8 GB/s | +48% |
| Read/64KB | rabbit | 12.7 GB/s | 3.9x faster |

### ParallelWrite Operations

**Best for ParallelWrite:** usagi (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | usagi | 66.7 MB/s | 8.0x faster |
| ParallelWrite/1KB/C10 | usagi | 3.4 MB/s | 2.9x faster |
| ParallelWrite/1KB/C100 | usagi | 0.6 MB/s | 8.4x faster |
| ParallelWrite/1KB/C200 | usagi | 0.3 MB/s | 14.5x faster |
| ParallelWrite/1KB/C25 | usagi | 1.6 MB/s | 4.1x faster |
| ParallelWrite/1KB/C50 | usagi | 0.9 MB/s | 5.0x faster |

### ParallelRead Operations

**Best for ParallelRead:** rabbit (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | rabbit | 812.0 MB/s | 15.5x faster |
| ParallelRead/1KB/C10 | rabbit | 454.8 MB/s | 27.1x faster |
| ParallelRead/1KB/C100 | rabbit | 441.0 MB/s | 34.4x faster |
| ParallelRead/1KB/C200 | rabbit | 433.5 MB/s | 35.7x faster |
| ParallelRead/1KB/C25 | rabbit | 456.4 MB/s | 30.4x faster |
| ParallelRead/1KB/C50 | rabbit | 451.5 MB/s | 33.1x faster |

### Delete Operations

**Best for Delete:** usagi (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | usagi | 33.4K ops/s | +86% |

### Stat Operations

**Best for Stat:** usagi (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | usagi | 1.5M ops/s | +46% |

### List Operations

**Best for List:** usagi (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | usagi | 15.5K ops/s | 5.1x faster |

### Copy Operations

**Best for Copy:** usagi (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | usagi | 28.8 MB/s | 13.6x faster |

### FileCount Operations

**Best for FileCount:** usagi (won 14/15)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| FileCount/Delete/1 | usagi | 46.2K ops/s | 3.9x faster |
| FileCount/Delete/10 | usagi | 10.6K ops/s | 5.9x faster |
| FileCount/Delete/100 | usagi | 879 ops/s | 5.9x faster |
| FileCount/Delete/1000 | usagi | 108 ops/s | 8.2x faster |
| FileCount/Delete/10000 | usagi | 7 ops/s | 6.5x faster |
| FileCount/List/1 | rabbit | 14.6K ops/s | 4.2x faster |
| FileCount/List/10 | usagi | 8.1K ops/s | ~equal |
| FileCount/List/100 | usagi | 5.5K ops/s | 3.9x faster |
| FileCount/List/1000 | usagi | 1.4K ops/s | 7.6x faster |
| FileCount/List/10000 | usagi | 192 ops/s | 13.0x faster |
| FileCount/Write/1 | usagi | 16.0 MB/s | 7.3x faster |
| FileCount/Write/10 | usagi | 42.6 MB/s | 6.4x faster |
| FileCount/Write/100 | usagi | 74.6 MB/s | 13.4x faster |
| FileCount/Write/1000 | usagi | 60.9 MB/s | 7.2x faster |
| FileCount/Write/10000 | usagi | 65.9 MB/s | 9.0x faster |

---

*Generated by storage benchmark CLI*
