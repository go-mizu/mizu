package amazon

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DB wraps a DuckDB database for storing Amazon data.
type DB struct {
	db   *sql.DB
	path string
}

// OpenDB opens or creates a DuckDB database at the given path.
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
		`CREATE TABLE IF NOT EXISTS products (
			asin            VARCHAR PRIMARY KEY,
			title           VARCHAR,
			brand           VARCHAR,
			brand_id        VARCHAR,
			price           DOUBLE,
			currency        VARCHAR,
			list_price      DOUBLE,
			rating          DOUBLE,
			ratings_count   BIGINT,
			reviews_count   BIGINT,
			answered_qs     INTEGER,
			availability    VARCHAR,
			description     VARCHAR,
			bullet_points   VARCHAR,
			specs           VARCHAR,
			images          VARCHAR,
			category_path   VARCHAR,
			browse_node_ids VARCHAR,
			seller_id       VARCHAR,
			seller_name     VARCHAR,
			sold_by         VARCHAR,
			fulfilled_by    VARCHAR,
			variant_asins   VARCHAR,
			parent_asin     VARCHAR,
			similar_asins   VARCHAR,
			rank            INTEGER,
			rank_category   VARCHAR,
			url             VARCHAR,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS brands (
			brand_id       VARCHAR PRIMARY KEY,
			name           VARCHAR,
			description    VARCHAR,
			logo_url       VARCHAR,
			banner_url     VARCHAR,
			follower_count INTEGER,
			url            VARCHAR,
			featured_asins VARCHAR,
			fetched_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS authors (
			author_id      VARCHAR PRIMARY KEY,
			name           VARCHAR,
			bio            VARCHAR,
			photo_url      VARCHAR,
			website        VARCHAR,
			twitter        VARCHAR,
			book_asins     VARCHAR,
			follower_count INTEGER,
			url            VARCHAR,
			fetched_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			node_id        VARCHAR PRIMARY KEY,
			name           VARCHAR,
			parent_node_id VARCHAR,
			breadcrumb     VARCHAR,
			child_node_ids VARCHAR,
			top_asins      VARCHAR,
			url            VARCHAR,
			fetched_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS bestseller_lists (
			list_id       VARCHAR PRIMARY KEY,
			list_type     VARCHAR,
			category      VARCHAR,
			node_id       VARCHAR,
			snapshot_date DATE,
			url           VARCHAR,
			fetched_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS bestseller_entries (
			list_id       VARCHAR NOT NULL,
			asin          VARCHAR NOT NULL,
			rank          INTEGER,
			title         VARCHAR,
			price         DOUBLE,
			rating        DOUBLE,
			ratings_count BIGINT,
			PRIMARY KEY (list_id, asin)
		)`,
		`CREATE TABLE IF NOT EXISTS reviews (
			review_id         VARCHAR PRIMARY KEY,
			asin              VARCHAR,
			reviewer_id       VARCHAR,
			reviewer_name     VARCHAR,
			rating            INTEGER,
			title             VARCHAR,
			text              VARCHAR,
			date_posted       TIMESTAMP,
			verified_purchase BOOLEAN,
			helpful_votes     INTEGER,
			total_votes       INTEGER,
			images            VARCHAR,
			variant_attrs     VARCHAR,
			url               VARCHAR,
			fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS qa (
			qa_id           VARCHAR PRIMARY KEY,
			asin            VARCHAR,
			question        VARCHAR,
			question_by     VARCHAR,
			question_date   TIMESTAMP,
			answer          VARCHAR,
			answer_by       VARCHAR,
			answer_date     TIMESTAMP,
			helpful_votes   INTEGER,
			is_seller_answer BOOLEAN,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sellers (
			seller_id    VARCHAR PRIMARY KEY,
			name         VARCHAR,
			rating       DOUBLE,
			rating_count INTEGER,
			positive_pct DOUBLE,
			neutral_pct  DOUBLE,
			negative_pct DOUBLE,
			url          VARCHAR,
			fetched_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS search_results (
			search_id     VARCHAR PRIMARY KEY,
			query         VARCHAR,
			page          INTEGER,
			result_asins  VARCHAR,
			total_results VARCHAR,
			fetched_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

// UpsertProduct inserts or replaces a product record.
func (d *DB) UpsertProduct(p Product) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO products (
		asin, title, brand, brand_id,
		price, currency, list_price, rating,
		ratings_count, reviews_count, answered_qs,
		availability, description,
		bullet_points, specs, images,
		category_path, browse_node_ids,
		seller_id, seller_name, sold_by, fulfilled_by,
		variant_asins, parent_asin, similar_asins,
		rank, rank_category, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ASIN, nullStr(p.Title), nullStr(p.Brand), nullStr(p.BrandID),
		p.Price, nullStr(p.Currency), p.ListPrice, p.Rating,
		p.RatingsCount, p.ReviewsCount, nullInt(p.AnsweredQs),
		nullStr(p.Availability), nullStr(p.Description),
		encodeStringSlice(p.BulletPoints), encodeMap(p.Specs), encodeStringSlice(p.Images),
		encodeStringSlice(p.CategoryPath), encodeStringSlice(p.BrowseNodeIDs),
		nullStr(p.SellerID), nullStr(p.SellerName), nullStr(p.SoldBy), nullStr(p.FulfilledBy),
		encodeStringSlice(p.VariantASINs), nullStr(p.ParentASIN), encodeStringSlice(p.SimilarASINs),
		nullInt(p.Rank), nullStr(p.RankCategory), nullStr(p.URL), p.FetchedAt,
	)
	return err
}

// UpsertBrand inserts or replaces a brand record.
func (d *DB) UpsertBrand(b Brand) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO brands (
		brand_id, name, description, logo_url, banner_url,
		follower_count, url, featured_asins, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?)`,
		b.BrandID, nullStr(b.Name), nullStr(b.Description),
		nullStr(b.LogoURL), nullStr(b.BannerURL),
		b.FollowerCount, nullStr(b.URL),
		encodeStringSlice(b.FeaturedASINs), b.FetchedAt,
	)
	return err
}

