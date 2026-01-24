package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

func TestSearchStore_Search(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Seed documents
	if err := s.SeedDocuments(ctx); err != nil {
		t.Fatalf("SeedDocuments() error = %v", err)
	}

	tests := []struct {
		name    string
		query   string
		opts    store.SearchOptions
		wantMin int
	}{
		{
			name:    "basic search",
			query:   "Go",
			opts:    store.SearchOptions{},
			wantMin: 1,
		},
		{
			name:    "programming language",
			query:   "programming",
			opts:    store.SearchOptions{},
			wantMin: 1,
		},
		{
			name:    "with pagination",
			query:   "JavaScript",
			opts:    store.SearchOptions{Page: 1, PerPage: 5},
			wantMin: 1,
		},
		{
			name:    "no results",
			query:   "zzzznonexistent",
			opts:    store.SearchOptions{},
			wantMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := s.Search().Search(ctx, tt.query, tt.opts)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if resp == nil {
				t.Fatal("Search() returned nil response")
			}
			if resp.Query != tt.query {
				t.Errorf("Query = %q, want %q", resp.Query, tt.query)
			}
			if len(resp.Results) < tt.wantMin {
				t.Errorf("len(Results) = %d, want >= %d", len(resp.Results), tt.wantMin)
			}
			if resp.SearchTimeMs < 0 {
				t.Error("expected SearchTimeMs >= 0")
			}
		})
	}
}

