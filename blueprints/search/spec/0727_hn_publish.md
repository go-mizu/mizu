# Spec 0727: HN Publish — Stream Hacker News to Hugging Face

**Date:** 2026-03-14
**Command:** `search hn publish`
**HF repo:** `open-index/hacker-news`
**Local root:** `$HOME/data/hn/repo`

---

## Overview

Continuously publish the full Hacker News dataset to Hugging Face as monthly parquet files, partitioned by year/month, with 5-minute live blocks for today. Historical months are backfilled once and skipped on resume. Live mode (`--live`) polls every 5 minutes and creates a new parquet block; at midnight it merges the day's blocks into the monthly parquet in a single atomic commit.

---

## File Layout

```
$HOME/data/hn/repo/
  README.md                          ← generated from embed template + both CSVs
  stats.csv                          ← one row per committed historical month
  stats_today.csv                    ← one row per committed 5-min live block (cleared at rollover)
  data/
    2006/
      2006-10.parquet                ← first HN month (discovered from remote)
      2006-11.parquet
      ...
    2007/ ... 2025/ ... 2026/
      2026-01.parquet
      2026-02.parquet
      # 2026-03.parquet created at day-rollover, NOT during historical backfill
  today/
    2026-03-14_00_00.parquet         ← 5-min blocks, live mode only
    2026-03-14_00_05.parquet
    ...
    2026-03-14_23_55.parquet
```

**Hugging Face repo path mirrors local layout:**
- `data/2006/2006-10.parquet`
- `today/2026-03-14_00_00.parquet`
- `stats.csv`, `stats_today.csv`, `README.md`

---

## stats.csv Schema

One row per committed historical month. After each new month is committed, the full file is rewritten in sorted order (year ASC, month ASC) and written atomically via `.tmp` rename.

```
year,month,lowest_id,highest_id,count,dur_fetch_s,dur_commit_s,size_bytes,committed_at
2006,10,1,45231,45231,42,8,1823400,2026-03-14T12:00:00Z
2006,11,45232,89012,44100,38,7,1901230,2026-03-14T12:02:11Z
```

Fields:
| Column | Type | Description |
|--------|------|-------------|
| `year` | int | Calendar year |
| `month` | int | Calendar month (1–12) |
| `lowest_id` | int64 | Minimum HN item ID in this month |
| `highest_id` | int64 | Maximum HN item ID in this month |
| `count` | int64 | Number of items in the parquet |
| `dur_fetch_s` | int | Seconds to fetch parquet from remote |
| `dur_commit_s` | int | Seconds to upload + HF commit |
| `size_bytes` | int64 | Parquet file size on disk |
| `committed_at` | RFC3339 | When this month was committed |

**Write strategy:** compute new row → load existing rows → append → sort by (year, month) → write to `stats.csv.tmp` → atomic rename to `stats.csv`. Never append in-place.

**At day rollover:** a new row is added to stats.csv for the completed day's merged parquet (representing the full month-to-date). stats_today.csv is reset to a header-only file in the same rollover commit.

---

## stats_today.csv Schema

One row per committed 5-min live block. Reset to header-only at each day rollover.

```
date,block,lowest_id,highest_id,count,dur_fetch_s,dur_commit_s,size_bytes,committed_at
2026-03-14,00:00,43123456,43124500,1044,2,3,45200,2026-03-14T00:01:22Z
2026-03-14,00:05,43124501,43125890,1389,1,3,51100,2026-03-14T00:06:44Z
```

Fields:
| Column | Type | Description |
|--------|------|-------------|
| `date` | YYYY-MM-DD | Calendar date of this block |
| `block` | HH:MM | 5-min aligned block start time (UTC) |
| `lowest_id` | int64 | Minimum HN item ID in this block |
| `highest_id` | int64 | Maximum HN item ID in this block |
| `count` | int64 | Number of items in this block |
| `dur_fetch_s` | int | Seconds to fetch from remote |
| `dur_commit_s` | int | Seconds to upload + HF commit |
| `size_bytes` | int64 | Parquet file size on disk |
| `committed_at` | RFC3339 | When this block was committed |

**Write strategy:** same atomic rewrite as stats.csv (sorted by date+block).

---

## README Template (Go Embed)

Stored at `cli/embed/hn_readme.md.tmpl`, embedded with `//go:embed`.

Template variables populated by aggregating **both** stats.csv and stats_today.csv on every commit, so the README always shows the complete current picture — historical months plus today's live progress:

