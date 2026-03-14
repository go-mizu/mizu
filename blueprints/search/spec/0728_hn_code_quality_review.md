# pkg/hn2 Code Quality Review

Date: 2026-03-14
Scope: `blueprints/search/pkg/hn2/*.go`

---

## Summary

The package is functional and well-structured in broad strokes, but has accumulated
several small inconsistencies against Go standard library conventions. This document
catalogs every issue found and describes the target state after enhancement.

---

## File-by-file Issues

### config.go

| # | Issue | Fix |
|---|-------|-----|
| C1 | `WithDefaults()` is called at the top of almost every method creating redundant allocations and obscuring which fields are canonical. | Make `WithDefaults` private (`withDefaults`); expose only `DefaultConfig()` and let callers pass a plain `Config`. |
| C2 | `EnsureDirs` only creates `DataDir` and `TodayDir`, not per-year subdirectories. But year dirs are created inline in `FetchMonth`. | Keep as-is; document that year dirs are created on demand. |
| C3 | `httpClient()` and `downloadHTTPClient()` are helper methods but are already private — good. `fqTable()`, `quoteIdent()` are also private — good. | No change needed. |
| C4 | Config zero-value is not useful (empty `RepoRoot`, no client). Callers always need `WithDefaults`. | Rename `WithDefaults` → `withDefaults` (private). Add `(c Config) resolved() Config` as the single internal resolver. All method receivers call `c.resolved()` at top. |

### client.go

| # | Issue | Fix |
|---|-------|-----|
| CL1 | `RemoteInfo` and `ListMonths` are public methods on `Config`. They belong there — good. But `MonthInfo` is used only internally by `task_historical`. | Move `MonthInfo` to `task_historical.go` or keep in `client.go` but mark it unexported: `monthInfo`. |
| CL2 | `parseIntAny` and `parseFloatAny` are package-level helpers scattered across two files (`client.go` and `analytics.go`). | Consolidate both into a single private file `parse.go`. |
| CL3 | `query()` reads up to 1 MiB (`1<<20`). Analytics queries return small payloads but `ListMonths` could theoretically return more. | Raise limit to `16<<20` with a comment explaining it. |
| CL4 | `downloadHTTPClient()` clones the base HTTP client but mutates a local copy — subtle and correct but not obvious. | Add comment: "clone to avoid mutating the shared client". |
| CL5 | `RemoteInfo` is exported; `RemoteInfo` (the type) is also exported. Name collision is confusing — method and type have the same name. | Rename the method to `remoteInfo` (private) since it is only called from `task_live.go` in the cold-start path. Callers inside the package use the private form. |

### analytics.go

