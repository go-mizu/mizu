package comments

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/users"
	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrNotFound = errors.New("comment not found")
)

// Service implements the comments API.
type Service struct {
	store Store
	users users.API
}

// NewService creates a new comments service.
func NewService(store Store, users users.API) *Service {
	return &Service{store: store, users: users}
}

// Create creates a new comment.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Comment, error) {
	now := time.Now()
	comment := &Comment{
		ID:          ulid.New(),
		WorkspaceID: in.WorkspaceID,
		TargetType:  in.TargetType,
		TargetID:    in.TargetID,
		ParentID:    in.ParentID,
		Content:     in.Content,
		AuthorID:    in.AuthorID,
		IsResolved:  false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Create(ctx, comment); err != nil {
		return nil, err
	}

	return s.enrichComment(ctx, comment)
}

// GetByID retrieves a comment by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Comment, error) {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return s.enrichComment(ctx, comment)
}

// Update updates a comment.
func (s *Service) Update(ctx context.Context, id string, content []blocks.RichText) (*Comment, error) {
	if err := s.store.Update(ctx, id, content); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// Delete deletes a comment.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListByTarget lists comments for a specific target.
func (s *Service) ListByTarget(ctx context.Context, workspaceID string, targetType TargetType, targetID string) ([]*Comment, error) {
	comments, err := s.store.ListByTarget(ctx, workspaceID, targetType, targetID)
	if err != nil {
		return nil, err
	}

	// Build tree and enrich
	return s.buildTree(ctx, comments)
}

// ListByPage lists comments for a page (backwards compatibility).
func (s *Service) ListByPage(ctx context.Context, pageID string) ([]*Comment, error) {
	// For backwards compatibility, search without workspace filter
	comments, err := s.store.ListByTarget(ctx, "", TargetPage, pageID)
	if err != nil {
		return nil, err
	}

	// Build tree and enrich
	return s.buildTree(ctx, comments)
}

// ListByBlock lists comments for a block (backwards compatibility).
func (s *Service) ListByBlock(ctx context.Context, blockID string) ([]*Comment, error) {
	comments, err := s.store.ListByTarget(ctx, "", TargetBlock, blockID)
	if err != nil {
		return nil, err
	}
	return s.enrichComments(ctx, comments)
}

// Resolve marks a comment as resolved.
func (s *Service) Resolve(ctx context.Context, id string) error {
	return s.store.SetResolved(ctx, id, true)
}

// Unresolve marks a comment as unresolved.
func (s *Service) Unresolve(ctx context.Context, id string) error {
	return s.store.SetResolved(ctx, id, false)
}

// enrichComment adds user data to a comment.
func (s *Service) enrichComment(ctx context.Context, c *Comment) (*Comment, error) {
	if c.AuthorID != "" {
		author, _ := s.users.GetByID(ctx, c.AuthorID)
		c.Author = author
	}
	return c, nil
}

// enrichComments adds user data to multiple comments.
func (s *Service) enrichComments(ctx context.Context, comments []*Comment) ([]*Comment, error) {
	if len(comments) == 0 {
		return comments, nil
	}

	// Collect author IDs
	authorIDs := make([]string, 0, len(comments))
	for _, c := range comments {
		authorIDs = append(authorIDs, c.AuthorID)
	}

	// Batch fetch users
	usersMap, _ := s.users.GetByIDs(ctx, authorIDs)

	// Attach users
	for _, c := range comments {
		c.Author = usersMap[c.AuthorID]
	}

	return comments, nil
}

// buildTree organizes comments into a tree structure.
func (s *Service) buildTree(ctx context.Context, comments []*Comment) ([]*Comment, error) {
	if len(comments) == 0 {
		return comments, nil
	}

	// Enrich first
	comments, _ = s.enrichComments(ctx, comments)

	// Create map for quick lookup
	commentMap := make(map[string]*Comment)
	for _, c := range comments {
		c.Replies = []*Comment{}
		commentMap[c.ID] = c
	}

	// Build tree
	var roots []*Comment
	for _, c := range comments {
		if c.ParentID == "" {
			roots = append(roots, c)
		} else if parent, ok := commentMap[c.ParentID]; ok {
			parent.Replies = append(parent.Replies, c)
		} else {
			roots = append(roots, c)
		}
	}

	return roots, nil
}
