package export

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

// Service implements the export API.
type Service struct {
	pages     pages.API
	blocks    blocks.API
	databases databases.API
	exportDir string
}

// NewService creates a new export service.
func NewService(pagesAPI pages.API, blocksAPI blocks.API, databasesAPI databases.API, exportDir string) *Service {
	// Ensure export directory exists
	os.MkdirAll(exportDir, 0755)

	return &Service{
		pages:     pagesAPI,
		blocks:    blocksAPI,
		databases: databasesAPI,
		exportDir: exportDir,
	}
}

// Export initiates an export and returns the result.
func (s *Service) Export(ctx context.Context, userID string, req *Request) (*Result, error) {
	// Validate request
	if err := s.validateRequest(req); err != nil {
		return nil, err
	}

	// Get the page to export
	page, err := s.pages.GetByID(ctx, req.PageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get page: %w", err)
	}

	// Build the export tree
	exportedPage, err := s.buildExportTree(ctx, page, req)
	if err != nil {
		return nil, fmt.Errorf("failed to build export tree: %w", err)
	}

	// Create the export bundle
	bundle, err := CreateBundle(exportedPage, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create export: %w", err)
	}

	// Generate export ID
	exportID := ulid.New()

	// Save to file
	filePath := filepath.Join(s.exportDir, exportID)
	if err := os.WriteFile(filePath, bundle.Data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save export: %w", err)
	}

	return &Result{
		ID:          exportID,
		DownloadURL: fmt.Sprintf("/api/v1/exports/%s/download", exportID),
		Filename:    bundle.Filename,
		Size:        int64(len(bundle.Data)),
		Format:      string(req.Format),
		PageCount:   bundle.PageCount,
	}, nil
}

// GetExport retrieves export status/result.
func (s *Service) GetExport(ctx context.Context, id string) (*Export, error) {
	filePath := filepath.Join(s.exportDir, id)

	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("export not found")
		}
		return nil, err
	}

	return &Export{
		ID:        id,
		Status:    "completed",
		Size:      info.Size(),
		CreatedAt: info.ModTime(),
		ExpiresAt: info.ModTime().Add(24 * time.Hour),
	}, nil
}

// Download returns the export file reader.
func (s *Service) Download(ctx context.Context, id string) (io.ReadCloser, *Export, error) {
	exp, err := s.GetExport(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	filePath := filepath.Join(s.exportDir, id)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open export file: %w", err)
	}

	return file, exp, nil
}

// Cleanup removes expired exports.
func (s *Service) Cleanup(ctx context.Context) error {
	entries, err := os.ReadDir(s.exportDir)
	if err != nil {
		return err
	}

	expireThreshold := time.Now().Add(-24 * time.Hour)
	deleted := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(expireThreshold) {
			filePath := filepath.Join(s.exportDir, entry.Name())
			if err := os.Remove(filePath); err == nil {
				deleted++
			}
		}
	}

	return nil
}

// validateRequest validates the export request.
func (s *Service) validateRequest(req *Request) error {
	if req.PageID == "" {
		return fmt.Errorf("page_id is required")
	}

	switch req.Format {
	case FormatPDF, FormatHTML, FormatMarkdown:
		// Valid formats
	case "":
		return fmt.Errorf("format is required")
	default:
		return fmt.Errorf("invalid format: %s", req.Format)
	}

	// Set defaults
	if req.PageSize == "" {
		req.PageSize = PageSizeLetter
	}
	if req.Orientation == "" {
		req.Orientation = OrientationPortrait
	}
	if req.Scale <= 0 || req.Scale > 200 {
		req.Scale = 100
	}

	return nil
}

// buildExportTree builds the complete export tree for a page.
func (s *Service) buildExportTree(ctx context.Context, page *pages.Page, req *Request) (*ExportedPage, error) {
	// Get blocks for this page
	pageBlocks, err := s.blocks.GetByPage(ctx, page.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get blocks: %w", err)
	}

	// Build block tree (organize parent-child relationships)
	blockTree := s.buildBlockTree(pageBlocks)

	exported := &ExportedPage{
		ID:         page.ID,
		Title:      page.Title,
		Icon:       page.Icon,
		Cover:      page.Cover,
		Blocks:     blockTree,
		DatabaseID: page.DatabaseID,
		Properties: page.Properties,
	}

	// Recursively get child pages if requested
	if req.IncludeSubpages {
		children, err := s.pages.ListByParent(ctx, page.ID, pages.ParentPage)
		if err != nil {
			return nil, fmt.Errorf("failed to get child pages: %w", err)
		}

		for _, child := range children {
			if child.IsArchived {
				continue
			}
			childExport, err := s.buildExportTree(ctx, child, req)
			if err != nil {
				// Log error but continue with other children
				continue
			}
			exported.Children = append(exported.Children, childExport)
		}
	}

	return exported, nil
}

