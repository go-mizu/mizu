package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// RecordsStore provides DuckDB-based record storage.
type RecordsStore struct {
	db *sql.DB
}

// NewRecordsStore creates a new records store.
func NewRecordsStore(db *sql.DB) *RecordsStore {
	return &RecordsStore{db: db}
}

// Create creates a new record.
func (s *RecordsStore) Create(ctx context.Context, record *records.Record) error {
	now := time.Now()
	record.CreatedAt = now
	record.UpdatedAt = now

	if record.Cells == nil {
		record.Cells = make(map[string]any)
	}

	// Get max position
	var maxPos sql.NullInt64
	s.db.QueryRowContext(ctx, `SELECT MAX(position) FROM records WHERE table_id = $1`, record.TableID).Scan(&maxPos)
	if maxPos.Valid {
		record.Position = int(maxPos.Int64) + 1
	}

	cellsJSON, err := json.Marshal(record.Cells)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO records (id, table_id, cells, position, created_by, created_at, updated_at, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, record.ID, record.TableID, string(cellsJSON), record.Position, record.CreatedBy, record.CreatedAt, record.UpdatedAt, record.UpdatedBy)
	return err
}

// CreateBatch creates multiple records.
func (s *RecordsStore) CreateBatch(ctx context.Context, recs []*records.Record) error {
	for _, rec := range recs {
		if err := s.Create(ctx, rec); err != nil {
			return err
		}
	}
	return nil
}

// GetByID retrieves a record by ID.
func (s *RecordsStore) GetByID(ctx context.Context, id string) (*records.Record, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, table_id, cells, position, created_by, created_at, updated_at, updated_by
		FROM records WHERE id = $1
	`, id)
	return s.scanRecord(row)
}

// GetByIDs retrieves multiple records by IDs.
func (s *RecordsStore) GetByIDs(ctx context.Context, ids []string) (map[string]*records.Record, error) {
	if len(ids) == 0 {
		return make(map[string]*records.Record), nil
	}

	result := make(map[string]*records.Record)
	for _, id := range ids {
		rec, err := s.GetByID(ctx, id)
		if err == records.ErrNotFound {
			continue
		}
		if err != nil {
			return nil, err
		}
		result[id] = rec
	}
	return result, nil
}

// Update updates a record.
func (s *RecordsStore) Update(ctx context.Context, record *records.Record) error {
	record.UpdatedAt = time.Now()

	cellsJSON, err := json.Marshal(record.Cells)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE records SET
			cells = $1, position = $2, updated_at = $3, updated_by = $4
		WHERE id = $5
	`, string(cellsJSON), record.Position, record.UpdatedAt, record.UpdatedBy, record.ID)
	return err
}

// Delete deletes a record.
func (s *RecordsStore) Delete(ctx context.Context, id string) error {
	// Delete related data
	_, _ = s.db.ExecContext(ctx, `DELETE FROM comments WHERE record_id = $1`, id)
	_, _ = s.db.ExecContext(ctx, `DELETE FROM attachments WHERE record_id = $1`, id)
	_, _ = s.db.ExecContext(ctx, `DELETE FROM record_links WHERE source_record_id = $1 OR target_record_id = $1`, id)

	_, err := s.db.ExecContext(ctx, `DELETE FROM records WHERE id = $1`, id)
	return err
}

// DeleteBatch deletes multiple records.
func (s *RecordsStore) DeleteBatch(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := s.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// List lists records in a table.
func (s *RecordsStore) List(ctx context.Context, tableID string, opts records.ListOpts) (*records.RecordList, error) {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}

	// Get total count
	var total int
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM records WHERE table_id = $1`, tableID).Scan(&total)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, table_id, cells, position, created_by, created_at, updated_at, updated_by
		FROM records WHERE table_id = $1
		ORDER BY position ASC
		LIMIT $2 OFFSET $3
	`, tableID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recordList []*records.Record
	for rows.Next() {
		rec, err := s.scanRecordRows(rows)
		if err != nil {
			return nil, err
		}
		recordList = append(recordList, rec)
	}

	return &records.RecordList{
		Records: recordList,
		Offset:  opts.Offset,
		Total:   total,
	}, rows.Err()
}

