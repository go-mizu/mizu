package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/pulls"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// PullsStore handles pull request data access.
type PullsStore struct {
	db *sql.DB
}

// NewPullsStore creates a new pulls store.
func NewPullsStore(db *sql.DB) *PullsStore {
	return &PullsStore{db: db}
}

func (s *PullsStore) Create(ctx context.Context, pr *pulls.PullRequest) error {
	now := time.Now()
	if pr.CreatedAt.IsZero() {
		pr.CreatedAt = now
	}
	pr.UpdatedAt = now

	headSHA := ""
	baseSHA := ""
	headLabel := ""
	baseLabel := ""
	headRef := ""
	baseRef := ""
	if pr.Head != nil {
		headSHA = pr.Head.SHA
		headLabel = pr.Head.Label
		headRef = pr.Head.Ref
	}
	if pr.Base != nil {
		baseSHA = pr.Base.SHA
		baseLabel = pr.Base.Label
		baseRef = pr.Base.Ref
	}

	var milestoneID *int64
	if pr.Milestone != nil {
		milestoneID = &pr.Milestone.ID
	}

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO pull_requests (node_id, repo_id, number, state, locked, title, body, creator_id,
			head_ref, head_sha, head_label, base_ref, base_sha, base_label, draft, maintainer_can_modify,
			milestone_id, commits, additions, deletions, changed_files, comments, review_comments,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
		RETURNING id
	`, "", pr.RepoID, pr.Number, pr.State, pr.Locked, pr.Title, pr.Body, pr.CreatorID,
		headRef, headSHA, headLabel, baseRef, baseSHA, baseLabel,
		pr.Draft, pr.MaintainerCanModify, nullInt64Ptr(milestoneID),
		pr.Commits, pr.Additions, pr.Deletions, pr.ChangedFiles, pr.Comments, pr.ReviewComments,
		pr.CreatedAt, pr.UpdatedAt).Scan(&pr.ID)
	if err != nil {
		return err
	}

	pr.NodeID = generateNodeID("PR", pr.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE pull_requests SET node_id = $1 WHERE id = $2`, pr.NodeID, pr.ID)
	return err
}

func (s *PullsStore) GetByID(ctx context.Context, id int64) (*pulls.PullRequest, error) {
	pr := &pulls.PullRequest{
		Head: &pulls.PRBranch{},
		Base: &pulls.PRBranch{},
	}
	var closedAt, mergedAt sql.NullTime
	var mergedByID, milestoneID sql.NullInt64
	var mergeable, rebaseable sql.NullBool

	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, repo_id, number, state, locked, title, body, creator_id,
			head_ref, head_sha, head_label, base_ref, base_sha, base_label, draft, merged, mergeable,
			rebaseable, mergeable_state, merge_commit_sha, merged_at, merged_by_id,
			comments, review_comments, maintainer_can_modify, commits, additions, deletions,
			changed_files, closed_at, milestone_id, active_lock_reason, created_at, updated_at
		FROM pull_requests WHERE id = $1
	`, id).Scan(&pr.ID, &pr.NodeID, &pr.RepoID, &pr.Number, &pr.State, &pr.Locked,
		&pr.Title, &pr.Body, &pr.CreatorID, &pr.Head.Ref, &pr.Head.SHA, &pr.Head.Label,
		&pr.Base.Ref, &pr.Base.SHA, &pr.Base.Label, &pr.Draft, &pr.Merged, &mergeable,
		&rebaseable, &pr.MergeableState, &pr.MergeCommitSHA, &mergedAt, &mergedByID,
		&pr.Comments, &pr.ReviewComments, &pr.MaintainerCanModify, &pr.Commits,
		&pr.Additions, &pr.Deletions, &pr.ChangedFiles, &closedAt, &milestoneID,
		&pr.ActiveLockReason, &pr.CreatedAt, &pr.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if closedAt.Valid {
		pr.ClosedAt = &closedAt.Time
	}
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}
	if mergedByID.Valid {
		pr.MergedBy = &users.SimpleUser{ID: mergedByID.Int64}
	}
	if milestoneID.Valid {
		pr.Milestone = &pulls.Milestone{ID: milestoneID.Int64}
	}
	if mergeable.Valid {
		pr.Mergeable = &mergeable.Bool
	}
	if rebaseable.Valid {
		pr.Rebaseable = &rebaseable.Bool
	}
	return pr, nil
}

func (s *PullsStore) GetByNumber(ctx context.Context, repoID int64, number int) (*pulls.PullRequest, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM pull_requests WHERE repo_id = $1 AND number = $2`, repoID, number).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *PullsStore) Update(ctx context.Context, id int64, in *pulls.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE pull_requests SET
			title = COALESCE($2, title),
			body = COALESCE($3, body),
			state = COALESCE($4, state),
			base_ref = COALESCE($5, base_ref),
			maintainer_can_modify = COALESCE($6, maintainer_can_modify),
			updated_at = $7
		WHERE id = $1
	`, id, nullStringPtr(in.Title), nullStringPtr(in.Body), nullStringPtr(in.State),
		nullStringPtr(in.Base), nullBoolPtr(in.MaintainerCanModify), time.Now())
	return err
}

func (s *PullsStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pull_requests WHERE id = $1`, id)
	return err
}

