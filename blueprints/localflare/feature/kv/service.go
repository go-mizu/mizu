package kv

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the KV API.
type Service struct {
	store Store
}

// NewService creates a new KV service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateNamespace creates a new KV namespace.
func (s *Service) CreateNamespace(ctx context.Context, in *CreateNamespaceIn) (*Namespace, error) {
	if in.Title == "" {
		return nil, ErrTitleRequired
	}

	ns := &Namespace{
		ID:        ulid.Make().String(),
		Title:     in.Title,
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateNamespace(ctx, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

// GetNamespace retrieves a namespace by ID.
func (s *Service) GetNamespace(ctx context.Context, id string) (*Namespace, error) {
	ns, err := s.store.GetNamespace(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return ns, nil
}

// ListNamespaces lists all namespaces.
func (s *Service) ListNamespaces(ctx context.Context) ([]*Namespace, error) {
	return s.store.ListNamespaces(ctx)
}

// DeleteNamespace deletes a namespace.
func (s *Service) DeleteNamespace(ctx context.Context, id string) error {
	return s.store.DeleteNamespace(ctx, id)
}

// Put stores a key-value pair.
func (s *Service) Put(ctx context.Context, nsID string, in *PutIn) error {
	pair := &Pair{
		Key:        in.Key,
		Value:      in.Value,
		Metadata:   in.Metadata,
		Expiration: in.Expiration,
	}
	return s.store.Put(ctx, nsID, pair)
}

// Get retrieves a value by key.
func (s *Service) Get(ctx context.Context, nsID, key string) (*Pair, error) {
	pair, err := s.store.Get(ctx, nsID, key)
	if err != nil {
		return nil, ErrKeyNotFound
	}
	return pair, nil
}

// Delete removes a key.
func (s *Service) Delete(ctx context.Context, nsID, key string) error {
	return s.store.Delete(ctx, nsID, key)
}

// ListKeys lists keys in a namespace.
func (s *Service) ListKeys(ctx context.Context, nsID string, opts ListOpts) ([]*KeyInfo, error) {
	if opts.Limit <= 0 {
		opts.Limit = 1000
	}

	pairs, err := s.store.List(ctx, nsID, opts.Prefix, opts.Limit)
	if err != nil {
		return nil, err
	}

	keys := make([]*KeyInfo, len(pairs))
	for i, pair := range pairs {
		keys[i] = &KeyInfo{
			Name:       pair.Key,
			Expiration: pair.Expiration,
			Metadata:   pair.Metadata,
		}
	}

	return keys, nil
}
