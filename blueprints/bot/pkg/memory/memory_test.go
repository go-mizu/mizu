package memory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestManager(t *testing.T, workspaceDir string) *MemoryManager {
	t.Helper()
	t.Setenv("OPENAI_API_KEY", "") // force FTS-only mode

	dbPath := filepath.Join(t.TempDir(), "mem.db")
	cfg := DefaultMemoryConfig()
	cfg.WorkspaceDir = workspaceDir

	mgr, err := NewMemoryManager(dbPath, workspaceDir, cfg)
	if err != nil {
		t.Fatalf("NewMemoryManager: %v", err)
	}
	t.Cleanup(func() { mgr.Close() })
	return mgr
}

func TestNewMemoryManager_FTSOnlyMode(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	dir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	mgr, err := NewMemoryManager(dbPath, dir, DefaultMemoryConfig())
	if err != nil {
		t.Fatalf("NewMemoryManager: %v", err)
	}
	defer mgr.Close()

	if mgr.embedder != nil {
		t.Error("expected nil embedder in FTS-only mode (no OPENAI_API_KEY)")
	}
	if mgr.store == nil {
		t.Error("expected non-nil store")
	}
}

func TestDefaultMemoryConfig(t *testing.T) {
	cfg := DefaultMemoryConfig()
	if cfg.ChunkTokens != 400 {
		t.Errorf("ChunkTokens = %d, want 400", cfg.ChunkTokens)
	}
	if cfg.ChunkOverlap != 80 {
		t.Errorf("ChunkOverlap = %d, want 80", cfg.ChunkOverlap)
	}
	if cfg.VectorWeight != 0.7 {
		t.Errorf("VectorWeight = %f, want 0.7", cfg.VectorWeight)
	}
	if cfg.TextWeight != 0.3 {
		t.Errorf("TextWeight = %f, want 0.3", cfg.TextWeight)
	}
	if cfg.MinScore != 0.35 {
		t.Errorf("MinScore = %f, want 0.35", cfg.MinScore)
	}
	if cfg.MaxResults != 6 {
		t.Errorf("MaxResults = %d, want 6", cfg.MaxResults)
	}
}

func TestIndexFile_CreatesChunks(t *testing.T) {
	dir := t.TempDir()
	// Create a file with known content.
	content := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mgr := newTestManager(t, dir)

	if err := mgr.IndexFile("main.go"); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	// Verify chunks exist in the store.
	var count int
	err := mgr.store.db.QueryRow(`SELECT count(*) FROM chunks WHERE path = ?`, "main.go").Scan(&count)
	if err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if count == 0 {
		t.Error("expected at least 1 chunk after IndexFile")
	}

	// Verify file record exists.
	hash, err := mgr.store.GetFileHash("main.go")
	if err != nil {
		t.Fatalf("GetFileHash: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty file hash after IndexFile")
	}
}

