package shares

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/drive/pkg/crypto"
	"github.com/go-mizu/blueprints/drive/pkg/password"
	"github.com/go-mizu/blueprints/drive/pkg/ulid"
)

// Service implements the shares API.
type Service struct {
	store     Store
	linkStore LinkStore
	baseURL   string
}

// NewService creates a new shares service.
func NewService(store Store, linkStore LinkStore, baseURL string) *Service {
	return &Service{
		store:     store,
		linkStore: linkStore,
		baseURL:   baseURL,
	}
}

// Create creates a new share.
func (s *Service) Create(ctx context.Context, ownerID string, in *CreateShareIn) (*Share, error) {
	// Check if already shared
	if existing, _ := s.store.GetByItemAndUser(ctx, in.ItemID, in.ItemType, in.SharedWith); existing != nil {
		return nil, ErrAlreadyShared
	}

	now := time.Now()
	share := &Share{
		ID:         ulid.New(),
		ItemID:     in.ItemID,
		ItemType:   in.ItemType,
		OwnerID:    ownerID,
		SharedWith: in.SharedWith,
		Permission: in.Permission,
		Notify:     in.Notify,
		Message:    in.Message,
		CreatedAt:  now,
		UpdatedAt:  now,
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

// ListByOwner lists shares created by owner.
func (s *Service) ListByOwner(ctx context.Context, ownerID string) ([]*Share, error) {
	return s.store.ListByOwner(ctx, ownerID)
}

// ListSharedWithMe lists items shared with the user.
func (s *Service) ListSharedWithMe(ctx context.Context, accountID string) ([]*SharedItem, error) {
	shares, err := s.store.ListBySharedWith(ctx, accountID)
	if err != nil {
		return nil, err
	}

	var items []*SharedItem
	for _, share := range shares {
		items = append(items, &SharedItem{
			Share:    share,
			ItemID:   share.ItemID,
			ItemType: share.ItemType,
		})
	}

	return items, nil
}

// ListForItem lists all shares for an item.
func (s *Service) ListForItem(ctx context.Context, itemID, itemType string) ([]*Share, error) {
	return s.store.ListByItem(ctx, itemID, itemType)
}

// Update updates a share's permission.
func (s *Service) Update(ctx context.Context, id, ownerID string, permission string) (*Share, error) {
	share, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if share.OwnerID != ownerID {
		return nil, ErrNotOwner
	}

	if err := s.store.Update(ctx, id, permission); err != nil {
		return nil, err
	}

	return s.store.GetByID(ctx, id)
}

// Delete deletes a share.
func (s *Service) Delete(ctx context.Context, id, ownerID string) error {
	share, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if share.OwnerID != ownerID {
		return ErrNotOwner
	}

	return s.store.Delete(ctx, id)
}

// CreateLink creates a share link.
func (s *Service) CreateLink(ctx context.Context, ownerID string, in *CreateLinkIn) (*ShareLink, error) {
	token, err := crypto.GenerateShareToken()
	if err != nil {
		return nil, err
	}

	var passwordHash string
	if in.Password != "" {
		passwordHash, err = password.Hash(in.Password)
		if err != nil {
			return nil, err
		}
	}

	link := &ShareLink{
		ID:            ulid.New(),
		ItemID:        in.ItemID,
		ItemType:      in.ItemType,
		OwnerID:       ownerID,
		Token:         token,
		Permission:    in.Permission,
		PasswordHash:  passwordHash,
		HasPassword:   passwordHash != "",
		ExpiresAt:     in.ExpiresAt,
		DownloadLimit: in.DownloadLimit,
		DownloadCount: 0,
		AllowDownload: in.AllowDownload,
		Disabled:      false,
		CreatedAt:     time.Now(),
	}

	if err := s.linkStore.Create(ctx, link); err != nil {
		return nil, err
	}

	link.URL = s.baseURL + "/s/" + token

	return link, nil
}

// GetLinkByID retrieves a link by ID.
func (s *Service) GetLinkByID(ctx context.Context, id string) (*ShareLink, error) {
	link, err := s.linkStore.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	link.URL = s.baseURL + "/s/" + link.Token
	return link, nil
}

// GetLinkByToken retrieves a link by token.
func (s *Service) GetLinkByToken(ctx context.Context, token string) (*ShareLink, error) {
	link, err := s.linkStore.GetByToken(ctx, token)
	if err != nil {
		return nil, ErrLinkNotFound
	}

	if link.Disabled {
		return nil, ErrLinkDisabled
	}

	if link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt) {
		return nil, ErrLinkExpired
	}

	if link.DownloadLimit != nil && link.DownloadCount >= *link.DownloadLimit {
		return nil, ErrDownloadLimit
	}

	link.URL = s.baseURL + "/s/" + link.Token
	return link, nil
}

// ListLinksForItem lists all links for an item.
func (s *Service) ListLinksForItem(ctx context.Context, itemID, itemType string) ([]*ShareLink, error) {
	links, err := s.linkStore.ListByItem(ctx, itemID, itemType)
	if err != nil {
		return nil, err
	}

	for _, link := range links {
		link.URL = s.baseURL + "/s/" + link.Token
	}

	return links, nil
}

// UpdateLink updates a share link.
func (s *Service) UpdateLink(ctx context.Context, id, ownerID string, in *UpdateLinkIn) (*ShareLink, error) {
	link, err := s.linkStore.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if link.OwnerID != ownerID {
		return nil, ErrNotOwner
	}

	var passwordHash string
	if in.Password != nil && *in.Password != "" {
		passwordHash, _ = password.Hash(*in.Password)
	}

	if err := s.linkStore.Update(ctx, id, in, passwordHash); err != nil {
		return nil, err
	}

	return s.GetLinkByID(ctx, id)
}

// DeleteLink deletes a share link.
func (s *Service) DeleteLink(ctx context.Context, id, ownerID string) error {
	link, err := s.linkStore.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if link.OwnerID != ownerID {
		return ErrNotOwner
	}

	return s.linkStore.Delete(ctx, id)
}

// VerifyLinkPassword verifies a share link password.
func (s *Service) VerifyLinkPassword(ctx context.Context, token, pwd string) (bool, error) {
	link, err := s.linkStore.GetByToken(ctx, token)
	if err != nil {
		return false, ErrLinkNotFound
	}

	if link.PasswordHash == "" {
		return true, nil
	}

	return password.Verify(pwd, link.PasswordHash)
}

// RecordLinkAccess records access to a link.
func (s *Service) RecordLinkAccess(ctx context.Context, token string) error {
	link, err := s.linkStore.GetByToken(ctx, token)
	if err != nil {
		return err
	}

	return s.linkStore.UpdateAccess(ctx, link.ID)
}

// GetEffectivePermission returns the effective permission for a user on an item.
func (s *Service) GetEffectivePermission(ctx context.Context, accountID, itemID, itemType string) (*EffectivePermission, error) {
	// Check direct share
	share, err := s.store.GetByItemAndUser(ctx, itemID, itemType, accountID)
	if err == nil && share != nil {
		return &EffectivePermission{
			Permission: share.Permission,
			Source:     "share",
			ItemID:     itemID,
			ItemType:   itemType,
		}, nil
	}

	return nil, ErrNoPermission
}

// CanPerform checks if a user can perform an action.
func (s *Service) CanPerform(ctx context.Context, accountID, itemID, itemType, action string) (bool, error) {
	perm, err := s.GetEffectivePermission(ctx, accountID, itemID, itemType)
	if err != nil {
		return false, nil
	}

	return permissionAllows(perm.Permission, action), nil
}

var permissionActions = map[string][]string{
	PermissionOwner:     {"view", "download", "comment", "edit", "upload", "move", "copy", "share", "delete", "manage"},
	PermissionEditor:    {"view", "download", "comment", "edit", "upload", "move", "copy", "share"},
	PermissionCommenter: {"view", "download", "comment"},
	PermissionViewer:    {"view", "download"},
}

func permissionAllows(perm, action string) bool {
	actions, ok := permissionActions[perm]
	if !ok {
		return false
	}
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}
