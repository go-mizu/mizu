package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape/kaggle"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func NewKaggle() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kaggle",
		Short: "Scrape public Kaggle datasets, models, competitions, notebooks, and profiles",
		Long: `Scrape public Kaggle content into a local DuckDB database.

Discovery is API-first for datasets and models. Competitions, notebooks, and
profiles are supported from direct URLs and queue seeds using public HTML metadata.

Examples:
  search kaggle dataset zynicide/wine-reviews
  search kaggle model google/gemma
  search kaggle competition titanic
  search kaggle notebook pmarcelino/comprehensive-data-exploration-with-python
  search kaggle profile zynicide
  search kaggle discover --types dataset,model --max-pages 3
  search kaggle crawl --workers 2
  search kaggle info`,
	}
	cmd.AddCommand(newKaggleDataset())
	cmd.AddCommand(newKaggleModel())
	cmd.AddCommand(newKaggleCompetition())
	cmd.AddCommand(newKaggleNotebook())
	cmd.AddCommand(newKaggleProfile())
	cmd.AddCommand(newKaggleDiscover())
	cmd.AddCommand(newKaggleSeed())
	cmd.AddCommand(newKaggleCrawl())
	cmd.AddCommand(newKaggleInfo())
	cmd.AddCommand(newKaggleJobs())
	cmd.AddCommand(newKaggleQueue())
	return cmd
}

