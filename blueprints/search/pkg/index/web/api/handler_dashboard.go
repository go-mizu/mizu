package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	mizu "github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

// ── Overview types ────────────────────────────────────────────────────────────

// OverviewResponse is the structured /api/overview payload with pipeline stages.
type OverviewResponse struct {
	CrawlID   string `json:"crawl_id"`
	CrawlName string `json:"crawl_name,omitempty"`
	CrawlFrom string `json:"crawl_from,omitempty"`
	CrawlTo   string `json:"crawl_to,omitempty"`

	Manifest   ManifestStage   `json:"manifest"`
	Downloaded DownloadedStage `json:"downloaded"`
	Markdown   MarkdownStage   `json:"markdown"`
	Pack       PackStage       `json:"pack"`
	Indexed    IndexedStage    `json:"indexed"`

	Storage StorageInfo  `json:"storage"`
	System  SystemInfo   `json:"system"`
	Meta    OverviewMeta `json:"meta"`
}

// ManifestStage holds manifest-level pipeline stats.
type ManifestStage struct {
	TotalWARCs        int   `json:"total_warcs"`
	EstTotalSizeBytes int64 `json:"est_total_size_bytes"`
	EstTotalURLs      int64 `json:"est_total_urls"`
}

// DownloadedStage holds download-stage stats.
type DownloadedStage struct {
	Count        int   `json:"count"`
	TotalWARCs   int   `json:"total_warcs"`
	SizeBytes    int64 `json:"size_bytes"`
	AvgWARCBytes int64 `json:"avg_warc_bytes"`
}

// MarkdownStage holds markdown-stage stats.
type MarkdownStage struct {
	Count          int   `json:"count"`
	TotalWARCs     int   `json:"total_warcs"`
	SizeBytes      int64 `json:"size_bytes"`
	TotalDocs      int64 `json:"total_docs"`
	AvgDocsPerWARC int64 `json:"avg_docs_per_warc"`
	AvgDocBytes    int64 `json:"avg_doc_bytes"`
}

// PackStage holds pack-stage stats.
type PackStage struct {
	Count        int   `json:"count"`
	TotalWARCs   int   `json:"total_warcs"`
	ParquetBytes int64 `json:"parquet_bytes"`
	WARCMdBytes  int64 `json:"warc_md_bytes"`
}

// IndexedStage holds FTS index stage stats.
type IndexedStage struct {
	Count         int   `json:"count"`
	TotalWARCs    int   `json:"total_warcs"`
	DahliaBytes   int64 `json:"dahlia_bytes"`
	DahliaShards  int   `json:"dahlia_shards"`
	TantivyBytes  int64 `json:"tantivy_bytes"`
	TantivyShards int   `json:"tantivy_shards"`
}

// StorageInfo holds disk storage stats.
type StorageInfo struct {
	DiskTotal          int64 `json:"disk_total"`
	DiskUsed           int64 `json:"disk_used"`
	DiskFree           int64 `json:"disk_free"`
	CrawlBytes         int64 `json:"crawl_bytes"`
	ProjectedFullBytes int64 `json:"projected_full_bytes"`
}

// SystemInfo holds Go runtime and host system stats.
type SystemInfo struct {
	HeapAlloc  int64  `json:"heap_alloc"`
	HeapSys    int64  `json:"heap_sys"`
	StackInuse int64  `json:"stack_inuse"`
	NumGC      int64  `json:"num_gc"`
	Goroutines int    `json:"goroutines"`
	GoVersion  string `json:"go_version"`
	Uptime     int64  `json:"uptime_seconds"`
	PID        int    `json:"pid"`
	GOMEMLIMIT int64  `json:"gomemlimit"`

	CPUs          int    `json:"cpus"`
	GOOS          string `json:"goos"`
	GOARCH        string `json:"goarch"`
	MemTotalBytes int64  `json:"mem_total_bytes"`
	MemAvailBytes int64  `json:"mem_avail_bytes"`
	MemUsedBytes  int64  `json:"mem_used_bytes"`

	NetBytesRecv int64 `json:"net_bytes_recv"`
	NetBytesSent int64 `json:"net_bytes_sent"`

	DiskReadBytes  int64 `json:"disk_read_bytes"`
	DiskWriteBytes int64 `json:"disk_write_bytes"`
}

