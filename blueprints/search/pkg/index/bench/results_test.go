package bench_test

import (
	"encoding/json"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/bench"
)

func TestBenchResults_RoundTrip(t *testing.T) {
	r := &bench.BenchResults{
		Details: map[string][]bench.EngineDetails{
			"rose": {{Docs: 100, IndexTimeSec: 1.5, DiskMB: 12}},
		},
		Results: map[string]map[string][]bench.QueryResult{
			"TOP_10": {
				"rose": {
					{Query: "+foo +bar", Tags: []string{"intersection"}, Count: 3, Duration: []int{1000, 1100, 1200}},
				},
			},
		},
	}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got bench.BenchResults
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Details["rose"][0].Docs != 100 {
		t.Errorf("docs: got %d, want 100", got.Details["rose"][0].Docs)
	}
	if got.Results["TOP_10"]["rose"][0].Query != "+foo +bar" {
		t.Errorf("query mismatch")
	}
	if len(got.Results["TOP_10"]["rose"][0].Duration) != 3 {
		t.Errorf("duration len: got %d, want 3", len(got.Results["TOP_10"]["rose"][0].Duration))
	}
}

func TestSortDurations(t *testing.T) {
	qr := bench.QueryResult{Duration: []int{5000, 1000, 3000, 2000, 4000}}
	qr.SortDurations()
	for i := 1; i < len(qr.Duration); i++ {
		if qr.Duration[i] < qr.Duration[i-1] {
			t.Errorf("duration not sorted at index %d: %v", i, qr.Duration)
		}
	}
}

func TestNewBenchResults(t *testing.T) {
	r := bench.NewBenchResults()
	r.SetDetails("rose", bench.EngineDetails{Docs: 50, IndexTimeSec: 2.0, DiskMB: 10})
	r.AddQueryResults("TOP_10", "rose", []bench.QueryResult{
		{Query: "foo", Tags: []string{"union"}, Count: 5, Duration: []int{500}},
	})

	if r.Details["rose"][0].Docs != 50 {
		t.Errorf("SetDetails: docs mismatch")
	}
	if len(r.Results["TOP_10"]["rose"]) != 1 {
		t.Errorf("AddQueryResults: wrong count")
	}
}

func TestResultsPath(t *testing.T) {
	p := bench.ResultsPath("/tmp/bench")
	if p == "/tmp/bench" {
		t.Error("ResultsPath should include subdirectory and timestamp")
	}
	// Must contain "results/" directory
	if len(p) < len("/tmp/bench/results/") {
		t.Errorf("path too short: %s", p)
	}
}
