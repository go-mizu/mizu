# Arctic Publish Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `pkg/arctic` + `search arctic publish` command that downloads the Arctic Shift Reddit dump from torrents month-by-month, converts to parquet shards, and publishes to HuggingFace as `open-index/arctic`.

**Architecture:** Sequential month×type loop (oldest-first, comments then submissions). Each iteration: download `.zst` → stream-process into parquet shards (chunk by chunk, cleaning as we go) → HF commit → delete local files. Resume via `stats.csv`. Disk constraint managed by per-iteration free-space check.

**Tech Stack:** Go, `pkg/torrent` (anacrolix/torrent wrapper), klauspost/compress/zstd, DuckDB (:memory: per chunk), Cobra CLI, existing `cli` HF client, lipgloss output.

**Spec:** `spec/0729_arctic.md`

---

## Chunk 1: Config, Stats, HF types

### Task 1: `pkg/arctic/config.go` — Config struct and path helpers

**Files:**
- Create: `pkg/arctic/config.go`

This file owns all path derivation. No business logic. Follow `pkg/hn2/config.go` exactly.

- [ ] **Step 1: Create `pkg/arctic/config.go`**

```go
package arctic

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// Config holds all configuration for the arctic publish pipeline.
type Config struct {
	RepoRoot  string // local HF repo clone root
	HFRepo    string // HuggingFace repo ID
	RawDir    string // where .zst files are downloaded
	WorkDir   string // where chunk .jsonl and shard .parquet files live
	MinFreeGB int    // minimum free disk GB before starting a download
	ChunkLines int   // JSONL lines per parquet shard (0 → default 2_000_000)
}

// DefaultConfig returns a Config filled with environment-derived defaults.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	root := envOr("MIZU_ARCTIC_REPO_ROOT", filepath.Join(home, "data", "arctic", "repo"))
	raw  := envOr("MIZU_ARCTIC_RAW_DIR",   filepath.Join(home, "data", "arctic", "raw"))
	work := envOr("MIZU_ARCTIC_WORK_DIR",  filepath.Join(home, "data", "arctic", "work"))
	minFree := envIntOr("MIZU_ARCTIC_MIN_FREE_GB", 30)
	chunkLines := envIntOr("MIZU_ARCTIC_CHUNK_LINES", 2_000_000)
	return Config{
		RepoRoot:   root,
		HFRepo:     "open-index/arctic",
		RawDir:     raw,
		WorkDir:    work,
		MinFreeGB:  minFree,
		ChunkLines: chunkLines,
	}
}

// WithDefaults fills zero fields with DefaultConfig values.
func (c Config) WithDefaults() Config {
	def := DefaultConfig()
	if c.RepoRoot   == "" { c.RepoRoot   = def.RepoRoot }
	if c.HFRepo     == "" { c.HFRepo     = def.HFRepo }
	if c.RawDir     == "" { c.RawDir     = def.RawDir }
	if c.WorkDir    == "" { c.WorkDir    = def.WorkDir }
	if c.MinFreeGB  == 0  { c.MinFreeGB  = def.MinFreeGB }
	if c.ChunkLines == 0  { c.ChunkLines = def.ChunkLines }
	return c
}

// StatsCSVPath returns the path to stats.csv inside the repo root.
func (c Config) StatsCSVPath() string { return filepath.Join(c.RepoRoot, "stats.csv") }

// READMEPath returns the path to README.md inside the repo root.
func (c Config) READMEPath() string { return filepath.Join(c.RepoRoot, "README.md") }

// ZstPath returns the local path for a downloaded .zst file.
// prefix is "RC" (comments) or "RS" (submissions), ym is "YYYY-MM".
func (c Config) ZstPath(prefix, ym string) string {
	return filepath.Join(c.RawDir, fmt.Sprintf("%s_%s.zst", prefix, ym))
}

// ChunkPath returns the path for a temporary JSONL chunk file.
func (c Config) ChunkPath(n int) string {
	return filepath.Join(c.WorkDir, fmt.Sprintf("chunk_%04d.jsonl", n))
}

// ShardLocalDir returns the local directory for shards of a (type, year, month).
// typ is "comments" or "submissions", year is "2025", mm is "03".
func (c Config) ShardLocalDir(typ, year, mm string) string {
	return filepath.Join(c.WorkDir, typ, year, mm)
}

// ShardLocalPath returns the path for a specific shard.
func (c Config) ShardLocalPath(typ, year, mm string, n int) string {
	return filepath.Join(c.ShardLocalDir(typ, year, mm), fmt.Sprintf("%03d.parquet", n))
}

// ShardHFPath returns the HuggingFace path for a shard.
// e.g. "data/comments/2025/03/000.parquet"
func (c Config) ShardHFPath(typ, year, mm string, n int) string {
	return fmt.Sprintf("data/%s/%s/%s/%03d.parquet", typ, year, mm, n)
}

// EnsureDirs creates all required local directories.
func (c Config) EnsureDirs() error {
	for _, dir := range []string{c.RepoRoot, c.RawDir, c.WorkDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	return nil
}

// FreeDiskGB returns free disk space in GB for the partition containing WorkDir.
func (c Config) FreeDiskGB() (float64, error) {
	var st syscall.Statfs_t
	dir := c.WorkDir
	if dir == "" {
		dir = c.RawDir
	}
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, fmt.Errorf("statfs %s: %w", dir, err)
	}
	freeBytes := st.Bavail * uint64(st.Bsize)
	return float64(freeBytes) / (1024 * 1024 * 1024), nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return def
}
```

- [ ] **Step 2: Compile check**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build ./pkg/arctic/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/arctic/config.go
git commit -m "feat(arctic): add Config with path helpers and disk-free check"
```

---

### Task 2: `pkg/arctic/hf.go` and `pkg/arctic/stats.go`

**Files:**
- Create: `pkg/arctic/hf.go`
- Create: `pkg/arctic/stats.go`

- [ ] **Step 1: Create `pkg/arctic/hf.go`**

```go
package arctic

import "context"

// HFOp describes a single file operation in a Hugging Face commit.
type HFOp struct {
	LocalPath  string // empty if Delete=true
	PathInRepo string
	Delete     bool
}

// CommitFn commits a batch of HF operations and returns the commit URL.
type CommitFn func(ctx context.Context, ops []HFOp, message string) (string, error)
```

- [ ] **Step 2: Create `pkg/arctic/stats.go`**

Follow `pkg/hn2/stats.go` exactly. Atomic CSV rewrite via temp+rename.

```go
package arctic

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// StatsRow tracks one committed (year, month, type) triple.
type StatsRow struct {
	Year         int
	Month        int
	Type         string // "comments" | "submissions"
	Shards       int
	Count        int64
	SizeBytes    int64
	DurDownloadS float64
	DurProcessS  float64
	DurCommitS   float64
	CommittedAt  time.Time
}

