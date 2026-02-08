package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/spf13/cobra"
)

func newCCSite() *cobra.Command {
	var (
		mode    string
		crawlID string
		workers int
		timeout int
		maxBody int
		resume  bool
	)

	cmd := &cobra.Command{
		Use:   "site <domain>",
		Short: "Extract all pages for a domain from Common Crawl",
		Long: `Extract all web pages for a domain from Common Crawl archives.

Three extraction modes:

  urls   URL + metadata only (CDX API, no WARC fetching — fastest)
  links  URLs + outgoing links from HTML pages (WARC fetch)
  full   URLs + links + full body content (WARC fetch)

Data is stored in DuckDB at $HOME/data/common-crawl/site/{domain}/site.duckdb

Examples:
  search cc site duckdb.org
  search cc site duckdb.org --mode links
  search cc site duckdb.org --mode full --workers 1000
  search cc site duckdb.org --mode urls --crawl CC-MAIN-2025-51`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			var siteMode cc.SiteMode
			switch strings.ToLower(mode) {
			case "urls":
				siteMode = cc.SiteModeURLs
			case "links":
				siteMode = cc.SiteModeLinks
			case "full":
				siteMode = cc.SiteModeFull
			default:
				return fmt.Errorf("invalid mode %q: use urls, links, or full", mode)
			}

			return runCCSite(cmd.Context(), domain, siteMode, crawlID, workers, timeout, maxBody, resume)
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "urls", "Extraction mode: urls, links, full")
	cmd.Flags().StringVar(&crawlID, "crawl", "CC-MAIN-2026-04", "Crawl ID")
	cmd.Flags().IntVar(&workers, "workers", 500, "WARC fetch workers")
	cmd.Flags().IntVar(&timeout, "timeout", 30000, "Per-request timeout ms")
	cmd.Flags().IntVar(&maxBody, "max-body", 512*1024, "Max body size bytes")
	cmd.Flags().BoolVar(&resume, "resume", false, "Skip already-extracted URLs")

	return cmd
}

