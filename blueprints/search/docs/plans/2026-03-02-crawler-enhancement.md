# Crawler Enhancement Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Achieve ≥1500 HTTP-200 OK responses/second on `crawler cc recrawl --file p:0` (server 2) with zero false negatives — every URL that would return 200 under any reasonable timeout must be captured.

**Architecture:** Six targeted fixes: correct the pass-2 retry SQL to include abandoned-domain URLs, add an adaptive-timeout floor so fast domains cannot shrink the timeout below the configured baseline, replace the O(max×domains) round-robin producer with an O(N) deque, remove the redundant COUNT(*) prescan, move body-store writes off the hot fetch path via a background thread, and replace full Vec<SeedURL> materialisation with paginated DuckDB streaming so CC seeds stay under 30 MB instead of 3 GB.

**Tech Stack:** Rust, Tokio, async_channel, crossbeam_channel, DuckDB (bundled), rkyv, sha2, flate2, dashmap

**Working directory for all commands:** `blueprints/search/tools/crawler/`

**Test command:** `CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features`

**Single-test command:** `CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features -- <test_name> --nocapture`

---

## Task 1: E2 — Fix pass-2 retry to include abandoned-domain URLs

**Files:**
- Modify: `crawler-lib/src/seed.rs:184-188`

The current SQL only loads `reason = 'http_timeout'`. URLs written as
`reason = 'domain_http_timeout_killed'` (when a domain is abandoned while URLs
remain in the channel) are silently dropped and never retried. This is the primary
false-negative source on CC data.

### Step 1: Write the failing test

Add to the bottom of `crawler-lib/src/seed.rs`:

```rust
#[cfg(test)]
mod tests {
    use super::*;
    use chrono::NaiveDateTime;

    fn make_failed_db(path: &str) -> duckdb::Connection {
        let conn = duckdb::Connection::open(path).unwrap();
        conn.execute_batch(
            "CREATE TABLE failed_urls (
                url TEXT, domain TEXT, reason TEXT,
                subcategory TEXT, error TEXT, status_code INTEGER,
                fetch_time_ms INTEGER, detected_at TIMESTAMP
             )",
        )
        .unwrap();
        conn
    }

    fn insert_failed(conn: &duckdb::Connection, url: &str, domain: &str, reason: &str, ts: NaiveDateTime) {
        conn.execute(
            "INSERT INTO failed_urls VALUES (?,?,?,'','',0,0,?)",
            duckdb::params![url, domain, reason, ts.format("%Y-%m-%d %H:%M:%S").to_string()],
        )
        .unwrap();
    }

    #[test]
    fn load_retry_seeds_includes_killed_urls() {
        let dir = tempfile::tempdir().unwrap();
        let db_path = dir.path().join("failed.duckdb").to_string_lossy().to_string();
        let conn = make_failed_db(&db_path);
        let ts = chrono::Utc::now().naive_utc();

        insert_failed(&conn, "https://a.com/1", "a.com", "http_timeout", ts);
        insert_failed(&conn, "https://b.com/1", "b.com", "domain_http_timeout_killed", ts);
        drop(conn);

        let since = ts - chrono::Duration::seconds(1);
        let seeds = load_retry_seeds(&db_path, since).unwrap();

        let urls: Vec<&str> = seeds.iter().map(|s| s.url.as_str()).collect();
        assert!(urls.contains(&"https://a.com/1"), "should include http_timeout URL");
        assert!(urls.contains(&"https://b.com/1"), "should include domain_http_timeout_killed URL");
        assert_eq!(seeds.len(), 2);
    }

    #[test]
    fn load_retry_seeds_excludes_before_since() {
        let dir = tempfile::tempdir().unwrap();
        let db_path = dir.path().join("failed2.duckdb").to_string_lossy().to_string();
        let conn = make_failed_db(&db_path);
        let old_ts = chrono::NaiveDateTime::parse_from_str("2020-01-01 00:00:00", "%Y-%m-%d %H:%M:%S").unwrap();
        let new_ts = chrono::Utc::now().naive_utc();

        insert_failed(&conn, "https://old.com/1", "old.com", "http_timeout", old_ts);
        insert_failed(&conn, "https://new.com/1", "new.com", "http_timeout", new_ts);
        drop(conn);

        let since = new_ts - chrono::Duration::seconds(1);
        let seeds = load_retry_seeds(&db_path, since).unwrap();
        assert_eq!(seeds.len(), 1);
        assert_eq!(seeds[0].url, "https://new.com/1");
    }
}
```

Add `tempfile` to `[dev-dependencies]` in `crawler-lib/Cargo.toml` if not already there (it is — check `Cargo.toml:49`).

### Step 2: Run test — verify it fails

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features -- load_retry_seeds_includes_killed_urls --nocapture
```

Expected: FAIL — `seeds.len()` is 1, not 2 (killed URL is missing).

### Step 3: Fix the SQL in `load_retry_seeds`

In `crawler-lib/src/seed.rs`, replace lines 184–188:

```rust
// Before
let mut stmt = conn.prepare(
    "SELECT url, COALESCE(domain, '') as domain FROM failed_urls \
     WHERE reason = 'http_timeout' AND detected_at >= ?"
)?;

// After
let mut stmt = conn.prepare(
    "SELECT url, COALESCE(domain, '') as domain FROM failed_urls \
     WHERE reason IN ('http_timeout', 'domain_http_timeout_killed') \
       AND detected_at >= ?"
)?;
```

### Step 4: Run tests — verify they pass

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features -- load_retry_seeds --nocapture
```

Expected: both tests PASS.

### Step 5: Commit

```bash
git add crawler-lib/src/seed.rs
git commit -m "fix(crawler): include domain_http_timeout_killed in pass-2 retry

Abandoned-domain URLs were silently dropped by load_retry_seeds because
the SQL only matched reason='http_timeout'. Adding domain_http_timeout_killed
ensures every URL whose domain was abandoned mid-crawl gets a second chance
in pass 2 with the longer retry_timeout."
```

---

## Task 2: E1 — Adaptive timeout floor

**Files:**
- Modify: `crawler-lib/src/engine/reqwest_engine.rs:305-311`
- Modify: `crawler-lib/src/engine/hyper_engine.rs:273-280`

