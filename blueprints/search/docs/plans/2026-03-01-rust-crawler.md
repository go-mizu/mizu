# Rust Crawler Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a high-throughput multi-domain Rust recrawler at `tools/crawler/` with two HTTP engines (reqwest, hyper), three write modes (duckdb, parquet, binary), and two-pass retry — matching `pkg/crawl`'s keepalive engine architecture.

**Architecture:** Cargo workspace with `crawler-lib` (core library) and `crawler-cli` (binary). Domain-grouped batch processing with adaptive timeouts, dead-domain probing, and lock-free atomic stats. Generic library for HN and CC recrawl.

**Tech Stack:** Rust 2024 edition, tokio async runtime, reqwest + hyper-rustls, duckdb-rs, arrow/parquet, clap v4, crossbeam channels.

---

### Task 1: Scaffold Cargo workspace

**Files:**
- Create: `tools/crawler/Cargo.toml`
- Create: `tools/crawler/crawler-lib/Cargo.toml`
- Create: `tools/crawler/crawler-lib/src/lib.rs`
- Create: `tools/crawler/crawler-cli/Cargo.toml`
- Create: `tools/crawler/crawler-cli/src/main.rs`

**Step 1: Create workspace root Cargo.toml**

```toml
# tools/crawler/Cargo.toml
[workspace]
members = ["crawler-lib", "crawler-cli"]
resolver = "2"
```

**Step 2: Create crawler-lib/Cargo.toml**

```toml
[package]
name = "crawler-lib"
version = "0.1.0"
edition = "2024"

[dependencies]
tokio = { version = "1", features = ["full"] }
reqwest = { version = "0.12", default-features = false, features = ["rustls-tls", "gzip", "brotli", "deflate"] }
hyper = { version = "1", features = ["client", "http1", "http2"] }
hyper-util = { version = "0.1", features = ["client-legacy", "tokio", "http1", "http2"] }
hyper-rustls = { version = "0.27", features = ["http2", "ring"] }
rustls = "0.23"
http-body-util = "0.1"
bytes = "1"

duckdb = { version = "1.2", features = ["bundled"] }
arrow = { version = "54", default-features = false }
parquet = { version = "54", features = ["snap", "zstd"] }

chrono = { version = "0.4", features = ["serde"] }
url = "2"
addr = "0.15"
serde = { version = "1", features = ["derive"] }
bincode = "1"
tracing = "0.1"
anyhow = "1"
thiserror = "2"
rand = "0.9"
ahash = "0.8"
crossbeam-channel = "0.5"
sysinfo = "0.34"
```

**Step 3: Create crawler-lib/src/lib.rs**

```rust
pub mod config;
pub mod domain;
pub mod engine;
pub mod job;
pub mod seed;
pub mod stats;
pub mod types;
pub mod ua;
pub mod writer;
```

**Step 4: Create crawler-cli/Cargo.toml**

```toml
[package]
name = "crawler-cli"
version = "0.1.0"
edition = "2024"

[[bin]]
name = "crawler"
path = "src/main.rs"

[dependencies]
crawler-lib = { path = "../crawler-lib" }
clap = { version = "4", features = ["derive"] }
tokio = { version = "1", features = ["full"] }
tracing = "0.1"
tracing-subscriber = { version = "0.3", features = ["env-filter"] }
anyhow = "1"
chrono = "0.4"
```

**Step 5: Create minimal main.rs**

```rust
use clap::Parser;

#[derive(Parser)]
#[command(name = "crawler", about = "High-throughput multi-domain recrawler")]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(clap::Subcommand)]
enum Commands {
    /// HN recrawl
    Hn {
        #[command(subcommand)]
        action: HnAction,
    },
}

#[derive(clap::Subcommand)]
enum HnAction {
    /// Recrawl HN seed URLs
    Recrawl(Box<crawler_cli::hn::RecrawlArgs>),
}

fn main() {
    // placeholder
}
```

**Step 6: Verify it compiles**

Run: `cd tools/crawler && cargo check`
Expected: compiles with warnings about unused imports

**Step 7: Commit**

```bash
git add tools/crawler/
git commit -m "feat(crawler): scaffold Rust workspace with crawler-lib and crawler-cli"
```

---

### Task 2: Core types and config

**Files:**
- Create: `crawler-lib/src/types.rs`
- Create: `crawler-lib/src/config.rs`
- Create: `crawler-lib/src/ua.rs`
- Create: `crawler-lib/src/stats.rs`

**Step 1: Write types.rs**

```rust
use chrono::NaiveDateTime;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SeedURL {
    pub url: String,
    pub domain: String,
}

#[derive(Debug, Clone)]
pub struct CrawlResult {
    pub url: String,
    pub domain: String,
    pub status_code: u16,
    pub content_type: String,
    pub content_length: i64,
    pub title: String,
    pub description: String,
    pub language: String,
    pub redirect_url: String,
    pub fetch_time_ms: i64,
    pub crawled_at: NaiveDateTime,
    pub error: String,
    pub body: String, // always empty (overflow block fix)
}

#[derive(Debug, Clone)]
pub struct FailedURL {
    pub url: String,
    pub domain: String,
    pub reason: String,
    pub error: String,
    pub status_code: u16,
    pub fetch_time_ms: i64,
    pub detected_at: NaiveDateTime,
}

#[derive(Debug, Clone)]
pub struct FailedDomain {
    pub domain: String,
    pub reason: String,
    pub error: String,
    pub url_count: i64,
    pub detected_at: NaiveDateTime,
}
```

