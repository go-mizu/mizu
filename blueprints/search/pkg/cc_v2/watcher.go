package cc_v2

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// HFCommitter is the interface for committing files to HuggingFace.
// Injected from CLI layer since it depends on hf_commit.py + uv.
type HFCommitter interface {
	CreateRepo(ctx context.Context, repoID string, private bool) error
	Commit(ctx context.Context, repoID, branch, message string, ops []HFOp) (commitURL string, err error)
	DownloadFile(ctx context.Context, repoID, path string) ([]byte, error)
}

// HFOp describes a file operation for an HF commit.
type HFOp struct {
	LocalPath  string
	PathInRepo string
}

// RepoFileWriter generates repo metadata files (README.md, LICENSE).
// Injected from CLI layer so v2 reuses the exact same README as v1.
type RepoFileWriter func(repoRoot, crawlID, statsCSV string) error

// Watcher polls the parquet directory for new files and commits them to HuggingFace.
// It is the ONLY component that deletes files (after successful HF commit).
type Watcher struct {
	cfg           WatcherConfig
	store         Store
	log           *Logger
	hf            HFCommitter
	writeRepoFn   RepoFileWriter // injected from CLI to reuse v1 README
	parquetDir    string
	repoRoot      string
	committed     map[int]bool
	commitNum     int
	lastCommit    time.Time
}

// NewWatcher creates a watcher.
func NewWatcher(cfg WatcherConfig, store Store, hf HFCommitter, writeRepoFn RepoFileWriter) *Watcher {
	return &Watcher{
		cfg:         cfg,
		store:       store,
		log:         NewLogger("watcher", store),
		hf:          hf,
		writeRepoFn: writeRepoFn,
		parquetDir:  filepath.Join(cfg.DataDir, "parquet"),
		repoRoot:    cfg.RepoRoot,
		committed:   make(map[int]bool),
	}
}

// Run starts the watcher loop.
func (w *Watcher) Run(ctx context.Context) error {
	// Defaults.
	if w.cfg.PollInterval == 0 {
		w.cfg.PollInterval = 10 * time.Second
	}
	if w.cfg.CommitInterval == 0 {
		w.cfg.CommitInterval = 90 * time.Second
	}
	if w.cfg.MaxBatch == 0 {
		w.cfg.MaxBatch = 30
	}
	if w.cfg.ChartsEvery == 0 {
		w.cfg.ChartsEvery = 60 * time.Minute
	}

	// Ensure dirs.
	for _, dir := range []string{w.parquetDir, w.repoRoot} {
		os.MkdirAll(dir, 0o755)
	}

	// Create HF repo if needed.
	if err := w.hf.CreateRepo(ctx, w.cfg.RepoID, w.cfg.Private); err != nil {
		w.log.Warn("create repo", "err", err)
	}

	// Load committed set from Redis.
	if rs := w.store.CommittedSet(ctx); rs != nil {
		w.committed = rs
	}

	// Sync stats.csv from HF for committed set baseline.
	w.syncStatsFromHF(ctx)

	w.log.PrintBanner("Watcher", map[string]string{
		"Crawl":    w.cfg.CrawlID,
		"Watch":    w.parquetDir,
		"HF repo":  w.cfg.RepoID,
		"Poll":     w.cfg.PollInterval.String(),
		"Commit":   fmt.Sprintf("min %s (≤%d/hr)", w.cfg.CommitInterval, int(time.Hour/w.cfg.CommitInterval)),
		"Redis":    fmt.Sprintf("%v", w.store.Available()),
		"Committed": fmt.Sprintf("%d", len(w.committed)),
	})

	// Flush once on startup (handle leftovers).
	w.flush(ctx)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.flush(ctx); err != nil {
				w.log.Error("flush", "err", err)
			}
		}
	}
}

