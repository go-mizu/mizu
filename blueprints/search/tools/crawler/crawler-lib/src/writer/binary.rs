use super::{FailureWriter, ResultWriter};
use crate::types::{CrawlResult, FailedDomain, FailedURL};
use anyhow::{Context, Result};
use std::fs::File;
use std::io::{BufWriter, Read as _, Write};
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use tracing::{debug, error, info};

const DEFAULT_SEG_SIZE_MB: usize = 64;
const DEFAULT_CHANNEL_CAP: usize = 65536;

/// Opens a new segment file with a BufWriter.
fn open_segment(dir: &Path, idx: usize) -> Result<BufWriter<File>> {
    let path = dir.join(format!("seg_{:03}.bin", idx));
    let f = File::create(&path)
        .with_context(|| format!("failed to create segment file: {}", path.display()))?;
    Ok(BufWriter::new(f))
}

// ---------------------------------------------------------------------------
// BinaryResultWriter
// ---------------------------------------------------------------------------

/// Non-blocking result writer that serializes CrawlResults to length-prefixed
/// bincode segment files. Workers send results through a bounded crossbeam
/// channel; a dedicated flusher thread handles all disk I/O.
///
/// Architecture:
/// ```text
/// Workers -> write() -> crossbeam bounded channel -> flusher thread -> seg_NNN.bin files
/// ```
///
/// Segment files rotate at a configurable size (default 64 MB). Each record is
/// stored as `[u32 le length][bincode data]`.
pub struct BinaryResultWriter {
    /// Wrapped in Option so close() can take and drop it to signal the flusher.
    tx: Mutex<Option<crossbeam_channel::Sender<CrawlResult>>>,
    handle: Mutex<Option<std::thread::JoinHandle<()>>>,
    dir: PathBuf,
}

impl BinaryResultWriter {
    pub fn new(dir: &Path, channel_cap: usize, seg_size_mb: usize) -> Result<Self> {
        std::fs::create_dir_all(dir)
            .with_context(|| format!("failed to create binary writer dir: {}", dir.display()))?;

        let (tx, rx) = crossbeam_channel::bounded::<CrawlResult>(channel_cap);
        let seg_size_bytes = seg_size_mb * 1024 * 1024;
        let dir_path = dir.to_path_buf();

        let handle = std::thread::Builder::new()
            .name("bin-result-flusher".into())
            .spawn(move || {
                result_flusher_loop(&dir_path, &rx, seg_size_bytes);
            })
            .context("failed to spawn result flusher thread")?;

        Ok(Self {
            tx: Mutex::new(Some(tx)),
            handle: Mutex::new(Some(handle)),
            dir: dir.to_path_buf(),
        })
    }

    /// Create with default channel capacity (65536) and segment size (64 MB).
    pub fn with_defaults(dir: &Path) -> Result<Self> {
        Self::new(dir, DEFAULT_CHANNEL_CAP, DEFAULT_SEG_SIZE_MB)
    }

    /// Returns the directory where segment files are written.
    pub fn dir(&self) -> &Path {
        &self.dir
    }
}

fn result_flusher_loop(
    dir: &Path,
    rx: &crossbeam_channel::Receiver<CrawlResult>,
    seg_size_bytes: usize,
) {
    let mut seg_idx: usize = 0;
    let mut writer = match open_segment(dir, seg_idx) {
        Ok(w) => w,
        Err(e) => {
            error!("bin-result-flusher: failed to open initial segment: {e}");
            return;
        }
    };
    let mut seg_bytes: usize = 0;
    let mut total_records: u64 = 0;

    for result in rx.iter() {
        let encoded = match bincode::serialize(&result) {
            Ok(v) => v,
            Err(e) => {
                error!("bin-result-flusher: serialize error: {e}");
                continue;
            }
        };

        let len = encoded.len() as u32;
        if let Err(e) = writer
            .write_all(&len.to_le_bytes())
            .and_then(|_| writer.write_all(&encoded))
        {
            error!("bin-result-flusher: write error on seg_{seg_idx:03}: {e}");
            continue;
        }

        seg_bytes += 4 + encoded.len();
        total_records += 1;

        if seg_bytes >= seg_size_bytes {
            if let Err(e) = writer.flush() {
                error!("bin-result-flusher: flush error on seg_{seg_idx:03}: {e}");
            }
            debug!(
                "bin-result-flusher: rotated seg_{:03}.bin ({} bytes)",
                seg_idx, seg_bytes
            );
            seg_idx += 1;
            writer = match open_segment(dir, seg_idx) {
                Ok(w) => w,
                Err(e) => {
                    error!("bin-result-flusher: failed to open seg_{seg_idx:03}: {e}");
                    return;
                }
            };
            seg_bytes = 0;
        }
    }

    // Channel closed -- flush remaining data.
    if let Err(e) = writer.flush() {
        error!("bin-result-flusher: final flush error: {e}");
    }
    info!(
        "bin-result-flusher: done, {total_records} records in {} segments",
        seg_idx + 1
    );
}

