package sqlite

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

func TestHistoryStore_RecordSearch(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	history := s.History()

	entry := &store.SearchHistory{
		Query:      "test query",
		Results:    10,
		ClickedURL: "https://example.com",
	}

	if err := history.RecordSearch(ctx, entry); err != nil {
		t.Fatalf("RecordSearch() error = %v", err)
	}

	if entry.ID == "" {
		t.Error("expected ID to be set")
	}
	if entry.SearchedAt.IsZero() {
		t.Error("expected SearchedAt to be set")
	}
}

func TestHistoryStore_GetHistory(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	history := s.History()

	// Record some searches
	for i := 0; i < 5; i++ {
		entry := &store.SearchHistory{
			Query:   "query " + string(rune('A'+i)),
			Results: i * 10,
		}
		if err := history.RecordSearch(ctx, entry); err != nil {
			t.Fatalf("RecordSearch() error = %v", err)
		}
	}

	entries, err := history.GetHistory(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}

	if len(entries) != 5 {
		t.Errorf("len(entries) = %d, want 5", len(entries))
	}
}

func TestHistoryStore_GetHistory_OrderByRecent(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	history := s.History()

	entries := []*store.SearchHistory{
		{Query: "first"},
		{Query: "second"},
		{Query: "third"},
	}

	for _, e := range entries {
		if err := history.RecordSearch(ctx, e); err != nil {
			t.Fatalf("RecordSearch() error = %v", err)
		}
	}

	list, err := history.GetHistory(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}

	// Should be ordered by most recent first
	if list[0].Query != "third" {
		t.Errorf("first entry = %q, want 'third'", list[0].Query)
	}
}

func TestHistoryStore_GetHistory_Pagination(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	history := s.History()

	for i := 0; i < 20; i++ {
		entry := &store.SearchHistory{
			Query: "query " + string(rune('a'+i)),
		}
		if err := history.RecordSearch(ctx, entry); err != nil {
			t.Fatalf("RecordSearch() error = %v", err)
		}
	}

	// First page
	page1, err := history.GetHistory(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}

	if len(page1) != 10 {
		t.Errorf("page 1: len = %d, want 10", len(page1))
	}

	// Second page
	page2, err := history.GetHistory(ctx, 10, 10)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}

	if len(page2) != 10 {
		t.Errorf("page 2: len = %d, want 10", len(page2))
	}
}

func TestHistoryStore_GetHistory_DefaultLimit(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	history := s.History()

	for i := 0; i < 60; i++ {
		entry := &store.SearchHistory{
			Query: "query " + string(rune('A'+i%26)),
		}
		if err := history.RecordSearch(ctx, entry); err != nil {
			t.Fatalf("RecordSearch() error = %v", err)
		}
	}

	// Use default limit (0 or negative)
	entries, err := history.GetHistory(ctx, 0, 0)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}

	if len(entries) != 50 {
		t.Errorf("len(entries) = %d, want 50 (default)", len(entries))
	}
}

func TestHistoryStore_GetHistory_WithClickedURL(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	history := s.History()

	entry := &store.SearchHistory{
		Query:      "with click",
		Results:    5,
		ClickedURL: "https://clicked.example.com",
	}

	if err := history.RecordSearch(ctx, entry); err != nil {
		t.Fatalf("RecordSearch() error = %v", err)
	}

	entries, err := history.GetHistory(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}

	if entries[0].ClickedURL != "https://clicked.example.com" {
		t.Errorf("ClickedURL = %q, want 'https://clicked.example.com'", entries[0].ClickedURL)
	}
}

func TestHistoryStore_DeleteHistoryEntry(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	history := s.History()

	entry := &store.SearchHistory{
		Query: "to delete",
	}

	if err := history.RecordSearch(ctx, entry); err != nil {
		t.Fatalf("RecordSearch() error = %v", err)
	}

	if err := history.DeleteHistoryEntry(ctx, entry.ID); err != nil {
		t.Fatalf("DeleteHistoryEntry() error = %v", err)
	}

	// Verify deleted
	entries, err := history.GetHistory(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0 after deletion", len(entries))
	}
}

func TestHistoryStore_DeleteHistoryEntry_NotFound(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	history := s.History()

	err := history.DeleteHistoryEntry(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent entry")
	}
}

func TestHistoryStore_ClearHistory(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	history := s.History()

	// Add multiple entries
	for i := 0; i < 10; i++ {
		entry := &store.SearchHistory{
			Query: "query " + string(rune('A'+i)),
		}
		if err := history.RecordSearch(ctx, entry); err != nil {
			t.Fatalf("RecordSearch() error = %v", err)
		}
	}

	if err := history.ClearHistory(ctx); err != nil {
		t.Fatalf("ClearHistory() error = %v", err)
	}

	entries, err := history.GetHistory(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0 after clear", len(entries))
	}
}

func TestHistoryStore_ClearHistory_Empty(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	history := s.History()

	// Should not error on empty history
	if err := history.ClearHistory(ctx); err != nil {
		t.Errorf("ClearHistory() error = %v", err)
	}
}
