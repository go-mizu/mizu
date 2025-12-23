package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/news/feature/comments"
	"github.com/go-mizu/mizu/blueprints/news/feature/stories"
	"github.com/go-mizu/mizu/blueprints/news/feature/tags"
	"github.com/go-mizu/mizu/blueprints/news/feature/users"
	"github.com/go-mizu/mizu/blueprints/news/pkg/markdown"
	"github.com/go-mizu/mizu/blueprints/news/pkg/ulid"
	"github.com/go-mizu/mizu/blueprints/news/store/duckdb"
)

var (
	seedFeed         string
	seedLimit        int
	seedWithComments bool
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

	return cmd
}

// NewSeedSample creates the seed sample command.
func NewSeedSample() *cobra.Command {
	return &cobra.Command{
		Use:   "sample",
		Short: "Seed with sample data",
		Long:  `Creates sample users, tags, stories, and comments for testing.`,
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

	// Create services
	usersStore := store.Users()
	storiesStore := store.Stories()
	tagsStore := store.Tags()
	seedMappings := store.SeedMappings()

	ctx := cmd.Context()

	// Fetch story IDs from HN
	ui.StartSpinner(fmt.Sprintf("Fetching %s stories...", seedFeed))
	start = time.Now()

	storyIDs, err := fetchHNFeed(ctx, seedFeed)
	if err != nil {
		ui.StopSpinnerError("Failed to fetch feed")
		return err
	}

	if len(storyIDs) > seedLimit {
		storyIDs = storyIDs[:seedLimit]
	}

	ui.StopSpinner(fmt.Sprintf("Found %d stories", len(storyIDs)), time.Since(start))

	// Import stories
	var imported, skipped int
	ui.StartSpinner("Importing stories...")

	for i, storyID := range storyIDs {
		ui.UpdateSpinner(fmt.Sprintf("Importing story %d/%d...", i+1, len(storyIDs)))

		// Check if already imported
		externalID := strconv.Itoa(storyID)
		if localID, _ := seedMappings.GetLocalID(ctx, "hn", "story", externalID); localID != "" {
			skipped++
			continue
		}

		// Fetch story from HN
		item, err := fetchHNItem(ctx, storyID)
		if err != nil || item == nil || item.Type != "story" {
			continue
		}

		// Get or create user
		userID, err := getOrCreateHNUser(ctx, usersStore, seedMappings, item.By)
		if err != nil {
			continue
		}

		// Create story
		story := &stories.Story{
			ID:           ulid.New(),
			AuthorID:     userID,
			Title:        item.Title,
			URL:          item.URL,
			Domain:       stories.ExtractDomain(item.URL),
			Text:         item.Text,
			Score:        int64(item.Score),
			CommentCount: int64(item.Descendants),
			CreatedAt:    time.Unix(int64(item.Time), 0),
		}

		if story.Text != "" {
			story.TextHTML = markdown.RenderPlain(story.Text)
		}

		if err := storiesStore.Create(ctx, story, nil); err != nil {
			continue
		}

		// Create mapping
		mapping := &duckdb.SeedMapping{
			Source:     "hn",
			EntityType: "story",
			ExternalID: externalID,
			LocalID:    story.ID,
			CreatedAt:  time.Now(),
		}
		_ = seedMappings.Create(ctx, mapping)

		imported++

		// Import comments if requested
		if seedWithComments && len(item.Kids) > 0 {
			importHNComments(ctx, store, seedMappings, story.ID, item.Kids, "", 0)
		}
	}

	ui.StopSpinner(fmt.Sprintf("Imported %d stories", imported), time.Since(start))

	// Create default tags if they don't exist
	createDefaultTags(ctx, tagsStore)

	ui.Summary([][2]string{
		{"Feed", seedFeed},
		{"Imported", strconv.Itoa(imported)},
		{"Skipped", strconv.Itoa(skipped)},
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

	// Create default tags
	ui.StartSpinner("Creating tags...")
	createDefaultTags(ctx, store.Tags())
	ui.StopSpinner("Tags created", time.Since(start))

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

// HN API types
type hnItem struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	By          string `json:"by"`
	Time        int    `json:"time"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Text        string `json:"text"`
	Score       int    `json:"score"`
	Descendants int    `json:"descendants"`
	Kids        []int  `json:"kids"`
	Parent      int    `json:"parent"`
	Deleted     bool   `json:"deleted"`
	Dead        bool   `json:"dead"`
}

func fetchHNFeed(ctx context.Context, feed string) ([]int, error) {
	feedURL := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/%sstories.json", feed)

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ids []int
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return nil, err
	}

	return ids, nil
}

func fetchHNItem(ctx context.Context, id int) (*hnItem, error) {
	itemURL := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)

	req, err := http.NewRequestWithContext(ctx, "GET", itemURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var item hnItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}

	if item.Deleted || item.Dead {
		return nil, nil
	}

	return &item, nil
}

func getOrCreateHNUser(ctx context.Context, store *duckdb.UsersStore, mappings *duckdb.SeedMappingsStore, username string) (string, error) {
	if username == "" {
		username = "anonymous"
	}

	// Check mapping
	if localID, _ := mappings.GetLocalID(ctx, "hn", "user", username); localID != "" {
		return localID, nil
	}

	// Check if user exists
	if user, _ := store.GetByUsername(ctx, username); user != nil {
		// Create mapping
		mapping := &duckdb.SeedMapping{
			Source:     "hn",
			EntityType: "user",
			ExternalID: username,
			LocalID:    user.ID,
			CreatedAt:  time.Now(),
		}
		_ = mappings.Create(ctx, mapping)
		return user.ID, nil
	}

	// Create new user
	user := &users.User{
		ID:        ulid.New(),
		Username:  username,
		Email:     username + "@hn.example.com",
		Karma:     1,
		CreatedAt: time.Now(),
	}

	if err := store.Create(ctx, user); err != nil {
		return "", err
	}

	// Create mapping
	mapping := &duckdb.SeedMapping{
		Source:     "hn",
		EntityType: "user",
		ExternalID: username,
		LocalID:    user.ID,
		CreatedAt:  time.Now(),
	}
	_ = mappings.Create(ctx, mapping)

	return user.ID, nil
}

func importHNComments(ctx context.Context, store *duckdb.Store, mappings *duckdb.SeedMappingsStore, storyID string, commentIDs []int, parentID string, depth int) {
	if depth > 5 { // Limit depth
		return
	}

	commentsStore := store.Comments()
	usersStore := store.Users()

	for _, commentID := range commentIDs {
		externalID := strconv.Itoa(commentID)
		if localID, _ := mappings.GetLocalID(ctx, "hn", "comment", externalID); localID != "" {
			continue
		}

		item, err := fetchHNItem(ctx, commentID)
		if err != nil || item == nil || item.Type != "comment" {
			continue
		}

		userID, err := getOrCreateHNUser(ctx, usersStore, mappings, item.By)
		if err != nil {
			continue
		}

		commentLocalID := ulid.New()
		path := commentLocalID
		if parentID != "" {
			path = parentID + "/" + commentLocalID
		}

		comment := &comments.Comment{
			ID:        commentLocalID,
			StoryID:   storyID,
			ParentID:  parentID,
			AuthorID:  userID,
			Text:      item.Text,
			TextHTML:  markdown.RenderPlain(item.Text),
			Score:     1,
			Depth:     depth,
			Path:      path,
			CreatedAt: time.Unix(int64(item.Time), 0),
		}

		if err := commentsStore.Create(ctx, comment); err != nil {
			continue
		}

		mapping := &duckdb.SeedMapping{
			Source:     "hn",
			EntityType: "comment",
			ExternalID: externalID,
			LocalID:    comment.ID,
			CreatedAt:  time.Now(),
		}
		_ = mappings.Create(ctx, mapping)

		if len(item.Kids) > 0 {
			importHNComments(ctx, store, mappings, storyID, item.Kids, comment.ID, depth+1)
		}
	}
}

func createDefaultTags(ctx context.Context, store *duckdb.TagsStore) {
	defaultTags := []struct {
		name        string
		description string
		color       string
	}{
		{"programming", "Software development and programming languages", "#3B82F6"},
		{"tech", "Technology news and trends", "#8B5CF6"},
		{"science", "Science and research", "#10B981"},
		{"ask", "Ask the community", "#F59E0B"},
		{"show", "Show your work", "#EF4444"},
		{"meta", "About this site", "#6B7280"},
	}

	for _, t := range defaultTags {
		if existing, _ := store.GetByName(ctx, t.name); existing != nil {
			continue
		}

		tag := &tags.Tag{
			ID:          ulid.New(),
			Name:        t.name,
			Description: t.description,
			Color:       t.color,
		}
		_ = store.Create(ctx, tag)
	}
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

		if err := store.Create(ctx, story, nil); err == nil {
			count++
		}
	}

	return count
}
