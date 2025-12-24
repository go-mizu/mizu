package members

import (
	"context"
	"time"
)

// Service implements the members API.
type Service struct {
	store Store
}

// NewService creates a new members service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Join adds a user to a server.
func (s *Service) Join(ctx context.Context, serverID, userID string) (*Member, error) {
	// Check if banned
	banned, err := s.store.IsBanned(ctx, serverID, userID)
	if err != nil {
		return nil, err
	}
	if banned {
		return nil, ErrBanned
	}

	// Check if already member
	exists, err := s.store.IsMember(ctx, serverID, userID)
	if err != nil {
		return nil, err
	}
	if exists {
		return s.store.Get(ctx, serverID, userID)
	}

	member := &Member{
		ServerID: serverID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}

	if err := s.store.Insert(ctx, member); err != nil {
		return nil, err
	}

	return member, nil
}

// Leave removes a user from a server.
func (s *Service) Leave(ctx context.Context, serverID, userID string) error {
	return s.store.Delete(ctx, serverID, userID)
}

// Get retrieves a member.
func (s *Service) Get(ctx context.Context, serverID, userID string) (*Member, error) {
	return s.store.Get(ctx, serverID, userID)
}

// Update updates a member.
func (s *Service) Update(ctx context.Context, serverID, userID string, in *UpdateIn) (*Member, error) {
	if err := s.store.Update(ctx, serverID, userID, in); err != nil {
		return nil, err
	}
	return s.store.Get(ctx, serverID, userID)
}

// Kick removes a member from a server.
func (s *Service) Kick(ctx context.Context, serverID, userID string) error {
	return s.store.Delete(ctx, serverID, userID)
}

// List lists members in a server.
func (s *Service) List(ctx context.Context, serverID string, limit, offset int) ([]*Member, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	return s.store.List(ctx, serverID, limit, offset)
}

// Count counts members in a server.
func (s *Service) Count(ctx context.Context, serverID string) (int, error) {
	return s.store.Count(ctx, serverID)
}

// IsMember checks if a user is a member.
func (s *Service) IsMember(ctx context.Context, serverID, userID string) (bool, error) {
	return s.store.IsMember(ctx, serverID, userID)
}

// Search searches for members.
func (s *Service) Search(ctx context.Context, serverID, query string, limit int) ([]*Member, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	return s.store.Search(ctx, serverID, query, limit)
}

// AddRole adds a role to a member.
func (s *Service) AddRole(ctx context.Context, serverID, userID, roleID string) error {
	return s.store.AddRole(ctx, serverID, userID, roleID)
}

// RemoveRole removes a role from a member.
func (s *Service) RemoveRole(ctx context.Context, serverID, userID, roleID string) error {
	return s.store.RemoveRole(ctx, serverID, userID, roleID)
}

// Ban bans a user from a server.
func (s *Service) Ban(ctx context.Context, serverID, userID, bannedBy, reason string) error {
	return s.store.Ban(ctx, serverID, userID, bannedBy, reason)
}

// Unban removes a ban.
func (s *Service) Unban(ctx context.Context, serverID, userID string) error {
	return s.store.Unban(ctx, serverID, userID)
}

// IsBanned checks if a user is banned.
func (s *Service) IsBanned(ctx context.Context, serverID, userID string) (bool, error) {
	return s.store.IsBanned(ctx, serverID, userID)
}

// ListBans lists bans in a server.
func (s *Service) ListBans(ctx context.Context, serverID string, limit, offset int) ([]*Ban, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.store.ListBans(ctx, serverID, limit, offset)
}
