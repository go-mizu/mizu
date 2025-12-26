package assignees

import (
	"context"
	"errors"
)

var (
	ErrAlreadyAssigned = errors.New("user already assigned")
	ErrNotAssigned     = errors.New("user not assigned")
)

// Service implements the assignees API.
type Service struct {
	store Store
}

// NewService creates a new assignees service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Add(ctx context.Context, issueID, userID string) error {
	// Check if already assigned
	assigned, err := s.store.List(ctx, issueID)
	if err != nil {
		return err
	}
	for _, id := range assigned {
		if id == userID {
			return ErrAlreadyAssigned
		}
	}

	return s.store.Add(ctx, issueID, userID)
}

func (s *Service) Remove(ctx context.Context, issueID, userID string) error {
	return s.store.Remove(ctx, issueID, userID)
}

func (s *Service) List(ctx context.Context, issueID string) ([]string, error) {
	return s.store.List(ctx, issueID)
}

func (s *Service) ListByUser(ctx context.Context, userID string) ([]string, error) {
	return s.store.ListByUser(ctx, userID)
}
