package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/qa/feature/badges"
)

// BadgesStore implements badges.Store.
type BadgesStore struct {
	db *sql.DB
}

// NewBadgesStore creates a new badges store.
func NewBadgesStore(db *sql.DB) *BadgesStore {
	return &BadgesStore{db: db}
}

// Create creates a badge.
func (s *BadgesStore) Create(ctx context.Context, badge *badges.Badge) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO badges (id, name, tier, description)
		VALUES ($1, $2, $3, $4)
	`, badge.ID, badge.Name, badge.Tier, badge.Description)
	return err
}

// List lists badges.
func (s *BadgesStore) List(ctx context.Context, limit int) ([]*badges.Badge, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, tier, description
		FROM badges
		ORDER BY tier DESC, name ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*badges.Badge
	for rows.Next() {
		badge := &badges.Badge{}
		if err := rows.Scan(&badge.ID, &badge.Name, &badge.Tier, &badge.Description); err != nil {
			return nil, err
		}
		result = append(result, badge)
	}
	return result, rows.Err()
}

// CreateAward creates a badge award.
func (s *BadgesStore) CreateAward(ctx context.Context, award *badges.Award) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO badge_awards (id, account_id, badge_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, award.ID, award.AccountID, award.BadgeID, award.CreatedAt)
	return err
}

// ListAwards lists awards for an account.
func (s *BadgesStore) ListAwards(ctx context.Context, accountID string) ([]*badges.Award, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, badge_id, created_at
		FROM badge_awards
		WHERE account_id = $1
		ORDER BY created_at DESC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*badges.Award
	for rows.Next() {
		award := &badges.Award{}
		if err := rows.Scan(&award.ID, &award.AccountID, &award.BadgeID, &award.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, award)
	}
	return result, rows.Err()
}
