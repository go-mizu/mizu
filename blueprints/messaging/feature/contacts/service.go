package contacts

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/messaging/pkg/ulid"
)

// Service implements the contacts API.
type Service struct {
	store Store
}

// NewService creates a new contacts service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Add adds a contact.
func (s *Service) Add(ctx context.Context, userID string, in *AddIn) (*Contact, error) {
	if userID == in.ContactUserID {
		return nil, ErrSelfContact
	}

	exists, err := s.store.Exists(ctx, userID, in.ContactUserID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyAdded
	}

	c := &Contact{
		UserID:        userID,
		ContactUserID: in.ContactUserID,
		DisplayName:   in.DisplayName,
		IsBlocked:     false,
		IsFavorite:    false,
		CreatedAt:     time.Now(),
	}

	if err := s.store.Insert(ctx, c); err != nil {
		return nil, err
	}

	return c, nil
}

// Get retrieves a contact.
func (s *Service) Get(ctx context.Context, userID, contactUserID string) (*Contact, error) {
	return s.store.Get(ctx, userID, contactUserID)
}

// List lists all contacts for a user.
func (s *Service) List(ctx context.Context, userID string) ([]*Contact, error) {
	return s.store.List(ctx, userID)
}

// ListFavorites lists favorite contacts.
func (s *Service) ListFavorites(ctx context.Context, userID string) ([]*Contact, error) {
	return s.store.ListFavorites(ctx, userID)
}

// ListBlocked lists blocked contacts.
func (s *Service) ListBlocked(ctx context.Context, userID string) ([]*Contact, error) {
	return s.store.ListBlocked(ctx, userID)
}

// Update updates a contact.
func (s *Service) Update(ctx context.Context, userID, contactUserID string, in *UpdateIn) (*Contact, error) {
	if err := s.store.Update(ctx, userID, contactUserID, in); err != nil {
		return nil, err
	}
	return s.store.Get(ctx, userID, contactUserID)
}

// Remove removes a contact.
func (s *Service) Remove(ctx context.Context, userID, contactUserID string) error {
	return s.store.Delete(ctx, userID, contactUserID)
}

// Block blocks a user.
func (s *Service) Block(ctx context.Context, userID, contactUserID string) error {
	// First check if contact exists, if not create it
	exists, err := s.store.Exists(ctx, userID, contactUserID)
	if err != nil {
		return err
	}
	if !exists {
		c := &Contact{
			UserID:        userID,
			ContactUserID: contactUserID,
			IsBlocked:     true,
			BlockedAt:     time.Now(),
			CreatedAt:     time.Now(),
		}
		return s.store.Insert(ctx, c)
	}
	return s.store.Block(ctx, userID, contactUserID)
}

// Unblock unblocks a user.
func (s *Service) Unblock(ctx context.Context, userID, contactUserID string) error {
	return s.store.Unblock(ctx, userID, contactUserID)
}

// IsBlocked checks if a user is blocked.
func (s *Service) IsBlocked(ctx context.Context, userID, targetUserID string) (bool, error) {
	return s.store.IsBlocked(ctx, userID, targetUserID)
}

// Ensure ulid is used
var _ = ulid.New
