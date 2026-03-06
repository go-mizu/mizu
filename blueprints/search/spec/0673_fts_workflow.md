# 0673 — FTS Workflow (`search cc fts`)

Full-text search pipeline: Common Crawl WARC → Markdown → Pack → FTS Index → Dashboard.

## Pipeline Overview

```
WARC Download          Markdown Pack         FTS Index            Dashboard
  cc warc download  →   cc warc pack     →   cc fts index    →   cc fts dashboard
  .warc.gz (1 GB)       .md.warc.gz          fts/{engine}/       localhost:3456
                        warc_md/              per-shard dirs      REST + WebSocket
```

## CLI Commands

| Command | Purpose |
|---------|---------|
| `search cc fts index --file 0 --engine rose` | Build FTS index from markdown/pack |
| `search cc fts search "query" --engine rose` | CLI full-text search (shard fanout) |
| `search cc fts pack --file 0 --format all` | Pre-compute pack formats (parquet, bin, duckdb, markdown) |
| `search cc fts web --port 3456` | Search-only web GUI |
| `search cc fts dashboard --port 3456` | Full admin dashboard with jobs + metadata |
| `search cc fts embed run --file 0` | Compute vector embeddings |
| `search cc fts vector load/search` | Load + query vector stores |

## Directory Layout

```
$HOME/data/common-crawl/{crawlID}/
├── warc/                           # Downloaded .warc.gz files
│   └── CC-MAIN-...-00000.warc.gz
├── warc_md/                        # Markdown WARC (single-pass packed)
│   ├── 00000.md.warc.gz           # WARC-Type=conversion, Content-Type=text/markdown
│   └── 00000.meta.duckdb          # Per-shard doc metadata (DocStore)
├── markdown/                       # Unpacked markdown (legacy, optional)
│   └── 00000/{uuid-path}.md
├── pack/                           # Pre-computed load formats
│   ├── parquet/00000.parquet
│   ├── bin/00000.bin
│   ├── duckdb/00000.duckdb
│   └── markdown/00000.bin.gz
├── fts/                            # FTS indexes, per engine per shard
│   ├── rose/00000/
│   ├── dahlia/00000/
│   ├── duckdb/00000/
│   └── bleve/00000/
├── embed/                          # Vector embeddings
│   └── llamacpp/00000/
│       ├── vectors.bin             # Raw float32 (N x dim x 4)
│       └── meta.jsonl
└── .meta/
    └── dashboard_meta.sqlite       # Metastore cache
```

## Stage 1: WARC Download

`search cc warc download --file 0`

Downloads `.warc.gz` from Common Crawl S3 to `warc/`. Each file ~1 GB, ~1000 files per crawl.

## Stage 2: WARC → Markdown Pack

`search cc warc pack --file 0`

Single-pass pipeline: reads `.warc.gz`, filters HTTP 200 + text/html, converts HTML to
Markdown (go-readability or trafilatura), writes `.md.warc.gz` in `warc_md/`.

Output format: seekable concatenated gzip. Each record is a WARC conversion record:
```
WARC-Type: conversion
WARC-Target-URI: <original URL>
WARC-Date: <crawl timestamp>
WARC-Record-ID: <urn:uuid:...>
WARC-Refers-To: <original record ID>
Content-Type: text/markdown
Content-Length: <body length>

<markdown body>
```

Each record wrapped in its own gzip member for random access.

## Stage 3: Pack (Optional Pre-compute)

`search cc fts pack --file 0 --format parquet`

Converts `markdown/{warcIdx}/` into optimized load formats:

| Format | File | Use Case |
|--------|------|----------|
| parquet | `pack/parquet/00000.parquet` | DuckDB native `read_parquet()` |
| bin | `pack/bin/00000.bin` | Fastest sequential read |
| markdown | `pack/markdown/00000.bin.gz` | Compressed binary |
| duckdb | `pack/duckdb/00000.duckdb` | Direct DuckDB table |

## Stage 4: FTS Index

`search cc fts index --file 0 --engine rose --source files`

Per-WARC shard indexing. For each selected WARC:
1. Create engine instance (registered driver)
2. Open index at `fts/{engine}/{warcIdx}/`
3. Stream documents from markdown dir or pack file
4. Batch insert (default 5000 docs/batch)
5. Call `CreateFTSIndex()` for SQL engines (BM25)

### Registered Engines

| Engine | Type | Notes |
|--------|------|-------|
| rose, dahlia | Embedded Go | High-performance, default |
| duckdb | SQL + FTS5 | BM25, vectorized |
| sqlite | SQL + FTS5 | Lightweight |
| bleve | Pure Go | Standard full-text |
| elasticsearch, opensearch | External | Distributed cluster |
| meilisearch, quickwit | External | Cloud/log search |
| devnull | Benchmark | I/O measurement only |

### Engine Interface

```go
type Engine interface {
    Open(ctx, indexDir) error
    Index(docs []Document) error
    Search(ctx, query) (Results, error)
    Stats(ctx) (DocCount, DiskBytes, ...) error
    Close() error
}

// Optional:
type BulkLoader interface { BulkLoad(ctx, format, path) (int64, error) }
type FTSIndexer interface { CreateFTSIndex(ctx) error }
```

## Stage 5: Dashboard

`search cc fts dashboard --port 3456 --engine rose`

Full admin dashboard with REST API, WebSocket progress, job management.

### API Endpoints

