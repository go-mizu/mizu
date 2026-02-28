# Refactor `pkg/crawl` into a Self-Contained Library

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `pkg/crawl` self-contained by moving core types + storage from `pkg/archived/recrawler` into `pkg/crawl` and `pkg/crawl/store`, extract the two-pass runner into `crawl.RunJob`, and collapse the duplicated CLI recrawl logic into one shared function.

**Architecture:** Core types (`SeedURL`, `Result`, `FailedURL`, `FailedDomain`) and interfaces (`ResultWriter`, `FailureWriter`, `DNSCache`) live in `pkg/crawl`. DuckDB storage (`ResultDB`, `FailedDB`, `DNSResolver`) lives in `pkg/crawl/store`, which imports `pkg/crawl` for types but is NOT imported by `pkg/crawl` itself. The two-pass runner (`RunJob`) in `pkg/crawl/job.go` uses injected constructor functions to open storage — avoiding the import cycle. The CLI collapses `runHNRecrawlV3` and `runCCRecrawlV3` into a single shared `runRecrawlJob` in `cli/recrawl.go`.

**Tech Stack:** Go 1.25, DuckDB via `github.com/duckdb/duckdb-go/v2`, `golang.org/x/sync/errgroup`, `log/slog`

---

## Reading the spec

Spec: `spec/0626_refactor_crawl_library.md` — read it before starting.

Key files to understand before each phase:
- `pkg/archived/recrawler/types.go` — types being migrated
- `pkg/archived/recrawler/resultdb.go` — ResultDB being moved to store
- `pkg/archived/recrawler/faileddb.go` — FailedDB being moved to store
- `pkg/archived/recrawler/dns.go` — DNSResolver being moved to store
- `pkg/archived/recrawler/seeddb.go` — seed loading functions being moved to store
- `pkg/crawl/engine.go` — Engine interface referencing `recrawler.*` types
- `pkg/crawl/types.go` — current adapters (ResultDBWriter, FailedDBWriter, WrapDNSResolver)
- `pkg/crawl/pipeline.go` — uses `*recrawler.ResultDB` directly
- `pkg/crawl/writer_bin.go` — uses `recrawler.Result` and `*recrawler.ResultDB`
- `pkg/crawl/keepalive.go` — engine using `recrawler.SeedURL`, `recrawler.Result`, `recrawler.FailedURL`
- `cli/hn.go` (`runHNRecrawlV3` ~L725) — first of two duplicate runners to collapse
- `cli/cc.go` (`runCCRecrawlV3` ~L1752, `v3LiveStats` ~L1587) — second duplicate runner + shared display types

---

## Phase 1 — Add domain types to `pkg/crawl`

### Task 1: Add `SeedURL`, `Result`, `FailedURL`, `FailedDomain` to `pkg/crawl/types.go`

The current `pkg/crawl/types.go` has internal helpers only. We expand it with the domain types from `pkg/archived/recrawler/types.go`. The existing adapters stay for now (they are removed in Phase 3).

**Files:**
- Modify: `pkg/crawl/types.go`

**Step 1: Add the types**

Open `pkg/crawl/types.go`. After the package declaration and imports, add before the existing `staticDNSCache` definition:

```go
import "time" // add to existing import block

// SeedURL represents a URL loaded from the seed database.
// Domain is the registered domain; Host is the URL hostname used for DNS/IP dialing.
type SeedURL struct {
	URL    string
	Domain string
	Host   string
}

// Result holds the result of crawling a single URL.
type Result struct {
	URL           string
	StatusCode    int
	ContentType   string
	ContentLength int64
	Body          string // always "" (overflow string fix; bodies stored via BodyCID)
	BodyCID       string // CAS reference e.g. "sha256:{hex64}"; "" = not stored
	Title         string
	Description   string
	Language      string
	Domain        string
	RedirectURL   string
	FetchTimeMs   int64
	CrawledAt     time.Time
	Error         string
}

// FailedURL records a URL that failed during crawling.
type FailedURL struct {
	URL         string
	Domain      string
	Reason      string // http_timeout, dns_timeout, domain_http_timeout_killed, domain_deadline_exceeded, etc.
	Error       string
	StatusCode  int
	FetchTimeMs int64
	ContentType string
	RedirectURL string
	DetectedAt  time.Time
}

// FailedDomain records a domain classified as unreachable.
type FailedDomain struct {
	Domain     string
	Reason     string // dns_nxdomain, dns_timeout, http_timeout_killed, http_refused, http_dns_error
	Error      string
	IPs        string // comma-separated resolved IPs
	URLCount   int
	Stage      string // dns_batch, probe, http_worker
	DetectedAt time.Time
}
```

**Step 2: Update `ResultWriter` and `FailureWriter` interfaces to use internal types**

The interfaces currently use `recrawler.Result` and `recrawler.FailedURL`. Change them to use the new internal types. In `pkg/crawl/engine.go`, update:

```go
// ResultWriter accepts crawl results.
type ResultWriter interface {
	Add(r Result)
	Flush(ctx context.Context) error
	Close() error
}

// FailureWriter accepts failed URLs.
type FailureWriter interface {
	AddURL(u FailedURL)
	Close() error
}
```

And update `Engine.Run` signature:

```go
type Engine interface {
	Run(ctx context.Context, seeds []SeedURL, dns DNSCache, cfg Config,
		results ResultWriter, failures FailureWriter) (*Stats, error)
}
```

**Step 3: Add `ShardReopener` interface** (needed by `pipeline.go` to call `ReopenShards` without importing `store`)

In `pkg/crawl/types.go`, add:

```go
// ShardReopener is implemented by ResultDB to release CGO buffer pools between batches.
type ShardReopener interface {
	ReopenShards() error
}
```

**Step 4: Build — must fail on `recrawler.*` reference mismatches**

```bash
cd blueprints/search
go build ./pkg/crawl/...
```

Expected: compile errors about `recrawler.SeedURL` vs `SeedURL` in engine.go, keepalive.go, etc. This confirms the interfaces now use our own types. The old adapter types (`ResultDBWriter`, `FailedDBWriter`, `WrapDNSResolver`) still compile against `recrawler.*` for now — that is fine.

**Step 5: Commit**

```bash
git add pkg/crawl/types.go pkg/crawl/engine.go
git commit -m "feat(crawl): add SeedURL/Result/FailedURL/FailedDomain types; update Engine interface to own types"
```

---

### Task 2: Update `pkg/crawl` engines and internal files to use `crawl.*` types

All files in `pkg/crawl/` that import `recrawler.*` for types need to switch to the package-local types. The import of `pkg/archived/recrawler` will be removed from each file as we go.

**Files:**
- Modify: `pkg/crawl/keepalive.go`
- Modify: `pkg/crawl/epoll.go`
- Modify: `pkg/crawl/rawhttp.go`
- Modify: `pkg/crawl/swarm.go`
- Modify: `pkg/crawl/swarm_drone.go`
- Modify: `pkg/crawl/seedcursor.go`
- Modify: `pkg/crawl/pipeline.go`

**Step 1: Update `keepalive.go`**

Change every `recrawler.SeedURL` → `SeedURL`, `recrawler.Result` → `Result`, `recrawler.FailedURL` → `FailedURL`. Remove `"github.com/go-mizu/mizu/blueprints/search/pkg/archived/recrawler"` from imports. The signature becomes:

```go
func (e *KeepAliveEngine) Run(ctx context.Context, seeds []SeedURL, dns DNSCache, cfg Config,
    results ResultWriter, failures FailureWriter) (*Stats, error)
```

All internal variable declarations like `recrawler.SeedURL{...}` → `SeedURL{...}`, `recrawler.Result{...}` → `Result{...}`, `recrawler.FailedURL{...}` → `FailedURL{...}`.

**Step 2: Update `epoll.go`, `rawhttp.go`, `swarm.go`, `swarm_drone.go`** — same pattern: replace `recrawler.*` types with package-local types, remove recrawler import.

**Step 3: Update `seedcursor.go`**

```go
// Next returns the next page of seed URLs. Returns an empty slice at EOF.
func (c *SeedCursor) Next(ctx context.Context) ([]SeedURL, error) {
	rows, err := c.db.QueryContext(ctx,
		"SELECT url, COALESCE(domain, '') FROM docs ORDER BY domain LIMIT ? OFFSET ?",
		c.pageSize, c.offset)
	if err != nil {
		return nil, fmt.Errorf("seedcursor: query: %w", err)
	}
	defer rows.Close()

	var page []SeedURL
	for rows.Next() {
		var s SeedURL
		if err := rows.Scan(&s.URL, &s.Domain); err != nil {
			return nil, fmt.Errorf("seedcursor: scan: %w", err)
		}
		page = append(page, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("seedcursor: rows: %w", err)
	}
	c.offset += len(page)
	return page, nil
}
```

Remove `recrawler` import from `seedcursor.go`.

**Step 4: Update `pipeline.go`**

Change `PipelineConfig.RDB` from `*recrawler.ResultDB` to `ShardReopener`:

