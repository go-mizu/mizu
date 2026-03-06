# 0672 — WARC-MD Browse Enhancement

**Date:** 2026-03-06
**Scope:** `search cc fts` pipeline + browse UI
**Branch:** vector-pane

---

## Overview

Switch the canonical markdown intermediate format from per-document `.md` files to `.md.warc.gz` (WARC conversion records). This preserves rich header metadata — URL, crawl date, original record ID — that the `.md` format discards. Use that metadata to populate a new per-document metastore table and drive a significantly enhanced browse UI.

---

## Current State

### Pipeline

```
.warc.gz → [cc warc markdown phase 1] → warc_single/ (temp)
         → [cc warc markdown phase 2] → markdown/{warcIdx}/{xx}/{yy}/{zz}/{uuid}.md
```

Each `.md` file is a plain UTF-8 text file — **no URL, no date, no record ID**. The metadata is permanently discarded during conversion.

### Metastore

Two aggregate record types only:
- `WARCRecord` — per-WARC aggregate (doc count, bytes)
- `SummaryRecord` — per-crawl aggregate

No per-document table exists.

### Browse API

```
GET /api/browse?shard=00001
→ { shards: [{name, file_count}] }  (if no shard param)
→ { files: [{name, size}] }          (if shard specified)
```

`name` = UUID filename, `size` = file size in bytes. No URL, no title, no date.

```
GET /api/doc/{shard}/{docid}
→ { doc_id, shard, markdown, html, word_count, size }
```

No URL, no title, no date.

### Browse UI

- Shard list: name + estimated doc count
- Document list: UUID filename + bytes
- Document detail: rendered markdown only

---

## Target State

### Pipeline

```
.warc.gz → [cc warc pack] → warc_md/{warcIdx}.md.warc.gz
```

Each record in `.md.warc.gz` contains:
```
WARC-Type:       conversion
WARC-Target-URI: https://example.com/page     ← URL preserved
WARC-Date:       2026-01-15T03:22:11Z         ← crawl date preserved
WARC-Record-ID:  <urn:uuid:abc123>
WARC-Refers-To:  <urn:uuid:def456>            ← original .warc.gz record ID
Content-Type:    text/markdown
Content-Length:  8421
[blank line]
[markdown body]
```

The `cc warc pack` pipeline (`pkg/warc_md/pack.go`) already produces this format. The pipeline change is to make `pack` the canonical intermediate step and update all downstream tooling to read from `warc_md/` instead of `markdown/`.

### Document IDs

In the new format, document IDs are derived from `WARC-Record-ID`:
```
<urn:uuid:9c4852b9-f2bb-46c8-92a2-ab8619823d9e>  →  doc_id = 9c4852b9-f2bb-46c8-92a2-ab8619823d9e
```

The shard is the zero-padded WARC index (e.g. `00001`), same as before.

### Metastore: New `DocRecord` Table

Add a new `DocRecord` type to `pkg/index/web/metastore/types.go`:

```go
// DocRecord is a per-document metadata entry derived from a .md.warc.gz record header.
// Body is NOT stored — only headers + stats extracted at scan time.
type DocRecord struct {
    CrawlID      string    // e.g. "CC-MAIN-2026-04"
    Shard        string    // WARC index, e.g. "00001"
    DocID        string    // UUID from WARC-Record-ID
    URL          string    // WARC-Target-URI
    Title        string    // extracted from first H1/H2 in markdown, or URL hostname
    CrawlDate    time.Time // WARC-Date parsed
    SizeBytes    int64     // Content-Length
    WordCount    int       // word count estimated from Content-Length ÷ 5 (or full scan)
    WARCRecordID string    // WARC-Record-ID (full, including <urn:uuid:...>)
    RefersTo     string    // WARC-Refers-To (original .warc.gz record ID)
    ScannedAt    time.Time // when this record was indexed into metastore
}
```

