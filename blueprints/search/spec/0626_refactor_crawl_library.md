# spec/0626: Refactor `pkg/crawl` into a Self-Contained Library

**Status:** planned
**Date:** 2026-02-28

---

## Motivation

`pkg/crawl` currently imports `pkg/archived/recrawler` for its core types (`SeedURL`, `Result`, `FailedURL`) and storage (`ResultDB`, `FailedDB`). This means the package is not self-contained and cannot be understood or tested independently.

In addition, `cli/hn.go` (`runHNRecrawlV3`, ~350 lines) and `cli/cc.go` (`runCCRecrawlV3`, ~280 lines) contain near-identical two-pass crawl execution logic embedded with display code. This duplication causes drift every time the runner changes.

This spec covers:
1. Moving core types and storage from `pkg/archived/recrawler` to `pkg/crawl` and `pkg/crawl/store`
2. Extracting the two-pass runner into `pkg/crawl.RunJob`
3. Collapsing the CLI duplication into a single shared `runRecrawlJob` function
4. Reviewing and tidying all conventions, naming, locking, memory, and logging

---

## Package Structure (target)

```
pkg/crawl/                          # core library — engine + job runner
├── engine.go                       # Engine interface, New(), Stats, Config, DomainNotifier
├── types.go                        # SeedURL, Result, FailedURL, FailedDomain
│                                   # ResultWriter, FailureWriter, DNSCache, NoopDNS
├── job.go                          # NEW: RunJob(), JobConfig, JobResult
├── autoconfig.go                   # hardware auto-tuning (unchanged)
├── pipeline.go                     # RunPipeline, PipelineConfig (updated for internal types)
├── seedcursor.go                   # SeedCursor (updated for internal types)
├── keepalive.go                    # KeepAlive engine (updated for internal types)
├── epoll.go, rawhttp.go, swarm.go, swarm_drone.go
├── writer_bin.go                   # BinSegWriter (updated for internal types)
├── writer_devnull.go               # DevNull writers
├── chunkbench.go                   # BenchTracker
├── sysinfo.go + platform variants
├── hw_monitor.go + platform variants
├── rlimit_*.go
│
├── store/                          # DuckDB storage — separate import to avoid CGO bleed
│   ├── result.go                   # ResultDB (from archived/recrawler/resultdb.go)
│   ├── failed.go                   # FailedDB (from archived/recrawler/faileddb.go)
│   ├── dns.go                      # DNSResolver, DNSProgress (from archived/recrawler/dns.go)
│   └── seed.go                     # LoadSeedURLs, LoadRetryURLsSince (from archived/recrawler/seeddb.go)
│
├── bseg/                           # binary segment encoding — unchanged
└── bodystore/                      # content-addressable body store — unchanged
```

### Import rule

`pkg/crawl/store` imports `pkg/crawl` (for shared types).
`pkg/crawl` does **not** import `pkg/crawl/store`.
`RunJob` uses injected constructor functions to open/close storage — this breaks the cycle cleanly.

### pkg/archived/recrawler (after)

Gutted to thin re-export shims pointing to `pkg/crawl` and `pkg/crawl/store`. The old v1/v2 `recrawler.New` / `RunWithDisplay` code path (reachable from `cli/cc.go` when `--engine` is not set) is removed — `keepalive` is now always used.

---

## Type Migration

Core domain types move from `pkg/archived/recrawler/types.go` → `pkg/crawl/types.go`:

```go
// pkg/crawl/types.go

type SeedURL struct {
    URL    string
    Domain string
    Host   string
}

type Result struct {
    URL           string
    StatusCode    int
    ContentType   string
    ContentLength int64
    Body          string  // always "" (overflow string fix — see spec/0618)
    BodyCID       string  // CAS reference; "" = not stored
    Title         string
    Description   string
    Language      string
    Domain        string
    RedirectURL   string
    FetchTimeMs   int64
    CrawledAt     time.Time
    Error         string
}

type FailedURL struct {
    URL         string
    Domain      string
    Reason      string
    Error       string
    StatusCode  int
    FetchTimeMs int64
    ContentType string
    RedirectURL string
    DetectedAt  time.Time
}

type FailedDomain struct {
    Domain     string
    Reason     string
    Error      string
    IPs        string
    URLCount   int
    Stage      string
    DetectedAt time.Time
}

type ResultWriter interface {
    Add(Result)
    Flush(ctx context.Context) error
    Close() error
}

type FailureWriter interface {
    AddURL(FailedURL)
    Close() error
}
```

### Adapter cleanup

Current `pkg/crawl/types.go` has `ResultDBWriter` and `FailedDBWriter` wrapper structs. After migration:
- `store.ResultDB` implements `crawl.ResultWriter` directly (no wrapper needed)
- `store.FailedDB` implements `crawl.FailureWriter` directly (no wrapper needed)

`crawl.WrapDNSResolver(r)` becomes `(*store.DNSResolver).Cache() crawl.DNSCache` — a method on the resolver.

---

## Job Runner (`pkg/crawl/job.go`)

The two-pass crawl execution logic extracted from the CLI into a library function.

