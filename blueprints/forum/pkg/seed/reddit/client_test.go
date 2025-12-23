package reddit

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/pkg/seed"
)

func TestClient_FetchSubreddit_Golang(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sub, err := client.FetchSubreddit(ctx, "golang")
	if err != nil {
		t.Fatalf("FetchSubreddit failed: %v", err)
	}

	if sub.Name == "" {
		t.Error("expected non-empty subreddit name")
	}
	if sub.Name != "golang" {
		t.Errorf("expected subreddit name 'golang', got '%s'", sub.Name)
	}
	if sub.Subscribers <= 0 {
		t.Error("expected positive subscriber count")
	}

	t.Logf("Subreddit: %s, Subscribers: %d", sub.Name, sub.Subscribers)
}

func TestClient_FetchSubreddit_Programming(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wait to avoid rate limiting from previous test
	time.Sleep(2 * time.Second)

	sub, err := client.FetchSubreddit(ctx, "programming")
	if err != nil {
		t.Fatalf("FetchSubreddit failed: %v", err)
	}

	if sub.Name != "programming" {
		t.Errorf("expected subreddit name 'programming', got '%s'", sub.Name)
	}
	if sub.Subscribers <= 0 {
		t.Error("expected positive subscriber count")
	}

	t.Logf("Subreddit: %s, Subscribers: %d", sub.Name, sub.Subscribers)
}

func TestClient_FetchThreads_Golang(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	threads, err := client.FetchThreads(ctx, "golang", seed.FetchOpts{Limit: 5})
	if err != nil {
		t.Fatalf("FetchThreads failed: %v", err)
	}

	if len(threads) == 0 {
		t.Fatal("expected at least one thread")
	}

	t.Logf("Fetched %d threads from r/golang", len(threads))

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
			i+1, thread.Title[:min(50, len(thread.Title))], thread.Author, thread.Score, thread.CommentCount)
	}
}

func TestClient_FetchThreads_Programming(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wait to avoid rate limiting
	time.Sleep(2 * time.Second)

	threads, err := client.FetchThreads(ctx, "programming", seed.FetchOpts{Limit: 5})
	if err != nil {
		t.Fatalf("FetchThreads failed: %v", err)
	}

	if len(threads) == 0 {
		t.Fatal("expected at least one thread")
	}

	t.Logf("Fetched %d threads from r/programming", len(threads))

	// Verify thread data
	for _, thread := range threads {
		if thread.ExternalID == "" {
			t.Error("expected non-empty external ID")
		}
		if thread.Title == "" {
			t.Error("expected non-empty title")
		}
	}
}

func TestClient_FetchComments(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First, get a thread with comments
	threads, err := client.FetchThreads(ctx, "golang", seed.FetchOpts{Limit: 10})
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

	t.Logf("Fetching comments for thread: %s (expected ~%d comments)",
		threadWithComments.Title[:min(50, len(threadWithComments.Title))],
		threadWithComments.CommentCount)

	// Wait for rate limit
	time.Sleep(2 * time.Second)

	comments, err := client.FetchComments(ctx, "golang", threadWithComments.ExternalID)
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
			t.Logf("  Comment by %s: %s...", comment.Author, truncate(comment.Content, 50))
		}
	}
}

func TestClient_ImplementsSource(t *testing.T) {
	var _ seed.Source = (*Client)(nil)
}

