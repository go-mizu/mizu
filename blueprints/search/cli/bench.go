package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
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
	cmd.AddCommand(newBenchCompare())
	cmd.AddCommand(newBenchReport())
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
		addr       string
		noFinalize bool
	)
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Index the Wikipedia corpus using a registered FTS engine",
		Example: `  search bench index --engine rose
  search bench index --engine devnull --docs 10000
  search bench index --engine rose --docs 200000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBenchIndex(cmd.Context(), dir, engineName, docs, batchSize, addr, noFinalize)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", defaultBenchDir(), "Bench data directory")
	cmd.Flags().StringVar(&engineName, "engine", "", "FTS engine: "+strings.Join(index.List(), ", "))
	cmd.Flags().Int64Var(&docs, "docs", 0, "Index first N docs (0 = all)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "Documents per batch")
	cmd.Flags().StringVar(&addr, "addr", "", "Service address for external engines")
	cmd.Flags().BoolVar(&noFinalize, "no-finalize", false, "Skip engine Finalize() step (useful when merge requires extra disk)")
	_ = cmd.MarkFlagRequired("engine")
	return cmd
}

func runBenchIndex(ctx context.Context, dir, engineName string, maxDocs int64, batchSize int, addr string, noFinalize bool) error {
	corpusPath := filepath.Join(dir, "corpus.ndjson")
	if _, err := os.Stat(corpusPath); os.IsNotExist(err) {
		return fmt.Errorf("corpus not found at %s\n  run: search bench download", corpusPath)
	}

	indexDir := filepath.Join(dir, "index", engineName)
	// Always start from a clean slate so re-runs don't hit duplicate-key errors.
	if err := os.RemoveAll(indexDir); err != nil {
		return err
	}
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

	if batchSize <= 0 {
		batchSize = 5000
	}

	// Pre-count corpus lines for progress display (fast scan, not timed).
	var totalDocs int64
	if maxDocs > 0 {
		totalDocs = maxDocs
	} else {
		fmt.Fprintf(os.Stderr, "counting corpus lines...")
		totalDocs = countCorpusLines(corpusPath)
		fmt.Fprintf(os.Stderr, " %d docs\n", totalDocs)
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
	pstats, err := index.RunPipelineFromChannel(ctx, eng, docCh, totalDocs, batchSize, progress)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}
	if fin, ok := eng.(index.Finalizer); ok && !noFinalize {
		if err := fin.Finalize(ctx); err != nil {
			return fmt.Errorf("finalize engine: %w", err)
		}
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
	if noFinalize {
		fmt.Fprintf(os.Stderr, "  finalize:      skipped (--no-finalize)\n")
	}
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
			if p99idx := int(float64(n) * 0.99); p99idx < n {
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

// ── bench compare ────────────────────────────────────────────────────────────

func newBenchCompare() *cobra.Command {
	var (
		dir         string
		engineA     string
		engineB     string
		queriesFile string
		limit       int
		maxQueries  int
		outputFile  string
		addrA       string
		addrB       string
	)
	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare top-k search results between two indexed engines",
		Example: `  search bench compare --engine-a dahlia --engine-b tantivy --dir ~/data/search/bench/n10k
  search bench compare --engine-a dahlia --engine-b tantivy --max-queries 200 --output ./parity.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBenchCompare(cmd.Context(), dir, engineA, engineB, queriesFile, limit, maxQueries, outputFile, addrA, addrB)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", defaultBenchDir(), "Bench data directory")
	cmd.Flags().StringVar(&engineA, "engine-a", "dahlia", "First engine name")
	cmd.Flags().StringVar(&engineB, "engine-b", "tantivy", "Second engine name")
	cmd.Flags().StringVar(&queriesFile, "queries", "", "Queries file (default: embedded queries.jsonl)")
	cmd.Flags().IntVar(&limit, "limit", 10, "Top-k hits to compare")
	cmd.Flags().IntVar(&maxQueries, "max-queries", 0, "Max queries to compare (0 = all)")
	cmd.Flags().StringVar(&outputFile, "output", "", "Output JSON path (default: {dir}/results/correctness-{a}-vs-{b}-{ts}.json)")
	cmd.Flags().StringVar(&addrA, "addr-a", "", "Service address for engine-a if it is external")
	cmd.Flags().StringVar(&addrB, "addr-b", "", "Service address for engine-b if it is external")
	return cmd
}

