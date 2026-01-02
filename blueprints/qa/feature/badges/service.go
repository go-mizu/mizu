package badges

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/pkg/ulid"
)

// Service implements the badges API.
type Service struct {
	store Store
}

// NewService creates a new badges service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new badge.
func (s *Service) Create(ctx context.Context, in Badge) (*Badge, error) {
	badge := &Badge{
		ID:          ulid.New(),
		Name:        in.Name,
		Tier:        in.Tier,
		Description: in.Description,
	}
	if err := s.store.Create(ctx, badge); err != nil {
		return nil, err
	}
	return badge, nil
}

// List lists badges.
func (s *Service) List(ctx context.Context, limit int) ([]*Badge, error) {
	return s.store.List(ctx, limit)
}

// Award grants a badge.
func (s *Service) Award(ctx context.Context, accountID, badgeID string) (*Award, error) {
	award := &Award{
		ID:        ulid.New(),
		AccountID: accountID,
		BadgeID:   badgeID,
		CreatedAt: time.Now(),
	}
	if err := s.store.CreateAward(ctx, award); err != nil {
		return nil, err
	}
	return award, nil
}

// ListAwards lists awards for an account.
func (s *Service) ListAwards(ctx context.Context, accountID string) ([]*Award, error) {
	return s.store.ListAwards(ctx, accountID)
}
