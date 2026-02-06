package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/analytics"
	"github.com/spf13/cobra"
)

// NewAnalytics creates the analytics command.
func NewAnalytics() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analytics",
		Short: "Generate comprehensive dataset analytics reports",
		Long: `Analyzes FineWeb-2 parquet data using DuckDB and generates rich Markdown reports
with 49 charts across 5 categories:

  1. Text Statistics      - length distributions, word frequencies, character analysis
  2. Temporal Analysis    - crawl patterns over time, dump coverage
  3. URL & Domain         - top domains, TLD distribution, URL structure
  4. Quality Metrics      - language scores, deduplication clusters
  5. Vietnamese Content   - diacritics, tones, vowels, content types

Examples:
  search analytics --lang vie_Latn
  search analytics --lang vie_Latn --split test
  search analytics --lang vie_Latn --split train`,
		RunE: runAnalytics,
	}

	home, _ := os.UserHomeDir()
	defaultData := filepath.Join(home, "data", "fineweb-2")

	cmd.Flags().String("lang", "vie_Latn", "Language code")
	cmd.Flags().StringSlice("split", []string{"train", "test"}, "Dataset splits to analyze")
	cmd.Flags().String("output", "", "Output directory (default: pkg/analytics/report/{lang}/)")
	cmd.Flags().String("data", defaultData, "Data directory")

	return cmd
}

func runAnalytics(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("FineWeb-2 Dataset Analytics (DuckDB)"))
	fmt.Println()

	lang, _ := cmd.Flags().GetString("lang")
	splits, _ := cmd.Flags().GetStringSlice("split")
	outputDir, _ := cmd.Flags().GetString("output")
	dataDir, _ := cmd.Flags().GetString("data")

	if outputDir == "" {
		outputDir = filepath.Join("pkg", "analytics", "report", lang)
	}

	for _, split := range splits {
		if err := runSplitAnalysis(ctx, dataDir, lang, split, outputDir); err != nil {
			fmt.Printf("  %s  Error analyzing %s: %v\n", errorStyle.Render("ERROR"), split, err)
			continue
		}
	}

	fmt.Println()
	fmt.Println(successStyle.Render("Analytics complete!"))
	fmt.Println()

	return nil
}

func runSplitAnalysis(ctx context.Context, dataDir, lang, split, outputDir string) error {
	parquetDir := filepath.Join(dataDir, lang, split)

	// Verify directory exists
	entries, err := os.ReadDir(parquetDir)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", parquetDir, err)
	}

	parquetCount := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".parquet") {
			parquetCount++
		}
	}
	if parquetCount == 0 {
		return fmt.Errorf("no parquet files found in %s", parquetDir)
	}

	fmt.Printf("  %s  Analyzing %s split (%d parquet files)...\n",
		infoStyle.Render(split), split, parquetCount)

	startTime := time.Now()

	// Create DuckDB analyzer
	analyzer, err := analytics.NewAnalyzer(parquetDir)
	if err != nil {
		return fmt.Errorf("creating analyzer: %w", err)
	}
	defer analyzer.Close()

	// Run all queries with progress
	progress := func(step, total int, label string) {
		fmt.Printf("\r    [%d/%d] %s...                    ", step, total, label)
	}

	report, err := analyzer.Run(ctx, progress)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("running analytics: %w", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\r    Completed %d queries in %s                    \n",
		23, elapsed.Round(time.Millisecond))

	// Write report
	outputPath := filepath.Join(outputDir, split+".md")
	fmt.Printf("    Writing report to %s...\n", outputPath)

	if err := analytics.WriteReport(report, split, lang, outputPath, parquetDir); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}

	fmt.Printf("  %s  %s report complete (%s)\n", successStyle.Render("OK"), split, outputPath)
	return nil
}
