package relationships

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrSelfAction = errors.New("cannot perform action on yourself")
	ErrBlocked    = errors.New("user is blocked")
)

// Service handles relationship operations.
// Implements API interface.
type Service struct {
	store Store
}

// NewService creates a new relationships service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Get returns the relationship between two accounts.
func (s *Service) Get(ctx context.Context, accountID, targetID string) (*Relationship, error) {
	rel := &Relationship{ID: targetID}

	rel.Following, _ = s.store.IsFollowing(ctx, accountID, targetID)
	rel.FollowedBy, _ = s.store.IsFollowing(ctx, targetID, accountID)
	rel.Blocking, _ = s.store.IsBlocking(ctx, accountID, targetID)
	rel.BlockedBy, _ = s.store.IsBlocking(ctx, targetID, accountID)

	muting, hideNotifs, err := s.store.IsMuting(ctx, accountID, targetID)
	if err == nil {
		rel.Muting = muting
		rel.MutingNotifications = hideNotifs
	}

	return rel, nil
}

// Follow creates a follow relationship.
func (s *Service) Follow(ctx context.Context, accountID, targetID string) error {
	if accountID == targetID {
		return ErrSelfAction
	}

	// Check if blocked
	blocked, _ := s.store.IsBlocking(ctx, targetID, accountID)
	if blocked {
		return ErrBlocked
	}

	// Check if already following
	following, _ := s.store.IsFollowing(ctx, accountID, targetID)
	if following {
		return nil // Idempotent
	}

	if err := s.store.Follow(ctx, accountID, targetID); err != nil {
		return err
	}

	_ = s.store.CreateNotification(ctx, targetID, accountID, "follow")
	return nil
}

// Unfollow removes a follow relationship.
func (s *Service) Unfollow(ctx context.Context, accountID, targetID string) error {
	if accountID == targetID {
		return ErrSelfAction
	}
	return s.store.Unfollow(ctx, accountID, targetID)
}

// Block creates a block relationship.
func (s *Service) Block(ctx context.Context, accountID, targetID string) error {
	if accountID == targetID {
		return ErrSelfAction
	}

	// Check if already blocking
	blocking, _ := s.store.IsBlocking(ctx, accountID, targetID)
	if blocking {
		return nil // Idempotent
	}

	if err := s.store.Block(ctx, accountID, targetID); err != nil {
		return err
	}

	// Remove any existing follow relationships (both directions)
	_ = s.store.RemoveFollowsBetween(ctx, accountID, targetID)
	return nil
}

// Unblock removes a block relationship.
func (s *Service) Unblock(ctx context.Context, accountID, targetID string) error {
	if accountID == targetID {
		return ErrSelfAction
	}
	return s.store.Unblock(ctx, accountID, targetID)
}

// Mute creates a mute relationship.
func (s *Service) Mute(ctx context.Context, accountID, targetID string, hideNotifications bool, duration *time.Duration) error {
	if accountID == targetID {
		return ErrSelfAction
	}

	var expiresAt *time.Time
	if duration != nil {
		t := time.Now().Add(*duration)
		expiresAt = &t
	}

	return s.store.Mute(ctx, accountID, targetID, hideNotifications, expiresAt)
}

// Unmute removes a mute relationship.
func (s *Service) Unmute(ctx context.Context, accountID, targetID string) error {
	if accountID == targetID {
		return ErrSelfAction
	}
	return s.store.Unmute(ctx, accountID, targetID)
}

// GetFollowers returns accounts following the target.
func (s *Service) GetFollowers(ctx context.Context, targetID string, limit, offset int) ([]string, error) {
	return s.store.GetFollowers(ctx, targetID, limit, offset)
}

// GetFollowing returns accounts the target is following.
func (s *Service) GetFollowing(ctx context.Context, targetID string, limit, offset int) ([]string, error) {
	return s.store.GetFollowing(ctx, targetID, limit, offset)
}

// CountFollowers returns the number of followers for an account.
func (s *Service) CountFollowers(ctx context.Context, accountID string) (int, error) {
	return s.store.CountFollowers(ctx, accountID)
}

// CountFollowing returns the number of accounts an account is following.
func (s *Service) CountFollowing(ctx context.Context, accountID string) (int, error) {
	return s.store.CountFollowing(ctx, accountID)
}

// GetBlocked returns blocked account IDs.
func (s *Service) GetBlocked(ctx context.Context, accountID string, limit, offset int) ([]string, error) {
	return s.store.GetBlocked(ctx, accountID, limit, offset)
}

// GetMuted returns muted account IDs.
func (s *Service) GetMuted(ctx context.Context, accountID string, limit, offset int) ([]string, error) {
	return s.store.GetMuted(ctx, accountID, limit, offset)
}

// IsBlocked checks if targetID is blocked by accountID.
func (s *Service) IsBlocked(ctx context.Context, accountID, targetID string) (bool, error) {
	return s.store.IsBlocking(ctx, accountID, targetID)
}

// IsMuted checks if targetID is muted by accountID.
func (s *Service) IsMuted(ctx context.Context, accountID, targetID string) (bool, error) {
	muting, _, err := s.store.IsMuting(ctx, accountID, targetID)
	return muting, err
}
