package threads

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/markdown"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/text"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/ulid"
)

// Service implements the threads API.
type Service struct {
	store    Store
	accounts accounts.API
	boards   boards.API
}

// NewService creates a new threads service.
func NewService(store Store, accounts accounts.API, boards boards.API) *Service {
	return &Service{
		store:    store,
		accounts: accounts,
		boards:   boards,
	}
}

// Create creates a new thread.
func (s *Service) Create(ctx context.Context, authorID string, in CreateIn) (*Thread, error) {
	// Validate title
	if len(in.Title) < TitleMinLen || len(in.Title) > TitleMaxLen {
		return nil, errors.New("invalid title length")
	}

	// Validate content
	if len(in.Content) > ContentMaxLen {
		return nil, errors.New("content too long")
	}

	// Validate URL if link post
	if in.Type == ThreadTypeLink && in.URL == "" {
		return nil, errors.New("URL required for link posts")
	}
	if len(in.URL) > URLMaxLen {
		return nil, errors.New("URL too long")
	}

	// Check board exists and is not archived
	board, err := s.boards.GetByID(ctx, in.BoardID)
	if err != nil {
		return nil, err
	}
	if board.IsArchived {
		return nil, ErrBoardLocked
	}

	// Determine thread type
	threadType := in.Type
	if threadType == "" {
		if in.URL != "" {
			threadType = ThreadTypeLink
		} else {
			threadType = ThreadTypeText
		}
	}

	// Render content
	var contentHTML string
	if in.Content != "" {
		html, err := markdown.RenderSafe(in.Content)
		if err == nil {
			contentHTML = html
		}
	}

	// Extract domain from URL
	var domain string
	if in.URL != "" {
		domain = text.ExtractDomain(in.URL)
	}

	now := time.Now()
	createdAt := now
	if in.CreatedAt != nil {
		createdAt = *in.CreatedAt
	}

	// Use initial counts if provided (for seeding), otherwise default to 1 (auto-upvote)
	upvotes := int64(1)
	downvotes := int64(0)
	commentCount := int64(0)
	if in.InitialUpvotes > 0 || in.InitialDownvotes > 0 {
		upvotes = in.InitialUpvotes
		downvotes = in.InitialDownvotes
	}
	if in.InitialComments > 0 {
		commentCount = in.InitialComments
	}
	score := upvotes - downvotes

	thread := &Thread{
		ID:            ulid.New(),
		BoardID:       in.BoardID,
		AuthorID:      authorID,
		Title:         in.Title,
		Content:       in.Content,
		ContentHTML:   contentHTML,
		URL:           in.URL,
		Domain:        domain,
		Type:          threadType,
		Score:         score,
		UpvoteCount:   upvotes,
		DownvoteCount: downvotes,
		CommentCount:  commentCount,
		HotScore:      HotScore(upvotes, downvotes, createdAt),
		IsNSFW:        in.IsNSFW || board.IsNSFW, // Inherit from board
		IsSpoiler:     in.IsSpoiler,
		CreatedAt:     createdAt,
		UpdatedAt:     now,
	}

	if err := s.store.Create(ctx, thread); err != nil {
		return nil, err
	}

	// Update board thread count
	_ = s.boards.IncrementThreadCount(ctx, in.BoardID, 1)

	// Update author karma
	_ = s.accounts.UpdateKarma(ctx, authorID, 1, 0)

	// Load relationships
	thread.Author, _ = s.accounts.GetByID(ctx, authorID)
	thread.Board = board
	thread.IsOwner = true
	thread.CanEdit = true
	thread.CanDelete = true
	thread.Vote = 1

	return thread, nil
}

// GetByID retrieves a thread by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Thread, error) {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load relationships
	thread.Author, _ = s.accounts.GetByID(ctx, thread.AuthorID)
	thread.Board, _ = s.boards.GetByID(ctx, thread.BoardID)

	return thread, nil
}

// GetByIDs retrieves multiple threads by their IDs.
func (s *Service) GetByIDs(ctx context.Context, ids []string) (map[string]*Thread, error) {
	if len(ids) == 0 {
		return make(map[string]*Thread), nil
	}
	return s.store.GetByIDs(ctx, ids)
}

// Update updates a thread.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*Thread, error) {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Content != nil {
		content := *in.Content
		if len(content) > ContentMaxLen {
			content = content[:ContentMaxLen]
		}
		thread.Content = content

		// Re-render content
		html, err := markdown.RenderSafe(content)
		if err == nil {
			thread.ContentHTML = html
		}

		now := time.Now()
		thread.EditedAt = &now
	}

	if in.IsNSFW != nil {
		thread.IsNSFW = *in.IsNSFW
	}
	if in.IsSpoiler != nil {
		thread.IsSpoiler = *in.IsSpoiler
	}

	thread.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, thread); err != nil {
		return nil, err
	}

	return thread, nil
}

// Delete deletes a thread.
func (s *Service) Delete(ctx context.Context, id string) error {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}

	// Update board thread count
	_ = s.boards.IncrementThreadCount(ctx, thread.BoardID, -1)

	return nil
}

// IncrementViews increments the view count.
func (s *Service) IncrementViews(ctx context.Context, id string) error {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	thread.ViewCount++
	return s.store.Update(ctx, thread)
}

// List lists all threads.
func (s *Service) List(ctx context.Context, opts ListOpts) ([]*Thread, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 25
	}
	if opts.SortBy == "" {
		opts.SortBy = SortHot
	}

	threads, err := s.store.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Batch load authors
	s.loadAuthors(ctx, threads)

	return threads, nil
}

