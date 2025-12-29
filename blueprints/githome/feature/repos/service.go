package repos

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/users"
	pkggit "github.com/go-mizu/blueprints/githome/pkg/git"
)

// Service implements the repos API
type Service struct {
	store     Store
	userStore users.Store
	orgStore  orgs.Store
	baseURL   string
	reposDir  string
}

// NewService creates a new repos service
func NewService(store Store, userStore users.Store, orgStore orgs.Store, baseURL, reposDir string) *Service {
	return &Service{
		store:     store,
		userStore: userStore,
		orgStore:  orgStore,
		baseURL:   baseURL,
		reposDir:  reposDir,
	}
}

// getRepoPath returns the filesystem path for a repository.
// It checks for both bare repos ({owner}/{repo}.git) and regular repos ({owner}/{repo}).
func (s *Service) getRepoPath(owner, repo string) string {
	// First check for bare repo with .git suffix
	barePath := filepath.Join(s.reposDir, owner, repo+".git")
	if _, err := os.Stat(barePath); err == nil {
		return barePath
	}
	// Fall back to regular repo (no .git suffix)
	return filepath.Join(s.reposDir, owner, repo)
}

// openRepo opens a git repository
func (s *Service) openRepo(owner, repo string) (*pkggit.Repository, error) {
	path := s.getRepoPath(owner, repo)
	return pkggit.Open(path)
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
	// Populate owner
	if repo.OwnerType == "User" {
		user, _ := s.userStore.GetByID(ctx, repo.OwnerID)
		if user != nil {
			repo.Owner = user.ToSimple()
		}
	} else {
		org, _ := s.orgStore.GetByID(ctx, repo.OwnerID)
		if org != nil {
			repo.Owner = &users.SimpleUser{
				ID:        org.ID,
				NodeID:    org.NodeID,
				Login:     org.Login,
				AvatarURL: org.AvatarURL,
				Type:      "Organization",
			}
		}
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

	// Get owner login and populate owner
	var ownerLogin string
	if repo.OwnerType == "User" {
		user, _ := s.userStore.GetByID(ctx, repo.OwnerID)
		if user != nil {
			ownerLogin = user.Login
			repo.Owner = user.ToSimple()
		}
	} else {
		org, _ := s.orgStore.GetByID(ctx, repo.OwnerID)
		if org != nil {
			ownerLogin = org.Login
			repo.Owner = &users.SimpleUser{
				ID:        org.ID,
				NodeID:    org.NodeID,
				Login:     org.Login,
				AvatarURL: org.AvatarURL,
				Type:      "Organization",
			}
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

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	// Try to open git repository
	gitRepo, err := s.openRepo(owner, repoName)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return []*Contributor{}, nil
		}
		return nil, err
	}

	// Get commit log from default branch
	commits, err := gitRepo.Log(repo.DefaultBranch, 1000)
	if err != nil {
		return []*Contributor{}, nil
	}

	// Aggregate contributions by email
	contributions := make(map[string]*Contributor)
	for _, c := range commits {
		email := c.Author.Email
		if contrib, exists := contributions[email]; exists {
			contrib.Contributions++
		} else {
			// Try to find user by email
			user, _ := s.userStore.GetByEmail(ctx, email)
			contrib := &Contributor{
				Contributions: 1,
			}
			if user != nil {
				contrib.SimpleUser = user.ToSimple()
			} else {
				// Create a placeholder SimpleUser
				contrib.SimpleUser = &users.SimpleUser{
					Login:     c.Author.Name,
					AvatarURL: "",
					Type:      "User",
				}
			}
			contributions[email] = contrib
		}
	}

	// Convert to slice and sort by contributions
	result := make([]*Contributor, 0, len(contributions))
	for _, contrib := range contributions {
		result = append(result, contrib)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Contributions > result[j].Contributions
	})

	// Apply pagination
	start := 0
	if opts.Page > 1 {
		start = (opts.Page - 1) * opts.PerPage
	}
	end := start + opts.PerPage
	if start > len(result) {
		return []*Contributor{}, nil
	}
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
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

	// Try common README filenames
	readmeNames := []string{"README.md", "README", "README.txt", "readme.md", "Readme.md"}
	for _, name := range readmeNames {
		content, err := s.GetContents(ctx, owner, repoName, name, ref)
		if err == nil && content != nil {
			return content, nil
		}
	}

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

	// Try to open git repository
	gitRepo, err := s.openRepo(owner, repoName)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Determine ref to use
	if ref == "" {
		ref = repo.DefaultBranch
	}

	// Resolve ref to commit SHA
	sha, err := gitRepo.ResolveRef(ref)
	if err != nil {
		return nil, ErrNotFound
	}

	// Get commit
	commit, err := gitRepo.GetCommit(sha)
	if err != nil {
		return nil, ErrNotFound
	}

	// Get tree
	tree, err := gitRepo.GetTree(commit.TreeSHA)
	if err != nil {
		return nil, ErrNotFound
	}

	// Clean path
	path = strings.TrimPrefix(path, "/")

	// Navigate to target path
	if path != "" {
		parts := strings.Split(path, "/")
		currentTree := tree

		for i, part := range parts {
			found := false
			for _, entry := range currentTree.Entries {
				if entry.Name == part {
					found = true
					if i == len(parts)-1 {
						// This is the final target
						if entry.Type == pkggit.ObjectBlob {
							// It's a file
							blob, err := gitRepo.GetBlob(entry.SHA)
							if err != nil {
								return nil, ErrNotFound
							}

							content := &Content{
								Name:        entry.Name,
								Path:        path,
								SHA:         entry.SHA,
								Size:        int(blob.Size),
								Type:        "file",
								Content:     base64.StdEncoding.EncodeToString(blob.Content),
								Encoding:    "base64",
								URL:         fmt.Sprintf("%s/api/v3/repos/%s/%s/contents/%s", s.baseURL, owner, repoName, path),
								HTMLURL:     fmt.Sprintf("%s/%s/%s/blob/%s/%s", s.baseURL, owner, repoName, ref, path),
								GitURL:      fmt.Sprintf("%s/api/v3/repos/%s/%s/git/blobs/%s", s.baseURL, owner, repoName, entry.SHA),
								DownloadURL: fmt.Sprintf("%s/%s/%s/raw/%s/%s", s.baseURL, owner, repoName, ref, path),
							}
							return content, nil
						} else if entry.Type == pkggit.ObjectTree {
							// It's a directory - would need to return array
							subTree, err := gitRepo.GetTree(entry.SHA)
							if err != nil {
								return nil, ErrNotFound
							}
							currentTree = subTree
						}
					} else {
						// Navigate deeper
						if entry.Type == pkggit.ObjectTree {
							subTree, err := gitRepo.GetTree(entry.SHA)
							if err != nil {
								return nil, ErrNotFound
							}
							currentTree = subTree
						} else {
							return nil, ErrNotFound
						}
					}
					break
				}
			}
			if !found {
				return nil, ErrNotFound
			}
		}

		// If we get here for a directory, return first entry info
		if len(currentTree.Entries) > 0 {
			return &Content{
				Name:    filepath.Base(path),
				Path:    path,
				Type:    "dir",
				URL:     fmt.Sprintf("%s/api/v3/repos/%s/%s/contents/%s", s.baseURL, owner, repoName, path),
				HTMLURL: fmt.Sprintf("%s/%s/%s/tree/%s/%s", s.baseURL, owner, repoName, ref, path),
			}, nil
		}
	}

	// Root directory
	return &Content{
		Name:    "",
		Path:    "",
		Type:    "dir",
		URL:     fmt.Sprintf("%s/api/v3/repos/%s/%s/contents", s.baseURL, owner, repoName),
		HTMLURL: fmt.Sprintf("%s/%s/%s/tree/%s", s.baseURL, owner, repoName, ref),
	}, nil
}

