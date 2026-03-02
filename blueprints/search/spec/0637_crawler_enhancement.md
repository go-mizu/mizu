# spec/0637 — Rust Crawler Enhancements: 1500 OK/s + Zero False Negatives

**Target**: `crawler cc recrawl --file p:0` on server 2 (12 GB RAM, Ubuntu 24.04 Noble)
**Goals**: ≥1500 successful (HTTP 200) OK responses/second, zero false negatives
  (every URL that would return 200 under any reasonable timeout must be captured)

---

## Root Cause Analysis

Six bugs identified via deep code review of `tools/crawler/`:

### Bug 1 — O(max_urls × num_domains) producer starvation
**File**: `crawler-lib/src/engine/reqwest_engine.rs` + `hyper_engine.rs`
**Code**: The round-robin producer iterates ALL domains for every slot index 0..max_len:
```rust
for slot in 0..max_len {          // e.g. 100 iterations
    for (urls, entry) in &domain_batches {  // e.g. 500K domains
        if let Some(url) = urls.get(slot) { // 99% return None
```
With CC p:0 (~500K domains, max domain has ~100 URLs): 100 × 500K = **50M iterations**,
~99% wasted. Workers drain the channel and go idle while the producer spins.
This is the single largest throughput bottleneck.

### Bug 2 — Abandoned-domain URLs never retried (false-negative source)
**File**: `crawler-lib/src/seed.rs` (`load_retry_seeds`)
When a domain is abandoned mid-crawl (dead/stall probe), remaining URLs in the
async_channel queue are written with `reason="domain_http_timeout_killed"` but the
pass-2 loader SQL is:
```sql
WHERE reason = 'http_timeout' AND detected_at >= ?
```
All `domain_http_timeout_killed` URLs are **silently dropped**. On CC data with many
slow domains, this is the primary false-negative source.

### Bug 3 — Adaptive timeout feedback loop (false-negative source)
**File**: `crawler-lib/src/engine/reqwest_engine.rs` (`process_one_url`)
`adaptive.record()` is called only on **successful** responses. If 50% of URLs timeout
(not recorded), P95 is biased toward fast responses only (e.g., 150ms). Effective
timeout = min(P95×2, cfg.timeout×5) = 300ms — URLs needing 400–800ms now timeout too,
lowering P95 further. Positive feedback loop kills medium-speed valid pages.

### Bug 4 — Body store blocks Tokio worker tasks
**File**: `crawler-lib/src/bodystore.rs` + `reqwest_engine.rs` (`fetch_one`)
`store.put(&body_bytes)` runs synchronously in the hot fetch path:
SHA-256 + `path.exists()` + `create_dir_all` + `GzEncoder` + `File::create` +
`write_all` + `rename`. At 1500 OK/s this is 1500 blocking file ops/sec inside
Tokio tasks, stealing thread-pool slots from network I/O.

### Bug 5 — 3 GB peak memory for CC seed loading
**File**: `crawler-lib/src/seed.rs` (`load_seeds_cc_parquet`)
Full materialization: 10M seeds × ~150 bytes = 1.5 GB (Vec<SeedURL>), then
`group_by_domain` sorts+copies = another 1.5 GB. On server 2 (12 GB), this consumes
25% of RAM before the crawl starts, competing with DuckDB + binary writer + body store.
Also: a `COUNT(*)` pre-scan doubles the parquet read time for no value.

### Bug 6 — Async channel capacity mismatched for deque producer
Once the producer fix (Bug 1) is applied, the channel cap of `workers * 4` (8000) is
the right size for bursting but the seed channel (new, from Bug 5 fix) must be
independently sized to avoid back-pressure coupling between the parquet reader and the
URL dispatcher.

---

## Enhancements

### E1 — Adaptive timeout floor
**Files**: `crawler-lib/src/engine/reqwest_engine.rs`, `hyper_engine.rs`

Change the effective timeout computation so adaptive can only **extend** the timeout
(for slow domains), never reduce it below the configured baseline:

```rust
// Before
let effective_timeout = adaptive
    .timeout(cfg.adaptive_timeout_max)
    .unwrap_or(cfg.timeout)
    .min(cfg.timeout.saturating_mul(5));

// After
let adaptive_val = adaptive
    .timeout(cfg.adaptive_timeout_max)
    .unwrap_or(cfg.timeout);
let effective_timeout = adaptive_val
    .max(cfg.timeout)                   // floor: never below configured baseline
    .min(cfg.timeout.saturating_mul(3)); // tighter ceiling: ×3 not ×5
```

