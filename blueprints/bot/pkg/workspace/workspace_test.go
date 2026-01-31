package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// 7. Constants test -- verify all 8 constants match OpenClaw file names
// ---------------------------------------------------------------------------

func TestConstants(t *testing.T) {
	expected := map[string]string{
		"AgentsFile":    "AGENTS.md",
		"SoulFile":      "SOUL.md",
		"ToolsFile":     "TOOLS.md",
		"IdentityFile":  "IDENTITY.md",
		"UserFile":      "USER.md",
		"HeartbeatFile": "HEARTBEAT.md",
		"BootFile":      "BOOTSTRAP.md",
		"MemoryFile":    "MEMORY.md",
	}

	actual := map[string]string{
		"AgentsFile":    AgentsFile,
		"SoulFile":      SoulFile,
		"ToolsFile":     ToolsFile,
		"IdentityFile":  IdentityFile,
		"UserFile":      UserFile,
		"HeartbeatFile": HeartbeatFile,
		"BootFile":      BootFile,
		"MemoryFile":    MemoryFile,
	}

	if len(expected) != 8 {
		t.Fatalf("expected 8 constants, got %d", len(expected))
	}

	for name, want := range expected {
		got, ok := actual[name]
		if !ok {
			t.Errorf("constant %s not found", name)
			continue
		}
		if got != want {
			t.Errorf("constant %s = %q, want %q", name, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// 1. LoadBootstrapFiles
// ---------------------------------------------------------------------------

func TestLoadBootstrapFiles_Order(t *testing.T) {
	dir := t.TempDir()

	// Create a subset of files -- some present, some missing.
	presentFiles := map[string]string{
		AgentsFile:   "agents content",
		SoulFile:     "soul content",
		ToolsFile:    "tools content",
		UserFile:     "user content",
		HeartbeatFile: "heartbeat content",
	}
	for name, content := range presentFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	files, err := LoadBootstrapFiles(dir)
	if err != nil {
		t.Fatalf("LoadBootstrapFiles: %v", err)
	}

	// Must always return exactly 8 files.
	if len(files) != 8 {
		t.Fatalf("expected 8 files, got %d", len(files))
	}

	// Verify OpenClaw order.
	wantOrder := []string{
		AgentsFile,
		SoulFile,
		ToolsFile,
		IdentityFile,
		UserFile,
		HeartbeatFile,
		BootFile,
		MemoryFile,
	}
	for i, want := range wantOrder {
		if files[i].Name != want {
			t.Errorf("files[%d].Name = %q, want %q", i, files[i].Name, want)
		}
	}
}

func TestLoadBootstrapFiles_PresentFiles(t *testing.T) {
	dir := t.TempDir()

	content := "present file content\n"
	for _, name := range []string{AgentsFile, SoulFile, ToolsFile} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	files, err := LoadBootstrapFiles(dir)
	if err != nil {
		t.Fatalf("LoadBootstrapFiles: %v", err)
	}

	for _, f := range files {
		switch f.Name {
		case AgentsFile, SoulFile, ToolsFile:
			if f.Missing {
				t.Errorf("%s: expected Missing=false", f.Name)
			}
			if f.Content != content {
				t.Errorf("%s: Content = %q, want %q", f.Name, f.Content, content)
			}
			if f.Path != filepath.Join(dir, f.Name) {
				t.Errorf("%s: Path = %q, want %q", f.Name, f.Path, filepath.Join(dir, f.Name))
			}
		}
	}
}

func TestLoadBootstrapFiles_MissingFiles(t *testing.T) {
	dir := t.TempDir()

	// Only create AGENTS.md -- all others should be missing.
	if err := os.WriteFile(filepath.Join(dir, AgentsFile), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := LoadBootstrapFiles(dir)
	if err != nil {
		t.Fatalf("LoadBootstrapFiles: %v", err)
	}

	for _, f := range files {
		if f.Name == AgentsFile {
			if f.Missing {
				t.Errorf("AGENTS.md should not be missing")
			}
			continue
		}
		if !f.Missing {
			t.Errorf("%s: expected Missing=true", f.Name)
		}
		if f.Content != "" {
			t.Errorf("%s: expected empty Content for missing file, got %q", f.Name, f.Content)
		}
	}
}

func TestLoadBootstrapFiles_AlwaysReturnsEight(t *testing.T) {
	dir := t.TempDir()

	// Empty directory -- no files at all.
	files, err := LoadBootstrapFiles(dir)
	if err != nil {
		t.Fatalf("LoadBootstrapFiles: %v", err)
	}
	if len(files) != 8 {
		t.Fatalf("expected 8 files from empty dir, got %d", len(files))
	}

	// All missing.
	for _, f := range files {
		if !f.Missing {
			t.Errorf("%s should be missing in empty dir", f.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// 2. FilterForSubagent
// ---------------------------------------------------------------------------

func TestFilterForSubagent(t *testing.T) {
	dir := t.TempDir()

	// Create all 8 files so none are missing.
	allNames := []string{
		AgentsFile, SoulFile, ToolsFile, IdentityFile,
		UserFile, HeartbeatFile, BootFile, MemoryFile,
	}
	for _, name := range allNames {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name+" content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := LoadBootstrapFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	sub := FilterForSubagent(files)

	// Must return exactly AGENTS.md and TOOLS.md.
	if len(sub) != 2 {
		t.Fatalf("expected 2 subagent files, got %d", len(sub))
	}

	names := map[string]bool{}
	for _, f := range sub {
		names[f.Name] = true
	}
	if !names[AgentsFile] {
		t.Error("subagent filter should include AGENTS.md")
	}
	if !names[ToolsFile] {
		t.Error("subagent filter should include TOOLS.md")
	}
}

func TestFilterForSubagent_SkipsMissing(t *testing.T) {
	// Provide all 8 files but mark AGENTS.md and TOOLS.md as missing.
	files := []BootstrapFile{
		{Name: AgentsFile, Missing: true},
		{Name: SoulFile, Missing: false, Content: "soul"},
		{Name: ToolsFile, Missing: true},
		{Name: IdentityFile, Missing: false, Content: "id"},
		{Name: UserFile, Missing: false, Content: "user"},
		{Name: HeartbeatFile, Missing: false, Content: "hb"},
		{Name: BootFile, Missing: false, Content: "boot"},
		{Name: MemoryFile, Missing: false, Content: "mem"},
	}

	sub := FilterForSubagent(files)
	if len(sub) != 0 {
		t.Fatalf("expected 0 subagent files when both are missing, got %d", len(sub))
	}
}

func TestFilterForSubagent_OnlyOnePresent(t *testing.T) {
	files := []BootstrapFile{
		{Name: AgentsFile, Missing: false, Content: "agents"},
		{Name: SoulFile, Missing: false, Content: "soul"},
		{Name: ToolsFile, Missing: true},
		{Name: IdentityFile, Missing: false, Content: "id"},
	}

	sub := FilterForSubagent(files)
	if len(sub) != 1 {
		t.Fatalf("expected 1 subagent file, got %d", len(sub))
	}
	if sub[0].Name != AgentsFile {
		t.Errorf("expected AGENTS.md, got %s", sub[0].Name)
	}
}

// ---------------------------------------------------------------------------
// 3. FilterForMain
// ---------------------------------------------------------------------------

func TestFilterForMain(t *testing.T) {
	files := []BootstrapFile{
		{Name: AgentsFile, Missing: false, Content: "agents"},
		{Name: SoulFile, Missing: true},
		{Name: ToolsFile, Missing: false, Content: "tools"},
		{Name: IdentityFile, Missing: true},
		{Name: UserFile, Missing: false, Content: "user"},
		{Name: HeartbeatFile, Missing: true},
		{Name: BootFile, Missing: false, Content: "boot"},
		{Name: MemoryFile, Missing: true},
	}

	main := FilterForMain(files)

	if len(main) != 4 {
		t.Fatalf("expected 4 non-missing files, got %d", len(main))
	}

	wantNames := []string{AgentsFile, ToolsFile, UserFile, BootFile}
	for i, want := range wantNames {
		if main[i].Name != want {
			t.Errorf("main[%d].Name = %q, want %q", i, main[i].Name, want)
		}
	}
}

func TestFilterForMain_AllPresent(t *testing.T) {
	files := make([]BootstrapFile, 0, 8)
	for _, name := range []string{
		AgentsFile, SoulFile, ToolsFile, IdentityFile,
		UserFile, HeartbeatFile, BootFile, MemoryFile,
	} {
		files = append(files, BootstrapFile{Name: name, Missing: false, Content: name})
	}

	main := FilterForMain(files)
	if len(main) != 8 {
		t.Fatalf("expected 8 when all present, got %d", len(main))
	}
}

func TestFilterForMain_AllMissing(t *testing.T) {
	files := make([]BootstrapFile, 0, 8)
	for _, name := range []string{
		AgentsFile, SoulFile, ToolsFile, IdentityFile,
		UserFile, HeartbeatFile, BootFile, MemoryFile,
	} {
		files = append(files, BootstrapFile{Name: name, Missing: true})
	}

	main := FilterForMain(files)
	if len(main) != 0 {
		t.Fatalf("expected 0 when all missing, got %d", len(main))
	}
}

// ---------------------------------------------------------------------------
// 4. BuildContextPrompt
// ---------------------------------------------------------------------------

func TestBuildContextPrompt_Header(t *testing.T) {
	files := []BootstrapFile{
		{Name: AgentsFile, Content: "agents content\n"},
	}

	prompt := BuildContextPrompt(files)
	if !strings.HasPrefix(prompt, "# Project Context\n") {
		t.Errorf("prompt should start with '# Project Context\\n', got prefix: %q", prompt[:min(len(prompt), 40)])
	}
}

func TestBuildContextPrompt_FileHeaders(t *testing.T) {
	files := []BootstrapFile{
		{Name: AgentsFile, Content: "agents content\n"},
		{Name: SoulFile, Content: "soul content\n"},
	}

	prompt := BuildContextPrompt(files)

	if !strings.Contains(prompt, "## "+AgentsFile+"\n") {
		t.Error("prompt should contain '## AGENTS.md' header")
	}
	if !strings.Contains(prompt, "## "+SoulFile+"\n") {
		t.Error("prompt should contain '## SOUL.md' header")
	}
	if !strings.Contains(prompt, "agents content") {
		t.Error("prompt should contain agents content")
	}
	if !strings.Contains(prompt, "soul content") {
		t.Error("prompt should contain soul content")
	}
}

func TestBuildContextPrompt_ExcludesMissing(t *testing.T) {
	files := []BootstrapFile{
		{Name: AgentsFile, Content: "agents content\n"},
		{Name: SoulFile, Missing: true, Content: ""},
		{Name: ToolsFile, Content: "tools content\n"},
	}

	prompt := BuildContextPrompt(files)

	if strings.Contains(prompt, SoulFile) {
		t.Error("prompt should not contain missing file SOUL.md")
	}
	if !strings.Contains(prompt, AgentsFile) {
		t.Error("prompt should contain AGENTS.md")
	}
	if !strings.Contains(prompt, ToolsFile) {
		t.Error("prompt should contain TOOLS.md")
	}
}

func TestBuildContextPrompt_ExcludesEmptyContent(t *testing.T) {
	files := []BootstrapFile{
		{Name: AgentsFile, Content: "agents content\n"},
		{Name: SoulFile, Missing: false, Content: ""},
		{Name: ToolsFile, Content: "tools content\n"},
	}

	prompt := BuildContextPrompt(files)

	if strings.Contains(prompt, "## "+SoulFile) {
		t.Error("prompt should not contain file with empty content")
	}
}

func TestBuildContextPrompt_NoTrailingNewlineHandled(t *testing.T) {
	// Content without trailing newline should still produce clean output.
	files := []BootstrapFile{
		{Name: AgentsFile, Content: "no trailing newline"},
	}

	prompt := BuildContextPrompt(files)
	// The implementation appends \n if content doesn't end with one.
	if !strings.Contains(prompt, "no trailing newline\n") {
		t.Error("content without trailing newline should get one appended")
	}
}

func TestBuildContextPrompt_EmptyFiles(t *testing.T) {
	files := []BootstrapFile{}

	prompt := BuildContextPrompt(files)
	if prompt != "# Project Context\n" {
		t.Errorf("empty files should produce only header, got %q", prompt)
	}
}

// ---------------------------------------------------------------------------
// 5. EnsureWorkspace
// ---------------------------------------------------------------------------

func TestEnsureWorkspace_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "newworkspace")

	if err := EnsureWorkspace(dir); err != nil {
		t.Fatalf("EnsureWorkspace: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("workspace dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("workspace path is not a directory")
	}
}

func TestEnsureWorkspace_CreatesDefaultFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ws")

	if err := EnsureWorkspace(dir); err != nil {
		t.Fatal(err)
	}

	expectedFiles := []string{AgentsFile, SoulFile, UserFile, ToolsFile}
	for _, name := range expectedFiles {
		p := filepath.Join(dir, name)
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
			continue
		}
		if info.IsDir() {
			t.Errorf("%s should be a file, not a directory", name)
		}
		if info.Size() == 0 {
			t.Errorf("%s should have non-empty default content", name)
		}
	}
}

func TestEnsureWorkspace_CreatesMemoryDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ws")

	if err := EnsureWorkspace(dir); err != nil {
		t.Fatal(err)
	}

	memDir := filepath.Join(dir, "memory")
	info, err := os.Stat(memDir)
	if err != nil {
		t.Fatalf("memory dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("memory path should be a directory")
	}
}

func TestEnsureWorkspace_DoesNotOverwrite(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ws")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a custom AGENTS.md before EnsureWorkspace.
	customContent := "my custom agents content"
	agentsPath := filepath.Join(dir, AgentsFile)
	if err := os.WriteFile(agentsPath, []byte(customContent), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureWorkspace(dir); err != nil {
		t.Fatal(err)
	}

	// Verify custom content is preserved.
	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != customContent {
		t.Errorf("AGENTS.md was overwritten: got %q, want %q", string(data), customContent)
	}
}

func TestEnsureWorkspace_AgentsDefaultContainsSoulRef(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ws")

	if err := EnsureWorkspace(dir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, AgentsFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "SOUL.md") {
		t.Error("AGENTS.md default content should reference SOUL.md")
	}
}

func TestEnsureWorkspace_SoulDefaultContainsHelpful(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ws")

	if err := EnsureWorkspace(dir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, SoulFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "helpful") {
		t.Error("SOUL.md default content should contain 'helpful'")
	}
}

func TestEnsureWorkspace_Idempotent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ws")

	// Run twice -- second call should succeed without error.
	if err := EnsureWorkspace(dir); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := EnsureWorkspace(dir); err != nil {
		t.Fatalf("second call: %v", err)
	}

	// Files should still exist.
	for _, name := range []string{AgentsFile, SoulFile, UserFile, ToolsFile} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s missing after idempotent call: %v", name, err)
		}
	}
}

// ---------------------------------------------------------------------------
// 6. EnsureMemoryDir
// ---------------------------------------------------------------------------

func TestEnsureMemoryDir_CreatesDir(t *testing.T) {
	dir := t.TempDir()

	if err := EnsureMemoryDir(dir); err != nil {
		t.Fatalf("EnsureMemoryDir: %v", err)
	}

	memDir := filepath.Join(dir, "memory")
	info, err := os.Stat(memDir)
	if err != nil {
		t.Fatalf("memory dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("memory path should be a directory")
	}
}

func TestEnsureMemoryDir_Idempotent(t *testing.T) {
	dir := t.TempDir()

	if err := EnsureMemoryDir(dir); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := EnsureMemoryDir(dir); err != nil {
		t.Fatalf("second call: %v", err)
	}

	memDir := filepath.Join(dir, "memory")
	info, err := os.Stat(memDir)
	if err != nil {
		t.Fatalf("memory dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("memory path should be a directory")
	}
}

// ---------------------------------------------------------------------------
// Integration: LoadBootstrapFiles after EnsureWorkspace
// ---------------------------------------------------------------------------

func TestLoadBootstrapFiles_AfterEnsureWorkspace(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ws")

	if err := EnsureWorkspace(dir); err != nil {
		t.Fatal(err)
	}

	files, err := LoadBootstrapFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 8 {
		t.Fatalf("expected 8 files, got %d", len(files))
	}

	// AGENTS, SOUL, TOOLS, USER should be present (created by EnsureWorkspace).
	presentNames := map[string]bool{
		AgentsFile: true,
		SoulFile:   true,
		ToolsFile:  true,
		UserFile:   true,
	}

	for _, f := range files {
		if presentNames[f.Name] {
			if f.Missing {
				t.Errorf("%s should not be missing after EnsureWorkspace", f.Name)
			}
			if f.Content == "" {
				t.Errorf("%s should have content after EnsureWorkspace", f.Name)
			}
		} else {
			// IDENTITY, HEARTBEAT, BOOTSTRAP, MEMORY are not created by default.
			if !f.Missing {
				t.Errorf("%s should be missing (not a default template file)", f.Name)
			}
		}
	}
}
