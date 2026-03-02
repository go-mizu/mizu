package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
	"github.com/spf13/cobra"
)

func newCCMarkdown() *cobra.Command {
	var (
		crawlID   string
		bodyStore string
		workers   int
		force     bool
		fast      bool
	)

	cmd := &cobra.Command{
		Use:   "markdown",
		Short: "Convert HTML bodies to clean readable markdown",
		Long: `Reads gzipped HTML files from the body store, extracts readable content
using trafilatura (readability + fallback), converts to markdown, and writes
gzipped markdown files with the same directory structure.

Tracks all conversions in a DuckDB index with size, token estimates, timing.
Skips already-converted files unless --force is set.

  Default mode: trafilatura (F1=0.91) — ~100-200 files/s per worker
  Fast mode:    go-readability (Mozilla Readability.js) — ~600-1000 files/s per worker
                3-8x faster at the cost of slightly lower extraction quality on noisy pages.

Input:  ~/data/common-crawl/bodies/ab/cd/ef...89.gz      (raw HTML)
Output: ~/data/common-crawl/markdown/ab/cd/ef...89.md.gz  (clean markdown)
Index:  ~/data/common-crawl/markdown/index.duckdb
`,
		Example: `  search cc markdown
  search cc markdown --fast
  search cc markdown --force
  search cc markdown --workers 16
  search cc markdown --body-store ~/data/common-crawl/bodies`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCMarkdown(cmd.Context(), crawlID, bodyStore, workers, force, fast)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&bodyStore, "body-store", "", "Body store directory (default: ~/data/common-crawl/bodies)")
	cmd.Flags().IntVar(&workers, "workers", 0, "Parallel workers (default: NumCPU)")
	cmd.Flags().BoolVar(&force, "force", false, "Re-convert existing files")
	cmd.Flags().BoolVar(&fast, "fast", false, "Use go-readability instead of trafilatura (3-8x faster, slightly lower quality)")

	return cmd
}

