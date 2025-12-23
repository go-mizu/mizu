package seed

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/ulid"
	"github.com/go-mizu/mizu/blueprints/forum/store"
)

// Entity types for seed mappings.
const (
	EntityAccount = "account"
	EntityBoard   = "board"
	EntityThread  = "thread"
	EntityComment = "comment"
)

// SeedOpts contains options for seeding.
type SeedOpts struct {
	Subreddits    []string
	ThreadLimit   int
	WithComments  bool
	CommentDepth  int
	DryRun        bool
	OnProgress    func(msg string)
	SortBy        string // Sort order for fetching threads (e.g., "hot", "new", "top" for Reddit; "top", "new", "best" for HN)
	TimeRange     string // Time range for "top" sort (Reddit only)
	SkipExisting  bool   // Skip items that already exist
}

// SeedResult contains statistics from a seed operation.
type SeedResult struct {
	BoardsCreated   int
	BoardsSkipped   int
	ThreadsCreated  int
	ThreadsSkipped  int
	CommentsCreated int
	CommentsSkipped int
	UsersCreated    int
	UsersSkipped    int
	Errors          []error
}

// Seeder handles idempotent seeding from external sources.
type Seeder struct {
	accounts     accounts.API
	boards       boards.API
	threads      threads.API
	comments     comments.API
	seedMappings store.SeedMappingsStore

	// Cache for resolved IDs during a seed run
	userCache  map[string]string // external username -> local ID
	boardCache map[string]string // board name -> local ID
}

// NewSeeder creates a new seeder.
func NewSeeder(
	accountsAPI accounts.API,
	boardsAPI boards.API,
	threadsAPI threads.API,
	commentsAPI comments.API,
	seedMappings store.SeedMappingsStore,
) *Seeder {
	return &Seeder{
		accounts:     accountsAPI,
		boards:       boardsAPI,
		threads:      threadsAPI,
		comments:     commentsAPI,
		seedMappings: seedMappings,
		userCache:    make(map[string]string),
		boardCache:   make(map[string]string),
	}
}

// SeedFromSource seeds data from an external source.
func (s *Seeder) SeedFromSource(ctx context.Context, source Source, opts SeedOpts) (*SeedResult, error) {
	result := &SeedResult{}
	sourceName := source.Name()

	for _, subreddit := range opts.Subreddits {
		if err := s.seedSubreddit(ctx, source, sourceName, subreddit, opts, result); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("subreddit %s: %w", subreddit, err))
		}
	}

	return result, nil
}

func (s *Seeder) seedSubreddit(ctx context.Context, source Source, sourceName, subreddit string, opts SeedOpts, result *SeedResult) error {
	s.progress(opts, "Fetching subreddit %s...", subreddit)

	// Fetch subreddit metadata
	subData, err := source.FetchSubreddit(ctx, subreddit)
	if err != nil {
		return fmt.Errorf("fetch subreddit: %w", err)
	}

	// Create or get board
	boardID, created, err := s.ensureBoard(ctx, sourceName, subData, opts.DryRun)
	if err != nil {
		return fmt.Errorf("ensure board: %w", err)
	}
	if created {
		result.BoardsCreated++
	} else {
		result.BoardsSkipped++
	}

	// Fetch threads
	s.progress(opts, "Fetching threads from %s...", subreddit)
	threadData, err := source.FetchThreads(ctx, subreddit, FetchOpts{
		Limit:     opts.ThreadLimit,
		SortBy:    opts.SortBy,
		TimeRange: opts.TimeRange,
	})
	if err != nil {
		return fmt.Errorf("fetch threads: %w", err)
	}

	// Filter out existing threads if SkipExisting is enabled
	if opts.SkipExisting {
		threadData, _ = s.FilterNewThreads(ctx, sourceName, threadData)
		s.progress(opts, "Found %d new threads to seed...", len(threadData))
	}

	// Seed threads
	for _, td := range threadData {
		threadID, created, err := s.seedThread(ctx, sourceName, boardID, td, opts.DryRun)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("thread %s: %w", td.ExternalID, err))
			continue
		}
		if created {
			result.ThreadsCreated++
			result.UsersCreated++ // Author was potentially created
		} else {
			result.ThreadsSkipped++
		}

		// Seed comments if requested
		if opts.WithComments && threadID != "" {
			if err := s.seedComments(ctx, source, sourceName, subreddit, td.ExternalID, threadID, opts, result); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("comments for %s: %w", td.ExternalID, err))
			}
		}
	}

	return nil
}