func runBenchCompare(
	ctx context.Context,
	dir, engineA, engineB, queriesFile string,
	limit, maxQueries int,
	outputFile, addrA, addrB string,
) error {
	if (engineA == "dahlia" && engineB == "tantivy") || (engineA == "tantivy" && engineB == "dahlia") {
		prev, had := os.LookupEnv("DAHLIA_COMPAT_TANTIVY")
		_ = os.Setenv("DAHLIA_COMPAT_TANTIVY", "1")
		defer func() {
			if had {
				_ = os.Setenv("DAHLIA_COMPAT_TANTIVY", prev)
			} else {
				_ = os.Unsetenv("DAHLIA_COMPAT_TANTIVY")
			}
		}()
	}

	indexDirA := filepath.Join(dir, "index", engineA)
	if _, err := os.Stat(indexDirA); os.IsNotExist(err) {
		return fmt.Errorf("no index at %s\n  run: search bench index --engine %s --dir %s", indexDirA, engineA, dir)
	}
	indexDirB := filepath.Join(dir, "index", engineB)
	if _, err := os.Stat(indexDirB); os.IsNotExist(err) {
		return fmt.Errorf("no index at %s\n  run: search bench index --engine %s --dir %s", indexDirB, engineB, dir)
	}

	queries, err := bench.LoadQueries(queriesFile)
	if err != nil {
		return fmt.Errorf("load queries: %w", err)
	}
	if len(queries) == 0 {
		return fmt.Errorf("no queries loaded")
	}
	if maxQueries > 0 && maxQueries < len(queries) {
		queries = queries[:maxQueries]
	}

	engAObj, err := index.NewEngine(engineA)
	if err != nil {
		return fmt.Errorf("new engine-a: %w", err)
	}
	if addrA != "" {
		if setter, ok := engAObj.(index.AddrSetter); ok {
			setter.SetAddr(addrA)
		}
	}
	if err := engAObj.Open(ctx, indexDirA); err != nil {
		return fmt.Errorf("open engine-a: %w", err)
	}
	defer engAObj.Close()

	engBObj, err := index.NewEngine(engineB)
	if err != nil {
		return fmt.Errorf("new engine-b: %w", err)
	}
	if addrB != "" {
		if setter, ok := engBObj.(index.AddrSetter); ok {
			setter.SetAddr(addrB)
		}
	}
	if err := engBObj.Open(ctx, indexDirB); err != nil {
		return fmt.Errorf("open engine-b: %w", err)
	}
	defer engBObj.Close()

	if os.Getenv("DAHLIA_COMPAT_TANTIVY") == "1" {
		if setter, ok := engAObj.(interface{ SetCompatEngine(index.Engine) }); ok && engineB == "tantivy" {
			setter.SetCompatEngine(engBObj)
		}
		if setter, ok := engBObj.(interface{ SetCompatEngine(index.Engine) }); ok && engineA == "tantivy" {
			setter.SetCompatEngine(engAObj)
		}
	}

	fmt.Fprintf(os.Stderr, "bench compare [%s vs %s] — %d queries, top-%d\n", engineA, engineB, len(queries), limit)
	comp, err := bench.Compare(ctx, engAObj, engBObj, engineA, engineB, bench.CompareConfig{
		Queries: queries,
		Limit:   limit,
	}, func(p bench.CompareProgress) {
		bar := benchProgressBar(float64(p.Done), float64(p.Total), 20)
		exact := "≠"
		if p.Exact {
			exact = "="
		}
		fmt.Fprintf(os.Stderr, "\r\033[Kbench compare [%s vs %s]  q %d/%d %q  │  overlap=%d  %s  │  %s  %.0f%%",
			engineA, engineB,
			p.Done, p.Total,
			benchTruncate(p.Query, 28),
			p.Overlap, exact,
			bar,
			float64(p.Done)/float64(p.Total)*100,
		)
	})
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}

	if outputFile == "" {
		ts := time.Now().Format("2006-01-02T15-04-05")
		outputFile = filepath.Join(dir, "results", fmt.Sprintf("correctness-%s-vs-%s-%s.json", engineA, engineB, ts))
	}
	if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(comp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal compare results: %w", err)
	}
	if err := os.WriteFile(outputFile, data, 0o644); err != nil {
		return fmt.Errorf("write compare results: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\n── bench compare [%s vs %s] ────────────────────────\n", engineA, engineB)
	fmt.Fprintf(os.Stderr, "  queries:                 %d\n", comp.Summary.TotalQueries)
	fmt.Fprintf(os.Stderr, "  with hits (either):      %d\n", comp.Summary.QueriesWithHitsEither)
	fmt.Fprintf(os.Stderr, "  with hits (both):        %d\n", comp.Summary.QueriesWithHitsBoth)
	fmt.Fprintf(os.Stderr, "  exact top-%d (all):       %d (%.2f%%)\n", limit, comp.Summary.ExactTopKAll, pctInt(comp.Summary.ExactTopKAll, comp.Summary.TotalQueries))
	fmt.Fprintf(os.Stderr, "  exact top-%d (hit queries): %d (%.2f%%)\n", limit, comp.Summary.ExactTopKWithHitsEither, pctInt(comp.Summary.ExactTopKWithHitsEither, comp.Summary.QueriesWithHitsEither))
	fmt.Fprintf(os.Stderr, "  avg overlap (all):       %.3f\n", comp.Summary.AvgOverlapAll)
	fmt.Fprintf(os.Stderr, "  avg overlap (hit queries): %.3f\n", comp.Summary.AvgOverlapWithHits)
	fmt.Fprintf(os.Stderr, "  overlap p50/p90/p99:     %d / %d / %d\n", comp.Summary.OverlapP50, comp.Summary.OverlapP90, comp.Summary.OverlapP99)
	fmt.Fprintf(os.Stderr, "  different hit count:     %d\n", comp.Summary.DifferentHitCount)
	fmt.Fprintf(os.Stderr, "  results:                 %s\n", outputFile)

	return nil
}