func addKaggleFlags(cmd *cobra.Command, dbPath, statePath *string, delay, maxPages *int) {
	cfg := kaggle.DefaultConfig()
	cmd.Flags().StringVar(dbPath, "db", cfg.DBPath, "Path to kaggle.duckdb")
	cmd.Flags().StringVar(statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
	cmd.Flags().IntVar(maxPages, "max-pages", cfg.MaxPages, "Maximum pages to fetch during discovery")
}

func openKaggleDBs(dbPath, statePath string, delay, maxPages int) (*kaggle.DB, *kaggle.State, *kaggle.Client, kaggle.Config, error) {
	cfg := kaggle.DefaultConfig()
	cfg.DBPath = dbPath
	cfg.StatePath = statePath
	cfg.Delay = time.Duration(delay) * time.Millisecond
	cfg.MaxPages = maxPages

	db, err := kaggle.OpenDB(dbPath)
	if err != nil {
		return nil, nil, nil, cfg, fmt.Errorf("open db: %w", err)
	}
	stateDB, err := kaggle.OpenState(statePath)
	if err != nil {
		db.Close()
		return nil, nil, nil, cfg, fmt.Errorf("open state: %w", err)
	}
	client := kaggle.NewClient(cfg)
	return db, stateDB, client, cfg, nil
}

func newKaggleDataset() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int
	cmd := &cobra.Command{
		Use:   "dataset <owner/slug|url>",
		Short: "Fetch a single Kaggle dataset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openKaggleDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := kaggle.NormalizeDatasetURL(args[0])
			task := &kaggle.DatasetTask{URL: url, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *kaggle.DatasetState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addKaggleFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

func newKaggleModel() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int
	cmd := &cobra.Command{
		Use:   "model <owner/slug|url>",
		Short: "Fetch a single Kaggle model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openKaggleDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := kaggle.NormalizeModelURL(args[0])
			task := &kaggle.ModelTask{URL: url, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *kaggle.ModelState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addKaggleFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

func newKaggleCompetition() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int
	cmd := &cobra.Command{
		Use:   "competition <slug|url>",
		Short: "Fetch a single Kaggle competition page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openKaggleDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := kaggle.NormalizeCompetitionURL(args[0])
			task := &kaggle.CompetitionTask{URL: url, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *kaggle.CompetitionState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addKaggleFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

func newKaggleNotebook() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int
	cmd := &cobra.Command{
		Use:   "notebook <owner/slug|url>",
		Short: "Fetch a single Kaggle notebook page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openKaggleDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := kaggle.NormalizeNotebookURL(args[0])
			task := &kaggle.NotebookTask{URL: url, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *kaggle.NotebookState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addKaggleFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

func newKaggleProfile() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int
	cmd := &cobra.Command{
		Use:   "profile <handle|url>",
		Short: "Fetch a single Kaggle profile page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openKaggleDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := kaggle.NormalizeProfileURL(args[0])
			task := &kaggle.ProfileTask{URL: url, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *kaggle.ProfileState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addKaggleFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

func newKaggleDiscover() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int
	var typesCSV string
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover public Kaggle datasets and models and enqueue them",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			db, stateDB, client, cfg, err := openKaggleDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			cfg.Types = parseCSV(typesCSV)
			if len(cfg.Types) == 0 {
				cfg.Types = []string{kaggle.EntityDataset, kaggle.EntityModel}
			}
			if err := kaggle.ValidateDiscoverTypes(cfg.Types); err != nil {
				return err
			}

			task := &kaggle.DiscoverTask{Config: cfg, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *kaggle.DiscoverState) {
				fmt.Printf("  [%s] page=%d datasets=%d models=%d profiles=%d\n",
					s.Status, s.Page, s.DatasetsFound, s.ModelsFound, s.ProfilesEnqueued)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: datasets=%d models=%d profiles=%d pages=%d\n",
				m.DatasetsFound, m.ModelsFound, m.ProfilesEnqueued, m.Pages)
			return nil
		},
	}
	addKaggleFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	cmd.Flags().StringVar(&typesCSV, "types", "dataset,model", "Discovery types: dataset,model")
	return cmd
}

func newKaggleSeed() *cobra.Command {
	var dbPath, statePath, file, entityType string
	var delay, maxPages, priority int
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the Kaggle crawl queue from a file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			_, stateDB, _, _, err := openKaggleDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			f, err := os.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			count := 0
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				switch entityType {
				case kaggle.EntityDataset:
					line = kaggle.NormalizeDatasetURL(line)
				case kaggle.EntityModel:
					line = kaggle.NormalizeModelURL(line)
				case kaggle.EntityCompetition:
					line = kaggle.NormalizeCompetitionURL(line)
				case kaggle.EntityNotebook:
					line = kaggle.NormalizeNotebookURL(line)
				case kaggle.EntityProfile:
					line = kaggle.NormalizeProfileURL(line)
				default:
					return fmt.Errorf("unsupported entity type: %s", entityType)
				}
				if err := stateDB.Enqueue(line, entityType, priority); err != nil {
					return err
				}
				count++
			}
			if err := scanner.Err(); err != nil {
				return err
			}
			fmt.Printf("Enqueued %d items.\n", count)
			return nil
		},
	}
	addKaggleFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	cmd.Flags().StringVar(&file, "file", "", "Path to a file with one item per line")
	cmd.Flags().StringVar(&entityType, "type", kaggle.EntityDataset, "Entity type: dataset|model|competition|notebook|profile")
	cmd.Flags().IntVar(&priority, "priority", 0, "Queue priority")
	return cmd
}

func newKaggleCrawl() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages, workers int
	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Drain the Kaggle crawl queue",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			db, stateDB, client, cfg, err := openKaggleDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			cfg.Workers = workers
			jobID := uuid.NewString()
			_ = stateDB.CreateJob(jobID, "kaggle crawl", "crawl")
			task := &kaggle.CrawlTask{Config: cfg, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *kaggle.CrawlState) {
				kaggle.PrintCrawlProgress(s)
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
	addKaggleFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	cmd.Flags().IntVar(&workers, "workers", kaggle.DefaultWorkers, "Number of concurrent crawl workers")
	return cmd
}

func newKaggleInfo() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show Kaggle DB and queue stats",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			db, stateDB, _, _, err := openKaggleDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			return kaggle.PrintStats(db, stateDB)
		},
	}
	addKaggleFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

func newKaggleJobs() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages, limit int
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "List recent Kaggle crawl jobs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, stateDB, _, _, err := openKaggleDBs(dbPath, statePath, delay, maxPages)
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
	addKaggleFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum jobs to show")
	return cmd
}

func newKaggleQueue() *cobra.Command {
	var dbPath, statePath, status string
	var delay, maxPages, limit int
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "List queued Kaggle crawl items",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, stateDB, _, _, err := openKaggleDBs(dbPath, statePath, delay, maxPages)
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
				fmt.Printf("[%d] %-11s p=%d  %s\n", item.ID, item.EntityType, item.Priority, item.URL)
			}
			return nil
		},
	}
	addKaggleFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	cmd.Flags().StringVar(&status, "status", "pending", "Queue status: pending, in_progress, done, failed")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum queue items to show")
	return cmd
}

func parseCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
