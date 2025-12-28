package git

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	pkggit "github.com/go-mizu/blueprints/githome/pkg/git"
)

// Service implements the git API
type Service struct {
	store     Store
	repoStore repos.Store
	baseURL   string
	reposDir  string // Base directory for git repositories
}

// NewService creates a new git service
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

// GetBlob retrieves a blob by SHA
func (s *Service) GetBlob(ctx context.Context, owner, repo, sha string) (*Blob, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Check cache first
	blob, err := s.store.GetCachedBlob(ctx, r.ID, sha)
	if err != nil {
		return nil, err
	}
	if blob != nil {
		s.populateBlobURLs(blob, owner, repo)
		return blob, nil
	}

	// Get from git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, ErrNotFound
		}
		return nil, err
	}

	gitBlob, err := gitRepo.GetBlob(sha)
	if err != nil {
		if err == pkggit.ErrNotFound {
			return nil, ErrNotFound
		}
		if err == pkggit.ErrInvalidSHA {
			return nil, ErrNotFound
		}
		return nil, err
	}

	blob = &Blob{
		SHA:      gitBlob.SHA,
		NodeID:   base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Blob:%s", sha))),
		Size:     int(gitBlob.Size),
		Content:  string(gitBlob.Content),
		Encoding: "utf-8",
	}

	// Cache the blob
	_ = s.store.CacheBlob(ctx, r.ID, blob)

	s.populateBlobURLs(blob, owner, repo)
	return blob, nil
}

// CreateBlob creates a new blob
func (s *Service) CreateBlob(ctx context.Context, owner, repo string, in *CreateBlobIn) (*Blob, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	encoding := in.Encoding
	if encoding == "" {
		encoding = "utf-8"
	}

	// Decode content if base64
	content := []byte(in.Content)
	if encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(in.Content)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 content: %w", err)
		}
		content = decoded
	}

	// Open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, repos.ErrNotFound
		}
		return nil, err
	}

	// Create blob in git
	sha, err := gitRepo.CreateBlob(content)
	if err != nil {
		return nil, fmt.Errorf("create blob: %w", err)
	}

	blob := &Blob{
		SHA:      sha,
		NodeID:   base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Blob:%s", sha))),
		Size:     len(content),
		Content:  in.Content,
		Encoding: encoding,
	}

	// Cache the blob
	if err := s.store.CacheBlob(ctx, r.ID, blob); err != nil {
		return nil, err
	}

	s.populateBlobURLs(blob, owner, repo)
	return blob, nil
}

// GetGitCommit retrieves a git commit by SHA
func (s *Service) GetGitCommit(ctx context.Context, owner, repo, sha string) (*GitCommit, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, ErrNotFound
		}
		return nil, err
	}

	gitCommit, err := gitRepo.GetCommit(sha)
	if err != nil {
		if err == pkggit.ErrNotFound {
			return nil, ErrNotFound
		}
		if err == pkggit.ErrInvalidSHA {
			return nil, ErrNotFound
		}
		return nil, err
	}

	parents := make([]*TreeRef, 0, len(gitCommit.Parents))
	for _, p := range gitCommit.Parents {
		parents = append(parents, &TreeRef{SHA: p})
	}

	commit := &GitCommit{
		SHA:     sha,
		NodeID:  base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("GitCommit:%s", sha))),
		Message: gitCommit.Message,
		Author: &CommitAuthor{
			Name:  gitCommit.Author.Name,
			Email: gitCommit.Author.Email,
			Date:  gitCommit.Author.When,
		},
		Committer: &CommitAuthor{
			Name:  gitCommit.Committer.Name,
			Email: gitCommit.Committer.Email,
			Date:  gitCommit.Committer.When,
		},
		Tree:    &TreeRef{SHA: gitCommit.TreeSHA},
		Parents: parents,
	}

	s.populateCommitURLs(commit, owner, repo)
	return commit, nil
}

