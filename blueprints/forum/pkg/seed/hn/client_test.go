package hn

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/pkg/seed"
)

func TestClient_FetchFeed_Top(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ids, err := client.FetchFeed(ctx, FeedTop)
	if err != nil {
		t.Fatalf("FetchFeed failed: %v", err)
	}

	if len(ids) == 0 {
		t.Fatal("expected at least one story ID")
	}

	t.Logf("Fetched %d top story IDs", len(ids))
	t.Logf("First 5 IDs: %v", ids[:min(5, len(ids))])
}

func TestClient_FetchFeed_New(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ids, err := client.FetchFeed(ctx, FeedNew)
	if err != nil {
		t.Fatalf("FetchFeed failed: %v", err)
	}

	if len(ids) == 0 {
		t.Fatal("expected at least one story ID")
	}

	t.Logf("Fetched %d new story IDs", len(ids))
}

func TestClient_FetchFeed_Best(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ids, err := client.FetchFeed(ctx, FeedBest)
	if err != nil {
		t.Fatalf("FetchFeed failed: %v", err)
	}

	if len(ids) == 0 {
		t.Fatal("expected at least one story ID")
	}

	t.Logf("Fetched %d best story IDs", len(ids))
}

func TestClient_FetchItem(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First get some IDs
	ids, err := client.FetchFeed(ctx, FeedTop)
	if err != nil {
		t.Fatalf("FetchFeed failed: %v", err)
	}
	if len(ids) == 0 {
		t.Fatal("no story IDs to test")
	}

	// Fetch the first item
	item, err := client.FetchItem(ctx, ids[0])
	if err != nil {
		t.Fatalf("FetchItem failed: %v", err)
	}

	if item == nil {
		t.Skip("item was deleted or null")
	}

	if item.ID == 0 {
		t.Error("expected non-zero item ID")
	}
	if item.Title == "" && item.Type == ItemTypeStory {
		t.Error("expected non-empty title for story")
	}
	if item.By == "" && !item.IsDeleted() {
		t.Error("expected non-empty author")
	}

	t.Logf("Item %d: %s by %s (score: %d, comments: %d)",
		item.ID, truncate(item.Title, 50), item.By, item.Score, item.Descendants)
}

func TestClient_FetchItems(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get some IDs
	ids, err := client.FetchFeed(ctx, FeedTop)
	if err != nil {
		t.Fatalf("FetchFeed failed: %v", err)
	}
	if len(ids) < 5 {
		t.Skip("not enough stories to test")
	}

	// Fetch first 5 items
	items, err := client.FetchItems(ctx, ids[:5])
	if err != nil {
		t.Fatalf("FetchItems failed: %v", err)
	}

	if len(items) == 0 {
		t.Fatal("expected at least one item")
	}

	t.Logf("Fetched %d items", len(items))
	for i, item := range items {
		if item != nil {
			t.Logf("  [%d] %s by %s", i, truncate(item.Title, 40), item.By)
		}
	}
}

func TestClient_FetchMaxItem(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	maxID, err := client.FetchMaxItem(ctx)
	if err != nil {
		t.Fatalf("FetchMaxItem failed: %v", err)
	}

	if maxID <= 0 {
		t.Error("expected positive max item ID")
	}

	t.Logf("Max item ID: %d", maxID)
}

func TestClient_FetchUser(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// dang is a well-known HN mod
	user, err := client.FetchUser(ctx, "dang")
	if err != nil {
		t.Fatalf("FetchUser failed: %v", err)
	}

	if user == nil {
		t.Fatal("expected non-nil user")
	}

	if user.ID != "dang" {
		t.Errorf("expected user ID 'dang', got '%s'", user.ID)
	}
	if user.Karma <= 0 {
		t.Error("expected positive karma")
	}

	t.Logf("User: %s, karma: %d", user.ID, user.Karma)
}

func TestClient_FetchThreads(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	threads, err := client.FetchThreads(ctx, "", seed.FetchOpts{Limit: 5})
	if err != nil {
		t.Fatalf("FetchThreads failed: %v", err)
	}

	if len(threads) == 0 {
		t.Fatal("expected at least one thread")
	}

	t.Logf("Fetched %d threads", len(threads))
	for i, thread := range threads {
		if thread.ExternalID == "" {
			t.Errorf("thread %d: expected non-empty external ID", i)
		}
		if thread.Title == "" {
			t.Errorf("thread %d: expected non-empty title", i)
		}
		if thread.Author == "" {
			t.Errorf("thread %d: expected non-empty author", i)
		}
		if thread.CreatedAt.IsZero() {
			t.Errorf("thread %d: expected non-zero created time", i)
		}

		t.Logf("  [%d] %s by %s (score: %d, comments: %d)",
			i, truncate(thread.Title, 50), thread.Author, thread.Score, thread.CommentCount)
	}
}

