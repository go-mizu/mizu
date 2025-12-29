package git

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// Repository wraps a git repository for object operations
type Repository struct {
	repo *git.Repository
	path string
}

// Open opens an existing repository
func Open(path string) (*Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			return nil, ErrNotARepository
		}
		return nil, fmt.Errorf("open repository: %w", err)
	}
	return &Repository{repo: repo, path: path}, nil
}

// Init creates a new bare repository
func Init(path string) (*Repository, error) {
	repo, err := git.PlainInit(path, true)
	if err != nil {
		return nil, fmt.Errorf("init repository: %w", err)
	}
	return &Repository{repo: repo, path: path}, nil
}

// Clone clones a repository
func Clone(url, path string, opts *CloneOptions) (*Repository, error) {
	cloneOpts := &git.CloneOptions{
		URL: url,
	}
	if opts != nil {
		if opts.Bare {
			cloneOpts.Mirror = true
		}
		if opts.Depth > 0 {
			cloneOpts.Depth = opts.Depth
		}
		if opts.Branch != "" {
			cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(opts.Branch)
		}
	}

	repo, err := git.PlainClone(path, opts != nil && opts.Bare, cloneOpts)
	if err != nil {
		return nil, fmt.Errorf("clone repository: %w", err)
	}
	return &Repository{repo: repo, path: path}, nil
}

// Path returns the repository path
func (r *Repository) Path() string {
	return r.path
}

// validateSHA checks if a SHA is valid hex format
func validateSHA(sha string) error {
	if len(sha) != 40 {
		return ErrInvalidSHA
	}
	_, err := hex.DecodeString(sha)
	if err != nil {
		return ErrInvalidSHA
	}
	return nil
}

// GetBlob retrieves a blob by SHA
func (r *Repository) GetBlob(sha string) (*Blob, error) {
	if err := validateSHA(sha); err != nil {
		return nil, err
	}

	hash := plumbing.NewHash(sha)
	blob, err := r.repo.BlobObject(hash)
	if err != nil {
		if err == plumbing.ErrObjectNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get blob: %w", err)
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, fmt.Errorf("blob reader: %w", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read blob: %w", err)
	}

	return &Blob{
		SHA:     sha,
		Size:    blob.Size,
		Content: content,
	}, nil
}

// CreateBlob creates a new blob and returns its SHA
func (r *Repository) CreateBlob(content []byte) (string, error) {
	obj := r.repo.Storer.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	obj.SetSize(int64(len(content)))

	writer, err := obj.Writer()
	if err != nil {
		return "", fmt.Errorf("blob writer: %w", err)
	}

	if _, err := writer.Write(content); err != nil {
		writer.Close()
		return "", fmt.Errorf("write blob: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close blob: %w", err)
	}

	hash, err := r.repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", fmt.Errorf("store blob: %w", err)
	}

	return hash.String(), nil
}

// GetCommit retrieves a commit by SHA
func (r *Repository) GetCommit(sha string) (*Commit, error) {
	if err := validateSHA(sha); err != nil {
		return nil, err
	}

	hash := plumbing.NewHash(sha)
	commit, err := r.repo.CommitObject(hash)
	if err != nil {
		if err == plumbing.ErrObjectNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get commit: %w", err)
	}

	parents := make([]string, 0, commit.NumParents())
	for _, p := range commit.ParentHashes {
		parents = append(parents, p.String())
	}

	return &Commit{
		SHA:     sha,
		TreeSHA: commit.TreeHash.String(),
		Parents: parents,
		Author: Signature{
			Name:  commit.Author.Name,
			Email: commit.Author.Email,
			When:  commit.Author.When,
		},
		Committer: Signature{
			Name:  commit.Committer.Name,
			Email: commit.Committer.Email,
			When:  commit.Committer.When,
		},
		Message: commit.Message,
	}, nil
}

