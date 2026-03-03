# search bench Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement `search bench {download,index,search}` commands to benchmark FTS engines (rose, tantivy) against the standard Wikipedia corpus from quickwit-oss/search-benchmark-game.

**Architecture:** Three phases: (1) `bench download` streams wiki-articles.json.bz2 via HTTP, decompresses bzip2, normalizes text, writes `corpus.ndjson`; (2) `bench index` reads corpus.ndjson into any registered `pkg/index.Engine`; (3) `bench search` runs 962 standard queries with warmup + timed iterations, writing `results.json` in quickwit-oss-compatible format.

**Tech Stack:** Go stdlib `compress/bzip2`, `pkg/index.Engine` + `RunPipelineFromChannel`, `//go:embed`, Cobra, existing `formatBytes`/`DirSizeBytes` helpers.

---

## Context You Need

**Module:** `github.com/go-mizu/mizu/blueprints/search`

**Working directory:** `/Users/apple/github/go-mizu/mizu/blueprints/search`

**Key existing types (pkg/index):**
- `Engine` interface: `Open(ctx, dir)`, `Close()`, `Stats(ctx)`, `Index(ctx, []Document)`, `Search(ctx, Query) Results`
- `Document{DocID string, Text []byte}`, `Query{Text string, Limit int}`, `Results{Hits []Hit, Total int}`
- `PackProgressFunc = func(done, total int64, elapsed time.Duration)`
- `PipelineStats{DocsIndexed atomic.Int64, StartTime time.Time, PeakRSSMB atomic.Int64}`
- `RunPipelineFromChannel(ctx, engine, docCh <-chan Document, total int64, batchSize int, progress PackProgressFunc)`
- `DirSizeBytes(dir string) int64`
- `NewEngine(name string) (Engine, error)`, `AddrSetter` interface

**Key CLI helpers (package `cli`):**
- `formatBytes(b int64) string` — defined in `cli/fw2.go`
- All commands use `cobra.Command` + `fang`

**Tantivy note:** The `tantivy` driver requires build tag `tantivy`. Default builds have `devnull`, `sqlite`, `duckdb`, `bleve`, `rose`, and external HTTP engines. Tests should use `devnull` or `rose` (no build tags needed).

**bzip2:** Use stdlib `compress/bzip2` (read-only decompressor). It is single-threaded but sufficient for a one-time download.

**Corpus NDJSON keys:** The bench corpus uses `{"doc_id":"...","text":"..."}` — NOT the internal `{"i":"...","t":"..."}` short keys used by `RunPipelineFromNDJSON`. The bench index command reads corpus.ndjson with a dedicated scanner.

**Reference repo queries:** 962 lines at `/Users/apple/github/quickwit-oss/search-benchmark-game/queries.txt` — already NDJSON format, copy verbatim.

**Data directory:** `$HOME/data/search/bench/`

---

### Task 1: Create embedded query file

**Files:**
- Create: `data/queries.jsonl`

**Step 1: Copy queries from reference repo**

```bash
mkdir -p data
cp /Users/apple/github/quickwit-oss/search-benchmark-game/queries.txt data/queries.jsonl
```

**Step 2: Verify line count and format**

```bash
wc -l data/queries.jsonl
head -3 data/queries.jsonl
```

Expected output:
```
962 data/queries.jsonl
{"query": "+griffith +observatory", "tags": ["intersection", ...]}
...
```

**Step 3: Commit**

```bash
git add data/queries.jsonl
git commit -m "data(bench): add 962 Wikipedia benchmark queries from quickwit-oss"
```

---

### Task 2: `pkg/index/bench/results.go` — Result types and JSON

**Files:**
- Create: `pkg/index/bench/results.go`
- Create: `pkg/index/bench/results_test.go`

**Step 1: Write the failing test**

`pkg/index/bench/results_test.go`:
```go
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
```

**Step 2: Run test to verify it fails**

```bash
go test ./pkg/index/bench/... 2>&1 | head -5
```

Expected: compilation error — package doesn't exist yet.

**Step 3: Implement `results.go`**

`pkg/index/bench/results.go`:
```go
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

// AddQueryResults appends query results for (command, engine).
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

// ResultsPath returns the default timestamped output path.
func ResultsPath(dir string) string {
	ts := time.Now().Format("2006-01-02T15-04-05")
	return filepath.Join(dir, "results", ts+".json")
}
```

**Step 4: Run tests**

```bash
go test ./pkg/index/bench/... -v -run TestBenchResults
```

Expected: PASS

**Step 5: Commit**

```bash
git add pkg/index/bench/results.go pkg/index/bench/results_test.go
git commit -m "feat(bench): add BenchResults types and JSON serialization"
```

---

### Task 3: `pkg/index/bench/corpus.go` — Wikipedia download pipeline

**Files:**
- Create: `pkg/index/bench/corpus.go`
- Create: `pkg/index/bench/corpus_test.go`

**Step 1: Write the failing test**

