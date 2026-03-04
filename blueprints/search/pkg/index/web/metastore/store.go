package metastore

import "context"

// Store is the metadata persistence API used by the dashboard meta manager.
type Store interface {
	Name() string
	Init(ctx context.Context) error
	GetSummary(ctx context.Context, crawlID string) (SummaryRecord, bool, error)
	PutSummary(ctx context.Context, rec SummaryRecord) error
	ListWARCs(ctx context.Context, crawlID string) ([]WARCRecord, error)
	GetWARC(ctx context.Context, crawlID, warcIndex string) (WARCRecord, bool, error)
	GetRefreshState(ctx context.Context, crawlID string) (RefreshState, bool, error)
	SetRefreshState(ctx context.Context, st RefreshState) error
	Close() error
}