**Storage**: One SQLite table `doc_records` per crawl in the metadata DB:
```sql
CREATE TABLE IF NOT EXISTS doc_records (
    doc_id        TEXT PRIMARY KEY,
    shard         TEXT NOT NULL,
    url           TEXT NOT NULL,
    title         TEXT NOT NULL,
    crawl_date    DATETIME,
    size_bytes    INTEGER DEFAULT 0,
    word_count    INTEGER DEFAULT 0,
    warc_record_id TEXT,
    refers_to     TEXT,
    scanned_at    DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_doc_shard ON doc_records(shard);
CREATE INDEX IF NOT EXISTS idx_doc_url   ON doc_records(url);
```

**Population**: New `ScanDocRecords(crawlID, warcMdDir, db)` function in `pkg/index/web/metastore/`:

```go
// ScanDocRecords scans all .md.warc.gz files in warcMdDir and upserts DocRecords.
// Only reads WARC headers (no body) for efficiency — each member is a separate gzip stream.
// Returns (inserted, updated, error).
func ScanDocRecords(ctx context.Context, crawlID, warcMdDir string, db *sql.DB) (int64, int64, error)
```

The scanner:
1. Lists `warc_md/*.md.warc.gz` files
2. For each file, opens it as a multi-member gzip stream
3. Reads WARC headers only (stops reading body via `io.Discard` + `Content-Length`)
4. Extracts `WARC-Target-URI`, `WARC-Date`, `WARC-Record-ID`, `WARC-Refers-To`, `Content-Length`
5. Reads only the first 256 bytes of markdown body to extract title (first non-empty `# ` or `## ` line)
6. Upserts into `doc_records` table

**Title extraction** from first 256 bytes of markdown:
```go
func extractTitle(markdownHead []byte, fallbackURL string) string {
    for _, line := range bytes.Split(markdownHead, []byte("\n")) {
        line = bytes.TrimSpace(line)
        if bytes.HasPrefix(line, []byte("# ")) {
            return string(bytes.TrimPrefix(line, []byte("# ")))
        }
        if bytes.HasPrefix(line, []byte("## ")) {
            return string(bytes.TrimPrefix(line, []byte("## ")))
        }
    }
    // Fallback: hostname from URL
    if u, err := url.Parse(fallbackURL); err == nil {
        return u.Hostname()
    }
    return fallbackURL
}
```

### Stale Detection

Add `DocScanMeta` to `SummaryRecord` (or a separate table):

```go
type DocScanMeta struct {
    CrawlID      string
    TotalDocs    int64
    LastScannedAt time.Time
    // TTL: 1 hour. If Now() - LastScannedAt > 1h → meta_stale = true in API response.
}
```

**Background refresh logic** in `meta_manager.go`:
- `GET /api/browse` checks `LastScannedAt`; if > 1h stale → sets `meta_stale: true` in response AND calls `TriggerDocScan()` in background
- `TriggerDocScan()` runs `ScanDocRecords` in a goroutine (non-blocking)
- Progress broadcast via WebSocket job event: `{type: "doc_scan", shard: ..., progress: N}`
- On scan completion: `LastScannedAt` updated; WebSocket broadcast `{type: "doc_scan_done", total: N}`
- Frontend listens for `doc_scan_done` → auto-reload browse page

**Frontend stale indicator**: If `meta_stale: true` in browse response, show amber banner:
```
"Document metadata is being refreshed in the background..."
```
Banner auto-dismisses on `doc_scan_done` WebSocket event.

---

## API Changes

### `GET /api/browse`

**Without `shard` param** (shard list):
```json
{
  "shards": [
    { "name": "00001", "file_count": 18432, "total_size": 142389120 }
  ],
  "meta_stale": false,
  "last_scanned_at": "2026-03-06T10:22:00Z"
}
```

**With `shard=00001`** (document list):
```json
{
  "shard": "00001",
  "docs": [
    {
      "doc_id": "9c4852b9-f2bb-46c8-92a2-ab8619823d9e",
      "url": "https://example.com/article/hello-world",
      "title": "Hello World — Example Blog",
      "crawl_date": "2026-01-15T03:22:11Z",
      "size_bytes": 8421,
      "word_count": 1680
    }
  ],
  "total": 18432,
  "page": 1,
  "page_size": 100,
  "meta_stale": false,
  "last_scanned_at": "2026-03-06T10:22:00Z"
}
```