func runCCSite(ctx context.Context, domain string, mode cc.SiteMode, crawlID string, workers, timeout, maxBody int, resume bool) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Common Crawl Site Extraction"))
	fmt.Println()

	modeStr := "urls"
	switch mode {
	case cc.SiteModeLinks:
		modeStr = "links"
	case cc.SiteModeFull:
		modeStr = "full"
	}

	cfg := cc.DefaultSiteConfig(domain)
	cfg.CrawlID = crawlID
	cfg.Mode = mode
	cfg.Workers = workers
	cfg.Timeout = time.Duration(timeout) * time.Millisecond
	cfg.MaxBodySize = maxBody
	cfg.Resume = resume

	ccCfg := cc.DefaultConfig()
	siteDir := ccCfg.SiteDir(domain)

	fmt.Printf("  Domain:   %s\n", infoStyle.Render(domain))
	fmt.Printf("  Crawl:    %s\n", infoStyle.Render(crawlID))
	fmt.Printf("  Mode:     %s\n", infoStyle.Render(modeStr))
	fmt.Printf("  Output:   %s\n", labelStyle.Render(filepath.Join(siteDir, "site.duckdb")))
	fmt.Println()

	// Step 1: CDX API lookup
	fmt.Println(infoStyle.Render("Step 1: Discovering URLs via CDX API..."))

	start := time.Now()
	entries, err := cc.LookupDomainAll(ctx, crawlID, domain, 10, func(done, total int) {
		fmt.Printf("\r  CDX page %d/%d", done, total)
	})
	if err != nil {
		return fmt.Errorf("CDX lookup: %w", err)
	}
	fmt.Println()

	if len(entries) == 0 {
		fmt.Println(warningStyle.Render("  No pages found for this domain in this crawl"))
		return nil
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  Found %s URLs (%s)",
		ccFmtInt64(int64(len(entries))), time.Since(start).Truncate(time.Millisecond))))
	fmt.Println()

	// Open SiteDB
	sdb, err := cc.NewSiteDB(siteDir, cfg.BatchSize)
	if err != nil {
		return fmt.Errorf("opening site db: %w", err)
	}
	defer sdb.Close()

	sdb.SetMeta("domain", domain)
	sdb.SetMeta("crawl_id", crawlID)
	sdb.SetMeta("mode", modeStr)
	sdb.SetMeta("started_at", time.Now().Format(time.RFC3339))

	// URLs mode: store CDX entries directly
	if mode == cc.SiteModeURLs {
		fmt.Println(infoStyle.Render("Step 2: Storing URL metadata..."))

		stats := cc.NewSiteStats(len(entries))
		extractor := cc.NewSiteExtractor(cfg, nil, sdb, stats)
		extractor.ExtractURLsOnly(entries)

		sdb.Flush(ctx)
		stats.Freeze()

		sdb.SetMeta("finished_at", time.Now().Format(time.RFC3339))
		sdb.SetMeta("total_pages", fmt.Sprintf("%d", len(entries)))

		fmt.Println(successStyle.Render(fmt.Sprintf("  Stored %s pages", ccFmtInt64(int64(len(entries))))))
		fmt.Println()
		fmt.Println(successStyle.Render("Extraction complete!"))
		fmt.Println(labelStyle.Render(fmt.Sprintf("  %s", sdb.Path())))
		fmt.Println()
		return nil
	}

	// Links or Full mode: need WARC fetching
	fmt.Println(infoStyle.Render("Step 2: Converting CDX entries to WARC pointers..."))

	var pointers []cc.WARCPointer
	var skippedConv int
	for _, e := range entries {
		ptr, convErr := cc.CDXJToWARCPointer(e, domain)
		if convErr != nil {
			skippedConv++
			continue
		}
		pointers = append(pointers, ptr)
	}

	if skippedConv > 0 {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Skipped %d entries (invalid offset/length)", skippedConv)))
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("  %s WARC pointers ready", ccFmtInt64(int64(len(pointers))))))
	fmt.Println()

	if len(pointers) == 0 {
		fmt.Println(warningStyle.Render("  No valid WARC pointers"))
		return nil
	}

	// Check for resume
	var skip map[string]bool
	if resume {
		fmt.Println(infoStyle.Render("Checking for previous results..."))
		skip, err = cc.LoadAlreadyExtracted(siteDir)
		if err != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  Could not load state: %v", err)))
		} else if len(skip) > 0 {
			fmt.Println(successStyle.Render(fmt.Sprintf("  Resuming: skipping %d already-extracted URLs", len(skip))))
		}
	}

	// Step 3: Fetch WARC records
	fmt.Println(infoStyle.Render(fmt.Sprintf("Step 3: Fetching WARC records (%d workers, mode=%s)...", workers, modeStr)))
	fmt.Println()

	client := cc.NewClient("", cfg.TransportShards)
	stats := cc.NewSiteStats(len(pointers))
	extractor := cc.NewSiteExtractor(cfg, client, sdb, stats)

	err = cc.RunSiteWithDisplay(ctx, extractor, pointers, skip, stats)

	sdb.Flush(ctx)
	sdb.SetMeta("finished_at", time.Now().Format(time.RFC3339))
	sdb.SetMeta("total_pages", fmt.Sprintf("%d", stats.Done()))
	sdb.SetMeta("total_links", fmt.Sprintf("%d", sdb.LinkCount()))

	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("Extraction finished with error: %v", err)))
	} else {
		fmt.Println(successStyle.Render("Extraction complete!"))
	}
	fmt.Printf("  Pages: %s  │  Links: %s\n", ccFmtInt64(sdb.PageCount()), ccFmtInt64(sdb.LinkCount()))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  %s", sdb.Path())))
	fmt.Println()

	return err
}