```go
// JobConfig configures a two-pass recrawl job.
type JobConfig struct {
    Engine            string        // "keepalive" | "epoll" | "swarm" | "rawhttp"
    Workers           int           // -1 = auto from SysInfo
    MaxConnsPerDomain int           // -1 = auto from SysInfo
    Timeout           time.Duration // pass-1 per-request timeout
    RetryTimeout      time.Duration // pass-2 timeout; 0 or NoRetry=true = skip pass 2
    NoRetry           bool

    StatusOnly          bool
    InsecureTLS         bool
    DomainFailThreshold int
    DomainTimeout       time.Duration
    BatchSize           int

    SysInfo *SysInfo // nil = auto-gather (LoadOrGatherSysInfo)

    // Storage — injected by caller (typically pkg/crawl/store implementations).
    // If OpenResultWriter is nil, results are discarded (DevNull).
    // If OpenFailureWriter is nil, failures are discarded.
    OpenResultWriter  func() (ResultWriter, error)
    OpenFailureWriter func() (FailureWriter, error)

    // LoadRetrySeeds is called after pass 1 to fetch URLs for pass 2.
    // If nil or returns empty, pass 2 is skipped regardless of RetryTimeout.
    // Signature matches store.LoadRetryURLsSince for direct assignment.
    LoadRetrySeeds func(ctx context.Context, since time.Time) ([]SeedURL, error)

    // Progress — attached by CLI for live display.
    Notifier DomainNotifier // domain lifecycle callbacks

    // ChunkMode controls how seeds are fed to the engine: "stream" | "batch" | "pipeline"
    ChunkMode string
    ChunkSize int    // domains per batch (0 = auto)
    SeedPath  string // for pipeline mode (cursor reads from DB directly)

    // Pass2Workers overrides worker count for pass 2 (0 = same as pass 1)
    Pass2Workers int
}

// JobResult holds combined results from both passes.
type JobResult struct {
    Pass1 *Stats
    Pass2 *Stats  // nil if pass 2 was not run
    Total *Stats  // merged pass1 + pass2
    Start time.Time
    End   time.Time
}

// RunJob executes a two-pass recrawl job.
//
// Pass 1: runs engine with cfg.Timeout; writes results + failures.
// Pass 2: if RetryTimeout > 0 and LoadRetrySeeds is set, reopens failure storage,
//         loads timeout URLs from the current run (time-filtered), and re-runs
//         the engine with cfg.RetryTimeout and DomainFailThreshold=0.
//
// Hardware auto-config (GOMEMLIMIT, workers) is applied when Workers <= 0.
func RunJob(ctx context.Context, seeds []SeedURL, dns DNSCache, cfg JobConfig) (*JobResult, error)
```

### Pass-2 lifecycle (inside RunJob)

```
1. OpenResultWriter()         → pass1ResultWriter (kept open for both passes)
2. OpenFailureWriter()        → pass1FailureWriter
3. eng.Run(seeds, pass1ResultWriter, pass1FailureWriter)  [pass 1]
4. pass1FailureWriter.Close() → releases DuckDB lock
5. LoadRetrySeeds(ctx, runStart) → retrySeeds
6. if len(retrySeeds) > 0:
     OpenFailureWriter()      → pass2FailureWriter (appends to same file)
     retryCfg = cfg with RetryTimeout, DomainFailThreshold=0
     eng2.Run(retrySeeds, pass1ResultWriter, pass2FailureWriter)  [pass 2]
     pass2FailureWriter.Close()
7. pass1ResultWriter.Close()
8. return JobResult{Pass1, Pass2, Total}
```

---

## CLI Unification (`cli/recrawl.go`)

A new shared file replaces the duplicate runner logic in `hn.go` and `cc.go`.

```go
// cli/recrawl.go

type recrawlJobArgs struct {
    Seeds         []crawl.SeedURL
    DNSCache      crawl.DNSCache
    JobCfg        crawl.JobConfig
    ResultDir     string
    FailedDBPath  string
    WriterMode    string  // "duckdb" | "bin" | "devnull"
    SlowDomainMs  int64
    SegSizeMB     int
    BodyStoreDir  string
    DBShards      int
    DBMemMB       int
    SysInfo       crawl.SysInfo
}

func runRecrawlJob(ctx context.Context, args recrawlJobArgs) error
```

`runRecrawlJob` handles:
- Hardware config summary print
- `v3LiveStats` setup + progress goroutine
- Injecting `OpenResultWriter`, `OpenFailureWriter`, `LoadRetrySeeds` into `JobCfg`
- Calling `crawl.RunJob`
- Final stats print

`runHNRecrawlV3` and `runCCRecrawlV3` are deleted and replaced by calls to `runRecrawlJob` after building `recrawlJobArgs` from their respective flags.

---

## Naming Conventions

Following Go stdlib patterns (`database/sql`, `net/http`, `io`):

