package git

import (
	"os"
	"testing"
	"time"
)

func setupTestRepo(t *testing.T) (*Repository, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	author := Signature{
		Name:  "Test Author",
		Email: "test@example.com",
		When:  time.Now(),
	}

	repo, _, err := InitWithCommit(tmpDir, author, "Initial commit")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init repo: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return repo, tmpDir, cleanup
}

// Blob Tests

func TestRepository_GetBlob(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	content := []byte("Hello, World!")
	sha, err := repo.CreateBlob(content)
	if err != nil {
		t.Fatalf("CreateBlob failed: %v", err)
	}

	blob, err := repo.GetBlob(sha)
	if err != nil {
		t.Fatalf("GetBlob failed: %v", err)
	}

	if blob.SHA != sha {
		t.Errorf("got SHA %q, want %q", blob.SHA, sha)
	}
	if string(blob.Content) != string(content) {
		t.Errorf("got content %q, want %q", string(blob.Content), string(content))
	}
	if blob.Size != int64(len(content)) {
		t.Errorf("got size %d, want %d", blob.Size, len(content))
	}
}

func TestRepository_GetBlob_NotFound(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	_, err := repo.GetBlob("0000000000000000000000000000000000000000")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRepository_GetBlob_InvalidSHA(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	_, err := repo.GetBlob("invalid")
	if err != ErrInvalidSHA {
		t.Errorf("expected ErrInvalidSHA, got %v", err)
	}
}

func TestRepository_CreateBlob(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	sha, err := repo.CreateBlob([]byte("test content"))
	if err != nil {
		t.Fatalf("CreateBlob failed: %v", err)
	}

	if len(sha) != 40 {
		t.Errorf("expected 40-char SHA, got %d chars", len(sha))
	}
}

func TestRepository_CreateBlob_Empty(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	sha, err := repo.CreateBlob([]byte{})
	if err != nil {
		t.Fatalf("CreateBlob with empty content failed: %v", err)
	}

	if sha == "" {
		t.Error("expected non-empty SHA")
	}

	blob, err := repo.GetBlob(sha)
	if err != nil {
		t.Fatalf("GetBlob failed: %v", err)
	}
	if blob.Size != 0 {
		t.Errorf("expected size 0, got %d", blob.Size)
	}
}

func TestRepository_CreateBlob_Binary(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	content := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	sha, err := repo.CreateBlob(content)
	if err != nil {
		t.Fatalf("CreateBlob with binary content failed: %v", err)
	}

	blob, err := repo.GetBlob(sha)
	if err != nil {
		t.Fatalf("GetBlob failed: %v", err)
	}

	if string(blob.Content) != string(content) {
		t.Errorf("binary content mismatch")
	}
}

// Commit Tests

func TestRepository_GetCommit(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head failed: %v", err)
	}

	commit, err := repo.GetCommit(head.SHA)
	if err != nil {
		t.Fatalf("GetCommit failed: %v", err)
	}

	if commit.SHA != head.SHA {
		t.Errorf("got SHA %q, want %q", commit.SHA, head.SHA)
	}
	if commit.Message == "" {
		t.Error("expected non-empty message")
	}
	if commit.TreeSHA == "" {
		t.Error("expected non-empty tree SHA")
	}
}

