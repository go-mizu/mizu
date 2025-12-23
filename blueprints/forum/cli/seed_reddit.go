package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/seed"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/seed/reddit"
	"github.com/go-mizu/mizu/blueprints/forum/store/duckdb"
)

var (
	redditSubreddits  string
	redditLimit       int
	redditWithComments bool
	redditCommentDepth int
	redditDryRun      bool
)

// NewSeedReddit creates the seed reddit command.
func NewSeedReddit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reddit",
		Short: "Seed the database with data from Reddit",
		Long: `Fetches real posts and comments from Reddit subreddits and imports them into the forum.

The seeding is idempotent - running it multiple times will not create duplicate data.

Examples:
  forum seed reddit --subreddits golang,programming --limit 25
  forum seed reddit --subreddits golang --limit 10 --with-comments
  forum seed reddit --subreddits golang --dry-run`,
		RunE: runSeedReddit,
	}

	cmd.Flags().StringVarP(&redditSubreddits, "subreddits", "s", "golang,programming", "Comma-separated list of subreddits to seed from")
	cmd.Flags().IntVarP(&redditLimit, "limit", "l", 10, "Number of threads to fetch per subreddit")
	cmd.Flags().BoolVar(&redditWithComments, "with-comments", false, "Also fetch and seed comments")
	cmd.Flags().IntVar(&redditCommentDepth, "comment-depth", 5, "Maximum depth for nested comments")
	cmd.Flags().BoolVar(&redditDryRun, "dry-run", false, "Show what would be seeded without making changes")

	return cmd
}

func runSeedReddit(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconDatabase, "Seeding Forum from Reddit")
	ui.Blank()

	// Parse subreddits
	subs := parseSubreddits(redditSubreddits)
	if len(subs) == 0 {
		return fmt.Errorf("no subreddits specified")
	}

	ui.Info("Subreddits", strings.Join(subs, ", "))
	ui.Info("Thread limit", fmt.Sprintf("%d per subreddit", redditLimit))
	if redditWithComments {
		ui.Info("Comments", fmt.Sprintf("enabled (depth %d)", redditCommentDepth))
	}
	if redditDryRun {
		ui.Warn("DRY RUN - no changes will be made")
	}
	ui.Blank()

	// Open database
	start := time.Now()
	ui.StartSpinner("Opening database...")

	store, err := duckdb.Open(dataDir)
	if err != nil {
		ui.StopSpinnerError("Failed to open database")
		return err
	}
	defer store.Close()

	ui.StopSpinner("Database opened", time.Since(start))

	ctx := context.Background()

	// Create services
	accountsSvc := accounts.NewService(store.Accounts())
	boardsSvc := boards.NewService(store.Boards())
	threadsSvc := threads.NewService(store.Threads(), accountsSvc, boardsSvc)
	commentsSvc := comments.NewService(store.Comments(), accountsSvc, threadsSvc)

	// Create seeder
	seeder := seed.NewSeeder(
		accountsSvc,
		boardsSvc,
		threadsSvc,
		commentsSvc,
		store.SeedMappings(),
	)

	// Create Reddit client
	client := reddit.NewClient()

	// Seed from Reddit
	ui.StartSpinner("Fetching data from Reddit...")

	result, err := seeder.SeedFromSource(ctx, client, seed.SeedOpts{
		Subreddits:   subs,
		ThreadLimit:  redditLimit,
		WithComments: redditWithComments,
		CommentDepth: redditCommentDepth,
		DryRun:       redditDryRun,
		OnProgress: func(msg string) {
			ui.UpdateSpinner(msg)
		},
	})

	ui.StopSpinner("Done fetching", time.Since(start))

	if err != nil {
		ui.Error(fmt.Sprintf("Seeding failed: %v", err))
		return err
	}

	// Show results
	ui.Blank()
	ui.Header(iconCheck, "Seed Results")

	// Boards
	if result.BoardsCreated > 0 {
		ui.Progress(iconBoard, fmt.Sprintf("Boards created: %d", result.BoardsCreated))
	}
	if result.BoardsSkipped > 0 {
		ui.Progress(iconInfo, fmt.Sprintf("Boards skipped (already exist): %d", result.BoardsSkipped))
	}

	// Threads
	if result.ThreadsCreated > 0 {
		ui.Progress(iconThread, fmt.Sprintf("Threads created: %d", result.ThreadsCreated))
	}
	if result.ThreadsSkipped > 0 {
		ui.Progress(iconInfo, fmt.Sprintf("Threads skipped (already exist): %d", result.ThreadsSkipped))
	}

	// Comments
	if redditWithComments {
		if result.CommentsCreated > 0 {
			ui.Progress(iconComment, fmt.Sprintf("Comments created: %d", result.CommentsCreated))
		}
		if result.CommentsSkipped > 0 {
			ui.Progress(iconInfo, fmt.Sprintf("Comments skipped (already exist): %d", result.CommentsSkipped))
		}
	}

	// Users
	if result.UsersCreated > 0 {
		ui.Progress(iconUser, fmt.Sprintf("Users created: %d", result.UsersCreated))
	}

	// Errors
	if len(result.Errors) > 0 {
		ui.Blank()
		ui.Warn(fmt.Sprintf("Encountered %d errors:", len(result.Errors)))
		for i, err := range result.Errors {
			if i >= 5 {
				ui.Warn(fmt.Sprintf("  ... and %d more", len(result.Errors)-5))
				break
			}
			ui.Warn(fmt.Sprintf("  - %v", err))
		}
	}

	ui.Blank()
	if redditDryRun {
		ui.Hint("This was a dry run. Remove --dry-run to apply changes.")
	} else {
		ui.Success("Reddit data seeded successfully!")
		ui.Hint("Run 'forum serve' to start the server and view the data")
	}

	return nil
}

func parseSubreddits(s string) []string {
	var result []string
	for _, sub := range strings.Split(s, ",") {
		sub = strings.TrimSpace(sub)
		sub = strings.TrimPrefix(sub, "r/")
		sub = strings.TrimPrefix(sub, "/r/")
		if sub != "" {
			result = append(result, sub)
		}
	}
	return result
}
