package enrich

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// mockSmallWebStore implements store.SmallWebStore for testing.
type mockSmallWebStore struct {
	webEntries  []*types.SmallWebEntry
	newsEntries []*types.SmallWebEntry
}

func newMockSmallWebStore() *mockSmallWebStore {
	return &mockSmallWebStore{
		webEntries:  make([]*types.SmallWebEntry, 0),
		newsEntries: make([]*types.SmallWebEntry, 0),
	}
}

func (m *mockSmallWebStore) IndexEntry(ctx context.Context, entry *types.SmallWebEntry) error {
	if entry.SourceType == "web" {
		m.webEntries = append(m.webEntries, entry)
	} else {
		m.newsEntries = append(m.newsEntries, entry)
	}
	return nil
}

func (m *mockSmallWebStore) SearchWeb(ctx context.Context, query string, limit int) ([]*store.EnrichmentResult, error) {
	var results []*store.EnrichmentResult
	for i, e := range m.webEntries {
		if i >= limit {
			break
		}
		pub := e.PublishedAt
		results = append(results, &store.EnrichmentResult{
			Type:      types.EnrichTypeResult,
			Rank:      i + 1,
			URL:       e.URL,
			Title:     e.Title,
			Snippet:   e.Snippet,
			Published: &pub,
		})
	}
	return results, nil
}

func (m *mockSmallWebStore) SearchNews(ctx context.Context, query string, limit int) ([]*store.EnrichmentResult, error) {
	var results []*store.EnrichmentResult
	for i, e := range m.newsEntries {
		if i >= limit {
			break
		}
		pub := e.PublishedAt
		results = append(results, &store.EnrichmentResult{
			Type:      types.EnrichTypeResult,
			Rank:      i + 1,
			URL:       e.URL,
			Title:     e.Title,
			Snippet:   e.Snippet,
			Published: &pub,
		})
	}
	return results, nil
}

func (m *mockSmallWebStore) SeedSmallWeb(ctx context.Context) error {
	return nil
}

func TestService_SearchWeb(t *testing.T) {
	st := newMockSmallWebStore()
	now := time.Now()
	st.webEntries = []*types.SmallWebEntry{
		{
			URL:         "https://example1.com",
			Title:       "Example Blog Post 1",
			Snippet:     "This is a small web blog post about technology",
			SourceType:  "web",
			Domain:      "example1.com",
			PublishedAt: now,
		},
		{
			URL:         "https://example2.com",
			Title:       "Example Blog Post 2",
			Snippet:     "Another indie web article about programming",
			SourceType:  "web",
			Domain:      "example2.com",
			PublishedAt: now,
		},
	}

	svc := NewService(st)
	ctx := context.Background()

	resp, err := svc.SearchWeb(ctx, "technology", 10)
	if err != nil {
		t.Fatalf("SearchWeb() error = %v", err)
	}

	if len(resp.Data) != 2 {
		t.Errorf("SearchWeb() returned %d results, want 2", len(resp.Data))
	}

	if resp.Meta.Node != "local" {
		t.Errorf("SearchWeb() meta.node = %q, want %q", resp.Meta.Node, "local")
	}

	if resp.Meta.Ms < 0 {
		t.Error("SearchWeb() meta.ms should be >= 0")
	}
}

func TestService_SearchWeb_LimitResults(t *testing.T) {
	st := newMockSmallWebStore()
	now := time.Now()
	for i := 0; i < 20; i++ {
		st.webEntries = append(st.webEntries, &types.SmallWebEntry{
			URL:         "https://example.com/" + string(rune('a'+i)),
			Title:       "Post",
			Snippet:     "Content",
			SourceType:  "web",
			Domain:      "example.com",
			PublishedAt: now,
		})
	}

	svc := NewService(st)
	ctx := context.Background()

	resp, err := svc.SearchWeb(ctx, "test", 5)
	if err != nil {
		t.Fatalf("SearchWeb() error = %v", err)
	}

	if len(resp.Data) != 5 {
		t.Errorf("SearchWeb() returned %d results, want 5 (limit)", len(resp.Data))
	}
}

