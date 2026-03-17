package soundcloud

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
		`CREATE TABLE IF NOT EXISTS users (
			user_id BIGINT PRIMARY KEY,
			username VARCHAR,
			full_name VARCHAR,
			description VARCHAR,
			avatar_url VARCHAR,
			city VARCHAR,
			country_code VARCHAR,
			followers_count INTEGER,
			followings_count INTEGER,
			track_count INTEGER,
			playlist_count INTEGER,
			likes_count INTEGER,
			playlist_likes_count INTEGER,
			verified BOOLEAN,
			url VARCHAR,
			created_at TIMESTAMP,
			fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS tracks (
			track_id BIGINT PRIMARY KEY,
			user_id BIGINT,
			title VARCHAR,
			description VARCHAR,
			genre VARCHAR,
			tag_list VARCHAR,
			artwork_url VARCHAR,
			waveform_url VARCHAR,
			label_name VARCHAR,
			license VARCHAR,
			duration_ms BIGINT,
			playback_count BIGINT,
			likes_count INTEGER,
			comment_count INTEGER,
			download_count INTEGER,
			reposts_count INTEGER,
			downloadable BOOLEAN,
			streamable BOOLEAN,
			release_date TIMESTAMP,
			created_at TIMESTAMP,
			url VARCHAR,
			fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS playlists (
			playlist_id BIGINT PRIMARY KEY,
			user_id BIGINT,
			title VARCHAR,
			description VARCHAR,
			artwork_url VARCHAR,
			track_count INTEGER,
			duration_ms BIGINT,
			likes_count INTEGER,
			reposts_count INTEGER,
			set_type VARCHAR,
			is_album BOOLEAN,
			created_at TIMESTAMP,
			published_at TIMESTAMP,
			url VARCHAR,
			fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS playlist_tracks (
			playlist_id BIGINT NOT NULL,
			track_id BIGINT NOT NULL,
			position INTEGER,
			track_url VARCHAR,
			PRIMARY KEY (playlist_id, track_id)
		)`,
		`CREATE TABLE IF NOT EXISTS comments (
			comment_id VARCHAR PRIMARY KEY,
			track_id BIGINT,
			user_name VARCHAR,
			user_url VARCHAR,
			body VARCHAR,
			posted_at TIMESTAMP,
			fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS search_results (
			search_id VARCHAR PRIMARY KEY,
			query VARCHAR,
			kind VARCHAR,
			title VARCHAR,
			url VARCHAR,
			fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

func (d *DB) UpsertUser(u User) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO users (
		user_id, username, full_name, description, avatar_url, city, country_code,
		followers_count, followings_count, track_count, playlist_count,
		likes_count, playlist_likes_count, verified, url, created_at, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		u.UserID, nullStr(u.Username), nullStr(u.FullName), nullStr(u.Description),
		nullStr(u.AvatarURL), nullStr(u.City), nullStr(u.CountryCode),
		u.FollowersCount, u.FollowingsCount, u.TrackCount, u.PlaylistCount,
		u.LikesCount, u.PlaylistLikesCount, u.Verified, nullStr(u.URL),
		nullTime(u.CreatedAt), u.FetchedAt,
	)
	return err
}

func (d *DB) UpsertTrack(t Track) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO tracks (
		track_id, user_id, title, description, genre, tag_list, artwork_url, waveform_url,
		label_name, license, duration_ms, playback_count, likes_count, comment_count,
		download_count, reposts_count, downloadable, streamable, release_date, created_at, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.TrackID, nullInt64(t.UserID), nullStr(t.Title), nullStr(t.Description), nullStr(t.Genre),
		nullStr(t.TagList), nullStr(t.ArtworkURL), nullStr(t.WaveformURL), nullStr(t.LabelName),
		nullStr(t.License), t.DurationMS, t.PlaybackCount, t.LikesCount, t.CommentCount,
		t.DownloadCount, t.RepostsCount, t.Downloadable, t.Streamable, nullTime(t.ReleaseDate),
		nullTime(t.CreatedAt), nullStr(t.URL), t.FetchedAt,
	)
	return err
}

func (d *DB) UpsertPlaylist(p Playlist) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO playlists (
		playlist_id, user_id, title, description, artwork_url, track_count, duration_ms,
		likes_count, reposts_count, set_type, is_album, created_at, published_at, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.PlaylistID, nullInt64(p.UserID), nullStr(p.Title), nullStr(p.Description),
		nullStr(p.ArtworkURL), p.TrackCount, p.DurationMS, p.LikesCount, p.RepostsCount,
		nullStr(p.SetType), p.IsAlbum, nullTime(p.CreatedAt), nullTime(p.PublishedAt),
		nullStr(p.URL), p.FetchedAt,
	)
	return err
}

func (d *DB) InsertPlaylistTracks(rels []PlaylistTrack) error {
	if len(rels) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, rel := range rels {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO playlist_tracks (playlist_id, track_id, position, track_url) VALUES (?,?,?,?)`,
			rel.PlaylistID, rel.TrackID, rel.Position, nullStr(rel.TrackURL),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) InsertComments(comments []Comment) error {
	if len(comments) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, c := range comments {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO comments (comment_id, track_id, user_name, user_url, body, posted_at, fetched_at) VALUES (?,?,?,?,?,?,?)`,
			c.CommentID, c.TrackID, nullStr(c.UserName), nullStr(c.UserURL), nullStr(c.Body), nullTime(c.PostedAt), c.FetchedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) UpsertSearchResults(results []SearchResult) error {
	if len(results) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, r := range results {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO search_results (search_id, query, kind, title, url, fetched_at) VALUES (?,?,?,?,?,?)`,
			r.SearchID, r.Query, r.Kind, nullStr(r.Title), nullStr(r.URL), r.FetchedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type DBStats struct {
	Users      int64
	Tracks     int64
	Playlists  int64
	Comments   int64
	SearchRows int64
	DBSize     int64
}

func (d *DB) GetStats() (DBStats, error) {
	var s DBStats
	d.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&s.Users)
	d.db.QueryRow(`SELECT COUNT(*) FROM tracks`).Scan(&s.Tracks)
	d.db.QueryRow(`SELECT COUNT(*) FROM playlists`).Scan(&s.Playlists)
	d.db.QueryRow(`SELECT COUNT(*) FROM comments`).Scan(&s.Comments)
	d.db.QueryRow(`SELECT COUNT(*) FROM search_results`).Scan(&s.SearchRows)
	if fi, err := os.Stat(d.path); err == nil {
		s.DBSize = fi.Size()
	}
	return s, nil
}

func (d *DB) Close() error { return d.db.Close() }
func (d *DB) Path() string { return d.path }

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

func nullInt64(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}