// CreateCommit creates a new commit
func (r *Repository) CreateCommit(opts *CreateCommitOpts) (string, error) {
	if err := validateSHA(opts.TreeSHA); err != nil {
		return "", fmt.Errorf("invalid tree SHA: %w", err)
	}

	parentHashes := make([]plumbing.Hash, 0, len(opts.Parents))
	for _, p := range opts.Parents {
		if err := validateSHA(p); err != nil {
			return "", fmt.Errorf("invalid parent SHA %s: %w", p, err)
		}
		parentHashes = append(parentHashes, plumbing.NewHash(p))
	}

	commit := &object.Commit{
		TreeHash:     plumbing.NewHash(opts.TreeSHA),
		ParentHashes: parentHashes,
		Author: object.Signature{
			Name:  opts.Author.Name,
			Email: opts.Author.Email,
			When:  opts.Author.When,
		},
		Committer: object.Signature{
			Name:  opts.Committer.Name,
			Email: opts.Committer.Email,
			When:  opts.Committer.When,
		},
		Message: opts.Message,
	}

	obj := r.repo.Storer.NewEncodedObject()
	if err := commit.Encode(obj); err != nil {
		return "", fmt.Errorf("encode commit: %w", err)
	}

	hash, err := r.repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", fmt.Errorf("store commit: %w", err)
	}

	return hash.String(), nil
}

// GetTree retrieves a tree by SHA
func (r *Repository) GetTree(sha string) (*Tree, error) {
	return r.getTree(sha, false, 0)
}

// GetTreeRecursive retrieves a tree with all nested entries
func (r *Repository) GetTreeRecursive(sha string) (*Tree, error) {
	return r.getTree(sha, true, 0)
}

func (r *Repository) getTree(sha string, recursive bool, depth int) (*Tree, error) {
	if err := validateSHA(sha); err != nil {
		return nil, err
	}

	const maxDepth = 100
	const maxEntries = 100000

	hash := plumbing.NewHash(sha)
	tree, err := r.repo.TreeObject(hash)
	if err != nil {
		if err == plumbing.ErrObjectNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get tree: %w", err)
	}

	entries := make([]TreeEntry, 0, len(tree.Entries))
	truncated := false

	for _, entry := range tree.Entries {
		if len(entries) >= maxEntries {
			truncated = true
			break
		}

		te := TreeEntry{
			Name: entry.Name,
			Mode: fileModeToMode(entry.Mode),
			SHA:  entry.Hash.String(),
		}

		if entry.Mode.IsFile() {
			te.Type = ObjectBlob
			// Get blob size
			blob, err := r.repo.BlobObject(entry.Hash)
			if err == nil {
				te.Size = blob.Size
			}
		} else if entry.Mode == filemode.Dir {
			te.Type = ObjectTree
		} else if entry.Mode == filemode.Submodule {
			te.Type = ObjectCommit
		}

		entries = append(entries, te)

		// Recurse into subdirectories
		if recursive && entry.Mode == filemode.Dir && depth < maxDepth {
			subTree, err := r.getTree(entry.Hash.String(), true, depth+1)
			if err == nil {
				for _, subEntry := range subTree.Entries {
					if len(entries) >= maxEntries {
						truncated = true
						break
					}
					subEntry.Name = filepath.Join(entry.Name, subEntry.Name)
					entries = append(entries, subEntry)
				}
				if subTree.Truncated {
					truncated = true
				}
			}
		}
	}

	return &Tree{
		SHA:       sha,
		Entries:   entries,
		Truncated: truncated,
	}, nil
}

func fileModeToMode(fm filemode.FileMode) FileMode {
	switch fm {
	case filemode.Regular:
		return ModeFile
	case filemode.Executable:
		return ModeExecutable
	case filemode.Symlink:
		return ModeSymlink
	case filemode.Submodule:
		return ModeSubmodule
	case filemode.Dir:
		return ModeDir
	default:
		return ModeFile
	}
}

func modeToFileMode(m FileMode) filemode.FileMode {
	switch m {
	case ModeFile:
		return filemode.Regular
	case ModeExecutable:
		return filemode.Executable
	case ModeSymlink:
		return filemode.Symlink
	case ModeSubmodule:
		return filemode.Submodule
	case ModeDir:
		return filemode.Dir
	default:
		return filemode.Regular
	}
}

