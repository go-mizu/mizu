// Package hyperdrive provides Hyperdrive configuration management.
package hyperdrive

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("config not found")
	ErrNameRequired = errors.New("name is required")
)

// Config represents a Hyperdrive configuration.
type Config struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Origin    Origin    `json:"origin"`
	Caching   Caching   `json:"caching"`
	CreatedAt time.Time `json:"created_at"`
}

// Origin defines the database connection.
type Origin struct {
	Database string `json:"database"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Scheme   string `json:"scheme"`
	User     string `json:"user"`
	Password string `json:"-"`
}

// Caching configures query caching.
type Caching struct {
	Disabled             bool `json:"disabled"`
	MaxAge               int  `json:"max_age"`
	StaleWhileRevalidate int  `json:"stale_while_revalidate"`
}

// Stats contains connection pool statistics.
type Stats struct {
	ActiveConnections int     `json:"active_connections"`
	IdleConnections   int     `json:"idle_connections"`
	TotalConnections  int     `json:"total_connections"`
	QueriesPerSecond  float64 `json:"queries_per_second"`
	CacheHitRate      float64 `json:"cache_hit_rate"`
}

// CreateConfigIn contains input for creating a config.
type CreateConfigIn struct {
	Name     string  `json:"name"`
	Origin   Origin  `json:"origin"`
	Caching  Caching `json:"caching"`
}

// UpdateConfigIn contains input for updating a config.
type UpdateConfigIn struct {
	Origin  *Origin  `json:"origin,omitempty"`
	Caching *Caching `json:"caching,omitempty"`
}

// API defines the Hyperdrive service contract.
type API interface {
	Create(ctx context.Context, in *CreateConfigIn) (*Config, error)
	Get(ctx context.Context, id string) (*Config, error)
	List(ctx context.Context) ([]*Config, error)
	Update(ctx context.Context, id string, in *UpdateConfigIn) (*Config, error)
	Delete(ctx context.Context, id string) error
	GetStats(ctx context.Context, id string) (*Stats, error)
}

// Store defines the data access contract.
type Store interface {
	CreateConfig(ctx context.Context, cfg *Config) error
	GetConfig(ctx context.Context, id string) (*Config, error)
	ListConfigs(ctx context.Context) ([]*Config, error)
	UpdateConfig(ctx context.Context, cfg *Config) error
	DeleteConfig(ctx context.Context, id string) error
	GetStats(ctx context.Context, configID string) (*Stats, error)
}
