package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// getTestRepoPath returns the path to a git repository for testing
func getTestRepoPath() string {
	// Use the parent mizu repository as test data
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Walk up to find any git repo root
	for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
	}
	return ""
}

func TestOpen(t *testing.T) {
	repoPath := getTestRepoPath()
	if repoPath == "" {
		t.Skip("no git repository found for testing")
	}

	repo, err := Open(repoPath)
	if err != nil {
		t.Fatalf("Failed to open repository: %v", err)
	}

	if repo.Path() != repoPath {
		t.Errorf("Expected path %s, got %s", repoPath, repo.Path())
	}
}

func TestOpenNotARepo(t *testing.T) {
	_, err := Open("/tmp")
	if err != ErrNotARepo {
		t.Errorf("Expected ErrNotARepo, got %v", err)
	}
}

func TestResolveRef(t *testing.T) {
	repoPath := getTestRepoPath()
	if repoPath == "" {
		t.Skip("no git repository found for testing")
	}

	repo, _ := Open(repoPath)
	ctx := context.Background()

	// Test resolving HEAD
	sha, err := repo.ResolveRef(ctx, "HEAD")
	if err != nil {
		t.Fatalf("Failed to resolve HEAD: %v", err)
	}

	if len(sha) != 40 {
		t.Errorf("Expected 40-char SHA, got %s (len %d)", sha, len(sha))
	}
}

func TestGetDefaultBranch(t *testing.T) {
	repoPath := getTestRepoPath()
	if repoPath == "" {
		t.Skip("no git repository found for testing")
	}

	repo, _ := Open(repoPath)
	ctx := context.Background()

	branch, err := repo.GetDefaultBranch(ctx)
	if err != nil {
		t.Fatalf("Failed to get default branch: %v", err)
	}

	if branch == "" {
		t.Error("Default branch should not be empty")
	}
}

func TestListBranches(t *testing.T) {
	repoPath := getTestRepoPath()
	if repoPath == "" {
		t.Skip("no git repository found for testing")
	}

	repo, _ := Open(repoPath)
	ctx := context.Background()

	branches, err := repo.ListBranches(ctx)
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}

	if len(branches) == 0 {
		t.Error("Expected at least one branch")
	}

	// Check that all branches have required fields
	for _, b := range branches {
		if b.Name == "" {
			t.Error("Branch name should not be empty")
		}
		if b.Type != "branch" {
			t.Errorf("Expected type 'branch', got %s", b.Type)
		}
		if len(b.SHA) != 40 {
			t.Errorf("Expected 40-char SHA, got %s", b.SHA)
		}
	}
}

func TestGetTree(t *testing.T) {
	repoPath := getTestRepoPath()
	if repoPath == "" {
		t.Skip("no git repository found for testing")
	}

	repo, _ := Open(repoPath)
	ctx := context.Background()

	tree, err := repo.GetTree(ctx, "HEAD", "")
	if err != nil {
		t.Fatalf("Failed to get tree: %v", err)
	}

	if len(tree.Entries) == 0 {
		t.Error("Expected at least one entry in root tree")
	}

	// Verify sorting: directories should come before files
	lastDir := -1
	firstFile := -1
	for i, entry := range tree.Entries {
		if entry.IsDir() && lastDir < i {
			lastDir = i
		}
		if entry.IsFile() && firstFile == -1 {
			firstFile = i
		}
	}

	if lastDir > firstFile && firstFile >= 0 {
		t.Error("Directories should be sorted before files")
	}
}

func TestGetBlob(t *testing.T) {
	repoPath := getTestRepoPath()
	if repoPath == "" {
		t.Skip("no git repository found for testing")
	}

	repo, _ := Open(repoPath)
	ctx := context.Background()

	// Get tree to find a .go file
	tree, _ := repo.GetTree(ctx, "HEAD", "")
	var goFile string
	for _, entry := range tree.Entries {
		if filepath.Ext(entry.Name) == ".go" {
			goFile = entry.Name
			break
		}
	}

	if goFile == "" {
		t.Skip("No .go file found in root")
	}

	blob, err := repo.GetBlob(ctx, "HEAD", goFile)
	if err != nil {
		t.Fatalf("Failed to get blob: %v", err)
	}

	if blob.Name != goFile {
		t.Errorf("Expected name %s, got %s", goFile, blob.Name)
	}

	if blob.Language != "Go" {
		t.Errorf("Expected language 'Go', got %s", blob.Language)
	}

	if blob.IsBinary {
		t.Error("Go files should not be binary")
	}

	if blob.Lines == 0 {
		t.Error("Expected non-zero line count")
	}
}