// CreateTree creates a new tree
func (r *Repository) CreateTree(opts *CreateTreeOpts) (string, error) {
	var baseEntries map[string]object.TreeEntry

	// Load base tree if specified
	if opts.BaseSHA != "" {
		if err := validateSHA(opts.BaseSHA); err != nil {
			return "", fmt.Errorf("invalid base tree SHA: %w", err)
		}
		baseTree, err := r.repo.TreeObject(plumbing.NewHash(opts.BaseSHA))
		if err != nil {
			return "", fmt.Errorf("get base tree: %w", err)
		}
		baseEntries = make(map[string]object.TreeEntry, len(baseTree.Entries))
		for _, e := range baseTree.Entries {
			baseEntries[e.Name] = e
		}
	} else {
		baseEntries = make(map[string]object.TreeEntry)
	}

	// Process entries
	for _, input := range opts.Entries {
		// Handle nested paths by creating intermediate trees
		parts := strings.Split(input.Path, "/")
		if len(parts) > 1 {
			// For nested paths, we need to create/modify subtrees
			// This is a simplified implementation - full implementation would recursively build trees
			continue
		}

		// Direct entry
		var hash plumbing.Hash
		if input.SHA != "" {
			if err := validateSHA(input.SHA); err != nil {
				return "", fmt.Errorf("invalid entry SHA for %s: %w", input.Path, err)
			}
			hash = plumbing.NewHash(input.SHA)
		} else if len(input.Content) > 0 {
			// Create blob from content
			sha, err := r.CreateBlob(input.Content)
			if err != nil {
				return "", fmt.Errorf("create blob for %s: %w", input.Path, err)
			}
			hash = plumbing.NewHash(sha)
		} else {
			// Delete entry (don't add to new tree)
			delete(baseEntries, input.Path)
			continue
		}

		baseEntries[input.Path] = object.TreeEntry{
			Name: input.Path,
			Mode: modeToFileMode(input.Mode),
			Hash: hash,
		}
	}

	// Build final tree
	entries := make([]object.TreeEntry, 0, len(baseEntries))
	for _, e := range baseEntries {
		entries = append(entries, e)
	}

	tree := &object.Tree{Entries: entries}
	obj := r.repo.Storer.NewEncodedObject()
	if err := tree.Encode(obj); err != nil {
		return "", fmt.Errorf("encode tree: %w", err)
	}

	hash, err := r.repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", fmt.Errorf("store tree: %w", err)
	}

	return hash.String(), nil
}

// GetTag retrieves an annotated tag by SHA
func (r *Repository) GetTag(sha string) (*Tag, error) {
	if err := validateSHA(sha); err != nil {
		return nil, err
	}

	hash := plumbing.NewHash(sha)
	tag, err := r.repo.TagObject(hash)
	if err != nil {
		if err == plumbing.ErrObjectNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get tag: %w", err)
	}

	var targetType ObjectType
	switch tag.TargetType {
	case plumbing.CommitObject:
		targetType = ObjectCommit
	case plumbing.TreeObject:
		targetType = ObjectTree
	case plumbing.BlobObject:
		targetType = ObjectBlob
	case plumbing.TagObject:
		targetType = ObjectTag
	}

	return &Tag{
		SHA:        sha,
		Name:       tag.Name,
		TargetSHA:  tag.Target.String(),
		TargetType: targetType,
		Message:    tag.Message,
		Tagger: Signature{
			Name:  tag.Tagger.Name,
			Email: tag.Tagger.Email,
			When:  tag.Tagger.When,
		},
	}, nil
}

// CreateTag creates an annotated tag
func (r *Repository) CreateTag(opts *CreateTagOpts) (string, error) {
	if err := validateSHA(opts.TargetSHA); err != nil {
		return "", fmt.Errorf("invalid target SHA: %w", err)
	}

	var targetType plumbing.ObjectType
	switch opts.TargetType {
	case ObjectCommit:
		targetType = plumbing.CommitObject
	case ObjectTree:
		targetType = plumbing.TreeObject
	case ObjectBlob:
		targetType = plumbing.BlobObject
	case ObjectTag:
		targetType = plumbing.TagObject
	default:
		targetType = plumbing.CommitObject
	}

	tag := &object.Tag{
		Name:       opts.Name,
		Target:     plumbing.NewHash(opts.TargetSHA),
		TargetType: targetType,
		Message:    opts.Message,
		Tagger: object.Signature{
			Name:  opts.Tagger.Name,
			Email: opts.Tagger.Email,
			When:  opts.Tagger.When,
		},
	}

	obj := r.repo.Storer.NewEncodedObject()
	if err := tag.Encode(obj); err != nil {
		return "", fmt.Errorf("encode tag: %w", err)
	}

	hash, err := r.repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", fmt.Errorf("store tag: %w", err)
	}

	return hash.String(), nil
}

