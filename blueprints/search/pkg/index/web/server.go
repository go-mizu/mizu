// Package web provides an embedded HTTP server for browsing and searching
// Common Crawl FTS indexes through a modern web GUI.
package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	mizu "github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed static
var staticFS embed.FS

// ── Response types ──────────────────────────────────────────────────────

// errResp is the standard JSON error shape.
type errResp struct {
	Error string `json:"error"`
}

// SearchResponse is returned by GET /api/search.
type SearchResponse struct {
	Hits      []searchHit `json:"hits"`
	Total     int         `json:"total"`
	ElapsedMs int64       `json:"elapsed_ms"`
	Query     string      `json:"query"`
	Engine    string      `json:"engine"`
	Shards    int         `json:"shards"`
}

type searchHit struct {
	DocID     string     `json:"doc_id"`
	Shard     string     `json:"shard"`
	Score     float64    `json:"score,omitempty"`
	Snippet   string     `json:"snippet,omitempty"`
	URL       string     `json:"url,omitempty"`
	Title     string     `json:"title,omitempty"`
	CrawlDate *time.Time `json:"crawl_date,omitempty"`
	SizeBytes int64      `json:"size_bytes,omitempty"`
	WordCount int        `json:"word_count,omitempty"`
}

// StatsResponse is returned by GET /api/stats.
type StatsResponse struct {
	Engine    string `json:"engine"`
	Crawl     string `json:"crawl"`
	Shards    int    `json:"shards"`
	TotalDocs int64  `json:"total_docs"`
	TotalDisk string `json:"total_disk"`
}

// DocResponse is returned by GET /api/doc/{shard}/{docid}.
type DocResponse struct {
	DocID        string `json:"doc_id"`
	Shard        string `json:"shard"`
	URL          string `json:"url"`
	Title        string `json:"title"`
	CrawlDate    string `json:"crawl_date"`
	SizeBytes    int64  `json:"size_bytes"`
	WordCount    int    `json:"word_count"`
	WARCRecordID string `json:"warc_record_id"`
	RefersTo     string `json:"refers_to"`
	Markdown     string `json:"markdown"`
	HTML         string `json:"html"`
}

// BrowseShardsResponse is returned by GET /api/browse (no shard param).
type BrowseShardsResponse struct {
	Shards     []shardEntry `json:"shards"`
	HasDocMeta bool         `json:"has_doc_meta"`
}

type shardEntry struct {
	Name          string `json:"name"`
	HasPack       bool   `json:"has_pack"`
	HasScan       bool   `json:"has_scan"`
	Scanning      bool   `json:"scanning"`
	FileCount     int    `json:"file_count"`
	TotalSize     int64  `json:"total_size,omitempty"`
	LastDocDate   string `json:"last_doc_date,omitempty"`
	MetaStale     bool   `json:"meta_stale"`
	LastScannedAt string `json:"last_scanned_at,omitempty"`
}

// BrowseDocsResponse is returned by GET /api/browse?shard=xxx.
type BrowseDocsResponse struct {
	Shard         string    `json:"shard"`
	Docs          []docJSON `json:"docs"`
	Total         int       `json:"total"`
	Page          int       `json:"page"`
	PageSize      int       `json:"page_size,omitempty"`
	MetaStale     bool      `json:"meta_stale,omitempty"`
	Scanning      bool      `json:"scanning"`
	NotScanned    bool      `json:"not_scanned,omitempty"`
	LastScannedAt string    `json:"last_scanned_at,omitempty"`
}

