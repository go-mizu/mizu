package metastore_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/metastore"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/web/metastore/drivers/duckdb"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/web/metastore/drivers/sqlite"
)

func TestSQLiteStore_RoundTrip(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "meta.sqlite")
	store, err := metastore.Open("sqlite", dbPath, metastore.Options{BusyTimeout: 5 * time.Second, JournalMode: "WAL"})
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer store.Close()
	runStoreRoundTrip(t, store)
}

func TestDuckDBStore_RoundTrip(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "meta.duckdb")
	store, err := metastore.Open("duckdb", dbPath, metastore.Options{})
	if err != nil {
		t.Skipf("duckdb store unavailable in this build/runtime: %v", err)
	}
	defer store.Close()
	runStoreRoundTrip(t, store)
}

func runStoreRoundTrip(t *testing.T, store metastore.Store) {
	t.Helper()
	ctx := context.Background()
	if err := store.Init(ctx); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unavailable") {
			t.Skipf("store unavailable: %v", err)
		}
		t.Fatalf("Init: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	rec := metastore.SummaryRecord{
		CrawlID:       "CC-MAIN-2026-08",
		WARCCount:     3,
		WARCTotalSize: 1024,
		MDShards:      2,
		MDTotalSize:   2048,
		MDDocEstimate: 99,
		PackFormats: map[string]int64{
			"parquet": 1234,
			"bin":     456,
		},
		FTSEngines: map[string]int64{
			"duckdb": 777,
			"sqlite": 888,
		},
		FTSShardCount: map[string]int64{
			"duckdb": 2,
			"sqlite": 1,
		},
		WARCs: []metastore.WARCRecord{
			{
				CrawlID:       "CC-MAIN-2026-08",
				WARCIndex:     "00000",
				ManifestIndex: 0,
				Filename:      "CC-MAIN-x-00000.warc.gz",
				RemotePath:    "crawl-data/.../00000.warc.gz",
				WARCBytes:     400,
				MarkdownDocs:  22,
				MarkdownBytes: 120,
				PackBytes: map[string]int64{
					"parquet": 80,
				},
				FTSBytes: map[string]int64{
					"duckdb": 64,
				},
				TotalBytes: 664,
				UpdatedAt:  now,
			},
			{
				CrawlID:       "CC-MAIN-2026-08",
				WARCIndex:     "00001",
				ManifestIndex: 1,
				Filename:      "CC-MAIN-x-00001.warc.gz",
				RemotePath:    "crawl-data/.../00001.warc.gz",
				WARCBytes:     200,
				MarkdownDocs:  11,
				MarkdownBytes: 55,
				PackBytes: map[string]int64{
					"bin": 44,
				},
				FTSBytes: map[string]int64{
					"sqlite": 33,
				},
				TotalBytes: 332,
				UpdatedAt:  now,
			},
		},
		GeneratedAt:  now,
		ScanDuration: 1500 * time.Millisecond,
	}
	if err := store.PutSummary(ctx, rec); err != nil {
		t.Fatalf("PutSummary: %v", err)
	}

	got, ok, err := store.GetSummary(ctx, rec.CrawlID)
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if !ok {
		t.Fatal("GetSummary ok=false, want true")
	}
	if got.CrawlID != rec.CrawlID || got.MDDocEstimate != rec.MDDocEstimate || got.WARCCount != rec.WARCCount {
		t.Fatalf("GetSummary mismatch: got=%+v want=%+v", got, rec)
	}
	if got.PackFormats["parquet"] != 1234 || got.FTSEngines["sqlite"] != 888 || got.FTSShardCount["duckdb"] != 2 {
		t.Fatalf("GetSummary map mismatch: got=%+v", got)
	}

	warcs, err := store.ListWARCs(ctx, rec.CrawlID)
	if err != nil {
		t.Fatalf("ListWARCs: %v", err)
	}
	if len(warcs) != 2 {
		t.Fatalf("ListWARCs len=%d want=2", len(warcs))
	}
	if warcs[0].WARCIndex != "00000" || warcs[1].WARCIndex != "00001" {
		t.Fatalf("ListWARCs order mismatch: %+v", warcs)
	}
	if warcs[0].PackBytes["parquet"] != 80 || warcs[1].FTSBytes["sqlite"] != 33 {
		t.Fatalf("ListWARCs phase maps mismatch: %+v", warcs)
	}

	w0, ok, err := store.GetWARC(ctx, rec.CrawlID, "00000")
	if err != nil {
		t.Fatalf("GetWARC: %v", err)
	}
	if !ok {
		t.Fatal("GetWARC ok=false, want true")
	}
	if w0.Filename == "" || w0.WARCBytes != 400 || w0.MarkdownDocs != 22 {
		t.Fatalf("GetWARC mismatch: %+v", w0)
	}

	started := now.Add(-2 * time.Second)
	finished := now
	st := metastore.RefreshState{
		CrawlID:    rec.CrawlID,
		Status:     "idle",
		StartedAt:  &started,
		FinishedAt: &finished,
		LastError:  "",
		Generation: 2,
	}
	if err := store.SetRefreshState(ctx, st); err != nil {
		t.Fatalf("SetRefreshState: %v", err)
	}

	gotSt, ok, err := store.GetRefreshState(ctx, rec.CrawlID)
	if err != nil {
		t.Fatalf("GetRefreshState: %v", err)
	}
	if !ok {
		t.Fatal("GetRefreshState ok=false, want true")
	}
	if gotSt.Status != "idle" || gotSt.Generation != 2 {
		t.Fatalf("GetRefreshState mismatch: got=%+v", gotSt)
	}
	if gotSt.StartedAt == nil || gotSt.FinishedAt == nil {
		t.Fatalf("GetRefreshState expected timestamps, got=%+v", gotSt)
	}
}
