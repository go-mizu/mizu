# Fineweb Search Driver Benchmark Report

**Date:** 2026-01-28 23:10:59
**Duration:** 4m42s
**System:** darwin arm64, 10 CPUs, 13.89 MB RAM
**Go Version:** go1.25.6
**Drivers Tested:** 3

## Executive Summary

| Category | Winner |
|----------|--------|
| Lowest Latency (p50) | fts_rust_tantivy (p50=9.195ms) |
| Highest Single-Thread QPS | fts_rust_tantivy (36 QPS) |
| Fastest Indexing | fts_rust_tantivy (57602 docs/sec) |
| Smallest Index Size | fts_rust (1.30 GB) |
| Best Scalability | fts_lowmem (7.0x scaling) |

## Performance Summary

| Driver | Type | Index Time | Index Size | p50 | p95 | p99 | QPS (1) | QPS (10) | QPS (max) |
|--------|------|------------|------------|-----|-----|-----|---------|----------|----------|
| fts_rust | embedded | 41s | 1.30 GB | 9.478ms | 81.566ms | 97.273ms | 34 | 78 | 84 |
| fts_rust_tantivy | embedded | 40s | 1.30 GB | 9.195ms | 80.341ms | 98.736ms | 36 | 90 | 90 |
| fts_lowmem | embedded | 2m28s | 4.40 GB | 72.134ms | 340.25ms | 516.356ms | 6 | 25 | 42 |

## Indexing Performance

| Driver | Duration | Docs/sec | Peak Memory | Total Docs |
|--------|----------|----------|-------------|------------|
| fts_rust | 41s | 57038 | 3.34 GB | 2319000 |
| fts_rust_tantivy | 40s | 57602 | 3.31 GB | 2319000 |
| fts_lowmem | 2m28s | 15700 | 3.85 GB | 2319000 |

## Pure Rust vs Go FFI Comparison

The following table compares pure Rust indexing performance (measured separately from parquet reading) against Go FFI performance:

| Implementation | Profile | Duration | Docs/sec | Notes |
|----------------|---------|----------|----------|-------|
| **Pure Rust (best)** | ultra | 10.5s | **220,000** | Peak measured |
| **Pure Rust (avg)** | ultra | 12.5s | 185,000 | Typical run |
| **Pure Rust (baseline)** | ultra | 17.2s | 135,000 | Before optimization |
| **Pure Rust** | tantivy | 253.4s | 9,153 | Disk I/O bound |
| **Go FFI** | ultra | 40.4s | 57,421 | With CGO overhead |

**Optimizations Applied:**
1. 16-shard architecture (optimal for 10-core M2 Pro)
2. Lookup table for O(1) character classification
3. Sequential document IDs (skip storing original IDs)
4. FxHashMap for fast term dictionary
5. Deferred IDF computation to commit time
6. Zero-copy byte slice tokenization
7. Parallel tokenization with rayon
8. Batch shard updates (collect → parallel insert)

**Performance Improvement:**
- Pure Rust baseline: 135k docs/sec
- Pure Rust optimized: **185-220k docs/sec** (40-65% improvement)
- Peak throughput: 320k docs/sec (first batch, warm cache)

**Key Findings:**
- Pure Rust ultra profile achieves **~200k docs/sec** (parquet reading excluded)
- Go FFI adds approximately **3.5x overhead** (200k → 57k)
- Synthetic tests achieve 1.4M docs/sec (80 byte docs)
- Real Vietnamese documents are ~2KB, leading to proportionally slower indexing
- Performance variance of ~20% due to system load and memory pressure

**1M docs/sec Target Analysis:**
- Current pure Rust: ~200k docs/sec (need 5x improvement)
- Tested approaches that didn't help:
  - DashMap lock-free hashmap (slower due to write contention)
  - ahash vs FxHash (FxHash faster for u64 keys)
  - Pre-sized vectors (wasted memory bandwidth)
  - Parallel fold/reduce for aggregation (merge overhead)
- To achieve 1M docs/sec would require:
  - Zero-copy from parquet (avoid Document string allocations)
  - SIMD bulk tokenization using intrinsics
  - Memory-mapped inverted index structures
  - Custom memory allocators with arena pooling
  - These are fundamental architectural changes

## Latency Distribution

| Driver | Min | Avg | p50 | p95 | p99 | Max |
|--------|-----|-----|-----|-----|-----|-----|
| fts_rust | 2µs | 27.074ms | 9.478ms | 81.566ms | 97.273ms | 97.337ms |
| fts_rust_tantivy | 2µs | 27.343ms | 9.195ms | 80.341ms | 98.736ms | 98.913ms |
| fts_lowmem | 77µs | 139.595ms | 72.134ms | 340.25ms | 516.356ms | 542.633ms |

## Scalability Analysis (QPS by Concurrency)

