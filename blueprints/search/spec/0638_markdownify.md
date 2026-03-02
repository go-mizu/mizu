# Spec 0638: HTML → Markdown Conversion Pipeline

## Goal

Generic `pkg/markdown` package that converts HTML bodies to clean, readable markdown
optimized for human reading and AI/LLM consumption. First consumer: `search cc markdown`
for Common Crawl bodies. Generic enough that `search hn markdown` works later without
reimplementing anything.

## Research: Readability + Markdown Algorithms

### Pipeline (two-step, order matters)

1. **Readability extraction** — remove nav, ads, footers, boilerplate → clean article HTML
2. **HTML → Markdown conversion** — structural transformation to CommonMark

Applying markdown conversion to raw HTML first wastes 9x tokens on noise (nav menus,
cookie banners, footer legal text).

### Libraries chosen

| Step | Library | Why |
|------|---------|-----|
| Readability | `markusmobius/go-trafilatura` v1.12 | Best F1=0.91 (benchmarked 983 pages), uses go-readability + go-domdistiller as fallback. Port of Python trafilatura (gold standard). |
| HTML→MD | `JohannesKaufmann/html-to-markdown/v2` v2.5 | 3.5K stars, dominant Go library. CommonMark + GFM tables. Smart escaping. ~25 MB/s. |

### Alternatives considered

- **go-readability alone**: F1=0.875, faster but lower recall on complex pages
- **No readability**: 9x more tokens, includes nav/ads/footers — unusable for LLM
- **Reader-LM (Jina)**: LLM-based approach, requires inference infra, not worth it for bulk

### Trafilatura internals

1. Preprocessing: removes scripts, styles, forms, nav, inline comments
2. Candidate extraction: text/link density ratios (Boilerpipe-inspired)
3. Tree scoring: paragraph count, link density, text length, element types
4. Deduplication: LRU sentence-level dedup
5. Metadata: JSON-LD, OpenGraph, Twitter cards, Schema.org
6. Fallback: go-readability → go-domdistiller when primary fails

### Key design decisions

- **FavorRecall** focus: for LLM consumption, prefer more content over higher precision
- **EnableFallback=true**: use readability + domdistiller when trafilatura primary fails
- **IncludeLinks=true**: preserve link structure in markdown
- **Token estimation**: `len(text) / 4` (standard GPT/Claude approximation for English)
- **Atomic writes**: `.md.gz.tmp` → rename, safe for concurrent workers

## Architecture

```
pkg/markdown/
  converter.go       — Convert(html []byte, url string) → Result
  walker.go          — Walk(ctx, cfg) — walk input dir, convert, write output
  index.go           — IndexDB: DuckDB tracking all conversions
  progress.go        — live progress display

cli/cc.go            — registers `search cc markdown` subcommand
```

### Converter pipeline

```
[]byte HTML
  → go-trafilatura (extract main content → *html.Node)
  → html.Render → clean HTML string
  → html-to-markdown/v2 → markdown string
  → Result{Markdown, Title, Language, HasContent, ...}
```

### Walker (generic, reusable)

```go
type WalkConfig struct {
    InputDir   string   // bodystore root (e.g. ~/data/common-crawl/bodies)
    OutputDir  string   // markdown root  (e.g. ~/data/common-crawl/markdown)
    IndexDB    string   // DuckDB path    (e.g. ~/data/common-crawl/markdown/index.duckdb)
    Workers    int      // errgroup limit (default: NumCPU)
    Force      bool     // re-convert existing files
    BatchSize  int      // DB write batch size
}
```

Walks `InputDir/**/*.gz`, converts each, writes to `OutputDir` (same relative path,
`.gz` → `.md.gz`). Skips existing unless `--force`.

### Directory mapping

```
Input:  ~/data/common-crawl/bodies/ab/cd/ef01...89.gz
Output: ~/data/common-crawl/markdown/ab/cd/ef01...89.md.gz

Input:  ~/data/hn/bodies/ab/cd/ef01...89.gz          (future)
Output: ~/data/hn/markdown/ab/cd/ef01...89.md.gz
```

### Index DuckDB schema

```sql
CREATE TABLE IF NOT EXISTS files (
    cid VARCHAR PRIMARY KEY,
    html_size INTEGER,
    markdown_size INTEGER,
    html_tokens INTEGER,
    markdown_tokens INTEGER,
    compression_ratio FLOAT,
    title VARCHAR,
    language VARCHAR,
    has_content BOOLEAN,
    convert_ms INTEGER,
    created_at TIMESTAMP DEFAULT current_timestamp,
    error VARCHAR
);
```

### CLI

```
search cc markdown [--force] [--workers N] [--dir PATH] [--body-store PATH]
```

### Progress display

```
Converting: 12,345 / 89,012 (13.8%)  342/s  avg 2.9ms
Skipped: 45,123  Errors: 23
HTML: 1.2 GB → MD: 234 MB (5.1x)
```

### Summary

```
Done: 12,345 converted, 45,123 skipped, 23 errors (36.2s, 341/s)
HTML: 1.2 GB → MD: 234 MB (5.1x reduction)
Tokens: avg 1,247 → 312 (4.0x reduction)
```
