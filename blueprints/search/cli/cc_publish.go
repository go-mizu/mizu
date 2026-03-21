package cli

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	warcmd "github.com/go-mizu/mizu/blueprints/search/pkg/warc_md"
	"github.com/spf13/cobra"
)

type ccPublishUploadFile struct {
	LocalPath  string
	PathInRepo string
}

func newCCPublish() *cobra.Command {
	var (
		crawlID        string
		fileIdx        string
		repoRoot       string
		repoID         string
		republish      bool
		private        bool
		pipeline       bool
		watch          bool
		schedule       bool
		list           bool
		gaps           bool
		cleanup        bool
		lightConvert   bool
		skipErrors     bool
		watchInterval   int
		commitInterval  int
		chartsEvery     int
		schedStart      int
		schedEnd        int
		schedMaxSess    int
		schedChunk      int
		schedDonePct    int
		schedStall      int
		ramPerSession   float64
	)

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish exported Common Crawl parquet shards to Hugging Face",
		Long: `Publish $HOME/data/common-crawl/{crawl}/export/repo to a Hugging Face dataset repo.

With --pipeline:  download, pack, export shards as .parquet files (no HF push).
With --watch:     watch the parquet folder and push new files to HF in real-time.
With --schedule:  manage pipeline screen sessions across a file index range.
                  Starts/restarts sessions, detects stalls, self-heals on crash.
Run --watch and --schedule as separate processes; use multiple --pipeline workers.`,
		Example: `  # Watch-only (one per server):
  search cc publish --watch

  # Scheduler (self-healing, adaptive resource management):
  search cc publish --schedule --start 0    --end 4999   # server1
  search cc publish --schedule --start 5000 --end 9999   # server2

  # Pipeline-only (multiple per server, no HF push):
  search cc publish --pipeline --file 68-300 --cleanup --skip-errors

  # Manual one-shot publish (legacy):
  search cc publish --file 0 --republish`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCPublish(cmd.Context(), crawlID, fileIdx, repoRoot, repoID,
				republish, private, pipeline, watch, schedule, list, gaps, cleanup, lightConvert, skipErrors,
				watchInterval, commitInterval, chartsEvery,
				schedStart, schedEnd, schedMaxSess, schedChunk, schedDonePct, schedStall, ramPerSession)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "all", "File index, range (0-9), comma-separated list, or all (pipeline mode)")
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "Local export repo root (default: $HOME/data/common-crawl/{crawl}/export/repo)")
	cmd.Flags().StringVar(&repoID, "repo", "open-index/open-markdown", "Hugging Face dataset repo ID")
	cmd.Flags().BoolVar(&republish, "republish", false, "Upload even if the remote path already exists (manual mode only)")
	cmd.Flags().BoolVar(&private, "private", false, "Create the Hugging Face dataset repo as private")
	cmd.Flags().BoolVar(&pipeline, "pipeline", false, "Download, pack, export shards (writes parquets locally; use --watch to push)")
	cmd.Flags().BoolVar(&watch, "watch", false, "Watch parquet folder and push new files to HuggingFace in real-time")
	cmd.Flags().BoolVar(&schedule, "schedule", false, "Manage pipeline screen sessions across a file index range (self-healing scheduler)")
	cmd.Flags().BoolVar(&list, "list", false, "List committed shards as ranges (from stats.csv / HF)")
	cmd.Flags().BoolVar(&gaps, "gaps", false, "Detect and display gap shards; with --schedule: backfill gaps via scheduler; with --pipeline: process gaps directly")
	cmd.Flags().BoolVar(&cleanup, "cleanup", false, "Delete raw .warc.gz after packing (--pipeline only)")
	cmd.Flags().BoolVar(&lightConvert, "light", true, "Use lightweight HTML→Markdown converter (~10x faster, --no-light for trafilatura)")
	cmd.Flags().BoolVar(&skipErrors, "skip-errors", false, "Skip shards that fail pack/export instead of aborting (--pipeline only)")
	cmd.Flags().IntVar(&watchInterval, "watch-interval", 10, "Watcher poll interval in seconds (--watch only)")
	cmd.Flags().IntVar(&commitInterval, "commit-interval", 120, "Minimum seconds between HF commits (--watch only). HF allows 128 commits/hour shared across all repos/tokens; default 120s → ≤30/hour per server, leaving headroom for arctic/HN/other repos")
	cmd.Flags().IntVar(&chartsEvery, "charts-every", 60, "Regenerate charts every N minutes (--watch only, 0=disable)")
	cmd.Flags().IntVar(&schedStart, "start", 0, "First file index in range (--schedule/--gaps)")
	cmd.Flags().IntVar(&schedEnd, "end", 9999, "Last file index in range (--schedule/--gaps)")
	cmd.Flags().IntVar(&schedMaxSess, "max-sessions", 0, "Max concurrent screen sessions (0=auto-detect from hardware; --schedule only)")
	cmd.Flags().IntVar(&schedChunk, "chunk-size", 50, "Gap indices per screen session chunk (--schedule/--gaps; smaller = faster cycling, natural memory release)")
	cmd.Flags().IntVar(&schedDonePct, "done-pct", 95, "% of shards committed before chunk is considered done (--schedule/--gaps)")
	cmd.Flags().IntVar(&schedStall, "stall-rounds", 40, "Rounds with no new commits before killing a stalled session (--schedule only; ~30 min at 45s/round)")
	cmd.Flags().Float64Var(&ramPerSession, "ram-per-session", 0, "GB of RAM budgeted per pipeline session (0=default 1.2; --schedule only)")
	return cmd
}