`pkg/index/bench/corpus_test.go`:
```go
package bench_test

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/bench"
)

func TestNormalizeText(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Hello, World! 123", "hello world "},
		{"machine-learning", "machine learning"},
		{"New York City", "new york city"},
		{"café résumé", "caf r sum "},   // non-ASCII stripped
		{"", ""},
	}
	for _, tc := range cases {
		got := bench.NormalizeText(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeText(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestTransformWikiLine(t *testing.T) {
	// Valid doc
	line := `{"url":"https://en.wikipedia.org/wiki/Test","title":"Test","body":"Hello World!"}`
	doc, ok, err := bench.TransformWikiLine([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}
	if doc.DocID != "https://en.wikipedia.org/wiki/Test" {
		t.Errorf("docID: got %q", doc.DocID)
	}
	if !strings.Contains(doc.Text, "hello world") {
		t.Errorf("text not normalized: %q", doc.Text)
	}

	// Empty URL → skip
	emptyURL := `{"url":"","title":"T","body":"B"}`
	_, ok2, err2 := bench.TransformWikiLine([]byte(emptyURL))
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	if ok2 {
		t.Error("expected ok=false for empty url")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./pkg/index/bench/... -run TestNormalize 2>&1 | head -5
```

Expected: compilation error.

**Step 3: Implement `corpus.go`**

`pkg/index/bench/corpus.go`:
```go
package bench

import (
	"bufio"
	"compress/bzip2"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

const DefaultCorpusURL = "https://www.dropbox.com/s/wwnfnu441w1ec9p/wiki-articles.json.bz2?dl=1"

var nonAlphaRe = regexp.MustCompile(`[^a-zA-Z]+`)

// NormalizeText replaces non-alpha characters with spaces and lowercases the result.
// Exported for testing.
func NormalizeText(s string) string {
	return strings.ToLower(nonAlphaRe.ReplaceAllString(s, " "))
}

// wikiRaw is the raw JSON shape of one Wikipedia article line.
type wikiRaw struct {
	URL  string `json:"url"`
	Body string `json:"body"`
}

// corpusDoc is the normalized output shape written to corpus.ndjson.
type corpusDoc struct {
	DocID string `json:"doc_id"`
	Text  string `json:"text"`
}

// TransformWikiLine parses one raw Wikipedia NDJSON line and returns a normalized
// corpusDoc. ok=false means the line should be skipped (empty URL or parse error
// that is not a hard failure).
// Exported for testing.
func TransformWikiLine(line []byte) (corpusDoc, bool, error) {
	var raw wikiRaw
	if err := json.Unmarshal(line, &raw); err != nil {
		return corpusDoc{}, false, nil // skip malformed lines silently
	}
	if raw.URL == "" {
		return corpusDoc{}, false, nil
	}
	return corpusDoc{DocID: raw.URL, Text: NormalizeText(raw.Body)}, true, nil
}

// DownloadConfig controls the corpus download.
type DownloadConfig struct {
	URL     string // default: DefaultCorpusURL
	OutPath string // absolute path for corpus.ndjson
	MaxDocs int64  // 0 = unlimited
	Force   bool   // overwrite existing file
}

// DownloadStats tracks live download progress.
type DownloadStats struct {
	BytesDownloaded atomic.Int64 // compressed bytes received
	BytesWritten    atomic.Int64 // bytes written to corpus.ndjson
	DocsWritten     atomic.Int64
	StartTime       time.Time
	TotalBytes      int64 // from Content-Length (0 if unknown)
}

// countingReader wraps an io.Reader and counts bytes read into stats.
type countingReader struct {
	r     io.Reader
	stats *DownloadStats
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	cr.stats.BytesDownloaded.Add(int64(n))
	return n, err
}

// Download streams the Wikipedia bzip2 corpus, normalizes it, and writes corpus.ndjson.
// progress is called every 200 ms; pass nil to disable.
func Download(ctx context.Context, cfg DownloadConfig, progress func(*DownloadStats)) (*DownloadStats, error) {
	if cfg.URL == "" {
		cfg.URL = DefaultCorpusURL
	}

	// Check existing file.
	if !cfg.Force {
		if _, err := os.Stat(cfg.OutPath); err == nil {
			return nil, fmt.Errorf("corpus already exists at %s (use --force to overwrite)", cfg.OutPath)
		}
	}

	if err := os.MkdirAll(filepath.Dir(cfg.OutPath), 0o755); err != nil {
		return nil, err
	}

	// HTTP GET with context.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d from %s", resp.StatusCode, cfg.URL)
	}

	stats := &DownloadStats{
		StartTime:  time.Now(),
		TotalBytes: resp.ContentLength,
	}

	// Progress ticker.
	if progress != nil {
		ticker := time.NewTicker(200 * time.Millisecond)
		stopTicker := make(chan struct{})
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					progress(stats)
				case <-stopTicker:
					return
				}
			}
		}()
		defer close(stopTicker)
	}

	// Pipeline: HTTP body → counting reader → bzip2 → line scanner → transform → NDJSON writer.
	cr := &countingReader{r: resp.Body, stats: stats}
	bzr := bzip2.NewReader(cr)

	outFile, err := os.Create(cfg.OutPath)
	if err != nil {
		return nil, fmt.Errorf("create corpus: %w", err)
	}
	bw := bufio.NewWriterSize(outFile, 4<<20) // 4 MB write buffer
	enc := json.NewEncoder(bw)
	enc.SetEscapeHTML(false)

	scanner := bufio.NewScanner(bzr)
	scanner.Buffer(make([]byte, 4<<20), 4<<20) // 4 MB line buffer (some Wikipedia articles are large)

	for scanner.Scan() {
		if ctx.Err() != nil {
			bw.Flush()
			outFile.Close()
			os.Remove(cfg.OutPath)
			return stats, ctx.Err()
		}
		doc, ok, err := TransformWikiLine(scanner.Bytes())
		if err != nil || !ok {
			continue
		}
		if err := enc.Encode(doc); err != nil {
			bw.Flush()
			outFile.Close()
			os.Remove(cfg.OutPath)
			return stats, fmt.Errorf("encode: %w", err)
		}
		n := stats.DocsWritten.Add(1)
		stats.BytesWritten.Add(int64(len(doc.Text) + len(doc.DocID) + 20))
		if cfg.MaxDocs > 0 && n >= cfg.MaxDocs {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		bw.Flush()
		outFile.Close()
		os.Remove(cfg.OutPath)
		return stats, fmt.Errorf("scan: %w", err)
	}

	if err := bw.Flush(); err != nil {
		outFile.Close()
		return stats, err
	}
	return stats, outFile.Close()
}

// CorpusReader reads corpus.ndjson and emits index.Document via a channel.
// Respects ctx cancellation. Closes docCh when done.
// total is the number of docs in the corpus (0 = unknown).
func CorpusReader(ctx context.Context, corpusPath string, maxDocs int64, docCh chan<- index.Document) (total int64, err error) {
	f, err := os.Open(corpusPath)
	if err != nil {
		close(docCh)
		return 0, fmt.Errorf("open corpus: %w", err)
	}
	go func() {
		defer f.Close()
		defer close(docCh)
		br := bufio.NewReaderSize(f, 4<<20)
		var doc corpusDoc
		for {
			if ctx.Err() != nil {
				return
			}
			line, err := br.ReadBytes('\n')
			if len(line) > 0 {
				line = line[:len(line)-1] // trim newline
				if json.Unmarshal(line, &doc) == nil && doc.DocID != "" {
					select {
					case docCh <- index.Document{DocID: doc.DocID, Text: []byte(doc.Text)}:
						total++
						if maxDocs > 0 && total >= maxDocs {
							return
						}
					case <-ctx.Done():
						return
					}
				}
			}
			if err == io.EOF {
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return 0, nil // total is filled async; caller uses PipelineStats
}
```

