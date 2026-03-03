package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/bench"
	"github.com/spf13/cobra"
)

// NewBench returns the root "bench" command.
func NewBench() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bench",
		Short: "Benchmark FTS engines on the Wikipedia corpus",
		Long:  `Download the Wikipedia corpus and benchmark indexing and search performance against it.`,
		Example: `  search bench download --docs 100000
  search bench index --engine rose --docs 100000
  search bench search --engine rose --commands TOP_10 --iter 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newBenchDownload())
	cmd.AddCommand(newBenchIndex())
	cmd.AddCommand(newBenchSearch())
	return cmd
}

func defaultBenchDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "search", "bench")
}

// ── bench download ────────────────────────────────────────────────────────────

func newBenchDownload() *cobra.Command {
	var (
		url   string
		dir   string
		docs  int64
		force bool
	)
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download and preprocess Wikipedia corpus to corpus.ndjson",
		Example: `  search bench download
  search bench download --docs 100000
  search bench download --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBenchDownload(cmd.Context(), url, dir, docs, force)
		},
	}
	cmd.Flags().StringVar(&url, "url", bench.DefaultCorpusURL, "Source URL (bzip2)")
	cmd.Flags().StringVar(&dir, "dir", defaultBenchDir(), "Bench data directory")
	cmd.Flags().Int64Var(&docs, "docs", 0, "Stop after N docs (0 = all ~6M)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing corpus.ndjson")
	return cmd
}

func runBenchDownload(ctx context.Context, url, dir string, maxDocs int64, force bool) error {
	outPath := filepath.Join(dir, "corpus.ndjson")

	cfg := bench.DownloadConfig{
		URL:     url,
		OutPath: outPath,
		MaxDocs: maxDocs,
		Force:   force,
	}

	fmt.Fprintf(os.Stderr, "downloading Wikipedia corpus → %s\n", outPath)

	progress := func(s *bench.DownloadStats) {
		elapsed := time.Since(s.StartTime).Seconds()
		if elapsed <= 0 {
			elapsed = 0.001
		}
		dlBytes := s.BytesDownloaded.Load()
		dlMB := float64(dlBytes) / 1e6
		totalMB := float64(s.TotalBytes) / 1e6
		docs := s.DocsWritten.Load()
		writtenBytes := s.BytesWritten.Load()
		dlSpeed := dlMB / elapsed
		docRate := float64(docs) / elapsed

		bar := benchProgressBar(dlMB, totalMB, 20)

		eta := ""
		if s.TotalBytes > 0 && dlSpeed > 0 {
			remainSec := (totalMB - dlMB) / dlSpeed
			if remainSec > 0 {
				eta = fmt.Sprintf("  eta %s", benchFmtDuration(time.Duration(remainSec)*time.Second))
			}
		}
		fmt.Fprintf(os.Stderr, "\r\033[Kdownloading  %s  %.2f/%.2f GB  │  %.1f MB/s  │  %d docs  │  %.0f docs/s  │  out %s%s",
			bar,
			dlMB/1000, totalMB/1000,
			dlSpeed,
			docs, docRate,
			formatBytes(writtenBytes),
			eta,
		)
	}

	stats, err := bench.Download(ctx, cfg, progress)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}

	elapsed := time.Since(stats.StartTime)
	fi, _ := os.Stat(outPath)
	var corpusSize int64
	if fi != nil {
		corpusSize = fi.Size()
	}

	dlSpeed := float64(stats.BytesDownloaded.Load()) / 1e6 / elapsed.Seconds()
	docRate := float64(stats.DocsWritten.Load()) / elapsed.Seconds()

	fmt.Fprintf(os.Stderr, "\n── bench download complete ──────────────────────────────\n")
	fmt.Fprintf(os.Stderr, "  docs:          %d\n", stats.DocsWritten.Load())
	fmt.Fprintf(os.Stderr, "  corpus size:   %s\n", formatBytes(corpusSize))
	fmt.Fprintf(os.Stderr, "  elapsed:       %s\n", elapsed.Round(time.Second))
	fmt.Fprintf(os.Stderr, "  avg dl speed:  %.1f MB/s\n", dlSpeed)
	fmt.Fprintf(os.Stderr, "  avg doc rate:  %.0f docs/s\n", docRate)
	fmt.Fprintf(os.Stderr, "  path:          %s\n", outPath)
	return nil
}