func runCCPublish(ctx context.Context, crawlID, fileIdx, repoRoot, repoID string,
	republish, private, pipeline, watch, schedule, list, gaps, cleanup, lightConvert, skipErrors bool,
	watchInterval, commitInterval, chartsEvery int,
	schedStart, schedEnd, schedMaxSess, schedChunk, schedDonePct, schedStall int, ramPerSession float64) error {

	resolvedID, note, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedID
	if note != "" {
		ccPrintDefaultCrawlResolution(crawlID, note)
	}

	if repoRoot == "" {
		repoRoot = ccDefaultExportRepoRoot(crawlID)
	}

	// ── List mode: show committed shards as ranges ────────────────────────────
	if list {
		return ccListCommittedShards(ctx, crawlID, repoRoot, repoID)
	}

	// ── Gap mode: detect/display/backfill uncommitted shards ──────────────────
	if gaps {
		statsCSV := ccStatsCSVPath(repoRoot)
		token := strings.TrimSpace(os.Getenv("HF_TOKEN"))
		if token != "" {
			hf := newHFClient(token)
			ccMergeStatsFromHF(ctx, hf, repoID, statsCSV)
		}
		gapIndices := ccComputeGapIndices(statsCSV, crawlID, schedStart, schedEnd)

		if schedule {
			// Route large gap sets through the scheduler using gap-specific chunks.
			if len(gapIndices) == 0 {
				fmt.Printf("  No gaps found in %d–%d for %s\n", schedStart, schedEnd, crawlID)
				return nil
			}
			fmt.Printf("  Gap backfill: %d uncommitted shards in %d–%d → scheduler\n",
				len(gapIndices), schedStart, schedEnd)
			cfg := ccScheduleConfig{
				CrawlID:       crawlID,
				RepoRoot:      repoRoot,
				Start:         schedStart,
				End:           schedEnd,
				MaxSessions:   schedMaxSess,
				RAMPerSession: ramPerSession,
				ChunkSize:     schedChunk,
				DonePct:       schedDonePct,
				StallRounds:   schedStall,
				GapIndices:    gapIndices,
			}
			return runCCScheduleLoop(ctx, cfg)
		}

		if pipeline {
			// Build comma-separated file selector from gap indices and run pipeline.
			if len(gapIndices) == 0 {
				fmt.Printf("  No gaps found in %d–%d for %s\n", schedStart, schedEnd, crawlID)
				return nil
			}
			parts := make([]string, len(gapIndices))
			for i, g := range gapIndices {
				parts[i] = strconv.Itoa(g)
			}
			fmt.Printf("  Gap pipeline: %d uncommitted shards in %d–%d\n",
				len(gapIndices), schedStart, schedEnd)
			return ccRunPipeline(ctx, crawlID, strings.Join(parts, ","), repoRoot, cleanup, lightConvert, skipErrors)
		}

		// Default: print gap analysis only.
		return ccPrintGaps(crawlID, statsCSV, schedStart, schedEnd, gapIndices)
	}

	// ── Watch mode: poll parquet folder → push to HF ─────────────────────────
	if watch {
		if watchInterval < 1 {
			watchInterval = 15
		}
		if commitInterval < 1 {
			commitInterval = 90
		}
		return ccRunWatcher(ctx, crawlID, repoRoot, repoID, private,
			time.Duration(watchInterval)*time.Second,
			time.Duration(commitInterval)*time.Second,
			time.Duration(chartsEvery)*time.Minute,
		)
	}

	// ── Schedule mode: manage pipeline screen sessions (self-healing) ────────
	if schedule {
		cfg := ccScheduleConfig{
			CrawlID:       crawlID,
			RepoRoot:      repoRoot,
			Start:         schedStart,
			End:           schedEnd,
			MaxSessions:   schedMaxSess,
			RAMPerSession: ramPerSession,
			ChunkSize:     schedChunk,
			DonePct:       schedDonePct,
			StallRounds:   schedStall,
		}
		return runCCScheduleLoop(ctx, cfg)
	}

	// ── Pipeline mode: download → pack → export (no HF push) ─────────────────
	if pipeline {
		return ccRunPipeline(ctx, crawlID, fileIdx, repoRoot, cleanup, lightConvert, skipErrors)
	}

	// ── Collect stats from all exported parquet files ───────────────────────
	statsCSV := ccStatsCSVPath(repoRoot)
	if err := ccRefreshStats(crawlID, repoRoot, statsCSV); err != nil {
		return fmt.Errorf("refresh stats: %w", err)
	}

	// ── Write README + LICENSE with real numbers ─────────────────────────────
	if err := ccEnsurePublishRepoFiles(repoRoot, crawlID, statsCSV); err != nil {
		return err
	}

	files, err := ccResolvePublishUploadFiles(repoRoot, crawlID, fileIdx)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no local parquet files selected under %s", filepath.Join(repoRoot, "data"))
	}

	token := strings.TrimSpace(os.Getenv("HF_TOKEN"))
	if token == "" {
		return fmt.Errorf("HF_TOKEN environment variable is not set")
	}

	hf := newHFClient(token)

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Common Crawl Publish"))
	fmt.Println()
	fmt.Printf("  Crawl      %s\n", labelStyle.Render(crawlID))
	fmt.Printf("  Repo root  %s\n", labelStyle.Render(repoRoot))
	fmt.Printf("  HF repo    %s\n", infoStyle.Render(repoID))
	// Show processed shards from stats
	if allStats, err := ccReadStatsCSV(statsCSV); err == nil {
		t := ccComputeTotals(allStats, crawlID)
		if t.Shards > 0 {
			shardList := make([]string, 0, t.Shards)
			for _, s := range allStats {
				if s.CrawlID == crawlID {
					shardList = append(shardList, fmt.Sprintf("%05d", s.FileIdx))
				}
			}
			fmt.Printf("  Processed  %s shards: %s\n",
				infoStyle.Render(ccFmtInt64(int64(t.Shards))),
				labelStyle.Render(strings.Join(shardList, ", ")))
		}
	}
	fmt.Println()

	// Create repo if needed
	if err := hf.createDatasetRepo(ctx, repoID, private); err != nil {
		return fmt.Errorf("create repo: %w", err)
	}

	// Always upload README + LICENSE + stats.csv + selected parquet
	allFiles := append([]ccPublishUploadFile{
		{LocalPath: filepath.Join(repoRoot, "README.md"), PathInRepo: "README.md"},
		{LocalPath: filepath.Join(repoRoot, "LICENSE"), PathInRepo: "LICENSE"},
		{LocalPath: statsCSV, PathInRepo: "stats.csv"},
	}, files...)

	var ops []hfOperation
	var skipped []string
	if !republish {
		paths := make([]string, len(allFiles))
		for i, f := range allFiles {
			paths[i] = f.PathInRepo
		}
		existing, err := hf.pathsExist(ctx, repoID, paths)
		if err != nil {
			return fmt.Errorf("checking existing files: %w", err)
		}
		for _, f := range allFiles {
			if f.PathInRepo == "README.md" || f.PathInRepo == "LICENSE" || f.PathInRepo == "stats.csv" {
				// Always re-upload metadata files to keep them current
				ops = append(ops, hfOperation{LocalPath: f.LocalPath, PathInRepo: f.PathInRepo})
			} else if existing[f.PathInRepo] {
				skipped = append(skipped, f.PathInRepo)
			} else {
				ops = append(ops, hfOperation{LocalPath: f.LocalPath, PathInRepo: f.PathInRepo})
			}
		}
	} else {
		for _, f := range allFiles {
			ops = append(ops, hfOperation{LocalPath: f.LocalPath, PathInRepo: f.PathInRepo})
		}
	}

	if len(ops) == 0 {
		fmt.Printf("  Uploaded   %s\n", infoStyle.Render("0"))
		fmt.Printf("  Skipped    %s\n", warningStyle.Render(ccFmtInt64(int64(len(skipped)))))
		return nil
	}

	commitMsg := ccPublishCommitMessage(fileIdx, files)
	commitURL, err := hf.createCommit(ctx, repoID, "main", commitMsg, ops)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	fmt.Printf("  Uploaded   %s\n", successStyle.Render(ccFmtInt64(int64(len(ops)))))
	if len(skipped) > 0 {
		fmt.Printf("  Skipped    %s\n", warningStyle.Render(ccFmtInt64(int64(len(skipped)))))
	}
	if commitURL != "" {
		fmt.Printf("  Commit     %s\n", labelStyle.Render(commitURL))
	}

	// Print cumulative stats
	if allStats, err := ccReadStatsCSV(statsCSV); err == nil && len(allStats) > 0 {
		t := ccComputeTotals(allStats, crawlID)
		if t.Shards > 0 {
			fmt.Println()
			fmt.Printf("  ── Cumulative stats (%s) ──\n", crawlID)
			fmt.Printf("  Shards     %s\n", infoStyle.Render(ccFmtInt64(int64(t.Shards))))
			fmt.Printf("  Documents  %s\n", infoStyle.Render(ccFmtInt64(t.Rows)))
			fmt.Printf("  HTML       %s\n", infoStyle.Render(ccFmtBytes(t.HTMLBytes)))
			fmt.Printf("  Markdown   %s  (-%s%%)\n",
				infoStyle.Render(ccFmtBytes(t.MDBytes)),
				infoStyle.Render(ccPctReduction(t.HTMLBytes, t.MDBytes)))
			fmt.Printf("  Parquet    %s\n", infoStyle.Render(ccFmtBytes(t.ParquetBytes)))
		}
	}
	return nil
}

