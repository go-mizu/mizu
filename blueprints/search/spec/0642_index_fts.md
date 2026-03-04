# spec/0642 — Full-Text Search Index for Common Crawl Markdown

## Goal

Index 154,990 `.md`/`.md.gz` files from `$HOME/data/common-crawl/CC-MAIN-2026-08/markdown/`
into a pluggable FTS engine and provide keyword search with ranked results.

## CLI

```
search cc fts index --engine duckdb|sqlite|chdb|devnull
    [--crawl CC-MAIN-2026-08]   # default: latest crawl
    [--batch-size 5000]          # docs per transaction
    [--workers 4]                # parallel file readers

search cc fts "keyword" --engine duckdb
    [--limit 10] [--offset 0]
    [--crawl CC-MAIN-2026-08]
```

## Data

### Source

`$HOME/data/common-crawl/{crawl}/markdown/` — 3-level hex-sharded directory tree.
Files are either `.md` (plain) or `.md.gz` (gzip-compressed).
Reader detects format by extension.

### Document Schema

| Field | Type | Source |
|-------|------|--------|
| DocID | string | UUID filename (minus extension) |
| Text  | string | full decompressed markdown body |

Uses existing `pkg/index.Document{DocID, Text}` unchanged.

### Output

`$HOME/data/common-crawl/{crawl}/fts/{driver}/` — each driver owns its subdirectory.

| Driver | Files |
|--------|-------|
| devnull | (empty) |
| duckdb | `fts.duckdb` |
| sqlite | `fts.db` |
| chdb | `chdb_data/` (directory) |

---

## Architecture

### Driver Interface

Extends `pkg/index/index.go` with a concrete engine wrapper:

```go
// pkg/index/engine.go
type Engine interface {
    Name() string
    Open(ctx context.Context, dir string) error   // open or create at path
    Close() error
    Stats() (EngineStats, error)

    // Indexing
    Index(ctx context.Context, docs []Document) error

    // Searching
    Search(ctx context.Context, q Query) (Results, error)
}

type EngineStats struct {
    DocCount  int64
    DiskBytes int64
}
```

### Driver Registry

```go
// pkg/index/registry.go
var registry = map[string]func() Engine{}

func Register(name string, factory func() Engine)
func Open(name string) (Engine, error)
func List() []string
```

Each driver self-registers in `init()`.

### Drivers

#### 1. devnull (`pkg/index/driver/devnull/`)

No-op implementation. `Index()` discards documents, `Search()` returns empty.
Purpose: benchmark I/O + decompression overhead without any DB cost.

#### 2. duckdb (`pkg/index/driver/duckdb/`)

**Index creation:**
```sql
CREATE TABLE documents (
    doc_id VARCHAR PRIMARY KEY,
    text   VARCHAR
);

-- After all inserts:
INSTALL fts; LOAD fts;
PRAGMA create_fts_index('documents', 'doc_id', 'text',
    stemmer='english', stopwords='english', lower=1);
```

**Batch insert:**
```sql
INSERT INTO documents (doc_id, text) VALUES (?, ?)
-- batch_size rows per transaction
```

**Search:**
```sql
SELECT doc_id, text, fts_main_documents.match_bm25(doc_id, ?, fields := 'text') AS score
FROM documents
WHERE score IS NOT NULL
ORDER BY score DESC
LIMIT ? OFFSET ?;
```

**Scoring:** BM25 via DuckDB FTS extension.

#### 3. sqlite (`pkg/index/driver/sqlite/`)

**Schema:**
```sql
CREATE TABLE IF NOT EXISTS documents (
    doc_id TEXT PRIMARY KEY,
    text   TEXT
);

CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(
    text,
    content='documents',
    content_rowid='rowid',
    tokenize='unicode61 remove_diacritics 0'
);

-- Sync triggers
CREATE TRIGGER IF NOT EXISTS documents_ai AFTER INSERT ON documents BEGIN
    INSERT INTO documents_fts(rowid, text) VALUES (new.rowid, new.text);
END;
CREATE TRIGGER IF NOT EXISTS documents_ad AFTER DELETE ON documents BEGIN
    INSERT INTO documents_fts(documents_fts, rowid, text) VALUES ('delete', old.rowid, old.text);
END;
```

**Search:**
```sql
SELECT d.doc_id, snippet(f, 0, '', '', '...', 40) AS snippet,
       bm25(f) AS score
FROM documents_fts f
JOIN documents d ON d.rowid = f.rowid
WHERE f.text MATCH ?
ORDER BY bm25(f)
LIMIT ? OFFSET ?;
```

**Scoring:** FTS5 native BM25. SQLite FTS5 `bm25()` returns negative values (lower = better).
Negate for display: `score = -bm25(f)`.

**Connection flags:** `?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000`

#### 4. chdb (`pkg/index/driver/chdb/`)

chdb embeds ClickHouse v25.8 kernel. The newest FTS (`TYPE text` + `searchAny`/`searchAll`)
requires ClickHouse 25.9+. We use the older `inverted()` index which is stable on 25.8.

When chdb upgrades to 25.9+, switch to `TYPE text(tokenizer='default')` + `searchAny`/`searchAll`.

Ref: https://clickhouse.com/blog/clickhouse-full-text-search
Ref: https://clickhouse.com/blog/chdb-kernel-update-25.8

