package comments

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/markdown"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/ulid"
)

// Service implements the comments API.
type Service struct {
	store    Store
	accounts accounts.API
	threads  threads.API
}

// NewService creates a new comments service.
func NewService(store Store, accounts accounts.API, threads threads.API) *Service {
	return &Service{
		store:    store,
		accounts: accounts,
		threads:  threads,
	}
}

// Create creates a new comment.
func (s *Service) Create(ctx context.Context, authorID string, in CreateIn) (*Comment, error) {
	// Validate content
	if in.Content == "" {
		return nil, errors.New("content is required")
	}
	if len(in.Content) > ContentMaxLen {
		return nil, errors.New("content too long")
	}

	// Check thread exists and is not locked
	thread, err := s.threads.GetByID(ctx, in.ThreadID)
	if err != nil {
		return nil, err
	}
	if thread.IsLocked {
		return nil, ErrThreadLocked
	}
	if thread.IsRemoved {
		return nil, ErrThreadRemoved
	}

	// Determine depth and path
	var depth int
	var path string

	if in.ParentID != "" {
		parent, err := s.store.GetByID(ctx, in.ParentID)
		if err != nil {
			return nil, err
		}
		if parent.IsRemoved {
			return nil, ErrCommentRemoved
		}
		if parent.Depth >= MaxDepth {
			return nil, ErrMaxDepth
		}
		depth = parent.Depth + 1
		path = parent.Path
	}

	// Render content
	contentHTML, err := markdown.RenderSafe(in.Content)
	if err != nil {
		contentHTML = in.Content
	}

	now := time.Now()
	createdAt := now
	if in.CreatedAt != nil {
		createdAt = *in.CreatedAt
	}

	id := ulid.New()

	// Build path
	if path == "" {
		path = "/" + id
	} else {
		path = path + "/" + id
	}

	// Use initial counts if provided (for seeding), otherwise default to 1 (auto-upvote)
	upvotes := int64(1)
	downvotes := int64(0)
	if in.InitialUpvotes > 0 || in.InitialDownvotes > 0 {
		upvotes = in.InitialUpvotes
		downvotes = in.InitialDownvotes
	}
	score := upvotes - downvotes

	comment := &Comment{
		ID:            id,
		ThreadID:      in.ThreadID,
		ParentID:      in.ParentID,
		AuthorID:      authorID,
		Content:       in.Content,
		ContentHTML:   contentHTML,
		Score:         score,
		UpvoteCount:   upvotes,
		DownvoteCount: downvotes,
		Depth:         depth,
		Path:          path,
		CreatedAt:     createdAt,
		UpdatedAt:     now,
	}

	if err := s.store.Create(ctx, comment); err != nil {
		return nil, err
	}

	// Update parent child count
	if in.ParentID != "" {
		_ = s.store.IncrementChildCount(ctx, in.ParentID, 1)
	}

	// Update thread comment count
	_ = s.threads.IncrementCommentCount(ctx, in.ThreadID, 1)

	// Update author karma
	_ = s.accounts.UpdateKarma(ctx, authorID, 0, 1)

	// Load author
	comment.Author, _ = s.accounts.GetByID(ctx, authorID)
	comment.IsOwner = true
	comment.CanEdit = true
	comment.CanDelete = true
	comment.Vote = 1

	return comment, nil
}

// GetByID retrieves a comment by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Comment, error) {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load author
	comment.Author, _ = s.accounts.GetByID(ctx, comment.AuthorID)

	return comment, nil
}