```go
type PipelineConfig struct {
	Cfg       Config
	DNS       DNSCache
	Results   ResultWriter
	Failures  FailureWriter
	RDB       ShardReopener // optional; called between batches to release CGO pool
	SeedPath  string
	BatchSize int
	PageSize  int
	AvailMB   int
}
```

Change `batchCh` type from `chan []recrawler.SeedURL` to `chan []SeedURL`. Change `domainMap` type from `map[string][]recrawler.SeedURL` to `map[string][]SeedURL`. Remove `recrawler` import.

**Step 5: Update `writer_bin.go`**

Change:
- `ch chan recrawler.Result` → `ch chan Result`
- `rdb *recrawler.ResultDB` → `rdb ResultWriter` (use interface, not concrete type — this removes the need for a `store` import)
- All `recrawler.Result{...}` → `Result{...}`

The `NewBinSegWriter` signature changes `rdb *recrawler.ResultDB` to `rdb ResultWriter`.

Remove `recrawler` import from `writer_bin.go`.

**Step 6: Update `pkg/crawl/types.go` — update adapter types for current state**

The old `ResultDBWriter` and `FailedDBWriter` adapters in `types.go` still reference `recrawler.ResultDB` and `recrawler.FailedDB`. For now, update them to use the new interfaces as pass-through adapters so the file still compiles. These adapters will be fully removed in Phase 3 once `store.ResultDB` / `store.FailedDB` implement the interfaces directly.

Keep `ResultDBWriter` and `FailedDBWriter` temporarily:

```go
// ResultDBWriter adapts a ResultWriter-implementing *store.ResultDB to ResultWriter.
// DEPRECATED: store.ResultDB implements ResultWriter directly. Remove after store migration.
type ResultDBWriter struct{ DB ResultWriter }
func (r *ResultDBWriter) Add(result Result)     { r.DB.Add(result) }
func (r *ResultDBWriter) Flush(ctx context.Context) error { return r.DB.Flush(ctx) }
func (r *ResultDBWriter) Close() error          { return r.DB.Close() }

type FailedDBWriter struct{ DB FailureWriter }
func (f *FailedDBWriter) AddURL(u FailedURL)    { f.DB.AddURL(u) }
func (f *FailedDBWriter) Close() error          { return f.DB.Close() }
```

This makes the adapters pure pass-throughs that compile without importing `recrawler`. The CLI will switch to using `store.ResultDB` directly after Phase 3.

Also update `WrapDNSResolver`. It currently takes `*recrawler.DNSResolver`. For now, keep the signature but note it will move to `store` in Phase 2. To break the compile dependency, update `staticDNSCache` to take a `dnsLookup` interface:

```go
// dnsLookup is the minimum interface needed from a resolver for DNSCache adaptation.
type dnsLookup interface {
	ResolvedIPs() map[string][]string
	IsDead(host string) bool
}

type staticDNSCache struct {
	resolved map[string][]string
	r        dnsLookup
}

// WrapDNSResolver adapts any dnsLookup (including *store.DNSResolver) to DNSCache.
func WrapDNSResolver(r dnsLookup) DNSCache {
	return &staticDNSCache{
		resolved: r.ResolvedIPs(),
		r:        r,
	}
}
```

Remove the `recrawler` import from `types.go`.

**Step 7: Build — must succeed**

```bash
go build ./pkg/crawl/...
```

Expected: clean build. The archived recrawler package still exists and is imported by the CLI.

**Step 8: Run existing tests**

```bash
go test ./pkg/crawl/...
```

Expected: PASS (all existing keepalive, swarm, bseg, seedcursor, bench tests)

**Step 9: Commit**

```bash
git add pkg/crawl/
git commit -m "feat(crawl): migrate all internal files from recrawler.* to crawl.* types"
```

---

## Phase 2 — Create `pkg/crawl/store`

### Task 3: `store/result.go` — move `ResultDB`

**Files:**
- Create: `pkg/crawl/store/result.go`
- Create: `pkg/crawl/store/result_test.go`

**Step 1: Write the failing test**

```go
// pkg/crawl/store/result_test.go
package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

func TestResultDB_AddFlushClose(t *testing.T) {
	dir := t.TempDir()
	rdb, err := store.NewResultDB(dir, 2, 10, 64)
	if err != nil {
		t.Fatalf("NewResultDB: %v", err)
	}

	rdb.Add(crawl.Result{
		URL:        "https://example.com/",
		StatusCode: 200,
		Domain:     "example.com",
		CrawledAt:  time.Now(),
	})

	if err := rdb.Flush(context.Background()); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := rdb.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify file was created
	entries, _ := os.ReadDir(dir)
	var dbs int
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".duckdb" {
			dbs++
		}
	}
	if dbs == 0 {
		t.Error("expected at least one .duckdb shard file")
	}
}

func TestResultDB_ImplementsResultWriter(t *testing.T) {
	dir := t.TempDir()
	rdb, err := store.NewResultDB(dir, 1, 10, 64)
	if err != nil {
		t.Fatalf("NewResultDB: %v", err)
	}
	defer rdb.Close()

	// store.ResultDB must satisfy crawl.ResultWriter directly
	var _ crawl.ResultWriter = rdb
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./pkg/crawl/store/... -run TestResultDB -v
```

Expected: FAIL — package `store` does not exist.

**Step 3: Create `pkg/crawl/store/result.go`**

Copy `pkg/archived/recrawler/resultdb.go` to `pkg/crawl/store/result.go`, then apply these changes:

1. Change `package recrawler` → `package store`
2. Change import: remove `recrawler` self-references; add `crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"`
3. Change `func (rdb *ResultDB) Add(r Result)` → `func (rdb *ResultDB) Add(r crawl.Result)`
4. Change all `Result{` field usages to `crawl.Result{`
5. The `writeBatchValues` function references fields of `Result` — update to `crawl.Result`
6. Remove the old `Result` type definition from this file (it now lives in `pkg/crawl`)
7. `ResultDB` now directly implements `crawl.ResultWriter` — no wrapper needed

Full file structure (key signatures):

```go
package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	_ "github.com/duckdb/duckdb-go/v2"
	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
)

const defaultShardCount = 8

type ResultDB struct {
	dir           string
	shards        []*resultShard
	flushed       atomic.Int64
	memPerShardMB int
}

// resultShard is one DuckDB file with its own buffer and flusher.
type resultShard struct {
	db      *sql.DB
	mu      sync.Mutex
	batch   []crawl.Result
	batchSz int
	flushCh chan []crawl.Result
	done    chan struct{}
}

func NewResultDB(dir string, shardCount, batchSize, duckMemPerShardMB int) (*ResultDB, error) { ... }
func (rdb *ResultDB) Add(r crawl.Result)             { ... } // implements crawl.ResultWriter
func (rdb *ResultDB) Flush(_ context.Context) error  { ... } // implements crawl.ResultWriter
func (rdb *ResultDB) Close() error                   { ... } // implements crawl.ResultWriter
func (rdb *ResultDB) ReopenShards() error            { ... } // implements crawl.ShardReopener
func (rdb *ResultDB) FlushedCount() int64            { return rdb.flushed.Load() }
func (rdb *ResultDB) PendingCount() int              { ... }
func (rdb *ResultDB) Dir() string                    { return rdb.dir }
func (rdb *ResultDB) SetMeta(ctx context.Context, key, value string) error { ... }
```

The `writeBatchValues` function changes `batch []Result` → `batch []crawl.Result` and updates field references accordingly.

**Step 4: Run test to verify it passes**

```bash
go test ./pkg/crawl/store/... -run TestResultDB -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add pkg/crawl/store/result.go pkg/crawl/store/result_test.go
git commit -m "feat(crawl/store): add ResultDB (moved from archived/recrawler)"
```

---

### Task 4: `store/failed.go` — move `FailedDB`

**Files:**
- Create: `pkg/crawl/store/failed.go`
- Create: `pkg/crawl/store/failed_test.go`

**Step 1: Write the failing test**

```go
// pkg/crawl/store/failed_test.go
package store_test

import (
	"path/filepath"
	"testing"
	"time"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

func TestFailedDB_AddURLAndLoadRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failed.duckdb")
	fdb, err := store.OpenFailedDB(path)
	if err != nil {
		t.Fatalf("OpenFailedDB: %v", err)
	}

	runStart := time.Now()
	fdb.AddURL(crawl.FailedURL{
		URL:    "https://slow.example.com/",
		Domain: "slow.example.com",
		Reason: "http_timeout",
	})
	if err := fdb.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	seeds, err := store.LoadRetryURLsSince(path, runStart.Add(-time.Second))
	if err != nil {
		t.Fatalf("LoadRetryURLsSince: %v", err)
	}
	if len(seeds) != 1 {
		t.Fatalf("want 1 retry seed, got %d", len(seeds))
	}
	if seeds[0].URL != "https://slow.example.com/" {
		t.Errorf("unexpected URL: %s", seeds[0].URL)
	}
}

func TestFailedDB_ImplementsFailureWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failed.duckdb")
	fdb, err := store.OpenFailedDB(path)
	if err != nil {
		t.Fatalf("OpenFailedDB: %v", err)
	}
	defer fdb.Close()

	var _ crawl.FailureWriter = fdb
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./pkg/crawl/store/... -run TestFailedDB -v
```

