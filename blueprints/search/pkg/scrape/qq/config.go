// Package qq provides a crawler for news.qq.com (Tencent News).
// It discovers articles via sitemaps and channel feed APIs,
// then fetches full content by parsing server-rendered window.DATA.
package qq

import (
	"os"
	"path/filepath"
	"time"
)

const (
	// Sitemap endpoints
	SitemapIndexURL = "https://news.qq.com/sitemap/index.xml"

	// Feed API endpoints
	FeedAPIURL    = "https://r.inews.qq.com/web_feed/getPCList"
	HotRankingURL = "https://r.inews.qq.com/gw/event/hot_ranking_list"

	// Article page base URL
	ArticleBaseURL = "https://news.qq.com/rain/a/"

	// Default User-Agent
	DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

	// Bloom filter settings
	BloomCapacity = 10_000_000
	BloomFPR      = 0.001
)

// ChannelIDs is the list of all known news.qq.com channel IDs.
var ChannelIDs = []string{
	"news_news_top",
	"news_news_tech",
	"news_news_finance",
	"news_news_ent",
	"news_news_sports",
	"news_news_world",
	"news_news_mil",
	"news_news_auto",
	"news_news_game",
	"news_news_kepu",
	"news_news_antip",
	"news_news_history",
	"news_news_edu",
	"news_news_nba",
	"news_news_football",
	"news_news_house",
	"news_news_digi",
	"news_news_esport",
	"news_news_baby",
	"news_news_lic",
	"news_news_istock",
	"news_news_video",
	"news_news_nchupin",
}

// Config holds configuration for the QQ News crawler.
type Config struct {
	DataDir    string
	Workers    int
	Timeout    time.Duration
	UserAgent  string
	Resume     bool
	Channels   bool    // also crawl channel feed APIs
	Probe      bool    // enumerate ALL possible sitemaps beyond index.xml
	RateLimit  float64 // requests per second (0 = unlimited)
	MaxRetry   int     // max retries for transient errors (567)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		DataDir:   defaultDataDir(),
		Workers:   20,
		Timeout:   15 * time.Second,
		UserAgent: DefaultUserAgent,
		RateLimit: 10, // 10 req/s to avoid anti-bot
		MaxRetry:  2,
	}
}

func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "qq-news")
}

// DBPath returns the DuckDB database path.
func (c Config) DBPath() string {
	return filepath.Join(c.DataDir, "qq.duckdb")
}
