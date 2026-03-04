# spec/0651: FTS Web GUI

## Goal

Add `search cc fts web` command that launches an embedded HTTP server with a
modern, shadcn-inspired web GUI for full-text search and markdown browsing.
Single Go binary — no npm, no build step, no external dependencies.

## CLI

```
search cc fts web [flags]
  --port     int    Listen port (default 3456)
  --engine   string FTS engine (default "duckdb")
  --crawl    string Crawl ID (default: auto-detect latest)
  --addr     string External engine address (meilisearch, etc.)
  --open            Open browser on start
```

## Architecture

```
cli/cc_fts_web.go          CLI command + HTTP server setup
pkg/index/web/             Web server package
  server.go                Mizu app, routes, middleware
  handler.go               API handlers (search, stats, doc, browse)
  html.go                  Embedded HTML/CSS/JS (go:embed)
  markdown.go              goldmark renderer
pkg/index/web/static/
  index.html               Single-page app (Tailwind CDN + vanilla JS)
```

## API Endpoints

| Method | Path | Response |
|--------|------|----------|
| GET | `/api/search?q=&limit=10&offset=0` | `{hits, total, elapsed_ms}` |
| GET | `/api/stats` | `{shards, total_docs, total_disk, engine}` |
| GET | `/api/doc/{shard}/{docid}` | `{doc_id, shard, markdown, html}` |
| GET | `/api/browse?shard=&prefix=` | `{files: [{name, size, is_dir}]}` |
| GET | `/{path...}` | Embedded SPA HTML |

### Search Response

```json
{
  "hits": [
    {"doc_id": "abc123.md", "shard": "00000", "score": 4.21, "snippet": "...highlighted..."}
  ],
  "total": 1234,
  "elapsed_ms": 42,
  "query": "machine learning",
  "engine": "duckdb",
  "shards": 3
}
```

## Pages

### 1. Search (default `/`)

- Large centered search input with keyboard shortcut (Ctrl+K focus)
- Engine name + shard count badge
- Results as cards: doc title (derived from DocID), score badge, snippet with
  highlighting, shard indicator
- Pagination (prev/next + page numbers)
- Click result → document viewer
- Empty state: shows stats (doc count, engine, shards)

### 2. Document Viewer (`#/doc/{shard}/{docid}`)

- Back button to search results
- Breadcrumb: shard / filename
- Rendered markdown with prose typography (headings, lists, code blocks, tables)
- Raw markdown toggle
- Metadata sidebar: shard, file size, word count

### 3. Browse (`#/browse`)

- Left sidebar: shard list (00000, 00001, ...)
- Main area: file list for selected shard
- Click file → document viewer
- File count per shard
- Sort by name

## Styling (shadcn-inspired)

- **Colors**: Zinc-900 bg (dark), white bg (light), slate accents
- **Font**: Inter (CDN) for body, JetBrains Mono for code
- **Components**: rounded-lg borders, ring focus states, subtle shadows
- **Dark mode**: `prefers-color-scheme` + toggle button
- **Cards**: border border-zinc-200 dark:border-zinc-800, hover:shadow-md
- **Search input**: large, centered, with search icon and Ctrl+K hint
- **Prose**: Tailwind Typography plugin (CDN) for markdown rendering
- **Responsive**: mobile-friendly with collapsible sidebar

## Implementation Notes

- **goldmark** for markdown→HTML (pure Go, no CGO)
- Fan-out search reuses the same parallel shard pattern from `runCCFTSSearch`
- DocID maps to filename in `markdown/{warcIdx}/` directory
- Document fetch reads raw `.md` file, renders with goldmark
- Browse reads directory listing from filesystem
- Server gracefully shuts down on SIGINT/SIGTERM
- `--open` uses `exec.Command("open", url)` on macOS

## Non-Goals

- No authentication / multi-user
- No index modification from the web UI
- No WebSocket / real-time updates
- No server-side sessions