// OverviewMeta holds metadata cache info for the overview.
type OverviewMeta struct {
	Backend     string `json:"backend"`
	GeneratedAt string `json:"generated_at"`
	Stale       bool   `json:"stale"`
	Refreshing  bool   `json:"refreshing"`
}

var overviewStartTime = time.Now()

// ── Handlers ──────────────────────────────────────────────────────────────────

func handleOverview(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		var crawlBytes int64
		if d.CrawlSizeBytes != nil {
			crawlBytes = d.CrawlSizeBytes()
		}
		if d.RefreshCrawlSize != nil {
			d.RefreshCrawlSize()
		}

		resp := buildOverviewResponse(d, crawlBytes)
		if d.Meta != nil {
			resp.Meta.Backend = d.Meta.Backend()
			resp.Meta.Stale = d.Meta.IsStale(d.CrawlID)
			resp.Meta.Refreshing = d.Meta.IsRefreshing(d.CrawlID)
			if resp.Meta.Stale && !resp.Meta.Refreshing {
				d.Meta.TriggerRefresh(d.CrawlID, d.CrawlDir, false)
			}
		}
		return c.JSON(200, resp)
	}
}

func buildOverviewResponse(d *Deps, crawlBytes int64) OverviewResponse {
	crawlID := d.CrawlID
	crawlDir := d.CrawlDir
	manifestTotal := 0
	if d.ManifestTotal != nil {
		manifestTotal = d.ManifestTotal()
	}
	isNumeric := d.IsNumericName
	if isNumeric == nil {
		isNumeric = defaultIsNumericName
	}

	resp := OverviewResponse{CrawlID: crawlID}

	resp.Manifest = ManifestStage{
		TotalWARCs:   manifestTotal,
		EstTotalURLs: int64(manifestTotal) * 30_000,
	}

	resp.Downloaded = scanDownloadedStage(crawlDir)
	resp.Downloaded.TotalWARCs = manifestTotal

	if resp.Downloaded.Count > 0 && resp.Downloaded.AvgWARCBytes > 0 {
		resp.Manifest.EstTotalSizeBytes = int64(manifestTotal) * resp.Downloaded.AvgWARCBytes
	}

	resp.Markdown = scanMarkdownStage(crawlDir, d.Docs, isNumeric)
	resp.Markdown.TotalWARCs = resp.Downloaded.Count

	resp.Pack = scanPackStage(crawlDir, isNumeric)
	resp.Pack.TotalWARCs = resp.Downloaded.Count

	resp.Indexed = scanIndexedStage(crawlDir, isNumeric)
	resp.Indexed.TotalWARCs = resp.Downloaded.Count

	resp.Storage = collectStorageInfoAPI(crawlDir, manifestTotal, resp.Downloaded.AvgWARCBytes, crawlBytes)
	resp.System = collectSystemInfoAPI()
	resp.Meta = OverviewMeta{GeneratedAt: time.Now().UTC().Format(time.RFC3339)}

	return resp
}

func scanDownloadedStage(crawlDir string) DownloadedStage {
	var stage DownloadedStage
	warcDir := filepath.Join(crawlDir, "warc")
	entries, err := os.ReadDir(warcDir)
	if err != nil {
		return stage
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".warc.gz") {
			continue
		}
		stage.Count++
		if info, err := e.Info(); err == nil {
			stage.SizeBytes += info.Size()
		}
	}
	if stage.Count > 0 {
		stage.AvgWARCBytes = stage.SizeBytes / int64(stage.Count)
	}
	return stage
}

