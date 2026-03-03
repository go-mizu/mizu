package rose

import (
	"reflect"
	"testing"
)

func TestAnalyze_Basic(t *testing.T) {
	got := analyze("Machine learning algorithms")
	if len(got) == 0 {
		t.Fatal("expected tokens, got none")
	}
	found := false
	for _, tok := range got {
		if tok == "learn" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected stemmed 'learn' in %v", got)
	}
}

func TestAnalyze_Stopwords(t *testing.T) {
	got := analyze("the cat and the dog is here")
	for _, tok := range got {
		if tok == "the" || tok == "and" || tok == "is" {
			t.Errorf("stopword %q should be removed, got %v", tok, got)
		}
	}
}

func TestAnalyze_Empty(t *testing.T) {
	if len(analyze("")) != 0 {
		t.Error("empty → []")
	}
}

func TestAnalyze_MinLen(t *testing.T) {
	// Single char "I" should be dropped (< 2 bytes after lowercase)
	got := analyze("I go")
	for _, tok := range got {
		if len(tok) < 2 {
			t.Errorf("token %q too short (< 2)", tok)
		}
	}
}

func TestAnalyze_MaxLen(t *testing.T) {
	long := ""
	for i := 0; i < 65; i++ {
		long += "a"
	} // 65-char token
	got := analyze("normal " + long)
	for _, tok := range got {
		if len(tok) > 64 {
			t.Errorf("token %q too long (> 64)", tok)
		}
	}
}

func TestAnalyze_Unicode(t *testing.T) {
	got := analyze("Café résumé naïve")
	if len(got) == 0 {
		t.Fatal("unicode tokens should not be empty")
	}
}

func TestAnalyze_Dedup(t *testing.T) {
	got := analyze("cat cat cat")
	count := 0
	for _, tok := range got {
		if tok == "cat" {
			count++
		}
	}
	if count != 3 {
		t.Errorf("expected 3 'cat', got %d in %v", count, got)
	}
}

func TestAnalyze_Punctuation(t *testing.T) {
	got := analyze("Hello, world!")
	for _, tok := range got {
		if tok == "hello," || tok == "world!" {
			t.Errorf("punctuation not stripped: %q", tok)
		}
	}
}

func TestAnalyzeQuery_SameAsIndex(t *testing.T) {
	if !reflect.DeepEqual(analyze("machine learning"), analyzeQuery("machine learning")) {
		t.Error("analyzeQuery must produce identical output to analyze")
	}
}

func TestProcessTok_Stem(t *testing.T) {
	// "running" should stem to "run"
	got := processTok("running")
	if got != "run" {
		t.Errorf("processTok('running') = %q, want 'run'", got)
	}
}

func TestProcessTok_Stopword(t *testing.T) {
	if processTok("the") != "" {
		t.Error("'the' should return empty (stopword)")
	}
}

func TestProcessTok_TooShort(t *testing.T) {
	if processTok("a") != "" {
		t.Error("single char should return empty")
	}
}

func TestProcessTok_TooLong(t *testing.T) {
	long := ""
	for i := 0; i < 65; i++ {
		long += "x"
	}
	if processTok(long) != "" {
		t.Error("65-char token should return empty")
	}
}
