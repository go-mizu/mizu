package api

import (
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline"
)

// ── Search / Browse response types ───────────────────────────────────────────

// SearchResponse is returned by GET /api/search.
type SearchResponse struct {
	Hits      []SearchHit `json:"hits"`
	Total     int         `json:"total"`
	ElapsedMs int64       `json:"elapsed_ms"`
	Query     string      `json:"query"`
	Engine    string      `json:"engine"`
	Shards    int         `json:"shards"`
}

// SearchHit is one result in a SearchResponse.
type SearchHit struct {
	DocID     string     `json:"doc_id"`
	Shard     string     `json:"shard"`
	Score     float64    `json:"score,omitempty"`
	Snippet   string     `json:"snippet,omitempty"`
	URL       string     `json:"url,omitempty"`
	Host      string     `json:"host,omitempty"`
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
	Shards     []ShardEntry `json:"shards"`
	HasDocMeta bool         `json:"has_doc_meta"`
}

// ShardEntry is one shard in a BrowseShardsResponse.
type ShardEntry struct {
	Name          string `json:"name"`
	HasPack       bool   `json:"has_pack"`
	HasFTS        bool   `json:"has_fts"`
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
	Docs          []DocJSON `json:"docs"`
	Total         int       `json:"total"`
	Page          int       `json:"page"`
	PageSize      int       `json:"page_size,omitempty"`
	MetaStale     bool      `json:"meta_stale,omitempty"`
	Scanning      bool      `json:"scanning"`
	NotScanned    bool      `json:"not_scanned,omitempty"`
	LastScannedAt string    `json:"last_scanned_at,omitempty"`
}

// DocJSON is a document in a BrowseDocsResponse.
type DocJSON struct {
	DocID     string `json:"doc_id"`
	Shard     string `json:"shard"`
	URL       string `json:"url"`
	Host      string `json:"host"`
	Title     string `json:"title"`
	CrawlDate string `json:"crawl_date,omitempty"`
	SizeBytes int64  `json:"size_bytes"`
	WordCount int    `json:"word_count"`
}

// ── Dashboard response types ──────────────────────────────────────────────────

