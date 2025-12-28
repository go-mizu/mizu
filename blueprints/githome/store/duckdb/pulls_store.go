package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/pulls"
)

// PullsStore implements pulls.Store
type PullsStore struct {
	db *sql.DB
}

// NewPullsStore creates a new pulls store
func NewPullsStore(db *sql.DB) *PullsStore {
	return &PullsStore{db: db}
}

// Create creates a new pull request
func (s *PullsStore) Create(ctx context.Context, pr *pulls.PullRequest) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pull_requests (id, repo_id, number, title, body, author_id, head_repo_id, head_branch, head_sha, base_branch, base_sha, state, is_draft, is_locked, lock_reason, mergeable, mergeable_state, merge_method, merge_commit_sha, merge_message, merged_at, merged_by_id, additions, deletions, changed_files, comment_count, review_comments, commits, milestone_id, created_at, updated_at, closed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32)
	`, pr.ID, pr.RepoID, pr.Number, pr.Title, pr.Body, pr.AuthorID, nullString(pr.HeadRepoID), pr.HeadBranch, pr.HeadSHA, pr.BaseBranch, pr.BaseSHA, pr.State, pr.IsDraft, pr.IsLocked, pr.LockReason, pr.Mergeable, pr.MergeableState, pr.MergeMethod, pr.MergeCommitSHA, pr.MergeMessage, pr.MergedAt, nullString(pr.MergedByID), pr.Additions, pr.Deletions, pr.ChangedFiles, pr.CommentCount, pr.ReviewComments, pr.Commits, nullString(pr.MilestoneID), pr.CreatedAt, pr.UpdatedAt, pr.ClosedAt)
	return err
}

// GetByID retrieves a pull request by ID
func (s *PullsStore) GetByID(ctx context.Context, id string) (*pulls.PullRequest, error) {
	pr := &pulls.PullRequest{}
	var headRepoID, mergedByID, milestoneID sql.NullString
	var mergedAt, closedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, number, title, body, author_id, head_repo_id, head_branch, head_sha, base_branch, base_sha, state, is_draft, is_locked, lock_reason, mergeable, mergeable_state, merge_method, merge_commit_sha, merge_message, merged_at, merged_by_id, additions, deletions, changed_files, comment_count, review_comments, commits, milestone_id, created_at, updated_at, closed_at
		FROM pull_requests WHERE id = $1
	`, id).Scan(&pr.ID, &pr.RepoID, &pr.Number, &pr.Title, &pr.Body, &pr.AuthorID, &headRepoID, &pr.HeadBranch, &pr.HeadSHA, &pr.BaseBranch, &pr.BaseSHA, &pr.State, &pr.IsDraft, &pr.IsLocked, &pr.LockReason, &pr.Mergeable, &pr.MergeableState, &pr.MergeMethod, &pr.MergeCommitSHA, &pr.MergeMessage, &mergedAt, &mergedByID, &pr.Additions, &pr.Deletions, &pr.ChangedFiles, &pr.CommentCount, &pr.ReviewComments, &pr.Commits, &milestoneID, &pr.CreatedAt, &pr.UpdatedAt, &closedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.populateNullables(pr, headRepoID, mergedByID, milestoneID, mergedAt, closedAt)
	return pr, nil
}

