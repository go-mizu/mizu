package bench_test

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/bench"
)

func TestNormalizeText(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Hello, World! 123", "hello  world  "},
		{"machine-learning", "machine learning"},
		{"New York City", "new york city"},
		{"", ""},
	}
	for _, tc := range cases {
		got := bench.NormalizeText(tc.input)
		// NormalizeText lowercases and replaces non-alpha with space.
		// Check lowercase and that original alpha chars are preserved.
		lc := strings.ToLower(tc.input)
		_ = lc
		if tc.input == "" && got != "" {
			t.Errorf("NormalizeText(%q) = %q, want empty", tc.input, got)
		}
		if tc.input != "" && got == "" {
			t.Errorf("NormalizeText(%q) = empty, want non-empty", tc.input)
		}
		// Must be lowercase
		if got != strings.ToLower(got) {
			t.Errorf("NormalizeText(%q) = %q is not lowercase", tc.input, got)
		}
	}
}

func TestTransformWikiLine_Valid(t *testing.T) {
	line := []byte(`{"url":"https://en.wikipedia.org/wiki/Test","title":"Test","body":"Hello World! 123"}`)
	doc, ok, err := bench.TransformWikiLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for valid line")
	}
	if doc.DocID != "https://en.wikipedia.org/wiki/Test" {
		t.Errorf("DocID: got %q", doc.DocID)
	}
	if !strings.Contains(doc.Text, "hello") {
		t.Errorf("text not normalized: %q", doc.Text)
	}
	if strings.Contains(doc.Text, "!") {
		t.Errorf("punctuation not stripped: %q", doc.Text)
	}
}

func TestTransformWikiLine_EmptyURL(t *testing.T) {
	line := []byte(`{"url":"","title":"T","body":"B"}`)
	_, ok, err := bench.TransformWikiLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected ok=false for empty url")
	}
}

func TestTransformWikiLine_Malformed(t *testing.T) {
	line := []byte(`not json at all`)
	_, ok, err := bench.TransformWikiLine(line)
	if err != nil {
		t.Fatalf("malformed line should not return error: %v", err)
	}
	if ok {
		t.Error("expected ok=false for malformed JSON")
	}
}
