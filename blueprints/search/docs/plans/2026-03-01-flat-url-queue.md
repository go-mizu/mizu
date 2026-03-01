# Flat URL Task Queue + Parallel Drain Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Increase Rust crawler avg RPS from 1,684 to 5,000+ on server2 by replacing domain-batch workers with a flat URL task queue and parallelizing the post-crawl drain.

**Architecture:** Replace domain-batch workers (worker holds whole domain, waits for all inner_n tasks) with flat URL workers (worker processes one URL, immediately picks the next). Per-domain `tokio::sync::Semaphore` enforces inner_n concurrency limit per domain without blocking the worker. Rayon-parallel drain eliminates the 96s post-crawl DuckDB insert bottleneck.

**Tech Stack:** Rust, tokio 1, async-channel 2, dashmap 6, rayon 1, crossbeam-channel 0.5, duckdb 1.4 (bundled)

---

## Background Context

**Files involved:**
- `tools/crawler/crawler-lib/src/engine/reqwest_engine.rs` — engine implementation (763 lines)
- `tools/crawler/crawler-lib/src/domain.rs` — `DomainBatch`, `DomainState` (142 lines)
- `tools/crawler/crawler-lib/src/writer/binary.rs` — `BinaryResultWriter`, `drain_to_duckdb` (781 lines)
- `tools/crawler/crawler-lib/src/config.rs` — `Config`, `SysInfo`, `auto_config` (226 lines)
- `tools/crawler/crawler-lib/Cargo.toml` — dependencies

**Key types:**
- `SeedURL` = `{ url: String, domain: String }`
- `CrawlResult` = struct with 12 fields (url, status_code, title, etc.)
- `DomainState` = `{ successes: u64, timeouts: u64 }` with `should_abandon(threshold, probe, ratio, inner_n) -> bool`
- `Config` = contains `workers`, `inner_n`, `timeout`, `domain_dead_probe`, `domain_stall_ratio`, `domain_fail_threshold`, `disable_adaptive_timeout`, `adaptive_timeout_max`, `max_body_bytes`
- `ResultWriter` / `FailureWriter` = Arc-wrapped trait objects

**Current benchmark (server2, 200K seeds, binary, --no-retry):**
- avg RPS: 1,684 | peak: 6,067 | workers: 3,004 | duration: 1m58s | drain: 96.3s

**Bottleneck:** Domain-batch workers idle 95.5% of time. Little's Law: 1,684 RPS × 80ms avg latency = 134 effective concurrent out of 3,004 workers.

---

## Task 1: Add dashmap and rayon dependencies

**Files:**
- Modify: `tools/crawler/crawler-lib/Cargo.toml` (append two lines)

**Step 1: Open Cargo.toml and add deps after `async-channel`**

```toml
dashmap = "6"
rayon = "1"
```

**Step 2: Verify it compiles**

```bash
cd tools/crawler && cargo check 2>&1 | tail -5
```
Expected: `Finished` with no errors (dashmap + rayon download on first run).

**Step 3: Commit**

```bash
git add tools/crawler/crawler-lib/Cargo.toml
git commit -m "feat(crawler): add dashmap + rayon deps for flat URL queue + parallel drain"
```

---

## Task 2: Add DomainEntry struct to reqwest_engine.rs

This is the per-domain state that replaces the old per-domain `AtomicU64` scattered across tasks.

**Files:**
- Modify: `tools/crawler/crawler-lib/src/engine/reqwest_engine.rs` (add struct near top, after imports)

**Step 1: Add import and DomainEntry struct**

After the existing `use` block at the top of `reqwest_engine.rs`, add:

```rust
use dashmap::DashMap;
```

Then add this struct definition after the `ReqwestEngine` struct (after line ~27):

