package commits

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	pkggit "github.com/go-mizu/blueprints/githome/pkg/git"
)

// Service implements the commits API
type Service struct {
	store     Store
	repoStore repos.Store
	userStore users.Store
	baseURL   string
	reposDir  string
}

// NewService creates a new commits service
func NewService(store Store, repoStore repos.Store, userStore users.Store, baseURL, reposDir string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		userStore: userStore,
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

// List returns commits for a repository
func (s *Service) List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Commit, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30, Page: 1}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}
	if opts.Page == 0 {
		opts.Page = 1
	}

	// Try to open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return []*Commit{}, nil
		}
		return nil, err
	}

	// Determine starting ref
	ref := opts.SHA
	if ref == "" {
		ref = r.DefaultBranch
	}

	// Calculate offset for pagination
	skip := (opts.Page - 1) * opts.PerPage

	// Get commit log with pagination
	var gitCommits []*pkggit.Commit
	if opts.Path != "" {
		// Use file-specific log for path filtering
		gitCommits, err = gitRepo.FileLogWithSkip(ref, opts.Path, opts.PerPage, skip)
	} else {
		gitCommits, err = gitRepo.LogWithSkip(ref, opts.PerPage, skip)
	}
	if err != nil {
		if err == pkggit.ErrRefNotFound || err == pkggit.ErrEmptyRepository {
			return []*Commit{}, nil
		}
		return nil, err
	}

	// Filter commits if needed
	commits := make([]*Commit, 0, len(gitCommits))
	for _, gc := range gitCommits {
		// Apply author filter
		if opts.Author != "" && !strings.Contains(gc.Author.Email, opts.Author) && !strings.Contains(gc.Author.Name, opts.Author) {
			continue
		}

		// Apply committer filter
		if opts.Committer != "" && !strings.Contains(gc.Committer.Email, opts.Committer) && !strings.Contains(gc.Committer.Name, opts.Committer) {
			continue
		}

		// Apply time filters
		if !opts.Since.IsZero() && gc.Author.When.Before(opts.Since) {
			continue
		}
		if !opts.Until.IsZero() && gc.Author.When.After(opts.Until) {
			continue
		}

		parents := make([]*CommitRef, 0, len(gc.Parents))
		for _, p := range gc.Parents {
			parents = append(parents, &CommitRef{
				SHA:     p,
				URL:     fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repo, p),
				HTMLURL: fmt.Sprintf("%s/%s/%s/commit/%s", s.baseURL, owner, repo, p),
			})
		}

		commit := &Commit{
			SHA:    gc.SHA,
			NodeID: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Commit:%s", gc.SHA))),
			Commit: &CommitData{
				Message:      gc.Message,
				CommentCount: 0,
				Author: &CommitAuthor{
					Name:  gc.Author.Name,
					Email: gc.Author.Email,
					Date:  gc.Author.When,
				},
				Committer: &CommitAuthor{
					Name:  gc.Committer.Name,
					Email: gc.Committer.Email,
					Date:  gc.Committer.When,
				},
				Tree: &TreeRef{
					SHA: gc.TreeSHA,
					URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/git/trees/%s", s.baseURL, owner, repo, gc.TreeSHA),
				},
				Verification: &Verification{
					Verified: false,
					Reason:   "unsigned",
				},
			},
			Parents: parents,
		}

		// Try to find matching user by email
		author, _ := s.userStore.GetByEmail(ctx, gc.Author.Email)
		if author != nil {
			commit.Author = author.ToSimple()
		}
		committer, _ := s.userStore.GetByEmail(ctx, gc.Committer.Email)
		if committer != nil {
			commit.Committer = committer.ToSimple()
		}

		s.populateURLs(commit, owner, repo)
		commits = append(commits, commit)
	}

	return commits, nil
}

