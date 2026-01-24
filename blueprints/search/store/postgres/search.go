package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// SearchStore implements store.SearchStore using PostgreSQL.
type SearchStore struct {
	db *sql.DB
}

// Search performs a full-text search.
func (s *SearchStore) Search(ctx context.Context, query string, opts store.SearchOptions) (*store.SearchResponse, error) {
	start := time.Now()

	// Set defaults
	if opts.PerPage <= 0 {
		opts.PerPage = 10
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	offset := (opts.Page - 1) * opts.PerPage

	// Build the search query
	var conditions []string
	var args []interface{}
	argIdx := 1

	// Main full-text search condition
	searchQuery := normalizeQuery(query)
	if searchQuery != "" {
		conditions = append(conditions, fmt.Sprintf("search_vector @@ plainto_tsquery('english', $%d)", argIdx))
		args = append(args, searchQuery)
		argIdx++
	}

	// Site filter
	if opts.Site != "" {
		conditions = append(conditions, fmt.Sprintf("domain = $%d", argIdx))
		args = append(args, opts.Site)
		argIdx++
	}

	// Exclude site
	if opts.ExcludeSite != "" {
		conditions = append(conditions, fmt.Sprintf("domain != $%d", argIdx))
		args = append(args, opts.ExcludeSite)
		argIdx++
	}

	// Time range filter
	if opts.TimeRange != "" {
		var interval string
		switch opts.TimeRange {
		case "day":
			interval = "1 day"
		case "week":
			interval = "7 days"
		case "month":
			interval = "30 days"
		case "year":
			interval = "365 days"
		}
		if interval != "" {
			conditions = append(conditions, fmt.Sprintf("crawled_at > NOW() - INTERVAL '%s'", interval))
		}
	}

	// Language filter
	if opts.Language != "" {
		conditions = append(conditions, fmt.Sprintf("language = $%d", argIdx))
		args = append(args, opts.Language)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total results
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM search.documents %s", whereClause)
	var totalResults int64
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalResults); err != nil {
		return nil, fmt.Errorf("failed to count results: %w", err)
	}

	// Get results with ranking
	rankExpr := "1.0"
	if searchQuery != "" {
		rankExpr = fmt.Sprintf("ts_rank_cd(search_vector, plainto_tsquery('english', $1))")
	}

	selectQuery := fmt.Sprintf(`
		SELECT
			id, url, title, description, domain, favicon, crawled_at,
			%s as score,
			ts_headline('english', content, plainto_tsquery('english', $1),
				'StartSel=<mark>, StopSel=</mark>, MaxWords=35, MinWords=15') as snippet
		FROM search.documents
		%s
		ORDER BY score DESC, crawled_at DESC
		LIMIT $%d OFFSET $%d
	`, rankExpr, whereClause, argIdx, argIdx+1)

	args = append(args, opts.PerPage, offset)

	rows, err := s.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer rows.Close()

	var results []store.SearchResult
	for rows.Next() {
		var r store.SearchResult
		var favicon, snippet sql.NullString
		if err := rows.Scan(&r.ID, &r.URL, &r.Title, &snippet, &r.Domain, &favicon, &r.CrawledAt, &r.Score, &snippet); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		if favicon.Valid {
			r.Favicon = favicon.String
		}
		if snippet.Valid {
			r.Snippet = snippet.String
		}
		results = append(results, r)
	}

	searchTime := float64(time.Since(start).Microseconds()) / 1000.0

	return &store.SearchResponse{
		Query:        query,
		TotalResults: totalResults,
		Results:      results,
		SearchTimeMs: searchTime,
		Page:         opts.Page,
		PerPage:      opts.PerPage,
	}, nil
}

// SearchImages searches for images.
func (s *SearchStore) SearchImages(ctx context.Context, query string, opts store.SearchOptions) ([]store.ImageResult, error) {
	if opts.PerPage <= 0 {
		opts.PerPage = 20
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	offset := (opts.Page - 1) * opts.PerPage

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, url, thumbnail_url, title, source_url, source_domain, width, height, file_size, format
		FROM search.images
		WHERE title ILIKE $1 OR alt_text ILIKE $1
		ORDER BY crawled_at DESC
		LIMIT $2 OFFSET $3
	`, "%"+query+"%", opts.PerPage, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search images: %w", err)
	}
	defer rows.Close()

	var results []store.ImageResult
	for rows.Next() {
		var r store.ImageResult
		var thumbnailURL, title, format sql.NullString
		var width, height sql.NullInt32
		var fileSize sql.NullInt64
		if err := rows.Scan(&r.ID, &r.URL, &thumbnailURL, &title, &r.SourceURL, &r.SourceDomain, &width, &height, &fileSize, &format); err != nil {
			return nil, fmt.Errorf("failed to scan image: %w", err)
		}
		if thumbnailURL.Valid {
			r.ThumbnailURL = thumbnailURL.String
		}
		if title.Valid {
			r.Title = title.String
		}
		if format.Valid {
			r.Format = format.String
		}
		if width.Valid {
			r.Width = int(width.Int32)
		}
		if height.Valid {
			r.Height = int(height.Int32)
		}
		if fileSize.Valid {
			r.FileSize = fileSize.Int64
		}
		results = append(results, r)
	}

	return results, nil
}

// SearchVideos searches for videos.
func (s *SearchStore) SearchVideos(ctx context.Context, query string, opts store.SearchOptions) ([]store.VideoResult, error) {
	if opts.PerPage <= 0 {
		opts.PerPage = 20
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	offset := (opts.Page - 1) * opts.PerPage

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, url, thumbnail_url, title, description, duration_seconds, channel, views, published_at
		FROM search.videos
		WHERE title ILIKE $1 OR description ILIKE $1
		ORDER BY views DESC, published_at DESC
		LIMIT $2 OFFSET $3
	`, "%"+query+"%", opts.PerPage, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search videos: %w", err)
	}
	defer rows.Close()

	var results []store.VideoResult
	for rows.Next() {
		var r store.VideoResult
		var thumbnailURL, description, channel sql.NullString
		var duration sql.NullInt32
		var views sql.NullInt64
		var publishedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.URL, &thumbnailURL, &r.Title, &description, &duration, &channel, &views, &publishedAt); err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}
		if thumbnailURL.Valid {
			r.ThumbnailURL = thumbnailURL.String
		}
		if description.Valid {
			r.Description = description.String
		}
		if channel.Valid {
			r.Channel = channel.String
		}
		if duration.Valid {
			r.Duration = int(duration.Int32)
		}
		if views.Valid {
			r.Views = views.Int64
		}
		if publishedAt.Valid {
			r.PublishedAt = publishedAt.Time
		}
		results = append(results, r)
	}

	return results, nil
}

