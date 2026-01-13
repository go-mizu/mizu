// Package ai_gateway provides AI Gateway management.
package ai_gateway

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("gateway not found")
	ErrNameRequired = errors.New("name is required")
)

// Gateway represents an AI Gateway.
type Gateway struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	CollectLogs      bool      `json:"collect_logs"`
	CacheEnabled     bool      `json:"cache_enabled"`
	CacheTTL         int       `json:"cache_ttl"`
	RateLimitEnabled bool      `json:"rate_limit_enabled"`
	RateLimitCount   int       `json:"rate_limit_count"`
	RateLimitPeriod  int       `json:"rate_limit_period"`
	CreatedAt        time.Time `json:"created_at"`
}

// Log represents a gateway request log.
type Log struct {
	ID        string            `json:"id"`
	GatewayID string            `json:"gateway_id"`
	Provider  string            `json:"provider"`
	Model     string            `json:"model"`
	Cached    bool              `json:"cached"`
	Status    int               `json:"status"`
	Duration  int               `json:"duration_ms"`
	Tokens    int               `json:"tokens"`
	Cost      float64           `json:"cost"`
	Request   []byte            `json:"request"`
	Response  []byte            `json:"response"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// CreateGatewayIn contains input for creating a gateway.
type CreateGatewayIn struct {
	Name             string `json:"name"`
	CollectLogs      bool   `json:"collect_logs"`
	CacheEnabled     bool   `json:"cache_enabled"`
	CacheTTL         int    `json:"cache_ttl"`
	RateLimitEnabled bool   `json:"rate_limit_enabled"`
	RateLimitCount   int    `json:"rate_limit_count"`
	RateLimitPeriod  int    `json:"rate_limit_period"`
}

// UpdateGatewayIn contains input for updating a gateway.
type UpdateGatewayIn struct {
	CollectLogs      *bool `json:"collect_logs,omitempty"`
	CacheEnabled     *bool `json:"cache_enabled,omitempty"`
	CacheTTL         *int  `json:"cache_ttl,omitempty"`
	RateLimitEnabled *bool `json:"rate_limit_enabled,omitempty"`
	RateLimitCount   *int  `json:"rate_limit_count,omitempty"`
	RateLimitPeriod  *int  `json:"rate_limit_period,omitempty"`
}

// API defines the AI Gateway service contract.
type API interface {
	Create(ctx context.Context, in *CreateGatewayIn) (*Gateway, error)
	Get(ctx context.Context, id string) (*Gateway, error)
	List(ctx context.Context) ([]*Gateway, error)
	Update(ctx context.Context, id string, in *UpdateGatewayIn) (*Gateway, error)
	Delete(ctx context.Context, id string) error
	GetLogs(ctx context.Context, gatewayID string, limit, offset int) ([]*Log, error)
}

// Store defines the data access contract.
type Store interface {
	CreateGateway(ctx context.Context, gw *Gateway) error
	GetGateway(ctx context.Context, id string) (*Gateway, error)
	ListGateways(ctx context.Context) ([]*Gateway, error)
	UpdateGateway(ctx context.Context, gw *Gateway) error
	DeleteGateway(ctx context.Context, id string) error
	GetLogs(ctx context.Context, gatewayID string, limit, offset int) ([]*Log, error)
}
