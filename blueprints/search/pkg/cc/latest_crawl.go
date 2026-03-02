package cc

import (
	"context"
	"fmt"
	"time"
)

// LatestCrawlResult describes how the latest crawl ID was resolved.
type LatestCrawlResult struct {
	ID        string
	FromCache bool
	Stale     bool
}

// ResolveLatestCrawlID returns the latest Common Crawl dataset ID, using cache.json first
// and refreshing from collinfo.json when needed. The resolved latest crawl ID is persisted
// back into cache.json.
func ResolveLatestCrawlID(ctx context.Context, cfg Config) (LatestCrawlResult, error) {
	cache := NewCache(cfg.DataDir)
	if cd := cache.Load(); cd != nil {
		if id := latestCrawlIDFromCacheData(cd); id != "" && cache.IsFresh(cd) {
			return LatestCrawlResult{ID: id, FromCache: true}, nil
		}
	}

	client := NewClient("", 4)
	crawls, err := client.ListCrawls(ctx)
	if err == nil {
		id := latestCrawlID(crawls)
		if id == "" {
			return LatestCrawlResult{}, fmt.Errorf("no crawls returned from Common Crawl API")
		}
		cd := cache.Load()
		if cd == nil {
			cd = &CacheData{}
		}
		cd.Crawls = crawls
		cd.LatestCrawlID = id
		cd.FetchedAt = time.Now()
		_ = cache.Save(cd)
		return LatestCrawlResult{ID: id}, nil
	}

	// Fallback to stale cache if API is unavailable.
	if cd := cache.Load(); cd != nil {
		if id := latestCrawlIDFromCacheData(cd); id != "" {
			return LatestCrawlResult{ID: id, FromCache: true, Stale: true}, nil
		}
	}
	return LatestCrawlResult{}, fmt.Errorf("resolving latest crawl: %w", err)
}

func latestCrawlIDFromCacheData(cd *CacheData) string {
	if cd == nil {
		return ""
	}
	if cd.LatestCrawlID != "" {
		return cd.LatestCrawlID
	}
	return latestCrawlID(cd.Crawls)
}

func latestCrawlID(crawls []Crawl) string {
	var best Crawl
	have := false
	for _, c := range crawls {
		if c.ID == "" {
			continue
		}
		if !have {
			best = c
			have = true
			continue
		}
		if c.To.After(best.To) ||
			(c.To.Equal(best.To) && c.From.After(best.From)) ||
			(c.To.Equal(best.To) && c.From.Equal(best.From) && c.ID > best.ID) {
			best = c
		}
	}
	if !have {
		return ""
	}
	return best.ID
}
