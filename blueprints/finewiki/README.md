# FineWiki

**Read Wikipedia offline, blazingly fast.**

FineWiki is a self-hosted Wikipedia viewer that serves articles directly from Parquet files using DuckDB. No database setup, no ETL pipelines—just download and run.

```
┌─────────────────────────────────────────────────────────────────┐
│  Parquet Files (HuggingFace)  →  DuckDB Index  →  Web Server   │
│         ~800MB per language         instant search    SSR HTML │
└─────────────────────────────────────────────────────────────────┘
```

## Features

- **325 languages** available from the FineWiki dataset
- **Instant title search** with prefix matching and fuzzy fallback
- **Server-side rendering** - no JavaScript frameworks, works everywhere
- **Single binary** - zero runtime dependencies
- **Offline-first** - download once, read forever

---

## Quick Start

Get Wikipedia running locally in under 2 minutes:

```bash
# 1. Build the CLI
make build

# 2. Download Vietnamese Wikipedia (~800 MB)
finewiki import vi

# 3. Start the server
finewiki serve vi

# 4. Open http://localhost:8080
```

That's it! Your local Wikipedia is ready.

---

## Commands Reference

### `finewiki import <lang>`

Download Wikipedia data for a specific language from HuggingFace.

```bash
# Basic usage
finewiki import vi        # Vietnamese (~800 MB)
finewiki import en        # English (~15 GB)
finewiki import ja        # Japanese
finewiki import de        # German

# Custom data directory
finewiki import vi --data ~/my-wiki-data
```

| Flag | Default | Description |
|------|---------|-------------|
| `--data` | `$HOME/data/blueprint/finewiki` | Base directory for all data |