**Step 2: Write stats.rs** — lock-free atomic counters + adaptive timeout histogram

```rust
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{Duration, Instant};

#[derive(Debug)]
pub struct Stats {
    pub ok: AtomicU64,
    pub failed: AtomicU64,
    pub timeout: AtomicU64,
    pub skipped: AtomicU64,
    pub bytes_downloaded: AtomicU64,
    pub total: AtomicU64,
    pub start: Instant,
    pub peak_rps: AtomicU64,
}

impl Stats {
    pub fn new() -> Self {
        Self {
            ok: AtomicU64::new(0),
            failed: AtomicU64::new(0),
            timeout: AtomicU64::new(0),
            skipped: AtomicU64::new(0),
            bytes_downloaded: AtomicU64::new(0),
            total: AtomicU64::new(0),
            start: Instant::now(),
            peak_rps: AtomicU64::new(0),
        }
    }

    pub fn snapshot(&self) -> StatsSnapshot {
        StatsSnapshot {
            ok: self.ok.load(Ordering::Relaxed),
            failed: self.failed.load(Ordering::Relaxed),
            timeout: self.timeout.load(Ordering::Relaxed),
            skipped: self.skipped.load(Ordering::Relaxed),
            bytes_downloaded: self.bytes_downloaded.load(Ordering::Relaxed),
            total: self.total.load(Ordering::Relaxed),
            duration: self.start.elapsed(),
            peak_rps: self.peak_rps.load(Ordering::Relaxed),
        }
    }
}

#[derive(Debug, Clone)]
pub struct StatsSnapshot {
    pub ok: u64,
    pub failed: u64,
    pub timeout: u64,
    pub skipped: u64,
    pub bytes_downloaded: u64,
    pub total: u64,
    pub duration: Duration,
    pub peak_rps: u64,
}

impl StatsSnapshot {
    pub fn avg_rps(&self) -> f64 {
        let secs = self.duration.as_secs_f64();
        if secs > 0.0 { self.total as f64 / secs } else { 0.0 }
    }
}

/// Lock-free latency histogram for P95-based adaptive timeout.
/// Matches Go's adaptiveTracker: 8 buckets, atomic operations, no mutex.
const ADAPTIVE_EDGES: [i64; 8] = [100, 250, 500, 1000, 2000, 3500, 5000, 10000];

pub struct AdaptiveTimeout {
    buckets: [AtomicU64; 8],
    total: AtomicU64,
}

impl AdaptiveTimeout {
    pub fn new() -> Self {
        Self {
            buckets: std::array::from_fn(|_| AtomicU64::new(0)),
            total: AtomicU64::new(0),
        }
    }

    pub fn record(&self, ms: i64) {
        self.total.fetch_add(1, Ordering::Relaxed);
        for (i, &edge) in ADAPTIVE_EDGES.iter().enumerate() {
            if ms < edge {
                self.buckets[i].fetch_add(1, Ordering::Relaxed);
                return;
            }
        }
        self.buckets[7].fetch_add(1, Ordering::Relaxed);
    }

    /// Returns P95×2 clamped to [500ms, ceiling]. Returns None if <5 samples.
    pub fn timeout(&self, ceiling: Duration) -> Option<Duration> {
        let n = self.total.load(Ordering::Relaxed);
        if n < 5 { return None; }
        let target = (n as f64 * 0.95) as u64;
        let mut cum = 0u64;
        for (i, &edge) in ADAPTIVE_EDGES.iter().enumerate() {
            cum += self.buckets[i].load(Ordering::Relaxed);
            if cum >= target {
                let ms = (edge * 2).max(500);
                let ceil_ms = ceiling.as_millis() as i64;
                let result_ms = ms.min(ceil_ms);
                return Some(Duration::from_millis(result_ms as u64));
            }
        }
        Some(ceiling)
    }

    pub fn p95_ms(&self) -> Option<i64> {
        let n = self.total.load(Ordering::Relaxed);
        if n < 10 { return None; }
        let target = (n as f64 * 0.95) as u64;
        let mut cum = 0u64;
        for (i, &edge) in ADAPTIVE_EDGES.iter().enumerate() {
            cum += self.buckets[i].load(Ordering::Relaxed);
            if cum >= target {
                return Some(edge);
            }
        }
        Some(ADAPTIVE_EDGES[7])
    }
}

/// Tracks peak RPS using a sliding 1-second window.
pub struct PeakTracker {
    count: AtomicU64,
    last_reset: std::sync::Mutex<Instant>,
    peak: AtomicU64,
}

impl PeakTracker {
    pub fn new() -> Self {
        Self {
            count: AtomicU64::new(0),
            last_reset: std::sync::Mutex::new(Instant::now()),
            peak: AtomicU64::new(0),
        }
    }

    pub fn record(&self) {
        let c = self.count.fetch_add(1, Ordering::Relaxed) + 1;
        if let Ok(mut last) = self.last_reset.try_lock() {
            let elapsed = last.elapsed();
            if elapsed >= Duration::from_secs(1) {
                let rps = (c as f64 / elapsed.as_secs_f64()) as u64;
                self.peak.fetch_max(rps, Ordering::Relaxed);
                self.count.store(0, Ordering::Relaxed);
                *last = Instant::now();
            }
        }
    }

    pub fn peak(&self) -> u64 {
        self.peak.load(Ordering::Relaxed)
    }
}
```

