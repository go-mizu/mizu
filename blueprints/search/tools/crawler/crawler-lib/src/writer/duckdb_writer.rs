use super::{FailureWriter, ResultWriter};
use crate::types::{CrawlResult, FailedDomain, FailedURL};
use anyhow::{Context, Result};
use crossbeam_channel::{bounded, Sender};
use duckdb::Connection;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicI64, Ordering};
use std::sync::{Arc, Mutex};
use std::thread::JoinHandle;
use tracing::{debug, error, info, warn};

// ---------------------------------------------------------------------------
// FNV-1a hash (matches Go: seed=2166136261, multiply=16777619)
// ---------------------------------------------------------------------------
const FNV_OFFSET: u32 = 2_166_136_261;
const FNV_PRIME: u32 = 16_777_619;

pub(crate) fn fnv1a_hash(data: &[u8]) -> u32 {
    let mut h = FNV_OFFSET;
    for &b in data {
        h ^= b as u32;
        h = h.wrapping_mul(FNV_PRIME);
    }
    h
}

pub(crate) fn shard_for_url(url: &str, num_shards: usize) -> usize {
    (fnv1a_hash(url.as_bytes()) as usize) % num_shards
}

// ---------------------------------------------------------------------------
// Sanitize strings for DuckDB VARCHAR (no null bytes, valid UTF-8)
// ---------------------------------------------------------------------------
fn sanitize_str(s: &str) -> String {
    s.replace('\0', "")
}

// ---------------------------------------------------------------------------
// Stale lock removal
// ---------------------------------------------------------------------------
fn remove_stale_lock(db_path: &Path) {
    let lock_path = db_path.with_extension("duckdb.lock");
    if !lock_path.exists() {
        return;
    }
    // Try to read PID from the lock file
    if let Ok(contents) = std::fs::read_to_string(&lock_path) {
        if let Ok(pid) = contents.trim().parse::<u32>() {
            // Check if process is alive
            #[cfg(unix)]
            {
                let ret = unsafe { libc::kill(pid as i32, 0) };
                if ret != 0 {
                    warn!(
                        "removing stale lock file {:?} (pid {} is dead)",
                        lock_path, pid
                    );
                    let _ = std::fs::remove_file(&lock_path);
                }
            }
            #[cfg(not(unix))]
            {
                let _ = pid;
                warn!("removing stale lock file {:?}", lock_path);
                let _ = std::fs::remove_file(&lock_path);
            }
        } else {
            // Can't parse PID, remove it
            warn!(
                "removing stale lock file {:?} (unparseable PID)",
                lock_path
            );
            let _ = std::fs::remove_file(&lock_path);
        }
    }
}

// ---------------------------------------------------------------------------
// Open DuckDB connection with per-shard settings
// ---------------------------------------------------------------------------
pub(crate) fn open_result_db(path: &Path, mem_mb: usize) -> Result<Connection> {
    remove_stale_lock(path);
    let conn = Connection::open(path)
        .with_context(|| format!("failed to open DuckDB at {:?}", path))?;

    let checkpoint_mb = (mem_mb / 40).max(1);
    conn.execute_batch(&format!(
        "SET memory_limit = '{}MB';
         SET threads = 1;
         SET preserve_insertion_order = false;
         SET checkpoint_threshold = '{}MB';",
        mem_mb, checkpoint_mb,
    ))
    .context("failed to set DuckDB shard settings")?;

    conn.execute_batch(
        "CREATE TABLE IF NOT EXISTS results (
            url VARCHAR, status_code INTEGER, content_type VARCHAR,
            content_length BIGINT, body VARCHAR, title VARCHAR,
            description VARCHAR, language VARCHAR, domain VARCHAR,
            redirect_url VARCHAR, fetch_time_ms BIGINT,
            crawled_at TIMESTAMP, error VARCHAR,
            status VARCHAR DEFAULT 'done', body_cid VARCHAR DEFAULT ''
        );",
    )
    .context("failed to create results table")?;

    Ok(conn)
}

