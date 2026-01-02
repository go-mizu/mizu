package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/qa/feature/votes"
)

// VotesStore implements votes.Store.
type VotesStore struct {
	db *sql.DB
}

// NewVotesStore creates a new votes store.
func NewVotesStore(db *sql.DB) *VotesStore {
	return &VotesStore{db: db}
}

// Upsert creates or updates a vote.
func (s *VotesStore) Upsert(ctx context.Context, vote *votes.Vote) (*votes.Vote, error) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO votes (id, voter_id, target_type, target_id, value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (voter_id, target_type, target_id)
		DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, vote.ID, vote.VoterID, vote.TargetType, vote.TargetID, vote.Value, vote.CreatedAt, vote.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, vote.VoterID, vote.TargetType, vote.TargetID)
}

// Get retrieves a vote.
func (s *VotesStore) Get(ctx context.Context, voterID string, targetType votes.TargetType, targetID string) (*votes.Vote, error) {
	v := &votes.Vote{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, voter_id, target_type, target_id, value, created_at, updated_at
		FROM votes WHERE voter_id = $1 AND target_type = $2 AND target_id = $3
	`, voterID, targetType, targetID).Scan(
		&v.ID, &v.VoterID, &v.TargetType, &v.TargetID, &v.Value, &v.CreatedAt, &v.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}
