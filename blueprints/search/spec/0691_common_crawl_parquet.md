# 0691: Common Crawl Parquet Tab for FTS Dashboard

## Goal

Add a "Parquet" tab to the FTS dashboard (`search cc fts dashboard`) for exploring
Common Crawl's columnar index — listing manifest files, viewing stats, downloading
parquet files, and querying them directly via DuckDB `read_parquet()` (zero import).

## Architecture

New tab at `#/parquet` (dashboard-only, like WARC/Jobs). Backend adds 5 API endpoints
under `/api/parquet/`. Frontend adds `static/js/parquet.js`. All queries use DuckDB
`read_parquet()` on local files — no import step needed. Downloads run through the
existing job system with WebSocket progress.

## UI Sections

### 1. Summary Bar
- Total files in manifest (remote), downloaded count (local), disk usage
- Subset breakdown: warc / non200responses / robotstxt / crawldiagnostics
- Download progress (if any active download job)

### 2. File Browser
- Table of manifest entries: index, filename, subset, size (local), status (downloaded/remote)
- Subset filter tabs (All / warc / non200responses / robotstxt / crawldiagnostics)
- Search filter by filename
- Pagination (200 per page, matching WARC tab pattern)
- Bulk download button (selected subset or all)

### 3. Schema Viewer
- Columns from parquet metadata: name, type, order
- Auto-detected from first available local parquet file
- Collapsible section

### 4. Query Console
- SQL textarea with monospace font
- Preset query dropdown:
  - Top 20 TLDs
  - Top 20 domains
  - Status code distribution
  - MIME type distribution
  - Language distribution
  - Record count
  - Sample 100 rows
- Execute button, results as paginated table
- Row count + elapsed time shown
- Toggle: local files vs remote (S3 via httpfs)

## Backend API

### GET /api/parquet/manifest
Lists all parquet files from manifest with local download status.

Query params:
- `subset` — filter by subset (default: all)
- `q` — filter by filename
- `offset`, `limit` — pagination

Response:
```json
{
  "files": [
    {
      "manifest_index": 0,
      "remote_path": "cc-index/table/cc-main/warc/crawl=CC-MAIN-2026-04/subset=warc/part-00000.parquet",
      "filename": "part-00000.parquet",
      "subset": "warc",
      "downloaded": true,
      "local_size": 223456789
    }
  ],
  "summary": {
    "total": 300,
    "downloaded": 5,
    "disk_bytes": 1117283945,
    "by_subset": {
      "warc": {"total": 300, "downloaded": 5},
      "non200responses": {"total": 100, "downloaded": 0}
    }
  },
  "total": 300,
  "offset": 0,
  "limit": 200
}
```

### GET /api/parquet/schema
Returns schema from first available local parquet file.

Response:
```json
{
  "columns": [
    {"name": "url", "type": "VARCHAR", "order": 0},
    {"name": "url_host_name", "type": "VARCHAR", "order": 1}
  ],
  "source": "part-00000.parquet"
}
```

### POST /api/parquet/query
Executes SQL against local parquet files via `read_parquet()`.

Request:
```json
{
  "sql": "SELECT url_host_tld, COUNT(*) as cnt FROM ... GROUP BY 1 ORDER BY 2 DESC LIMIT 20",
  "mode": "local",
  "limit": 1000
}
```

Response:
```json
{
  "columns": ["url_host_tld", "cnt"],
  "rows": [["com", 1234567], ["org", 456789]],
  "total_rows": 20,
  "elapsed_ms": 1234,
  "truncated": false
}
```

### POST /api/parquet/download
Triggers download of parquet files (runs as a job).

Request:
```json
{
  "subset": "warc",
  "indices": [0, 1, 2],
  "sample": 5
}
```

Response:
```json
{
  "job": { "id": "abc123", "type": "parquet_download", "status": "running" }
}
```

### GET /api/parquet/stats
Returns aggregate stats from local parquet files.

Response:
```json
{
  "local_files": 5,
  "total_rows": 12500000,
  "disk_bytes": 1117283945,
  "schema_columns": 18,
  "crawl_id": "CC-MAIN-2026-04"
}
```

## Implementation Plan

### Task 1: Backend — Parquet handler file
Create `pkg/index/web/handler_parquet.go` with all 5 API handlers.

### Task 2: Backend — Register routes
Add parquet API routes in `server.go` Handler() method (dashboard-only block).

### Task 3: Backend — Download job type
Add "parquet_download" job type to executors.go.

### Task 4: Frontend — parquet.js
Create `static/js/parquet.js` with:
- `renderParquet()` — main entry, summary + file browser
- `renderParquetSchema()` — schema viewer
- `renderParquetQuery()` — query console with presets
- API helper functions

### Task 5: Frontend — Wire into SPA
- Add `<script src>` in index.html
- Add nav tab in index.html
- Add route in router.js
- Add state fields in state.js
- Add API functions in api.js
- Update init.js tab hiding for search-only mode

### Task 6: Frontend — Query Console
SQL editor, preset dropdown, results table, local/remote toggle.
