package shares

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// Service implements the shares API.
type Service struct {
	store Store
}

// NewService creates a new shares service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new share.
func (s *Service) Create(ctx context.Context, userID string, in CreateIn) (*Share, error) {
	share := &Share{
		ID:         ulid.New(),
		BaseID:     in.BaseID,
		TableID:    in.TableID,
		ViewID:     in.ViewID,
		Type:       in.Type,
		Permission: in.Permission,
		UserID:     in.UserID,
		Email:      in.Email,
		ExpiresAt:  in.ExpiresAt,
		CreatedBy:  userID,
	}

	// Generate token for link shares
	if share.Type == TypeLink {
		share.Token = generateToken()
	}

	if err := s.store.Create(ctx, share); err != nil {
		return nil, err
	}

	return share, nil
}

// GetByID retrieves a share by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Share, error) {
	return s.store.GetByID(ctx, id)
}

// GetByToken retrieves a share by token.
func (s *Service) GetByToken(ctx context.Context, token string) (*Share, error) {
	share, err := s.store.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check expiration
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	return share, nil
}

// Update updates a share.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*Share, error) {
	share, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Permission != nil {
		share.Permission = *in.Permission
	}
	if in.ExpiresAt != nil {
		share.ExpiresAt = in.ExpiresAt
	}

	if err := s.store.Update(ctx, share); err != nil {
		return nil, err
	}

	return share, nil
}

// Delete deletes a share.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListByBase lists all shares for a base.
func (s *Service) ListByBase(ctx context.Context, baseID string) ([]*Share, error) {
	return s.store.ListByBase(ctx, baseID)
}

// CanAccess checks if a user can access a base with the given permission.
func (s *Service) CanAccess(ctx context.Context, userID string, baseID string, permission string) (bool, error) {
	shares, err := s.store.ListByUser(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, share := range shares {
		if share.BaseID == baseID && hasPermission(share.Permission, permission) {
			return true, nil
		}
	}

	return false, nil
}

// CanAccessTable checks if a user can access a table with the given permission.
func (s *Service) CanAccessTable(ctx context.Context, userID string, tableID string, permission string) (bool, error) {
	shares, err := s.store.ListByUser(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, share := range shares {
		if share.TableID == tableID && hasPermission(share.Permission, permission) {
			return true, nil
		}
	}

	return false, nil
}

// CanAccessView checks if a user can access a view with the given permission.
func (s *Service) CanAccessView(ctx context.Context, userID string, viewID string, permission string) (bool, error) {
	shares, err := s.store.ListByUser(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, share := range shares {
		if share.ViewID == viewID && hasPermission(share.Permission, permission) {
			return true, nil
		}
	}

	return false, nil
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hasPermission(granted, required string) bool {
	order := map[string]int{
		PermRead:    1,
		PermComment: 2,
		PermEdit:    3,
		PermAdmin:   4,
	}
	return order[granted] >= order[required]
}
