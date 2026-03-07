package web

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

// createTestWARCMd writes a .md.warc.gz file with N conversion records.
// Each record has a UUID doc ID, a target URI, a date, and a markdown body.
func createTestWARCMd(t *testing.T, path string, count int) []string {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()

	var docIDs []string
	for i := range count {
		docID := fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
		body := fmt.Sprintf("# Document %d\n\nThis is test document number %d.\n\nURL: https://example-%d.com/page\n", i, i, i)

		hdr := warcpkg.Header{
			"WARC-Type":       warcpkg.TypeConversion,
			"WARC-Target-URI": fmt.Sprintf("https://example-%d.com/page", i),
			"WARC-Date":       time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Hour).Format(time.RFC3339),
			"WARC-Record-ID":  fmt.Sprintf("<urn:uuid:%s>", docID),
			"WARC-Refers-To":  fmt.Sprintf("<urn:uuid:ref-%012d>", i),
			"Content-Type":    "text/markdown",
			"Content-Length":  strconv.Itoa(len(body)),
		}

		rec := &warcpkg.Record{
			Header: hdr,
			Body:   bytes.NewReader([]byte(body)),
		}

		// Each record in its own gzip member (as used by the pack pipeline).
		gz, err := gzip.NewWriterLevel(f, gzip.BestSpeed)
		if err != nil {
			t.Fatalf("gzip writer: %v", err)
		}
		w := warcpkg.NewWriter(gz)
		if err := w.WriteRecord(rec); err != nil {
			t.Fatalf("write record: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("close writer: %v", err)
		}
		if err := gz.Close(); err != nil {
			t.Fatalf("close gzip: %v", err)
		}
		docIDs = append(docIDs, docID)
	}
	return docIDs
}

func TestDocStoreScanAndList(t *testing.T) {
	root := t.TempDir()
	warcMdDir := filepath.Join(root, "warc_md")
	mustMkdir(t, warcMdDir)

	// Create a test .md.warc.gz with 5 records.
	warcMdPath := filepath.Join(warcMdDir, "00000.md.warc.gz")
	docIDs := createTestWARCMd(t, warcMdPath, 5)

	ds, err := NewDocStore(warcMdDir)
	if err != nil {
		t.Fatalf("NewDocStore: %v", err)
	}
	defer ds.Close()

	ctx := context.Background()

	// Scan the shard.
	n, err := ds.ScanShard(ctx, "", "00000", warcMdPath)
	if err != nil {
		t.Fatalf("ScanShard: %v", err)
	}
	if n != 5 {
		t.Fatalf("expected 5 docs, got %d", n)
	}

	// List all docs.
	docs, total, err := ds.ListDocs(ctx, "", "00000", 1, 100, "", "date")
	if err != nil {
		t.Fatalf("ListDocs: %v", err)
	}
	if total != 5 {
		t.Fatalf("expected total=5, got %d", total)
	}
	if len(docs) != 5 {
		t.Fatalf("expected 5 docs returned, got %d", len(docs))
	}

	// Verify metadata.
	for _, d := range docs {
		if d.URL == "" {
			t.Errorf("doc %s: expected non-empty URL", d.DocID)
		}
		if d.Host == "" {
			t.Errorf("doc %s: expected non-empty Host", d.DocID)
		}
		if d.Title == "" {
			t.Errorf("doc %s: expected non-empty Title", d.DocID)
		}
		if d.CrawlDate.IsZero() {
			t.Errorf("doc %s: expected non-zero CrawlDate", d.DocID)
		}
		if d.GzipOffset < 0 {
			t.Errorf("doc %s: expected non-negative GzipOffset", d.DocID)
		}
		if d.GzipSize <= 0 {
			t.Errorf("doc %s: expected positive GzipSize", d.DocID)
		}
	}

	// Verify host extraction.
	doc0, ok, err := ds.GetDoc(ctx, "", "00000", docIDs[0])
	if err != nil {
		t.Fatalf("GetDoc: %v", err)
	}
	if !ok {
		t.Fatal("expected doc to be found")
	}
	if doc0.Host != "example-0.com" {
		t.Fatalf("expected host=example-0.com, got %q", doc0.Host)
	}

	// Verify filtering by host.
	filtered, filteredTotal, err := ds.ListDocs(ctx, "", "00000", 1, 100, "example-3", "date")
	if err != nil {
		t.Fatalf("ListDocs with filter: %v", err)
	}
	if filteredTotal != 1 {
		t.Fatalf("expected 1 filtered doc, got %d", filteredTotal)
	}
	if filtered[0].Host != "example-3.com" {
		t.Fatalf("expected host=example-3.com, got %q", filtered[0].Host)
	}
}

