// Package web provides an embedded HTTP server for browsing and searching
// Common Crawl FTS indexes through a modern web GUI.
package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	mizu "github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/api"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/scrape"
	webstore "github.com/go-mizu/mizu/blueprints/search/pkg/index/web/store"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed static
var staticFS embed.FS

// jsHashOnce / jsHashMap cache 8-char SHA-256 fingerprints for each JS file.
// Used to append ?v=<hash> to <script src> tags so browsers invalidate stale
// files after a deploy while allowing max-age=immutable caching.
var (
	jsHashOnce sync.Once
	jsHashMap  map[string]string // basename → 8 hex chars
)

func buildJSHashes() {
	jsHashMap = make(map[string]string)
	entries, err := fs.ReadDir(staticFS, "static/js")
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		data, err := staticFS.ReadFile("static/js/" + e.Name())
		if err != nil {
			continue
		}
		h := sha256.Sum256(data)
		jsHashMap[e.Name()] = fmt.Sprintf("%x", h[:4]) // 8 hex chars
	}
}

// ── Response type aliases (types defined in api package) ────────────────

// Response type aliases - types now defined in api/
// Note: OverviewResponse is kept in overview.go (web package), not aliased here.
type (
	SearchResponse              = api.SearchResponse
	StatsResponse               = api.StatsResponse
	DocResponse                 = api.DocResponse
	BrowseShardsResponse        = api.BrowseShardsResponse
	BrowseDocsResponse          = api.BrowseDocsResponse
	MetaScanDocsResponse        = api.MetaScanDocsResponse
	MetaRefreshResponse         = api.MetaRefreshResponse
	EnginesResponse             = api.EnginesResponse
	WARCListResponse            = api.WARCListResponse
	WARCDetailResponse          = api.WARCDetailResponse
	WARCActionResponse          = api.WARCActionResponse
	BrowseExportParquetResponse = api.BrowseExportParquetResponse
)

// errResp is the standard JSON error shape (local copy since api.errResp is unexported).
type errResp struct {
	Error string `json:"error"`
}

// ── Server ──────────────────────────────────────────────────────────────

// Server serves the FTS web GUI and JSON API.
type Server struct {
	EngineName    string
	CrawlID       string
	Addr          string // external engine address
	FTSBase       string // ~/data/common-crawl/{crawlID}/fts/{engine}
	WARCBase      string // ~/data/common-crawl/{crawlID}/warc
	WARCMdBase    string // ~/data/common-crawl/{crawlID}/warc_md
	CrawlDir      string // ~/data/common-crawl/{crawlID} — set by NewDashboard
	Hub           *pipeline.Hub
	Jobs          *pipeline.Manager
	Meta          *MetaManager
	Docs          *DocStore      // per-document browse metadata (dashboard only)
	DomainStore   *DomainStore   // cross-shard domain count cache (dashboard only)
	CCDomainStore *CCDomainStore // Common Crawl CDX domain cache (separate DB)
	Scrape        scrape.Store   // per-domain crawler result store (dashboard only)

	manifestTotal int             // cached count of WARCs in CC manifest
	crawlSize     *crawlSizeCache // async-cached dirSize(crawlDir)

	md goldmark.Markdown
}

// DashboardOptions configures dashboard-only behavior.
type DashboardOptions struct {
	MetaDriver      string
	MetaDSN         string
	MetaRefreshTTL  time.Duration
	MetaPrewarm     bool
	MetaBusyTimeout time.Duration
}

// New creates a Server for the given crawl and engine.
func New(engineName, crawlID, addr, baseDir string) *Server {
	return &Server{
		EngineName: engineName,
		CrawlID:    crawlID,
		Addr:       addr,
		FTSBase:    filepath.Join(baseDir, "fts", engineName),
		WARCBase:   filepath.Join(baseDir, "warc"),
		WARCMdBase: filepath.Join(baseDir, "warc_md"),
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	}
}