```rust
/// Per-domain state shared across all workers fetching from the same domain.
/// The semaphore limits concurrency to inner_n; abandoned flag short-circuits remaining URLs.
struct DomainEntry {
    semaphore: tokio::sync::Semaphore,
    abandoned: std::sync::atomic::AtomicBool,
    ok: std::sync::atomic::AtomicU64,
    timeouts: std::sync::atomic::AtomicU64,
}

impl DomainEntry {
    fn new(inner_n: usize) -> Self {
        Self {
            semaphore: tokio::sync::Semaphore::new(inner_n.max(1)),
            abandoned: std::sync::atomic::AtomicBool::new(false),
            ok: std::sync::atomic::AtomicU64::new(0),
            timeouts: std::sync::atomic::AtomicU64::new(0),
        }
    }
}
```

**Step 2: Verify it compiles**

```bash
cd tools/crawler && cargo check 2>&1 | tail -5
```
Expected: `Finished` (no errors — just a new struct, no logic changes).

---

## Task 3: Implement process_one_url()

Replace the existing `process_one_domain()` function with a per-URL version.

**Files:**
- Modify: `tools/crawler/crawler-lib/src/engine/reqwest_engine.rs`

**Step 1: Add `process_one_url` function**

Add this NEW function AFTER `process_one_domain` (do not delete process_one_domain yet — keep it for reference). It will go at approximately line 380+ (after the domain timeout code):

```rust
/// Process a single URL using the shared reqwest client.
///
/// Acquires a per-domain semaphore permit before fetching (limits inner_n concurrency
/// per domain without blocking the worker task itself — tokio suspends and schedules
/// other tasks while waiting for the permit).
async fn process_one_url(
    seed: SeedURL,
    domain_entry: &Arc<DomainEntry>,
    cfg: &Config,
    adaptive: &Arc<AdaptiveTimeout>,
    inner_n: usize,
    client: &Arc<reqwest::Client>,
    results: &Arc<dyn ResultWriter>,
    failures: &Arc<dyn FailureWriter>,
    stats: &Arc<Stats>,
    peak: &Arc<PeakTracker>,
) {
    // Skip if domain has been abandoned (dead/stalling).
    if domain_entry.abandoned.load(Ordering::Relaxed) {
        stats.skipped.fetch_add(1, Ordering::Relaxed);
        let _ = failures.write_url(FailedURL::new(
            &seed.url,
            &seed.domain,
            "domain_http_timeout_killed",
        ));
        return;
    }

    // Acquire per-domain concurrency permit.
    // tokio suspends this task (not the thread) if inner_n fetches are already in flight.
    let _permit = match domain_entry.semaphore.acquire().await {
        Ok(p) => p,
        Err(_) => return, // semaphore closed (should not happen)
    };

    // Compute effective timeout (adaptive or fixed, capped at 5× base).
    let effective_timeout = if !cfg.disable_adaptive_timeout {
        adaptive
            .timeout(cfg.adaptive_timeout_max)
            .unwrap_or(cfg.timeout)
            .min(cfg.timeout.saturating_mul(5))
    } else {
        cfg.timeout
    };

    // Fetch the URL.
    let result = fetch_one(client, &seed, effective_timeout, cfg.max_body_bytes).await;
    stats.total.fetch_add(1, Ordering::Relaxed);
    peak.record();

    // Classify result and update domain state.
    if !result.error.is_empty() {
        let is_timeout = result.error.contains("timeout")
            || result.error.contains("Timeout")
            || result.error.contains("deadline")
            || result.error.contains("timed out");

        if is_timeout {
            stats.timeout.fetch_add(1, Ordering::Relaxed);
            let t = domain_entry.timeouts.fetch_add(1, Ordering::Relaxed) + 1;
            let s = domain_entry.ok.load(Ordering::Relaxed);

            let ds = DomainState { successes: s, timeouts: t };
            if ds.should_abandon(
                cfg.domain_fail_threshold,
                cfg.domain_dead_probe,
                cfg.domain_stall_ratio,
                inner_n,
            ) {
                // Only emit warning on the first abandonment (swap returns old value).
                if !domain_entry.abandoned.swap(true, Ordering::Relaxed) {
                    stats.push_warning(format!(
                        "abandoned {} (timeouts={}, ok={})",
                        seed.domain, t, s
                    ));
                }
            }

            let _ = failures.write_url(FailedURL {
                url: seed.url.clone(),
                domain: seed.domain.clone(),
                reason: "http_timeout".to_string(),
                error: result.error.clone(),
                status_code: 0,
                fetch_time_ms: result.fetch_time_ms,
                detected_at: chrono::Utc::now().naive_utc(),
            });
        } else {
            stats.failed.fetch_add(1, Ordering::Relaxed);
            let _ = failures.write_url(FailedURL {
                url: seed.url.clone(),
                domain: seed.domain.clone(),
                reason: "http_error".to_string(),
                error: result.error.clone(),
                status_code: result.status_code,
                fetch_time_ms: result.fetch_time_ms,
                detected_at: chrono::Utc::now().naive_utc(),
            });
        }
    } else {
        stats.ok.fetch_add(1, Ordering::Relaxed);
        domain_entry.ok.fetch_add(1, Ordering::Relaxed);
        adaptive.record(result.fetch_time_ms);
    }

    stats
        .bytes_downloaded
        .fetch_add(result.content_length as u64, Ordering::Relaxed);
    let _ = results.write(result);
}
```