// ccRunPipeline downloads, packs, and exports shards to local .parquet files.
// It does NOT push to HuggingFace — run `--watch` in a separate session for that.
// Parquets are written atomically (via .parquet.tmp → rename) so the watcher
// only sees fully-written files. A .meta sidecar carries timing info to the watcher.
func ccRunPipeline(ctx context.Context, crawlID, fileIdx, repoRoot string, cleanup, lightConvert, skipErrors bool) error {
	indices, err := ccParseOpenFileSelector(fileIdx)
	if err != nil {
		return fmt.Errorf("--file: %w", err)
	}

	warcMdDir := ccDefaultWARCMdConfig(crawlID)
	dataDir := filepath.Join(repoRoot, "data", crawlID)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	statsCSV := ccStatsCSVPath(repoRoot)
	skippedCSV := ccSkippedCSVPath(repoRoot)

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("CC Pipeline: download → pack → export"))
	fmt.Println()
	fmt.Printf("  Crawl     %s\n", labelStyle.Render(crawlID))
	fmt.Printf("  Files     %s\n", infoStyle.Render(strconv.Itoa(len(indices))))
	fmt.Printf("  Output    %s\n", labelStyle.Render(dataDir))
	fmt.Println()

	// Load committed set once at startup (watcher maintains this via stats.csv).
	committed := ccLoadCommittedSet(statsCSV, crawlID)

	// Prefetch: pack the next shard in background while exporting the current.
	// This overlaps network I/O (WARC download) with CPU-bound work (export),
	// roughly doubling per-session throughput.
	type prefetchResult struct {
		idx          int
		err          error
		durDownloadS int64
		durConvertS  int64
	}
	var prefetchCh chan prefetchResult

	// cancelPrefetch cancels any in-flight prefetch goroutine.
	var prefetchCancel context.CancelFunc
	defer func() {
		if prefetchCancel != nil {
			prefetchCancel()
		}
	}()

	// startPrefetch kicks off pack for the next shard in background.
	startPrefetch := func(nextIdx int) {
		nextShard := fmt.Sprintf("%05d", nextIdx)
		nextMdWARC := filepath.Join(warcMdDir, nextShard+".md.warc.gz")
		nextParquet := filepath.Join(dataDir, nextShard+".parquet")

		// Don't prefetch if already done.
		if committed[nextIdx] || fileExists(nextParquet) || fileExists(nextMdWARC) {
			return
		}

		prefetchCh = make(chan prefetchResult, 1)
		var pfCtx context.Context
		pfCtx, prefetchCancel = context.WithCancel(ctx)

		go func() {
			rawExists := ccFindRawWARC(crawlID, nextIdx) != ""
			t0 := time.Now()
			packErr := runCCWarcPack(pfCtx, crawlID, strconv.Itoa(nextIdx), -1, -1, 0, false, false, lightConvert, 200, "text/html", 512*1024, nextShard)
			elapsed := int64(time.Since(t0).Seconds())
			r := prefetchResult{idx: nextIdx, err: packErr}
			if rawExists {
				r.durConvertS = elapsed
			} else {
				r.durDownloadS = elapsed
			}
			prefetchCh <- r
		}()
	}

	for i, idx := range indices {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		shard := fmt.Sprintf("%05d", idx)
		mdWARCPath := filepath.Join(warcMdDir, shard+".md.warc.gz")
		parquetPath := filepath.Join(dataDir, shard+".parquet")
		tmpPath := parquetPath + ".tmp"

		fmt.Printf("  ── [%d/%d] %s ──\n", i+1, len(indices), labelStyle.Render(shard))

		// Skip: already committed to HF (watcher updated stats.csv).
		if committed[idx] {
			fmt.Printf("  [%s] already committed, skipping\n", labelStyle.Render(shard))
			fmt.Println()
			continue
		}

		// Skip: parquet already exported and waiting for watcher to push.
		if fileExists(parquetPath) {
			fmt.Printf("  [%s] parquet ready, waiting for watcher\n", labelStyle.Render(shard))
			fmt.Println()
			continue
		}

		// Clean up orphaned .tmp from a previous crash.
		if fileExists(tmpPath) {
			_ = os.Remove(tmpPath)
		}

		var durDownloadS, durConvertS, durExportS int64

		// Check if prefetch already packed this shard.
		prefetched := false
		if prefetchCh != nil {
			select {
			case pf := <-prefetchCh:
				if pf.idx == idx && pf.err == nil {
					prefetched = true
					durDownloadS = pf.durDownloadS
					durConvertS = pf.durConvertS
					fmt.Printf("  [%s] prefetched (pack done in background)\n", labelStyle.Render(shard))
				} else if pf.idx == idx && pf.err != nil {
					// Prefetch failed — fall through to normal pack.
					fmt.Printf("  [%s] prefetch failed: %v — retrying\n", labelStyle.Render(shard), pf.err)
				}
			default:
				// Prefetch not done yet or for different shard — ignore.
			}
			prefetchCh = nil
			prefetchCancel = nil
		}

		// Pack if md.warc.gz is missing and not prefetched.
		if !prefetched && !fileExists(mdWARCPath) {
			rawWARCExists := ccFindRawWARC(crawlID, idx) != ""
			fmt.Printf("  [%s] packing...\n", labelStyle.Render(shard))
			t0 := time.Now()
			if packErr := runCCWarcPack(ctx, crawlID, strconv.Itoa(idx), -1, -1, 0, false, false, lightConvert, 200, "text/html", 512*1024, shard); packErr != nil {
				if skipErrors {
					fmt.Printf("  [%s] %s pack error (skipping): %v\n", labelStyle.Render(shard), warningStyle.Render("⚠"), packErr)
					ccRecordSkip(skippedCSV, crawlID, idx, "pack", packErr)
					fmt.Println()
					continue
				}
				return fmt.Errorf("pack %d: %w", idx, packErr)
			}
			elapsed := int64(time.Since(t0).Seconds())
			if rawWARCExists {
				durConvertS = elapsed
			} else {
				durDownloadS = elapsed
			}
		} else if !prefetched {
			fmt.Printf("  [%s] md.warc.gz exists, skipping pack\n", labelStyle.Render(shard))
		}

		if cleanup {
			if rawPath := ccFindRawWARC(crawlID, idx); rawPath != "" {
				_ = os.Remove(rawPath)
				fmt.Printf("  [%s] cleaned up %s\n", labelStyle.Render(shard), filepath.Base(rawPath))
			}
		}

		// Start prefetching the NEXT shard while we export the current one.
		// This overlaps network download (I/O) with parquet export (CPU).
		if i+1 < len(indices) {
			startPrefetch(indices[i+1])
		}

		// Export to .tmp then atomically rename to .parquet.
		fmt.Printf("  [%s] exporting  ", labelStyle.Render(shard))
		t0 := time.Now()
		rows, _, _, exportErr := exportWARCMdShardToParquet(mdWARCPath, tmpPath, func(n int64, elapsed time.Duration) {
			secs := elapsed.Seconds()
			rate := float64(n) / secs
			if secs < 0.1 {
				rate = 0
			}
			fmt.Printf("\r  [%s] exporting  %s docs  %s/s  %s",
				labelStyle.Render(shard),
				infoStyle.Render(ccFmtInt64(n)),
				infoStyle.Render(fmt.Sprintf("%.0f", rate)),
				elapsed.Round(time.Second),
			)
		})
		if exportErr != nil {
			_ = os.Remove(tmpPath)
			fmt.Println()
			if skipErrors {
				fmt.Printf("  [%s] %s export error (skipping): %v\n", labelStyle.Render(shard), warningStyle.Render("⚠"), exportErr)
				ccRecordSkip(skippedCSV, crawlID, idx, "export", exportErr)
				fmt.Println()
				continue
			}
			return fmt.Errorf("export %d: %w", idx, exportErr)
		}
		durExportS = int64(time.Since(t0).Seconds())

		// Atomic rename: watcher only sees complete files.
		if renameErr := os.Rename(tmpPath, parquetPath); renameErr != nil {
			_ = os.Remove(tmpPath)
			if skipErrors {
				fmt.Printf("  [%s] %s rename error (skipping): %v\n", labelStyle.Render(shard), warningStyle.Render("⚠"), renameErr)
				ccRecordSkip(skippedCSV, crawlID, idx, "rename", renameErr)
				fmt.Println()
				continue
			}
			return fmt.Errorf("rename %d: %w", idx, renameErr)
		}

		fmt.Printf("\r  [%s] exported   %s docs  %s\n",
			labelStyle.Render(shard),
			infoStyle.Render(ccFmtInt64(rows)),
			successStyle.Render("done"),
		)

		// Write .meta sidecar with timing and memory for the watcher.
		peakRSS := int64(warcmd.ReadRSSMB())
		metaData, _ := json.Marshal(ccShardMeta{
			DurDownloadS: durDownloadS,
			DurConvertS:  durConvertS,
			DurExportS:   durExportS,
			PeakRSSMB:    peakRSS,
		})
		_ = os.WriteFile(filepath.Join(dataDir, shard+".meta"), metaData, 0o644)

		// Aggressive cleanup: delete md.warc.gz after successful export.
		if cleanup {
			_ = os.Remove(mdWARCPath)
		}

		fmt.Println()
	}
	return nil
}

