package qlocal

import (
	"strings"
	"testing"
)

func TestGlobToRegexp_DoubleStarMatchesRootFiles(t *testing.T) {
	re, err := globToRegexp("**/*.md")
	if err != nil {
		t.Fatalf("globToRegexp error: %v", err)
	}
	cases := []struct {
		path string
		want bool
	}{
		{"README.md", true},
		{"docs/README.md", true},
		{"docs/nested/file.md", true},
		{"README.txt", false},
	}
	for _, tc := range cases {
		if got := re.MatchString(tc.path); got != tc.want {
			t.Fatalf("match(%q)=%v want %v", tc.path, got, tc.want)
		}
	}
}

func TestParseStructuredQuery(t *testing.T) {
	t.Run("plain single line is implicit expand", func(t *testing.T) {
		out, err := ParseStructuredQuery("CAP theorem")
		if err != nil {
			t.Fatalf("ParseStructuredQuery error: %v", err)
		}
		if out != nil {
			t.Fatalf("expected nil for plain query, got %#v", out)
		}
	})

	t.Run("explicit expand returns nil", func(t *testing.T) {
		out, err := ParseStructuredQuery("expand: error handling best practices")
		if err != nil {
			t.Fatalf("ParseStructuredQuery error: %v", err)
		}
		if out != nil {
			t.Fatalf("expected nil for explicit expand, got %#v", out)
		}
	})

	t.Run("single typed line parses", func(t *testing.T) {
		out, err := ParseStructuredQuery("Lex: CAP theorem")
		if err != nil {
			t.Fatalf("ParseStructuredQuery error: %v", err)
		}
		if len(out) != 1 {
			t.Fatalf("len(out)=%d want 1", len(out))
		}
		if out[0].Type != "lex" || out[0].Query != "CAP theorem" || out[0].Line != 1 {
			t.Fatalf("unexpected output: %#v", out)
		}
	})

	t.Run("multi line typed preserves order and line numbers", func(t *testing.T) {
		in := "  hyde: hypothetical answer  \n\n vec: how parser works\nlex: compiler"
		out, err := ParseStructuredQuery(in)
		if err != nil {
			t.Fatalf("ParseStructuredQuery error: %v", err)
		}
		if len(out) != 3 {
			t.Fatalf("len(out)=%d want 3", len(out))
		}
		if out[0].Type != "hyde" || out[0].Line != 1 {
			t.Fatalf("unexpected first query: %#v", out[0])
		}
		if out[1].Type != "vec" || out[1].Line != 3 {
			t.Fatalf("unexpected second query: %#v", out[1])
		}
		if out[2].Type != "lex" || out[2].Line != 4 {
			t.Fatalf("unexpected third query: %#v", out[2])
		}
	})

	t.Run("mixed plain and typed lines errors", func(t *testing.T) {
		_, err := ParseStructuredQuery("plain keywords\nvec: semantic question")
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "missing a lex:/vec:/hyde: prefix") {
			t.Fatalf("expected missing-prefix error, got %v", err)
		}
	})

	t.Run("mixing expand with typed lines errors", func(t *testing.T) {
		_, err := ParseStructuredQuery("expand: question\nlex: keywords")
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "cannot mix expand") {
			t.Fatalf("expected expand-mix error, got %v", err)
		}
	})

	t.Run("expand without text errors", func(t *testing.T) {
		_, err := ParseStructuredQuery("expand:   ")
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "must include text") {
			t.Fatalf("expected empty-expand error, got %v", err)
		}
	})

	t.Run("typed line without text errors", func(t *testing.T) {
		_, err := ParseStructuredQuery("lex:   \nvec: real")
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "must include text") {
			t.Fatalf("expected empty-typed-line error, got %v", err)
		}
	})

	t.Run("colon in query text is preserved", func(t *testing.T) {
		out, err := ParseStructuredQuery("vec: what does lex: mean")
		if err != nil {
			t.Fatalf("ParseStructuredQuery error: %v", err)
		}
		if len(out) != 1 || out[0].Query != "what does lex: mean" {
			t.Fatalf("unexpected output: %#v", out)
		}
	})
}
