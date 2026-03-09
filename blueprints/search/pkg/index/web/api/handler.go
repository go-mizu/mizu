// Package api provides Mizu HTTP handlers for the pipeline dashboard API.
// It is decoupled from the main web.Server via the Deps struct.
package api

import (
	"context"
	"errors"
	"io/fs"
	"time"

	mizu "github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/metastore"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/scrape"
)

// ── Interfaces for web-package store dependencies ────────────────────────────

// MetaProvider is the interface satisfied by *web.MetaManager.
type MetaProvider interface {
	Backend() string
	IsStale(crawlID string) bool
	IsRefreshing(crawlID string) bool
	TriggerRefresh(crawlID, crawlDir string, force bool) bool
	Status(ctx context.Context, crawlID string) MetaStatus
	GetSummary(ctx context.Context, crawlID, crawlDir string) DataSummaryWithMeta
	ListWARCs(ctx context.Context, crawlID, crawlDir string) ([]metastore.WARCRecord, DataSummaryWithMeta, error)
	GetWARC(ctx context.Context, crawlID, crawlDir, warcIndex string) (metastore.WARCRecord, bool, DataSummaryWithMeta, error)
	Store() metastore.Store
	Close() error
}

// DocProvider is the interface satisfied by *web.DocStore.
type DocProvider interface {
	GetDoc(ctx context.Context, crawlID, shard, docID string) (DocRecord, bool, error)
	GetShardMeta(ctx context.Context, crawlID, shard string) (DocShardMeta, bool, error)
	ListShardMetas(ctx context.Context, crawlID string) ([]DocShardMeta, error)
	ListDocs(ctx context.Context, crawlID, shard string, page, pageSize int, q, sortBy string) ([]DocRecord, int64, error)
	IsScanning(crawlID, shard string) bool
	ScanShard(ctx context.Context, crawlID, shard, warcMdPath string) (int64, error)
	ScanAll(ctx context.Context, crawlID, crawlBase string) (int64, error)
	ShardStats(ctx context.Context, crawlID, shard string) (ShardStatsResponse, error)
	Close() error
}

// DomainProvider is the interface satisfied by *web.DomainStore.
type DomainProvider interface {
	EnsureFresh(ctx context.Context) error
	GetOverviewStats(ctx context.Context) (*DomainsOverview, bool, bool)
	ListDomains(ctx context.Context, sortBy, q string, page, pageSize int) (DomainsResponse, error)
	ListDomainURLs(ctx context.Context, domain, sortBy, statusGroup string, page, pageSize int) (DomainDetailResponse, error)
}

// CCDomainProvider is the interface satisfied by *web.CCDomainStore.
type CCDomainProvider interface {
	FetchAndCache(ctx context.Context, domain, crawlID string, maxURLs int) (CCDomainFetchResponse, error)
	GetDomainURLs(ctx context.Context, domain, crawlID, sortBy, statusGroup, q string, page, pageSize int) (CCDomainDetailResponse, error)
	Close() error
}

// HubBroadcaster allows handlers to notify WebSocket clients.
type HubBroadcaster interface {
	BroadcastAll(data any)
}

// ── Doc and domain types (used in DocProvider / DomainProvider interfaces) ───

// DocRecord is per-document metadata derived from a .md.warc.gz WARC record header.
type DocRecord struct {
	DocID        string    `json:"doc_id"`
	Shard        string    `json:"shard"`
	URL          string    `json:"url"`
	Host         string    `json:"host"`
	Title        string    `json:"title"`
	CrawlDate    time.Time `json:"crawl_date,omitempty"`
	SizeBytes    int64     `json:"size_bytes"`
	WordCount    int       `json:"word_count"`
	WARCRecordID string    `json:"warc_record_id,omitempty"`
	RefersTo     string    `json:"refers_to,omitempty"`
	GzipOffset   int64     `json:"gzip_offset,omitempty"`
	GzipSize     int64     `json:"gzip_size,omitempty"`
}

// DocShardMeta holds per-shard scan statistics.
type DocShardMeta struct {
	Shard          string
	TotalDocs      int64
	TotalSizeBytes int64
	LastDocDate    time.Time
	LastScannedAt  time.Time
}

