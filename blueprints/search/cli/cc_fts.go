package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	// Import all drivers for registration.
	// duckdb registration happens via cli/duckdb_ops.go (excluded when -tags chdb).
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/chdb"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/devnull"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/sqlite"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/bleve"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/clickhouse"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/elasticsearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/flower/dahlia"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/flower/rose"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/meilisearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/opensearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/postgres"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/quickwit"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/tantivy-lnx"
	"github.com/spf13/cobra"
)

// warcIndexFromPath extracts the zero-padded 5-digit WARC file index from a WARC
// filename. Falls back to fmt.Sprintf("%05d", fallback) if not parseable.
//
//	"CC-MAIN-20260206181458-20260206211458-00000.warc.gz" → "00000"
func warcIndexFromPath(warcPath string, fallback int) string {
	base := filepath.Base(warcPath)
	name := strings.TrimSuffix(strings.TrimSuffix(base, ".gz"), ".warc")
	parts := strings.Split(name, "-")
	if last := parts[len(parts)-1]; len(last) == 5 {
		if _, err := strconv.Atoi(last); err == nil {
			return last
		}
	}
	return fmt.Sprintf("%05d", fallback)
}

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
	cmd.AddCommand(newCCFTSPack())
	cmd.AddCommand(newCCFTSWeb())
	cmd.AddCommand(newCCFTSDashboard())
	return cmd
}

func newCCFTSIndex() *cobra.Command {
	var (
		crawlID   string
		fileIdx   string
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
  search cc fts index --engine devnull --source bin      # benchmark from flatbin pack
  search cc fts index --engine devnull --source markdown --file 0  # benchmark from bin.gz pack`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCFTSIndex(cmd.Context(), crawlID, fileIdx, engine, source, batchSize, workers, addr)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "0", "File index, range (0-9)")
	cmd.Flags().StringVar(&engine, "engine", "duckdb", "FTS engine: "+strings.Join(index.List(), ", "))
	cmd.Flags().StringVar(&source, "source", "files", "Source: files, parquet, bin, markdown, duckdb")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "Documents per batch insert")
	cmd.Flags().IntVar(&workers, "workers", 0, "Parallel file readers (0 = NumCPU)")
	cmd.Flags().StringVar(&addr, "addr", "", "Service address for external engines (e.g. http://localhost:7700)")
	return cmd
}

func newCCFTSSearch() *cobra.Command {
	var (
		crawlID string
		fileIdx string
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
			return runCCFTSSearch(cmd.Context(), crawlID, fileIdx, engine, query, limit, offset, addr)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "", "File index to search (default: all WARCs)")
	cmd.Flags().StringVar(&engine, "engine", "duckdb", "FTS engine")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "Result offset")
	cmd.Flags().StringVar(&addr, "addr", "", "Service address for external engines")
	return cmd
}

