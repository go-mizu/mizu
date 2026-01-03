package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/rows"
)

// RowsStore implements rows.Store using DuckDB.
type RowsStore struct {
	db *sql.DB
}

// NewRowsStore creates a new RowsStore.
func NewRowsStore(db *sql.DB) *RowsStore {
	return &RowsStore{db: db}
}

func (s *RowsStore) Create(ctx context.Context, row *rows.Row) error {
	propsJSON, err := json.Marshal(row.Properties)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO database_rows (id, database_id, properties, created_by, created_at, updated_by, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, row.ID, row.DatabaseID, string(propsJSON), row.CreatedBy, row.CreatedAt, row.UpdatedBy, row.UpdatedAt)
	return err
}

func (s *RowsStore) GetByID(ctx context.Context, id string) (*rows.Row, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, database_id, CAST(properties AS VARCHAR), created_by, created_at, updated_by, updated_at
		FROM database_rows WHERE id = ?
	`, id)
	return s.scanRow(row)
}

func (s *RowsStore) Update(ctx context.Context, id string, in *rows.UpdateIn) error {
	propsJSON, err := json.Marshal(in.Properties)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE database_rows
		SET properties = ?, updated_by = ?, updated_at = ?
		WHERE id = ?
	`, string(propsJSON), in.UpdatedBy, time.Now(), id)
	return err
}

func (s *RowsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM database_rows WHERE id = ?", id)
	return err
}

func (s *RowsStore) DeleteByDatabase(ctx context.Context, databaseID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM database_rows WHERE database_id = ?", databaseID)
	return err
}

func (s *RowsStore) List(ctx context.Context, in *rows.ListIn) ([]*rows.Row, error) {
	query := `
		SELECT id, database_id, CAST(properties AS VARCHAR), created_by, created_at, updated_by, updated_at
		FROM database_rows
		WHERE database_id = ?
	`
	args := []interface{}{in.DatabaseID}

	// Apply cursor pagination
	if in.Cursor != "" {
		query += " AND id > ?"
		args = append(args, in.Cursor)
	}

	// Apply filters
	filterSQL, filterArgs := s.buildFilterSQL(in.Filters)
	if filterSQL != "" {
		query += " AND " + filterSQL
		args = append(args, filterArgs...)
	}

	// Apply sorts
	sortSQL := s.buildSortSQL(in.Sorts)
	if sortSQL != "" {
		query += " ORDER BY " + sortSQL
	} else {
		query += " ORDER BY created_at DESC"
	}

	// Apply limit
	if in.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", in.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRows(rows)
}

