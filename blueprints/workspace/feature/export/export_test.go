package export

import (
	"strings"
	"testing"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

func TestMarkdownConverter_Paragraph(t *testing.T) {
	converter := NewMarkdownConverter()

	page := &ExportedPage{
		Title: "Test Page",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockParagraph,
				Content: blocks.Content{
					RichText: []blocks.RichText{
						{Type: "text", Text: "Hello, world!"},
					},
				},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "# Test Page") {
		t.Error("Expected title in output")
	}
	if !strings.Contains(content, "Hello, world!") {
		t.Error("Expected paragraph text in output")
	}
}

func TestMarkdownConverter_Headings(t *testing.T) {
	converter := NewMarkdownConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockHeading1,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "Heading 1"}},
				},
			},
			{
				Type: blocks.BlockHeading2,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "Heading 2"}},
				},
			},
			{
				Type: blocks.BlockHeading3,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "Heading 3"}},
				},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "# Heading 1") {
		t.Error("Expected H1 in output")
	}
	if !strings.Contains(content, "## Heading 2") {
		t.Error("Expected H2 in output")
	}
	if !strings.Contains(content, "### Heading 3") {
		t.Error("Expected H3 in output")
	}
}

func TestMarkdownConverter_Lists(t *testing.T) {
	converter := NewMarkdownConverter()

	checked := true
	unchecked := false

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockBulletList,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "Bullet item"}},
				},
			},
			{
				Type: blocks.BlockNumberList,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "Numbered item"}},
				},
			},
			{
				Type: blocks.BlockTodo,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "Checked task"}},
					Checked:  &checked,
				},
			},
			{
				Type: blocks.BlockTodo,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "Unchecked task"}},
					Checked:  &unchecked,
				},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "- Bullet item") {
		t.Error("Expected bullet list in output")
	}
	if !strings.Contains(content, "1. Numbered item") {
		t.Error("Expected numbered list in output")
	}
	if !strings.Contains(content, "- [x] Checked task") {
		t.Error("Expected checked todo in output")
	}
	if !strings.Contains(content, "- [ ] Unchecked task") {
		t.Error("Expected unchecked todo in output")
	}
}

func TestMarkdownConverter_CodeBlock(t *testing.T) {
	converter := NewMarkdownConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockCode,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "const x = 1;"}},
					Language: "javascript",
				},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "```javascript") {
		t.Error("Expected code block with language in output")
	}
	if !strings.Contains(content, "const x = 1;") {
		t.Error("Expected code content in output")
	}
}

func TestMarkdownConverter_Quote(t *testing.T) {
	converter := NewMarkdownConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockQuote,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "A quote"}},
				},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "> A quote") {
		t.Error("Expected quote in output")
	}
}

func TestMarkdownConverter_RichTextFormatting(t *testing.T) {
	converter := NewMarkdownConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockParagraph,
				Content: blocks.Content{
					RichText: []blocks.RichText{
						{
							Type: "text",
							Text: "bold",
							Annotations: blocks.Annotations{
								Bold: true,
							},
						},
						{Type: "text", Text: " and "},
						{
							Type: "text",
							Text: "italic",
							Annotations: blocks.Annotations{
								Italic: true,
							},
						},
						{Type: "text", Text: " and "},
						{
							Type: "text",
							Text: "code",
							Annotations: blocks.Annotations{
								Code: true,
							},
						},
					},
				},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "**bold**") {
		t.Error("Expected bold formatting in output")
	}
	if !strings.Contains(content, "*italic*") {
		t.Error("Expected italic formatting in output")
	}
	if !strings.Contains(content, "`code`") {
		t.Error("Expected code formatting in output")
	}
}

