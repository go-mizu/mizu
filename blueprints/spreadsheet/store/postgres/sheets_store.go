package postgres

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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, sheet.ID, sheet.WorkbookID, sheet.Name, sheet.Index, sheet.Hidden, sheet.Color,
		sheet.GridColor, sheet.FrozenRows, sheet.FrozenCols, sheet.DefaultRowHeight,
		sheet.DefaultColWidth, string(rowHeights), string(colWidths),
		string(hiddenRows), string(hiddenCols), sheet.CreatedAt, sheet.UpdatedAt)
	return err
}

// GetByID retrieves a sheet by ID.
func (s *SheetsStore) GetByID(ctx context.Context, id string) (*sheets.Sheet, error) {
	sheet := &sheets.Sheet{}
	var rowHeights, colWidths, hiddenRows, hiddenCols []byte
	var color sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, workbook_id, name, index_num, hidden, color, grid_color,
			frozen_rows, frozen_cols, default_row_height, default_col_width,
			row_heights, col_widths, hidden_rows, hidden_cols, created_at, updated_at
		FROM sheets WHERE id = $1
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

	if len(rowHeights) > 0 {
		json.Unmarshal(rowHeights, &sheet.RowHeights)
	}
	if len(colWidths) > 0 {
		json.Unmarshal(colWidths, &sheet.ColWidths)
	}
	if len(hiddenRows) > 0 {
		json.Unmarshal(hiddenRows, &sheet.HiddenRows)
	}
	if len(hiddenCols) > 0 {
		json.Unmarshal(hiddenCols, &sheet.HiddenCols)
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
			row_heights, col_widths, hidden_rows, hidden_cols, created_at, updated_at
		FROM sheets WHERE workbook_id = $1
		ORDER BY index_num
	`, workbookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*sheets.Sheet, 0)
	for rows.Next() {
		sheet := &sheets.Sheet{}
		var rowHeights, colWidths, hiddenRows, hiddenCols []byte
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

		if len(rowHeights) > 0 {
			json.Unmarshal(rowHeights, &sheet.RowHeights)
		}
		if len(colWidths) > 0 {
			json.Unmarshal(colWidths, &sheet.ColWidths)
		}
		if len(hiddenRows) > 0 {
			json.Unmarshal(hiddenRows, &sheet.HiddenRows)
		}
		if len(hiddenCols) > 0 {
			json.Unmarshal(hiddenCols, &sheet.HiddenCols)
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
		UPDATE sheets SET name = $1, index_num = $2, hidden = $3, color = $4, grid_color = $5,
			frozen_rows = $6, frozen_cols = $7, default_row_height = $8, default_col_width = $9,
			row_heights = $10, col_widths = $11, hidden_rows = $12, hidden_cols = $13, updated_at = $14
		WHERE id = $15
	`, sheet.Name, sheet.Index, sheet.Hidden, sheet.Color, sheet.GridColor,
		sheet.FrozenRows, sheet.FrozenCols, sheet.DefaultRowHeight, sheet.DefaultColWidth,
		string(rowHeights), string(colWidths), string(hiddenRows), string(hiddenCols),
		sheet.UpdatedAt, sheet.ID)
	return err
}

// Delete deletes a sheet and all related data.
func (s *SheetsStore) Delete(ctx context.Context, id string) error {
	// First, delete comment_replies for all comments in this sheet (nested FK)
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM comment_replies WHERE comment_id IN (
			SELECT id FROM comments WHERE sheet_id = $1
		)
	`, id); err != nil {
		return err
	}

	// Delete related data in correct order (foreign key constraints)
	tables := []string{
		"named_ranges", "merged_regions", "cells", "conditional_formats",
		"data_validations", "charts", "pivot_tables", "comments", "auto_filters",
	}
	for _, table := range tables {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM `+table+` WHERE sheet_id = $1`, id); err != nil {
			return err
		}
	}

	// Delete the sheet itself
	_, err := s.db.ExecContext(ctx, `DELETE FROM sheets WHERE id = $1`, id)
	return err
}

