# Rust Crawler — High-Throughput Multi-Domain Recrawler

**Date**: 2026-03-01
**Location**: `tools/crawler/`
**Target**: 2000+ pages/s minimum, 5000+ stretch (server2: 12GB RAM, 8 CPUs)

## Overview

A custom Rust recrawler replacing spider-rs (single-site crawler, wrong fit) with a
domain-grouped batch architecture matching `pkg/crawl`'s keepalive engine. Two HTTP
engines (reqwest, hyper+rustls) benchmarked head-to-head. Three write modes (duckdb,
parquet, binary) benchmarked for throughput. Generic library for HN and CC recrawl.

## Architecture

```
tools/crawler/
├── Cargo.toml              # workspace root
├── Cargo.lock
├── crawler-lib/            # core library crate
│   ├── Cargo.toml
│   └── src/
│       ├── lib.rs
│       ├── config.rs       # Config, AutoConfig, SysInfo
│       ├── types.rs        # SeedURL, CrawlResult, FailedURL, FailedDomain
│       ├── seed.rs         # DuckDB + parquet seed loading
│       ├── domain.rs       # domain grouping, sorting, batching
│       ├── stats.rs        # Stats (atomic counters), AdaptiveTimeout (lock-free histogram)
│       ├── job.rs          # RunJob: two-pass orchestration
│       ├── ua.rs           # browser User-Agent rotation pool
│       ├── engine/
│       │   ├── mod.rs      # Engine trait + EngineConfig
│       │   ├── reqwest.rs  # reqwest-based engine (connection pooling built-in)
│       │   └── hyper.rs    # hyper + hyper-rustls (manual connection management)
│       └── writer/
│           ├── mod.rs      # ResultWriter + FailureWriter traits
│           ├── duckdb.rs   # sharded DuckDB via appender API
│           ├── parquet.rs  # Arrow RecordBatch → Parquet files
│           ├── binary.rs   # binary segment files → deferred DuckDB drain
│           └── devnull.rs  # no-op (benchmarking)
│
├── crawler-cli/            # binary crate
│   ├── Cargo.toml
│   └── src/
│       ├── main.rs         # clap v4 entry point
│       ├── hn.rs           # `crawler hn recrawl` subcommand
│       ├── cc.rs           # `crawler cc recrawl` subcommand (stub, extend later)
│       └── display.rs      # live progress: ok/total, avg/peak rps, slow domains
│
├── Dockerfile              # Ubuntu 24.04, x86-64-v3 (AVX2)
└── Makefile                # build, deploy, benchmark targets
```

## Core Types

```rust
// types.rs
pub struct SeedURL {
    pub url: String,
    pub domain: String,
}

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
    pub crawled_at: chrono::NaiveDateTime,
    pub error: String,
    pub body: String,           // empty by default (overflow block fix)
}

pub struct FailedURL {
    pub url: String,
    pub domain: String,
    pub reason: String,         // http_timeout, dns_timeout, domain_killed, etc.
    pub error: String,
    pub status_code: u16,
    pub fetch_time_ms: i64,
    pub detected_at: chrono::NaiveDateTime,
}

pub struct FailedDomain {
    pub domain: String,
    pub reason: String,         // dns_nxdomain, http_timeout_killed, http_refused
    pub error: String,
    pub url_count: i64,
    pub detected_at: chrono::NaiveDateTime,
}

pub struct Stats {
    pub ok: AtomicU64,
    pub failed: AtomicU64,
    pub timeout: AtomicU64,
    pub skipped: AtomicU64,
    pub bytes_downloaded: AtomicU64,
    pub start: Instant,
    pub peak_rps: AtomicU64,
}
```

## Engine Trait

```rust
// engine/mod.rs
#[async_trait]
pub trait Engine: Send + Sync {
    async fn run(
        &self,
        seeds: Vec<SeedURL>,
        cfg: &Config,
        result_tx: mpsc::Sender<CrawlResult>,
        failed_tx: mpsc::Sender<FailedURL>,
    ) -> Stats;
}
```

Both engines implement the same domain-grouped processing:

