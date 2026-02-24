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
	writeAPIChunk(t, filepath.Join(cfg.APIChunksDir(), "items_000000003_000000003.jsonl"), []string{
		`{"id":3,"type":"story","time":1700000002,"by":"carol","title":"s3-api-v1"}`,
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
	if err := os.Remove(filepath.Join(cfg.APIChunksDir(), "items_000000003_000000003.jsonl")); err != nil {
		t.Fatalf("remove old api chunk: %v", err)
	}
	writeAPIChunk(t, filepath.Join(cfg.APIChunksDir(), "items_000000003_000000005.jsonl"), []string{
		`{"id":3,"type":"story","time":1700000002,"by":"carol","title":"s3-api-v2"}`,
		`{"id":5,"type":"job","time":1700000004,"by":"erin","title":"j5"}`,
	})
	if err := cfg.WriteDownloadState(&DownloadState{
		SourceUsed: "hybrid",
		ClickHouse: &ClickHouseRunState{StartID: 1, EndID: 4, ChunkIDSpan: 2, IncrementalFromID: 3},
		API:        &APIRunState{StartID: 3, EndID: 5, MaxItem: 5, IsDelta: true},
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
	if title3 != "s3-api-v2" {
		t.Fatalf("id=3 title=%q want api overlay update", title3)
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
