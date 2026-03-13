package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape/spotify"
	"github.com/spf13/cobra"
)

func NewSpotify() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spotify",
		Short: "Scrape public Spotify tracks, albums, artists, and playlists",
		Long: `Scrape public Spotify entity pages into a local DuckDB database.

Supports track, album, artist, and playlist fetches plus queue-driven graph crawl.
Data is stored in $HOME/data/spotify/spotify.duckdb.`,
	}

	cmd.AddCommand(newSpotifyTrack())
	cmd.AddCommand(newSpotifyAlbum())
	cmd.AddCommand(newSpotifyArtist())
	cmd.AddCommand(newSpotifyPlaylist())
	cmd.AddCommand(newSpotifySeed())
	cmd.AddCommand(newSpotifyCrawl())
	cmd.AddCommand(newSpotifyInfo())
	cmd.AddCommand(newSpotifyJobs())
	cmd.AddCommand(newSpotifyQueue())

	return cmd
}

func addSpotifyFlags(cmd *cobra.Command, dbPath, statePath *string, delay *int) {
	cfg := spotify.DefaultConfig()
	cmd.Flags().StringVar(dbPath, "db", cfg.DBPath, "Path to spotify.duckdb")
	cmd.Flags().StringVar(statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
}

func openSpotifyDBs(dbPath, statePath string, delay int) (*spotify.DB, *spotify.State, *spotify.Client, error) {
	cfg := spotify.DefaultConfig()
	cfg.DBPath = dbPath
	cfg.StatePath = statePath
	cfg.Delay = time.Duration(delay) * time.Millisecond

	db, err := spotify.OpenDB(dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open db: %w", err)
	}
	stateDB, err := spotify.OpenState(statePath)
	if err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("open state: %w", err)
	}
	client := spotify.NewClient(cfg)
	return db, stateDB, client, nil
}

func newSpotifyTrack() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "track <id|uri|url>",
		Short: "Fetch a single Spotify track page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openSpotifyDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			task := &spotify.TrackTask{URL: args[0], Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *spotify.TrackState) {
				if s.Error != "" {
					fmt.Printf("  [%s] %s error=%s\n", s.Status, s.URL, s.Error)
					return
				}
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addSpotifyFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

func newSpotifyAlbum() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "album <id|uri|url>",
		Short: "Fetch a single Spotify album page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openSpotifyDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			task := &spotify.AlbumTask{URL: args[0], Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *spotify.AlbumState) {
				if s.Error != "" {
					fmt.Printf("  [%s] %s tracks=%d error=%s\n", s.Status, s.URL, s.TracksSeen, s.Error)
					return
				}
				fmt.Printf("  [%s] %s tracks=%d\n", s.Status, s.URL, s.TracksSeen)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addSpotifyFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

func newSpotifyArtist() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "artist <id|uri|url>",
		Short: "Fetch a single Spotify artist page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openSpotifyDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			task := &spotify.ArtistTask{URL: args[0], Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *spotify.ArtistState) {
				if s.Error != "" {
					fmt.Printf("  [%s] %s albums=%d tracks=%d error=%s\n", s.Status, s.URL, s.AlbumsSeen, s.TracksSeen, s.Error)
					return
				}
				fmt.Printf("  [%s] %s albums=%d tracks=%d\n", s.Status, s.URL, s.AlbumsSeen, s.TracksSeen)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addSpotifyFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

func newSpotifyPlaylist() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "playlist <id|uri|url>",
		Short: "Fetch a single Spotify playlist page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openSpotifyDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			task := &spotify.PlaylistTask{URL: args[0], Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *spotify.PlaylistState) {
				if s.Error != "" {
					fmt.Printf("  [%s] %s tracks=%d error=%s\n", s.Status, s.URL, s.TracksSeen, s.Error)
					return
				}
				fmt.Printf("  [%s] %s tracks=%d\n", s.Status, s.URL, s.TracksSeen)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addSpotifyFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

