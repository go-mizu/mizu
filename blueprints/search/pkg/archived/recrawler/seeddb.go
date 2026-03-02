package recrawler

import (
	"context"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

// LoadSeedURLs is deprecated. Use store.LoadSeedURLs.
func LoadSeedURLs(ctx context.Context, dbPath string, expectedCount int) ([]crawl.SeedURL, error) {
	return store.LoadSeedURLs(ctx, dbPath, expectedCount)
}

// LoadSeedStats is deprecated. Use store.LoadSeedStats.
func LoadSeedStats(ctx context.Context, dbPath string) (*SeedStats, error) {
	s, err := store.LoadSeedStats(ctx, dbPath)
	if err != nil || s == nil {
		return nil, err
	}
	return &SeedStats{
		TotalURLs:     s.TotalURLs,
		UniqueDomains: s.UniqueDomains,
		Protocols:     s.Protocols,
		ContentTypes:  s.ContentTypes,
		TLDs:          s.TLDs,
	}, nil
}
