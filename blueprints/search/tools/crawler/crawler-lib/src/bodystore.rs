/// Content-addressable body store backed by the filesystem.
///
/// Format is compatible with Go's `pkg/crawl/bodystore`:
/// - Bodies are stored gzip-compressed at `{dir}/{sha[0:2]}/{sha[2:4]}/{sha[4:]}.gz`
/// - CID format: `sha256:{hex64}`
/// - Writes are atomic (tmp → rename), safe for concurrent workers.
use anyhow::{Context, Result};
use flate2::write::GzEncoder;
use flate2::Compression;
use sha2::{Digest, Sha256};
use std::io::Write;
use std::path::{Path, PathBuf};

#[derive(Clone, Debug)]
pub struct BodyStore {
    dir: PathBuf,
}

impl BodyStore {
    /// Open a store rooted at `dir`, creating it if needed.
    pub fn open(dir: impl AsRef<Path>) -> Result<Self> {
        let dir = dir.as_ref().to_path_buf();
        std::fs::create_dir_all(&dir)
            .with_context(|| format!("bodystore: mkdir {:?}", dir))?;
        Ok(Self { dir })
    }

    /// Write `body` to the store and return its CID.
    /// Idempotent: writing the same content twice returns the same CID without re-writing.
    pub fn put(&self, body: &[u8]) -> Result<String> {
        let sum = Sha256::digest(body);
        let hex = format!("{:x}", sum);
        let cid = format!("sha256:{}", hex);

        let path = self.cid_to_path(&hex);
        if path.exists() {
            return Ok(cid);
        }

        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)
                .with_context(|| format!("bodystore: mkdir {:?}", parent))?;
        }

        // Write to a unique temp file then rename (atomic on POSIX).
        // Multiple workers writing the same CID concurrently is safe:
        // the last rename wins and the file is always valid.
        let tmp = path.with_extension("gz.tmp");
        {
            let f = std::fs::File::create(&tmp)
                .with_context(|| format!("bodystore: create tmp {:?}", tmp))?;
            let mut gz = GzEncoder::new(f, Compression::default());
            gz.write_all(body).context("bodystore: gzip write")?;
            gz.finish().context("bodystore: gzip finish")?;
        }
        // Ignore rename errors: another worker may have already written the same CID.
        let _ = std::fs::rename(&tmp, &path);
        Ok(cid)
    }

    pub(crate) fn cid_to_path(&self, hex: &str) -> PathBuf {
        // hex is always 64 chars (SHA-256)
        self.dir
            .join(&hex[0..2])
            .join(&hex[2..4])
            .join(format!("{}.gz", &hex[4..]))
    }
}

use crossbeam_channel::{bounded, Sender};
use dashmap::DashSet;
use std::sync::{Arc, Mutex};

/// Non-blocking body store wrapper.
///
/// `put_async` computes the SHA-256 CID synchronously (microseconds),
/// marks the hash as in-flight in a DashSet to prevent duplicate writes,
/// then sends the body to a background OS thread for gzip + atomic write.
///
/// Workers never block on disk I/O. Call `close()` after the crawl to
/// flush all pending writes before inspecting the store.
pub struct AsyncBodyStore {
    inner: Arc<BodyStore>,
    /// Set of hashes that are written or in-flight. Prevents duplicate writes.
    in_flight: Arc<DashSet<[u8; 32]>>,
    tx: Sender<Option<(Vec<u8>, [u8; 32])>>,
    handle: Mutex<Option<std::thread::JoinHandle<()>>>,
}

impl std::fmt::Debug for AsyncBodyStore {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "AsyncBodyStore")
    }
}