// UpsertAuthor inserts or replaces an author record.
func (d *DB) UpsertAuthor(a Author) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO authors (
		author_id, name, bio, photo_url, website, twitter,
		book_asins, follower_count, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		a.AuthorID, nullStr(a.Name), nullStr(a.Bio),
		nullStr(a.PhotoURL), nullStr(a.Website), nullStr(a.Twitter),
		encodeStringSlice(a.BookASINs), a.FollowerCount,
		nullStr(a.URL), a.FetchedAt,
	)
	return err
}

// UpsertCategory inserts or replaces a category record.
func (d *DB) UpsertCategory(c Category) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO categories (
		node_id, name, parent_node_id,
		breadcrumb, child_node_ids, top_asins, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?)`,
		c.NodeID, nullStr(c.Name), nullStr(c.ParentNodeID),
		encodeStringSlice(c.Breadcrumb), encodeStringSlice(c.ChildNodeIDs),
		encodeStringSlice(c.TopASINs), nullStr(c.URL), c.FetchedAt,
	)
	return err
}

// UpsertBestsellerList inserts or replaces a bestseller list record.
func (d *DB) UpsertBestsellerList(l BestsellerList) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO bestseller_lists (
		list_id, list_type, category, node_id, snapshot_date, url, fetched_at
	) VALUES (?,?,?,?,?,?,?)`,
		l.ListID, nullStr(l.ListType), nullStr(l.Category),
		nullStr(l.NodeID), nullStr(l.SnapshotDate),
		nullStr(l.URL), l.FetchedAt,
	)
	return err
}

