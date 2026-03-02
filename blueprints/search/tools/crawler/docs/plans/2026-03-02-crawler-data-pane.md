# Crawler Data Pane Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a live "Data" pane to both TUI and web GUI showing disk usage (seg files, DuckDB shards, bodies, failures) sampled every 10s in a background task, plus one-time row counts after drain completes.

**Architecture:** Ten new `AtomicU64` fields in `Stats` are updated by a `disk_sampler` tokio task (spawn_blocking for FS I/O) spawned in `common.rs`. After `result_writer.close()` returns (drain done), a blocking `finalize_disk_stats` call queries DuckDB for row counts. TUI gets a new "Data" line; GUI gets a new "Data" card in the dashboard.

**Tech Stack:** Rust, tokio (spawn_blocking), duckdb-rs (already in crawler-lib), ratatui (TUI), axum SSE + vanilla JS (GUI).

---

### Task 1: Add disk stat fields to Stats

**Files:**
- Modify: `crawler-lib/src/stats.rs`

**Step 1: Add 10 AtomicU64 fields to the `Stats` struct after the `open_fds` field (line ~70)**

```rust
    // --- Disk stats (updated by disk_sampler every 10s) ---
    /// Number of live seg_*.bin files
    pub disk_seg_files: AtomicU64,
    /// Total MB of seg_*.bin files
    pub disk_seg_mb: AtomicU64,
    /// Total MB of results_*.duckdb shards
    pub disk_duckdb_mb: AtomicU64,
    /// Row count in result DuckDB shards (set once, post-drain)
    pub disk_results_rows: AtomicU64,
    /// Total MB of failures/ dir
    pub disk_failures_mb: AtomicU64,
    /// Row count in failed.duckdb (set once, post-drain)
    pub disk_failed_rows: AtomicU64,
    /// File count in bodies/ CAS dir
    pub disk_bodies_count: AtomicU64,
    /// Total MB of bodies/ dir
    pub disk_bodies_mb: AtomicU64,
    /// Grand total disk MB (seg + duckdb + failures + bodies)
    pub disk_total_mb: AtomicU64,
    /// Unix seconds of last disk scan
    pub disk_last_updated: AtomicU64,
```

**Step 2: Initialize all 10 fields in `Stats::new()` after the `open_fds` init**

```rust
            disk_seg_files: AtomicU64::new(0),
            disk_seg_mb: AtomicU64::new(0),
            disk_duckdb_mb: AtomicU64::new(0),
            disk_results_rows: AtomicU64::new(0),
            disk_failures_mb: AtomicU64::new(0),
            disk_failed_rows: AtomicU64::new(0),
            disk_bodies_count: AtomicU64::new(0),
            disk_bodies_mb: AtomicU64::new(0),
            disk_total_mb: AtomicU64::new(0),
            disk_last_updated: AtomicU64::new(0),
```

**Step 3: Run cargo check from the crawler workspace root**

```bash
cd blueprints/search/tools/crawler && source ~/.cargo/env && cargo check
```

Expected: `Finished` with at most the existing dead_code warning.

**Step 4: Commit**

```bash
git add crawler-lib/src/stats.rs
git commit -m "feat(crawler): add disk stat fields to Stats"
```

---

### Task 2: Add row-count helpers to duckdb_writer

**Files:**
- Modify: `crawler-lib/src/writer/duckdb_writer.rs`

We need two functions that open DuckDB files (read-only) and return row counts. These are called only after drain (writer closed), so no lock contention.

**Step 1: Add `count_result_rows` function at the end of the file**

```rust
/// Count total rows across all results_NNN.duckdb shards in `duckdb_dir`.
/// Only call after the result writer has been closed (post-drain).
pub fn count_result_rows(duckdb_dir: &std::path::Path) -> anyhow::Result<u64> {
    if !duckdb_dir.exists() {
        return Ok(0);
    }
    let mut total: u64 = 0;
    let mut paths: Vec<std::path::PathBuf> = std::fs::read_dir(duckdb_dir)?
        .filter_map(|e| e.ok())
        .map(|e| e.path())
        .filter(|p| {
            p.extension().map_or(false, |ext| ext == "duckdb")
                && p.file_name()
                    .and_then(|n| n.to_str())
                    .map_or(false, |n| n.starts_with("results_"))
        })
        .collect();
    paths.sort();
    for path in &paths {
        let conn = duckdb::Connection::open_with_flags(
            path,
            duckdb::Config::default().access_mode(duckdb::AccessMode::ReadOnly)?,
        )?;
        let n: u64 = conn.query_row("SELECT COUNT(*) FROM results", [], |r| r.get(0))?;
        total += n;
    }
    Ok(total)
}

/// Count rows in `failed.duckdb`.
/// Only call after the failure writer has been closed (post-drain).
pub fn count_failed_rows(failed_db: &std::path::Path) -> anyhow::Result<u64> {
    if !failed_db.exists() {
        return Ok(0);
    }
    let conn = duckdb::Connection::open_with_flags(
        failed_db,
        duckdb::Config::default().access_mode(duckdb::AccessMode::ReadOnly)?,
    )?;
    let n: u64 = conn.query_row("SELECT COUNT(*) FROM failed_urls", [], |r| r.get(0))?;
    Ok(n)
}
```

