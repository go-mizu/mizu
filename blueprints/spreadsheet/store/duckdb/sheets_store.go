package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
)

// SheetsStore implements sheets.Store.
type SheetsStore struct {
	db *sql.DB
}

// NewSheetsStore creates a new sheets store.
func NewSheetsStore(db *sql.DB) *SheetsStore {
	return &SheetsStore{db: db}
}

// Create creates a new sheet.
func (s *SheetsStore) Create(ctx context.Context, sheet *sheets.Sheet) error {
	rowHeights, _ := json.Marshal(sheet.RowHeights)
	colWidths, _ := json.Marshal(sheet.ColWidths)
	hiddenRows, _ := json.Marshal(sheet.HiddenRows)
	hiddenCols, _ := json.Marshal(sheet.HiddenCols)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sheets (id, workbook_id, name, index_num, hidden, color, grid_color,
			frozen_rows, frozen_cols, default_row_height, default_col_width,
			row_heights, col_widths, hidden_rows, hidden_cols, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sheet.ID, sheet.WorkbookID, sheet.Name, sheet.Index, sheet.Hidden, sheet.Color,
		sheet.GridColor, sheet.FrozenRows, sheet.FrozenCols, sheet.DefaultRowHeight,
		sheet.DefaultColWidth, string(rowHeights), string(colWidths),
		string(hiddenRows), string(hiddenCols), sheet.CreatedAt, sheet.UpdatedAt)
	return err
}

// GetByID retrieves a sheet by ID.
func (s *SheetsStore) GetByID(ctx context.Context, id string) (*sheets.Sheet, error) {
	sheet := &sheets.Sheet{}
	var rowHeights, colWidths, hiddenRows, hiddenCols sql.NullString
	var color sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, workbook_id, name, index_num, hidden, color, grid_color,
			frozen_rows, frozen_cols, default_row_height, default_col_width,
			CAST(row_heights AS VARCHAR), CAST(col_widths AS VARCHAR),
			CAST(hidden_rows AS VARCHAR), CAST(hidden_cols AS VARCHAR), created_at, updated_at
		FROM sheets WHERE id = ?
	`, id).Scan(&sheet.ID, &sheet.WorkbookID, &sheet.Name, &sheet.Index, &sheet.Hidden,
		&color, &sheet.GridColor, &sheet.FrozenRows, &sheet.FrozenCols,
		&sheet.DefaultRowHeight, &sheet.DefaultColWidth, &rowHeights, &colWidths,
		&hiddenRows, &hiddenCols, &sheet.CreatedAt, &sheet.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, sheets.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if color.Valid {
		sheet.Color = color.String
	}

	if rowHeights.Valid {
		json.Unmarshal([]byte(rowHeights.String), &sheet.RowHeights)
	}
	if colWidths.Valid {
		json.Unmarshal([]byte(colWidths.String), &sheet.ColWidths)
	}
	if hiddenRows.Valid {
		json.Unmarshal([]byte(hiddenRows.String), &sheet.HiddenRows)
	}
	if hiddenCols.Valid {
		json.Unmarshal([]byte(hiddenCols.String), &sheet.HiddenCols)
	}

	if sheet.RowHeights == nil {
		sheet.RowHeights = make(map[int]int)
	}
	if sheet.ColWidths == nil {
		sheet.ColWidths = make(map[int]int)
	}
	if sheet.HiddenRows == nil {
		sheet.HiddenRows = []int{}
	}
	if sheet.HiddenCols == nil {
		sheet.HiddenCols = []int{}
	}

	return sheet, nil
}

// ListByWorkbook lists sheets in a workbook.
func (s *SheetsStore) ListByWorkbook(ctx context.Context, workbookID string) ([]*sheets.Sheet, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workbook_id, name, index_num, hidden, color, grid_color,
			frozen_rows, frozen_cols, default_row_height, default_col_width,
			CAST(row_heights AS VARCHAR), CAST(col_widths AS VARCHAR),
			CAST(hidden_rows AS VARCHAR), CAST(hidden_cols AS VARCHAR), created_at, updated_at
		FROM sheets WHERE workbook_id = ?
		ORDER BY index_num
	`, workbookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*sheets.Sheet, 0)
	for rows.Next() {
		sheet := &sheets.Sheet{}
		var rowHeights, colWidths, hiddenRows, hiddenCols sql.NullString
		var color sql.NullString

		if err := rows.Scan(&sheet.ID, &sheet.WorkbookID, &sheet.Name, &sheet.Index,
			&sheet.Hidden, &color, &sheet.GridColor, &sheet.FrozenRows, &sheet.FrozenCols,
			&sheet.DefaultRowHeight, &sheet.DefaultColWidth, &rowHeights, &colWidths,
			&hiddenRows, &hiddenCols, &sheet.CreatedAt, &sheet.UpdatedAt); err != nil {
			return nil, err
		}

		if color.Valid {
			sheet.Color = color.String
		}

		if rowHeights.Valid {
			json.Unmarshal([]byte(rowHeights.String), &sheet.RowHeights)
		}
		if colWidths.Valid {
			json.Unmarshal([]byte(colWidths.String), &sheet.ColWidths)
		}
		if hiddenRows.Valid {
			json.Unmarshal([]byte(hiddenRows.String), &sheet.HiddenRows)
		}
		if hiddenCols.Valid {
			json.Unmarshal([]byte(hiddenCols.String), &sheet.HiddenCols)
		}

		if sheet.RowHeights == nil {
			sheet.RowHeights = make(map[int]int)
		}
		if sheet.ColWidths == nil {
			sheet.ColWidths = make(map[int]int)
		}

		result = append(result, sheet)
	}
	return result, nil
}

