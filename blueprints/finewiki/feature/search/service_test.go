package search

import (
	"context"
	"errors"
	"testing"
)

// mockStore implements Store for testing.
type mockStore struct {
	results []Result
	err     error
	called  bool
	query   Query
}

func (m *mockStore) Search(ctx context.Context, q Query) ([]Result, error) {
	m.called = true
	m.query = q
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func TestService_Search(t *testing.T) {
	tests := []struct {
		name        string
		store       *mockStore
		query       Query
		wantResults int
		wantErr     bool
		wantCalled  bool
	}{
		{
			name:  "empty query returns empty results",
			store: &mockStore{},
			query: Query{Text: ""},
		},
		{
			name:  "short query returns empty results",
			store: &mockStore{},
			query: Query{Text: "a"},
		},
		{
			name: "valid query calls store",
			store: &mockStore{
				results: []Result{
					{ID: "1", Title: "Test"},
				},
			},
			query:       Query{Text: "test"},
			wantResults: 1,
			wantCalled:  true,
		},
		{
			name:       "store error propagates",
			store:      &mockStore{err: errors.New("db error")},
			query:      Query{Text: "test"},
			wantErr:    true,
			wantCalled: true,
		},
		{
			name: "default limit is set",
			store: &mockStore{
				results: []Result{},
			},
			query:      Query{Text: "test"},
			wantCalled: true,
		},
		{
			name:    "negative limit returns error",
			store:   &mockStore{},
			query:   Query{Text: "test", Limit: -1},
			wantErr: true,
		},
		{
			name: "limit capped at max",
			store: &mockStore{
				results: []Result{},
			},
			query:      Query{Text: "test", Limit: 500},
			wantCalled: true,
		},
		{
			name: "query whitespace normalized",
			store: &mockStore{
				results: []Result{},
			},
			query:      Query{Text: "  hello   world  "},
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := New(tt.store)
			results, err := svc.Search(context.Background(), tt.query)

			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) != tt.wantResults {
				t.Errorf("Search() returned %d results, want %d", len(results), tt.wantResults)
			}

			if tt.store.called != tt.wantCalled {
				t.Errorf("store.Search() called = %v, want %v", tt.store.called, tt.wantCalled)
			}
		})
	}
}

func TestService_NilStore(t *testing.T) {
	svc := New(nil)
	_, err := svc.Search(context.Background(), Query{Text: "test"})
	if err == nil {
		t.Error("expected error for nil store")
	}
}

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"  ", ""},
		{"hello", "hello"},
		{"  hello  ", "hello"},
		{"hello world", "hello world"},
		{"hello  world", "hello world"},
		{"  hello   world  ", "hello world"},
		{"hello\tworld", "hello world"},
		{"hello\nworld", "hello world"},
		{"hello\r\nworld", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeQuery(tt.input)
			if got != tt.want {
				t.Errorf("normalizeQuery(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRuneLen(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"a", 1},
		{"abc", 3},
		{"日本語", 3},
		{"hello世界", 7},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := runeLen(tt.input)
			if got != tt.want {
				t.Errorf("runeLen(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestService_LimitEnforcement(t *testing.T) {
	store := &mockStore{results: []Result{}}
	svc := New(store)

	// Test default limit
	svc.Search(context.Background(), Query{Text: "test", Limit: 0})
	if store.query.Limit != 20 {
		t.Errorf("default limit = %d, want 20", store.query.Limit)
	}

	// Test max limit
	store.called = false
	svc.Search(context.Background(), Query{Text: "test", Limit: 500})
	if store.query.Limit != 200 {
		t.Errorf("capped limit = %d, want 200", store.query.Limit)
	}

	// Test custom limit within bounds
	store.called = false
	svc.Search(context.Background(), Query{Text: "test", Limit: 50})
	if store.query.Limit != 50 {
		t.Errorf("custom limit = %d, want 50", store.query.Limit)
	}
}
