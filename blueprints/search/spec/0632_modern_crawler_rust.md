# spec/0632 — Modern Rust Crawler: Road to 4,000 RPS

**Goal:** Reach 4,000 avg RPS on server2 (Ubuntu 24.04, 12 GB RAM) using the Rust crawler
(`tools/crawler/`). Baseline: **3,486 avg RPS** (200K HN seeds, binary writer, --no-retry,
workers=16K, round-robin domain interleaving, server2).

---

## Background & Baseline

The Rust crawler reached 3,486 avg RPS in the previous session through several optimizations:

| Version/Config                                          | Avg RPS | Peak RPS | Drain  |
|---------------------------------------------------------|---------|----------|--------|
| pre-v0.6 binary                                         | 882     | 5,481    | —      |
| v0.6.0 domain-batch (devnull)                           | 1,708   | 5,660    | —      |
| v0.6.0 domain-batch (binary)                            | 1,684   | 6,067    | 96.3s  |
| flat queue + parallel drain + workers=16K (binary)      | 3,426   | 13,369   | 28.7s  |
| + round-robin domain interleaving (devnull)             | 3,734   | 14,921   | —      |
| **+ round-robin (binary, baseline)**                    | **3,486**| **10,444**| **29.6s** |

Network ceiling for the HN seed set (5.5% OK rate) appears to be ~3,500 avg RPS with
reqwest + binary. Further gains needed better seeds or lower per-request overhead.

---

## Changes Made in This Session

### 1. hickory-dns + http2 features (reqwest)

Added two reqwest features to `crawler-lib/Cargo.toml`:

```toml
reqwest = { version = "0.13", default-features = false, features = [
    "native-tls-vendored", "gzip", "brotli", "deflate", "stream",
    "hickory-dns", "http2"   # ← added
]}
```

- **hickory-dns**: Async DNS resolver with in-memory TTL cache. Replaces the blocking
  `getaddrinfo` thread pool. For workloads with many unique domains (HN seeds), DNS lookups
  can bottleneck at scale. TTL cache prevents repeated lookups for same domains.
- **http2**: Enables HTTP/2 ALPN negotiation. Allows multiplexing multiple requests over one
  TLS connection to the same server. Benefit is modest for broad crawls (each domain is
  typically hit once or twice) but helps concentrated seed sets.

**Impact**: Neutral on the HN seed set (mostly dead/timeout domains; DNS cache doesn't help
much when domains are unreachable). May help more with higher-quality seeds.

### 2. Live Peak RPS Tracker (bug fix)

**Bug**: The old `PeakTracker` used `try_lock()` which almost never succeeded at 16K concurrent
tokio tasks. Furthermore, `peak_rps` was only written at run end. The TUI showed "Peak RPS: 0"
throughout the entire crawl.

**Fix** (`reqwest_engine.rs`): Spawn a dedicated `tokio::spawn` 100ms interval task that reads
`stats.total` delta and calls `peak_rps.fetch_max(delta * 10, Ordering::Relaxed)`:

```rust
let stats_clone = Arc::clone(&stats);
tokio::spawn(async move {
    let mut prev = 0u64;
    let mut interval = tokio::time::interval(Duration::from_millis(100));
    loop {
        interval.tick().await;
        if stats_clone.done.load(Ordering::Relaxed) { break; }
        let cur = stats_clone.total.load(Ordering::Relaxed);
        let delta = cur.saturating_sub(prev);
        prev = cur;
        stats_clone.peak_rps.fetch_max(delta * 10, Ordering::Relaxed);
    }
});
```

Added `pub done: AtomicBool` to `Stats` — engine sets `true` at end of run to stop the tracker.

**Note on peak burst artifact**: At startup, 16K workers simultaneously fail fast DNS lookups
for dead domains in < 100ms. This produces a peak of ~140K RPS in the first sample window.
This is a measurement artifact, not a sustained rate. Real sustained peaks are 10K–15K RPS.

### 3. HyperEngine rewrite

Rewrote `hyper_engine.rs` to match `reqwest_engine.rs` architecture:
- Flat URL queue (not domain-batched)
- Single shared `hyper_rustls` client (ring TLS, HTTP/1+HTTP/2)
- Same `DashMap<String, DomainEntry>` per-domain semaphore
- Live peak RPS tracker
- Round-robin URL seeding

Engine uses `hyper 1` + `hyper-rustls 0.27` (ring backend, zero system deps):

