package views

import (
	"testing"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

func TestApplyFilter(t *testing.T) {
	s := &Service{}

	items := []*pages.Page{
		{
			ID: "1",
			Properties: pages.Properties{
				"status":   {Type: "select", Value: "done"},
				"priority": {Type: "select", Value: "high"},
				"estimate": {Type: "number", Value: float64(5)},
				"name":     {Type: "text", Value: "Task 1"},
			},
		},
		{
			ID: "2",
			Properties: pages.Properties{
				"status":   {Type: "select", Value: "in_progress"},
				"priority": {Type: "select", Value: "medium"},
				"estimate": {Type: "number", Value: float64(10)},
				"name":     {Type: "text", Value: "Task 2"},
			},
		},
		{
			ID: "3",
			Properties: pages.Properties{
				"status":   {Type: "select", Value: "done"},
				"priority": {Type: "select", Value: "low"},
				"estimate": {Type: "number", Value: float64(3)},
				"name":     {Type: "text", Value: "Another Task"},
			},
		},
	}

	tests := []struct {
		name     string
		filter   *Filter
		expected int
	}{
		{
			name:     "nil filter returns all",
			filter:   nil,
			expected: 3,
		},
		{
			name:     "equals operator",
			filter:   &Filter{PropertyID: "status", Operator: "equals", Value: "done"},
			expected: 2,
		},
		{
			name:     "does_not_equal operator",
			filter:   &Filter{PropertyID: "status", Operator: "does_not_equal", Value: "done"},
			expected: 1,
		},
		{
			name:     "contains operator",
			filter:   &Filter{PropertyID: "name", Operator: "contains", Value: "Task"},
			expected: 3,
		},
		{
			name:     "contains case insensitive",
			filter:   &Filter{PropertyID: "name", Operator: "contains", Value: "task"},
			expected: 3,
		},
		{
			name:     "starts_with operator",
			filter:   &Filter{PropertyID: "name", Operator: "starts_with", Value: "Task"},
			expected: 2,
		},
		{
			name:     "ends_with operator",
			filter:   &Filter{PropertyID: "name", Operator: "ends_with", Value: "Task"},
			expected: 1,
		},
		{
			name:     "greater_than operator",
			filter:   &Filter{PropertyID: "estimate", Operator: "greater_than", Value: float64(5)},
			expected: 1,
		},
		{
			name:     "less_than operator",
			filter:   &Filter{PropertyID: "estimate", Operator: "less_than", Value: float64(5)},
			expected: 1,
		},
		{
			name:     "greater_than_or_equal_to operator",
			filter:   &Filter{PropertyID: "estimate", Operator: "greater_than_or_equal_to", Value: float64(5)},
			expected: 2,
		},
		{
			name: "nested AND",
			filter: &Filter{
				And: []Filter{
					{PropertyID: "status", Operator: "equals", Value: "done"},
					{PropertyID: "priority", Operator: "equals", Value: "high"},
				},
			},
			expected: 1,
		},
		{
			name: "nested OR",
			filter: &Filter{
				Or: []Filter{
					{PropertyID: "status", Operator: "equals", Value: "done"},
					{PropertyID: "priority", Operator: "equals", Value: "medium"},
				},
			},
			expected: 3,
		},
		{
			name: "complex nested",
			filter: &Filter{
				And: []Filter{
					{PropertyID: "status", Operator: "equals", Value: "done"},
					{
						Or: []Filter{
							{PropertyID: "priority", Operator: "equals", Value: "high"},
							{PropertyID: "priority", Operator: "equals", Value: "low"},
						},
					},
				},
			},
			expected: 2,
		},
		{
			name:     "is_empty on missing property",
			filter:   &Filter{PropertyID: "missing", Operator: "is_empty"},
			expected: 3,
		},
		{
			name:     "is_not_empty on missing property",
			filter:   &Filter{PropertyID: "missing", Operator: "is_not_empty"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.applyFilter(items, tt.filter)
			if len(result) != tt.expected {
				t.Errorf("got %d items, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestApplySort(t *testing.T) {
	s := &Service{}

	items := []*pages.Page{
		{ID: "1", Properties: pages.Properties{"name": {Value: "Charlie"}, "score": {Value: float64(85)}}},
		{ID: "2", Properties: pages.Properties{"name": {Value: "Alice"}, "score": {Value: float64(95)}}},
		{ID: "3", Properties: pages.Properties{"name": {Value: "Bob"}, "score": {Value: float64(75)}}},
	}

	t.Run("sort by string ascending", func(t *testing.T) {
		result := s.applySort(items, []Sort{{PropertyID: "name", Direction: "asc"}})
		if result[0].ID != "2" || result[1].ID != "3" || result[2].ID != "1" {
			t.Errorf("unexpected order: %s, %s, %s", result[0].ID, result[1].ID, result[2].ID)
		}
	})

	t.Run("sort by string descending", func(t *testing.T) {
		result := s.applySort(items, []Sort{{PropertyID: "name", Direction: "desc"}})
		if result[0].ID != "1" || result[1].ID != "3" || result[2].ID != "2" {
			t.Errorf("unexpected order: %s, %s, %s", result[0].ID, result[1].ID, result[2].ID)
		}
	})

	t.Run("sort by number ascending", func(t *testing.T) {
		result := s.applySort(items, []Sort{{PropertyID: "score", Direction: "asc"}})
		if result[0].ID != "3" || result[1].ID != "1" || result[2].ID != "2" {
			t.Errorf("unexpected order: %s, %s, %s", result[0].ID, result[1].ID, result[2].ID)
		}
	})

	t.Run("sort by number descending", func(t *testing.T) {
		result := s.applySort(items, []Sort{{PropertyID: "score", Direction: "desc"}})
		if result[0].ID != "2" || result[1].ID != "1" || result[2].ID != "3" {
			t.Errorf("unexpected order: %s, %s, %s", result[0].ID, result[1].ID, result[2].ID)
		}
	})

	t.Run("empty sort returns original", func(t *testing.T) {
		result := s.applySort(items, nil)
		if len(result) != len(items) {
			t.Errorf("got %d items, want %d", len(result), len(items))
		}
	})
}

func TestDateFilter(t *testing.T) {
	s := &Service{}

	date1 := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)
	date3 := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	items := []*pages.Page{
		{ID: "1", Properties: pages.Properties{"due": {Value: date1.Format(time.RFC3339)}}},
		{ID: "2", Properties: pages.Properties{"due": {Value: date2.Format(time.RFC3339)}}},
		{ID: "3", Properties: pages.Properties{"due": {Value: date3.Format(time.RFC3339)}}},
	}

	t.Run("before date", func(t *testing.T) {
		result := s.applyFilter(items, &Filter{
			PropertyID: "due",
			Operator:   "before",
			Value:      date2.Format(time.RFC3339),
		})
		if len(result) != 1 || result[0].ID != "1" {
			t.Errorf("expected 1 item with ID 1, got %d items", len(result))
		}
	})

	t.Run("after date", func(t *testing.T) {
		result := s.applyFilter(items, &Filter{
			PropertyID: "due",
			Operator:   "after",
			Value:      date2.Format(time.RFC3339),
		})
		if len(result) != 1 || result[0].ID != "3" {
			t.Errorf("expected 1 item with ID 3, got %d items", len(result))
		}
	})

	t.Run("on_or_before date", func(t *testing.T) {
		result := s.applyFilter(items, &Filter{
			PropertyID: "due",
			Operator:   "on_or_before",
			Value:      date2.Format(time.RFC3339),
		})
		if len(result) != 2 {
			t.Errorf("expected 2 items, got %d", len(result))
		}
	})

	t.Run("on_or_after date", func(t *testing.T) {
		result := s.applyFilter(items, &Filter{
			PropertyID: "due",
			Operator:   "on_or_after",
			Value:      date2.Format(time.RFC3339),
		})
		if len(result) != 2 {
			t.Errorf("expected 2 items, got %d", len(result))
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("toString", func(t *testing.T) {
		if s := toString(nil); s != "" {
			t.Errorf("expected empty string for nil, got %q", s)
		}
		if s := toString("hello"); s != "hello" {
			t.Errorf("expected 'hello', got %q", s)
		}
		if s := toString(3.14); s != "3.14" {
			t.Errorf("expected '3.14', got %q", s)
		}
		if s := toString(true); s != "true" {
			t.Errorf("expected 'true', got %q", s)
		}
	})

	t.Run("toFloat", func(t *testing.T) {
		if f := toFloat(nil); f != 0 {
			t.Errorf("expected 0 for nil, got %f", f)
		}
		if f := toFloat(3.14); f != 3.14 {
			t.Errorf("expected 3.14, got %f", f)
		}
		if f := toFloat("3.14"); f != 3.14 {
			t.Errorf("expected 3.14, got %f", f)
		}
		if f := toFloat(42); f != 42 {
			t.Errorf("expected 42, got %f", f)
		}
	})

	t.Run("containsIgnoreCase", func(t *testing.T) {
		if !containsIgnoreCase("Hello World", "world") {
			t.Error("expected true for 'world' in 'Hello World'")
		}
		if containsIgnoreCase("Hello", "world") {
			t.Error("expected false for 'world' in 'Hello'")
		}
	})

	t.Run("startsWithIgnoreCase", func(t *testing.T) {
		if !startsWithIgnoreCase("Hello World", "hello") {
			t.Error("expected true for 'hello' at start of 'Hello World'")
		}
		if startsWithIgnoreCase("Hello World", "world") {
			t.Error("expected false for 'world' at start of 'Hello World'")
		}
	})

	t.Run("endsWithIgnoreCase", func(t *testing.T) {
		if !endsWithIgnoreCase("Hello World", "WORLD") {
			t.Error("expected true for 'WORLD' at end of 'Hello World'")
		}
		if endsWithIgnoreCase("Hello World", "hello") {
			t.Error("expected false for 'hello' at end of 'Hello World'")
		}
	})
}
