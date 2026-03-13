package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape/facebook"
	"github.com/spf13/cobra"
)

func NewFacebook() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "facebook",
		Short: "Scrape public Facebook pages, profiles, groups, posts, and search results",
		Long: `Scrape public Facebook data into a local DuckDB database.

The implementation targets the mbasic/mobile HTML surface and supports optional cookies.
Data is stored in $HOME/data/facebook/facebook.duckdb by default.`,
	}

	cmd.AddCommand(newFacebookPost())
	cmd.AddCommand(newFacebookPage())
	cmd.AddCommand(newFacebookProfile())
	cmd.AddCommand(newFacebookGroup())
	cmd.AddCommand(newFacebookSearch())
	cmd.AddCommand(newFacebookSeed())
	cmd.AddCommand(newFacebookCrawl())
	cmd.AddCommand(newFacebookInfo())
	cmd.AddCommand(newFacebookJobs())
	cmd.AddCommand(newFacebookQueue())
	return cmd
}

func addFacebookFlags(cmd *cobra.Command, dbPath, statePath *string, delay *int, cookie, cookieFile *string) {
	cfg := facebook.DefaultConfig()
	cmd.Flags().StringVar(dbPath, "db", cfg.DBPath, "Path to facebook.duckdb")
	cmd.Flags().StringVar(statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
	cmd.Flags().StringVar(cookie, "cookie", "", "Raw Cookie header value for Facebook requests")
	cmd.Flags().StringVar(cookieFile, "cookie-file", "", "Path to a file containing the Cookie header value")
}

func openFacebookDBs(dbPath, statePath string, delay int, cookie, cookieFile string) (*facebook.DB, *facebook.State, *facebook.Client, error) {
	cfg := facebook.DefaultConfig()
	cfg.DBPath = dbPath
	cfg.StatePath = statePath
	cfg.Delay = time.Duration(delay) * time.Millisecond
	cfg.Cookies = cookie
	cfg.CookiesFile = cookieFile

	db, err := facebook.OpenDB(dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open db: %w", err)
	}
	stateDB, err := facebook.OpenState(statePath)
	if err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("open state: %w", err)
	}
	client, err := facebook.NewClient(cfg)
	if err != nil {
		db.Close()
		stateDB.Close()
		return nil, nil, nil, fmt.Errorf("facebook client: %w", err)
	}
	return db, stateDB, client, nil
}