// ccComputeGapIndices returns a sorted list of uncommitted shard indices in [start, end].
func ccComputeGapIndices(statsCSV, crawlID string, start, end int) []int {
	committed := ccLoadCommittedSet(statsCSV, crawlID)
	var gaps []int
	for i := start; i <= end; i++ {
		if !committed[i] {
			gaps = append(gaps, i)
		}
	}
	return gaps
}

// ccPrintGaps prints a gap analysis report and suggests the next action.
func ccPrintGaps(crawlID, statsCSV string, start, end int, gapIndices []int) error {
	total := end - start + 1
	committed := total - len(gapIndices)

	fmt.Printf("  Crawl    %s\n", labelStyle.Render(crawlID))
	fmt.Printf("  Range    %d–%d (%d shards)\n", start, end, total)
	fmt.Printf("  Done     %s / %d  (%.1f%%)\n",
		infoStyle.Render(strconv.Itoa(committed)), total, float64(committed)*100/float64(total))

	if len(gapIndices) == 0 {
		fmt.Printf("  Gaps     %s\n", successStyle.Render("none — all shards committed"))
		return nil
	}

	fmt.Printf("  Gaps     %s\n", warningStyle.Render(strconv.Itoa(len(gapIndices))))
	fmt.Println()

	// Collapse into ranges for display.
	type rng struct{ lo, hi int }
	var ranges []rng
	lo, hi := gapIndices[0], gapIndices[0]
	for _, n := range gapIndices[1:] {
		if n == hi+1 {
			hi = n
		} else {
			ranges = append(ranges, rng{lo, hi})
			lo, hi = n, n
		}
	}
	ranges = append(ranges, rng{lo, hi})

	for _, r := range ranges {
		if r.lo == r.hi {
			fmt.Printf("    %5d\n", r.lo)
		} else {
			fmt.Printf("    %5d – %5d  (%d)\n", r.lo, r.hi, r.hi-r.lo+1)
		}
	}
	fmt.Println()

	if len(gapIndices) <= 200 {
		parts := make([]string, len(gapIndices))
		for i, g := range gapIndices {
			parts[i] = strconv.Itoa(g)
		}
		fmt.Printf("  Suggest  search cc publish --gaps --pipeline --start %d --end %d\n", start, end)
	} else {
		fmt.Printf("  Suggest  search cc publish --gaps --schedule --start %d --end %d\n", start, end)
	}
	return nil
}

