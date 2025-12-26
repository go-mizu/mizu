package fields

import (
	"context"
	"errors"
	"strings"

	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrKeyExists = errors.New("field key already exists")
	ErrNotFound  = errors.New("field not found")
)

// Service implements the fields API.
type Service struct {
	store Store
}

// NewService creates a new fields service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, projectID string, in *CreateIn) (*Field, error) {
	key := strings.ToLower(in.Key)

	// Check if key exists in project
	existing, err := s.store.GetByKey(ctx, projectID, key)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrKeyExists
	}

	// Get existing fields to determine position
	fields, err := s.store.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	position := in.Position
	if position == 0 && len(fields) > 0 {
		position = len(fields)
	}

	field := &Field{
		ID:           ulid.New(),
		ProjectID:    projectID,
		Key:          key,
		Name:         in.Name,
		Kind:         in.Kind,
		Position:     position,
		IsRequired:   in.IsRequired,
		IsArchived:   false,
		SettingsJSON: in.SettingsJSON,
	}

	if err := s.store.Create(ctx, field); err != nil {
		return nil, err
	}

	return field, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Field, error) {
	f, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, ErrNotFound
	}
	return f, nil
}

func (s *Service) GetByKey(ctx context.Context, projectID, key string) (*Field, error) {
	f, err := s.store.GetByKey(ctx, projectID, strings.ToLower(key))
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, ErrNotFound
	}
	return f, nil
}

func (s *Service) ListByProject(ctx context.Context, projectID string) ([]*Field, error) {
	return s.store.ListByProject(ctx, projectID)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Field, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) UpdatePosition(ctx context.Context, id string, position int) error {
	return s.store.UpdatePosition(ctx, id, position)
}

func (s *Service) Archive(ctx context.Context, id string) error {
	return s.store.Archive(ctx, id)
}

func (s *Service) Unarchive(ctx context.Context, id string) error {
	return s.store.Unarchive(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