// Get retrieves a commit by ref
func (s *Service) Get(ctx context.Context, owner, repo, ref string) (*Commit, error) {
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

	// Resolve ref to SHA if needed (could be branch name, tag, etc.)
	sha := ref
	if len(ref) != 40 {
		resolvedSHA, err := gitRepo.ResolveRef(ref)
		if err != nil {
			if err == pkggit.ErrRefNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		sha = resolvedSHA
	}

	// Get the commit
	gc, err := gitRepo.GetCommit(sha)
	if err != nil {
		if err == pkggit.ErrNotFound || err == pkggit.ErrInvalidSHA {
			return nil, ErrNotFound
		}
		return nil, err
	}

	parents := make([]*CommitRef, 0, len(gc.Parents))
	for _, p := range gc.Parents {
		parents = append(parents, &CommitRef{
			SHA:     p,
			URL:     fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repo, p),
			HTMLURL: fmt.Sprintf("%s/%s/%s/commit/%s", s.baseURL, owner, repo, p),
		})
	}

	commit := &Commit{
		SHA:    gc.SHA,
		NodeID: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Commit:%s", gc.SHA))),
		Commit: &CommitData{
			Message:      gc.Message,
			CommentCount: 0,
			Author: &CommitAuthor{
				Name:  gc.Author.Name,
				Email: gc.Author.Email,
				Date:  gc.Author.When,
			},
			Committer: &CommitAuthor{
				Name:  gc.Committer.Name,
				Email: gc.Committer.Email,
				Date:  gc.Committer.When,
			},
			Tree: &TreeRef{
				SHA: gc.TreeSHA,
				URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/git/trees/%s", s.baseURL, owner, repo, gc.TreeSHA),
			},
			Verification: &Verification{
				Verified: false,
				Reason:   "unsigned",
			},
		},
		Parents: parents,
	}

	// Get stats and files for single commit view
	var parentSHA string
	if len(gc.Parents) > 0 {
		parentSHA = gc.Parents[0]
	}

	// Get diff stats
	stats, err := gitRepo.DiffStats(parentSHA, sha)
	if err == nil {
		commit.Stats = &CommitStats{
			Additions: stats.Additions,
			Deletions: stats.Deletions,
			Total:     stats.Total,
		}
	}

	// Get diff files
	diffFiles, err := gitRepo.DiffFiles(parentSHA, sha)
	if err == nil {
		files := make([]*CommitFile, 0, len(diffFiles))
		for _, df := range diffFiles {
			file := &CommitFile{
				SHA:              df.SHA,
				Filename:         df.Filename,
				Status:           df.Status,
				Additions:        df.Additions,
				Deletions:        df.Deletions,
				Changes:          df.Changes,
				Patch:            df.Patch,
				PreviousFilename: df.PreviousFilename,
			}
			s.populateFileURLs(file, owner, repo, sha)
			files = append(files, file)
		}
		commit.Files = files
	}

	// Try to find matching user by email
	author, _ := s.userStore.GetByEmail(ctx, gc.Author.Email)
	if author != nil {
		commit.Author = author.ToSimple()
	}
	committer, _ := s.userStore.GetByEmail(ctx, gc.Committer.Email)
	if committer != nil {
		commit.Committer = committer.ToSimple()
	}

	s.populateURLs(commit, owner, repo)
	return commit, nil
}