func TestDocStoreOffsetRead(t *testing.T) {
	root := t.TempDir()
	warcMdDir := filepath.Join(root, "warc_md")
	mustMkdir(t, warcMdDir)

	warcMdPath := filepath.Join(warcMdDir, "00000.md.warc.gz")
	docIDs := createTestWARCMd(t, warcMdPath, 10)

	ds, err := NewDocStore(warcMdDir)
	if err != nil {
		t.Fatalf("NewDocStore: %v", err)
	}
	defer ds.Close()

	ctx := context.Background()
	n, err := ds.ScanShard(ctx, "", "00000", warcMdPath)
	if err != nil {
		t.Fatalf("ScanShard: %v", err)
	}
	if n != 10 {
		t.Fatalf("expected 10 docs, got %d", n)
	}

	// Read each doc by offset and verify content.
	for i, id := range docIDs {
		rec, ok, err := ds.GetDoc(ctx, "", "00000", id)
		if err != nil {
			t.Fatalf("GetDoc(%s): %v", id, err)
		}
		if !ok {
			t.Fatalf("GetDoc(%s): not found", id)
		}

		if rec.GzipOffset <= 0 && i > 0 {
			t.Fatalf("doc %s (idx %d): expected positive offset, got %d", id, i, rec.GzipOffset)
		}
		if rec.GzipSize <= 0 {
			t.Fatalf("doc %s: expected positive size, got %d", id, rec.GzipSize)
		}

		// Read by offset.
		body, err := ReadDocByOffset(warcMdPath, rec.GzipOffset, rec.GzipSize)
		if err != nil {
			t.Fatalf("ReadDocByOffset(%s, offset=%d, size=%d): %v", id, rec.GzipOffset, rec.GzipSize, err)
		}
		if len(body) == 0 {
			t.Fatalf("ReadDocByOffset(%s): empty body", id)
		}

		expected := fmt.Sprintf("# Document %d", i)
		if !bytes.Contains(body, []byte(expected)) {
			t.Fatalf("doc %s: expected body to contain %q, got %q", id, expected, string(body[:min(100, len(body))]))
		}
	}
}

