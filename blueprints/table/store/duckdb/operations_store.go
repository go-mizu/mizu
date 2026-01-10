package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/table/feature/operations"
)

// OperationsStore provides DuckDB-based operation storage.
type OperationsStore struct {
	db *sql.DB
}

// NewOperationsStore creates a new operations store.
func NewOperationsStore(db *sql.DB) *OperationsStore {
	return &OperationsStore{db: db}
}

// Create records a new operation.
func (s *OperationsStore) Create(ctx context.Context, op *operations.Operation) error {
	if op.Timestamp.IsZero() {
		op.Timestamp = time.Now()
	}

	var oldValueStr, newValueStr *string
	if op.OldValue != nil {
		str := string(op.OldValue)
		oldValueStr = &str
	}
	if op.NewValue != nil {
		str := string(op.NewValue)
		newValueStr = &str
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO operations (id, table_id, record_id, field_id, view_id, op_type, old_value, new_value, user_id, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, op.ID, op.TableID, op.RecordID, op.FieldID, op.ViewID, op.OpType, oldValueStr, newValueStr, op.UserID, op.Timestamp)
	return err
}

// CreateBatch records multiple operations.
func (s *OperationsStore) CreateBatch(ctx context.Context, ops []*operations.Operation) error {
	for _, op := range ops {
		if err := s.Create(ctx, op); err != nil {
			return err
		}
	}
	return nil
}

// GetByID retrieves an operation by ID.
func (s *OperationsStore) GetByID(ctx context.Context, id string) (*operations.Operation, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, table_id, record_id, field_id, view_id, op_type,
			CAST(old_value AS VARCHAR), CAST(new_value AS VARCHAR), user_id, timestamp
		FROM operations WHERE id = $1
	`, id)
	return s.scanOperation(row)
}

// ListByTable lists operations for a table.
func (s *OperationsStore) ListByTable(ctx context.Context, tableID string, opts operations.ListOpts) ([]*operations.Operation, error) {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}

	query := `
		SELECT id, table_id, record_id, field_id, view_id, op_type,
			CAST(old_value AS VARCHAR), CAST(new_value AS VARCHAR), user_id, timestamp
		FROM operations WHERE table_id = $1
	`
	args := []any{tableID}

	if !opts.Since.IsZero() {
		query += ` AND timestamp >= $2`
		args = append(args, opts.Since)
	}
	if !opts.Until.IsZero() {
		query += ` AND timestamp <= $` + string(rune('0'+len(args)+1))
		args = append(args, opts.Until)
	}

	query += ` ORDER BY timestamp DESC LIMIT $` + string(rune('0'+len(args)+1))
	args = append(args, opts.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanOperations(rows)
}

// ListByRecord lists operations for a record.
func (s *OperationsStore) ListByRecord(ctx context.Context, recordID string, opts operations.ListOpts) ([]*operations.Operation, error) {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, table_id, record_id, field_id, view_id, op_type,
			CAST(old_value AS VARCHAR), CAST(new_value AS VARCHAR), user_id, timestamp
		FROM operations WHERE record_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, recordID, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanOperations(rows)
}

// ListByUser lists operations by a user.
func (s *OperationsStore) ListByUser(ctx context.Context, userID string, opts operations.ListOpts) ([]*operations.Operation, error) {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, table_id, record_id, field_id, view_id, op_type,
			CAST(old_value AS VARCHAR), CAST(new_value AS VARCHAR), user_id, timestamp
		FROM operations WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, userID, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanOperations(rows)
}

func (s *OperationsStore) scanOperation(row *sql.Row) (*operations.Operation, error) {
	op := &operations.Operation{}
	var tableID, recordID, fieldID, viewID, oldValueStr, newValueStr sql.NullString

	err := row.Scan(&op.ID, &tableID, &recordID, &fieldID, &viewID, &op.OpType, &oldValueStr, &newValueStr, &op.UserID, &op.Timestamp)
	if err == sql.ErrNoRows {
		return nil, operations.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if tableID.Valid {
		op.TableID = tableID.String
	}
	if recordID.Valid {
		op.RecordID = recordID.String
	}
	if fieldID.Valid {
		op.FieldID = fieldID.String
	}
	if viewID.Valid {
		op.ViewID = viewID.String
	}
	if oldValueStr.Valid {
		op.OldValue = json.RawMessage(oldValueStr.String)
	}
	if newValueStr.Valid {
		op.NewValue = json.RawMessage(newValueStr.String)
	}

	return op, nil
}

func (s *OperationsStore) scanOperations(rows *sql.Rows) ([]*operations.Operation, error) {
	var ops []*operations.Operation
	for rows.Next() {
		op := &operations.Operation{}
		var tableID, recordID, fieldID, viewID, oldValueStr, newValueStr sql.NullString

		err := rows.Scan(&op.ID, &tableID, &recordID, &fieldID, &viewID, &op.OpType, &oldValueStr, &newValueStr, &op.UserID, &op.Timestamp)
		if err != nil {
			return nil, err
		}

		if tableID.Valid {
			op.TableID = tableID.String
		}
		if recordID.Valid {
			op.RecordID = recordID.String
		}
		if fieldID.Valid {
			op.FieldID = fieldID.String
		}
		if viewID.Valid {
			op.ViewID = viewID.String
		}
		if oldValueStr.Valid {
			op.OldValue = json.RawMessage(oldValueStr.String)
		}
		if newValueStr.Valid {
			op.NewValue = json.RawMessage(newValueStr.String)
		}

		ops = append(ops, op)
	}
	return ops, rows.Err()
}
