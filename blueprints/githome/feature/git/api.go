package git

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrRefExists   = errors.New("reference already exists")
	ErrInvalidRef  = errors.New("invalid reference")
)

// Blob represents a Git blob
type Blob struct {
	URL      string `json:"url"`
	SHA      string `json:"sha"`
	NodeID   string `json:"node_id"`
	Size     int    `json:"size"`
	Content  string `json:"content,omitempty"`
	Encoding string `json:"encoding"` // utf-8, base64
}

// GitCommit represents a low-level Git commit
type GitCommit struct {
	URL          string        `json:"url"`
	SHA          string        `json:"sha"`
	NodeID       string        `json:"node_id"`
	Author       *CommitAuthor `json:"author"`
	Committer    *CommitAuthor `json:"committer"`
	Message      string        `json:"message"`
	Tree         *TreeRef      `json:"tree"`
	Parents      []*TreeRef    `json:"parents"`
	Verification *Verification `json:"verification,omitempty"`
	HTMLURL      string        `json:"html_url"`
}

// CommitAuthor represents a commit author
type CommitAuthor struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

// Reference represents a Git reference
type Reference struct {
	Ref    string     `json:"ref"`
	NodeID string     `json:"node_id"`
	URL    string     `json:"url"`
	Object *GitObject `json:"object"`
}

// GitObject represents a Git object reference
type GitObject struct {
	Type string `json:"type"` // commit, tag, blob, tree
	SHA  string `json:"sha"`
	URL  string `json:"url"`
}

// Tree represents a Git tree
type Tree struct {
	SHA       string       `json:"sha"`
	URL       string       `json:"url"`
	Tree      []*TreeEntry `json:"tree"`
	Truncated bool         `json:"truncated"`
}

// TreeEntry represents an entry in a tree
type TreeEntry struct {
	Path string `json:"path"`
	Mode string `json:"mode"` // 100644, 100755, 040000, 160000, 120000
	Type string `json:"type"` // blob, tree, commit
	Size int    `json:"size,omitempty"`
	SHA  string `json:"sha"`
	URL  string `json:"url"`
}

// TreeRef represents a reference to a tree
type TreeRef struct {
	SHA string `json:"sha"`
	URL string `json:"url"`
}

// Tag represents a Git tag (annotated)
type Tag struct {
	NodeID       string        `json:"node_id"`
	Tag          string        `json:"tag"`
	SHA          string        `json:"sha"`
	URL          string        `json:"url"`
	Message      string        `json:"message"`
	Tagger       *CommitAuthor `json:"tagger"`
	Object       *GitObject    `json:"object"`
	Verification *Verification `json:"verification,omitempty"`
}

// LightweightTag represents a lightweight tag
type LightweightTag struct {
	Name       string     `json:"name"`
	ZipballURL string     `json:"zipball_url"`
	TarballURL string     `json:"tarball_url"`
	Commit     *CommitRef `json:"commit"`
	NodeID     string     `json:"node_id"`
}

// CommitRef represents a commit reference
type CommitRef struct {
	SHA string `json:"sha"`
	URL string `json:"url"`
}

// Verification represents signature verification
type Verification struct {
	Verified  bool   `json:"verified"`
	Reason    string `json:"reason"`
	Signature string `json:"signature,omitempty"`
	Payload   string `json:"payload,omitempty"`
}

// CreateBlobIn represents input for creating a blob
type CreateBlobIn struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding,omitempty"` // utf-8 (default), base64
}

// CreateGitCommitIn represents input for creating a commit
type CreateGitCommitIn struct {
	Message   string        `json:"message"`
	Tree      string        `json:"tree"`
	Parents   []string      `json:"parents,omitempty"`
	Author    *CommitAuthor `json:"author,omitempty"`
	Committer *CommitAuthor `json:"committer,omitempty"`
	Signature string        `json:"signature,omitempty"`
}

// CreateRefIn represents input for creating a reference
type CreateRefIn struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

// CreateTreeIn represents input for creating a tree
type CreateTreeIn struct {
	BaseTree string           `json:"base_tree,omitempty"`
	Tree     []*TreeEntryIn   `json:"tree"`
}

// TreeEntryIn represents input for a tree entry
type TreeEntryIn struct {
	Path    string `json:"path"`
	Mode    string `json:"mode"` // 100644, 100755, 040000, 160000, 120000
	Type    string `json:"type"` // blob, tree, commit
	SHA     string `json:"sha,omitempty"`
	Content string `json:"content,omitempty"` // For creating blob inline
}

// CreateTagIn represents input for creating an annotated tag
type CreateTagIn struct {
	Tag     string        `json:"tag"`
	Message string        `json:"message"`
	Object  string        `json:"object"`
	Type    string        `json:"type"` // commit, tree, blob
	Tagger  *CommitAuthor `json:"tagger,omitempty"`
}

// ListOpts contains pagination options
type ListOpts struct {
	Page    int `json:"page,omitempty"`
	PerPage int `json:"per_page,omitempty"`
}

// API defines the git service interface
type API interface {
	// Blobs
	GetBlob(ctx context.Context, owner, repo, sha string) (*Blob, error)
	CreateBlob(ctx context.Context, owner, repo string, in *CreateBlobIn) (*Blob, error)

	// Commits
	GetGitCommit(ctx context.Context, owner, repo, sha string) (*GitCommit, error)
	CreateGitCommit(ctx context.Context, owner, repo string, in *CreateGitCommitIn) (*GitCommit, error)

	// References
	GetRef(ctx context.Context, owner, repo, ref string) (*Reference, error)
	ListMatchingRefs(ctx context.Context, owner, repo, ref string) ([]*Reference, error)
	CreateRef(ctx context.Context, owner, repo string, in *CreateRefIn) (*Reference, error)
	UpdateRef(ctx context.Context, owner, repo, ref, sha string, force bool) (*Reference, error)
	DeleteRef(ctx context.Context, owner, repo, ref string) error

	// Trees
	GetTree(ctx context.Context, owner, repo, sha string, recursive bool) (*Tree, error)
	CreateTree(ctx context.Context, owner, repo string, in *CreateTreeIn) (*Tree, error)

	// Tags (annotated)
	GetTag(ctx context.Context, owner, repo, sha string) (*Tag, error)
	CreateTag(ctx context.Context, owner, repo string, in *CreateTagIn) (*Tag, error)

	// Tags (lightweight list)
	ListTags(ctx context.Context, owner, repo string, opts *ListOpts) ([]*LightweightTag, error)
}

// Store defines the data access interface for git objects
// Note: Most git operations will be performed through the git library
// rather than the database, so the store interface is minimal
type Store interface {
	// For caching/indexing git objects if needed
	CacheBlob(ctx context.Context, repoID int64, blob *Blob) error
	GetCachedBlob(ctx context.Context, repoID int64, sha string) (*Blob, error)
}
