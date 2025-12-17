package view

import (
	"html/template"
	"testing"
)

func TestDictFunc(t *testing.T) {
	t.Run("valid pairs", func(t *testing.T) {
		m, err := dictFunc("key1", "value1", "key2", 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m["key1"] != "value1" {
			t.Errorf("expected key1='value1', got %v", m["key1"])
		}
		if m["key2"] != 42 {
			t.Errorf("expected key2=42, got %v", m["key2"])
		}
	})

	t.Run("odd number of args", func(t *testing.T) {
		_, err := dictFunc("key1", "value1", "key2")
		if err == nil {
			t.Error("expected error for odd number of arguments")
		}
	})

	t.Run("non-string key", func(t *testing.T) {
		_, err := dictFunc(123, "value1")
		if err == nil {
			t.Error("expected error for non-string key")
		}
	})
}

func TestListFunc(t *testing.T) {
	list := listFunc("a", "b", "c")
	if len(list) != 3 {
		t.Errorf("expected 3 items, got %d", len(list))
	}
	if list[0] != "a" || list[1] != "b" || list[2] != "c" {
		t.Errorf("unexpected list contents: %v", list)
	}
}

func TestDefaultFunc(t *testing.T) {
	t.Run("returns default for nil", func(t *testing.T) {
		result := defaultFunc("default", nil)
		if result != "default" {
			t.Errorf("expected 'default', got %v", result)
		}
	})

	t.Run("returns default for empty string", func(t *testing.T) {
		result := defaultFunc("default", "")
		if result != "default" {
			t.Errorf("expected 'default', got %v", result)
		}
	})

	t.Run("returns value when not empty", func(t *testing.T) {
		result := defaultFunc("default", "actual")
		if result != "actual" {
			t.Errorf("expected 'actual', got %v", result)
		}
	})

	t.Run("returns default for zero int", func(t *testing.T) {
		result := defaultFunc(42, 0)
		if result != 42 {
			t.Errorf("expected 42, got %v", result)
		}
	})
}

func TestEmptyFunc(t *testing.T) {
	tests := []struct {
		name     string
		val      any
		expected bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"false bool", false, true},
		{"true bool", true, false},
		{"empty slice", []string{}, true},
		{"non-empty slice", []string{"a"}, false},
		{"empty map", map[string]int{}, true},
		{"non-empty map", map[string]int{"a": 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := emptyFunc(tt.val)
			if result != tt.expected {
				t.Errorf("empty(%v) = %v, want %v", tt.val, result, tt.expected)
			}
		})
	}
}

func TestSafeFuncs(t *testing.T) {
	t.Run("safeHTML", func(t *testing.T) {
		result := safeHTMLFunc("<b>bold</b>")
		if result != template.HTML("<b>bold</b>") {
			t.Errorf("unexpected result: %v", result)
		}
	})

	t.Run("safeCSS", func(t *testing.T) {
		result := safeCSSFunc("color: red")
		if result != template.CSS("color: red") {
			t.Errorf("unexpected result: %v", result)
		}
	})

	t.Run("safeJS", func(t *testing.T) {
		result := safeJSFunc("alert('hi')")
		if result != template.JS("alert('hi')") {
			t.Errorf("unexpected result: %v", result)
		}
	})

	t.Run("safeURL", func(t *testing.T) {
		result := safeURLFunc("https://example.com")
		if result != template.URL("https://example.com") {
			t.Errorf("unexpected result: %v", result)
		}
	})
}

func TestTernaryFunc(t *testing.T) {
	t.Run("true condition", func(t *testing.T) {
		result := ternaryFunc(true, "yes", "no")
		if result != "yes" {
			t.Errorf("expected 'yes', got %v", result)
		}
	})

	t.Run("false condition", func(t *testing.T) {
		result := ternaryFunc(false, "yes", "no")
		if result != "no" {
			t.Errorf("expected 'no', got %v", result)
		}
	})
}

func TestCoalesceFunc(t *testing.T) {
	t.Run("returns first non-empty", func(t *testing.T) {
		result := coalesceFunc("", nil, "value", "other")
		if result != "value" {
			t.Errorf("expected 'value', got %v", result)
		}
	})

	t.Run("returns nil if all empty", func(t *testing.T) {
		result := coalesceFunc("", nil, 0)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestComparisonFuncs(t *testing.T) {
	t.Run("eq", func(t *testing.T) {
		if !eqFunc(5, 5) {
			t.Error("expected 5 == 5")
		}
		if eqFunc(5, 6) {
			t.Error("expected 5 != 6")
		}
	})

	t.Run("ne", func(t *testing.T) {
		if !neFunc(5, 6) {
			t.Error("expected 5 != 6")
		}
		if neFunc(5, 5) {
			t.Error("expected 5 == 5")
		}
	})

	t.Run("lt", func(t *testing.T) {
		if !ltFunc(5, 6) {
			t.Error("expected 5 < 6")
		}
		if ltFunc(6, 5) {
			t.Error("expected not 6 < 5")
		}
	})

	t.Run("le", func(t *testing.T) {
		if !leFunc(5, 6) {
			t.Error("expected 5 <= 6")
		}
		if !leFunc(5, 5) {
			t.Error("expected 5 <= 5")
		}
	})

	t.Run("gt", func(t *testing.T) {
		if !gtFunc(6, 5) {
			t.Error("expected 6 > 5")
		}
		if gtFunc(5, 6) {
			t.Error("expected not 5 > 6")
		}
	})

	t.Run("ge", func(t *testing.T) {
		if !geFunc(6, 5) {
			t.Error("expected 6 >= 5")
		}
		if !geFunc(5, 5) {
			t.Error("expected 5 >= 5")
		}
	})
}

func TestMathFuncs(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		result := addFunc(5, 3)
		if result != 8.0 {
			t.Errorf("expected 8, got %v", result)
		}
	})

	t.Run("sub", func(t *testing.T) {
		result := subFunc(5, 3)
		if result != 2.0 {
			t.Errorf("expected 2, got %v", result)
		}
	})

	t.Run("mul", func(t *testing.T) {
		result := mulFunc(5, 3)
		if result != 15.0 {
			t.Errorf("expected 15, got %v", result)
		}
	})

	t.Run("div", func(t *testing.T) {
		result := divFunc(6, 2)
		if result != 3.0 {
			t.Errorf("expected 3, got %v", result)
		}
	})

	t.Run("div by zero", func(t *testing.T) {
		result := divFunc(6, 0)
		if result != 0.0 {
			t.Errorf("expected 0, got %v", result)
		}
	})

	t.Run("mod", func(t *testing.T) {
		result := modFunc(7, 3)
		if result != int64(1) {
			t.Errorf("expected 1, got %v", result)
		}
	})

	t.Run("mod by zero", func(t *testing.T) {
		result := modFunc(7, 0)
		if result != int64(0) {
			t.Errorf("expected 0, got %v", result)
		}
	})
}
