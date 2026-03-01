use anyhow::Result;
use clap::Args;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::Duration;

use crawler_lib::config::{Config, EngineType, WriterType};
use crawler_lib::job::run_job;
use crawler_lib::seed::{load_retry_seeds, load_seeds_duckdb, load_seeds_parquet};
use crawler_lib::stats::Stats;
use crawler_lib::writer::devnull::{DevNullFailureWriter, DevNullResultWriter};
use crawler_lib::writer::duckdb_writer::{DuckDBFailureWriter, DuckDBResultWriter};
use crawler_lib::writer::parquet_writer::{ParquetFailureWriter, ParquetResultWriter};
use crawler_lib::writer::binary::{BinDrainConfig, BinaryFailureWriter, BinaryResultWriter};
use crawler_lib::writer::{FailureWriter, ResultWriter};

use crate::display::{format_duration, print_summary};
use crate::tui;

#[derive(Args, Debug)]
pub struct RecrawlArgs {
    /// Seed DuckDB or Parquet file path (default: ~/data/hn/recrawl/hn_pages.duckdb)
    #[arg(long, default_value = "~/data/hn/recrawl/hn_pages.duckdb")]
    pub seed: String,

    /// Output directory for results (default: ~/data/hn/results/)
    #[arg(long, default_value = "")]
    pub output: String,

    /// HTTP engine (reqwest or hyper)
    #[arg(long, default_value = "reqwest")]
    pub engine: String,

    /// Writer mode (duckdb, parquet, binary, devnull)
    #[arg(long, default_value = "binary")]
    pub writer: String,

    /// Worker count (0 = auto)
    #[arg(long, default_value_t = 0)]
    pub workers: usize,

    /// Per-domain concurrency (0 = auto)
    #[arg(long, default_value_t = 0)]
    pub inner_n: usize,

    /// Pass-1 timeout in ms
    #[arg(long, default_value_t = 1000)]
    pub timeout: u64,

    /// Pass-2 retry timeout in ms
    #[arg(long, default_value_t = 15000)]
    pub retry_timeout: u64,

    /// Skip pass-2 retry
    #[arg(long)]
    pub no_retry: bool,

    /// Dead domain probe count (abandon after N timeouts with 0 successes)
    #[arg(long, default_value_t = 3)]
    pub domain_dead_probe: usize,

    /// Domain stall ratio (abandon if timeouts >= successes * ratio)
    #[arg(long, default_value_t = 5)]
    pub domain_stall_ratio: usize,

    /// Limit number of seeds (0 = all)
    #[arg(long, default_value_t = 0)]
    pub limit: usize,

    /// DuckDB shard count (0 = auto)
    #[arg(long, default_value_t = 0)]
    pub db_shards: usize,

    /// DuckDB memory per shard in MB (0 = auto)
    #[arg(long, default_value_t = 0)]
    pub db_mem_mb: usize,

    /// Disable TUI dashboard (text-only output)
    #[arg(long)]
    pub no_tui: bool,
}

