package cli

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	// Import all drivers for registration.
	// duckdb registration happens via cli/duckdb_ops.go (excluded when -tags chdb).
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/chdb"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/devnull"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/sqlite"
	// TODO: uncomment after driver packages are created:
	// _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/bleve"
	// _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/meilisearch"
	// _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/clickhouse"
	// _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/postgres"
	// _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/quickwit"
	// _ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/tantivy-lnx"
	"github.com/spf13/cobra"
)

func newCCFTS() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fts",
		Short: "Full-text search index and query for CC markdown",
		Long:  `Build FTS indexes from Common Crawl markdown files and search them.`,
		Example: `  search cc fts index --engine duckdb
  search cc fts index --engine sqlite --workers 8 --batch-size 10000
  search cc fts search "machine learning" --engine duckdb --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newCCFTSIndex())
	cmd.AddCommand(newCCFTSSearch())
	cmd.AddCommand(newCCFTSDecompress())
	cmd.AddCommand(newCCFTSPack())
	return cmd
}

func newCCFTSIndex() *cobra.Command {
	var (
		crawlID   string
		engine    string
		source    string
		batchSize int
		workers   int
		addr      string
	)

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Build FTS index from CC markdown files or a pre-packed bundle",
		Example: `  search cc fts index --engine duckdb
  search cc fts index --engine sqlite --crawl CC-MAIN-2026-08
  search cc fts index --engine devnull  # benchmark I/O only
  search cc fts index --engine devnull --source parquet  # benchmark from parquet pack
  search cc fts index --engine devnull --source bin      # benchmark from flatbin pack`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCFTSIndex(cmd.Context(), crawlID, engine, source, batchSize, workers, addr)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&engine, "engine", "duckdb", "FTS engine: "+strings.Join(index.List(), ", "))
	cmd.Flags().StringVar(&source, "source", "files", "Source: files, parquet, bin, ndjson, duckdb")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "Documents per batch insert")
	cmd.Flags().IntVar(&workers, "workers", 0, "Parallel file readers (0 = NumCPU)")
	cmd.Flags().StringVar(&addr, "addr", "", "Service address for external engines (e.g. http://localhost:7700)")
	return cmd
}