// ── bench index ──────────────────────────────────────────────────────────────

func newBenchIndex() *cobra.Command {
	var (
		dir        string
		engineName string
		docs       int64
		batchSize  int
		workers    int
		addr       string
	)
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Index the Wikipedia corpus using a registered FTS engine",
		Example: `  search bench index --engine rose
  search bench index --engine devnull --docs 10000
  search bench index --engine rose --docs 200000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBenchIndex(cmd.Context(), dir, engineName, docs, batchSize, workers, addr)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", defaultBenchDir(), "Bench data directory")
	cmd.Flags().StringVar(&engineName, "engine", "", "FTS engine: "+strings.Join(index.List(), ", "))
	cmd.Flags().Int64Var(&docs, "docs", 0, "Index first N docs (0 = all)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "Documents per batch")
	cmd.Flags().IntVar(&workers, "workers", 0, "Indexing workers (0 = NumCPU)")
	cmd.Flags().StringVar(&addr, "addr", "", "Service address for external engines")
	_ = cmd.MarkFlagRequired("engine")
	return cmd
}

func runBenchIndex(ctx context.Context, dir, engineName string, maxDocs int64, batchSize, workers int, addr string) error {
	corpusPath := filepath.Join(dir, "corpus.ndjson")
	if _, err := os.Stat(corpusPath); os.IsNotExist(err) {
		return fmt.Errorf("corpus not found at %s\n  run: search bench download", corpusPath)
	}

	indexDir := filepath.Join(dir, "index", engineName)
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return err
	}

	eng, err := index.NewEngine(engineName)
	if err != nil {
		return err
	}
	if addr != "" {
		if setter, ok := eng.(index.AddrSetter); ok {
			setter.SetAddr(addr)
		} else {
			fmt.Fprintf(os.Stderr, "warning: engine %q does not support --addr\n", engineName)
		}
	}
	if err := eng.Open(ctx, indexDir); err != nil {
		return fmt.Errorf("open engine: %w", err)
	}
	defer eng.Close()

	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if batchSize <= 0 {
		batchSize = 5000
	}

	docCh := make(chan index.Document, batchSize*2)
	if err := bench.CorpusReader(ctx, corpusPath, maxDocs, docCh); err != nil {
		return err
	}

	progress := func(done, total int64, elapsed time.Duration) {
		secs := elapsed.Seconds()
		if secs <= 0 {
			secs = 0.001
		}
		rate := float64(done) / secs
		disk := index.DirSizeBytes(indexDir)
		rss := currentRSSMB()
		bar := benchProgressBar(float64(done), float64(total), 20)
		if total > 0 {
			fmt.Fprintf(os.Stderr, "\r\033[Kbench index [%s]  %s  %d/%d docs  │  %.0f docs/s  │  %.1fs  │  RSS %d MB  │  disk %s",
				engineName, bar, done, total, rate, secs, rss, formatBytes(disk))
		} else {
			fmt.Fprintf(os.Stderr, "\r\033[Kbench index [%s]  %d docs  │  %.0f docs/s  │  %.1fs  │  RSS %d MB  │  disk %s",
				engineName, done, rate, secs, rss, formatBytes(disk))
		}
	}

	t0 := time.Now()
	pstats, err := index.RunPipelineFromChannel(ctx, eng, docCh, 0, batchSize, progress)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}

	engStats, _ := eng.Stats(ctx)
	elapsed := time.Since(t0)
	avgRate := float64(pstats.DocsIndexed.Load()) / elapsed.Seconds()

	fmt.Fprintf(os.Stderr, "\n── bench index complete ─────────────────────────────────\n")
	fmt.Fprintf(os.Stderr, "  engine:        %s\n", engineName)
	fmt.Fprintf(os.Stderr, "  docs:          %d\n", engStats.DocCount)
	fmt.Fprintf(os.Stderr, "  elapsed:       %s\n", elapsed.Round(100*time.Millisecond))
	fmt.Fprintf(os.Stderr, "  avg rate:      %.0f docs/s\n", avgRate)
	fmt.Fprintf(os.Stderr, "  peak RSS:      %d MB\n", pstats.PeakRSSMB.Load())
	fmt.Fprintf(os.Stderr, "  disk:          %s\n", formatBytes(engStats.DiskBytes))
	fmt.Fprintf(os.Stderr, "  path:          %s\n", indexDir)
	return nil
}

// ── bench search ─────────────────────────────────────────────────────────────

func newBenchSearch() *cobra.Command {
	var (
		dir         string
		engineName  string
		queriesFile string
		commands    string
		iter        int
		warmup      time.Duration
		outputFile  string
		addr        string
	)
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Run benchmark queries against an indexed FTS engine",
		Example: `  search bench search --engine rose
  search bench search --engine rose --commands TOP_10,COUNT --iter 5 --warmup 10s
  search bench search --engine devnull --warmup 0s --iter 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmds := splitCommands(commands)
			return runBenchSearch(cmd.Context(), dir, engineName, queriesFile, cmds, iter, warmup, outputFile, addr)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", defaultBenchDir(), "Bench data directory")
	cmd.Flags().StringVar(&engineName, "engine", "", "FTS engine")
	cmd.Flags().StringVar(&queriesFile, "queries", "", "Queries file (default: embedded queries.jsonl)")
	cmd.Flags().StringVar(&commands, "commands", "TOP_10", "Commands: TOP_10,COUNT,TOP_10_COUNT")
	cmd.Flags().IntVar(&iter, "iter", 10, "Iterations per query")
	cmd.Flags().DurationVar(&warmup, "warmup", 30*time.Second, "Warmup duration before timing")
	cmd.Flags().StringVar(&outputFile, "output", "", "Output JSON path (default: {dir}/results/{ts}.json)")
	cmd.Flags().StringVar(&addr, "addr", "", "Service address for external engines")
	_ = cmd.MarkFlagRequired("engine")
	return cmd
}

