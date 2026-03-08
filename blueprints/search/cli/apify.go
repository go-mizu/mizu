package cli

import (
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler/apify"
	"github.com/spf13/cobra"
)

// NewApify creates the apify command.
func NewApify() *cobra.Command {
	var (
		workers        int
		qps            float64
		timeout        int
		retries        int
		hitsPerPage    int
		maxDetails     int
		dataDir        string
		dbPath         string
		refreshDetails bool
		indexOnly      bool
		detailOnly     bool
		enrichVersions bool
		enrichBuild    bool
	)

	cmd := &cobra.Command{
		Use:   "apify",
		Short: "Crawl Apify Store actors into DuckDB",
		Long: `Download Apify Store actor index and actor details into DuckDB.

Source pages:
  https://apify.com/store/categories

Output:
  $HOME/data/apify/apify.duckdb

Examples:
  search apify
  search apify --workers 32 --qps 50
  search apify --index-only
  search apify --detail-only --max-details 500
  search apify info`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := apify.DefaultConfig()
			if dataDir != "" {
				cfg.DataDir = dataDir
			}
			if dbPath != "" {
				cfg.DBPath = dbPath
			}
			cfg.Workers = workers
			cfg.QPS = qps
			cfg.Timeout = time.Duration(timeout) * time.Second
			cfg.MaxRetries = retries
			cfg.HitsPerPage = hitsPerPage
			cfg.MaxDetails = maxDetails
			cfg.RefreshDetails = refreshDetails
			cfg.IndexOnly = indexOnly
			cfg.DetailOnly = detailOnly
			cfg.EnrichVersions = enrichVersions
			cfg.EnrichLatestBuild = enrichBuild

			crawler, err := apify.New(cfg)
			if err != nil {
				return err
			}
			defer crawler.Close()

			fmt.Println(Banner())
			fmt.Println(subtitleStyle.Render("Apify Store Crawler"))
			fmt.Println()
			fmt.Println(infoStyle.Render(fmt.Sprintf("  Store:    %s", cfg.StoreURL)))
			fmt.Println(infoStyle.Render(fmt.Sprintf("  DB:       %s", cfg.DBPath)))
			fmt.Println(infoStyle.Render(fmt.Sprintf("  Workers:  %d", cfg.Workers)))
			if cfg.QPS > 0 {
				fmt.Println(infoStyle.Render(fmt.Sprintf("  QPS:      %.2f", cfg.QPS)))
			} else {
				fmt.Println(infoStyle.Render("  QPS:      unlimited"))
			}
			fmt.Println()

			start := time.Now()
			err = crawler.Run(cmd.Context())
			stats := crawler.Stats()
			elapsed := time.Since(start).Truncate(time.Second)

			fmt.Println()
			if err != nil {
				fmt.Println(errorStyle.Render(fmt.Sprintf("  Crawl finished with errors: %v", err)))
			} else {
				fmt.Println(successStyle.Render("  Crawl completed"))
			}
			fmt.Println(infoStyle.Render(fmt.Sprintf("  Duration:        %s", elapsed)))
			fmt.Println(infoStyle.Render(fmt.Sprintf("  Expected total:  %d", stats.ExpectedTotal)))
			fmt.Println(infoStyle.Render(fmt.Sprintf("  Indexed total:   %d", stats.IndexedTotal)))
			fmt.Println(infoStyle.Render(fmt.Sprintf("  Detail queued:   %d", stats.DetailQueued)))
			fmt.Println(infoStyle.Render(fmt.Sprintf("  Detail success:  %d", stats.DetailSuccess)))
			fmt.Println(infoStyle.Render(fmt.Sprintf("  Detail failed:   %d", stats.DetailFailed)))

			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&workers, "workers", 16, "Concurrent workers for index/detail requests")
	cmd.Flags().Float64Var(&qps, "qps", 25, "Max detail requests per second (0 = unlimited)")
	cmd.Flags().IntVar(&timeout, "timeout", 30, "Request timeout in seconds")
	cmd.Flags().IntVar(&retries, "retries", 3, "Max retries per request")
	cmd.Flags().IntVar(&hitsPerPage, "hits-per-page", 1000, "Algolia hits per page (max 1000)")
	cmd.Flags().IntVar(&maxDetails, "max-details", 0, "Limit number of actor details to fetch (0 = all)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "Override data directory (default $HOME/data/apify)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Override DuckDB path (default $HOME/data/apify/apify.duckdb)")
	cmd.Flags().BoolVar(&refreshDetails, "refresh-details", false, "Re-fetch details even if already present")
	cmd.Flags().BoolVar(&indexOnly, "index-only", false, "Only fetch index pages (skip detail fetch)")
	cmd.Flags().BoolVar(&detailOnly, "detail-only", false, "Only fetch details from existing index table")
	cmd.Flags().BoolVar(&enrichVersions, "enrich-versions", true, "Fetch full /versions pages for each actor")
	cmd.Flags().BoolVar(&enrichBuild, "enrich-build", true, "Fetch latest actor build metadata via /actor-builds/{buildId}")

	cmd.AddCommand(newApifyInfo())
	return cmd
}

func newApifyInfo() *cobra.Command {
	var dbPath string
	var dataDir string
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show Apify crawl stats from DuckDB",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := apify.DefaultConfig()
			if dataDir != "" {
				cfg.DataDir = dataDir
			}
			if dbPath != "" {
				cfg.DBPath = dbPath
			}
			db, err := apify.OpenDB(cfg.DBPath)
			if err != nil {
				return err
			}
			defer db.Close()

			indexed, detailed, failed, err := db.Counts()
			if err != nil {
				return err
			}

			fmt.Println("── Apify Crawl Statistics ──")
			fmt.Printf("  Indexed actors:      %d\n", indexed)
			fmt.Printf("  Detail success:      %d\n", detailed)
			fmt.Printf("  Detail failed:       %d\n", failed)
			fmt.Printf("  DB path:             %s\n", cfg.DBPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "Override data directory")
	cmd.Flags().StringVar(&dbPath, "db", "", "Override DuckDB path")
	return cmd
}
