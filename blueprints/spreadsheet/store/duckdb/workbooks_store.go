package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// WorkbooksStore implements workbooks.Store.
type WorkbooksStore struct {
	db *sql.DB
}

// NewWorkbooksStore creates a new workbooks store.
func NewWorkbooksStore(db *sql.DB) *WorkbooksStore {
	return &WorkbooksStore{db: db}
}

// Create creates a new workbook.
func (s *WorkbooksStore) Create(ctx context.Context, wb *workbooks.Workbook) error {
	settings, _ := json.Marshal(wb.Settings)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workbooks (id, name, owner_id, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, wb.ID, wb.Name, wb.OwnerID, string(settings), wb.CreatedAt, wb.UpdatedAt)
	return err
}

// GetByID retrieves a workbook by ID.
func (s *WorkbooksStore) GetByID(ctx context.Context, id string) (*workbooks.Workbook, error) {
	wb := &workbooks.Workbook{}
	var settings sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, owner_id, CAST(settings AS VARCHAR), created_at, updated_at
		FROM workbooks WHERE id = ?
	`, id).Scan(&wb.ID, &wb.Name, &wb.OwnerID, &settings, &wb.CreatedAt, &wb.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, workbooks.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if settings.Valid {
		json.Unmarshal([]byte(settings.String), &wb.Settings)
	}
	return wb, nil
}

// ListByOwner lists workbooks for an owner.
func (s *WorkbooksStore) ListByOwner(ctx context.Context, ownerID string) ([]*workbooks.Workbook, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, owner_id, CAST(settings AS VARCHAR), created_at, updated_at
		FROM workbooks WHERE owner_id = ?
		ORDER BY updated_at DESC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*workbooks.Workbook, 0)
	for rows.Next() {
		wb := &workbooks.Workbook{}
		var settings sql.NullString
		if err := rows.Scan(&wb.ID, &wb.Name, &wb.OwnerID, &settings, &wb.CreatedAt, &wb.UpdatedAt); err != nil {
			return nil, err
		}
		if settings.Valid {
			json.Unmarshal([]byte(settings.String), &wb.Settings)
		}
		result = append(result, wb)
	}
	return result, nil
}

// Update updates a workbook.
func (s *WorkbooksStore) Update(ctx context.Context, wb *workbooks.Workbook) error {
	settings, _ := json.Marshal(wb.Settings)
	_, err := s.db.ExecContext(ctx, `
		UPDATE workbooks SET name = ?, settings = ?, updated_at = ?
		WHERE id = ?
	`, wb.Name, string(settings), wb.UpdatedAt, wb.ID)
	return err
}

// Delete deletes a workbook and related data.
// Properly handles cascade deletes with error checking.
func (s *WorkbooksStore) Delete(ctx context.Context, id string) error {
	// First get all sheet IDs for this workbook to cascade delete sheet-related data
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM sheets WHERE workbook_id = ?`, id)
	if err != nil {
		return err
	}
	var sheetIDs []string
	for rows.Next() {
		var sheetID string
		if err := rows.Scan(&sheetID); err != nil {
			rows.Close()
			return err
		}
		sheetIDs = append(sheetIDs, sheetID)
	}
	rows.Close()

	// Delete all sheet-related data for each sheet
	for _, sheetID := range sheetIDs {
		// Delete comment_replies first (nested FK)
		if _, err := s.db.ExecContext(ctx, `
			DELETE FROM comment_replies WHERE comment_id IN (
				SELECT id FROM comments WHERE sheet_id = ?
			)
		`, sheetID); err != nil {
			return err
		}

		// Delete sheet-related tables (named_ranges deleted at workbook level)
		sheetTables := []string{
			"merged_regions", "cells", "conditional_formats", "data_validations",
			"charts", "pivot_tables", "comments", "auto_filters",
		}
		for _, table := range sheetTables {
			if _, err := s.db.ExecContext(ctx, `DELETE FROM `+table+` WHERE sheet_id = ?`, sheetID); err != nil {
				return err
			}
		}
	}

	// Delete sheets themselves
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sheets WHERE workbook_id = ?`, id); err != nil {
		return err
	}

	// Delete workbook-level related data
	workbookTables := []string{"named_ranges", "shares", "versions"}
	for _, table := range workbookTables {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM `+table+` WHERE workbook_id = ?`, id); err != nil {
			return err
		}
	}

	// Delete the workbook itself
	_, err = s.db.ExecContext(ctx, `DELETE FROM workbooks WHERE id = ?`, id)
	return err
}
