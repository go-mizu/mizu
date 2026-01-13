package ai

import (
	"context"
)

// Service implements the AI API.
type Service struct {
	store Store
}

// NewService creates a new AI service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// ListModels lists available models.
func (s *Service) ListModels(ctx context.Context, task string) ([]*Model, error) {
	return s.store.ListModels(ctx, task)
}

// GetModel retrieves a model by name.
func (s *Service) GetModel(ctx context.Context, name string) (*Model, error) {
	model, err := s.store.GetModel(ctx, name)
	if err != nil {
		return nil, ErrModelNotFound
	}
	return model, nil
}

// RunModel runs inference on a model.
func (s *Service) RunModel(ctx context.Context, modelName string, in *RunModelIn) (*InferenceResult, error) {
	return s.store.Run(ctx, modelName, in.Inputs, in.Options)
}
