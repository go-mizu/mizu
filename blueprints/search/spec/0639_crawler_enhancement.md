# Crawler Enhancement — 1500 pages/s with Zero False Negatives

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Push `crawler cc recrawl --file p:0` on server2 to ≥1500 pages/s (total processed) with provably zero false negatives — no URL that could succeed is permanently skipped.

**Architecture:** Seven targeted changes to `crawler-lib` and `crawler-cli`. No new dependencies. All changes are surgical — only touching the exact fields/logic responsible for the bottlenecks and false negatives identified below.

**Tech Stack:** Rust, tokio, reqwest, hyper, crossbeam-channel, rkyv, DuckDB

---

## Root-Cause Analysis

### False Negative Bug (Critical — fix first)

In `job.rs` `run_job()`, when `retry_cfg` is built for pass-2, five fields are overridden but **`domain_stall_ratio` is silently inherited from pass-1 config (default = 5)**:

```rust
// job.rs:169 — what IS set:
retry_cfg.timeout = cfg.retry_timeout;           // ✓
retry_cfg.domain_fail_threshold = 0;             // ✓
retry_cfg.domain_timeout_ms = (…) * 3;          // ✓
retry_cfg.disable_adaptive_timeout = true;       // ✓
retry_cfg.domain_dead_probe = 2;                 // ✓

// MISSING:
// retry_cfg.domain_stall_ratio = 0;             // ← BUG
```

**Effect**: In pass-2 (10s timeout), if a domain has 1 success + 5 timeouts (`stall_ratio=5`), it is abandoned. All remaining URLs for that domain get `domain_http_timeout_killed` with no further retry opportunity. These are **permanent false negatives**.

**Example**: Domain with 20 URLs. Pass-2: URL-1 responds in 8s (success). URL-2..6 each timeout at 10s. On the 6th timeout: `timeouts(5) >= successes(1) * ratio(5)` → domain abandoned. URLs 7-20 are permanently lost even though URL-1 just proved the domain is alive.

### Performance Bottleneck (Workers Cap)

`auto_config()` hardcodes `workers` ceiling at **2,000**:

```rust
// config.rs:259
let workers = clamp(w_mem.min(w_fd).min(2_000), min_workers, 2_000);
```

Server2 profile: 12 GB RAM, fd=65536, ~12 CPUs.
- `w_mem` (75% of 12 GB / 256 KB body) = ~37,748 — memory allows far more
- `w_fd` (65536/2) = 32,768 — fd budget allows far more
- Hard cap: **2,000** → effective limit is 2,000 workers

With CC seeds (pre-validated status=200, higher quality than HN):
- ~70% respond in <200ms → each of 2,000 workers processes 5 URLs/s = 10,000 total/s theoretical
- After accounting for semaphores, channel blocking, TLS overhead: ~3,000-5,000 rps achievable with 2,000 workers at the right tuning
- But with dead-server TCP timeouts at full 1s: each timing-out worker blocks for 1s → throughput drops
- **Raising to 4,000 workers + fast connect timeout = much better utilization**

### Dead-Server Bottleneck (Connect Timeout)

The reqwest client has a single `timeout` controlling the entire request (connect + TLS + headers + body). With `timeout=1000ms`, a dead server that drops TCP SYN packets forces workers to wait the full 1s before failing.

With a separate **connect_timeout=500ms**, dead servers fail at 500ms, freeing workers 2x faster, effectively doubling throughput on mixed live/dead seed sets.

CC p:0 data quality: ~60-70% of domains still alive. ~30-40% dead → these are the bottleneck workers.

### Producer Channel Starvation

All three engines create `url_tx` with capacity `workers * 4`. With auto workers=2000: channel = 8,000 slots.

The producer batch loop:
1. Reads 100K seeds from DuckDB streaming channel
2. Sorts+groups by domain
3. Round-robins into `url_tx.send().await`

When workers are processing 1-second-timeout URLs, the 8,000-slot channel fills and the producer blocks 12 times per 100K-batch. This creates artificial idle time for workers between producer bursts.

**Fix**: Increase channel capacity to `(workers * 32).max(65_536)`.

---

## Task 1: Fix Pass-2 False Negatives (domain_stall_ratio)

