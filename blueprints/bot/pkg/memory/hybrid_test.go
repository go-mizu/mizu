package memory

import (
	"math"
	"testing"
)

func TestBM25RankToScore_Rank0(t *testing.T) {
	// rank 0 => 1/(1+0) = 1.0
	got := BM25RankToScore(0)
	if got != 1.0 {
		t.Errorf("BM25RankToScore(0) = %f, want 1.0", got)
	}
}

func TestBM25RankToScore_NegativeRank(t *testing.T) {
	// rank -1 => 1/(1+1) = 0.5
	got := BM25RankToScore(-1)
	if got != 0.5 {
		t.Errorf("BM25RankToScore(-1) = %f, want 0.5", got)
	}

	// rank -9 => 1/(1+9) = 0.1
	got = BM25RankToScore(-9)
	if math.Abs(got-0.1) > 1e-9 {
		t.Errorf("BM25RankToScore(-9) = %f, want 0.1", got)
	}
}

func TestBM25RankToScore_PositiveRank(t *testing.T) {
	// rank 4 => 1/(1+4) = 0.2
	got := BM25RankToScore(4)
	if got != 0.2 {
		t.Errorf("BM25RankToScore(4) = %f, want 0.2", got)
	}
}

func TestBuildFTSQuery_Normal(t *testing.T) {
	got := BuildFTSQuery("hello world")
	want := `"hello" AND "world"`
	if got != want {
		t.Errorf("BuildFTSQuery(\"hello world\") = %q, want %q", got, want)
	}
}

func TestBuildFTSQuery_Empty(t *testing.T) {
	got := BuildFTSQuery("")
	if got != "" {
		t.Errorf("BuildFTSQuery(\"\") = %q, want \"\"", got)
	}
}

func TestBuildFTSQuery_SpecialChars(t *testing.T) {
	// Special characters should be stripped, only alpha+digit tokens remain.
	got := BuildFTSQuery("foo!@#$bar")
	want := `"foo" AND "bar"`
	if got != want {
		t.Errorf("BuildFTSQuery(\"foo!@#$bar\") = %q, want %q", got, want)
	}
}

func TestBuildFTSQuery_OnlySpecialChars(t *testing.T) {
	got := BuildFTSQuery("!@#$%^&*()")
	if got != "" {
		t.Errorf("BuildFTSQuery(\"!@#$%%^&*()\") = %q, want \"\"", got)
	}
}

func TestMergeHybridResults_VectorOnly(t *testing.T) {
	cfg := DefaultHybridConfig()
	vector := []VectorResult{
		{Path: "a.go", StartLine: 1, EndLine: 10, Score: 0.9, Snippet: "vector match"},
	}

	results := MergeHybridResults(vector, nil, cfg)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Vector-only: score = 0.7 * 0.9 = 0.63
	want := 0.7 * 0.9
	if math.Abs(results[0].Score-want) > 1e-9 {
		t.Errorf("Score = %f, want %f", results[0].Score, want)
	}
}

func TestMergeHybridResults_KeywordOnly_FilteredByMinScore(t *testing.T) {
	cfg := DefaultHybridConfig() // MinScore = 0.35
	keyword := []KeywordResult{
		{Path: "b.go", StartLine: 5, EndLine: 15, Rank: -1.0, Snippet: "keyword match"},
	}

	// Keyword-only: score = 0.3 * BM25RankToScore(-1) = 0.3 * 0.5 = 0.15
	// 0.15 < MinScore (0.35) => filtered out.
	results := MergeHybridResults(nil, keyword, cfg)
	if len(results) != 0 {
		t.Fatalf("expected 0 results (0.15 < MinScore 0.35), got %d with score %f",
			len(results), results[0].Score)
	}
}

func TestMergeHybridResults_KeywordOnly_AboveMin(t *testing.T) {
	cfg := DefaultHybridConfig()
	cfg.MinScore = 0.1 // lower threshold to let keyword-only results through
	keyword := []KeywordResult{
		{Path: "b.go", StartLine: 5, EndLine: 15, Rank: -1.0, Snippet: "keyword match"},
	}

	results := MergeHybridResults(nil, keyword, cfg)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Keyword-only: score = 0.3 * BM25RankToScore(-1) = 0.3 * 0.5 = 0.15
	want := 0.3 * 0.5
	if math.Abs(results[0].Score-want) > 1e-9 {
		t.Errorf("Score = %f, want %f", results[0].Score, want)
	}
}

