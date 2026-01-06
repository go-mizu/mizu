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

	var result []*sheets.Sheet
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

// Delete deletes a sheet.
func (s *SheetsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sheets WHERE id = ?`, id)
	return err
}
