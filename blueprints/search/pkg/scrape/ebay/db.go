package ebay

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DB wraps a DuckDB database for storing eBay data.
type DB struct {
	db   *sql.DB
	path string
}

// OpenDB opens or creates the eBay DuckDB database.
func OpenDB(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb %s: %w", path, err)
	}

	d := &DB{db: db, path: path}
	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) initSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS items (
			item_id                VARCHAR PRIMARY KEY,
			title                  VARCHAR,
			subtitle               VARCHAR,
			description            VARCHAR,
			price                  DOUBLE,
			currency               VARCHAR,
			original_price         DOUBLE,
			condition              VARCHAR,
			availability           VARCHAR,
			seller_name            VARCHAR,
			seller_url             VARCHAR,
			seller_feedback_score  BIGINT,
			seller_positive_pct    DOUBLE,
			shipping_text          VARCHAR,
			returns_text           VARCHAR,
			location               VARCHAR,
			image_urls             VARCHAR,
			category_path          VARCHAR,
			item_specifics         VARCHAR,
			raw_jsonld             VARCHAR,
			url                    VARCHAR,
			fetched_at             TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS search_results (
			search_id        VARCHAR PRIMARY KEY,
			query            VARCHAR,
			page             INTEGER,
			total_results    VARCHAR,
			result_item_ids  VARCHAR,
			url              VARCHAR,
			next_page_url    VARCHAR,
			fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

// UpsertItem inserts or replaces an eBay item record.
func (d *DB) UpsertItem(item Item) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO items (
		item_id, title, subtitle, description,
		price, currency, original_price, condition, availability,
		seller_name, seller_url, seller_feedback_score, seller_positive_pct,
		shipping_text, returns_text, location,
		image_urls, category_path, item_specifics, raw_jsonld, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		item.ItemID, nullStr(item.Title), nullStr(item.Subtitle), nullStr(item.Description),
		nullFloat(item.Price), nullStr(item.Currency), nullFloat(item.OriginalPrice), nullStr(item.Condition), nullStr(item.Availability),
		nullStr(item.SellerName), nullStr(item.SellerURL), nullInt64(item.SellerFeedbackScore), nullFloat(item.SellerPositivePct),
		nullStr(item.ShippingText), nullStr(item.ReturnsText), nullStr(item.Location),
		encodeJSON(item.ImageURLs), encodeJSON(item.CategoryPath), encodeJSON(item.ItemSpecifics), nullStr(item.RawJSONLD), nullStr(item.URL), item.FetchedAt,
	)
	return err
}

// UpsertSearchResult inserts or replaces a search page snapshot.
func (d *DB) UpsertSearchResult(sr SearchResult) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO search_results (
		search_id, query, page, total_results,
		result_item_ids, url, next_page_url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?)`,
		sr.SearchID, nullStr(sr.Query), sr.Page, nullStr(sr.TotalResults),
		encodeJSON(sr.ResultItemIDs), nullStr(sr.URL), nullStr(sr.NextPageURL), sr.FetchedAt,
	)
	return err
}

// GetStats returns row counts for the eBay database.
func (d *DB) GetStats() (DBStats, error) {
	var s DBStats
	d.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&s.Items)
	d.db.QueryRow("SELECT COUNT(*) FROM search_results").Scan(&s.SearchPages)
	if fi, err := os.Stat(d.path); err == nil {
		s.DBSize = fi.Size()
	}
	return s, nil
}

// RecentItems returns the most recently fetched items.
func (d *DB) RecentItems(limit int) ([]Item, error) {
	rows, err := d.db.Query(`
		SELECT item_id, title, price, currency, seller_name, url
		FROM items ORDER BY fetched_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		var title, currency, sellerName, itemURL sql.NullString
		var price sql.NullFloat64
		if err := rows.Scan(&item.ItemID, &title, &price, &currency, &sellerName, &itemURL); err != nil {
			return items, err
		}
		item.Title = title.String
		item.Currency = currency.String
		item.SellerName = sellerName.String
		item.URL = itemURL.String
		item.Price = price.Float64
		items = append(items, item)
	}
	return items, rows.Err()
}

// Close closes the database.
func (d *DB) Close() error { return d.db.Close() }

// Path returns the database file path.
func (d *DB) Path() string { return d.path }

func encodeJSON(v any) any {
	switch x := v.(type) {
	case []string:
		if len(x) == 0 {
			return nil
		}
	case map[string]string:
		if len(x) == 0 {
			return nil
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return string(b)
}

func nullStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullFloat(f float64) any {
	if f == 0 {
		return nil
	}
	return f
}

func nullInt64(n int64) any {
	if n == 0 {
		return nil
	}
	return n
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
