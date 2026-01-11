package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/table/feature/operations"
)

// OperationsStore provides PostgreSQL-based operation storage.
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

	var oldValueJSON, newValueJSON []byte
	if op.OldValue != nil {
		oldValueJSON = op.OldValue
	}
	if op.NewValue != nil {
		newValueJSON = op.NewValue
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO operations (id, table_id, record_id, field_id, view_id, op_type, old_value, new_value, user_id, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, op.ID, nullString(op.TableID), nullString(op.RecordID), nullString(op.FieldID), nullString(op.ViewID), op.OpType, oldValueJSON, newValueJSON, op.UserID, op.Timestamp)
	return err
}

// CreateBatch records multiple operations efficiently.
func (s *OperationsStore) CreateBatch(ctx context.Context, ops []*operations.Operation) error {
	if len(ops) == 0 {
		return nil
	}

	// Use a single transaction for batch insert
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO operations (id, table_id, record_id, field_id, view_id, op_type, old_value, new_value, user_id, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, op := range ops {
		if op.Timestamp.IsZero() {
			op.Timestamp = time.Now()
		}

		var oldValueJSON, newValueJSON []byte
		if op.OldValue != nil {
			oldValueJSON = op.OldValue
		}
		if op.NewValue != nil {
			newValueJSON = op.NewValue
		}

		_, err = stmt.ExecContext(ctx, op.ID, nullString(op.TableID), nullString(op.RecordID), nullString(op.FieldID), nullString(op.ViewID), op.OpType, oldValueJSON, newValueJSON, op.UserID, op.Timestamp)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetByID retrieves an operation by ID.
func (s *OperationsStore) GetByID(ctx context.Context, id string) (*operations.Operation, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, table_id, record_id, field_id, view_id, op_type, old_value, new_value, user_id, timestamp
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
		SELECT id, table_id, record_id, field_id, view_id, op_type, old_value, new_value, user_id, timestamp
		FROM operations WHERE table_id = $1
	`
	args := []any{tableID}
	argIdx := 2

	if !opts.Since.IsZero() {
		query += fmt.Sprintf(` AND timestamp >= $%d`, argIdx)
		args = append(args, opts.Since)
		argIdx++
	}
	if !opts.Until.IsZero() {
		query += fmt.Sprintf(` AND timestamp <= $%d`, argIdx)
		args = append(args, opts.Until)
		argIdx++
	}

	query += fmt.Sprintf(` ORDER BY timestamp DESC LIMIT $%d`, argIdx)
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
		SELECT id, table_id, record_id, field_id, view_id, op_type, old_value, new_value, user_id, timestamp
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
		SELECT id, table_id, record_id, field_id, view_id, op_type, old_value, new_value, user_id, timestamp
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
	var tableID, recordID, fieldID, viewID sql.NullString
	var oldValueJSON, newValueJSON []byte

	err := row.Scan(&op.ID, &tableID, &recordID, &fieldID, &viewID, &op.OpType, &oldValueJSON, &newValueJSON, &op.UserID, &op.Timestamp)
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
	if len(oldValueJSON) > 0 {
		op.OldValue = json.RawMessage(oldValueJSON)
	}
	if len(newValueJSON) > 0 {
		op.NewValue = json.RawMessage(newValueJSON)
	}

	return op, nil
}

func (s *OperationsStore) scanOperations(rows *sql.Rows) ([]*operations.Operation, error) {
	var ops []*operations.Operation
	for rows.Next() {
		op := &operations.Operation{}
		var tableID, recordID, fieldID, viewID sql.NullString
		var oldValueJSON, newValueJSON []byte

		err := rows.Scan(&op.ID, &tableID, &recordID, &fieldID, &viewID, &op.OpType, &oldValueJSON, &newValueJSON, &op.UserID, &op.Timestamp)
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
		if len(oldValueJSON) > 0 {
			op.OldValue = json.RawMessage(oldValueJSON)
		}
		if len(newValueJSON) > 0 {
			op.NewValue = json.RawMessage(newValueJSON)
		}

		ops = append(ops, op)
	}
	return ops, rows.Err()
}