// ListTags returns all tag references
func (r *Repository) ListTags() ([]*TagRef, error) {
	refs, err := r.repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}

	var tags []*TagRef
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()

		// Resolve to commit SHA (handle both annotated and lightweight tags)
		sha := ref.Hash().String()

		// Try to get annotated tag
		tagObj, err := r.repo.TagObject(ref.Hash())
		if err == nil {
			// It's an annotated tag, get the target commit
			commit, err := tagObj.Commit()
			if err == nil {
				sha = commit.Hash.String()
			}
		}

		tags = append(tags, &TagRef{
			Name:      name,
			CommitSHA: sha,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}

	return tags, nil
}

// GetRef retrieves a reference
func (r *Repository) GetRef(name string) (*Ref, error) {
	refName := normalizeRefName(name)
	ref, err := r.repo.Reference(refName, true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return nil, ErrRefNotFound
		}
		return nil, fmt.Errorf("get ref: %w", err)
	}

	sha := ref.Hash().String()
	objType := ObjectCommit

	// Determine object type
	obj, err := r.repo.Object(plumbing.AnyObject, ref.Hash())
	if err == nil {
		switch obj.Type() {
		case plumbing.CommitObject:
			objType = ObjectCommit
		case plumbing.TreeObject:
			objType = ObjectTree
		case plumbing.BlobObject:
			objType = ObjectBlob
		case plumbing.TagObject:
			objType = ObjectTag
		}
	}

	return &Ref{
		Name:       ref.Name().String(),
		SHA:        sha,
		ObjectType: objType,
	}, nil
}

func normalizeRefName(name string) plumbing.ReferenceName {
	if strings.HasPrefix(name, "refs/") {
		return plumbing.ReferenceName(name)
	}
	// Try common prefixes
	if strings.HasPrefix(name, "heads/") {
		return plumbing.ReferenceName("refs/" + name)
	}
	if strings.HasPrefix(name, "tags/") {
		return plumbing.ReferenceName("refs/" + name)
	}
	// Default to heads
	return plumbing.ReferenceName("refs/heads/" + name)
}

// ListRefs returns references matching a pattern
func (r *Repository) ListRefs(pattern string) ([]*Ref, error) {
	iter, err := r.repo.References()
	if err != nil {
		return nil, fmt.Errorf("list refs: %w", err)
	}

	var refs []*Ref
	prefix := ""
	if pattern != "" {
		if !strings.HasPrefix(pattern, "refs/") {
			prefix = "refs/" + pattern
		} else {
			prefix = pattern
		}
	}

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().String()
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			return nil
		}

		objType := ObjectCommit
		obj, err := r.repo.Object(plumbing.AnyObject, ref.Hash())
		if err == nil {
			switch obj.Type() {
			case plumbing.CommitObject:
				objType = ObjectCommit
			case plumbing.TreeObject:
				objType = ObjectTree
			case plumbing.BlobObject:
				objType = ObjectBlob
			case plumbing.TagObject:
				objType = ObjectTag
			}
		}

		refs = append(refs, &Ref{
			Name:       name,
			SHA:        ref.Hash().String(),
			ObjectType: objType,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterate refs: %w", err)
	}

	return refs, nil
}

// CreateRef creates a new reference
func (r *Repository) CreateRef(name, sha string) error {
	if err := validateSHA(sha); err != nil {
		return err
	}

	refName := normalizeRefName(name)

	// Check if ref already exists
	_, err := r.repo.Reference(refName, false)
	if err == nil {
		return ErrRefExists
	}
	if err != plumbing.ErrReferenceNotFound {
		return fmt.Errorf("check ref: %w", err)
	}

	ref := plumbing.NewHashReference(refName, plumbing.NewHash(sha))
	if err := r.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("create ref: %w", err)
	}

	return nil
}

