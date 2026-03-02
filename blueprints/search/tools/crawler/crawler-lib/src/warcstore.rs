//! WARC 1.1 body store for crawl results.
//!
//! Each crawled URL gets one `.warc` or `.warc.gz` file containing three records:
//! 1. warcinfo — crawl session metadata
//! 2. request  — HTTP request headers
//! 3. response — HTTP response (status + headers + body)
//!
//! The WARC-Record-ID for the response record is a deterministic UUIDv5 derived
//! from the RFC 3986 canonical form of the URL. Compatible with Go's pkg/crawl/warcstore.
//!
//! # Namespace chain (fixed, documented in spec/0641_warc_crawler.md)
//! DNS_NS = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"    (RFC 4122 DNS namespace)
//! root   = UUIDv5(DNS_NS, "go-mizu.search.warc")
//! ns_response = UUIDv5(root, "response")   <- stored as warc_id
//! ns_request  = UUIDv5(root, "request")    <- WARC-Concurrent-To
//! ns_warcinfo = UUIDv5(root, "warcinfo")   <- warcinfo record

use anyhow::{Context, Result};
use flate2::write::GzEncoder;
use flate2::Compression;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::sync::{Mutex, OnceLock};
use uuid::Uuid;

/// Pre-computed fixed namespace UUIDs (must match Go implementation).
static NS_RESPONSE: OnceLock<Uuid> = OnceLock::new();
static NS_REQUEST: OnceLock<Uuid> = OnceLock::new();
static NS_WARCINFO: OnceLock<Uuid> = OnceLock::new();

fn ns_response() -> &'static Uuid {
    NS_RESPONSE.get_or_init(|| {
        let root = Uuid::new_v5(&Uuid::NAMESPACE_DNS, b"go-mizu.search.warc");
        Uuid::new_v5(&root, b"response")
    })
}

fn ns_request() -> &'static Uuid {
    NS_REQUEST.get_or_init(|| {
        let root = Uuid::new_v5(&Uuid::NAMESPACE_DNS, b"go-mizu.search.warc");
        Uuid::new_v5(&root, b"request")
    })
}

fn ns_warcinfo() -> &'static Uuid {
    NS_WARCINFO.get_or_init(|| {
        let root = Uuid::new_v5(&Uuid::NAMESPACE_DNS, b"go-mizu.search.warc");
        Uuid::new_v5(&root, b"warcinfo")
    })
}

/// RFC 3986 canonical form of a URL:
/// - lowercase scheme and host
/// - remove default ports (80 for http, 443 for https)
/// - strip fragment
/// - preserve path case
pub fn canonical_url(raw: &str) -> String {
    let Ok(mut u) = url::Url::parse(raw) else {
        return raw.to_string();
    };
    // Remove fragment
    u.set_fragment(None);
    // url crate stores scheme lowercase by default; host is normalized too.
    // Remove default ports.
    let scheme = u.scheme().to_string();
    if let Some(port) = u.port() {
        let is_default = (scheme == "http" && port == 80) || (scheme == "https" && port == 443);
        if is_default {
            let _ = u.set_port(None);
        }
    }
    u.into()
}

/// Input data for writing one WARC file.
#[derive(Debug, Clone)]
pub struct WarcEntry {
    pub url: String,
    pub method: String,        // default "GET"
    pub proto: String,         // default "HTTP/1.1"
    pub req_headers: Vec<(String, String)>,
    pub status_code: u16,
    pub status_text: String,   // default "{code} OK"
    pub resp_headers: Vec<(String, String)>,
    pub body: Vec<u8>,
    pub ip: String,            // "" = omit WARC-IP-Address
    pub crawled_at: chrono::DateTime<chrono::Utc>,
    pub run_id: String,        // for warcinfo isPartOf
}

/// Synchronous WARC 1.1 store backed by the filesystem.
#[derive(Clone, Debug)]
pub struct WarcStore {
    dir: PathBuf,
    compress: bool,
}

