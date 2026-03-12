# spec/0701 — Store Refactor: Centralized DuckDB-backed Store Interfaces

## Status: Draft — 2026-03-09

## 1. Problem Statement

The codebase has grown a collection of store types spread across many packages, each with its own patterns:

| Package | Type | Backend | Interface? |
|---------|------|---------|------------|
| `pkg/index/web/metastore` | `Store` | DuckDB / SQLite | ✅ clean |
| `pkg/index/web` | `DocStore` | DuckDB (per-shard) | ❌ concrete struct |
| `pkg/index/web` | `DomainStore` | DuckDB | ❌ concrete struct |
| `pkg/index/web` | `CCDomainStore` | DuckDB | ❌ concrete struct |
| `pkg/index/web/pipeline/scrape` | `Store` | DuckDB meta + shard reads | ❌ concrete struct |
| `pkg/crawl/store` | `ResultDB` | DuckDB (sharded) | ❌ concrete struct |
| `pkg/crawl/store` | `FailedDB` | DuckDB | ❌ concrete struct |
| `pkg/crawl/store` | `DNSResolver` | DuckDB cache | ❌ concrete struct |
| `pkg/engine/fineweb` | `Store` | DuckDB | ❌ concrete struct |
| `pkg/serp` | `Store` | JSON file | n/a (not DuckDB) |
| `pkg/crawl/warcstore` | `Store` | Filesystem | n/a (not DuckDB) |
| `pkg/index/driver/flower/dahlia` | `storeWriter/Reader` | Binary blocks | n/a (internal) |

**Pain points:**
- No way to swap implementations for testing — callers hold concrete types
- DuckDB initialization scattered: `SetMaxOpenConns(1)` sometimes missing
- Constructor signatures differ: some take `dir`, some `path`, some have many positional args
- No consistent options pattern for DuckDB tuning (memory_limit, threads, etc.)
- `pkg/index/web` exposes three concrete structs that all interact; callers use them directly

## 2. Goals

1. **Define `Store` interfaces** for every DuckDB-backed store — callers program against interfaces
2. **Standardize constructors** to `Open(path string, opts ...Option) (*Store, error)` or `New(cfg Config) (*Store, error)`
3. **Enforce single-connection rule** — all DuckDB stores call `db.SetMaxOpenConns(1)` immediately after `sql.Open`
4. **Common `Option` pattern** per package, covering at least `WithMemoryLimitMB`, `WithBatchSize`
5. **Context on all query methods** — `ctx context.Context` is always the first parameter
6. **`Close() error` on every store** — always present, always safe to call multiple times via `sync.Once`
7. Follow the `database/sql` / `net/http` design philosophy: thin interfaces, concrete drivers, no magic

## 3. Out of Scope

The following stores are intentionally excluded — they are not DuckDB-backed or are private internals:

- `pkg/serp/store.go` — JSON file, appropriately lightweight
- `pkg/crawl/warcstore/store.go` — filesystem WARC writer
- `pkg/index/driver/flower/dahlia/store.go` — private binary block store for FTS
- `pkg/index/driver/flower/rose/docstore.go` — private append-only binary store
- `pkg/index/driver/flower/lotus/store.go` — private FTS index driver

## 4. Interface Designs

### 4.1 `pkg/crawl/store` — Crawl Pipeline Stores

Three interfaces for the crawl pipeline, each in the same package.

#### `ResultStore`

```go
// ResultStore writes crawl results to persistent storage.
type ResultStore interface {
    // Add queues a result for batch writing. Never blocks.
    Add(r crawl.Result)
    // Flush sends all pending batches to the underlying writer. Blocks until done.
    Flush(ctx context.Context) error
    // FlushedCount returns the number of results successfully written.
    FlushedCount() int64
    // PendingCount returns the number of results waiting to be flushed.
    PendingCount() int
    // Close flushes remaining results and releases all resources.
    Close() error
}
```

Implemented by: `ResultDB` (sharded DuckDB). Constructor unchanged:
`NewResultDB(dir string, shardCount, batchSize, duckMemPerShardMB int) (*ResultDB, error)`
→ refactor to:
`OpenResultDB(dir string, opts ...ResultOption) (*ResultDB, error)`

