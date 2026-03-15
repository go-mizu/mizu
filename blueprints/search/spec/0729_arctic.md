# 0729 — Arctic Shift Reddit Dump → HuggingFace

Publish the full Arctic Shift Reddit monthly dump to `open-index/arctic` on HuggingFace as
clean monthly parquet shards, organized so HF Data Studio recognises two typed datasets:
`comments` and `submissions`.

Key constraint: server disk is 200 GB total. We stream-process one month+type at a time,
chunking the JSONL stream into small parquet shards so neither a full JSONL file nor a full
parquet file ever accumulates on disk.

---

## Overview

- **Source**: Arctic Shift monthly torrent archives (all subreddits, all time)
  - 2005-06 → 2023-12: bundle torrent `9c263fc85366c1ef8f5bb9da0203f4c8c8db75f4` (selective file priority)
  - 2024-01 → present: individual monthly torrents (hashes hardcoded + fallback API lookup)
- **Command**: `search arctic publish`
- **HF Repo**: `open-index/arctic`
- **Local repo root**: `$HOME/data/arctic/repo` (override: `MIZU_ARCTIC_REPO_ROOT`)

Processing order: oldest-first, comments before submissions within each month.

---

## HuggingFace File Layout

```
open-index/arctic/
  data/
    comments/
      2005/
        06/
          000.parquet
        07/
          000.parquet
        ...
      2025/
        03/
          000.parquet
          001.parquet
          002.parquet
    submissions/
      2005/
        06/
          000.parquet
        ...
  stats.csv
  README.md
```

Recent months with large volumes produce multiple shards (`000.parquet`, `001.parquet`, …).
Old months (pre-2010) typically fit in one shard.

**Target shard size**: ~200 MB parquet (zstd). At ~8:1 JSON-to-parquet ratio, each shard
consumes ~1.6 GB of JSONL input. A large recent comments month (~30 GB JSONL) yields ~20 shards.

**File count estimate**: 240 months × 2 types × avg 5 shards = ~2,400 files — well under HF's
100k file limit.

### README.md frontmatter (DatasetDict config)

```yaml
---
configs:
- config_name: comments
  data_files:
  - split: train
    path: "data/comments/**/*.parquet"
- config_name: submissions
  data_files:
  - split: train
    path: "data/submissions/**/*.parquet"
---
```

This makes `load_dataset("open-index/arctic", "comments")` and
`load_dataset("open-index/arctic", "submissions")` work directly in HF Data Studio and via the
Python `datasets` library.

---

## Torrent Source Details

### File naming inside torrents

Arctic Shift uses the PushShift naming convention:

| File              | Contents                                   |
|-------------------|--------------------------------------------|
| `RC_YYYY-MM.zst`  | Reddit Comments, zstd-compressed JSONL     |
| `RS_YYYY-MM.zst`  | Reddit Submissions, zstd-compressed JSONL  |

Each line of the decompressed JSONL is one JSON object (one comment or one submission).

### Torrent map

**Bundle torrent (2005-06 → 2023-12)** — use selective file priority to download only the
target file (e.g. `RC_2005-06.zst`):
```
InfoHash: 9c263fc85366c1ef8f5bb9da0203f4c8c8db75f4
Files within: RC_YYYY-MM.zst, RS_YYYY-MM.zst for each month 2005-06..2023-12
```

**Individual monthly torrents (2024-01 → 2026-02)** — hardcoded in `torrent.go`:
```
2026-01: 8412b89151101d88c915334c45d9c223169a1a60
2026-02: c5ba00048236b60f819dbf010e9034d24fc291fb
# 2024-01 through 2025-12: see monthlyHashes map in torrent.go
```

For months not in the hardcoded map, fall back to querying Academic Torrents search API.
If no torrent peers found within 60 s, fall back to direct HTTP download from Arctic Shift's
filen.io mirror.

---

## stats.csv Schema

Tracks every committed (type, year-month) pair. Written atomically via temp+rename.
Sorted by `(year, month, type)`. Upsert semantics on re-run.

