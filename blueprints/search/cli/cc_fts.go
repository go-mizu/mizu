package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	// Import all drivers for registration
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/chdb"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/devnull"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/duckdb"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/sqlite"
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
	return cmd
}

func newCCFTSIndex() *cobra.Command {
	var (
		crawlID   string
		engine    string
		batchSize int
		workers   int
	)

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Build FTS index from CC markdown files",
		Example: `  search cc fts index --engine duckdb
  search cc fts index --engine sqlite --crawl CC-MAIN-2026-08
  search cc fts index --engine devnull  # benchmark I/O only`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCFTSIndex(cmd.Context(), crawlID, engine, batchSize, workers)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&engine, "engine", "duckdb", "FTS engine: "+strings.Join(index.List(), ", "))
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "Documents per batch insert")
	cmd.Flags().IntVar(&workers, "workers", 4, "Parallel file readers")
	return cmd
}

func newCCFTSSearch() *cobra.Command {
	var (
		crawlID string
		engine  string
		limit   int
		offset  int
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search the FTS index",
		Args:  cobra.MinimumNArgs(1),
		Example: `  search cc fts search "machine learning" --engine duckdb
  search cc fts search "climate change" --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			return runCCFTSSearch(cmd.Context(), crawlID, engine, query, limit, offset)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&engine, "engine", "duckdb", "FTS engine")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "Result offset")
	return cmd
}

func runCCFTSIndex(ctx context.Context, crawlID, engineName string, batchSize, workers int) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}

	homeDir, _ := os.UserHomeDir()
	sourceDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "markdown")
	outputDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "fts", engineName)

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("markdown dir not found: %s", sourceDir)
	}

	eng, err := index.NewEngine(engineName)
	if err != nil {
		return err
	}

	if err := eng.Open(ctx, outputDir); err != nil {
		return fmt.Errorf("open engine: %w", err)
	}
	defer eng.Close()

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

		// Progress bar
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

	// Create FTS index for DuckDB (post-insert step)
	if engineName == "duckdb" {
		fmt.Fprintf(os.Stderr, "creating FTS index (BM25)...\n")
		if ddb, ok := eng.(interface{ CreateFTSIndex(context.Context) error }); ok {
			if err := ddb.CreateFTSIndex(ctx); err != nil {
				return fmt.Errorf("create FTS index: %w", err)
			}
		}
	}

	// Final summary
	engineStats, _ := eng.Stats(ctx)
	elapsed := time.Since(stats.StartTime)
	avgRate := float64(stats.DocsIndexed.Load()) / elapsed.Seconds()

	fmt.Fprintf(os.Stderr, "\n── FTS Index Complete ──────────────────────────\n")
	fmt.Fprintf(os.Stderr, "  engine:    %s\n", engineName)
	fmt.Fprintf(os.Stderr, "  docs:      %d\n", engineStats.DocCount)
	fmt.Fprintf(os.Stderr, "  errors:    %d\n", stats.Errors.Load())
	fmt.Fprintf(os.Stderr, "  elapsed:   %s\n", elapsed.Round(100*time.Millisecond))
	fmt.Fprintf(os.Stderr, "  avg rate:  %.0f docs/s\n", avgRate)
	fmt.Fprintf(os.Stderr, "  peak RSS:  %d MB\n", stats.PeakRSSMB.Load())
	fmt.Fprintf(os.Stderr, "  disk:      %s\n", formatBytes(engineStats.DiskBytes))
	fmt.Fprintf(os.Stderr, "  path:      %s\n", outputDir)

	return nil
}

func runCCFTSSearch(ctx context.Context, crawlID, engineName, query string, limit, offset int) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}

	homeDir, _ := os.UserHomeDir()
	outputDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "fts", engineName)

	eng, err := index.NewEngine(engineName)
	if err != nil {
		return err
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