**Step 3: Write ua.rs** — browser User-Agent rotation pool

```rust
use rand::Rng;

pub const BROWSER_USER_AGENTS: &[&str] = &[
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:133.0) Gecko/20100101 Firefox/133.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Safari/605.1.15",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
    "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.6778.39 Mobile Safari/537.36",
];

pub fn pick_user_agent() -> &'static str {
    let idx = rand::rng().random_range(0..BROWSER_USER_AGENTS.len());
    BROWSER_USER_AGENTS[idx]
}
```

**Step 4: Write config.rs**

```rust
use std::time::Duration;
use sysinfo::System;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EngineType { Reqwest, Hyper }

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum WriterType { DuckDB, Parquet, Binary, DevNull }

#[derive(Debug, Clone)]
pub struct Config {
    pub workers: usize,
    pub inner_n: usize,
    pub timeout: Duration,
    pub domain_timeout: i64,        // ms; 0=disabled, <0=adaptive
    pub adaptive_timeout_max: Duration,
    pub domain_fail_threshold: usize,
    pub domain_dead_probe: usize,
    pub domain_stall_ratio: usize,
    pub disable_adaptive_timeout: bool,
    pub engine: EngineType,
    pub writer: WriterType,
    pub batch_size: usize,
    pub db_shards: usize,
    pub db_mem_mb: usize,
    pub retry_timeout: Duration,
    pub no_retry: bool,
    pub pass2_workers: usize,
    pub max_body_bytes: usize,
    pub output_dir: String,
    pub failed_db_path: String,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            workers: 0,
            inner_n: 0,
            timeout: Duration::from_millis(1000),
            domain_timeout: -1, // adaptive
            adaptive_timeout_max: Duration::from_secs(600),
            domain_fail_threshold: 3,
            domain_dead_probe: 10,
            domain_stall_ratio: 20,
            disable_adaptive_timeout: false,
            engine: EngineType::Reqwest,
            writer: WriterType::Binary,
            batch_size: 5000,
            db_shards: 0,
            db_mem_mb: 0,
            retry_timeout: Duration::from_millis(15000),
            no_retry: false,
            pass2_workers: 0,
            max_body_bytes: 256 * 1024,
            output_dir: String::new(),
            failed_db_path: String::new(),
        }
    }
}

#[derive(Debug, Clone)]
pub struct SysInfo {
    pub cpu_count: usize,
    pub mem_total_mb: u64,
    pub mem_available_mb: u64,
    pub fd_soft_limit: u64,
}

impl SysInfo {
    pub fn gather() -> Self {
        let mut sys = System::new_all();
        sys.refresh_all();
        let cpu_count = sys.cpus().len();
        let mem_total_mb = sys.total_memory() / (1024 * 1024);
        let mem_available_mb = sys.available_memory() / (1024 * 1024);

        #[cfg(unix)]
        let fd_soft_limit = {
            use std::io;
            let mut rlim = libc::rlimit { rlim_cur: 0, rlim_max: 0 };
            if unsafe { libc::getrlimit(libc::RLIMIT_NOFILE, &mut rlim) } == 0 {
                rlim.rlim_cur
            } else { 1024 }
        };
        #[cfg(not(unix))]
        let fd_soft_limit = 1024u64;

        Self { cpu_count, mem_total_mb, mem_available_mb, fd_soft_limit }
    }
}

fn clamp(val: usize, min: usize, max: usize) -> usize {
    val.max(min).min(max)
}

/// Auto-configure workers and inner_n based on hardware.
/// Matches Go's AutoConfigKeepAlive formula.
pub fn auto_config(si: &SysInfo, full_body: bool) -> Config {
    let body_kb: usize = if full_body { 256 } else { 4 };
    let avail_kb = (si.mem_available_mb as usize) * 1024;

    let inner_n_min = 4usize;
    let w_mem_uncapped = std::cmp::min(
        avail_kb * 70 / 100 / (inner_n_min * body_kb / 4),
        avail_kb * 80 / 100 / (inner_n_min * body_kb),
    );

    let fd = si.fd_soft_limit as usize;
    let inner_n;
    if fd / (inner_n_min * 2) <= w_mem_uncapped {
        inner_n = inner_n_min;
    } else {
        inner_n = clamp(
            si.cpu_count * 2,
            4,
            std::cmp::min(16, fd / (2 * w_mem_uncapped.max(1))),
        );
    }

    let w_mem = std::cmp::min(
        avail_kb * 70 / 100 / (inner_n * body_kb / 4).max(1),
        avail_kb * 80 / 100 / (inner_n * body_kb).max(1),
    );
    let w_fd = fd / (inner_n * 2).max(1);
    let workers = clamp(std::cmp::min(w_mem, w_fd).min(10000), 200, 10000);

    let db_shards = clamp(si.cpu_count * 2, 4, 16);
    let db_mem_mb = ((si.mem_available_mb as usize) * 15 / 100 / db_shards).max(64);

    let mut cfg = Config::default();
    cfg.workers = workers;
    cfg.inner_n = inner_n;
    cfg.db_shards = db_shards;
    cfg.db_mem_mb = db_mem_mb;
    cfg
}
```