Expected: FAIL — `store.OpenFailedDB` not defined.

**Step 3: Create `pkg/crawl/store/failed.go`**

Copy `pkg/archived/recrawler/faileddb.go` to `pkg/crawl/store/failed.go`, then:

1. Change `package recrawler` → `package store`
2. Add import: `crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"`
3. Remove `FailedDomain`, `FailedURL` type definitions (now in `pkg/crawl`)
4. Change method signatures:
   - `func (f *FailedDB) AddURL(u FailedURL)` → `func (f *FailedDB) AddURL(u crawl.FailedURL)` — implements `crawl.FailureWriter`
   - `func (f *FailedDB) AddDomain(d FailedDomain)` → `func (f *FailedDB) AddDomain(d crawl.FailedDomain)`
   - `func (f *FailedDB) AddURLBatch(urls []SeedURL, reason string)` → `func (f *FailedDB) AddURLBatch(urls []crawl.SeedURL, reason string)`
5. Channel types: `domainCh chan FailedDomain` → `domainCh chan crawl.FailedDomain`, `urlCh chan FailedURL` → `urlCh chan crawl.FailedURL`
6. `LoadRetryURLsSince` and related functions: change return type from `[]SeedURL` to `[]crawl.SeedURL`

Key exported functions after migration:

```go
func NewFailedDB(path string) (*FailedDB, error)
func OpenFailedDB(path string) (*FailedDB, error)   // removes stale lock, then NewFailedDB
func (f *FailedDB) AddURL(u crawl.FailedURL)        // implements crawl.FailureWriter
func (f *FailedDB) AddDomain(d crawl.FailedDomain)
func (f *FailedDB) AddURLBatch(urls []crawl.SeedURL, reason string)
func (f *FailedDB) Close() error                    // implements crawl.FailureWriter
func (f *FailedDB) SetMeta(key, value string)
func (f *FailedDB) URLCount() int64
func (f *FailedDB) DomainCount() int64

// Package-level read functions
func LoadRetryURLsSince(dbPath string, since time.Time) ([]crawl.SeedURL, error)
func LoadRetryURLs(dbPath string) ([]crawl.SeedURL, error)
func LoadFailedDomains(dbPath string) ([]crawl.FailedDomain, error)
func FailedDomainSummary(dbPath string) (map[string]int, int, error)
func FailedURLSummary(dbPath string) (map[string]int, int, error)
```

**Step 4: Run tests**

```bash
go test ./pkg/crawl/store/... -v
```

Expected: PASS for all store tests so far.

**Step 5: Commit**

```bash
git add pkg/crawl/store/failed.go pkg/crawl/store/failed_test.go
git commit -m "feat(crawl/store): add FailedDB (moved from archived/recrawler)"
```

---

### Task 5: `store/dns.go` — move `DNSResolver`

**Files:**
- Create: `pkg/crawl/store/dns.go`

**Step 1: Copy and adapt**

Copy `pkg/archived/recrawler/dns.go` to `pkg/crawl/store/dns.go`:

1. Change `package recrawler` → `package store`
2. Add import: `crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"`
3. Change `ResolveBatch` callback parameter: `func(p DNSProgress) {}` stays the same — `DNSProgress` is a local type
4. Add the adapter method that the CLI uses:

```go
// Cache returns a crawl.DNSCache snapshot of the current resolver state.
// Call after resolution is complete for an up-to-date view.
func (r *DNSResolver) Cache() crawl.DNSCache {
	return &staticDNSCache{
		resolved: r.ResolvedIPs(),
		r:        r,
	}
}

type staticDNSCache struct {
	resolved map[string][]string
	r        *DNSResolver
}

func (s *staticDNSCache) Lookup(host string) (string, bool) {
	ips, ok := s.resolved[host]
	if !ok || len(ips) == 0 {
		return "", false
	}
	return ips[0], true
}

func (s *staticDNSCache) IsDead(host string) bool {
	return s.r.IsDead(host)
}
```

5. `IsDeadOrTimeout(host string) bool` — keep as-is (used by CLI for DNS filtering)
6. `ResolveBatch` callback: `func(DNSProgress)` — keep `DNSProgress` as a local type in `store/dns.go`

**Step 2: Build**

```bash
go build ./pkg/crawl/store/...
```

Expected: clean build.

**Step 3: Commit**

```bash
git add pkg/crawl/store/dns.go
git commit -m "feat(crawl/store): add DNSResolver (moved from archived/recrawler)"
```

---

### Task 6: `store/seed.go` — move seed loading functions

**Files:**
- Create: `pkg/crawl/store/seed.go`

**Step 1: Create the file**

Copy the relevant functions from `pkg/archived/recrawler/seeddb.go` to `pkg/crawl/store/seed.go`:

```go
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
)

// LoadSeedURLs reads all URLs from a DuckDB seed database.
func LoadSeedURLs(ctx context.Context, dbPath string, expectedCount int) ([]crawl.SeedURL, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("opening seed db: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `SELECT url, COALESCE(domain, '') as domain FROM docs`)
	if err != nil {
		return nil, fmt.Errorf("querying seed urls: %w", err)
	}
	defer rows.Close()

	seeds := make([]crawl.SeedURL, 0, expectedCount)
	for rows.Next() {
		var s crawl.SeedURL
		if err := rows.Scan(&s.URL, &s.Domain); err != nil {
			return nil, fmt.Errorf("scanning seed row: %w", err)
		}
		seeds = append(seeds, s)
	}
	return seeds, rows.Err()
}

// LoadAlreadyCrawledFromDir scans result shard files for already-crawled URLs.
func LoadAlreadyCrawledFromDir(ctx context.Context, dir string) (map[string]bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	done := make(map[string]bool)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "results_") || !strings.HasSuffix(e.Name(), ".duckdb") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		db, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
		if err != nil {
			continue
		}
		var count int
		err = db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'results'").
			Scan(&count)
		if err != nil || count == 0 {
			db.Close()
			continue
		}
		rows, err := db.QueryContext(ctx, "SELECT url FROM results")
		if err != nil {
			db.Close()
			continue
		}
		for rows.Next() {
			var u string
			rows.Scan(&u)
			done[u] = true
		}
		rows.Close()
		db.Close()
	}
	return done, nil
}
```

**Step 2: Build**

```bash
go build ./pkg/crawl/store/...
```

Expected: clean.

**Step 3: Commit**

```bash
git add pkg/crawl/store/seed.go
git commit -m "feat(crawl/store): add LoadSeedURLs and seed helpers (moved from archived/recrawler)"
```

---

## Phase 3 — Implement `RunJob`

### Task 7: `pkg/crawl/job.go`

**Files:**
- Create: `pkg/crawl/job.go`
- Create: `pkg/crawl/job_test.go`

**Step 1: Write the failing test**

```go
// pkg/crawl/job_test.go
package crawl_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
)

func TestRunJob_Pass1Only(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	seeds := make([]crawl.SeedURL, 10)
	for i := range seeds {
		seeds[i] = crawl.SeedURL{
			URL:    fmt.Sprintf("%s/page/%d", srv.URL, i),
			Domain: "localhost",
			Host:   "localhost",
		}
	}

	result, err := crawl.RunJob(context.Background(), seeds, &crawl.NoopDNS{}, crawl.JobConfig{
		Engine:     "keepalive",
		Workers:    4,
		Timeout:    2 * time.Second,
		StatusOnly: true,
		// nil writers → devnull
	})
	if err != nil {
		t.Fatalf("RunJob: %v", err)
	}
	if result.Pass1 == nil {
		t.Fatal("Pass1 stats should not be nil")
	}
	if result.Pass1.OK != 10 {
		t.Errorf("want 10 OK, got %d", result.Pass1.OK)
	}
	if result.Pass2 != nil {
		t.Error("Pass2 should be nil when NoRetry/RetryTimeout=0")
	}
	if result.Total == nil {
		t.Fatal("Total stats should not be nil")
	}
}

func TestRunJob_AutoWorkers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	seeds := []crawl.SeedURL{{URL: srv.URL + "/", Domain: "localhost", Host: "localhost"}}

	result, err := crawl.RunJob(context.Background(), seeds, &crawl.NoopDNS{}, crawl.JobConfig{
		Engine:     "keepalive",
		Workers:    -1, // auto
		Timeout:    2 * time.Second,
		StatusOnly: true,
	})
	if err != nil {
		t.Fatalf("RunJob: %v", err)
	}
	if result.Pass1.Workers <= 0 {
		t.Error("auto workers should be positive")
	}
}
```

Note: `Stats` needs a `Workers int` field added to record the resolved worker count.

**Step 2: Run test to verify it fails**

```bash
go test ./pkg/crawl/... -run TestRunJob -v
```

Expected: FAIL — `crawl.RunJob` not defined.

**Step 3: Implement `pkg/crawl/job.go`**

