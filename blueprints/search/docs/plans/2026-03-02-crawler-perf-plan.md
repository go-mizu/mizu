# Crawler Binary Writer Performance Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate the binary writer's single-threaded Mutex bottleneck to reach ≥1,500 OK pages/s on server 2.

**Architecture:** Replace `Mutex<Option<Sender<T>>>` with a lock-free `Sender<Option<T>>` (sentinel-based close); spawn N flusher threads sharing the receiver, each writing to their own segment file family; drop the domain semaphore before writing so channel backpressure never stalls domain-level fetches.

**Tech Stack:** Rust 2024, crossbeam-channel 0.5 (MPMC — Receiver is Clone), tokio (engine workers), rkyv 0.8 (binary serialization)

**Working directory:** `blueprints/search/tools/crawler/`

**Build:** `cargo build --release --no-default-features` (macOS) or `make build`
**Test:** `cargo test` or `make test`

---

## Task 1: Lock-free write path in `BinaryResultWriter`

> Core fix: remove Mutex from hot write path, switch to sentinel-based close, spawn N flusher threads.

**Files:**
- Modify: `crawler-lib/src/writer/binary.rs`

### Step 1: Add the concurrent stress test (will fail before fix)

Add this test at the bottom of the `tests` module in `crawler-lib/src/writer/binary.rs`:

```rust
#[test]
fn test_concurrent_writes_no_deadlock() {
    use std::sync::Arc;
    let dir = tempfile::tempdir().unwrap();
    let seg_dir = dir.path().join("concurrent");

    // 4 flusher threads, large channel
    let writer = Arc::new(BinaryResultWriter::new(&seg_dir, 65536, 64, 4).unwrap());

    let mut handles = Vec::new();
    for t in 0..200usize {
        let w = Arc::clone(&writer);
        handles.push(std::thread::spawn(move || {
            for i in 0..100usize {
                let url = format!("https://example.com/t{t}/p{i}");
                w.write(make_result(&url)).unwrap();
            }
        }));
    }
    for h in handles {
        h.join().unwrap();
    }
    writer.close().unwrap();

    let results = read_result_segments(&seg_dir).unwrap();
    assert_eq!(results.len(), 20_000, "expected 20_000 records, got {}", results.len());
}
```

### Step 2: Run test to confirm it fails (compilation error — `new` takes 3 args, not 4)

```bash
cd blueprints/search/tools/crawler
cargo test test_concurrent_writes_no_deadlock 2>&1 | tail -20
```

Expected: compile error about `new` argument count.

### Step 3: Rewrite `BinaryResultWriter` in `binary.rs`

Replace the full `BinaryResultWriter` block (struct + impl). The key changes:

**a) Change `open_segment` to include thread ID in filename:**

```rust
fn open_segment(dir: &Path, thread_id: usize, idx: usize) -> Result<BufWriter<File>> {
    let path = dir.join(format!("seg_t{:02}_{:03}.bin", thread_id, idx));
    let f = File::create(&path)
        .with_context(|| format!("failed to create segment file: {}", path.display()))?;
    Ok(BufWriter::new(f))
}
```

**b) Change `run_flusher_loop` signature and inner loop:**

```rust
fn run_flusher_loop<T>(
    dir: &Path,
    thread_id: usize,
    rx: &crossbeam_channel::Receiver<Option<T>>,
    seg_size_bytes: usize,
    label: &str,
    encode: impl Fn(&T) -> Result<Vec<u8>>,
) {
    let mut seg_idx: usize = 0;
    let mut writer = match open_segment(dir, thread_id, seg_idx) {
        Ok(w) => w,
        Err(e) => {
            error!("bin-{label}-flusher-{thread_id}: failed to open initial segment: {e}");
            return;
        }
    };
    let mut seg_bytes: usize = 0;
    let mut total_records: u64 = 0;

    for msg in rx.iter() {
        let item = match msg {
            Some(item) => item,
            None => break, // sentinel: this thread is done
        };

        let encoded = match encode(&item) {
            Ok(v) => v,
            Err(e) => {
                error!("bin-{label}-flusher-{thread_id}: encode error: {e}");
                continue;
            }
        };

        let len = encoded.len() as u32;
        if let Err(e) = writer
            .write_all(&len.to_le_bytes())
            .and_then(|_| writer.write_all(&encoded))
        {
            error!("bin-{label}-flusher-{thread_id}: write error on seg_t{thread_id:02}_{seg_idx:03}: {e}");
            continue;
        }

        seg_bytes += 4 + encoded.len();
        total_records += 1;

        if seg_size_bytes > 0 && seg_bytes >= seg_size_bytes {
            if let Err(e) = writer.flush() {
                error!("bin-{label}-flusher-{thread_id}: flush error: {e}");
            }
            seg_idx += 1;
            writer = match open_segment(dir, thread_id, seg_idx) {
                Ok(w) => w,
                Err(e) => {
                    error!("bin-{label}-flusher-{thread_id}: failed to open seg: {e}");
                    return;
                }
            };
            seg_bytes = 0;
        }
    }

    if let Err(e) = writer.flush() {
        error!("bin-{label}-flusher-{thread_id}: final flush error: {e}");
    }
    info!(
        "bin-{label}-flusher-{thread_id}: done, {total_records} records in {} segments",
        seg_idx + 1
    );
}
```

