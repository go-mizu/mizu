// Package session manages AI research sessions with conversation history.
package session

import (
	"context"
	"encoding/json"
	"time"
)

// Session represents an AI research session.
type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages,omitempty"`
}

// Message represents a message in a session.
type Message struct {
	ID        string     `json:"id"`
	SessionID string     `json:"session_id"`
	Role      string     `json:"role"` // user, assistant
	Content   string     `json:"content"`
	Citations []Citation `json:"citations,omitempty"`
	Mode      string     `json:"mode,omitempty"` // quick, deep, research
	CreatedAt time.Time  `json:"created_at"`
}

// Citation represents a source citation in a message.
type Citation struct {
	Index        int    `json:"index"`
	URL          string `json:"url"`
	Title        string `json:"title"`
	Snippet      string `json:"snippet"`
	Domain       string `json:"domain,omitempty"`        // Source domain for badge display
	Favicon      string `json:"favicon,omitempty"`       // Source favicon URL
	OtherSources int    `json:"other_sources,omitempty"` // Count of other sources (for "+N" display)
}

// Store defines the interface for session storage.
type Store interface {
	// Create creates a new session.
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by ID.
	Get(ctx context.Context, id string) (*Session, error)

	// List lists sessions with pagination.
	List(ctx context.Context, limit, offset int) ([]Session, int, error)

	// Update updates a session.
	Update(ctx context.Context, session *Session) error

	// Delete deletes a session.
	Delete(ctx context.Context, id string) error

	// AddMessage adds a message to a session.
	AddMessage(ctx context.Context, sessionID string, msg *Message) error

	// GetMessages retrieves messages for a session.
	GetMessages(ctx context.Context, sessionID string) ([]Message, error)
}

// Service manages sessions.
type Service struct {
	store Store
}

// New creates a new session service.
func New(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new session.
func (s *Service) Create(ctx context.Context, title string) (*Session, error) {
	session := &Session{
		ID:        generateID(),
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if title == "" {
		session.Title = "New Research Session"
	}

	if err := s.store.Create(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// Get retrieves a session by ID with messages.
func (s *Service) Get(ctx context.Context, id string) (*Session, error) {
	session, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	messages, err := s.store.GetMessages(ctx, id)
	if err != nil {
		return nil, err
	}
	session.Messages = messages

	return session, nil
}

// List lists sessions with pagination.
func (s *Service) List(ctx context.Context, limit, offset int) ([]Session, int, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.store.List(ctx, limit, offset)
}

// Update updates a session.
func (s *Service) Update(ctx context.Context, session *Session) error {
	session.UpdatedAt = time.Now()
	return s.store.Update(ctx, session)
}

// Delete deletes a session.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// AddMessage adds a message to a session.
func (s *Service) AddMessage(ctx context.Context, sessionID string, role, content, mode string, citations []Citation) (*Message, error) {
	msg := &Message{
		ID:        generateID(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Mode:      mode,
		Citations: citations,
		CreatedAt: time.Now(),
	}

	if err := s.store.AddMessage(ctx, sessionID, msg); err != nil {
		return nil, err
	}

	// Update session timestamp
	session, err := s.store.Get(ctx, sessionID)
	if err == nil {
		session.UpdatedAt = time.Now()
		// Auto-generate title from first user message if still default
		if session.Title == "New Research Session" && role == "user" {
			session.Title = truncateTitle(content)
		}
		_ = s.store.Update(ctx, session)
	}

	return msg, nil
}

// GetConversationContext returns messages formatted for LLM context.
func (s *Service) GetConversationContext(ctx context.Context, sessionID string) ([]map[string]string, error) {
	messages, err := s.store.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	context := make([]map[string]string, 0, len(messages))
	for _, msg := range messages {
		context = append(context, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	return context, nil
}

// MarshalCitations converts citations to JSON.
func MarshalCitations(citations []Citation) string {
	if len(citations) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(citations)
	return string(data)
}

// UnmarshalCitations parses citations from JSON.
func UnmarshalCitations(data string) []Citation {
	if data == "" || data == "null" {
		return nil
	}
	var citations []Citation
	_ = json.Unmarshal([]byte(data), &citations)
	return citations
}

func truncateTitle(s string) string {
	if len(s) > 50 {
		return s[:47] + "..."
	}
	return s
}
