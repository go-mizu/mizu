package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

// LocalesStore handles localized field value operations.
type LocalesStore struct {
	db *sql.DB
}

// NewLocalesStore creates a new LocalesStore.
func NewLocalesStore(db *sql.DB) *LocalesStore {
	return &LocalesStore{db: db}
}

// LocaleValue represents a localized field value.
type LocaleValue struct {
	ID         string
	Collection string
	DocumentID string
	FieldPath  string
	Locale     string
	Value      any
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// GetLocalizedValue retrieves a localized value for a field.
func (s *LocalesStore) GetLocalizedValue(ctx context.Context, collection, docID, fieldPath, locale string) (any, error) {
	query := `SELECT value FROM _locales WHERE collection = ? AND document_id = ? AND field_path = ? AND locale = ?`

	var valueJSON string
	err := s.db.QueryRowContext(ctx, query, collection, docID, fieldPath, locale).Scan(&valueJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get localized value: %w", err)
	}

	var value any
	if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
		return valueJSON, nil // Return as string if not JSON
	}

	return value, nil
}

// SetLocalizedValue stores a localized value for a field.
func (s *LocalesStore) SetLocalizedValue(ctx context.Context, collection, docID, fieldPath, locale string, value any) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	now := time.Now()

	// Try update first
	updateQuery := `UPDATE _locales SET value = ?, updated_at = ?
		WHERE collection = ? AND document_id = ? AND field_path = ? AND locale = ?`

	result, err := s.db.ExecContext(ctx, updateQuery, string(valueJSON), now, collection, docID, fieldPath, locale)
	if err != nil {
		return fmt.Errorf("update locale: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected > 0 {
		return nil
	}

	// Insert new record
	insertQuery := `INSERT INTO _locales (id, collection, document_id, field_path, locale, value, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.ExecContext(ctx, insertQuery, ulid.New(), collection, docID, fieldPath, locale, string(valueJSON), now, now)
	if err != nil {
		return fmt.Errorf("insert locale: %w", err)
	}

	return nil
}

// GetAllLocales retrieves all localized values for a field.
func (s *LocalesStore) GetAllLocales(ctx context.Context, collection, docID, fieldPath string) (map[string]any, error) {
	query := `SELECT locale, value FROM _locales WHERE collection = ? AND document_id = ? AND field_path = ?`

	rows, err := s.db.QueryContext(ctx, query, collection, docID, fieldPath)
	if err != nil {
		return nil, fmt.Errorf("get all locales: %w", err)
	}
	defer rows.Close()

	result := make(map[string]any)
	for rows.Next() {
		var locale, valueJSON string
		if err := rows.Scan(&locale, &valueJSON); err != nil {
			return nil, fmt.Errorf("scan locale: %w", err)
		}

		var value any
		if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
			value = valueJSON
		}

		result[locale] = value
	}

	return result, nil
}

// GetDocumentLocales retrieves all localized values for a document.
func (s *LocalesStore) GetDocumentLocales(ctx context.Context, collection, docID string) (map[string]map[string]any, error) {
	query := `SELECT field_path, locale, value FROM _locales WHERE collection = ? AND document_id = ?`

	rows, err := s.db.QueryContext(ctx, query, collection, docID)
	if err != nil {
		return nil, fmt.Errorf("get document locales: %w", err)
	}
	defer rows.Close()

	// Map of field_path -> locale -> value
	result := make(map[string]map[string]any)
	for rows.Next() {
		var fieldPath, locale, valueJSON string
		if err := rows.Scan(&fieldPath, &locale, &valueJSON); err != nil {
			return nil, fmt.Errorf("scan locale: %w", err)
		}

		if result[fieldPath] == nil {
			result[fieldPath] = make(map[string]any)
		}

		var value any
		if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
			value = valueJSON
		}

		result[fieldPath][locale] = value
	}

	return result, nil
}

// DeleteLocale removes a locale value for a field.
func (s *LocalesStore) DeleteLocale(ctx context.Context, collection, docID, fieldPath, locale string) error {
	query := `DELETE FROM _locales WHERE collection = ? AND document_id = ? AND field_path = ? AND locale = ?`

	_, err := s.db.ExecContext(ctx, query, collection, docID, fieldPath, locale)
	if err != nil {
		return fmt.Errorf("delete locale: %w", err)
	}

	return nil
}

// DeleteDocumentLocales removes all localized values for a document.
func (s *LocalesStore) DeleteDocumentLocales(ctx context.Context, collection, docID string) error {
	query := `DELETE FROM _locales WHERE collection = ? AND document_id = ?`

	_, err := s.db.ExecContext(ctx, query, collection, docID)
	if err != nil {
		return fmt.Errorf("delete document locales: %w", err)
	}

	return nil
}

// GetAvailableLocales returns all locales that have values for a document.
func (s *LocalesStore) GetAvailableLocales(ctx context.Context, collection, docID string) ([]string, error) {
	query := `SELECT DISTINCT locale FROM _locales WHERE collection = ? AND document_id = ?`

	rows, err := s.db.QueryContext(ctx, query, collection, docID)
	if err != nil {
		return nil, fmt.Errorf("get available locales: %w", err)
	}
	defer rows.Close()

	var locales []string
	for rows.Next() {
		var locale string
		if err := rows.Scan(&locale); err != nil {
			return nil, fmt.Errorf("scan locale: %w", err)
		}
		locales = append(locales, locale)
	}

	return locales, nil
}
