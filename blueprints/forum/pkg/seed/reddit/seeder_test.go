package reddit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/seed"
	"github.com/go-mizu/mizu/blueprints/forum/store/duckdb"
)

func TestSeeder_SeedFromReddit_Golang(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "forum-seed-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Open database
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

	// Create seeder
	seeder := seed.NewSeeder(
		accountsSvc,
		boardsSvc,
		threadsSvc,
		commentsSvc,
		store.SeedMappings(),
	)

	// Create Reddit client
	client := NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Seed from r/golang
	result, err := seeder.SeedFromSource(ctx, client, seed.SeedOpts{
		Subreddits:   []string{"golang"},
		ThreadLimit:  3,
		WithComments: false,
		OnProgress: func(msg string) {
			t.Logf("Progress: %s", msg)
		},
	})
	if err != nil {
		t.Fatalf("SeedFromSource failed: %v", err)
	}

	t.Logf("Seed result:")
	t.Logf("  Boards created: %d, skipped: %d", result.BoardsCreated, result.BoardsSkipped)
	t.Logf("  Threads created: %d, skipped: %d", result.ThreadsCreated, result.ThreadsSkipped)
	t.Logf("  Users created: %d, skipped: %d", result.UsersCreated, result.UsersSkipped)

	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			t.Logf("  Error: %v", err)
		}
	}

	// Verify board was created
	board, err := boardsSvc.GetByName(ctx, "golang")
	if err != nil {
		t.Fatalf("failed to get board: %v", err)
	}
	if board == nil {
		t.Fatal("expected golang board to exist")
	}
	t.Logf("Board created: %s (%s)", board.Name, board.ID)

	// Verify threads were created
	threadList, err := threadsSvc.ListByBoard(ctx, board.ID, threads.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("failed to list threads: %v", err)
	}
	t.Logf("Threads in board: %d", len(threadList))

	if result.ThreadsCreated == 0 && result.ThreadsSkipped == 0 {
		t.Error("expected at least some threads to be processed")
	}
}

func TestSeeder_SeedFromReddit_Programming(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "forum-seed-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Open database
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

	// Create seeder
	seeder := seed.NewSeeder(
		accountsSvc,
		boardsSvc,
		threadsSvc,
		commentsSvc,
		store.SeedMappings(),
	)

	// Create Reddit client
	client := NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Seed from r/programming
	result, err := seeder.SeedFromSource(ctx, client, seed.SeedOpts{
		Subreddits:   []string{"programming"},
		ThreadLimit:  3,
		WithComments: false,
		OnProgress: func(msg string) {
			t.Logf("Progress: %s", msg)
		},
	})
	if err != nil {
		t.Fatalf("SeedFromSource failed: %v", err)
	}

	t.Logf("Seed result:")
	t.Logf("  Boards created: %d, skipped: %d", result.BoardsCreated, result.BoardsSkipped)
	t.Logf("  Threads created: %d, skipped: %d", result.ThreadsCreated, result.ThreadsSkipped)
	t.Logf("  Users created: %d, skipped: %d", result.UsersCreated, result.UsersSkipped)

	// Verify board was created
	board, err := boardsSvc.GetByName(ctx, "programming")
	if err != nil {
		t.Fatalf("failed to get board: %v", err)
	}
	if board == nil {
		t.Fatal("expected programming board to exist")
	}
	t.Logf("Board created: %s (%s)", board.Name, board.ID)
}