```
// From stats.csv (historical):
{{.TotalHistoricalItems}}  — sum of count across all stats.csv rows
{{.TotalMonths}}           — number of committed months
{{.FirstMonth}}            — earliest year-month in stats.csv (e.g. "2006-10")
{{.LastMonth}}             — most recent committed month (e.g. "2026-02")
{{.HistoricalSizeBytes}}   — sum of size_bytes in stats.csv
{{.AvgFetchSec}}           — avg dur_fetch_s across months

// From stats_today.csv (live — zero-values when file absent or empty):
{{.TodayDate}}             — date of live blocks (e.g. "2026-03-14")
{{.TodayBlocks}}           — number of committed blocks today
{{.TodayItems}}            — sum of count across stats_today.csv rows
{{.TodayLastBlock}}        — last committed block time (e.g. "23:45")
{{.TodaySizeBytes}}        — sum of size_bytes in stats_today.csv

// Combined (historical + today):
{{.TotalItems}}            — TotalHistoricalItems + TodayItems
{{.TotalSizeBytes}}        — HistoricalSizeBytes + TodaySizeBytes
{{.LastUpdated}}           — max(committed_at) across both files
```

The README includes:
- HF YAML front matter (license, tags, size_categories, configs pointing to `data/*/*.parquet` and `today/*.parquet`)
- Dataset description with dynamic **combined** total item count and date range
- Stats summary: historical months table (year, items, size) + live-today row (always rendered, shows zeros when no live data)
- Python / DuckDB / huggingface_hub usage examples
- Full schema table (all HN item columns)

---

## pkg/hn2 Package

New package. **No "ClickHouse" in any type, variable, or function name** — one package-level comment explains the backing source.

### Files

| File | Responsibility |
|------|---------------|
| `config.go` | Config struct, WithDefaults(), env overrides, dir helpers |
| `client.go` | RemoteInfo, HTTP query helpers, month list query |
| `fetch.go` | FetchMonth(), FetchSince() — stream to parquet file |
| `task_historical.go` | HistoricalTask[HistoricalState, HistoricalMetric] |
| `task_live.go` | LiveTask[LiveState, LiveMetric] |
| `task_rollover.go` | DayRolloverTask[RolloverState, RolloverMetric] |
| `stats.go` | Read/write stats.csv and stats_today.csv (atomic rewrite) |
| `readme.go` | GenerateREADME(statsPath, todayPath) from embed template |

### Config

```go
// Package hn2 publishes the Hacker News dataset to Hugging Face.
// Data is fetched from the ClickHouse public SQL playground (sql.clickhouse.com).
package hn2

type Config struct {
    RepoRoot    string // $HOME/data/hn/repo
    EndpointURL string // remote SQL HTTP endpoint
    Database    string // remote database name
    Table       string // remote table name
    User        string // remote user (unauthenticated public)
    DNSServer   string // optional custom DNS resolver
    HTTPClient  *http.Client
}
```

Env overrides: `MIZU_HN2_ENDPOINT`, `MIZU_HN2_DATABASE`, `MIZU_HN2_TABLE`, `MIZU_HN2_USER`, `MIZU_HN2_DNS_SERVER`, `MIZU_HN2_REPO_ROOT`.

### RemoteInfo

```go
type RemoteInfo struct {
    Count     int64
    MaxID     int64
    MaxTime   string
    CheckedAt time.Time
}
```

Query: `SELECT toInt64(count()) AS c, toInt64(max(id)) AS max_id, toString(max(time)) AS max_time FROM <table> FORMAT JSONEachRow`

### FetchResult

Returned by both FetchMonth and FetchSince:

```go
type FetchResult struct {
    LowestID  int64
    HighestID int64
    Count     int64
    Bytes     int64         // bytes written to disk
    Duration  time.Duration
}
```

If the remote returns zero rows (empty parquet), Count == 0, LowestID == 0, HighestID == 0 — the caller must handle the no-data case (skip writing, do not commit).

### FetchMonth

```go
func (c Config) FetchMonth(ctx context.Context, year, month int, outPath string) (FetchResult, error)
```

Query: `SELECT * FROM <table> WHERE time >= toDateTime('<YYYY-MM-01 00:00:00>') AND time < toDateTime('<YYYY-MM+1-01 00:00:00>') ORDER BY id FORMAT Parquet`

