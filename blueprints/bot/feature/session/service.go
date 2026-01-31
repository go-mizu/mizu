package session

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/bot/store"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// Service manages conversation sessions.
type Service struct {
	store store.SessionStore
}

// NewService creates a session service.
func NewService(s store.SessionStore) *Service {
	return &Service{store: s}
}

func (s *Service) List(ctx context.Context) ([]types.Session, error) {
	return s.store.ListSessions(ctx)
}

func (s *Service) Get(ctx context.Context, id string) (*types.Session, error) {
	return s.store.GetSession(ctx, id)
}

func (s *Service) GetOrCreate(ctx context.Context, agentID, channelID, channelType, peerID, displayName, origin string) (*types.Session, error) {
	return s.store.GetOrCreateSession(ctx, agentID, channelID, channelType, peerID, displayName, origin)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.DeleteSession(ctx, id)
}

// Reset marks a session as expired and creates a fresh one.
func (s *Service) Reset(ctx context.Context, id string) error {
	sess, err := s.store.GetSession(ctx, id)
	if err != nil {
		return err
	}
	sess.Status = "expired"
	return s.store.UpdateSession(ctx, sess)
}

// ExpireIdle expires sessions that have been idle for the given number of minutes.
func (s *Service) ExpireIdle(ctx context.Context, idleMinutes int) (int, error) {
	return s.store.ExpireSessions(ctx, "idle", idleMinutes)
}

// ExpireDaily expires sessions created before today.
func (s *Service) ExpireDaily(ctx context.Context) (int, error) {
	return s.store.ExpireSessions(ctx, "daily", 0)
}