func TestClient_ListSubreddits(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := client.ListSubreddits(ctx, seed.ListSubredditsOpts{
		Where: "popular",
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("ListSubreddits failed: %v", err)
	}

	if len(result.Subreddits) == 0 {
		t.Fatal("expected at least one subreddit")
	}

	t.Logf("Fetched %d subreddits (HasMore: %v, After: %s)",
		len(result.Subreddits), result.HasMore, result.After)

	for i, sub := range result.Subreddits {
		if sub.Name == "" {
			t.Errorf("subreddit %d: expected non-empty name", i)
		}
		t.Logf("  [%d] r/%s - %d subscribers", i+1, sub.Name, sub.Subscribers)
	}
}

func TestClient_ListSubreddits_Pagination(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First page
	result1, err := client.ListSubreddits(ctx, seed.ListSubredditsOpts{
		Where: "popular",
		Limit: 3,
	})
	if err != nil {
		t.Fatalf("First ListSubreddits failed: %v", err)
	}

	if !result1.HasMore {
		t.Skip("no more pages available")
	}

	t.Logf("First page: %d subreddits, After: %s", len(result1.Subreddits), result1.After)

	// Wait for rate limit
	time.Sleep(2 * time.Second)

	// Second page
	result2, err := client.ListSubreddits(ctx, seed.ListSubredditsOpts{
		Where: "popular",
		Limit: 3,
		After: result1.After,
	})
	if err != nil {
		t.Fatalf("Second ListSubreddits failed: %v", err)
	}

	t.Logf("Second page: %d subreddits", len(result2.Subreddits))

	// Verify different subreddits
	page1Names := make(map[string]bool)
	for _, sub := range result1.Subreddits {
		page1Names[sub.Name] = true
	}

	overlap := 0
	for _, sub := range result2.Subreddits {
		if page1Names[sub.Name] {
			overlap++
		}
	}

	if overlap > 0 {
		t.Logf("Note: %d subreddits overlap between pages", overlap)
	}
}

func TestClient_FetchThreadsWithCursor(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First page
	result1, err := client.FetchThreadsWithCursor(ctx, "golang", seed.FetchOpts{
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("First FetchThreadsWithCursor failed: %v", err)
	}

	if len(result1.Threads) == 0 {
		t.Fatal("expected at least one thread")
	}

	t.Logf("First page: %d threads (HasMore: %v)", len(result1.Threads), result1.HasMore)

	if !result1.HasMore {
		t.Skip("no more pages available")
	}

	// Wait for rate limit
	time.Sleep(2 * time.Second)

	// Second page
	result2, err := client.FetchThreadsWithCursor(ctx, "golang", seed.FetchOpts{
		Limit: 5,
		After: result1.After,
	})
	if err != nil {
		t.Fatalf("Second FetchThreadsWithCursor failed: %v", err)
	}

	t.Logf("Second page: %d threads", len(result2.Threads))

	// Verify different threads
	page1IDs := make(map[string]bool)
	for _, thread := range result1.Threads {
		page1IDs[thread.ExternalID] = true
	}

	for _, thread := range result2.Threads {
		if page1IDs[thread.ExternalID] {
			t.Errorf("thread %s appears on both pages", thread.ExternalID)
		}
	}
}

func TestClient_FetchThreads_SortOptions(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sortOptions := []struct {
		name      string
		sortBy    string
		timeRange string
	}{
		{"hot", "hot", ""},
		{"new", "new", ""},
		{"top_day", "top", "day"},
		{"rising", "rising", ""},
	}

	for _, opt := range sortOptions {
		t.Run(opt.name, func(t *testing.T) {
			time.Sleep(2 * time.Second) // Rate limit

			threads, err := client.FetchThreads(ctx, "programming", seed.FetchOpts{
				Limit:     3,
				SortBy:    opt.sortBy,
				TimeRange: opt.timeRange,
			})
			if err != nil {
				t.Fatalf("FetchThreads with sort=%s failed: %v", opt.sortBy, err)
			}

			t.Logf("Sort %s: fetched %d threads", opt.name, len(threads))
			if len(threads) > 0 {
				t.Logf("  First: %s (score: %d)", truncate(threads[0].Title, 40), threads[0].Score)
			}
		})
	}
}

func TestClient_FetchCommentsWithOpts(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get a thread with comments
	threads, err := client.FetchThreads(ctx, "programming", seed.FetchOpts{Limit: 10})
	if err != nil {
		t.Fatalf("FetchThreads failed: %v", err)
	}

	var threadWithComments *seed.ThreadData
	for _, thread := range threads {
		if thread.CommentCount > 5 {
			threadWithComments = thread
			break
		}
	}

	if threadWithComments == nil {
		t.Skip("no threads with enough comments found")
	}

	time.Sleep(2 * time.Second)

	sortOptions := []string{"best", "top", "new", "controversial"}
	for _, sort := range sortOptions {
		t.Run(sort, func(t *testing.T) {
			time.Sleep(2 * time.Second)

			comments, err := client.FetchCommentsWithOpts(ctx, "programming", threadWithComments.ExternalID, seed.CommentOpts{
				Limit: 10,
				Depth: 3,
				Sort:  sort,
			})
			if err != nil {
				t.Fatalf("FetchCommentsWithOpts sort=%s failed: %v", sort, err)
			}

			t.Logf("Sort %s: %d comments", sort, len(comments))
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
