package hn

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestHeadParquet(t *testing.T) {
	parquetBytes := []byte("parquet-bytes")
	ts := newHNTestServer(t, parquetBytes, nil)
	defer ts.Close()

	cfg := Config{DataDir: t.TempDir(), ParquetURL: ts.URL + "/items.parquet"}
	info, err := cfg.HeadParquet(context.Background())
	if err != nil {
		t.Fatalf("HeadParquet error: %v", err)
	}
	if info.Size != int64(len(parquetBytes)) {
		t.Fatalf("HeadParquet size=%d want %d", info.Size, len(parquetBytes))
	}
	if !info.AcceptRanges {
		t.Fatalf("HeadParquet AcceptRanges=false, want true")
	}
	if info.ETag == "" {
		t.Fatalf("HeadParquet missing ETag")
	}
}

func TestBuildClickHouseChunkRangesAligned(t *testing.T) {
	got := buildClickHouseChunkRanges(47499908, 47500065, 500000, true)
	want := [][2]int64{
		{47499908, 47500000},
		{47500001, 47500065},
	}
	if len(got) != len(want) {
		t.Fatalf("len(ranges)=%d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("range[%d]=%v want %v (all=%v)", i, got[i], want[i], got)
		}
	}
}

func TestDownloadParquetResume(t *testing.T) {
	parquetBytes := []byte(strings.Repeat("abc123XYZ", 1024))
	ts := newHNTestServer(t, parquetBytes, nil)
	defer ts.Close()

	cfg := Config{DataDir: t.TempDir(), ParquetURL: ts.URL + "/items.parquet"}
	if err := cfg.EnsureRawDirs(); err != nil {
		t.Fatalf("EnsureRawDirs: %v", err)
	}
	partial := parquetBytes[:len(parquetBytes)/3]
	if err := os.WriteFile(cfg.RawParquetPath(), partial, 0o644); err != nil {
		t.Fatalf("write partial parquet: %v", err)
	}

	res, err := cfg.DownloadParquet(context.Background(), false, nil)
	if err != nil {
		t.Fatalf("DownloadParquet resume error: %v", err)
	}
	if !res.Resumed {
		t.Fatalf("DownloadParquet Resumed=false, want true")
	}
	got, err := os.ReadFile(cfg.RawParquetPath())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, parquetBytes) {
		t.Fatalf("downloaded bytes mismatch")
	}

	res2, err := cfg.DownloadParquet(context.Background(), false, nil)
	if err != nil {
		t.Fatalf("DownloadParquet second run error: %v", err)
	}
	if !res2.Skipped {
		t.Fatalf("second run Skipped=false, want true")
	}
}