**Step 2: Run cargo check**

```bash
cargo check
```

Expected: clean compile.

**Step 3: Commit**

```bash
git add crawler-lib/src/writer/duckdb_writer.rs
git commit -m "feat(crawler): add count_result_rows / count_failed_rows helpers"
```

---

### Task 3: Add DiskPaths struct and disk_sampler in common.rs

**Files:**
- Modify: `crawler-cli/src/common.rs`

**Step 1: Add `DiskPaths` struct and imports at the top of `common.rs` (after existing use lines)**

```rust
use crawler_lib::writer::duckdb_writer::{count_failed_rows, count_result_rows};

/// Paths to monitor for live disk stats.
#[derive(Clone)]
pub struct DiskPaths {
    /// results/ subdir — contains seg_*.bin files (binary writer)
    pub seg_dir: Option<PathBuf>,
    /// output_dir — contains results_*.duckdb shards
    pub duckdb_dir: Option<PathBuf>,
    /// failures/ subdir
    pub failures_dir: Option<PathBuf>,
    /// failed.duckdb path
    pub failed_db: Option<PathBuf>,
    /// CAS body store dir
    pub bodies_dir: Option<PathBuf>,
}
```

**Step 2: Add helper functions for filesystem scanning after the `DiskPaths` struct**

```rust
/// Recursively sum file sizes in a directory (bytes). Returns (file_count, bytes).
/// Uses spawn_blocking — call from a tokio context.
fn scan_dir(dir: &Path) -> (u64, u64) {
    if !dir.exists() {
        return (0, 0);
    }
    let mut count = 0u64;
    let mut bytes = 0u64;
    // Use a stack to avoid recursion overhead for flat CAS dirs.
    let mut stack = vec![dir.to_path_buf()];
    while let Some(d) = stack.pop() {
        let rd = match std::fs::read_dir(&d) {
            Ok(r) => r,
            Err(_) => continue,
        };
        for entry in rd.flatten() {
            let meta = match entry.metadata() {
                Ok(m) => m,
                Err(_) => continue,
            };
            if meta.is_file() {
                count += 1;
                bytes += meta.len();
            } else if meta.is_dir() {
                stack.push(entry.path());
            }
        }
    }
    (count, bytes)
}

/// Count seg_*.bin files and their total size in `dir`.
fn scan_seg_dir(dir: &Path) -> (u64, u64) {
    if !dir.exists() {
        return (0, 0);
    }
    let mut count = 0u64;
    let mut bytes = 0u64;
    if let Ok(rd) = std::fs::read_dir(dir) {
        for entry in rd.flatten() {
            let p = entry.path();
            if p.extension().map_or(false, |e| e == "bin")
                && p.file_name()
                    .and_then(|n| n.to_str())
                    .map_or(false, |n| n.starts_with("seg_"))
            {
                if let Ok(m) = entry.metadata() {
                    count += 1;
                    bytes += m.len();
                }
            }
        }
    }
    (count, bytes)
}

/// Sum sizes of results_*.duckdb files in `dir`.
fn scan_duckdb_dir(dir: &Path) -> u64 {
    if !dir.exists() {
        return 0;
    }
    let mut bytes = 0u64;
    if let Ok(rd) = std::fs::read_dir(dir) {
        for entry in rd.flatten() {
            let p = entry.path();
            if p.extension().map_or(false, |e| e == "duckdb")
                && p.file_name()
                    .and_then(|n| n.to_str())
                    .map_or(false, |n| n.starts_with("results_"))
            {
                if let Ok(m) = entry.metadata() {
                    bytes += m.len();
                }
            }
        }
    }
    bytes
}

fn bytes_to_mb(b: u64) -> u64 {
    b / (1024 * 1024)
}

/// Spawn a background tokio task that updates disk stats in `stats` every 10s.
/// Stops after `stats.done` is observed (row counts are filled by `finalize_disk_stats`).
pub fn spawn_disk_sampler(stats: Arc<Stats>, paths: DiskPaths) {
    tokio::spawn(async move {
        let mut interval = tokio::time::interval(Duration::from_secs(10));
        // first tick fires immediately — give engine 2s to start writing
        tokio::time::sleep(Duration::from_secs(2)).await;
        loop {
            interval.tick().await;

            let s = stats.clone();
            let p = paths.clone();
            tokio::task::spawn_blocking(move || {
                // seg files
                if let Some(ref d) = p.seg_dir {
                    let (cnt, bytes) = scan_seg_dir(d);
                    s.disk_seg_files.store(cnt, Ordering::Relaxed);
                    s.disk_seg_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
                }

                // duckdb shards
                if let Some(ref d) = p.duckdb_dir {
                    let bytes = scan_duckdb_dir(d);
                    s.disk_duckdb_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
                }

                // failures dir
                if let Some(ref d) = p.failures_dir {
                    let (_, bytes) = scan_dir(d);
                    s.disk_failures_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
                }

                // bodies dir (potentially large — scan_dir handles it iteratively)
                if let Some(ref d) = p.bodies_dir {
                    let (cnt, bytes) = scan_dir(d);
                    s.disk_bodies_count.store(cnt, Ordering::Relaxed);
                    s.disk_bodies_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
                }

                // total
                let total = s.disk_seg_mb.load(Ordering::Relaxed)
                    + s.disk_duckdb_mb.load(Ordering::Relaxed)
                    + s.disk_failures_mb.load(Ordering::Relaxed)
                    + s.disk_bodies_mb.load(Ordering::Relaxed);
                s.disk_total_mb.store(total, Ordering::Relaxed);

                let now = std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .unwrap_or_default()
                    .as_secs();
                s.disk_last_updated.store(now, Ordering::Relaxed);
            })
            .await
            .ok();

            if stats.done.load(Ordering::Relaxed) {
                break;
            }
        }
    });
}

/// Called from `run_crawl_job` AFTER `result_writer.close()` returns.
/// Does a final filesystem scan + DuckDB row counts (safe — writers closed).
pub fn finalize_disk_stats(stats: &Stats, paths: &DiskPaths) {
    // final filesystem scan
    if let Some(ref d) = paths.seg_dir {
        let (cnt, bytes) = scan_seg_dir(d);
        stats.disk_seg_files.store(cnt, Ordering::Relaxed);
        stats.disk_seg_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
    }
    if let Some(ref d) = paths.duckdb_dir {
        let bytes = scan_duckdb_dir(d);
        stats.disk_duckdb_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
    }
    if let Some(ref d) = paths.failures_dir {
        let (_, bytes) = scan_dir(d);
        stats.disk_failures_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
    }
    if let Some(ref d) = paths.bodies_dir {
        let (cnt, bytes) = scan_dir(d);
        stats.disk_bodies_count.store(cnt, Ordering::Relaxed);
        stats.disk_bodies_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
    }

    // row counts (writers closed, safe to open DuckDB)
    if let Some(ref d) = paths.duckdb_dir {
        if let Ok(rows) = count_result_rows(d) {
            stats.disk_results_rows.store(rows, Ordering::Relaxed);
        }
    }
    if let Some(ref p) = paths.failed_db {
        if let Ok(rows) = count_failed_rows(p) {
            stats.disk_failed_rows.store(rows, Ordering::Relaxed);
        }
    }

    let total = stats.disk_seg_mb.load(Ordering::Relaxed)
        + stats.disk_duckdb_mb.load(Ordering::Relaxed)
        + stats.disk_failures_mb.load(Ordering::Relaxed)
        + stats.disk_bodies_mb.load(Ordering::Relaxed);
    stats.disk_total_mb.store(total, Ordering::Relaxed);
}
```

