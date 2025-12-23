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
