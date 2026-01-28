# Fineweb Search Driver Benchmark Report

**Date:** 2026-01-28 02:28:10
**Duration:** 6m47s
**System:** darwin arm64, 10 CPUs, 13.64 MB RAM
**Go Version:** go1.25.6
**Drivers Tested:** 13

## Executive Summary

| Category | Winner |
|----------|--------|
| Lowest Latency (p50) | manticore (p50=283µs) |
| Highest Single-Thread QPS | manticore (3402 QPS) |
| Fastest Indexing | quickwit (12157 docs/sec) |
| Smallest Index Size | duckdb (121.51 MB) |
| Best Scalability | opensearch (7.2x scaling) |

## Performance Summary

| Driver | Type | Index Time | Index Size | p50 | p95 | p99 | QPS (1) | QPS (10) | QPS (max) |
|--------|------|------------|------------|-----|-----|-----|---------|----------|----------|
| bleve | embedded | 13s | 708.36 MB | 1.203ms | 4.333ms | 5.165ms | 1022 | 2819 | 2819 |
| bluge | embedded | 9s | 344.46 MB | 1.086ms | 3.521ms | 4.573ms | 627 | 1930 | 1930 |
| duckdb | embedded | 13s | 121.51 MB | 6.646ms | 46.638ms | 50.825ms | 94 | 393 | 418 |
| elasticsearch | external | 9s | - | 6.892ms | 15.602ms | 19.639ms | 159 | 814 | 1102 |
| manticore | external | - | - | 283µs | 429µs | 462µs | 3402 | 10326 | 10326 |
| meilisearch | external | 2m2s | - | 455µs | 1.587ms | 2.465ms | 2928 | 8740 | 8932 |
| opensearch | external | 8s | - | 6.257ms | 13.166ms | 16.905ms | 178 | 948 | 1290 |
| porter | embedded | 26s | 592.96 MB | 504µs | 2.271ms | 3.019ms | 1074 | 4630 | 4630 |
| postgres | external | 17s | - | 37.23ms | 110.665ms | 114.31ms | 18 | 85 | 120 |
| quickwit | external | 2s | - | 507µs | 907µs | 1.417ms | 1992 | 3144 | 3784 |
| sqlite | embedded | 12s | 262.77 MB | 8.155ms | 58.359ms | 92.798ms | 74 | 95 | 95 |
| typesense | external | - | - | 19.183ms | 47.218ms | 59.586ms | 71 | 208 | 212 |
| zinc | external | - | - | 10.371ms | 22.102ms | 25.532ms | 266 | 889 | 944 |

## Indexing Performance

| Driver | Duration | Docs/sec | Peak Memory | Total Docs |
|--------|----------|----------|-------------|------------|
| bleve | 13s | 2139 | 834.01 MB | 28276 |
| bluge | 9s | 3298 | 577.63 MB | 28276 |
| duckdb | 13s | 2191 | 173.39 MB | 28276 |
| elasticsearch | 9s | 3210 | 275.43 MB | 28276 |
| meilisearch | 2m2s | 232 | 418.25 MB | 28276 |
| opensearch | 8s | 3431 | 316.60 MB | 28276 |
| porter | 26s | 1078 | 2.64 GB | 28276 |
| postgres | 17s | 1668 | 2.64 GB | 28276 |
| quickwit | 2s | 12157 | 280.09 MB | 28276 |
| sqlite | 12s | 2378 | 264.65 MB | 28276 |

## Latency Distribution

