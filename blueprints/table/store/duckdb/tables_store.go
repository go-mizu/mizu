package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/table/feature/tables"
)

// TablesStore provides DuckDB-based table storage.
type TablesStore struct {
	db *sql.DB
}

// NewTablesStore creates a new tables store.
func NewTablesStore(db *sql.DB) *TablesStore {
	return &TablesStore{db: db}
}

// Create creates a new table.
func (s *TablesStore) Create(ctx context.Context, tbl *tables.Table) error {
	now := time.Now()
	tbl.CreatedAt = now
	tbl.UpdatedAt = now

	// Get max position
	var maxPos sql.NullInt64
	s.db.QueryRowContext(ctx, `SELECT MAX(position) FROM tables WHERE base_id = $1`, tbl.BaseID).Scan(&maxPos)
	if maxPos.Valid {
		tbl.Position = int(maxPos.Int64) + 1
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tables (id, base_id, name, description, icon, position, primary_field_id, auto_number_seq, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, tbl.ID, tbl.BaseID, tbl.Name, tbl.Description, tbl.Icon, tbl.Position, tbl.PrimaryFieldID, tbl.AutoNumberSeq, tbl.CreatedBy, tbl.CreatedAt, tbl.UpdatedAt)
	return err
}

// GetByID retrieves a table by ID.
func (s *TablesStore) GetByID(ctx context.Context, id string) (*tables.Table, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, base_id, name, description, icon, position, primary_field_id, auto_number_seq, created_by, created_at, updated_at
		FROM tables WHERE id = $1
	`, id)
	return s.scanTable(row)
}

// Update updates a table.
func (s *TablesStore) Update(ctx context.Context, tbl *tables.Table) error {
	tbl.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		UPDATE tables SET
			name = $1, description = $2, icon = $3, position = $4, primary_field_id = $5, auto_number_seq = $6, updated_at = $7
		WHERE id = $8
	`, tbl.Name, tbl.Description, tbl.Icon, tbl.Position, tbl.PrimaryFieldID, tbl.AutoNumberSeq, tbl.UpdatedAt, tbl.ID)
	return err
}

// Delete deletes a table and all related data.
func (s *TablesStore) Delete(ctx context.Context, id string) error {
	// Delete related data
	_, _ = s.db.ExecContext(ctx, `DELETE FROM views WHERE table_id = $1`, id)
	_, _ = s.db.ExecContext(ctx, `DELETE FROM records WHERE table_id = $1`, id)
	_, _ = s.db.ExecContext(ctx, `DELETE FROM fields WHERE table_id = $1`, id)

	_, err := s.db.ExecContext(ctx, `DELETE FROM tables WHERE id = $1`, id)
	return err
}

// ListByBase lists all tables in a base.
func (s *TablesStore) ListByBase(ctx context.Context, baseID string) ([]*tables.Table, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, base_id, name, description, icon, position, primary_field_id, auto_number_seq, created_by, created_at, updated_at
		FROM tables WHERE base_id = $1
		ORDER BY position ASC
	`, baseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tableList []*tables.Table
	for rows.Next() {
		tbl, err := s.scanTableRows(rows)
		if err != nil {
			return nil, err
		}
		tableList = append(tableList, tbl)
	}
	return tableList, rows.Err()
}

// SetPrimaryField sets the primary field for a table.
func (s *TablesStore) SetPrimaryField(ctx context.Context, tableID, fieldID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE tables SET primary_field_id = $1, updated_at = $2 WHERE id = $3
	`, fieldID, time.Now(), tableID)
	return err
}

// NextAutoNumber gets and increments the auto number sequence.
func (s *TablesStore) NextAutoNumber(ctx context.Context, tableID string) (int64, error) {
	var seq int64
	err := s.db.QueryRowContext(ctx, `
		UPDATE tables SET auto_number_seq = auto_number_seq + 1
		WHERE id = $1
		RETURNING auto_number_seq
	`, tableID).Scan(&seq)
	return seq, err
}

func (s *TablesStore) scanTable(row *sql.Row) (*tables.Table, error) {
	tbl := &tables.Table{}
	var description, icon, primaryFieldID sql.NullString

	err := row.Scan(&tbl.ID, &tbl.BaseID, &tbl.Name, &description, &icon, &tbl.Position, &primaryFieldID, &tbl.AutoNumberSeq, &tbl.CreatedBy, &tbl.CreatedAt, &tbl.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, tables.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if description.Valid {
		tbl.Description = description.String
	}
	if icon.Valid {
		tbl.Icon = icon.String
	}
	if primaryFieldID.Valid {
		tbl.PrimaryFieldID = primaryFieldID.String
	}
	return tbl, nil
}

func (s *TablesStore) scanTableRows(rows *sql.Rows) (*tables.Table, error) {
	tbl := &tables.Table{}
	var description, icon, primaryFieldID sql.NullString

	err := rows.Scan(&tbl.ID, &tbl.BaseID, &tbl.Name, &description, &icon, &tbl.Position, &primaryFieldID, &tbl.AutoNumberSeq, &tbl.CreatedBy, &tbl.CreatedAt, &tbl.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		tbl.Description = description.String
	}
	if icon.Valid {
		tbl.Icon = icon.String
	}
	if primaryFieldID.Valid {
		tbl.PrimaryFieldID = primaryFieldID.String
	}
	return tbl, nil
}