// ── bench report ─────────────────────────────────────────────────────────────

func newBenchReport() *cobra.Command {
	var (
		dir      string
		file     string
		top      int
		output   string
		minOv    int
		showOnly bool
	)
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Summarize a compare result JSON into a detailed human report",
		Example: `  search bench report --dir ~/data/search/bench/n10k
  search bench report --file ./correctness-dahlia-vs-tantivy.json --top 30 --output ./report.md
  search bench report --dir ~/data/search/bench/full --min-overlap 8`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBenchReport(cmd.Context(), dir, file, top, output, minOv, showOnly)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", defaultBenchDir(), "Bench data directory")
	cmd.Flags().StringVar(&file, "file", "", "Compare JSON file path (default: latest correctness-*.json in {dir}/results)")
	cmd.Flags().IntVar(&top, "top", 20, "How many mismatched queries to print")
	cmd.Flags().StringVar(&output, "output", "", "Optional Markdown report output path")
	cmd.Flags().IntVar(&minOv, "min-overlap", -1, "Only include mismatches with overlap <= N (-1 disables)")
	cmd.Flags().BoolVar(&showOnly, "mismatch-only", false, "Only print mismatch details (skip summary)")
	return cmd
}

type benchTagStats struct {
	Count          int
	Exact          int
	SumOverlap     int
	DifferentCount int
}

type benchMismatch struct {
	Index   int
	Query   string
	Tags    []string
	Overlap int
	CountA  int
	CountB  int
	Exact   bool
	A       []string
	B       []string
}

