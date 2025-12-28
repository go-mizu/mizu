package issues

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Service implements the issues API
type Service struct {
	store Store
}

// NewService creates a new issues service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new issue
func (s *Service) Create(ctx context.Context, repoID, authorID string, in *CreateIn) (*Issue, error) {
	if in.Title == "" {
		return nil, ErrMissingTitle
	}

	// Get next issue number
	number, err := s.store.GetNextNumber(ctx, repoID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	issue := &Issue{
		ID:        ulid.New(),
		RepoID:    repoID,
		Number:    number,
		Title:     in.Title,
		Body:      in.Body,
		AuthorID:  authorID,
		State:     "open",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if in.MilestoneID != "" {
		issue.MilestoneID = in.MilestoneID
	}

	if err := s.store.Create(ctx, issue); err != nil {
		return nil, err
	}

	// Add labels
	for _, labelID := range in.Labels {
		il := &IssueLabel{
			IssueID:   issue.ID,
			LabelID:   labelID,
			CreatedAt: now,
		}
		s.store.AddLabel(ctx, il)
	}

	// Add assignees
	for _, userID := range in.Assignees {
		ia := &IssueAssignee{
			IssueID:   issue.ID,
			UserID:    userID,
			CreatedAt: now,
		}
		s.store.AddAssignee(ctx, ia)
	}

	return issue, nil
}

// GetByID retrieves an issue by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Issue, error) {
	issue, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}

	// Load labels and assignees
	issue.Assignees, _ = s.store.ListAssignees(ctx, id)

	return issue, nil
}

// GetByNumber retrieves an issue by repo ID and number
func (s *Service) GetByNumber(ctx context.Context, repoID string, number int) (*Issue, error) {
	issue, err := s.store.GetByNumber(ctx, repoID, number)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}

	// Load labels and assignees
	issue.Assignees, _ = s.store.ListAssignees(ctx, issue.ID)

	return issue, nil
}

// Update updates an issue
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Issue, error) {
	issue, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}

	if in.Title != nil {
		issue.Title = *in.Title
	}
	if in.Body != nil {
		issue.Body = *in.Body
	}
	if in.State != nil {
		issue.State = *in.State
		if *in.State == "closed" && issue.ClosedAt == nil {
			now := time.Now()
			issue.ClosedAt = &now
		} else if *in.State == "open" {
			issue.ClosedAt = nil
		}
	}
	if in.StateReason != nil {
		issue.StateReason = *in.StateReason
	}
	if in.MilestoneID != nil {
		issue.MilestoneID = *in.MilestoneID
	}

	if err := s.store.Update(ctx, issue); err != nil {
		return nil, err
	}

	// Update labels if provided
	if in.Labels != nil {
		s.SetLabels(ctx, id, *in.Labels)
	}

	// Update assignees if provided
	if in.Assignees != nil {
		// Remove all existing assignees
		existing, _ := s.store.ListAssignees(ctx, id)
		for _, userID := range existing {
			s.store.RemoveAssignee(ctx, id, userID)
		}
		// Add new assignees
		s.AddAssignees(ctx, id, *in.Assignees)
	}

	return issue, nil
}

// Delete deletes an issue
func (s *Service) Delete(ctx context.Context, id string) error {
	issue, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if issue == nil {
		return ErrNotFound
	}
	return s.store.Delete(ctx, id)
}