impl ResultWriter for BinaryResultWriter {
    fn write(&self, result: CrawlResult) -> Result<()> {
        let guard = self.tx.lock().unwrap();
        match guard.as_ref() {
            Some(tx) => tx
                .send(result)
                .map_err(|e| anyhow::anyhow!("binary result channel closed: {e}")),
            None => Err(anyhow::anyhow!("binary result writer already closed")),
        }
    }

    fn flush(&self) -> Result<()> {
        // Flush is deferred to the flusher thread; no-op here.
        Ok(())
    }

    fn close(&self) -> Result<()> {
        // Drop the sender to signal the flusher thread that no more data is coming.
        // This causes rx.iter() to terminate.
        {
            let mut guard = self.tx.lock().unwrap();
            guard.take(); // drops the Sender
        }

        // Now join the flusher thread -- it will drain remaining items and exit.
        let handle = self.handle.lock().unwrap().take();
        if let Some(h) = handle {
            h.join()
                .map_err(|_| anyhow::anyhow!("result flusher thread panicked"))?;
        }
        Ok(())
    }
}

// ---------------------------------------------------------------------------
// BinaryFailureWriter
// ---------------------------------------------------------------------------

/// Non-blocking failure writer that serializes FailedURL and FailedDomain
/// to separate segment file streams via bounded crossbeam channels.
pub struct BinaryFailureWriter {
    url_tx: Mutex<Option<crossbeam_channel::Sender<FailedURL>>>,
    domain_tx: Mutex<Option<crossbeam_channel::Sender<FailedDomain>>>,
    handles: Mutex<Vec<std::thread::JoinHandle<()>>>,
    dir: PathBuf,
}

impl BinaryFailureWriter {
    pub fn new(dir: &Path, channel_cap: usize, seg_size_mb: usize) -> Result<Self> {
        std::fs::create_dir_all(dir)
            .with_context(|| format!("failed to create failure writer dir: {}", dir.display()))?;

        let seg_size_bytes = seg_size_mb * 1024 * 1024;

        // URL flusher
        let (url_tx, url_rx) = crossbeam_channel::bounded::<FailedURL>(channel_cap);
        let url_dir = dir.join("failed_urls");
        std::fs::create_dir_all(&url_dir)?;
        let url_handle = std::thread::Builder::new()
            .name("bin-fail-url-flusher".into())
            .spawn({
                let d = url_dir;
                move || {
                    generic_flusher_loop::<FailedURL>(&d, &url_rx, seg_size_bytes, "fail-url");
                }
            })
            .context("failed to spawn URL failure flusher thread")?;

        // Domain flusher
        let (domain_tx, domain_rx) = crossbeam_channel::bounded::<FailedDomain>(channel_cap);
        let domain_dir = dir.join("failed_domains");
        std::fs::create_dir_all(&domain_dir)?;
        let domain_handle = std::thread::Builder::new()
            .name("bin-fail-domain-flusher".into())
            .spawn({
                let d = domain_dir;
                move || {
                    generic_flusher_loop::<FailedDomain>(
                        &d,
                        &domain_rx,
                        seg_size_bytes,
                        "fail-domain",
                    );
                }
            })
            .context("failed to spawn domain failure flusher thread")?;

        Ok(Self {
            url_tx: Mutex::new(Some(url_tx)),
            domain_tx: Mutex::new(Some(domain_tx)),
            handles: Mutex::new(vec![url_handle, domain_handle]),
            dir: dir.to_path_buf(),
        })
    }

    /// Create with default channel capacity (65536) and segment size (64 MB).
    pub fn with_defaults(dir: &Path) -> Result<Self> {
        Self::new(dir, DEFAULT_CHANNEL_CAP, DEFAULT_SEG_SIZE_MB)
    }

    /// Returns the directory where failure segment files are written.
    pub fn dir(&self) -> &Path {
        &self.dir
    }
}

