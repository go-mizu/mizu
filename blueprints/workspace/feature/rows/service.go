package rows

import (
	"context"
	"errors"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

var (
	ErrNotFound = errors.New("row not found")
)

const DefaultLimit = 100

// Service implements the rows API.
// It uses pages with database_id as the underlying storage.
type Service struct {
	store Store
	pages pages.API
}

// NewService creates a new rows service.
func NewService(store Store, pages pages.API) *Service {
	return &Service{store: store, pages: pages}
}

// Create creates a new row in a database.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Row, error) {
	props := in.Properties
	if props == nil {
		props = make(map[string]interface{})
	}

	// Convert row properties to page properties
	pageProps := make(pages.Properties)
	for k, v := range props {
		pageProps[k] = pages.PropertyValue{
			Type:  "unknown", // Type will be determined by database schema
			Value: v,
		}
	}

	// Extract title from properties if present
	title := ""
	if t, ok := props["title"].(string); ok {
		title = t
	}

	// Create a page with database_id set (this makes it a row)
	page, err := s.pages.Create(ctx, &pages.CreateIn{
		WorkspaceID: in.WorkspaceID,
		ParentType:  pages.ParentDatabase,
		ParentID:    in.DatabaseID,
		DatabaseID:  in.DatabaseID,
		Title:       title,
		Properties:  pageProps,
		CreatedBy:   in.CreatedBy,
	})
	if err != nil {
		return nil, err
	}

	return PageToRow(page), nil
}

// GetByID retrieves a row by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Row, error) {
	row, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return row, nil
}

// Update updates a row's properties.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Row, error) {
	existing, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	// Merge properties
	for k, v := range in.Properties {
		existing.Properties[k] = v
	}

	updateIn := &UpdateIn{
		Properties: existing.Properties,
		UpdatedBy:  in.UpdatedBy,
	}

	if err := s.store.Update(ctx, id, updateIn); err != nil {
		return nil, err
	}

	return s.store.GetByID(ctx, id)
}

// Delete deletes a row.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// List lists rows in a database with optional filtering and sorting.
func (s *Service) List(ctx context.Context, in *ListIn) (*ListResult, error) {
	limit := in.Limit
	if limit <= 0 || limit > DefaultLimit {
		limit = DefaultLimit
	}

	// Request one more than limit to detect if there are more results
	listIn := &ListIn{
		DatabaseID: in.DatabaseID,
		Filters:    in.Filters,
		Sorts:      in.Sorts,
		Limit:      limit + 1,
		Cursor:     in.Cursor,
	}

	rows, err := s.store.List(ctx, listIn)
	if err != nil {
		return nil, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	var nextCursor string
	if hasMore && len(rows) > 0 {
		nextCursor = rows[len(rows)-1].ID
	}

	return &ListResult{
		Rows:       rows,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// DeleteByDatabase deletes all rows in a database.
func (s *Service) DeleteByDatabase(ctx context.Context, databaseID string) error {
	return s.store.DeleteByDatabase(ctx, databaseID)
}

// DuplicateRow creates a copy of a row.
func (s *Service) DuplicateRow(ctx context.Context, id string, userID string) (*Row, error) {
	original, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	// Copy properties
	newProps := make(map[string]interface{})
	for k, v := range original.Properties {
		newProps[k] = v
	}

	// Append "(copy)" to title if it exists
	if title, ok := newProps["title"].(string); ok {
		newProps["title"] = title + " (copy)"
	}

	return s.Create(ctx, &CreateIn{
		DatabaseID:  original.DatabaseID,
		WorkspaceID: original.WorkspaceID,
		Properties:  newProps,
		CreatedBy:   userID,
	})
}