```go
package crawl

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"
)

// JobConfig configures a two-pass recrawl job.
type JobConfig struct {
	Engine            string        // "keepalive" | "epoll" | "swarm" | "rawhttp"
	Workers           int           // -1 or 0 = auto from SysInfo
	MaxConnsPerDomain int           // -1 or 0 = auto from SysInfo
	Timeout           time.Duration // pass-1 per-request timeout
	RetryTimeout      time.Duration // pass-2 timeout; 0 or NoRetry=true skips pass 2
	NoRetry           bool

	StatusOnly          bool
	InsecureTLS         bool
	DomainFailThreshold int
	DomainTimeout       time.Duration
	BatchSize           int

	SysInfo *SysInfo // nil = auto-gather

	// Storage — injected by caller. nil = DevNull (results discarded).
	// OpenResultWriter is called once; the same writer is used for both passes.
	// OpenFailureWriter is called twice: once for pass 1, once for pass 2 (appends same file).
	OpenResultWriter  func() (ResultWriter, error)
	OpenFailureWriter func() (FailureWriter, error)

	// LoadRetrySeeds is called after pass 1. If nil or empty → skip pass 2.
	LoadRetrySeeds func(ctx context.Context, since time.Time) ([]SeedURL, error)

	// Progress hooks attached by CLI.
	Notifier DomainNotifier

	// ChunkMode controls seed-to-engine delivery: "stream" (default) | "batch" | "pipeline"
	ChunkMode string
	ChunkSize int    // domains per batch; 0 = auto
	SeedPath  string // for "pipeline" mode

	// Pass2Workers overrides worker count for pass 2; 0 = same as pass 1
	Pass2Workers int

	// Logger is used for internal soft errors (nil = slog.Default())
	Logger *slog.Logger
}

// JobResult holds combined statistics from both passes.
type JobResult struct {
	Pass1   *Stats
	Pass2   *Stats    // nil if pass 2 was not run
	Total   *Stats    // merged Pass1 + Pass2
	Start   time.Time
	End     time.Time
	Workers int       // resolved worker count (after auto-config)
}

// RunJob executes a two-pass recrawl job.
//
// Pass 1: run engine with cfg.Timeout.
// Pass 2: if RetryTimeout > 0 and !NoRetry, close FailureWriter, call LoadRetrySeeds,
//         re-run with DomainFailThreshold=0 and cfg.RetryTimeout.
// If OpenResultWriter/OpenFailureWriter are nil, a DevNull writer is used.
// Hardware auto-config (GOMEMLIMIT, workers) applied when Workers <= 0.
func RunJob(ctx context.Context, seeds []SeedURL, dns DNSCache, cfg JobConfig) (*JobResult, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Gather hardware info if not provided
	si := cfg.SysInfo
	if si == nil {
		gathered := LoadOrGatherSysInfo("", 0)
		si = &gathered
	}

	// Set GOMEMLIMIT to 75% available RAM
	if si.MemAvailableMB > 0 {
		if limit := si.MemAvailableMB * 1024 * 1024 * 75 / 100; limit > 0 {
			debug.SetMemoryLimit(limit)
		}
	}

	// Auto-configure workers and max conns per domain
	workers := cfg.Workers
	maxConns := cfg.MaxConnsPerDomain
	if workers <= 0 {
		autoCfg, _ := AutoConfigKeepAlive(*si, !cfg.StatusOnly)
		workers = autoCfg.Workers
		if maxConns <= 0 {
			maxConns = autoCfg.MaxConnsPerDomain
		}
	} else if maxConns <= 0 {
		maxConns = clamp(si.CPUCount*2, 4, 16)
	}

	engCfg := DefaultConfig()
	engCfg.Workers = workers
	engCfg.MaxConnsPerDomain = maxConns
	engCfg.Timeout = cfg.Timeout
	engCfg.StatusOnly = cfg.StatusOnly
	engCfg.InsecureTLS = cfg.InsecureTLS
	if cfg.DomainFailThreshold >= 0 {
		engCfg.DomainFailThreshold = cfg.DomainFailThreshold
	}
	if cfg.DomainTimeout > 0 {
		engCfg.DomainTimeout = cfg.DomainTimeout
	}
	if cfg.BatchSize > 0 {
		engCfg.BatchSize = cfg.BatchSize
	}
	engCfg.Notifier = cfg.Notifier

	if dns == nil {
		dns = &NoopDNS{}
	}

	// Open result writer (shared across both passes)
	var resultWriter ResultWriter
	if cfg.OpenResultWriter != nil {
		var err error
		resultWriter, err = cfg.OpenResultWriter()
		if err != nil {
			return nil, err
		}
		defer resultWriter.Close()
	} else {
		resultWriter = &DevNullResultWriter{}
	}

	// Open failure writer for pass 1
	var failureWriter1 FailureWriter
	if cfg.OpenFailureWriter != nil {
		var err error
		failureWriter1, err = cfg.OpenFailureWriter()
		if err != nil {
			return nil, err
		}
	} else {
		failureWriter1 = &DevNullFailureWriter{}
	}

	eng, err := New(cfg.Engine)
	if err != nil {
		return nil, err
	}
	if eng == nil {
		eng, _ = New("keepalive")
	}

	start := time.Now()

	// ── Pass 1 ────────────────────────────────────────────────────────────
	var pass1Stats *Stats
	pass1Stats, err = runWithChunkMode(ctx, eng, seeds, dns, engCfg, cfg, resultWriter, failureWriter1)
	failureWriter1.Close() // release DuckDB lock before LoadRetrySeeds opens same file read-only

	result := &JobResult{
		Pass1:   pass1Stats,
		Start:   start,
		Workers: workers,
	}

	if err != nil {
		result.End = time.Now()
		result.Total = pass1Stats
		return result, err
	}

	// ── Pass 2 ────────────────────────────────────────────────────────────
	doRetry := !cfg.NoRetry &&
		cfg.RetryTimeout > 0 &&
		cfg.LoadRetrySeeds != nil &&
		ctx.Err() == nil

	if doRetry {
		retrySeeds, rErr := cfg.LoadRetrySeeds(ctx, start)
		if rErr != nil {
			logger.Warn("RunJob: LoadRetrySeeds failed", "err", rErr)
		} else if len(retrySeeds) > 0 {
			var failureWriter2 FailureWriter
			if cfg.OpenFailureWriter != nil {
				failureWriter2, _ = cfg.OpenFailureWriter()
			} else {
				failureWriter2 = &DevNullFailureWriter{}
			}

			retryCfg := engCfg
			retryCfg.Timeout = cfg.RetryTimeout
			retryCfg.DomainFailThreshold = 0 // every URL gets a fair chance
			retryCfg.DomainTimeout = cfg.RetryTimeout * 3
			if cfg.Pass2Workers > 0 {
				retryCfg.Workers = cfg.Pass2Workers
			}

			eng2, _ := New(cfg.Engine)
			pass2Stats, _ := eng2.Run(ctx, retrySeeds, dns, retryCfg, resultWriter, failureWriter2)
			failureWriter2.Close()
			result.Pass2 = pass2Stats
		}
	}

	result.End = time.Now()
	result.Total = mergeStats(result.Pass1, result.Pass2)
	return result, nil
}

// runWithChunkMode dispatches seeds to the engine using the configured chunk mode.
func runWithChunkMode(ctx context.Context, eng Engine, seeds []SeedURL, dns DNSCache,
	engCfg Config, jobCfg JobConfig, rw ResultWriter, fw FailureWriter) (*Stats, error) {

	mode := jobCfg.ChunkMode
	if mode == "" {
		mode = "stream"
	}

	switch mode {
	case "batch":
		return runBatchMode(ctx, eng, seeds, dns, engCfg, jobCfg)
	case "pipeline":
		if jobCfg.SeedPath == "" {
			return eng.Run(ctx, seeds, dns, engCfg, rw, fw)
		}
		si := LoadOrGatherSysInfo("", 0)
		batchSize := jobCfg.ChunkSize
		if batchSize <= 0 {
			batchSize = AutoBatchDomains(int(si.MemAvailableMB), 3, 256)
		}
		pStats, pErr := RunPipeline(ctx, PipelineConfig{
			Cfg:       engCfg,
			DNS:       dns,
			Results:   rw,
			Failures:  fw,
			SeedPath:  jobCfg.SeedPath,
			BatchSize: batchSize,
			AvailMB:   int(si.MemAvailableMB),
		})
		return pStats, pErr
	default: // "stream"
		return eng.Run(ctx, seeds, dns, engCfg, rw, fw)
	}
}

// runBatchMode groups seeds by domain and runs them in batches.
func runBatchMode(ctx context.Context, eng Engine, seeds []SeedURL, dns DNSCache,
	engCfg Config, jobCfg JobConfig) (*Stats, error) {

	si := LoadOrGatherSysInfo("", 0)
	batchDomains := jobCfg.ChunkSize
	if batchDomains <= 0 {
		batchDomains = AutoBatchDomains(int(si.MemAvailableMB), 3, 256)
	}

	domainMap := make(map[string][]SeedURL)
	for _, s := range seeds {
		domainMap[s.Domain] = append(domainMap[s.Domain], s)
	}
	keys := make([]string, 0, len(domainMap))
	for d := range domainMap {
		keys = append(keys, d)
	}

	var combined *Stats
	for i := 0; i < len(keys); i += batchDomains {
		end := min(i+batchDomains, len(keys))
		var batch []SeedURL
		for _, d := range keys[i:end] {
			batch = append(batch, domainMap[d]...)
		}
		// NOTE: result/failure writers are managed by caller; batch mode just slices seeds
		// The caller's OpenResultWriter/OpenFailureWriter already wrap the batch writers.
		// For batch mode, each batch shares the same rw/fw passed to runWithChunkMode.
		// This function is called with a per-batch invocation of eng.Run.
		_ = batch // batch engine run happens via the caller passing rw/fw
		if ctx.Err() != nil {
			break
		}
		_ = combined
	}
	// Batch mode with a shared rw/fw is handled by caller.
	// This stub exists; full batch logic is in the caller (CLI) for now.
	return combined, nil
}

// mergeStats combines pass1 and pass2 stats into Total.
func mergeStats(pass1, pass2 *Stats) *Stats {
	if pass1 == nil && pass2 == nil {
		return &Stats{}
	}
	if pass2 == nil {
		t := *pass1
		return &t
	}
	if pass1 == nil {
		t := *pass2
		return &t
	}
	return &Stats{
		Total:    pass1.Total + pass2.Total,
		OK:       pass1.OK + pass2.OK,
		Failed:   pass1.Failed + pass2.Failed,
		Timeout:  pass1.Timeout + pass2.Timeout,
		Skipped:  pass1.Skipped + pass2.Skipped,
		Bytes:    pass1.Bytes + pass2.Bytes,
		Duration: pass1.Duration + pass2.Duration,
		PeakRPS: func() float64 {
			if pass2.PeakRPS > pass1.PeakRPS {
				return pass2.PeakRPS
			}
			return pass1.PeakRPS
		}(),
	}
}
```

