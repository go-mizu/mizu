/// CC recrawl subcommand — stub for future extension.
///
/// Will implement Common Crawl index → recrawl pipeline using the same
/// crawler-lib engine and writer infrastructure as `hn recrawl`.
use clap::Args;

#[derive(Args, Debug)]
pub struct RecrawlArgs {
    /// Seed DuckDB or Parquet file path (or CC crawl ID)
    #[arg(long)]
    pub seed: Option<String>,

    /// CC parquet file index (e.g. "p:0" for partition 0)
    #[arg(long)]
    pub file: Option<String>,

    /// Sample N URLs from the CC index
    #[arg(long)]
    pub sample: Option<usize>,

    /// Output directory for results
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
    #[arg(long, default_value_t = 10000)]
    pub retry_timeout: u64,

    /// Skip pass-2 retry
    #[arg(long)]
    pub no_retry: bool,

    /// Limit number of seeds (0 = all)
    #[arg(long, default_value_t = 0)]
    pub limit: usize,

    /// DuckDB shard count (0 = auto)
    #[arg(long, default_value_t = 0)]
    pub db_shards: usize,

    /// DuckDB memory per shard in MB (0 = auto)
    #[arg(long, default_value_t = 0)]
    pub db_mem_mb: usize,
}

pub async fn run_recrawl(_args: RecrawlArgs) -> anyhow::Result<()> {
    // TODO: implement CC recrawl
    // - Load seed URLs from CC parquet index via DuckDB read_parquet()
    // - Use same crawler-lib::job::run_job() as HN recrawl
    // - Write to {output}/CC-MAIN-{crawl_id}/recrawl/
    anyhow::bail!("CC recrawl not yet implemented. Coming soon!");
}