**Query params** (document list):
- `page` (default 1)
- `page_size` (default 100, max 500)
- `q` — filter by URL substring or title substring (case-insensitive SQLite `LIKE`)
- `sort` — `url|title|date|size|words` (default `date` desc)

**Fallback**: If no `doc_records` exist for a shard (doc scan not yet run), fall back to current filesystem scan returning `{doc_id: name, url: "", title: name, crawl_date: null, size_bytes: size}`.

### `GET /api/doc/{shard}/{docid}`

Enhanced response:
```json
{
  "doc_id": "9c4852b9-...",
  "shard": "00001",
  "url": "https://example.com/article/hello-world",
  "title": "Hello World — Example Blog",
  "crawl_date": "2026-01-15T03:22:11Z",
  "size_bytes": 8421,
  "word_count": 1680,
  "warc_record_id": "<urn:uuid:9c4852b9-...>",
  "refers_to": "<urn:uuid:def45678-...>",
  "markdown": "# Hello World\n\n...",
  "html": "<h1>Hello World</h1>..."
}
```

The `markdown` and `html` fields are populated by reading the body from `.md.warc.gz` (same as before from `.md` files, but now via WARC reader). The metadata fields come from `doc_records` table.

### `POST /api/meta/scan-docs`

Trigger an immediate doc scan (dashboard mode only):
```json
{ "crawl_id": "CC-MAIN-2026-04" }
```
Response: `{ "job_id": "...", "status": "queued" }`

Progress tracked as a job, visible in Jobs tab.

---

## Browse UI Changes

### Shard List Page

Current:
```
Shard    Docs
00001    ~18432
00002    ~17891
```

Enhanced:
```
Shard    Docs      Size       Last Document Date    Status
00001    18,432    135.7 MB   2026-01-15            ●
00002    17,891    129.3 MB   2026-01-14            ● stale
```

- Size column: sum of `size_bytes` for shard from `doc_records`
- Last Document Date: max `crawl_date` for shard
- Status dot: green if fresh, amber if stale (>1h since `last_scanned_at`)
- "Refresh metadata" button triggers `POST /api/meta/scan-docs`

### Document List Page (`#/browse/00001`)

Current: UUID filename + bytes, no other info.

Enhanced table:
```
Title                          URL                              Date          Size     Words
Hello World — Example Blog     https://example.com/...          Jan 15, 2026  8.2 KB   1,680
Another Article                https://news.ycombinator.com/... Jan 14, 2026  4.1 KB   820
```

- **Title**: clickable → opens doc detail page
- **URL**: clickable external link (opens in new tab, `rel="noopener noreferrer"`)
- **Date**: human-readable crawl date
- **Size**: `fmtBytes(size_bytes)`
- **Words**: word count
- Search/filter bar: live filter by title or URL (sends `?q=` to API, debounced 300ms)
- Sort controls: Date (default), Size, Words, Title
- Pagination: "Showing 1–100 of 18,432 documents" + prev/next buttons
- If `meta_stale: true`: amber banner "Refreshing metadata in background..." with spinner

### Document Detail Page (`#/browse/00001/9c4852b9-...`)

Enhanced header section above markdown content:

```
┌─────────────────────────────────────────────────────────────────┐
│  Hello World — Example Blog                                     │
│  https://example.com/article/hello-world  [↗ Open]             │
│  Crawled: January 15, 2026 03:22 UTC  ·  8.2 KB  ·  1,680 words│
│  Record: 9c4852b9-f2bb-46c8-92a2-ab8619823d9e                   │
└─────────────────────────────────────────────────────────────────┘

[markdown rendered below]
```

CSS: `doc-header` card with left border `var(--accent)`, title in `h2`, URL in monospace with external link icon, metadata row in `text-muted`.

---

## Implementation Plan

### Phase 1 — Metastore: DocRecord table + scanner

**Files to create/modify:**

1. `pkg/index/web/metastore/types.go` — add `DocRecord`, `DocScanMeta`
2. `pkg/index/web/metastore/doc_scan.go` (new) — `ScanDocRecords()`, `extractTitle()`
3. `pkg/index/web/meta_manager.go` — wire `TriggerDocScan()`, `DocScanMeta` refresh

