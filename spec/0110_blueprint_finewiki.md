# FineWiki Blueprint Specification

Version: 1.0.0
Status: Implementation
Module: `github.com/go-mizu/blueprints/finewiki`

## Overview

FineWiki is a fast, read-only wiki viewer that serves Wikipedia-style articles from Parquet files using DuckDB as an embedded query engine. The application is built on the Mizu web framework and provides server-side rendered HTML pages without client-side JavaScript dependencies for core functionality.

### Design Principles

1. **Single Binary Deployment** - No external services, databases, or message queues
2. **Parquet as Source of Truth** - DuckDB reads Parquet directly; no ETL pipelines
3. **Title-First Search** - Fast autocomplete via indexed title lookups
4. **Server-Side Rendering** - All pages rendered on server; minimal/no JS
5. **Blueprint Quality** - Production-ready code that serves as a reference implementation

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         HTTP Layer                               │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Mizu Router + Middleware (logging, recovery)               ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Web Handlers                               │
│  ┌──────────┐  ┌──────────────┐  ┌────────────────────────────┐│
│  │  Home    │  │   Search     │  │       Page View            ││
│  │    /     │  │   /search    │  │   /page?id=... or          ││
│  └──────────┘  └──────────────┘  │   /page?wiki=...&title=... ││
│                                   └────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Feature Services                            │
│  ┌─────────────────────────────┐  ┌────────────────────────────┐│
│  │     search.Service          │  │      view.Service          ││
│  │  • Query normalization      │  │  • Page retrieval by ID    ││
│  │  • Limit enforcement        │  │  • Page retrieval by title ││
│  │  • Store delegation         │  │  • Content preparation     ││
│  └─────────────────────────────┘  └────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Data Store                                │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    duckdb.Store                             ││
│  │  • Schema initialization (titles table, meta table)         ││
│  │  • Title extraction and indexing                            ││
│  │  • Fast title search (exact → prefix → FTS)                 ││
│  │  • Page retrieval from Parquet                              ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                      DuckDB                                 ││
│  │  • Native Parquet reader                                    ││
│  │  • In-process SQL engine                                    ││
│  │  • FTS extension (optional)                                 ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## Package Structure

```
finewiki/
├── cmd/finewiki/
│   ├── main.go              # CLI entrypoint, server bootstrap
│   └── views/               # Embedded HTML templates
│       ├── layout/app.html
│       ├── component/
│       │   ├── topbar.html
│       │   ├── sidebar.html
│       │   └── chips.html
│       └── page/
│           ├── home.html
│           ├── search.html
│           └── view.html
│
├── app/web/
│   ├── server.go            # Server struct, route registration
│   ├── handlers.go          # HTTP handlers
│   ├── render.go            # Template rendering interface
│   └── middleware.go        # Logging middleware
│
├── feature/
│   ├── search/
│   │   ├── api.go           # Query, Result, Store/API interfaces
│   │   └── service.go       # Business logic
│   └── view/
│       ├── api.go           # Page struct, Store/API interfaces
│       └── service.go       # Business logic
│
├── store/duckdb/
│   ├── store.go             # Store implementation
│   ├── import.go            # Parquet import helpers
│   ├── schema.sql           # DDL statements
│   └── seed.sql             # Title extraction
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Data Model

### FineWiki Dataset Schema (Parquet)

The FineWiki dataset from Hugging Face contains the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (e.g., "enwiki/32552979") |
| `wikiname` | string | Wiki source (e.g., "enwiki") |
| `page_id` | int64 | MediaWiki page ID |
| `title` | string | Article title |
| `url` | string | Original Wikipedia URL |
| `date_modified` | string | Last modification timestamp |
| `in_language` | string | Language code (e.g., "en") |
| `text` | string | Plain text content |
| `wikidata_id` | string | Wikidata entity ID (optional) |
| `bytes_html` | int64 | Original HTML size |
| `has_math` | bool | Contains math markup |
| `wikitext` | string | MediaWiki markup (optional) |
| `version` | string | Dataset version |
| `infoboxes` | string | JSON array of infoboxes |

### Local Database Schema

**titles** - Fast title search index

```sql
CREATE TABLE IF NOT EXISTS titles (
  id          VARCHAR PRIMARY KEY,
  wikiname    VARCHAR NOT NULL,
  in_language VARCHAR NOT NULL,
  title       VARCHAR NOT NULL,
  title_lc    VARCHAR NOT NULL  -- lowercase for case-insensitive search
);

