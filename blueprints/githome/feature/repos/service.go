package repos

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Service implements the repos API
type Service struct {
	store     Store
	userStore users.Store
	orgStore  orgs.Store
	baseURL   string
}

// NewService creates a new repos service
func NewService(store Store, userStore users.Store, orgStore orgs.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		userStore: userStore,
		orgStore:  orgStore,
		baseURL:   baseURL,
	}
}

// Create creates a new repository for a user
func (s *Service) Create(ctx context.Context, ownerID int64, in *CreateIn) (*Repository, error) {
	// Check if repo exists
	existing, err := s.store.GetByOwnerAndName(ctx, ownerID, in.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrRepoExists
	}

	owner, err := s.userStore.GetByID(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if owner == nil {
		return nil, users.ErrNotFound
	}

	now := time.Now()
	visibility := in.Visibility
	if visibility == "" {
		if in.Private {
			visibility = "private"
		} else {
			visibility = "public"
		}
	}

	repo := &Repository{
		Name:                in.Name,
		FullName:            fmt.Sprintf("%s/%s", owner.Login, in.Name),
		Description:         in.Description,
		Homepage:            in.Homepage,
		Private:             visibility == "private",
		Visibility:          visibility,
		OwnerID:             ownerID,
		OwnerType:           "User",
		Owner:               owner.ToSimple(),
		HasIssues:           defaultBool(in.HasIssues, true),
		HasProjects:         defaultBool(in.HasProjects, true),
		HasWiki:             defaultBool(in.HasWiki, true),
		HasDiscussions:      defaultBool(in.HasDiscussions, false),
		HasDownloads:        true,
		IsTemplate:          in.IsTemplate,
		DefaultBranch:       "main",
		AllowSquashMerge:    defaultBool(in.AllowSquashMerge, true),
		AllowMergeCommit:    defaultBool(in.AllowMergeCommit, true),
		AllowRebaseMerge:    defaultBool(in.AllowRebaseMerge, true),
		AllowAutoMerge:      defaultBool(in.AllowAutoMerge, false),
		DeleteBranchOnMerge: defaultBool(in.DeleteBranchOnMerge, false),
		AllowForking:        true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.store.Create(ctx, repo); err != nil {
		return nil, err
	}

	s.populateURLs(repo, owner.Login)
	return repo, nil
}

// CreateForOrg creates a new repository for an organization
func (s *Service) CreateForOrg(ctx context.Context, orgLogin string, in *CreateIn) (*Repository, error) {
	org, err := s.orgStore.GetByLogin(ctx, orgLogin)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, orgs.ErrNotFound
	}

	// Check if repo exists
	existing, err := s.store.GetByOwnerAndName(ctx, org.ID, in.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrRepoExists
	}

	now := time.Now()
	visibility := in.Visibility
	if visibility == "" {
		if in.Private {
			visibility = "private"
		} else {
			visibility = "public"
		}
	}

	repo := &Repository{
		Name:                in.Name,
		FullName:            fmt.Sprintf("%s/%s", org.Login, in.Name),
		Description:         in.Description,
		Homepage:            in.Homepage,
		Private:             visibility == "private",
		Visibility:          visibility,
		OwnerID:             org.ID,
		OwnerType:           "Organization",
		HasIssues:           defaultBool(in.HasIssues, true),
		HasProjects:         defaultBool(in.HasProjects, true),
		HasWiki:             defaultBool(in.HasWiki, true),
		HasDiscussions:      defaultBool(in.HasDiscussions, false),
		HasDownloads:        true,
		IsTemplate:          in.IsTemplate,
		DefaultBranch:       "main",
		AllowSquashMerge:    defaultBool(in.AllowSquashMerge, true),
		AllowMergeCommit:    defaultBool(in.AllowMergeCommit, true),
		AllowRebaseMerge:    defaultBool(in.AllowRebaseMerge, true),
		AllowAutoMerge:      defaultBool(in.AllowAutoMerge, false),
		DeleteBranchOnMerge: defaultBool(in.DeleteBranchOnMerge, false),
		AllowForking:        true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.store.Create(ctx, repo); err != nil {
		return nil, err
	}

	s.populateURLs(repo, org.Login)
	return repo, nil
}

// Get retrieves a repository by owner and name
func (s *Service) Get(ctx context.Context, owner, repoName string) (*Repository, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}
	s.populateURLs(repo, owner)
	return repo, nil
}

// GetByID retrieves a repository by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*Repository, error) {
	repo, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}

	// Get owner login
	var ownerLogin string
	if repo.OwnerType == "User" {
		user, _ := s.userStore.GetByID(ctx, repo.OwnerID)
		if user != nil {
			ownerLogin = user.Login
		}
	} else {
		org, _ := s.orgStore.GetByID(ctx, repo.OwnerID)
		if org != nil {
			ownerLogin = org.Login
		}
	}

	s.populateURLs(repo, ownerLogin)
	return repo, nil
}

