# `pkg/hn` (Hacker News CLI backend)

This package powers `search hn` and manages a local Hacker News warehouse under `$HOME/data/hn/`.

It is ClickHouse-only by design:
- Base snapshots come from the ClickHouse Hacker News table (exported as Parquet chunks).
- Incremental updates also come from ClickHouse (delta Parquet chunks).
- Monthly exports are streamed directly from ClickHouse as Parquet.
- DuckDB is used locally for import/merge/indexing and optional compaction of delta back into base chunks.

## Quick Start (CLI)

```bash
search hn sync
search hn sync --every 1m
search hn compact
search hn export
```

## Local Layout

Default root: `$HOME/data/hn/`

- `raw/clickhouse/`
  - Base checkpoint-aligned Parquet partitions, e.g. `id_46500001_47000000.parquet`
- `raw/clickhouse_delta/`
  - Incremental Parquet chunks downloaded after the base snapshot exists
  - Delta downloads are auto-split at checkpoint boundaries (default span: `500,000`)
- `hn.duckdb`
  - Local queryable DuckDB database (`items` table + indexes)
- `state/download_state.json`
  - Last download metadata (ranges, remote max id, chunk span)
- `state/import_state.json`
  - Last import metadata (rows before/after, source, import mode)
- `export/hn/monthly/`
  - `items_YYYY_MM.parquet` monthly exports

## How `search hn sync` Works

### First run (no local base chunks)
1. Query ClickHouse for remote metadata (`count`, `max(id)`, `max(time)`).
2. Download base Parquet partitions into `raw/clickhouse/` in parallel.
3. Import into DuckDB (full build).

### Subsequent runs (incremental)
1. Compute local high-watermark from:
   - DuckDB max `id`
   - local base chunk max `id`
   - local delta chunk max `id`
   - download state
2. Query ClickHouse remote `max(id)`.
3. Download only delta range (`local_max+1 ... remote_max`) into `raw/clickhouse_delta/`.
4. Delta chunks are checkpoint-aligned (default `500,000`) to make later compaction simple.
5. Import into DuckDB incrementally (merge only changed/new IDs).
6. CLI reports exact `rows prev`, `rows delta`, and `rows`.

## Why checkpoint-aligned delta chunks

If a delta crosses a checkpoint boundary, the downloader splits it so files line up with base partitions.

Example with span `500,000`:
- Requested delta: `47,499,908 .. 47,500,065`
- Actual delta files:
  - `47,499,908 .. 47,500,000`
  - `47,500,001 .. 47,500,065`

This makes `search hn compact` efficient because each delta file maps cleanly into one base partition range.

## `search hn compact` (delta -> base parquet)

`compact` uses DuckDB to merge local delta parquet into base parquet partitions:

1. Read base partition(s) from `raw/clickhouse/`.
2. Read matching delta parquet from `raw/clickhouse_delta/`.
3. De-duplicate by `id` (delta wins).
4. Write rewritten base partition as Parquet (zstd, high compression by default).

This reduces the number of delta files and keeps the base parquet set current.

Notes:
- `compact` does **not** require re-importing the full DB.
- Use `--prune-delta` to delete fully compacted delta files.

## `search hn import` (local parquet -> DuckDB)

Import modes used by the CLI:
- `clickhouse`: base parquet only
- `hybrid`: base parquet + local delta parquet overlay

Default `search hn import` uses `auto` and chooses the best local source.

Import behavior:
- If DB does not exist: full build (`CREATE TABLE items AS ...`)
- If DB exists: incremental merge (delete overlapping IDs and insert delta subset)
- Creates indexes for common query patterns (`id`, `time`, `type`, etc.)

The CLI prints exact row counts before and after import.

## `search hn export` (DuckDB-first, ClickHouse fallback)

Exports monthly Parquet files with a local-first strategy:

- If local `hn.duckdb` exists, export from DuckDB first (fast, no network).
- If local DuckDB is missing/unusable, fall back to ClickHouse.
- Skips existing historical month files by default (immutable).
- For the latest/current month, performs a smart row-count check:
  - if exported parquet row count matches source row count, skip (`latest_unchanged`)
  - otherwise rewrite (when `--refresh-latest` is enabled, default true)
- Supports `--from-month` / `--to-month` filtering.
- CLI prints source, output dir, per-month progress, and skip reasons.

Example:

```bash
search hn export --from-month 2026-01 --to-month 2026-02
```

## Performance Notes

- ClickHouse downloads use `errgroup` with a concurrency limit (`--parallel`).
- Detailed progress shows per-chunk bytes/s and aggregate throughput.
- Incremental syncs should be much faster than base downloads because they fetch only delta.
- `compact` is optional and can be run periodically (e.g. daily) instead of every sync.

## Troubleshooting

- DNS/network issues to ClickHouse:
  - Retry; downloads are resumable/chunked.
  - Existing chunk files are skipped unless `--force` or tail refresh applies.
- Import uses local disk for DuckDB temp work:
  - Ensure free space before full rebuilds.
  - Prefer incremental syncs for routine updates.
- If schema behavior changes across versions:
  - Run `search hn import --rebuild`.

## Developer Notes (package internals)

Key areas:
- `clickhouse.go`
  - Remote metadata queries, chunk export, parallel download, progress reporting
- `live.go`
  - Local high-watermark detection and delta range suggestion
- `import.go`
  - DuckDB full/incremental import paths
- `compact.go`
  - DuckDB-based delta-to-base Parquet merge
- `export.go`
  - Remote monthly Parquet export from ClickHouse
- `state.go`
  - Download/import run state persistence

The CLI is intentionally opinionated:
- ClickHouse is the only remote source
- sane defaults (`500,000` checkpoint span, parallel downloads, latest-month refresh)
- explicit commands for advanced maintenance (`compact`, `export`)
