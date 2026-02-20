# Storage Benchmark Summary

**Generated:** 2026-02-21T00:40:01+07:00

## Overall Winner

**falcon** won 20/48 categories (42%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| falcon | 20 | 42% |
| jaguar | 11 | 23% |
| owl | 9 | 19% |
| ant | 5 | 10% |
| kangaroo | 3 | 6% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **falcon** | 607.7 MB/s | gecko | 287.5 MB/s | 2.1x faster |
| Delete | **owl** | 3.5M ops/s | gecko | 1.8M ops/s | +99% |
| EdgeCase/DeepNested | **falcon** | 89.4 MB/s | gecko | 38.1 MB/s | 2.3x faster |
| EdgeCase/EmptyObject | **falcon** | 1.5M ops/s | narwhal | 515.3K ops/s | 2.8x faster |
| EdgeCase/LongKey256 | **falcon** | 68.7 MB/s | gecko | 14.9 MB/s | 4.6x faster |
| List/100 | **ant** | 54.9K ops/s | owl | 15.0K ops/s | 3.7x faster |
| MixedWorkload/Balanced_50_50 | **falcon** | 433.4 MB/s | jaguar | 11.9 MB/s | 36.3x faster |
| MixedWorkload/ReadHeavy_90_10 | **falcon** | 460.7 MB/s | jaguar | 82.1 MB/s | 5.6x faster |
| MixedWorkload/WriteHeavy_10_90 | **falcon** | 374.6 MB/s | gecko | 3.8 MB/s | 98.8x faster |
| Multipart/15MB_3Parts | **falcon** | 480.1 MB/s | fox | 466.5 MB/s | ~equal |
| ParallelRead/1KB/C1 | **jaguar** | 2.7 GB/s | falcon | 1.4 GB/s | +94% |
| ParallelRead/1KB/C10 | **jaguar** | 1.8 GB/s | falcon | 363.0 MB/s | 5.0x faster |
| ParallelRead/1KB/C100 | **falcon** | 564.4 MB/s | jaguar | 528.4 MB/s | ~equal |
| ParallelRead/1KB/C200 | **jaguar** | 1.3 GB/s | falcon | 348.0 MB/s | 3.6x faster |
| ParallelRead/1KB/C25 | **jaguar** | 1.9 GB/s | gecko | 398.1 MB/s | 4.8x faster |
| ParallelRead/1KB/C50 | **jaguar** | 1.7 GB/s | falcon | 680.0 MB/s | 2.5x faster |
| ParallelWrite/1KB/C1 | **falcon** | 1.0 GB/s | gecko | 279.8 MB/s | 3.7x faster |
| ParallelWrite/1KB/C10 | **falcon** | 327.0 MB/s | gecko | 32.9 MB/s | 9.9x faster |
| ParallelWrite/1KB/C100 | **falcon** | 110.8 MB/s | gecko | 2.7 MB/s | 40.3x faster |
| ParallelWrite/1KB/C200 | **falcon** | 439.9 MB/s | gecko | 1.3 MB/s | 348.8x faster |
| ParallelWrite/1KB/C25 | **falcon** | 301.2 MB/s | gecko | 14.2 MB/s | 21.2x faster |
| ParallelWrite/1KB/C50 | **falcon** | 385.4 MB/s | gecko | 5.9 MB/s | 65.0x faster |
| RangeRead/End_256KB | **jaguar** | 58.0 GB/s | falcon | 2.9 GB/s | 19.8x faster |
| RangeRead/Middle_256KB | **jaguar** | 57.5 GB/s | falcon | 3.3 GB/s | 17.2x faster |
| RangeRead/Start_256KB | **jaguar** | 50.7 GB/s | falcon | 5.3 GB/s | 9.6x faster |
| Read/100MB | **kangaroo** | 54.2 GB/s | gecko | 4.7 GB/s | 11.5x faster |
| Read/10MB | **kangaroo** | 54.9 GB/s | gecko | 4.8 GB/s | 11.5x faster |
| Read/1KB | **jaguar** | 4.3 GB/s | kangaroo | 3.6 GB/s | +21% |
| Read/1MB | **kangaroo** | 58.5 GB/s | falcon | 5.4 GB/s | 10.9x faster |
| Read/64KB | **jaguar** | 49.0 GB/s | kangaroo | 47.1 GB/s | ~equal |
| Scale/Delete/10 | **owl** | 96.0K ops/s | gecko | 51.7K ops/s | +86% |
| Scale/Delete/100 | **owl** | 16.4K ops/s | gecko | 9.1K ops/s | +81% |
| Scale/Delete/1000 | **owl** | 2.2K ops/s | gecko | 1.5K ops/s | +49% |
| Scale/Delete/10000 | **owl** | 208 ops/s | gecko | 170 ops/s | +22% |
| Scale/List/10 | **ant** | 101.3K ops/s | narwhal | 53 ops/s | 1905.7x faster |
| Scale/List/100 | **ant** | 49.8K ops/s | narwhal | 54 ops/s | 929.8x faster |
| Scale/List/1000 | **ant** | 9.0K ops/s | narwhal | 52 ops/s | 171.1x faster |
| Scale/List/10000 | **ant** | 528 ops/s | narwhal | 48 ops/s | 10.9x faster |
| Scale/Write/10 | **jaguar** | 86.9 MB/s | gecko | 64.1 MB/s | +36% |
| Scale/Write/100 | **owl** | 323.5 MB/s | jaguar | 109.5 MB/s | 3.0x faster |
| Scale/Write/1000 | **owl** | 395.5 MB/s | jaguar | 160.5 MB/s | 2.5x faster |
| Scale/Write/10000 | **owl** | 412.6 MB/s | gecko | 149.7 MB/s | 2.8x faster |
| Stat | **falcon** | 10.8M ops/s | kangaroo | 8.3M ops/s | +30% |
| Write/100MB | **falcon** | 8.9 GB/s | kangaroo | 2.4 GB/s | 3.7x faster |
| Write/10MB | **falcon** | 8.2 GB/s | kangaroo | 3.3 GB/s | 2.5x faster |
| Write/1KB | **owl** | 2.5 GB/s | falcon | 1.7 GB/s | +45% |
| Write/1MB | **falcon** | 8.4 GB/s | owl | 2.7 GB/s | 3.1x faster |
| Write/64KB | **falcon** | 10.0 GB/s | fox | 1.0 GB/s | 9.6x faster |

## Category Summaries

### Write Operations

**Best for Write:** falcon (won 4/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | falcon | 8.9 GB/s | 3.7x faster |
| Write/10MB | falcon | 8.2 GB/s | 2.5x faster |
| Write/1KB | owl | 2.5 GB/s | +45% |
| Write/1MB | falcon | 8.4 GB/s | 3.1x faster |
| Write/64KB | falcon | 10.0 GB/s | 9.6x faster |

### Read Operations

**Best for Read:** kangaroo (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | kangaroo | 54.2 GB/s | 11.5x faster |
| Read/10MB | kangaroo | 54.9 GB/s | 11.5x faster |
| Read/1KB | jaguar | 4.3 GB/s | +21% |
| Read/1MB | kangaroo | 58.5 GB/s | 10.9x faster |
| Read/64KB | jaguar | 49.0 GB/s | ~equal |

### ParallelWrite Operations

**Best for ParallelWrite:** falcon (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | falcon | 1.0 GB/s | 3.7x faster |
| ParallelWrite/1KB/C10 | falcon | 327.0 MB/s | 9.9x faster |
| ParallelWrite/1KB/C100 | falcon | 110.8 MB/s | 40.3x faster |
| ParallelWrite/1KB/C200 | falcon | 439.9 MB/s | 348.8x faster |
| ParallelWrite/1KB/C25 | falcon | 301.2 MB/s | 21.2x faster |
| ParallelWrite/1KB/C50 | falcon | 385.4 MB/s | 65.0x faster |

### ParallelRead Operations

**Best for ParallelRead:** jaguar (won 5/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | jaguar | 2.7 GB/s | +94% |
| ParallelRead/1KB/C10 | jaguar | 1.8 GB/s | 5.0x faster |
| ParallelRead/1KB/C100 | falcon | 564.4 MB/s | ~equal |
| ParallelRead/1KB/C200 | jaguar | 1.3 GB/s | 3.6x faster |
| ParallelRead/1KB/C25 | jaguar | 1.9 GB/s | 4.8x faster |
| ParallelRead/1KB/C50 | jaguar | 1.7 GB/s | 2.5x faster |

### Delete Operations

**Best for Delete:** owl (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | owl | 3.5M ops/s | +99% |

### Stat Operations

**Best for Stat:** falcon (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | falcon | 10.8M ops/s | +30% |

### List Operations

**Best for List:** ant (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | ant | 54.9K ops/s | 3.7x faster |

### Copy Operations

**Best for Copy:** falcon (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | falcon | 607.7 MB/s | 2.1x faster |

### Scale Operations

**Best for Scale:** owl (won 7/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | owl | 96.0K ops/s | +86% |
| Scale/Delete/100 | owl | 16.4K ops/s | +81% |
| Scale/Delete/1000 | owl | 2.2K ops/s | +49% |
| Scale/Delete/10000 | owl | 208 ops/s | +22% |
| Scale/List/10 | ant | 101.3K ops/s | 1905.7x faster |
| Scale/List/100 | ant | 49.8K ops/s | 929.8x faster |
| Scale/List/1000 | ant | 9.0K ops/s | 171.1x faster |
| Scale/List/10000 | ant | 528 ops/s | 10.9x faster |
| Scale/Write/10 | jaguar | 86.9 MB/s | +36% |
| Scale/Write/100 | owl | 323.5 MB/s | 3.0x faster |
| Scale/Write/1000 | owl | 395.5 MB/s | 2.5x faster |
| Scale/Write/10000 | owl | 412.6 MB/s | 2.8x faster |

## Runtime Resource Usage

| Driver | Peak RSS | Go Heap | Go Sys | Disk Usage | GC Cycles |
|--------|----------|---------|--------|------------|----------|
| ant | 11102.3 MB | 31437.3 MB | 92575.2 MB | 6411.8 MB | 94 |
| falcon | 11102.3 MB | 37357.2 MB | 47066.2 MB | 26683.8 MB | 34 |
| fox | 11102.3 MB | 25011.8 MB | 60907.8 MB | 673.6 MB | 64 |
| gecko | 11102.3 MB | 45290.2 MB | 62126.4 MB | 7329.1 MB | 80 |
| jaguar | 11102.3 MB | 37914.7 MB | 60911.1 MB | 9249.0 MB | 72 |
| kangaroo | 11102.3 MB | 35664.8 MB | 92575.2 MB | 576.0 MB | 99 |
| narwhal | 11102.3 MB | 43971.7 MB | 92575.2 MB | 4986.5 MB | 104 |
| owl | 11102.3 MB | 70898.4 MB | 92575.2 MB | 6958.6 MB | 87 |
| spider | 11102.3 MB | 53317.3 MB | 60819.1 MB | 1947.1 MB | 52 |

---

*Generated by storage benchmark CLI*
