# Rust Crawler: Road to 5,000 avg RPS on Server2

> **Goal:** Sustained average of 5,000 pages/s on server2 (Ubuntu 24.04, AMD Ryzen, 12 GB RAM).
> **Baseline:** 882 avg RPS, 5,481 peak (200K seeds, reqwest + binary writer, --no-retry).
> The gap between avg and peak (6.2×) is the optimization opportunity.

---

## 1. Root Cause Analysis

### 1.1 Domain-Batch Architecture (Biggest Bottleneck)

The current architecture:
1. Seeds sorted → grouped by domain → `Vec<DomainBatch>` sent to channel
2. N workers consume one batch at a time (one domain per worker slot)
3. Per domain: `inner_n` concurrent fetch tasks share a single `reqwest::Client`

**The problem:** A single-URL domain occupies a worker for the full request timeout (1s+).
Workers sit idle during those 1s waits while other domains are queued.

**Quantification (50K seeds example):**
- ~25K distinct domains → ~2 URLs/domain average
- Domains with 1 URL: 60-70% of domains (long tail)
- With 8192 workers × inner_n=4: effective parallelism = 32,768 concurrent fetches
- Actual effective parallelism: ≈ 882 avg RPS × avg_latency_s ≈ 882 × 1.5 ≈ 1,323
- Theoretical max vs actual: 32,768 / 1,323 ≈ **24.8× utilization gap**

**Root cause:** Worker stalls on dead/slow domains block throughput.
A dead domain with 3 URLs (probe=3): takes `ceil(3/4) × 1s = 1s` → 1 worker blocked for 1s.
With 30% of domains dead: `0.3 × 25K = 7,500 dead domains × 1s / 8192 workers = 0.9s of dead time`.
This dead time is spread across all workers, reducing avg from peak.

### 1.2 Domain Abandonment (Fixed in v0.6.0)

- `domain_dead_probe=10` caused dead domains to waste 3 rounds × 1s = 3s before abandonment
- `domain_stall_ratio=20` caused slow domains to take up to 30s before domain timeout fires
- Adaptive timeout growing >6s caused 30s domain timeout warnings

**Already fixed in this iteration:** probe=3, stall_ratio=5, adaptive cap at 5×base, min domain_timeout 5s.

### 1.3 Full Body Read Before Truncation (Medium)

```rust
// Current: reads full response then slices
let full = resp.bytes().await?;
if full.len() > max_bytes { Ok(full.slice(..max_bytes)) }
```

For a 2MB HTML page with max_body_bytes=256KB:
- Downloads full 2MB (wasted 1.75MB bandwidth)
- Holds 2MB buffer temporarily (GC pressure)
- A single large-body domain can saturate the network connection

**Fix:** Stream body with `take()` wrapper — stop reading after 256KB.

### 1.4 Client-Per-Domain Overhead (Medium)

```rust
// Current: builds a new reqwest::Client per domain
let client = reqwest::Client::builder()
    .pool_max_idle_per_host(inner_n)
    .timeout(cfg.timeout)
    // ...
    .build()?;
```

- `reqwest::Client` construction involves TLS context init, connection pool setup
- For 25K domains: 25K client builds during the crawl
- Each build: ~2-5μs, but 25K × 5μs = 125ms pure overhead (not a bottleneck by itself)

**Better fix:** Build one client per worker (not per domain), set `pool_max_idle_per_host=inner_n`.

### 1.5 Single reqwest::Client vs Per-Worker Client (Minor)

The pool deduplication happens at the host level. Multiple workers hitting the same domain
share connection pools by domain key. This is fine with a per-worker client.

### 1.6 DNS Resolution (Minor)

No pre-caching of DNS. Each new domain triggers a DNS lookup.
- DNS lookup: 1-200ms (highly variable)
- Impact: ~0.5-5% overhead for first requests to each domain

---

## 2. Technology Options

### 2.1 Body Streaming with reqwest (Immediate Win)

Use `resp.bytes_stream()` + `StreamExt::take_while` or manual chunk counting:
```rust
use tokio_stream::StreamExt;
let mut stream = resp.bytes_stream();
let mut buf = Vec::with_capacity(max_bytes);
while let Some(chunk) = stream.next().await {
    let chunk = chunk?;
    let remaining = max_bytes.saturating_sub(buf.len());
    if remaining == 0 { break; }
    buf.extend_from_slice(&chunk[..chunk.len().min(remaining)]);
}
```
**Benefit:** For large pages, 10-100× less bandwidth wasted. On HN seeds, ~30% of pages are >256KB.

