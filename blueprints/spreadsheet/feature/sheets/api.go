// Package sheets provides sheet management functionality.
package sheets

import (
	"context"
	"time"
)

// Sheet represents a single sheet within a workbook.
type Sheet struct {
	ID               string      `json:"id"`
	WorkbookID       string      `json:"workbookId"`
	Name             string      `json:"name"`
	Index            int         `json:"index"`
	Hidden           bool        `json:"hidden"`
	Color            string      `json:"color,omitempty"`
	GridColor        string      `json:"gridColor"`
	FrozenRows       int         `json:"frozenRows"`
	FrozenCols       int         `json:"frozenCols"`
	DefaultRowHeight int         `json:"defaultRowHeight"`
	DefaultColWidth  int         `json:"defaultColWidth"`
	RowHeights       map[int]int `json:"rowHeights,omitempty"`
	ColWidths        map[int]int `json:"colWidths,omitempty"`
	HiddenRows       []int       `json:"hiddenRows,omitempty"`
	HiddenCols       []int       `json:"hiddenCols,omitempty"`
	CreatedAt        time.Time   `json:"createdAt"`
	UpdatedAt        time.Time   `json:"updatedAt"`
}

// CreateIn contains sheet creation input.
type CreateIn struct {
	WorkbookID string `json:"workbookId"`
	Name       string `json:"name"`
	Index      int    `json:"index"`
	Color      string `json:"color,omitempty"`
	CreatedBy  string `json:"createdBy"`
}

// UpdateIn contains sheet update input.
type UpdateIn struct {
	Name             string `json:"name,omitempty"`
	Index            int    `json:"index,omitempty"`
	Hidden           *bool  `json:"hidden,omitempty"`
	Color            string `json:"color,omitempty"`
	GridColor        string `json:"gridColor,omitempty"`
	FrozenRows       *int   `json:"frozenRows,omitempty"`
	FrozenCols       *int   `json:"frozenCols,omitempty"`
	DefaultRowHeight *int   `json:"defaultRowHeight,omitempty"`
	DefaultColWidth  *int   `json:"defaultColWidth,omitempty"`
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
	// UpdateRowHeight updates a single row height without loading the full object.
	UpdateRowHeight(ctx context.Context, sheetID string, row, height int) error
	// UpdateColWidth updates a single column width without loading the full object.
	UpdateColWidth(ctx context.Context, sheetID string, col, width int) error
	// AddHiddenRow adds a row to the hidden rows list.
	AddHiddenRow(ctx context.Context, sheetID string, row int) error
	// RemoveHiddenRow removes a row from the hidden rows list.
	RemoveHiddenRow(ctx context.Context, sheetID string, row int) error
	// AddHiddenCol adds a column to the hidden columns list.
	AddHiddenCol(ctx context.Context, sheetID string, col int) error
	// RemoveHiddenCol removes a column from the hidden columns list.
	RemoveHiddenCol(ctx context.Context, sheetID string, col int) error
}
