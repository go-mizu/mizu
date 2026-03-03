# spec/0645 — OpenSearch & Elasticsearch FTS Drivers

**Status:** implemented
**Branch:** index-pane

---

## Goal

Add two external FTS drivers — `opensearch` and `elasticsearch` — to `pkg/index`, each backed
by an HTTP REST client talking to the respective search engine. Both engines share the same
Elasticsearch-compatible REST API (OpenSearch forked from Elasticsearch).

Benchmark both drivers on the same 173,720-document CC-MAIN-2026-08 corpus used in spec/0644.

---

## Versions

| Engine | Version | Docker Image |
|--------|---------|--------------|
| OpenSearch | 3.5.0 | `opensearchproject/opensearch:3.5.0` |
| OpenSearch Dashboards | 3.5.0 | `opensearchproject/opensearch-dashboards:3.5.0` |
| Elasticsearch | 9.0.0 | `docker.elastic.co/elasticsearch/elasticsearch:9.0.0` |
| Kibana | 9.0.0 | `docker.elastic.co/kibana/kibana:9.0.0` |

---

## Docker Setup

### Start (engine only — minimal, fits small servers)

```bash
# OpenSearch on port 9200
docker compose -f docker/opensearch/docker-compose.yml up -d

# Elasticsearch on port 9201
docker compose -f docker/elasticsearch/docker-compose.yml up -d
```

### Start with dashboard (full)

```bash
docker compose -f docker/opensearch/docker-compose.yml --profile full up -d
# OpenSearch Dashboards → http://localhost:5601

docker compose -f docker/elasticsearch/docker-compose.yml --profile full up -d
# Kibana → http://localhost:5602
```

### Port map

| Service | Host port | Notes |
|---------|-----------|-------|
| OpenSearch | 9200 | default |
| OpenSearch Dashboards | 5601 | profile: full |
| Elasticsearch | 9201 | 9200 inside container; 9201 avoids conflict |
| Kibana | 5602 | 5601 inside container; 5602 avoids conflict |

### Data directories

```
$HOME/data/fts/opensearch/        # bind-mounted into OpenSearch container
$HOME/data/fts/elasticsearch/     # bind-mounted into Elasticsearch container
```

---

## Driver API Details

Both drivers (`opensearch`, `elasticsearch`) use identical REST calls since OpenSearch is
API-compatible with Elasticsearch. The only differences are the registered engine name and
the default address.

| | opensearch | elasticsearch |
|--|-----------|--------------|
| Registered name | `opensearch` | `elasticsearch` |
| Default addr | `http://localhost:9200` | `http://localhost:9201` |

### Index mapping (created at Open)

```
PUT /fts_docs
Content-Type: application/json

{
  "mappings": {
    "properties": {
      "doc_id": { "type": "keyword" },
      "text":   { "type": "text", "analyzer": "english" }
    }
  }
}
```

Idempotent: HTTP 200 (created) and 400 (already exists) are both accepted.

### Bulk ingest (called per batch in Index)

```
POST /_bulk
Content-Type: application/x-ndjson

{"index": {"_index": "fts_docs", "_id": "{docID}"}}
{"doc_id": "{docID}", "text": "{text}"}
... (one action+doc pair per document)
```

Checks `errors` field in bulk response and returns first item-level error if found.

### Search

```
POST /fts_docs/_search
Content-Type: application/json

{
  "query":     { "match": { "text": "<query>" } },
  "highlight": { "fields": { "text": { "fragment_size": 200, "number_of_fragments": 1 } } },
  "size": <limit>,
  "from": <offset>
}
```

Returns `Hit{DocID: _id, Score: _score, Snippet: highlight.text[0]}`.

### Stats

```
GET /fts_docs/_count          → .count
GET /fts_docs/_stats/store    → ._all.total.store.size_in_bytes
```

---

## Benchmark Plan

Dataset: CC-MAIN-2026-08, 173,720 docs, `--source bin` (pre-packed docs.bin).
Same 10 queries as spec/0644. Apple M-series Mac (ARM64).

### Index Benchmark

```bash
search cc fts index --engine opensearch --source bin --crawl CC-MAIN-2026-08
search cc fts index --engine elasticsearch --addr http://localhost:9201 --source bin --crawl CC-MAIN-2026-08
```

| Engine | Time (s) | Docs/s | Peak RSS (MB) | Disk (MB) |
|--------|----------|--------|--------------|-----------|
| opensearch | | | | |
| elasticsearch | | | | |

### Search Benchmark

10 queries, warm run, limit=10.

```bash
search cc fts search --engine opensearch --query "machine learning"
search cc fts search --engine elasticsearch --addr http://localhost:9201 --query "machine learning"
```

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
| opensearch | | | BM25, english analyzer |
| elasticsearch | | | BM25, english analyzer |