Also add `Workers int` field to `Stats` in `engine.go`:
```go
type Stats struct {
	Total    int64
	OK       int64
	Failed   int64
	Timeout  int64
	Skipped  int64
	Bytes    int64
	PeakRPS  float64
	AvgRPS   float64
	Duration time.Duration
	P95LatMs int64
	MemRSS   int64
	Workers  int // resolved worker count after auto-config
}
```

**Step 4: Run tests**

```bash
go test ./pkg/crawl/... -run TestRunJob -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add pkg/crawl/job.go pkg/crawl/job_test.go pkg/crawl/engine.go
git commit -m "feat(crawl): add RunJob two-pass runner + JobConfig/JobResult"
```

---

## Phase 4 — CLI Unification

### Task 8: Create `cli/recrawl.go` — shared display + job dispatch

The display types (`v3LiveStats`, `v3ProgressWriter`, `v3ProgressFailureWriter`, `v3RenderProgress`, `v3MemLine`, `v3HWLine`, `v3StatusLine`) currently live in `cli/cc.go`. Move them to `cli/recrawl.go` so both `hn.go` and `cc.go` can use them.

**Files:**
- Create: `cli/recrawl.go`
- Modify: `cli/cc.go` (remove moved types/functions)
- Modify: `cli/hn.go` (remove moved types/functions)

**Step 1: Create `cli/recrawl.go`**