// InsertBestsellerEntries upserts bestseller entries in a transaction.
func (d *DB) InsertBestsellerEntries(entries []BestsellerEntry) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, e := range entries {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO bestseller_entries (list_id, asin, rank, title, price, rating, ratings_count) VALUES (?,?,?,?,?,?,?)`,
			e.ListID, e.ASIN, e.Rank, nullStr(e.Title), e.Price, e.Rating, e.RatingsCount,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpsertReview inserts or replaces a review record.
func (d *DB) UpsertReview(r Review) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO reviews (
		review_id, asin, reviewer_id, reviewer_name, rating, title, text,
		date_posted, verified_purchase, helpful_votes, total_votes,
		images, variant_attrs, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.ReviewID, nullStr(r.ASIN), nullStr(r.ReviewerID), nullStr(r.ReviewerName),
		r.Rating, nullStr(r.Title), nullStr(r.Text),
		nullTime(r.DatePosted), r.VerifiedPurchase, r.HelpfulVotes, r.TotalVotes,
		encodeStringSlice(r.Images), encodeMap(r.VariantAttrs),
		nullStr(r.URL), r.FetchedAt,
	)
	return err
}

// UpsertQA inserts or replaces a QA record.
func (d *DB) UpsertQA(q QA) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO qa (
		qa_id, asin, question, question_by, question_date,
		answer, answer_by, answer_date,
		helpful_votes, is_seller_answer, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		q.QAID, nullStr(q.ASIN), nullStr(q.Question), nullStr(q.QuestionBy), nullTime(q.QuestionDate),
		nullStr(q.Answer), nullStr(q.AnswerBy), nullTime(q.AnswerDate),
		q.HelpfulVotes, q.IsSellerAnswer, q.FetchedAt,
	)
	return err
}

// UpsertSeller inserts or replaces a seller record.
func (d *DB) UpsertSeller(s Seller) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO sellers (
		seller_id, name, rating, rating_count,
		positive_pct, neutral_pct, negative_pct, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?)`,
		s.SellerID, nullStr(s.Name), s.Rating, s.RatingCount,
		s.PositivePct, s.NeutralPct, s.NegativePct,
		nullStr(s.URL), s.FetchedAt,
	)
	return err
}

// UpsertSearchResult inserts or replaces a search result record.
func (d *DB) UpsertSearchResult(sr SearchResult) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO search_results (
		search_id, query, page, result_asins, total_results, fetched_at
	) VALUES (?,?,?,?,?,?)`,
		sr.SearchID, nullStr(sr.Query), sr.Page,
		encodeStringSlice(sr.ResultASINs), nullStr(sr.TotalResults), sr.FetchedAt,
	)
	return err
}

// DBStats holds row counts per table.
type DBStats struct {
	Products    int64
	Brands      int64
	Authors     int64
	Categories  int64
	Bestsellers int64
	Reviews     int64
	QAs         int64
	Sellers     int64
	Searches    int64
	DBSize      int64
}

// GetStats returns row counts for all tables.
func (d *DB) GetStats() (DBStats, error) {
	var s DBStats
	d.db.QueryRow("SELECT COUNT(*) FROM products").Scan(&s.Products)
	d.db.QueryRow("SELECT COUNT(*) FROM brands").Scan(&s.Brands)
	d.db.QueryRow("SELECT COUNT(*) FROM authors").Scan(&s.Authors)
	d.db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&s.Categories)
	d.db.QueryRow("SELECT COUNT(*) FROM bestseller_lists").Scan(&s.Bestsellers)
	d.db.QueryRow("SELECT COUNT(*) FROM reviews").Scan(&s.Reviews)
	d.db.QueryRow("SELECT COUNT(*) FROM qa").Scan(&s.QAs)
	d.db.QueryRow("SELECT COUNT(*) FROM sellers").Scan(&s.Sellers)
	d.db.QueryRow("SELECT COUNT(*) FROM search_results").Scan(&s.Searches)
	if fi, err := os.Stat(d.path); err == nil {
		s.DBSize = fi.Size()
	}
	return s, nil
}

// Close closes the database.
func (d *DB) Close() error {
	return d.db.Close()
}

// Path returns the database file path.
func (d *DB) Path() string {
	return d.path
}

// ── helpers ──────────────────────────────────────────────────────────────────

func encodeStringSlice(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ss)
	return string(b)
}

func decodeStringSlice(s string) []string {
	if s == "" || s == "[]" {
		return nil
	}
	var out []string
	json.Unmarshal([]byte(s), &out)
	return out
}

func encodeMap(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullInt(i int) any {
	if i == 0 {
		return nil
	}
	return i
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