impl WarcStore {
    /// Open a store rooted at `dir`, creating it if needed.
    pub fn open(dir: impl AsRef<Path>, compress: bool) -> Result<Self> {
        let dir = dir.as_ref().to_path_buf();
        std::fs::create_dir_all(&dir)
            .with_context(|| format!("warcstore: mkdir {:?}", dir))?;
        Ok(Self { dir, compress })
    }

    pub fn compressed(&self) -> bool {
        self.compress
    }

    /// Write a WARC file for `entry` and return the response record's UUID (warc_id).
    /// Idempotent: if the file already exists, returns the UUID without re-writing.
    pub fn put(&self, entry: &WarcEntry) -> Result<String> {
        let canonical = canonical_url(&entry.url);
        let resp_uuid = Uuid::new_v5(ns_response(), canonical.as_bytes());
        let req_uuid  = Uuid::new_v5(ns_request(),  canonical.as_bytes());
        let info_uuid = Uuid::new_v5(ns_warcinfo(),  canonical.as_bytes());

        let warc_id = resp_uuid.to_string();
        let path = self.uuid_to_path(&warc_id);

        // Idempotent: skip if file already exists.
        if path.exists() {
            return Ok(warc_id);
        }

        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)
                .with_context(|| format!("warcstore: mkdir {:?}", parent))?;
        }

        // Defaults
        let method = if entry.method.is_empty() { "GET" } else { &entry.method };
        let proto  = if entry.proto.is_empty()  { "HTTP/1.1" } else { &entry.proto };
        let status_text = if entry.status_text.is_empty() {
            format!("{} OK", entry.status_code)
        } else {
            entry.status_text.clone()
        };
        let warc_date = entry.crawled_at.format("%Y-%m-%dT%H:%M:%SZ").to_string();

        // Relative path for WARC-Filename
        let rel_path = self.rel_path(&warc_id);

        // --- Build record blocks ---

        // 1. warcinfo block
        let mut warcinfo_block = Vec::new();
        writeln!(warcinfo_block, "software: go-mizu/warcstore\r")?;
        writeln!(warcinfo_block, "format: WARC File Format 1.1\r")?;
        writeln!(warcinfo_block, "conformsTo: https://iipc.github.io/warc-specifications/specifications/warc-format/warc-1.1/\r")?;
        if !entry.run_id.is_empty() {
            writeln!(warcinfo_block, "isPartOf: {}\r", entry.run_id)?;
        }

        // 2. request block
        let mut req_block = Vec::new();
        if let Ok(u) = url::Url::parse(&canonical) {
            let req_path = if u.query().is_some() {
                format!("{}?{}", u.path(), u.query().unwrap())
            } else {
                u.path().to_string()
            };
            let req_path = if req_path.is_empty() { "/".to_string() } else { req_path };
            writeln!(req_block, "{} {} {}\r", method, req_path, proto)?;
            writeln!(req_block, "Host: {}\r", u.host_str().unwrap_or(""))?;
        } else {
            writeln!(req_block, "{} / {}\r", method, proto)?;
        }
        let mut sorted_req = entry.req_headers.clone();
        sorted_req.sort_by(|a, b| a.0.cmp(&b.0));
        for (k, v) in &sorted_req {
            writeln!(req_block, "{}: {}\r", k, v)?;
        }
        req_block.extend_from_slice(b"\r\n");

        // 3. response block
        let mut resp_block = Vec::new();
        writeln!(resp_block, "{} {}\r", proto, status_text)?;
        let mut sorted_resp = entry.resp_headers.clone();
        sorted_resp.sort_by(|a, b| a.0.cmp(&b.0));
        for (k, v) in &sorted_resp {
            writeln!(resp_block, "{}: {}\r", k, v)?;
        }
        resp_block.extend_from_slice(b"\r\n");
        resp_block.extend_from_slice(&entry.body);

        // --- Assemble full WARC file ---
        let mut buf: Vec<u8> = Vec::new();

        write_warc_record(&mut buf, WarcRecordParams {
            typ: "warcinfo",
            id: &info_uuid.to_string(),
            date: &warc_date,
            target_uri: None,
            concurrent_to: None,
            filename: Some(&rel_path),
            ip: None,
            block: &warcinfo_block,
            content_type: "application/warc-fields",
        });
        buf.extend_from_slice(b"\r\n\r\n");

        write_warc_record(&mut buf, WarcRecordParams {
            typ: "request",
            id: &req_uuid.to_string(),
            date: &warc_date,
            target_uri: Some(&canonical),
            concurrent_to: Some(&resp_uuid.to_string()),
            filename: None,
            ip: if entry.ip.is_empty() { None } else { Some(&entry.ip) },
            block: &req_block,
            content_type: "application/http;msgtype=request",
        });
        buf.extend_from_slice(b"\r\n\r\n");

        write_warc_record(&mut buf, WarcRecordParams {
            typ: "response",
            id: &resp_uuid.to_string(),
            date: &warc_date,
            target_uri: Some(&canonical),
            concurrent_to: Some(&req_uuid.to_string()),
            filename: None,
            ip: if entry.ip.is_empty() { None } else { Some(&entry.ip) },
            block: &resp_block,
            content_type: "application/http;msgtype=response",
        });
        buf.extend_from_slice(b"\r\n\r\n");

        // --- Atomic write ---
        let tmp = path.with_extension(if self.compress { "warc.gz.tmp" } else { "warc.tmp" });
        if self.compress {
            let f = std::fs::File::create(&tmp)
                .with_context(|| format!("warcstore: create tmp {:?}", tmp))?;
            let mut gz = GzEncoder::new(f, Compression::fast());
            gz.write_all(&buf).context("warcstore: gzip write")?;
            gz.finish().context("warcstore: gzip finish")?;
        } else {
            std::fs::write(&tmp, &buf)
                .with_context(|| format!("warcstore: write tmp {:?}", tmp))?;
        }
        // Ignore rename error: concurrent worker may have written same UUID already.
        let _ = std::fs::rename(&tmp, &path);

        Ok(warc_id)
    }

    fn ext(&self) -> &'static str {
        if self.compress { ".warc.gz" } else { ".warc" }
    }

    pub fn uuid_to_path(&self, warc_id: &str) -> PathBuf {
        let hex: String = warc_id.chars().filter(|c| *c != '-').collect();
        self.dir
            .join(&hex[0..2])
            .join(&hex[2..4])
            .join(&hex[4..6])
            .join(format!("{}{}", warc_id, self.ext()))
    }

    fn rel_path(&self, warc_id: &str) -> String {
        let hex: String = warc_id.chars().filter(|c| *c != '-').collect();
        format!("{}/{}/{}/{}{}", &hex[0..2], &hex[2..4], &hex[4..6], warc_id, self.ext())
    }
}