| Driver | Min | Avg | p50 | p95 | p99 | Max |
|--------|-----|-----|-----|-----|-----|-----|
| bleve | 134µs | 1.493ms | 1.203ms | 4.333ms | 5.165ms | 5.642ms |
| bluge | 84µs | 1.575ms | 1.086ms | 3.521ms | 4.573ms | 5.83ms |
| duckdb | 2.036ms | 11.176ms | 6.646ms | 46.638ms | 50.825ms | 52.57ms |
| elasticsearch | 4.188ms | 8.727ms | 6.892ms | 15.602ms | 19.639ms | 58.334ms |
| manticore | 218µs | 302µs | 283µs | 429µs | 462µs | 475µs |
| meilisearch | 338µs | 664µs | 455µs | 1.587ms | 2.465ms | 4.909ms |
| opensearch | 4.085ms | 7.745ms | 6.257ms | 13.166ms | 16.905ms | 43.63ms |
| porter | 5µs | 921µs | 504µs | 2.271ms | 3.019ms | 3.207ms |
| postgres | 8.99ms | 58.492ms | 37.23ms | 110.665ms | 114.31ms | 117.717ms |
| quickwit | 362µs | 625µs | 507µs | 907µs | 1.417ms | 5.15ms |
| sqlite | 61µs | 16.352ms | 8.155ms | 58.359ms | 92.798ms | 157.444ms |
| typesense | 7.11ms | 24.278ms | 19.183ms | 47.218ms | 59.586ms | 69.327ms |
| zinc | 2.42ms | 11.06ms | 10.371ms | 22.102ms | 25.532ms | 30.871ms |

## Scalability Analysis (QPS by Concurrency)

| Driver | 10 | 20 | 40 | 80 | Scaling |
|--------|------|------|------|------|---------|
| bleve | 2819 | 2776 | 2774 | 2802 | 1.0x |
| bluge | 1930 | 1878 | 1902 | 1858 | 1.0x |
| duckdb | 393 | 393 | 414 | 418 | 1.1x |
| elasticsearch | 814 | 1007 | 1102 | 1084 | 1.4x |
| manticore | 10326 | 9735 | 9890 | 9958 | 1.1x |
| meilisearch | 8740 | 8932 | 8582 | 8716 | 1.0x |
| opensearch | 948 | 1163 | 1230 | 1290 | 1.4x |
| porter | 4630 | 4265 | 4324 | 4110 | 1.1x |
| postgres | 85 | 89 | 80 | 120 | 1.5x |
| quickwit | 3144 | 3784 | 3737 | 3606 | 1.2x |
| sqlite | 95 | 40 | 60 | 42 | 2.4x |
| typesense | 208 | 210 | 206 | 212 | 1.0x |
| zinc | 889 | 907 | 944 | 888 | 1.1x |


## Detailed Results

### bleve

#### Indexing

- Duration: 13s
- Documents: 28276
- Throughput: 2139 docs/sec
- Peak Memory: 834.01 MB

#### Index Size

- Size: 708.36 MB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 1.203ms |
| p95 | 4.333ms |
| p99 | 5.165ms |
| max | 5.642ms |
| avg | 1.493ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 2819 |
| 20 | 2776 |
| 40 | 2774 |
| 80 | 2802 |

#### Memory Usage

- Indexing Peak: 834.01 MB


### bluge

#### Indexing

- Duration: 9s
- Documents: 28276
- Throughput: 3298 docs/sec
- Peak Memory: 577.63 MB

#### Index Size

- Size: 344.46 MB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 1.086ms |
| p95 | 3.521ms |
| p99 | 4.573ms |
| max | 5.83ms |
| avg | 1.575ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 1930 |
| 20 | 1878 |
| 40 | 1902 |
| 80 | 1858 |

#### Memory Usage

- Indexing Peak: 577.63 MB


### duckdb

#### Indexing

- Duration: 13s
- Documents: 28276
- Throughput: 2191 docs/sec
- Peak Memory: 173.39 MB

#### Index Size

- Size: 121.51 MB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 6.646ms |
| p95 | 46.638ms |
| p99 | 50.825ms |
| max | 52.57ms |
| avg | 11.176ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 393 |
| 20 | 393 |
| 40 | 414 |
| 80 | 418 |

#### Memory Usage

- Indexing Peak: 173.39 MB


### elasticsearch

#### Indexing

- Duration: 9s
- Documents: 28276
- Throughput: 3210 docs/sec
- Peak Memory: 275.43 MB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 6.892ms |
| p95 | 15.602ms |
| p99 | 19.639ms |
| max | 58.334ms |
| avg | 8.727ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 814 |
| 20 | 1007 |
| 40 | 1102 |
| 80 | 1084 |