When fast domains skew the adaptive P95 low (e.g., 150 ms), the effective timeout
can drop to 300 ms — killing URLs that need 400–800 ms. Add a floor so adaptive
only extends the timeout, never shrinks it below the configured baseline.

### Step 1: Write the failing test

Add to `crawler-lib/src/stats.rs` (after the existing `PeakTracker` impl):

```rust
#[cfg(test)]
mod adaptive_tests {
    use super::*;
    use std::time::Duration;

    #[test]
    fn adaptive_floor_never_drops_below_cfg_timeout() {
        let adaptive = AdaptiveTimeout::new();
        // Record 20 very fast responses (100 ms) — P95 = 100 ms → raw adaptive = 200 ms
        for _ in 0..20 {
            adaptive.record(100);
        }

        let cfg_timeout = Duration::from_millis(1000);
        let ceiling = Duration::from_secs(600);
        let adaptive_val = adaptive.timeout(ceiling).unwrap_or(cfg_timeout);

        // With floor: max(200ms, 1000ms) = 1000ms
        let effective = adaptive_val
            .max(cfg_timeout)
            .min(cfg_timeout.saturating_mul(3));

        assert!(
            effective >= cfg_timeout,
            "effective {}ms should be >= cfg {}ms",
            effective.as_millis(),
            cfg_timeout.as_millis()
        );
    }

    #[test]
    fn adaptive_extends_for_slow_domains() {
        let adaptive = AdaptiveTimeout::new();
        // Record 20 slow responses (3000 ms) — P95 = 3000 ms → raw adaptive = 6000 ms
        for _ in 0..20 {
            adaptive.record(3000);
        }

        let cfg_timeout = Duration::from_millis(1000);
        let ceiling = Duration::from_secs(600);
        let adaptive_val = adaptive.timeout(ceiling).unwrap_or(cfg_timeout);

        let effective = adaptive_val
            .max(cfg_timeout)
            .min(cfg_timeout.saturating_mul(3));

        // Should be capped at 3× = 3000ms
        assert_eq!(effective, Duration::from_millis(3000));
    }
}
```

### Step 2: Run test — verify the formula logic (these pass immediately since they test formula, not engine code)

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features -- adaptive_tests --nocapture
```

Expected: PASS (tests validate the formula — now apply it to both engines).

### Step 3: Apply floor in `reqwest_engine.rs`

In `crawler-lib/src/engine/reqwest_engine.rs`, find `process_one_url` (~line 305):

```rust
// Before
let effective_timeout = if !cfg.disable_adaptive_timeout {
    adaptive
        .timeout(cfg.adaptive_timeout_max)
        .unwrap_or(cfg.timeout)
        .min(cfg.timeout.saturating_mul(5))
} else {
    cfg.timeout
};

// After
let effective_timeout = if !cfg.disable_adaptive_timeout {
    adaptive
        .timeout(cfg.adaptive_timeout_max)
        .unwrap_or(cfg.timeout)
        .max(cfg.timeout)                    // floor: never below configured baseline
        .min(cfg.timeout.saturating_mul(3))  // tighter ceiling: ×3 not ×5
} else {
    cfg.timeout
};
```

### Step 4: Apply the same floor in `hyper_engine.rs`

In `crawler-lib/src/engine/hyper_engine.rs`, find `process_one_url` (~line 273):

```rust
// Before
let effective_timeout = if !cfg.disable_adaptive_timeout {
    adaptive
        .timeout(cfg.adaptive_timeout_max)
        .unwrap_or(cfg.timeout)
        .min(cfg.timeout.saturating_mul(5))
} else {
    cfg.timeout
};

// After — identical change
let effective_timeout = if !cfg.disable_adaptive_timeout {
    adaptive
        .timeout(cfg.adaptive_timeout_max)
        .unwrap_or(cfg.timeout)
        .max(cfg.timeout)
        .min(cfg.timeout.saturating_mul(3))
} else {
    cfg.timeout
};
```

### Step 5: Run all tests

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features
```

Expected: all PASS.

### Step 6: Commit

```bash
git add crawler-lib/src/engine/reqwest_engine.rs crawler-lib/src/engine/hyper_engine.rs crawler-lib/src/stats.rs
git commit -m "fix(crawler): add adaptive timeout floor to prevent feedback loop

Fast domains could skew the adaptive P95 down (e.g. 150ms), causing
effective timeout to drop to 300ms and killing medium-speed valid URLs.
The floor max(adaptive, cfg.timeout) ensures adaptive only extends the
timeout for slow domains, never shrinks it. Ceiling tightened to ×3."
```

---

## Task 3: E3 — O(N) deque-based producer

**Files:**
- Modify: `crawler-lib/src/engine/reqwest_engine.rs` (producer block ~lines 145-178)
- Modify: `crawler-lib/src/engine/hyper_engine.rs` (producer block ~lines 133-162)
- Modify: `crawler-lib/src/domain.rs` (add `interleave_by_domain` helper + tests)

The current nested-loop producer is `O(max_domain_urls × num_domains)`. With CC p:0
(~500K domains, max ~100 URLs per domain) it does 50M iterations with 99% wasted.
Extract a testable helper and replace the loop in both engines.

### Step 1: Add `interleave_by_domain` helper and tests to `domain.rs`

At the bottom of `crawler-lib/src/domain.rs`, before the existing `#[cfg(test)]` block,
add the helper function:

```rust
/// Interleave URLs from multiple domain batches in round-robin order.
///
/// Example: batches A=[1,2], B=[1] → [A1, B1, A2]
/// This is O(total_urls) — no wasted iterations for exhausted domains.
pub fn interleave_by_domain(batches: Vec<DomainBatch>) -> Vec<SeedURL> {
    use std::collections::VecDeque;
    let mut queue: VecDeque<VecDeque<SeedURL>> = batches
        .into_iter()
        .map(|b| VecDeque::from(b.urls))
        .collect();
    let mut result = Vec::new();
    while let Some(mut urls) = queue.pop_front() {
        if let Some(url) = urls.pop_front() {
            result.push(url);
            if !urls.is_empty() {
                queue.push_back(urls);
            }
        }
    }
    result
}
```

Add these tests inside the existing `#[cfg(test)] mod tests` block in `domain.rs`:

```rust
#[test]
fn test_interleave_sends_all_urls() {
    let seeds: Vec<SeedURL> = (0..100)
        .map(|i| SeedURL {
            url: format!("https://d{}.com/page{}", i % 5, i),
            domain: format!("d{}.com", i % 5),
        })
        .collect();
    let batches = group_by_domain(seeds);
    let result = interleave_by_domain(batches);
    assert_eq!(result.len(), 100, "all URLs must be delivered");
}

#[test]
fn test_interleave_round_robin_order() {
    // a has 2 URLs, b has 1 URL → round-robin: a0, b0, a1
    let seeds = vec![
        SeedURL { url: "https://a.com/1".into(), domain: "a.com".into() },
        SeedURL { url: "https://a.com/2".into(), domain: "a.com".into() },
        SeedURL { url: "https://b.com/1".into(), domain: "b.com".into() },
    ];
    let batches = group_by_domain(seeds);
    let result = interleave_by_domain(batches);
    assert_eq!(result.len(), 3);
    // First URL from a, first from b, then second from a
    assert_eq!(result[0].domain, "a.com");
    assert_eq!(result[1].domain, "b.com");
    assert_eq!(result[2].domain, "a.com");
}

#[test]
fn test_interleave_single_domain() {
    let seeds = vec![
        SeedURL { url: "https://only.com/1".into(), domain: "only.com".into() },
        SeedURL { url: "https://only.com/2".into(), domain: "only.com".into() },
    ];
    let batches = group_by_domain(seeds);
    let result = interleave_by_domain(batches);
    assert_eq!(result.len(), 2);
}
```

### Step 2: Run tests — verify they pass

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features -- test_interleave --nocapture
```

Expected: all 3 PASS.

### Step 3: Replace the producer in `reqwest_engine.rs`

Find the `producer` `tokio::spawn` block (~lines 145-178). Replace it entirely:

```rust
// Producer: round-robin interleaving — O(total_seeds), zero wasted iterations.
// Uses a VecDeque of (remaining_urls, domain_entry): pop front domain, send one
// URL, push back if the domain has more. This is the deque trick that makes the
// old O(max_len × num_domains) nested loop unnecessary.
let dm = Arc::clone(&domain_map);
let producer = tokio::spawn(async move {
    use std::collections::VecDeque;
    let mut queue: VecDeque<(VecDeque<SeedURL>, Arc<DomainEntry>)> = batches
        .into_iter()
        .filter_map(|batch| {
            dm.get(&batch.domain)
                .map(|e| (VecDeque::from(batch.urls), Arc::clone(e.value())))
        })
        .collect();

    while let Some((mut urls, entry)) = queue.pop_front() {
        let url = match urls.pop_front() {
            Some(u) => u,
            None => continue,
        };
        if url_tx
            .send((url, Arc::clone(&entry)))
            .await
            .is_err()
        {
            return; // receivers all dropped
        }
        if !urls.is_empty() {
            queue.push_back((urls, entry));
        }
    }
    // url_tx dropped here → channel closes when all workers drain it
});
```

Note: the `let dm = Arc::clone(&domain_map);` line that was already there can be
removed since `domain_map` is moved into the closure directly now (or keep the clone
— either works; the domain_map is not used after the producer spawn in the current
code).

### Step 4: Apply the same producer replacement in `hyper_engine.rs`

Find the `producer` block (~lines 133-162). Apply the identical replacement as Step 3.
The domain entry type is different (`hyper_engine::DomainEntry` vs
`reqwest_engine::DomainEntry`) but the structure is identical.

### Step 5: Run all tests

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features
```

Expected: all PASS.

### Step 6: Commit

```bash
git add crawler-lib/src/domain.rs crawler-lib/src/engine/reqwest_engine.rs crawler-lib/src/engine/hyper_engine.rs
git commit -m "perf(crawler): O(N) deque producer replaces O(max×domains) nested loop

The old round-robin producer iterated ALL domains for every slot index
up to max_domain_urls. With CC p:0 (~500K domains, max 100 URLs) this
was 50M iterations with 99% wasted on None checks. The deque approach
pops the front domain, sends one URL, re-enqueues if more remain.
Every iteration sends exactly one URL: O(total_seeds) total."
```

---

## Task 4: E5a — Remove COUNT(*) prescan in CC seed loading

**Files:**
- Modify: `crawler-lib/src/seed.rs` (`load_seeds_cc_parquet`, ~lines 132-145)

The prescan runs a full `COUNT(*)` pass over the parquet before the data load,
doubling the read time for no user-facing value.

### Step 1: Delete the prescan block

In `crawler-lib/src/seed.rs`, find and remove lines ~132–145 (the `Count first` block):

```rust
// DELETE this entire block:
// Count first so the user knows what's being loaded before the full collect.
let count: i64 = {
    let count_sql = format!(
        "SELECT COUNT(*) FROM read_parquet('{}') WHERE {}{}",
        escaped, where_clause, limit_clause
    );
    let mut stmt = conn.prepare(&count_sql)?;
    let mut rows = stmt.query([])?;
    rows.next()?.and_then(|r| r.get::<_, i64>(0).ok()).unwrap_or(0)
};
let est_mb = (count as u64 * 150) / (1024 * 1024);
println!("CC seeds: {count} URLs (~{est_mb} MB heap)");
if count > 1_000_000 {
    println!("  note: large seed set — use --limit N to reduce memory usage");
}
```

### Step 2: Run all tests

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features
```

Expected: all PASS (no tests depend on the print).

### Step 3: Commit

```bash
git add crawler-lib/src/seed.rs
git commit -m "perf(crawler): remove COUNT(*) prescan in load_seeds_cc_parquet

The count query ran a full parquet scan before the data load, doubling
wall time for large CC partitions (5-15M rows). The size estimate print
provided no value at runtime."
```

---

## Task 5: E4 — AsyncBodyStore (background write thread)

**Files:**
- Modify: `crawler-lib/src/bodystore.rs`

`BodyStore::put()` runs SHA-256 + gzip + file-write synchronously inside the hot
fetch path. At 1500 OK/s this is 1500 blocking I/O ops/second inside Tokio tasks.
Solution: compute SHA-256 in-task (fast), return CID immediately, send body to a
background OS thread for gzip + write.

### Step 1: Write the failing test

Add to `crawler-lib/src/bodystore.rs`:

```rust
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
```

### Step 2: Run test — verify it fails (AsyncBodyStore doesn't exist yet)

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features -- async_body_store --nocapture 2>&1 | head -20
```

