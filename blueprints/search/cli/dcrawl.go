package cli

import (
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler"
	"github.com/spf13/cobra"
)

// NewCrawlDomain creates the crawl-domain CLI command.
func NewCrawlDomain() *cobra.Command {
	var (
		workers          int
		maxConns         int
		maxDepth         int
		maxPages         int
		timeout          int
		rateLimit        int
		transportShards  int
		storeBody        bool
		noLinks          bool
		noRobots         bool
		noSitemap        bool
		includeSubdomain bool
		resume           bool
		http1            bool
		continuous       bool
		crawlerDataDir   string
		userAgent        string
		seedFile         string
	)

	cmd := &cobra.Command{
		Use:   "crawl-domain <domain>",
		Short: "Crawl all pages from a single domain",
		Long: `High-throughput single-domain web crawler targeting 10K+ pages/second.

Uses HTTP/2 multiplexing, bloom filter URL dedup, BFS frontier,
and sharded DuckDB storage for maximum throughput.

Results are stored in $HOME/data/crawler/<domain>/results/

Examples:
  search crawl-domain kenh14.vn --continuous
  search crawl-domain dantri.com.vn --max-pages 100000 --workers 200
  search crawl-domain dantri.com.vn --store-body --max-depth 3
  search crawl-domain dantri.com.vn --resume`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := dcrawler.DefaultConfig()
			cfg.Domain = args[0]
			cfg.Workers = workers
			cfg.MaxConns = maxConns
			cfg.MaxDepth = maxDepth
			cfg.MaxPages = maxPages
			cfg.Timeout = time.Duration(timeout) * time.Second
			cfg.RateLimit = rateLimit
			cfg.StoreBody = storeBody
			cfg.StoreLinks = !noLinks
			cfg.RespectRobots = !noRobots
			cfg.FollowSitemap = !noSitemap
			cfg.IncludeSubdomain = includeSubdomain
			cfg.Resume = resume
			cfg.ForceHTTP1 = http1
			cfg.Continuous = continuous
			cfg.TransportShards = transportShards
			cfg.SeedFile = seedFile
			if crawlerDataDir != "" {
				cfg.DataDir = crawlerDataDir
			}
			if userAgent != "" {
				cfg.UserAgent = userAgent
			}

			return runCrawlDomain(cmd, cfg)
		},
	}

	cmd.Flags().IntVar(&workers, "workers", 1000, "Concurrent fetch workers")
	cmd.Flags().IntVar(&maxConns, "max-conns", 200, "Max TCP connections to domain")
	cmd.Flags().IntVar(&maxDepth, "max-depth", 0, "Max BFS depth (0=unlimited)")
	cmd.Flags().IntVar(&maxPages, "max-pages", 0, "Max pages to crawl (0=unlimited)")
	cmd.Flags().IntVar(&timeout, "timeout", 10, "Per-request timeout in seconds")
	cmd.Flags().IntVar(&rateLimit, "rate-limit", 0, "Max requests/sec (0=unlimited)")
	cmd.Flags().BoolVar(&storeBody, "store-body", false, "Store compressed HTML body")
	cmd.Flags().BoolVar(&noLinks, "no-links", false, "Don't store extracted links")
	cmd.Flags().BoolVar(&noRobots, "no-robots", false, "Don't obey robots.txt")
	cmd.Flags().BoolVar(&noSitemap, "no-sitemap", false, "Don't parse sitemap.xml")
	cmd.Flags().BoolVar(&includeSubdomain, "include-subdomain", false, "Also crawl subdomains")
	cmd.Flags().BoolVar(&resume, "resume", false, "Skip already-crawled URLs")
	cmd.Flags().BoolVar(&continuous, "continuous", false, "Run non-stop, re-seed from sitemap when frontier drains (Ctrl+C to stop)")
	cmd.Flags().BoolVar(&http1, "http1", false, "Force HTTP/1.1 (disable HTTP/2)")
	cmd.Flags().IntVar(&transportShards, "transport-shards", 16, "Number of HTTP transport shards")
	cmd.Flags().StringVar(&seedFile, "seed-file", "", "File with seed URLs (one per line)")
	cmd.Flags().StringVar(&crawlerDataDir, "crawler-data", "", "Crawler data directory (default $HOME/data/crawler/)")
	cmd.Flags().StringVar(&userAgent, "user-agent", "", "User-Agent header")

	return cmd
}

func runCrawlDomain(cmd *cobra.Command, cfg dcrawler.Config) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Domain Crawler"))
	fmt.Println()

	c, err := dcrawler.New(cfg)
	if err != nil {
		return err
	}

	h2 := "enabled"
	if cfg.ForceHTTP1 {
		h2 = "disabled"
	}

	fmt.Println(infoStyle.Render(fmt.Sprintf("  Target:   %s", cfg.Domain)))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Workers:  %d  |  Max Conns: %d  |  Shards: %d  |  HTTP/2: %s",
		cfg.Workers, cfg.MaxConns, cfg.TransportShards, h2)))

	maxDepthStr := "unlimited"
	if cfg.MaxDepth > 0 {
		maxDepthStr = fmt.Sprintf("%d", cfg.MaxDepth)
	}
	maxPagesStr := "unlimited"
	if cfg.MaxPages > 0 {
		maxPagesStr = fmt.Sprintf("%d", cfg.MaxPages)
	}
	modeStr := ""
	if cfg.Continuous {
		modeStr = "  |  Mode: continuous"
	}
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Depth:    %s  |  Max Pages: %s%s",
		maxDepthStr, maxPagesStr, modeStr)))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Data:     %s", c.DataDir())))
	fmt.Println()

	err = dcrawler.RunWithDisplay(cmd.Context(), c)

	fmt.Println()
	if err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("  Crawl failed: %v", err)))
		return err
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  Crawl complete in %s  |  %d pages",
		c.Stats().Elapsed().Truncate(time.Second), c.Stats().Done())))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Results:  %s", c.ResultDB().Dir())))

	return nil
}