func TestDownloadAPIChunksResume(t *testing.T) {
	items := map[int64]string{
		1: `{"id":1,"type":"story","time":1700000000,"by":"a","title":"one"}`,
		2: `{"id":2,"type":"comment","time":1700000001,"by":"b","parent":1,"text":"two"}`,
		3: `null`,
		4: `{"id":4,"type":"job","time":1700000002,"by":"c","title":"job"}`,
		5: `{"id":5,"type":"story","time":1700000003,"by":"d","title":"five"}`,
	}
	ts := newHNTestServer(t, nil, items)
	defer ts.Close()

	cfg := Config{DataDir: t.TempDir(), APIBaseURL: ts.URL + "/v0"}
	res, err := cfg.DownloadAPI(context.Background(), APIDownloadOptions{
		FromID:    1,
		ToID:      5,
		ChunkSize: 2,
		Workers:   3,
	}, nil)
	if err != nil {
		t.Fatalf("DownloadAPI error: %v", err)
	}
	if res.ChunksTotal != 3 {
		t.Fatalf("ChunksTotal=%d want 3", res.ChunksTotal)
	}
	if res.ChunksDone != 3 {
		t.Fatalf("ChunksDone=%d want 3", res.ChunksDone)
	}
	if res.ItemsWritten != 4 {
		t.Fatalf("ItemsWritten=%d want 4 (one null item skipped)", res.ItemsWritten)
	}

	files, err := sortedGlob(filepath.Join(cfg.APIChunksDir(), "*.jsonl"))
	if err != nil {
		t.Fatalf("glob chunks: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("chunk file count=%d want 3", len(files))
	}

	res2, err := cfg.DownloadAPI(context.Background(), APIDownloadOptions{
		FromID:    1,
		ToID:      5,
		ChunkSize: 2,
		Workers:   2,
	}, nil)
	if err != nil {
		t.Fatalf("DownloadAPI second run error: %v", err)
	}
	if res2.ChunksSkipped != 3 {
		t.Fatalf("ChunksSkipped=%d want 3", res2.ChunksSkipped)
	}
}

func TestImportParquet(t *testing.T) {
	cfg := Config{DataDir: t.TempDir()}
	if err := cfg.EnsureRawDirs(); err != nil {
		t.Fatalf("EnsureRawDirs: %v", err)
	}
	createTestParquet(t, cfg.RawParquetPath())

	res, err := cfg.Import(context.Background(), ImportOptions{Source: ImportSourceParquet})
	if err != nil {
		t.Fatalf("Import parquet error: %v", err)
	}
	if res.Rows != 2 {
		t.Fatalf("rows=%d want 2", res.Rows)
	}

	db, err := sql.Open("duckdb", res.DBPath+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()
	var timeTSCount int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM items WHERE time_ts IS NOT NULL`).Scan(&timeTSCount); err != nil {
		t.Fatalf("query time_ts: %v", err)
	}
	if timeTSCount != 2 {
		t.Fatalf("time_ts non-null count=%d want 2", timeTSCount)
	}
}

func TestImportAPIChunks(t *testing.T) {
	cfg := Config{DataDir: t.TempDir()}
	if err := cfg.EnsureRawDirs(); err != nil {
		t.Fatalf("EnsureRawDirs: %v", err)
	}
	chunkPath := filepath.Join(cfg.APIChunksDir(), chunkFileName(1, 2))
	lines := strings.Join([]string{
		`{"id":1,"type":"story","time":1700000000,"by":"a","title":"one"}`,
		`{"id":2,"type":"comment","time":1700000001,"by":"b","parent":1,"text":"two"}`,
	}, "\n") + "\n"
	if err := os.WriteFile(chunkPath, []byte(lines), 0o644); err != nil {
		t.Fatalf("write chunk: %v", err)
	}

	res, err := cfg.Import(context.Background(), ImportOptions{Source: ImportSourceAPI})
	if err != nil {
		t.Fatalf("Import API error: %v", err)
	}
	if res.Rows != 2 {
		t.Fatalf("rows=%d want 2", res.Rows)
	}

	st, err := cfg.LocalStatus(context.Background())
	if err != nil {
		t.Fatalf("LocalStatus: %v", err)
	}
	if !st.DBExists || st.DBRows != 2 {
		t.Fatalf("LocalStatus DB exists=%v rows=%d; want true,2", st.DBExists, st.DBRows)
	}
}

func TestImportHybridIncremental(t *testing.T) {
	cfg := Config{DataDir: t.TempDir()}
	if err := cfg.EnsureRawDirs(); err != nil {
		t.Fatalf("EnsureRawDirs: %v", err)
	}

	ch1 := filepath.Join(cfg.ClickHouseParquetDir(), "id_000000001_000000002.parquet")
	createTestClickHouseParquet(t, ch1, []hnCHRow{
		{ID: 1, TypeCode: 1, By: "alice", Time: 1700000000, Title: "s1"},
		{ID: 2, TypeCode: 2, By: "bob", Time: 1700000001, Parent: sqlNullInt64(1), Text: "c2"},
	})
	createTestClickHouseParquet(t, filepath.Join(cfg.ClickHouseDeltaParquetDir(), "id_000000003_000000003.parquet"), []hnCHRow{
		{ID: 3, TypeCode: 1, By: "carol", Time: 1700000002, Title: "s3-delta-v1"},
	})

	res1, err := cfg.Import(context.Background(), ImportOptions{Source: ImportSourceAuto})
	if err != nil {
		t.Fatalf("first Import hybrid error: %v", err)
	}
	if res1.Mode != "full" {
		t.Fatalf("first import mode=%q want full", res1.Mode)
	}
	if res1.Rows != 3 {
		t.Fatalf("first import rows=%d want 3", res1.Rows)
	}

	ch2 := filepath.Join(cfg.ClickHouseParquetDir(), "id_000000003_000000004.parquet")
	createTestClickHouseParquet(t, ch2, []hnCHRow{
		{ID: 3, TypeCode: 1, By: "carol", Time: 1700000002, Title: "s3-ch-v2"},
		{ID: 4, TypeCode: 1, By: "dave", Time: 1700000003, Title: "s4"},
	})
	if err := os.Remove(filepath.Join(cfg.ClickHouseDeltaParquetDir(), "id_000000003_000000003.parquet")); err != nil {
		t.Fatalf("remove old delta chunk: %v", err)
	}
	createTestClickHouseParquet(t, filepath.Join(cfg.ClickHouseDeltaParquetDir(), "id_000000003_000000005.parquet"), []hnCHRow{
		{ID: 3, TypeCode: 1, By: "carol", Time: 1700000002, Title: "s3-delta-v2"},
		{ID: 5, TypeCode: 5, By: "erin", Time: 1700000004, Title: "j5"},
	})
	if err := cfg.WriteDownloadState(&DownloadState{
		SourceUsed: "hybrid",
		ClickHouse: &ClickHouseRunState{StartID: 1, EndID: 4, ChunkIDSpan: 2, IncrementalFromID: 3},
		Delta:      &ClickHouseRunState{StartID: 3, EndID: 5, ChunkIDSpan: 2, IncrementalFromID: 3},
	}); err != nil {
		t.Fatalf("WriteDownloadState: %v", err)
	}

	res2, err := cfg.Import(context.Background(), ImportOptions{Source: ImportSourceAuto})
	if err != nil {
		t.Fatalf("second Import hybrid incremental error: %v", err)
	}
	if res2.Mode != "incremental" {
		t.Fatalf("second import mode=%q want incremental", res2.Mode)
	}
	if res2.RowsBefore != 3 || res2.Rows != 5 || res2.RowsDelta != 2 {
		t.Fatalf("rows before/after/delta = %d/%d/%d want 3/5/2", res2.RowsBefore, res2.Rows, res2.RowsDelta)
	}
	if res2.ImportFromID != 3 {
		t.Fatalf("ImportFromID=%d want 3", res2.ImportFromID)
	}

	db, err := sql.Open("duckdb", res2.DBPath+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()
	var title3 string
	if err := db.QueryRow(`SELECT title FROM items WHERE id=3`).Scan(&title3); err != nil {
		t.Fatalf("query id=3 title: %v", err)
	}
	if title3 != "s3-delta-v2" {
		t.Fatalf("id=3 title=%q want delta overlay update", title3)
	}
}

func TestClickHouseChunkTailHelpers(t *testing.T) {
	if got := tailRefreshStartID(1, 1_250_000, 500_000, 2); got != 500_001 {
		t.Fatalf("tailRefreshStartID=%d want 500001", got)
	}
	if end, ok := expectedCHChunkEnd(1_000_001, 1, 1_250_000, 500_000); !ok || end != 1_250_000 {
		t.Fatalf("expectedCHChunkEnd tail = (%d,%v) want (1250000,true)", end, ok)
	}
	if _, ok := expectedCHChunkEnd(1_100_000, 1, 1_250_000, 500_000); ok {
		t.Fatalf("expectedCHChunkEnd should reject non-aligned start")
	}
	p := filepath.Join(t.TempDir(), "id_000500001_001000000.parquet")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatalf("write chunk fixture: %v", err)
	}
	cf, ok := parseCHChunkFilePath(p)
	if !ok {
		t.Fatalf("parseCHChunkFilePath failed")
	}
	if cf.StartID != 500001 || cf.EndID != 1000000 {
		t.Fatalf("parsed range=%d-%d want 500001-1000000", cf.StartID, cf.EndID)
	}
	span := detectCHChunkSpan([]localChunkFile{
		{StartID: 1, EndID: 1_000_000},
		{StartID: 1_000_001, EndID: 2_000_000},
		{StartID: 2_000_001, EndID: 2_132_548},
	})
	if span != 1_000_000 {
		t.Fatalf("detectCHChunkSpan=%d want 1000000", span)
	}
}

func TestCompactDeltaToClickHouseParquet(t *testing.T) {
	cfg := Config{DataDir: t.TempDir()}
	if err := cfg.EnsureRawDirs(); err != nil {
		t.Fatalf("EnsureRawDirs: %v", err)
	}
	createTestClickHouseParquet(t, filepath.Join(cfg.ClickHouseParquetDir(), "id_000000001_000001000.parquet"), []hnCHRow{
		{ID: 1, TypeCode: 1, By: "alice", Time: 1700000000, Title: "story-1"},
		{ID: 2, TypeCode: 2, By: "bob", Time: 1700000001, Parent: sqlNullInt64(1), Text: "old-comment"},
	})
	createTestClickHouseParquet(t, filepath.Join(cfg.ClickHouseDeltaParquetDir(), "id_000000002_000000003.parquet"), []hnCHRow{
		{ID: 2, TypeCode: 2, By: "bob", Time: 1700000001, Parent: sqlNullInt64(1), Text: "new-comment"},
		{ID: 3, TypeCode: 5, By: "carol", Time: 1700000002, Title: "job-3"},
	})
	if err := cfg.WriteDownloadState(&DownloadState{
		SourceUsed: "hybrid",
		Delta:      &ClickHouseRunState{StartID: 2, EndID: 3, ChunkIDSpan: 1000, IncrementalFromID: 2},
		ClickHouse: &ClickHouseRunState{StartID: 1, EndID: 2, ChunkIDSpan: 1000, IncrementalFromID: 1},
	}); err != nil {
		t.Fatalf("WriteDownloadState: %v", err)
	}

	res, err := cfg.CompactDeltaToClickHouseParquet(context.Background(), CompactOptions{ChunkIDSpan: 1000})
	if err != nil {
		t.Fatalf("CompactDeltaToClickHouseParquet: %v", err)
	}
	if res.ChunksWritten != 1 {
		t.Fatalf("ChunksWritten=%d want 1", res.ChunksWritten)
	}
	if _, err := os.Stat(filepath.Join(cfg.ClickHouseParquetDir(), "id_000000001_000001000.parquet")); err != nil {
		t.Fatalf("expected compacted chunk file: %v", err)
	}

	imp, err := cfg.Import(context.Background(), ImportOptions{Source: ImportSourceClickHouse})
	if err != nil {
		t.Fatalf("Import clickhouse after compact: %v", err)
	}
	db, err := sql.Open("duckdb", imp.DBPath+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()
	var gotText string
	if err := db.QueryRow(`SELECT text FROM items WHERE id=2`).Scan(&gotText); err != nil {
		t.Fatalf("query id=2 text: %v", err)
	}
	if gotText != "new-comment" {
		t.Fatalf("id=2 text=%q want new-comment", gotText)
	}
	var count int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM items WHERE id=3`).Scan(&count); err != nil {
		t.Fatalf("query id=3 count: %v", err)
	}
	if count != 1 {
		t.Fatalf("id=3 count=%d want 1", count)
	}
}

func TestExportMonthlyParquet(t *testing.T) {
	cfg := Config{DataDir: t.TempDir()}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		q := string(body)
		if strings.Contains(q, "GROUP BY ym") && strings.Contains(q, "FORMAT JSONEachRow") {
			_, _ = io.WriteString(w, `{"ym":"2023-11","n":"2"}`+"\n"+`{"ym":"2023-12","n":"1"}`+"\n")
			return
		}
		if strings.Contains(q, "FORMAT Parquet") {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte("PAR1fake"))
			return
		}
		http.Error(w, "unexpected query", http.StatusBadRequest)
	}))
	defer srv.Close()
	cfg.ClickHouseBaseURL = srv.URL
	outDir := filepath.Join(cfg.BaseDir(), "export-test")

	res1, err := cfg.ExportMonthlyParquet(context.Background(), ExportOptions{
		OutDir:        outDir,
		RefreshLatest: true,
	})
	if err != nil {
		t.Fatalf("ExportMonthlyParquet first: %v", err)
	}
	if res1.MonthsWritten != 2 {
		t.Fatalf("MonthsWritten=%d want 2", res1.MonthsWritten)
	}
	if !fileExistsNonEmpty(filepath.Join(outDir, "items_2023_11.parquet")) || !fileExistsNonEmpty(filepath.Join(outDir, "items_2023_12.parquet")) {
		t.Fatalf("expected exported month parquet files")
	}

	res2, err := cfg.ExportMonthlyParquet(context.Background(), ExportOptions{
		OutDir:        outDir,
		RefreshLatest: true,
	})
	if err != nil {
		t.Fatalf("ExportMonthlyParquet second: %v", err)
	}
	if res2.MonthsSkipped != 1 || res2.MonthsWritten != 1 {
		t.Fatalf("second run scanned=%d written=%d skipped=%d want written=1 skipped=1", res2.MonthsScanned, res2.MonthsWritten, res2.MonthsSkipped)
	}
}

