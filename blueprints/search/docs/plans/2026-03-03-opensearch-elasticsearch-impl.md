# OpenSearch & Elasticsearch FTS Drivers — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `opensearch` and `elasticsearch` drivers to `pkg/index`, each with a Docker Compose file (engine + optional dashboard via `--profile full`), tests, CLI wiring, and a benchmark spec at `spec/0645_opensearch.md`.

**Architecture:** Two independent raw-HTTP drivers following the quickwit pattern (`net/http` + JSON, `BaseExternal` embedded, `_bulk` for ingest, `_search` for queries). Both register via `init()` side-effect imports in `cli/cc_fts.go`. No new go.mod dependencies.

**Tech Stack:** Go `net/http`, OpenSearch 3.5.0 REST API, Elasticsearch 9.0.0 REST API, Docker Compose profiles.

---

## Context: Existing Pattern to Follow

Read these files before starting — every decision below mirrors them:

- `pkg/index/driver/quickwit/quickwit.go` — the template for raw-HTTP drivers
- `pkg/index/driver/quickwit/quickwit_test.go` — the template for integration tests
- `pkg/index/engine.go` — `Engine`, `BaseExternal`, `AddrSetter` interfaces
- `cli/cc_fts.go` — where driver imports live (lines 20–30)
- `docker/meilisearch/docker-compose.yml` — existing compose reference

Working directory for all commands: `blueprints/search/`

---

## Task 1: Write `spec/0645_opensearch.md`

**Files:**
- Create: `spec/0645_opensearch.md`

**Step 1: Create the spec file**

```markdown
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
PUT /{index}
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
search cc fts index --engine opensearch
search cc fts index --engine elasticsearch --addr http://localhost:9201
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
```

**Step 2: Commit**

```bash
git add spec/0645_opensearch.md
git commit -m "docs(spec/0645): add OpenSearch + Elasticsearch FTS driver spec"
```

---

## Task 2: Docker Compose — OpenSearch

**Files:**
- Create: `docker/opensearch/docker-compose.yml`

**Step 1: Create the compose file**

```yaml
services:
  opensearch:
    image: opensearchproject/opensearch:3.5.0
    container_name: fts-opensearch
    ports:
      - "9200:9200"
      - "9600:9600"
    environment:
      - discovery.type=single-node
      - DISABLE_SECURITY_PLUGIN=true
      - bootstrap.memory_lock=true
      - OPENSEARCH_JAVA_OPTS=-Xms512m -Xmx512m
    ulimits:
      memlock:
        soft: -1
        hard: -1
      nofile:
        soft: 65536
        hard: 65536
    volumes:
      - ${FTS_DATA_DIR:-${HOME}/data/fts}/opensearch:/usr/share/opensearch/data
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:9200/_cluster/health | grep -qE '\"status\":\"(green|yellow)\"'"]
      interval: 10s
      timeout: 5s
      retries: 12
      start_period: 30s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 4G

  opensearch-dashboards:
    image: opensearchproject/opensearch-dashboards:3.5.0
    container_name: fts-opensearch-dashboards
    profiles: [full]
    ports:
      - "5601:5601"
    environment:
      - OPENSEARCH_HOSTS=http://opensearch:9200
      - DISABLE_SECURITY_DASHBOARDS_PLUGIN=true
    depends_on:
      opensearch:
        condition: service_healthy
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 2G
```

**Step 2: Smoke-test the compose file parses**

```bash
docker compose -f docker/opensearch/docker-compose.yml config --quiet
```

Expected: no output (exit 0). If it errors, fix the YAML.

**Step 3: Commit**

```bash
git add docker/opensearch/docker-compose.yml
git commit -m "feat(docker/opensearch): add OpenSearch 3.5.0 compose with dashboards profile"
```

---

## Task 3: Docker Compose — Elasticsearch

**Files:**
- Create: `docker/elasticsearch/docker-compose.yml`

**Step 1: Create the compose file**

