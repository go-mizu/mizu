package relationships

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/social/pkg/ulid"
)

// Service implements the relationships API.
type Service struct {
	store Store
	// accountChecker is used to check if accounts are private
	isAccountPrivate func(ctx context.Context, id string) (bool, error)
}

// NewService creates a new relationships service.
func NewService(store Store, isAccountPrivate func(ctx context.Context, id string) (bool, error)) *Service {
	return &Service{
		store:            store,
		isAccountPrivate: isAccountPrivate,
	}
}

// Follow follows another user.
func (s *Service) Follow(ctx context.Context, followerID, followingID string) (*Relationship, error) {
	if followerID == followingID {
		return nil, ErrCannotFollowSelf
	}

	// Check if blocked in either direction
	blocked, err := s.store.ExistsBlockEither(ctx, followerID, followingID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrBlocked
	}

	// Check if already following
	exists, err := s.store.ExistsFollow(ctx, followerID, followingID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyFollowing
	}

	// Check if target is private
	pending := false
	if s.isAccountPrivate != nil {
		isPrivate, err := s.isAccountPrivate(ctx, followingID)
		if err != nil {
			return nil, err
		}
		pending = isPrivate
	}

	follow := &Follow{
		ID:          ulid.New(),
		FollowerID:  followerID,
		FollowingID: followingID,
		Pending:     pending,
		CreatedAt:   time.Now(),
	}

	if err := s.store.InsertFollow(ctx, follow); err != nil {
		return nil, err
	}

	return s.GetRelationship(ctx, followerID, followingID)
}

// Unfollow unfollows another user.
func (s *Service) Unfollow(ctx context.Context, followerID, followingID string) (*Relationship, error) {
	if followerID == followingID {
		return nil, ErrCannotFollowSelf
	}

	exists, err := s.store.ExistsFollow(ctx, followerID, followingID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFollowing
	}

	if err := s.store.DeleteFollow(ctx, followerID, followingID); err != nil {
		return nil, err
	}

	return s.GetRelationship(ctx, followerID, followingID)
}

// AcceptFollow accepts a follow request.
func (s *Service) AcceptFollow(ctx context.Context, ownerID, followerID string) error {
	follow, err := s.store.GetFollow(ctx, followerID, ownerID)
	if err != nil {
		return ErrFollowRequestNotFound
	}
	if !follow.Pending {
		return ErrFollowRequestNotFound
	}

	return s.store.SetFollowPending(ctx, followerID, ownerID, false)
}

// RejectFollow rejects a follow request.
func (s *Service) RejectFollow(ctx context.Context, ownerID, followerID string) error {
	follow, err := s.store.GetFollow(ctx, followerID, ownerID)
	if err != nil {
		return ErrFollowRequestNotFound
	}
	if !follow.Pending {
		return ErrFollowRequestNotFound
	}

	return s.store.DeleteFollow(ctx, followerID, ownerID)
}

// GetFollowers returns followers of an account.
func (s *Service) GetFollowers(ctx context.Context, opts FollowersOpts) ([]*Follow, error) {
	return s.store.GetFollowers(ctx, opts.AccountID, opts.Limit, opts.Offset)
}

// GetFollowing returns accounts that an account follows.
func (s *Service) GetFollowing(ctx context.Context, opts FollowersOpts) ([]*Follow, error) {
	return s.store.GetFollowing(ctx, opts.AccountID, opts.Limit, opts.Offset)
}

// GetPendingFollowers returns pending follow requests.
func (s *Service) GetPendingFollowers(ctx context.Context, accountID string, limit, offset int) ([]*Follow, error) {
	return s.store.GetPendingFollowers(ctx, accountID, limit, offset)
}

// Block blocks another user.
func (s *Service) Block(ctx context.Context, accountID, targetID string) (*Relationship, error) {
	if accountID == targetID {
		return nil, ErrCannotBlockSelf
	}

	exists, err := s.store.ExistsBlock(ctx, accountID, targetID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyBlocked
	}

	// Remove any follows in both directions
	_ = s.store.DeleteFollow(ctx, accountID, targetID)
	_ = s.store.DeleteFollow(ctx, targetID, accountID)

	block := &Block{
		ID:        ulid.New(),
		AccountID: accountID,
		TargetID:  targetID,
		CreatedAt: time.Now(),
	}

	if err := s.store.InsertBlock(ctx, block); err != nil {
		return nil, err
	}

	return s.GetRelationship(ctx, accountID, targetID)
}