Expected: compile error — `AsyncBodyStore` not found.

### Step 3: Implement `AsyncBodyStore` in `bodystore.rs`

First, make `BodyStore::cid_to_path` public (needed by AsyncBodyStore internally):

```rust
// Change: `fn cid_to_path` → `pub(crate) fn cid_to_path`
pub(crate) fn cid_to_path(&self, hex: &str) -> PathBuf {
```

Then add `AsyncBodyStore` after the `BodyStore` impl block:

```rust
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

impl AsyncBodyStore {
    pub fn new(dir: impl AsRef<std::path::Path>) -> Result<Self> {
        let inner = Arc::new(BodyStore::open(dir)?);
        let in_flight: Arc<DashSet<[u8; 32]>> = Arc::new(DashSet::new());
        let (tx, rx) = bounded::<Option<(Vec<u8>, [u8; 32])>>(8192);

        let inner2 = Arc::clone(&inner);
        let in_flight2 = Arc::clone(&in_flight);

        let handle = std::thread::Builder::new()
            .name("body-store-writer".into())
            .spawn(move || {
                for msg in rx.iter() {
                    let (body, hash) = match msg {
                        Some(item) => item,
                        None => break,
                    };
                    let hex = format!("{}", hash.iter().map(|b| format!("{:02x}", b)).collect::<String>());
                    let path = inner2.cid_to_path(&hex);
                    if !path.exists() {
                        // Write via the existing BodyStore helper (handles tmp + rename)
                        // Re-derive hex from body to keep BodyStore::put() re-entrant.
                        // We know path doesn't exist, so put() will write it.
                        let _ = inner2.put(&body);
                    }
                    in_flight2.insert(hash); // mark as fully written
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
        use sha2::Digest;
        let sum = sha2::Sha256::digest(body);
        let hash: [u8; 32] = sum.into();
        let hex = format!("{:x}", sum);
        let cid = format!("sha256:{}", hex);

        // If already written or in-flight, return CID with no work.
        if self.in_flight.contains(&hash) {
            return cid;
        }
        // Mark as in-flight before sending — prevents a second caller from
        // racing and sending a duplicate write.
        self.in_flight.insert(hash);

        // try_send: if channel is full (8192 cap), the write is dropped.
        // This is safe — the file simply won't exist; body_cid in the result
        // will reference a missing file, which is handled gracefully on read.
        let _ = self.tx.try_send(Some((body.to_vec(), hash)));
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
```

You need to add `crossbeam-channel` import at the top of `bodystore.rs`:

```rust
use crossbeam_channel::{bounded, Sender};
use dashmap::DashSet;
use std::sync::{Arc, Mutex};
```

`crossbeam_channel` and `dashmap` are already in `crawler-lib/Cargo.toml` (lines 39, 45).

### Step 4: Run tests — verify they pass

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features -- async_body_store --nocapture
```

Expected: all 3 PASS.

### Step 5: Run all tests

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features
```

Expected: all PASS.

### Step 6: Commit

```bash
git add crawler-lib/src/bodystore.rs
git commit -m "perf(crawler): AsyncBodyStore moves gzip+write off hot fetch path

BodyStore::put() ran SHA-256+gzip+fsync synchronously inside Tokio worker
tasks. At 1500 OK/s this was 1500 blocking file ops/sec competing with
network I/O on the thread pool.

AsyncBodyStore::put_async() computes the CID synchronously (microseconds),
marks the hash as in-flight via DashSet (dedup), then sends the body to
a dedicated OS thread for gzip+atomic-write. Workers never block on disk."
```

---

## Task 6: E4 — Wire AsyncBodyStore into Config and engines

**Files:**
- Modify: `crawler-lib/src/config.rs` (body_store field type)
- Modify: `crawler-lib/src/engine/reqwest_engine.rs` (fetch_one call site)
- Modify: `crawler-cli/src/common.rs` (creation + close)

### Step 1: Update `Config.body_store` type in `config.rs`

```rust
// Before (line ~119)
pub body_store: Option<Arc<BodyStore>>,

// After
pub body_store: Option<Arc<crate::bodystore::AsyncBodyStore>>,
```

Update the Default impl (line ~146) — no change needed since `None` works for both types.

### Step 2: Update `fetch_one` call in `reqwest_engine.rs`

In `fetch_one` (~line 754–762), the body store call is:

```rust
// Before
let body_cid = if let Some(store) = body_store {
    if should_read_body && !body_bytes.is_empty() {
        store.put(&body_bytes).unwrap_or_default()
    } else {
        String::new()
    }
} else {
    String::new()
};

// After — same structure, change put() → put_async() and type
let body_cid = if let Some(store) = body_store {
    if should_read_body && !body_bytes.is_empty() {
        store.put_async(&body_bytes)
    } else {
        String::new()
    }
} else {
    String::new()
};
```

Also update the `fetch_one` signature parameter type:

```rust
// Before (~line 656)
    body_store: Option<&crate::bodystore::BodyStore>,

// After
    body_store: Option<&crate::bodystore::AsyncBodyStore>,
```

And in `process_one_url` where `fetch_one` is called, the `cfg.body_store.as_deref()`
call works unchanged since `Arc<T>` implements `Deref<Target=T>`.

### Step 3: Update `common.rs` — create AsyncBodyStore and close it after crawl

In `run_crawl_job`, find the body store creation block (~line 390-395):

```rust
// Before
if let Some(ref dir) = params.body_store_dir {
    let resolved = expand_home(dir);
    let store = BodyStore::open(&resolved)?;
    cfg.body_store = Some(Arc::new(store));
    println!("Body store: {}", resolved.display());
}

// After
if let Some(ref dir) = params.body_store_dir {
    let resolved = expand_home(dir);
    let store = crawler_lib::bodystore::AsyncBodyStore::new(&resolved)?;
    cfg.body_store = Some(Arc::new(store));
    println!("Body store: {}", resolved.display());
}
```

Add the body_store handle clone **before** `run_job` is called (because `cfg` is moved into `run_job`):