func TestExportMonthlyParquet_PrefersDuckDBAndSkipsLatestUnchanged(t *testing.T) {
	cfg := Config{DataDir: t.TempDir()}
	makeItemsDBForExport(t, cfg.DefaultDBPath())
	outDir := filepath.Join(cfg.BaseDir(), "export-test-local")

	var progress1 []ExportProgress
	res1, err := cfg.ExportMonthlyParquet(context.Background(), ExportOptions{
		OutDir:        outDir,
		RefreshLatest: true,
		Progress: func(p ExportProgress) {
			progress1 = append(progress1, p)
		},
	})
	if err != nil {
		t.Fatalf("ExportMonthlyParquet local first: %v", err)
	}
	if res1.SourceUsed != "duckdb" {
		t.Fatalf("SourceUsed=%q want duckdb", res1.SourceUsed)
	}
	if res1.MonthsWritten != 2 {
		t.Fatalf("MonthsWritten=%d want 2", res1.MonthsWritten)
	}
	if !fileExistsNonEmpty(filepath.Join(outDir, "items_2023_11.parquet")) || !fileExistsNonEmpty(filepath.Join(outDir, "items_2023_12.parquet")) {
		t.Fatalf("expected exported month parquet files from local duckdb")
	}
	if len(progress1) == 0 {
		t.Fatalf("expected progress events")
	}

	res2, err := cfg.ExportMonthlyParquet(context.Background(), ExportOptions{
		OutDir:        outDir,
		RefreshLatest: true,
	})
	if err != nil {
		t.Fatalf("ExportMonthlyParquet local second: %v", err)
	}
	if res2.SourceUsed != "duckdb" {
		t.Fatalf("second SourceUsed=%q want duckdb", res2.SourceUsed)
	}
	if res2.MonthsSkipped != 2 || res2.MonthsWritten != 0 {
		t.Fatalf("second run written=%d skipped=%d want written=0 skipped=2", res2.MonthsWritten, res2.MonthsSkipped)
	}
	var latest ExportMonth
	foundLatest := false
	for _, m := range res2.Months {
		if m.Month == res2.LatestMonth {
			latest = m
			foundLatest = true
			break
		}
	}
	if !foundLatest {
		t.Fatalf("latest month %q not found in result months", res2.LatestMonth)
	}
	if latest.SkipReason != "latest_unchanged" {
		t.Fatalf("latest skip reason=%q want latest_unchanged", latest.SkipReason)
	}
}

