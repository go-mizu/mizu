package duckdb

import (
	"context"
	"database/sql"
)

// Usermeta represents a user meta entry.
type Usermeta struct {
	UmetaID   string
	UserID    string
	MetaKey   string
	MetaValue string
}

// UsermetaStore handles user meta persistence.
type UsermetaStore struct {
	db *sql.DB
}

// NewUsermetaStore creates a new usermeta store.
func NewUsermetaStore(db *sql.DB) *UsermetaStore {
	return &UsermetaStore{db: db}
}

// Create creates a new user meta entry.
func (s *UsermetaStore) Create(ctx context.Context, m *Usermeta) error {
	query := `INSERT INTO wp_usermeta (umeta_id, user_id, meta_key, meta_value) VALUES ($1, $2, $3, $4)`
	_, err := s.db.ExecContext(ctx, query, m.UmetaID, m.UserID, m.MetaKey, m.MetaValue)
	return err
}

// Get retrieves a user meta value.
func (s *UsermetaStore) Get(ctx context.Context, userID, metaKey string) (string, error) {
	query := `SELECT meta_value FROM wp_usermeta WHERE user_id = $1 AND meta_key = $2 LIMIT 1`
	var value string
	err := s.db.QueryRowContext(ctx, query, userID, metaKey).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// GetAll retrieves all meta for a user.
func (s *UsermetaStore) GetAll(ctx context.Context, userID string) (map[string]string, error) {
	query := `SELECT meta_key, meta_value FROM wp_usermeta WHERE user_id = $1`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	meta := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		meta[key] = value
	}
	return meta, rows.Err()
}

// Update updates a user meta value.
func (s *UsermetaStore) Update(ctx context.Context, userID, metaKey, metaValue string) error {
	query := `UPDATE wp_usermeta SET meta_value = $3 WHERE user_id = $1 AND meta_key = $2`
	_, err := s.db.ExecContext(ctx, query, userID, metaKey, metaValue)
	return err
}

// Delete deletes a user meta entry.
func (s *UsermetaStore) Delete(ctx context.Context, userID, metaKey string) error {
	query := `DELETE FROM wp_usermeta WHERE user_id = $1 AND meta_key = $2`
	_, err := s.db.ExecContext(ctx, query, userID, metaKey)
	return err
}

// DeleteAllForUser deletes all meta for a user.
func (s *UsermetaStore) DeleteAllForUser(ctx context.Context, userID string) error {
	query := `DELETE FROM wp_usermeta WHERE user_id = $1`
	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}

// Postmeta represents a post meta entry.
type Postmeta struct {
	MetaID    string
	PostID    string
	MetaKey   string
	MetaValue string
}

// PostmetaStore handles post meta persistence.
type PostmetaStore struct {
	db *sql.DB
}

// NewPostmetaStore creates a new postmeta store.
func NewPostmetaStore(db *sql.DB) *PostmetaStore {
	return &PostmetaStore{db: db}
}

// Create creates a new post meta entry.
func (s *PostmetaStore) Create(ctx context.Context, m *Postmeta) error {
	query := `INSERT INTO wp_postmeta (meta_id, post_id, meta_key, meta_value) VALUES ($1, $2, $3, $4)`
	_, err := s.db.ExecContext(ctx, query, m.MetaID, m.PostID, m.MetaKey, m.MetaValue)
	return err
}