// Key returns the canonical string key for this row.
func (r StatsRow) Key() string {
	return fmt.Sprintf("%04d-%02d/%s", r.Year, r.Month, r.Type)
}

// ReadStatsCSV reads stats.csv; returns empty slice (not error) if file absent.
func ReadStatsCSV(path string) ([]StatsRow, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	// skip header
	if _, err := r.Read(); err != nil {
		return nil, nil
	}
	var rows []StatsRow
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 10 {
			continue
		}
		var row StatsRow
		row.Year, _         = strconv.Atoi(rec[0])
		row.Month, _        = strconv.Atoi(rec[1])
		row.Type             = rec[2]
		row.Shards, _       = strconv.Atoi(rec[3])
		row.Count, _        = strconv.ParseInt(rec[4], 10, 64)
		row.SizeBytes, _    = strconv.ParseInt(rec[5], 10, 64)
		row.DurDownloadS, _ = strconv.ParseFloat(rec[6], 64)
		row.DurProcessS, _  = strconv.ParseFloat(rec[7], 64)
		row.DurCommitS, _   = strconv.ParseFloat(rec[8], 64)
		row.CommittedAt, _  = time.Parse(time.RFC3339, rec[9])
		rows = append(rows, row)
	}
	return rows, nil
}

// WriteStatsCSV upserts rows and atomically rewrites stats.csv.
func WriteStatsCSV(path string, rows []StatsRow) error {
	// upsert: merge by key
	index := make(map[string]StatsRow)
	for _, r := range rows {
		index[r.Key()] = r
	}
	merged := make([]StatsRow, 0, len(index))
	for _, r := range index {
		merged = append(merged, r)
	}
	sort.Slice(merged, func(i, j int) bool {
		a, b := merged[i], merged[j]
		if a.Year != b.Year   { return a.Year < b.Year }
		if a.Month != b.Month { return a.Month < b.Month }
		return a.Type < b.Type
	})
	return writeCSVAtomic(path, merged)
}

// CommittedSet returns a set of keys for already-committed (month, type) pairs.
func CommittedSet(rows []StatsRow) map[string]bool {
	m := make(map[string]bool, len(rows))
	for _, r := range rows {
		m[r.Key()] = true
	}
	return m
}

func writeCSVAtomic(path string, rows []StatsRow) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".stats_*.csv")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpPath) // no-op if rename succeeded
	}()

	w := csv.NewWriter(tmp)
	w.Write([]string{"year","month","type","shards","count","size_bytes",
		"dur_download_s","dur_process_s","dur_commit_s","committed_at"})
	for _, r := range rows {
		w.Write([]string{
			strconv.Itoa(r.Year),
			strconv.Itoa(r.Month),
			r.Type,
			strconv.Itoa(r.Shards),
			strconv.FormatInt(r.Count, 10),
			strconv.FormatInt(r.SizeBytes, 10),
			strconv.FormatFloat(r.DurDownloadS, 'f', 2, 64),
			strconv.FormatFloat(r.DurProcessS, 'f', 2, 64),
			strconv.FormatFloat(r.DurCommitS, 'f', 2, 64),
			r.CommittedAt.UTC().Format(time.RFC3339),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
```

- [ ] **Step 3: Build check**

```bash
go build ./pkg/arctic/...
```

- [ ] **Step 4: Commit**

```bash
git add pkg/arctic/hf.go pkg/arctic/stats.go
git commit -m "feat(arctic): add HFOp/CommitFn types and StatsRow CSV read/write"
```

---

## Chunk 2: Torrent download

### Task 3: `pkg/arctic/torrent.go` — download one .zst file

**Files:**
- Create: `pkg/arctic/torrent.go`

Downloads one `RC_YYYY-MM.zst` or `RS_YYYY-MM.zst` from the appropriate torrent to `outPath`.
Uses the same `pkg/torrent` client as `pkg/reddit/arctic_torrent.go`. For months 2005-06 to 2023-12, uses the bundle torrent with selective file priority. For 2024-01+, uses per-month torrent hashes.

- [ ] **Step 1: Create `pkg/arctic/torrent.go`**

```go
package arctic

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/torrent"
)

// bundleInfoHash is the Academic Torrents info hash for the full Arctic Shift
// bundle torrent covering 2005-06 through 2023-12.
const bundleInfoHash = "9c263fc85366c1ef8f5bb9da0203f4c8c8db75f4"

// arcticTrackers are the tracker URLs for the bundle torrent.
var arcticTrackers = []string{
	"https://academictorrents.com/announce.php",
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://tracker.openbittorrent.com:6969/announce",
	"udp://open.stealth.si:80/announce",
	"udp://exodus.desync.com:6969/announce",
	"udp://tracker.torrent.eu.org:451/announce",
}

// monthlyInfoHashes maps "YYYY-MM" to the per-month torrent info hash (2024-01+).
// For months not listed here, the bundle torrent is used (selective download).
// Fill in 2024-01..2025-12 from download_links.md before deploying.
var monthlyInfoHashes = map[string]string{
	"2026-01": "8412b89151101d88c915334c45d9c223169a1a60",
	"2026-02": "c5ba00048236b60f819dbf010e9034d24fc291fb",
}

// zstPrefix returns the file prefix for a type: "RC" for comments, "RS" for submissions.
func zstPrefix(typ string) string {
	if typ == "comments" {
		return "RC"
	}
	return "RS"
}

// DownloadProgress reports torrent download progress.
type DownloadProgress struct {
	Phase      string // "metadata" | "peers" | "downloading" | "done"
	BytesDone  int64
	BytesTotal int64
	SpeedBps   float64
	Peers      int
	Elapsed    time.Duration
}

// DownloadCallback is called with progress updates during download.
type DownloadCallback func(DownloadProgress)

// DownloadZst downloads RC_YYYY-MM.zst or RS_YYYY-MM.zst to outPath.
// year and month are the calendar year/month (e.g. 2023, 6).
// typ is "comments" or "submissions".
// Returns download duration on success.
func DownloadZst(ctx context.Context, cfg Config, year, month int, typ string,
	outPath string, cb DownloadCallback) (time.Duration, error) {

	ym := fmt.Sprintf("%04d-%02d", year, month)
	prefix := zstPrefix(typ)
	// Path inside the torrent (bundle layout): prefix_YYYY-MM.zst
	fileInTorrent := fmt.Sprintf("%s_%s.zst", prefix, ym)

	start := time.Now()

	if cb != nil {
		cb(DownloadProgress{Phase: "metadata"})
	}

	// Determine which torrent to use.
	infoHash := bundleInfoHash
	if h, ok := monthlyInfoHashes[ym]; ok {
		infoHash = h
	}

	tcfg := torrent.Config{
		DataDir:  cfg.RawDir,
		InfoHash: infoHash,
		Trackers: arcticTrackers,
		NoUpload: true,
	}

	cl, err := torrent.New(tcfg)
	if err != nil {
		return 0, fmt.Errorf("torrent client: %w", err)
	}
	defer cl.Close()

	// 60-second peer discovery timeout: cancel if no bytes flow.
	dlCtx, dlCancel := context.WithCancel(ctx)
	defer dlCancel()

	var lastBytes int64
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		noProgress := time.Now()
		for {
			select {
			case <-dlCtx.Done():
				return
			case <-t.C:
				if lastBytes > 0 {
					noProgress = time.Now()
				} else if time.Since(noProgress) > 60*time.Second {
					dlCancel()
					return
				}
			}
		}
	}()

	err = cl.Download(dlCtx, []string{fileInTorrent}, func(p torrent.Progress) {
		lastBytes = p.BytesCompleted
		if cb != nil {
			cb(DownloadProgress{
				Phase:      "downloading",
				BytesDone:  p.BytesCompleted,
				BytesTotal: p.BytesTotal,
				SpeedBps:   p.Speed,
				Peers:      p.Peers,
				Elapsed:    p.Elapsed,
			})
		}
	})
	if err != nil {
		if dlCtx.Err() != nil && ctx.Err() == nil {
			return 0, fmt.Errorf("torrent timeout: no peers found after 60s for %s", fileInTorrent)
		}
		return 0, fmt.Errorf("torrent download %s: %w", fileInTorrent, err)
	}

	if cb != nil {
		cb(DownloadProgress{Phase: "done", Elapsed: time.Since(start)})
	}
	return time.Since(start), nil
}
```

- [ ] **Step 2: Build check**

```bash
go build ./pkg/arctic/...
```

- [ ] **Step 3: Commit**

```bash
git add pkg/arctic/torrent.go
git commit -m "feat(arctic): torrent download for one month's .zst file"
```

---

## Chunk 3: Streaming chunk processor

### Task 4: `pkg/arctic/process.go` — stream .zst → JSONL chunks → parquet shards

**Files:**
- Create: `pkg/arctic/process.go`

This is the core of the pipeline. Opens the `.zst` file, wraps with a zstd decoder, reads
`cfg.ChunkLines` lines at a time into a temp `.jsonl` file, imports via DuckDB `:memory:`,
exports to a numbered shard parquet, then deletes the chunk file before moving to the next.

The schema SELECT is explicit per type — no unknown columns leak into the parquet.

- [ ] **Step 1: Create `pkg/arctic/process.go`**

```go
package arctic

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/klauspost/compress/zstd"
)