func TestBuildDomains(t *testing.T) {
	cfg := Config{DataDir: t.TempDir()}
	srcDB := cfg.DefaultDBPath()
	makeItemsDBForDomains(t, srcDB)

	var progress []DomainsProgress
	res1, err := cfg.BuildDomains(context.Background(), DomainsOptions{
		SourceDBPath: srcDB,
		OutDBPath:    filepath.Join(cfg.BaseDir(), "hn_domains.duckdb"),
		Progress: func(p DomainsProgress) {
			progress = append(progress, p)
		},
	})
	if err != nil {
		t.Fatalf("BuildDomains first: %v", err)
	}
	if res1.PagesRows != 4 {
		t.Fatalf("PagesRows=%d want 4", res1.PagesRows)
	}
	if res1.DomainsRows != 3 {
		t.Fatalf("DomainsRows=%d want 3", res1.DomainsRows)
	}
	if !res1.PagesBuilt || res1.PagesReused {
		t.Fatalf("first run pages flags built=%v reused=%v want built=true reused=false", res1.PagesBuilt, res1.PagesReused)
	}
	if len(progress) == 0 {
		t.Fatalf("expected progress events")
	}

	db, err := sql.Open("duckdb", res1.OutDBPath+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open domains db: %v", err)
	}
	defer db.Close()
	var pagesN int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM pages`).Scan(&pagesN); err != nil {
		t.Fatalf("count pages: %v", err)
	}
	if pagesN != 4 {
		t.Fatalf("pages count=%d want 4", pagesN)
	}
	var itemCount, firstItemID, latestItemID int64
	if err := db.QueryRow(`SELECT item_count, first_item_id, latest_item_id FROM domains WHERE domain='example.com'`).Scan(&itemCount, &firstItemID, &latestItemID); err != nil {
		t.Fatalf("query example.com domain row: %v", err)
	}
	if itemCount != 2 || firstItemID != 1 || latestItemID != 2 {
		t.Fatalf("example.com domain stats got count=%d first=%d latest=%d want 2/1/2", itemCount, firstItemID, latestItemID)
	}
	_ = db.Close()

	res2, err := cfg.BuildDomains(context.Background(), DomainsOptions{
		SourceDBPath: srcDB,
		OutDBPath:    res1.OutDBPath,
	})
	if err != nil {
		t.Fatalf("BuildDomains second: %v", err)
	}
	if res2.PagesBuilt || !res2.PagesReused {
		t.Fatalf("second run pages flags built=%v reused=%v want built=false reused=true", res2.PagesBuilt, res2.PagesReused)
	}
}

func TestBuildRecrawlSeedDB(t *testing.T) {
	cfg := Config{DataDir: t.TempDir()}
	srcDB := cfg.DefaultDBPath()
	makeItemsDBForDomains(t, srcDB)
	if _, err := cfg.BuildDomains(context.Background(), DomainsOptions{
		SourceDBPath: srcDB,
		OutDBPath:    cfg.DomainsDBPath(),
	}); err != nil {
		t.Fatalf("BuildDomains for recrawl seed test: %v", err)
	}

	res, err := cfg.BuildRecrawlSeedDB(context.Background(), RecrawlSeedOptions{
		DomainsDBPath: cfg.DomainsDBPath(),
		OutDBPath:     cfg.RecrawlSeedDBPath(),
		Limit:         3,
	})
	if err != nil {
		t.Fatalf("BuildRecrawlSeedDB: %v", err)
	}
	if res.Rows != 3 {
		t.Fatalf("seed rows=%d want 3", res.Rows)
	}
	db, err := sql.Open("duckdb", res.OutDBPath+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	defer db.Close()
	var docsCount int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM docs`).Scan(&docsCount); err != nil {
		t.Fatalf("count docs: %v", err)
	}
	if docsCount != 3 {
		t.Fatalf("docs count=%d want 3", docsCount)
	}
	var domain, protocol, tld string
	if err := db.QueryRow(`SELECT domain, protocol, tld FROM docs ORDER BY item_id DESC LIMIT 1`).Scan(&domain, &protocol, &tld); err != nil {
		t.Fatalf("query docs fields: %v", err)
	}
	if strings.TrimSpace(domain) == "" || strings.TrimSpace(protocol) == "" || strings.TrimSpace(tld) == "" {
		t.Fatalf("expected non-empty domain/protocol/tld, got domain=%q protocol=%q tld=%q", domain, protocol, tld)
	}
}

