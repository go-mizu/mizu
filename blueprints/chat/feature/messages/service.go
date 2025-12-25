package messages

import (
	"context"
	"regexp"
	"time"

	"github.com/go-mizu/blueprints/chat/pkg/ulid"
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
func (s *Service) Create(ctx context.Context, authorID string, in *CreateIn) (*Message, error) {
	msgType := in.Type
	if msgType == "" {
		msgType = TypeDefault
		if in.ReplyToID != "" {
			msgType = TypeReply
		}
	}

	contentHTML := processContent(in.Content)

	msg := &Message{
		ID:              ulid.New(),
		ChannelID:       in.ChannelID,
		AuthorID:        authorID,
		Content:         in.Content,
		ContentHTML:     contentHTML,
		Type:            msgType,
		ReplyToID:       in.ReplyToID,
		Mentions:        in.Mentions,
		MentionEveryone: in.MentionEveryone,
		CreatedAt:       time.Now(),
	}

	// Extract mentions from content
	if len(msg.Mentions) == 0 {
		msg.Mentions = extractMentions(in.Content)
	}

	if err := s.store.Insert(ctx, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// GetByID retrieves a message by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Message, error) {
	return s.store.GetByID(ctx, id)
}

// Update updates a message.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Message, error) {
	// Process content if provided
	if in.Content != nil {
		html := processContent(*in.Content)
		in.ContentHTML = &html
	}

	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// Delete deletes a message.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// List lists messages in a channel.
func (s *Service) List(ctx context.Context, channelID string, opts ListOpts) ([]*Message, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 50
	}
	return s.store.List(ctx, channelID, opts)
}

// Search searches messages.
func (s *Service) Search(ctx context.Context, opts SearchOpts) ([]*Message, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 25
	}
	return s.store.Search(ctx, opts)
}

// Pin pins a message.
func (s *Service) Pin(ctx context.Context, channelID, messageID, userID string) error {
	return s.store.Pin(ctx, channelID, messageID, userID)
}

// Unpin unpins a message.
func (s *Service) Unpin(ctx context.Context, channelID, messageID string) error {
	return s.store.Unpin(ctx, channelID, messageID)
}

// ListPinned lists pinned messages.
func (s *Service) ListPinned(ctx context.Context, channelID string) ([]*Message, error) {
	return s.store.ListPinned(ctx, channelID)
}

// AddReaction adds a reaction.
func (s *Service) AddReaction(ctx context.Context, messageID, userID, emoji string) error {
	return s.store.AddReaction(ctx, messageID, userID, emoji)
}

// RemoveReaction removes a reaction.
func (s *Service) RemoveReaction(ctx context.Context, messageID, userID, emoji string) error {
	return s.store.RemoveReaction(ctx, messageID, userID, emoji)
}

// GetReactionUsers gets users who reacted.
func (s *Service) GetReactionUsers(ctx context.Context, messageID, emoji string, limit int) ([]string, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	return s.store.GetReactionUsers(ctx, messageID, emoji, limit)
}

// CreateAttachment creates an attachment.
func (s *Service) CreateAttachment(ctx context.Context, att *Attachment) error {
	att.ID = ulid.New()
	att.CreatedAt = time.Now()
	return s.store.InsertAttachment(ctx, att)
}

// CreateEmbed creates an embed.
func (s *Service) CreateEmbed(ctx context.Context, messageID string, embed *Embed) error {
	embed.ID = ulid.New()
	return s.store.InsertEmbed(ctx, messageID, embed)
}

// processContent returns the raw content - markdown rendering is handled client-side.
// We only process mentions to preserve user/channel references.
func processContent(content string) string {
	// Return raw content - client-side marked.js handles markdown rendering
	// We escape HTML in the template or client-side to prevent XSS
	return content
}

// extractMentions extracts user IDs from mentions in content.
func extractMentions(content string) []string {
	re := regexp.MustCompile(`<@([A-Za-z0-9]+)>`)
	matches := re.FindAllStringSubmatch(content, -1)

	seen := make(map[string]bool)
	var mentions []string
	for _, m := range matches {
		if len(m) > 1 && !seen[m[1]] {
			seen[m[1]] = true
			mentions = append(mentions, m[1])
		}
	}
	return mentions
}
