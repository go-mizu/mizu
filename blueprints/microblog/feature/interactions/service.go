package interactions

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrPollExpired   = errors.New("poll has expired")
	ErrAlreadyVoted  = errors.New("already voted")
	ErrMultipleNotAllowed = errors.New("multiple choices not allowed")
)

// Service handles interaction operations.
// Implements API interface.
type Service struct {
	store Store
}

// NewService creates a new interactions service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Like creates a like on a post.
func (s *Service) Like(ctx context.Context, accountID, postID string) error {
	created, err := s.store.Like(ctx, accountID, postID)
	if err != nil {
		return err
	}
	if created {
		_ = s.store.IncrementLikes(ctx, postID)
		s.createNotification(ctx, postID, accountID, "like")
	}
	return nil
}

// Unlike removes a like from a post.
func (s *Service) Unlike(ctx context.Context, accountID, postID string) error {
	deleted, err := s.store.Unlike(ctx, accountID, postID)
	if err != nil {
		return err
	}
	if deleted {
		_ = s.store.DecrementLikes(ctx, postID)
	}
	return nil
}

// Repost creates a repost (boost) of a post.
func (s *Service) Repost(ctx context.Context, accountID, postID string) error {
	created, err := s.store.Repost(ctx, accountID, postID)
	if err != nil {
		return err
	}
	if created {
		_ = s.store.IncrementReposts(ctx, postID)
		s.createNotification(ctx, postID, accountID, "repost")
	}
	return nil
}

// Unrepost removes a repost.
func (s *Service) Unrepost(ctx context.Context, accountID, postID string) error {
	deleted, err := s.store.Unrepost(ctx, accountID, postID)
	if err != nil {
		return err
	}
	if deleted {
		_ = s.store.DecrementReposts(ctx, postID)
	}
	return nil
}

// Bookmark saves a post privately.
func (s *Service) Bookmark(ctx context.Context, accountID, postID string) error {
	return s.store.Bookmark(ctx, accountID, postID)
}

// Unbookmark removes a bookmark.
func (s *Service) Unbookmark(ctx context.Context, accountID, postID string) error {
	return s.store.Unbookmark(ctx, accountID, postID)
}

// VotePoll casts a vote on a poll.
func (s *Service) VotePoll(ctx context.Context, accountID, pollID string, choices []int) error {
	// Check if poll exists and not expired
	expiresAt, multiple, err := s.store.GetPollInfo(ctx, pollID)
	if err != nil {
		return ErrNotFound
	}

	if expiresAt != nil {
		t, _ := time.Parse(time.RFC3339, *expiresAt)
		if time.Now().After(t) {
			return ErrPollExpired
		}
	}

	// Check if already voted
	voted, err := s.store.CheckVoted(ctx, pollID, accountID)
	if err != nil {
		return err
	}
	if voted {
		return ErrAlreadyVoted
	}

	// Validate choices
	if !multiple && len(choices) > 1 {
		return ErrMultipleNotAllowed
	}

	return s.store.VotePoll(ctx, accountID, pollID, choices)
}

// GetLikedBy returns accounts that liked a post.
func (s *Service) GetLikedBy(ctx context.Context, postID string, limit, offset int) ([]string, error) {
	return s.store.GetLikedBy(ctx, postID, limit, offset)
}

// GetRepostedBy returns accounts that reposted a post.
func (s *Service) GetRepostedBy(ctx context.Context, postID string, limit, offset int) ([]string, error) {
	return s.store.GetRepostedBy(ctx, postID, limit, offset)
}

func (s *Service) createNotification(ctx context.Context, postID, actorID, notifType string) {
	ownerID, err := s.store.GetPostOwner(ctx, postID)
	if err != nil || ownerID == actorID {
		return // Don't notify yourself
	}
	_ = s.store.CreateNotification(ctx, ownerID, actorID, notifType, postID)
}
