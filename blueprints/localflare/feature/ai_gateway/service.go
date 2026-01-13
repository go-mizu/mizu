package ai_gateway

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the AI Gateway API.
type Service struct {
	store Store
}

// NewService creates a new AI Gateway service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new gateway.
func (s *Service) Create(ctx context.Context, in *CreateGatewayIn) (*Gateway, error) {
	if in.Name == "" {
		return nil, ErrNameRequired
	}

	cacheTTL := in.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 3600
	}

	rateLimitCount := in.RateLimitCount
	if rateLimitCount == 0 {
		rateLimitCount = 100
	}

	rateLimitPeriod := in.RateLimitPeriod
	if rateLimitPeriod == 0 {
		rateLimitPeriod = 60
	}

	gw := &Gateway{
		ID:               ulid.Make().String(),
		Name:             in.Name,
		CollectLogs:      in.CollectLogs,
		CacheEnabled:     in.CacheEnabled,
		CacheTTL:         cacheTTL,
		RateLimitEnabled: in.RateLimitEnabled,
		RateLimitCount:   rateLimitCount,
		RateLimitPeriod:  rateLimitPeriod,
		CreatedAt:        time.Now(),
	}

	if err := s.store.CreateGateway(ctx, gw); err != nil {
		return nil, err
	}

	return gw, nil
}

// Get retrieves a gateway by ID.
func (s *Service) Get(ctx context.Context, id string) (*Gateway, error) {
	gw, err := s.store.GetGateway(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return gw, nil
}

// List lists all gateways.
func (s *Service) List(ctx context.Context) ([]*Gateway, error) {
	return s.store.ListGateways(ctx)
}

// Update updates a gateway.
func (s *Service) Update(ctx context.Context, id string, in *UpdateGatewayIn) (*Gateway, error) {
	gw, err := s.store.GetGateway(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	if in.CollectLogs != nil {
		gw.CollectLogs = *in.CollectLogs
	}
	if in.CacheEnabled != nil {
		gw.CacheEnabled = *in.CacheEnabled
	}
	if in.CacheTTL != nil {
		gw.CacheTTL = *in.CacheTTL
	}
	if in.RateLimitEnabled != nil {
		gw.RateLimitEnabled = *in.RateLimitEnabled
	}
	if in.RateLimitCount != nil {
		gw.RateLimitCount = *in.RateLimitCount
	}
	if in.RateLimitPeriod != nil {
		gw.RateLimitPeriod = *in.RateLimitPeriod
	}

	if err := s.store.UpdateGateway(ctx, gw); err != nil {
		return nil, err
	}

	return gw, nil
}

// Delete deletes a gateway.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.DeleteGateway(ctx, id)
}

// GetLogs gets gateway logs.
func (s *Service) GetLogs(ctx context.Context, gatewayID string, limit, offset int) ([]*Log, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.store.GetLogs(ctx, gatewayID, limit, offset)
}
