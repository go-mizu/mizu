package web

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// crawlSizeCache caches the result of dirSize(crawlDir) so the overview
// endpoint returns instantly instead of walking 250K+ files synchronously.
type crawlSizeCache struct {
	bytes   atomic.Int64
	running atomic.Bool
	mu      sync.Mutex
	dir     string
}

func newCrawlSizeCache(dir string) *crawlSizeCache {
	c := &crawlSizeCache{dir: dir}
	c.refresh() // kick off first background scan
	return c
}

func (c *crawlSizeCache) get() int64 {
	return c.bytes.Load()
}

func (c *crawlSizeCache) refresh() {
	if !c.running.CompareAndSwap(false, true) {
		return // already running
	}
	go func() {
		defer c.running.Store(false)
		c.bytes.Store(dirSize(c.dir))
	}()
}

// hostMemInfo is filled by platform-specific sysinfo_*.go files.
// Returns (total, available) bytes. Returns zeros if unavailable.
var hostMemInfo func() (total, avail int64)

// hostNetBytes returns cumulative (recv, sent) bytes across all interfaces.
var hostNetBytes func() (recv, sent int64)

// hostDiskBytes returns cumulative (read, written) bytes across all block devices.
var hostDiskBytes func() (read, written int64)

var startTime = time.Now()

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

type ManifestStage struct {
	TotalWARCs        int   `json:"total_warcs"`
	EstTotalSizeBytes int64 `json:"est_total_size_bytes"`
	EstTotalURLs      int64 `json:"est_total_urls"`
}

type DownloadedStage struct {
	Count        int   `json:"count"`
	TotalWARCs   int   `json:"total_warcs"`
	SizeBytes    int64 `json:"size_bytes"`
	AvgWARCBytes int64 `json:"avg_warc_bytes"`
}

type MarkdownStage struct {
	Count          int   `json:"count"`
	TotalWARCs     int   `json:"total_warcs"`
	SizeBytes      int64 `json:"size_bytes"`
	TotalDocs      int64 `json:"total_docs"`
	AvgDocsPerWARC int64 `json:"avg_docs_per_warc"`
	AvgDocBytes    int64 `json:"avg_doc_bytes"`
}

type PackStage struct {
	Count        int   `json:"count"`
	TotalWARCs   int   `json:"total_warcs"`
	ParquetBytes int64 `json:"parquet_bytes"`
	WARCMdBytes  int64 `json:"warc_md_bytes"`
}

type IndexedStage struct {
	Count         int   `json:"count"`
	TotalWARCs    int   `json:"total_warcs"`
	DahliaBytes   int64 `json:"dahlia_bytes"`
	DahliaShards  int   `json:"dahlia_shards"`
	TantivyBytes  int64 `json:"tantivy_bytes"`
	TantivyShards int   `json:"tantivy_shards"`
}

type StorageInfo struct {
	DiskTotal          int64 `json:"disk_total"`
	DiskUsed           int64 `json:"disk_used"`
	DiskFree           int64 `json:"disk_free"`
	CrawlBytes         int64 `json:"crawl_bytes"`
	ProjectedFullBytes int64 `json:"projected_full_bytes"`
}

type SystemInfo struct {
	// Go runtime
	HeapAlloc  int64  `json:"heap_alloc"`
	HeapSys    int64  `json:"heap_sys"`
	StackInuse int64  `json:"stack_inuse"`
	NumGC      int64  `json:"num_gc"`
	Goroutines int    `json:"goroutines"`
	GoVersion  string `json:"go_version"`
	Uptime     int64  `json:"uptime_seconds"`
	PID        int    `json:"pid"`
	GOMEMLIMIT int64  `json:"gomemlimit"`

	// Host hardware
	CPUs          int    `json:"cpus"`
	GOOS          string `json:"goos"`
	GOARCH        string `json:"goarch"`
	MemTotalBytes int64  `json:"mem_total_bytes"`
	MemAvailBytes int64  `json:"mem_avail_bytes"`
	MemUsedBytes  int64  `json:"mem_used_bytes"`

	// Network (cumulative, since boot)
	NetBytesRecv int64 `json:"net_bytes_recv"`
	NetBytesSent int64 `json:"net_bytes_sent"`

	// Disk I/O (cumulative, since boot)
	DiskReadBytes  int64 `json:"disk_read_bytes"`
	DiskWriteBytes int64 `json:"disk_write_bytes"`
}

type OverviewMeta struct {
	Backend     string `json:"backend"`
	GeneratedAt string `json:"generated_at"`
	Stale       bool   `json:"stale"`
	Refreshing  bool   `json:"refreshing"`
}