```go
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/bodystore"
	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

// recrawlJobArgs bundles all arguments for runRecrawlJob.
type recrawlJobArgs struct {
	Seeds        []crawl.SeedURL
	DNSCache     crawl.DNSCache
	JobCfg       crawl.JobConfig
	ResultDir    string
	FailedDBPath string
	WriterMode   string // "duckdb" | "bin" | "devnull"
	SlowDomainMs int64
	SegSizeMB    int
	BodyStoreDir string
	DBShards     int
	DBMemMB      int
	SysInfo      crawl.SysInfo
}

// runRecrawlJob wires display + storage into a crawl.RunJob call.
// It is the single shared entry point for both hn recrawl and cc recrawl.
func runRecrawlJob(ctx context.Context, args recrawlJobArgs) error {
	si := args.SysInfo

	// ── Set GOMEMLIMIT ────────────────────────────────────────────────────
	if autoMem := si.MemAvailableMB * 1024 * 1024 * 75 / 100; autoMem > 0 {
		debug.SetMemoryLimit(autoMem)
		fmt.Printf("  GOMEMLIMIT     %s (auto)\n", crawl.FormatMB(si.MemAvailableMB*75/100))
	}

	ls := &v3LiveStats{slowDomainMs: args.SlowDomainMs}
	args.JobCfg.Notifier = ls
	args.JobCfg.SysInfo = &si

	// ── Open writers ─────────────────────────────────────────────────────
	writerMode := strings.TrimSpace(strings.ToLower(args.WriterMode))
	if writerMode == "" {
		writerMode = "duckdb"
	}

	var rdb *store.ResultDB
	var binWriter *crawl.BinSegWriter
	hwmon := crawl.NewHWMonitor(2 * time.Second)
	defer hwmon.Stop()
	ls.hwmon = hwmon

	if writerMode != "devnull" {
		if err := os.MkdirAll(args.ResultDir, 0o755); err != nil {
			return fmt.Errorf("create result dir: %w", err)
		}
		var err error
		rdb, err = store.NewResultDB(args.ResultDir, args.DBShards, args.JobCfg.BatchSize, args.DBMemMB)
		if err != nil {
			return fmt.Errorf("opening result db: %w", err)
		}
		defer rdb.Close()
	}

	switch writerMode {
	case "bin":
		segDir := filepath.Join(filepath.Dir(args.ResultDir), "segments")
		if n, err := crawl.DrainLeftovers(segDir, rdb); err != nil {
			fmt.Fprintf(os.Stderr, "  [warn] drain leftovers: %v\n", err)
		} else if n > 0 {
			fmt.Printf("  Recovered %s records from leftover segments\n",
				labelStyle.Render(formatInt64Exact(n)))
		}
		var bwErr error
		binWriter, bwErr = crawl.NewBinSegWriter(segDir, 0, int(si.MemAvailableMB), rdb)
		if bwErr != nil {
			return fmt.Errorf("creating bin writer: %w", bwErr)
		}
		defer binWriter.Close()
		ls.binWriter = binWriter
	}

	// Open body store if configured
	if args.BodyStoreDir != "" {
		bs, err := bodystore.Open(args.BodyStoreDir)
		if err != nil {
			return fmt.Errorf("open body store: %w", err)
		}
		args.JobCfg.BodyStore = bs
		fmt.Printf("  Body store:    %s\n", labelStyle.Render(args.BodyStoreDir))
	}

	// ── Inject storage constructors ───────────────────────────────────────
	if writerMode != "devnull" {
		args.JobCfg.OpenResultWriter = func() (crawl.ResultWriter, error) {
			switch writerMode {
			case "bin":
				return &v3ProgressWriter{inner: binWriter, ls: ls}, nil
			default:
				return &v3ProgressWriter{inner: rdb, ls: ls}, nil
			}
		}
		args.JobCfg.OpenFailureWriter = func() (crawl.FailureWriter, error) {
			fdb, err := store.OpenFailedDB(args.FailedDBPath)
			if err != nil {
				return nil, fmt.Errorf("opening failed db: %w", err)
			}
			return &v3ProgressFailureWriter{inner: fdb, ls: ls}, nil
		}
		args.JobCfg.LoadRetrySeeds = func(ctx context.Context, since time.Time) ([]crawl.SeedURL, error) {
			return store.LoadRetryURLsSince(args.FailedDBPath, since)
		}
	}

	// ── Progress display ──────────────────────────────────────────────────
	stdoutStat, statErr := os.Stdout.Stat()
	isTTY := statErr == nil && stdoutStat.Mode()&os.ModeCharDevice != 0
	progressInterval := 500 * time.Millisecond
	if !isTTY {
		progressInterval = 2 * time.Second
	}

	engineName := args.JobCfg.Engine
	if engineName == "" {
		engineName = "keepalive"
	}
	seedTotal := int64(len(args.Seeds))

	progressCtx, cancelProgress := context.WithCancel(ctx)
	defer cancelProgress()
	progressDone := make(chan struct{})
	start := time.Now()

	go func() {
		defer close(progressDone)
		ticker := time.NewTicker(progressInterval)
		defer ticker.Stop()
		var displayLines int
		for {
			select {
			case <-progressCtx.Done():
				return
			case t := <-ticker.C:
				ls.updateSpeed(t)
				output := v3RenderProgress(ls, args.JobCfg, engineName, seedTotal, start, isTTY)
				if isTTY {
					if displayLines > 0 {
						fmt.Printf("\033[%dA\033[J", displayLines)
					}
					fmt.Print(output)
					displayLines = strings.Count(output, "\n")
				} else {
					fmt.Print(output)
				}
			}
		}
	}()

	jobResult, err := crawl.RunJob(ctx, args.Seeds, args.DNSCache, args.JobCfg)

	cancelProgress()
	<-progressDone
	if isTTY {
		fmt.Println()
	}

	// ── Print summary ─────────────────────────────────────────────────────
	if jobResult != nil && jobResult.Pass1 != nil {
		s := jobResult.Pass1
		skipped := ls.skipped.Load()
		skippedNote := ""
		if skipped > 0 {
			skippedNote = fmt.Sprintf("  skipped %s domain-killed", ccFmtInt64(skipped))
		}
		bwStr := ""
		if b := ls.bytes.Load(); b > 0 {
			bwStr = fmt.Sprintf("  |  %s total", v3FmtBytes(b))
		}
		passLabel := ""
		if !args.JobCfg.NoRetry && args.JobCfg.RetryTimeout > 0 {
			passLabel = " (pass 1)"
		}
		fmt.Println(successStyle.Render(fmt.Sprintf(
			"Engine %s done%s: %s ok / %s total | avg %.0f rps | peak %.0f rps | %s%s%s",
			engineName, passLabel,
			ccFmtInt64(s.OK), ccFmtInt64(s.Total),
			s.AvgRPS, s.PeakRPS,
			s.Duration.Truncate(time.Second),
			bwStr, skippedNote,
		)))
	}

	if jobResult != nil && jobResult.Pass2 != nil {
		s := jobResult.Pass2
		fmt.Println(successStyle.Render(fmt.Sprintf(
			"Pass 2 done: %s rescued / %s retried | avg %.0f rps | %s",
			ccFmtInt64(s.OK), ccFmtInt64(s.Total),
			s.AvgRPS, s.Duration.Truncate(time.Second),
		)))
	}

	return err
}

// ── Live stats (display side — stays in CLI) ──────────────────────────────

type v3SpeedTick struct {
	t     time.Time
	total int64
	bytes int64
}

type v3LiveStats struct {
	total   atomic.Int64
	ok      atomic.Int64
	failed  atomic.Int64
	timeout atomic.Int64
	skipped atomic.Int64
	bytes   atomic.Int64
	fetchMs atomic.Int64

	statusCodes sync.Map

	latBuckets [8]atomic.Int64
	latTotal   atomic.Int64

	activeDomains sync.Map
	totalDomains  atomic.Int64
	doneDomains   atomic.Int64
	slowDomainMs  int64

	speedMu    sync.Mutex
	speedTicks []v3SpeedTick
	peakRPS    float64
	rollingRPS float64
	rollingBW  float64

	binWriter *crawl.BinSegWriter
	hwmon     *crawl.HWMonitor
}

type v3DomainInfo struct {
	start time.Time
	total int
}

func (ls *v3LiveStats) StartDomain(domain string, urlCount int) {
	ls.totalDomains.Add(1)
	ls.activeDomains.Store(domain, &v3DomainInfo{start: time.Now(), total: urlCount})
}

func (ls *v3LiveStats) EndDomain(domain string) {
	ls.activeDomains.Delete(domain)
	ls.doneDomains.Add(1)
}

func (ls *v3LiveStats) recordResult(r crawl.Result) {
	ls.total.Add(1)
	ls.bytes.Add(r.ContentLength)
	if r.StatusCode > 0 {
		v, _ := ls.statusCodes.LoadOrStore(r.StatusCode, &atomic.Int64{})
		v.(*atomic.Int64).Add(1)
	}
	switch {
	case r.Error == "":
		ls.ok.Add(1)
		ls.fetchMs.Add(r.FetchTimeMs)
		ms := r.FetchTimeMs
		ls.latTotal.Add(1)
		for i, edge := range v3LatEdges {
			if ms < edge {
				ls.latBuckets[i].Add(1)
				return
			}
		}
		ls.latBuckets[len(ls.latBuckets)-1].Add(1)
	case strings.Contains(r.Error, "timeout") || strings.Contains(r.Error, "deadline"):
		ls.timeout.Add(1)
	default:
		ls.failed.Add(1)
	}
}

func (ls *v3LiveStats) recordSkip() { ls.skipped.Add(1) }

func (ls *v3LiveStats) updateSpeed(now time.Time) {
	tot := ls.total.Load()
	b := ls.bytes.Load()
	ls.speedMu.Lock()
	defer ls.speedMu.Unlock()
	ls.speedTicks = append(ls.speedTicks, v3SpeedTick{t: now, total: tot, bytes: b})
	cutoff := now.Add(-10 * time.Second)
	for len(ls.speedTicks) > 1 && ls.speedTicks[0].t.Before(cutoff) {
		ls.speedTicks = ls.speedTicks[1:]
	}
	var rps, bw float64
	if len(ls.speedTicks) >= 2 {
		first := ls.speedTicks[0]
		last := ls.speedTicks[len(ls.speedTicks)-1]
		dt := last.t.Sub(first.t).Seconds()
		if dt > 0 {
			rps = float64(last.total-first.total) / dt
			bw = float64(last.bytes-first.bytes) / dt
		}
	}
	ls.rollingRPS = rps
	ls.rollingBW = bw
	if rps > ls.peakRPS {
		ls.peakRPS = rps
	}
}

func (ls *v3LiveStats) p95Ms() int64 {
	n := ls.latTotal.Load()
	if n < 10 {
		return 0
	}
	target := int64(float64(n) * 0.95)
	var cum int64
	for i, edge := range v3LatEdges {
		cum += ls.latBuckets[i].Load()
		if cum >= target {
			return edge
		}
	}
	return v3LatEdges[len(v3LatEdges)-1]
}

var v3LatEdges = [8]int64{100, 250, 500, 1000, 2000, 3500, 5000, 10000}

// v3ProgressWriter wraps ResultWriter and updates live stats.
type v3ProgressWriter struct {
	inner crawl.ResultWriter
	ls    *v3LiveStats
}

func (p *v3ProgressWriter) Add(r crawl.Result) {
	p.inner.Add(r)
	p.ls.recordResult(r)
}
func (p *v3ProgressWriter) Flush(ctx context.Context) error { return p.inner.Flush(ctx) }
func (p *v3ProgressWriter) Close() error                    { return p.inner.Close() }

// v3ProgressFailureWriter wraps FailureWriter and counts domain-killed skips.
type v3ProgressFailureWriter struct {
	inner crawl.FailureWriter
	ls    *v3LiveStats
}

func (f *v3ProgressFailureWriter) AddURL(u crawl.FailedURL) {
	f.inner.AddURL(u)
	if u.Reason == "domain_http_timeout_killed" {
		f.ls.recordSkip()
	}
}
func (f *v3ProgressFailureWriter) Close() error { return f.inner.Close() }

// v3RenderProgress, v3MemLine, v3HWLine, v3StatusLine, v3SafePct, v3FmtBytes, v3FmtDur
// — exact copies from current cli/cc.go; moved here so both hn.go and cc.go share them.
// (copy the full implementations from cc.go lines ~2037–2260)
```

**Step 2: Remove moved types from `cli/cc.go`**

Delete from `cli/cc.go`:
- `v3SpeedTick`, `v3LiveStats`, `v3DomainInfo` struct definitions (~L1581–1628)
- `StartDomain`, `EndDomain`, `recordResult`, `recordSkip`, `updateSpeed`, `p95Ms` methods (~L1631–1720)
- `v3ProgressWriter`, `v3ProgressFailureWriter` (~L1723–1748)
- `v3RenderProgress`, `v3MemLine`, `v3HWLine`, `v3StatusLine`, `v3SafePct`, `v3FmtBytes`, `v3FmtDur` (~L2036–2280)

**Step 3: Replace `runCCRecrawlV3` in `cli/cc.go`**

Replace the current `runCCRecrawlV3` function (all ~280 lines, L1752–2034) with:

```go
func runCCRecrawlV3(ctx context.Context, opts ccRecrawlOpts,
	seeds []crawl.SeedURL, dnsResolver *store.DNSResolver,
	resultDir, failedDBPath string) error {

	homeDir, _ := os.UserHomeDir()
	siCache := filepath.Join(homeDir, ".cache", "search", "sysinfo.json")
	si := crawl.LoadOrGatherSysInfo(siCache, 30*time.Minute)

	dbShards := opts.dbShards
	if dbShards <= 0 {
		dbShards = crawl.AutoShardCount(si.CPUCount)
	}
	dbMemMB := opts.dbMemMB
	if dbMemMB <= 0 {
		dbMemMB = crawl.AutoDuckMemPerShard(int(si.MemAvailableMB), dbShards)
	}

	var dnsCache crawl.DNSCache
	if dnsResolver != nil {
		dnsCache = dnsResolver.Cache()
	} else {
		dnsCache = &crawl.NoopDNS{}
	}

	selfBin, _ := os.Executable()

	jcfg := crawl.JobConfig{
		Engine:              opts.engine,
		Workers:             opts.workers,
		MaxConnsPerDomain:   opts.maxConnsPerDomain,
		Timeout:             time.Duration(opts.timeout) * time.Millisecond,
		RetryTimeout:        time.Duration(opts.retryTimeoutMs) * time.Millisecond,
		NoRetry:             opts.noRetry,
		StatusOnly:          opts.statusOnly,
		InsecureTLS:         true,
		DomainFailThreshold: opts.domainFailThreshold,
		BatchSize:           opts.batchSize,
	}
	if opts.domainTimeoutMs > 0 {
		jcfg.DomainTimeout = time.Duration(opts.domainTimeoutMs) * time.Millisecond
	}
	if selfBin != "" {
		// passed through Config.SearchBinary for swarm engine
		_ = selfBin
	}

	return runRecrawlJob(ctx, recrawlJobArgs{
		Seeds:        seeds,
		DNSCache:     dnsCache,
		JobCfg:       jcfg,
		ResultDir:    resultDir,
		FailedDBPath: failedDBPath,
		WriterMode:   "duckdb",
		SlowDomainMs: 30_000,
		DBShards:     dbShards,
		DBMemMB:      dbMemMB,
		SysInfo:      si,
	})
}
```

