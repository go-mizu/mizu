package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ccShardMeta is written by the pipeline alongside each parquet to pass timing info to the watcher.
type ccShardMeta struct {
	DurDownloadS int64 `json:"dur_download_s"`
	DurConvertS  int64 `json:"dur_convert_s"`
	DurExportS   int64 `json:"dur_export_s"`
}

// ccUncommittedParquet holds a parquet file not yet committed to HF.
type ccUncommittedParquet struct {
	shard      string
	fileIdx    int
	localPath  string
	remotePath string
}

// ccRunWatcher polls the parquet data directory for new .parquet files and commits them to
// HuggingFace. On startup it immediately flushes any leftover parquets from previous runs,
// then polls every pollInterval. Only one HF commit happens at a time (serialized).
// minCommitInterval enforces a floor between HF commits to stay under the 128 commits/hour
// rate limit (with 2 servers, use ≥57s; default 90s → 40/hour per server, 80 total).
// Charts + README are regenerated every chartsEvery duration and included in the next commit.
func ccRunWatcher(ctx context.Context, crawlID, repoRoot, repoID string, private bool,
	pollInterval, minCommitInterval, chartsEvery time.Duration) error {

	token := strings.TrimSpace(os.Getenv("HF_TOKEN"))
	if token == "" {
		return fmt.Errorf("HF_TOKEN is not set")
	}
	hf := newHFClient(token)

	statsCSV := ccStatsCSVPath(repoRoot)
	dataDir := filepath.Join(repoRoot, "data", crawlID)
	warcMdDir := ccDefaultWARCMdConfig(crawlID)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("CC Watcher: parquet folder → HuggingFace"))
	fmt.Println()
	fmt.Printf("  Crawl     %s\n", labelStyle.Render(crawlID))
	fmt.Printf("  Watch dir %s\n", labelStyle.Render(dataDir))
	fmt.Printf("  HF repo   %s\n", infoStyle.Render(repoID))
	fmt.Printf("  Poll      every %s\n", infoStyle.Render(pollInterval.String()))
	fmt.Printf("  Commit    min interval %s (≤%d/hour; HF limit 128/hour across all servers)\n",
		infoStyle.Render(minCommitInterval.String()),
		int(time.Hour/minCommitInterval),
	)
	if chartsEvery > 0 {
		fmt.Printf("  Charts    every %s\n", infoStyle.Render(chartsEvery.String()))
	}
	fmt.Println()

	// Ensure repo exists (failure is soft — repo may already exist).
	if err := hf.createDatasetRepo(ctx, repoID, private); err != nil {
		fmt.Printf("  [watcher] note: create repo: %v\n", err)
	}

	// Seed committed set: merge HF stats.csv into local so we see all servers' progress.
	fmt.Printf("  [watcher] syncing stats from HF...\n")
	ccMergeStatsFromHF(ctx, hf, repoID, statsCSV)
	committed := ccLoadCommittedSet(statsCSV, crawlID)
	fmt.Printf("  [watcher] %d shards already committed\n", len(committed))

	// Purge local files that are already committed (leftover from old pipeline runs).
	if n := ccPurgeCommittedLocals(crawlID, dataDir, warcMdDir, committed); n > 0 {
		fmt.Printf("  [watcher] purged %d already-committed local file(s)\n", n)
	}
	fmt.Println()

	// Seed lastChartTime from the newest chart PNG on disk so a restart doesn't
	// redundantly regenerate charts that were just produced by the previous run.
	lastChartTime := ccNewestChartTime(repoRoot)
	// lastCommitTime tracks when we last committed to HF so we can enforce minCommitInterval.
	// Seed to zero so the very first flush (leftover parquets) is never blocked.
	var lastCommitTime time.Time

	// Flush immediately on startup (handles leftovers from previous runs), then tick.
	flush := func() {
		if err := ccWatcherFlush(ctx, hf, crawlID, repoRoot, repoID, statsCSV, dataDir, warcMdDir,
			committed, &lastChartTime, chartsEvery, minCommitInterval, &lastCommitTime); err != nil {
			fmt.Printf("  [watcher] flush: %v\n", err)
		}
	}
	flush()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			flush()
		}
	}
}

