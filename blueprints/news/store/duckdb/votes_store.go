package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/news/feature/votes"
)

// VotesStore implements votes.Store.
type VotesStore struct {
	db *sql.DB
}

// NewVotesStore creates a new votes store.
func NewVotesStore(db *sql.DB) *VotesStore {
	return &VotesStore{db: db}
}

// GetByUserAndTarget retrieves a vote by user and target.
func (s *VotesStore) GetByUserAndTarget(ctx context.Context, userID, targetType, targetID string) (*votes.Vote, error) {
	vote := &votes.Vote{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, target_type, target_id, value, created_at
		FROM votes
		WHERE user_id = $1 AND target_type = $2 AND target_id = $3
	`, userID, targetType, targetID).Scan(
		&vote.ID, &vote.UserID, &vote.TargetType, &vote.TargetID, &vote.Value, &vote.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, votes.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return vote, nil
}

// GetByUserAndTargets retrieves votes for multiple targets.
func (s *VotesStore) GetByUserAndTargets(ctx context.Context, userID, targetType string, targetIDs []string) (map[string]*votes.Vote, error) {
	if len(targetIDs) == 0 {
		return make(map[string]*votes.Vote), nil
	}

	placeholders := make([]string, len(targetIDs))
	args := make([]any, len(targetIDs)+2)
	args[0] = userID
	args[1] = targetType
	for i, id := range targetIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+3)
		args[i+2] = id
	}

	query := `
		SELECT id, user_id, target_type, target_id, value, created_at
		FROM votes
		WHERE user_id = $1 AND target_type = $2 AND target_id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*votes.Vote)
	for rows.Next() {
		vote := &votes.Vote{}
		if err := rows.Scan(&vote.ID, &vote.UserID, &vote.TargetType, &vote.TargetID, &vote.Value, &vote.CreatedAt); err != nil {
			return nil, err
		}
		result[vote.TargetID] = vote
	}
	return result, rows.Err()
}

// CountByTarget counts votes for a target.
func (s *VotesStore) CountByTarget(ctx context.Context, targetType, targetID string) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(value), 0)
		FROM votes
		WHERE target_type = $1 AND target_id = $2
	`, targetType, targetID).Scan(&count)
	return count, err
}