// UpdateRef updates a reference
func (r *Repository) UpdateRef(name, sha string, force bool) error {
	if err := validateSHA(sha); err != nil {
		return err
	}

	refName := normalizeRefName(name)

	// Get current ref
	oldRef, err := r.repo.Reference(refName, true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return ErrRefNotFound
		}
		return fmt.Errorf("get ref: %w", err)
	}

	// Check for fast-forward if not forcing
	if !force {
		newHash := plumbing.NewHash(sha)
		oldHash := oldRef.Hash()

		// Check if new commit is ancestor of old (would be backwards)
		newCommit, err := r.repo.CommitObject(newHash)
		if err == nil {
			oldCommit, err := r.repo.CommitObject(oldHash)
			if err == nil {
				isAncestor, err := newCommit.IsAncestor(oldCommit)
				if err == nil && isAncestor {
					return ErrNonFastForward
				}
			}
		}
	}

	ref := plumbing.NewHashReference(refName, plumbing.NewHash(sha))
	if err := r.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("update ref: %w", err)
	}

	return nil
}

// DeleteRef deletes a reference
func (r *Repository) DeleteRef(name string) error {
	refName := normalizeRefName(name)

	// Check if ref exists
	_, err := r.repo.Reference(refName, false)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return ErrRefNotFound
		}
		return fmt.Errorf("check ref: %w", err)
	}

	if err := r.repo.Storer.RemoveReference(refName); err != nil {
		return fmt.Errorf("delete ref: %w", err)
	}

	return nil
}

// Head returns the HEAD reference
func (r *Repository) Head() (*Ref, error) {
	ref, err := r.repo.Head()
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return nil, ErrEmptyRepository
		}
		return nil, fmt.Errorf("get HEAD: %w", err)
	}

	return &Ref{
		Name:       ref.Name().String(),
		SHA:        ref.Hash().String(),
		ObjectType: ObjectCommit,
	}, nil
}

// SetHead sets the HEAD reference
func (r *Repository) SetHead(ref string) error {
	refName := normalizeRefName(ref)
	symbolic := plumbing.NewSymbolicReference(plumbing.HEAD, refName)
	return r.repo.Storer.SetReference(symbolic)
}

// ResolveRef resolves a ref name to a commit SHA
func (r *Repository) ResolveRef(name string) (string, error) {
	ref, err := r.GetRef(name)
	if err != nil {
		return "", err
	}

	// If it's a tag, resolve to the tagged object
	if ref.ObjectType == ObjectTag {
		tag, err := r.GetTag(ref.SHA)
		if err == nil {
			return tag.TargetSHA, nil
		}
	}

	return ref.SHA, nil
}

// ObjectExists checks if an object exists
func (r *Repository) ObjectExists(sha string) bool {
	if err := validateSHA(sha); err != nil {
		return false
	}
	hash := plumbing.NewHash(sha)
	_, err := r.repo.Object(plumbing.AnyObject, hash)
	return err == nil
}

