package pipeline

import (
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
type CrawlSizeCache struct {
	bytes   atomic.Int64
	running atomic.Bool
	mu      sync.Mutex
	dir     string
}

func NewCrawlSizeCache(dir string) *CrawlSizeCache {
	c := &CrawlSizeCache{dir: dir}
	c.Refresh()
	return c
}

func (c *CrawlSizeCache) Get() int64 {
	return c.bytes.Load()
}

func (c *CrawlSizeCache) Refresh() {
	if !c.running.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer c.running.Store(false)
		c.bytes.Store(dirSize(c.dir))
	}()
}

// hostMemInfo is filled by platform-specific sysinfo_*.go files.
var HostMemInfo func() (total, avail int64)

// hostNetBytes returns cumulative (recv, sent) bytes across all interfaces.
var HostNetBytes func() (recv, sent int64)

// hostDiskBytes returns cumulative (read, written) bytes across all block devices.
var HostDiskBytes func() (read, written int64)

var startTime = time.Now()

// Response is the structured /api/overview payload with pipeline stages.
type Response struct {
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
	Meta    ResponseMeta `json:"meta"`
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

type ResponseMeta struct {
	Backend     string `json:"backend"`
	GeneratedAt string `json:"generated_at"`
	Stale       bool   `json:"stale"`
	Refreshing  bool   `json:"refreshing"`
}

// ShardDocCounter returns the doc count for a shard; used to populate markdown stage.
type ShardDocCounter func(shard string) (int64, bool)

// BuildOverview scans the crawl directory and assembles a structured overview
// with pipeline stage stats, storage info, and runtime metrics.
func BuildOverview(crawlID, crawlDir string, manifestTotal int, docCounter ShardDocCounter, crawlBytes int64) Response {
	resp := Response{CrawlID: crawlID}

	resp.Manifest = ManifestStage{
		TotalWARCs:   manifestTotal,
		EstTotalURLs: int64(manifestTotal) * 30_000,
	}

	resp.Downloaded = scanDownloaded(crawlDir)
	resp.Downloaded.TotalWARCs = manifestTotal

	if resp.Downloaded.Count > 0 && resp.Downloaded.AvgWARCBytes > 0 {
		resp.Manifest.EstTotalSizeBytes = int64(manifestTotal) * resp.Downloaded.AvgWARCBytes
	}

	resp.Markdown = scanMarkdown(crawlDir, docCounter)
	resp.Markdown.TotalWARCs = resp.Downloaded.Count

	resp.Pack = scanPack(crawlDir)
	resp.Pack.TotalWARCs = resp.Downloaded.Count

	resp.Indexed = scanIndexed(crawlDir)
	resp.Indexed.TotalWARCs = resp.Downloaded.Count

	resp.Storage = collectStorage(crawlDir, manifestTotal, resp.Downloaded.AvgWARCBytes, crawlBytes)
	resp.System = collectSystem()
	resp.Meta = ResponseMeta{GeneratedAt: time.Now().UTC().Format(time.RFC3339)}

	return resp
}

func scanDownloaded(crawlDir string) DownloadedStage {
	var stage DownloadedStage
	entries, err := os.ReadDir(filepath.Join(crawlDir, "warc"))
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

func scanMarkdown(crawlDir string, docCounter ShardDocCounter) MarkdownStage {
	var stage MarkdownStage
	entries, _ := os.ReadDir(filepath.Join(crawlDir, "warc_md"))
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
		if docCounter != nil {
			if n, ok := docCounter(shard); ok {
				stage.TotalDocs += n
			}
		}
	}
	if stage.Count > 0 && stage.TotalDocs > 0 {
		stage.AvgDocsPerWARC = stage.TotalDocs / int64(stage.Count)
		stage.AvgDocBytes = stage.SizeBytes / stage.TotalDocs
	}
	return stage
}

func scanPack(crawlDir string) PackStage {
	var stage PackStage
	seen := make(map[string]bool)

	if entries, err := os.ReadDir(filepath.Join(crawlDir, "pack", "parquet")); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if !strings.HasSuffix(e.Name(), ".parquet") {
				continue
			}
			idx := strings.TrimSuffix(e.Name(), ".parquet")
			if isNumericName(idx) {
				seen[idx] = true
			}
			if info, err := e.Info(); err == nil {
				stage.ParquetBytes += info.Size()
			}
		}
	}

	if entries, err := os.ReadDir(filepath.Join(crawlDir, "warc_md")); err == nil {
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

func scanIndexed(crawlDir string) IndexedStage {
	var stage IndexedStage
	ftsDir := filepath.Join(crawlDir, "fts")
	seen := make(map[string]bool)

	if shards, err := os.ReadDir(filepath.Join(ftsDir, "dahlia")); err == nil {
		for _, s := range shards {
			if !s.IsDir() || !isNumericName(s.Name()) {
				continue
			}
			stage.DahliaShards++
			seen[s.Name()] = true
			stage.DahliaBytes += dirSize(filepath.Join(ftsDir, "dahlia", s.Name()))
		}
	}

	if shards, err := os.ReadDir(filepath.Join(ftsDir, "tantivy")); err == nil {
		for _, s := range shards {
			if !s.IsDir() || !isNumericName(s.Name()) {
				continue
			}
			stage.TantivyShards++
			seen[s.Name()] = true
			stage.TantivyBytes += dirSize(filepath.Join(ftsDir, "tantivy", s.Name()))
		}
	}

	stage.Count = len(seen)
	return stage
}

func collectSystem() SystemInfo {
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
		Uptime:     int64(time.Since(startTime).Seconds()),
		PID:        os.Getpid(),
		GOMEMLIMIT: memlimit,
		CPUs:       runtime.NumCPU(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	}

	if HostMemInfo != nil {
		si.MemTotalBytes, si.MemAvailBytes = HostMemInfo()
		si.MemUsedBytes = si.MemTotalBytes - si.MemAvailBytes
	}
	if HostNetBytes != nil {
		si.NetBytesRecv, si.NetBytesSent = HostNetBytes()
	}
	if HostDiskBytes != nil {
		si.DiskReadBytes, si.DiskWriteBytes = HostDiskBytes()
	}

	return si
}

func collectStorage(crawlDir string, manifestTotal int, avgWARCBytes, crawlBytes int64) StorageInfo {
	info := StorageInfo{CrawlBytes: crawlBytes}
	info.DiskTotal, info.DiskFree = diskUsage(crawlDir)
	if info.DiskTotal > 0 {
		info.DiskUsed = info.DiskTotal - info.DiskFree
	}
	if manifestTotal > 0 && avgWARCBytes > 0 {
		info.ProjectedFullBytes = int64(manifestTotal) * avgWARCBytes
	}
	return info
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

func dirSize(path string) int64 {
	var size int64
	_ = filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			size += info.Size()
		}
		return nil
	})
	return size
}
