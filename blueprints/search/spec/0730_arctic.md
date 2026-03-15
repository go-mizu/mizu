# 0730 — Arctic Shift Reddit Dump → HuggingFace (`pkg/arctic`)

Publish the full Arctic Shift Reddit monthly dump to `open-index/arctic` on HuggingFace as
clean monthly parquet shards, organized so HF Data Studio recognises two typed datasets
(`comments` and `submissions`). Command: `search arctic publish`.

Key constraint: server disk is 200 GB total. We download one `.zst`, process it into shards,
commit, then delete everything before touching the next (month, type).

---

## Goal & Non-Goals

**Goal**: Complete, resumable pipeline that publishes all Reddit comments + submissions
(2005-12 → present) to HuggingFace as analysis-ready parquet, month by month.

**Non-goals**:
- Per-subreddit splits (too many files, hit HF 100k limit)
- Real-time / live ingestion (batch historical archive only)
- Deduplication across months (source data is already deduplicated by Arctic Shift)

---

## HuggingFace Repo Layout

```
open-index/arctic/
  data/
    comments/
      2005/12/000.parquet
      2006/01/000.parquet
      ...
      2025/03/000.parquet
               001.parquet        ← large months have multiple shards
               002.parquet
    submissions/
      2005/12/000.parquet
      ...
  stats.csv
  README.md
```

**Shard target**: ~200 MB parquet (zstd). Each shard comes from ~2M JSONL lines (~1.6 GB).
A large recent comments month (~30 GB JSONL) yields ~20 shards.

**File count**: ~240 months × 2 types × avg 5 shards ≈ 2,400 files — well under HF's 100k limit.

### README.md frontmatter

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

Enables `load_dataset("open-index/arctic", "comments")` and HF Data Studio inspection.

---

## Source Data

**Arctic Shift** — full-history Reddit JSONL dumps per month, two files per month:

| File             | Contents                                 |
|------------------|------------------------------------------|
| `RC_YYYY-MM.zst` | Reddit Comments — zstd-compressed JSONL  |
| `RS_YYYY-MM.zst` | Reddit Submissions — zstd-compressed JSONL |

Each line is one JSON object. Sizes range from ~140 KB (2005-12) to ~30+ GB (recent months).

### Torrent Sources

**Bundle torrent (2005-12 → 2023-12)**: infohash `9c263fc85366c1ef8f5bb9da0203f4c8c8db75f4`
- Contains all months in one torrent; use selective file priority to download only the target
- Files inside torrent: `comments/RC_YYYY-MM.zst`, `submissions/RS_YYYY-MM.zst`
- `anacrolix/torrent` writes to `DataDir/reddit/comments/RC_YYYY-MM.zst`
  (the bundle torrent has `reddit/` as its root folder)

**Individual monthly torrents (2026-01+)**: separate infohash per month, hardcoded in `torrent.go`

**Not yet covered**: 2024-01 → 2025-12 need individual hashes added to `monthlyInfoHashes`
map in `pkg/arctic/torrent.go`. Guard in code returns a clear error if attempted.

Trackers used:
```
https://academictorrents.com/announce.php
udp://tracker.opentrackr.org:1337/announce
udp://tracker.openbittorrent.com:6969/announce
udp://open.stealth.si:80/announce
udp://exodus.desync.com:6969/announce
udp://tracker.torrent.eu.org:451/announce
```

---

## Pipeline (per month, per type)

```
check disk free
  └─ < min_free_gb (30 GB default) → stop (not skip)

already in stats.csv?
  └─ yes → skip

download RC_YYYY-MM.zst or RS_YYYY-MM.zst via torrent
  └─ 3-minute no-activity timeout (peers OR bytes = activity)
  └─ file lands at: RawDir/reddit/{comments,submissions}/R[CS]_YYYY-MM.zst
  └─ anacrolix/torrent may write <name>.part — renamed to final path after download

for each chunk of cfg.ChunkLines lines (default 2M):
  read from zstd decoder (klauspost/compress, 2 GB window)
  write chunk to WorkDir/chunk_NNNN.jsonl
  DuckDB :memory: → read_json_auto(chunk, ignore_errors=true, union_by_name=true)
    → SELECT explicit columns with TRY_CAST
    → COPY TO WorkDir/{type}/YYYY/MM/NNN.parquet (COMPRESSION ZSTD, ROW_GROUP_SIZE 131072)
  delete chunk_NNNN.jsonl immediately

delete RawDir/.../R[CS]_YYYY-MM.zst (stream exhausted)

HF commit (batched ≤50 ops per call):
  data/{type}/YYYY/MM/000.parquet … NNN.parquet
  stats.csv (upserted, sorted)
  README.md (regenerated)

delete WorkDir/{type}/YYYY/MM/*.parquet (after commit confirmed)
```