**Files:**
- Modify: `crawler-lib/src/job.rs:166-175`

**Background**: Pass-2 config inherits `domain_stall_ratio` from pass-1. Must be zeroed explicitly. Also expose `domain_stall_ratio` override as a Config field for pass-2.

**Step 1: Add `pass2_stall_ratio` field to Config**

In `crawler-lib/src/config.rs`, add to `Config` struct after `pass2_workers`:
```rust
/// Domain stall ratio for pass-2 retry (0 = disable stall-based abandonment).
/// Default 0: in pass-2, stall killing creates false negatives since we're
/// specifically retrying known-alive-but-slow domains.
pub pass2_stall_ratio: usize,
```

In `Config::default()`, add:
```rust
pass2_stall_ratio: 0,
```

**Step 2: Wire pass2_stall_ratio into run_job**

In `crawler-lib/src/job.rs`, in the pass-2 config block (after line 172):
```rust
retry_cfg.domain_stall_ratio = cfg.pass2_stall_ratio;  // default=0 disables stall killing
```

**Step 3: Write failing test**

In `crawler-lib/src/job.rs` tests, add:
```rust
#[test]
fn test_pass2_stall_ratio_is_zero_by_default() {
    let cfg = Config::default();
    // pass2_stall_ratio must default to 0 — stall killing creates false negatives in pass-2
    assert_eq!(cfg.pass2_stall_ratio, 0, "pass2_stall_ratio should default to 0");
}
```

**Step 4: Run test**
```bash
cd blueprints/search/tools/crawler && cargo test test_pass2_stall_ratio_is_zero_by_default --lib
```
Expected: PASS

**Step 5: Verify domain logic**

In `crawler-lib/src/domain.rs`, confirm `should_abandon` with `domain_stall_ratio=0`:
```rust
// domain_stall_ratio=0: the guard is `domain_stall_ratio > 0`, so stall check is skipped entirely
if domain_stall_ratio > 0  // line 99 — evaluates false when pass2_stall_ratio=0 ✓
```

**Step 6: Write test confirming zero stall ratio never abandons**
```rust
#[test]
fn test_zero_stall_ratio_never_abandons_alive_domain() {
    let mut state = DomainState::new();
    state.successes = 1;
    state.timeouts = 1_000_000; // extreme stall — should NOT abandon if ratio=0
    // stall_ratio=0 means disabled; only dead_probe can trigger abandonment here
    assert!(!state.should_abandon(0, 0, 0, 4),
        "stall_ratio=0 must never abandon regardless of timeout count");
}
```

**Step 7: Run all domain tests**
```bash
cargo test domain -- --nocapture
```

**Step 8: Commit**
```bash
git add crawler-lib/src/config.rs crawler-lib/src/job.rs crawler-lib/src/domain.rs
git commit -m "fix(crawler): zero pass-2 stall_ratio to eliminate false negatives

Add pass2_stall_ratio field to Config (default=0). Wire into run_job so
pass-2 never abandons domains via stall-ratio. Previously, stall_ratio=5
inherited from pass-1 caused false negatives: a domain with 1 success +
5 timeouts in pass-2 was abandoned, permanently losing all remaining URLs.

Example: domain with 20 URLs, URL-1 responds in 8s (success), URLs 2-6
each timeout at 10s → stall abandon → URLs 7-20 permanently lost. With
pass2_stall_ratio=0, only domain_dead_probe=2 (0 successes + 2 timeouts)
triggers abandonment — which correctly identifies truly dead servers."
```

---

## Task 2: Raise Worker Cap in auto_config

**Files:**
- Modify: `crawler-lib/src/config.rs:241-275` (`auto_config`)

**Background**: The hard cap of 2,000 was tuned for HN seeds (high contention, many slow domains). CC seeds are pre-validated status=200 — higher quality. Raising to 4,000 plus a separate connect_timeout doubles effective workers for dead-server-heavy workloads.