// Update updates a sheet.
func (s *SheetsStore) Update(ctx context.Context, sheet *sheets.Sheet) error {
	rowHeights, _ := json.Marshal(sheet.RowHeights)
	colWidths, _ := json.Marshal(sheet.ColWidths)
	hiddenRows, _ := json.Marshal(sheet.HiddenRows)
	hiddenCols, _ := json.Marshal(sheet.HiddenCols)

	_, err := s.db.ExecContext(ctx, `
		UPDATE sheets SET name = ?, index_num = ?, hidden = ?, color = ?, grid_color = ?,
			frozen_rows = ?, frozen_cols = ?, default_row_height = ?, default_col_width = ?,
			row_heights = ?, col_widths = ?, hidden_rows = ?, hidden_cols = ?, updated_at = ?
		WHERE id = ?
	`, sheet.Name, sheet.Index, sheet.Hidden, sheet.Color, sheet.GridColor,
		sheet.FrozenRows, sheet.FrozenCols, sheet.DefaultRowHeight, sheet.DefaultColWidth,
		string(rowHeights), string(colWidths), string(hiddenRows), string(hiddenCols),
		sheet.UpdatedAt, sheet.ID)
	return err
}

// Delete deletes a sheet and all related data.
// Properly handles cascade deletes with error checking.
func (s *SheetsStore) Delete(ctx context.Context, id string) error {
	// First, delete comment_replies for all comments in this sheet (nested FK)
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM comment_replies WHERE comment_id IN (
			SELECT id FROM comments WHERE sheet_id = ?
		)
	`, id); err != nil {
		return err
	}

	// Delete related data in correct order (foreign key constraints)
	// named_ranges has FK to sheets via sheet_id
	tables := []string{
		"named_ranges", "merged_regions", "cells", "conditional_formats",
		"data_validations", "charts", "pivot_tables", "comments", "auto_filters",
	}
	for _, table := range tables {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM `+table+` WHERE sheet_id = ?`, id); err != nil {
			return err
		}
	}

	// Delete the sheet itself
	_, err := s.db.ExecContext(ctx, `DELETE FROM sheets WHERE id = ?`, id)
	return err
}

// UpdateRowHeight updates a single row height using JSON patch operation.
func (s *SheetsStore) UpdateRowHeight(ctx context.Context, sheetID string, row, height int) error {
	// Fetch current row_heights, update it, and save back - but only that field
	var rowHeightsStr sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT CAST(row_heights AS VARCHAR) FROM sheets WHERE id = ?
	`, sheetID).Scan(&rowHeightsStr)
	if err != nil {
		return err
	}

	rowHeights := make(map[int]int)
	if rowHeightsStr.Valid {
		json.Unmarshal([]byte(rowHeightsStr.String), &rowHeights)
	}
	rowHeights[row] = height

	updated, _ := json.Marshal(rowHeights)
	_, err = s.db.ExecContext(ctx, `
		UPDATE sheets SET row_heights = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, string(updated), sheetID)
	return err
}