// ListTreeEntries returns all entries in a directory
func (s *Service) ListTreeEntries(ctx context.Context, owner, repoName, path, ref string) ([]*TreeEntry, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}

	// Try to open git repository
	gitRepo, err := s.openRepo(owner, repoName)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Determine ref to use
	if ref == "" {
		ref = repo.DefaultBranch
	}

	// Resolve ref to commit SHA
	sha, err := gitRepo.ResolveRef(ref)
	if err != nil {
		return nil, ErrNotFound
	}

	// Get commit
	commit, err := gitRepo.GetCommit(sha)
	if err != nil {
		return nil, ErrNotFound
	}

	// Get root tree
	tree, err := gitRepo.GetTree(commit.TreeSHA)
	if err != nil {
		return nil, ErrNotFound
	}

	// Navigate to target path if not root
	path = strings.TrimPrefix(path, "/")
	if path != "" {
		parts := strings.Split(path, "/")
		for _, part := range parts {
			found := false
			for _, entry := range tree.Entries {
				if entry.Name == part {
					if entry.Type == pkggit.ObjectTree {
						subTree, err := gitRepo.GetTree(entry.SHA)
						if err != nil {
							return nil, ErrNotFound
						}
						tree = subTree
						found = true
						break
					} else {
						// Not a directory
						return nil, ErrNotFound
					}
				}
			}
			if !found {
				return nil, ErrNotFound
			}
		}
	}

	// Convert git entries to TreeEntry structs
	entries := make([]*TreeEntry, 0, len(tree.Entries))
	for _, e := range tree.Entries {
		entryPath := e.Name
		if path != "" {
			entryPath = path + "/" + e.Name
		}

		entryType := "file"
		mode := "100644"
		if e.Type == pkggit.ObjectTree {
			entryType = "dir"
			mode = "040000"
		} else if e.Mode == pkggit.ModeExecutable {
			mode = "100755"
		} else if e.Mode == pkggit.ModeSymlink {
			entryType = "symlink"
			mode = "120000"
		} else if e.Mode == pkggit.ModeSubmodule {
			entryType = "submodule"
			mode = "160000"
		}

		entry := &TreeEntry{
			Name:    e.Name,
			Path:    entryPath,
			SHA:     e.SHA,
			Size:    int64(e.Size),
			Type:    entryType,
			Mode:    mode,
			URL:     fmt.Sprintf("%s/api/v3/repos/%s/%s/contents/%s?ref=%s", s.baseURL, owner, repoName, entryPath, ref),
			HTMLURL: fmt.Sprintf("%s/%s/%s/%s/%s/%s", s.baseURL, owner, repoName, entryType, ref, entryPath),
		}

		if entryType == "file" {
			entry.DownloadURL = fmt.Sprintf("%s/%s/%s/raw/%s/%s", s.baseURL, owner, repoName, ref, entryPath)
			// Fix HTMLURL for files (use "blob" not "file")
			entry.HTMLURL = fmt.Sprintf("%s/%s/%s/blob/%s/%s", s.baseURL, owner, repoName, ref, entryPath)
		} else if entryType == "dir" {
			entry.HTMLURL = fmt.Sprintf("%s/%s/%s/tree/%s/%s", s.baseURL, owner, repoName, ref, entryPath)
		}

		entries = append(entries, entry)
	}

	// Sort: directories first, then alphabetically
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type == "dir" && entries[j].Type != "dir" {
			return true
		}
		if entries[i].Type != "dir" && entries[j].Type == "dir" {
			return false
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return entries, nil
}