func (w *Watcher) flush(ctx context.Context) error {
	// 1. Scan parquet dir for new files.
	pending := w.scanParquetDir()
	if len(pending) == 0 {
		return nil
	}

	// 2. Rate limit check.
	if !w.lastCommit.IsZero() {
		elapsed := time.Since(w.lastCommit)
		if elapsed < w.cfg.CommitInterval {
			return nil // wait quietly; scheduler shows pending count
		}
	}

	// 3. Cap batch size.
	if len(pending) > w.cfg.MaxBatch {
		w.log.Info("batch capped", "total", len(pending), "batch", w.cfg.MaxBatch)
		pending = pending[:w.cfg.MaxBatch]
	}

	totalBytes := int64(0)
	for _, f := range pending {
		totalBytes += f.Size
	}
	fmt.Fprintf(os.Stderr, "  commit %d shards (%s) ...\n", len(pending), FmtBytes(totalBytes))

	// 4. Write stats.csv with stats from all pending shards.
	t0 := time.Now()
	w.updateStatsCSV(ctx, pending)

	// 5. Generate README + LICENSE.
	w.writeRepoFiles()
	tPrep := time.Since(t0)

	// 6. Build commit ops.
	commitMsg := w.commitMessage(pending)
	ops := w.buildOps(pending)
	if len(ops) == 0 {
		w.log.Warn("no ops remain (all files vanished)")
		return nil
	}

	// 7. Commit with retry.
	tUpload := time.Now()
	commitURL, err := w.commitWithRetry(ctx, commitMsg, ops)
	if err != nil {
		return err
	}
	uploadDur := time.Since(tUpload)
	w.lastCommit = time.Now()
	fmt.Fprintf(os.Stderr, "  committed #%d  %d shards  prep=%s upload=%s  total=%s\n",
		w.commitNum+1, len(pending), tPrep.Round(time.Millisecond),
		uploadDur.Round(time.Second), time.Since(t0).Round(time.Second))

	// 8. Post-commit: mark committed, delete locals.
	for _, f := range pending {
		w.store.MarkCommitted(ctx, f.Idx)
		w.store.RecordEvent(ctx, "committed") // per-shard for accurate rate tracking
		w.committed[f.Idx] = true

		// Delete parquet + meta.
		os.Remove(f.Path)
		os.Remove(f.MetaPath)

		// Delete raw WARC (the ONLY place WARCs are ever deleted).
		if warcPath := w.store.GetWARCPath(ctx, f.Idx); warcPath != "" {
			os.Remove(warcPath)
		}
		// Also clean up warc_path sidecar (file store fallback).
		os.Remove(filepath.Join(w.parquetDir, f.Shard+".warc_path"))
	}

	// 9. Update watcher status.
	w.commitNum++
	status := WatcherStatus{
		CommitNum:      w.commitNum,
		Message:        commitMsg,
		CommitURL:      commitURL,
		ShardsInCommit: len(pending),
		TotalCommitted: len(w.committed),
		Timestamp:      time.Now(),
	}
	w.store.SetWatcherStatus(ctx, status)

	// Write status to file for scheduler.
	statusJSON, _ := json.Marshal(status)
	os.WriteFile(filepath.Join(w.repoRoot, "watcher_status.json"), statusJSON, 0o644)

	w.log.Info("published", "url", commitURL, "shards", len(pending),
		"total", len(w.committed))
	return nil
}

func (w *Watcher) scanParquetDir() []ParquetFile {
	entries, err := os.ReadDir(w.parquetDir)
	if err != nil {
		return nil
	}
	var result []ParquetFile
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".parquet") {
			continue
		}
		// Skip .parquet.tmp (in-progress writes).
		if strings.HasSuffix(name, ".tmp") {
			continue
		}
		shard := strings.TrimSuffix(name, ".parquet")
		idx, err := strconv.Atoi(shard)
		if err != nil {
			continue
		}
		if w.committed[idx] {
			// Already committed — clean up leftover.
			os.Remove(filepath.Join(w.parquetDir, name))
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		// Skip corrupt parquets (< 100 bytes).
		if fi.Size() < 100 {
			w.log.Warn("corrupt parquet, deleting", "shard", idx, "size", fi.Size())
			os.Remove(filepath.Join(w.parquetDir, name))
			continue
		}
		result = append(result, ParquetFile{
			Idx:      idx,
			Shard:    shard,
			Path:     filepath.Join(w.parquetDir, name),
			MetaPath: filepath.Join(w.parquetDir, shard+".meta.json"),
			Size:     fi.Size(),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Idx < result[j].Idx })
	return result
}

func (w *Watcher) commitMessage(files []ParquetFile) string {
	switch len(files) {
	case 0:
		return fmt.Sprintf("Update README for %s", w.cfg.CrawlID)
	case 1:
		return fmt.Sprintf("Publish shard %s/%s", w.cfg.CrawlID, files[0].Shard)
	default:
		return fmt.Sprintf("Publish %d shards %s/%s–%s",
			len(files), w.cfg.CrawlID, files[0].Shard, files[len(files)-1].Shard)
	}
}

