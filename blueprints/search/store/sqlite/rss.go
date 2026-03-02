package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/types"
)

// RSSStore handles RSS feed and item storage.
type RSSStore struct {
	db *sql.DB
}

// NewRSSStore creates a new RSSStore.
func NewRSSStore(db *sql.DB) *RSSStore {
	return &RSSStore{db: db}
}

// AddFeed adds a new RSS feed to the database.
func (s *RSSStore) AddFeed(ctx context.Context, feed *types.RSSFeed) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO rss_feeds (title, url, site_url, description)
		VALUES (?, ?, ?, ?)
	`, feed.Title, feed.URL, feed.SiteURL, feed.Description)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetFeed retrieves an RSS feed by its ID.
func (s *RSSStore) GetFeed(ctx context.Context, id int64) (*types.RSSFeed, error) {
	feed := &types.RSSFeed{}
	var lastCrawled sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, title, url, site_url, description, last_crawled_at
		FROM rss_feeds
		WHERE id = ?
	`, id).Scan(&feed.ID, &feed.Title, &feed.URL, &feed.SiteURL, &feed.Description, &lastCrawled)
	if err != nil {
		return nil, err
	}
	if lastCrawled.Valid {
		feed.LastCrawledAt = &lastCrawled.Time
	}
	return feed, nil
}
// GetFeedByurl retrieves an RSS feed by its ID.
func (s *RSSStore) GetFeedByurl(ctx context.Context, url string) (*types.RSSFeed, error) {
	feed := &types.RSSFeed{}
	var lastCrawled sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, title, url, site_url, description, last_crawled_at
		FROM rss_feeds
		WHERE url = ?
	`, url).Scan(&feed.ID, &feed.Title, &feed.URL, &feed.SiteURL, &feed.Description, &lastCrawled)
	if err != nil {
		return nil, err
	}
	if lastCrawled.Valid {
		feed.LastCrawledAt = &lastCrawled.Time
	}
	return feed, nil
}

// ListFeeds retrieves all RSS feeds from the database.
func (s *RSSStore) ListFeeds(ctx context.Context) ([]*types.RSSFeed, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, url, site_url, description, last_crawled_at
		FROM rss_feeds
		ORDER BY title
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []*types.RSSFeed
	for rows.Next() {
		feed := &types.RSSFeed{}
		var lastCrawled sql.NullTime
		if err := rows.Scan(&feed.ID, &feed.Title, &feed.URL, &feed.SiteURL, &feed.Description, &lastCrawled); err != nil {
			return nil, err
		}
		if lastCrawled.Valid {
			feed.LastCrawledAt = &lastCrawled.Time
		}
		feeds = append(feeds, feed)
	}
	return feeds, nil
}

// AddItem adds a new RSS item to the database.
func (s *RSSStore) AddItem(ctx context.Context, item *types.RSSItem) (int64, error) {
	publishedAt := time.Now()
	if item.PublishedAt != nil {
		publishedAt = *item.PublishedAt
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO rss_items (feed_id, title, url, content, published_at)
		VALUES (?, ?, ?, ?, ?)
	`, item.FeedID, item.Title, item.URL, item.Content, publishedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
// GetItemByUrl retrieves an RSS item by its ID.
func (s *RSSStore) GetItemByUrl(ctx context.Context, url string) (*types.RSSItem, error) {
	item := &types.RSSItem{}
	var publishedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, feed_id, title, url, content, published_at
		FROM rss_items
		WHERE url = ?
	`, url).Scan(&item.ID, &item.FeedID, &item.Title, &item.URL, &item.Content, &publishedAt)
	if err != nil {
		return nil, err
	}
	if publishedAt.Valid {
		item.PublishedAt = &publishedAt.Time
	}
	return item, nil
}
// ListItems retrieves all RSS items for a given feed.
func (s *RSSStore) ListItems(ctx context.Context, feedID int64) ([]*types.RSSItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, feed_id, title, url, content, published_at
		FROM rss_items
		WHERE feed_id = ?
		ORDER BY published_at DESC
	`, feedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*types.RSSItem
	for rows.Next() {
		item := &types.RSSItem{}
		var publishedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.FeedID, &item.Title, &item.URL, &item.Content, &publishedAt); err != nil {
			return nil, err
		}
		if publishedAt.Valid {
			item.PublishedAt = &publishedAt.Time
		}
		items = append(items, item)
	}
	return items, nil
}