// CreateGitCommit creates a new git commit
func (s *Service) CreateGitCommit(ctx context.Context, owner, repo string, in *CreateGitCommitIn) (*GitCommit, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, repos.ErrNotFound
		}
		return nil, err
	}

	now := time.Now()

	author := in.Author
	if author == nil {
		author = &CommitAuthor{
			Name:  "GitHome",
			Email: "noreply@githome.local",
			Date:  now,
		}
	}
	if author.Date.IsZero() {
		author.Date = now
	}

	committer := in.Committer
	if committer == nil {
		committer = author
	}
	if committer.Date.IsZero() {
		committer.Date = now
	}

	// Create commit in git
	sha, err := gitRepo.CreateCommit(&pkggit.CreateCommitOpts{
		Message: in.Message,
		TreeSHA: in.Tree,
		Parents: in.Parents,
		Author: pkggit.Signature{
			Name:  author.Name,
			Email: author.Email,
			When:  author.Date,
		},
		Committer: pkggit.Signature{
			Name:  committer.Name,
			Email: committer.Email,
			When:  committer.Date,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create commit: %w", err)
	}

	parents := make([]*TreeRef, 0, len(in.Parents))
	for _, p := range in.Parents {
		parents = append(parents, &TreeRef{SHA: p})
	}

	commit := &GitCommit{
		SHA:       sha,
		NodeID:    base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("GitCommit:%s", sha))),
		Message:   in.Message,
		Author:    author,
		Committer: committer,
		Tree:      &TreeRef{SHA: in.Tree},
		Parents:   parents,
	}

	s.populateCommitURLs(commit, owner, repo)
	return commit, nil
}

// GetRef retrieves a reference
func (s *Service) GetRef(ctx context.Context, owner, repo, ref string) (*Reference, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, ErrNotFound
		}
		return nil, err
	}

	gitRef, err := gitRepo.GetRef(ref)
	if err != nil {
		if err == pkggit.ErrRefNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	reference := &Reference{
		Ref:    gitRef.Name,
		NodeID: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Ref:%s", ref))),
		Object: &GitObject{
			Type: string(gitRef.ObjectType),
			SHA:  gitRef.SHA,
		},
	}

	s.populateRefURLs(reference, owner, repo)
	return reference, nil
}

// ListMatchingRefs returns refs matching a pattern
func (s *Service) ListMatchingRefs(ctx context.Context, owner, repo, ref string) ([]*Reference, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, repos.ErrNotFound
		}
		return nil, err
	}

	gitRefs, err := gitRepo.ListRefs(ref)
	if err != nil {
		return nil, err
	}

	refs := make([]*Reference, 0, len(gitRefs))
	for _, gitRef := range gitRefs {
		reference := &Reference{
			Ref:    gitRef.Name,
			NodeID: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Ref:%s", gitRef.Name))),
			Object: &GitObject{
				Type: string(gitRef.ObjectType),
				SHA:  gitRef.SHA,
			},
		}
		s.populateRefURLs(reference, owner, repo)
		refs = append(refs, reference)
	}

	return refs, nil
}

// CreateRef creates a new reference
func (s *Service) CreateRef(ctx context.Context, owner, repo string, in *CreateRefIn) (*Reference, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, repos.ErrNotFound
		}
		return nil, err
	}

	if err := gitRepo.CreateRef(in.Ref, in.SHA); err != nil {
		if err == pkggit.ErrRefExists {
			return nil, ErrRefExists
		}
		return nil, err
	}

	// Get the created ref to get full info
	gitRef, err := gitRepo.GetRef(in.Ref)
	if err != nil {
		return nil, err
	}

	reference := &Reference{
		Ref:    gitRef.Name,
		NodeID: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Ref:%s", in.Ref))),
		Object: &GitObject{
			Type: string(gitRef.ObjectType),
			SHA:  gitRef.SHA,
		},
	}

	s.populateRefURLs(reference, owner, repo)
	return reference, nil
}

// UpdateRef updates a reference
func (s *Service) UpdateRef(ctx context.Context, owner, repo, ref, sha string, force bool) (*Reference, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, repos.ErrNotFound
		}
		return nil, err
	}

	if err := gitRepo.UpdateRef(ref, sha, force); err != nil {
		if err == pkggit.ErrRefNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Get the updated ref
	gitRef, err := gitRepo.GetRef(ref)
	if err != nil {
		return nil, err
	}

	reference := &Reference{
		Ref:    gitRef.Name,
		NodeID: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Ref:%s", ref))),
		Object: &GitObject{
			Type: string(gitRef.ObjectType),
			SHA:  gitRef.SHA,
		},
	}

	s.populateRefURLs(reference, owner, repo)
	return reference, nil
}

