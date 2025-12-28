package git

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
)

// Service implements the git API
type Service struct {
	store     Store
	repoStore repos.Store
	baseURL   string
}

// NewService creates a new git service
func NewService(store Store, repoStore repos.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		baseURL:   baseURL,
	}
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

	// Would integrate with git to get blob
	return nil, ErrNotFound
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

	// Would integrate with git to create blob
	// For now return placeholder
	sha := fmt.Sprintf("blob_%d", time.Now().UnixNano())
	blob := &Blob{
		SHA:      sha,
		NodeID:   base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Blob:%s", sha))),
		Size:     len(in.Content),
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

	// Would integrate with git to get commit
	// For now return placeholder
	commit := &GitCommit{
		SHA:     sha,
		NodeID:  base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("GitCommit:%s", sha))),
		Message: "Commit message",
		Author: &CommitAuthor{
			Name:  "Author",
			Email: "author@example.com",
			Date:  time.Now(),
		},
		Committer: &CommitAuthor{
			Name:  "Committer",
			Email: "committer@example.com",
			Date:  time.Now(),
		},
		Parents: []*TreeRef{},
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

	// Would integrate with git to create commit
	sha := fmt.Sprintf("commit_%d", time.Now().UnixNano())
	now := time.Now()

	author := in.Author
	if author == nil {
		author = &CommitAuthor{
			Name:  "Author",
			Email: "author@example.com",
			Date:  now,
		}
	}

	committer := in.Committer
	if committer == nil {
		committer = author
	}

	commit := &GitCommit{
		SHA:       sha,
		NodeID:    base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("GitCommit:%s", sha))),
		Message:   in.Message,
		Author:    author,
		Committer: committer,
		Tree:      &TreeRef{SHA: in.Tree},
		Parents:   []*TreeRef{},
	}

	for _, p := range in.Parents {
		commit.Parents = append(commit.Parents, &TreeRef{SHA: p})
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

	// Would integrate with git to get ref
	reference := &Reference{
		Ref:    "refs/" + ref,
		NodeID: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Ref:%s", ref))),
		Object: &GitObject{
			Type: "commit",
			SHA:  "HEAD",
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

	// Would integrate with git to list matching refs
	return []*Reference{}, nil
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

	// Would integrate with git to create ref
	reference := &Reference{
		Ref:    in.Ref,
		NodeID: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Ref:%s", in.Ref))),
		Object: &GitObject{
			Type: "commit",
			SHA:  in.SHA,
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

	// Would integrate with git to update ref
	reference := &Reference{
		Ref:    "refs/" + ref,
		NodeID: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Ref:%s", ref))),
		Object: &GitObject{
			Type: "commit",
			SHA:  sha,
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

	// Would integrate with git to delete ref
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

	// Would integrate with git to get tree
	tree := &Tree{
		SHA:       sha,
		Tree:      []*TreeEntry{},
		Truncated: false,
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

	// Would integrate with git to create tree
	sha := fmt.Sprintf("tree_%d", time.Now().UnixNano())
	tree := &Tree{
		SHA:       sha,
		Tree:      []*TreeEntry{},
		Truncated: false,
	}

	for _, entry := range in.Tree {
		tree.Tree = append(tree.Tree, &TreeEntry{
			Path: entry.Path,
			Mode: entry.Mode,
			Type: entry.Type,
			SHA:  entry.SHA,
		})
	}

	tree.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/trees/%s", s.baseURL, owner, repo, sha)
	return tree, nil
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

	// Would integrate with git to get tag
	tag := &Tag{
		NodeID:  base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Tag:%s", sha))),
		SHA:     sha,
		Tag:     "v1.0.0",
		Message: "Release v1.0.0",
		Tagger: &CommitAuthor{
			Name:  "Tagger",
			Email: "tagger@example.com",
			Date:  time.Now(),
		},
		Object: &GitObject{
			Type: "commit",
			SHA:  "HEAD",
		},
	}

	tag.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/git/tags/%s", s.baseURL, owner, repo, sha)
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

	// Would integrate with git to create tag
	sha := fmt.Sprintf("tag_%d", time.Now().UnixNano())

	tagger := in.Tagger
	if tagger == nil {
		tagger = &CommitAuthor{
			Name:  "Tagger",
			Email: "tagger@example.com",
			Date:  time.Now(),
		}
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

	// Would integrate with git to list tags
	return []*LightweightTag{}, nil
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