**Search & Browse:**

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/search?q=...&engine=rose&page_size=20` | Full-text search (shard fanout + merge) |
| GET | `/api/doc/{shard}/{docid}` | Fetch document: markdown + HTML render |
| GET | `/api/browse?shard={idx}&page=1` | Paginated doc list from DocStore |
| GET | `/api/browse/stats?shard={idx}` | Shard aggregation stats (domains, sizes, dates) |
| GET | `/api/stats` | Index statistics |

**WARC Management:**

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/warc` | List all WARCs with sizes, status chips, doc counts |
| GET | `/api/warc/{index}` | Detail for single WARC (pack/fts/md sizes) |

**Dashboard Admin:**

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/overview` | Crawl summary + metadata status |
| GET | `/api/crawls` | List all crawls |
| POST | `/api/jobs` | Create job (download, markdown, pack, index) |
| GET | `/api/jobs` | List running/completed jobs |
| DELETE | `/api/jobs/{id}` | Cancel a job |
| POST | `/api/meta/scan-docs` | Trigger DocStore scan |
| POST | `/api/meta/refresh` | Trigger metadata refresh |
| GET | `/ws` | WebSocket for real-time progress |

### Search Enrichment

Search results are enriched with per-document metadata from DocStore:

```json
{
  "doc_id": "f9f0dea7-...",
  "shard": "00000",
  "score": 13.27,
  "snippet": "...",
  "url": "https://example.com/article",
  "title": "Example Article",
  "crawl_date": "2026-02-06T21:04:17Z",
  "size_bytes": 108204,
  "word_count": 21640
}
```

Enrichment runs in parallel goroutines after pagination, using in-memory cache.

## Metadata Layer

### MetaManager (Crawl-level)

Caches WARC file metadata: sizes, record counts, pack/FTS status.

- Drivers: SQLite (default), DuckDB, none (scan-fallback)
- Storage: `$HOME/data/common-crawl/.meta/dashboard_meta.sqlite`
- Background refresh with configurable TTL (default 30s)
- Prewarm on startup: scans active crawl directory

### DocStore (Per-shard Document Metadata)

Per-document metadata extracted from `.md.warc.gz` WARC headers.

- Storage: one DuckDB per shard at `warc_md/{shard}.meta.duckdb`
- In-memory cache: all records loaded on first access via `warmShard()`
- Bulk insert via DuckDB Appender (avoids SQL parameter binding issues)
- Cache invalidated after scan; next access re-warms from DuckDB

**Schema:**
```sql
CREATE TABLE doc_records (
    doc_id TEXT PRIMARY KEY, url TEXT, title TEXT,
    crawl_date TEXT, size_bytes BIGINT, word_count INTEGER,
    warc_record_id TEXT, refers_to TEXT, scanned_at TEXT
);
CREATE TABLE doc_scan_meta (
    id INTEGER PRIMARY KEY, total_docs BIGINT,
    total_size_bytes BIGINT, last_doc_date TEXT, last_scanned_at TEXT
);
```

**Scan triggers:**
1. Manual: `POST /api/meta/scan-docs`
2. Post-job: after pack/index jobs complete (via `Jobs.SetCompleteHook`)
3. Auto-refresh: when browse API detects stale metadata (>1h)

**Stats aggregation** (opened read-only, not cached):
- Top 20 domains by doc count (`regexp_extract`)
- Size distribution buckets (<1KB, 1-5KB, 5-20KB, 20-100KB, >100KB)
- Date histogram (docs per day)
- Summary: total docs, total/avg/min/max size, date range

## Vector Pipeline (Optional)

### Embed: `search cc fts embed run`

4-stage streaming pipeline:
1. Reader → collect `.md` files
2. Batcher → group by token budget
3. Embed workers → parallel inference (llamacpp HTTP or ONNX local)
4. Writer → `vectors.bin` (raw float32) + `meta.jsonl`

Drivers: llamacpp (HTTP to llama.cpp server), onnx (local ONNX Runtime).

### Vector Store: `search cc fts vector load/search`

Load embeddings into vector stores (qdrant, pgvector, elasticsearch, etc.),
then search via query embedding → cosine similarity → top-K results.

## Performance

| Stage | Throughput | Bottleneck |
|-------|-----------|------------|
| Download | Network (100+ Mbps) | S3 rate limit |
| WARC → Markdown | 200-600 docs/s (default), 800-2K/s (fast) | HTML conversion |
| Pack | ~1K docs/s | Gzip compression |
| FTS Index | 1-100K docs/s | Engine (rose: 50K/s, duckdb: 10K/s) |
| Search | 1-10ms per query | Shard fanout parallelism |
| DocStore scan | ~350 docs/s | WARC parsing + DuckDB append |
| DocStore lookup | O(1) | In-memory map (after warm) |

## Key Lessons

- **DuckDB Appender vs PrepareContext**: DuckDB Go v2 `tx.PrepareContext` + `stmt.ExecContext`
  fails with "could not bind parameter". Use `duckdb.NewAppenderFromConn` via `conn.Raw()`
  for bulk inserts — bypasses SQL parameter binding entirely.
- **DuckDB MaxOpenConns=1**: Must close `sql.Conn` before using `db.ExecContext` on the same
  `*sql.DB` with `SetMaxOpenConns(1)`, otherwise blocks forever waiting for the connection.
- **Doc ID consistency**: FTS index doc_ids come from WARC-Record-ID (UUID) generated during
  packing. Different pack runs produce different UUIDs. Search enrichment only works when FTS
  index and DocStore are from the same `.md.warc.gz`.