func (s *Seeder) ensureBoard(ctx context.Context, sourceName string, data *SubredditData, dryRun bool) (string, bool, error) {
	// Check cache
	if id, ok := s.boardCache[data.Name]; ok {
		return id, false, nil
	}

	// Check if board exists by name
	board, err := s.boards.GetByName(ctx, data.Name)
	if err != nil && err != boards.ErrNotFound {
		return "", false, err
	}
	if board != nil {
		s.boardCache[data.Name] = board.ID
		return board.ID, false, nil
	}

	if dryRun {
		return "", true, nil
	}

	// Create a system user for the board if needed
	systemUserID, _, err := s.ensureUser(ctx, sourceName, "system", false)
	if err != nil {
		return "", false, fmt.Errorf("ensure system user: %w", err)
	}

	// Create board
	board, err = s.boards.Create(ctx, systemUserID, boards.CreateIn{
		Name:        data.Name,
		Title:       data.Title,
		Description: data.Description,
		MemberCount: data.Subscribers,
	})
	if err != nil {
		// Board might have been created concurrently
		if existingBoard, _ := s.boards.GetByName(ctx, data.Name); existingBoard != nil {
			s.boardCache[data.Name] = existingBoard.ID
			return existingBoard.ID, false, nil
		}
		return "", false, err
	}

	s.boardCache[data.Name] = board.ID
	return board.ID, true, nil
}

func (s *Seeder) seedThread(ctx context.Context, sourceName, boardID string, data *ThreadData, dryRun bool) (string, bool, error) {
	// Check if already seeded
	exists, err := s.seedMappings.Exists(ctx, sourceName, EntityThread, data.ExternalID)
	if err != nil {
		return "", false, err
	}
	if exists {
		// Get the local ID
		mapping, _ := s.seedMappings.GetByExternalID(ctx, sourceName, EntityThread, data.ExternalID)
		if mapping != nil {
			return mapping.LocalID, false, nil
		}
		return "", false, nil
	}

	if dryRun {
		return "", true, nil
	}

	// Skip deleted authors
	if data.Author == "[deleted]" || data.Author == "" {
		return "", false, nil
	}

	// Ensure author exists
	authorID, _, err := s.ensureUser(ctx, sourceName, data.Author, false)
	if err != nil {
		return "", false, fmt.Errorf("ensure author: %w", err)
	}

	// Determine thread type
	threadType := threads.ThreadTypeText
	if !data.IsSelf && data.URL != "" {
		threadType = threads.ThreadTypeLink
	}

	// Create thread with original vote counts and timestamp
	createdAt := data.CreatedAt
	thread, err := s.threads.Create(ctx, authorID, threads.CreateIn{
		BoardID:          boardID,
		Title:            data.Title,
		Content:          data.Content,
		URL:              data.URL,
		Type:             threadType,
		IsNSFW:           data.IsNSFW,
		IsSpoiler:        data.IsSpoiler,
		InitialUpvotes:   data.UpvoteCount,
		InitialDownvotes: data.DownvoteCount,
		InitialComments:  data.CommentCount,
		CreatedAt:        &createdAt,
	})
	if err != nil {
		return "", false, err
	}

	// Create mapping
	if err := s.seedMappings.Create(ctx, &store.SeedMapping{
		Source:     sourceName,
		EntityType: EntityThread,
		ExternalID: data.ExternalID,
		LocalID:    thread.ID,
	}); err != nil {
		// Non-fatal, continue anyway
	}

	return thread.ID, true, nil
}

func (s *Seeder) seedComments(ctx context.Context, source Source, sourceName, subreddit, threadExternalID, threadLocalID string, opts SeedOpts, result *SeedResult) error {
	// Fetch comments
	commentData, err := source.FetchComments(ctx, subreddit, threadExternalID)
	if err != nil {
		return err
	}

	// Seed comments recursively
	for _, cd := range commentData {
		s.seedCommentTree(ctx, sourceName, threadLocalID, "", cd, opts, result, 0)
	}

	return nil
}

func (s *Seeder) seedCommentTree(ctx context.Context, sourceName, threadID, parentID string, data *CommentData, opts SeedOpts, result *SeedResult, depth int) {
	if depth > opts.CommentDepth && opts.CommentDepth > 0 {
		return
	}

	commentID, created, err := s.seedComment(ctx, sourceName, threadID, parentID, data, opts.DryRun)
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
		s.seedCommentTree(ctx, sourceName, threadID, commentID, reply, opts, result, depth+1)
	}
}