// Unblock unblocks another user.
func (s *Service) Unblock(ctx context.Context, accountID, targetID string) (*Relationship, error) {
	if accountID == targetID {
		return nil, ErrCannotBlockSelf
	}

	exists, err := s.store.ExistsBlock(ctx, accountID, targetID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotBlocked
	}

	if err := s.store.DeleteBlock(ctx, accountID, targetID); err != nil {
		return nil, err
	}

	return s.GetRelationship(ctx, accountID, targetID)
}

// GetBlocked returns blocked accounts.
func (s *Service) GetBlocked(ctx context.Context, accountID string, limit, offset int) ([]*Block, error) {
	return s.store.GetBlocks(ctx, accountID, limit, offset)
}

// Mute mutes another user.
func (s *Service) Mute(ctx context.Context, accountID, targetID string, in *MuteIn) (*Relationship, error) {
	if accountID == targetID {
		return nil, ErrCannotMuteSelf
	}

	exists, err := s.store.ExistsMute(ctx, accountID, targetID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyMuted
	}

	var expiresAt *time.Time
	if in != nil && in.Duration != "" {
		duration, err := time.ParseDuration(in.Duration)
		if err == nil {
			t := time.Now().Add(duration)
			expiresAt = &t
		}
	}

	hideNotifications := true
	if in != nil {
		hideNotifications = in.HideNotifications
	}

	mute := &Mute{
		ID:                ulid.New(),
		AccountID:         accountID,
		TargetID:          targetID,
		HideNotifications: hideNotifications,
		ExpiresAt:         expiresAt,
		CreatedAt:         time.Now(),
	}

	if err := s.store.InsertMute(ctx, mute); err != nil {
		return nil, err
	}

	return s.GetRelationship(ctx, accountID, targetID)
}

// Unmute unmutes another user.
func (s *Service) Unmute(ctx context.Context, accountID, targetID string) (*Relationship, error) {
	if accountID == targetID {
		return nil, ErrCannotMuteSelf
	}

	exists, err := s.store.ExistsMute(ctx, accountID, targetID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotMuted
	}

	if err := s.store.DeleteMute(ctx, accountID, targetID); err != nil {
		return nil, err
	}

	return s.GetRelationship(ctx, accountID, targetID)
}

// GetMuted returns muted accounts.
func (s *Service) GetMuted(ctx context.Context, accountID string, limit, offset int) ([]*Mute, error) {
	return s.store.GetMutes(ctx, accountID, limit, offset)
}

// GetRelationship returns the relationship between two accounts.
func (s *Service) GetRelationship(ctx context.Context, accountID, targetID string) (*Relationship, error) {
	return s.store.GetRelationship(ctx, accountID, targetID)
}

// GetRelationships returns relationships to multiple accounts.
func (s *Service) GetRelationships(ctx context.Context, accountID string, targetIDs []string) ([]*Relationship, error) {
	rels := make([]*Relationship, 0, len(targetIDs))
	for _, targetID := range targetIDs {
		rel, err := s.store.GetRelationship(ctx, accountID, targetID)
		if err != nil {
			continue
		}
		rel.ID = targetID
		rels = append(rels, rel)
	}
	return rels, nil
}

// IsFollowing checks if one user follows another.
func (s *Service) IsFollowing(ctx context.Context, followerID, followingID string) (bool, error) {
	follow, err := s.store.GetFollow(ctx, followerID, followingID)
	if err != nil {
		return false, nil
	}
	return !follow.Pending, nil
}

// IsBlocked checks if one user blocks another.
func (s *Service) IsBlocked(ctx context.Context, accountID, targetID string) (bool, error) {
	return s.store.ExistsBlock(ctx, accountID, targetID)
}

// IsMuted checks if one user mutes another.
func (s *Service) IsMuted(ctx context.Context, accountID, targetID string) (bool, error) {
	return s.store.ExistsMute(ctx, accountID, targetID)
}
