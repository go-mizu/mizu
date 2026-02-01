package sqlite

import (
	"context"
	"database/sql"
)

func (s *Store) GetConfigVal(ctx context.Context, key string) (string, error) {
	var val string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM config WHERE key=?`, key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func (s *Store) SetConfigVal(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value)
	return err
}

func (s *Store) DeleteConfigVal(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM config WHERE key=?`, key)
	return err
}

func (s *Store) ListConfigVals(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM config ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}