**c) New `BinaryResultWriter` struct and impl:**

```rust
pub struct BinaryResultWriter {
    /// Lock-free write path: send Some(result) directly, no Mutex.
    tx: crossbeam_channel::Sender<Option<CrawlResult>>,
    /// Flusher thread handles — only locked in close().
    handles: Mutex<Vec<std::thread::JoinHandle<()>>>,
    /// Number of flusher threads (must match how many None sentinels close() sends).
    num_flushers: usize,
    dir: PathBuf,
    drain_config: Option<BinDrainConfig>,
}

impl BinaryResultWriter {
    pub fn new(dir: &Path, channel_cap: usize, seg_size_mb: usize, num_flushers: usize) -> Result<Self> {
        Self::new_inner(dir, channel_cap, seg_size_mb, num_flushers, None)
    }

    pub fn new_with_drain(
        dir: &Path,
        channel_cap: usize,
        seg_size_mb: usize,
        num_flushers: usize,
        drain: BinDrainConfig,
    ) -> Result<Self> {
        Self::new_inner(dir, channel_cap, seg_size_mb, num_flushers, Some(drain))
    }

    fn new_inner(
        dir: &Path,
        channel_cap: usize,
        seg_size_mb: usize,
        num_flushers: usize,
        drain_config: Option<BinDrainConfig>,
    ) -> Result<Self> {
        std::fs::create_dir_all(dir)
            .with_context(|| format!("failed to create binary writer dir: {}", dir.display()))?;

        // Remove stale seg_*.bin files from a previous failed run.
        if let Ok(rd) = std::fs::read_dir(dir) {
            for entry in rd.flatten() {
                let p = entry.path();
                if p.extension().map_or(false, |e| e == "bin")
                    && p.file_name()
                        .and_then(|n| n.to_str())
                        .map_or(false, |n| n.starts_with("seg_"))
                {
                    let _ = std::fs::remove_file(&p);
                }
            }
        }

        let n = num_flushers.max(1);
        let seg_size_bytes = seg_size_mb * 1024 * 1024;

        // crossbeam Receiver is Clone (MPMC), so each flusher gets its own clone.
        let (tx, rx) = crossbeam_channel::bounded::<Option<CrawlResult>>(channel_cap);

        let mut handles = Vec::with_capacity(n);
        for thread_id in 0..n {
            let rx = rx.clone();
            let dir_path = dir.to_path_buf();
            let handle = std::thread::Builder::new()
                .name(format!("bin-result-flusher-{thread_id}"))
                .spawn(move || {
                    run_flusher_loop(&dir_path, thread_id, &rx, seg_size_bytes, "result", |item| {
                        rkyv::to_bytes::<RkyvError>(item)
                            .map(|v| v.to_vec())
                            .map_err(|e| anyhow::anyhow!("rkyv encode CrawlResult: {e}"))
                    });
                })
                .context("failed to spawn result flusher thread")?;
            handles.push(handle);
        }
        // Drop the original rx — only the per-thread clones remain.
        drop(rx);

        Ok(Self {
            tx,
            handles: Mutex::new(handles),
            num_flushers: n,
            dir: dir.to_path_buf(),
            drain_config,
        })
    }

    pub fn with_defaults(dir: &Path) -> Result<Self> {
        Self::new(dir, DEFAULT_CHANNEL_CAP, DEFAULT_SEG_SIZE_MB, 1)
    }

    pub fn with_drain(dir: &Path, drain: BinDrainConfig) -> Result<Self> {
        Self::new_with_drain(dir, DEFAULT_CHANNEL_CAP, DEFAULT_SEG_SIZE_MB, 1, drain)
    }

    pub fn dir(&self) -> &Path {
        &self.dir
    }
}

impl ResultWriter for BinaryResultWriter {
    fn write(&self, result: CrawlResult) -> Result<()> {
        // Hot path: lock-free send directly through crossbeam channel.
        self.tx
            .send(Some(result))
            .map_err(|_| anyhow::anyhow!("binary result channel closed"))
    }

    fn flush(&self) -> Result<()> {
        Ok(())
    }

    fn close(&self) -> Result<()> {
        // Send one None sentinel per flusher thread so each exits its loop.
        for _ in 0..self.num_flushers {
            let _ = self.tx.send(None);
        }

        let handles = {
            let mut guard = self.handles.lock().unwrap();
            guard.drain(..).collect::<Vec<_>>()
        };
        for h in handles {
            h.join()
                .map_err(|_| anyhow::anyhow!("result flusher thread panicked"))?;
        }

        if let Some(cfg) = &self.drain_config {
            drain_to_duckdb(&self.dir, cfg)?;
        }

        Ok(())
    }
}
```