// ShardResult describes one written parquet shard.
type ShardResult struct {
	Index     int
	LocalPath string
	Rows      int64
	SizeBytes int64
}

// ProcessResult is the aggregate result for one (month, type) processing run.
type ProcessResult struct {
	Shards    []ShardResult
	TotalRows int64
	TotalSize int64
	Duration  time.Duration
}

// ShardCallback is called after each shard is written.
type ShardCallback func(ShardResult)

// ProcessZst streams the .zst file at zstPath through a zstd decoder, reads
// lines in chunks, and writes each chunk to a numbered parquet shard.
// typ must be "comments" or "submissions".
// year and mm are the "YYYY" and "MM" strings used for output path construction.
// Deletes each chunk temp file immediately after the shard is written.
func ProcessZst(ctx context.Context, cfg Config, zstPath, typ, year, mm string,
	cb ShardCallback) (ProcessResult, error) {

	start := time.Now()

	f, err := os.Open(zstPath)
	if err != nil {
		return ProcessResult{}, fmt.Errorf("open zst: %w", err)
	}
	defer f.Close()

	dec, err := zstd.NewReader(f, zstd.WithDecoderMaxWindow(1<<31))
	if err != nil {
		return ProcessResult{}, fmt.Errorf("zstd reader: %w", err)
	}
	defer dec.Close()

	// Ensure output shard directory exists.
	shardDir := cfg.ShardLocalDir(typ, year, mm)
	if err := os.MkdirAll(shardDir, 0o755); err != nil {
		return ProcessResult{}, fmt.Errorf("mkdir shards: %w", err)
	}

	var result ProcessResult
	scanner := bufio.NewScanner(dec)
	// 16 MB token buffer for very long JSON lines.
	scanner.Buffer(make([]byte, 16*1024*1024), 16*1024*1024)

	chunkIdx := 0
	lines := make([]string, 0, cfg.ChunkLines)

	flush := func() error {
		if len(lines) == 0 {
			return nil
		}
		sr, err := writeChunkToShard(ctx, cfg, lines, typ, year, mm, chunkIdx)
		if err != nil {
			return fmt.Errorf("shard %d: %w", chunkIdx, err)
		}
		result.Shards = append(result.Shards, sr)
		result.TotalRows += sr.Rows
		result.TotalSize += sr.SizeBytes
		if cb != nil {
			cb(sr)
		}
		lines = lines[:0]
		chunkIdx++
		return nil
	}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ProcessResult{}, ctx.Err()
		default:
		}
		lines = append(lines, scanner.Text())
		if len(lines) >= cfg.ChunkLines {
			if err := flush(); err != nil {
				return ProcessResult{}, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return ProcessResult{}, fmt.Errorf("scan jsonl: %w", err)
	}
	if err := flush(); err != nil {
		return ProcessResult{}, err
	}

	result.Duration = time.Since(start)
	return result, nil
}

// writeChunkToShard writes lines to a temp chunk file, imports to DuckDB :memory:,
// and exports as a zstd parquet shard. Deletes the chunk file on completion.
func writeChunkToShard(ctx context.Context, cfg Config, lines []string,
	typ, year, mm string, idx int) (ShardResult, error) {

	chunkPath := cfg.ChunkPath(idx)
	if err := os.MkdirAll(filepath.Dir(chunkPath), 0o755); err != nil {
		return ShardResult{}, err
	}

	// Write chunk jsonl.
	cf, err := os.Create(chunkPath)
	if err != nil {
		return ShardResult{}, fmt.Errorf("create chunk: %w", err)
	}
	w := bufio.NewWriterSize(cf, 8*1024*1024)
	for _, l := range lines {
		w.WriteString(l)
		w.WriteByte('\n')
	}
	if err := w.Flush(); err != nil {
		cf.Close()
		os.Remove(chunkPath)
		return ShardResult{}, fmt.Errorf("flush chunk: %w", err)
	}
	cf.Close()

	// Always clean up chunk file.
	defer os.Remove(chunkPath)

	shardPath := cfg.ShardLocalPath(typ, year, mm, idx)

	// DuckDB in-memory: import chunk, export shard.
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return ShardResult{}, fmt.Errorf("duckdb open: %w", err)
	}
	defer db.Close()

	esc := func(s string) string { return strings.ReplaceAll(s, "'", "''") }

	selectCols := commentsSelect
	if typ == "submissions" {
		selectCols = submissionsSelect
	}

	importSQL := fmt.Sprintf(`CREATE TABLE data AS
SELECT %s
FROM read_json_auto('%s',
    format='newline_delimited',
    maximum_object_size=10485760,
    ignore_errors=true,
    union_by_name=true
)`, selectCols, esc(chunkPath))

	if _, err := db.ExecContext(ctx, importSQL); err != nil {
		return ShardResult{}, fmt.Errorf("duckdb import: %w", err)
	}

	var rowCount int64
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM data").Scan(&rowCount)

	if err := os.MkdirAll(filepath.Dir(shardPath), 0o755); err != nil {
		return ShardResult{}, err
	}
	exportSQL := fmt.Sprintf("COPY data TO '%s' (FORMAT PARQUET, COMPRESSION ZSTD, ROW_GROUP_SIZE 131072)",
		esc(shardPath))
	if _, err := db.ExecContext(ctx, exportSQL); err != nil {
		os.Remove(shardPath)
		return ShardResult{}, fmt.Errorf("duckdb export: %w", err)
	}

	fi, err := os.Stat(shardPath)
	if err != nil {
		return ShardResult{}, fmt.Errorf("stat shard: %w", err)
	}

	return ShardResult{
		Index:     idx,
		LocalPath: shardPath,
		Rows:      rowCount,
		SizeBytes: fi.Size(),
	}, nil
}

