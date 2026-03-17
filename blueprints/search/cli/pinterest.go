package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape/pinterest"
	"github.com/spf13/cobra"
)

// NewPinterest creates the pinterest CLI command.
func NewPinterest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pinterest",
		Short: "Scrape Pinterest pins, boards, and users",
		Long: `Scrape public Pinterest data into a local DuckDB database.

Uses Pinterest's internal Resource API — no browser required.
Data is stored in $HOME/data/pinterest/pinterest.duckdb.

Examples:
  search pinterest search "gouache painting"     # Search and store pins
  search pinterest search "watercolor" --max-pins 1000
  search pinterest board user/board-name         # Fetch all pins from a board
  search pinterest board https://www.pinterest.com/username/board/
  search pinterest user someusername             # Fetch user profile + boards
  search pinterest user someusername --boards    # Also enqueue each board
  search pinterest seed --file urls.txt          # Seed queue from file
  search pinterest crawl --workers 2             # Bulk crawl the queue
  search pinterest info                          # Show database stats`,
	}

	cmd.AddCommand(newPinterestSearch())
	cmd.AddCommand(newPinterestBoard())
	cmd.AddCommand(newPinterestUser())
	cmd.AddCommand(newPinterestSeed())
	cmd.AddCommand(newPinterestCrawl())
	cmd.AddCommand(newPinterestInfo())
	cmd.AddCommand(newPinterestJobs())
	cmd.AddCommand(newPinterestQueue())

	return cmd
}

// ── Shared flag helpers ───────────────────────────────────────────────────────

func addPinterestFlags(cmd *cobra.Command, dbPath, statePath *string, delay *int) {
	cfg := pinterest.DefaultConfig()
	cmd.Flags().StringVar(dbPath, "db", cfg.DBPath, "Path to pinterest.duckdb")
	cmd.Flags().StringVar(statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
}

func openPinterestDBs(dbPath, statePath string, delay int) (*pinterest.DB, *pinterest.State, *pinterest.Client, error) {
	cfg := pinterest.DefaultConfig()
	cfg.DBPath = dbPath
	cfg.StatePath = statePath
	cfg.Delay = time.Duration(delay) * time.Millisecond

	db, err := pinterest.OpenDB(dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open db: %w", err)
	}

	stateDB, err := pinterest.OpenState(statePath)
	if err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("open state: %w", err)
	}

	client, err := pinterest.NewClient(cfg)
	if err != nil {
		db.Close()
		stateDB.Close()
		return nil, nil, nil, fmt.Errorf("pinterest client: %w", err)
	}

	return db, stateDB, client, nil
}

// ── search ────────────────────────────────────────────────────────────────────

func newPinterestSearch() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPins int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Pinterest pins and store results",
		Args:  cobra.ExactArgs(1),
		Example: `  search pinterest search "gouache painting"
  search pinterest search "watercolor" --max-pins 1000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openPinterestDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			query := args[0]
			fmt.Printf("Searching Pinterest for %q (max %d pins)...\n", query, maxPins)

			task := &pinterest.SearchTask{
				Query:   query,
				MaxPins: maxPins,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *pinterest.SearchState) {
				if s.PinsFound > 0 {
					fmt.Printf("\r  [%s] %d pins found", s.Status, s.PinsFound)
				}
			})
			fmt.Println()
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d failed=%d\n", m.Fetched, m.Failed)
			return nil
		},
	}

	addPinterestFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().IntVar(&maxPins, "max-pins", pinterest.DefaultMaxPins, "Maximum pins to fetch (0 = unlimited)")
	return cmd
}

// ── board ─────────────────────────────────────────────────────────────────────

func newPinterestBoard() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPins int

	cmd := &cobra.Command{
		Use:   "board <url|user/board>",
		Short: "Fetch all pins from a Pinterest board",
		Args:  cobra.ExactArgs(1),
		Example: `  search pinterest board username/board-slug
  search pinterest board https://www.pinterest.com/username/board/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openPinterestDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			boardURL := pinterest.NormalizeBoardURL(args[0])
			fmt.Printf("Fetching board: %s\n", boardURL)

			task := &pinterest.BoardTask{
				URL:     boardURL,
				MaxPins: maxPins,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *pinterest.BoardState) {
				fmt.Printf("\r  [%s] boardID=%s pins=%d", s.Status, s.BoardID, s.PinsFound)
			})
			fmt.Println()
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d pages=%d\n",
				m.Fetched, m.Skipped, m.Failed, m.Pages)
			return nil
		},
	}

	addPinterestFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().IntVar(&maxPins, "max-pins", 0, "Maximum pins to fetch (0 = unlimited)")
	return cmd
}