// NewDashboard creates a Server with dashboard capabilities (WebSocket hub,
// job manager, and data directory scanning). The baseDir should be the crawl
// data directory (e.g. ~/data/common-crawl/{crawlID}).
func NewDashboard(engineName, crawlID, addr, baseDir string) *Server {
	return NewDashboardWithOptions(engineName, crawlID, addr, baseDir, DashboardOptions{
		MetaDriver:      defaultMetaDriver,
		MetaRefreshTTL:  defaultMetaRefreshTTL,
		MetaPrewarm:     true,
		MetaBusyTimeout: 5 * time.Second,
	})
}

// NewDashboardWithOptions creates a dashboard server with metadata cache config.
func NewDashboardWithOptions(engineName, crawlID, addr, baseDir string, opts DashboardOptions) *Server {
	s := New(engineName, crawlID, addr, baseDir)
	s.CrawlDir = baseDir
	s.Hub = pipeline.NewHub()
	s.Jobs = pipeline.NewManager(s.Hub, baseDir, crawlID)

	metaCfg := MetaConfig{
		Driver:      opts.MetaDriver,
		DSN:         opts.MetaDSN,
		RefreshTTL:  opts.MetaRefreshTTL,
		Prewarm:     opts.MetaPrewarm,
		BusyTimeout: opts.MetaBusyTimeout,
		ActiveCrawl: crawlID,
		ActiveDir:   baseDir,
		CommonCrawl: filepath.Dir(baseDir),
	}
	meta, err := NewMetaManager(context.Background(), metaCfg)
	if err != nil {
		logErrorf("meta manager init failed driver=%s err=%v; falling back to scan mode", opts.MetaDriver, err)
		// Fallback to scan mode if metadata store cannot initialize.
		meta, _ = NewMetaManager(context.Background(), MetaConfig{
			Driver:      "none",
			ActiveCrawl: crawlID,
			ActiveDir:   baseDir,
			CommonCrawl: filepath.Dir(baseDir),
		})
	}
	s.Meta = meta

	// Wire persistence store to Manager and load history.
	if store := meta.Store(); store != nil {
		s.Jobs.SetStore(store)
		s.Jobs.LoadHistory(context.Background())
	}

	s.Jobs.SetCompleteHook(func(_ *pipeline.Job, crawlID, crawlDir string) {
		if s.Meta != nil {
			s.Meta.TriggerRefresh(crawlID, crawlDir, true)
		}
		// Trigger doc scan after pack/index jobs complete.
		if s.Docs != nil {
			go func() {
				if _, err := s.Docs.ScanAll(context.Background(), crawlID, crawlDir); err != nil {
					logErrorf("doc_store: post-job scan failed crawl=%s err=%v", crawlID, err)
				}
				broadcastShardScan(s.Hub, "*")
			}()
		}
	})

	// Start async crawl dir size cache (avoids blocking overview on 250K+ file walk).
	s.crawlSize = newCrawlSizeCache(baseDir)

	// Initialize per-document browse metadata store (per-shard DuckDB + in-memory cache).
	if docs, err := NewDocStore(s.WARCMdBase); err != nil {
		logErrorf("doc_store: init failed dir=%s err=%v (browse metadata disabled)", s.WARCMdBase, err)
	} else {
		s.Docs = docs
		logInfof("doc_store: opened dir=%s (per-shard duckdb)", s.WARCMdBase)
	}

	// Initialize domain count cache (lazily syncs from parquet files).
	s.DomainStore = NewDomainStore(baseDir)
	s.CCDomainStore = NewCCDomainStore(baseDir)

	// Initialize per-domain scrape store.
	{
		home, _ := os.UserHomeDir()
		s.Scrape = scrape.NewStore(filepath.Join(home, "data", "crawler"))
		s.Jobs.SetScrapeInvalidator(s.Scrape.InvalidateCache)
	}

	// Fetch manifest total in background for overview pipeline progress.
	go func() {
		client := cc.NewClient("", 4)
		paths, err := client.DownloadManifest(context.Background(), crawlID, "warc.paths.gz")
		if err != nil {
			logErrorf("manifest fetch failed crawl=%s err=%v", crawlID, err)
			return
		}
		s.manifestTotal = len(paths)
		logInfof("manifest fetched crawl=%s total=%d", crawlID, len(paths))
	}()

	logInfof("dashboard init crawl=%s engine=%s base_dir=%s meta_driver=%s ttl=%s prewarm=%t",
		crawlID, engineName, baseDir, opts.MetaDriver, opts.MetaRefreshTTL, opts.MetaPrewarm)

	return s
}

