package bench

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

type fakeSearchEngine struct {
	out map[string][]string
}

func (f *fakeSearchEngine) Name() string { return "fake" }
func (f *fakeSearchEngine) Open(context.Context, string) error {
	return nil
}
func (f *fakeSearchEngine) Close() error { return nil }
func (f *fakeSearchEngine) Stats(context.Context) (index.EngineStats, error) {
	return index.EngineStats{}, nil
}
func (f *fakeSearchEngine) Index(context.Context, []index.Document) error { return nil }
func (f *fakeSearchEngine) Search(_ context.Context, q index.Query) (index.Results, error) {
	ids := f.out[q.Text]
	hits := make([]index.Hit, len(ids))
	for i, id := range ids {
		hits[i] = index.Hit{DocID: id}
	}
	return index.Results{Hits: hits, Total: len(hits)}, nil
}

func TestCompare(t *testing.T) {
	a := &fakeSearchEngine{
		out: map[string][]string{
			"q1": {"a", "b"},
			"q2": {"x"},
			"q3": {},
		},
	}
	b := &fakeSearchEngine{
		out: map[string][]string{
			"q1": {"a", "c"},
			"q2": {"x"},
			"q3": {},
		},
	}

	res, err := Compare(context.Background(), a, b, "a", "b", CompareConfig{
		Queries: []BenchQuery{
			{Query: "q1"},
			{Query: "q2"},
			{Query: "q3"},
		},
		Limit: 10,
	}, nil)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}

	if res.Summary.TotalQueries != 3 {
		t.Fatalf("TotalQueries = %d, want 3", res.Summary.TotalQueries)
	}
	if res.Summary.ExactTopKAll != 2 {
		t.Fatalf("ExactTopKAll = %d, want 2", res.Summary.ExactTopKAll)
	}
	if res.Summary.ExactSetTopKAll != 2 {
		t.Fatalf("ExactSetTopKAll = %d, want 2", res.Summary.ExactSetTopKAll)
	}
	if res.Summary.QueriesWithHitsEither != 2 {
		t.Fatalf("QueriesWithHitsEither = %d, want 2", res.Summary.QueriesWithHitsEither)
	}
	if res.Summary.DifferentHitCount != 0 {
		t.Fatalf("DifferentHitCount = %d, want 0", res.Summary.DifferentHitCount)
	}
	if got := res.Queries[0].Overlap; got != 1 {
		t.Fatalf("q1 overlap = %d, want 1", got)
	}
}

func TestPercentileInt(t *testing.T) {
	in := []int{1, 2, 3, 4, 5}
	if got := percentileInt(in, 50); got != 3 {
		t.Fatalf("p50 = %d, want 3", got)
	}
	if got := percentileInt(in, 90); got != 4 {
		t.Fatalf("p90 = %d, want 4", got)
	}
}
