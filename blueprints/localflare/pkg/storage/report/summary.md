# Storage Benchmark Summary

**Generated:** 2026-01-22T18:26:51+07:00

## Overall Winner

**usagi** won 43/48 categories (90%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| usagi | 43 | 90% |
| usagi_s3 | 4 | 8% |
| rustfs | 1 | 2% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **usagi** | 94.4 MB/s | usagi_s3 | 1.0 MB/s | 92.5x faster |
| Delete | **usagi** | 210.6K ops/s | usagi_s3 | 4.6K ops/s | 45.6x faster |
| EdgeCase/DeepNested | **usagi** | 1.8 MB/s | rustfs | 0.2 MB/s | 12.1x faster |
| EdgeCase/EmptyObject | **usagi** | 277.0K ops/s | rustfs | 1.6K ops/s | 178.7x faster |
| EdgeCase/LongKey256 | **usagi** | 15.5 MB/s | rustfs | 0.1 MB/s | 112.6x faster |
| List/100 | **usagi** | 61.7K ops/s | usagi_s3 | 1.0K ops/s | 61.3x faster |
| MixedWorkload/Balanced_50_50 | **usagi** | 3.1 MB/s | usagi_s3 | 0.5 MB/s | 5.6x faster |
| MixedWorkload/ReadHeavy_90_10 | **usagi** | 8.7 MB/s | usagi_s3 | 0.7 MB/s | 12.5x faster |
| MixedWorkload/WriteHeavy_10_90 | **usagi** | 1.8 MB/s | usagi_s3 | 0.3 MB/s | 5.9x faster |
| Multipart/15MB_3Parts | **usagi** | 190.2 MB/s | usagi_s3 | 110.0 MB/s | +73% |
| ParallelRead/1KB/C1 | **usagi** | 713.9 MB/s | usagi_s3 | 4.0 MB/s | 177.4x faster |
| ParallelRead/1KB/C10 | **usagi** | 81.8 MB/s | usagi_s3 | 1.3 MB/s | 63.2x faster |
| ParallelRead/1KB/C100 | **usagi** | 7.0 MB/s | usagi_s3 | 0.2 MB/s | 42.8x faster |
| ParallelRead/1KB/C200 | **usagi** | 4.1 MB/s | usagi_s3 | 0.1 MB/s | 45.6x faster |
| ParallelRead/1KB/C25 | **usagi** | 28.4 MB/s | usagi_s3 | 0.6 MB/s | 50.3x faster |
| ParallelRead/1KB/C50 | **usagi** | 14.5 MB/s | usagi_s3 | 0.3 MB/s | 42.8x faster |
| ParallelWrite/1KB/C1 | **usagi** | 93.1 MB/s | rustfs | 1.4 MB/s | 67.0x faster |
| ParallelWrite/1KB/C10 | **usagi** | 21.5 MB/s | rustfs | 0.5 MB/s | 47.5x faster |
| ParallelWrite/1KB/C100 | **usagi** | 2.3 MB/s | usagi_s3 | 0.1 MB/s | 41.2x faster |
| ParallelWrite/1KB/C200 | **usagi** | 0.8 MB/s | usagi_s3 | 0.0 MB/s | 32.6x faster |
| ParallelWrite/1KB/C25 | **rustfs** | 0.2 MB/s | usagi_s3 | 0.2 MB/s | ~equal |
| ParallelWrite/1KB/C50 | **usagi** | 4.0 MB/s | rustfs | 0.1 MB/s | 43.0x faster |
| RangeRead/End_256KB | **usagi** | 10.6 GB/s | usagi_s3 | 151.8 MB/s | 69.8x faster |
| RangeRead/Middle_256KB | **usagi** | 10.7 GB/s | usagi_s3 | 144.7 MB/s | 73.9x faster |
| RangeRead/Start_256KB | **usagi** | 8.8 GB/s | usagi_s3 | 137.0 MB/s | 64.5x faster |
| Read/100MB | **usagi** | 6.5 GB/s | rustfs | 216.8 MB/s | 30.1x faster |
| Read/10MB | **usagi** | 8.9 GB/s | rustfs | 220.6 MB/s | 40.2x faster |
| Read/1KB | **usagi** | 3.4 GB/s | usagi_s3 | 3.6 MB/s | 941.2x faster |
| Read/1MB | **usagi** | 8.3 GB/s | usagi_s3 | 174.3 MB/s | 47.9x faster |
| Read/64KB | **usagi** | 16.5 GB/s | usagi_s3 | 118.3 MB/s | 139.5x faster |
| Scale/Delete/10 | **usagi** | 4.0K ops/s | usagi_s3 | 125 ops/s | 31.8x faster |
| Scale/Delete/100 | **usagi** | 1.4K ops/s | usagi_s3 | 45 ops/s | 30.2x faster |
| Scale/Delete/1000 | **usagi** | 277 ops/s | usagi_s3 | 4 ops/s | 63.3x faster |
| Scale/Delete/10000 | **usagi** | 19 ops/s | usagi_s3 | 0 ops/s | 51.8x faster |
| Scale/List/10 | **usagi_s3** | 744 ops/s | rustfs | 347 ops/s | 2.1x faster |
| Scale/List/100 | **usagi_s3** | 842 ops/s | rustfs | 145 ops/s | 5.8x faster |
| Scale/List/1000 | **usagi_s3** | 130 ops/s | rustfs | 20 ops/s | 6.5x faster |
| Scale/List/10000 | **usagi_s3** | 5 ops/s | rustfs | 2 ops/s | 2.9x faster |
| Scale/Write/10 | **usagi** | 15.1 MB/s | rustfs | 0.4 MB/s | 33.6x faster |
| Scale/Write/100 | **usagi** | 24.3 MB/s | rustfs | 0.4 MB/s | 62.0x faster |
| Scale/Write/1000 | **usagi** | 21.7 MB/s | rustfs | 0.4 MB/s | 61.4x faster |
| Scale/Write/10000 | **usagi** | 31.5 MB/s | rustfs | 0.4 MB/s | 86.7x faster |
| Stat | **usagi** | 4.9M ops/s | usagi_s3 | 3.9K ops/s | 1264.8x faster |
| Write/100MB | **usagi** | 483.5 MB/s | rustfs | 164.4 MB/s | 2.9x faster |
| Write/10MB | **usagi** | 398.5 MB/s | rustfs | 169.8 MB/s | 2.3x faster |
| Write/1KB | **usagi** | 231.7 MB/s | rustfs | 1.6 MB/s | 146.6x faster |
| Write/1MB | **usagi** | 692.5 MB/s | rustfs | 137.6 MB/s | 5.0x faster |
| Write/64KB | **usagi** | 708.6 MB/s | rustfs | 55.4 MB/s | 12.8x faster |

## Category Summaries

### Write Operations

**Best for Write:** usagi (won 5/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | usagi | 483.5 MB/s | 2.9x faster |
| Write/10MB | usagi | 398.5 MB/s | 2.3x faster |
| Write/1KB | usagi | 231.7 MB/s | 146.6x faster |
| Write/1MB | usagi | 692.5 MB/s | 5.0x faster |
| Write/64KB | usagi | 708.6 MB/s | 12.8x faster |

### Read Operations

**Best for Read:** usagi (won 5/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | usagi | 6.5 GB/s | 30.1x faster |
| Read/10MB | usagi | 8.9 GB/s | 40.2x faster |
| Read/1KB | usagi | 3.4 GB/s | 941.2x faster |
| Read/1MB | usagi | 8.3 GB/s | 47.9x faster |
| Read/64KB | usagi | 16.5 GB/s | 139.5x faster |

### ParallelWrite Operations

**Best for ParallelWrite:** usagi (won 5/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | usagi | 93.1 MB/s | 67.0x faster |
| ParallelWrite/1KB/C10 | usagi | 21.5 MB/s | 47.5x faster |
| ParallelWrite/1KB/C100 | usagi | 2.3 MB/s | 41.2x faster |
| ParallelWrite/1KB/C200 | usagi | 0.8 MB/s | 32.6x faster |
| ParallelWrite/1KB/C25 | rustfs | 0.2 MB/s | ~equal |
| ParallelWrite/1KB/C50 | usagi | 4.0 MB/s | 43.0x faster |

### ParallelRead Operations

**Best for ParallelRead:** usagi (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | usagi | 713.9 MB/s | 177.4x faster |
| ParallelRead/1KB/C10 | usagi | 81.8 MB/s | 63.2x faster |
| ParallelRead/1KB/C100 | usagi | 7.0 MB/s | 42.8x faster |
| ParallelRead/1KB/C200 | usagi | 4.1 MB/s | 45.6x faster |
| ParallelRead/1KB/C25 | usagi | 28.4 MB/s | 50.3x faster |
| ParallelRead/1KB/C50 | usagi | 14.5 MB/s | 42.8x faster |

### Delete Operations

**Best for Delete:** usagi (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | usagi | 210.6K ops/s | 45.6x faster |

### Stat Operations

**Best for Stat:** usagi (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | usagi | 4.9M ops/s | 1264.8x faster |

### List Operations

**Best for List:** usagi (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | usagi | 61.7K ops/s | 61.3x faster |

### Copy Operations

**Best for Copy:** usagi (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | usagi | 94.4 MB/s | 92.5x faster |

### Scale Operations

**Best for Scale:** usagi (won 8/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | usagi | 4.0K ops/s | 31.8x faster |
| Scale/Delete/100 | usagi | 1.4K ops/s | 30.2x faster |
| Scale/Delete/1000 | usagi | 277 ops/s | 63.3x faster |
| Scale/Delete/10000 | usagi | 19 ops/s | 51.8x faster |
| Scale/List/10 | usagi_s3 | 744 ops/s | 2.1x faster |
| Scale/List/100 | usagi_s3 | 842 ops/s | 5.8x faster |
| Scale/List/1000 | usagi_s3 | 130 ops/s | 6.5x faster |
| Scale/List/10000 | usagi_s3 | 5 ops/s | 2.9x faster |
| Scale/Write/10 | usagi | 15.1 MB/s | 33.6x faster |
| Scale/Write/100 | usagi | 24.3 MB/s | 62.0x faster |
| Scale/Write/1000 | usagi | 21.7 MB/s | 61.4x faster |
| Scale/Write/10000 | usagi | 31.5 MB/s | 86.7x faster |

---

*Generated by storage benchmark CLI*
