package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
)

// AccountsStore implements accounts.Store using DuckDB.
type AccountsStore struct {
	db *sql.DB
}

// NewAccountsStore creates a new accounts store.
func NewAccountsStore(db *sql.DB) *AccountsStore {
	return &AccountsStore{db: db}
}

func (s *AccountsStore) Insert(ctx context.Context, a *accounts.Account, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (id, username, display_name, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, a.ID, a.Username, a.DisplayName, a.Email, passwordHash, a.CreatedAt, a.UpdatedAt)
	return err
}

func (s *AccountsStore) GetByID(ctx context.Context, id string) (*accounts.Account, error) {
	return s.scanAccount(s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, fields,
		       verified, admin, suspended, created_at, updated_at
		FROM accounts WHERE id = $1
	`, id))
}

func (s *AccountsStore) GetByUsername(ctx context.Context, username string) (*accounts.Account, error) {
	return s.scanAccount(s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, fields,
		       verified, admin, suspended, created_at, updated_at
		FROM accounts WHERE LOWER(username) = LOWER($1)
	`, username))
}

func (s *AccountsStore) GetByEmail(ctx context.Context, email string) (*accounts.Account, error) {
	return s.scanAccount(s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, fields,
		       verified, admin, suspended, created_at, updated_at
		FROM accounts WHERE LOWER(email) = LOWER($1)
	`, email))
}

func (s *AccountsStore) scanAccount(row *sql.Row) (*accounts.Account, error) {
	var a accounts.Account
	var bio, avatarURL, headerURL, fieldsJSON sql.NullString
	var email sql.NullString

	err := row.Scan(
		&a.ID, &a.Username, &a.DisplayName, &email,
		&bio, &avatarURL, &headerURL, &fieldsJSON,
		&a.Verified, &a.Admin, &a.Suspended, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("account not found")
		}
		return nil, err
	}

	a.Email = email.String
	a.Bio = bio.String
	a.AvatarURL = avatarURL.String
	a.HeaderURL = headerURL.String

	if fieldsJSON.Valid && fieldsJSON.String != "" {
		_ = json.Unmarshal([]byte(fieldsJSON.String), &a.Fields)
	}

	return &a, nil
}

func (s *AccountsStore) Update(ctx context.Context, id string, in *accounts.UpdateIn) error {
	var sets []string
	var args []any
	argIdx := 1

	if in.DisplayName != nil {
		sets = append(sets, fmt.Sprintf("display_name = $%d", argIdx))
		args = append(args, *in.DisplayName)
		argIdx++
	}
	if in.Bio != nil {
		sets = append(sets, fmt.Sprintf("bio = $%d", argIdx))
		args = append(args, *in.Bio)
		argIdx++
	}
	if in.AvatarURL != nil {
		sets = append(sets, fmt.Sprintf("avatar_url = $%d", argIdx))
		args = append(args, *in.AvatarURL)
		argIdx++
	}
	if in.HeaderURL != nil {
		sets = append(sets, fmt.Sprintf("header_url = $%d", argIdx))
		args = append(args, *in.HeaderURL)
		argIdx++
	}
	if in.Fields != nil {
		fieldsJSON, _ := json.Marshal(in.Fields)
		sets = append(sets, fmt.Sprintf("fields = $%d", argIdx))
		args = append(args, string(fieldsJSON))
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, id)
	query := fmt.Sprintf("UPDATE accounts SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *AccountsStore) ExistsUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM accounts WHERE LOWER(username) = LOWER($1))", username).Scan(&exists)
	return exists, err
}

func (s *AccountsStore) ExistsEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM accounts WHERE LOWER(email) = LOWER($1))", email).Scan(&exists)
	return exists, err
}

func (s *AccountsStore) GetPasswordHash(ctx context.Context, usernameOrEmail string) (id, hash string, suspended bool, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT id, password_hash, suspended
		FROM accounts WHERE LOWER(username) = LOWER($1) OR LOWER(email) = LOWER($1)
	`, usernameOrEmail).Scan(&id, &hash, &suspended)
	return
}

