package seed

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/news/feature/comments"
	"github.com/go-mizu/mizu/blueprints/news/feature/stories"
	"github.com/go-mizu/mizu/blueprints/news/feature/users"
	"github.com/go-mizu/mizu/blueprints/news/pkg/markdown"
	"github.com/go-mizu/mizu/blueprints/news/pkg/ulid"
	"github.com/go-mizu/mizu/blueprints/news/store/duckdb"
)

// Entity types for seed mappings.
const (
	EntityUser    = "user"
	EntityStory   = "story"
	EntityComment = "comment"
)

// SeedOpts contains options for seeding.
type SeedOpts struct {
	StoryLimit   int
	WithComments bool
	CommentDepth int
	DryRun       bool
	OnProgress   func(msg string)
	SortBy       string // "top", "new", "best", "ask", "show"
	SkipExisting bool
}

// SeedResult contains statistics from a seed operation.
type SeedResult struct {
	StoriesCreated  int
	StoriesSkipped  int
	CommentsCreated int
	CommentsSkipped int
	UsersCreated    int
	UsersSkipped    int
	Errors          []error
}

// Seeder handles idempotent seeding from external sources.
type Seeder struct {
	usersStore    *duckdb.UsersStore
	storiesStore  *duckdb.StoriesStore
	commentsStore *duckdb.CommentsStore
	seedMappings  *duckdb.SeedMappingsStore

	// Cache for resolved IDs during a seed run
	userCache map[string]string // external username -> local ID
}

// NewSeeder creates a new seeder.
func NewSeeder(
	usersStore *duckdb.UsersStore,
	storiesStore *duckdb.StoriesStore,
	commentsStore *duckdb.CommentsStore,
	seedMappings *duckdb.SeedMappingsStore,
) *Seeder {
	return &Seeder{
		usersStore:    usersStore,
		storiesStore:  storiesStore,
		commentsStore: commentsStore,
		seedMappings:  seedMappings,
		userCache:     make(map[string]string),
	}
}

// SeedFromSource seeds data from an external source.
func (s *Seeder) SeedFromSource(ctx context.Context, source Source, opts SeedOpts) (*SeedResult, error) {
	result := &SeedResult{}
	sourceName := source.Name()

	// Fetch stories
	s.progress(opts, "Fetching stories from %s...", sourceName)
	storyData, err := source.FetchStories(ctx, FetchOpts{
		Limit:  opts.StoryLimit,
		SortBy: opts.SortBy,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch stories: %w", err)
	}

	// Filter out existing stories if SkipExisting is enabled
	if opts.SkipExisting {
		storyData, _ = s.filterNewStories(ctx, sourceName, storyData)
		s.progress(opts, "Found %d new stories to seed...", len(storyData))
	}

	// Seed stories
	for i, sd := range storyData {
		s.progress(opts, "Seeding story %d/%d...", i+1, len(storyData))

		storyID, created, err := s.seedStory(ctx, sourceName, sd, opts.DryRun)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("story %s: %w", sd.ExternalID, err))
			continue
		}
		if created {
			result.StoriesCreated++
		} else {
			result.StoriesSkipped++
		}

		// Seed comments if requested
		if opts.WithComments && storyID != "" {
			if err := s.seedComments(ctx, source, sourceName, sd.ExternalID, storyID, opts, result); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("comments for %s: %w", sd.ExternalID, err))
			}
		}
	}

	return result, nil
}

func (s *Seeder) seedStory(ctx context.Context, sourceName string, data *StoryData, dryRun bool) (string, bool, error) {
	// Check if already seeded
	if localID, _ := s.seedMappings.GetLocalID(ctx, sourceName, EntityStory, data.ExternalID); localID != "" {
		return localID, false, nil
	}

	if dryRun {
		return "", true, nil
	}

	// Skip deleted authors
	if data.Author == "" {
		return "", false, nil
	}

	// Ensure author exists
	authorID, err := s.ensureUser(ctx, sourceName, data.Author, dryRun)
	if err != nil {
		return "", false, fmt.Errorf("ensure author: %w", err)
	}

	// Create story
	story := &stories.Story{
		ID:           ulid.New(),
		AuthorID:     authorID,
		Title:        data.Title,
		URL:          data.URL,
		Domain:       data.Domain,
		Text:         data.Content,
		Score:        data.Score,
		CommentCount: data.CommentCount,
		CreatedAt:    data.CreatedAt,
	}

	// Render markdown for text posts
	if story.Text != "" {
		story.TextHTML = markdown.RenderPlain(story.Text)
	}

	if err := s.storiesStore.Create(ctx, story, nil); err != nil {
		return "", false, err
	}

	// Create mapping
	mapping := &duckdb.SeedMapping{
		Source:     sourceName,
		EntityType: EntityStory,
		ExternalID: data.ExternalID,
		LocalID:    story.ID,
		CreatedAt:  time.Now(),
	}
	_ = s.seedMappings.Create(ctx, mapping)

	return story.ID, true, nil
}

func (s *Seeder) seedComments(ctx context.Context, source Source, sourceName, storyExternalID, storyLocalID string, opts SeedOpts, result *SeedResult) error {
	// Fetch comments
	commentData, err := source.FetchComments(ctx, storyExternalID)
	if err != nil {
		return err
	}

	// Seed comments recursively
	for _, cd := range commentData {
		s.seedCommentTree(ctx, sourceName, storyLocalID, "", cd, opts, result, 0)
	}

	return nil
}

