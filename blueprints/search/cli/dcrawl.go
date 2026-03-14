package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape"
	"github.com/spf13/cobra"
)

// NewScrape creates the scrape CLI command (formerly crawl-domain).
func NewScrape() *cobra.Command {
	var (
		workers          int
		maxConns         int
		maxDepth         int
		maxPages         int
		timeout          int
		rateLimit        int
		transportShards  int
		noBody           bool
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
		useRod           bool
		useLightpanda    bool
		rodWorkers       int
		scrollCount      int
		extractImages    bool
		downloadImages   bool
		staleHours       int
		domainAliases    []string
		noRenderWait     bool
		proxyURL         string
		proxyFile        string
		useWorker        bool
		workerURL        string
		workerToken      string
		workerBrowser    bool
		useTUI           bool
		useCloudflare       bool
		cfLimit             int
		cfDepth             int
		cfSource            string
		cfRender            bool
		cfSubdomains        bool
		cfInclude           []string
		cfExclude           []string
		cfRejectResources   []string
		cfWaitSelector      string
		cfGotoWaitUntil     string
		cfGotoTimeout       int
		cfUserAgent         string
	)

	cmd := &cobra.Command{
		Use:     "scrape <domain>",
		Aliases: []string{"crawl-domain"},
		Short:   "Crawl all pages from a single domain",
		Long: `High-throughput single-domain web crawler targeting 10K+ pages/second.

Uses HTTP/2 multiplexing, bloom filter URL dedup, BFS frontier,
and sharded DuckDB storage for maximum throughput.

Results are stored in $HOME/data/crawler/<domain>/results/

Examples:
  search scrape kenh14.vn --continuous
  search scrape dantri.com.vn --max-pages 100000 --workers 200
  search scrape dantri.com.vn --max-depth 3
  search scrape dantri.com.vn --resume

Pinterest (auto-detected, uses internal API - no browser needed):
  search scrape 'https://www.pinterest.com/search/pins/?q=gouache' --download-images
  search scrape 'https://www.pinterest.com/search/pins/?q=watercolor' --max-pages 200

Browser mode (JS-rendered pages, bypasses Cloudflare):
  search scrape openai.com --browser
  search scrape openai.com --browser --no-render-wait   # faster for SSG/Next.js sites
  search scrape openai.com --browser --browser-pages 80 # explicit tab count`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if useRod && useLightpanda {
				return fmt.Errorf("--browser and --lightpanda are mutually exclusive")
			}
			if useCloudflare {
				opts := scrape.CFOptions{
					Source:              cfSource,
					IncludeSubdomains:   cfSubdomains,
					IncludePatterns:     cfInclude,
					ExcludePatterns:     cfExclude,
					RejectResourceTypes: cfRejectResources,
					WaitForSelector:     cfWaitSelector,
					GotoWaitUntil:       cfGotoWaitUntil,
					GotoTimeout:         cfGotoTimeout,
					UserAgent:           cfUserAgent,
				}
				if cfRender {
					t := true
					opts.Render = &t
				}
				// default: opts.Render == nil → buildCFRequest sets render=false
				return runCloudflareScrape(cmd, args[0], cfLimit, cfDepth, crawlerDataDir, opts)
			}
			if (proxyURL != "" || proxyFile != "") && !useRod && !useLightpanda {
				return fmt.Errorf("--proxy-url and --proxy-file require --browser (or --lightpanda) mode")
			}
			cfg := scrape.DefaultConfig()
			// If user passed a full URL, use it as seed
			if seedURL := scrape.ExtractSeedURL(args[0]); seedURL != "" {
				cfg.SeedURLs = []string{seedURL}
			}
			cfg.Domain = args[0]
			cfg.Workers = workers
			cfg.MaxConns = maxConns
			cfg.MaxDepth = maxDepth
			cfg.MaxPages = maxPages
			cfg.Timeout = time.Duration(timeout) * time.Second
			cfg.RateLimit = rateLimit
			cfg.StoreBody = !noBody
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
			cfg.UseRod = useRod
			cfg.UseLightpanda = useLightpanda
			cfg.RodWorkers = rodWorkers
			cfg.RodHeadless = true
			cfg.RodBlockResources = useRod // block images/fonts/CSS by default in browser mode (not needed for lightpanda)
			// Browser mode: auto-bump timeout for heavy JS sites.
			if (useRod || useLightpanda) && cfg.Timeout < 30*time.Second {
				cfg.Timeout = 30 * time.Second
			}
			cfg.ScrollCount = scrollCount
			// Browser mode: auto-scroll to discover lazy-loaded content (infinite scroll, AJAX feeds).
			// High count (20) is safe: early termination stops when page stops growing.
			if (useRod || useLightpanda) && !cmd.Flags().Changed("scroll") {
				cfg.ScrollCount = 20
			}
			cfg.ExtractImages = extractImages || downloadImages
			cfg.StaleHours = staleHours
			cfg.DomainAliases = domainAliases
			cfg.RodNoRenderWait = noRenderWait
			cfg.ProxyURL = proxyURL
			cfg.ProxyFile = proxyFile

			// Worker mode
			cfg.UseWorker = useWorker
			cfg.WorkerURL = workerURL
			cfg.WorkerBrowser = workerBrowser
			cfg.WorkerToken = workerToken
			if cfg.WorkerToken == "" {
				cfg.WorkerToken = os.Getenv("MIZU_TOKEN")
			}
			if cfg.UseWorker && cfg.WorkerToken == "" {
				return fmt.Errorf("--worker requires --worker-token or MIZU_TOKEN env var")
			}

			return runCrawlDomain(cmd, cfg, downloadImages, useTUI)
		},
	}

	cmd.Flags().IntVar(&workers, "workers", 1000, "Concurrent fetch workers")
	cmd.Flags().IntVar(&maxConns, "max-conns", 200, "Max TCP connections to domain")
	cmd.Flags().IntVar(&maxDepth, "max-depth", 0, "Max BFS depth (0=unlimited)")
	cmd.Flags().IntVar(&maxPages, "max-pages", 0, "Max pages to crawl (0=unlimited)")
	cmd.Flags().IntVar(&timeout, "timeout", 10, "Per-request timeout in seconds")
	cmd.Flags().IntVar(&rateLimit, "rate-limit", 0, "Max requests/sec (0=unlimited)")
	cmd.Flags().BoolVar(&noBody, "no-body", false, "Don't store compressed HTML body")
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
	cmd.Flags().BoolVar(&useRod, "browser", false, "Use headless Chrome for JS-rendered pages (bypasses Cloudflare)")
	cmd.Flags().BoolVar(&useLightpanda, "lightpanda", false, "Use Lightpanda browser (faster, less RAM, but less stable than Chrome)")
	cmd.Flags().IntVar(&rodWorkers, "browser-pages", 0, "Number of concurrent browser tabs (0=auto from RAM)")
	cmd.Flags().BoolVar(&noRenderWait, "no-render-wait", false, "Skip DOM stabilization wait in browser mode (faster for SSG/Next.js sites like openai.com)")
	cmd.Flags().IntVar(&scrollCount, "scroll", 0, "Scroll N times in browser mode for infinite scroll pages (Pinterest, etc.)")
	cmd.Flags().BoolVar(&extractImages, "extract-images", false, "Extract <img> URLs and store in links table")
	cmd.Flags().BoolVar(&downloadImages, "download-images", false, "Download discovered images after crawl (implies --extract-images)")
	cmd.Flags().IntVar(&staleHours, "stale", 0, "Re-crawl pages older than N hours on --resume (0=disabled)")
	cmd.Flags().StringSliceVar(&domainAliases, "domain-alias", nil, "Additional domains to treat as same-domain (e.g., --domain-alias new.qq.com)")
	cmd.Flags().StringVar(&proxyURL, "proxy-url", "", "HTTP/SOCKS5 proxy for Chrome (e.g. http://user:pass@host:port or socks5://host:port)")
	cmd.Flags().StringVar(&proxyFile, "proxy-file", "", "File with one proxy URL per line (enables one Chrome instance per proxy, round-robin)")

	// Worker mode
	cmd.Flags().BoolVar(&useWorker, "worker", false, "Proxy fetches through CF Worker (returns HTML + markdown)")
	cmd.Flags().StringVar(&workerURL, "worker-url", "", "Worker endpoint (default https://crawler.go-mizu.workers.dev)")
	cmd.Flags().StringVar(&workerToken, "worker-token", "", "Worker auth token (default $MIZU_TOKEN)")
	cmd.Flags().BoolVar(&workerBrowser, "worker-browser", false, "Enable CF Browser Rendering on worker side")
	cmd.Flags().BoolVar(&useTUI, "tui", false, "Use full-screen TUI dashboard (requires terminal)")

	// Cloudflare Browser Rendering REST API mode
	cmd.Flags().BoolVar(&useCloudflare, "cloudflare", false, "Use Cloudflare Browser Rendering /crawl API (credentials from ~/data/cloudflare/cloudflare.json)")
	cmd.Flags().IntVar(&cfLimit, "cf-limit", 0, "Max pages for CF crawl (0=CF default of 10)")
	cmd.Flags().IntVar(&cfDepth, "cf-depth", 0, "Max link depth for CF crawl (0=CF default unlimited)")
	cmd.Flags().StringVar(&cfSource, "cf-source", "", "Link discovery: all (default), sitemaps, links")
	cmd.Flags().BoolVar(&cfRender, "cf-render", false, "Enable JS rendering via CF Browser (default: static HTML fetch, faster)")
	cmd.Flags().BoolVar(&cfSubdomains, "cf-subdomains", false, "Follow links to subdomains")
	cmd.Flags().StringSliceVar(&cfInclude, "cf-include", nil, "Wildcard URL patterns to include (e.g. '*/blog/*')")
	cmd.Flags().StringSliceVar(&cfExclude, "cf-exclude", nil, "Wildcard URL patterns to exclude (e.g. '*/tag/*')")
	cmd.Flags().StringSliceVar(&cfRejectResources, "cf-reject-resources", nil, "Block resource types: image, media, font, stylesheet")
	cmd.Flags().StringVar(&cfWaitSelector, "cf-wait-selector", "", "CSS selector to wait for before extracting content")
	cmd.Flags().StringVar(&cfGotoWaitUntil, "cf-goto-wait", "", "Navigation event: load, domcontentloaded, networkidle0, networkidle2")
	cmd.Flags().IntVar(&cfGotoTimeout, "cf-goto-timeout", 0, "Per-page navigation timeout in ms (0=CF default)")
	cmd.Flags().StringVar(&cfUserAgent, "cf-user-agent", "", "Custom User-Agent for CF crawl requests")

	return cmd
}

