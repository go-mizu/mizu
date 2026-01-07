// Package export provides spreadsheet export functionality.
package export

import (
	"context"
	"io"
)

// Format represents an export format.
type Format string

const (
	FormatCSV  Format = "csv"
	FormatTSV  Format = "tsv"
	FormatXLSX Format = "xlsx"
	FormatJSON Format = "json"
	FormatPDF  Format = "pdf"
	FormatHTML Format = "html"
)

// Options contains export options.
type Options struct {
	// CSV/TSV options
	Delimiter      string `json:"delimiter,omitempty"`
	IncludeHeaders bool   `json:"includeHeaders,omitempty"`
	QuoteAll       bool   `json:"quoteAll,omitempty"`

	// Range options
	Range *Range `json:"range,omitempty"`

	// Value options
	ExportFormulas   bool `json:"exportFormulas,omitempty"`
	ExportFormatting bool `json:"exportFormatting,omitempty"`

	// PDF options
	Orientation      string `json:"orientation,omitempty"` // portrait, landscape
	PaperSize        string `json:"paperSize,omitempty"`   // letter, a4, legal
	FitToPage        bool   `json:"fitToPage,omitempty"`
	IncludeGridlines bool   `json:"includeGridlines,omitempty"`

	// JSON options
	Compact         bool `json:"compact,omitempty"`
	IncludeMetadata bool `json:"includeMetadata,omitempty"`
}

// Range defines a cell range.
type Range struct {
	StartRow int `json:"startRow"`
	StartCol int `json:"startCol"`
	EndRow   int `json:"endRow"`
	EndCol   int `json:"endCol"`
}

// Result contains export result information.
type Result struct {
	ContentType string
	Filename    string
	Data        io.Reader
	Size        int64
}

// API defines the export service interface.
type API interface {
	// ExportWorkbook exports an entire workbook.
	ExportWorkbook(ctx context.Context, workbookID string, format Format, opts *Options) (*Result, error)

	// ExportSheet exports a single sheet.
	ExportSheet(ctx context.Context, sheetID string, format Format, opts *Options) (*Result, error)

	// SupportedFormats returns list of supported export formats.
	SupportedFormats() []Format
}

// SheetData represents sheet data for export.
type SheetData struct {
	ID            string
	Name          string
	Cells         []CellData
	MergedRegions []MergedRegion
	ColWidths     map[int]int
	RowHeights    map[int]int
	MaxRow        int
	MaxCol        int
}

// CellData represents cell data for export.
type CellData struct {
	Row       int
	Col       int
	Value     interface{}
	Formula   string
	Display   string
	Type      string
	Format    *CellFormat
	Note      string
	Hyperlink string
}

// CellFormat represents cell formatting.
type CellFormat struct {
	FontFamily      string
	FontSize        int
	FontColor       string
	Bold            bool
	Italic          bool
	Underline       bool
	Strikethrough   bool
	BackgroundColor string
	HAlign          string
	VAlign          string
	WrapText        bool
	NumberFormat    string
	BorderTop       Border
	BorderRight     Border
	BorderBottom    Border
	BorderLeft      Border
}

// Border represents a border style.
type Border struct {
	Style string
	Color string
}

// MergedRegion represents a merged cell region.
type MergedRegion struct {
	StartRow int
	StartCol int
	EndRow   int
	EndCol   int
}

// WorkbookData represents workbook data for export.
type WorkbookData struct {
	ID       string
	Name     string
	Settings WorkbookSettings
	Sheets   []SheetData
}

// WorkbookSettings represents workbook settings.
type WorkbookSettings struct {
	Locale          string
	TimeZone        string
	CalculationMode string
}