**Step 4: Run tests**

```bash
go test ./pkg/index/bench/... -v -run "TestNormalize|TestTransform" 2>&1
```

Expected: PASS

**Step 5: Commit**

```bash
git add pkg/index/bench/corpus.go pkg/index/bench/corpus_test.go
git commit -m "feat(bench): add corpus download pipeline with bzip2 streaming and text normalization"
```

---

### Task 4: `pkg/index/bench/runner.go` — Query benchmark runner

**Files:**
- Create: `pkg/index/bench/runner.go`
- Create: `pkg/index/bench/runner_test.go`

**Step 1: Write the failing test**

`pkg/index/bench/runner_test.go`:
```go
package bench_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/devnull"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/bench"
)

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
		for i := 1; i < len(r.Duration); i++ {
			if r.Duration[i] < r.Duration[i-1] {
				t.Errorf("durations not sorted for %q", r.Query)
			}
		}
	}
}

func TestLoadQueries_Embedded(t *testing.T) {
	queries, err := bench.LoadQueries("")
	if err != nil {
		t.Fatalf("LoadQueries: %v", err)
	}
	if len(queries) < 900 {
		t.Errorf("expected ≥900 queries, got %d", len(queries))
	}
}

func TestCommandToQuery(t *testing.T) {
	cases := []struct{
		command string
		wantLimit int
	}{
		{"TOP_10", 10},
		{"COUNT", 1000},
		{"TOP_10_COUNT", 10},
	}
	for _, tc := range cases {
		q := bench.CommandToQuery("machine learning", tc.command)
		if q.Limit != tc.wantLimit {
			t.Errorf("command %s: limit got %d want %d", tc.command, q.Limit, tc.wantLimit)
		}
	}
}

func TestIterStats_Percentiles(t *testing.T) {
	durations := []time.Duration{1*time.Millisecond, 2*time.Millisecond, 3*time.Millisecond,
		4*time.Millisecond, 5*time.Millisecond, 6*time.Millisecond, 7*time.Millisecond,
		8*time.Millisecond, 9*time.Millisecond, 10*time.Millisecond}
	s := bench.CalcIterStats(durations)
	if s.Min != 1*time.Millisecond {
		t.Errorf("min: got %v", s.Min)
	}
	if s.Max != 10*time.Millisecond {
		t.Errorf("max: got %v", s.Max)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./pkg/index/bench/... -run TestRun 2>&1 | head -5
```

Expected: compilation error.

**Step 3: Implement `runner.go`**

