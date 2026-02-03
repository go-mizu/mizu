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

// --- New tests for memory/sessions feature parity ---

func TestIndexFileWithSource_UsesMemorySource(t *testing.T) {
	dir := t.TempDir()
	content := "# My Memory\n\nSome important facts.\n"
	filePath := filepath.Join(dir, "MEMORY.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mgr := newTestManager(t, dir)

	if err := mgr.IndexFileWithSource("MEMORY.md", "memory"); err != nil {
		t.Fatalf("IndexFileWithSource: %v", err)
	}

	// Verify chunks have source="memory".
	var source string
	err := mgr.store.db.QueryRow(`SELECT source FROM chunks WHERE path = ? LIMIT 1`, "MEMORY.md").Scan(&source)
	if err != nil {
		t.Fatalf("query chunk source: %v", err)
	}
	if source != "memory" {
		t.Errorf("chunk source = %q, want %q", source, "memory")
	}

	// Verify file record has source="memory".
	err = mgr.store.db.QueryRow(`SELECT source FROM files WHERE path = ?`, "MEMORY.md").Scan(&source)
	if err != nil {
		t.Fatalf("query file source: %v", err)
	}
	if source != "memory" {
		t.Errorf("file source = %q, want %q", source, "memory")
	}
}

func TestIndexFile_DefaultsToMemorySource(t *testing.T) {
	dir := t.TempDir()
	content := "package main\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mgr := newTestManager(t, dir)
	if err := mgr.IndexFile("main.go"); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	// IndexFile should now default to "memory" source.
	var source string
	err := mgr.store.db.QueryRow(`SELECT source FROM chunks WHERE path = ? LIMIT 1`, "main.go").Scan(&source)
	if err != nil {
		t.Fatalf("query chunk source: %v", err)
	}
	if source != "memory" {
		t.Errorf("IndexFile default source = %q, want %q", source, "memory")
	}
}

func TestIndexAll_UsesMemorySource(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# Readme\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mgr := newTestManager(t, dir)
	if err := mgr.IndexAll(); err != nil {
		t.Fatalf("IndexAll: %v", err)
	}

	var source string
	err := mgr.store.db.QueryRow(`SELECT source FROM files WHERE path = ?`, "readme.md").Scan(&source)
	if err != nil {
		t.Fatalf("query file source: %v", err)
	}
	if source != "memory" {
		t.Errorf("IndexAll source = %q, want %q", source, "memory")
	}
}

func TestIndexSessionTranscript_ExtractsAssistantMessages(t *testing.T) {
	dir := t.TempDir()

	// Create a JSONL transcript with mixed roles.
	lines := []string{
		`{"role":"user","content":"Hello, how are you?"}`,
		`{"role":"assistant","content":"I am doing well, thank you for asking!"}`,
		`{"role":"user","content":"Tell me about Go programming."}`,
		`{"role":"assistant","content":"Go is a statically typed language designed at Google. It is known for its simplicity and concurrency support."}`,
		`{"role":"system","content":"You are a helpful assistant."}`,
	}
	transcriptContent := strings.Join(lines, "\n") + "\n"

	sessDir := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	transcriptPath := filepath.Join(sessDir, "abc123.jsonl")
	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mgr := newTestManager(t, dir)

	relPath := filepath.Join("sessions", "abc123.jsonl")
	if err := mgr.IndexSessionTranscript(relPath); err != nil {
		t.Fatalf("IndexSessionTranscript: %v", err)
	}

	// Verify chunks exist with source="sessions".
	var count int
	err := mgr.store.db.QueryRow(`SELECT COUNT(*) FROM chunks WHERE path = ? AND source = ?`,
		relPath, "sessions").Scan(&count)
	if err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if count == 0 {
		t.Error("expected at least 1 chunk from session transcript")
	}

	// Verify chunk text contains assistant content, not user/system.
	var text string
	err = mgr.store.db.QueryRow(`SELECT text FROM chunks WHERE path = ? LIMIT 1`, relPath).Scan(&text)
	if err != nil {
		t.Fatalf("query chunk text: %v", err)
	}
	if !strings.Contains(text, "doing well") {
		t.Error("expected assistant message content in chunks")
	}
	if strings.Contains(text, "Hello, how are you") {
		t.Error("user messages should not be in session transcript chunks")
	}

	// Verify file record.
	var source string
	err = mgr.store.db.QueryRow(`SELECT source FROM files WHERE path = ?`, relPath).Scan(&source)
	if err != nil {
		t.Fatalf("query file source: %v", err)
	}
	if source != "sessions" {
		t.Errorf("file source = %q, want %q", source, "sessions")
	}
}

func TestIndexSessionTranscript_SkipsUnchanged(t *testing.T) {
	dir := t.TempDir()
	content := `{"role":"assistant","content":"Hello world"}` + "\n"
	sessDir := filepath.Join(dir, "sessions")
	os.MkdirAll(sessDir, 0755)
	fpath := filepath.Join(sessDir, "test.jsonl")
	os.WriteFile(fpath, []byte(content), 0644)

	mgr := newTestManager(t, dir)
	relPath := filepath.Join("sessions", "test.jsonl")

	// Index first time.
	if err := mgr.IndexSessionTranscript(relPath); err != nil {
		t.Fatalf("first index: %v", err)
	}
	hash1, _ := mgr.store.GetFileHash(relPath)

	// Index again without changes.
	if err := mgr.IndexSessionTranscript(relPath); err != nil {
		t.Fatalf("second index: %v", err)
	}
	hash2, _ := mgr.store.GetFileHash(relPath)

	if hash1 != hash2 {
		t.Error("hash should not change on re-index of unchanged transcript")
	}
}

func TestIndexSessionTranscripts_WalksDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create a sessions directory with multiple JSONL files.
	sessDir := filepath.Join(dir, "agents", "main", "sessions")
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	for _, name := range []string{"sess1.jsonl", "sess2.jsonl"} {
		content := `{"role":"assistant","content":"Reply from ` + name + `"}` + "\n"
		if err := os.WriteFile(filepath.Join(sessDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	// Also create a non-JSONL file that should be skipped.
	os.WriteFile(filepath.Join(sessDir, "sessions.json"), []byte("{}"), 0644)

	mgr := newTestManager(t, dir)

	if err := mgr.IndexSessionTranscripts(filepath.Join("agents", "main", "sessions")); err != nil {
		t.Fatalf("IndexSessionTranscripts: %v", err)
	}

	// Both JSONL files should be indexed.
	for _, name := range []string{"sess1.jsonl", "sess2.jsonl"} {
		relPath := filepath.Join("agents", "main", "sessions", name)
		hash, err := mgr.store.GetFileHash(relPath)
		if err != nil {
			t.Fatalf("GetFileHash(%s): %v", relPath, err)
		}
		if hash == "" {
			t.Errorf("transcript %s was not indexed", name)
		}
	}

	// sessions.json should NOT be indexed (not .jsonl).
	jsonPath := filepath.Join("agents", "main", "sessions", "sessions.json")
	hash, _ := mgr.store.GetFileHash(jsonPath)
	if hash != "" {
		t.Error("sessions.json should not be indexed by IndexSessionTranscripts")
	}
}

func TestEnsureDailyLog_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	mgr := newTestManager(t, dir)

	if err := mgr.EnsureDailyLog(); err != nil {
		t.Fatalf("EnsureDailyLog: %v", err)
	}

	// Check that the memory directory and today's log file exist.
	memDir := filepath.Join(dir, "memory")
	entries, err := os.ReadDir(memDir)
	if err != nil {
		t.Fatalf("read memory dir: %v", err)
	}

	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			found = true
			// Verify the file has a header.
			content, err := os.ReadFile(filepath.Join(memDir, e.Name()))
			if err != nil {
				t.Fatalf("read log: %v", err)
			}
			if !strings.HasPrefix(string(content), "# ") {
				t.Error("daily log should start with a date header")
			}
			break
		}
	}
	if !found {
		t.Error("EnsureDailyLog did not create a .md file in workspace/memory/")
	}
}