// ListTreeEntriesWithCommits returns directory entries with last commit info
func (s *Service) ListTreeEntriesWithCommits(ctx context.Context, owner, repoName, path, ref string) ([]*TreeEntry, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}

	// Try to open git repository
	gitRepo, err := s.openRepo(owner, repoName)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Determine ref to use
	if ref == "" {
		ref = repo.DefaultBranch
	}

	// Resolve ref to commit SHA
	sha, err := gitRepo.ResolveRef(ref)
	if err != nil {
		return nil, ErrNotFound
	}

	// Get commit
	commit, err := gitRepo.GetCommit(sha)
	if err != nil {
		return nil, ErrNotFound
	}

	// Get root tree
	treeSHA := commit.TreeSHA

	// Navigate to target path if not root
	path = strings.TrimPrefix(path, "/")
	if path != "" {
		tree, err := gitRepo.GetTree(treeSHA)
		if err != nil {
			return nil, ErrNotFound
		}
		parts := strings.Split(path, "/")
		for _, part := range parts {
			found := false
			for _, entry := range tree.Entries {
				if entry.Name == part {
					if entry.Type == pkggit.ObjectTree {
						treeSHA = entry.SHA
						subTree, err := gitRepo.GetTree(entry.SHA)
						if err != nil {
							return nil, ErrNotFound
						}
						tree = subTree
						found = true
						break
					} else {
						return nil, ErrNotFound
					}
				}
			}
			if !found {
				return nil, ErrNotFound
			}
		}
	}

	// Get tree entries with last commit info
	// Pass the commit SHA (not tree SHA) so it can walk the history
	gitEntries, err := gitRepo.GetTreeWithLastCommits(sha, 500)
	if err != nil {
		// Fall back to simple tree listing if last commit info fails
		return s.ListTreeEntries(ctx, owner, repoName, path, ref)
	}

	// If we navigated to a subdirectory, we need to filter entries
	// The GetTreeWithLastCommits returns root tree entries, so for subdirs we need different logic
	if path != "" {
		// For subdirectories, get tree from the resolved tree SHA and then get commits per entry
		tree, err := gitRepo.GetTree(treeSHA)
		if err != nil {
			return nil, ErrNotFound
		}
		gitEntries = make([]*pkggit.TreeEntryWithCommit, len(tree.Entries))
		for i, e := range tree.Entries {
			gitEntries[i] = &pkggit.TreeEntryWithCommit{TreeEntry: e}
		}
		// Try to get last commits for these entries
		entriesWithCommits, err := gitRepo.GetTreeWithLastCommits(sha, 500)
		if err == nil {
			// Build map of entry names to commits from the walk result
			commitMap := make(map[string]*pkggit.Commit)
			for _, e := range entriesWithCommits {
				// For subdirectory, look for entries with matching path prefix
				if strings.HasPrefix(e.Name, path+"/") {
					subName := strings.TrimPrefix(e.Name, path+"/")
					if !strings.Contains(subName, "/") {
						commitMap[subName] = e.LastCommit
					}
				} else if e.Name == path {
					// Directory itself
					commitMap[e.Name] = e.LastCommit
				}
			}
			// Apply commits to our entries
			for i, e := range gitEntries {
				if c, ok := commitMap[e.Name]; ok {
					gitEntries[i].LastCommit = c
				}
			}
		}
	}

	// Convert git entries to TreeEntry structs
	entries := make([]*TreeEntry, 0, len(gitEntries))
	for _, e := range gitEntries {
		entryPath := e.Name
		if path != "" {
			entryPath = path + "/" + e.Name
		}

		entryType := "file"
		mode := "100644"
		if e.Type == pkggit.ObjectTree {
			entryType = "dir"
			mode = "040000"
		} else if e.Mode == pkggit.ModeExecutable {
			mode = "100755"
		} else if e.Mode == pkggit.ModeSymlink {
			entryType = "symlink"
			mode = "120000"
		} else if e.Mode == pkggit.ModeSubmodule {
			entryType = "submodule"
			mode = "160000"
		}

		entry := &TreeEntry{
			Name:    e.Name,
			Path:    entryPath,
			SHA:     e.SHA,
			Size:    int64(e.Size),
			Type:    entryType,
			Mode:    mode,
			URL:     fmt.Sprintf("%s/api/v3/repos/%s/%s/contents/%s?ref=%s", s.baseURL, owner, repoName, entryPath, ref),
			HTMLURL: fmt.Sprintf("%s/%s/%s/%s/%s/%s", s.baseURL, owner, repoName, entryType, ref, entryPath),
		}

		if entryType == "file" {
			entry.DownloadURL = fmt.Sprintf("%s/%s/%s/raw/%s/%s", s.baseURL, owner, repoName, ref, entryPath)
			entry.HTMLURL = fmt.Sprintf("%s/%s/%s/blob/%s/%s", s.baseURL, owner, repoName, ref, entryPath)
		} else if entryType == "dir" {
			entry.HTMLURL = fmt.Sprintf("%s/%s/%s/tree/%s/%s", s.baseURL, owner, repoName, ref, entryPath)
		}

		// Add last commit info if available
		if e.LastCommit != nil {
			entry.LastCommitSHA = e.LastCommit.SHA
			entry.LastCommitMessage = strings.Split(e.LastCommit.Message, "\n")[0] // First line only
			entry.LastCommitAuthor = e.LastCommit.Author.Name
			entry.LastCommitDate = e.LastCommit.Author.When
		}

		entries = append(entries, entry)
	}

	// Sort: directories first, then alphabetically
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type == "dir" && entries[j].Type != "dir" {
			return true
		}
		if entries[i].Type != "dir" && entries[j].Type == "dir" {
			return false
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return entries, nil
}

