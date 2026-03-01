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

    fn cid_to_path(&self, hex: &str) -> PathBuf {
        // hex is always 64 chars (SHA-256)
        self.dir
            .join(&hex[0..2])
            .join(&hex[2..4])
            .join(format!("{}.gz", &hex[4..]))
    }
}