// buildBlockTree organizes flat blocks into a tree structure.
func (s *Service) buildBlockTree(flatBlocks []*blocks.Block) []*blocks.Block {
	if len(flatBlocks) == 0 {
		return nil
	}

	// Create a map for quick lookup
	blockMap := make(map[string]*blocks.Block)
	for _, b := range flatBlocks {
		blockMap[b.ID] = b
		b.Children = nil // Reset children
	}

	// Build tree structure
	var roots []*blocks.Block
	for _, b := range flatBlocks {
		if b.ParentID == "" {
			roots = append(roots, b)
		} else if parent, ok := blockMap[b.ParentID]; ok {
			parent.Children = append(parent.Children, b)
		} else {
			// Orphan block - treat as root
			roots = append(roots, b)
		}
	}

	// Sort roots by position
	sortBlocksByPosition(roots)

	// Sort children recursively
	for _, root := range roots {
		sortBlocksRecursive(root)
	}

	return roots
}

// sortBlocksByPosition sorts blocks by their position.
func sortBlocksByPosition(blockList []*blocks.Block) {
	// Simple insertion sort for small lists
	for i := 1; i < len(blockList); i++ {
		j := i
		for j > 0 && blockList[j-1].Position > blockList[j].Position {
			blockList[j-1], blockList[j] = blockList[j], blockList[j-1]
			j--
		}
	}
}

// sortBlocksRecursive sorts children of a block recursively.
func sortBlocksRecursive(block *blocks.Block) {
	if len(block.Children) > 0 {
		sortBlocksByPosition(block.Children)
		for _, child := range block.Children {
			sortBlocksRecursive(child)
		}
	}
}

// ExportDatabase exports a database to CSV.
func (s *Service) ExportDatabase(ctx context.Context, userID, databaseID string, req *Request) (*Result, error) {
	// Get database
	db, err := s.databases.GetByID(ctx, databaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	// Get database page to access rows
	// Database rows are stored as pages with database_id set
	// We need to query pages that belong to this database

	exportedDB := &ExportedDatabase{
		ID:         db.ID,
		Title:      db.Title,
		Properties: db.Properties,
		Rows:       nil, // Would need to fetch rows
	}

	// Convert to CSV
	csvConverter := NewCSVConverter()
	csvData, err := csvConverter.Convert(exportedDB)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to CSV: %w", err)
	}

	// Generate export ID
	exportID := ulid.New()

	// Save to file
	filePath := filepath.Join(s.exportDir, exportID)
	if err := os.WriteFile(filePath, csvData, 0644); err != nil {
		return nil, fmt.Errorf("failed to save export: %w", err)
	}

	filename := sanitizeFilename(db.Title) + ".csv"

	return &Result{
		ID:          exportID,
		DownloadURL: fmt.Sprintf("/api/v1/exports/%s/download", exportID),
		Filename:    filename,
		Size:        int64(len(csvData)),
		Format:      "csv",
		PageCount:   len(exportedDB.Rows),
	}, nil
}

// GetFilename generates a filename for an export based on format.
func GetFilename(title string, format Format, isZip bool) string {
	name := sanitizeFilename(title)
	if name == "" {
		name = "export"
	}

	if isZip {
		return name + ".zip"
	}

	switch format {
	case FormatMarkdown:
		return name + ".md"
	case FormatHTML:
		return name + ".html"
	case FormatPDF:
		return name + ".pdf"
	default:
		return name + ".html"
	}
}

// GetContentType returns the content type for a format.
func GetContentType(format Format, isZip bool) string {
	if isZip {
		return "application/zip"
	}

	switch format {
	case FormatMarkdown:
		return "text/markdown; charset=utf-8"
	case FormatHTML:
		return "text/html; charset=utf-8"
	case FormatPDF:
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

// DetectContentType detects content type from filename.
func DetectContentType(filename string) string {
	lower := strings.ToLower(filename)

	switch {
	case strings.HasSuffix(lower, ".zip"):
		return "application/zip"
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(lower, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(lower, ".md"):
		return "text/markdown; charset=utf-8"
	case strings.HasSuffix(lower, ".csv"):
		return "text/csv; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