// MetaScanDocsResponse is returned by POST /api/meta/scan-docs.
type MetaScanDocsResponse struct {
	Accepted bool   `json:"accepted"`
	CrawlID  string `json:"crawl_id,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// EnginesResponse is returned by GET /api/engines.
type EnginesResponse struct {
	Engines []string `json:"engines"`
}

// ── WARC response types ───────────────────────────────────────────────────────

// WARCListResponse is returned by GET /api/warc.
type WARCListResponse struct {
	CrawlID         string          `json:"crawl_id"`
	Offset          int             `json:"offset"`
	Limit           int             `json:"limit"`
	Total           int             `json:"total"`
	Summary         WARCSummary     `json:"summary"`
	WARCs           []WARCAPIRecord `json:"warcs"`
	System          WARCSystemStats `json:"system"`
	MetaBackend     string          `json:"meta_backend"`
	MetaGeneratedAt string          `json:"meta_generated_at"`
	MetaStale       bool            `json:"meta_stale"`
	MetaRefreshing  bool            `json:"meta_refreshing"`
	MetaLastError   string          `json:"meta_last_error"`
}

// WARCDetailResponse is returned by GET /api/warc/{index}.
type WARCDetailResponse struct {
	CrawlID         string           `json:"crawl_id"`
	WARC            WARCAPIRecord    `json:"warc"`
	Jobs            []*pipeline.Job  `json:"jobs"`
	System          WARCSystemStats  `json:"system"`
	MetaBackend     string           `json:"meta_backend"`
	MetaGeneratedAt string           `json:"meta_generated_at"`
	MetaStale       bool             `json:"meta_stale"`
	MetaRefreshing  bool             `json:"meta_refreshing"`
	MetaLastError   string           `json:"meta_last_error"`
}

// WARCActionResponse is returned by POST /api/warc/{index}/action.
type WARCActionResponse struct {
	OK              bool            `json:"ok"`
	Action          string          `json:"action"`
	CrawlID         string          `json:"crawl_id"`
	WARCIndex       string          `json:"warc_index"`
	Job             *pipeline.Job   `json:"job"`
	DeletedPaths    []string        `json:"deleted_paths"`
	RefreshAccepted bool            `json:"refresh_accepted"`
}

// WARCSummary holds aggregated WARC stats for the list view.
type WARCSummary struct {
	Total         int   `json:"total"`
	Downloaded    int   `json:"downloaded"`
	MarkdownReady int   `json:"markdown_ready"`
	Packed        int   `json:"packed"`
	Indexed       int   `json:"indexed"`
	WARCBytes     int64 `json:"warc_bytes"`
	MarkdownBytes int64 `json:"markdown_bytes"`
	PackBytes     int64 `json:"pack_bytes"`
	FTSBytes      int64 `json:"fts_bytes"`
	TotalBytes    int64 `json:"total_bytes"`
}

// WARCSystemStats holds runtime system stats included in WARC responses.
type WARCSystemStats struct {
	MemAlloc      int64 `json:"mem_alloc"`
	MemHeapSys    int64 `json:"mem_heap_sys"`
	MemStackInuse int64 `json:"mem_stack_inuse"`
	Goroutines    int   `json:"goroutines"`
	DiskTotal     int64 `json:"disk_total"`
	DiskUsed      int64 `json:"disk_used"`
	DiskFree      int64 `json:"disk_free"`
}

// WARCAPIRecord is a single WARC entry in the list/detail view.
type WARCAPIRecord struct {
	Index         string           `json:"index"`
	ManifestIndex int64            `json:"manifest_index"`
	Filename      string           `json:"filename"`
	RemotePath    string           `json:"remote_path"`
	WARCBytes     int64            `json:"warc_bytes"`
	WARCMdBytes   int64            `json:"warc_md_bytes"`
	WARCMdDocs    int64            `json:"warc_md_docs"`
	MarkdownDocs  int64            `json:"markdown_docs"`
	MarkdownBytes int64            `json:"markdown_bytes"`
	PackBytes     map[string]int64 `json:"pack_bytes"`
	FTSBytes      map[string]int64 `json:"fts_bytes"`
	TotalBytes    int64            `json:"total_bytes"`
	HasWARC       bool             `json:"has_warc"`
	HasMarkdown   bool             `json:"has_markdown"`
	HasPack       bool             `json:"has_pack"`
	HasFTS        bool             `json:"has_fts"`
	UpdatedAt     string           `json:"updated_at,omitempty"`
}

// ── Parquet response types ────────────────────────────────────────────────────

// ParquetManifestResponse is returned by GET /api/parquet/manifest.
type ParquetManifestResponse struct {
	Files   []ParquetFileEntry    `json:"files"`
	Summary ParquetManifestSummary `json:"summary"`
	Total   int                    `json:"total"`
	Offset  int                    `json:"offset"`
	Limit   int                    `json:"limit"`
}

// ParquetFileEntry is one file in the parquet manifest.
type ParquetFileEntry struct {
	ManifestIndex int    `json:"manifest_index"`
	RemotePath    string `json:"remote_path"`
	Filename      string `json:"filename"`
	Subset        string `json:"subset"`
	Downloaded    bool   `json:"downloaded"`
	Invalid       bool   `json:"invalid,omitempty"`
	LocalSize     int64  `json:"local_size,omitempty"`
}

// ParquetManifestSummary holds aggregate stats for the manifest.
type ParquetManifestSummary struct {
	Total      int                       `json:"total"`
	Downloaded int                       `json:"downloaded"`
	Invalid    int                       `json:"invalid,omitempty"`
	DiskBytes  int64                     `json:"disk_bytes"`
	BySubset   map[string]SubsetSummary  `json:"by_subset"`
}

// SubsetSummary is a per-subset download summary.
type SubsetSummary struct {
	Total      int `json:"total"`
	Downloaded int `json:"downloaded"`
}

// ParquetSchemaResponse is returned by GET /api/parquet/schema.
type ParquetSchemaResponse struct {
	Columns []ParquetColumnInfo `json:"columns"`
	Source  string              `json:"source"`
}

// ParquetColumnInfo is one column in a parquet schema.
type ParquetColumnInfo struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Order int    `json:"order"`
}

// ParquetQueryRequest is the body for POST /api/parquet/query.
type ParquetQueryRequest struct {
	SQL   string `json:"sql"`
	Limit int    `json:"limit"`
}

// ParquetQueryResponse is returned by POST /api/parquet/query.
type ParquetQueryResponse struct {
	Columns   []string        `json:"columns"`
	Rows      [][]interface{} `json:"rows"`
	TotalRows int             `json:"total_rows"`
	ElapsedMs int64           `json:"elapsed_ms"`
	Truncated bool            `json:"truncated"`
}

// ParquetDownloadRequest is the body for POST /api/parquet/download.
type ParquetDownloadRequest struct {
	Subset  string `json:"subset"`
	Indices []int  `json:"indices,omitempty"`
	Sample  int    `json:"sample,omitempty"`
}

// ParquetDownloadResponse is returned by POST /api/parquet/download.
type ParquetDownloadResponse struct {
	Started   bool   `json:"started"`
	Message   string `json:"message"`
	FileCount int    `json:"file_count"`
	JobID     string `json:"job_id,omitempty"`
}

// ParquetFileDetailResponse is returned by GET /api/parquet/file/{index}.
type ParquetFileDetailResponse struct {
	ManifestIndex int                 `json:"manifest_index"`
	RemotePath    string              `json:"remote_path"`
	Filename      string              `json:"filename"`
	Subset        string              `json:"subset"`
	Downloaded    bool                `json:"downloaded"`
	LocalSize     int64               `json:"local_size,omitempty"`
	LocalPath     string              `json:"local_path,omitempty"`
	RowCount      int64               `json:"row_count,omitempty"`
	Columns       []ParquetColumnInfo `json:"columns,omitempty"`
}

// ParquetFileDataResponse is returned by GET /api/parquet/file/{index}/data.
type ParquetFileDataResponse struct {
	Columns   []string        `json:"columns"`
	Rows      [][]interface{} `json:"rows"`
	Total     int64           `json:"total"`
	Page      int             `json:"page"`
	PageSize  int             `json:"page_size"`
	ElapsedMs int64           `json:"elapsed_ms"`
}

// ParquetStatsResponse is returned by GET /api/parquet/stats.
type ParquetStatsResponse struct {
	LocalFiles    int    `json:"local_files"`
	TotalRows     int64  `json:"total_rows"`
	DiskBytes     int64  `json:"disk_bytes"`
	SchemaColumns int    `json:"schema_columns"`
	CrawlID       string `json:"crawl_id"`
}

// ChartEntry is one entry in a distribution chart.
type ChartEntry struct {
	Label string `json:"label"`
	Value int64  `json:"value"`
}

// ParquetSubsetStatsResponse is returned by GET /api/parquet/subset/{subset}/stats.
type ParquetSubsetStatsResponse struct {
	Subset    string                  `json:"subset"`
	TotalRows int64                   `json:"total_rows"`
	FileCount int                     `json:"file_count"`
	DiskBytes int64                   `json:"disk_bytes"`
	ElapsedMs int64                   `json:"elapsed_ms"`
	Charts    map[string][]ChartEntry `json:"charts"`
}

// ParquetFileStatsResponse is returned by GET /api/parquet/file/{index}/stats.
type ParquetFileStatsResponse struct {
	ManifestIndex int                     `json:"manifest_index"`
	Subset        string                  `json:"subset"`
	RowCount      int64                   `json:"row_count"`
	ElapsedMs     int64                   `json:"elapsed_ms"`
	KPIs          map[string]float64      `json:"kpis,omitempty"`
	Charts        map[string][]ChartEntry `json:"charts"`
}

// ── Browse export types ───────────────────────────────────────────────────────

// BrowseExportParquetRequest is the body for POST /api/browse/export-parquet.
type BrowseExportParquetRequest struct {
	Shard               string `json:"shard"`
	IncludeMarkdownBody *bool  `json:"include_markdown_body,omitempty"`
	Overwrite           bool   `json:"overwrite,omitempty"`
}

// BrowseExportParquetResponse is returned by POST /api/browse/export-parquet.
type BrowseExportParquetResponse struct {
	Shard               string `json:"shard"`
	IncludeMarkdownBody bool   `json:"include_markdown_body"`
	OutputPath          string `json:"output_path"`
	Rows                int64  `json:"rows"`
	SizeBytes           int64  `json:"size_bytes"`
	ElapsedMs           int64  `json:"elapsed_ms"`
}