// commentsSelect selects the canonical comments schema from read_json_auto output.
const commentsSelect = `
    TRY_CAST(id AS VARCHAR)                                         AS id,
    TRY_CAST(author AS VARCHAR)                                     AS author,
    TRY_CAST(subreddit AS VARCHAR)                                  AS subreddit,
    TRY_CAST(body AS VARCHAR)                                       AS body,
    TRY_CAST(score AS BIGINT)                                       AS score,
    TRY_CAST(created_utc AS BIGINT)                                 AS created_utc,
    CASE WHEN created_utc IS NOT NULL
         THEN epoch_ms(CAST(created_utc AS BIGINT) * 1000)
         ELSE NULL END                                              AS created_at,
    CASE WHEN body IS NOT NULL
         THEN LENGTH(CAST(body AS VARCHAR))
         ELSE 0 END                                                 AS body_length,
    TRY_CAST(link_id AS VARCHAR)                                    AS link_id,
    TRY_CAST(parent_id AS VARCHAR)                                  AS parent_id,
    TRY_CAST(distinguished AS VARCHAR)                              AS distinguished,
    TRY_CAST(author_flair_text AS VARCHAR)                          AS author_flair_text`

// submissionsSelect selects the canonical submissions schema.
const submissionsSelect = `
    TRY_CAST(id AS VARCHAR)                                         AS id,
    TRY_CAST(author AS VARCHAR)                                     AS author,
    TRY_CAST(subreddit AS VARCHAR)                                  AS subreddit,
    TRY_CAST(title AS VARCHAR)                                      AS title,
    TRY_CAST(selftext AS VARCHAR)                                   AS selftext,
    TRY_CAST(score AS BIGINT)                                       AS score,
    TRY_CAST(created_utc AS BIGINT)                                 AS created_utc,
    CASE WHEN created_utc IS NOT NULL
         THEN epoch_ms(CAST(created_utc AS BIGINT) * 1000)
         ELSE NULL END                                              AS created_at,
    CASE WHEN title IS NOT NULL
         THEN LENGTH(CAST(title AS VARCHAR))
         ELSE 0 END                                                 AS title_length,
    TRY_CAST(num_comments AS BIGINT)                                AS num_comments,
    TRY_CAST(url AS VARCHAR)                                        AS url,
    TRY_CAST(over_18 AS BOOLEAN)                                    AS over_18,
    TRY_CAST(link_flair_text AS VARCHAR)                            AS link_flair_text,
    TRY_CAST(author_flair_text AS VARCHAR)                          AS author_flair_text`
```

- [ ] **Step 2: Build check**

```bash
go build ./pkg/arctic/...
```

- [ ] **Step 3: Commit**

```bash
git add pkg/arctic/process.go
git commit -m "feat(arctic): streaming zst → chunk → parquet shard processor"
```

---

## Chunk 4: README generation

### Task 5: `pkg/arctic/readme.go` — README template

**Files:**
- Create: `pkg/arctic/readme.go`

Generates a world-class README with DatasetDict frontmatter, usage examples, aggregate stats
table, year-by-year bar chart, and schema tables. Uses only data from `stats.csv`.

- [ ] **Step 1: Create `pkg/arctic/readme.go`**

```go
package arctic

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"time"
)

// ReadmeData holds all template variables for README generation.
type ReadmeData struct {
	LatestMonth     string // "YYYY-MM"
	CommentMonths   int
	SubmissionMonths int
	CommentRows     int64
	SubmissionRows  int64
	CommentSize     int64  // bytes
	SubmissionSize  int64  // bytes
	GrowthChart     string // pre-rendered bar chart
	GeneratedAt     string
}

