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
	MDBase     string // ~/data/common-crawl/{crawlID}/markdown
	CrawlDir   string // ~/data/common-crawl/{crawlID} — set by NewDashboard
	Hub        *WSHub
	Jobs       *JobManager

	md goldmark.Markdown
}

// New creates a Server for the given crawl and engine.
func New(engineName, crawlID, addr, baseDir string) *Server {
	return &Server{
		EngineName: engineName,
		CrawlID:    crawlID,
		Addr:       addr,
		FTSBase:    filepath.Join(baseDir, "fts", engineName),
		MDBase:     filepath.Join(baseDir, "markdown"),
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
	s := New(engineName, crawlID, addr, baseDir)
	s.CrawlDir = baseDir
	s.Hub = NewWSHub()
	s.Jobs = NewJobManager(s.Hub, baseDir, crawlID)
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
		mux.HandleFunc("GET /api/crawls", s.handleCrawls)
		mux.HandleFunc("GET /api/crawl/{id}/warcs", s.handleCrawlWarcs)
		mux.HandleFunc("GET /api/crawl/{id}/data", s.handleCrawlData)
		mux.HandleFunc("GET /api/engines", s.handleEngines)
		mux.HandleFunc("GET /api/jobs", s.handleListJobs)
		mux.HandleFunc("GET /api/jobs/{id}", s.handleGetJob)
		mux.HandleFunc("POST /api/jobs", s.handleCreateJob)
		mux.HandleFunc("DELETE /api/jobs/{id}", s.handleCancelJob)
		mux.HandleFunc("GET /ws", s.Hub.HandleWS)
	}

	return mux
}

// ListenAndServe starts the HTTP server, blocking until ctx is cancelled.
func (s *Server) ListenAndServe(ctx context.Context, port int) error {
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
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutCtx)
	}
}

// ── API Handlers ────────────────────────────────────────────────────────

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "internal error", 500)
		return
	}
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
	limit := queryInt(r, "limit", 20)
	offset := queryInt(r, "offset", 0)
	if limit > 100 {
		limit = 100
	}

	// Discover per-WARC shard directories.
	shardDirs, err := discoverShards(s.FTSBase)
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
			eng, err := index.NewEngine(s.EngineName)
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
		DocID   string  `json:"doc_id"`
		Shard   string  `json:"shard"`
		Score   float64 `json:"score,omitempty"`
		Snippet string  `json:"snippet,omitempty"`
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

	elapsed := time.Since(t0).Milliseconds()
	writeJSON(w, 200, map[string]any{
		"hits":       allHits,
		"total":      totalCount,
		"elapsed_ms": elapsed,
		"query":      q,
		"engine":     s.EngineName,
		"shards":     len(shardDirs),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	shardDirs, err := discoverShards(s.FTSBase)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "no FTS index: " + err.Error()})
		return
	}

	var totalDocs int64
	var totalDisk int64
	for _, sd := range shardDirs {
		eng, err := index.NewEngine(s.EngineName)
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
		"engine":     s.EngineName,
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

	// Resolve the document path. Files are stored in nested UUID directories:
	// {shard}/{xx}/{yy}/{zz}/{uuid}.md where xx/yy/zz are the first 6 hex chars.
	resolved := resolveDocPath(s.MDBase, shard, docid)
	if resolved == "" {
		writeJSON(w, 404, map[string]string{"error": "document not found"})
		return
	}
	// Security: ensure resolved path stays under MDBase.
	if !strings.HasPrefix(resolved, s.MDBase) {
		writeJSON(w, 400, map[string]string{"error": "invalid path"})
		return
	}

	raw, err := os.ReadFile(resolved)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "document not found"})
		return
	}

	// Render markdown to HTML.
	var buf bytes.Buffer
	if err := s.md.Convert(raw, &buf); err != nil {
		writeJSON(w, 500, map[string]string{"error": "markdown render failed"})
		return
	}

	wordCount := len(strings.Fields(string(raw)))
	fi, _ := os.Stat(resolved)
	var size int64
	if fi != nil {
		size = fi.Size()
	}

	writeJSON(w, 200, map[string]any{
		"doc_id":     docid,
		"shard":      shard,
		"markdown":   string(raw),
		"html":       buf.String(),
		"word_count": wordCount,
		"size":       size,
	})
}

