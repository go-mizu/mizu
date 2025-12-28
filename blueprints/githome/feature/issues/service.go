package issues

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Service implements the issues API
type Service struct {
	store     Store
	repoStore repos.Store
	userStore users.Store
	baseURL   string
}

// NewService creates a new issues service
func NewService(store Store, repoStore repos.Store, userStore users.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		userStore: userStore,
		baseURL:   baseURL,
	}
}

// Create creates a new issue
func (s *Service) Create(ctx context.Context, owner, repo string, creatorID int64, in *CreateIn) (*Issue, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	creator, err := s.userStore.GetByID(ctx, creatorID)
	if err != nil {
		return nil, err
	}
	if creator == nil {
		return nil, users.ErrNotFound
	}

	number, err := s.store.NextNumber(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	issue := &Issue{
		Number:            number,
		State:             "open",
		Title:             in.Title,
		Body:              in.Body,
		User:              creator.ToSimple(),
		Labels:            []*Label{},
		Assignees:         []*users.SimpleUser{},
		Comments:          0,
		CreatedAt:         now,
		UpdatedAt:         now,
		RepoID:            r.ID,
		CreatorID:         creatorID,
		AuthorAssociation: s.getAuthorAssociation(ctx, r, creatorID),
	}

	if err := s.store.Create(ctx, issue); err != nil {
		return nil, err
	}

	// Add assignees
	if in.Assignee != "" {
		in.Assignees = append([]string{in.Assignee}, in.Assignees...)
	}
	for _, login := range in.Assignees {
		user, err := s.userStore.GetByLogin(ctx, login)
		if err != nil {
			continue
		}
		if user != nil {
			if err := s.store.AddAssignee(ctx, issue.ID, user.ID); err == nil {
				issue.Assignees = append(issue.Assignees, user.ToSimple())
			}
		}
	}
	if len(issue.Assignees) > 0 {
		issue.Assignee = issue.Assignees[0]
	}

	// Increment repo open issues count
	if err := s.repoStore.IncrementOpenIssues(ctx, r.ID, 1); err != nil {
		return nil, err
	}

	s.populateURLs(issue, owner, repo)
	return issue, nil
}

// Get retrieves an issue by number
func (s *Service) Get(ctx context.Context, owner, repo string, number int) (*Issue, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	issue, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}

	s.populateURLs(issue, owner, repo)
	return issue, nil
}

// GetByID retrieves an issue by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*Issue, error) {
	issue, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}

	// Get repo info for URLs
	r, err := s.repoStore.GetByID(ctx, issue.RepoID)
	if err != nil {
		return nil, err
	}
	if r != nil {
		s.populateURLs(issue, r.FullName, "")
	}
	return issue, nil
}

// Update updates an issue
func (s *Service) Update(ctx context.Context, owner, repo string, number int, in *UpdateIn) (*Issue, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	issue, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}

	// Track state change for counter updates
	oldState := issue.State
	newState := oldState
	if in.State != nil {
		newState = *in.State
	}

	if err := s.store.Update(ctx, issue.ID, in); err != nil {
		return nil, err
	}

	// Update repo open issues count if state changed
	if oldState != newState {
		if newState == "closed" && oldState == "open" {
			if err := s.repoStore.IncrementOpenIssues(ctx, r.ID, -1); err != nil {
				return nil, err
			}
		} else if newState == "open" && oldState == "closed" {
			if err := s.repoStore.IncrementOpenIssues(ctx, r.ID, 1); err != nil {
				return nil, err
			}
		}
	}

	return s.Get(ctx, owner, repo, number)
}

// Lock locks an issue
func (s *Service) Lock(ctx context.Context, owner, repo string, number int, reason string) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	issue, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return err
	}
	if issue == nil {
		return ErrNotFound
	}

	return s.store.SetLocked(ctx, issue.ID, true, reason)
}

// Unlock unlocks an issue
func (s *Service) Unlock(ctx context.Context, owner, repo string, number int) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	issue, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return err
	}
	if issue == nil {
		return ErrNotFound
	}

	return s.store.SetLocked(ctx, issue.ID, false, "")
}

