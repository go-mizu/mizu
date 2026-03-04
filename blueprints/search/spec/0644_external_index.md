# spec/0644 — External FTS Index Drivers

**Status:** implemented (embedded engines benchmarked; external engines require Docker)
**Branch:** index-pane

---

## Goal

Extend `pkg/index` with six new FTS drivers — two embedded (bleve, tantivy-go CGO) and
four external HTTP/TCP services (meilisearch, clickhouse, postgres, quickwit) — plus a
seventh driver for Tantivy via the lnx REST server.  Each external engine gets its own
`docker/{engine}/docker-compose.yml` with data mounted to `$HOME/data/fts/{engine}/`.
All engines are benchmarked on the 173,720-document CC-MAIN-2026-08 corpus.

---

## Driver List

| Name | Type | Transport | New dep |
|------|------|-----------|---------|
| `bleve` | embedded | pure Go | none (already in go.mod) |
| `tantivy-go` | embedded | CGO | `anyproto/tantivy-go` (build-tagged) |
| `meilisearch` | external | HTTP | none (already in go.mod) |
| `clickhouse` | external | native TCP | `ClickHouse/clickhouse-go/v2` |
| `postgres` | external | TCP | none (`jackc/pgx/v5` already in go.mod) |
| `quickwit` | external | HTTP REST | custom client (no SDK needed) |
| `tantivy-lnx` | external | HTTP REST | custom client |

---

## Interface Change: `AddrSetter`

No changes to the core `Engine` interface.  One optional interface is added:

```go
// pkg/index/engine.go

// AddrSetter is implemented by external engines that connect to a remote service.
// The CLI calls SetAddr before Open when --addr is provided.
type AddrSetter interface {
    SetAddr(addr string)
}
```

External drivers embed a `baseExternal` helper:

```go
type baseExternal struct{ addr string }
func (b *baseExternal) SetAddr(a string) { b.addr = a }
func (b *baseExternal) effectiveAddr(def string) string {
    if b.addr != "" { return b.addr }
    return def
}
```

Default addresses per driver:

| Driver | Default addr |
|--------|-------------|
| meilisearch | `http://localhost:7700` |
| clickhouse | `localhost:9000` |
| postgres | `postgres://fineweb:fineweb@localhost:5432/fts` |
| quickwit | `http://localhost:7280` |
| tantivy-lnx | `http://localhost:8000` |

---

## CLI Changes

`search cc fts index` and `search cc fts search` gain a new flag:

```
--addr string   Service address for external engines.
                Ignored for embedded engines (bleve, tantivy-go, duckdb, sqlite, chdb).
                Engine-specific default used when empty.
```

The CLI checks `engine.(index.AddrSetter)` and calls `SetAddr(addr)` before `Open()`.

Updated example block in `cc_fts.go`:

```
search cc fts index --engine meilisearch
search cc fts index --engine meilisearch --addr http://my-server:7700
search cc fts index --engine clickhouse   --addr my-ch-host:9000
search cc fts index --engine postgres     --addr "postgres://user:pass@host:5432/fts"
search cc fts index --engine quickwit     --addr http://localhost:7280
search cc fts index --engine tantivy-lnx  --addr http://localhost:8000
search cc fts index --engine bleve
search cc fts index --engine tantivy-go
```

Driver import side-effect registrations in `cli/cc_fts.go`:

```go
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/bleve"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/meilisearch"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/clickhouse"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/postgres"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/quickwit"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/tantivy-lnx"
// tantivy-go: build-tagged, imported only with //go:build tantivy
```

---

## Docker Compose Layout

```
docker/
  meilisearch/
    docker-compose.yml          # Meilisearch v1.x
  clickhouse/
    docker-compose.yml          # ClickHouse 25.x
  postgres-fts/
    docker-compose.yml          # PostgreSQL 17 + tsvector/GIN
  quickwit/
    docker-compose.yml          # Quickwit 0.9.x
  lnx/
    docker-compose.yml          # lnx (Tantivy REST) latest
```

Data mount convention (all services):
```
${FTS_DATA_DIR:-${HOME}/data/fts}/{engine}/:/var/{engine}/data
```

---

## Driver Specifications

### 1. bleve (`pkg/index/driver/bleve/`)

**Type:** embedded, pure Go, no Docker
**Build tag:** none (always included)
**Index type:** BM25 via bleve's default `en` text analysis

