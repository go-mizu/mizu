package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/news/feature/stories"
	"github.com/go-mizu/mizu/blueprints/news/feature/users"
	"github.com/go-mizu/mizu/blueprints/news/pkg/markdown"
	"github.com/go-mizu/mizu/blueprints/news/pkg/seed"
	"github.com/go-mizu/mizu/blueprints/news/pkg/seed/hn"
	"github.com/go-mizu/mizu/blueprints/news/pkg/ulid"
	"github.com/go-mizu/mizu/blueprints/news/store/duckdb"
)

var (
	seedFeed         string
	seedLimit        int
	seedWithComments bool
	seedCommentDepth int
	seedSkipExisting bool
)

// NewSeed creates the seed command.
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed data from external sources",
		Long:  `Seed the database with content from external sources like Hacker News.`,
	}

	cmd.AddCommand(NewSeedHN())
	cmd.AddCommand(NewSeedSample())

	return cmd
}

// NewSeedHN creates the seed hn command.
func NewSeedHN() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hn",
		Short: "Seed from Hacker News",
		Long: `Imports stories and comments from Hacker News.

Example:
  news seed hn --feed top --limit 30
  news seed hn --feed best --limit 50 --with-comments`,
		RunE: runSeedHN,
	}

	cmd.Flags().StringVar(&seedFeed, "feed", "top", "Feed type: top, new, best, ask, show")
	cmd.Flags().IntVar(&seedLimit, "limit", 30, "Number of stories to fetch")
	cmd.Flags().BoolVar(&seedWithComments, "with-comments", false, "Also fetch comments")
	cmd.Flags().IntVar(&seedCommentDepth, "comment-depth", 5, "Maximum comment depth")
	cmd.Flags().BoolVar(&seedSkipExisting, "skip-existing", true, "Skip already imported items")

	return cmd
}

// NewSeedSample creates the seed sample command.
func NewSeedSample() *cobra.Command {
	return &cobra.Command{
		Use:   "sample",
		Short: "Seed with sample data",
		Long:  `Creates sample users, stories, and comments for testing.`,
		RunE:  runSeedSample,
	}
}

func runSeedHN(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconStory, "Seeding from Hacker News")
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

	ui.StopSpinner("Database ready", time.Since(start))

	ctx := cmd.Context()

	// Create seeder
	seeder := seed.NewSeeder(
		store.Users(),
		store.Stories(),
		store.Comments(),
		store.SeedMappings(),
	)

	// Create HN client
	hnClient := hn.NewClient()

	// Set up progress callback
	opts := seed.SeedOpts{
		StoryLimit:   seedLimit,
		WithComments: seedWithComments,
		CommentDepth: seedCommentDepth,
		SortBy:       seedFeed,
		SkipExisting: seedSkipExisting,
		OnProgress: func(msg string) {
			ui.UpdateSpinner(msg)
		},
	}

	// Seed from HN
	ui.StartSpinner(fmt.Sprintf("Fetching %s stories...", seedFeed))
	start = time.Now()

	result, err := seeder.SeedFromSource(ctx, hnClient, opts)
	if err != nil {
		ui.StopSpinnerError("Seeding failed")
		return err
	}

	ui.StopSpinner("Seeding complete", time.Since(start))

	ui.Summary([][2]string{
		{"Feed", seedFeed},
		{"Stories Created", strconv.Itoa(result.StoriesCreated)},
		{"Stories Skipped", strconv.Itoa(result.StoriesSkipped)},
		{"Comments Created", strconv.Itoa(result.CommentsCreated)},
		{"Comments Skipped", strconv.Itoa(result.CommentsSkipped)},
		{"Errors", strconv.Itoa(len(result.Errors))},
	})

	ui.Success("Seeding complete!")
	ui.Blank()

	return nil
}

func runSeedSample(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconStory, "Seeding Sample Data")
	ui.Blank()

	start := time.Now()
	ui.StartSpinner("Opening database...")

	store, err := duckdb.Open(dataDir)
	if err != nil {
		ui.StopSpinnerError("Failed to open database")
		return err
	}
	defer store.Close()

	ui.StopSpinner("Database ready", time.Since(start))

	ctx := cmd.Context()

	// Create sample users
	ui.StartSpinner("Creating users...")
	userIDs := createSampleUsers(ctx, store.Users())
	ui.StopSpinner(fmt.Sprintf("Created %d users", len(userIDs)), time.Since(start))

	// Create sample stories
	ui.StartSpinner("Creating stories...")
	storyCount := createSampleStories(ctx, store.Stories(), userIDs)
	ui.StopSpinner(fmt.Sprintf("Created %d stories", storyCount), time.Since(start))

	ui.Success("Sample data created!")
	ui.Blank()

	return nil
}

func createSampleUsers(ctx context.Context, store *duckdb.UsersStore) []string {
	sampleUsers := []string{"alice", "bob", "carol", "dave", "eve"}
	var ids []string

	for _, username := range sampleUsers {
		if existing, _ := store.GetByUsername(ctx, username); existing != nil {
			ids = append(ids, existing.ID)
			continue
		}

		user := &users.User{
			ID:        ulid.New(),
			Username:  username,
			Email:     username + "@example.com",
			Karma:     100,
			CreatedAt: time.Now(),
		}
		if err := store.Create(ctx, user); err == nil {
			ids = append(ids, user.ID)
		}
	}

	return ids
}

func createSampleStories(ctx context.Context, store *duckdb.StoriesStore, userIDs []string) int {
	sampleStories := []struct {
		title string
		url   string
	}{
		{"Go 1.22 Released with Enhanced Routing", "https://go.dev/blog/go1.22"},
		{"Why We Chose DuckDB for Our Analytics Stack", "https://duckdb.org"},
		{"The Art of Writing Clean Code", ""},
		{"Show: I built a link aggregator in Go", ""},
		{"Ask: What's your favorite Go library for web development?", ""},
	}

	count := 0
	for i, s := range sampleStories {
		userID := userIDs[i%len(userIDs)]

		story := &stories.Story{
			ID:        ulid.New(),
			AuthorID:  userID,
			Title:     s.title,
			URL:       s.url,
			Domain:    stories.ExtractDomain(s.url),
			Score:     int64(10 + i*5),
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour),
		}

		if s.url == "" {
			story.Text = "This is a sample text post for testing."
			story.TextHTML = markdown.RenderPlain(story.Text)
		}

		if err := store.Create(ctx, story); err == nil {
			count++
		}
	}

	return count
}
