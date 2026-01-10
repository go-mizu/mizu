package comments

import (
	"context"

	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// Service implements the comments API.
type Service struct {
	store Store
}

// NewService creates a new comments service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new comment.
func (s *Service) Create(ctx context.Context, userID string, in CreateIn) (*Comment, error) {
	comment := &Comment{
		ID:       ulid.New(),
		RecordID: in.RecordID,
		ParentID: in.ParentID,
		UserID:   userID,
		Content:  in.Content,
	}

	if err := s.store.Create(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// GetByID retrieves a comment by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Comment, error) {
	return s.store.GetByID(ctx, id)
}

// Update updates a comment.
func (s *Service) Update(ctx context.Context, id string, content string) (*Comment, error) {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	comment.Content = content

	if err := s.store.Update(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// Delete deletes a comment.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Resolve marks a comment as resolved.
func (s *Service) Resolve(ctx context.Context, id string) error {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	comment.IsResolved = true
	return s.store.Update(ctx, comment)
}

// Unresolve marks a comment as unresolved.
func (s *Service) Unresolve(ctx context.Context, id string) error {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	comment.IsResolved = false
	return s.store.Update(ctx, comment)
}

// ListByRecord lists all comments for a record.
func (s *Service) ListByRecord(ctx context.Context, recordID string) ([]*Comment, error) {
	return s.store.ListByRecord(ctx, recordID)
}