func TestHTMLConverter_BasicStructure(t *testing.T) {
	converter := NewHTMLConverter()

	page := &ExportedPage{
		Title: "Test Page",
		Icon:  "📄",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockParagraph,
				Content: blocks.Content{
					RichText: []blocks.RichText{
						{Type: "text", Text: "Hello, world!"},
					},
				},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Error("Expected DOCTYPE in output")
	}
	if !strings.Contains(content, "<title>Test Page</title>") {
		t.Error("Expected title tag in output")
	}
	if !strings.Contains(content, "<h1 class=\"page-title\">Test Page</h1>") {
		t.Error("Expected page title in output")
	}
	if !strings.Contains(content, "<p>Hello, world!</p>") {
		t.Error("Expected paragraph in output")
	}
}

func TestHTMLConverter_Callout(t *testing.T) {
	converter := NewHTMLConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockCallout,
				Content: blocks.Content{
					RichText: []blocks.RichText{
						{Type: "text", Text: "Important note"},
					},
					Icon:  "💡",
					Color: "yellow",
				},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "callout callout-yellow") {
		t.Error("Expected callout with color class in output")
	}
	if !strings.Contains(content, "💡") {
		t.Error("Expected callout icon in output")
	}
	if !strings.Contains(content, "Important note") {
		t.Error("Expected callout content in output")
	}
}

func TestHTMLConverter_Image(t *testing.T) {
	converter := NewHTMLConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockImage,
				Content: blocks.Content{
					URL: "https://example.com/image.png",
					Caption: []blocks.RichText{
						{Type: "text", Text: "An image"},
					},
				},
			},
		},
	}

	// With images included
	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "<img src=\"https://example.com/image.png\"") {
		t.Error("Expected image in output")
	}
	if !strings.Contains(content, "<figcaption>An image</figcaption>") {
		t.Error("Expected caption in output")
	}

	// Without images
	result, err = converter.Convert(page, &Request{IncludeImages: false})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content = string(result)
	if strings.Contains(content, "<img") {
		t.Error("Expected no image when IncludeImages is false")
	}
}

func TestCSVConverter_BasicDatabase(t *testing.T) {
	converter := NewCSVConverter()

	db := &ExportedDatabase{
		Title: "Tasks",
		Properties: []databases.Property{
			{ID: "title", Name: "Name", Type: databases.PropTitle},
			{ID: "status", Name: "Status", Type: databases.PropSelect},
			{ID: "done", Name: "Done", Type: databases.PropCheckbox},
		},
		Rows: []*ExportedPage{
			{
				Title: "Task 1",
				Properties: pages.Properties{
					"title":  pages.PropertyValue{Type: "title", Value: "Task 1"},
					"status": pages.PropertyValue{Type: "select", Value: map[string]interface{}{"name": "In Progress"}},
					"done":   pages.PropertyValue{Type: "checkbox", Value: false},
				},
			},
			{
				Title: "Task 2",
				Properties: pages.Properties{
					"title":  pages.PropertyValue{Type: "title", Value: "Task 2"},
					"status": pages.PropertyValue{Type: "select", Value: map[string]interface{}{"name": "Done"}},
					"done":   pages.PropertyValue{Type: "checkbox", Value: true},
				},
			},
		},
	}

	result, err := converter.Convert(db)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	lines := strings.Split(strings.TrimSpace(content), "\n")

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines (header + 2 rows), got %d", len(lines))
	}

	// Check header
	if !strings.Contains(lines[0], "Name") {
		t.Error("Expected Name column in header")
	}
	if !strings.Contains(lines[0], "Status") {
		t.Error("Expected Status column in header")
	}

	// Check data
	if !strings.Contains(lines[1], "Task 1") {
		t.Error("Expected Task 1 in first row")
	}
	if !strings.Contains(lines[1], "In Progress") {
		t.Error("Expected In Progress status in first row")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal.txt", "normal.txt"},
		{"file/with/slashes", "file-with-slashes"},
		{"file:with:colons", "file-with-colons"},
		{"file*with*stars", "file-with-stars"},
		{"  spaces  ", "spaces"},
		{"", "untitled"},
		{strings.Repeat("a", 200), strings.Repeat("a", 100)},
	}

	for _, tt := range tests {
		result := sanitizeFilename(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCreateBundle_SinglePage(t *testing.T) {
	page := &ExportedPage{
		Title: "Test Page",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockParagraph,
				Content: blocks.Content{
					RichText: []blocks.RichText{
						{Type: "text", Text: "Hello"},
					},
				},
			},
		},
	}

	// Test Markdown export
	bundle, err := CreateBundle(page, &Request{
		Format:          FormatMarkdown,
		IncludeSubpages: false,
		IncludeImages:   true,
	})
	if err != nil {
		t.Fatalf("CreateBundle failed: %v", err)
	}

	if bundle.Filename != "Test Page.md" {
		t.Errorf("Expected filename 'Test Page.md', got %q", bundle.Filename)
	}
	if bundle.ContentType != "text/markdown; charset=utf-8" {
		t.Errorf("Expected content type 'text/markdown; charset=utf-8', got %q", bundle.ContentType)
	}
	if bundle.PageCount != 1 {
		t.Errorf("Expected page count 1, got %d", bundle.PageCount)
	}
	if !strings.Contains(string(bundle.Data), "# Test Page") {
		t.Error("Expected page title in output")
	}

	// Test HTML export
	bundle, err = CreateBundle(page, &Request{
		Format:          FormatHTML,
		IncludeSubpages: false,
		IncludeImages:   true,
	})
	if err != nil {
		t.Fatalf("CreateBundle failed: %v", err)
	}

	if bundle.Filename != "Test Page.html" {
		t.Errorf("Expected filename 'Test Page.html', got %q", bundle.Filename)
	}
	if !strings.Contains(string(bundle.Data), "<!DOCTYPE html>") {
		t.Error("Expected HTML doctype in output")
	}
}