// Update updates a repository
func (s *Service) Update(ctx context.Context, owner, repoName string, in *UpdateIn) (*Repository, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}

	if err := s.store.Update(ctx, repo.ID, in); err != nil {
		return nil, err
	}

	// If name changed, get by new name
	if in.Name != nil {
		return s.Get(ctx, owner, *in.Name)
	}
	return s.Get(ctx, owner, repoName)
}

// Delete removes a repository
func (s *Service) Delete(ctx context.Context, owner, repoName string) error {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return err
	}
	if repo == nil {
		return ErrNotFound
	}
	return s.store.Delete(ctx, repo.ID)
}

// Transfer transfers a repository to a new owner
func (s *Service) Transfer(ctx context.Context, owner, repoName string, in *TransferIn) (*Repository, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}

	// Find new owner
	var newOwnerID int64
	var newOwnerType string
	newOwnerLogin := in.NewOwner

	// Try user first
	user, err := s.userStore.GetByLogin(ctx, in.NewOwner)
	if err != nil {
		return nil, err
	}
	if user != nil {
		newOwnerID = user.ID
		newOwnerType = "User"
	} else {
		// Try org
		org, err := s.orgStore.GetByLogin(ctx, in.NewOwner)
		if err != nil {
			return nil, err
		}
		if org == nil {
			return nil, ErrNotFound
		}
		newOwnerID = org.ID
		newOwnerType = "Organization"
	}

	newName := repoName
	if in.NewName != nil {
		newName = *in.NewName
	}

	updateIn := &UpdateIn{
		Name: &newName,
	}
	repo.OwnerID = newOwnerID
	repo.OwnerType = newOwnerType
	repo.FullName = fmt.Sprintf("%s/%s", newOwnerLogin, newName)

	if err := s.store.Update(ctx, repo.ID, updateIn); err != nil {
		return nil, err
	}

	return s.Get(ctx, newOwnerLogin, newName)
}

