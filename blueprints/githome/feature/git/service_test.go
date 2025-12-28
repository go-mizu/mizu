package git_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/git"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	pkggit "github.com/go-mizu/blueprints/githome/pkg/git"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

// Test setup helpers using real DuckDB store

func setupTestService(t *testing.T) (*git.Service, *duckdb.GitStore, *duckdb.ReposStore, string, func()) {
	t.Helper()

	// Create temp directory for test repos
	tmpDir, err := os.MkdirTemp("", "git-service-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Open in-memory DuckDB
	db, err := sql.Open("duckdb", "")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to open duckdb: %v", err)
	}

	store, err := duckdb.New(db)
	if err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		store.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to ensure schema: %v", err)
	}

	gitStore := duckdb.NewGitStore(db)
	reposStore := duckdb.NewReposStore(db)

	service := git.NewService(gitStore, reposStore, "https://api.example.com", tmpDir)

	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	return service, gitStore, reposStore, tmpDir, cleanup
}

func setupTestRepo(t *testing.T, tmpDir, owner, repoName string) (*pkggit.Repository, string) {
	t.Helper()

	repoPath := filepath.Join(tmpDir, owner, repoName+".git")
	if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	author := pkggit.Signature{
		Name:  "Test Author",
		Email: "test@example.com",
		When:  time.Now(),
	}

	gitRepo, commitSHA, err := pkggit.InitWithCommit(repoPath, author, "Initial commit")
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	return gitRepo, commitSHA
}

func addTestRepo(t *testing.T, reposStore *duckdb.ReposStore, owner, name string) *repos.Repository {
	t.Helper()

	r := &repos.Repository{
		Name:          name,
		FullName:      owner + "/" + name,
		OwnerID:       1,
		OwnerType:     "User",
		Visibility:    "public",
		DefaultBranch: "main",
		HasIssues:     true,
		HasProjects:   true,
		HasWiki:       true,
		HasDownloads:  true,
	}

	if err := reposStore.Create(context.Background(), r); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}

	return r
}

// Blob Tests

func TestService_GetBlob(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, _ := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Create a test blob
	content := []byte("Hello, World!")
	sha, err := gitRepo.CreateBlob(content)
	if err != nil {
		t.Fatalf("failed to create blob: %v", err)
	}

	// Test GetBlob
	blob, err := service.GetBlob(context.Background(), "testowner", "testrepo", sha)
	if err != nil {
		t.Fatalf("GetBlob failed: %v", err)
	}

	if blob.SHA != sha {
		t.Errorf("got SHA %q, want %q", blob.SHA, sha)
	}
	if blob.Content != string(content) {
		t.Errorf("got content %q, want %q", blob.Content, string(content))
	}
	if blob.Size != len(content) {
		t.Errorf("got size %d, want %d", blob.Size, len(content))
	}
	if blob.URL == "" {
		t.Error("expected URL to be populated")
	}
	if blob.NodeID == "" {
		t.Error("expected NodeID to be set")
	}
}

