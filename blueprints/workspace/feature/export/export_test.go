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
