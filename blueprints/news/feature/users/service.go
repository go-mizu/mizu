package users

import (
	"context"
)

// Service implements the users.API interface.
type Service struct {
	store Store
}

// NewService creates a new users service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// GetByID retrieves a user by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	return s.store.GetByID(ctx, id)
}

// GetByIDs retrieves multiple users by their IDs.
func (s *Service) GetByIDs(ctx context.Context, ids []string) (map[string]*User, error) {
	return s.store.GetByIDs(ctx, ids)
}

// GetByUsername retrieves a user by username.
func (s *Service) GetByUsername(ctx context.Context, username string) (*User, error) {
	return s.store.GetByUsername(ctx, username)
}

// GetSession retrieves a session by token.
func (s *Service) GetSession(ctx context.Context, token string) (*Session, error) {
	session, err := s.store.GetSessionByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if session.IsExpired() {
		return nil, ErrSessionExpired
	}

	return session, nil
}

// List lists users.
func (s *Service) List(ctx context.Context, limit, offset int) ([]*User, error) {
	if limit <= 0 {
		limit = 30
	}
	return s.store.List(ctx, limit, offset)
}
