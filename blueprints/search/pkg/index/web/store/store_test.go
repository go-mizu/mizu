//go:build !chdb

package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/store"
)

func TestOpenAndClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.duckdb")

	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// Close twice — must not panic or error.
	if err := s.Close(); err != nil {
		t.Errorf("Close #1: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close #2 (idempotent): %v", err)
	}
}

func TestOpenEmptyPath(t *testing.T) {
	_, err := store.Open("")
	if err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
}

func TestPutGetSummary(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "meta.duckdb"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Not found before insert.
	_, ok, err := s.GetSummary(ctx, "CC-MAIN-2025-01")
	if err != nil {
		t.Fatalf("GetSummary (missing): %v", err)
	}
	if ok {
		t.Fatal("expected not-found, got ok=true")
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	rec := store.SummaryRecord{
		CrawlID:       "CC-MAIN-2025-01",
		WARCCount:     42,
		WARCTotalSize: 1024 * 1024,
		MDShards:      3,
		MDTotalSize:   512 * 1024,
		MDDocEstimate: 1000,
		PackFormats:   map[string]int64{"parquet": 200000},
		FTSEngines:    map[string]int64{"bleve": 300000},
		FTSShardCount: map[string]int64{"bleve": 3},
		GeneratedAt:   now,
		ScanDuration:  500 * time.Millisecond,
	}
	if err := s.PutSummary(ctx, rec); err != nil {
		t.Fatalf("PutSummary: %v", err)
	}

	got, ok, err := s.GetSummary(ctx, "CC-MAIN-2025-01")
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true after insert")
	}
	if got.WARCCount != 42 {
		t.Errorf("WARCCount: got %d, want 42", got.WARCCount)
	}
	if got.PackFormats["parquet"] != 200000 {
		t.Errorf("PackFormats[parquet]: got %d, want 200000", got.PackFormats["parquet"])
	}
	if got.FTSShardCount["bleve"] != 3 {
		t.Errorf("FTSShardCount[bleve]: got %d, want 3", got.FTSShardCount["bleve"])
	}
	if got.ScanDuration != 500*time.Millisecond {
		t.Errorf("ScanDuration: got %v, want 500ms", got.ScanDuration)
	}
}

func TestPutListJobs(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "meta.duckdb"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	jobs, err := s.ListJobs(ctx)
	if err != nil {
		t.Fatalf("ListJobs (empty): %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("expected 0 jobs, got %d", len(jobs))
	}

	now := time.Now().UTC().Truncate(time.Second)
	rec := store.JobRecord{
		ID:        "abc12345",
		Type:      "download",
		Status:    "completed",
		Config:    `{"type":"download"}`,
		Progress:  1.0,
		Message:   "done",
		Rate:      500,
		StartedAt: now,
	}
	if err := s.PutJob(ctx, rec); err != nil {
		t.Fatalf("PutJob: %v", err)
	}

	jobs, err = s.ListJobs(ctx)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].ID != "abc12345" {
		t.Errorf("ID: got %q, want abc12345", jobs[0].ID)
	}
	if jobs[0].Status != "completed" {
		t.Errorf("Status: got %q, want completed", jobs[0].Status)
	}

	// Upsert (replace).
	rec.Status = "failed"
	rec.Error = "something went wrong"
	if err := s.PutJob(ctx, rec); err != nil {
		t.Fatalf("PutJob (update): %v", err)
	}
	jobs, _ = s.ListJobs(ctx)
	if len(jobs) != 1 {
		t.Fatalf("expected still 1 job after upsert, got %d", len(jobs))
	}
	if jobs[0].Status != "failed" {
		t.Errorf("Status after update: got %q, want failed", jobs[0].Status)
	}

	if err := s.DeleteAllJobs(ctx); err != nil {
		t.Fatalf("DeleteAllJobs: %v", err)
	}
	jobs, _ = s.ListJobs(ctx)
	if len(jobs) != 0 {
		t.Fatalf("expected 0 jobs after delete, got %d", len(jobs))
	}
}

func TestSetGetRefreshState(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "meta.duckdb"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	_, ok, err := s.GetRefreshState(ctx, "CC-MAIN-2025-01")
	if err != nil {
		t.Fatalf("GetRefreshState (missing): %v", err)
	}
	if ok {
		t.Fatal("expected not-found")
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	st := store.RefreshState{
		CrawlID:    "CC-MAIN-2025-01",
		Status:     "refreshing",
		StartedAt:  &now,
		Generation: 1,
	}
	if err := s.SetRefreshState(ctx, st); err != nil {
		t.Fatalf("SetRefreshState: %v", err)
	}

	got, ok, err := s.GetRefreshState(ctx, "CC-MAIN-2025-01")
	if err != nil {
		t.Fatalf("GetRefreshState: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true after set")
	}
	if got.Status != "refreshing" {
		t.Errorf("Status: got %q, want refreshing", got.Status)
	}
	if got.Generation != 1 {
		t.Errorf("Generation: got %d, want 1", got.Generation)
	}
	if got.StartedAt == nil {
		t.Error("StartedAt: expected non-nil")
	}
}

func TestListGetWARCs(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "meta.duckdb"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	summary := store.SummaryRecord{
		CrawlID:     "CC-MAIN-2025-01",
		WARCCount:   2,
		GeneratedAt: now,
		WARCs: []store.WARCRecord{
			{
				CrawlID:       "CC-MAIN-2025-01",
				WARCIndex:     "00001",
				ManifestIndex: 1,
				Filename:      "crawl-data.warc.gz",
				WARCBytes:     1000,
				MarkdownDocs:  5,
				MarkdownBytes: 200,
				PackBytes:     map[string]int64{"parquet": 150},
				FTSBytes:      map[string]int64{"bleve": 100},
				TotalBytes:    1450,
				UpdatedAt:     now,
			},
			{
				CrawlID:       "CC-MAIN-2025-01",
				WARCIndex:     "00002",
				ManifestIndex: 2,
				Filename:      "crawl-data2.warc.gz",
				WARCBytes:     2000,
				PackBytes:     map[string]int64{},
				FTSBytes:      map[string]int64{},
				TotalBytes:    2000,
				UpdatedAt:     now,
			},
		},
	}
	if err := s.PutSummary(ctx, summary); err != nil {
		t.Fatalf("PutSummary: %v", err)
	}

	recs, err := s.ListWARCs(ctx, "CC-MAIN-2025-01")
	if err != nil {
		t.Fatalf("ListWARCs: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 WARC records, got %d", len(recs))
	}
	if recs[0].WARCIndex != "00001" {
		t.Errorf("WARCIndex[0]: got %q, want 00001", recs[0].WARCIndex)
	}
	if recs[0].PackBytes["parquet"] != 150 {
		t.Errorf("PackBytes[parquet]: got %d, want 150", recs[0].PackBytes["parquet"])
	}

	rec, ok, err := s.GetWARC(ctx, "CC-MAIN-2025-01", "00001")
	if err != nil {
		t.Fatalf("GetWARC: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for existing WARC")
	}
	if rec.FTSBytes["bleve"] != 100 {
		t.Errorf("FTSBytes[bleve]: got %d, want 100", rec.FTSBytes["bleve"])
	}

	_, ok, err = s.GetWARC(ctx, "CC-MAIN-2025-01", "99999")
	if err != nil {
		t.Fatalf("GetWARC (missing): %v", err)
	}
	if ok {
		t.Fatal("expected not-found for missing WARC")
	}
}