type docJSON struct {
	DocID     string `json:"doc_id"`
	Shard     string `json:"shard"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	CrawlDate string `json:"crawl_date,omitempty"`
	SizeBytes int64  `json:"size_bytes"`
	WordCount int    `json:"word_count"`
}

// MetaScanDocsResponse is returned by POST /api/meta/scan-docs.
type MetaScanDocsResponse struct {
	Accepted bool   `json:"accepted"`
	CrawlID  string `json:"crawl_id,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// MetaRefreshResponse is returned by POST /api/meta/refresh.
type MetaRefreshResponse struct {
	Accepted bool       `json:"accepted"`
	Status   MetaStatus `json:"status"`
}

// EnginesResponse is returned by GET /api/engines.
type EnginesResponse struct {
	Engines []string `json:"engines"`
}

// JobsListResponse is returned by GET /api/jobs.
type JobsListResponse struct {
	Jobs []*Job `json:"jobs"`
}

// CancelJobResponse is returned by DELETE /api/jobs/{id}.
type CancelJobResponse struct {
	Status string `json:"status"`
}

// ── Server ──────────────────────────────────────────────────────────────

// Server serves the FTS web GUI and JSON API.
type Server struct {
	EngineName string
	CrawlID    string
	Addr       string // external engine address
	FTSBase    string // ~/data/common-crawl/{crawlID}/fts/{engine}
	WARCBase   string // ~/data/common-crawl/{crawlID}/warc
	WARCMdBase string // ~/data/common-crawl/{crawlID}/warc_md
	CrawlDir   string // ~/data/common-crawl/{crawlID} — set by NewDashboard
	Hub        *WSHub
	Jobs       *JobManager
	Meta       *MetaManager
	Docs       *DocStore // per-document browse metadata (dashboard only)

	manifestTotal int // cached count of WARCs in CC manifest

	md goldmark.Markdown
}

// DashboardOptions configures dashboard-only behavior.
type DashboardOptions struct {
	MetaDriver      string
	MetaDSN         string
	MetaRefreshTTL  time.Duration
	MetaPrewarm     bool
	MetaBusyTimeout time.Duration
	MetaJournalMode string
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
		MetaJournalMode: "WAL",
	})
}