**Step 3: Wire DiskPaths construction and sampler into `run_crawl_job`**

In `run_crawl_job`, after the body store setup block (after `println!("Body store: ...")`), add:

```rust
    // Build disk paths for live disk stats
    let disk_paths = DiskPaths {
        seg_dir: match writer_type {
            WriterType::Binary => Some(output_dir.join("results")),
            _ => None,
        },
        duckdb_dir: match writer_type {
            WriterType::Binary | WriterType::DuckDB => Some(output_dir.clone()),
            _ => None,
        },
        failures_dir: match writer_type {
            WriterType::Binary | WriterType::Parquet => Some(output_dir.join("failures")),
            _ => None,
        },
        failed_db: Some(failed_db_path.clone()),
        bodies_dir: params.body_store_dir.as_deref().map(expand_home),
    };
    spawn_disk_sampler(live_stats.clone(), disk_paths.clone());
```

Then, after the line `result_writer.close()?;`, add:

```rust
    finalize_disk_stats(&live_stats, &disk_paths);
```

**Step 4: Add `use std::sync::atomic::Ordering;` if not already present in common.rs**

Check the imports — if `Ordering` isn't already imported, add:
```rust
use std::sync::atomic::Ordering;
```

**Step 5: Run cargo check**

```bash
cargo check
```