func newCCFTSSearch() *cobra.Command {
	var (
		crawlID string
		engine  string
		limit   int
		offset  int
		addr    string
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search the FTS index",
		Args:  cobra.MinimumNArgs(1),
		Example: `  search cc fts search "machine learning" --engine duckdb
  search cc fts search "climate change" --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			return runCCFTSSearch(cmd.Context(), crawlID, engine, query, limit, offset, addr)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&engine, "engine", "duckdb", "FTS engine")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "Result offset")
	cmd.Flags().StringVar(&addr, "addr", "", "Service address for external engines")
	return cmd
}

func runCCFTSIndex(ctx context.Context, crawlID, engineName, source string, batchSize, workers int, addr string) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}

	homeDir, _ := os.UserHomeDir()
	outputDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "fts", engineName)

	eng, err := index.NewEngine(engineName)
	if err != nil {
		return err
	}
	if addr != "" {
		if setter, ok := eng.(index.AddrSetter); ok {
			setter.SetAddr(addr)
		} else {
			fmt.Fprintf(os.Stderr, "warning: engine %q does not support --addr flag\n", engineName)
		}
	}
	if err := eng.Open(ctx, outputDir); err != nil {
		return fmt.Errorf("open engine: %w", err)
	}
	defer eng.Close()

	if source != "files" {
		return runCCFTSIndexFromPack(ctx, crawlID, engineName, source, eng, outputDir, batchSize)
	}

	sourceDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "markdown")
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("markdown dir not found: %s", sourceDir)
	}

	fmt.Fprintf(os.Stderr, "indexing %s → %s (engine=%s, batch=%d, workers=%d)\n",
		sourceDir, outputDir, engineName, batchSize, workers)

	cfg := index.PipelineConfig{
		SourceDir: sourceDir,
		BatchSize: batchSize,
		Workers:   workers,
	}

	progress := func(stats *index.PipelineStats) {
		total := stats.TotalFiles.Load()
		done := stats.DocsIndexed.Load()
		elapsed := time.Since(stats.StartTime).Seconds()
		rate := float64(0)
		if elapsed > 0 {
			rate = float64(done) / elapsed
		}
		disk := index.DirSizeBytes(outputDir)
		peakMB := stats.PeakRSSMB.Load()

		pct := float64(0)
		if total > 0 {
			pct = float64(done) / float64(total)
		}
		barWidth := 20
		filled := int(pct * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		fmt.Fprintf(os.Stderr, "\r\033[Kindexing [%s] %s %d/%d docs │ %.0f docs/s │ %.1fs │ RSS %d MB │ disk %s",
			engineName, bar, done, total, rate, elapsed, peakMB, formatBytes(disk))
	}

	stats, err := index.RunPipeline(ctx, eng, cfg, progress)
	fmt.Fprintln(os.Stderr) // newline after progress

	if err != nil {
		return err
	}

	if err := ftsCreateDuckDBIndex(ctx, engineName, eng); err != nil {
		return err
	}

	engineStats, _ := eng.Stats(ctx)
	elapsed := time.Since(stats.StartTime)
	avgRate := float64(stats.DocsIndexed.Load()) / elapsed.Seconds()

	fmt.Fprintf(os.Stderr, "\n── FTS Index Complete ──────────────────────────\n")
	fmt.Fprintf(os.Stderr, "  engine:    %s\n", engineName)
	fmt.Fprintf(os.Stderr, "  source:    files\n")
	fmt.Fprintf(os.Stderr, "  docs:      %d\n", engineStats.DocCount)
	fmt.Fprintf(os.Stderr, "  errors:    %d\n", stats.Errors.Load())
	fmt.Fprintf(os.Stderr, "  elapsed:   %s\n", elapsed.Round(100*time.Millisecond))
	fmt.Fprintf(os.Stderr, "  avg rate:  %.0f docs/s\n", avgRate)
	fmt.Fprintf(os.Stderr, "  peak RSS:  %d MB\n", stats.PeakRSSMB.Load())
	fmt.Fprintf(os.Stderr, "  disk:      %s\n", formatBytes(engineStats.DiskBytes))
	fmt.Fprintf(os.Stderr, "  path:      %s\n", outputDir)

	return nil
}

func runCCFTSIndexFromPack(ctx context.Context, crawlID, engineName, source string, eng index.Engine, outputDir string, batchSize int) error {
	homeDir, _ := os.UserHomeDir()
	packDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "fts", "pack")

	packFile, err := packFilePath(packDir, source)
	if err != nil {
		return err
	}
	if _, err := os.Stat(packFile); os.IsNotExist(err) {
		return fmt.Errorf("pack file not found: %s\n  run: search cc fts pack --format %s", packFile, source)
	}

	fmt.Fprintf(os.Stderr, "indexing [%s ← %s] from %s\n", engineName, source, packFile)

	progress := func(done, total int64, elapsed time.Duration) {
		secs := elapsed.Seconds()
		rate := float64(0)
		if secs > 0 {
			rate = float64(done) / secs
		}
		disk := index.DirSizeBytes(outputDir)

		pct := float64(0)
		if total > 0 {
			pct = float64(done) / float64(total)
		}
		barWidth := 20
		filled := int(pct * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		if total > 0 {
			fmt.Fprintf(os.Stderr, "\r\033[Kindexing [%s←%s] %s %d/%d docs │ %.0f docs/s │ %.1fs │ disk %s",
				engineName, source, bar, done, total, rate, secs, formatBytes(disk))
		} else {
			fmt.Fprintf(os.Stderr, "\r\033[Kindexing [%s←%s] %d docs │ %.0f docs/s │ %.1fs │ disk %s",
				engineName, source, done, rate, secs, formatBytes(disk))
		}
	}

	var stats *index.PipelineStats
	switch source {
	case "parquet":
		stats, err = index.RunPipelineFromParquet(ctx, eng, packFile, batchSize, progress)
	case "bin":
		stats, err = index.RunPipelineFromFlatBin(ctx, eng, packFile, batchSize, progress)
	case "ndjson":
		stats, err = index.RunPipelineFromNDJSON(ctx, eng, packFile, batchSize, progress)
	case "duckdb":
		stats, err = runPipelineFromDuckDBRaw(ctx, eng, packFile, batchSize, progress)
	default:
		return fmt.Errorf("unknown source %q (valid: files, parquet, bin, ndjson, duckdb)", source)
	}
	fmt.Fprintln(os.Stderr) // newline after progress

	if err != nil {
		return err
	}

	if err := ftsCreateDuckDBIndex(ctx, engineName, eng); err != nil {
		return err
	}

	engineStats, _ := eng.Stats(ctx)
	elapsed := time.Since(stats.StartTime)
	avgRate := float64(stats.DocsIndexed.Load()) / elapsed.Seconds()

	fmt.Fprintf(os.Stderr, "\n── FTS Index Complete ──────────────────────────\n")
	fmt.Fprintf(os.Stderr, "  engine:    %s\n", engineName)
	fmt.Fprintf(os.Stderr, "  source:    %s\n", source)
	fmt.Fprintf(os.Stderr, "  docs:      %d\n", engineStats.DocCount)
	fmt.Fprintf(os.Stderr, "  elapsed:   %s\n", elapsed.Round(100*time.Millisecond))
	fmt.Fprintf(os.Stderr, "  avg rate:  %.0f docs/s\n", avgRate)
	fmt.Fprintf(os.Stderr, "  peak RSS:  %d MB\n", stats.PeakRSSMB.Load())
	fmt.Fprintf(os.Stderr, "  disk:      %s\n", formatBytes(engineStats.DiskBytes))
	fmt.Fprintf(os.Stderr, "  path:      %s\n", outputDir)

	return nil
}

func ftsCreateDuckDBIndex(ctx context.Context, engineName string, eng index.Engine) error {
	if engineName != "duckdb" {
		return nil
	}
	fmt.Fprintf(os.Stderr, "creating FTS index (BM25)...\n")
	type ftsIndexer interface{ CreateFTSIndex(context.Context) error }
	if ddb, ok := eng.(ftsIndexer); ok {
		if err := ddb.CreateFTSIndex(ctx); err != nil {
			return fmt.Errorf("create FTS index: %w", err)
		}
	}
	return nil
}

func packFilePath(packDir, source string) (string, error) {
	switch source {
	case "parquet":
		return filepath.Join(packDir, "docs.parquet"), nil
	case "bin":
		return filepath.Join(packDir, "docs.bin"), nil
	case "ndjson":
		return filepath.Join(packDir, "docs.ndjson"), nil
	case "duckdb":
		return filepath.Join(packDir, "docs.raw.duckdb"), nil
	default:
		return "", fmt.Errorf("unknown source %q (valid: parquet, bin, ndjson, duckdb)", source)
	}
}

func newCCFTSPack() *cobra.Command {
	var (
		crawlID   string
		format    string
		batchSize int
		workers   int
	)

	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Pack CC markdown files into a high-perf bundle for FTS import benchmarking",
		Long: `Pre-compute markdown files into one or more fast-load formats.
Packed files are stored in fts/pack/ and can be used with 'fts index --source <format>'.`,
		Example: `  search cc fts pack --format parquet   # Parquet columnar
  search cc fts pack --format bin        # flat binary (fastest read)
  search cc fts pack --format ndjson     # newline-delimited JSON
  search cc fts pack --format duckdb    # DuckDB raw table
  search cc fts pack --format all        # all four formats`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCFTSPack(cmd.Context(), crawlID, format, batchSize, workers)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&format, "format", "all", "Format: parquet, bin, ndjson, duckdb, all")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "Documents per batch")
	cmd.Flags().IntVar(&workers, "workers", 0, "Parallel file readers (0 = NumCPU)")
	return cmd
}

func runCCFTSPack(ctx context.Context, crawlID, format string, batchSize, workers int) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}

	homeDir, _ := os.UserHomeDir()
	markdownDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "markdown")
	packDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "fts", "pack")

	if _, err := os.Stat(markdownDir); os.IsNotExist(err) {
		return fmt.Errorf("markdown dir not found: %s", markdownDir)
	}

	formats := []string{format}
	if format == "all" {
		formats = []string{"parquet", "bin", "ndjson", "duckdb"}
	}

	for _, fmt_ := range formats {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		packFile, err := packFilePath(packDir, fmt_)
		if err != nil {
			return err
		}
		if err := runPackFormat(ctx, fmt_, markdownDir, packFile, batchSize, workers); err != nil {
			return fmt.Errorf("pack %s: %w", fmt_, err)
		}
	}
	return nil
}

