package branches

import (
	"context"
	"fmt"

	"github.com/go-mizu/blueprints/githome/feature/repos"
)

// Service implements the branches API
type Service struct {
	store     Store
	repoStore repos.Store
	baseURL   string
}

// NewService creates a new branches service
func NewService(store Store, repoStore repos.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		baseURL:   baseURL,
	}
}

// List returns branches for a repository
func (s *Service) List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Branch, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Would integrate with git to list branches
	// For now return default branch
	return []*Branch{
		{
			Name: r.DefaultBranch,
			Commit: &CommitRef{
				SHA: "HEAD",
				URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/HEAD", s.baseURL, owner, repo),
			},
			Protected: false,
		},
	}, nil
}

// Get retrieves a branch by name
func (s *Service) Get(ctx context.Context, owner, repo, branch string) (*Branch, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Would integrate with git to get branch
	protection, _ := s.store.GetProtection(ctx, r.ID, branch)

	return &Branch{
		Name: branch,
		Commit: &CommitRef{
			SHA: "HEAD",
			URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/HEAD", s.baseURL, owner, repo),
		},
		Protected: protection != nil && protection.Enabled,
	}, nil
}

// Rename renames a branch
func (s *Service) Rename(ctx context.Context, owner, repo, branch, newName string) (*Branch, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Would integrate with git to rename branch
	return &Branch{
		Name: newName,
		Commit: &CommitRef{
			SHA: "HEAD",
			URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/HEAD", s.baseURL, owner, repo),
		},
	}, nil
}

// GetProtection retrieves branch protection settings
func (s *Service) GetProtection(ctx context.Context, owner, repo, branch string) (*BranchProtection, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	protection, err := s.store.GetProtection(ctx, r.ID, branch)
	if err != nil {
		return nil, err
	}
	if protection == nil {
		return nil, ErrNotFound
	}

	s.populateProtectionURLs(protection, owner, repo, branch)
	return protection, nil
}

// UpdateProtection updates branch protection settings
func (s *Service) UpdateProtection(ctx context.Context, owner, repo, branch string, in *UpdateProtectionIn) (*BranchProtection, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	protection := &BranchProtection{
		Enabled: true,
		EnforceAdmins: &EnforceAdmins{
			Enabled: in.EnforceAdmins,
		},
		RequiredLinearHistory: &EnabledSetting{
			Enabled: in.RequiredLinearHistory,
		},
		AllowDeletions: &EnabledSetting{
			Enabled: in.AllowDeletions,
		},
		RequiredConversationResolution: &EnabledSetting{
			Enabled: in.RequiredConversationResolution,
		},
	}

	if in.AllowForcePushes != nil {
		protection.AllowForcePushes = &EnabledSetting{
			Enabled: *in.AllowForcePushes,
		}
	}

	if in.RequiredStatusChecks != nil {
		protection.RequiredStatusChecks = &RequiredStatusChecks{
			Strict:   in.RequiredStatusChecks.Strict,
			Contexts: in.RequiredStatusChecks.Contexts,
		}
		if in.RequiredStatusChecks.Checks != nil {
			for _, c := range in.RequiredStatusChecks.Checks {
				protection.RequiredStatusChecks.Checks = append(protection.RequiredStatusChecks.Checks, &Check{
					Context: c.Context,
					AppID:   c.AppID,
				})
			}
		}
	}

	if in.RequiredPullRequestReviews != nil {
		protection.RequiredPullRequestReviews = &RequiredPullRequestReviews{
			DismissStaleReviews:          in.RequiredPullRequestReviews.DismissStaleReviews,
			RequireCodeOwnerReviews:      in.RequiredPullRequestReviews.RequireCodeOwnerReviews,
			RequiredApprovingReviewCount: in.RequiredPullRequestReviews.RequiredApprovingReviewCount,
			RequireLastPushApproval:      in.RequiredPullRequestReviews.RequireLastPushApproval,
		}
	}

	if in.Restrictions != nil {
		protection.Restrictions = &BranchRestrictions{
			Users: []User{},
			Teams: []Team{},
			Apps:  []App{},
		}
	}

	if err := s.store.SetProtection(ctx, r.ID, branch, protection); err != nil {
		return nil, err
	}

	s.populateProtectionURLs(protection, owner, repo, branch)
	return protection, nil
}

// DeleteProtection removes branch protection
func (s *Service) DeleteProtection(ctx context.Context, owner, repo, branch string) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	return s.store.DeleteProtection(ctx, r.ID, branch)
}

// GetRequiredStatusChecks retrieves required status checks
func (s *Service) GetRequiredStatusChecks(ctx context.Context, owner, repo, branch string) (*RequiredStatusChecks, error) {
	protection, err := s.GetProtection(ctx, owner, repo, branch)
	if err != nil {
		return nil, err
	}
	if protection.RequiredStatusChecks == nil {
		return nil, ErrNotFound
	}
	return protection.RequiredStatusChecks, nil
}

