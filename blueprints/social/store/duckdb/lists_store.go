package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/social/feature/lists"
)

// ListsStore implements lists.Store.
type ListsStore struct {
	db *sql.DB
}

// NewListsStore creates a new lists store.
func NewListsStore(db *sql.DB) *ListsStore {
	return &ListsStore{db: db}
}

// Insert inserts a new list.
func (s *ListsStore) Insert(ctx context.Context, l *lists.List) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO lists (id, account_id, title, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, l.ID, l.AccountID, l.Title, l.CreatedAt, l.UpdatedAt)
	return err
}

// GetByID retrieves a list by ID.
func (s *ListsStore) GetByID(ctx context.Context, id string) (*lists.List, error) {
	var l lists.List
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, title, created_at, updated_at
		FROM lists WHERE id = $1
	`, id).Scan(&l.ID, &l.AccountID, &l.Title, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// GetByAccount retrieves lists owned by an account.
func (s *ListsStore) GetByAccount(ctx context.Context, accountID string) ([]*lists.List, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, title, created_at, updated_at
		FROM lists WHERE account_id = $1
		ORDER BY title ASC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ls []*lists.List
	for rows.Next() {
		var l lists.List
		if err := rows.Scan(&l.ID, &l.AccountID, &l.Title, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		ls = append(ls, &l)
	}
	return ls, rows.Err()
}

// Update updates a list.
func (s *ListsStore) Update(ctx context.Context, id string, in *lists.UpdateIn) error {
	if in.Title != nil {
		_, err := s.db.ExecContext(ctx, "UPDATE lists SET title = $1, updated_at = $2 WHERE id = $3", *in.Title, time.Now(), id)
		return err
	}
	return nil
}

// Delete deletes a list.
func (s *ListsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM list_members WHERE list_id = $1", id)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "DELETE FROM lists WHERE id = $1", id)
	return err
}

// InsertMember inserts a list member.
func (s *ListsStore) InsertMember(ctx context.Context, m *lists.ListMember) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO list_members (list_id, account_id, created_at)
		VALUES ($1, $2, $3)
	`, m.ListID, m.AccountID, m.CreatedAt)
	return err
}

// DeleteMember removes a list member.
func (s *ListsStore) DeleteMember(ctx context.Context, listID, accountID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM list_members WHERE list_id = $1 AND account_id = $2", listID, accountID)
	return err
}

// GetMembers returns members of a list.
func (s *ListsStore) GetMembers(ctx context.Context, listID string, limit, offset int) ([]*lists.ListMember, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT list_id, account_id, created_at
		FROM list_members WHERE list_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, listID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*lists.ListMember
	for rows.Next() {
		var m lists.ListMember
		if err := rows.Scan(&m.ListID, &m.AccountID, &m.CreatedAt); err != nil {
			return nil, err
		}
		members = append(members, &m)
	}
	return members, rows.Err()
}

// ExistsMember checks if a member exists.
func (s *ListsStore) ExistsMember(ctx context.Context, listID, accountID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM list_members WHERE list_id = $1 AND account_id = $2)", listID, accountID).Scan(&exists)
	return exists, err
}

// GetMemberCount returns the member count.
func (s *ListsStore) GetMemberCount(ctx context.Context, listID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM list_members WHERE list_id = $1", listID).Scan(&count)
	return count, err
}

// GetListsContaining returns lists containing a specific account.
func (s *ListsStore) GetListsContaining(ctx context.Context, targetID string) ([]*lists.List, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT l.id, l.account_id, l.title, l.created_at, l.updated_at
		FROM lists l
		JOIN list_members lm ON l.id = lm.list_id
		WHERE lm.account_id = $1
		ORDER BY l.title ASC
	`, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ls []*lists.List
	for rows.Next() {
		var l lists.List
		if err := rows.Scan(&l.ID, &l.AccountID, &l.Title, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		ls = append(ls, &l)
	}
	return ls, rows.Err()
}
