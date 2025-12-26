package issues

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/projects"
	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrNotFound = errors.New("issue not found")
)

// Service implements the issues API.
type Service struct {
	store    Store
	projects projects.API
}

// NewService creates a new issues service.
func NewService(store Store, projects projects.API) *Service {
	return &Service{store: store, projects: projects}
}

func (s *Service) Create(ctx context.Context, projectID, creatorID string, in *CreateIn) (*Issue, error) {
	// Get project to generate issue key
	project, err := s.projects.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Get next issue number
	number, err := s.projects.NextIssueNumber(ctx, projectID)
	if err != nil {
		return nil, err
	}

	issueType := in.Type
	if issueType == "" {
		issueType = TypeTask
	}

	status := in.Status
	if status == "" {
		status = StatusBacklog
	}

	priority := in.Priority
	if priority == "" {
		priority = PriorityNone
	}

	now := time.Now()
	issue := &Issue{
		ID:          ulid.New(),
		ProjectID:   projectID,
		Number:      number,
		Key:         fmt.Sprintf("%s-%d", project.Key, number),
		Title:       in.Title,
		Description: in.Description,
		Type:        issueType,
		Status:      status,
		Priority:    priority,
		ParentID:    in.ParentID,
		CreatorID:   creatorID,
		SprintID:    in.SprintID,
		DueDate:     in.DueDate,
		Estimate:    in.Estimate,
		Position:    0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Create(ctx, issue); err != nil {
		return nil, err
	}

	// Add assignees
	for _, userID := range in.AssigneeIDs {
		if err := s.store.AddAssignee(ctx, issue.ID, userID); err != nil {
			return nil, err
		}
	}

	// Add labels
	for _, labelID := range in.LabelIDs {
		if err := s.store.AddLabel(ctx, issue.ID, labelID); err != nil {
			return nil, err
		}
	}

	issue.AssigneeIDs = in.AssigneeIDs
	issue.LabelIDs = in.LabelIDs

	return issue, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Issue, error) {
	issue, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}

	// Load relations
	issue.AssigneeIDs, _ = s.store.GetAssignees(ctx, id)
	issue.LabelIDs, _ = s.store.GetLabels(ctx, id)

	return issue, nil
}

func (s *Service) GetByKey(ctx context.Context, key string) (*Issue, error) {
	issue, err := s.store.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}

	// Load relations
	issue.AssigneeIDs, _ = s.store.GetAssignees(ctx, issue.ID)
	issue.LabelIDs, _ = s.store.GetLabels(ctx, issue.ID)

	return issue, nil
}

func (s *Service) ListByProject(ctx context.Context, projectID string, filter *Filter) ([]*Issue, error) {
	issues, err := s.store.ListByProject(ctx, projectID, filter)
	if err != nil {
		return nil, err
	}

	// Load relations for each issue
	for _, issue := range issues {
		issue.AssigneeIDs, _ = s.store.GetAssignees(ctx, issue.ID)
		issue.LabelIDs, _ = s.store.GetLabels(ctx, issue.ID)
	}

	return issues, nil
}

func (s *Service) ListByStatus(ctx context.Context, projectID string) (map[string][]*Issue, error) {
	result, err := s.store.ListByStatus(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Load relations for each issue
	for _, issues := range result {
		for _, issue := range issues {
			issue.AssigneeIDs, _ = s.store.GetAssignees(ctx, issue.ID)
			issue.LabelIDs, _ = s.store.GetLabels(ctx, issue.ID)
		}
	}

	return result, nil
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Issue, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Service) Move(ctx context.Context, id string, in *MoveIn) (*Issue, error) {
	if err := s.store.UpdatePosition(ctx, id, in.Status, in.Position); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) AddAssignee(ctx context.Context, issueID, userID string) error {
	return s.store.AddAssignee(ctx, issueID, userID)
}

func (s *Service) RemoveAssignee(ctx context.Context, issueID, userID string) error {
	return s.store.RemoveAssignee(ctx, issueID, userID)
}

func (s *Service) AddLabel(ctx context.Context, issueID, labelID string) error {
	return s.store.AddLabel(ctx, issueID, labelID)
}

func (s *Service) RemoveLabel(ctx context.Context, issueID, labelID string) error {
	return s.store.RemoveLabel(ctx, issueID, labelID)
}

func (s *Service) Search(ctx context.Context, projectID, query string, limit int) ([]*Issue, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.store.Search(ctx, projectID, query, limit)
}