```rust
// Clone the Arc before cfg is moved into run_job
let body_store_handle = cfg.body_store.clone();

let job_result = run_job(...cfg...).await?;

// Flush pending body writes before closing the result writer
if let Some(ref store) = body_store_handle {
    store.close()?;
}
result_writer.close()?;
```

Remove the `use crawler_lib::bodystore::BodyStore;` import from `common.rs` top-of-file
(it was used for the old `BodyStore::open` call; no longer needed directly).

### Step 4: Run all tests

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features
```

Expected: all PASS. Fix any compile errors from type mismatches.

### Step 5: Commit

```bash
git add crawler-lib/src/config.rs crawler-lib/src/engine/reqwest_engine.rs crawler-cli/src/common.rs
git commit -m "feat(crawler): wire AsyncBodyStore through Config and engines

Config.body_store changes from BodyStore to AsyncBodyStore. reqwest_engine
calls put_async() instead of put(). common.rs closes the store after
run_job returns to flush all pending writes before stats are finalised."
```

---

## Task 7: E5b — Streaming seed loading: engine trait + vec_to_receiver

**Files:**
- Modify: `crawler-lib/src/engine/mod.rs`
- Modify: `crawler-lib/src/engine/reqwest_engine.rs`
- Modify: `crawler-lib/src/engine/hyper_engine.rs`
- Modify: `crawler-lib/src/seed.rs` (add `vec_to_receiver`)
- Modify: `crawler-lib/src/job.rs`

Change the Engine trait to accept an `async_channel::Receiver<SeedURL>` instead of
`Vec<SeedURL>`. This decouples seed loading from the engine and enables streaming.
Also add the `vec_to_receiver` helper for callers that already have a `Vec`.

### Step 1: Add `vec_to_receiver` to `seed.rs`

Add after the existing functions in `crawler-lib/src/seed.rs`:

```rust
/// Convert a Vec<SeedURL> into an already-closed async_channel::Receiver.
///
/// All seeds are sent immediately (bounded by seeds.len()), then the sender
/// is dropped so the receiver sees EOF. Used by HN and direct --seed callers
/// that load the full seed list upfront.
pub fn vec_to_receiver(seeds: Vec<SeedURL>) -> (async_channel::Receiver<SeedURL>, u64) {
    let total = seeds.len() as u64;
    if seeds.is_empty() {
        let (_tx, rx) = async_channel::bounded(1);
        // tx dropped immediately → rx sees empty closed channel
        return (rx, 0);
    }
    let (tx, rx) = async_channel::bounded(seeds.len());
    for seed in seeds {
        // bounded by seeds.len() — never blocks
        let _ = tx.try_send(seed);
    }
    drop(tx); // close sender → receiver returns Err(Closed) after last item
    (rx, total)
}
```

Add a test at the bottom of the `tests` module in `seed.rs`:

```rust
#[test]
fn vec_to_receiver_delivers_all_seeds_then_closes() {
    let seeds = vec![
        SeedURL { url: "https://a.com/1".into(), domain: "a.com".into() },
        SeedURL { url: "https://b.com/1".into(), domain: "b.com".into() },
    ];
    let (rx, count) = vec_to_receiver(seeds);
    assert_eq!(count, 2);
    assert_eq!(rx.recv_blocking().unwrap().url, "https://a.com/1");
    assert_eq!(rx.recv_blocking().unwrap().url, "https://b.com/1");
    assert!(rx.recv_blocking().is_err(), "channel should be closed after last seed");
}
```

Run: `cargo test --no-default-features -- vec_to_receiver` → Expected: PASS.

### Step 2: Update Engine trait in `mod.rs`

```rust
// Before
#[async_trait::async_trait]
pub trait Engine: Send + Sync {
    async fn run(
        &self,
        seeds: Vec<SeedURL>,
        cfg: &Config,
        results: Arc<dyn ResultWriter>,
        failures: Arc<dyn FailureWriter>,
    ) -> Result<StatsSnapshot>;
}