// NewDashboardWithOptions creates a dashboard server with metadata cache config.
func NewDashboardWithOptions(engineName, crawlID, addr, baseDir string, opts DashboardOptions) *Server {
	s := New(engineName, crawlID, addr, baseDir)
	s.CrawlDir = baseDir
	s.Hub = NewWSHub()
	s.Jobs = NewJobManager(s.Hub, baseDir, crawlID)

	metaCfg := MetaConfig{
		Driver:      opts.MetaDriver,
		DSN:         opts.MetaDSN,
		RefreshTTL:  opts.MetaRefreshTTL,
		Prewarm:     opts.MetaPrewarm,
		BusyTimeout: opts.MetaBusyTimeout,
		JournalMode: opts.MetaJournalMode,
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

	s.Jobs.SetCompleteHook(func(_ *Job, crawlID, crawlDir string) {
		if s.Meta != nil {
			s.Meta.TriggerRefresh(crawlID, crawlDir, true)
		}
		// Trigger doc scan after pack/index jobs complete.
		if s.Docs != nil {
			go func() {
				warcMdBase := filepath.Join(crawlDir, "warc_md")
				if _, err := s.Docs.ScanAll(context.Background(), crawlID, warcMdBase); err != nil {
					logErrorf("doc_store: post-job scan failed crawl=%s err=%v", crawlID, err)
				}
			}()
		}
	})

	// Initialize per-document browse metadata store (per-shard DuckDB + in-memory cache).
	if docs, err := NewDocStore(s.WARCMdBase); err != nil {
		logErrorf("doc_store: init failed dir=%s err=%v (browse metadata disabled)", s.WARCMdBase, err)
	} else {
		s.Docs = docs
		logInfof("doc_store: opened dir=%s (per-shard duckdb)", s.WARCMdBase)
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

	router.Get("/api/search", s.handleSearch)
	router.Get("/api/stats", s.handleStats)
	router.Get("/api/doc/{shard}/{docid...}", s.handleDoc)
	router.Get("/api/browse", s.handleBrowse)
	router.Get("/static/{path...}", func(c *mizu.Ctx) error {
		http.FileServer(http.FS(staticFS)).ServeHTTP(c.Writer(), c.Request())
		return nil
	})
	router.Get("/", s.handleIndex)

	// Dashboard routes — only registered when Hub is non-nil (NewDashboard mode).
	if s.Hub != nil {
		router.Get("/api/overview", s.handleOverview)
		router.Get("/api/meta/status", s.handleMetaStatus)
		router.Post("/api/meta/refresh", s.handleMetaRefresh)
		router.Get("/api/crawl/{id}/data", s.handleCrawlData)
		router.Get("/api/warc", s.handleWARCList)
		router.Get("/api/warc/{index}", s.handleWARCDetail)
		router.Post("/api/warc/{index}/action", s.handleWARCAction)
		router.Get("/api/engines", s.handleEngines)
		router.Get("/api/jobs", s.handleListJobs)
		router.Get("/api/jobs/{id}", s.handleGetJob)
		router.Post("/api/jobs", s.handleCreateJob)
		router.Delete("/api/jobs/{id}", s.handleCancelJob)
		router.Post("/api/meta/scan-docs", s.handleMetaScanDocs)
		router.Get("/api/browse/stats", s.handleBrowseStats)
		// WebSocket endpoint — delegate to raw http handler.
		router.Get("/ws", func(c *mizu.Ctx) error {
			s.Hub.HandleWS(c.Writer(), c.Request())
			return nil
		})
	}

	if s.Hub != nil {
		return withRequestLogging(router)
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
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
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
		if s.Meta != nil {
			_ = s.Meta.Close()
		}
		if s.Docs != nil {
			_ = s.Docs.Close()
		}
		if err != nil {
			logErrorf("server shutdown error: %v", err)
		}
		return err
	}
}

// ── API Handlers ────────────────────────────────────────────────────────

func (s *Server) handleIndex(c *mizu.Ctx) error {
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		return c.Text(500, "internal error")
	}
	// Inject server mode so the SPA can adapt its UI.
	mode := "search"
	if s.Hub != nil {
		mode = "dashboard"
	}
	data = bytes.Replace(data, []byte(`"__SERVER_MODE__"`), []byte(`"`+mode+`"`), 1)
	data = bytes.Replace(data, []byte(`"__DEFAULT_ENGINE__"`), []byte(`"`+s.EngineName+`"`), 1)
	c.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Header().Set("Cache-Control", "no-cache")
	_, err = c.Writer().Write(data)
	return err
}

func (s *Server) handleSearch(c *mizu.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return c.JSON(400, errResp{"missing q parameter"})
	}
	engineName := s.searchEngineFromCtx(c)
	limit := queryIntCtx(c, "limit", 20)
	offset := queryIntCtx(c, "offset", 0)
	if limit > 100 {
		limit = 100
	}

	// Discover per-WARC shard directories.
	shardDirs, err := discoverShards(s.resolveFTSBase(engineName))
	if err != nil {
		return c.JSON(500, errResp{"no FTS index: " + err.Error()})
	}

	type shardResult struct {
		shard string
		hits  []index.Hit
		total int
		err   error
	}

	t0 := time.Now()
	results := make([]shardResult, len(shardDirs))
	var wg sync.WaitGroup
	for i, sd := range shardDirs {
		wg.Add(1)
		go func(idx int, dir, shardName string) {
			defer wg.Done()
			eng, err := index.NewEngine(engineName)
			if err != nil {
				results[idx].err = err
				return
			}
			if s.Addr != "" {
				if setter, ok := eng.(index.AddrSetter); ok {
					setter.SetAddr(s.Addr)
				}
			}
			if err := eng.Open(c.Context(), dir); err != nil {
				results[idx].err = err
				return
			}
			defer eng.Close()

			res, err := eng.Search(c.Context(), index.Query{Text: q, Limit: limit + offset})
			if err != nil {
				results[idx].err = err
				return
			}
			results[idx] = shardResult{shard: shardName, hits: res.Hits, total: res.Total}
		}(i, sd.path, sd.name)
	}
	wg.Wait()

	// Merge results.
	var allHits []searchHit
	totalCount := 0
	for _, sr := range results {
		if sr.err != nil {
			continue
		}
		totalCount += sr.total
		for _, h := range sr.hits {
			allHits = append(allHits, searchHit{
				DocID:   h.DocID,
				Shard:   sr.shard,
				Score:   h.Score,
				Snippet: h.Snippet,
			})
		}
	}
	sort.Slice(allHits, func(i, j int) bool { return allHits[i].Score > allHits[j].Score })
	if offset < len(allHits) {
		allHits = allHits[offset:]
	} else {
		allHits = nil
	}
	if len(allHits) > limit {
		allHits = allHits[:limit]
	}

	// Enrich hits with DocStore metadata (title, URL, date, size).
	if s.Docs != nil && len(allHits) > 0 {
		var mu sync.Mutex
		var wg2 sync.WaitGroup
		for i := range allHits {
			wg2.Add(1)
			go func(idx int) {
				defer wg2.Done()
				rec, ok, _ := s.Docs.GetDoc(c.Context(), s.CrawlID, allHits[idx].Shard, allHits[idx].DocID)
				if !ok {
					return
				}
				mu.Lock()
				allHits[idx].URL = rec.URL
				allHits[idx].Title = rec.Title
				if !rec.CrawlDate.IsZero() {
					t := rec.CrawlDate
					allHits[idx].CrawlDate = &t
				}
				allHits[idx].SizeBytes = rec.SizeBytes
				allHits[idx].WordCount = rec.WordCount
				mu.Unlock()
			}(i)
		}
		wg2.Wait()
	}

	elapsed := time.Since(t0).Milliseconds()
	return c.JSON(200, SearchResponse{
		Hits:      allHits,
		Total:     totalCount,
		ElapsedMs: elapsed,
		Query:     q,
		Engine:    engineName,
		Shards:    len(shardDirs),
	})
}