CREATE INDEX idx_titles_title_lc ON titles(title_lc);
CREATE INDEX idx_titles_wikiname ON titles(wikiname);
CREATE INDEX idx_titles_lang ON titles(in_language);
```

**meta** - Database metadata

```sql
CREATE TABLE IF NOT EXISTS meta (
  k VARCHAR PRIMARY KEY,
  v VARCHAR NOT NULL
);
```

## Search Design

### Search Strategy

Search uses a tiered approach for optimal performance:

1. **Exact Match** (fastest)
   - `WHERE title_lc = lower(:query)`
   - Returns immediately if results found

2. **Prefix Match** (fast)
   - `WHERE title_lc LIKE lower(:query) || '%'`
   - Covers autocomplete use cases

3. **FTS Fallback** (optional, slower)
   - Uses DuckDB FTS extension
   - Only enabled via `--fts` flag
   - Handles word-based fuzzy matching

### Query Parameters

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `q` | string | Search text (required, min 2 chars) | - |
| `wiki` | string | Filter by wiki name | all |
| `lang` | string | Filter by language | all |
| `limit` | int | Max results | 20 |

### Search Results

```go
type Result struct {
    ID         string // "enwiki/32552979"
    WikiName   string // "enwiki"
    InLanguage string // "en"
    Title      string // "Alan Turing"
}
```

## View Design

### Page Retrieval

Pages can be retrieved by:
- **ID**: `/page?id=enwiki/32552979`
- **Title**: `/page?wiki=enwiki&title=Alan%20Turing`

### Page Content

```go
type Page struct {
    ID           string
    WikiName     string
    PageID       int64
    Title        string
    URL          string
    DateModified string
    InLanguage   string
    Text         string
    WikidataID   string
    BytesHTML    int64
    HasMath      bool
    WikiText     string
    Version      string
    InfoboxesJSON string
}
```

## UI Design

### Layout Structure

```
┌─────────────────────────────────────────────────────────────────┐
│  TOPBAR: Logo | Search Input | Theme Toggle                    │
├─────────────────────────────────────────────────────────────────┤
│        │                                                        │
│ SIDEBAR│                    MAIN CONTENT                        │
│  (TOC) │                                                        │
│        │                                                        │
│        │                                                        │
│        │                                                        │
│        │                                                        │
└────────┴────────────────────────────────────────────────────────┘
```

### Theme Support

- Light theme (default)
- Dark theme (via CSS custom properties)
- Theme persistence via cookie

### Keyboard Shortcuts

- `/` or `Ctrl+K` - Focus search
- `Escape` - Close search/clear focus

### Responsive Design

- Desktop: Sidebar + content side-by-side
- Tablet: Collapsible sidebar
- Mobile: Stacked layout, hamburger menu

## Templates

### Base Layout (app.html)

```html
<!DOCTYPE html>
<html lang="en" data-theme="{{.Theme}}">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{{block "title" .}}FineWiki{{end}}</title>
    <style>{{template "styles" .}}</style>
</head>
<body>
    {{template "topbar.html" .}}
    <main>
        {{template "sidebar.html" .}}
        <article>
            {{block "content" .}}{{end}}
        </article>
    </main>
    {{block "scripts" .}}{{end}}
</body>
</html>
```

### CSS Variables

```css
:root {
    --bg: #ffffff;
    --fg: #1a1a1a;
    --accent: #2563eb;
    --muted: #6b7280;
    --border: #e5e7eb;
    --surface: #f9fafb;
}

[data-theme="dark"] {
    --bg: #0f0f0f;
    --fg: #fafafa;
    --accent: #60a5fa;
    --muted: #9ca3af;
    --border: #27272a;
    --surface: #18181b;
}
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Home page with search |
| GET | `/search?q=...` | Search results page |
| GET | `/page?id=...` | View page by ID |
| GET | `/page?wiki=...&title=...` | View page by wiki/title |
| GET | `/healthz` | Health check endpoint |

## CLI Commands

### Main Server

```bash
finewiki [flags]
```

**Flags:**
- `--addr` / `FINEWIKI_ADDR` - Listen address (default: `:8080`)
- `--db` / `FINEWIKI_DUCKDB` - DuckDB file path (default: `finewiki.duckdb`)
- `--parquet` / `FINEWIKI_PARQUET` - Parquet glob pattern (required)
- `--fts` / `FINEWIKI_FTS` - Enable FTS fallback (default: `false`)

### Import Command

```bash
finewiki import <path-or-url> [flags]
```

**Flags:**
- `--dir` / `FINEWIKI_DATA` - Destination directory (default: `data`)

### List Command

```bash
finewiki list <hf-dataset>
```

Lists available Parquet shard URLs from a Hugging Face dataset.

## Performance Targets

| Metric | Target |
|--------|--------|
| Cold start | < 5 seconds |
| Search latency (p99) | < 50ms |
| Page load latency (p99) | < 100ms |
| Memory usage | < 500MB |

## Error Handling

- Invalid search query → Empty results (graceful degradation)
- Page not found → 404 with helpful message
- Database errors → 500 with generic message (no leak of internals)
- Parquet read errors → Logged, 500 returned

## Security Considerations

- No user input in SQL without parameterization
- HTML content properly escaped (Go templates)
- No file path traversal vulnerabilities
- Rate limiting via reverse proxy (out of scope)

## Testing Strategy

### Unit Tests
- `search.Service` - Query normalization, limit handling
- `view.Service` - ID/title validation
- `duckdb.Store` - SQL query construction

### Integration Tests
- Store operations with test Parquet files
- Handler response validation

### End-to-End Tests
- Full request/response cycles
- Search and view flows

## Future Extensions

These are explicitly out of scope for MVP but the architecture supports them:

1. **Random Page** - `/random` endpoint
2. **Language Filter UI** - Dropdown selector
3. **HTML Rendering** - Process wikitext to HTML
4. **Table of Contents** - Parse headings from text
5. **Related Pages** - Links/see-also section
6. **JSON API** - `/api/search`, `/api/page` endpoints
7. **Multi-Dataset** - Support multiple Parquet sources

## Dependencies

```
github.com/go-mizu/mizu           # Web framework
github.com/marcboeker/go-duckdb   # DuckDB driver
github.com/spf13/cobra            # CLI framework
github.com/charmbracelet/fang     # Cobra helpers
```

## Implementation Checklist

- [x] Project structure and README
- [x] CLI entrypoint with Cobra
- [x] Feature API interfaces (search, view)
- [x] Feature services (search, view)
- [x] DuckDB schema and seed SQL
- [x] Parquet import functionality
- [ ] DuckDB store implementation
- [ ] HTML templates (layout, components, pages)
- [ ] Middleware (logging)
- [ ] Comprehensive tests
- [ ] go.mod dependencies
