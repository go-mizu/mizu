# Storage Benchmark Summary

**Generated:** 2026-01-21T16:18:28+07:00

## Overall Winner

**rabbit_s3** won 21/39 categories (54%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| rabbit_s3 | 21 | 54% |
| liteio | 11 | 28% |
| usagi_s3 | 7 | 18% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **liteio** | 1.2 MB/s | usagi_s3 | 1.2 MB/s | ~equal |
| Delete | **rabbit_s3** | 4.1K ops/s | usagi_s3 | 3.9K ops/s | ~equal |
| EdgeCase/DeepNested | **usagi_s3** | 0.2 MB/s | rabbit_s3 | 0.1 MB/s | +14% |
| EdgeCase/EmptyObject | **liteio** | 1.6K ops/s | rabbit_s3 | 1.5K ops/s | ~equal |
| EdgeCase/LongKey256 | **rabbit_s3** | 0.1 MB/s | liteio | 0.1 MB/s | ~equal |
| List/100 | **rabbit_s3** | 900 ops/s | liteio | 893 ops/s | ~equal |
| MixedWorkload/Balanced_50_50 | **liteio** | 0.4 MB/s | usagi_s3 | 0.4 MB/s | +11% |
| MixedWorkload/ReadHeavy_90_10 | **liteio** | 0.6 MB/s | rabbit_s3 | 0.6 MB/s | ~equal |
| MixedWorkload/WriteHeavy_10_90 | **liteio** | 0.4 MB/s | usagi_s3 | 0.3 MB/s | +11% |
| Multipart/15MB_3Parts | **liteio** | 112.0 MB/s | rabbit_s3 | 95.9 MB/s | +17% |
| ParallelRead/1KB/C1 | **rabbit_s3** | 3.5 MB/s | usagi_s3 | 3.3 MB/s | ~equal |
| ParallelRead/1KB/C10 | **usagi_s3** | 1.0 MB/s | rabbit_s3 | 1.0 MB/s | ~equal |
| ParallelRead/1KB/C100 | **rabbit_s3** | 0.2 MB/s | liteio | 0.2 MB/s | ~equal |
| ParallelRead/1KB/C200 | **usagi_s3** | 0.1 MB/s | liteio | 0.1 MB/s | ~equal |
| ParallelRead/1KB/C25 | **rabbit_s3** | 0.6 MB/s | usagi_s3 | 0.6 MB/s | ~equal |
| ParallelRead/1KB/C50 | **rabbit_s3** | 0.4 MB/s | usagi_s3 | 0.3 MB/s | +10% |
| ParallelWrite/1KB/C1 | **rabbit_s3** | 1.4 MB/s | liteio | 1.4 MB/s | ~equal |
| ParallelWrite/1KB/C10 | **usagi_s3** | 0.4 MB/s | rabbit_s3 | 0.4 MB/s | ~equal |
| ParallelWrite/1KB/C100 | **liteio** | 0.0 MB/s | usagi_s3 | 0.0 MB/s | ~equal |
| ParallelWrite/1KB/C200 | **rabbit_s3** | 0.0 MB/s | liteio | 0.0 MB/s | +20% |
| ParallelWrite/1KB/C25 | **rabbit_s3** | 0.2 MB/s | usagi_s3 | 0.2 MB/s | +13% |
| ParallelWrite/1KB/C50 | **rabbit_s3** | 0.1 MB/s | usagi_s3 | 0.1 MB/s | +53% |
| RangeRead/End_256KB | **usagi_s3** | 150.3 MB/s | rabbit_s3 | 148.1 MB/s | ~equal |
| RangeRead/Middle_256KB | **rabbit_s3** | 149.9 MB/s | liteio | 148.1 MB/s | ~equal |
| RangeRead/Start_256KB | **liteio** | 131.2 MB/s | rabbit_s3 | 131.1 MB/s | ~equal |
| Read/100MB | **rabbit_s3** | 174.5 MB/s | liteio | 170.8 MB/s | ~equal |
| Read/10MB | **rabbit_s3** | 177.4 MB/s | liteio | 173.4 MB/s | ~equal |
| Read/1KB | **liteio** | 4.2 MB/s | rabbit_s3 | 4.1 MB/s | ~equal |
| Read/1MB | **rabbit_s3** | 179.6 MB/s | liteio | 174.1 MB/s | ~equal |
| Read/64KB | **rabbit_s3** | 105.9 MB/s | liteio | 103.1 MB/s | ~equal |
| Scale/Delete/10 | **rabbit_s3** | 401 ops/s | liteio | 392 ops/s | ~equal |
| Scale/List/10 | **liteio** | 1.5K ops/s | rabbit_s3 | 1.4K ops/s | ~equal |
| Scale/Write/10 | **usagi_s3** | 1.3 MB/s | rabbit_s3 | 1.3 MB/s | ~equal |
| Stat | **rabbit_s3** | 4.3K ops/s | liteio | 4.0K ops/s | ~equal |
| Write/100MB | **liteio** | 132.0 MB/s | usagi_s3 | 122.7 MB/s | ~equal |
| Write/10MB | **rabbit_s3** | 122.4 MB/s | usagi_s3 | 115.5 MB/s | ~equal |
| Write/1KB | **usagi_s3** | 1.7 MB/s | rabbit_s3 | 1.7 MB/s | ~equal |
| Write/1MB | **rabbit_s3** | 129.3 MB/s | usagi_s3 | 113.2 MB/s | +14% |
| Write/64KB | **rabbit_s3** | 68.1 MB/s | usagi_s3 | 65.8 MB/s | ~equal |

## Category Summaries

### Write Operations

**Best for Write:** rabbit_s3 (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | liteio | 132.0 MB/s | ~equal |
| Write/10MB | rabbit_s3 | 122.4 MB/s | ~equal |
| Write/1KB | usagi_s3 | 1.7 MB/s | ~equal |
| Write/1MB | rabbit_s3 | 129.3 MB/s | +14% |
| Write/64KB | rabbit_s3 | 68.1 MB/s | ~equal |

### Read Operations

**Best for Read:** rabbit_s3 (won 4/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | rabbit_s3 | 174.5 MB/s | ~equal |
| Read/10MB | rabbit_s3 | 177.4 MB/s | ~equal |
| Read/1KB | liteio | 4.2 MB/s | ~equal |
| Read/1MB | rabbit_s3 | 179.6 MB/s | ~equal |
| Read/64KB | rabbit_s3 | 105.9 MB/s | ~equal |

### ParallelWrite Operations

**Best for ParallelWrite:** rabbit_s3 (won 4/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | rabbit_s3 | 1.4 MB/s | ~equal |
| ParallelWrite/1KB/C10 | usagi_s3 | 0.4 MB/s | ~equal |
| ParallelWrite/1KB/C100 | liteio | 0.0 MB/s | ~equal |
| ParallelWrite/1KB/C200 | rabbit_s3 | 0.0 MB/s | +20% |
| ParallelWrite/1KB/C25 | rabbit_s3 | 0.2 MB/s | +13% |
| ParallelWrite/1KB/C50 | rabbit_s3 | 0.1 MB/s | +53% |

### ParallelRead Operations

**Best for ParallelRead:** rabbit_s3 (won 4/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | rabbit_s3 | 3.5 MB/s | ~equal |
| ParallelRead/1KB/C10 | usagi_s3 | 1.0 MB/s | ~equal |
| ParallelRead/1KB/C100 | rabbit_s3 | 0.2 MB/s | ~equal |
| ParallelRead/1KB/C200 | usagi_s3 | 0.1 MB/s | ~equal |
| ParallelRead/1KB/C25 | rabbit_s3 | 0.6 MB/s | ~equal |
| ParallelRead/1KB/C50 | rabbit_s3 | 0.4 MB/s | +10% |

### Delete Operations

**Best for Delete:** rabbit_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | rabbit_s3 | 4.1K ops/s | ~equal |

### Stat Operations

**Best for Stat:** rabbit_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | rabbit_s3 | 4.3K ops/s | ~equal |

### List Operations

**Best for List:** rabbit_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | rabbit_s3 | 900 ops/s | ~equal |

### Copy Operations

**Best for Copy:** liteio (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | liteio | 1.2 MB/s | ~equal |

### Scale Operations

**Best for Scale:** rabbit_s3 (won 1/3)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | rabbit_s3 | 401 ops/s | ~equal |
| Scale/List/10 | liteio | 1.5K ops/s | ~equal |
| Scale/Write/10 | usagi_s3 | 1.3 MB/s | ~equal |

---

*Generated by storage benchmark CLI*