```yaml
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:9.0.0
    container_name: fts-elasticsearch
    ports:
      - "9201:9200"
      - "9301:9300"
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - bootstrap.memory_lock=true
      - ES_JAVA_OPTS=-Xms512m -Xmx512m
    ulimits:
      memlock:
        soft: -1
        hard: -1
      nofile:
        soft: 65536
        hard: 65536
    volumes:
      - ${FTS_DATA_DIR:-${HOME}/data/fts}/elasticsearch:/usr/share/elasticsearch/data
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:9200/_cluster/health | grep -qE '\"status\":\"(green|yellow)\"'"]
      interval: 10s
      timeout: 5s
      retries: 12
      start_period: 30s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 4G

  kibana:
    image: docker.elastic.co/kibana/kibana:9.0.0
    container_name: fts-kibana
    profiles: [full]
    ports:
      - "5602:5601"
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
    depends_on:
      elasticsearch:
        condition: service_healthy
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 2G
```

**Step 2: Smoke-test the compose file parses**

```bash
docker compose -f docker/elasticsearch/docker-compose.yml config --quiet
```

Expected: no output (exit 0).

**Step 3: Commit**

```bash
git add docker/elasticsearch/docker-compose.yml
git commit -m "feat(docker/elasticsearch): add Elasticsearch 9.0.0 compose with Kibana profile"
```

---

## Task 4: OpenSearch Driver

**Files:**
- Create: `pkg/index/driver/opensearch/opensearch.go`
- Create: `pkg/index/driver/opensearch/opensearch_test.go`

### Step 1: Write the failing test first

`pkg/index/driver/opensearch/opensearch_test.go`:

```go
package opensearch_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/opensearch"
)

func skipIfOpenSearchDown(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:9200/", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("opensearch not available at localhost:9200: %v", err)
	}
	resp.Body.Close()
}

func TestOpenSearchEngine_AddrSetter(t *testing.T) {
	eng, err := index.NewEngine("opensearch")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	setter, ok := eng.(index.AddrSetter)
	if !ok {
		t.Fatal("opensearch Engine does not implement AddrSetter")
	}
	setter.SetAddr("http://my-server:9200")
}

func TestOpenSearchEngine_Roundtrip(t *testing.T) {
	skipIfOpenSearchDown(t)
	ctx := context.Background()
	dir := t.TempDir()

	eng, err := index.NewEngine("opensearch")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	eng.(index.AddrSetter).SetAddr("http://localhost:9200")

	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	docs := []index.Document{
		{DocID: "os-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "os-doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "os-doc3", Text: []byte("open source software development programming")},
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := eng.Search(ctx, index.Query{Text: "machine learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("expected hits, got none")
	}

	stats, err := eng.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DocCount < 3 {
		t.Errorf("DocCount: got %d, want >= 3", stats.DocCount)
	}
}
```

### Step 2: Run the test — expect compile failure (package doesn't exist yet)

```bash
go test ./pkg/index/driver/opensearch/...
```

Expected: `cannot find package` (or similar). This confirms we haven't written it yet.

### Step 3: Write the implementation

`pkg/index/driver/opensearch/opensearch.go`:

```go
package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

const (
	defaultAddr = "http://localhost:9200"
	indexName   = "fts_docs"
)

func init() {
	index.Register("opensearch", func() index.Engine { return NewEngine() })
}

// Engine is an external FTS driver backed by OpenSearch via the REST API.
type Engine struct {
	index.BaseExternal
	client *http.Client
	base   string
}

func NewEngine() *Engine {
	return &Engine{client: &http.Client{Timeout: 120 * time.Second}}
}

func (e *Engine) Name() string { return "opensearch" }

// Open ensures the data directory exists, sets the base URL, and creates the index
// with an English-language text mapping if it does not already exist.
func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.base = e.EffectiveAddr(defaultAddr)

	mapping := map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"doc_id": map[string]any{"type": "keyword"},
				"text":   map[string]any{"type": "text", "analyzer": "english"},
			},
		},
	}
	body, _ := json.Marshal(mapping)
	req, _ := http.NewRequestWithContext(ctx, "PUT", e.base+"/"+indexName, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("opensearch create index: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	// 200 = created, 400 = already exists — both are fine
	if resp.StatusCode >= 500 {
		return fmt.Errorf("opensearch create index HTTP %d", resp.StatusCode)
	}
	return nil
}

func (e *Engine) Close() error { return nil }

// Stats returns the document count and store size in bytes.
func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	var stats index.EngineStats

	req, _ := http.NewRequestWithContext(ctx, "GET", e.base+"/"+indexName+"/_count", nil)
	resp, err := e.client.Do(req)
	if err != nil {
		return stats, fmt.Errorf("opensearch count: %w", err)
	}
	defer resp.Body.Close()
	var countResp struct {
		Count int64 `json:"count"`
	}
	json.NewDecoder(resp.Body).Decode(&countResp) //nolint:errcheck
	stats.DocCount = countResp.Count

	req2, _ := http.NewRequestWithContext(ctx, "GET", e.base+"/"+indexName+"/_stats/store", nil)
	resp2, err := e.client.Do(req2)
	if err != nil {
		return stats, nil // doc count is still valid
	}
	defer resp2.Body.Close()
	var storeResp struct {
		All struct {
			Total struct {
				Store struct {
					SizeInBytes int64 `json:"size_in_bytes"`
				} `json:"store"`
			} `json:"total"`
		} `json:"_all"`
	}
	json.NewDecoder(resp2.Body).Decode(&storeResp) //nolint:errcheck
	stats.DiskBytes = storeResp.All.Total.Store.SizeInBytes

	return stats, nil
}

// Index bulk-ingests documents using the _bulk API.
func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	for _, d := range docs {
		enc.Encode(map[string]any{"index": map[string]any{"_index": indexName, "_id": d.DocID}}) //nolint:errcheck
		enc.Encode(map[string]string{"doc_id": d.DocID, "text": string(d.Text)})                 //nolint:errcheck
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", e.base+"/_bulk", strings.NewReader(sb.String()))
	req.Header.Set("Content-Type", "application/x-ndjson")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("opensearch bulk: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("opensearch bulk HTTP %d: %s", resp.StatusCode, b)
	}
	var bulkResp struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			Error *struct{ Reason string `json:"reason"` } `json:"error"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&bulkResp); err != nil {
		return fmt.Errorf("opensearch bulk decode: %w", err)
	}
	if bulkResp.Errors {
		for _, item := range bulkResp.Items {
			for _, op := range item {
				if op.Error != nil {
					return fmt.Errorf("opensearch bulk item error: %s", op.Error.Reason)
				}
			}
		}
	}
	return nil
}

