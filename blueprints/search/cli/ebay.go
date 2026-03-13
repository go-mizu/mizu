package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape/ebay"
	"github.com/spf13/cobra"
)

// NewEbay creates the eBay CLI command.
func NewEbay() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ebay",
		Short: "Scrape eBay item pages and search results",
		Long: `Scrape public eBay item pages and search-result pages into a local DuckDB database.

The primary durable entity is the item page. Search pages are used to discover
new items and expand the crawl queue.

Examples:
  search ebay item 276703509680
  search ebay search "iphone"
  search ebay seed --file queries.txt --type search
  search ebay crawl --workers 2
  search ebay info`,
	}

	cmd.AddCommand(newEbayItem())
	cmd.AddCommand(newEbaySearch())
	cmd.AddCommand(newEbaySeed())
	cmd.AddCommand(newEbayCrawl())
	cmd.AddCommand(newEbayInfo())
	cmd.AddCommand(newEbayJobs())
	cmd.AddCommand(newEbayQueue())
	return cmd
}

func addEbayFlags(cmd *cobra.Command, dbPath, statePath *string, delay, maxPages *int) {
	cfg := ebay.DefaultConfig()
	cmd.Flags().StringVar(dbPath, "db", cfg.DBPath, "Path to ebay.duckdb")
	cmd.Flags().StringVar(statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
	cmd.Flags().IntVar(maxPages, "max-pages", cfg.MaxPages, "Max search pages to fetch per task")
}

func openEbayDBs(dbPath, statePath string, delay, maxPages int) (*ebay.DB, *ebay.State, *ebay.Client, ebay.Config, error) {
	cfg := ebay.DefaultConfig()
	cfg.DBPath = dbPath
	cfg.StatePath = statePath
	cfg.Delay = time.Duration(delay) * time.Millisecond
	cfg.MaxPages = maxPages

	db, err := ebay.OpenDB(dbPath)
	if err != nil {
		return nil, nil, nil, cfg, fmt.Errorf("open db: %w", err)
	}
	stateDB, err := ebay.OpenState(statePath)
	if err != nil {
		db.Close()
		return nil, nil, nil, cfg, fmt.Errorf("open state: %w", err)
	}
	client := ebay.NewClient(cfg)
	return db, stateDB, client, cfg, nil
}

func newEbayItem() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int

	cmd := &cobra.Command{
		Use:   "item <id|url>",
		Short: "Fetch a single eBay item page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openEbayDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			itemURL := ebay.NormalizeItemURL(args[0])
			fmt.Printf("Fetching %s ...\n", itemURL)

			task := &ebay.ItemTask{
				URL:     itemURL,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *ebay.ItemState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addEbayFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

func newEbaySearch() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Fetch eBay search results and enqueue discovered items",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, cfg, err := openEbayDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			query := strings.TrimSpace(args[0])
			fmt.Printf("Searching eBay for %q ...\n", query)

			task := &ebay.SearchTask{
				Query:   query,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
				Config:  cfg,
			}
			m, err := task.Run(cmd.Context(), func(s *ebay.SearchState) {
				fmt.Printf("  [%s] page=%d items=%d\n", s.Status, s.Page, s.ItemsFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: pages=%d items=%d failed=%d\n", m.Pages, m.Fetched, m.Failed)
			return nil
		},
	}

	addEbayFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

func newEbaySeed() *cobra.Command {
	var dbPath, statePath, filePath, entityType string
	var delay, maxPages int

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the crawl queue from a file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(filePath) == "" {
				return fmt.Errorf("--file is required")
			}
			if entityType != ebay.EntityItem && entityType != ebay.EntitySearch {
				return fmt.Errorf("--type must be %q or %q", ebay.EntityItem, ebay.EntitySearch)
			}

			_, stateDB, _, _, err := openEbayDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			f, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer f.Close()

			var items []ebay.QueueItem
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				switch entityType {
				case ebay.EntityItem:
					items = append(items, ebay.QueueItem{
						URL:        ebay.NormalizeItemURL(line),
						EntityType: ebay.EntityItem,
						Priority:   10,
					})
				case ebay.EntitySearch:
					searchURL := line
					if !strings.HasPrefix(searchURL, "http://") && !strings.HasPrefix(searchURL, "https://") {
						searchURL = ebay.SearchURL(line, 1)
					}
					items = append(items, ebay.QueueItem{
						URL:        searchURL,
						EntityType: ebay.EntitySearch,
						Priority:   20,
					})
				}
			}
			if err := scanner.Err(); err != nil {
				return err
			}
			if err := stateDB.EnqueueBatch(items); err != nil {
				return err
			}
			fmt.Printf("Enqueued %d %s entries\n", len(items), entityType)
			return nil
		},
	}

	addEbayFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	cmd.Flags().StringVar(&filePath, "file", "", "Path to input file")
	cmd.Flags().StringVar(&entityType, "type", ebay.EntitySearch, "Queue type: search or item")
	return cmd
}

func newEbayCrawl() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages, workers int

	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Run the eBay frontier crawler",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			db, stateDB, client, cfg, err := openEbayDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			cfg.Workers = workers
			pending, err := stateDB.PendingCount()
			if err != nil {
				return err
			}
			if pending == 0 {
				fmt.Println("Queue is empty. Run 'search ebay seed' or 'search ebay search' first.")
				return nil
			}

			jobID := time.Now().UTC().Format("20060102T150405Z")
			_ = stateDB.CreateJob(jobID, "ebay crawl", "crawl")

			task := &ebay.CrawlTask{
				Config:  cfg,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *ebay.CrawlState) {
				ebay.PrintCrawlProgress(s)
			})
			fmt.Println()
			if err != nil {
				_ = stateDB.UpdateJob(jobID, "failed", err.Error())
				return err
			}

			stats := fmt.Sprintf("done=%d failed=%d duration=%s", m.Done, m.Failed, m.Duration.Round(time.Second))
			_ = stateDB.UpdateJob(jobID, "completed", stats)
			fmt.Printf("Done: processed=%d failed=%d duration=%s\n", m.Done, m.Failed, m.Duration.Round(time.Second))
			return nil
		},
	}

	addEbayFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	cmd.Flags().IntVar(&workers, "workers", ebay.DefaultWorkers, "Number of concurrent crawl workers")
	return cmd
}

