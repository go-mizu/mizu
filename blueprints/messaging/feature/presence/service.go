package presence

import (
	"context"
	"sync"
	"time"
)

// Service implements the presence API.
type Service struct {
	store   Store
	typing  map[string]map[string]*TypingState // chatID -> userID -> state
	typingM sync.RWMutex
}

// NewService creates a new presence service.
func NewService(store Store) *Service {
	return &Service{
		store:  store,
		typing: make(map[string]map[string]*TypingState),
	}
}

// Get retrieves a user's presence.
func (s *Service) Get(ctx context.Context, userID string) (*Presence, error) {
	return s.store.Get(ctx, userID)
}

// GetMany retrieves presence for multiple users.
func (s *Service) GetMany(ctx context.Context, userIDs []string) ([]*Presence, error) {
	return s.store.GetMany(ctx, userIDs)
}

// SetOnline sets a user as online.
func (s *Service) SetOnline(ctx context.Context, userID string) error {
	p := &Presence{
		UserID:     userID,
		Status:     StatusOnline,
		LastSeenAt: time.Now(),
	}
	return s.store.Upsert(ctx, p)
}

// SetOffline sets a user as offline.
func (s *Service) SetOffline(ctx context.Context, userID string) error {
	if err := s.store.UpdateLastSeen(ctx, userID); err != nil {
		return err
	}
	return s.store.UpdateStatus(ctx, userID, StatusOffline)
}

// SetCustomStatus sets a user's custom status.
func (s *Service) SetCustomStatus(ctx context.Context, userID, status string) error {
	return s.store.UpdateCustomStatus(ctx, userID, status)
}

// StartTyping starts typing indicator.
func (s *Service) StartTyping(ctx context.Context, userID, chatID string) error {
	s.typingM.Lock()
	defer s.typingM.Unlock()

	if s.typing[chatID] == nil {
		s.typing[chatID] = make(map[string]*TypingState)
	}

	s.typing[chatID][userID] = &TypingState{
		UserID:    userID,
		ChatID:    chatID,
		StartedAt: time.Now(),
	}

	return nil
}

// StopTyping stops typing indicator.
func (s *Service) StopTyping(ctx context.Context, userID, chatID string) error {
	s.typingM.Lock()
	defer s.typingM.Unlock()

	if s.typing[chatID] != nil {
		delete(s.typing[chatID], userID)
		if len(s.typing[chatID]) == 0 {
			delete(s.typing, chatID)
		}
	}

	return nil
}

// GetTyping gets typing indicators for a chat.
func (s *Service) GetTyping(ctx context.Context, chatID string) ([]*TypingState, error) {
	s.typingM.RLock()
	defer s.typingM.RUnlock()

	var states []*TypingState
	now := time.Now()
	timeout := 10 * time.Second

	if chatUsers, ok := s.typing[chatID]; ok {
		for _, state := range chatUsers {
			// Filter out stale typing indicators
			if now.Sub(state.StartedAt) < timeout {
				states = append(states, state)
			}
		}
	}

	return states, nil
}

// CleanupTyping removes stale typing indicators.
func (s *Service) CleanupTyping() {
	s.typingM.Lock()
	defer s.typingM.Unlock()

	now := time.Now()
	timeout := 10 * time.Second

	for chatID, chatUsers := range s.typing {
		for userID, state := range chatUsers {
			if now.Sub(state.StartedAt) >= timeout {
				delete(chatUsers, userID)
			}
		}
		if len(chatUsers) == 0 {
			delete(s.typing, chatID)
		}
	}
}
