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

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed static/index.html
var staticFS embed.FS

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

// Handler returns an http.Handler with all routes registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/search", s.handleSearch)
	mux.HandleFunc("GET /api/stats", s.handleStats)
	mux.HandleFunc("GET /api/doc/{shard}/{docid...}", s.handleDoc)
	mux.HandleFunc("GET /api/browse", s.handleBrowse)
	mux.HandleFunc("GET /", s.handleIndex)

	// Dashboard routes — only registered when Hub is non-nil (NewDashboard mode).
	if s.Hub != nil {
		mux.HandleFunc("GET /api/overview", s.handleOverview)
		mux.HandleFunc("GET /api/meta/status", s.handleMetaStatus)
		mux.HandleFunc("POST /api/meta/refresh", s.handleMetaRefresh)
		mux.HandleFunc("GET /api/crawl/{id}/data", s.handleCrawlData)
		mux.HandleFunc("GET /api/warc", s.handleWARCList)
		mux.HandleFunc("GET /api/warc/{index}", s.handleWARCDetail)
		mux.HandleFunc("POST /api/warc/{index}/action", s.handleWARCAction)
		mux.HandleFunc("GET /api/engines", s.handleEngines)
		mux.HandleFunc("GET /api/jobs", s.handleListJobs)
		mux.HandleFunc("GET /api/jobs/{id}", s.handleGetJob)
		mux.HandleFunc("POST /api/jobs", s.handleCreateJob)
		mux.HandleFunc("DELETE /api/jobs/{id}", s.handleCancelJob)
		mux.HandleFunc("POST /api/meta/scan-docs", s.handleMetaScanDocs)
		mux.HandleFunc("GET /api/browse/stats", s.handleBrowseStats)
		mux.HandleFunc("GET /ws", s.Hub.HandleWS)
	}

	if s.Hub != nil {
		return withRequestLogging(mux)
	}
	return mux
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

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "internal error", 500)
		return
	}
	// Inject server mode so the SPA can adapt its UI.
	mode := "search"
	if s.Hub != nil {
		mode = "dashboard"
	}
	data = bytes.Replace(data, []byte(`"__SERVER_MODE__"`), []byte(`"`+mode+`"`), 1)
	data = bytes.Replace(data, []byte(`"__DEFAULT_ENGINE__"`), []byte(`"`+s.EngineName+`"`), 1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(data)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, 400, map[string]string{"error": "missing q parameter"})
		return
	}
	engineName := s.searchEngine(r)
	limit := queryInt(r, "limit", 20)
	offset := queryInt(r, "offset", 0)
	if limit > 100 {
		limit = 100
	}

	// Discover per-WARC shard directories.
	shardDirs, err := discoverShards(s.resolveFTSBase(engineName))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "no FTS index: " + err.Error()})
		return
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
			if err := eng.Open(r.Context(), dir); err != nil {
				results[idx].err = err
				return
			}
			defer eng.Close()

			res, err := eng.Search(r.Context(), index.Query{Text: q, Limit: limit + offset})
			if err != nil {
				results[idx].err = err
				return
			}
			results[idx] = shardResult{shard: shardName, hits: res.Hits, total: res.Total}
		}(i, sd.path, sd.name)
	}
	wg.Wait()

	// Merge results.
	type apiHit struct {
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
	var allHits []apiHit
	totalCount := 0
	for _, sr := range results {
		if sr.err != nil {
			continue
		}
		totalCount += sr.total
		for _, h := range sr.hits {
			allHits = append(allHits, apiHit{
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
				rec, ok, _ := s.Docs.GetDoc(r.Context(), s.CrawlID, allHits[idx].Shard, allHits[idx].DocID)
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
	writeJSON(w, 200, map[string]any{
		"hits":       allHits,
		"total":      totalCount,
		"elapsed_ms": elapsed,
		"query":      q,
		"engine":     engineName,
		"shards":     len(shardDirs),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	engineName := s.searchEngine(r)
	shardDirs, err := discoverShards(s.resolveFTSBase(engineName))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "no FTS index: " + err.Error()})
		return
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
		if err := eng.Open(r.Context(), sd.path); err != nil {
			continue
		}
		stats, err := eng.Stats(r.Context())
		eng.Close()
		if err != nil {
			continue
		}
		totalDocs += stats.DocCount
		totalDisk += stats.DiskBytes
	}

	writeJSON(w, 200, map[string]any{
		"engine":     engineName,
		"crawl":      s.CrawlID,
		"shards":     len(shardDirs),
		"total_docs": totalDocs,
		"total_disk": FormatBytes(totalDisk),
	})
}

func (s *Server) handleDoc(w http.ResponseWriter, r *http.Request) {
	shard := r.PathValue("shard")
	docid := r.PathValue("docid")
	if shard == "" || docid == "" {
		writeJSON(w, 400, map[string]string{"error": "missing shard or docid"})
		return
	}

	var raw []byte
	var meta DocRecord

	// Read from .md.warc.gz (canonical format preserving URL/date metadata).
	warcMdPath := filepath.Join(s.WARCMdBase, shard+".md.warc.gz")
	body, found, err := readDocFromWARCMd(warcMdPath, docid)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "read error: " + err.Error()})
		return
	}
	if !found {
		writeJSON(w, 404, map[string]string{"error": "document not found"})
		return
	}
	raw = body

	// Enrich with stored metadata if DocStore available.
	if s.Docs != nil {
		if rec, ok, _ := s.Docs.GetDoc(r.Context(), s.CrawlID, shard, docid); ok {
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
		writeJSON(w, 500, map[string]string{"error": "markdown render failed"})
		return
	}

	wordCount := len(strings.Fields(string(raw)))

	crawlDateStr := ""
	if !meta.CrawlDate.IsZero() {
		crawlDateStr = meta.CrawlDate.UTC().Format(time.RFC3339)
	}

	writeJSON(w, 200, map[string]any{
		"doc_id":         docid,
		"shard":          shard,
		"url":            meta.URL,
		"title":          meta.Title,
		"crawl_date":     crawlDateStr,
		"size_bytes":     int64(len(raw)),
		"word_count":     wordCount,
		"warc_record_id": meta.WARCRecordID,
		"refers_to":      meta.RefersTo,
		"markdown":       string(raw),
		"html":           buf.String(),
	})
}

func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	shard := r.URL.Query().Get("shard")

	if shard == "" {
		s.handleBrowseShards(w, r)
		return
	}
	s.handleBrowseDocs(w, r, shard)
}