func (s *Seeder) seedCommentTree(ctx context.Context, sourceName, storyID, parentID string, data *CommentData, opts SeedOpts, result *SeedResult, depth int) {
	if depth > opts.CommentDepth && opts.CommentDepth > 0 {
		return
	}

	commentID, created, err := s.seedComment(ctx, sourceName, storyID, parentID, data, opts.DryRun)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("comment %s: %w", data.ExternalID, err))
		return
	}
	if created {
		result.CommentsCreated++
	} else {
		result.CommentsSkipped++
	}

	// Seed replies
	for _, reply := range data.Replies {
		s.seedCommentTree(ctx, sourceName, storyID, commentID, reply, opts, result, depth+1)
	}
}

func (s *Seeder) seedComment(ctx context.Context, sourceName, storyID, parentID string, data *CommentData, dryRun bool) (string, bool, error) {
	// Check if already seeded
	if localID, _ := s.seedMappings.GetLocalID(ctx, sourceName, EntityComment, data.ExternalID); localID != "" {
		return localID, false, nil
	}

	if dryRun {
		return "", true, nil
	}

	// Skip deleted authors
	if data.Author == "" {
		return "", false, nil
	}

	// Ensure author exists
	authorID, err := s.ensureUser(ctx, sourceName, data.Author, dryRun)
	if err != nil {
		return "", false, fmt.Errorf("ensure author: %w", err)
	}

	commentID := ulid.New()
	path := commentID
	if parentID != "" {
		path = parentID + "/" + commentID
	}

	// Create comment
	comment := &comments.Comment{
		ID:        commentID,
		StoryID:   storyID,
		ParentID:  parentID,
		AuthorID:  authorID,
		Text:      data.Content,
		TextHTML:  markdown.RenderPlain(data.Content),
		Score:     data.Score,
		Depth:     data.Depth,
		Path:      path,
		CreatedAt: data.CreatedAt,
	}

	if err := s.commentsStore.Create(ctx, comment); err != nil {
		return "", false, err
	}

	// Create mapping
	mapping := &duckdb.SeedMapping{
		Source:     sourceName,
		EntityType: EntityComment,
		ExternalID: data.ExternalID,
		LocalID:    comment.ID,
		CreatedAt:  time.Now(),
	}
	_ = s.seedMappings.Create(ctx, mapping)

	return comment.ID, true, nil
}

func (s *Seeder) ensureUser(ctx context.Context, sourceName, username string, dryRun bool) (string, error) {
	// Normalize username
	localUsername := s.localUsername(sourceName, username)

	// Check cache
	if id, ok := s.userCache[localUsername]; ok {
		return id, nil
	}

	// Check mapping
	if localID, _ := s.seedMappings.GetLocalID(ctx, sourceName, EntityUser, username); localID != "" {
		s.userCache[localUsername] = localID
		return localID, nil
	}

	// Check if user exists by username
	if user, _ := s.usersStore.GetByUsername(ctx, localUsername); user != nil {
		s.userCache[localUsername] = user.ID
		// Create mapping
		mapping := &duckdb.SeedMapping{
			Source:     sourceName,
			EntityType: EntityUser,
			ExternalID: username,
			LocalID:    user.ID,
			CreatedAt:  time.Now(),
		}
		_ = s.seedMappings.Create(ctx, mapping)
		return user.ID, nil
	}

	if dryRun {
		return "", nil
	}

	// Create user
	user := &users.User{
		ID:        ulid.New(),
		Username:  localUsername,
		Email:     fmt.Sprintf("%s@%s.seed", localUsername, sourceName),
		Karma:     1,
		CreatedAt: time.Now(),
	}

	if err := s.usersStore.Create(ctx, user); err != nil {
		// Might have been created concurrently
		if existingUser, _ := s.usersStore.GetByUsername(ctx, localUsername); existingUser != nil {
			s.userCache[localUsername] = existingUser.ID
			return existingUser.ID, nil
		}
		return "", err
	}

	s.userCache[localUsername] = user.ID

	// Create mapping
	mapping := &duckdb.SeedMapping{
		Source:     sourceName,
		EntityType: EntityUser,
		ExternalID: username,
		LocalID:    user.ID,
		CreatedAt:  time.Now(),
	}
	_ = s.seedMappings.Create(ctx, mapping)

	return user.ID, nil
}

func (s *Seeder) localUsername(sourceName, username string) string {
	// Sanitize username
	sanitized := strings.ToLower(username)
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")

	// Ensure valid format
	if len(sanitized) < 2 {
		sanitized = sanitized + "_u"
	}
	if len(sanitized) > 15 {
		sanitized = sanitized[:15]
	}

	return sanitized
}

func (s *Seeder) progress(opts SeedOpts, format string, args ...any) {
	if opts.OnProgress != nil {
		opts.OnProgress(fmt.Sprintf(format, args...))
	}
}

// ClearCache clears the internal caches.
func (s *Seeder) ClearCache() {
	s.userCache = make(map[string]string)
}

// filterNewStories returns only stories that haven't been seeded yet.
func (s *Seeder) filterNewStories(ctx context.Context, sourceName string, storyData []*StoryData) ([]*StoryData, error) {
	if len(storyData) == 0 {
		return nil, nil
	}

	extIDs := make([]string, len(storyData))
	for i, sd := range storyData {
		extIDs[i] = sd.ExternalID
	}

	existing, err := s.seedMappings.GetLocalIDs(ctx, sourceName, EntityStory, extIDs)
	if err != nil {
		return storyData, nil // On error, return all stories
	}

	newStories := make([]*StoryData, 0, len(storyData))
	for _, sd := range storyData {
		if _, exists := existing[sd.ExternalID]; !exists {
			newStories = append(newStories, sd)
		}
	}

	return newStories, nil
}
