package bookmarks

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/pkg/ulid"
)

// Service implements the bookmarks API.
type Service struct {
	store Store
}

// NewService creates a new bookmarks service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Add adds a bookmark.
func (s *Service) Add(ctx context.Context, accountID, questionID string) (*Bookmark, error) {
	bookmark := &Bookmark{
		ID:         ulid.New(),
		AccountID:  accountID,
		QuestionID: questionID,
		CreatedAt:  time.Now(),
	}
	if err := s.store.Create(ctx, bookmark); err != nil {
		return nil, err
	}
	return bookmark, nil
}

// Remove removes a bookmark.
func (s *Service) Remove(ctx context.Context, accountID, questionID string) error {
	return s.store.Delete(ctx, accountID, questionID)
}

// ListByAccount lists bookmarks.
func (s *Service) ListByAccount(ctx context.Context, accountID string, limit int) ([]*Bookmark, error) {
	return s.store.ListByAccount(ctx, accountID, limit)
}