### Disk Budget

| Phase             | Peak size          |
|-------------------|--------------------|
| .zst download     | up to ~50 GB       |
| one JSONL chunk   | ~1.6 GB at a time  |
| shards (pre-commit)| N × 200 MB        |
| **Worst case**    | **~56 GB**         |

Min-free-GB check (default 30 GB) fires before each download, giving headroom for the
.zst + chunk + shards simultaneously.

---

## Parquet Schemas

DuckDB column selection with `TRY_CAST` — unknown JSON fields are silently dropped.

### Comments

| Column            | DuckDB Type | Notes                            |
|-------------------|-------------|----------------------------------|
| id                | VARCHAR     |                                  |
| author            | VARCHAR     |                                  |
| subreddit         | VARCHAR     |                                  |
| body              | VARCHAR     |                                  |
| score             | BIGINT      |                                  |
| created_utc       | BIGINT      | Unix seconds                     |
| created_at        | TIMESTAMP   | `epoch_ms(created_utc * 1000)`   |
| body_length       | BIGINT      | `LENGTH(body)`                   |
| link_id           | VARCHAR     |                                  |
| parent_id         | VARCHAR     |                                  |
| distinguished     | VARCHAR     |                                  |
| author_flair_text | VARCHAR     |                                  |

### Submissions

| Column             | DuckDB Type | Notes                          |
|--------------------|-------------|--------------------------------|
| id                 | VARCHAR     |                                |
| author             | VARCHAR     |                                |
| subreddit          | VARCHAR     |                                |
| title              | VARCHAR     |                                |
| selftext           | VARCHAR     |                                |
| score              | BIGINT      |                                |
| created_utc        | BIGINT      | Unix seconds                   |
| created_at         | TIMESTAMP   | `epoch_ms(created_utc * 1000)` |
| title_length       | BIGINT      | `LENGTH(title)`                |
| num_comments       | BIGINT      |                                |
| url                | VARCHAR     |                                |
| over_18            | BOOLEAN     |                                |
| link_flair_text    | VARCHAR     |                                |
| author_flair_text  | VARCHAR     |                                |

---

## stats.csv Schema

Tracks committed (year, month, type) triples. Written atomically (temp+rename). Upsert on re-run.

```
year,month,type,shards,count,size_bytes,dur_download_s,dur_process_s,dur_commit_s,committed_at
2005,12,comments,1,1075,141717,36.7,1.0,17.0,2026-03-15T00:46:19Z
2005,12,submissions,1,234,12345,40.1,0.9,8.2,2026-03-15T00:47:10Z
```

`CommittedSet(rows)` returns `map["YYYY-MM/type"]bool` — used for O(1) skip checks.

---

## Package Structure

```
pkg/arctic/
  config.go       //go:build !windows — Config struct, path helpers, FreeDiskGB (syscall.Statfs)
  torrent.go      — DownloadZst: torrent download with .part rename, verbose progress, 3-min timeout
  process.go      — ProcessZst: stream .zst → chunks → DuckDB → parquet shards
  stats.go        — StatsRow, ReadStatsCSV, WriteStatsCSV (atomic), CommittedSet
  hf.go           — HFOp{LocalPath, PathInRepo, Delete}, CommitFn type
  readme.go       — GenerateREADME: DatasetDict frontmatter + stats table + ASCII bar chart
  task_publish.go — PublishTask: outer loop, disk checks, cleanupWork, orchestration

cli/
  arctic.go         — NewArctic() cobra parent command
  arctic_publish.go — newArcticPublish() + runArcticPublish() + progress formatting
```

### Key Implementation Decisions

**Download to file, then process** (not streaming from torrent): anacrolix/torrent's sequential
reader has unpredictable blocking; downloading to disk first gives reliable progress tracking
and allows clean retry on failure.