func (s *RowsStore) Count(ctx context.Context, databaseID string, filters []rows.Filter) (int, error) {
	query := "SELECT COUNT(*) FROM database_rows WHERE database_id = ?"
	args := []interface{}{databaseID}

	filterSQL, filterArgs := s.buildFilterSQL(filters)
	if filterSQL != "" {
		query += " AND " + filterSQL
		args = append(args, filterArgs...)
	}

	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// buildFilterSQL builds SQL WHERE clause from filters.
func (s *RowsStore) buildFilterSQL(filters []rows.Filter) (string, []interface{}) {
	if len(filters) == 0 {
		return "", nil
	}

	var conditions []string
	var args []interface{}

	for _, f := range filters {
		condition, filterArgs := s.buildSingleFilter(f)
		if condition != "" {
			conditions = append(conditions, condition)
			args = append(args, filterArgs...)
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "(" + strings.Join(conditions, " AND ") + ")", args
}

func (s *RowsStore) buildSingleFilter(f rows.Filter) (string, []interface{}) {
	// Escape property name for JSON path
	propPath := fmt.Sprintf("$.%s", f.Property)

	switch f.Operator {
	// Text operators
	case "is":
		return fmt.Sprintf("json_extract_string(properties, '%s') = ?", propPath), []interface{}{f.Value}
	case "is_not":
		return fmt.Sprintf("json_extract_string(properties, '%s') != ?", propPath), []interface{}{f.Value}
	case "contains":
		return fmt.Sprintf("json_extract_string(properties, '%s') LIKE ?", propPath), []interface{}{"%" + fmt.Sprint(f.Value) + "%"}
	case "does_not_contain":
		return fmt.Sprintf("json_extract_string(properties, '%s') NOT LIKE ?", propPath), []interface{}{"%" + fmt.Sprint(f.Value) + "%"}
	case "starts_with":
		return fmt.Sprintf("json_extract_string(properties, '%s') LIKE ?", propPath), []interface{}{fmt.Sprint(f.Value) + "%"}
	case "ends_with":
		return fmt.Sprintf("json_extract_string(properties, '%s') LIKE ?", propPath), []interface{}{"%" + fmt.Sprint(f.Value)}
	case "is_empty":
		return fmt.Sprintf("(json_extract_string(properties, '%s') IS NULL OR json_extract_string(properties, '%s') = '')", propPath, propPath), nil
	case "is_not_empty":
		return fmt.Sprintf("(json_extract_string(properties, '%s') IS NOT NULL AND json_extract_string(properties, '%s') != '')", propPath, propPath), nil

	// Number operators
	case "=":
		return fmt.Sprintf("CAST(json_extract_string(properties, '%s') AS DOUBLE) = ?", propPath), []interface{}{f.Value}
	case "!=", "≠":
		return fmt.Sprintf("CAST(json_extract_string(properties, '%s') AS DOUBLE) != ?", propPath), []interface{}{f.Value}
	case ">":
		return fmt.Sprintf("CAST(json_extract_string(properties, '%s') AS DOUBLE) > ?", propPath), []interface{}{f.Value}
	case "<":
		return fmt.Sprintf("CAST(json_extract_string(properties, '%s') AS DOUBLE) < ?", propPath), []interface{}{f.Value}
	case ">=", "≥":
		return fmt.Sprintf("CAST(json_extract_string(properties, '%s') AS DOUBLE) >= ?", propPath), []interface{}{f.Value}
	case "<=", "≤":
		return fmt.Sprintf("CAST(json_extract_string(properties, '%s') AS DOUBLE) <= ?", propPath), []interface{}{f.Value}

	// Date operators
	case "before":
		dateVal := s.resolveDateValue(f.Value)
		return fmt.Sprintf("CAST(json_extract_string(properties, '%s') AS DATE) < ?", propPath), []interface{}{dateVal}
	case "after":
		dateVal := s.resolveDateValue(f.Value)
		return fmt.Sprintf("CAST(json_extract_string(properties, '%s') AS DATE) > ?", propPath), []interface{}{dateVal}
	case "on_or_before":
		dateVal := s.resolveDateValue(f.Value)
		return fmt.Sprintf("CAST(json_extract_string(properties, '%s') AS DATE) <= ?", propPath), []interface{}{dateVal}
	case "on_or_after":
		dateVal := s.resolveDateValue(f.Value)
		return fmt.Sprintf("CAST(json_extract_string(properties, '%s') AS DATE) >= ?", propPath), []interface{}{dateVal}

	// Checkbox
	case "is_true":
		return fmt.Sprintf("json_extract(properties, '%s') = true", propPath), nil
	case "is_false":
		return fmt.Sprintf("(json_extract(properties, '%s') = false OR json_extract(properties, '%s') IS NULL)", propPath, propPath), nil
	}

	return "", nil
}

// resolveDateValue resolves relative date values to actual dates.
func (s *RowsStore) resolveDateValue(value interface{}) string {
	// Check if it's a relative date
	if m, ok := value.(map[string]interface{}); ok {
		if m["type"] == "relative" {
			return s.resolveRelativeDate(fmt.Sprint(m["value"]))
		}
	}

	// Return as-is for absolute dates
	return fmt.Sprint(value)
}

func (s *RowsStore) resolveRelativeDate(value string) string {
	now := time.Now()

	switch value {
	case "today":
		return now.Format("2006-01-02")
	case "tomorrow":
		return now.AddDate(0, 0, 1).Format("2006-01-02")
	case "yesterday":
		return now.AddDate(0, 0, -1).Format("2006-01-02")
	case "one_week_ago":
		return now.AddDate(0, 0, -7).Format("2006-01-02")
	case "one_week_from_now":
		return now.AddDate(0, 0, 7).Format("2006-01-02")
	case "one_month_ago":
		return now.AddDate(0, -1, 0).Format("2006-01-02")
	case "one_month_from_now":
		return now.AddDate(0, 1, 0).Format("2006-01-02")
	case "this_week":
		// Return start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return now.AddDate(0, 0, -weekday+1).Format("2006-01-02")
	case "last_week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return now.AddDate(0, 0, -weekday-6).Format("2006-01-02")
	case "next_week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return now.AddDate(0, 0, 8-weekday).Format("2006-01-02")
	case "this_month":
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	case "last_month":
		return time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	case "next_month":
		return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	case "this_year":
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	case "last_year":
		return time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	case "next_year":
		return time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	}

	return value
}

// buildSortSQL builds SQL ORDER BY clause from sorts.
func (s *RowsStore) buildSortSQL(sorts []rows.Sort) string {
	if len(sorts) == 0 {
		return ""
	}

	var parts []string
	for _, sort := range sorts {
		propPath := fmt.Sprintf("$.%s", sort.Property)
		direction := "ASC"
		if sort.Direction == "desc" {
			direction = "DESC"
		}
		parts = append(parts, fmt.Sprintf("json_extract_string(properties, '%s') %s", propPath, direction))
	}

	return strings.Join(parts, ", ")
}

func (s *RowsStore) scanRow(row *sql.Row) (*rows.Row, error) {
	var r rows.Row
	var propsJSON string
	err := row.Scan(&r.ID, &r.DatabaseID, &propsJSON, &r.CreatedBy, &r.CreatedAt, &r.UpdatedBy, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(propsJSON), &r.Properties)
	if r.Properties == nil {
		r.Properties = make(map[string]interface{})
	}
	return &r, nil
}

func (s *RowsStore) scanRows(sqlRows *sql.Rows) ([]*rows.Row, error) {
	var result []*rows.Row
	for sqlRows.Next() {
		var r rows.Row
		var propsJSON string
		err := sqlRows.Scan(&r.ID, &r.DatabaseID, &propsJSON, &r.CreatedBy, &r.CreatedAt, &r.UpdatedBy, &r.UpdatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(propsJSON), &r.Properties)
		if r.Properties == nil {
			r.Properties = make(map[string]interface{})
		}
		result = append(result, &r)
	}
	return result, sqlRows.Err()
}
