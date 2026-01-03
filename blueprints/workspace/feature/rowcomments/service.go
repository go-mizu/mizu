package rowcomments

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/users"
	"github.com/oklog/ulid/v2"
)

// Service implements the row comments API.
type Service struct {
	store Store
	users users.API
}

// NewService creates a new row comments service.
func NewService(store Store, users users.API) *Service {
	return &Service{store: store, users: users}
}

// Create creates a new comment on a row.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Comment, error) {
	comment := &Comment{
		ID:        ulid.Make().String(),
		RowID:     in.RowID,
		UserID:    in.UserID,
		Content:   in.Content,
		Resolved:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.store.Create(ctx, comment); err != nil {
		return nil, err
	}

	// Enrich with user info
	if s.users != nil {
		if user, err := s.users.GetByID(ctx, in.UserID); err == nil && user != nil {
			comment.UserName = user.Name
			comment.UserAvatar = user.AvatarURL
		}
	}

	return comment, nil
}

// GetByID retrieves a comment by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Comment, error) {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Enrich with user info
	if s.users != nil && comment != nil {
		if user, err := s.users.GetByID(ctx, comment.UserID); err == nil && user != nil {
			comment.UserName = user.Name
			comment.UserAvatar = user.AvatarURL
		}
	}

	return comment, nil
}

// Update updates a comment's content.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Comment, error) {
	if err := s.store.Update(ctx, id, in.Content); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// Delete deletes a comment.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListByRow retrieves all comments for a row.
func (s *Service) ListByRow(ctx context.Context, rowID string) ([]*Comment, error) {
	comments, err := s.store.ListByRow(ctx, rowID)
	if err != nil {
		return nil, err
	}

	// Batch enrich with user info to avoid N+1 queries
	if s.users != nil && len(comments) > 0 {
		userIDs := make([]string, 0, len(comments))
		for _, comment := range comments {
			if comment.UserID != "" {
				userIDs = append(userIDs, comment.UserID)
			}
		}

		if len(userIDs) > 0 {
			usersMap, err := s.users.GetByIDs(ctx, userIDs)
			if err == nil && usersMap != nil {
				for _, comment := range comments {
					if user := usersMap[comment.UserID]; user != nil {
						comment.UserName = user.Name
						comment.UserAvatar = user.AvatarURL
					}
				}
			}
		}
	}

	return comments, nil
}

// Resolve marks a comment as resolved.
func (s *Service) Resolve(ctx context.Context, id string) error {
	return s.store.SetResolved(ctx, id, true)
}

// Unresolve marks a comment as unresolved.
func (s *Service) Unresolve(ctx context.Context, id string) error {
	return s.store.SetResolved(ctx, id, false)
}
