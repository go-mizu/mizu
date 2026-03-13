package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape/youtube"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func NewYouTube() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "youtube",
		Short: "Scrape public YouTube videos, channels, playlists, and search results",
		Long: `Scrape public YouTube data into a local DuckDB database.

Uses public YouTube HTML and embedded JSON only. No API key is required.
Data is stored in $HOME/data/youtube/youtube.duckdb.

Examples:
  search youtube video dQw4w9WgXcQ
  search youtube channel @GoogleDevelopers
  search youtube playlist PL590L5WQmH8fJ54F1L7aRQlQ-Qc8-ND8B
  search youtube search "golang" --max-results 20 --enqueue
  search youtube seed --file urls.txt --entity video
  search youtube crawl --workers 2
  search youtube info`,
	}
	cmd.AddCommand(newYouTubeVideo())
	cmd.AddCommand(newYouTubeChannel())
	cmd.AddCommand(newYouTubePlaylist())
	cmd.AddCommand(newYouTubeSearch())
	cmd.AddCommand(newYouTubeSeed())
	cmd.AddCommand(newYouTubeCrawl())
	cmd.AddCommand(newYouTubeInfo())
	cmd.AddCommand(newYouTubeJobs())
	cmd.AddCommand(newYouTubeQueue())
	return cmd
}

func addYouTubeFlags(cmd *cobra.Command, dbPath, statePath *string, delay *int) {
	cfg := youtube.DefaultConfig()
	cmd.Flags().StringVar(dbPath, "db", cfg.DBPath, "Path to youtube.duckdb")
	cmd.Flags().StringVar(statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
}

func openYouTubeDBs(dbPath, statePath string, delay int) (*youtube.DB, *youtube.State, *youtube.Client, error) {
	cfg := youtube.DefaultConfig()
	cfg.DBPath = dbPath
	cfg.StatePath = statePath
	cfg.Delay = time.Duration(delay) * time.Millisecond
	db, err := youtube.OpenDB(dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open db: %w", err)
	}
	stateDB, err := youtube.OpenState(statePath)
	if err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("open state: %w", err)
	}
	return db, stateDB, youtube.NewClient(cfg), nil
}