### Step 4: Fix existing tests that call `BinaryResultWriter::new` with 3 args

In the `tests` module, update these calls to add `num_flushers=1`:

```rust
// test_result_writer_roundtrip
let writer = BinaryResultWriter::new(&seg_dir, 100, 1, 1).unwrap();

// test_segment_rotation
let writer = BinaryResultWriter::new(&seg_dir, 100, 0, 1).unwrap();

// test_write_after_close_returns_error
let writer = BinaryResultWriter::new(&seg_dir, 100, 1, 1).unwrap();
```

`test_drain_to_duckdb` uses `BinaryResultWriter::with_drain` — no change needed (it now uses num_flushers=1 default).

### Step 5: Run all writer tests

```bash
cargo test --package crawler-lib writer 2>&1 | tail -30
```

Expected: all tests PASS including `test_concurrent_writes_no_deadlock`.

### Step 6: Commit

```bash
git add crawler-lib/src/writer/binary.rs
git commit -m "perf(binary-writer): lock-free write path via sentinel close + multi-flusher"
```

---

## Task 2: Lock-free write path in `BinaryFailureWriter`

> Same sentinel pattern for the failure writer — eliminates Mutex from the failure write path.

**Files:**
- Modify: `crawler-lib/src/writer/binary.rs` (BinaryFailureWriter section)

### Step 1: Replace `BinaryFailureWriter` struct and impl

The failure writer has two channels (url + domain). Apply the same sentinel pattern to each.