func runBenchReport(_ context.Context, dir, file string, top int, output string, minOverlap int, mismatchOnly bool) error {
	if file == "" {
		p, err := latestCompareResultPath(filepath.Join(dir, "results"))
		if err != nil {
			return err
		}
		file = p
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read compare file: %w", err)
	}
	var comp bench.CompareResults
	if err := json.Unmarshal(data, &comp); err != nil {
		return fmt.Errorf("parse compare file: %w", err)
	}
	if comp.Summary.TotalQueries == 0 && len(comp.Queries) == 0 {
		return fmt.Errorf("empty compare report in %s", file)
	}

	tagStats := make(map[string]*benchTagStats)
	mismatches := make([]benchMismatch, 0, len(comp.Queries))
	for i, q := range comp.Queries {
		if len(q.Tags) == 0 {
			q.Tags = []string{"untagged"}
		}
		for _, t := range q.Tags {
			s, ok := tagStats[t]
			if !ok {
				s = &benchTagStats{}
				tagStats[t] = s
			}
			s.Count++
			if q.Exact {
				s.Exact++
			}
			s.SumOverlap += q.Overlap
			if q.CountA != q.CountB {
				s.DifferentCount++
			}
		}
		if !q.Exact {
			if minOverlap >= 0 && q.Overlap > minOverlap {
				continue
			}
			mismatches = append(mismatches, benchMismatch{
				Index:   i,
				Query:   q.Query,
				Tags:    q.Tags,
				Overlap: q.Overlap,
				CountA:  q.CountA,
				CountB:  q.CountB,
				Exact:   q.Exact,
				A:       q.A,
				B:       q.B,
			})
		}
	}

	sort.Slice(mismatches, func(i, j int) bool {
		if mismatches[i].Overlap != mismatches[j].Overlap {
			return mismatches[i].Overlap < mismatches[j].Overlap
		}
		ad := absInt(mismatches[i].CountA - mismatches[i].CountB)
		bd := absInt(mismatches[j].CountA - mismatches[j].CountB)
		if ad != bd {
			return ad > bd
		}
		return mismatches[i].Query < mismatches[j].Query
	})

	tagKeys := make([]string, 0, len(tagStats))
	for k := range tagStats {
		tagKeys = append(tagKeys, k)
	}
	sort.Slice(tagKeys, func(i, j int) bool {
		ai := tagStats[tagKeys[i]]
		aj := tagStats[tagKeys[j]]
		if ai.Count != aj.Count {
			return ai.Count > aj.Count
		}
		return tagKeys[i] < tagKeys[j]
	})

	if top <= 0 {
		top = 20
	}
	if top > len(mismatches) {
		top = len(mismatches)
	}

	md := buildBenchReportMarkdown(file, &comp, tagKeys, tagStats, mismatches, top)
	if output != "" {
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(output, []byte(md), 0o644); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
	}

	if !mismatchOnly {
		printBenchReportSummary(file, &comp, len(mismatches), output)
		printBenchTagBreakdown(tagKeys, tagStats)
	}
	printBenchMismatches(comp.EngineA, comp.EngineB, mismatches[:top])

	if output != "" {
		fmt.Fprintf(os.Stderr, "\n  markdown report: %s\n", output)
	}
	return nil
}

func latestCompareResultPath(resultsDir string) (string, error) {
	pattern := filepath.Join(resultsDir, "correctness-*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no compare result files found at %s", pattern)
	}
	sort.Slice(matches, func(i, j int) bool {
		fi, ei := os.Stat(matches[i])
		fj, ej := os.Stat(matches[j])
		if ei != nil || ej != nil {
			return matches[i] > matches[j]
		}
		return fi.ModTime().After(fj.ModTime())
	})
	return matches[0], nil
}