Schema:
```go
mapping := bleve.NewIndexMapping()
docMapping := bleve.NewDocumentMapping()
textField := mapping.NewTextFieldMapping()
textField.Analyzer = "en"
docMapping.AddFieldMappingsAt("text", textField)
mapping.AddDocumentMapping("doc", docMapping)
```

Open:
```go
idx, err = bleve.Open(filepath.Join(dir, "bleve.db"))
if os.IsNotExist(err) {
    idx, err = bleve.New(filepath.Join(dir, "bleve.db"), mapping)
}
```

Index (batch):
```go
b := idx.NewBatch()
for _, doc := range docs {
    b.Index(doc.DocID, map[string]string{"text": string(doc.Text)})
}
idx.Batch(b)
```

Search:
```go
q := bleve.NewMatchQuery(queryText)
q.FieldVal = "text"
req := bleve.NewSearchRequestOptions(q, limit, offset, false)
req.Fields = []string{"text"}
req.Highlight = bleve.NewHighlight()
idx.Search(req)
```

Stats:
```go
count, _ := idx.DocCount()
disk = index.DirSizeBytes(dir)
```

---

### 2. tantivy-go (`pkg/index/driver/tantivy-go/`)

**Type:** embedded, CGO, build-tagged
**Build tag:** `//go:build tantivy`
**Dep:** `github.com/anyproto/tantivy-go` (requires Rust toolchain at build time)
**Docker:** none

Schema:
```go
sb := tantivy.NewSchemaBuilder()
sb.AddTextField("doc_id", tantivy.IndexRecordOptionWithFreqsAndPositions, tantivy.Stored)
sb.AddTextField("text",   tantivy.IndexRecordOptionWithFreqsAndPositions, tantivy.NotStored)
schema, _ := sb.Build()
idx, _ = tantivy.NewIndexWithPath(schema, filepath.Join(dir, "tantivy"))
```

Batch index:
```go
writer, _ := idx.Writer(50_000_000) // 50MB heap
for _, doc := range docs {
    d, _ := idx.ParseDocument(fmt.Sprintf(`{"doc_id":%q,"text":%q}`, doc.DocID, doc.Text))
    writer.AddDocument(d)
}
writer.Commit()
```

Search:
```go
searcher, _ := idx.Searcher()
query, _ := idx.ParseQuery(queryText, []string{"text"})
result, _ := searcher.Search(query, uint32(limit), true, "text", uint32(offset))
```

Stats:
```go
count, _ := idx.NumDocs()
```

Note: `tantivy-go` registers under name `"tantivy"` (not `"tantivy-go"`).
`tantivy-lnx` registers under name `"tantivy-lnx"`.

---

### 3. meilisearch (`pkg/index/driver/meilisearch/`)

**Type:** external HTTP, `github.com/meilisearch/meilisearch-go`
**Default addr:** `http://localhost:7700`
**API key:** read from `MEILISEARCH_API_KEY` env (default `""` = masterKey not set)

Init/Open:
```go
client := meilisearch.New(addr)
// create or get index "fts_docs"
_, err = client.GetIndex("fts_docs")
if err != nil {
    task, _ := client.CreateIndex(&meilisearch.IndexConfig{
        Uid:        "fts_docs",
        PrimaryKey: "doc_id",
    })
    client.WaitForTask(task.TaskUID)
}
// configure searchable attributes
task, _ = client.Index("fts_docs").UpdateSearchableAttributes(&[]string{"text"})
client.WaitForTask(task.TaskUID)
```

Batch index:
```go
docs := make([]map[string]any, len(batch))
for i, d := range batch {
    docs[i] = map[string]any{"doc_id": d.DocID, "text": string(d.Text)}
}
task, _ := client.Index("fts_docs").AddDocuments(docs, "doc_id")
// for throughput: batch without waiting; wait only at Close()
```

Search:
```go
res, _ := client.Index("fts_docs").Search(queryText, &meilisearch.SearchRequest{
    Limit:         int64(limit),
    Offset:        int64(offset),
    AttributesToHighlight: []string{"text"},
})
```

Stats:
```go
stats, _ := client.Index("fts_docs").GetStats()
// DocCount = stats.NumberOfDocuments
// DiskBytes: not exposed by Meilisearch API; use dir size of mounted volume
```

---

