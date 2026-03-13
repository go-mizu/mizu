package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape/discord"
	"github.com/spf13/cobra"
)

func NewDiscord() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discord",
		Short: "Scrape Discord guilds, channels, messages, and users via user token",
		Long: `Scrape Discord entity pages into a local DuckDB database using a Discord user token.

Supports guild, channel, message, and user fetches plus queue-driven graph crawl.
Data is stored in $HOME/data/discord/discord.duckdb.

Set DISCORD_TOKEN env var or use --token flag.

Test server: Reactiflux (guild ID 102860784329052160) — public React/JS community.`,
	}

	cmd.AddCommand(newDiscordMe())
	cmd.AddCommand(newDiscordGuild())
	cmd.AddCommand(newDiscordChannel())
	cmd.AddCommand(newDiscordMessages())
	cmd.AddCommand(newDiscordUser())
	cmd.AddCommand(newDiscordSeed())
	cmd.AddCommand(newDiscordCrawl())
	cmd.AddCommand(newDiscordInfo())
	cmd.AddCommand(newDiscordJobs())
	cmd.AddCommand(newDiscordQueue())

	return cmd
}

// -- shared helpers --------------------------------------------------------

func addDiscordFlags(cmd *cobra.Command, token, dbPath, statePath *string, delay *int) {
	cfg := discord.DefaultConfig()
	cmd.Flags().StringVar(token, "token", cfg.Token, "Discord user token (default: $DISCORD_TOKEN)")
	cmd.Flags().StringVar(dbPath, "db", cfg.DBPath, "Path to discord.duckdb")
	cmd.Flags().StringVar(statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
}

func openDiscordDBs(token, dbPath, statePath string, delay int) (*discord.DB, *discord.State, *discord.Client, error) {
	cfg := discord.DefaultConfig()
	if token != "" {
		cfg.Token = token
	}
	cfg.DBPath = dbPath
	cfg.StatePath = statePath
	cfg.Delay = time.Duration(delay) * time.Millisecond

	if cfg.Token == "" {
		return nil, nil, nil, fmt.Errorf("Discord token required: set DISCORD_TOKEN or --token")
	}

	db, err := discord.OpenDB(dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open db: %w", err)
	}
	stateDB, err := discord.OpenState(statePath)
	if err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("open state: %w", err)
	}
	client := discord.NewClient(cfg)
	return db, stateDB, client, nil
}

// -- me command ------------------------------------------------------------

func newDiscordMe() *cobra.Command {
	var token, dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "me",
		Short: "Fetch /users/@me and enqueue all joined guilds",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openDiscordDBs(token, dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			me, _, err := client.FetchMe(cmd.Context())
			if err != nil {
				return fmt.Errorf("fetch me: %w", err)
			}
			if me != nil {
				u := discord.ParseUser(me)
				_ = db.UpsertUser(u)
				fmt.Printf("Logged in as: %s", u.Username)
				if u.GlobalName != "" && u.GlobalName != u.Username {
					fmt.Printf(" (%s)", u.GlobalName)
				}
				fmt.Println()
			}

			guilds, _, err := client.FetchGuilds(cmd.Context())
			if err != nil {
				return fmt.Errorf("fetch guilds: %w", err)
			}
			enqueued := 0
			for _, g := range guilds {
				id := discord.ParseGuild(g).GuildID
				if id == "" {
					continue
				}
				if err := stateDB.Enqueue(discord.GuildQueueURL(id), discord.EntityGuild, 15); err == nil {
					enqueued++
					fmt.Printf("  enqueued guild: %s  %s\n", id, discord.GuildName(g))
				}
			}
			fmt.Printf("Enqueued %d guilds. Run 'search discord crawl' to start.\n", enqueued)
			return nil
		},
	}

	addDiscordFlags(cmd, &token, &dbPath, &statePath, &delay)
	return cmd
}

// -- guild command ---------------------------------------------------------

