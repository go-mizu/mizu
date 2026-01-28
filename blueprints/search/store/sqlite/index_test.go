package sqlite

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

func TestIndexStore_IndexDocument(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	doc := &store.Document{
		URL:         "https://example.com/test",
		Title:       "Test Document",
		Description: "A test document description",
		Content:     "This is the full content of the test document with many words for testing word count",
		Domain:      "example.com",
		Language:    "en",
		ContentType: "text/html",
		Favicon:     "https://example.com/favicon.ico",
	}

	if err := idx.IndexDocument(ctx, doc); err != nil {
		t.Fatalf("IndexDocument() error = %v", err)
	}

	if doc.ID == "" {
		t.Error("expected ID to be set")
	}
	if doc.WordCount == 0 {
		t.Error("expected WordCount to be calculated")
	}
}

func TestIndexStore_IndexDocument_Upsert(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	doc := &store.Document{
		URL:     "https://example.com/upsert",
		Title:   "Original Title",
		Content: "Original content",
		Domain:  "example.com",
	}

	if err := idx.IndexDocument(ctx, doc); err != nil {
		t.Fatalf("first IndexDocument() error = %v", err)
	}

	// Update with same URL
	doc2 := &store.Document{
		URL:     "https://example.com/upsert",
		Title:   "Updated Title",
		Content: "Updated content with more words",
		Domain:  "example.com",
	}

	if err := idx.IndexDocument(ctx, doc2); err != nil {
		t.Fatalf("second IndexDocument() error = %v", err)
	}

	// Verify update
	retrieved, err := idx.GetDocumentByURL(ctx, "https://example.com/upsert")
	if err != nil {
		t.Fatalf("GetDocumentByURL() error = %v", err)
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("expected title = 'Updated Title', got %q", retrieved.Title)
	}
}

func TestIndexStore_GetDocument(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	doc := &store.Document{
		URL:         "https://example.com/get",
		Title:       "Get Test",
		Description: "Description",
		Content:     "Content",
		Domain:      "example.com",
		Language:    "en",
		ContentType: "text/html",
		Favicon:     "https://example.com/favicon.ico",
	}

	if err := idx.IndexDocument(ctx, doc); err != nil {
		t.Fatalf("IndexDocument() error = %v", err)
	}

	retrieved, err := idx.GetDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}

	if retrieved.URL != doc.URL {
		t.Errorf("URL = %q, want %q", retrieved.URL, doc.URL)
	}
	if retrieved.Title != doc.Title {
		t.Errorf("Title = %q, want %q", retrieved.Title, doc.Title)
	}
	if retrieved.Description != doc.Description {
		t.Errorf("Description = %q, want %q", retrieved.Description, doc.Description)
	}
}

func TestIndexStore_GetDocument_NotFound(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	_, err := idx.GetDocument(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent document")
	}
}

func TestIndexStore_GetDocumentByURL(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	doc := &store.Document{
		URL:    "https://example.com/byurl",
		Title:  "By URL Test",
		Domain: "example.com",
	}

	if err := idx.IndexDocument(ctx, doc); err != nil {
		t.Fatalf("IndexDocument() error = %v", err)
	}

	retrieved, err := idx.GetDocumentByURL(ctx, "https://example.com/byurl")
	if err != nil {
		t.Fatalf("GetDocumentByURL() error = %v", err)
	}

	if retrieved.Title != "By URL Test" {
		t.Errorf("Title = %q, want 'By URL Test'", retrieved.Title)
	}
}

func TestIndexStore_UpdateDocument(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	doc := &store.Document{
		URL:    "https://example.com/update",
		Title:  "Original",
		Domain: "example.com",
	}

	if err := idx.IndexDocument(ctx, doc); err != nil {
		t.Fatalf("IndexDocument() error = %v", err)
	}

	doc.Title = "Updated"
	doc.Description = "New description"
	doc.Content = "New content"

	if err := idx.UpdateDocument(ctx, doc); err != nil {
		t.Fatalf("UpdateDocument() error = %v", err)
	}

	retrieved, err := idx.GetDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}

	if retrieved.Title != "Updated" {
		t.Errorf("Title = %q, want 'Updated'", retrieved.Title)
	}
	if retrieved.Description != "New description" {
		t.Errorf("Description = %q, want 'New description'", retrieved.Description)
	}
}

func TestIndexStore_UpdateDocument_NotFound(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	doc := &store.Document{
		ID:    "nonexistent",
		Title: "Test",
	}

	err := idx.UpdateDocument(ctx, doc)
	if err == nil {
		t.Error("expected error for nonexistent document")
	}
}

