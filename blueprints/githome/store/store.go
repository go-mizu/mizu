// Package store defines the unified storage interface for GitHome.
// This interface can be implemented by multiple drivers (duckdb, postgres, etc.)
package store

import (
	"context"
	"io"

	"github.com/go-mizu/blueprints/githome/feature/git"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Store is the main interface for all GitHome data access operations.
// It provides access to all feature-specific stores through getter methods.
type Store interface {
	// Lifecycle
	io.Closer

	// Ensure initializes the database schema
	Ensure(ctx context.Context) error

	// Feature stores
	Users() users.Store
	Repos() repos.Store
	Git() git.Store

	// Stats returns storage statistics
	Stats(ctx context.Context) (map[string]any, error)
}

// Config holds configuration for creating a store
type Config struct {
	// Driver specifies the storage driver (duckdb, postgres, sqlite, etc.)
	Driver string

	// DSN is the data source name/connection string
	// For duckdb: file path or "" for in-memory
	// For postgres: "postgres://user:pass@host/db"
	DSN string

	// ReposDir is the base directory for git repository storage
	ReposDir string
}

// Option is a functional option for store configuration
type Option func(*Config)

// WithDriver sets the storage driver
func WithDriver(driver string) Option {
	return func(c *Config) {
		c.Driver = driver
	}
}

// WithDSN sets the data source name
func WithDSN(dsn string) Option {
	return func(c *Config) {
		c.DSN = dsn
	}
}

// WithReposDir sets the git repositories directory
func WithReposDir(dir string) Option {
	return func(c *Config) {
		c.ReposDir = dir
	}
}

// NewConfig creates a Config with the given options
func NewConfig(opts ...Option) *Config {
	cfg := &Config{
		Driver: "duckdb",
		DSN:    "",
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
