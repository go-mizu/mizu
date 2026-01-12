package tables

import (
	"context"

	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// Service implements the tables API.
type Service struct {
	store Store
}

// NewService creates a new tables service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new table.
func (s *Service) Create(ctx context.Context, userID string, in CreateIn) (*Table, error) {
	tbl := &Table{
		ID:        ulid.New(),
		BaseID:    in.BaseID,
		Name:      in.Name,
		Description: in.Description,
		Icon:      in.Icon,
		CreatedBy: userID,
	}

	if err := s.store.Create(ctx, tbl); err != nil {
		return nil, err
	}

	return tbl, nil
}

// GetByID retrieves a table by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Table, error) {
	return s.store.GetByID(ctx, id)
}

// Update updates a table.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*Table, error) {
	tbl, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		tbl.Name = *in.Name
	}
	if in.Description != nil {
		tbl.Description = *in.Description
	}
	if in.Icon != nil {
		tbl.Icon = *in.Icon
	}

	if err := s.store.Update(ctx, tbl); err != nil {
		return nil, err
	}

	return tbl, nil
}

// Delete deletes a table.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Duplicate duplicates a table.
func (s *Service) Duplicate(ctx context.Context, id string, newName string) (*Table, error) {
	original, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	tbl := &Table{
		ID:        ulid.New(),
		BaseID:    original.BaseID,
		Name:      newName,
		Description: original.Description,
		Icon:      original.Icon,
		CreatedBy: original.CreatedBy,
	}

	if err := s.store.Create(ctx, tbl); err != nil {
		return nil, err
	}

	return tbl, nil
}

// ListByBase lists all tables in a base.
func (s *Service) ListByBase(ctx context.Context, baseID string) ([]*Table, error) {
	return s.store.ListByBase(ctx, baseID)
}

// Reorder reorders tables efficiently using O(1) map lookups instead of O(n²) nested loops.
func (s *Service) Reorder(ctx context.Context, baseID string, tableIDs []string) error {
	tables, err := s.store.ListByBase(ctx, baseID)
	if err != nil {
		return err
	}

	// Build map for O(1) lookups instead of O(n) search for each tableID
	tableMap := make(map[string]*Table, len(tables))
	for _, tbl := range tables {
		tableMap[tbl.ID] = tbl
	}

	// Update positions with O(n) complexity instead of O(n²)
	for i, tableID := range tableIDs {
		if tbl := tableMap[tableID]; tbl != nil {
			tbl.Position = i
			s.store.Update(ctx, tbl)
		}
	}

	return nil
}

// SetPrimaryField sets the primary field for a table.
func (s *Service) SetPrimaryField(ctx context.Context, tableID, fieldID string) error {
	return s.store.SetPrimaryField(ctx, tableID, fieldID)
}

// NextAutoNumber gets the next auto-number for a table.
func (s *Service) NextAutoNumber(ctx context.Context, tableID string) (int64, error) {
	return s.store.NextAutoNumber(ctx, tableID)
}
