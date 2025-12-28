package activities

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Service implements the activities API
type Service struct {
	store Store
}

// NewService creates a new activities service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Record records a new activity
func (s *Service) Record(ctx context.Context, in *RecordIn) (*Activity, error) {
	if in.ActorID == "" || in.EventType == "" {
		return nil, ErrInvalidInput
	}

	payload := in.Payload
	if payload == "" {
		payload = "{}"
	}

	activity := &Activity{
		ID:         ulid.New(),
		ActorID:    in.ActorID,
		EventType:  in.EventType,
		RepoID:     in.RepoID,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
		Ref:        in.Ref,
		RefType:    in.RefType,
		Payload:    payload,
		IsPublic:   in.IsPublic,
		CreatedAt:  time.Now(),
	}

	if err := s.store.Create(ctx, activity); err != nil {
		return nil, err
	}

	return activity, nil
}

// GetByID retrieves an activity by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Activity, error) {
	activity, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if activity == nil {
		return nil, ErrNotFound
	}
	return activity, nil
}

// ListByUser lists activities for a user
func (s *Service) ListByUser(ctx context.Context, userID string, opts *ListOpts) ([]*Activity, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListByUser(ctx, userID, limit, offset)
}

// ListByRepo lists activities for a repository
func (s *Service) ListByRepo(ctx context.Context, repoID string, opts *ListOpts) ([]*Activity, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListByRepo(ctx, repoID, limit, offset)
}

// ListPublic lists public activities
func (s *Service) ListPublic(ctx context.Context, opts *ListOpts) ([]*Activity, error) {
	limit, offset := s.getPageParams(opts)
	return s.store.ListPublic(ctx, limit, offset)
}

// ListFeed lists activities for a user's feed (from followed users and watched repos)
func (s *Service) ListFeed(ctx context.Context, userID string, opts *ListOpts) ([]*Activity, error) {
	// For now, just return public activities
	// TODO: Implement feed based on followed users and watched repos
	return s.ListPublic(ctx, opts)
}

// Delete deletes an activity
func (s *Service) Delete(ctx context.Context, id string) error {
	activity, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if activity == nil {
		return ErrNotFound
	}
	return s.store.Delete(ctx, id)
}

func (s *Service) getPageParams(opts *ListOpts) (int, int) {
	limit := 30
	offset := 0
	if opts != nil {
		if opts.Limit > 0 && opts.Limit <= 100 {
			limit = opts.Limit
		}
		if opts.Offset >= 0 {
			offset = opts.Offset
		}
	}
	return limit, offset
}
