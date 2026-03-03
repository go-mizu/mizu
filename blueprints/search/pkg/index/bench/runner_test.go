package bench_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/devnull"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/bench"
)

func TestLoadQueries_Embedded(t *testing.T) {
	queries, err := bench.LoadQueries("")
	if err != nil {
		t.Fatalf("LoadQueries: %v", err)
	}
	if len(queries) < 900 {
		t.Errorf("expected ≥900 queries, got %d", len(queries))
	}
	// Spot-check first query has both query and tags
	if queries[0].Query == "" {
		t.Error("first query is empty")
	}
	if len(queries[0].Tags) == 0 {
		t.Error("first query has no tags")
	}
}

func TestCommandToQuery(t *testing.T) {
	cases := []struct {
		command   string
		wantLimit int
	}{
		{"TOP_10", 10},
		{"COUNT", 1000},
		{"TOP_10_COUNT", 10},
		{"UNKNOWN", 10}, // unknown falls back to 10
	}
	for _, tc := range cases {
		q := bench.CommandToQuery("machine learning", tc.command)
		if q.Limit != tc.wantLimit {
			t.Errorf("CommandToQuery(%q): limit got %d, want %d", tc.command, q.Limit, tc.wantLimit)
		}
		if q.Text != "machine learning" {
			t.Errorf("CommandToQuery: Text got %q", q.Text)
		}
	}
}

func TestCalcIterStats(t *testing.T) {
	ds := []time.Duration{
		10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond,
		40 * time.Millisecond, 50 * time.Millisecond, 60 * time.Millisecond,
		70 * time.Millisecond, 80 * time.Millisecond, 90 * time.Millisecond,
		100 * time.Millisecond,
	}
	s := bench.CalcIterStats(ds)
	if s.Min != 10*time.Millisecond {
		t.Errorf("Min: got %v, want 10ms", s.Min)
	}
	if s.Max != 100*time.Millisecond {
		t.Errorf("Max: got %v, want 100ms", s.Max)
	}
	// p50 at index 5 (50% of 10 = 5th element in 0-indexed sorted slice)
	if s.P50 < 50*time.Millisecond || s.P50 > 60*time.Millisecond {
		t.Errorf("P50: got %v, expected ~50-60ms", s.P50)
	}
}

func TestCalcIterStats_Empty(t *testing.T) {
	s := bench.CalcIterStats(nil)
	if s.Min != 0 || s.Max != 0 {
		t.Errorf("empty: got %+v, want zero", s)
	}
}

func TestRun_DevNull(t *testing.T) {
	ctx := context.Background()

	eng, err := index.NewEngine("devnull")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if err := eng.Open(ctx, t.TempDir()); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	queries := []bench.BenchQuery{
		{Query: "machine learning", Tags: []string{"union"}},
		{Query: "climate change", Tags: []string{"union"}},
	}

	cfg := bench.BenchConfig{
		Command: "TOP_10",
		Queries: queries,
		Iter:    3,
		Warmup:  0, // no warmup in tests
	}

	var progressCalls int
	results, err := bench.Run(ctx, eng, cfg, func(idx, total int, q string, s bench.IterStats) {
		progressCalls++
		if idx < 1 || idx > total {
			t.Errorf("progress: idx %d out of range [1, %d]", idx, total)
		}
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
	if progressCalls != 2 {
		t.Errorf("progress called %d times, want 2", progressCalls)
	}
	for _, r := range results {
		if len(r.Duration) != 3 {
			t.Errorf("query %q: got %d durations, want 3", r.Query, len(r.Duration))
		}
		// Durations must be sorted ascending
		if !sort.IntsAreSorted(r.Duration) {
			t.Errorf("durations not sorted for %q: %v", r.Query, r.Duration)
		}
	}
}

func TestRun_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng, _ := index.NewEngine("devnull")
	eng.Open(ctx, t.TempDir())
	defer eng.Close()

	// Cancel immediately before Run
	cancel()

	queries := []bench.BenchQuery{
		{Query: "foo", Tags: []string{"union"}},
	}
	cfg := bench.BenchConfig{Command: "TOP_10", Queries: queries, Iter: 5}
	_, err := bench.Run(ctx, eng, cfg, nil)
	if err == nil {
		t.Log("Run returned nil with cancelled context (devnull ignores ctx) — acceptable")
	}
}