func scanMarkdownStage(crawlDir string, docs DocProvider, isNumeric func(string) bool) MarkdownStage {
	var stage MarkdownStage
	warcMdDir := filepath.Join(crawlDir, "warc_md")
	entries, _ := os.ReadDir(warcMdDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md.warc.gz") {
			continue
		}
		shard := strings.TrimSuffix(e.Name(), ".md.warc.gz")
		if !isNumeric(shard) {
			continue
		}
		stage.Count++
		if info, err := e.Info(); err == nil {
			stage.SizeBytes += info.Size()
		}
		if docs != nil {
			if meta, ok, _ := docs.GetShardMeta(context.Background(), "", shard); ok {
				stage.TotalDocs += meta.TotalDocs
			}
		}
	}
	if stage.Count > 0 && stage.TotalDocs > 0 {
		stage.AvgDocsPerWARC = stage.TotalDocs / int64(stage.Count)
		stage.AvgDocBytes = stage.SizeBytes / stage.TotalDocs
	}
	return stage
}

func scanPackStage(crawlDir string, isNumeric func(string) bool) PackStage {
	var stage PackStage
	seen := make(map[string]bool)

	parquetDir := filepath.Join(crawlDir, "pack", "parquet")
	if entries, err := os.ReadDir(parquetDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, ".parquet") {
				continue
			}
			idx := strings.TrimSuffix(name, ".parquet")
			if isNumeric(idx) {
				seen[idx] = true
			}
			if info, err := e.Info(); err == nil {
				stage.ParquetBytes += info.Size()
			}
		}
	}

	warcMdDir := filepath.Join(crawlDir, "warc_md")
	if entries, err := os.ReadDir(warcMdDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md.warc.gz") {
				continue
			}
			shard := strings.TrimSuffix(e.Name(), ".md.warc.gz")
			if isNumeric(shard) {
				seen[shard] = true
			}
			if info, err := e.Info(); err == nil {
				stage.WARCMdBytes += info.Size()
			}
		}
	}

	stage.Count = len(seen)
	return stage
}

func scanIndexedStage(crawlDir string, isNumeric func(string) bool) IndexedStage {
	var stage IndexedStage
	ftsDir := filepath.Join(crawlDir, "fts")
	seen := make(map[string]bool)

	dahliaDir := filepath.Join(ftsDir, "dahlia")
	if shards, err := os.ReadDir(dahliaDir); err == nil {
		for _, s := range shards {
			if !s.IsDir() || !isNumeric(s.Name()) {
				continue
			}
			stage.DahliaShards++
			seen[s.Name()] = true
			stage.DahliaBytes += dirSizeAPI(filepath.Join(dahliaDir, s.Name()))
		}
	}

	tantivyDir := filepath.Join(ftsDir, "tantivy")
	if shards, err := os.ReadDir(tantivyDir); err == nil {
		for _, s := range shards {
			if !s.IsDir() || !isNumeric(s.Name()) {
				continue
			}
			stage.TantivyShards++
			seen[s.Name()] = true
			stage.TantivyBytes += dirSizeAPI(filepath.Join(tantivyDir, s.Name()))
		}
	}

	stage.Count = len(seen)
	return stage
}