// GetBlame returns blame information for a file
func (s *Service) GetBlame(ctx context.Context, owner, repoName, ref, path string) (*BlameResult, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}

	gitRepo, err := s.openRepo(owner, repoName)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if ref == "" {
		ref = repo.DefaultBranch
	}

	blameResult, err := gitRepo.Blame(ref, path)
	if err != nil {
		return nil, ErrNotFound
	}

	lines := make([]*BlameLine, len(blameResult.Lines))
	for i, line := range blameResult.Lines {
		lines[i] = &BlameLine{
			LineNumber: line.LineNumber,
			Content:    line.Content,
			CommitSHA:  line.CommitSHA,
			Author:     line.Author.Name,
			AuthorMail: line.Author.Email,
			Date:       line.Author.When,
		}
	}

	return &BlameResult{
		Path:  path,
		Lines: lines,
	}, nil
}

// GetCommitCount returns the total number of commits from a ref
func (s *Service) GetCommitCount(ctx context.Context, owner, repoName, ref string) (int, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return 0, err
	}
	if repo == nil {
		return 0, ErrNotFound
	}

	gitRepo, err := s.openRepo(owner, repoName)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return 0, ErrNotFound
		}
		return 0, err
	}

	if ref == "" {
		ref = repo.DefaultBranch
	}

	return gitRepo.CommitCount(ref)
}

