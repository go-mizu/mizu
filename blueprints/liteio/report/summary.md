# Storage Benchmark Summary

**Generated:** 2026-02-19T03:49:39+07:00

## Overall Winner

**turtle** won 44/48 categories (92%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| turtle | 44 | 92% |
| local | 4 | 8% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **turtle** | 29.9 MB/s | local | 2.4 MB/s | 12.4x faster |
| Delete | **turtle** | 133.0K ops/s | local | 24.1K ops/s | 5.5x faster |
| EdgeCase/DeepNested | **turtle** | 85.9 MB/s | local | 0.2 MB/s | 390.2x faster |
| EdgeCase/EmptyObject | **turtle** | 1.9M ops/s | local | 2.5K ops/s | 756.7x faster |
| EdgeCase/LongKey256 | **turtle** | 5.1 MB/s | local | 0.2 MB/s | 21.6x faster |
| List/100 | **turtle** | 89.2K ops/s | local | 4.6K ops/s | 19.3x faster |
| MixedWorkload/Balanced_50_50 | **turtle** | 1.4 MB/s | local | 0.5 MB/s | 2.6x faster |
| MixedWorkload/ReadHeavy_90_10 | **turtle** | 7.4 MB/s | local | 2.8 MB/s | 2.6x faster |
| MixedWorkload/WriteHeavy_10_90 | **turtle** | 0.8 MB/s | local | 0.3 MB/s | 2.8x faster |
| Multipart/15MB_3Parts | **turtle** | 400.1 MB/s | local | 242.4 MB/s | +65% |
| ParallelRead/1KB/C1 | **turtle** | 4.6 GB/s | local | 1.5 GB/s | 3.0x faster |
| ParallelRead/1KB/C10 | **turtle** | 3.5 GB/s | local | 986.2 MB/s | 3.6x faster |
| ParallelRead/1KB/C100 | **turtle** | 2.9 GB/s | local | 636.7 MB/s | 4.5x faster |
| ParallelRead/1KB/C200 | **turtle** | 2.6 GB/s | local | 580.1 MB/s | 4.5x faster |
| ParallelRead/1KB/C25 | **turtle** | 3.4 GB/s | local | 902.1 MB/s | 3.8x faster |
| ParallelRead/1KB/C50 | **turtle** | 3.1 GB/s | local | 756.1 MB/s | 4.0x faster |
| ParallelWrite/1KB/C1 | **turtle** | 272.2 MB/s | local | 2.1 MB/s | 127.5x faster |
| ParallelWrite/1KB/C10 | **turtle** | 6.5 MB/s | local | 0.7 MB/s | 8.9x faster |
| ParallelWrite/1KB/C100 | **turtle** | 9.6 MB/s | local | 0.0 MB/s | 212.0x faster |
| ParallelWrite/1KB/C200 | **turtle** | 6.8 MB/s | local | 0.0 MB/s | 296.8x faster |
| ParallelWrite/1KB/C25 | **turtle** | 68.9 MB/s | local | 0.2 MB/s | 324.6x faster |
| ParallelWrite/1KB/C50 | **turtle** | 1.7 MB/s | local | 0.1 MB/s | 18.8x faster |
| RangeRead/End_256KB | **turtle** | 1948.1 GB/s | local | 16.1 GB/s | 120.9x faster |
| RangeRead/Middle_256KB | **turtle** | 1928.9 GB/s | local | 16.3 GB/s | 118.3x faster |
| RangeRead/Start_256KB | **turtle** | 1842.5 GB/s | local | 16.3 GB/s | 113.0x faster |
| Read/100MB | **turtle** | 781684.1 GB/s | local | 16.7 GB/s | 46819.9x faster |
| Read/10MB | **turtle** | 77141.5 GB/s | local | 13.1 GB/s | 5866.8x faster |
| Read/1KB | **local** | 7.4 GB/s | turtle | 6.9 GB/s | ~equal |
| Read/1MB | **turtle** | 7574.5 GB/s | local | 96.1 GB/s | 78.8x faster |
| Read/64KB | **turtle** | 468.1 GB/s | local | 29.6 GB/s | 15.8x faster |
| Scale/Delete/10 | **turtle** | 263.8K ops/s | local | 2.2K ops/s | 118.6x faster |
| Scale/Delete/100 | **turtle** | 36.6K ops/s | local | 266 ops/s | 137.8x faster |
| Scale/Delete/1000 | **turtle** | 3.6K ops/s | local | 19 ops/s | 188.4x faster |
| Scale/Delete/10000 | **turtle** | 271 ops/s | local | 2 ops/s | 145.4x faster |
| Scale/List/10 | **turtle** | 153.8K ops/s | local | 9.9K ops/s | 15.5x faster |
| Scale/List/100 | **turtle** | 40.9K ops/s | local | 3.0K ops/s | 13.5x faster |
| Scale/List/1000 | **turtle** | 3.8K ops/s | local | 314 ops/s | 12.0x faster |
| Scale/List/10000 | **turtle** | 270 ops/s | local | 35 ops/s | 7.7x faster |
| Scale/Write/10 | **turtle** | 113.6 MB/s | local | 0.7 MB/s | 156.5x faster |
| Scale/Write/100 | **turtle** | 198.1 MB/s | local | 0.7 MB/s | 302.6x faster |
| Scale/Write/1000 | **turtle** | 176.2 MB/s | local | 0.6 MB/s | 272.1x faster |
| Scale/Write/10000 | **turtle** | 167.7 MB/s | local | 0.6 MB/s | 282.2x faster |
| Stat | **turtle** | 12.5M ops/s | local | 5.6M ops/s | 2.3x faster |
| Write/100MB | **local** | 2.8 GB/s | turtle | 667.4 MB/s | 4.2x faster |
| Write/10MB | **local** | 2.7 GB/s | turtle | 759.6 MB/s | 3.6x faster |
| Write/1KB | **turtle** | 147.8 MB/s | local | 2.7 MB/s | 55.8x faster |
| Write/1MB | **local** | 1.5 GB/s | turtle | 647.6 MB/s | 2.4x faster |
| Write/64KB | **turtle** | 576.7 MB/s | local | 149.0 MB/s | 3.9x faster |

## Category Summaries

### Write Operations

**Best for Write:** local (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | local | 2.8 GB/s | 4.2x faster |
| Write/10MB | local | 2.7 GB/s | 3.6x faster |
| Write/1KB | turtle | 147.8 MB/s | 55.8x faster |
| Write/1MB | local | 1.5 GB/s | 2.4x faster |
| Write/64KB | turtle | 576.7 MB/s | 3.9x faster |

### Read Operations

**Best for Read:** turtle (won 4/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | turtle | 781684.1 GB/s | 46819.9x faster |
| Read/10MB | turtle | 77141.5 GB/s | 5866.8x faster |
| Read/1KB | local | 7.4 GB/s | ~equal |
| Read/1MB | turtle | 7574.5 GB/s | 78.8x faster |
| Read/64KB | turtle | 468.1 GB/s | 15.8x faster |

### ParallelWrite Operations

**Best for ParallelWrite:** turtle (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | turtle | 272.2 MB/s | 127.5x faster |
| ParallelWrite/1KB/C10 | turtle | 6.5 MB/s | 8.9x faster |
| ParallelWrite/1KB/C100 | turtle | 9.6 MB/s | 212.0x faster |
| ParallelWrite/1KB/C200 | turtle | 6.8 MB/s | 296.8x faster |
| ParallelWrite/1KB/C25 | turtle | 68.9 MB/s | 324.6x faster |
| ParallelWrite/1KB/C50 | turtle | 1.7 MB/s | 18.8x faster |

### ParallelRead Operations

**Best for ParallelRead:** turtle (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | turtle | 4.6 GB/s | 3.0x faster |
| ParallelRead/1KB/C10 | turtle | 3.5 GB/s | 3.6x faster |
| ParallelRead/1KB/C100 | turtle | 2.9 GB/s | 4.5x faster |
| ParallelRead/1KB/C200 | turtle | 2.6 GB/s | 4.5x faster |
| ParallelRead/1KB/C25 | turtle | 3.4 GB/s | 3.8x faster |
| ParallelRead/1KB/C50 | turtle | 3.1 GB/s | 4.0x faster |

### Delete Operations

**Best for Delete:** turtle (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | turtle | 133.0K ops/s | 5.5x faster |

### Stat Operations

**Best for Stat:** turtle (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | turtle | 12.5M ops/s | 2.3x faster |

### List Operations

**Best for List:** turtle (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | turtle | 89.2K ops/s | 19.3x faster |

### Copy Operations

**Best for Copy:** turtle (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | turtle | 29.9 MB/s | 12.4x faster |

### Scale Operations

**Best for Scale:** turtle (won 12/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | turtle | 263.8K ops/s | 118.6x faster |
| Scale/Delete/100 | turtle | 36.6K ops/s | 137.8x faster |
| Scale/Delete/1000 | turtle | 3.6K ops/s | 188.4x faster |
| Scale/Delete/10000 | turtle | 271 ops/s | 145.4x faster |
| Scale/List/10 | turtle | 153.8K ops/s | 15.5x faster |
| Scale/List/100 | turtle | 40.9K ops/s | 13.5x faster |
| Scale/List/1000 | turtle | 3.8K ops/s | 12.0x faster |
| Scale/List/10000 | turtle | 270 ops/s | 7.7x faster |
| Scale/Write/10 | turtle | 113.6 MB/s | 156.5x faster |
| Scale/Write/100 | turtle | 198.1 MB/s | 302.6x faster |
| Scale/Write/1000 | turtle | 176.2 MB/s | 272.1x faster |
| Scale/Write/10000 | turtle | 167.7 MB/s | 282.2x faster |

---

*Generated by storage benchmark CLI*