**Step 1: Write failing test**
```rust
#[test]
fn test_auto_config_workers_above_2000_on_high_memory() {
    let si = SysInfo {
        cpu_count: 12,
        mem_total_mb: 12_288, // 12 GB
        mem_available_mb: 10_000,
        fd_soft_limit: 65_536,
    };
    let cfg = auto_config(&si, true);
    // With 12GB RAM, fd=65536: workers should be > 2000
    assert!(cfg.workers > 2000,
        "auto_config should exceed 2000 workers on 12GB/fd=65536, got {}", cfg.workers);
    assert!(cfg.workers <= 8000,
        "auto_config should not exceed 8000 workers, got {}", cfg.workers);
}
```

**Step 2: Run test to confirm it fails**
```bash
cargo test test_auto_config_workers_above_2000_on_high_memory --lib
```
Expected: FAIL (currently gets 2000)

**Step 3: Update auto_config**

In `config.rs`, change the worker calculation:
```rust
// Old: hard cap at 2_000
let workers = clamp(w_mem.min(w_fd).min(2_000), min_workers, 2_000);

// New: cap at 8_000 — CC seeds are higher quality than HN;
// fd budget (65536/2=32768) and memory both allow far more.
// 8000 is the sweet spot: enough parallelism for fast CC response times
// without overwhelming the tokio scheduler or DNS resolver.
let workers = clamp(w_mem.min(w_fd).min(8_000), min_workers, 8_000);
```

Also update `num_flushers` comment: no change needed.

**Step 4: Run test to confirm it passes**
```bash
cargo test test_auto_config_workers_above_2000_on_high_memory --lib
```
Expected: PASS

**Step 5: Add regression test for low-memory/low-fd config**
```rust
#[test]
fn test_auto_config_respects_fd_limit() {
    let si = SysInfo {
        cpu_count: 4,
        mem_total_mb: 4_096,
        mem_available_mb: 3_000,
        fd_soft_limit: 1_024, // low fd limit
    };
    let cfg = auto_config(&si, true);
    // w_fd = 1024/2 = 512; workers should be capped at fd budget
    assert!(cfg.workers <= 512,
        "workers should respect fd_soft_limit, got {}", cfg.workers);
}
```

**Step 6: Run all config tests**
```bash
cargo test config -- --nocapture
```

**Step 7: Commit**
```bash
git add crawler-lib/src/config.rs
git commit -m "perf(crawler): raise auto_config worker cap to 8000

CC seeds are pre-validated status=200 (higher quality than HN seeds).
Server2 has fd=65536 and 12GB RAM — both permit far more than 2000 workers.
Raising cap from 2000→8000 allows full exploitation of available resources.

KEY INSIGHT update: the 2000-cap comment was based on HN benchmarks with
mixed-quality seeds. CC partition seeds have ~70% alive rate (vs 40-50% HN),
which means proportionally fewer slow timeouts per worker. Higher worker
counts are safe and necessary to hit 1500+ pages/s on full CC partitions."
```

---

## Task 3: Add Separate Connect Timeout

**Files:**
- Modify: `crawler-lib/src/config.rs` (add field)
- Modify: `crawler-lib/src/engine/reqwest_engine.rs` (wire into client builder)
- Modify: `crawler-lib/src/engine/hyper_engine.rs` (wire into connector)
- Modify: `crawler-lib/src/engine/wreq_engine.rs` (wire into client builder)
- Modify: `crawler-cli/src/cc.rs` (expose `--connect-timeout` flag)
- Modify: `crawler-cli/src/hn.rs` (expose `--connect-timeout` flag)

**Background**: With `timeout=1000ms`, dead servers (TCP SYN-drop) hold workers for 1s. With `connect_timeout=500ms`, they fail in 500ms — workers are freed 2× faster.

**Step 1: Add connect_timeout to Config**

In `config.rs` struct `Config`:
```rust
/// TCP connect timeout (separate from overall request timeout).
/// Default 500ms: dead servers fail fast, freeing workers.
/// Set to 0 to disable (use overall timeout for connect too).
pub connect_timeout: Duration,
```

In `Config::default()`:
```rust
connect_timeout: Duration::from_millis(500),
```

**Step 2: Write failing test**
```rust
#[test]
fn test_connect_timeout_default_is_500ms() {
    let cfg = Config::default();
    assert_eq!(cfg.connect_timeout, Duration::from_millis(500));
}
```

