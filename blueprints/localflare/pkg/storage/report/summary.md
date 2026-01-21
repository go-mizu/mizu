# Storage Benchmark Summary

**Generated:** 2026-01-21T17:46:37+07:00

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
| Copy/1KB | **usagi_s3** | 1.2 MB/s | minio | 0.8 MB/s | +44% |
| Delete | **usagi_s3** | 3.9K ops/s | minio | 2.3K ops/s | +65% |
| EdgeCase/DeepNested | **usagi_s3** | 0.1 MB/s | minio | 0.1 MB/s | +68% |
| EdgeCase/EmptyObject | **usagi_s3** | 1.6K ops/s | minio | 896 ops/s | +78% |
| EdgeCase/LongKey256 | **usagi_s3** | 0.1 MB/s | minio | 0.1 MB/s | +58% |
| List/100 | **usagi_s3** | 886 ops/s | minio | 496 ops/s | +79% |
| MixedWorkload/Balanced_50_50 | **usagi_s3** | 0.3 MB/s | minio | 0.2 MB/s | +28% |
| MixedWorkload/ReadHeavy_90_10 | **usagi_s3** | 0.6 MB/s | minio | 0.4 MB/s | +35% |
| MixedWorkload/WriteHeavy_10_90 | **usagi_s3** | 0.2 MB/s | minio | 0.2 MB/s | +23% |
| Multipart/15MB_3Parts | **minio** | 121.9 MB/s | usagi_s3 | 107.8 MB/s | +13% |
| ParallelRead/1KB/C1 | **usagi_s3** | 3.2 MB/s | minio | 2.3 MB/s | +38% |
| ParallelRead/1KB/C10 | **usagi_s3** | 1.0 MB/s | minio | 0.7 MB/s | +50% |
| ParallelRead/1KB/C100 | **usagi_s3** | 0.2 MB/s | minio | 0.1 MB/s | +88% |
| ParallelRead/1KB/C200 | **usagi_s3** | 0.1 MB/s | minio | 0.0 MB/s | +70% |
| ParallelRead/1KB/C25 | **usagi_s3** | 0.5 MB/s | minio | 0.3 MB/s | +51% |
| ParallelRead/1KB/C50 | **usagi_s3** | 0.3 MB/s | minio | 0.2 MB/s | +63% |
| ParallelWrite/1KB/C1 | **usagi_s3** | 1.2 MB/s | minio | 1.0 MB/s | +14% |
| ParallelWrite/1KB/C10 | **usagi_s3** | 0.4 MB/s | minio | 0.2 MB/s | +64% |
| ParallelWrite/1KB/C100 | **usagi_s3** | 0.0 MB/s | minio | 0.0 MB/s | +42% |
| ParallelWrite/1KB/C200 | **usagi_s3** | 0.0 MB/s | minio | 0.0 MB/s | +51% |
| ParallelWrite/1KB/C25 | **minio** | 0.1 MB/s | usagi_s3 | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C50 | **usagi_s3** | 0.1 MB/s | minio | 0.0 MB/s | +29% |
| RangeRead/End_256KB | **usagi_s3** | 148.6 MB/s | minio | 116.7 MB/s | +27% |
| RangeRead/Middle_256KB | **usagi_s3** | 142.7 MB/s | minio | 116.0 MB/s | +23% |
| RangeRead/Start_256KB | **usagi_s3** | 132.8 MB/s | minio | 94.1 MB/s | +41% |
| Read/100MB | **minio** | 187.9 MB/s | usagi_s3 | 171.6 MB/s | ~equal |
| Read/10MB | **minio** | 172.8 MB/s | usagi_s3 | 162.1 MB/s | ~equal |
| Read/1KB | **usagi_s3** | 3.8 MB/s | minio | 2.8 MB/s | +39% |
| Read/1MB | **usagi_s3** | 167.6 MB/s | minio | 162.1 MB/s | ~equal |
| Read/64KB | **usagi_s3** | 91.8 MB/s | minio | 76.8 MB/s | +19% |
| Scale/Delete/10 | **usagi_s3** | 368 ops/s | minio | 229 ops/s | +61% |
| Scale/Delete/100 | **usagi_s3** | 38 ops/s | minio | 22 ops/s | +70% |
| Scale/Delete/1000 | **usagi_s3** | 3 ops/s | minio | 2 ops/s | +53% |
| Scale/Delete/10000 | **usagi_s3** | 0 ops/s | minio | 0 ops/s | +71% |
| Scale/List/10 | **usagi_s3** | 2.1K ops/s | minio | 1.2K ops/s | +77% |
| Scale/List/100 | **usagi_s3** | 961 ops/s | minio | 405 ops/s | 2.4x faster |
| Scale/List/1000 | **usagi_s3** | 129 ops/s | minio | 64 ops/s | 2.0x faster |
| Scale/List/10000 | **minio** | 5 ops/s | usagi_s3 | 4 ops/s | +22% |
| Scale/Write/10 | **usagi_s3** | 0.3 MB/s | minio | 0.2 MB/s | +44% |
| Scale/Write/100 | **usagi_s3** | 0.4 MB/s | minio | 0.2 MB/s | +56% |
| Scale/Write/1000 | **usagi_s3** | 0.3 MB/s | minio | 0.2 MB/s | +52% |
| Scale/Write/10000 | **usagi_s3** | 0.4 MB/s | minio | 0.2 MB/s | +77% |
| Stat | **usagi_s3** | 3.7K ops/s | minio | 3.1K ops/s | +20% |
| Write/100MB | **minio** | 148.5 MB/s | usagi_s3 | 135.0 MB/s | +10% |
| Write/10MB | **minio** | 138.5 MB/s | usagi_s3 | 127.8 MB/s | ~equal |
| Write/1KB | **usagi_s3** | 1.8 MB/s | minio | 1.3 MB/s | +36% |
| Write/1MB | **minio** | 113.3 MB/s | usagi_s3 | 108.6 MB/s | ~equal |
| Write/64KB | **usagi_s3** | 56.2 MB/s | minio | 21.3 MB/s | 2.6x faster |

