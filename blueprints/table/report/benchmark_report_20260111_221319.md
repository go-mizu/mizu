# Storage Backend Benchmark Report

**Generated:** 2026-01-11 22:13:19

**Duration:** 3s

---

## Executive Summary

### Winners by Category

| Category | Best Backend | Notes |
|----------|--------------|-------|
| Single Record | **sqlite** | CRUD operations on individual records |
| Batch Operations | **sqlite** | Bulk insert/delete operations |
| Queries | **sqlite** | List and filter operations |
| Field Operations | **sqlite** | Schema operations |
| Concurrent | **sqlite** | Parallel workloads |

### Key Findings

1. **sqlite** leads in 5 out of 5 categories
2. **duckdb**: Strong performance for analytical workloads
2. **sqlite**: Best for single-writer scenarios

## Environment

| Property | Value |
|----------|-------|
| Go Version | go1.25.5 |
| OS | darwin |
| Architecture | arm64 |
| CPUs | 10 |
| Hostname | USERnoMacBook-Air.local |
| PostgreSQL | (not configured) |
| Data Dir | /var/folders/_g/lq_pglm508df70x751kkxrl80000gp/T/storebench |

### Configuration

| Setting | Value |
|---------|-------|
| Backends | duckdb, sqlite |
| Scenarios | records, batch, query, fields, concurrent |
| Iterations | 50 |
| Concurrency | 25 |
| Warmup | 10 |

## Results by Category

### Single Record

#### record_create

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 445µs | 431µs | 536µs | 615µs | 2225.58 | 0 |
| sqlite | 47µs | 44µs | 61µs | 62µs | 20776.70 | 0 |

**Comparison:**
- duckdb: +855.4% slower
- sqlite: baseline (fastest)

#### record_get_by_id

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 147µs | 142µs | 172µs | 201µs | 1736.82 | 0 |
| sqlite | 13µs | 12µs | 14µs | 16µs | 14111.01 | 0 |

**Comparison:**
- duckdb: +1040.4% slower
- sqlite: baseline (fastest)

#### record_update

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 311µs | 301µs | 365µs | 409µs | 1283.80 | 0 |
| sqlite | 17µs | 17µs | 19µs | 22µs | 8544.27 | 0 |

**Comparison:**
- duckdb: +1726.0% slower
- sqlite: baseline (fastest)

#### record_update_cell

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 351µs | 341µs | 397µs | 433µs | 1234.67 | 0 |
| sqlite | 25µs | 24µs | 25µs | 27µs | 14887.78 | 0 |

**Comparison:**
- duckdb: +1331.5% slower
- sqlite: baseline (fastest)

#### record_delete

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 487µs | 483µs | 527µs | 606µs | 1033.60 | 0 |
| sqlite | 67µs | 40µs | 60µs | 84µs | 8836.00 | 0 |

**Comparison:**
- duckdb: +623.7% slower
- sqlite: baseline (fastest)

### Batch Operations

#### batch_create_10

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 942µs | 920µs | 1.034ms | 1.259ms | 1044.24 | 0 |
| sqlite | 161µs | 122µs | 177µs | 440µs | 5840.35 | 0 |

**Comparison:**
- duckdb: +485.9% slower
- sqlite: baseline (fastest)

#### batch_create_100

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 6.452ms | 6.352ms | 6.841ms | 6.882ms | 152.69 | 0 |
| sqlite | 1.124ms | 899µs | 1.288ms | 1.495ms | 802.60 | 0 |

**Comparison:**
- duckdb: +473.9% slower
- sqlite: baseline (fastest)

#### batch_create_500

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 93.36ms | 92.807ms | 96.116ms | 96.116ms | 10.65 | 0 |
| sqlite | 5.616ms | 4.886ms | 8.031ms | 8.031ms | 163.93 | 0 |

**Comparison:**
- duckdb: +1562.4% slower
- sqlite: baseline (fastest)

#### batch_get_by_ids_10

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 443µs | 410µs | 704µs | 724µs | 683.56 | 0 |
| sqlite | 72µs | 60µs | 75µs | 274µs | 3687.00 | 0 |

**Comparison:**
- duckdb: +515.7% slower
- sqlite: baseline (fastest)

#### batch_get_by_ids_100

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 1.706ms | 1.67ms | 1.881ms | 1.951ms | 121.97 | 0 |
| sqlite | 626µs | 551µs | 925µs | 983µs | 531.83 | 0 |

**Comparison:**
- duckdb: +172.6% slower
- sqlite: baseline (fastest)

#### batch_delete_10

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 1.398ms | 1.379ms | 1.586ms | 1.662ms | 413.39 | 0 |
| sqlite | 142µs | 137µs | 165µs | 177µs | 3075.61 | 0 |