func (w *Watcher) buildOps(files []ParquetFile) []HFOp {
	var ops []HFOp

	// Always include repo metadata files.
	for _, meta := range []struct{ local, remote string }{
		{filepath.Join(w.repoRoot, "README.md"), "README.md"},
		{filepath.Join(w.repoRoot, "LICENSE"), "LICENSE"},
		{filepath.Join(w.repoRoot, "stats.csv"), "stats.csv"},
	} {
		if fileExists(meta.local) {
			ops = append(ops, HFOp{LocalPath: meta.local, PathInRepo: meta.remote})
		}
	}

	// Add parquet files (re-verify they still exist).
	for _, f := range files {
		if fileExists(f.Path) {
			remotePath := fmt.Sprintf("data/%s/%s.parquet", w.cfg.CrawlID, f.Shard)
			ops = append(ops, HFOp{LocalPath: f.Path, PathInRepo: remotePath})
		} else {
			w.log.Warn("file vanished before upload", "shard", f.Idx)
		}
	}
	return ops
}

func (w *Watcher) commitWithRetry(ctx context.Context, msg string, ops []HFOp) (string, error) {
	const maxAttempts = 5
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * 10 * time.Second
			w.log.Info("retry", "attempt", attempt+1, "backoff", backoff)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
			// Re-sync stats from HF to pick up other server's rows.
			w.syncStatsFromHF(ctx)
			w.writeRepoFiles()
		}
		url, err := w.hf.Commit(ctx, w.cfg.RepoID, "main", msg, ops)
		if err == nil {
			return url, nil
		}
		lastErr = err
		w.log.Warn("commit failed", "attempt", attempt+1, "err", err)
	}
	return "", fmt.Errorf("commit failed after %d attempts: %w", maxAttempts, lastErr)
}

func (w *Watcher) updateStatsCSV(ctx context.Context, files []ParquetFile) {
	csvPath := filepath.Join(w.repoRoot, "stats.csv")
	existing, _ := readStatsCSV(csvPath)

	for _, f := range files {
		meta := w.readMeta(f)
		existing = upsertStats(existing, StatsRow{
			CrawlID:    w.cfg.CrawlID,
			FileIdx:    f.Idx,
			Rows:       meta.Rows,
			HTMLBytes:  meta.HTMLBytes,
			MDBytes:    meta.MDBytes,
			PqBytes:    f.Size,
			CreatedAt:  time.Now().UTC().Format(time.RFC3339),
			DurDlS:     meta.DurDlS,
			DurPackS:   meta.DurPackS,
			PeakRSSMB:  meta.PeakRSSMB,
		})
	}

	writeStatsCSV(csvPath, existing)
}

func (w *Watcher) readMeta(f ParquetFile) ShardStats {
	data, err := os.ReadFile(f.MetaPath)
	if err != nil {
		return ShardStats{}
	}
	var s ShardStats
	json.Unmarshal(data, &s)
	return s
}

func (w *Watcher) syncStatsFromHF(ctx context.Context) {
	data, err := w.hf.DownloadFile(ctx, w.cfg.RepoID, "stats.csv")
	if err != nil || len(data) == 0 {
		return
	}
	csvPath := filepath.Join(w.repoRoot, "stats.csv")
	mergeStatsFromRemote(csvPath, data, w.cfg.CrawlID)

	// Refresh committed set from merged CSV.
	if rows, err := readStatsCSV(csvPath); err == nil {
		for _, r := range rows {
			if r.CrawlID == w.cfg.CrawlID {
				w.committed[r.FileIdx] = true
			}
		}
	}
}

func (w *Watcher) writeRepoFiles() {
	csvPath := filepath.Join(w.repoRoot, "stats.csv")
	if w.writeRepoFn != nil {
		// Use injected function (reuses v1 README generation).
		if err := w.writeRepoFn(w.repoRoot, w.cfg.CrawlID, csvPath); err != nil {
			w.log.Warn("write repo files", "err", err)
		}
	} else {
		// Fallback: simple README.
		rows, _ := readStatsCSV(csvPath)
		readme := generateREADME(w.cfg.CrawlID, rows)
		os.WriteFile(filepath.Join(w.repoRoot, "README.md"), []byte(readme), 0o644)
		os.WriteFile(filepath.Join(w.repoRoot, "LICENSE"), []byte(licenseText), 0o644)
	}
}
