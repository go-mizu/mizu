package warc_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

func buildHTTPResponseRecord(uri, contentType string, status int, body string) string {
	httpResp := fmt.Sprintf("HTTP/1.1 %d OK\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n%s",
		status, contentType, len(body), body)
	return "WARC/1.1\r\n" +
		"WARC-Type: response\r\n" +
		"WARC-Target-URI: " + uri + "\r\n" +
		"WARC-Date: 2026-01-01T00:00:00Z\r\n" +
		"WARC-Record-ID: <urn:uuid:" + uri + ">\r\n" +
		"Content-Length: " + fmt.Sprintf("%d", len(httpResp)) + "\r\n" +
		"\r\n" +
		httpResp + "\r\n\r\n"
}

func gzipRecords(records []string) []byte {
	var buf bytes.Buffer
	for _, r := range records {
		gz := gzip.NewWriter(&buf)
		gz.Write([]byte(r))
		gz.Close()
	}
	return buf.Bytes()
}

func TestImporter_FilterByMIME(t *testing.T) {
	records := []string{
		buildHTTPResponseRecord("https://a.com/", "text/html", 200, "<html><title>A</title></html>"),
		buildHTTPResponseRecord("https://b.com/", "application/json", 200, `{"k":"v"}`),
		buildHTTPResponseRecord("https://c.com/", "text/html", 200, "<html><title>C</title></html>"),
	}
	data := gzipRecords(records)

	dir := t.TempDir()
	db, err := warc.OpenRecordDB(dir, 100)
	if err != nil {
		t.Fatalf("OpenRecordDB: %v", err)
	}
	defer db.Close()

	im := warc.NewImporter(db, warc.ImportOptions{
		MIMETypes:   []string{"text/html"},
		RecordTypes: []string{"response"},
	})
	r := warc.NewReader(bytes.NewReader(data))
	var last warc.ImportStats
	if err := im.Import(context.Background(), r, func(s warc.ImportStats) { last = s }); err != nil {
		t.Fatalf("Import: %v", err)
	}

	// Force flush
	db.Close()

	// Verify counts: 2 html imported, 1 json skipped
	if last.Imported < 2 {
		t.Errorf("expected >=2 imported, got %d", last.Imported)
	}
	if last.Skipped < 1 {
		t.Errorf("expected >=1 skipped, got %d", last.Skipped)
	}

	// Verify DuckDB shards exist
	entries, _ := os.ReadDir(dir)
	var dbCount int
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".duckdb" {
			dbCount++
		}
	}
	if dbCount == 0 {
		t.Error("no .duckdb files created")
	}
}

func TestImporter_FilterByStatus(t *testing.T) {
	records := []string{
		buildHTTPResponseRecord("https://ok.com/", "text/html", 200, "<html>ok</html>"),
		buildHTTPResponseRecord("https://redir.com/", "text/html", 301, ""),
		buildHTTPResponseRecord("https://notfound.com/", "text/html", 404, "not found"),
	}

	dir := t.TempDir()
	db, err := warc.OpenRecordDB(dir, 100)
	if err != nil {
		t.Fatalf("OpenRecordDB: %v", err)
	}

	im := warc.NewImporter(db, warc.ImportOptions{
		StatusCodes: []int{200},
		RecordTypes: []string{"response"},
	})
	r := warc.NewReader(bytes.NewReader([]byte(records[0] + records[1] + records[2])))
	var last warc.ImportStats
	im.Import(context.Background(), r, func(s warc.ImportStats) { last = s })
	db.Close()

	if last.Imported != 1 {
		t.Errorf("expected 1 imported (status=200), got %d", last.Imported)
	}
}
