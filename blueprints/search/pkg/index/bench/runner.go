package bench

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	benchdata "github.com/go-mizu/mizu/blueprints/search/data"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

// embeddedQueries is the embedded queries.jsonl content from the data package.
var embeddedQueries = benchdata.QueriesJSONL

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

// IterStats holds percentile stats for one query's timed iterations.
type IterStats struct {
	P50, P95, Min, Max time.Duration
}

// CalcIterStats computes min, max, p50 and p95 over a slice of durations.
// Returns zero IterStats for empty input.
// Exported for testing.
func CalcIterStats(ds []time.Duration) IterStats {
	if len(ds) == 0 {
		return IterStats{}
	}
	sorted := make([]time.Duration, len(ds))
	copy(sorted, ds)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	pct := func(p float64) time.Duration {
		idx := int(p / 100 * float64(len(sorted)-1))
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		return sorted[idx]
	}
	return IterStats{
		P50: pct(50),
		P95: pct(95),
		Min: sorted[0],
		Max: sorted[len(sorted)-1],
	}
}

// LoadQueries reads queries.jsonl from path (or embedded file if path == "").
func LoadQueries(path string) ([]BenchQuery, error) {
	var scanner *bufio.Scanner
	if path == "" {
		scanner = bufio.NewScanner(bytes.NewReader(embeddedQueries))
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open queries: %w", err)
		}
		defer f.Close()
		scanner = bufio.NewScanner(f)
	}
	scanner.Buffer(make([]byte, 64<<10), 64<<10)

	var queries []BenchQuery
	for scanner.Scan() {
		var q BenchQuery
		if err := json.Unmarshal(scanner.Bytes(), &q); err != nil {
			continue // skip malformed lines
		}
		if q.Query != "" {
			queries = append(queries, q)
		}
	}
	return queries, scanner.Err()
}

// CommandToQuery converts a command name and query text to an index.Query.
// Exported for testing.
func CommandToQuery(queryText, command string) index.Query {
	switch command {
	case "COUNT":
		return index.Query{Text: queryText, Limit: 1000}
	default: // TOP_10, TOP_10_COUNT, and unknown commands
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
// Warmup phase: runs all queries in a loop until cfg.Warmup expires (not timed).
// Timed phase: each query is run cfg.Iter times; latency stored as sorted microseconds.
// progress is called after each query's iterations (nil = disabled).
func Run(ctx context.Context, eng index.Engine, cfg BenchConfig, progress func(idx, total int, q string, s IterStats)) ([]QueryResult, error) {
	if cfg.Iter <= 0 {
		cfg.Iter = 10
	}

	total := len(cfg.Queries)
	results := make([]QueryResult, total)
	for i, bq := range cfg.Queries {
		results[i] = QueryResult{Query: bq.Query, Tags: bq.Tags}
	}

	// Warmup phase.
	if cfg.Warmup > 0 {
		deadline := time.Now().Add(cfg.Warmup)
		for time.Now().Before(deadline) {
			for _, bq := range cfg.Queries {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				q := CommandToQuery(bq.Query, cfg.Command)
				eng.Search(ctx, q) //nolint:errcheck
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
		sorted := make([]time.Duration, cfg.Iter)
		copy(sorted, durations)
		sort.Slice(sorted, func(a, b int) bool { return sorted[a] < sorted[b] })

		intDurs := make([]int, cfg.Iter)
		for j, d := range sorted {
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