Expected: clean.

**Step 6: Commit**

```bash
git add crawler-cli/src/common.rs
git commit -m "feat(crawler): spawn disk_sampler task, finalize disk stats post-drain"
```

---

### Task 4: TUI — render Data line

**Files:**
- Modify: `crawler-cli/src/tui.rs`

**Step 1: In the render loop, load the new disk stat fields**

In `render_main` (around line 248 where other stats are loaded), add after the `mem_total` line:

```rust
    // Disk stats
    let disk_segs     = stats.disk_seg_files.load(Ordering::Relaxed);
    let disk_seg_mb   = stats.disk_seg_mb.load(Ordering::Relaxed);
    let disk_db_mb    = stats.disk_duckdb_mb.load(Ordering::Relaxed);
    let disk_db_rows  = stats.disk_results_rows.load(Ordering::Relaxed);
    let disk_fail_mb  = stats.disk_failures_mb.load(Ordering::Relaxed);
    let disk_fail_rows = stats.disk_failed_rows.load(Ordering::Relaxed);
    let disk_bodies   = stats.disk_bodies_count.load(Ordering::Relaxed);
    let disk_bod_mb   = stats.disk_bodies_mb.load(Ordering::Relaxed);
    let disk_total_mb = stats.disk_total_mb.load(Ordering::Relaxed);
```

**Step 2: Pass these values down to `render_throughput`**

Update `render_main` call to `render_throughput` to add the disk params, and update the `render_throughput` signature accordingly:

Add to the `render_throughput` parameter list (after `open_fds`):
```rust
    disk_segs: u64, disk_seg_mb: u64,
    disk_db_mb: u64, disk_db_rows: u64,
    disk_fail_mb: u64, disk_fail_rows: u64,
    disk_bodies: u64, disk_bod_mb: u64,
    disk_total_mb: u64,
```

Pass them through in the `render_main` → `render_throughput` call chain (there are two levels: `render_main` → `render_throughput` helper → `render_right_panel`).

**Step 3: Add Data line to `render_right_panel` after the `sys_line` push (line ~524)**

Replace the current `metrics.push(sys_line);` with:

```rust
    metrics.push(sys_line);

    // Data line
    let any_disk = disk_seg_mb + disk_db_mb + disk_fail_mb + disk_bod_mb > 0;
    if any_disk {
        let mut parts: Vec<Span> = vec![Span::styled(" Data ", dim)];

        if disk_segs > 0 {
            parts.push(Span::styled(
                format!("{}segs {}MB", disk_segs, disk_seg_mb),
                Style::default().fg(Color::Yellow),
            ));
            parts.push(Span::styled("  ", dim));
        }

        if disk_db_rows > 0 {
            parts.push(Span::styled(
                format!("db {}k/{MB}MB", disk_db_rows / 1000, disk_db_mb),
                Style::default().fg(Color::Cyan),
            ));
        } else if disk_db_mb > 0 {
            parts.push(Span::styled(
                format!("db {}MB", disk_db_mb),
                Style::default().fg(Color::Cyan),
            ));
        }
        if disk_db_mb > 0 { parts.push(Span::styled("  ", dim)); }

        if disk_bod_mb > 0 {
            parts.push(Span::styled(
                format!("bodies {}k/{}GB",
                    disk_bodies / 1000,
                    disk_bod_mb / 1024),
                Style::default().fg(Color::Green),
            ));
            parts.push(Span::styled("  ", dim));
        }

        if disk_fail_mb > 0 {
            if disk_fail_rows > 0 {
                parts.push(Span::styled(
                    format!("fail {}k/{}MB", disk_fail_rows / 1000, disk_fail_mb),
                    Style::default().fg(Color::Red),
                ));
            } else {
                parts.push(Span::styled(
                    format!("fail {}MB", disk_fail_mb),
                    Style::default().fg(Color::Red),
                ));
            }
        }

        if disk_total_mb > 1024 {
            parts.push(Span::styled("  ", dim));
            parts.push(Span::styled(
                format!("total {}GB", disk_total_mb / 1024),
                Style::default().fg(Color::White),
            ));
        }

        metrics.push(Line::from(parts));
    }
```

