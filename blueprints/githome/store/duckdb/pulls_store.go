package duckdb

import (
	"context"
	"database/sql"
	"strings"
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
	if pr.UpdatedAt.IsZero() {
		pr.UpdatedAt = now
	}

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

func (s *PullsStore) CountOpen(ctx context.Context, repoID int64) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pull_requests WHERE repo_id = $1 AND state = 'open'
	`, repoID).Scan(&count)
	return count, err
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
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}

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

func (s *PullsStore) CreateCommit(ctx context.Context, prID int64, commit *pulls.Commit) error {
	authorName := ""
	authorEmail := ""
	var authorDate *time.Time
	committerName := ""
	committerEmail := ""
	var committerDate *time.Time
	treeSHA := ""

	if commit.Commit != nil {
		if commit.Commit.Author != nil {
			authorName = commit.Commit.Author.Name
			authorEmail = commit.Commit.Author.Email
			if !commit.Commit.Author.Date.IsZero() {
				authorDate = &commit.Commit.Author.Date
			}
		}
		if commit.Commit.Committer != nil {
			committerName = commit.Commit.Committer.Name
			committerEmail = commit.Commit.Committer.Email
			if !commit.Commit.Committer.Date.IsZero() {
				committerDate = &commit.Commit.Committer.Date
			}
		}
		if commit.Commit.Tree != nil {
			treeSHA = commit.Commit.Tree.SHA
		}
	}

	var authorID, committerID *int64
	if commit.Author != nil {
		authorID = &commit.Author.ID
	}
	if commit.Committer != nil {
		committerID = &commit.Committer.ID
	}

	// Build parent SHAs as JSON array
	parentSHAs := "[]"
	if len(commit.Parents) > 0 {
		shas := make([]string, len(commit.Parents))
		for i, p := range commit.Parents {
			shas[i] = p.SHA
		}
		parentSHAs = `["` + joinStrings(shas, `","`) + `"]`
	}

	message := ""
	if commit.Commit != nil {
		message = commit.Commit.Message
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pr_commits (pr_id, sha, node_id, message, author_name, author_email, author_date, author_id,
			committer_name, committer_email, committer_date, committer_id, tree_sha, parent_shas)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (pr_id, sha) DO UPDATE SET
			message = EXCLUDED.message,
			author_name = EXCLUDED.author_name,
			author_email = EXCLUDED.author_email,
			author_date = EXCLUDED.author_date,
			author_id = EXCLUDED.author_id,
			committer_name = EXCLUDED.committer_name,
			committer_email = EXCLUDED.committer_email,
			committer_date = EXCLUDED.committer_date,
			committer_id = EXCLUDED.committer_id,
			tree_sha = EXCLUDED.tree_sha,
			parent_shas = EXCLUDED.parent_shas
	`, prID, commit.SHA, commit.NodeID, message, authorName, authorEmail, authorDate, authorID,
		committerName, committerEmail, committerDate, committerID, treeSHA, parentSHAs)
	return err
}