**What happens:**
1. Fetches Parquet file from [HuggingFace FineWiki dataset](https://huggingface.co/datasets/HuggingFaceFW/finewiki)
2. Saves to `<data>/<lang>/data.parquet`
3. Shows download progress

**For private/rate-limited access:**
```bash
export HF_TOKEN=your_huggingface_token
finewiki import en
```

---

### `finewiki serve <lang>`

Start the web server for a specific language.

```bash
# Basic usage (serves on :8080)
finewiki serve vi

# Custom port
finewiki serve en --addr :3000

# Custom data directory
finewiki serve ja --data ~/my-wiki-data
```

| Flag | Default | Description |
|------|---------|-------------|
| `--addr` | `:8080` | HTTP listen address (host:port) |
| `--data` | `$HOME/data/blueprint/finewiki` | Base directory for data |

**What happens on first run:**
1. Reads `<data>/<lang>/data.parquet`
2. Creates DuckDB index at `<data>/<lang>/wiki.duckdb`
3. Builds search indexes (takes ~30s for Vietnamese, longer for English)
4. Starts HTTP server

**Subsequent runs:** Index is reused, starts in under 100ms.

---

### `finewiki list`

Discover available languages or check what you've installed.

```bash
# Show all 325 available languages
finewiki list

# Show only languages you've downloaded
finewiki list --installed
```

| Flag | Default | Description |
|------|---------|-------------|
| `--installed` | `false` | Only show locally installed languages |
| `--data` | `$HOME/data/blueprint/finewiki` | Base directory to check |

**Example output:**
```
Available languages (325):

  en          vi          ja          de          fr          es
  zh          ko          ru          pt          it          pl
  ...
```

---

## Database Schema

FineWiki uses DuckDB with three tables. Here's the complete schema:

### `titles` - Fast Search Index

Lightweight table optimized for instant title search.

```sql
CREATE TABLE titles (
  id          VARCHAR PRIMARY KEY,  -- Unique article identifier
  wikiname    VARCHAR NOT NULL,     -- Wiki name (e.g., "viwiki", "enwiki")
  in_language VARCHAR NOT NULL,     -- Language code (e.g., "vi", "en")
  title       VARCHAR NOT NULL,     -- Original article title
  title_lc    VARCHAR NOT NULL      -- Lowercase title for case-insensitive search
);
```

| Column | Type | Example | Description |
|--------|------|---------|-------------|
| `id` | VARCHAR | `"viwiki_12345"` | Unique identifier combining wiki + page_id |
| `wikiname` | VARCHAR | `"viwiki"` | Which wiki this belongs to |
| `in_language` | VARCHAR | `"vi"` | ISO language code |
| `title` | VARCHAR | `"Việt Nam"` | Display title with original casing |
| `title_lc` | VARCHAR | `"việt nam"` | Lowercase for prefix search |

**Indexes:**
```sql
CREATE INDEX idx_titles_title_lc ON titles(title_lc);   -- Fast prefix search
CREATE INDEX idx_titles_wikiname ON titles(wikiname);   -- Filter by wiki
CREATE INDEX idx_titles_lang ON titles(in_language);    -- Filter by language
```

---

### `pages` - Full Article Content

Complete article data including text, wikitext, and metadata.

```sql
CREATE TABLE pages (
  id            VARCHAR PRIMARY KEY,  -- Same as titles.id
  wikiname      VARCHAR NOT NULL,     -- Wiki name
  page_id       BIGINT NOT NULL,      -- Wikipedia's internal page ID
  title         VARCHAR NOT NULL,     -- Article title
  title_lc      VARCHAR NOT NULL,     -- Lowercase title
  url           VARCHAR NOT NULL,     -- Original Wikipedia URL
  date_modified VARCHAR,              -- Last modification date
  in_language   VARCHAR NOT NULL,     -- Language code
  text          VARCHAR,              -- Plain text content
  wikidata_id   VARCHAR,              -- Wikidata entity ID (Q-number)
  bytes_html    BIGINT,               -- Size of rendered HTML in bytes
  has_math      BOOLEAN,              -- Contains mathematical formulas
  wikitext      VARCHAR,              -- Original MediaWiki markup
  version       VARCHAR,              -- Dataset version
  infoboxes     VARCHAR               -- JSON array of infobox data
);
```

| Column | Type | Example | Description |
|--------|------|---------|-------------|
| `id` | VARCHAR | `"viwiki_12345"` | Primary key, links to titles |
| `wikiname` | VARCHAR | `"viwiki"` | Wiki identifier |
| `page_id` | BIGINT | `12345` | Wikipedia's page ID |
| `title` | VARCHAR | `"Việt Nam"` | Display title |
| `title_lc` | VARCHAR | `"việt nam"` | For case-insensitive lookup |
| `url` | VARCHAR | `"https://vi.wikipedia.org/wiki/Việt_Nam"` | Source URL |
| `date_modified` | VARCHAR | `"2024-01-15"` | Last edit date |
| `in_language` | VARCHAR | `"vi"` | ISO language code |
| `text` | VARCHAR | `"Việt Nam, tên..."` | Cleaned plain text |
| `wikidata_id` | VARCHAR | `"Q881"` | Links to Wikidata |
| `bytes_html` | BIGINT | `156789` | HTML size metric |
| `has_math` | BOOLEAN | `false` | Has LaTeX math |
| `wikitext` | VARCHAR | `"{{Infobox..."` | Original wiki markup |
| `version` | VARCHAR | `"2024.01"` | FineWiki version |
| `infoboxes` | VARCHAR | `"[{...}]"` | Parsed infobox JSON |

**Indexes:**
```sql
CREATE INDEX idx_pages_title_lc ON pages(title_lc);              -- Title lookup
CREATE INDEX idx_pages_wikiname ON pages(wikiname);              -- Filter by wiki
CREATE INDEX idx_pages_wikiname_title ON pages(wikiname, title_lc); -- Combined lookup
```

---

### `meta` - Store Metadata

Tracks database state for cache invalidation and updates.

```sql
CREATE TABLE meta (
  k VARCHAR PRIMARY KEY,  -- Key name
  v VARCHAR NOT NULL      -- Value
);
```

| Key | Example Value | Description |
|-----|---------------|-------------|
| `seeded_at` | `"2024-01-15 10:30:00"` | When data was imported |
| `parquet_count` | `"1500000"` | Expected row count for validation |
| `parquet_glob` | `"/path/to/data.parquet"` | Source file path |

Used internally to detect when re-seeding is needed (e.g., parquet file changed).

---

## Data Directory Structure

```
$HOME/data/blueprint/finewiki/
├── vi/
│   ├── data.parquet    # Vietnamese Wikipedia articles (~800 MB)
│   └── wiki.duckdb     # Title search index (~100 MB)
├── en/
│   ├── data.parquet    # English Wikipedia (~15 GB)
│   └── wiki.duckdb     # Index (~2 GB)
└── ja/
    ├── data.parquet
    └── wiki.duckdb
```

Each language is completely isolated—you can add/remove languages independently.

---

## Architecture

```
finewiki/
├── cmd/finewiki/        # CLI entry point
├── cli/                 # Command implementations
│   ├── serve.go         #   → finewiki serve
│   ├── import.go        #   → finewiki import
│   ├── list.go          #   → finewiki list
│   └── views/           #   → HTML templates
├── app/web/             # HTTP handlers & routing
├── feature/
│   ├── search/          # Title search service
│   └── view/            # Page view service
└── store/duckdb/        # DuckDB storage layer
    ├── schema.sql       #   → Table definitions
    ├── seed.sql         #   → Data import queries
    └── store.go         #   → Go interface
```

### Design Philosophy

| Principle | What it means |
|-----------|---------------|
| **Parquet as source** | No data transformation—Parquet files are the truth |
| **Title-only index** | DuckDB indexes titles for instant search; full pages read from Parquet |
| **Per-language isolation** | Each language is independent, no shared state |
| **Server-side rendering** | HTML generated on server, no client-side JS required |
| **Single binary** | Everything embedded, `go build` produces one executable |

---

## Development

```bash
# Run server with auto-rebuild (requires air or similar)
make run ARGS="serve vi"

# Run tests
make test

# Build binary to $HOME/bin
make build

# Clean all downloaded data
make clean-data
```

### Running End-to-End Tests

```bash
# E2E tests require the E2E_TEST flag
E2E_TEST=1 go test ./store/duckdb/...
```

---

## System Requirements

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| Go version | 1.22+ | Latest |
| CGO | Enabled | - |
| RAM | 512 MB | 2 GB+ |
| Disk | Per language | See below |

**Disk space by language (approximate):**
- Vietnamese: ~1 GB
- Japanese: ~3 GB
- German: ~5 GB
- English: ~18 GB

---

## Performance

| Metric | Value |
|--------|-------|
| Cold start (first run) | 1-3 seconds |
| Warm start | < 100 ms |
| Search latency | < 10 ms |
| Memory usage | ~50 MB + index |

---

## Troubleshooting

### "CGO disabled" error
DuckDB requires CGO. Enable it:
```bash
export CGO_ENABLED=1
go build ./cmd/finewiki
```

### "HuggingFace rate limited"
Set your token:
```bash
export HF_TOKEN=hf_xxxxxxxxxxxxx
finewiki import en
```

### "Index out of date" warning
The server auto-reseeds if the parquet file changes. This is normal.

### Slow first start
First run builds the search index from Parquet. Subsequent starts are fast.

---

## Dataset

FineWiki uses the [FineWiki dataset](https://huggingface.co/datasets/HuggingFaceFW/finewiki) from HuggingFace—cleaned Wikipedia dumps in Parquet format covering **325 languages**.

---

## License

MIT