// UpdateColWidth updates a single column width using JSON patch operation.
func (s *SheetsStore) UpdateColWidth(ctx context.Context, sheetID string, col, width int) error {
	var colWidthsStr sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT CAST(col_widths AS VARCHAR) FROM sheets WHERE id = ?
	`, sheetID).Scan(&colWidthsStr)
	if err != nil {
		return err
	}

	colWidths := make(map[int]int)
	if colWidthsStr.Valid {
		json.Unmarshal([]byte(colWidthsStr.String), &colWidths)
	}
	colWidths[col] = width

	updated, _ := json.Marshal(colWidths)
	_, err = s.db.ExecContext(ctx, `
		UPDATE sheets SET col_widths = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, string(updated), sheetID)
	return err
}

// AddHiddenRow adds a row to the hidden rows list.
func (s *SheetsStore) AddHiddenRow(ctx context.Context, sheetID string, row int) error {
	var hiddenRowsStr sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT CAST(hidden_rows AS VARCHAR) FROM sheets WHERE id = ?
	`, sheetID).Scan(&hiddenRowsStr)
	if err != nil {
		return err
	}

	var hiddenRows []int
	if hiddenRowsStr.Valid {
		json.Unmarshal([]byte(hiddenRowsStr.String), &hiddenRows)
	}

	// Check if already hidden
	for _, r := range hiddenRows {
		if r == row {
			return nil
		}
	}

	hiddenRows = append(hiddenRows, row)
	updated, _ := json.Marshal(hiddenRows)
	_, err = s.db.ExecContext(ctx, `
		UPDATE sheets SET hidden_rows = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, string(updated), sheetID)
	return err
}

// RemoveHiddenRow removes a row from the hidden rows list.
func (s *SheetsStore) RemoveHiddenRow(ctx context.Context, sheetID string, row int) error {
	var hiddenRowsStr sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT CAST(hidden_rows AS VARCHAR) FROM sheets WHERE id = ?
	`, sheetID).Scan(&hiddenRowsStr)
	if err != nil {
		return err
	}

	var hiddenRows []int
	if hiddenRowsStr.Valid {
		json.Unmarshal([]byte(hiddenRowsStr.String), &hiddenRows)
	}

	// Remove the row
	newHidden := make([]int, 0, len(hiddenRows))
	for _, r := range hiddenRows {
		if r != row {
			newHidden = append(newHidden, r)
		}
	}

	updated, _ := json.Marshal(newHidden)
	_, err = s.db.ExecContext(ctx, `
		UPDATE sheets SET hidden_rows = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, string(updated), sheetID)
	return err
}

// AddHiddenCol adds a column to the hidden columns list.
func (s *SheetsStore) AddHiddenCol(ctx context.Context, sheetID string, col int) error {
	var hiddenColsStr sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT CAST(hidden_cols AS VARCHAR) FROM sheets WHERE id = ?
	`, sheetID).Scan(&hiddenColsStr)
	if err != nil {
		return err
	}

	var hiddenCols []int
	if hiddenColsStr.Valid {
		json.Unmarshal([]byte(hiddenColsStr.String), &hiddenCols)
	}

	// Check if already hidden
	for _, c := range hiddenCols {
		if c == col {
			return nil
		}
	}

	hiddenCols = append(hiddenCols, col)
	updated, _ := json.Marshal(hiddenCols)
	_, err = s.db.ExecContext(ctx, `
		UPDATE sheets SET hidden_cols = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, string(updated), sheetID)
	return err
}

// RemoveHiddenCol removes a column from the hidden columns list.
func (s *SheetsStore) RemoveHiddenCol(ctx context.Context, sheetID string, col int) error {
	var hiddenColsStr sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT CAST(hidden_cols AS VARCHAR) FROM sheets WHERE id = ?
	`, sheetID).Scan(&hiddenColsStr)
	if err != nil {
		return err
	}

	var hiddenCols []int
	if hiddenColsStr.Valid {
		json.Unmarshal([]byte(hiddenColsStr.String), &hiddenCols)
	}

	// Remove the column
	newHidden := make([]int, 0, len(hiddenCols))
	for _, c := range hiddenCols {
		if c != col {
			newHidden = append(newHidden, c)
		}
	}

	updated, _ := json.Marshal(newHidden)
	_, err = s.db.ExecContext(ctx, `
		UPDATE sheets SET hidden_cols = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, string(updated), sheetID)
	return err
}
