package duckdb

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"now/pkg/chat"
)

// DB wraps a DuckDB connection.
type DB struct {
	db *sql.DB
}

// Open opens a DuckDB database at path and runs migrations.
func Open(path string) (*DB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db: db}, nil
}

// Close closes the database.
func (d *DB) Close() error {
	return d.db.Close()
}

// Chats returns a ChatStore backed by this database.
func (d *DB) Chats() *ChatStore { return &ChatStore{db: d.db} }

// Members returns a MemberStore backed by this database.
func (d *DB) Members() *MemberStore { return &MemberStore{db: d.db} }

// Messages returns a MessageStore backed by this database.
func (d *DB) Messages() *MessageStore { return &MessageStore{db: d.db} }

// Keys returns a KeyStore backed by this database.
func (d *DB) Keys() *KeyStore { return &KeyStore{db: d.db} }

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS keys (
			actor       TEXT PRIMARY KEY,
			public_key  BLOB NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS chats (
			id          TEXT PRIMARY KEY,
			kind        TEXT NOT NULL,
			title       TEXT,
			creator     TEXT,
			fingerprint TEXT,
			created_at  TIMESTAMPTZ NOT NULL
		);
		CREATE TABLE IF NOT EXISTS members (
			chat  TEXT NOT NULL,
			actor TEXT NOT NULL,
			PRIMARY KEY (chat, actor)
		);
		CREATE TABLE IF NOT EXISTS messages (
			id          TEXT PRIMARY KEY,
			chat        TEXT NOT NULL,
			actor       TEXT NOT NULL,
			fingerprint TEXT NOT NULL,
			text        TEXT NOT NULL,
			signature   TEXT NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL
		);
	`)
	return err
}

// --- KeyStore ---

// KeyStore implements auth.KeyStore.
type KeyStore struct {
	db *sql.DB
}

// Register binds an actor name to a public key.
func (s *KeyStore) Register(ctx context.Context, actor string, publicKey []byte) error {
	var existing []byte
	err := s.db.QueryRowContext(ctx, "SELECT public_key FROM keys WHERE actor = ?", actor).Scan(&existing)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = s.db.ExecContext(ctx,
			"INSERT INTO keys (actor, public_key) VALUES (?, ?)",
			actor, publicKey)
		return err
	}
	if err != nil {
		return err
	}
	if !bytes.Equal(existing, publicKey) {
		return errors.New("identity conflict: actor already registered with a different key")
	}
	return nil
}

// Lookup returns the public key for an actor.
func (s *KeyStore) Lookup(ctx context.Context, actor string) ([]byte, error) {
	var key []byte
	err := s.db.QueryRowContext(ctx, "SELECT public_key FROM keys WHERE actor = ?", actor).Scan(&key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("actor not found")
	}
	return key, err
}

// --- ChatStore ---

// ChatStore implements chat.ChatStore.
type ChatStore struct {
	db *sql.DB
}

// Create creates a chat.
func (s *ChatStore) Create(ctx context.Context, c chat.Chat) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO chats (id, kind, title, creator, fingerprint, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		c.ID, c.Kind, c.Title, c.Creator, c.Fingerprint, c.Time)
	return err
}

// Get returns a chat by ID.
func (s *ChatStore) Get(ctx context.Context, id string) (chat.Chat, error) {
	var c chat.Chat
	var title, creator, fp sql.NullString
	err := s.db.QueryRowContext(ctx,
		"SELECT id, kind, title, creator, fingerprint, created_at FROM chats WHERE id = ?", id,
	).Scan(&c.ID, &c.Kind, &title, &creator, &fp, &c.Time)
	if errors.Is(err, sql.ErrNoRows) {
		return chat.Chat{}, errors.New("chat not found")
	}
	if err != nil {
		return chat.Chat{}, err
	}
	c.Title = title.String
	c.Creator = creator.String
	c.Fingerprint = fp.String
	return c, nil
}

// List returns chats filtered by kind.
func (s *ChatStore) List(ctx context.Context, in chat.ListInput) (chat.Chats, error) {
	query := "SELECT id, kind, title, creator, fingerprint, created_at FROM chats"
	var args []any

	if in.Kind != "" {
		query += " WHERE kind = ?"
		args = append(args, in.Kind)
	}

	query += " ORDER BY created_at DESC"

	if in.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, in.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return chat.Chats{}, err
	}
	defer rows.Close()

	var items []chat.Chat
	for rows.Next() {
		var c chat.Chat
		var title, creator, fp sql.NullString
		if err := rows.Scan(&c.ID, &c.Kind, &title, &creator, &fp, &c.Time); err != nil {
			return chat.Chats{}, err
		}
		c.Title = title.String
		c.Creator = creator.String
		c.Fingerprint = fp.String
		items = append(items, c)
	}

	return chat.Chats{Items: items}, rows.Err()
}

// --- MemberStore ---

// MemberStore implements chat.MemberStore.
type MemberStore struct {
	db *sql.DB
}

// Join adds an actor to a chat.
func (s *MemberStore) Join(ctx context.Context, chatID string, actor string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO members (chat, actor) VALUES (?, ?)",
		chatID, actor)
	return err
}

// Leave removes an actor from a chat.
func (s *MemberStore) Leave(ctx context.Context, chatID string, actor string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM members WHERE chat = ? AND actor = ?",
		chatID, actor)
	return err
}

// Has reports whether an actor is a member of a chat.
func (s *MemberStore) Has(ctx context.Context, chatID string, actor string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM members WHERE chat = ? AND actor = ?",
		chatID, actor).Scan(&n)
	return n > 0, err
}

// List returns members of a chat.
func (s *MemberStore) List(ctx context.Context, chatID string, limit int) ([]string, error) {
	query := "SELECT actor FROM members WHERE chat = ?"
	var args []any
	args = append(args, chatID)

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actors []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		actors = append(actors, a)
	}

	return actors, rows.Err()
}

// --- MessageStore ---

// MessageStore implements chat.MessageStore.
type MessageStore struct {
	db *sql.DB
}

// Create creates a message.
func (s *MessageStore) Create(ctx context.Context, m chat.Message) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO messages (id, chat, actor, fingerprint, text, signature, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		m.ID, m.Chat, m.Actor, m.Fingerprint, m.Text, m.Signature, m.Time)
	return err
}

// Get returns a message by ID.
func (s *MessageStore) Get(ctx context.Context, id string) (chat.Message, error) {
	var m chat.Message
	err := s.db.QueryRowContext(ctx,
		"SELECT id, chat, actor, fingerprint, text, signature, created_at FROM messages WHERE id = ?", id,
	).Scan(&m.ID, &m.Chat, &m.Actor, &m.Fingerprint, &m.Text, &m.Signature, &m.Time)
	if errors.Is(err, sql.ErrNoRows) {
		return chat.Message{}, errors.New("message not found")
	}
	return m, err
}

// List returns messages for a chat with cursor-based pagination.
func (s *MessageStore) List(ctx context.Context, in chat.MessagesInput) (chat.Messages, error) {
	query := "SELECT id, chat, actor, fingerprint, text, signature, created_at FROM messages WHERE chat = ?"
	args := []any{in.Chat}

	if in.Before != "" {
		query += " AND created_at < (SELECT created_at FROM messages WHERE id = ?)"
		args = append(args, in.Before)
	}

	query += " ORDER BY created_at DESC"

	if in.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, in.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return chat.Messages{}, err
	}
	defer rows.Close()

	var items []chat.Message
	for rows.Next() {
		var m chat.Message
		if err := rows.Scan(&m.ID, &m.Chat, &m.Actor, &m.Fingerprint, &m.Text, &m.Signature, &m.Time); err != nil {
			return chat.Messages{}, err
		}
		items = append(items, m)
	}

	return chat.Messages{Items: items}, rows.Err()
}

// scanTime is a helper to handle DuckDB timestamp scanning.
func scanTime(t *time.Time) *time.Time {
	return t
}
