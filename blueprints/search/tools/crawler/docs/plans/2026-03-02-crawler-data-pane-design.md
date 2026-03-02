# Crawler Data Pane Design

**Date**: 2026-03-02
**Status**: Approved

## Goal

Add a "Data" pane to both the TUI and web GUI that shows live disk usage from the crawler's output directories — segment files, DuckDB shards, failure store, and body CAS store — without impacting crawl performance.

## Approach

Option A: Add `DiskStats` atomic fields directly to the existing `Stats` struct and spawn a dedicated `disk_sampler` tokio task in `common.rs`. The 10s sampling interval keeps filesystem I/O negligible. Post-drain, a one-time DuckDB `COUNT(*)` adds row counts safely (writer is closed when `stats.done` is set).

## New Stats Fields

Ten new `AtomicU64` fields added to `crawler-lib/src/stats.rs`:

| Field | Description | Interval |
|---|---|---|
| `disk_seg_files` | Count of live `seg_*.bin` files | 10s |
| `disk_seg_mb` | Total MB of seg files | 10s |
| `disk_duckdb_mb` | Total MB of `results_*.duckdb` shards | 10s |
| `disk_results_rows` | Row count in result shards | Once, post-drain |
| `disk_failures_mb` | Total MB of `failures/` dir | 10s |
| `disk_failed_rows` | Row count in `failed.duckdb` | Once, post-drain |
| `disk_bodies_count` | File count in `bodies/` dir | 10s |
| `disk_bodies_mb` | Total MB of `bodies/` dir | 10s |
| `disk_total_mb` | Sum of all above | 10s |
| `disk_last_updated` | Unix secs of last scan | 10s |

## DiskPaths Struct

```rust
pub struct DiskPaths {
    pub seg_dir: Option<PathBuf>,       // results/ binary segment dir
    pub duckdb_dir: Option<PathBuf>,    // results_*.duckdb shard dir (same or sub)
    pub failures_dir: Option<PathBuf>,  // failures/ dir
    pub failed_db: Option<PathBuf>,     // failed.duckdb path
    pub bodies_dir: Option<PathBuf>,    // CAS body store dir
}
```

Constructed in `common.rs` from `CrawlJobParams` fields, passed into `spawn_disk_sampler`.

## Sampler Task (`common.rs`)

Spawned as a tokio task alongside the engine:

```
loop every 10s:
  scan seg_dir      → disk_seg_files, disk_seg_mb
  scan duckdb_dir   → disk_duckdb_mb
  scan failures_dir → disk_failures_mb
  scan bodies_dir   → disk_bodies_count, disk_bodies_mb (readdir only, no file reads)
  update disk_total_mb, disk_last_updated
  if stats.done:
    open duckdb_dir shards → COUNT(*) → disk_results_rows
    open failed_db         → COUNT(*) → disk_failed_rows
    break
```

Filesystem scanning uses `std::fs::read_dir` + metadata (no file contents read). Bodies dir: metadata only, accumulate file count + size. All writes to Stats atomics via `Relaxed` ordering.

## TUI Rendering

New "Data" line in the right panel, below the existing "Sys" line:

```
 Data  12 segs 847MB  DuckDB —  Bodies 1.2M/38GB  Fail 12MB
```

After drain (rows known):
```
 Data  — segs  DuckDB 2.1M rows/3.4GB  Bodies 1.2M/38GB  Fail 41K/12MB
```

Fields shown: seg count (during crawl), DuckDB size/rows, bodies count/size, failures size/rows. Hidden when all zero (pre-crawl).

## GUI Rendering

New **"Data"** card in the 3-column sub-grid (alongside Errors, HTTP Status, System), or as a fourth card in a 2×2 layout:

```
┌─────────────────────────────┐
│ Data                        │
│ Segments    12 files 847 MB │
│ Results DB  — (draining…)   │
│ Failures    12 MB           │
│ Bodies      1.2M  38.4 GB   │
│ Total disk  39.3 GB         │
│ Updated 8s ago              │
└─────────────────────────────┘
```

After drain:
```
│ Results DB  2.1M rows  3.4 GB │
│ Failures    41K rows   12 MB  │
```

SSE payload extended with all 10 new fields. Hidden/grayed when zero.

## Files Changed

1. `crawler-lib/src/stats.rs` — add 10 `AtomicU64` fields + init in `Stats::new()`
2. `crawler-lib/src/lib.rs` — export `DiskPaths` (new struct, defined in `stats.rs` or new `disk.rs`)
3. `crawler-cli/src/common.rs` — construct `DiskPaths`, spawn `disk_sampler` task
4. `crawler-cli/src/tui.rs` — render Data line in right panel
5. `crawler-cli/src/gui.rs` — add 10 fields to `StatsPayload` + `snapshot_stats`
6. `crawler-cli/assets/dashboard.html` — add Data card, wire SSE fields