// ccListCommittedShards prints committed shards as compact ranges from stats.csv (synced from HF).
func ccListCommittedShards(ctx context.Context, crawlID, repoRoot, repoID string) error {
	statsCSV := ccStatsCSVPath(repoRoot)
	token := strings.TrimSpace(os.Getenv("HF_TOKEN"))
	if token != "" {
		hf := newHFClient(token)
		ccMergeStatsFromHF(ctx, hf, repoID, statsCSV)
	}
	all, err := ccReadStatsCSV(statsCSV)
	if err != nil {
		return err
	}
	var indices []int
	for _, s := range all {
		if s.CrawlID == crawlID {
			indices = append(indices, s.FileIdx)
		}
	}
	sort.Ints(indices)

	fmt.Printf("  Crawl    %s\n", labelStyle.Render(crawlID))
	fmt.Printf("  Shards   %s committed\n", infoStyle.Render(strconv.Itoa(len(indices))))

	// Show skipped shards from skipped.csv if present.
	skippedCSV := ccSkippedCSVPath(repoRoot)
	if sf, err := os.Open(skippedCSV); err == nil {
		defer sf.Close()
		r := csv.NewReader(sf)
		r.Read() // skip header
		type skipRow struct{ idx int; stage, errMsg, ts string }
		var skips []skipRow
		for {
			row, err := r.Read()
			if err != nil {
				break
			}
			if len(row) < 5 || row[0] != crawlID {
				continue
			}
			idx, _ := strconv.Atoi(row[1])
			skips = append(skips, skipRow{idx, row[2], row[3], row[4]})
		}
		if len(skips) > 0 {
			fmt.Printf("  Skipped  %s (see %s)\n", warningStyle.Render(strconv.Itoa(len(skips))), skippedCSV)
			for _, s := range skips {
				fmt.Printf("    %05d  [%s]  %s\n", s.idx, s.stage, s.errMsg)
			}
		}
	}

	if len(indices) == 0 {
		return nil
	}

	// Collapse into ranges.
	type rng struct{ lo, hi int }
	var ranges []rng
	lo, hi := indices[0], indices[0]
	for _, n := range indices[1:] {
		if n == hi+1 {
			hi = n
		} else {
			ranges = append(ranges, rng{lo, hi})
			lo, hi = n, n
		}
	}
	ranges = append(ranges, rng{lo, hi})

	var parts []string
	for _, r := range ranges {
		if r.lo == r.hi {
			parts = append(parts, strconv.Itoa(r.lo))
		} else {
			parts = append(parts, fmt.Sprintf("%d–%d (%d)", r.lo, r.hi, r.hi-r.lo+1))
		}
	}
	fmt.Printf("  Ranges   %s\n", strings.Join(parts, ",  "))
	return nil
}

// ccDefaultWARCMdConfig returns the warc_md directory path for a crawl.
func ccDefaultWARCMdConfig(crawlID string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "common-crawl", crawlID, "warc_md")
}

// ccFindRawWARC finds the raw .warc.gz file for a given file index.
// It first checks for a .warc.path sidecar written by runCCWarcPack, which
// records the actual CC filename (e.g. CC-MAIN-...-00044.warc.gz) that does
// not contain the file index in its name.
func ccFindRawWARC(crawlID string, idx int) string {
	home, _ := os.UserHomeDir()
	shard := fmt.Sprintf("%05d", idx)
	warcMdDir := filepath.Join(home, "data", "common-crawl", crawlID, "warc_md")
	sidecarPath := filepath.Join(warcMdDir, shard+".warc.path")
	if data, err := os.ReadFile(sidecarPath); err == nil {
		rawPath := strings.TrimSpace(string(data))
		if rawPath != "" {
			if _, err := os.Stat(rawPath); err == nil {
				return rawPath
			}
		}
	}
	// Fallback: legacy glob pattern (pre-sidecar pipelines).
	warcDir := filepath.Join(home, "data", "common-crawl", crawlID, "warc")
	pattern := filepath.Join(warcDir, fmt.Sprintf("*-%05d.warc.gz", idx))
	matches, _ := filepath.Glob(pattern)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// ccRefreshStats scans all exported parquet files and updates stats.csv.
func ccRefreshStats(crawlID, repoRoot, statsCSV string) error {
	dataDir := filepath.Join(repoRoot, "data", crawlID)
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	existing, err := ccReadStatsCSV(statsCSV)
	if err != nil {
		existing = nil
	}
	// Build index of already-known file stats
	known := make(map[int]bool)
	for _, s := range existing {
		if s.CrawlID == crawlID {
			known[s.FileIdx] = true
		}
	}

	var newStats []ccShardStats
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".parquet") {
			continue
		}
		idxStr := strings.TrimSuffix(e.Name(), ".parquet")
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			continue
		}
		if known[idx] {
			continue // already tracked
		}
		parquetPath := filepath.Join(dataDir, e.Name())
		fi, err := os.Stat(parquetPath)
		if err != nil {
			continue
		}
		rows, htmlBytes, mdBytes, err := ccScanParquetStats(parquetPath)
		if err != nil {
			continue
		}
		newStats = append(newStats, ccShardStats{
			CrawlID:      crawlID,
			FileIdx:      idx,
			Rows:         rows,
			HTMLBytes:    htmlBytes,
			MDBytes:      mdBytes,
			ParquetBytes: fi.Size(),
		})
	}

	if len(newStats) == 0 {
		return nil // nothing new
	}

	// Merge with existing
	updated := append(existing, newStats...)
	sort.Slice(updated, func(i, j int) bool {
		if updated[i].CrawlID != updated[j].CrawlID {
			return updated[i].CrawlID < updated[j].CrawlID
		}
		return updated[i].FileIdx < updated[j].FileIdx
	})
	return ccWriteStatsCSV(statsCSV, updated)
}

func ccDefaultExportRepoRoot(crawlID string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "common-crawl", crawlID, "export", "repo")
}

func ccEnsurePublishRepoFiles(repoRoot, crawlID, statsCSV string) error {
	if err := os.MkdirAll(filepath.Join(repoRoot, "data"), 0o755); err != nil {
		return fmt.Errorf("create repo root: %w", err)
	}

	// Load stats for real numbers in README
	allStats, _ := ccReadStatsCSV(statsCSV)
	totals := ccComputeTotals(allStats, crawlID)

	files := map[string]string{
		filepath.Join(repoRoot, "README.md"): ccPublishREADME(crawlID, &totals),
		filepath.Join(repoRoot, "LICENSE"):   ccPublishLicense(),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", filepath.Base(path), err)
		}
	}
	return nil
}

