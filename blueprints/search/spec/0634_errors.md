# Crawler Error Investigation & Classification

> Living document — updated as new error patterns are discovered.

## Error Classification Architecture

### Previous approach (broken): String matching on `e.to_string()`

reqwest's `Error::to_string()` only shows the **outer wrapper**, not the inner cause:
```
"error sending request for url (http://example.com/)"
```

The actual inner error (DNS failure, TLS error, timeout, connection refused) is only
available via the `source()` chain. String-matching on the outer message misclassifies
almost everything as "Connection" because `"error sending request"` matches our
connection pattern.

**Result**: 99,845 "conn" errors in 200K benchmark — almost all misclassified.

### Current approach: reqwest typed methods + error chain walking

```rust
fn classify_reqwest_error(e: &reqwest::Error) -> ErrorCategory {
    if e.is_timeout()  → Timeout
    if e.is_builder()  → InvalidUrl
    if e.is_connect()  → walk chain → DNS / TLS / Connection
    if e.is_request()  → walk chain → DNS / TLS / Timeout / Connection
    else               → Other
}
```

`error_chain_string()` walks `source()` to build the full error message for logging.

---

## Error Categories

### 1. InvalidUrl (builder error)
**reqwest method**: `is_builder() == true`
**Cause**: URL is so malformed that reqwest can't even construct an HTTP request.
**fetch_time_ms**: 0 (no network call made)

**Common patterns in HN seed data**:
- Template syntax leaked: `{page.url|replace`, `{{...}}`
- Random garbage: `$$t%$.issomerandom!@`, `0963347006` (phone number)
- URL-encoded junk as domain: `%22+http`, `%22http`
- HTML entities as domain: `&lt;ahref=&quot;bugreport.cgi`
- Article slugs as domain: `2021-11-10-world-starting-to-notice-...`
- Leading dots: `.com`, `.nytimes.com`
- Wildcards: `*.travian.*`
- Facebook numeric IDs: `100009676428753`

**Scale**: ~35/10K seeds (0.35%), ~700/200K estimated

**Fix**: These are seed data quality issues. Should be pre-filtered during seed import,
but the crawler handles them gracefully (instant fail, no network overhead).

---

### 2. DNS Errors
**reqwest method**: `is_connect() == true` with `source()` chain containing DNS keywords
**Cause**: Domain doesn't resolve — NXDOMAIN, no A/AAAA records, resolver failure.
**fetch_time_ms**: 1-50ms typically (hickory-dns async resolver, fast NXDOMAIN)

**Inner error patterns** (from `source()` chain):
- `dns error: failed to lookup address information`
- `dns error: no record found for name: example.com type: A class: IN`
- `resolve error: no addresses returned`
- `nxdomain`

**Scale**: ~942/10K seeds (9.4%). This is the expected dead-domain rate for HN URLs.

**Note**: With hickory-dns (async DNS), NXDOMAIN returns in <5ms. Without it (system DNS
via getaddrinfo), NXDOMAIN takes 3-15s and gets misclassified as timeout.

---

### 3. Connection Errors
**reqwest method**: `is_connect() == true` (after excluding DNS/TLS from chain)
**Cause**: DNS resolved but TCP connection failed — server is down, port closed, firewall.
**fetch_time_ms**: 50-250ms (TCP SYN timeout, RST response, or ICMP unreachable)

**Inner error patterns**:
- `tcp connect error: Connection refused (os error 111)`
- `tcp connect error: Connection reset by peer`
- `tcp connect error: Network is unreachable (os error 101)`
- `tcp connect error: No route to host (os error 113)`
- `connection closed before message completed`

**Scale**: ~75/10K seeds (0.75%). These are genuinely dead servers.

**Subcategories**:
- **Connection refused** (port closed): server exists but not listening on 80/443
- **Connection reset**: server actively rejects the connection
- **Network unreachable**: routing failure
- **Localhost URLs**: `127.0.0.1:8000`, `0.0.0.0:8080` — HN submissions with dev URLs

---

### 4. TLS Errors
**reqwest method**: `is_connect() == true` with `source()` chain containing TLS keywords
**Cause**: TCP connected but TLS handshake failed — expired cert, wrong hostname, unsupported protocol.
**fetch_time_ms**: 100-500ms (TCP handshake + partial TLS handshake)

**Inner error patterns**:
- `ssl error: certificate verify failed`
- `tls error: handshake failure`
- `tls error: alert received: handshake_failure`
- `ssl error: unsupported protocol`

**Scale**: ~17/10K seeds (0.17%)

