package views

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrNotFound = errors.New("view not found")
)

// Service implements the views API.
type Service struct {
	store Store
	pages pages.API
}

// NewService creates a new views service.
func NewService(store Store, pages pages.API) *Service {
	return &Service{store: store, pages: pages}
}

// Create creates a new view.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*View, error) {
	// Get position
	views, _ := s.store.ListByDatabase(ctx, in.DatabaseID)
	position := len(views)

	view := &View{
		ID:         ulid.New(),
		DatabaseID: in.DatabaseID,
		Name:       in.Name,
		Type:       in.Type,
		Filter:     in.Filter,
		Sorts:      in.Sorts,
		GroupBy:    in.GroupBy,
		CalendarBy: in.CalendarBy,
		Position:   position,
		CreatedBy:  in.CreatedBy,
		CreatedAt:  time.Now(),
	}

	if view.Name == "" {
		view.Name = string(view.Type)
	}

	if view.Type == "" {
		view.Type = ViewTable
	}

	if err := s.store.Create(ctx, view); err != nil {
		return nil, err
	}

	return view, nil
}

// GetByID retrieves a view by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*View, error) {
	view, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return view, nil
}

// Update updates a view.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*View, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// Delete deletes a view.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListByDatabase lists views for a database.
func (s *Service) ListByDatabase(ctx context.Context, databaseID string) ([]*View, error) {
	return s.store.ListByDatabase(ctx, databaseID)
}

// Reorder reorders views.
func (s *Service) Reorder(ctx context.Context, databaseID string, viewIDs []string) error {
	return s.store.Reorder(ctx, databaseID, viewIDs)
}

// Duplicate creates a copy of a view.
func (s *Service) Duplicate(ctx context.Context, id string, userID string) (*View, error) {
	original, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	return s.Create(ctx, &CreateIn{
		DatabaseID: original.DatabaseID,
		Name:       original.Name + " (copy)",
		Type:       original.Type,
		Filter:     original.Filter,
		Sorts:      original.Sorts,
		GroupBy:    original.GroupBy,
		CalendarBy: original.CalendarBy,
		CreatedBy:  userID,
	})
}

// Query queries a view and returns matching pages.
func (s *Service) Query(ctx context.Context, viewID string, cursor string, limit int) (*QueryResult, error) {
	view, err := s.store.GetByID(ctx, viewID)
	if err != nil {
		return nil, ErrNotFound
	}

	if limit <= 0 {
		limit = 50
	}

	// Get all pages for the database (as database items)
	items, err := s.pages.ListByParent(ctx, view.DatabaseID, pages.ParentDatabase)
	if err != nil {
		return nil, err
	}

	// Apply filter if present
	if view.Filter != nil {
		items = s.applyFilter(items, view.Filter)
	}

	// Apply sorting if present
	if len(view.Sorts) > 0 {
		items = s.applySort(items, view.Sorts)
	}

	// Apply pagination
	start := 0
	if cursor != "" {
		for i, item := range items {
			if item.ID == cursor {
				start = i + 1
				break
			}
		}
	}

	end := start + limit
	if end > len(items) {
		end = len(items)
	}

	result := &QueryResult{
		Items:   items[start:end],
		HasMore: end < len(items),
	}

	if result.HasMore && len(result.Items) > 0 {
		result.NextCursor = result.Items[len(result.Items)-1].ID
	}

	return result, nil
}

// applyFilter filters pages based on the filter configuration.
func (s *Service) applyFilter(items []*pages.Page, filter *Filter) []*pages.Page {
	// For now, return all items. A full implementation would evaluate filter conditions.
	return items
}

// applySort sorts pages based on the sort configuration.
func (s *Service) applySort(items []*pages.Page, sorts []Sort) []*pages.Page {
	// For now, return items as-is. A full implementation would sort based on properties.
	return items
}
