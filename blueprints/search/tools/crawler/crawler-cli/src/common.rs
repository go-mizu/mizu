//! Shared utilities for HN and CC recrawl commands.

use anyhow::Result;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::sync::atomic::Ordering;
use std::time::Duration;

use crawler_lib::bodystore::AsyncBodyStore;
use crawler_lib::config::{Config, EngineType, WriterType};
use crawler_lib::job::run_job;
use crawler_lib::seed::vec_to_receiver;
use crawler_lib::stats::Stats;
use crawler_lib::types::SeedURL;
use crawler_lib::writer::binary::{BinDrainConfig, BinFailureDrainConfig, BinaryFailureWriter, BinaryResultWriter};
use crawler_lib::writer::devnull::{DevNullFailureWriter, DevNullResultWriter};
use crawler_lib::writer::duckdb_writer::{count_failed_rows, count_result_rows, DuckDBFailureWriter, DuckDBResultWriter};
use crawler_lib::writer::parquet_writer::{ParquetFailureWriter, ParquetResultWriter};
use crawler_lib::writer::{FailureWriter, ResultWriter};

use crate::display::{format_duration, print_summary};
use crate::gui;
use crate::tui;

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

/// Recursively sum file sizes and count files in a directory. Returns (file_count, bytes).
/// Uses iterative stack to handle large flat CAS dirs without stack overflow.
fn scan_dir(dir: &Path) -> (u64, u64) {
    if !dir.exists() {
        return (0, 0);
    }
    let mut count = 0u64;
    let mut bytes = 0u64;
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

/// Spawn a background tokio task that updates disk stats every 10s.
/// Stops after stats.done is observed; row counts are filled by finalize_disk_stats.
pub fn spawn_disk_sampler(stats: Arc<Stats>, paths: DiskPaths) {
    tokio::spawn(async move {
        // give engine 2s to start writing before first scan
        tokio::time::sleep(Duration::from_secs(2)).await;
        let mut interval = tokio::time::interval(Duration::from_secs(10));
        loop {
            interval.tick().await;

            let s = stats.clone();
            let p = paths.clone();
            tokio::task::spawn_blocking(move || {
                if let Some(ref d) = p.seg_dir {
                    let (cnt, bytes) = scan_seg_dir(d);
                    s.disk_seg_files.store(cnt, Ordering::Relaxed);
                    s.disk_seg_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
                }
                if let Some(ref d) = p.duckdb_dir {
                    let bytes = scan_duckdb_dir(d);
                    s.disk_duckdb_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
                }
                if let Some(ref d) = p.failures_dir {
                    let (_, bytes) = scan_dir(d);
                    s.disk_failures_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
                }
                if let Some(ref d) = p.bodies_dir {
                    let (cnt, bytes) = scan_dir(d);
                    s.disk_bodies_count.store(cnt, Ordering::Relaxed);
                    s.disk_bodies_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
                }
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

/// Called AFTER result_writer.close() returns (drain complete).
/// Scans fast local dirs (seg, duckdb shards, small failures dir) + queries DuckDB row counts.
/// Bodies dir is intentionally skipped here — it can contain millions of existing files and
/// scanning it synchronously would block the tokio runtime. The disk_sampler task handles it.
pub fn finalize_disk_stats(stats: &Stats, paths: &DiskPaths) {
    // seg files — small dir, fast
    if let Some(ref d) = paths.seg_dir {
        let (cnt, bytes) = scan_seg_dir(d);
        stats.disk_seg_files.store(cnt, Ordering::Relaxed);
        stats.disk_seg_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
    }
    // duckdb shards — just file metadata, fast
    if let Some(ref d) = paths.duckdb_dir {
        let bytes = scan_duckdb_dir(d);
        stats.disk_duckdb_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
    }
    // failures dir — small (one segment file), fast
    if let Some(ref d) = paths.failures_dir {
        let (_, bytes) = scan_dir(d);
        stats.disk_failures_mb.store(bytes_to_mb(bytes), Ordering::Relaxed);
    }
    // row counts via DuckDB (safe — writers are closed)
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

/// Expand `~` in a path string to the user's home directory.
pub fn expand_home(path: &str) -> PathBuf {
    if path.starts_with("~/") || path == "~" {
        let home = std::env::var("HOME").unwrap_or_else(|_| "/tmp".to_string());
        if path == "~" {
            PathBuf::from(home)
        } else {
            PathBuf::from(home).join(&path[2..])
        }
    } else {
        PathBuf::from(path)
    }
}

/// Parse the engine string into EngineType.
pub fn parse_engine(s: &str) -> Result<EngineType> {
    s.parse::<EngineType>()
        .map_err(|e| anyhow::anyhow!("invalid engine '{}': {}", s, e))
}

/// Parse the writer string into WriterType.
pub fn parse_writer(s: &str) -> Result<WriterType> {
    s.parse::<WriterType>()
        .map_err(|e| anyhow::anyhow!("invalid writer '{}': {}", s, e))
}

/// Create a result writer for the given writer type.
pub fn create_result_writer(
    writer_type: WriterType,
    output_dir: &Path,
    db_shards: usize,
    db_mem_mb: usize,
    batch_size: usize,
    num_flushers: usize,
) -> Result<Arc<dyn ResultWriter>> {
    let shards = if db_shards == 0 { 8 } else { db_shards };
    let mem_mb = if db_mem_mb == 0 { 64 } else { db_mem_mb };

    match writer_type {
        WriterType::DevNull => Ok(Arc::new(DevNullResultWriter)),
        WriterType::DuckDB => {
            let w = DuckDBResultWriter::new(
                &output_dir.to_string_lossy(),
                shards,
                mem_mb,
                batch_size,
            )?;
            Ok(Arc::new(w))
        }
        WriterType::Parquet => {
            let w = ParquetResultWriter::new(output_dir, batch_size)?;
            Ok(Arc::new(w))
        }
        WriterType::Binary => {
            let seg_dir = output_dir.join("results");
            let drain = BinDrainConfig {
                duckdb_dir: output_dir.to_path_buf(),
                num_shards: shards,
                mem_mb,
                batch_size,
            };
            let w = BinaryResultWriter::new_with_drain(
                &seg_dir,
                65536,
                64,
                num_flushers.max(1),
                drain,
            )?;
            Ok(Arc::new(w))
        }
    }
}

/// Create a failure writer for the given writer type.
///
/// When `no_retry` is true, the binary writer skips draining into `failed.duckdb`
/// since pass-2 retry seeds will never be loaded from it.
pub fn create_failure_writer(
    writer_type: WriterType,
    failed_db_path: &str,
    output_dir: &Path,
    db_mem_mb: usize,
    batch_size: usize,
    no_retry: bool,
) -> Result<Arc<dyn FailureWriter>> {
    let mem_mb = if db_mem_mb == 0 { 64 } else { db_mem_mb };

    match writer_type {
        WriterType::DevNull => Ok(Arc::new(DevNullFailureWriter)),
        WriterType::DuckDB => {
            let w = DuckDBFailureWriter::new(failed_db_path, mem_mb, batch_size)?;
            Ok(Arc::new(w))
        }
        WriterType::Parquet => {
            let failures_dir = output_dir.join("failures");
            let w = ParquetFailureWriter::new(&failures_dir, batch_size)?;
            Ok(Arc::new(w))
        }
        WriterType::Binary => {
            let failures_dir = output_dir.join("failures");
            if no_retry {
                // Pass-2 retry is disabled — no need to drain into failed.duckdb.
                // This avoids DuckDB checkpoint latency on potentially large accumulated files.
                let w = BinaryFailureWriter::new(&failures_dir, 65536, 64)?;
                Ok(Arc::new(w))
            } else {
                let drain = BinFailureDrainConfig {
                    db_path: PathBuf::from(failed_db_path),
                    mem_mb,
                    batch_size,
                };
                let w = BinaryFailureWriter::with_drain(&failures_dir, drain)?;
                Ok(Arc::new(w))
            }
        }
    }
}

/// Streaming variant of `CrawlJobParams`.
///
/// Seeds arrive via a pre-created channel + a total-count hint (from COUNT(*));
/// no `Vec<SeedURL>` is materialised. Used by the CC recrawl command to avoid
/// loading all 15 M rows into memory at once.
pub struct CrawlJobParamsStreaming {
    pub title: String,
    /// Channel of seeds from the streaming loader.
    pub seed_rx: async_channel::Receiver<crawler_lib::types::SeedURL>,
    /// Total seed count (hint for TUI progress %; 0 = unknown).
    pub seed_count: u64,
    pub output_dir: PathBuf,
    pub engine: String,
    pub writer: String,
    pub workers: usize,
    pub inner_n: usize,
    pub timeout_ms: u64,
    pub retry_timeout_ms: u64,
    pub no_retry: bool,
    pub domain_dead_probe: usize,
    pub domain_stall_ratio: usize,
    pub db_shards: usize,
    pub db_mem_mb: usize,
    pub no_tui: bool,
    pub body_store_dir: Option<String>,
    pub gui: bool,
    pub gui_port: u16,
    pub flusher_threads: usize,
    /// TCP connect timeout in ms (0 = use overall timeout).
    pub connect_timeout_ms: u64,
    /// Pass-2 domain stall ratio (0 = disabled, prevents false negatives).
    pub pass2_stall_ratio: usize,
    /// Pass-2 worker count override (0 = use pass-1 workers).
    pub pass2_workers: usize,
}

/// Shared crawl job configuration. Both HN and CC commands build this,
/// then call `run_crawl_job()` to execute the two-pass crawl.
pub struct CrawlJobParams {
    pub title: String,
    pub seeds: Vec<SeedURL>,
    pub output_dir: PathBuf,
    pub engine: String,
    pub writer: String,
    pub workers: usize,
    pub inner_n: usize,
    pub timeout_ms: u64,
    pub retry_timeout_ms: u64,
    pub no_retry: bool,
    pub domain_dead_probe: usize,
    pub domain_stall_ratio: usize,
    pub db_shards: usize,
    pub db_mem_mb: usize,
    pub no_tui: bool,
    /// Optional body store directory. When set, HTML bodies are stored in a
    /// content-addressable store (SHA-256, gzip) and body_cid is populated.
    pub body_store_dir: Option<String>,
    /// Enable web GUI dashboard (disables TUI).
    pub gui: bool,
    /// GUI server port (default 9111).
    pub gui_port: u16,
    /// Binary writer flusher thread count (0 = use Config.num_flushers from auto_config).
    pub flusher_threads: usize,
    /// TCP connect timeout in ms (0 = use overall timeout).
    pub connect_timeout_ms: u64,
    /// Pass-2 domain stall ratio (0 = disabled, prevents false negatives).
    pub pass2_stall_ratio: usize,
    /// Pass-2 worker count override (0 = use pass-1 workers).
    pub pass2_workers: usize,
}

/// Run a two-pass crawl job with TUI, writers, retry logic, and summary.
pub async fn run_crawl_job(params: CrawlJobParams) -> Result<()> {
    let output_dir = &params.output_dir;
    let output_dir_str = output_dir.to_string_lossy().to_string();

    let engine_type = parse_engine(&params.engine)?;
    let writer_type = parse_writer(&params.writer)?;

    let failed_db_path = output_dir.join("failed.duckdb");
    let failed_db_str = failed_db_path.to_string_lossy().to_string();

    let live_stats = Arc::new(Stats::new());

    let mut cfg = Config::default();
    cfg.workers = params.workers;
    cfg.inner_n = params.inner_n;
    cfg.timeout = Duration::from_millis(params.timeout_ms);
    cfg.retry_timeout = Duration::from_millis(params.retry_timeout_ms);
    cfg.no_retry = params.no_retry;
    cfg.domain_dead_probe = params.domain_dead_probe;
    cfg.domain_stall_ratio = params.domain_stall_ratio;
    cfg.engine = engine_type;
    cfg.writer = writer_type;
    cfg.db_shards = params.db_shards;
    cfg.db_mem_mb = params.db_mem_mb;
    cfg.output_dir = output_dir_str.clone();
    cfg.failed_db_path = failed_db_str.clone();
    cfg.live_stats = Some(live_stats.clone());
    if params.flusher_threads > 0 {
        cfg.num_flushers = params.flusher_threads;
    }
    if params.connect_timeout_ms > 0 {
        cfg.connect_timeout = Duration::from_millis(params.connect_timeout_ms);
    }
    cfg.pass2_stall_ratio = params.pass2_stall_ratio;
    if params.pass2_workers > 0 {
        cfg.pass2_workers = params.pass2_workers;
    }
    if let Some(ref dir) = params.body_store_dir {
        let resolved = expand_home(dir);
        let store = AsyncBodyStore::new(&resolved)?;
        cfg.body_store = Some(Arc::new(store));
        println!("Body store: {}", resolved.display());
    }

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
        // Only track failed.duckdb when retry is enabled — avoids slow COUNT(*) on
        // the large accumulated file when --no-retry skips the DuckDB drain entirely.
        failed_db: if params.no_retry { None } else { Some(failed_db_path.clone()) },
        bodies_dir: params.body_store_dir.as_deref().map(expand_home),
    };
    spawn_disk_sampler(live_stats.clone(), disk_paths.clone());

    println!(
        "Config: engine={} writer={} workers={} inner_n={} timeout={}ms retry_timeout={}ms",
        cfg.engine,
        cfg.writer,
        if cfg.workers == 0 { "auto".to_string() } else { cfg.workers.to_string() },
        if cfg.inner_n == 0 { "auto".to_string() } else { cfg.inner_n.to_string() },
        cfg.timeout.as_millis(),
        cfg.retry_timeout.as_millis(),
    );

    // Create result writer
    let result_writer: Arc<dyn ResultWriter> = create_result_writer(
        writer_type,
        output_dir,
        cfg.db_shards,
        cfg.db_mem_mb,
        cfg.batch_size,
        cfg.num_flushers,
    )?;

    // Failure writer factory (fresh per pass for DuckDB lock safety)
    let failed_db_str2 = failed_db_str.clone();
    let output_dir2 = output_dir.clone();
    let mem_mb = cfg.db_mem_mb;
    let batch_size = cfg.batch_size;
    let no_retry = params.no_retry;

    let open_failure_writer: Box<dyn Fn() -> Result<Arc<dyn FailureWriter>>> =
        Box::new(move || {
            create_failure_writer(writer_type, &failed_db_str2, &output_dir2, mem_mb, batch_size, no_retry)
        });

    // Retry seed loader
    let failed_db_for_retry = failed_db_str.clone();
    let load_retry: Option<Box<dyn Fn(chrono::NaiveDateTime) -> Result<Vec<SeedURL>>>> =
        if !params.no_retry {
            Some(Box::new(move |since| {
                crawler_lib::seed::load_retry_seeds(&failed_db_for_retry, since)
            }))
        } else {
            None
        };

    println!("Starting crawl job...");
    let job_start = std::time::Instant::now();

    let workers_str = if cfg.workers == 0 {
        "auto".to_string()
    } else {
        cfg.workers.to_string()
    };

    // GUI server (mutually exclusive with TUI)
    if params.gui {
        let gui_cfg = gui::GuiConfig {
            title: params.title.clone(),
            engine: cfg.engine.to_string(),
            writer: cfg.writer.to_string(),
            workers: workers_str.clone(),
            timeout_ms: cfg.timeout.as_millis() as u64,
            retry_timeout_ms: cfg.retry_timeout.as_millis() as u64,
            no_retry: cfg.no_retry,
        };
        match gui::spawn(live_stats.clone(), gui_cfg, params.gui_port).await {
            Ok(addr) => {
                let host = hostname::get()
                    .map(|h| h.to_string_lossy().to_string())
                    .unwrap_or_else(|_| "localhost".to_string());
                println!("Dashboard: http://{}:{}", host, addr.port());
            }
            Err(e) => {
                eprintln!("GUI server failed to start: {e}");
            }
        }
    }

    // TUI (only if not GUI and not no_tui)
    let tui_handle = if !params.gui && !params.no_tui {
        let tui_cfg = tui::TuiConfig {
            title: params.title,
            engine: cfg.engine.to_string(),
            writer: cfg.writer.to_string(),
            workers: workers_str,
            timeout_ms: cfg.timeout.as_millis() as u64,
        };
        tui::spawn(live_stats.clone(), tui_cfg)
    } else {
        None
    };

    // Clone body store handle before cfg is moved into run_job, so we can close it after.
    let body_store_handle = cfg.body_store.clone();

    // Convert Vec<SeedURL> to a closed receiver — enables the streaming engine interface.
    let (seed_rx, seed_count) = vec_to_receiver(params.seeds);

    // Run job
    let job_result = run_job(
        seed_rx,
        seed_count,
        cfg,
        result_writer.clone(),
        open_failure_writer.as_ref(),
        load_retry
            .as_ref()
            .map(|f| f.as_ref() as &dyn Fn(chrono::NaiveDateTime) -> Result<Vec<SeedURL>>),
    )
    .await?;

    let total_elapsed = job_start.elapsed();
    // Flush pending async body writes before closing the result writer.
    if let Some(ref store) = body_store_handle {
        store.close()?;
    }
    result_writer.close()?;
    finalize_disk_stats(&live_stats, &disk_paths);

    if let Some(h) = tui_handle {
        h.stop_and_join();
    }

    print_summary(
        &job_result.pass1,
        job_result.pass2.as_ref(),
        &job_result.total,
        job_result.workers,
    );

    println!("Wall time: {}", format_duration(total_elapsed));

    if params.gui {
        println!("Crawl complete. Dashboard available for 30s...");
        tokio::time::sleep(Duration::from_secs(30)).await;
    }

    Ok(())
}

/// Run a two-pass crawl job from a streaming seed channel.
///
/// Identical to `run_crawl_job` but takes `CrawlJobParamsStreaming` — seeds arrive
/// via `seed_rx` (pre-created by `stream_seeds_cc_parquet_async`) so no full
/// `Vec<SeedURL>` is held in memory.
pub async fn run_crawl_job_streaming(params: CrawlJobParamsStreaming) -> Result<()> {
    let output_dir = &params.output_dir;
    let output_dir_str = output_dir.to_string_lossy().to_string();

    let engine_type = parse_engine(&params.engine)?;
    let writer_type = parse_writer(&params.writer)?;

    let failed_db_path = output_dir.join("failed.duckdb");
    let failed_db_str = failed_db_path.to_string_lossy().to_string();

    let live_stats = Arc::new(Stats::new());

    let mut cfg = Config::default();
    cfg.workers = params.workers;
    cfg.inner_n = params.inner_n;
    cfg.timeout = Duration::from_millis(params.timeout_ms);
    cfg.retry_timeout = Duration::from_millis(params.retry_timeout_ms);
    cfg.no_retry = params.no_retry;
    cfg.domain_dead_probe = params.domain_dead_probe;
    cfg.domain_stall_ratio = params.domain_stall_ratio;
    cfg.engine = engine_type;
    cfg.writer = writer_type;
    cfg.db_shards = params.db_shards;
    cfg.db_mem_mb = params.db_mem_mb;
    cfg.output_dir = output_dir_str.clone();
    cfg.failed_db_path = failed_db_str.clone();
    cfg.live_stats = Some(live_stats.clone());
    if params.flusher_threads > 0 {
        cfg.num_flushers = params.flusher_threads;
    }
    if params.connect_timeout_ms > 0 {
        cfg.connect_timeout = Duration::from_millis(params.connect_timeout_ms);
    }
    cfg.pass2_stall_ratio = params.pass2_stall_ratio;
    if params.pass2_workers > 0 {
        cfg.pass2_workers = params.pass2_workers;
    }
    if let Some(ref dir) = params.body_store_dir {
        let resolved = expand_home(dir);
        let store = AsyncBodyStore::new(&resolved)?;
        cfg.body_store = Some(Arc::new(store));
        println!("Body store: {}", resolved.display());
    }

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
        failed_db: if params.no_retry { None } else { Some(failed_db_path.clone()) },
        bodies_dir: params.body_store_dir.as_deref().map(expand_home),
    };
    spawn_disk_sampler(live_stats.clone(), disk_paths.clone());

    println!(
        "Config: engine={} writer={} workers={} inner_n={} timeout={}ms retry_timeout={}ms",
        cfg.engine,
        cfg.writer,
        if cfg.workers == 0 { "auto".to_string() } else { cfg.workers.to_string() },
        if cfg.inner_n == 0 { "auto".to_string() } else { cfg.inner_n.to_string() },
        cfg.timeout.as_millis(),
        cfg.retry_timeout.as_millis(),
    );

    let result_writer: Arc<dyn ResultWriter> = create_result_writer(
        writer_type,
        output_dir,
        cfg.db_shards,
        cfg.db_mem_mb,
        cfg.batch_size,
        cfg.num_flushers,
    )?;

    let failed_db_str2 = failed_db_str.clone();
    let output_dir2 = output_dir.clone();
    let mem_mb = cfg.db_mem_mb;
    let batch_size = cfg.batch_size;
    let no_retry = params.no_retry;

    let open_failure_writer: Box<dyn Fn() -> Result<Arc<dyn FailureWriter>>> =
        Box::new(move || {
            create_failure_writer(writer_type, &failed_db_str2, &output_dir2, mem_mb, batch_size, no_retry)
        });

    let failed_db_for_retry = failed_db_str.clone();
    let load_retry: Option<Box<dyn Fn(chrono::NaiveDateTime) -> Result<Vec<crawler_lib::types::SeedURL>>>> =
        if !params.no_retry {
            Some(Box::new(move |since| {
                crawler_lib::seed::load_retry_seeds(&failed_db_for_retry, since)
            }))
        } else {
            None
        };

    println!("Starting crawl job...");
    let job_start = std::time::Instant::now();

    let workers_str = if cfg.workers == 0 {
        "auto".to_string()
    } else {
        cfg.workers.to_string()
    };

    if params.gui {
        let gui_cfg = gui::GuiConfig {
            title: params.title.clone(),
            engine: cfg.engine.to_string(),
            writer: cfg.writer.to_string(),
            workers: workers_str.clone(),
            timeout_ms: cfg.timeout.as_millis() as u64,
            retry_timeout_ms: cfg.retry_timeout.as_millis() as u64,
            no_retry: cfg.no_retry,
        };
        match gui::spawn(live_stats.clone(), gui_cfg, params.gui_port).await {
            Ok(addr) => {
                let host = hostname::get()
                    .map(|h| h.to_string_lossy().to_string())
                    .unwrap_or_else(|_| "localhost".to_string());
                println!("Dashboard: http://{}:{}", host, addr.port());
            }
            Err(e) => {
                eprintln!("GUI server failed to start: {e}");
            }
        }
    }

    let tui_handle = if !params.gui && !params.no_tui {
        let tui_cfg = tui::TuiConfig {
            title: params.title,
            engine: cfg.engine.to_string(),
            writer: cfg.writer.to_string(),
            workers: workers_str,
            timeout_ms: cfg.timeout.as_millis() as u64,
        };
        tui::spawn(live_stats.clone(), tui_cfg)
    } else {
        None
    };

    let body_store_handle = cfg.body_store.clone();

    let job_result = run_job(
        params.seed_rx,
        params.seed_count,
        cfg,
        result_writer.clone(),
        open_failure_writer.as_ref(),
        load_retry
            .as_ref()
            .map(|f| f.as_ref() as &dyn Fn(chrono::NaiveDateTime) -> Result<Vec<crawler_lib::types::SeedURL>>),
    )
    .await?;

    let total_elapsed = job_start.elapsed();
    if let Some(ref store) = body_store_handle {
        store.close()?;
    }
    result_writer.close()?;
    finalize_disk_stats(&live_stats, &disk_paths);

    if let Some(h) = tui_handle {
        h.stop_and_join();
    }

    print_summary(
        &job_result.pass1,
        job_result.pass2.as_ref(),
        &job_result.total,
        job_result.workers,
    );

    println!("Wall time: {}", format_duration(total_elapsed));

    if params.gui {
        println!("Crawl complete. Dashboard available for 30s...");
        tokio::time::sleep(Duration::from_secs(30)).await;
    }

    Ok(())
}
