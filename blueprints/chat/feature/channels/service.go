package channels

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/chat/pkg/ulid"
)

// Service implements the channels API.
type Service struct {
	store Store
}

// NewService creates a new channels service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new channel.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Channel, error) {
	now := time.Now()
	ch := &Channel{
		ID:            ulid.New(),
		ServerID:      in.ServerID,
		CategoryID:    in.CategoryID,
		Type:          in.Type,
		Name:          in.Name,
		Topic:         in.Topic,
		IsPrivate:     in.IsPrivate,
		IsNSFW:        in.IsNSFW,
		SlowModeDelay: in.SlowModeDelay,
		Bitrate:       in.Bitrate,
		UserLimit:     in.UserLimit,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.store.Insert(ctx, ch); err != nil {
		return nil, err
	}

	return ch, nil
}

// GetByID retrieves a channel by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Channel, error) {
	ch, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load recipients for DMs
	if ch.Type == TypeDM || ch.Type == TypeGroupDM {
		ch.Recipients, _ = s.store.GetRecipients(ctx, id)
	}

	return ch, nil
}

// Update updates a channel.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Channel, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// Delete deletes a channel.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListByServer lists channels in a server.
func (s *Service) ListByServer(ctx context.Context, serverID string) ([]*Channel, error) {
	return s.store.ListByServer(ctx, serverID)
}

// ListDMsByUser lists DM channels for a user.
func (s *Service) ListDMsByUser(ctx context.Context, userID string) ([]*Channel, error) {
	channels, err := s.store.ListDMsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Load recipients for each channel
	for _, ch := range channels {
		ch.Recipients, _ = s.store.GetRecipients(ctx, ch.ID)
	}

	return channels, nil
}

// GetOrCreateDM gets or creates a DM channel between two users.
func (s *Service) GetOrCreateDM(ctx context.Context, userID1, userID2 string) (*Channel, error) {
	// Try to find existing DM
	ch, err := s.store.GetDMChannel(ctx, userID1, userID2)
	if err == nil {
		ch.Recipients, _ = s.store.GetRecipients(ctx, ch.ID)
		return ch, nil
	}
	if err != ErrNotFound {
		return nil, err
	}

	// Create new DM
	now := time.Now()
	ch = &Channel{
		ID:        ulid.New(),
		Type:      TypeDM,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Insert(ctx, ch); err != nil {
		return nil, err
	}

	// Add recipients
	if err := s.store.AddRecipient(ctx, ch.ID, userID1); err != nil {
		return nil, err
	}
	if err := s.store.AddRecipient(ctx, ch.ID, userID2); err != nil {
		return nil, err
	}

	ch.Recipients = []string{userID1, userID2}
	return ch, nil
}

// CreateGroupDM creates a group DM.
func (s *Service) CreateGroupDM(ctx context.Context, ownerID string, recipientIDs []string, name string) (*Channel, error) {
	now := time.Now()
	ch := &Channel{
		ID:        ulid.New(),
		Type:      TypeGroupDM,
		Name:      name,
		OwnerID:   ownerID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Insert(ctx, ch); err != nil {
		return nil, err
	}

	// Add owner as recipient
	if err := s.store.AddRecipient(ctx, ch.ID, ownerID); err != nil {
		return nil, err
	}

	// Add other recipients
	for _, id := range recipientIDs {
		s.store.AddRecipient(ctx, ch.ID, id)
	}

	ch.Recipients = append([]string{ownerID}, recipientIDs...)
	return ch, nil
}

// AddRecipient adds a recipient to a group DM.
func (s *Service) AddRecipient(ctx context.Context, channelID, userID string) error {
	return s.store.AddRecipient(ctx, channelID, userID)
}

// RemoveRecipient removes a recipient from a group DM.
func (s *Service) RemoveRecipient(ctx context.Context, channelID, userID string) error {
	return s.store.RemoveRecipient(ctx, channelID, userID)
}

// GetRecipients gets all recipients of a channel.
func (s *Service) GetRecipients(ctx context.Context, channelID string) ([]string, error) {
	return s.store.GetRecipients(ctx, channelID)
}

// UpdateLastMessage updates the last message info.
func (s *Service) UpdateLastMessage(ctx context.Context, channelID, messageID string, at time.Time) error {
	return s.store.UpdateLastMessage(ctx, channelID, messageID, at)
}

// CreateCategory creates a new category.
func (s *Service) CreateCategory(ctx context.Context, serverID, name string, position int) (*Category, error) {
	cat := &Category{
		ID:        ulid.New(),
		ServerID:  serverID,
		Name:      name,
		Position:  position,
		CreatedAt: time.Now(),
	}

	if err := s.store.InsertCategory(ctx, cat); err != nil {
		return nil, err
	}

	return cat, nil
}

// GetCategory retrieves a category by ID.
func (s *Service) GetCategory(ctx context.Context, id string) (*Category, error) {
	return s.store.GetCategory(ctx, id)
}

// ListCategories lists categories in a server.
func (s *Service) ListCategories(ctx context.Context, serverID string) ([]*Category, error) {
	return s.store.ListCategories(ctx, serverID)
}

// DeleteCategory deletes a category.
func (s *Service) DeleteCategory(ctx context.Context, id string) error {
	return s.store.DeleteCategory(ctx, id)
}