**Step 5: Verify it compiles**

Run: `cd tools/crawler && cargo check`

**Step 6: Commit**

```bash
git add tools/crawler/
git commit -m "feat(crawler): core types, config, stats, and UA rotation"
```

---

### Task 3: Domain grouping and management

**Files:**
- Create: `crawler-lib/src/domain.rs`

**Step 1: Write domain.rs** — sorting, batching, and per-domain state

```rust
use crate::types::SeedURL;
use crate::config::Config;

#[derive(Debug)]
pub struct DomainBatch {
    pub domain: String,
    pub urls: Vec<SeedURL>,
}

/// Sort seeds by domain, yield contiguous batches.
pub fn group_by_domain(mut seeds: Vec<SeedURL>) -> Vec<DomainBatch> {
    seeds.sort_by(|a, b| a.domain.cmp(&b.domain));
    let mut batches = Vec::new();
    let mut current_domain = String::new();
    let mut current_urls = Vec::new();
    for seed in seeds {
        if seed.domain != current_domain {
            if !current_urls.is_empty() {
                batches.push(DomainBatch {
                    domain: std::mem::take(&mut current_domain),
                    urls: std::mem::take(&mut current_urls),
                });
            }
            current_domain = seed.domain.clone();
        }
        current_urls.push(seed);
    }
    if !current_urls.is_empty() {
        batches.push(DomainBatch {
            domain: current_domain,
            urls: current_urls,
        });
    }
    batches
}

/// Per-domain state tracking during crawl.
pub struct DomainState {
    pub successes: u64,
    pub timeouts: u64,
    pub consecutive_timeouts: u64,
}

impl DomainState {
    pub fn new() -> Self {
        Self { successes: 0, timeouts: 0, consecutive_timeouts: 0 }
    }

    /// Check if domain should be abandoned based on config rules.
    pub fn should_abandon(&self, cfg: &Config, inner_n: usize) -> bool {
        // DomainFailThreshold: N rounds of all-timeout
        if cfg.domain_fail_threshold > 0 {
            let effective = (cfg.domain_fail_threshold * inner_n.max(1)) as u64;
            if self.timeouts >= effective {
                return true;
            }
        }
        // DomainDeadProbe: N timeouts with 0 success
        if cfg.domain_dead_probe > 0 && self.timeouts >= cfg.domain_dead_probe as u64 {
            if self.successes == 0 {
                return true;
            }
            // Stall ratio: timeouts >= successes * ratio
            if cfg.domain_stall_ratio > 0
                && self.successes > 0
                && self.timeouts >= self.successes * cfg.domain_stall_ratio as u64
            {
                return true;
            }
        }
        false
    }
}
```

**Step 2: Verify**

Run: `cd tools/crawler && cargo check`

**Step 3: Commit**

```bash
git add tools/crawler/crawler-lib/src/domain.rs
git commit -m "feat(crawler): domain grouping and abandonment logic"
```

---

### Task 4: Seed loading (DuckDB + parquet)

**Files:**
- Create: `crawler-lib/src/seed.rs`

**Step 1: Write seed.rs**

```rust
use crate::types::SeedURL;
use anyhow::{Context, Result};
use chrono::NaiveDateTime;
use duckdb::Connection;

/// Load seed URLs from a DuckDB database.
pub fn load_seeds_duckdb(path: &str, limit: usize) -> Result<Vec<SeedURL>> {
    let conn = Connection::open_with_flags(path, duckdb::Config::default()
        .access_mode(duckdb::AccessMode::ReadOnly)?)?;

    let query = if limit > 0 {
        format!("SELECT url, COALESCE(domain, '') as domain FROM docs LIMIT {}", limit)
    } else {
        "SELECT url, COALESCE(domain, '') as domain FROM docs".to_string()
    };

    let mut stmt = conn.prepare(&query)?;
    let seeds: Vec<SeedURL> = stmt
        .query_map([], |row| {
            Ok(SeedURL {
                url: row.get(0)?,
                domain: row.get(1)?,
            })
        })?
        .filter_map(|r| r.ok())
        .collect();

    Ok(seeds)
}

/// Load seed URLs from a parquet file using DuckDB's read_parquet.
pub fn load_seeds_parquet(path: &str, limit: usize) -> Result<Vec<SeedURL>> {
    let conn = Connection::open_in_memory()?;

    let query = if limit > 0 {
        format!(
            "SELECT url, COALESCE(domain, '') as domain FROM read_parquet('{}') LIMIT {}",
            path.replace('\'', "''"), limit
        )
    } else {
        format!(
            "SELECT url, COALESCE(domain, '') as domain FROM read_parquet('{}')",
            path.replace('\'', "''")
        )
    };

    let mut stmt = conn.prepare(&query)?;
    let seeds: Vec<SeedURL> = stmt
        .query_map([], |row| {
            Ok(SeedURL {
                url: row.get(0)?,
                domain: row.get(1)?,
            })
        })?
        .filter_map(|r| r.ok())
        .collect();

    Ok(seeds)
}

/// Load timeout URLs from failed DB for pass-2 retry.
/// Only loads URLs from after `since` to avoid accumulation from prior runs.
pub fn load_retry_seeds(path: &str, since: NaiveDateTime) -> Result<Vec<SeedURL>> {
    let conn = Connection::open_with_flags(path, duckdb::Config::default()
        .access_mode(duckdb::AccessMode::ReadOnly)?)?;

    let mut stmt = conn.prepare(
        "SELECT url, COALESCE(domain, '') as domain FROM failed_urls \
         WHERE reason = 'http_timeout' AND detected_at >= ?"
    )?;

    let seeds: Vec<SeedURL> = stmt
        .query_map([since.to_string()], |row| {
            Ok(SeedURL {
                url: row.get(0)?,
                domain: row.get(1)?,
            })
        })?
        .filter_map(|r| r.ok())
        .collect();

    Ok(seeds)
}
```