func (s *Server) handleStats(c *mizu.Ctx) error {
	engineName := s.searchEngineFromCtx(c)
	shardDirs, err := discoverShards(s.resolveFTSBase(engineName))
	if err != nil {
		return c.JSON(500, errResp{"no FTS index: " + err.Error()})
	}

	var totalDocs int64
	var totalDisk int64
	for _, sd := range shardDirs {
		eng, err := index.NewEngine(engineName)
		if err != nil {
			continue
		}
		if s.Addr != "" {
			if setter, ok := eng.(index.AddrSetter); ok {
				setter.SetAddr(s.Addr)
			}
		}
		if err := eng.Open(c.Context(), sd.path); err != nil {
			continue
		}
		stats, err := eng.Stats(c.Context())
		eng.Close()
		if err != nil {
			continue
		}
		totalDocs += stats.DocCount
		totalDisk += stats.DiskBytes
	}

	return c.JSON(200, StatsResponse{
		Engine:    engineName,
		Crawl:     s.CrawlID,
		Shards:    len(shardDirs),
		TotalDocs: totalDocs,
		TotalDisk: FormatBytes(totalDisk),
	})
}

func (s *Server) handleDoc(c *mizu.Ctx) error {
	shard := c.Param("shard")
	docid := c.Param("docid")
	if shard == "" || docid == "" {
		return c.JSON(400, errResp{"missing shard or docid"})
	}

	var raw []byte
	var meta DocRecord

	// Read from .md.warc.gz (canonical format preserving URL/date metadata).
	warcMdPath := filepath.Join(s.WARCMdBase, shard+".md.warc.gz")
	body, found, err := readDocFromWARCMd(warcMdPath, docid)
	if err != nil {
		return c.JSON(500, errResp{"read error: " + err.Error()})
	}
	if !found {
		return c.JSON(404, errResp{"document not found"})
	}
	raw = body

	// Enrich with stored metadata if DocStore available.
	if s.Docs != nil {
		if rec, ok, _ := s.Docs.GetDoc(c.Context(), s.CrawlID, shard, docid); ok {
			meta = rec
		}
	}
	// If metadata not in store yet, extract title from body.
	if meta.Title == "" && len(raw) > 0 {
		head := raw
		if len(head) > 256 {
			head = head[:256]
		}
		meta.Title = extractDocTitle(head, meta.URL)
	}

	// Render markdown to HTML.
	var buf bytes.Buffer
	if err := s.md.Convert(raw, &buf); err != nil {
		return c.JSON(500, errResp{"markdown render failed"})
	}

	wordCount := len(strings.Fields(string(raw)))

	crawlDateStr := ""
	if !meta.CrawlDate.IsZero() {
		crawlDateStr = meta.CrawlDate.UTC().Format(time.RFC3339)
	}

	return c.JSON(200, DocResponse{
		DocID:        docid,
		Shard:        shard,
		URL:          meta.URL,
		Title:        meta.Title,
		CrawlDate:    crawlDateStr,
		SizeBytes:    int64(len(raw)),
		WordCount:    wordCount,
		WARCRecordID: meta.WARCRecordID,
		RefersTo:     meta.RefersTo,
		Markdown:     string(raw),
		HTML:         buf.String(),
	})
}

