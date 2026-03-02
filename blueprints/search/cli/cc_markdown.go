package cli

import (
	"context"
	"io/fs"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
	"github.com/spf13/cobra"
)

func newCCMarkdown() *cobra.Command {
	var (
		crawlID          string
		bodyStore        string
		workers          int
		force            bool
		fast             bool
		cpuProfile       string
		phases           bool
		keepIntermediate bool
		inMemory         bool
	)

	cmd := &cobra.Command{
		Use:   "markdown",
		Short: "Convert CC HTML bodies → clean markdown (3-phase pipeline)",
		Long: `3-phase pipeline: Extract → Convert → Compress

  Phase 1  bodies/*.gz → html/*.html    decompress gzip
  Phase 2  html/*.html → md/*.md         HTML → Markdown (trafilatura or go-readability)
  Phase 3  md/*.md     → md-gz/*.md.gz   re-compress with gzip

Before Phase 2, auto-benchmarks 8 / 16 / 32 / 64 / 128 / 256 workers on a
200-file sample and selects the fastest count for this machine.
Override with --workers N to skip the benchmark.

Per-phase progress: docs/s · MB/s read · MB/s write · peak RAM
Per-phase summary:  disk used · RAM before → after (peak)
Final table:        all phases side-by-side

  Default:  trafilatura (F1=0.91) — quality extraction, ~80–200 files/s
  --fast:   go-readability          — 3–8× faster, slightly lower quality

Directories (relative to body-store parent):
  html/            raw HTML — delete after convert to free space
  md/              markdown files
  md-gz/           gzipped markdown
  md/index.duckdb  conversion index (size, tokens, timing, errors)
`,
		Example: `  search cc markdown --phases
  search cc markdown --phases --fast
  search cc markdown --phases --workers 32     # skip auto-tune, use 32 workers
  search cc markdown --phases --force          # re-process all files
  search cc markdown --mem                     # streaming, no temp dirs
  search cc markdown --mem --fast              # streaming + go-readability`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !phases && !inMemory {
				return cmd.Help()
			}
			if inMemory {
				return runCCMarkdownPipeline(cmd.Context(), crawlID, bodyStore, workers, force, fast, cpuProfile)
			}
			return runCCMarkdownPhases(cmd.Context(), crawlID, bodyStore, workers, force, fast, keepIntermediate, cpuProfile)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&bodyStore, "body-store", "", "Body store directory (default: ~/data/common-crawl/bodies)")
	cmd.Flags().IntVar(&workers, "workers", 0, "Parallel workers (0 = auto-tune via benchmark)")
	cmd.Flags().BoolVar(&force, "force", false, "Re-process existing files in all phases")
	cmd.Flags().BoolVar(&fast, "fast", false, "Use go-readability instead of trafilatura (3–8× faster)")
	cmd.Flags().StringVar(&cpuProfile, "cpuprofile", "", "Write CPU profile (analyze: go tool pprof <file>)")
	cmd.Flags().BoolVar(&phases, "phases", false, "Run 3-phase pipeline with per-phase stats and worker auto-tune")
	cmd.Flags().BoolVar(&keepIntermediate, "keep-intermediate", false, "Keep html/ and md/*.md files after pipeline (default: auto-delete)")
	cmd.Flags().BoolVar(&inMemory, "mem", false, "Streaming pipeline: no intermediate html/ or md/ dirs (fastest)")

	return cmd
}

// ─── helpers ────────────────────────────────────────────────────────────────

// memSysMB returns current total OS-allocated process memory in MB.
func memSysMB() float64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return float64(ms.Sys) / (1024 * 1024)
}

