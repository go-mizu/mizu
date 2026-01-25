package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// SmallWebStore handles small web index for enrichment.
type SmallWebStore struct {
	db *sql.DB
}

// IndexEntry indexes a small web entry.
func (s *SmallWebStore) IndexEntry(ctx context.Context, entry *store.SmallWebEntry) error {
	entry.IndexedAt = time.Now()

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO small_web (url, title, snippet, source_type, domain, published_at, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			title = excluded.title,
			snippet = excluded.snippet,
			indexed_at = excluded.indexed_at
	`, entry.URL, entry.Title, entry.Snippet, entry.SourceType, entry.Domain, entry.PublishedAt, entry.IndexedAt)

	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	entry.ID = id
	return nil
}

// SearchWeb searches the small web index (Teclis-style).
func (s *SmallWebStore) SearchWeb(ctx context.Context, query string, limit int) ([]*store.EnrichmentResult, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT sw.url, sw.title, sw.snippet, sw.published_at
		FROM small_web sw
		JOIN small_web_fts fts ON sw.rowid = fts.rowid
		WHERE small_web_fts MATCH ?
		AND sw.source_type IN ('blog', 'forum', 'discussion')
		ORDER BY rank
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*store.EnrichmentResult
	rank := 1
	for rows.Next() {
		var result store.EnrichmentResult
		var published sql.NullTime

		if err := rows.Scan(&result.URL, &result.Title, &result.Snippet, &published); err != nil {
			return nil, err
		}

		result.Type = types.EnrichTypeResult
		result.Rank = rank
		if published.Valid {
			result.Published = &published.Time
		}

		results = append(results, &result)
		rank++
	}

	return results, nil
}

// SearchNews searches for non-mainstream news (TinyGem-style).
func (s *SmallWebStore) SearchNews(ctx context.Context, query string, limit int) ([]*store.EnrichmentResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// First try the small_web table for news-type content
	rows, err := s.db.QueryContext(ctx, `
		SELECT sw.url, sw.title, sw.snippet, sw.published_at
		FROM small_web sw
		JOIN small_web_fts fts ON sw.rowid = fts.rowid
		WHERE small_web_fts MATCH ?
		AND sw.source_type = 'news'
		ORDER BY sw.published_at DESC
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*store.EnrichmentResult
	rank := 1
	for rows.Next() {
		var result store.EnrichmentResult
		var published sql.NullTime

		if err := rows.Scan(&result.URL, &result.Title, &result.Snippet, &published); err != nil {
			return nil, err
		}

		result.Type = types.EnrichTypeResult
		result.Rank = rank
		if published.Valid {
			result.Published = &published.Time
		}

		results = append(results, &result)
		rank++
	}

	return results, nil
}

// SeedSmallWeb inserts sample small web entries.
func (s *SmallWebStore) SeedSmallWeb(ctx context.Context) error {
	entries := getDefaultSmallWebEntries()
	for _, entry := range entries {
		if err := s.IndexEntry(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

// getDefaultSmallWebEntries returns sample small web content.
func getDefaultSmallWebEntries() []*store.SmallWebEntry {
	now := time.Now()
	return []*store.SmallWebEntry{
		{
			URL:         "https://blog.golang.org/go1.22",
			Title:       "Go 1.22 Release Notes",
			Snippet:     "Go 1.22 includes improvements to the language, tooling, and standard library.",
			SourceType:  "blog",
			Domain:      "blog.golang.org",
			PublishedAt: now.AddDate(0, -3, 0),
		},
		{
			URL:         "https://jvns.ca/blog/2024/02/16/popular-git-config-options/",
			Title:       "Popular git config options",
			Snippet:     "A comprehensive look at git configuration options that developers find most useful.",
			SourceType:  "blog",
			Domain:      "jvns.ca",
			PublishedAt: now.AddDate(0, -2, 0),
		},
		{
			URL:         "https://www.tedinski.com/2018/01/30/the-maybe-monad.html",
			Title:       "The Maybe Monad in Various Languages",
			Snippet:     "Exploring Option/Maybe types across different programming languages.",
			SourceType:  "blog",
			Domain:      "tedinski.com",
			PublishedAt: now.AddDate(-1, 0, 0),
		},
		{
			URL:         "https://lobste.rs/s/abc123/rust_vs_go_which_one_choose",
			Title:       "Rust vs Go: Which One to Choose?",
			Snippet:     "A community discussion comparing Rust and Go for different use cases.",
			SourceType:  "discussion",
			Domain:      "lobste.rs",
			PublishedAt: now.AddDate(0, -1, 0),
		},
		{
			URL:         "https://news.ycombinator.com/item?id=12345678",
			Title:       "Show HN: A minimal HTTP framework in Go",
			Snippet:     "I built a lightweight HTTP framework that focuses on simplicity and performance.",
			SourceType:  "discussion",
			Domain:      "news.ycombinator.com",
			PublishedAt: now.AddDate(0, 0, -7),
		},
		{
			URL:         "https://eli.thegreenplace.net/2023/common-pitfalls-in-go/",
			Title:       "Common Pitfalls in Go",
			Snippet:     "A guide to avoiding common mistakes when writing Go code.",
			SourceType:  "blog",
			Domain:      "eli.thegreenplace.net",
			PublishedAt: now.AddDate(0, -6, 0),
		},
		{
			URL:         "https://fasterthanli.me/articles/whats-in-the-box",
			Title:       "What's in the Box?",
			Snippet:     "A deep dive into Rust's ownership and borrowing system.",
			SourceType:  "blog",
			Domain:      "fasterthanli.me",
			PublishedAt: now.AddDate(0, -4, 0),
		},
		{
			URL:         "https://smallcultfollowing.com/babysteps/blog/2024/01/15/async-vision/",
			Title:       "Async Vision for Rust",
			Snippet:     "Exploring the future of async programming in Rust.",
			SourceType:  "blog",
			Domain:      "smallcultfollowing.com",
			PublishedAt: now.AddDate(0, -5, 0),
		},
		{
			URL:         "https://drewdevault.com/2024/01/10/2024-01-10-Open-source-sustainability.html",
			Title:       "On Open Source Sustainability",
			Snippet:     "Thoughts on making open source projects sustainable long-term.",
			SourceType:  "blog",
			Domain:      "drewdevault.com",
			PublishedAt: now.AddDate(0, -1, -10),
		},
		{
			URL:         "https://matklad.github.io/2023/10/12/rust-analyzer-lessons.html",
			Title:       "Lessons from rust-analyzer",
			Snippet:     "Key lessons learned from building the rust-analyzer project.",
			SourceType:  "blog",
			Domain:      "matklad.github.io",
			PublishedAt: now.AddDate(0, -8, 0),
		},
	}
}
