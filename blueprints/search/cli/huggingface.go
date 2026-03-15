package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape/huggingface"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var hfPaperIDPattern = regexp.MustCompile(`^\d{4}\.\d{4,5}$`)

func NewHuggingFace() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "huggingface",
		Short: "Scrape public Hugging Face Hub models, datasets, spaces, collections, and papers",
		Long: `Scrape public Hugging Face Hub metadata into a local DuckDB database.

Supports one-off fetches, queue seeding, API discovery, and queue-driven crawl.
Data is stored in $HOME/data/huggingface/huggingface.duckdb.`,
	}

	cmd.AddCommand(newHFModel())
	cmd.AddCommand(newHFDataset())
	cmd.AddCommand(newHFSpace())
	cmd.AddCommand(newHFCollection())
	cmd.AddCommand(newHFPaper())
	cmd.AddCommand(newHFDiscover())
	cmd.AddCommand(newHFSeed())
	cmd.AddCommand(newHFCrawl())
	cmd.AddCommand(newHFInfo())
	cmd.AddCommand(newHFJobs())
	cmd.AddCommand(newHFQueue())

	return cmd
}

func addHFFlags(cmd *cobra.Command, dbPath, statePath *string, delay, workers, maxPages *int, types *[]string) {
	cfg := huggingface.DefaultConfig()
	cmd.Flags().StringVar(dbPath, "db", cfg.DBPath, "Path to huggingface.duckdb")
	cmd.Flags().StringVar(statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
	if workers != nil {
		cmd.Flags().IntVar(workers, "workers", cfg.Workers, "Worker concurrency")
	}
	if maxPages != nil {
		cmd.Flags().IntVar(maxPages, "max-pages", cfg.MaxPages, "Maximum discovery pages per entity (0 = unlimited)")
	}
	if types != nil {
		cmd.Flags().StringSliceVar(types, "types", cfg.Types, "Entity types to include")
	}
}

func openHFDBs(dbPath, statePath string, delay, workers, maxPages int, types []string) (*huggingface.DB, *huggingface.State, *huggingface.Client, huggingface.Config, error) {
	cfg := huggingface.DefaultConfig()
	cfg.DBPath = dbPath
	cfg.StatePath = statePath
	cfg.Delay = time.Duration(delay) * time.Millisecond
	if workers > 0 {
		cfg.Workers = workers
	}
	cfg.MaxPages = maxPages
	cfg.Types = types

	db, err := huggingface.OpenDB(dbPath)
	if err != nil {
		return nil, nil, nil, cfg, fmt.Errorf("open db: %w", err)
	}
	stateDB, err := huggingface.OpenState(statePath)
	if err != nil {
		db.Close()
		return nil, nil, nil, cfg, fmt.Errorf("open state: %w", err)
	}
	client := huggingface.NewClient(cfg)
	return db, stateDB, client, cfg, nil
}

func newHFModel() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "model <repo_id|url>",
		Short: "Fetch a single Hugging Face model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openHFDBs(dbPath, statePath, delay, 0, 0, nil)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			task := &huggingface.ModelTask{URL: args[0], Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *huggingface.ModelState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addHFFlags(cmd, &dbPath, &statePath, &delay, nil, nil, nil)
	return cmd
}

func newHFDataset() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "dataset <repo_id|url>",
		Short: "Fetch a single Hugging Face dataset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openHFDBs(dbPath, statePath, delay, 0, 0, nil)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			task := &huggingface.DatasetTask{URL: args[0], Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *huggingface.DatasetState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addHFFlags(cmd, &dbPath, &statePath, &delay, nil, nil, nil)
	return cmd
}

func newHFSpace() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "space <repo_id|url>",
		Short: "Fetch a single Hugging Face space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openHFDBs(dbPath, statePath, delay, 0, 0, nil)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			task := &huggingface.SpaceTask{URL: args[0], Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *huggingface.SpaceState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addHFFlags(cmd, &dbPath, &statePath, &delay, nil, nil, nil)
	return cmd
}

