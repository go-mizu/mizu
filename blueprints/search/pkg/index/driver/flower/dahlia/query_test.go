package dahlia

import "testing"

func TestParseQuerySingleTerm(t *testing.T) {
	q := parseQuery("hello")
	tq, ok := q.(termQuery)
	if !ok {
		t.Fatalf("expected termQuery, got %T", q)
	}
	if tq.term == "" {
		t.Fatal("term should not be empty")
	}
}

func TestParseQueryPhrase(t *testing.T) {
	q := parseQuery(`"hello world"`)
	pq, ok := q.(phraseQuery)
	if !ok {
		t.Fatalf("expected phraseQuery, got %T (%+v)", q, q)
	}
	if len(pq.terms) != 2 {
		t.Fatalf("expected 2 terms, got %d: %v", len(pq.terms), pq.terms)
	}
}

func TestParseQueryBooleanMust(t *testing.T) {
	q := parseQuery("+apple +banana")
	bq, ok := q.(booleanQuery)
	if !ok {
		t.Fatalf("expected booleanQuery, got %T", q)
	}
	if len(bq.must) != 2 {
		t.Fatalf("expected 2 must clauses, got %d", len(bq.must))
	}
	if len(bq.should) != 0 {
		t.Fatalf("expected 0 should clauses, got %d", len(bq.should))
	}
}

func TestParseQueryBooleanShould(t *testing.T) {
	q := parseQuery("apple banana cherry")
	bq, ok := q.(booleanQuery)
	if !ok {
		t.Fatalf("expected booleanQuery, got %T", q)
	}
	if len(bq.should) != 3 {
		t.Fatalf("expected 3 should clauses, got %d", len(bq.should))
	}
}

func TestParseQueryMustNot(t *testing.T) {
	q := parseQuery("+apple -banana")
	bq, ok := q.(booleanQuery)
	if !ok {
		t.Fatalf("expected booleanQuery, got %T", q)
	}
	if len(bq.must) != 1 {
		t.Fatalf("expected 1 must, got %d", len(bq.must))
	}
	if len(bq.mustNot) != 1 {
		t.Fatalf("expected 1 mustNot, got %d", len(bq.mustNot))
	}
}

func TestParseQueryPhraseWithPrefix(t *testing.T) {
	q := parseQuery(`+"machine learning"`)
	bq, ok := q.(booleanQuery)
	if !ok {
		t.Fatalf("expected booleanQuery, got %T", q)
	}
	if len(bq.must) != 1 {
		t.Fatalf("expected 1 must, got %d", len(bq.must))
	}
	_, isPhraseQ := bq.must[0].(phraseQuery)
	if !isPhraseQ {
		t.Fatalf("must[0] should be phraseQuery, got %T", bq.must[0])
	}
}

func TestParseQueryEmpty(t *testing.T) {
	q := parseQuery("")
	bq, ok := q.(booleanQuery)
	if !ok {
		t.Fatalf("expected booleanQuery, got %T", q)
	}
	if len(bq.must)+len(bq.should)+len(bq.mustNot) != 0 {
		t.Fatal("empty query should have no clauses")
	}
}

func TestParseQueryStopwordsOnly(t *testing.T) {
	q := parseQuery("the and is")
	bq, ok := q.(booleanQuery)
	if !ok {
		t.Fatalf("expected booleanQuery, got %T", q)
	}
	if len(bq.must)+len(bq.should)+len(bq.mustNot) != 0 {
		t.Fatal("stopwords-only query should have no clauses")
	}
}

func TestSplitQueryTokens(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{`hello world`, 2},
		{`"hello world"`, 1},
		{`+hello -world`, 2},
		{`+"hello world" -foo`, 2},
		{``, 0},
	}
	for _, tt := range tests {
		tokens := splitQueryTokens(tt.input)
		if len(tokens) != tt.want {
			t.Errorf("splitQueryTokens(%q) = %v (len %d), want len %d", tt.input, tokens, len(tokens), tt.want)
		}
	}
}