func newFacebookPost() *cobra.Command {
	var dbPath, statePath, cookie, cookieFile string
	var delay, maxComments int

	cmd := &cobra.Command{
		Use:   "post <url>",
		Short: "Fetch a single Facebook post permalink",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openFacebookDBs(dbPath, statePath, delay, cookie, cookieFile)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			task := &facebook.PostTask{
				URL:         args[0],
				Client:      client,
				DB:          db,
				StateDB:     stateDB,
				MaxComments: maxComments,
			}
			m, err := task.Run(cmd.Context(), func(s *facebook.PostState) {
				fmt.Printf("  [%s] %s comments=%d\n", s.Status, s.URL, s.CommentsFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d comments=%d\n", m.Fetched, m.Skipped, m.Failed, m.Comments)
			return nil
		},
	}

	addFacebookFlags(cmd, &dbPath, &statePath, &delay, &cookie, &cookieFile)
	cmd.Flags().IntVar(&maxComments, "max-comments", facebook.DefaultConfig().MaxComments, "Maximum visible comments to store per post")
	return cmd
}

func newFacebookPage() *cobra.Command {
	var dbPath, statePath, cookie, cookieFile string
	var delay, maxPages, maxComments int

	cmd := &cobra.Command{
		Use:   "page <url|slug>",
		Short: "Fetch a Facebook page and visible feed posts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openFacebookDBs(dbPath, statePath, delay, cookie, cookieFile)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			task := &facebook.PageTask{
				URL:         args[0],
				Client:      client,
				DB:          db,
				StateDB:     stateDB,
				MaxPages:    maxPages,
				MaxComments: maxComments,
			}
			m, err := task.Run(cmd.Context(), func(s *facebook.PageState) {
				fmt.Printf("  [%s] %s posts=%d\n", s.Status, s.URL, s.PostsFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d posts=%d\n", m.Fetched, m.Skipped, m.Failed, m.Posts)
			return nil
		},
	}

	addFacebookFlags(cmd, &dbPath, &statePath, &delay, &cookie, &cookieFile)
	cmd.Flags().IntVar(&maxPages, "max-pages", facebook.DefaultConfig().MaxPages, "Maximum feed pages to follow")
	cmd.Flags().IntVar(&maxComments, "max-comments", facebook.DefaultConfig().MaxComments, "Maximum visible comments to store per post")
	return cmd
}

func newFacebookProfile() *cobra.Command {
	var dbPath, statePath, cookie, cookieFile string
	var delay, maxPages, maxComments int

	cmd := &cobra.Command{
		Use:   "profile <url|username|id>",
		Short: "Fetch a Facebook public profile and visible feed posts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openFacebookDBs(dbPath, statePath, delay, cookie, cookieFile)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			task := &facebook.ProfileTask{
				URL:         args[0],
				Client:      client,
				DB:          db,
				StateDB:     stateDB,
				MaxPages:    maxPages,
				MaxComments: maxComments,
			}
			m, err := task.Run(cmd.Context(), func(s *facebook.ProfileState) {
				fmt.Printf("  [%s] %s posts=%d\n", s.Status, s.URL, s.PostsFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d posts=%d\n", m.Fetched, m.Skipped, m.Failed, m.Posts)
			return nil
		},
	}

	addFacebookFlags(cmd, &dbPath, &statePath, &delay, &cookie, &cookieFile)
	cmd.Flags().IntVar(&maxPages, "max-pages", facebook.DefaultConfig().MaxPages, "Maximum feed pages to follow")
	cmd.Flags().IntVar(&maxComments, "max-comments", facebook.DefaultConfig().MaxComments, "Maximum visible comments to store per post")
	return cmd
}

func newFacebookGroup() *cobra.Command {
	var dbPath, statePath, cookie, cookieFile string
	var delay, maxPages, maxComments int

	cmd := &cobra.Command{
		Use:   "group <url|slug|id>",
		Short: "Fetch a Facebook public group and visible feed posts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openFacebookDBs(dbPath, statePath, delay, cookie, cookieFile)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			task := &facebook.GroupTask{
				URL:         args[0],
				Client:      client,
				DB:          db,
				StateDB:     stateDB,
				MaxPages:    maxPages,
				MaxComments: maxComments,
			}
			m, err := task.Run(cmd.Context(), func(s *facebook.GroupState) {
				fmt.Printf("  [%s] %s posts=%d\n", s.Status, s.URL, s.PostsFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d posts=%d\n", m.Fetched, m.Skipped, m.Failed, m.Posts)
			return nil
		},
	}

	addFacebookFlags(cmd, &dbPath, &statePath, &delay, &cookie, &cookieFile)
	cmd.Flags().IntVar(&maxPages, "max-pages", facebook.DefaultConfig().MaxPages, "Maximum feed pages to follow")
	cmd.Flags().IntVar(&maxComments, "max-comments", facebook.DefaultConfig().MaxComments, "Maximum visible comments to store per post")
	return cmd
}

func newFacebookSearch() *cobra.Command {
	var dbPath, statePath, cookie, cookieFile string
	var delay, maxPages int
	var searchType string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Best-effort Facebook keyword search and queue seeding",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openFacebookDBs(dbPath, statePath, delay, cookie, cookieFile)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			task := &facebook.SearchTask{
				Query:      args[0],
				SearchType: searchType,
				Client:     client,
				DB:         db,
				StateDB:    stateDB,
				MaxPages:   maxPages,
			}
			m, err := task.Run(cmd.Context(), func(s *facebook.SearchState) {
				fmt.Printf("  [%s] page=%d results=%d\n", s.Status, s.Page, s.ResultsFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d failed=%d results=%d\n", m.Fetched, m.Failed, m.Results)
			return nil
		},
	}

	addFacebookFlags(cmd, &dbPath, &statePath, &delay, &cookie, &cookieFile)
	cmd.Flags().StringVar(&searchType, "type", "top", "Search type: top, posts, pages, people, groups")
	cmd.Flags().IntVar(&maxPages, "max-pages", facebook.DefaultConfig().MaxPages, "Maximum search pages to follow")
	return cmd
}

func newFacebookSeed() *cobra.Command {
	var statePath string
	var file, entityType string
	var priority int

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the Facebook crawl queue from a file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			if entityType == "" {
				return fmt.Errorf("--entity is required")
			}

			stateDB, err := facebook.OpenState(statePath)
			if err != nil {
				return fmt.Errorf("open state: %w", err)
			}
			defer stateDB.Close()

			f, err := os.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()

			var items []facebook.QueueItem
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				line := strings.TrimSpace(sc.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				items = append(items, facebook.QueueItem{
					URL:        line,
					EntityType: entityType,
					Priority:   priority,
				})
			}
			if err := sc.Err(); err != nil {
				return err
			}
			if err := stateDB.EnqueueBatch(items); err != nil {
				return err
			}
			fmt.Printf("Enqueued %d items\n", len(items))
			return nil
		},
	}

	cfg := facebook.DefaultConfig()
	cmd.Flags().StringVar(&statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().StringVar(&file, "file", "", "Path to file with one URL per line")
	cmd.Flags().StringVar(&entityType, "entity", "", "Entity type: page, profile, group, post")
	cmd.Flags().IntVar(&priority, "priority", 0, "Queue priority")
	return cmd
}

func newFacebookCrawl() *cobra.Command {
	var dbPath, statePath, cookie, cookieFile string
	var delay, workers, maxPages, maxComments int

	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Run the Facebook crawl queue",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openFacebookDBs(dbPath, statePath, delay, cookie, cookieFile)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			cfg := facebook.DefaultConfig()
			cfg.DBPath = dbPath
			cfg.StatePath = statePath
			cfg.Delay = time.Duration(delay) * time.Millisecond
			cfg.Workers = workers
			cfg.MaxPages = maxPages
			cfg.MaxComments = maxComments
			cfg.Cookies = cookie
			cfg.CookiesFile = cookieFile

			jobID := fmt.Sprintf("facebook-crawl-%d", time.Now().Unix())
			_ = stateDB.CreateJob(jobID, "facebook crawl", "crawl")

			task := &facebook.CrawlTask{Config: cfg, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *facebook.CrawlState) {
				fmt.Printf("\r  done=%d pending=%d failed=%d in_flight=%d rps=%.2f", s.Done, s.Pending, s.Failed, len(s.InFlight), s.RPS)
			})
			fmt.Println()
			if err != nil {
				_ = stateDB.UpdateJob(jobID, "failed", err.Error())
				return err
			}
			_ = stateDB.UpdateJob(jobID, "done", fmt.Sprintf(`{"done":%d,"failed":%d,"duration":"%s"}`, m.Done, m.Failed, m.Duration))
			fmt.Printf("Done: processed=%d failed=%d duration=%s\n", m.Done, m.Failed, m.Duration.Round(time.Second))
			return nil
		},
	}

	addFacebookFlags(cmd, &dbPath, &statePath, &delay, &cookie, &cookieFile)
	cmd.Flags().IntVar(&workers, "workers", facebook.DefaultConfig().Workers, "Concurrent workers")
	cmd.Flags().IntVar(&maxPages, "max-pages", facebook.DefaultConfig().MaxPages, "Maximum feed/search pages per entity")
	cmd.Flags().IntVar(&maxComments, "max-comments", facebook.DefaultConfig().MaxComments, "Maximum visible comments to store per post")
	return cmd
}

func newFacebookInfo() *cobra.Command {
	var dbPath, statePath, cookie, cookieFile string
	var delay int

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show Facebook database and queue stats",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, _, err := openFacebookDBs(dbPath, statePath, delay, cookie, cookieFile)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			stats, err := db.GetStats()
			if err != nil {
				return err
			}
			pending, inProgress, done, failed := stateDB.QueueStats()
			fmt.Printf("DB: pages=%d profiles=%d groups=%d posts=%d comments=%d search_results=%d size=%s\n",
				stats.Pages, stats.Profiles, stats.Groups, stats.Posts, stats.Comments, stats.SearchResults, facebook.HumanBytes(stats.DBSize))
			fmt.Printf("Queue: pending=%d in_progress=%d done=%d failed=%d\n", pending, inProgress, done, failed)

			recent, err := db.RecentPosts(5)
			if err == nil && len(recent) > 0 {
				fmt.Println("Recent posts:")
				for _, p := range recent {
					fmt.Printf("  - %s %s\n", p.PostID, trimOneLine(p.Text, 100))
				}
			}
			return nil
		},
	}

	addFacebookFlags(cmd, &dbPath, &statePath, &delay, &cookie, &cookieFile)
	return cmd
}