Note: fix the format string for `disk_db_rows` — use:
```rust
format!("db {}k/{}MB", disk_db_rows / 1000, disk_db_mb)
```

**Step 4: Run cargo check**

```bash
cargo check
```

Fix any compiler errors (type mismatches, missing params).

**Step 5: Commit**

```bash
git add crawler-cli/src/tui.rs
git commit -m "feat(crawler-tui): add Data line with disk stats"
```

---

### Task 5: GUI — extend StatsPayload and snapshot_stats

**Files:**
- Modify: `crawler-cli/src/gui.rs`

**Step 1: Add disk fields to `StatsPayload` struct (after the `done: bool` field)**

```rust
    // Disk stats
    disk_seg_files: u64,
    disk_seg_mb: u64,
    disk_duckdb_mb: u64,
    disk_results_rows: u64,
    disk_failures_mb: u64,
    disk_failed_rows: u64,
    disk_bodies_count: u64,
    disk_bodies_mb: u64,
    disk_total_mb: u64,
    disk_last_updated: u64,
```

**Step 2: Populate them in `snapshot_stats` (after the `done:` line)**

```rust
        disk_seg_files:    stats.disk_seg_files.load(Ordering::Relaxed),
        disk_seg_mb:       stats.disk_seg_mb.load(Ordering::Relaxed),
        disk_duckdb_mb:    stats.disk_duckdb_mb.load(Ordering::Relaxed),
        disk_results_rows: stats.disk_results_rows.load(Ordering::Relaxed),
        disk_failures_mb:  stats.disk_failures_mb.load(Ordering::Relaxed),
        disk_failed_rows:  stats.disk_failed_rows.load(Ordering::Relaxed),
        disk_bodies_count: stats.disk_bodies_count.load(Ordering::Relaxed),
        disk_bodies_mb:    stats.disk_bodies_mb.load(Ordering::Relaxed),
        disk_total_mb:     stats.disk_total_mb.load(Ordering::Relaxed),
        disk_last_updated: stats.disk_last_updated.load(Ordering::Relaxed),
```

**Step 3: Run cargo check**

```bash
cargo check
```

**Step 4: Commit**

```bash
git add crawler-cli/src/gui.rs
git commit -m "feat(crawler-gui): add disk stats to SSE payload"
```

---

### Task 6: dashboard.html — add Data card

**Files:**
- Modify: `crawler-cli/assets/dashboard.html`

**Step 1: Add Data card CSS**

In the `<style>` block, after the `.card` rules, add (or find a good insertion point near the System card styles):

```css
.data-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 2px 0;
  font-size: 0.82rem;
}
.data-label {
  color: var(--muted);
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  min-width: 90px;
}
.data-value {
  font-variant-numeric: tabular-nums;
  font-weight: 500;
}
.data-sub {
  color: var(--muted);
  font-size: 0.75rem;
  margin-left: 6px;
}
#dataUpdated {
  font-size: 0.72rem;
  color: var(--muted);
  text-align: right;
  margin-top: 6px;
}
```

**Step 2: Add the Data card HTML**

Find the closing `</div>` of the 3-column sub-grid section (the one containing the Errors, HTTP Status, and System cards). After it, insert a new full-width Data card:

```html
<!-- Data card -->
<div class="card" id="dataCard" style="display:none">
  <div class="card-title">Data</div>

  <div class="data-row" id="dataSegsRow" style="display:none">
    <span class="data-label">Segments</span>
    <span class="data-value">
      <span id="dataSegCount">—</span> files
      <span class="data-sub" id="dataSegMb"></span>
    </span>
  </div>

  <div class="data-row" id="dataDbRow" style="display:none">
    <span class="data-label">Results DB</span>
    <span class="data-value">
      <span id="dataDbRows">—</span>
      <span class="data-sub" id="dataDbMb"></span>
    </span>
  </div>

  <div class="data-row" id="dataBodiesRow" style="display:none">
    <span class="data-label">Bodies</span>
    <span class="data-value">
      <span id="dataBodiesCount">—</span>
      <span class="data-sub" id="dataBodiesMb"></span>
    </span>
  </div>

  <div class="data-row" id="dataFailRow" style="display:none">
    <span class="data-label">Failures</span>
    <span class="data-value">
      <span id="dataFailRows">—</span>
      <span class="data-sub" id="dataFailMb"></span>
    </span>
  </div>

  <div class="data-row" id="dataTotalRow" style="display:none">
    <span class="data-label">Total disk</span>
    <span class="data-value" id="dataTotalMb">—</span>
  </div>

  <div id="dataUpdated"></div>
</div>
```

