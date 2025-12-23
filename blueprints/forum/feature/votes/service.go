package votes

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/pkg/ulid"
)

// Service implements the votes API.
type Service struct {
	store         Store
	threadVoter   ThreadVoter
	commentVoter  CommentVoter
}

// NewService creates a new votes service.
func NewService(store Store, threadVoter ThreadVoter, commentVoter CommentVoter) *Service {
	return &Service{
		store:        store,
		threadVoter:  threadVoter,
		commentVoter: commentVoter,
	}
}

// Vote creates or updates a vote.
func (s *Service) Vote(ctx context.Context, accountID, targetType, targetID string, value int) error {
	// Validate value
	if value != -1 && value != 1 {
		return ErrInvalidValue
	}

	// Check for existing vote
	existing, err := s.store.GetByTarget(ctx, accountID, targetType, targetID)
	if err != nil && err != ErrNotFound {
		return err
	}

	if existing != nil {
		// Update existing vote if value changed
		if existing.Value == value {
			return nil // No change
		}

		// Calculate delta
		var upDelta, downDelta int64
		if existing.Value == 1 {
			upDelta = -1 // Remove old upvote
		} else {
			downDelta = -1 // Remove old downvote
		}
		if value == 1 {
			upDelta++ // Add new upvote
		} else {
			downDelta++ // Add new downvote
		}

		// Update vote
		existing.Value = value
		existing.UpdatedAt = time.Now()
		if err := s.store.Update(ctx, existing); err != nil {
			return err
		}

		// Update target counts
		return s.updateTarget(ctx, targetType, targetID, upDelta, downDelta)
	}

	// Create new vote
	vote := &Vote{
		ID:         ulid.New(),
		AccountID:  accountID,
		TargetType: targetType,
		TargetID:   targetID,
		Value:      value,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.store.Create(ctx, vote); err != nil {
		return err
	}

	// Update target counts
	var upDelta, downDelta int64
	if value == 1 {
		upDelta = 1
	} else {
		downDelta = 1
	}
	return s.updateTarget(ctx, targetType, targetID, upDelta, downDelta)
}

// Unvote removes a vote.
func (s *Service) Unvote(ctx context.Context, accountID, targetType, targetID string) error {
	existing, err := s.store.GetByTarget(ctx, accountID, targetType, targetID)
	if err != nil {
		if err == ErrNotFound {
			return nil // Already no vote
		}
		return err
	}

	// Delete vote
	if err := s.store.Delete(ctx, accountID, targetType, targetID); err != nil {
		return err
	}

	// Update target counts
	var upDelta, downDelta int64
	if existing.Value == 1 {
		upDelta = -1
	} else {
		downDelta = -1
	}
	return s.updateTarget(ctx, targetType, targetID, upDelta, downDelta)
}

// GetVote retrieves a user's vote on a target.
func (s *Service) GetVote(ctx context.Context, accountID, targetType, targetID string) (*Vote, error) {
	return s.store.GetByTarget(ctx, accountID, targetType, targetID)
}

// GetVotes retrieves a user's votes on multiple targets.
func (s *Service) GetVotes(ctx context.Context, accountID, targetType string, targetIDs []string) (map[string]int, error) {
	votes, err := s.store.GetByTargets(ctx, accountID, targetType, targetIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int, len(votes))
	for _, v := range votes {
		result[v.TargetID] = v.Value
	}
	return result, nil
}

// GetVoteCounts retrieves vote counts for a target.
func (s *Service) GetVoteCounts(ctx context.Context, targetType, targetID string) (up, down int64, err error) {
	return s.store.CountByTarget(ctx, targetType, targetID)
}

// updateTarget updates the target's vote counts.
func (s *Service) updateTarget(ctx context.Context, targetType, targetID string, upDelta, downDelta int64) error {
	switch targetType {
	case TargetThread:
		if s.threadVoter != nil {
			return s.threadVoter.UpdateVotes(ctx, targetID, upDelta, downDelta)
		}
	case TargetComment:
		if s.commentVoter != nil {
			return s.commentVoter.UpdateVotes(ctx, targetID, upDelta, downDelta)
		}
	}
	return nil
}
