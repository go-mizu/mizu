package votes

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/news/pkg/ulid"
)

// Service implements the votes.API interface.
type Service struct {
	store Store
}

// NewService creates a new votes service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Vote creates a vote.
func (s *Service) Vote(ctx context.Context, userID string, in VoteIn) error {
	if err := in.Validate(); err != nil {
		return err
	}

	// Check if already voted
	if existing, _ := s.store.GetByUserAndTarget(ctx, userID, in.TargetType, in.TargetID); existing != nil {
		return ErrAlreadyVoted
	}

	vote := &Vote{
		ID:         ulid.New(),
		UserID:     userID,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
		Value:      1, // Only upvotes for now
		CreatedAt:  time.Now(),
	}

	return s.store.Create(ctx, vote)
}

// Unvote removes a vote.
func (s *Service) Unvote(ctx context.Context, userID, targetType, targetID string) error {
	return s.store.Delete(ctx, userID, targetType, targetID)
}

// GetVote retrieves a user's vote on a target.
func (s *Service) GetVote(ctx context.Context, userID, targetType, targetID string) (*Vote, error) {
	return s.store.GetByUserAndTarget(ctx, userID, targetType, targetID)
}

// GetVotes retrieves a user's votes on multiple targets.
func (s *Service) GetVotes(ctx context.Context, userID, targetType string, targetIDs []string) (map[string]*Vote, error) {
	return s.store.GetByUserAndTargets(ctx, userID, targetType, targetIDs)
}