func createTestParquet(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parquet dir: %v", err)
	}
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()
	escaped := strings.ReplaceAll(path, "'", "''")
	q := fmt.Sprintf(`COPY (
		SELECT 1::BIGINT AS id,
		       'story'::VARCHAR AS type,
		       1700000000::BIGINT AS time,
		       'alice'::VARCHAR AS "by",
		       'hello'::VARCHAR AS title,
		       NULL::VARCHAR AS text,
		       NULL::BIGINT AS parent
		UNION ALL
		SELECT 2::BIGINT AS id,
		       'comment'::VARCHAR AS type,
		       1700000001::BIGINT AS time,
		       'bob'::VARCHAR AS "by",
		       NULL::VARCHAR AS title,
		       'reply'::VARCHAR AS text,
		       1::BIGINT AS parent
	) TO '%s' (FORMAT PARQUET)`, escaped)
	if _, err := db.Exec(q); err != nil {
		t.Fatalf("create parquet fixture: %v", err)
	}
}

func makeItemsDBForExport(t *testing.T, dbPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatalf("mkdir db dir: %v", err)
	}
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE items AS
SELECT 1::BIGINT AS id, 0::BIGINT AS deleted, 'story'::VARCHAR AS type, 'alice'::VARCHAR AS "by",
       1700000000::BIGINT AS time, epoch_ms(1700000000::BIGINT * 1000) AS time_ts,
       NULL::VARCHAR AS text, 0::BIGINT AS dead, NULL::BIGINT AS parent, NULL::BIGINT AS poll,
       NULL::BIGINT[] AS kids, NULL::VARCHAR AS url, 1::BIGINT AS score, 'nov-story'::VARCHAR AS title,
       NULL::BIGINT[] AS parts, NULL::BIGINT AS descendants