`pkg/index/bench/runner.go`:
```go
package bench

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

//go:embed ../../data/queries.jsonl
var embeddedQueries []byte

// BenchQuery is one entry from queries.jsonl.
type BenchQuery struct {
	Query string   `json:"query"`
	Tags  []string `json:"tags"`
}

// BenchConfig controls a benchmark run for one command.
type BenchConfig struct {
	Command string        // "TOP_10" | "COUNT" | "TOP_10_COUNT"
	Queries []BenchQuery  // parsed from queries.jsonl
	Iter    int           // timing iterations per query (default 10)
	Warmup  time.Duration // warmup duration before timing (default 30s)
}

// IterStats holds percentile stats for one query's iterations.
type IterStats struct {
	P50, P95, Min, Max time.Duration
}

// CalcIterStats computes stats over a slice of durations (need not be sorted).
// Exported for testing.
func CalcIterStats(ds []time.Duration) IterStats {
	if len(ds) == 0 {
		return IterStats{}
	}
	sorted := make([]time.Duration, len(ds))
	copy(sorted, ds)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	p := func(pct float64) time.Duration {
		idx := int(pct / 100 * float64(len(sorted)-1))
		return sorted[idx]
	}
	return IterStats{
		P50: p(50),
		P95: p(95),
		Min: sorted[0],
		Max: sorted[len(sorted)-1],
	}
}

// LoadQueries reads queries.jsonl from path (or embedded if path == "").
func LoadQueries(path string) ([]BenchQuery, error) {
	var r *bufio.Scanner
	if path == "" {
		r = bufio.NewScanner(bytes.NewReader(embeddedQueries))
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open queries: %w", err)
		}
		defer f.Close()
		r = bufio.NewScanner(f)
	}
	r.Buffer(make([]byte, 64<<10), 64<<10)

	var queries []BenchQuery
	for r.Scan() {
		var q BenchQuery
		if err := json.Unmarshal(r.Bytes(), &q); err != nil {
			continue
		}
		if q.Query != "" {
			queries = append(queries, q)
		}
	}
	return queries, r.Err()
}

// CommandToQuery converts a command name and query string to an index.Query.
// Exported for testing.
func CommandToQuery(queryText, command string) index.Query {
	switch command {
	case "COUNT":
		return index.Query{Text: queryText, Limit: 1000}
	case "TOP_10_COUNT":
		return index.Query{Text: queryText, Limit: 10}
	default: // TOP_10
		return index.Query{Text: queryText, Limit: 10}
	}
}

// extractCount extracts the result count from Results based on command.
func extractCount(res index.Results, command string) int {
	switch command {
	case "COUNT", "TOP_10_COUNT":
		if res.Total > 0 {
			return res.Total
		}
		return len(res.Hits)
	default:
		return len(res.Hits)
	}
}

// Run executes the benchmark for cfg.Command across all cfg.Queries.
// After warmup, each query is run cfg.Iter times and latency is measured.
// progress is called after each query completes its iterations (nil = disabled).
func Run(ctx context.Context, eng index.Engine, cfg BenchConfig, progress func(idx, total int, q string, s IterStats)) ([]QueryResult, error) {
	if cfg.Iter <= 0 {
		cfg.Iter = 10
	}

	total := len(cfg.Queries)
	results := make([]QueryResult, total)
	for i, bq := range cfg.Queries {
		results[i] = QueryResult{Query: bq.Query, Tags: bq.Tags}
	}

	// Warmup phase: run all queries in a loop until warmup duration expires.
	if cfg.Warmup > 0 {
		deadline := time.Now().Add(cfg.Warmup)
		for time.Now().Before(deadline) {
			for _, bq := range cfg.Queries {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				q := CommandToQuery(bq.Query, cfg.Command)
				eng.Search(ctx, q) //nolint:errcheck // warmup, ignore errors
				if time.Now().After(deadline) {
					break
				}
			}
		}
	}

	// Timed phase.
	durations := make([]time.Duration, cfg.Iter)
	for i, bq := range cfg.Queries {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		q := CommandToQuery(bq.Query, cfg.Command)

		var lastCount int
		for iter := 0; iter < cfg.Iter; iter++ {
			t0 := time.Now()
			res, err := eng.Search(ctx, q)
			elapsed := time.Since(t0)
			if err != nil {
				return nil, fmt.Errorf("search %q: %w", bq.Query, err)
			}
			durations[iter] = elapsed
			lastCount = extractCount(res, cfg.Command)
		}

		// Sort durations and convert to microseconds.
		sortedDurs := make([]time.Duration, cfg.Iter)
		copy(sortedDurs, durations)
		sort.Slice(sortedDurs, func(a, b int) bool { return sortedDurs[a] < sortedDurs[b] })

		intDurs := make([]int, cfg.Iter)
		for j, d := range sortedDurs {
			intDurs[j] = int(d.Microseconds())
		}

		results[i].Count = lastCount
		results[i].Duration = intDurs

		if progress != nil {
			progress(i+1, total, bq.Query, CalcIterStats(durations))
		}
	}

	return results, nil
}
```

**Step 4: Run tests**

```bash
go test ./pkg/index/bench/... -v 2>&1
```