// diskUsageBytes sums the sizes of all regular files under path.
func diskUsageBytes(path string) int64 {
	var total int64
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func mdSep() {
	fmt.Println(labelStyle.Render("  ────────────────────────────────────────────────────────────────"))
}

// ─── per-phase display ───────────────────────────────────────────────────────

// mdPhaseProgress writes a single-line live progress update.
// When total=0 (unknown), only the done count is shown without percentage.
// When writeBytes=0, the W: column is omitted to avoid misleading "W:0.0 MB/s".
func mdPhaseProgress(verb string, done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64) {
	rate := float64(0)
	readMBs := float64(0)
	writeMBs := float64(0)
	if elapsed.Seconds() > 0 {
		rate = float64(done) / elapsed.Seconds()
		readMBs = float64(readBytes) / (1024 * 1024) / elapsed.Seconds()
		writeMBs = float64(writeBytes) / (1024 * 1024) / elapsed.Seconds()
	}

	var progress string
	if total > 0 {
		pct := float64(done) / float64(total) * 100
		progress = fmt.Sprintf("%s/%s (%.1f%%)", ccFmtInt64(done), ccFmtInt64(total), pct)
	} else {
		progress = ccFmtInt64(done)
	}

	if writeBytes > 0 {
		fmt.Printf("\r\033[K  %s: %s  %.0f docs/s  R:%.1f MB/s  W:%.1f MB/s  Mem:%.0f MB",
			verb, progress, rate, readMBs, writeMBs, peakMemMB)
	} else {
		fmt.Printf("\r\033[K  %s: %s  %.0f docs/s  R:%.1f MB/s  Mem:%.0f MB",
			verb, progress, rate, readMBs, peakMemMB)
	}
	if errors > 0 {
		fmt.Printf("  err:%s", ccFmtInt64(errors))
	}
}

// mdPhaseEnd prints the per-phase summary block.
func mdPhaseEnd(name string, s *markdown.PhaseStats, memBef, memAft float64, diskOut int64, dirName string) {
	fmt.Printf("\n  %s %s done\n", successStyle.Render("✓"), name)
	fmt.Printf("    Files   %s processed", infoStyle.Render(ccFmtInt64(s.Files)))
	if s.Skipped > 0 {
		fmt.Printf("  %s skipped", ccFmtInt64(s.Skipped))
	}
	if s.Errors > 0 {
		fmt.Printf("  %s", warningStyle.Render(ccFmtInt64(s.Errors)+" errors"))
	}
	fmt.Println()

	// Only show throughput metrics when actual work was done (not all-skipped)
	if s.Files > 0 || s.Errors > 0 {
		rate := float64(0)
		readMBs := float64(0)
		writeMBs := float64(0)
		if s.Duration.Seconds() > 0 {
			rate = float64(s.Files+s.Errors) / s.Duration.Seconds()
			readMBs = float64(s.ReadBytes) / (1024 * 1024) / s.Duration.Seconds()
			writeMBs = float64(s.WriteBytes) / (1024 * 1024) / s.Duration.Seconds()
		}
		fmt.Printf("    Rate    %.0f docs/s  ·  %.1f MB/s read  ·  %.1f MB/s write\n",
			rate, readMBs, writeMBs)
	}

	fmt.Printf("    Time    %s\n", s.Duration.Round(time.Millisecond))
	fmt.Printf("    Disk    %s/  →  %s\n", dirName, formatBytes(diskOut))
	fmt.Printf("    RAM     before %.0f MB  →  after %.0f MB  (peak %.0f MB)\n",
		memBef, memAft, s.PeakMemMB)
}

// ─── worker auto-tune ────────────────────────────────────────────────────────

// autoTuneWorkers benchmarks worker counts 8/16/32/64/128/256 on a 200-file
// sample and returns the count with the highest throughput.
func autoTuneWorkers(ctx context.Context, htmlDir string, fast bool) int {
	const sample = 200
	counts := []int{8, 16, 32, 64, 128, 256}

	fmt.Println()
	mdSep()
	fmt.Printf("  Worker auto-tune  (Phase 2 convert, %d-file sample)\n", sample)
	mdSep()
	fmt.Printf("  %-8s  %8s  %9s  %10s\n", "Workers", "Files", "Rate", "MB/s Read")
	fmt.Println(labelStyle.Render("  ──────────────────────────────────────────────"))

	results, err := markdown.BenchmarkConvertWorkers(ctx, htmlDir, counts, fast, sample)
	if err != nil || len(results) == 0 {
		fallback := runtime.NumCPU()
		fmt.Printf("  %s  using NumCPU=%d\n", warningStyle.Render("benchmark failed —"), fallback)
		mdSep()
		fmt.Println()
		return fallback
	}

	bestIdx := 0
	for i, r := range results {
		if r.FilesPerSec > results[bestIdx].FilesPerSec {
			bestIdx = i
		}
	}
	for i, r := range results {
		marker := ""
		if i == bestIdx {
			marker = "  " + successStyle.Render("← best")
		}
		fmt.Printf("  %-8d  %8s  %7.0f/s  %8.1f MB/s%s\n",
			r.Workers, ccFmtInt64(r.Processed), r.FilesPerSec, r.ReadMBPerSec, marker)
	}

	best := results[bestIdx]
	fmt.Println(labelStyle.Render("  ──────────────────────────────────────────────"))
	fmt.Printf("  %s %d workers  (%.0f docs/s)\n",
		successStyle.Render("→ Optimal:"), best.Workers, best.FilesPerSec)
	mdSep()
	fmt.Println()

	return best.Workers
}

// ─── pipeline ────────────────────────────────────────────────────────────────

func runCCMarkdownPhases(ctx context.Context, crawlID, bodyStore string, workers int, force, fast, keepIntermediate bool, cpuProfile string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("  HTML → Markdown   3-Phase Pipeline"))
	fmt.Println()

	// CPU profiling
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			return fmt.Errorf("create cpu profile: %w", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("start cpu profile: %w", err)
		}
		defer pprof.StopCPUProfile()
		fmt.Printf("  Profiling → %s\n\n", labelStyle.Render(cpuProfile))
	}

	// Resolve body store
	ccCfg := cc.DefaultConfig()
	if crawlID != "" {
		ccCfg.CrawlID = crawlID
	}
	if bodyStore == "" {
		bodyStore = filepath.Join(ccCfg.DataDir, "bodies")
	} else if strings.HasPrefix(bodyStore, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home: %w", err)
		}
		bodyStore = filepath.Join(home, bodyStore[2:])
	}
	if info, err := os.Stat(bodyStore); err != nil || !info.IsDir() {
		return fmt.Errorf("body store not found: %s\n  Run 'search cc recrawl' first", bodyStore)
	}

	base := filepath.Dir(bodyStore)
	htmlDir := filepath.Join(base, "html")
	mdDir := filepath.Join(base, "md")
	mdGzDir := filepath.Join(base, "md-gz")
	indexPath := filepath.Join(mdDir, "index.duckdb")

	extractor := "trafilatura (quality)"
	if fast {
		extractor = "go-readability (fast)"
	}
	workersStr := "auto-tune via benchmark"
	if workers > 0 {
		workersStr = fmt.Sprintf("%d (fixed, skipping benchmark)", workers)
	}

	fmt.Printf("  bodies/   %s\n", labelStyle.Render(bodyStore))
	fmt.Printf("  html/     %s\n", labelStyle.Render(htmlDir))
	fmt.Printf("  md/       %s\n", labelStyle.Render(mdDir))
	fmt.Printf("  md-gz/    %s\n", labelStyle.Render(mdGzDir))
	fmt.Printf("  Engine    %s\n", infoStyle.Render(extractor))
	fmt.Printf("  Workers   %s\n", infoStyle.Render(workersStr))
	if force {
		fmt.Printf("  Mode      %s\n", warningStyle.Render("force — re-process all files"))
	} else {
		fmt.Printf("  Mode      %s\n", infoStyle.Render("incremental — skip existing"))
	}
	fmt.Println()

	// Per-phase accumulators for final summary
	type phaseRow struct {
		name    string
		stats   *markdown.PhaseStats
		diskOut int64
	}
	var rows []phaseRow
	pipeStart := time.Now()

	// ═══════════════════════════════════════════════════════════════════
	// Phase 1 — Extract   bodies/*.gz → html/*.html
	// ═══════════════════════════════════════════════════════════════════
	fmt.Println(subtitleStyle.Render("  Phase 1 / 3 — Extract   bodies/*.gz → html/*.html"))
	fmt.Println()
	memBef1 := memSysMB()

	// Phase 1 uses NumCPU (I/O-bound, worker count matters less)
	p1Workers := workers
	if p1Workers <= 0 {
		p1Workers = runtime.NumCPU()
	}
	s1, err := markdown.RunExtract(ctx, markdown.ExtractConfig{
		InputDir:  bodyStore,
		OutputDir: htmlDir,
		Workers:   p1Workers,
		Force:     force,
	}, func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64) {
		mdPhaseProgress("Extracting", done, total, errors, readBytes, writeBytes, elapsed, peakMemMB)
	})
	fmt.Printf("\r\033[K")
	if err != nil {
		return fmt.Errorf("phase 1 extract: %w", err)
	}
	disk1 := diskUsageBytes(htmlDir)
	mdPhaseEnd("Extract", s1, memBef1, memSysMB(), disk1, "html")
	rows = append(rows, phaseRow{"Extract", s1, disk1})
	fmt.Println()

	// ═══════════════════════════════════════════════════════════════════
	// Auto-tune workers (convert is CPU-bound — worker count matters)
	// ═══════════════════════════════════════════════════════════════════
	if workers <= 0 {
		workers = autoTuneWorkers(ctx, htmlDir, fast)
	}

	// ═══════════════════════════════════════════════════════════════════
	// Phase 2 — Convert   html/*.html → md/*.md
	// ═══════════════════════════════════════════════════════════════════
	fmt.Println(subtitleStyle.Render("  Phase 2 / 3 — Convert   html/*.html → md/*.md"))
	fmt.Println()
	memBef2 := memSysMB()

	s2, err := markdown.RunConvertPhase(ctx, markdown.ConvertPhaseConfig{
		InputDir:  htmlDir,
		OutputDir: mdDir,
		IndexPath: indexPath,
		Workers:   workers,
		Force:     force,
		BatchSize: 1000,
		Fast:      fast,
	}, func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64) {
		mdPhaseProgress("Converting", done, total, errors, readBytes, writeBytes, elapsed, peakMemMB)
	})
	fmt.Printf("\r\033[K")
	if err != nil {
		return fmt.Errorf("phase 2 convert: %w", err)
	}
	disk2 := diskUsageBytes(mdDir)
	mdPhaseEnd("Convert", s2, memBef2, memSysMB(), disk2, "md")
	rows = append(rows, phaseRow{"Convert", s2, disk2})
	fmt.Println()

	// ═══════════════════════════════════════════════════════════════════
	// Phase 3 — Compress   md/*.md → md-gz/*.md.gz
	// ═══════════════════════════════════════════════════════════════════
	fmt.Println(subtitleStyle.Render("  Phase 3 / 3 — Compress  md/*.md → md-gz/*.md.gz"))
	fmt.Println()
	memBef3 := memSysMB()

	s3, err := markdown.RunCompress(ctx, markdown.CompressConfig{
		InputDir:  mdDir,
		OutputDir: mdGzDir,
		Workers:   workers,
		Force:     force,
	}, func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64) {
		mdPhaseProgress("Compressing", done, total, errors, readBytes, writeBytes, elapsed, peakMemMB)
	})
	fmt.Printf("\r\033[K")
	if err != nil {
		return fmt.Errorf("phase 3 compress: %w", err)
	}
	disk3 := diskUsageBytes(mdGzDir)
	mdPhaseEnd("Compress", s3, memBef3, memSysMB(), disk3, "md-gz")
	rows = append(rows, phaseRow{"Compress", s3, disk3})
	fmt.Println()

	// ═══════════════════════════════════════════════════════════════════
	// Final summary
	// ═══════════════════════════════════════════════════════════════════
	totalDuration := time.Since(pipeStart)
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
		// Rate: count only actually-processed files (skip all-incremental runs)
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
	if totalDuration.Seconds() > 0 {
		overallRate = float64(s1.Files+s1.Skipped) / totalDuration.Seconds()
	}
	fmt.Printf("  %-10s  %8s  %8s  %8s  %8s  %7.0f/s  %6.0fMB  %s\n",
		"Total",
		ccFmtInt64(s1.Files+s1.Skipped),
		formatBytes(totalRead),
		formatBytes(s3.WriteBytes),
		formatBytes(disk3),
		overallRate,
		peakRAM,
		totalDuration.Round(time.Millisecond),
	)
	mdSep()
	fmt.Println()

	// Disk layout
	fmt.Println(subtitleStyle.Render("  Disk layout:"))
	fmt.Printf("  html/    %s\n", formatBytes(disk1))
	fmt.Printf("  md/      %s\n", formatBytes(disk2))
	fmt.Printf("  md-gz/   %s", formatBytes(disk3))
	if disk1 > 0 && disk3 > 0 {
		fmt.Printf("  (-%.1f%% vs html/)", (1.0-float64(disk3)/float64(disk1))*100)
	}
	fmt.Println()
	fmt.Println()

	// Convert error breakdown (from DuckDB)
	if s2.Errors > 0 {
		cats, err := markdown.QueryErrors(indexPath)
		if err == nil && len(cats) > 0 {
			fmt.Println(subtitleStyle.Render("  Convert Error Breakdown:"))
			fmt.Println()
			var totalDBErrors int64
			for _, c := range cats {
				totalDBErrors += int64(c.Count)
			}
			for _, c := range cats {
				pct := float64(c.Count) / float64(totalDBErrors) * 100
				fmt.Printf("    %-30s %s (%.1f%%)\n",
					c.Category, warningStyle.Render(ccFmtInt64(int64(c.Count))), pct)
			}
			fmt.Println()
		}
	}

	// Auto-cleanup intermediate files (unless --keep-intermediate)
	if !keepIntermediate {
		fmt.Println(subtitleStyle.Render("  Cleanup:"))
		// Remove html/ entirely
		if err := os.RemoveAll(htmlDir); err != nil {
			fmt.Printf("  %s remove html/: %v\n", warningStyle.Render("warn:"), err)
		} else {
			fmt.Printf("  %s  removed html/  (%s freed)\n", successStyle.Render("✓"), formatBytes(disk1))
		}
		// Remove *.md files in md/ but keep index.duckdb
		var mdRemoved int64
		_ = filepath.WalkDir(mdDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			if info, e := d.Info(); e == nil {
				mdRemoved += info.Size()
			}
			_ = os.Remove(path)
			return nil
		})
		if mdRemoved > 0 {
			fmt.Printf("  %s  removed md/*.md (%s freed, index.duckdb kept)\n",
				successStyle.Render("✓"), formatBytes(mdRemoved))
		}
		fmt.Println()
	}

	return nil
}