func TestCreateBundle_MultiPage(t *testing.T) {
	page := &ExportedPage{
		Title: "Parent Page",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockParagraph,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "Parent content"}},
				},
			},
		},
		Children: []*ExportedPage{
			{
				Title: "Child Page 1",
				Blocks: []*blocks.Block{
					{
						Type: blocks.BlockParagraph,
						Content: blocks.Content{
							RichText: []blocks.RichText{{Type: "text", Text: "Child 1 content"}},
						},
					},
				},
			},
			{
				Title: "Child Page 2",
				Blocks: []*blocks.Block{
					{
						Type: blocks.BlockParagraph,
						Content: blocks.Content{
							RichText: []blocks.RichText{{Type: "text", Text: "Child 2 content"}},
						},
					},
				},
			},
		},
	}

	bundle, err := CreateBundle(page, &Request{
		Format:          FormatMarkdown,
		IncludeSubpages: true,
		IncludeImages:   true,
		CreateFolders:   true,
	})
	if err != nil {
		t.Fatalf("CreateBundle failed: %v", err)
	}

	if bundle.Filename != "Parent Page.zip" {
		t.Errorf("Expected filename 'Parent Page.zip', got %q", bundle.Filename)
	}
	if bundle.ContentType != "application/zip" {
		t.Errorf("Expected content type 'application/zip', got %q", bundle.ContentType)
	}
	if bundle.PageCount != 3 {
		t.Errorf("Expected page count 3, got %d", bundle.PageCount)
	}
}

func TestBundler_AddFile(t *testing.T) {
	bundler := NewBundler()

	err := bundler.AddFile("test.txt", []byte("Hello, world!"))
	if err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	if bundler.FileCount() != 1 {
		t.Errorf("Expected file count 1, got %d", bundler.FileCount())
	}

	// Adding same file again should be a no-op
	err = bundler.AddFile("test.txt", []byte("Different content"))
	if err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	if bundler.FileCount() != 1 {
		t.Errorf("Expected file count still 1, got %d", bundler.FileCount())
	}

	// Add another file
	err = bundler.AddFile("other.txt", []byte("Other content"))
	if err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	if bundler.FileCount() != 2 {
		t.Errorf("Expected file count 2, got %d", bundler.FileCount())
	}

	data, err := bundler.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty ZIP data")
	}
}