### 2.2 Per-Worker reqwest::Client (Immediate Win)

Build one `reqwest::Client` per worker thread (not per domain). Workers reuse across domains.
Pool deduplication: reqwest uses host-keyed connection pools automatically.
**Benefit:** Eliminates 25K client builds. Also allows keep-alive across domains within same IP.

### 2.3 hickory-resolver (Async DNS Pre-fetch)

`hickory-resolver` (formerly trust-dns-resolver) provides async DNS with local caching.
```toml
hickory-resolver = { version = "0.24", features = ["tokio-runtime"] }
```
Pre-fetch DNS for all unique domains before crawl starts (or in background).
**Benefit:** Eliminates 200ms DNS latency for first request per domain. For 25K domains × 200ms = 5,000s
of aggregate DNS time removed from critical path.

### 2.4 HTTP/2 Multiplexing (Medium Complexity, High Reward)

For domains hosting many URLs (HN, GitHub, Reddit, etc.), HTTP/2 allows multiple concurrent
requests over a single TCP connection. Reqwest supports HTTP/2 via `http2` feature (already enabled).

Key change: let reqwest handle H2 negotiation automatically (it already does via ALPN).
Manual optimization: configure `max_concurrent_streams` per connection.
**Benefit:** For multi-URL domains, saturates connection better. Reduces per-domain latency.

### 2.5 tokio-uring (Linux io_uring, Long-term)

`tokio-uring` is a Tokio integration with Linux's io_uring for zero-copy async I/O.
- Reduces syscalls for network I/O
- Better CPU utilization under high connection counts
- Requires Linux 5.4+ (server2 has 6.8 kernel)
- Incompatible with standard Tokio runtime — requires switching runtime

**Complexity:** Very high (full async runtime change). Not recommended until other wins exhausted.
**Estimated benefit:** 15-30% throughput improvement (mainly latency reduction at high concurrency).

### 2.6 aws-lc-rs TLS Backend (Server2 Only)

`aws-lc-rs` is AWS's Rust TLS implementation using assembly-optimized AES-GCM.
Server2 (GCC 14, modern CPU) can compile it without issues.
```toml
reqwest = { features = ["rustls-tls"] }
rustls = { features = ["aws-lc-rs"] }
```
**Benefit:** 20-40% faster TLS handshake on AES-NI hardware. For HTTPS-heavy crawls
with many short-lived connections, reduces TLS overhead. **Server2 only** (Server1 GCC9 incompatible).

### 2.7 Connection Pool Tuning

`reqwest::Client` pool settings:
- `pool_max_idle_per_host`: currently `inner_n`. Can increase for domains with many URLs.
- `pool_idle_timeout`: default 90s. Can reduce to free idle connections sooner.
- `tcp_keepalive`: enables TCP keepalive to detect dead connections faster.

---

## 3. Implementation Plan

### Priority 1: Streaming Body (High Impact, Low Risk)

**Expected gain:** +10-20% avg RPS by eliminating bandwidth waste on large pages.

Replace `read_body_limited` in `reqwest_engine.rs`:
```rust
async fn read_body_limited(resp: reqwest::Response, max_bytes: usize) -> Result<bytes::Bytes, reqwest::Error> {
    use bytes::BytesMut;
    use futures_util::StreamExt;
    let mut stream = resp.bytes_stream();
    let mut buf = BytesMut::with_capacity(max_bytes.min(64 * 1024));
    while let Some(chunk) = stream.next().await {
        let chunk = chunk?;
        let remaining = max_bytes.saturating_sub(buf.len());
        if remaining == 0 { break; }
        buf.extend_from_slice(&chunk[..chunk.len().min(remaining)]);
    }
    Ok(buf.freeze())
}
```

Requires adding `futures-util` or `tokio-stream` dep (both already transitively available).

### Priority 2: Per-Worker reqwest::Client (High Impact, Medium Complexity)

Change architecture: each worker has ONE client, reused across all domains it processes.