// ccWatcherFlush finds uncommitted parquets, pushes them to HF, deletes local copies.
// Retries up to 3 times on commit error, re-merging stats from HF each attempt
// so that concurrent commits from two servers don't clobber each other's stats.csv rows.
// minCommitInterval enforces a minimum gap between commits to stay under HF's 128/hour limit.
func ccWatcherFlush(ctx context.Context, hf *hfClient, crawlID, repoRoot, repoID, statsCSV, dataDir, warcMdDir string,
	committed map[int]bool, lastChartTime *time.Time, chartsEvery time.Duration,
	minCommitInterval time.Duration, lastCommitTime *time.Time) error {

	newFiles := ccFindUncommittedParquets(dataDir, crawlID, committed)
	chartsStale := chartsEvery > 0 && time.Since(*lastChartTime) >= chartsEvery
	if len(newFiles) == 0 && !chartsStale {
		return nil
	}

	// Rate-limit: HF allows 128 commits/hour across all users of a token.
	// With 2 servers each running a watcher, cap at minCommitInterval per server.
	// Skip the commit this round — files accumulate and will be batched next time.
	if len(newFiles) > 0 && minCommitInterval > 0 && !lastCommitTime.IsZero() {
		if elapsed := time.Since(*lastCommitTime); elapsed < minCommitInterval {
			waitFor := (minCommitInterval - elapsed).Round(time.Second)
			fmt.Printf("  [watcher] rate-limit: next commit in %s (holding %d file(s) to batch)\n",
				waitFor, len(newFiles))
			return nil
		}
	}

	if len(newFiles) > 0 {
		fmt.Printf("  [watcher] %d new parquet(s) — committing to HF...\n", len(newFiles))
	}

	// Step 1: Write our stats rows first (local wins in later merge).
	for _, f := range newFiles {
		meta := ccReadShardMeta(dataDir, f.shard)
		rows, htmlBytes, mdBytes, _ := ccScanParquetStats(f.localPath)
		fi, _ := os.Stat(f.localPath)
		pqBytes := int64(0)
		if fi != nil {
			pqBytes = fi.Size()
		}
		_ = ccUpsertShardStats(statsCSV, ccShardStats{
			CrawlID:      crawlID,
			FileIdx:      f.fileIdx,
			Rows:         rows,
			HTMLBytes:    htmlBytes,
			MDBytes:      mdBytes,
			ParquetBytes: pqBytes,
			CreatedAt:    time.Now().UTC().Format(time.RFC3339),
			DurDownloadS: meta.DurDownloadS,
			DurConvertS:  meta.DurConvertS,
			DurExportS:   meta.DurExportS,
		})
	}

	// Step 2: Merge from HF AFTER writing local rows (merge keeps local wins,
	// so our rows survive; we also pick up the other server's latest rows).
	ccMergeStatsFromHF(ctx, hf, repoID, statsCSV)

	// Step 3: Regenerate README + charts with merged stats.
	if err := ccEnsurePublishRepoFiles(repoRoot, crawlID, statsCSV); err != nil {
		return fmt.Errorf("write repo files: %w", err)
	}
	var chartRelPaths []string
	if chartsStale {
		chartRelPaths = ccRunCharts(statsCSV, repoRoot, crawlID)
		if len(chartRelPaths) > 0 {
			fmt.Printf("  [watcher] regenerated %d chart(s)\n", len(chartRelPaths))
		}
	}

	// Build commit operations.
	shards := make([]string, len(newFiles))
	for i, f := range newFiles {
		shards[i] = f.shard
	}
	var commitMsg string
	switch {
	case len(newFiles) == 0:
		commitMsg = fmt.Sprintf("Update charts/README for %s", crawlID)
	case len(newFiles) == 1:
		commitMsg = fmt.Sprintf("Publish shard %s/%s", crawlID, newFiles[0].shard)
	default:
		commitMsg = fmt.Sprintf("Publish %d shards %s/%s–%s", len(newFiles), crawlID, shards[0], shards[len(shards)-1])
	}

	buildOps := func() []hfOperation {
		ops := []hfOperation{
			{LocalPath: filepath.Join(repoRoot, "README.md"), PathInRepo: "README.md"},
			{LocalPath: filepath.Join(repoRoot, "LICENSE"), PathInRepo: "LICENSE"},
			{LocalPath: statsCSV, PathInRepo: "stats.csv"},
		}
		for _, rel := range chartRelPaths {
			ops = append(ops, hfOperation{
				LocalPath:  filepath.Join(repoRoot, rel),
				PathInRepo: filepath.ToSlash(rel),
			})
		}
		for _, f := range newFiles {
			ops = append(ops, hfOperation{LocalPath: f.localPath, PathInRepo: f.remotePath})
		}
		return ops
	}

	// Step 4: Commit with retry — on failure re-merge stats so we don't lose
	// the other server's rows, then retry (handles transient HF errors and 429s).
	// On 429, sleep for the server-requested Retry-After duration (plus 30s buffer)
	// instead of the normal exponential backoff, to avoid hammering the rate limit.
	const maxAttempts = 5
	var (
		commitURL string
		elapsed   time.Duration
		commitErr error
	)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			var rlErr *HFRateLimitError
			if errors.As(commitErr, &rlErr) {
				wait := rlErr.RetryAfter + 30*time.Second
				if wait < 30*time.Second {
					wait = 30 * time.Second
				}
				fmt.Printf("  [watcher] 429 rate limited — sleeping %s before retry %d/%d\n",
					wait.Round(time.Second), attempt+1, maxAttempts)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(wait):
				}
			} else {
				backoff := time.Duration(attempt*attempt) * 10 * time.Second
				fmt.Printf("  [watcher] retrying in %s (attempt %d/%d)...\n", backoff, attempt+1, maxAttempts)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
				}
			}
			// Re-merge from HF so we include any commits from the other server
			// that happened while we were preparing our commit.
			ccMergeStatsFromHF(ctx, hf, repoID, statsCSV)
			if err := ccEnsurePublishRepoFiles(repoRoot, crawlID, statsCSV); err != nil {
				fmt.Printf("  [watcher] retry repo files: %v\n", err)
			}
		}
		t0 := time.Now()
		commitURL, commitErr = hf.createCommit(ctx, repoID, "main", commitMsg, buildOps())
		elapsed = time.Since(t0)
		if commitErr == nil {
			break
		}
		fmt.Printf("  [watcher] commit error (attempt %d): %v\n", attempt+1, commitErr)
	}
	if commitErr != nil {
		return fmt.Errorf("HF commit after %d attempts: %w", maxAttempts, commitErr)
	}
	*lastCommitTime = time.Now()
	if len(chartRelPaths) > 0 {
		*lastChartTime = time.Now()
	}

	// Step 5: Update publish timing, delete all local intermediates, mark committed.
	durPublishS := int64(elapsed.Seconds())
	if len(newFiles) > 1 {
		durPublishS = int64(elapsed.Seconds()) / int64(len(newFiles))
	}
	for _, f := range newFiles {
		if all, _ := ccReadStatsCSV(statsCSV); all != nil {
			for _, s := range all {
				if s.CrawlID == crawlID && s.FileIdx == f.fileIdx {
					s.DurPublishS = durPublishS
					_ = ccUpsertShardStats(statsCSV, s)
					break
				}
			}
		}
		_ = os.Remove(f.localPath)
		_ = os.Remove(filepath.Join(dataDir, f.shard+".meta"))
		// Also delete intermediate pipeline files to keep disk free.
		_ = os.Remove(filepath.Join(warcMdDir, f.shard+".md.warc.gz"))
		_ = os.Remove(filepath.Join(warcMdDir, f.shard+".meta.duckdb"))
		if rawPath := ccFindRawWARC(crawlID, f.fileIdx); rawPath != "" {
			_ = os.Remove(rawPath)
		}
		// Delete sidecar regardless (cleanup is complete for this shard).
		_ = os.Remove(filepath.Join(warcMdDir, f.shard+".warc.path"))
		committed[f.fileIdx] = true
		fmt.Printf("  [watcher] deleted local %s.parquet\n", f.shard)
	}

	fmt.Printf("  [watcher] %s  (%d shards, %s)\n",
		successStyle.Render("published "+commitURL),
		len(newFiles), elapsed.Round(time.Second),
	)
	// Run brute-force WARC cleanup after each commit to catch any orphaned raw WARCs
	// from finished/crashed pipeline sessions (Pass 3 in ccPurgeCommittedLocals).
	if n := ccPurgeCommittedLocals(crawlID, dataDir, warcMdDir, committed); n > 0 {
		fmt.Printf("  [watcher] purged %d orphaned intermediate file(s)\n", n)
	}
	fmt.Println()
	return nil
}