The `×3` ceiling (vs current `×5`) still gives slow domains 3× the base timeout while
preventing runaway adaptive growth.

### E2 — Include skipped URLs in pass-2 retry
**Files**: `crawler-lib/src/seed.rs`

Add `domain_http_timeout_killed` to the retry SQL:

```rust
// Before
"SELECT url, COALESCE(domain, '') as domain FROM failed_urls \
 WHERE reason = 'http_timeout' AND detected_at >= ?"

// After
"SELECT url, COALESCE(domain, '') as domain FROM failed_urls \
 WHERE reason IN ('http_timeout', 'domain_http_timeout_killed') \
   AND detected_at >= ?"
```

No schema or writer changes required — domain/url/detected_at are already written
correctly for killed URLs. Pass-2 config (`disable_adaptive_timeout=true`,
`domain_dead_probe=2`) handles both timeout and abandoned-domain retries correctly.

### E3 — O(N) deque-based producer
**Files**: `crawler-lib/src/engine/reqwest_engine.rs`, `hyper_engine.rs`

Replace the `O(max_len × num_domains)` nested loop with a `VecDeque`-based
round-robin. Every iteration sends exactly one URL:

```rust
// Before: O(max_len × num_domains), ~50M wasted iterations on CC p:0
for slot in 0..max_len {
    for (urls, entry) in &domain_batches {
        if let Some(url) = urls.get(slot) { ... }
    }
}

// After: O(total_seeds), zero wasted iterations
use std::collections::VecDeque;
let mut queue: VecDeque<(VecDeque<SeedURL>, Arc<DomainEntry>)> = batches
    .into_iter()
    .filter_map(|b| dm.get(&b.domain)
        .map(|e| (VecDeque::from(b.urls), Arc::clone(e.value()))))
    .collect();

while let Some((mut urls, entry)) = queue.pop_front() {
    let url = urls.pop_front().unwrap();
    if url_tx.send((url, Arc::clone(&entry))).await.is_err() { return; }
    if !urls.is_empty() { queue.push_back((urls, entry)); }
}
```

Applied to both `reqwest_engine.rs` and `hyper_engine.rs`.

**Memory note**: This still holds all seeds in the producer's deque. E5 (streaming)
removes this; E3 is the algorithm fix, E5 is the memory fix. Both are needed.

### E4 — Async BodyStore
**Files**: `crawler-lib/src/bodystore.rs`

Split `put()` into two phases:

**Phase 1 (in-task, microseconds)**:
- Compute SHA-256 → derive CID string
- Check `Arc<DashSet<[u8; 32]>>` (in-memory set of already-written hashes)
- If present: return CID immediately, skip write entirely
- If absent: mark as pending, send `(body_bytes, path)` to bounded crossbeam channel

**Phase 2 (background OS thread)**:
- Single dedicated thread drains the write channel
- GzEncoder + atomic rename write (same as current)
- On write complete: insert hash into `DashSet`

New API:
```rust
impl BodyStore {
    // Synchronous fast path: returns CID, schedules write
    pub fn put_async(&self, body: &[u8]) -> String { ... }
    // Called at crawl end to flush pending writes
    pub fn close(&self) -> Result<()> { ... }
}
```

Config change: `BodyStore` gets wrapped in a new `AsyncBodyStore` struct that owns the
channel + thread. `Config.body_store` type changes from `Option<Arc<BodyStore>>` to
`Option<Arc<AsyncBodyStore>>`.

### E5 — Streaming seed loading
**Files**: `crawler-lib/src/seed.rs`, `crawler-lib/src/engine/{reqwest,hyper}_engine.rs`,
           `crawler-cli/src/common.rs`, `crawler-cli/src/cc.rs`

**5a. Remove COUNT(*) prescan** in `load_seeds_cc_parquet`.
Delete the count query entirely. The "~{est_mb} MB heap" print is removed.

**5b. Streaming via DuckDB Arrow batches**

Replace the Vec-materializing loader with a streaming function that feeds a bounded
async channel:

```rust
/// Stream CC seeds from parquet into `tx`, `batch_size` rows at a time.
/// Caller closes the channel when done; engine reads from the `rx` end.
pub fn stream_seeds_cc_parquet(
    path: &str,
    limit: usize,
    filters: &CcSeedFilter,
    tx: async_channel::Sender<SeedURL>,
) -> Result<()>  // runs in tokio::task::spawn_blocking
```

