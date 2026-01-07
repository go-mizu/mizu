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

	result := make([]*cells.Cell, 0)
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

	result := make([]*cells.MergedRegion, 0)
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

// GetByPositions retrieves multiple cells by their positions in a single query.
// This avoids N+1 query problems when batch updating cells.
func (s *CellsStore) GetByPositions(ctx context.Context, sheetID string, positions []cells.CellPosition) (map[cells.CellPosition]*cells.Cell, error) {
	if len(positions) == 0 {
		return make(map[cells.CellPosition]*cells.Cell), nil
	}

	// Find bounding box of all positions for efficient range query
	minRow, maxRow := positions[0].Row, positions[0].Row
	minCol, maxCol := positions[0].Col, positions[0].Col
	posSet := make(map[cells.CellPosition]bool, len(positions))

	for _, pos := range positions {
		posSet[pos] = true
		if pos.Row < minRow {
			minRow = pos.Row
		}
		if pos.Row > maxRow {
			maxRow = pos.Row
		}
		if pos.Col < minCol {
			minCol = pos.Col
		}
		if pos.Col > maxCol {
			maxCol = pos.Col
		}
	}

	// Query all cells in the bounding box
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, sheet_id, row_num, col_num, CAST(value AS VARCHAR), formula, display, cell_type,
			CAST(format AS VARCHAR), CAST(hyperlink AS VARCHAR), note, updated_at
		FROM cells
		WHERE sheet_id = ?
			AND row_num >= ? AND row_num <= ?
			AND col_num >= ? AND col_num <= ?
	`, sheetID, minRow, maxRow, minCol, maxCol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[cells.CellPosition]*cells.Cell)
	for rows.Next() {
		cell := &cells.Cell{}
		var value, format, hyperlink sql.NullString
		var note sql.NullString

		if err := rows.Scan(&cell.ID, &cell.SheetID, &cell.Row, &cell.Col,
			&value, &cell.Formula, &cell.Display, &cell.Type,
			&format, &hyperlink, &note, &cell.UpdatedAt); err != nil {
			return nil, err
		}

		pos := cells.CellPosition{Row: cell.Row, Col: cell.Col}
		// Only include cells that were requested
		if !posSet[pos] {
			continue
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

		result[pos] = cell
	}
	return result, nil
}

// DeleteRowsRange deletes multiple rows and shifts remaining cells up in a single operation.
// This is more efficient than calling ShiftRows in a loop.
func (s *CellsStore) DeleteRowsRange(ctx context.Context, sheetID string, startRow, count int) error {
	if count <= 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all rows in the range at once
	endRow := startRow + count - 1
	_, err = tx.ExecContext(ctx, `
		DELETE FROM cells WHERE sheet_id = ? AND row_num >= ? AND row_num <= ?
	`, sheetID, startRow, endRow)
	if err != nil {
		return err
	}

	// Shift remaining rows up by count
	_, err = tx.ExecContext(ctx, `
		UPDATE cells SET row_num = row_num - ?
		WHERE sheet_id = ? AND row_num > ?
	`, count, sheetID, endRow)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteColsRange deletes multiple columns and shifts remaining cells left in a single operation.
// This is more efficient than calling ShiftCols in a loop.
func (s *CellsStore) DeleteColsRange(ctx context.Context, sheetID string, startCol, count int) error {
	if count <= 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all columns in the range at once
	endCol := startCol + count - 1
	_, err = tx.ExecContext(ctx, `
		DELETE FROM cells WHERE sheet_id = ? AND col_num >= ? AND col_num <= ?
	`, sheetID, startCol, endCol)
	if err != nil {
		return err
	}

	// Shift remaining columns left by count
	_, err = tx.ExecContext(ctx, `
		UPDATE cells SET col_num = col_num - ?
		WHERE sheet_id = ? AND col_num > ?
	`, count, sheetID, endCol)
	if err != nil {
		return err
	}

	return tx.Commit()
}