func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	shard := r.URL.Query().Get("shard")

	if shard == "" {
		// List shards: check both FTS index dirs and markdown dirs.
		shards := listShards(s.MDBase)
		writeJSON(w, 200, map[string]any{"shards": shards})
		return
	}

	// List markdown files in shard by walking the nested UUID directory tree.
	dir := filepath.Join(s.MDBase, shard)
	resolved, err := filepath.Abs(dir)
	if err != nil || !strings.HasPrefix(resolved, s.MDBase) {
		writeJSON(w, 400, map[string]string{"error": "invalid shard"})
		return
	}

	type fileInfo struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	}
	var files []fileInfo
	filepath.WalkDir(resolved, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		info, _ := d.Info()
		var sz int64
		if info != nil {
			sz = info.Size()
		}
		// Use the UUID filename (without .md extension is confusing; keep full name).
		files = append(files, fileInfo{Name: d.Name(), Size: sz})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	// Cap at 500 files for the API response.
	if len(files) > 500 {
		files = files[:500]
	}

	writeJSON(w, 200, map[string]any{"files": files, "shard": shard, "total": len(files)})
}

// ── Dashboard Handlers ──────────────────────────────────────────────────

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	ds := ScanDataDir(s.CrawlDir)
	ds.CrawlID = s.CrawlID
	writeJSON(w, 200, ds)
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

	// Resolve crawl dir: if the requested crawl matches ours, use s.CrawlDir;
	// otherwise compute it from the parent directory.
	crawlDir := s.CrawlDir
	if crawlID != s.CrawlID {
		crawlDir = filepath.Join(filepath.Dir(s.CrawlDir), crawlID)
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

type shardInfo struct {
	Name      string `json:"name"`
	FileCount int    `json:"file_count"`
}

func listShards(mdBase string) []shardInfo {
	entries, err := os.ReadDir(mdBase)
	if err != nil {
		return nil
	}
	var shards []shardInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		count := countMDFilesEstimate(filepath.Join(mdBase, e.Name()))
		if count == 0 {
			continue // skip empty shards
		}
		shards = append(shards, shardInfo{Name: e.Name(), FileCount: count})
	}
	sort.Slice(shards, func(i, j int) bool { return shards[i].Name < shards[j].Name })
	return shards
}

// countMDFilesEstimate estimates .md file count by sampling the first hex bucket.
// For UUID-nested dirs ({xx}/{yy}/{zz}/), sampling one top-level bucket and
// multiplying by 256 gives a reasonable estimate without walking all 21K files.
func countMDFilesEstimate(dir string) int {
	topEntries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	// Check if this is a flat directory (files directly here) or nested (hex subdirs).
	mdCount := 0
	dirCount := 0
	for _, e := range topEntries {
		if e.IsDir() {
			dirCount++
		} else if strings.HasSuffix(e.Name(), ".md") {
			mdCount++
		}
	}
	// Flat directory: return direct count.
	if mdCount > 0 || dirCount == 0 {
		return mdCount
	}
	// Nested UUID dirs: sample first bucket and extrapolate.
	sampleDir := filepath.Join(dir, topEntries[0].Name())
	sampleCount := 0
	filepath.WalkDir(sampleDir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			sampleCount++
		}
		return nil
	})
	return sampleCount * dirCount
}

// resolveDocPath resolves a DocID (UUID) to its nested file path under
// {mdBase}/{shard}/. The UUID files are stored in a hierarchy like:
//
//	{shard}/{xx}/{yy}/{zz}/{uuid}.md
//
// where xx, yy, zz are derived from the first characters of the UUID.
func resolveDocPath(mdBase, shard, docid string) string {
	// Strip .md if already present.
	name := strings.TrimSuffix(docid, ".md")

	// Try the nested UUID path: {xx}/{yy}/{zz}/{uuid}.md
	// UUIDs are like "9c4852b9-f2bb-46c8-92a2-ab8619823d9e"
	// Nested as: 9c/48/52/9c4852b9-f2bb-46c8-92a2-ab8619823d9e.md
	clean := strings.ReplaceAll(name, "-", "")
	if len(clean) >= 6 {
		nested := filepath.Join(mdBase, shard, clean[0:2], clean[2:4], clean[4:6], name+".md")
		if abs, err := filepath.Abs(nested); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}

	// Fallback: try flat path {shard}/{docid} and {shard}/{docid}.md
	for _, candidate := range []string{
		filepath.Join(mdBase, shard, docid),
		filepath.Join(mdBase, shard, docid+".md"),
		filepath.Join(mdBase, shard, name+".md"),
	} {
		if abs, err := filepath.Abs(candidate); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}

	return ""
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

// Ensure embed.FS satisfies fs.FS at compile time.
var _ fs.FS = staticFS