// ListByBoard lists threads in a board.
func (s *Service) ListByBoard(ctx context.Context, boardID string, opts ListOpts) ([]*Thread, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 25
	}
	if opts.SortBy == "" {
		opts.SortBy = SortHot
	}

	threads, err := s.store.ListByBoard(ctx, boardID, opts)
	if err != nil {
		return nil, err
	}

	// Batch load authors
	s.loadAuthors(ctx, threads)

	return threads, nil
}

// ListByAuthor lists threads by an author.
func (s *Service) ListByAuthor(ctx context.Context, authorID string, opts ListOpts) ([]*Thread, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 25
	}
	if opts.SortBy == "" {
		opts.SortBy = SortNew
	}

	threads, err := s.store.ListByAuthor(ctx, authorID, opts)
	if err != nil {
		return nil, err
	}

	// Batch load authors
	s.loadAuthors(ctx, threads)

	return threads, nil
}

// Remove removes a thread (moderator action).
func (s *Service) Remove(ctx context.Context, id string, reason string) error {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	thread.IsRemoved = true
	thread.RemoveReason = reason
	thread.UpdatedAt = time.Now()

	return s.store.Update(ctx, thread)
}

// Approve approves a removed thread.
func (s *Service) Approve(ctx context.Context, id string) error {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	thread.IsRemoved = false
	thread.RemoveReason = ""
	thread.UpdatedAt = time.Now()

	return s.store.Update(ctx, thread)
}

// Lock locks a thread.
func (s *Service) Lock(ctx context.Context, id string) error {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	thread.IsLocked = true
	thread.UpdatedAt = time.Now()

	return s.store.Update(ctx, thread)
}

// Unlock unlocks a thread.
func (s *Service) Unlock(ctx context.Context, id string) error {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	thread.IsLocked = false
	thread.UpdatedAt = time.Now()

	return s.store.Update(ctx, thread)
}

// Pin pins a thread.
func (s *Service) Pin(ctx context.Context, id string) error {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	thread.IsPinned = true
	thread.UpdatedAt = time.Now()

	return s.store.Update(ctx, thread)
}

// Unpin unpins a thread.
func (s *Service) Unpin(ctx context.Context, id string) error {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	thread.IsPinned = false
	thread.UpdatedAt = time.Now()

	return s.store.Update(ctx, thread)
}

// SetNSFW sets the NSFW flag.
func (s *Service) SetNSFW(ctx context.Context, id string, nsfw bool) error {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	thread.IsNSFW = nsfw
	thread.UpdatedAt = time.Now()

	return s.store.Update(ctx, thread)
}

// SetSpoiler sets the spoiler flag.
func (s *Service) SetSpoiler(ctx context.Context, id string, spoiler bool) error {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	thread.IsSpoiler = spoiler
	thread.UpdatedAt = time.Now()

	return s.store.Update(ctx, thread)
}

// UpdateVotes updates vote counts.
func (s *Service) UpdateVotes(ctx context.Context, id string, upDelta, downDelta int64) error {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	thread.UpvoteCount += upDelta
	thread.DownvoteCount += downDelta
	thread.Score = thread.UpvoteCount - thread.DownvoteCount
	thread.HotScore = HotScore(thread.UpvoteCount, thread.DownvoteCount, thread.CreatedAt)
	thread.UpdatedAt = time.Now()

	// Update author karma
	karmaDelta := upDelta - downDelta
	if karmaDelta != 0 {
		_ = s.accounts.UpdateKarma(ctx, thread.AuthorID, karmaDelta, 0)
	}

	return s.store.Update(ctx, thread)
}

// IncrementCommentCount updates the comment count.
func (s *Service) IncrementCommentCount(ctx context.Context, id string, delta int64) error {
	thread, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	thread.CommentCount += delta
	if thread.CommentCount < 0 {
		thread.CommentCount = 0
	}

	return s.store.Update(ctx, thread)
}

// EnrichThread enriches a thread with viewer state.
func (s *Service) EnrichThread(ctx context.Context, thread *Thread, viewerID string) error {
	if viewerID == "" {
		return nil
	}

	thread.IsOwner = thread.AuthorID == viewerID
	thread.CanEdit = thread.IsOwner && !thread.IsRemoved && !thread.IsLocked
	thread.CanDelete = thread.IsOwner

	// Vote and bookmark state are set by the handler

	return nil
}

// EnrichThreads enriches multiple threads with viewer state.
func (s *Service) EnrichThreads(ctx context.Context, threads []*Thread, viewerID string) error {
	if viewerID == "" {
		return nil
	}

	for _, thread := range threads {
		if err := s.EnrichThread(ctx, thread, viewerID); err != nil {
			return err
		}
	}
	return nil
}

// RecalculateHotScores recalculates hot scores for all threads.
func (s *Service) RecalculateHotScores(ctx context.Context) error {
	return s.store.UpdateHotScores(ctx)
}

// loadAuthors batch loads authors for threads.
func (s *Service) loadAuthors(ctx context.Context, threads []*Thread) {
	if len(threads) == 0 {
		return
	}

	// Collect unique author IDs
	authorIDs := make([]string, 0, len(threads))
	seen := make(map[string]bool)
	for _, t := range threads {
		if t.AuthorID != "" && !seen[t.AuthorID] {
			authorIDs = append(authorIDs, t.AuthorID)
			seen[t.AuthorID] = true
		}
	}

	// Batch fetch authors
	authors, err := s.accounts.GetByIDs(ctx, authorIDs)
	if err != nil {
		return
	}

	// Assign authors to threads
	for _, t := range threads {
		if author, ok := authors[t.AuthorID]; ok {
			t.Author = author
		}
	}
}