struct WarcRecordParams<'a> {
    typ:           &'a str,
    id:            &'a str,
    date:          &'a str,
    target_uri:    Option<&'a str>,
    concurrent_to: Option<&'a str>,
    filename:      Option<&'a str>,
    ip:            Option<&'a str>,
    block:         &'a [u8],
    content_type:  &'a str,
}

fn write_warc_record(buf: &mut Vec<u8>, p: WarcRecordParams) {
    let _ = write!(buf, "WARC/1.1\r\n");
    let _ = write!(buf, "WARC-Type: {}\r\n", p.typ);
    let _ = write!(buf, "WARC-Date: {}\r\n", p.date);
    let _ = write!(buf, "WARC-Record-ID: <urn:uuid:{}>\r\n", p.id);
    if let Some(uri) = p.target_uri {
        let _ = write!(buf, "WARC-Target-URI: {}\r\n", uri);
    }
    if let Some(fname) = p.filename {
        let _ = write!(buf, "WARC-Filename: {}\r\n", fname);
    }
    if let Some(ct) = p.concurrent_to {
        let _ = write!(buf, "WARC-Concurrent-To: <urn:uuid:{}>\r\n", ct);
    }
    if let Some(ip) = p.ip {
        let _ = write!(buf, "WARC-IP-Address: {}\r\n", ip);
    }
    let _ = write!(buf, "Content-Type: {}\r\n", p.content_type);
    let _ = write!(buf, "Content-Length: {}\r\n", p.block.len());
    let _ = write!(buf, "\r\n");
    buf.extend_from_slice(p.block);
}

