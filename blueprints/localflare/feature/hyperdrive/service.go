package hyperdrive

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the Hyperdrive API.
type Service struct {
	store Store
}

// NewService creates a new Hyperdrive service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new config.
func (s *Service) Create(ctx context.Context, in *CreateConfigIn) (*Config, error) {
	if in.Name == "" {
		return nil, ErrNameRequired
	}

	// Apply defaults
	caching := in.Caching
	if caching.MaxAge == 0 {
		caching.MaxAge = 60
	}
	if caching.StaleWhileRevalidate == 0 {
		caching.StaleWhileRevalidate = 15
	}

	origin := in.Origin
	if origin.Scheme == "" {
		origin.Scheme = "postgres"
	}

	cfg := &Config{
		ID:        ulid.Make().String(),
		Name:      in.Name,
		Origin:    origin,
		Caching:   caching,
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateConfig(ctx, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Get retrieves a config by ID.
func (s *Service) Get(ctx context.Context, id string) (*Config, error) {
	cfg, err := s.store.GetConfig(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return cfg, nil
}

// List lists all configs.
func (s *Service) List(ctx context.Context) ([]*Config, error) {
	return s.store.ListConfigs(ctx)
}

// Update updates a config.
func (s *Service) Update(ctx context.Context, id string, in *UpdateConfigIn) (*Config, error) {
	cfg, err := s.store.GetConfig(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	if in.Origin != nil {
		cfg.Origin = *in.Origin
	}
	if in.Caching != nil {
		cfg.Caching = *in.Caching
	}

	if err := s.store.UpdateConfig(ctx, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Delete deletes a config.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.DeleteConfig(ctx, id)
}

// GetStats gets connection pool statistics.
func (s *Service) GetStats(ctx context.Context, id string) (*Stats, error) {
	return s.store.GetStats(ctx, id)
}
