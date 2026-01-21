# Storage Benchmark Summary

**Generated:** 2026-01-21T17:42:23+07:00

## Overall Winner

**usagi_s3** won 40/48 categories (83%)

### Win Counts

| Driver | Wins | Percentage |
|--------|------|------------|
| usagi_s3 | 40 | 83% |
| minio | 8 | 17% |

## Best Driver by Category

| Category | Winner | Performance | Runner-up | Runner-up Perf | Margin |
|----------|--------|-------------|-----------|----------------|--------|
| Copy/1KB | **usagi_s3** | 1.0 MB/s | minio | 0.9 MB/s | +20% |
| Delete | **usagi_s3** | 3.8K ops/s | minio | 2.5K ops/s | +51% |
| EdgeCase/DeepNested | **usagi_s3** | 0.1 MB/s | minio | 0.1 MB/s | +50% |
| EdgeCase/EmptyObject | **usagi_s3** | 1.6K ops/s | minio | 965 ops/s | +64% |
| EdgeCase/LongKey256 | **usagi_s3** | 0.1 MB/s | minio | 0.1 MB/s | +71% |
| List/100 | **usagi_s3** | 869 ops/s | minio | 528 ops/s | +65% |
| MixedWorkload/Balanced_50_50 | **usagi_s3** | 0.4 MB/s | minio | 0.2 MB/s | +65% |
| MixedWorkload/ReadHeavy_90_10 | **usagi_s3** | 0.6 MB/s | minio | 0.4 MB/s | +38% |
| MixedWorkload/WriteHeavy_10_90 | **usagi_s3** | 0.3 MB/s | minio | 0.2 MB/s | +53% |
| Multipart/15MB_3Parts | **minio** | 118.3 MB/s | usagi_s3 | 99.1 MB/s | +19% |
| ParallelRead/1KB/C1 | **usagi_s3** | 3.1 MB/s | minio | 2.3 MB/s | +33% |
| ParallelRead/1KB/C10 | **usagi_s3** | 1.0 MB/s | minio | 0.7 MB/s | +45% |
| ParallelRead/1KB/C100 | **usagi_s3** | 0.2 MB/s | minio | 0.1 MB/s | +80% |
| ParallelRead/1KB/C200 | **usagi_s3** | 0.1 MB/s | minio | 0.0 MB/s | +73% |
| ParallelRead/1KB/C25 | **usagi_s3** | 0.5 MB/s | minio | 0.3 MB/s | +54% |
| ParallelRead/1KB/C50 | **usagi_s3** | 0.3 MB/s | minio | 0.2 MB/s | +53% |
| ParallelWrite/1KB/C1 | **usagi_s3** | 1.3 MB/s | minio | 1.1 MB/s | +24% |
| ParallelWrite/1KB/C10 | **minio** | 0.3 MB/s | usagi_s3 | 0.3 MB/s | ~equal |
| ParallelWrite/1KB/C100 | **usagi_s3** | 0.0 MB/s | minio | 0.0 MB/s | +33% |
| ParallelWrite/1KB/C200 | **usagi_s3** | 0.0 MB/s | minio | 0.0 MB/s | +33% |
| ParallelWrite/1KB/C25 | **usagi_s3** | 0.1 MB/s | minio | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C50 | **usagi_s3** | 0.1 MB/s | minio | 0.0 MB/s | +16% |
| RangeRead/End_256KB | **usagi_s3** | 146.8 MB/s | minio | 118.2 MB/s | +24% |
| RangeRead/Middle_256KB | **usagi_s3** | 144.3 MB/s | minio | 118.6 MB/s | +22% |
| RangeRead/Start_256KB | **usagi_s3** | 131.2 MB/s | minio | 91.9 MB/s | +43% |
| Read/100MB | **minio** | 196.9 MB/s | usagi_s3 | 164.5 MB/s | +20% |
| Read/10MB | **minio** | 199.2 MB/s | usagi_s3 | 163.6 MB/s | +22% |
| Read/1KB | **usagi_s3** | 3.9 MB/s | minio | 2.7 MB/s | +43% |
| Read/1MB | **minio** | 172.0 MB/s | usagi_s3 | 162.9 MB/s | ~equal |
| Read/64KB | **usagi_s3** | 95.2 MB/s | minio | 90.6 MB/s | ~equal |
| Scale/Delete/10 | **usagi_s3** | 388 ops/s | minio | 216 ops/s | +80% |
| Scale/Delete/100 | **usagi_s3** | 41 ops/s | minio | 24 ops/s | +70% |
| Scale/Delete/1000 | **usagi_s3** | 4 ops/s | minio | 2 ops/s | +54% |
| Scale/Delete/10000 | **usagi_s3** | 0 ops/s | minio | 0 ops/s | +61% |
| Scale/List/10 | **minio** | 989 ops/s | usagi_s3 | 946 ops/s | ~equal |
| Scale/List/100 | **usagi_s3** | 836 ops/s | minio | 436 ops/s | +92% |
| Scale/List/1000 | **usagi_s3** | 125 ops/s | minio | 68 ops/s | +83% |
| Scale/List/10000 | **minio** | 6 ops/s | usagi_s3 | 4 ops/s | +43% |
| Scale/Write/10 | **usagi_s3** | 0.4 MB/s | minio | 0.2 MB/s | +53% |
| Scale/Write/100 | **usagi_s3** | 0.3 MB/s | minio | 0.2 MB/s | +17% |
| Scale/Write/1000 | **usagi_s3** | 0.3 MB/s | minio | 0.2 MB/s | +47% |
| Scale/Write/10000 | **usagi_s3** | 0.4 MB/s | minio | 0.2 MB/s | +72% |
| Stat | **usagi_s3** | 3.8K ops/s | minio | 3.3K ops/s | +17% |
| Write/100MB | **minio** | 130.5 MB/s | usagi_s3 | 125.0 MB/s | ~equal |
| Write/10MB | **usagi_s3** | 130.5 MB/s | minio | 126.0 MB/s | ~equal |
| Write/1KB | **usagi_s3** | 1.8 MB/s | minio | 1.2 MB/s | +45% |
| Write/1MB | **usagi_s3** | 105.0 MB/s | minio | 103.2 MB/s | ~equal |
| Write/64KB | **usagi_s3** | 59.6 MB/s | minio | 43.1 MB/s | +38% |

