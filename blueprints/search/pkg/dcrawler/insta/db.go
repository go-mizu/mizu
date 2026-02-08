package insta

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DB wraps a DuckDB database for storing Instagram data.
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
		`CREATE TABLE IF NOT EXISTS posts (
			id              VARCHAR PRIMARY KEY,
			shortcode       VARCHAR NOT NULL,
			type_name       VARCHAR NOT NULL,
			caption         VARCHAR,
			display_url     VARCHAR NOT NULL,
			video_url       VARCHAR,
			is_video        BOOLEAN DEFAULT FALSE,
			width           INTEGER,
			height          INTEGER,
			like_count      BIGINT DEFAULT 0,
			comment_count   BIGINT DEFAULT 0,
			view_count      BIGINT DEFAULT 0,
			taken_at        TIMESTAMP,
			location_id     VARCHAR,
			location_name   VARCHAR,
			owner_id        VARCHAR,
			owner_username  VARCHAR,
			children_count  INTEGER DEFAULT 0,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS comments (
			id              VARCHAR PRIMARY KEY,
			post_id         VARCHAR NOT NULL,
			text            VARCHAR,
			author_id       VARCHAR,
			author_name     VARCHAR,
			like_count      BIGINT DEFAULT 0,
			created_at      TIMESTAMP,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS media (
			url             VARCHAR PRIMARY KEY,
			post_id         VARCHAR NOT NULL,
			shortcode       VARCHAR NOT NULL,
			type            VARCHAR NOT NULL,
			width           INTEGER,
			height          INTEGER,
			idx             INTEGER DEFAULT 0,
			downloaded      BOOLEAN DEFAULT FALSE,
			local_path      VARCHAR,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

// InsertPosts bulk-inserts posts into the database.
func (d *DB) InsertPosts(posts []Post) error {
	if len(posts) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Build batch INSERT using VALUES
	const batchSize = 500
	for i := 0; i < len(posts); i += batchSize {
		end := i + batchSize
		if end > len(posts) {
			end = len(posts)
		}
		batch := posts[i:end]

		var sb strings.Builder
		sb.WriteString(`INSERT OR REPLACE INTO posts (id, shortcode, type_name, caption, display_url, video_url, is_video, width, height, like_count, comment_count, view_count, taken_at, location_id, location_name, owner_id, owner_username, children_count, fetched_at) VALUES `)

		args := make([]any, 0, len(batch)*19)
		for j, p := range batch {
			if j > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString("(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")

			childCount := len(p.Children)
			args = append(args,
				p.ID, p.Shortcode, p.TypeName, p.Caption, p.DisplayURL,
				nullStr(p.VideoURL), p.IsVideo, p.Width, p.Height,
				p.LikeCount, p.CommentCount, p.ViewCount,
				p.TakenAt, nullStr(p.LocationID), nullStr(p.LocationName),
				p.OwnerID, p.OwnerName, childCount, p.FetchedAt,
			)
		}

		if _, err := tx.Exec(sb.String(), args...); err != nil {
			return fmt.Errorf("insert posts batch: %w", err)
		}
	}

	return tx.Commit()
}

// InsertComments bulk-inserts comments into the database.
func (d *DB) InsertComments(comments []Comment) error {
	if len(comments) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const batchSize = 500
	for i := 0; i < len(comments); i += batchSize {
		end := i + batchSize
		if end > len(comments) {
			end = len(comments)
		}
		batch := comments[i:end]

		var sb strings.Builder
		sb.WriteString(`INSERT OR REPLACE INTO comments (id, post_id, text, author_id, author_name, like_count, created_at, fetched_at) VALUES `)

		args := make([]any, 0, len(batch)*8)
		for j, c := range batch {
			if j > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString("(?,?,?,?,?,?,?,?)")
			args = append(args, c.ID, c.PostID, c.Text, c.AuthorID, c.AuthorName, c.LikeCount, c.CreatedAt, time.Now())
		}

		if _, err := tx.Exec(sb.String(), args...); err != nil {
			return fmt.Errorf("insert comments batch: %w", err)
		}
	}

	return tx.Commit()
}

// InsertMedia bulk-inserts media items into the database.
func (d *DB) InsertMedia(items []MediaItem) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const batchSize = 500
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]

		var sb strings.Builder
		sb.WriteString(`INSERT OR REPLACE INTO media (url, post_id, shortcode, type, width, height, idx, fetched_at) VALUES `)

		args := make([]any, 0, len(batch)*8)
		for j, m := range batch {
			if j > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString("(?,?,?,?,?,?,?,?)")
			args = append(args, m.URL, m.PostID, m.Shortcode, m.Type, m.Width, m.Height, m.Index, time.Now())
		}

		if _, err := tx.Exec(sb.String(), args...); err != nil {
			return fmt.Errorf("insert media batch: %w", err)
		}
	}

	return tx.Commit()
}

// Stats returns statistics about the database.
type DBStats struct {
	Posts    int64
	Comments int64
	Media    int64
	DBSize  int64
}

// GetStats returns database statistics.
func (d *DB) GetStats() (DBStats, error) {
	var stats DBStats

	row := d.db.QueryRow("SELECT COUNT(*) FROM posts")
	row.Scan(&stats.Posts)

	row = d.db.QueryRow("SELECT COUNT(*) FROM comments")
	row.Scan(&stats.Comments)

	row = d.db.QueryRow("SELECT COUNT(*) FROM media")
	row.Scan(&stats.Media)

	if fi, err := os.Stat(d.path); err == nil {
		stats.DBSize = fi.Size()
	}

	return stats, nil
}

// TopPosts returns the top N posts by like count.
func (d *DB) TopPosts(limit int) ([]Post, error) {
	rows, err := d.db.Query(`
		SELECT id, shortcode, type_name, caption, display_url, video_url,
		       is_video, width, height, like_count, comment_count, view_count,
		       taken_at, location_id, location_name, owner_id, owner_username,
		       children_count, fetched_at
		FROM posts
		ORDER BY like_count DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		var videoURL, locID, locName sql.NullString
		err := rows.Scan(
			&p.ID, &p.Shortcode, &p.TypeName, &p.Caption, &p.DisplayURL,
			&videoURL, &p.IsVideo, &p.Width, &p.Height,
			&p.LikeCount, &p.CommentCount, &p.ViewCount,
			&p.TakenAt, &locID, &locName,
			&p.OwnerID, &p.OwnerName,
			new(int), &p.FetchedAt,
		)
		if err != nil {
			return posts, err
		}
		p.VideoURL = videoURL.String
		p.LocationID = locID.String
		p.LocationName = locName.String
		posts = append(posts, p)
	}
	return posts, rows.Err()
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