// ListForUser returns repositories for a user
func (s *Service) ListForUser(ctx context.Context, username string, opts *ListOpts) ([]*Repository, error) {
	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	repos, err := s.store.ListByOwner(ctx, user.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, r := range repos {
		s.populateURLs(r, username)
	}
	return repos, nil
}

// ListForOrg returns repositories for an organization
func (s *Service) ListForOrg(ctx context.Context, orgLogin string, opts *ListOpts) ([]*Repository, error) {
	org, err := s.orgStore.GetByLogin(ctx, orgLogin)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, orgs.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	repos, err := s.store.ListByOwner(ctx, org.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, r := range repos {
		s.populateURLs(r, orgLogin)
	}
	return repos, nil
}

// ListForAuthenticatedUser returns repositories for the authenticated user
func (s *Service) ListForAuthenticatedUser(ctx context.Context, userID int64, opts *ListOpts) ([]*Repository, error) {
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	repos, err := s.store.ListByOwner(ctx, userID, opts)
	if err != nil {
		return nil, err
	}

	for _, r := range repos {
		s.populateURLs(r, user.Login)
	}
	return repos, nil
}

// ListForks returns forks of a repository
func (s *Service) ListForks(ctx context.Context, owner, repoName string, opts *ListOpts) ([]*Repository, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	return s.store.ListForks(ctx, repo.ID, opts)
}

// CreateFork creates a fork of a repository
func (s *Service) CreateFork(ctx context.Context, owner, repoName string, targetOrg, targetName string) (*Repository, error) {
	// Get source repo
	source, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, ErrNotFound
	}

	// Determine target owner
	var targetOwnerID int64
	var targetOwnerType, targetOwnerLogin string

	if targetOrg != "" {
		org, err := s.orgStore.GetByLogin(ctx, targetOrg)
		if err != nil {
			return nil, err
		}
		if org == nil {
			return nil, orgs.ErrNotFound
		}
		targetOwnerID = org.ID
		targetOwnerType = "Organization"
		targetOwnerLogin = org.Login
	} else {
		return nil, ErrAccessDenied // Need to specify target
	}

	if targetName == "" {
		targetName = source.Name
	}

	// Create fork
	now := time.Now()
	fork := &Repository{
		Name:                targetName,
		FullName:            fmt.Sprintf("%s/%s", targetOwnerLogin, targetName),
		Description:         source.Description,
		Private:             source.Private,
		Visibility:          source.Visibility,
		OwnerID:             targetOwnerID,
		OwnerType:           targetOwnerType,
		Fork:                true,
		HasIssues:           source.HasIssues,
		HasProjects:         source.HasProjects,
		HasWiki:             source.HasWiki,
		HasDiscussions:      source.HasDiscussions,
		HasDownloads:        source.HasDownloads,
		DefaultBranch:       source.DefaultBranch,
		AllowSquashMerge:    source.AllowSquashMerge,
		AllowMergeCommit:    source.AllowMergeCommit,
		AllowRebaseMerge:    source.AllowRebaseMerge,
		AllowAutoMerge:      source.AllowAutoMerge,
		DeleteBranchOnMerge: source.DeleteBranchOnMerge,
		AllowForking:        true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.store.Create(ctx, fork); err != nil {
		return nil, err
	}

	// Increment source fork count
	if err := s.store.IncrementForks(ctx, source.ID, 1); err != nil {
		return nil, err
	}

	s.populateURLs(fork, targetOwnerLogin)
	return fork, nil
}

// ListLanguages returns language statistics
func (s *Service) ListLanguages(ctx context.Context, owner, repoName string) (map[string]int, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}
	return s.store.GetLanguages(ctx, repo.ID)
}

// ListTopics returns repository topics
func (s *Service) ListTopics(ctx context.Context, owner, repoName string) ([]string, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}
	return s.store.GetTopics(ctx, repo.ID)
}

// ReplaceTopics replaces all topics
func (s *Service) ReplaceTopics(ctx context.Context, owner, repoName string, topics []string) ([]string, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}

	if err := s.store.SetTopics(ctx, repo.ID, topics); err != nil {
		return nil, err
	}
	return topics, nil
}

// ListContributors returns repository contributors
func (s *Service) ListContributors(ctx context.Context, owner, repoName string, opts *ListOpts) ([]*Contributor, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}
	// TODO: Implement contributor tracking
	return []*Contributor{}, nil
}

// GetReadme returns the README content
func (s *Service) GetReadme(ctx context.Context, owner, repoName, ref string) (*Content, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}
	// TODO: Implement file content retrieval from git
	return nil, ErrNotFound
}

// GetContents returns file or directory contents
func (s *Service) GetContents(ctx context.Context, owner, repoName, path, ref string) (*Content, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}
	// TODO: Implement file content retrieval from git
	return nil, ErrNotFound
}

// CreateOrUpdateFile creates or updates a file
func (s *Service) CreateOrUpdateFile(ctx context.Context, owner, repoName, path, message, content, sha, branch string, author *CommitAuthor) (*FileCommit, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}
	// TODO: Implement file creation/update via git
	return nil, ErrNotFound
}

// DeleteFile deletes a file
func (s *Service) DeleteFile(ctx context.Context, owner, repoName, path, message, sha, branch string, author *CommitAuthor) (*FileCommit, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}
	// TODO: Implement file deletion via git
	return nil, ErrNotFound
}

// IncrementOpenIssues adjusts the open issues count
func (s *Service) IncrementOpenIssues(ctx context.Context, id int64, delta int) error {
	return s.store.IncrementOpenIssues(ctx, id, delta)
}

// IncrementStargazers adjusts the stargazers count
func (s *Service) IncrementStargazers(ctx context.Context, id int64, delta int) error {
	return s.store.IncrementStargazers(ctx, id, delta)
}

// IncrementWatchers adjusts the watchers count
func (s *Service) IncrementWatchers(ctx context.Context, id int64, delta int) error {
	return s.store.IncrementWatchers(ctx, id, delta)
}

// IncrementForks adjusts the forks count
func (s *Service) IncrementForks(ctx context.Context, id int64, delta int) error {
	return s.store.IncrementForks(ctx, id, delta)
}