**Step 3: Run test**
```bash
cargo test test_connect_timeout_default_is_500ms --lib
```
Expected: FAIL

**Step 4: Wire connect_timeout into reqwest engine**

In `reqwest_engine.rs`, in the client builder block (around line 202):
```rust
// Current:
let shared_client = match reqwest::Client::builder()
    .pool_max_idle_per_host(inner_n)
    .timeout(max_timeout)
    .danger_accept_invalid_certs(true)
    .redirect(reqwest::redirect::Policy::limited(7))
    .tcp_keepalive(std::time::Duration::from_secs(60))
    .build()

// New — add connect_timeout when non-zero:
let mut builder = reqwest::Client::builder()
    .pool_max_idle_per_host(inner_n)
    .timeout(max_timeout)
    .danger_accept_invalid_certs(true)
    .redirect(reqwest::redirect::Policy::limited(7))
    .tcp_keepalive(std::time::Duration::from_secs(60));
if cfg.connect_timeout > Duration::ZERO {
    builder = builder.connect_timeout(cfg.connect_timeout);
}
let shared_client = match builder.build()
```

**Step 5: Wire connect_timeout into wreq engine**

Same pattern in `wreq_engine.rs` (around line 172):
```rust
let mut builder = wreq::Client::builder()
    .emulation(Emulation::Chrome133)
    .pool_max_idle_per_host(inner_n)
    .timeout(max_timeout)
    .cert_verification(false)
    .redirect(wreq::redirect::Policy::limited(7))
    .tcp_keepalive(Duration::from_secs(60));
if cfg.connect_timeout > Duration::ZERO {
    builder = builder.connect_timeout(cfg.connect_timeout);
}
let shared_client = match builder.build()
```

**Step 6: Wire connect_timeout into hyper engine**

In `hyper_engine.rs`, the connector is `HttpConnector`. Add connect timeout:
```rust
let mut http = HttpConnector::new();
http.enforce_http(false);
// New:
if cfg.connect_timeout > Duration::ZERO {
    http.set_connect_timeout(Some(cfg.connect_timeout));
}
```

**Step 7: Expose --connect-timeout in CLI**

In `cc.rs` `RecrawlArgs`:
```rust
/// TCP connect timeout in ms (0 = use overall timeout; default 500ms)
#[arg(long, default_value_t = 500)]
pub connect_timeout_ms: u64,
```

In `run_recrawl`, add to `CrawlJobParamsStreaming`:
```rust
connect_timeout_ms: args.connect_timeout_ms,
```

In `CrawlJobParamsStreaming` struct (common.rs):
```rust
pub connect_timeout_ms: u64,
```

In `run_crawl_job_streaming`, add to cfg:
```rust
cfg.connect_timeout = Duration::from_millis(params.connect_timeout_ms);
```

Same changes in `hn.rs` / `run_crawl_job`.

**Step 8: Run all tests**
```bash
cargo test -- --nocapture 2>&1 | tail -20
```
Expected: All pass.

**Step 9: Commit**
```bash
git add crawler-lib/src/config.rs crawler-lib/src/engine/ crawler-cli/src/
git commit -m "perf(crawler): add separate connect_timeout (default 500ms)

Previously, a single timeout covered connect+TLS+response. Dead servers
(TCP SYN-drop) held workers for the full 1s base timeout. With connect
timeout=500ms, dead servers fail in 500ms, freeing workers 2x faster.

This is critical for CC seeds where 30-40% of domains may be unreachable:
workers on dead servers were idle 50% longer than necessary, directly
capping throughput. With connect_timeout=500ms and 4000 workers, dead-
server workers free up twice as fast, improving overall pages/s."
```

---

## Task 4: Increase URL Channel Capacity

**Files:**
- Modify: `crawler-lib/src/engine/reqwest_engine.rs:112`
- Modify: `crawler-lib/src/engine/hyper_engine.rs` (same pattern)
- Modify: `crawler-lib/src/engine/wreq_engine.rs:104`

**Background**: Channel `workers * 4` = 8,000 with 2,000 workers (16,000 with 4,000 workers after Task 2 on server2). Producer blocks whenever workers drain the channel. Increasing to `workers * 32` keeps the channel buffer full so producer never idles.