// List lists issues for a repository
func (s *Service) List(ctx context.Context, repoID string, opts *ListOpts) ([]*Issue, int, error) {
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

	issues, total, err := s.store.List(ctx, repoID, state, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Load assignees for each issue
	for _, issue := range issues {
		issue.Assignees, _ = s.store.ListAssignees(ctx, issue.ID)
	}

	return issues, total, nil
}

// Close closes an issue
func (s *Service) Close(ctx context.Context, id, userID, reason string) error {
	issue, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if issue == nil {
		return ErrNotFound
	}

	if issue.State == "closed" {
		return nil
	}

	now := time.Now()
	issue.State = "closed"
	issue.StateReason = reason
	issue.ClosedAt = &now
	issue.ClosedByID = userID

	return s.store.Update(ctx, issue)
}

// Reopen reopens an issue
func (s *Service) Reopen(ctx context.Context, id string) error {
	issue, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if issue == nil {
		return ErrNotFound
	}

	if issue.State == "open" {
		return nil
	}

	issue.State = "open"
	issue.StateReason = "reopened"
	issue.ClosedAt = nil
	issue.ClosedByID = ""

	return s.store.Update(ctx, issue)
}

// Lock locks an issue
func (s *Service) Lock(ctx context.Context, id, reason string) error {
	issue, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if issue == nil {
		return ErrNotFound
	}

	issue.IsLocked = true
	issue.LockReason = reason

	return s.store.Update(ctx, issue)
}

// Unlock unlocks an issue
func (s *Service) Unlock(ctx context.Context, id string) error {
	issue, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if issue == nil {
		return ErrNotFound
	}

	issue.IsLocked = false
	issue.LockReason = ""

	return s.store.Update(ctx, issue)
}

// AddLabels adds labels to an issue
func (s *Service) AddLabels(ctx context.Context, id string, labelIDs []string) error {
	issue, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if issue == nil {
		return ErrNotFound
	}

	now := time.Now()
	for _, labelID := range labelIDs {
		il := &IssueLabel{
			IssueID:   id,
			LabelID:   labelID,
			CreatedAt: now,
		}
		s.store.AddLabel(ctx, il)
	}

	return nil
}

// RemoveLabel removes a label from an issue
func (s *Service) RemoveLabel(ctx context.Context, id, labelID string) error {
	return s.store.RemoveLabel(ctx, id, labelID)
}

// SetLabels replaces all labels on an issue
func (s *Service) SetLabels(ctx context.Context, id string, labelIDs []string) error {
	// Remove all existing labels
	existing, _ := s.store.ListLabels(ctx, id)
	for _, labelID := range existing {
		s.store.RemoveLabel(ctx, id, labelID)
	}

	// Add new labels
	return s.AddLabels(ctx, id, labelIDs)
}

// AddAssignees adds assignees to an issue
func (s *Service) AddAssignees(ctx context.Context, id string, userIDs []string) error {
	issue, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if issue == nil {
		return ErrNotFound
	}

	now := time.Now()
	for _, userID := range userIDs {
		ia := &IssueAssignee{
			IssueID:   id,
			UserID:    userID,
			CreatedAt: now,
		}
		s.store.AddAssignee(ctx, ia)
	}

	return nil
}

// RemoveAssignees removes assignees from an issue
func (s *Service) RemoveAssignees(ctx context.Context, id string, userIDs []string) error {
	for _, userID := range userIDs {
		s.store.RemoveAssignee(ctx, id, userID)
	}
	return nil
}

// AddComment adds a comment to an issue
func (s *Service) AddComment(ctx context.Context, issueID, userID, body string) (*Comment, error) {
	issue, err := s.store.GetByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}

	if issue.IsLocked {
		return nil, ErrLocked
	}

	now := time.Now()
	comment := &Comment{
		ID:         ulid.New(),
		TargetType: "issue",
		TargetID:   issueID,
		UserID:     userID,
		Body:       body,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// TODO: Save comment to store
	// Increment comment count
	issue.CommentCount++
	s.store.Update(ctx, issue)

	return comment, nil
}

// UpdateComment updates a comment
func (s *Service) UpdateComment(ctx context.Context, commentID, body string) (*Comment, error) {
	// TODO: Implement comment update
	return nil, nil
}

// DeleteComment deletes a comment
func (s *Service) DeleteComment(ctx context.Context, commentID string) error {
	// TODO: Implement comment deletion
	return nil
}

// ListComments lists comments for an issue
func (s *Service) ListComments(ctx context.Context, issueID string) ([]*Comment, error) {
	// TODO: Implement comment listing
	return nil, nil
}