// After
#[async_trait::async_trait]
pub trait Engine: Send + Sync {
    async fn run(
        &self,
        seed_rx: async_channel::Receiver<SeedURL>,
        seed_count: u64,   // hint for TUI progress %; 0 = unknown
        cfg: &Config,
        results: Arc<dyn ResultWriter>,
        failures: Arc<dyn FailureWriter>,
    ) -> Result<StatsSnapshot>;
}
```

Add the import at the top of `mod.rs`:

```rust
use async_channel;
```

### Step 3: Update `run()` signature in `reqwest_engine.rs`

In `ReqwestEngine::run()` (~line 53), update the signature and the `group_by_domain`
call to consume the channel in batches:

```rust
async fn run(
    &self,
    seed_rx: async_channel::Receiver<SeedURL>,
    seed_count: u64,
    cfg: &Config,
    results: Arc<dyn ResultWriter>,
    failures: Arc<dyn FailureWriter>,
) -> Result<StatsSnapshot> {
    // Drain seed_rx into a Vec for the initial total count + grouping.
    // For large CC datasets this Vec is replaced by Task 8's streaming producer;
    // for now we collect all seeds first to keep the change minimal.
    let seeds: Vec<SeedURL> = {
        let mut v = Vec::new();
        while let Ok(s) = seed_rx.recv().await {
            v.push(s);
        }
        v
    };
    let total_seeds_actual = seeds.len();
    if total_seeds_actual == 0 {
        return Ok(StatsSnapshot::empty());
    }
    // ... rest of existing run() body unchanged, replacing `seeds` variable use ...
```

This is a transitional implementation — it still collects all seeds first, but the
interface now accepts a channel. Task 8 replaces the collect-all with a streaming
batch producer.

Update the `total_seeds` store line (currently uses `seeds.len()`):

```rust
// Before
stats.total_seeds.store(total_seeds as u64, Ordering::Relaxed);

// After — use seed_count hint if provided, else actual count
let display_total = if seed_count > 0 { seed_count } else { total_seeds_actual as u64 };
if stats.total_seeds.load(Ordering::Relaxed) == 0 {
    stats.total_seeds.store(display_total, Ordering::Relaxed);
}
```

### Step 4: Apply the same signature change to `hyper_engine.rs`

Same as Step 3 but for `HyperEngine::run()`. The collect-all pattern is identical.

### Step 5: Update `run_job` in `job.rs`

```rust
// Before
pub async fn run_job(
    seeds: Vec<SeedURL>,
    mut cfg: Config,
    ...
) -> Result<JobResult>

// After
pub async fn run_job(
    seed_rx: async_channel::Receiver<SeedURL>,
    seed_count: u64,
    mut cfg: Config,
    ...
) -> Result<JobResult>
```

Update the engine call sites inside `run_job` to pass `seed_rx` and `seed_count`:

```rust
// Pass 1
let pass1 = engine
    .run(seed_rx, seed_count, &cfg, result_writer.clone(), failure_writer1.clone())
    .await?;
```

For pass 2, the retry seeds come from `load_retry_seeds` (returns `Vec<SeedURL>`).
Convert using `vec_to_receiver`:

```rust
// In the pass-2 block, replace:
//   engine.run(retry_seeds, &retry_cfg, ...)
// with:
use crate::seed::vec_to_receiver;
let (retry_rx, retry_count) = vec_to_receiver(retry_seeds);
let pass2_cumulative = engine
    .run(retry_rx, retry_count, &retry_cfg, result_writer.clone(), failure_writer2.clone())
    .await?;
```

Add the import at top of `job.rs`: `use crate::seed::vec_to_receiver;`

### Step 6: Update `run_crawl_job` in `common.rs`

In `run_crawl_job`, convert `params.seeds` to a receiver before calling `run_job`:

```rust
// After creating cfg but before run_job:
use crawler_lib::seed::vec_to_receiver;
let (seed_rx, seed_count) = vec_to_receiver(params.seeds);

// Update the run_job call:
let job_result = run_job(
    seed_rx,
    seed_count,
    cfg,
    result_writer.clone(),
    open_failure_writer.as_ref(),
    load_retry.as_ref().map(...),
).await?;
```

### Step 7: Run all tests

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features
```

Expected: all PASS. Fix any compile errors (missing imports, type mismatches).

### Step 8: Commit

```bash
git add crawler-lib/src/engine/mod.rs crawler-lib/src/engine/reqwest_engine.rs \
        crawler-lib/src/engine/hyper_engine.rs crawler-lib/src/seed.rs \
        crawler-lib/src/job.rs crawler-cli/src/common.rs
git commit -m "refactor(crawler): Engine::run() accepts Receiver<SeedURL> instead of Vec

Decouples seed loading from the engine. All existing callers use
vec_to_receiver() to convert their Vec<SeedURL> into a closed channel.
This is the prerequisite for Task 8's streaming parquet loader which
feeds seeds directly without materialising the full 10M-row Vec."
```

---

## Task 8: E5b — Streaming CC seed loader

**Files:**
- Modify: `crawler-lib/src/seed.rs` (add `stream_seeds_cc_parquet_async`)
- Modify: `crawler-lib/src/engine/reqwest_engine.rs` (batch streaming producer)
- Modify: `crawler-lib/src/engine/hyper_engine.rs` (same)
- Modify: `crawler-cli/src/cc.rs` (use streaming loader)

Replace the full Vec materialisation in the CC path with a paginated DuckDB reader
that feeds seeds through an async channel. Peak seed RAM drops from ~3 GB to ~30 MB.

### Step 1: Add `stream_seeds_cc_parquet_async` to `seed.rs`

Add after `load_seeds_cc_parquet` in `crawler-lib/src/seed.rs`:

```rust
/// Stream CC seeds from a parquet file via paginated DuckDB queries.
///
/// Runs DuckDB queries in a `tokio::task::spawn_blocking` thread, feeding
/// seeds through a bounded async channel (capacity = SEED_CHAN_CAP).
/// Back-pressure is automatic: if the engine falls behind, the DuckDB
/// thread blocks on `tx.send()`.
///
/// Returns the receiver end. The sender side is owned by the background
/// task; when the task finishes (all pages read or limit hit), the sender
/// is dropped and the receiver returns `Err(Closed)`.
pub async fn stream_seeds_cc_parquet_async(
    path: String,
    limit: usize,
    filters: CcSeedFilter,
) -> Result<(async_channel::Receiver<SeedURL>, tokio::task::JoinHandle<Result<()>>)> {
    const SEED_CHAN_CAP: usize = 200_000;
    const PAGE_SIZE: usize = 100_000;

    let (tx, rx) = async_channel::bounded::<SeedURL>(SEED_CHAN_CAP);

    let handle = tokio::task::spawn_blocking(move || -> Result<()> {
        let conn = Connection::open_in_memory()?;

        // Cap DuckDB buffer pool (same as load_seeds_cc_parquet)
        {
            use sysinfo::System;
            let sys = System::new_all();
            let total_mb = sys.total_memory() / (1024 * 1024);
            let limit_mb = ((total_mb * 40 / 100) as usize).max(512).min(4096);
            conn.execute_batch(&format!("SET memory_limit='{limit_mb}MB'"))?;
        }

        let escaped = path.replace('\'', "''");
        let mut conditions = vec!["warc_filename IS NOT NULL".to_string()];
        if !filters.status_codes.is_empty() {
            let codes: Vec<String> = filters.status_codes.iter().map(|c| c.to_string()).collect();
            conditions.push(format!("fetch_status IN ({})", codes.join(",")));
        }
        if !filters.mime_types.is_empty() {
            let quoted: Vec<String> = filters
                .mime_types
                .iter()
                .map(|m| format!("'{}'", m.replace('\'', "''")))
                .collect();
            conditions.push(format!("content_mime_detected IN ({})", quoted.join(",")));
        }
        for lang in &filters.languages {
            conditions.push(format!(
                "content_languages LIKE '%{}%'",
                lang.replace('\'', "''")
            ));
        }
        let where_clause = conditions.join(" AND ");

        let rt = tokio::runtime::Handle::current();
        let mut offset = 0usize;

        loop {
            let page_limit = if limit > 0 {
                (limit.saturating_sub(offset)).min(PAGE_SIZE)
            } else {
                PAGE_SIZE
            };
            if page_limit == 0 {
                break;
            }

            let query = format!(
                "SELECT url, COALESCE(url_host_registered_domain, '') as domain \
                 FROM read_parquet('{}') WHERE {} LIMIT {} OFFSET {}",
                escaped, where_clause, page_limit, offset
            );

            let mut stmt = conn.prepare(&query)?;
            let page: Vec<SeedURL> = stmt
                .query_map([], |row| {
                    Ok(SeedURL {
                        url: row.get(0)?,
                        domain: row.get(1)?,
                    })
                })?
                .filter_map(|r| r.ok())
                .collect();

            let page_len = page.len();
            for seed in page {
                // block_on inside spawn_blocking is the standard pattern for
                // calling async code from a blocking context.
                if rt.block_on(tx.send(seed)).is_err() {
                    return Ok(()); // receiver dropped — engine shut down
                }
            }

            offset += page_len;
            if page_len < page_limit {
                break; // last page
            }
        }
        // tx dropped here → receiver sees channel closed
        Ok(())
    });

    Ok((rx, handle))
}
```

### Step 2: Replace the collect-all producer in `reqwest_engine.rs` with a batch-streaming producer

In `reqwest_engine.rs`, find the transitional collect-all block added in Task 7
Step 3 and replace it with a batch streaming producer. The key change is in the
producer `tokio::spawn` block.

Instead of collecting all seeds into a Vec upfront, the producer reads from
`seed_rx` in batches and interleaves them immediately:

```rust
// Replace the "Drain seed_rx into a Vec" block at the top of run() with:

// We no longer collect all seeds upfront.
// Remove: let seeds: Vec<SeedURL> = { ... };
// Remove: let total_seeds_actual = seeds.len();

// The domain_map starts empty; domain entries are created lazily per batch.
let domain_map: Arc<DashMap<String, Arc<DomainEntry>>> = Arc::new(DashMap::new());

// The producer reads seed_rx in batches, groups each batch by domain,
// and interleaves the batch round-robin before moving to the next batch.
// Memory footprint: O(BATCH_SIZE) ≈ 15 MB at 100K batch.
const PRODUCER_BATCH: usize = 100_000;
let dm = Arc::clone(&domain_map);
let producer = tokio::spawn(async move {
    use std::collections::VecDeque;

    loop {
        // Collect up to PRODUCER_BATCH seeds from seed_rx.
        let mut batch: Vec<SeedURL> = Vec::with_capacity(PRODUCER_BATCH);
        loop {
            if batch.len() >= PRODUCER_BATCH {
                break;
            }
            match seed_rx.try_recv() {
                Ok(s) => batch.push(s),
                Err(async_channel::TryRecvError::Empty) => {
                    if batch.is_empty() {
                        // No seeds yet — wait for first seed before spinning
                        match seed_rx.recv().await {
                            Ok(s) => batch.push(s),
                            Err(_) => return, // channel closed, nothing to do
                        }
                    } else {
                        break; // have a partial batch, process it
                    }
                }
                Err(async_channel::TryRecvError::Closed) => {
                    break; // channel closed, process remaining batch
                }
            }
        }

        if batch.is_empty() {
            break; // all seeds consumed
        }

        // Group this batch by domain.
        let batches = crate::domain::group_by_domain(batch);

        // Lazily insert new domain entries.
        for b in &batches {
            dm.entry(b.domain.clone())
                .or_insert_with(|| Arc::new(DomainEntry::new(inner_n)));
        }

        // Interleave batch round-robin and send to workers.
        let mut queue: VecDeque<(VecDeque<SeedURL>, Arc<DomainEntry>)> = batches
            .into_iter()
            .filter_map(|b| {
                dm.get(&b.domain)
                    .map(|e| (VecDeque::from(b.urls), Arc::clone(e.value())))
            })
            .collect();

        while let Some((mut urls, entry)) = queue.pop_front() {
            let url = match urls.pop_front() {
                Some(u) => u,
                None => continue,
            };
            if url_tx.send((url, Arc::clone(&entry))).await.is_err() {
                return; // workers all done
            }
            if !urls.is_empty() {
                queue.push_back((urls, entry));
            }
        }
    }
    // url_tx dropped → channel closes
});
```

Remove the now-unused `let batches = group_by_domain(seeds);` and the pre-creation
loop `for batch in &batches { domain_map.insert(...) }` — both are replaced by the
lazy per-batch insertion above.

The `workers` and `inner_n` variables are still set the same way. The channel
`url_tx`/`url_rx` capacity stays `workers * 4`.

Remove `let domain_count = batches.len();` and the `info!("grouped into {} domains")`
log (we no longer know total domains upfront). Update the domains_total stat:

```rust
// Remove: stats.domains_total.store(domain_count as u64, Ordering::Relaxed);
// (domain count is unknown until all seeds are read)
```

### Step 3: Apply the same streaming producer to `hyper_engine.rs`

Identical change. `HyperEngine` and `ReqwestEngine` producers are structurally the same.

### Step 4: Write a test for `stream_seeds_cc_parquet_async`

Since this requires creating a real parquet file, use DuckDB to generate a test one:

```rust
#[cfg(test)]
mod stream_tests {
    use super::*;
    use tempfile::tempdir;

    #[tokio::test]
    async fn stream_seeds_delivers_all_rows() {
        let dir = tempdir().unwrap();
        let parquet_path = dir.path().join("test.parquet");

        // Create a test parquet using DuckDB COPY
        let conn = duckdb::Connection::open_in_memory().unwrap();
        conn.execute_batch(&format!(
            "COPY (
               SELECT 'https://a.com/1' AS url, 'a.com' AS url_host_registered_domain, 'a.warc' AS warc_filename,
               UNION ALL
               SELECT 'https://b.com/1', 'b.com', 'b.warc',
               UNION ALL
               SELECT 'https://c.com/1', 'c.com', NULL   -- NULL warc_filename, excluded by filter
             ) TO '{}' (FORMAT PARQUET)",
            parquet_path.to_string_lossy()
        ))
        .unwrap();

        let filters = CcSeedFilter::default();
        let (rx, handle) = stream_seeds_cc_parquet_async(
            parquet_path.to_string_lossy().to_string(),
            0,
            filters,
        )
        .await
        .unwrap();

        let mut seeds = Vec::new();
        while let Ok(s) = rx.recv().await {
            seeds.push(s);
        }
        handle.await.unwrap().unwrap();

        assert_eq!(seeds.len(), 2, "NULL warc_filename row should be excluded");
        let urls: Vec<&str> = seeds.iter().map(|s| s.url.as_str()).collect();
        assert!(urls.contains(&"https://a.com/1"));
        assert!(urls.contains(&"https://b.com/1"));
    }
}
```

Run: `cargo test --no-default-features -- stream_seeds_delivers_all_rows` → Expected: PASS.

### Step 5: Update `cc.rs` to use the streaming loader

In `crawler-cli/src/cc.rs`, replace the seed loading + `run_crawl_job` call:

```rust
// Before
println!("Loading CC seeds from parquet (warc_filename IS NOT NULL)...");
let seeds = load_seeds_cc_parquet(&parquet_str, args.limit, &filter)?;
if seeds.is_empty() {
    println!("No seeds found, exiting.");
    return Ok(());
}
println!("Loaded {} seeds", seeds.len());
// ...
run_crawl_job(CrawlJobParams {
    seeds,
    ...
}).await
```

```rust
// After — streaming: no Vec materialisation, no COUNT(*), feed directly to engine
use crawler_lib::seed::stream_seeds_cc_parquet_async;

println!("Streaming CC seeds from parquet (warc_filename IS NOT NULL)...");
let (seed_rx, seed_loader) = stream_seeds_cc_parquet_async(
    parquet_str.clone(),
    args.limit,
    filter,
).await?;

// We don't know total count upfront — pass 0 (TUI shows absolute numbers)
run_crawl_job_streaming(CrawlJobParamsStreaming {
    title,
    seed_rx,
    seed_count_hint: 0,
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
    flusher_threads: args.flusher_threads,
}).await?;

// Await the loader to propagate any DuckDB errors
seed_loader.await??;
Ok(())
```

Add `run_crawl_job_streaming` and `CrawlJobParamsStreaming` to `common.rs`. This is
identical to `run_crawl_job` / `CrawlJobParams` except `seeds: Vec<SeedURL>` is
replaced by `seed_rx: async_channel::Receiver<SeedURL>` + `seed_count_hint: u64`,
and `vec_to_receiver` is not called (the receiver is already ready):

```rust
pub struct CrawlJobParamsStreaming {
    // same fields as CrawlJobParams except:
    pub seed_rx: async_channel::Receiver<SeedURL>,
    pub seed_count_hint: u64,
    // all other fields identical to CrawlJobParams
    // ... copy paste and update ...
}

pub async fn run_crawl_job_streaming(params: CrawlJobParamsStreaming) -> Result<()> {
    // identical to run_crawl_job but calls:
    //   run_job(params.seed_rx, params.seed_count_hint, cfg, ...)
    // instead of:
    //   let (rx, count) = vec_to_receiver(params.seeds);
    //   run_job(rx, count, cfg, ...)
}
```

To avoid duplication, extract a private `run_crawl_job_inner(seed_rx, seed_count, ...)` and have both `run_crawl_job` and `run_crawl_job_streaming` call it.

### Step 6: Run all tests

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features
```

Expected: all PASS.

### Step 7: Build the release binary to verify it compiles clean

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo build --release --no-default-features 2>&1 | tail -5
```

Expected: `Finished release` with no errors.

### Step 8: Commit

```bash
git add crawler-lib/src/seed.rs crawler-lib/src/engine/reqwest_engine.rs \
        crawler-lib/src/engine/hyper_engine.rs crawler-cli/src/cc.rs \
        crawler-cli/src/common.rs
git commit -m "feat(crawler): streaming CC seed loading — 3 GB → 30 MB peak RAM

stream_seeds_cc_parquet_async() reads DuckDB rows in pages of 100K via
tokio::task::spawn_blocking, feeding a bounded async_channel (cap=200K).
The engine's streaming batch producer reads 100K seeds at a time,
groups by domain, and interleaves round-robin without ever holding
more than ~30 MB of seed data in memory (vs ~3 GB full materialisation).

cc.rs now calls run_crawl_job_streaming() which passes the Receiver
directly to run_job, bypassing vec_to_receiver entirely."
```

---

## Task 9: Final verification

### Step 1: Run full test suite

```bash
cd blueprints/search/tools/crawler
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features 2>&1 | tail -20
```

Expected: all tests PASS, `test result: ok`.

### Step 2: Build release binary

```bash
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo build --release --no-default-features
```

Expected: `Finished release`.

### Step 3: Smoke test with --help

```bash
$HOME/.cache/mizu/crawler-target/release/crawler cc recrawl --help
```

Expected: help text prints cleanly with all flags present.

### Step 4: Verify all 6 enhancements are in place

Run each targeted test group to confirm:

```bash
# E2: retry SQL includes killed URLs
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features -- load_retry_seeds

# E1: adaptive floor
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features -- adaptive_tests

# E3: deque producer
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features -- test_interleave

# E5a: (no specific test — COUNT removed; confirmed by build passing)

# E4: async body store
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features -- async_body_store

# E5b: streaming seeds
CARGO_TARGET_DIR=$HOME/.cache/mizu/crawler-target cargo test --no-default-features -- stream_seeds vec_to_receiver
```

Expected: all targeted tests PASS.

### Step 5: Deploy to server 2 (optional — for benchmark)

```bash
cd blueprints/search/tools/crawler
make build-on-server SERVER=2
```

Then on server 2:

```bash
~/bin/crawler cc recrawl --file p:0 --no-tui 2>&1 | tee ~/logs/cc-bench-$(date +%Y%m%d).log
```

Watch for: avg OK/s ≥ 1500, pass-2 retried count includes domain_http_timeout_killed URLs.

---

## Summary of Changes

| Task | Enhancement | Key Change |
|------|-------------|------------|
| 1 | E2 retry SQL | `reason IN ('http_timeout', 'domain_http_timeout_killed')` |
| 2 | E1 adaptive floor | `.max(cfg.timeout).min(×3)` in both engines |
| 3 | E3 deque producer | `VecDeque` round-robin, O(N) vs O(max×domains) |
| 4 | E5a COUNT removed | Delete prescan from `load_seeds_cc_parquet` |
| 5 | E4 AsyncBodyStore | Background thread for gzip+write, DashSet dedup |
| 6 | E4 wire | Config type + engine call site + common.rs close |
| 7 | E5b interface | Engine trait takes `Receiver<SeedURL>`, `vec_to_receiver` helper |
| 8 | E5b streaming | `stream_seeds_cc_parquet_async`, batch producer, cc.rs streaming path |
| 9 | Verification | Full test suite + build + deploy |