// populateURLs fills in the URL fields for a repository
func (s *Service) populateURLs(r *Repository, ownerLogin string) {
	r.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Repository:%d", r.ID)))
	r.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s", s.baseURL, ownerLogin, r.Name)
	r.HTMLURL = fmt.Sprintf("%s/%s/%s", s.baseURL, ownerLogin, r.Name)
	r.ForksURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/forks", s.baseURL, ownerLogin, r.Name)
	r.KeysURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/keys{/key_id}", s.baseURL, ownerLogin, r.Name)
	r.CollaboratorsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/collaborators{/collaborator}", s.baseURL, ownerLogin, r.Name)
	r.TeamsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/teams", s.baseURL, ownerLogin, r.Name)
	r.HooksURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/hooks", s.baseURL, ownerLogin, r.Name)
	r.IssueEventsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/issues/events{/number}", s.baseURL, ownerLogin, r.Name)
	r.EventsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/events", s.baseURL, ownerLogin, r.Name)
	r.AssigneesURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/assignees{/user}", s.baseURL, ownerLogin, r.Name)
	r.BranchesURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/branches{/branch}", s.baseURL, ownerLogin, r.Name)
	r.TagsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/tags", s.baseURL, ownerLogin, r.Name)
	r.BlobsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/blobs{/sha}", s.baseURL, ownerLogin, r.Name)
	r.GitTagsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/tags{/sha}", s.baseURL, ownerLogin, r.Name)
	r.GitRefsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/refs{/sha}", s.baseURL, ownerLogin, r.Name)
	r.TreesURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/trees{/sha}", s.baseURL, ownerLogin, r.Name)
	r.StatusesURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/statuses/{sha}", s.baseURL, ownerLogin, r.Name)
	r.LanguagesURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/languages", s.baseURL, ownerLogin, r.Name)
	r.StargazersURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/stargazers", s.baseURL, ownerLogin, r.Name)
	r.ContributorsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/contributors", s.baseURL, ownerLogin, r.Name)
	r.SubscribersURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/subscribers", s.baseURL, ownerLogin, r.Name)
	r.SubscriptionURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/subscription", s.baseURL, ownerLogin, r.Name)
	r.CommitsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/commits{/sha}", s.baseURL, ownerLogin, r.Name)
	r.GitCommitsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/commits{/sha}", s.baseURL, ownerLogin, r.Name)
	r.CommentsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/comments{/number}", s.baseURL, ownerLogin, r.Name)
	r.IssueCommentURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/issues/comments{/number}", s.baseURL, ownerLogin, r.Name)
	r.ContentsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/contents/{+path}", s.baseURL, ownerLogin, r.Name)
	r.CompareURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/compare/{base}...{head}", s.baseURL, ownerLogin, r.Name)
	r.MergesURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/merges", s.baseURL, ownerLogin, r.Name)
	r.ArchiveURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/{archive_format}{/ref}", s.baseURL, ownerLogin, r.Name)
	r.DownloadsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/downloads", s.baseURL, ownerLogin, r.Name)
	r.IssuesURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/issues{/number}", s.baseURL, ownerLogin, r.Name)
	r.PullsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/pulls{/number}", s.baseURL, ownerLogin, r.Name)
	r.MilestonesURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/milestones{/number}", s.baseURL, ownerLogin, r.Name)
	r.NotificationsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/notifications{?since,all,participating}", s.baseURL, ownerLogin, r.Name)
	r.LabelsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/labels{/name}", s.baseURL, ownerLogin, r.Name)
	r.ReleasesURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/releases{/id}", s.baseURL, ownerLogin, r.Name)
	r.DeploymentsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/deployments", s.baseURL, ownerLogin, r.Name)
	r.GitURL = fmt.Sprintf("git://%s/%s/%s.git", s.baseURL, ownerLogin, r.Name)
	r.SSHURL = fmt.Sprintf("git@%s:%s/%s.git", s.baseURL, ownerLogin, r.Name)
	r.CloneURL = fmt.Sprintf("%s/%s/%s.git", s.baseURL, ownerLogin, r.Name)
	r.SVNURL = fmt.Sprintf("%s/%s/%s", s.baseURL, ownerLogin, r.Name)

	// Copy count fields
	r.Forks = r.ForksCount
	r.Watchers = r.WatchersCount
	r.OpenIssues = r.OpenIssuesCount
}

func defaultBool(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}
