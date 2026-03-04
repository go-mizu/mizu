# External FTS Drivers Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add seven new FTS engine drivers (bleve, tantivy-go CGO, meilisearch, clickhouse, postgres, quickwit, tantivy-lnx), docker-compose files per engine, `--addr` CLI flag, and benchmark all against the 173,720-doc CC-MAIN-2026-08 corpus.

**Architecture:** Each driver implements the existing `index.Engine` interface (`Name/Open/Close/Stats/Index/Search`). External drivers additionally implement the new optional `index.AddrSetter` interface so the CLI can inject `--addr` before `Open`. Docker services mount data to `$HOME/data/fts/{engine}/`. Embedded drivers (bleve, tantivy-go) need no Docker.

**Tech Stack:** Go 1.25, bleve/v2 (already in go.mod), meilisearch-go (already in go.mod), jackc/pgx/v5 (already in go.mod), ClickHouse/clickhouse-go/v2 (new), anyproto/tantivy-go (new, CGO, build-tagged `tantivy`), Docker Compose v3.

**Spec:** `spec/0644_external_index.md`

---

## Task 1: Add `AddrSetter` interface to engine registry

**Files:**
- Modify: `pkg/index/engine.go`
- Create: `pkg/index/engine_test.go`

### Step 1: Write the failing test

Create `pkg/index/engine_test.go`:

```go
package index_test

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

// fakeExternal is a minimal engine that implements AddrSetter for testing.
type fakeExternal struct {
	addr    string
	opened  bool
	closed  bool
}

func (f *fakeExternal) Name() string                                              { return "fake-external" }
func (f *fakeExternal) Open(_ context.Context, _ string) error                    { f.opened = true; return nil }
func (f *fakeExternal) Close() error                                              { f.closed = true; return nil }
func (f *fakeExternal) Stats(_ context.Context) (index.EngineStats, error)        { return index.EngineStats{}, nil }
func (f *fakeExternal) Index(_ context.Context, _ []index.Document) error         { return nil }
func (f *fakeExternal) Search(_ context.Context, _ index.Query) (index.Results, error) {
	return index.Results{}, nil
}
func (f *fakeExternal) SetAddr(a string) { f.addr = a }

func TestAddrSetter(t *testing.T) {
	eng := &fakeExternal{}
	// Engine should implement AddrSetter
	setter, ok := any(eng).(index.AddrSetter)
	if !ok {
		t.Fatal("fakeExternal does not implement AddrSetter")
	}
	setter.SetAddr("http://localhost:9999")
	if eng.addr != "http://localhost:9999" {
		t.Errorf("SetAddr: got %q, want %q", eng.addr, "http://localhost:9999")
	}
}

func TestRegistry_ListAndNew(t *testing.T) {
	// Register a fresh engine under a unique test name
	name := "test-fake-external-registry"
	index.Register(name, func() index.Engine { return &fakeExternal{} })

	names := index.List()
	found := false
	for _, n := range names {
		if n == name {
			found = true
		}
	}
	if !found {
		t.Errorf("List() does not include registered driver %q; got %v", name, names)
	}

	eng, err := index.NewEngine(name)
	if err != nil {
		t.Fatal(err)
	}
	if eng.Name() == "" {
		t.Error("NewEngine returned engine with empty Name()")
	}

	_, err = index.NewEngine("definitely-not-registered-xyz")
	if err == nil {
		t.Error("expected error for unknown driver, got nil")
	}
}
```

### Step 2: Run tests to see current state

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go test ./pkg/index/... -v -run TestAddrSetter
```

Expected: compile error — `index.AddrSetter` undefined.

### Step 3: Add `AddrSetter` and `baseExternal` to `pkg/index/engine.go`

Append to the end of `pkg/index/engine.go` (after the `List()` function):

```go
// AddrSetter is implemented by external engines that connect to a remote service.
// The CLI calls SetAddr before Open when --addr is provided.
type AddrSetter interface {
	SetAddr(addr string)
}

// BaseExternal provides a default SetAddr implementation for external engines.
// Embed this in external engine structs.
type BaseExternal struct {
	Addr string
}

// SetAddr stores the address.
func (b *BaseExternal) SetAddr(a string) { b.Addr = a }

// EffectiveAddr returns Addr if set, otherwise returns def.
func (b *BaseExternal) EffectiveAddr(def string) string {
	if b.Addr != "" {
		return b.Addr
	}
	return def
}
```

### Step 4: Run tests again

```bash
go test ./pkg/index/... -v -run 'TestAddrSetter|TestRegistry'
```

Expected: PASS

### Step 5: Commit

```bash
git add pkg/index/engine.go pkg/index/engine_test.go
git commit -m "feat(index): add AddrSetter interface and BaseExternal helper"
```

---

## Task 2: Add `--addr` flag to `fts index` and `fts search`

**Files:**
- Modify: `cli/cc_fts.go`

### Step 1: Locate the flag variables and RunE functions

Open `cli/cc_fts.go`. The two commands are `newCCFTSIndex()` and `newCCFTSSearch()`.

### Step 2: Add `--addr` flag to both commands

In `newCCFTSIndex()`, add `addr string` to the var block and a flag:

```go
// In newCCFTSIndex() var block, add:
addr string

// Add flag after existing flags:
cmd.Flags().StringVar(&addr, "addr", "", "Service address for external engines (e.g. http://localhost:7700)")

// Update RunE to pass addr:
return runCCFTSIndex(cmd.Context(), crawlID, engine, source, batchSize, workers, addr)
```

In `newCCFTSSearch()`, add `addr string` and:

```go
// In newCCFTSSearch() var block, add:
addr string

// Add flag:
cmd.Flags().StringVar(&addr, "addr", "", "Service address for external engines")

// Update RunE:
return runCCFTSSearch(cmd.Context(), crawlID, engine, query, limit, offset, addr)
```

### Step 3: Wire SetAddr in `runCCFTSIndex` and `runCCFTSSearch`

In `runCCFTSIndex`, after `eng, err := index.NewEngine(engineName)`, add:

```go
if addr != "" {
    if setter, ok := eng.(index.AddrSetter); ok {
        setter.SetAddr(addr)
    } else {
        fmt.Fprintf(os.Stderr, "warning: engine %q does not support --addr flag\n", engineName)
    }
}
```

Apply the same pattern in `runCCFTSSearch` (same location after `NewEngine`).

Update function signatures:
- `runCCFTSIndex(ctx, crawlID, engineName, source string, batchSize, workers int, addr string)`
- `runCCFTSSearch(ctx, crawlID, engineName, query string, limit, offset int, addr string)`

### Step 4: Add new driver imports to `cc_fts.go`

In the import block, add (below the existing driver imports):

```go
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/bleve"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/meilisearch"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/clickhouse"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/postgres"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/quickwit"
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/tantivy-lnx"
```

Note: `tantivy-go` driver is build-tagged and will be added in Task 9.

### Step 5: Build to verify no compile errors

```bash
go build ./cli/... 2>&1 | head -20
```

Expected: errors about missing packages (not yet created) — that's expected at this stage. Fix: comment out the new imports temporarily with `// TODO: uncomment after driver creation`, then:

