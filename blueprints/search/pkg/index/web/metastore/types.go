package metastore

import "time"

// WARCRecord is a per-WARC metadata snapshot persisted for dashboard pages.
type WARCRecord struct {
	CrawlID       string
	WARCIndex     string
	ManifestIndex int64
	Filename      string
	RemotePath    string
	WARCBytes     int64
	MarkdownDocs  int64
	MarkdownBytes int64
	PackBytes     map[string]int64
	FTSBytes      map[string]int64
	TotalBytes    int64
	UpdatedAt     time.Time
}

// SummaryRecord is the persisted metadata snapshot for one crawl.
type SummaryRecord struct {
	CrawlID       string
	WARCCount     int64
	WARCTotalSize int64
	MDShards      int64
	MDTotalSize   int64
	MDDocEstimate int64
	PackFormats   map[string]int64
	FTSEngines    map[string]int64
	FTSShardCount map[string]int64
	WARCs         []WARCRecord
	GeneratedAt   time.Time
	ScanDuration  time.Duration
}

// RefreshState tracks metadata refresh lifecycle for one crawl.
type RefreshState struct {
	CrawlID    string
	Status     string // idle, refreshing, error
	StartedAt  *time.Time
	FinishedAt *time.Time
	LastError  string
	Generation int64
}