// GetObjectType returns the type of an object
func (r *Repository) GetObjectType(sha string) (ObjectType, error) {
	if err := validateSHA(sha); err != nil {
		return "", err
	}

	hash := plumbing.NewHash(sha)
	obj, err := r.repo.Object(plumbing.AnyObject, hash)
	if err != nil {
		if err == plumbing.ErrObjectNotFound {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("get object: %w", err)
	}

	switch obj.Type() {
	case plumbing.CommitObject:
		return ObjectCommit, nil
	case plumbing.TreeObject:
		return ObjectTree, nil
	case plumbing.BlobObject:
		return ObjectBlob, nil
	case plumbing.TagObject:
		return ObjectTag, nil
	default:
		return "", fmt.Errorf("unknown object type: %s", obj.Type())
	}
}

// Log returns commit history starting from a ref
func (r *Repository) Log(ref string, limit int) ([]*Commit, error) {
	sha, err := r.ResolveRef(ref)
	if err != nil {
		return nil, err
	}

	hash := plumbing.NewHash(sha)
	iter, err := r.repo.Log(&git.LogOptions{From: hash})
	if err != nil {
		return nil, fmt.Errorf("log: %w", err)
	}

	var commits []*Commit
	err = iter.ForEach(func(c *object.Commit) error {
		if limit > 0 && len(commits) >= limit {
			return storer.ErrStop
		}

		parents := make([]string, 0, c.NumParents())
		for _, p := range c.ParentHashes {
			parents = append(parents, p.String())
		}

		commits = append(commits, &Commit{
			SHA:     c.Hash.String(),
			TreeSHA: c.TreeHash.String(),
			Parents: parents,
			Author: Signature{
				Name:  c.Author.Name,
				Email: c.Author.Email,
				When:  c.Author.When,
			},
			Committer: Signature{
				Name:  c.Committer.Name,
				Email: c.Committer.Email,
				When:  c.Committer.When,
			},
			Message: c.Message,
		})
		return nil
	})
	if err != nil && err != storer.ErrStop {
		return nil, fmt.Errorf("iterate commits: %w", err)
	}

	return commits, nil
}

// Diff returns the diff between two commits
func (r *Repository) Diff(fromSHA, toSHA string) (string, error) {
	if err := validateSHA(fromSHA); err != nil {
		return "", fmt.Errorf("invalid from SHA: %w", err)
	}
	if err := validateSHA(toSHA); err != nil {
		return "", fmt.Errorf("invalid to SHA: %w", err)
	}

	fromCommit, err := r.repo.CommitObject(plumbing.NewHash(fromSHA))
	if err != nil {
		return "", fmt.Errorf("get from commit: %w", err)
	}

	toCommit, err := r.repo.CommitObject(plumbing.NewHash(toSHA))
	if err != nil {
		return "", fmt.Errorf("get to commit: %w", err)
	}

	patch, err := fromCommit.Patch(toCommit)
	if err != nil {
		return "", fmt.Errorf("create patch: %w", err)
	}

	var buf bytes.Buffer
	if err := patch.Encode(&buf); err != nil {
		return "", fmt.Errorf("encode patch: %w", err)
	}

	return buf.String(), nil
}

// InitWithCommit initializes a repo with an initial commit
func InitWithCommit(path string, author Signature, message string) (*Repository, string, error) {
	r, err := Init(path)
	if err != nil {
		return nil, "", err
	}

	// Create empty tree
	treeSHA, err := r.CreateTree(&CreateTreeOpts{})
	if err != nil {
		return nil, "", fmt.Errorf("create empty tree: %w", err)
	}

	// Create initial commit
	if author.When.IsZero() {
		author.When = time.Now()
	}

	commitSHA, err := r.CreateCommit(&CreateCommitOpts{
		Message:   message,
		TreeSHA:   treeSHA,
		Parents:   nil,
		Author:    author,
		Committer: author,
	})
	if err != nil {
		return nil, "", fmt.Errorf("create initial commit: %w", err)
	}

	// Set up main branch
	if err := r.CreateRef("refs/heads/main", commitSHA); err != nil {
		return nil, "", fmt.Errorf("create main ref: %w", err)
	}

	// Set HEAD to main
	if err := r.SetHead("refs/heads/main"); err != nil {
		return nil, "", fmt.Errorf("set HEAD: %w", err)
	}

	return r, commitSHA, nil
}

// TreeEntryWithCommit extends TreeEntry with last commit info
type TreeEntryWithCommit struct {
	TreeEntry
	LastCommit *Commit
}

// GetTreeWithLastCommits returns tree entries with the last commit that modified each entry
func (r *Repository) GetTreeWithLastCommits(sha string, maxCommits int) ([]*TreeEntryWithCommit, error) {
	// Get commit object first
	commit, err := r.repo.CommitObject(plumbing.NewHash(sha))
	if err != nil {
		return nil, fmt.Errorf("get commit: %w", err)
	}

	// Get tree from commit
	tree, err := r.GetTree(commit.TreeHash.String())
	if err != nil {
		return nil, err
	}

	// Use native git commands for all files - much faster than go-git for file history
	lastCommits := make(map[string]*object.Commit)
	type result struct {
		fileName string
		commit   *object.Commit
	}
	results := make(chan result, len(tree.Entries))
	var wg sync.WaitGroup

	for _, e := range tree.Entries {
		wg.Add(1)
		go func(fn string) {
			defer wg.Done()
			cmd := exec.Command("git", "log", "-1", "--format=%H", "--", fn)
			cmd.Dir = r.path
			output, err := cmd.Output()
			if err != nil {
				return
			}
			commitSHA := strings.TrimSpace(string(output))
			if commitSHA == "" {
				return
			}
			if c, err := r.repo.CommitObject(plumbing.NewHash(commitSHA)); err == nil {
				results <- result{fileName: fn, commit: c}
			}
		}(e.Name)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		lastCommits[r.fileName] = r.commit
	}

	// Build result
	entries := make([]*TreeEntryWithCommit, len(tree.Entries))
	for i, e := range tree.Entries {
		entry := &TreeEntryWithCommit{TreeEntry: e}
		if c, ok := lastCommits[e.Name]; ok {
			parents := make([]string, 0, c.NumParents())
			for _, p := range c.ParentHashes {
				parents = append(parents, p.String())
			}
			entry.LastCommit = &Commit{
				SHA:     c.Hash.String(),
				TreeSHA: c.TreeHash.String(),
				Parents: parents,
				Author: Signature{
					Name:  c.Author.Name,
					Email: c.Author.Email,
					When:  c.Author.When,
				},
				Committer: Signature{
					Name:  c.Committer.Name,
					Email: c.Committer.Email,
					When:  c.Committer.When,
				},
				Message: c.Message,
			}
		}
		entries[i] = entry
	}

	return entries, nil
}

// GetTreeWithLastCommitsForPath returns tree entries for a specific path with last commit info
func (r *Repository) GetTreeWithLastCommitsForPath(sha, dirPath string) ([]*TreeEntryWithCommit, error) {
	// Get commit object first
	commit, err := r.repo.CommitObject(plumbing.NewHash(sha))
	if err != nil {
		return nil, fmt.Errorf("get commit: %w", err)
	}

	// Get tree from commit
	commitTree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	// Navigate to target directory
	dirPath = strings.TrimPrefix(dirPath, "/")
	dirPath = strings.TrimSuffix(dirPath, "/")

	var targetTree *object.Tree
	if dirPath == "" {
		targetTree = commitTree
	} else {
		targetTree, err = commitTree.Tree(dirPath)
		if err != nil {
			return nil, fmt.Errorf("tree not found: %w", err)
		}
	}

	// Build basic tree entries
	tree := &Tree{Entries: make([]TreeEntry, 0, len(targetTree.Entries))}
	for _, entry := range targetTree.Entries {
		entryType := ObjectBlob
		mode := ModeFile
		if entry.Mode == filemode.Dir {
			entryType = ObjectTree
			mode = ModeDir
		} else if entry.Mode == filemode.Executable {
			mode = ModeExecutable
		} else if entry.Mode == filemode.Symlink {
			mode = ModeSymlink
		} else if entry.Mode == filemode.Submodule {
			mode = ModeSubmodule
		}
		tree.Entries = append(tree.Entries, TreeEntry{
			Name: entry.Name,
			Mode: mode,
			Type: entryType,
			SHA:  entry.Hash.String(),
		})
	}

	// Get last commit info for each entry using native git (fast)
	lastCommits := make(map[string]*object.Commit)
	type result struct {
		fileName string
		commit   *object.Commit
	}
	results := make(chan result, len(tree.Entries))
	var wg sync.WaitGroup

	for _, e := range tree.Entries {
		wg.Add(1)
		go func(fn string) {
			defer wg.Done()
			// Build full path for git log
			fullPath := fn
			if dirPath != "" {
				fullPath = dirPath + "/" + fn
			}
			// Use native git log command - much faster than go-git
			cmd := exec.Command("git", "log", "-1", "--format=%H", "--", fullPath)
			cmd.Dir = r.path
			output, err := cmd.Output()
			if err != nil {
				return
			}
			sha := strings.TrimSpace(string(output))
			if sha == "" {
				return
			}
			// Get the commit object from go-git
			if c, err := r.repo.CommitObject(plumbing.NewHash(sha)); err == nil {
				results <- result{fileName: fn, commit: c}
			}
		}(e.Name)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		lastCommits[r.fileName] = r.commit
	}

	// Build result
	entries := make([]*TreeEntryWithCommit, len(tree.Entries))
	for i, e := range tree.Entries {
		entry := &TreeEntryWithCommit{TreeEntry: e}
		if c, ok := lastCommits[e.Name]; ok {
			parents := make([]string, 0, c.NumParents())
			for _, p := range c.ParentHashes {
				parents = append(parents, p.String())
			}
			entry.LastCommit = &Commit{
				SHA:     c.Hash.String(),
				TreeSHA: c.TreeHash.String(),
				Parents: parents,
				Author: Signature{
					Name:  c.Author.Name,
					Email: c.Author.Email,
					When:  c.Author.When,
				},
				Committer: Signature{
					Name:  c.Committer.Name,
					Email: c.Committer.Email,
					When:  c.Committer.When,
				},
				Message: c.Message,
			}
		}
		entries[i] = entry
	}

	return entries, nil
}

// FileLog returns commits that modified a specific file
func (r *Repository) FileLog(ref, path string, limit int) ([]*Commit, error) {
	sha, err := r.ResolveRef(ref)
	if err != nil {
		return nil, err
	}

	hash := plumbing.NewHash(sha)
	iter, err := r.repo.Log(&git.LogOptions{
		From:     hash,
		FileName: &path,
	})
	if err != nil {
		return nil, fmt.Errorf("log: %w", err)
	}

	var commits []*Commit
	err = iter.ForEach(func(c *object.Commit) error {
		if limit > 0 && len(commits) >= limit {
			return storer.ErrStop
		}

		parents := make([]string, 0, c.NumParents())
		for _, p := range c.ParentHashes {
			parents = append(parents, p.String())
		}

		commits = append(commits, &Commit{
			SHA:     c.Hash.String(),
			TreeSHA: c.TreeHash.String(),
			Parents: parents,
			Author: Signature{
				Name:  c.Author.Name,
				Email: c.Author.Email,
				When:  c.Author.When,
			},
			Committer: Signature{
				Name:  c.Committer.Name,
				Email: c.Committer.Email,
				When:  c.Committer.When,
			},
			Message: c.Message,
		})
		return nil
	})
	if err != nil && err != storer.ErrStop {
		return nil, fmt.Errorf("iterate commits: %w", err)
	}

	return commits, nil
}

// BlameLine represents a line with blame information
type BlameLine struct {
	LineNumber int
	Content    string
	CommitSHA  string
	Author     Signature
}

// BlameResult contains blame information for a file
type BlameResult struct {
	Path  string
	Lines []*BlameLine
}

// Blame returns blame information for a file
func (r *Repository) Blame(ref, path string) (*BlameResult, error) {
	sha, err := r.ResolveRef(ref)
	if err != nil {
		return nil, err
	}

	commit, err := r.repo.CommitObject(plumbing.NewHash(sha))
	if err != nil {
		return nil, fmt.Errorf("get commit: %w", err)
	}

	// Use go-git's blame
	blameResult, err := git.Blame(commit, path)
	if err != nil {
		return nil, fmt.Errorf("blame: %w", err)
	}

	lines := make([]*BlameLine, len(blameResult.Lines))
	for i, line := range blameResult.Lines {
		lines[i] = &BlameLine{
			LineNumber: i + 1,
			Content:    line.Text,
			CommitSHA:  line.Hash.String(),
			Author: Signature{
				Name:  line.AuthorName,
				Email: line.Author, // Author field in go-git Line is the email
				When:  line.Date,
			},
		}
	}

	return &BlameResult{
		Path:  path,
		Lines: lines,
	}, nil
}

// CommitCount returns the total number of commits from a ref
func (r *Repository) CommitCount(ref string) (int, error) {
	// Use native git command - much faster than iterating through all commits
	cmd := exec.Command("git", "rev-list", "--count", ref)
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("rev-list: %w", err)
	}
	var count int
	_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count)
	if err != nil {
		return 0, fmt.Errorf("parse count: %w", err)
	}
	return count, nil
}
