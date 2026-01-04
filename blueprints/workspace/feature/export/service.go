package export

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

// exportMeta stores export metadata alongside the export file.
type exportMeta struct {
	Filename string `json:"filename"`
	Format   Format `json:"format"`
}

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
	slog.Debug("starting export",
		"page_id", req.PageID,
		"format", req.Format,
		"user_id", userID,
	)

	// Validate request
	if err := s.validateRequest(req); err != nil {
		slog.Error("export validation failed", "error", err)
		return nil, err
	}

	// Get or create the page for export
	var page *pages.Page
	if len(req.Blocks) > 0 && req.PageTitle != "" {
		// Dev mode: use provided blocks and title
		slog.Debug("using provided page title and blocks", "title", req.PageTitle)
		page = &pages.Page{
			ID:    req.PageID,
			Title: req.PageTitle,
		}
	} else {
		// Production mode: fetch from database
		slog.Debug("fetching page", "page_id", req.PageID)
		var err error
		page, err = s.pages.GetByID(ctx, req.PageID)
		if err != nil {
			slog.Error("failed to get page for export",
				"page_id", req.PageID,
				"error", err,
			)
			return nil, fmt.Errorf("failed to get page: %w", err)
		}
		slog.Debug("page fetched", "title", page.Title)
	}

	// Build the export tree
	slog.Debug("building export tree", "include_subpages", req.IncludeSubpages)
	exportedPage, err := s.buildExportTree(ctx, page, req)
	if err != nil {
		slog.Error("failed to build export tree", "error", err)
		return nil, fmt.Errorf("failed to build export tree: %w", err)
	}
	slog.Debug("export tree built",
		"block_count", len(exportedPage.Blocks),
		"children_count", len(exportedPage.Children),
	)

	// Create the export bundle
	slog.Debug("creating export bundle", "format", req.Format)
	bundle, err := CreateBundle(exportedPage, req)
	if err != nil {
		slog.Error("failed to create export bundle",
			"format", req.Format,
			"error", err,
		)
		return nil, fmt.Errorf("failed to create export: %w", err)
	}
	slog.Debug("export bundle created",
		"filename", bundle.Filename,
		"size", len(bundle.Data),
		"content_type", bundle.ContentType,
	)

	// Generate export ID
	exportID := ulid.New()

	// Save to file
	filePath := filepath.Join(s.exportDir, exportID)
	slog.Debug("saving export to file", "path", filePath)
	if err := os.WriteFile(filePath, bundle.Data, 0644); err != nil {
		slog.Error("failed to save export file",
			"path", filePath,
			"error", err,
		)
		return nil, fmt.Errorf("failed to save export: %w", err)
	}

	// Save metadata file for filename retrieval on download
	meta := exportMeta{Filename: bundle.Filename, Format: req.Format}
	metaBytes, _ := json.Marshal(meta)
	metaPath := filePath + ".meta"
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		slog.Warn("failed to save export metadata", "error", err)
	}

	slog.Info("export completed successfully",
		"export_id", exportID,
		"format", req.Format,
		"size", len(bundle.Data),
	)

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

	// Read metadata for filename
	var filename string
	var format Format
	metaPath := filePath + ".meta"
	if metaBytes, err := os.ReadFile(metaPath); err == nil {
		var meta exportMeta
		if json.Unmarshal(metaBytes, &meta) == nil {
			filename = meta.Filename
			format = meta.Format
		}
	}

	return &Export{
		ID:        id,
		Status:    "completed",
		Filename:  filename,
		Format:    format,
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

		// Skip metadata files (they'll be deleted with their export)
		if strings.HasSuffix(entry.Name(), ".meta") {
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
				// Also remove metadata file
				os.Remove(filePath + ".meta")
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
	var blockTree []*blocks.Block

	// If blocks are provided in request, use those (for frontend dev mode)
	if len(req.Blocks) > 0 {
		blockTree = s.convertRequestBlocks(req.Blocks)
		slog.Debug("using provided blocks from request", "count", len(blockTree))
	} else {
		// Get blocks for this page from database
		pageBlocks, err := s.blocks.GetByPage(ctx, page.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get blocks: %w", err)
		}

		// Build block tree (organize parent-child relationships)
		blockTree = s.buildBlockTree(pageBlocks)
	}

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

// convertRequestBlocks converts JSON block data from the request into Block structures.
func (s *Service) convertRequestBlocks(reqBlocks []map[string]interface{}) []*blocks.Block {
	var result []*blocks.Block
	for i, rb := range reqBlocks {
		block := &blocks.Block{
			ID:       getString(rb, "id"),
			Type:     blocks.BlockType(getString(rb, "type")),
			Content:  convertContent(getMap(rb, "content")),
			Position: i,
		}

		// Handle children recursively
		if children, ok := rb["children"].([]interface{}); ok {
			for j, child := range children {
				if childMap, ok := child.(map[string]interface{}); ok {
					childBlock := &blocks.Block{
						ID:       getString(childMap, "id"),
						Type:     blocks.BlockType(getString(childMap, "type")),
						Content:  convertContent(getMap(childMap, "content")),
						ParentID: block.ID,
						Position: j,
					}
					// Handle nested children
					if nestedChildren, ok := childMap["children"].([]interface{}); ok {
						childBlock.Children = s.convertChildBlocks(nestedChildren, childBlock.ID)
					}
					block.Children = append(block.Children, childBlock)
				}
			}
		}
		result = append(result, block)
	}
	return result
}

// convertChildBlocks recursively converts child blocks.
func (s *Service) convertChildBlocks(children []interface{}, parentID string) []*blocks.Block {
	var result []*blocks.Block
	for i, child := range children {
		if childMap, ok := child.(map[string]interface{}); ok {
			block := &blocks.Block{
				ID:       getString(childMap, "id"),
				Type:     blocks.BlockType(getString(childMap, "type")),
				Content:  convertContent(getMap(childMap, "content")),
				ParentID: parentID,
				Position: i,
			}
			if nestedChildren, ok := childMap["children"].([]interface{}); ok {
				block.Children = s.convertChildBlocks(nestedChildren, block.ID)
			}
			result = append(result, block)
		}
	}
	return result
}

// getString safely gets a string from a map.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getBool safely gets a bool from a map.
func getBool(m map[string]interface{}, key string) *bool {
	if v, ok := m[key].(bool); ok {
		return &v
	}
	return nil
}

// getMap safely gets a map from a map.
func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

// convertContent converts a map to blocks.Content.
func convertContent(m map[string]interface{}) blocks.Content {
	if m == nil {
		return blocks.Content{}
	}

	content := blocks.Content{
		Icon:        getString(m, "icon"),
		Color:       getString(m, "color"),
		Language:    getString(m, "language"),
		URL:         getString(m, "url"),
		Title:       getString(m, "title"),
		Description: getString(m, "description"),
		DatabaseID:  getString(m, "database_id"),
		SyncedFrom:  getString(m, "synced_from"),
		ButtonText:  getString(m, "button_text"),
		ButtonStyle: getString(m, "button_style"),
		Checked:     getBool(m, "checked"),
	}

	// Convert rich_text array
	if richText, ok := m["rich_text"].([]interface{}); ok {
		for _, rt := range richText {
			if rtMap, ok := rt.(map[string]interface{}); ok {
				richTextItem := blocks.RichText{
					Type: getString(rtMap, "type"),
					Text: getString(rtMap, "text"),
					Link: getString(rtMap, "link"),
				}
				// Convert annotations
				if ann, ok := rtMap["annotations"].(map[string]interface{}); ok {
					richTextItem.Annotations = blocks.Annotations{
						Bold:          ann["bold"] == true,
						Italic:        ann["italic"] == true,
						Strikethrough: ann["strikethrough"] == true,
						Underline:     ann["underline"] == true,
						Code:          ann["code"] == true,
						Color:         getString(ann, "color"),
					}
				}
				content.RichText = append(content.RichText, richTextItem)
			}
		}
	}

	// Convert expression for equations (stored in expression field)
	if expr, ok := m["expression"].(string); ok && expr != "" {
		// For equations, store expression in rich_text as a single item
		content.RichText = append(content.RichText, blocks.RichText{
			Type: "text",
			Text: expr,
		})
	}

	return content
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