UNION ALL
SELECT 2::BIGINT, 0::BIGINT, 'comment'::VARCHAR, 'bob'::VARCHAR,
       1701000000::BIGINT, epoch_ms(1701000000::BIGINT * 1000),
       'nov-comment'::VARCHAR, 0::BIGINT, 1::BIGINT, NULL::BIGINT, NULL::BIGINT[], NULL::VARCHAR, NULL::BIGINT, NULL::VARCHAR, NULL::BIGINT[], NULL::BIGINT
UNION ALL
SELECT 3::BIGINT, 0::BIGINT, 'story'::VARCHAR, 'carol'::VARCHAR,
       1701388800::BIGINT, epoch_ms(1701388800::BIGINT * 1000),
       NULL::VARCHAR, 0::BIGINT, NULL::BIGINT, NULL::BIGINT, NULL::BIGINT[], NULL::VARCHAR, 5::BIGINT, 'dec-story'::VARCHAR, NULL::BIGINT[], NULL::BIGINT`); err != nil {
		t.Fatalf("create items export fixture: %v", err)
	}
}

func makeItemsDBForDomains(t *testing.T, dbPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatalf("mkdir db dir: %v", err)
	}
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE items (
id BIGINT,
deleted BIGINT,
type VARCHAR,
"by" VARCHAR,
time BIGINT,
time_ts TIMESTAMP,
text VARCHAR,
dead BIGINT,
parent BIGINT,
poll BIGINT,
kids BIGINT[],
url VARCHAR,
score BIGINT,
title VARCHAR,
parts BIGINT[],
descendants BIGINT
)`); err != nil {
		t.Fatalf("create items table for domains: %v", err)
	}
	stmt := `INSERT INTO items (
id, deleted, type, "by", time, time_ts, text, dead, parent, poll, kids, url, score, title, parts, descendants
) VALUES (?, ?, ?, ?, ?, epoch_ms(? * 1000), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	rows := []struct {
		id          int64
		typ         string
		by          string
		t           int64
		text        any
		parent      any
		url         any
		score       any
		title       any
		descendants any
	}{
		{1, "story", "alice", 1700000000, nil, nil, "https://example.com/a", int64(10), "a", int64(1)},
		{2, "story", "bob", 1700001000, nil, nil, "https://example.com/b?q=1", int64(12), "b", int64(2)},
		{3, "job", "carol", 1700002000, nil, nil, "http://news.ycombinator.com/item?id=1", nil, "job", nil},
		{4, "story", "dave", 1700003000, nil, nil, "https://sub.example.org/path", int64(8), "c", int64(3)},
		{5, "comment", "eve", 1700004000, "no link", int64(1), nil, nil, nil, nil},
	}
	for _, r := range rows {
		if _, err := db.Exec(stmt,
			r.id, int64(0), r.typ, r.by, r.t, r.t,
			r.text, int64(0), r.parent, nil, nil, r.url, r.score, r.title, nil, r.descendants,
		); err != nil {
			t.Fatalf("insert item %d for domains: %v", r.id, err)
		}
	}
}

type hnCHRow struct {
	ID       int64
	TypeCode int64
	By       string
	Time     int64
	Title    string
	Text     string
	Parent   sql.NullInt64
}

func sqlNullInt64(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: true}
}

func createTestClickHouseParquet(t *testing.T, path string, rows []hnCHRow) {
	t.Helper()
	if len(rows) == 0 {
		t.Fatalf("createTestClickHouseParquet requires rows")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir clickhouse parquet dir: %v", err)
	}
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TEMP TABLE ch_items (
		id BIGINT,
		deleted BIGINT,
		type BIGINT,
		"by" VARCHAR,
		time BIGINT,
		text VARCHAR,
		dead BIGINT,
		parent BIGINT,
		poll BIGINT,
		kids BIGINT[],
		url VARCHAR,
		score BIGINT,
		title VARCHAR,
		parts BIGINT[],
		descendants BIGINT
	)`); err != nil {
		t.Fatalf("create temp ch_items: %v", err)
	}
	for _, r := range rows {
		if _, err := db.Exec(`INSERT INTO ch_items VALUES (?, 0, ?, ?, ?, ?, 0, ?, NULL, NULL, NULL, NULL, ?, NULL, NULL)`,
			r.ID, r.TypeCode, r.By, r.Time, nullIfEmpty(r.Text), nullInt64Arg(r.Parent), nullIfEmpty(r.Title),
		); err != nil {
			t.Fatalf("insert ch row %d: %v", r.ID, err)
		}
	}
	escaped := strings.ReplaceAll(path, "'", "''")
	if _, err := db.Exec(fmt.Sprintf(`COPY ch_items TO '%s' (FORMAT PARQUET)`, escaped)); err != nil {
		t.Fatalf("copy ch_items parquet: %v", err)
	}
}

