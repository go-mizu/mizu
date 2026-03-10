# X Viewer API: Token-Protected Endpoints + Worker Mode

**Date:** 2026-03-10
**Status:** Implemented

## Overview

Two enhancements to the X viewer ecosystem:
1. **x-viewer Worker API** — token-protected JSON API with D1 caching, cache metadata in responses, `?reload=1` force-refresh, and background profile refresh
2. **CLI `--worker` mode** — `search x tweets <user>` connects to the deployed worker instead of hitting X's API directly, benefiting from the worker's D1 cache

## x-viewer Worker Changes

### API Token Protection

All `/api/*` routes now require `Authorization: Bearer <API_TOKEN>`.

- Set secret via: `wrangler secret put API_TOKEN`
- If `API_TOKEN` is not set in env, all requests are allowed (backward compat for local dev)
- HTML routes (`/:username`, `/search/*`, etc.) remain public — browser access unaffected
- OPTIONS preflight (CORS) runs before auth check

### Cache TTLs (updated)

| Resource | Old TTL | New TTL |
|----------|---------|---------|
| Profile  | 5 min   | 24 h    |
| Search   | 2 min   | 15 min  |
| Timeline | 2 min   | 2 min   |
| Tweet    | 1 h     | 1 h     |
| Follow   | 5 min   | 5 min   |

### Cache Metadata in API Responses

All `/api/*` endpoints now include a `meta` object:

**Served from cache:**
```json
{
  "profile": { ... },
  "meta": { "fromCache": true, "age": 3600, "cachedAt": 1741612800 }
}
```

**Freshly fetched:**
```json
{
  "profile": { ... },
  "meta": { "fromCache": false, "duration": 245 }
}
```

Fields:
- `fromCache` — `true` if served from D1, `false` if freshly fetched
- `age` — seconds since this entry was cached (only when `fromCache: true`)
- `cachedAt` — unix timestamp when it was cached (only when `fromCache: true`)
- `duration` — milliseconds taken to fetch (only when `fromCache: false`)

### Force Reload

Append `?reload=1` to any API endpoint to bypass cache and fetch fresh data:

```
GET /api/profile/karpathy?reload=1
GET /api/search?q=AI&reload=1
```

### Background Profile Refresh (stale-while-revalidate)

When a cached profile's `fetched_at` is >24h ago, the worker serves the stale cached version immediately and triggers a background refresh via `executionCtx.waitUntil()`. The next request will see the freshly fetched version.

### D1 Schema Change

Added `fetched_at INTEGER NOT NULL DEFAULT 0` to all tables for tracking when entries were last fetched (separate from `expires_at` which tracks expiry).

Migration for existing deployments:
```sql
ALTER TABLE profiles ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tweets ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE timelines ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE searches ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE follows ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE lists ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE list_content ADD COLUMN fetched_at INTEGER NOT NULL DEFAULT 0;
```

## CLI `--worker` Mode

`search x tweets <username>` gains a `--worker <url>` flag (default: `https://x-viewer.go-mizu.workers.dev`).

When `--worker` is set:
- Calls `GET /api/tweets/:username` on the worker instead of X's API directly
- Auth token read from env `X_VIEWER_TOKEN`
- Shows `fromCache` and `duration/age` from the meta field
- Works for limited fetches only (not `--all` — the worker returns one page at a time)

```bash
# Use worker API (reads X_VIEWER_TOKEN from env)
search x tweets karpathy --worker https://x-viewer.go-mizu.workers.dev

# Custom worker URL
search x tweets karpathy --worker http://localhost:8787
```

Environment variable: `X_VIEWER_TOKEN` — bearer token matching the worker's `API_TOKEN` secret.

## Guest Token Proxy Pool

### Background

X's guest token rate limits are per-IP at `/1.1/guest/activate.json`. By routing guest token requests through free proxies, each proxy IP gets its own search rate-limit bucket. A pool of N proxies effectively multiplies throughput N×.

### Implementation

New file `pkg/dcrawler/x/proxy_pool.go`:

- `ProxyPool` — manages a pool of proxies with good/bad caching
- `FetchGuestToken()` — tries good (cached) proxies first, falls back to fetching fresh proxy list
- `FetchGuestTokenViaProxy(addr, proto)` — fetch guest token through a specific proxy (HTTP CONNECT or SOCKS5)
- Good/bad proxy persistence: `~/data/x/good_proxies.json` and `~/data/x/bad_proxies.json`
- Bad proxies auto-expire after 24h
- Good proxy list capped at 100 entries (MRU ordering)

New file `pkg/dcrawler/x/proxy_guest.go`:
- `FetchGuestTokenFromPool()` — global singleton proxy pool, lazy initialized

### Proxy Sources

| Source | Protocol | URL |
|--------|----------|-----|
| proxifly | HTTPS | `https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/https/data.txt` |
| TheSpeedX | HTTP | `https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt` |
| monosans | SOCKS5 | `https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt` |

### Integration with `doGuestFirst`

In `doGuestFirst` (client.go), after the direct guest token rate-limit retry fails, try the proxy pool before falling back to cookie auth:

```go
if rle := asRateLimitError(guestErr); rle != nil {
    invalidateGuestToken()
    // ... existing token rotation ...
    // Proxy pool: get token from a different IP's rate-limit bucket
    if poolToken, err := FetchGuestTokenFromPool(); err == nil {
        if data, err := doGuestGraphQL(poolToken, endpoint, vars, toggles); err == nil {
            return data, nil
        }
    }
}
```

### Deployment

The worker binary (`search`) on server 2 picks up the proxy pool automatically. On first use, it fetches proxy lists and saves good proxies to `~/data/x/good_proxies.json` for reuse.
