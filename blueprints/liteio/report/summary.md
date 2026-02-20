# Storage Benchmark Summary

**Generated:** 2026-02-20T23:58:59+07:00

## Overall Winner

**horse** won 42/48 categories (88%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| horse | 42 | 88% |
| usagi | 4 | 8% |
| devnull_s3 | 1 | 2% |
| rabbit | 1 | 2% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **horse** | 719.9 MB/s | usagi | 306.9 MB/s | 2.3x faster |
| Delete | **horse** | 2.5M ops/s | usagi | 592.6K ops/s | 4.2x faster |
| EdgeCase/DeepNested | **horse** | 88.1 MB/s | usagi | 4.6 MB/s | 19.0x faster |
| EdgeCase/EmptyObject | **horse** | 1.5M ops/s | usagi | 556.1K ops/s | 2.7x faster |
| EdgeCase/LongKey256 | **horse** | 64.6 MB/s | usagi | 29.9 MB/s | 2.2x faster |
| List/100 | **usagi** | 174.0K ops/s | horse | 120.6K ops/s | +44% |
| MixedWorkload/Balanced_50_50 | **horse** | 121.6 MB/s | usagi | 12.4 MB/s | 9.8x faster |
| MixedWorkload/ReadHeavy_90_10 | **horse** | 361.0 MB/s | rabbit | 31.9 MB/s | 11.3x faster |
| MixedWorkload/WriteHeavy_10_90 | **horse** | 10.9 MB/s | usagi | 8.2 MB/s | +33% |
| Multipart/15MB_3Parts | **devnull_s3** | 1.3 GB/s | usagi | 792.0 MB/s | +69% |
| ParallelRead/1KB/C1 | **horse** | 4.5 GB/s | usagi | 1.9 GB/s | 2.3x faster |
| ParallelRead/1KB/C10 | **horse** | 3.2 GB/s | rabbit | 1.4 GB/s | 2.3x faster |
| ParallelRead/1KB/C100 | **horse** | 2.9 GB/s | rabbit | 905.6 MB/s | 3.2x faster |
| ParallelRead/1KB/C200 | **horse** | 2.5 GB/s | rabbit | 834.5 MB/s | 3.0x faster |
| ParallelRead/1KB/C25 | **horse** | 3.4 GB/s | rabbit | 1.2 GB/s | 2.7x faster |
| ParallelRead/1KB/C50 | **horse** | 3.1 GB/s | rabbit | 1.1 GB/s | 2.8x faster |
| ParallelWrite/1KB/C1 | **horse** | 932.8 MB/s | usagi | 247.5 MB/s | 3.8x faster |
| ParallelWrite/1KB/C10 | **horse** | 125.8 MB/s | usagi | 53.4 MB/s | 2.4x faster |
| ParallelWrite/1KB/C100 | **horse** | 12.5 MB/s | usagi | 6.1 MB/s | 2.0x faster |
| ParallelWrite/1KB/C200 | **horse** | 6.6 MB/s | usagi | 3.5 MB/s | +90% |
| ParallelWrite/1KB/C25 | **horse** | 83.8 MB/s | usagi | 18.6 MB/s | 4.5x faster |
| ParallelWrite/1KB/C50 | **horse** | 31.5 MB/s | usagi | 12.3 MB/s | 2.6x faster |
| RangeRead/End_256KB | **horse** | 54.0 GB/s | usagi | 16.0 GB/s | 3.4x faster |
| RangeRead/Middle_256KB | **horse** | 58.6 GB/s | usagi | 15.9 GB/s | 3.7x faster |
| RangeRead/Start_256KB | **horse** | 54.0 GB/s | usagi | 15.7 GB/s | 3.4x faster |
| Read/100MB | **usagi** | 11.4 GB/s | local | 11.0 GB/s | ~equal |
| Read/10MB | **horse** | 15.4 GB/s | usagi | 12.1 GB/s | +27% |
| Read/1KB | **horse** | 6.2 GB/s | local | 5.3 GB/s | +16% |
| Read/1MB | **horse** | 54.2 GB/s | local | 17.5 GB/s | 3.1x faster |
| Read/64KB | **rabbit** | 50.7 GB/s | horse | 50.0 GB/s | ~equal |
| Scale/Delete/10 | **horse** | 180.5K ops/s | usagi | 14.6K ops/s | 12.3x faster |
| Scale/Delete/100 | **horse** | 35.7K ops/s | usagi | 4.0K ops/s | 8.9x faster |
| Scale/Delete/1000 | **horse** | 3.6K ops/s | usagi | 448 ops/s | 8.0x faster |
| Scale/Delete/10000 | **horse** | 274 ops/s | usagi | 58 ops/s | 4.7x faster |
| Scale/List/10 | **horse** | 170.2K ops/s | rabbit | 13.1K ops/s | 13.0x faster |
| Scale/List/100 | **horse** | 65.9K ops/s | rabbit | 3.6K ops/s | 18.2x faster |
| Scale/List/1000 | **horse** | 4.8K ops/s | rabbit | 377 ops/s | 12.8x faster |
| Scale/List/10000 | **horse** | 393 ops/s | rabbit | 35 ops/s | 11.2x faster |
| Scale/Write/10 | **horse** | 95.3 MB/s | usagi | 35.2 MB/s | 2.7x faster |
| Scale/Write/100 | **horse** | 309.5 MB/s | usagi | 55.1 MB/s | 5.6x faster |
| Scale/Write/1000 | **horse** | 267.0 MB/s | usagi | 71.8 MB/s | 3.7x faster |
| Scale/Write/10000 | **horse** | 307.4 MB/s | usagi | 57.9 MB/s | 5.3x faster |
| Stat | **usagi** | 19.0M ops/s | horse | 15.1M ops/s | +26% |
| Write/100MB | **usagi** | 1.9 GB/s | horse | 1.7 GB/s | ~equal |
| Write/10MB | **horse** | 2.4 GB/s | local | 1.7 GB/s | +43% |
| Write/1KB | **horse** | 1.4 GB/s | usagi | 295.3 MB/s | 4.8x faster |
| Write/1MB | **horse** | 2.1 GB/s | rabbit | 1.8 GB/s | +17% |
| Write/64KB | **horse** | 1.6 GB/s | usagi | 1.4 GB/s | +13% |

## Category Summaries

### Write Operations

**Best for Write:** horse (won 4/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | usagi | 1.9 GB/s | ~equal |
| Write/10MB | horse | 2.4 GB/s | +43% |
| Write/1KB | horse | 1.4 GB/s | 4.8x faster |
| Write/1MB | horse | 2.1 GB/s | +17% |
| Write/64KB | horse | 1.6 GB/s | +13% |

### Read Operations

**Best for Read:** horse (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | usagi | 11.4 GB/s | ~equal |
| Read/10MB | horse | 15.4 GB/s | +27% |
| Read/1KB | horse | 6.2 GB/s | +16% |
| Read/1MB | horse | 54.2 GB/s | 3.1x faster |
| Read/64KB | rabbit | 50.7 GB/s | ~equal |

### ParallelWrite Operations

**Best for ParallelWrite:** horse (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | horse | 932.8 MB/s | 3.8x faster |
| ParallelWrite/1KB/C10 | horse | 125.8 MB/s | 2.4x faster |
| ParallelWrite/1KB/C100 | horse | 12.5 MB/s | 2.0x faster |
| ParallelWrite/1KB/C200 | horse | 6.6 MB/s | +90% |
| ParallelWrite/1KB/C25 | horse | 83.8 MB/s | 4.5x faster |
| ParallelWrite/1KB/C50 | horse | 31.5 MB/s | 2.6x faster |

### ParallelRead Operations

**Best for ParallelRead:** horse (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | horse | 4.5 GB/s | 2.3x faster |
| ParallelRead/1KB/C10 | horse | 3.2 GB/s | 2.3x faster |
| ParallelRead/1KB/C100 | horse | 2.9 GB/s | 3.2x faster |
| ParallelRead/1KB/C200 | horse | 2.5 GB/s | 3.0x faster |
| ParallelRead/1KB/C25 | horse | 3.4 GB/s | 2.7x faster |
| ParallelRead/1KB/C50 | horse | 3.1 GB/s | 2.8x faster |

### Delete Operations

**Best for Delete:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | horse | 2.5M ops/s | 4.2x faster |

### Stat Operations

**Best for Stat:** usagi (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | usagi | 19.0M ops/s | +26% |

### List Operations

**Best for List:** usagi (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | usagi | 174.0K ops/s | +44% |

### Copy Operations

**Best for Copy:** horse (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | horse | 719.9 MB/s | 2.3x faster |

### Scale Operations

**Best for Scale:** horse (won 12/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | horse | 180.5K ops/s | 12.3x faster |
| Scale/Delete/100 | horse | 35.7K ops/s | 8.9x faster |
| Scale/Delete/1000 | horse | 3.6K ops/s | 8.0x faster |
| Scale/Delete/10000 | horse | 274 ops/s | 4.7x faster |
| Scale/List/10 | horse | 170.2K ops/s | 13.0x faster |
| Scale/List/100 | horse | 65.9K ops/s | 18.2x faster |
| Scale/List/1000 | horse | 4.8K ops/s | 12.8x faster |
| Scale/List/10000 | horse | 393 ops/s | 11.2x faster |
| Scale/Write/10 | horse | 95.3 MB/s | 2.7x faster |
| Scale/Write/100 | horse | 309.5 MB/s | 5.6x faster |
| Scale/Write/1000 | horse | 267.0 MB/s | 3.7x faster |
| Scale/Write/10000 | horse | 307.4 MB/s | 5.3x faster |

## Runtime Resource Usage

| Driver | Peak RSS | Go Heap | Go Sys | Disk Usage | GC Cycles |
|--------|----------|---------|--------|------------|----------|
| horse | 6695.9 MB | 3120.2 MB | 6860.5 MB | 32768.0 MB | 1502 |
| local | 6695.9 MB | 1302.2 MB | 6836.8 MB | 6076.7 MB | 1432 |
| rabbit | 2039.4 MB | 1849.6 MB | 3806.6 MB | 5531.0 MB | 997 |
| usagi | 6695.1 MB | 5382.5 MB | 6836.8 MB | 12243.8 MB | 1295 |

---

*Generated by storage benchmark CLI*