// Update updates a comment.
func (s *Service) Update(ctx context.Context, id string, content string) (*Comment, error) {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if comment.IsRemoved {
		return nil, ErrCommentRemoved
	}

	if len(content) > ContentMaxLen {
		content = content[:ContentMaxLen]
	}

	comment.Content = content
	contentHTML, err := markdown.RenderSafe(content)
	if err == nil {
		comment.ContentHTML = contentHTML
	}

	now := time.Now()
	comment.EditedAt = &now
	comment.UpdatedAt = now

	if err := s.store.Update(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// Delete deletes a comment (user delete - content hidden but structure remains).
func (s *Service) Delete(ctx context.Context, id string) error {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// If comment has children, soft delete (preserve structure)
	if comment.ChildCount > 0 {
		comment.IsDeleted = true
		comment.Content = "[deleted]"
		comment.ContentHTML = "<p>[deleted]</p>"
		comment.UpdatedAt = time.Now()
		return s.store.Update(ctx, comment)
	}

	// Otherwise, hard delete
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}

	// Update parent child count
	if comment.ParentID != "" {
		_ = s.store.IncrementChildCount(ctx, comment.ParentID, -1)
	}

	// Update thread comment count
	_ = s.threads.IncrementCommentCount(ctx, comment.ThreadID, -1)

	return nil
}

// ListByThread lists comments for a thread.
func (s *Service) ListByThread(ctx context.Context, threadID string, opts ListOpts) ([]*Comment, error) {
	if opts.Limit <= 0 || opts.Limit > 500 {
		opts.Limit = 100
	}
	if opts.SortBy == "" {
		opts.SortBy = CommentSortBest
	}

	comments, err := s.store.ListByThread(ctx, threadID, opts)
	if err != nil {
		return nil, err
	}

	// Load authors
	for _, c := range comments {
		c.Author, _ = s.accounts.GetByID(ctx, c.AuthorID)
	}

	return comments, nil
}

// ListByParent lists direct children of a comment.
func (s *Service) ListByParent(ctx context.Context, parentID string, opts ListOpts) ([]*Comment, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 25
	}
	if opts.SortBy == "" {
		opts.SortBy = CommentSortBest
	}

	comments, err := s.store.ListByParent(ctx, parentID, opts)
	if err != nil {
		return nil, err
	}

	// Load authors
	for _, c := range comments {
		c.Author, _ = s.accounts.GetByID(ctx, c.AuthorID)
	}

	return comments, nil
}

// ListByAuthor lists comments by an author.
func (s *Service) ListByAuthor(ctx context.Context, authorID string, opts ListOpts) ([]*Comment, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 25
	}
	if opts.SortBy == "" {
		opts.SortBy = CommentSortNew
	}

	comments, err := s.store.ListByAuthor(ctx, authorID, opts)
	if err != nil {
		return nil, err
	}

	// Load authors
	for _, c := range comments {
		c.Author, _ = s.accounts.GetByID(ctx, c.AuthorID)
	}

	return comments, nil
}

// GetTree gets a comment tree for a thread.
func (s *Service) GetTree(ctx context.Context, threadID string, opts TreeOpts) ([]*Comment, error) {
	if opts.Limit <= 0 || opts.Limit > 500 {
		opts.Limit = 200
	}
	if opts.MaxDepth <= 0 || opts.MaxDepth > MaxDepth {
		opts.MaxDepth = MaxDepth
	}
	if opts.CollapseAt <= 0 {
		opts.CollapseAt = DefaultCollapseAt
	}
	if opts.Sort == "" {
		opts.Sort = CommentSortBest
	}

	comments, err := s.store.ListByThread(ctx, threadID, ListOpts{
		Limit:  opts.Limit,
		SortBy: opts.Sort,
	})
	if err != nil {
		return nil, err
	}

	// Load authors
	for _, c := range comments {
		c.Author, _ = s.accounts.GetByID(ctx, c.AuthorID)

		// Mark collapsed
		if c.Depth >= opts.CollapseAt {
			c.IsCollapsed = true
		}
	}

	return s.BuildTree(comments), nil
}