func TestMarkdownConverter_Toggle(t *testing.T) {
	converter := NewMarkdownConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockToggle,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "Toggle Header"}},
				},
				Children: []*blocks.Block{
					{
						Type: blocks.BlockParagraph,
						Content: blocks.Content{
							RichText: []blocks.RichText{{Type: "text", Text: "Toggle content"}},
						},
					},
				},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "<details>") {
		t.Error("Expected <details> tag for toggle block")
	}
	if !strings.Contains(content, "<summary>Toggle Header</summary>") {
		t.Error("Expected summary with toggle header")
	}
	if !strings.Contains(content, "Toggle content") {
		t.Error("Expected toggle content in output")
	}
}

func TestMarkdownConverter_Divider(t *testing.T) {
	converter := NewMarkdownConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type:    blocks.BlockParagraph,
				Content: blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Before"}}},
			},
			{
				Type:    blocks.BlockDivider,
				Content: blocks.Content{},
			},
			{
				Type:    blocks.BlockParagraph,
				Content: blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "After"}}},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "---") {
		t.Error("Expected horizontal rule (---) for divider")
	}
}

func TestMarkdownConverter_Image(t *testing.T) {
	converter := NewMarkdownConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockImage,
				Content: blocks.Content{
					URL: "https://example.com/image.png",
					Caption: []blocks.RichText{
						{Type: "text", Text: "Image caption"},
					},
				},
			},
		},
	}

	// With images included
	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "![Image caption](https://example.com/image.png)") {
		t.Error("Expected image markdown with caption as alt text")
	}

	// Without images
	result, err = converter.Convert(page, &Request{IncludeImages: false})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content = string(result)
	if strings.Contains(content, "![") {
		t.Error("Expected no image when IncludeImages is false")
	}
}

func TestHTMLConverter_Toggle(t *testing.T) {
	converter := NewHTMLConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockToggle,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "Toggle Title"}},
				},
				Children: []*blocks.Block{
					{
						Type: blocks.BlockParagraph,
						Content: blocks.Content{
							RichText: []blocks.RichText{{Type: "text", Text: "Nested content"}},
						},
					},
				},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "<details") {
		t.Error("Expected <details> element for toggle")
	}
	if !strings.Contains(content, "<summary>Toggle Title</summary>") {
		t.Error("Expected summary with toggle title")
	}
	if !strings.Contains(content, "Nested content") {
		t.Error("Expected nested content in toggle")
	}
}

func TestHTMLConverter_Divider(t *testing.T) {
	converter := NewHTMLConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type:    blocks.BlockDivider,
				Content: blocks.Content{},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "<hr") {
		t.Error("Expected <hr> element for divider")
	}
}

