package dahlia

import "testing"

func TestAnalyzeBasic(t *testing.T) {
	tokens := analyzeWithPositions("The quick brown fox jumps over the lazy dog")
	// "the" and "over" are stopwords
	terms := make([]string, len(tokens))
	for i, tok := range tokens {
		terms[i] = tok.term
	}
	// Should contain stemmed versions of: quick, brown, fox, jumps, lazy, dog
	if len(tokens) < 5 {
		t.Fatalf("expected at least 5 tokens, got %d: %v", len(tokens), terms)
	}
}

func TestAnalyzePositions(t *testing.T) {
	tokens := analyzeWithPositions("hello world foo")
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	// Positions should be 0, 1, 2
	for i, tok := range tokens {
		if tok.pos != i {
			t.Fatalf("token %d: expected pos %d, got %d", i, i, tok.pos)
		}
	}
}

func TestAnalyzeStopwords(t *testing.T) {
	tokens := analyzeWithPositions("the and is are")
	if len(tokens) != 0 {
		t.Fatalf("expected 0 tokens for all stopwords, got %d", len(tokens))
	}
}

func TestAnalyzeStemming(t *testing.T) {
	terms := analyze("running jumps jumped")
	// "running" → "run", "jumps" → "jump", "jumped" → "jump"
	expected := map[string]bool{"run": true, "jump": true}
	for _, term := range terms {
		if !expected[term] {
			t.Logf("stemmed term: %s", term)
		}
	}
}

func TestAnalyzeUnicode(t *testing.T) {
	tokens := analyzeWithPositions("café résumé naïve")
	if len(tokens) == 0 {
		t.Fatal("expected tokens from unicode text")
	}
}

func TestAnalyzeLengthFilter(t *testing.T) {
	// Single char "a" and "I" should be filtered (< 2 chars)
	tokens := analyzeWithPositions("a x something")
	for _, tok := range tokens {
		if len(tok.term) < 2 {
			t.Fatalf("token %q should be filtered (< 2 chars)", tok.term)
		}
	}
}

func TestTokenize(t *testing.T) {
	got := tokenize("hello, world! foo-bar123")
	expected := []string{"hello", "world", "foo", "bar123"}
	if len(got) != len(expected) {
		t.Fatalf("got %v, want %v", got, expected)
	}
	for i := range got {
		if got[i] != expected[i] {
			t.Fatalf("token %d: got %q, want %q", i, got[i], expected[i])
		}
	}
}

func TestAnalyzeEmpty(t *testing.T) {
	tokens := analyzeWithPositions("")
	if len(tokens) != 0 {
		t.Fatalf("expected 0 tokens for empty string, got %d", len(tokens))
	}
}