## Category Summaries

### Write Operations

**Best for Write:** usagi_s3 (won 4/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | minio | 130.5 MB/s | ~equal |
| Write/10MB | usagi_s3 | 130.5 MB/s | ~equal |
| Write/1KB | usagi_s3 | 1.8 MB/s | +45% |
| Write/1MB | usagi_s3 | 105.0 MB/s | ~equal |
| Write/64KB | usagi_s3 | 59.6 MB/s | +38% |

### Read Operations

**Best for Read:** minio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 196.9 MB/s | +20% |
| Read/10MB | minio | 199.2 MB/s | +22% |
| Read/1KB | usagi_s3 | 3.9 MB/s | +43% |
| Read/1MB | minio | 172.0 MB/s | ~equal |
| Read/64KB | usagi_s3 | 95.2 MB/s | ~equal |

### ParallelWrite Operations

**Best for ParallelWrite:** usagi_s3 (won 5/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | usagi_s3 | 1.3 MB/s | +24% |
| ParallelWrite/1KB/C10 | minio | 0.3 MB/s | ~equal |
| ParallelWrite/1KB/C100 | usagi_s3 | 0.0 MB/s | +33% |
| ParallelWrite/1KB/C200 | usagi_s3 | 0.0 MB/s | +33% |
| ParallelWrite/1KB/C25 | usagi_s3 | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C50 | usagi_s3 | 0.1 MB/s | +16% |

### ParallelRead Operations

**Best for ParallelRead:** usagi_s3 (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | usagi_s3 | 3.1 MB/s | +33% |
| ParallelRead/1KB/C10 | usagi_s3 | 1.0 MB/s | +45% |
| ParallelRead/1KB/C100 | usagi_s3 | 0.2 MB/s | +80% |
| ParallelRead/1KB/C200 | usagi_s3 | 0.1 MB/s | +73% |
| ParallelRead/1KB/C25 | usagi_s3 | 0.5 MB/s | +54% |
| ParallelRead/1KB/C50 | usagi_s3 | 0.3 MB/s | +53% |

### Delete Operations

**Best for Delete:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | usagi_s3 | 3.8K ops/s | +51% |

### Stat Operations

**Best for Stat:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | usagi_s3 | 3.8K ops/s | +17% |

### List Operations

**Best for List:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | usagi_s3 | 869 ops/s | +65% |

### Copy Operations

**Best for Copy:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | usagi_s3 | 1.0 MB/s | +20% |

### Scale Operations

**Best for Scale:** usagi_s3 (won 10/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | usagi_s3 | 388 ops/s | +80% |
| Scale/Delete/100 | usagi_s3 | 41 ops/s | +70% |
| Scale/Delete/1000 | usagi_s3 | 4 ops/s | +54% |
| Scale/Delete/10000 | usagi_s3 | 0 ops/s | +61% |
| Scale/List/10 | minio | 989 ops/s | ~equal |
| Scale/List/100 | usagi_s3 | 836 ops/s | +92% |
| Scale/List/1000 | usagi_s3 | 125 ops/s | +83% |
| Scale/List/10000 | minio | 6 ops/s | +43% |
| Scale/Write/10 | usagi_s3 | 0.4 MB/s | +53% |
| Scale/Write/100 | usagi_s3 | 0.3 MB/s | +17% |
| Scale/Write/1000 | usagi_s3 | 0.3 MB/s | +47% |
| Scale/Write/10000 | usagi_s3 | 0.4 MB/s | +72% |

---

*Generated by storage benchmark CLI*