#### `FailedStore`

```go
// FailedStore records failed domains and URLs from a crawl run.
type FailedStore interface {
    AddDomain(d FailedDomain)
    AddURL(u FailedURL)
    AddURLBatch(urls []string, reason string)
    DomainCount() int64
    URLCount() int64
    Close() error
}
```

Implemented by: `FailedDB`. Constructor:
`OpenFailedDB(path string, opts ...FailedOption) (*FailedDB, error)`

#### `DNSCache`

The `DNSResolver` is primarily a service (parallel batch resolution), not just a store. Extract the persistence aspect as a `DNSCache` interface backed by DuckDB, keep resolution logic in the struct.

```go
// DNSCache persists and retrieves DNS resolution results.
type DNSCache interface {
    SaveCache(ctx context.Context, path string) error
    LoadCache(ctx context.Context, path string) error
    IsDead(domain string) bool
    IsResolved(domain string) bool
    ResolvedIPs(domain string) ([]string, bool)
}
```

The `DNSResolver` struct satisfies `DNSCache` by embedding the current map-based implementation. No schema change needed.

### 4.2 `pkg/index/web` — Web Index Stores

Three interfaces replacing the concrete structs. Types (`DocRecord`, `DomainRecord`, etc.) stay in the same package.

#### `DocStore`

```go
// DocStore manages per-shard markdown document metadata from WARC archives.
type DocStore interface {
    ScanShard(ctx context.Context, crawlID, shard, warcMdPath string) error
    ScanAll(ctx context.Context, crawlID, crawlBase string) error
    ListDocs(ctx context.Context, crawlID, shard string, page, pageSize int, q, sortBy string) ([]DocRecord, int64, error)
    GetDoc(ctx context.Context, crawlID, shard, docID string) (DocRecord, bool, error)
    ShardStats(ctx context.Context, crawlID, shard string) (DocScanMeta, error)
    Close() error
}
```

Constructor: `OpenDocStore(baseDir string, opts ...DocOption) (*docStore, error)`
(returns concrete type; variable declared as `DocStore` at call sites)

#### `DomainStore`

```go
// DomainStore manages domain URL counts derived from parquet recrawl results.
type DomainStore interface {
    EnsureFresh(ctx context.Context)
    ListDomains(ctx context.Context, sortBy, q string, page, pageSize int) ([]DomainRecord, int64, error)
    ListDomainURLs(ctx context.Context, domain, sortBy, statusGroup string, page, pageSize int) ([]URLRecord, int64, error)
    GetOverviewStats(ctx context.Context) (OverviewStats, error)
    Close() error
}
```

Constructor: `OpenDomainStore(dataDir string, opts ...DomainOption) (*domainStore, error)`

#### `CCDomainStore`

```go
// CCDomainStore caches Common Crawl CDX API results for domain pages.
type CCDomainStore interface {
    FetchAndCache(ctx context.Context, domain, crawlID string, maxURLs int) error
    GetDomainURLs(ctx context.Context, domain, crawlID, sortBy, statusGroup, q string, page, pageSize int) ([]CCURLRecord, int64, error)
    Close() error
}
```

Constructor: `OpenCCDomainStore(dataDir string, opts ...CCDomainOption) (*ccDomainStore, error)`

### 4.3 `pkg/index/web/pipeline/scrape` — Scrape Dashboard Store

```go
// Store reads and caches per-domain dcrawler statistics.
type Store interface {
    ListDomains() (*ListResponse, error)
    GetDomainStats(domain string) (*Domain, error)
    GetPages(domain string, page, pageSize int, q, sortBy, statusFilter string) (*PagesResponse, error)
    GetDomainSummary(domain string) *DomainSummary
    InvalidateCache()
    Close()
}
```

The concrete `*store` (rename to unexported) implements `Store`. Constructor:
`NewStore(dataDir string) Store`

### 4.4 `pkg/engine/fineweb` — FineWeb Document Store

