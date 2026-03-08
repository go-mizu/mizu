# 0693: Parquet Deep-Dive Data Pages

## Goal

Enhance the Parquet tab with subset-specific deep-dive pages that show prebuilt
stats, distribution charts, and optimized data browsers for each CC index subset
(warc, non200responses, robotstxt, crawldiagnostics).

## Problems Solved

1. **Linking** — make every cell in the file table clickable to the detail page
2. **Subset overview pages** — clicking a subset tab shows aggregate stats with
   charts (bar charts for distributions) computed from all downloaded files of
   that subset
3. **Domain-specific columns** — each subset shows its most relevant columns
   first and hides irrelevant ones
4. **Prebuilt stats** — each subset gets a tailored stats panel with the most
   useful breakdowns for that data type

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

## Subset-Specific Stats

### warc (main web content)
- Top 20 TLDs (bar chart)
- Top 20 domains (bar chart)
- MIME type distribution (bar chart)
- Language distribution (bar chart)
- Content charset distribution (bar chart)
- Protocol distribution (http vs https)
- URL path depth histogram

### non200responses
- Status code distribution (bar chart — 301, 302, 404, 403, 500, etc.)
- Top 20 domains with errors (bar chart)
- Redirect targets (fetch_redirect top values)
- Status code by TLD crosstab

### robotstxt
- Top 20 domains (bar chart)
- TLD distribution (bar chart)
- Record count summary

### crawldiagnostics
- Top 20 domains (bar chart)
- MIME type distribution (bar chart)
- Status code distribution (bar chart)

## Backend API

### GET /api/parquet/subset/{subset}/stats
Returns precomputed distribution stats for a subset from all local parquet files
of that subset type. Each stat is a `{label, value}[]` array for easy charting.

Response:
```json
{
  "subset": "warc",
  "total_rows": 2500000,
  "file_count": 1,
  "elapsed_ms": 1234,
  "charts": {
    "tld": [{"label": "com", "value": 1234567}, ...],
    "domain": [...],
    "mime": [...],
    "language": [...],
    "charset": [...],
    "protocol": [...],
    "status": [...]
  }
}
```

## Frontend

### Enhanced file table
- Entire row clickable (via link on index # and filename)
- Downloaded status also links to detail page
- Subset name links to subset stats page

### Subset stats page (#/parquet/subset/{subset})
- Summary bar (total rows, file count, disk usage)
- Grid of bar charts — each chart is a sorted horizontal bar chart
  rendered with pure CSS (no charting library needed — use the existing
  `renderBars()` utility from utils.js)
- "View data" button that opens the query console pre-filtered to that subset

## Implementation

### Task 1: Backend handler for subset stats
### Task 2: Route registration
### Task 3: Frontend — subset stats page
### Task 4: Frontend — enhanced linking in file table
### Task 5: Frontend — router wiring
