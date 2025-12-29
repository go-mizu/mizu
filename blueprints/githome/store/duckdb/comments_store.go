package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// CommentsStore handles comment data access.
type CommentsStore struct {
	db *sql.DB
}

// NewCommentsStore creates a new comments store.
func NewCommentsStore(db *sql.DB) *CommentsStore {
	return &CommentsStore{db: db}
}

func (s *CommentsStore) CreateIssueComment(ctx context.Context, c *comments.IssueComment) error {
	now := time.Now()
	// Only set timestamps if not already set (preserves original timestamps during seeding)
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO issue_comments (node_id, issue_id, repo_id, creator_id, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, "", c.IssueID, c.RepoID, c.CreatorID, c.Body, c.CreatedAt, c.UpdatedAt).Scan(&c.ID)
	if err != nil {
		return err
	}

	c.NodeID = generateNodeID("IC", c.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE issue_comments SET node_id = $1 WHERE id = $2`, c.NodeID, c.ID)
	return err
}

func (s *CommentsStore) GetIssueCommentByID(ctx context.Context, id int64) (*comments.IssueComment, error) {
	c := &comments.IssueComment{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, issue_id, repo_id, creator_id, body, created_at, updated_at
		FROM issue_comments WHERE id = $1
	`, id).Scan(&c.ID, &c.NodeID, &c.IssueID, &c.RepoID, &c.CreatorID, &c.Body, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (s *CommentsStore) UpdateIssueComment(ctx context.Context, id int64, body string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE issue_comments SET body = $2, updated_at = $3 WHERE id = $1
	`, id, body, time.Now())
	return err
}

func (s *CommentsStore) DeleteIssueComment(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM issue_comments WHERE id = $1`, id)
	return err
}

func (s *CommentsStore) ListIssueCommentsForRepo(ctx context.Context, repoID int64, opts *comments.ListOpts) ([]*comments.IssueComment, error) {
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
		SELECT id, node_id, issue_id, repo_id, creator_id, body, created_at, updated_at
		FROM issue_comments WHERE repo_id = $1
		ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssueComments(rows)
}

func (s *CommentsStore) ListIssueCommentsForIssue(ctx context.Context, issueID int64, opts *comments.ListOpts) ([]*comments.IssueComment, error) {
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
		SELECT c.id, c.node_id, c.issue_id, c.repo_id, c.creator_id, c.body, c.created_at, c.updated_at,
			   u.id, u.login, u.avatar_url
		FROM issue_comments c
		LEFT JOIN users u ON c.creator_id = u.id
		WHERE c.issue_id = $1
		ORDER BY c.created_at ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanIssueCommentsWithUser(rows)
}

func (s *CommentsStore) CreateCommitComment(ctx context.Context, c *comments.CommitComment) error {
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO commit_comments (node_id, repo_id, creator_id, commit_sha, body, path, position, line, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, "", c.RepoID, c.CreatorID, c.CommitID, c.Body, c.Path, c.Position, c.Line,
		c.CreatedAt, c.UpdatedAt).Scan(&c.ID)
	if err != nil {
		return err
	}

	c.NodeID = generateNodeID("CC", c.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE commit_comments SET node_id = $1 WHERE id = $2`, c.NodeID, c.ID)
	return err
}

func (s *CommentsStore) GetCommitCommentByID(ctx context.Context, id int64) (*comments.CommitComment, error) {
	c := &comments.CommitComment{}
	var position, line sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, repo_id, creator_id, commit_sha, body, path, position, line, created_at, updated_at
		FROM commit_comments WHERE id = $1
	`, id).Scan(&c.ID, &c.NodeID, &c.RepoID, &c.CreatorID, &c.CommitID, &c.Body, &c.Path,
		&position, &line, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if position.Valid {
		c.Position = int(position.Int64)
	}
	if line.Valid {
		c.Line = int(line.Int64)
	}
	return c, err
}

func (s *CommentsStore) UpdateCommitComment(ctx context.Context, id int64, body string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE commit_comments SET body = $2, updated_at = $3 WHERE id = $1
	`, id, body, time.Now())
	return err
}

func (s *CommentsStore) DeleteCommitComment(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM commit_comments WHERE id = $1`, id)
	return err
}

func (s *CommentsStore) ListCommitCommentsForRepo(ctx context.Context, repoID int64, opts *comments.ListOpts) ([]*comments.CommitComment, error) {
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
		SELECT id, node_id, repo_id, creator_id, commit_sha, body, path, position, line, created_at, updated_at
		FROM commit_comments WHERE repo_id = $1
		ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCommitComments(rows)
}

func (s *CommentsStore) ListCommitCommentsForCommit(ctx context.Context, repoID int64, sha string, opts *comments.ListOpts) ([]*comments.CommitComment, error) {
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
		SELECT id, node_id, repo_id, creator_id, commit_sha, body, path, position, line, created_at, updated_at
		FROM commit_comments WHERE repo_id = $1 AND commit_sha = $2
		ORDER BY created_at ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, repoID, sha)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCommitComments(rows)
}

func scanIssueComments(rows *sql.Rows) ([]*comments.IssueComment, error) {
	var list []*comments.IssueComment
	for rows.Next() {
		c := &comments.IssueComment{}
		if err := rows.Scan(&c.ID, &c.NodeID, &c.IssueID, &c.RepoID, &c.CreatorID, &c.Body, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func scanIssueCommentsWithUser(rows *sql.Rows) ([]*comments.IssueComment, error) {
	var list []*comments.IssueComment
	for rows.Next() {
		c := &comments.IssueComment{}
		var userID sql.NullInt64
		var userLogin, userAvatarURL sql.NullString
		if err := rows.Scan(
			&c.ID, &c.NodeID, &c.IssueID, &c.RepoID, &c.CreatorID, &c.Body, &c.CreatedAt, &c.UpdatedAt,
			&userID, &userLogin, &userAvatarURL,
		); err != nil {
			return nil, err
		}
		if userID.Valid {
			c.User = &users.SimpleUser{
				ID:        userID.Int64,
				Login:     userLogin.String,
				AvatarURL: userAvatarURL.String,
			}
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// ListUniqueCommenters returns unique users who commented on an issue
func (s *CommentsStore) ListUniqueCommenters(ctx context.Context, issueID int64) ([]*users.SimpleUser, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM issue_comments c
		JOIN users u ON c.creator_id = u.id
		WHERE c.issue_id = $1
		ORDER BY u.login
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimpleUsers(rows)
}

func scanCommitComments(rows *sql.Rows) ([]*comments.CommitComment, error) {
	var list []*comments.CommitComment
	for rows.Next() {
		c := &comments.CommitComment{}
		var position, line sql.NullInt64
		if err := rows.Scan(&c.ID, &c.NodeID, &c.RepoID, &c.CreatorID, &c.CommitID, &c.Body, &c.Path,
			&position, &line, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if position.Valid {
			c.Position = int(position.Int64)
		}
		if line.Valid {
			c.Line = int(line.Int64)
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
