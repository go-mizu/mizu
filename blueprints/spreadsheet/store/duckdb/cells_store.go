package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
)

// CellsStore implements cells.Store.
type CellsStore struct {
	db *sql.DB
}

// NewCellsStore creates a new cells store.
func NewCellsStore(db *sql.DB) *CellsStore {
	return &CellsStore{db: db}
}

// Get retrieves a cell by position.
func (s *CellsStore) Get(ctx context.Context, sheetID string, row, col int) (*cells.Cell, error) {
	cell := &cells.Cell{}
	var value, format, hyperlink sql.NullString
	var note sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, sheet_id, row_num, col_num, CAST(value AS VARCHAR), formula, display, cell_type,
			CAST(format AS VARCHAR), CAST(hyperlink AS VARCHAR), note, updated_at
		FROM cells WHERE sheet_id = ? AND row_num = ? AND col_num = ?
	`, sheetID, row, col).Scan(&cell.ID, &cell.SheetID, &cell.Row, &cell.Col,
		&value, &cell.Formula, &cell.Display, &cell.Type,
		&format, &hyperlink, &note, &cell.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, cells.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if value.Valid {
		json.Unmarshal([]byte(value.String), &cell.Value)
	}
	if format.Valid {
		json.Unmarshal([]byte(format.String), &cell.Format)
	}
	if hyperlink.Valid {
		json.Unmarshal([]byte(hyperlink.String), &cell.Hyperlink)
	}
	if note.Valid {
		cell.Note = note.String
	}

	return cell, nil
}

// GetRange retrieves cells in a range.
func (s *CellsStore) GetRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([]*cells.Cell, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, sheet_id, row_num, col_num, CAST(value AS VARCHAR), formula, display, cell_type,
			CAST(format AS VARCHAR), CAST(hyperlink AS VARCHAR), note, updated_at
		FROM cells
		WHERE sheet_id = ?
			AND row_num >= ? AND row_num <= ?
			AND col_num >= ? AND col_num <= ?
		ORDER BY row_num, col_num
	`, sheetID, startRow, endRow, startCol, endCol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*cells.Cell
	for rows.Next() {
		cell := &cells.Cell{}
		var value, format, hyperlink sql.NullString
		var note sql.NullString

		if err := rows.Scan(&cell.ID, &cell.SheetID, &cell.Row, &cell.Col,
			&value, &cell.Formula, &cell.Display, &cell.Type,
			&format, &hyperlink, &note, &cell.UpdatedAt); err != nil {
			return nil, err
		}

		if value.Valid {
			json.Unmarshal([]byte(value.String), &cell.Value)
		}
		if format.Valid {
			json.Unmarshal([]byte(format.String), &cell.Format)
		}
		if hyperlink.Valid {
			json.Unmarshal([]byte(hyperlink.String), &cell.Hyperlink)
		}
		if note.Valid {
			cell.Note = note.String
		}

		result = append(result, cell)
	}
	return result, nil
}

// Set sets a cell.
func (s *CellsStore) Set(ctx context.Context, cell *cells.Cell) error {
	value, _ := json.Marshal(cell.Value)
	format, _ := json.Marshal(cell.Format)
	hyperlink, _ := json.Marshal(cell.Hyperlink)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cells (id, sheet_id, row_num, col_num, value, formula, display, cell_type,
			format, hyperlink, note, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (sheet_id, row_num, col_num) DO UPDATE SET
			value = excluded.value,
			formula = excluded.formula,
			display = excluded.display,
			cell_type = excluded.cell_type,
			format = excluded.format,
			hyperlink = excluded.hyperlink,
			note = excluded.note,
			updated_at = excluded.updated_at
	`, cell.ID, cell.SheetID, cell.Row, cell.Col, string(value), cell.Formula,
		cell.Display, cell.Type, string(format), string(hyperlink), cell.Note, cell.UpdatedAt)
	return err
}

// BatchSet sets multiple cells.
func (s *CellsStore) BatchSet(ctx context.Context, cellList []*cells.Cell) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO cells (id, sheet_id, row_num, col_num, value, formula, display, cell_type,
			format, hyperlink, note, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (sheet_id, row_num, col_num) DO UPDATE SET
			value = excluded.value,
			formula = excluded.formula,
			display = excluded.display,
			cell_type = excluded.cell_type,
			format = excluded.format,
			hyperlink = excluded.hyperlink,
			note = excluded.note,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, cell := range cellList {
		value, _ := json.Marshal(cell.Value)
		format, _ := json.Marshal(cell.Format)
		hyperlink, _ := json.Marshal(cell.Hyperlink)

		_, err := stmt.ExecContext(ctx, cell.ID, cell.SheetID, cell.Row, cell.Col,
			string(value), cell.Formula, cell.Display, cell.Type,
			string(format), string(hyperlink), cell.Note, cell.UpdatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Delete deletes a cell.
func (s *CellsStore) Delete(ctx context.Context, sheetID string, row, col int) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM cells WHERE sheet_id = ? AND row_num = ? AND col_num = ?
	`, sheetID, row, col)
	return err
}

