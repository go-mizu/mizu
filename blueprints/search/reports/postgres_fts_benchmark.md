# Fineweb Search Driver Benchmark Report

**Date:** 2026-01-28 07:35:38
**Duration:** 1m14s
**System:** darwin arm64, 10 CPUs, 17.64 MB RAM
**Go Version:** go1.25.6
**Drivers Tested:** 4

## Executive Summary

| Category | Winner |
|----------|--------|
| Lowest Latency (p50) | postgres_trgm (p50=385µs) |
| Highest Single-Thread QPS | postgres_trgm (2728 QPS) |
| Fastest Indexing | postgres_pgsearch (5751 docs/sec) |
| Best Scalability | postgres_pgsearch (5.6x scaling) |

## Performance Summary

| Driver | Type | Index Time | Index Size | p50 | p95 | p99 | QPS (1) | QPS (10) | QPS (max) |
|--------|------|------------|------------|-----|-----|-----|---------|----------|----------|
| postgres | external | - | - | 17.037ms | 71.057ms | 113.276ms | 38 | 140 | 142 |
| postgres_pgsearch | embedded | 5s | - | 9.372ms | 12.239ms | 13.454ms | 93 | 501 | 520 |
| postgres_pgroonga | embedded | 8s | - | 12.939ms | 66.218ms | 68.492ms | 28 | 58 | 80 |
| postgres_trgm | embedded | 13s | - | 385µs | 606µs | 768µs | 2728 | 7622 | 7622 |

## Indexing Performance

| Driver | Duration | Docs/sec | Peak Memory | Total Docs |
|--------|----------|----------|-------------|------------|
| postgres_pgsearch | 5s | 5751 | 200.79 MB | 28276 |
| postgres_pgroonga | 8s | 3637 | 311.14 MB | 28276 |
| postgres_trgm | 13s | 2121 | 256.43 MB | 28276 |

## Latency Distribution

| Driver | Min | Avg | p50 | p95 | p99 | Max |
|--------|-----|-----|-----|-----|-----|-----|
| postgres | 742µs | 27.683ms | 17.037ms | 71.057ms | 113.276ms | 187.927ms |
| postgres_pgsearch | 6.304ms | 9.785ms | 9.372ms | 12.239ms | 13.454ms | 29.015ms |
| postgres_pgroonga | 866µs | 34.215ms | 12.939ms | 66.218ms | 68.492ms | 174.508ms |
| postgres_trgm | 314µs | 446µs | 385µs | 606µs | 768µs | 2.126ms |

## Scalability Analysis (QPS by Concurrency)

| Driver | 10 | 20 | 40 | 80 | Scaling |
|--------|------|------|------|------|---------|
| postgres | 140 | 142 | 94 | 118 | 1.5x |
| postgres_pgsearch | 501 | 520 | 505 | 520 | 1.0x |
| postgres_pgroonga | 58 | 58 | 60 | 80 | 1.4x |
| postgres_trgm | 7622 | 7224 | 7204 | 7148 | 1.1x |


## Detailed Results

### postgres

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 17.037ms |
| p95 | 71.057ms |
| p99 | 113.276ms |
| max | 187.927ms |
| avg | 27.683ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 140 |
| 20 | 142 |
| 40 | 94 |
| 80 | 118 |


### postgres_pgsearch

#### Indexing

- Duration: 5s
- Documents: 28276
- Throughput: 5751 docs/sec
- Peak Memory: 200.79 MB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 9.372ms |
| p95 | 12.239ms |
| p99 | 13.454ms |
| max | 29.015ms |
| avg | 9.785ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 501 |
| 20 | 520 |
| 40 | 505 |
| 80 | 520 |

#### Memory Usage

- Indexing Peak: 200.79 MB


### postgres_pgroonga

#### Indexing

- Duration: 8s
- Documents: 28276
- Throughput: 3637 docs/sec
- Peak Memory: 311.14 MB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 12.939ms |
| p95 | 66.218ms |
| p99 | 68.492ms |
| max | 174.508ms |
| avg | 34.215ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 58 |
| 20 | 58 |
| 40 | 60 |
| 80 | 80 |

#### Memory Usage

- Indexing Peak: 311.14 MB


### postgres_trgm

#### Indexing

- Duration: 13s
- Documents: 28276
- Throughput: 2121 docs/sec
- Peak Memory: 256.43 MB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 385µs |
| p95 | 606µs |
| p99 | 768µs |
| max | 2.126ms |
| avg | 446µs |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 7622 |
| 20 | 7224 |
| 40 | 7204 |
| 80 | 7148 |

#### Memory Usage

- Indexing Peak: 256.43 MB

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
- **Lowest Latency:** postgres_trgm (p50=385µs)
- **Best Throughput:** postgres_trgm (2728 QPS)
- **Fastest Indexing:** postgres_pgsearch (5751 docs/sec)

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