// GetByNumber retrieves a pull request by repo ID and number
func (s *PullsStore) GetByNumber(ctx context.Context, repoID string, number int) (*pulls.PullRequest, error) {
	pr := &pulls.PullRequest{}
	var headRepoID, mergedByID, milestoneID sql.NullString
	var mergedAt, closedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, number, title, body, author_id, head_repo_id, head_branch, head_sha, base_branch, base_sha, state, is_draft, is_locked, lock_reason, mergeable, mergeable_state, merge_method, merge_commit_sha, merge_message, merged_at, merged_by_id, additions, deletions, changed_files, comment_count, review_comments, commits, milestone_id, created_at, updated_at, closed_at
		FROM pull_requests WHERE repo_id = $1 AND number = $2
	`, repoID, number).Scan(&pr.ID, &pr.RepoID, &pr.Number, &pr.Title, &pr.Body, &pr.AuthorID, &headRepoID, &pr.HeadBranch, &pr.HeadSHA, &pr.BaseBranch, &pr.BaseSHA, &pr.State, &pr.IsDraft, &pr.IsLocked, &pr.LockReason, &pr.Mergeable, &pr.MergeableState, &pr.MergeMethod, &pr.MergeCommitSHA, &pr.MergeMessage, &mergedAt, &mergedByID, &pr.Additions, &pr.Deletions, &pr.ChangedFiles, &pr.CommentCount, &pr.ReviewComments, &pr.Commits, &milestoneID, &pr.CreatedAt, &pr.UpdatedAt, &closedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.populateNullables(pr, headRepoID, mergedByID, milestoneID, mergedAt, closedAt)
	return pr, nil
}

func (s *PullsStore) populateNullables(pr *pulls.PullRequest, headRepoID, mergedByID, milestoneID sql.NullString, mergedAt, closedAt sql.NullTime) {
	if headRepoID.Valid {
		pr.HeadRepoID = headRepoID.String
	}
	if mergedByID.Valid {
		pr.MergedByID = mergedByID.String
	}
	if milestoneID.Valid {
		pr.MilestoneID = milestoneID.String
	}
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}
	if closedAt.Valid {
		pr.ClosedAt = &closedAt.Time
	}
}

// Update updates a pull request
func (s *PullsStore) Update(ctx context.Context, pr *pulls.PullRequest) error {
	pr.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE pull_requests SET title = $2, body = $3, base_branch = $4, state = $5, is_draft = $6, is_locked = $7, lock_reason = $8, mergeable = $9, mergeable_state = $10, merge_method = $11, merge_commit_sha = $12, merge_message = $13, merged_at = $14, merged_by_id = $15, milestone_id = $16, updated_at = $17, closed_at = $18
		WHERE id = $1
	`, pr.ID, pr.Title, pr.Body, pr.BaseBranch, pr.State, pr.IsDraft, pr.IsLocked, pr.LockReason, pr.Mergeable, pr.MergeableState, pr.MergeMethod, pr.MergeCommitSHA, pr.MergeMessage, pr.MergedAt, nullString(pr.MergedByID), nullString(pr.MilestoneID), pr.UpdatedAt, pr.ClosedAt)
	return err
}

// Delete deletes a pull request
func (s *PullsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pull_requests WHERE id = $1`, id)
	return err
}

// List lists pull requests for a repository
func (s *PullsStore) List(ctx context.Context, repoID string, state string, limit, offset int) ([]*pulls.PullRequest, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM pull_requests WHERE repo_id = $1`
	args := []interface{}{repoID}
	if state != "" && state != "all" {
		countQuery += ` AND state = $2`
		args = append(args, state)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get pull requests
	query := `
		SELECT id, repo_id, number, title, body, author_id, head_repo_id, head_branch, head_sha, base_branch, base_sha, state, is_draft, is_locked, lock_reason, mergeable, mergeable_state, merge_method, merge_commit_sha, merge_message, merged_at, merged_by_id, additions, deletions, changed_files, comment_count, review_comments, commits, milestone_id, created_at, updated_at, closed_at
		FROM pull_requests WHERE repo_id = $1`
	if state != "" && state != "all" {
		query += ` AND state = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		args = append(args, limit, offset)
	} else {
		query += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*pulls.PullRequest
	for rows.Next() {
		pr := &pulls.PullRequest{}
		var headRepoID, mergedByID, milestoneID sql.NullString
		var mergedAt, closedAt sql.NullTime
		if err := rows.Scan(&pr.ID, &pr.RepoID, &pr.Number, &pr.Title, &pr.Body, &pr.AuthorID, &headRepoID, &pr.HeadBranch, &pr.HeadSHA, &pr.BaseBranch, &pr.BaseSHA, &pr.State, &pr.IsDraft, &pr.IsLocked, &pr.LockReason, &pr.Mergeable, &pr.MergeableState, &pr.MergeMethod, &pr.MergeCommitSHA, &pr.MergeMessage, &mergedAt, &mergedByID, &pr.Additions, &pr.Deletions, &pr.ChangedFiles, &pr.CommentCount, &pr.ReviewComments, &pr.Commits, &milestoneID, &pr.CreatedAt, &pr.UpdatedAt, &closedAt); err != nil {
			return nil, 0, err
		}
		s.populateNullables(pr, headRepoID, mergedByID, milestoneID, mergedAt, closedAt)
		list = append(list, pr)
	}
	return list, total, rows.Err()
}

// GetNextNumber gets the next PR number for a repository
func (s *PullsStore) GetNextNumber(ctx context.Context, repoID string) (int, error) {
	var maxNum sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(number) FROM pull_requests WHERE repo_id = $1
	`, repoID).Scan(&maxNum)
	if err != nil {
		return 1, err
	}
	if !maxNum.Valid {
		return 1, nil
	}
	return int(maxNum.Int64) + 1, nil
}

// AddLabel adds a label to a PR - uses composite PK (pr_id, label_id)
func (s *PullsStore) AddLabel(ctx context.Context, pl *pulls.PRLabel) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pr_labels (pr_id, label_id, created_at)
		VALUES ($1, $2, $3)
	`, pl.PRID, pl.LabelID, pl.CreatedAt)
	return err
}

// RemoveLabel removes a label from a PR
func (s *PullsStore) RemoveLabel(ctx context.Context, prID, labelID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pr_labels WHERE pr_id = $1 AND label_id = $2`, prID, labelID)
	return err
}