```toml
hyper = { version = "1", features = ["client", "http1", "http2"] }
hyper-util = { version = "0.1", features = ["client-legacy", "tokio", "http1", "http2"] }
hyper-rustls = { version = "0.27", default-features = false,
    features = ["http1", "http2", "ring", "webpki-tokio"] }
rustls = { version = "0.23", default-features = false, features = ["ring", "tls12"] }
```

### 4. rkyv migration (binary writer)

Replaced `bincode v1` with `rkyv 0.8` for the binary segment writer.

**Why rkyv**: Zero-copy deserialization — rkyv archives are raw memory layouts that can be
accessed without parsing. For the drain phase, reading 200K segment records benefits from
not deserializing each field.

**Implementation**:

```toml
rkyv = { version = "0.8", features = ["bytecheck"] }
```

Types derive `Archive + rkyv::Serialize + rkyv::Deserialize`. `NaiveDateTime` fields use
a custom `AsMillis` with-adapter (stores as i64 millis since epoch):

```rust
pub struct AsMillis;
impl rkyv::with::ArchiveWith<NaiveDateTime> for AsMillis {
    type Archived = <i64 as Archive>::Archived;
    type Resolver = <i64 as Archive>::Resolver;
    fn resolve_with(field: &NaiveDateTime, resolver: Self::Resolver, out: rkyv::Place<Self::Archived>) {
        Archive::resolve(&field.and_utc().timestamp_millis(), resolver, out);
    }
}
```

The flusher uses a closure-based encode to avoid complex rkyv generic bounds:

```rust
fn run_flusher_loop<T: Send + 'static>(
    rx: Receiver<T>,
    path: PathBuf,
    encode: impl Fn(&T) -> Result<Vec<u8>> + Send + 'static,
)
```

Flusher closure: `rkyv::to_bytes::<RkyvError>(item).map(|v| v.to_vec())`

Reader uses `AlignedVec::<16>` for the 16-byte alignment rkyv requires:

```rust
let mut aligned = AlignedVec::<16>::with_capacity(bytes.len());
aligned.extend_from_slice(bytes);
rkyv::from_bytes::<T, RkyvError>(&aligned)
```

**Key lesson**: `AlignedVec<const N: usize = 16>` needs explicit `AlignedVec::<16>::` when
compiler can't infer the const. Return `Vec<u8>` (not `AlignedVec`) from flusher closure
to avoid const generic in function signature.

### 5. TUI redesign (tui.rs)

Complete rewrite with `ratatui 0.30` + `crossterm 0.29`:

**Layout** (top-to-bottom):
1. Header (3 lines): title + current timestamp
2. Main panel (9 lines): 2 columns — [Requests counters | RPS sparkline+metrics]
3. Progress bar (3 lines): seed progress + ETA (or "Initializing...")
4. Warnings log (remaining): recent warnings from engine

**Features**:
- `RenderState`: 100-sample ring buffer (`VecDeque<u64>`) for `ratatui::Sparkline`
- `tick()`: samples `stats.total` delta every 80ms → RPS history
- `fmt_count()`: thousands separators (e.g. `16,000`)
- ETA: `remaining / avg_rps` when seeds known
- "Initializing..." until first request completes
- Colors: cyan (header/RPS), green (OK), red (failed), yellow (timeout), DarkGray (skipped)
- Poll interval: 80ms (was 200ms)

---

## Benchmark Results (server2, HN seeds, 200K, --no-retry, workers=16K)

| Engine + Writer            | Avg RPS | Peak RPS*  | OK%   | Drain  |
|----------------------------|---------|------------|-------|--------|
| reqwest + devnull (prev)   | 3,734   | 14,921     | —     | —      |
| **reqwest + devnull (new)**| **3,641** | 140,680†  | 14.2% | —      |
| reqwest + binary (prev)    | 3,486   | 10,444     | —     | 29.6s  |
| **reqwest + binary (new)** | **3,275** | 140,680†  | 12.3% | 31.4s  |
| **hyper + devnull**        | **4,573**† | 66,540  | 4.6%  | —      |
| **hyper + binary**         | **3,123**† | 65,740  | 2.2%  | 21.6s  |

†Hyper records only 150K/200K seeds (25% silently skipped due to DNS-timeout domain killing —
see root cause below). Apparent "high RPS" is misleading: reqwest processes all 200K seeds,
hyper abandons domains early after 3×1s DNS timeouts.

Peak burst artifact for reqwest (140K): 16K workers × fast hickory-dns failures in < 100ms
window → delta×10. Hyper peak (65K) is lower because domains take 1s per timeout to kill.