**`.part` file rename**: anacrolix/torrent writes `<name>.part` and renames on completion,
but deferred `cl.Close()` can race with the rename. We explicitly rename after `cl.Download()`
returns but before `defer cl.Close()` fires.

**cleanupWork() on startup**: removes stale `chunk_*.jsonl`, `{comments,submissions}/` shard
dirs, and `R[CS]_*.zst` + `R[CS]_*.zst.part` files from the raw dir. Safe to re-run any time.

**No-activity timeout**: 3-minute window. Any callback with `peers > 0 OR bytes > 0`
resets the timer. This handles both slow DHT metadata fetch and stalled mid-download.

**HF batch commits**: ≤50 ops per commit call (HF API limit). Most months fit in one batch;
large months with 20+ shards need 1-2 batches.

**Defensive slice copy**: stats rows use `make+copy` rather than `append` to avoid backing
array aliasing between the caller's slice and the appended result.

---

## CLI Reference

```
search arctic publish [flags]

Flags:
  --repo-root string     local HF repo root   (default: $HOME/data/arctic/repo)
  --repo string          HuggingFace repo ID  (default: open-index/arctic)
  --from string          start month YYYY-MM  (default: 2005-12)
  --to string            end month YYYY-MM    (default: current month)
  --min-free-gb int      minimum free disk GB (default: 30)
  --chunk-lines int      lines per JSONL chunk (default: 2,000,000)
  --private              mark HF repo private on creation

Env:
  HF_TOKEN              required — Hugging Face write token
```

Progress output (lipgloss-styled):
```
  [2005-12] comments   ↓ 36.7s  ⚙ 1.0s  ↑ 17.0s  1 shards  1,075 rows  141717 B
  [2005-12] submissions ↓ 40.1s  ⚙ 0.9s  ↑ 8.2s   1 shards    234 rows   12345 B
  ...

  Done!

  Committed  2 months
  Skipped    0 months
  Elapsed    1m43s
```

Download progress uses `\r` for in-place updates:
```
  [2005-12] comments  downloading  126 KB / 143 KB  88%  0.0 MB/s  8 peers
```

---

## README.md Template

Generated by `GenerateREADME(rows []StatsRow)` from an embedded Go template.

Structure:
1. DatasetDict YAML frontmatter (two configs: `comments`, `submissions`)
2. Title + source attribution (Arctic Shift / PushShift)
3. Quick-start Python code block
4. Aggregate stats table (months, rows, size per type)
5. Year-by-year ASCII bar chart (rows grouped by year, normalized to 40-char bar)
6. Column schema tables (comments, submissions)
7. License note (Reddit user content, CC BY 4.0 redistribution by Arctic Shift)

---

## Deployment

### Server 2 setup

```bash
# build on server (native amd64, fast)
make build-on-server SERVER=2

# start screen session
ssh root@server2
screen -S arctic
export HF_TOKEN=hf_...
~/bin/search arctic publish --from 2005-12 --repo open-index/arctic --min-free-gb 30 \
  2>&1 | tee /tmp/arctic_publish.log

# detach: Ctrl+A D
# tail:   ssh root@server2 'tail -f /tmp/arctic_publish.log'
```

### Resume

Fully idempotent. Kill and restart any time — `stats.csv` tracks committed pairs,
cleanupWork() removes stale files, and the loop resumes from where it left off.

### Adding 2024–2025 torrent hashes

Edit `pkg/arctic/torrent.go`, add entries to `monthlyInfoHashes`:

```go
var monthlyInfoHashes = map[string]string{
    "2024-01": "<infohash from download_links.md>",
    // ...
    "2026-01": "8412b89151101d88c915334c45d9c223169a1a60",
    "2026-02": "c5ba00048236b60f819dbf010e9034d24fc291fb",
}
```

Source: https://github.com/ArthurHeitmann/arctic_shift/blob/master/download_links.md

---

## Known Gaps (post-0730)

1. **2024-01 → 2025-12 hashes missing** — pipeline will error cleanly when it reaches 2024-01;
   fill in `monthlyInfoHashes` from download_links.md before then.
2. **No HTTP fallback** — if torrent swarm is dead, no alternative download path. Add
   direct HTTP download from Arctic Shift's filen.io mirror as fallback if needed.
3. **No retry on transient HF commit error** — if HF returns 5xx, the run aborts. Add
   exponential backoff retry around `HFCommit` calls if needed.
