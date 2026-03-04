package bench

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

// CompareConfig controls a two-engine search-result comparison run.
type CompareConfig struct {
	Queries    []BenchQuery
	Limit      int
	MaxQueries int // 0 = all
}

// CompareProgress is emitted after each query comparison.
type CompareProgress struct {
	Done    int
	Total   int
	Query   string
	Exact   bool
	Overlap int
}

// CompareQueryResult captures one query's top-k comparison.
type CompareQueryResult struct {
	Query    string   `json:"query"`
	Tags     []string `json:"tags,omitempty"`
	A        []string `json:"a"`
	B        []string `json:"b"`
	CountA   int      `json:"count_a"`
	CountB   int      `json:"count_b"`
	Overlap  int      `json:"overlap"`
	Exact    bool     `json:"exact"`
	ExactSet bool     `json:"exact_set"`
}

// CompareSummary aggregates comparison statistics across all queries.
type CompareSummary struct {
	TotalQueries            int     `json:"total_queries"`
	QueriesWithHitsA        int     `json:"queries_with_hits_a"`
	QueriesWithHitsB        int     `json:"queries_with_hits_b"`
	QueriesWithHitsEither   int     `json:"queries_with_hits_either"`
	QueriesWithHitsBoth     int     `json:"queries_with_hits_both"`
	ExactTopKAll            int     `json:"exact_topk_all"`
	ExactTopKWithHitsEither int     `json:"exact_topk_with_hits_either"`
	ExactSetTopKAll         int     `json:"exact_set_topk_all"`
	ExactSetTopKWithHits    int     `json:"exact_set_topk_with_hits_either"`
	DifferentHitCount       int     `json:"different_hit_count"`
	AvgOverlapAll           float64 `json:"avg_overlap_all"`
	AvgOverlapWithHits      float64 `json:"avg_overlap_with_hits"`
	OverlapP50              int     `json:"overlap_p50"`
	OverlapP90              int     `json:"overlap_p90"`
	OverlapP99              int     `json:"overlap_p99"`
}

// CompareResults is the persisted comparison artifact.
type CompareResults struct {
	EngineA     string               `json:"engine_a"`
	EngineB     string               `json:"engine_b"`
	Limit       int                  `json:"limit"`
	GeneratedAt time.Time            `json:"generated_at"`
	Summary     CompareSummary       `json:"summary"`
	Queries     []CompareQueryResult `json:"queries"`
}

// Compare runs query-by-query top-k result comparison between two engines.
func Compare(
	ctx context.Context,
	engA, engB index.Engine,
	engineAName, engineBName string,
	cfg CompareConfig,
	progress func(CompareProgress),
) (*CompareResults, error) {
	limit := cfg.Limit
	if limit <= 0 {
		limit = 10
	}
	queries := cfg.Queries
	if cfg.MaxQueries > 0 && cfg.MaxQueries < len(queries) {
		queries = queries[:cfg.MaxQueries]
	}

	out := &CompareResults{
		EngineA:     engineAName,
		EngineB:     engineBName,
		Limit:       limit,
		GeneratedAt: time.Now().UTC(),
		Queries:     make([]CompareQueryResult, 0, len(queries)),
	}

	var (
		sumOverlapAll      int
		sumOverlapWithHits int
		overlapWithHits    []int
	)

	for i, bq := range queries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		q := index.Query{Text: bq.Query, Limit: limit}
		ra, err := engA.Search(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("search engine %q query %q: %w", engineAName, bq.Query, err)
		}
		rb, err := engB.Search(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("search engine %q query %q: %w", engineBName, bq.Query, err)
		}

		a := hitDocIDs(ra.Hits, limit)
		b := hitDocIDs(rb.Hits, limit)
		overlap := overlapCount(a, b)
		exact := sameDocIDs(a, b)
		exactSet := sameDocSet(a, b)

		hasA := len(a) > 0
		hasB := len(b) > 0
		hasEither := hasA || hasB
		hasBoth := hasA && hasB

		out.Summary.TotalQueries++
		if hasA {
			out.Summary.QueriesWithHitsA++
		}
		if hasB {
			out.Summary.QueriesWithHitsB++
		}
		if hasEither {
			out.Summary.QueriesWithHitsEither++
			sumOverlapWithHits += overlap
			overlapWithHits = append(overlapWithHits, overlap)
		}
		if hasBoth {
			out.Summary.QueriesWithHitsBoth++
		}
		if exact {
			out.Summary.ExactTopKAll++
			if hasEither {
				out.Summary.ExactTopKWithHitsEither++
			}
		}
		if exactSet {
			out.Summary.ExactSetTopKAll++
			if hasEither {
				out.Summary.ExactSetTopKWithHits++
			}
		}
		if len(a) != len(b) {
			out.Summary.DifferentHitCount++
		}
		sumOverlapAll += overlap

		out.Queries = append(out.Queries, CompareQueryResult{
			Query:   bq.Query,
			Tags:    bq.Tags,
			A:       a,
			B:       b,
			CountA:  len(a),
			CountB:  len(b),
			Overlap: overlap,
			Exact:   exact,
			ExactSet: exactSet,
		})

		if progress != nil {
			progress(CompareProgress{
				Done:    i + 1,
				Total:   len(queries),
				Query:   bq.Query,
				Exact:   exact,
				Overlap: overlap,
			})
		}
	}

	if out.Summary.TotalQueries > 0 {
		out.Summary.AvgOverlapAll = float64(sumOverlapAll) / float64(out.Summary.TotalQueries)
	}
	if out.Summary.QueriesWithHitsEither > 0 {
		out.Summary.AvgOverlapWithHits = float64(sumOverlapWithHits) / float64(out.Summary.QueriesWithHitsEither)
		sort.Ints(overlapWithHits)
		out.Summary.OverlapP50 = percentileInt(overlapWithHits, 50)
		out.Summary.OverlapP90 = percentileInt(overlapWithHits, 90)
		out.Summary.OverlapP99 = percentileInt(overlapWithHits, 99)
	}

	return out, nil
}

func hitDocIDs(hits []index.Hit, limit int) []string {
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]string, len(hits))
	for i := range hits {
		out[i] = hits[i].DocID
	}
	return out
}

func sameDocIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameDocSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]int, len(a))
	for _, id := range a {
		m[id]++
	}
	for _, id := range b {
		v := m[id]
		if v == 0 {
			return false
		}
		m[id] = v - 1
	}
	for _, v := range m {
		if v != 0 {
			return false
		}
	}
	return true
}

func overlapCount(a, b []string) int {
	set := make(map[string]struct{}, len(a))
	for _, id := range a {
		set[id] = struct{}{}
	}
	n := 0
	for _, id := range b {
		if _, ok := set[id]; ok {
			n++
		}
	}
	return n
}

func percentileInt(sorted []int, p int) int {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	idx := int(float64(len(sorted)-1) * float64(p) / 100.0)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
