package hn

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/seed"
	"github.com/go-mizu/mizu/blueprints/forum/store/duckdb"
)

func TestSeeder_SeedFromHN(t *testing.T) {
	// Create temp database
	tmpDir, err := os.MkdirTemp("", "forum-hn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := duckdb.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// Create services
	accountsSvc := accounts.NewService(store.Accounts())
	boardsSvc := boards.NewService(store.Boards())
	threadsSvc := threads.NewService(store.Threads(), accountsSvc, boardsSvc)
	commentsSvc := comments.NewService(store.Comments(), accountsSvc, threadsSvc)

	// Create seeder and client
	seeder := seed.NewSeeder(accountsSvc, boardsSvc, threadsSvc, commentsSvc, store.SeedMappings())
	client := NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Seed from HN
	result, err := seeder.SeedFromSource(ctx, client, seed.SeedOpts{
		Subreddits:  []string{"hackernews"},
		ThreadLimit: 5,
		SortBy:      "top",
		OnProgress: func(msg string) {
			t.Logf("Progress: %s", msg)
		},
	})

	if err != nil {
		t.Fatalf("SeedFromSource failed: %v", err)
	}

	// Verify results
	if result.BoardsCreated == 0 && result.BoardsSkipped == 0 {
		t.Error("expected at least one board created or skipped")
	}
	if result.ThreadsCreated == 0 {
		t.Error("expected at least one thread created")
	}

	t.Logf("Results: boards created=%d skipped=%d, threads created=%d skipped=%d, users=%d, errors=%d",
		result.BoardsCreated, result.BoardsSkipped,
		result.ThreadsCreated, result.ThreadsSkipped,
		result.UsersCreated, len(result.Errors))

	// Log any errors
	for _, err := range result.Errors {
		t.Logf("Error: %v", err)
	}
}

func TestSeeder_SeedFromHN_WithComments(t *testing.T) {
	// Create temp database
	tmpDir, err := os.MkdirTemp("", "forum-hn-comments-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := duckdb.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// Create services
	accountsSvc := accounts.NewService(store.Accounts())
	boardsSvc := boards.NewService(store.Boards())
	threadsSvc := threads.NewService(store.Threads(), accountsSvc, boardsSvc)
	commentsSvc := comments.NewService(store.Comments(), accountsSvc, threadsSvc)

	// Create seeder and client
	seeder := seed.NewSeeder(accountsSvc, boardsSvc, threadsSvc, commentsSvc, store.SeedMappings())
	client := NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Seed from HN with comments
	result, err := seeder.SeedFromSource(ctx, client, seed.SeedOpts{
		Subreddits:   []string{"hackernews"},
		ThreadLimit:  3,
		WithComments: true,
		CommentDepth: 3,
		SortBy:       "top",
		OnProgress: func(msg string) {
			t.Logf("Progress: %s", msg)
		},
	})

	if err != nil {
		t.Fatalf("SeedFromSource failed: %v", err)
	}

	t.Logf("Results: boards=%d, threads=%d, comments=%d, users=%d, errors=%d",
		result.BoardsCreated+result.BoardsSkipped,
		result.ThreadsCreated+result.ThreadsSkipped,
		result.CommentsCreated+result.CommentsSkipped,
		result.UsersCreated,
		len(result.Errors))

	// Comments might be 0 if the top stories have no comments
	if result.ThreadsCreated == 0 {
		t.Error("expected at least one thread created")
	}
}

func TestSeeder_Idempotent_HN(t *testing.T) {
	// Create temp database
	tmpDir, err := os.MkdirTemp("", "forum-hn-idempotent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := duckdb.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// Create services
	accountsSvc := accounts.NewService(store.Accounts())
	boardsSvc := boards.NewService(store.Boards())
	threadsSvc := threads.NewService(store.Threads(), accountsSvc, boardsSvc)
	commentsSvc := comments.NewService(store.Comments(), accountsSvc, threadsSvc)

	// Create seeder and client
	seeder := seed.NewSeeder(accountsSvc, boardsSvc, threadsSvc, commentsSvc, store.SeedMappings())
	client := NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	opts := seed.SeedOpts{
		Subreddits:  []string{"hackernews"},
		ThreadLimit: 3,
		SortBy:      "best", // Use best for more stable results
		OnProgress: func(msg string) {
			t.Logf("Progress: %s", msg)
		},
	}

	// First seed
	result1, err := seeder.SeedFromSource(ctx, client, opts)
	if err != nil {
		t.Fatalf("First SeedFromSource failed: %v", err)
	}

	t.Logf("First run: boards=%d, threads=%d",
		result1.BoardsCreated, result1.ThreadsCreated)

	// Clear cache to simulate fresh run
	seeder.ClearCache()

	// Second seed - should skip existing items
	result2, err := seeder.SeedFromSource(ctx, client, opts)
	if err != nil {
		t.Fatalf("Second SeedFromSource failed: %v", err)
	}

	t.Logf("Second run: boards created=%d skipped=%d, threads created=%d skipped=%d",
		result2.BoardsCreated, result2.BoardsSkipped,
		result2.ThreadsCreated, result2.ThreadsSkipped)

	// Verify idempotency - no new items should be created
	if result2.BoardsCreated != 0 {
		t.Errorf("expected 0 boards created on second run, got %d", result2.BoardsCreated)
	}
	// Note: ThreadsCreated might be > 0 if new stories appeared between runs,
	// but ThreadsSkipped should be > 0
	if result2.ThreadsSkipped == 0 && result1.ThreadsCreated > 0 {
		t.Error("expected some threads to be skipped on second run")
	}
}

func TestSeeder_DifferentFeeds(t *testing.T) {
	// Create temp database
	tmpDir, err := os.MkdirTemp("", "forum-hn-feeds-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := duckdb.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	// Create services
	accountsSvc := accounts.NewService(store.Accounts())
	boardsSvc := boards.NewService(store.Boards())
	threadsSvc := threads.NewService(store.Threads(), accountsSvc, boardsSvc)
	commentsSvc := comments.NewService(store.Comments(), accountsSvc, threadsSvc)

	// Create seeder and client
	seeder := seed.NewSeeder(accountsSvc, boardsSvc, threadsSvc, commentsSvc, store.SeedMappings())
	client := NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	feeds := []string{"top", "new", "best"}
	for _, feed := range feeds {
		seeder.ClearCache()

		result, err := seeder.SeedFromSource(ctx, client, seed.SeedOpts{
			Subreddits:  []string{"hackernews"},
			ThreadLimit: 2,
			SortBy:      feed,
			OnProgress: func(msg string) {
				t.Logf("[%s] %s", feed, msg)
			},
		})

		if err != nil {
			t.Errorf("Feed %s failed: %v", feed, err)
			continue
		}

		t.Logf("Feed %s: threads=%d, errors=%d",
			feed, result.ThreadsCreated+result.ThreadsSkipped, len(result.Errors))
	}
}