| # | Issue | Fix |
|---|-------|-----|
| A1 | `QueryAnalytics` makes 4 serial HTTP round-trips. Each is independent. | Keep serial for now (ClickHouse public endpoint rate-limits); document why. |
| A2 | Inline anonymous structs for JSON decoding (`var r1 struct{ ... }`) are used in `QueryAnalytics`. This is idiomatic Go — fine. | No change. |
| A3 | `parseFloatAny` is defined in `analytics.go`. `parseIntAny` is defined in `client.go`. They belong together. | Move both to `parse.go`. |
| A4 | The ClickHouse query strings use string concatenation, making them hard to read. | Use raw string literals (`` ` ``) and join with spaces; already partially done. OK as-is. |

### fetch.go

| # | Issue | Fix |
|---|-------|-----|
| F1 | `streamToFile` is 60 lines. It mixes retry logic, HTTP, file I/O, and atomic rename into one function. | Extract `doHTTPDownload(ctx, req, dst *os.File) (int64, error)` so the retry loop only handles transport concerns. |
| F2 | `FetchMonth` calls `os.MkdirAll` before `streamToFile`. `FetchSince` does the same. Duplicated setup. | Extract `ensureParentDir(path string) error` (private). |
| F3 | `escapeSQLStr` is defined in `fetch.go` but used in `task_rollover.go`. | Move to a dedicated `sql.go` or `internal.go`. Since it is tiny, place in `fetch.go` is acceptable — document cross-file usage. |
| F4 | `sleepWithContext` is defined in `fetch.go` but is a generic utility. | Move to a `time.go` or keep in `fetch.go` with a comment. |
| F5 | `scanParquetResult` opens a new DuckDB connection every call. | Acceptable for now — DuckDB in-process connections are cheap. Document it. |
| F6 | On retry, the function does not log which attempt is in progress. | Add `fmt.Fprintf(os.Stderr, ...)` for attempt > 1 so operators can see retries. |

### stats.go

| # | Issue | Fix |
|---|-------|-----|
| S1 | `WriteStatsTodayCSV(path, rows, newRow)` always appends. `WriteStatsTodayCSVAll(path, rows)` writes a full slice. Having two functions that look similar is confusing — `WriteStatsTodayCSV` should be the only public one (accepting a variadic or a slice). | Merge: `WriteStatsTodayCSV(path string, rows []TodayRow)` writes the given slice (sorted). Callers append before calling. Drop `WriteStatsTodayCSVAll` as a separate name. |
| S2 | `writeStatsCSVExact` is private but duplicates the sort+write logic of `WriteStatsCSV`. | Replace `writeStatsCSVExact` by calling `writeStatsTodayCSVAll` (internal form). For stats.csv, inline the same write helper. |
| S3 | `writeCSVAtomic` uses `outPath + ".tmp"` as the temp path — same race hazard fixed in `fetch.go`. | Use `os.CreateTemp` in the same directory, then rename. |
| S4 | `CommittedMonthSet` is a free function — fine for Go. Naming is clear. | No change. |
| S5 | `ClearStatsTodayCSV` writes a header-only file. Internally it's `WriteStatsTodayCSVAll(path, nil)`. | Implement as one-liner calling the shared writer. |

### readme.go

| # | Issue | Fix |
|---|-------|-----|
| R1 | `buildGrowthChart`, `buildTypeTable`, etc. are package-level private functions — good Go style. | No change. |
| R2 | `fmtCount` (human-readable, e.g. "1.2M") and `fmtInt` (comma-formatted, e.g. "1,234") are both in the package but in different files (`readme.go` and `task_historical.go`). | Move both formatting helpers to a single `format.go`. |
| R3 | `GenerateREADME` accepts `[]MonthRow`, `[]TodayRow`, `*Analytics` — good named parameters. | No change. |
| R4 | `BuildReadmeData` is exported but is only used by `GenerateREADME` internally. | Make private: `buildReadmeData`. |

### task_historical.go

| # | Issue | Fix |
|---|-------|-----|
| T1 | `HFOp` is defined here but is also used by `task_live` and `task_rollover`. This couples files to an unrelated concept. | Move `HFOp` to `hf.go` (new file) alongside the `HFCommitFn` type alias. |
| T2 | `fmtInt` is defined here. Should be in `format.go` (see R2). | Move. |
| T3 | Phase strings (`"fetch"`, `"commit"`, `"skip"`) are stringly typed. If a caller misspells, there is no compile-time check. | Define `const Phase...` or a `Phase` type with named constants in each task file. |
| T4 | The inline `append(append([]MonthRow{}, existingRows...), newRow)` pattern is idiomatic but dense. | No change — it is clear enough. |
| T5 | `HistoricalTask`, `LiveTask`, `DayRolloverTask` all have a `cfg Config` + `opts *Opts` pattern. Consistent — good. | No change. |

### task_live.go

| # | Issue | Fix |
|---|-------|-----|
| L1 | Today backfill block is a 100-line anonymous `{ }` block inside `Run`. | Extract to `(t *LiveTask) backfillToday(ctx, cfg, today, interval, lastHighestID, todayRows) ([]TodayRow, int64, error)`. |
| L2 | Orphan rollover detection is another embedded block inside `Run`. | Extract to `(t *LiveTask) rolloverOrphans(ctx, cfg, today, todayRows) []TodayRow`. |
| L3 | The main poll loop is ~80 lines. | Extract to `(t *LiveTask) pollOnce(ctx, cfg, date, hhmm, lastID, interval) (result FetchResult, skip bool, err error)`. |
| L4 | `strings.ReplaceAll(blockHHMM, ":", "_")` appears 3+ times across task_live and task_rollover. | Move to a helper `blockFilename(date, hhmm string) string` (private). |
| L5 | `ReadStatsTodayCSV` + `WriteStatsTodayCSV` are called multiple times in close succession, sometimes within a few lines. This is a smell — stats are re-read when they could be passed in. | In the main loop, hold a local `todayRows` slice and update it in-memory, only writing to disk when ready to commit. |

### task_rollover.go

| # | Issue | Fix |
|---|-------|-----|
| RO1 | `DayRolloverTask` opens two separate DuckDB connections: one for merge, one for scan. | Reuse one connection for both. |
| RO2 | `buildParquetList` is a private helper — good. | No change. |
| RO3 | `fileExistsNE` ("file exists, non-empty") is a good private helper with a confusing name. | Rename to `fileExists(path string) bool`. The non-empty check is implicit from its usage. |
| RO4 | The rollover does not remove the local today files if the HF commit fails. This means a re-run would re-commit them. That is actually correct behaviour — safe to leave. | Document with a comment. |

---

## Cross-cutting Issues

| # | Issue | Fix |
|---|-------|-----|
| X1 | Phase strings repeated across tasks: `"fetch"`, `"commit"`, `"skip"`, `"wait"`, `"rollover"`. | Define typed constants per task, e.g. `HistoricalPhase`, `LivePhase`. |
| X2 | `opts.HFCommit` type `func(ctx, ops, msg) (string, error)` is inlined in every task. | Define `type CommitFn func(ctx context.Context, ops []HFOp, message string) (string, error)` in `hf.go`. |
| X3 | `WithDefaults()` is public but is a footgun — callers calling it twice silently create duplicated defaults. | Make private `resolved()`. Expose only `DefaultConfig()`. |
| X4 | No `_test.go` files exist. | Out of scope for this review (can be added separately). |
| X5 | `fmt.Fprintf(os.Stderr, "warn: ...")` is the error logging pattern — consistent and simple. Good. | No change. |

---

## Target File Layout After Enhancement

```
pkg/hn2/
  config.go          Config, DefaultConfig, resolved() [was WithDefaults]
  hf.go              HFOp, CommitFn type
  client.go          remoteInfo(), listMonths(), query(), newRequest(), downloadHTTPClient()
  parse.go           parseIntAny, parseFloatAny  [consolidated from client+analytics]
  format.go          fmtInt, fmtCount, blockFilename  [consolidated from task_historical+readme]
  fetch.go           FetchMonth, FetchSince, streamToFile, ensureParentDir, doHTTPDownload
  stats.go           MonthRow, TodayRow, Read/Write stats CSVs, writeCSVAtomic
  analytics.go       Analytics, NameCount, QueryAnalytics
  readme.go          ReadmeData, buildReadmeData, GenerateREADME, build* helpers
  task_historical.go HistoricalTask, HistoricalState, HistoricalMetric, HistoricalTaskOptions
  task_live.go       LiveTask, LiveState, LiveMetric, LiveTaskOptions + private helpers
  task_rollover.go   DayRolloverTask, RolloverState, RolloverMetric, RolloverTaskOptions
```

---

## Naming Conventions Adopted (from Go stdlib)

- Unexported identifiers for everything not needed by `cli/` callers.
- Constructor functions: `NewXxx(cfg Config, opts XxxOptions) *Xxx` — keep as-is.
- Options structs: `XxxOptions` — keep as-is.
- Method receivers: single-letter `t` for tasks, `c` for Config — keep as-is.
- Error wrapping: `fmt.Errorf("verb noun: %w", err)` — already consistent.
- No stuttering: `hn2.Config`, not `hn2.HN2Config`.
- Phase constants use string type alias for light safety without full enum overhead.

---

## Priority Order for Implementation

1. `format.go` — consolidate `fmtInt`, `fmtCount`, `blockFilename`
2. `parse.go` — consolidate `parseIntAny`, `parseFloatAny`
3. `hf.go` — extract `HFOp`, `CommitFn`
4. `config.go` — rename `WithDefaults` → `resolved` (private)
5. `stats.go` — merge `WriteStatsTodayCSV`/`WriteStatsTodayCSVAll`; fix `writeCSVAtomic` race
6. `fetch.go` — extract `ensureParentDir`, `doHTTPDownload`; log retries
7. `client.go` — make `MonthInfo` private; raise query limit; rename `RemoteInfo` method private
8. `readme.go` — make `BuildReadmeData` private
9. `task_historical.go` — remove `fmtInt` (moved); phase constants
10. `task_live.go` — extract `backfillToday`, `rolloverOrphans`; fix repeated ReplaceAll
11. `task_rollover.go` — single DuckDB conn; rename `fileExistsNE`