func TestEnsureDailyLog_DoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	mgr := newTestManager(t, dir)

	// Create the log first with custom content.
	memDir := filepath.Join(dir, "memory")
	os.MkdirAll(memDir, 0755)

	// We need to know today's date to create the file.
	if err := mgr.EnsureDailyLog(); err != nil {
		t.Fatalf("first EnsureDailyLog: %v", err)
	}

	// Find the created file and modify it.
	entries, _ := os.ReadDir(memDir)
	if len(entries) == 0 {
		t.Fatal("no log file created")
	}

	logPath := filepath.Join(memDir, entries[0].Name())
	os.WriteFile(logPath, []byte("# Custom Content\n\nImportant note.\n"), 0644)

	// Call again - should not overwrite.
	if err := mgr.EnsureDailyLog(); err != nil {
		t.Fatalf("second EnsureDailyLog: %v", err)
	}

	content, _ := os.ReadFile(logPath)
	if !strings.Contains(string(content), "Important note") {
		t.Error("EnsureDailyLog overwrote existing content")
	}
}

func TestReIndex(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "doc.md"), []byte("# Original\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mgr := newTestManager(t, dir)
	if err := mgr.IndexAll(); err != nil {
		t.Fatalf("IndexAll: %v", err)
	}

	// Verify doc.md is indexed.
	hash1, _ := mgr.store.GetFileHash("doc.md")
	if hash1 == "" {
		t.Fatal("doc.md not indexed")
	}

	// Modify the file.
	os.WriteFile(filepath.Join(dir, "doc.md"), []byte("# Updated Content\n\nNew stuff.\n"), 0644)

	// ReIndex should pick up the change.
	if err := mgr.ReIndex(); err != nil {
		t.Fatalf("ReIndex: %v", err)
	}

	hash2, _ := mgr.store.GetFileHash("doc.md")
	if hash2 == hash1 {
		t.Error("ReIndex did not detect file change")
	}
}

