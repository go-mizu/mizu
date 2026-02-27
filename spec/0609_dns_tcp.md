# 0609: DNS + TCP Probe Diagnosis and Fix

## Summary

Investigated why TCP unreachable rate was 63–74% across all CC partitions. Root cause: probe concurrency (2000 workers) saturated the local OS/network stack, causing ~20% false failure rate. Fixed with adaptive multi-phase probe. TCP unreachable reduced from 74% → 4.3% (2-pass) → 1.1% (3-pass adaptive).

---

## 1. Observed Problem

CC recrawl partitions showed extremely high TCP unreachable rates:

| Partition | Domains     | TCP reachable | TCP unreachable |
|-----------|-------------|---------------|-----------------|
| p:0 (.ru) | ~37,500     | ~36.4%        | **63.6%**       |
| p:50      | ~37,500     | ~25.7%        | **74.3%**       |
| p:200 (.com) | ~131,428 | ~25.7%        | **74.3%**       |

Manual verification confirmed: 14/14 sampled tcp_unreachable domains with IPv4 addresses were actually OPEN on port 443 when tested manually via `nc`. So the probe was producing massive false negatives.

---

## 2. Root Cause Analysis

### Bug 1 (PRIMARY): Concurrency saturation causes false TCP failures

The `directFeed` function ran TCP probes with 2000 workers and 2s timeout. At this concurrency, the local OS/network stack (macOS ARM64) becomes saturated:

```
concurrency=  50: 0/900 fail    (0.0%)   ← safe
concurrency= 100: 0/900 fail    (0.0%)   ← safe
concurrency= 200: 0/900 fail    (0.0%)   ← safe
concurrency= 300: 0/900 fail    (0.0%)   ← safe
concurrency= 500: 4/900 fail    (0.4%)   ← acceptable
concurrency=1000: 38/900 fail   (4.2%)   ← problematic
concurrency=2000: 179/900 fail  (19.9%)  ← used in prod!
```

With 2000 concurrent TCP dials to known-open IPs, ~20% of connections fail due to local TCP state exhaustion — not because servers are unreachable.

**Evidence**:
- DuckDB query: 8,711 of 10,932 tcp_unreachable domains in p:0 had valid IPv4 addresses cached
- Manual nc test: 8/8 .ru + 6/6 .com sampled tcp_unreachable IPv4 domains were OPEN
- Benchmark: Go program with 900 probes to 9 well-known IPs confirmed failure rates above

### Bug 2: IPv6 addresses in DNS cache

The DNS resolver uses `LookupNetIP("ip4", ...)` for fastdns and stdlib `LookupHost` (IPv4+IPv6) as fallback. First-success wins. When stdlib wins, the cache may store IPv6-first or IPv6-only addresses.

`tcpProbeURL` (pre-fix) used `ips[0]` unconditionally. On a machine without IPv6 routing, dialing an IPv6 address fails instantly, causing false unreachable classification.

- p:200 analysis: 0.2% of domains had IPv6-only cached IPs (304 domains)
- Fix: added IPv4-preference loop in `tcpProbeHostPort`

### Bug 3: Pass 2 threshold too restrictive (pre-fix)

Original 2-pass probe retried only domains with ≥100 URLs. Most false-negative domains had <100 URLs and were never retried.

---

## 3. Fix 1: Reduce Probe Concurrency (2-pass → 2-pass 500 workers)

**Change**: Reduce Pass 1 from 2000 → 500 workers, Pass 2 from 500 → 500 workers with 2s timeout, remove ≥100 URL threshold.

**Result** (p:200, 131,428 domains):
- Probe: 125,738 reachable (95.7%), 5,663 unreachable (4.3%)
- Improvement: 74% → 4.3% tcp_unreachable

The remaining 5,663 fall into two categories:
1. **Port mismatch**: 443 closed but 80 open (HTTP-only sites) — false unreachable
2. **Truly dead**: SYN black hole on all ports — genuine unreachable

---

## 4. Fix 2: Adaptive Multi-Phase Probe (3-pass)

Manual verification of top tcp_unreachable domains:

| Domain         | IP            | Port 443 | Port 80  | Reason              |
|----------------|---------------|----------|----------|---------------------|
| trustburn.com  | 144.168.80.166| CLOSED   | **OPEN** | HTTP-only site      |
| tripatini.com  | 38.173.123.8  | CLOSED   | **OPEN** | HTTP-only site      |
| triego.com     | 168.76.252.72 | TIMEOUT  | TIMEOUT  | Truly dead          |
| truedungeon.com| 34.160.37.117 | TIMEOUT  | TIMEOUT  | Truly dead          |

`trustburn.com` has 20,787 CC URLs — all were being skipped due to false unreachable.

### New Probe Design

Replaced 2-pass with adaptive 3-pass:

```
Pass 1: 500 workers, 3s timeout, URL's actual port
  → catches 99.6% of alive domains

Pass 2: 500 workers, 1s timeout, alternate port (80↔443)
  → recovers HTTP-only sites (port 443 refused, port 80 open)
  → recovers 0.4% saturation false-negatives from Pass 1

Pass 3: 200 workers, 8s timeout, both ports
  → catches high-latency hosts (>3s RTT)
  → adaptive: repeats until newAlive == 0 (converged to truly-dead set)

Result: only genuine SYN black holes remain as "unreachable"
```