func TestClient_FetchThreads_Feeds(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	feeds := []struct {
		name   string
		sortBy string
	}{
		{"top", "top"},
		{"new", "new"},
		{"best", "best"},
		{"ask", "ask"},
		{"show", "show"},
	}

	for _, feed := range feeds {
		t.Run(feed.name, func(t *testing.T) {
			threads, err := client.FetchThreads(ctx, "", seed.FetchOpts{
				Limit:  3,
				SortBy: feed.sortBy,
			})
			if err != nil {
				t.Fatalf("FetchThreads failed for %s: %v", feed.name, err)
			}

			t.Logf("%s: fetched %d threads", feed.name, len(threads))
			if len(threads) > 0 {
				t.Logf("  First: %s", truncate(threads[0].Title, 50))
			}
		})
	}
}

func TestClient_FetchComments(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// First, get some threads
	threads, err := client.FetchThreads(ctx, "", seed.FetchOpts{Limit: 10})
	if err != nil {
		t.Fatalf("FetchThreads failed: %v", err)
	}

	// Find a thread with comments
	var threadWithComments *seed.ThreadData
	for _, thread := range threads {
		if thread.CommentCount > 0 {
			threadWithComments = thread
			break
		}
	}

	if threadWithComments == nil {
		t.Skip("no threads with comments found")
	}

	t.Logf("Fetching comments for: %s (expected ~%d comments)",
		truncate(threadWithComments.Title, 50), threadWithComments.CommentCount)

	comments, err := client.FetchComments(ctx, "", threadWithComments.ExternalID)
	if err != nil {
		t.Fatalf("FetchComments failed: %v", err)
	}

	if len(comments) == 0 {
		t.Log("No comments returned (possibly all deleted)")
		return
	}

	t.Logf("Fetched %d top-level comments", len(comments))

	// Count total including nested
	totalComments := countComments(comments)
	t.Logf("Total comments (including nested): %d", totalComments)

	// Verify comment data
	for i, comment := range comments {
		if comment.ExternalID == "" {
			t.Errorf("comment %d: expected non-empty external ID", i)
		}
		if comment.Author == "" {
			t.Errorf("comment %d: expected non-empty author", i)
		}
		if comment.Content == "" {
			t.Errorf("comment %d: expected non-empty content", i)
		}

		if i < 3 {
			t.Logf("  Comment by %s: %s", comment.Author, truncate(comment.Content, 50))
		}
	}
}

func TestClient_ImplementsSource(t *testing.T) {
	var _ seed.Source = (*Client)(nil)
}

func TestClient_htmlToText(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "simple paragraph",
			html:     "<p>Hello world</p>",
			expected: "Hello world",
		},
		{
			name:     "multiple paragraphs",
			html:     "<p>First</p><p>Second</p>",
			expected: "First\n\nSecond",
		},
		{
			name:     "with link",
			html:     `<a href="https://example.com">Click here</a>`,
			expected: "[Click here](https://example.com)",
		},
		{
			name:     "with HTML entities",
			html:     "Hello &amp; goodbye",
			expected: "Hello & goodbye",
		},
		{
			name:     "with br tags",
			html:     "Line 1<br>Line 2<br/>Line 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "empty string",
			html:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.htmlToText(tt.html)
			if result != tt.expected {
				t.Errorf("htmlToText(%q) = %q, want %q", tt.html, result, tt.expected)
			}
		})
	}
}

func TestClient_extractDomain(t *testing.T) {
	client := NewClient()

	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/path", "example.com"},
		{"http://www.example.com/path", "example.com"},
		{"https://sub.example.com/path?query=1", "sub.example.com"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := client.extractDomain(tt.url)
			if result != tt.expected {
				t.Errorf("extractDomain(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func countComments(comments []*seed.CommentData) int {
	total := len(comments)
	for _, c := range comments {
		total += countComments(c.Replies)
	}
	return total
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
