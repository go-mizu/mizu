package spotify

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
		`CREATE TABLE IF NOT EXISTS tracks (
			track_id            VARCHAR PRIMARY KEY,
			name                VARCHAR,
			duration_ms         BIGINT,
			track_number        INTEGER,
			disc_number         INTEGER,
			playable            BOOLEAN,
			preview_url         VARCHAR,
			playcount           BIGINT,
			album_id            VARCHAR,
			album_name          VARCHAR,
			cover_url           VARCHAR,
			release_date        VARCHAR,
			url                 VARCHAR,
			spotify_uri         VARCHAR,
			source_title        VARCHAR,
			source_description  VARCHAR,
			fetched_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS albums (
			album_id            VARCHAR PRIMARY KEY,
			name                VARCHAR,
			album_type          VARCHAR,
			release_date        VARCHAR,
			total_tracks        INTEGER,
			cover_url           VARCHAR,
			copyright_text      VARCHAR,
			url                 VARCHAR,
			spotify_uri         VARCHAR,
			source_title        VARCHAR,
			source_description  VARCHAR,
			fetched_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS artists (
			artist_id           VARCHAR PRIMARY KEY,
			name                VARCHAR,
			biography           VARCHAR,
			followers           BIGINT,
			monthly_listeners   BIGINT,
			avatar_url          VARCHAR,
			external_links_json VARCHAR,
			url                 VARCHAR,
			spotify_uri         VARCHAR,
			source_title        VARCHAR,
			source_description  VARCHAR,
			fetched_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS playlists (
			playlist_id         VARCHAR PRIMARY KEY,
			name                VARCHAR,
			description         VARCHAR,
			followers           BIGINT,
			owner_name          VARCHAR,
			owner_username      VARCHAR,
			owner_uri           VARCHAR,
			image_url           VARCHAR,
			total_items         INTEGER,
			next_offset         INTEGER,
			url                 VARCHAR,
			spotify_uri         VARCHAR,
			source_title        VARCHAR,
			source_description  VARCHAR,
			fetched_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS track_artists (
			track_id            VARCHAR NOT NULL,
			artist_id           VARCHAR NOT NULL,
			artist_name         VARCHAR,
			ord                 INTEGER,
			PRIMARY KEY (track_id, artist_id)
		)`,
		`CREATE TABLE IF NOT EXISTS album_artists (
			album_id            VARCHAR NOT NULL,
			artist_id           VARCHAR NOT NULL,
			artist_name         VARCHAR,
			ord                 INTEGER,
			PRIMARY KEY (album_id, artist_id)
		)`,
		`CREATE TABLE IF NOT EXISTS album_tracks (
			album_id            VARCHAR NOT NULL,
			track_id            VARCHAR NOT NULL,
			ord                 INTEGER,
			PRIMARY KEY (album_id, track_id)
		)`,
		`CREATE TABLE IF NOT EXISTS playlist_tracks (
			playlist_id         VARCHAR NOT NULL,
			track_id            VARCHAR NOT NULL,
			ord                 INTEGER,
			added_by            VARCHAR,
			PRIMARY KEY (playlist_id, track_id)
		)`,
		`CREATE TABLE IF NOT EXISTS artist_related (
			artist_id           VARCHAR NOT NULL,
			related_artist_id   VARCHAR NOT NULL,
			related_name        VARCHAR,
			ord                 INTEGER,
			PRIMARY KEY (artist_id, related_artist_id)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

func (d *DB) UpsertTrack(t Track) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO tracks (
		track_id, name, duration_ms, track_number, disc_number, playable,
		preview_url, playcount, album_id, album_name, cover_url, release_date,
		url, spotify_uri, source_title, source_description, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.TrackID, nullStr(t.Name), nullInt64(t.DurationMS), nullInt(t.TrackNumber), nullInt(t.DiscNumber), t.Playable,
		nullStr(t.PreviewURL), nullInt64(t.Playcount), nullStr(t.AlbumID), nullStr(t.AlbumName), nullStr(t.CoverURL), nullStr(t.ReleaseDate),
		nullStr(t.URL), nullStr(t.SpotifyURI), nullStr(t.SourceTitle), nullStr(t.SourceDescription), nullTime(t.FetchedAt),
	)
	return err
}

func (d *DB) UpsertAlbum(a Album) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO albums (
		album_id, name, album_type, release_date, total_tracks, cover_url, copyright_text,
		url, spotify_uri, source_title, source_description, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.AlbumID, nullStr(a.Name), nullStr(a.AlbumType), nullStr(a.ReleaseDate), nullInt(a.TotalTracks), nullStr(a.CoverURL), nullStr(a.CopyrightText),
		nullStr(a.URL), nullStr(a.SpotifyURI), nullStr(a.SourceTitle), nullStr(a.SourceDescription), nullTime(a.FetchedAt),
	)
	return err
}

func (d *DB) UpsertArtist(a Artist) error {
	linksJSON, _ := json.Marshal(a.ExternalLinks)
	_, err := d.db.Exec(`INSERT OR REPLACE INTO artists (
		artist_id, name, biography, followers, monthly_listeners, avatar_url,
		external_links_json, url, spotify_uri, source_title, source_description, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.ArtistID, nullStr(a.Name), nullStr(a.Biography), nullInt64(a.Followers), nullInt64(a.MonthlyListeners), nullStr(a.AvatarURL),
		nullBytes(linksJSON), nullStr(a.URL), nullStr(a.SpotifyURI), nullStr(a.SourceTitle), nullStr(a.SourceDescription), nullTime(a.FetchedAt),
	)
	return err
}

