use crate::stats::Stats;
use std::sync::Arc;
use std::time::Duration;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EngineType {
    Reqwest,
    Hyper,
}

impl std::fmt::Display for EngineType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            EngineType::Reqwest => write!(f, "reqwest"),
            EngineType::Hyper => write!(f, "hyper"),
        }
    }
}

impl std::str::FromStr for EngineType {
    type Err = String;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "reqwest" => Ok(EngineType::Reqwest),
            "hyper" => Ok(EngineType::Hyper),
            _ => Err(format!("unknown engine: {}", s)),
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum WriterType {
    DuckDB,
    Parquet,
    Binary,
    DevNull,
}

impl std::fmt::Display for WriterType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            WriterType::DuckDB => write!(f, "duckdb"),
            WriterType::Parquet => write!(f, "parquet"),
            WriterType::Binary => write!(f, "binary"),
            WriterType::DevNull => write!(f, "devnull"),
        }
    }
}

impl std::str::FromStr for WriterType {
    type Err = String;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "duckdb" => Ok(WriterType::DuckDB),
            "parquet" => Ok(WriterType::Parquet),
            "binary" => Ok(WriterType::Binary),
            "devnull" => Ok(WriterType::DevNull),
            _ => Err(format!("unknown writer: {}", s)),
        }
    }
}

#[derive(Debug, Clone)]
pub struct Config {
    // Concurrency
    pub workers: usize,    // 0 = auto
    pub inner_n: usize,    // concurrent fetchers per domain (0 = auto)

    // Timeouts
    pub timeout: Duration,          // per-request HTTP timeout
    pub domain_timeout_ms: i64,     // ms; 0=disabled, <0=adaptive
    pub adaptive_timeout_max: Duration,

    // Domain management
    pub domain_fail_threshold: usize,
    pub domain_dead_probe: usize,
    pub domain_stall_ratio: usize,
    pub disable_adaptive_timeout: bool,

    // Engine
    pub engine: EngineType,

    // Writer
    pub writer: WriterType,
    pub batch_size: usize,
    pub db_shards: usize, // 0 = auto
    pub db_mem_mb: usize, // 0 = auto

    // Retry
    pub retry_timeout: Duration,
    pub no_retry: bool,
    pub pass2_workers: usize,

    // Network
    pub max_body_bytes: usize,

    // Paths
    pub output_dir: String,
    pub failed_db_path: String,

    /// Optional shared stats for live TUI display.
    /// If Some, the engine uses this Arc instead of creating a new Stats.
    /// The caller (CLI) creates Arc<Stats>, passes here, and reads from it in a TUI thread.
    #[allow(clippy::type_complexity)]
    pub live_stats: Option<Arc<Stats>>,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            workers: 0,
            inner_n: 0,
            timeout: Duration::from_millis(1000),
            domain_timeout_ms: -1, // adaptive
            adaptive_timeout_max: Duration::from_secs(600),
            domain_fail_threshold: 3,
            domain_dead_probe: 3,
            domain_stall_ratio: 5,
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
            live_stats: None,
        }
    }
}

// --- SysInfo and auto-config ---

#[derive(Debug, Clone)]
pub struct SysInfo {
    pub cpu_count: usize,
    pub mem_total_mb: u64,
    pub mem_available_mb: u64,
    pub fd_soft_limit: u64,
}

impl SysInfo {
    pub fn gather() -> Self {
        let sys = sysinfo::System::new_all();
        let cpu_count = sys.cpus().len().max(1);
        let mem_total_mb = sys.total_memory() / (1024 * 1024);
        let mem_available_mb = sys.available_memory() / (1024 * 1024);

        #[cfg(unix)]
        let fd_soft_limit = {
            let mut rlim = libc::rlimit {
                rlim_cur: 0,
                rlim_max: 0,
            };
            if unsafe { libc::getrlimit(libc::RLIMIT_NOFILE, &mut rlim) } == 0 {
                rlim.rlim_cur as u64
            } else {
                1024u64
            }
        };
        #[cfg(not(unix))]
        let fd_soft_limit = 1024u64;

        Self {
            cpu_count,
            mem_total_mb,
            mem_available_mb,
            fd_soft_limit,
        }
    }
}

fn clamp(val: usize, min_v: usize, max_v: usize) -> usize {
    val.max(min_v).min(max_v)
}

/// Auto-configure workers and inner_n based on hardware.
/// Matches Go's AutoConfigKeepAlive formula.
pub fn auto_config(si: &SysInfo, full_body: bool) -> Config {
    let body_kb: usize = if full_body { 256 } else { 4 };
    let avail_kb = (si.mem_available_mb as usize) * 1024;

    let inner_n_min = 4usize;
    let denom1 = (inner_n_min * body_kb / 4).max(1);
    let denom2 = (inner_n_min * body_kb).max(1);
    let w_mem_uncapped = std::cmp::min(
        avail_kb * 70 / 100 / denom1,
        avail_kb * 80 / 100 / denom2,
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

    let denom3 = (inner_n * body_kb / 4).max(1);
    let denom4 = (inner_n * body_kb).max(1);
    let w_mem = std::cmp::min(
        avail_kb * 70 / 100 / denom3,
        avail_kb * 80 / 100 / denom4,
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