### 4. clickhouse (`pkg/index/driver/clickhouse/`)

**Type:** external native TCP
**Dep:** `github.com/ClickHouse/clickhouse-go/v2`
**Default addr:** `localhost:9000`

Schema (inverted token index):
```sql
CREATE TABLE IF NOT EXISTS fts_docs (
    doc_id String,
    text   String,
    INDEX  text_idx text TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1
) ENGINE = MergeTree()
ORDER BY doc_id
SETTINGS index_granularity = 8192;
```

Batch index (native protocol batch):
```go
batch, _ := conn.PrepareBatch(ctx, "INSERT INTO fts_docs (doc_id, text)")
for _, doc := range docs {
    batch.Append(doc.DocID, string(doc.Text))
}
batch.Send()
```

Search (uses token bloom filter for fast pre-filter, then LIKE):
```sql
SELECT doc_id, substring(text, 1, 200) AS snippet, 1.0 AS score
FROM fts_docs
WHERE hasTokenCaseInsensitive(text, ?)
ORDER BY doc_id
LIMIT ? OFFSET ?
```

Stats:
```sql
SELECT count() FROM fts_docs
SELECT sum(data_compressed_bytes) FROM system.parts WHERE table='fts_docs' AND active
```

---

### 5. postgres (`pkg/index/driver/postgres/`)

**Type:** external TCP (pgx/v5)
**Default addr:** `postgres://fineweb:fineweb@localhost:5432/fts`

Schema (tsvector + GIN — native FTS):
```sql
CREATE TABLE IF NOT EXISTS fts_docs (
    doc_id TEXT PRIMARY KEY,
    text   TEXT,
    tsv    TSVECTOR GENERATED ALWAYS AS (to_tsvector('english', text)) STORED
);
CREATE INDEX IF NOT EXISTS fts_docs_tsv_idx ON fts_docs USING GIN(tsv);
```

Batch insert (COPY protocol for max throughput):
```go
rows := make([][]any, len(docs))
for i, d := range docs { rows[i] = []any{d.DocID, string(d.Text)} }
conn.CopyFrom(ctx, pgx.Identifier{"fts_docs"}, []string{"doc_id", "text"}, pgx.CopyFromRows(rows))
```

Search:
```sql
SELECT doc_id,
       ts_headline('english', text, websearch_to_tsquery('english', $1),
                   'MaxFragments=1,MaxWords=20') AS snippet,
       ts_rank_cd(tsv, websearch_to_tsquery('english', $1)) AS score
FROM fts_docs
WHERE tsv @@ websearch_to_tsquery('english', $1)
ORDER BY score DESC
LIMIT $2 OFFSET $3
```

Stats:
```sql
SELECT count(*) FROM fts_docs
SELECT pg_total_relation_size('fts_docs')
```

---

### 6. quickwit (`pkg/index/driver/quickwit/`)

**Type:** external HTTP REST (custom client)
**Default addr:** `http://localhost:7280`

Index schema (created via `PUT /api/v1/indexes`):
```json
{
  "index_id": "fts_docs",
  "doc_mapping": {
    "field_mappings": [
      {"name": "doc_id", "type": "text", "tokenizer": "raw", "stored": true},
      {"name": "text",   "type": "text", "tokenizer": "default", "stored": true,
       "record": "position", "fieldnorms": true}
    ]
  },
  "search_settings": {"default_search_fields": ["text"]}
}
```

Batch index (`POST /api/v1/fts_docs/ingest`):
```
Content-Type: application/x-ndjson
{"doc_id":"...","text":"..."}
{"doc_id":"...","text":"..."}
...
```
Use `?commit=force` on the last batch to flush.

Search (`POST /api/v1/fts_docs/search`):
```json
{"query": "...", "max_hits": 10, "start_offset": 0, "snippet_fields": ["text"]}
```

Stats: `GET /api/v1/fts_docs/describe`

---

### 7. tantivy-lnx (`pkg/index/driver/tantivy-lnx/`)

**Type:** external HTTP REST (custom client)
**Default addr:** `http://localhost:8000`
**Note:** lnx is a lightweight REST server wrapping Tantivy