func runCCFTSIndex(ctx context.Context, crawlID, fileIdx, engineName, source string, batchSize, workers int, addr string) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}

	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, "data", "common-crawl", crawlID)

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	selected, err := ccParseFileSelector(fileIdx, len(paths))
	if err != nil {
		return fmt.Errorf("--file: %w", err)
	}

	for _, idx := range selected {
		warcIdx := warcIndexFromPath(paths[idx], idx)
		outputDir := filepath.Join(baseDir, "fts", engineName, warcIdx)

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

		var stats *index.PipelineStats
		var pipeErr error

		if source == "files" {
			sourceDir := filepath.Join(baseDir, "markdown", warcIdx)
			if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
				eng.Close()
				return fmt.Errorf("markdown dir not found: %s", sourceDir)
			}

			fmt.Fprintf(os.Stderr, "indexing %s → %s (engine=%s, batch=%d, workers=%d)\n",
				sourceDir, outputDir, engineName, batchSize, workers)

			cfg := index.PipelineConfig{
				SourceDir: sourceDir,
				BatchSize: batchSize,
				Workers:   workers,
			}
			progress := makeFTSIndexProgress(engineName, outputDir)
			stats, pipeErr = index.RunPipeline(ctx, eng, cfg, progress)
			fmt.Fprintln(os.Stderr) // newline after progress
		} else {
			packDir := filepath.Join(baseDir, "pack")
			packFile, perr := packFilePath(packDir, source, warcIdx)
			if perr != nil {
				eng.Close()
				return perr
			}
			if _, err := os.Stat(packFile); os.IsNotExist(err) {
				eng.Close()
				return fmt.Errorf("pack file not found: %s\n  run: search cc fts pack --format %s --file %s", packFile, source, fileIdx)
			}
			fmt.Fprintf(os.Stderr, "indexing [%s ← %s] from %s\n", engineName, source, packFile)
			progress := makeFTSPackProgress(engineName, source, outputDir)
			stats, pipeErr = runCCFTSIndexFromPackFile(ctx, source, eng, packFile, batchSize, progress)
			fmt.Fprintln(os.Stderr) // newline after progress
		}

		if pipeErr != nil {
			eng.Close()
			return pipeErr
		}

		if err := ftsCreateDuckDBIndex(ctx, eng); err != nil {
			eng.Close()
			return err
		}

		engineStats, _ := eng.Stats(ctx)
		elapsed := time.Since(stats.StartTime)
		avgRate := float64(stats.DocsIndexed.Load()) / elapsed.Seconds()

		fmt.Fprintf(os.Stderr, "\n── FTS Index Complete ──────────────────────────\n")
		fmt.Fprintf(os.Stderr, "  engine:    %s\n", engineName)
		fmt.Fprintf(os.Stderr, "  source:    %s\n", source)
		fmt.Fprintf(os.Stderr, "  warc:      %s\n", warcIdx)
		fmt.Fprintf(os.Stderr, "  docs:      %d\n", engineStats.DocCount)
		fmt.Fprintf(os.Stderr, "  errors:    %d\n", stats.Errors.Load())
		fmt.Fprintf(os.Stderr, "  elapsed:   %s\n", elapsed.Round(100*time.Millisecond))
		fmt.Fprintf(os.Stderr, "  avg rate:  %.0f docs/s\n", avgRate)
		fmt.Fprintf(os.Stderr, "  peak RSS:  %d MB\n", stats.PeakRSSMB.Load())
		fmt.Fprintf(os.Stderr, "  disk:      %s\n", formatBytes(engineStats.DiskBytes))
		fmt.Fprintf(os.Stderr, "  path:      %s\n", outputDir)

		eng.Close()
	}
	return nil
}