func ccResolvePublishUploadFiles(repoRoot, crawlID, selector string) ([]ccPublishUploadFile, error) {
	dataDir := filepath.Join(repoRoot, "data")
	crawlDataDir := filepath.Join(dataDir, crawlID)
	if selector == "" || selector == "all" {
		var files []ccPublishUploadFile
		_ = filepath.WalkDir(dataDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".parquet") {
				return nil
			}
			rel, _ := filepath.Rel(repoRoot, path)
			files = append(files, ccPublishUploadFile{
				LocalPath:  path,
				PathInRepo: filepath.ToSlash(rel),
			})
			return nil
		})
		sort.Slice(files, func(i, j int) bool { return files[i].PathInRepo < files[j].PathInRepo })
		return files, nil
	}

	indices, err := ccParseOpenFileSelector(selector)
	if err != nil {
		return nil, err
	}
	files := make([]ccPublishUploadFile, 0, len(indices))
	for _, idx := range indices {
		name := fmt.Sprintf("%05d.parquet", idx)
		localPath := filepath.Join(crawlDataDir, name)
		if !fileExists(localPath) {
			return nil, fmt.Errorf("selected parquet file not found: %s", localPath)
		}
		files = append(files, ccPublishUploadFile{
			LocalPath:  localPath,
			PathInRepo: filepath.ToSlash(filepath.Join("data", crawlID, name)),
		})
	}
	return files, nil
}

func ccParseOpenFileSelector(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "all" {
		return nil, nil
	}
	// Comma-separated list: "1,2,5-10,42"
	if strings.Contains(s, ",") {
		seen := make(map[int]bool)
		var out []int
		for _, part := range strings.Split(s, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			sub, err := ccParseOpenFileSelector(part)
			if err != nil {
				return nil, err
			}
			for _, n := range sub {
				if !seen[n] {
					seen[n] = true
					out = append(out, n)
				}
			}
		}
		sort.Ints(out)
		return out, nil
	}
	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		lo, err1 := strconv.Atoi(parts[0])
		hi, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || lo < 0 || hi < lo {
			return nil, fmt.Errorf("invalid range %q", s)
		}
		out := make([]int, hi-lo+1)
		for i := range out {
			out[i] = lo + i
		}
		return out, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return nil, fmt.Errorf("invalid file index %q", s)
	}
	return []int{n}, nil
}

func ccPublishCommitMessage(fileIdx string, files []ccPublishUploadFile) string {
	if len(files) == 1 {
		return "Publish " + files[0].PathInRepo
	}
	if fileIdx != "" && fileIdx != "all" {
		return "Publish Common Crawl shards " + fileIdx
	}
	return fmt.Sprintf("Publish %d Common Crawl parquet shards", len(files))
}