Index creation (`POST /api/v1/indexes`):
```json
{
  "override_if_exists": true,
  "index": {
    "name": "fts_docs",
    "writer_threads": 4,
    "writer_heap_size_bytes": 67108864,
    "reader_threads": 4,
    "max_concurrency": 10,
    "search_fields": ["text"],
    "store_records": true,
    "fields": {
      "doc_id": {"type": "text", "stored": true},
      "text":   {"type": "text", "stored": false}
    }
  }
}
```

Batch index (`POST /api/v1/indexes/fts_docs/documents`):
```json
[{"doc_id":"...","text":"..."},{"doc_id":"...","text":"..."}]
```

Search (`POST /api/v1/indexes/fts_docs/search`):
```json
{"query": "...", "limit": 10, "offset": 0}
```

Stats: `GET /api/v1/indexes/fts_docs/summary`

---

## Docker Compose Files

All services use bind-mount (not named volume) so disk usage is measurable via
`du -sh $HOME/data/fts/{engine}/`.

### docker/meilisearch/docker-compose.yml

```yaml
services:
  meilisearch:
    image: getmeili/meilisearch:v1.13
    container_name: fts-meilisearch
    ports:
      - "7700:7700"
    environment:
      - MEILI_ENV=development
      - MEILI_NO_ANALYTICS=true
      - MEILI_DB_PATH=/meili_data
    volumes:
      - ${FTS_DATA_DIR:-$HOME/data/fts}/meilisearch:/meili_data
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:7700/health | grep -q '\"status\":\"available\"'"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 20s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 4G
```

### docker/clickhouse/docker-compose.yml

```yaml
services:
  clickhouse:
    image: clickhouse/clickhouse-server:25.4
    container_name: fts-clickhouse
    ports:
      - "8123:8123"   # HTTP interface
      - "9000:9000"   # Native TCP interface
    environment:
      - CLICKHOUSE_DB=fts
      - CLICKHOUSE_USER=fts
      - CLICKHOUSE_PASSWORD=fts
      - CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT=1
    volumes:
      - ${FTS_DATA_DIR:-$HOME/data/fts}/clickhouse:/var/lib/clickhouse
    ulimits:
      nofile:
        soft: 262144
        hard: 262144
    healthcheck:
      test: ["CMD-SHELL", "clickhouse-client --query 'SELECT 1'"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 4G
```

### docker/postgres-fts/docker-compose.yml

```yaml
services:
  postgres:
    image: postgres:17
    container_name: fts-postgres
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_DB=fts
      - POSTGRES_USER=fineweb
      - POSTGRES_PASSWORD=fineweb
    command: >
      postgres
      -c shared_buffers=1GB
      -c effective_cache_size=3GB
      -c maintenance_work_mem=512MB
      -c work_mem=64MB
      -c max_parallel_workers_per_gather=4
      -c max_parallel_workers=8
      -c checkpoint_completion_target=0.9
      -c wal_buffers=16MB
      -c max_wal_size=4GB
    volumes:
      - ${FTS_DATA_DIR:-$HOME/data/fts}/postgres:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U fineweb -d fts"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 4G
```

### docker/quickwit/docker-compose.yml

```yaml
services:
  quickwit:
    image: quickwit/quickwit:0.9
    container_name: fts-quickwit
    ports:
      - "7280:7280"
      - "7281:7281"
    environment:
      - QW_ENABLE_OPENTELEMETRY_OTLP_EXPORTER=false
    command: ["run"]
    volumes:
      - ${FTS_DATA_DIR:-$HOME/data/fts}/quickwit:/quickwit/qwdata
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:7280/api/v1/version"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 4G
```

### docker/lnx/docker-compose.yml

```yaml
services:
  lnx:
    image: ghcr.io/lnx-search/lnx:latest
    container_name: fts-lnx
    ports:
      - "8000:8000"
    environment:
      - LNX_LOG_LEVEL=info
    volumes:
      - ${FTS_DATA_DIR:-$HOME/data/fts}/lnx:/var/lib/lnx
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:8000/api/v1/indexes"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 4G
```

---

## File Layout

