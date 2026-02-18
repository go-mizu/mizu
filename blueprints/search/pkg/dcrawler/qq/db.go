package qq

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DB wraps a DuckDB database for storing QQ News data.
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
		`CREATE TABLE IF NOT EXISTS articles (
			article_id    VARCHAR PRIMARY KEY,
			title         VARCHAR,
			content       VARCHAR,
			abstract      VARCHAR,
			publish_time  TIMESTAMP,
			channel       VARCHAR,
			source        VARCHAR,
			source_id     VARCHAR,
			article_type  INTEGER DEFAULT 0,
			url           VARCHAR,
			image_url     VARCHAR,
			comment_id    VARCHAR,
			crawled_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status_code   INTEGER,
			error         VARCHAR
		)`,
		`CREATE TABLE IF NOT EXISTS sitemaps (
			url           VARCHAR PRIMARY KEY,
			fetched_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			article_count INTEGER DEFAULT 0
		)`,
	}

	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

// InsertArticles bulk-inserts articles into the database.
func (d *DB) InsertArticles(articles []Article) error {
	if len(articles) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const batchSize = 500
	const nCols = 15
	placeholders := "(" + strings.Repeat("?,", nCols-1) + "?)"

	for i := 0; i < len(articles); i += batchSize {
		end := min(i+batchSize, len(articles))
		batch := articles[i:end]

		var sb strings.Builder
		sb.WriteString(`INSERT OR REPLACE INTO articles (
			article_id, title, content, abstract, publish_time,
			channel, source, source_id, article_type, url,
			image_url, comment_id, crawled_at, status_code, error
		) VALUES `)

		args := make([]any, 0, len(batch)*nCols)
		for j, a := range batch {
			if j > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(placeholders)

			args = append(args,
				a.ArticleID, a.Title, nullStr(a.Content), nullStr(a.Abstract),
				nullTime(a.PublishTime),
				nullStr(a.Channel), nullStr(a.Source), nullStr(a.SourceID),
				a.ArticleType, nullStr(a.URL),
				nullStr(a.ImageURL), nullStr(a.CommentID),
				a.CrawledAt, a.StatusCode, nullStr(a.Error),
			)
		}

		if _, err := tx.Exec(sb.String(), args...); err != nil {
			return fmt.Errorf("insert articles batch: %w", err)
		}
	}

	return tx.Commit()
}

// MarkSitemap records that a sitemap URL has been fetched.
func (d *DB) MarkSitemap(url string, articleCount int) error {
	_, err := d.db.Exec(
		`INSERT OR REPLACE INTO sitemaps (url, fetched_at, article_count) VALUES (?, ?, ?)`,
		url, time.Now(), articleCount,
	)
	return err
}

// FetchedSitemaps returns the set of already-fetched sitemap URLs.
func (d *DB) FetchedSitemaps() (map[string]bool, error) {
	rows, err := d.db.Query("SELECT url FROM sitemaps")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return result, err
		}
		result[url] = true
	}
	return result, rows.Err()
}

// CrawledArticleIDs returns article IDs that were successfully crawled.
func (d *DB) CrawledArticleIDs() ([]string, error) {
	return d.queryIDs("SELECT article_id FROM articles WHERE error IS NULL")
}

// AllArticleIDs returns all article IDs in the database (including deleted/errored).
func (d *DB) AllArticleIDs() ([]string, error) {
	return d.queryIDs("SELECT article_id FROM articles")
}

func (d *DB) queryIDs(query string) ([]string, error) {
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DBStats holds database statistics.
type DBStats struct {
	Articles    int64
	Sitemaps    int64
	WithContent int64
	Deleted     int64
	WithError   int64
	Channels    map[string]int64
	DBSize      int64
}

// GetStats returns database statistics.
func (d *DB) GetStats() (DBStats, error) {
	var stats DBStats
	stats.Channels = make(map[string]int64)

	d.db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&stats.Articles)
	d.db.QueryRow("SELECT COUNT(*) FROM sitemaps").Scan(&stats.Sitemaps)
	d.db.QueryRow("SELECT COUNT(*) FROM articles WHERE content IS NOT NULL AND content != ''").Scan(&stats.WithContent)
	d.db.QueryRow("SELECT COUNT(*) FROM articles WHERE error = 'deleted'").Scan(&stats.Deleted)
	d.db.QueryRow("SELECT COUNT(*) FROM articles WHERE error IS NOT NULL AND error != 'deleted'").Scan(&stats.WithError)

	rows, err := d.db.Query("SELECT COALESCE(channel, 'unknown'), COUNT(*) FROM articles GROUP BY channel ORDER BY COUNT(*) DESC")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ch string
			var cnt int64
			if rows.Scan(&ch, &cnt) == nil {
				stats.Channels[ch] = cnt
			}
		}
	}

	if fi, err := os.Stat(d.path); err == nil {
		stats.DBSize = fi.Size()
	}

	return stats, nil
}

// TopArticles returns the most recent N articles.
func (d *DB) TopArticles(limit int) ([]Article, error) {
	rows, err := d.db.Query(`
		SELECT article_id, title, abstract, publish_time, channel, source, article_type, url
		FROM articles
		WHERE error IS NULL
		ORDER BY publish_time DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		var a Article
		var pubTime sql.NullTime
		var abstract, channel, source, url sql.NullString
		err := rows.Scan(&a.ArticleID, &a.Title, &abstract, &pubTime, &channel, &source, &a.ArticleType, &url)
		if err != nil {
			return articles, err
		}
		if pubTime.Valid {
			a.PublishTime = pubTime.Time
		}
		a.Abstract = abstract.String
		a.Channel = channel.String
		a.Source = source.String
		a.URL = url.String
		articles = append(articles, a)
	}
	return articles, rows.Err()
}

// Close closes the database.
func (d *DB) Close() error {
	return d.db.Close()
}

// Path returns the database file path.
func (d *DB) Path() string {
	return d.path
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