// GetSubtree gets a subtree starting from a comment.
func (s *Service) GetSubtree(ctx context.Context, parentID string, depth int) ([]*Comment, error) {
	if depth <= 0 || depth > MaxDepth {
		depth = 3
	}

	parent, err := s.store.GetByID(ctx, parentID)
	if err != nil {
		return nil, err
	}

	comments, err := s.store.ListByPath(ctx, parent.Path, ListOpts{
		Limit:  100,
		SortBy: CommentSortBest,
	})
	if err != nil {
		return nil, err
	}

	// Filter by depth
	maxDepth := parent.Depth + depth
	var filtered []*Comment
	for _, c := range comments {
		if c.Depth <= maxDepth {
			c.Author, _ = s.accounts.GetByID(ctx, c.AuthorID)
			filtered = append(filtered, c)
		}
	}

	return s.BuildTree(filtered), nil
}

// BuildTree builds a tree structure from flat comments.
func (s *Service) BuildTree(comments []*Comment) []*Comment {
	byID := make(map[string]*Comment, len(comments))
	roots := make([]*Comment, 0)

	// Index comments by ID
	for _, c := range comments {
		c.Children = make([]*Comment, 0)
		byID[c.ID] = c
	}

	// Build tree
	for _, c := range comments {
		if c.ParentID == "" {
			roots = append(roots, c)
		} else if parent, ok := byID[c.ParentID]; ok {
			parent.Children = append(parent.Children, c)
		} else {
			// Parent not in result set, treat as root
			roots = append(roots, c)
		}
	}

	// Sort children at each level
	var sortChildren func([]*Comment)
	sortChildren = func(comments []*Comment) {
		sort.Slice(comments, func(i, j int) bool {
			return WilsonScore(comments[i].UpvoteCount, comments[i].DownvoteCount) >
				WilsonScore(comments[j].UpvoteCount, comments[j].DownvoteCount)
		})
		for _, c := range comments {
			if len(c.Children) > 0 {
				sortChildren(c.Children)
			}
		}
	}
	sortChildren(roots)

	return roots
}

// Remove removes a comment (moderator action).
func (s *Service) Remove(ctx context.Context, id string, reason string) error {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	comment.IsRemoved = true
	comment.RemoveReason = reason
	comment.UpdatedAt = time.Now()

	return s.store.Update(ctx, comment)
}

// Approve approves a removed comment.
func (s *Service) Approve(ctx context.Context, id string) error {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	comment.IsRemoved = false
	comment.RemoveReason = ""
	comment.UpdatedAt = time.Now()

	return s.store.Update(ctx, comment)
}

// UpdateVotes updates vote counts.
func (s *Service) UpdateVotes(ctx context.Context, id string, upDelta, downDelta int64) error {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	comment.UpvoteCount += upDelta
	comment.DownvoteCount += downDelta
	comment.Score = comment.UpvoteCount - comment.DownvoteCount
	comment.UpdatedAt = time.Now()

	// Update author karma
	karmaDelta := upDelta - downDelta
	if karmaDelta != 0 {
		_ = s.accounts.UpdateKarma(ctx, comment.AuthorID, 0, karmaDelta)
	}

	return s.store.Update(ctx, comment)
}

// EnrichComment enriches a comment with viewer state.
func (s *Service) EnrichComment(ctx context.Context, comment *Comment, viewerID string) error {
	if viewerID == "" {
		return nil
	}

	comment.IsOwner = comment.AuthorID == viewerID
	comment.CanEdit = comment.IsOwner && !comment.IsRemoved && !comment.IsDeleted
	comment.CanDelete = comment.IsOwner && !comment.IsRemoved

	return nil
}

// EnrichComments enriches multiple comments with viewer state.
func (s *Service) EnrichComments(ctx context.Context, comments []*Comment, viewerID string) error {
	if viewerID == "" {
		return nil
	}

	var enrich func([]*Comment)
	enrich = func(comments []*Comment) {
		for _, c := range comments {
			_ = s.EnrichComment(ctx, c, viewerID)
			if len(c.Children) > 0 {
				enrich(c.Children)
			}
		}
	}
	enrich(comments)

	return nil
}

// FormatPath creates a materialized path.
func FormatPath(parentPath, id string) string {
	if parentPath == "" {
		return "/" + id
	}
	return fmt.Sprintf("%s/%s", parentPath, id)
}