func (s *PullsStore) List(ctx context.Context, repoID int64, opts *pulls.ListOpts) ([]*pulls.PullRequest, error) {
	page, perPage := 1, 30
	state := "open"
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
		if opts.State != "" {
			state = opts.State
		}
	}

	query := `SELECT id FROM pull_requests WHERE repo_id = $1`
	args := []any{repoID}
	if state != "all" {
		query += ` AND state = $2`
		args = append(args, state)
	}
	query += ` ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*pulls.PullRequest
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		pr, err := s.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if pr != nil {
			list = append(list, pr)
		}
	}
	return list, rows.Err()
}

func (s *PullsStore) NextNumber(ctx context.Context, repoID int64) (int, error) {
	var maxNumber sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(number) FROM (
			SELECT number FROM pull_requests WHERE repo_id = $1
			UNION ALL
			SELECT number FROM issues WHERE repo_id = $1
		)
	`, repoID).Scan(&maxNumber)
	if err != nil {
		return 0, err
	}
	if maxNumber.Valid {
		return int(maxNumber.Int64) + 1, nil
	}
	return 1, nil
}

func (s *PullsStore) SetMerged(ctx context.Context, id int64, mergedAt time.Time, mergeCommitSHA string, mergedByID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE pull_requests SET
			merged = TRUE,
			merged_at = $2,
			merge_commit_sha = $3,
			merged_by_id = $4,
			state = 'closed',
			closed_at = $2,
			updated_at = $5
		WHERE id = $1
	`, id, mergedAt, mergeCommitSHA, nullInt64(mergedByID), time.Now())
	return err
}

// Review methods

func (s *PullsStore) CreateReview(ctx context.Context, review *pulls.Review) error {
	if review.SubmittedAt.IsZero() {
		review.SubmittedAt = time.Now()
	}

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO pr_reviews (node_id, pr_id, user_id, body, state, commit_id, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, "", review.PRID, review.UserID, review.Body, review.State, review.CommitID, review.SubmittedAt).Scan(&review.ID)
	if err != nil {
		return err
	}

	review.NodeID = generateNodeID("PRR", review.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE pr_reviews SET node_id = $1 WHERE id = $2`, review.NodeID, review.ID)
	return err
}

func (s *PullsStore) GetReviewByID(ctx context.Context, id int64) (*pulls.Review, error) {
	r := &pulls.Review{User: &users.SimpleUser{}}
	var email sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT r.id, r.node_id, r.pr_id, r.body, r.state, r.commit_id, r.submitted_at,
			u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM pr_reviews r
		JOIN users u ON u.id = r.user_id
		WHERE r.id = $1
	`, id).Scan(&r.ID, &r.NodeID, &r.PRID, &r.Body, &r.State, &r.CommitID, &r.SubmittedAt,
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

func (s *PullsStore) UpdateReview(ctx context.Context, id int64, body string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE pr_reviews SET body = $2 WHERE id = $1`, id, body)
	return err
}

func (s *PullsStore) SetReviewState(ctx context.Context, id int64, state string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE pr_reviews SET state = $2, submitted_at = $3 WHERE id = $1`, id, state, time.Now())
	return err
}

func (s *PullsStore) ListReviews(ctx context.Context, prID int64, opts *pulls.ListOpts) ([]*pulls.Review, error) {
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
		SELECT r.id, r.node_id, r.pr_id, r.body, r.state, r.commit_id, r.submitted_at,
			u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM pr_reviews r
		JOIN users u ON u.id = r.user_id
		WHERE r.pr_id = $1
		ORDER BY r.submitted_at ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*pulls.Review
	for rows.Next() {
		r := &pulls.Review{User: &users.SimpleUser{}}
		var email sql.NullString
		if err := rows.Scan(&r.ID, &r.NodeID, &r.PRID, &r.Body, &r.State, &r.CommitID, &r.SubmittedAt,
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

// Review comment methods

func (s *PullsStore) CreateReviewComment(ctx context.Context, c *pulls.ReviewComment) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO pr_review_comments (node_id, pr_id, review_id, user_id, diff_hunk, path, position,
			original_position, commit_id, original_commit_id, in_reply_to_id, body, line,
			original_line, start_line, original_start_line, side, start_side, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING id
	`, "", c.PRID, nullInt64Ptr(c.ReviewID), c.UserID, c.DiffHunk, c.Path, c.Position,
		c.OriginalPosition, c.CommitID, c.OriginalCommitID, nullInt64(c.InReplyToID),
		c.Body, c.Line, c.OriginalLine, c.StartLine, c.OriginalStartLine, c.Side, c.StartSide,
		c.CreatedAt, c.UpdatedAt).Scan(&c.ID)
	if err != nil {
		return err
	}

	c.NodeID = generateNodeID("PRRC", c.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE pr_review_comments SET node_id = $1 WHERE id = $2`, c.NodeID, c.ID)
	return err
}

func (s *PullsStore) GetReviewCommentByID(ctx context.Context, id int64) (*pulls.ReviewComment, error) {
	c := &pulls.ReviewComment{User: &users.SimpleUser{}}
	var reviewID, inReplyToID sql.NullInt64
	var email sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT c.id, c.node_id, c.pr_id, c.review_id, c.diff_hunk, c.path, c.position,
			c.original_position, c.commit_id, c.original_commit_id, c.in_reply_to_id, c.body,
			c.line, c.original_line, c.start_line, c.original_start_line, c.side, c.start_side,
			c.created_at, c.updated_at,
			u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM pr_review_comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.id = $1
	`, id).Scan(&c.ID, &c.NodeID, &c.PRID, &reviewID, &c.DiffHunk, &c.Path, &c.Position,
		&c.OriginalPosition, &c.CommitID, &c.OriginalCommitID, &inReplyToID, &c.Body,
		&c.Line, &c.OriginalLine, &c.StartLine, &c.OriginalStartLine, &c.Side, &c.StartSide,
		&c.CreatedAt, &c.UpdatedAt,
		&c.User.ID, &c.User.NodeID, &c.User.Login, &c.User.Name, &email,
		&c.User.AvatarURL, &c.User.Type, &c.User.SiteAdmin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if reviewID.Valid {
		c.ReviewID = &reviewID.Int64
	}
	if inReplyToID.Valid {
		c.InReplyToID = inReplyToID.Int64
	}
	if email.Valid {
		c.User.Email = email.String
	}
	return c, err
}

