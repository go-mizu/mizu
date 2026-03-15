package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape/soundcloud"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func NewSoundcloud() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "soundcloud",
		Short: "Scrape SoundCloud tracks, users, playlists, and search results",
		Long: `Scrape public SoundCloud pages into a local DuckDB database.

The implementation is HTML-first and uses SoundCloud's public page hydration data.

Examples:
  search soundcloud track https://soundcloud.com/forss/flickermood
  search soundcloud user forss
  search soundcloud playlist https://soundcloud.com/forss/sets/soulhack
  search soundcloud search "forss" --type all --limit 25
  search soundcloud crawl --workers 2
  search soundcloud info`,
	}
	cmd.AddCommand(newSoundcloudTrack())
	cmd.AddCommand(newSoundcloudUser())
	cmd.AddCommand(newSoundcloudPlaylist())
	cmd.AddCommand(newSoundcloudSearch())
	cmd.AddCommand(newSoundcloudCrawl())
	cmd.AddCommand(newSoundcloudInfo())
	cmd.AddCommand(newSoundcloudJobs())
	cmd.AddCommand(newSoundcloudQueue())
	return cmd
}

func addSoundcloudFlags(cmd *cobra.Command, dbPath, statePath *string, delay *int) {
	cfg := soundcloud.DefaultConfig()
	cmd.Flags().StringVar(dbPath, "db", cfg.DBPath, "Path to soundcloud.duckdb")
	cmd.Flags().StringVar(statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
}

func openSoundcloudDBs(dbPath, statePath string, delay int) (*soundcloud.DB, *soundcloud.State, *soundcloud.Client, soundcloud.Config, error) {
	cfg := soundcloud.DefaultConfig()
	cfg.DBPath = dbPath
	cfg.StatePath = statePath
	cfg.Delay = time.Duration(delay) * time.Millisecond

	db, err := soundcloud.OpenDB(dbPath)
	if err != nil {
		return nil, nil, nil, cfg, fmt.Errorf("open db: %w", err)
	}
	stateDB, err := soundcloud.OpenState(statePath)
	if err != nil {
		db.Close()
		return nil, nil, nil, cfg, fmt.Errorf("open state: %w", err)
	}
	client := soundcloud.NewClient(cfg)
	return db, stateDB, client, cfg, nil
}

func newSoundcloudTrack() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "track <url|user/track>",
		Short: "Fetch a single SoundCloud track",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openSoundcloudDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := normalizeSoundcloudTrackURL(args[0])
			task := &soundcloud.TrackTask{URL: url, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *soundcloud.TrackState) {
				fmt.Printf("  [%s] %s comments=%d\n", s.Status, s.URL, s.Comments)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addSoundcloudFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

func newSoundcloudUser() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "user <handle|url>",
		Short: "Fetch a single SoundCloud user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openSoundcloudDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := normalizeSCUserURL(args[0])
			task := &soundcloud.UserTask{URL: url, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *soundcloud.UserState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addSoundcloudFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

func newSoundcloudPlaylist() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "playlist <url|user/sets/name>",
		Short: "Fetch a single SoundCloud playlist or album",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openSoundcloudDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := normalizeSoundcloudPlaylistURL(args[0])
			task := &soundcloud.PlaylistTask{URL: url, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *soundcloud.PlaylistState) {
				fmt.Printf("  [%s] %s tracks=%d\n", s.Status, s.URL, s.TracksFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}
	addSoundcloudFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

func newSoundcloudSearch() *cobra.Command {
	var dbPath, statePath string
	var delay, limit int
	var kind string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search SoundCloud and enqueue results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, _, err := openSoundcloudDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			task := &soundcloud.SearchTask{
				Query:   args[0],
				Kind:    kind,
				Limit:   limit,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *soundcloud.SearchState) {
				fmt.Printf("  [%s] query=%q results=%d\n", s.Status, s.Query, s.ResultsFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Enqueued %d results\n", m.Fetched)
			return nil
		},
	}
	addSoundcloudFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().StringVar(&kind, "type", "all", "Search type: all, tracks, playlists, users")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum results to enqueue")
	return cmd
}

func newSoundcloudCrawl() *cobra.Command {
	var dbPath, statePath string
	var delay, workers int
	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Drain the SoundCloud crawl queue",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, cfg, err := openSoundcloudDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			cfg.Workers = workers
			jobID := uuid.NewString()
			_ = stateDB.CreateJob(jobID, "soundcloud crawl", "crawl")
			task := &soundcloud.CrawlTask{Config: cfg, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *soundcloud.CrawlState) {
				fmt.Printf("done=%d pending=%d failed=%d rps=%.2f in_flight=%d\n", s.Done, s.Pending, s.Failed, s.RPS, len(s.InFlight))
			})
			status := "done"
			if err != nil {
				status = "failed"
			}
			_ = stateDB.UpdateJob(jobID, status, fmt.Sprintf(`{"done":%d,"failed":%d,"duration_ms":%d}`, m.Done, m.Failed, m.Duration.Milliseconds()))
			if err != nil {
				return err
			}
			fmt.Printf("Done: processed=%d failed=%d duration=%s\n", m.Done, m.Failed, m.Duration.Round(time.Second))
			return nil
		},
	}
	addSoundcloudFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().IntVar(&workers, "workers", soundcloud.DefaultWorkers, "Number of concurrent workers")
	return cmd
}

func newSoundcloudInfo() *cobra.Command {
	var dbPath, statePath string
	var delay int
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show SoundCloud DB and queue stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, _, _, err := openSoundcloudDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			return soundcloud.PrintStats(db, stateDB)
		},
	}
	addSoundcloudFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

func newSoundcloudJobs() *cobra.Command {
	var statePath, dbPath string
	var delay, limit int
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "List recent SoundCloud crawl jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, stateDB, _, _, err := openSoundcloudDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer stateDB.Close()
			jobs, err := stateDB.ListJobs(limit)
			if err != nil {
				return err
			}
			for _, j := range jobs {
				fmt.Printf("%s  %-8s  %-10s  %s\n", j.StartedAt.Format(time.RFC3339), j.Status, j.Type, j.Name)
			}
			return nil
		},
	}
	addSoundcloudFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of jobs to show")
	return cmd
}

func newSoundcloudQueue() *cobra.Command {
	var statePath, dbPath string
	var delay, limit int
	var status string
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "List queued SoundCloud URLs",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, stateDB, _, _, err := openSoundcloudDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer stateDB.Close()
			items, err := stateDB.ListQueue(status, limit)
			if err != nil {
				return err
			}
			for _, it := range items {
				fmt.Printf("%-10s p=%d %s\n", it.EntityType, it.Priority, it.URL)
			}
			return nil
		},
	}
	addSoundcloudFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().StringVar(&status, "status", "pending", "Queue status: pending, in_progress, done, failed")
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of items to show")
	return cmd
}

func normalizeSoundcloudTrackURL(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	return soundcloud.BaseURL + "/" + strings.TrimLeft(s, "/")
}

func normalizeSCUserURL(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	return soundcloud.BaseURL + "/" + strings.TrimLeft(s, "/")
}

func normalizeSoundcloudPlaylistURL(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	return soundcloud.BaseURL + "/" + strings.TrimLeft(s, "/")
}