func (d *DB) UpsertPlaylist(p Playlist) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO playlists (
		playlist_id, name, description, followers, owner_name, owner_username, owner_uri,
		image_url, total_items, next_offset, url, spotify_uri, source_title, source_description, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.PlaylistID, nullStr(p.Name), nullStr(p.Description), nullInt64(p.Followers), nullStr(p.OwnerName), nullStr(p.OwnerUsername), nullStr(p.OwnerURI),
		nullStr(p.ImageURL), nullInt(p.TotalItems), nullInt(p.NextOffset), nullStr(p.URL), nullStr(p.SpotifyURI), nullStr(p.SourceTitle), nullStr(p.SourceDescription), nullTime(p.FetchedAt),
	)
	return err
}

func (d *DB) UpsertTrackArtist(rel TrackArtist) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO track_artists (track_id, artist_id, artist_name, ord) VALUES (?,?,?,?)`,
		rel.TrackID, rel.ArtistID, nullStr(rel.ArtistName), rel.Ord)
	return err
}

func (d *DB) UpsertAlbumArtist(rel AlbumArtist) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO album_artists (album_id, artist_id, artist_name, ord) VALUES (?,?,?,?)`,
		rel.AlbumID, rel.ArtistID, nullStr(rel.ArtistName), rel.Ord)
	return err
}

func (d *DB) UpsertAlbumTrack(rel AlbumTrack) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO album_tracks (album_id, track_id, ord) VALUES (?,?,?)`,
		rel.AlbumID, rel.TrackID, rel.Ord)
	return err
}

func (d *DB) UpsertPlaylistTrack(rel PlaylistTrack) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO playlist_tracks (playlist_id, track_id, ord, added_by) VALUES (?,?,?,?)`,
		rel.PlaylistID, rel.TrackID, rel.Ord, nullStr(rel.AddedBy))
	return err
}

func (d *DB) UpsertArtistRelated(rel ArtistRelated) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO artist_related (artist_id, related_artist_id, related_name, ord) VALUES (?,?,?,?)`,
		rel.ArtistID, rel.RelatedArtistID, nullStr(rel.RelatedName), rel.Ord)
	return err
}

func (d *DB) GetStats() (DBStats, error) {
	var s DBStats
	d.db.QueryRow(`SELECT COUNT(*) FROM tracks`).Scan(&s.Tracks)
	d.db.QueryRow(`SELECT COUNT(*) FROM albums`).Scan(&s.Albums)
	d.db.QueryRow(`SELECT COUNT(*) FROM artists`).Scan(&s.Artists)
	d.db.QueryRow(`SELECT COUNT(*) FROM playlists`).Scan(&s.Playlists)
	if fi, err := os.Stat(d.path); err == nil {
		s.DBSize = fi.Size()
	}
	return s, nil
}

func (d *DB) RecentTracks(limit int) ([]Track, error) {
	rows, err := d.db.Query(`SELECT track_id, name, album_name, url FROM tracks ORDER BY fetched_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Track
	for rows.Next() {
		var t Track
		var name, albumName, rawURL sql.NullString
		if err := rows.Scan(&t.TrackID, &name, &albumName, &rawURL); err != nil {
			return out, err
		}
		t.Name = name.String
		t.AlbumName = albumName.String
		t.URL = rawURL.String
		out = append(out, t)
	}
	return out, rows.Err()
}

func (d *DB) Close() error { return d.db.Close() }
func (d *DB) Path() string { return d.path }

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

func nullInt64(i int64) any {
	if i == 0 {
		return nil
	}
	return i
}

func nullBytes(b []byte) any {
	if len(b) == 0 || string(b) == "null" || string(b) == "[]" {
		return nil
	}
	return string(b)
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
