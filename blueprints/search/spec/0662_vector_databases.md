# 0662 Vector Databases: Sequential Podman Verification and Performance Report

Date: 2026-03-04
Workspace: `/Users/apple/github/go-mizu/mizu/blueprints/search`

## Scope

Validated 10 vector drivers implemented under `pkg/vector/driver/*` against live containers started with Podman Compose, one backend at a time:

1. qdrant
2. weaviate
3. milvus
4. chroma
5. elasticsearch
6. opensearch
7. meilisearch
8. typesense
9. pgvector
10. solr

All stacks were shut down first, then each backend was started, tested, and shut down before moving to the next.

## Commands Used

Shutdown all:

```bash
for d in qdrant weaviate milvus chroma elasticsearch opensearch meilisearch typesense pgvector solr; do
  DOCKER_CONFIG=/tmp/podman-docker-config podman compose -f docker/$d/docker-compose.yaml down
 done
```

Per-backend verification test:

```bash
go test ./pkg/vector/driver -run "TestVectorDriversRoundTrip/<backend>$" -count=1 -json
```

Final package test:

```bash
go test ./pkg/vector/...
```

## Methodology

For each backend, measured:

- `compose_up_s`: time for `podman compose up -d`
- `ready_s`: time until service readiness probe succeeded
- `test_s`: elapsed time of `TestVectorDriversRoundTrip/<backend>` from `go test -json`

Readiness probes:

- qdrant: `http://localhost:6333/healthz`
- weaviate: `http://localhost:8080/v1/.well-known/ready`
- milvus: `http://localhost:9091/healthz`
- chroma: `http://localhost:8000/api/v2/heartbeat`
- elasticsearch: `http://localhost:9201`
- opensearch: `http://localhost:9200`
- meilisearch: `http://localhost:7700/health`
- typesense: `http://localhost:8108/health`
- pgvector: TCP check `localhost:5432`
- solr: `http://localhost:8983/solr/admin/info/system?wt=json`

## Performance Results

| Backend | compose_up_s | ready_s | test_s | Status |
|---|---:|---:|---:|---|
| qdrant | 0.278 | 0.053 | 0.06 | PASS |
| weaviate | 0.239 | 7.630 | 0.20 | PASS |
| milvus | 0.351 | 2.270 | 1.47 | PASS |
| chroma | 0.241 | 0.015 | 0.04 | PASS |
| elasticsearch | 0.230 | 7.334 | 0.34 | PASS |
| opensearch | 0.235 | 7.319 | 0.42 | PASS |
| meilisearch | 0.239 | 0.017 | 0.95 | PASS |
| typesense | 0.235 | 3.114 | 0.02 | PASS |
| pgvector | 0.242 | 0.016 | 0.03 | PASS |
| solr | 0.216 | 1.395 | 0.15 | PASS |

## Driver Conformance to `pkg/vector` Spec

Reference contract: `pkg/vector/vector.go`.

### Common behavior verified by tests

- `Store.Collection(name)` returns a collection handle for the logical namespace.
- `Collection.Init(ctx)` is callable and prepares backing schema/index.
- `Collection.Index(ctx, []Item)` accepts vector items and indexes them.
- `Collection.Search(ctx, Query)` returns at least one nearest-neighbor hit for a known query.
- Dimension mismatch checks are enforced in drivers before write.
- `Query.K` is normalized to backend defaults when unset/invalid.

### Backend-specific implementation notes

- `qdrant`
  - Uses `/collections/{name}` and `/points` APIs.
  - Converts non-numeric IDs to deterministic UUIDs to satisfy Qdrant ID constraints.

- `weaviate`
  - Uses class schema + batch objects + GraphQL `nearVector` query.
  - Converts IDs to deterministic UUIDs (Weaviate requires UUID IDs).

- `milvus`
  - Uses REST v2 endpoints on port `19530`.
  - Converts IDs to int64 and explicitly loads collection before/around search.
  - Includes retry window for eventual visibility after insert/load.

- `chroma`
  - Uses Chroma v2 tenant/database API paths.
  - Resolves collection name to v2 collection ID and then uses `add`/`query` by ID.

- `elasticsearch`
  - Creates `dense_vector` mapping with cosine similarity.
  - Uses `_bulk?refresh=true` + kNN search.

- `opensearch`
  - Creates `knn_vector` mapping with KNN enabled.
  - Uses OpenSearch KNN query search path.

- `meilisearch`
  - Configures `userProvided` embedder before vector search.
  - Waits on async tasks for settings/doc updates.
  - Uses required `hybrid.embedder` + vector query shape for current server behavior.

- `typesense`
  - Creates schema with `embedding` plus a searchable `text` field.
  - Uses vector query syntax with `query_by=text`.

- `pgvector`
  - Creates extension/table/index and performs cosine NN query via `<=>`.
  - Implements `vector.Closer` for DB pool lifecycle.

- `solr`
  - Targets precreated core `gettingstarted` in current compose setup.
  - Adds `DenseVectorField` schema and executes `{!knn ...}` query.

## Outcome

- All 10 backends passed live round-trip validation under Podman.
- Sequential one-by-one orchestration completed successfully.
- No runtime test failures remained after API compatibility fixes.

## Repro Notes

- Podman in this environment delegates compose to external provider (`docker-compose`).
- `DOCKER_CONFIG=/tmp/podman-docker-config` was used to avoid Docker credential helper errors.
- Performance values are from this host/session only and primarily reflect startup/readiness + minimal round-trip workload, not throughput benchmarking.