// makeFTSIndexProgress returns a ProgressFunc for the files pipeline.
func makeFTSIndexProgress(engineName, outputDir string) index.ProgressFunc {
	return func(stats *index.PipelineStats) {
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
}

// makeFTSPackProgress returns a PackProgressFunc for pack-based pipelines.
func makeFTSPackProgress(engineName, source, outputDir string) index.PackProgressFunc {
	return func(done, total int64, elapsed time.Duration) {
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
}

// runCCFTSIndexFromPackFile indexes documents from a pre-packed file into eng.
// The caller is responsible for opening eng before calling and closing it after.
// source determines which reader is used; packFile is the absolute path to the pack file.
func runCCFTSIndexFromPackFile(ctx context.Context, source string, eng index.Engine, packFile string, batchSize int, progress index.PackProgressFunc) (*index.PipelineStats, error) {
	// Fast path: if the engine implements BulkLoader and source is parquet,
	// bypass the Go streaming pipeline entirely — let the engine ingest the
	// file natively (e.g. DuckDB's read_parquet() vectorised path).
	if bl, ok := eng.(index.BulkLoader); ok && source == "parquet" {
		t0 := time.Now()
		fmt.Fprintf(os.Stderr, "bulk loading via native read_parquet()...\n")
		n, bulkErr := bl.BulkLoad(ctx, "parquet", packFile)
		if bulkErr != nil {
			return nil, fmt.Errorf("bulk load: %w", bulkErr)
		}
		stats := &index.PipelineStats{StartTime: t0}
		stats.DocsIndexed.Store(n)
		elapsed := time.Since(t0)
		fmt.Fprintf(os.Stderr, "  bulk load: %d docs in %s (%.0f docs/s)\n",
			n, elapsed.Round(10*time.Millisecond), float64(n)/elapsed.Seconds())
		return stats, nil
	}

	switch source {
	case "parquet":
		return index.RunPipelineFromParquet(ctx, eng, packFile, batchSize, progress)
	case "bin":
		return index.RunPipelineFromFlatBin(ctx, eng, packFile, batchSize, progress)
	case "markdown":
		return index.RunPipelineFromFlatBinGz(ctx, eng, packFile, batchSize, progress)
	case "duckdb":
		return runPipelineFromDuckDBRaw(ctx, eng, packFile, batchSize, progress)
	default:
		return nil, fmt.Errorf("unknown source %q (valid: files, parquet, bin, markdown, duckdb)", source)
	}
}

// ftsCreateDuckDBIndex calls CreateFTSIndex on any engine that implements it
// (all DuckDB variants). The old name-based "duckdb" check was too narrow.
func ftsCreateDuckDBIndex(ctx context.Context, eng index.Engine) error {
	type ftsIndexer interface{ CreateFTSIndex(context.Context) error }
	ddb, ok := eng.(ftsIndexer)
	if !ok {
		return nil
	}
	fmt.Fprintf(os.Stderr, "creating FTS index (BM25)...\n")
	if err := ddb.CreateFTSIndex(ctx); err != nil {
		return fmt.Errorf("create FTS index: %w", err)
	}
	return nil
}

func packFilePath(packDir, format, warcIdx string) (string, error) {
	switch format {
	case "parquet":
		return filepath.Join(packDir, "parquet", warcIdx+".parquet"), nil
	case "bin":
		return filepath.Join(packDir, "bin", warcIdx+".bin"), nil
	case "duckdb":
		return filepath.Join(packDir, "duckdb", warcIdx+".duckdb"), nil
	case "markdown":
		return filepath.Join(packDir, "markdown", warcIdx+".bin.gz"), nil
	default:
		return "", fmt.Errorf("unknown format %q (valid: parquet, bin, duckdb, markdown)", format)
	}
}

func newCCFTSPack() *cobra.Command {
	var (
		crawlID   string
		fileIdx   string
		format    string
		batchSize int
		workers   int
	)

	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Pack CC markdown files into a high-perf bundle for FTS import benchmarking",
		Long: `Pre-compute markdown files into one or more fast-load formats.
Packed files are stored in pack/{format}/{warcIdx}.{ext} and can be used with 'fts index --source <format>'.`,
		Example: `  search cc fts pack --format parquet   # Parquet columnar
  search cc fts pack --format bin        # flat binary (fastest read)
  search cc fts pack --format markdown   # concatenated gzip (bin.gz)
  search cc fts pack --format duckdb    # DuckDB raw table
  search cc fts pack --format all        # all four formats`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCFTSPack(cmd.Context(), crawlID, fileIdx, format, batchSize, workers)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "0", "File index, range (0-9), or all")
	cmd.Flags().StringVar(&format, "format", "all", "Format: parquet, bin, duckdb, markdown, all")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "Documents per batch")
	cmd.Flags().IntVar(&workers, "workers", 0, "Parallel file readers (0 = NumCPU)")
	return cmd
}