func runPackFormat(ctx context.Context, format, markdownDir, packFile string, batchSize, workers int) error {
	fmt.Fprintf(os.Stderr, "packing [%s] → %s\n", format, packFile)

	progress := func(stats *index.PipelineStats) {
		total := stats.TotalFiles.Load()
		done := stats.DocsIndexed.Load()
		elapsed := time.Since(stats.StartTime).Seconds()
		rate := float64(0)
		if elapsed > 0 {
			rate = float64(done) / elapsed
		}
		peakMB := stats.PeakRSSMB.Load()

		pct := float64(0)
		if total > 0 {
			pct = float64(done) / float64(total)
		}
		barWidth := 20
		filled := int(pct * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		fmt.Fprintf(os.Stderr, "\r\033[Kpacking [%s] %s %d/%d docs │ %.0f docs/s │ %.1fs │ RSS %d MB",
			format, bar, done, total, rate, elapsed, peakMB)
	}

	var (
		stats *index.PipelineStats
		err   error
	)
	switch format {
	case "parquet":
		stats, err = index.PackParquet(ctx, markdownDir, packFile, workers, batchSize, progress)
	case "bin":
		stats, err = index.PackFlatBin(ctx, markdownDir, packFile, workers, batchSize, progress)
	case "ndjson":
		stats, err = index.PackNDJSON(ctx, markdownDir, packFile, workers, batchSize, progress)
	case "duckdb":
		stats, err = packDuckDBRaw(ctx, markdownDir, packFile, workers, batchSize, progress)
	default:
		return fmt.Errorf("unknown format %q", format)
	}
	fmt.Fprintln(os.Stderr) // newline after progress

	if err != nil {
		return err
	}

	fi, _ := os.Stat(packFile)
	fileSize := int64(0)
	if fi != nil {
		fileSize = fi.Size()
	}
	elapsed := time.Since(stats.StartTime)
	avgRate := float64(stats.DocsIndexed.Load()) / elapsed.Seconds()

	fmt.Fprintf(os.Stderr, "\n── Pack Complete [%s] ───────────────────────────\n", format)
	fmt.Fprintf(os.Stderr, "  docs:      %d\n", stats.DocsIndexed.Load())
	fmt.Fprintf(os.Stderr, "  errors:    %d\n", stats.Errors.Load())
	fmt.Fprintf(os.Stderr, "  elapsed:   %s\n", elapsed.Round(100*time.Millisecond))
	fmt.Fprintf(os.Stderr, "  avg rate:  %.0f docs/s\n", avgRate)
	fmt.Fprintf(os.Stderr, "  peak RSS:  %d MB\n", stats.PeakRSSMB.Load())
	fmt.Fprintf(os.Stderr, "  file size: %s\n", formatBytes(fileSize))
	fmt.Fprintf(os.Stderr, "  path:      %s\n\n", packFile)

	return nil
}

func runCCFTSSearch(ctx context.Context, crawlID, engineName, query string, limit, offset int, addr string) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}

	homeDir, _ := os.UserHomeDir()
	outputDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "fts", engineName)

	eng, err := index.NewEngine(engineName)
	if err != nil {
		return err
	}
	if addr != "" {
		if setter, ok := eng.(index.AddrSetter); ok {
			setter.SetAddr(addr)
		} else {
			fmt.Fprintf(os.Stderr, "warning: engine %q does not support --addr flag\n", engineName)
		}
	}

	if err := eng.Open(ctx, outputDir); err != nil {
		return fmt.Errorf("open engine: %w", err)
	}
	defer eng.Close()

	start := time.Now()
	results, err := eng.Search(ctx, index.Query{
		Text:   query,
		Limit:  limit,
		Offset: offset,
	})
	elapsed := time.Since(start)

	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	fmt.Fprintf(os.Stderr, "── Results for %q (engine: %s, %d hits, %s) ──\n",
		query, engineName, results.Total, elapsed.Round(time.Microsecond))
	fmt.Fprintf(os.Stderr, "  %-4s %-8s %-40s %s\n", "#", "Score", "DocID", "Snippet")
	fmt.Fprintf(os.Stderr, "  %-4s %-8s %-40s %s\n", "──", "────────", strings.Repeat("─", 40), strings.Repeat("─", 40))

	for i, hit := range results.Hits {
		snippet := hit.Snippet
		if len(snippet) > 80 {
			snippet = snippet[:80] + "..."
		}
		// Replace newlines with spaces for display
		snippet = strings.ReplaceAll(snippet, "\n", " ")
		snippet = strings.ReplaceAll(snippet, "\r", "")
		fmt.Fprintf(os.Stderr, "  %-4d %-8.2f %-40s %s\n",
			i+1+offset, hit.Score, hit.DocID, snippet)
	}

	return nil
}

