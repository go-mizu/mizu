// Package export provides data export functionality.
package export

import (
	"context"
	"errors"
	"io"
	"time"
)

// Errors
var (
	ErrUnsupportedFormat = errors.New("unsupported export format")
	ErrNoData            = errors.New("no data to export")
	ErrQuestionNotFound  = errors.New("question not found")
	ErrDashboardNotFound = errors.New("dashboard not found")
	ErrExportFailed      = errors.New("export failed")
)

// Format represents an export format.
type Format string

const (
	FormatCSV   Format = "csv"
	FormatXLSX  Format = "xlsx"
	FormatJSON  Format = "json"
	FormatPNG   Format = "png"
	FormatPDF   Format = "pdf"
)

// ExportResult represents the result of an export operation.
type ExportResult struct {
	Data        []byte    `json:"-"`
	ContentType string    `json:"content_type"`
	Filename    string    `json:"filename"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
}

// QuestionExportIn contains input for exporting a question.
type QuestionExportIn struct {
	QuestionID string            `json:"question_id"`
	Format     Format            `json:"format"`
	Options    map[string]any    `json:"options,omitempty"`
	Parameters map[string]any    `json:"parameters,omitempty"`
	Limit      int               `json:"limit,omitempty"`
}

// DashboardExportIn contains input for exporting a dashboard.
type DashboardExportIn struct {
	DashboardID string         `json:"dashboard_id"`
	Format      Format         `json:"format"` // pdf or png
	Options     map[string]any `json:"options,omitempty"`
	Width       int            `json:"width,omitempty"`
	Height      int            `json:"height,omitempty"`
}

// QueryResultExportIn contains input for exporting query results.
type QueryResultExportIn struct {
	Columns []Column         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	Format  Format           `json:"format"`
	Options map[string]any   `json:"options,omitempty"`
}

// Column represents a column for export.
type Column struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

// CSVOptions contains CSV-specific export options.
type CSVOptions struct {
	Delimiter   string `json:"delimiter,omitempty"`
	IncludeHeader bool `json:"include_header"`
	Encoding    string `json:"encoding,omitempty"`
}

// XLSXOptions contains Excel-specific export options.
type XLSXOptions struct {
	SheetName     string `json:"sheet_name,omitempty"`
	IncludeHeader bool   `json:"include_header"`
	AutoFilter    bool   `json:"auto_filter"`
	FreezeHeader  bool   `json:"freeze_header"`
}

// PDFOptions contains PDF-specific export options.
type PDFOptions struct {
	PageSize    string `json:"page_size,omitempty"`  // A4, Letter, etc.
	Orientation string `json:"orientation,omitempty"` // portrait, landscape
	IncludeDate bool   `json:"include_date"`
	Title       string `json:"title,omitempty"`
}

// PNGOptions contains PNG-specific export options.
type PNGOptions struct {
	Width      int  `json:"width,omitempty"`
	Height     int  `json:"height,omitempty"`
	Scale      int  `json:"scale,omitempty"`
	DarkMode   bool `json:"dark_mode"`
}

// API defines the Export service contract.
type API interface {
	// ExportQuestion exports a question's results.
	ExportQuestion(ctx context.Context, in *QuestionExportIn) (*ExportResult, error)

	// ExportDashboard exports a dashboard as PDF or PNG.
	ExportDashboard(ctx context.Context, in *DashboardExportIn) (*ExportResult, error)

	// ExportQueryResult exports raw query results.
	ExportQueryResult(ctx context.Context, in *QueryResultExportIn) (*ExportResult, error)

	// WriteCSV writes data as CSV to a writer.
	WriteCSV(ctx context.Context, w io.Writer, columns []Column, rows []map[string]any, opts *CSVOptions) error

	// WriteXLSX writes data as Excel to a writer.
	WriteXLSX(ctx context.Context, w io.Writer, columns []Column, rows []map[string]any, opts *XLSXOptions) error

	// WriteJSON writes data as JSON to a writer.
	WriteJSON(ctx context.Context, w io.Writer, columns []Column, rows []map[string]any) error
}

// QuestionStore defines data access for questions.
type QuestionStore interface {
	GetByID(ctx context.Context, id string) (*Question, error)
}

// DashboardStore defines data access for dashboards.
type DashboardStore interface {
	GetByID(ctx context.Context, id string) (*Dashboard, error)
}

// Question represents a saved question.
type Question struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Query         map[string]any `json:"query"`
	DataSourceID  string         `json:"datasource_id"`
	Visualization map[string]any `json:"visualization"`
}

// Dashboard represents a dashboard.
type Dashboard struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// QueryExecutor executes queries.
type QueryExecutor interface {
	Execute(ctx context.Context, query map[string]any) ([]Column, []map[string]any, error)
}

// ScreenshotRenderer renders dashboards as images.
type ScreenshotRenderer interface {
	RenderDashboard(ctx context.Context, dashboardID string, width, height int) ([]byte, error)
	RenderDashboardPDF(ctx context.Context, dashboardID string, opts *PDFOptions) ([]byte, error)
}