func newHFCollection() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "collection <namespace/slug|url>",
		Short: "Fetch a single Hugging Face collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openHFDBs(dbPath, statePath, delay, 0, 0, nil)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			task := &huggingface.CollectionTask{URL: args[0], Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *huggingface.CollectionState) {
				fmt.Printf("  [%s] %s items=%d\n", s.Status, s.URL, s.ItemsFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addHFFlags(cmd, &dbPath, &statePath, &delay, nil, nil, nil)
	return cmd
}

func newHFPaper() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "paper <id|url>",
		Short: "Fetch a single Hugging Face paper",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openHFDBs(dbPath, statePath, delay, 0, 0, nil)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			task := &huggingface.PaperTask{URL: args[0], Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *huggingface.PaperState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addHFFlags(cmd, &dbPath, &statePath, &delay, nil, nil, nil)
	return cmd
}

func newHFDiscover() *cobra.Command {
	var dbPath, statePath string
	var delay, workers, maxPages, priority int
	var types []string
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover public Hugging Face Hub entities through list APIs and enqueue them",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, cfg, err := openHFDBs(dbPath, statePath, delay, workers, maxPages, types)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			jobID := uuid.NewString()
			_ = stateDB.CreateJob(jobID, "huggingface discover", "discover")

			task := &huggingface.DiscoverTask{Config: cfg, Client: client, StateDB: stateDB, Priority: priority}
			metric, err := task.Run(cmd.Context(), func(s *huggingface.DiscoverState) {
				switch s.Status {
				case "fetching":
					fmt.Printf("  [%s] page=%d type=%s\n", s.Status, s.Page, s.EntityType)
				default:
					fmt.Printf("  [%s] page=%d type=%s discovered=%d enqueued=%d\n", s.Status, s.Page, s.EntityType, s.Discovered, s.Enqueued)
				}
			})
			stats, _ := json.Marshal(metric)
			if err != nil {
				_ = stateDB.UpdateJob(jobID, "failed", string(stats))
				return err
			}
			_ = stateDB.UpdateJob(jobID, "done", string(stats))
			fmt.Printf("Done: discovered=%d enqueued=%d pages=%d\n", metric.Discovered, metric.Enqueued, metric.Pages)
			return nil
		},
	}
	addHFFlags(cmd, &dbPath, &statePath, &delay, &workers, &maxPages, &types)
	cmd.Flags().IntVar(&priority, "priority", 1, "Queue priority for discovered URLs")
	return cmd
}

func newHFSeed() *cobra.Command {
	var statePath string
	var file, entityType string
	var priority int
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the Hugging Face queue from a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			stateDB, err := huggingface.OpenState(statePath)
			if err != nil {
				return fmt.Errorf("open state: %w", err)
			}
			defer stateDB.Close()

			f, err := os.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()

			var queued int
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				etype, url, err := normalizeHFSeedRef(entityType, line)
				if err != nil {
					fmt.Printf("  [skip] %s (%v)\n", line, err)
					continue
				}
				if err := stateDB.Enqueue(url, etype, priority); err != nil {
					return err
				}
				queued++
			}
			if err := scanner.Err(); err != nil {
				return err
			}
			fmt.Printf("Seeded %d queue items\n", queued)
			return nil
		},
	}
	cfg := huggingface.DefaultConfig()
	cmd.Flags().StringVar(&statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().StringVar(&file, "file", "", "Path to newline-delimited refs")
	cmd.Flags().StringVar(&entityType, "type", "", "Entity type: model|dataset|space|collection|paper (default: infer)")
	cmd.Flags().IntVar(&priority, "priority", 1, "Queue priority")
	return cmd
}

