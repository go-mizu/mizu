package memory

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// SearchResult represents a single result from hybrid search.
type SearchResult struct {
	Path      string  // File path relative to workspace.
	StartLine int     // 1-based start line.
	EndLine   int     // 1-based end line (inclusive).
	Score     float64 // Combined score in [0,1].
	Snippet   string  // The matching text snippet.
	Source    string  // Origin source identifier.
}

// VectorResult is a raw result from vector similarity search.
type VectorResult struct {
	Path      string
	StartLine int
	EndLine   int
	Score     float64 // Cosine similarity, already in [0,1].
	Snippet   string
	Source    string
}

// KeywordResult is a raw result from FTS5 BM25 search.
type KeywordResult struct {
	Path      string
	StartLine int
	EndLine   int
	Rank      float64 // Raw BM25 rank (lower = better match in SQLite FTS5).
	Snippet   string
	Source    string
}

// HybridConfig controls the hybrid search scoring.
type HybridConfig struct {
	VectorWeight        float64 // Weight for vector similarity (default 0.7).
	TextWeight          float64 // Weight for BM25 keyword match (default 0.3).
	MinScore            float64 // Minimum combined score to include (default 0.35).
	MaxResults          int     // Maximum results returned (default 6).
	CandidateMultiplier int     // Fetch CandidateMultiplier * MaxResults candidates (default 4).
}

// DefaultHybridConfig returns the default hybrid search parameters.
func DefaultHybridConfig() HybridConfig {
	return HybridConfig{
		VectorWeight:        0.7,
		TextWeight:          0.3,
		MinScore:            0.35,
		MaxResults:          6,
		CandidateMultiplier: 4,
	}
}

// BM25RankToScore converts a SQLite FTS5 BM25 rank value to a 0-1 score.
// FTS5 BM25 ranks are negative (more negative = better match). We take the
// absolute value and apply 1/(1+|rank|) so that a perfect match approaches 1.
func BM25RankToScore(rank float64) float64 {
	absRank := math.Abs(rank)
	return 1.0 / (1.0 + absRank)
}

// BuildFTSQuery tokenizes the raw text and joins tokens with AND for use
// with SQLite FTS5 MATCH syntax. Non-alphanumeric characters are stripped,
// and each remaining token is quoted to avoid FTS5 syntax issues.
func BuildFTSQuery(raw string) string {
	words := tokenize(raw)
	if len(words) == 0 {
		return ""
	}

	quoted := make([]string, len(words))
	for i, w := range words {
		// Quote each token to prevent FTS5 operator interpretation.
		quoted[i] = "\"" + w + "\""
	}
	return strings.Join(quoted, " AND ")
}

// MergeHybridResults combines vector and keyword results using weighted
// scoring. Results are de-duplicated by (path, startLine) key. The final
// list is sorted by descending score and trimmed to MaxResults.
func MergeHybridResults(vector []VectorResult, keyword []KeywordResult, cfg HybridConfig) []SearchResult {
	type key struct {
		path      string
		startLine int
	}

	type candidate struct {
		result      SearchResult
		vectorScore float64
		textScore   float64
		hasVector   bool
		hasText     bool
	}

	merged := make(map[key]*candidate)

	// Accumulate vector results.
	for _, v := range vector {
		k := key{path: v.Path, startLine: v.StartLine}
		c, ok := merged[k]
		if !ok {
			c = &candidate{
				result: SearchResult{
					Path:      v.Path,
					StartLine: v.StartLine,
					EndLine:   v.EndLine,
					Snippet:   v.Snippet,
					Source:    v.Source,
				},
			}
			merged[k] = c
		}
		c.vectorScore = v.Score
		c.hasVector = true
		// Prefer longer snippet if available.
		if len(v.Snippet) > len(c.result.Snippet) {
			c.result.Snippet = v.Snippet
		}
	}

	// Accumulate keyword results.
	for _, kw := range keyword {
		k := key{path: kw.Path, startLine: kw.StartLine}
		c, ok := merged[k]
		if !ok {
			c = &candidate{
				result: SearchResult{
					Path:      kw.Path,
					StartLine: kw.StartLine,
					EndLine:   kw.EndLine,
					Snippet:   kw.Snippet,
					Source:    kw.Source,
				},
			}
			merged[k] = c
		}
		c.textScore = BM25RankToScore(kw.Rank)
		c.hasText = true
		if len(kw.Snippet) > len(c.result.Snippet) {
			c.result.Snippet = kw.Snippet
		}
	}

	// Score and filter.
	results := make([]SearchResult, 0, len(merged))
	for _, c := range merged {
		score := 0.0
		if c.hasVector {
			score += cfg.VectorWeight * c.vectorScore
		}
		if c.hasText {
			score += cfg.TextWeight * c.textScore
		}

		// Boost results that appear in both sources.
		if c.hasVector && c.hasText {
			score = math.Min(score*1.1, 1.0)
		}

		if score < cfg.MinScore {
			continue
		}

		c.result.Score = score
		results = append(results, c.result)
	}

	// Sort by descending score, then by path+startLine for stability.
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].Path != results[j].Path {
			return results[i].Path < results[j].Path
		}
		return results[i].StartLine < results[j].StartLine
	})

	if len(results) > cfg.MaxResults {
		results = results[:cfg.MaxResults]
	}

	return results
}

// tokenize splits text into lowercase alphanumeric tokens.
func tokenize(text string) []string {
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(unicode.ToLower(r))
		} else {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}