Streams response body directly to `outPath + ".tmp"`, renames atomically on success, removes `.tmp` on failure. Returns FetchResult populated from a quick DuckDB COUNT + MIN/MAX scan of the written file (avoids parsing the parquet stream mid-flight).

Retry: up to 4 attempts with 1s/2s/4s backoff for 5xx / network errors.

### FetchSince

```go
func (c Config) FetchSince(ctx context.Context, afterID int64, ceilTime time.Time, outPath string) (FetchResult, error)
```

Query: `SELECT * FROM <table> WHERE id > <afterID> AND time < toDateTime('<ceilTime>') ORDER BY id FORMAT Parquet`

`ceilTime` is the snapshot time passed in by the caller (typically `time.Now().UTC()` at the start of the poll tick). Bounding by `ceilTime` prevents items that arrive at the remote after the poll started from leaking into the block, and ensures items don't straddle midnight into the next day's block.

Same retry and atomic-write pattern as FetchMonth.

### Month List Query

```go
func (c Config) ListMonths(ctx context.Context) ([]MonthInfo, error)
```

```go
type MonthInfo struct {
    Year  int
    Month int
    Count int64
}
```

Query: `SELECT toYear(time) AS y, toMonth(time) AS m, toInt64(count()) AS n FROM <table> WHERE time IS NOT NULL GROUP BY y, m ORDER BY y, m FORMAT JSONEachRow`

**Current month is excluded:** the caller (HistoricalTask) filters out the month that matches the current calendar month, so the in-progress month is never committed as a historical parquet.

---

### Tasks

#### HistoricalTask

```go
type HistoricalState struct {
    Phase        string        // "fetch" | "commit" | "skip"
    Month        string        // "2006-10"
    MonthIndex   int
    MonthTotal   int
    Rows         int64
    BytesDone    int64
    ElapsedTotal time.Duration
    SpeedBytesPS float64
}

type HistoricalMetric struct {
    MonthsWritten int
    MonthsSkipped int
    RowsWritten   int64
    BytesWritten  int64
    Elapsed       time.Duration
}
```

Run logic:
1. Read stats.csv → build set of already-committed `(year, month)` pairs
2. Call `ListMonths(ctx)` → exclude the current calendar month (prevent partial-month parquet)
3. Apply `--from` filter: skip any month before the user-supplied start month
4. For each `(year, month)` not in committed set:
   - Emit `Phase="fetch"`
   - `FetchMonth(ctx, year, month, "data/YYYY/YYYY-MM.parquet")`
   - If `Count == 0`: skip (month has no data yet), remove any `.tmp` file — emit `Phase="skip"`
   - Emit `Phase="commit"`
   - HF commit: ADD `data/YYYY/YYYY-MM.parquet` + rewritten `stats.csv` + `README.md`
   - Append row to stats.csv (rewrite sorted)
5. Emit final state on completion

**Concurrency safety with live mode:** HistoricalTask only writes `data/YYYY/YYYY-MM.parquet` for *completed past months* (never the current month). LiveTask only writes `today/YYYY-MM-DD_HH_MM.parquet` and the rollover merges *only* today's blocks. The two processes never touch the same parquet file. `stats.csv` is written by HistoricalTask only; `stats_today.csv` by LiveTask only. The rollover (run by LiveTask) writes to `stats.csv` exactly once per day at midnight — after historical has already moved on to the next month. No file locking is needed given this invariant. **Recommended deployment:** start live mode only after historical reaches the current month (or use `--from` to limit historical scope).

#### LiveTask

```go
type LiveState struct {
    Phase          string        // "fetch" | "commit" | "wait" | "rollover"
    Block          string        // "2026-03-14 00:05"
    NewItems       int64
    HighestID      int64
    NextFetchIn    time.Duration
    BlocksToday    int
    TotalCommitted int64
}

type LiveMetric struct {
    BlocksWritten int
    RowsWritten   int64
    Rollovers     int
    Elapsed       time.Duration
}
```

**Cold-start watermark resolution** (in priority order):

`lastDate` is **always** initialized to `time.Now().UTC()` truncated to the calendar date, regardless of which case applies below — this prevents a spurious rollover trigger on restart.

`lastHighestID`:
1. If stats_today.csv has rows whose `date` == today → use `max(highest_id)` from those rows
2. Else if stats.csv has rows → use `max(highest_id)` across all stats.csv rows
3. Else → call `RemoteInfo(ctx)` and use `MaxID` (first-ever run, no history committed yet)