// DeleteRange deletes cells in a range.
func (s *CellsStore) DeleteRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM cells
		WHERE sheet_id = ?
			AND row_num >= ? AND row_num <= ?
			AND col_num >= ? AND col_num <= ?
	`, sheetID, startRow, endRow, startCol, endCol)
	return err
}

// CreateMerge creates a merged region.
func (s *CellsStore) CreateMerge(ctx context.Context, region *cells.MergedRegion) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO merged_regions (id, sheet_id, start_row, start_col, end_row, end_col)
		VALUES (?, ?, ?, ?, ?, ?)
	`, region.ID, region.SheetID, region.StartRow, region.StartCol, region.EndRow, region.EndCol)
	return err
}

// DeleteMerge deletes a merged region.
func (s *CellsStore) DeleteMerge(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM merged_regions
		WHERE sheet_id = ? AND start_row = ? AND start_col = ? AND end_row = ? AND end_col = ?
	`, sheetID, startRow, startCol, endRow, endCol)
	return err
}

// GetMergedRegions retrieves merged regions for a sheet.
func (s *CellsStore) GetMergedRegions(ctx context.Context, sheetID string) ([]*cells.MergedRegion, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, sheet_id, start_row, start_col, end_row, end_col
		FROM merged_regions WHERE sheet_id = ?
	`, sheetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*cells.MergedRegion
	for rows.Next() {
		region := &cells.MergedRegion{}
		if err := rows.Scan(&region.ID, &region.SheetID, &region.StartRow,
			&region.StartCol, &region.EndRow, &region.EndCol); err != nil {
			return nil, err
		}
		result = append(result, region)
	}
	return result, nil
}

// ShiftRows shifts rows (for insert/delete operations).
func (s *CellsStore) ShiftRows(ctx context.Context, sheetID string, startRow, count int) error {
	if count > 0 {
		// Insert - shift rows down
		_, err := s.db.ExecContext(ctx, `
			UPDATE cells SET row_num = row_num + ?
			WHERE sheet_id = ? AND row_num >= ?
		`, count, sheetID, startRow)
		return err
	} else {
		// Delete - remove rows then shift up
		_, err := s.db.ExecContext(ctx, `
			DELETE FROM cells WHERE sheet_id = ? AND row_num = ?
		`, sheetID, startRow)
		if err != nil {
			return err
		}
		_, err = s.db.ExecContext(ctx, `
			UPDATE cells SET row_num = row_num - 1
			WHERE sheet_id = ? AND row_num > ?
		`, sheetID, startRow)
		return err
	}
}

// ShiftCols shifts columns (for insert/delete operations).
func (s *CellsStore) ShiftCols(ctx context.Context, sheetID string, startCol, count int) error {
	if count > 0 {
		// Insert - shift columns right
		_, err := s.db.ExecContext(ctx, `
			UPDATE cells SET col_num = col_num + ?
			WHERE sheet_id = ? AND col_num >= ?
		`, count, sheetID, startCol)
		return err
	} else {
		// Delete - remove columns then shift left
		_, err := s.db.ExecContext(ctx, `
			DELETE FROM cells WHERE sheet_id = ? AND col_num = ?
		`, sheetID, startCol)
		if err != nil {
			return err
		}
		_, err = s.db.ExecContext(ctx, `
			UPDATE cells SET col_num = col_num - 1
			WHERE sheet_id = ? AND col_num > ?
		`, sheetID, startCol)
		return err
	}
}