func splitCommands(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToUpper(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func runBenchSearch(ctx context.Context, dir, engineName, queriesFile string, commands []string, iter int, warmup time.Duration, outputFile, addr string) error {
	indexDir := filepath.Join(dir, "index", engineName)
	if _, err := os.Stat(indexDir); os.IsNotExist(err) {
		return fmt.Errorf("no index at %s\n  run: search bench index --engine %s", indexDir, engineName)
	}

	queries, err := bench.LoadQueries(queriesFile)
	if err != nil {
		return fmt.Errorf("load queries: %w", err)
	}
	if len(queries) == 0 {
		return fmt.Errorf("no queries loaded")
	}

	eng, err := index.NewEngine(engineName)
	if err != nil {
		return err
	}
	if addr != "" {
		if setter, ok := eng.(index.AddrSetter); ok {
			setter.SetAddr(addr)
		}
	}
	if err := eng.Open(ctx, indexDir); err != nil {
		return fmt.Errorf("open engine: %w", err)
	}
	defer eng.Close()

	results := bench.NewBenchResults()

	// Record index details.
	engStats, _ := eng.Stats(ctx)
	diskMB := index.DirSizeBytes(indexDir) >> 20
	results.SetDetails(engineName, bench.EngineDetails{
		Docs:   engStats.DocCount,
		DiskMB: diskMB,
	})

	for _, command := range commands {
		cfg := bench.BenchConfig{
			Command: command,
			Queries: queries,
			Iter:    iter,
			Warmup:  warmup,
		}

		fmt.Fprintf(os.Stderr, "\nbench search [%s / %s] — %d queries, %d iter, warmup %s\n",
			engineName, command, len(queries), iter, warmup)
		if warmup > 0 {
			fmt.Fprintf(os.Stderr, "  warming up...\n")
		}

		var allQueryStats []benchQueryStat

		progress := func(idx, total int, q string, s bench.IterStats) {
			allQueryStats = append(allQueryStats, benchQueryStat{q, s.P50})
			pct := float64(idx) / float64(total)
			bar := benchProgressBar(float64(idx), float64(total), 20)
			fmt.Fprintf(os.Stderr, "\r\033[Kbench search [%s / %s]  q %d/%d %q  │  p50=%s  p95=%s  min=%s  max=%s  │  %s  %.0f%%",
				engineName, command,
				idx, total,
				benchTruncate(q, 28),
				benchFmtDuration(s.P50), benchFmtDuration(s.P95),
				benchFmtDuration(s.Min), benchFmtDuration(s.Max),
				bar, pct*100,
			)
		}

		t0 := time.Now()
		qrs, err := bench.Run(ctx, eng, cfg, progress)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("bench %s: %w", command, err)
		}
		elapsed := time.Since(t0)
		results.AddQueryResults(command, engineName, qrs)

		// Aggregate stats across all queries.
		var allP50, allP95, allP99 []int
		for _, qr := range qrs {
			if len(qr.Duration) == 0 {
				continue
			}
			n := len(qr.Duration)
			allP50 = append(allP50, qr.Duration[n/2])
			allP95 = append(allP95, qr.Duration[int(float64(n)*0.95)])
			if p99idx := int(float64(n)*0.99); p99idx < n {
				allP99 = append(allP99, qr.Duration[p99idx])
			}
		}

		slowest, fastest := findSlowFast(allQueryStats)

		fmt.Fprintf(os.Stderr, "\n── bench search [%s / %s] ─────────────────────────\n", engineName, command)
		fmt.Fprintf(os.Stderr, "  queries:       %d\n", len(qrs))
		fmt.Fprintf(os.Stderr, "  iterations:    %d  (after %s warmup)\n", iter, warmup)
		fmt.Fprintf(os.Stderr, "  elapsed:       %s\n", elapsed.Round(100*time.Millisecond))
		if len(allP50) > 0 {
			fmt.Fprintf(os.Stderr, "  median p50:    %s\n", benchFmtDuration(time.Duration(medianInt(allP50))*time.Microsecond))
			fmt.Fprintf(os.Stderr, "  median p95:    %s\n", benchFmtDuration(time.Duration(medianInt(allP95))*time.Microsecond))
			if len(allP99) > 0 {
				fmt.Fprintf(os.Stderr, "  median p99:    %s\n", benchFmtDuration(time.Duration(medianInt(allP99))*time.Microsecond))
			}
		}
		if slowest.query != "" {
			fmt.Fprintf(os.Stderr, "  slowest:       %q → %s\n", slowest.query, benchFmtDuration(slowest.p50))
		}
		if fastest.query != "" {
			fmt.Fprintf(os.Stderr, "  fastest:       %q → %s\n", fastest.query, benchFmtDuration(fastest.p50))
		}
	}

	// Write results JSON.
	if outputFile == "" {
		outputFile = bench.ResultsPath(dir)
	}
	if err := bench.SaveResults(outputFile, results); err != nil {
		return fmt.Errorf("save results: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  results:       %s\n", outputFile)
	return nil
}

// ── helpers local to bench CLI ────────────────────────────────────────────────

// benchProgressBar returns a Unicode block progress bar of width w.
func benchProgressBar(value, total float64, w int) string {
	if total <= 0 {
		return strings.Repeat("░", w)
	}
	filled := int(value / total * float64(w))
	if filled < 0 {
		filled = 0
	}
	if filled > w {
		filled = w
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", w-filled)
}

// benchFmtDuration formats a duration as "1.2ms", "34µs", "5.1s".
func benchFmtDuration(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
	default:
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
}

// benchTruncate truncates s to n runes, adding "…" if truncated.
func benchTruncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

// benchQueryStat holds per-query p50 latency for slowest/fastest reporting.
type benchQueryStat struct {
	query string
	p50   time.Duration
}

func findSlowFast(stats []benchQueryStat) (slowest, fastest benchQueryStat) {
	if len(stats) == 0 {
		return
	}
	slowest = stats[0]
	fastest = stats[0]
	for _, s := range stats[1:] {
		if s.p50 > slowest.p50 {
			slowest = s
		}
		if s.p50 < fastest.p50 {
			fastest = s
		}
	}
	return
}

func medianInt(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	cp := make([]int, len(vals))
	copy(cp, vals)
	sort.Ints(cp)
	return cp[len(cp)/2]
}

// currentRSSMB returns the current process RSS in MB (best-effort).
func currentRSSMB() int64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return int64(ms.Sys >> 20)
}
