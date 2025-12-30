package comments

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

var (
	ErrNotFound       = errors.New("comment not found")
	ErrMissingContent = errors.New("content is required")
	ErrMissingPostID  = errors.New("post_id is required")
)

// Service implements the comments API.
type Service struct {
	store Store
}

// NewService creates a new comments service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, in *CreateIn) (*Comment, error) {
	if in.PostID == "" {
		return nil, ErrMissingPostID
	}
	if in.Content == "" {
		return nil, ErrMissingContent
	}

	now := time.Now()
	status := "pending"
	if in.AuthorID != "" {
		// Auto-approve for logged-in users
		status = "approved"
	}

	comment := &Comment{
		ID:          ulid.New(),
		PostID:      in.PostID,
		ParentID:    in.ParentID,
		AuthorID:    in.AuthorID,
		AuthorName:  in.AuthorName,
		AuthorEmail: in.AuthorEmail,
		AuthorURL:   in.AuthorURL,
		Content:     in.Content,
		Status:      status,
		IPAddress:   in.IPAddress,
		UserAgent:   in.UserAgent,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Create(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

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

func (s *Service) ListByPost(ctx context.Context, postID string, in *ListIn) ([]*Comment, int, error) {
	if in.Limit <= 0 {
		in.Limit = 50
	}
	in.PostID = postID
	return s.store.ListByPost(ctx, postID, in)
}

func (s *Service) List(ctx context.Context, in *ListIn) ([]*Comment, int, error) {
	if in.Limit <= 0 {
		in.Limit = 50
	}
	return s.store.List(ctx, in)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Comment, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) Approve(ctx context.Context, id string) (*Comment, error) {
	status := "approved"
	if err := s.store.Update(ctx, id, &UpdateIn{Status: &status}); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) MarkAsSpam(ctx context.Context, id string) (*Comment, error) {
	status := "spam"
	if err := s.store.Update(ctx, id, &UpdateIn{Status: &status}); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) CountByPost(ctx context.Context, postID string) (int, error) {
	return s.store.CountByPost(ctx, postID)
}