/// Generic flusher loop for any Serialize type.
fn generic_flusher_loop<T: serde::Serialize>(
    dir: &Path,
    rx: &crossbeam_channel::Receiver<T>,
    seg_size_bytes: usize,
    label: &str,
) {
    let mut seg_idx: usize = 0;
    let mut writer = match open_segment(dir, seg_idx) {
        Ok(w) => w,
        Err(e) => {
            error!("bin-{label}-flusher: failed to open initial segment: {e}");
            return;
        }
    };
    let mut seg_bytes: usize = 0;
    let mut total_records: u64 = 0;

    for item in rx.iter() {
        let encoded = match bincode::serialize(&item) {
            Ok(v) => v,
            Err(e) => {
                error!("bin-{label}-flusher: serialize error: {e}");
                continue;
            }
        };

        let len = encoded.len() as u32;
        if let Err(e) = writer
            .write_all(&len.to_le_bytes())
            .and_then(|_| writer.write_all(&encoded))
        {
            error!("bin-{label}-flusher: write error on seg_{seg_idx:03}: {e}");
            continue;
        }

        seg_bytes += 4 + encoded.len();
        total_records += 1;

        if seg_bytes >= seg_size_bytes {
            if let Err(e) = writer.flush() {
                error!("bin-{label}-flusher: flush error on seg_{seg_idx:03}: {e}");
            }
            debug!(
                "bin-{label}-flusher: rotated seg_{:03}.bin ({} bytes)",
                seg_idx, seg_bytes
            );
            seg_idx += 1;
            writer = match open_segment(dir, seg_idx) {
                Ok(w) => w,
                Err(e) => {
                    error!("bin-{label}-flusher: failed to open seg_{seg_idx:03}: {e}");
                    return;
                }
            };
            seg_bytes = 0;
        }
    }

    if let Err(e) = writer.flush() {
        error!("bin-{label}-flusher: final flush error: {e}");
    }
    info!(
        "bin-{label}-flusher: done, {total_records} records in {} segments",
        seg_idx + 1
    );
}

impl FailureWriter for BinaryFailureWriter {
    fn write_url(&self, failed: FailedURL) -> Result<()> {
        let guard = self.url_tx.lock().unwrap();
        match guard.as_ref() {
            Some(tx) => tx
                .send(failed)
                .map_err(|e| anyhow::anyhow!("binary failure URL channel closed: {e}")),
            None => Err(anyhow::anyhow!("binary failure writer already closed")),
        }
    }

    fn write_domain(&self, failed: FailedDomain) -> Result<()> {
        let guard = self.domain_tx.lock().unwrap();
        match guard.as_ref() {
            Some(tx) => tx
                .send(failed)
                .map_err(|e| anyhow::anyhow!("binary failure domain channel closed: {e}")),
            None => Err(anyhow::anyhow!("binary failure writer already closed")),
        }
    }

    fn flush(&self) -> Result<()> {
        Ok(())
    }

    fn close(&self) -> Result<()> {
        // Drop both senders to signal the flusher threads.
        {
            self.url_tx.lock().unwrap().take();
            self.domain_tx.lock().unwrap().take();
        }

        // Join all flusher threads.
        let handles: Vec<_> = {
            let mut guard = self.handles.lock().unwrap();
            guard.drain(..).collect()
        };
        for h in handles {
            h.join()
                .map_err(|_| anyhow::anyhow!("failure flusher thread panicked"))?;
        }
        Ok(())
    }
}

// ---------------------------------------------------------------------------
// Segment reading (for future import)
// ---------------------------------------------------------------------------

/// Read all CrawlResult records from segment files in a directory.
///
/// Segment files are expected to be named `seg_*.bin` and contain
/// length-prefixed bincode records: `[u32 le len][bincode data]...`
pub fn read_result_segments(dir: &Path) -> Result<Vec<CrawlResult>> {
    read_segments::<CrawlResult>(dir)
}

/// Read all FailedURL records from segment files in a directory.
pub fn read_failed_url_segments(dir: &Path) -> Result<Vec<FailedURL>> {
    read_segments::<FailedURL>(dir)
}

/// Read all FailedDomain records from segment files in a directory.
pub fn read_failed_domain_segments(dir: &Path) -> Result<Vec<FailedDomain>> {
    read_segments::<FailedDomain>(dir)
}