func TestService_SearchWeb_DefaultLimit(t *testing.T) {
	st := newMockSmallWebStore()
	now := time.Now()
	for i := 0; i < 20; i++ {
		st.webEntries = append(st.webEntries, &types.SmallWebEntry{
			URL:         "https://example.com/" + string(rune('a'+i)),
			Title:       "Post",
			Snippet:     "Content",
			SourceType:  "web",
			Domain:      "example.com",
			PublishedAt: now,
		})
	}

	svc := NewService(st)
	ctx := context.Background()

	// Pass 0 limit, should default to 10
	resp, err := svc.SearchWeb(ctx, "test", 0)
	if err != nil {
		t.Fatalf("SearchWeb() error = %v", err)
	}

	if len(resp.Data) != 10 {
		t.Errorf("SearchWeb() with limit=0 returned %d results, want 10 (default)", len(resp.Data))
	}
}

func TestService_SearchNews(t *testing.T) {
	st := newMockSmallWebStore()
	now := time.Now()
	st.newsEntries = []*types.SmallWebEntry{
		{
			URL:         "https://news1.com",
			Title:       "Breaking News Article",
			Snippet:     "Independent news coverage",
			SourceType:  "news",
			Domain:      "news1.com",
			PublishedAt: now,
		},
	}

	svc := NewService(st)
	ctx := context.Background()

	resp, err := svc.SearchNews(ctx, "breaking", 10)
	if err != nil {
		t.Fatalf("SearchNews() error = %v", err)
	}

	if len(resp.Data) != 1 {
		t.Errorf("SearchNews() returned %d results, want 1", len(resp.Data))
	}
}

func TestService_Index(t *testing.T) {
	st := newMockSmallWebStore()
	svc := NewService(st)
	ctx := context.Background()

	entry := &types.SmallWebEntry{
		URL:         "https://myblog.com/post",
		Title:       "My Blog Post",
		Snippet:     "This is my indie blog post",
		SourceType:  "web",
		Domain:      "myblog.com",
		PublishedAt: time.Now(),
	}

	err := svc.Index(ctx, entry)
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	if len(st.webEntries) != 1 {
		t.Errorf("Index() didn't add entry to store")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()

	// Small delay to ensure different timestamp
	time.Sleep(time.Millisecond)

	id2 := generateID()

	// IDs should be different
	if id1 == id2 {
		t.Error("generateID() should produce unique IDs")
	}

	// ID should be in expected format (timestamp with milliseconds)
	if len(id1) != 18 { // "20060102150405.000"
		t.Errorf("generateID() length = %d, want 18", len(id1))
	}
}

func TestToEnrichmentResults(t *testing.T) {
	now := time.Now()
	input := []*store.EnrichmentResult{
		{
			Type:      types.EnrichTypeResult,
			Rank:      1,
			URL:       "https://example.com",
			Title:     "Example",
			Snippet:   "This is an example",
			Published: &now,
		},
		{
			Type:      types.EnrichTypeResult,
			Rank:      2,
			URL:       "https://news.com",
			Title:     "News Article",
			Snippet:   "Breaking news",
			Published: &now,
		},
	}

	output := toEnrichmentResults(input)

	if len(output) != 2 {
		t.Fatalf("toEnrichmentResults() returned %d items, want 2", len(output))
	}

	if output[0].Rank != 1 {
		t.Errorf("output[0].Rank = %d, want 1", output[0].Rank)
	}

	if output[0].URL != "https://example.com" {
		t.Errorf("output[0].URL = %q, want %q", output[0].URL, "https://example.com")
	}
}

func TestToEnrichmentResults_EmptyInput(t *testing.T) {
	output := toEnrichmentResults([]*store.EnrichmentResult{})

	if len(output) != 0 {
		t.Errorf("toEnrichmentResults([]) returned %d items, want 0", len(output))
	}
}

func TestToEnrichmentResults_NilInput(t *testing.T) {
	output := toEnrichmentResults(nil)

	if len(output) != 0 {
		t.Errorf("toEnrichmentResults(nil) returned %d items, want 0", len(output))
	}
}