// GenerateREADME renders the README template with data derived from stats rows.
func GenerateREADME(rows []StatsRow) ([]byte, error) {
	data := buildReadmeData(rows)
	tmpl, err := template.New("readme").Parse(readmeTmpl)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildReadmeData(rows []StatsRow) ReadmeData {
	var data ReadmeData
	data.GeneratedAt = time.Now().UTC().Format("2006-01-02")

	yearRows := make(map[int][2]int64) // year → [comments, submissions]
	latestYM := ""

	for _, r := range rows {
		ym := fmt.Sprintf("%04d-%02d", r.Year, r.Month)
		if ym > latestYM {
			latestYM = ym
		}
		yr := yearRows[r.Year]
		if r.Type == "comments" {
			data.CommentMonths++
			data.CommentRows += r.Count
			data.CommentSize += r.SizeBytes
			yr[0] += r.Count
		} else {
			data.SubmissionMonths++
			data.SubmissionRows += r.Count
			data.SubmissionSize += r.SizeBytes
			yr[1] += r.Count
		}
		yearRows[r.Year] = yr
	}
	if latestYM == "" {
		latestYM = "—"
	}
	data.LatestMonth = latestYM
	data.GrowthChart = buildGrowthChart(yearRows)
	return data
}

func buildGrowthChart(yearRows map[int][2]int64) string {
	if len(yearRows) == 0 {
		return ""
	}
	years := make([]int, 0, len(yearRows))
	for y := range yearRows {
		years = append(years, y)
	}
	sort.Ints(years)

	var maxRows int64
	for _, yr := range yearRows {
		if yr[0]+yr[1] > maxRows {
			maxRows = yr[0] + yr[1]
		}
	}
	if maxRows == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("```\n")
	for _, y := range years {
		yr := yearRows[y]
		total := yr[0] + yr[1]
		barLen := int(float64(total) / float64(maxRows) * 40)
		bar := strings.Repeat("█", barLen)
		sb.WriteString(fmt.Sprintf("%d  %-40s  %s\n", y, bar, fmtCount(total)))
	}
	sb.WriteString("```")
	return sb.String()
}

func fmtCount(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1e9)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1e3)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func fmtBytes(n int64) string {
	switch {
	case n >= 1<<40:
		return fmt.Sprintf("%.1f TB", float64(n)/(1<<40))
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

const readmeTmpl = `---
configs:
- config_name: comments
  data_files:
  - split: train
    path: "data/comments/**/*.parquet"
- config_name: submissions
  data_files:
  - split: train
    path: "data/submissions/**/*.parquet"
license: other
language:
- en
tags:
- reddit
- social-media
- arctic-shift
- pushshift
pretty_name: Arctic Shift Reddit Archive
size_categories:
- 100B<n<1T
---

# Arctic Shift Reddit Archive

Full Reddit dataset (comments + submissions) sourced from the
[Arctic Shift](https://github.com/ArthurHeitmann/arctic_shift) project,
covering all subreddits from 2005-06 through **{{.LatestMonth}}**.

Data is organized as monthly parquet shards by type, making it easy to load
specific time ranges or work with comments and submissions independently.

## Quick Start

` + "```python" + `
from datasets import load_dataset

# Stream all comments (recommended — dataset is very large)
comments = load_dataset("open-index/arctic", "comments", streaming=True)
for item in comments["train"]:
    print(item["author"], item["body"][:80])

# Load a specific month's submissions (adjust path glob as needed)
from datasets import load_dataset
subs = load_dataset("open-index/arctic", "submissions",
                    data_files="data/submissions/2020/**/*.parquet")
` + "```" + `

## Dataset Stats

| Type        | Months | Rows | Parquet Size |
|-------------|--------|------|--------------|
| comments    | {{.CommentMonths}} | {{commentRows .}} | {{commentSize .}} |
| submissions | {{.SubmissionMonths}} | {{submissionRows .}} | {{submissionSize .}} |

*Updated: {{.GeneratedAt}}*

## Growth (rows per year, comments + submissions)

{{.GrowthChart}}

## Schema

### Comments

| Column | Type | Description |
|--------|------|-------------|
| id | VARCHAR | Comment ID |
| author | VARCHAR | Username |
| subreddit | VARCHAR | Subreddit name |
| body | VARCHAR | Comment text |
| score | BIGINT | Net upvotes |
| created_utc | BIGINT | Unix timestamp |
| created_at | TIMESTAMP | Derived from created_utc |
| body_length | BIGINT | Character count of body |
| link_id | VARCHAR | Parent submission ID |
| parent_id | VARCHAR | Parent comment or submission ID |
| distinguished | VARCHAR | mod/admin/null |
| author_flair_text | VARCHAR | Author flair |

### Submissions

| Column | Type | Description |
|--------|------|-------------|
| id | VARCHAR | Submission ID |
| author | VARCHAR | Username |
| subreddit | VARCHAR | Subreddit name |
| title | VARCHAR | Post title |
| selftext | VARCHAR | Post body (self posts) |
| score | BIGINT | Net upvotes |
| created_utc | BIGINT | Unix timestamp |
| created_at | TIMESTAMP | Derived from created_utc |
| title_length | BIGINT | Character count of title |
| num_comments | BIGINT | Comment count |
| url | VARCHAR | External URL or permalink |
| over_18 | BOOLEAN | NSFW flag |
| link_flair_text | VARCHAR | Post flair |
| author_flair_text | VARCHAR | Author flair |

## Source & License

Repackaged from [Arctic Shift](https://github.com/ArthurHeitmann/arctic_shift) monthly dumps,
which re-process the [PushShift](https://pushshift.io) Reddit archive.
Original content by Reddit users; distribution under Reddit's
[Public Content Policy](https://www.redditinc.com/policies/data-api-terms).
`
```

Note: The template uses `{{commentRows .}}` etc. which need to be registered as `FuncMap` entries. Update `GenerateREADME` to use a FuncMap:

```go
func GenerateREADME(rows []StatsRow) ([]byte, error) {
	data := buildReadmeData(rows)
	funcMap := template.FuncMap{
		"commentRows":     func(d ReadmeData) string { return fmtCount(d.CommentRows) },
		"commentSize":     func(d ReadmeData) string { return fmtBytes(d.CommentSize) },
		"submissionRows":  func(d ReadmeData) string { return fmtCount(d.SubmissionRows) },
		"submissionSize":  func(d ReadmeData) string { return fmtBytes(d.SubmissionSize) },
	}
	tmpl, err := template.New("readme").Funcs(funcMap).Parse(readmeTmpl)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
```

- [ ] **Step 2: Build check**

```bash
go build ./pkg/arctic/...
```

- [ ] **Step 3: Commit**

```bash
git add pkg/arctic/readme.go
git commit -m "feat(arctic): README template with DatasetDict frontmatter and stats"
```

---

## Chunk 5: Publish task and cleanup

### Task 6: `pkg/arctic/task_publish.go` — orchestration loop

**Files:**
- Create: `pkg/arctic/task_publish.go`

The main loop. Reads stats.csv, iterates (year, month, type) oldest-first,
checks disk, downloads, processes, commits, and cleans up.

- [ ] **Step 1: Create `pkg/arctic/task_publish.go`**

```go
package arctic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PublishOptions configures the publish run.
type PublishOptions struct {
	FromYear  int
	FromMonth int
	ToYear    int
	ToMonth   int
	HFCommit  CommitFn
}

// PublishState is emitted on each significant event.
type PublishState struct {
	Phase   string // "skip" | "disk_check" | "download" | "process" | "commit" | "done"
	YM      string // "YYYY-MM"
	Type    string // "comments" | "submissions"
	Shards  int
	Rows    int64
	Bytes   int64
	DurDown time.Duration
	DurProc time.Duration
	DurComm time.Duration
	Message string // for "disk_check" warning
}

// PublishMetric is the aggregate result returned on completion.
type PublishMetric struct {
	Committed int // (month,type) pairs committed
	Skipped   int
	Elapsed   time.Duration
}

// PublishTask orchestrates the full arctic publish pipeline.
type PublishTask struct {
	cfg  Config
	opts PublishOptions
}

// NewPublishTask constructs a PublishTask.
func NewPublishTask(cfg Config, opts PublishOptions) *PublishTask {
	return &PublishTask{cfg: cfg, opts: opts}
}

// Run executes the publish loop. Calls emit (if non-nil) on state changes.
func (t *PublishTask) Run(ctx context.Context, emit func(*PublishState)) (PublishMetric, error) {
	start := time.Now()
	metric := PublishMetric{}

	// Clean up any leftover work files from a previous interrupted run.
	t.cleanupWork()

	// Load existing stats.
	rows, err := ReadStatsCSV(t.cfg.StatsCSVPath())
	if err != nil {
		return metric, fmt.Errorf("read stats: %w", err)
	}
	committed := CommittedSet(rows)

	// Build month list.
	months := t.monthRange()

	for _, ym := range months {
		for _, typ := range []string{"comments", "submissions"} {
			if err := ctx.Err(); err != nil {
				return metric, err
			}

			key := ym.Key() + "/" + typ

			if committed[key] {
				metric.Skipped++
				if emit != nil {
					emit(&PublishState{Phase: "skip", YM: ym.String(), Type: typ})
				}
				continue
			}

			// Disk check.
			free, err := t.cfg.FreeDiskGB()
			if err != nil {
				return metric, fmt.Errorf("disk check: %w", err)
			}
			if free < float64(t.cfg.MinFreeGB) {
				msg := fmt.Sprintf("only %.1f GB free, need %d GB — stopping", free, t.cfg.MinFreeGB)
				if emit != nil {
					emit(&PublishState{Phase: "disk_check", YM: ym.String(), Type: typ, Message: msg})
				}
				return metric, fmt.Errorf("disk full: %s", msg)
			}

			if err := t.processOne(ctx, ym, typ, rows, emit); err != nil {
				return metric, fmt.Errorf("[%s] %s: %w", ym.String(), typ, err)
			}

			// Reload stats after successful commit.
			rows, _ = ReadStatsCSV(t.cfg.StatsCSVPath())
			committed = CommittedSet(rows)
			metric.Committed++
		}
	}

	metric.Elapsed = time.Since(start)
	if emit != nil {
		emit(&PublishState{Phase: "done"})
	}
	return metric, nil
}

// processOne handles the full pipeline for one (month, type):
// download → process → commit → cleanup.
func (t *PublishTask) processOne(ctx context.Context, ym ymKey, typ string,
	existingRows []StatsRow, emit func(*PublishState)) error {

	cfg := t.cfg
	prefix := zstPrefix(typ)
	zstPath := cfg.ZstPath(prefix, ym.String())
	year := fmt.Sprintf("%04d", ym.Year)
	mm := fmt.Sprintf("%02d", ym.Month)

	// --- Download ---
	if emit != nil {
		emit(&PublishState{Phase: "download", YM: ym.String(), Type: typ})
	}
	t0 := time.Now()
	durDown, err := DownloadZst(ctx, cfg, ym.Year, ym.Month, typ, zstPath, func(p DownloadProgress) {
		if emit != nil {
			emit(&PublishState{
				Phase: "download", YM: ym.String(), Type: typ,
				Bytes: p.BytesDone, Message: p.Phase,
			})
		}
	})
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	_ = t0
	_ = durDown

	// --- Process (stream zst → shards) ---
	if emit != nil {
		emit(&PublishState{Phase: "process", YM: ym.String(), Type: typ})
	}
	t1 := time.Now()
	procResult, err := ProcessZst(ctx, cfg, zstPath, typ, year, mm, func(sr ShardResult) {
		if emit != nil {
			emit(&PublishState{
				Phase: "process", YM: ym.String(), Type: typ,
				Shards: sr.Index + 1, Rows: sr.Rows, Bytes: sr.SizeBytes,
			})
		}
	})
	if err != nil {
		return fmt.Errorf("process: %w", err)
	}
	durProc := time.Since(t1)

	// Delete .zst now that we've consumed it.
	os.Remove(zstPath)

	// --- HF Commit ---
	if emit != nil {
		emit(&PublishState{Phase: "commit", YM: ym.String(), Type: typ,
			Shards: len(procResult.Shards), Rows: procResult.TotalRows, Bytes: procResult.TotalSize})
	}
	t2 := time.Now()

	newRow := StatsRow{
		Year:         ym.Year,
		Month:        ym.Month,
		Type:         typ,
		Shards:       len(procResult.Shards),
		Count:        procResult.TotalRows,
		SizeBytes:    procResult.TotalSize,
		DurDownloadS: durDown.Seconds(),
		DurProcessS:  durProc.Seconds(),
		CommittedAt:  time.Now().UTC(),
	}
	allRows := append(existingRows, newRow)

	readme, err := GenerateREADME(allRows)
	if err != nil {
		return fmt.Errorf("readme: %w", err)
	}
	if err := os.WriteFile(cfg.READMEPath(), readme, 0o644); err != nil {
		return fmt.Errorf("write readme: %w", err)
	}
	if err := WriteStatsCSV(cfg.StatsCSVPath(), allRows); err != nil {
		return fmt.Errorf("write stats: %w", err)
	}

	var ops []HFOp
	for _, sr := range procResult.Shards {
		ops = append(ops, HFOp{
			LocalPath:  sr.LocalPath,
			PathInRepo: cfg.ShardHFPath(typ, year, mm, sr.Index),
		})
	}
	ops = append(ops,
		HFOp{LocalPath: cfg.StatsCSVPath(), PathInRepo: "stats.csv"},
		HFOp{LocalPath: cfg.READMEPath(), PathInRepo: "README.md"},
	)

	// Batch commits: ≤50 ops per call.
	const batchSize = 50
	for i := 0; i < len(ops); i += batchSize {
		end := i + batchSize
		if end > len(ops) {
			end = len(ops)
		}
		msg := fmt.Sprintf("add %s/%s %s (%d shards, %s rows)",
			typ, ym.String(), year+"/"+mm, len(procResult.Shards),
			fmtCount(procResult.TotalRows))
		if _, err := t.opts.HFCommit(ctx, ops[i:end], msg); err != nil {
			return fmt.Errorf("hf commit: %w", err)
		}
	}

	durComm := time.Since(t2)
	newRow.DurCommitS = durComm.Seconds()

	// Update stats.csv with commit duration.
	allRows[len(allRows)-1] = newRow
	_ = WriteStatsCSV(cfg.StatsCSVPath(), allRows)

	// Delete local shards after commit.
	for _, sr := range procResult.Shards {
		os.Remove(sr.LocalPath)
	}
	// Remove shard dir if empty.
	shardDir := cfg.ShardLocalDir(typ, year, mm)
	os.Remove(shardDir)
	os.Remove(filepath.Dir(shardDir)) // year dir — ignore if not empty

	if emit != nil {
		emit(&PublishState{
			Phase: "committed", YM: ym.String(), Type: typ,
			Shards: len(procResult.Shards), Rows: procResult.TotalRows, Bytes: procResult.TotalSize,
			DurDown: durDown, DurProc: durProc, DurComm: durComm,
		})
	}

	return nil
}

// cleanupWork removes leftover work files from interrupted previous runs.
func (t *PublishTask) cleanupWork() {
	// Chunk files.
	matches, _ := filepath.Glob(filepath.Join(t.cfg.WorkDir, "chunk_*.jsonl"))
	for _, m := range matches {
		os.Remove(m)
	}
	// Leftover shard parquets.
	for _, typ := range []string{"comments", "submissions"} {
		dir := filepath.Join(t.cfg.WorkDir, typ)
		os.RemoveAll(dir)
	}
	// Leftover .zst files (from HTTP fallback).
	matches, _ = filepath.Glob(filepath.Join(t.cfg.RawDir, "R[CS]_*.zst"))
	for _, m := range matches {
		os.Remove(m)
	}
}

// ymKey is a (Year, Month) pair.
type ymKey struct {
	Year  int
	Month int
}

func (k ymKey) String() string { return fmt.Sprintf("%04d-%02d", k.Year, k.Month) }
func (k ymKey) Key() string    { return k.String() }

// monthRange returns all (year, month) pairs from opts.From to opts.To inclusive.
func (t *PublishTask) monthRange() []ymKey {
	from := ymKey{Year: t.opts.FromYear, Month: t.opts.FromMonth}
	to := ymKey{Year: t.opts.ToYear, Month: t.opts.ToMonth}
	if from.Year == 0 {
		from = ymKey{Year: 2005, Month: 6} // earliest Arctic Shift data
	}
	if to.Year == 0 {
		now := time.Now().UTC()
		to = ymKey{Year: now.Year(), Month: int(now.Month())}
	}

	var keys []ymKey
	cur := from
	for !after(cur, to) {
		keys = append(keys, cur)
		cur.Month++
		if cur.Month > 12 {
			cur.Month = 1
			cur.Year++
		}
	}
	return keys
}

func after(a, b ymKey) bool {
	if a.Year != b.Year {
		return a.Year > b.Year
	}
	return a.Month > b.Month
}
```

- [ ] **Step 2: Build check**

```bash
go build ./pkg/arctic/...
```

- [ ] **Step 3: Commit**

```bash
git add pkg/arctic/task_publish.go
git commit -m "feat(arctic): publish task — month loop, disk check, download→process→commit→cleanup"
```

---

## Chunk 6: CLI command

### Task 7: `cli/arctic.go` and `cli/arctic_publish.go` — CLI wiring

**Files:**
- Create: `cli/arctic.go`
- Create: `cli/arctic_publish.go`
- Modify: `cli/root.go` — add `root.AddCommand(NewArctic())`

Pattern: identical to `cli/hn.go` + `cli/hn_publish.go`. The arctic command is a parent
with `publish` as a subcommand.

- [ ] **Step 1: Create `cli/arctic.go`**

```go
package cli

import "github.com/spf13/cobra"

// NewArctic returns the "arctic" parent command.
func NewArctic() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "arctic",
		Short: "Arctic Shift Reddit dataset publishing",
	}
	cmd.AddCommand(newArcticPublish())
	return cmd
}
```

- [ ] **Step 2: Create `cli/arctic_publish.go`**

```go
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/arctic"
	"github.com/spf13/cobra"
)

