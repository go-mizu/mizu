//! CC (Common Crawl) recrawl subcommand.
//!
//! Implements `crawler cc recrawl --file p:N` pipeline:
//! 1. Resolve CC crawl ID (latest or explicit)
//! 2. Download manifest, resolve file selector to local parquet path
//! 3. Load seeds from parquet (filtered: warc_filename IS NOT NULL)
//! 4. Run two-pass crawl job (reuses same engine/writer/TUI as HN)

use anyhow::{bail, Context, Result};
use clap::Args;
use std::io::Read as IoRead;
use std::path::{Path, PathBuf};

use crawler_lib::seed::{load_seeds_cc_parquet, CcSeedFilter};

use crate::common::{expand_home, run_crawl_job, CrawlJobParams};

/// Base URL for Common Crawl data.
const CC_BASE_URL: &str = "https://data.commoncrawl.org";
/// URL for crawl list (latest crawl IDs).
const CC_COLLINFO_URL: &str = "https://index.commoncrawl.org/collinfo.json";

#[derive(Args, Debug)]
pub struct RecrawlArgs {
    /// Seed parquet or DuckDB file path (skips manifest resolution)
    #[arg(long)]
    pub seed: Option<String>,

    /// CC parquet file selector: N, p:N, w:N, m:N (e.g. "p:0" for first warc partition)
    #[arg(long)]
    pub file: Option<String>,

    /// CC crawl ID (default: latest from CC API)
    #[arg(long)]
    pub crawl: Option<String>,

    /// Output directory (default: ~/data/common-crawl/{crawl_id}/recrawl/{part}/)
    #[arg(long, default_value = "")]
    pub output: String,

    /// HTTP engine (reqwest, hyper, wreq)
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

    /// Filter: only seeds with these HTTP status codes (e.g. 200)
    #[arg(long, value_delimiter = ',')]
    pub status: Vec<i32>,

    /// Filter: language code (e.g. "eng")
    #[arg(long)]
    pub lang: Vec<String>,

    /// Filter: MIME type (e.g. "text/html")
    #[arg(long)]
    pub mime: Vec<String>,

    /// Body CAS store directory (default: ~/data/common-crawl/bodies).
    /// HTML bodies are stored as sha256:{hex}.gz; body_cid is populated in results.
    #[arg(long, default_value = "~/data/common-crawl/bodies")]
    pub body_store: String,

    /// Disable body CAS store (skip saving HTML bodies)
    #[arg(long)]
    pub no_body_store: bool,

    /// Enable web GUI dashboard (disables TUI)
    #[arg(long)]
    pub gui: bool,

    /// GUI server port
    #[arg(long, default_value_t = 9111)]
    pub gui_port: u16,
}

pub async fn run_recrawl(args: RecrawlArgs) -> Result<()> {
    // 1. Determine seed source: --seed (direct) or --file (CC selector)
    let (parquet_path, part_name, crawl_id) = if let Some(ref seed) = args.seed {
        // Direct path
        let p = expand_home(seed);
        let part = extract_part_name(&p.to_string_lossy());
        (p, part, args.crawl.clone().unwrap_or_default())
    } else if let Some(ref file_selector) = args.file {
        // CC file selector: resolve crawl ID → manifest → parquet path
        let crawl_id = resolve_crawl_id(args.crawl.as_deref()).await?;
        println!("Crawl ID: {}", crawl_id);
        let resolved = resolve_cc_file(&crawl_id, file_selector).await?;
        let part = extract_part_name(&resolved.to_string_lossy());
        (resolved, part, crawl_id)
    } else {
        bail!("Either --seed <path> or --file <selector> is required.\n\
               Examples:\n  \
               crawler cc recrawl --file p:0 --crawl CC-MAIN-2026-08\n  \
               crawler cc recrawl --seed ~/data/common-crawl/CC-MAIN-2026-08/index/.../part-00000.parquet");
    };

    let parquet_str = parquet_path.to_string_lossy().to_string();
    println!("Parquet file: {} ({})", parquet_str, part_name);

    // 2. Load seeds from CC parquet
    let filter = CcSeedFilter {
        status_codes: args.status.clone(),
        mime_types: args.mime.clone(),
        languages: args.lang.clone(),
    };
    println!("Loading CC seeds from parquet (warc_filename IS NOT NULL)...");
    let seeds = load_seeds_cc_parquet(&parquet_str, args.limit, &filter)?;

    if seeds.is_empty() {
        println!("No seeds found, exiting.");
        return Ok(());
    }
    println!("Loaded {} seeds", seeds.len());

    // 3. Resolve output directory
    let output_dir = if args.output.is_empty() {
        let base = expand_home("~/data/common-crawl");
        if crawl_id.is_empty() {
            base.join("recrawl").join(&part_name)
        } else {
            base.join(&crawl_id).join("recrawl").join(&part_name)
        }
    } else {
        expand_home(&args.output)
    };
    println!("Output directory: {}", output_dir.display());

    // 4. Run crawl job (reuses shared infrastructure)
    let title = if crawl_id.is_empty() {
        format!("CC Recrawl — {}", part_name)
    } else {
        format!("CC {} — {}", crawl_id, part_name)
    };

    run_crawl_job(CrawlJobParams {
        title,
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
        body_store_dir: if args.no_body_store { None } else { Some(args.body_store) },
        gui: args.gui,
        gui_port: args.gui_port,
    })
    .await
}

