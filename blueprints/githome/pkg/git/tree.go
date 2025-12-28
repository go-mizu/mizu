package git

import (
	"bufio"
	"context"
	"path/filepath"
	"strconv"
	"strings"
)

// TreeEntry represents a file or directory in a git tree
type TreeEntry struct {
	Name string `json:"name"` // File or directory name
	Path string `json:"path"` // Full path from repository root
	Type string `json:"type"` // "blob" (file) or "tree" (directory)
	Mode string `json:"mode"` // Git file mode (100644, 040000, etc.)
	SHA  string `json:"sha"`  // Object SHA
	Size int64  `json:"size"` // Size in bytes (only for blobs)
}

// Tree represents a directory listing
type Tree struct {
	SHA        string       `json:"sha"`
	Path       string       `json:"path"`
	Entries    []*TreeEntry `json:"entries"`
	TotalCount int          `json:"total_count"`
}

// IsDir returns true if the entry is a directory
func (e *TreeEntry) IsDir() bool {
	return e.Type == "tree"
}

// IsFile returns true if the entry is a file
func (e *TreeEntry) IsFile() bool {
	return e.Type == "blob"
}

// GetTree retrieves the tree (directory listing) at the given ref and path
func (r *Repository) GetTree(ctx context.Context, ref, path string) (*Tree, error) {
	if !IsValidPath(path) {
		return nil, ErrPathTraversal
	}

	sha, err := r.ResolveRef(ctx, ref)
	if err != nil {
		return nil, err
	}

	// Build the tree spec
	treeSpec := sha
	if path != "" {
		treeSpec = sha + ":" + path
	}

	// Get tree listing with sizes
	out, err := r.git(ctx, "ls-tree", "-l", treeSpec)
	if err != nil {
		return nil, ErrNotFound
	}

	tree := &Tree{
		SHA:     sha,
		Path:    path,
		Entries: make([]*TreeEntry, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		entry := parseTreeLine(line, path)
		if entry != nil {
			tree.Entries = append(tree.Entries, entry)
		}
	}

	// Sort: directories first, then files, both alphabetically
	sortTreeEntries(tree.Entries)

	tree.TotalCount = len(tree.Entries)
	return tree, nil
}

// GetTreeRecursive retrieves all entries in the tree recursively
func (r *Repository) GetTreeRecursive(ctx context.Context, ref string) (*Tree, error) {
	sha, err := r.ResolveRef(ctx, ref)
	if err != nil {
		return nil, err
	}

	out, err := r.git(ctx, "ls-tree", "-r", "-l", sha)
	if err != nil {
		return nil, ErrNotFound
	}

	tree := &Tree{
		SHA:     sha,
		Path:    "",
		Entries: make([]*TreeEntry, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		entry := parseTreeLine(line, "")
		if entry != nil {
			tree.Entries = append(tree.Entries, entry)
		}
	}

	tree.TotalCount = len(tree.Entries)
	return tree, nil
}

// parseTreeLine parses a line from git ls-tree -l output
// Format: <mode> SP <type> SP <sha> SP <size> TAB <name>
func parseTreeLine(line, basePath string) *TreeEntry {
	// Split by tab to get name
	parts := strings.SplitN(line, "\t", 2)
	if len(parts) != 2 {
		return nil
	}

	name := parts[1]
	meta := strings.Fields(parts[0])
	if len(meta) < 4 {
		return nil
	}

	mode := meta[0]
	objType := meta[1]
	sha := meta[2]
	sizeStr := meta[3]

	var size int64
	if sizeStr != "-" {
		size, _ = strconv.ParseInt(sizeStr, 10, 64)
	}

	// Build full path
	path := name
	if basePath != "" {
		path = filepath.Join(basePath, name)
	}

	return &TreeEntry{
		Name: name,
		Path: path,
		Type: objType,
		Mode: mode,
		SHA:  sha,
		Size: size,
	}
}

// sortTreeEntries sorts entries: directories first, then files, both alphabetically
func sortTreeEntries(entries []*TreeEntry) {
	// Simple bubble sort for small lists
	n := len(entries)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if shouldSwap(entries[j], entries[j+1]) {
				entries[j], entries[j+1] = entries[j+1], entries[j]
			}
		}
	}
}

func shouldSwap(a, b *TreeEntry) bool {
	// Directories come before files
	if a.IsDir() != b.IsDir() {
		return !a.IsDir()
	}
	// Alphabetical within same type
	return strings.ToLower(a.Name) > strings.ToLower(b.Name)
}

// PathExists checks if a path exists in the repository at the given ref
func (r *Repository) PathExists(ctx context.Context, ref, path string) (bool, error) {
	if !IsValidPath(path) {
		return false, ErrPathTraversal
	}

	sha, err := r.ResolveRef(ctx, ref)
	if err != nil {
		return false, err
	}

	if path == "" {
		return true, nil
	}

	_, err = r.git(ctx, "cat-file", "-e", sha+":"+path)
	return err == nil, nil
}

// GetPathType returns "tree" for directories, "blob" for files, or empty if not found
func (r *Repository) GetPathType(ctx context.Context, ref, path string) (string, error) {
	if !IsValidPath(path) {
		return "", ErrPathTraversal
	}

	sha, err := r.ResolveRef(ctx, ref)
	if err != nil {
		return "", err
	}

	if path == "" {
		return "tree", nil
	}

	out, err := r.git(ctx, "cat-file", "-t", sha+":"+path)
	if err != nil {
		return "", ErrNotFound
	}

	return strings.TrimSpace(string(out)), nil
}
