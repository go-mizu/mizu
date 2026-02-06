package crawler

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

// SeedURL represents a URL loaded from the seed database.
type SeedURL struct {
	URL         string
	Domain      string
	Host        string
	ContentType string
	Language    string
	TextLen     int64
	WordCount   int64
	TLD         string
	Protocol    string
}

// SeedStats holds aggregate stats about the seed database.
type SeedStats struct {
	TotalURLs     int
	UniqueDomains int
	Protocols     map[string]int // HTTP vs HTTPS
	ContentTypes  map[string]int
	TLDs          map[string]int
}

// LoadSeedURLs reads all URLs from a DuckDB seed database.
// The database must have a `docs` table with at least a `url` column.
func LoadSeedURLs(ctx context.Context, dbPath string) ([]SeedURL, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("opening seed db: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT
			url,
			COALESCE(domain, '') as domain,
			COALESCE(host, '') as host,
			COALESCE(content_type, '') as content_type,
			COALESCE(language, '') as language,
			COALESCE(text_len, 0) as text_len,
			COALESCE(word_count, 0) as word_count,
			COALESCE(tld, '') as tld,
			COALESCE(protocol, '') as protocol
		FROM docs
		ORDER BY domain, url
	`)
	if err != nil {
		return nil, fmt.Errorf("querying seed urls: %w", err)
	}
	defer rows.Close()

	var seeds []SeedURL
	for rows.Next() {
		var s SeedURL
		if err := rows.Scan(&s.URL, &s.Domain, &s.Host, &s.ContentType,
			&s.Language, &s.TextLen, &s.WordCount, &s.TLD, &s.Protocol); err != nil {
			return nil, fmt.Errorf("scanning seed row: %w", err)
		}
		seeds = append(seeds, s)
	}
	return seeds, rows.Err()
}

// LoadSeedStats computes aggregate statistics about the seed database.
func LoadSeedStats(ctx context.Context, dbPath string) (*SeedStats, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("opening seed db: %w", err)
	}
	defer db.Close()

	stats := &SeedStats{
		Protocols:    make(map[string]int),
		ContentTypes: make(map[string]int),
		TLDs:         make(map[string]int),
	}

	// Total + unique domains
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*), COUNT(DISTINCT domain) FROM docs").
		Scan(&stats.TotalURLs, &stats.UniqueDomains)
	if err != nil {
		return nil, fmt.Errorf("counting: %w", err)
	}

	// Protocol distribution
	rows, err := db.QueryContext(ctx,
		"SELECT COALESCE(protocol,'?'), COUNT(*) FROM docs GROUP BY protocol")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var k string
		var v int
		rows.Scan(&k, &v)
		stats.Protocols[k] = v
	}
	rows.Close()

	// Content type distribution
	rows, err = db.QueryContext(ctx,
		"SELECT COALESCE(content_type,'?'), COUNT(*) FROM docs GROUP BY content_type ORDER BY COUNT(*) DESC LIMIT 20")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var k string
		var v int
		rows.Scan(&k, &v)
		stats.ContentTypes[k] = v
	}
	rows.Close()

	// TLD distribution
	rows, err = db.QueryContext(ctx,
		"SELECT COALESCE(tld,'?'), COUNT(*) FROM docs GROUP BY tld ORDER BY COUNT(*) DESC LIMIT 20")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var k string
		var v int
		rows.Scan(&k, &v)
		stats.TLDs[k] = v
	}
	rows.Close()

	return stats, nil
}

// LoadAlreadyCrawled loads URLs that were already crawled from the state DB.
// Returns a set of URLs to skip.
func LoadAlreadyCrawled(ctx context.Context, stateDBPath string) (map[string]bool, error) {
	db, err := sql.Open("duckdb", stateDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		// State DB doesn't exist yet — nothing crawled
		return nil, nil
	}
	defer db.Close()

	// Check if table exists
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'state'").
		Scan(&count)
	if err != nil || count == 0 {
		return nil, nil
	}

	rows, err := db.QueryContext(ctx,
		"SELECT url FROM state WHERE status IN ('done', 'failed')")
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	done := make(map[string]bool)
	for rows.Next() {
		var u string
		rows.Scan(&u)
		done[u] = true
	}
	return done, nil
}