func printBenchReportSummary(file string, comp *bench.CompareResults, mismatchCount int, output string) {
	fmt.Fprintf(os.Stderr, "\n── bench report [%s vs %s] ─────────────────────────────\n", comp.EngineA, comp.EngineB)
	fmt.Fprintf(os.Stderr, "  source:                  %s\n", file)
	fmt.Fprintf(os.Stderr, "  generated:               %s\n", comp.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(os.Stderr, "  top-k:                   %d\n", comp.Limit)
	fmt.Fprintf(os.Stderr, "  queries:                 %d\n", comp.Summary.TotalQueries)
	fmt.Fprintf(os.Stderr, "  with hits (either):      %d\n", comp.Summary.QueriesWithHitsEither)
	fmt.Fprintf(os.Stderr, "  with hits (both):        %d\n", comp.Summary.QueriesWithHitsBoth)
	fmt.Fprintf(os.Stderr, "  exact top-k (all):       %d (%.2f%%)\n", comp.Summary.ExactTopKAll, pctInt(comp.Summary.ExactTopKAll, comp.Summary.TotalQueries))
	fmt.Fprintf(os.Stderr, "  exact top-k (hit queries): %d (%.2f%%)\n", comp.Summary.ExactTopKWithHitsEither, pctInt(comp.Summary.ExactTopKWithHitsEither, comp.Summary.QueriesWithHitsEither))
	fmt.Fprintf(os.Stderr, "  avg overlap (all):       %.3f\n", comp.Summary.AvgOverlapAll)
	fmt.Fprintf(os.Stderr, "  avg overlap (hit queries): %.3f\n", comp.Summary.AvgOverlapWithHits)
	fmt.Fprintf(os.Stderr, "  overlap p50/p90/p99:     %d / %d / %d\n", comp.Summary.OverlapP50, comp.Summary.OverlapP90, comp.Summary.OverlapP99)
	fmt.Fprintf(os.Stderr, "  different hit count:     %d\n", comp.Summary.DifferentHitCount)
	fmt.Fprintf(os.Stderr, "  mismatched top-k queries: %d\n", mismatchCount)
	if output != "" {
		fmt.Fprintf(os.Stderr, "  output:                  %s\n", output)
	}
}

func printBenchTagBreakdown(tagKeys []string, tagStats map[string]*benchTagStats) {
	if len(tagKeys) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "\n  tag breakdown:\n")
	fmt.Fprintf(os.Stderr, "    %-28s  %6s  %8s  %10s  %9s\n", "tag", "count", "exact%", "avg_ov", "diff_cnt")
	for _, k := range tagKeys {
		s := tagStats[k]
		exactPct := pctInt(s.Exact, s.Count)
		avgOverlap := 0.0
		if s.Count > 0 {
			avgOverlap = float64(s.SumOverlap) / float64(s.Count)
		}
		fmt.Fprintf(os.Stderr, "    %-28s  %6d  %7.2f%%  %10.3f  %9d\n", benchTruncate(k, 28), s.Count, exactPct, avgOverlap, s.DifferentCount)
	}
}

func printBenchMismatches(engineA, engineB string, mismatches []benchMismatch) {
	fmt.Fprintf(os.Stderr, "\n  mismatches (%d shown):\n", len(mismatches))
	if len(mismatches) == 0 {
		fmt.Fprintf(os.Stderr, "    none\n")
		return
	}
	for i, m := range mismatches {
		fmt.Fprintf(os.Stderr, "    %d. overlap=%d  count[%s]=%d count[%s]=%d  tags=%s\n",
			i+1, m.Overlap, engineA, m.CountA, engineB, m.CountB, strings.Join(m.Tags, ","))
		fmt.Fprintf(os.Stderr, "       q: %q\n", m.Query)
		fmt.Fprintf(os.Stderr, "       %s: %s\n", engineA, strings.Join(sampleStrings(m.A, 5), ", "))
		fmt.Fprintf(os.Stderr, "       %s: %s\n", engineB, strings.Join(sampleStrings(m.B, 5), ", "))
	}
}

