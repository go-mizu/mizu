# spec/0703 — Export: Dashboard UI + CLI Integration

## Status: Draft — 2026-03-10

## 1. Summary

Add export actions to the dashboard UI (Scrape tab per-domain and Domains tab for CC) and CLI subcommands. Also add `markdown` export format with navigable internal links.

## 2. Export Formats

| Format | Description | Output |
|--------|-------------|--------|
| `html` | Rewritten HTML with local links, asset refs | `export/html/{domain}/` |
| `raw` | Original HTML, no rewriting | `export/raw/{domain}/` |
| `markdown` | HTML→Markdown with navigable internal links | `export/markdown/{domain}/` |

## 3. Dashboard UI Changes

### 3.1 Scrape Tab — Domain Detail

Add "Export" button group next to existing "→ Markdown" and "→ Index" buttons in `renderScrapeDomainStatus()`:

```
[New Crawl] [Resume] [→ Markdown] [→ Index] [Export ▼]
                                                ├─ HTML
                                                ├─ Markdown
                                                └─ Raw
```

Uses `POST /api/scrape/{domain}/export` with `{format: "html"|"markdown"|"raw"}`.

### 3.2 Domains Tab — CC Domain Detail

Add export button to domain detail view. Uses `POST /api/jobs` with `{type:"cc_export", domain, format}`.

### 3.3 API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/scrape/{domain}/export` | Start scrape export job |

The generic `POST /api/jobs` already supports `cc_export` type via `createJob`.

## 4. CLI Changes

### 4.1 `search export` command

```bash
# Scrape export
search export <domain> [--format html|markdown|raw] [--data-dir DIR]

# CC export
search export <domain> --cc [--crawl CC-MAIN-2026-04] [--format html]
```

## 5. Markdown Export Format

Converts HTML to Markdown via `pkg/markdown.ConvertFast()`, then rewrites internal `[text](url)` links to relative `.md` paths.

Directory structure mirrors the site:
```
export/markdown/example.com/
├── index.md              # /
├── about/index.md        # /about
├── blog/
│   └── post-1/index.md   # /blog/post-1
└── _index.md             # site index
```

Internal link rewriting: `[About](/about)` → `[About](../about/index.md)`