// Get retrieves a post meta value.
func (s *PostmetaStore) Get(ctx context.Context, postID, metaKey string) (string, error) {
	query := `SELECT meta_value FROM wp_postmeta WHERE post_id = $1 AND meta_key = $2 LIMIT 1`
	var value string
	err := s.db.QueryRowContext(ctx, query, postID, metaKey).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// GetAll retrieves all meta for a post.
func (s *PostmetaStore) GetAll(ctx context.Context, postID string) (map[string]string, error) {
	query := `SELECT meta_key, meta_value FROM wp_postmeta WHERE post_id = $1`
	rows, err := s.db.QueryContext(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	meta := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		meta[key] = value
	}
	return meta, rows.Err()
}

// Update updates a post meta value.
func (s *PostmetaStore) Update(ctx context.Context, postID, metaKey, metaValue string) error {
	query := `UPDATE wp_postmeta SET meta_value = $3 WHERE post_id = $1 AND meta_key = $2`
	_, err := s.db.ExecContext(ctx, query, postID, metaKey, metaValue)
	return err
}

// Delete deletes a post meta entry.
func (s *PostmetaStore) Delete(ctx context.Context, postID, metaKey string) error {
	query := `DELETE FROM wp_postmeta WHERE post_id = $1 AND meta_key = $2`
	_, err := s.db.ExecContext(ctx, query, postID, metaKey)
	return err
}

// DeleteAllForPost deletes all meta for a post.
func (s *PostmetaStore) DeleteAllForPost(ctx context.Context, postID string) error {
	query := `DELETE FROM wp_postmeta WHERE post_id = $1`
	_, err := s.db.ExecContext(ctx, query, postID)
	return err
}

// Termmeta represents a term meta entry.
type Termmeta struct {
	MetaID    string
	TermID    string
	MetaKey   string
	MetaValue string
}

// TermmetaStore handles term meta persistence.
type TermmetaStore struct {
	db *sql.DB
}

// NewTermmetaStore creates a new termmeta store.
func NewTermmetaStore(db *sql.DB) *TermmetaStore {
	return &TermmetaStore{db: db}
}

// Create creates a new term meta entry.
func (s *TermmetaStore) Create(ctx context.Context, m *Termmeta) error {
	query := `INSERT INTO wp_termmeta (meta_id, term_id, meta_key, meta_value) VALUES ($1, $2, $3, $4)`
	_, err := s.db.ExecContext(ctx, query, m.MetaID, m.TermID, m.MetaKey, m.MetaValue)
	return err
}

// Get retrieves a term meta value.
func (s *TermmetaStore) Get(ctx context.Context, termID, metaKey string) (string, error) {
	query := `SELECT meta_value FROM wp_termmeta WHERE term_id = $1 AND meta_key = $2 LIMIT 1`
	var value string
	err := s.db.QueryRowContext(ctx, query, termID, metaKey).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// GetAll retrieves all meta for a term.
func (s *TermmetaStore) GetAll(ctx context.Context, termID string) (map[string]string, error) {
	query := `SELECT meta_key, meta_value FROM wp_termmeta WHERE term_id = $1`
	rows, err := s.db.QueryContext(ctx, query, termID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	meta := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		meta[key] = value
	}
	return meta, rows.Err()
}

// Update updates a term meta value.
func (s *TermmetaStore) Update(ctx context.Context, termID, metaKey, metaValue string) error {
	query := `UPDATE wp_termmeta SET meta_value = $3 WHERE term_id = $1 AND meta_key = $2`
	_, err := s.db.ExecContext(ctx, query, termID, metaKey, metaValue)
	return err
}

// Delete deletes a term meta entry.
func (s *TermmetaStore) Delete(ctx context.Context, termID, metaKey string) error {
	query := `DELETE FROM wp_termmeta WHERE term_id = $1 AND meta_key = $2`
	_, err := s.db.ExecContext(ctx, query, termID, metaKey)
	return err
}

// Commentmeta represents a comment meta entry.
type Commentmeta struct {
	MetaID    string
	CommentID string
	MetaKey   string
	MetaValue string
}

// CommentmetaStore handles comment meta persistence.
type CommentmetaStore struct {
	db *sql.DB
}

// NewCommentmetaStore creates a new commentmeta store.
func NewCommentmetaStore(db *sql.DB) *CommentmetaStore {
	return &CommentmetaStore{db: db}
}

// Create creates a new comment meta entry.
func (s *CommentmetaStore) Create(ctx context.Context, m *Commentmeta) error {
	query := `INSERT INTO wp_commentmeta (meta_id, comment_id, meta_key, meta_value) VALUES ($1, $2, $3, $4)`
	_, err := s.db.ExecContext(ctx, query, m.MetaID, m.CommentID, m.MetaKey, m.MetaValue)
	return err
}

// Get retrieves a comment meta value.
func (s *CommentmetaStore) Get(ctx context.Context, commentID, metaKey string) (string, error) {
	query := `SELECT meta_value FROM wp_commentmeta WHERE comment_id = $1 AND meta_key = $2 LIMIT 1`
	var value string
	err := s.db.QueryRowContext(ctx, query, commentID, metaKey).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// GetAll retrieves all meta for a comment.
func (s *CommentmetaStore) GetAll(ctx context.Context, commentID string) (map[string]string, error) {
	query := `SELECT meta_key, meta_value FROM wp_commentmeta WHERE comment_id = $1`
	rows, err := s.db.QueryContext(ctx, query, commentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	meta := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		meta[key] = value
	}
	return meta, rows.Err()
}

// Update updates a comment meta value.
func (s *CommentmetaStore) Update(ctx context.Context, commentID, metaKey, metaValue string) error {
	query := `UPDATE wp_commentmeta SET meta_value = $3 WHERE comment_id = $1 AND meta_key = $2`
	_, err := s.db.ExecContext(ctx, query, commentID, metaKey, metaValue)
	return err
}

// Delete deletes a comment meta entry.
func (s *CommentmetaStore) Delete(ctx context.Context, commentID, metaKey string) error {
	query := `DELETE FROM wp_commentmeta WHERE comment_id = $1 AND meta_key = $2`
	_, err := s.db.ExecContext(ctx, query, commentID, metaKey)
	return err
}