func TestGetCommit(t *testing.T) {
	repoPath := getTestRepoPath()
	if repoPath == "" {
		t.Skip("no git repository found for testing")
	}

	repo, _ := Open(repoPath)
	ctx := context.Background()

	commit, err := repo.GetLatestCommit(ctx, "HEAD")
	if err != nil {
		t.Fatalf("Failed to get latest commit: %v", err)
	}

	if len(commit.SHA) != 40 {
		t.Errorf("Expected 40-char SHA, got %s", commit.SHA)
	}

	if len(commit.ShortSHA) != 7 {
		t.Errorf("Expected 7-char short SHA, got %s", commit.ShortSHA)
	}

	if commit.Author.Name == "" {
		t.Error("Author name should not be empty")
	}

	if commit.Title == "" {
		t.Error("Commit title should not be empty")
	}
}

func TestGetCommitHistory(t *testing.T) {
	repoPath := getTestRepoPath()
	if repoPath == "" {
		t.Skip("no git repository found for testing")
	}

	repo, _ := Open(repoPath)
	ctx := context.Background()

	commits, err := repo.GetCommitHistory(ctx, "HEAD", 5)
	if err != nil {
		t.Fatalf("Failed to get commit history: %v", err)
	}

	if len(commits) == 0 {
		t.Error("Expected at least one commit")
	}

	if len(commits) > 5 {
		t.Errorf("Expected at most 5 commits, got %d", len(commits))
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"main.go", "Go"},
		{"app.js", "JavaScript"},
		{"index.ts", "TypeScript"},
		{"script.py", "Python"},
		{"lib.rs", "Rust"},
		{"Makefile", "Makefile"},
		{"Dockerfile", "Dockerfile"},
		{"style.css", "CSS"},
		{"README.md", "Markdown"},
		{"config.yaml", "YAML"},
		{"unknown.xyz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := DetectLanguage(tt.filename)
			if result != tt.expected {
				t.Errorf("DetectLanguage(%s) = %s, want %s", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestLanguageColor(t *testing.T) {
	tests := []struct {
		language string
		expected string
	}{
		{"Go", "#00ADD8"},
		{"JavaScript", "#f1e05a"},
		{"Python", "#3572A5"},
		{"Unknown", "#808080"},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			result := LanguageColor(tt.language)
			if result != tt.expected {
				t.Errorf("LanguageColor(%s) = %s, want %s", tt.language, result, tt.expected)
			}
		})
	}
}

func TestIsValidPath(t *testing.T) {
	tests := []struct {
		path  string
		valid bool
	}{
		{"", true},
		{"src/main.go", true},
		{"a/b/c/d.txt", true},
		{"../etc/passwd", false},
		{"foo/../bar", false},
		{"/etc/passwd", false},
		{"./foo", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsValidPath(tt.path)
			if result != tt.valid {
				t.Errorf("IsValidPath(%s) = %v, want %v", tt.path, result, tt.valid)
			}
		})
	}
}

func TestPathExists(t *testing.T) {
	repoPath := getTestRepoPath()
	if repoPath == "" {
		t.Skip("no git repository found for testing")
	}

	repo, _ := Open(repoPath)
	ctx := context.Background()

	// Root should exist
	exists, err := repo.PathExists(ctx, "HEAD", "")
	if err != nil {
		t.Fatalf("PathExists error: %v", err)
	}
	if !exists {
		t.Error("Root path should exist")
	}

	// Non-existent path
	exists, err = repo.PathExists(ctx, "HEAD", "this-path-does-not-exist-12345")
	if err != nil {
		t.Fatalf("PathExists error: %v", err)
	}
	if exists {
		t.Error("Non-existent path should not exist")
	}
}

func TestGetPathType(t *testing.T) {
	repoPath := getTestRepoPath()
	if repoPath == "" {
		t.Skip("no git repository found for testing")
	}

	repo, _ := Open(repoPath)
	ctx := context.Background()

	// Root is a tree
	pathType, err := repo.GetPathType(ctx, "HEAD", "")
	if err != nil {
		t.Fatalf("GetPathType error: %v", err)
	}
	if pathType != "tree" {
		t.Errorf("Root should be 'tree', got %s", pathType)
	}

	// Find a file to test
	tree, _ := repo.GetTree(ctx, "HEAD", "")
	for _, entry := range tree.Entries {
		if entry.IsFile() {
			pathType, err := repo.GetPathType(ctx, "HEAD", entry.Path)
			if err != nil {
				t.Fatalf("GetPathType error: %v", err)
			}
			if pathType != "blob" {
				t.Errorf("File should be 'blob', got %s", pathType)
			}
			break
		}
	}
}

func TestRelativeTime(t *testing.T) {
	// Just ensure it doesn't panic
	result := relativeTime(time.Now())
	if result != "just now" {
		t.Errorf("Expected 'just now', got %s", result)
	}
}