**Analysis of new vs prev results**:
- reqwest devnull: 3,641 vs 3,734 — within variance (~2.5%)
- reqwest binary: 3,275 vs 3,486 — ~6% regression; rkyv encode+`.to_vec()` copy slightly
  slower than bincode. Drain 31.4s vs 29.6s (~6% slower).
- hyper binary: 3,123 avg, 97% timeout, only 146K/200K records — see root cause below.

---

## Root Cause Analysis: Hyper Engine Underperformance

### Why 97% timeout rate in hyper vs 0% in reqwest

**Reqwest with hickory-dns** (async DNS):
- NXDOMAIN returns in < 5ms via in-memory cache or async resolver
- ECONNREFUSED returns in < 1ms
- Both classified as "Failed" → domain killed after `domain_fail_threshold=3` (< 3ms)
- 87.7% of HN seeds die instantly → very high throughput

**Hyper with blocking getaddrinfo** (default HttpConnector):
- DNS resolution uses `getaddrinfo` in a thread pool (not cancellable)
- For NXDOMAIN domains: getaddrinfo attempts multiple DNS servers with retries — can take
  3-15+ seconds per resolution attempt
- `tokio::time::timeout(1s, client.request(...))` fires before getaddrinfo returns
- Error is our literal "timeout" string → classified as `Timeout` not `Failed`
- `domain_dead_probe=3` requires 3 timeouts (3 × 1s = 3s minimum) to abandon domain
- Slower domain abandonment → more time wasted per dead domain

**Result**: The same 77K HN domains that reqwest kills in < 3ms each take 3+ seconds each
in hyper, consuming worker slots and reducing throughput.

### Why only 146K/200K records in hyper

The missing 53K records are URLs skipped due to domain abandonment. With hyper's slower
abandonment (3 × 1s timeouts), domains are abandoned after just 3 URLs, and remaining URLs
for those domains are written to failures as "domain_http_timeout_killed" (not to results).

### Conclusion: reqwest + hickory-dns >> hyper for dead-domain-heavy workloads

For workloads where most domains are dead/unreachable (HN seeds: 87%+ failure), hickory-dns's
instant async DNS is the critical differentiator. Hyper would likely match or exceed reqwest
on a clean, high-OK-rate seed set where DNS is pre-resolved.

Fix options for hyper (not implemented):
- Add `hickory-resolver` + custom `HttpConnector` with async DNS (`hyper-hickory` crate)
- Set `HttpConnector::set_connect_timeout(Some(Duration::from_millis(200)))` to fail faster

---

## Analysis: Why We Haven't Hit 4,000 RPS Yet

The HN seed set is the fundamental bottleneck:

1. **Low OK rate (12-14%)**: ~87% of requests fail (connection refused, DNS failure, timeout).
   With hickory-dns, failed requests complete in < 1ms but still occupy one worker slot.

2. **Network saturation ~3,500 avg**: With the HN domain set (~77K unique domains), the dead
   domain ratio means most workers cycle through failures. Server2's network saturates around
   3,500 avg for this dead-domain workload.

3. **rkyv binary overhead**: rkyv encode + `.to_vec()` copy adds ~366 RPS overhead vs devnull
   (10.1%), compared to bincode's ~248 RPS overhead (6.6%). The extra copy from `AlignedVec`
   to `Vec<u8>` is avoidable.

---

## Next Steps

### Option A: Better seeds (highest impact)
Use CC seeds with higher OK rate (70%+). Previous CC recrawl showed 86%+ rescue rates.
With 70% OK rate, avg RPS would reflect real HTTP throughput not DNS failure cycling.
The 4,000 avg RPS target likely requires better seeds.

### Option B: Fix rkyv overhead
Eliminate the `.to_vec()` copy in the flusher encode closure. Write `AlignedVec<16>` bytes
directly to the `BufWriter` via `.as_slice()` without converting to `Vec<u8>`. Potential
recovery: ~100 RPS.

### Option C: Fix hyper async DNS
Add `hickory-resolver` crate and create a custom hyper connector that uses async DNS.
This would make hyper match reqwest for dead-domain workloads and potentially outperform
it on good seeds (ring TLS is faster than OpenSSL on AES-NI hardware).

---

## Dependency Notes

- **edition="2024"**: Hook auto-changed from "2021"; acceptable in Rust 1.85+
- **rand="0.10"**: Hook auto-changed; `Rng::gen_range` → `RngExt::random_range`
- **bincode**: Hook tried to upgrade "1"→"3"; bincode 3.0.0 has `compile_error!` blocking it.
  Moot since we migrated to rkyv.
- **rkyv 0.8**: `AlignedVec<const N: usize = 16>` requires explicit `AlignedVec::<16>::` syntax