// ---------------------------------------------------------------------------
// Batch INSERT for results
// ---------------------------------------------------------------------------
pub(crate) fn flush_result_batch(conn: &Connection, batch: &[CrawlResult]) -> Result<()> {
    if batch.is_empty() {
        return Ok(());
    }

    // Build multi-row INSERT with placeholders.
    // 13 columns per row (status and body_cid have defaults, we INSERT explicitly).
    const COLS: usize = 15;
    let row_placeholder = format!("({})", vec!["?"; COLS].join(", "));
    let all_rows = vec![row_placeholder.as_str(); batch.len()].join(", ");
    let sql = format!(
        "INSERT INTO results (url, status_code, content_type, content_length, body, \
         title, description, language, domain, redirect_url, fetch_time_ms, \
         crawled_at, error, status, body_cid) VALUES {}",
        all_rows
    );

    // Build parameter list as Vec<Box<dyn duckdb::ToSql>>.
    let mut params: Vec<Box<dyn duckdb::ToSql>> = Vec::with_capacity(batch.len() * COLS);
    for r in batch {
        params.push(Box::new(sanitize_str(&r.url)));
        params.push(Box::new(r.status_code as i32));
        params.push(Box::new(sanitize_str(&r.content_type)));
        params.push(Box::new(r.content_length));
        params.push(Box::new(sanitize_str(&r.body)));
        params.push(Box::new(sanitize_str(&r.title)));
        params.push(Box::new(sanitize_str(&r.description)));
        params.push(Box::new(sanitize_str(&r.language)));
        params.push(Box::new(sanitize_str(&r.domain)));
        params.push(Box::new(sanitize_str(&r.redirect_url)));
        params.push(Box::new(r.fetch_time_ms));
        params.push(Box::new(r.crawled_at.format("%Y-%m-%d %H:%M:%S").to_string()));
        params.push(Box::new(sanitize_str(&r.error)));
        params.push(Box::new("done".to_string()));
        params.push(Box::new(sanitize_str(&r.body_cid)));
    }

    let param_refs: Vec<&dyn duckdb::ToSql> = params.iter().map(|p| p.as_ref()).collect();

    conn.execute(&sql, param_refs.as_slice())
        .with_context(|| format!("failed to insert batch of {} results", batch.len()))?;
    Ok(())
}

// ---------------------------------------------------------------------------
// ResultShard
// ---------------------------------------------------------------------------
struct ResultShard {
    batch: Mutex<Vec<CrawlResult>>,
    batch_size: usize,
    tx: Mutex<Option<Sender<Vec<CrawlResult>>>>,
    handle: Mutex<Option<JoinHandle<()>>>,
}

impl ResultShard {
    fn new(shard_idx: usize, path: PathBuf, mem_mb: usize, batch_size: usize) -> Result<Self> {
        let conn =
            open_result_db(&path, mem_mb).with_context(|| format!("shard {}", shard_idx))?;

        let (tx, rx) = bounded::<Vec<CrawlResult>>(16);

        let handle = std::thread::Builder::new()
            .name(format!("duckdb-flusher-{}", shard_idx))
            .spawn(move || {
                for batch in rx.iter() {
                    if let Err(e) = flush_result_batch(&conn, &batch) {
                        error!("shard {} flush error: {:?}", shard_idx, e);
                    }
                }
                debug!("shard {} flusher thread exiting", shard_idx);
            })
            .with_context(|| format!("failed to spawn flusher thread for shard {}", shard_idx))?;

        Ok(Self {
            batch: Mutex::new(Vec::with_capacity(batch_size)),
            batch_size,
            tx: Mutex::new(Some(tx)),
            handle: Mutex::new(Some(handle)),
        })
    }

    fn send_batch(&self, b: Vec<CrawlResult>) -> Result<()> {
        let guard = self.tx.lock().unwrap();
        if let Some(tx) = guard.as_ref() {
            tx.send(b).map_err(|_| anyhow::anyhow!("flusher channel closed"))?;
        }
        Ok(())
    }

    fn add(&self, result: CrawlResult) -> Result<()> {
        let batch_to_send = {
            let mut batch = self.batch.lock().unwrap();
            batch.push(result);
            if batch.len() >= self.batch_size {
                Some(std::mem::replace(
                    &mut *batch,
                    Vec::with_capacity(self.batch_size),
                ))
            } else {
                None
            }
        };
        if let Some(b) = batch_to_send {
            debug!("sent batch of {} to flusher", b.len());
            self.send_batch(b)?;
        }
        Ok(())
    }