**Step 2: Verify**

Run: `cd tools/crawler && cargo check`

**Step 3: Commit**

```bash
git add tools/crawler/crawler-lib/src/seed.rs
git commit -m "feat(crawler): seed loading from DuckDB and parquet"
```

---

### Task 5: Writer traits + DevNull writer

**Files:**
- Create: `crawler-lib/src/writer/mod.rs`
- Create: `crawler-lib/src/writer/devnull.rs`

**Step 1: Write writer/mod.rs** — traits

```rust
pub mod devnull;
pub mod duckdb_writer;
pub mod parquet_writer;
pub mod binary;

use crate::types::{CrawlResult, FailedURL, FailedDomain};
use anyhow::Result;

pub trait ResultWriter: Send + Sync {
    fn write(&self, result: CrawlResult) -> Result<()>;
    fn flush(&self) -> Result<()>;
    fn close(&self) -> Result<()>;
}

pub trait FailureWriter: Send + Sync {
    fn write_url(&self, failed: FailedURL) -> Result<()>;
    fn write_domain(&self, failed: FailedDomain) -> Result<()>;
    fn flush(&self) -> Result<()>;
    fn close(&self) -> Result<()>;
}
```

**Step 2: Write writer/devnull.rs**

```rust
use super::{ResultWriter, FailureWriter};
use crate::types::{CrawlResult, FailedURL, FailedDomain};
use anyhow::Result;

pub struct DevNullResultWriter;
pub struct DevNullFailureWriter;

impl ResultWriter for DevNullResultWriter {
    fn write(&self, _result: CrawlResult) -> Result<()> { Ok(()) }
    fn flush(&self) -> Result<()> { Ok(()) }
    fn close(&self) -> Result<()> { Ok(()) }
}

impl FailureWriter for DevNullFailureWriter {
    fn write_url(&self, _failed: FailedURL) -> Result<()> { Ok(()) }
    fn write_domain(&self, _failed: FailedDomain) -> Result<()> { Ok(()) }
    fn flush(&self) -> Result<()> { Ok(()) }
    fn close(&self) -> Result<()> { Ok(()) }
}
```

**Step 3: Create stub files for duckdb_writer.rs, parquet_writer.rs, binary.rs** (empty modules for now)

**Step 4: Verify**

Run: `cd tools/crawler && cargo check`

**Step 5: Commit**

```bash
git add tools/crawler/crawler-lib/src/writer/
git commit -m "feat(crawler): writer traits and devnull implementation"
```

---

### Task 6: DuckDB sharded writer

**Files:**
- Create: `crawler-lib/src/writer/duckdb_writer.rs`

**Step 1: Write the sharded DuckDB writer** — matches Go's ResultDB

Key features: N shards, FNV-1a hash, batched multi-row VALUES insert, async flusher per shard via crossbeam channel.

Schema matches Go's `initResultSchema`:
```sql
CREATE TABLE IF NOT EXISTS results (
    url VARCHAR, status_code INTEGER, content_type VARCHAR,
    content_length BIGINT, body VARCHAR, title VARCHAR,
    description VARCHAR, language VARCHAR, domain VARCHAR,
    redirect_url VARCHAR, fetch_time_ms BIGINT,
    crawled_at TIMESTAMP, error VARCHAR,
    status VARCHAR DEFAULT 'done', body_cid VARCHAR DEFAULT ''
)
```

DuckDB per-shard settings: `SET memory_limit`, `SET threads=1`, `SET preserve_insertion_order=false`, `SET checkpoint_threshold`.

Use `std::thread::spawn` for flusher threads (DuckDB is sync, crossbeam channels bridge async→sync).

Also implement `DuckDBFailureWriter` with two tables: `failed_domains` and `failed_urls` matching Go's `initFailedSchema`.

**Step 2: Verify**

Run: `cd tools/crawler && cargo check`

**Step 3: Commit**

```bash
git add tools/crawler/crawler-lib/src/writer/duckdb_writer.rs
git commit -m "feat(crawler): sharded DuckDB result and failure writers"
```

---

### Task 7: Parquet writer

**Files:**
- Create: `crawler-lib/src/writer/parquet_writer.rs`

**Step 1: Write parquet writer**

Use `arrow::array::*` builders to construct `RecordBatch`, write with `parquet::arrow::ArrowWriter`. Buffer results in memory, flush as row groups when batch_size reached. Snappy compression by default.

Schema: same columns as DuckDB results table, mapped to Arrow types (Utf8, Int32, Int64, TimestampMillisecond).

**Step 2: Verify + commit**

---

### Task 8: Binary segment writer