func TestHTMLConverter_Table(t *testing.T) {
	converter := NewHTMLConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockTable,
				Content: blocks.Content{
					TableWidth: 2,
					HasHeader:  true,
				},
				Children: []*blocks.Block{
					{
						Type: blocks.BlockTableRow,
						Content: blocks.Content{
							RichText: []blocks.RichText{
								{Type: "text", Text: "Header 1"},
								{Type: "text", Text: "Header 2"},
							},
						},
					},
					{
						Type: blocks.BlockTableRow,
						Content: blocks.Content{
							RichText: []blocks.RichText{
								{Type: "text", Text: "Cell 1"},
								{Type: "text", Text: "Cell 2"},
							},
						},
					},
				},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "<table") {
		t.Error("Expected <table> element")
	}
}

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name      string
		req       *Request
		wantError bool
		errContains string
	}{
		{
			name:      "missing page_id",
			req:       &Request{Format: FormatMarkdown},
			wantError: true,
			errContains: "page_id",
		},
		{
			name:      "missing format",
			req:       &Request{PageID: "page-123"},
			wantError: true,
			errContains: "format",
		},
		{
			name:      "invalid format",
			req:       &Request{PageID: "page-123", Format: "invalid"},
			wantError: true,
			errContains: "invalid format",
		},
		{
			name: "valid markdown request",
			req:  &Request{PageID: "page-123", Format: FormatMarkdown},
			wantError: false,
		},
		{
			name: "valid html request",
			req:  &Request{PageID: "page-123", Format: FormatHTML},
			wantError: false,
		},
		{
			name: "valid pdf request",
			req:  &Request{PageID: "page-123", Format: FormatPDF},
			wantError: false,
		},
	}

	svc := &Service{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateRequest(tt.req)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateRequest_Defaults(t *testing.T) {
	svc := &Service{}
	req := &Request{PageID: "page-123", Format: FormatPDF}

	err := svc.validateRequest(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check defaults are applied
	if req.PageSize != PageSizeLetter {
		t.Errorf("Expected default page size 'letter', got %q", req.PageSize)
	}
	if req.Orientation != OrientationPortrait {
		t.Errorf("Expected default orientation 'portrait', got %q", req.Orientation)
	}
	if req.Scale != 100 {
		t.Errorf("Expected default scale 100, got %d", req.Scale)
	}
}

func TestValidateRequest_ScaleRange(t *testing.T) {
	svc := &Service{}

	// Scale = 0 should default to 100
	req := &Request{PageID: "page-123", Format: FormatPDF, Scale: 0}
	_ = svc.validateRequest(req)
	if req.Scale != 100 {
		t.Errorf("Expected scale 100 for 0 input, got %d", req.Scale)
	}

	// Scale > 200 should default to 100
	req = &Request{PageID: "page-123", Format: FormatPDF, Scale: 300}
	_ = svc.validateRequest(req)
	if req.Scale != 100 {
		t.Errorf("Expected scale 100 for 300 input, got %d", req.Scale)
	}

	// Valid scale should be preserved
	req = &Request{PageID: "page-123", Format: FormatPDF, Scale: 75}
	_ = svc.validateRequest(req)
	if req.Scale != 75 {
		t.Errorf("Expected scale 75, got %d", req.Scale)
	}
}

func TestGetFilename(t *testing.T) {
	tests := []struct {
		title    string
		format   Format
		isZip    bool
		expected string
	}{
		{"My Page", FormatMarkdown, false, "My Page.md"},
		{"My Page", FormatHTML, false, "My Page.html"},
		{"My Page", FormatPDF, false, "My Page.pdf"},
		{"My Page", FormatMarkdown, true, "My Page.zip"},
		{"", FormatMarkdown, false, "untitled.md"},
	}

	for _, tt := range tests {
		result := GetFilename(tt.title, tt.format, tt.isZip)
		if result != tt.expected {
			t.Errorf("GetFilename(%q, %q, %v) = %q, want %q",
				tt.title, tt.format, tt.isZip, result, tt.expected)
		}
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		format   Format
		isZip    bool
		expected string
	}{
		{FormatMarkdown, false, "text/markdown; charset=utf-8"},
		{FormatHTML, false, "text/html; charset=utf-8"},
		{FormatPDF, false, "application/pdf"},
		{FormatMarkdown, true, "application/zip"},
		{FormatHTML, true, "application/zip"},
	}

	for _, tt := range tests {
		result := GetContentType(tt.format, tt.isZip)
		if result != tt.expected {
			t.Errorf("GetContentType(%q, %v) = %q, want %q",
				tt.format, tt.isZip, result, tt.expected)
		}
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"file.zip", "application/zip"},
		{"file.pdf", "application/pdf"},
		{"file.html", "text/html; charset=utf-8"},
		{"file.HTML", "text/html; charset=utf-8"},
		{"file.md", "text/markdown; charset=utf-8"},
		{"file.csv", "text/csv; charset=utf-8"},
		{"file.unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		result := DetectContentType(tt.filename)
		if result != tt.expected {
			t.Errorf("DetectContentType(%q) = %q, want %q",
				tt.filename, result, tt.expected)
		}
	}
}

func TestMarkdownConverter_Callout(t *testing.T) {
	converter := NewMarkdownConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockCallout,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "Important note"}},
					Icon:     "💡",
					Color:    "yellow",
				},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "> 💡") {
		t.Error("Expected callout with icon as blockquote")
	}
	if !strings.Contains(content, "Important note") {
		t.Error("Expected callout content")
	}
}

