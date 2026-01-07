// Package cells provides cell management functionality.
package cells

import (
	"context"
	"time"
)

// CellType represents the type of cell value.
type CellType string

const (
	CellTypeText    CellType = "text"
	CellTypeNumber  CellType = "number"
	CellTypeDate    CellType = "date"
	CellTypeBool    CellType = "bool"
	CellTypeError   CellType = "error"
	CellTypeFormula CellType = "formula"
)

// Cell represents a single cell in a sheet.
type Cell struct {
	ID        string      `json:"id"`
	SheetID   string      `json:"sheet_id"`
	Row       int         `json:"row"`
	Col       int         `json:"col"`
	Value     interface{} `json:"value"`
	Formula   string      `json:"formula,omitempty"`
	Display   string      `json:"display"`
	Type      CellType    `json:"type"`
	Format    Format      `json:"format"`
	Hyperlink *Hyperlink  `json:"hyperlink,omitempty"`
	Note      string      `json:"note,omitempty"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// Format contains cell formatting options.
type Format struct {
	// Font
	FontFamily    string `json:"font_family,omitempty"`
	FontSize      int    `json:"font_size,omitempty"`
	FontColor     string `json:"font_color,omitempty"`
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Underline     bool   `json:"underline,omitempty"`
	Strikethrough bool   `json:"strikethrough,omitempty"`

	// Fill
	BackgroundColor string `json:"background_color,omitempty"`

	// Alignment
	HAlign       string `json:"h_align,omitempty"` // left, center, right
	VAlign       string `json:"v_align,omitempty"` // top, middle, bottom
	WrapText     bool   `json:"wrap_text,omitempty"`
	TextRotation int    `json:"text_rotation,omitempty"` // -90 to 90
	Indent       int    `json:"indent,omitempty"`

	// Borders
	BorderTop    Border `json:"border_top,omitempty"`
	BorderRight  Border `json:"border_right,omitempty"`
	BorderBottom Border `json:"border_bottom,omitempty"`
	BorderLeft   Border `json:"border_left,omitempty"`

	// Number format
	NumberFormat string `json:"number_format,omitempty"` // "0.00", "#,##0", "yyyy-mm-dd"
}

// Border represents a cell border.
type Border struct {
	Style string `json:"style,omitempty"` // none, thin, medium, thick, dotted, dashed, double
	Color string `json:"color,omitempty"`
}

// Hyperlink represents a cell hyperlink.
type Hyperlink struct {
	URL   string `json:"url"`
	Label string `json:"label,omitempty"`
}

// CellRef represents a cell reference.
type CellRef struct {
	Sheet string `json:"sheet,omitempty"`
	Row   int    `json:"row"`
	Col   int    `json:"col"`
}

// Range represents a range of cells.
type Range struct {
	Sheet    string `json:"sheet,omitempty"`
	StartRow int    `json:"start_row"`
	StartCol int    `json:"start_col"`
	EndRow   int    `json:"end_row"`
	EndCol   int    `json:"end_col"`
}

// SetIn contains cell set input.
type SetIn struct {
	SheetID   string      `json:"sheet_id"`
	Row       int         `json:"row"`
	Col       int         `json:"col"`
	Value     interface{} `json:"value,omitempty"`
	Formula   string      `json:"formula,omitempty"`
	CreatedBy string      `json:"created_by"`
}

// SetCellIn contains cell update input for a single cell.
type SetCellIn struct {
	Value   interface{} `json:"value,omitempty"`
	Formula string      `json:"formula,omitempty"`
	Format  *Format     `json:"format,omitempty"`
}

// BatchUpdateIn contains batch cell update input.
type BatchUpdateIn struct {
	Cells []CellUpdate `json:"cells"`
}

// CellUpdate represents a single cell update in a batch.
type CellUpdate struct {
	Row     int         `json:"row"`
	Col     int         `json:"col"`
	Value   interface{} `json:"value,omitempty"`
	Formula string      `json:"formula,omitempty"`
	Format  *Format     `json:"format,omitempty"`
}

// SetFormatIn contains cell format input.
type SetFormatIn struct {
	SheetID string `json:"sheet_id"`
	Row     int    `json:"row"`
	Col     int    `json:"col"`
	Format  Format `json:"format"`
}

// BatchSetIn contains batch cell set input.
type BatchSetIn struct {
	SheetID   string   `json:"sheet_id"`
	Cells     []SetIn  `json:"cells"`
	CreatedBy string   `json:"created_by"`
}

// MergedRegion represents a merged cell region.
type MergedRegion struct {
	ID       string `json:"id"`
	SheetID  string `json:"sheet_id"`
	StartRow int    `json:"start_row"`
	StartCol int    `json:"start_col"`
	EndRow   int    `json:"end_row"`
	EndCol   int    `json:"end_col"`
}

// API defines the cells service interface.
type API interface {
	// Get retrieves a cell by position.
	Get(ctx context.Context, sheetID string, row, col int) (*Cell, error)

	// GetRange retrieves cells in a range.
	GetRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([]*Cell, error)

	// Set sets a single cell value or formula.
	Set(ctx context.Context, sheetID string, row, col int, in *SetCellIn) (*Cell, error)

	// BatchUpdate updates multiple cells at once.
	BatchUpdate(ctx context.Context, sheetID string, in *BatchUpdateIn) ([]*Cell, error)

	// Delete deletes a cell.
	Delete(ctx context.Context, sheetID string, row, col int) error

	// SetFormat sets cell formatting.
	SetFormat(ctx context.Context, in *SetFormatIn) error

	// SetRangeFormat sets formatting for a range.
	SetRangeFormat(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int, format Format) error

	// Clear clears a cell.
	Clear(ctx context.Context, sheetID string, row, col int) error

	// ClearRange clears a range of cells.
	ClearRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error

	// SetNote sets a cell note.
	SetNote(ctx context.Context, sheetID string, row, col int, note string) error

	// SetHyperlink sets a cell hyperlink.
	SetHyperlink(ctx context.Context, sheetID string, row, col int, hyperlink *Hyperlink) error

	// Merge merges cells.
	Merge(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) (*MergedRegion, error)

	// Unmerge unmerges cells.
	Unmerge(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error

	// GetMergedRegions gets merged regions in a sheet.
	GetMergedRegions(ctx context.Context, sheetID string) ([]*MergedRegion, error)

	// CopyRange copies a range of cells.
	CopyRange(ctx context.Context, sourceSheetID string, sourceRange Range, destSheetID string, destRow, destCol int) error

	// InsertRows inserts rows at the specified index.
	InsertRows(ctx context.Context, sheetID string, rowIndex, count int) error

	// InsertCols inserts columns at the specified index.
	InsertCols(ctx context.Context, sheetID string, colIndex, count int) error

	// DeleteRows deletes rows starting at the specified index.
	DeleteRows(ctx context.Context, sheetID string, startRow, count int) error

	// DeleteCols deletes columns starting at the specified index.
	DeleteCols(ctx context.Context, sheetID string, startCol, count int) error

	// EvaluateFormula evaluates a formula and returns the result.
	EvaluateFormula(ctx context.Context, sheetID, formula string) (interface{}, error)
}

// CellPosition represents a cell position for batch lookups.
type CellPosition struct {
	Row int
	Col int
}

// Store defines the cells data access interface.
type Store interface {
	Get(ctx context.Context, sheetID string, row, col int) (*Cell, error)
	GetRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([]*Cell, error)
	// GetByPositions retrieves multiple cells by their positions in a single query.
	GetByPositions(ctx context.Context, sheetID string, positions []CellPosition) (map[CellPosition]*Cell, error)
	Set(ctx context.Context, cell *Cell) error
	BatchSet(ctx context.Context, cells []*Cell) error
	Delete(ctx context.Context, sheetID string, row, col int) error
	DeleteRange(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error
	// DeleteRowsRange deletes multiple rows and shifts remaining cells up in a single operation.
	DeleteRowsRange(ctx context.Context, sheetID string, startRow, count int) error
	// DeleteColsRange deletes multiple columns and shifts remaining cells left in a single operation.
	DeleteColsRange(ctx context.Context, sheetID string, startCol, count int) error
	CreateMerge(ctx context.Context, region *MergedRegion) error
	DeleteMerge(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) error
	GetMergedRegions(ctx context.Context, sheetID string) ([]*MergedRegion, error)
	ShiftRows(ctx context.Context, sheetID string, startRow, count int) error
	ShiftCols(ctx context.Context, sheetID string, startCol, count int) error
}