// ListLabels lists labels for a PR
func (s *PullsStore) ListLabels(ctx context.Context, prID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT label_id FROM pr_labels WHERE pr_id = $1`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []string
	for rows.Next() {
		var labelID string
		if err := rows.Scan(&labelID); err != nil {
			return nil, err
		}
		labels = append(labels, labelID)
	}
	return labels, rows.Err()
}

// AddAssignee adds an assignee to a PR - uses composite PK (pr_id, user_id)
func (s *PullsStore) AddAssignee(ctx context.Context, pa *pulls.PRAssignee) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pr_assignees (pr_id, user_id, created_at)
		VALUES ($1, $2, $3)
	`, pa.PRID, pa.UserID, pa.CreatedAt)
	return err
}

// RemoveAssignee removes an assignee from a PR
func (s *PullsStore) RemoveAssignee(ctx context.Context, prID, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pr_assignees WHERE pr_id = $1 AND user_id = $2`, prID, userID)
	return err
}

// ListAssignees lists assignees for a PR
func (s *PullsStore) ListAssignees(ctx context.Context, prID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id FROM pr_assignees WHERE pr_id = $1`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignees []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		assignees = append(assignees, userID)
	}
	return assignees, rows.Err()
}

// AddReviewer adds a reviewer to a PR - uses composite PK (pr_id, user_id)
func (s *PullsStore) AddReviewer(ctx context.Context, pr *pulls.PRReviewer) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pr_reviewers (pr_id, user_id, state, created_at)
		VALUES ($1, $2, $3, $4)
	`, pr.PRID, pr.UserID, pr.State, pr.CreatedAt)
	return err
}

// RemoveReviewer removes a reviewer from a PR
func (s *PullsStore) RemoveReviewer(ctx context.Context, prID, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pr_reviewers WHERE pr_id = $1 AND user_id = $2`, prID, userID)
	return err
}

// ListReviewers lists reviewers for a PR
func (s *PullsStore) ListReviewers(ctx context.Context, prID string) ([]*pulls.PRReviewer, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT pr_id, user_id, state, created_at
		FROM pr_reviewers WHERE pr_id = $1
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*pulls.PRReviewer
	for rows.Next() {
		pr := &pulls.PRReviewer{}
		if err := rows.Scan(&pr.PRID, &pr.UserID, &pr.State, &pr.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, pr)
	}
	return list, rows.Err()
}

// CreateReview creates a review
func (s *PullsStore) CreateReview(ctx context.Context, r *pulls.Review) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pr_reviews (id, pr_id, user_id, body, state, commit_sha, created_at, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, r.ID, r.PRID, r.UserID, r.Body, r.State, r.CommitSHA, r.CreatedAt, r.SubmittedAt)
	return err
}