func dirSizeAPI(dir string) int64 {
	var total int64
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func collectSystemInfoAPI() SystemInfo {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	goVersion := runtime.Version()
	if bi, ok := debug.ReadBuildInfo(); ok && bi.GoVersion != "" {
		goVersion = bi.GoVersion
	}

	memlimit := debug.SetMemoryLimit(-1)

	si := SystemInfo{
		HeapAlloc:  int64(ms.HeapAlloc),
		HeapSys:    int64(ms.HeapSys),
		StackInuse: int64(ms.StackInuse),
		NumGC:      int64(ms.NumGC),
		Goroutines: runtime.NumGoroutine(),
		GoVersion:  goVersion,
		Uptime:     int64(time.Since(overviewStartTime).Seconds()),
		PID:        os.Getpid(),
		GOMEMLIMIT: memlimit,
		CPUs:       runtime.NumCPU(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	}
	return si
}

func collectStorageInfoAPI(crawlDir string, manifestTotal int, avgWARCBytes int64, crawlBytes int64) StorageInfo {
	info := StorageInfo{}
	info.CrawlBytes = crawlBytes
	// Disk usage via syscall — delegate to web package via Deps if needed.
	// For now just populate crawl bytes; web/server.go provides the hook.
	if manifestTotal > 0 && avgWARCBytes > 0 {
		info.ProjectedFullBytes = int64(manifestTotal) * avgWARCBytes
	}
	return info
}

func handleMetaStatus(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		crawlID := c.Query("crawl")
		if crawlID == "" {
			crawlID = d.CrawlID
		}
		if d.Meta == nil {
			return c.JSON(200, MetaStatus{
				CrawlID:      crawlID,
				Backend:      "scan-fallback",
				Enabled:      false,
				Status:       "idle",
				Refreshing:   false,
				RefreshTTLMS: 0,
			})
		}
		return c.JSON(200, d.Meta.Status(c.Context(), crawlID))
	}
}

func handleMetaRefresh(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		if d.Meta == nil {
			return c.JSON(200, MetaRefreshResponse{
				Accepted: false,
				Status: MetaStatus{
					CrawlID:      d.CrawlID,
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
			crawlID = d.CrawlID
		}
		crawlDir := d.CrawlDir
		if d.ResolveCrawlDir != nil {
			crawlDir = d.ResolveCrawlDir(crawlID)
		}
		accepted := d.Meta.TriggerRefresh(crawlID, crawlDir, body.Force)
		status := d.Meta.Status(c.Context(), crawlID)
		code := 200
		if accepted {
			code = http.StatusAccepted
		}
		return c.JSON(code, MetaRefreshResponse{Accepted: accepted, Status: status})
	}
}

func handleMetaScanDocs(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		if d.Docs == nil {
			return c.JSON(200, MetaScanDocsResponse{Accepted: false, Reason: "doc store not available"})
		}
		crawlID := d.CrawlID
		crawlDir := d.CrawlDir
		go func() {
			total, err := d.Docs.ScanAll(context.Background(), crawlID, crawlDir)
			if err != nil {
				return
			}
			_ = total
			if d.Hub != nil {
				d.Hub.BroadcastAll(map[string]string{"type": "shard_scan", "shard": "*"})
			}
		}()
		return c.JSON(http.StatusAccepted, MetaScanDocsResponse{Accepted: true, CrawlID: crawlID})
	}
}

func handleCrawlData(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		crawlID := c.Param("id")
		if crawlID == "" {
			return c.JSON(400, errResp{"missing crawl id"})
		}

		crawlDir := d.CrawlDir
		if d.ResolveCrawlDir != nil {
			crawlDir = d.ResolveCrawlDir(crawlID)
		}

		if d.Meta != nil {
			return c.JSON(200, d.Meta.GetSummary(c.Context(), crawlID, crawlDir))
		}

		if d.ScanDataDir != nil {
			ds := d.ScanDataDir(crawlDir)
			ds.CrawlID = crawlID
			return c.JSON(200, ds)
		}

		return c.JSON(200, DataSummary{CrawlID: crawlID})
	}
}

var dashboardEngines = []string{"dahlia", "tantivy"}

func handleEngines(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		all := index.List()
		var engines []string
		for _, e := range all {
			for _, want := range dashboardEngines {
				if e == want {
					engines = append(engines, e)
					break
				}
			}
		}
		if len(engines) == 0 {
			engines = all
		}
		return c.JSON(200, EnginesResponse{Engines: engines})
	}
}