If `--from` is set and stats.csv has no rows before that month, the watermark falls through to case 3 — the live session starts from the current remote head, which is correct.

Run logic:
1. Resolve `lastHighestID` and `lastDate` via cold-start watermark above
2. Compute current block time: truncate `time.Now().UTC()` to 5-min boundary
3. Loop:
   a. Emit `Phase="fetch"`
   b. `FetchSince(ctx, lastHighestID, time.Now().UTC(), "today/YYYY-MM-DD_HH_MM.parquet")`
   c. If `Count == 0`: remove any `.tmp` file, skip commit, sleep to next boundary, continue
   d. Update `lastHighestID = result.HighestID`
   e. Append row to stats_today.csv (rewrite sorted)
   f. Emit `Phase="commit"`
   g. HF commit: ADD `today/YYYY-MM-DD_HH_MM.parquet` + rewritten `stats_today.csv` + `README.md`
   h. If `time.Now().UTC().Date() != lastDate` → emit `Phase="rollover"` → run `DayRolloverTask`
   i. Sleep until next 5-min boundary

#### DayRolloverTask

```go
type RolloverState struct {
    Phase      string // "merge" | "commit"
    PrevDate   string // "2026-03-13"
    FilesFound int
    RowsMerged int64
}

type RolloverMetric struct {
    PrevDate    string
    MonthPath   string
    RowsMerged  int64
    FilesPruned int
    CommitURL   string
}
```

Run logic:
1. Collect all local `today/PREV_DATE_*.parquet` files (sorted). If none found (already removed — re-entrant restart after successful commit), check whether `monthPath` was already written; if so, skip merge and proceed to step 4 using the existing file.
2. Determine `monthPath = data/YYYY/YYYY-MM.parquet` for prev date's year+month
3. **Merge sources:** if `monthPath` already exists on disk (written by a prior historical backfill or partial rollover), merge the existing monthly parquet AND today's files together; otherwise merge only today's files:
   ```sql
   COPY (
     SELECT * FROM read_parquet(['existing_monthly_if_any', 'today/...', ...])
     ORDER BY id
   ) TO 'data/YYYY/YYYY-MM.parquet.tmp'
   (FORMAT PARQUET, COMPRESSION zstd, COMPRESSION_LEVEL 22)
   ```
   Atomic rename `.tmp → monthPath` on success, remove `.tmp` on failure.
4. Scan merged file for lowest_id, highest_id, count (DuckDB MIN/MAX/COUNT)
5. **Upsert row in stats.csv:** if a row with `(year, month)` already exists (written by HistoricalTask), replace it with the new merged row; otherwise append. Then rewrite the full sorted file atomically. Reset stats_today.csv to header-only (write file with only the header line).
6. Generate README from both updated CSVs
7. Single HF commit:
   - DELETE: each `today/PREV_DATE_HH_MM.parquet` path (via `hfOperation{Delete: true}`)
   - ADD: `data/YYYY/YYYY-MM.parquet`
   - ADD: `stats.csv`
   - ADD: `stats_today.csv` (header-only)
   - ADD: `README.md`
   - Message: `"Merge 2026-03-13 → data/2026/2026-03.parquet (N items)"`
8. **After confirmed successful HF commit:** delete all local `today/PREV_DATE_*.parquet` files from disk. If deletion of a local file fails, log a warning and continue — the files are inert since stats_today.csv no longer references them and the HF repo no longer contains them.

---

## HF Commit — Delete Support

The existing `hfOperation` struct in `cli/cc_publish_hf.go` is add-only. For rollover, it must be extended with a `Delete` flag:

```go
type hfOperation struct {
    LocalPath  string // empty for delete operations
    PathInRepo string
    Delete     bool   // if true, creates CommitOperationDelete
}
```

In `hfCommitPayload` passed to `hf_commit.py`:
```json
{
  "ops": [
    {"path_in_repo": "today/2026-03-13_23_55.parquet", "delete": true},
    {"local_path": "/abs/path/data/2026/2026-03.parquet", "path_in_repo": "data/2026/2026-03.parquet"}
  ]
}
```

**`hf_commit.py` must be updated** (currently only imports/uses `CommitOperationAdd`). Required changes:
1. Add `CommitOperationDelete` to the import from `huggingface_hub`
2. In the ops loop, dispatch on `op.get("delete", False)`:
   - `True` → `CommitOperationDelete(path_in_repo=op["path_in_repo"])`
   - `False` → `CommitOperationAdd(path_or_fileobj=op["local_path"], path_in_repo=op["path_in_repo"])` (existing behaviour)