```bash
go build ./cli/... 2>&1 | head -5
```

Expected: PASS (compiles)

### Step 6: Commit

```bash
git add cli/cc_fts.go
git commit -m "feat(cli): add --addr flag to fts index/search; wire AddrSetter"
```

---

## Task 3: Bleve embedded FTS driver

**Files:**
- Create: `pkg/index/driver/bleve/bleve.go`
- Create: `pkg/index/driver/bleve/bleve_test.go`

The `blevesearch/bleve/v2` package is already in `go.mod`. No new dependencies needed.

### Step 1: Write the failing test

Create `pkg/index/driver/bleve/bleve_test.go`:

```go
package bleve_test

import (
	"context"
	"testing"

	blevedrv "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/bleve"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func TestBleveEngine_Roundtrip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	eng := blevedrv.NewEngine()
	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	docs := []index.Document{
		{DocID: "doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "doc3", Text: []byte("open source software development programming")},
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := eng.Search(ctx, index.Query{Text: "machine learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("Search: expected hits, got none")
	}
	if results.Hits[0].DocID != "doc1" {
		t.Errorf("Search: expected doc1 as top hit, got %q", results.Hits[0].DocID)
	}

	stats, err := eng.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DocCount != 3 {
		t.Errorf("Stats.DocCount: got %d, want 3", stats.DocCount)
	}
	if stats.DiskBytes == 0 {
		t.Error("Stats.DiskBytes: expected > 0 after indexing")
	}
}

func TestBleveEngine_Name(t *testing.T) {
	eng := blevedrv.NewEngine()
	if eng.Name() != "bleve" {
		t.Errorf("Name: got %q, want %q", eng.Name(), "bleve")
	}
}

func TestBleveEngine_Registered(t *testing.T) {
	names := index.List()
	for _, n := range names {
		if n == "bleve" {
			return
		}
	}
	t.Errorf("bleve not found in registered engines: %v", names)
}
```

### Step 2: Run test to verify it fails

```bash
go test ./pkg/index/driver/bleve/... -v
```

Expected: compile error — package does not exist.

### Step 3: Implement `pkg/index/driver/bleve/bleve.go`

```go
package bleve

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	blevelib "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/en"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("bleve", func() index.Engine { return NewEngine() })
}

// Engine is an embedded BM25 FTS engine backed by Bleve.
type Engine struct {
	idx blevelib.Index
	dir string
}

// NewEngine returns a new bleve Engine. Also called by init().
func NewEngine() *Engine { return &Engine{} }

func (e *Engine) Name() string { return "bleve" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir
	dbPath := filepath.Join(dir, "bleve.db")

	idx, err := blevelib.Open(dbPath)
	if err != nil {
		// Index does not exist — create it
		mapping := blevelib.NewIndexMapping()
		docMapping := blevelib.NewDocumentMapping()

		textField := blevelib.NewTextFieldMapping()
		textField.Analyzer = en.AnalyzerName
		docMapping.AddFieldMappingsAt("text", textField)
		mapping.AddDocumentMapping("doc", docMapping)
		mapping.DefaultMapping = docMapping

		idx, err = blevelib.New(dbPath, mapping)
		if err != nil {
			return fmt.Errorf("bleve create: %w", err)
		}
	}
	e.idx = idx
	return nil
}

func (e *Engine) Close() error {
	if e.idx == nil {
		return nil
	}
	return e.idx.Close()
}

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	count, err := e.idx.DocCount()
	if err != nil {
		return index.EngineStats{}, err
	}
	return index.EngineStats{
		DocCount:  int64(count),
		DiskBytes: index.DirSizeBytes(e.dir),
	}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	b := e.idx.NewBatch()
	for _, doc := range docs {
		if err := b.Index(doc.DocID, struct{ Text string }{Text: string(doc.Text)}); err != nil {
			return fmt.Errorf("bleve batch: %w", err)
		}
	}
	return e.idx.Batch(b)
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	mq := blevelib.NewMatchQuery(q.Text)
	mq.SetField("text")

	req := blevelib.NewSearchRequestOptions(mq, limit, q.Offset, false)
	req.Highlight = blevelib.NewHighlight()
	req.Highlight.AddField("text")

	sr, err := e.idx.SearchInContext(ctx, req)
	if err != nil {
		return index.Results{}, fmt.Errorf("bleve search: %w", err)
	}

	results := index.Results{Total: int(sr.Total)}
	for _, hit := range sr.Hits {
		h := index.Hit{
			DocID: hit.ID,
			Score: hit.Score,
		}
		if frags, ok := hit.Fragments["text"]; ok && len(frags) > 0 {
			h.Snippet = frags[0]
		}
		results.Hits = append(results.Hits, h)
	}
	return results, nil
}

var _ index.Engine = (*Engine)(nil)
```

### Step 4: Run tests

```bash
go test ./pkg/index/driver/bleve/... -v
```

Expected: PASS for all three tests.

### Step 5: Commit

```bash
git add pkg/index/driver/bleve/
git commit -m "feat(index/bleve): embedded BM25 driver with bleve/v2"
```

---

## Task 4: Add `clickhouse-go/v2` dependency and update `go.mod`

**Files:**
- Modify: `go.mod`, `go.sum`

### Step 1: Add ClickHouse Go client

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go get github.com/ClickHouse/clickhouse-go/v2@latest
```

Expected: `go.mod` updated with `github.com/ClickHouse/clickhouse-go/v2`.

### Step 2: Tidy

```bash
go mod tidy
```

### Step 3: Verify build still works

```bash
go build ./... 2>&1 | grep -v "^#" | head -10
```

Expected: no errors (or only the expected missing driver packages if not yet created).

### Step 4: Commit

```bash
git add go.mod go.sum
git commit -m "deps: add ClickHouse/clickhouse-go/v2"
```

---

## Task 5: Docker Compose files for all external engines

**Files:**
- Create: `docker/meilisearch/docker-compose.yml`
- Create: `docker/clickhouse/docker-compose.yml`
- Create: `docker/postgres-fts/docker-compose.yml`
- Create: `docker/quickwit/docker-compose.yml`
- Create: `docker/lnx/docker-compose.yml`

All files use bind mounts to `${FTS_DATA_DIR:-$HOME/data/fts}/{engine}/` so disk usage is easily measurable.

### Step 1: Create `docker/meilisearch/docker-compose.yml`

```yaml
# Meilisearch v1.x — Full-text search engine
# Usage:
#   FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/meilisearch/docker-compose.yml up -d
#   search cc fts index --engine meilisearch
#   search cc fts search "machine learning" --engine meilisearch

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
      - ${FTS_DATA_DIR:-${HOME}/data/fts}/meilisearch:/meili_data
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:7700/health | grep -q available"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 20s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 4G
        reservations:
          memory: 512M