// Handler returns an http.Handler with all routes registered via mizu Router.
func (s *Server) Handler() http.Handler {
	router := mizu.NewRouter()
	router.ClearMiddleware() // use our own logging middleware

	deps := &api.Deps{
		// Pipeline
		Jobs:   s.Jobs,
		Scrape: s.Scrape,
		// Core identity
		CrawlID:    s.CrawlID,
		CrawlDir:   s.CrawlDir,
		EngineName: s.EngineName,
		WARCBase:   s.WARCBase,
		WARCMdBase: s.WARCMdBase,
		Addr:       s.Addr,
		// Store interfaces are set below to avoid the Go nil-interface trap.
		// (A nil concrete pointer assigned directly to an interface field produces
		//  a non-nil interface value, causing d.Meta != nil to be true erroneously.)

		// Static FS
		StaticFS: staticFS,
		// Function fields
		ResolveFTSBase:  s.resolveFTSBase,
		ResolveCrawlDir: s.resolveCrawlDir,
		ManifestTotal:   func() int { return s.manifestTotal },
		CrawlSizeBytes: func() int64 {
			if s.crawlSize != nil {
				return s.crawlSize.get()
			}
			return 0
		},
		RefreshCrawlSize: func() {
			if s.crawlSize != nil {
				s.crawlSize.refresh()
			}
		},
		ScanDataDir:       ScanDataDir,
		CCConfig:          nil,
		ReadDocByOffset:   ReadDocByOffset,
		ReadDocFromWARCMd: readDocFromWARCMd,
		ExtractDocTitle:   extractDocTitle,
		RenderMarkdown: func(src []byte) (string, error) {
			var buf bytes.Buffer
			if err := s.md.Convert(src, &buf); err != nil {
				return "", err
			}
			return buf.String(), nil
		},
		SumInt64Map:          sumInt64Map,
		IsNumericName:        isNumericName,
		WARCIndexFromPath:    warcIndexFromPathStrict,
		FormatBytes:          FormatBytes,
		NormalizeDomainInput: normalizeDomainInput,
		// WARC helpers
		ListWARCsFallback: func(ctx context.Context, crawlID, crawlDir string) ([]webstore.WARCRecord, api.DataSummaryWithMeta) {
			recs := buildWARCRecords(crawlID, crawlDir, nil, time.Now().UTC())
			return recs, api.DataSummaryWithMeta{
				MetaBackend:     "scan-fallback",
				MetaGeneratedAt: time.Now().UTC().Format(time.RFC3339),
			}
		},
		SummarizeWARCs: summarizeWARCRecords,
		BuildWARCRow: func(ctx context.Context, rec webstore.WARCRecord, crawlDir string) api.WARCAPIRecord {
			row := toWARCAPIRecord(rec)
			enrichWARCAPIRecord(ctx, &row, crawlDir, s.Docs)
			return row
		},
		CollectSystemStats:  collectWARCSystemStats,
		DeleteWARCArtifacts: deleteWARCArtifacts,
		ExportShardMeta:     exportShardMetaParquet,
	}
	// Assign interface fields only when the concrete pointer is non-nil.
	// A nil *T assigned to an interface I creates a typed nil (non-nil interface),
	// which would cause d.X != nil checks inside handlers to be true erroneously.
	if s.Meta != nil {
		deps.Meta = s.Meta
	}
	if s.Docs != nil {
		deps.Docs = s.Docs
	}
	if s.DomainStore != nil {
		deps.Domains = s.DomainStore
	}
	if s.CCDomainStore != nil {
		deps.CCDomains = s.CCDomainStore
	}
	if s.Hub != nil {
		deps.Hub = s.Hub
	}

	// Static files served before dashboard routes.
	router.Get("/static/{path...}", func(c *mizu.Ctx) error {
		// Fingerprinted requests (have ?v=) get long-lived immutable caching.
		if c.Request().URL.Query().Get("v") != "" {
			c.Header().Set("Cache-Control", "max-age=31536000, immutable")
		}
		http.FileServer(http.FS(staticFS)).ServeHTTP(c.Writer(), c.Request())
		return nil
	})

	api.RegisterDashboardRoutes(router, deps)

	if s.Hub != nil {
		// Register /ws directly on a plain ServeMux so the raw http.ResponseWriter
		// (which implements http.Hijacker) reaches the WebSocket upgrader.
		// Mizu wraps ResponseWriter in its own type that doesn't forward Hijacker.
		mux := http.NewServeMux()
		mux.HandleFunc("/ws", s.Hub.HandleWS)
		mux.Handle("/", withRequestLogging(router))
		return mux
	}
	return router
}