1. Sort seeds by domain
2. Fan out: N worker tasks, each picks a domain batch from a work channel
3. Per domain: spawn `inner_n` concurrent fetchers sharing one connection pool
4. Per-domain adaptive timeout, dead-domain probing, stall-ratio abandonment
5. Results/failures sent via bounded mpsc channels (back-pressure)

### Engine 1: reqwest

- `reqwest::Client` with `pool_max_idle_per_host(inner_n)`
- Built-in redirect following, compression, cookie handling
- Simple: ~200 lines for the full engine

### Engine 2: hyper + rustls

- `hyper::Client` with `hyper_rustls::HttpsConnectorBuilder`
- Manual redirect handling (loop with 301/302/307/308 detection)
- Manual connection pool tuning via `hyper::pool::Config`
- More control over TCP keepalive, TLS session resumption
- Expected: lower overhead per request, higher peak throughput

## Domain Management

```rust
// domain.rs
pub struct DomainBatch {
    pub domain: String,
    pub urls: Vec<String>,
}

/// Sort seeds by domain, yield contiguous batches
pub fn group_by_domain(seeds: Vec<SeedURL>) -> Vec<DomainBatch>;

/// Adaptive timeout tracker (lock-free)
pub struct AdaptiveTimeout {
    histogram: [AtomicU64; 8],  // 100ms, 250ms, 500ms, 1s, 2s, 5s, 10s, 15s
    sample_count: AtomicU64,
}

impl AdaptiveTimeout {
    pub fn record(&self, latency: Duration);
    pub fn p95(&self) -> Duration;
    pub fn effective_timeout(&self, max: Duration) -> Duration; // min(p95*2, max)
}

/// Per-domain state during crawl
struct DomainState {
    successes: u64,
    timeouts: u64,
    consecutive_timeouts: u64,
}

impl DomainState {
    fn should_abandon(&self, cfg: &Config) -> bool;  // dead probe + stall ratio
}
```

**Abandonment rules** (matching Go):
- `domain_fail_threshold`: N rounds of all-timeout → abandon
- `domain_dead_probe`: N consecutive timeouts with 0 success → abandon (pass-2)
- `domain_stall_ratio`: timeout >= success × ratio → abandon

## Config

```rust
// config.rs
pub struct Config {
    // Concurrency
    pub workers: usize,             // 0 = auto
    pub inner_n: usize,             // concurrent fetchers per domain (0 = auto)

    // Timeouts
    pub timeout: Duration,          // per-request HTTP timeout
    pub domain_timeout: Duration,   // per-domain deadline (0=disabled, negative=adaptive)
    pub adaptive_timeout_max: Duration,

    // Domain management
    pub domain_fail_threshold: usize,
    pub domain_dead_probe: usize,
    pub domain_stall_ratio: usize,
    pub disable_adaptive_timeout: bool,

    // Engine
    pub engine: EngineType,         // Reqwest | Hyper

    // Writer
    pub writer: WriterType,         // DuckDB | Parquet | Binary | DevNull
    pub batch_size: usize,
    pub db_shards: usize,           // 0 = auto
    pub db_mem_mb: usize,           // 0 = auto

    // Retry
    pub retry_timeout: Duration,
    pub no_retry: bool,
    pub pass2_workers: usize,

    // Network
    pub user_agents: Vec<String>,   // browser UA rotation pool
    pub max_body_bytes: usize,      // 256KB default
}
```

