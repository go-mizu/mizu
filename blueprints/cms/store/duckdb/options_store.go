package duckdb

import (
	"context"
	"database/sql"
)

// Option represents a WordPress option.
type Option struct {
	OptionID    string
	OptionName  string
	OptionValue string
	Autoload    string
}

// OptionsStore handles option persistence.
type OptionsStore struct {
	db *sql.DB
}

// NewOptionsStore creates a new options store.
func NewOptionsStore(db *sql.DB) *OptionsStore {
	return &OptionsStore{db: db}
}

// Get retrieves an option by name.
func (s *OptionsStore) Get(ctx context.Context, name string) (string, error) {
	query := `SELECT option_value FROM wp_options WHERE option_name = $1`
	var value string
	err := s.db.QueryRowContext(ctx, query, name).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// GetWithDefault retrieves an option by name, returning a default if not found.
func (s *OptionsStore) GetWithDefault(ctx context.Context, name, defaultValue string) (string, error) {
	value, err := s.Get(ctx, name)
	if err != nil {
		return "", err
	}
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

// Set sets an option value, creating or updating as needed.
func (s *OptionsStore) Set(ctx context.Context, name, value string, autoload bool) error {
	autoloadStr := "yes"
	if !autoload {
		autoloadStr = "no"
	}

	// Try update first
	query := `UPDATE wp_options SET option_value = $2, autoload = $3 WHERE option_name = $1`
	result, err := s.db.ExecContext(ctx, query, name, value, autoloadStr)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		return nil
	}

	// Insert if not exists
	insertQuery := `INSERT INTO wp_options (option_id, option_name, option_value, autoload)
		VALUES ($1, $2, $3, $4)`
	_, err = s.db.ExecContext(ctx, insertQuery, NewID(), name, value, autoloadStr)
	return err
}

// Delete deletes an option.
func (s *OptionsStore) Delete(ctx context.Context, name string) error {
	query := `DELETE FROM wp_options WHERE option_name = $1`
	_, err := s.db.ExecContext(ctx, query, name)
	return err
}

// GetAutoloaded retrieves all options with autoload='yes'.
func (s *OptionsStore) GetAutoloaded(ctx context.Context) (map[string]string, error) {
	query := `SELECT option_name, option_value FROM wp_options WHERE autoload = 'yes'`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	options := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		options[name] = value
	}
	return options, rows.Err()
}

// GetMultiple retrieves multiple options by name.
func (s *OptionsStore) GetMultiple(ctx context.Context, names []string) (map[string]string, error) {
	if len(names) == 0 {
		return make(map[string]string), nil
	}

	query := `SELECT option_name, option_value FROM wp_options WHERE option_name = ANY($1)`
	rows, err := s.db.QueryContext(ctx, query, names)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	options := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		options[name] = value
	}
	return options, rows.Err()
}

// List lists all options.
func (s *OptionsStore) List(ctx context.Context) ([]*Option, error) {
	query := `SELECT option_id, option_name, option_value, autoload FROM wp_options ORDER BY option_name`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []*Option
	for rows.Next() {
		o := &Option{}
		if err := rows.Scan(&o.OptionID, &o.OptionName, &o.OptionValue, &o.Autoload); err != nil {
			return nil, err
		}
		options = append(options, o)
	}
	return options, rows.Err()
}

// SetMultiple sets multiple options at once.
func (s *OptionsStore) SetMultiple(ctx context.Context, options map[string]string, autoload bool) error {
	for name, value := range options {
		if err := s.Set(ctx, name, value, autoload); err != nil {
			return err
		}
	}
	return nil
}