func newFacebookJobs() *cobra.Command {
	var statePath string
	var limit int

	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "List recent Facebook crawl jobs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDB, err := facebook.OpenState(statePath)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			jobs, err := stateDB.ListJobs(limit)
			if err != nil {
				return err
			}
			for _, job := range jobs {
				fmt.Printf("%s\t%s\t%s\t%s\n", job.StartedAt.Format(time.RFC3339), job.Status, job.Type, job.Name)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&statePath, "state", facebook.DefaultConfig().StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of jobs to show")
	return cmd
}

func newFacebookQueue() *cobra.Command {
	var statePath, status string
	var limit int

	cmd := &cobra.Command{
		Use:   "queue",
		Short: "List queue items by status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDB, err := facebook.OpenState(statePath)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			items, err := stateDB.ListQueue(status, limit)
			if err != nil {
				return err
			}
			for _, item := range items {
				fmt.Printf("%d\t%s\t%s\t%d\n", item.ID, item.EntityType, item.URL, item.Priority)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&statePath, "state", facebook.DefaultConfig().StatePath, "Path to state.duckdb")
	cmd.Flags().StringVar(&status, "status", "pending", "Queue status: pending, in_progress, done, failed")
	cmd.Flags().IntVar(&limit, "limit", 50, "Number of queue items to show")
	return cmd
}

func trimOneLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n]
}
