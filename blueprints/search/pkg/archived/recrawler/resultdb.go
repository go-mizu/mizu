package recrawler

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

// ResultDB is deprecated. Use store.ResultDB directly.
type ResultDB = store.ResultDB

// NewResultDB is deprecated. Use store.NewResultDB.
func NewResultDB(dir string, shardCount, batchSize, duckMemPerShardMB int) (*ResultDB, error) {
	return store.NewResultDB(dir, shardCount, batchSize, duckMemPerShardMB)
}

// LoadAlreadyCrawledFromDir is deprecated. Use store.LoadAlreadyCrawledFromDir.
func LoadAlreadyCrawledFromDir(ctx context.Context, dir string) (map[string]bool, error) {
	return store.LoadAlreadyCrawledFromDir(ctx, dir)
}
