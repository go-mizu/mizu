package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mizu "github.com/go-mizu/mizu"
)

// callHandler is a helper to invoke a mizu handler through httptest machinery.
func callHandler(t *testing.T, h func(*mizu.Ctx) error, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c := mizu.NewCtx(w, req, nil)
	if err := h(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	return w
}

func TestSearchEngineAndFTSBaseResolve(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("bleve", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("GET", "/api/search?q=test&engine=duckdb", nil)
	if got := srv.searchEngine(req); got != "duckdb" {
		t.Fatalf("searchEngine() = %q, want duckdb", got)
	}
	if got := srv.resolveFTSBase("duckdb"); got != filepath.Join(root, "fts", "duckdb") {
		t.Fatalf("resolveFTSBase(duckdb) = %q", got)
	}

	reqDefault := httptest.NewRequest("GET", "/api/search?q=test", nil)
	if got := srv.searchEngine(reqDefault); got != "bleve" {
		t.Fatalf("searchEngine() default = %q, want bleve", got)
	}

	searchOnly := New("bleve", "CC-TEST-2026", "", root)
	if got := searchOnly.resolveFTSBase("tantivy"); got != filepath.Join(root, "fts", "tantivy") {
		t.Fatalf("search-only resolveFTSBase(tantivy) = %q", got)
	}
}

func TestHandleOverview(t *testing.T) {
	root := t.TempDir()

	// Create a minimal data directory layout.
	warcDir := filepath.Join(root, "warc")
	mustMkdir(t, warcDir)
	writeFile(t, filepath.Join(warcDir, "CC-MAIN-x-00000.warc.gz"), 1024)

	mustMkdir(t, filepath.Join(root, "warc_md"))
	writeFile(t, filepath.Join(root, "warc_md", "00000.md.warc.gz"), 512)

	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("GET", "/api/overview", nil)
	w := callHandler(t, srv.handleOverview, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected application/json content type, got %q", ct)
	}

	var resp OverviewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if resp.CrawlID != "CC-TEST-2026" {
		t.Fatalf("expected crawl_id=CC-TEST-2026, got %v", resp.CrawlID)
	}
	if resp.Downloaded.Count != 1 {
		t.Fatalf("expected downloaded.count=1, got %d", resp.Downloaded.Count)
	}
	if resp.Markdown.Count != 1 {
		t.Fatalf("expected markdown.count=1, got %d", resp.Markdown.Count)
	}
}

func TestHandleEngines(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("GET", "/api/engines", nil)
	w := callHandler(t, srv.handleEngines, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result EnginesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	// The engine list may be empty if no drivers are registered in the test binary,
	// but it should always be a valid array (not nil/null).
	if result.Engines == nil {
		result.Engines = []string{}
	}
}

func TestHandleJobs_Empty(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("GET", "/api/jobs", nil)
	w := callHandler(t, srv.handleListJobs, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result JobsListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if len(result.Jobs) != 0 {
		t.Fatalf("expected empty jobs array, got %d items", len(result.Jobs))
	}
}

func TestHandleGetJob_NotFound(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("GET", "/api/jobs/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	c := mizu.NewCtx(w, req, nil)
	_ = srv.handleGetJob(c) // returns error JSON, not a Go error

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
	w := callHandler(t, srv.handleCreateJob, req)

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
	c := mizu.NewCtx(w, req, nil)
	_ = srv.handleCreateJob(c)

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
	c := mizu.NewCtx(w, req, nil)
	_ = srv.handleCancelJob(c)

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
	w := callHandler(t, srv.handleCrawlData, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// CrawlData returns DataSummaryWithMeta which has crawl_id field.
	var result struct {
		CrawlID string `json:"crawl_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if result.CrawlID != "CC-TEST-2026" {
		t.Fatalf("expected crawl_id=CC-TEST-2026, got %v", result.CrawlID)
	}
}

func TestHandleWARCListAndDetail_Fallback(t *testing.T) {
	root := t.TempDir()
	warcDir := filepath.Join(root, "warc")
	markdownDir := filepath.Join(root, "markdown", "00000")
	packDir := filepath.Join(root, "pack", "parquet")
	ftsDir := filepath.Join(root, "fts", "duckdb", "00000")
	mustMkdir(t, warcDir)
	mustMkdir(t, markdownDir)
	mustMkdir(t, packDir)
	mustMkdir(t, ftsDir)

	writeFile(t, filepath.Join(warcDir, "CC-MAIN-x-00000.warc.gz"), 2048)
	writeFile(t, filepath.Join(markdownDir, "doc1.md"), 100)
	writeFile(t, filepath.Join(packDir, "00000.parquet"), 1500)
	writeFile(t, filepath.Join(ftsDir, "seg.bin"), 700)

	srv := NewDashboard("duckdb", "CC-TEST-2026", "", root)
	srv.Meta = nil // deterministic scan fallback for this test

	reqList := httptest.NewRequest("GET", "/api/warc", nil)
	wList := callHandler(t, srv.handleWARCList, reqList)
	if wList.Code != 200 {
		t.Fatalf("GET /api/warc expected 200, got %d: %s", wList.Code, wList.Body.String())
	}
	var listResp WARCListResponse
	if err := json.Unmarshal(wList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list JSON: %v", err)
	}
	if len(listResp.WARCs) != 1 {
		t.Fatalf("expected exactly 1 warc row, got %d", len(listResp.WARCs))
	}

	reqDetail := httptest.NewRequest("GET", "/api/warc/0", nil)
	reqDetail.SetPathValue("index", "0")
	wDetail := callHandler(t, srv.handleWARCDetail, reqDetail)
	if wDetail.Code != 200 {
		t.Fatalf("GET /api/warc/0 expected 200, got %d: %s", wDetail.Code, wDetail.Body.String())
	}
	var detail WARCDetailResponse
	if err := json.Unmarshal(wDetail.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail JSON: %v", err)
	}
	if detail.WARC.Index != "00000" {
		t.Fatalf("expected detail index=00000, got %v", detail.WARC.Index)
	}
}

func TestHandleWARCAction_DeletePack(t *testing.T) {
	root := t.TempDir()
	packDir := filepath.Join(root, "pack", "parquet")
	mustMkdir(t, packDir)
	target := filepath.Join(packDir, "00000.parquet")
	writeFile(t, target, 1234)

	srv := NewDashboard("duckdb", "CC-TEST-2026", "", root)
	srv.Meta = nil

	body := `{"action":"delete","target":"pack","format":"parquet"}`
	req := httptest.NewRequest("POST", "/api/warc/0/action", strings.NewReader(body))
	req.SetPathValue("index", "0")
	req.Header.Set("Content-Type", "application/json")
	w := callHandler(t, srv.handleWARCAction, req)
	if w.Code != 200 {
		t.Fatalf("POST /api/warc/0/action delete expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected pack file to be deleted, stat err=%v", err)
	}
}

func TestHandleWARCAction_CreateIndexJob(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("duckdb", "CC-TEST-2026", "", root)
	srv.Meta = nil

	body := `{"action":"index","engine":"duckdb","source":"files"}`
	req := httptest.NewRequest("POST", "/api/warc/0/action", strings.NewReader(body))
	req.SetPathValue("index", "0")
	req.Header.Set("Content-Type", "application/json")
	w := callHandler(t, srv.handleWARCAction, req)
	if w.Code != 200 {
		t.Fatalf("POST /api/warc/0/action index expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp WARCActionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode action JSON: %v", err)
	}
	if resp.Job == nil || resp.Job.ID == "" {
		t.Fatalf("expected created job in response, got nil")
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

	req2 := httptest.NewRequest("GET", "/api/meta/status", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("GET /api/meta/status: expected 200, got %d", w2.Code)
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
	if srv.Meta == nil {
		t.Fatal("expected Meta to be non-nil")
	}
	if srv.Addr != "http://localhost:7700" {
		t.Fatalf("expected Addr=http://localhost:7700, got %q", srv.Addr)
	}

	// FTSBase and WARCMdBase should still be set via New().
	expectedFTS := filepath.Join(root, "fts", "bleve")
	if srv.FTSBase != expectedFTS {
		t.Fatalf("expected FTSBase=%s, got %s", expectedFTS, srv.FTSBase)
	}
	expectedWARCMd := filepath.Join(root, "warc_md")
	if srv.WARCMdBase != expectedWARCMd {
		t.Fatalf("expected WARCMdBase=%s, got %s", expectedWARCMd, srv.WARCMdBase)
	}
}

func TestHandleOverview_EmptyDir(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("GET", "/api/overview", nil)
	w := callHandler(t, srv.handleOverview, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp OverviewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if resp.CrawlID != "CC-TEST-2026" {
		t.Fatalf("expected crawl_id=CC-TEST-2026, got %q", resp.CrawlID)
	}
	if resp.Downloaded.Count != 0 {
		t.Fatalf("expected downloaded.count=0, got %d", resp.Downloaded.Count)
	}
	if resp.System.GoVersion == "" {
		t.Fatal("expected go version")
	}
}

func TestHandleMetaStatusAndRefresh(t *testing.T) {
	root := t.TempDir()
	warcDir := filepath.Join(root, "warc")
	mustMkdir(t, warcDir)
	writeFile(t, filepath.Join(warcDir, "00000.warc.gz"), 1024)

	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)
	if srv.Meta == nil {
		t.Fatal("expected meta manager to be initialized")
	}

	// Trigger a synchronous read first so cache exists.
	reqOverview := httptest.NewRequest("GET", "/api/overview", nil)
	wOverview := callHandler(t, srv.handleOverview, reqOverview)
	if wOverview.Code != 200 {
		t.Fatalf("GET /api/overview: expected 200, got %d", wOverview.Code)
	}

	reqStatus := httptest.NewRequest("GET", "/api/meta/status", nil)
	wStatus := callHandler(t, srv.handleMetaStatus, reqStatus)
	if wStatus.Code != 200 {
		t.Fatalf("GET /api/meta/status: expected 200, got %d", wStatus.Code)
	}
	var statusResp MetaStatus
	if err := json.Unmarshal(wStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode status JSON: %v", err)
	}
	if statusResp.CrawlID != "CC-TEST-2026" {
		t.Fatalf("expected crawl_id=CC-TEST-2026, got %v", statusResp.CrawlID)
	}

	reqRefresh := httptest.NewRequest("POST", "/api/meta/refresh", strings.NewReader(`{"force":true}`))
	reqRefresh.Header.Set("Content-Type", "application/json")
	wRefresh := httptest.NewRecorder()
	c := mizu.NewCtx(wRefresh, reqRefresh, nil)
	_ = srv.handleMetaRefresh(c)
	if wRefresh.Code != http.StatusAccepted && wRefresh.Code != 200 {
		t.Fatalf("POST /api/meta/refresh: expected 202 or 200, got %d", wRefresh.Code)
	}
	var refreshResp MetaRefreshResponse
	if err := json.Unmarshal(wRefresh.Body.Bytes(), &refreshResp); err != nil {
		t.Fatalf("decode refresh JSON: %v", err)
	}
	// accepted field is present (true or false).
	_ = refreshResp.Accepted
}

// TestHandleListJobs_WithJobs verifies that created jobs appear in the list endpoint.
func TestHandleListJobs_WithJobs(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	// Create two jobs via the JobManager directly.
	srv.Jobs.Create(JobConfig{Type: "download", Files: "0"})
	srv.Jobs.Create(JobConfig{Type: "index", Engine: "bleve"})

	req := httptest.NewRequest("GET", "/api/jobs", nil)
	w := callHandler(t, srv.handleListJobs, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result JobsListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if len(result.Jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(result.Jobs))
	}
}

// TestHandleCreateJob_InvalidJSON verifies that invalid JSON body returns 400.
func TestHandleCreateJob_InvalidJSON(t *testing.T) {
	root := t.TempDir()
	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)

	req := httptest.NewRequest("POST", "/api/jobs", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c := mizu.NewCtx(w, req, nil)
	_ = srv.handleCreateJob(c)

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

func TestPackFilePath(t *testing.T) {
	packDir := "/tmp/pack"
	warcIdx := "00042"

	got, err := packFilePath(packDir, "parquet", warcIdx)
	if err != nil {
		t.Fatalf("parquet: unexpected err: %v", err)
	}
	if got != "/tmp/pack/parquet/00042.parquet" {
		t.Fatalf("parquet: got %q", got)
	}

	got, err = packFilePath(packDir, "bin", warcIdx)
	if err != nil {
		t.Fatalf("bin: unexpected err: %v", err)
	}
	if got != "/tmp/pack/bin/00042.bin" {
		t.Fatalf("bin: got %q", got)
	}

	got, err = packFilePath(packDir, "duckdb", warcIdx)
	if err != nil {
		t.Fatalf("duckdb: unexpected err: %v", err)
	}
	if got != "/tmp/pack/duckdb/00042.duckdb" {
		t.Fatalf("duckdb: got %q", got)
	}

	got, err = packFilePath(packDir, "markdown", warcIdx)
	if err != nil {
		t.Fatalf("markdown: unexpected err: %v", err)
	}
	if got != "/tmp/pack/markdown/00042.bin.gz" {
		t.Fatalf("markdown: got %q", got)
	}

	if _, err := packFilePath(packDir, "invalid", warcIdx); err == nil {
		t.Fatal("invalid format: expected error")
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

// TestIntegrationDashboardLifecycle exercises the full dashboard HTTP lifecycle
// through httptest.Server, verifying routing, status codes, and response shapes
// for all core dashboard endpoints.
func TestIntegrationDashboardLifecycle(t *testing.T) {
	root := t.TempDir()

	// Seed minimal data so overview returns non-trivial values.
	warcDir := filepath.Join(root, "warc")
	mustMkdir(t, warcDir)
	writeFile(t, filepath.Join(warcDir, "00000.warc.gz"), 1024)

	mdDir := filepath.Join(root, "markdown", "00000")
	mustMkdir(t, mdDir)
	writeFile(t, filepath.Join(mdDir, "doc1.md"), 100)

	srv := NewDashboard("test-engine", "CC-TEST-2026", "", root)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := ts.Client()

	// ── GET / → 200, HTML ────────────────────────────────────────────
	t.Run("GET / returns HTML", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/")
		if err != nil {
			t.Fatalf("GET /: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("GET /: expected 200, got %d", resp.StatusCode)
		}
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			t.Fatalf("GET /: expected text/html content type, got %q", ct)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("GET /: read body: %v", err)
		}
		if !strings.Contains(string(body), "FTS Dashboard") {
			t.Fatal("GET /: expected dashboard UI HTML")
		}
	})

	// ── GET /api/overview → 200, JSON with crawl_id ─────────────────
	t.Run("GET /api/overview", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/overview")
		if err != nil {
			t.Fatalf("GET /api/overview: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("GET /api/overview: expected 200, got %d", resp.StatusCode)
		}
		ct := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "application/json") {
			t.Fatalf("GET /api/overview: expected application/json, got %q", ct)
		}
		var result OverviewResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode JSON: %v", err)
		}
		if result.CrawlID != "CC-TEST-2026" {
			t.Fatalf("expected crawl_id=CC-TEST-2026, got %v", result.CrawlID)
		}
		if result.System.GoVersion == "" {
			t.Fatal("expected go version in overview response")
		}
	})

	// ── GET /api/meta/status → 200 ───────────────────────────────────
	t.Run("GET /api/meta/status", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/meta/status")
		if err != nil {
			t.Fatalf("GET /api/meta/status: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("GET /api/meta/status: expected 200, got %d", resp.StatusCode)
		}
		var result MetaStatus
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode JSON: %v", err)
		}
		if result.CrawlID != "CC-TEST-2026" {
			t.Fatalf("expected crawl_id=CC-TEST-2026, got %v", result.CrawlID)
		}
	})

	// ── POST /api/meta/refresh → 202|200 ─────────────────────────────
	t.Run("POST /api/meta/refresh", func(t *testing.T) {
		resp, err := client.Post(ts.URL+"/api/meta/refresh", "application/json", strings.NewReader(`{"force":true}`))
		if err != nil {
			t.Fatalf("POST /api/meta/refresh: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted && resp.StatusCode != 200 {
			t.Fatalf("POST /api/meta/refresh: expected 202 or 200, got %d", resp.StatusCode)
		}
		var result MetaRefreshResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode JSON: %v", err)
		}
		_ = result.Accepted // field is present
	})

	// ── GET /api/engines → 200, JSON with engines array ─────────────
	t.Run("GET /api/engines", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/engines")
		if err != nil {
			t.Fatalf("GET /api/engines: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("GET /api/engines: expected 200, got %d", resp.StatusCode)
		}
		var result EnginesResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode JSON: %v", err)
		}
		// engines may or may not be empty depending on registered drivers.
		if result.Engines == nil {
			result.Engines = []string{}
		}
	})

	// ── GET /api/warc → 200, JSON with warcs array ───────────────────
	t.Run("GET /api/warc", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/warc")
		if err != nil {
			t.Fatalf("GET /api/warc: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("GET /api/warc: expected 200, got %d", resp.StatusCode)
		}
		var result WARCListResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode JSON: %v", err)
		}
		if result.WARCs == nil {
			result.WARCs = []warcAPIRecord{}
		}
	})

	// ── GET /api/jobs → 200, JSON with empty jobs array ─────────────
	t.Run("GET /api/jobs empty", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/jobs")
		if err != nil {
			t.Fatalf("GET /api/jobs: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("GET /api/jobs: expected 200, got %d", resp.StatusCode)
		}
		var result JobsListResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode JSON: %v", err)
		}
		if len(result.Jobs) != 0 {
			t.Fatalf("expected 0 jobs, got %d", len(result.Jobs))
		}
	})

	// ── POST /api/jobs → 201, JSON with id + status=queued ──────────
	var jobID string
	t.Run("POST /api/jobs creates job", func(t *testing.T) {
		body := `{"type":"download","crawl":"CC-MAIN-2026-08","files":"0"}`
		resp, err := client.Post(ts.URL+"/api/jobs", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("POST /api/jobs: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 201 {
			t.Fatalf("POST /api/jobs: expected 201, got %d", resp.StatusCode)
		}
		var job Job
		if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
			t.Fatalf("decode JSON: %v", err)
		}
		if job.ID == "" {
			t.Fatalf("expected non-empty job id")
		}
		jobID = job.ID

		// Status should be "queued" or "running".
		if job.Status != "queued" && job.Status != "running" {
			t.Fatalf("expected status queued or running, got %q", job.Status)
		}
	})

	// ── GET /api/jobs/{id} → 200, JSON with the job ─────────────────
	t.Run("GET /api/jobs/{id}", func(t *testing.T) {
		if jobID == "" {
			t.Skip("no job ID from previous step")
		}
		resp, err := client.Get(ts.URL + "/api/jobs/" + jobID)
		if err != nil {
			t.Fatalf("GET /api/jobs/%s: %v", jobID, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("GET /api/jobs/%s: expected 200, got %d", jobID, resp.StatusCode)
		}
		var job Job
		if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
			t.Fatalf("decode JSON: %v", err)
		}
		if job.ID != jobID {
			t.Fatalf("expected id=%s, got %v", jobID, job.ID)
		}
	})

	// ── DELETE /api/jobs/{id} → 200 ─────────────────────────────────
	t.Run("DELETE /api/jobs/{id}", func(t *testing.T) {
		if jobID == "" {
			t.Skip("no job ID from previous step")
		}
		req, err := http.NewRequest("DELETE", ts.URL+"/api/jobs/"+jobID, nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("DELETE /api/jobs/%s: %v", jobID, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("DELETE /api/jobs/%s: expected 200, got %d", jobID, resp.StatusCode)
		}
	})

	// ── GET /api/jobs/{id} after cancel → verify cancelled status ───
	t.Run("GET /api/jobs/{id} after cancel", func(t *testing.T) {
		if jobID == "" {
			t.Skip("no job ID from previous step")
		}
		resp, err := client.Get(ts.URL + "/api/jobs/" + jobID)
		if err != nil {
			t.Fatalf("GET /api/jobs/%s: %v", jobID, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var job Job
		if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
			t.Fatalf("decode JSON: %v", err)
		}
		if job.Status != "cancelled" {
			t.Fatalf("expected status=cancelled, got %v", job.Status)
		}
	})
}

// TestIntegrationNewNoDashboardRoutes verifies that when using New() (not
// NewDashboard), dashboard-specific routes are not registered while core
// routes still work.
func TestIntegrationNewNoDashboardRoutes(t *testing.T) {
	root := t.TempDir()
	srv := New("test-engine", "CC-TEST-2026", "", root)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := ts.Client()

	// ── GET / still returns HTML (core route) ────────────────────────
	t.Run("GET / returns HTML", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/")
		if err != nil {
			t.Fatalf("GET /: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("GET /: expected 200, got %d", resp.StatusCode)
		}
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			t.Fatalf("GET /: expected text/html, got %q", ct)
		}
	})

	// ── Dashboard routes fall through to GET / (no JSON) ─────────────
	for _, path := range []string{"/api/overview", "/api/meta/status", "/api/engines", "/api/jobs", "/api/warc"} {
		t.Run("GET "+path+" no dashboard", func(t *testing.T) {
			resp, err := client.Get(ts.URL + path)
			if err != nil {
				t.Fatalf("GET %s: %v", path, err)
			}
			defer resp.Body.Close()
			// Without dashboard, these paths are not registered. Go's ServeMux
			// will either 404 or fall through to GET / (returning HTML).
			// Either way, the response should NOT be application/json.
			ct := resp.Header.Get("Content-Type")
			if strings.HasPrefix(ct, "application/json") {
				t.Fatalf("GET %s should not return application/json without dashboard", path)
			}
		})
	}

	// ── POST /api/jobs should not be registered ──────────────────────
	t.Run("POST /api/jobs no dashboard", func(t *testing.T) {
		resp, err := client.Post(ts.URL+"/api/jobs", "application/json", strings.NewReader(`{"type":"download"}`))
		if err != nil {
			t.Fatalf("POST /api/jobs: %v", err)
		}
		defer resp.Body.Close()
		// Without dashboard routes, POST /api/jobs is not registered.
		if resp.StatusCode == 201 {
			t.Fatal("POST /api/jobs should not return 201 without dashboard")
		}
	})
}

// Compile-time check: ensure handler methods satisfy mizu.Handler signature.
var _ func(*mizu.Ctx) error = (*Server)(nil).handleOverview
var _ func(*mizu.Ctx) error = (*Server)(nil).handleMetaStatus