```
year,month,type,shards,count,size_bytes,dur_download_s,dur_process_s,dur_commit_s,committed_at
2005,6,comments,1,1234567,98765432,42.1,18.3,6.7,2029-07-29T12:00:00Z
2005,6,submissions,1,234567,12345678,11.2,4.1,3.2,2029-07-29T12:05:00Z
2025,3,comments,18,189000000,3720000000,3610.2,820.5,91.1,2029-07-29T14:00:00Z
```

Fields:
- `shards`: number of parquet files committed for this (month, type)
- `count`: total row count across all shards
- `size_bytes`: total parquet bytes across all shards
- `dur_download_s`: seconds to download .zst from torrent (or HTTP)
- `dur_process_s`: seconds for all chunks: decompress + DuckDB + parquet export
- `dur_commit_s`: seconds for HF commit
- `committed_at`: RFC3339 UTC

A committed set is derived from existing rows — resume simply skips already-committed pairs.

---

## Disk Space Management

**Before starting each (month, type)**, check free bytes on the partition containing RawDir:

```
min_free_gb = 30  (configurable, flag: --min-free-gb)
```

If free < min_free_gb: log warning and stop (do not skip and continue — a full disk means
the pipeline is stuck and operator attention is required).

**Peak disk usage per (month, type)**:
- Active `.zst` download: up to ~50 GB for a large recent month
- JSONL chunk: 1.6 GB at a time (deleted after shard is written)
- Shard parquets accumulate locally until HF commit: N × 200 MB (deleted after commit)
- DuckDB: used per-chunk, deleted per-chunk

Worst case for a recent comments month: ~50 GB (.zst) + ~1.6 GB (chunk) + ~4 GB (shards in
flight before commit) ≈ 56 GB. The min_free_gb=30 check triggers before .zst download, so
effective minimum needed on disk is ~56 GB free before we start a large month.

**Cleanup order** (strictly enforced):
1. After each JSONL chunk → shard written: delete chunk temp file
2. After .zst fully consumed: delete .zst file
3. After HF commit confirmed: delete all local shard parquets

No intermediate files are left on disk between runs. On crash, leftover .zst and chunk files
are deleted at startup before resuming.

---

## Streaming Pipeline (per month, per type)

Instead of materializing a full JSONL file, we stream the .zst through a zstd decoder and
read it in chunks of `chunk_lines` lines (default 2,000,000) or `chunk_bytes` raw JSON bytes
(default 1.6 GB), whichever is hit first.

```
month YYYY-MM, type ∈ {comments, submissions}

 ┌─ check free disk ─────────────────────────────────────────┐
 │  < min_free_gb  → log warning, stop                       │
 └───────────────────────────────────────────────────────────┘
            │
            ▼
 ┌─ already committed? ──────────────────────────────────────┐
 │  yes (in stats.csv) → skip                               │
 └───────────────────────────────────────────────────────────┘
            │
            ▼
 ┌─ cleanup leftover work files ─────────────────────────────┐
 │  delete any stale .zst, chunk, .db, .parquet in WorkDir   │
 └───────────────────────────────────────────────────────────┘
            │
            ▼
 ┌─ open torrent stream ─────────────────────────────────────┐
 │  connect to torrent, set selective priority for target    │
 │  file (RC_YYYY-MM.zst or RS_YYYY-MM.zst)                 │
 │  anacrolix/torrent: t.Files()[i].NewReader() → io.Reader  │
 │  wrap with klauspost/compress/zstd.NewReader()            │
 │  (2 GB decode window for Reddit compat)                   │
 │  60s peer discovery timeout → fallback HTTP download      │
 └───────────────────────────────────────────────────────────┘
            │
            ▼
 ┌─ chunk loop ──────────────────────────────────────────────┐
 │  read up to chunk_lines JSON lines from zstd stream       │
 │  → write to temp chunk file (WorkDir/chunk_NNN.jsonl)     │
 │  → DuckDB: read_json_auto(chunk), SELECT schema cols,     │
 │    add derived cols, COPY TO shard NNN.parquet (ZSTD)     │
 │  → delete chunk_NNN.jsonl                                 │
 │  → append shard to local shard list                       │
 │  repeat until EOF                                         │
 └───────────────────────────────────────────────────────────┘
            │ (all shards written, .zst stream exhausted)
            ▼
 ┌─ delete .zst / close torrent client ──────────────────────┐
 │  (torrent streamed in place — no .zst file on disk if     │
 │   using streaming reader; if fallback HTTP: delete file)  │
 └───────────────────────────────────────────────────────────┘
            │
            ▼
 ┌─ HF commit ───────────────────────────────────────────────┐
 │  upload: data/{type}/YYYY/MM/000.parquet … NNN.parquet    │
 │  upload: stats.csv (upserted with this month's row)       │
 │  upload: README.md (regenerated from all stats rows)      │
 │  batch: ≤50 ops per HF commit call                        │
 └───────────────────────────────────────────────────────────┘
            │
            ▼
 ┌─ delete local shards ─────────────────────────────────────┐
 │  delete WorkDir/YYYY-MM/{type}/000.parquet … NNN.parquet  │
 └───────────────────────────────────────────────────────────┘
```