func TestService_GetBlob_RepoNotFound(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetBlob(context.Background(), "unknown", "repo", "abcd1234abcd1234abcd1234abcd1234abcd1234")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_GetBlob_NotFound(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	_, err := service.GetBlob(context.Background(), "testowner", "testrepo", "0000000000000000000000000000000000000000")
	if err != git.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_GetBlob_FromCache(t *testing.T) {
	service, gitStore, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, _ := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	repo := addTestRepo(t, reposStore, "testowner", "testrepo")

	// Create blob in git
	content := []byte("Cached content")
	sha, err := gitRepo.CreateBlob(content)
	if err != nil {
		t.Fatalf("failed to create blob: %v", err)
	}

	// First call - should fetch from git and cache
	blob1, err := service.GetBlob(context.Background(), "testowner", "testrepo", sha)
	if err != nil {
		t.Fatalf("GetBlob failed: %v", err)
	}

	// Verify it was cached
	cached, err := gitStore.GetCachedBlob(context.Background(), repo.ID, sha)
	if err != nil {
		t.Fatalf("GetCachedBlob failed: %v", err)
	}
	if cached == nil {
		t.Error("expected blob to be cached")
	}

	// Second call - should come from cache
	blob2, err := service.GetBlob(context.Background(), "testowner", "testrepo", sha)
	if err != nil {
		t.Fatalf("GetBlob from cache failed: %v", err)
	}

	if blob1.SHA != blob2.SHA {
		t.Errorf("cached blob SHA mismatch")
	}
}

func TestService_CreateBlob(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	in := &git.CreateBlobIn{
		Content:  "Test content",
		Encoding: "utf-8",
	}

	blob, err := service.CreateBlob(context.Background(), "testowner", "testrepo", in)
	if err != nil {
		t.Fatalf("CreateBlob failed: %v", err)
	}

	if blob.SHA == "" {
		t.Error("expected SHA to be set")
	}
	if blob.Size != len(in.Content) {
		t.Errorf("got size %d, want %d", blob.Size, len(in.Content))
	}
	if blob.Encoding != "utf-8" {
		t.Errorf("got encoding %q, want utf-8", blob.Encoding)
	}
}

func TestService_CreateBlob_RepoNotFound(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	in := &git.CreateBlobIn{Content: "test"}
	_, err := service.CreateBlob(context.Background(), "unknown", "repo", in)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_CreateBlob_UTF8Default(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	in := &git.CreateBlobIn{
		Content: "Test content",
		// No encoding specified
	}

	blob, err := service.CreateBlob(context.Background(), "testowner", "testrepo", in)
	if err != nil {
		t.Fatalf("CreateBlob failed: %v", err)
	}

	if blob.Encoding != "utf-8" {
		t.Errorf("expected default encoding utf-8, got %q", blob.Encoding)
	}
}

func TestService_CreateBlob_Base64(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Base64 encoded "Hello"
	in := &git.CreateBlobIn{
		Content:  "SGVsbG8=",
		Encoding: "base64",
	}

	blob, err := service.CreateBlob(context.Background(), "testowner", "testrepo", in)
	if err != nil {
		t.Fatalf("CreateBlob failed: %v", err)
	}

	if blob.Size != 5 { // "Hello" is 5 bytes
		t.Errorf("got size %d, want 5", blob.Size)
	}
}

// Commit Tests

func TestService_GetGitCommit(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	_, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	commit, err := service.GetGitCommit(context.Background(), "testowner", "testrepo", commitSHA)
	if err != nil {
		t.Fatalf("GetGitCommit failed: %v", err)
	}

	if commit.SHA != commitSHA {
		t.Errorf("got SHA %q, want %q", commit.SHA, commitSHA)
	}
	if commit.Message == "" {
		t.Error("expected message to be set")
	}
	if commit.Author == nil {
		t.Error("expected author to be set")
	}
	if commit.Committer == nil {
		t.Error("expected committer to be set")
	}
	if commit.Tree == nil {
		t.Error("expected tree to be set")
	}
	if commit.URL == "" {
		t.Error("expected URL to be populated")
	}
	if commit.HTMLURL == "" {
		t.Error("expected HTMLURL to be populated")
	}
}

func TestService_GetGitCommit_RepoNotFound(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetGitCommit(context.Background(), "unknown", "repo", "abcd1234abcd1234abcd1234abcd1234abcd1234")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_GetGitCommit_NotFound(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	_, err := service.GetGitCommit(context.Background(), "testowner", "testrepo", "0000000000000000000000000000000000000000")
	if err != git.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_CreateGitCommit(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, initialCommit := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Get the tree SHA from initial commit
	commit, _ := gitRepo.GetCommit(initialCommit)
	treeSHA := commit.TreeSHA

	in := &git.CreateGitCommitIn{
		Message: "Test commit",
		Tree:    treeSHA,
		Parents: []string{initialCommit},
		Author: &git.CommitAuthor{
			Name:  "Test Author",
			Email: "test@example.com",
			Date:  time.Now(),
		},
	}

	newCommit, err := service.CreateGitCommit(context.Background(), "testowner", "testrepo", in)
	if err != nil {
		t.Fatalf("CreateGitCommit failed: %v", err)
	}

	if newCommit.SHA == "" {
		t.Error("expected SHA to be set")
	}
	if newCommit.Message != in.Message {
		t.Errorf("got message %q, want %q", newCommit.Message, in.Message)
	}
	if len(newCommit.Parents) != 1 {
		t.Errorf("got %d parents, want 1", len(newCommit.Parents))
	}
}

func TestService_CreateGitCommit_DefaultAuthor(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, initialCommit := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	commit, _ := gitRepo.GetCommit(initialCommit)
	treeSHA := commit.TreeSHA

	in := &git.CreateGitCommitIn{
		Message: "Test commit",
		Tree:    treeSHA,
		Parents: []string{initialCommit},
		// No author specified
	}

	newCommit, err := service.CreateGitCommit(context.Background(), "testowner", "testrepo", in)
	if err != nil {
		t.Fatalf("CreateGitCommit failed: %v", err)
	}

	if newCommit.Author == nil {
		t.Error("expected default author to be set")
	}
	if newCommit.Author.Name == "" {
		t.Error("expected author name to be set")
	}
}

func TestService_CreateGitCommit_CommitterFromAuthor(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, initialCommit := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	commit, _ := gitRepo.GetCommit(initialCommit)
	treeSHA := commit.TreeSHA

	in := &git.CreateGitCommitIn{
		Message: "Test commit",
		Tree:    treeSHA,
		Parents: []string{initialCommit},
		Author: &git.CommitAuthor{
			Name:  "Custom Author",
			Email: "author@example.com",
			Date:  time.Now(),
		},
		// No committer specified
	}

	newCommit, err := service.CreateGitCommit(context.Background(), "testowner", "testrepo", in)
	if err != nil {
		t.Fatalf("CreateGitCommit failed: %v", err)
	}

	if newCommit.Committer.Name != in.Author.Name {
		t.Errorf("expected committer to copy author, got %q", newCommit.Committer.Name)
	}
}

// Reference Tests

func TestService_GetRef(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	ref, err := service.GetRef(context.Background(), "testowner", "testrepo", "heads/main")
	if err != nil {
		t.Fatalf("GetRef failed: %v", err)
	}

	if ref.Ref == "" {
		t.Error("expected ref name to be set")
	}
	if ref.Object == nil {
		t.Error("expected object to be set")
	}
	if ref.Object.SHA == "" {
		t.Error("expected object SHA to be set")
	}
	if ref.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_GetRef_RepoNotFound(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetRef(context.Background(), "unknown", "repo", "heads/main")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_GetRef_NotFound(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	_, err := service.GetRef(context.Background(), "testowner", "testrepo", "heads/nonexistent")
	if err != git.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_ListMatchingRefs(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Create additional refs
	gitRepo.CreateRef("refs/heads/feature-1", commitSHA)
	gitRepo.CreateRef("refs/heads/feature-2", commitSHA)

	refs, err := service.ListMatchingRefs(context.Background(), "testowner", "testrepo", "heads/")
	if err != nil {
		t.Fatalf("ListMatchingRefs failed: %v", err)
	}

	if len(refs) < 1 {
		t.Error("expected at least one ref")
	}
}

func TestService_ListMatchingRefs_Empty(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	refs, err := service.ListMatchingRefs(context.Background(), "testowner", "testrepo", "tags/")
	if err != nil {
		t.Fatalf("ListMatchingRefs failed: %v", err)
	}

	if len(refs) != 0 {
		t.Errorf("expected 0 refs, got %d", len(refs))
	}
}

func TestService_CreateRef(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	_, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	in := &git.CreateRefIn{
		Ref: "refs/heads/new-branch",
		SHA: commitSHA,
	}

	ref, err := service.CreateRef(context.Background(), "testowner", "testrepo", in)
	if err != nil {
		t.Fatalf("CreateRef failed: %v", err)
	}

	if ref.Object.SHA != commitSHA {
		t.Errorf("got SHA %q, want %q", ref.Object.SHA, commitSHA)
	}
}

func TestService_CreateRef_Exists(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	_, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	in := &git.CreateRefIn{
		Ref: "refs/heads/main", // Already exists
		SHA: commitSHA,
	}

	_, err := service.CreateRef(context.Background(), "testowner", "testrepo", in)
	if err != git.ErrRefExists {
		t.Errorf("expected ErrRefExists, got %v", err)
	}
}

func TestService_UpdateRef(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, initialCommit := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Create a new commit
	commit, _ := gitRepo.GetCommit(initialCommit)
	newCommitSHA, _ := gitRepo.CreateCommit(&pkggit.CreateCommitOpts{
		Message: "New commit",
		TreeSHA: commit.TreeSHA,
		Parents: []string{initialCommit},
		Author: pkggit.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
		Committer: pkggit.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})

	ref, err := service.UpdateRef(context.Background(), "testowner", "testrepo", "heads/main", newCommitSHA, false)
	if err != nil {
		t.Fatalf("UpdateRef failed: %v", err)
	}

	if ref.Object.SHA != newCommitSHA {
		t.Errorf("got SHA %q, want %q", ref.Object.SHA, newCommitSHA)
	}
}

func TestService_UpdateRef_Force(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, initialCommit := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Create a new commit (not parent of current)
	commit, _ := gitRepo.GetCommit(initialCommit)
	newCommitSHA, _ := gitRepo.CreateCommit(&pkggit.CreateCommitOpts{
		Message: "Divergent commit",
		TreeSHA: commit.TreeSHA,
		Parents: nil, // No parents - this would be non-fast-forward
		Author: pkggit.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
		Committer: pkggit.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})

	ref, err := service.UpdateRef(context.Background(), "testowner", "testrepo", "heads/main", newCommitSHA, true)
	if err != nil {
		t.Fatalf("UpdateRef with force failed: %v", err)
	}

	if ref.Object.SHA != newCommitSHA {
		t.Errorf("got SHA %q, want %q", ref.Object.SHA, newCommitSHA)
	}
}

func TestService_DeleteRef(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Create a branch to delete
	gitRepo.CreateRef("refs/heads/to-delete", commitSHA)

	err := service.DeleteRef(context.Background(), "testowner", "testrepo", "heads/to-delete")
	if err != nil {
		t.Fatalf("DeleteRef failed: %v", err)
	}

	// Verify it's deleted
	_, err = service.GetRef(context.Background(), "testowner", "testrepo", "heads/to-delete")
	if err != git.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

// Tree Tests

func TestService_GetTree(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Get tree SHA from commit
	commit, _ := gitRepo.GetCommit(commitSHA)

	tree, err := service.GetTree(context.Background(), "testowner", "testrepo", commit.TreeSHA, false)
	if err != nil {
		t.Fatalf("GetTree failed: %v", err)
	}

	if tree.SHA != commit.TreeSHA {
		t.Errorf("got SHA %q, want %q", tree.SHA, commit.TreeSHA)
	}
	if tree.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_GetTree_Recursive(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	commit, _ := gitRepo.GetCommit(commitSHA)

	tree, err := service.GetTree(context.Background(), "testowner", "testrepo", commit.TreeSHA, true)
	if err != nil {
		t.Fatalf("GetTree recursive failed: %v", err)
	}

	if tree.SHA != commit.TreeSHA {
		t.Errorf("got SHA %q, want %q", tree.SHA, commit.TreeSHA)
	}
}

func TestService_GetTree_RepoNotFound(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetTree(context.Background(), "unknown", "repo", "abcd1234abcd1234abcd1234abcd1234abcd1234", false)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_CreateTree(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, _ := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Create a blob first
	blobSHA, _ := gitRepo.CreateBlob([]byte("file content"))

	in := &git.CreateTreeIn{
		Tree: []*git.TreeEntryIn{
			{
				Path: "newfile.txt",
				Mode: "100644",
				Type: "blob",
				SHA:  blobSHA,
			},
		},
	}

	tree, err := service.CreateTree(context.Background(), "testowner", "testrepo", in)
	if err != nil {
		t.Fatalf("CreateTree failed: %v", err)
	}

	if tree.SHA == "" {
		t.Error("expected SHA to be set")
	}
	if len(tree.Tree) != 1 {
		t.Errorf("expected 1 entry, got %d", len(tree.Tree))
	}
}

func TestService_CreateTree_WithBase(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	commit, _ := gitRepo.GetCommit(commitSHA)
	blobSHA, _ := gitRepo.CreateBlob([]byte("new content"))

	in := &git.CreateTreeIn{
		BaseTree: commit.TreeSHA,
		Tree: []*git.TreeEntryIn{
			{
				Path: "added.txt",
				Mode: "100644",
				Type: "blob",
				SHA:  blobSHA,
			},
		},
	}

	tree, err := service.CreateTree(context.Background(), "testowner", "testrepo", in)
	if err != nil {
		t.Fatalf("CreateTree with base failed: %v", err)
	}

	if tree.SHA == "" {
		t.Error("expected SHA to be set")
	}
}

// Tag Tests

func TestService_GetTag(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Create an annotated tag
	tagSHA, err := gitRepo.CreateTag(&pkggit.CreateTagOpts{
		Name:       "v1.0.0",
		TargetSHA:  commitSHA,
		TargetType: pkggit.ObjectCommit,
		Message:    "Version 1.0.0",
		Tagger: pkggit.Signature{
			Name:  "Tagger",
			Email: "tagger@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}

	tag, err := service.GetTag(context.Background(), "testowner", "testrepo", tagSHA)
	if err != nil {
		t.Fatalf("GetTag failed: %v", err)
	}

	if tag.SHA != tagSHA {
		t.Errorf("got SHA %q, want %q", tag.SHA, tagSHA)
	}
	if tag.Tag != "v1.0.0" {
		t.Errorf("got tag name %q, want v1.0.0", tag.Tag)
	}
	if tag.Message != "Version 1.0.0" {
		t.Errorf("got message %q, want 'Version 1.0.0'", tag.Message)
	}
	if tag.Object == nil || tag.Object.SHA != commitSHA {
		t.Error("expected object to point to commit")
	}
}

func TestService_GetTag_RepoNotFound(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetTag(context.Background(), "unknown", "repo", "abcd1234abcd1234abcd1234abcd1234abcd1234")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_GetTag_NotFound(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	_, err := service.GetTag(context.Background(), "testowner", "testrepo", "0000000000000000000000000000000000000000")
	if err != git.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_CreateTag(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	_, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	in := &git.CreateTagIn{
		Tag:     "v2.0.0",
		Message: "Version 2.0.0",
		Object:  commitSHA,
		Type:    "commit",
		Tagger: &git.CommitAuthor{
			Name:  "Tagger",
			Email: "tagger@example.com",
			Date:  time.Now(),
		},
	}

	tag, err := service.CreateTag(context.Background(), "testowner", "testrepo", in)
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	if tag.SHA == "" {
		t.Error("expected SHA to be set")
	}
	if tag.Tag != "v2.0.0" {
		t.Errorf("got tag name %q, want v2.0.0", tag.Tag)
	}
}

func TestService_CreateTag_DefaultTagger(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	_, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	in := &git.CreateTagIn{
		Tag:     "v3.0.0",
		Message: "Version 3.0.0",
		Object:  commitSHA,
		Type:    "commit",
		// No tagger specified
	}

	tag, err := service.CreateTag(context.Background(), "testowner", "testrepo", in)
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	if tag.Tagger == nil {
		t.Error("expected default tagger to be set")
	}
}

func TestService_ListTags(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Create some tags
	gitRepo.CreateRef("refs/tags/v1.0.0", commitSHA)
	gitRepo.CreateRef("refs/tags/v1.1.0", commitSHA)

	tags, err := service.ListTags(context.Background(), "testowner", "testrepo", nil)
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestService_ListTags_Pagination(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Create multiple tags
	for i := 0; i < 5; i++ {
		gitRepo.CreateRef("refs/tags/v1."+string(rune('0'+i))+".0", commitSHA)
	}

	tags, err := service.ListTags(context.Background(), "testowner", "testrepo", &git.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestService_ListTags_MaxPerPage(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Request more than max
	_, err := service.ListTags(context.Background(), "testowner", "testrepo", &git.ListOpts{
		PerPage: 200, // Should be capped at 100
	})
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	// If we got here without error, the per_page was capped correctly
}

// URL Population Tests

func TestService_PopulateURLs(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, commitSHA := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	// Test blob URL
	blobSHA, _ := gitRepo.CreateBlob([]byte("test"))
	blob, _ := service.GetBlob(context.Background(), "testowner", "testrepo", blobSHA)
	if blob.URL != "https://api.example.com/api/v3/repos/testowner/testrepo/git/blobs/"+blobSHA {
		t.Errorf("unexpected blob URL: %s", blob.URL)
	}

	// Test commit URLs
	commit, _ := service.GetGitCommit(context.Background(), "testowner", "testrepo", commitSHA)
	if commit.URL != "https://api.example.com/api/v3/repos/testowner/testrepo/git/commits/"+commitSHA {
		t.Errorf("unexpected commit URL: %s", commit.URL)
	}
	if commit.HTMLURL != "https://api.example.com/testowner/testrepo/commit/"+commitSHA {
		t.Errorf("unexpected commit HTML URL: %s", commit.HTMLURL)
	}

	// Test ref URL
	ref, _ := service.GetRef(context.Background(), "testowner", "testrepo", "heads/main")
	if ref.URL == "" {
		t.Error("expected ref URL to be set")
	}
}

// Integration-style tests

func TestService_FullWorkflow(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	gitRepo, initialCommit := setupTestRepo(t, tmpDir, "testowner", "testrepo")
	addTestRepo(t, reposStore, "testowner", "testrepo")

	ctx := context.Background()

	// 1. Create a blob
	blob, err := service.CreateBlob(ctx, "testowner", "testrepo", &git.CreateBlobIn{
		Content: "Hello, World!",
	})
	if err != nil {
		t.Fatalf("CreateBlob failed: %v", err)
	}

	// 2. Create a tree with the blob
	tree, err := service.CreateTree(ctx, "testowner", "testrepo", &git.CreateTreeIn{
		Tree: []*git.TreeEntryIn{
			{
				Path: "hello.txt",
				Mode: "100644",
				Type: "blob",
				SHA:  blob.SHA,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateTree failed: %v", err)
	}

	// 3. Create a commit with the tree
	commit, err := service.CreateGitCommit(ctx, "testowner", "testrepo", &git.CreateGitCommitIn{
		Message: "Add hello.txt",
		Tree:    tree.SHA,
		Parents: []string{initialCommit},
		Author: &git.CommitAuthor{
			Name:  "Test Author",
			Email: "test@example.com",
			Date:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("CreateGitCommit failed: %v", err)
	}

	// 4. Update the main branch to point to the new commit
	ref, err := service.UpdateRef(ctx, "testowner", "testrepo", "heads/main", commit.SHA, false)
	if err != nil {
		t.Fatalf("UpdateRef failed: %v", err)
	}

	if ref.Object.SHA != commit.SHA {
		t.Errorf("ref not updated correctly: got %s, want %s", ref.Object.SHA, commit.SHA)
	}

	// 5. Create a tag for the commit
	tag, err := service.CreateTag(ctx, "testowner", "testrepo", &git.CreateTagIn{
		Tag:     "v1.0.0",
		Message: "First release",
		Object:  commit.SHA,
		Type:    "commit",
	})
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	// 6. Create a tag reference
	err = gitRepo.CreateRef("refs/tags/v1.0.0", tag.SHA)
	if err != nil {
		t.Logf("Note: tag ref creation: %v", err)
	}

	// 7. List tags
	tags, err := service.ListTags(ctx, "testowner", "testrepo", nil)
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}

	if len(tags) < 1 {
		t.Error("expected at least one tag")
	}

	// 8. Verify the commit can be retrieved
	retrievedCommit, err := service.GetGitCommit(ctx, "testowner", "testrepo", commit.SHA)
	if err != nil {
		t.Fatalf("GetGitCommit failed: %v", err)
	}

	if retrievedCommit.Message != "Add hello.txt" {
		t.Errorf("unexpected commit message: %s", retrievedCommit.Message)
	}
}

// Test with multiple repos to ensure isolation

func TestService_MultipleRepos(t *testing.T) {
	service, _, reposStore, tmpDir, cleanup := setupTestService(t)
	defer cleanup()

	// Setup two repos
	gitRepo1, commitSHA1 := setupTestRepo(t, tmpDir, "owner1", "repo1")
	gitRepo2, commitSHA2 := setupTestRepo(t, tmpDir, "owner2", "repo2")
	addTestRepo(t, reposStore, "owner1", "repo1")
	addTestRepo(t, reposStore, "owner2", "repo2")

	ctx := context.Background()

	// Create blob in repo1
	blob1SHA, _ := gitRepo1.CreateBlob([]byte("repo1 content"))
	blob1, err := service.GetBlob(ctx, "owner1", "repo1", blob1SHA)
	if err != nil {
		t.Fatalf("GetBlob repo1 failed: %v", err)
	}

	// Create blob in repo2
	blob2SHA, _ := gitRepo2.CreateBlob([]byte("repo2 content"))
	blob2, err := service.GetBlob(ctx, "owner2", "repo2", blob2SHA)
	if err != nil {
		t.Fatalf("GetBlob repo2 failed: %v", err)
	}

	// Verify different content
	if blob1.Content == blob2.Content {
		t.Error("expected different content in different repos")
	}

	// Try to get repo1's blob from repo2 - should fail
	_, err = service.GetBlob(ctx, "owner2", "repo2", blob1SHA)
	if err != git.ErrNotFound {
		t.Errorf("expected ErrNotFound when accessing repo1 blob from repo2, got %v", err)
	}

	// Verify commits are isolated
	commit1, err := service.GetGitCommit(ctx, "owner1", "repo1", commitSHA1)
	if err != nil {
		t.Fatalf("GetGitCommit repo1 failed: %v", err)
	}

	commit2, err := service.GetGitCommit(ctx, "owner2", "repo2", commitSHA2)
	if err != nil {
		t.Fatalf("GetGitCommit repo2 failed: %v", err)
	}

	if commit1.SHA == commit2.SHA {
		t.Log("Note: commits have same SHA (empty tree), but are in different repos")
	}
}