func (s *PullsStore) UpdateReviewComment(ctx context.Context, id int64, body string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE pr_review_comments SET body = $2, updated_at = $3 WHERE id = $1
	`, id, body, time.Now())
	return err
}

func (s *PullsStore) DeleteReviewComment(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pr_review_comments WHERE id = $1`, id)
	return err
}

func (s *PullsStore) ListReviewComments(ctx context.Context, prID int64, opts *pulls.ListOpts) ([]*pulls.ReviewComment, error) {
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
		SELECT c.id, c.node_id, c.pr_id, c.review_id, c.diff_hunk, c.path, c.position,
			c.original_position, c.commit_id, c.original_commit_id, c.in_reply_to_id, c.body,
			c.line, c.original_line, c.start_line, c.original_start_line, c.side, c.start_side,
			c.created_at, c.updated_at,
			u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM pr_review_comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.pr_id = $1
		ORDER BY c.created_at ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*pulls.ReviewComment
	for rows.Next() {
		c := &pulls.ReviewComment{User: &users.SimpleUser{}}
		var reviewID, inReplyToID sql.NullInt64
		var email sql.NullString
		if err := rows.Scan(&c.ID, &c.NodeID, &c.PRID, &reviewID, &c.DiffHunk, &c.Path, &c.Position,
			&c.OriginalPosition, &c.CommitID, &c.OriginalCommitID, &inReplyToID, &c.Body,
			&c.Line, &c.OriginalLine, &c.StartLine, &c.OriginalStartLine, &c.Side, &c.StartSide,
			&c.CreatedAt, &c.UpdatedAt,
			&c.User.ID, &c.User.NodeID, &c.User.Login, &c.User.Name, &email,
			&c.User.AvatarURL, &c.User.Type, &c.User.SiteAdmin); err != nil {
			return nil, err
		}
		if reviewID.Valid {
			c.ReviewID = &reviewID.Int64
		}
		if inReplyToID.Valid {
			c.InReplyToID = inReplyToID.Int64
		}
		if email.Valid {
			c.User.Email = email.String
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// Requested reviewers

func (s *PullsStore) AddRequestedReviewer(ctx context.Context, prID, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pr_requested_reviewers (pr_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, prID, userID)
	return err
}

func (s *PullsStore) RemoveRequestedReviewer(ctx context.Context, prID, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM pr_requested_reviewers WHERE pr_id = $1 AND user_id = $2
	`, prID, userID)
	return err
}

func (s *PullsStore) ListRequestedReviewers(ctx context.Context, prID int64) ([]*users.SimpleUser, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM users u
		JOIN pr_requested_reviewers r ON r.user_id = u.id
		WHERE r.pr_id = $1
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimpleUsers(rows)
}

// Requested teams

func (s *PullsStore) AddRequestedTeam(ctx context.Context, prID, teamID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pr_requested_teams (pr_id, team_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, prID, teamID)
	return err
}

func (s *PullsStore) RemoveRequestedTeam(ctx context.Context, prID, teamID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM pr_requested_teams WHERE pr_id = $1 AND team_id = $2
	`, prID, teamID)
	return err
}

func (s *PullsStore) ListRequestedTeams(ctx context.Context, prID int64) ([]*pulls.TeamSimple, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.node_id, t.name, t.slug, t.description
		FROM teams t
		JOIN pr_requested_teams r ON r.team_id = t.id
		WHERE r.pr_id = $1
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*pulls.TeamSimple
	for rows.Next() {
		t := &pulls.TeamSimple{}
		if err := rows.Scan(&t.ID, &t.NodeID, &t.Name, &t.Slug, &t.Description); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}