// DeleteRef deletes a reference
func (s *Service) DeleteRef(ctx context.Context, owner, repo, ref string) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	// Open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return repos.ErrNotFound
		}
		return err
	}

	if err := gitRepo.DeleteRef(ref); err != nil {
		if err == pkggit.ErrRefNotFound {
			return ErrNotFound
		}
		return err
	}

	return nil
}

// GetTree retrieves a tree
func (s *Service) GetTree(ctx context.Context, owner, repo, sha string, recursive bool) (*Tree, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var gitTree *pkggit.Tree
	if recursive {
		gitTree, err = gitRepo.GetTreeRecursive(sha)
	} else {
		gitTree, err = gitRepo.GetTree(sha)
	}
	if err != nil {
		if err == pkggit.ErrNotFound {
			return nil, ErrNotFound
		}
		if err == pkggit.ErrInvalidSHA {
			return nil, ErrNotFound
		}
		return nil, err
	}

	entries := make([]*TreeEntry, 0, len(gitTree.Entries))
	for _, e := range gitTree.Entries {
		entry := &TreeEntry{
			Path: e.Name,
			Mode: e.Mode.String(),
			Type: string(e.Type),
			SHA:  e.SHA,
		}
		if e.Type == pkggit.ObjectBlob {
			entry.Size = int(e.Size)
		}
		entry.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/%ss/%s", s.baseURL, owner, repo, e.Type, e.SHA)
		entries = append(entries, entry)
	}

	tree := &Tree{
		SHA:       sha,
		Tree:      entries,
		Truncated: gitTree.Truncated,
	}
	tree.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/trees/%s", s.baseURL, owner, repo, sha)
	return tree, nil
}

// CreateTree creates a new tree
func (s *Service) CreateTree(ctx context.Context, owner, repo string, in *CreateTreeIn) (*Tree, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, repos.ErrNotFound
		}
		return nil, err
	}

	// Convert entries
	entries := make([]pkggit.TreeEntryInput, 0, len(in.Tree))
	for _, e := range in.Tree {
		entry := pkggit.TreeEntryInput{
			Path: e.Path,
			Mode: pkggit.ParseFileMode(e.Mode),
			SHA:  e.SHA,
		}
		if e.Content != "" {
			entry.Content = []byte(e.Content)
		}
		switch e.Type {
		case "blob":
			entry.Type = pkggit.ObjectBlob
		case "tree":
			entry.Type = pkggit.ObjectTree
		case "commit":
			entry.Type = pkggit.ObjectCommit
		default:
			entry.Type = pkggit.ObjectBlob
		}
		entries = append(entries, entry)
	}

	sha, err := gitRepo.CreateTree(&pkggit.CreateTreeOpts{
		BaseSHA: in.BaseTree,
		Entries: entries,
	})
	if err != nil {
		return nil, fmt.Errorf("create tree: %w", err)
	}

	// Get the created tree to return full info
	return s.GetTree(ctx, owner, repo, sha, false)
}

// GetTag retrieves an annotated tag
func (s *Service) GetTag(ctx context.Context, owner, repo, sha string) (*Tag, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, ErrNotFound
		}
		return nil, err
	}

	gitTag, err := gitRepo.GetTag(sha)
	if err != nil {
		if err == pkggit.ErrNotFound {
			return nil, ErrNotFound
		}
		if err == pkggit.ErrInvalidSHA {
			return nil, ErrNotFound
		}
		return nil, err
	}

	tag := &Tag{
		NodeID:  base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Tag:%s", sha))),
		SHA:     sha,
		Tag:     gitTag.Name,
		Message: gitTag.Message,
		Tagger: &CommitAuthor{
			Name:  gitTag.Tagger.Name,
			Email: gitTag.Tagger.Email,
			Date:  gitTag.Tagger.When,
		},
		Object: &GitObject{
			Type: string(gitTag.TargetType),
			SHA:  gitTag.TargetSHA,
		},
	}

	tag.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/tags/%s", s.baseURL, owner, repo, sha)
	if tag.Object != nil {
		tag.Object.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/%ss/%s", s.baseURL, owner, repo, tag.Object.Type, tag.Object.SHA)
	}
	return tag, nil
}

