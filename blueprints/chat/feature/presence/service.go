package presence

import (
	"context"
	"sync"
	"time"
)

// Service implements the presence API.
type Service struct {
	store Store

	// In-memory typing indicators
	typing     map[string]map[string]time.Time // channelID -> userID -> timestamp
	typingLock sync.RWMutex
}

// NewService creates a new presence service.
func NewService(store Store) *Service {
	return &Service{
		store:  store,
		typing: make(map[string]map[string]time.Time),
	}
}

// Update updates a user's presence.
func (s *Service) Update(ctx context.Context, userID string, in *UpdateIn) (*Presence, error) {
	p, err := s.store.Get(ctx, userID)
	if err != nil {
		p = &Presence{
			UserID: userID,
			Status: StatusOnline,
		}
	}

	if in.Status != nil {
		p.Status = *in.Status
	}
	if in.CustomStatus != nil {
		p.CustomStatus = *in.CustomStatus
	}
	if in.Activities != nil {
		p.Activities = *in.Activities
	}
	p.LastSeenAt = time.Now()

	if err := s.store.Upsert(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}

// Get retrieves a user's presence.
func (s *Service) Get(ctx context.Context, userID string) (*Presence, error) {
	return s.store.Get(ctx, userID)
}

// GetBulk retrieves presence for multiple users.
func (s *Service) GetBulk(ctx context.Context, userIDs []string) ([]*Presence, error) {
	return s.store.GetBulk(ctx, userIDs)
}

// SetOnline sets a user online.
func (s *Service) SetOnline(ctx context.Context, userID string) error {
	return s.store.UpdateStatus(ctx, userID, StatusOnline)
}

// SetOffline sets a user offline.
func (s *Service) SetOffline(ctx context.Context, userID string) error {
	return s.store.SetOffline(ctx, userID)
}

// SetIdle sets a user idle.
func (s *Service) SetIdle(ctx context.Context, userID string) error {
	return s.store.UpdateStatus(ctx, userID, StatusIdle)
}

// Heartbeat updates the last seen timestamp.
func (s *Service) Heartbeat(ctx context.Context, userID string) error {
	p := &Presence{
		UserID:     userID,
		Status:     StatusOnline,
		LastSeenAt: time.Now(),
	}
	return s.store.Upsert(ctx, p)
}

// StartTyping records that a user is typing.
func (s *Service) StartTyping(ctx context.Context, userID, channelID string) error {
	s.typingLock.Lock()
	defer s.typingLock.Unlock()

	if s.typing[channelID] == nil {
		s.typing[channelID] = make(map[string]time.Time)
	}
	s.typing[channelID][userID] = time.Now()

	return nil
}

// GetTypingUsers returns users currently typing in a channel.
func (s *Service) GetTypingUsers(channelID string) []string {
	s.typingLock.RLock()
	defer s.typingLock.RUnlock()

	now := time.Now()
	cutoff := now.Add(-10 * time.Second) // Typing expires after 10 seconds

	var users []string
	if channelTyping, ok := s.typing[channelID]; ok {
		for userID, timestamp := range channelTyping {
			if timestamp.After(cutoff) {
				users = append(users, userID)
			}
		}
	}

	return users
}

// CleanupTyping removes stale typing indicators.
func (s *Service) CleanupTyping() {
	s.typingLock.Lock()
	defer s.typingLock.Unlock()

	cutoff := time.Now().Add(-10 * time.Second)

	for channelID, channelTyping := range s.typing {
		for userID, timestamp := range channelTyping {
			if timestamp.Before(cutoff) {
				delete(channelTyping, userID)
			}
		}
		if len(channelTyping) == 0 {
			delete(s.typing, channelID)
		}
	}
}

// CleanupStale cleans up stale presence entries.
func (s *Service) CleanupStale(ctx context.Context) error {
	// Mark users offline if they haven't sent a heartbeat in 5 minutes
	cutoff := time.Now().Add(-5 * time.Minute)
	return s.store.CleanupStale(ctx, cutoff)
}

// IsOnline checks if a user is online.
func (s *Service) IsOnline(ctx context.Context, userID string) bool {
	p, err := s.store.Get(ctx, userID)
	if err != nil {
		return false
	}
	return p.Status != StatusOffline && p.Status != StatusInvisible
}