// Compare compares two commits
func (s *Service) Compare(ctx context.Context, owner, repo, base, head string) (*Comparison, error) {
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

	// Resolve base and head refs
	baseSHA := base
	if len(base) != 40 {
		resolvedSHA, err := gitRepo.ResolveRef(base)
		if err != nil {
			if err == pkggit.ErrRefNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		baseSHA = resolvedSHA
	}

	headSHA := head
	if len(head) != 40 {
		resolvedSHA, err := gitRepo.ResolveRef(head)
		if err != nil {
			if err == pkggit.ErrRefNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		headSHA = resolvedSHA
	}

	// Get base commit
	baseCommit, err := s.Get(ctx, owner, repo, baseSHA)
	if err != nil {
		return nil, err
	}

	// Get commits between base and head
	// Get log from head and count until we hit base
	headCommits, err := gitRepo.Log(headSHA, 250) // Limit to 250 commits
	if err != nil {
		return nil, err
	}

	// Find commits ahead of base
	commits := make([]*Commit, 0)
	aheadBy := 0
	for _, gc := range headCommits {
		if gc.SHA == baseSHA {
			break
		}
		aheadBy++

		parents := make([]*CommitRef, 0, len(gc.Parents))
		for _, p := range gc.Parents {
			parents = append(parents, &CommitRef{
				SHA: p,
				URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repo, p),
			})
		}

		commit := &Commit{
			SHA:    gc.SHA,
			NodeID: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Commit:%s", gc.SHA))),
			Commit: &CommitData{
				Message: gc.Message,
				Author: &CommitAuthor{
					Name:  gc.Author.Name,
					Email: gc.Author.Email,
					Date:  gc.Author.When,
				},
				Committer: &CommitAuthor{
					Name:  gc.Committer.Name,
					Email: gc.Committer.Email,
					Date:  gc.Committer.When,
				},
				Tree: &TreeRef{
					SHA: gc.TreeSHA,
					URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/git/trees/%s", s.baseURL, owner, repo, gc.TreeSHA),
				},
			},
			Parents: parents,
		}
		s.populateURLs(commit, owner, repo)
		commits = append(commits, commit)
	}

	// Determine status
	status := "ahead"
	if aheadBy == 0 {
		status = "identical"
	}

	// Get diff to find files
	files := make([]*CommitFile, 0)
	if baseSHA != headSHA {
		diff, err := gitRepo.Diff(baseSHA, headSHA)
		if err == nil && diff != "" {
			// Parse diff for file info (simplified)
			files = s.parseDiffFiles(diff, owner, repo)
		}
	}

	comparison := &Comparison{
		Status:       status,
		AheadBy:      aheadBy,
		BehindBy:     0, // Would need additional logic to compute
		TotalCommits: aheadBy,
		Commits:      commits,
		Files:        files,
		BaseCommit:   baseCommit,
	}

	comparison.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/compare/%s...%s", s.baseURL, owner, repo, base, head)
	comparison.HTMLURL = fmt.Sprintf("%s/%s/%s/compare/%s...%s", s.baseURL, owner, repo, base, head)
	comparison.PermalinkURL = comparison.HTMLURL
	comparison.DiffURL = fmt.Sprintf("%s/%s/%s/compare/%s...%s.diff", s.baseURL, owner, repo, base, head)
	comparison.PatchURL = fmt.Sprintf("%s/%s/%s/compare/%s...%s.patch", s.baseURL, owner, repo, base, head)

	return comparison, nil
}

// parseDiffFiles parses a diff string to extract file information
func (s *Service) parseDiffFiles(diff, owner, repo string) []*CommitFile {
	files := make([]*CommitFile, 0)
	lines := strings.Split(diff, "\n")

	var currentFile *CommitFile
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			if currentFile != nil {
				files = append(files, currentFile)
			}
			// Parse filename from "diff --git a/file b/file"
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				filename := strings.TrimPrefix(parts[3], "b/")
				currentFile = &CommitFile{
					Filename:    filename,
					Status:      "modified",
					ContentsURL: fmt.Sprintf("%s/api/v3/repos/%s/%s/contents/%s", s.baseURL, owner, repo, filename),
				}
			}
		} else if currentFile != nil {
			if strings.HasPrefix(line, "new file") {
				currentFile.Status = "added"
			} else if strings.HasPrefix(line, "deleted file") {
				currentFile.Status = "removed"
			} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				currentFile.Additions++
				currentFile.Changes++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				currentFile.Deletions++
				currentFile.Changes++
			}
		}
	}
	if currentFile != nil {
		files = append(files, currentFile)
	}

	return files
}

// ListBranchesForHead returns branches containing the commit
func (s *Service) ListBranchesForHead(ctx context.Context, owner, repo, sha string) ([]*Branch, error) {
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
			return []*Branch{}, nil
		}
		return nil, err
	}

	// Get all branch refs
	refs, err := gitRepo.ListRefs("heads")
	if err != nil {
		return nil, err
	}

	// Check which branches contain the commit
	branches := make([]*Branch, 0)
	for _, ref := range refs {
		branchName := strings.TrimPrefix(ref.Name, "refs/heads/")

		// Check if this branch contains the commit by walking its history
		commits, err := gitRepo.Log(ref.SHA, 100)
		if err != nil {
			continue
		}

		for _, c := range commits {
			if c.SHA == sha {
				branches = append(branches, &Branch{
					Name: branchName,
					Commit: &CommitRef{
						SHA: ref.SHA,
						URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repo, ref.SHA),
					},
				})
				break
			}
		}
	}

	return branches, nil
}