```rust
pub struct BinaryFailureWriter {
    url_tx: crossbeam_channel::Sender<Option<FailedURL>>,
    domain_tx: crossbeam_channel::Sender<Option<FailedDomain>>,
    handles: Mutex<Vec<std::thread::JoinHandle<()>>>,
    /// Each channel has its own flusher count (currently 1 each).
    url_flushers: usize,
    domain_flushers: usize,
    dir: PathBuf,
    drain_config: Option<BinFailureDrainConfig>,
}

impl BinaryFailureWriter {
    pub fn new(dir: &Path, channel_cap: usize, seg_size_mb: usize) -> Result<Self> {
        Self::new_inner(dir, channel_cap, seg_size_mb, None)
    }

    pub fn with_drain(dir: &Path, drain: BinFailureDrainConfig) -> Result<Self> {
        Self::new_inner(dir, DEFAULT_CHANNEL_CAP, DEFAULT_SEG_SIZE_MB, Some(drain))
    }

    fn new_inner(
        dir: &Path,
        channel_cap: usize,
        seg_size_mb: usize,
        drain_config: Option<BinFailureDrainConfig>,
    ) -> Result<Self> {
        std::fs::create_dir_all(dir)
            .with_context(|| format!("failed to create failure writer dir: {}", dir.display()))?;

        let seg_size_bytes = seg_size_mb * 1024 * 1024;

        // URL flusher (1 thread)
        let (url_tx, url_rx) = crossbeam_channel::bounded::<Option<FailedURL>>(channel_cap);
        let url_dir = dir.join("failed_urls");
        std::fs::create_dir_all(&url_dir)?;
        let url_handle = std::thread::Builder::new()
            .name("bin-fail-url-flusher-0".into())
            .spawn({
                let d = url_dir;
                move || {
                    run_flusher_loop(&d, 0, &url_rx, seg_size_bytes, "fail-url", |item| {
                        rkyv::to_bytes::<RkyvError>(item)
                            .map(|v| v.to_vec())
                            .map_err(|e| anyhow::anyhow!("rkyv encode FailedURL: {e}"))
                    });
                }
            })
            .context("failed to spawn URL failure flusher thread")?;

        // Domain flusher (1 thread)
        let (domain_tx, domain_rx) = crossbeam_channel::bounded::<Option<FailedDomain>>(channel_cap);
        let domain_dir = dir.join("failed_domains");
        std::fs::create_dir_all(&domain_dir)?;
        let domain_handle = std::thread::Builder::new()
            .name("bin-fail-domain-flusher-0".into())
            .spawn({
                let d = domain_dir;
                move || {
                    run_flusher_loop(&d, 0, &domain_rx, seg_size_bytes, "fail-domain", |item| {
                        rkyv::to_bytes::<RkyvError>(item)
                            .map(|v| v.to_vec())
                            .map_err(|e| anyhow::anyhow!("rkyv encode FailedDomain: {e}"))
                    });
                }
            })
            .context("failed to spawn domain failure flusher thread")?;

        Ok(Self {
            url_tx,
            domain_tx,
            handles: Mutex::new(vec![url_handle, domain_handle]),
            url_flushers: 1,
            domain_flushers: 1,
            dir: dir.to_path_buf(),
            drain_config,
        })
    }

    pub fn with_defaults(dir: &Path) -> Result<Self> {
        Self::new(dir, DEFAULT_CHANNEL_CAP, DEFAULT_SEG_SIZE_MB)
    }

    pub fn dir(&self) -> &Path {
        &self.dir
    }
}

impl FailureWriter for BinaryFailureWriter {
    fn write_url(&self, failed: FailedURL) -> Result<()> {
        self.url_tx
            .send(Some(failed))
            .map_err(|_| anyhow::anyhow!("binary failure URL channel closed"))
    }

    fn write_domain(&self, failed: FailedDomain) -> Result<()> {
        self.domain_tx
            .send(Some(failed))
            .map_err(|_| anyhow::anyhow!("binary failure domain channel closed"))
    }

    fn flush(&self) -> Result<()> {
        Ok(())
    }

    fn close(&self) -> Result<()> {
        // Send sentinels for each flusher.
        for _ in 0..self.url_flushers {
            let _ = self.url_tx.send(None);
        }
        for _ in 0..self.domain_flushers {
            let _ = self.domain_tx.send(None);
        }

        let handles: Vec<_> = {
            let mut guard = self.handles.lock().unwrap();
            guard.drain(..).collect()
        };
        for h in handles {
            h.join()
                .map_err(|_| anyhow::anyhow!("failure flusher thread panicked"))?;
        }

        // Drain binary URL segments → failed.duckdb for pass-2 retry.
        if let Some(ref cfg) = self.drain_config {
            let url_dir = self.dir.join("failed_urls");
            if url_dir.exists() {
                let records = read_failed_url_segments(&url_dir)
                    .context("reading failed_url binary segments for drain")?;
                if !records.is_empty() {
                    let conn = open_failed_db(&cfg.db_path, cfg.mem_mb)
                        .context("opening failed.duckdb for drain")?;
                    for chunk in records.chunks(cfg.batch_size.max(1)) {
                        flush_failed_url_batch(&conn, chunk)
                            .context("inserting failed URLs into duckdb")?;
                    }
                    info!(
                        "bin-fail-drain: {} failed URL records → {:?}",
                        records.len(),
                        cfg.db_path
                    );
                }
            }
        }

        Ok(())
    }
}
```

### Step 2: Run all writer tests

```bash
cargo test --package crawler-lib writer 2>&1 | tail -30
```

Expected: all PASS.

### Step 3: Commit

```bash
git add crawler-lib/src/writer/binary.rs
git commit -m "perf(binary-writer): sentinel close for BinaryFailureWriter (lock-free write_url/write_domain)"
```

---

## Task 3: Add `num_flushers` to `Config` and `auto_config`

**Files:**
- Modify: `crawler-lib/src/config.rs`

### Step 1: Add field to `Config` struct

In the `Config` struct (after `db_mem_mb`):

```rust
/// Number of binary flusher threads (0 = auto via auto_config).
pub num_flushers: usize,
```

