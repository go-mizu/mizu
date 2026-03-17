package pinterest

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DB wraps a DuckDB database for storing Pinterest data.
type DB struct {
	db   *sql.DB
	path string
}

// OpenDB opens or creates the Pinterest DuckDB database at the given path.
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
		`CREATE TABLE IF NOT EXISTS pins (
			pin_id         VARCHAR PRIMARY KEY,
			title          VARCHAR,
			description    VARCHAR,
			alt_text       VARCHAR,
			image_url      VARCHAR,
			image_width    INTEGER,
			image_height   INTEGER,
			pin_url        VARCHAR,
			source_url     VARCHAR,
			board_id       VARCHAR,
			board_name     VARCHAR,
			user_id        VARCHAR,
			username       VARCHAR,
			saved_count    INTEGER,
			comment_count  INTEGER,
			created_at     TIMESTAMP,
			fetched_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS boards (
			board_id       VARCHAR PRIMARY KEY,
			name           VARCHAR,
			slug           VARCHAR,
			description    VARCHAR,
			user_id        VARCHAR,
			username       VARCHAR,
			pin_count      INTEGER,
			follower_count INTEGER,
			cover_url      VARCHAR,
			category       VARCHAR,
			is_secret      BOOLEAN DEFAULT FALSE,
			url            VARCHAR,
			fetched_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			user_id          VARCHAR PRIMARY KEY,
			username         VARCHAR,
			full_name        VARCHAR,
			bio              VARCHAR,
			website          VARCHAR,
			follower_count   INTEGER,
			following_count  INTEGER,
			board_count      INTEGER,
			pin_count        INTEGER,
			monthly_views    BIGINT,
			avatar_url       VARCHAR,
			url              VARCHAR,
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

// UpsertPin inserts or replaces a pin record.
func (d *DB) UpsertPin(p Pin) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO pins (
		pin_id, title, description, alt_text,
		image_url, image_width, image_height,
		pin_url, source_url,
		board_id, board_name, user_id, username,
		saved_count, comment_count, created_at, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.PinID, nullStr(p.Title), nullStr(p.Description), nullStr(p.AltText),
		nullStr(p.ImageURL), nullInt(p.ImageWidth), nullInt(p.ImageHeight),
		nullStr(p.PinURL), nullStr(p.SourceURL),
		nullStr(p.BoardID), nullStr(p.BoardName), nullStr(p.UserID), nullStr(p.Username),
		p.SavedCount, p.CommentCount, nullTime(p.CreatedAt), p.FetchedAt,
	)
	return err
}

// UpsertBoard inserts or replaces a board record.
func (d *DB) UpsertBoard(b Board) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO boards (
		board_id, name, slug, description,
		user_id, username, pin_count, follower_count,
		cover_url, category, is_secret, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		b.BoardID, nullStr(b.Name), nullStr(b.Slug), nullStr(b.Description),
		nullStr(b.UserID), nullStr(b.Username), b.PinCount, b.FollowerCount,
		nullStr(b.CoverURL), nullStr(b.Category), b.IsSecret, nullStr(b.URL), b.FetchedAt,
	)
	return err
}

// UpsertUser inserts or replaces a user record.
func (d *DB) UpsertUser(u User) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO users (
		user_id, username, full_name, bio, website,
		follower_count, following_count, board_count, pin_count,
		monthly_views, avatar_url, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		u.UserID, nullStr(u.Username), nullStr(u.FullName), nullStr(u.Bio), nullStr(u.Website),
		u.FollowerCount, u.FollowingCount, u.BoardCount, u.PinCount,
		u.MonthlyViews, nullStr(u.AvatarURL), nullStr(u.URL), u.FetchedAt,
	)
	return err
}

// GetStats returns row counts for all tables.
func (d *DB) GetStats() (DBStats, error) {
	var s DBStats
	d.db.QueryRow("SELECT COUNT(*) FROM pins").Scan(&s.Pins)
	d.db.QueryRow("SELECT COUNT(*) FROM boards").Scan(&s.Boards)
	d.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&s.Users)
	if fi, err := os.Stat(d.path); err == nil {
		s.DBSize = fi.Size()
	}
	return s, nil
}

// RecentPins returns the most recently fetched pins.
func (d *DB) RecentPins(limit int) ([]Pin, error) {
	rows, err := d.db.Query(`
		SELECT pin_id, title, username, saved_count, pin_url
		FROM pins ORDER BY fetched_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pins []Pin
	for rows.Next() {
		var p Pin
		var title, username, pinURL sql.NullString
		if err := rows.Scan(&p.PinID, &title, &username, &p.SavedCount, &pinURL); err != nil {
			return pins, err
		}
		p.Title = title.String
		p.Username = username.String
		p.PinURL = pinURL.String
		pins = append(pins, p)
	}
	return pins, rows.Err()
}

// Close closes the database.
func (d *DB) Close() error { return d.db.Close() }

// Path returns the database file path.
func (d *DB) Path() string { return d.path }

// ── helpers ──────────────────────────────────────────────────────────────────

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

