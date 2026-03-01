use anyhow::Result;
use clap::Args;

use crawler_lib::seed::{load_seeds_duckdb, load_seeds_parquet};

use crate::common::{expand_home, run_crawl_job, CrawlJobParams};

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

pub async fn run_recrawl(args: RecrawlArgs) -> Result<()> {
    // 1. Load seeds
    let seed_resolved = expand_home(&args.seed);
    let seed_path = seed_resolved.to_string_lossy().to_string();
    let is_parquet = seed_path.ends_with(".parquet") || seed_path.ends_with(".parq");

    println!("Loading seeds from: {}", seed_path);
    let seeds = if is_parquet {
        load_seeds_parquet(&seed_path, args.limit)?
    } else {
        load_seeds_duckdb(&seed_path, args.limit)?
    };

    if seeds.is_empty() {
        println!("No seeds found, exiting.");
        return Ok(());
    }
    println!("Loaded {} seeds", seeds.len());

    // 2. Resolve output directory
    let output_dir = if args.output.is_empty() {
        expand_home("~/data/hn/results")
    } else {
        expand_home(&args.output)
    };
    println!("Output directory: {}", output_dir.display());

    // 3. Run crawl job
    run_crawl_job(CrawlJobParams {
        title: "HN Recrawl".to_string(),
        seeds,
        output_dir,
        engine: args.engine,
        writer: args.writer,
        workers: args.workers,
        inner_n: args.inner_n,
        timeout_ms: args.timeout,
        retry_timeout_ms: args.retry_timeout,
        no_retry: args.no_retry,
        domain_dead_probe: args.domain_dead_probe,
        domain_stall_ratio: args.domain_stall_ratio,
        db_shards: args.db_shards,
        db_mem_mb: args.db_mem_mb,
        no_tui: args.no_tui,
    })
    .await
}
