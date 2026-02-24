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