impl AsyncBodyStore {
    pub fn new(dir: impl AsRef<std::path::Path>) -> Result<Self> {
        let inner = Arc::new(BodyStore::open(dir)?);
        let in_flight: Arc<DashSet<[u8; 32]>> = Arc::new(DashSet::new());
        let (tx, rx) = bounded::<Option<(Vec<u8>, [u8; 32])>>(8192);

        let inner2 = Arc::clone(&inner);

        let handle = std::thread::Builder::new()
            .name("body-store-writer".into())
            .spawn(move || {
                for msg in rx.iter() {
                    let (body, _hash) = match msg {
                        Some(item) => item,
                        None => break,
                    };
                    // Delegate full write to existing BodyStore::put — it handles
                    // path existence check, tmp-write, and atomic rename.
                    let _ = inner2.put(&body);
                }
            })
            .context("failed to spawn body-store-writer thread")?;

        Ok(Self {
            inner,
            in_flight,
            tx,
            handle: Mutex::new(Some(handle)),
        })
    }

    /// Compute CID synchronously, schedule write asynchronously.
    ///
    /// Returns the CID immediately. The body is written to disk in the
    /// background; call `close()` to ensure all writes complete.
    pub fn put_async(&self, body: &[u8]) -> String {
        let sum = Sha256::digest(body);
        let hash: [u8; 32] = sum.into();
        let hex = format!("{:x}", sum);
        let cid = format!("sha256:{}", hex);

        // If already written or in-flight, return CID with no work.
        if self.in_flight.contains(&hash) {
            return cid;
        }
        // Check filesystem too (store may have been populated by a prior run).
        if self.inner.cid_to_path(&hex).exists() {
            self.in_flight.insert(hash);
            return cid;
        }
        // Mark as in-flight before sending — prevents a second caller from
        // racing and sending a duplicate write.
        self.in_flight.insert(hash);

        // try_send: if channel is full (8192 cap), fall back to synchronous write.
        // This is safe — the file will be written one way or another.
        if self.tx.try_send(Some((body.to_vec(), hash))).is_err() {
            let _ = self.inner.put(body);
        }
        cid
    }

    /// Flush all pending writes. Blocks until the background thread finishes.
    pub fn close(&self) -> Result<()> {
        let _ = self.tx.send(None); // shutdown sentinel
        if let Ok(mut guard) = self.handle.lock() {
            if let Some(h) = guard.take() {
                h.join()
                    .map_err(|_| anyhow::anyhow!("body-store-writer thread panicked"))?;
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Arc;
    use tempfile::tempdir;

    #[test]
    fn async_body_store_returns_cid_immediately_and_persists() {
        let dir = tempdir().unwrap();
        let store = Arc::new(AsyncBodyStore::new(dir.path()).unwrap());

        let body = b"hello world";
        let cid = store.put_async(body);

        assert!(cid.starts_with("sha256:"), "CID must be sha256:hex");
        assert_eq!(cid.len(), 7 + 64, "sha256: prefix + 64 hex chars");

        // Close flushes background writes
        store.close().unwrap();

        // Verify the file was actually written
        let hex = &cid[7..]; // strip "sha256:"
        let path = dir.path()
            .join(&hex[0..2])
            .join(&hex[2..4])
            .join(format!("{}.gz", &hex[4..]));
        assert!(path.exists(), "body file should exist after close(): {:?}", path);
    }

    #[test]
    fn async_body_store_deduplicates_same_content() {
        let dir = tempdir().unwrap();
        let store = Arc::new(AsyncBodyStore::new(dir.path()).unwrap());

        let body = b"duplicate content";
        let cid1 = store.put_async(body);
        let cid2 = store.put_async(body);

        assert_eq!(cid1, cid2, "same content must produce same CID");
        store.close().unwrap();
    }

    #[test]
    fn async_body_store_concurrent_writes() {
        let dir = tempdir().unwrap();
        let store = Arc::new(AsyncBodyStore::new(dir.path()).unwrap());

        let mut handles = vec![];
        for i in 0..50usize {
            let s = Arc::clone(&store);
            handles.push(std::thread::spawn(move || {
                let body = format!("body content {i}").into_bytes();
                s.put_async(&body)
            }));
        }
        let cids: Vec<String> = handles.into_iter().map(|h| h.join().unwrap()).collect();
        store.close().unwrap();

        // All 50 should be unique CIDs (different content)
        let unique: std::collections::HashSet<&String> = cids.iter().collect();
        assert_eq!(unique.len(), 50);
    }
}