// ── user ──────────────────────────────────────────────────────────────────────

func newPinterestUser() *cobra.Command {
	var dbPath, statePath string
	var delay int
	var boards bool

	cmd := &cobra.Command{
		Use:   "user <username>",
		Short: "Fetch a Pinterest user profile and their boards",
		Args:  cobra.ExactArgs(1),
		Example: `  search pinterest user someusername
  search pinterest user someusername --boards`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openPinterestDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			username := args[0]
			fmt.Printf("Fetching user: %s\n", username)

			task := &pinterest.UserTask{
				URL:           pinterest.NormalizeUserURL(username),
				IncludeBoards: boards,
				Client:        client,
				DB:            db,
				StateDB:       stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *pinterest.UserState) {
				fmt.Printf("\r  [%s] @%s boards=%d", s.Status, s.Username, s.BoardsFound)
			})
			fmt.Println()
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			if boards {
				fmt.Println("  Boards enqueued into queue — run 'search pinterest crawl' to fetch pins.")
			}
			return nil
		},
	}

	addPinterestFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().BoolVar(&boards, "boards", false, "Enqueue each board for crawling")
	return cmd
}

// ── seed ──────────────────────────────────────────────────────────────────────

func newPinterestSeed() *cobra.Command {
	var dbPath, statePath string
	var delay, priority int
	var file, entityType string

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the queue from a file (one URL or query per line)",
		Args:  cobra.NoArgs,
		Example: `  search pinterest seed --file boards.txt --entity board
  search pinterest seed --file users.txt --entity user --priority 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}

			cfg := pinterest.DefaultConfig()
			cfg.DBPath = dbPath
			cfg.StatePath = statePath

			stateDB, err := pinterest.OpenState(statePath)
			if err != nil {
				return fmt.Errorf("open state: %w", err)
			}
			defer stateDB.Close()

			f, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("open file: %w", err)
			}
			defer f.Close()

			entity := entityType
			if entity == "" {
				entity = pinterest.EntityBoard
			}

			total := 0
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				var rawURL string
				switch entity {
				case pinterest.EntityBoard:
					rawURL = pinterest.NormalizeBoardURL(line)
				case pinterest.EntityUser:
					rawURL = pinterest.NormalizeUserURL(line)
				default:
					rawURL = line
				}
				if err := stateDB.Enqueue(rawURL, entity, priority); err == nil {
					total++
				}
			}
			if err := scanner.Err(); err != nil {
				return err
			}

			fmt.Printf("Enqueued %d %s URLs from %s\n", total, entity, file)
			return nil
		},
	}

	addPinterestFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().StringVar(&file, "file", "", "File with URLs or queries (one per line)")
	cmd.Flags().StringVar(&entityType, "entity", "board", "Entity type: board, user, search")
	cmd.Flags().IntVar(&priority, "priority", 10, "Queue priority")
	return cmd
}

// ── crawl ─────────────────────────────────────────────────────────────────────

func newPinterestCrawl() *cobra.Command {
	var dbPath, statePath string
	var delay, workers int

	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Bulk crawl from the queue (seed with 'user --boards' or 'seed' first)",
		Args:  cobra.NoArgs,
		Example: `  search pinterest user someuser --boards && search pinterest crawl
  search pinterest crawl --workers 3 --delay 300`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := pinterest.DefaultConfig()
			cfg.DBPath = dbPath
			cfg.StatePath = statePath
			cfg.Workers = workers
			cfg.Delay = time.Duration(delay) * time.Millisecond

			db, err := pinterest.OpenDB(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			stateDB, err := pinterest.OpenState(statePath)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			client, err := pinterest.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("pinterest client: %w", err)
			}

			pending, _, done, failed := stateDB.QueueStats()
			fmt.Printf("Queue: pending=%d done=%d failed=%d\n", pending, done, failed)
			if pending == 0 {
				fmt.Println("Queue is empty. Run 'search pinterest user --boards' or 'search pinterest seed' first.")
				return nil
			}

			fmt.Printf("Starting crawl with %d workers, delay=%s ...\n", workers, cfg.Delay)
			stateDB.CreateJob("crawl-"+fmt.Sprintf("%d", time.Now().Unix()), "bulk-crawl", "crawl")

			task := &pinterest.CrawlTask{
				Config:  cfg,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}

			m, err := task.Run(cmd.Context(), func(s *pinterest.CrawlState) {
				pinterest.PrintCrawlProgress(s)
			})
			if err != nil {
				return err
			}

			fmt.Printf("\nCrawl complete: done=%d failed=%d duration=%s\n",
				m.Done, m.Failed, m.Duration.Round(time.Second))
			return nil
		},
	}

	cfg := pinterest.DefaultConfig()
	cmd.Flags().StringVar(&dbPath, "db", cfg.DBPath, "Path to pinterest.duckdb")
	cmd.Flags().StringVar(&statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(&workers, "workers", cfg.Workers, "Concurrent fetch workers")
	cmd.Flags().IntVar(&delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
	return cmd
}

// ── info ──────────────────────────────────────────────────────────────────────

func newPinterestInfo() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show Pinterest database stats and queue depth",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := pinterest.OpenDB(dbPath)
			if err != nil {
				return fmt.Errorf("open db: %w", err)
			}
			defer db.Close()

			stateDB, err := pinterest.OpenState(statePath)
			if err != nil {
				return fmt.Errorf("open state: %w", err)
			}
			defer stateDB.Close()

			return pinterest.PrintStats(db, stateDB)
		},
	}

	addPinterestFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

// ── jobs ──────────────────────────────────────────────────────────────────────

func newPinterestJobs() *cobra.Command {
	var dbPath, statePath string
	var delay, limit int

	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "List recent crawl jobs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDB, err := pinterest.OpenState(statePath)
			if err != nil {
				return fmt.Errorf("open state: %w", err)
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
			fmt.Println("── Recent Jobs ──")
			for _, j := range jobs {
				dur := ""
				if !j.CompletedAt.IsZero() {
					dur = j.CompletedAt.Sub(j.StartedAt).Round(time.Second).String()
				}
				fmt.Printf("  [%s] %s  %s  started=%s  duration=%s\n",
					j.Status, j.JobID, j.Name,
					j.StartedAt.Format("2006-01-02 15:04"), dur)
			}
			return nil
		},
	}

	addPinterestFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of jobs to show")
	return cmd
}

// ── queue ─────────────────────────────────────────────────────────────────────

func newPinterestQueue() *cobra.Command {
	var dbPath, statePath string
	var delay, limit int
	var status string

	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Inspect queue items",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDB, err := pinterest.OpenState(statePath)
			if err != nil {
				return fmt.Errorf("open state: %w", err)
			}
			defer stateDB.Close()

			items, err := stateDB.ListQueue(status, limit)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Printf("No %s items in queue.\n", status)
				return nil
			}
			fmt.Printf("── Queue items (status=%s) ──\n", status)
			for _, it := range items {
				fmt.Printf("  [%s] %s\n", it.EntityType, it.URL)
			}
			return nil
		},
	}

	addPinterestFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().StringVar(&status, "status", "pending", "Filter by status: pending, failed, done, in_progress")
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of items to show")
	return cmd
}