**Step 2: Verify it compiles**

```bash
cd tools/crawler && cargo check 2>&1 | tail -10
```
Expected: `Finished` or warnings about unused `process_one_domain` (that's fine for now).

---

## Task 4: Replace run() with flat URL queue architecture

This is the core change — rewrite the `run()` function in `ReqwestEngine`.

**Files:**
- Modify: `tools/crawler/crawler-lib/src/engine/reqwest_engine.rs` (replace run() body, lines ~31–163)

**Step 1: Replace the entire `run()` body** (the `async fn run(...)` implementation inside the `impl super::Engine for ReqwestEngine` block)

The new implementation:

```rust
async fn run(
    &self,
    seeds: Vec<SeedURL>,
    cfg: &Config,
    results: Arc<dyn ResultWriter>,
    failures: Arc<dyn FailureWriter>,
) -> Result<StatsSnapshot> {
    let total_seeds = seeds.len();
    if total_seeds == 0 {
        return Ok(StatsSnapshot {
            ok: 0,
            failed: 0,
            timeout: 0,
            skipped: 0,
            bytes_downloaded: 0,
            total: 0,
            duration: Duration::ZERO,
            peak_rps: 0,
        });
    }

    info!(
        "reqwest engine: {} seeds, {} workers, inner_n={}",
        total_seeds, cfg.workers, cfg.inner_n
    );

    // Group seeds by domain to create per-domain state entries.
    let batches = group_by_domain(seeds);
    let domain_count = batches.len();
    info!("grouped into {} domains", domain_count);

    // Shared stats — use caller-provided Arc for live TUI display, or create fresh.
    let stats = cfg.live_stats.clone().unwrap_or_else(|| Arc::new(Stats::new()));
    if stats.total_seeds.load(Ordering::Relaxed) == 0 {
        stats.total_seeds.store(total_seeds as u64, Ordering::Relaxed);
    }
    let adaptive = Arc::new(AdaptiveTimeout::new());
    let peak = Arc::new(PeakTracker::new());

    let workers = cfg.workers.max(1);
    let inner_n = cfg.inner_n.max(1);

    // Pre-create per-domain entries (semaphore + abandonment state).
    let domain_map: Arc<DashMap<String, Arc<DomainEntry>>> =
        Arc::new(DashMap::with_capacity(domain_count));
    for batch in &batches {
        domain_map.insert(
            batch.domain.clone(),
            Arc::new(DomainEntry::new(inner_n)),
        );
    }

    // Flat URL channel: capacity = workers * 4 so producer never blocks on startup.
    // Each item carries the URL and a reference to its domain's shared state.
    let (url_tx, url_rx) =
        async_channel::bounded::<(SeedURL, Arc<DomainEntry>)>(workers * 4);

    // Producer: flatten all domain batches into the URL channel.
    let dm = Arc::clone(&domain_map);
    let producer = tokio::spawn(async move {
        for batch in batches {
            if let Some(entry_ref) = dm.get(&batch.domain) {
                let entry = Arc::clone(entry_ref.value());
                for url in batch.urls {
                    if url_tx.send((url, Arc::clone(&entry))).await.is_err() {
                        return; // receivers all dropped
                    }
                }
            }
        }
        // url_tx dropped here → channel closes when all receivers see EOF
    });

    // Build ONE shared reqwest::Client for all workers.
    let max_timeout = cfg.timeout.saturating_mul(5);
    let shared_client = match reqwest::Client::builder()
        .pool_max_idle_per_host(inner_n)
        .timeout(max_timeout)
        .danger_accept_invalid_certs(true)
        .redirect(reqwest::redirect::Policy::limited(7))
        .tcp_keepalive(std::time::Duration::from_secs(60))
        .build()
    {
        Ok(c) => Arc::new(c),
        Err(e) => return Err(anyhow::anyhow!("failed to build reqwest client: {}", e)),
    };

    // Spawn N worker tasks.
    // Each worker loops: pop URL from channel → acquire domain semaphore → fetch → update state.
    // Workers never idle between URLs (no domain-batch boundaries).
    let mut worker_handles = Vec::with_capacity(workers);

    for _ in 0..workers {
        let rx = url_rx.clone();
        let cfg = cfg.clone();
        let results = Arc::clone(&results);
        let failures = Arc::clone(&failures);
        let stats = Arc::clone(&stats);
        let adaptive = Arc::clone(&adaptive);
        let peak = Arc::clone(&peak);
        let client = Arc::clone(&shared_client);

        let handle = tokio::spawn(async move {
            while let Ok((seed, domain_entry)) = rx.recv().await {
                process_one_url(
                    seed,
                    &domain_entry,
                    &cfg,
                    &adaptive,
                    inner_n,
                    &client,
                    &results,
                    &failures,
                    &stats,
                    &peak,
                )
                .await;
            }
        });
        worker_handles.push(handle);
    }

    // Wait for producer to finish sending all URLs.
    let _ = producer.await;
    // Close channel so workers see EOF when all URLs are consumed.
    url_rx.close();

    // Wait for all workers to finish.
    for h in worker_handles {
        let _ = h.await;
    }

    stats.peak_rps.store(peak.peak(), Ordering::Relaxed);

    let snapshot = stats.snapshot();
    info!(
        "reqwest engine done: total={} ok={} failed={} timeout={} skipped={} peak_rps={} duration={:.1}s",
        snapshot.total,
        snapshot.ok,
        snapshot.failed,
        snapshot.timeout,
        snapshot.skipped,
        snapshot.peak_rps,
        snapshot.duration.as_secs_f64()
    );

    Ok(snapshot)
}
```

**Step 2: Delete the old `process_one_domain` function** (it's no longer called). Remove the entire `async fn process_one_domain(...)` block and the `compute_domain_timeout` function (now unused).

**Step 3: Run all tests**

```bash
cd tools/crawler && cargo test 2>&1 | tail -20
```
Expected: All tests PASS. If `test_compute_domain_timeout_*` tests fail because that function was removed, delete those tests too.

**Step 4: Commit**

```bash
git add tools/crawler/crawler-lib/src/engine/reqwest_engine.rs
git commit -m "feat(crawler): flat URL task queue with per-domain semaphores

Replace domain-batch workers with flat URL queue + DomainEntry semaphores.
Workers process one URL at a time; per-domain tokio::sync::Semaphore limits
inner_n concurrency per domain. Workers are never idle between domain batches.

Eliminates the 3.6× avg/peak gap (134 effective concurrent → 3,004+).
Expected: 1,684 avg RPS → 5,000+ avg RPS on server2."
```

---

## Task 5: Implement parallel drain in binary.rs

Replace sequential `drain_to_duckdb` with rayon-parallel per-shard insertion.

**Files:**
- Modify: `tools/crawler/crawler-lib/src/writer/binary.rs` (replace `drain_to_duckdb` function, lines ~243–328)

**Step 1: Add rayon import at top of binary.rs**

Add to the `use` block:
```rust
use rayon::prelude::*;
```

**Step 2: Replace `drain_to_duckdb` function body** (keep the function signature, replace the body)

```rust
pub fn drain_to_duckdb(seg_dir: &Path, cfg: &BinDrainConfig) -> Result<u64> {
    let mut paths = list_segment_files(seg_dir)?;
    if paths.is_empty() {
        info!("drain_to_duckdb: no segments found in {:?}", seg_dir);
        return Ok(0);
    }
    paths.sort();

    println!(
        "Draining {} segment(s) → {} DuckDB shard(s) in {:?} (parallel)",
        paths.len(),
        cfg.num_shards,
        cfg.duckdb_dir
    );

    std::fs::create_dir_all(&cfg.duckdb_dir)
        .with_context(|| format!("failed to create DuckDB drain dir {:?}", cfg.duckdb_dir))?;

    let start = std::time::Instant::now();

    // Phase 1 (sequential I/O): read all segment files into memory.
    // Sequential to avoid saturating disk I/O with concurrent reads.
    let mut all_records: Vec<CrawlResult> = Vec::new();
    for (i, path) in paths.iter().enumerate() {
        let seg_start = std::time::Instant::now();
        let mut records = read_one_segment_file::<CrawlResult>(path)
            .with_context(|| format!("reading segment {:?}", path))?;
        println!(
            "  [read {}/{} segs] {:?}: {} records in {:.1}s",
            i + 1,
            paths.len(),
            path.file_name().unwrap_or_default(),
            records.len(),
            seg_start.elapsed().as_secs_f64(),
        );
        all_records.append(&mut records);
        std::fs::remove_file(path)
            .with_context(|| format!("failed to delete drained segment {:?}", path))?;
    }

    let total = all_records.len() as u64;
    println!("  Read complete: {} records in {:.1}s", total, start.elapsed().as_secs_f64());

    // Phase 2 (partition): group records by shard index.
    let mut shard_batches: Vec<Vec<CrawlResult>> =
        (0..cfg.num_shards).map(|_| Vec::new()).collect();
    for r in all_records {
        let idx = shard_for_url(&r.url, cfg.num_shards);
        shard_batches[idx].push(r);
    }

    // Phase 3 (parallel I/O): insert into each shard concurrently.
    // Each rayon worker opens its own DuckDB connection (Connection is not Sync).
    let partition_elapsed = start.elapsed().as_secs_f64();
    println!(
        "  Partition complete in {:.1}s, inserting into {} shards in parallel...",
        partition_elapsed, cfg.num_shards
    );

    let duckdb_dir = &cfg.duckdb_dir;
    let mem_mb = cfg.mem_mb;
    let batch_size = cfg.batch_size;
    let num_shards = cfg.num_shards;

    let errors: Vec<anyhow::Error> = shard_batches
        .into_par_iter()
        .enumerate()
        .filter_map(|(i, batch)| {
            if batch.is_empty() {
                return None;
            }
            let path = duckdb_dir.join(format!("results_{:03}.duckdb", i));
            let conn = match open_result_db(&path, mem_mb) {
                Ok(c) => c,
                Err(e) => return Some(e.context(format!("open shard {i}"))),
            };
            // Insert in batches to avoid huge single transactions.
            for chunk in batch.chunks(batch_size) {
                if let Err(e) = flush_result_batch(&conn, chunk) {
                    return Some(e.context(format!("flush batch to shard {i}")));
                }
            }
            None
        })
        .collect();

    if let Some(e) = errors.into_iter().next() {
        return Err(e);
    }

    println!(
        "Drain complete: {} records in {:.1}s → {:?}",
        total,
        start.elapsed().as_secs_f64(),
        duckdb_dir,
    );

    Ok(total)
}
```

**Step 3: Run tests**

```bash
cd tools/crawler && cargo test writer::binary 2>&1 | tail -20
```
Expected: All binary writer tests pass (drain test still works with parallel impl).

**Step 4: Commit**

```bash
git add tools/crawler/crawler-lib/src/writer/binary.rs
git commit -m "feat(crawler): parallel drain — rayon per-shard DuckDB insertion

Phase 1: sequential read all segments into memory.
Phase 2: partition records by shard hash.
Phase 3: rayon par_iter — each thread inserts into its own DuckDB shard.

Expected: 96s → ~14s for 200K records (7× speedup on 8-shard config)."
```

---

## Task 6: Update auto_config for flat URL queue workers

Fix memory formula: remove inner_n multiplier (flat queue = 1 body/worker, not inner_n bodies).

**Files:**
- Modify: `tools/crawler/crawler-lib/src/config.rs` (update `auto_config` function, ~lines 183–225)

**Step 1: Replace `auto_config` body** with this flat-queue-aware version:

```rust
pub fn auto_config(si: &SysInfo, full_body: bool) -> Config {
    let body_kb: usize = if full_body { 256 } else { 4 };
    // Use total memory (stable) not available (snapshot, can be low at startup).
    // Flat queue: each worker holds 1 body at a time, so formula = total / body_kb.
    // Reserve 25% for OS + DuckDB drain + other processes.
    let total_kb = (si.mem_total_mb as usize) * 1024;
    let fd = si.fd_soft_limit as usize;

    // inner_n: CPU×2 clamped to [4, 16], further limited by fd
    let inner_n = clamp(si.cpu_count * 2, 4, 16).min(fd / 2);

    // workers: memory-limited (1 body per worker) and fd-limited
    let w_mem = total_kb * 75 / 100 / body_kb.max(1);
    let w_fd = fd / 2; // reserve half fd for connections inside each worker
    let workers = clamp(w_mem.min(w_fd).min(16_000), 200, 16_000);

    let db_shards = clamp(si.cpu_count * 2, 4, 16);
    let db_mem_mb = ((si.mem_total_mb as usize) * 10 / 100 / db_shards).max(64);

    let mut cfg = Config::default();
    cfg.workers = workers;
    cfg.inner_n = inner_n;
    cfg.db_shards = db_shards;
    cfg.db_mem_mb = db_mem_mb;
    cfg
}
```

Expected for server2 (12GB total, 24 CPUs, fd=65536):
- inner_n = clamp(48, 4, 16) = 16 → but that might be too high. Actually for HN seeds (mostly dead domains), inner_n=4 is better since dead domains have few URLs. Let me keep inner_n at 4 max for now... actually the formula clamp(cpu×2, 4, 16) gives 16 on a 24-core server, which is too many concurrent per-domain fetches for most domains. But with the semaphore, this just means we can fetch up to 16 URLs in parallel per domain (fine for domains with many URLs, wasteful for domains with 1-2).

Actually wait — the inner_n with flat queue is fine at higher values. It just means domains with few URLs won't use all their permits. The semaphore is created with `inner_n` permits; if a domain has 2 URLs, only 2 permits will ever be acquired. No issue.

For server2 (12GB, 24 CPUs, 65536 fd):
- inner_n = clamp(48, 4, 16) = 16
- w_mem = 12×1024² × 75% / 256 = 38,400 → min(38,400, 16,000) = 16,000 → clamp(16,000, 200, 16,000) = 16,000
- w_fd = 65,536 / 2 = 32,768 → min(16,000, 32,768) = 16,000
- workers = 16,000

That's a big jump from 3,004. 16,000 workers × 256KB max body = 4GB peak. Server2 has 12GB total → fine.

**Step 2: Run all tests**

```bash
cd tools/crawler && cargo test 2>&1 | tail -20
```
Expected: All tests pass.

**Step 3: Commit**

```bash
git add tools/crawler/crawler-lib/src/config.rs
git commit -m "feat(crawler): auto_config uses mem_total_mb + flat-queue formula

Flat URL queue: 1 body per worker (not inner_n bodies). Use total_mb (stable)
instead of available_mb (snapshot, varies). Workers cap raised to 16,000.
Server2: workers 3,004 → 16,000, inner_n 4 → 16."
```

---

## Task 7: Build, deploy, and benchmark

**Step 1: Run full test suite**

```bash
cd tools/crawler && cargo test 2>&1 | tail -30
```
Expected: All tests pass.

**Step 2: Build on server2**

```bash
cd tools/crawler && make build-on-server SERVER=2
```
Expected: `BUILD_OK` (first build ~2min, incremental ~20s).

**Step 3: Deploy to server2**

```bash
cd tools/crawler && make deploy SERVER=2
```

**Step 4: Clean old results and run benchmark**

```bash
ssh -i ~/.ssh/id_ed25519_deploy root@server2 "
  rm -rf ~/data/hn/results/results ~/data/hn/results/results_*.duckdb ~/data/hn/results/failed.duckdb 2>/dev/null
  ~/bin/crawler hn recrawl \
    --seed ~/data/hn/recrawl/hn_pages.duckdb \
    --limit 200000 \
    --writer binary \
    --no-retry \
    2>/dev/null
"
```

**Step 5: Record benchmark results**

Expected outcome (success criteria):
- **avg RPS ≥ 5,000** (up from 1,684)
- **peak RPS ≥ 6,000** (unchanged or higher)
- **drain ≤ 20s** (down from 96.3s)
- Duration ≤ 45s (down from 1m58s)

If avg RPS is below 5,000: check if workers is too low, try `--workers 8192` explicitly.

**Step 6: Commit benchmark results to spec**

Update `spec/0631_modern_crawler_rust.md` with actual benchmark numbers.

```bash
git add spec/0631_modern_crawler_rust.md
git commit -m "docs(spec/0631): add post-flat-queue benchmark results"
```

---

## Task 8: Tune and iterate (if needed)

Only if Task 7 benchmark does not meet 5,000 avg RPS target.

**Option A: More workers**

```bash
ssh root@server2 "~/bin/crawler hn recrawl --seed ... --limit 200000 --writer binary --no-retry --workers 8192 2>/dev/null"
```

**Option B: Increase channel capacity**

If workers are blocked waiting on the URL channel, increase `workers * 4` → `workers * 8`.
Modify `reqwest_engine.rs` run(): `async_channel::bounded(workers * 8)`.

**Option C: hickory-resolver DNS pre-fetch**

If many timeouts are DNS-related (not HTTP), add hickory-resolver:
```toml
hickory-resolver = { version = "0.24", features = ["tokio-runtime"] }
```
Configure reqwest: `.dns_resolver(Arc::new(resolver))` in the client builder.

---

## Verification Checklist

Before considering this complete:
- [ ] All `cargo test` tests pass
- [ ] Binary writer benchmark avg RPS ≥ 5,000 on server2
- [ ] Drain completes in ≤ 20s for 200K records
- [ ] No regression in OK rate vs baseline (10% OK rate expected for HN seeds)
- [ ] spec/0631_modern_crawler_rust.md updated with actual results
- [ ] memory/rust-crawler.md updated with new architecture notes
