package views

import (
	"context"
	"encoding/json"

	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// Service implements the views API.
type Service struct {
	store Store
}

// NewService creates a new views service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new view.
func (s *Service) Create(ctx context.Context, userID string, in CreateIn) (*View, error) {
	view := &View{
		ID:        ulid.New(),
		TableID:   in.TableID,
		Name:      in.Name,
		Type:      in.Type,
		IsDefault: in.IsDefault,
		CreatedBy: userID,
	}

	if view.Type == "" {
		view.Type = TypeGrid
	}

	if err := s.store.Create(ctx, view); err != nil {
		return nil, err
	}

	return view, nil
}

// GetByID retrieves a view by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*View, error) {
	return s.store.GetByID(ctx, id)
}

// Update updates a view.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*View, error) {
	view, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		view.Name = *in.Name
	}
	if in.IsLocked != nil {
		view.IsLocked = *in.IsLocked
	}

	if err := s.store.Update(ctx, view); err != nil {
		return nil, err
	}

	return view, nil
}

// Delete deletes a view.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Duplicate duplicates a view.
func (s *Service) Duplicate(ctx context.Context, id string, newName string) (*View, error) {
	original, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	view := &View{
		ID:          ulid.New(),
		TableID:     original.TableID,
		Name:        newName,
		Type:        original.Type,
		Config:      original.Config,
		Filters:     original.Filters,
		Sorts:       original.Sorts,
		Groups:      original.Groups,
		FieldConfig: original.FieldConfig,
		IsDefault:   false,
		IsLocked:    false,
		CreatedBy:   original.CreatedBy,
	}

	if err := s.store.Create(ctx, view); err != nil {
		return nil, err
	}

	return view, nil
}

// ListByTable lists all views for a table.
func (s *Service) ListByTable(ctx context.Context, tableID string) ([]*View, error) {
	return s.store.ListByTable(ctx, tableID)
}

// SetFilters sets the filters for a view.
func (s *Service) SetFilters(ctx context.Context, viewID string, filters []Filter) error {
	view, err := s.store.GetByID(ctx, viewID)
	if err != nil {
		return err
	}

	view.Filters = filters
	return s.store.Update(ctx, view)
}

// SetSorts sets the sorts for a view.
func (s *Service) SetSorts(ctx context.Context, viewID string, sorts []SortSpec) error {
	view, err := s.store.GetByID(ctx, viewID)
	if err != nil {
		return err
	}

	view.Sorts = sorts
	return s.store.Update(ctx, view)
}

// SetGroups sets the groups for a view.
func (s *Service) SetGroups(ctx context.Context, viewID string, groups []GroupSpec) error {
	view, err := s.store.GetByID(ctx, viewID)
	if err != nil {
		return err
	}

	view.Groups = groups
	return s.store.Update(ctx, view)
}

// SetFieldConfig sets the field configuration for a view.
func (s *Service) SetFieldConfig(ctx context.Context, viewID string, config []FieldViewConfig) error {
	view, err := s.store.GetByID(ctx, viewID)
	if err != nil {
		return err
	}

	view.FieldConfig = config
	return s.store.Update(ctx, view)
}

// SetConfig sets the view-specific configuration.
func (s *Service) SetConfig(ctx context.Context, viewID string, config map[string]any) error {
	view, err := s.store.GetByID(ctx, viewID)
	if err != nil {
		return err
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	view.Config = configJSON
	return s.store.Update(ctx, view)
}