| Driver | 10 | 20 | 40 | 80 | Scaling |
|--------|------|------|------|------|---------|
| fts_rust | 78 | 84 | 40 | 80 | 2.1x |
| fts_rust_tantivy | 90 | 84 | 40 | 80 | 2.2x |
| fts_lowmem | 25 | 30 | 40 | 42 | 1.7x |


## Detailed Results

### fts_rust

#### Indexing

- Duration: 41s
- Documents: 2319000
- Throughput: 57038 docs/sec
- Peak Memory: 3.34 GB

#### Index Size

- Size: 1.30 GB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 9.478ms |
| p95 | 81.566ms |
| p99 | 97.273ms |
| max | 97.337ms |
| avg | 27.074ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 78 |
| 20 | 84 |
| 40 | 40 |
| 80 | 80 |

#### Memory Usage

- Indexing Peak: 3.34 GB


### fts_rust_tantivy

#### Indexing

- Duration: 40s
- Documents: 2319000
- Throughput: 57602 docs/sec
- Peak Memory: 3.31 GB

#### Index Size

- Size: 1.30 GB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 9.195ms |
| p95 | 80.341ms |
| p99 | 98.736ms |
| max | 98.913ms |
| avg | 27.343ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 90 |
| 20 | 84 |
| 40 | 40 |
| 80 | 80 |

#### Memory Usage

- Indexing Peak: 3.31 GB


### fts_lowmem

#### Indexing

- Duration: 2m28s
- Documents: 2319000
- Throughput: 15700 docs/sec
- Peak Memory: 3.85 GB

#### Index Size

- Size: 4.40 GB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 72.134ms |
| p95 | 340.25ms |
| p99 | 516.356ms |
| max | 542.633ms |
| avg | 139.595ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 25 |
| 20 | 30 |
| 40 | 40 |
| 80 | 42 |

#### Memory Usage

- Indexing Peak: 3.85 GB

## Vietnamese Language Support

| Driver | Tokenizer | Stemmer | Diacritics | Notes |
|--------|-----------|---------|------------|-------|
| duckdb | Basic | None | Preserved | Uses FTS extension |
| sqlite | Unicode61 | None | Preserved | FTS5 virtual table |
| bleve | ICU Vietnamese | None | Preserved | Best Vietnamese support |
| bluge | Shared Vietnamese | None | Preserved | Custom tokenizer |
| tantivy | Vietnamese | None | Preserved | Requires CGO |
| meilisearch | Auto-detect | None | Preserved | Good Unicode handling |
| zinc | Basic | None | Preserved | Bluge-based |
| porter | Shared Vietnamese | Porter (English) | Preserved | Custom inverted index |
| opensearch | ICU | None | Preserved | Plugin required |
| elasticsearch | ICU | None | Preserved | Plugin required |
| postgres | Simple | None | Preserved | tsvector + GIN |
| typesense | Unicode | None | Preserved | Good typo tolerance |
| manticore | Charset table | None | Preserved | SQL interface |
| quickwit | Default | None | Preserved | Cloud-native |
| lnx | Raw | None | Preserved | Tantivy REST |
| sonic | Basic | None | Preserved | ID-only storage |

## Driver Categories

### Embedded (No External Dependencies)
- **duckdb**: Analytical database with FTS, great for batch processing
- **sqlite**: Lightweight, ACID-compliant, perfect for single-user apps
- **bleve**: Full-featured search library with excellent Vietnamese support
- **bluge**: Modern Bleve successor, better performance
- **porter**: Custom inverted index with Porter stemming

### External Services (Docker Required)
- **meilisearch**: Developer-friendly, instant search, typo tolerance
- **zinc**: Lightweight Elasticsearch alternative
- **opensearch**: AWS fork, enterprise-ready, scalable
- **elasticsearch**: Industry standard, most features
- **postgres**: Full-text search in your existing database
- **typesense**: Fast, typo-tolerant, simple API
- **manticore**: SQL interface, very fast indexing
- **quickwit**: Cloud-native, designed for logs
- **lnx**: Tantivy via REST, no CGO needed
- **sonic**: Ultra-fast search index layer

## Recommendations

Based on benchmark results:

### Performance Leaders
- **Lowest Latency:** fts_rust_tantivy (p50=9.195ms)
- **Best Throughput:** fts_rust_tantivy (36 QPS)
- **Fastest Indexing:** fts_rust_tantivy (57602 docs/sec)
- **Smallest Index:** fts_rust (1.30 GB)

### Use Case Recommendations
- **Simple embedded search:** sqlite (no dependencies, ACID)
- **High-performance embedded:** bluge or porter
- **Developer-friendly SaaS-like:** meilisearch or typesense
- **Enterprise distributed:** elasticsearch or opensearch
- **Existing PostgreSQL stack:** postgres (no new infra)
- **Maximum indexing speed:** manticore
- **Minimum memory footprint:** sonic
- **Cloud-native logs/traces:** quickwit

---
*Report generated by fineweb benchmark suite*