func TestHTMLConverter_Bookmark(t *testing.T) {
	converter := NewHTMLConverter()

	page := &ExportedPage{
		Title: "Test",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockBookmark,
				Content: blocks.Content{
					URL: "https://example.com",
					Caption: []blocks.RichText{
						{Type: "text", Text: "Example Site"},
					},
				},
			},
		},
	}

	result, err := converter.Convert(page, &Request{IncludeImages: true})
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	content := string(result)
	if !strings.Contains(content, "https://example.com") {
		t.Error("Expected bookmark URL in output")
	}
	if !strings.Contains(content, "bookmark") {
		t.Error("Expected bookmark class in output")
	}
}

func TestCreateBundle_PDF(t *testing.T) {
	page := &ExportedPage{
		Title: "Test Page",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockParagraph,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "PDF content"}},
				},
			},
		},
	}

	bundle, err := CreateBundle(page, &Request{
		Format:      FormatPDF,
		PageSize:    PageSizeA4,
		Orientation: OrientationPortrait,
		Scale:       100,
	})
	if err != nil {
		// PDF may fail without chromedp/wkhtmltopdf, which is expected
		t.Skipf("PDF creation skipped (likely no PDF renderer available): %v", err)
	}

	if bundle.Filename != "Test Page.pdf" {
		t.Errorf("Expected filename 'Test Page.pdf', got %q", bundle.Filename)
	}
	if bundle.ContentType != "application/pdf" {
		t.Errorf("Expected content type 'application/pdf', got %q", bundle.ContentType)
	}
}

func TestCreateBundle_PDF_ValidFormat(t *testing.T) {
	page := &ExportedPage{
		Title: "Test Page",
		Blocks: []*blocks.Block{
			{
				Type: blocks.BlockParagraph,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "PDF content"}},
				},
			},
			{
				Type: blocks.BlockHeading1,
				Content: blocks.Content{
					RichText: []blocks.RichText{{Type: "text", Text: "A Heading"}},
				},
			},
		},
	}

	bundle, err := CreateBundle(page, &Request{
		Format:      FormatPDF,
		PageSize:    PageSizeA4,
		Orientation: OrientationPortrait,
		Scale:       100,
	})
	if err != nil {
		t.Skipf("PDF creation skipped (likely no PDF renderer available): %v", err)
	}

	// Verify the output is actually a valid PDF (starts with %PDF magic bytes)
	if len(bundle.Data) < 4 {
		t.Fatalf("PDF data too short: %d bytes", len(bundle.Data))
	}

	header := string(bundle.Data[:4])
	if header != "%PDF" {
		t.Errorf("Invalid PDF header: expected '%%PDF', got %q", header)
		if len(bundle.Data) > 100 {
			t.Errorf("First 100 bytes: %s", string(bundle.Data[:100]))
		} else {
			t.Errorf("Full content: %s", string(bundle.Data))
		}
	}

	// Check for PDF EOF marker (should end with %%EOF)
	if len(bundle.Data) > 5 {
		// PDF files should contain %%EOF somewhere near the end
		tail := string(bundle.Data[len(bundle.Data)-20:])
		if !strings.Contains(tail, "%%EOF") {
			// Not a hard failure - some PDF generators may format differently
			t.Logf("Warning: PDF may be truncated - no %%EOF found in last 20 bytes")
		}
	}
}