func newDiscordGuild() *cobra.Command {
	var token, dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "guild <id>",
		Short: "Fetch a guild and enqueue its text channels",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openDiscordDBs(token, dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			task := &discord.GuildTask{ID: args[0], Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *discord.GuildState) {
				if s.Error != "" {
					fmt.Printf("  [%s] guild=%s error=%s\n", s.Status, s.GuildID, s.Error)
					return
				}
				if s.Name != "" {
					fmt.Printf("  [%s] %s (%s) channels=%d\n", s.Status, s.Name, s.GuildID, s.ChannelsSeen)
				} else {
					fmt.Printf("  [%s] guild=%s\n", s.Status, s.GuildID)
				}
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d channels=%d\n",
				m.Fetched, m.Skipped, m.Failed, m.Channels)
			return nil
		},
	}

	addDiscordFlags(cmd, &token, &dbPath, &statePath, &delay)
	return cmd
}

// -- channel command -------------------------------------------------------

func newDiscordChannel() *cobra.Command {
	var token, dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "channel <id>",
		Short: "Fetch a channel and enqueue its message pages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openDiscordDBs(token, dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			task := &discord.ChannelTask{ID: args[0], Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *discord.ChannelState) {
				if s.Error != "" {
					fmt.Printf("  [%s] channel=%s error=%s\n", s.Status, s.ChannelID, s.Error)
					return
				}
				if s.Name != "" {
					fmt.Printf("  [%s] #%s (%s)\n", s.Status, s.Name, s.ChannelID)
				} else {
					fmt.Printf("  [%s] channel=%s\n", s.Status, s.ChannelID)
				}
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addDiscordFlags(cmd, &token, &dbPath, &statePath, &delay)
	return cmd
}

// -- messages command ------------------------------------------------------

func newDiscordMessages() *cobra.Command {
	var token, dbPath, statePath string
	var delay int
	var before string

	cmd := &cobra.Command{
		Use:   "messages <channel_id>",
		Short: "Fetch a page of messages from a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openDiscordDBs(token, dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			pageURL := discord.MessagePageQueueURL(args[0], before)
			task := &discord.MessagesTask{URL: pageURL, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *discord.MessagesState) {
				if s.Error != "" {
					fmt.Printf("  [%s] channel=%s error=%s\n", s.Status, s.ChannelID, s.Error)
					return
				}
				fmt.Printf("  [%s] channel=%s before=%s count=%d\n",
					s.Status, s.ChannelID, s.Before, s.Count)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: stored=%d skipped=%d failed=%d pages=%d\n",
				m.Stored, m.Skipped, m.Failed, m.Pages)
			return nil
		},
	}

	addDiscordFlags(cmd, &token, &dbPath, &statePath, &delay)
	cmd.Flags().StringVar(&before, "before", "", "Fetch messages before this snowflake ID (empty = latest)")
	return cmd
}

// -- user command ----------------------------------------------------------

func newDiscordUser() *cobra.Command {
	var token, dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "user <id>",
		Short: "Fetch a single user profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openDiscordDBs(token, dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			task := &discord.UserTask{ID: args[0], Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *discord.UserState) {
				if s.Error != "" {
					fmt.Printf("  [%s] user=%s error=%s\n", s.Status, s.UserID, s.Error)
					return
				}
				if s.Username != "" {
					fmt.Printf("  [%s] %s (%s)\n", s.Status, s.Username, s.UserID)
				} else {
					fmt.Printf("  [%s] user=%s\n", s.Status, s.UserID)
				}
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addDiscordFlags(cmd, &token, &dbPath, &statePath, &delay)
	return cmd
}

// -- seed command ----------------------------------------------------------

func newDiscordSeed() *cobra.Command {
	var token, dbPath, statePath string
	var delay, priority int
	var file, entityType string

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the queue from a file (one ID per line)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			cfg := discord.DefaultConfig()
			if token != "" {
				cfg.Token = token
			}
			cfg.DBPath = dbPath
			cfg.StatePath = statePath

			stateDB, err := discord.OpenState(statePath)
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
				ref, err := discord.ParseRef(line, entityType)
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
			fmt.Printf("Enqueued %d discord %s items from %s\n", total, entityType, file)
			return nil
		},
	}

	cfg := discord.DefaultConfig()
	cmd.Flags().StringVar(&token, "token", cfg.Token, "Discord user token")
	cmd.Flags().StringVar(&dbPath, "db", cfg.DBPath, "Path to discord.duckdb")
	cmd.Flags().StringVar(&statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(&delay, "delay", int(cfg.Delay/time.Millisecond), "Delay in ms")
	cmd.Flags().StringVar(&file, "file", "", "File with IDs (one per line)")
	cmd.Flags().StringVar(&entityType, "entity", discord.EntityGuild, "Entity type: guild, channel, user")
	cmd.Flags().IntVar(&priority, "priority", 15, "Queue priority")
	return cmd
}