```
pkg/index/
├── engine.go                           # + AddrSetter interface
├── driver/
│   ├── bleve/
│   │   └── bleve.go                    # new: embedded BM25
│   ├── tantivy-go/
│   │   └── tantivy.go                  # new: CGO //go:build tantivy
│   ├── meilisearch/
│   │   └── meilisearch.go              # new: HTTP
│   ├── clickhouse/
│   │   └── clickhouse.go               # new: native TCP
│   ├── postgres/
│   │   └── postgres.go                 # new: pgx/v5 COPY + GIN
│   ├── quickwit/
│   │   └── quickwit.go                 # new: HTTP REST
│   ├── tantivy-lnx/
│   │   └── lnx.go                      # new: HTTP REST
│   ├── devnull/devnull.go              # existing
│   ├── duckdb/duckdb.go                # existing
│   ├── sqlite/sqlite.go                # existing
│   └── chdb/chdb.go                    # existing (CGO //go:build chdb)

docker/
├── meilisearch/docker-compose.yml      # new
├── clickhouse/docker-compose.yml       # new
├── postgres-fts/docker-compose.yml     # new
├── quickwit/docker-compose.yml         # new
└── lnx/docker-compose.yml              # new

cli/
└── cc_fts.go                           # + --addr flag + new driver imports

go.mod                                  # + clickhouse-go/v2, + tantivy-go (build-tagged)
```

---

## Build Tags

| Driver | Tag | Command |
|--------|-----|---------|
| tantivy-go | `tantivy` | `go build -tags tantivy ./...` |
| chdb | `chdb` | `go build -tags chdb ./...` |
| all others | _(none)_ | `go build ./...` |

---

## Benchmark Plan

Dataset: `CC-MAIN-2026-08`, 173,720 docs, source `--source bin` (docs.bin pre-packed).

### Index Benchmark

Dataset: CC-MAIN-2026-08, 173,720 docs, source `--source bin`, Apple M-series Mac (ARM64).

| Engine | Time (s) | Docs/s | Peak RSS (MB) | Disk (MB) |
|--------|----------|--------|--------------|-----------|
| devnull | 0.2 | 787,840 | 0 | 0 |
| sqlite | 0.8 | 212,491 | 314 | 1,126 |
| tantivy (CGO) | 34.8 | 4,998 | 327 | 764 |
| bleve | 75.9 | 2,288 | 3,690 | 3,584 |
| duckdb | 100.8 | 1,724 | 323 | 1,536 |
| meilisearch | *(requires Docker)* | — | — | — |
| clickhouse | *(requires Docker)* | — | — | — |
| postgres | *(requires Docker)* | — | — | — |
| quickwit | *(requires Docker)* | — | — | — |
| tantivy-lnx | *(requires Docker)* | — | — | — |

### Search Benchmark

10 queries, warm run, limit=10. Apple M-series Mac (ARM64), 173,720-doc index.

Queries:
1. `machine learning`
2. `climate change`
3. `artificial intelligence`
4. `United States`
5. `open source software`
6. `COVID-19 pandemic`
7. `data privacy`
8. `renewable energy`
9. `blockchain technology`
10. `neural network`

| Engine | Avg ms | P95 ms | Notes |
|--------|--------|--------|-------|
| tantivy (CGO) | 2 | 2 | BM25, sub-ms after warm-up |
| bleve | 31 | 52 | BM25, returns total hit count |
| sqlite | 85 | 287 | FTS5 BM25; "COVID-19 pandemic" unsupported (hyphen) |
| duckdb | 218 | 244 | tfidf; full-scan on each query; cold start ~800ms |
| meilisearch | *(requires Docker)* | — | — |
| clickhouse | *(requires Docker)* | — | — |
| postgres | *(requires Docker)* | — | — |
| quickwit | *(requires Docker)* | — | — |
| tantivy-lnx | *(requires Docker)* | — | — |

---

## Implementation Order

1. `pkg/index/engine.go` — add `AddrSetter` interface
2. `cli/cc_fts.go` — add `--addr` flag, SetAddr wiring, new driver imports
3. `pkg/index/driver/bleve/` — embedded BM25 (no Docker needed)
4. `go.mod` — add `clickhouse-go/v2`; add `tantivy-go` (build-tagged)
5. `docker/meilisearch/`, `docker/clickhouse/`, `docker/postgres-fts/`, `docker/quickwit/`, `docker/lnx/`
6. `pkg/index/driver/meilisearch/`
7. `pkg/index/driver/clickhouse/`
8. `pkg/index/driver/postgres/`
9. `pkg/index/driver/quickwit/`
10. `pkg/index/driver/tantivy-lnx/`
11. `pkg/index/driver/tantivy-go/` (CGO, build-tagged, last due to Rust dep complexity)
12. Benchmark all engines; fill tables in this spec