    fn flush(&self) -> Result<()> {
        let batch_to_send = {
            let mut batch = self.batch.lock().unwrap();
            if batch.is_empty() {
                None
            } else {
                Some(std::mem::replace(
                    &mut *batch,
                    Vec::with_capacity(self.batch_size),
                ))
            }
        };
        if let Some(b) = batch_to_send {
            self.send_batch(b)?;
        }
        Ok(())
    }

    fn close(&self) -> Result<()> {
        // 1. Flush remaining data into the channel
        self.flush()?;
        // 2. Drop the sender — this signals rx.iter() to terminate in the flusher thread
        {
            let mut guard = self.tx.lock().unwrap();
            *guard = None; // drops the Sender
        }
        // 3. Join the flusher thread to ensure all data is written to DuckDB
        if let Ok(mut guard) = self.handle.lock() {
            if let Some(h) = guard.take() {
                let _ = h.join();
            }
        }
        Ok(())
    }
}

// ---------------------------------------------------------------------------
// DuckDBResultWriter
// ---------------------------------------------------------------------------
pub struct DuckDBResultWriter {
    shards: Vec<ResultShard>,
    num_shards: usize,
    flushed: Arc<AtomicI64>,
}

impl DuckDBResultWriter {
    /// Create a new sharded DuckDB result writer.
    ///
    /// - `output_dir`: directory to store shard files (`results_000.duckdb` .. `results_NNN.duckdb`)
    /// - `num_shards`: number of shards (e.g. 8)
    /// - `mem_mb`: DuckDB memory limit per shard
    /// - `batch_size`: rows buffered before sending to flusher
    pub fn new(
        output_dir: &str,
        num_shards: usize,
        mem_mb: usize,
        batch_size: usize,
    ) -> Result<Self> {
        let dir = Path::new(output_dir);
        std::fs::create_dir_all(dir)
            .with_context(|| format!("failed to create output dir {:?}", dir))?;

        let mut shards = Vec::with_capacity(num_shards);
        for i in 0..num_shards {
            let path = dir.join(format!("results_{:03}.duckdb", i));
            let shard = ResultShard::new(i, path, mem_mb, batch_size)?;
            shards.push(shard);
        }

        info!(
            "opened {} DuckDB result shards in {:?} (mem={}MB, batch={})",
            num_shards, dir, mem_mb, batch_size
        );

        Ok(Self {
            shards,
            num_shards,
            flushed: Arc::new(AtomicI64::new(0)),
        })
    }

    /// Total number of flushed rows (approximate, updated per batch send).
    pub fn flushed_count(&self) -> i64 {
        self.flushed.load(Ordering::Relaxed)
    }
}

impl ResultWriter for DuckDBResultWriter {
    fn write(&self, result: CrawlResult) -> Result<()> {
        let idx = shard_for_url(&result.url, self.num_shards);
        self.shards[idx].add(result)?;
        self.flushed.fetch_add(1, Ordering::Relaxed);
        Ok(())
    }

    fn flush(&self) -> Result<()> {
        for shard in &self.shards {
            shard.flush()?;
        }
        Ok(())
    }

    fn close(&self) -> Result<()> {
        for shard in &self.shards {
            shard.close()?; // flushes remaining, drops sender, joins flusher thread
        }
        info!(
            "DuckDBResultWriter closed, total written: {}",
            self.flushed.load(Ordering::Relaxed)
        );
        Ok(())
    }
}

// When the writer is dropped, ensure all shards are properly closed.
impl Drop for DuckDBResultWriter {
    fn drop(&mut self) {
        for shard in &self.shards {
            let _ = shard.close(); // idempotent: close() is safe to call multiple times
        }
    }
}

// ===========================================================================
// FailedDB (single DuckDB, two tables, two flusher threads)
// ===========================================================================