// GetReview retrieves a review by ID
func (s *PullsStore) GetReview(ctx context.Context, id string) (*pulls.Review, error) {
	r := &pulls.Review{}
	var submittedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, pr_id, user_id, body, state, commit_sha, created_at, submitted_at
		FROM pr_reviews WHERE id = $1
	`, id).Scan(&r.ID, &r.PRID, &r.UserID, &r.Body, &r.State, &r.CommitSHA, &r.CreatedAt, &submittedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if submittedAt.Valid {
		r.SubmittedAt = &submittedAt.Time
	}
	return r, nil
}

// UpdateReview updates a review
func (s *PullsStore) UpdateReview(ctx context.Context, r *pulls.Review) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE pr_reviews SET body = $2, state = $3, submitted_at = $4
		WHERE id = $1
	`, r.ID, r.Body, r.State, r.SubmittedAt)
	return err
}

// ListReviews lists reviews for a PR
func (s *PullsStore) ListReviews(ctx context.Context, prID string) ([]*pulls.Review, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, pr_id, user_id, body, state, commit_sha, created_at, submitted_at
		FROM pr_reviews WHERE pr_id = $1 ORDER BY created_at ASC
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*pulls.Review
	for rows.Next() {
		r := &pulls.Review{}
		var submittedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.PRID, &r.UserID, &r.Body, &r.State, &r.CommitSHA, &r.CreatedAt, &submittedAt); err != nil {
			return nil, err
		}
		if submittedAt.Valid {
			r.SubmittedAt = &submittedAt.Time
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// CreateReviewComment creates a review comment
func (s *PullsStore) CreateReviewComment(ctx context.Context, rc *pulls.ReviewComment) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO review_comments (id, review_id, user_id, path, position, original_position, diff_hunk, line, original_line, side, body, in_reply_to_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, rc.ID, rc.ReviewID, rc.UserID, rc.Path, rc.Position, rc.OriginalPosition, rc.DiffHunk, rc.Line, rc.OriginalLine, rc.Side, rc.Body, nullString(rc.InReplyToID), rc.CreatedAt, rc.UpdatedAt)
	return err
}

// GetReviewComment retrieves a review comment by ID
func (s *PullsStore) GetReviewComment(ctx context.Context, id string) (*pulls.ReviewComment, error) {
	rc := &pulls.ReviewComment{}
	var inReplyToID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, review_id, user_id, path, position, original_position, diff_hunk, line, original_line, side, body, in_reply_to_id, created_at, updated_at
		FROM review_comments WHERE id = $1
	`, id).Scan(&rc.ID, &rc.ReviewID, &rc.UserID, &rc.Path, &rc.Position, &rc.OriginalPosition, &rc.DiffHunk, &rc.Line, &rc.OriginalLine, &rc.Side, &rc.Body, &inReplyToID, &rc.CreatedAt, &rc.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if inReplyToID.Valid {
		rc.InReplyToID = inReplyToID.String
	}
	return rc, nil
}

// UpdateReviewComment updates a review comment
func (s *PullsStore) UpdateReviewComment(ctx context.Context, rc *pulls.ReviewComment) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE review_comments SET body = $2, updated_at = $3
		WHERE id = $1
	`, rc.ID, rc.Body, rc.UpdatedAt)
	return err
}

// DeleteReviewComment deletes a review comment
func (s *PullsStore) DeleteReviewComment(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM review_comments WHERE id = $1`, id)
	return err
}

// ListReviewComments lists review comments for a PR
func (s *PullsStore) ListReviewComments(ctx context.Context, prID string) ([]*pulls.ReviewComment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT rc.id, rc.review_id, rc.user_id, rc.path, rc.position, rc.original_position, rc.diff_hunk, rc.line, rc.original_line, rc.side, rc.body, rc.in_reply_to_id, rc.created_at, rc.updated_at
		FROM review_comments rc
		JOIN pr_reviews r ON rc.review_id = r.id
		WHERE r.pr_id = $1
		ORDER BY rc.created_at ASC
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*pulls.ReviewComment
	for rows.Next() {
		rc := &pulls.ReviewComment{}
		var inReplyToID sql.NullString
		if err := rows.Scan(&rc.ID, &rc.ReviewID, &rc.UserID, &rc.Path, &rc.Position, &rc.OriginalPosition, &rc.DiffHunk, &rc.Line, &rc.OriginalLine, &rc.Side, &rc.Body, &inReplyToID, &rc.CreatedAt, &rc.UpdatedAt); err != nil {
			return nil, err
		}
		if inReplyToID.Valid {
			rc.InReplyToID = inReplyToID.String
		}
		list = append(list, rc)
	}
	return list, rows.Err()
}