**Step 3: Add JavaScript to update the Data card**

In the SSE `onmessage` handler (the big `function updateStats(s)` block), add after the System card updates:

```javascript
// Data card
const anyDisk = s.disk_seg_mb + s.disk_duckdb_mb + s.disk_failures_mb + s.disk_bodies_mb > 0;
$('dataCard').style.display = anyDisk ? '' : 'none';

if (s.disk_seg_files > 0) {
  $('dataSegsRow').style.display = '';
  $('dataSegCount').textContent = fmtNum(s.disk_seg_files);
  $('dataSegMb').textContent = s.disk_seg_mb + ' MB';
} else {
  $('dataSegsRow').style.display = 'none';
}

if (s.disk_duckdb_mb > 0) {
  $('dataDbRow').style.display = '';
  if (s.disk_results_rows > 0) {
    $('dataDbRows').textContent = fmtNum(s.disk_results_rows) + ' rows';
  } else {
    $('dataDbRows').textContent = s.done ? 'draining…' : '—';
  }
  $('dataDbMb').textContent = fmtBytes(s.disk_duckdb_mb * 1024 * 1024);
} else {
  $('dataDbRow').style.display = 'none';
}

if (s.disk_bodies_mb > 0 || s.disk_bodies_count > 0) {
  $('dataBodiesRow').style.display = '';
  $('dataBodiesCount').textContent = fmtNum(s.disk_bodies_count) + ' files';
  $('dataBodiesMb').textContent = fmtBytes(s.disk_bodies_mb * 1024 * 1024);
} else {
  $('dataBodiesRow').style.display = 'none';
}

if (s.disk_failures_mb > 0) {
  $('dataFailRow').style.display = '';
  if (s.disk_failed_rows > 0) {
    $('dataFailRows').textContent = fmtNum(s.disk_failed_rows) + ' rows';
  } else {
    $('dataFailRows').textContent = '—';
  }
  $('dataFailMb').textContent = fmtBytes(s.disk_failures_mb * 1024 * 1024);
} else {
  $('dataFailRow').style.display = 'none';
}

if (s.disk_total_mb > 0) {
  $('dataTotalRow').style.display = '';
  $('dataTotalMb').textContent = fmtBytes(s.disk_total_mb * 1024 * 1024);
} else {
  $('dataTotalRow').style.display = 'none';
}

if (s.disk_last_updated > 0) {
  const ageSecs = Math.floor(Date.now() / 1000) - s.disk_last_updated;
  $('dataUpdated').textContent = 'Updated ' + ageSecs + 's ago';
}
```

Note: `fmtBytes` and `fmtNum` should already exist in the dashboard. If not, verify the exact helper function names used in the existing dashboard JS and use those instead.

**Step 4: Run cargo check (html changes don't need this but do it anyway)**

```bash
cargo check
```

**Step 5: Commit**

```bash
git add crawler-cli/assets/dashboard.html
git commit -m "feat(crawler-dashboard): add Data card with disk stats"
```

---

### Task 7: Build and deploy

**Step 1: Full build check**

```bash
source ~/.cargo/env && cargo check
```

Expected: clean (1 dead_code warning is OK).

**Step 2: Deploy to server2**

```bash
make build-on-server SERVER=2
```

Expected: `BUILD_OK` and `~/bin/crawler --help` succeeds.

**Step 3: Verify on server2**

SSH tunnel if using GUI:
```bash
make tunnel SERVER=2
```

Run a quick test crawl with `--gui`:
```bash
ssh root@server2 "~/bin/crawler cc recrawl --file p:0 --limit 1000 --gui --no-retry"
```

Open `http://localhost:9111` and confirm the Data card appears after ~12s (2s startup + first 10s interval).

**Step 4: Final commit if any tweaks needed**

```bash
git add -p
git commit -m "fix(crawler-data-pane): <describe tweak>"
```
