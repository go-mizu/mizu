package web

import "context"

// DocStoreIface is the interface satisfied by *DocStore.
type DocStoreIface interface {
	ScanShard(ctx context.Context, crawlID, shard, warcMdPath string) (int64, error)
	ScanAll(ctx context.Context, crawlID, crawlBase string) (int64, error)
	ListShardMetas(ctx context.Context, crawlID string) ([]DocShardMeta, error)
	GetShardMeta(ctx context.Context, crawlID, shard string) (DocShardMeta, bool, error)
	ListDocs(ctx context.Context, crawlID, shard string, page, pageSize int, q, sortBy string) ([]DocRecord, int64, error)
	GetDoc(ctx context.Context, crawlID, shard, docID string) (DocRecord, bool, error)
	ShardStats(ctx context.Context, crawlID, shard string) (ShardStatsResponse, error)
	IsScanning(crawlID, shard string) bool
	Close() error
}

// DomainStoreIface is the interface satisfied by *DomainStore.
type DomainStoreIface interface {
	EnsureFresh(ctx context.Context) error
	IsSyncing() bool
	ListDomains(ctx context.Context, sortBy, q string, page, pageSize int) (DomainsResponse, error)
	ListDomainURLs(ctx context.Context, domain, sortBy, statusGroup string, page, pageSize int) (DomainDetailResponse, error)
	GetOverviewStats(ctx context.Context) (*DomainsOverview, bool, bool)
	Close() error
}

// CCDomainStoreIface is the interface satisfied by *CCDomainStore.
type CCDomainStoreIface interface {
	FetchAndCache(ctx context.Context, domain, crawlID string, maxURLs int) (CCDomainFetchResponse, error)
	GetDomainURLs(ctx context.Context, domain, crawlID, sortBy, statusGroup, q string, page, pageSize int) (CCDomainDetailResponse, error)
	Close() error
}

// Compile-time checks.
var (
	_ DocStoreIface      = (*DocStore)(nil)
	_ DomainStoreIface   = (*DomainStore)(nil)
	_ CCDomainStoreIface = (*CCDomainStore)(nil)
)