// ccLoadCommittedSet returns the set of file indices already committed for crawlID.
func ccLoadCommittedSet(statsCSV, crawlID string) map[int]bool {
	all, _ := ccReadStatsCSV(statsCSV)
	set := make(map[int]bool, len(all))
	for _, s := range all {
		if s.CrawlID == crawlID {
			set[s.FileIdx] = true
		}
	}
	return set
}

// ccFindUncommittedParquets scans dataDir for .parquet files whose index is not in committed.
// .parquet.tmp files (in-progress writes) are ignored.
func ccFindUncommittedParquets(dataDir, crawlID string, committed map[int]bool) []ccUncommittedParquet {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil
	}
	var result []ccUncommittedParquet
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".parquet") {
			continue // skip .parquet.tmp and non-parquet files
		}
		shard := strings.TrimSuffix(name, ".parquet")
		idx, err := strconv.Atoi(shard)
		if err != nil {
			continue
		}
		if committed[idx] {
			continue
		}
		localPath := filepath.Join(dataDir, name)
		remotePath := filepath.ToSlash(filepath.Join("data", crawlID, name))
		result = append(result, ccUncommittedParquet{
			shard: shard, fileIdx: idx,
			localPath: localPath, remotePath: remotePath,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].fileIdx < result[j].fileIdx })
	return result
}