// ---------------------------------------------------------------------------
// Async wrapper
// ---------------------------------------------------------------------------

use crossbeam_channel::{bounded, Sender};
use dashmap::DashSet;
use std::sync::Arc;

/// Non-blocking WARC store wrapper.
///
/// `put_async` computes the WARC UUID synchronously (microseconds) and
/// returns it immediately; the actual WARC file write is offloaded to a
/// background thread.
pub struct AsyncWarcStore {
    pub(crate) inner: Arc<WarcStore>,
    /// Set of UUIDs (as raw bytes) that are written or in-flight. Prevents duplicate writes.
    in_flight: Arc<DashSet<uuid::Bytes>>,
    tx: Sender<Option<WarcEntry>>,
    handle: Mutex<Option<std::thread::JoinHandle<()>>>,
}

impl std::fmt::Debug for AsyncWarcStore {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "AsyncWarcStore")
    }
}

impl AsyncWarcStore {
    pub fn new(dir: impl AsRef<Path>, compress: bool) -> Result<Self> {
        let inner = Arc::new(WarcStore::open(dir, compress)?);
        let in_flight: Arc<DashSet<uuid::Bytes>> = Arc::new(DashSet::new());
        let (tx, rx) = bounded::<Option<WarcEntry>>(4096);

        let inner2 = Arc::clone(&inner);

        let handle = std::thread::Builder::new()
            .name("warc-store-writer".into())
            .spawn(move || {
                for msg in rx.iter() {
                    let entry = match msg {
                        Some(e) => e,
                        None => break,
                    };
                    let _ = inner2.put(&entry);
                }
            })
            .context("failed to spawn warc-store-writer thread")?;

        Ok(Self {
            inner,
            in_flight,
            tx,
            handle: Mutex::new(Some(handle)),
        })
    }

    /// Returns whether the store writes compressed (.warc.gz) files.
    pub fn compressed(&self) -> bool {
        self.inner.compressed()
    }

    /// Compute warc_id synchronously; schedule WARC write asynchronously.
    /// Returns the UUID string immediately.
    pub fn put_async(&self, entry: WarcEntry) -> String {
        let canonical = canonical_url(&entry.url);
        let resp_uuid = Uuid::new_v5(ns_response(), canonical.as_bytes());
        let warc_id = resp_uuid.to_string();
        let key: uuid::Bytes = *resp_uuid.as_bytes();

        // Already in-flight or written: return immediately.
        if self.in_flight.contains(&key) {
            return warc_id;
        }
        if self.inner.uuid_to_path(&warc_id).exists() {
            self.in_flight.insert(key);
            return warc_id;
        }

        self.in_flight.insert(key);

        // try_send: fall back to sync write if channel is full.
        if self.tx.try_send(Some(entry.clone())).is_err() {
            let _ = self.inner.put(&entry);
        }

        warc_id
    }