Expected: all PASS. Note: `TestLoadQueries_Embedded` requires `data/queries.jsonl` exists (Task 1).

**Step 5: Commit**

```bash
git add pkg/index/bench/runner.go pkg/index/bench/runner_test.go
git commit -m "feat(bench): add BenchRunner with warmup loop, timed iterations, and percentile stats"
```

---

### Task 5: `cli/bench.go` — three subcommands

**Files:**
- Create: `cli/bench.go`

**No unit test for CLI layer** — tested by smoke test in Task 7.

**Step 1: Create `cli/bench.go`**

```go
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/devnull"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/sqlite"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/bleve"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/flower/rose"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/bench"
	"github.com/spf13/cobra"
)

func NewBench() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bench",
		Short: "Benchmark FTS engines on the Wikipedia corpus",
		Long:  `Download the Wikipedia corpus and benchmark indexing and search performance.`,
		Example: `  search bench download --docs 100000
  search bench index --engine rose --docs 100000
  search bench search --engine rose --commands TOP_10 --iter 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newBenchDownload())
	cmd.AddCommand(newBenchIndex())
	cmd.AddCommand(newBenchSearch())
	return cmd
}

func defaultBenchDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "search", "bench")
}

// ── bench download ────────────────────────────────────────────────────────────

func newBenchDownload() *cobra.Command {
	var (
		url   string
		dir   string
		docs  int64
		force bool
	)
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download and preprocess Wikipedia corpus to corpus.ndjson",
		Example: `  search bench download
  search bench download --docs 100000
  search bench download --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBenchDownload(cmd.Context(), url, dir, docs, force)
		},
	}
	cmd.Flags().StringVar(&url, "url", bench.DefaultCorpusURL, "Source URL (bz2)")
	cmd.Flags().StringVar(&dir, "dir", defaultBenchDir(), "Bench data directory")
	cmd.Flags().Int64Var(&docs, "docs", 0, "Stop after N docs (0 = all)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing corpus.ndjson")
	return cmd
}

func runBenchDownload(ctx context.Context, url, dir string, maxDocs int64, force bool) error {
	outPath := filepath.Join(dir, "corpus.ndjson")

	cfg := bench.DownloadConfig{
		URL:     url,
		OutPath: outPath,
		MaxDocs: maxDocs,
		Force:   force,
	}

	fmt.Fprintf(os.Stderr, "downloading Wikipedia corpus → %s\n", outPath)

	progress := func(s *bench.DownloadStats) {
		elapsed := time.Since(s.StartTime).Seconds()
		dlMB := float64(s.BytesDownloaded.Load()) / 1e6
		totalMB := float64(s.TotalBytes) / 1e6
		docs := s.DocsWritten.Load()
		writtenMB := float64(s.BytesWritten.Load()) / 1e6
		dlSpeed := dlMB / elapsed
		docRate := float64(docs) / elapsed

		bar := progressBar(dlMB, totalMB, 20)
		eta := ""
		if s.TotalBytes > 0 && dlSpeed > 0 {
			remainSec := (totalMB - dlMB) / dlSpeed
			eta = fmt.Sprintf("  │  eta %s", fmtDuration(time.Duration(remainSec)*time.Second))
		}
		fmt.Fprintf(os.Stderr, "\r\033[Kdownloading  %s  %.1f/%.1f GB  │  %.1f MB/s  │  %d docs  │  %.0f docs/s  │  %.1f MB written%s",
			bar, dlMB/1000, totalMB/1000, dlSpeed, docs, docRate, writtenMB, eta)
	}

	stats, err := bench.Download(ctx, cfg, progress)
	fmt.Fprintln(os.Stderr) // newline after progress
	if err != nil {
		return err
	}

	elapsed := time.Since(stats.StartTime)
	fi, _ := os.Stat(outPath)
	var corpusSize int64
	if fi != nil {
		corpusSize = fi.Size()
	}

	fmt.Fprintf(os.Stderr, "\n── bench download complete ──────────────────────────────\n")
	fmt.Fprintf(os.Stderr, "  docs:          %d\n", stats.DocsWritten.Load())
	fmt.Fprintf(os.Stderr, "  corpus size:   %s\n", formatBytes(corpusSize))
	fmt.Fprintf(os.Stderr, "  elapsed:       %s\n", elapsed.Round(time.Second))
	fmt.Fprintf(os.Stderr, "  avg dl speed:  %.1f MB/s\n", float64(stats.BytesDownloaded.Load())/1e6/elapsed.Seconds())
	fmt.Fprintf(os.Stderr, "  avg doc rate:  %.0f docs/s\n", float64(stats.DocsWritten.Load())/elapsed.Seconds())
	fmt.Fprintf(os.Stderr, "  path:          %s\n", outPath)
	return nil
}

// ── bench index ──────────────────────────────────────────────────────────────

func newBenchIndex() *cobra.Command {
	var (
		dir       string
		engineName string
		docs      int64
		batchSize int
		workers   int
		addr      string
	)
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Index the Wikipedia corpus with a registered FTS engine",
		Example: `  search bench index --engine rose
  search bench index --engine devnull --docs 10000
  search bench index --engine tantivy --docs 200000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBenchIndex(cmd.Context(), dir, engineName, docs, batchSize, workers, addr)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", defaultBenchDir(), "Bench data directory")
	cmd.Flags().StringVar(&engineName, "engine", "rose", "FTS engine: "+strings.Join(index.List(), ", "))
	cmd.Flags().Int64Var(&docs, "docs", 0, "Index first N docs (0 = all)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "Documents per batch")
	cmd.Flags().IntVar(&workers, "workers", 0, "Parallel indexing workers (0 = NumCPU)")
	cmd.Flags().StringVar(&addr, "addr", "", "Service address for external engines")
	cmd.MarkFlagRequired("engine")
	return cmd
}

func runBenchIndex(ctx context.Context, dir, engineName string, maxDocs int64, batchSize, workers int, addr string) error {
	corpusPath := filepath.Join(dir, "corpus.ndjson")
	if _, err := os.Stat(corpusPath); os.IsNotExist(err) {
		return fmt.Errorf("corpus not found at %s — run 'search bench download' first", corpusPath)
	}

	indexDir := filepath.Join(dir, "index", engineName)
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return err
	}

	eng, err := index.NewEngine(engineName)
	if err != nil {
		return err
	}
	if addr != "" {
		if setter, ok := eng.(index.AddrSetter); ok {
			setter.SetAddr(addr)
		} else {
			fmt.Fprintf(os.Stderr, "warning: engine %q does not support --addr\n", engineName)
		}
	}
	if err := eng.Open(ctx, indexDir); err != nil {
		return fmt.Errorf("open engine: %w", err)
	}
	defer eng.Close()

	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if batchSize <= 0 {
		batchSize = 5000
	}

	// Count total docs for progress bar (best-effort, non-blocking).
	var totalDocs int64
	// We don't pre-scan; RunPipelineFromChannel accepts total=0 (unknown).

	docCh := make(chan index.Document, batchSize*2)
	bench.CorpusReader(ctx, corpusPath, maxDocs, docCh) //nolint:errcheck

	progress := func(done, total int64, elapsed time.Duration) {
		secs := elapsed.Seconds()
		rate := float64(0)
		if secs > 0 {
			rate = float64(done) / secs
		}
		disk := index.DirSizeBytes(indexDir)
		bar := progressBar(float64(done), float64(total), 20)
		_ = totalDocs
		if total > 0 {
			fmt.Fprintf(os.Stderr, "\r\033[Kbench index [%s]  %s  %d/%d docs  │  %.0f docs/s  │  %.1fs  │  disk %s",
				engineName, bar, done, total, rate, secs, formatBytes(disk))
		} else {
			fmt.Fprintf(os.Stderr, "\r\033[Kbench index [%s]  %d docs  │  %.0f docs/s  │  %.1fs  │  disk %s",
				engineName, done, rate, secs, formatBytes(disk))
		}
	}

	t0 := time.Now()
	pstats, err := index.RunPipelineFromChannel(ctx, eng, docCh, 0, batchSize, progress)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}

	engStats, _ := eng.Stats(ctx)
	elapsed := time.Since(t0)
	avgRate := float64(pstats.DocsIndexed.Load()) / elapsed.Seconds()

	fmt.Fprintf(os.Stderr, "\n── bench index complete ─────────────────────────────────\n")
	fmt.Fprintf(os.Stderr, "  engine:        %s\n", engineName)
	fmt.Fprintf(os.Stderr, "  docs:          %d\n", engStats.DocCount)
	fmt.Fprintf(os.Stderr, "  elapsed:       %s\n", elapsed.Round(100*time.Millisecond))
	fmt.Fprintf(os.Stderr, "  avg rate:      %.0f docs/s\n", avgRate)
	fmt.Fprintf(os.Stderr, "  peak RSS:      %d MB\n", pstats.PeakRSSMB.Load())
	fmt.Fprintf(os.Stderr, "  disk:          %s\n", formatBytes(engStats.DiskBytes))
	fmt.Fprintf(os.Stderr, "  path:          %s\n", indexDir)
	return nil
}

