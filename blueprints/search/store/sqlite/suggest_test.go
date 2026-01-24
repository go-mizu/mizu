package sqlite

import (
	"context"
	"testing"
)

func TestSuggestStore_RecordQuery(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	suggest := s.Suggest()

	if err := suggest.RecordQuery(ctx, "golang tutorial"); err != nil {
		t.Fatalf("RecordQuery() error = %v", err)
	}

	// Verify by getting suggestions
	suggestions, err := suggest.GetSuggestions(ctx, "golang", 10)
	if err != nil {
		t.Fatalf("GetSuggestions() error = %v", err)
	}

	if len(suggestions) == 0 {
		t.Error("expected at least one suggestion")
	}
}

func TestSuggestStore_RecordQuery_EmptyQuery(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	suggest := s.Suggest()

	// Should not error on empty query
	if err := suggest.RecordQuery(ctx, ""); err != nil {
		t.Errorf("RecordQuery() error = %v", err)
	}

	if err := suggest.RecordQuery(ctx, "   "); err != nil {
		t.Errorf("RecordQuery() error = %v", err)
	}
}

func TestSuggestStore_RecordQuery_IncrementFrequency(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	suggest := s.Suggest()

	// Record same query multiple times
	for i := 0; i < 5; i++ {
		if err := suggest.RecordQuery(ctx, "popular query"); err != nil {
			t.Fatalf("RecordQuery() error = %v", err)
		}
	}

	suggestions, err := suggest.GetSuggestions(ctx, "popular", 10)
	if err != nil {
		t.Fatalf("GetSuggestions() error = %v", err)
	}

	if len(suggestions) != 1 {
		t.Fatalf("len(suggestions) = %d, want 1", len(suggestions))
	}

	if suggestions[0].Frequency != 5 {
		t.Errorf("Frequency = %d, want 5", suggestions[0].Frequency)
	}
}

func TestSuggestStore_GetSuggestions(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	suggest := s.Suggest()

	// Record various queries
	queries := []string{
		"go programming",
		"go tutorial",
		"go web development",
		"python programming",
		"rust language",
	}

	for _, q := range queries {
		if err := suggest.RecordQuery(ctx, q); err != nil {
			t.Fatalf("RecordQuery() error = %v", err)
		}
	}

	// Get suggestions starting with "go"
	suggestions, err := suggest.GetSuggestions(ctx, "go", 10)
	if err != nil {
		t.Fatalf("GetSuggestions() error = %v", err)
	}

	if len(suggestions) != 3 {
		t.Errorf("len(suggestions) = %d, want 3", len(suggestions))
	}

	for _, s := range suggestions {
		if s.Type != "query" {
			t.Errorf("Type = %q, want 'query'", s.Type)
		}
	}
}

func TestSuggestStore_GetSuggestions_EmptyPrefix(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	suggest := s.Suggest()

	suggestions, err := suggest.GetSuggestions(ctx, "", 10)
	if err != nil {
		t.Fatalf("GetSuggestions() error = %v", err)
	}

	if len(suggestions) != 0 {
		t.Errorf("len(suggestions) = %d, want 0 for empty prefix", len(suggestions))
	}
}

func TestSuggestStore_GetSuggestions_Limit(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	suggest := s.Suggest()

	// Record many queries
	for i := 0; i < 20; i++ {
		if err := suggest.RecordQuery(ctx, "test"+string(rune('a'+i))); err != nil {
			t.Fatalf("RecordQuery() error = %v", err)
		}
	}

	// Request with limit
	suggestions, err := suggest.GetSuggestions(ctx, "test", 5)
	if err != nil {
		t.Fatalf("GetSuggestions() error = %v", err)
	}

	if len(suggestions) != 5 {
		t.Errorf("len(suggestions) = %d, want 5", len(suggestions))
	}
}

func TestSuggestStore_GetSuggestions_CaseInsensitive(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	suggest := s.Suggest()

	if err := suggest.RecordQuery(ctx, "JavaScript Tutorial"); err != nil {
		t.Fatalf("RecordQuery() error = %v", err)
	}

	// Search with different case
	suggestions, err := suggest.GetSuggestions(ctx, "javascript", 10)
	if err != nil {
		t.Fatalf("GetSuggestions() error = %v", err)
	}

	if len(suggestions) != 1 {
		t.Errorf("len(suggestions) = %d, want 1 (case-insensitive match)", len(suggestions))
	}
}

func TestSuggestStore_GetSuggestions_OrderByFrequency(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	suggest := s.Suggest()

	// Record queries with different frequencies - prefix search so query must start with prefix
	for i := 0; i < 1; i++ {
		suggest.RecordQuery(ctx, "test rare")
	}
	for i := 0; i < 5; i++ {
		suggest.RecordQuery(ctx, "test common")
	}
	for i := 0; i < 10; i++ {
		suggest.RecordQuery(ctx, "test popular")
	}

	suggestions, err := suggest.GetSuggestions(ctx, "test", 10)
	if err != nil {
		t.Fatalf("GetSuggestions() error = %v", err)
	}

	if len(suggestions) != 3 {
		t.Fatalf("len(suggestions) = %d, want 3", len(suggestions))
	}

	// Should be ordered by frequency
	if suggestions[0].Text != "test popular" {
		t.Errorf("first suggestion = %q, want 'test popular'", suggestions[0].Text)
	}
}

func TestSuggestStore_GetTrendingQueries(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	suggest := s.Suggest()

	// Record queries with different frequencies
	for i := 0; i < 10; i++ {
		suggest.RecordQuery(ctx, "trending topic")
	}
	for i := 0; i < 5; i++ {
		suggest.RecordQuery(ctx, "somewhat popular")
	}
	for i := 0; i < 1; i++ {
		suggest.RecordQuery(ctx, "not popular")
	}

	trending, err := suggest.GetTrendingQueries(ctx, 10)
	if err != nil {
		t.Fatalf("GetTrendingQueries() error = %v", err)
	}

	if len(trending) != 3 {
		t.Fatalf("len(trending) = %d, want 3", len(trending))
	}

	if trending[0] != "trending topic" {
		t.Errorf("first trending = %q, want 'trending topic'", trending[0])
	}
}

func TestSuggestStore_GetTrendingQueries_Limit(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	suggest := s.Suggest()

	for i := 0; i < 20; i++ {
		suggest.RecordQuery(ctx, "trending"+string(rune('a'+i)))
	}

	trending, err := suggest.GetTrendingQueries(ctx, 5)
	if err != nil {
		t.Fatalf("GetTrendingQueries() error = %v", err)
	}

	if len(trending) != 5 {
		t.Errorf("len(trending) = %d, want 5", len(trending))
	}
}

func TestSuggestStore_GetTrendingQueries_DefaultLimit(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	suggest := s.Suggest()

	for i := 0; i < 15; i++ {
		suggest.RecordQuery(ctx, "topic"+string(rune('a'+i)))
	}

	// Use default limit (0 or negative)
	trending, err := suggest.GetTrendingQueries(ctx, 0)
	if err != nil {
		t.Fatalf("GetTrendingQueries() error = %v", err)
	}

	if len(trending) != 10 {
		t.Errorf("len(trending) = %d, want 10 (default limit)", len(trending))
	}
}