func TestIndexFile_SkipsUnchanged(t *testing.T) {
	dir := t.TempDir()
	content := "package foo\n\nvar x = 1\n"
	filePath := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mgr := newTestManager(t, dir)

	// Index the first time.
	if err := mgr.IndexFile("foo.go"); err != nil {
		t.Fatalf("IndexFile first: %v", err)
	}

	// Get the current hash.
	hash1, err := mgr.store.GetFileHash("foo.go")
	if err != nil {
		t.Fatalf("GetFileHash: %v", err)
	}

	// Index again without changing the file -- should be a no-op.
	if err := mgr.IndexFile("foo.go"); err != nil {
		t.Fatalf("IndexFile second: %v", err)
	}

	hash2, err := mgr.store.GetFileHash("foo.go")
	if err != nil {
		t.Fatalf("GetFileHash second: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("hash changed on re-index of unchanged file: %q vs %q", hash1, hash2)
	}
}

func TestIndexAll(t *testing.T) {
	dir := t.TempDir()

	// Create indexable files.
	files := map[string]string{
		"readme.md": "# Hello\n\nThis is a readme.",
		"main.go":   "package main\n\nfunc main() {}\n",
		"utils.py":  "def hello():\n    print('hi')\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	mgr := newTestManager(t, dir)
	if err := mgr.IndexAll(); err != nil {
		t.Fatalf("IndexAll: %v", err)
	}

	// All three files should be indexed.
	for name := range files {
		hash, err := mgr.store.GetFileHash(name)
		if err != nil {
			t.Fatalf("GetFileHash(%s): %v", name, err)
		}
		if hash == "" {
			t.Errorf("file %q was not indexed", name)
		}
	}
}

func TestIndexAll_SkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()

	// Create a hidden directory with a file.
	hiddenDir := filepath.Join(dir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "secret.go"), []byte("package secret\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create node_modules.
	nmDir := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(nmDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nmDir, "pkg.js"), []byte("module.exports = {}\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create vendor.
	vendorDir := filepath.Join(dir, "vendor")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "lib.go"), []byte("package lib\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create build and dist.
	for _, d := range []string{"build", "dist", "__pycache__"} {
		p := filepath.Join(dir, d)
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatalf("MkdirAll %s: %v", d, err)
		}
		if err := os.WriteFile(filepath.Join(p, "out.js"), []byte("output\n"), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	// Create a normal file that should be indexed.
	if err := os.WriteFile(filepath.Join(dir, "app.go"), []byte("package app\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mgr := newTestManager(t, dir)
	if err := mgr.IndexAll(); err != nil {
		t.Fatalf("IndexAll: %v", err)
	}

	// app.go should be indexed.
	hash, err := mgr.store.GetFileHash("app.go")
	if err != nil {
		t.Fatalf("GetFileHash(app.go): %v", err)
	}
	if hash == "" {
		t.Error("app.go was not indexed")
	}

	// Files in skipped directories should NOT be indexed.
	skipped := []string{
		filepath.Join(".hidden", "secret.go"),
		filepath.Join("node_modules", "pkg.js"),
		filepath.Join("vendor", "lib.go"),
		filepath.Join("build", "out.js"),
		filepath.Join("dist", "out.js"),
		filepath.Join("__pycache__", "out.js"),
	}
	for _, f := range skipped {
		hash, err := mgr.store.GetFileHash(f)
		if err != nil {
			t.Fatalf("GetFileHash(%s): %v", f, err)
		}
		if hash != "" {
			t.Errorf("file %q in skipped dir was indexed (hash=%q)", f, hash)
		}
	}
}

func TestIndexAll_SkipsNonIndexableFiles(t *testing.T) {
	dir := t.TempDir()

	// Create non-indexable files.
	nonIndexable := []string{"photo.jpg", "image.png", "program.exe", "archive.zip"}
	for _, name := range nonIndexable {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("binary data"), 0644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	// Create one indexable file.
	if err := os.WriteFile(filepath.Join(dir, "code.ts"), []byte("const x = 1;\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mgr := newTestManager(t, dir)
	if err := mgr.IndexAll(); err != nil {
		t.Fatalf("IndexAll: %v", err)
	}

	// code.ts should be indexed.
	hash, err := mgr.store.GetFileHash("code.ts")
	if err != nil {
		t.Fatalf("GetFileHash(code.ts): %v", err)
	}
	if hash == "" {
		t.Error("code.ts was not indexed")
	}

	// Non-indexable files should not be indexed.
	for _, name := range nonIndexable {
		hash, err := mgr.store.GetFileHash(name)
		if err != nil {
			t.Fatalf("GetFileHash(%s): %v", name, err)
		}
		if hash != "" {
			t.Errorf("non-indexable file %q was indexed", name)
		}
	}
}

func TestIndexAll_SkipsLargeFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a file larger than 1MB.
	largeContent := strings.Repeat("x", 1<<20+1) // 1MB + 1 byte
	if err := os.WriteFile(filepath.Join(dir, "large.go"), []byte(largeContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create a normal-size file.
	if err := os.WriteFile(filepath.Join(dir, "small.go"), []byte("package small\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mgr := newTestManager(t, dir)
	if err := mgr.IndexAll(); err != nil {
		t.Fatalf("IndexAll: %v", err)
	}

	// small.go should be indexed.
	hash, err := mgr.store.GetFileHash("small.go")
	if err != nil {
		t.Fatalf("GetFileHash(small.go): %v", err)
	}
	if hash == "" {
		t.Error("small.go was not indexed")
	}

	// large.go should NOT be indexed.
	hash, err = mgr.store.GetFileHash("large.go")
	if err != nil {
		t.Fatalf("GetFileHash(large.go): %v", err)
	}
	if hash != "" {
		t.Errorf("large.go (> 1MB) was indexed, should have been skipped")
	}
}

func TestSearch_FTSOnly(t *testing.T) {
	dir := t.TempDir()

	// Create files with distinct content.
	if err := os.WriteFile(filepath.Join(dir, "alpha.md"), []byte("The quick brown fox jumps over the lazy dog.\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "beta.md"), []byte("Rust is a systems programming language focused on safety.\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mgr := newTestManager(t, dir)
	if err := mgr.IndexAll(); err != nil {
		t.Fatalf("IndexAll: %v", err)
	}

	// Search for "fox" should find alpha.md.
	results, err := mgr.Search(context.Background(), "fox", 10, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result for 'fox'")
	}
	found := false
	for _, r := range results {
		if r.Path == "alpha.md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected alpha.md in search results for 'fox'")
	}

	// Search for "rust" should find beta.md.
	results, err = mgr.Search(context.Background(), "rust", 10, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result for 'rust'")
	}
	found = false
	for _, r := range results {
		if r.Path == "beta.md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected beta.md in search results for 'rust'")
	}

	// Search for something absent should return empty.
	results, err = mgr.Search(context.Background(), "quantum", 10, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for 'quantum', got %d", len(results))
	}
}

func TestGetLines(t *testing.T) {
	dir := t.TempDir()
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\n"
	filePath := filepath.Join(dir, "lines.txt")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mgr := newTestManager(t, dir)

	// Read lines 3-5 (1-based from=3, count=3).
	got, err := mgr.GetLines("lines.txt", 3, 3)
	if err != nil {
		t.Fatalf("GetLines: %v", err)
	}
	want := "line3\nline4\nline5"
	if got != want {
		t.Errorf("GetLines(3,3) = %q, want %q", got, want)
	}

	// Read from the beginning.
	got, err = mgr.GetLines("lines.txt", 1, 2)
	if err != nil {
		t.Fatalf("GetLines(1,2): %v", err)
	}
	want = "line1\nline2"
	if got != want {
		t.Errorf("GetLines(1,2) = %q, want %q", got, want)
	}
}

func TestIsIndexableFile(t *testing.T) {
	indexable := []string{
		"readme.md", "main.go", "script.py", "app.js", "index.ts",
		"query.sql", "data.json", "style.css", "page.html",
		"config.yaml", "Makefile", "Dockerfile",
	}
	for _, name := range indexable {
		if !isIndexableFile(name) {
			t.Errorf("isIndexableFile(%q) = false, want true", name)
		}
	}

	notIndexable := []string{
		"photo.jpg", "image.png", "program.exe", "archive.zip",
		"video.mp4", "sound.mp3", "font.ttf", "data.bin",
	}
	for _, name := range notIndexable {
		if isIndexableFile(name) {
			t.Errorf("isIndexableFile(%q) = true, want false", name)
		}
	}
}

func TestShouldSkipDir(t *testing.T) {
	skip := []string{".git", "node_modules", "vendor", "build", "dist", "__pycache__", ".svn", ".hg"}
	for _, name := range skip {
		if !shouldSkipDir(name) {
			t.Errorf("shouldSkipDir(%q) = false, want true", name)
		}
	}

	noSkip := []string{"src", "pkg", "cmd", "internal", "lib", "tests"}
	for _, name := range noSkip {
		if shouldSkipDir(name) {
			t.Errorf("shouldSkipDir(%q) = true, want false", name)
		}
	}
}
