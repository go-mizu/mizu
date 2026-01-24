// Package canvas provides a rich workspace for organizing AI research.
package canvas

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"
)

// Canvas represents a research workspace.
type Canvas struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Title     string    `json:"title"`
	Blocks    []Block   `json:"blocks"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Block represents a content block in the canvas.
type Block struct {
	ID        string    `json:"id"`
	CanvasID  string    `json:"canvas_id"`
	Type      BlockType `json:"type"`
	Content   string    `json:"content"`
	Meta      any       `json:"meta,omitempty"`
	Order     int       `json:"order"`
	CreatedAt time.Time `json:"created_at"`
}

// BlockType defines the type of content block.
type BlockType string

const (
	BlockTypeText       BlockType = "text"
	BlockTypeAIResponse BlockType = "ai_response"
	BlockTypeNote       BlockType = "note"
	BlockTypeCitation   BlockType = "citation"
	BlockTypeHeading    BlockType = "heading"
	BlockTypeDivider    BlockType = "divider"
	BlockTypeCode       BlockType = "code"
)

// ExportFormat defines export output formats.
type ExportFormat string

const (
	ExportMarkdown ExportFormat = "markdown"
	ExportHTML     ExportFormat = "html"
	ExportJSON     ExportFormat = "json"
)

// Store defines the interface for canvas storage.
type Store interface {
	// Create creates a new canvas.
	Create(ctx context.Context, canvas *Canvas) error

	// Get retrieves a canvas by ID.
	Get(ctx context.Context, id string) (*Canvas, error)

	// GetBySessionID retrieves a canvas by session ID.
	GetBySessionID(ctx context.Context, sessionID string) (*Canvas, error)

	// Update updates a canvas.
	Update(ctx context.Context, canvas *Canvas) error

	// Delete deletes a canvas.
	Delete(ctx context.Context, id string) error

	// AddBlock adds a block to a canvas.
	AddBlock(ctx context.Context, canvasID string, block *Block) error

	// UpdateBlock updates a block.
	UpdateBlock(ctx context.Context, block *Block) error

	// DeleteBlock deletes a block.
	DeleteBlock(ctx context.Context, blockID string) error

	// GetBlocks retrieves all blocks for a canvas.
	GetBlocks(ctx context.Context, canvasID string) ([]Block, error)

	// ReorderBlocks updates the order of blocks.
	ReorderBlocks(ctx context.Context, canvasID string, blockIDs []string) error
}

// Service manages canvases.
type Service struct {
	store Store
}

// New creates a new canvas service.
func New(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new canvas for a session.
func (s *Service) Create(ctx context.Context, sessionID, title string) (*Canvas, error) {
	canvas := &Canvas{
		ID:        generateID(),
		SessionID: sessionID,
		Title:     title,
		Blocks:    []Block{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if title == "" {
		canvas.Title = "Research Notes"
	}

	if err := s.store.Create(ctx, canvas); err != nil {
		return nil, err
	}

	return canvas, nil
}

// Get retrieves a canvas by ID.
func (s *Service) Get(ctx context.Context, id string) (*Canvas, error) {
	canvas, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	blocks, err := s.store.GetBlocks(ctx, id)
	if err != nil {
		return nil, err
	}
	canvas.Blocks = blocks

	return canvas, nil
}

// GetBySessionID retrieves or creates a canvas for a session.
func (s *Service) GetBySessionID(ctx context.Context, sessionID string) (*Canvas, error) {
	canvas, err := s.store.GetBySessionID(ctx, sessionID)
	if err != nil {
		// Create a new canvas if none exists
		return s.Create(ctx, sessionID, "Research Notes")
	}

	blocks, err := s.store.GetBlocks(ctx, canvas.ID)
	if err != nil {
		return nil, err
	}
	canvas.Blocks = blocks

	return canvas, nil
}

// Update updates a canvas.
func (s *Service) Update(ctx context.Context, canvas *Canvas) error {
	canvas.UpdatedAt = time.Now()
	return s.store.Update(ctx, canvas)
}

// Delete deletes a canvas.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// AddBlock adds a block to a canvas.
func (s *Service) AddBlock(ctx context.Context, canvasID string, blockType BlockType, content string, order int) (*Block, error) {
	block := &Block{
		ID:        generateID(),
		CanvasID:  canvasID,
		Type:      blockType,
		Content:   content,
		Order:     order,
		CreatedAt: time.Now(),
	}

	if err := s.store.AddBlock(ctx, canvasID, block); err != nil {
		return nil, err
	}

	return block, nil
}

// UpdateBlock updates a block.
func (s *Service) UpdateBlock(ctx context.Context, block *Block) error {
	return s.store.UpdateBlock(ctx, block)
}

// DeleteBlock deletes a block.
func (s *Service) DeleteBlock(ctx context.Context, blockID string) error {
	return s.store.DeleteBlock(ctx, blockID)
}

// ReorderBlocks updates the order of blocks.
func (s *Service) ReorderBlocks(ctx context.Context, canvasID string, blockIDs []string) error {
	return s.store.ReorderBlocks(ctx, canvasID, blockIDs)
}

// AddAIResponse adds an AI response block from a session message.
func (s *Service) AddAIResponse(ctx context.Context, canvasID string, content string, citations any) (*Block, error) {
	canvas, err := s.store.Get(ctx, canvasID)
	if err != nil {
		return nil, err
	}

	blocks, err := s.store.GetBlocks(ctx, canvasID)
	if err != nil {
		return nil, err
	}

	order := len(blocks)

	block := &Block{
		ID:        generateID(),
		CanvasID:  canvasID,
		Type:      BlockTypeAIResponse,
		Content:   content,
		Meta:      map[string]any{"citations": citations},
		Order:     order,
		CreatedAt: time.Now(),
	}

	if err := s.store.AddBlock(ctx, canvasID, block); err != nil {
		return nil, err
	}

	canvas.UpdatedAt = time.Now()
	_ = s.store.Update(ctx, canvas)

	return block, nil
}

// Export exports a canvas to the specified format.
func (s *Service) Export(ctx context.Context, id string, format ExportFormat) ([]byte, string, error) {
	canvas, err := s.Get(ctx, id)
	if err != nil {
		return nil, "", err
	}

	switch format {
	case ExportMarkdown:
		return s.exportMarkdown(canvas)
	case ExportHTML:
		return s.exportHTML(canvas)
	case ExportJSON:
		return s.exportJSON(canvas)
	default:
		return nil, "", fmt.Errorf("unsupported export format: %s", format)
	}
}

func (s *Service) exportMarkdown(canvas *Canvas) ([]byte, string, error) {
	var buf bytes.Buffer

	buf.WriteString("# " + canvas.Title + "\n\n")
	buf.WriteString(fmt.Sprintf("*Created: %s*\n\n", canvas.CreatedAt.Format("2006-01-02 15:04")))
	buf.WriteString("---\n\n")

	for _, block := range canvas.Blocks {
		switch block.Type {
		case BlockTypeHeading:
			buf.WriteString("## " + block.Content + "\n\n")
		case BlockTypeText, BlockTypeAIResponse:
			buf.WriteString(block.Content + "\n\n")
		case BlockTypeNote:
			buf.WriteString("> **Note:** " + block.Content + "\n\n")
		case BlockTypeCitation:
			buf.WriteString("- " + block.Content + "\n")
		case BlockTypeDivider:
			buf.WriteString("---\n\n")
		case BlockTypeCode:
			buf.WriteString("```\n" + block.Content + "\n```\n\n")
		}
	}

	return buf.Bytes(), "text/markdown", nil
}

func (s *Service) exportHTML(canvas *Canvas) ([]byte, string, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>{{.Title}}</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 800px; margin: 0 auto; padding: 2rem; line-height: 1.6; }
        h1 { border-bottom: 2px solid #333; padding-bottom: 0.5rem; }
        h2 { color: #333; }
        .meta { color: #666; font-style: italic; margin-bottom: 2rem; }
        .note { background: #fff3cd; border-left: 4px solid #ffc107; padding: 1rem; margin: 1rem 0; }
        .citation { color: #666; padding-left: 1rem; border-left: 2px solid #ddd; margin: 0.5rem 0; }
        .ai-response { background: #f8f9fa; padding: 1rem; border-radius: 8px; margin: 1rem 0; }
        hr { border: none; border-top: 1px solid #ddd; margin: 2rem 0; }
        pre { background: #f4f4f4; padding: 1rem; overflow-x: auto; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p class="meta">Created: {{.CreatedAt}}</p>
    <hr>
    {{range .Blocks}}
    {{if eq .Type "heading"}}<h2>{{.Content}}</h2>{{end}}
    {{if eq .Type "text"}}<p>{{.Content}}</p>{{end}}
    {{if eq .Type "ai_response"}}<div class="ai-response">{{.Content}}</div>{{end}}
    {{if eq .Type "note"}}<div class="note"><strong>Note:</strong> {{.Content}}</div>{{end}}
    {{if eq .Type "citation"}}<div class="citation">{{.Content}}</div>{{end}}
    {{if eq .Type "divider"}}<hr>{{end}}
    {{if eq .Type "code"}}<pre>{{.Content}}</pre>{{end}}
    {{end}}
</body>
</html>`

	t, err := template.New("html").Parse(tmpl)
	if err != nil {
		return nil, "", err
	}

	data := struct {
		Title     string
		CreatedAt string
		Blocks    []Block
	}{
		Title:     canvas.Title,
		CreatedAt: canvas.CreatedAt.Format("2006-01-02 15:04"),
		Blocks:    canvas.Blocks,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), "text/html", nil
}

func (s *Service) exportJSON(canvas *Canvas) ([]byte, string, error) {
	data, err := json.MarshalIndent(canvas, "", "  ")
	if err != nil {
		return nil, "", err
	}
	return data, "application/json", nil
}

// MarshalMeta converts meta to JSON string for storage.
func MarshalMeta(meta any) string {
	if meta == nil {
		return "{}"
	}
	data, _ := json.Marshal(meta)
	return string(data)
}

// UnmarshalMeta parses meta from JSON string.
func UnmarshalMeta(data string) map[string]any {
	if data == "" || data == "null" {
		return nil
	}
	var meta map[string]any
	_ = json.Unmarshal([]byte(data), &meta)
	return meta
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// SanitizeContent removes potentially harmful content.
func SanitizeContent(s string) string {
	// Basic sanitization - remove script tags
	s = strings.ReplaceAll(s, "<script", "&lt;script")
	s = strings.ReplaceAll(s, "</script>", "&lt;/script&gt;")
	return s
}