// GetLatestCommit returns the latest commit for a ref
func (s *Service) GetLatestCommit(ctx context.Context, owner, repoName, ref string) (*Commit, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}

	gitRepo, err := s.openRepo(owner, repoName)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if ref == "" {
		ref = repo.DefaultBranch
	}

	commits, err := gitRepo.Log(ref, 1)
	if err != nil || len(commits) == 0 {
		return nil, ErrNotFound
	}

	c := commits[0]
	parents := make([]*TreeRef, 0, len(c.Parents))
	for _, p := range c.Parents {
		parents = append(parents, &TreeRef{
			SHA: p,
			URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/git/commits/%s", s.baseURL, owner, repoName, p),
		})
	}

	return &Commit{
		SHA:    c.SHA,
		NodeID: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Commit:%s", c.SHA))),
		URL:    fmt.Sprintf("%s/api/v3/repos/%s/%s/git/commits/%s", s.baseURL, owner, repoName, c.SHA),
		HTMLURL: fmt.Sprintf("%s/%s/%s/commit/%s", s.baseURL, owner, repoName, c.SHA),
		Author: &CommitAuthor{
			Name:  c.Author.Name,
			Email: c.Author.Email,
			Date:  c.Author.When,
		},
		Committer: &CommitAuthor{
			Name:  c.Committer.Name,
			Email: c.Committer.Email,
			Date:  c.Committer.When,
		},
		Message: c.Message,
		Tree: &TreeRef{
			SHA: c.TreeSHA,
			URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/git/trees/%s", s.baseURL, owner, repoName, c.TreeSHA),
		},
		Parents: parents,
	}, nil
}