The Go `opJSON` struct in `createCommitPython` must also include the `Delete bool` field so it serializes correctly.

---

## CLI Command

### `cli/hn_publish.go`

```go
func newHNPublish() *cobra.Command
```

Flags:
| Flag | Default | Description |
|------|---------|-------------|
| `--repo` | `open-index/hacker-news` | HF dataset repo ID |
| `--repo-root` | `$HOME/data/hn/repo` | Local root directory |
| `--live` | false | Enable continuous 5-min polling after backfill |
| `--interval` | `5m` | Live poll interval (minimum 1m) |
| `--from` | `""` | Start month YYYY-MM (skip older months in historical) |
| `--private` | false | Create HF repo as private |

The command:
1. Ensures `HF_TOKEN` is set
2. Creates HF repo if missing
3. Runs HistoricalTask with Lipgloss progress rendering (month-by-month table)
4. If `--live`: runs LiveTask (looping, prints block-level dashboard)

`--from` affects HistoricalTask only. LiveTask always uses the cold-start watermark (stats-based or remote head), independent of `--from`.

---

## HN Parquet Schema

The parquet files contain the native HN item schema as exported from the remote source. No transformation — the remote exports columns as-is. All numeric IDs are `int64` to match the remote source type and avoid truncation as IDs grow.

| Column | Type | Description |
|--------|------|-------------|
| `id` | int64 | Item ID |
| `deleted` | bool | Soft-deleted flag |
| `type` | string | story, comment, ask, show, job, poll, pollopt |
| `by` | string | Username of author |
| `time` | DateTime | Post timestamp (UTC) |
| `text` | string | HTML body (comments, Ask HN text) |
| `dead` | bool | Flagged/killed by moderators |
| `parent` | int64 | Parent item ID (comments) |
| `poll` | int64 | Poll item ID (pollopts) |
| `kids` | Array(int64) | Child item IDs |
| `url` | string | External URL (stories) |
| `score` | int64 | Points |
| `title` | string | Story/Ask/Show title |
| `parts` | Array(int64) | Poll option IDs |
| `descendants` | int64 | Total comment count |

*(The actual column types in the parquet are determined by the remote source's FORMAT Parquet export; the table above is the expected schema. No Go-side schema struct is defined — files are streamed directly to disk.)*

---

## Deployment

Two screen sessions on server 2:

```bash
# Historical backfill (runs once, exits when complete):
screen -S hn-history
HF_TOKEN=hf_... search hn publish

# Live mode (permanent):
screen -S hn-live
HF_TOKEN=hf_... search hn publish --live
```

**Safe to run concurrently** with the following invariant:
- HistoricalTask writes `data/YYYY/YYYY-MM.parquet` for **completed past months only** (current month excluded)
- LiveTask writes `today/YYYY-MM-DD_HH_MM.parquet` for the current day only
- HistoricalTask owns `stats.csv`; LiveTask owns `stats_today.csv`
- DayRolloverTask (run by LiveTask at midnight) writes to `stats.csv` once per day — this is the only point where LiveTask touches stats.csv, and it does so at midnight when HistoricalTask is processing a different (earlier) month

No file locking needed given these invariants. If historical and rollover happen to both write stats.csv within the same second on the last historical month, the result is two sorted rewrites of the same file — idempotent and safe.

---

## Commit Strategy Summary

| Event | Files in commit | Message pattern |
|-------|----------------|-----------------|
| Historical month | ADD `data/YYYY/YYYY-MM.parquet` + `stats.csv` + `README.md` | `Add 2006-10 (45231 items)` |
| Live block | ADD `today/YYYY-MM-DD_HH_MM.parquet` + `stats_today.csv` + `README.md` | `Live 2026-03-14 00:05 (+1044 items)` |
| Day rollover | DELETE `today/PREV_DATE_*.parquet` + ADD `data/YYYY/YYYY-MM.parquet` + `stats.csv` + `stats_today.csv` (header-only) + `README.md` | `Merge 2026-03-13 → data/2026/2026-03.parquet (N items)` |

---

## Non-Goals

- No deduplication across months (source is authoritative)
- No Go-side parquet schema struct — files streamed directly from remote
- No chart generation (unlike cc publish) — README stats table is sufficient
- If HF commit fails, local parquet stays on disk; stats CSV not updated until commit succeeds (safe retry on next run)
