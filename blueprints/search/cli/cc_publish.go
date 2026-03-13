package cli

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type ccPublishUploadFile struct {
	LocalPath  string
	PathInRepo string
}

func newCCPublish() *cobra.Command {
	var (
		crawlID      string
		fileIdx      string
		repoRoot     string
		repoID       string
		republish    bool
		private      bool
		pipeline     bool
		cleanup      bool
		lightConvert bool
	)

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish exported Common Crawl parquet shards to Hugging Face",
		Long: `Publish $HOME/data/common-crawl/{crawl}/export/repo to a Hugging Face dataset repo.

The command creates the dataset repo if needed, ensures README.md and LICENSE
exist locally, uploads only missing parquet files by default, and supports
targeting one shard with --file 0.

With --pipeline, automatically downloads, packs and exports any missing shards
before uploading. Use --cleanup to delete raw .warc.gz files after packing
to save disk space.`,
		Example: `  search cc publish
  search cc publish --file 0
  search cc publish --crawl CC-MAIN-2026-08 --repo open-index/draft
  search cc publish --file 0 --republish
  search cc publish --file 11-90 --pipeline --cleanup`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCPublish(cmd.Context(), crawlID, fileIdx, repoRoot, repoID, republish, private, pipeline, cleanup, lightConvert)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "all", "File index, range (0-9), or all")
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "Local export repo root (default: $HOME/data/common-crawl/{crawl}/export/repo)")
	cmd.Flags().StringVar(&repoID, "repo", "open-index/draft", "Hugging Face dataset repo ID")
	cmd.Flags().BoolVar(&republish, "republish", false, "Upload even if the remote path already exists")
	cmd.Flags().BoolVar(&private, "private", false, "Create the Hugging Face dataset repo as private")
	cmd.Flags().BoolVar(&pipeline, "pipeline", false, "Auto-download, pack, and export missing shards before publishing")
	cmd.Flags().BoolVar(&cleanup, "cleanup", false, "Delete raw .warc.gz after packing (requires --pipeline)")
	cmd.Flags().BoolVar(&lightConvert, "light", true, "Use lightweight HTML→Markdown converter (~10x faster than trafilatura, --no-light for trafilatura)")
	return cmd
}

