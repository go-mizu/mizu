package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

// CollectionsStore handles generic collection CRUD operations.
type CollectionsStore struct {
	db *sql.DB
}

// NewCollectionsStore creates a new CollectionsStore.
func NewCollectionsStore(db *sql.DB) *CollectionsStore {
	return &CollectionsStore{db: db}
}

// Document represents a generic document.
type Document struct {
	ID        string
	Data      map[string]any
	Status    string
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FindOptions holds options for find operations.
type FindOptions struct {
	Where          map[string]any
	Sort           []SortField
	Limit          int
	Page           int
	Select         []string
	SelectExclude  []string
}

// SortField represents a sort field.
type SortField struct {
	Field string
	Desc  bool
}

// FindResult holds the result of a find operation.
type FindResult struct {
	Docs          []Document
	TotalDocs     int
	Limit         int
	TotalPages    int
	Page          int
	PagingCounter int
	HasPrevPage   bool
	HasNextPage   bool
	PrevPage      *int
	NextPage      *int
}

// Create creates a new document in a collection.
func (s *CollectionsStore) Create(ctx context.Context, collection string, data map[string]any) (*Document, error) {
	id := ulid.New()
	now := time.Now()

	// Build columns and values
	columns := []string{"id", "created_at", "updated_at"}
	placeholders := []string{"?", "?", "?"}
	values := []any{id, now, now}

	for key, value := range data {
		columns = append(columns, toSnakeCase(key))
		placeholders = append(placeholders, "?")
		values = append(values, prepareValue(value))
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		collection,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := s.db.ExecContext(ctx, query, values...)
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}

	return &Document{
		ID:        id,
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// FindByID finds a document by ID.
func (s *CollectionsStore) FindByID(ctx context.Context, collection, id string) (*Document, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ?", collection)
	row := s.db.QueryRowContext(ctx, query, id)

	doc, err := scanDocument(row, collection)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find by id: %w", err)
	}

	return doc, nil
}

// Find finds documents matching the options.
func (s *CollectionsStore) Find(ctx context.Context, collection string, opts *FindOptions) (*FindResult, error) {
	if opts == nil {
		opts = &FindOptions{}
	}

	// Default limit
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}

	// Build WHERE clause
	whereClause, whereArgs := buildWhereClause(opts.Where)

	// Build ORDER BY clause
	orderClause := buildOrderClause(opts.Sort)

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", collection, whereClause)
	var totalDocs int
	if err := s.db.QueryRowContext(ctx, countQuery, whereArgs...).Scan(&totalDocs); err != nil {
		return nil, fmt.Errorf("count documents: %w", err)
	}

	// Calculate pagination
	totalPages := (totalDocs + opts.Limit - 1) / opts.Limit
	offset := (opts.Page - 1) * opts.Limit

	// Build main query
	query := fmt.Sprintf(
		"SELECT * FROM %s %s %s LIMIT ? OFFSET ?",
		collection, whereClause, orderClause,
	)
	args := append(whereArgs, opts.Limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find documents: %w", err)
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		doc, err := scanDocumentFromRows(rows, collection)
		if err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		docs = append(docs, *doc)
	}

	result := &FindResult{
		Docs:          docs,
		TotalDocs:     totalDocs,
		Limit:         opts.Limit,
		TotalPages:    totalPages,
		Page:          opts.Page,
		PagingCounter: offset + 1,
		HasPrevPage:   opts.Page > 1,
		HasNextPage:   opts.Page < totalPages,
	}

	if result.HasPrevPage {
		prev := opts.Page - 1
		result.PrevPage = &prev
	}
	if result.HasNextPage {
		next := opts.Page + 1
		result.NextPage = &next
	}

	return result, nil
}

// Count counts documents matching the where clause.
func (s *CollectionsStore) Count(ctx context.Context, collection string, where map[string]any) (int, error) {
	whereClause, whereArgs := buildWhereClause(where)
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", collection, whereClause)

	var count int
	if err := s.db.QueryRowContext(ctx, query, whereArgs...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}

	return count, nil
}

// UpdateByID updates a document by ID.
func (s *CollectionsStore) UpdateByID(ctx context.Context, collection, id string, data map[string]any) (*Document, error) {
	now := time.Now()

	// Build SET clause
	var sets []string
	var values []any

	for key, value := range data {
		sets = append(sets, fmt.Sprintf("%s = ?", toSnakeCase(key)))
		values = append(values, prepareValue(value))
	}

	sets = append(sets, "updated_at = ?")
	values = append(values, now)
	values = append(values, id)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = ?",
		collection, strings.Join(sets, ", "),
	)

	result, err := s.db.ExecContext(ctx, query, values...)
	if err != nil {
		return nil, fmt.Errorf("update document: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, nil
	}

	return s.FindByID(ctx, collection, id)
}

// Update updates documents matching the where clause.
func (s *CollectionsStore) Update(ctx context.Context, collection string, where map[string]any, data map[string]any) (int64, error) {
	now := time.Now()

	// Build SET clause
	var sets []string
	var values []any

	for key, value := range data {
		sets = append(sets, fmt.Sprintf("%s = ?", toSnakeCase(key)))
		values = append(values, prepareValue(value))
	}

	sets = append(sets, "updated_at = ?")
	values = append(values, now)

	// Build WHERE clause
	whereClause, whereArgs := buildWhereClause(where)
	values = append(values, whereArgs...)

	query := fmt.Sprintf(
		"UPDATE %s SET %s %s",
		collection, strings.Join(sets, ", "), whereClause,
	)

	result, err := s.db.ExecContext(ctx, query, values...)
	if err != nil {
		return 0, fmt.Errorf("update documents: %w", err)
	}

	return result.RowsAffected()
}

// DeleteByID deletes a document by ID.
func (s *CollectionsStore) DeleteByID(ctx context.Context, collection, id string) (bool, error) {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", collection)
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return false, fmt.Errorf("delete document: %w", err)
	}

	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

// Delete deletes documents matching the where clause.
func (s *CollectionsStore) Delete(ctx context.Context, collection string, where map[string]any) (int64, error) {
	whereClause, whereArgs := buildWhereClause(where)
	query := fmt.Sprintf("DELETE FROM %s %s", collection, whereClause)

	result, err := s.db.ExecContext(ctx, query, whereArgs...)
	if err != nil {
		return 0, fmt.Errorf("delete documents: %w", err)
	}

	return result.RowsAffected()
}

// Helper functions

func buildWhereClause(where map[string]any) (string, []any) {
	if len(where) == 0 {
		return "", nil
	}

	var conditions []string
	var args []any

	for field, condition := range where {
		cond, ok := condition.(map[string]any)
		if !ok {
			// Simple equality
			conditions = append(conditions, fmt.Sprintf("%s = ?", toSnakeCase(field)))
			args = append(args, condition)
			continue
		}

		for op, value := range cond {
			clause, opArgs := buildOperatorClause(field, op, value)
			conditions = append(conditions, clause)
			args = append(args, opArgs...)
		}
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

func buildOperatorClause(field, op string, value any) (string, []any) {
	col := toSnakeCase(field)

	switch op {
	case "equals":
		return fmt.Sprintf("%s = ?", col), []any{value}
	case "not_equals":
		return fmt.Sprintf("%s != ?", col), []any{value}
	case "greater_than":
		return fmt.Sprintf("%s > ?", col), []any{value}
	case "greater_than_equal":
		return fmt.Sprintf("%s >= ?", col), []any{value}
	case "less_than":
		return fmt.Sprintf("%s < ?", col), []any{value}
	case "less_than_equal":
		return fmt.Sprintf("%s <= ?", col), []any{value}
	case "like":
		return fmt.Sprintf("LOWER(%s) LIKE LOWER(?)", col), []any{"%" + fmt.Sprint(value) + "%"}
	case "contains":
		return fmt.Sprintf("%s LIKE ?", col), []any{"%" + fmt.Sprint(value) + "%"}
	case "in":
		vals, ok := value.([]any)
		if !ok {
			return fmt.Sprintf("%s = ?", col), []any{value}
		}
		placeholders := strings.Repeat("?,", len(vals))
		placeholders = placeholders[:len(placeholders)-1]
		return fmt.Sprintf("%s IN (%s)", col, placeholders), vals
	case "not_in":
		vals, ok := value.([]any)
		if !ok {
			return fmt.Sprintf("%s != ?", col), []any{value}
		}
		placeholders := strings.Repeat("?,", len(vals))
		placeholders = placeholders[:len(placeholders)-1]
		return fmt.Sprintf("%s NOT IN (%s)", col, placeholders), vals
	case "exists":
		if value == true {
			return fmt.Sprintf("%s IS NOT NULL", col), nil
		}
		return fmt.Sprintf("%s IS NULL", col), nil
	default:
		return fmt.Sprintf("%s = ?", col), []any{value}
	}
}

func buildOrderClause(sort []SortField) string {
	if len(sort) == 0 {
		return "ORDER BY created_at DESC"
	}

	var orders []string
	for _, s := range sort {
		dir := "ASC"
		if s.Desc {
			dir = "DESC"
		}
		orders = append(orders, fmt.Sprintf("%s %s", toSnakeCase(s.Field), dir))
	}

	return "ORDER BY " + strings.Join(orders, ", ")
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(r + 32)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func prepareValue(v any) any {
	switch val := v.(type) {
	case map[string]any, []any, []string, []int:
		b, _ := json.Marshal(val)
		return string(b)
	default:
		return val
	}
}

func scanDocument(row *sql.Row, collection string) (*Document, error) {
	// This is a simplified version - in production, we'd need to handle
	// the dynamic column structure properly
	var id string
	var createdAt, updatedAt time.Time

	// For now, scan into a map using column info
	// This would need to be more sophisticated for real use
	if err := row.Scan(&id, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	return &Document{
		ID:        id,
		Data:      make(map[string]any),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func scanDocumentFromRows(rows *sql.Rows, collection string) (*Document, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	doc := &Document{
		Data: make(map[string]any),
	}

	for i, col := range columns {
		val := values[i]

		switch col {
		case "id":
			doc.ID = fmt.Sprint(val)
		case "created_at":
			if t, ok := val.(time.Time); ok {
				doc.CreatedAt = t
			}
		case "updated_at":
			if t, ok := val.(time.Time); ok {
				doc.UpdatedAt = t
			}
		case "_status":
			doc.Status = fmt.Sprint(val)
		case "_version":
			if v, ok := val.(int64); ok {
				doc.Version = int(v)
			}
		default:
			// Convert to camelCase for response
			doc.Data[toCamelCase(col)] = val
		}
	}

	return doc, nil
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}
