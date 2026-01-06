// Package sheets provides sheet management functionality.
package sheets

import (
	"context"
	"time"
)

// Sheet represents a single sheet within a workbook.
type Sheet struct {
	ID               string         `json:"id"`
	WorkbookID       string         `json:"workbook_id"`
	Name             string         `json:"name"`
	Index            int            `json:"index"`
	Hidden           bool           `json:"hidden"`
	Color            string         `json:"color,omitempty"`
	GridColor        string         `json:"grid_color"`
	FrozenRows       int            `json:"frozen_rows"`
	FrozenCols       int            `json:"frozen_cols"`
	DefaultRowHeight int            `json:"default_row_height"`
	DefaultColWidth  int            `json:"default_col_width"`
	RowHeights       map[int]int    `json:"row_heights,omitempty"`
	ColWidths        map[int]int    `json:"col_widths,omitempty"`
	HiddenRows       []int          `json:"hidden_rows,omitempty"`
	HiddenCols       []int          `json:"hidden_cols,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// CreateIn contains sheet creation input.
type CreateIn struct {
	WorkbookID string `json:"workbook_id"`
	Name       string `json:"name"`
	Index      int    `json:"index"`
	Color      string `json:"color,omitempty"`
	CreatedBy  string `json:"created_by"`
}

// UpdateIn contains sheet update input.
type UpdateIn struct {
	Name             string `json:"name,omitempty"`
	Index            int    `json:"index,omitempty"`
	Hidden           *bool  `json:"hidden,omitempty"`
	Color            string `json:"color,omitempty"`
	GridColor        string `json:"grid_color,omitempty"`
	FrozenRows       *int   `json:"frozen_rows,omitempty"`
	FrozenCols       *int   `json:"frozen_cols,omitempty"`
	DefaultRowHeight *int   `json:"default_row_height,omitempty"`
	DefaultColWidth  *int   `json:"default_col_width,omitempty"`
}

// API defines the sheets service interface.
type API interface {
	// Create creates a new sheet.
	Create(ctx context.Context, in *CreateIn) (*Sheet, error)

	// GetByID retrieves a sheet by ID.
	GetByID(ctx context.Context, id string) (*Sheet, error)

	// List lists sheets in a workbook.
	List(ctx context.Context, workbookID string) ([]*Sheet, error)

	// Update updates a sheet.
	Update(ctx context.Context, id string, in *UpdateIn) (*Sheet, error)

	// Delete deletes a sheet.
	Delete(ctx context.Context, id string) error

	// Copy creates a copy of a sheet.
	Copy(ctx context.Context, id string, newName string) (*Sheet, error)

	// SetRowHeight sets the height of a specific row.
	SetRowHeight(ctx context.Context, sheetID string, row int, height int) error

	// SetColWidth sets the width of a specific column.
	SetColWidth(ctx context.Context, sheetID string, col int, width int) error

	// HideRow hides a row.
	HideRow(ctx context.Context, sheetID string, row int) error

	// HideCol hides a column.
	HideCol(ctx context.Context, sheetID string, col int) error

	// ShowRow shows a hidden row.
	ShowRow(ctx context.Context, sheetID string, row int) error

	// ShowCol shows a hidden column.
	ShowCol(ctx context.Context, sheetID string, col int) error
}

// Store defines the sheets data access interface.
type Store interface {
	Create(ctx context.Context, sheet *Sheet) error
	GetByID(ctx context.Context, id string) (*Sheet, error)
	ListByWorkbook(ctx context.Context, workbookID string) ([]*Sheet, error)
	Update(ctx context.Context, sheet *Sheet) error
	Delete(ctx context.Context, id string) error
}