func (s *Seeder) seedComment(ctx context.Context, sourceName, threadID, parentID string, data *CommentData, dryRun bool) (string, bool, error) {
	// Check if already seeded
	exists, err := s.seedMappings.Exists(ctx, sourceName, EntityComment, data.ExternalID)
	if err != nil {
		return "", false, err
	}
	if exists {
		mapping, _ := s.seedMappings.GetByExternalID(ctx, sourceName, EntityComment, data.ExternalID)
		if mapping != nil {
			return mapping.LocalID, false, nil
		}
		return "", false, nil
	}

	if dryRun {
		return "", true, nil
	}

	// Skip deleted authors
	if data.Author == "[deleted]" || data.Author == "" {
		return "", false, nil
	}

	// Ensure author exists
	authorID, _, err := s.ensureUser(ctx, sourceName, data.Author, false)
	if err != nil {
		return "", false, fmt.Errorf("ensure author: %w", err)
	}

	// Create comment with original vote counts and timestamp
	createdAt := data.CreatedAt
	comment, err := s.comments.Create(ctx, authorID, comments.CreateIn{
		ThreadID:         threadID,
		ParentID:         parentID,
		Content:          data.Content,
		InitialUpvotes:   data.UpvoteCount,
		InitialDownvotes: data.DownvoteCount,
		CreatedAt:        &createdAt,
	})
	if err != nil {
		return "", false, err
	}

	// Create mapping
	if err := s.seedMappings.Create(ctx, &store.SeedMapping{
		Source:     sourceName,
		EntityType: EntityComment,
		ExternalID: data.ExternalID,
		LocalID:    comment.ID,
	}); err != nil {
		// Non-fatal
	}

	return comment.ID, true, nil
}

func (s *Seeder) ensureUser(ctx context.Context, sourceName, username string, dryRun bool) (string, bool, error) {
	// Normalize username
	localUsername := s.localUsername(sourceName, username)

	// Check cache
	if id, ok := s.userCache[localUsername]; ok {
		return id, false, nil
	}

	// Check if user exists
	account, err := s.accounts.GetByUsername(ctx, localUsername)
	if err != nil && err != accounts.ErrNotFound {
		return "", false, err
	}
	if account != nil {
		s.userCache[localUsername] = account.ID
		return account.ID, false, nil
	}

	if dryRun {
		return "", true, nil
	}

	// Create user
	account, err = s.accounts.Create(ctx, accounts.CreateIn{
		Username: localUsername,
		Email:    fmt.Sprintf("%s@%s.seed", localUsername, sourceName),
		Password: ulid.New(), // Random password
	})
	if err != nil {
		// Might have been created concurrently
		if existingAccount, _ := s.accounts.GetByUsername(ctx, localUsername); existingAccount != nil {
			s.userCache[localUsername] = existingAccount.ID
			return existingAccount.ID, false, nil
		}
		return "", false, err
	}

	s.userCache[localUsername] = account.ID
	return account.ID, true, nil
}

func (s *Seeder) localUsername(sourceName, username string) string {
	// Sanitize username
	sanitized := strings.ToLower(username)
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")

	// Determine prefix based on source
	prefix := "r_" // Default for reddit
	switch sourceName {
	case "reddit":
		prefix = "r_"
	case "hn":
		prefix = "hn_"
	default:
		prefix = sourceName[:min(3, len(sourceName))] + "_"
	}

	// Ensure valid format
	if len(sanitized) < 3 {
		sanitized = sanitized + "_u"
	}
	maxLen := 20 - len(prefix) // Leave room for prefix
	if len(sanitized) > maxLen {
		sanitized = sanitized[:maxLen]
	}

	return prefix + sanitized
}

func (s *Seeder) progress(opts SeedOpts, format string, args ...any) {
	if opts.OnProgress != nil {
		opts.OnProgress(fmt.Sprintf(format, args...))
	}
}

// ClearCache clears the internal caches.
func (s *Seeder) ClearCache() {
	s.userCache = make(map[string]string)
	s.boardCache = make(map[string]string)
}

// CheckExistsBatch checks which external IDs already exist in seed_mappings.
// Returns a map of externalID -> localID for existing items.
func (s *Seeder) CheckExistsBatch(ctx context.Context, sourceName, entityType string, externalIDs []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, extID := range externalIDs {
		mapping, err := s.seedMappings.GetByExternalID(ctx, sourceName, entityType, extID)
		if err != nil {
			continue // Ignore errors, treat as not found
		}
		if mapping != nil {
			result[extID] = mapping.LocalID
		}
	}
	return result, nil
}

// FilterNewThreads returns only threads that haven't been seeded yet.
func (s *Seeder) FilterNewThreads(ctx context.Context, sourceName string, threads []*ThreadData) ([]*ThreadData, error) {
	if len(threads) == 0 {
		return nil, nil
	}

	extIDs := make([]string, len(threads))
	for i, t := range threads {
		extIDs[i] = t.ExternalID
	}

	existing, err := s.CheckExistsBatch(ctx, sourceName, EntityThread, extIDs)
	if err != nil {
		return threads, nil // On error, return all threads
	}

	newThreads := make([]*ThreadData, 0, len(threads))
	for _, t := range threads {
		if _, exists := existing[t.ExternalID]; !exists {
			newThreads = append(newThreads, t)
		}
	}

	return newThreads, nil
}