func newEbayInfo() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show eBay DB and queue stats",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			db, stateDB, _, _, err := openEbayDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			return ebay.PrintStats(db, stateDB)
		},
	}

	addEbayFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

func newEbayJobs() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages, limit int

	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "List recent eBay crawl jobs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, stateDB, _, _, err := openEbayDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			jobs, err := stateDB.ListJobs(limit)
			if err != nil {
				return err
			}
			if len(jobs) == 0 {
				fmt.Println("No jobs found.")
				return nil
			}

			for _, job := range jobs {
				fmt.Printf("%s  %-10s  %-8s  %s\n",
					job.StartedAt.Format(time.RFC3339), job.Status, job.Type, job.Name)
			}
			return nil
		},
	}

	addEbayFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum jobs to show")
	return cmd
}

func newEbayQueue() *cobra.Command {
	var dbPath, statePath, status string
	var delay, maxPages, limit int

	cmd := &cobra.Command{
		Use:   "queue",
		Short: "List queued eBay crawl items",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, stateDB, _, _, err := openEbayDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			items, err := stateDB.ListQueue(status, limit)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Printf("No queue items with status %q.\n", status)
				return nil
			}

			for _, item := range items {
				fmt.Printf("[%d] %-6s p=%d  %s\n", item.ID, item.EntityType, item.Priority, item.URL)
			}
			return nil
		},
	}

	addEbayFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	cmd.Flags().StringVar(&status, "status", "pending", "Queue status: pending, in_progress, done, failed")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum queue items to show")
	return cmd
}