```go
// Store manages FineWeb document import and full-text search.
type Store interface {
    Import(ctx context.Context, parquetDir string, progress func(file string, rows int64)) error
    CreateFTSIndex(ctx context.Context) error
    CreateFTSIndexWithConfig(ctx context.Context, cfg FTSConfig) error
    HasFTSIndex(ctx context.Context) bool
    DropFTSIndex(ctx context.Context) error
    Search(ctx context.Context, query string, limit, offset int) ([]Document, error)
    SearchFTS(ctx context.Context, query string, limit, offset int) (*SearchResult, error)
    Count(ctx context.Context) (int64, error)
    GetImportState(ctx context.Context) ([]ImportState, error)
    Close() error
}
```

Constructor: `Open(lang, dataDir string, opts ...Option) (Store, error)`

### 4.5 `pkg/index/web/metastore` → `pkg/index/web/store` — Flatten & Simplify

The `metastore` sub-package is overengineered for its actual use. It has:
- A 12-method monolithic `Store` interface covering summaries, WARCs, refresh state, and jobs
- A `Driver` interface + global registry (`Register/Open/List`) mimicking `database/sql`
- Two driver implementations (DuckDB and SQLite) — only DuckDB is ever used in production

**Plan: delete the abstraction layer. Move to a concrete DuckDB-backed `*Store`.**

#### What changes

1. **New package**: `pkg/index/web/store/` (replaces `pkg/index/web/metastore/`)

2. **Remove**: `Store` interface, `Driver` interface, `Register/Open/List` registry functions, `Options` struct

3. **Remove**: `drivers/sqlite/` sub-package entirely (SQLite driver is unused in production)

4. **Remove**: `drivers/duckdb/` sub-package (implementation moves up to `store/`)

5. **New `store/store.go`** — concrete `*Store` backed by DuckDB directly:
   ```go
   package store

   // Store is the DuckDB-backed persistence layer for the web index dashboard.
   // It holds crawl summaries, WARC metadata, refresh state, and job history.
   type Store struct {
       db   *sql.DB
       once sync.Once
   }

   func Open(path string, opts ...Option) (*Store, error)
   func (s *Store) Close() error

   // Crawl summary
   func (s *Store) GetSummary(ctx, crawlID) (SummaryRecord, bool, error)
   func (s *Store) PutSummary(ctx, rec SummaryRecord) error

   // WARC metadata
   func (s *Store) ListWARCs(ctx, crawlID) ([]WARCRecord, error)
   func (s *Store) GetWARC(ctx, crawlID, warcIndex) (WARCRecord, bool, error)

   // Refresh state
   func (s *Store) GetRefreshState(ctx, crawlID) (RefreshState, bool, error)
   func (s *Store) SetRefreshState(ctx, st RefreshState) error

   // Job history
   func (s *Store) ListJobs(ctx) ([]JobRecord, error)
   func (s *Store) PutJob(ctx, rec JobRecord) error
   func (s *Store) DeleteAllJobs(ctx) error
   ```

6. **`store/types.go`** — unchanged types: `SummaryRecord`, `WARCRecord`, `JobRecord`, `RefreshState`

7. **`store/options.go`** — functional options:
   ```go
   type Option func(*options)
   func WithBusyTimeout(ms int) Option { ... }
   ```

8. **Update all callers** in `meta_manager.go`, `pipeline/manager.go`, `api/handler.go`, `api/handler_warc.go`, `warc_meta_scan.go`, `warc_api.go`, `server.go`:
   - Replace `metastore.Store` type → `*store.Store`
   - Replace `metastore.Open(driver, dsn, opts)` → `store.Open(path, opts...)`
   - Replace `metastore.WARCRecord` → `store.WARCRecord`, etc.

#### Why remove the interface

The interface existed to support SQLite as an alternative backend. In practice:
- SQLite is only used in `metastore_integration_test.go`
- All production deployments use DuckDB
- The interface provides no testability benefit (the 12-method surface is too large to mock usefully)
- `meta_manager.go` and `pipeline/manager.go` are the only consumers, and they can hold `*store.Store` directly

Testability for `MetaManager` and `Manager` comes from the higher-level focused interfaces (Task 3), not from swapping the underlying persistence layer.