pub fn open_failed_db(path: &Path, mem_mb: usize) -> Result<Connection> {
    remove_stale_lock(path);
    let conn = Connection::open(path)
        .with_context(|| format!("failed to open FailedDB at {:?}", path))?;

    let checkpoint_mb = (mem_mb / 40).max(1);
    conn.execute_batch(&format!(
        "SET memory_limit = '{}MB';
         SET threads = 1;
         SET preserve_insertion_order = false;
         SET checkpoint_threshold = '{}MB';",
        mem_mb, checkpoint_mb,
    ))
    .context("failed to set FailedDB settings")?;

    conn.execute_batch(
        "CREATE TABLE IF NOT EXISTS failed_domains (
            domain VARCHAR, reason VARCHAR, error VARCHAR,
            url_count BIGINT, detected_at TIMESTAMP
        );
         CREATE TABLE IF NOT EXISTS failed_urls (
            url VARCHAR, domain VARCHAR, reason VARCHAR, subcategory VARCHAR,
            error VARCHAR, status_code INTEGER, fetch_time_ms BIGINT,
            detected_at TIMESTAMP
        );",
    )
    .context("failed to create failed_domains/failed_urls tables")?;

    Ok(conn)
}

// ---------------------------------------------------------------------------
// Batch INSERT for failed_urls
// ---------------------------------------------------------------------------
pub fn flush_failed_url_batch(conn: &Connection, batch: &[FailedURL]) -> Result<()> {
    if batch.is_empty() {
        return Ok(());
    }
    const COLS: usize = 8;
    let row_ph = format!("({})", vec!["?"; COLS].join(", "));
    let all_rows = vec![row_ph.as_str(); batch.len()].join(", ");
    let sql = format!(
        "INSERT INTO failed_urls (url, domain, reason, subcategory, error, status_code, \
         fetch_time_ms, detected_at) VALUES {}",
        all_rows
    );

    let mut params: Vec<Box<dyn duckdb::ToSql>> = Vec::with_capacity(batch.len() * COLS);
    for f in batch {
        params.push(Box::new(sanitize_str(&f.url)));
        params.push(Box::new(sanitize_str(&f.domain)));
        params.push(Box::new(sanitize_str(&f.reason)));
        params.push(Box::new(sanitize_str(&f.subcategory)));
        params.push(Box::new(sanitize_str(&f.error)));
        params.push(Box::new(f.status_code as i32));
        params.push(Box::new(f.fetch_time_ms));
        params.push(Box::new(
            f.detected_at.format("%Y-%m-%d %H:%M:%S").to_string(),
        ));
    }
    let param_refs: Vec<&dyn duckdb::ToSql> = params.iter().map(|p| p.as_ref()).collect();
    conn.execute(&sql, param_refs.as_slice())
        .with_context(|| format!("failed to insert {} failed_urls", batch.len()))?;
    Ok(())
}

// ---------------------------------------------------------------------------
// Batch INSERT for failed_domains
// ---------------------------------------------------------------------------
fn flush_failed_domain_batch(conn: &Connection, batch: &[FailedDomain]) -> Result<()> {
    if batch.is_empty() {
        return Ok(());
    }
    const COLS: usize = 5;
    let row_ph = format!("({})", vec!["?"; COLS].join(", "));
    let all_rows = vec![row_ph.as_str(); batch.len()].join(", ");
    let sql = format!(
        "INSERT INTO failed_domains (domain, reason, error, url_count, detected_at) VALUES {}",
        all_rows
    );

    let mut params: Vec<Box<dyn duckdb::ToSql>> = Vec::with_capacity(batch.len() * COLS);
    for f in batch {
        params.push(Box::new(sanitize_str(&f.domain)));
        params.push(Box::new(sanitize_str(&f.reason)));
        params.push(Box::new(sanitize_str(&f.error)));
        params.push(Box::new(f.url_count));
        params.push(Box::new(
            f.detected_at.format("%Y-%m-%d %H:%M:%S").to_string(),
        ));
    }
    let param_refs: Vec<&dyn duckdb::ToSql> = params.iter().map(|p| p.as_ref()).collect();
    conn.execute(&sql, param_refs.as_slice())
        .with_context(|| format!("failed to insert {} failed_domains", batch.len()))?;
    Ok(())
}