## Category Summaries

### Write Operations

**Best for Write:** minio (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Write/100MB | minio | 148.5 MB/s | +10% |
| Write/10MB | minio | 138.5 MB/s | ~equal |
| Write/1KB | usagi_s3 | 1.8 MB/s | +36% |
| Write/1MB | minio | 113.3 MB/s | ~equal |
| Write/64KB | usagi_s3 | 56.2 MB/s | 2.6x faster |

### Read Operations

**Best for Read:** usagi_s3 (won 3/5)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Read/100MB | minio | 187.9 MB/s | ~equal |
| Read/10MB | minio | 172.8 MB/s | ~equal |
| Read/1KB | usagi_s3 | 3.8 MB/s | +39% |
| Read/1MB | usagi_s3 | 167.6 MB/s | ~equal |
| Read/64KB | usagi_s3 | 91.8 MB/s | +19% |

### ParallelWrite Operations

**Best for ParallelWrite:** usagi_s3 (won 5/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelWrite/1KB/C1 | usagi_s3 | 1.2 MB/s | +14% |
| ParallelWrite/1KB/C10 | usagi_s3 | 0.4 MB/s | +64% |
| ParallelWrite/1KB/C100 | usagi_s3 | 0.0 MB/s | +42% |
| ParallelWrite/1KB/C200 | usagi_s3 | 0.0 MB/s | +51% |
| ParallelWrite/1KB/C25 | minio | 0.1 MB/s | ~equal |
| ParallelWrite/1KB/C50 | usagi_s3 | 0.1 MB/s | +29% |

### ParallelRead Operations

**Best for ParallelRead:** usagi_s3 (won 6/6)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| ParallelRead/1KB/C1 | usagi_s3 | 3.2 MB/s | +38% |
| ParallelRead/1KB/C10 | usagi_s3 | 1.0 MB/s | +50% |
| ParallelRead/1KB/C100 | usagi_s3 | 0.2 MB/s | +88% |
| ParallelRead/1KB/C200 | usagi_s3 | 0.1 MB/s | +70% |
| ParallelRead/1KB/C25 | usagi_s3 | 0.5 MB/s | +51% |
| ParallelRead/1KB/C50 | usagi_s3 | 0.3 MB/s | +63% |

### Delete Operations

**Best for Delete:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Delete | usagi_s3 | 3.9K ops/s | +65% |

### Stat Operations

**Best for Stat:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Stat | usagi_s3 | 3.7K ops/s | +20% |

### List Operations

**Best for List:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| List/100 | usagi_s3 | 886 ops/s | +79% |

### Copy Operations

**Best for Copy:** usagi_s3 (won 1/1)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Copy/1KB | usagi_s3 | 1.2 MB/s | +44% |

### Scale Operations

**Best for Scale:** usagi_s3 (won 11/12)

| Operation | Winner | Performance | vs Runner-up |
|-----------|--------|-------------|-------------|
| Scale/Delete/10 | usagi_s3 | 368 ops/s | +61% |
| Scale/Delete/100 | usagi_s3 | 38 ops/s | +70% |
| Scale/Delete/1000 | usagi_s3 | 3 ops/s | +53% |
| Scale/Delete/10000 | usagi_s3 | 0 ops/s | +71% |
| Scale/List/10 | usagi_s3 | 2.1K ops/s | +77% |
| Scale/List/100 | usagi_s3 | 961 ops/s | 2.4x faster |
| Scale/List/1000 | usagi_s3 | 129 ops/s | 2.0x faster |
| Scale/List/10000 | minio | 5 ops/s | +22% |
| Scale/Write/10 | usagi_s3 | 0.3 MB/s | +44% |
| Scale/Write/100 | usagi_s3 | 0.4 MB/s | +56% |
| Scale/Write/1000 | usagi_s3 | 0.3 MB/s | +52% |
| Scale/Write/10000 | usagi_s3 | 0.4 MB/s | +77% |

---

*Generated by storage benchmark CLI*