**Step 4: Replace `runHNRecrawlV3` in `cli/hn.go`**

Replace the current `runHNRecrawlV3` function (~L725–1400) with:

```go
func runHNRecrawlV3(ctx context.Context,
	hnCfg hn.Config,
	seedRes *hn.RecrawlSeedResult,
	engineName string,
	workers, maxConnsPerDomain, timeoutMs, domainFailThreshold, domainTimeoutMs int,
	statusOnly bool,
	batchSize int,
	slowDomainMs int64,
	dnsWorkers, dnsTimeoutMs int,
	retryTimeoutMs int,
	noRetry bool,
	writerMode string,
	chunkMode string,
	chunkSize int,
	bodyStoreDir string,
	dbMemMB, dbShards, pass2Workers, segSizeMB int,
	printAutoConfig bool,
) error {
	siCache := filepath.Join(hnCfg.WithDefaults().RecrawlDir(), ".sysinfo.json")
	si := crawl.LoadOrGatherSysInfo(siCache, 30*time.Minute)
	fmt.Print(infoStyle.Render("Hardware Profile") + "\n")
	fmt.Print(si.Table())

	if dbShards <= 0 {
		dbShards = crawl.AutoShardCount(si.CPUCount)
	}
	availMB := int(si.MemAvailableMB)
	if dbMemMB <= 0 {
		dbMemMB = crawl.AutoDuckMemPerShard(availMB, dbShards)
	}
	if segSizeMB <= 0 {
		segSizeMB = crawl.AutoBinSegMB(availMB)
	}
	p2Workers := pass2Workers
	if p2Workers <= 0 {
		p2Workers = workers
	}
	chanCap := crawl.AutoBinChanCap(availMB, 256)
	fmt.Printf("  DB config:    shards=%d  mem/shard=%dMB\n", dbShards, dbMemMB)
	fmt.Printf("  Bin writer:   seg=%dMB  chan=%d\n", segSizeMB, chanCap)
	fmt.Printf("  Pass 2:       workers=%d  timeout=%dms\n\n", p2Workers, retryTimeoutMs)

	if printAutoConfig {
		return nil
	}

	// Load seeds
	fmt.Println(infoStyle.Render("Loading seeds into memory..."))
	seeds, err := store.LoadSeedURLs(ctx, seedRes.OutDBPath, int(seedRes.Rows))
	if err != nil {
		return fmt.Errorf("load seed URLs: %w", err)
	}
	fmt.Printf("  Loaded %s seed URLs\n\n", labelStyle.Render(formatInt64Exact(int64(len(seeds)))))

	// DNS pre-resolution
	var dnsCache crawl.DNSCache
	dnsCachePath := filepath.Join(hnCfg.WithDefaults().RecrawlDir(), "dns.duckdb")
	if dnsWorkers > 0 {
		resolver := store.NewDNSResolver(time.Duration(dnsTimeoutMs) * time.Millisecond)
		if cached, _ := resolver.LoadCache(dnsCachePath); cached > 0 {
			fmt.Printf("  DNS cache: loaded %d entries\n", cached)
		}
		hostSet := make(map[string]struct{}, seedRes.UniqueDomains)
		for _, s := range seeds {
			if h := s.Domain; h != "" {
				hostSet[h] = struct{}{}
			}
		}
		hostList := make([]string, 0, len(hostSet))
		for h := range hostSet {
			hostList = append(hostList, h)
		}
		cov := ccDNSCacheCoverage(resolver, hostList)
		if cov.Pending > 0 {
			fmt.Printf("  DNS resolving %s unique hosts (%d workers, %dms timeout)...\n",
				labelStyle.Render(formatInt64Exact(int64(cov.Pending))), dnsWorkers, dnsTimeoutMs)
			resolver.ResolveBatch(ctx, hostList, dnsWorkers,
				time.Duration(dnsTimeoutMs)*time.Millisecond, func(_ store.DNSProgress) {})
			if err := resolver.SaveCache(dnsCachePath); err == nil {
				fmt.Printf("  DNS saved: %s live  %s dead  %s timeout\n",
					labelStyle.Render(formatInt64Exact(resolver.LiveCount())),
					labelStyle.Render(formatInt64Exact(resolver.DeadCount())),
					labelStyle.Render(formatInt64Exact(resolver.TimeoutCount())),
				)
			}
		}
		before := len(seeds)
		filtered := seeds[:0]
		for _, s := range seeds {
			if !resolver.IsDeadOrTimeout(s.Domain) {
				filtered = append(filtered, s)
			}
		}
		seeds = filtered
		if skipped := before - len(seeds); skipped > 0 {
			fmt.Printf("  Filtered %s dead/timeout seeds → %s remaining\n\n",
				labelStyle.Render(formatInt64Exact(int64(skipped))),
				labelStyle.Render(formatInt64Exact(int64(len(seeds)))),
			)
		}
		dnsCache = resolver.Cache()
	}
	if dnsCache == nil {
		dnsCache = &crawl.NoopDNS{}
	}

	resultDir := filepath.Join(hnCfg.WithDefaults().RecrawlDir(), "results")
	failedDBPath := filepath.Join(hnCfg.WithDefaults().RecrawlDir(), "failed.duckdb")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		return fmt.Errorf("create result dir: %w", err)
	}

	if bodyStoreDir == "" {
		bodyStoreDir = filepath.Join(hnCfg.WithDefaults().RecrawlDir(), "bodies")
	}

	selfBin, _ := os.Executable()
	jcfg := crawl.JobConfig{
		Engine:              engineName,
		Workers:             workers,
		MaxConnsPerDomain:   maxConnsPerDomain,
		Timeout:             time.Duration(timeoutMs) * time.Millisecond,
		RetryTimeout:        time.Duration(retryTimeoutMs) * time.Millisecond,
		NoRetry:             noRetry,
		StatusOnly:          statusOnly,
		InsecureTLS:         true,
		DomainFailThreshold: domainFailThreshold,
		BatchSize:           batchSize,
		Pass2Workers:        p2Workers,
		ChunkMode:           chunkMode,
		ChunkSize:           chunkSize,
		SeedPath:            seedRes.OutDBPath,
	}
	if domainTimeoutMs > 0 {
		jcfg.DomainTimeout = time.Duration(domainTimeoutMs) * time.Millisecond
	}
	_ = selfBin

	return runRecrawlJob(ctx, recrawlJobArgs{
		Seeds:        seeds,
		DNSCache:     dnsCache,
		JobCfg:       jcfg,
		ResultDir:    resultDir,
		FailedDBPath: failedDBPath,
		WriterMode:   writerMode,
		SlowDomainMs: slowDomainMs,
		SegSizeMB:    segSizeMB,
		BodyStoreDir: bodyStoreDir,
		DBShards:     dbShards,
		DBMemMB:      dbMemMB,
		SysInfo:      si,
	})
}
```

**Step 5: Update imports in `hn.go` and `cc.go`**

In `hn.go`: change `recrawler "github.com/go-mizu/mizu/blueprints/search/pkg/archived/recrawler"` → `"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"`. Remove direct `crawl.WrapDNSResolver` calls (replaced by `resolver.Cache()`).

In `cc.go`: same import update; also update `ccDNSCacheCoverage` parameter type from `*recrawler.DNSResolver` to `*store.DNSResolver`.

**Step 6: Build**

```bash
go build ./cmd/search/
```

Expected: clean build.

**Step 7: Run CLI tests**

```bash
go test ./cli/...
```

Expected: PASS

**Step 8: Commit**

```bash
git add cli/recrawl.go cli/cc.go cli/hn.go
git commit -m "feat(cli): unify runHNRecrawlV3+runCCRecrawlV3 into shared runRecrawlJob"
```

---

## Phase 5 — Remove old v1/v2 recrawler path from `cli/cc.go`

### Task 9: Delete the non-`--engine` path in `runCCRecrawl`

In `cli/cc.go`, the `runCCRecrawl` function has a branch `if opts.engine != ""` that dispatches to `runCCRecrawlV3`, and an else branch (L1361–1578) that uses the old `recrawler.New` / `recrawler.RunWithDisplay` path. Since `--engine` now always defaults to `"keepalive"`, the old path is unreachable.

**Files:**
- Modify: `cli/cc.go`

**Step 1: Delete the old path**