func TestMergeHybridResults_BothSources_Boost(t *testing.T) {
	cfg := DefaultHybridConfig()
	vector := []VectorResult{
		{Path: "c.go", StartLine: 1, EndLine: 10, Score: 0.9, Snippet: "match"},
	}
	keyword := []KeywordResult{
		{Path: "c.go", StartLine: 1, EndLine: 10, Rank: -1.0, Snippet: "match"},
	}

	results := MergeHybridResults(vector, keyword, cfg)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Both: (0.7*0.9 + 0.3*0.5) * 1.1 = (0.63 + 0.15) * 1.1 = 0.858
	base := 0.7*0.9 + 0.3*0.5
	want := math.Min(base*1.1, 1.0)
	if math.Abs(results[0].Score-want) > 1e-9 {
		t.Errorf("Score = %f, want %f", results[0].Score, want)
	}
}

func TestMergeHybridResults_MinScoreFilter(t *testing.T) {
	cfg := DefaultHybridConfig()
	// A vector result with a low score that won't meet 0.35 threshold.
	vector := []VectorResult{
		{Path: "low.go", StartLine: 1, EndLine: 5, Score: 0.1, Snippet: "low"},
	}
	// 0.7 * 0.1 = 0.07 < 0.35 => should be filtered.
	results := MergeHybridResults(vector, nil, cfg)
	if len(results) != 0 {
		t.Errorf("expected 0 results (below MinScore), got %d", len(results))
	}
}

func TestMergeHybridResults_MaxResults(t *testing.T) {
	cfg := DefaultHybridConfig() // MaxResults = 6
	var vector []VectorResult
	for i := 0; i < 10; i++ {
		vector = append(vector, VectorResult{
			Path: "file.go", StartLine: i * 10, EndLine: i*10 + 9,
			Score: 0.9, Snippet: "chunk",
		})
	}

	results := MergeHybridResults(vector, nil, cfg)
	if len(results) > cfg.MaxResults {
		t.Errorf("got %d results, want at most %d", len(results), cfg.MaxResults)
	}
}

func TestMergeHybridResults_SortedDescending(t *testing.T) {
	cfg := DefaultHybridConfig()
	cfg.MinScore = 0.0 // accept everything

	vector := []VectorResult{
		{Path: "a.go", StartLine: 1, EndLine: 5, Score: 0.5, Snippet: "a"},
		{Path: "b.go", StartLine: 1, EndLine: 5, Score: 0.9, Snippet: "b"},
		{Path: "c.go", StartLine: 1, EndLine: 5, Score: 0.7, Snippet: "c"},
	}

	results := MergeHybridResults(vector, nil, cfg)
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted descending: result[%d].Score (%f) > result[%d].Score (%f)",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

func TestDefaultHybridConfig(t *testing.T) {
	cfg := DefaultHybridConfig()
	if cfg.VectorWeight != 0.7 {
		t.Errorf("VectorWeight = %f, want 0.7", cfg.VectorWeight)
	}
	if cfg.TextWeight != 0.3 {
		t.Errorf("TextWeight = %f, want 0.3", cfg.TextWeight)
	}
	if cfg.MinScore != 0.35 {
		t.Errorf("MinScore = %f, want 0.35", cfg.MinScore)
	}
	if cfg.MaxResults != 6 {
		t.Errorf("MaxResults = %d, want 6", cfg.MaxResults)
	}
	if cfg.CandidateMultiplier != 4 {
		t.Errorf("CandidateMultiplier = %d, want 4", cfg.CandidateMultiplier)
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"Hello World", []string{"hello", "world"}},
		{"foo123bar", []string{"foo123bar"}},
		{"one--two__three", []string{"one", "two", "three"}},
		{"UPPER case", []string{"upper", "case"}},
		{"", nil},
		{"!@#$%", nil},
	}

	for _, tt := range tests {
		got := tokenize(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("tokenize(%q) len = %d, want %d; got %v", tt.input, len(got), len(tt.want), got)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("tokenize(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}
