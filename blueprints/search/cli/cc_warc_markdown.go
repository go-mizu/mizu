package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
	warcmd "github.com/go-mizu/mizu/blueprints/search/pkg/warc_md"
	"github.com/spf13/cobra"
)

// newCCWarcMarkdown returns the `cc warc markdown` command.
func newCCWarcMarkdown() *cobra.Command {
	var (
		crawlID    string
		fileIdx    string
		workers    int
		force      bool
		fast       bool
		keepTemp   bool
		inMemory   bool
		statusCode int
		mimeFilter string
		maxBody    int64
	)

	cmd := &cobra.Command{
		Use:   "markdown",
		Short: "Convert WARC HTML records to clean Markdown (3-phase pipeline)",
		Long: `3-phase pipeline: Extract → Convert → Compress

  Phase 1  warc/*.warc.gz      → warc_single/**/*.warc      (extract HTML records)
  Phase 2  warc_single/**      → markdown_raw/**/*.md        (HTML → Markdown)
  Phase 3  markdown_raw/**     → markdown/**/*.md.gz         (gzip compress)

After all phases succeed, warc_single/ and markdown_raw/ are removed.
Use --keep-temp to retain them for inspection.
Final output: markdown/**/*.md.gz

In-memory mode (--mem): phases run as a streaming pipeline connected by
channels. No warc_single/ or markdown_raw/ directories are created. Typically
5–15% faster and uses less disk I/O.

WARC record path sharding:
  <urn:uuid:5d0e2270-...ab> → 5d/0e/22/5d0e2270-...ab.warc
  (first 6 hex chars → 3-level directory nesting, ~256 files per leaf dir)

Converters:
  default:  trafilatura (quality, F1=0.91)   ~200–600 docs/s
  --fast:   go-readability (3–8× faster)     ~800–2,000 docs/s

Progress (every 500ms): docs/s · MB/s read · MB/s write · peak RAM
Summary: disk before/after · RAM before/after/peak · per-phase table
`,
		Example: `  search cc warc markdown --file 0
  search cc warc markdown --file 0 --fast
  search cc warc markdown --file 0 --mem
  search cc warc markdown --file 0-4 --workers 16
  search cc warc markdown --file 0 --keep-temp     # inspect warc_single/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCWarcMarkdown(cmd.Context(),
				crawlID, fileIdx, workers, force, fast, keepTemp, inMemory,
				statusCode, mimeFilter, maxBody)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "0", "File index, range (0-9), or all")
	cmd.Flags().IntVar(&workers, "workers", 0, "Parallel workers (0 = NumCPU)")
	cmd.Flags().BoolVar(&force, "force", false, "Re-process existing files in all phases")
	cmd.Flags().BoolVar(&fast, "fast", false, "Use go-readability instead of trafilatura")
	cmd.Flags().BoolVar(&keepTemp, "keep-temp", false, "Keep warc_single/ and markdown/ after pipeline")
	cmd.Flags().BoolVar(&inMemory, "mem", false, "Streaming pipeline with no temp files")
	cmd.Flags().IntVar(&statusCode, "status", 200, "HTTP status filter (0 = all)")
	cmd.Flags().StringVar(&mimeFilter, "mime", "text/html", "MIME type filter")
	cmd.Flags().Int64Var(&maxBody, "max-body", 512*1024, "Max HTML body bytes per record")

	return cmd
}

// warcMDPhaseRow holds per-phase results for the final summary table.
type warcMDPhaseRow struct {
	name    string
	stats   *warcmd.PhaseStats
	diskOut int64
}

func runCCWarcMarkdown(ctx context.Context,
	crawlID, fileIdx string, workers int, force, fast, keepTemp, inMemory bool,
	statusCode int, mimeFilter string, maxBody int64) error {

	// Resolve crawl ID
	resolvedID, note, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedID
	if note != "" {
		ccPrintDefaultCrawlResolution(crawlID, note)
	}

	// Build config
	cfg := warcmd.DefaultConfig(crawlID)
	cfg.Workers = workers
	cfg.Force = force
	cfg.Fast = fast
	cfg.KeepTemp = keepTemp
	cfg.InMemory = inMemory
	cfg.StatusCode = statusCode
	cfg.MIMEFilter = mimeFilter
	cfg.MaxBodySize = maxBody

	// Resolve WARC manifest
	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	selected, err := ccParseFileSelector(fileIdx, len(paths))
	if err != nil {
		return fmt.Errorf("--file: %w", err)
	}

	// Build list of local .warc.gz paths; auto-download if missing
	warcDir := cfg.WARCDir()
	var inputFiles []string
	for _, idx := range selected {
		localPath := filepath.Join(warcDir, filepath.Base(paths[idx]))
		if !fileExists(localPath) {
			if err := downloadWithProgress(ctx, client, paths[idx], localPath); err != nil {
				return fmt.Errorf("downloading %s: %w", filepath.Base(localPath), err)
			}
		}
		inputFiles = append(inputFiles, localPath)
	}

	// Effective worker counts (adaptive if --workers not specified)
	effConvert := cfg.ConvertWorkers()
	effCompress := cfg.CompressWorkers()

	// Print header
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("  WARC → Markdown   3-Phase Pipeline"))
	fmt.Println()

	mode := "file mode (3-phase with temp dirs)"
	if inMemory {
		mode = "in-memory streaming (no temp files)"
	}
	extractor := "trafilatura (quality)"
	if fast {
		extractor = "go-readability (fast)"
	}

	fmt.Printf("  Crawl     %s\n", labelStyle.Render(crawlID))
	fmt.Printf("  Files     %d file(s): %s\n", len(inputFiles), labelStyle.Render(filepath.Base(inputFiles[0])))
	fmt.Printf("  Mode      %s\n", infoStyle.Render(mode))
	fmt.Printf("  Engine    %s\n", infoStyle.Render(extractor))
	if workers > 0 {
		fmt.Printf("  Workers   %s\n", infoStyle.Render(fmt.Sprintf("%d (manual)", workers)))
	} else {
		fmt.Printf("  Workers   convert=%d  compress=%d  (adaptive)\n", effConvert, effCompress)
	}
	fmt.Printf("  Output    %s\n", labelStyle.Render(cfg.MarkdownGzDir()))
	if force {
		fmt.Printf("  Force     %s\n", warningStyle.Render("re-process all files"))
	}
	fmt.Println()

	if inMemory {
		return runWARCMDInMemory(ctx, cfg, inputFiles, effConvert)
	}
	return runWARCMDFileMode(ctx, cfg, inputFiles, effConvert, effCompress)
}

// runWARCMDFileMode runs the 3-phase file-based pipeline.
func runWARCMDFileMode(ctx context.Context, cfg warcmd.Config, inputFiles []string, convertWorkers, compressWorkers int) error {
	var rows []warcMDPhaseRow
	pipeStart := time.Now()

	// ── Phase 1 — Extract ────────────────────────────────────────────────────
	fmt.Println(subtitleStyle.Render("  Phase 1 / 3 — Extract   warc.gz → warc_single/**/*.warc"))
	fmt.Println()
	memBef1 := memSysMB()

	s1, err := warcmd.RunExtract(ctx, warcmd.ExtractConfig{
		InputFiles:  inputFiles,
		OutputDir:   cfg.WARCSingleDir(),
		Workers:     len(inputFiles),
		Force:       cfg.Force,
		StatusCode:  cfg.StatusCode,
		MIMEFilter:  cfg.MIMEFilter,
		MaxBodySize: cfg.MaxBodySize,
	}, func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64) {
		mdPhaseProgress("Extracting", done, total, errors, readBytes, writeBytes, elapsed, peakMemMB)
	})
	fmt.Printf("\r\033[K")
	if err != nil {
		return fmt.Errorf("phase 1 extract: %w", err)
	}
	disk1 := warcmd.DiskUsageBytes(cfg.WARCSingleDir())
	mdPhaseEnd("Extract", toMDPhaseStats(s1), memBef1, memSysMB(), disk1, "warc_single")
	rows = append(rows, warcMDPhaseRow{"Extract", s1, disk1})
	fmt.Println()

	// ── Phase 2 — Convert ────────────────────────────────────────────────────
	fmt.Println(subtitleStyle.Render("  Phase 2 / 3 — Convert   warc_single/**/*.warc → markdown_raw/**/*.md"))
	fmt.Println()
	memBef2 := memSysMB()

	s2, err := warcmd.RunConvert(ctx, warcmd.ConvertConfig{
		InputDir:  cfg.WARCSingleDir(),
		OutputDir: cfg.MarkdownDir(),
		IndexPath: cfg.IndexPath(),
		Workers:   convertWorkers,
		Force:     cfg.Force,
		BatchSize: 1000,
		Fast:      cfg.Fast,
	}, func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64) {
		mdPhaseProgress("Converting", done, total, errors, readBytes, writeBytes, elapsed, peakMemMB)
	})
	fmt.Printf("\r\033[K")
	if err != nil {
		return fmt.Errorf("phase 2 convert: %w", err)
	}
	disk2 := warcmd.DiskUsageBytes(cfg.MarkdownDir())
	mdPhaseEnd("Convert", toMDPhaseStats(s2), memBef2, memSysMB(), disk2, "markdown_raw")
	rows = append(rows, warcMDPhaseRow{"Convert", s2, disk2})
	fmt.Println()

	// ── Phase 3 — Compress ───────────────────────────────────────────────────
	fmt.Println(subtitleStyle.Render("  Phase 3 / 3 — Compress  markdown_raw/**/*.md → markdown/**/*.md.gz"))
	fmt.Println()
	memBef3 := memSysMB()

	s3, err := warcmd.RunCompress(ctx, warcmd.CompressConfig{
		InputDir:  cfg.MarkdownDir(),
		OutputDir: cfg.MarkdownGzDir(),
		Workers:   compressWorkers,
		Force:     cfg.Force,
	}, func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64) {
		mdPhaseProgress("Compressing", done, total, errors, readBytes, writeBytes, elapsed, peakMemMB)
	})
	fmt.Printf("\r\033[K")
	if err != nil {
		return fmt.Errorf("phase 3 compress: %w", err)
	}
	disk3 := warcmd.DiskUsageBytes(cfg.MarkdownGzDir())
	mdPhaseEnd("Compress", toMDPhaseStats(s3), memBef3, memSysMB(), disk3, "markdown")
	rows = append(rows, warcMDPhaseRow{"Compress", s3, disk3})
	fmt.Println()

	totalDuration := time.Since(pipeStart)

	// ── Final summary table ───────────────────────────────────────────────────
	printWarcMDSummary(rows, s1, s3, disk1, disk2, disk3, totalDuration)

	// ── Cleanup temp dirs ─────────────────────────────────────────────────────
	if !cfg.KeepTemp {
		fmt.Printf("  Cleaning up temp dirs...\n")
		os.RemoveAll(cfg.WARCSingleDir())
		os.RemoveAll(cfg.MarkdownDir())
		fmt.Printf("  %s  removed warc_single/ and markdown_raw/\n", successStyle.Render("✓"))
	} else {
		fmt.Printf("  %s  warc_single/ and markdown_raw/ kept\n", infoStyle.Render("ℹ"))
	}
	fmt.Println()

	// ── Convert error breakdown ───────────────────────────────────────────────
	if s2.Errors > 0 {
		cats, qerr := markdown.QueryErrors(cfg.IndexPath())
		if qerr == nil && len(cats) > 0 {
			fmt.Println(subtitleStyle.Render("  Convert Error Breakdown:"))
			fmt.Println()
			for _, c := range cats {
				pct := float64(c.Count) / float64(s2.Errors) * 100
				fmt.Printf("    %-32s %s (%.1f%%)\n",
					c.Category, warningStyle.Render(ccFmtInt64(int64(c.Count))), pct)
			}
			fmt.Println()
		}
	}

	return nil
}

// runWARCMDInMemory runs the streaming in-memory pipeline.
func runWARCMDInMemory(ctx context.Context, cfg warcmd.Config, inputFiles []string, workers int) error {
	cfg.Workers = workers
	fmt.Println(subtitleStyle.Render("  Pipeline  warc.gz → [warcCh] → convert → [mdCh] → markdown/"))
	fmt.Println()
	memBef := memSysMB()
	pipeStart := time.Now()

	result, err := warcmd.RunInMemoryPipeline(ctx, cfg, inputFiles,
		func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64) {
			mdPhaseProgress("Pipeline", done, total, errors, readBytes, writeBytes, elapsed, peakMemMB)
		})
	fmt.Printf("\r\033[K")
	if err != nil {
		return fmt.Errorf("in-memory pipeline: %w", err)
	}

	memAft := memSysMB()
	totalDuration := time.Since(pipeStart)
	disk3 := warcmd.DiskUsageBytes(cfg.MarkdownGzDir())

	// In-memory mode has no intermediate disk stages; we know read bytes (HTML)
	// and write bytes (compressed), but not the plain markdown size.
	htmlBytes := result.Extract.ReadBytes
	gzSave := float64(0)
	if htmlBytes > 0 {
		gzSave = (1 - float64(disk3)/float64(htmlBytes)) * 100
	}

	fmt.Printf("\n  %s in-memory pipeline done\n", successStyle.Render("✓"))
	fmt.Printf("    Extracted  %s HTML records\n", infoStyle.Render(ccFmtInt64(result.Extract.Files)))
	fmt.Printf("    Converted  %s to Markdown\n", infoStyle.Render(ccFmtInt64(result.Convert.Files)))
	fmt.Printf("    Compressed %s → markdown/  (%s)\n",
		infoStyle.Render(ccFmtInt64(result.Compress.Files)), formatBytes(disk3))
	if result.Extract.Errors > 0 {
		fmt.Printf("    Errors     %s\n", warningStyle.Render(ccFmtInt64(result.Extract.Errors)))
	}
	rate := float64(0)
	readMBs := float64(0)
	writeMBs := float64(0)
	if totalDuration.Seconds() > 0 {
		rate = float64(result.Compress.Files) / totalDuration.Seconds()
		readMBs = float64(htmlBytes) / (1024 * 1024) / totalDuration.Seconds()
		writeMBs = float64(result.Compress.WriteBytes) / (1024 * 1024) / totalDuration.Seconds()
	}
	fmt.Printf("    Rate       %.0f docs/s  ·  %.1f MB/s read  ·  %.1f MB/s write\n",
		rate, readMBs, writeMBs)
	fmt.Printf("    Time       %s\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("    RAM        before %.0f MB → after %.0f MB  (peak %.0f MB)\n",
		memBef, memAft, result.Compress.PeakMemMB)
	fmt.Println()

	// Compression comparison
	fmt.Println(subtitleStyle.Render("  Output  markdown/  (" + ccFmtInt64(result.Extract.Files) + " records)"))
	fmt.Println()
	fmt.Printf("  %-20s  %10s  %8s\n", "Stage", "Size", "vs HTML")
	fmt.Printf("  %-20s  %10s  %8s\n", "--------------------", "----------", "--------")
	fmt.Printf("  %-20s  %10s  %8s\n", "HTML body (read)", formatBytes(htmlBytes), "baseline")
	fmt.Printf("  %-20s  %10s  %+7.1f%%  (%.1fx smaller)\n",
		"markdown/ (.md.gz)", formatBytes(disk3), -gzSave, float64(htmlBytes)/float64(disk3))
	fmt.Println()

	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

// toMDPhaseStats adapts warcmd.PhaseStats to markdown.PhaseStats for use with
// the shared display helpers in cc_markdown.go (mdPhaseEnd).
func toMDPhaseStats(s *warcmd.PhaseStats) *markdown.PhaseStats {
	if s == nil {
		return &markdown.PhaseStats{}
	}
	return &markdown.PhaseStats{
		Files:      s.Files,
		Skipped:    s.Skipped,
		Errors:     s.Errors,
		ReadBytes:  s.ReadBytes,
		WriteBytes: s.WriteBytes,
		PeakMemMB:  s.PeakMemMB,
		Duration:   s.Duration,
	}
}

// printWarcMDSummary prints the final per-phase table, disk compression stats.
// disk1 = warc_single (raw HTML), disk2 = markdown_raw (plain .md), disk3 = markdown (compressed .md.gz).
func printWarcMDSummary(rows []warcMDPhaseRow, s1, s3 *warcmd.PhaseStats, disk1, disk2, disk3 int64, totalDuration time.Duration) {
	fmt.Println(successStyle.Render("  ✓ All phases complete!"))
	fmt.Println()
	mdSep()
	fmt.Printf("  %-10s  %8s  %8s  %8s  %8s  %8s  %7s  %s\n",
		"Phase", "Files", "Read", "Write", "Disk out", "Rate", "RAM pk", "Time")
	mdSep()

	var totalRead int64
	var peakRAM float64
	for _, r := range rows {
		s := r.stats
		rate := float64(0)
		if s.Duration.Seconds() > 0 && (s.Files > 0 || s.Errors > 0) {
			rate = float64(s.Files+s.Errors) / s.Duration.Seconds()
		}
		if s.PeakMemMB > peakRAM {
			peakRAM = s.PeakMemMB
		}
		totalRead += s.ReadBytes
		fmt.Printf("  %-10s  %8s  %8s  %8s  %8s  %7.0f/s  %6.0fMB  %s\n",
			r.name,
			ccFmtInt64(s.Files+s.Skipped),
			formatBytes(s.ReadBytes),
			formatBytes(s.WriteBytes),
			formatBytes(r.diskOut),
			rate,
			s.PeakMemMB,
			s.Duration.Round(time.Millisecond),
		)
	}

	mdSep()
	overallRate := float64(0)
	totalFiles := int64(0)
	if s1 != nil {
		totalFiles = s1.Files + s1.Skipped
		if totalDuration.Seconds() > 0 {
			overallRate = float64(totalFiles) / totalDuration.Seconds()
		}
	}
	var s3Write int64
	if s3 != nil {
		s3Write = s3.WriteBytes
	}
	fmt.Printf("  %-10s  %8s  %8s  %8s  %8s  %7.0f/s  %6.0fMB  %s\n",
		"Total",
		ccFmtInt64(totalFiles),
		formatBytes(totalRead),
		formatBytes(s3Write),
		formatBytes(disk3),
		overallRate,
		peakRAM,
		totalDuration.Round(time.Millisecond),
	)
	mdSep()
	fmt.Println()

	// Compression comparison table
	fmt.Println(subtitleStyle.Render("  Output  markdown/  (" + ccFmtInt64(totalFiles) + " records)"))
	fmt.Println()
	if disk1 > 0 {
		mdRatio := float64(0)
		gzRatio := float64(0)
		mdSave := float64(0)
		gzSave := float64(0)
		if disk1 > 0 {
			mdRatio = float64(disk2) / float64(disk1) * 100
			gzRatio = float64(disk3) / float64(disk1) * 100
			mdSave = 100 - mdRatio
			gzSave = 100 - gzRatio
		}
		fmt.Printf("  %-20s  %10s  %8s\n", "Stage", "Size", "vs HTML")
		fmt.Printf("  %-20s  %10s  %8s\n", "--------------------", "----------", "--------")
		fmt.Printf("  %-20s  %10s  %8s\n", "warc_single/ (HTML)", formatBytes(disk1), "baseline")
		if disk2 > 0 {
			fmt.Printf("  %-20s  %10s  %+7.1f%%  (%.1fx smaller)\n",
				"markdown_raw/ (.md)", formatBytes(disk2), -mdSave, float64(disk1)/float64(disk2))
		}
		fmt.Printf("  %-20s  %10s  %+7.1f%%  (%.1fx smaller)\n",
			"markdown/ (.md.gz)", formatBytes(disk3), -gzSave, float64(disk1)/float64(disk3))
		fmt.Println()
	} else {
		fmt.Printf("  markdown/   %s\n", formatBytes(disk3))
		fmt.Println()
	}
}

// ── download progress ─────────────────────────────────────────────────────────

// downloadWithProgress downloads a WARC file and prints a live progress line:
//
//	↓ filename.warc.gz  [████████░░░░░░░░░░░░]  45.3%  234.5/512.0 MB  12.3 MB/s  ETA 23s
func downloadWithProgress(ctx context.Context, client *cc.Client, remotePath, localPath string) error {
	name := filepath.Base(localPath)
	fmt.Printf("  %s  %s\n", labelStyle.Render("↓"), name)

	start := time.Now()

	// Bar geometry: fixed 20-cell width.
	const barWidth = 20

	progress := func(received, total int64) {
		elapsed := time.Since(start).Seconds()
		speedMBs := float64(received) / (1024 * 1024) / elapsed

		var bar, pctStr, etaStr string
		if total > 0 {
			pct := float64(received) / float64(total)
			filled := int(pct * barWidth)
			if filled > barWidth {
				filled = barWidth
			}
			bar = "[" + repeatChar('█', filled) + repeatChar('░', barWidth-filled) + "]"
			pctStr = fmt.Sprintf("%5.1f%%", pct*100)
			remaining := float64(total-received) / (1024 * 1024) / speedMBs
			if speedMBs > 0 {
				etaStr = "  ETA " + fmtDuration(remaining)
			}
		} else {
			// Unknown total: show spinner + bytes received
			bar = "[" + repeatChar('█', barWidth) + "]"
			pctStr = "  ?  "
		}

		recvMB := float64(received) / (1024 * 1024)
		totMB := float64(total) / (1024 * 1024)
		var sizeStr string
		if total > 0 {
			sizeStr = fmt.Sprintf("  %6.1f/%6.1f MB", recvMB, totMB)
		} else {
			sizeStr = fmt.Sprintf("  %6.1f MB", recvMB)
		}

		fmt.Printf("\r\033[K  %s %s  %6.1f MB/s%s%s",
			bar, pctStr, speedMBs, sizeStr, etaStr)
	}

	if err := client.DownloadFile(ctx, remotePath, localPath, progress); err != nil {
		fmt.Printf("\r\033[K  %s %s\n", warningStyle.Render("✗"), name)
		return err
	}

	elapsed := time.Since(start)
	if fi, err := os.Stat(localPath); err == nil {
		avgMBs := float64(fi.Size()) / (1024 * 1024) / elapsed.Seconds()
		fmt.Printf("\r\033[K  %s %s  (%s  avg %.1f MB/s  %s)\n",
			successStyle.Render("✓"), name,
			formatBytes(fi.Size()), avgMBs, elapsed.Round(time.Millisecond))
	} else {
		fmt.Printf("\r\033[K  %s %s\n", successStyle.Render("✓"), name)
	}
	return nil
}

// repeatChar returns a string of n copies of ch.
func repeatChar(ch rune, n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]rune, n)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}

// fmtDuration formats seconds as "1h23m", "4m12s", or "45s".
func fmtDuration(secs float64) string {
	s := int(secs)
	if s < 0 {
		s = 0
	}
	if s >= 3600 {
		return fmt.Sprintf("%dh%02dm", s/3600, (s%3600)/60)
	}
	if s >= 60 {
		return fmt.Sprintf("%dm%02ds", s/60, s%60)
	}
	return fmt.Sprintf("%ds", s)
}