Uses DuckDB's `query_arrow()` to get `RecordBatch` stream, converts each batch to
`SeedURL` and sends through channel (back-pressure via bounded capacity = 200K).

**Engine signature change**:

```rust
// Before
async fn run(seeds: Vec<SeedURL>, cfg: &Config, ...) -> Result<StatsSnapshot>;

// After
async fn run(
    seed_rx: async_channel::Receiver<SeedURL>,
    total_seeds_hint: Option<u64>,  // for TUI progress % (from parquet metadata, optional)
    cfg: &Config,
    ...
) -> Result<StatsSnapshot>;
```

**Lazy domain grouping in producer**:
The producer no longer calls `group_by_domain` upfront. Instead, it reads from
`seed_rx` and lazily inserts into a `DashMap<String, Arc<DomainEntry>>` + a
`VecDeque<(VecDeque<SeedURL>, Arc<DomainEntry>)>` (the deque from E3):

```rust
// Producer loop (simplified):
loop {
    // Fill deque from seed_rx when it drops below LOW_WATERMARK
    while deque.len() < LOW_WATERMARK {
        match seed_rx.try_recv() {
            Ok(seed) => insert_into_deque(&mut deque, &mut domain_map, seed),
            Err(TryRecvError::Empty) => break,
            Err(TryRecvError::Closed) => { seed_rx_done = true; break; }
        }
    }
    // Send one URL from deque head
    if let Some((mut urls, entry)) = deque.pop_front() {
        url_tx.send((urls.pop_front().unwrap(), entry.clone())).await?;
        if !urls.is_empty() { deque.push_back((urls, entry)); }
    } else if seed_rx_done {
        break;
    } else {
        // Both deque empty and seed_rx not done yet — yield to let seeds arrive
        tokio::task::yield_now().await;
    }
}
```

**Memory**: peak seed footprint = LOW_WATERMARK × ~150 bytes = ~30 MB (200K watermark)
vs current 3 GB.

**Backward compatibility**: `run_job()` and callers that have a `Vec<SeedURL>` (HN,
direct --seed flag) convert with a helper:
```rust
pub fn vec_to_receiver(seeds: Vec<SeedURL>) -> (async_channel::Receiver<SeedURL>, u64) {
    let total = seeds.len() as u64;
    let (tx, rx) = async_channel::bounded(seeds.len().max(1));
    for s in seeds { let _ = tx.try_send(s); }
    drop(tx);
    (rx, total)
}
```

---

## Expected Performance (server 2)

| Metric | Before | After |
|--------|--------|-------|
| Producer iterations (CC p:0) | ~50M (99% wasted) | ~10M (100% useful) |
| Peak seed RAM | ~3 GB | ~30 MB |
| Worker idle (producer starvation) | High | Near zero |
| Pass-2 retry coverage | `http_timeout` only | `http_timeout` + `domain_http_timeout_killed` |
| Adaptive timeout on fast-domain bias | Drops to 300ms (kills 400ms URLs) | Floor = cfg.timeout (1000ms) |
| Body store latency in hot path | SHA256+gzip+write per OK | SHA256 only (write is async) |
| Workers | 2000 (unchanged) | 2000 (unchanged) |
| **Expected OK/s** | ~600–900 OK/s (estimated) | **≥1500 OK/s** |

---

## File Change Summary

| File | Changes |
|------|---------|
| `crawler-lib/src/engine/reqwest_engine.rs` | E1 (adaptive floor), E3 (deque producer), E5 (seed_rx signature) |
| `crawler-lib/src/engine/hyper_engine.rs` | E1 (adaptive floor), E3 (deque producer), E5 (seed_rx signature) |
| `crawler-lib/src/engine/mod.rs` | E5 (Engine trait signature) |
| `crawler-lib/src/seed.rs` | E2 (retry SQL), E5a (remove COUNT), E5b (stream_seeds_cc_parquet) |
| `crawler-lib/src/bodystore.rs` | E4 (AsyncBodyStore with background thread) |
| `crawler-lib/src/config.rs` | E4 (body_store type: BodyStore → AsyncBodyStore) |
| `crawler-lib/src/job.rs` | E5 (accept seed_rx, pass total_seeds_hint to engine) |
| `crawler-cli/src/common.rs` | E5 (vec_to_receiver helper, update run_crawl_job) |
| `crawler-cli/src/cc.rs` | E5 (call stream_seeds_cc_parquet instead of load_seeds_cc_parquet) |
| `crawler-cli/src/hn.rs` | E5 (use vec_to_receiver for existing Vec<SeedURL> path) |