### Step 2: Add default value

In `impl Default for Config`:

```rust
num_flushers: 0,
```

### Step 3: Set value in `auto_config`

At the end of `auto_config`, before `cfg`:

```rust
// num_flushers: 1 flusher per 2 CPUs, clamped [2, 8].
// Each flusher is a dedicated OS thread doing rkyv serialize + disk write.
let num_flushers = clamp(si.cpu_count / 2, 2, 8);
cfg.num_flushers = num_flushers;
```

### Step 4: Run tests

```bash
cargo test --package crawler-lib 2>&1 | tail -20
```

Expected: all PASS (Config change is additive).

### Step 5: Commit

```bash
git add crawler-lib/src/config.rs
git commit -m "feat(config): add num_flushers field (0=auto, auto_config sets clamp(cpus/2, 2, 8))"
```

---

## Task 4: Drop semaphore before write in `reqwest_engine.rs`

> Prevents channel backpressure from stalling domain-level concurrency.

**Files:**
- Modify: `crawler-lib/src/engine/reqwest_engine.rs`

### Step 1: Add `drop(_permit)` at the right place

In `process_one_url`, after the fetch completes and `stats.total` is incremented, add an explicit `drop(_permit)` before the `match fetch_result` block:

```rust
    let fetch_result = fetch_one(
        client,
        &seed,
        effective_timeout,
        cfg.max_body_bytes,
        cfg.body_store.as_deref(),
    )
    .await;
    stats.total.fetch_add(1, Ordering::Relaxed);

    // Release domain semaphore permit NOW — before any writer call.
    // When the binary writer channel is full, write() blocks; if the permit
    // is still held during that block, no other worker can fetch from this
    // domain, serializing all domain fetches behind writer backpressure.
    drop(_permit);

    match fetch_result {
        // ... rest unchanged ...
```

### Step 2: Run tests

```bash
cargo test --package crawler-lib engine 2>&1 | tail -20
```

Expected: all PASS.

### Step 3: Commit

```bash
git add crawler-lib/src/engine/reqwest_engine.rs
git commit -m "perf(reqwest-engine): drop semaphore permit before write to decouple fetch from writer backpressure"
```

---

## Task 5: Drop semaphore before write in `hyper_engine.rs`

> Same fix as Task 4 for the hyper engine.

**Files:**
- Modify: `crawler-lib/src/engine/hyper_engine.rs`

### Step 1: Locate the write call in hyper's `process_one_url`

In `hyper_engine.rs`, `process_one_url` ends around line 347 with:
```rust
    stats.bytes_downloaded.fetch_add(result.content_length as u64, Ordering::Relaxed);
    let _ = results.write(result);
```

The semaphore acquire is around line 268. Add `drop(_permit)` after `stats.total.fetch_add(1, ...)` and before the `if !result.error.is_empty()` block:

```rust
    let result = hyper_fetch_one(client, &seed, effective_timeout, cfg.max_body_bytes).await;
    stats.total.fetch_add(1, Ordering::Relaxed);

    // Release domain semaphore before any writer call (see reqwest_engine.rs comment).
    drop(_permit);

    if !result.error.is_empty() {
        // ... unchanged ...
```

### Step 2: Run all tests

```bash
cargo test --package crawler-lib 2>&1 | tail -20
```

Expected: all PASS.

### Step 3: Commit

```bash
git add crawler-lib/src/engine/hyper_engine.rs
git commit -m "perf(hyper-engine): drop semaphore permit before write"
```

---

## Task 6: Wire `num_flushers` through CLI and writer construction

> Pass the new config field into the binary writer constructors end-to-end.

**Files:**
- Modify: `crawler-cli/src/common.rs`
- Modify: `crawler-cli/src/hn.rs`
- Modify: `crawler-cli/src/cc.rs`

### Step 1: Add `flusher_threads` to `CrawlJobParams` in `common.rs`

Find the `CrawlJobParams` struct definition. Add:

```rust
/// Binary writer flusher thread count (0 = use auto_config value from Config).
pub flusher_threads: usize,
```

### Step 2: Thread `flusher_threads` into binary writer construction

In `create_result_writer` (or wherever `BinaryResultWriter::new` / `BinaryResultWriter::with_drain` is called in `common.rs`), pass `num_flushers`. Find the call and replace like:

```rust
// Before:
BinaryResultWriter::with_drain(&seg_dir, drain)
// OR: BinaryResultWriter::new(&seg_dir, ...)

// After:
BinaryResultWriter::new_with_drain(&seg_dir, DEFAULT_CHANNEL_CAP, DEFAULT_SEG_SIZE_MB, num_flushers, drain)
// OR: BinaryResultWriter::new(&seg_dir, DEFAULT_CHANNEL_CAP, DEFAULT_SEG_SIZE_MB, num_flushers)
```

Where `num_flushers` comes from `cfg.num_flushers` (which is set by auto_config or CLI override).

To read the exact call site first, run:

```bash
grep -n "BinaryResultWriter" crawler-cli/src/common.rs
```

Then apply the `num_flushers` argument to each call found.

### Step 3: Override `cfg.num_flushers` from `CrawlJobParams`

In `run_crawl_job`, after the config is built (either from auto_config or manually), add:

```rust
if params.flusher_threads > 0 {
    cfg.num_flushers = params.flusher_threads;
}
// If 0, keep the auto_config value already in cfg.num_flushers.
```

### Step 4: Add `--flusher-threads` arg to `hn.rs`

In `RecrawlArgs`:

```rust
/// Binary writer flusher thread count (0 = auto)
#[arg(long, default_value_t = 0)]
pub flusher_threads: usize,
```

In `run_recrawl`, add to the `CrawlJobParams` call:

```rust
flusher_threads: args.flusher_threads,
```

### Step 5: Add `--flusher-threads` arg to `cc.rs`

Same pattern as hn.rs — find `RecrawlArgs` (or equivalent struct) and add the field + pass it through.

### Step 6: Build to verify compilation

```bash
cargo build --no-default-features 2>&1 | tail -20
```

Expected: clean build.

### Step 7: Run all tests

```bash
cargo test 2>&1 | tail -30
```

Expected: all PASS.

### Step 8: Commit

```bash
git add crawler-cli/src/common.rs crawler-cli/src/hn.rs crawler-cli/src/cc.rs
git commit -m "feat(cli): add --flusher-threads flag, wire num_flushers through CrawlJobParams"
```

---

## Task 7: Local smoke test + server 2 benchmark

> Verify the fix works end-to-end before deploying.

### Step 1: Quick local smoke test (devnull vs binary)

```bash
# Build
cargo build --release --no-default-features

# Devnull baseline (needs a small seed file — adjust path)
~/bin/crawler hn recrawl --writer devnull --limit 1000 --no-retry 2>&1 | grep -E "avg rps|ok="

# Binary writer (1 flusher)
~/bin/crawler hn recrawl --writer binary --flusher-threads 1 --limit 1000 --no-retry 2>&1 | grep -E "avg rps|ok="

# Binary writer (4 flushers)
~/bin/crawler hn recrawl --writer binary --flusher-threads 4 --limit 1000 --no-retry 2>&1 | grep -E "avg rps|ok="
```

Expected: binary-4 avg RPS within 10% of devnull; channel saturation < 20%.

### Step 2: Deploy to server 2 and benchmark

```bash
cd blueprints/search/tools/crawler
make build-on-server SERVER=2
```

On server 2 (via SSH):

```bash
# DevNull baseline (200K seeds)
~/bin/crawler hn recrawl --writer devnull --limit 200000 --no-retry 2>&1 | tail -5

# Binary 4 flushers (200K seeds)
# Delete stale duckdb before run
rm -f ~/data/hn/results/results_*.duckdb
~/bin/crawler hn recrawl --writer binary --flusher-threads 4 --limit 200000 --no-retry 2>&1 | tail -10
```

Expected output (binary-4 should approach devnull):
- `avg rps >= 3500` (was 2547)
- `ok pages/s >= 1500` with ≥80% seed quality
- No channel saturation message

### Step 3: If results look good, commit a benchmark note

```bash
git add -A
git commit -m "bench: binary-4-flusher results on server2 — [fill in numbers]"
```

---

## Acceptance Criteria

| Metric | Before | Target |
|--------|--------|--------|
| Binary avg RPS (200K seeds, server2) | 2,547 | ≥ 3,500 |
| Binary vs devnull gap | 36% | ≤ 10% |
| Channel saturation | 100% | ≤ 20% |
| OK pages/s (80% seed quality) | ~2,037 | ≥ 1,500 |
| All existing tests | PASS | PASS |
| `test_concurrent_writes_no_deadlock` (new) | N/A | PASS |
