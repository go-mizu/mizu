package types

import "time"

// RSSFeed represents a single RSS feed.
type RSSFeed struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	SiteURL     string    `json:"site_url"`
	Description string    `json:"description"`
	LastCrawledAt *time.Time `json:"last_crawled_at"`
}

// RSSItem represents a single item in an RSS feed.
type RSSItem struct {
	ID          int64     `json:"id"`
	FeedID      int64     `json:"feed_id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Content     string    `json:"content"`
	PublishedAt *time.Time `json:"published_at"`
}
