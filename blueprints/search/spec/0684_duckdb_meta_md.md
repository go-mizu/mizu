# 0684: DuckDB Metadata from .md.warc.gz

## Problem

The Browse page shows empty URL, Host, Date, and Top Domains because the current
scan path falls back to individual `.md` files in `markdown/{shard}/` which have
no metadata. The `.md.warc.gz` files (created by the pack pipeline) preserve all
WARC headers: URL, date, record-ID, refers-to.

Additionally, reading a single document requires scanning the entire `.md.warc.gz`
file sequentially — O(N) per doc view.

## Solution

### 1. Schema: add offset, size, host

```sql
CREATE TABLE IF NOT EXISTS doc_records (
  doc_id         TEXT PRIMARY KEY,
  url            TEXT NOT NULL DEFAULT '',
  host           TEXT NOT NULL DEFAULT '',
  title          TEXT NOT NULL DEFAULT '',
  crawl_date     TEXT NOT NULL DEFAULT '',
  size_bytes     BIGINT DEFAULT 0,
  word_count     INTEGER DEFAULT 0,
  warc_record_id TEXT NOT NULL DEFAULT '',
  refers_to      TEXT NOT NULL DEFAULT '',
  gzip_offset    BIGINT DEFAULT 0,
  gzip_size      BIGINT DEFAULT 0,
  scanned_at     TEXT NOT NULL DEFAULT ''
)
```

- `gzip_offset`: byte offset of the gzip member in `.md.warc.gz`
- `gzip_size`: byte length of the compressed gzip member
- `host`: extracted from URL via `url.Parse().Hostname()` (stripped `www.`)

### 2. CountingReader for offset tracking

During `scanWARCMd`, wrap the `*os.File` in a `CountingReader` that tracks the
current read position. Before each gzip member reset, record the offset. After
reading headers + body, record end offset. `gzip_size = end - start`.

### 3. Fast random-access doc retrieval

`ReadDocByOffset(warcMdPath string, offset, size int64) ([]byte, error)`:
1. `os.Open(path)` → `Seek(offset, io.SeekStart)`
2. `io.LimitReader(f, size)` → `gzip.NewReader()` → read single WARC record
3. Return markdown body

Replaces the sequential `readDocFromWARCMd` scan.

### 4. Remove markdown/ fallback

- Delete `scanMarkdownDir`, `docIDToRelPath`
- Remove `resolveMarkdownPath` fallback to `markdown/{shard}/`
- `handleDoc` uses offset-based read, falls back to sequential scan only if
  offset is 0 (legacy data)

### 5. Title extraction improvement

Increase head read from 256 → 2048 bytes for better H1/H2 hit rate.

### 6. Host extraction for Top Domains

Extract host from `WARC-Target-URI` during scan. Strip `www.` prefix.
The ShardStats `TOP DOMAINS` query already works when `url != ''` — it was
empty only because markdown/ files had no URL.

## Files Changed

- `pkg/index/web/doc_store.go` — schema, CountingReader, offset scan, fast read
- `pkg/index/web/server.go` — remove markdown/ fallback, offset-based doc read
- `pkg/index/web/static/js/browse.js` — add Host column
- `pkg/index/web/doc_store_test.go` — new test file
