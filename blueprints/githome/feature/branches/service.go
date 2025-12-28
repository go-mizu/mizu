package branches

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	pkggit "github.com/go-mizu/blueprints/githome/pkg/git"
)

// Service implements the branches API
type Service struct {
	store     Store
	repoStore repos.Store
	baseURL   string
	reposDir  string
}

// NewService creates a new branches service
func NewService(store Store, repoStore repos.Store, baseURL, reposDir string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		baseURL:   baseURL,
		reposDir:  reposDir,
	}
}

// getRepoPath returns the filesystem path for a repository
func (s *Service) getRepoPath(owner, repo string) string {
	return filepath.Join(s.reposDir, owner, repo+".git")
}

// openRepo opens a git repository
func (s *Service) openRepo(owner, repo string) (*pkggit.Repository, error) {
	path := s.getRepoPath(owner, repo)
	return pkggit.Open(path)
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

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}
	if opts.Page < 1 {
		opts.Page = 1
	}

	// Try to open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			// No git repo yet, return default branch placeholder
			return []*Branch{
				{
					Name: r.DefaultBranch,
					Commit: &CommitRef{
						SHA: "",
						URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/HEAD", s.baseURL, owner, repo),
					},
					Protected: false,
				},
			}, nil
		}
		return nil, err
	}

	// List all branch refs from git
	refs, err := gitRepo.ListRefs("heads")
	if err != nil {
		return nil, err
	}

	// Convert to branches
	branches := make([]*Branch, 0, len(refs))
	for _, ref := range refs {
		branchName := strings.TrimPrefix(ref.Name, "refs/heads/")

		// Check if protected
		protection, _ := s.store.GetProtection(ctx, r.ID, branchName)
		isProtected := protection != nil && protection.Enabled

		// Filter by protected if requested
		if opts.Protected != nil {
			if *opts.Protected != isProtected {
				continue
			}
		}

		branches = append(branches, &Branch{
			Name: branchName,
			Commit: &CommitRef{
				SHA: ref.SHA,
				URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repo, ref.SHA),
			},
			Protected: isProtected,
		})
	}

	// Apply pagination
	start := (opts.Page - 1) * opts.PerPage
	end := start + opts.PerPage
	if start > len(branches) {
		return []*Branch{}, nil
	}
	if end > len(branches) {
		end = len(branches)
	}

	return branches[start:end], nil
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

	protection, _ := s.store.GetProtection(ctx, r.ID, branch)

	// Try to open git repository
	if s.reposDir == "" {
		// No reposDir configured - return placeholder
		return &Branch{
			Name: branch,
			Commit: &CommitRef{
				SHA: "",
				URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/HEAD", s.baseURL, owner, repo),
			},
			Protected: protection != nil && protection.Enabled,
		}, nil
	}

	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			// No git repo - return placeholder
			return &Branch{
				Name: branch,
				Commit: &CommitRef{
					SHA: "",
					URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/HEAD", s.baseURL, owner, repo),
				},
				Protected: protection != nil && protection.Enabled,
			}, nil
		}
		return nil, err
	}

	// Get the branch ref
	ref, err := gitRepo.GetRef("refs/heads/" + branch)
	if err != nil {
		if err == pkggit.ErrRefNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &Branch{
		Name: branch,
		Commit: &CommitRef{
			SHA: ref.SHA,
			URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repo, ref.SHA),
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

	// Try to open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Get the old branch ref
	oldRef, err := gitRepo.GetRef("refs/heads/" + branch)
	if err != nil {
		if err == pkggit.ErrRefNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Check if branch is protected
	protection, _ := s.store.GetProtection(ctx, r.ID, branch)
	if protection != nil && protection.Enabled {
		return nil, ErrProtected
	}

	// Check if new branch already exists
	_, err = gitRepo.GetRef("refs/heads/" + newName)
	if err == nil {
		return nil, ErrBranchExists
	}
	if err != pkggit.ErrRefNotFound {
		return nil, err
	}

	// Create new branch ref pointing to same commit
	if err := gitRepo.CreateRef("refs/heads/"+newName, oldRef.SHA); err != nil {
		return nil, err
	}

	// Delete old branch ref
	if err := gitRepo.DeleteRef("refs/heads/" + branch); err != nil {
		// Try to clean up the new ref if delete fails
		_ = gitRepo.DeleteRef("refs/heads/" + newName)
		return nil, err
	}

	// Move protection settings if they exist
	if protection != nil {
		_ = s.store.SetProtection(ctx, r.ID, newName, protection)
		_ = s.store.DeleteProtection(ctx, r.ID, branch)
	}

	return &Branch{
		Name: newName,
		Commit: &CommitRef{
			SHA: oldRef.SHA,
			URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repo, oldRef.SHA),
		},
		Protected: protection != nil && protection.Enabled,
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