**Comparison:**
- duckdb: +883.3% slower
- sqlite: baseline (fastest)

#### batch_delete_100

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 5.06ms | 5.077ms | 5.3ms | 5.302ms | 86.23 | 0 |
| sqlite | 1.012ms | 819µs | 1.562ms | 1.601ms | 462.82 | 0 |

**Comparison:**
- duckdb: +399.9% slower
- sqlite: baseline (fastest)

### Queries

#### list_100_records

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 1.26ms | 1.221ms | 1.528ms | 1.651ms | 793.65 | 0 |
| sqlite | 602µs | 552µs | 879µs | 916µs | 1661.15 | 0 |

**Comparison:**
- duckdb: +109.3% slower
- sqlite: baseline (fastest)

#### list_500_records

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 3.082ms | 3.117ms | 3.419ms | 3.484ms | 324.43 | 0 |
| sqlite | 3.218ms | 2.664ms | 4.365ms | 6.136ms | 310.75 | 0 |

**Comparison:**
- duckdb: baseline (fastest)
- sqlite: +4.4% slower

#### list_with_sort

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 1.212ms | 1.194ms | 1.399ms | 1.474ms | 825.02 | 0 |
| sqlite | 609µs | 546µs | 905µs | 924µs | 1642.44 | 0 |

**Comparison:**
- duckdb: +99.1% slower
- sqlite: baseline (fastest)

#### list_with_filter

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 1.134ms | 1.11ms | 1.417ms | 1.456ms | 881.97 | 0 |
| sqlite | 564µs | 493µs | 839µs | 872µs | 1771.85 | 0 |

**Comparison:**
- duckdb: +100.9% slower
- sqlite: baseline (fastest)

### Field Operations

#### field_create

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 565µs | 548µs | 722µs | 738µs | 1721.25 | 0 |
| sqlite | 31µs | 30µs | 41µs | 45µs | 29678.73 | 0 |

**Comparison:**
- duckdb: +1705.2% slower
- sqlite: baseline (fastest)

#### field_list_by_table

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 381µs | 367µs | 456µs | 471µs | 2626.86 | 0 |
| sqlite | 31µs | 26µs | 51µs | 89µs | 32035.88 | 0 |

**Comparison:**
- duckdb: +1121.6% slower
- sqlite: baseline (fastest)

#### field_update

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 405µs | 397µs | 468µs | 518µs | 986.12 | 0 |
| sqlite | 17µs | 17µs | 19µs | 23µs | 20305.25 | 0 |

**Comparison:**
- duckdb: +2267.9% slower
- sqlite: baseline (fastest)

#### select_choice_add

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 380µs | 360µs | 456µs | 471µs | 2486.57 | 0 |
| sqlite | 26µs | 25µs | 30µs | 34µs | 35689.86 | 0 |

**Comparison:**
- duckdb: +1371.6% slower
- sqlite: baseline (fastest)

#### select_choice_list

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 253µs | 238µs | 337µs | 345µs | 2330.82 | 0 |
| sqlite | 22µs | 21µs | 24µs | 26µs | 30242.71 | 0 |

**Comparison:**
- duckdb: +1068.4% slower
- sqlite: baseline (fastest)

### Concurrent

#### concurrent_reads_10

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 493µs | 501µs | 787µs | 1.103ms | 5131.96 | 0 |
| sqlite | 155µs | 124µs | 339µs | 374µs | 23364.94 | 0 |

**Comparison:**
- duckdb: +217.9% slower
- sqlite: baseline (fastest)

#### concurrent_writes_10

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 2.379ms | 2.295ms | 3.868ms | 4.362ms | 4032.05 | 0 |
| sqlite | 854µs | 450µs | 2.793ms | 2.97ms | 10025.57 | 0 |

**Comparison:**
- duckdb: +178.7% slower
- sqlite: baseline (fastest)

#### concurrent_mixed_10

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 790µs | 448µs | 2.5ms | 2.621ms | 4405.64 | 0 |
| sqlite | 176µs | 137µs | 404µs | 597µs | 21985.27 | 0 |

**Comparison:**
- duckdb: +348.8% slower
- sqlite: baseline (fastest)

#### concurrent_reads_25

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 618µs | 578µs | 986µs | 2.145ms | 5238.96 | 0 |
| sqlite | 293µs | 207µs | 698µs | 728µs | 25147.22 | 0 |

**Comparison:**
- duckdb: +111.0% slower
- sqlite: baseline (fastest)