func (s *Server) handleBrowse(c *mizu.Ctx) error {
	shard := c.Query("shard")
	if shard == "" {
		return s.handleBrowseShards(c)
	}
	return s.handleBrowseDocs(c, shard)
}

func (s *Server) handleBrowseShards(c *mizu.Ctx) error {
	// Collect all downloaded WARC shard indexes from warc/ directory.
	allShards := listWARCShards(s.WARCBase)

	// Collect packed shard names from warc_md/.
	packedSet := make(map[string]bool)
	for _, s := range listWARCMdShards(s.WARCMdBase) {
		packedSet[s] = true
	}

	// Collect DocStore scan metadata.
	var metas []DocShardMeta
	if s.Docs != nil {
		metas, _ = s.Docs.ListShardMetas(c.Context(), s.CrawlID)
	}
	metaByName := make(map[string]DocShardMeta, len(metas))
	for _, m := range metas {
		metaByName[m.Shard] = m
	}

	var entries []shardEntry
	for _, name := range allShards {
		e := shardEntry{Name: name, HasPack: packedSet[name]}
		if m, ok := metaByName[name]; ok {
			e.HasScan = true
			e.FileCount = int(m.TotalDocs)
			e.TotalSize = m.TotalSizeBytes
			if !m.LastDocDate.IsZero() {
				e.LastDocDate = m.LastDocDate.UTC().Format(time.RFC3339)
			}
			e.MetaStale = time.Since(m.LastScannedAt) > time.Hour
			if !m.LastScannedAt.IsZero() {
				e.LastScannedAt = m.LastScannedAt.UTC().Format(time.RFC3339)
			}
			if e.HasPack && e.MetaStale && s.Docs != nil {
				e.Scanning = s.Docs.IsScanning(s.CrawlID, name)
			}
		} else if e.HasPack && s.Docs != nil {
			e.Scanning = s.Docs.IsScanning(s.CrawlID, name)
		}
		entries = append(entries, e)
	}

	return c.JSON(200, BrowseShardsResponse{
		Shards:     entries,
		HasDocMeta: s.Docs != nil,
	})
}