func TestSearchWithSource_FiltersBySource(t *testing.T) {
	dir := t.TempDir()

	// Create a workspace file.
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte("Important workspace notes about authentication.\n"), 0644)

	// Create a session transcript.
	sessDir := filepath.Join(dir, "sessions")
	os.MkdirAll(sessDir, 0755)
	os.WriteFile(filepath.Join(sessDir, "sess.jsonl"),
		[]byte(`{"role":"assistant","content":"We discussed authentication patterns and OAuth2 flows."}`+"\n"), 0644)

	mgr := newTestManager(t, dir)

	// Index workspace files.
	if err := mgr.IndexAll(); err != nil {
		t.Fatalf("IndexAll: %v", err)
	}

	// Index session transcript.
	if err := mgr.IndexSessionTranscript(filepath.Join("sessions", "sess.jsonl")); err != nil {
		t.Fatalf("IndexSessionTranscript: %v", err)
	}

	// Search all sources.
	allResults, err := mgr.SearchWithSource(context.Background(), "authentication", "all", 10, 0)
	if err != nil {
		t.Fatalf("SearchWithSource all: %v", err)
	}

	// Search memory only.
	memResults, err := mgr.SearchWithSource(context.Background(), "authentication", "memory", 10, 0)
	if err != nil {
		t.Fatalf("SearchWithSource memory: %v", err)
	}

	// Search sessions only.
	sessResults, err := mgr.SearchWithSource(context.Background(), "authentication", "sessions", 10, 0)
	if err != nil {
		t.Fatalf("SearchWithSource sessions: %v", err)
	}

	// All results should include both sources.
	if len(allResults) == 0 {
		t.Fatal("expected results for 'authentication' across all sources")
	}

	// Memory results should only have memory source.
	for _, r := range memResults {
		if r.Source != "memory" {
			t.Errorf("memory filter returned source=%q", r.Source)
		}
	}

	// Session results should only have sessions source.
	for _, r := range sessResults {
		if r.Source != "sessions" {
			t.Errorf("sessions filter returned source=%q", r.Source)
		}
	}
}