// ListPullsForCommit returns PRs associated with a commit
// Note: This requires the pulls store to be injected for full functionality.
// Currently returns empty as pulls store is not a dependency.
func (s *Service) ListPullsForCommit(ctx context.Context, owner, repo, sha string, opts *ListOpts) ([]*PullRequest, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// To implement fully, would need pulls.Store as a dependency
	// For now, return empty list as this would require adding pulls.Store
	// which could introduce circular dependencies
	return []*PullRequest{}, nil
}

// GetCombinedStatus returns combined status for a ref
func (s *Service) GetCombinedStatus(ctx context.Context, owner, repo, ref string) (*CombinedStatus, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	combined, err := s.store.GetCombinedStatus(ctx, r.ID, ref)
	if err != nil {
		return nil, err
	}
	if combined == nil {
		combined = &CombinedStatus{
			State:      "pending",
			Statuses:   []*Status{},
			SHA:        ref,
			TotalCount: 0,
			Repository: &Repository{
				ID:       r.ID,
				NodeID:   base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Repository:%d", r.ID))),
				Name:     r.Name,
				FullName: r.FullName,
			},
		}
	}

	combined.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s/status", s.baseURL, owner, repo, ref)
	combined.CommitURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repo, ref)
	if combined.Repository != nil {
		combined.Repository.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s", s.baseURL, owner, repo)
		combined.Repository.HTMLURL = fmt.Sprintf("%s/%s/%s", s.baseURL, owner, repo)
	}

	return combined, nil
}

// ListStatuses returns statuses for a ref
func (s *Service) ListStatuses(ctx context.Context, owner, repo, ref string, opts *ListOpts) ([]*Status, error) {
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

	statuses, err := s.store.ListStatuses(ctx, r.ID, ref, opts)
	if err != nil {
		return nil, err
	}

	for _, status := range statuses {
		s.populateStatusURLs(status, owner, repo)
	}
	return statuses, nil
}

// CreateStatus creates a status for a SHA
func (s *Service) CreateStatus(ctx context.Context, owner, repo, sha string, creatorID int64, in *CreateStatusIn) (*Status, error) {
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

	context := in.Context
	if context == "" {
		context = "default"
	}

	now := time.Now()
	status := &Status{
		State:       in.State,
		TargetURL:   in.TargetURL,
		Description: in.Description,
		Context:     context,
		CreatedAt:   now,
		UpdatedAt:   now,
		Creator:     creator.ToSimple(),
	}

	if err := s.store.CreateStatus(ctx, r.ID, sha, status); err != nil {
		return nil, err
	}

	s.populateStatusURLs(status, owner, repo)
	return status, nil
}

// populateURLs fills in the URL fields for a commit
func (s *Service) populateURLs(c *Commit, owner, repo string) {
	c.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repo, c.SHA)
	c.HTMLURL = fmt.Sprintf("%s/%s/%s/commit/%s", s.baseURL, owner, repo, c.SHA)
	c.CommentsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s/comments", s.baseURL, owner, repo, c.SHA)
	if c.Commit != nil {
		c.Commit.URL = c.URL
	}
}

// populateFileURLs fills in the URL fields for a commit file
func (s *Service) populateFileURLs(f *CommitFile, owner, repo, sha string) {
	// URL-encode the filename for paths with special characters
	encodedFilename := strings.ReplaceAll(f.Filename, "/", "%2F")
	f.BlobURL = fmt.Sprintf("%s/%s/%s/blob/%s/%s", s.baseURL, owner, repo, sha, f.Filename)
	f.RawURL = fmt.Sprintf("%s/%s/%s/raw/%s/%s", s.baseURL, owner, repo, sha, f.Filename)
	f.ContentsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/contents/%s?ref=%s", s.baseURL, owner, repo, encodedFilename, sha)
}

// populateStatusURLs fills in the URL fields for a status
func (s *Service) populateStatusURLs(status *Status, owner, repo string) {
	status.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Status:%d", status.ID)))
	status.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/statuses/%d", s.baseURL, owner, repo, status.ID)
}
