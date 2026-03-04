package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/bench"
)

func TestLatestCompareResultPath(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "correctness-a-vs-b-older.json")
	newPath := filepath.Join(dir, "correctness-a-vs-b-newer.json")

	if err := os.WriteFile(oldPath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write old: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write new: %v", err)
	}

	now := time.Now()
	if err := os.Chtimes(oldPath, now.Add(-2*time.Hour), now.Add(-2*time.Hour)); err != nil {
		t.Fatalf("chtime old: %v", err)
	}
	if err := os.Chtimes(newPath, now, now); err != nil {
		t.Fatalf("chtime new: %v", err)
	}

	got, err := latestCompareResultPath(dir)
	if err != nil {
		t.Fatalf("latestCompareResultPath: %v", err)
	}
	if got != newPath {
		t.Fatalf("latest path = %q, want %q", got, newPath)
	}
}

func TestBuildBenchReportMarkdown(t *testing.T) {
	comp := &bench.CompareResults{
		EngineA:     "dahlia",
		EngineB:     "tantivy",
		Limit:       10,
		GeneratedAt: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC),
		Summary: bench.CompareSummary{
			TotalQueries:            4,
			QueriesWithHitsEither:   4,
			QueriesWithHitsBoth:     4,
			ExactTopKAll:            1,
			ExactTopKWithHitsEither: 1,
			AvgOverlapAll:           6.5,
			AvgOverlapWithHits:      6.5,
			OverlapP50:              7,
			OverlapP90:              10,
			OverlapP99:              10,
			DifferentHitCount:       1,
		},
	}

	tagStats := map[string]*benchTagStats{
		"union":  {Count: 2, Exact: 1, SumOverlap: 11, DifferentCount: 1},
		"phrase": {Count: 2, Exact: 0, SumOverlap: 8, DifferentCount: 0},
	}
	tagKeys := []string{"union", "phrase"}
	mismatches := []benchMismatch{
		{
			Query:   "foo bar",
			Tags:    []string{"union"},
			Overlap: 3,
			CountA:  10,
			CountB:  9,
			A:       []string{"a1", "a2"},
			B:       []string{"b1", "b2"},
		},
	}

	md := buildBenchReportMarkdown("/tmp/correctness.json", comp, tagKeys, tagStats, mismatches, 10)
	mustContain := []string{
		"# Bench Compare Report",
		"Engines: `dahlia` vs `tantivy`",
		"| Queries | 4 |",
		"| `union` | 2 | 50.00%",
		"## Worst Mismatches",
		"`foo bar`",
		"- Overlap: `3`",
	}
	for _, m := range mustContain {
		if !strings.Contains(md, m) {
			t.Fatalf("markdown missing %q\n%s", m, md)
		}
	}
}
