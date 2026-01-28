package sqlite

import (
	"context"
	"testing"
	"time"
)

func TestLLMCacheStore_SetAndGet(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	cache := store.LLMCache()

	// Create a cache entry
	entry := &LLMCacheEntry{
		QueryHash:        "abc123",
		Query:            "What is Go?",
		Mode:             "quick",
		Model:            "claude-haiku-4.5",
		ResponseText:     "Go is a programming language.",
		Citations:        "[]",
		FollowUps:        "[]",
		RelatedQuestions: "[]",
		InputTokens:      100,
		OutputTokens:     50,
	}

	// Set the entry
	if err := cache.Set(ctx, entry); err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Get the entry
	retrieved, err := cache.Get(ctx, "abc123", "quick", "claude-haiku-4.5")
	if err != nil {
		t.Fatalf("Failed to get cache entry: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected cache entry, got nil")
	}
	if retrieved.ResponseText != entry.ResponseText {
		t.Errorf("Expected response %q, got %q", entry.ResponseText, retrieved.ResponseText)
	}
	if retrieved.InputTokens != entry.InputTokens {
		t.Errorf("Expected input tokens %d, got %d", entry.InputTokens, retrieved.InputTokens)
	}
}

func TestLLMCacheStore_GetNonExistent(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	cache := store.LLMCache()

	// Get non-existent entry
	entry, err := cache.Get(ctx, "nonexistent", "quick", "model")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if entry != nil {
		t.Error("Expected nil for non-existent entry")
	}
}

func TestLLMCacheStore_Delete(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	cache := store.LLMCache()

	// Create and set entry
	entry := &LLMCacheEntry{
		QueryHash:    "delete-test",
		Query:        "test",
		Mode:         "quick",
		Model:        "model",
		ResponseText: "response",
	}
	if err := cache.Set(ctx, entry); err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Delete
	if err := cache.Delete(ctx, "delete-test", "quick", "model"); err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify deleted
	retrieved, _ := cache.Get(ctx, "delete-test", "quick", "model")
	if retrieved != nil {
		t.Error("Entry should have been deleted")
	}
}

func TestLLMCacheStore_GetStats(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	cache := store.LLMCache()

	// Add some entries
	for i := 0; i < 3; i++ {
		entry := &LLMCacheEntry{
			QueryHash:    "hash-" + string(rune('a'+i)),
			Query:        "query-" + string(rune('a'+i)),
			Mode:         "quick",
			Model:        "model",
			ResponseText: "response",
			InputTokens:  100,
			OutputTokens: 50,
		}
		cache.Set(ctx, entry)
	}

	stats, err := cache.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	total := stats["total_entries"].(int64)
	if total != 3 {
		t.Errorf("Expected 3 entries, got %d", total)
	}
}

func TestLLMLogStore_Log(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	log := store.LLMLog()

	// Log an entry
	entry := &LLMLogEntry{
		RequestID:    "req-123",
		Provider:     "claude",
		Model:        "claude-haiku-4.5",
		Mode:         "quick",
		Query:        "What is Go?",
		RequestJSON:  "{}",
		ResponseJSON: "{}",
		Status:       "success",
		InputTokens:  100,
		OutputTokens: 50,
		DurationMs:   500,
		CostUSD:      0.001,
	}

	if err := log.Log(ctx, entry); err != nil {
		t.Fatalf("Failed to log entry: %v", err)
	}

	// Verify entry was logged
	retrieved, err := log.GetByRequestID(ctx, "req-123")
	if err != nil {
		t.Fatalf("Failed to get log entry: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected log entry, got nil")
	}
	if retrieved.Status != "success" {
		t.Errorf("Expected status 'success', got %q", retrieved.Status)
	}
}

func TestLLMLogStore_List(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	log := store.LLMLog()

	// Log multiple entries
	for i := 0; i < 5; i++ {
		entry := &LLMLogEntry{
			RequestID:   "req-" + string(rune('a'+i)),
			Provider:    "claude",
			Model:       "claude-haiku-4.5",
			Status:      "success",
			RequestJSON: "{}",
		}
		log.Log(ctx, entry)
	}

	// List entries
	entries, total, err := log.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list: %v", err)
	}
	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}
	if len(entries) != 5 {
		t.Errorf("Expected 5 entries, got %d", len(entries))
	}
}

func TestLLMLogStore_GetStats(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	log := store.LLMLog()

	// Log entries with different statuses
	log.Log(ctx, &LLMLogEntry{
		RequestID: "req-1", Provider: "claude", Model: "haiku", Status: "success",
		InputTokens: 100, OutputTokens: 50, CostUSD: 0.001, RequestJSON: "{}",
	})
	log.Log(ctx, &LLMLogEntry{
		RequestID: "req-2", Provider: "claude", Model: "haiku", Status: "success",
		InputTokens: 200, OutputTokens: 100, CostUSD: 0.002, RequestJSON: "{}",
	})
	log.Log(ctx, &LLMLogEntry{
		RequestID: "req-3", Provider: "claude", Model: "haiku", Status: "error",
		ErrorMessage: "API error", RequestJSON: "{}",
	})

	stats, err := log.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats["total_requests"].(int64) != 3 {
		t.Errorf("Expected 3 total requests, got %d", stats["total_requests"])
	}
	if stats["success_count"].(int64) != 2 {
		t.Errorf("Expected 2 success, got %d", stats["success_count"])
	}
	if stats["error_count"].(int64) != 1 {
		t.Errorf("Expected 1 error, got %d", stats["error_count"])
	}
}

func TestLLMLogStore_DeleteOld(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	log := store.LLMLog()

	// Log an entry
	log.Log(ctx, &LLMLogEntry{
		RequestID: "old-req", Provider: "claude", Model: "haiku", Status: "success", RequestJSON: "{}",
	})

	// Verify entry exists
	_, total, _ := log.List(ctx, 10, 0)
	if total != 1 {
		t.Fatalf("Expected 1 entry, got %d", total)
	}

	// Delete with a very long duration (should delete nothing)
	// The function should work without error
	_, err := log.DeleteOld(ctx, 365*24*time.Hour) // 1 year
	if err != nil {
		t.Fatalf("Failed to call delete old: %v", err)
	}
}