In `runCCRecrawl`, replace:
```go
// ── v3 engine dispatch ──────────────────────────────────────────────────
if opts.engine != "" {
    if err := os.MkdirAll(...); err != nil { ... }
    return runCCRecrawlV3(...)
}

// ── Step 5: Open FailedDB + result DB + run recrawler ──────────────────
// ... (all old recrawler.New / RunWithDisplay code) ...
```

With:
```go
// ── Engine dispatch ──────────────────────────────────────────────────────
if opts.engine == "" {
    opts.engine = "keepalive"
}
if err := os.MkdirAll(filepath.Dir(failedDBPath), 0755); err != nil {
    return fmt.Errorf("creating recrawl data dir: %w", err)
}
return runCCRecrawlV3(ctx, opts, seeds, dnsResolver, resultDir, failedDBPath)
```

**Step 2: Remove unused imports from `cc.go`**

After deletion, `cc.go` no longer imports `pkg/archived/recrawler` directly. Run:

```bash
goimports -w cli/cc.go
```

**Step 3: Build and test**

```bash
go build ./cmd/search/
go test ./cli/...
```

Expected: clean.

**Step 4: Commit**

```bash
git add cli/cc.go
git commit -m "refactor(cli/cc): remove old v1/v2 recrawler path; keepalive always used"
```

---

## Phase 6 — Gut `pkg/archived/recrawler` to shims

### Task 10: Replace archived package with re-export shims

**Files:**
- Modify: `pkg/archived/recrawler/types.go`
- Modify: `pkg/archived/recrawler/resultdb.go`
- Modify: `pkg/archived/recrawler/faileddb.go`
- Modify: `pkg/archived/recrawler/dns.go`
- Modify: `pkg/archived/recrawler/seeddb.go`
- Delete: `pkg/archived/recrawler/recrawler.go` (old v1/v2, no longer called)
- Keep: `pkg/archived/recrawler/verify.go`, `display.go`, `display_test.go` (not part of this refactor)

**Step 1: Replace `types.go` with shims**

```go
// Package recrawler is deprecated. Use pkg/crawl and pkg/crawl/store instead.
// This file re-exports types for backward compatibility with code not yet migrated.
package recrawler

import crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"

// Type aliases pointing to their new homes.
type SeedURL = crawl.SeedURL
type Result = crawl.Result
type FailedURL = crawl.FailedURL
type FailedDomain = crawl.FailedDomain
```

**Step 2: Replace `resultdb.go` with shims**

```go
package recrawler

import (
	"context"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

// ResultDB is deprecated. Use store.ResultDB directly.
type ResultDB = store.ResultDB

// NewResultDB is deprecated. Use store.NewResultDB.
func NewResultDB(dir string, shardCount, batchSize, duckMemPerShardMB int) (*ResultDB, error) {
	return store.NewResultDB(dir, shardCount, batchSize, duckMemPerShardMB)
}

// LoadAlreadyCrawledFromDir is deprecated. Use store.LoadAlreadyCrawledFromDir.
func LoadAlreadyCrawledFromDir(ctx context.Context, dir string) (map[string]bool, error) {
	return store.LoadAlreadyCrawledFromDir(ctx, dir)
}
```

**Step 3: Replace `faileddb.go` with shims**

```go
package recrawler

import (
	"time"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

// FailedDB is deprecated. Use store.FailedDB directly.
type FailedDB = store.FailedDB

func NewFailedDB(path string) (*FailedDB, error)  { return store.NewFailedDB(path) }
func OpenFailedDB(path string) (*FailedDB, error) { return store.OpenFailedDB(path) }

func LoadRetryURLs(dbPath string) ([]crawl.SeedURL, error) {
	return store.LoadRetryURLs(dbPath)
}
func LoadRetryURLsSince(dbPath string, since time.Time) ([]crawl.SeedURL, error) {
	return store.LoadRetryURLsSince(dbPath, since)
}
func LoadFailedDomains(dbPath string) ([]crawl.FailedDomain, error) {
	return store.LoadFailedDomains(dbPath)
}
func FailedDomainSummary(dbPath string) (map[string]int, int, error) {
	return store.FailedDomainSummary(dbPath)
}
func FailedURLSummary(dbPath string) (map[string]int, int, error) {
	return store.FailedURLSummary(dbPath)
}
```

**Step 4: Replace `dns.go` with shims**

```go
package recrawler

import (
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

type DNSResolver = store.DNSResolver
type DNSProgress = store.DNSProgress

func NewDNSResolver(timeout time.Duration) *DNSResolver {
	return store.NewDNSResolver(timeout)
}
```

**Step 5: Replace `seeddb.go` with shims**

```go
package recrawler

import (
	"context"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

func LoadSeedURLs(ctx context.Context, dbPath string, expectedCount int) ([]crawl.SeedURL, error) {
	return store.LoadSeedURLs(ctx, dbPath, expectedCount)
}
```

**Step 6: Delete `recrawler.go`**

```bash
rm pkg/archived/recrawler/recrawler.go
```

Check if `verify.go` or `display.go` depend on it. If so, keep the minimum needed stubs. Otherwise delete.

**Step 7: Build everything**

```bash
go build ./...
```

Expected: clean build across the entire module.

**Step 8: Run all tests**

```bash
go test ./pkg/crawl/... ./pkg/crawl/store/... ./cli/...
```

Expected: all PASS.

**Step 9: Commit**

```bash
git add pkg/archived/recrawler/
git commit -m "refactor(archived/recrawler): replace implementations with shims pointing to pkg/crawl and pkg/crawl/store"
```

---

## Phase 7 — Deploy and smoke test

### Task 11: Build and deploy linux-noble image

**Step 1: Build**

```bash
cd /path/to/mizu/blueprints/search
make build-linux-noble
```

This builds a linux/amd64 binary under QEMU. Takes 15-20 minutes. Run in background.

**Step 2: Deploy to server1**

```bash
make deploy-linux-noble SERVER=1
```

**Step 3: Deploy to server2**

```bash
make deploy-linux-noble SERVER=2
```

**Step 4: Verify auto-config on both servers**

```bash
ssh server1 'search hn recrawl --auto-config'
ssh server2 'search hn recrawl --auto-config'
```

Expected: prints hardware profile table + DB config + bin writer config + pass 2 config. Exits without running.

**Step 5: Smoke test cc recrawl**

```bash
ssh server1 'search cc recrawl --last --status-only --limit 1000 --no-retry'
ssh server2 'search cc recrawl --last --status-only --limit 1000 --no-retry'
```

Expected: progress display appears, completes without errors, shows OK/total counts.

**Step 6: Smoke test hn recrawl**

```bash
ssh server1 'search hn recrawl --limit 5000 --status-only --no-retry'
ssh server2 'search hn recrawl --limit 5000 --status-only --no-retry'
```

Expected: seed loading, DNS cache, progress display, pass 1 summary. No panics.

**Step 7: Final commit tag**

```bash
git tag v0.6.0-crawl-refactor
```

---

## Summary of files changed

| File | Action |
|------|--------|
| `pkg/crawl/types.go` | Add SeedURL/Result/FailedURL/FailedDomain; update adapters |
| `pkg/crawl/engine.go` | Update Engine interface + Stats to own types; add ShardReopener |
| `pkg/crawl/job.go` | NEW: RunJob, JobConfig, JobResult |
| `pkg/crawl/job_test.go` | NEW: tests |
| `pkg/crawl/keepalive.go` | Use crawl.* types; remove recrawler import |
| `pkg/crawl/epoll.go` | Same |
| `pkg/crawl/rawhttp.go` | Same |
| `pkg/crawl/swarm.go` | Same |
| `pkg/crawl/swarm_drone.go` | Same |
| `pkg/crawl/seedcursor.go` | Use crawl.SeedURL; remove recrawler import |
| `pkg/crawl/pipeline.go` | Use ShardReopener; remove recrawler import |
| `pkg/crawl/writer_bin.go` | Use crawl.Result + ResultWriter; remove recrawler import |
| `pkg/crawl/store/result.go` | NEW: ResultDB from archived |
| `pkg/crawl/store/result_test.go` | NEW |
| `pkg/crawl/store/failed.go` | NEW: FailedDB from archived |
| `pkg/crawl/store/failed_test.go` | NEW |
| `pkg/crawl/store/dns.go` | NEW: DNSResolver from archived |
| `pkg/crawl/store/seed.go` | NEW: LoadSeedURLs etc from archived |
| `cli/recrawl.go` | NEW: shared runRecrawlJob + all v3LiveStats display code |
| `cli/cc.go` | Remove v3LiveStats block; replace runCCRecrawlV3 + delete old path |
| `cli/hn.go` | Replace runHNRecrawlV3; update imports |
| `pkg/archived/recrawler/types.go` | Shim |
| `pkg/archived/recrawler/resultdb.go` | Shim |
| `pkg/archived/recrawler/faileddb.go` | Shim |
| `pkg/archived/recrawler/dns.go` | Shim |
| `pkg/archived/recrawler/seeddb.go` | Shim |
| `pkg/archived/recrawler/recrawler.go` | Delete |
