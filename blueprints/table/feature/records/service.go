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

// UpdateBatch updates multiple records.
func (s *Service) UpdateBatch(ctx context.Context, updates []RecordUpdate, userID string) ([]*Record, error) {
	var results []*Record
	for _, update := range updates {
		record, err := s.Update(ctx, update.ID, update.Cells, userID)
		if err != nil {
			return nil, err
		}
		results = append(results, record)
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

// UpdateFieldValues updates a field value across multiple records.
func (s *Service) UpdateFieldValues(ctx context.Context, tableID, fieldID string, updates map[string]interface{}, userID string) error {
	for recordID, value := range updates {
		if err := s.store.UpdateCell(ctx, recordID, fieldID, value); err != nil {
			return err
		}
	}
	return nil
}
