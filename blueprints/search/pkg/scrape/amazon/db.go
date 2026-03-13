package amazon

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

type DB struct {
	db   *sql.DB
	path string
}

func OpenDB(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, err
	}
	d := &DB{db: db, path: path}
	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) initSchema() error {
	_, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS amazon_products (
		query         VARCHAR,
		asin          VARCHAR,
		title         VARCHAR,
		url           VARCHAR,
		image_url     VARCHAR,
		price_text    VARCHAR,
		price_value   DOUBLE,
		currency      VARCHAR,
		rating        DOUBLE,
		review_count  INTEGER,
		is_prime      BOOLEAN,
		is_sponsored  BOOLEAN,
		badge         VARCHAR,
		position      INTEGER,
		result_page   INTEGER,
		raw_container VARCHAR,
		scraped_at    TIMESTAMP,
		PRIMARY KEY (query, asin, result_page)
	)`)
	return err
}

func (d *DB) InsertProducts(products []Product) error {
	if len(products) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO amazon_products (
		query, asin, title, url, image_url, price_text, price_value, currency,
		rating, review_count, is_prime, is_sponsored, badge, position, result_page,
		raw_container, scraped_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, p := range products {
		if p.ScrapedAt.IsZero() {
			p.ScrapedAt = now
		}
		if _, err := stmt.Exec(
			p.Query, p.ASIN, p.Title, p.URL, p.ImageURL, p.PriceText, p.PriceValue, p.Currency,
			p.Rating, p.ReviewCount, p.IsPrime, p.IsSponsored, p.Badge, p.Position, p.ResultPage,
			p.RawContainer, p.ScrapedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) LastPageForQuery(query string) (int, error) {
	var page sql.NullInt64
	if err := d.db.QueryRow(`SELECT MAX(result_page) FROM amazon_products WHERE query = ?`, query).Scan(&page); err != nil {
		return 0, err
	}
	if !page.Valid {
		return 0, nil
	}
	return int(page.Int64), nil
}

func (d *DB) Stats() (CrawlStats, error) {
	var s CrawlStats
	err := d.db.QueryRow(`
		SELECT
			COALESCE(MAX(query), ''),
			COALESCE(MAX(result_page), 0),
			COUNT(*),
			COUNT(DISTINCT asin)
		FROM amazon_products`).Scan(&s.Query, &s.Pages, &s.Products, &s.UniqueASIN)
	return s, err
}

func (d *DB) Path() string { return d.path }

func (d *DB) Close() error {
	if d.db == nil {
		return nil
	}
	if err := d.db.Close(); err != nil {
		return fmt.Errorf("close db: %w", err)
	}
	return nil
}