func newCCFTSDecompress() *cobra.Command {
	var (
		crawlID string
		workers int
		dryRun  bool
	)

	cmd := &cobra.Command{
		Use:   "decompress",
		Short: "Decompress .md.gz → .md (one-time, speeds up indexing)",
		Long: `Convert all .md.gz files in the markdown/ directory to plain .md files,
then delete the .gz originals. Run once before indexing to eliminate
gzip decompression overhead on every subsequent 'fts index' call.`,
		Example: `  search cc fts decompress                # decompress all .md.gz
  search cc fts decompress --dry-run      # preview without changes
  search cc fts decompress --workers 8    # use 8 parallel workers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCFTSDecompress(cmd.Context(), crawlID, workers, dryRun)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().IntVar(&workers, "workers", 0, "Parallel workers (0 = NumCPU)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	return cmd
}

func runCCFTSDecompress(ctx context.Context, crawlID string, workers int, dryRun bool) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	homeDir, _ := os.UserHomeDir()
	markdownDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "markdown")

	if _, err := os.Stat(markdownDir); os.IsNotExist(err) {
		return fmt.Errorf("markdown dir not found: %s", markdownDir)
	}

	fmt.Fprintf(os.Stderr, "scanning %s...\n", markdownDir)

	// Collect all .md.gz files
	var files []string
	err := filepath.WalkDir(markdownDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md.gz") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk: %w", err)
	}

	total := len(files)
	if total == 0 {
		fmt.Fprintf(os.Stderr, "no .md.gz files found in %s\n", markdownDir)
		return nil
	}

	fmt.Fprintf(os.Stderr, "found %d .md.gz files\n", total)
	if dryRun {
		fmt.Fprintf(os.Stderr, "dry-run: would decompress %d files to .md and remove .gz\n", total)
		return nil
	}

	var (
		done   atomic.Int64
		errors atomic.Int64
		readB  atomic.Int64
		writeB atomic.Int64
	)

	start := time.Now()
	fileCh := make(chan string, workers*4)

	// Progress reporter
	stopProgress := make(chan struct{})
	go func() {
		tick := time.NewTicker(500 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				d := done.Load()
				elapsed := time.Since(start).Seconds()
				rate := float64(0)
				if elapsed > 0 {
					rate = float64(d) / elapsed
				}
				pct := float64(d) / float64(total) * 100
				fmt.Fprintf(os.Stderr, "\r\033[Kdecompressing: %d/%d (%.1f%%) │ %.0f files/s │ read %s → write %s",
					d, total, pct, rate, formatBytes(readB.Load()), formatBytes(writeB.Load()))
			case <-stopProgress:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Feed goroutine
	go func() {
		defer close(fileCh)
		for _, f := range files {
			select {
			case fileCh <- f:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for gzPath := range fileCh {
				if ctx.Err() != nil {
					return
				}

				// Track compressed size
				fi, statErr := os.Stat(gzPath)
				if statErr == nil {
					readB.Add(fi.Size())
				}

				// Read + decompress
				f, err := os.Open(gzPath)
				if err != nil {
					errors.Add(1)
					done.Add(1)
					continue
				}
				gr, err := gzip.NewReader(f)
				if err != nil {
					f.Close()
					errors.Add(1)
					done.Add(1)
					continue
				}
				data, err := io.ReadAll(gr)
				gr.Close()
				f.Close()
				if err != nil {
					errors.Add(1)
					done.Add(1)
					continue
				}

				writeB.Add(int64(len(data)))

				// Write plain .md
				mdPath := strings.TrimSuffix(gzPath, ".gz")
				if err := os.WriteFile(mdPath, data, 0o644); err != nil {
					errors.Add(1)
					done.Add(1)
					continue
				}

				// Remove .gz
				os.Remove(gzPath)
				done.Add(1)
			}
		}()
	}
	wg.Wait()
	close(stopProgress)
	fmt.Fprintln(os.Stderr) // newline after progress

	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, "\n── Decompress Complete ─────────────────────────────\n")
	fmt.Fprintf(os.Stderr, "  files:     %d\n", done.Load())
	fmt.Fprintf(os.Stderr, "  errors:    %d\n", errors.Load())
	fmt.Fprintf(os.Stderr, "  elapsed:   %s\n", elapsed.Round(time.Millisecond))
	if elapsed.Seconds() > 0 {
		fmt.Fprintf(os.Stderr, "  rate:      %.0f files/s\n", float64(done.Load())/elapsed.Seconds())
	}
	fmt.Fprintf(os.Stderr, "  read:      %s (.gz compressed)\n", formatBytes(readB.Load()))
	fmt.Fprintf(os.Stderr, "  written:   %s (plain .md)\n", formatBytes(writeB.Load()))
	return ctx.Err()
}

func detectLatestCrawl() string {
	homeDir, _ := os.UserHomeDir()
	ccDir := filepath.Join(homeDir, "data", "common-crawl")
	entries, err := os.ReadDir(ccDir)
	if err != nil {
		return "CC-MAIN-2026-08"
	}
	// Find latest CC-MAIN-* directory
	latest := ""
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "CC-MAIN-") {
			if e.Name() > latest {
				latest = e.Name()
			}
		}
	}
	if latest == "" {
		return "CC-MAIN-2026-08"
	}
	return latest
}