// UpdateRowHeight updates a single row height.
func (s *SheetsStore) UpdateRowHeight(ctx context.Context, sheetID string, row, height int) error {
	var rowHeights []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT row_heights FROM sheets WHERE id = $1
	`, sheetID).Scan(&rowHeights)
	if err != nil {
		return err
	}

	rowHeightsMap := make(map[int]int)
	if len(rowHeights) > 0 {
		json.Unmarshal(rowHeights, &rowHeightsMap)
	}
	rowHeightsMap[row] = height

	updated, _ := json.Marshal(rowHeightsMap)
	_, err = s.db.ExecContext(ctx, `
		UPDATE sheets SET row_heights = $1, updated_at = NOW() WHERE id = $2
	`, string(updated), sheetID)
	return err
}

// UpdateColWidth updates a single column width.
func (s *SheetsStore) UpdateColWidth(ctx context.Context, sheetID string, col, width int) error {
	var colWidths []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT col_widths FROM sheets WHERE id = $1
	`, sheetID).Scan(&colWidths)
	if err != nil {
		return err
	}

	colWidthsMap := make(map[int]int)
	if len(colWidths) > 0 {
		json.Unmarshal(colWidths, &colWidthsMap)
	}
	colWidthsMap[col] = width

	updated, _ := json.Marshal(colWidthsMap)
	_, err = s.db.ExecContext(ctx, `
		UPDATE sheets SET col_widths = $1, updated_at = NOW() WHERE id = $2
	`, string(updated), sheetID)
	return err
}

// AddHiddenRow adds a row to the hidden rows list.
func (s *SheetsStore) AddHiddenRow(ctx context.Context, sheetID string, row int) error {
	var hiddenRows []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT hidden_rows FROM sheets WHERE id = $1
	`, sheetID).Scan(&hiddenRows)
	if err != nil {
		return err
	}

	var hiddenRowsList []int
	if len(hiddenRows) > 0 {
		json.Unmarshal(hiddenRows, &hiddenRowsList)
	}

	// Check if already hidden
	for _, r := range hiddenRowsList {
		if r == row {
			return nil
		}
	}

	hiddenRowsList = append(hiddenRowsList, row)
	updated, _ := json.Marshal(hiddenRowsList)
	_, err = s.db.ExecContext(ctx, `
		UPDATE sheets SET hidden_rows = $1, updated_at = NOW() WHERE id = $2
	`, string(updated), sheetID)
	return err
}

// RemoveHiddenRow removes a row from the hidden rows list.
func (s *SheetsStore) RemoveHiddenRow(ctx context.Context, sheetID string, row int) error {
	var hiddenRows []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT hidden_rows FROM sheets WHERE id = $1
	`, sheetID).Scan(&hiddenRows)
	if err != nil {
		return err
	}

	var hiddenRowsList []int
	if len(hiddenRows) > 0 {
		json.Unmarshal(hiddenRows, &hiddenRowsList)
	}

	// Remove the row
	newHidden := make([]int, 0, len(hiddenRowsList))
	for _, r := range hiddenRowsList {
		if r != row {
			newHidden = append(newHidden, r)
		}
	}

	updated, _ := json.Marshal(newHidden)
	_, err = s.db.ExecContext(ctx, `
		UPDATE sheets SET hidden_rows = $1, updated_at = NOW() WHERE id = $2
	`, string(updated), sheetID)
	return err
}

// AddHiddenCol adds a column to the hidden columns list.
func (s *SheetsStore) AddHiddenCol(ctx context.Context, sheetID string, col int) error {
	var hiddenCols []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT hidden_cols FROM sheets WHERE id = $1
	`, sheetID).Scan(&hiddenCols)
	if err != nil {
		return err
	}

	var hiddenColsList []int
	if len(hiddenCols) > 0 {
		json.Unmarshal(hiddenCols, &hiddenColsList)
	}

	// Check if already hidden
	for _, c := range hiddenColsList {
		if c == col {
			return nil
		}
	}

	hiddenColsList = append(hiddenColsList, col)
	updated, _ := json.Marshal(hiddenColsList)
	_, err = s.db.ExecContext(ctx, `
		UPDATE sheets SET hidden_cols = $1, updated_at = NOW() WHERE id = $2
	`, string(updated), sheetID)
	return err
}

// RemoveHiddenCol removes a column from the hidden columns list.
func (s *SheetsStore) RemoveHiddenCol(ctx context.Context, sheetID string, col int) error {
	var hiddenCols []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT hidden_cols FROM sheets WHERE id = $1
	`, sheetID).Scan(&hiddenCols)
	if err != nil {
		return err
	}

	var hiddenColsList []int
	if len(hiddenCols) > 0 {
		json.Unmarshal(hiddenCols, &hiddenColsList)
	}

	// Remove the column
	newHidden := make([]int, 0, len(hiddenColsList))
	for _, c := range hiddenColsList {
		if c != col {
			newHidden = append(newHidden, c)
		}
	}

	updated, _ := json.Marshal(newHidden)
	_, err = s.db.ExecContext(ctx, `
		UPDATE sheets SET hidden_cols = $1, updated_at = NOW() WHERE id = $2
	`, string(updated), sheetID)
	return err
}
