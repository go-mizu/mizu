package votes

import (
	"context"
)

// Service implements the votes.API interface.
type Service struct {
	store Store
}

// NewService creates a new votes service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// GetVote retrieves a user's vote on a target.
func (s *Service) GetVote(ctx context.Context, userID, targetType, targetID string) (*Vote, error) {
	return s.store.GetByUserAndTarget(ctx, userID, targetType, targetID)
}

// GetVotes retrieves a user's votes on multiple targets.
func (s *Service) GetVotes(ctx context.Context, userID, targetType string, targetIDs []string) (map[string]*Vote, error) {
	return s.store.GetByUserAndTargets(ctx, userID, targetType, targetIDs)
}
