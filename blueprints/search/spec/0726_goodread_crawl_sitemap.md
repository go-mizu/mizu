# spec/0726 — Goodreads Sitemap-Seeded Crawl

## Goal

Add `search goodread crawl --seed sitemap` to discover all Goodreads URLs from
sitemaps and immediately crawl them in one resumable command, with rich progress,
Ctrl+C cancellation, and before/after summaries.

---

## Problem

Today the workflow is two separate commands with no progress on seeding:

```bash
search goodread sitemap --limit 0   # seed (slow, silent)
search goodread crawl               # crawl (ok progress, no ETA)
```

Pain points:
1. Sitemap seeding is silent — no progress, no count, no estimated time
2. Two separate commands — easy to forget to seed, or not know queue state
3. Crawl progress has no ETA
4. No pre-run summary (how many pending? how many already done?)
5. Ctrl+C kills the process without printing a final summary
6. `search[field]=books` URL bug made HTML search return 0 results (**already fixed**)

---

## Solution: `crawl --seed sitemap`

Extend the crawl command with a single new flag:

```
search goodread crawl --seed sitemap [--workers 4] [--delay 1500] [--type book,author]
```

### Phase 1 — Seed from sitemaps

Runs before crawl starts. For each siteindex file in robots.txt:

1. Print each siteindex being fetched and count of .gz files inside it
2. Fetch every .gz sitemap file, decompress, enqueue URLs
3. Show a live counter: `  Seeding: 1,234,567 URLs discovered (12 of 48 files)`
4. Skip URL types not in `--type` filter (default: book, author, series, list, quote, user, genre)
5. At end: print total newly enqueued vs already-in-queue (skipped as duplicate)

### Phase 2 — Crawl

Starts immediately after seeding (or without seeding if `--seed` not given).

Progress line (overwritten every 2s):
```
  done=12,345  pending=987,654  failed=23  in-flight=4  rps=2.1  eta=131h
```

- **eta**: `pending / rps` formatted as `Xd Xh`, `Xh Xm`, or `Xm`
- **rps**: rolling average over the session

### Pre-run summary

Printed before starting crawl:
```
── Queue before crawl ──
  Pending:    987,654
  In progress: 0
  Done:        12,345
  Failed:      23
  Total:       1,000,022

Starting crawl: workers=4  delay=1500ms  db=/Users/.../goodread.duckdb
```

### Post-run / Ctrl+C summary

Same format, printed on clean exit or Ctrl+C:
```
── Crawl summary ──
  Done:        24,690
  Failed:      45
  Duration:    1h23m
  Throughput:  5.0 req/s avg

── Queue after ──
  Pending:    975,309
  Done:        24,690
  Failed:      45
```

On Ctrl+C, signal handler:
1. Cancels the context (workers drain gracefully)
2. Waits for in-flight requests to complete (up to 30s)
3. Prints the summary

---

## Architecture

### Files changed

| File | Change |
|------|--------|
| `cli/goodread.go` | Extend `newGoodreadCrawl`: add `--seed` flag, pre/post summary, Ctrl+C handler, ETA display |
| `cli/goodread.go` | Extract `seedFromSitemap(ctx, stateDB, filter, verbose)` helper with progress |
| `pkg/scrape/goodread/display.go` | Update `PrintCrawlProgress` to include ETA; add `PrintCrawlSummary` |

No new files needed — all changes are in existing files.

### ETA calculation

```go
elapsed := time.Since(start).Seconds()
rps := float64(done) / elapsed
var etaStr string
if rps > 0 && pending > 0 {
    secs := float64(pending) / rps
    etaStr = formatETA(time.Duration(secs) * time.Second)
}
```

`formatETA`:
- `>= 24h` → `"Xd Xh"`
- `>= 1h`  → `"Xh Xm"`
- `>= 1m`  → `"Xm Xs"`
- else     → `"Xs"`

### Ctrl+C handler

```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
go func() {
    <-sigCh
    fmt.Println("\nInterrupted — finishing in-flight requests...")
    cancel()
}()
```

After `task.Run()` returns, always print summary regardless of exit reason.

### Sitemap seed with progress

```go
func seedFromSitemap(ctx context.Context, stateDB *goodread.State, typeFilter string, verbose bool) (int, int, error) {
    // Fetch robots.txt → siteindex URLs
    // For each siteindex:
    //   Fetch .gz list → for each .gz: decompress + enqueue
    //   Print progress counter
    // Return (newlyEnqueued, skipped)
}
```

Progress output (overwritten with `\r`):
```
  Seeding [book]: 123,456 URLs  (file 7/23)
```

---

## CLI flags

```
search goodread crawl [flags]

Flags:
  --seed string       Seed strategy before crawling: sitemap (default: none)
  --type string       Comma-separated entity types to seed/crawl: book,author,series,list,quote,user,genre
                      (default: all)
  --workers int       Concurrent fetch workers (default: 2)
  --delay int         Delay between requests in milliseconds (default: 2000)
  --max-pages int     Max pages per entity (default: 0 = unlimited)
  --db string         Path to goodread.duckdb
  --state string      Path to state.duckdb
```

---

## Resumability

Already handled by the existing DuckDB queue:
- `status='done'` URLs are skipped by `Enqueue` (UNIQUE constraint → silently ignored)
- Re-running the command re-seeds (new URLs only) and re-crawls (pending only)
- `status='in_progress'` items from a killed run stay in-progress — add a reset step at startup: `UPDATE queue SET status='pending' WHERE status='in_progress'`

---

## Siteindex types discovered from robots.txt

Expected types from `https://www.goodreads.com/robots.txt`:
- `siteindex.author.xml` → author
- `siteindex.book.xml` → book
- `siteindex.list.xml` → list
- `siteindex.quote.xml` → quote
- `siteindex.user.xml` → user
- `siteindex.topic.xml` → (skip, no entity task)
- `siteindex.group.xml` → (skip)
- `siteindex.work.xml` → book (same as book)

The `--type` filter allows focusing on high-value types (e.g. `--type book,author`).