func (s *PullsStore) ListCommits(ctx context.Context, prID int64, opts *pulls.ListOpts) ([]*pulls.Commit, error) {
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
		SELECT sha, node_id, message, author_name, author_email, author_date, author_id,
			committer_name, committer_email, committer_date, committer_id, tree_sha
		FROM pr_commits
		WHERE pr_id = $1
		ORDER BY author_date DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*pulls.Commit
	for rows.Next() {
		c := &pulls.Commit{
			Commit: &pulls.CommitData{
				Author:    &pulls.CommitAuthor{},
				Committer: &pulls.CommitAuthor{},
				Tree:      &pulls.CommitRef{},
			},
		}
		var authorID, committerID sql.NullInt64
		var authorDate, committerDate sql.NullTime
		if err := rows.Scan(
			&c.SHA, &c.NodeID, &c.Commit.Message,
			&c.Commit.Author.Name, &c.Commit.Author.Email, &authorDate, &authorID,
			&c.Commit.Committer.Name, &c.Commit.Committer.Email, &committerDate, &committerID,
			&c.Commit.Tree.SHA,
		); err != nil {
			return nil, err
		}
		if authorDate.Valid {
			c.Commit.Author.Date = authorDate.Time
		}
		if committerDate.Valid {
			c.Commit.Committer.Date = committerDate.Time
		}
		if authorID.Valid {
			c.Author = &users.SimpleUser{ID: authorID.Int64}
		}
		if committerID.Valid {
			c.Committer = &users.SimpleUser{ID: committerID.Int64}
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (s *PullsStore) GetCommitBySHA(ctx context.Context, repoID int64, sha string) (*pulls.Commit, error) {
	c := &pulls.Commit{
		Commit: &pulls.CommitData{
			Author:    &pulls.CommitAuthor{},
			Committer: &pulls.CommitAuthor{},
			Tree:      &pulls.CommitRef{},
		},
	}
	var authorID, committerID sql.NullInt64
	var authorDate, committerDate sql.NullTime
	var parentSHAs string

	err := s.db.QueryRowContext(ctx, `
		SELECT c.sha, c.node_id, c.message, c.author_name, c.author_email, c.author_date, c.author_id,
			c.committer_name, c.committer_email, c.committer_date, c.committer_id, c.tree_sha, c.parent_shas
		FROM pr_commits c
		JOIN pull_requests pr ON pr.id = c.pr_id
		WHERE pr.repo_id = $1 AND c.sha = $2
		LIMIT 1
	`, repoID, sha).Scan(
		&c.SHA, &c.NodeID, &c.Commit.Message,
		&c.Commit.Author.Name, &c.Commit.Author.Email, &authorDate, &authorID,
		&c.Commit.Committer.Name, &c.Commit.Committer.Email, &committerDate, &committerID,
		&c.Commit.Tree.SHA, &parentSHAs,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if authorDate.Valid {
		c.Commit.Author.Date = authorDate.Time
	}
	if committerDate.Valid {
		c.Commit.Committer.Date = committerDate.Time
	}
	if authorID.Valid {
		c.Author = &users.SimpleUser{ID: authorID.Int64}
	}
	if committerID.Valid {
		c.Committer = &users.SimpleUser{ID: committerID.Int64}
	}

	// Parse parent SHAs from JSON array
	if parentSHAs != "" && parentSHAs != "[]" {
		// Simple parsing of ["sha1","sha2"] format
		trimmed := strings.Trim(parentSHAs, "[]")
		if trimmed != "" {
			parts := strings.Split(trimmed, ",")
			for _, p := range parts {
				sha := strings.Trim(p, `"`)
				if sha != "" {
					c.Parents = append(c.Parents, &pulls.CommitRef{SHA: sha})
				}
			}
		}
	}

	return c, nil
}

func (s *PullsStore) CreateFile(ctx context.Context, prID int64, file *pulls.PRFile) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pr_files (pr_id, sha, filename, status, additions, deletions, changes, blob_url, raw_url, contents_url, patch, previous_filename)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (pr_id, filename) DO UPDATE SET
			sha = EXCLUDED.sha,
			status = EXCLUDED.status,
			additions = EXCLUDED.additions,
			deletions = EXCLUDED.deletions,
			changes = EXCLUDED.changes,
			blob_url = EXCLUDED.blob_url,
			raw_url = EXCLUDED.raw_url,
			contents_url = EXCLUDED.contents_url,
			patch = EXCLUDED.patch,
			previous_filename = EXCLUDED.previous_filename
	`, prID, file.SHA, file.Filename, file.Status, file.Additions, file.Deletions, file.Changes,
		file.BlobURL, file.RawURL, file.ContentsURL, file.Patch, file.PreviousFilename)
	return err
}

func (s *PullsStore) ListFiles(ctx context.Context, prID int64, opts *pulls.ListOpts) ([]*pulls.PRFile, error) {
	page, perPage := 1, 100
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := `
		SELECT sha, filename, status, additions, deletions, changes, blob_url, raw_url, contents_url, patch, previous_filename
		FROM pr_files
		WHERE pr_id = $1
		ORDER BY filename ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*pulls.PRFile
	for rows.Next() {
		f := &pulls.PRFile{}
		if err := rows.Scan(
			&f.SHA, &f.Filename, &f.Status, &f.Additions, &f.Deletions, &f.Changes,
			&f.BlobURL, &f.RawURL, &f.ContentsURL, &f.Patch, &f.PreviousFilename,
		); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

// ListFilesByCommitSHA returns files for the PR that contains the given commit SHA
func (s *PullsStore) ListFilesByCommitSHA(ctx context.Context, repoID int64, sha string) ([]*pulls.PRFile, error) {
	query := `
		SELECT f.sha, f.filename, f.status, f.additions, f.deletions, f.changes,
		       f.blob_url, f.raw_url, f.contents_url, f.patch, f.previous_filename
		FROM pr_files f
		JOIN pr_commits c ON c.pr_id = f.pr_id
		JOIN pull_requests pr ON pr.id = c.pr_id
		WHERE pr.repo_id = $1 AND c.sha = $2
		ORDER BY f.filename ASC`

	rows, err := s.db.QueryContext(ctx, query, repoID, sha)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*pulls.PRFile
	for rows.Next() {
		f := &pulls.PRFile{}
		if err := rows.Scan(
			&f.SHA, &f.Filename, &f.Status, &f.Additions, &f.Deletions, &f.Changes,
			&f.BlobURL, &f.RawURL, &f.ContentsURL, &f.Patch, &f.PreviousFilename,
		); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

// joinStrings joins strings with a separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
