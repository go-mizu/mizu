package discord

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DB is the main data store for Discord entities.
type DB struct {
	db   *sql.DB
	path string
}

// OpenDB opens (or creates) the Discord DuckDB database.
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
		`CREATE TABLE IF NOT EXISTS guilds (
			guild_id                        VARCHAR PRIMARY KEY,
			name                            VARCHAR,
			description                     VARCHAR,
			icon_url                        VARCHAR,
			member_count                    BIGINT,
			approximate_presence_count      BIGINT,
			owner_id                        VARCHAR,
			features_json                   VARCHAR,
			fetched_at                      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS channels (
			channel_id      VARCHAR PRIMARY KEY,
			guild_id        VARCHAR,
			name            VARCHAR,
			channel_type    INTEGER,
			topic           VARCHAR,
			position        INTEGER,
			parent_id       VARCHAR,
			nsfw            BOOLEAN,
			last_message_id VARCHAR,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_channels_guild ON channels(guild_id)`,
		`CREATE TABLE IF NOT EXISTS messages (
			message_id              VARCHAR PRIMARY KEY,
			channel_id              VARCHAR,
			guild_id                VARCHAR,
			author_id               VARCHAR,
			author_username         VARCHAR,
			content                 VARCHAR,
			timestamp               TIMESTAMP,
			edited_timestamp        TIMESTAMP,
			message_type            INTEGER,
			pinned                  BOOLEAN,
			mention_everyone        BOOLEAN,
			attachments_json        VARCHAR,
			embeds_json             VARCHAR,
			reactions_json          VARCHAR,
			referenced_message_id   VARCHAR,
			fetched_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_channel ON messages(channel_id, message_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_author  ON messages(author_id)`,
		`CREATE TABLE IF NOT EXISTS users (
			user_id         VARCHAR PRIMARY KEY,
			username        VARCHAR,
			global_name     VARCHAR,
			discriminator   VARCHAR,
			avatar_url      VARCHAR,
			bot             BOOLEAN,
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

func (d *DB) UpsertGuild(g Guild) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO guilds (
		guild_id, name, description, icon_url, member_count,
		approximate_presence_count, owner_id, features_json, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?)`,
		g.GuildID, nullStr(g.Name), nullStr(g.Description), nullStr(g.IconURL),
		nullInt64(g.MemberCount), nullInt64(g.ApproximatePresenceCount),
		nullStr(g.OwnerID), nullStr(g.FeaturesJSON), nullTime(g.FetchedAt),
	)
	return err
}

func (d *DB) UpsertChannel(c Channel) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO channels (
		channel_id, guild_id, name, channel_type, topic,
		position, parent_id, nsfw, last_message_id, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		c.ChannelID, nullStr(c.GuildID), nullStr(c.Name), c.ChannelType,
		nullStr(c.Topic), c.Position, nullStr(c.ParentID),
		c.NSFW, nullStr(c.LastMessageID), nullTime(c.FetchedAt),
	)
	return err
}

func (d *DB) UpsertMessage(m Message) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO messages (
		message_id, channel_id, guild_id, author_id, author_username,
		content, timestamp, edited_timestamp, message_type, pinned,
		mention_everyone, attachments_json, embeds_json, reactions_json,
		referenced_message_id, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		m.MessageID, nullStr(m.ChannelID), nullStr(m.GuildID),
		nullStr(m.AuthorID), nullStr(m.AuthorUsername),
		nullStr(m.Content), nullTime(m.Timestamp), nullTime(m.EditedTimestamp),
		m.MessageType, m.Pinned, m.MentionEveryone,
		nullStr(m.AttachmentsJSON), nullStr(m.EmbedsJSON), nullStr(m.ReactionsJSON),
		nullStr(m.ReferencedMessageID), nullTime(m.FetchedAt),
	)
	return err
}

func (d *DB) UpsertUser(u User) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO users (
		user_id, username, global_name, discriminator, avatar_url, bot, fetched_at
	) VALUES (?,?,?,?,?,?,?)`,
		u.UserID, nullStr(u.Username), nullStr(u.GlobalName),
		nullStr(u.Discriminator), nullStr(u.AvatarURL), u.Bot, nullTime(u.FetchedAt),
	)
	return err
}

func (d *DB) GetStats() (DBStats, error) {
	var s DBStats
	d.db.QueryRow(`SELECT COUNT(*) FROM guilds`).Scan(&s.Guilds)
	d.db.QueryRow(`SELECT COUNT(*) FROM channels`).Scan(&s.Channels)
	d.db.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&s.Messages)
	d.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&s.Users)
	if fi, err := os.Stat(d.path); err == nil {
		s.DBSize = fi.Size()
	}
	return s, nil
}

func (d *DB) RecentMessages(limit int) ([]Message, error) {
	rows, err := d.db.Query(`
		SELECT message_id, channel_id, author_username, content
		FROM messages ORDER BY fetched_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		var channelID, authorUsername, content sql.NullString
		if err := rows.Scan(&m.MessageID, &channelID, &authorUsername, &content); err != nil {
			return out, err
		}
		m.ChannelID = channelID.String
		m.AuthorUsername = authorUsername.String
		m.Content = content.String
		out = append(out, m)
	}
	return out, rows.Err()
}

func (d *DB) Close() error { return d.db.Close() }
func (d *DB) Path() string  { return d.path }

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullInt64(i int64) any {
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