func runCCFTSPack(ctx context.Context, crawlID, fileIdx, format string, batchSize, workers int) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}

	homeDir, _ := os.UserHomeDir()
	packDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "pack")

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	selected, err := ccParseFileSelector(fileIdx, len(paths))
	if err != nil {
		return fmt.Errorf("--file: %w", err)
	}

	formats := []string{format}
	if format == "all" {
		formats = []string{"parquet", "bin", "duckdb", "markdown"}
	}

	for _, idx := range selected {
		warcIdx := warcIndexFromPath(paths[idx], idx)
		markdownDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "markdown", warcIdx)
		if _, err := os.Stat(markdownDir); os.IsNotExist(err) {
			return fmt.Errorf("markdown dir not found: %s\n  run: search cc warc markdown --file %d", markdownDir, idx)
		}

		for _, fmt_ := range formats {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			packFile, err := packFilePath(packDir, fmt_, warcIdx)
			if err != nil {
				return err
			}
			if err := runPackFormat(ctx, fmt_, markdownDir, packFile, batchSize, workers); err != nil {
				return fmt.Errorf("pack %s: %w", fmt_, err)
			}
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
	case "markdown":
		stats, err = index.PackFlatBinGz(ctx, markdownDir, packFile, workers, batchSize, progress)
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

func runCCFTSSearch(ctx context.Context, crawlID, fileIdx, engineName, query string, limit, offset int, addr string) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}

	homeDir, _ := os.UserHomeDir()
	ftsBase := filepath.Join(homeDir, "data", "common-crawl", crawlID, "fts", engineName)

	// Collect target directories.
	var targetDirs []string
	if fileIdx != "" {
		// Single WARC mode: resolve via manifest.
		client := cc.NewClient("", 4)
		paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
		if err != nil {
			return err
		}
		selected, err := ccParseFileSelector(fileIdx, len(paths))
		if err != nil {
			return err
		}
		for _, idx := range selected {
			targetDirs = append(targetDirs, filepath.Join(ftsBase, warcIndexFromPath(paths[idx], idx)))
		}
	} else {
		// Fan-out: discover all per-WARC directories.
		entries, err := os.ReadDir(ftsBase)
		if err != nil {
			return fmt.Errorf("no FTS index at %s — run 'cc fts index' first", ftsBase)
		}
		for _, e := range entries {
			if e.IsDir() {
				targetDirs = append(targetDirs, filepath.Join(ftsBase, e.Name()))
			}
		}
		if len(targetDirs) == 0 {
			return fmt.Errorf("no per-WARC FTS indices found under %s", ftsBase)
		}
	}

	// Search all target dirs in parallel, collect results.
	type shardResult struct {
		hits  []index.Hit
		total int
		err   error
	}
	results := make([]shardResult, len(targetDirs))
	var wg sync.WaitGroup
	for i, dir := range targetDirs {
		i, dir := i, dir
		wg.Add(1)
		go func() {
			defer wg.Done()
			eng, err := index.NewEngine(engineName)
			if err != nil {
				results[i].err = err
				return
			}
			if addr != "" {
				if setter, ok := eng.(index.AddrSetter); ok {
					setter.SetAddr(addr)
				}
			}
			if err := eng.Open(ctx, dir); err != nil {
				results[i].err = fmt.Errorf("open %s: %w", dir, err)
				return
			}
			defer eng.Close()

			res, err := eng.Search(ctx, index.Query{Text: query, Limit: limit + offset, Offset: 0})
			if err != nil {
				results[i].err = err
				return
			}
			results[i].hits = res.Hits
			results[i].total = res.Total
		}()
	}
	wg.Wait()

	// Merge: collect all hits, sort by score descending, take top limit after offset.
	var allHits []index.Hit
	var totalCount int
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "warning: shard error: %v\n", r.err)
			continue
		}
		allHits = append(allHits, r.hits...)
		totalCount += r.total
	}
	sort.Slice(allHits, func(i, j int) bool {
		return allHits[i].Score > allHits[j].Score
	})
	if offset < len(allHits) {
		allHits = allHits[offset:]
	} else {
		allHits = nil
	}
	if len(allHits) > limit {
		allHits = allHits[:limit]
	}

	fmt.Fprintf(os.Stderr, "── Results for %q (engine: %s, shards: %d, total: %d) ──\n",
		query, engineName, len(targetDirs), totalCount)
	for i, hit := range allHits {
		snippet := strings.ReplaceAll(hit.Snippet, "\n", " ")
		if len(snippet) > 80 {
			snippet = snippet[:80] + "..."
		}
		fmt.Fprintf(os.Stderr, "  %-4d %-8.2f %-40s %s\n", i+1+offset, hit.Score, hit.DocID, snippet)
	}

	return nil
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
