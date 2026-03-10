package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/scrape"
	"github.com/spf13/cobra"
)

// NewExport creates the export CLI command.
func NewExport() *cobra.Command {
	var (
		format string
		useCC  bool
	)

	cmd := &cobra.Command{
		Use:   "export <domain>",
		Short: "Export a crawled domain to browsable offline site",
		Long: `Export a domain's crawled pages to a local, browsable site mirror.

Formats:
  html      Rewrite links + inline styles/images (default)
  markdown  Convert to Markdown with navigable internal links
  raw       Copy original HTML without link rewriting

Source:
  By default, exports from dcrawler data ($HOME/data/crawler/<domain>/).
  Use --cc to export from Common Crawl recrawl data instead.

Output:
  Scrape:  $HOME/data/crawler/<domain>/export/<format>/
  CC:      $HOME/data/common-crawl/export/<format>/<domain>/

Examples:
  search export example.com
  search export example.com --format markdown
  search export example.com --cc --format html`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := dcrawler.NormalizeDomain(args[0])
			if domain == "" {
				return fmt.Errorf("invalid domain: %s", args[0])
			}
			if format != "html" && format != "markdown" && format != "raw" {
				return fmt.Errorf("format must be html, markdown, or raw (got %q)", format)
			}

			if useCC {
				return runCCExportCLI(cmd.Context(), domain, format)
			}
			return runScrapeExportCLI(cmd.Context(), domain, format)
		},
	}

	cmd.Flags().StringVar(&format, "format", "html", "Export format: html, markdown, or raw")
	cmd.Flags().BoolVar(&useCC, "cc", false, "Export from Common Crawl data instead of dcrawler")

	return cmd
}

func runScrapeExportCLI(ctx context.Context, domain, format string) error {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "crawler")

	fmt.Println(subtitleStyle.Render("Export Domain (Scrape)"))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Domain:  %s", domain)))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Format:  %s", format)))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Data:    %s", filepath.Join(dataDir, domain))))
	fmt.Println()

	task := scrape.NewExportTask(domain, dataDir, format)
	start := time.Now()

	emit := func(s *scrape.ExportState) {
		if s.PagesExported%50 == 0 || s.Progress >= 1.0 {
			fmt.Printf("  [%d/%d] %.0f pages/s (%.0f%%)\n",
				s.PagesExported, s.PagesTotal, s.PagesPerSec, s.Progress*100)
		}
	}

	metric, err := task.Run(ctx, emit)
	if err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("  Export failed: %v", err)))
		return err
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Exported %d pages in %s", metric.Pages, time.Since(start).Truncate(time.Second))))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Output:  %s", metric.OutDir)))
	return nil
}

func runCCExportCLI(ctx context.Context, domain, format string) error {
	home, _ := os.UserHomeDir()
	// Find the latest crawl directory
	ccDir := filepath.Join(home, "data", "common-crawl")
	entries, err := os.ReadDir(ccDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", ccDir, err)
	}
	var crawlDir string
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.IsDir() && e.Name() != "export" {
			crawlDir = filepath.Join(ccDir, e.Name())
			break
		}
	}
	if crawlDir == "" {
		return fmt.Errorf("no crawl directory found in %s", ccDir)
	}

	fmt.Println(subtitleStyle.Render("Export Domain (Common Crawl)"))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Domain:    %s", domain)))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Format:    %s", format)))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  CrawlDir:  %s", crawlDir)))
	fmt.Println()

	task := cc.NewCCExportTask(domain, crawlDir, format)
	start := time.Now()

	emit := func(s *cc.CCExportState) {
		if s.PagesExported%50 == 0 || s.Progress >= 1.0 {
			fmt.Printf("  [%d/%d] %.0f pages/s (%.0f%%)\n",
				s.PagesExported, s.PagesTotal, s.PagesPerSec, s.Progress*100)
		}
	}

	metric, err := task.Run(ctx, emit)
	if err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("  Export failed: %v", err)))
		return err
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Exported %d pages in %s", metric.Pages, time.Since(start).Truncate(time.Second))))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Output:  %s", metric.OutDir)))
	return nil
}

