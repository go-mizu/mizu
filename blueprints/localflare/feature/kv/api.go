// Package kv provides KV namespace management.
package kv

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound        = errors.New("namespace not found")
	ErrKeyNotFound     = errors.New("key not found")
	ErrTitleRequired   = errors.New("title is required")
)

// Namespace represents a KV namespace.
type Namespace struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// Pair represents a key-value pair.
type Pair struct {
	Key        string            `json:"key"`
	Value      []byte            `json:"value"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Expiration *time.Time        `json:"expiration,omitempty"`
}

// KeyInfo represents key metadata without value.
type KeyInfo struct {
	Name       string            `json:"name"`
	Expiration *time.Time        `json:"expiration,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// CreateNamespaceIn contains input for creating a namespace.
type CreateNamespaceIn struct {
	Title string `json:"title"`
}

// PutIn contains input for putting a value.
type PutIn struct {
	Key        string            `json:"key"`
	Value      []byte            `json:"value"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Expiration *time.Time        `json:"expiration,omitempty"`
}

// ListOpts specifies options for listing keys.
type ListOpts struct {
	Prefix string
	Limit  int
}

// API defines the KV service contract.
type API interface {
	CreateNamespace(ctx context.Context, in *CreateNamespaceIn) (*Namespace, error)
	GetNamespace(ctx context.Context, id string) (*Namespace, error)
	ListNamespaces(ctx context.Context) ([]*Namespace, error)
	DeleteNamespace(ctx context.Context, id string) error
	Put(ctx context.Context, nsID string, in *PutIn) error
	Get(ctx context.Context, nsID, key string) (*Pair, error)
	Delete(ctx context.Context, nsID, key string) error
	ListKeys(ctx context.Context, nsID string, opts ListOpts) ([]*KeyInfo, error)
}

// Store defines the data access contract.
type Store interface {
	CreateNamespace(ctx context.Context, ns *Namespace) error
	GetNamespace(ctx context.Context, id string) (*Namespace, error)
	ListNamespaces(ctx context.Context) ([]*Namespace, error)
	DeleteNamespace(ctx context.Context, id string) error
	Put(ctx context.Context, nsID string, pair *Pair) error
	Get(ctx context.Context, nsID, key string) (*Pair, error)
	Delete(ctx context.Context, nsID, key string) error
	List(ctx context.Context, nsID, prefix string, limit int) ([]*Pair, error)
}