/// Generic segment reader for any Deserialize type.
fn read_segments<T: serde::de::DeserializeOwned>(dir: &Path) -> Result<Vec<T>> {
    let mut paths: Vec<PathBuf> = std::fs::read_dir(dir)
        .with_context(|| format!("failed to read segment dir: {}", dir.display()))?
        .filter_map(|e| e.ok())
        .map(|e| e.path())
        .filter(|p| {
            p.extension().map_or(false, |ext| ext == "bin")
                && p.file_name()
                    .and_then(|n| n.to_str())
                    .map_or(false, |n| n.starts_with("seg_"))
        })
        .collect();

    paths.sort();

    let mut results = Vec::new();

    for path in &paths {
        let mut file = File::open(path)
            .with_context(|| format!("failed to open segment: {}", path.display()))?;

        loop {
            // Read length prefix (4 bytes, little-endian u32).
            let mut len_buf = [0u8; 4];
            match file.read_exact(&mut len_buf) {
                Ok(()) => {}
                Err(e) if e.kind() == std::io::ErrorKind::UnexpectedEof => break,
                Err(e) => {
                    return Err(e)
                        .with_context(|| format!("read len from {}", path.display()));
                }
            }

            let len = u32::from_le_bytes(len_buf) as usize;
            let mut data = vec![0u8; len];
            file.read_exact(&mut data)
                .with_context(|| format!("read record data from {}", path.display()))?;

            let item: T = bincode::deserialize(&data)
                .with_context(|| format!("deserialize record from {}", path.display()))?;
            results.push(item);
        }
    }

    Ok(results)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::{CrawlResult, FailedDomain, FailedURL};

    fn make_result(url: &str) -> CrawlResult {
        CrawlResult {
            url: url.to_string(),
            domain: "example.com".to_string(),
            status_code: 200,
            content_type: "text/html".to_string(),
            content_length: 1234,
            title: "Test".to_string(),
            description: "A test page".to_string(),
            language: "en".to_string(),
            redirect_url: String::new(),
            fetch_time_ms: 42,
            crawled_at: chrono::Utc::now().naive_utc(),
            error: String::new(),
            body: String::new(),
        }
    }

    #[test]
    fn test_result_writer_roundtrip() {
        let dir = tempfile::tempdir().unwrap();
        let seg_dir = dir.path().join("results");

        let writer = BinaryResultWriter::new(&seg_dir, 100, 1).unwrap();

        for i in 0..10 {
            writer
                .write(make_result(&format!("https://example.com/{i}")))
                .unwrap();
        }

        writer.close().unwrap();

        let results = read_result_segments(&seg_dir).unwrap();
        assert_eq!(results.len(), 10);
        assert_eq!(results[0].url, "https://example.com/0");
        assert_eq!(results[9].url, "https://example.com/9");
    }

    #[test]
    fn test_failure_writer_roundtrip() {
        let dir = tempfile::tempdir().unwrap();
        let fail_dir = dir.path().join("failures");

        let writer = BinaryFailureWriter::new(&fail_dir, 100, 1).unwrap();

        for i in 0..5 {
            writer
                .write_url(FailedURL::new(
                    &format!("https://fail.com/{i}"),
                    "fail.com",
                    "http_timeout",
                ))
                .unwrap();
        }

        writer
            .write_domain(FailedDomain {
                domain: "dead.com".to_string(),
                reason: "dns_dead".to_string(),
                error: "NXDOMAIN".to_string(),
                url_count: 42,
                detected_at: chrono::Utc::now().naive_utc(),
            })
            .unwrap();

        writer.close().unwrap();

        let urls = read_failed_url_segments(&fail_dir.join("failed_urls")).unwrap();
        assert_eq!(urls.len(), 5);
        assert_eq!(urls[0].reason, "http_timeout");

        let domains = read_failed_domain_segments(&fail_dir.join("failed_domains")).unwrap();
        assert_eq!(domains.len(), 1);
        assert_eq!(domains[0].domain, "dead.com");
    }

    #[test]
    fn test_segment_rotation() {
        let dir = tempfile::tempdir().unwrap();
        let seg_dir = dir.path().join("rotation");

        // Use seg_size_mb=0 which means seg_size_bytes=0, so every write rotates.
        let writer = BinaryResultWriter::new(&seg_dir, 100, 0).unwrap();

        for i in 0..3 {
            writer
                .write(make_result(&format!("https://example.com/{i}")))
                .unwrap();
        }

        writer.close().unwrap();

        // Should have multiple segment files.
        let seg_count = std::fs::read_dir(&seg_dir)
            .unwrap()
            .filter_map(|e| e.ok())
            .filter(|e| {
                e.path()
                    .file_name()
                    .and_then(|n| n.to_str())
                    .map_or(false, |n| n.starts_with("seg_") && n.ends_with(".bin"))
            })
            .count();
        // With seg_size=0, every record triggers rotation so we get N+1 files
        // (the last one may be empty). At least 3 segments.
        assert!(seg_count >= 3, "expected >= 3 segments, got {seg_count}");

        // All records should still be readable.
        let results = read_result_segments(&seg_dir).unwrap();
        assert_eq!(results.len(), 3);
    }

    #[test]
    fn test_write_after_close_returns_error() {
        let dir = tempfile::tempdir().unwrap();
        let seg_dir = dir.path().join("closed");

        let writer = BinaryResultWriter::new(&seg_dir, 100, 1).unwrap();
        writer.close().unwrap();

        let err = writer.write(make_result("https://example.com/late"));
        assert!(err.is_err());
    }
}