func (s *Server) handleBrowseDocs(c *mizu.Ctx, shard string) error {
	page := queryIntCtx(c, "page", 1)
	pageSize := queryIntCtx(c, "page_size", 100)
	if pageSize > 500 {
		pageSize = 500
	}
	q := c.Query("q")
	sortBy := c.Query("sort")

	if s.Docs == nil {
		return c.JSON(503, errResp{"doc store not available"})
	}

	meta, hasMeta, _ := s.Docs.GetShardMeta(c.Context(), s.CrawlID, shard)
	scanning := s.Docs.IsScanning(s.CrawlID, shard)

	if !hasMeta {
		// Not scanned yet — check if .md.warc.gz exists.
		warcMdPath := filepath.Join(s.WARCMdBase, shard+".md.warc.gz")
		if _, err := os.Stat(warcMdPath); err != nil {
			return c.JSON(404, errResp{"shard not packed yet"})
		}
		return c.JSON(200, BrowseDocsResponse{
			Shard:      shard,
			Docs:       []docJSON{},
			Total:      0,
			Page:       1,
			Scanning:   scanning,
			NotScanned: true,
		})
	}

	docs, total, err := s.Docs.ListDocs(c.Context(), s.CrawlID, shard, page, pageSize, q, sortBy)
	if err != nil {
		return c.JSON(500, errResp{err.Error()})
	}

	metaStale := time.Since(meta.LastScannedAt) > time.Hour
	if metaStale && !scanning {
		go func() {
			path := filepath.Join(s.WARCMdBase, shard+".md.warc.gz")
			if _, err := s.Docs.ScanShard(context.Background(), s.CrawlID, shard, path); err != nil {
				logErrorf("doc_store: bg rescan shard=%s err=%v", shard, err)
			}
		}()
	}

	docsOut := make([]docJSON, len(docs))
	for i, d := range docs {
		docsOut[i] = docJSON{
			DocID:     d.DocID,
			Shard:     d.Shard,
			URL:       d.URL,
			Title:     d.Title,
			SizeBytes: d.SizeBytes,
			WordCount: d.WordCount,
		}
		if !d.CrawlDate.IsZero() {
			docsOut[i].CrawlDate = d.CrawlDate.UTC().Format(time.RFC3339)
		}
	}

	lastScannedAt := ""
	if !meta.LastScannedAt.IsZero() {
		lastScannedAt = meta.LastScannedAt.UTC().Format(time.RFC3339)
	}
	return c.JSON(200, BrowseDocsResponse{
		Shard:         shard,
		Docs:          docsOut,
		Total:         int(total),
		Page:          page,
		PageSize:      pageSize,
		MetaStale:     metaStale,
		Scanning:      scanning,
		LastScannedAt: lastScannedAt,
	})
}

// ── Dashboard Handlers ──────────────────────────────────────────────────

func (s *Server) handleOverview(c *mizu.Ctx) error {
	resp := buildOverviewResponse(s.CrawlID, s.CrawlDir, s.manifestTotal, s.Docs)
	if s.Meta != nil {
		summary := s.Meta.GetSummary(c.Context(), s.CrawlID, s.CrawlDir)
		resp.Meta.Backend = summary.MetaBackend
		resp.Meta.Stale = summary.MetaStale
		resp.Meta.Refreshing = summary.MetaRefreshing
	}
	return c.JSON(200, resp)
}

func (s *Server) handleMetaStatus(c *mizu.Ctx) error {
	crawlID := c.Query("crawl")
	if crawlID == "" {
		crawlID = s.CrawlID
	}
	if s.Meta == nil {
		return c.JSON(200, MetaStatus{
			CrawlID:      crawlID,
			Backend:      "scan-fallback",
			Enabled:      false,
			Status:       "idle",
			Refreshing:   false,
			RefreshTTLMS: 0,
		})
	}
	return c.JSON(200, s.Meta.Status(c.Context(), crawlID))
}

func (s *Server) handleMetaScanDocs(c *mizu.Ctx) error {
	if s.Docs == nil {
		return c.JSON(200, MetaScanDocsResponse{Accepted: false, Reason: "doc store not available"})
	}
	crawlID := s.CrawlID
	warcMdBase := s.WARCMdBase
	go func() {
		total, err := s.Docs.ScanAll(context.Background(), crawlID, warcMdBase)
		if err != nil {
			logErrorf("doc_store: manual scan-all crawl=%s err=%v", crawlID, err)
			return
		}
		logInfof("doc_store: manual scan-all crawl=%s total=%d", crawlID, total)
	}()
	return c.JSON(http.StatusAccepted, MetaScanDocsResponse{Accepted: true, CrawlID: crawlID})
}