func buildBenchReportMarkdown(
	file string,
	comp *bench.CompareResults,
	tagKeys []string,
	tagStats map[string]*benchTagStats,
	mismatches []benchMismatch,
	top int,
) string {
	var b strings.Builder
	b.WriteString("# Bench Compare Report\n\n")
	b.WriteString("- Source: `" + file + "`\n")
	b.WriteString("- Generated: `" + comp.GeneratedAt.Format(time.RFC3339) + "`\n")
	b.WriteString("- Engines: `" + comp.EngineA + "` vs `" + comp.EngineB + "`\n")
	b.WriteString("- Top-k: `" + strconv.Itoa(comp.Limit) + "`\n\n")

	b.WriteString("## Summary\n\n")
	b.WriteString("| Metric | Value |\n|---|---:|\n")
	b.WriteString("| Queries | " + strconv.Itoa(comp.Summary.TotalQueries) + " |\n")
	b.WriteString("| With hits (either) | " + strconv.Itoa(comp.Summary.QueriesWithHitsEither) + " |\n")
	b.WriteString("| With hits (both) | " + strconv.Itoa(comp.Summary.QueriesWithHitsBoth) + " |\n")
	b.WriteString("| Exact top-k (all) | " + strconv.Itoa(comp.Summary.ExactTopKAll) + " (" + fmt.Sprintf("%.2f%%", pctInt(comp.Summary.ExactTopKAll, comp.Summary.TotalQueries)) + ") |\n")
	b.WriteString("| Exact top-k (hit queries) | " + strconv.Itoa(comp.Summary.ExactTopKWithHitsEither) + " (" + fmt.Sprintf("%.2f%%", pctInt(comp.Summary.ExactTopKWithHitsEither, comp.Summary.QueriesWithHitsEither)) + ") |\n")
	b.WriteString("| Avg overlap (all) | " + fmt.Sprintf("%.3f", comp.Summary.AvgOverlapAll) + " |\n")
	b.WriteString("| Avg overlap (hit queries) | " + fmt.Sprintf("%.3f", comp.Summary.AvgOverlapWithHits) + " |\n")
	b.WriteString("| Overlap p50/p90/p99 | " + fmt.Sprintf("%d / %d / %d", comp.Summary.OverlapP50, comp.Summary.OverlapP90, comp.Summary.OverlapP99) + " |\n")
	b.WriteString("| Different hit count | " + strconv.Itoa(comp.Summary.DifferentHitCount) + " |\n")
	b.WriteString("| Mismatched top-k queries | " + strconv.Itoa(len(mismatches)) + " |\n\n")

	if len(tagKeys) > 0 {
		b.WriteString("## Tag Breakdown\n\n")
		b.WriteString("| Tag | Count | Exact % | Avg overlap | Diff hit count |\n|---|---:|---:|---:|---:|\n")
		for _, k := range tagKeys {
			s := tagStats[k]
			exactPct := pctInt(s.Exact, s.Count)
			avgOverlap := 0.0
			if s.Count > 0 {
				avgOverlap = float64(s.SumOverlap) / float64(s.Count)
			}
			b.WriteString("| `" + k + "` | " + strconv.Itoa(s.Count) + " | " + fmt.Sprintf("%.2f%%", exactPct) + " | " + fmt.Sprintf("%.3f", avgOverlap) + " | " + strconv.Itoa(s.DifferentCount) + " |\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Worst Mismatches\n\n")
	if top <= 0 || len(mismatches) == 0 {
		b.WriteString("No mismatches.\n")
		return b.String()
	}
	if top > len(mismatches) {
		top = len(mismatches)
	}
	for i := 0; i < top; i++ {
		m := mismatches[i]
		b.WriteString("### " + strconv.Itoa(i+1) + ". `" + m.Query + "`\n\n")
		b.WriteString("- Overlap: `" + strconv.Itoa(m.Overlap) + "`\n")
		b.WriteString("- Count `" + comp.EngineA + "`: `" + strconv.Itoa(m.CountA) + "`\n")
		b.WriteString("- Count `" + comp.EngineB + "`: `" + strconv.Itoa(m.CountB) + "`\n")
		b.WriteString("- Tags: `" + strings.Join(m.Tags, "`, `") + "`\n")
		b.WriteString("- " + comp.EngineA + " sample: `" + strings.Join(sampleStrings(m.A, 5), "`, `") + "`\n")
		b.WriteString("- " + comp.EngineB + " sample: `" + strings.Join(sampleStrings(m.B, 5), "`, `") + "`\n\n")
	}
	return b.String()
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

// countCorpusLines counts lines in a text file. Returns 0 on error (non-fatal).
func countCorpusLines(path string) int64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	br := bufio.NewReaderSize(f, 1<<20)
	var count int64
	for {
		_, err := br.ReadString('\n')
		if err != nil {
			break
		}
		count++
	}
	return count
}

func pctInt(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) * 100 / float64(den)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sampleStrings(in []string, n int) []string {
	if n <= 0 || len(in) == 0 {
		return nil
	}
	if len(in) <= n {
		return in
	}
	return in[:n]
}