**Key constraint**: Scanner must read headers only for speed. A 10K-doc shard file at ~5 KB/doc = ~50 MB. Full body read = slow. Header-only scan: seek past body using `Content-Length`, read first 256 bytes for title, then skip rest.

**Estimated scan speed**: ~5,000 docs/second (header-only gzip read). 18K docs = ~4s per shard.

### Phase 2 — API: enhanced browse + doc endpoints

**Files to modify:**

4. `pkg/index/web/server.go` — update `handleBrowse`, `handleDoc`, add `handleMetaScanDocs`

Changes to `handleBrowse`:
- Query `doc_records` for shard list with doc count + total size + max date
- Query `doc_records` for document list with pagination + filter + sort
- Add `meta_stale` check: `time.Since(docScanMeta.LastScannedAt) > time.Hour`
- Fallback to filesystem scan if no doc records

Changes to `handleDoc`:
- After loading markdown from `.md.warc.gz`, join with `doc_records` for URL/title/date/word_count

### Phase 3 — Frontend: browse UI

**File to modify:**

5. `pkg/index/web/static/index.html` — rewrite `renderBrowseContent()` and browse detail section

Key JS functions to update:
- `apiBrowseShards()` — parse new `total_size`, `last_document_date`, `meta_stale`
- `apiBrowseDocs(shard, page, q, sort)` — new pagination + filter params
- `renderBrowseShard(shard, docs, meta)` — new table with title/url/date/size/words
- `renderBrowseDoc(shard, docid)` — add header card with URL, date, size

New JS helpers:
- `fmtDate(isoStr)` — "Jan 15, 2026"
- `truncateURL(url, maxLen)` — truncate long URLs with ellipsis in middle
- `renderDocMetaHeader(doc)` — header card HTML

### Phase 4 — Pipeline: `cc warc markdown` deprecation notice

6. `cli/cc_warc_markdown.go` — add deprecation notice pointing to `cc warc pack`
7. `spec/0670_warc_md.md` — update to document new canonical pipeline position

The old 2-phase pipeline (`cc warc markdown`) can remain for users who need plain `.md` files, but documentation should clearly indicate `cc warc pack` → `.md.warc.gz` is the recommended path for dashboard use.

---

## Data Flow Summary

```
.warc.gz
   │
   ▼ cc warc pack (existing)
warc_md/{warcIdx}.md.warc.gz
   │
   ├──▶ FTS indexing (existing, reads markdown body)
   │
   └──▶ Doc scan (new, reads headers + 256B title)
            │
            ▼
        doc_records (SQLite)
            │
            ├──▶ GET /api/browse?shard=X
            │       → paginated doc list with URL/title/date/size
            │
            └──▶ GET /api/doc/{shard}/{docid}
                    → full metadata + markdown body
```

---

## Design Decisions

### Why header-only scan?

`.md.warc.gz` uses per-record gzip members. Each member is a separate gzip stream. We can open the file, iterate gzip members, read each WARC header (typically 200–400 bytes), read 256 bytes of markdown for title, then call `gzip.Reader.Close()` and open the next member. This avoids reading the full body (~5 KB/doc avg). For 18K docs: ~8 MB of header reads vs ~90 MB full scan.

### Why SQLite for doc_records?

Same metastore DB as WARC records. Simple `INSERT OR REPLACE` upsert. Fast `LIKE` for filtering. No additional dependency. For 1M+ docs, consider DuckDB shard, but current CC crawls have ~500K docs/crawl.

### Why 1-hour TTL?

Dashboard is typically left open while running the pack pipeline. 1h is long enough to avoid constant re-scans while short enough to catch new `.md.warc.gz` files created by a completed pack job. The scan trigger is also exposed via "Refresh Metadata" button for immediate manual refresh.

### Title extraction limit of 256 bytes

Nearly all markdown documents have their `# Title` in the first line. 256 bytes covers even long titles. Avoids reading multi-KB bodies during bulk scan.

### Fallback behavior

If `doc_records` is empty (first launch, before any scan), `GET /api/browse` returns the same data as today (filesystem-based). Progressive enhancement — the browse page degrades gracefully and shows the "Refreshing metadata" banner until the first scan completes.