func TestSeeder_Idempotent(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "forum-seed-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Open database
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

	// Create seeder
	seeder := seed.NewSeeder(
		accountsSvc,
		boardsSvc,
		threadsSvc,
		commentsSvc,
		store.SeedMappings(),
	)

	// Create Reddit client
	client := NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	opts := seed.SeedOpts{
		Subreddits:   []string{"golang"},
		ThreadLimit:  2,
		WithComments: false,
	}

	// First seed
	result1, err := seeder.SeedFromSource(ctx, client, opts)
	if err != nil {
		t.Fatalf("First seed failed: %v", err)
	}
	t.Logf("First seed: boards=%d, threads=%d", result1.BoardsCreated, result1.ThreadsCreated)

	// Clear cache and seed again
	seeder.ClearCache()

	// Wait for rate limit
	time.Sleep(3 * time.Second)

	// Second seed (should be idempotent)
	result2, err := seeder.SeedFromSource(ctx, client, opts)
	if err != nil {
		t.Fatalf("Second seed failed: %v", err)
	}
	t.Logf("Second seed: boards=%d (skipped=%d), threads=%d (skipped=%d)",
		result2.BoardsCreated, result2.BoardsSkipped,
		result2.ThreadsCreated, result2.ThreadsSkipped)

	// Verify idempotency: second seed should have everything skipped
	if result2.BoardsCreated > 0 {
		t.Errorf("expected 0 boards created on second seed, got %d", result2.BoardsCreated)
	}
	if result2.BoardsSkipped == 0 {
		t.Error("expected boards to be skipped on second seed")
	}

	// Threads should be skipped (already seeded)
	if result2.ThreadsCreated > 0 {
		t.Errorf("expected 0 threads created on second seed, got %d", result2.ThreadsCreated)
	}
	if result2.ThreadsSkipped == 0 {
		t.Error("expected threads to be skipped on second seed")
	}

	// Verify we still have the same number of threads
	board, _ := boardsSvc.GetByName(ctx, "golang")
	threadList, _ := threadsSvc.ListByBoard(ctx, board.ID, threads.ListOpts{Limit: 10})
	expectedThreads := result1.ThreadsCreated
	if len(threadList) != expectedThreads {
		t.Errorf("expected %d threads after idempotent seed, got %d", expectedThreads, len(threadList))
	}

	t.Logf("Idempotency verified: %d threads remain after second seed", len(threadList))
}

func TestSeeder_WithComments(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "forum-seed-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Open database
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

	// Create seeder
	seeder := seed.NewSeeder(
		accountsSvc,
		boardsSvc,
		threadsSvc,
		commentsSvc,
		store.SeedMappings(),
	)

	// Create Reddit client
	client := NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Seed with comments
	result, err := seeder.SeedFromSource(ctx, client, seed.SeedOpts{
		Subreddits:   []string{"golang"},
		ThreadLimit:  1,
		WithComments: true,
		CommentDepth: 3,
		OnProgress: func(msg string) {
			t.Logf("Progress: %s", msg)
		},
	})
	if err != nil {
		t.Fatalf("SeedFromSource failed: %v", err)
	}

	t.Logf("Seed result:")
	t.Logf("  Boards created: %d", result.BoardsCreated)
	t.Logf("  Threads created: %d", result.ThreadsCreated)
	t.Logf("  Comments created: %d, skipped: %d", result.CommentsCreated, result.CommentsSkipped)

	// We may not get comments if the thread has none or they're all deleted
	if result.ThreadsCreated > 0 {
		t.Logf("Successfully seeded thread with %d comments", result.CommentsCreated)
	}
}

func TestSeeder_MultipleSubreddits(t *testing.T) {
	// Create temp directory for test database
	tmpDir := filepath.Join(os.TempDir(), "forum-seed-multi-test")
	os.RemoveAll(tmpDir)
	defer os.RemoveAll(tmpDir)

	// Open database
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

	// Create seeder
	seeder := seed.NewSeeder(
		accountsSvc,
		boardsSvc,
		threadsSvc,
		commentsSvc,
		store.SeedMappings(),
	)

	// Create Reddit client
	client := NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Seed from both subreddits
	result, err := seeder.SeedFromSource(ctx, client, seed.SeedOpts{
		Subreddits:   []string{"golang", "programming"},
		ThreadLimit:  2,
		WithComments: false,
		OnProgress: func(msg string) {
			t.Logf("Progress: %s", msg)
		},
	})
	if err != nil {
		t.Fatalf("SeedFromSource failed: %v", err)
	}

	t.Logf("Seed result:")
	t.Logf("  Boards created: %d", result.BoardsCreated)
	t.Logf("  Threads created: %d", result.ThreadsCreated)
	t.Logf("  Users created: %d", result.UsersCreated)

	// Verify both boards were created
	golangBoard, err := boardsSvc.GetByName(ctx, "golang")
	if err != nil || golangBoard == nil {
		t.Error("expected golang board to exist")
	}

	programmingBoard, err := boardsSvc.GetByName(ctx, "programming")
	if err != nil || programmingBoard == nil {
		t.Error("expected programming board to exist")
	}

	if result.BoardsCreated < 2 {
		t.Errorf("expected at least 2 boards created, got %d", result.BoardsCreated)
	}
}