func newArcticPublish() *cobra.Command {
	var (
		repoRoot   string
		repoID     string
		fromStr    string
		toStr      string
		minFreeGB  int
		chunkLines int
		private    bool
	)

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish Arctic Shift Reddit dump to Hugging Face",
		Long: `Publish the full Arctic Shift Reddit dataset to a Hugging Face repo.

Downloads monthly torrent archives (RC_YYYY-MM.zst for comments,
RS_YYYY-MM.zst for submissions), streams each through a zstd decoder,
chunks into parquet shards, uploads to HuggingFace, then deletes local files.

Fully resumable: reads stats.csv on startup and skips already-committed
(month, type) pairs. Re-run the same command after crash or disk-full.

Run in a screen session on the server:
  export HF_TOKEN=hf_...
  screen -S arctic
  search arctic publish --from 2005-06 --repo open-index/arctic`,
		Example: `  search arctic publish
  search arctic publish --from 2020-01
  search arctic publish --from 2005-06 --to 2023-12 --min-free-gb 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArcticPublish(cmd.Context(), repoRoot, repoID, fromStr, toStr,
				minFreeGB, chunkLines, private)
		},
	}

	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "Local HF repo root (default: $HOME/data/arctic/repo)")
	cmd.Flags().StringVar(&repoID, "repo", "open-index/arctic", "HuggingFace dataset repo ID")
	cmd.Flags().StringVar(&fromStr, "from", "2005-06", "Start month YYYY-MM (inclusive)")
	cmd.Flags().StringVar(&toStr, "to", "", "End month YYYY-MM inclusive (default: current month)")
	cmd.Flags().IntVar(&minFreeGB, "min-free-gb", 30, "Minimum free disk GB before each download")
	cmd.Flags().IntVar(&chunkLines, "chunk-lines", 0, "JSONL lines per parquet shard (default 2,000,000)")
	cmd.Flags().BoolVar(&private, "private", false, "Create HF repo as private if it does not exist")

	return cmd
}

func runArcticPublish(ctx context.Context, repoRoot, repoID, fromStr, toStr string,
	minFreeGB, chunkLines int, private bool) error {

	token := strings.TrimSpace(os.Getenv("HF_TOKEN"))
	if token == "" {
		return fmt.Errorf("HF_TOKEN environment variable is not set")
	}

	cfg := arctic.Config{
		RepoRoot:   repoRoot,
		HFRepo:     repoID,
		MinFreeGB:  minFreeGB,
		ChunkLines: chunkLines,
	}
	cfg = cfg.WithDefaults()

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("ensure dirs: %w", err)
	}

	hf := newHFClient(token)
	if err := hf.createDatasetRepo(ctx, repoID, private); err != nil {
		fmt.Printf("  note: create repo: %v\n", err)
	}

	hfCommitFn := func(ctx context.Context, ops []arctic.HFOp, message string) (string, error) {
		var hfOps []hfOperation
		for _, op := range ops {
			hfOps = append(hfOps, hfOperation{
				LocalPath:  op.LocalPath,
				PathInRepo: op.PathInRepo,
				Delete:     op.Delete,
			})
		}
		return hf.createCommit(ctx, repoID, "main", message, hfOps)
	}

	// Parse --from flag.
	var fromYear, fromMonth int
	if fromStr != "" {
		t, err := time.Parse("2006-01", fromStr)
		if err != nil {
			return fmt.Errorf("--from: expected YYYY-MM, got %q", fromStr)
		}
		fromYear, fromMonth = t.Year(), int(t.Month())
	}

	var toYear, toMonth int
	if toStr != "" {
		t, err := time.Parse("2006-01", toStr)
		if err != nil {
			return fmt.Errorf("--to: expected YYYY-MM, got %q", toStr)
		}
		toYear, toMonth = t.Year(), int(t.Month())
	}

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Arctic Publish → " + repoID))
	fmt.Println()
	fmt.Printf("  Repo root   %s\n", labelStyle.Render(cfg.RepoRoot))
	fmt.Printf("  HF repo     %s\n", infoStyle.Render(repoID))
	fmt.Printf("  From        %s\n", labelStyle.Render(fromStr))
	if toStr != "" {
		fmt.Printf("  To          %s\n", labelStyle.Render(toStr))
	}
	fmt.Printf("  Min free    %s GB\n", labelStyle.Render(fmt.Sprintf("%d", cfg.MinFreeGB)))
	fmt.Println()

	task := arctic.NewPublishTask(cfg, arctic.PublishOptions{
		FromYear:  fromYear,
		FromMonth: fromMonth,
		ToYear:    toYear,
		ToMonth:   toMonth,
		HFCommit:  hfCommitFn,
	})

	metric, err := task.Run(ctx, func(s *arctic.PublishState) {
		switch s.Phase {
		case "skip":
			fmt.Printf("  [%s] %s  %s\n",
				labelStyle.Render(s.YM), dimStyle(s.Type), labelStyle.Render("skip"))
		case "download":
			fmt.Printf("  [%s] %s  %s\n",
				labelStyle.Render(s.YM), infoStyle.Render(s.Type), labelStyle.Render("↓ downloading…"))
		case "process":
			fmt.Printf("  [%s] %s  %s\n",
				labelStyle.Render(s.YM), infoStyle.Render(s.Type), labelStyle.Render("⚙ processing…"))
		case "commit":
			fmt.Printf("  [%s] %s  %s  %d shards  %s rows\n",
				labelStyle.Render(s.YM), infoStyle.Render(s.Type),
				labelStyle.Render("↑ committing"),
				s.Shards, ccFmtInt64(s.Rows))
		case "committed":
			fmt.Printf("  [%s] %s  ↓ %s  ⚙ %s  ↑ %s  %d shards  %s rows  %s\n",
				labelStyle.Render(s.YM),
				successStyle.Render(s.Type),
				fmtDur(s.DurDown), fmtDur(s.DurProc), fmtDur(s.DurComm),
				s.Shards, ccFmtInt64(s.Rows), fmtSize(s.Bytes))
		case "disk_check":
			fmt.Printf("  %s %s\n", warningStyle.Render("DISK:"), s.Message)
		case "done":
			fmt.Println()
		}
	})

	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  Committed  %s\n", infoStyle.Render(fmt.Sprintf("%d", metric.Committed)))
	fmt.Printf("  Skipped    %s\n", labelStyle.Render(fmt.Sprintf("%d", metric.Skipped)))
	fmt.Printf("  Elapsed    %s\n", labelStyle.Render(metric.Elapsed.Round(time.Second).String()))
	return nil
}

func fmtDur(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func fmtSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func dimStyle(s string) string {
	return labelStyle.Render(s)
}
```

- [ ] **Step 3: Wire into `cli/root.go`**

Find the line `root.AddCommand(NewDiscord())` and add after it:

```go
root.AddCommand(NewArctic())
```

- [ ] **Step 4: Build check**

```bash
go build ./...
```

If compile errors about missing `dimStyle` or style helpers, look at existing style vars in
`cli/` (e.g. `cc_defaults.go` or `styles.go`) and use the appropriate existing style.

- [ ] **Step 5: Commit**

```bash
git add cli/arctic.go cli/arctic_publish.go cli/root.go
git commit -m "feat(arctic): CLI command 'search arctic publish'"
```

---

## Chunk 7: Build, deploy, and run on server 2

### Task 8: Build linux binary and deploy

**Files:**
- No new files. Uses existing Makefile targets.

- [ ] **Step 1: Verify the binary builds on macOS first**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build -o /tmp/search-test ./cmd/search/
/tmp/search-test arctic --help
```

Expected: shows arctic subcommand with publish.

- [ ] **Step 2: Check Makefile for server 2 target**

```bash
grep -n "SERVER=2\|server.2\|REMOTE_SSH" Makefile | head -20
```

Note the SSH host and deploy key variable names.

- [ ] **Step 3: Build linux binary on server (fastest method)**

```bash
make build-on-server SERVER=2
```

If `build-on-server` is not available, use:
```bash
make build-linux-noble
make deploy-linux-noble SERVER=2
```

- [ ] **Step 4: SSH to server 2 and verify**

```bash
# use the server 2 SSH alias from Makefile
ssh <server2-host> "search arctic --help"
```

Expected: shows the `arctic publish` subcommand.

- [ ] **Step 5: Check disk space on server 2**

```bash
ssh <server2-host> "df -h ~ && df -h /home"
```

Confirm at least 80 GB free before starting.

- [ ] **Step 6: Start the screen session on server 2**

```bash
ssh <server2-host>
export HF_TOKEN=<your-token>
screen -S arctic
search arctic publish \
  --from 2005-06 \
  --repo open-index/arctic \
  --min-free-gb 30
```

Detach with `Ctrl+A D`. Reattach with `screen -r arctic`.

- [ ] **Step 7: Monitor first few months**

Reattach after ~10 minutes and confirm:
- 2005-06 comments: downloaded, processed, committed, cleaned up
- 2005-06 submissions: same
- 2005-07 started or done
- No leftover `.zst` or chunk files in `~/data/arctic/raw/` or `~/data/arctic/work/`

```bash
ssh <server2-host> "ls ~/data/arctic/raw/ && ls ~/data/arctic/work/ && df -h ~"
```

- [ ] **Step 8: Verify HuggingFace repo**

Open `https://huggingface.co/datasets/open-index/arctic` and confirm:
- README renders with DatasetDict config
- `data/comments/2005/06/000.parquet` visible in file tree
- Data Studio can preview the dataset with `load_dataset("open-index/arctic", "comments")`

---

## Appendix: Key existing files to reference

| File | Purpose |
|------|---------|
| `cli/hn_publish.go` | Pattern for CLI command (flags, HF bridge, progress output) |
| `cli/hn.go` | Pattern for parent command with subcommands |
| `pkg/hn2/stats.go` | Pattern for atomic CSV read/write |
| `pkg/hn2/hf.go` | HFOp/CommitFn type definitions |
| `pkg/reddit/arctic_torrent.go` | Torrent download pattern with 60s peer timeout |
| `pkg/reddit/arctic_import.go` | DuckDB import pattern (schema SQL, derived cols) |
| `pkg/torrent/client.go` | `torrent.Config`, `torrent.New()`, `cl.Download()` API |
| `cli/cc_publish_hf.go` | `hfOperation`, `createCommit`, `createDatasetRepo` |

## Appendix: Disk budget (worst case)

| Stage | Disk usage |
|-------|-----------|
| `.zst` download (large recent month) | ~50 GB |
| Active JSONL chunk | ~1.6 GB |
| Parquet shards (before commit) | ~4 GB (20 shards × 200 MB) |
| **Total peak** | **~56 GB** |
| After cleanup | 0 (deleted after commit) |
| Minimum free recommended | 80 GB |

Set `--min-free-gb 80` on server 2 if disk is tight.
