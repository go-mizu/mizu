package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
)

// BlocksStore implements blocks.Store.
type BlocksStore struct {
	db *sql.DB
}

// NewBlocksStore creates a new BlocksStore.
func NewBlocksStore(db *sql.DB) *BlocksStore {
	return &BlocksStore{db: db}
}

func (s *BlocksStore) Create(ctx context.Context, b *blocks.Block) error {
	contentJSON, _ := json.Marshal(b.Content)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO blocks (id, page_id, parent_id, type, content, position, created_by, created_at, updated_by, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, b.ID, b.PageID, b.ParentID, b.Type, string(contentJSON), b.Position, b.CreatedBy, b.CreatedAt, b.UpdatedBy, b.UpdatedAt)
	return err
}

func (s *BlocksStore) GetByID(ctx context.Context, id string) (*blocks.Block, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, page_id, parent_id, type, CAST(content AS VARCHAR), position, created_by, created_at, updated_by, updated_at
		FROM blocks WHERE id = ?
	`, id)
	return s.scanBlock(row)
}

func (s *BlocksStore) Update(ctx context.Context, id string, in *blocks.UpdateIn) error {
	contentJSON, _ := json.Marshal(in.Content)
	blockType := in.Type
	if blockType == "" {
		// Get existing type
		row := s.db.QueryRowContext(ctx, "SELECT type FROM blocks WHERE id = ?", id)
		row.Scan(&blockType)
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE blocks SET type = ?, content = ?, updated_by = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, blockType, string(contentJSON), in.UpdatedBy, id)
	return err
}

func (s *BlocksStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM blocks WHERE id = ?", id)
	return err
}

func (s *BlocksStore) GetByPage(ctx context.Context, pageID string) ([]*blocks.Block, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, page_id, parent_id, type, CAST(content AS VARCHAR), position, created_by, created_at, updated_by, updated_at
		FROM blocks WHERE page_id = ?
		ORDER BY position
	`, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanBlocks(rows)
}

func (s *BlocksStore) GetChildren(ctx context.Context, blockID string) ([]*blocks.Block, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, page_id, parent_id, type, CAST(content AS VARCHAR), position, created_by, created_at, updated_by, updated_at
		FROM blocks WHERE parent_id = ?
		ORDER BY position
	`, blockID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanBlocks(rows)
}

func (s *BlocksStore) Move(ctx context.Context, id string, newParentID string, position int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE blocks SET parent_id = ?, position = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, newParentID, position, id)
	return err
}

func (s *BlocksStore) Reorder(ctx context.Context, parentID string, blockIDs []string) error {
	if len(blockIDs) == 0 {
		return nil
	}

	// Build batch UPDATE with CASE statement to avoid N individual updates
	var caseBuilder strings.Builder
	args := make([]interface{}, 0, len(blockIDs)*2+len(blockIDs))

	caseBuilder.WriteString("UPDATE blocks SET position = CASE id ")
	for i, id := range blockIDs {
		caseBuilder.WriteString("WHEN ? THEN ? ")
		args = append(args, id, i)
	}
	caseBuilder.WriteString("END WHERE id IN (")

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

func (s *BlocksStore) DeleteByPage(ctx context.Context, pageID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM blocks WHERE page_id = ?", pageID)
	return err
}

func (s *BlocksStore) BatchCreate(ctx context.Context, blocks []*blocks.Block) error {
	if len(blocks) == 0 {
		return nil
	}

	// Build multi-row INSERT
	var sb strings.Builder
	sb.WriteString("INSERT INTO blocks (id, page_id, parent_id, type, content, position, created_by, created_at, updated_by, updated_at) VALUES ")

	args := make([]interface{}, 0, len(blocks)*10)
	for i, b := range blocks {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		contentJSON, _ := json.Marshal(b.Content)
		args = append(args, b.ID, b.PageID, b.ParentID, b.Type, string(contentJSON), b.Position, b.CreatedBy, b.CreatedAt, b.UpdatedBy, b.UpdatedAt)
	}

	_, err := s.db.ExecContext(ctx, sb.String(), args...)
	return err
}

func (s *BlocksStore) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := "DELETE FROM blocks WHERE id IN (" + strings.Join(placeholders, ", ") + ")"
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *BlocksStore) scanBlock(row *sql.Row) (*blocks.Block, error) {
	var b blocks.Block
	var contentJSON string
	err := row.Scan(&b.ID, &b.PageID, &b.ParentID, &b.Type, &contentJSON, &b.Position, &b.CreatedBy, &b.CreatedAt, &b.UpdatedBy, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(contentJSON), &b.Content)
	return &b, nil
}

func (s *BlocksStore) scanBlocks(rows *sql.Rows) ([]*blocks.Block, error) {
	var result []*blocks.Block
	for rows.Next() {
		var b blocks.Block
		var contentJSON string
		err := rows.Scan(&b.ID, &b.PageID, &b.ParentID, &b.Type, &contentJSON, &b.Position, &b.CreatedBy, &b.CreatedAt, &b.UpdatedBy, &b.UpdatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(contentJSON), &b.Content)
		result = append(result, &b)
	}
	return result, rows.Err()
}