func runCrawlDomain(cmd *cobra.Command, cfg scrape.Config, downloadImages, useTUI bool) error {
	c, err := scrape.New(cfg)
	if err != nil {
		return err
	}

	// Pinterest: use internal API instead of browser/HTTP crawl
	if scrape.IsPinterestDomain(cfg.Domain) {
		query := ""
		for _, seed := range cfg.SeedURLs {
			if q := scrape.ExtractPinterestQuery(seed); q != "" {
				query = q
				break
			}
		}
		if query != "" {
			return runPinterestSearch(cmd, c, cfg, query, downloadImages)
		}
	}

	if useTUI {
		err = scrape.RunWithDisplay(cmd.Context(), c)
	} else {
		fmt.Println(subtitleStyle.Render("Scraping " + cfg.Domain))
		fmt.Println(infoStyle.Render(fmt.Sprintf("  Workers: %d  |  Conns: %d  |  Timeout: %s",
			cfg.Workers, cfg.MaxConns, cfg.Timeout)))
		fmt.Println(infoStyle.Render(fmt.Sprintf("  Data:    %s", cfg.DomainDir())))
		fmt.Println()
		err = scrape.RunWithProgress(cmd.Context(), c)
	}

	// After TUI exits (alt screen restored), print final summary
	fmt.Println()
	if err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("  Crawl failed: %v", err)))
		return err
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  Crawl complete in %s  |  %d pages",
		c.Stats().Elapsed().Truncate(time.Second), c.Stats().Done())))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Results:  %s", c.ResultDB().Dir())))

	if downloadImages {
		fmt.Println()
		fmt.Println(subtitleStyle.Render("Downloading Images"))
		fmt.Println()
		if dlErr := scrape.DownloadImages(cmd.Context(), cfg); dlErr != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("  Image download: %v", dlErr)))
		}
	}

	return nil
}

