package duckdb

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/commits"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// CommitsStore handles commit status data access.
type CommitsStore struct {
	db *sql.DB
}

// NewCommitsStore creates a new commits store.
func NewCommitsStore(db *sql.DB) *CommitsStore {
	return &CommitsStore{db: db}
}

func (s *CommitsStore) CreateStatus(ctx context.Context, repoID int64, sha string, status *commits.Status) error {
	now := time.Now()
	if status.CreatedAt.IsZero() {
		status.CreatedAt = now
	}
	status.UpdatedAt = now

	creatorID := int64(0)
	if status.Creator != nil {
		creatorID = status.Creator.ID
	}

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO commit_statuses (node_id, repo_id, sha, state, target_url, description, context, creator_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, "", repoID, sha, status.State, status.TargetURL, status.Description, status.Context,
		creatorID, status.CreatedAt, status.UpdatedAt).Scan(&status.ID)
	if err != nil {
		return err
	}

	status.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Status:%d", status.ID)))
	_, err = s.db.ExecContext(ctx, `UPDATE commit_statuses SET node_id = $1 WHERE id = $2`, status.NodeID, status.ID)
	return err
}

func (s *CommitsStore) GetStatusByID(ctx context.Context, id int64) (*commits.Status, error) {
	status := &commits.Status{Creator: &users.SimpleUser{}}
	var email sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT cs.id, cs.node_id, cs.state, cs.target_url, cs.description, cs.context, cs.created_at, cs.updated_at,
			u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM commit_statuses cs
		LEFT JOIN users u ON u.id = cs.creator_id
		WHERE cs.id = $1
	`, id).Scan(&status.ID, &status.NodeID, &status.State, &status.TargetURL, &status.Description, &status.Context,
		&status.CreatedAt, &status.UpdatedAt,
		&status.Creator.ID, &status.Creator.NodeID, &status.Creator.Login, &status.Creator.Name, &email,
		&status.Creator.AvatarURL, &status.Creator.Type, &status.Creator.SiteAdmin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if email.Valid {
		status.Creator.Email = email.String
	}
	return status, nil
}

func (s *CommitsStore) ListStatuses(ctx context.Context, repoID int64, sha string, opts *commits.ListOpts) ([]*commits.Status, error) {
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
		SELECT cs.id, cs.node_id, cs.state, cs.target_url, cs.description, cs.context, cs.created_at, cs.updated_at,
			u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM commit_statuses cs
		LEFT JOIN users u ON u.id = cs.creator_id
		WHERE cs.repo_id = $1 AND cs.sha = $2
		ORDER BY cs.created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, repoID, sha)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*commits.Status
	for rows.Next() {
		status := &commits.Status{Creator: &users.SimpleUser{}}
		var email sql.NullString
		if err := rows.Scan(&status.ID, &status.NodeID, &status.State, &status.TargetURL, &status.Description,
			&status.Context, &status.CreatedAt, &status.UpdatedAt,
			&status.Creator.ID, &status.Creator.NodeID, &status.Creator.Login, &status.Creator.Name, &email,
			&status.Creator.AvatarURL, &status.Creator.Type, &status.Creator.SiteAdmin); err != nil {
			return nil, err
		}
		if email.Valid {
			status.Creator.Email = email.String
		}
		list = append(list, status)
	}
	return list, rows.Err()
}

func (s *CommitsStore) GetCombinedStatus(ctx context.Context, repoID int64, sha string) (*commits.CombinedStatus, error) {
	statuses, err := s.ListStatuses(ctx, repoID, sha, &commits.ListOpts{PerPage: 100})
	if err != nil {
		return nil, err
	}

	combined := &commits.CombinedStatus{
		State:      "pending",
		Statuses:   statuses,
		SHA:        sha,
		TotalCount: len(statuses),
	}

	if len(statuses) == 0 {
		return combined, nil
	}

	// Calculate combined state
	hasFailure := false
	hasError := false
	hasPending := false
	allSuccess := true

	for _, s := range statuses {
		switch s.State {
		case "failure":
			hasFailure = true
			allSuccess = false
		case "error":
			hasError = true
			allSuccess = false
		case "pending":
			hasPending = true
			allSuccess = false
		case "success":
			// continue
		default:
			allSuccess = false
		}
	}

	switch {
	case hasFailure:
		combined.State = "failure"
	case hasError:
		combined.State = "error"
	case hasPending:
		combined.State = "pending"
	case allSuccess:
		combined.State = "success"
	}

	return combined, nil
}