// buildOverviewResponse scans the crawl directory and assembles a structured
// overview with pipeline stage stats, storage info, and runtime metrics.
func buildOverviewResponse(crawlID, crawlDir string, manifestTotal int, docs *DocStore, crawlBytes int64) OverviewResponse {
	resp := OverviewResponse{
		CrawlID: crawlID,
	}

	// Stage 1: Manifest
	resp.Manifest = ManifestStage{
		TotalWARCs:   manifestTotal,
		EstTotalURLs: int64(manifestTotal) * 30_000,
	}

	// Stage 2: Downloaded
	resp.Downloaded = scanDownloadedStage(crawlDir)
	resp.Downloaded.TotalWARCs = manifestTotal

	// Estimate manifest total size from downloaded average
	if resp.Downloaded.Count > 0 && resp.Downloaded.AvgWARCBytes > 0 {
		resp.Manifest.EstTotalSizeBytes = int64(manifestTotal) * resp.Downloaded.AvgWARCBytes
	}

	// Stage 3: Markdown
	resp.Markdown = scanMarkdownStage(crawlDir, docs)
	resp.Markdown.TotalWARCs = resp.Downloaded.Count

	// Stage 4: Pack
	resp.Pack = scanPackStage(crawlDir)
	resp.Pack.TotalWARCs = resp.Downloaded.Count

	// Stage 5: FTS Index
	resp.Indexed = scanIndexedStage(crawlDir)
	resp.Indexed.TotalWARCs = resp.Downloaded.Count

	// Storage
	resp.Storage = collectStorageInfo(crawlDir, manifestTotal, resp.Downloaded.AvgWARCBytes, crawlBytes)

	// System
	resp.System = collectSystemInfo()

	// Meta
	resp.Meta = OverviewMeta{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

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

func scanMarkdownStage(crawlDir string, docs *DocStore) MarkdownStage {
	var stage MarkdownStage
	warcMdDir := filepath.Join(crawlDir, "warc_md")
	entries, err := os.ReadDir(warcMdDir)
	if err != nil {
		return stage
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md.warc.gz") {
			continue
		}
		shard := strings.TrimSuffix(e.Name(), ".md.warc.gz")
		if !isNumericName(shard) {
			continue
		}
		stage.Count++
		if info, err := e.Info(); err == nil {
			stage.SizeBytes += info.Size()
		}
	}

	// Get total docs from DocStore if available
	if docs != nil && stage.Count > 0 {
		// Sum docs across all scanned shards
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md.warc.gz") {
				continue
			}
			shard := strings.TrimSuffix(e.Name(), ".md.warc.gz")
			if !isNumericName(shard) {
				continue
			}
			meta, ok, _ := docs.GetShardMeta(context.Background(), "", shard)
			if ok {
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

func scanPackStage(crawlDir string) PackStage {
	var stage PackStage
	seen := make(map[string]bool)

	// Parquet
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
			if isNumericName(idx) {
				seen[idx] = true
			}
			if info, err := e.Info(); err == nil {
				stage.ParquetBytes += info.Size()
			}
		}
	}

	// .md.warc.gz (from warc_md/)
	warcMdDir := filepath.Join(crawlDir, "warc_md")
	if entries, err := os.ReadDir(warcMdDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md.warc.gz") {
				continue
			}
			shard := strings.TrimSuffix(e.Name(), ".md.warc.gz")
			if isNumericName(shard) {
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

func scanIndexedStage(crawlDir string) IndexedStage {
	var stage IndexedStage
	ftsDir := filepath.Join(crawlDir, "fts")
	seen := make(map[string]bool)

	// Dahlia
	dahliaDir := filepath.Join(ftsDir, "dahlia")
	if shards, err := os.ReadDir(dahliaDir); err == nil {
		for _, s := range shards {
			if !s.IsDir() || !isNumericName(s.Name()) {
				continue
			}
			stage.DahliaShards++
			seen[s.Name()] = true
			stage.DahliaBytes += dirSize(filepath.Join(dahliaDir, s.Name()))
		}
	}

	// Tantivy
	tantivyDir := filepath.Join(ftsDir, "tantivy")
	if shards, err := os.ReadDir(tantivyDir); err == nil {
		for _, s := range shards {
			if !s.IsDir() || !isNumericName(s.Name()) {
				continue
			}
			stage.TantivyShards++
			seen[s.Name()] = true
			stage.TantivyBytes += dirSize(filepath.Join(tantivyDir, s.Name()))
		}
	}

	stage.Count = len(seen)
	return stage
}

func collectSystemInfo() SystemInfo {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	goVersion := runtime.Version()
	if bi, ok := debug.ReadBuildInfo(); ok && bi.GoVersion != "" {
		goVersion = bi.GoVersion
	}

	// GOMEMLIMIT from debug.SetMemoryLimit(-1) returns current limit without changing it.
	memlimit := debug.SetMemoryLimit(-1)

	si := SystemInfo{
		HeapAlloc:  int64(ms.HeapAlloc),
		HeapSys:    int64(ms.HeapSys),
		StackInuse: int64(ms.StackInuse),
		NumGC:      int64(ms.NumGC),
		Goroutines: runtime.NumGoroutine(),
		GoVersion:  goVersion,
		Uptime:     int64(time.Since(startTime).Seconds()),
		PID:        os.Getpid(),
		GOMEMLIMIT: memlimit,
		CPUs:       runtime.NumCPU(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	}

	if hostMemInfo != nil {
		si.MemTotalBytes, si.MemAvailBytes = hostMemInfo()
		si.MemUsedBytes = si.MemTotalBytes - si.MemAvailBytes
	}
	if hostNetBytes != nil {
		si.NetBytesRecv, si.NetBytesSent = hostNetBytes()
	}
	if hostDiskBytes != nil {
		si.DiskReadBytes, si.DiskWriteBytes = hostDiskBytes()
	}

	return si
}

func collectStorageInfo(crawlDir string, manifestTotal int, avgWARCBytes int64, crawlBytes int64) StorageInfo {
	info := StorageInfo{}

	// Total crawl bytes on disk (pre-computed by background cache)
	info.CrawlBytes = crawlBytes

	// Disk usage from statfs
	info.DiskTotal, info.DiskFree = diskUsage(crawlDir)
	if info.DiskTotal > 0 {
		info.DiskUsed = info.DiskTotal - info.DiskFree
	}

	// Projected full crawl size
	if manifestTotal > 0 && avgWARCBytes > 0 {
		info.ProjectedFullBytes = int64(manifestTotal) * avgWARCBytes
	}

	return info
}
