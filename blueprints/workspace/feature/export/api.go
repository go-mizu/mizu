// Package export provides document export functionality.
package export

import (
	"context"
	"io"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

// Format represents the export output format.
type Format string

const (
	FormatPDF      Format = "pdf"
	FormatHTML     Format = "html"
	FormatMarkdown Format = "markdown"
)

// PageSize represents PDF page size.
type PageSize string

const (
	PageSizeA4      PageSize = "a4"
	PageSizeA3      PageSize = "a3"
	PageSizeLetter  PageSize = "letter"
	PageSizeLegal   PageSize = "legal"
	PageSizeTabloid PageSize = "tabloid"
	PageSizeAuto    PageSize = "auto"
)

// Orientation represents PDF orientation.
type Orientation string

const (
	OrientationPortrait  Orientation = "portrait"
	OrientationLandscape Orientation = "landscape"
)

// Request contains export parameters.
type Request struct {
	PageID          string      `json:"page_id"`
	PageTitle       string      `json:"page_title,omitempty"`       // Optional page title (for dev mode)
	Format          Format      `json:"format"`
	IncludeSubpages bool        `json:"include_subpages"`
	IncludeImages   bool        `json:"include_images"`
	IncludeFiles    bool        `json:"include_files"`
	CreateFolders   bool        `json:"create_folders"`
	IncludeComments bool        `json:"include_comments"`

	// Blocks to export (optional - if provided, uses these instead of fetching from DB)
	Blocks []map[string]interface{} `json:"blocks,omitempty"`

	// PDF-specific
	PageSize    PageSize    `json:"page_size,omitempty"`
	Orientation Orientation `json:"orientation,omitempty"`
	Scale       int         `json:"scale,omitempty"` // 10-200, default 100
}

// Result contains the export output.
type Result struct {
	ID          string `json:"id"`
	DownloadURL string `json:"download_url"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	Format      string `json:"format"`
	PageCount   int    `json:"page_count,omitempty"`
}

// Export represents an export job record.
type Export struct {
	ID          string    `json:"id"`
	PageID      string    `json:"page_id"`
	UserID      string    `json:"user_id"`
	Format      Format    `json:"format"`
	Status      string    `json:"status"` // pending, processing, completed, failed
	Filename    string    `json:"filename"`
	FilePath    string    `json:"-"`
	Size        int64     `json:"size"`
	PageCount   int       `json:"page_count"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// ExportedPage represents a page ready for export.
type ExportedPage struct {
	ID         string
	Title      string
	Icon       string
	Cover      string
	Blocks     []*blocks.Block
	Children   []*ExportedPage
	DatabaseID string
	Properties pages.Properties
	Path       string // Relative path in export
}

// ExportedDatabase represents a database ready for CSV export.
type ExportedDatabase struct {
	ID         string
	Title      string
	Properties []databases.Property
	Rows       []*ExportedPage
}

// API defines the export service contract.
type API interface {
	// Export initiates an export and returns the result.
	Export(ctx context.Context, userID string, req *Request) (*Result, error)

	// GetExport retrieves export status/result.
	GetExport(ctx context.Context, id string) (*Export, error)

	// Download returns the export file reader.
	Download(ctx context.Context, id string) (io.ReadCloser, *Export, error)

	// Cleanup removes expired exports.
	Cleanup(ctx context.Context) error
}

// Converter defines the interface for format converters.
type Converter interface {
	// Convert converts an exported page to the target format.
	Convert(page *ExportedPage, opts *Request) ([]byte, error)

	// ContentType returns the MIME type for this format.
	ContentType() string

	// Extension returns the file extension.
	Extension() string
}

// Store defines the data access contract for exports.
type Store interface {
	Create(ctx context.Context, e *Export) error
	GetByID(ctx context.Context, id string) (*Export, error)
	Update(ctx context.Context, id string, e *Export) error
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) (int, error)
}