func handleBrowseStats(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		shard := c.Query("shard")
		if shard == "" {
			return c.JSON(400, errResp{"shard required"})
		}
		if d.Docs == nil {
			return c.JSON(503, errResp{"doc store not available"})
		}
		stats, err := d.Docs.ShardStats(c.Context(), d.CrawlID, shard)
		if err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
		return c.JSON(200, stats)
	}
}

func handleDomainList(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		if d.Domains == nil {
			return c.JSON(503, errResp{"domain store not available"})
		}
		if err := d.Domains.EnsureFresh(c.Context()); err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
		page := queryInt(c, "page", 1)
		pageSize := queryInt(c, "page_size", 100)
		sortBy := c.Query("sort")
		q := c.Query("q")
		resp, err := d.Domains.ListDomains(c.Context(), sortBy, q, page, pageSize)
		if err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
		return c.JSON(200, resp)
	}
}

func handleDomainOverview(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		if d.Domains == nil {
			return c.JSON(503, errResp{"domain store not available"})
		}
		if err := d.Domains.EnsureFresh(c.Context()); err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
		ov, syncing, locked := d.Domains.GetOverviewStats(c.Context())
		return c.JSON(200, map[string]any{
			"overview": ov,
			"syncing":  syncing,
			"locked":   locked,
		})
	}
}

func handleDomainDetail(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		if d.Domains == nil {
			return c.JSON(503, errResp{"domain store not available"})
		}
		domain := c.Param("domain")
		if domain == "" {
			return c.JSON(400, errResp{"domain required"})
		}
		normFn := d.NormalizeDomainInput
		if normFn != nil {
			domain = normFn(domain)
		}
		if err := d.Domains.EnsureFresh(c.Context()); err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
		page := queryInt(c, "page", 1)
		pageSize := queryInt(c, "page_size", 100)
		sortBy := c.Query("sort")
		statusGroup := c.Query("status_group")
		resp, err := d.Domains.ListDomainURLs(c.Context(), domain, sortBy, statusGroup, page, pageSize)
		if err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
		return c.JSON(200, resp)
	}
}

func handleCCDomainFetch(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		if d.CCDomains == nil {
			return c.JSON(503, errResp{"cc domain store not available"})
		}
		type reqBody struct {
			Domain  string `json:"domain"`
			CrawlID string `json:"crawl_id"`
			MaxURLs int    `json:"max_urls"`
		}
		var body reqBody
		if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
			return c.JSON(400, errResp{"invalid JSON: " + err.Error()})
		}
		domain := body.Domain
		if d.NormalizeDomainInput != nil {
			domain = d.NormalizeDomainInput(domain)
		}
		if domain == "" {
			return c.JSON(400, errResp{"domain required"})
		}
		resp, err := d.CCDomains.FetchAndCache(c.Context(), domain, strings.TrimSpace(body.CrawlID), body.MaxURLs)
		if err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
		return c.JSON(200, resp)
	}
}

func handleCCDomainDetail(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		if d.CCDomains == nil {
			return c.JSON(503, errResp{"cc domain store not available"})
		}
		domain := c.Param("domain")
		if d.NormalizeDomainInput != nil {
			domain = d.NormalizeDomainInput(domain)
		}
		if domain == "" {
			return c.JSON(400, errResp{"domain required"})
		}
		page := queryInt(c, "page", 1)
		pageSize := queryInt(c, "page_size", 100)
		sortBy := c.Query("sort")
		statusGroup := c.Query("status_group")
		q := c.Query("q")
		crawlID := strings.TrimSpace(c.Query("crawl"))

		resp, err := d.CCDomains.GetDomainURLs(c.Context(), domain, crawlID, sortBy, statusGroup, q, page, pageSize)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return c.JSON(404, errResp{"domain not found in Common Crawl cache; fetch it first"})
			}
			return c.JSON(500, errResp{err.Error()})
		}
		return c.JSON(200, resp)
	}
}