func (s *Server) handleMetaRefresh(c *mizu.Ctx) error {
	if s.Meta == nil {
		return c.JSON(200, MetaRefreshResponse{
			Accepted: false,
			Status: MetaStatus{
				CrawlID:      s.CrawlID,
				Backend:      "scan-fallback",
				Enabled:      false,
				Status:       "idle",
				Refreshing:   false,
				RefreshTTLMS: 0,
			},
		})
	}

	type reqBody struct {
		Crawl string `json:"crawl"`
		Force bool   `json:"force"`
	}
	var body reqBody
	if c.Request().Body != nil {
		_ = json.NewDecoder(c.Request().Body).Decode(&body)
	}
	crawlID := body.Crawl
	if crawlID == "" {
		crawlID = s.CrawlID
	}
	crawlDir := s.resolveCrawlDir(crawlID)
	accepted := s.Meta.TriggerRefresh(crawlID, crawlDir, body.Force)
	status := s.Meta.Status(c.Context(), crawlID)
	code := 200
	if accepted {
		code = http.StatusAccepted
	}
	return c.JSON(code, MetaRefreshResponse{Accepted: accepted, Status: status})
}

func (s *Server) handleCrawls(c *mizu.Ctx) error {
	client := cc.NewClient("", 4)
	crawls, err := client.ListCrawls(c.Context())
	if err != nil {
		return c.JSON(500, errResp{err.Error()})
	}
	return c.JSON(200, struct {
		Crawls any `json:"crawls"`
	}{Crawls: crawls})
}

func (s *Server) handleCrawlWarcs(c *mizu.Ctx) error {
	crawlID := c.Param("id")
	if crawlID == "" {
		return c.JSON(400, errResp{"missing crawl id"})
	}

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(c.Context(), crawlID, "warc.paths.gz")
	if err != nil {
		return c.JSON(500, errResp{err.Error()})
	}

	// Check which WARCs are downloaded locally.
	warcDir := filepath.Join(s.CrawlDir, "warc")

	type warcInfo struct {
		Index      int    `json:"index"`
		RemotePath string `json:"remote_path"`
		Filename   string `json:"filename"`
		Downloaded bool   `json:"downloaded"`
		LocalSize  int64  `json:"local_size,omitempty"`
	}

	warcs := make([]warcInfo, 0, len(paths))
	for i, p := range paths {
		fname := filepath.Base(p)
		localPath := filepath.Join(warcDir, fname)
		info := warcInfo{
			Index:      i,
			RemotePath: p,
			Filename:   fname,
		}
		if fi, err := os.Stat(localPath); err == nil {
			info.Downloaded = true
			info.LocalSize = fi.Size()
		}
		warcs = append(warcs, info)
	}

	return c.JSON(200, struct {
		CrawlID string    `json:"crawl_id"`
		Total   int       `json:"total"`
		WARCs   []warcInfo `json:"warcs"`
	}{CrawlID: crawlID, Total: len(paths), WARCs: warcs})
}

func (s *Server) handleCrawlData(c *mizu.Ctx) error {
	crawlID := c.Param("id")
	if crawlID == "" {
		return c.JSON(400, errResp{"missing crawl id"})
	}

	crawlDir := s.resolveCrawlDir(crawlID)
	if s.Meta != nil {
		return c.JSON(200, s.Meta.GetSummary(c.Context(), crawlID, crawlDir))
	}

	ds := ScanDataDir(crawlDir)
	ds.CrawlID = crawlID
	return c.JSON(200, ds)
}

func (s *Server) handleEngines(c *mizu.Ctx) error {
	return c.JSON(200, EnginesResponse{Engines: index.List()})
}

func (s *Server) handleListJobs(c *mizu.Ctx) error {
	return c.JSON(200, JobsListResponse{Jobs: s.Jobs.List()})
}