// ── bench search ─────────────────────────────────────────────────────────────

func newBenchSearch() *cobra.Command {
	var (
		dir        string
		engineName string
		queriesFile string
		commands   string
		iter       int
		warmup     time.Duration
		outputFile string
		addr       string
	)
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Run benchmark queries against an indexed FTS engine",
		Example: `  search bench search --engine rose
  search bench search --engine rose --commands TOP_10,COUNT --iter 5 --warmup 10s
  search bench search --engine devnull --warmup 0s --iter 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmds := strings.Split(commands, ",")
			for i := range cmds {
				cmds[i] = strings.TrimSpace(strings.ToUpper(cmds[i]))
			}
			return runBenchSearch(cmd.Context(), dir, engineName, queriesFile, cmds, iter, warmup, outputFile, addr)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", defaultBenchDir(), "Bench data directory")
	cmd.Flags().StringVar(&engineName, "engine", "rose", "FTS engine")
	cmd.Flags().StringVar(&queriesFile, "queries", "", "Queries file (default: embedded)")
	cmd.Flags().StringVar(&commands, "commands", "TOP_10", "Commands: TOP_10,COUNT,TOP_10_COUNT")
	cmd.Flags().IntVar(&iter, "iter", 10, "Iterations per query")
	cmd.Flags().DurationVar(&warmup, "warmup", 30*time.Second, "Warmup duration")
	cmd.Flags().StringVar(&outputFile, "output", "", "Output JSON path (default: {dir}/results/{ts}.json)")
	cmd.Flags().StringVar(&addr, "addr", "", "Service address for external engines")
	cmd.MarkFlagRequired("engine")
	return cmd
}

func runBenchSearch(ctx context.Context, dir, engineName, queriesFile string, commands []string, iter int, warmup time.Duration, outputFile, addr string) error {
	indexDir := filepath.Join(dir, "index", engineName)
	if _, err := os.Stat(indexDir); os.IsNotExist(err) {
		return fmt.Errorf("no index for %q at %s — run 'bench index --engine %s' first", engineName, indexDir, engineName)
	}

	queries, err := bench.LoadQueries(queriesFile)
	if err != nil {
		return fmt.Errorf("load queries: %w", err)
	}
	if len(queries) == 0 {
		return fmt.Errorf("no queries loaded")
	}

	eng, err := index.NewEngine(engineName)
	if err != nil {
		return err
	}
	if addr != "" {
		if setter, ok := eng.(index.AddrSetter); ok {
			setter.SetAddr(addr)
		}
	}
	if err := eng.Open(ctx, indexDir); err != nil {
		return fmt.Errorf("open engine: %w", err)
	}
	defer eng.Close()

	results := bench.NewBenchResults()

	// Record index details.
	engStats, _ := eng.Stats(ctx)
	fi, _ := os.Stat(indexDir)
	diskMB := index.DirSizeBytes(indexDir) / (1 << 20)
	_ = fi
	results.SetDetails(engineName, bench.EngineDetails{
		Docs:   engStats.DocCount,
		DiskMB: diskMB,
	})

	for _, command := range commands {
		cfg := bench.BenchConfig{
			Command: command,
			Queries: queries,
			Iter:    iter,
			Warmup:  warmup,
		}

		fmt.Fprintf(os.Stderr, "\nbench search [%s / %s] — %d queries, %d iter, warmup %s\n",
			engineName, command, len(queries), iter, warmup)
		if warmup > 0 {
			fmt.Fprintf(os.Stderr, "  warming up...\n")
		}

		var slowestQuery string
		var slowestP50 time.Duration
		var fastestQuery string
		fastestP50 := time.Duration(1<<62)

		progress := func(idx, total int, q string, s bench.IterStats) {
			pct := float64(idx) / float64(total)
			bar := progressBar(float64(idx), float64(total), 20)
			fmt.Fprintf(os.Stderr, "\r\033[Kbench search [%s / %s]  q %d/%d %q  │  p50=%s  p95=%s  min=%s  max=%s  │  %s  %.0f%%",
				engineName, command, idx, total, truncate(q, 30),
				fmtDuration(s.P50), fmtDuration(s.P95), fmtDuration(s.Min), fmtDuration(s.Max),
				bar, pct*100)

			if s.P50 > slowestP50 {
				slowestP50 = s.P50
				slowestQuery = q
			}
			if s.P50 < fastestP50 {
				fastestP50 = s.P50
				fastestQuery = q
			}
		}

		t0 := time.Now()
		qrs, err := bench.Run(ctx, eng, cfg, progress)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("bench %s: %w", command, err)
		}
		elapsed := time.Since(t0)
		results.AddQueryResults(command, engineName, qrs)

		// Compute aggregate stats from all query results.
		var allP50, allP95, allP99 []time.Duration
		for _, qr := range qrs {
			if len(qr.Duration) == 0 {
				continue
			}
			allP50 = append(allP50, time.Duration(qr.Duration[len(qr.Duration)/2])*time.Microsecond)
			allP95 = append(allP95, time.Duration(qr.Duration[int(float64(len(qr.Duration))*0.95)])*time.Microsecond)
			allP99 = append(allP99, time.Duration(qr.Duration[int(float64(len(qr.Duration))*0.99)])*time.Microsecond)
		}

		fmt.Fprintf(os.Stderr, "\n── bench search [%s / %s] ─────────────────────────\n", engineName, command)
		fmt.Fprintf(os.Stderr, "  queries:       %d\n", len(qrs))
		fmt.Fprintf(os.Stderr, "  iterations:    %d  (after %s warmup)\n", iter, warmup)
		fmt.Fprintf(os.Stderr, "  elapsed:       %s\n", elapsed.Round(100*time.Millisecond))
		if len(allP50) > 0 {
			fmt.Fprintf(os.Stderr, "  median p50:    %s\n", fmtDuration(medianDuration(allP50)))
			fmt.Fprintf(os.Stderr, "  median p95:    %s\n", fmtDuration(medianDuration(allP95)))
			fmt.Fprintf(os.Stderr, "  median p99:    %s\n", fmtDuration(medianDuration(allP99)))
		}
		if slowestQuery != "" {
			fmt.Fprintf(os.Stderr, "  slowest:       %q → %s\n", slowestQuery, fmtDuration(slowestP50))
		}
		if fastestQuery != "" && fastestP50 < time.Duration(1<<62) {
			fmt.Fprintf(os.Stderr, "  fastest:       %q → %s\n", fastestQuery, fmtDuration(fastestP50))
		}
	}

	// Write results.json
	if outputFile == "" {
		outputFile = bench.ResultsPath(dir)
	}
	if err := bench.SaveResults(outputFile, results); err != nil {
		return fmt.Errorf("save results: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  results:       %s\n", outputFile)
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

// progressBar returns a Unicode block progress bar of width w for value/total.
func progressBar(value, total float64, w int) string {
	if total <= 0 || value >= total {
		return strings.Repeat("█", w)
	}
	filled := int(value / total * float64(w))
	if filled < 0 {
		filled = 0
	}
	if filled > w {
		filled = w
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", w-filled)
}

// fmtDuration formats d as human-readable: "1.2ms", "34µs", "5.1s".
func fmtDuration(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
	default:
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
}

// fmtDurationFull formats a duration as "1m23s".
func fmtDurationFull(d time.Duration) string {
	return d.Round(time.Second).String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func medianDuration(ds []time.Duration) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(ds))
	copy(sorted, ds)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	return sorted[len(sorted)/2]
}
```

**Step 2: Commit**

```bash
git add cli/bench.go
git commit -m "feat(bench/cli): add bench download/index/search subcommands with live progress and summaries"
```

---

### Task 6: Wire `NewBench()` into `cli/root.go`

**Files:**
- Modify: `cli/root.go`

**Step 1: Add `NewBench()` to root command**

In `cli/root.go`, find the block of `root.AddCommand(...)` calls (around line 67–83) and add:

```go
root.AddCommand(NewBench())
```

Place it after `root.AddCommand(NewLocal())` or at the end of the block.

**Step 2: Verify build**

```bash
go build ./... 2>&1
```

Expected: clean build (no errors). If there are `fmtDurationFull` unused errors, remove the unused function.

**Step 3: Commit**

```bash
git add cli/root.go
git commit -m "feat(bench): wire search bench command into CLI root"
```

---

### Task 7: Smoke test end-to-end

**Step 1: Build binary**

```bash
go build -o /tmp/search-bench-smoke ./cmd/search/
```

**Step 2: Test download with --docs 500**

```bash
/tmp/search-bench-smoke bench download --docs 500 --force 2>&1 | tail -10
```

Expected:
```
── bench download complete ──────────────────────────────
  docs:          500
  ...
  path:          ~/data/search/bench/corpus.ndjson
```

**Step 3: Test index with devnull (fast, no real storage)**

```bash
/tmp/search-bench-smoke bench index --engine devnull --docs 500 2>&1 | tail -10
```

Expected:
```
── bench index complete ─────────────────────────────────
  engine:        devnull
  docs:          0
  ...
```

**Step 4: Test index with rose**

```bash
/tmp/search-bench-smoke bench index --engine rose --docs 500 2>&1 | tail -10
```

Expected:
```
── bench index complete ─────────────────────────────────
  engine:        rose
  docs:          500
  ...
```

**Step 5: Test search with rose, no warmup, 1 iteration**

```bash
/tmp/search-bench-smoke bench search --engine rose --warmup 0s --iter 1 2>&1 | tail -15
```

Expected: summary with timing stats and `results.json` path.

**Step 6: Verify results.json format**

```bash
cat ~/data/search/bench/results/$(ls -t ~/data/search/bench/results/ | head -1) | python3 -m json.tool | head -20
```

Expected: valid JSON matching quickwit-oss format.

**Step 7: Commit**

```bash
git add .
git commit -m "test(bench): smoke-tested end-to-end download/index/search pipeline"
```

---

### Task 8: Run unit tests

**Step 1: Run all bench package tests**

```bash
go test ./pkg/index/bench/... -v 2>&1
```

Expected: all PASS.

**Step 2: Run full test suite**

```bash
go test ./pkg/index/... 2>&1 | tail -20
```

Expected: no regressions.

**Step 3: Commit if any fixes needed**

```bash
git add -p
git commit -m "fix(bench): address test failures"
```

---

### Task 9: Write spec/0649 benchmark results table (after real run)

After running with `--docs 173720` (the CC-MAIN-2026-08 size), fill in the tables in
`spec/0649_bench_index.md`.

**Step 1: Run rose index benchmark**

```bash
/tmp/search-bench-smoke bench index --engine rose --docs 173720 2>&1 | tee /tmp/bench-rose-index.txt
```

**Step 2: Run rose search benchmark**

```bash
/tmp/search-bench-smoke bench search --engine rose --commands TOP_10,COUNT,TOP_10_COUNT --iter 10 --warmup 30s 2>&1 | tee /tmp/bench-rose-search.txt
```

**Step 3: Update spec/0649 with measured results**

Add a "Benchmark Results" section to `spec/0649_bench_index.md` with the actual numbers.

**Step 4: Commit**

```bash
git add spec/0649_bench_index.md
git commit -m "docs(bench): add measured benchmark results to spec/0649"
```