func TestDocStoreScanAll(t *testing.T) {
	root := t.TempDir()
	warcMdDir := filepath.Join(root, "warc_md")
	mustMkdir(t, warcMdDir)

	// Create two shards.
	createTestWARCMd(t, filepath.Join(warcMdDir, "00000.md.warc.gz"), 3)
	createTestWARCMd(t, filepath.Join(warcMdDir, "00001.md.warc.gz"), 2)

	ds, err := NewDocStore(warcMdDir)
	if err != nil {
		t.Fatalf("NewDocStore: %v", err)
	}
	defer ds.Close()

	ctx := context.Background()
	total, err := ds.ScanAll(ctx, "", root)
	if err != nil {
		t.Fatalf("ScanAll: %v", err)
	}
	if total != 5 {
		t.Fatalf("expected 5 total docs, got %d", total)
	}

	// Verify both shards have metadata.
	metas, err := ds.ListShardMetas(ctx, "")
	if err != nil {
		t.Fatalf("ListShardMetas: %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("expected 2 shard metas, got %d", len(metas))
	}
	if metas[0].TotalDocs != 3 {
		t.Fatalf("expected shard 00000 to have 3 docs, got %d", metas[0].TotalDocs)
	}
	if metas[1].TotalDocs != 2 {
		t.Fatalf("expected shard 00001 to have 2 docs, got %d", metas[1].TotalDocs)
	}
}

func TestDocStoreShardStats(t *testing.T) {
	root := t.TempDir()
	warcMdDir := filepath.Join(root, "warc_md")
	mustMkdir(t, warcMdDir)

	warcMdPath := filepath.Join(warcMdDir, "00000.md.warc.gz")
	createTestWARCMd(t, warcMdPath, 5)

	ds, err := NewDocStore(warcMdDir)
	if err != nil {
		t.Fatalf("NewDocStore: %v", err)
	}
	defer ds.Close()

	ctx := context.Background()
	ds.ScanShard(ctx, "", "00000", warcMdPath)

	stats, err := ds.ShardStats(ctx, "", "00000")
	if err != nil {
		t.Fatalf("ShardStats: %v", err)
	}

	if stats.TotalDocs != 5 {
		t.Fatalf("expected 5 total docs, got %d", stats.TotalDocs)
	}
	if stats.TotalSize <= 0 {
		t.Fatalf("expected positive total size, got %d", stats.TotalSize)
	}

	// Top domains should be populated (one doc per unique host).
	if len(stats.TopDomains) != 5 {
		t.Fatalf("expected 5 top domains, got %d", len(stats.TopDomains))
	}
	for _, d := range stats.TopDomains {
		if d.Domain == "" {
			t.Error("expected non-empty domain")
		}
		if d.Count != 1 {
			t.Errorf("expected count=1 for %s, got %d", d.Domain, d.Count)
		}
	}

	// Size buckets should be populated.
	if len(stats.SizeBuckets) == 0 {
		t.Fatal("expected non-empty size buckets")
	}

	// Date histogram should be populated.
	if len(stats.DateHistogram) == 0 {
		t.Fatal("expected non-empty date histogram")
	}
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://www.example.com/page", "example.com"},
		{"https://example.com/page", "example.com"},
		{"http://sub.example.com:8080/path?q=1", "sub.example.com"},
		{"", ""},
		{"not-a-url", ""},
	}
	for _, tc := range tests {
		got := extractHost(tc.url)
		if got != tc.want {
			t.Errorf("extractHost(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}

func TestExtractDocTitle(t *testing.T) {
	tests := []struct {
		head string
		url  string
		want string
	}{
		{"# Hello World\nSome content", "", "Hello World"},
		{"## Section Title\nContent", "", "Section Title"},
		{"No heading here\nJust text", "https://example.com/page", "example.com"},
		{"No heading, no URL", "", ""},
	}
	for _, tc := range tests {
		got := extractDocTitle([]byte(tc.head), tc.url)
		if got != tc.want {
			t.Errorf("extractDocTitle(%q, %q) = %q, want %q", tc.head, tc.url, got, tc.want)
		}
	}
}

func TestHandleBrowseDocs_WithWARCMd(t *testing.T) {
	root := t.TempDir()
	warcMdDir := filepath.Join(root, "warc_md")
	mustMkdir(t, warcMdDir)
	warcDir := filepath.Join(root, "warc")
	mustMkdir(t, warcDir)
	writeFile(t, filepath.Join(warcDir, "CC-MAIN-x-00000.warc.gz"), 1024)

	warcMdPath := filepath.Join(warcMdDir, "00000.md.warc.gz")
	docIDs := createTestWARCMd(t, warcMdPath, 3)

	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	// Pre-scan so metadata is available.
	ctx := context.Background()
	n, err := srv.Docs.ScanShard(ctx, "", "00000", warcMdPath)
	if err != nil {
		t.Fatalf("ScanShard: %v", err)
	}
	if n != 3 {
		t.Fatalf("expected 3 docs scanned, got %d", n)
	}

	// Test browse docs endpoint.
	req := httptest.NewRequest("GET", "/api/browse?shard=00000", nil)
	w := callHandler(t, srv.handleBrowse, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp BrowseDocsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if resp.Total != 3 {
		t.Fatalf("expected total=3, got %d", resp.Total)
	}
	if len(resp.Docs) != 3 {
		t.Fatalf("expected 3 docs, got %d", len(resp.Docs))
	}

	// Verify each doc has URL, Host, Title, Date.
	for _, d := range resp.Docs {
		if d.URL == "" {
			t.Errorf("doc %s: empty URL", d.DocID)
		}
		if d.Host == "" {
			t.Errorf("doc %s: empty Host", d.DocID)
		}
		if d.Title == "" {
			t.Errorf("doc %s: empty Title", d.DocID)
		}
		if d.CrawlDate == "" {
			t.Errorf("doc %s: empty CrawlDate", d.DocID)
		}
	}

	// Test doc endpoint with offset-based read.
	docReq := httptest.NewRequest("GET", "/api/doc/00000/"+docIDs[1], nil)
	docReq.SetPathValue("shard", "00000")
	docReq.SetPathValue("docid", docIDs[1])
	dw := callHandler(t, srv.handleDoc, docReq)

	if dw.Code != 200 {
		t.Fatalf("GET /api/doc: expected 200, got %d; body: %s", dw.Code, dw.Body.String())
	}

	var docResp DocResponse
	if err := json.Unmarshal(dw.Body.Bytes(), &docResp); err != nil {
		t.Fatalf("decode doc JSON: %v", err)
	}
	if docResp.DocID != docIDs[1] {
		t.Fatalf("expected doc_id=%s, got %s", docIDs[1], docResp.DocID)
	}
	if docResp.URL == "" {
		t.Fatal("expected non-empty URL in doc response")
	}
	if docResp.Markdown == "" {
		t.Fatal("expected non-empty Markdown")
	}
	if docResp.HTML == "" {
		t.Fatal("expected non-empty HTML")
	}
	if !bytes.Contains([]byte(docResp.Markdown), []byte("Document 1")) {
		t.Fatalf("expected markdown to contain 'Document 1', got: %s", docResp.Markdown[:min(100, len(docResp.Markdown))])
	}
}

func TestHandleBrowseStats_TopDomains(t *testing.T) {
	root := t.TempDir()
	warcMdDir := filepath.Join(root, "warc_md")
	mustMkdir(t, warcMdDir)

	warcMdPath := filepath.Join(warcMdDir, "00000.md.warc.gz")
	createTestWARCMd(t, warcMdPath, 5)

	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	ctx := context.Background()
	srv.Docs.ScanShard(ctx, "", "00000", warcMdPath)

	req := httptest.NewRequest("GET", "/api/browse/stats?shard=00000", nil)
	w := callHandler(t, srv.handleBrowseStats, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var stats ShardStatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if stats.TotalDocs != 5 {
		t.Fatalf("expected 5 docs, got %d", stats.TotalDocs)
	}
	if len(stats.TopDomains) == 0 {
		t.Fatal("expected non-empty TopDomains (was the bug)")
	}
	if len(stats.SizeBuckets) == 0 {
		t.Fatal("expected non-empty SizeBuckets")
	}
}