// ccPurgeCommittedLocals deletes all intermediate files for already-committed shards:
//   - {dataDir}/{shard}.parquet + .meta  (watcher output)
//   - {warcMdDir}/{shard}.md.warc.gz + .meta.duckdb  (pack output)
//   - raw .warc.gz from ccFindRawWARC    (download output)
//
// It uses two passes so orphaned raw WARCs are caught even when the parquet was
// already deleted by a previous watcher run:
//
//  1. Scan dataDir for leftover .parquet files and delete the full chain.
//  2. Scan the raw warc/ directory for any *-XXXXX.warc.gz whose index is committed
//     (catches files whose parquet + md.warc.gz were already cleaned).
//
// Returns the total number of files deleted.
func ccPurgeCommittedLocals(crawlID, dataDir, warcMdDir string, committed map[int]bool) int {
	n := 0

	// Pass 1: parquet-led sweep (same as before).
	if entries, err := os.ReadDir(dataDir); err == nil {
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".parquet") {
				continue
			}
			shard := strings.TrimSuffix(name, ".parquet")
			idx, err := strconv.Atoi(shard)
			if err != nil || !committed[idx] {
				continue
			}
			if rmErr := os.Remove(filepath.Join(dataDir, name)); rmErr == nil {
				n++
			}
			_ = os.Remove(filepath.Join(dataDir, shard+".meta"))
			mdWARC := filepath.Join(warcMdDir, shard+".md.warc.gz")
			if rmErr := os.Remove(mdWARC); rmErr == nil {
				n++
			}
			_ = os.Remove(filepath.Join(warcMdDir, shard+".meta.duckdb"))
			if rawPath := ccFindRawWARC(crawlID, idx); rawPath != "" {
				if rmErr := os.Remove(rawPath); rmErr == nil {
					n++
				}
			}
		}
	}

	// Pass 2: sidecar sweep — catch orphaned raw WARCs via .warc.path sidecars written
	// by runCCWarcPack. CC filenames (e.g. CC-MAIN-...-00044.warc.gz) do not contain
	// the file index, so filename-based matching doesn't work. The sidecar records the
	// actual path for each shard index, making cleanup reliable.
	home, _ := os.UserHomeDir()
	if entries, err := os.ReadDir(warcMdDir); err == nil {
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".warc.path") {
				continue
			}
			shard := strings.TrimSuffix(name, ".warc.path")
			idx, err := strconv.Atoi(shard)
			if err != nil || !committed[idx] {
				continue
			}
			sidecarPath := filepath.Join(warcMdDir, name)
			data, err := os.ReadFile(sidecarPath)
			if err != nil {
				continue
			}
			rawPath := strings.TrimSpace(string(data))
			if rawPath != "" {
				if rmErr := os.Remove(rawPath); rmErr == nil {
					n++
				}
			}
			_ = os.Remove(sidecarPath)
		}
	}

	// Pass 3: brute-force raw WARC dir — delete any .warc.gz not currently open by
	// any process and older than 10 minutes (safely orphaned from crashed pipelines).
	warcDir := filepath.Join(home, "data", "common-crawl", crawlID, "warc")
	openFiles := ccOpenFiles()
	if entries, err := os.ReadDir(warcDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".warc.gz") {
				continue
			}
			fullPath := filepath.Join(warcDir, e.Name())
			if openFiles[fullPath] {
				continue // currently in use by a pipeline
			}
			fi, err := e.Info()
			if err != nil || time.Since(fi.ModTime()) < 10*time.Minute {
				continue // too recent — may still be downloading
			}
			if rmErr := os.Remove(fullPath); rmErr == nil {
				n++
			}
		}
	}

	return n
}