    /// Flush all pending writes. Blocks until the background thread finishes.
    pub fn close(&self) -> Result<()> {
        let _ = self.tx.send(None);
        if let Ok(mut guard) = self.handle.lock() {
            if let Some(h) = guard.take() {
                h.join()
                    .map_err(|_| anyhow::anyhow!("warc-store-writer thread panicked"))?;
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    fn sample_entry() -> WarcEntry {
        WarcEntry {
            url: "https://example.com/page".to_string(),
            method: "GET".to_string(),
            proto: "HTTP/1.1".to_string(),
            req_headers: vec![("Accept".to_string(), "text/html".to_string())],
            status_code: 200,
            status_text: "200 OK".to_string(),
            resp_headers: vec![("Content-Type".to_string(), "text/html; charset=utf-8".to_string())],
            body: b"<html><body>hello</body></html>".to_vec(),
            ip: "93.184.216.34".to_string(),
            crawled_at: chrono::DateTime::parse_from_rfc3339("2024-01-15T12:00:00Z")
                .unwrap()
                .with_timezone(&chrono::Utc),
            run_id: "test-run-001".to_string(),
        }
    }

    #[test]
    fn test_canonical_url() {
        assert_eq!(canonical_url("https://Example.COM/path"), "https://example.com/path");
        assert_eq!(canonical_url("http://example.com:80/path"), "http://example.com/path");
        assert_eq!(canonical_url("https://example.com:443/path"), "https://example.com/path");
        assert_eq!(canonical_url("https://example.com:8080/path"), "https://example.com:8080/path");
        assert_eq!(canonical_url("https://example.com/path#fragment"), "https://example.com/path");
    }

    #[test]
    fn test_put_creates_file() {
        let tmp = tempdir().unwrap();
        let store = WarcStore::open(tmp.path(), false).unwrap();
        let entry = sample_entry();
        let warc_id = store.put(&entry).unwrap();

        assert!(!warc_id.is_empty(), "warc_id must not be empty");

        let path = store.uuid_to_path(&warc_id);
        assert!(path.exists(), "WARC file must exist: {:?}", path);

        let content = std::fs::read_to_string(&path).unwrap();
        assert!(content.contains("WARC/1.1"));
        assert!(content.contains("WARC-Type: warcinfo"));
        assert!(content.contains("WARC-Type: request"));
        assert!(content.contains("WARC-Type: response"));
        assert!(content.contains("WARC-IP-Address: 93.184.216.34"));
    }

    #[test]
    fn test_put_idempotent() {
        let tmp = tempdir().unwrap();
        let store = WarcStore::open(tmp.path(), false).unwrap();
        let entry = sample_entry();

        let id1 = store.put(&entry).unwrap();
        let path = store.uuid_to_path(&id1);
        let mtime1 = std::fs::metadata(&path).unwrap().modified().unwrap();

        let id2 = store.put(&entry).unwrap();
        let mtime2 = std::fs::metadata(&path).unwrap().modified().unwrap();

        assert_eq!(id1, id2);
        assert_eq!(mtime1, mtime2, "file must not be rewritten on second put");
    }

    #[test]
    fn test_put_deterministic() {
        let tmp = tempdir().unwrap();
        let store = WarcStore::open(tmp.path(), false).unwrap();
        let e1 = sample_entry();
        let mut e2 = e1.clone();
        e2.url = "https://example.com/other".to_string();

        let id1 = store.put(&e1).unwrap();
        let id2 = store.put(&e2).unwrap();
        assert_ne!(id1, id2, "different URLs must produce different warc_ids");
    }

    #[test]
    fn test_put_compress() {
        let tmp = tempdir().unwrap();
        let store = WarcStore::open(tmp.path(), true).unwrap();
        let entry = sample_entry();
        let warc_id = store.put(&entry).unwrap();

        let path = store.uuid_to_path(&warc_id);
        assert!(path.to_str().unwrap().ends_with(".warc.gz"), "should be .warc.gz: {:?}", path);
        assert!(path.exists());

        // Verify it's valid gzip
        let f = std::fs::File::open(&path).unwrap();
        use flate2::read::GzDecoder;
        use std::io::Read;
        let mut gz = GzDecoder::new(f);
        let mut content = String::new();
        gz.read_to_string(&mut content).unwrap();
        assert!(content.contains("WARC/1.1"));
        assert!(content.contains("WARC-Type: response"));
    }

    #[test]
    fn test_async_warc_store() {
        let tmp = tempdir().unwrap();
        let store = Arc::new(AsyncWarcStore::new(tmp.path(), false).unwrap());
        let entry = sample_entry();
        let warc_id = store.put_async(entry);
        assert!(!warc_id.is_empty());
        store.close().unwrap();

        // File should exist after close()
        let path = store.inner.uuid_to_path(&warc_id);
        assert!(path.exists(), "WARC file must exist after close: {:?}", path);
    }
}
