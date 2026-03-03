package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleOverview(t *testing.T) {
	root := t.TempDir()

	// Create a minimal data directory layout.
	warcDir := filepath.Join(root, "warc")
	mustMkdir(t, warcDir)
	writeFile(t, filepath.Join(warcDir, "00000.warc.gz"), 1024)

	mdDir := filepath.Join(root, "markdown", "00000")
	mustMkdir(t, mdDir)
	writeFile(t, filepath.Join(mdDir, "doc1.md"), 100)

	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("GET", "/api/overview", nil)
	w := httptest.NewRecorder()
	srv.handleOverview(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected application/json content type, got %q", ct)
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if result["crawl_id"] != "CC-TEST-2026" {
		t.Fatalf("expected crawl_id=CC-TEST-2026, got %v", result["crawl_id"])
	}
	if result["warc_count"].(float64) != 1 {
		t.Fatalf("expected warc_count=1, got %v", result["warc_count"])
	}
}

func TestHandleEngines(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("GET", "/api/engines", nil)
	w := httptest.NewRecorder()
	srv.handleEngines(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	engines, ok := result["engines"].([]any)
	if !ok {
		t.Fatalf("expected engines to be an array, got %T", result["engines"])
	}

	// The engine list may be empty if no drivers are registered in the test binary,
	// but it should always be a valid array (not nil/null).
	_ = engines
}

func TestHandleJobs_Empty(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("GET", "/api/jobs", nil)
	w := httptest.NewRecorder()
	srv.handleListJobs(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	jobs, ok := result["jobs"].([]any)
	if !ok {
		t.Fatalf("expected jobs to be an array, got %T", result["jobs"])
	}
	if len(jobs) != 0 {
		t.Fatalf("expected empty jobs array, got %d items", len(jobs))
	}
}

func TestHandleGetJob_NotFound(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("GET", "/api/jobs/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	srv.handleGetJob(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleCreateJob(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	body := `{"type":"download","crawl":"CC-MAIN-2026-08","files":"0"}`
	req := httptest.NewRequest("POST", "/api/jobs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleCreateJob(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}

	var job Job
	if err := json.Unmarshal(w.Body.Bytes(), &job); err != nil {
		t.Fatalf("failed to decode job JSON: %v", err)
	}
	if job.ID == "" {
		t.Fatal("expected non-empty job ID")
	}
	if job.Type != "download" {
		t.Fatalf("expected type=download, got %q", job.Type)
	}
	if job.Config.CrawlID != "CC-MAIN-2026-08" {
		t.Fatalf("expected config.crawl=CC-MAIN-2026-08, got %q", job.Config.CrawlID)
	}
}

func TestHandleCreateJob_MissingType(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	body := `{"crawl":"CC-MAIN-2026-08"}`
	req := httptest.NewRequest("POST", "/api/jobs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleCreateJob(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCancelJob_NotFound(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("DELETE", "/api/jobs/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	srv.handleCancelJob(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleCrawlData(t *testing.T) {
	root := t.TempDir()

	// Create a minimal data layout.
	warcDir := filepath.Join(root, "warc")
	mustMkdir(t, warcDir)
	writeFile(t, filepath.Join(warcDir, "00000.warc.gz"), 2048)

	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("GET", "/api/crawl/CC-TEST-2026/data", nil)
	req.SetPathValue("id", "CC-TEST-2026")
	w := httptest.NewRecorder()
	srv.handleCrawlData(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if result["crawl_id"] != "CC-TEST-2026" {
		t.Fatalf("expected crawl_id=CC-TEST-2026, got %v", result["crawl_id"])
	}
}

func TestDashboardRoutes_Registered(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)
	handler := srv.Handler()

	// Test that dashboard routes are registered by sending requests.
	// The overview endpoint should return 200.
	req := httptest.NewRequest("GET", "/api/overview", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("GET /api/overview: expected 200, got %d", w.Code)
	}
}

func TestNonDashboardRoutes_NoDashboard(t *testing.T) {
	root := t.TempDir()
	// Use New (not NewDashboard) — dashboard routes should not be registered.
	srv := New("test-engine", "CC-TEST-2026", "", root)

	// Verify Hub and Jobs are nil (no dashboard capability).
	if srv.Hub != nil {
		t.Fatal("expected Hub to be nil when created via New()")
	}
	if srv.Jobs != nil {
		t.Fatal("expected Jobs to be nil when created via New()")
	}

	handler := srv.Handler()

	// Without dashboard, /api/overview falls through to GET / which returns HTML.
	// Verify it does NOT return JSON (i.e., the dashboard handler is not registered).
	req := httptest.NewRequest("GET", "/api/overview", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		t.Fatal("GET /api/overview should not return JSON when created via New()")
	}
}

func TestNewDashboard_SetsFields(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("bleve", "CC-MAIN-2026-08", "http://localhost:7700", root)

	if srv.EngineName != "bleve" {
		t.Fatalf("expected EngineName=bleve, got %q", srv.EngineName)
	}
	if srv.CrawlID != "CC-MAIN-2026-08" {
		t.Fatalf("expected CrawlID=CC-MAIN-2026-08, got %q", srv.CrawlID)
	}
	if srv.CrawlDir != root {
		t.Fatalf("expected CrawlDir=%s, got %s", root, srv.CrawlDir)
	}
	if srv.Hub == nil {
		t.Fatal("expected Hub to be non-nil")
	}
	if srv.Jobs == nil {
		t.Fatal("expected Jobs to be non-nil")
	}
	if srv.Addr != "http://localhost:7700" {
		t.Fatalf("expected Addr=http://localhost:7700, got %q", srv.Addr)
	}

	// FTSBase and MDBase should still be set via New().
	expectedFTS := filepath.Join(root, "fts", "bleve")
	if srv.FTSBase != expectedFTS {
		t.Fatalf("expected FTSBase=%s, got %s", expectedFTS, srv.FTSBase)
	}
	expectedMD := filepath.Join(root, "markdown")
	if srv.MDBase != expectedMD {
		t.Fatalf("expected MDBase=%s, got %s", expectedMD, srv.MDBase)
	}
}

func TestHandleOverview_EmptyDir(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("GET", "/api/overview", nil)
	w := httptest.NewRecorder()
	srv.handleOverview(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result DataSummary
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if result.CrawlID != "CC-TEST-2026" {
		t.Fatalf("expected crawl_id=CC-TEST-2026, got %q", result.CrawlID)
	}
	if result.WARCCount != 0 {
		t.Fatalf("expected warc_count=0, got %d", result.WARCCount)
	}
	// Maps should be non-nil for clean JSON.
	if result.PackFormats == nil {
		t.Fatal("expected PackFormats to be non-nil")
	}
	if result.FTSEngines == nil {
		t.Fatal("expected FTSEngines to be non-nil")
	}
}

// TestHandleListJobs_WithJobs verifies that created jobs appear in the list endpoint.
func TestHandleListJobs_WithJobs(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	// Create two jobs via the JobManager directly.
	srv.Jobs.Create(JobConfig{Type: "download", Files: "0"})
	srv.Jobs.Create(JobConfig{Type: "index", Engine: "bleve"})

	req := httptest.NewRequest("GET", "/api/jobs", nil)
	w := httptest.NewRecorder()
	srv.handleListJobs(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	jobs, ok := result["jobs"].([]any)
	if !ok {
		t.Fatalf("expected jobs to be an array, got %T", result["jobs"])
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
}

// TestHandleCreateJob_InvalidJSON verifies that invalid JSON body returns 400.
func TestHandleCreateJob_InvalidJSON(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("POST", "/api/jobs", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleCreateJob(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d; body: %s", w.Code, w.Body.String())
	}
}

// TestParseFileSelector verifies the helper function that executors use.
func TestParseFileSelector(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		total   int
		want    []int
		wantErr bool
	}{
		{"single", "0", 10, []int{0}, false},
		{"single high", "5", 10, []int{5}, false},
		{"range", "2-4", 10, []int{2, 3, 4}, false},
		{"all", "all", 3, []int{0, 1, 2}, false},
		{"empty", "", 3, []int{0, 1, 2}, false},
		{"out of bounds", "10", 5, nil, true},
		{"bad range", "3-1", 5, nil, true},
		{"invalid", "abc", 5, nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseFileSelector(tc.input, tc.total)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("expected %d indices, got %d: %v", len(tc.want), len(got), got)
			}
			for i, v := range got {
				if v != tc.want[i] {
					t.Fatalf("index %d: expected %d, got %d", i, tc.want[i], v)
				}
			}
		})
	}
}

// TestWarcIndexFromPath verifies the helper function used by executors.
func TestWarcIndexFromPath(t *testing.T) {
	tests := []struct {
		path     string
		fallback int
		want     string
	}{
		{"crawl-data/CC-MAIN-2026-08/segments/1738964620578.15/warc/CC-MAIN-20260206181458-20260206211458-00000.warc.gz", 0, "00000"},
		{"CC-MAIN-20260206181458-20260206211458-00042.warc.gz", 0, "00042"},
		{"some-other-file.warc.gz", 7, "00007"},
		{"", 3, "00003"},
	}

	for _, tc := range tests {
		got := warcIndexFromPath(tc.path, tc.fallback)
		if got != tc.want {
			t.Errorf("warcIndexFromPath(%q, %d) = %q, want %q", tc.path, tc.fallback, got, tc.want)
		}
	}
}

// Compile-time check: ensure handler methods satisfy http.HandlerFunc signature.
var _ http.HandlerFunc = (*Server)(nil).handleOverview
var _ http.HandlerFunc = (*Server)(nil).handleEngines
var _ http.HandlerFunc = (*Server)(nil).handleListJobs
var _ http.HandlerFunc = (*Server)(nil).handleGetJob
var _ http.HandlerFunc = (*Server)(nil).handleCreateJob
var _ http.HandlerFunc = (*Server)(nil).handleCancelJob
var _ http.HandlerFunc = (*Server)(nil).handleCrawlData
var _ http.HandlerFunc = (*Server)(nil).handleCrawlWarcs
var _ http.HandlerFunc = (*Server)(nil).handleCrawls
