# Design: OpenSearch & Elasticsearch FTS Drivers

**Date:** 2026-03-03
**Branch:** index-pane
**Status:** Approved

---

## Goal

Add two external FTS drivers — `opensearch` and `elasticsearch` — to `pkg/index`, each with a
Docker Compose file (engine + optional dashboard via profile), and benchmark both with the same
173,720-doc CC-MAIN-2026-08 corpus used in spec/0644. Write the detailed spec to
`spec/0645_opensearch.md`.

---

## Approach

Two fully independent drivers following the existing quickwit pattern (raw `net/http`, no new
go.mod dependencies). One docker-compose.yml per engine using Docker Compose profiles for the
dashboard service.

---

## Section 1: Docker Compose

### `docker/opensearch/docker-compose.yml`

- **Engine service** (`opensearch`): always started
  - Image: `opensearchproject/opensearch:3.5.0`
  - Ports: `9200:9200` (HTTP), `9600:9600` (performance analyser)
  - Security: disabled via `DISABLE_SECURITY_PLUGIN=true` (dev)
  - JVM: `-Xms512m -Xmx512m`
  - Data volume: `${FTS_DATA_DIR:-$HOME/data/fts}/opensearch:/usr/share/opensearch/data`
  - Memory limit: 4 GB

- **Dashboard service** (`opensearch-dashboards`): profile `full`
  - Image: `opensearchproject/opensearch-dashboards:3.5.0`
  - Port: `5601:5601`
  - Connects to `opensearch` service

### `docker/elasticsearch/docker-compose.yml`

- **Engine service** (`elasticsearch`): always started
  - Image: `docker.elastic.co/elasticsearch/elasticsearch:9.0.0`
  - Ports: `9201:9200` (host 9201 avoids collision with OpenSearch on 9200)
  - Security: disabled via `xpack.security.enabled=false` (dev)
  - JVM: `-Xms512m -Xmx512m`
  - Data volume: `${FTS_DATA_DIR:-$HOME/data/fts}/elasticsearch:/usr/share/elasticsearch/data`
  - Memory limit: 4 GB

- **Dashboard service** (`kibana`): profile `full`
  - Image: `docker.elastic.co/kibana/kibana:9.0.0`
  - Port: `5602:5601` (host 5602 avoids collision with OpenSearch Dashboards on 5601)
  - Connects to `elasticsearch` service

### Usage

```bash
# Engine only (minimal, fits small servers)
docker compose -f docker/opensearch/docker-compose.yml up -d
docker compose -f docker/elasticsearch/docker-compose.yml up -d

# Engine + dashboard (full)
docker compose -f docker/opensearch/docker-compose.yml --profile full up -d
docker compose -f docker/elasticsearch/docker-compose.yml --profile full up -d
```

---

## Section 2: Go Drivers

Both follow the quickwit driver pattern exactly. No new go.mod dependencies.

### Common API Shape (OpenSearch and Elasticsearch share the same REST API)

**Index mapping** (created at `Open` via `PUT /fts_docs`):
```json
{
  "mappings": {
    "properties": {
      "doc_id": { "type": "keyword" },
      "text":   { "type": "text", "analyzer": "english" }
    }
  }
}
```
Idempotent — HTTP 200 (created) and 400 (already exists) are both OK.

**Bulk indexing** (`POST /_bulk`, NDJSON):
```
{"index": {"_index": "fts_docs", "_id": "{docID}"}}
{"doc_id": "{docID}", "text": "{text}"}
```
One action+doc pair per document, all in a single request per batch.

**Search** (`POST /fts_docs/_search`):
```json
{
  "query": { "match": { "text": "<query>" } },
  "highlight": { "fields": { "text": { "fragment_size": 200, "number_of_fragments": 1 } } },
  "size": 10,
  "from": 0
}
```
BM25 scoring (default for both engines). Snippet from highlight fragments.

**Stats**:
- Doc count: `GET /fts_docs/_count` → `.count`
- Disk bytes: `GET /fts_docs/_stats/store` → `._all.total.store.size_in_bytes`

**Close**: no-op (stateless HTTP client).

### `pkg/index/driver/opensearch/opensearch.go`

- Package: `opensearch`
- Registers as: `"opensearch"`
- Default addr: `http://localhost:9200`
- Embeds `index.BaseExternal` (AddrSetter)
- HTTP client timeout: 120s

### `pkg/index/driver/elasticsearch/elasticsearch.go`

- Package: `elasticsearch`
- Registers as: `"elasticsearch"`
- Default addr: `http://localhost:9201`
- Embeds `index.BaseExternal` (AddrSetter)
- HTTP client timeout: 120s

---

## Section 3: CLI + Spec

### `cli/cc_fts.go`

Add two side-effect imports alongside existing external drivers:
```go
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/opensearch"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/elasticsearch"
```

### `spec/0645_opensearch.md`

New spec covering:
- Goal and background
- Docker setup + startup commands for each engine
- Driver API details (mapping, bulk ingest, search, stats)
- Benchmark plan: 173,720-doc CC-MAIN-2026-08 corpus, `--source bin`, same 10 queries as spec/0644
- Index benchmark table: time / docs-per-s / peak RSS / disk (to be filled)
- Search benchmark table: avg ms / P95 ms per engine (to be filled)
- Benchmark results (filled after implementation and runs)

---

## File Layout

```
docker/
  opensearch/
    docker-compose.yml          # new: OS 3.5.0 + dashboards (profile: full)
  elasticsearch/
    docker-compose.yml          # new: ES 9.0.0 + Kibana (profile: full)

pkg/index/driver/
  opensearch/
    opensearch.go               # new: raw HTTP driver, name="opensearch"
  elasticsearch/
    elasticsearch.go            # new: raw HTTP driver, name="elasticsearch"

cli/
  cc_fts.go                     # + 2 import lines

spec/
  0645_opensearch.md            # new: detailed spec + benchmark results
```

No changes to `go.mod` (raw `net/http` only).
