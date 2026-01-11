package records

import (
	"context"

	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// Service implements the records API.
type Service struct {
	store Store
}

// NewService creates a new records service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new record.
func (s *Service) Create(ctx context.Context, tableID string, cells map[string]interface{}, userID string) (*Record, error) {
	record := &Record{
		ID:        ulid.New(),
		TableID:   tableID,
		Cells:     cells,
		CreatedBy: userID,
		UpdatedBy: userID,
	}

	if record.Cells == nil {
		record.Cells = make(map[string]interface{})
	}

	if err := s.store.Create(ctx, record); err != nil {
		return nil, err
	}

	return record, nil
}

// CreateBatch creates multiple records.
func (s *Service) CreateBatch(ctx context.Context, tableID string, recordsData []map[string]interface{}, userID string) ([]*Record, error) {
	var records []*Record
	for _, data := range recordsData {
		record := &Record{
			ID:        ulid.New(),
			TableID:   tableID,
			Cells:     data,
			CreatedBy: userID,
			UpdatedBy: userID,
		}
		if record.Cells == nil {
			record.Cells = make(map[string]interface{})
		}
		records = append(records, record)
	}

	if err := s.store.CreateBatch(ctx, records); err != nil {
		return nil, err
	}

	return records, nil
}

// GetByID retrieves a record by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Record, error) {
	return s.store.GetByID(ctx, id)
}

// GetByIDs retrieves multiple records by IDs.
func (s *Service) GetByIDs(ctx context.Context, ids []string) (map[string]*Record, error) {
	return s.store.GetByIDs(ctx, ids)
}

// Update updates a record.
func (s *Service) Update(ctx context.Context, id string, cells map[string]interface{}, userID string) (*Record, error) {
	record, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Merge cells
	if record.Cells == nil {
		record.Cells = make(map[string]interface{})
	}
	for k, v := range cells {
		if v == nil {
			delete(record.Cells, k)
		} else {
			record.Cells[k] = v
		}
	}
	record.UpdatedBy = userID

	if err := s.store.Update(ctx, record); err != nil {
		return nil, err
	}

	return record, nil
}

// UpdateBatch updates multiple records efficiently using batch operations.
// Pre-allocates result slice and uses batch cell updates for better performance.
func (s *Service) UpdateBatch(ctx context.Context, updates []RecordUpdate, userID string) ([]*Record, error) {
	if len(updates) == 0 {
		return nil, nil
	}

	// Pre-allocate results slice with known capacity
	results := make([]*Record, 0, len(updates))

	// Collect all record IDs for batch fetch
	ids := make([]string, len(updates))
	for i, update := range updates {
		ids[i] = update.ID
	}

	// Batch fetch all records at once (1 query instead of N)
	recordMap, err := s.store.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Collect all cell updates for batch operation
	var cellUpdates []CellUpdate
	for _, update := range updates {
		record, ok := recordMap[update.ID]
		if !ok {
			return nil, ErrNotFound
		}

		// Merge cells and collect updates
		if record.Cells == nil {
			record.Cells = make(map[string]interface{})
		}
		for k, v := range update.Cells {
			if v == nil {
				delete(record.Cells, k)
			} else {
				record.Cells[k] = v
				cellUpdates = append(cellUpdates, CellUpdate{
					RecordID: record.ID,
					FieldID:  k,
					Value:    v,
				})
			}
		}
		record.UpdatedBy = userID
		results = append(results, record)
	}

	// Batch update all cells at once (1 transaction instead of N queries)
	if len(cellUpdates) > 0 {
		if err := s.store.UpdateCellsBatch(ctx, cellUpdates); err != nil {
			return nil, err
		}
	}

	return results, nil
}

// Delete deletes a record.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// DeleteBatch deletes multiple records.
func (s *Service) DeleteBatch(ctx context.Context, ids []string) error {
	return s.store.DeleteBatch(ctx, ids)
}

// List lists records with options.
func (s *Service) List(ctx context.Context, tableID string, opts ListOpts) (*RecordList, error) {
	return s.store.List(ctx, tableID, opts)
}

// Search searches records.
func (s *Service) Search(ctx context.Context, tableID, query string, opts ListOpts) (*RecordList, error) {
	// For now, just list all records (search is done client-side)
	return s.store.List(ctx, tableID, opts)
}

// UpdateCell updates a single cell value.
func (s *Service) UpdateCell(ctx context.Context, recordID, fieldID string, value interface{}, userID string) error {
	return s.store.UpdateCell(ctx, recordID, fieldID, value)
}

// ClearCell clears a cell value.
func (s *Service) ClearCell(ctx context.Context, recordID, fieldID string, userID string) error {
	return s.store.ClearCell(ctx, recordID, fieldID)
}

// UpdateFieldValues updates a field value across multiple records efficiently.
// Uses batch cell update for better performance (1 transaction instead of N queries).
func (s *Service) UpdateFieldValues(ctx context.Context, tableID, fieldID string, updates map[string]interface{}, userID string) error {
	if len(updates) == 0 {
		return nil
	}

	// Convert map to CellUpdate slice for batch operation
	cellUpdates := make([]CellUpdate, 0, len(updates))
	for recordID, value := range updates {
		cellUpdates = append(cellUpdates, CellUpdate{
			RecordID: recordID,
			FieldID:  fieldID,
			Value:    value,
		})
	}

	// Batch update all cells at once
	return s.store.UpdateCellsBatch(ctx, cellUpdates)
}
