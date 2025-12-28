// Package git provides Git repository operations for GitHome.
// It wraps the git command-line tool to provide tree, blob, commit,
// and reference operations.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrNotARepo     = errors.New("not a git repository")
	ErrInvalidRef   = errors.New("invalid reference")
	ErrBinaryFile   = errors.New("binary file")
	ErrPathTraversal = errors.New("path traversal detected")
)

// Repository represents a local git repository
type Repository struct {
	path string // Absolute path to .git directory or worktree
}

// Open opens a git repository at the given path
func Open(path string) (*Repository, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// Check if path is a git repository
	gitDir := filepath.Join(absPath, ".git")
	if info, err := os.Stat(gitDir); err != nil || !info.IsDir() {
		// Maybe it's a bare repository
		if info, err := os.Stat(filepath.Join(absPath, "HEAD")); err != nil || info.IsDir() {
			return nil, ErrNotARepo
		}
	}

	return &Repository{path: absPath}, nil
}

// Path returns the repository path
func (r *Repository) Path() string {
	return r.path
}

// git executes a git command in the repository context
func (r *Repository) git(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), errMsg)
	}

	return stdout.Bytes(), nil
}

// gitPipe executes a git command and returns a reader for the output
func (r *Repository) gitPipe(ctx context.Context, args ...string) (io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.path

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &cmdReader{ReadCloser: stdout, cmd: cmd}, nil
}

// cmdReader wraps a pipe to wait for the command to complete
type cmdReader struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (cr *cmdReader) Close() error {
	err := cr.ReadCloser.Close()
	cr.cmd.Wait()
	return err
}

// ResolveRef resolves a reference (branch, tag, SHA) to a full commit SHA
func (r *Repository) ResolveRef(ctx context.Context, ref string) (string, error) {
	if ref == "" {
		ref = "HEAD"
	}

	out, err := r.git(ctx, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", ErrInvalidRef
	}

	return strings.TrimSpace(string(out)), nil
}

// GetDefaultBranch returns the default branch name
func (r *Repository) GetDefaultBranch(ctx context.Context) (string, error) {
	// Try to get symbolic ref of HEAD
	out, err := r.git(ctx, "symbolic-ref", "--short", "HEAD")
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}

	// Fallback to reading HEAD directly for bare repos
	out, err = r.git(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "main", nil // Default fallback
	}

	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		return "main", nil
	}
	return branch, nil
}

// IsValidPath checks if a path is safe (no traversal attacks)
func IsValidPath(path string) bool {
	if path == "" {
		return true
	}

	// Check for .. anywhere in the path (before normalization)
	if strings.Contains(path, "..") {
		return false
	}

	// Normalize and check for absolute paths
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) {
		return false
	}

	// Clean path should not start with ..
	if strings.HasPrefix(clean, "..") {
		return false
	}

	return true
}