func runCloudflareScrape(cmd *cobra.Command, domain string, limit, depth int, dataDir string, opts scrape.CFOptions) error {
	cfg := scrape.DefaultConfig()
	cfg.Domain = domain
	if dataDir != "" {
		cfg.DataDir = dataDir
	}

	// Build seed URL
	seedURL := domain
	if !strings.HasPrefix(seedURL, "http://") && !strings.HasPrefix(seedURL, "https://") {
		seedURL = "https://" + seedURL
	}

	fmt.Println(subtitleStyle.Render("Scraping " + domain + " via Cloudflare Browser Rendering"))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Data: %s", cfg.DomainDir())))
	fmt.Println()

	return scrape.RunCloudflareCrawl(cmd.Context(), cfg, seedURL, limit, depth, opts)
}

func runPinterestSearch(cmd *cobra.Command, c *scrape.Crawler, cfg scrape.Config, query string, downloadImages bool) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Domain Crawler"))
	fmt.Println()
	fmt.Println(infoStyle.Render("  Target:   pinterest.com"))
	fmt.Println(infoStyle.Render("  Mode:     Pinterest API (no browser)"))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Data:     %s", c.DataDir())))
	fmt.Println()

	start := time.Now()
	if err := scrape.RunPinterestSearch(cmd.Context(), c, query); err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render(fmt.Sprintf("  Pinterest search failed: %v", err)))
		return err
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Done in %s", time.Since(start).Truncate(time.Second))))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Results:  %s", cfg.ResultDir())))

	if downloadImages {
		fmt.Println()
		fmt.Println(subtitleStyle.Render("Downloading Images"))
		fmt.Println()
		if dlErr := scrape.DownloadImages(cmd.Context(), cfg); dlErr != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("  Image download: %v", dlErr)))
		}
	}

	return nil
}
