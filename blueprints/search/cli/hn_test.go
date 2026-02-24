package cli

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

func TestNewHN_Subcommands(t *testing.T) {
	cmd := NewHN()
	_ = findSubcommand(t, cmd, "list")
	_ = findSubcommand(t, cmd, "download")
	_ = findSubcommand(t, cmd, "import")
	_ = findSubcommand(t, cmd, "status")
	_ = findSubcommand(t, cmd, "sync")
}

func TestHNCommands_EndToEnd(t *testing.T) {
	parquetBytes := buildHNParquetFixtureBytes(t)
	items := map[int64]string{
		1: `{"id":1,"type":"story","time":1700000000,"by":"a","title":"one"}`,
		2: `{"id":2,"type":"comment","time":1700000001,"by":"b","parent":1,"text":"two"}`,
		3: `null`,
		4: `{"id":4,"type":"job","time":1700000002,"by":"c","title":"job"}`,
	}
	server := newHNCLITestServer(parquetBytes, items)
	defer server.Close()

	dataDir := t.TempDir()
	t.Setenv("MIZU_HN_DATA_DIR", dataDir)
	t.Setenv("MIZU_HN_PARQUET_URL", server.URL+"/items.parquet")
	t.Setenv("MIZU_HN_API_BASE_URL", server.URL+"/v0")

	runHNCommand(t, "list", "--no-remote")
	runHNCommand(t, "download", "--source", "parquet")
	runHNCommand(t, "import", "--source", "parquet")
	runHNCommand(t, "status")

	if _, err := os.Stat(filepath.Join(dataDir, "hn.duckdb")); err != nil {
		t.Fatalf("expected hn.duckdb after import: %v", err)
	}

	runHNCommand(t, "sync", "--source", "api", "--from-id", "1", "--to-id", "4", "--chunk-size", "2", "--workers", "2")

	db, err := sql.Open("duckdb", filepath.Join(dataDir, "hn.duckdb")+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()
	var rows int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&rows); err != nil {
		t.Fatalf("query rows: %v", err)
	}
	if rows == 0 {
		t.Fatalf("expected imported rows > 0 after sync")
	}
}

func runHNCommand(t *testing.T, args ...string) {
	t.Helper()
	cmd := NewHN()
	cmd.SetContext(context.Background())
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("hn %s: %v", strings.Join(args, " "), err)
	}
}

func buildHNParquetFixtureBytes(t *testing.T) []byte {
	t.Helper()
	tmp := t.TempDir()
	pqPath := filepath.Join(tmp, "items.parquet")
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()
	escaped := strings.ReplaceAll(pqPath, "'", "''")
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
	b, err := os.ReadFile(pqPath)
	if err != nil {
		t.Fatalf("read parquet fixture: %v", err)
	}
	return b
}

func newHNCLITestServer(parquetBytes []byte, items map[int64]string) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/items.parquet", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"cli-test-etag"`)
		reader := bytes.NewReader(parquetBytes)
		http.ServeContent(w, r, "items.parquet", time.Unix(1700000000, 0), reader)
	})
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
	return httptest.NewServer(mux)
}