func TestIndexStore_DeleteDocument(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	doc := &store.Document{
		URL:    "https://example.com/delete",
		Title:  "To Delete",
		Domain: "example.com",
	}

	if err := idx.IndexDocument(ctx, doc); err != nil {
		t.Fatalf("IndexDocument() error = %v", err)
	}

	if err := idx.DeleteDocument(ctx, doc.ID); err != nil {
		t.Fatalf("DeleteDocument() error = %v", err)
	}

	_, err := idx.GetDocument(ctx, doc.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestIndexStore_DeleteDocument_NotFound(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	err := idx.DeleteDocument(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent document")
	}
}

func TestIndexStore_ListDocuments(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	// Create multiple documents
	for i := 0; i < 5; i++ {
		doc := &store.Document{
			URL:    "https://example.com/list" + string(rune('0'+i)),
			Title:  "List Test " + string(rune('0'+i)),
			Domain: "example.com",
		}
		if err := idx.IndexDocument(ctx, doc); err != nil {
			t.Fatalf("IndexDocument() error = %v", err)
		}
	}

	// List with limit
	docs, err := idx.ListDocuments(ctx, 3, 0)
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}

	if len(docs) != 3 {
		t.Errorf("len(docs) = %d, want 3", len(docs))
	}

	// List with offset
	docs, err = idx.ListDocuments(ctx, 10, 2)
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}

	if len(docs) != 3 {
		t.Errorf("len(docs) = %d, want 3", len(docs))
	}
}

func TestIndexStore_BulkIndex(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	docs := []*store.Document{
		{URL: "https://example.com/bulk1", Title: "Bulk 1", Domain: "example.com"},
		{URL: "https://example.com/bulk2", Title: "Bulk 2", Domain: "example.com"},
		{URL: "https://example.com/bulk3", Title: "Bulk 3", Domain: "example.com"},
	}

	if err := idx.BulkIndex(ctx, docs); err != nil {
		t.Fatalf("BulkIndex() error = %v", err)
	}

	// Verify all docs have IDs
	for _, doc := range docs {
		if doc.ID == "" {
			t.Error("expected ID to be set for bulk indexed doc")
		}
	}

	// Verify docs exist
	for _, doc := range docs {
		_, err := idx.GetDocument(ctx, doc.ID)
		if err != nil {
			t.Errorf("GetDocument(%s) error = %v", doc.ID, err)
		}
	}
}

func TestIndexStore_GetIndexStats(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	// Add documents
	docs := []*store.Document{
		{URL: "https://go.dev/1", Title: "Go 1", Domain: "go.dev", Language: "en", ContentType: "text/html", Content: "Hello world"},
		{URL: "https://go.dev/2", Title: "Go 2", Domain: "go.dev", Language: "en", ContentType: "text/html", Content: "More content"},
		{URL: "https://rust-lang.org/1", Title: "Rust 1", Domain: "rust-lang.org", Language: "en", ContentType: "text/html", Content: "Rust stuff"},
	}

	for _, doc := range docs {
		if err := idx.IndexDocument(ctx, doc); err != nil {
			t.Fatalf("IndexDocument() error = %v", err)
		}
	}

	stats, err := idx.GetIndexStats(ctx)
	if err != nil {
		t.Fatalf("GetIndexStats() error = %v", err)
	}

	if stats.TotalDocuments != 3 {
		t.Errorf("TotalDocuments = %d, want 3", stats.TotalDocuments)
	}
	if stats.Languages["en"] != 3 {
		t.Errorf("Languages[en] = %d, want 3", stats.Languages["en"])
	}
	if stats.ContentTypes["text/html"] != 3 {
		t.Errorf("ContentTypes[text/html] = %d, want 3", stats.ContentTypes["text/html"])
	}
	if len(stats.TopDomains) != 2 {
		t.Errorf("len(TopDomains) = %d, want 2", len(stats.TopDomains))
	}
}

func TestIndexStore_RebuildIndex(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	// Add a document first
	doc := &store.Document{
		URL:    "https://example.com/rebuild",
		Title:  "Rebuild Test",
		Domain: "example.com",
	}
	if err := idx.IndexDocument(ctx, doc); err != nil {
		t.Fatalf("IndexDocument() error = %v", err)
	}

	if err := idx.RebuildIndex(ctx); err != nil {
		t.Errorf("RebuildIndex() error = %v", err)
	}
}

func TestIndexStore_OptimizeIndex(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	idx := s.Index()

	// Add a document first
	doc := &store.Document{
		URL:    "https://example.com/optimize",
		Title:  "Optimize Test",
		Domain: "example.com",
	}
	if err := idx.IndexDocument(ctx, doc); err != nil {
		t.Fatalf("IndexDocument() error = %v", err)
	}

	if err := idx.OptimizeIndex(ctx); err != nil {
		t.Errorf("OptimizeIndex() error = %v", err)
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  hello   world  ", 2},
		{"one two three four five", 5},
	}

	for _, tt := range tests {
		got := countWords(tt.input)
		if got != tt.want {
			t.Errorf("countWords(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