**Files:**
- Create: `crawler-lib/src/writer/binary.rs`

**Step 1: Write binary segment writer**

Architecture matching Go's BinSegWriter:
- Workers → crossbeam bounded channel → flusher thread → bincode segment files
- Segment rotation at 64MB
- After crawl: drain segments → DuckDB via `duckdb_writer`

Use `bincode` for serialization. Segment files: `seg_000.bin`, `seg_001.bin`, etc.

**Step 2: Verify + commit**

---

### Task 9: Engine trait + reqwest engine

**Files:**
- Create: `crawler-lib/src/engine/mod.rs`
- Create: `crawler-lib/src/engine/reqwest_engine.rs`

**Step 1: Write engine/mod.rs** — Engine trait

```rust
use crate::config::Config;
use crate::stats::{Stats, StatsSnapshot};
use crate::types::SeedURL;
use crate::writer::{ResultWriter, FailureWriter};
use anyhow::Result;
use std::sync::Arc;

#[async_trait::async_trait]
pub trait Engine: Send + Sync {
    async fn run(
        &self,
        seeds: Vec<SeedURL>,
        cfg: &Config,
        results: Arc<dyn ResultWriter>,
        failures: Arc<dyn FailureWriter>,
    ) -> Result<StatsSnapshot>;
}
```

Add `async-trait = "0.1"` to Cargo.toml.

**Step 2: Write engine/reqwest_engine.rs** — the main engine

This is the critical file. It must implement:

1. Sort seeds by domain (reuse `domain::group_by_domain`)
2. Create `tokio::sync::mpsc` work channel of `DomainBatch`
3. Spawn `workers` tokio tasks, each draining from the work channel
4. Per domain: create `reqwest::Client` with `pool_max_idle_per_host(inner_n)`, `timeout`, `danger_accept_invalid_certs`
5. Spawn `inner_n` concurrent fetch tasks per domain sharing the client
6. Per-domain adaptive timeout via `AdaptiveTimeout`
7. Domain abandonment via `DomainState::should_abandon`
8. Domain context deadline (adaptive sweep calculation matching Go)
9. Timeout detection: check if error message contains "timeout" or "deadline"
10. Results/failures written via trait objects

Key patterns from Go's keepalive.go:
- `abandonCh` → use `tokio::sync::watch` or `tokio_util::sync::CancellationToken`
- `domainTimeouts` / `domainSuccesses` → `AtomicU64` per domain
- `effectiveDomainTimeout` → adaptive sweep: `len(urls) * timeout / inner_n * 2`, clamped [30s, max]
- Fetch function: `reqwest::Client::get(url).header("User-Agent", pick_ua).send().await`
- Body reading: `response.bytes().await` limited to `max_body_bytes`
- HTML metadata extraction: parse `<title>` and `<meta name="description">` from body

**Step 3: Verify**

Run: `cd tools/crawler && cargo check`

**Step 4: Commit**

```bash
git add tools/crawler/crawler-lib/src/engine/
git commit -m "feat(crawler): engine trait and reqwest engine with domain-grouped processing"
```

---

### Task 10: Hyper engine

**Files:**
- Create: `crawler-lib/src/engine/hyper_engine.rs`

**Step 1: Write hyper engine**

Same domain-grouped architecture as reqwest engine but using:
- `hyper_util::client::legacy::Client` with `hyper_rustls::HttpsConnectorBuilder`
- Manual redirect following (301/302/307/308 loop, max 7 redirects)
- Manual body reading via `http_body_util::BodyExt::collect().await`
- TCP_NODELAY via custom connector
- HTTP/2 prior knowledge option

**Step 2: Verify + commit**

---

### Task 11: Two-pass job orchestration

**Files:**
- Create: `crawler-lib/src/job.rs`

**Step 1: Write job.rs** — matching Go's RunJob

```rust
pub struct JobConfig {
    pub config: Config,
    pub seed_path: String,
}

pub struct JobResult {
    pub pass1: StatsSnapshot,
    pub pass2: Option<StatsSnapshot>,
    pub total: StatsSnapshot,
    pub start: chrono::NaiveDateTime,
    pub end: chrono::NaiveDateTime,
    pub workers: usize,
}

pub async fn run_job(
    seeds: Vec<SeedURL>,
    cfg: Config,
    result_writer: Arc<dyn ResultWriter>,
    open_failure_writer: impl Fn() -> Result<Arc<dyn FailureWriter>>,
    load_retry_seeds: Option<Box<dyn Fn(NaiveDateTime) -> Result<Vec<SeedURL>> + Send>>,
) -> Result<JobResult>
```

Logic:
1. Auto-config if workers=0
2. Create engine (reqwest or hyper based on cfg.engine)
3. Pass 1: `engine.run(seeds, &cfg, results, failures1)`
4. Close failure writer 1
5. If `!cfg.no_retry && cfg.retry_timeout > 0`:
   a. Load retry seeds via `load_retry_seeds(start_time)`
   b. Create pass-2 config: `disable_adaptive_timeout=true`, `domain_dead_probe=2`, `timeout=retry_timeout`
   c. `engine.run(retry_seeds, &retry_cfg, results, failures2)`
6. Merge stats
7. Return JobResult

**Step 2: Verify + commit**

---

### Task 12: CLI — HN recrawl subcommand

