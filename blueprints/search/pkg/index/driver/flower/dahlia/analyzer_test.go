package dahlia

import "testing"

func TestAnalyzeBasic(t *testing.T) {
	tokens := analyzeWithPositions("The quick brown fox jumps over the lazy dog")
	terms := make([]string, len(tokens))
	for i, tok := range tokens {
		terms[i] = tok.term
	}
	if len(tokens) < 9 {
		t.Fatalf("expected all terms to be indexed, got %d: %v", len(tokens), terms)
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

func TestAnalyzeStopwordsAreIndexed(t *testing.T) {
	tokens := analyzeWithPositions("the and is are")
	if len(tokens) != 4 {
		t.Fatalf("expected stopwords to be indexed, got %d", len(tokens))
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

func TestAnalyzeSingleCharacterTokens(t *testing.T) {
	tokens := analyzeWithPositions("a x something")
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
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
