package bench_test

import (
	"encoding/json"
	"strings"
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
	// Must contain the results/ subdirectory
	if !strings.Contains(p, "/results/") {
		t.Errorf("path missing /results/: %s", p)
	}
	// Must end in .json
	if !strings.HasSuffix(p, ".json") {
		t.Errorf("path must end in .json: %s", p)
	}
}

func TestLoadResults_RoundTrip(t *testing.T) {
	r := bench.NewBenchResults()
	r.SetDetails("rose", bench.EngineDetails{Docs: 42, IndexTimeSec: 1.0, DiskMB: 5})
	r.AddQueryResults("TOP_10", "rose", []bench.QueryResult{
		{Query: "foo", Tags: []string{"union"}, Count: 1, Duration: []int{100}},
	})

	tmpDir := t.TempDir()
	path := tmpDir + "/results/test.json"
	if err := bench.SaveResults(path, r); err != nil {
		t.Fatalf("SaveResults: %v", err)
	}

	got, err := bench.LoadResults(path)
	if err != nil {
		t.Fatalf("LoadResults: %v", err)
	}
	if got.Details["rose"][0].Docs != 42 {
		t.Errorf("docs: got %d, want 42", got.Details["rose"][0].Docs)
	}
	if len(got.Results["TOP_10"]["rose"]) != 1 {
		t.Errorf("results count: got %d, want 1", len(got.Results["TOP_10"]["rose"]))
	}
}