func (s *Server) handleGetJob(c *mizu.Ctx) error {
	id := c.Param("id")
	job := s.Jobs.Get(id)
	if job == nil {
		return c.JSON(404, errResp{"job not found"})
	}
	return c.JSON(200, job)
}

func (s *Server) handleCreateJob(c *mizu.Ctx) error {
	var cfg JobConfig
	if err := json.NewDecoder(c.Request().Body).Decode(&cfg); err != nil {
		return c.JSON(400, errResp{"invalid JSON: " + err.Error()})
	}
	if cfg.Type == "" {
		return c.JSON(400, errResp{"missing type field"})
	}

	job := s.Jobs.Create(cfg)
	logInfof("job create id=%s type=%s crawl=%s files=%s engine=%s source=%s format=%s fast=%t",
		job.ID, cfg.Type, cfg.CrawlID, cfg.Files, cfg.Engine, cfg.Source, cfg.Format, cfg.Fast)
	s.Jobs.RunJob(job)
	return c.JSON(201, job)
}

func (s *Server) handleCancelJob(c *mizu.Ctx) error {
	id := c.Param("id")
	if ok := s.Jobs.Cancel(id); !ok {
		return c.JSON(404, errResp{"job not found"})
	}
	logInfof("job cancel id=%s", id)
	return c.JSON(200, CancelJobResponse{Status: "cancelled"})
}

// ── Helpers ─────────────────────────────────────────────────────────────

type shardDir struct {
	name string
	path string
}

func discoverShards(ftsBase string) ([]shardDir, error) {
	entries, err := os.ReadDir(ftsBase)
	if err != nil {
		return nil, err
	}
	var dirs []shardDir
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Only accept numeric shard names (WARC indices like "00000").
		// Skip engine-internal dirs (e.g. tantivy-index/, .wal).
		if !isNumericName(e.Name()) {
			continue
		}
		dirs = append(dirs, shardDir{name: e.Name(), path: filepath.Join(ftsBase, e.Name())})
	}
	if len(dirs) == 0 {
		return nil, fmt.Errorf("no shard directories in %s", ftsBase)
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	return dirs, nil
}

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

func (s *Server) searchEngineFromCtx(c *mizu.Ctx) string {
	if engine := strings.TrimSpace(c.Query("engine")); engine != "" {
		return engine
	}
	return s.EngineName
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

// handleBrowseStats returns shard-level statistics for the browse summary page.
func (s *Server) handleBrowseStats(c *mizu.Ctx) error {
	shard := c.Query("shard")
	if shard == "" {
		return c.JSON(400, errResp{"shard required"})
	}
	if s.Docs == nil {
		return c.JSON(503, errResp{"doc store not available"})
	}
	stats, err := s.Docs.ShardStats(c.Context(), s.CrawlID, shard)
	if err != nil {
		return c.JSON(500, errResp{err.Error()})
	}
	return c.JSON(200, stats)
}

// listWARCShards returns sorted 5-digit shard names from downloaded warc/ files.
func listWARCShards(warcBase string) []string {
	entries, err := os.ReadDir(warcBase)
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".warc.gz") {
			continue
		}
		// Extract 5-digit index from filename like CC-MAIN-*-00000.warc.gz
		name := e.Name()
		// Find the last occurrence of "-" before ".warc.gz" to extract index.
		base := strings.TrimSuffix(name, ".warc.gz")
		if idx := strings.LastIndex(base, "-"); idx >= 0 {
			shard := base[idx+1:]
			if len(shard) == 5 {
				seen[shard] = true
			}
		}
	}
	shards := make([]string, 0, len(seen))
	for s := range seen {
		shards = append(shards, s)
	}
	sort.Strings(shards)
	return shards
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func queryInt(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return def
	}
	return v
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

func (s *Server) resolveCrawlDir(crawlID string) string {
	if crawlID == s.CrawlID {
		return s.CrawlDir
	}
	return filepath.Join(filepath.Dir(s.CrawlDir), crawlID)
}

// Ensure embed.FS satisfies fs.FS at compile time.
var _ fs.FS = staticFS