func (s *AccountsStore) List(ctx context.Context, limit, offset int) ([]*accounts.Account, int, error) {
	var total int
	err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM accounts WHERE suspended = FALSE").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, display_name, email, bio, avatar_url, header_url, fields,
		       verified, admin, suspended, created_at, updated_at
		FROM accounts WHERE suspended = FALSE
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*accounts.Account
	for rows.Next() {
		var a accounts.Account
		var bio, avatarURL, headerURL, fieldsJSON, email sql.NullString

		if err := rows.Scan(
			&a.ID, &a.Username, &a.DisplayName, &email,
			&bio, &avatarURL, &headerURL, &fieldsJSON,
			&a.Verified, &a.Admin, &a.Suspended, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			continue
		}

		a.Email = email.String
		a.Bio = bio.String
		a.AvatarURL = avatarURL.String
		a.HeaderURL = headerURL.String

		if fieldsJSON.Valid && fieldsJSON.String != "" {
			_ = json.Unmarshal([]byte(fieldsJSON.String), &a.Fields)
		}

		result = append(result, &a)
	}

	return result, total, nil
}

func (s *AccountsStore) Search(ctx context.Context, query string, limit int) ([]*accounts.Account, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, display_name, bio, avatar_url, verified, created_at
		FROM accounts
		WHERE suspended = FALSE AND (
			LOWER(username) LIKE LOWER($1) || '%'
			OR LOWER(display_name) LIKE '%' || LOWER($1) || '%'
		)
		ORDER BY
			CASE WHEN LOWER(username) = LOWER($1) THEN 0 ELSE 1 END,
			CASE WHEN LOWER(username) LIKE LOWER($1) || '%' THEN 0 ELSE 1 END,
			LENGTH(username)
		LIMIT $2
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*accounts.Account
	for rows.Next() {
		var a accounts.Account
		var bio, avatarURL sql.NullString

		if err := rows.Scan(&a.ID, &a.Username, &a.DisplayName, &bio, &avatarURL, &a.Verified, &a.CreatedAt); err != nil {
			continue
		}

		a.Bio = bio.String
		a.AvatarURL = avatarURL.String
		result = append(result, &a)
	}

	return result, nil
}

func (s *AccountsStore) SetVerified(ctx context.Context, id string, verified bool) error {
	_, err := s.db.ExecContext(ctx, "UPDATE accounts SET verified = $1, updated_at = $2 WHERE id = $3",
		verified, time.Now(), id)
	return err
}

func (s *AccountsStore) SetSuspended(ctx context.Context, id string, suspended bool) error {
	_, err := s.db.ExecContext(ctx, "UPDATE accounts SET suspended = $1, updated_at = $2 WHERE id = $3",
		suspended, time.Now(), id)
	return err
}

func (s *AccountsStore) SetAdmin(ctx context.Context, id string, admin bool) error {
	_, err := s.db.ExecContext(ctx, "UPDATE accounts SET admin = $1, updated_at = $2 WHERE id = $3",
		admin, time.Now(), id)
	return err
}

func (s *AccountsStore) CreateSession(ctx context.Context, sess *accounts.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, account_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, sess.ID, sess.AccountID, sess.Token, sess.ExpiresAt, sess.CreatedAt)
	return err
}

func (s *AccountsStore) GetSession(ctx context.Context, token string) (*accounts.Session, error) {
	var sess accounts.Session
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, token, expires_at, created_at
		FROM sessions WHERE token = $1 AND expires_at > CURRENT_TIMESTAMP
	`, token).Scan(&sess.ID, &sess.AccountID, &sess.Token, &sess.ExpiresAt, &sess.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *AccountsStore) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token = $1", token)
	return err
}