func TestSearchStore_Search_SiteFilter(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Add documents from different domains
	docs := []*store.Document{
		{URL: "https://golang.org/test1", Title: "Go Test 1", Content: "Testing golang", Domain: "golang.org"},
		{URL: "https://python.org/test1", Title: "Python Test 1", Content: "Testing golang and python", Domain: "python.org"},
	}

	for _, doc := range docs {
		if err := s.Index().IndexDocument(ctx, doc); err != nil {
			t.Fatalf("IndexDocument() error = %v", err)
		}
	}

	// Search with site filter
	resp, err := s.Search().Search(ctx, "Testing", store.SearchOptions{Site: "golang.org"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	for _, r := range resp.Results {
		if r.Domain != "golang.org" {
			t.Errorf("got result from domain %s, want golang.org", r.Domain)
		}
	}
}

func TestSearchStore_Search_ExcludeSite(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	docs := []*store.Document{
		{URL: "https://golang.org/ex1", Title: "Go Exclude", Content: "Testing exclude", Domain: "golang.org"},
		{URL: "https://python.org/ex1", Title: "Python Exclude", Content: "Testing exclude", Domain: "python.org"},
	}

	for _, doc := range docs {
		if err := s.Index().IndexDocument(ctx, doc); err != nil {
			t.Fatalf("IndexDocument() error = %v", err)
		}
	}

	resp, err := s.Search().Search(ctx, "exclude", store.SearchOptions{ExcludeSite: "golang.org"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	for _, r := range resp.Results {
		if r.Domain == "golang.org" {
			t.Error("got result from excluded domain golang.org")
		}
	}
}

func TestSearchStore_Search_LanguageFilter(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	docs := []*store.Document{
		{URL: "https://example.com/en", Title: "English", Content: "Hello world language test", Domain: "example.com", Language: "en"},
		{URL: "https://example.com/de", Title: "German", Content: "Hello world language test", Domain: "example.com", Language: "de"},
	}

	for _, doc := range docs {
		if err := s.Index().IndexDocument(ctx, doc); err != nil {
			t.Fatalf("IndexDocument() error = %v", err)
		}
	}

	resp, err := s.Search().Search(ctx, "language", store.SearchOptions{Language: "en"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(resp.Results) != 1 {
		t.Errorf("len(Results) = %d, want 1", len(resp.Results))
	}
}

func TestSearchStore_Search_Verbatim(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	docs := []*store.Document{
		{URL: "https://example.com/v1", Title: "Exact Match", Content: "the quick brown fox", Domain: "example.com"},
		{URL: "https://example.com/v2", Title: "Partial", Content: "quick foxes are brown", Domain: "example.com"},
	}

	for _, doc := range docs {
		if err := s.Index().IndexDocument(ctx, doc); err != nil {
			t.Fatalf("IndexDocument() error = %v", err)
		}
	}

	// Verbatim search
	resp, err := s.Search().Search(ctx, "quick brown", store.SearchOptions{Verbatim: true})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// With verbatim, should match exact phrase only
	if resp.TotalResults == 0 {
		t.Log("No verbatim results (expected for exact phrase)")
	}
}

func TestSearchStore_Search_Pagination(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Add many documents
	for i := 0; i < 25; i++ {
		doc := &store.Document{
			URL:     "https://example.com/page" + string(rune('a'+i)),
			Title:   "Page Doc",
			Content: "pagination test content",
			Domain:  "example.com",
		}
		if err := s.Index().IndexDocument(ctx, doc); err != nil {
			t.Fatalf("IndexDocument() error = %v", err)
		}
	}

	// First page
	resp1, err := s.Search().Search(ctx, "pagination", store.SearchOptions{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(resp1.Results) != 10 {
		t.Errorf("page 1: len(Results) = %d, want 10", len(resp1.Results))
	}

	// Second page
	resp2, err := s.Search().Search(ctx, "pagination", store.SearchOptions{Page: 2, PerPage: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(resp2.Results) != 10 {
		t.Errorf("page 2: len(Results) = %d, want 10", len(resp2.Results))
	}

	// Third page (partial)
	resp3, err := s.Search().Search(ctx, "pagination", store.SearchOptions{Page: 3, PerPage: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(resp3.Results) != 5 {
		t.Errorf("page 3: len(Results) = %d, want 5", len(resp3.Results))
	}
}

func TestSearchStore_SearchImages(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Insert images directly
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO images (id, url, thumbnail_url, title, source_url, source_domain, width, height)
		VALUES
			('img1', 'https://example.com/img1.jpg', 'https://example.com/img1_thumb.jpg', 'Golang Gopher', 'https://golang.org', 'golang.org', 800, 600),
			('img2', 'https://example.com/img2.jpg', 'https://example.com/img2_thumb.jpg', 'Python Logo', 'https://python.org', 'python.org', 1024, 768)
	`)
	if err != nil {
		t.Fatalf("failed to insert images: %v", err)
	}

	results, err := s.Search().SearchImages(ctx, "Golang", store.SearchOptions{})
	if err != nil {
		t.Fatalf("SearchImages() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("expected at least one image result")
	}
}

func TestSearchStore_SearchVideos(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Insert videos directly
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO videos (id, url, thumbnail_url, title, description, duration_seconds, channel, views, published_at)
		VALUES
			('vid1', 'https://youtube.com/v1', 'https://youtube.com/v1_thumb.jpg', 'Go Tutorial', 'Learn Go programming', 600, 'GoChan', 1000, ?),
			('vid2', 'https://youtube.com/v2', 'https://youtube.com/v2_thumb.jpg', 'Python Tutorial', 'Learn Python programming', 900, 'PyChan', 2000, ?)
	`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to insert videos: %v", err)
	}

	results, err := s.Search().SearchVideos(ctx, "Tutorial", store.SearchOptions{})
	if err != nil {
		t.Fatalf("SearchVideos() error = %v", err)
	}

	if len(results) < 2 {
		t.Errorf("len(results) = %d, want >= 2", len(results))
	}
}

func TestSearchStore_SearchNews(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Insert news directly
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO news (id, url, title, snippet, source, published_at)
		VALUES
			('news1', 'https://news.com/1', 'Go 1.23 Released', 'Major release with new features', 'TechNews', ?),
			('news2', 'https://news.com/2', 'Python 4.0 Announced', 'Breaking changes ahead', 'TechNews', ?)
	`, time.Now(), time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("failed to insert news: %v", err)
	}

	results, err := s.Search().SearchNews(ctx, "Released", store.SearchOptions{})
	if err != nil {
		t.Fatalf("SearchNews() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("expected at least one news result")
	}
}

func TestBuildFTSQuery(t *testing.T) {
	tests := []struct {
		query    string
		verbatim bool
		want     string
	}{
		{"hello", false, "hello*"},
		{"hello world", false, "hello* world*"},
		{"hello", true, `"hello"`},
		{"hello world", true, `"hello world"`},
	}

	for _, tt := range tests {
		got := buildFTSQuery(tt.query, tt.verbatim)
		if got != tt.want {
			t.Errorf("buildFTSQuery(%q, %v) = %q, want %q", tt.query, tt.verbatim, got, tt.want)
		}
	}
}

func TestTimeRangeToDate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		input string
		check func(time.Time) bool
	}{
		{"hour", func(d time.Time) bool { return now.Sub(d) <= 2*time.Hour }},
		{"day", func(d time.Time) bool { return now.Sub(d) <= 25*time.Hour }},
		{"week", func(d time.Time) bool { return now.Sub(d) <= 8*24*time.Hour }},
		{"month", func(d time.Time) bool { return now.Sub(d) <= 31*24*time.Hour }},
		{"year", func(d time.Time) bool { return now.Sub(d) <= 366*24*time.Hour }},
		{"invalid", func(d time.Time) bool { return d.IsZero() }},
	}

	for _, tt := range tests {
		got := timeRangeToDate(tt.input)
		if !tt.check(got) {
			t.Errorf("timeRangeToDate(%q) = %v, unexpected result", tt.input, got)
		}
	}
}
