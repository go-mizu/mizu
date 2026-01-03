package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/rowblocks"
)

// RowBlocksStore implements rowblocks.Store using DuckDB.
type RowBlocksStore struct {
	db *sql.DB
}

// NewRowBlocksStore creates a new RowBlocksStore.
func NewRowBlocksStore(db *sql.DB) *RowBlocksStore {
	return &RowBlocksStore{db: db}
}

func (s *RowBlocksStore) Create(ctx context.Context, b *rowblocks.Block) error {
	propsJSON, err := json.Marshal(b.Properties)
	if err != nil {
		propsJSON = []byte("{}")
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO row_content_blocks (id, row_id, parent_id, type, content, properties, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, b.ID, b.RowID, nullString(b.ParentID), b.Type, b.Content, string(propsJSON), b.Order, b.CreatedAt, b.UpdatedAt)
	return err
}

func (s *RowBlocksStore) GetByID(ctx context.Context, id string) (*rowblocks.Block, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, row_id, parent_id, type, content, CAST(properties AS VARCHAR), sort_order, created_at, updated_at
		FROM row_content_blocks WHERE id = ?
	`, id)
	return s.scanBlock(row)
}

func (s *RowBlocksStore) Update(ctx context.Context, id string, in *rowblocks.UpdateIn) error {
	// Get existing block to merge properties
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	content := existing.Content
	if in.Content != "" {
		content = in.Content
	}

	props := existing.Properties
	if props == nil {
		props = make(map[string]interface{})
	}

	// Merge new properties
	if in.Properties != nil {
		for k, v := range in.Properties {
			props[k] = v
		}
	}

	// Handle checked field for to_do blocks
	if in.Checked != nil {
		props["checked"] = *in.Checked
	}

	propsJSON, err := json.Marshal(props)
	if err != nil {
		propsJSON = []byte("{}")
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE row_content_blocks SET content = ?, properties = ?, updated_at = ? WHERE id = ?
	`, content, string(propsJSON), time.Now(), id)
	return err
}

func (s *RowBlocksStore) Delete(ctx context.Context, id string) error {
	// Also delete any child blocks
	_, err := s.db.ExecContext(ctx, "DELETE FROM row_content_blocks WHERE parent_id = ?", id)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM row_content_blocks WHERE id = ?", id)
	return err
}

func (s *RowBlocksStore) ListByRow(ctx context.Context, rowID string) ([]*rowblocks.Block, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, row_id, parent_id, type, content, CAST(properties AS VARCHAR), sort_order, created_at, updated_at
		FROM row_content_blocks WHERE row_id = ?
		ORDER BY sort_order ASC
	`, rowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanBlocks(rows)
}

func (s *RowBlocksStore) GetMaxOrder(ctx context.Context, rowID string) (int, error) {
	var maxOrder sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(sort_order) FROM row_content_blocks WHERE row_id = ?
	`, rowID).Scan(&maxOrder)
	if err != nil {
		return -1, err
	}
	if !maxOrder.Valid {
		return -1, nil
	}
	return int(maxOrder.Int64), nil
}

func (s *RowBlocksStore) UpdateOrders(ctx context.Context, rowID string, blockIDs []string) error {
	if len(blockIDs) == 0 {
		return nil
	}

	// Build batch UPDATE with CASE statement to avoid N individual updates
	var caseBuilder strings.Builder
	now := time.Now()
	args := make([]interface{}, 0, len(blockIDs)*2+len(blockIDs)+2)

	caseBuilder.WriteString("UPDATE row_content_blocks SET updated_at = ?, sort_order = CASE id ")
	args = append(args, now)

	for i, id := range blockIDs {
		caseBuilder.WriteString("WHEN ? THEN ? ")
		args = append(args, id, i)
	}
	caseBuilder.WriteString("END WHERE row_id = ? AND id IN (")
	args = append(args, rowID)

	placeholders := make([]string, len(blockIDs))
	for i, id := range blockIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	caseBuilder.WriteString(strings.Join(placeholders, ", "))
	caseBuilder.WriteString(")")

	_, err := s.db.ExecContext(ctx, caseBuilder.String(), args...)
	return err
}

func (s *RowBlocksStore) scanBlock(row *sql.Row) (*rowblocks.Block, error) {
	var b rowblocks.Block
	var parentID sql.NullString
	var propsJSON string
	err := row.Scan(&b.ID, &b.RowID, &parentID, &b.Type, &b.Content, &propsJSON, &b.Order, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		b.ParentID = parentID.String
	}
	json.Unmarshal([]byte(propsJSON), &b.Properties)
	if b.Properties == nil {
		b.Properties = make(map[string]interface{})
	}
	return &b, nil
}

func (s *RowBlocksStore) scanBlocks(rows *sql.Rows) ([]*rowblocks.Block, error) {
	var result []*rowblocks.Block
	for rows.Next() {
		var b rowblocks.Block
		var parentID sql.NullString
		var propsJSON string
		err := rows.Scan(&b.ID, &b.RowID, &parentID, &b.Type, &b.Content, &propsJSON, &b.Order, &b.CreatedAt, &b.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if parentID.Valid {
			b.ParentID = parentID.String
		}
		json.Unmarshal([]byte(propsJSON), &b.Properties)
		if b.Properties == nil {
			b.Properties = make(map[string]interface{})
		}
		result = append(result, &b)
	}
	return result, rows.Err()
}

// nullString returns a sql.NullString for optional string values.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