func writeAPIChunk(t *testing.T, path string, lines []string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir api chunk dir: %v", err)
	}
	body := strings.Join(lines, "\n")
	if body != "" {
		body += "\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write api chunk: %v", err)
	}
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullInt64Arg(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}

func newHNTestServer(t *testing.T, parquetBytes []byte, items map[int64]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	if parquetBytes != nil {
		mux.HandleFunc("/items.parquet", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("ETag", `"test-etag"`)
			w.Header().Set("Last-Modified", time.Unix(1700000000, 0).UTC().Format(http.TimeFormat))
			reader := bytes.NewReader(parquetBytes)
			http.ServeContent(w, r, "items.parquet", time.Unix(1700000000, 0), reader)
		})
	}
	if items != nil {
		var maxID int64
		for id := range items {
			if id > maxID {
				maxID = id
			}
		}
		mux.HandleFunc("/v0/maxitem.json", func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, strconv.FormatInt(maxID, 10))
		})
		mux.HandleFunc("/v0/item/", func(w http.ResponseWriter, r *http.Request) {
			idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v0/item/"), ".json")
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				http.Error(w, "bad id", http.StatusBadRequest)
				return
			}
			payload, ok := items[id]
			if !ok {
				payload = "null"
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, payload)
		})
	}
	return httptest.NewServer(mux)
}