// ListenAndServe starts the HTTP server, blocking until ctx is cancelled.
func (s *Server) ListenAndServe(ctx context.Context, port int) error {
	logInfof("server listen addr=:%d crawl=%s engine=%s dashboard=%t", port, s.CrawlID, s.EngineName, s.Hub != nil)
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0,           // Disabled for SSE/WebSocket long-lived connections
		IdleTimeout:  120 * time.Second,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)), // Disable HTTP/2
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			logErrorf("server exited with error: %v", err)
		}
		return err
	case <-ctx.Done():
		logInfof("server shutdown requested")
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := srv.Shutdown(shutCtx)
		if s.Hub != nil {
			s.Hub.Close()
		}
		if s.Jobs != nil {
			s.Jobs.StopPersist()
		}
		if s.Meta != nil {
			_ = s.Meta.Close()
		}
		if s.Docs != nil {
			_ = s.Docs.Close()
		}
		if s.CCDomainStore != nil {
			_ = s.CCDomainStore.Close()
		}
		if err != nil {
			logErrorf("server shutdown error: %v", err)
		}
		return err
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────

func isNumericName(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// searchEngine is kept for test compatibility where a raw *http.Request is available.
func (s *Server) searchEngine(r *http.Request) string {
	if r == nil {
		return s.EngineName
	}
	if engine := strings.TrimSpace(r.URL.Query().Get("engine")); engine != "" {
		return engine
	}
	return s.EngineName
}

func (s *Server) resolveFTSBase(engine string) string {
	engine = strings.TrimSpace(engine)
	if engine == "" {
		engine = s.EngineName
	}
	if s.CrawlDir != "" {
		return filepath.Join(s.CrawlDir, "fts", engine)
	}
	// In search-only mode, FTSBase is initialized to {base}/fts/{engine}.
	return filepath.Join(filepath.Dir(s.FTSBase), engine)
}

func (s *Server) resolveCrawlDir(crawlID string) string {
	if crawlID == s.CrawlID {
		return s.CrawlDir
	}
	return filepath.Join(filepath.Dir(s.CrawlDir), crawlID)
}

// broadcastShardScan sends a shard_scan WebSocket event so connected
// browse pages know to refresh their shard list and doc table.
func broadcastShardScan(hub *pipeline.Hub, shard string) {
	if hub == nil {
		return
	}
	hub.BroadcastAll(map[string]string{
		"type":  "shard_scan",
		"shard": shard,
	})
}

func normalizeDomainInput(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	if strings.Contains(s, "://") {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			s = strings.ToLower(strings.TrimSpace(u.Host))
		}
	}
	s = strings.TrimPrefix(s, "www.")
	if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	if i := strings.IndexByte(s, ':'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func queryIntCtx(c *mizu.Ctx, key string, def int) int {
	s := c.Query(key)
	if s == "" {
		return def
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return def
	}
	return v
}

// Ensure embed.FS satisfies fs.FS at compile time.
var _ fs.FS = staticFS