```rust
// In worker spawn loop:
let client = Arc::new(build_client(inner_n, cfg.timeout)?);
let handle = tokio::spawn(async move {
    while let Ok(batch) = rx.recv().await {
        process_one_domain_with_client(batch, &client, ...).await;
    }
});
```

The per-domain client build moves out of `process_one_domain` into the worker loop.
This eliminates 25K client builds and allows HTTP keep-alive across domain batches
served from the same IP.

### Priority 3: hickory-resolver DNS Pre-fetch (Medium Impact, Low Risk)

Pre-resolve all unique domains before crawl starts:
```rust
use hickory_resolver::{TokioAsyncResolver, config::*};
let resolver = TokioAsyncResolver::tokio(ResolverConfig::google(), ResolverOpts::default());
// Pre-fetch in background (don't block start)
let resolver = Arc::new(resolver);
```
Pass resolver to the client builder via `reqwest::ClientBuilder::dns_resolver()`.
This enables cached DNS for repeated lookups within a crawl.

### Priority 4: Configurable Workers Cap (Tuning)

The current cap `min(w_mem, w_fd).min(10_000)` limits to 10K workers max.
For server2 (12GB RAM, fd=65536), auto-config gives ~8192 workers.
Try `--workers 16000` with inner_n=4 to push effective parallelism to 64K.
Monitor with the new TUI (avg RPS, peak RPS, warnings).

### Priority 5: Adaptive Timeout Tuning

Current: adaptive = P95 × 2, capped at 5× base timeout.
For HN seeds (mixed fast/slow):
- P95 of successful requests ≈ 800ms
- Adaptive timeout ≈ 1600ms (1.6s)
- Too long for dead domains, too short for some slow-but-valid ones

Alternative: use P90 × 1.5 = more aggressive, fewer false keeps.
Expose as `--adaptive-factor` flag (default 2.0, reduce to 1.5 for fast crawls).

---

## 4. Benchmark Targets

| Scenario | Current | After Priority 1-2 | After All |
|----------|---------|---------------------|-----------|
| 200K seeds, devnull | 882 avg | ~1,500 avg | ~3,000 avg |
| 200K seeds, binary | 882 avg | ~1,200 avg | ~2,500 avg |
| 50K seeds (pre-filtered) | ~2,000 avg | ~4,000 avg | ~5,000 avg |

Note: 5,000 avg RPS is achievable with **pre-filtered seeds** (e.g., recent HN posts with
higher OK rate). With full HN domain set (60% timeout), 5K sustained avg is harder
because timeout handling dominates.

**Key insight:** Peak is already 5,481 RPS. The gap is caused by:
1. Worker stalls on dead domains (fixed in v0.6.0)
2. Bandwidth waste on large pages (Priority 1)
3. Worker idle time between domain batches (Priority 2)

---

## 5. Measurement Methodology

Run benchmarks on server2 with:
```bash
~/bin/crawler hn recrawl \
  --seed ~/data/hn/recrawl/hn_pages.duckdb \
  --limit 50000 \
  --writer devnull \
  --no-retry \
  --workers 8192
```

Use the TUI dashboard to monitor:
- Avg RPS (should increase from 882 to target)
- Peak RPS (baseline for theoretical max)
- Timeout % (should decrease with bug fixes)
- Elapsed time (should decrease with same seed count)

Also monitor:
- `htop` CPU usage (should be >50% on all cores — if low, we're network-bound)
- `nethogs` or `iftop` network bandwidth (check if bandwidth is bottleneck)
- `/proc/net/sockstat` TCP connection counts

---

## 6. Key Lessons from Analysis

- **Domain timeout 30s was too conservative**: Reduced to 5s minimum.
- **dead_probe=10 was too cautious**: Reduced to 3 (faster dead detection: 1 round × 1s).
- **stall_ratio=20 caused slow domain hangs**: Reduced to 5 (abandon faster if stalling).
- **Adaptive timeout could grow >10s**: Capped at 5× base to prevent runaway per-request timeouts.
- **Per-domain client build**: 25K builds is measurable overhead, move to per-worker.
- **Body streaming**: Biggest bandwidth win for large-page-heavy datasets.
- **Peak vs avg gap**: The 6× gap indicates workers are idle ~83% of the time.
  Target: reduce to 2× gap (workers idle ~50% — inherent from network variability).