### Torrent streaming vs file download

Prefer streaming (no .zst on disk): use `anacrolix/torrent`'s sequential reader with
read-ahead so pieces arrive in order. The reader blocks until each piece is available,
which is acceptable since we process sequentially.

For the HTTP fallback (filen.io direct download), we download to a temp `.zst` file first,
then stream-decompress from that file, then delete it.

---

## Parquet Schemas

All columns are explicitly selected via DuckDB `SELECT … FROM read_json_auto(…)` to drop
unknown fields from the JSONL.

### Comments (`data/comments/YYYY/MM/NNN.parquet`)

| Column            | Type      | Source field        |
|-------------------|-----------|---------------------|
| id                | VARCHAR   | id                  |
| author            | VARCHAR   | author              |
| subreddit         | VARCHAR   | subreddit           |
| body              | VARCHAR   | body                |
| score             | BIGINT    | score               |
| created_utc       | BIGINT    | created_utc         |
| created_at        | TIMESTAMP | epoch_ms(created_utc * 1000) |
| body_length       | BIGINT    | LENGTH(body)        |
| link_id           | VARCHAR   | link_id             |
| parent_id         | VARCHAR   | parent_id           |
| distinguished     | VARCHAR   | distinguished       |
| author_flair_text | VARCHAR   | author_flair_text   |

### Submissions (`data/submissions/YYYY/MM/NNN.parquet`)

| Column             | Type      | Source field             |
|--------------------|-----------|--------------------------|
| id                 | VARCHAR   | id                       |
| author             | VARCHAR   | author                   |
| subreddit          | VARCHAR   | subreddit                |
| title              | VARCHAR   | title                    |
| selftext           | VARCHAR   | selftext                 |
| score              | BIGINT    | score                    |
| created_utc        | BIGINT    | created_utc              |
| created_at         | TIMESTAMP | epoch_ms(created_utc * 1000) |
| title_length       | BIGINT    | LENGTH(title)            |
| num_comments       | BIGINT    | num_comments             |
| url                | VARCHAR   | url                      |
| over_18            | BOOLEAN   | over_18                  |
| link_flair_text    | VARCHAR   | link_flair_text          |
| author_flair_text  | VARCHAR   | author_flair_text        |

DuckDB is invoked with `ignore_errors=true` on the JSON reader. The DuckDB `.db` file is
in-memory (`:memory:`) per chunk — no persistent DB file needed.

---

## pkg/arctic Package

```
pkg/arctic/
  config.go         — Config struct, env overrides, all path helpers, EnsureDirs
  torrent.go        — torrent stream reader + HTTP fallback for one month/type
  process.go        — chunk loop: JSONL chunk → DuckDB → shard parquet
  stats.go          — stats.csv read/write (atomic rewrite, upsert)
  hf.go             — HFOp + CommitFn (same pattern as pkg/hn2/hf.go)
  readme.go         — README.md template generation
  task_publish.go   — outer month×type loop, disk checks, orchestration
```

### config.go

