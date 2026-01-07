// Package importer provides spreadsheet import functionality.
package importer

import (
	"context"
	"io"
)

// Format represents an import format.
type Format string

const (
	FormatCSV  Format = "csv"
	FormatTSV  Format = "tsv"
	FormatXLSX Format = "xlsx"
	FormatJSON Format = "json"
)

// Options contains import options.
type Options struct {
	// CSV/TSV options
	Delimiter      string `json:"delimiter,omitempty"`
	HasHeaders     bool   `json:"hasHeaders,omitempty"`
	SkipEmptyRows  bool   `json:"skipEmptyRows,omitempty"`
	TrimWhitespace bool   `json:"trimWhitespace,omitempty"`

	// Target options
	StartRow int `json:"startRow,omitempty"`
	StartCol int `json:"startCol,omitempty"`

	// Conflict resolution
	OverwriteExisting bool `json:"overwriteExisting,omitempty"`

	// Type detection
	AutoDetectTypes bool   `json:"autoDetectTypes,omitempty"`
	DateFormat      string `json:"dateFormat,omitempty"`

	// XLSX options
	ImportSheet      string `json:"importSheet,omitempty"`
	ImportFormatting bool   `json:"importFormatting,omitempty"`
	ImportFormulas   bool   `json:"importFormulas,omitempty"`

	// Sheet creation
	CreateNewSheet bool   `json:"createNewSheet,omitempty"`
	SheetName      string `json:"sheetName,omitempty"`
}

// Result contains the result of an import operation.
type Result struct {
	SheetID       string   `json:"sheetId"`
	RowsImported  int      `json:"rowsImported"`
	ColsImported  int      `json:"colsImported"`
	CellsImported int      `json:"cellsImported"`
	Warnings      []string `json:"warnings,omitempty"`
}

// API defines the import service interface.
type API interface {
	// ImportToWorkbook imports data to a workbook (creates new sheet).
	ImportToWorkbook(ctx context.Context, workbookID string, reader io.Reader, filename string, format Format, opts *Options) (*Result, error)

	// ImportToSheet imports data to an existing sheet.
	ImportToSheet(ctx context.Context, sheetID string, reader io.Reader, filename string, format Format, opts *Options) (*Result, error)

	// DetectFormat detects the format of the input file.
	DetectFormat(filename string) Format

	// SupportedFormats returns list of supported import formats.
	SupportedFormats() []Format

	// ValidateFile validates file before import.
	ValidateFile(ctx context.Context, reader io.Reader, format Format) error
}

// CellImport represents a cell to be imported.
type CellImport struct {
	Row     int
	Col     int
	Value   interface{}
	Formula string
	Format  *CellFormat
}

// CellFormat represents cell formatting for import.
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
}

// MergedRegionImport represents a merged region to import.
type MergedRegionImport struct {
	StartRow int
	StartCol int
	EndRow   int
	EndCol   int
}

// SheetImport represents imported sheet data.
type SheetImport struct {
	Name          string
	Cells         []CellImport
	MergedRegions []MergedRegionImport
}