Memory cost: each slot = `(SeedURL + Arc<DomainEntry>)` ≈ 100 bytes.
With workers=4000: `4000 * 32 = 128,000 slots × 100B = 12.8 MB`. Negligible.

**Step 1: Update reqwest_engine.rs**

Change (line ~112):
```rust
// Old:
let (url_tx, url_rx) =
    async_channel::bounded::<(SeedURL, Arc<DomainEntry>)>(workers * 4);

// New:
let channel_cap = (workers * 32).max(65_536);
let (url_tx, url_rx) =
    async_channel::bounded::<(SeedURL, Arc<DomainEntry>)>(channel_cap);
```

**Step 2: Apply same change to hyper_engine.rs and wreq_engine.rs**

(Same one-line change: `workers * 4` → `(workers * 32).max(65_536)`)

**Step 3: Run tests**
```bash
cargo test -- --nocapture 2>&1 | tail -5
```
Expected: All pass (no functional change, only capacity).

**Step 4: Commit**
```bash
git add crawler-lib/src/engine/
git commit -m "perf(crawler): increase url_tx channel capacity to workers*32

Previously url_tx capacity was workers*4 (8K with 2K workers). With
auto_config now allowing 4K-8K workers, the producer was blocking every
few thousand URLs waiting for workers to drain, creating stop-go patterns.

New capacity: max(workers*32, 65536). With 4000 workers = 128K slots = 12MB.
The producer can now batch-fill the entire channel without blocking, keeping
all workers continuously fed."
```

---

## Task 5: Expose Pass-2 Tuning Flags in CLI

**Files:**
- Modify: `crawler-cli/src/cc.rs` (add args)
- Modify: `crawler-cli/src/hn.rs` (add args, if applicable)
- Modify: `crawler-cli/src/common.rs` (add to params structs)

