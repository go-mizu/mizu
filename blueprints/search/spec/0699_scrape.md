# 0699 — Scrape Dashboard: Caching & Live Results

## Problems

### 1. Scrape list page slow to load
`ListDomains()` opens each DuckDB shard file sequentially for every domain,
runs aggregate queries, then closes. With N domains × M shards, this means
N×M DuckDB opens per page load — each taking 50-200ms.

### 2. Domain page only shows results after crawl finishes
The pages table queries DuckDB shards. During an active crawl, dcrawler batches
writes via its flusher — rows aren't visible until the shard is flushed and the
WAL checkpointed. The frontend auto-refreshes status (2s) but never refreshes
the pages table while a job is running.

### 3. HTTP/2 protocol error
`protocol error: received DATA after END_STREAM` — Go's HTTP/2 implementation
conflicts with long-lived SSE/WebSocket connections when WriteTimeout=0.

## Solution

### A. In-memory stats cache for ListDomains

Add a `statsCache` to `Store` with a 60s TTL. On first call (or after TTL),
scan domains and query shards in the background. Subsequent calls return cached
results instantly.

**Cache invalidation**: When a scrape job completes or is stopped, the executor
calls `store.InvalidateCache(domain)` to force a refresh on next request.

```go
type cachedStats struct {
    resp      *ListResponse
    fetchedAt time.Time
}

func (s *Store) ListDomains() (*ListResponse, error) {
    s.mu.RLock()
    if s.cache != nil && time.Since(s.cache.fetchedAt) < s.cacheTTL {
        r := s.cache.resp
        s.mu.RUnlock()
        return r, nil
    }
    s.mu.RUnlock()
    // ... query shards, update cache ...
}
```

### B. Auto-refresh pages table during active crawl

Frontend change: when `active_job` is present and running, set a 3s timer to
also reload the pages table (not just the status pane). This shows pages as
they're flushed to DuckDB during crawl.

```js
if (active && active.status === 'running') {
    setTimeout(() => {
        loadScrapeDomainStatus(domain);
        loadScrapePages(domain);  // <-- NEW: refresh pages too
    }, 3000);
}
```

### C. Disable HTTP/2 to fix protocol error

Go's HTTP/2 has known issues with hijacked connections (WebSocket) and
long-lived responses (SSE). Disable HTTP/2 by setting `TLSNextProto` to an
empty map on the `http.Server`. The dashboard runs on localhost — HTTP/2
provides no benefit.

```go
srv := &http.Server{
    TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
    // ...
}
```

### D. Parallel shard queries in quickStats

Instead of sequential shard opens, use a goroutine-per-shard pattern with
`sync.WaitGroup`. With 4-8 shards typical, this cuts ListDomains latency
from N×200ms to max(200ms).

## Files Changed

| File | Change |
|------|--------|
| `pipeline/scrape/store.go` | Add stats cache + parallel quickStats + InvalidateCache |
| `pipeline/executor.go` | Call InvalidateCache on scrape job completion |
| `web/static/js/scrape.js` | Auto-refresh pages table during active crawl |
| `web/server.go` | Disable HTTP/2 via TLSNextProto |

## Implementation Order

1. Stats cache + parallel quickStats (fixes slow list page)
2. Frontend auto-refresh pages (fixes "results only after done")
3. Disable HTTP/2 (fixes protocol error)
4. Cache invalidation on job completion