// ListForRepo returns issues for a repository
func (s *Service) ListForRepo(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Issue, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30, State: "open"}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}
	if opts.State == "" {
		opts.State = "open"
	}

	issues, err := s.store.List(ctx, r.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, issue := range issues {
		s.populateURLs(issue, owner, repo)
	}
	return issues, nil
}

// ListForOrg returns issues for an organization
func (s *Service) ListForOrg(ctx context.Context, org string, opts *ListOpts) ([]*Issue, error) {
	if opts == nil {
		opts = &ListOpts{PerPage: 30, State: "open"}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	// For org, we need to find org ID first - simplified here
	return []*Issue{}, nil
}

// ListForUser returns issues assigned to/created by the authenticated user
func (s *Service) ListForUser(ctx context.Context, userID int64, opts *ListOpts) ([]*Issue, error) {
	if opts == nil {
		opts = &ListOpts{PerPage: 30, State: "open"}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.ListForUser(ctx, userID, opts)
}

// ListAssignees returns users who can be assigned to issues
func (s *Service) ListAssignees(ctx context.Context, owner, repo string) ([]*users.SimpleUser, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Return collaborators - simplified, return empty list
	return []*users.SimpleUser{}, nil
}

// CheckAssignee checks if a user can be assigned
func (s *Service) CheckAssignee(ctx context.Context, owner, repo, assignee string) (bool, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return false, err
	}
	if r == nil {
		return false, repos.ErrNotFound
	}

	user, err := s.userStore.GetByLogin(ctx, assignee)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, nil
	}

	// Simplified: any valid user can be assigned
	return true, nil
}

// AddAssignees adds assignees to an issue
func (s *Service) AddAssignees(ctx context.Context, owner, repo string, number int, assignees []string) (*Issue, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	issue, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}

	for _, login := range assignees {
		user, err := s.userStore.GetByLogin(ctx, login)
		if err != nil {
			continue
		}
		if user != nil {
			_ = s.store.AddAssignee(ctx, issue.ID, user.ID)
		}
	}

	return s.Get(ctx, owner, repo, number)
}

// RemoveAssignees removes assignees from an issue
func (s *Service) RemoveAssignees(ctx context.Context, owner, repo string, number int, assignees []string) (*Issue, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	issue, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}

	for _, login := range assignees {
		user, err := s.userStore.GetByLogin(ctx, login)
		if err != nil {
			continue
		}
		if user != nil {
			_ = s.store.RemoveAssignee(ctx, issue.ID, user.ID)
		}
	}

	return s.Get(ctx, owner, repo, number)
}

// ListEvents returns events for an issue
func (s *Service) ListEvents(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*IssueEvent, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	issue, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	return s.store.ListEvents(ctx, issue.ID, opts)
}

// CreateEvent creates an event (internal use)
func (s *Service) CreateEvent(ctx context.Context, issueID, actorID int64, eventType string, data map[string]interface{}) (*IssueEvent, error) {
	actor, err := s.userStore.GetByID(ctx, actorID)
	if err != nil {
		return nil, err
	}

	event := &IssueEvent{
		Actor:     actor.ToSimple(),
		Event:     eventType,
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateEvent(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

// getAuthorAssociation determines the relationship between user and repo
func (s *Service) getAuthorAssociation(ctx context.Context, r *repos.Repository, userID int64) string {
	if r.OwnerID == userID {
		return "OWNER"
	}
	// Simplified - would check collaborators, org membership, etc.
	return "NONE"
}

// populateURLs fills in the URL fields for an issue
func (s *Service) populateURLs(issue *Issue, owner, repo string) {
	issue.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Issue:%d", issue.ID)))
	issue.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/issues/%d", s.baseURL, owner, repo, issue.Number)
	issue.RepositoryURL = fmt.Sprintf("%s/api/v3/repos/%s/%s", s.baseURL, owner, repo)
	issue.LabelsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/issues/%d/labels{/name}", s.baseURL, owner, repo, issue.Number)
	issue.CommentsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/issues/%d/comments", s.baseURL, owner, repo, issue.Number)
	issue.EventsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/issues/%d/events", s.baseURL, owner, repo, issue.Number)
	issue.HTMLURL = fmt.Sprintf("%s/%s/%s/issues/%d", s.baseURL, owner, repo, issue.Number)
}