// -- crawl command ---------------------------------------------------------

func newDiscordCrawl() *cobra.Command {
	var token, dbPath, statePath string
	var delay, workers int

	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Bulk crawl from the queue",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := discord.DefaultConfig()
			if token != "" {
				cfg.Token = token
			}
			cfg.DBPath = dbPath
			cfg.StatePath = statePath
			cfg.Workers = workers
			cfg.Delay = time.Duration(delay) * time.Millisecond

			if cfg.Token == "" {
				return fmt.Errorf("Discord token required: set DISCORD_TOKEN or --token")
			}

			db, err := discord.OpenDB(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			stateDB, err := discord.OpenState(statePath)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			client := discord.NewClient(cfg)
			pending, _, done, failed := stateDB.QueueStats()
			fmt.Printf("Queue: pending=%d done=%d failed=%d\n", pending, done, failed)
			if pending == 0 {
				fmt.Println("Queue is empty. Run 'search discord me' or 'search discord guild <id>' first.")
				return nil
			}

			jobID := fmt.Sprintf("crawl-%d", time.Now().Unix())
			_ = stateDB.CreateJob(jobID, "bulk-crawl", "crawl")
			task := &discord.CrawlTask{Config: cfg, Client: client, DB: db, StateDB: stateDB}
			m, err := task.Run(cmd.Context(), func(s *discord.CrawlState) {
				discord.PrintCrawlProgress(s)
			})
			if err != nil {
				return err
			}
			_ = stateDB.UpdateJob(jobID, "done",
				fmt.Sprintf("done=%d duration=%s", m.Done, m.Duration.Round(time.Second)))
			fmt.Printf("\nCrawl complete: done=%d failed=%d duration=%s\n",
				m.Done, m.Failed, m.Duration.Round(time.Second))
			return nil
		},
	}

	cfg := discord.DefaultConfig()
	cmd.Flags().StringVar(&token, "token", cfg.Token, "Discord user token (default: $DISCORD_TOKEN)")
	cmd.Flags().StringVar(&dbPath, "db", cfg.DBPath, "Path to discord.duckdb")
	cmd.Flags().StringVar(&statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(&workers, "workers", cfg.Workers, "Concurrent fetch workers")
	cmd.Flags().IntVar(&delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
	return cmd
}

// -- info command ----------------------------------------------------------

func newDiscordInfo() *cobra.Command {
	var token, dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show Discord database stats and queue depth",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := discord.OpenDB(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			stateDB, err := discord.OpenState(statePath)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			return discord.PrintStats(db, stateDB)
		},
	}

	addDiscordFlags(cmd, &token, &dbPath, &statePath, &delay)
	return cmd
}

// -- jobs command ----------------------------------------------------------

func newDiscordJobs() *cobra.Command {
	var token, dbPath, statePath string
	var delay, limit int

	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "List recent crawl jobs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDB, err := discord.OpenState(statePath)
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

	addDiscordFlags(cmd, &token, &dbPath, &statePath, &delay)
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of jobs to show")
	return cmd
}

// -- queue command ---------------------------------------------------------

func newDiscordQueue() *cobra.Command {
	var token, dbPath, statePath string
	var delay, limit int
	var status string

	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Inspect queue items",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDB, err := discord.OpenState(statePath)
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

	addDiscordFlags(cmd, &token, &dbPath, &statePath, &delay)
	cmd.Flags().StringVar(&status, "status", "pending", "Filter by status: pending, failed, done, in_progress")
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of items to show")
	return cmd
}
