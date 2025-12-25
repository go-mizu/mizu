package chats

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/messaging/pkg/ulid"
)

// Service implements the chats API.
type Service struct {
	store Store
}

// NewService creates a new chats service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateDirect creates a direct (1-on-1) chat.
func (s *Service) CreateDirect(ctx context.Context, userID string, in *CreateDirectIn) (*Chat, error) {
	// Check if direct chat already exists
	existing, err := s.store.GetDirectChat(ctx, userID, in.RecipientID)
	if err == nil && existing != nil {
		return existing, nil
	}

	now := time.Now()
	chat := &Chat{
		ID:        ulid.New(),
		Type:      TypeDirect,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Insert(ctx, chat); err != nil {
		return nil, err
	}

	// Add both participants
	participants := []*Participant{
		{
			ChatID:   chat.ID,
			UserID:   userID,
			Role:     "member",
			JoinedAt: now,
		},
		{
			ChatID:   chat.ID,
			UserID:   in.RecipientID,
			Role:     "member",
			JoinedAt: now,
		},
	}

	for _, p := range participants {
		if err := s.store.InsertParticipant(ctx, p); err != nil {
			return nil, err
		}
	}

	return chat, nil
}

// CreateGroup creates a group chat.
func (s *Service) CreateGroup(ctx context.Context, userID string, in *CreateGroupIn) (*Chat, error) {
	now := time.Now()
	chat := &Chat{
		ID:          ulid.New(),
		Type:        TypeGroup,
		Name:        in.Name,
		Description: in.Description,
		IconURL:     in.IconURL,
		OwnerID:     userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Insert(ctx, chat); err != nil {
		return nil, err
	}

	// Add creator as owner
	owner := &Participant{
		ChatID:   chat.ID,
		UserID:   userID,
		Role:     "owner",
		JoinedAt: now,
	}
	if err := s.store.InsertParticipant(ctx, owner); err != nil {
		return nil, err
	}

	// Add other participants
	for _, pid := range in.ParticipantIDs {
		if pid == userID {
			continue
		}
		p := &Participant{
			ChatID:   chat.ID,
			UserID:   pid,
			Role:     "member",
			JoinedAt: now,
		}
		if err := s.store.InsertParticipant(ctx, p); err != nil {
			return nil, err
		}
	}

	return chat, nil
}

// GetByID retrieves a chat by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Chat, error) {
	return s.store.GetByID(ctx, id)
}

// GetByIDForUser retrieves a chat for a specific user (with unread count, etc.).
func (s *Service) GetByIDForUser(ctx context.Context, id, userID string) (*Chat, error) {
	return s.store.GetByIDForUser(ctx, id, userID)
}

// GetDirectChat finds an existing direct chat between two users.
func (s *Service) GetDirectChat(ctx context.Context, userID1, userID2 string) (*Chat, error) {
	return s.store.GetDirectChat(ctx, userID1, userID2)
}

// List lists chats for a user.
func (s *Service) List(ctx context.Context, userID string, opts ListOpts) ([]*Chat, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 50
	}
	return s.store.List(ctx, userID, opts)
}

// Update updates a chat.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Chat, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// Delete deletes a chat.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// AddParticipant adds a participant to a chat.
func (s *Service) AddParticipant(ctx context.Context, chatID, userID string, role string) error {
	if role == "" {
		role = "member"
	}
	p := &Participant{
		ChatID:   chatID,
		UserID:   userID,
		Role:     role,
		JoinedAt: time.Now(),
	}
	return s.store.InsertParticipant(ctx, p)
}

// RemoveParticipant removes a participant from a chat.
func (s *Service) RemoveParticipant(ctx context.Context, chatID, userID string) error {
	return s.store.DeleteParticipant(ctx, chatID, userID)
}

// UpdateParticipantRole updates a participant's role.
func (s *Service) UpdateParticipantRole(ctx context.Context, chatID, userID, role string) error {
	return s.store.UpdateParticipantRole(ctx, chatID, userID, role)
}

// GetParticipants gets all participants in a chat.
func (s *Service) GetParticipants(ctx context.Context, chatID string) ([]*Participant, error) {
	return s.store.GetParticipants(ctx, chatID)
}

// IsParticipant checks if a user is a participant in a chat.
func (s *Service) IsParticipant(ctx context.Context, chatID, userID string) (bool, error) {
	return s.store.IsParticipant(ctx, chatID, userID)
}

// Mute mutes a chat for a user.
func (s *Service) Mute(ctx context.Context, chatID, userID string, until *time.Time) error {
	return s.store.Mute(ctx, chatID, userID, until)
}

// Unmute unmutes a chat for a user.
func (s *Service) Unmute(ctx context.Context, chatID, userID string) error {
	return s.store.Unmute(ctx, chatID, userID)
}

// Archive archives a chat for a user.
func (s *Service) Archive(ctx context.Context, chatID, userID string) error {
	return s.store.Archive(ctx, chatID, userID)
}

// Unarchive unarchives a chat for a user.
func (s *Service) Unarchive(ctx context.Context, chatID, userID string) error {
	return s.store.Unarchive(ctx, chatID, userID)
}

// Pin pins a chat for a user.
func (s *Service) Pin(ctx context.Context, chatID, userID string) error {
	return s.store.Pin(ctx, chatID, userID)
}

// Unpin unpins a chat for a user.
func (s *Service) Unpin(ctx context.Context, chatID, userID string) error {
	return s.store.Unpin(ctx, chatID, userID)
}

// MarkAsRead marks a chat as read up to a message.
func (s *Service) MarkAsRead(ctx context.Context, chatID, userID, messageID string) error {
	if err := s.store.MarkAsRead(ctx, chatID, userID, messageID); err != nil {
		return err
	}
	return s.store.ResetUnreadCount(ctx, chatID, userID)
}

// IncrementMessageCount increments the message count.
func (s *Service) IncrementMessageCount(ctx context.Context, chatID string) error {
	return s.store.IncrementMessageCount(ctx, chatID)
}

// UpdateLastMessage updates the last message info.
func (s *Service) UpdateLastMessage(ctx context.Context, chatID, messageID string) error {
	return s.store.UpdateLastMessage(ctx, chatID, messageID)
}
