# spec/0702 — Website Export (Clone)

## Status: Draft — 2026-03-10

## 1. Problem Statement

Users crawl websites via the **scrape pipeline** (dcrawler → DuckDB shards) or ingest them via the **CC pipeline** (Common Crawl WARC → DuckDB). Currently there is no way to export a crawled site as a browsable offline mirror — you can only extract markdown or parquet.

The export task produces a directory tree that mirrors the original site structure with:
- HTML files with rewritten links (navigable offline)
- Downloaded/inlined CSS, images, JS assets
- Preserved directory hierarchy matching URL paths

## 2. Goals

1. `scrape_export` job type — exports a domain from dcrawler DuckDB shards
2. `cc_export` job type — exports a domain from CC result DuckDB
3. Output: `$HOME/data/common-crawl/export/{FORMAT}/{DOMAIN}/` (or `$HOME/data/crawler/{domain}/export/`)
4. FORMAT: `html` (rewritten HTML with assets), `raw` (original HTML, no rewriting)
5. All internal links rewritten to relative paths for offline navigation
6. CSS `url()` references and `<img src>` rewritten to local paths
7. Integrates with existing pipeline job system (Task[State, Metric], NonBlockingEmit)

## 3. Design

### 3.1 Export Pipeline

```
DuckDB pages → decompress body → parse HTML → rewrite URLs → write files
                                      ↓
                              extract asset URLs → download assets → write to dirs
```

### 3.2 URL Rewriting Strategy

Unlike goclone's flat structure (all CSS in `css/`, all images in `imgs/`), we preserve the original URL path structure:

```
export/html/example.com/
├── index.html                    # /
├── about/
│   └── index.html               # /about or /about/
├── blog/
│   ├── index.html                # /blog
│   └── my-post/
│       └── index.html            # /blog/my-post
├── _assets/                      # extracted assets
│   ├── css/
│   │   └── {hash}.css            # deduplicated by content hash
│   ├── js/
│   │   └── {hash}.js
│   └── img/
│       └── {hash}.{ext}
└── _index.html                   # site index with all pages listed
```

### 3.3 Link Rewriting Rules

| Original | Rewritten |
|----------|-----------|
| `href="/about"` | `href="../about/index.html"` (relative from current page) |
| `href="https://example.com/blog"` | `href="../../blog/index.html"` (same domain → relative) |
| `href="https://other.com/page"` | unchanged (external link) |
| `src="/css/main.css"` | `src="../../_assets/css/{hash}.css"` |
| `url(/images/bg.png)` in CSS | `url(../_assets/img/{hash}.png)` |

### 3.4 Data Sources

**Scrape (dcrawler)**: Pages table with zstd-compressed `body` BLOB. Query across sharded `results_*.duckdb` files.

**CC**: Results table with uncompressed `body` VARCHAR. Query across sharded `results_*.duckdb` files in `{crawlID}/recrawl/` dir.

### 3.5 Shared Export Logic

Both scrape and CC tasks share the same core export package (`pkg/export/`) which handles:
- HTML parsing and URL rewriting (goquery)
- Asset extraction and deduplication
- Directory structure creation
- CSS `url()` rewriting
- Site index generation

## 4. Implementation

### Package: `pkg/export/`

Core export engine, shared by both pipelines.

### Task files:
- `pkg/index/web/pipeline/scrape/task_export.go`
- `pkg/index/web/pipeline/cc/task_export.go`

### Executor wiring:
- `pkg/index/web/pipeline/executor.go` — add `scrape_export` and `cc_export` cases

## 5. Non-Goals

- No live asset fetching from the internet — only assets embedded in the crawled HTML
- No JavaScript execution or SPA rendering
- No recursive crawling from the export task
- No CSS parsing for `@import` chains (only inline `url()` in `<style>` tags and style attrs)