func newHFCrawl() *cobra.Command {
	var dbPath, statePath string
	var delay, workers, maxPages int
	var types []string
	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Run the Hugging Face queue-driven crawl",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, cfg, err := openHFDBs(dbPath, statePath, delay, workers, maxPages, types)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			jobID := uuid.NewString()
			_ = stateDB.CreateJob(jobID, "huggingface crawl", "crawl")

			task := &huggingface.CrawlTask{Config: cfg, Client: client, DB: db, StateDB: stateDB}
			metric, err := task.Run(cmd.Context(), func(s *huggingface.CrawlState) {
				huggingface.PrintCrawlProgress(s)
			})
			fmt.Println()
			stats, _ := json.Marshal(metric)
			if err != nil {
				_ = stateDB.UpdateJob(jobID, "failed", string(stats))
				return err
			}
			_ = stateDB.UpdateJob(jobID, "done", string(stats))
			fmt.Printf("Done: fetched=%d duration=%s\n", metric.Done, metric.Duration.Round(time.Second))
			return nil
		},
	}
	addHFFlags(cmd, &dbPath, &statePath, &delay, &workers, &maxPages, &types)
	return cmd
}

func newHFInfo() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show Hugging Face database and queue stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, _, _, err := openHFDBs(dbPath, statePath, delay, 0, 0, nil)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			return huggingface.PrintStats(db, stateDB)
		},
	}
	addHFFlags(cmd, &dbPath, &statePath, &delay, nil, nil, nil)
	return cmd
}

func newHFJobs() *cobra.Command {
	var statePath string
	var limit int
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "List recent Hugging Face crawl jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDB, err := huggingface.OpenState(statePath)
			if err != nil {
				return err
			}
			defer stateDB.Close()
			jobs, err := stateDB.ListJobs(limit)
			if err != nil {
				return err
			}
			for _, job := range jobs {
				fmt.Printf("%s  %-8s %-10s %s\n", job.StartedAt.Format(time.RFC3339), job.Status, job.Type, job.Name)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&statePath, "state", huggingface.DefaultConfig().StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum jobs to show")
	return cmd
}

func newHFQueue() *cobra.Command {
	var statePath, status string
	var limit int
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Inspect Hugging Face queue entries by status",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDB, err := huggingface.OpenState(statePath)
			if err != nil {
				return err
			}
			defer stateDB.Close()
			items, err := stateDB.ListQueue(status, limit)
			if err != nil {
				return err
			}
			for _, item := range items {
				fmt.Printf("%6d  %-10s  p=%d  %s\n", item.ID, item.EntityType, item.Priority, item.URL)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&statePath, "state", huggingface.DefaultConfig().StatePath, "Path to state.duckdb")
	cmd.Flags().StringVar(&status, "status", "pending", "Queue status filter")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum rows to show")
	return cmd
}

func normalizeHFSeedRef(entityType, ref string) (string, string, error) {
	switch strings.ToLower(strings.TrimSpace(entityType)) {
	case "", "auto":
		if hfPaperIDPattern.MatchString(strings.TrimSpace(ref)) {
			entityType = huggingface.EntityPaper
		} else {
			entityType = huggingface.InferEntityType(ref)
		}
	case "models":
		entityType = huggingface.EntityModel
	case "datasets":
		entityType = huggingface.EntityDataset
	case "spaces":
		entityType = huggingface.EntitySpace
	case "collections":
		entityType = huggingface.EntityCollection
	case "papers":
		entityType = huggingface.EntityPaper
	}
	switch entityType {
	case huggingface.EntityModel:
		_, url, err := huggingface.NormalizeModelRef(ref)
		return entityType, url, err
	case huggingface.EntityDataset:
		_, url, err := huggingface.NormalizeDatasetRef(ref)
		return entityType, url, err
	case huggingface.EntitySpace:
		_, url, err := huggingface.NormalizeSpaceRef(ref)
		return entityType, url, err
	case huggingface.EntityCollection:
		_, url, err := huggingface.NormalizeCollectionRef(ref)
		return entityType, url, err
	case huggingface.EntityPaper:
		_, url, err := huggingface.NormalizePaperRef(ref)
		return entityType, url, err
	default:
		return "", "", fmt.Errorf("unknown entity type: %s", entityType)
	}
}
