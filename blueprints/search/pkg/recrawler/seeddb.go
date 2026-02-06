package recrawler

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

// LoadSeedURLs reads all URLs from a DuckDB seed database.
// Only reads url and domain columns for maximum loading speed.
// No ORDER BY since interleaveByDomain handles distribution.
func LoadSeedURLs(ctx context.Context, dbPath string, expectedCount int) ([]SeedURL, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("opening seed db: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT url, COALESCE(domain, '') as domain FROM docs
	`)
	if err != nil {
		return nil, fmt.Errorf("querying seed urls: %w", err)
	}
	defer rows.Close()

	seeds := make([]SeedURL, 0, expectedCount)
	for rows.Next() {
		var s SeedURL
		if err := rows.Scan(&s.URL, &s.Domain); err != nil {
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

	// Content type distribution (optional column)
	rows, err = db.QueryContext(ctx,
		"SELECT COALESCE(content_type,'?'), COUNT(*) FROM docs GROUP BY content_type ORDER BY COUNT(*) DESC LIMIT 20")
	if err == nil {
		for rows.Next() {
			var k string
			var v int
			rows.Scan(&k, &v)
			stats.ContentTypes[k] = v
		}
		rows.Close()
	}

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
