package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/forum/feature/votes"
)

// VotesStore implements votes.Store.
type VotesStore struct {
	db *sql.DB
}

// NewVotesStore creates a new votes store.
func NewVotesStore(db *sql.DB) *VotesStore {
	return &VotesStore{db: db}
}

// Create creates a vote.
func (s *VotesStore) Create(ctx context.Context, vote *votes.Vote) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO votes (id, account_id, target_type, target_id, value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, vote.ID, vote.AccountID, vote.TargetType, vote.TargetID, vote.Value, vote.CreatedAt, vote.UpdatedAt)
	return err
}

// GetByTarget retrieves a vote by target.
func (s *VotesStore) GetByTarget(ctx context.Context, accountID, targetType, targetID string) (*votes.Vote, error) {
	vote := &votes.Vote{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, target_type, target_id, value, created_at, updated_at
		FROM votes WHERE account_id = $1 AND target_type = $2 AND target_id = $3
	`, accountID, targetType, targetID).Scan(
		&vote.ID, &vote.AccountID, &vote.TargetType, &vote.TargetID,
		&vote.Value, &vote.CreatedAt, &vote.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, votes.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return vote, nil
}

// Update updates a vote.
func (s *VotesStore) Update(ctx context.Context, vote *votes.Vote) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE votes SET value = $4, updated_at = $5
		WHERE account_id = $1 AND target_type = $2 AND target_id = $3
	`, vote.AccountID, vote.TargetType, vote.TargetID, vote.Value, vote.UpdatedAt)
	return err
}

// Delete deletes a vote.
func (s *VotesStore) Delete(ctx context.Context, accountID, targetType, targetID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM votes WHERE account_id = $1 AND target_type = $2 AND target_id = $3
	`, accountID, targetType, targetID)
	return err
}

// GetByTargets retrieves votes for multiple targets.
func (s *VotesStore) GetByTargets(ctx context.Context, accountID, targetType string, targetIDs []string) ([]*votes.Vote, error) {
	if len(targetIDs) == 0 {
		return nil, nil
	}

	// Build IN clause
	query := `
		SELECT id, account_id, target_type, target_id, value, created_at, updated_at
		FROM votes WHERE account_id = $1 AND target_type = $2 AND target_id IN (`

	args := []any{accountID, targetType}
	for i, id := range targetIDs {
		if i > 0 {
			query += ", "
		}
		query += "?"
		args = append(args, id)
	}
	query += ")"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*votes.Vote
	for rows.Next() {
		vote := &votes.Vote{}
		err := rows.Scan(
			&vote.ID, &vote.AccountID, &vote.TargetType, &vote.TargetID,
			&vote.Value, &vote.CreatedAt, &vote.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, vote)
	}
	return result, rows.Err()
}

// CountByTarget counts votes for a target.
func (s *VotesStore) CountByTarget(ctx context.Context, targetType, targetID string) (up, down int64, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN value = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN value = -1 THEN 1 ELSE 0 END), 0)
		FROM votes WHERE target_type = $1 AND target_id = $2
	`, targetType, targetID).Scan(&up, &down)
	return
}
