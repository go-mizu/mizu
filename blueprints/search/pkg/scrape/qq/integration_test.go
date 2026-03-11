package qq

import (
	"context"
	"testing"
	"time"
)

// TestSitemapDiscovery verifies we can fetch the real sitemap index.
// Run with: go test -run TestSitemapDiscovery -v -count=1
func TestSitemapDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := DefaultConfig()
	client := NewClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	idx, err := client.FetchSitemapIndex(ctx)
	if err != nil {
		t.Fatalf("FetchSitemapIndex: %v", err)
	}

	t.Logf("Sitemap index contains %d sitemaps", len(idx.Sitemaps))
	if len(idx.Sitemaps) == 0 {
		t.Fatal("expected at least 1 sitemap in index")
	}

	// Fetch first sitemap to verify parsing works
	first := idx.Sitemaps[0]
	t.Logf("Fetching first sitemap: %s", first.Loc)

	urlSet, err := client.FetchSitemap(ctx, first.Loc)
	if err != nil {
		t.Fatalf("FetchSitemap: %v", err)
	}

	t.Logf("First sitemap contains %d URLs", len(urlSet.URLs))
	if len(urlSet.URLs) == 0 {
		t.Fatal("expected at least 1 URL in sitemap")
	}

	// Extract article IDs
	var ids []string
	for _, u := range urlSet.URLs {
		if id := ExtractArticleID(u.Loc); id != "" {
			ids = append(ids, id)
		}
	}
	t.Logf("Extracted %d article IDs from first sitemap", len(ids))

	if len(ids) > 0 {
		t.Logf("Sample article IDs: %v", ids[:min(5, len(ids))])
	}
}

// TestFetchArticle verifies we can fetch and parse a real article.
// Run with: go test -run TestFetchArticle -v -count=1
func TestFetchArticle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := DefaultConfig()
	client := NewClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First discover an article ID from sitemap
	idx, err := client.FetchSitemapIndex(ctx)
	if err != nil {
		t.Fatalf("FetchSitemapIndex: %v", err)
	}

	if len(idx.Sitemaps) == 0 {
		t.Fatal("no sitemaps found")
	}

	// Get the last sitemap (most recent articles)
	lastSitemap := idx.Sitemaps[len(idx.Sitemaps)-1]
	urlSet, err := client.FetchSitemap(ctx, lastSitemap.Loc)
	if err != nil {
		t.Fatalf("FetchSitemap: %v", err)
	}

	if len(urlSet.URLs) == 0 {
		t.Fatal("no URLs in sitemap")
	}

	// Try articles until we find a live one (many are deleted)
	var article *Article
	for i, u := range urlSet.URLs {
		if i >= 20 { // try at most 20
			break
		}
		articleID := ExtractArticleID(u.Loc)
		if articleID == "" {
			continue
		}

		t.Logf("Trying article %d: %s", i, articleID)
		html, statusCode, err := client.FetchArticlePage(ctx, articleID)
		if err != nil {
			t.Logf("  Skipped (err: %v)", err)
			continue
		}
		if statusCode != 200 {
			t.Logf("  Skipped (status %d)", statusCode)
			continue
		}

		a, err := ParseArticlePage(html, articleID)
		if err != nil {
			t.Logf("  Skipped (parse err: %v)", err)
			continue
		}

		article = a
		t.Logf("  Found live article: %s", a.Title)
		break
	}

	if article == nil {
		t.Skip("no live articles found in first 20 URLs of last sitemap")
	}

	t.Logf("Article: ID=%s Title=%s Channel=%s Source=%s PubTime=%s HasBody=%v",
		article.ArticleID, article.Title, article.Channel, article.Source,
		article.PublishTime.Format("2006-01-02 15:04"), article.Content != "")

	if article.Title == "" {
		t.Error("expected non-empty title")
	}
}

// TestHotRanking verifies the hot ranking API works.
// Run with: go test -run TestHotRanking -v -count=1
func TestHotRanking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := DefaultConfig()
	client := NewClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	items, err := client.FetchHotRanking(ctx)
	if err != nil {
		t.Fatalf("FetchHotRanking: %v", err)
	}

	t.Logf("Hot ranking contains %d items", len(items))
	for i, item := range items {
		if i >= 5 {
			break
		}
		t.Logf("  #%d: [%s] %s — %s", i+1, item.ChlName, item.Title, item.Source)
	}

	if len(items) == 0 {
		t.Error("expected at least 1 hot ranking item")
	}
}

// TestChannelFeed verifies the channel feed API works.
// Run with: go test -run TestChannelFeed -v -count=1
func TestChannelFeed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := DefaultConfig()
	client := NewClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	items, err := client.FetchChannelFeed(ctx, "news_news_tech")
	if err != nil {
		t.Fatalf("FetchChannelFeed: %v", err)
	}

	t.Logf("Tech channel contains %d items", len(items))
	for i, item := range items {
		if i >= 5 {
			break
		}
		t.Logf("  [%s] %s", item.ID, item.Title)
	}
}
