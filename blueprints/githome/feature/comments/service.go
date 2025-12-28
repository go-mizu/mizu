package comments

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Service implements the comments API
type Service struct {
	store Store
}

// NewService creates a new comments service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new comment
func (s *Service) Create(ctx context.Context, targetType, targetID, userID string, in *CreateIn) (*Comment, error) {
	body := strings.TrimSpace(in.Body)
	if body == "" {
		return nil, ErrMissingBody
	}

	now := time.Now()
	comment := &Comment{
		ID:         ulid.New(),
		TargetType: targetType,
		TargetID:   targetID,
		UserID:     userID,
		Body:       body,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.store.Create(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// GetByID retrieves a comment by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Comment, error) {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, ErrNotFound
	}
	return comment, nil
}

// Update updates a comment
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Comment, error) {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, ErrNotFound
	}

	body := strings.TrimSpace(in.Body)
	if body == "" {
		return nil, ErrMissingBody
	}

	comment.Body = body
	comment.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// Delete deletes a comment
func (s *Service) Delete(ctx context.Context, id string) error {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if comment == nil {
		return ErrNotFound
	}
	return s.store.Delete(ctx, id)
}

// List lists comments for a target
func (s *Service) List(ctx context.Context, targetType, targetID string, opts *ListOpts) ([]*Comment, int, error) {
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

	return s.store.List(ctx, targetType, targetID, limit, offset)
}

// CountByTarget counts comments for a target
func (s *Service) CountByTarget(ctx context.Context, targetType, targetID string) (int, error) {
	return s.store.CountByTarget(ctx, targetType, targetID)
}