// ---------------------------------------------------------------------------
// Crawl ID resolution
// ---------------------------------------------------------------------------

/// Resolve crawl ID: use explicit value or fetch latest from CC API.
async fn resolve_crawl_id(explicit: Option<&str>) -> Result<String> {
    if let Some(id) = explicit {
        return Ok(id.to_string());
    }

    // Check local cache first
    let cache_path = expand_home("~/data/common-crawl/crawls.json");
    if let Ok(cached) = read_cached_crawl_id(&cache_path) {
        println!("Using cached latest crawl: {}", cached);
        return Ok(cached);
    }

    // Fetch from CC API
    println!("Fetching crawl list from {}...", CC_COLLINFO_URL);
    let client = reqwest::Client::builder()
        .timeout(std::time::Duration::from_secs(30))
        .build()?;
    let resp = client.get(CC_COLLINFO_URL).send().await?;
    let body = resp.text().await?;

    let crawls: Vec<CcCrawlInfo> = serde_json::from_str(&body)
        .context("parsing CC crawl list")?;

    if crawls.is_empty() {
        bail!("No crawls found in CC API response");
    }

    let latest = &crawls[0].id;
    println!("Latest crawl: {}", latest);

    // Cache for future use
    if let Some(parent) = cache_path.parent() {
        std::fs::create_dir_all(parent).ok();
    }
    std::fs::write(&cache_path, &body).ok();

    Ok(latest.clone())
}

#[derive(serde::Deserialize)]
struct CcCrawlInfo {
    #[serde(rename = "id")]
    id: String,
}

fn read_cached_crawl_id(path: &Path) -> Result<String> {
    let metadata = std::fs::metadata(path)?;
    // Cache valid for 24 hours
    let age = metadata
        .modified()?
        .elapsed()
        .unwrap_or(std::time::Duration::from_secs(u64::MAX));
    if age > std::time::Duration::from_secs(86400) {
        bail!("cache expired");
    }
    let data = std::fs::read_to_string(path)?;
    let crawls: Vec<CcCrawlInfo> = serde_json::from_str(&data)?;
    if crawls.is_empty() {
        bail!("empty cache");
    }
    Ok(crawls[0].id.clone())
}

// ---------------------------------------------------------------------------
// File selector resolution
// ---------------------------------------------------------------------------

