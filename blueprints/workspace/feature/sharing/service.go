package sharing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/users"
	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrNotFound       = errors.New("share not found")
	ErrShareExists    = errors.New("share already exists")
	ErrInvalidToken   = errors.New("invalid share token")
	ErrShareExpired   = errors.New("share link has expired")
	ErrInvalidPassword = errors.New("invalid password")
	ErrNoAccess       = errors.New("no access to page")
)

// Service implements the sharing API.
type Service struct {
	store Store
	users users.API
}

// NewService creates a new sharing service.
func NewService(store Store, users users.API) *Service {
	return &Service{store: store, users: users}
}

// ShareWithUser shares a page with a specific user.
func (s *Service) ShareWithUser(ctx context.Context, pageID, userID string, perm Permission, createdBy string) (*Share, error) {
	// Check if share already exists
	existing, _ := s.store.GetByPageAndUser(ctx, pageID, userID)
	if existing != nil {
		return nil, ErrShareExists
	}

	share := &Share{
		ID:         ulid.New(),
		PageID:     pageID,
		Type:       ShareUser,
		Permission: perm,
		UserID:     userID,
		CreatedBy:  createdBy,
		CreatedAt:  time.Now(),
	}

	if err := s.store.Create(ctx, share); err != nil {
		return nil, err
	}

	return s.enrichShare(ctx, share)
}

// UpdateUserPermission updates a user share's permission.
func (s *Service) UpdateUserPermission(ctx context.Context, id string, perm Permission) error {
	return s.store.Update(ctx, id, perm)
}

// RemoveUserShare removes a user share.
func (s *Service) RemoveUserShare(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// CreateShareLink creates a shareable link.
func (s *Service) CreateShareLink(ctx context.Context, pageID string, opts LinkOpts, createdBy string) (*Share, error) {
	token := generateToken()

	share := &Share{
		ID:         ulid.New(),
		PageID:     pageID,
		Type:       ShareLink,
		Permission: opts.Permission,
		Token:      token,
		Password:   opts.Password,
		ExpiresAt:  opts.ExpiresAt,
		CreatedBy:  createdBy,
		CreatedAt:  time.Now(),
	}

	if share.Permission == "" {
		share.Permission = PermRead
	}

	if err := s.store.Create(ctx, share); err != nil {
		return nil, err
	}

	return share, nil
}

// UpdateShareLink updates a share link.
func (s *Service) UpdateShareLink(ctx context.Context, id string, opts LinkOpts) error {
	return s.store.UpdateLink(ctx, id, opts)
}

// DeleteShareLink deletes a share link.
func (s *Service) DeleteShareLink(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// GetByToken retrieves a share by token.
func (s *Service) GetByToken(ctx context.Context, token string) (*Share, error) {
	share, err := s.store.GetByToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		return nil, ErrShareExpired
	}

	return share, nil
}

// EnablePublic enables public access to a page.
func (s *Service) EnablePublic(ctx context.Context, pageID, createdBy string) (*Share, error) {
	// Check if already public
	existing, _ := s.store.GetPublicByPage(ctx, pageID)
	if existing != nil {
		return existing, nil
	}

	share := &Share{
		ID:         ulid.New(),
		PageID:     pageID,
		Type:       SharePublic,
		Permission: PermRead,
		CreatedBy:  createdBy,
		CreatedAt:  time.Now(),
	}

	if err := s.store.Create(ctx, share); err != nil {
		return nil, err
	}

	return share, nil
}

// DisablePublic disables public access to a page.
func (s *Service) DisablePublic(ctx context.Context, pageID string) error {
	share, err := s.store.GetPublicByPage(ctx, pageID)
	if err != nil {
		return nil // Already not public
	}
	return s.store.Delete(ctx, share.ID)
}

// ListByPage lists all shares for a page.
func (s *Service) ListByPage(ctx context.Context, pageID string) ([]*Share, error) {
	shares, err := s.store.ListByPage(ctx, pageID)
	if err != nil {
		return nil, err
	}
	return s.enrichShares(ctx, shares)
}

// CanAccess checks if a user can access a page.
func (s *Service) CanAccess(ctx context.Context, userID, pageID string) (Permission, error) {
	// Check direct user share
	share, err := s.store.GetByPageAndUser(ctx, pageID, userID)
	if err == nil && share != nil {
		return share.Permission, nil
	}

	// Check public share
	publicShare, _ := s.store.GetPublicByPage(ctx, pageID)
	if publicShare != nil {
		return publicShare.Permission, nil
	}

	return "", ErrNoAccess
}

// enrichShare adds user data to a share.
func (s *Service) enrichShare(ctx context.Context, share *Share) (*Share, error) {
	if share.UserID != "" {
		user, _ := s.users.GetByID(ctx, share.UserID)
		share.User = user
	}
	return share, nil
}

// enrichShares adds user data to multiple shares.
func (s *Service) enrichShares(ctx context.Context, shares []*Share) ([]*Share, error) {
	if len(shares) == 0 {
		return shares, nil
	}

	// Collect user IDs
	userIDs := make([]string, 0)
	for _, share := range shares {
		if share.UserID != "" {
			userIDs = append(userIDs, share.UserID)
		}
	}

	if len(userIDs) == 0 {
		return shares, nil
	}

	// Batch fetch users
	usersMap, _ := s.users.GetByIDs(ctx, userIDs)

	// Attach users
	for _, share := range shares {
		if share.UserID != "" {
			share.User = usersMap[share.UserID]
		}
	}

	return shares, nil
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