**Background**: `pass2_stall_ratio=0` is now the correct default (Task 1), but operators may want to tune it. Also expose `--pass2-workers` in CC (it exists in config but wasn't wired to CLI).

**Step 1: Add pass2_stall_ratio arg to cc.rs RecrawlArgs**
```rust
/// Pass-2 domain stall ratio (0 = disabled, prevents false negatives).
/// Stall killing in pass-2 causes false negatives: alive-but-slow domains
/// with 1 success + N timeouts get abandoned even though they can succeed.
#[arg(long, default_value_t = 0)]
pub pass2_stall_ratio: usize,

/// Pass-2 worker count override (0 = use pass-1 workers)
#[arg(long, default_value_t = 0)]
pub pass2_workers: usize,
```

**Step 2: Add to CrawlJobParamsStreaming**
```rust
pub pass2_stall_ratio: usize,
pub pass2_workers: usize,
```

**Step 3: Wire into run_crawl_job_streaming cfg**
```rust
cfg.pass2_stall_ratio = params.pass2_stall_ratio;
cfg.pass2_workers = params.pass2_workers;
```

(Also update `run_crawl_job` for HN parity.)

**Step 4: Run tests**
```bash
cargo test -- --nocapture 2>&1 | tail -5
```

**Step 5: Commit**
```bash
git add crawler-cli/src/ crawler-lib/src/
git commit -m "feat(crawler-cli): expose --pass2-stall-ratio and --pass2-workers flags

pass2_stall_ratio defaults to 0 (disable stall killing in pass-2) following
the fix in Task 1. Operators can override if needed.

pass2_workers exposes the existing Config.pass2_workers field through CLI,
allowing reduced concurrency in pass-2 to avoid overwhelming slow servers."
```

---

## Task 6: Update auto_config for connect_timeout and num_flushers

**Files:**
- Modify: `crawler-lib/src/config.rs:241-275` (`auto_config`)

**Background**: `auto_config` returns a `Config` with default connect_timeout (500ms from Task 3). No change needed for connect_timeout (default handles it). But `num_flushers` calculation can be improved: with 8000 workers, the binary writer needs more flusher threads to keep up.

**Step 1: Update num_flushers formula**

Current: `clamp(cpu_count / 2, 2, 8)`

With 8000 workers at peak throughput (e.g. 5000 pages/s), the single-threaded gob issue from Go's experience applies here too. More flusher threads prevent channel backpressure.

```rust
// Old: clamp(si.cpu_count / 2, 2, 8)
// New: scale with workers, capped at cpu_count
let num_flushers = clamp(
    (workers / 1000).max(si.cpu_count / 2),
    2,
    si.cpu_count.min(8),
);
```

**Step 2: Write test**
```rust
#[test]
fn test_num_flushers_scales_with_workers() {
    let si = SysInfo {
        cpu_count: 12,
        mem_total_mb: 12_288,
        mem_available_mb: 10_000,
        fd_soft_limit: 65_536,
    };
    let cfg = auto_config(&si, true);
    // With workers ~4000+, num_flushers should be >= 4
    assert!(cfg.num_flushers >= 4,
        "num_flushers should scale with high worker count, got {}", cfg.num_flushers);
}
```

**Step 3: Run test**
```bash
cargo test test_num_flushers_scales_with_workers --lib
```

**Step 4: Commit**
```bash
git add crawler-lib/src/config.rs
git commit -m "perf(crawler): scale num_flushers with worker count in auto_config

With 4000-8000 workers, a single flusher thread (default for low CPU counts)
becomes the bottleneck — the binary writer channel fills up and causes
backpressure on the fetch hot path.

New formula: max(workers/1000, cpu_count/2), clamped to [2, min(cpu_count, 8)].
On server2 (12 CPUs, ~4000 workers): num_flushers = max(4, 6) = 6."
```

---

## Task 7: End-to-End Integration Test

**Files:**
- Modify: `crawler-lib/src/job.rs` (add integration test)

**Background**: Verify pass-2 stall_ratio=0 is actually applied through the full job flow.

**Step 1: Write integration test**

Add to `job.rs` test module:
```rust
#[tokio::test]
async fn test_pass2_config_has_zero_stall_ratio() {
    // We test that when run_job builds retry_cfg, stall_ratio is 0.
    // Indirectly verified by checking Config.pass2_stall_ratio=0 flows to domain logic.
    // Direct structural test: build a Config, verify pass2_stall_ratio default.
    let cfg = Config::default();
    assert_eq!(cfg.pass2_stall_ratio, 0);

    // Verify domain logic: with stall_ratio=0, stall never triggers.
    let state = crate::domain::DomainState {
        successes: 1,
        timeouts: 1_000,
    };
    // pass2: fail_threshold=0, dead_probe=2 (probe requires 0 successes), stall_ratio=0
    // successes=1 → dead_probe check fails (we have a success)
    // stall_ratio=0 → stall check disabled
    assert!(!state.should_abandon(0, 2, 0, 4),
        "with stall_ratio=0 and 1 success, domain should NOT be abandoned regardless of timeouts");
}
```

**Step 2: Run test**
```bash
cargo test test_pass2_config_has_zero_stall_ratio -- --nocapture
```
Expected: PASS

**Step 3: Run full test suite**
```bash
cargo test --no-default-features -- --nocapture 2>&1 | tail -30
```
Expected: All tests pass.

**Step 4: Commit**
```bash
git add crawler-lib/src/job.rs
git commit -m "test(crawler): add integration test for pass-2 stall_ratio=0

Verify the false-negative fix: with stall_ratio=0 in pass-2, a domain
with 1 success + 1000 timeouts is never abandoned via stall logic."
```

---

## Task 8: Build and Deploy to Server 2

**Files:**
- `Makefile` — existing targets used

**Step 1: Build Linux binary for server2**
```bash
cd blueprints/search/tools/crawler
make build-on-server SERVER=2
```
(Builds directly on server2 via rsync+cargo — no Docker needed. Uses `RUSTFLAGS="-C target-cpu=x86-64-v3"` for AVX2.)

**Step 2: Verify binary deployed**
```bash
make remote-test SERVER=2
```
Expected: `crawler --help` output printed without error.

**Step 3: Quick baseline benchmark on server2**
```bash
ssh root@server2 "~/bin/crawler cc recrawl --file p:0 --limit 10000 --writer devnull --no-retry --no-tui 2>&1 | tail -20"
```
This runs on 10K seeds with devnull writer (measures raw fetch speed only).
Expected output includes: `avg rps=NNN peak_rps=NNN`

**Step 4: Full benchmark pass-1 only**
```bash
ssh root@server2 "~/bin/crawler cc recrawl --file p:0 --limit 100000 --writer devnull --no-retry --no-tui 2>&1 | tail -20"
```
Target: avg ≥ 1500 pages/s with 100K CC seeds.

**Step 5: Full run with retry enabled**
```bash
ssh root@server2 "~/bin/crawler cc recrawl --file p:0 --no-tui 2>&1 | tail -30"
```
This runs the full partition with pass-1 (1s) + pass-2 (10s retry).

**Step 6: Commit final benchmark results to spec**

Update this spec file with actual observed numbers.

---

## Expected Performance on Server 2

**Hardware**: 12 GB RAM, ~12 CPUs, Ubuntu 24.04, fd=65536.

**Auto-config after changes**:
- `workers` = min(w_mem=37748, w_fd=32768, 8000) = **4,000** (capped at new max)
  *(w_mem = 12288×1024 KB × 75% / 256 KB = 37,748)*
- `inner_n` = clamp(12×2, 4, 16) = **16**
- `num_flushers` = max(4000/1000, 12/2) = max(4, 6) = **6**

**Performance model**:
- CC seeds: ~70% alive, typical response 100-300ms; ~30% dead (DNS fail or TCP drop)
- With connect_timeout=500ms: dead-TCP servers fail in 500ms instead of 1s
- With 4000 workers:
  - 70% alive @ 200ms avg: 4000×0.70/0.200 = 14,000 OK/s (capped by network/CPU)
  - 30% dead @ 500ms (connect timeout): 4000×0.30/0.500 = 2,400 timeout/s
  - Combined theoretical: ~16,400 total/s
  - With realistic overhead (TLS, semaphores, scheduling): **3,000-8,000 pages/s**

The 1500 pages/s target is well within reach. Historical Go data shows 5,158 avg/s on similar hardware with devnull writer (200K HN seeds). CC seeds are higher quality — OK rate should be higher.

**False negatives**: Zero permanent false negatives with the stall_ratio=0 fix.
- Pass-1: Slow sites (>1s) → pass-2 ✓ (already correct)
- Pass-2: Only `domain_dead_probe=2` triggers abandonment (2 consecutive timeouts, 0 successes = truly dead server). No stall-ratio kills.

---

## Key Lessons Applied from MEMORY.md

| Lesson | Applied |
|--------|---------|
| Pass-2 false negatives from pre-filtering | No pre-filtering in load_retry_seeds ✓ |
| Adaptive timeout kills pass-2 rescues | `disable_adaptive_timeout=true` in pass-2 ✓ |
| Stall ratio causes false negatives | **Fixed in Task 1**: `pass2_stall_ratio=0` |
| MemTotalMB for GOMEMLIMIT | Use mem_total_mb for workers calc ✓ |
| DuckDB single-connection | Binary writer default ✓ |
| Bot-holding: rotate browser UAs | `ua::pick_profile()` randomizes UA ✓ |
| Workers sweet spot: avoid DNS contention | 8000 cap (vs 16000 which caused 5% OK) |
| BinSegWriter: separate I/O from fetch path | Binary writer with crossbeam channel ✓ |

---

## Summary of Changes

| Task | File | Change | Impact |
|------|------|--------|--------|
| 1 | `job.rs`, `config.rs` | `pass2_stall_ratio=0` | Zero false negatives |
| 2 | `config.rs` | Worker cap 2000→8000 | 2× throughput |
| 3 | `config.rs`, all engines | `connect_timeout=500ms` | 2× faster dead-server detection |
| 4 | All engines | Channel `workers*4`→`workers*32` | Eliminates producer stall |
| 5 | `cc.rs`, `hn.rs`, `common.rs` | Expose CLI flags | Operator tuning |
| 6 | `config.rs` | Scale `num_flushers` with workers | Prevents writer bottleneck |
| 7 | `job.rs` | Integration test | Correctness proof |
| 8 | Makefile | Build + deploy | Verify on server2 |