**Files:**
- Create: `crawler-cli/src/hn.rs`
- Create: `crawler-cli/src/display.rs`
- Modify: `crawler-cli/src/main.rs`

**Step 1: Write hn.rs** — RecrawlArgs with clap derive

```rust
#[derive(clap::Args, Debug)]
pub struct RecrawlArgs {
    #[arg(long)]
    pub seed: String,
    #[arg(long, default_value = "")]
    pub output: String,
    #[arg(long, default_value = "reqwest")]
    pub engine: String,       // reqwest | hyper
    #[arg(long, default_value = "binary")]
    pub writer: String,       // duckdb | parquet | binary | devnull
    #[arg(long, default_value_t = 0)]
    pub workers: usize,
    #[arg(long, default_value_t = 0)]
    pub inner_n: usize,
    #[arg(long, default_value_t = 1000)]
    pub timeout: u64,         // ms
    #[arg(long, default_value_t = 15000)]
    pub retry_timeout: u64,   // ms
    #[arg(long)]
    pub no_retry: bool,
    #[arg(long, default_value_t = 10)]
    pub domain_dead_probe: usize,
    #[arg(long, default_value_t = 20)]
    pub domain_stall_ratio: usize,
    #[arg(long, default_value_t = 0)]
    pub limit: usize,
    #[arg(long, default_value_t = 0)]
    pub db_shards: usize,
    #[arg(long, default_value_t = 0)]
    pub db_mem_mb: usize,
}
```

**Step 2: Write display.rs** — live progress display

Print every 500ms: `ok/total | avg rps | peak rps | timeout | elapsed`

**Step 3: Wire up main.rs** — parse CLI, load seeds, create writers, call `run_job`, display results

**Step 4: Verify end-to-end**

Run: `cd tools/crawler && cargo build --release`
Run: `./target/release/crawler hn recrawl --seed /path/to/seeds.duckdb --limit 100 --writer devnull`

**Step 5: Commit**

```bash
git add tools/crawler/crawler-cli/
git commit -m "feat(crawler): HN recrawl CLI with progress display"
```

---

### Task 13: Build and deploy infrastructure

**Files:**
- Create: `tools/crawler/Dockerfile`
- Create: `tools/crawler/Makefile`

**Step 1: Write Dockerfile** (Ubuntu 24.04, AVX2)

```dockerfile
FROM ubuntu:24.04 AS builder
RUN apt-get update && apt-get install -y curl build-essential pkg-config
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:$PATH"
ENV RUSTFLAGS="-C target-cpu=x86-64-v3"
WORKDIR /build
COPY . .
RUN cargo build --release
FROM scratch
COPY --from=builder /build/target/release/crawler /out/crawler
```

**Step 2: Write Makefile**

```makefile
DEPLOY_KEY    ?= $(HOME)/.ssh/id_ed25519_deploy
SERVER1_USER  ?= tam
SERVER1_HOST  ?= server
SERVER2_USER  ?= root
SERVER2_HOST  ?= server2
SERVER        ?= 1
REMOTE_USER   ?= $(if $(filter 2,$(SERVER)),$(SERVER2_USER),$(SERVER1_USER))
REMOTE_HOST   ?= $(if $(filter 2,$(SERVER)),$(SERVER2_HOST),$(SERVER1_HOST))
REMOTE_SSH     = $(REMOTE_USER)@$(REMOTE_HOST)
SSH           ?= ssh -i $(DEPLOY_KEY) -o BatchMode=yes
SCP           ?= scp -i $(DEPLOY_KEY)

.PHONY: build build-linux deploy test benchmark

build:
	cargo build --release

build-linux:
	docker build -t crawler-linux -f Dockerfile .
	@docker rm -f crawler-tmp 2>/dev/null || true
	docker create --name crawler-tmp crawler-linux
	docker cp crawler-tmp:/out/crawler ./crawler-linux
	docker rm crawler-tmp

deploy: build-linux
	$(SCP) ./crawler-linux $(REMOTE_SSH):~/bin/.crawler-upload.tmp
	$(SSH) $(REMOTE_SSH) 'mv ~/bin/.crawler-upload.tmp ~/bin/crawler && chmod +x ~/bin/crawler'
	$(SSH) $(REMOTE_SSH) '~/bin/crawler --help'

deploy-server1:
	$(MAKE) deploy SERVER=1

deploy-server2:
	$(MAKE) deploy SERVER=2

remote-test:
	$(SSH) $(REMOTE_SSH) '~/bin/crawler --help'

remote-hn-recrawl:
	$(SSH) $(REMOTE_SSH) '~/bin/crawler hn recrawl --seed ~/data/hn/hn_domains.duckdb'

remote-benchmark:
	$(SSH) $(REMOTE_SSH) '~/bin/crawler hn recrawl --seed ~/data/hn/hn_domains.duckdb --limit 10000 --writer devnull'

test:
	cargo test

clean:
	cargo clean
	rm -f crawler-linux
```

**Step 3: Commit**

```bash
git add tools/crawler/Dockerfile tools/crawler/Makefile
git commit -m "feat(crawler): Dockerfile and Makefile for build and deploy"
```

---

### Task 14: Local end-to-end test

**Step 1: Build release binary**

Run: `cd tools/crawler && cargo build --release`

**Step 2: Test with small seed file**

