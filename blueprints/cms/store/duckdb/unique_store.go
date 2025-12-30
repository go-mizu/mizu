package duckdb

import (
	"context"
	"database/sql"
	"fmt"
)

// UniqueStore provides uniqueness checking for validation.
type UniqueStore struct {
	db *sql.DB
}

// NewUniqueStore creates a new unique store.
func NewUniqueStore(db *sql.DB) *UniqueStore {
	return &UniqueStore{db: db}
}

// IsUnique checks if a value is unique in a collection for a given field.
func (s *UniqueStore) IsUnique(ctx context.Context, collection, field string, value any, excludeID string) (bool, error) {
	col := toSnakeCase(field)

	var query string
	var args []any

	if excludeID != "" {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ? AND id != ?", collection, col)
		args = []any{value, excludeID}
	} else {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", collection, col)
		args = []any{value}
	}

	var count int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return false, fmt.Errorf("check unique: %w", err)
	}

	return count == 0, nil
}