func (s *Server) handleBrowseShards(w http.ResponseWriter, r *http.Request) {
	type shardEntry struct {
		Name          string `json:"name"`
		HasPack       bool   `json:"has_pack"`   // .md.warc.gz exists
		HasScan       bool   `json:"has_scan"`   // DocStore metadata populated
		Scanning      bool   `json:"scanning"`   // scan in progress
		FileCount     int    `json:"file_count"` // 0 if not scanned
		TotalSize     int64  `json:"total_size,omitempty"`
		LastDocDate   string `json:"last_doc_date,omitempty"`
		MetaStale     bool   `json:"meta_stale"` // scanned >1h ago
		LastScannedAt string `json:"last_scanned_at,omitempty"`
	}

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
		metas, _ = s.Docs.ListShardMetas(r.Context(), s.CrawlID)
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

	writeJSON(w, 200, map[string]any{
		"shards":       entries,
		"has_doc_meta": s.Docs != nil,
	})
}

func (s *Server) handleBrowseDocs(w http.ResponseWriter, r *http.Request, shard string) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 100)
	if pageSize > 500 {
		pageSize = 500
	}
	q := r.URL.Query().Get("q")
	sortBy := r.URL.Query().Get("sort")

	if s.Docs == nil {
		writeJSON(w, 503, map[string]string{"error": "doc store not available"})
		return
	}

	meta, hasMeta, _ := s.Docs.GetShardMeta(r.Context(), s.CrawlID, shard)
	scanning := s.Docs.IsScanning(s.CrawlID, shard)

	if !hasMeta {
		// Not scanned yet — check if .md.warc.gz exists.
		warcMdPath := filepath.Join(s.WARCMdBase, shard+".md.warc.gz")
		if _, err := os.Stat(warcMdPath); err != nil {
			writeJSON(w, 404, map[string]string{"error": "shard not packed yet"})
			return
		}
		writeJSON(w, 200, map[string]any{
			"shard":    shard,
			"docs":     []any{},
			"total":    0,
			"page":     1,
			"scanning": scanning,
			"not_scanned": true,
		})
		return
	}

	docs, total, err := s.Docs.ListDocs(r.Context(), s.CrawlID, shard, page, pageSize, q, sortBy)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
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

	type docJSON struct {
		DocID     string `json:"doc_id"`
		Shard     string `json:"shard"`
		URL       string `json:"url"`
		Title     string `json:"title"`
		CrawlDate string `json:"crawl_date,omitempty"`
		SizeBytes int64  `json:"size_bytes"`
		WordCount int    `json:"word_count"`
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
	writeJSON(w, 200, map[string]any{
		"shard":           shard,
		"docs":            docsOut,
		"total":           total,
		"page":            page,
		"page_size":       pageSize,
		"meta_stale":      metaStale,
		"scanning":        scanning,
		"last_scanned_at": lastScannedAt,
	})
}

// ── Dashboard Handlers ──────────────────────────────────────────────────

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	resp := buildOverviewResponse(s.CrawlID, s.CrawlDir, s.manifestTotal, s.Docs)
	if s.Meta != nil {
		summary := s.Meta.GetSummary(r.Context(), s.CrawlID, s.CrawlDir)
		resp.Meta.Backend = summary.MetaBackend
		resp.Meta.Stale = summary.MetaStale
		resp.Meta.Refreshing = summary.MetaRefreshing
	}
	writeJSON(w, 200, resp)
}