Run: `./target/release/crawler hn recrawl --seed ~/data/hn/hn_domains.duckdb --limit 100 --writer devnull`

Expected: Runs, shows progress, completes with stats summary.

**Step 3: Test with DuckDB writer**

Run: `./target/release/crawler hn recrawl --seed ~/data/hn/hn_domains.duckdb --limit 100 --writer duckdb --output /tmp/crawler-test/`

Expected: Creates sharded DuckDB files in `/tmp/crawler-test/`

**Step 4: Verify DuckDB output**

Run: `duckdb /tmp/crawler-test/results_000.duckdb "SELECT COUNT(*) FROM results"`

**Step 5: Commit any fixes**

---

### Task 15: Deploy to servers and benchmark

**Step 1: Build Linux binary**

Run: `cd tools/crawler && make build-linux`
(Takes 15-20 min under QEMU on ARM Mac — use `run_in_background=true`)

**Step 2: Deploy to server 1**

Run: `make deploy SERVER=1`

**Step 3: Deploy to server 2**

Run: `make deploy SERVER=2`

**Step 4: Benchmark on server 2 — devnull baseline**

Run: `make remote-benchmark SERVER=2`
Target: establish raw fetch baseline (pages/s without write overhead)

**Step 5: Benchmark on server 2 — with DuckDB writer**

Run: `ssh -i ~/.ssh/id_ed25519_deploy root@server2 '~/bin/crawler hn recrawl --seed ~/data/hn/hn_domains.duckdb --limit 50000 --writer duckdb'`

**Step 6: Benchmark on server 2 — with binary writer**

Run: `ssh -i ~/.ssh/id_ed25519_deploy root@server2 '~/bin/crawler hn recrawl --seed ~/data/hn/hn_domains.duckdb --limit 50000 --writer binary'`

**Step 7: Benchmark on server 2 — hyper engine**

Run: `ssh -i ~/.ssh/id_ed25519_deploy root@server2 '~/bin/crawler hn recrawl --seed ~/data/hn/hn_domains.duckdb --limit 50000 --engine hyper --writer devnull'`

**Step 8: Record benchmark results and commit**

---

### Task 16: Optimize for 5000+ pages/s on server 2

Based on benchmark results from Task 15, apply targeted optimizations:

**Likely bottlenecks and fixes:**
1. **Connection pool exhaustion**: Increase `pool_max_idle_per_host`, add TCP_NODELAY
2. **TLS handshake overhead**: Enable TLS session resumption in rustls
3. **Channel contention**: Switch from `tokio::sync::mpsc` to `crossbeam` for hot paths
4. **Body parsing**: Use `lol_html` streaming HTML parser for zero-copy title/description extraction
5. **Memory allocation**: Use `Bytes` instead of `String` for body, arena allocators for results
6. **fd limits**: Call `setrlimit(RLIMIT_NOFILE, 131072)` at startup
7. **DNS**: Use `hickory-dns` (formerly trust-dns) for async cached resolution

Each optimization should be measured independently against the devnull baseline.

**Step 1: Profile**

Run: `ssh ... 'perf record -g ~/bin/crawler hn recrawl --seed ... --limit 100000 --writer devnull'`

**Step 2: Apply top optimization**

**Step 3: Re-benchmark**

**Step 4: Iterate until 5000+ pages/s**

**Step 5: Commit**

---

### Task 17: CC recrawl stub

**Files:**
- Create: `crawler-cli/src/cc.rs`
- Modify: `crawler-cli/src/main.rs`

**Step 1: Add CC subcommand stub**

```rust
#[derive(clap::Args, Debug)]
pub struct CcRecrawlArgs {
    #[arg(long)]
    pub seed: String,
    // Same flags as HN + CC-specific:
    #[arg(long)]
    pub file: Option<String>,
    #[arg(long)]
    pub sample: Option<usize>,
    // ... inherits all HN flags
}
```

Wire into main.rs `Commands::Cc { action: CcAction }`.

**Step 2: Commit**

```bash
git add tools/crawler/crawler-cli/src/cc.rs
git commit -m "feat(crawler): CC recrawl subcommand stub for future extension"
```

---

## Execution Order Summary

| Task | Component | Dependencies |
|------|-----------|-------------|
| 1 | Scaffold workspace | none |
| 2 | Types, config, stats, UA | Task 1 |
| 3 | Domain grouping | Task 2 |
| 4 | Seed loading | Task 2 |
| 5 | Writer traits + devnull | Task 2 |
| 6 | DuckDB writer | Task 5 |
| 7 | Parquet writer | Task 5 |
| 8 | Binary writer | Task 5, 6 |
| 9 | Engine trait + reqwest | Task 2, 3, 5 |
| 10 | Hyper engine | Task 9 |
| 11 | Job orchestration | Task 4, 9, 5 |
| 12 | CLI (HN recrawl) | Task 11 |
| 13 | Dockerfile + Makefile | Task 12 |
| 14 | Local e2e test | Task 12 |
| 15 | Deploy + benchmark | Task 13, 14 |
| 16 | Optimize for 5000+ | Task 15 |
| 17 | CC stub | Task 12 |

Tasks 3, 4, 5 can run in parallel after Task 2.
Tasks 6, 7 can run in parallel after Task 5.
Task 9 depends on 2, 3, 5. Task 10 depends on 9.