## 5. Common DuckDB Conventions (all packages)

Every DuckDB-backed store MUST follow these rules:

```
1. db.SetMaxOpenConns(1)          — DuckDB single-writer rule; call immediately after sql.Open
2. db.SetMaxIdleConns(1)          — keep one idle connection alive
3. db.SetConnMaxLifetime(0)        — connections live forever (no reconnect churn)
4. initSchema() called in Open()  — CREATE TABLE IF NOT EXISTS on all tables
5. Close() uses sync.Once         — safe to call multiple times; logs but does not panic on error
6. Context propagation            — all query/exec calls use ExecContext / QueryRowContext / QueryContext
```

### Option Pattern (per-package example)

```go
type Option func(*options)

type options struct {
    memoryLimitMB int    // DuckDB SET memory_limit
    busyTimeout   int    // ms, DuckDB busy_timeout
    readOnly      bool
}

func WithMemoryLimitMB(mb int) Option { return func(o *options) { o.memoryLimitMB = mb } }
func WithBusyTimeout(ms int) Option   { return func(o *options) { o.busyTimeout = ms } }
func WithReadOnly() Option            { return func(o *options) { o.readOnly = true } }
```

## 6. Migration Strategy

All refactors are **additive first**:

1. Add the interface definition alongside the existing concrete type
2. Rename concrete struct to unexported (e.g., `DocStore` → `docStore`)
3. Update constructor to return the interface
4. Update all callers to hold the interface type
5. Add `Close()` / `sync.Once` where missing
6. Add `SetMaxOpenConns(1)` where missing
7. Retrofit option pattern to constructor

No schema changes. No data migrations. No behavioral changes.

## 7. File Layout After Refactor

```
pkg/
  crawl/store/
    interface.go          ← NEW: ResultStore, FailedStore, DNSCache interfaces
    result.go             ← resultDB struct (unexported), OpenResultDB()
    failed.go             ← failedDB struct (unexported), OpenFailedDB()
    dns.go                ← DNSResolver (keeps public fields), satisfies DNSCache
    seed.go               ← unchanged (no Store type)

  index/web/
    store.go              ← NEW: DocStore, DomainStore, CCDomainStore interfaces + shared types
    doc_store.go          ← docStore struct (unexported), OpenDocStore()
    domain_store.go       ← domainStore struct (unexported), OpenDomainStore()
    domain_cc_store.go    ← ccDomainStore struct (unexported), OpenCCDomainStore()

    store/                ← RENAMED from metastore/; big Store interface REMOVED
      store.go            ← concrete *Store backed by DuckDB (no interface, no registry)
      types.go            ← SummaryRecord, WARCRecord, JobRecord, RefreshState (unchanged)
      options.go          ← Option funcs: WithBusyTimeout
    metastore/            ← DELETED (drivers/duckdb moved up; drivers/sqlite dropped)

    pipeline/scrape/
      store.go            ← Store interface + types (Domain, Page, etc.)
      store_impl.go       ← NEW: unexported store struct + constructor
      store_meta.go       ← unchanged

  engine/fineweb/
    store.go              ← Store interface + FTSConfig, SearchResult, Document types
    store_impl.go         ← NEW: unexported store struct + constructor
```

## 8. Tasks

See task list in project tracker. Tasks are ordered by dependency:
1. Common conventions doc + helper (no code change, just pattern lock-in)
2. `pkg/crawl/store` interfaces
3. `pkg/index/web` interfaces
4. `pkg/index/web/pipeline/scrape` interface
5. `pkg/engine/fineweb` interface
6. `pkg/index/web/metastore` constructor cleanup
7. Caller updates across cmd/ and server code
8. Tests: add interface-based test mocks for each store

## 9. Non-Goals

- No cross-package consolidation into a single `store` package — domain isolation is valuable
- No query builder / ORM — raw SQL stays
- No connection pooling above 1 — DuckDB's single-writer model is fundamental
- No breaking API changes to CLI commands or HTTP handlers (behavior unchanged)
- No SQLite removal from metastore (existing deployments may use it)