// ShardStatsResponse holds aggregated statistics for a shard.
type ShardStatsResponse struct {
	Shard             string       `json:"shard"`
	TotalDocs         int64        `json:"total_docs"`
	TotalSize         int64        `json:"total_size"`
	AvgSize           int64        `json:"avg_size"`
	MinSize           int64        `json:"min_size"`
	MaxSize           int64        `json:"max_size"`
	DateFrom          string       `json:"date_from"`
	DateTo            string       `json:"date_to"`
	TotalDomains      int64        `json:"total_domains"`
	TopDomains        []DomainRow  `json:"top_domains"`
	SizeBuckets       []SizeBucket `json:"size_buckets"`
	DomainSizeBuckets []SizeBucket `json:"domain_size_buckets"`
}

// DomainRow is a domain + count pair.
type DomainRow struct {
	Domain string `json:"domain"`
	Count  int64  `json:"count"`
}

// SizeBucket is a size range + count pair.
type SizeBucket struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

// ── DomainStore types ─────────────────────────────────────────────────────────

// DomainsOverview holds aggregate domain stats.
type DomainsOverview struct {
	TotalDomains int64       `json:"total_domains"`
	TotalURLs    int64       `json:"total_urls"`
	SizeBuckets  []SizeItem  `json:"size_buckets"`
}

// SizeItem is a label + count pair for domain size distributions.
type SizeItem struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

