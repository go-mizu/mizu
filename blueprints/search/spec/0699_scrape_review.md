# 0699 — Scrape Dashboard: CLI Parity (Speed + Detail)

## Problem

The dashboard scrape (`/scrape`) is missing configuration options and live stats
that the CLI `search crawl-domain` provides. This causes:

1. **Speed gap**: Missing toggles for robots.txt, sitemap, subdomain, timeout
2. **Detail gap**: Live stats panel lacks timeout/blocked/skipped counts, bytes/sec,
   peak speed, retry queue, avg fetch time
3. **Config gap**: No way to set advanced options (continuous, scroll, seed URL, stale hours)

## Changes

### 1. Enhanced ScrapeState (live stats)

Add fields to `ScrapeState` emitted every 500ms:

| Field | Source | Purpose |
|-------|--------|---------|
| `timeout` | `Stats.Timeout()` | Timeout count (was lumped into failed) |
| `blocked` | `Stats.Blocked()` | Anti-bot/soft-404 detected |
| `skipped` | `Stats.Skipped()` | Adaptive URL class filter |
| `bytes_per_sec` | `Stats.ByteSpeed()` | Rolling bytes/sec |
| `peak_speed` | `Stats.peakSpeed` | Peak pages/sec |
| `retry_queue` | `Stats.RetryQLen()` | Retry queue length |
| `avg_fetch_ms` | `Stats.AvgFetchMs()` | Average fetch time |

### 2. Enhanced StartParams (config)

Add fields to `StartParams` and wire to `buildDCrawlerConfig()`:

| Field | CLI flag | dcrawler.Config |
|-------|----------|-----------------|
| `no_robots` | `--no-robots` | `RespectRobots = false` |
| `no_sitemap` | `--no-sitemap` | `FollowSitemap = false` |
| `include_subdomain` | `--include-subdomain` | `IncludeSubdomain = true` |
| `scroll_count` | `--scroll N` | `ScrollCount = N` |
| `continuous` | `--continuous` | `Continuous = true` |
| `stale_hours` | `--stale N` | `StaleHours = N` |
| `seed_url` | seed arg | `SeedURLs = [url]` |

### 3. Stats export additions

Add to `stats_export.go`:
- `PeakSpeed() float64`
- `RetryQLen() int`

### 4. Frontend: Start form advanced options

Add collapsible "Advanced" section with toggles:
- Timeout (seconds input)
- No-robots checkbox
- No-sitemap checkbox
- Include subdomain checkbox
- Scroll count (browser mode, number input)
- Continuous mode checkbox
- Seed URL (text input)

### 5. Frontend: Enhanced live stats panel

Show additional metrics in the live stats panel:
- Timeout / Blocked / Skipped counts
- Bytes/sec rate
- Peak speed
- Retry queue length
- Avg fetch time

### 6. API startRequest additions

Add new fields to `startRequest` struct and pass through to JobConfig.Source JSON.

### 7. Blocked page display fix

**Problem**: Blocked pages (CloudFront 403, soft-404 anti-bot) stored with `StatusCode: 0`
and long error strings shown verbatim in title column.

**Fix**:
- `recordBlockedHTTP()`: New method preserves HTTP status code (403) in result
- CloudFront WAF: stores `status_code=403` + short error `"blocked: WAF/CloudFront"`
- Soft-404 (isBlockedPage): stores actual HTTP status code (200/302) + reason
- Frontend: Status column shows `403` with warning color; title shows `blocked` tag

### 8. CloudFront WAF bypass analysis

CloudFront WAF uses server-side IP/rate-based rules — not a JS challenge.
Browser mode detects CloudFront and records as blocked immediately (no bypass attempt).
Unlike Cloudflare Turnstile, there is no client-side challenge to solve.

For qiita.com: 100% blocked in HTTP mode (CloudFront 403). Browser mode unlikely to help
since CloudFront blocks at the CDN layer before page rendering.

## Files Changed

| File | Change |
|------|--------|
| `pkg/dcrawler/stats_export.go` | Add PeakSpeed(), RetryQLen() |
| `pkg/dcrawler/crawler.go` | Add recordBlockedHTTP() preserving status code |
| `pkg/index/web/pipeline/scrape/store.go` | Add fields to StartParams |
| `pkg/index/web/pipeline/scrape/task_scrape.go` | Add fields to ScrapeState, snapshot() |
| `pkg/index/web/pipeline/executor.go` | Wire new params in buildDCrawlerConfig() |
| `pkg/index/web/api/scrape.go` | Add fields to startRequest, use json.Marshal for Source |
| `pkg/index/web/static/js/scrape.js` | Advanced form, enhanced live stats, blocked tag display |

## Implementation Order

1. Stats exports (PeakSpeed, RetryQLen)
2. StartParams + ScrapeState fields
3. buildDCrawlerConfig wiring
4. API request fields
5. Frontend form + live stats
6. Blocked page status code + display fix
7. Build + test on qiita.com

## Test Results

- **books.toscrape.com**: 59 pages, 21.9 pages/sec, 582 KB/s — matches CLI speed
- **qiita.com**: 8,568 pages/sec, 100% blocked (CloudFront 403) — status codes now show correctly
- All tests pass: `pkg/dcrawler/...`, `pkg/index/web/pipeline/scrape/...`
