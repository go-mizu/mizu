package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
)

func ccResolveCrawlID(ctx context.Context, crawlID string) (resolved string, note string, err error) {
	if strings.TrimSpace(crawlID) != "" {
		return crawlID, "", nil
	}
	cfg := cc.DefaultConfig()
	res, err := cc.ResolveLatestCrawlID(ctx, cfg)
	if err != nil {
		return "", "", err
	}
	note = "latest from API"
	if res.FromCache && !res.Stale {
		note = "latest from cache"
	} else if res.FromCache && res.Stale {
		note = "latest from stale cache (API unavailable)"
	}
	return res.ID, note, nil
}

func ccNormalizeParquetSubset(subset string) string {
	s := strings.TrimSpace(strings.ToLower(subset))
	switch s {
	case "", "warc":
		return "warc"
	case "all", "*":
		return ""
	default:
		return s
	}
}

func ccPrintDefaultCrawlResolution(crawlID, note string) {
	if note == "" {
		return
	}
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Using crawl: %s (%s)", crawlID, note)))
}