```go
type Config struct {
    RepoRoot  string  // $HOME/data/arctic/repo   (MIZU_ARCTIC_REPO_ROOT)
    HFRepo    string  // "open-index/arctic"
    RawDir    string  // $HOME/data/arctic/raw    (MIZU_ARCTIC_RAW_DIR)
    WorkDir   string  // $HOME/data/arctic/work   (MIZU_ARCTIC_WORK_DIR)
    MinFreeGB int     // 30                        (MIZU_ARCTIC_MIN_FREE_GB)
    ChunkLines int    // 2_000_000 lines per chunk (MIZU_ARCTIC_CHUNK_LINES)
}

func DefaultConfig() Config
func (c Config) StatsCSVPath() string                           // RepoRoot/stats.csv
func (c Config) READMEPath() string                             // RepoRoot/README.md
func (c Config) ShardHFPath(typ, year, month string, n int) string  // data/comments/2025/03/000.parquet
func (c Config) ShardLocalPath(typ, year, month string, n int) string // WorkDir/comments/2025/03/000.parquet
func (c Config) ChunkPath(n int) string                         // WorkDir/chunk_000.jsonl
func (c Config) EnsureDirs() error
```

### torrent.go

```go
// MonthStream opens a streaming io.Reader for the target .zst file.
// For months 2005-06..2023-12: connects to bundle torrent, sets selective priority.
// For months 2024-01+: connects to individual monthly torrent.
// Falls back to HTTP download (writes temp .zst, returns reader over it + cleanup func).
// Returns: zstd-decoded io.Reader, a cleanup func (closes torrent client / deletes temp file),
// and download duration.
func MonthStream(ctx context.Context, year, month int, typ string,
    cb DownloadProgressCallback) (r io.Reader, cleanup func(), durDownload time.Duration, err error)

type DownloadProgress struct {
    Phase      string   // "metadata" | "peers" | "downloading" | "done"
    BytesDone  int64
    BytesTotal int64
    SpeedBps   float64
    Peers      int
}
type DownloadProgressCallback func(DownloadProgress)

// monthlyHashes: embedded map for 2024-01..2026-02
// bundleInfoHash: "9c263fc85366c1ef8f5bb9da0203f4c8c8db75f4" for 2005-06..2023-12
```

### process.go

```go
type ShardResult struct {
    Index     int
    Rows      int64
    SizeBytes int64
}

type ProcessResult struct {
    Shards    []ShardResult
    TotalRows int64
    TotalSize int64
    Duration  time.Duration
}

// ProcessStream reads JSONL from r in chunks of cfg.ChunkLines lines,
// writes each chunk to a temp file, imports via DuckDB (:memory:),
// exports to shard parquet (ZSTD), deletes the temp chunk file,
// and appends the shard path to the result.
// typ is "comments" or "submissions".
func ProcessStream(ctx context.Context, cfg Config, r io.Reader,
    typ, year, month string, cb func(shard ShardResult)) (ProcessResult, error)
```

### stats.go

```go
type StatsRow struct {
    Year         int
    Month        int
    Type         string
    Shards       int
    Count        int64
    SizeBytes    int64
    DurDownloadS float64
    DurProcessS  float64
    DurCommitS   float64
    CommittedAt  time.Time
}

func ReadStatsCSV(path string) ([]StatsRow, error)
func WriteStatsCSV(path string, rows []StatsRow) error   // upsert + atomic rewrite, sort by (year,month,type)
func CommittedSet(rows []StatsRow) map[string]bool       // key: "2005-06/comments"
```

### hf.go

```go
// Identical to pkg/hn2/hf.go.
type HFOp struct {
    LocalPath  string
    PathInRepo string
    Delete     bool
}
type CommitFn func(ctx context.Context, ops []HFOp, msg string) (commitURL string, err error)
```

### task_publish.go

```go
type PublishOptions struct {
    From     time.Month   // first year-month (inclusive)
    FromYear int
    To       time.Month   // last year-month (inclusive)
    ToYear   int
    HFCommit CommitFn
}

type PublishTask struct{ cfg Config; opts PublishOptions }

func NewPublishTask(cfg Config, opts PublishOptions) *PublishTask
// Run iterates (month, type) pairs oldest-first.
// For each pair: disk check → skip-if-done → MonthStream → ProcessStream →
//   HF commit → delete shards → upsert stats.csv.
func (t *PublishTask) Run(ctx context.Context) error
```

---

## CLI Command

File: `cmd/search/arctic_publish.go`

