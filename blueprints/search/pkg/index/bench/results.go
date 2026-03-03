package bench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// EngineDetails records index-time metadata for one engine run.
type EngineDetails struct {
	Docs         int64   `json:"docs"`
	IndexTimeSec float64 `json:"index_time_s"`
	DiskMB       int64   `json:"disk_mb"`
}

// QueryResult holds timing data for one (engine, command, query) triple.
type QueryResult struct {
	Query    string   `json:"query"`
	Tags     []string `json:"tags"`
	Count    int      `json:"count"`
	Duration []int    `json:"duration"` // sorted ascending microseconds
}

// SortDurations sorts Duration ascending in-place.
func (q *QueryResult) SortDurations() {
	sort.Ints(q.Duration)
}

// BenchResults is the top-level results.json structure.
// Outer key of Results: command ("TOP_10", "COUNT", "TOP_10_COUNT").
// Inner key: engine name.
type BenchResults struct {
	Details map[string][]EngineDetails          `json:"details"`
	Results map[string]map[string][]QueryResult `json:"results"`
}

// NewBenchResults allocates an empty BenchResults.
func NewBenchResults() *BenchResults {
	return &BenchResults{
		Details: make(map[string][]EngineDetails),
		Results: make(map[string]map[string][]QueryResult),
	}
}

// SetDetails records engine details for the given engine name.
func (b *BenchResults) SetDetails(engine string, d EngineDetails) {
	b.Details[engine] = []EngineDetails{d}
}

// AddQueryResults sets query results for (command, engine), replacing any prior value.
func (b *BenchResults) AddQueryResults(command, engine string, qrs []QueryResult) {
	if b.Results[command] == nil {
		b.Results[command] = make(map[string][]QueryResult)
	}
	b.Results[command][engine] = qrs
}

// SaveResults writes BenchResults as indented JSON to path.
// Parent directories are created if needed.
func SaveResults(path string, r *BenchResults) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadResults reads a BenchResults JSON file from path.
func LoadResults(path string) (*BenchResults, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r BenchResults
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ResultsPath returns the default timestamped output path under dir/results/.
func ResultsPath(dir string) string {
	ts := time.Now().Format("2006-01-02T15-04-05")
	return filepath.Join(dir, "results", ts+".json")
}