func newSpotifySeed() *cobra.Command {
	var dbPath, statePath string
	var delay, priority int
	var file, entityType string

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the queue from a file (one id/uri/url per line)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			stateDB, err := spotify.OpenState(statePath)
			if err != nil {
				return fmt.Errorf("open state: %w", err)
			}
			defer stateDB.Close()

			f, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("open file: %w", err)
			}
			defer f.Close()

			total := 0
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				ref, err := spotify.ParseRef(line, entityType)
				if err != nil {
					fmt.Printf("skip %q: %v\n", line, err)
					continue
				}
				if err := stateDB.Enqueue(ref.URL, ref.EntityType, priority); err == nil {
					total++
				}
			}
			if err := scanner.Err(); err != nil {
				return err
			}
			fmt.Printf("Enqueued %d spotify %s items from %s\n", total, entityType, file)
			return nil
		},
	}

	addSpotifyFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().StringVar(&file, "file", "", "File with IDs, URIs, or URLs (one per line)")
	cmd.Flags().StringVar(&entityType, "entity", spotify.EntityArtist, "Entity type: track, album, artist, playlist")
	cmd.Flags().IntVar(&priority, "priority", 10, "Queue priority")
	return cmd
}

func newSpotifyCrawl() *cobra.Command {
	var dbPath, statePath string
	var delay, workers int

	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Bulk crawl from the queue",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := spotify.DefaultConfig()
			cfg.DBPath = dbPath
			cfg.StatePath = statePath
			cfg.Workers = workers
			cfg.Delay = time.Duration(delay) * time.Millisecond

			db, err := spotify.OpenDB(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			stateDB, err := spotify.OpenState(statePath)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			client := spotify.NewClient(cfg)
			pending, _, done, failed := stateDB.QueueStats()
			fmt.Printf("Queue: pending=%d done=%d failed=%d\n", pending, done, failed)
			if pending == 0 {
				fmt.Println("Queue is empty. Run 'search spotify seed' or a direct entity command first.")
				return nil
			}

			jobID := fmt.Sprintf("crawl-%d", time.Now().Unix())
			_ = stateDB.CreateJob(jobID, "bulk-crawl", "crawl")
			task := &spotify.CrawlTask{Config: cfg, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *spotify.CrawlState) {
				spotify.PrintCrawlProgress(s)
			})
			if err != nil {
				return err
			}
			_ = stateDB.UpdateJob(jobID, "done", fmt.Sprintf("done=%d duration=%s", m.Done, m.Duration.Round(time.Second)))
			fmt.Printf("\nCrawl complete: done=%d failed=%d duration=%s\n", m.Done, m.Failed, m.Duration.Round(time.Second))
			return nil
		},
	}

	cfg := spotify.DefaultConfig()
	cmd.Flags().StringVar(&dbPath, "db", cfg.DBPath, "Path to spotify.duckdb")
	cmd.Flags().StringVar(&statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(&workers, "workers", cfg.Workers, "Concurrent fetch workers")
	cmd.Flags().IntVar(&delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
	return cmd
}

func newSpotifyInfo() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show Spotify database stats and queue depth",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := spotify.OpenDB(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			stateDB, err := spotify.OpenState(statePath)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			return spotify.PrintStats(db, stateDB)
		},
	}

	addSpotifyFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

func newSpotifyJobs() *cobra.Command {
	var dbPath, statePath string
	var delay, limit int

	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "List recent crawl jobs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDB, err := spotify.OpenState(statePath)
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
			fmt.Println("── Recent Jobs ──")
			for _, j := range jobs {
				dur := ""
				if !j.CompletedAt.IsZero() {
					dur = j.CompletedAt.Sub(j.StartedAt).Round(time.Second).String()
				}
				fmt.Printf("  [%s] %s  %s  started=%s  duration=%s\n",
					j.Status, j.JobID, j.Name, j.StartedAt.Format("2006-01-02 15:04"), dur)
			}
			return nil
		},
	}

	addSpotifyFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of jobs to show")
	return cmd
}

func newSpotifyQueue() *cobra.Command {
	var dbPath, statePath string
	var delay, limit int
	var status string

	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Inspect queue items",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDB, err := spotify.OpenState(statePath)
			if err != nil {
				return err
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

	addSpotifyFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().StringVar(&status, "status", "pending", "Filter by status: pending, failed, done, in_progress")
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of items to show")
	return cmd
}