// ─── streaming pipeline (--mem) ──────────────────────────────────────────────

func runCCMarkdownPipeline(ctx context.Context, crawlID, bodyStore string, workers int, force, fast bool, cpuProfile string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("  HTML → Markdown   Streaming Pipeline  (--mem)"))
	fmt.Println()

	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			return fmt.Errorf("create cpu profile: %w", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("start cpu profile: %w", err)
		}
		defer pprof.StopCPUProfile()
		fmt.Printf("  Profiling → %s\n\n", labelStyle.Render(cpuProfile))
	}

	// Resolve body store
	ccCfg := cc.DefaultConfig()
	if crawlID != "" {
		ccCfg.CrawlID = crawlID
	}
	if bodyStore == "" {
		bodyStore = filepath.Join(ccCfg.DataDir, "bodies")
	} else if strings.HasPrefix(bodyStore, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home: %w", err)
		}
		bodyStore = filepath.Join(home, bodyStore[2:])
	}
	if info, err := os.Stat(bodyStore); err != nil || !info.IsDir() {
		return fmt.Errorf("body store not found: %s\n  Run 'search cc recrawl' first", bodyStore)
	}

	base := filepath.Dir(bodyStore)
	mdGzDir := filepath.Join(base, "md-gz")
	indexPath := filepath.Join(mdGzDir, "index.duckdb")

	extractor := "trafilatura (quality)"
	if fast {
		extractor = "go-readability (fast)"
	}
	effWorkers := workers
	if effWorkers <= 0 {
		effWorkers = runtime.NumCPU()
	}

	fmt.Printf("  bodies/   %s\n", labelStyle.Render(bodyStore))
	fmt.Printf("  md-gz/    %s\n", labelStyle.Render(mdGzDir))
	fmt.Printf("  Engine    %s\n", infoStyle.Render(extractor))
	fmt.Printf("  Workers   %s per stage  (3 stages × %d = %d goroutines)\n",
		infoStyle.Render(fmt.Sprintf("%d", effWorkers)), effWorkers, effWorkers*3)
	if force {
		fmt.Printf("  Mode      %s\n", warningStyle.Render("force — re-process all files"))
	} else {
		fmt.Printf("  Mode      %s\n", infoStyle.Render("incremental — skip existing"))
	}
	fmt.Println()

	fmt.Println(subtitleStyle.Render("  Pipeline  bodies/*.gz → [htmlCh] → convert → [mdCh] → md-gz/*.md.gz"))
	fmt.Println()

	diskIn := diskUsageBytes(bodyStore)
	memBef := memSysMB()

	result, err := markdown.RunPipeline(ctx, markdown.PipelineConfig{
		InputDir:  bodyStore,
		OutputDir: mdGzDir,
		IndexPath: indexPath,
		Workers:   workers,
		Fast:      fast,
		Force:     force,
		BatchSize: 1000,
	}, func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64) {
		mdPhaseProgress("Pipeline", done, total, errors, readBytes, writeBytes, elapsed, peakMemMB)
	})
	fmt.Printf("\r\033[K")
	if err != nil {
		return fmt.Errorf("pipeline: %w", err)
	}

	memAft := memSysMB()
	diskOut := diskUsageBytes(mdGzDir)

	// ── Summary ───────────────────────────────────────────────────────────────
	fmt.Printf("\n  %s pipeline done\n", successStyle.Render("✓"))
	fmt.Println()

	rate := float64(0)
	readMBs := float64(0)
	writeMBs := float64(0)
	if result.Duration.Seconds() > 0 {
		rate = float64(result.Written+result.Errors) / result.Duration.Seconds()
		readMBs = float64(result.ReadBytes) / (1024 * 1024) / result.Duration.Seconds()
		writeMBs = float64(result.WriteBytes) / (1024 * 1024) / result.Duration.Seconds()
	}

	fmt.Printf("    Decompressed  %s files\n", infoStyle.Render(ccFmtInt64(result.Read)))
	fmt.Printf("    Converted     %s to Markdown", infoStyle.Render(ccFmtInt64(result.Converted)))
	if result.Errors > 0 {
		denom := result.Read
		if denom < 1 {
			denom = 1
		}
		errPct := float64(result.Errors) / float64(denom) * 100
		fmt.Printf("  (%s, %.1f%% extraction errors)", warningStyle.Render(ccFmtInt64(result.Errors)+" skipped"), errPct)
	}
	fmt.Println()
	fmt.Printf("    Written       %s → md-gz/  (%s)\n",
		infoStyle.Render(ccFmtInt64(result.Written)), formatBytes(diskOut))
	if result.Skipped > 0 {
		fmt.Printf("    Skipped       %s (already exist)\n", ccFmtInt64(result.Skipped))
	}
	fmt.Printf("    Rate          %.0f docs/s  ·  %.1f MB/s read  ·  %.1f MB/s write\n",
		rate, readMBs, writeMBs)
	fmt.Printf("    Time          %s\n", result.Duration.Round(time.Millisecond))
	fmt.Printf("    RAM           before %.0f MB → after %.0f MB  (peak %.0f MB)\n",
		memBef, memAft, result.PeakMemMB)
	fmt.Println()

	// ── Disk comparison ───────────────────────────────────────────────────────
	fmt.Println(subtitleStyle.Render("  Disk:"))
	fmt.Printf("  bodies/   %s  (input .gz)\n", formatBytes(diskIn))
	fmt.Printf("  md-gz/    %s", formatBytes(diskOut))
	if diskIn > 0 && diskOut > 0 {
		fmt.Printf("  (-%.1f%% vs bodies/)", (1.0-float64(diskOut)/float64(diskIn))*100)
	}
	fmt.Println()
	fmt.Println()

	return nil
}