#### concurrent_writes_25

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 7.597ms | 5.984ms | 14.308ms | 15.055ms | 2965.52 | 0 |
| sqlite | 1.006ms | 887µs | 1.976ms | 2.298ms | 17486.85 | 0 |

**Comparison:**
- duckdb: +655.5% slower
- sqlite: baseline (fastest)

#### concurrent_mixed_25

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 1.185ms | 1.005ms | 2.902ms | 3.284ms | 4889.00 | 0 |
| sqlite | 342µs | 254µs | 808µs | 930µs | 22679.16 | 0 |

**Comparison:**
- duckdb: +246.8% slower
- sqlite: baseline (fastest)

#### concurrent_reads_50

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 491µs | 448µs | 965µs | 974µs | 5236.38 | 0 |
| sqlite | 849µs | 840µs | 1.188ms | 1.197ms | 13684.57 | 0 |

**Comparison:**
- duckdb: baseline (fastest)
- sqlite: +72.8% slower

#### concurrent_writes_50

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 7.867ms | 7.781ms | 12.958ms | 13.269ms | 3103.37 | 0 |
| sqlite | 1.556ms | 1.571ms | 2.499ms | 2.6ms | 18699.45 | 0 |

**Comparison:**
- duckdb: +405.8% slower
- sqlite: baseline (fastest)

#### concurrent_mixed_50

| Backend | Avg | P50 | P95 | P99 | Ops/s | Errors |
|---------|-----|-----|-----|-----|-------|--------|
| duckdb | 1.549ms | 1.473ms | 2.947ms | 3.419ms | 4164.89 | 0 |
| sqlite | 670µs | 611µs | 1.241ms | 1.299ms | 17094.02 | 0 |

**Comparison:**
- duckdb: +131.3% slower
- sqlite: baseline (fastest)

## Concurrency Analysis

### Concurrent Mixed

| Backend | Concurrency | Avg | P99 | Ops/s | Errors |
|---------|-------------|-----|-----|-------|--------|
| duckdb | 10 | 790µs | 2.621ms | 4405.64 | 0 |
| duckdb | 25 | 1.185ms | 3.284ms | 4889.00 | 0 |
| duckdb | 50 | 1.549ms | 3.419ms | 4164.89 | 0 |
| sqlite | 10 | 176µs | 597µs | 21985.27 | 0 |
| sqlite | 25 | 342µs | 930µs | 22679.16 | 0 |
| sqlite | 50 | 670µs | 1.299ms | 17094.02 | 0 |

### Concurrent Reads

| Backend | Concurrency | Avg | P99 | Ops/s | Errors |
|---------|-------------|-----|-----|-------|--------|
| duckdb | 10 | 493µs | 1.103ms | 5131.96 | 0 |
| duckdb | 25 | 618µs | 2.145ms | 5238.96 | 0 |
| duckdb | 50 | 491µs | 974µs | 5236.38 | 0 |
| sqlite | 10 | 155µs | 374µs | 23364.94 | 0 |
| sqlite | 25 | 293µs | 728µs | 25147.22 | 0 |
| sqlite | 50 | 849µs | 1.197ms | 13684.57 | 0 |

### Concurrent Writes

| Backend | Concurrency | Avg | P99 | Ops/s | Errors |
|---------|-------------|-----|-----|-------|--------|
| duckdb | 10 | 2.379ms | 4.362ms | 4032.05 | 0 |
| duckdb | 25 | 7.597ms | 15.055ms | 2965.52 | 0 |
| duckdb | 50 | 7.867ms | 13.269ms | 3103.37 | 0 |
| sqlite | 10 | 854µs | 2.97ms | 10025.57 | 0 |
| sqlite | 25 | 1.006ms | 2.298ms | 17486.85 | 0 |
| sqlite | 50 | 1.556ms | 2.6ms | 18699.45 | 0 |

### Scaling Observations

- **sqlite**: scales well for concurrent writes

## Recommendations

### Use Case Recommendations

#### Embedded / Single-User Applications

**Recommended: DuckDB**

- Better performance for analytical queries
- Good batch operation support
- Modern embedded database

#### Multi-User / Server Applications

**Recommended: PostgreSQL**

- Best concurrent write handling
- Mature connection pooling
- MVCC for high concurrency
- Rich querying capabilities

#### Analytical / Reporting Workloads

**Recommended: DuckDB**

- Columnar storage for analytics
- Efficient batch operations
- Good for read-heavy workloads

### Configuration Tips

1. **SQLite**: Use WAL mode (already configured) for better concurrency
2. **PostgreSQL**: Tune connection pool size based on workload
3. **DuckDB**: Consider memory settings for large datasets

---

*Report generated by StoreBench*
