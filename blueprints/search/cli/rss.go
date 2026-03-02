package cli

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/crawler"
	"github.com/go-mizu/mizu/blueprints/search/pkg/rss"
	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/store/sqlite"
	"github.com/go-mizu/mizu/blueprints/search/types"
	"github.com/spf13/cobra"
)

// NewRSS returns a new cobra command for rss.
func NewRSS() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rss",
		Short: "Manage RSS feeds",
	}

	cmd.AddCommand(NewRSSSeed())
	cmd.AddCommand(NewRSSList())
	cmd.AddCommand(NewRSSCrawl())
	cmd.AddCommand(NewRSSItems())
	cmd.AddCommand(NewRSSRecrawl())

	return cmd
}

// NewRSSRecrawl returns a new cobra command for recrawling rss feeds.
func NewRSSRecrawl() *cobra.Command {
	var feedID int64
	var workers int
	cmd := &cobra.Command{
		Use:   "recrawl",
		Short: "Crawl and index all items in a feed",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := sqlite.New(databasePath)
			if err != nil {
				return err
			}
			defer db.Close()

			var feeds []*types.RSSFeed
			if feedID != 0 {
				feed, err := db.RSS().GetFeed(context.Background(), feedID)
				if err != nil {
					return err
				}
				feeds = append(feeds, feed)
			} else {
				feeds, err = db.RSS().ListFeeds(context.Background())
				if err != nil {
					return err
				}
			}

			cfg := crawler.Config{
				Workers:       workers,
				MaxDepth:      0, // Only crawl the item itself
				MaxPages:      1000000,
				Delay:         500 * time.Millisecond,
				UserAgent:     "MizuRSSCrawler/1.0",
				Timeout:       30 * time.Second,
				RespectRobots: true,
				BatchSize:     10,
			}

			c, err := crawler.New(cfg)
			if err != nil {
				return err
			}

			var batch []*store.Document
			c.OnResult(func(r crawler.CrawlResult) {
				doc := resultToDocument(r)
				batch = append(batch, doc)
				fmt.Printf("  Indexed: %s\n", r.URL)

				if len(batch) >= cfg.BatchSize {
					if err := db.Index().BulkIndex(context.Background(), batch); err != nil {
						log.Printf("failed to index batch: %v", err)
					}
					batch = batch[:0]
				}
			})

			for _, feed := range feeds {
				fmt.Printf("Recrawling feed: %s\n", feed.Title)
				items, err := db.RSS().ListItems(context.Background(), feed.ID)
				if err != nil {
					log.Printf("failed to list items for feed %d: %v", feed.ID, err)
					continue
				}

				var urls []string
				for _, item := range items {
					urls = append(urls, item.URL)
				}

				if len(urls) > 0 {
					_, _ = c.CrawlURLs(context.Background(), urls)
				}
			}

			if len(batch) > 0 {
				_ = db.Index().BulkIndex(context.Background(), batch)
			}

			return nil
		},
	}

	cmd.Flags().Int64Var(&feedID, "feed-id", 0, "Recrawl a specific feed by ID")
	cmd.Flags().IntVar(&workers, "workers", 4, "Number of concurrent workers")

	return cmd
}

// NewRSSItems returns a new cobra command for listing rss items.
func NewRSSItems() *cobra.Command {
	var feedID int64
	cmd := &cobra.Command{
		Use:   "items",
		Short: "List items for a specific feed",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := sqlite.New(databasePath)
			if err != nil {
				return err
			}
			defer db.Close()

			if feedID == 0 {
				return fmt.Errorf("feed-id is required")
			}

			items, err := db.RSS().ListItems(context.Background(), feedID)
			if err != nil {
				return err
			}

			for _, item := range items {
				fmt.Printf("[%d] %s - %s\n", item.ID, item.Title, item.URL)
			}

			return nil
		},
	}

	cmd.Flags().Int64Var(&feedID, "feed-id", 0, "List items for a specific feed by ID")
	cmd.MarkFlagRequired("feed-id")

	return cmd
}

// NewRSSSeed returns a new cobra command for seeding rss feeds.
func NewRSSSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed RSS feeds from Kagi Small Web",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := sqlite.New(databasePath)
			if err != nil {
				return err
			}
			defer db.Close()
			if err := db.Ensure(context.Background()); err != nil {
				return nil
			}

			sources := []string{
				"https://kagi.com/api/v1/smallweb/feed/",
				"https://kagi.com/smallweb/opml",
			}

			for _, source := range sources {
				parsed, err := rss.ParseURL(source)
				if err != nil {
					log.Printf("failed to parse %s: %v", source, err)
					continue
				}

				switch p := parsed.(type) {
				case *rss.OPML:
					for _, outline := range p.Outlines {
						if outline.XMLURL != "" {
							fmt.Printf("Adding feed from opml: %s\n", outline.XMLURL)
							_, err := db.RSS().AddFeed(context.Background(), &types.RSSFeed{
								URL:   outline.XMLURL,
								Title: outline.Text,
							})
							if err != nil {
								log.Printf("failed to add feed %s: %v", outline.XMLURL, err)
							}
						}
					}
				case *rss.Feed:
					fmt.Printf("Adding feed from atom: %s\n", p.FeedLink)
					_, err := db.RSS().AddFeed(context.Background(), &types.RSSFeed{
						URL:         p.FeedLink,
						Title:       p.Title,
						Description: p.Description,
					})
					if err != nil {
						log.Printf("failed to add feed %s: %v", p.FeedLink, err)
					}
				}
			}

			return nil
		},
	}

	return cmd
}

// NewRSSList returns a new cobra command for listing rss feeds.
func NewRSSList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all RSS feeds",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := sqlite.New(databasePath)
			if err != nil {
				return err
			}
			defer db.Close()

			feeds, err := db.RSS().ListFeeds(context.Background())
			if err != nil {
				return err
			}

			for _, feed := range feeds {
				fmt.Printf("[%d] %s - %s\n", feed.ID, feed.Title, feed.URL)
			}

			return nil
		},
	}
	return cmd
}

// NewRSSCrawl returns a new cobra command for crawling rss feeds.
func NewRSSCrawl() *cobra.Command {
	var feedID int64
	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Crawl a specific feed or all feeds",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := sqlite.New(databasePath)
			if err != nil {
				return err
			}
			defer db.Close()

			var feeds []*types.RSSFeed
			if feedID != 0 {
				feed, err := db.RSS().GetFeed(context.Background(), feedID)
				if err != nil {
					return err
				}
				feeds = append(feeds, feed)
			} else {
				feeds, err = db.RSS().ListFeeds(context.Background())
				if err != nil {
					return err
				}
			}

			for _, feed := range feeds {
				fmt.Printf("Crawling feed: %s\n", feed.Title)
				parsed, err := rss.ParseURL(feed.URL)
				if err != nil {
					log.Printf("failed to parse feed %s: %v", feed.URL, err)
					continue
				}

				if p, ok := parsed.(*rss.Feed); ok {
					for _, item := range p.Items {
						_, err := db.RSS().AddItem(context.Background(), &types.RSSItem{
							FeedID:      feed.ID,
							Title:       item.Title,
							URL:         item.Link,
							Content:     item.Content,
							PublishedAt: item.Published,
						})
						if err != nil {
							// todo: ignore unique constraint errors
							log.Printf("failed to add item %s: %v", item.Link, err)
						}
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().Int64Var(&feedID, "feed-id", 0, "Crawl a specific feed by ID")

	return cmd
}
