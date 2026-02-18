package qq

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	if db.Path() != dbPath {
		t.Errorf("Path() = %q, want %q", db.Path(), dbPath)
	}
}

func TestInsertAndQueryArticles(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(filepath.Join(dir, "test.duckdb"))
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	now := time.Now().Truncate(time.Microsecond)
	articles := []Article{
		{
			ArticleID:   "20260217A001",
			Title:       "Test Article 1",
			Content:     "<p>Content 1</p>",
			Abstract:    "Abstract 1",
			PublishTime: now,
			Channel:     "tech",
			Source:      "Source 1",
			SourceID:    "s1",
			ArticleType: 0,
			URL:         "https://news.qq.com/rain/a/20260217A001",
			CrawledAt:   now,
			StatusCode:  200,
		},
		{
			ArticleID:   "20260217A002",
			Title:       "Test Article 2",
			Content:     "<p>Content 2</p>",
			Abstract:    "Abstract 2",
			PublishTime: now,
			Channel:     "finance",
			Source:      "Source 2",
			SourceID:    "s2",
			ArticleType: 0,
			URL:         "https://news.qq.com/rain/a/20260217A002",
			CrawledAt:   now,
			StatusCode:  200,
		},
	}

	if err := db.InsertArticles(articles); err != nil {
		t.Fatalf("InsertArticles failed: %v", err)
	}

	// Query stats
	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.Articles != 2 {
		t.Errorf("Articles = %d, want 2", stats.Articles)
	}
	if stats.WithContent != 2 {
		t.Errorf("WithContent = %d, want 2", stats.WithContent)
	}

	// Query crawled IDs
	ids, err := db.CrawledArticleIDs()
	if err != nil {
		t.Fatalf("CrawledArticleIDs failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("CrawledArticleIDs count = %d, want 2", len(ids))
	}

	// Top articles
	top, err := db.TopArticles(5)
	if err != nil {
		t.Fatalf("TopArticles failed: %v", err)
	}
	if len(top) != 2 {
		t.Errorf("TopArticles count = %d, want 2", len(top))
	}
}

func TestInsertArticlesUpsert(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(filepath.Join(dir, "test.duckdb"))
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	now := time.Now().Truncate(time.Microsecond)

	// Insert initial
	err = db.InsertArticles([]Article{{
		ArticleID: "20260217A001",
		Title:     "Original Title",
		CrawledAt: now,
	}})
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	// Upsert with updated title
	err = db.InsertArticles([]Article{{
		ArticleID: "20260217A001",
		Title:     "Updated Title",
		Content:   "<p>Now with content</p>",
		CrawledAt: now,
	}})
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	stats, _ := db.GetStats()
	if stats.Articles != 1 {
		t.Errorf("Articles = %d, want 1 (upsert should not duplicate)", stats.Articles)
	}
}

func TestMarkAndFetchSitemaps(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(filepath.Join(dir, "test.duckdb"))
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	// Mark some sitemaps
	db.MarkSitemap("https://news.qq.com/sitemap/sitemap_1.xml", 50)
	db.MarkSitemap("https://news.qq.com/sitemap/sitemap_2.xml", 30)

	// Fetch them back
	fetched, err := db.FetchedSitemaps()
	if err != nil {
		t.Fatalf("FetchedSitemaps failed: %v", err)
	}
	if len(fetched) != 2 {
		t.Errorf("FetchedSitemaps count = %d, want 2", len(fetched))
	}
	if !fetched["https://news.qq.com/sitemap/sitemap_1.xml"] {
		t.Error("sitemap_1.xml not found in fetched set")
	}

	// Stats should show sitemaps
	stats, _ := db.GetStats()
	if stats.Sitemaps != 2 {
		t.Errorf("Sitemaps = %d, want 2", stats.Sitemaps)
	}
}

func TestArticlesWithErrors(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(filepath.Join(dir, "test.duckdb"))
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	now := time.Now().Truncate(time.Microsecond)
	articles := []Article{
		{ArticleID: "ok1", Title: "Good", Content: "content", CrawledAt: now, StatusCode: 200},
		{ArticleID: "err1", CrawledAt: now, StatusCode: 404, Error: "not found"},
	}

	db.InsertArticles(articles)

	stats, _ := db.GetStats()
	if stats.WithError != 1 {
		t.Errorf("WithError = %d, want 1", stats.WithError)
	}

	// CrawledArticleIDs should only return successful ones
	ids, _ := db.CrawledArticleIDs()
	if len(ids) != 1 {
		t.Errorf("CrawledArticleIDs = %d, want 1 (errors excluded)", len(ids))
	}
}

func TestDBPathCreation(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "deep", "nested", "path")

	db, err := OpenDB(filepath.Join(nested, "test.duckdb"))
	if err != nil {
		t.Fatalf("OpenDB with nested dir failed: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(nested); os.IsNotExist(err) {
		t.Error("nested directory was not created")
	}
}