func TestRepository_GetCommit_NotFound(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	_, err := repo.GetCommit("0000000000000000000000000000000000000000")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRepository_CreateCommit(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, _ := repo.Head()
	commit, _ := repo.GetCommit(head.SHA)

	sha, err := repo.CreateCommit(&CreateCommitOpts{
		Message: "New commit",
		TreeSHA: commit.TreeSHA,
		Parents: []string{head.SHA},
		Author: Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
		Committer: Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("CreateCommit failed: %v", err)
	}

	if len(sha) != 40 {
		t.Errorf("expected 40-char SHA, got %d chars", len(sha))
	}

	newCommit, err := repo.GetCommit(sha)
	if err != nil {
		t.Fatalf("GetCommit failed: %v", err)
	}
	if newCommit.Message != "New commit" {
		t.Errorf("got message %q, want 'New commit'", newCommit.Message)
	}
	if len(newCommit.Parents) != 1 {
		t.Errorf("got %d parents, want 1", len(newCommit.Parents))
	}
}

func TestRepository_CreateCommit_NoParent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-test-orphan-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	treeSHA, err := repo.CreateTree(&CreateTreeOpts{})
	if err != nil {
		t.Fatalf("CreateTree failed: %v", err)
	}

	sha, err := repo.CreateCommit(&CreateCommitOpts{
		Message: "Initial commit",
		TreeSHA: treeSHA,
		Parents: nil,
		Author: Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
		Committer: Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("CreateCommit failed: %v", err)
	}

	commit, _ := repo.GetCommit(sha)
	if len(commit.Parents) != 0 {
		t.Errorf("expected 0 parents, got %d", len(commit.Parents))
	}
}

// Tree Tests

func TestRepository_GetTree(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, _ := repo.Head()
	commit, _ := repo.GetCommit(head.SHA)

	tree, err := repo.GetTree(commit.TreeSHA)
	if err != nil {
		t.Fatalf("GetTree failed: %v", err)
	}

	if tree.SHA != commit.TreeSHA {
		t.Errorf("got SHA %q, want %q", tree.SHA, commit.TreeSHA)
	}
}

func TestRepository_GetTree_NotFound(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	_, err := repo.GetTree("0000000000000000000000000000000000000000")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRepository_CreateTree(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	blobSHA, _ := repo.CreateBlob([]byte("content"))

	sha, err := repo.CreateTree(&CreateTreeOpts{
		Entries: []TreeEntryInput{
			{
				Path: "file.txt",
				Mode: ModeFile,
				Type: ObjectBlob,
				SHA:  blobSHA,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateTree failed: %v", err)
	}

	tree, _ := repo.GetTree(sha)
	if len(tree.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(tree.Entries))
	}
}

// Reference Tests

func TestRepository_GetRef(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ref, err := repo.GetRef("refs/heads/main")
	if err != nil {
		t.Fatalf("GetRef failed: %v", err)
	}

	if ref.Name != "refs/heads/main" {
		t.Errorf("got name %q, want 'refs/heads/main'", ref.Name)
	}
	if ref.SHA == "" {
		t.Error("expected non-empty SHA")
	}
}

func TestRepository_GetRef_NotFound(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	_, err := repo.GetRef("refs/heads/nonexistent")
	if err != ErrRefNotFound {
		t.Errorf("expected ErrRefNotFound, got %v", err)
	}
}

func TestRepository_CreateRef(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, _ := repo.Head()

	err := repo.CreateRef("refs/heads/new-branch", head.SHA)
	if err != nil {
		t.Fatalf("CreateRef failed: %v", err)
	}

	ref, err := repo.GetRef("refs/heads/new-branch")
	if err != nil {
		t.Fatalf("GetRef failed: %v", err)
	}
	if ref.SHA != head.SHA {
		t.Errorf("got SHA %q, want %q", ref.SHA, head.SHA)
	}
}

func TestRepository_CreateRef_Exists(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, _ := repo.Head()

	err := repo.CreateRef("refs/heads/main", head.SHA)
	if err != ErrRefExists {
		t.Errorf("expected ErrRefExists, got %v", err)
	}
}

func TestRepository_UpdateRef(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, _ := repo.Head()
	commit, _ := repo.GetCommit(head.SHA)

	newSHA, _ := repo.CreateCommit(&CreateCommitOpts{
		Message: "New",
		TreeSHA: commit.TreeSHA,
		Parents: []string{head.SHA},
		Author:  Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
		Committer: Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	err := repo.UpdateRef("refs/heads/main", newSHA, false)
	if err != nil {
		t.Fatalf("UpdateRef failed: %v", err)
	}

	ref, _ := repo.GetRef("refs/heads/main")
	if ref.SHA != newSHA {
		t.Errorf("got SHA %q, want %q", ref.SHA, newSHA)
	}
}

func TestRepository_DeleteRef(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, _ := repo.Head()
	repo.CreateRef("refs/heads/to-delete", head.SHA)

	err := repo.DeleteRef("refs/heads/to-delete")
	if err != nil {
		t.Fatalf("DeleteRef failed: %v", err)
	}

	_, err = repo.GetRef("refs/heads/to-delete")
	if err != ErrRefNotFound {
		t.Errorf("expected ErrRefNotFound, got %v", err)
	}
}

func TestRepository_DeleteRef_NotFound(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	err := repo.DeleteRef("refs/heads/nonexistent")
	if err != ErrRefNotFound {
		t.Errorf("expected ErrRefNotFound, got %v", err)
	}
}

// Tag Tests

func TestRepository_CreateTag(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, _ := repo.Head()

	sha, err := repo.CreateTag(&CreateTagOpts{
		Name:       "v1.0.0",
		TargetSHA:  head.SHA,
		TargetType: ObjectCommit,
		Message:    "Version 1.0.0",
		Tagger: Signature{
			Name:  "Tagger",
			Email: "tagger@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	tag, err := repo.GetTag(sha)
	if err != nil {
		t.Fatalf("GetTag failed: %v", err)
	}

	if tag.Name != "v1.0.0" {
		t.Errorf("got name %q, want 'v1.0.0'", tag.Name)
	}
	if tag.TargetSHA != head.SHA {
		t.Errorf("got target %q, want %q", tag.TargetSHA, head.SHA)
	}
}

func TestRepository_GetTag_NotFound(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	_, err := repo.GetTag("0000000000000000000000000000000000000000")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRepository_ListTags(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, _ := repo.Head()
	repo.CreateRef("refs/tags/v1.0.0", head.SHA)
	repo.CreateRef("refs/tags/v1.1.0", head.SHA)

	tags, err := repo.ListTags()
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestRepository_ListTags_Empty(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	tags, err := repo.ListTags()
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}

	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(tags))
	}
}

// Utility Tests

func TestRepository_ObjectExists(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, _ := repo.Head()

	if !repo.ObjectExists(head.SHA) {
		t.Error("expected object to exist")
	}

	if repo.ObjectExists("0000000000000000000000000000000000000000") {
		t.Error("expected object to not exist")
	}
}

func TestRepository_GetObjectType(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, _ := repo.Head()
	commit, _ := repo.GetCommit(head.SHA)

	// Test commit type
	objType, err := repo.GetObjectType(head.SHA)
	if err != nil {
		t.Fatalf("GetObjectType failed: %v", err)
	}
	if objType != ObjectCommit {
		t.Errorf("expected ObjectCommit, got %s", objType)
	}

	// Test tree type
	objType, err = repo.GetObjectType(commit.TreeSHA)
	if err != nil {
		t.Fatalf("GetObjectType failed: %v", err)
	}
	if objType != ObjectTree {
		t.Errorf("expected ObjectTree, got %s", objType)
	}

	// Test blob type
	blobSHA, _ := repo.CreateBlob([]byte("test"))
	objType, err = repo.GetObjectType(blobSHA)
	if err != nil {
		t.Fatalf("GetObjectType failed: %v", err)
	}
	if objType != ObjectBlob {
		t.Errorf("expected ObjectBlob, got %s", objType)
	}
}

func TestRepository_Head(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head failed: %v", err)
	}

	if head.SHA == "" {
		t.Error("expected non-empty SHA")
	}
	if head.Name != "refs/heads/main" {
		t.Errorf("got name %q, want 'refs/heads/main'", head.Name)
	}
}

func TestRepository_Log(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, _ := repo.Head()
	commit, _ := repo.GetCommit(head.SHA)

	// Create a few more commits
	for i := 0; i < 3; i++ {
		sha, _ := repo.CreateCommit(&CreateCommitOpts{
			Message: "Commit " + string(rune('1'+i)),
			TreeSHA: commit.TreeSHA,
			Parents: []string{head.SHA},
			Author:  Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
			Committer: Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
		})
		repo.UpdateRef("refs/heads/main", sha, true)
		head, _ = repo.Head()
	}

	commits, err := repo.Log("refs/heads/main", 10)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	if len(commits) != 4 { // 3 new + 1 initial
		t.Errorf("expected 4 commits, got %d", len(commits))
	}
}

func TestRepository_Log_Limit(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	head, _ := repo.Head()
	commit, _ := repo.GetCommit(head.SHA)

	// Create more commits
	for i := 0; i < 5; i++ {
		sha, _ := repo.CreateCommit(&CreateCommitOpts{
			Message: "Commit",
			TreeSHA: commit.TreeSHA,
			Parents: []string{head.SHA},
			Author:  Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
			Committer: Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
		})
		repo.UpdateRef("refs/heads/main", sha, true)
		head, _ = repo.Head()
	}

	commits, err := repo.Log("refs/heads/main", 3)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	if len(commits) != 3 {
		t.Errorf("expected 3 commits (limited), got %d", len(commits))
	}
}

func TestValidateSHA(t *testing.T) {
	tests := []struct {
		sha   string
		valid bool
	}{
		{"abcd1234abcd1234abcd1234abcd1234abcd1234", true},
		{"0000000000000000000000000000000000000000", true},
		{"ABCD1234ABCD1234ABCD1234ABCD1234ABCD1234", true},
		{"short", false},
		{"invalid", false},
		{"", false},
		{"zzzz1234abcd1234abcd1234abcd1234abcd1234", false}, // non-hex
	}

	for _, tt := range tests {
		err := validateSHA(tt.sha)
		if tt.valid && err != nil {
			t.Errorf("validateSHA(%q) should be valid, got error: %v", tt.sha, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("validateSHA(%q) should be invalid, got nil", tt.sha)
		}
	}
}

func TestFileMode_String(t *testing.T) {
	tests := []struct {
		mode FileMode
		want string
	}{
		{ModeFile, "100644"},
		{ModeExecutable, "100755"},
		{ModeSymlink, "120000"},
		{ModeSubmodule, "160000"},
		{ModeDir, "040000"},
	}

	for _, tt := range tests {
		got := tt.mode.String()
		if got != tt.want {
			t.Errorf("FileMode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestParseFileMode(t *testing.T) {
	tests := []struct {
		s    string
		want FileMode
	}{
		{"100644", ModeFile},
		{"100755", ModeExecutable},
		{"120000", ModeSymlink},
		{"160000", ModeSubmodule},
		{"040000", ModeDir},
		{"unknown", ModeFile}, // defaults to file
	}

	for _, tt := range tests {
		got := ParseFileMode(tt.s)
		if got != tt.want {
			t.Errorf("ParseFileMode(%q) = %d, want %d", tt.s, got, tt.want)
		}
	}
}