// UpdateRequiredStatusChecks updates required status checks
func (s *Service) UpdateRequiredStatusChecks(ctx context.Context, owner, repo, branch string, in *RequiredStatusChecksIn) (*RequiredStatusChecks, error) {
	protection, err := s.GetProtection(ctx, owner, repo, branch)
	if err != nil {
		if err == ErrNotFound {
			protection = &BranchProtection{Enabled: true}
		} else {
			return nil, err
		}
	}

	protection.RequiredStatusChecks = &RequiredStatusChecks{
		Strict:   in.Strict,
		Contexts: in.Contexts,
	}
	if in.Checks != nil {
		for _, c := range in.Checks {
			protection.RequiredStatusChecks.Checks = append(protection.RequiredStatusChecks.Checks, &Check{
				Context: c.Context,
				AppID:   c.AppID,
			})
		}
	}

	r, _ := s.repoStore.GetByFullName(ctx, owner, repo)
	if err := s.store.SetProtection(ctx, r.ID, branch, protection); err != nil {
		return nil, err
	}

	s.populateStatusChecksURLs(protection.RequiredStatusChecks, owner, repo, branch)
	return protection.RequiredStatusChecks, nil
}

// RemoveRequiredStatusChecks removes required status checks
func (s *Service) RemoveRequiredStatusChecks(ctx context.Context, owner, repo, branch string) error {
	protection, err := s.GetProtection(ctx, owner, repo, branch)
	if err != nil {
		return err
	}

	protection.RequiredStatusChecks = nil
	r, _ := s.repoStore.GetByFullName(ctx, owner, repo)
	return s.store.SetProtection(ctx, r.ID, branch, protection)
}

// GetRequiredSignatures retrieves required signatures setting
func (s *Service) GetRequiredSignatures(ctx context.Context, owner, repo, branch string) (*EnabledSetting, error) {
	protection, err := s.GetProtection(ctx, owner, repo, branch)
	if err != nil {
		return nil, err
	}
	if protection.RequiredSignatures == nil {
		return &EnabledSetting{Enabled: false}, nil
	}
	return protection.RequiredSignatures, nil
}

// CreateRequiredSignatures enables required signatures
func (s *Service) CreateRequiredSignatures(ctx context.Context, owner, repo, branch string) (*EnabledSetting, error) {
	protection, err := s.GetProtection(ctx, owner, repo, branch)
	if err != nil {
		if err == ErrNotFound {
			protection = &BranchProtection{Enabled: true}
		} else {
			return nil, err
		}
	}

	protection.RequiredSignatures = &EnabledSetting{Enabled: true}
	r, _ := s.repoStore.GetByFullName(ctx, owner, repo)
	if err := s.store.SetProtection(ctx, r.ID, branch, protection); err != nil {
		return nil, err
	}

	return protection.RequiredSignatures, nil
}

// DeleteRequiredSignatures disables required signatures
func (s *Service) DeleteRequiredSignatures(ctx context.Context, owner, repo, branch string) error {
	protection, err := s.GetProtection(ctx, owner, repo, branch)
	if err != nil {
		return err
	}

	protection.RequiredSignatures = nil
	r, _ := s.repoStore.GetByFullName(ctx, owner, repo)
	return s.store.SetProtection(ctx, r.ID, branch, protection)
}

// populateProtectionURLs fills in URL fields for branch protection
func (s *Service) populateProtectionURLs(p *BranchProtection, owner, repo, branch string) {
	p.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/branches/%s/protection", s.baseURL, owner, repo, branch)

	if p.RequiredStatusChecks != nil {
		s.populateStatusChecksURLs(p.RequiredStatusChecks, owner, repo, branch)
	}
	if p.EnforceAdmins != nil {
		p.EnforceAdmins.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/branches/%s/protection/enforce_admins", s.baseURL, owner, repo, branch)
	}
	if p.RequiredPullRequestReviews != nil {
		p.RequiredPullRequestReviews.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/branches/%s/protection/required_pull_request_reviews", s.baseURL, owner, repo, branch)
	}
	if p.Restrictions != nil {
		p.Restrictions.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/branches/%s/protection/restrictions", s.baseURL, owner, repo, branch)
		p.Restrictions.UsersURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/branches/%s/protection/restrictions/users", s.baseURL, owner, repo, branch)
		p.Restrictions.TeamsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/branches/%s/protection/restrictions/teams", s.baseURL, owner, repo, branch)
		p.Restrictions.AppsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/branches/%s/protection/restrictions/apps", s.baseURL, owner, repo, branch)
	}
}

// populateStatusChecksURLs fills in URL fields for status checks
func (s *Service) populateStatusChecksURLs(sc *RequiredStatusChecks, owner, repo, branch string) {
	sc.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/branches/%s/protection/required_status_checks", s.baseURL, owner, repo, branch)
	sc.ContextsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/branches/%s/protection/required_status_checks/contexts", s.baseURL, owner, repo, branch)
}
