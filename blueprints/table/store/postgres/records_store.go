package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// RecordsStore provides PostgreSQL-based record storage.
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
	`, record.ID, record.TableID, cellsJSON, record.Position, record.CreatedBy, record.CreatedAt, record.UpdatedAt, nullString(record.UpdatedBy))
	return err
}

// CreateBatch creates multiple records efficiently using batch insert.
// PostgreSQL-optimized: Uses COPY protocol for large batches when available.
func (s *RecordsStore) CreateBatch(ctx context.Context, recs []*records.Record) error {
	if len(recs) == 0 {
		return nil
	}

	now := time.Now()
	tableID := recs[0].TableID

	// Get max position once for all records
	var maxPos sql.NullInt64
	s.db.QueryRowContext(ctx, `SELECT MAX(position) FROM records WHERE table_id = $1`, tableID).Scan(&maxPos)
	startPos := 0
	if maxPos.Valid {
		startPos = int(maxPos.Int64) + 1
	}

	// Process in batches to avoid query size limits
	// PostgreSQL can handle larger batches than DuckDB
	batchSize := 1000
	for i := 0; i < len(recs); i += batchSize {
		end := i + batchSize
		if end > len(recs) {
			end = len(recs)
		}
		batch := recs[i:end]

		// Build batch insert query with unnest for efficiency
		query := `INSERT INTO records (id, table_id, cells, position, created_by, created_at, updated_at, updated_by) VALUES `
		args := make([]any, 0, len(batch)*8)
		for j, rec := range batch {
			rec.CreatedAt = now
			rec.UpdatedAt = now
			rec.Position = startPos + i + j

			if rec.Cells == nil {
				rec.Cells = make(map[string]any)
			}

			cellsJSON, err := json.Marshal(rec.Cells)
			if err != nil {
				return err
			}

			if j > 0 {
				query += ", "
			}
			paramOffset := j * 8
			query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				paramOffset+1, paramOffset+2, paramOffset+3, paramOffset+4,
				paramOffset+5, paramOffset+6, paramOffset+7, paramOffset+8)
			args = append(args, rec.ID, rec.TableID, cellsJSON, rec.Position,
				rec.CreatedBy, rec.CreatedAt, rec.UpdatedAt, nullString(rec.UpdatedBy))
		}

		if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
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

// GetByIDs retrieves multiple records by IDs efficiently using IN clause.
func (s *RecordsStore) GetByIDs(ctx context.Context, ids []string) (map[string]*records.Record, error) {
	if len(ids) == 0 {
		return make(map[string]*records.Record), nil
	}

	// Build query with placeholders
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, table_id, cells, position, created_by, created_at, updated_at, updated_by
		FROM records WHERE id IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*records.Record)
	for rows.Next() {
		rec, err := s.scanRecordRows(rows)
		if err != nil {
			return nil, err
		}
		result[rec.ID] = rec
	}

	return result, rows.Err()
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
	`, cellsJSON, record.Position, record.UpdatedAt, nullString(record.UpdatedBy), record.ID)
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

// DeleteBatch deletes multiple records efficiently using batch operations.
func (s *RecordsStore) DeleteBatch(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	inClause := strings.Join(placeholders, ", ")

	// Batch delete related data
	_, _ = s.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM comments WHERE record_id IN (%s)`, inClause), args...)
	_, _ = s.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM attachments WHERE record_id IN (%s)`, inClause), args...)
	_, _ = s.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM record_links WHERE source_record_id IN (%s) OR target_record_id IN (%s)`, inClause, inClause), append(args, args...)...)

	// Delete records
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM records WHERE id IN (%s)`, inClause), args...)
	return err
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

// UpdateCell updates a single cell value using PostgreSQL's JSONB operators.
func (s *RecordsStore) UpdateCell(ctx context.Context, recordID, fieldID string, value any) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// Use PostgreSQL's jsonb_set for efficient single-key update
	_, err = s.db.ExecContext(ctx, `
		UPDATE records
		SET cells = jsonb_set(cells, $1, $2::jsonb), updated_at = $3
		WHERE id = $4
	`, "{"+fieldID+"}", valueJSON, time.Now(), recordID)
	return err
}

// ClearCell clears a cell value using PostgreSQL's JSONB operators.
func (s *RecordsStore) ClearCell(ctx context.Context, recordID, fieldID string) error {
	// Use PostgreSQL's - operator to remove a key from JSONB
	_, err := s.db.ExecContext(ctx, `
		UPDATE records
		SET cells = cells - $1, updated_at = $2
		WHERE id = $3
	`, fieldID, time.Now(), recordID)
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

// UpdateCellsBatch updates multiple cell values efficiently using a single transaction.
// Uses PostgreSQL's jsonb_set for efficient atomic updates.
func (s *RecordsStore) UpdateCellsBatch(ctx context.Context, updates []records.CellUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	now := time.Now()

	// Use a transaction for atomicity and better performance
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare statement for reuse
	stmt, err := tx.PrepareContext(ctx, `
		UPDATE records
		SET cells = jsonb_set(cells, $1, $2::jsonb), updated_at = $3
		WHERE id = $4
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, update := range updates {
		valueJSON, err := json.Marshal(update.Value)
		if err != nil {
			return err
		}

		if _, err := stmt.ExecContext(ctx, "{"+update.FieldID+"}", valueJSON, now, update.RecordID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *RecordsStore) scanRecord(row *sql.Row) (*records.Record, error) {
	rec := &records.Record{}
	var cellsJSON []byte
	var updatedBy sql.NullString

	err := row.Scan(&rec.ID, &rec.TableID, &cellsJSON, &rec.Position, &rec.CreatedBy, &rec.CreatedAt, &rec.UpdatedAt, &updatedBy)
	if err == sql.ErrNoRows {
		return nil, records.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// PostgreSQL JSONB returns proper JSON bytes
	if len(cellsJSON) > 0 {
		if err := json.Unmarshal(cellsJSON, &rec.Cells); err != nil {
			rec.Cells = make(map[string]any)
		}
	} else {
		rec.Cells = make(map[string]any)
	}

	if updatedBy.Valid {
		rec.UpdatedBy = updatedBy.String
	}
	return rec, nil
}

func (s *RecordsStore) scanRecordRows(rows *sql.Rows) (*records.Record, error) {
	rec := &records.Record{}
	var cellsJSON []byte
	var updatedBy sql.NullString

	err := rows.Scan(&rec.ID, &rec.TableID, &cellsJSON, &rec.Position, &rec.CreatedBy, &rec.CreatedAt, &rec.UpdatedAt, &updatedBy)
	if err != nil {
		return nil, err
	}

	// PostgreSQL JSONB returns proper JSON bytes
	if len(cellsJSON) > 0 {
		if err := json.Unmarshal(cellsJSON, &rec.Cells); err != nil {
			rec.Cells = make(map[string]any)
		}
	} else {
		rec.Cells = make(map[string]any)
	}

	if updatedBy.Valid {
		rec.UpdatedBy = updatedBy.String
	}
	return rec, nil
}
