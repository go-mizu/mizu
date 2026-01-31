package agent

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/bot/store"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// Service manages AI agents.
type Service struct {
	store store.AgentStore
}

// NewService creates an agent service.
func NewService(s store.AgentStore) *Service {
	return &Service{store: s}
}

func (s *Service) List(ctx context.Context) ([]types.Agent, error) {
	return s.store.ListAgents(ctx)
}

func (s *Service) Get(ctx context.Context, id string) (*types.Agent, error) {
	return s.store.GetAgent(ctx, id)
}

func (s *Service) Create(ctx context.Context, a *types.Agent) error {
	if a.Model == "" {
		a.Model = "claude-sonnet-4-20250514"
	}
	if a.MaxTokens == 0 {
		a.MaxTokens = 4096
	}
	if a.Temperature == 0 {
		a.Temperature = 0.7
	}
	if a.Status == "" {
		a.Status = "active"
	}
	return s.store.CreateAgent(ctx, a)
}

func (s *Service) Update(ctx context.Context, a *types.Agent) error {
	return s.store.UpdateAgent(ctx, a)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.DeleteAgent(ctx, id)
}
