//! Shared utilities for HN and CC recrawl commands.

use anyhow::Result;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::Duration;

use crawler_lib::bodystore::BodyStore;
use crawler_lib::config::{Config, EngineType, WriterType};
use crawler_lib::job::run_job;
use crawler_lib::stats::Stats;
use crawler_lib::types::SeedURL;
use crawler_lib::writer::binary::{BinDrainConfig, BinFailureDrainConfig, BinaryFailureWriter, BinaryResultWriter};
use crawler_lib::writer::devnull::{DevNullFailureWriter, DevNullResultWriter};
use crawler_lib::writer::duckdb_writer::{DuckDBFailureWriter, DuckDBResultWriter};
use crawler_lib::writer::parquet_writer::{ParquetFailureWriter, ParquetResultWriter};
use crawler_lib::writer::{FailureWriter, ResultWriter};

use crate::display::{format_duration, print_summary};
use crate::tui;

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
            let w = BinaryResultWriter::with_drain(&seg_dir, drain)?;
            Ok(Arc::new(w))
        }
    }
}

/// Create a failure writer for the given writer type.
pub fn create_failure_writer(
    writer_type: WriterType,
    failed_db_path: &str,
    output_dir: &Path,
    db_mem_mb: usize,
    batch_size: usize,
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
    if let Some(ref dir) = params.body_store_dir {
        let resolved = expand_home(dir);
        let store = BodyStore::open(&resolved)?;
        cfg.body_store = Some(Arc::new(store));
        println!("Body store: {}", resolved.display());
    }

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
    )?;

    // Failure writer factory (fresh per pass for DuckDB lock safety)
    let failed_db_str2 = failed_db_str.clone();
    let output_dir2 = output_dir.clone();
    let mem_mb = cfg.db_mem_mb;
    let batch_size = cfg.batch_size;

    let open_failure_writer: Box<dyn Fn() -> Result<Arc<dyn FailureWriter>>> =
        Box::new(move || {
            create_failure_writer(writer_type, &failed_db_str2, &output_dir2, mem_mb, batch_size)
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

    // TUI
    let tui_handle = if !params.no_tui {
        let tui_cfg = tui::TuiConfig {
            title: params.title,
            engine: cfg.engine.to_string(),
            writer: cfg.writer.to_string(),
            workers: if cfg.workers == 0 {
                "auto".to_string()
            } else {
                cfg.workers.to_string()
            },
            timeout_ms: cfg.timeout.as_millis() as u64,
        };
        tui::spawn(live_stats.clone(), tui_cfg)
    } else {
        None
    };

    // Run job
    let job_result = run_job(
        params.seeds,
        cfg,
        result_writer.clone(),
        open_failure_writer.as_ref(),
        load_retry
            .as_ref()
            .map(|f| f.as_ref() as &dyn Fn(chrono::NaiveDateTime) -> Result<Vec<SeedURL>>),
    )
    .await?;

    let total_elapsed = job_start.elapsed();
    result_writer.close()?;

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

    Ok(())
}
