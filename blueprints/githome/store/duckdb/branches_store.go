package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/go-mizu/blueprints/githome/feature/branches"
)

// BranchesStore handles branch protection data access.
type BranchesStore struct {
	db *sql.DB
}

// NewBranchesStore creates a new branches store.
func NewBranchesStore(db *sql.DB) *BranchesStore {
	return &BranchesStore{db: db}
}

func (s *BranchesStore) GetProtection(ctx context.Context, repoID int64, branch string) (*branches.BranchProtection, error) {
	var enabled bool
	var settingsJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT enabled, settings_json FROM branch_protections WHERE repo_id = $1 AND branch = $2
	`, repoID, branch).Scan(&enabled, &settingsJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	bp := &branches.BranchProtection{Enabled: enabled}
	if settingsJSON != "" && settingsJSON != "{}" {
		_ = json.Unmarshal([]byte(settingsJSON), bp)
	}
	bp.Enabled = enabled
	return bp, nil
}

func (s *BranchesStore) SetProtection(ctx context.Context, repoID int64, branch string, protection *branches.BranchProtection) error {
	settingsJSON, err := json.Marshal(protection)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO branch_protections (repo_id, branch, enabled, settings_json)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (repo_id, branch) DO UPDATE SET enabled = $3, settings_json = $4
	`, repoID, branch, protection.Enabled, string(settingsJSON))
	return err
}

func (s *BranchesStore) DeleteProtection(ctx context.Context, repoID int64, branch string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM branch_protections WHERE repo_id = $1 AND branch = $2`, repoID, branch)
	return err
}
