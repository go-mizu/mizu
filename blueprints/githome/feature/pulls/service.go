package pulls

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Service implements the pulls API
type Service struct {
	store Store
}

// NewService creates a new pulls service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new pull request
func (s *Service) Create(ctx context.Context, repoID, authorID string, in *CreateIn) (*PullRequest, error) {
	if in.Title == "" {
		return nil, ErrMissingTitle
	}
	if in.HeadBranch == "" || in.BaseBranch == "" {
		return nil, ErrInvalidInput
	}

	// Get next PR number
	number, err := s.store.GetNextNumber(ctx, repoID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	pr := &PullRequest{
		ID:         ulid.New(),
		RepoID:     repoID,
		Number:     number,
		Title:      strings.TrimSpace(in.Title),
		Body:       in.Body,
		AuthorID:   authorID,
		HeadRepoID: in.HeadRepoID,
		HeadBranch: in.HeadBranch,
		HeadSHA:    "", // Should be set from git
		BaseBranch: in.BaseBranch,
		BaseSHA:    "", // Should be set from git
		State:      StateOpen,
		IsDraft:    in.IsDraft,
		Mergeable:  true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.store.Create(ctx, pr); err != nil {
		return nil, err
	}

	// Add labels
	for _, labelID := range in.Labels {
		pl := &PRLabel{
			ID:        ulid.New(),
			PRID:      pr.ID,
			LabelID:   labelID,
			CreatedAt: now,
		}
		s.store.AddLabel(ctx, pl)
	}

	// Add assignees
	for _, userID := range in.Assignees {
		pa := &PRAssignee{
			ID:        ulid.New(),
			PRID:      pr.ID,
			UserID:    userID,
			CreatedAt: now,
		}
		s.store.AddAssignee(ctx, pa)
	}

	// Request reviews
	for _, userID := range in.Reviewers {
		prr := &PRReviewer{
			ID:        ulid.New(),
			PRID:      pr.ID,
			UserID:    userID,
			State:     "pending",
			CreatedAt: now,
		}
		s.store.AddReviewer(ctx, prr)
	}

	return pr, nil
}

// GetByID retrieves a pull request by ID
func (s *Service) GetByID(ctx context.Context, id string) (*PullRequest, error) {
	pr, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	// Load labels, assignees, reviewers
	pr.Labels, _ = s.store.ListLabels(ctx, id)
	pr.Assignees, _ = s.store.ListAssignees(ctx, id)
	reviewers, _ := s.store.ListReviewers(ctx, id)
	for _, r := range reviewers {
		pr.Reviewers = append(pr.Reviewers, r.UserID)
	}

	return pr, nil
}

// GetByNumber retrieves a pull request by repo ID and number
func (s *Service) GetByNumber(ctx context.Context, repoID string, number int) (*PullRequest, error) {
	pr, err := s.store.GetByNumber(ctx, repoID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	// Load labels, assignees, reviewers
	pr.Labels, _ = s.store.ListLabels(ctx, pr.ID)
	pr.Assignees, _ = s.store.ListAssignees(ctx, pr.ID)
	reviewers, _ := s.store.ListReviewers(ctx, pr.ID)
	for _, r := range reviewers {
		pr.Reviewers = append(pr.Reviewers, r.UserID)
	}

	return pr, nil
}

// Update updates a pull request
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*PullRequest, error) {
	pr, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	if in.Title != nil {
		pr.Title = strings.TrimSpace(*in.Title)
	}
	if in.Body != nil {
		pr.Body = *in.Body
	}
	if in.BaseBranch != nil {
		pr.BaseBranch = *in.BaseBranch
	}
	if in.MilestoneID != nil {
		pr.MilestoneID = *in.MilestoneID
	}

	pr.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, pr); err != nil {
		return nil, err
	}

	// Update labels if provided
	if in.Labels != nil {
		s.SetLabels(ctx, id, *in.Labels)
	}

	// Update assignees if provided
	if in.Assignees != nil {
		existing, _ := s.store.ListAssignees(ctx, id)
		for _, userID := range existing {
			s.store.RemoveAssignee(ctx, id, userID)
		}
		s.AddAssignees(ctx, id, *in.Assignees)
	}

	return pr, nil
}

// List lists pull requests for a repository
func (s *Service) List(ctx context.Context, repoID string, opts *ListOpts) ([]*PullRequest, int, error) {
	state := "all"
	limit := 30
	offset := 0

	if opts != nil {
		if opts.State != "" {
			state = opts.State
		}
		if opts.Limit > 0 && opts.Limit <= 100 {
			limit = opts.Limit
		}
		if opts.Offset >= 0 {
			offset = opts.Offset
		}
	}

	prs, total, err := s.store.List(ctx, repoID, state, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Load related data for each PR
	for _, pr := range prs {
		pr.Labels, _ = s.store.ListLabels(ctx, pr.ID)
		pr.Assignees, _ = s.store.ListAssignees(ctx, pr.ID)
	}

	return prs, total, nil
}

// Close closes a pull request
func (s *Service) Close(ctx context.Context, id string) error {
	pr, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if pr == nil {
		return ErrNotFound
	}
	if pr.State == StateClosed || pr.State == StateMerged {
		return ErrAlreadyClosed
	}

	now := time.Now()
	pr.State = StateClosed
	pr.ClosedAt = &now
	pr.UpdatedAt = now

	return s.store.Update(ctx, pr)
}

// Reopen reopens a pull request
func (s *Service) Reopen(ctx context.Context, id string) error {
	pr, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if pr == nil {
		return ErrNotFound
	}
	if pr.State == StateOpen {
		return ErrAlreadyOpen
	}
	if pr.State == StateMerged {
		return ErrAlreadyMerged
	}

	pr.State = StateOpen
	pr.ClosedAt = nil
	pr.UpdatedAt = time.Now()

	return s.store.Update(ctx, pr)
}

// Merge merges a pull request
func (s *Service) Merge(ctx context.Context, id, userID string, method string, commitMessage string) error {
	pr, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if pr == nil {
		return ErrNotFound
	}
	if pr.State == StateMerged {
		return ErrAlreadyMerged
	}
	if pr.State == StateClosed {
		return ErrAlreadyClosed
	}
	if !pr.Mergeable {
		return ErrNotMergeable
	}

	now := time.Now()
	pr.State = StateMerged
	pr.MergedAt = &now
	pr.MergedByID = userID
	pr.ClosedAt = &now
	pr.UpdatedAt = now
	// MergeCommitSHA would be set by actual git merge

	return s.store.Update(ctx, pr)
}

// MarkReady marks a draft pull request as ready for review
func (s *Service) MarkReady(ctx context.Context, id string) error {
	pr, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if pr == nil {
		return ErrNotFound
	}

	pr.IsDraft = false
	pr.UpdatedAt = time.Now()

	return s.store.Update(ctx, pr)
}

// Lock locks a pull request
func (s *Service) Lock(ctx context.Context, id, reason string) error {
	pr, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if pr == nil {
		return ErrNotFound
	}

	pr.IsLocked = true
	pr.LockReason = reason
	pr.UpdatedAt = time.Now()

	return s.store.Update(ctx, pr)
}

// Unlock unlocks a pull request
func (s *Service) Unlock(ctx context.Context, id string) error {
	pr, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if pr == nil {
		return ErrNotFound
	}

	pr.IsLocked = false
	pr.LockReason = ""
	pr.UpdatedAt = time.Now()

	return s.store.Update(ctx, pr)
}

// AddLabels adds labels to a pull request
func (s *Service) AddLabels(ctx context.Context, id string, labelIDs []string) error {
	now := time.Now()
	for _, labelID := range labelIDs {
		pl := &PRLabel{
			ID:        ulid.New(),
			PRID:      id,
			LabelID:   labelID,
			CreatedAt: now,
		}
		s.store.AddLabel(ctx, pl)
	}
	return nil
}

// RemoveLabel removes a label from a pull request
func (s *Service) RemoveLabel(ctx context.Context, id, labelID string) error {
	return s.store.RemoveLabel(ctx, id, labelID)
}

// SetLabels replaces all labels on a pull request
func (s *Service) SetLabels(ctx context.Context, id string, labelIDs []string) error {
	existing, _ := s.store.ListLabels(ctx, id)
	for _, labelID := range existing {
		s.store.RemoveLabel(ctx, id, labelID)
	}
	return s.AddLabels(ctx, id, labelIDs)
}

// AddAssignees adds assignees to a pull request
func (s *Service) AddAssignees(ctx context.Context, id string, userIDs []string) error {
	now := time.Now()
	for _, userID := range userIDs {
		pa := &PRAssignee{
			ID:        ulid.New(),
			PRID:      id,
			UserID:    userID,
			CreatedAt: now,
		}
		s.store.AddAssignee(ctx, pa)
	}
	return nil
}

// RemoveAssignees removes assignees from a pull request
func (s *Service) RemoveAssignees(ctx context.Context, id string, userIDs []string) error {
	for _, userID := range userIDs {
		s.store.RemoveAssignee(ctx, id, userID)
	}
	return nil
}

// RequestReview requests review from users
func (s *Service) RequestReview(ctx context.Context, id string, userIDs []string) error {
	now := time.Now()
	for _, userID := range userIDs {
		prr := &PRReviewer{
			ID:        ulid.New(),
			PRID:      id,
			UserID:    userID,
			State:     "pending",
			CreatedAt: now,
		}
		s.store.AddReviewer(ctx, prr)
	}
	return nil
}

// RemoveReviewRequest removes review requests
func (s *Service) RemoveReviewRequest(ctx context.Context, id string, userIDs []string) error {
	for _, userID := range userIDs {
		s.store.RemoveReviewer(ctx, id, userID)
	}
	return nil
}

// CreateReview creates a review
func (s *Service) CreateReview(ctx context.Context, prID, userID string, in *CreateReviewIn) (*Review, error) {
	pr, err := s.store.GetByID(ctx, prID)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}
	if pr.IsLocked {
		return nil, ErrLocked
	}

	now := time.Now()
	review := &Review{
		ID:        ulid.New(),
		PRID:      prID,
		UserID:    userID,
		Body:      in.Body,
		State:     ReviewPending,
		CommitSHA: in.CommitSHA,
		CreatedAt: now,
	}

	if err := s.store.CreateReview(ctx, review); err != nil {
		return nil, err
	}

	// Create review comments
	for _, c := range in.Comments {
		rc := &ReviewComment{
			ID:        ulid.New(),
			ReviewID:  review.ID,
			PRID:      prID,
			UserID:    userID,
			Path:      c.Path,
			Position:  c.Position,
			Line:      c.Line,
			Side:      c.Side,
			Body:      c.Body,
			CreatedAt: now,
			UpdatedAt: now,
		}
		s.store.CreateReviewComment(ctx, rc)
	}

	// If event is provided, submit immediately
	if in.Event != "" {
		return s.SubmitReview(ctx, review.ID, in.Event)
	}

	return review, nil
}

// GetReview gets a review
func (s *Service) GetReview(ctx context.Context, id string) (*Review, error) {
	review, err := s.store.GetReview(ctx, id)
	if err != nil {
		return nil, err
	}
	if review == nil {
		return nil, ErrReviewNotFound
	}
	return review, nil
}

// SubmitReview submits a pending review
func (s *Service) SubmitReview(ctx context.Context, reviewID, event string) (*Review, error) {
	review, err := s.store.GetReview(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if review == nil {
		return nil, ErrReviewNotFound
	}

	switch event {
	case "APPROVE":
		review.State = ReviewApproved
	case "REQUEST_CHANGES":
		review.State = ReviewChangesRequested
	case "COMMENT":
		review.State = ReviewCommented
	default:
		return nil, ErrInvalidInput
	}

	now := time.Now()
	review.SubmittedAt = &now

	if err := s.store.UpdateReview(ctx, review); err != nil {
		return nil, err
	}

	return review, nil
}

// DismissReview dismisses a review
func (s *Service) DismissReview(ctx context.Context, reviewID, message string) error {
	review, err := s.store.GetReview(ctx, reviewID)
	if err != nil {
		return err
	}
	if review == nil {
		return ErrReviewNotFound
	}

	review.State = ReviewDismissed
	return s.store.UpdateReview(ctx, review)
}

// ListReviews lists reviews for a pull request
func (s *Service) ListReviews(ctx context.Context, prID string) ([]*Review, error) {
	return s.store.ListReviews(ctx, prID)
}

// CreateReviewComment creates a review comment
func (s *Service) CreateReviewComment(ctx context.Context, prID, reviewID, userID string, in *CreateReviewCommentIn) (*ReviewComment, error) {
	now := time.Now()
	rc := &ReviewComment{
		ID:        ulid.New(),
		ReviewID:  reviewID,
		PRID:      prID,
		UserID:    userID,
		Path:      in.Path,
		Position:  in.Position,
		Line:      in.Line,
		Side:      in.Side,
		Body:      in.Body,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.CreateReviewComment(ctx, rc); err != nil {
		return nil, err
	}

	return rc, nil
}

// UpdateReviewComment updates a review comment
func (s *Service) UpdateReviewComment(ctx context.Context, commentID, body string) (*ReviewComment, error) {
	rc, err := s.store.GetReviewComment(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if rc == nil {
		return nil, ErrCommentNotFound
	}

	rc.Body = body
	rc.UpdatedAt = time.Now()

	if err := s.store.UpdateReviewComment(ctx, rc); err != nil {
		return nil, err
	}

	return rc, nil
}

// DeleteReviewComment deletes a review comment
func (s *Service) DeleteReviewComment(ctx context.Context, commentID string) error {
	rc, err := s.store.GetReviewComment(ctx, commentID)
	if err != nil {
		return err
	}
	if rc == nil {
		return ErrCommentNotFound
	}
	return s.store.DeleteReviewComment(ctx, commentID)
}

// ListReviewComments lists review comments for a pull request
func (s *Service) ListReviewComments(ctx context.Context, prID string) ([]*ReviewComment, error) {
	return s.store.ListReviewComments(ctx, prID)
}