// ---------------------------------------------------------------------------
// DuckDBFailureWriter
// ---------------------------------------------------------------------------
pub struct DuckDBFailureWriter {
    url_batch: Mutex<Vec<FailedURL>>,
    domain_batch: Mutex<Vec<FailedDomain>>,
    batch_size: usize,
    url_tx: Mutex<Option<Sender<Vec<FailedURL>>>>,
    domain_tx: Mutex<Option<Sender<Vec<FailedDomain>>>>,
    handles: Mutex<Vec<Option<JoinHandle<()>>>>,
}

impl DuckDBFailureWriter {
    /// Create a new failure writer backed by a single DuckDB file.
    ///
    /// - `path`: path to the DuckDB file (e.g. `failed.duckdb`)
    /// - `mem_mb`: DuckDB memory limit
    /// - `batch_size`: rows buffered before sending to flusher
    pub fn new(path: &str, mem_mb: usize, batch_size: usize) -> Result<Self> {
        let db_path = Path::new(path);
        if let Some(parent) = db_path.parent() {
            std::fs::create_dir_all(parent)
                .with_context(|| format!("failed to create parent dir for {:?}", db_path))?;
        }

        // DuckDB only allows one connection per file from one process.
        // We open two separate connections by using the same file -- duckdb-rs
        // uses an in-process DuckDB instance that supports multiple connections
        // to the same database.
        let conn_urls = open_failed_db(db_path, mem_mb)?;
        let conn_domains = open_failed_db(db_path, mem_mb)?;

        let (url_tx, url_rx) = bounded::<Vec<FailedURL>>(16);
        let (domain_tx, domain_rx) = bounded::<Vec<FailedDomain>>(16);

        let url_handle = std::thread::Builder::new()
            .name("duckdb-failed-urls".to_string())
            .spawn(move || {
                for batch in url_rx.iter() {
                    if let Err(e) = flush_failed_url_batch(&conn_urls, &batch) {
                        error!("failed_urls flush error: {:?}", e);
                    }
                }
                debug!("failed_urls flusher thread exiting");
            })
            .context("failed to spawn failed_urls flusher thread")?;

        let domain_handle = std::thread::Builder::new()
            .name("duckdb-failed-domains".to_string())
            .spawn(move || {
                for batch in domain_rx.iter() {
                    if let Err(e) = flush_failed_domain_batch(&conn_domains, &batch) {
                        error!("failed_domains flush error: {:?}", e);
                    }
                }
                debug!("failed_domains flusher thread exiting");
            })
            .context("failed to spawn failed_domains flusher thread")?;

        info!("opened FailedDB at {:?} (mem={}MB, batch={})", db_path, mem_mb, batch_size);

        Ok(Self {
            url_batch: Mutex::new(Vec::with_capacity(batch_size)),
            domain_batch: Mutex::new(Vec::with_capacity(batch_size)),
            batch_size,
            url_tx: Mutex::new(Some(url_tx)),
            domain_tx: Mutex::new(Some(domain_tx)),
            handles: Mutex::new(vec![Some(url_handle), Some(domain_handle)]),
        })
    }
}

impl FailureWriter for DuckDBFailureWriter {
    fn write_url(&self, failed: FailedURL) -> Result<()> {
        let batch_to_send = {
            let mut batch = self.url_batch.lock().unwrap();
            batch.push(failed);
            if batch.len() >= self.batch_size {
                Some(std::mem::replace(&mut *batch, Vec::with_capacity(self.batch_size)))
            } else {
                None
            }
        };
        if let Some(b) = batch_to_send {
            if let Some(tx) = self.url_tx.lock().unwrap().as_ref() {
                tx.send(b).map_err(|_| anyhow::anyhow!("failed_urls flusher channel closed"))?;
            }
        }
        Ok(())
    }

    fn write_domain(&self, failed: FailedDomain) -> Result<()> {
        let batch_to_send = {
            let mut batch = self.domain_batch.lock().unwrap();
            batch.push(failed);
            if batch.len() >= self.batch_size {
                Some(std::mem::replace(&mut *batch, Vec::with_capacity(self.batch_size)))
            } else {
                None
            }
        };
        if let Some(b) = batch_to_send {
            if let Some(tx) = self.domain_tx.lock().unwrap().as_ref() {
                tx.send(b).map_err(|_| anyhow::anyhow!("failed_domains flusher channel closed"))?;
            }
        }
        Ok(())
    }