```

### Step 2: Create `docker/clickhouse/docker-compose.yml`

```yaml
# ClickHouse 25.x — OLAP database with inverted index FTS
# Usage:
#   FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/clickhouse/docker-compose.yml up -d
#   search cc fts index --engine clickhouse

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
      - ${FTS_DATA_DIR:-${HOME}/data/fts}/clickhouse:/var/lib/clickhouse
    ulimits:
      nofile:
        soft: 262144
        hard: 262144
    healthcheck:
      test: ["CMD-SHELL", "clickhouse-client --user fts --password fts --query 'SELECT 1'"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 4G
        reservations:
          memory: 512M
```

### Step 3: Create `docker/postgres-fts/docker-compose.yml`

```yaml
# PostgreSQL 17 — native tsvector/GIN full-text search
# Usage:
#   FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/postgres-fts/docker-compose.yml up -d
#   search cc fts index --engine postgres
#   search cc fts search "machine learning" --engine postgres

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
      -c synchronous_commit=off
    volumes:
      - ${FTS_DATA_DIR:-${HOME}/data/fts}/postgres:/var/lib/postgresql/data
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
        reservations:
          memory: 512M
```

### Step 4: Create `docker/quickwit/docker-compose.yml`

```yaml
# Quickwit 0.9 — distributed search engine built on Tantivy
# Usage:
#   FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/quickwit/docker-compose.yml up -d
#   search cc fts index --engine quickwit

services:
  quickwit:
    image: quickwit/quickwit:0.9
    container_name: fts-quickwit
    ports:
      - "7280:7280"   # REST API
      - "7281:7281"   # gRPC
    environment:
      - NO_COLOR=1
    command: ["run"]
    volumes:
      - ${FTS_DATA_DIR:-${HOME}/data/fts}/quickwit:/quickwit/qwdata
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:7280/api/v1/version 2>/dev/null | grep -q version"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 4G
        reservations:
          memory: 512M
```

### Step 5: Create `docker/lnx/docker-compose.yml`

```yaml
# lnx — Tantivy-backed lightweight REST search server
# Usage:
#   FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/lnx/docker-compose.yml up -d
#   search cc fts index --engine tantivy-lnx

services:
  lnx:
    image: ghcr.io/lnx-search/lnx:latest
    container_name: fts-lnx
    ports:
      - "8000:8000"
    environment:
      - LNX_LOG_LEVEL=info
      - LNX_DATA_PATH=/var/lib/lnx
    volumes:
      - ${FTS_DATA_DIR:-${HOME}/data/fts}/lnx:/var/lib/lnx
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:8000/api/v1/indexes 2>/dev/null || exit 0"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 4G
        reservations:
          memory: 512M
```

### Step 6: Create data dirs and verify docker compose syntax

```bash
mkdir -p $HOME/data/fts/{meilisearch,clickhouse,postgres,quickwit,lnx}

docker compose -f /Users/apple/github/go-mizu/mizu/blueprints/search/docker/meilisearch/docker-compose.yml config --quiet && echo "meilisearch OK"
docker compose -f /Users/apple/github/go-mizu/mizu/blueprints/search/docker/clickhouse/docker-compose.yml config --quiet && echo "clickhouse OK"
docker compose -f /Users/apple/github/go-mizu/mizu/blueprints/search/docker/postgres-fts/docker-compose.yml config --quiet && echo "postgres OK"
docker compose -f /Users/apple/github/go-mizu/mizu/blueprints/search/docker/quickwit/docker-compose.yml config --quiet && echo "quickwit OK"
docker compose -f /Users/apple/github/go-mizu/mizu/blueprints/search/docker/lnx/docker-compose.yml config --quiet && echo "lnx OK"
```

Expected: all five print `OK`.

### Step 7: Commit

```bash
git add docker/meilisearch/ docker/clickhouse/ docker/postgres-fts/ docker/quickwit/ docker/lnx/
git commit -m "docker: add compose files for meilisearch, clickhouse, postgres, quickwit, lnx"
```

---

## Task 6: Meilisearch driver

**Files:**
- Create: `pkg/index/driver/meilisearch/meilisearch.go`
- Create: `pkg/index/driver/meilisearch/meilisearch_test.go`

### Step 1: Write the failing test

Create `pkg/index/driver/meilisearch/meilisearch_test.go`:

```go
package meilisearch_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	msdrv "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/meilisearch"
)

func skipIfMeilisearchDown(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:7700/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("meilisearch not available at localhost:7700: %v", err)
	}
	resp.Body.Close()
}