// SearchNews searches for news articles.
func (s *SearchStore) SearchNews(ctx context.Context, query string, opts store.SearchOptions) ([]store.NewsResult, error) {
	if opts.PerPage <= 0 {
		opts.PerPage = 20
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	offset := (opts.Page - 1) * opts.PerPage

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, url, title, snippet, source, image_url, published_at
		FROM search.news
		WHERE title ILIKE $1 OR snippet ILIKE $1
		ORDER BY published_at DESC
		LIMIT $2 OFFSET $3
	`, "%"+query+"%", opts.PerPage, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search news: %w", err)
	}
	defer rows.Close()

	var results []store.NewsResult
	for rows.Next() {
		var r store.NewsResult
		var snippet, imageURL sql.NullString
		if err := rows.Scan(&r.ID, &r.URL, &r.Title, &snippet, &r.Source, &imageURL, &r.PublishedAt); err != nil {
			return nil, fmt.Errorf("failed to scan news: %w", err)
		}
		if snippet.Valid {
			r.Snippet = snippet.String
		}
		if imageURL.Valid {
			r.ImageURL = imageURL.String
		}
		results = append(results, r)
	}

	return results, nil
}

// normalizeQuery cleans up a search query.
func normalizeQuery(q string) string {
	// Remove extra whitespace
	q = strings.TrimSpace(q)
	q = strings.Join(strings.Fields(q), " ")
	return q
}