**AutoConfig** (matching Go's `AutoConfigKeepAlive`):
```rust
pub fn auto_config(sys: &SysInfo, full_body: bool) -> Config;
// - workers = clamp(avail_mem * 0.7 / (inner_n * body_kb), fd_limit / (inner_n*2), 10000)
// - inner_n = clamp(cpus * 2, 4, 16)
// - db_shards = clamp(cpus * 2, 4, 16)
// - db_mem_mb = avail_mb * 15% / shards
```

## Writer Traits

```rust
// writer/mod.rs
#[async_trait]
pub trait ResultWriter: Send + Sync {
    async fn write(&self, result: CrawlResult) -> Result<()>;
    async fn flush(&self) -> Result<()>;
    async fn close(&self) -> Result<()>;
}

#[async_trait]
pub trait FailureWriter: Send + Sync {
    async fn write_url(&self, failed: FailedURL) -> Result<()>;
    async fn write_domain(&self, failed: FailedDomain) -> Result<()>;
    async fn flush(&self) -> Result<()>;
    async fn close(&self) -> Result<()>;
}
```

### Writer 1: DuckDB (sharded)

- N shards, each with dedicated `duckdb::Connection`
- Appender API for bulk inserts (`duckdb::Appender`)
- Hash URL to shard (avoid contention)
- Background flush task per shard
- Schema matches Go's ResultDB

### Writer 2: Parquet

- `arrow` + `parquet` crates
- Build `RecordBatch` from buffered results
- Write Parquet files with Snappy compression
- Row group size = batch_size
- No DuckDB dependency at write time

### Writer 3: Binary segments

- Binary-encoded result segments (bincode or custom)
- Workers → bounded channel → flusher task → segment files
- After crawl: drain segments → DuckDB (deferred, non-blocking)
- Segment size: 64MB default (auto-tuned)

### Writer 4: DevNull

- No-op, counts only (benchmarking baseline)

## Two-Pass Job Orchestration

```rust
// job.rs
pub struct JobConfig {
    pub engine: EngineType,
    pub config: Config,
    pub seed_path: String,
    pub output_dir: String,
    pub failed_db_path: String,
}

pub async fn run_job(job: JobConfig) -> JobResult {
    // 1. Load seeds from DuckDB/parquet
    // 2. Auto-config if workers=0
    // 3. Open result writer + failure writer
    // 4. Pass 1: run engine with cfg.timeout
    // 5. Close failure writer (release DuckDB lock)
    // 6. If !no_retry:
    //    a. Load retry seeds (timeout URLs since job start)
    //    b. Open failure writer 2
    //    c. Pass 2: run engine with retry_timeout,
    //       disable_adaptive_timeout=true, domain_dead_probe=2
    // 7. Merge stats (pass1 + pass2)
    // 8. Print 100% coverage summary
}
```

## Seed Loading

```rust
// seed.rs
pub fn load_seeds_duckdb(path: &str) -> Result<Vec<SeedURL>>;
pub fn load_seeds_parquet(path: &str) -> Result<Vec<SeedURL>>;

/// Load timeout URLs from failed DB for pass-2 retry
pub fn load_retry_seeds(path: &str, since: chrono::NaiveDateTime) -> Result<Vec<SeedURL>>;
```

## CLI

```
crawler hn recrawl [OPTIONS]
  --seed <PATH>              Seed DuckDB/parquet file
  --output <DIR>             Output directory [default: ~/data/hn/results/]
  --engine <ENGINE>          reqwest | hyper [default: reqwest]
  --writer <WRITER>          duckdb | parquet | binary | devnull [default: binary]
  --workers <N>              Worker count (0=auto) [default: 0]
  --inner-n <N>              Per-domain concurrency (0=auto) [default: 0]
  --timeout <MS>             Pass-1 timeout [default: 1000]
  --retry-timeout <MS>       Pass-2 timeout [default: 15000]
  --no-retry                 Skip pass-2
  --domain-dead-probe <N>    [default: 10]
  --domain-stall-ratio <N>   [default: 20]
  --limit <N>                Max seeds to process (0=all)
  --db-shards <N>            DuckDB shard count (0=auto)
  --db-mem-mb <N>            DuckDB memory per shard (0=auto)

crawler cc recrawl [OPTIONS]   # stub, extend later
  (same flags + CC-specific: --file, --sample, --remote)

crawler benchmark [OPTIONS]
  --seed <PATH>
  --limit <N>                [default: 10000]
  --engine <ENGINE>          reqwest | hyper | all
  --writer <WRITER>          duckdb | parquet | binary | devnull | all
```

## Dependencies

```toml
# crawler-lib/Cargo.toml
[dependencies]
tokio = { version = "1", features = ["full"] }
reqwest = { version = "0.12", features = ["rustls-tls", "gzip", "brotli", "deflate"] }
hyper = { version = "1", features = ["client", "http1", "http2"] }
hyper-util = { version = "0.1", features = ["client-legacy", "tokio"] }
hyper-rustls = { version = "0.27", features = ["http2"] }
rustls = "0.23"
http-body-util = "0.1"

duckdb = { version = "1.2", features = ["bundled"] }
arrow = "54"
parquet = { version = "54", features = ["snap"] }

chrono = "0.4"
url = "2"
publicsuffix = "2"             # domain extraction (registrable domain)
serde = { version = "1", features = ["derive"] }
bincode = "1"
clap = { version = "4", features = ["derive"] }
tracing = "0.1"
tracing-subscriber = "0.3"
anyhow = "1"
thiserror = "2"

# Performance
dashmap = "6"                  # concurrent hashmap (domain state)
ahash = "0.8"                 # fast hashing
crossbeam-channel = "0.5"     # lock-free bounded channels (writer pipeline)
```

## Build & Deploy

```makefile
# Makefile
build:
	cargo build --release

build-linux:
	docker build -t crawler-linux -f Dockerfile .
	docker create --name crawler-tmp crawler-linux
	docker cp crawler-tmp:/out/crawler .
	docker rm crawler-tmp

deploy: build-linux
	scp -i $(DEPLOY_KEY) crawler $(SERVER_USER)@$(SERVER_HOST):~/bin/crawler
	ssh -i $(DEPLOY_KEY) $(SERVER_USER)@$(SERVER_HOST) 'chmod +x ~/bin/crawler && ~/bin/crawler --help'

benchmark:
	ssh -i $(DEPLOY_KEY) $(SERVER_USER)@$(SERVER_HOST) \
	  '~/bin/crawler benchmark --seed ~/data/hn/seeds.duckdb --limit 10000 --engine all --writer all'
```

```dockerfile
# Dockerfile (Ubuntu 24.04, x86-64-v3)
FROM ubuntu:24.04 AS builder
RUN apt-get update && apt-get install -y curl build-essential pkg-config libssl-dev
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:$PATH"
ENV RUSTFLAGS="-C target-cpu=x86-64-v3"
WORKDIR /build
COPY . .
RUN cargo build --release
FROM scratch
COPY --from=builder /build/target/release/crawler /out/crawler
```

## Performance Strategy

### For 2000+ pages/s (minimum bar)

1. **Domain-grouped batching**: process all URLs for a domain together (connection reuse)
2. **Bounded channels**: back-pressure prevents OOM without blocking hot path
3. **Atomic stats**: no Mutex on hot counters
4. **Browser UA rotation**: avoid bot-holding (95% → 67% timeout rate fix)
5. **Adaptive P95 timeout**: kill slow domains quickly in pass-1

### For 5000+ pages/s (stretch)

6. **io_uring** (Linux): `tokio-uring` or `monoio` for async socket I/O
7. **HTTP/2 multiplexing**: single TCP connection, many concurrent streams per domain
8. **Zero-copy Bytes**: `bytes::Bytes` everywhere, no String cloning
9. **SIMD HTML parsing**: `lol_html` or custom title/description extraction
10. **Sharded connection pools**: N pools × M connections, hash domain → pool
11. **TCP_NODELAY + TCP_FASTOPEN**: reduce per-request latency
12. **Memory-mapped segment files**: mmap for binary writer segments
13. **Parquet writer with zstd**: faster than snappy at high compression

## Migration Path

1. **Phase 1**: Build crawler with reqwest engine + devnull writer → benchmark raw fetch speed
2. **Phase 2**: Add hyper engine → benchmark against reqwest
3. **Phase 3**: Add DuckDB + parquet + binary writers → benchmark write overhead
4. **Phase 4**: Deploy to servers, run against HN seeds, compare with Go version
5. **Phase 5**: Optimize winner engine for 5000+ pages/s
6. **Phase 6**: Add CC recrawl support (extend seed loading + CLI)

## Success Criteria

- [ ] Feature parity with `search hn recrawl` (seeds → crawl → results + failed DBs)
- [ ] Two-pass retry with adaptive timeout
- [ ] 2000+ pages/s on server2 with HN seeds
- [ ] All three write modes functional
- [ ] Both engines functional and benchmarked
- [ ] Deployable to server1 + server2 via Makefile
- [ ] Generic library extensible for CC recrawl
