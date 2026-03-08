# 0693: Parquet Deep-Dive Data Pages

## Goal

Enhance the single-file parquet detail page (`#/parquet/{index}`) with advanced
widget components: KPI metrics, distribution charts, and a data browser — all
tailored to the subset type (warc, non200responses, robotstxt, crawldiagnostics).

## Problems Solved

1. **Linking** — list page links index #, filename, subset, and "✓ local" badge
   to the detail page (`#/parquet/{index}`) and subset page (`#/parquet/subset/{name}`)
2. **Per-file stats** — new API endpoint runs subset-specific chart queries
   against just the one downloaded parquet file
3. **Tabbed detail page** — Overview (charts + KPIs) | Data Browser | Schema
4. **Domain-specific metrics** — each subset shows its most meaningful KPIs
   and distribution charts

## CC Columnar Index Schema

Full column set across subsets:
- URL: url, url_surtkey, url_host_name, url_host_tld, url_protocol, url_port,
  url_path, url_query, url_host_registered_domain, url_host_registry_suffix,
  url_host_private_suffix, url_host_private_domain, url_host_name_reversed,
  url_host_2nd_last_part .. url_host_5th_last_part
- Fetch: fetch_time, fetch_status, fetch_redirect
- Content: content_digest, content_mime_type, content_mime_detected,
  content_charset, content_languages, content_truncated
- WARC: warc_filename, warc_record_offset, warc_record_length, warc_segment
- Partition: crawl, subset

## Backend API

### GET /api/parquet/file/{index}/stats
Returns KPI scalars and distribution charts for a single downloaded parquet file.

Response:
```json
{
  "manifest_index": 600,
  "subset": "warc",
  "row_count": 2521033,
  "elapsed_ms": 2341,
  "kpis": {
    "unique_domains": 45231,
    "unique_tlds": 312,
    "https_pct": 87.3
  },
  "charts": {
    "tld":      [{"label": "com", "value": 1234567}, ...],
    "domain":   [{"label": "google.com", "value": 45000}, ...],
    "mime":     [...],
    "language": [...],
    "status":   [...],
    "charset":  [...],
    "protocol": [...],
    "segment":  [...]
  }
}
```

## Subset-Specific Metrics

### warc (main web content)
KPIs: unique_domains, unique_tlds, https_pct
Charts: Top TLDs, Top Domains, MIME Types, Languages, HTTP Status Codes,
        Charsets, Protocol, WARC Segments

### non200responses
KPIs: unique_domains, redirect_pct, unique_statuses
Charts: HTTP Status Codes, Top Domains, Top TLDs, Redirect Targets,
        MIME Types, Protocol

### robotstxt
KPIs: unique_domains, unique_tlds, https_pct
Charts: Top Domains, Top TLDs, HTTP Status Codes, Protocol, WARC Segments

### crawldiagnostics
KPIs: unique_domains, unique_statuses, unique_mimes
Charts: Top Domains, Top TLDs, HTTP Status Codes, MIME Types, WARC Segments

## Frontend: Single File Detail Page

### Layout

```
← Parquet Index

part-00600.parquet                        [warc]
┌───────────────────────────────────────────────┐
│  #600  │  ✓ local  │  2.52M rows  │  850MB  │  35 cols  │
│  cc-index/.../subset=warc/part-00600.parquet  │
└───────────────────────────────────────────────┘

[Overview]  [Data Browser]  [Schema]

Overview tab:
┌──────────────────────────────────────────────────────────┐
│  Unique Domains: 45,231  │  Unique TLDs: 312  │  HTTPS: 87.3%  │  Query: 2,341ms  │
└──────────────────────────────────────────────────────────┘
┌──────────────────────┬──────────────────────┐
│  Top TLDs            │  Top Domains         │
│  com ████████ 1.23M  │  google ████ 45K     │
│  org ████ 350K       │  github ████ 38K     │
│  ...                 │  ...                 │
├──────────────────────┼──────────────────────┤
│  MIME Types          │  Languages           │
│  text/html ████ ...  │  eng ████████ ...    │
├──────────────────────┼──────────────────────┤
│  HTTP Status Codes   │  Charsets            │
│  200 ██████████ ...  │  UTF-8 ████████ ...  │
└──────────────────────┴──────────────────────┘
```

### Tabs
- **Overview** (default): KPI row + 2-column chart grid. Loads async on page
  open, cached in DOM so switching tabs doesn't reload.
- **Data Browser**: paginated table with WHERE filter and ORDER BY selector.
  Loaded when first switching to data tab.
- **Schema**: column table with name, type, ordinal index.

### State management
- `state.parquetDetailIdx` — current file index (string)
- `state.parquetDetailTab` — active tab: 'overview' | 'data' | 'schema'
- `state.parquetDetailPage`, `state.parquetDetailFilter`, `state.parquetDetailSort`
- All reset when navigating to a different file index

## Implementation

### Backend (handler_parquet.go)
- `parquetFileStatsResponse` struct with manifest_index, subset, row_count,
  elapsed_ms, kpis (map[string]float64), charts (map[string][]chartEntry)
- `subsetKPIQueries` map — scalar metric queries per subset (single float64 scan)
- `subsetChartQueries` updated: increased limits to 25 domains/TLDs, added
  `segment` chart for warc/robots/diag
- `handleParquetFileStats` handler — opens single-file DuckDB view, runs KPI
  and chart queries in sequence

### Route (server.go)
```
GET /api/parquet/file/{index}/stats
```

### Frontend (parquet.js)
- `apiParquetFileStats(idx)` — API helper
- `CHART_LABELS`, `KPI_LABELS`, `fmtKPI(key, val)` — label/format helpers
- `renderParquetDetail(idx)` — resets state on new file, same shell structure
- `renderParquetDetailContent(detail)` — tabbed layout; auto-starts stats load
- `switchParquetDetailTab(tab)` — shows/hides panels, loads data on first visit
- `loadParquetFileStats()` — fetches /stats, calls renderParquetFileCharts
- `renderParquetFileCharts(data)` — KPI grid + 2-col chart grid with renderBars;
  odd-count charts get full-width last card
