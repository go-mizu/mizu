package messages

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/messaging/pkg/ulid"
)

// Service implements the messages API.
type Service struct {
	store Store
}

// NewService creates a new messages service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new message.
func (s *Service) Create(ctx context.Context, senderID string, in *CreateIn) (*Message, error) {
	now := time.Now()
	m := &Message{
		ID:              ulid.New(),
		ChatID:          in.ChatID,
		SenderID:        senderID,
		Type:            in.Type,
		Content:         in.Content,
		ReplyToID:       in.ReplyToID,
		MentionEveryone: in.MentionEveryone,
		Status:          StatusSent,
		CreatedAt:       now,
	}

	if in.Type == "" {
		m.Type = TypeText
	}

	if in.ExpiresIn > 0 {
		expiresAt := now.Add(time.Duration(in.ExpiresIn) * time.Second)
		m.ExpiresAt = &expiresAt
	}

	if err := s.store.Insert(ctx, m); err != nil {
		return nil, err
	}

	// Insert mentions
	for _, userID := range in.Mentions {
		if err := s.store.InsertMention(ctx, m.ID, userID); err != nil {
			// Log but don't fail
		}
	}

	return m, nil
}

// GetByID retrieves a message by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Message, error) {
	return s.store.GetByID(ctx, id)
}

// Update updates a message.
func (s *Service) Update(ctx context.Context, id, userID string, in *UpdateIn) (*Message, error) {
	m, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m.SenderID != userID {
		return nil, ErrForbidden
	}

	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// Delete deletes a message.
func (s *Service) Delete(ctx context.Context, id, userID string, forEveryone bool) error {
	m, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Only sender can delete for everyone
	if forEveryone && m.SenderID != userID {
		return ErrForbidden
	}

	return s.store.Delete(ctx, id, forEveryone)
}

// Forward forwards a message to other chats.
func (s *Service) Forward(ctx context.Context, id, userID string, in *ForwardIn) ([]*Message, error) {
	original, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var forwarded []*Message
	now := time.Now()

	for _, chatID := range in.ToChatIDs {
		m := &Message{
			ID:                    ulid.New(),
			ChatID:                chatID,
			SenderID:              userID,
			Type:                  original.Type,
			Content:               original.Content,
			ForwardFromID:         original.ID,
			ForwardFromChatID:     original.ChatID,
			IsForwarded:           true,
			Status:                StatusSent,
			CreatedAt:             now,
		}

		if err := s.store.Insert(ctx, m); err != nil {
			return nil, err
		}
		forwarded = append(forwarded, m)
	}

	return forwarded, nil
}

// List lists messages in a chat.
func (s *Service) List(ctx context.Context, chatID string, opts ListOpts) ([]*Message, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 50
	}
	return s.store.List(ctx, chatID, opts)
}

// Search searches messages.
func (s *Service) Search(ctx context.Context, opts SearchOpts) ([]*Message, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 50
	}
	return s.store.Search(ctx, opts)
}

// AddReaction adds a reaction to a message.
func (s *Service) AddReaction(ctx context.Context, messageID, userID, emoji string) error {
	return s.store.AddReaction(ctx, messageID, userID, emoji)
}

// RemoveReaction removes a reaction from a message.
func (s *Service) RemoveReaction(ctx context.Context, messageID, userID string) error {
	return s.store.RemoveReaction(ctx, messageID, userID)
}

// GetReactions gets reactions on a message.
func (s *Service) GetReactions(ctx context.Context, messageID string) ([]Reaction, error) {
	return s.store.GetReactions(ctx, messageID)
}

// Star stars a message.
func (s *Service) Star(ctx context.Context, messageID, userID string) error {
	return s.store.Star(ctx, messageID, userID)
}

// Unstar unstars a message.
func (s *Service) Unstar(ctx context.Context, messageID, userID string) error {
	return s.store.Unstar(ctx, messageID, userID)
}

// ListStarred lists starred messages.
func (s *Service) ListStarred(ctx context.Context, userID string, limit int) ([]*Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.store.ListStarred(ctx, userID, limit)
}

// AddMedia adds media to a message.
func (s *Service) AddMedia(ctx context.Context, messageID string, media *Media) error {
	media.ID = ulid.New()
	media.MessageID = messageID
	media.CreatedAt = time.Now()
	return s.store.InsertMedia(ctx, media)
}

// GetMedia gets media for a message.
func (s *Service) GetMedia(ctx context.Context, messageID string) ([]*Media, error) {
	return s.store.GetMedia(ctx, messageID)
}

// ViewMedia records a view of view-once media.
func (s *Service) ViewMedia(ctx context.Context, mediaID, userID string) error {
	return s.store.IncrementViewCount(ctx, mediaID)
}

// MarkDelivered marks a message as delivered.
func (s *Service) MarkDelivered(ctx context.Context, messageID, userID string) error {
	return s.store.UpdateRecipientStatus(ctx, messageID, userID, StatusDelivered)
}

// MarkRead marks a message as read.
func (s *Service) MarkRead(ctx context.Context, messageID, userID string) error {
	return s.store.UpdateRecipientStatus(ctx, messageID, userID, StatusRead)
}

// GetDeliveryStatus gets delivery status for a message.
func (s *Service) GetDeliveryStatus(ctx context.Context, messageID string) ([]*Recipient, error) {
	return s.store.GetRecipients(ctx, messageID)
}

// Pin pins a message in a chat.
func (s *Service) Pin(ctx context.Context, chatID, messageID, userID string) error {
	return s.store.Pin(ctx, chatID, messageID, userID)
}

// Unpin unpins a message from a chat.
func (s *Service) Unpin(ctx context.Context, chatID, messageID string) error {
	return s.store.Unpin(ctx, chatID, messageID)
}

// ListPinned lists pinned messages in a chat.
func (s *Service) ListPinned(ctx context.Context, chatID string) ([]*Message, error) {
	return s.store.ListPinned(ctx, chatID)
}
