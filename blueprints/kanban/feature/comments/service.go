package comments

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrNotFound = errors.New("comment not found")
)

// Service implements the comments API.
type Service struct {
	store Store
}

// NewService creates a new comments service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, issueID, authorID string, in *CreateIn) (*Comment, error) {
	comment := &Comment{
		ID:        ulid.New(),
		IssueID:   issueID,
		AuthorID:  authorID,
		Content:   in.Content,
		CreatedAt: time.Now(),
	}

	if err := s.store.Create(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Comment, error) {
	c, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *Service) ListByIssue(ctx context.Context, issueID string) ([]*Comment, error) {
	return s.store.ListByIssue(ctx, issueID)
}

func (s *Service) Update(ctx context.Context, id, content string) (*Comment, error) {
	if err := s.store.Update(ctx, id, content); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