func runCCMarkdown(ctx context.Context, crawlID, bodyStore string, workers int, force, fast bool) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("HTML → Markdown Conversion"))
	fmt.Println()

	ccCfg := cc.DefaultConfig()
	if crawlID != "" {
		ccCfg.CrawlID = crawlID
	}

	// Resolve body store directory
	if bodyStore == "" {
		bodyStore = filepath.Join(ccCfg.DataDir, "bodies")
	} else if strings.HasPrefix(bodyStore, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home dir: %w", err)
		}
		bodyStore = filepath.Join(home, bodyStore[2:])
	}

	// Output directory: sibling to bodies
	outputDir := filepath.Join(filepath.Dir(bodyStore), "markdown")
	indexPath := filepath.Join(outputDir, "index.duckdb")

	// Resolve workers for display (Walk also defaults to NumCPU if 0)
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Verify input exists and is a directory
	info, err := os.Stat(bodyStore)
	if err != nil {
		return fmt.Errorf("body store not found: %s\n  Run 'search cc recrawl --body-store %s' first", bodyStore, bodyStore)
	}
	if !info.IsDir() {
		return fmt.Errorf("body store is not a directory: %s", bodyStore)
	}

	fmt.Printf("  Input:   %s\n", labelStyle.Render(bodyStore))
	fmt.Printf("  Output:  %s\n", labelStyle.Render(outputDir))
	fmt.Printf("  Index:   %s\n", labelStyle.Render(indexPath))
	fmt.Printf("  Workers: %s\n", infoStyle.Render(fmt.Sprintf("%d", workers)))
	mode := "incremental (skip existing)"
	if force {
		mode = "force (re-convert all)"
	}
	extractor := "trafilatura (quality)"
	if fast {
		extractor = "go-readability (fast)"
	}
	if force {
		fmt.Printf("  Mode:    %s\n", warningStyle.Render(mode))
	} else {
		fmt.Printf("  Mode:    %s\n", infoStyle.Render(mode))
	}
	fmt.Printf("  Engine:  %s\n", infoStyle.Render(extractor))
	fmt.Println()

	cfg := markdown.WalkConfig{
		InputDir:  bodyStore,
		OutputDir: outputDir,
		IndexPath: indexPath,
		Workers:   workers,
		Force:     force,
		BatchSize: 1000,
		Fast:      fast,
	}

	// Progress display
	progressFn := func(converted, skipped, errors, total int64, htmlBytes, mdBytes int64, elapsed time.Duration) {
		done := converted + skipped + errors
		pct := float64(0)
		if total > 0 {
			pct = float64(done) / float64(total) * 100
		}
		rate := float64(0)
		if elapsed.Seconds() > 0 {
			rate = float64(converted) / elapsed.Seconds()
		}

		fmt.Printf("\r\033[K  Converting: %s / %s (%.1f%%)  %.0f/s  HTML: %s → MD: %s",
			ccFmtInt64(done), ccFmtInt64(total), pct, rate,
			formatBytes(htmlBytes), formatBytes(mdBytes))
		if htmlBytes > 0 && mdBytes > 0 {
			fmt.Printf(" (-%.1f%%)", (1.0-float64(mdBytes)/float64(htmlBytes))*100)
		}
		if skipped > 0 {
			fmt.Printf("  skip: %s", ccFmtInt64(skipped))
		}
		if errors > 0 {
			fmt.Printf("  err: %s", ccFmtInt64(errors))
		}
	}

	stats, err := markdown.Walk(ctx, cfg, progressFn)
	if err != nil {
		return err
	}

	// Clear progress line
	fmt.Printf("\r\033[K")

	// Summary
	fmt.Println(successStyle.Render("  Done!"))
	fmt.Println()
	fmt.Printf("  Converted: %s files in %s",
		infoStyle.Render(ccFmtInt64(stats.Converted)),
		stats.Duration.Round(time.Millisecond))
	if stats.Duration.Seconds() > 0 {
		fmt.Printf(" (%.0f/s)", float64(stats.Converted)/stats.Duration.Seconds())
	}
	fmt.Println()

	if stats.Skipped > 0 {
		fmt.Printf("  Skipped:   %s (already converted)\n", ccFmtInt64(stats.Skipped))
	}
	if stats.Errors > 0 {
		fmt.Printf("  Errors:    %s\n", warningStyle.Render(ccFmtInt64(stats.Errors)))
	}
	fmt.Println()

	if stats.TotalHTMLBytes > 0 && stats.Converted > 0 && stats.TotalMDBytes > 0 {
		reductionPct := (1.0 - float64(stats.TotalMDBytes)/float64(stats.TotalHTMLBytes)) * 100
		fmt.Printf("  HTML:      %s\n", formatBytes(stats.TotalHTMLBytes))
		fmt.Printf("  Markdown:  %s (-%.1f%%)\n", formatBytes(stats.TotalMDBytes), reductionPct)

		avgHTMLTokens := int64(markdown.EstimateTokens(int(stats.TotalHTMLBytes / stats.Converted)))
		avgMDTokens := int64(markdown.EstimateTokens(int(stats.TotalMDBytes / stats.Converted)))
		fmt.Printf("  Tokens:    avg %s → %s (-%.1f%%)\n",
			ccFmtInt64(avgHTMLTokens), ccFmtInt64(avgMDTokens), reductionPct)
	}
	fmt.Println()

	// Error breakdown from DuckDB index
	if stats.Errors > 0 {
		cats, err := markdown.QueryErrors(indexPath)
		if err == nil && len(cats) > 0 {
			fmt.Println(subtitleStyle.Render("  Error Breakdown"))
			fmt.Println()
			for _, c := range cats {
				pct := float64(c.Count) / float64(stats.Errors) * 100
				fmt.Printf("    %-30s %s (%.1f%%)\n",
					c.Category, warningStyle.Render(ccFmtInt64(int64(c.Count))), pct)
			}
			fmt.Println()
		}
	}

	return nil
}
