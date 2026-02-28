package crawl

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu/blueprints/search/pkg/archived/recrawler"
)

// SeedCursor pages through a DuckDB seed table without loading all rows into memory.
// Each Next() call returns up to pageSize rows. Returns empty slice at EOF.
type SeedCursor struct {
	db       *sql.DB
	pageSize int
	offset   int
}

// NewSeedCursor opens a read-only cursor over the docs table in dbPath.
// pageSize controls how many rows are returned per Next() call (default 10000 if ≤0).
func NewSeedCursor(dbPath string, pageSize int) (*SeedCursor, error) {
	if pageSize <= 0 {
		pageSize = 10_000
	}
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("seedcursor: open %s: %w", dbPath, err)
	}
	return &SeedCursor{db: db, pageSize: pageSize}, nil
}

// Next returns the next page of seed URLs. Returns an empty slice at EOF.
func (c *SeedCursor) Next(ctx context.Context) ([]recrawler.SeedURL, error) {
	rows, err := c.db.QueryContext(ctx,
		"SELECT url, COALESCE(domain, '') FROM docs ORDER BY domain LIMIT ? OFFSET ?",
		c.pageSize, c.offset)
	if err != nil {
		return nil, fmt.Errorf("seedcursor: query: %w", err)
	}
	defer rows.Close()

	var page []recrawler.SeedURL
	for rows.Next() {
		var s recrawler.SeedURL
		if err := rows.Scan(&s.URL, &s.Domain); err != nil {
			return nil, fmt.Errorf("seedcursor: scan: %w", err)
		}
		page = append(page, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("seedcursor: rows: %w", err)
	}
	c.offset += len(page)
	return page, nil
}

// Close closes the underlying database connection.
func (c *SeedCursor) Close() error {
	return c.db.Close()
}