func runCCPublish(ctx context.Context, crawlID, fileIdx, repoRoot, repoID string, republish, private, pipeline, cleanup, lightConvert bool) error {
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

	// ── Pipeline mode: per-file pack → export → commit ───────────────────────
	if pipeline {
		return ccRunPipelineWithCommits(ctx, crawlID, fileIdx, repoRoot, repoID, republish, private, cleanup, lightConvert)
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

// ccRunPipelineWithCommits processes each selected shard end-to-end:
// download → pack → export → upsert stats → update README → HF commit.
// Each shard is committed to HuggingFace individually so progress is saved after every file.
func ccRunPipelineWithCommits(ctx context.Context, crawlID, fileIdx, repoRoot, repoID string, republish, private, cleanup, lightConvert bool) error {
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

	token := strings.TrimSpace(os.Getenv("HF_TOKEN"))
	if token == "" {
		return fmt.Errorf("HF_TOKEN environment variable is not set")
	}
	hf := newHFClient(token)

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("CC Pipeline: download → pack → export → publish"))
	fmt.Println()
	fmt.Printf("  Crawl     %s\n", labelStyle.Render(crawlID))
	fmt.Printf("  Files     %s\n", infoStyle.Render(strconv.Itoa(len(indices))))
	fmt.Printf("  HF repo   %s\n", infoStyle.Render(repoID))
	fmt.Println()

	// Create repo once before the per-file loop.
	if err := hf.createDatasetRepo(ctx, repoID, private); err != nil {
		return fmt.Errorf("create repo: %w", err)
	}

	for i, idx := range indices {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		shard := fmt.Sprintf("%05d", idx)
		mdWARCPath := filepath.Join(warcMdDir, shard+".md.warc.gz")
		parquetPath := filepath.Join(dataDir, shard+".parquet")

		fmt.Printf("  ── [%d/%d] %s ──\n", i+1, len(indices), labelStyle.Render(shard))

		var rows, htmlBytes, mdBytes int64
		var durPackS, durExportS, durPublishS int64

		if !fileExists(parquetPath) {
			// Need to pack (and possibly download) before exporting.
			if !fileExists(mdWARCPath) {
				fmt.Printf("  [%s] packing...\n", labelStyle.Render(shard))
				t0 := time.Now()
				if packErr := runCCWarcPack(ctx, crawlID, strconv.Itoa(idx), -1, -1, 0, false, false, lightConvert, 200, "text/html", 512*1024); packErr != nil {
					return fmt.Errorf("pack %d: %w", idx, packErr)
				}
				durPackS = int64(time.Since(t0).Seconds())
				if cleanup {
					if rawPath := ccFindRawWARC(crawlID, idx); rawPath != "" {
						_ = os.Remove(rawPath)
						fmt.Printf("  [%s] cleaned up %s\n", labelStyle.Render(shard), filepath.Base(rawPath))
					}
				}
			} else {
				fmt.Printf("  [%s] md.warc.gz exists, skipping pack\n", labelStyle.Render(shard))
			}

			fmt.Printf("  [%s] exporting  ", labelStyle.Render(shard))
			t0 := time.Now()
			var exportErr error
			rows, htmlBytes, mdBytes, exportErr = exportWARCMdShardToParquet(mdWARCPath, parquetPath, func(n int64, elapsed time.Duration) {
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
				fmt.Println()
				return fmt.Errorf("export %d: %w", idx, exportErr)
			}
			durExportS = int64(time.Since(t0).Seconds())
			fmt.Printf("\r  [%s] exported   %s docs  %s\n",
				labelStyle.Render(shard),
				infoStyle.Render(ccFmtInt64(rows)),
				successStyle.Render("done"),
			)
		} else {
			fmt.Printf("  [%s] parquet exists, scanning stats\n", labelStyle.Render(shard))
			var scanErr error
			rows, htmlBytes, mdBytes, scanErr = ccScanParquetStats(parquetPath)
			if scanErr != nil {
				return fmt.Errorf("scan parquet stats %d: %w", idx, scanErr)
			}
			fmt.Printf("  [%s] %s docs\n", labelStyle.Render(shard), infoStyle.Render(ccFmtInt64(rows)))
		}

		// Upsert this shard's stats into stats.csv (safe: only touches this file's row).
		pqFI, _ := os.Stat(parquetPath)
		pqBytes := int64(0)
		if pqFI != nil {
			pqBytes = pqFI.Size()
		}
		if upsertErr := ccUpsertShardStats(statsCSV, ccShardStats{
			CrawlID: crawlID, FileIdx: idx,
			Rows: rows, HTMLBytes: htmlBytes, MDBytes: mdBytes, ParquetBytes: pqBytes,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			DurPackS: durPackS, DurExportS: durExportS,
		}); upsertErr != nil {
			return fmt.Errorf("upsert stats %d: %w", idx, upsertErr)
		}

		// Regenerate README with updated cumulative numbers.
		if repoErr := ccEnsurePublishRepoFiles(repoRoot, crawlID, statsCSV); repoErr != nil {
			return fmt.Errorf("write repo files %d: %w", idx, repoErr)
		}

		// Generate PNG charts from stats.csv (requires uv + chart_stats.py).
		chartRelPaths := ccRunCharts(statsCSV, repoRoot, crawlID)
		if len(chartRelPaths) > 0 {
			fmt.Printf("  [%s] charts   %s\n", labelStyle.Render(shard), infoStyle.Render(fmt.Sprintf("%d PNGs", len(chartRelPaths))))
		}

		// Decide whether to upload the parquet (--republish or not yet on HF).
		parquetRemotePath := filepath.ToSlash(filepath.Join("data", crawlID, shard+".parquet"))
		uploadParquet := republish
		if !uploadParquet {
			existing, pathErr := hf.pathsExist(ctx, repoID, []string{parquetRemotePath})
			if pathErr != nil {
				return fmt.Errorf("check remote %d: %w", idx, pathErr)
			}
			uploadParquet = !existing[parquetRemotePath]
		}

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
		if uploadParquet {
			ops = append(ops, hfOperation{LocalPath: parquetPath, PathInRepo: parquetRemotePath})
		}

		commitMsg := fmt.Sprintf("Publish shard %s/%s", crawlID, shard)
		t0 := time.Now()
		commitURL, commitErr := hf.createCommit(ctx, repoID, "main", commitMsg, ops)
		if commitErr != nil {
			return fmt.Errorf("commit %d: %w", idx, commitErr)
		}
		durPublishS = int64(time.Since(t0).Seconds())

		// Back-fill publish duration into stats.csv.
		_ = ccUpsertShardStats(statsCSV, ccShardStats{
			CrawlID: crawlID, FileIdx: idx,
			Rows: rows, HTMLBytes: htmlBytes, MDBytes: mdBytes, ParquetBytes: pqBytes,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			DurPackS: durPackS, DurExportS: durExportS, DurPublishS: durPublishS,
		})

		if uploadParquet {
			fmt.Printf("  [%s] %s  %s\n", labelStyle.Render(shard), successStyle.Render("published"), labelStyle.Render(commitURL))
		} else {
			fmt.Printf("  [%s] %s  %s\n", labelStyle.Render(shard), warningStyle.Render("metadata only"), labelStyle.Render(commitURL))
		}
		fmt.Println()
	}

	// Print final cumulative stats.
	if allStats, readErr := ccReadStatsCSV(statsCSV); readErr == nil && len(allStats) > 0 {
		t := ccComputeTotals(allStats, crawlID)
		if t.Shards > 0 {
			fmt.Printf("  ── Cumulative stats (%s) ──\n", crawlID)
			fmt.Printf("  Shards     %s\n", infoStyle.Render(strconv.Itoa(t.Shards)))
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

// ccDefaultWARCMdConfig returns the warc_md directory path for a crawl.
func ccDefaultWARCMdConfig(crawlID string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "common-crawl", crawlID, "warc_md")
}

// ccFindRawWARC finds the raw .warc.gz file for a given file index.
func ccFindRawWARC(crawlID string, idx int) string {
	home, _ := os.UserHomeDir()
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
		// Timing table defaults (shown only when we have timing data)
		timingSection = ""
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

		// Timing section — only shown when at least one shard has timing data.
		if totals.DurPackS+totals.DurExportS+totals.DurPublishS > 0 {
			avgPackS, avgExportS, avgPublishS := int64(0), int64(0), int64(0)
			if totals.Shards > 0 {
				avgPackS = totals.DurPackS / int64(totals.Shards)
				avgExportS = totals.DurExportS / int64(totals.Shards)
				avgPublishS = totals.DurPublishS / int64(totals.Shards)
			}
			timingSection = "\n### Processing Times\n\nPipeline timings across " +
				shardsCountStr + " shards of " + crawlID + ":\n\n" +
				"```\n" +
				ccTimingBar("Pack (download + HTML→MD)", totals.DurPackS, avgPackS, totals.DurPackS) +
				ccTimingBar("Export (Parquet)         ", totals.DurExportS, avgExportS, totals.DurPackS) +
				ccTimingBar("Publish (HuggingFace)    ", totals.DurPublishS, avgPublishS, totals.DurPackS) +
				"```\n"
		}
	}

	return fmt.Sprintf(`---
license: odc-by
task_categories:
- text-generation
- feature-extraction
language:
- en
pretty_name: Open Index
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

# Open Index

> Clean markdown from the web, ready for training and retrieval

## What is it?

Open Index is a large-scale web text dataset built from [Common Crawl](https://commoncrawl.org). Every page goes through a pipeline that extracts the main content from raw HTML, converts it to clean Markdown using [trafilatura](https://github.com/adbar/trafilatura), and packages the result into Parquet files with full WARC metadata preserved.

The dataset currently includes crawl **%[1]s** with **%[7]s**. We plan to add more snapshots over time.

Open Index is released under the **Open Data Commons Attribution License (ODC-By) v1.0**, the same license used by Common Crawl.

## What is being released?

Each Common Crawl WARC file (~1 GB of compressed HTML) becomes one Parquet shard. The shards live under a crawl-specific directory so multiple snapshots can coexist:

%[2]s
data/
  %[1]s/
    00000.parquet
    00001.parquet
    ...
%[2]s

Every row in a Parquet file is one web page. Along with the markdown body, we preserve the original WARC headers as a JSON column so you can always trace a document back to its source record.

## How to download and use Open Index

### Using %[3]sdatasets%[3]s

%[2]spython
from datasets import load_dataset

# stream the entire dataset
ds = load_dataset("open-index/draft", name="%[1]s", split="train", streaming=True)
for doc in ds:
    print(doc["url"], len(doc["markdown"]))

# load a single shard into memory
ds = load_dataset(
    "open-index/draft",
    data_files="data/%[1]s/00000.parquet",
    split="train",
)
%[2]s

### Using %[3]shuggingface_hub%[3]s

%[2]spython
from huggingface_hub import snapshot_download

folder = snapshot_download(
    "open-index/draft",
    repo_type="dataset",
    local_dir="./open-index/",
    allow_patterns="data/%[1]s/*",
)
%[2]s

For faster downloads, install %[3]spip install huggingface_hub[hf_transfer]%[3]s and set %[3]sHF_HUB_ENABLE_HF_TRANSFER=1%[3]s.

### Using DuckDB

%[2]ssql
SELECT url, host, markdown_length
FROM read_parquet('hf://datasets/open-index/draft/data/%[1]s/*.parquet')
WHERE host = 'en.wikipedia.org'
LIMIT 10;
%[2]s

# Dataset card for Open Index

## Dataset Description

- **Homepage and Repository:** [https://huggingface.co/datasets/open-index/draft](https://huggingface.co/datasets/open-index/draft)
- **Point of Contact:** please create a discussion on the Community tab
- **License:** Open Data Commons Attribution License (ODC-By) v1.0

## Dataset Structure

### Data Instance

The following is an example row from the dataset:

%[2]sjson
{
  "doc_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "url": "https://example.com/article/interesting-topic",
  "host": "example.com",
  "crawl_date": "2026-02-06T18:14:58Z",
  "warc_record_id": "<urn:uuid:a1b2c3d4-e5f6-7890-abcd-ef1234567890>",
  "warc_refers_to": "<urn:uuid:f9e8d7c6-b5a4-3210-fedc-ba0987654321>",
  "html_length": 48210,
  "markdown_length": 3847,
  "warc_headers_json": "{\"Content-Length\": \"3847\", ...}",
  "markdown": "# Interesting Topic\n\nThis is the main content of the page..."
}
%[2]s

### Data Fields

| Column | Type | Description |
|---|---|---|
| %[3]sdoc_id%[3]s | string | Unique identifier derived from the WARC-Record-ID (UUID) |
| %[3]surl%[3]s | string | Original URL of the crawled page |
| %[3]shost%[3]s | string | Lowercase hostname extracted from the URL |
| %[3]scrawl_date%[3]s | string | RFC 3339 timestamp from the WARC record |
| %[3]swarc_record_id%[3]s | string | Full WARC-Record-ID of the conversion record (%[3]s<urn:uuid:...>%[3]s) |
| %[3]swarc_refers_to%[3]s | string | WARC-Record-ID of the original HTTP response this was converted from |
| %[3]shtml_length%[3]s | int64 | Byte length of the original HTML body before conversion |
| %[3]smarkdown_length%[3]s | int64 | Byte length of the converted markdown body |
| %[3]swarc_headers_json%[3]s | string | All WARC headers as a sorted-key JSON object — full provenance in one column |
| %[3]smarkdown%[3]s | string | Clean markdown content extracted from the page |

### Data Splits

The default subset includes all available data across all crawl snapshots. You can also load a specific crawl by using its ID as the config name (e.g. %[3]s%[1]s%[3]s).

## Dataset Creation

### Curation Rationale

Most open web datasets either release raw text without structure or keep the HTML and leave parsing to the user. Open Index sits in between: it converts every page to Markdown so the content is immediately usable for training, while preserving the full WARC headers so you can always go back to the source if you need to.

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
| Raw WARC (.warc.gz, downloaded) | %[10]s | ~83 TB | — |
| HTML extracted (uncompressed) | %[11]s | ~295 TB | — |
| Packed markdown WARC (.md.warc.gz) | %[12]s | ~3.7 TB | **-%[13]s%%** vs HTML |
| Final Parquet (Zstd level 19) | %[14]s | ~2.9 TB | **-%[15]s%%** vs packed WARC |

The big win is the HTML → Markdown step: trafilatura strips all tags, scripts, styles, navigation, and ads, keeping only the main content. This cuts %[11]s of uncompressed HTML down to %[16]s of markdown — a **%[13]s%% reduction** — before any file-level compression is applied. Parquet with Zstd level 19 then compresses the markdown a further %[17]s%%.

End to end: %[10]s of raw gzipped WARCs becomes **%[14]s of Parquet** — a **%[18]s%% total reduction** — containing %[9]s clean markdown documents.
%[19]s
### Personal and Sensitive Information

No additional PII filtering is applied beyond what Common Crawl provides. As the dataset is sourced from the public web, it is likely that some personally identifiable information is present. If you find your own PII in the dataset and would like it removed, please open an issue on the repository.

## Considerations for Using the Data

### Social Impact

By releasing both the dataset and the full processing pipeline, we aim to lower the barrier to training and evaluating language models on high quality web data. Researchers and practitioners who cannot afford to run their own Common Crawl processing pipelines can use Open Index directly.

### Discussion of Biases

Open Index inherits the biases present in Common Crawl and the public web at large. The trafilatura extraction step favors article-like pages and may underrepresent content from forums, social media, and non-standard page layouts. We have not applied any machine-learning-based quality or toxicity filters, as such filters have been shown to disproportionately remove content from certain dialects and communities.

### Known Limitations

Code-heavy pages may not convert well to Markdown. If you are training a model that needs strong code performance, consider supplementing Open Index with a dedicated code dataset such as [The Stack v2](https://huggingface.co/datasets/bigcode/the-stack-v2). Similarly, highly structured pages like Wikipedia may have better formatting in dedicated Wikipedia dumps than in their Common Crawl versions.

## Additional Information

### Licensing

The dataset is released under the **Open Data Commons Attribution License (ODC-By) v1.0**. The use of this dataset is also subject to [Common Crawl's Terms of Use](https://commoncrawl.org/terms-of-use). The original content remains subject to the rights and terms of its respective publishers.

### Contact

Please open a discussion on the [Community tab](https://huggingface.co/datasets/open-index/draft/discussions) for questions, feedback, or issues.
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
	)
}

func ccPublishLicense() string {
	return `Open Data Commons Attribution License (ODC-By) v1.0

Full text: https://opendatacommons.org/licenses/by/1-0/

You are free to copy, distribute, use, modify, transform, and build upon
this database, as long as you attribute the source.

Attribution: "Open Index, derived from Common Crawl (https://commoncrawl.org)"

Note: This dataset contains data derived from Common Crawl, which archives
third-party web content. The original content remains subject to the rights
of its respective publishers. You are responsible for complying with applicable
law including downstream licensing obligations, robots.txt restrictions, privacy
requirements, and content removal requests. See Common Crawl's Terms of Use:
https://commoncrawl.org/terms-of-use
`
}
