package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// SearchStore handles search operations.
type SearchStore struct {
	db *sql.DB
}

// Search performs full-text search.
func (s *SearchStore) Search(ctx context.Context, query string, opts store.SearchOptions) (*store.SearchResponse, error) {
	start := time.Now()

	// Build FTS5 query
	ftsQuery := buildFTSQuery(query, opts.Verbatim)

	// Build WHERE clause
	whereClause := "WHERE documents_fts MATCH ?"
	args := []any{ftsQuery}

	if opts.Site != "" {
		whereClause += " AND d.domain = ?"
		args = append(args, opts.Site)
	}

	if opts.ExcludeSite != "" {
		whereClause += " AND d.domain != ?"
		args = append(args, opts.ExcludeSite)
	}

	if opts.TimeRange != "" {
		whereClause += " AND d.crawled_at >= ?"
		args = append(args, timeRangeToDate(opts.TimeRange))
	}

	if opts.Language != "" {
		whereClause += " AND d.language = ?"
		args = append(args, opts.Language)
	}

	// Count total results
	var total int64
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*) FROM documents d
		JOIN documents_fts fts ON d.rowid = fts.rowid
		%s
	`, whereClause)

	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count query failed: %w", err)
	}

	// Apply pagination defaults
	if opts.PerPage <= 0 {
		opts.PerPage = 10
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	offset := (opts.Page - 1) * opts.PerPage

	// Get paginated results with BM25 ranking
	searchSQL := fmt.Sprintf(`
		SELECT
			d.id, d.url, d.title, d.description, d.domain, d.favicon, d.crawled_at,
			bm25(documents_fts) as score
		FROM documents d
		JOIN documents_fts fts ON d.rowid = fts.rowid
		%s
		ORDER BY score
		LIMIT ? OFFSET ?
	`, whereClause)

	searchArgs := append(args, opts.PerPage, offset)

	rows, err := s.db.QueryContext(ctx, searchSQL, searchArgs...)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var results []store.SearchResult
	for rows.Next() {
		var r store.SearchResult
		var favicon sql.NullString
		var desc sql.NullString
		if err := rows.Scan(
			&r.ID, &r.URL, &r.Title, &desc, &r.Domain,
			&favicon, &r.CrawledAt, &r.Score,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		if favicon.Valid {
			r.Favicon = favicon.String
		}
		if desc.Valid {
			r.Snippet = desc.String
		}
		results = append(results, r)
	}

	return &store.SearchResponse{
		Query:        query,
		TotalResults: total,
		Results:      results,
		SearchTimeMs: float64(time.Since(start).Milliseconds()),
		Page:         opts.Page,
		PerPage:      opts.PerPage,
	}, nil
}

// SearchImages searches for images.
func (s *SearchStore) SearchImages(ctx context.Context, query string, opts store.SearchOptions) ([]store.ImageResult, error) {
	ftsQuery := buildFTSQuery(query, opts.Verbatim)

	if opts.PerPage <= 0 {
		opts.PerPage = 20
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	offset := (opts.Page - 1) * opts.PerPage

	imageSQL := `
		SELECT i.id, i.url, i.thumbnail_url, i.title, i.source_url,
			   i.source_domain, i.width, i.height, i.file_size, i.format
		FROM images i
		JOIN images_fts fts ON i.rowid = fts.rowid
		WHERE images_fts MATCH ?
		ORDER BY bm25(images_fts)
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, imageSQL, ftsQuery, opts.PerPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []store.ImageResult
	for rows.Next() {
		var r store.ImageResult
		var thumbnailURL, title, format sql.NullString
		var width, height sql.NullInt64
		var fileSize sql.NullInt64
		if err := rows.Scan(
			&r.ID, &r.URL, &thumbnailURL, &title, &r.SourceURL,
			&r.SourceDomain, &width, &height, &fileSize, &format,
		); err != nil {
			return nil, err
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
			r.Width = int(width.Int64)
		}
		if height.Valid {
			r.Height = int(height.Int64)
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
	ftsQuery := buildFTSQuery(query, opts.Verbatim)

	if opts.PerPage <= 0 {
		opts.PerPage = 20
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	offset := (opts.Page - 1) * opts.PerPage

	videoSQL := `
		SELECT v.id, v.url, v.thumbnail_url, v.title, v.description,
			   v.duration_seconds, v.channel, v.views, v.published_at
		FROM videos v
		JOIN videos_fts fts ON v.rowid = fts.rowid
		WHERE videos_fts MATCH ?
		ORDER BY bm25(videos_fts)
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, videoSQL, ftsQuery, opts.PerPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []store.VideoResult
	for rows.Next() {
		var r store.VideoResult
		var thumbnailURL, desc, channel sql.NullString
		var duration sql.NullInt64
		var views sql.NullInt64
		var publishedAt sql.NullTime
		if err := rows.Scan(
			&r.ID, &r.URL, &thumbnailURL, &r.Title, &desc,
			&duration, &channel, &views, &publishedAt,
		); err != nil {
			return nil, err
		}
		if thumbnailURL.Valid {
			r.ThumbnailURL = thumbnailURL.String
		}
		if desc.Valid {
			r.Description = desc.String
		}
		if channel.Valid {
			r.Channel = channel.String
		}
		if duration.Valid {
			r.Duration = int(duration.Int64)
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

// SearchNews searches for news.
func (s *SearchStore) SearchNews(ctx context.Context, query string, opts store.SearchOptions) ([]store.NewsResult, error) {
	ftsQuery := buildFTSQuery(query, opts.Verbatim)

	if opts.PerPage <= 0 {
		opts.PerPage = 20
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	offset := (opts.Page - 1) * opts.PerPage

	newsSQL := `
		SELECT n.id, n.url, n.title, n.snippet, n.source, n.image_url, n.published_at
		FROM news n
		JOIN news_fts fts ON n.rowid = fts.rowid
		WHERE news_fts MATCH ?
		ORDER BY n.published_at DESC, bm25(news_fts)
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, newsSQL, ftsQuery, opts.PerPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []store.NewsResult
	for rows.Next() {
		var r store.NewsResult
		var snippet, imageURL sql.NullString
		if err := rows.Scan(
			&r.ID, &r.URL, &r.Title, &snippet, &r.Source, &imageURL, &r.PublishedAt,
		); err != nil {
			return nil, err
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

// Helper functions

func buildFTSQuery(query string, verbatim bool) string {
	if verbatim {
		// Exact phrase match
		return fmt.Sprintf(`"%s"`, strings.ReplaceAll(query, `"`, `""`))
	}
	// Add prefix matching for each term
	terms := strings.Fields(query)
	for i, term := range terms {
		// Escape special FTS5 characters
		term = strings.ReplaceAll(term, `"`, `""`)
		terms[i] = term + "*"
	}
	return strings.Join(terms, " ")
}

func timeRangeToDate(tr string) time.Time {
	now := time.Now()
	switch tr {
	case "hour":
		return now.Add(-1 * time.Hour)
	case "day":
		return now.Add(-24 * time.Hour)
	case "week":
		return now.Add(-7 * 24 * time.Hour)
	case "month":
		return now.Add(-30 * 24 * time.Hour)
	case "year":
		return now.Add(-365 * 24 * time.Hour)
	default:
		return time.Time{}
	}
}