// ccPublishREADME generates the dataset README with real numbers from stats.
func ccPublishREADME(crawlID string, totals *ccTotals) string {
	c := crawlID
	cb := "```"
	bt := "`"

	// Compression table strings — use actual sums across all files, not per-shard averages.
	// Fall back to hardcoded shard-0 measurements when no stats are available.
	var (
		totalDocsStr    = "~21,000 per shard"
		shardsCountStr  = "1"
		totalDocsTable  = "~21,000"
		rawWARCEstStr   = "~0.8 GB"
		totalHTMLStr    = "2.7 GB"
		packMDEstStr    = "~75 MB"
		pctHTMLToMDStr  = "97.2"
		totalParquetStr = "27.9 MB"
		pctPackToPQStr  = "23.0"
		totalMDStr      = "79.2 MB"
		pctMDToPQStr    = "64.7"
		endToEndPctStr  = "96.5"
		// Projected 100k values (computed from per-shard averages)
		projRawWARCStr  = "~81 TB"
		projHTMLStr     = "~232 TB"
		projPackStr     = "~4.1 TB"
		projParquetStr  = "~2.8 TB"
		// Timing table defaults (shown only when we have timing data)
		timingSection = ""
		// Progress/ETA line (shown when we have throughput data)
		progressLine = ""
	)

	if totals != nil && totals.Shards > 0 {
		shardsCountStr = strconv.Itoa(totals.Shards)
		totalDocsTable = ccFmtInt64(totals.Rows)
		totalDocsStr = ccFmtInt64(totals.Rows) + " documents across " + strconv.Itoa(totals.Shards) + " shards"
		totalHTMLStr = ccFmtBytes(totals.HTMLBytes)
		totalMDStr = ccFmtBytes(totals.MDBytes)
		totalParquetStr = ccFmtBytes(totals.ParquetBytes)

		if totals.HTMLBytes > 0 {
			pct := float64(totals.HTMLBytes-totals.MDBytes) / float64(totals.HTMLBytes) * 100
			pctHTMLToMDStr = fmt.Sprintf("%.1f", pct)
		}
		if totals.MDBytes > 0 {
			pct := math.Max(0, float64(totals.MDBytes-totals.ParquetBytes)/float64(totals.MDBytes)*100)
			pctMDToPQStr = fmt.Sprintf("%.1f", pct)
		}

		// Estimate raw WARC (~830 MB per shard compressed) and packed WARC (~47% of uncompressed MD)
		rawWARCBytes := int64(totals.Shards) * 830 * 1024 * 1024
		rawWARCEstStr = "~" + ccFmtBytes(rawWARCBytes)
		packBytes := int64(float64(totals.MDBytes) * 0.47)
		packMDEstStr = "~" + ccFmtBytes(packBytes)
		if totals.HTMLBytes > 0 && packBytes > 0 {
			pct := float64(totals.HTMLBytes-packBytes) / float64(totals.HTMLBytes) * 100
			pctHTMLToMDStr = fmt.Sprintf("%.1f", pct)
		}
		if packBytes > totals.ParquetBytes {
			pct := float64(packBytes-totals.ParquetBytes) / float64(packBytes) * 100
			pctPackToPQStr = fmt.Sprintf("%.1f", pct)
		}
		if rawWARCBytes > totals.ParquetBytes {
			pct := float64(rawWARCBytes-totals.ParquetBytes) / float64(rawWARCBytes) * 100
			endToEndPctStr = fmt.Sprintf("%.1f", pct)
		}

		// Projected 100k values: scale per-shard averages to 100,000 files.
		const projFiles = 100_000
		projScale := float64(projFiles) / float64(totals.Shards)
		projRawWARCStr = "~" + ccFmtBytes(int64(float64(rawWARCBytes)*projScale))
		projHTMLStr = "~" + ccFmtBytes(int64(float64(totals.HTMLBytes)*projScale))
		projPackStr = "~" + ccFmtBytes(int64(float64(packBytes)*projScale))
		projParquetStr = "~" + ccFmtBytes(int64(float64(totals.ParquetBytes)*projScale))

		// Progress/ETA — computed from stats.csv timestamps.
		if totals.ShardsPerHour > 0 {
			remaining := projFiles - totals.Shards
			if remaining > 0 {
				etaHours := float64(remaining) / totals.ShardsPerHour
				etaDone := time.Now().Add(time.Duration(etaHours * float64(time.Hour)))
				if etaHours >= 24 {
					progressLine = fmt.Sprintf(
						"\nProcessing at **%.1f shards/hour** — estimated completion of all 100,000 shards: **%s** (~%.0f days).",
						totals.ShardsPerHour, etaDone.Format("January 2, 2006"), etaHours/24)
				} else {
					progressLine = fmt.Sprintf(
						"\nProcessing at **%.1f shards/hour** — estimated completion of all 100,000 shards: **%s** (~%.1f hours).",
						totals.ShardsPerHour, etaDone.Format("January 2, 2006 15:04 MST"), etaHours)
				}
			}
		}

		// Timing section — only shown when at least one shard has timing data.
		if totals.DurDownloadS+totals.DurConvertS+totals.DurExportS+totals.DurPublishS > 0 {
			avgDownloadS, avgConvertS, avgExportS, avgPublishS := int64(0), int64(0), int64(0), int64(0)
			if totals.Shards > 0 {
				avgDownloadS = totals.DurDownloadS / int64(totals.Shards)
				avgConvertS = totals.DurConvertS / int64(totals.Shards)
				avgExportS = totals.DurExportS / int64(totals.Shards)
				avgPublishS = totals.DurPublishS / int64(totals.Shards)
			}
			// Use the largest stage as the bar-chart reference so bars are relative.
			maxDurS := totals.DurDownloadS
			for _, v := range []int64{totals.DurConvertS, totals.DurExportS, totals.DurPublishS} {
				if v > maxDurS {
					maxDurS = v
				}
			}
			timingSection = "\n### Processing Times\n\nPipeline timings across " +
				shardsCountStr + " shards of " + crawlID + ":\n\n" +
				"```\n" +
				ccTimingBar("Download (raw WARC)      ", totals.DurDownloadS, avgDownloadS, maxDurS) +
				ccTimingBar("Convert  (HTML → MD)     ", totals.DurConvertS, avgConvertS, maxDurS) +
				ccTimingBar("Export   (Parquet)        ", totals.DurExportS, avgExportS, maxDurS) +
				ccTimingBar("Publish  (HuggingFace)    ", totals.DurPublishS, avgPublishS, maxDurS) +
				"```\n" +
				"\n### Dataset Charts\n\n" +
				"![Total size: HTML vs Markdown vs Parquet](charts/totals_chart.png)\n\n" +
				"![Pipeline stage durations](charts/timing_chart.png)\n"
		}
	}

	return fmt.Sprintf(`---
license: odc-by
task_categories:
- text-generation
- feature-extraction
language:
- en
pretty_name: Open Markdown
size_categories:
- 1M<n<10M
tags:
- common-crawl
- web-crawl
- markdown
- text
configs:
- config_name: default
  data_files:
  - split: train
    path: data/*/*
- config_name: %[1]s
  data_files:
  - split: train
    path: data/%[1]s/*
---

# **Open Markdown**

> Clean markdown from the web, ready for training and retrieval

## What is it?

**Open Markdown** is a large-scale web text dataset built from [Common Crawl](https://commoncrawl.org). Common Crawl is a non-profit that crawls the web and freely provides its archives and datasets to the public — see [their latest crawl announcement](https://commoncrawl.org/blog/march-2026-crawl-archive-now-available) for details on the source data. Every page goes through a pipeline that extracts the main content from raw HTML, converts it to clean Markdown, and packages the result into Parquet files with useful WARC metadata for traceability.

The dataset currently includes crawl **%[1]s** with **%[7]s**. We plan to add more snapshots over time.
%[24]s

**Open Markdown** is released under the **Open Data Commons Attribution License (ODC-By) v1.0**, the same license used by Common Crawl.

## What is being released?

Each Common Crawl WARC file (~1 GB of compressed HTML) becomes one Parquet shard. The shards live under a crawl-specific directory so multiple snapshots can coexist:

%[2]s
data/
  %[1]s/
    00000.parquet
    00001.parquet
    ...
%[2]s

Every row in a Parquet file is one web page. Each row includes the %[3]swarc_record_id%[3]s and %[3]swarc_refers_to%[3]s fields parsed from the original WARC headers, so you can trace any document back to its source record. We also store %[3]shtml_length%[3]s and %[3]smarkdown_length%[3]s to measure the compression from raw HTML to clean markdown.

## How to download and use Open Markdown

### Using %[3]sdatasets%[3]s

%[2]spython
from datasets import load_dataset

# stream the entire dataset
ds = load_dataset("open-index/open-markdown", name="%[1]s", split="train", streaming=True)
for doc in ds:
    print(doc["url"], len(doc["markdown"]))

# load a single shard into memory
ds = load_dataset(
    "open-index/open-markdown",
    data_files="data/%[1]s/00000.parquet",
    split="train",
)
%[2]s

### Using %[3]shuggingface_hub%[3]s

%[2]spython
from huggingface_hub import snapshot_download

folder = snapshot_download(
    "open-index/open-markdown",
    repo_type="dataset",
    local_dir="./open-index/",
    allow_patterns="data/%[1]s/*",
)
%[2]s

For faster downloads, install %[3]spip install huggingface_hub[hf_transfer]%[3]s and set %[3]sHF_HUB_ENABLE_HF_TRANSFER=1%[3]s.

### Using DuckDB

%[2]ssql
SELECT url, host, markdown_length
FROM read_parquet('hf://datasets/open-index/open-markdown/data/%[1]s/*.parquet')
WHERE host = 'en.wikipedia.org'
LIMIT 10;
%[2]s

# Dataset card for Open Markdown

## Dataset Description

- **Homepage and Repository:** [https://huggingface.co/datasets/open-index/open-markdown](https://huggingface.co/datasets/open-index/open-markdown)
- **Point of Contact:** please create a discussion on the Community tab
- **License:** Open Data Commons Attribution License (ODC-By) v1.0

## Dataset Structure

### Data Instance

The following is an example row from the dataset:

%[2]sjson
{
  "doc_id": "6aaa5be7-a917-5105-aa60-e39ea1d087fc",
  "url": "https://example.com/article/interesting-topic",
  "host": "example.com",
  "crawl_date": "2026-02-06T18:14:58Z",
  "warc_record_id": "<urn:uuid:a1b2c3d4-e5f6-7890-abcd-ef1234567890>",
  "warc_refers_to": "<urn:uuid:f9e8d7c6-b5a4-3210-fedc-ba0987654321>",
  "html_length": 48210,
  "markdown_length": 3847,
  "markdown": "# Interesting Topic\n\nThis is the main content of the page..."
}
%[2]s

### Data Fields

| Column | Type | Description |
|---|---|---|
| %[3]sdoc_id%[3]s | string | Deterministic UUID v5 derived from the canonical URL: %[3]sdoc_id = UUID5(NamespaceURL, url)%[3]s — identical URLs always produce the same %[3]sdoc_id%[3]s across crawls |
| %[3]surl%[3]s | string | Original URL of the crawled page |
| %[3]shost%[3]s | string | Lowercase hostname extracted from the URL |
| %[3]scrawl_date%[3]s | string | RFC 3339 timestamp from the WARC record |
| %[3]swarc_record_id%[3]s | string | Full WARC-Record-ID of this conversion record (%[3]s<urn:uuid:...>%[3]s) |
| %[3]swarc_refers_to%[3]s | string | WARC-Record-ID of the original HTTP response this was converted from |
| %[3]shtml_length%[3]s | int64 | Byte length of the original HTML body before conversion |
| %[3]smarkdown_length%[3]s | int64 | Byte length of the converted markdown body |
| %[3]smarkdown%[3]s | string | Clean markdown content extracted from the page |

### Data Splits

The default subset includes all available data across all crawl snapshots. You can also load a specific crawl by using its ID as the config name (e.g. %[3]s%[1]s%[3]s).

## Dataset Creation

### Curation Rationale

Most open web datasets either release raw text without structure or keep the HTML and leave parsing to the user. **Open Markdown** sits in between: it converts every page to Markdown so the content is immediately usable for training, while preserving key WARC identifiers (%[3]swarc_record_id%[3]s, %[3]swarc_refers_to%[3]s) so you can always trace back to the source record.

### Source Data

The source data consists of web pages crawled by the [Common Crawl](https://commoncrawl.org) foundation. Common Crawl archives billions of pages across the public web and makes the raw WARC files freely available on Amazon S3.

### Data Processing Steps

The processing pipeline runs in five stages:

1. **Download** raw .warc.gz files from Common Crawl S3 (each file is roughly 1 GB compressed)
2. **Filter** to keep only HTTP 200 responses with a text/html content type, discarding images, scripts, redirects, and error pages
3. **Convert** HTML to Markdown using [trafilatura](https://github.com/adbar/trafilatura), which extracts the main content and strips boilerplate, navigation, sidebars, footers, and ads
4. **Pack** converted records into seekable .md.warc.gz files where each record is wrapped in its own gzip member, matching Common Crawl's concatenated-gzip format
5. **Export** each shard to Apache Parquet with Zstd compression, 100,000 rows per row group, and an 8 MB page buffer

Empty conversions (pages where trafilatura could not extract meaningful content) are dropped.

### Compression Ratios

Numbers below are actual measurements summed across all %[8]s files of %[1]s (%[9]s pages total), projected to the full crawl of 100,000 WARC files.

| Stage | %[8]s files (measured) | 100,000 files (projected) | Reduction |
|---|---|---|---|
| Raw WARC (.warc.gz, downloaded) | %[10]s | %[20]s | — |
| HTML extracted (uncompressed) | %[11]s | %[21]s | — |
| Packed markdown WARC (.md.warc.gz) | %[12]s | %[22]s | **-%[13]s%%** vs HTML |
| Final Parquet (Zstd level 19) | %[14]s | %[23]s | **-%[15]s%%** vs packed WARC |

The big win is the HTML → Markdown step: trafilatura strips all tags, scripts, styles, navigation, and ads, keeping only the main content. This cuts %[11]s of uncompressed HTML down to %[16]s of markdown — a **%[13]s%% reduction** — before any file-level compression is applied. Parquet with Zstd level 19 then compresses the markdown a further %[17]s%%.

End to end: %[10]s of raw gzipped WARCs becomes **%[14]s of Parquet** — a **%[18]s%% total reduction** — containing %[9]s clean markdown documents.
%[19]s
### Personal and Sensitive Information

No additional PII filtering is applied beyond what Common Crawl provides. As the dataset is sourced from the public web, it is likely that some personally identifiable information is present. If you find your own PII in the dataset and would like it removed, please open an issue on the repository.

## Considerations for Using the Data

### Social Impact

By releasing both the dataset and the full processing pipeline, we aim to lower the barrier to training and evaluating language models on high quality web data. Researchers and practitioners who cannot afford to run their own Common Crawl processing pipelines can use **Open Markdown** directly.

### Discussion of Biases

**Open Markdown** inherits the biases present in Common Crawl and the public web at large. The trafilatura extraction step favors article-like pages and may underrepresent content from forums, social media, and non-standard page layouts. We have not applied any machine-learning-based quality or toxicity filters, as such filters have been shown to disproportionately remove content from certain dialects and communities.

### Known Limitations

Code-heavy pages may not convert well to Markdown. If you are training a model that needs strong code performance, consider supplementing **Open Markdown** with a dedicated code dataset such as [The Stack v2](https://huggingface.co/datasets/bigcode/the-stack-v2). Similarly, highly structured pages like Wikipedia may have better formatting in dedicated Wikipedia dumps than in their Common Crawl versions.

## Additional Information

### Licensing

The dataset is released under the **Open Data Commons Attribution License (ODC-By) v1.0**. The use of this dataset is also subject to [Common Crawl's Terms of Use](https://commoncrawl.org/terms-of-use). The original content remains subject to the rights and terms of its respective publishers.

### Contact

Please open a discussion on the [Community tab](https://huggingface.co/datasets/open-index/open-markdown/discussions) for questions, feedback, or issues.
`,
		c,               // [1] crawlID
		cb,              // [2] triple backtick
		bt,              // [3] single backtick
		"",              // [4] unused
		"",              // [5] unused
		"",              // [6] unused
		totalDocsStr,    // [7] "X documents across N shards"
		shardsCountStr,  // [8] number of files/shards
		totalDocsTable,  // [9] total row count
		rawWARCEstStr,   // [10] estimated raw WARC total
		totalHTMLStr,    // [11] total uncompressed HTML
		packMDEstStr,    // [12] estimated packed WARC total
		pctHTMLToMDStr,  // [13] HTML → packed WARC reduction %
		totalParquetStr, // [14] total parquet size
		pctPackToPQStr,  // [15] packed WARC → parquet reduction %
		totalMDStr,      // [16] total uncompressed markdown
		pctMDToPQStr,    // [17] markdown → parquet compression %
		endToEndPctStr,  // [18] raw WARC → parquet end-to-end %
		timingSection,   // [19] optional processing times table
		projRawWARCStr,  // [20] projected raw WARC for 100k files
		projHTMLStr,     // [21] projected HTML for 100k files
		projPackStr,     // [22] projected packed WARC for 100k files
		projParquetStr,  // [23] projected parquet for 100k files
		progressLine,    // [24] progress/ETA line
	)
}

func ccPublishLicense() string {
	return `Open Data Commons Attribution License (ODC-By) v1.0

Full text: https://opendatacommons.org/licenses/by/1-0/

You are free to copy, distribute, use, modify, transform, and build upon
this database, as long as you attribute the source.

Attribution: "Open Markdown, derived from Common Crawl (https://commoncrawl.org)"

Note: This dataset contains data derived from Common Crawl, which archives
third-party web content. The original content remains subject to the rights
of its respective publishers. You are responsible for complying with applicable
law including downstream licensing obligations, robots.txt restrictions, privacy
requirements, and content removal requests. See Common Crawl's Terms of Use:
https://commoncrawl.org/terms-of-use
`
}