func TestMeilisearchEngine_Roundtrip(t *testing.T) {
	skipIfMeilisearchDown(t)
	ctx := context.Background()
	dir := t.TempDir()

	eng := msdrv.NewEngine()
	eng.SetAddr("http://localhost:7700")
	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	// Clean up index from previous test runs
	eng.DeleteIndex(ctx)
	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Re-Open after delete: %v", err)
	}

	docs := []index.Document{
		{DocID: "ms-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "ms-doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "ms-doc3", Text: []byte("open source software development programming")},
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// Meilisearch indexing is sync in our driver (waits for task)
	results, err := eng.Search(ctx, index.Query{Text: "machine learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("Search: expected hits, got none")
	}

	stats, err := eng.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DocCount != 3 {
		t.Errorf("DocCount: got %d, want 3", stats.DocCount)
	}
}

func TestMeilisearchEngine_AddrSetter(t *testing.T) {
	eng := msdrv.NewEngine()
	setter, ok := any(eng).(index.AddrSetter)
	if !ok {
		t.Fatal("meilisearch.Engine does not implement index.AddrSetter")
	}
	setter.SetAddr("http://my-server:7700")
}
```

### Step 2: Run test to verify it fails (or skips)

```bash
go test ./pkg/index/driver/meilisearch/... -v -run TestMeilisearchEngine
```

Expected: compile error — package does not exist.

### Step 3: Implement `pkg/index/driver/meilisearch/meilisearch.go`

```go
package meilisearch

import (
	"context"
	"fmt"
	"os"

	ms "github.com/meilisearch/meilisearch-go"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

const (
	defaultAddr  = "http://localhost:7700"
	indexName    = "fts_docs"
)

func init() {
	index.Register("meilisearch", func() index.Engine { return NewEngine() })
}

// Engine is an external FTS driver backed by Meilisearch.
type Engine struct {
	index.BaseExternal
	client ms.ServiceManager
	idx    ms.IndexManager
}

// NewEngine returns a new Meilisearch Engine.
func NewEngine() *Engine { return &Engine{} }

func (e *Engine) Name() string { return "meilisearch" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	addr := e.EffectiveAddr(defaultAddr)
	e.client = ms.New(addr)
	e.idx = e.client.Index(indexName)

	// Ensure the index exists
	if _, err := e.client.GetIndex(indexName); err != nil {
		task, err := e.client.CreateIndex(&ms.IndexConfig{
			Uid:        indexName,
			PrimaryKey: "doc_id",
		})
		if err != nil {
			return fmt.Errorf("meilisearch create index: %w", err)
		}
		if _, err := e.client.WaitForTask(task.TaskUID, 0); err != nil {
			return fmt.Errorf("meilisearch wait create: %w", err)
		}
	}

	// Set searchable attributes
	task, err := e.idx.UpdateSearchableAttributes(&[]string{"text"})
	if err != nil {
		return fmt.Errorf("meilisearch set searchable attrs: %w", err)
	}
	if _, err := e.client.WaitForTask(task.TaskUID, 0); err != nil {
		return fmt.Errorf("meilisearch wait attrs: %w", err)
	}
	return nil
}

// DeleteIndex removes the index. Useful for test cleanup.
func (e *Engine) DeleteIndex(ctx context.Context) {
	if e.client != nil {
		task, _ := e.client.DeleteIndex(indexName)
		if task != nil {
			e.client.WaitForTask(task.TaskUID, 0) //nolint:errcheck
		}
	}
}

func (e *Engine) Close() error { return nil }

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	stats, err := e.idx.GetStats()
	if err != nil {
		return index.EngineStats{}, err
	}
	return index.EngineStats{
		DocCount:  int64(stats.NumberOfDocuments),
		DiskBytes: 0, // not exposed via API
	}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	payload := make([]map[string]any, len(docs))
	for i, d := range docs {
		payload[i] = map[string]any{
			"doc_id": d.DocID,
			"text":   string(d.Text),
		}
	}
	task, err := e.idx.AddDocuments(payload, "doc_id")
	if err != nil {
		return fmt.Errorf("meilisearch add docs: %w", err)
	}
	if _, err := e.client.WaitForTask(task.TaskUID, 0); err != nil {
		return fmt.Errorf("meilisearch wait index: %w", err)
	}
	return nil
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := int64(q.Limit)
	if limit <= 0 {
		limit = 10
	}
	res, err := e.idx.Search(q.Text, &ms.SearchRequest{
		Limit:                  limit,
		Offset:                 int64(q.Offset),
		AttributesToHighlight:  []string{"text"},
		HighlightPreTag:        "",
		HighlightPostTag:       "",
	})
	if err != nil {
		return index.Results{}, fmt.Errorf("meilisearch search: %w", err)
	}
	results := index.Results{Total: int(res.EstimatedTotalHits)}
	for _, hit := range res.Hits {
		m, ok := hit.(map[string]any)
		if !ok {
			continue
		}
		h := index.Hit{Score: 1.0}
		if v, ok := m["doc_id"].(string); ok {
			h.DocID = v
		}
		if v, ok := m["text"].(string); ok && len(v) > 200 {
			h.Snippet = v[:200]
		} else if ok {
			h.Snippet = v
		}
		results.Hits = append(results.Hits, h)
	}
	return results, nil
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
```

### Step 4: Run tests

Start meilisearch first:
```bash
FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/meilisearch/docker-compose.yml up -d
# Wait for healthy
docker ps --filter name=fts-meilisearch --format "{{.Status}}"
```

Then run tests:
```bash
go test ./pkg/index/driver/meilisearch/... -v -run TestMeilisearchEngine -timeout 60s
```

Expected: PASS (or SKIP if service not running).

### Step 5: Commit

```bash
git add pkg/index/driver/meilisearch/
git commit -m "feat(index/meilisearch): external HTTP driver with meilisearch-go"
```

---

## Task 7: ClickHouse driver

**Files:**
- Create: `pkg/index/driver/clickhouse/clickhouse.go`
- Create: `pkg/index/driver/clickhouse/clickhouse_test.go`

### Step 1: Write the failing test

Create `pkg/index/driver/clickhouse/clickhouse_test.go`:

```go
package clickhouse_test

import (
	"context"
	"net"
	"testing"
	"time"

	chdrv "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/clickhouse"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func skipIfClickHouseDown(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "localhost:9000", 2*time.Second)
	if err != nil {
		t.Skipf("clickhouse not available at localhost:9000: %v", err)
	}
	conn.Close()
}

func TestClickHouseEngine_Roundtrip(t *testing.T) {
	skipIfClickHouseDown(t)
	ctx := context.Background()
	dir := t.TempDir()

	eng := chdrv.NewEngine()
	eng.SetAddr("localhost:9000")
	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	// Drop and recreate for clean test state
	eng.DropTable(ctx)
	if err := eng.CreateTable(ctx); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	docs := []index.Document{
		{DocID: "ch-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "ch-doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "ch-doc3", Text: []byte("open source software development programming")},
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := eng.Search(ctx, index.Query{Text: "machine", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("Search: expected hits, got none")
	}

	stats, err := eng.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DocCount != 3 {
		t.Errorf("DocCount: got %d, want 3", stats.DocCount)
	}
}

func TestClickHouseEngine_AddrSetter(t *testing.T) {
	eng := chdrv.NewEngine()
	_, ok := any(eng).(index.AddrSetter)
	if !ok {
		t.Fatal("clickhouse.Engine does not implement AddrSetter")
	}
}
```

### Step 2: Run to verify compile failure

```bash
go test ./pkg/index/driver/clickhouse/... -v
```

### Step 3: Implement `pkg/index/driver/clickhouse/clickhouse.go`

```go
package clickhouse

import (
	"context"
	"fmt"
	"os"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

const (
	defaultAddr  = "localhost:9000"
	defaultDB    = "fts"
	defaultUser  = "fts"
	defaultPass  = "fts"
	tableName    = "fts_docs"
)

func init() {
	index.Register("clickhouse", func() index.Engine { return NewEngine() })
}

// Engine is an external FTS driver backed by ClickHouse.
type Engine struct {
	index.BaseExternal
	conn ch.Conn
	dir  string
}

// NewEngine returns a new ClickHouse Engine.
func NewEngine() *Engine { return &Engine{} }

func (e *Engine) Name() string { return "clickhouse" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir
	addr := e.EffectiveAddr(defaultAddr)

	conn, err := ch.Open(&ch.Options{
		Addr: []string{addr},
		Auth: ch.Auth{
			Database: defaultDB,
			Username: defaultUser,
			Password: defaultPass,
		},
		Settings: ch.Settings{
			"allow_experimental_inverted_index":   1,
			"allow_experimental_full_text_index":  1,
		},
		MaxOpenConns:     4,
		MaxIdleConns:     2,
	})
	if err != nil {
		return fmt.Errorf("clickhouse open %s: %w", addr, err)
	}
	if err := conn.Ping(ctx); err != nil {
		return fmt.Errorf("clickhouse ping: %w", err)
	}
	e.conn = conn
	return e.CreateTable(ctx)
}

// CreateTable creates the fts_docs table if it does not exist.
func (e *Engine) CreateTable(ctx context.Context) error {
	return e.conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS fts_docs (
			doc_id String,
			text   String,
			INDEX  text_idx text TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1
		) ENGINE = MergeTree()
		ORDER BY doc_id
		SETTINGS index_granularity = 8192
	`)
}

// DropTable drops the fts_docs table. Used during tests.
func (e *Engine) DropTable(ctx context.Context) {
	e.conn.Exec(ctx, "DROP TABLE IF EXISTS fts_docs") //nolint:errcheck
}

func (e *Engine) Close() error {
	if e.conn == nil {
		return nil
	}
	return e.conn.Close()
}

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	var count uint64
	if err := e.conn.QueryRow(ctx, "SELECT count() FROM fts_docs").Scan(&count); err != nil {
		return index.EngineStats{}, err
	}
	var diskBytes uint64
	e.conn.QueryRow(ctx,
		"SELECT sum(data_compressed_bytes) FROM system.parts WHERE table='fts_docs' AND active",
	).Scan(&diskBytes) //nolint:errcheck
	return index.EngineStats{DocCount: int64(count), DiskBytes: int64(diskBytes)}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	batch, err := e.conn.PrepareBatch(ctx, "INSERT INTO fts_docs (doc_id, text)")
	if err != nil {
		return fmt.Errorf("clickhouse prepare batch: %w", err)
	}
	for _, doc := range docs {
		if err := batch.Append(doc.DocID, string(doc.Text)); err != nil {
			return fmt.Errorf("clickhouse append: %w", err)
		}
	}
	return batch.Send()
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	rows, err := e.conn.Query(ctx,
		`SELECT doc_id, substring(text, 1, 200) AS snippet, 1.0 AS score
		 FROM fts_docs
		 WHERE hasTokenCaseInsensitive(text, ?)
		 ORDER BY doc_id
		 LIMIT ? OFFSET ?`,
		q.Text, limit, q.Offset,
	)
	if err != nil {
		return index.Results{}, fmt.Errorf("clickhouse search: %w", err)
	}
	defer rows.Close()

	var results index.Results
	for rows.Next() {
		var h index.Hit
		if err := rows.Scan(&h.DocID, &h.Snippet, &h.Score); err != nil {
			return results, err
		}
		results.Hits = append(results.Hits, h)
	}
	results.Total = len(results.Hits)
	return results, rows.Err()
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
```

### Step 4: Run tests

Start ClickHouse:
```bash
FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/clickhouse/docker-compose.yml up -d
```

Run tests:
```bash
go test ./pkg/index/driver/clickhouse/... -v -timeout 30s
```

Expected: PASS (or SKIP if service not running).

### Step 5: Commit

```bash
git add pkg/index/driver/clickhouse/
git commit -m "feat(index/clickhouse): external TCP driver with clickhouse-go/v2"
```

---

## Task 8: PostgreSQL native FTS driver

**Files:**
- Create: `pkg/index/driver/postgres/postgres.go`
- Create: `pkg/index/driver/postgres/postgres_test.go`

Uses `jackc/pgx/v5` with `COPY FROM` for batch inserts and `tsvector/GIN` for search.

### Step 1: Write the failing test

Create `pkg/index/driver/postgres/postgres_test.go`:

```go
package postgres_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	pgdrv "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/postgres"
)

func skipIfPostgresDown(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "localhost:5432", 2*time.Second)
	if err != nil {
		t.Skipf("postgres not available at localhost:5432: %v", err)
	}
	conn.Close()
}

func TestPostgresEngine_Roundtrip(t *testing.T) {
	skipIfPostgresDown(t)
	ctx := context.Background()
	dir := t.TempDir()

	eng := pgdrv.NewEngine()
	eng.SetAddr("postgres://fineweb:fineweb@localhost:5432/fts")
	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	eng.DropTable(ctx)
	if err := eng.CreateTable(ctx); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	docs := []index.Document{
		{DocID: "pg-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "pg-doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "pg-doc3", Text: []byte("open source software development programming")},
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := eng.Search(ctx, index.Query{Text: "machine learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("Search: expected hits, got none")
	}
	if results.Hits[0].DocID != "pg-doc1" {
		t.Errorf("expected pg-doc1 as top hit, got %q", results.Hits[0].DocID)
	}

	stats, err := eng.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DocCount != 3 {
		t.Errorf("DocCount: got %d, want 3", stats.DocCount)
	}
}

func TestPostgresEngine_AddrSetter(t *testing.T) {
	eng := pgdrv.NewEngine()
	if _, ok := any(eng).(index.AddrSetter); !ok {
		t.Fatal("postgres.Engine does not implement AddrSetter")
	}
}
```

### Step 2: Verify compile failure

```bash
go test ./pkg/index/driver/postgres/... -v
```

### Step 3: Implement `pkg/index/driver/postgres/postgres.go`

```go
package postgres

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

const defaultAddr = "postgres://fineweb:fineweb@localhost:5432/fts"

func init() {
	index.Register("postgres", func() index.Engine { return NewEngine() })
}

// Engine is an external FTS driver backed by PostgreSQL tsvector/GIN.
type Engine struct {
	index.BaseExternal
	pool *pgxpool.Pool
	dir  string
}

// NewEngine returns a new PostgreSQL Engine.
func NewEngine() *Engine { return &Engine{} }

func (e *Engine) Name() string { return "postgres" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir
	dsn := e.EffectiveAddr(defaultAddr)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("postgres connect %s: %w", dsn, err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("postgres ping: %w", err)
	}
	e.pool = pool
	return e.CreateTable(ctx)
}

// CreateTable creates the fts_docs table with a GIN index on the tsv column.
func (e *Engine) CreateTable(ctx context.Context) error {
	_, err := e.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS fts_docs (
			doc_id TEXT PRIMARY KEY,
			text   TEXT,
			tsv    TSVECTOR GENERATED ALWAYS AS (to_tsvector('english', text)) STORED
		)
	`)
	if err != nil {
		return fmt.Errorf("postgres create table: %w", err)
	}
	_, err = e.pool.Exec(ctx,
		`CREATE INDEX IF NOT EXISTS fts_docs_tsv_idx ON fts_docs USING GIN(tsv)`)
	return err
}

// DropTable drops fts_docs. Used during tests.
func (e *Engine) DropTable(ctx context.Context) {
	e.pool.Exec(ctx, "DROP TABLE IF EXISTS fts_docs") //nolint:errcheck
}

func (e *Engine) Close() error {
	if e.pool != nil {
		e.pool.Close()
	}
	return nil
}

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	var count int64
	if err := e.pool.QueryRow(ctx, "SELECT count(*) FROM fts_docs").Scan(&count); err != nil {
		return index.EngineStats{}, err
	}
	var diskBytes int64
	e.pool.QueryRow(ctx, "SELECT pg_total_relation_size('fts_docs')").Scan(&diskBytes) //nolint:errcheck
	return index.EngineStats{DocCount: count, DiskBytes: diskBytes}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	// Use COPY protocol for maximum throughput
	rows := make([][]any, len(docs))
	for i, d := range docs {
		rows[i] = []any{d.DocID, string(d.Text)}
	}
	_, err := e.pool.CopyFrom(ctx,
		pgx.Identifier{"fts_docs"},
		[]string{"doc_id", "text"},
		pgx.CopyFromRows(rows),
	)
	return err
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	sqlStr := `
		SELECT doc_id,
		       ts_headline('english', text,
		           websearch_to_tsquery('english', $1),
		           'MaxFragments=1,MaxWords=20') AS snippet,
		       ts_rank_cd(tsv, websearch_to_tsquery('english', $1)) AS score
		FROM fts_docs
		WHERE tsv @@ websearch_to_tsquery('english', $1)
		ORDER BY score DESC
		LIMIT $2 OFFSET $3`

	rows, err := e.pool.Query(ctx, sqlStr, q.Text, limit, q.Offset)
	if err != nil {
		return index.Results{}, fmt.Errorf("postgres search: %w", err)
	}
	defer rows.Close()

	var results index.Results
	for rows.Next() {
		var h index.Hit
		if err := rows.Scan(&h.DocID, &h.Snippet, &h.Score); err != nil {
			return results, err
		}
		results.Hits = append(results.Hits, h)
	}
	results.Total = len(results.Hits)
	return results, rows.Err()
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
```

### Step 4: Run tests

Start postgres:
```bash
FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/postgres-fts/docker-compose.yml up -d
```

Run tests:
```bash
go test ./pkg/index/driver/postgres/... -v -timeout 60s
```

Expected: PASS (or SKIP if service not running).

### Step 5: Commit

```bash
git add pkg/index/driver/postgres/
git commit -m "feat(index/postgres): external driver with pgx/v5 COPY + tsvector/GIN"
```

---

## Task 9: Quickwit driver

**Files:**
- Create: `pkg/index/driver/quickwit/quickwit.go`
- Create: `pkg/index/driver/quickwit/quickwit_test.go`

Custom HTTP client — no SDK. Quickwit REST API uses NDJSON ingest.

### Step 1: Write the failing test

Create `pkg/index/driver/quickwit/quickwit_test.go`:

```go
package quickwit_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	qwdrv "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/quickwit"
)

func skipIfQuickwitDown(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:7280/api/v1/version", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("quickwit not available at localhost:7280: %v", err)
	}
	resp.Body.Close()
}

func TestQuickwitEngine_Roundtrip(t *testing.T) {
	skipIfQuickwitDown(t)
	ctx := context.Background()
	dir := t.TempDir()

	eng := qwdrv.NewEngine()
	eng.SetAddr("http://localhost:7280")
	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	docs := []index.Document{
		{DocID: "qw-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "qw-doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "qw-doc3", Text: []byte("open source software development programming")},
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// Quickwit uses commit=force so docs are immediately searchable
	results, err := eng.Search(ctx, index.Query{Text: "machine learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("Search: expected hits, got none")
	}

	stats, err := eng.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DocCount < 3 {
		t.Errorf("DocCount: got %d, want >= 3", stats.DocCount)
	}
}

func TestQuickwitEngine_AddrSetter(t *testing.T) {
	eng := qwdrv.NewEngine()
	if _, ok := any(eng).(index.AddrSetter); !ok {
		t.Fatal("quickwit.Engine does not implement AddrSetter")
	}
}
```

### Step 2: Implement `pkg/index/driver/quickwit/quickwit.go`

```go
package quickwit

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
	defaultAddr = "http://localhost:7280"
	indexID     = "fts_docs"
)

func init() {
	index.Register("quickwit", func() index.Engine { return NewEngine() })
}

// Engine is an external FTS driver backed by Quickwit (Tantivy).
type Engine struct {
	index.BaseExternal
	client *http.Client
	base   string
	dir    string
}

// NewEngine returns a new Quickwit Engine.
func NewEngine() *Engine {
	return &Engine{client: &http.Client{Timeout: 120 * time.Second}}
}

func (e *Engine) Name() string { return "quickwit" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir
	e.base = e.EffectiveAddr(defaultAddr)

	// Create index if it does not exist
	indexSchema := map[string]any{
		"index_id": indexID,
		"doc_mapping": map[string]any{
			"field_mappings": []any{
				map[string]any{"name": "doc_id", "type": "text", "tokenizer": "raw", "stored": true},
				map[string]any{
					"name": "text", "type": "text", "tokenizer": "default",
					"stored": true, "record": "position", "fieldnorms": true,
				},
			},
		},
		"search_settings": map[string]any{
			"default_search_fields": []string{"text"},
		},
	}
	body, _ := json.Marshal(indexSchema)
	req, _ := http.NewRequestWithContext(ctx, "POST", e.base+"/api/v1/indexes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("quickwit create index: %w", err)
	}
	resp.Body.Close()
	// 200 = created, 400 = already exists — both OK
	return nil
}

func (e *Engine) Close() error { return nil }

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", e.base+"/api/v1/indexes/"+indexID, nil)
	resp, err := e.client.Do(req)
	if err != nil {
		return index.EngineStats{}, err
	}
	defer resp.Body.Close()
	var info struct {
		IndexConfig struct{} `json:"index_config"`
		Splits      []struct {
			NumDocs int64 `json:"num_docs"`
		} `json:"splits"`
	}
	json.NewDecoder(resp.Body).Decode(&info) //nolint:errcheck
	var total int64
	for _, s := range info.Splits {
		total += s.NumDocs
	}
	return index.EngineStats{DocCount: total}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	for _, d := range docs {
		enc.Encode(map[string]string{"doc_id": d.DocID, "text": string(d.Text)}) //nolint:errcheck
	}
	url := fmt.Sprintf("%s/api/v1/%s/ingest?commit=force", e.base, indexID)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(sb.String()))
	req.Header.Set("Content-Type", "application/x-ndjson")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("quickwit ingest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("quickwit ingest HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	payload := map[string]any{
		"query":          q.Text,
		"max_hits":       limit,
		"start_offset":   q.Offset,
		"snippet_fields": []string{"text"},
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/%s/search", e.base, indexID)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return index.Results{}, fmt.Errorf("quickwit search: %w", err)
	}
	defer resp.Body.Close()

	var sr struct {
		Hits      []map[string]any `json:"hits"`
		NumHits   int              `json:"num_hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return index.Results{}, err
	}
	results := index.Results{Total: sr.NumHits}
	for _, hit := range sr.Hits {
		h := index.Hit{Score: 1.0}
		if v, ok := hit["doc_id"].(string); ok {
			h.DocID = v
		}
		if snips, ok := hit["_snippets"].(map[string]any); ok {
			if texts, ok := snips["text"].([]any); ok && len(texts) > 0 {
				h.Snippet, _ = texts[0].(string)
			}
		}
		results.Hits = append(results.Hits, h)
	}
	return results, nil
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
```

### Step 3: Run tests

```bash
FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/quickwit/docker-compose.yml up -d
go test ./pkg/index/driver/quickwit/... -v -timeout 60s
```

Expected: PASS (or SKIP).

### Step 4: Commit

```bash
git add pkg/index/driver/quickwit/
git commit -m "feat(index/quickwit): external HTTP REST driver (Tantivy-backed)"
```

---

## Task 10: Tantivy-lnx driver

**Files:**
- Create: `pkg/index/driver/tantivy-lnx/lnx.go`
- Create: `pkg/index/driver/tantivy-lnx/lnx_test.go`

Note: Go package names cannot contain hyphens. The directory is `tantivy-lnx/`, the Go package is `lnx`.

### Step 1: Write the failing test

Create `pkg/index/driver/tantivy-lnx/lnx_test.go`:

```go
package lnx_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	lnxdrv "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/tantivy-lnx"
)

func skipIfLnxDown(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:8000/api/v1/indexes", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("lnx not available at localhost:8000: %v", err)
	}
	resp.Body.Close()
}

func TestLnxEngine_Roundtrip(t *testing.T) {
	skipIfLnxDown(t)
	ctx := context.Background()
	dir := t.TempDir()

	eng := lnxdrv.NewEngine()
	eng.SetAddr("http://localhost:8000")
	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	docs := []index.Document{
		{DocID: "lnx-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "lnx-doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "lnx-doc3", Text: []byte("open source software development programming")},
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := eng.Search(ctx, index.Query{Text: "machine learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("Search: expected hits, got none")
	}
}

func TestLnxEngine_AddrSetter(t *testing.T) {
	eng := lnxdrv.NewEngine()
	if _, ok := any(eng).(index.AddrSetter); !ok {
		t.Fatal("lnx.Engine does not implement AddrSetter")
	}
}
```

### Step 2: Implement `pkg/index/driver/tantivy-lnx/lnx.go`

```go
package lnx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

const (
	defaultAddr = "http://localhost:8000"
	lnxIndex    = "fts_docs"
)

func init() {
	index.Register("tantivy-lnx", func() index.Engine { return NewEngine() })
}

// Engine is an external FTS driver backed by lnx (Tantivy REST server).
type Engine struct {
	index.BaseExternal
	client *http.Client
	base   string
	dir    string
}

// NewEngine returns a new lnx Engine.
func NewEngine() *Engine {
	return &Engine{client: &http.Client{Timeout: 120 * time.Second}}
}

func (e *Engine) Name() string { return "tantivy-lnx" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir
	e.base = e.EffectiveAddr(defaultAddr)

	schema := map[string]any{
		"override_if_exists": false,
		"index": map[string]any{
			"name":                 lnxIndex,
			"writer_threads":       4,
			"writer_heap_size_bytes": 67108864,
			"reader_threads":       4,
			"max_concurrency":      10,
			"search_fields":        []string{"text"},
			"store_records":        true,
			"fields": map[string]any{
				"doc_id": map[string]any{"type": "text", "stored": true},
				"text":   map[string]any{"type": "text", "stored": false},
			},
		},
	}
	body, _ := json.Marshal(schema)
	req, _ := http.NewRequestWithContext(ctx, "POST", e.base+"/api/v1/indexes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("lnx create index: %w", err)
	}
	resp.Body.Close()
	// 200 or 400 (already exists) both acceptable
	return nil
}

func (e *Engine) Close() error { return nil }

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	url := fmt.Sprintf("%s/api/v1/indexes/%s/summary", e.base, lnxIndex)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := e.client.Do(req)
	if err != nil {
		return index.EngineStats{}, err
	}
	defer resp.Body.Close()
	var info struct {
		NumDocs int64 `json:"num_docs"`
	}
	json.NewDecoder(resp.Body).Decode(&info) //nolint:errcheck
	return index.EngineStats{DocCount: info.NumDocs}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	payload := make([]map[string]string, len(docs))
	for i, d := range docs {
		payload[i] = map[string]string{"doc_id": d.DocID, "text": string(d.Text)}
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/indexes/%s/documents", e.base, lnxIndex)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("lnx add docs: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("lnx add docs HTTP %d: %s", resp.StatusCode, b)
	}
	// Commit to make documents searchable
	return e.commit(ctx)
}

func (e *Engine) commit(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/indexes/%s/commit", e.base, lnxIndex)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, nil)
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("lnx commit: %w", err)
	}
	resp.Body.Close()
	return nil
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	payload := map[string]any{
		"query":  q.Text,
		"limit":  limit,
		"offset": q.Offset,
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/indexes/%s/search", e.base, lnxIndex)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return index.Results{}, fmt.Errorf("lnx search: %w", err)
	}
	defer resp.Body.Close()

	var sr struct {
		Data []struct {
			DocID string  `json:"doc_id"`
			Score float64 `json:"_score"`
		} `json:"data"`
		Count int `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return index.Results{}, err
	}
	results := index.Results{Total: sr.Count}
	for _, hit := range sr.Data {
		results.Hits = append(results.Hits, index.Hit{DocID: hit.DocID, Score: hit.Score})
	}
	return results, nil
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
```

### Step 3: Run tests

```bash
FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/lnx/docker-compose.yml up -d
go test ./pkg/index/driver/tantivy-lnx/... -v -timeout 60s
```

Expected: PASS (or SKIP).

### Step 4: Commit

```bash
git add "pkg/index/driver/tantivy-lnx/"
git commit -m "feat(index/tantivy-lnx): lnx HTTP REST driver (Tantivy-backed)"
```

---

## Task 11: Tantivy-go CGO embedded driver

**Files:**
- Create: `pkg/index/driver/tantivy-go/tantivy.go`
- Create: `pkg/index/driver/tantivy-go/tantivy_test.go`

Build-tagged with `//go:build tantivy`. Requires Rust toolchain and `anyproto/tantivy-go`.

### Step 1: Add the dependency

```bash
go get github.com/anyproto/tantivy-go@latest
go mod tidy
```

Note: `tantivy-go` requires CGO and a Rust build environment. The library builds Tantivy from source during `go get` (or uses prebuilt binaries if available). This step may take several minutes.

### Step 2: Write the failing test

Create `pkg/index/driver/tantivy-go/tantivy_test.go`:

```go
//go:build tantivy

package tantivy_test

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/tantivy-go"
)

func TestTantivyEngine_Roundtrip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	eng, err := index.NewEngine("tantivy")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	docs := []index.Document{
		{DocID: "tv-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "tv-doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "tv-doc3", Text: []byte("open source software development programming")},
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := eng.Search(ctx, index.Query{Text: "machine learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("Search: expected hits, got none")
	}

	stats, err := eng.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DocCount != 3 {
		t.Errorf("DocCount: got %d, want 3", stats.DocCount)
	}
}
```

### Step 3: Implement `pkg/index/driver/tantivy-go/tantivy.go`

```go
//go:build tantivy

package tantivy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tantivy "github.com/anyproto/tantivy-go"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("tantivy", func() index.Engine { return &Engine{} })
}

// Engine is an embedded FTS engine backed by Tantivy via CGO.
type Engine struct {
	idx *tantivy.Index
	dir string
}

func (e *Engine) Name() string { return "tantivy" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir

	sb := tantivy.NewSchemaBuilder()
	if err := sb.AddTextField(
		"doc_id",
		tantivy.IndexRecordOptionWithFreqsAndPositions,
		true, // stored
	); err != nil {
		return fmt.Errorf("tantivy schema doc_id: %w", err)
	}
	if err := sb.AddTextField(
		"text",
		tantivy.IndexRecordOptionWithFreqsAndPositions,
		false, // not stored (save space)
	); err != nil {
		return fmt.Errorf("tantivy schema text: %w", err)
	}

	schema, err := sb.Build()
	if err != nil {
		return fmt.Errorf("tantivy build schema: %w", err)
	}

	idxPath := filepath.Join(dir, "tantivy")
	if err := os.MkdirAll(idxPath, 0o755); err != nil {
		return err
	}

	idx, err := tantivy.NewIndexWithPath(schema, idxPath)
	if err != nil {
		return fmt.Errorf("tantivy open index: %w", err)
	}
	e.idx = idx
	return nil
}

func (e *Engine) Close() error {
	if e.idx != nil {
		e.idx.Free()
	}
	return nil
}

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	if e.idx == nil {
		return index.EngineStats{}, nil
	}
	count, err := e.idx.NumDocs()
	if err != nil {
		return index.EngineStats{}, err
	}
	return index.EngineStats{
		DocCount:  int64(count),
		DiskBytes: index.DirSizeBytes(e.dir),
	}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	writer, err := e.idx.Writer(50_000_000) // 50MB heap
	if err != nil {
		return fmt.Errorf("tantivy writer: %w", err)
	}
	for _, doc := range docs {
		d, err := e.idx.ParseDocument(
			fmt.Sprintf(`{"doc_id":%q,"text":%q}`, doc.DocID, string(doc.Text)))
		if err != nil {
			return fmt.Errorf("tantivy parse doc: %w", err)
		}
		if err := writer.AddDocument(d); err != nil {
			return fmt.Errorf("tantivy add doc: %w", err)
		}
	}
	return writer.Commit()
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := uint32(q.Limit)
	if limit == 0 {
		limit = 10
	}

	searcher, err := e.idx.Searcher()
	if err != nil {
		return index.Results{}, fmt.Errorf("tantivy searcher: %w", err)
	}
	defer searcher.Free()

	query, err := e.idx.ParseQuery(q.Text, []string{"text"})
	if err != nil {
		return index.Results{}, fmt.Errorf("tantivy parse query: %w", err)
	}
	defer query.Free()

	result, err := searcher.Search(query, limit, true, "doc_id", uint32(q.Offset))
	if err != nil {
		return index.Results{}, fmt.Errorf("tantivy search: %w", err)
	}

	results := index.Results{Total: int(result.Size)}
	for _, hit := range result.Hits {
		results.Hits = append(results.Hits, index.Hit{
			DocID: hit.ID,
			Score: float64(hit.Score),
		})
	}
	return results, nil
}

var _ index.Engine = (*Engine)(nil)
```

### Step 4: Run tests with build tag

```bash
go test -tags tantivy ./pkg/index/driver/tantivy-go/... -v -timeout 30s
```

Expected: PASS (Rust/CGO build may be slow first run).

### Step 5: Update go.mod and commit

```bash
go mod tidy
git add pkg/index/driver/tantivy-go/ go.mod go.sum
git commit -m "feat(index/tantivy-go): embedded CGO driver via anyproto/tantivy-go (build-tag: tantivy)"
```

---

## Task 12: Wire new drivers into CLI imports

**Files:**
- Modify: `cli/cc_fts.go` — remove `// TODO` comments, activate imports

### Step 1: Remove the commented-out imports from Task 2

In `cli/cc_fts.go`, replace the commented-out imports with the real ones:

```go
import (
    // ... existing imports ...
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/bleve"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/meilisearch"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/clickhouse"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/postgres"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/quickwit"
    _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/tantivy-lnx"
)
```

Note: `tantivy-go` driver is build-tagged and requires a separate import:
Create `cli/cc_fts_tantivy.go`:

```go
//go:build tantivy

package cli

import _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/tantivy-go"
```

### Step 2: Build and verify all engines are listed

```bash
go build -o /tmp/search-cli ./cmd/search/
/tmp/search-cli cc fts index --help 2>&1 | grep -A2 "engine"
```

Expected output includes: `bleve, clickhouse, chdb, devnull, duckdb, meilisearch, postgres, quickwit, sqlite, tantivy-lnx`

### Step 3: Commit

```bash
git add cli/cc_fts.go cli/cc_fts_tantivy.go
git commit -m "feat(cli): activate all new FTS driver imports in cc_fts.go"
```

---

## Task 13: Benchmark all engines and fill spec tables

**Prerequisite:** All docker services running. Pre-packed `docs.bin` available at
`$HOME/data/common-crawl/CC-MAIN-2026-08/fts/pack/docs.bin` (run `search cc fts pack --format bin` if not present).

### Step 1: Start all external services

```bash
FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/meilisearch/docker-compose.yml up -d
FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/clickhouse/docker-compose.yml up -d
FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/postgres-fts/docker-compose.yml up -d
FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/quickwit/docker-compose.yml up -d
FTS_DATA_DIR=$HOME/data/fts docker compose -f docker/lnx/docker-compose.yml up -d

# Wait for all healthy
docker ps --filter label=com.docker.compose.project --format "table {{.Names}}\t{{.Status}}" | grep fts
```

### Step 2: Run index benchmarks (embedded engines)

```bash
/tmp/search-cli cc fts index --engine devnull  --source bin
/tmp/search-cli cc fts index --engine bleve    --source bin
/tmp/search-cli cc fts index --engine duckdb   --source bin
/tmp/search-cli cc fts index --engine sqlite   --source bin
```

Record each output block:
- `elapsed`, `avg rate`, `peak RSS`, `disk`

### Step 3: Run index benchmarks (external engines)

```bash
/tmp/search-cli cc fts index --engine meilisearch --source bin
/tmp/search-cli cc fts index --engine clickhouse  --source bin
/tmp/search-cli cc fts index --engine postgres    --source bin
/tmp/search-cli cc fts index --engine quickwit    --source bin
/tmp/search-cli cc fts index --engine tantivy-lnx --source bin
```

### Step 4: Run search benchmarks

Run each query 3× warm, record latency:

```bash
for query in "machine learning" "climate change" "artificial intelligence" \
             "United States" "open source software" "COVID-19 pandemic" \
             "data privacy" "renewable energy" "blockchain technology" "neural network"; do
  for engine in bleve duckdb sqlite meilisearch clickhouse postgres quickwit tantivy-lnx; do
    echo "=== $engine: $query ==="
    for i in 1 2 3; do
      /tmp/search-cli cc fts search "$query" --engine "$engine" --limit 10 2>&1 | grep -E "hits|ms"
    done
  done
done
```

### Step 5: Fill benchmark tables in spec/0644_external_index.md

Edit `spec/0644_external_index.md` — replace all `—` placeholders with measured values.

### Step 6: Commit final results

```bash
git add spec/0644_external_index.md
git commit -m "docs(spec/0644): fill benchmark results for all 9 FTS engines"
```

---

## Quick Reference

### Start all services

```bash
FTS_DATA_DIR=$HOME/data/fts
for svc in meilisearch clickhouse postgres-fts quickwit lnx; do
  docker compose -f docker/$svc/docker-compose.yml up -d
done
```

### Check service health

```bash
docker ps --filter name=fts- --format "table {{.Names}}\t{{.Status}}"
```

### Run all driver tests

```bash
go test ./pkg/index/... -v -timeout 120s
```

### Build with tantivy

```bash
go build -tags tantivy -o /tmp/search-cli ./cmd/search/
```

### Disk usage per engine

```bash
du -sh $HOME/data/fts/*/
```
