package lotus

import "testing"

func TestAnalyze_Basic(t *testing.T) {
	terms := analyze("Hello World! This is a test.")
	if len(terms) < 2 {
		t.Fatalf("expected at least 2 tokens, got %d: %v", len(terms), terms)
	}
	for _, term := range terms {
		if term == "this" || term == "is" || term == "a" {
			t.Fatalf("stopword %q should be filtered", term)
		}
	}
}

func TestAnalyze_Positions(t *testing.T) {
	tokens := analyzeWithPositions("the quick brown fox")
	// "the" is stopword at pos 0, removed
	// "quick"=pos1, "brown"=pos2, "fox"=pos3
	for _, tok := range tokens {
		if tok.term == "the" {
			t.Fatal("stopword 'the' should be filtered")
		}
	}
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d: %v", len(tokens), tokens)
	}
	// Positions should be > 0 (since "the" at pos 0 was removed)
	if tokens[0].pos == 0 {
		t.Fatal("first token should not be at position 0 (that's 'the')")
	}
}

func TestAnalyze_Unicode(t *testing.T) {
	tokens := analyzeWithPositions("Unicode cafe resume")
	if len(tokens) < 2 {
		t.Fatalf("expected at least 2 tokens, got %d", len(tokens))
	}
}

func TestAnalyze_LengthFilter(t *testing.T) {
	tokens := analyzeWithPositions("I a b hello world")
	for _, tok := range tokens {
		if len(tok.term) < 2 {
			t.Fatalf("single-char token %q should be filtered", tok.term)
		}
	}
}

func TestAnalyze_Stemming(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"running", "run"},
		{"dogs", "dog"},
	}
	for _, c := range cases {
		terms := analyze(c.input)
		if len(terms) != 1 {
			t.Fatalf("analyze(%q) returned %d terms, want 1", c.input, len(terms))
		}
		if terms[0] != c.want {
			t.Fatalf("analyze(%q) = %q, want %q", c.input, terms[0], c.want)
		}
	}
}