### New Function: `tcpProbeHostPort`

Added `probeOutcome` type distinguishing refused vs timeout:

```go
type probeOutcome int8
const (
    probeOK      probeOutcome = iota // TCP connect succeeded
    probeRefused                     // RST — port is definitively closed
    probeTimeout                     // timed out — host filtered/slow
    probeError                       // network error
)
```

Also added `alternatePort(port string) string` returning 80↔443 for standard ports.

### Results (p:200, 131,428 domains)

| Pass                        | Workers | Timeout | Unreachable after |
|-----------------------------|---------|---------|-------------------|
| Baseline (2000 workers)     | 2000    | 2s      | 74.3% = ~97,700   |
| Fix 1 (500 workers)         | 500     | 3s      | 4.3% = 5,663      |
| Fix 2 (adaptive 3-pass)     | 500/500/200 | 3s/1s/8s | **1.1% = 1,471** |

**Adaptive probe recovered 4,192 additional alive domains** (vs 2-pass). These were HTTP-only sites where port 443 was closed but port 80 was open — probe 2 recovered them via alternate port.

Probe timing (131K domains):
- Pass 1: ~3 min (500 workers × avg 300ms)
- Pass 2: ~15s (500 workers × ~5,663 failures × 1s avg)
- Pass 3: ~5 min (200 workers × ~3,500 truly-dead × 16s avg)
- Total: ~8 minutes

---

## 5. Current Minimum ("Floor")

After 3-pass adaptive probe, **1,471 domains** remain tcp_unreachable (1.12% of 131K). These are genuinely unreachable:
- SYN black holes on all ports (both 80 and 443 timeout after 8s)
- IPv6-only domains with no IPv4 routing from this machine
- Firewalled domains (geo-blocking, IP blacklisting)

**Getting below 100 unreachable** would require:
1. Different network vantage point (CDN proxy, multi-region)
2. DNS re-resolution (some dead IPs might have new IPs now)
3. Waiting for blacklists to expire

For a single-machine recrawler, ~1,100–1,500 truly dead domains per 131K is the expected floor for diverse CC .com data.

---

## 6. IPv4 Preference Fix

In `tcpProbeHostPort` (renamed from `tcpProbeURL`), added explicit IPv4 preference:

```go
// Prefer IPv4: DNS cache may contain mixed IPv4+IPv6.
// IPv6 fails instantly on machines without IPv6 routing.
addr := host
if len(ips) > 0 {
    addr = ips[0]
    for _, ip := range ips {
        if !strings.Contains(ip, ":") { // IPv4 has no colons
            addr = ip
            break
        }
    }
}
```

Without this, IPv6-only cached IPs (0.2% of domains) would fail instantly and be classified as tcp_unreachable.

---

## 7. Failed Domain Breakdown (p:200 after fix)

From `failed.duckdb` after adaptive probe + partial HTTP crawl:

| Reason             | Count  | Explanation                           |
|--------------------|--------|---------------------------------------|
| http_timeout_killed| 13,057 | HTTP phase: domain killed (>N timeouts)|
| tcp_unreachable    |  1,471 | TCP probe: no port open (truly dead)  |
| http_refused       |  1,110 | HTTP phase: connection refused        |
| dns_timeout        |     15 | DNS resolution timed out              |
| dns_nxdomain       |     14 | Domain does not exist                 |
| http_dns_error     |      9 | DNS error during HTTP fetch           |

Note: `http_timeout_killed` (13K) is the HTTP crawl phase killing domains that had too many HTTP-level timeouts. This is separate from the TCP probe. These domains ARE TCP-alive (passed the probe) but have slow/non-responsive HTTP servers.

---

## 8. Code Changes

### `pkg/recrawler/recrawler.go`

1. **New `probeOutcome` type** (lines ~422–430): classifies TCP failure as refused/timeout/error

2. **New `tcpProbeHostPort` function** (replaces `tcpProbeURL`): takes explicit host/port, returns outcome, prefers IPv4

3. **New `alternatePort` function**: returns 80↔443 for HTTP/HTTPS port pairs

4. **Replaced 2-pass with adaptive 3-pass** in `directFeed`'s `!TwoPass` branch:
   - Pass 1: 500 workers, 3s, original port → `outcomes[]` array
   - Pass 2: 500 workers, 1s, alternate port + saturation retry
   - Pass 3: 200 workers, 8s, both ports, adaptive convergence

---

## 9. Key Lessons

1. **Local saturation ≠ server dead**: At 500+ concurrent TCP dials, the local OS creates false failures. 500 workers is the empirical limit (0.4% false fail, recovered by Pass 2).

2. **HTTP-only sites fail HTTPS probe**: ~3.2% of .com domains in CC 2026 are HTTP-only. Port 443 refused but port 80 open. Alternate port probe recovers them.

3. **Two classification axes**: TCP timeout (truly dead or geo-blocked) vs TCP refused (port closed, try alternate). Treating both as "dead" is wrong.

4. **Probe time vs HTTP crawl time**: For 131K domains, probe takes ~8 min. HTTP crawl of 7.2M URLs takes 20–40 min. Probe is small fraction of total time.

5. **IPv6 in DNS cache causes silent failures**: stdlib `LookupHost` can return IPv6-first when fastdns times out. Always prefer IPv4 when it's available.