#### Memory Usage

- Indexing Peak: 275.43 MB


### manticore

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 283µs |
| p95 | 429µs |
| p99 | 462µs |
| max | 475µs |
| avg | 302µs |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 10326 |
| 20 | 9735 |
| 40 | 9890 |
| 80 | 9958 |


### meilisearch

#### Indexing

- Duration: 2m2s
- Documents: 28276
- Throughput: 232 docs/sec
- Peak Memory: 418.25 MB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 455µs |
| p95 | 1.587ms |
| p99 | 2.465ms |
| max | 4.909ms |
| avg | 664µs |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 8740 |
| 20 | 8932 |
| 40 | 8582 |
| 80 | 8716 |

#### Memory Usage

- Indexing Peak: 418.25 MB


### opensearch

#### Indexing

- Duration: 8s
- Documents: 28276
- Throughput: 3431 docs/sec
- Peak Memory: 316.60 MB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 6.257ms |
| p95 | 13.166ms |
| p99 | 16.905ms |
| max | 43.63ms |
| avg | 7.745ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 948 |
| 20 | 1163 |
| 40 | 1230 |
| 80 | 1290 |

#### Memory Usage

- Indexing Peak: 316.60 MB


### porter

#### Indexing

- Duration: 26s
- Documents: 28276
- Throughput: 1078 docs/sec
- Peak Memory: 2.64 GB

#### Index Size

- Size: 592.96 MB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 504µs |
| p95 | 2.271ms |
| p99 | 3.019ms |
| max | 3.207ms |
| avg | 921µs |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 4630 |
| 20 | 4265 |
| 40 | 4324 |
| 80 | 4110 |

#### Memory Usage

- Indexing Peak: 2.64 GB


### postgres

#### Indexing

- Duration: 17s
- Documents: 28276
- Throughput: 1668 docs/sec
- Peak Memory: 2.64 GB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 37.23ms |
| p95 | 110.665ms |
| p99 | 114.31ms |
| max | 117.717ms |
| avg | 58.492ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 85 |
| 20 | 89 |
| 40 | 80 |
| 80 | 120 |

#### Memory Usage

- Indexing Peak: 2.64 GB


### quickwit

#### Indexing

- Duration: 2s
- Documents: 28276
- Throughput: 12157 docs/sec
- Peak Memory: 280.09 MB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 507µs |
| p95 | 907µs |
| p99 | 1.417ms |
| max | 5.15ms |
| avg | 625µs |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 3144 |
| 20 | 3784 |
| 40 | 3737 |
| 80 | 3606 |

#### Memory Usage

- Indexing Peak: 280.09 MB


### sqlite

#### Indexing

- Duration: 12s
- Documents: 28276
- Throughput: 2378 docs/sec
- Peak Memory: 264.65 MB

#### Index Size

- Size: 262.77 MB

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 8.155ms |
| p95 | 58.359ms |
| p99 | 92.798ms |
| max | 157.444ms |
| avg | 16.352ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 95 |
| 20 | 40 |
| 40 | 60 |
| 80 | 42 |

#### Memory Usage

- Indexing Peak: 264.65 MB


### typesense

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 19.183ms |
| p95 | 47.218ms |
| p99 | 59.586ms |
| max | 69.327ms |
| avg | 24.278ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 208 |
| 20 | 210 |
| 40 | 206 |
| 80 | 212 |


### zinc

#### Search Latency

| Percentile | Latency |
|------------|--------|
| p50 | 10.371ms |
| p95 | 22.102ms |
| p99 | 25.532ms |
| max | 30.871ms |
| avg | 11.06ms |

#### Throughput by Concurrency

| Goroutines | QPS |
|------------|-----|
| 10 | 889 |
| 20 | 907 |
| 40 | 944 |
| 80 | 888 |

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
- **Lowest Latency:** manticore (p50=283µs)
- **Best Throughput:** manticore (3402 QPS)
- **Fastest Indexing:** quickwit (12157 docs/sec)
- **Smallest Index:** duckdb (121.51 MB)

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