**Schema:**
```sql
CREATE TABLE IF NOT EXISTS documents (
    doc_id String,
    text   String,
    INDEX text_idx text TYPE inverted()
) ENGINE = MergeTree() ORDER BY doc_id
SETTINGS index_granularity = 8192;
```

**Batch insert:**
```sql
INSERT INTO documents (doc_id, text) VALUES (?, ?)
-- batch_size rows per transaction
```

**Search:**
```sql
SELECT doc_id, substring(text, 1, 200) AS snippet
FROM documents
WHERE hasAllTokens(lower(text), lower(?))
ORDER BY length(text) ASC
LIMIT ? OFFSET ?;
```

**Scoring:** No BM25. ClickHouse inverted indexes support token matching only.
Score = 1.0 for all matches (unranked). Order by text length as rough relevance proxy.

**Build:** Requires `libchdb` installed (`curl -sL https://lib.chdb.io | bash`).
CGO_ENABLED=1. Build-tagged `//go:build chdb`.

---

## Index Pipeline

Three-stage pipeline connected by channels:

```
[Walker] ──fileCh(1000)──▶ [N Readers] ──docCh(5000)──▶ [Batcher → Engine.Index()]
```

### Stage 1: Walker (1 goroutine)

- `filepath.WalkDir` over `markdown/`
- Filters: `.md` or `.md.gz` files only
- Sends file paths to `fileCh`
- Counts total files for progress denominator

### Stage 2: Readers (N goroutines, default 4)

- Reads file from path
- If `.md.gz`: gzip decompress; if `.md`: read raw
- Extracts UUID from filename (strip `.md.gz` or `.md`)
- Sends `Document{DocID: uuid, Text: content}` to `docCh`
- Skips empty files silently

### Stage 3: Batcher (1 goroutine)

- Collects `batch_size` documents from `docCh`
- Calls `engine.Index(ctx, batch)`
- Updates progress counter
- Single goroutine to avoid DB write contention

### Progress Display

Live-updating single line (stderr), 200ms refresh:

```
indexing [duckdb] ████████░░ 77,500/154,990 docs │ 3,412 docs/s │ 22.7s │ RSS 245 MB │ disk 892 MB
```

Fields:
- Engine name
- Progress bar (20 chars)
- docs indexed / total
- throughput (docs/s, smoothed)
- elapsed time
- RSS memory (`runtime.ReadMemStats`)
- disk usage of output dir (`du -s` equivalent)

### Final Summary

```
── FTS Index Complete ──────────────────────────
  engine:    duckdb
  docs:      154,990
  elapsed:   45.3s
  avg rate:  3,420 docs/s
  peak RSS:  312 MB
  disk:      1.24 GB
  path:      ~/data/common-crawl/CC-MAIN-2026-08/fts/duckdb/
```

---

## Search Output

Table format:

```
── Results for "machine learning" (engine: duckdb, 847 total) ──
  #  Score    DocID                                    Snippet
  1  12.34    a1b2c3d4-e5f6-...                       ...deep machine learning algorithms...
  2  11.87    f7e8d9c0-b1a2-...                       ...introduction to machine learning...
  3  10.22    12345678-abcd-...                        ...supervised machine learning model...
```

---

## Benchmark Plan

Run all 4 drivers on the same 154,990-file dataset on server2.

### Index Benchmark

| Metric | devnull | duckdb | sqlite | chdb |
|--------|---------|--------|--------|------|
| time (s) | — | — | — | — |
| docs/s | — | — | — | — |
| peak RSS (MB) | — | — | — | — |
| disk (MB) | 0 | — | — | — |

### Search Benchmark

10 queries, each run 3× warm:

```
"machine learning"
"climate change"
"artificial intelligence"
"United States"
"open source software"
"COVID-19 pandemic"
"data privacy"
"renewable energy"
"blockchain technology"
"neural network"
```

| Metric | duckdb | sqlite | chdb |
|--------|--------|--------|------|
| avg latency (ms) | — | — | — |
| p99 latency (ms) | — | — | — |
| avg hits | — | — | — |

---

## File Layout

```
pkg/index/
├── index.go          # existing interfaces (Document, Query, Results, etc.)
├── engine.go         # Engine interface, EngineStats, registry
├── driver/
│   ├── devnull/
│   │   └── devnull.go
│   ├── duckdb/
│   │   └── duckdb.go
│   ├── sqlite/
│   │   └── sqlite.go
│   └── chdb/
│       └── chdb.go   # build tag: //go:build chdb
└── pipeline.go       # Walker + Reader + Batcher pipeline

cli/
├── cc_fts.go         # "search cc fts" subcommands (index + search)
```

---

## Build

```makefile
# Default build (devnull + duckdb + sqlite)
make build

# With chdb support
make build TAGS=chdb

# Linux cross-compile (chdb excluded by default)
make build-linux-noble
make build-linux-noble TAGS=chdb   # if libchdb installed in container
```

---

## Implementation Order

1. `pkg/index/engine.go` — Engine interface + registry
2. `pkg/index/driver/devnull/` — no-op driver
3. `pkg/index/pipeline.go` — file walker + reader + batcher
4. `cli/cc_fts.go` — CLI commands (index + search)
5. `pkg/index/driver/duckdb/` — DuckDB FTS driver
6. `pkg/index/driver/sqlite/` — SQLite FTS5 driver
7. `pkg/index/driver/chdb/` — chdb driver (build-tagged)
8. Benchmark on server2
9. Fill in benchmark table in this spec