// CreateTag creates an annotated tag
func (s *Service) CreateTag(ctx context.Context, owner, repo string, in *CreateTagIn) (*Tag, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return nil, repos.ErrNotFound
		}
		return nil, err
	}

	tagger := in.Tagger
	if tagger == nil {
		tagger = &CommitAuthor{
			Name:  "GitHome",
			Email: "noreply@githome.local",
			Date:  time.Now(),
		}
	}
	if tagger.Date.IsZero() {
		tagger.Date = time.Now()
	}

	var targetType pkggit.ObjectType
	switch in.Type {
	case "commit":
		targetType = pkggit.ObjectCommit
	case "tree":
		targetType = pkggit.ObjectTree
	case "blob":
		targetType = pkggit.ObjectBlob
	default:
		targetType = pkggit.ObjectCommit
	}

	sha, err := gitRepo.CreateTag(&pkggit.CreateTagOpts{
		Name:       in.Tag,
		TargetSHA:  in.Object,
		TargetType: targetType,
		Message:    in.Message,
		Tagger: pkggit.Signature{
			Name:  tagger.Name,
			Email: tagger.Email,
			When:  tagger.Date,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create tag: %w", err)
	}

	tag := &Tag{
		NodeID:  base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Tag:%s", sha))),
		SHA:     sha,
		Tag:     in.Tag,
		Message: in.Message,
		Tagger:  tagger,
		Object: &GitObject{
			Type: in.Type,
			SHA:  in.Object,
		},
	}

	tag.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/tags/%s", s.baseURL, owner, repo, sha)
	if tag.Object != nil {
		tag.Object.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/%ss/%s", s.baseURL, owner, repo, tag.Object.Type, tag.Object.SHA)
	}
	return tag, nil
}

// ListTags returns lightweight tags
func (s *Service) ListTags(ctx context.Context, owner, repo string, opts *ListOpts) ([]*LightweightTag, error) {
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

	// Open git repository
	gitRepo, err := s.openRepo(owner, repo)
	if err != nil {
		if err == pkggit.ErrNotARepository {
			return []*LightweightTag{}, nil
		}
		return nil, err
	}

	gitTags, err := gitRepo.ListTags()
	if err != nil {
		return nil, err
	}

	// Apply pagination
	start := (opts.Page - 1) * opts.PerPage
	end := start + opts.PerPage
	if start > len(gitTags) {
		return []*LightweightTag{}, nil
	}
	if end > len(gitTags) {
		end = len(gitTags)
	}

	tags := make([]*LightweightTag, 0, end-start)
	for _, t := range gitTags[start:end] {
		tag := &LightweightTag{
			Name:       t.Name,
			NodeID:     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Tag:%s", t.Name))),
			ZipballURL: fmt.Sprintf("%s/%s/%s/archive/refs/tags/%s.zip", s.baseURL, owner, repo, t.Name),
			TarballURL: fmt.Sprintf("%s/%s/%s/archive/refs/tags/%s.tar.gz", s.baseURL, owner, repo, t.Name),
			Commit: &CommitRef{
				SHA: t.CommitSHA,
				URL: fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repo, t.CommitSHA),
			},
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// populateBlobURLs fills in URL fields for a blob
func (s *Service) populateBlobURLs(b *Blob, owner, repo string) {
	b.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/blobs/%s", s.baseURL, owner, repo, b.SHA)
}

// populateCommitURLs fills in URL fields for a git commit
func (s *Service) populateCommitURLs(c *GitCommit, owner, repo string) {
	c.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/commits/%s", s.baseURL, owner, repo, c.SHA)
	c.HTMLURL = fmt.Sprintf("%s/%s/%s/commit/%s", s.baseURL, owner, repo, c.SHA)
	if c.Tree != nil {
		c.Tree.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/trees/%s", s.baseURL, owner, repo, c.Tree.SHA)
	}
	for _, p := range c.Parents {
		p.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/commits/%s", s.baseURL, owner, repo, p.SHA)
	}
}

// populateRefURLs fills in URL fields for a reference
func (s *Service) populateRefURLs(r *Reference, owner, repo string) {
	r.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/ref/%s", s.baseURL, owner, repo, r.Ref)
	if r.Object != nil {
		r.Object.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/%ss/%s", s.baseURL, owner, repo, r.Object.Type, r.Object.SHA)
	}
}