// Search executes a BM25 match query with English analysis and returns highlighted snippets.
func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	payload := map[string]any{
		"query":     map[string]any{"match": map[string]any{"text": q.Text}},
		"highlight": map[string]any{"fields": map[string]any{"text": map[string]any{"fragment_size": 200, "number_of_fragments": 1}}},
		"size":      limit,
		"from":      q.Offset,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", e.base+"/"+indexName+"/_search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return index.Results{}, fmt.Errorf("opensearch search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return index.Results{}, fmt.Errorf("opensearch search HTTP %d: %s", resp.StatusCode, b)
	}
	var sr struct {
		Hits struct {
			Total struct{ Value int `json:"value"` } `json:"total"`
			Hits  []struct {
				ID        string              `json:"_id"`
				Score     float64             `json:"_score"`
				Highlight map[string][]string `json:"highlight"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return index.Results{}, fmt.Errorf("opensearch search decode: %w", err)
	}
	results := index.Results{Total: sr.Hits.Total.Value}
	for _, hit := range sr.Hits.Hits {
		h := index.Hit{DocID: hit.ID, Score: hit.Score}
		if frags := hit.Highlight["text"]; len(frags) > 0 {
			h.Snippet = frags[0]
		}
		results.Hits = append(results.Hits, h)
	}
	return results, nil
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
```

### Step 4: Run the AddrSetter test (no Docker needed)

```bash
go test -run TestOpenSearchEngine_AddrSetter ./pkg/index/driver/opensearch/...
```

Expected: `PASS`

### Step 5: Run the roundtrip test (requires Docker)

Start OpenSearch first if not running:
```bash
docker compose -f docker/opensearch/docker-compose.yml up -d
# wait for healthy
docker compose -f docker/opensearch/docker-compose.yml ps
```

Then run:
```bash
go test -v -run TestOpenSearchEngine_Roundtrip ./pkg/index/driver/opensearch/...
```

Expected: `PASS`. If OpenSearch is not running, test auto-skips.

### Step 6: Commit

```bash
git add pkg/index/driver/opensearch/
git commit -m "feat(index/opensearch): add OpenSearch FTS driver (raw HTTP, _bulk + _search)"
```

---

## Task 5: Elasticsearch Driver

**Files:**
- Create: `pkg/index/driver/elasticsearch/elasticsearch.go`
- Create: `pkg/index/driver/elasticsearch/elasticsearch_test.go`

### Step 1: Write the failing test first

`pkg/index/driver/elasticsearch/elasticsearch_test.go`:

```go
package elasticsearch_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/elasticsearch"
)

func skipIfElasticsearchDown(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:9201/", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("elasticsearch not available at localhost:9201: %v", err)
	}
	resp.Body.Close()
}

func TestElasticsearchEngine_AddrSetter(t *testing.T) {
	eng, err := index.NewEngine("elasticsearch")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	setter, ok := eng.(index.AddrSetter)
	if !ok {
		t.Fatal("elasticsearch Engine does not implement AddrSetter")
	}
	setter.SetAddr("http://my-server:9200")
}

func TestElasticsearchEngine_Roundtrip(t *testing.T) {
	skipIfElasticsearchDown(t)
	ctx := context.Background()
	dir := t.TempDir()

	eng, err := index.NewEngine("elasticsearch")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	eng.(index.AddrSetter).SetAddr("http://localhost:9201")

	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	docs := []index.Document{
		{DocID: "es-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "es-doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "es-doc3", Text: []byte("open source software development programming")},
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := eng.Search(ctx, index.Query{Text: "machine learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("expected hits, got none")
	}

	stats, err := eng.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DocCount < 3 {
		t.Errorf("DocCount: got %d, want >= 3", stats.DocCount)
	}
}
```

### Step 2: Run the test — expect compile failure

```bash
go test ./pkg/index/driver/elasticsearch/...
```

Expected: `cannot find package`.

### Step 3: Write the implementation

`pkg/index/driver/elasticsearch/elasticsearch.go` — identical to the OpenSearch driver except for the package name, registered engine name, and default address:

```go
package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

const (
	defaultAddr = "http://localhost:9201"
	indexName   = "fts_docs"
)

func init() {
	index.Register("elasticsearch", func() index.Engine { return NewEngine() })
}

// Engine is an external FTS driver backed by Elasticsearch via the REST API.
type Engine struct {
	index.BaseExternal
	client *http.Client
	base   string
}

func NewEngine() *Engine {
	return &Engine{client: &http.Client{Timeout: 120 * time.Second}}
}

func (e *Engine) Name() string { return "elasticsearch" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.base = e.EffectiveAddr(defaultAddr)

	mapping := map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"doc_id": map[string]any{"type": "keyword"},
				"text":   map[string]any{"type": "text", "analyzer": "english"},
			},
		},
	}
	body, _ := json.Marshal(mapping)
	req, _ := http.NewRequestWithContext(ctx, "PUT", e.base+"/"+indexName, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("elasticsearch create index: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 500 {
		return fmt.Errorf("elasticsearch create index HTTP %d", resp.StatusCode)
	}
	return nil
}

func (e *Engine) Close() error { return nil }

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	var stats index.EngineStats

	req, _ := http.NewRequestWithContext(ctx, "GET", e.base+"/"+indexName+"/_count", nil)
	resp, err := e.client.Do(req)
	if err != nil {
		return stats, fmt.Errorf("elasticsearch count: %w", err)
	}
	defer resp.Body.Close()
	var countResp struct {
		Count int64 `json:"count"`
	}
	json.NewDecoder(resp.Body).Decode(&countResp) //nolint:errcheck
	stats.DocCount = countResp.Count

	req2, _ := http.NewRequestWithContext(ctx, "GET", e.base+"/"+indexName+"/_stats/store", nil)
	resp2, err := e.client.Do(req2)
	if err != nil {
		return stats, nil
	}
	defer resp2.Body.Close()
	var storeResp struct {
		All struct {
			Total struct {
				Store struct {
					SizeInBytes int64 `json:"size_in_bytes"`
				} `json:"store"`
			} `json:"total"`
		} `json:"_all"`
	}
	json.NewDecoder(resp2.Body).Decode(&storeResp) //nolint:errcheck
	stats.DiskBytes = storeResp.All.Total.Store.SizeInBytes

	return stats, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	for _, d := range docs {
		enc.Encode(map[string]any{"index": map[string]any{"_index": indexName, "_id": d.DocID}}) //nolint:errcheck
		enc.Encode(map[string]string{"doc_id": d.DocID, "text": string(d.Text)})                 //nolint:errcheck
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", e.base+"/_bulk", strings.NewReader(sb.String()))
	req.Header.Set("Content-Type", "application/x-ndjson")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("elasticsearch bulk: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elasticsearch bulk HTTP %d: %s", resp.StatusCode, b)
	}
	var bulkResp struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			Error *struct{ Reason string `json:"reason"` } `json:"error"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&bulkResp); err != nil {
		return fmt.Errorf("elasticsearch bulk decode: %w", err)
	}
	if bulkResp.Errors {
		for _, item := range bulkResp.Items {
			for _, op := range item {
				if op.Error != nil {
					return fmt.Errorf("elasticsearch bulk item error: %s", op.Error.Reason)
				}
			}
		}
	}
	return nil
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	payload := map[string]any{
		"query":     map[string]any{"match": map[string]any{"text": q.Text}},
		"highlight": map[string]any{"fields": map[string]any{"text": map[string]any{"fragment_size": 200, "number_of_fragments": 1}}},
		"size":      limit,
		"from":      q.Offset,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", e.base+"/"+indexName+"/_search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return index.Results{}, fmt.Errorf("elasticsearch search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return index.Results{}, fmt.Errorf("elasticsearch search HTTP %d: %s", resp.StatusCode, b)
	}
	var sr struct {
		Hits struct {
			Total struct{ Value int `json:"value"` } `json:"total"`
			Hits  []struct {
				ID        string              `json:"_id"`
				Score     float64             `json:"_score"`
				Highlight map[string][]string `json:"highlight"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return index.Results{}, fmt.Errorf("elasticsearch search decode: %w", err)
	}
	results := index.Results{Total: sr.Hits.Total.Value}
	for _, hit := range sr.Hits.Hits {
		h := index.Hit{DocID: hit.ID, Score: hit.Score}
		if frags := hit.Highlight["text"]; len(frags) > 0 {
			h.Snippet = frags[0]
		}
		results.Hits = append(results.Hits, h)
	}
	return results, nil
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
```

### Step 4: Run the AddrSetter test (no Docker needed)

```bash
go test -run TestElasticsearchEngine_AddrSetter ./pkg/index/driver/elasticsearch/...
```

Expected: `PASS`

### Step 5: Run the roundtrip test (requires Docker)

Start Elasticsearch first if not running:
```bash
docker compose -f docker/elasticsearch/docker-compose.yml up -d
docker compose -f docker/elasticsearch/docker-compose.yml ps
```

Then run:
```bash
go test -v -run TestElasticsearchEngine_Roundtrip ./pkg/index/driver/elasticsearch/...
```

Expected: `PASS`. Auto-skips if Elasticsearch not running.

### Step 6: Commit

```bash
git add pkg/index/driver/elasticsearch/
git commit -m "feat(index/elasticsearch): add Elasticsearch FTS driver (raw HTTP, _bulk + _search)"
```

---

## Task 6: CLI Wiring

**Files:**
- Modify: `cli/cc_fts.go` (lines 20–30, the import block)

### Step 1: Find the import block

Open `cli/cc_fts.go` and locate the side-effect import block. It currently looks like:

```go
import (
    ...
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/bleve"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/chdb"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/clickhouse"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/devnull"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/duckdb"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/meilisearch"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/postgres"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/quickwit"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/sqlite"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/tantivy-lnx"
)
```

### Step 2: Add the two new imports (alphabetical order)

Add after the existing `devnull` import and before `duckdb`:

```go
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/elasticsearch"
```

And after `devnull` but before `duckdb` — wait, `elasticsearch` comes before `meilisearch` alphabetically. Insert in sorted position:

```go
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/bleve"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/chdb"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/clickhouse"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/devnull"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/duckdb"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/elasticsearch"   // NEW
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/meilisearch"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/opensearch"       // NEW
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/postgres"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/quickwit"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/sqlite"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/tantivy-lnx"
```

### Step 3: Verify the build compiles

```bash
go build ./cmd/search/
```

Expected: exits 0, no errors. The `search` binary is produced at `./search` (or wherever `cmd/search` outputs).

### Step 4: Verify both engines are listed

```bash
./search cc fts index --help 2>&1 | grep -i engine
```

Expected output includes `opensearch` and `elasticsearch` in the engine list (or the help text shows them as valid values).

### Step 5: Commit

```bash
git add cli/cc_fts.go
git commit -m "feat(cli): register opensearch and elasticsearch drivers in cc_fts"
```

---

## Task 7: Benchmark and Fill Spec

Run the full benchmark suite with both engines. Requires Docker running both containers.

### Step 1: Start both engines

```bash
docker compose -f docker/opensearch/docker-compose.yml up -d
docker compose -f docker/elasticsearch/docker-compose.yml up -d
# Wait for both to be healthy (check with docker compose ps)
```

### Step 2: Ensure the pre-packed bin source exists

The benchmark uses `--source bin` which reads from a pre-packed `docs.bin` file.
Verify it exists:
```bash
ls ~/data/common-crawl/CC-MAIN-2026-08/fts/pack/docs.bin
```

If missing, create it:
```bash
search cc fts pack --crawl CC-MAIN-2026-08 --format bin
```

### Step 3: Run index benchmark — OpenSearch

```bash
time search cc fts index --engine opensearch --source bin --crawl CC-MAIN-2026-08
```

Record: wall-clock time, docs/s from progress output, peak RSS from progress output, then check disk:
```bash
du -sm ~/data/fts/opensearch/
```

### Step 4: Run index benchmark — Elasticsearch

```bash
time search cc fts index \
  --engine elasticsearch \
  --addr http://localhost:9201 \
  --source bin \
  --crawl CC-MAIN-2026-08
```

Record: same metrics. Disk:
```bash
du -sm ~/data/fts/elasticsearch/
```

### Step 5: Run search benchmark — OpenSearch (10 queries, measure each)

```bash
for q in "machine learning" "climate change" "artificial intelligence" \
          "United States" "open source software" "COVID-19 pandemic" \
          "data privacy" "renewable energy" "blockchain technology" \
          "neural network"; do
  time search cc fts search --engine opensearch --query "$q" --limit 10 2>&1 | tail -1
done
```

Record avg and P95 latency across the 10 queries.

### Step 6: Run search benchmark — Elasticsearch (10 queries)

```bash
for q in "machine learning" "climate change" "artificial intelligence" \
          "United States" "open source software" "COVID-19 pandemic" \
          "data privacy" "renewable energy" "blockchain technology" \
          "neural network"; do
  time search cc fts search \
    --engine elasticsearch \
    --addr http://localhost:9201 \
    --query "$q" \
    --limit 10 2>&1 | tail -1
done
```

### Step 7: Fill benchmark tables in spec/0645_opensearch.md

Edit `spec/0645_opensearch.md` and fill in both tables with the measured values.

### Step 8: Commit

```bash
git add spec/0645_opensearch.md
git commit -m "docs(spec/0645): add benchmark results for opensearch and elasticsearch"
```

---

## Summary

| Task | Files | Commit message |
|------|-------|----------------|
| 1 | `spec/0645_opensearch.md` | `docs(spec/0645): add OpenSearch + Elasticsearch FTS driver spec` |
| 2 | `docker/opensearch/docker-compose.yml` | `feat(docker/opensearch): add OpenSearch 3.5.0 compose with dashboards profile` |
| 3 | `docker/elasticsearch/docker-compose.yml` | `feat(docker/elasticsearch): add Elasticsearch 9.0.0 compose with Kibana profile` |
| 4 | `pkg/index/driver/opensearch/` | `feat(index/opensearch): add OpenSearch FTS driver (raw HTTP, _bulk + _search)` |
| 5 | `pkg/index/driver/elasticsearch/` | `feat(index/elasticsearch): add Elasticsearch FTS driver (raw HTTP, _bulk + _search)` |
| 6 | `cli/cc_fts.go` | `feat(cli): register opensearch and elasticsearch drivers in cc_fts` |
| 7 | `spec/0645_opensearch.md` | `docs(spec/0645): add benchmark results for opensearch and elasticsearch` |
