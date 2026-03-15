package facebook

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

type DB struct {
	db   *sql.DB
	path string
}

func OpenDB(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
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
		`CREATE TABLE IF NOT EXISTS pages (
			page_id         VARCHAR PRIMARY KEY,
			slug            VARCHAR,
			name            VARCHAR,
			category        VARCHAR,
			about           VARCHAR,
			likes_count     BIGINT,
			followers_count BIGINT,
			verified        BOOLEAN,
			website         VARCHAR,
			phone           VARCHAR,
			address         VARCHAR,
			url             VARCHAR,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS profiles (
			profile_id      VARCHAR PRIMARY KEY,
			username        VARCHAR,
			name            VARCHAR,
			intro           VARCHAR,
			bio             VARCHAR,
			followers_count BIGINT,
			friends_count   BIGINT,
			verified        BOOLEAN,
			hometown        VARCHAR,
			current_city    VARCHAR,
			work            VARCHAR,
			education       VARCHAR,
			url             VARCHAR,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS groups (
			group_id       VARCHAR PRIMARY KEY,
			slug           VARCHAR,
			name           VARCHAR,
			description    VARCHAR,
			privacy        VARCHAR,
			members_count  BIGINT,
			url            VARCHAR,
			fetched_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			post_id          VARCHAR PRIMARY KEY,
			owner_id         VARCHAR,
			owner_name       VARCHAR,
			owner_type       VARCHAR,
			text             VARCHAR,
			created_at_text  VARCHAR,
			like_count       BIGINT,
			comment_count    BIGINT,
			share_count      BIGINT,
			permalink        VARCHAR,
			media_urls       VARCHAR,
			external_links   VARCHAR,
			fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS comments (
			comment_id       VARCHAR PRIMARY KEY,
			post_id          VARCHAR,
			author_id        VARCHAR,
			author_name      VARCHAR,
			text             VARCHAR,
			created_at_text  VARCHAR,
			like_count       BIGINT,
			permalink        VARCHAR,
			fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS search_results (
			query       VARCHAR NOT NULL,
			result_url  VARCHAR NOT NULL,
			entity_type VARCHAR,
			title       VARCHAR,
			snippet     VARCHAR,
			fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (query, result_url)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

func (d *DB) UpsertPage(p Page) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO pages (
		page_id, slug, name, category, about, likes_count, followers_count, verified,
		website, phone, address, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.PageID, nullStr(p.Slug), nullStr(p.Name), nullStr(p.Category), nullStr(p.About),
		p.LikesCount, p.FollowersCount, p.Verified, nullStr(p.Website), nullStr(p.Phone),
		nullStr(p.Address), nullStr(p.URL), p.FetchedAt,
	)
	return err
}

func (d *DB) UpsertProfile(p Profile) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO profiles (
		profile_id, username, name, intro, bio, followers_count, friends_count, verified,
		hometown, current_city, work, education, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ProfileID, nullStr(p.Username), nullStr(p.Name), nullStr(p.Intro), nullStr(p.Bio),
		p.FollowersCount, p.FriendsCount, p.Verified, nullStr(p.Hometown), nullStr(p.CurrentCity),
		nullStr(p.Work), nullStr(p.Education), nullStr(p.URL), p.FetchedAt,
	)
	return err
}

func (d *DB) UpsertGroup(g Group) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO groups (
		group_id, slug, name, description, privacy, members_count, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?)`,
		g.GroupID, nullStr(g.Slug), nullStr(g.Name), nullStr(g.Description), nullStr(g.Privacy),
		g.MembersCount, nullStr(g.URL), g.FetchedAt,
	)
	return err
}

func (d *DB) UpsertPost(p Post) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO posts (
		post_id, owner_id, owner_name, owner_type, text, created_at_text, like_count,
		comment_count, share_count, permalink, media_urls, external_links, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.PostID, nullStr(p.OwnerID), nullStr(p.OwnerName), nullStr(p.OwnerType), nullStr(p.Text),
		nullStr(p.CreatedAtText), p.LikeCount, p.CommentCount, p.ShareCount, nullStr(p.Permalink),
		encodeStringSlice(p.MediaURLs), encodeStringSlice(p.ExternalLinks), p.FetchedAt,
	)
	return err
}

func (d *DB) InsertComments(items []Comment) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, c := range items {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO comments (
			comment_id, post_id, author_id, author_name, text, created_at_text, like_count,
			permalink, fetched_at
		) VALUES (?,?,?,?,?,?,?,?,?)`,
			c.CommentID, nullStr(c.PostID), nullStr(c.AuthorID), nullStr(c.AuthorName),
			nullStr(c.Text), nullStr(c.CreatedAtText), c.LikeCount, nullStr(c.Permalink), c.FetchedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) InsertSearchResults(items []SearchResult) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, r := range items {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO search_results (
			query, result_url, entity_type, title, snippet, fetched_at
		) VALUES (?,?,?,?,?,?)`,
			r.Query, r.ResultURL, nullStr(r.EntityType), nullStr(r.Title), nullStr(r.Snippet), r.FetchedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type DBStats struct {
	Pages         int64
	Profiles      int64
	Groups        int64
	Posts         int64
	Comments      int64
	SearchResults int64
	DBSize        int64
}

func (d *DB) GetStats() (DBStats, error) {
	var s DBStats
	d.db.QueryRow(`SELECT COUNT(*) FROM pages`).Scan(&s.Pages)
	d.db.QueryRow(`SELECT COUNT(*) FROM profiles`).Scan(&s.Profiles)
	d.db.QueryRow(`SELECT COUNT(*) FROM groups`).Scan(&s.Groups)
	d.db.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&s.Posts)
	d.db.QueryRow(`SELECT COUNT(*) FROM comments`).Scan(&s.Comments)
	d.db.QueryRow(`SELECT COUNT(*) FROM search_results`).Scan(&s.SearchResults)
	if fi, err := os.Stat(d.path); err == nil {
		s.DBSize = fi.Size()
	}
	return s, nil
}

func (d *DB) RecentPosts(limit int) ([]Post, error) {
	rows, err := d.db.Query(`SELECT post_id, owner_name, text, permalink FROM posts ORDER BY fetched_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Post
	for rows.Next() {
		var p Post
		var ownerName, text, permalink sql.NullString
		if err := rows.Scan(&p.PostID, &ownerName, &text, &permalink); err != nil {
			return out, err
		}
		p.OwnerName = ownerName.String
		p.Text = text.String
		p.Permalink = permalink.String
		out = append(out, p)
	}
	return out, rows.Err()
}

func (d *DB) Close() error {
	return d.db.Close()
}

func encodeStringSlice(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ss)
	return string(b)
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func HumanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