// ccOpenFiles returns a set of absolute file paths currently held open by any process.
// Uses /proc/*/fd symlinks (Linux only); returns empty map on other platforms.
func ccOpenFiles() map[string]bool {
	open := make(map[string]bool)
	procs, err := os.ReadDir("/proc")
	if err != nil {
		return open
	}
	for _, pe := range procs {
		if !pe.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(pe.Name()); err != nil {
			continue
		}
		fdDir := filepath.Join("/proc", pe.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		for _, fd := range fds {
			target, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err == nil && target != "" {
				open[target] = true
			}
		}
	}
	return open
}

// ccNewestChartTime returns the mtime of the most recently modified PNG in repoRoot/charts/,
// or the zero time if no charts exist. Used to seed lastChartTime on startup so a restart
// doesn't regenerate charts that are still fresh.
func ccNewestChartTime(repoRoot string) time.Time {
	entries, err := os.ReadDir(filepath.Join(repoRoot, "charts"))
	if err != nil {
		return time.Time{}
	}
	var newest time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".png") {
			continue
		}
		if fi, err := e.Info(); err == nil && fi.ModTime().After(newest) {
			newest = fi.ModTime()
		}
	}
	return newest
}

// ccReadShardMeta reads the .meta sidecar file written by the pipeline for timing info.
// Returns zero-value if absent.
func ccReadShardMeta(dataDir, shard string) ccShardMeta {
	data, err := os.ReadFile(filepath.Join(dataDir, shard+".meta"))
	if err != nil {
		return ccShardMeta{}
	}
	var m ccShardMeta
	_ = json.Unmarshal(data, &m)
	return m
}