// UpdateCell updates a single cell value.
func (s *RecordsStore) UpdateCell(ctx context.Context, recordID, fieldID string, value any) error {
	rec, err := s.GetByID(ctx, recordID)
	if err != nil {
		return err
	}

	rec.Cells[fieldID] = value
	rec.UpdatedAt = time.Now()

	cellsJSON, err := json.Marshal(rec.Cells)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE records SET cells = $1, updated_at = $2 WHERE id = $3
	`, string(cellsJSON), rec.UpdatedAt, recordID)
	return err
}

// ClearCell clears a cell value.
func (s *RecordsStore) ClearCell(ctx context.Context, recordID, fieldID string) error {
	rec, err := s.GetByID(ctx, recordID)
	if err != nil {
		return err
	}

	delete(rec.Cells, fieldID)
	rec.UpdatedAt = time.Now()

	cellsJSON, err := json.Marshal(rec.Cells)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE records SET cells = $1, updated_at = $2 WHERE id = $3
	`, string(cellsJSON), rec.UpdatedAt, recordID)
	return err
}

// CreateLink creates a record link.
func (s *RecordsStore) CreateLink(ctx context.Context, link *records.RecordLink) error {
	if link.ID == "" {
		link.ID = ulid.New()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO record_links (id, source_record_id, source_field_id, target_record_id, position)
		VALUES ($1, $2, $3, $4, $5)
	`, link.ID, link.SourceRecordID, link.SourceFieldID, link.TargetRecordID, link.Position)
	return err
}

// DeleteLink deletes a record link.
func (s *RecordsStore) DeleteLink(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM record_links WHERE id = $1`, id)
	return err
}

// DeleteLinksBySource deletes all links from a source record/field.
func (s *RecordsStore) DeleteLinksBySource(ctx context.Context, recordID, fieldID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM record_links WHERE source_record_id = $1 AND source_field_id = $2
	`, recordID, fieldID)
	return err
}

// ListLinksBySource lists links by source record/field.
func (s *RecordsStore) ListLinksBySource(ctx context.Context, recordID, fieldID string) ([]*records.RecordLink, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, source_record_id, source_field_id, target_record_id, position
		FROM record_links WHERE source_record_id = $1 AND source_field_id = $2
		ORDER BY position ASC
	`, recordID, fieldID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*records.RecordLink
	for rows.Next() {
		link := &records.RecordLink{}
		if err := rows.Scan(&link.ID, &link.SourceRecordID, &link.SourceFieldID, &link.TargetRecordID, &link.Position); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

// ListLinksByTarget lists links by target record.
func (s *RecordsStore) ListLinksByTarget(ctx context.Context, targetRecordID string) ([]*records.RecordLink, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, source_record_id, source_field_id, target_record_id, position
		FROM record_links WHERE target_record_id = $1
	`, targetRecordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*records.RecordLink
	for rows.Next() {
		link := &records.RecordLink{}
		if err := rows.Scan(&link.ID, &link.SourceRecordID, &link.SourceFieldID, &link.TargetRecordID, &link.Position); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

func (s *RecordsStore) scanRecord(row *sql.Row) (*records.Record, error) {
	rec := &records.Record{}
	var cells any
	var updatedBy sql.NullString

	err := row.Scan(&rec.ID, &rec.TableID, &cells, &rec.Position, &rec.CreatedBy, &rec.CreatedAt, &rec.UpdatedAt, &updatedBy)
	if err == sql.ErrNoRows {
		return nil, records.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Handle both string and map types from DuckDB JSON
	switch v := cells.(type) {
	case map[string]any:
		rec.Cells = v
	case string:
		if err := json.Unmarshal([]byte(v), &rec.Cells); err != nil {
			rec.Cells = make(map[string]any)
		}
	default:
		rec.Cells = make(map[string]any)
	}
	if updatedBy.Valid {
		rec.UpdatedBy = updatedBy.String
	}
	return rec, nil
}

func (s *RecordsStore) scanRecordRows(rows *sql.Rows) (*records.Record, error) {
	rec := &records.Record{}
	var cells any
	var updatedBy sql.NullString

	err := rows.Scan(&rec.ID, &rec.TableID, &cells, &rec.Position, &rec.CreatedBy, &rec.CreatedAt, &rec.UpdatedAt, &updatedBy)
	if err != nil {
		return nil, err
	}

	// Handle both string and map types from DuckDB JSON
	switch v := cells.(type) {
	case map[string]any:
		rec.Cells = v
	case string:
		if err := json.Unmarshal([]byte(v), &rec.Cells); err != nil {
			rec.Cells = make(map[string]any)
		}
	default:
		rec.Cells = make(map[string]any)
	}
	if updatedBy.Valid {
		rec.UpdatedBy = updatedBy.String
	}
	return rec, nil
}