/// Expand `~` in a path string to the user's home directory.
fn expand_home(path: &str) -> PathBuf {
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

/// Resolve the output directory: use the provided path or default to ~/data/hn/results/.
fn resolve_output_dir(output: &str) -> PathBuf {
    if output.is_empty() {
        expand_home("~/data/hn/results")
    } else {
        expand_home(output)
    }
}

/// Parse the engine string into EngineType.
fn parse_engine(s: &str) -> Result<EngineType> {
    s.parse::<EngineType>()
        .map_err(|e| anyhow::anyhow!("invalid engine '{}': {}", s, e))
}

/// Parse the writer string into WriterType.
fn parse_writer(s: &str) -> Result<WriterType> {
    s.parse::<WriterType>()
        .map_err(|e| anyhow::anyhow!("invalid writer '{}': {}", s, e))
}

pub async fn run_recrawl(args: RecrawlArgs) -> Result<()> {
    // 1. Detect seed format and load seeds
    let seed_resolved = expand_home(&args.seed);
    let seed_path = seed_resolved.to_string_lossy();
    let seed_path = seed_path.as_ref();
    let is_parquet = seed_path.ends_with(".parquet") || seed_path.ends_with(".parq");

    println!("Loading seeds from: {}", seed_path);
    let seeds = if is_parquet {
        load_seeds_parquet(seed_path, args.limit)?
    } else {
        load_seeds_duckdb(seed_path, args.limit)?
    };

    if seeds.is_empty() {
        println!("No seeds found, exiting.");
        return Ok(());
    }
    println!("Loaded {} seeds", seeds.len());

    // 2. Resolve output directory
    let output_dir = resolve_output_dir(&args.output);
    let output_dir_str = output_dir.to_string_lossy().to_string();
    println!("Output directory: {}", output_dir_str);

    // 3. Build Config from args
    let engine_type = parse_engine(&args.engine)?;
    let writer_type = parse_writer(&args.writer)?;

    let failed_db_path = output_dir.join("failed.duckdb");
    let failed_db_str = failed_db_path.to_string_lossy().to_string();

    // Shared live-stats for TUI. Created here so TUI can read it before job starts.
    let live_stats = Arc::new(Stats::new());

    let mut cfg = Config::default();
    cfg.workers = args.workers;
    cfg.inner_n = args.inner_n;
    cfg.timeout = Duration::from_millis(args.timeout);
    cfg.retry_timeout = Duration::from_millis(args.retry_timeout);
    cfg.no_retry = args.no_retry;
    cfg.domain_dead_probe = args.domain_dead_probe;
    cfg.domain_stall_ratio = args.domain_stall_ratio;
    cfg.engine = engine_type;
    cfg.writer = writer_type;
    cfg.db_shards = args.db_shards;
    cfg.db_mem_mb = args.db_mem_mb;
    cfg.output_dir = output_dir_str.clone();
    cfg.failed_db_path = failed_db_str.clone();
    cfg.live_stats = Some(live_stats.clone());

    println!(
        "Config: engine={} writer={} workers={} inner_n={} timeout={}ms retry_timeout={}ms",
        cfg.engine,
        cfg.writer,
        if cfg.workers == 0 { "auto".to_string() } else { cfg.workers.to_string() },
        if cfg.inner_n == 0 { "auto".to_string() } else { cfg.inner_n.to_string() },
        cfg.timeout.as_millis(),
        cfg.retry_timeout.as_millis(),
    );

    // 4. Create result writer (shared across both passes via Arc)
    let result_writer: Arc<dyn ResultWriter> = create_result_writer(
        writer_type,
        &output_dir,
        cfg.db_shards,
        cfg.db_mem_mb,
        cfg.batch_size,
    )?;

    // 5. Build failure writer factory closure
    // Each call creates a fresh writer (needed for two-pass since pass-1 writer
    // must be closed before pass-2 can read from the same failed DB file).
    let failed_db_str2 = failed_db_str.clone();
    let output_dir2 = output_dir.clone();
    let writer_type2 = writer_type;
    let mem_mb = cfg.db_mem_mb;
    let batch_size = cfg.batch_size;

    let open_failure_writer: Box<dyn Fn() -> Result<Arc<dyn FailureWriter>>> =
        Box::new(move || {
            create_failure_writer(
                writer_type2,
                &failed_db_str2,
                &output_dir2,
                mem_mb,
                batch_size,
            )
        });

    // 6. Build retry seed loader (for pass-2) if retry is enabled
    let failed_db_for_retry = failed_db_str.clone();
    let load_retry: Option<Box<dyn Fn(chrono::NaiveDateTime) -> Result<Vec<crawler_lib::types::SeedURL>>>> =
        if !args.no_retry {
            Some(Box::new(move |since| {
                load_retry_seeds(&failed_db_for_retry, since)
            }))
        } else {
            None
        };

    println!("Starting crawl job...");
    let job_start = std::time::Instant::now();

    // 7. Start TUI dashboard (only when stdout is a terminal and --no-tui not set).
    let tui_handle = if !args.no_tui {
        let tui_cfg = tui::TuiConfig {
            title: "HN Recrawl".to_string(),
            engine: cfg.engine.to_string(),
            writer: cfg.writer.to_string(),
            workers: if cfg.workers == 0 { "auto".to_string() } else { cfg.workers.to_string() },
            timeout_ms: cfg.timeout.as_millis() as u64,
        };
        tui::spawn(live_stats.clone(), tui_cfg)
    } else {
        None
    };

    // 8. Run job (blocks until both passes complete)
    let job_result = run_job(
        seeds,
        cfg,
        result_writer.clone(),
        open_failure_writer.as_ref(),
        load_retry
            .as_ref()
            .map(|f| f.as_ref() as &dyn Fn(chrono::NaiveDateTime) -> Result<Vec<crawler_lib::types::SeedURL>>),
    )
    .await?;

    let total_elapsed = job_start.elapsed();

    // Close result writer after job completes
    result_writer.close()?;

    // 9. Stop TUI (does final render, then restores terminal)
    if let Some(h) = tui_handle {
        h.stop_and_join();
    }

    // 10. Print final summary (now on the restored normal screen)
    print_summary(
        &job_result.pass1,
        job_result.pass2.as_ref(),
        &job_result.total,
        job_result.workers,
    );

    println!("Wall time: {}", format_duration(total_elapsed));

    Ok(())
}

/// Create a result writer for the given writer type.
fn create_result_writer(
    writer_type: WriterType,
    output_dir: &Path,
    db_shards: usize,
    db_mem_mb: usize,
    batch_size: usize,
) -> Result<Arc<dyn ResultWriter>> {
    // Resolve actual shard/mem values if auto (0).
    // The run_job auto-config will finalize them, but we need something for the writer.
    // Use safe defaults if still 0 at writer creation time.
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
            // Segments go in output_dir/results/ (fast, non-blocking during crawl).
            // On close(), they are drained to sharded DuckDB in output_dir/ and deleted.
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
fn create_failure_writer(
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
            let w = BinaryFailureWriter::with_defaults(&failures_dir)?;
            Ok(Arc::new(w))
        }
    }
}