/// Resolve a CC file selector (e.g. "p:0") to a local parquet path.
/// Downloads the manifest and parquet file if not cached locally.
async fn resolve_cc_file(crawl_id: &str, selector: &str) -> Result<PathBuf> {
    // Try as local file path first
    let as_path = expand_home(selector);
    if as_path.exists() && as_path.is_file() {
        println!("Using local file: {}", as_path.display());
        return Ok(as_path);
    }

    // Parse selector
    let (kind, idx) = parse_cc_file_selector(selector)?;

    // Load manifest
    let manifest = load_manifest(crawl_id).await?;

    // Filter warc subset
    let warc_files: Vec<&str> = manifest
        .iter()
        .map(|s| s.as_str())
        .filter(|p| p.contains("subset=warc/"))
        .collect();

    if warc_files.is_empty() {
        bail!("No warc subset files found in manifest for {}", crawl_id);
    }

    let remote_path = match kind {
        SelectorKind::Warc => {
            if idx >= warc_files.len() {
                bail!(
                    "warc index {} out of range (warc subset has {} files)",
                    idx,
                    warc_files.len()
                );
            }
            warc_files[idx]
        }
        SelectorKind::Manifest => {
            if idx >= manifest.len() {
                bail!(
                    "manifest index {} out of range (manifest has {} files)",
                    idx,
                    manifest.len()
                );
            }
            let path = &manifest[idx];
            if !path.contains("subset=warc/") {
                bail!(
                    "manifest index {} is not a warc subset file (got: {})",
                    idx,
                    path
                );
            }
            path.as_str()
        }
    };

    println!(
        "Resolved: {}[{}] → {}",
        match kind {
            SelectorKind::Warc => "warc",
            SelectorKind::Manifest => "manifest",
        },
        idx,
        remote_path
    );

    // Check local cache
    let local_path = local_parquet_path(crawl_id, remote_path);
    if local_path.exists() {
        let size = std::fs::metadata(&local_path)
            .map(|m| m.len())
            .unwrap_or(0);
        if size > 0 {
            println!(
                "Using cached: {} ({} MB)",
                local_path.display(),
                size / 1_048_576
            );
            return Ok(local_path);
        }
    }

    // Download
    download_parquet(remote_path, &local_path).await?;
    Ok(local_path)
}

#[derive(Debug, Clone, Copy)]
enum SelectorKind {
    Warc,
    Manifest,
}

fn parse_cc_file_selector(s: &str) -> Result<(SelectorKind, usize)> {
    // Plain integer → warc index
    if let Ok(n) = s.parse::<usize>() {
        return Ok((SelectorKind::Warc, n));
    }

    let parts: Vec<&str> = s.splitn(2, ':').collect();
    if parts.len() != 2 {
        bail!("expected selector N, p:N, w:N, or m:N (got: {:?})", s);
    }

    let idx: usize = parts[1]
        .parse()
        .with_context(|| format!("selector index must be numeric: {:?}", parts[1]))?;

    match parts[0].to_lowercase().as_str() {
        "p" | "part" | "w" | "warc" => Ok((SelectorKind::Warc, idx)),
        "m" | "manifest" => Ok((SelectorKind::Manifest, idx)),
        other => bail!("unknown selector prefix {:?} (use p:, w:, or m:)", other),
    }
}

// ---------------------------------------------------------------------------
// Manifest
// ---------------------------------------------------------------------------

/// Load the CC index manifest (list of parquet paths).
/// Caches locally at `~/data/common-crawl/{crawl_id}/manifest.txt`.
async fn load_manifest(crawl_id: &str) -> Result<Vec<String>> {
    let cache_path = expand_home(&format!(
        "~/data/common-crawl/{}/manifest.txt",
        crawl_id
    ));

    // Try cache (valid for 7 days)
    if let Ok(cached) = read_cached_manifest(&cache_path) {
        println!("Using cached manifest ({} entries)", cached.len());
        return Ok(cached);
    }

    // Download manifest (gzipped)
    let url = format!(
        "{}/crawl-data/{}/cc-index-table.paths.gz",
        CC_BASE_URL, crawl_id
    );
    println!("Downloading manifest from {}...", url);

    let client = reqwest::Client::builder()
        .timeout(std::time::Duration::from_secs(120))
        .build()?;
    let resp = client.get(&url).send().await?;

    if !resp.status().is_success() {
        bail!(
            "Failed to download manifest: HTTP {} for {}",
            resp.status(),
            url
        );
    }

    let bytes = resp.bytes().await?;

    // Decompress gzip
    let mut decoder = flate2::read::GzDecoder::new(&bytes[..]);
    let mut text = String::new();
    decoder
        .read_to_string(&mut text)
        .context("decompressing manifest")?;

    let paths: Vec<String> = text
        .lines()
        .map(|l| l.trim().to_string())
        .filter(|l| !l.is_empty())
        .collect();

    println!("Manifest: {} total entries", paths.len());

    // Cache
    if let Some(parent) = cache_path.parent() {
        std::fs::create_dir_all(parent).ok();
    }
    std::fs::write(&cache_path, &text).ok();

    Ok(paths)
}

