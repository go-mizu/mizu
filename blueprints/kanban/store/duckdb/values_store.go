package duckdb

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/values"
)

// ValuesStore handles field value data access.
type ValuesStore struct {
	db *sql.DB
}

// NewValuesStore creates a new values store.
func NewValuesStore(db *sql.DB) *ValuesStore {
	return &ValuesStore{db: db}
}

func (s *ValuesStore) Set(ctx context.Context, v *values.Value) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO field_values (issue_id, field_id, value_text, value_num, value_bool, value_date, value_ts, value_ref, value_json, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (issue_id, field_id) DO UPDATE SET
			value_text = $3,
			value_num = $4,
			value_bool = $5,
			value_date = $6,
			value_ts = $7,
			value_ref = $8,
			value_json = $9,
			updated_at = $10
	`, v.IssueID, v.FieldID, v.ValueText, v.ValueNum, v.ValueBool, v.ValueDate, v.ValueTS, v.ValueRef, v.ValueJSON, time.Now())
	return err
}

func (s *ValuesStore) Get(ctx context.Context, issueID, fieldID string) (*values.Value, error) {
	v := &values.Value{}
	err := s.db.QueryRowContext(ctx, `
		SELECT issue_id, field_id, value_text, value_num, value_bool, value_date, value_ts, value_ref, value_json, updated_at
		FROM field_values WHERE issue_id = $1 AND field_id = $2
	`, issueID, fieldID).Scan(&v.IssueID, &v.FieldID, &v.ValueText, &v.ValueNum, &v.ValueBool, &v.ValueDate, &v.ValueTS, &v.ValueRef, &v.ValueJSON, &v.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return v, err
}

func (s *ValuesStore) ListByIssue(ctx context.Context, issueID string) ([]*values.Value, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT issue_id, field_id, value_text, value_num, value_bool, value_date, value_ts, value_ref, value_json, updated_at
		FROM field_values WHERE issue_id = $1
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanValues(rows)
}

func (s *ValuesStore) ListByField(ctx context.Context, fieldID string) ([]*values.Value, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT issue_id, field_id, value_text, value_num, value_bool, value_date, value_ts, value_ref, value_json, updated_at
		FROM field_values WHERE field_id = $1
	`, fieldID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanValues(rows)
}

func (s *ValuesStore) Delete(ctx context.Context, issueID, fieldID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM field_values WHERE issue_id = $1 AND field_id = $2
	`, issueID, fieldID)
	return err
}

func (s *ValuesStore) DeleteByIssue(ctx context.Context, issueID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM field_values WHERE issue_id = $1
	`, issueID)
	return err
}

func (s *ValuesStore) BulkSet(ctx context.Context, vs []*values.Value) error {
	if len(vs) == 0 {
		return nil
	}

	// Build bulk insert query
	var sb strings.Builder
	sb.WriteString(`
		INSERT INTO field_values (issue_id, field_id, value_text, value_num, value_bool, value_date, value_ts, value_ref, value_json, updated_at)
		VALUES
	`)

	args := make([]any, 0, len(vs)*10)
	now := time.Now()
	for i, v := range vs {
		if i > 0 {
			sb.WriteString(", ")
		}
		base := i * 10
		sb.WriteString("($")
		sb.WriteString(itoa(base + 1))
		sb.WriteString(", $")
		sb.WriteString(itoa(base + 2))
		sb.WriteString(", $")
		sb.WriteString(itoa(base + 3))
		sb.WriteString(", $")
		sb.WriteString(itoa(base + 4))
		sb.WriteString(", $")
		sb.WriteString(itoa(base + 5))
		sb.WriteString(", $")
		sb.WriteString(itoa(base + 6))
		sb.WriteString(", $")
		sb.WriteString(itoa(base + 7))
		sb.WriteString(", $")
		sb.WriteString(itoa(base + 8))
		sb.WriteString(", $")
		sb.WriteString(itoa(base + 9))
		sb.WriteString(", $")
		sb.WriteString(itoa(base + 10))
		sb.WriteString(")")
		args = append(args, v.IssueID, v.FieldID, v.ValueText, v.ValueNum, v.ValueBool, v.ValueDate, v.ValueTS, v.ValueRef, v.ValueJSON, now)
	}

	sb.WriteString(`
		ON CONFLICT (issue_id, field_id) DO UPDATE SET
			value_text = EXCLUDED.value_text,
			value_num = EXCLUDED.value_num,
			value_bool = EXCLUDED.value_bool,
			value_date = EXCLUDED.value_date,
			value_ts = EXCLUDED.value_ts,
			value_ref = EXCLUDED.value_ref,
			value_json = EXCLUDED.value_json,
			updated_at = EXCLUDED.updated_at
	`)

	_, err := s.db.ExecContext(ctx, sb.String(), args...)
	return err
}

func (s *ValuesStore) BulkGetByIssues(ctx context.Context, issueIDs []string) (map[string][]*values.Value, error) {
	if len(issueIDs) == 0 {
		return make(map[string][]*values.Value), nil
	}

	// Build query with placeholders
	var sb strings.Builder
	sb.WriteString(`
		SELECT issue_id, field_id, value_text, value_num, value_bool, value_date, value_ts, value_ref, value_json, updated_at
		FROM field_values WHERE issue_id IN (
	`)

	args := make([]any, len(issueIDs))
	for i, id := range issueIDs {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("$")
		sb.WriteString(itoa(i + 1))
		args[i] = id
	}
	sb.WriteString(")")

	rows, err := s.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]*values.Value)
	for rows.Next() {
		v := &values.Value{}
		if err := rows.Scan(&v.IssueID, &v.FieldID, &v.ValueText, &v.ValueNum, &v.ValueBool, &v.ValueDate, &v.ValueTS, &v.ValueRef, &v.ValueJSON, &v.UpdatedAt); err != nil {
			return nil, err
		}
		result[v.IssueID] = append(result[v.IssueID], v)
	}
	return result, rows.Err()
}

func scanValues(rows *sql.Rows) ([]*values.Value, error) {
	var list []*values.Value
	for rows.Next() {
		v := &values.Value{}
		if err := rows.Scan(&v.IssueID, &v.FieldID, &v.ValueText, &v.ValueNum, &v.ValueBool, &v.ValueDate, &v.ValueTS, &v.ValueRef, &v.ValueJSON, &v.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