func (s *Server) handleMetaStatus(w http.ResponseWriter, r *http.Request) {
	crawlID := r.URL.Query().Get("crawl")
	if crawlID == "" {
		crawlID = s.CrawlID
	}
	if s.Meta == nil {
		writeJSON(w, 200, MetaStatus{
			CrawlID:      crawlID,
			Backend:      "scan-fallback",
			Enabled:      false,
			Status:       "idle",
			Refreshing:   false,
			RefreshTTLMS: 0,
		})
		return
	}
	writeJSON(w, 200, s.Meta.Status(r.Context(), crawlID))
}

func (s *Server) handleMetaScanDocs(w http.ResponseWriter, r *http.Request) {
	if s.Docs == nil {
		writeJSON(w, 200, map[string]any{"accepted": false, "reason": "doc store not available"})
		return
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
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true, "crawl_id": crawlID})
}

func (s *Server) handleMetaRefresh(w http.ResponseWriter, r *http.Request) {
	if s.Meta == nil {
		writeJSON(w, 200, map[string]any{
			"accepted": false,
			"status": MetaStatus{
				CrawlID:      s.CrawlID,
				Backend:      "scan-fallback",
				Enabled:      false,
				Status:       "idle",
				Refreshing:   false,
				RefreshTTLMS: 0,
			},
		})
		return
	}

	type reqBody struct {
		Crawl string `json:"crawl"`
		Force bool   `json:"force"`
	}
	var body reqBody
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	crawlID := body.Crawl
	if crawlID == "" {
		crawlID = s.CrawlID
	}
	crawlDir := s.resolveCrawlDir(crawlID)
	accepted := s.Meta.TriggerRefresh(crawlID, crawlDir, body.Force)
	status := s.Meta.Status(r.Context(), crawlID)
	code := 200
	if accepted {
		code = http.StatusAccepted
	}
	writeJSON(w, code, map[string]any{
		"accepted": accepted,
		"status":   status,
	})
}

func (s *Server) handleCrawls(w http.ResponseWriter, r *http.Request) {
	client := cc.NewClient("", 4)
	crawls, err := client.ListCrawls(r.Context())
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"crawls": crawls})
}

func (s *Server) handleCrawlWarcs(w http.ResponseWriter, r *http.Request) {
	crawlID := r.PathValue("id")
	if crawlID == "" {
		writeJSON(w, 400, map[string]string{"error": "missing crawl id"})
		return
	}

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(r.Context(), crawlID, "warc.paths.gz")
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
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

	writeJSON(w, 200, map[string]any{
		"crawl_id": crawlID,
		"total":    len(paths),
		"warcs":    warcs,
	})
}

func (s *Server) handleCrawlData(w http.ResponseWriter, r *http.Request) {
	crawlID := r.PathValue("id")
	if crawlID == "" {
		writeJSON(w, 400, map[string]string{"error": "missing crawl id"})
		return
	}

	crawlDir := s.resolveCrawlDir(crawlID)
	if s.Meta != nil {
		writeJSON(w, 200, s.Meta.GetSummary(r.Context(), crawlID, crawlDir))
		return
	}

	ds := ScanDataDir(crawlDir)
	ds.CrawlID = crawlID
	writeJSON(w, 200, ds)
}

func (s *Server) handleEngines(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"engines": index.List()})
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"jobs": s.Jobs.List()})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job := s.Jobs.Get(id)
	if job == nil {
		writeJSON(w, 404, map[string]string{"error": "job not found"})
		return
	}
	writeJSON(w, 200, job)
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var cfg JobConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}
	if cfg.Type == "" {
		writeJSON(w, 400, map[string]string{"error": "missing type field"})
		return
	}

	job := s.Jobs.Create(cfg)
	logInfof("job create id=%s type=%s crawl=%s files=%s engine=%s source=%s format=%s fast=%t",
		job.ID, cfg.Type, cfg.CrawlID, cfg.Files, cfg.Engine, cfg.Source, cfg.Format, cfg.Fast)
	s.Jobs.RunJob(job)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(job)
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if ok := s.Jobs.Cancel(id); !ok {
		writeJSON(w, 404, map[string]string{"error": "job not found"})
		return
	}
	logInfof("job cancel id=%s", id)
	writeJSON(w, 200, map[string]string{"status": "cancelled"})
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
func (s *Server) handleBrowseStats(w http.ResponseWriter, r *http.Request) {
	shard := r.URL.Query().Get("shard")
	if shard == "" {
		writeJSON(w, 400, map[string]string{"error": "shard required"})
		return
	}
	if s.Docs == nil {
		writeJSON(w, 503, map[string]string{"error": "doc store not available"})
		return
	}
	stats, err := s.Docs.ShardStats(r.Context(), s.CrawlID, shard)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, stats)
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

func (s *Server) resolveCrawlDir(crawlID string) string {
	if crawlID == s.CrawlID {
		return s.CrawlDir
	}
	return filepath.Join(filepath.Dir(s.CrawlDir), crawlID)
}

// Ensure embed.FS satisfies fs.FS at compile time.
var _ fs.FS = staticFS
