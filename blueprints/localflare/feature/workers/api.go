// Package workers provides worker management.
package workers

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound   = errors.New("worker not found")
	ErrNameExists = errors.New("worker name already exists")
)

// Worker represents a Cloudflare Worker.
type Worker struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Script    string            `json:"script"`
	Routes    []string          `json:"routes"`
	Bindings  map[string]string `json:"bindings"`
	Enabled   bool              `json:"enabled"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Route represents a worker route.
type Route struct {
	ID       string `json:"id"`
	ZoneID   string `json:"zone_id"`
	Pattern  string `json:"pattern"`
	WorkerID string `json:"worker_id"`
	Enabled  bool   `json:"enabled"`
}

// CreateIn contains input for creating a worker.
type CreateIn struct {
	Name     string            `json:"name"`
	Script   string            `json:"script"`
	Routes   []string          `json:"routes"`
	Bindings map[string]string `json:"bindings"`
	Enabled  bool              `json:"enabled"`
}

// UpdateIn contains input for updating a worker.
type UpdateIn struct {
	Name     *string            `json:"name,omitempty"`
	Script   *string            `json:"script,omitempty"`
	Routes   []string           `json:"routes,omitempty"`
	Bindings map[string]string  `json:"bindings,omitempty"`
	Enabled  *bool              `json:"enabled,omitempty"`
}

// CreateRouteIn contains input for creating a route.
type CreateRouteIn struct {
	ZoneID   string `json:"zone_id"`
	Pattern  string `json:"pattern"`
	WorkerID string `json:"worker_id"`
	Enabled  bool   `json:"enabled"`
}

// LogEntry represents a worker log entry.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// DeployResult represents the result of a worker deployment.
type DeployResult struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	DeployedAt time.Time `json:"deployed_at"`
}

// API defines the workers service contract.
type API interface {
	Create(ctx context.Context, in *CreateIn) (*Worker, error)
	GetByID(ctx context.Context, id string) (*Worker, error)
	List(ctx context.Context) ([]*Worker, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Worker, error)
	Delete(ctx context.Context, id string) error
	Deploy(ctx context.Context, id string) (*DeployResult, error)
	Logs(ctx context.Context, id string) ([]*LogEntry, error)
	CreateRoute(ctx context.Context, in *CreateRouteIn) (*Route, error)
	ListRoutes(ctx context.Context, zoneID string) ([]*Route, error)
	DeleteRoute(ctx context.Context, id string) error
}

// Store defines the data access contract.
type Store interface {
	Create(ctx context.Context, worker *Worker) error
	GetByID(ctx context.Context, id string) (*Worker, error)
	GetByName(ctx context.Context, name string) (*Worker, error)
	List(ctx context.Context) ([]*Worker, error)
	Update(ctx context.Context, worker *Worker) error
	Delete(ctx context.Context, id string) error
	CreateRoute(ctx context.Context, route *Route) error
	ListRoutes(ctx context.Context, zoneID string) ([]*Route, error)
	DeleteRoute(ctx context.Context, id string) error
}
