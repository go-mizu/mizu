package durable_objects

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the Durable Objects API.
type Service struct {
	store Store
}

// NewService creates a new Durable Objects service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateNamespace creates a new namespace.
func (s *Service) CreateNamespace(ctx context.Context, in *CreateNamespaceIn) (*Namespace, error) {
	if in.Name == "" {
		return nil, ErrNameRequired
	}

	ns := &Namespace{
		ID:        ulid.Make().String(),
		Name:      in.Name,
		Script:    in.Script,
		ClassName: in.ClassName,
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

// ListObjects lists all instances in a namespace.
func (s *Service) ListObjects(ctx context.Context, nsID string) ([]*Instance, error) {
	return s.store.ListInstances(ctx, nsID)
}