```
search arctic publish [flags]

Flags:
  --repo-root string     local HF repo root (default $HOME/data/arctic/repo)
  --repo string          HuggingFace repo ID (default "open-index/arctic")
  --from string          start month YYYY-MM (default "2005-06")
  --to string            end month YYYY-MM (default current month)
  --min-free-gb int      minimum free disk GB (default 30)
  --chunk-lines int      JSONL lines per parquet shard (default 2000000)
  --private              mark HF repo private

Env:
  HF_TOKEN              required
```

Progress output (lipgloss-styled, matching hn_publish.go style):
```
[2005-06] comments   ↓ 42.1s  ⚙ 18.3s  ↑ 6.7s   1 shard   1.2M rows   93 MB
[2005-06] submissions ↓ 11.2s  ⚙ 4.1s   ↑ 3.2s   1 shard   234k rows   12 MB
[2005-07] comments   skip (committed)
[2025-03] comments   ↓ 3610s  ⚙ 820s   ↑ 91s   18 shards  189M rows  3.7 GB
```

---

## README Template

Embedded Go template in `readme.go`, rendered with data from `stats.csv`.

```markdown
---
[DatasetDict config frontmatter]
---

# Arctic Shift Reddit Archive

Full Reddit dataset sourced from [Arctic Shift](https://github.com/ArthurHeitmann/arctic_shift)
monthly dumps. Comments and submissions from all subreddits, 2005-06 through {LatestMonth}.

## Usage

```python
from datasets import load_dataset

# Load all comments (streaming recommended for full dataset)
comments = load_dataset("open-index/arctic", "comments", streaming=True)

# Load all submissions
submissions = load_dataset("open-index/arctic", "submissions", streaming=True)
```

## Dataset Stats

| Type        | Months | Rows       | Size    |
|-------------|--------|------------|---------|
| comments    | {N}    | {N}B       | {N} GB  |
| submissions | {N}    | {N}B       | {N} GB  |

## Growth (rows per year)

{bar chart from stats.csv grouped by year}

## Schema

### Comments

| Column | Type | Description |
...

### Submissions

| Column | Type | Description |
...

## Source & License

Data sourced from [Arctic Shift](https://github.com/ArthurHeitmann/arctic_shift) project,
which re-packages PushShift Reddit archives. Original content by Reddit users.
```

---

## Deployment

### Initial setup on server 2

```bash
# build natively on server (fast, no QEMU)
make build-on-server

# or: cross-compile locally and push
make build-linux-noble && make deploy-linux-noble
```

### Screen session

```bash
# SSH to server 2
screen -S arctic
export HF_TOKEN=hf_...
search arctic publish \
  --from 2005-06 \
  --repo open-index/arctic \
  --min-free-gb 30

# detach:   Ctrl+A D
# reattach: screen -r arctic
```

Fully resumable: reads `stats.csv` on startup, skips committed pairs. On crash or disk-full,
fix the condition and re-run the same command.

### Startup cleanup

On each startup, `task_publish.go` removes any leftover work files from a previous interrupted
run:
- `WorkDir/chunk_*.jsonl`
- `WorkDir/comments/**/*.parquet` and `WorkDir/submissions/**/*.parquet`
- Any `.zst` files in `RawDir` (only present if HTTP fallback was used)

This prevents stale data from causing incorrect shard counts.

---

## Torrent Monthly Hash Map

Embedded in `torrent.go`. Keys: `"YYYY-MM"`. For the bundle (pre-2024) there is one shared
infohash; for 2024+ each month may have a separate torrent. The map holds what is known at
build time; unknown future months fall back to Academic Torrents API search.

```go
const bundleInfoHash = "9c263fc85366c1ef8f5bb9da0203f4c8c8db75f4"  // 2005-06..2023-12

// monthlyInfoHashes maps "YYYY-MM" → infohash for individual monthly torrents (2024+)
var monthlyInfoHashes = map[string]string{
    "2026-01": "8412b89151101d88c915334c45d9c223169a1a60",
    "2026-02": "c5ba00048236b60f819dbf010e9034d24fc291fb",
    // 2024-01..2025-12: fill in from download_links.md before shipping
}
```

---

## Month/Type Processing Order

```
2005-06 comments
2005-06 submissions
2005-07 comments
2005-07 submissions
...
2026-02 comments
2026-02 submissions
```

Oldest-first means HF dataset is usable from the start. Each (month, type) is an independent
atomic unit: either fully committed (in stats.csv) or not started.
