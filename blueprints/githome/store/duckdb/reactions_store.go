package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/reactions"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// ReactionsStore handles reaction data access.
type ReactionsStore struct {
	db *sql.DB
}

// NewReactionsStore creates a new reactions store.
func NewReactionsStore(db *sql.DB) *ReactionsStore {
	return &ReactionsStore{db: db}
}

func (s *ReactionsStore) Create(ctx context.Context, subjectType string, subjectID, userID int64, content string) (*reactions.Reaction, error) {
	now := time.Now()
	r := &reactions.Reaction{
		User:      &users.SimpleUser{ID: userID},
		Content:   content,
		CreatedAt: now,
	}

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO reactions (node_id, subject_type, subject_id, user_id, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, "", subjectType, subjectID, userID, content, now).Scan(&r.ID)
	if err != nil {
		return nil, err
	}

	r.NodeID = generateNodeID("REA", r.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE reactions SET node_id = $1 WHERE id = $2`, r.NodeID, r.ID)
	if err != nil {
		return nil, err
	}

	// Load user info
	user := &users.SimpleUser{}
	var email sql.NullString
	err = s.db.QueryRowContext(ctx, `
		SELECT id, node_id, login, name, email, avatar_url, type, site_admin
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.NodeID, &user.Login, &user.Name, &email,
		&user.AvatarURL, &user.Type, &user.SiteAdmin)
	if err == nil {
		if email.Valid {
			user.Email = email.String
		}
		r.User = user
	}

	return r, nil
}

func (s *ReactionsStore) GetByID(ctx context.Context, id int64) (*reactions.Reaction, error) {
	r := &reactions.Reaction{User: &users.SimpleUser{}}
	var email sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT r.id, r.node_id, r.content, r.created_at,
			u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM reactions r
		JOIN users u ON u.id = r.user_id
		WHERE r.id = $1
	`, id).Scan(&r.ID, &r.NodeID, &r.Content, &r.CreatedAt,
		&r.User.ID, &r.User.NodeID, &r.User.Login, &r.User.Name, &email,
		&r.User.AvatarURL, &r.User.Type, &r.User.SiteAdmin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if email.Valid {
		r.User.Email = email.String
	}
	return r, err
}

func (s *ReactionsStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM reactions WHERE id = $1`, id)
	return err
}

func (s *ReactionsStore) List(ctx context.Context, subjectType string, subjectID int64, opts *reactions.ListOpts) ([]*reactions.Reaction, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := `
		SELECT r.id, r.node_id, r.content, r.created_at,
			u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM reactions r
		JOIN users u ON u.id = r.user_id
		WHERE r.subject_type = $1 AND r.subject_id = $2
		ORDER BY r.created_at ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, subjectType, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*reactions.Reaction
	for rows.Next() {
		r := &reactions.Reaction{User: &users.SimpleUser{}}
		var email sql.NullString
		if err := rows.Scan(&r.ID, &r.NodeID, &r.Content, &r.CreatedAt,
			&r.User.ID, &r.User.NodeID, &r.User.Login, &r.User.Name, &email,
			&r.User.AvatarURL, &r.User.Type, &r.User.SiteAdmin); err != nil {
			return nil, err
		}
		if email.Valid {
			r.User.Email = email.String
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (s *ReactionsStore) GetByUserAndContent(ctx context.Context, subjectType string, subjectID, userID int64, content string) (*reactions.Reaction, error) {
	r := &reactions.Reaction{User: &users.SimpleUser{}}
	var email sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT r.id, r.node_id, r.content, r.created_at,
			u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM reactions r
		JOIN users u ON u.id = r.user_id
		WHERE r.subject_type = $1 AND r.subject_id = $2 AND r.user_id = $3 AND r.content = $4
	`, subjectType, subjectID, userID, content).Scan(&r.ID, &r.NodeID, &r.Content, &r.CreatedAt,
		&r.User.ID, &r.User.NodeID, &r.User.Login, &r.User.Name, &email,
		&r.User.AvatarURL, &r.User.Type, &r.User.SiteAdmin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if email.Valid {
		r.User.Email = email.String
	}
	return r, err
}

func (s *ReactionsStore) GetRollup(ctx context.Context, subjectType string, subjectID int64) (*reactions.Reactions, error) {
	rollup := &reactions.Reactions{}

	rows, err := s.db.QueryContext(ctx, `
		SELECT content, COUNT(*) as cnt
		FROM reactions
		WHERE subject_type = $1 AND subject_id = $2
		GROUP BY content
	`, subjectType, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var content string
		var count int
		if err := rows.Scan(&content, &count); err != nil {
			return nil, err
		}
		rollup.TotalCount += count
		switch content {
		case reactions.ContentPlusOne:
			rollup.PlusOne = count
		case reactions.ContentMinusOne:
			rollup.MinusOne = count
		case reactions.ContentLaugh:
			rollup.Laugh = count
		case reactions.ContentConfused:
			rollup.Confused = count
		case reactions.ContentHeart:
			rollup.Heart = count
		case reactions.ContentHooray:
			rollup.Hooray = count
		case reactions.ContentRocket:
			rollup.Rocket = count
		case reactions.ContentEyes:
			rollup.Eyes = count
		}
	}
	return rollup, rows.Err()
}
