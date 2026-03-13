package youtube

import (
	"database/sql"
	"encoding/json"
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

type Stats struct {
	Videos       int64
	Channels     int64
	Playlists    int64
	PlaylistRows int64
	RelatedRows  int64
	CaptionRows  int64
	DBSize       int64
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
		`CREATE TABLE IF NOT EXISTS videos (
			video_id             VARCHAR PRIMARY KEY,
			title                VARCHAR,
			description          VARCHAR,
			channel_id           VARCHAR,
			channel_name         VARCHAR,
			duration_seconds     INTEGER,
			duration_text        VARCHAR,
			view_count           BIGINT,
			comment_count        BIGINT,
			like_count           BIGINT,
			published_text       VARCHAR,
			published_at         TIMESTAMP,
			upload_date          VARCHAR,
			is_live              BOOLEAN DEFAULT FALSE,
			is_short             BOOLEAN DEFAULT FALSE,
			category             VARCHAR,
			tags                 VARCHAR,
			thumbnail_url        VARCHAR,
			url                  VARCHAR,
			embed_url            VARCHAR,
			transcript           VARCHAR,
			transcript_language  VARCHAR,
			fetched_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS channels (
			channel_id           VARCHAR PRIMARY KEY,
			handle               VARCHAR,
			title                VARCHAR,
			description          VARCHAR,
			avatar_url           VARCHAR,
			banner_url           VARCHAR,
			subscribers_text     VARCHAR,
			videos_text          VARCHAR,
			views_text           VARCHAR,
			country              VARCHAR,
			joined_date_text     VARCHAR,
			uploads_playlist_id  VARCHAR,
			url                  VARCHAR,
			fetched_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS playlists (
			playlist_id          VARCHAR PRIMARY KEY,
			title                VARCHAR,
			description          VARCHAR,
			channel_id           VARCHAR,
			channel_name         VARCHAR,
			video_count          INTEGER,
			view_count_text      VARCHAR,
			last_updated_text    VARCHAR,
			url                  VARCHAR,
			fetched_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS playlist_videos (
			playlist_id VARCHAR NOT NULL,
			video_id    VARCHAR NOT NULL,
			position    INTEGER,
			PRIMARY KEY (playlist_id, video_id)
		)`,
		`CREATE TABLE IF NOT EXISTS related_videos (
			video_id         VARCHAR NOT NULL,
			related_video_id VARCHAR NOT NULL,
			position         INTEGER,
			PRIMARY KEY (video_id, related_video_id)
		)`,
		`CREATE TABLE IF NOT EXISTS caption_tracks (
			video_id           VARCHAR NOT NULL,
			language_code      VARCHAR NOT NULL,
			name               VARCHAR,
			base_url           VARCHAR,
			kind               VARCHAR,
			is_auto_generated  BOOLEAN DEFAULT FALSE,
			fetched_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (video_id, language_code)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

func (d *DB) UpsertVideo(v Video) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO videos (
		video_id, title, description, channel_id, channel_name,
		duration_seconds, duration_text, view_count, comment_count, like_count,
		published_text, published_at, upload_date, is_live, is_short,
		category, tags, thumbnail_url, url, embed_url, transcript, transcript_language, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.VideoID, nullStr(v.Title), nullStr(v.Description), nullStr(v.ChannelID), nullStr(v.ChannelName),
		nullInt(v.DurationSeconds), nullStr(v.DurationText), v.ViewCount, v.CommentCount, v.LikeCount,
		nullStr(v.PublishedText), nullTime(v.PublishedAt), nullStr(v.UploadDate), v.IsLive, v.IsShort,
		nullStr(v.Category), encodeStringSlice(v.Tags), nullStr(v.ThumbnailURL), nullStr(v.URL),
		nullStr(v.EmbedURL), nullStr(v.Transcript), nullStr(v.TranscriptLanguage), v.FetchedAt,
	)
	return err
}

func (d *DB) UpsertChannel(c Channel) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO channels (
		channel_id, handle, title, description, avatar_url, banner_url,
		subscribers_text, videos_text, views_text, country, joined_date_text,
		uploads_playlist_id, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ChannelID, nullStr(c.Handle), nullStr(c.Title), nullStr(c.Description), nullStr(c.AvatarURL),
		nullStr(c.BannerURL), nullStr(c.SubscribersText), nullStr(c.VideosText), nullStr(c.ViewsText),
		nullStr(c.Country), nullStr(c.JoinedDateText), nullStr(c.UploadsPlaylistID), nullStr(c.URL), c.FetchedAt,
	)
	return err
}

func (d *DB) UpsertPlaylist(p Playlist) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO playlists (
		playlist_id, title, description, channel_id, channel_name,
		video_count, view_count_text, last_updated_text, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		p.PlaylistID, nullStr(p.Title), nullStr(p.Description), nullStr(p.ChannelID), nullStr(p.ChannelName),
		p.VideoCount, nullStr(p.ViewCountText), nullStr(p.LastUpdatedText), nullStr(p.URL), p.FetchedAt,
	)
	return err
}

func (d *DB) InsertPlaylistVideos(items []PlaylistVideo) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, item := range items {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO playlist_videos (playlist_id, video_id, position) VALUES (?,?,?)`, item.PlaylistID, item.VideoID, item.Position); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) InsertRelatedVideos(items []RelatedVideo) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, item := range items {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO related_videos (video_id, related_video_id, position) VALUES (?,?,?)`, item.VideoID, item.RelatedVideoID, item.Position); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) InsertCaptionTracks(items []CaptionTrack) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, item := range items {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO caption_tracks (video_id, language_code, name, base_url, kind, is_auto_generated, fetched_at) VALUES (?,?,?,?,?,?,?)`,
			item.VideoID, item.LanguageCode, nullStr(item.Name), nullStr(item.BaseURL), nullStr(item.Kind), item.IsAutoGenerated, item.FetchedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) GetStats() (Stats, error) {
	var s Stats
	d.db.QueryRow(`SELECT COUNT(*) FROM videos`).Scan(&s.Videos)
	d.db.QueryRow(`SELECT COUNT(*) FROM channels`).Scan(&s.Channels)
	d.db.QueryRow(`SELECT COUNT(*) FROM playlists`).Scan(&s.Playlists)
	d.db.QueryRow(`SELECT COUNT(*) FROM playlist_videos`).Scan(&s.PlaylistRows)
	d.db.QueryRow(`SELECT COUNT(*) FROM related_videos`).Scan(&s.RelatedRows)
	d.db.QueryRow(`SELECT COUNT(*) FROM caption_tracks`).Scan(&s.CaptionRows)
	if fi, err := os.Stat(d.path); err == nil {
		s.DBSize = fi.Size()
	}
	return s, nil
}

func (d *DB) RecentVideos(limit int) ([]Video, error) {
	rows, err := d.db.Query(`SELECT video_id, title, channel_name, view_count, fetched_at FROM videos ORDER BY fetched_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Video
	for rows.Next() {
		var v Video
		if err := rows.Scan(&v.VideoID, &v.Title, &v.ChannelName, &v.ViewCount, &v.FetchedAt); err != nil {
			return out, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (d *DB) Path() string { return d.path }
func (d *DB) Close() error { return d.db.Close() }

func encodeStringSlice(v []string) string {
	if len(v) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
