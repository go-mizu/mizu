// Package durable_objects provides Durable Objects management.
package durable_objects

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound      = errors.New("namespace not found")
	ErrNameRequired  = errors.New("name is required")
)

// Namespace represents a Durable Object namespace.
type Namespace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Script    string    `json:"script"`
	ClassName string    `json:"class"`
	CreatedAt time.Time `json:"created_at"`
}

// Instance represents a Durable Object instance.
type Instance struct {
	ID          string    `json:"id"`
	NamespaceID string    `json:"namespace_id"`
	Name        string    `json:"name,omitempty"`
	HasStorage  bool      `json:"has_storage"`
	CreatedAt   time.Time `json:"created_at"`
	LastAccess  time.Time `json:"last_access"`
}

// CreateNamespaceIn contains input for creating a namespace.
type CreateNamespaceIn struct {
	Name      string `json:"name"`
	Script    string `json:"script"`
	ClassName string `json:"class"`
}

// API defines the Durable Objects service contract.
type API interface {
	CreateNamespace(ctx context.Context, in *CreateNamespaceIn) (*Namespace, error)
	GetNamespace(ctx context.Context, id string) (*Namespace, error)
	ListNamespaces(ctx context.Context) ([]*Namespace, error)
	DeleteNamespace(ctx context.Context, id string) error
	ListObjects(ctx context.Context, nsID string) ([]*Instance, error)
}

// Store defines the data access contract.
type Store interface {
	CreateNamespace(ctx context.Context, ns *Namespace) error
	GetNamespace(ctx context.Context, id string) (*Namespace, error)
	ListNamespaces(ctx context.Context) ([]*Namespace, error)
	DeleteNamespace(ctx context.Context, id string) error
	ListInstances(ctx context.Context, nsID string) ([]*Instance, error)
}
