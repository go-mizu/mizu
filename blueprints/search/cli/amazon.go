package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape/amazon"
	"github.com/spf13/cobra"
)

func NewAmazon() *cobra.Command {
	var (
		workers   int
		timeout   int
		rateLimit float64
		maxPages  int
		dataDir   string
		market    string
		resume    bool
		sortBy    string
	)

	cmd := &cobra.Command{
		Use:   "amazon <query>",
		Short: "Search and crawl Amazon product results",
		Long: `Crawl Amazon search result pages and persist normalized product data.

Discovery strategy:
  1) Generate result-page URLs from /s?k=<query>&page=<n>
  2) Crawl pages concurrently with throttling
  3) Stop at max pages or when Amazon reports no next page
  4) Persist product cards (ASIN, title, URL, price, rating, reviews, badges)

Examples:
  search amazon mechanical keyboard
  search amazon "wireless earbuds" --pages 10 --workers 8 --rate 2
  search amazon "desk lamp" --market www.amazon.co.uk --sort review-rank
  search amazon discover "usb c hub"
  search amazon info`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := amazon.DefaultConfig()
			cfg.Workers = workers
			cfg.Timeout = time.Duration(timeout) * time.Second
			cfg.RateLimit = rateLimit
			cfg.MaxPages = maxPages
			cfg.Market = market
			cfg.Resume = resume
			cfg.SortBy = sortBy
			if dataDir != "" {
				cfg.DataDir = dataDir
			}

			query := strings.Join(args, " ")
			crawler, err := amazon.New(cfg)
			if err != nil {
				return err
			}
			stats, err := crawler.Crawl(cmd.Context(), query)
			if err != nil {
				return err
			}

			fmt.Println("── Amazon Crawl Complete ──")
			fmt.Printf("  Query:        %s\n", stats.Query)
			fmt.Printf("  Pages:        %d\n", stats.Pages)
			fmt.Printf("  Products:     %d\n", stats.Products)
			fmt.Printf("  Unique ASINs: %d\n", stats.UniqueASIN)
			fmt.Printf("  DB:           %s\n", cfg.DBPath())
			return nil
		},
	}

	cmd.Flags().IntVar(&workers, "workers", 4, "Concurrent page workers")
	cmd.Flags().IntVar(&timeout, "timeout", 20, "Request timeout (seconds)")
	cmd.Flags().Float64Var(&rateLimit, "rate", 1.5, "Requests per second (0 = unlimited)")
	cmd.Flags().IntVar(&maxPages, "pages", 5, "Max result pages to crawl")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "Override data directory")
	cmd.Flags().StringVar(&market, "market", "www.amazon.com", "Amazon market host (e.g., www.amazon.co.uk)")
	cmd.Flags().BoolVar(&resume, "resume", false, "Resume from next uncrawled page for this query")
	cmd.Flags().StringVar(&sortBy, "sort", "", "Sort key for Amazon search (e.g., review-rank)")

	cmd.AddCommand(newAmazonDiscover(&workers, &timeout, &rateLimit, &maxPages, &dataDir, &market, &resume, &sortBy))
	cmd.AddCommand(newAmazonInfo())

	return cmd
}

func newAmazonDiscover(workers, timeout *int, rate *float64, pages *int, dataDir, market *string, resume *bool, sortBy *string) *cobra.Command {
	return &cobra.Command{
		Use:   "discover <query>",
		Short: "Print the Amazon result page URLs that will be crawled",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := amazon.DefaultConfig()
			cfg.Workers = *workers
			cfg.Timeout = time.Duration(*timeout) * time.Second
			cfg.RateLimit = *rate
			cfg.MaxPages = *pages
			cfg.Market = *market
			cfg.Resume = *resume
			cfg.SortBy = *sortBy
			if *dataDir != "" {
				cfg.DataDir = *dataDir
			}

			crawler, err := amazon.New(cfg)
			if err != nil {
				return err
			}
			defer crawler.Close()

			query := strings.Join(args, " ")
			pages, err := crawler.DiscoverPages(cmd.Context(), query)
			if err != nil {
				return err
			}
			for i, p := range pages {
				fmt.Printf("%2d. %s\n", i+1, p)
			}
			return nil
		},
	}
}

func newAmazonInfo() *cobra.Command {
	var dataDir string
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show Amazon scrape database statistics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := amazon.DefaultConfig()
			if dataDir != "" {
				cfg.DataDir = dataDir
			}
			db, err := amazon.OpenDB(cfg.DBPath())
			if err != nil {
				return err
			}
			defer db.Close()

			stats, err := db.Stats()
			if err != nil {
				return err
			}
			fmt.Println("── Amazon Search Crawl Statistics ──")
			fmt.Printf("  Last query:     %s\n", stats.Query)
			fmt.Printf("  Max page:       %d\n", stats.Pages)
			fmt.Printf("  Products:       %d\n", stats.Products)
			fmt.Printf("  Unique ASINs:   %d\n", stats.UniqueASIN)
			fmt.Printf("  DB path:        %s\n", cfg.DBPath())
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "Override data directory")
	return cmd
}
