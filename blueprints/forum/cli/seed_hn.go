package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/seed"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/seed/hn"
	"github.com/go-mizu/mizu/blueprints/forum/store/duckdb"
)

var (
	hnFeed         string
	hnLimit        int
	hnWithComments bool
	hnCommentDepth int
	hnSkipExisting bool
	hnForce        bool
	hnDryRun       bool
)

// NewSeedHN creates the seed hn command.
func NewSeedHN() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hn",
		Short: "Seed the database with data from Hacker News",
		Long: `Fetches real posts and comments from Hacker News and imports them into the forum.

The seeding is idempotent - running it multiple times will not create duplicate data.

Feed types:
  top   - Top stories (default)
  new   - Newest stories
  best  - Best stories
  ask   - Ask HN stories
  show  - Show HN stories
  jobs  - Job postings

Examples:
  forum seed hn --feed top --limit 25
  forum seed hn --feed new --limit 50 --with-comments
  forum seed hn --feed best --limit 25 --skip-existing
  forum seed hn --feed ask --limit 10 --dry-run`,
		RunE: runSeedHN,
	}

	cmd.Flags().StringVarP(&hnFeed, "feed", "f", "top", "Feed type: top, new, best, ask, show, jobs")
	cmd.Flags().IntVarP(&hnLimit, "limit", "l", 25, "Number of stories to fetch")
	cmd.Flags().BoolVar(&hnWithComments, "with-comments", false, "Also fetch and seed comments")
	cmd.Flags().IntVar(&hnCommentDepth, "comment-depth", 5, "Maximum depth for nested comments")
	cmd.Flags().BoolVar(&hnSkipExisting, "skip-existing", true, "Skip stories that already exist")
	cmd.Flags().BoolVar(&hnForce, "force", false, "Force re-fetch existing items")
	cmd.Flags().BoolVar(&hnDryRun, "dry-run", false, "Show what would be seeded without making changes")

	return cmd
}

func runSeedHN(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconDatabase, "Seeding Forum from Hacker News")
	ui.Blank()

	// Validate feed type
	feedType := hn.FeedType(hnFeed)
	switch feedType {
	case hn.FeedTop, hn.FeedNew, hn.FeedBest, hn.FeedAsk, hn.FeedShow, hn.FeedJobs:
		// Valid
	default:
		return fmt.Errorf("invalid feed type: %s (use: top, new, best, ask, show, jobs)", hnFeed)
	}

	ui.Info("Feed", string(feedType))
	ui.Info("Limit", fmt.Sprintf("%d stories", hnLimit))
	if hnWithComments {
		ui.Info("Comments", fmt.Sprintf("enabled (depth %d)", hnCommentDepth))
	}
	if hnSkipExisting && !hnForce {
		ui.Info("Skip existing", "yes")
	}
	if hnForce {
		ui.Warn("Force mode enabled - will re-fetch existing items")
	}
	if hnDryRun {
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

	// Create HN client
	client := hn.NewClient()

	// Seed from HN
	ui.StartSpinner("Fetching data from Hacker News...")

	result, err := seeder.SeedFromSource(ctx, client, seed.SeedOpts{
		Subreddits:   []string{"hackernews"}, // HN uses a single "board"
		ThreadLimit:  hnLimit,
		WithComments: hnWithComments,
		CommentDepth: hnCommentDepth,
		DryRun:       hnDryRun,
		SortBy:       string(feedType), // Maps to HN feed type
		SkipExisting: hnSkipExisting && !hnForce,
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
		ui.Progress(iconThread, fmt.Sprintf("Stories created: %d", result.ThreadsCreated))
	}
	if result.ThreadsSkipped > 0 {
		ui.Progress(iconInfo, fmt.Sprintf("Stories skipped (already exist): %d", result.ThreadsSkipped))
	}

	// Comments
	if hnWithComments {
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
	if hnDryRun {
		ui.Hint("This was a dry run. Remove --dry-run to apply changes.")
	} else {
		ui.Success("Hacker News data seeded successfully!")
		ui.Hint("Run 'forum serve' to start the server and view the data")
	}

	return nil
}