fn read_cached_manifest(path: &Path) -> Result<Vec<String>> {
    let metadata = std::fs::metadata(path)?;
    let age = metadata
        .modified()?
        .elapsed()
        .unwrap_or(std::time::Duration::from_secs(u64::MAX));
    if age > std::time::Duration::from_secs(7 * 86400) {
        bail!("manifest cache expired");
    }
    let text = std::fs::read_to_string(path)?;
    let paths: Vec<String> = text
        .lines()
        .map(|l| l.trim().to_string())
        .filter(|l| !l.is_empty())
        .collect();
    if paths.is_empty() {
        bail!("empty manifest cache");
    }
    Ok(paths)
}

// ---------------------------------------------------------------------------
// Parquet download
// ---------------------------------------------------------------------------

/// Map a remote CC parquet path to a local path under ~/data/common-crawl/{crawl_id}/index/.
fn local_parquet_path(crawl_id: &str, remote_path: &str) -> PathBuf {
    // Remote paths look like:
    //   cc-index/table/cc-main/warc/crawl=CC-MAIN-2026-08/subset=warc/part-00000-xxx.parquet
    // Local: ~/data/common-crawl/{crawl_id}/index/{hive_partitions}/filename.parquet
    let base = expand_home(&format!("~/data/common-crawl/{}/index", crawl_id));

    // Strip the prefix up to "crawl=" to get the hive-partitioned remainder
    let rel = if let Some(pos) = remote_path.find("crawl=") {
        &remote_path[pos..]
    } else {
        remote_path
    };

    base.join(rel)
}

/// Download a parquet file from CC with progress display.
async fn download_parquet(remote_path: &str, local_path: &Path) -> Result<()> {
    let url = format!("{}/{}", CC_BASE_URL, remote_path);
    let filename = local_path
        .file_name()
        .map(|f| f.to_string_lossy().to_string())
        .unwrap_or_else(|| "file".to_string());

    println!("Downloading {} ...", filename);

    if let Some(parent) = local_path.parent() {
        std::fs::create_dir_all(parent)?;
    }

    let client = reqwest::Client::builder()
        .timeout(std::time::Duration::from_secs(600))
        .build()?;

    let resp = client.get(&url).send().await?;
    if !resp.status().is_success() {
        bail!("Download failed: HTTP {} for {}", resp.status(), url);
    }

    let total_size = resp.content_length().unwrap_or(0);

    let pb = indicatif::ProgressBar::new(total_size);
    pb.set_style(
        indicatif::ProgressStyle::default_bar()
            .template("{spinner:.green} [{bar:40.cyan/blue}] {bytes}/{total_bytes} ({bytes_per_sec}, {eta})")
            .unwrap()
            .progress_chars("#>-"),
    );

    // Write to temp file then rename (atomic)
    let tmp_path = local_path.with_extension("parquet.tmp");
    let mut file = std::fs::File::create(&tmp_path)?;

    use futures_util::StreamExt;
    let mut stream = resp.bytes_stream();
    let mut downloaded: u64 = 0;

    while let Some(chunk) = stream.next().await {
        let chunk = chunk.context("reading download stream")?;
        std::io::Write::write_all(&mut file, &chunk)?;
        downloaded += chunk.len() as u64;
        pb.set_position(downloaded);
    }

    pb.finish_with_message("done");
    drop(file);

    std::fs::rename(&tmp_path, local_path)?;
    println!(
        "Downloaded: {} ({} MB)",
        local_path.display(),
        downloaded / 1_048_576
    );

    Ok(())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Extract a short part name from a parquet filename (e.g. "part-00000" from "part-00000-xxx.parquet").
fn extract_part_name(path: &str) -> String {
    let filename = Path::new(path)
        .file_name()
        .map(|f| f.to_string_lossy().to_string())
        .unwrap_or_default();

    // CC parquet filenames: part-00000-ad224845-8983-48a6-b378-96f1195914cb.c000.gz.parquet
    // Extract "part-00000"
    if filename.starts_with("part-") {
        if let Some(pos) = filename[5..].find('-') {
            return filename[..5 + pos].to_string();
        }
    }

    // Fallback: use stem
    Path::new(&filename)
        .file_stem()
        .map(|s| s.to_string_lossy().to_string())
        .unwrap_or_else(|| "unknown".to_string())
}