func newYouTubeVideo() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "video <id|url>",
		Short: "Fetch a single YouTube video",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openYouTubeDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			url := youtube.NormalizeVideoURL(args[0])
			task := &youtube.VideoTask{URL: url, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *youtube.VideoState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addYouTubeFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

func newYouTubeChannel() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "channel <id|url|@handle>",
		Short: "Fetch a YouTube channel and visible videos",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openYouTubeDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			url := youtube.NormalizeChannelURL(args[0])
			task := &youtube.ChannelTask{URL: url, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *youtube.ChannelState) {
				fmt.Printf("  [%s] %s videos=%d\n", s.Status, s.URL, s.VideosFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addYouTubeFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

func newYouTubePlaylist() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "playlist <id|url>",
		Short: "Fetch a YouTube playlist and visible items",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openYouTubeDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			url := youtube.NormalizePlaylistURL(args[0])
			task := &youtube.PlaylistTask{URL: url, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *youtube.PlaylistState) {
				fmt.Printf("  [%s] %s videos=%d\n", s.Status, s.URL, s.VideosFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addYouTubeFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

func newYouTubeSearch() *cobra.Command {
	var dbPath, statePath string
	var delay, maxResults int
	var enqueue bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search YouTube and store results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openYouTubeDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			task := &youtube.SearchTask{
				Query:      args[0],
				MaxResults: maxResults,
				Enqueue:    enqueue,
				Client:     client,
				DB:         db,
				StateDB:    stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *youtube.SearchState) {
				fmt.Printf("  [%s] query=%q results=%d\n", s.Status, s.Query, s.ResultsFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addYouTubeFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().IntVar(&maxResults, "max-results", youtube.DefaultMaxResults, "Maximum results to store")
	cmd.Flags().BoolVar(&enqueue, "enqueue", false, "Enqueue discovered entities for crawl")
	return cmd
}

func newYouTubeSeed() *cobra.Command {
	var dbPath, statePath, file, entityType string
	var delay, priority int
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the crawl queue from a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			stateDB, err := youtube.OpenState(statePath)
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
			var count int
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				switch entityType {
				case youtube.EntityVideo:
					line = youtube.NormalizeVideoURL(line)
				case youtube.EntityChannel:
					line = youtube.NormalizeChannelURL(line)
				case youtube.EntityPlaylist:
					line = youtube.NormalizePlaylistURL(line)
				case youtube.EntitySearch:
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
			_ = dbPath
			_ = delay
			return nil
		},
	}
	addYouTubeFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().StringVar(&file, "file", "", "Path to a file with one item per line")
	cmd.Flags().StringVar(&entityType, "entity", youtube.EntityVideo, "Entity type: video|channel|playlist|search")
	cmd.Flags().IntVar(&priority, "priority", 0, "Queue priority")
	return cmd
}

func newYouTubeCrawl() *cobra.Command {
	var dbPath, statePath string
	var delay, workers int
	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Process queued YouTube crawl tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := youtube.DefaultConfig()
			cfg.DBPath = dbPath
			cfg.StatePath = statePath
			cfg.Delay = time.Duration(delay) * time.Millisecond
			cfg.Workers = workers
			db, err := youtube.OpenDB(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()
			stateDB, err := youtube.OpenState(statePath)
			if err != nil {
				return err
			}
			defer stateDB.Close()
			pending, err := stateDB.PendingCount()
			if err != nil {
				return err
			}
			if pending == 0 {
				fmt.Println("Queue is empty. Run 'search youtube seed' or 'search youtube search --enqueue' first.")
				return nil
			}
			jobID := uuid.NewString()
			_ = stateDB.CreateJob(jobID, "youtube-crawl", "crawl")
			task := &youtube.CrawlTask{Config: cfg, Client: youtube.NewClient(cfg), DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *youtube.CrawlState) { youtube.PrintCrawlProgress(s) })
			fmt.Println()
			if err != nil {
				_ = stateDB.UpdateJob(jobID, "failed", err.Error())
				return err
			}
			_ = stateDB.UpdateJob(jobID, "done", fmt.Sprintf("done=%d duration=%s", m.Done, m.Duration))
			fmt.Printf("Done: processed=%d duration=%s\n", m.Done, m.Duration)
			return nil
		},
	}
	addYouTubeFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().IntVar(&workers, "workers", youtube.DefaultWorkers, "Number of concurrent workers")
	return cmd
}

func newYouTubeInfo() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show YouTube database and queue stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, _, err := openYouTubeDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			return youtube.PrintStats(db, stateDB)
		},
	}
	addYouTubeFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

func newYouTubeJobs() *cobra.Command {
	var dbPath, statePath string
	var delay, limit int
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "List recent crawl jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, stateDB, _, err := openYouTubeDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer stateDB.Close()
			jobs, err := stateDB.ListJobs(limit)
			if err != nil {
				return err
			}
			for _, job := range jobs {
				fmt.Printf("%s  %-8s  %-8s  %s\n", job.StartedAt.Format(time.RFC3339), job.Type, job.Status, job.JobID)
			}
			return nil
		},
	}
	addYouTubeFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum jobs to show")
	return cmd
}

func newYouTubeQueue() *cobra.Command {
	var dbPath, statePath, status string
	var delay, limit int
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "List queued items",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, stateDB, _, err := openYouTubeDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer stateDB.Close()
			items, err := stateDB.ListQueue(status, limit)
			if err != nil {
				return err
			}
			for _, item := range items {
				fmt.Printf("%6d  %-8s  p=%d  %s\n", item.ID, item.EntityType, item.Priority, item.URL)
			}
			return nil
		},
	}
	addYouTubeFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().StringVar(&status, "status", "pending", "Queue status: pending|failed|done|in_progress")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum items to show")
	return cmd
}