// CreateOrUpdateFile creates or updates a file
func (s *Service) CreateOrUpdateFile(ctx context.Context, owner, repoName, path, message, contentBase64, sha, branch string, author *CommitAuthor) (*FileCommit, error) {
	repo, err := s.store.GetByFullName(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrNotFound
	}

	// Try to open git repository
	gitRepo, err := s.openRepo(owner, repoName)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Determine branch
	if branch == "" {
		branch = repo.DefaultBranch
	}

	// Get current branch ref
	branchRef, err := gitRepo.GetRef("refs/heads/" + branch)
	if err != nil {
		return nil, ErrNotFound
	}

	// Get current commit
	currentCommit, err := gitRepo.GetCommit(branchRef.SHA)
	if err != nil {
		return nil, err
	}

	// Decode content
	contentBytes, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 content: %w", err)
	}

	// Create new blob
	blobSHA, err := gitRepo.CreateBlob(contentBytes)
	if err != nil {
		return nil, err
	}

	// Create new tree with updated file
	treeOpts := &pkggit.CreateTreeOpts{
		BaseSHA: currentCommit.TreeSHA,
		Entries: []pkggit.TreeEntryInput{
			{
				Path:    path,
				Mode:    pkggit.ModeFile,
				Type:    pkggit.ObjectBlob,
				SHA:     blobSHA,
			},
		},
	}

	newTreeSHA, err := gitRepo.CreateTree(treeOpts)
	if err != nil {
		return nil, err
	}

	// Determine author info
	now := time.Now()
	authorSig := pkggit.Signature{
		Name:  "System",
		Email: "system@githome.local",
		When:  now,
	}
	if author != nil {
		authorSig.Name = author.Name
		authorSig.Email = author.Email
		if !author.Date.IsZero() {
			authorSig.When = author.Date
		}
	}

	// Create commit
	newCommitSHA, err := gitRepo.CreateCommit(&pkggit.CreateCommitOpts{
		Message:   message,
		TreeSHA:   newTreeSHA,
		Parents:   []string{branchRef.SHA},
		Author:    authorSig,
		Committer: authorSig,
	})
	if err != nil {
		return nil, err
	}

	// Update branch ref
	if err := gitRepo.UpdateRef("refs/heads/"+branch, newCommitSHA, true); err != nil {
		return nil, err
	}

	// Build response
	fileContent := &Content{
		Name:     filepath.Base(path),
		Path:     path,
		SHA:      blobSHA,
		Size:     len(contentBytes),
		Type:     "file",
		URL:      fmt.Sprintf("%s/api/v3/repos/%s/%s/contents/%s", s.baseURL, owner, repoName, path),
		HTMLURL:  fmt.Sprintf("%s/%s/%s/blob/%s/%s", s.baseURL, owner, repoName, branch, path),
		GitURL:   fmt.Sprintf("%s/api/v3/repos/%s/%s/git/blobs/%s", s.baseURL, owner, repoName, blobSHA),
	}

	commitInfo := &Commit{
		SHA:     newCommitSHA,
		Message: message,
		Author:  author,
		URL:     fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repoName, newCommitSHA),
		HTMLURL: fmt.Sprintf("%s/%s/%s/commit/%s", s.baseURL, owner, repoName, newCommitSHA),
	}

	return &FileCommit{
		Content: fileContent,
		Commit:  commitInfo,
	}, nil
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

	// Try to open git repository
	gitRepo, err := s.openRepo(owner, repoName)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Determine branch
	if branch == "" {
		branch = repo.DefaultBranch
	}

	// Get current branch ref
	branchRef, err := gitRepo.GetRef("refs/heads/" + branch)
	if err != nil {
		return nil, ErrNotFound
	}

	// Get current commit
	currentCommit, err := gitRepo.GetCommit(branchRef.SHA)
	if err != nil {
		return nil, err
	}

	// Create new tree without the file (empty content means delete)
	treeOpts := &pkggit.CreateTreeOpts{
		BaseSHA: currentCommit.TreeSHA,
		Entries: []pkggit.TreeEntryInput{
			{
				Path: path,
				Mode: pkggit.ModeFile,
				Type: pkggit.ObjectBlob,
				// Empty SHA and Content means delete
			},
		},
	}

	newTreeSHA, err := gitRepo.CreateTree(treeOpts)
	if err != nil {
		return nil, err
	}

	// Determine author info
	now := time.Now()
	authorSig := pkggit.Signature{
		Name:  "System",
		Email: "system@githome.local",
		When:  now,
	}
	if author != nil {
		authorSig.Name = author.Name
		authorSig.Email = author.Email
		if !author.Date.IsZero() {
			authorSig.When = author.Date
		}
	}

	// Create commit
	newCommitSHA, err := gitRepo.CreateCommit(&pkggit.CreateCommitOpts{
		Message:   message,
		TreeSHA:   newTreeSHA,
		Parents:   []string{branchRef.SHA},
		Author:    authorSig,
		Committer: authorSig,
	})
	if err != nil {
		return nil, err
	}

	// Update branch ref
	if err := gitRepo.UpdateRef("refs/heads/"+branch, newCommitSHA, true); err != nil {
		return nil, err
	}

	commitInfo := &Commit{
		SHA:     newCommitSHA,
		Message: message,
		Author:  author,
		URL:     fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repoName, newCommitSHA),
		HTMLURL: fmt.Sprintf("%s/%s/%s/commit/%s", s.baseURL, owner, repoName, newCommitSHA),
	}

	return &FileCommit{
		Content: nil, // File was deleted
		Commit:  commitInfo,
	}, nil
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
