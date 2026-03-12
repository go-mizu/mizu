package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	crawler "github.com/go-mizu/mizu/blueprints/search/pkg/scrape"
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
  $HOME/data/export/<format>/<domain>/

Examples:
  search export example.com
  search export example.com --format markdown
  search export example.com --cc --format html`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := crawler.NormalizeDomain(args[0])
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
		switch s.Phase {
		case "assets":
			fmt.Printf("  \033[1;35m[assets %d/%d]\033[0m \033[1;32m%s\033[0m",
				s.AssetsDown+s.AssetsFailed, s.AssetsTotal, fmtExportBytes(s.AssetsBytes))
			if s.AssetsFailed > 0 {
				fmt.Printf(" \033[1;31m%d failed\033[0m", s.AssetsFailed)
			}
			fmt.Println()
		default:
			fmt.Printf("  \033[1;36m[%d/%d]\033[0m \033[1;32m%.0f pages/s\033[0m (%.0f%%)\n",
				s.PagesExported, s.PagesTotal, s.PagesPerSec, s.Progress*100)
		}
	}

	metric, err := task.Run(ctx, emit)
	if err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("  Export failed: %v", err)))
		return err
	}

	fmt.Println()
	summary := fmt.Sprintf("  Exported %d pages", metric.Pages)
	if metric.Assets > 0 {
		summary += fmt.Sprintf(" + %d assets", metric.Assets)
	}
	summary += fmt.Sprintf(" in %s", time.Since(start).Truncate(time.Second))
	fmt.Println(successStyle.Render(summary))
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
		fmt.Printf("  \033[1;36m[%d/%d]\033[0m \033[1;32m%.0f pages/s\033[0m (%.0f%%)\n",
			s.PagesExported, s.PagesTotal, s.PagesPerSec, s.Progress*100)
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

func fmtExportBytes(b int64) string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%d B", b)
	case b < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	case b < 1024*1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	default:
		return fmt.Sprintf("%.2f GB", float64(b)/(1024*1024*1024))
	}
}

