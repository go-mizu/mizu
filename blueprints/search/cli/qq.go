package cli

import (
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler/qq"
	"github.com/spf13/cobra"
)

// NewQQ creates the qq CLI command for crawling news.qq.com.
func NewQQ() *cobra.Command {
	var (
		workers   int
		timeout   int
		rateLimit float64
		maxRetry  int
		channels  bool
		probe     bool
		resume    bool
		dataDir   string
	)

	cmd := &cobra.Command{
		Use:   "qq",
		Short: "Crawl news.qq.com (Tencent News)",
		Long: `Crawl news.qq.com using sitemaps and channel feed APIs.

Discovers articles via sitemap index (1000+ sitemaps), then fetches full
article content by parsing server-rendered window.DATA from each article page.

Articles that 302-redirect to babygohome are marked as deleted (fast, no body read).
Rate limiting prevents HTTP 567 anti-bot blocks.

Results are stored in $HOME/data/qq-news/qq.duckdb

Examples:
  search qq                          # Crawl all articles from sitemaps
  search qq --probe                  # Also probe for sitemaps not in index.xml
  search qq --channels               # Also discover from channel feed APIs
  search qq --resume                 # Resume previous crawl (skip already-fetched)
  search qq --rate 20 --workers 20   # Faster crawl
  search qq info                     # Show crawl statistics`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := qq.DefaultConfig()
			if dataDir != "" {
				cfg.DataDir = dataDir
			}
			cfg.Workers = workers
			cfg.Timeout = time.Duration(timeout) * time.Second
			cfg.RateLimit = rateLimit
			cfg.MaxRetry = maxRetry
			cfg.Channels = channels
			cfg.Probe = probe
			cfg.Resume = resume

			c, err := qq.New(cfg)
			if err != nil {
				return err
			}

			return c.Run(cmd.Context())
		},
	}

	cmd.Flags().IntVar(&workers, "workers", 20, "Concurrent fetch workers")
	cmd.Flags().IntVar(&timeout, "timeout", 15, "Request timeout in seconds")
	cmd.Flags().Float64Var(&rateLimit, "rate", 10, "Max requests per second (0 = unlimited)")
	cmd.Flags().IntVar(&maxRetry, "max-retry", 2, "Max retries for rate-limited requests (567)")
	cmd.Flags().BoolVar(&channels, "channels", false, "Also crawl channel feed APIs for discovery")
	cmd.Flags().BoolVar(&probe, "probe", false, "Enumerate ALL possible sitemaps beyond index.xml (slow)")
	cmd.Flags().BoolVar(&resume, "resume", false, "Resume from previous crawl (skip already-fetched)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "Override data directory")

	// Subcommands
	cmd.AddCommand(newQQInfo())

	return cmd
}

func newQQInfo() *cobra.Command {
	var dataDir string

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show QQ News crawl statistics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := qq.DefaultConfig()
			if dataDir != "" {
				cfg.DataDir = dataDir
			}

			db, err := qq.OpenDB(cfg.DBPath())
			if err != nil {
				return fmt.Errorf("open db: %w", err)
			}
			defer db.Close()

			stats, err := db.GetStats()
			if err != nil {
				return err
			}

			fmt.Println("── QQ News Crawl Statistics ──")
			fmt.Printf("  Articles:          %d\n", stats.Articles)
			fmt.Printf("  With content:      %d\n", stats.WithContent)
			fmt.Printf("  Deleted:           %d\n", stats.Deleted)
			fmt.Printf("  With errors:       %d\n", stats.WithError)
			fmt.Printf("  Sitemaps tracked:  %d\n", stats.Sitemaps)
			fmt.Printf("  DB size:           %.1f MB\n", float64(stats.DBSize)/(1024*1024))

			if len(stats.Channels) > 0 {
				fmt.Println("  Channels:")
				for ch, cnt := range stats.Channels {
					fmt.Printf("    %-20s %d\n", ch, cnt)
				}
			}

			fmt.Printf("  DB path:           %s\n", db.Path())

			// Show recent articles
			articles, err := db.TopArticles(5)
			if err == nil && len(articles) > 0 {
				fmt.Println("\n  Recent articles:")
				for _, a := range articles {
					pubStr := ""
					if !a.PublishTime.IsZero() {
						pubStr = a.PublishTime.Format("2006-01-02 15:04")
					}
					fmt.Printf("    [%s] %s (%s) — %s\n", a.Channel, a.Title, pubStr, a.Source)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&dataDir, "data-dir", "", "Override data directory")
	return cmd
}
