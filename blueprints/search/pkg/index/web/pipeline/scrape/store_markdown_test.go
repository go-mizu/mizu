package scrape

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DataDog/zstd"
	_ "github.com/duckdb/duckdb-go/v2"
)

func TestStoreGetPages_NullableFieldsDoNotDropRows(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	domain := "example.com"
	resultDir := filepath.Join(base, domain, "results")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir results: %v", err)
	}

	dbPath := filepath.Join(resultDir, "results_000.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE pages (
			url            VARCHAR,
			url_hash       BIGINT,
			status_code    SMALLINT,
			content_type   VARCHAR,
			content_length BIGINT,
			title          VARCHAR,
			description    VARCHAR,
			language       VARCHAR,
			fetch_time_ms  BIGINT,
			crawled_at     TIMESTAMP,
			error          VARCHAR
		)
	`)
	if err != nil {
		t.Fatalf("create pages: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO pages
			(url, url_hash, status_code, content_type, content_length, title, description, language, fetch_time_ms, crawled_at, error)
		VALUES
			('https://example.com/a', 1, 200, 'text/html', 1234, 'A', 'desc', 'en', 10, NOW(), ''),
			('https://example.com/b', 2, 200, NULL, NULL, NULL, NULL, NULL, NULL, NOW(), NULL)
	`)
	if err != nil {
		t.Fatalf("insert pages: %v", err)
	}

	s := NewStore(base)
	resp, err := s.GetPages(domain, 1, 50, "", "crawled_at", "")
	if err != nil {
		t.Fatalf("GetPages: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("Total=%d want 2", resp.Total)
	}
	if len(resp.Pages) != 2 {
		t.Fatalf("len(Pages)=%d want 2", len(resp.Pages))
	}
}

func TestScrapeMarkdownTask_Run_ConvertsHTMLBody(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	domain := "example.com"
	resultDir := filepath.Join(base, domain, "results")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir results: %v", err)
	}

	dbPath := filepath.Join(resultDir, "results_000.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE pages (
			url          VARCHAR,
			url_hash     BIGINT,
			body         BLOB,
			content_type VARCHAR,
			status_code  SMALLINT
		)
	`)
	if err != nil {
		t.Fatalf("create pages: %v", err)
	}

	html := []byte("<html><head><title>Hello</title></head><body><h1>Hello</h1><p>World</p></body></html>")
	compressed, err := zstd.Compress(nil, html)
	if err != nil {
		t.Fatalf("compress html: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO pages (url, url_hash, body, content_type, status_code)
		VALUES ('https://example.com/', 12345, ?, 'text/html; charset=utf-8', 200)
	`, compressed)
	if err != nil {
		t.Fatalf("insert page: %v", err)
	}

	task := NewScrapeMarkdownTask(domain, base)
	metric, err := task.Run(context.Background(), func(*ScrapeMarkdownState) {})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if metric.Docs != 1 {
		t.Fatalf("Docs=%d want 1", metric.Docs)
	}

	mdPath := filepath.Join(base, domain, "markdown", "12345.md")
	data, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("read markdown: %v", err)
	}
	if !strings.Contains(string(data), "Hello") {
		t.Fatalf("markdown does not contain converted content: %q", string(data))
	}
}