    fn flush(&self) -> Result<()> {
        let url_batch = {
            let mut batch = self.url_batch.lock().unwrap();
            if batch.is_empty() { None }
            else { Some(std::mem::replace(&mut *batch, Vec::with_capacity(self.batch_size))) }
        };
        if let Some(b) = url_batch {
            if let Some(tx) = self.url_tx.lock().unwrap().as_ref() {
                let _ = tx.send(b);
            }
        }
        let domain_batch = {
            let mut batch = self.domain_batch.lock().unwrap();
            if batch.is_empty() { None }
            else { Some(std::mem::replace(&mut *batch, Vec::with_capacity(self.batch_size))) }
        };
        if let Some(b) = domain_batch {
            if let Some(tx) = self.domain_tx.lock().unwrap().as_ref() {
                let _ = tx.send(b);
            }
        }
        Ok(())
    }

    fn close(&self) -> Result<()> {
        // 1. Flush remaining into channels
        self.flush()?;
        // 2. Drop both senders to signal flusher threads to exit
        { *self.url_tx.lock().unwrap() = None; }
        { *self.domain_tx.lock().unwrap() = None; }
        // 3. Join both flusher threads
        if let Ok(mut handles) = self.handles.lock() {
            for h in handles.iter_mut() {
                if let Some(handle) = h.take() {
                    let _ = handle.join();
                }
            }
        }
        info!("DuckDBFailureWriter closed");
        Ok(())
    }
}

impl Drop for DuckDBFailureWriter {
    fn drop(&mut self) {
        // close() is idempotent (Option::take), safe to call from drop
        let _ = self.close();
    }
}

// ---------------------------------------------------------------------------
// Utility: load retry URLs from FailedDB (for pass-2)
// ---------------------------------------------------------------------------
/// Load URLs from failed_urls table that were detected after `since` timestamp.
/// This avoids loading stale failures from prior runs.
pub fn load_retry_urls_since(
    path: &str,
    since: chrono::NaiveDateTime,
) -> Result<Vec<crate::types::SeedURL>> {
    let db_path = Path::new(path);
    if !db_path.exists() {
        return Ok(Vec::new());
    }
    remove_stale_lock(db_path);
    let conn = Connection::open(db_path)
        .with_context(|| format!("failed to open FailedDB at {:?} for retry loading", db_path))?;

    let since_str = since.format("%Y-%m-%d %H:%M:%S").to_string();
    let mut stmt = conn
        .prepare(
            "SELECT url, domain FROM failed_urls \
             WHERE reason IN ('http_timeout', 'domain_deadline_exceeded') \
             AND detected_at >= ?",
        )
        .context("failed to prepare retry URL query")?;

    let rows = stmt
        .query_map([&since_str], |row| {
            Ok(crate::types::SeedURL {
                url: row.get(0)?,
                domain: row.get(1)?,
            })
        })
        .context("failed to query retry URLs")?;

    let mut seeds = Vec::new();
    for row in rows {
        match row {
            Ok(seed) => seeds.push(seed),
            Err(e) => warn!("skipping malformed retry URL row: {:?}", e),
        }
    }

    info!(
        "loaded {} retry URLs from {:?} (since {})",
        seeds.len(),
        db_path,
        since_str
    );
    Ok(seeds)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_fnv1a_hash() {
        // Verify FNV-1a matches known values.
        let h = fnv1a_hash(b"");
        assert_eq!(h, FNV_OFFSET); // empty string = offset basis

        // "foobar" FNV-1a 32-bit = 0xbf9cf968
        let h2 = fnv1a_hash(b"foobar");
        assert_eq!(h2, 0xbf9c_f968);
    }

    #[test]
    fn test_shard_distribution() {
        // Verify sharding is deterministic and distributes.
        let s1 = shard_for_url("https://example.com/a", 8);
        let s2 = shard_for_url("https://example.com/b", 8);
        let s1_again = shard_for_url("https://example.com/a", 8);
        assert_eq!(s1, s1_again);
        assert!(s1 < 8);
        assert!(s2 < 8);
    }

    #[test]
    fn test_sanitize_str() {
        assert_eq!(sanitize_str("hello\0world"), "helloworld");
        assert_eq!(sanitize_str("no nulls"), "no nulls");
    }
}