**Note**: We use `danger_accept_invalid_certs(true)`, so most TLS errors are protocol-level
failures (ancient TLS 1.0 servers, broken SNI, etc.), not certificate validation.

---

### 5. Timeout
**reqwest method**: `is_timeout() == true`
**Cause**: Server didn't respond within the configured timeout (default 1s).
**fetch_time_ms**: ~= timeout value (1000ms for pass 1, 15000ms for pass 2)

**Scale**: ~3,674/10K seeds (36.7%). This is the largest error category.

**Root causes**:
- **Dead servers that accept TCP but don't respond** (SYN-ACK then silence)
- **Bot-holding**: Some servers detect crawler User-Agent and hold the connection open
  (respond in <200ms for browser UAs, >5s for crawler UAs)
- **Overloaded servers**: legitimate sites that are too slow
- **Firewall drop**: SYN gets through but response is dropped (vs refused)

**Mitigation**: Pass 2 retries timeouts with 15s timeout — rescues ~86% of timeout URLs.

---

### 6. Other
**Catch-all**: Anything not matching the above categories.
**reqwest methods**: `is_body()`, `is_decode()`, `is_redirect()` (loop), or unknown.

**Scale**: ~0/10K seeds (0.0%) — currently perfect classification.

---

## Historical Misclassification Bug

### The 99K "conn" errors (pre-fix)

**Before** (string-based `e.to_string()` classification):
```
Failed: 101,013 (50.5%)
  dns: 171  conn: 99,845  tls: 686  other: 311
```

**After** (typed `reqwest::Error` method classification):
```
Timeout: 3,674 (36.7%)     ← were hidden in "conn"
Failed:  1,069 (10.7%)
  inv: 35  dns: 942  conn: 75  tls: 17  other: 0
```

**What happened**: `reqwest::Error::to_string()` produces:
```
"error sending request for url (http://example.com/)"
```
Our string matcher had `"error sending request"` → Connection. But the inner `source()`
chain could be:
- `operation timed out` → should be Timeout
- `dns error: no record found` → should be DNS
- `ssl error: certificate verify failed` → should be TLS

The fix: use `e.is_timeout()`, `e.is_connect()`, `e.is_builder()`, `e.is_request()`
for primary classification, then walk `source()` chain for sub-classification within
`is_connect()` and `is_request()`.

---

## Worker Count Impact on Error Rate

| Workers | OK Rate | Timeout | Connection | Notes |
|---------|---------|---------|------------|-------|
| 200     | 69.4%   | low     | moderate   | Sweet spot for reliability |
| 2,000   | 49.5%   | 36.7%   | 0.75%      | Current default, good balance |
| 16,000  | 5.1%    | ~90%    | ~5%        | OS DNS/TCP stack overwhelmed |

**Root cause at 16K workers**: All 16K tokio tasks fire DNS lookups + TCP handshakes
simultaneously, overwhelming the OS network stack. DNS resolver gets backlogged,
TCP SYN queue overflows, ephemeral ports exhausted.

**Fix**: Capped `auto_config` workers at 2,000.

---

## Error Chain Examples

### Timeout (was misclassified as Connection)
```
error sending request for url (http://example.com/)
  └─ error trying to connect: tcp connect error
       └─ operation timed out
```
`e.is_timeout() == true` catches this correctly.

### DNS (was misclassified as Connection)
```
error sending request for url (http://dead-domain.com/)
  └─ error trying to connect: dns error
       └─ failed to lookup address information: Name or service not known
```
`e.is_connect() == true`, chain contains "dns" → DNS.

### True Connection Error
```
error sending request for url (http://down-server.com/)
  └─ error trying to connect: tcp connect error
       └─ Connection refused (os error 111)
```
`e.is_connect() == true`, chain has no DNS/TLS keywords → Connection.

### Invalid URL
```
builder error for url (http://$$t%$.issomerandom!@)
```
`e.is_builder() == true` → InvalidUrl.

---

## Seed Data Quality (HN)

Total seeds: 1,539,560

| Issue | Count | % |
|-------|-------|---|
| No dot in domain | 282 | 0.018% |
| Leading dot | 5 | 0.0003% |
| Leading dash | 3 | 0.0002% |
| Space in domain | 93 | 0.006% |
| Wildcard in domain | 1 | 0.0001% |
| **Total garbage** | **~384** | **0.025%** |

Most seed URLs are syntactically valid but point to dead/unreachable servers.
The actual error breakdown at 10K scale: 52.6% OK, 36.7% Timeout, 9.4% DNS dead,
0.75% Connection dead, 0.35% Invalid URL, 0.17% TLS error.
