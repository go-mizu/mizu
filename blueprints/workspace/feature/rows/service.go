package rows

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrNotFound = errors.New("row not found")
)

const DefaultLimit = 100

// Service implements the rows API.
type Service struct {
	store Store
}

// NewService creates a new rows service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new row in a database.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Row, error) {
	now := time.Now()

	props := in.Properties
	if props == nil {
		props = make(map[string]interface{})
	}

	row := &Row{
		ID:         ulid.New(),
		DatabaseID: in.DatabaseID,
		Properties: props,
		CreatedBy:  in.CreatedBy,
		CreatedAt:  now,
		UpdatedBy:  in.CreatedBy,
		UpdatedAt:  now,
	}

	if err := s.store.Create(ctx, row); err != nil {
		return nil, err
	}

	return row, nil
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
		DatabaseID: original.DatabaseID,
		Properties: newProps,
		CreatedBy:  userID,
	})
}
