package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
	warcmd "github.com/go-mizu/mizu/blueprints/search/pkg/warc_md"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// newCCWarcMarkdown returns the `cc warc markdown` command.
func newCCWarcMarkdown() *cobra.Command {
	var (
		crawlID    string
		fileIdx    string
		from       int
		to         int
		workers    int
		force      bool
		keepTemp   bool
		statusCode int
		mimeFilter string
		maxBody    int64
	)

	cmd := &cobra.Command{
		Use:   "markdown",
		Short: "Convert WARC HTML records to clean Markdown (2-phase pipeline)",
		Long: `2-phase pipeline: Extract → Convert

  Phase 1  warc/*.warc.gz      → warc_single/**/*.warc      (extract HTML records)
  Phase 2  warc_single/**      → markdown/{warcIdx}/**/*.md  (HTML → Markdown)

After all phases succeed, warc_single/ is removed.
Use --keep-temp to retain it for inspection.
Final output: markdown/{warcIdx}/**/*.md

Parallel mode (--from/--to + --workers N): processes N files concurrently.
Each pipeline uses max(1, NumCPU/N) goroutines for HTML→Markdown conversion.

Note: prefer "cc warc pack" which produces seekable .md.warc.gz files directly.

Progress (every 500ms): docs/s · MB/s read · MB/s write · peak RAM
Summary: disk before/after · RAM before/after/peak · per-phase table
`,
		Example: `  search cc warc markdown --file 0
  search cc warc markdown --file 0-4 --workers 16
  search cc warc markdown --from 1 --to 20 --workers 10   # 10 parallel files
  search cc warc markdown --file 0 --keep-temp            # inspect warc_single/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCWarcMarkdown(cmd.Context(),
				crawlID, fileIdx, from, to, workers, force, keepTemp,
				statusCode, mimeFilter, maxBody)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "0", "File index, range (0-9), or all")
	cmd.Flags().IntVar(&from, "from", -1, "First file index (inclusive) for parallel range")
	cmd.Flags().IntVar(&to, "to", -1, "Last file index (inclusive) for parallel range")
	cmd.Flags().IntVar(&workers, "workers", 0, "Goroutines per file (single-file) or parallel files (multi-file, 0 = NumCPU)")
	cmd.Flags().BoolVar(&force, "force", false, "Re-process existing files in all phases")
	cmd.Flags().BoolVar(&keepTemp, "keep-temp", false, "Keep warc_single/ after pipeline")
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
	crawlID, fileIdx string, from, to, workers int, force, keepTemp bool,
	statusCode int, mimeFilter string, maxBody int64) error {

	// --from/--to overrides --file when both indices are provided
	if from >= 0 && to >= 0 {
		fileIdx = fmt.Sprintf("%d-%d", from, to)
	}

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
	cfg.KeepTemp = keepTemp
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

	// Parallel mode: multiple files + workers > 1 → run workers files concurrently.
	parallelMode := len(inputFiles) > 1 && workers > 1

	// Effective worker count for per-file convert
	effConvert := cfg.ConvertWorkers()

	// Print header
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("  WARC → Markdown   2-Phase Pipeline"))
	fmt.Println()

	fmt.Printf("  Crawl     %s\n", labelStyle.Render(crawlID))
	if len(inputFiles) == 1 {
		fmt.Printf("  Files     1 file: %s\n", labelStyle.Render(filepath.Base(inputFiles[0])))
	} else {
		fmt.Printf("  Files     %d files  [%s … %s]\n",
			len(inputFiles),
			labelStyle.Render(filepath.Base(inputFiles[0])),
			labelStyle.Render(filepath.Base(inputFiles[len(inputFiles)-1])))
	}
	fmt.Printf("  Engine    %s\n", infoStyle.Render("trafilatura"))
	if force {
		fmt.Printf("  Force     %s\n", warningStyle.Render("re-process all files"))
	}

	if parallelMode {
		perFile := max(1, runtime.NumCPU()/workers)
		fmt.Printf("  Mode      %s\n", infoStyle.Render("parallel per-file pipelines"))
		fmt.Printf("  Workers   %d parallel files · %d goroutines/file\n", workers, perFile)
		fmt.Printf("  Output    %s\n", labelStyle.Render(cfg.MarkdownWarcDir("<warcIdx>")))
		fmt.Println()
		return runWARCMDParallelFiles(ctx, cfg, inputFiles, workers)
	}

	fmt.Printf("  Mode      %s\n", infoStyle.Render("per-file 2-phase pipeline"))
	if workers > 0 {
		fmt.Printf("  Workers   %s\n", infoStyle.Render(fmt.Sprintf("%d (manual)", workers)))
	} else {
		fmt.Printf("  Workers   convert=%d  (adaptive)\n", effConvert)
	}
	fmt.Printf("  Output    %s\n", labelStyle.Render(cfg.MarkdownWarcDir("<warcIdx>")))
	fmt.Println()

	return runWARCMDSequential(ctx, cfg, inputFiles, paths, selected)
}

// runWARCMDSequential processes each WARC file sequentially using RunFilePipeline.
func runWARCMDSequential(ctx context.Context, cfg warcmd.Config, inputFiles []string, paths []string, selected []int) error {
	var rows []warcMDPhaseRow
	pipeStart := time.Now()

	for i, localPath := range inputFiles {
		warcIdx := warcIndexFromPath(paths[selected[i]], selected[i])
		fname := filepath.Base(localPath)

		fmt.Printf("%s\n", subtitleStyle.Render(fmt.Sprintf("  File [%d/%d]  %s  (warcIdx=%s)", i+1, len(inputFiles), fname, warcIdx)))
		fmt.Println()

		memBef := memSysMB()

		result, err := warcmd.RunFilePipeline(ctx, cfg, warcIdx, []string{localPath},
			func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64) {
				mdPhaseProgress("Extracting", done, total, errors, readBytes, writeBytes, elapsed, peakMemMB)
			},
			func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64) {
				mdPhaseProgress("Converting", done, total, errors, readBytes, writeBytes, elapsed, peakMemMB)
			},
		)
		fmt.Printf("\r\033[K")
		if err != nil {
			return fmt.Errorf("pipeline for %s: %w", fname, err)
		}

		diskOut := warcmd.DiskUsageBytes(cfg.MarkdownWarcDir(warcIdx))
		memAft := memSysMB()

		// Print phase summaries
		if result.Extract != nil {
			mdPhaseEnd("Extract", toMDPhaseStats(result.Extract), memBef, memSysMB(), warcmd.DiskUsageBytes(cfg.WARCSingleDir()), "warc_single")
		}
		if result.Convert != nil {
			mdPhaseEnd("Convert", toMDPhaseStats(result.Convert), memAft, memSysMB(), diskOut, "markdown/"+warcIdx)
		}
		fmt.Println()

		rows = append(rows, warcMDPhaseRow{"Convert", result.Convert, diskOut})
	}

	totalDuration := time.Since(pipeStart)

	// ── Final summary ─────────────────────────────────────────────────────────
	printWarcMDSummary(rows, totalDuration)

	return nil
}

// runWARCMDParallelFiles runs N file pipelines concurrently, one per WARC file.
// Per-pipeline goroutines = max(1, NumCPU/parallelN).
func runWARCMDParallelFiles(ctx context.Context, cfg warcmd.Config, inputFiles []string, parallelN int) error {
	perFileWorkers := max(1, runtime.NumCPU()/parallelN)
	cfg.Workers = perFileWorkers

	var (
		mu           sync.Mutex
		totalExtract atomic.Int64
		totalConvert atomic.Int64
		totalErrors  atomic.Int64
		totalRead    atomic.Int64
		totalWrite   atomic.Int64
	)

	pipeStart := time.Now()

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(parallelN)

	for i, warcPath := range inputFiles {
		i, warcPath := i, warcPath
		g.Go(func() error {
			fname := filepath.Base(warcPath)
			warcIdx := warcIndexFromPath(warcPath, i)

			mu.Lock()
			fmt.Printf("  [%d/%d] %s  starting\n", i+1, len(inputFiles), fname)
			mu.Unlock()

			result, err := warcmd.RunFilePipeline(gctx, cfg, warcIdx, []string{warcPath}, nil, nil)

			mu.Lock()
			if result != nil {
				if result.Extract != nil {
					totalExtract.Add(result.Extract.Files)
					totalErrors.Add(result.Extract.Errors)
					totalRead.Add(result.Extract.ReadBytes)
				}
				if result.Convert != nil {
					totalConvert.Add(result.Convert.Files)
					totalWrite.Add(result.Convert.WriteBytes)
				}
				extractFiles := int64(0)
				convertFiles := int64(0)
				if result.Extract != nil {
					extractFiles = result.Extract.Files
				}
				if result.Convert != nil {
					convertFiles = result.Convert.Files
				}
				fmt.Printf("  [%d/%d] %s  extract=%-6s  convert=%-6s  %s\n",
					i+1, len(inputFiles), fname,
					ccFmtInt64(extractFiles),
					ccFmtInt64(convertFiles),
					result.Duration.Round(time.Millisecond))
			}
			if err != nil {
				fmt.Printf("  [%d/%d] %s  %s\n", i+1, len(inputFiles), fname,
					warningStyle.Render("error: "+err.Error()))
			}
			mu.Unlock()

			return err
		})
	}

	gerr := g.Wait()
	totalDuration := time.Since(pipeStart)

	fmt.Println()
	fmt.Println(successStyle.Render("  ✓ All files complete!"))
	fmt.Println()

	rate := float64(0)
	if totalDuration.Seconds() > 0 {
		rate = float64(totalConvert.Load()) / totalDuration.Seconds()
	}
	fmt.Printf("  Files      %d processed  (%d parallel)\n", len(inputFiles), parallelN)
	fmt.Printf("  Extracted  %s HTML records\n", infoStyle.Render(ccFmtInt64(totalExtract.Load())))
	fmt.Printf("  Converted  %s to Markdown\n", infoStyle.Render(ccFmtInt64(totalConvert.Load())))
	if totalErrors.Load() > 0 {
		fmt.Printf("  Errors     %s\n", warningStyle.Render(ccFmtInt64(totalErrors.Load())))
	}
	readMBs := float64(totalRead.Load()) / (1024 * 1024) / totalDuration.Seconds()
	writeMBs := float64(totalWrite.Load()) / (1024 * 1024) / totalDuration.Seconds()
	fmt.Printf("  Rate       %.0f docs/s  ·  %.1f MB/s read  ·  %.1f MB/s write\n",
		rate, readMBs, writeMBs)
	fmt.Printf("  Time       %s\n", totalDuration.Round(time.Millisecond))
	fmt.Println()

	return gerr
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

// printWarcMDSummary prints the final per-file table after all files are processed.
func printWarcMDSummary(rows []warcMDPhaseRow, totalDuration time.Duration) {
	if len(rows) == 0 {
		return
	}

	fmt.Println(successStyle.Render("  ✓ All files complete!"))
	fmt.Println()
	mdSep()
	fmt.Printf("  %-10s  %8s  %8s  %8s  %8s  %7s  %s\n",
		"File", "Files", "Read", "Write", "Disk out", "Rate", "Time")
	mdSep()

	var totalFiles, totalRead, totalWrite, totalDisk int64
	var peakRAM float64
	for i, r := range rows {
		s := r.stats
		if s == nil {
			continue
		}
		rate := float64(0)
		if s.Duration.Seconds() > 0 {
			rate = float64(s.Files+s.Skipped+s.Errors) / s.Duration.Seconds()
		}
		if s.PeakMemMB > peakRAM {
			peakRAM = s.PeakMemMB
		}
		totalFiles += s.Files + s.Skipped
		totalRead += s.ReadBytes
		totalWrite += s.WriteBytes
		totalDisk += r.diskOut
		fmt.Printf("  %-10s  %8s  %8s  %8s  %8s  %7.0f/s  %s\n",
			fmt.Sprintf("[%d]", i+1),
			ccFmtInt64(s.Files+s.Skipped),
			formatBytes(s.ReadBytes),
			formatBytes(s.WriteBytes),
			formatBytes(r.diskOut),
			rate,
			s.Duration.Round(time.Millisecond),
		)
	}

	mdSep()
	overallRate := float64(0)
	if totalDuration.Seconds() > 0 {
		overallRate = float64(totalFiles) / totalDuration.Seconds()
	}
	fmt.Printf("  %-10s  %8s  %8s  %8s  %8s  %7.0f/s  %s\n",
		"Total",
		ccFmtInt64(totalFiles),
		formatBytes(totalRead),
		formatBytes(totalWrite),
		formatBytes(totalDisk),
		overallRate,
		totalDuration.Round(time.Millisecond),
	)
	mdSep()
	fmt.Println()
}

// ── download progress ─────────────────────────────────────────────────────────

// downloadWithProgress downloads a WARC file with a live progress bar.
// Retries up to 3 times on HTTP 503 or download stall (no bytes for 10 min).
//
//	↓ filename.warc.gz  [████████░░░░░░░░░░░░]  45.3%  234.5/512.0 MB  12.3 MB/s  ETA 23s
func downloadWithProgress(ctx context.Context, client *cc.Client, remotePath, localPath string) error {
	const maxAttempts = 3
	const stallTimeout = 10 * time.Minute
	const barWidth = 20

	name := filepath.Base(localPath)
	fmt.Printf("  %s  %s\n", labelStyle.Render("↓"), name)

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Delete partial file so next attempt starts fresh.
			os.Remove(localPath)
			backoff := time.Duration(attempt*30) * time.Second
			fmt.Printf("  %s attempt %d/%d in %s: %v\n",
				warningStyle.Render("↻"), attempt+1, maxAttempts, backoff, lastErr)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Stall detector: cancel download if no bytes received for stallTimeout.
		stallCtx, cancelStall := context.WithCancel(ctx)
		var stallMu sync.Mutex
		lastActivity := time.Now()
		var lastBytes int64
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-stallCtx.Done():
					return
				case <-ticker.C:
					stallMu.Lock()
					stalled := time.Since(lastActivity) > stallTimeout
					stallMu.Unlock()
					if stalled {
						fmt.Printf("\r\033[K  %s stalled (%s no progress), cancelling\n",
							warningStyle.Render("↻"), stallTimeout)
						cancelStall()
						return
					}
				}
			}
		}()

		start := time.Now()
		progress := func(received, total int64) {
			stallMu.Lock()
			if received > lastBytes {
				lastBytes = received
				lastActivity = time.Now()
			}
			stallMu.Unlock()

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
				if speedMBs > 0 {
					remaining := float64(total-received) / (1024 * 1024) / speedMBs
					etaStr = "  ETA " + fmtDuration(remaining)
				}
			} else {
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
			fmt.Printf("\r\033[K  %s %s  %6.1f MB/s%s%s", bar, pctStr, speedMBs, sizeStr, etaStr)
		}

		err := client.DownloadFile(stallCtx, remotePath, localPath, progress)
		cancelStall()

		if err == nil {
			elapsed := time.Since(start)
			if fi, statErr := os.Stat(localPath); statErr == nil {
				avgMBs := float64(fi.Size()) / (1024 * 1024) / elapsed.Seconds()
				fmt.Printf("\r\033[K  %s %s  (%s  avg %.1f MB/s  %s)\n",
					successStyle.Render("✓"), name,
					formatBytes(fi.Size()), avgMBs, elapsed.Round(time.Millisecond))
			} else {
				fmt.Printf("\r\033[K  %s %s\n", successStyle.Render("✓"), name)
			}
			return nil
		}

		lastErr = err
		// Retry on 503 or stall (context cancelled by stall detector, not by caller).
		isRetryable := strings.Contains(err.Error(), "HTTP 503") ||
			(stallCtx.Err() != nil && ctx.Err() == nil)
		if !isRetryable {
			break
		}
	}

	fmt.Printf("\r\033[K  %s %s\n", warningStyle.Render("✗"), name)
	return lastErr
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
