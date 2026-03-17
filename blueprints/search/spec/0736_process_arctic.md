# 0736: Arctic Processing Pipeline 10x Optimization

## Problem

Processing 2011-01 comments (6.6M rows, 592 MB .zst, 14 shards) takes **12m14s**.
Processing 2011-01 submissions (838K rows, 117 MB .zst, 2 shards) takes **12m30s**
(almost entirely semaphore wait behind comments).

**Targets**: comments ≤ 3 min, submissions ≤ 30s own processing time. 10x overall.

## Root Cause Analysis

Server2: 6-core AMD EPYC, 11 GB RAM, SSD.
Current: DuckDB engine, 3 convert workers, 500K lines/chunk.

### 1. DuckDB thread oversubscription (3x penalty)

`convertChunkToShard()` calls `sql.Open("duckdb", "")` but never sets `threads`.
DuckDB defaults to `hardware_concurrency()` = 6 threads per instance.
With 3 concurrent instances: 18 threads on 6 cores = **3x oversubscription**.
Each thread gets ~33% CPU → per-chunk wall time inflated 3x.

### 2. Triple-pass data processing (~2x penalty)

Current flow per chunk:
```
Pass 1: CREATE TABLE data AS SELECT ... FROM read_json(chunk)  → full materialization
Pass 2: SELECT COUNT(*) FROM data                             → full table scan
Pass 3: COPY data TO 'shard.parquet'                          → full table read
```

Could be **one streaming pass**:
```
COPY (SELECT ... FROM read_json(chunk)) TO 'shard.parquet'
```
Then read row count from parquet file metadata (4 bytes from footer).

### 3. Semaphore starvation cascade

The zstd scan loop dispatches chunks via `g.Go()` which blocks when all workers
are busy. The semaphore is released only after the scan loop exits — which
requires all 14 chunks to be dispatched — which waits for workers to free up.
Net: semaphore held for ~12 minutes, blocking submissions the entire time.

**Combined penalty: ~6-9x.** Fixing passes 1+2 gives ~6x. With more workers: 10x+.

## Phase 1: DuckDB Optimizations

### Fix 1: Set threads per DuckDB instance

```go
// In convertChunkToShard(), after SET memory_limit:
threadsPerWorker := max(1, runtime.NumCPU() / workers)
db.ExecContext(ctx, fmt.Sprintf("SET threads = %d", threadsPerWorker))
```

On server2: `6 / 3 = 2` threads per instance. 6 total threads = 6 cores. No oversubscription.

### Fix 2: Streaming COPY (eliminate triple-pass)

Replace:
```sql
CREATE TABLE data AS SELECT ... FROM read_json(...)
SELECT COUNT(*) FROM data
COPY data TO '...' (FORMAT PARQUET, ...)
```

With:
```sql
COPY (SELECT ... FROM read_json(...)) TO '...' (FORMAT PARQUET, ...)
```

Then read row count from the written parquet file:
```sql
SELECT COUNT(*) FROM read_parquet('shard.parquet')
```
(Or even cheaper: read parquet footer metadata directly in Go.)

### Fix 3: Increase MaxConvertWorkers

With `SET threads=1` per DuckDB, we can safely run 6 workers on 6 cores:
- Memory: 6 × 512 MB = 3 GB DuckDB + 4 GB OS = 7 GB total (4 GB free on 11 GB)
- CPU: 6 × 1 thread = 6 = exact match

Update `budget.go`: for DuckDB with explicit thread pinning, use `CPUCores` (not
`CPUCores/2`) as the convert worker cap.

### Fix 4: Reduce maximum_object_size

From 10 MB to 2 MB. Reddit JSON objects rarely exceed 100 KB; 10 MB wastes
per-thread buffer space. No functional impact.

### Expected DuckDB result

Per-chunk (500K rows, ~250 MB text):
- Single-threaded DuckDB streaming: JSON parse (~2.5s) + ZSTD L3 parquet (~1s)
  + disk I/O (~0.5s) + overhead (~0.5s) ≈ **5s per chunk**
- Conservative estimate with real-world overhead: **10-20s per chunk**

Total for 2011-01 comments (14 chunks):
- 6 workers: ceil(14/6) = 3 batches × 15s = **45-60s** (vs current 815s = 13-18x)
- 3 workers: ceil(14/3) = 5 batches × 15s = **75s** (vs current 815s = 10x)

## Phase 2: Go Engine (after DuckDB baseline established)

If DuckDB optimizations alone don't hit targets, switch to Go in-memory engine:

- **Eliminates chunk disk I/O entirely** (no 7 GB write+read per comments file)
- No DuckDB startup/teardown, no CGO bridge overhead
- Pure Go JSON parse → parquet-go → disk

Caution: **OOM risk**. Go in-memory chunks hold 500K lines in `[][]byte`.
For 250 MB text chunks, 3 workers = 750 MB just in line buffers.
Must profile with `pprof` before deploying.

### Go engine changes
- Lower ZSTD compression from `SpeedBestCompression` (level ~11) to level 3
  (match DuckDB output size, ~3x faster compression)
- Profile with `MIZU_ARCTIC_MEMPROF` and `MIZU_ARCTIC_CPUPROF` env vars

## Benchmark CLI Subcommand

New `search arctic bench` command:

```
search arctic bench --zst /path/to/RC_2011-01.zst --type comments \
  [--engine duckdb|go] [--workers 1,3,6] [--chunk-lines 500000]
```

Runs ProcessZst on a single .zst file with each combination.
Outputs a timing/comparison table:

```
Engine   Workers  Chunks  Time     Rows      Size      Rows/s
duckdb   3        14      12m14s   6603329   570.4 MB  9,001
duckdb   6        14      45.2s    6603329   570.4 MB  146,098
go-mem   3        14      32.1s    6603329   582.1 MB  205,710
```

Validates: row counts match across engines, file sizes within 10%.

## Verification

1. Copy 2011-01 .zst files to `/root/data/arctic/bench/` on server2
2. Run bench with current code (baseline)
3. Apply DuckDB fixes, rebuild, re-bench
4. Compare row counts (must match exactly), parquet sizes (within 10%)
5. Spot-check: `duckdb "SELECT * FROM read_parquet('shard.parquet') LIMIT 5"`
6. If DuckDB hits target, deploy to pipeline and restart arctic_run7
7. If not, proceed to Phase 2 (Go engine)

## Files to modify

| File | Change |
|------|--------|
| `pkg/arctic/process.go` | `convertChunkToShard`: add SET threads, streaming COPY |
| `pkg/arctic/process.go` | Pass `workers` count to convertChunkToShard |
| `pkg/arctic/budget.go` | Increase MaxConvertWorkers cap for thread-pinned DuckDB |
| `cli/arctic.go` | Add `newArcticBench()` subcommand |
| `cli/arctic_bench.go` | New file: bench subcommand implementation |