// DomainsResponse is returned by GET /api/domains.
type DomainsResponse struct {
	Domains  []DomainURLRow   `json:"domains"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	Overview *DomainsOverview `json:"overview,omitempty"`
	Syncing  bool             `json:"syncing,omitempty"`
	Locked   bool             `json:"locked,omitempty"`
}

// DomainURLRow is one domain in the domain list.
type DomainURLRow struct {
	Domain  string `json:"domain"`
	Count   int64  `json:"count"`
	Syncing bool   `json:"syncing,omitempty"`
}

// DomainDetailResponse is returned by GET /api/domains/{domain}.
type DomainDetailResponse struct {
	Domain      string            `json:"domain"`
	Total       int64             `json:"total"`
	Page        int               `json:"page"`
	PageSize    int               `json:"page_size"`
	StatusGroup string            `json:"status_group,omitempty"`
	Stats       *DomainStats      `json:"stats,omitempty"`
	Docs        []DomainDocURLRow `json:"docs"`
	Syncing     bool              `json:"syncing,omitempty"`
}

// DomainDocURLRow is a URL in a domain detail view.
type DomainDocURLRow struct {
	URL         string `json:"url"`
	Shard       string `json:"shard"`
	FetchStatus int    `json:"fetch_status,omitempty"`
}

// StatusBucket is a status code + count pair.
type StatusBucket struct {
	Code  int   `json:"code"`
	Count int64 `json:"count"`
}

// DomainStats holds aggregate stats for a single domain.
type DomainStats struct {
	Total         int64          `json:"total"`
	StatusBuckets []StatusBucket `json:"status_buckets"`
}

// ── CCDomainStore types ───────────────────────────────────────────────────────

// CCDomainFetchResponse is returned by POST /api/domains/cc/fetch.
type CCDomainFetchResponse struct {
	Domain      string `json:"domain"`
	CrawlID     string `json:"crawl_id"`
	TotalURLs   int64  `json:"total_urls"`
	Truncated   bool   `json:"truncated"`
	FetchedAt   string `json:"fetched_at"`
	TotalPages  int    `json:"total_pages"`
	FetchedPage int    `json:"fetched_pages"`
}

// CCDomainURLRow is a single CC domain URL row.
type CCDomainURLRow struct {
	URL          string `json:"url"`
	FetchStatus  int    `json:"fetch_status,omitempty"`
	Mime         string `json:"mime,omitempty"`
	Timestamp    string `json:"timestamp,omitempty"`
	Filename     string `json:"filename,omitempty"`
	RecordLength int64  `json:"record_length,omitempty"`
	RecordOffset int64  `json:"record_offset,omitempty"`
	Digest       string `json:"digest,omitempty"`
	Language     string `json:"language,omitempty"`
	Encoding     string `json:"encoding,omitempty"`
}

// CCDomainDetailResponse is returned by GET /api/domains/cc/{domain}.
type CCDomainDetailResponse struct {
	Domain      string           `json:"domain"`
	CrawlID     string           `json:"crawl_id"`
	Total       int64            `json:"total"`
	Page        int              `json:"page"`
	PageSize    int              `json:"page_size"`
	Query       string           `json:"query,omitempty"`
	Sort        string           `json:"sort,omitempty"`
	StatusGroup string           `json:"status_group,omitempty"`
	CachedAt    string           `json:"cached_at,omitempty"`
	Truncated   bool             `json:"truncated,omitempty"`
	Stats       CCDomainStats    `json:"stats"`
	Docs        []CCDomainURLRow `json:"docs"`
}

// CCDomainStats holds aggregate CC domain stats.
type CCDomainStats struct {
	Total       int64            `json:"total"`
	StatusCodes []CCStatusBucket `json:"status_codes"`
	MimeTypes   []CCStringBucket `json:"mime_types"`
}

// CCStatusBucket is a status code + count pair.
type CCStatusBucket struct {
	Code  int   `json:"code"`
	Count int64 `json:"count"`
}

// CCStringBucket is a string + count pair.
type CCStringBucket struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

// ── Meta types ────────────────────────────────────────────────────────────────

// DataSummary holds statistics about a crawl's local data directory.
type DataSummary struct {
	CrawlID       string           `json:"crawl_id"`
	WARCCount     int              `json:"warc_count"`
	WARCTotalSize int64            `json:"warc_total_size"`
	MDShards      int              `json:"md_shards"`
	MDTotalSize   int64            `json:"md_total_size"`
	MDDocEstimate int              `json:"md_doc_estimate"`
	PackFormats   map[string]int64 `json:"pack_formats"`
	FTSEngines    map[string]int64 `json:"fts_engines"`
	FTSShardCount map[string]int   `json:"fts_shard_count"`
}

// DataSummaryWithMeta wraps DataSummary with cache metadata.
type DataSummaryWithMeta struct {
	DataSummary
	MetaBackend     string `json:"meta_backend,omitempty"`
	MetaGeneratedAt string `json:"meta_generated_at,omitempty"`
	MetaStale       bool   `json:"meta_stale"`
	MetaRefreshing  bool   `json:"meta_refreshing"`
	MetaLastError   string `json:"meta_last_error,omitempty"`
}

// MetaStatus is returned by /api/meta/status and /api/meta/refresh.
type MetaStatus struct {
	CrawlID      string `json:"crawl_id"`
	Backend      string `json:"backend"`
	Enabled      bool   `json:"enabled"`
	Status       string `json:"status"`
	Refreshing   bool   `json:"refreshing"`
	Generation   int64  `json:"generation"`
	StartedAt    string `json:"started_at,omitempty"`
	FinishedAt   string `json:"finished_at,omitempty"`
	LastError    string `json:"last_error,omitempty"`
	RefreshTTLMS int64  `json:"refresh_ttl_ms"`
}

// MetaRefreshResponse is returned by POST /api/meta/refresh.
type MetaRefreshResponse struct {
	Accepted bool       `json:"accepted"`
	Status   MetaStatus `json:"status"`
}

// ── Deps ─────────────────────────────────────────────────────────────────────

// Deps holds all dependencies shared by all dashboard API handlers.
type Deps struct {
	// Pipeline
	Jobs   *pipeline.Manager
	Scrape *scrape.Store

	// Core identity
	CrawlID    string
	CrawlDir   string
	EngineName string
	WARCBase   string
	WARCMdBase string
	Addr       string // external FTS engine address

	// Store interfaces (nil in search-only mode)
	Meta      MetaProvider
	Docs      DocProvider
	Domains   DomainProvider
	CCDomains CCDomainProvider
	Hub       HubBroadcaster

	// Static file system (for index page)
	StaticFS fs.FS

	// Function fields for operations requiring web-package logic
	ResolveFTSBase   func(engine string) string
	ResolveCrawlDir  func(crawlID string) string
	ManifestTotal    func() int
	CrawlSizeBytes   func() int64
	RefreshCrawlSize func()
	ScanDataDir      func(crawlDir string) DataSummary
	CCConfig         func() cc.Config

	// Function fields for overview response (avoids importing overview types)
	BuildOverviewResponse func(crawlID, crawlDir string, manifestTotal int, crawlBytes int64) any

	// Helpers for doc/browse handlers
	ReadDocByOffset    func(warcMdPath string, offset, size int64) ([]byte, error)
	ReadDocFromWARCMd  func(warcMdPath, docID string) ([]byte, bool, error)
	ExtractDocTitle    func(head []byte, fallbackURL string) string
	RenderMarkdown     func(src []byte) (string, error)
	SumInt64Map        func(m map[string]int64) int64
	IsNumericName      func(s string) bool
	WARCIndexFromPath  func(path string) (string, bool)
	FormatWARCIndex    func(i int) string
	FormatBytes        func(b int64) string
	NormalizeDomainInput func(raw string) string

	// Helpers for WARC console handlers (provided by web.Server)
	ListWARCsFallback  func(ctx context.Context, crawlID, crawlDir string) ([]metastore.WARCRecord, DataSummaryWithMeta)
	SummarizeWARCs     func(recs []metastore.WARCRecord) WARCSummary
	BuildWARCRow       func(ctx context.Context, rec metastore.WARCRecord, crawlDir string) WARCAPIRecord
	CollectSystemStats func(crawlDir string) WARCSystemStats
	DeleteWARCArtifacts func(crawlDir, localIdx, target, format, engine string) ([]string, error)

	// ExportShardMeta runs the browse parquet export (provided by web package).
	ExportShardMeta func(ctx context.Context, metaPath, warcMdPath, outPath string, includeMarkdownBody bool) (int64, error)
}

// ErrCCDomainNotFound is returned by CCDomainProvider.GetDomainURLs when the
// domain has not been fetched and cached yet.
var ErrCCDomainNotFound = errors.New("cc domain not found in cache")

// errResp is the standard JSON error shape.
type errResp struct {
	Error string `json:"error"`
}

// RegisterJobRoutes mounts the job-management API routes on router.
func RegisterJobRoutes(router *mizu.Router, d *Deps) {
	router.Get("/api/jobs", listJobs(d))
	router.Get("/api/jobs/{id}", getJob(d))
	router.Post("/api/jobs", createJob(d))
	router.Delete("/api/jobs/{id}", cancelJob(d))
	router.Delete("/api/jobs", clearJobs(d))
}

// RegisterScrapeRoutes mounts the scrape pipeline API routes on router.
func RegisterScrapeRoutes(router *mizu.Router, d *Deps) {
	router.Post("/api/scrape", startScrape(d))
	router.Get("/api/scrape/list", listScrape(d))
	router.Post("/api/scrape/{domain}/resume", resumeScrape(d))
	router.Delete("/api/scrape/{domain}", stopScrape(d))
	router.Get("/api/scrape/{domain}/status", scrapeStatus(d))
	router.Get("/api/scrape/{domain}/pages", scrapePages(d))
	router.Post("/api/scrape/{domain}/pipeline", scrapePipeline(d))
}

// RegisterDashboardRoutes mounts all dashboard (non-job) routes.
func RegisterDashboardRoutes(router *mizu.Router, d *Deps) {
	// Static index
	router.Get("/", handleIndex(d))

	// Core API
	router.Get("/api/search", handleSearch(d))
	router.Get("/api/stats", handleStats(d))
	router.Get("/api/doc/{shard}/{docid...}", handleDoc(d))
	router.Get("/api/browse", handleBrowse(d))

	// Dashboard-only (requires Hub != nil)
	if d.Hub != nil {
		router.Get("/api/overview", handleOverview(d))
		router.Get("/api/meta/status", handleMetaStatus(d))
		router.Post("/api/meta/refresh", handleMetaRefresh(d))
		router.Post("/api/meta/scan-docs", handleMetaScanDocs(d))
		router.Get("/api/crawl/{id}/data", handleCrawlData(d))
		router.Get("/api/engines", handleEngines(d))
		router.Get("/api/warc", handleWARCList(d))
		router.Get("/api/warc/{index}", handleWARCDetail(d))
		router.Post("/api/warc/{index}/action", handleWARCAction(d))
		router.Get("/api/browse/stats", handleBrowseStats(d))
		router.Post("/api/browse/export-parquet", handleBrowseExportParquet(d))
		router.Get("/api/domains", handleDomainList(d))
		router.Get("/api/domains/overview", handleDomainOverview(d))
		router.Post("/api/domains/cc/fetch", handleCCDomainFetch(d))
		router.Get("/api/domains/cc/{domain}", handleCCDomainDetail(d))
		router.Get("/api/domains/{domain}", handleDomainDetail(d))
		router.Get("/api/parquet/manifest", handleParquetManifest(d))
		router.Get("/api/parquet/schema", handleParquetSchema(d))
		router.Post("/api/parquet/query", handleParquetQuery(d))
		router.Post("/api/parquet/download", handleParquetDownload(d))
		router.Get("/api/parquet/stats", handleParquetStats(d))
		router.Get("/api/parquet/file/{index}", handleParquetFileDetail(d))
		router.Get("/api/parquet/file/{index}/data", handleParquetFileData(d))
		router.Get("/api/parquet/file/{index}/stats", handleParquetFileStats(d))
		router.Get("/api/parquet/subset/{subset}/stats", handleParquetSubsetStats(d))
		RegisterJobRoutes(router, d)
		RegisterScrapeRoutes(router, d)
	}
}
