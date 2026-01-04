package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/synced_blocks"
)

// SyncedBlocksStore implements synced_blocks.Store using DuckDB.
type SyncedBlocksStore struct {
	db *sql.DB
}

// NewSyncedBlocksStore creates a new synced blocks store.
func NewSyncedBlocksStore(db *sql.DB) *SyncedBlocksStore {
	return &SyncedBlocksStore{db: db}
}

// Create inserts a new synced block.
func (s *SyncedBlocksStore) Create(ctx context.Context, sb *synced_blocks.SyncedBlock) error {
	contentJSON, err := json.Marshal(sb.Content)
	if err != nil {
		return fmt.Errorf("marshal content: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO synced_blocks (id, workspace_id, original_id, page_id, page_name, content, last_updated, created_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sb.ID, sb.WorkspaceID, sb.OriginalID, sb.PageID, sb.PageName, string(contentJSON), sb.LastUpdated, sb.CreatedAt, sb.CreatedBy)

	return err
}

// GetByID retrieves a synced block by ID.
func (s *SyncedBlocksStore) GetByID(ctx context.Context, id string) (*synced_blocks.SyncedBlock, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, original_id, page_id, page_name, CAST(content AS VARCHAR), last_updated, created_at, created_by
		FROM synced_blocks
		WHERE id = ?
	`, id)

	return s.scanSyncedBlock(row)
}

// Update updates a synced block.
func (s *SyncedBlocksStore) Update(ctx context.Context, sb *synced_blocks.SyncedBlock) error {
	contentJSON, err := json.Marshal(sb.Content)
	if err != nil {
		return fmt.Errorf("marshal content: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE synced_blocks
		SET content = ?, page_name = ?, last_updated = ?
		WHERE id = ?
	`, string(contentJSON), sb.PageName, sb.LastUpdated, sb.ID)

	return err
}

// Delete removes a synced block and its references.
func (s *SyncedBlocksStore) Delete(ctx context.Context, id string) error {
	// Delete references first
	_, err := s.db.ExecContext(ctx, `DELETE FROM synced_block_references WHERE synced_block_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete the synced block
	_, err = s.db.ExecContext(ctx, `DELETE FROM synced_blocks WHERE id = ?`, id)
	return err
}

// ListByPage returns synced blocks originating from a page.
func (s *SyncedBlocksStore) ListByPage(ctx context.Context, pageID string) ([]*synced_blocks.SyncedBlock, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, original_id, page_id, page_name, CAST(content AS VARCHAR), last_updated, created_at, created_by
		FROM synced_blocks
		WHERE page_id = ?
		ORDER BY created_at DESC
	`, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanSyncedBlocks(rows)
}

// ListByWorkspace returns all synced blocks in a workspace.
func (s *SyncedBlocksStore) ListByWorkspace(ctx context.Context, workspaceID string) ([]*synced_blocks.SyncedBlock, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, original_id, page_id, page_name, CAST(content AS VARCHAR), last_updated, created_at, created_by
		FROM synced_blocks
		WHERE workspace_id = ?
		ORDER BY created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanSyncedBlocks(rows)
}

// CreateReference creates a new synced block reference.
func (s *SyncedBlocksStore) CreateReference(ctx context.Context, ref *synced_blocks.SyncedBlockReference) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO synced_block_references (id, synced_block_id, page_id, block_id, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, ref.ID, ref.SyncedBlockID, ref.PageID, ref.BlockID, ref.CreatedAt)
	return err
}

// DeleteReference removes a synced block reference.
func (s *SyncedBlocksStore) DeleteReference(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM synced_block_references WHERE id = ?`, id)
	return err
}

// GetReferencesByBlock returns all references to a synced block.
func (s *SyncedBlocksStore) GetReferencesByBlock(ctx context.Context, syncedBlockID string) ([]*synced_blocks.SyncedBlockReference, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, synced_block_id, page_id, block_id, created_at
		FROM synced_block_references
		WHERE synced_block_id = ?
		ORDER BY created_at DESC
	`, syncedBlockID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []*synced_blocks.SyncedBlockReference
	for rows.Next() {
		ref := &synced_blocks.SyncedBlockReference{}
		if err := rows.Scan(&ref.ID, &ref.SyncedBlockID, &ref.PageID, &ref.BlockID, &ref.CreatedAt); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}

	return refs, rows.Err()
}

func (s *SyncedBlocksStore) scanSyncedBlock(row *sql.Row) (*synced_blocks.SyncedBlock, error) {
	sb := &synced_blocks.SyncedBlock{}
	var contentJSON string

	err := row.Scan(&sb.ID, &sb.WorkspaceID, &sb.OriginalID, &sb.PageID, &sb.PageName, &contentJSON, &sb.LastUpdated, &sb.CreatedAt, &sb.CreatedBy)
	if err != nil {
		return nil, err
	}

	if contentJSON != "" {
		var content []blocks.Block
		if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
			return nil, fmt.Errorf("unmarshal content: %w", err)
		}
		sb.Content = content
	}

	return sb, nil
}

func (s *SyncedBlocksStore) scanSyncedBlocks(rows *sql.Rows) ([]*synced_blocks.SyncedBlock, error) {
	var results []*synced_blocks.SyncedBlock

	for rows.Next() {
		sb := &synced_blocks.SyncedBlock{}
		var contentJSON string

		err := rows.Scan(&sb.ID, &sb.WorkspaceID, &sb.OriginalID, &sb.PageID, &sb.PageName, &contentJSON, &sb.LastUpdated, &sb.CreatedAt, &sb.CreatedBy)
		if err != nil {
			return nil, err
		}

		if contentJSON != "" {
			var content []blocks.Block
			if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
				return nil, fmt.Errorf("unmarshal content: %w", err)
			}
			sb.Content = content
		}

		results = append(results, sb)
	}

	return results, rows.Err()
}