| Old | New | Reason |
|-----|-----|--------|
| `recrawler.ResultDBWriter` | `store.ResultDB` (implements `ResultWriter` directly) | No wrapper type needed |
| `recrawler.FailedDBWriter` | `store.FailedDB` (implements `FailureWriter` directly) | No wrapper type needed |
| `crawl.WrapDNSResolver(r)` | `r.Cache()` — method on `*store.DNSResolver` | Method is more natural than free function |
| `recrawler.OpenFailedDB` | `store.OpenFailedDB` | Same semantics, new home |
| `recrawler.NewResultDB` | `store.NewResultDB` | Same semantics, new home |
| `recrawler.LoadRetryURLsSince` | `store.LoadRetryURLsSince` | Same semantics, new home |
| `recrawler.SeedURL` | `crawl.SeedURL` | Belongs with the engine types |
| `recrawler.Result` | `crawl.Result` | Belongs with the engine types |

Constructors return `(*T, error)`. Open functions (`OpenFailedDB`) remove stale locks before opening. Read-only functions (`LoadRetryURLsSince`) are package-level, not methods.

---

## Logging

Store packages currently use `fmt.Fprintf(os.Stderr, "[resultdb] ...")` for internal soft errors. After refactoring:
- Hard errors are returned to the caller (no change needed — already done)
- Soft/internal errors (e.g. `SET checkpoint_threshold` fails) use `slog.Default().Warn(...)` so callers can redirect if needed
- No new logger interface is introduced (YAGNI — `slog.Default()` is sufficient)

---

## Memory and Locking Review

No issues found in the current implementation. Noted for documentation:

| Point | Assessment |
|-------|-----------|
| `ResultDB.flushCh cap=2` | Intentional back-pressure; prevents body accumulation |
| `FailedDB.urlCh cap=100000` | ~20 MB at scale; fine for current workloads |
| `BinSegWriter.ch` cap auto-tuned | `AutoBinChanCap` scales with available RAM |
| `Pipeline.batchCh cap=1` | Intentional single-batch look-ahead |
| `ResultDB.Close()` calls `Flush` before closing | Could block if `flushCh` is full; acceptable since Close is called at shutdown |

---

## Database Schema

No schema changes. No index creation (DuckDB columnar storage handles range scans efficiently without explicit indexes; index creation would slow writes at high throughput).

Existing `results` table is preserved as-is. The `body` column stays empty per the overflow string fix (spec/0618); this is documented in the schema comment.

---

## Directory Structure

No changes to data directory layout. Existing paths remain:

```
$HOME/data/hn/recrawl/
    results/            ResultDB shards
    failed.duckdb       FailedDB
    dns.duckdb          DNS cache
    segments/           BinSeg (bin writer mode)
    bodies/             Body CAS store

$HOME/data/common-crawl/{crawl-id}/{parquet-slug}/
    results/
    failed.duckdb
    dns.duckdb
```

---

## Implementation Plan

### Phase 1 — Type extraction (no behavior change)
1. Add `SeedURL`, `Result`, `FailedURL`, `FailedDomain`, `ResultWriter`, `FailureWriter` to `pkg/crawl/types.go`; keep `recrawler.*` as type aliases in archived for backward compat
2. Update all `pkg/crawl/*.go` files to use internal types instead of `recrawler.*`
3. Update `pkg/crawl/engine.go` `ResultWriter`/`FailureWriter` to use internal types
4. Verify: `go build ./pkg/crawl/...`

### Phase 2 — Store sub-package
5. Create `pkg/crawl/store/` with `result.go`, `failed.go`, `dns.go`, `seed.go`
6. `store.ResultDB` and `store.FailedDB` implement `crawl.ResultWriter`/`crawl.FailureWriter` directly
7. Add `(*DNSResolver).Cache() crawl.DNSCache` method replacing `crawl.WrapDNSResolver`
8. Update `pkg/crawl/writer_bin.go` and `pkg/crawl/pipeline.go` to use `*store.ResultDB` via interface
9. Verify: `go build ./pkg/crawl/store/...`

### Phase 3 — Job runner
10. Implement `pkg/crawl/job.go` with `RunJob`, `JobConfig`, `JobResult`
11. Unit test: `go test ./pkg/crawl/...`

### Phase 4 — CLI unification
12. Create `cli/recrawl.go` with `runRecrawlJob`, `recrawlJobArgs`
13. Rewrite `runHNRecrawlV3` in `hn.go` to call `runRecrawlJob`
14. Rewrite `runCCRecrawlV3` in `cc.go` to call `runRecrawlJob`
15. Remove old v1/v2 non-`--engine` path from `cc.go`
16. Verify: `go build ./cmd/search/`

### Phase 5 — Cleanup
17. Gut `pkg/archived/recrawler/` to thin shims (re-export from `pkg/crawl` + `pkg/crawl/store`)
18. Remove `recrawler.ResultDBWriter`, `recrawler.FailedDBWriter`, `crawl.WrapDNSResolver`
19. Full build + tests: `go test ./...`

### Phase 6 — Deploy and verify
20. Build linux-noble Docker image
21. Deploy to server1 and server2
22. Run `search hn recrawl --auto-config` on both — verify config output
23. Run `search cc recrawl --last --status-only --limit 1000` on both — verify pipeline
24. Run `search hn recrawl --limit 50000` smoke test
