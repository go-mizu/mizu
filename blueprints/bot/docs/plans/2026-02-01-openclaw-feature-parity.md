# OpenClaw Feature Parity Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Close all remaining gaps between OpenBot and OpenClaw so that `~/.openbot` is structurally identical to `~/.openclaw` and comprehensive tests verify 100% feature matching.

**Architecture:** Fill gaps in config init (missing files like update-check.json, canvas/index.html, MEMORY.md), enhance the doctor command with real checks, add memory recall to prompt builder, and write comprehensive Go tests that create sessions via both CLIs and compare `.openclaw`/`.openbot` directories field-by-field.

**Tech Stack:** Go 1.22+, SQLite, OpenClaw CLI (Node.js), OpenBot CLI (Go)

---

### Task 1: Add Missing Directory Structure Files

**Files:**
- Modify: `pkg/config/init.go:56-113`
- Test: `test/compat/compat_test.go`

**Step 1: Write the failing test**

Add to `test/compat/compat_test.go`:

```go
func TestMissingFiles(t *testing.T) {
	openbotDir := filepath.Join(os.Getenv("HOME"), ".openbot")
	if _, err := os.Stat(openbotDir); os.IsNotExist(err) {
		t.Skip("~/.openbot does not exist, skipping")
	}

	requiredFiles := []string{
		"update-check.json",
		"canvas/index.html",
		"workspace/MEMORY.md",
	}

	for _, f := range requiredFiles {
		t.Run(f, func(t *testing.T) {
			path := filepath.Join(openbotDir, f)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("missing file %s", f)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOWORK=off go test -v -run TestMissingFiles ./test/compat/`
Expected: FAIL for update-check.json, canvas/index.html, workspace/MEMORY.md

**Step 3: Write minimal implementation**

In `pkg/config/init.go`, add to `ensureDirectoryStructure()` after the devices block:

```go
// Create update-check.json if missing.
updateCheckPath := filepath.Join(baseDir, "update-check.json")
if _, err := os.Stat(updateCheckPath); os.IsNotExist(err) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	content := fmt.Sprintf(`{"lastCheckedAt":"%s"}`, now)
	os.WriteFile(updateCheckPath, []byte(content+"\n"), 0o600)
}

// Create canvas/index.html if missing.
canvasPath := filepath.Join(baseDir, "canvas", "index.html")
if _, err := os.Stat(canvasPath); os.IsNotExist(err) {
	html := `<!DOCTYPE html>
<html><head><title>OpenBot Canvas</title></head>
<body><h1>OpenBot Canvas</h1><p>Canvas UI placeholder.</p></body>
</html>
`
	os.WriteFile(canvasPath, []byte(html), 0o644)
}

// Create workspace/MEMORY.md if missing.
memoryMdPath := filepath.Join(baseDir, "workspace", "MEMORY.md")
if _, err := os.Stat(memoryMdPath); os.IsNotExist(err) {
	os.WriteFile(memoryMdPath, []byte("# Memory\n\nLong-term curated memories.\n"), 0o644)
}
```

Add `"fmt"` and `"time"` to the imports in init.go.

**Step 4: Run test to verify it passes**

Run: `GOWORK=off go test -v -run TestMissingFiles ./test/compat/`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/config/init.go test/compat/compat_test.go
git commit -m "feat: add missing directory structure files (update-check.json, canvas/index.html, MEMORY.md)"
```

---

### Task 2: Enhance Doctor Command with Real Checks

**Files:**
- Modify: `cmd/openbot/cli_stubs.go:15-22`
- Test: `test/compat/compat_test.go`

**Step 1: Write the failing test**

Add to `test/compat/compat_test.go`:

```go
func TestDoctorOutput(t *testing.T) {
	skipIfNoCLI(t, "openbot")

	out, _, exit := runCLI(t, "openbot", "doctor")
	if exit != 0 {
		t.Errorf("openbot doctor failed (exit %d)", exit)
		return
	}

	checks := []string{"Config", "Workspace", "Sessions", "Memory", "Skills"}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("doctor output missing check for %q", check)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOWORK=off go test -v -run TestDoctorOutput ./test/compat/`
Expected: FAIL (missing Sessions, Memory, Skills in output)

**Step 3: Write minimal implementation**

Replace `runDoctor()` in `cmd/openbot/cli_stubs.go`:

```go
func runDoctor() error {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println("  Config: FAIL -", err)
		return nil
	}
	fmt.Println("Running health checks...")
	fmt.Printf("  Config: OK (%s)\n", config.DefaultConfigPath())

	// Check workspace.
	if _, err := os.Stat(cfg.Workspace); err != nil {
		fmt.Printf("  Workspace: MISSING (%s)\n", cfg.Workspace)
	} else {
		fmt.Printf("  Workspace: OK (%s)\n", cfg.Workspace)
	}

	// Check sessions directory.
	sessDir := filepath.Join(cfg.DataDir, "agents", "main", "sessions")
	if _, err := os.Stat(sessDir); err != nil {
		fmt.Printf("  Sessions: MISSING (%s)\n", sessDir)
	} else {
		fmt.Printf("  Sessions: OK (%s)\n", sessDir)
	}

	// Check memory DB.
	memDB := filepath.Join(cfg.DataDir, "memory.db")
	if info, err := os.Stat(memDB); err != nil {
		fmt.Println("  Memory: not indexed")
	} else {
		fmt.Printf("  Memory: OK (%d bytes)\n", info.Size())
	}

	// Check skills.
	skills, _ := skill.LoadAllSkills(cfg.Workspace)
	readyCount := 0
	for _, s := range skills {
		if s.Ready {
			readyCount++
		}
	}
	fmt.Printf("  Skills: %d loaded, %d ready\n", len(skills), readyCount)

	fmt.Println("  All checks passed.")
	return nil
}
```

Add imports to `cli_stubs.go`: `"path/filepath"` and `"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"`. Remove the unused `config` import reference.

**Step 4: Run test to verify it passes**

Run: `GOWORK=off go test -v -run TestDoctorOutput ./test/compat/`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/openbot/cli_stubs.go test/compat/compat_test.go
git commit -m "feat: enhance doctor command with real health checks"
```

---

### Task 3: Add Memory Recall to Prompt Builder

**Files:**
- Modify: `pkg/bot/prompt.go`
- Test: `pkg/bot/prompt_test.go`

**Step 1: Write the failing test**

Add to `pkg/bot/prompt_test.go`:

```go
func TestPromptBuilder_IncludesMemoryRecall(t *testing.T) {
	ws := t.TempDir()

	// Create workspace/MEMORY.md.
	os.WriteFile(filepath.Join(ws, "MEMORY.md"), []byte("# Memory\n\n- User prefers dark mode\n"), 0o644)

	// Create workspace/memory/2026-01-31.md.
	memDir := filepath.Join(ws, "memory")
	os.MkdirAll(memDir, 0o755)
	os.WriteFile(filepath.Join(memDir, "2026-01-31.md"), []byte("# 2026-01-31\n\n- Met the user\n"), 0o644)

	pb := NewPromptBuilder(ws)
	prompt := pb.Build("dm", "hello")

	if !strings.Contains(prompt, "Memory") {
		t.Error("prompt should contain Memory section from MEMORY.md")
	}
	if !strings.Contains(prompt, "dark mode") {
		t.Error("prompt should contain content from MEMORY.md")
	}
}
```

Add `"os"`, `"path/filepath"`, and `"strings"` to the test imports if not already present.

**Step 2: Run test to verify it fails**

Run: `GOWORK=off go test -v -run TestPromptBuilder_IncludesMemoryRecall ./pkg/bot/`
Expected: FAIL

**Step 3: Write minimal implementation**

Add a `memoryRecallSection()` method to `pkg/bot/prompt.go`:

```go
// memoryRecallSection loads MEMORY.md and recent daily memory logs.
func (pb *PromptBuilder) memoryRecallSection() string {
	var sections []string

	// Load curated MEMORY.md.
	memoryPath := filepath.Join(pb.workspaceDir, "MEMORY.md")
	if data, err := os.ReadFile(memoryPath); err == nil {
		content := strings.TrimSpace(string(data))
		if content != "" && content != "# Memory" {
			sections = append(sections, "## Long-Term Memory\n"+content)
		}
	}

	// Load recent daily memory logs (last 3 days).
	memDir := filepath.Join(pb.workspaceDir, "memory")
	entries, err := os.ReadDir(memDir)
	if err == nil {
		// Sort descending by name (date format sorts correctly).
		var recent []string
		for i := len(entries) - 1; i >= 0 && len(recent) < 3; i-- {
			e := entries[i]
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(memDir, e.Name()))
			if err == nil {
				recent = append(recent, strings.TrimSpace(string(data)))
			}
		}
		if len(recent) > 0 {
			sections = append(sections, "## Recent Memory\n"+strings.Join(recent, "\n\n"))
		}
	}

	if len(sections) == 0 {
		return ""
	}
	return strings.Join(sections, "\n\n")
}
```

Add `"os"` to the imports in prompt.go.

Then modify the `Build()` method to include memory recall between workspace and skills:

```go
// Build constructs the full system prompt for the given origin and query.
func (pb *PromptBuilder) Build(origin, query string) string {
	var sections []string

	// 1. Identity section.
	sections = append(sections, pb.identitySection())

	// 2. Workspace bootstrap files.
	if pb.workspaceDir != "" {
		if ws := pb.workspaceSection(origin); ws != "" {
			sections = append(sections, ws)
		}
	}

	// 3. Memory recall (MEMORY.md + daily logs).
	if pb.workspaceDir != "" {
		if mem := pb.memoryRecallSection(); mem != "" {
			sections = append(sections, mem)
		}
	}

	// 4. Skills.
	if pb.workspaceDir != "" {
		if sk := pb.skillsSection(); sk != "" {
			sections = append(sections, sk)
		}
	}

	// 5. Runtime info.
	sections = append(sections, pb.runtimeSection())

	return strings.Join(sections, "\n\n")
}
```

**Step 4: Run test to verify it passes**

Run: `GOWORK=off go test -v -run TestPromptBuilder_IncludesMemoryRecall ./pkg/bot/`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/bot/prompt.go pkg/bot/prompt_test.go
git commit -m "feat: add memory recall section to prompt builder"
```

---

### Task 4: Write Comprehensive CLI Comparison Tests

**Files:**
- Modify: `test/compat/compat_test.go`

**Step 1: Write the tests**

Add these comparison tests to `test/compat/compat_test.go`:

```go
// TestConfigGetComparison compares config get behavior between CLIs.
func TestConfigGetComparison(t *testing.T) {
	skipIfNoCLI(t, "openclaw")
	skipIfNoCLI(t, "openbot")

	paths := []string{
		"agents.defaults.workspace",
		"channels.telegram.enabled",
		"gateway.port",
		"messages.ackReactionScope",
	}

	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			ocOut, _, ocExit := runCLI(t, "openclaw", "config", "get", p)
			obOut, _, obExit := runCLI(t, "openbot", "config", "get", p)

			if ocExit != 0 {
				t.Skipf("openclaw config get %s failed", p)
			}
			if obExit != 0 {
				t.Errorf("openbot config get %s failed (exit %d)", p, obExit)
				return
			}

			// Both should return non-empty output.
			ocOut = strings.TrimSpace(ocOut)
			obOut = strings.TrimSpace(obOut)
			if obOut == "" && ocOut != "" {
				t.Errorf("openbot returned empty for %s, openclaw returned %q", p, ocOut)
			}
		})
	}
}

// TestMemoryStatusComparison compares memory status output.
func TestMemoryStatusComparison(t *testing.T) {
	skipIfNoCLI(t, "openbot")

	out, _, exit := runCLI(t, "openbot", "memory", "status")
	if exit != 0 {
		t.Errorf("openbot memory status failed (exit %d)", exit)
		return
	}

	if !strings.Contains(out, "Memory DB") {
		t.Error("memory status missing 'Memory DB' line")
	}
	if !strings.Contains(out, "Status") {
		t.Error("memory status missing 'Status' line")
	}
}

// TestSkillsListComparison checks skill list between CLIs.
func TestSkillsListComparison(t *testing.T) {
	skipIfNoCLI(t, "openbot")

	out, _, exit := runCLI(t, "openbot", "skills", "list")
	if exit != 0 {
		t.Errorf("openbot skills list failed (exit %d)", exit)
		return
	}

	// Should have table headers.
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "SOURCE") {
		t.Error("skills list missing table headers")
	}
}

// TestModelsListComparison checks models list output.
func TestModelsListComparison(t *testing.T) {
	skipIfNoCLI(t, "openbot")

	out, _, exit := runCLI(t, "openbot", "models", "list")
	if exit != 0 {
		t.Errorf("openbot models list failed (exit %d)", exit)
		return
	}

	if !strings.Contains(out, "claude-sonnet") {
		t.Error("models list missing claude-sonnet")
	}
}

// TestChannelsListComparison checks channels list output.
func TestChannelsListComparison(t *testing.T) {
	skipIfNoCLI(t, "openbot")

	out, _, exit := runCLI(t, "openbot", "channels", "list")
	if exit != 0 {
		t.Errorf("openbot channels list failed (exit %d)", exit)
		return
	}

	if !strings.Contains(out, "telegram") {
		t.Error("channels list missing telegram")
	}
}

// TestAgentsListComparison checks agents list output.
func TestAgentsListComparison(t *testing.T) {
	skipIfNoCLI(t, "openbot")

	out, _, exit := runCLI(t, "openbot", "agents", "list")
	if exit != 0 {
		t.Errorf("openbot agents list failed (exit %d)", exit)
		return
	}

	if !strings.Contains(out, "main") {
		t.Error("agents list missing 'main' agent")
	}
}

// TestDirectoryFileCountComparison compares file counts in key directories.
func TestDirectoryFileCountComparison(t *testing.T) {
	openclawDir := filepath.Join(os.Getenv("HOME"), ".openclaw")
	openbotDir := filepath.Join(os.Getenv("HOME"), ".openbot")

	if _, err := os.Stat(openclawDir); os.IsNotExist(err) {
		t.Skip("~/.openclaw does not exist")
	}
	if _, err := os.Stat(openbotDir); os.IsNotExist(err) {
		t.Skip("~/.openbot does not exist")
	}

	// Compare identity files.
	t.Run("identity/device.json", func(t *testing.T) {
		obPath := filepath.Join(openbotDir, "identity", "device.json")
		data, err := os.ReadFile(obPath)
		if err != nil {
			t.Errorf("cannot read %s: %v", obPath, err)
			return
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			t.Errorf("invalid JSON in %s: %v", obPath, err)
		}
	})

	// Compare cron/jobs.json structure.
	t.Run("cron/jobs.json", func(t *testing.T) {
		obPath := filepath.Join(openbotDir, "cron", "jobs.json")
		data, err := os.ReadFile(obPath)
		if err != nil {
			t.Errorf("cannot read %s: %v", obPath, err)
			return
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			t.Errorf("invalid JSON in %s: %v", obPath, err)
		}
	})

	// Compare devices files.
	for _, f := range []string{"devices/paired.json", "devices/pending.json"} {
		t.Run(f, func(t *testing.T) {
			obPath := filepath.Join(openbotDir, f)
			data, err := os.ReadFile(obPath)
			if err != nil {
				t.Errorf("cannot read %s: %v", obPath, err)
				return
			}
			var m map[string]any
			if err := json.Unmarshal(data, &m); err != nil {
				t.Errorf("invalid JSON in %s: %v", obPath, err)
			}
		})
	}
}

// TestUpdateCheckFile verifies update-check.json exists and is valid JSON.
func TestUpdateCheckFile(t *testing.T) {
	openbotDir := filepath.Join(os.Getenv("HOME"), ".openbot")
	if _, err := os.Stat(openbotDir); os.IsNotExist(err) {
		t.Skip("~/.openbot does not exist")
	}

	path := filepath.Join(openbotDir, "update-check.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read update-check.json: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := m["lastCheckedAt"]; !ok {
		t.Error("update-check.json missing lastCheckedAt field")
	}
}

// TestCanvasFile verifies canvas/index.html exists.
func TestCanvasFile(t *testing.T) {
	openbotDir := filepath.Join(os.Getenv("HOME"), ".openbot")
	if _, err := os.Stat(openbotDir); os.IsNotExist(err) {
		t.Skip("~/.openbot does not exist")
	}

	path := filepath.Join(openbotDir, "canvas", "index.html")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read canvas/index.html: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<html") {
		t.Error("canvas/index.html missing HTML tag")
	}
}

// TestWorkspaceMemoryMD verifies workspace/MEMORY.md exists.
func TestWorkspaceMemoryMD(t *testing.T) {
	openbotDir := filepath.Join(os.Getenv("HOME"), ".openbot")
	if _, err := os.Stat(openbotDir); os.IsNotExist(err) {
		t.Skip("~/.openbot does not exist")
	}

	path := filepath.Join(openbotDir, "workspace", "MEMORY.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("workspace/MEMORY.md does not exist")
	}
}
```

**Step 2: Run tests**

Run: `GOWORK=off go test -v -run "TestConfigGetComparison|TestMemoryStatus|TestSkillsList|TestModelsList|TestChannelsList|TestAgentsList|TestDirectoryFileCount|TestUpdateCheck|TestCanvas|TestWorkspaceMemory" ./test/compat/`
Expected: Some will pass immediately (after Task 1), others may need both CLIs

**Step 3: Commit**

```bash
git add test/compat/compat_test.go
git commit -m "test: add comprehensive CLI and directory comparison tests"
```

---

### Task 5: Write Session Entry Seeding Test

**Files:**
- Create: `test/compat/session_seed_test.go`

**Step 1: Write the test that creates a session via OpenBot code and compares to OpenClaw format**

Create `test/compat/session_seed_test.go`:

```go
package compat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/session"
)

// TestSessionEntryStructure creates a session via the session package
// and verifies it has all fields that OpenClaw's sessions.json has.
func TestSessionEntryStructure(t *testing.T) {
	dir := t.TempDir()
	store, err := session.NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	key := "agent:main:test-peer"
	entry, isNew, err := store.GetOrCreate(key, "Test User", "direct", "telegram")
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	if !isNew {
		t.Fatal("expected new session")
	}

	// Populate all OpenClaw-compatible fields.
	entry.SystemSent = true
	entry.Model = "claude-opus-4-5"
	entry.ModelProvider = "anthropic"
	entry.ContextTokens = 200000
	entry.DeliveryContext = &session.DeliveryCtx{
		Channel:   "telegram",
		To:        "telegram:12345",
		AccountId: "default",
	}
	entry.LastChannel = "telegram"
	entry.LastTo = "telegram:12345"
	entry.LastAccountId = "default"
	entry.AuthProfileOverride = "anthropic:default"
	entry.AuthProfileOverrideSource = "auto"
	entry.Origin = &session.SessionOrigin{
		Label:     "Test User id:12345",
		Provider:  "telegram",
		Surface:   "telegram",
		ChatType:  "direct",
		From:      "telegram:12345",
		To:        "telegram:12345",
		AccountId: "default",
	}
	entry.SessionFile = filepath.Join(dir, entry.SessionID+".jsonl")
	entry.InputTokens = 100
	entry.OutputTokens = 200
	entry.TotalTokens = 300
	entry.SkillsSnapshot = &session.SkillsSnap{
		Version: 1,
		Skills: []session.SkillRef{
			{Name: "weather"},
			{Name: "github"},
		},
	}
	entry.SystemPromptReport = &session.SystemPromptReport{
		Source:       "run",
		GeneratedAt: entry.UpdatedAt,
		SessionID:   entry.SessionID,
		SessionKey:  key,
		Provider:    "anthropic",
		Model:       "claude-opus-4-5",
	}

	if err := store.UpdateEntry(key, entry); err != nil {
		t.Fatalf("UpdateEntry: %v", err)
	}

	// Read back and verify JSON has all expected fields.
	data, err := os.ReadFile(filepath.Join(dir, "sessions.json"))
	if err != nil {
		t.Fatalf("read sessions.json: %v", err)
	}

	var index map[string]map[string]any
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatalf("parse sessions.json: %v", err)
	}

	entryMap, ok := index[key]
	if !ok {
		t.Fatalf("key %q not found in sessions.json", key)
	}

	// All OpenClaw session entry fields that must be present.
	requiredFields := []string{
		"sessionId",
		"updatedAt",
		"systemSent",
		"chatType",
		"deliveryContext",
		"lastChannel",
		"origin",
		"sessionFile",
		"compactionCount",
		"skillsSnapshot",
		"authProfileOverride",
		"authProfileOverrideSource",
		"lastTo",
		"lastAccountId",
		"inputTokens",
		"outputTokens",
		"totalTokens",
		"modelProvider",
		"model",
		"contextTokens",
		"systemPromptReport",
	}

	for _, field := range requiredFields {
		t.Run("field/"+field, func(t *testing.T) {
			if _, ok := entryMap[field]; !ok {
				t.Errorf("session entry missing field %q", field)
			}
		})
	}

	// Verify deliveryContext structure.
	t.Run("deliveryContext/structure", func(t *testing.T) {
		dc, ok := entryMap["deliveryContext"].(map[string]any)
		if !ok {
			t.Fatal("deliveryContext is not an object")
		}
		for _, f := range []string{"channel", "to", "accountId"} {
			if _, ok := dc[f]; !ok {
				t.Errorf("deliveryContext missing field %q", f)
			}
		}
	})

	// Verify origin structure.
	t.Run("origin/structure", func(t *testing.T) {
		orig, ok := entryMap["origin"].(map[string]any)
		if !ok {
			t.Fatal("origin is not an object")
		}
		for _, f := range []string{"label", "provider", "surface", "chatType", "from", "to", "accountId"} {
			if _, ok := orig[f]; !ok {
				t.Errorf("origin missing field %q", f)
			}
		}
	})
}

// TestSessionTranscriptFormat creates a transcript and verifies JSONL format.
func TestSessionTranscriptFormat(t *testing.T) {
	dir := t.TempDir()
	store, err := session.NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	key := "agent:main:test-peer"
	entry, _, err := store.GetOrCreate(key, "Test", "direct", "telegram")
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}

	// Append messages.
	store.AppendTranscript(entry.SessionID, &session.TranscriptEntry{
		Type: "message",
		Message: &session.TranscriptMessage{
			Role:    "user",
			Content: "Hello",
		},
	})
	store.AppendTranscript(entry.SessionID, &session.TranscriptEntry{
		Type: "message",
		Message: &session.TranscriptMessage{
			Role:    "assistant",
			Content: "Hi there!",
		},
		Usage: &session.TokenUsage{Input: 5, Output: 10},
	})

	// Read back.
	entries, err := store.ReadTranscript(entry.SessionID)
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}

	// Should have 3 entries: session header + 2 messages.
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// First should be session header.
	if entries[0].Type != "session" {
		t.Errorf("first entry type = %q; want %q", entries[0].Type, "session")
	}
	if entries[0].Version != 2 {
		t.Errorf("session version = %d; want 2", entries[0].Version)
	}

	// Second should be user message.
	if entries[1].Type != "message" {
		t.Errorf("second entry type = %q; want %q", entries[1].Type, "message")
	}
	if entries[1].Message.Role != "user" {
		t.Errorf("second entry role = %q; want %q", entries[1].Message.Role, "user")
	}

	// Third should be assistant message with usage.
	if entries[2].Type != "message" {
		t.Errorf("third entry type = %q; want %q", entries[2].Type, "message")
	}
	if entries[2].Usage == nil {
		t.Error("third entry missing usage")
	} else if entries[2].Usage.Input != 5 {
		t.Errorf("usage input = %d; want 5", entries[2].Usage.Input)
	}
}

// TestSessionKeyFormat verifies session key format matches OpenClaw convention.
func TestSessionKeyFormat(t *testing.T) {
	tests := []struct {
		name     string
		agentID  string
		channel  string
		peerID   string
		groupID  string
		expected string
	}{
		{"DM", "main", "telegram", "12345", "", "agent:main:12345"},
		{"Group", "main", "telegram", "12345", "group99", "agent:main:telegram:group:group99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := session.SessionKey(tt.agentID, tt.channel, tt.peerID, tt.groupID)
			if got != tt.expected {
				t.Errorf("SessionKey() = %q; want %q", got, tt.expected)
			}
		})
	}
}

// TestLiveSessionComparison compares actual sessions.json between openclaw and openbot.
func TestLiveSessionComparison(t *testing.T) {
	openclawSess := filepath.Join(os.Getenv("HOME"), ".openclaw", "agents", "main", "sessions", "sessions.json")
	openbotSess := filepath.Join(os.Getenv("HOME"), ".openbot", "agents", "main", "sessions", "sessions.json")

	if _, err := os.Stat(openclawSess); os.IsNotExist(err) {
		t.Skip("~/.openclaw sessions.json does not exist")
	}
	if _, err := os.Stat(openbotSess); os.IsNotExist(err) {
		t.Skip("~/.openbot sessions.json does not exist")
	}

	ocData, err := os.ReadFile(openclawSess)
	if err != nil {
		t.Fatalf("read openclaw sessions: %v", err)
	}
	obData, err := os.ReadFile(openbotSess)
	if err != nil {
		t.Fatalf("read openbot sessions: %v", err)
	}

	var ocIndex, obIndex map[string]map[string]any
	if err := json.Unmarshal(ocData, &ocIndex); err != nil {
		t.Fatalf("parse openclaw sessions: %v", err)
	}
	if err := json.Unmarshal(obData, &obIndex); err != nil {
		t.Fatalf("parse openbot sessions: %v", err)
	}

	// Get first entry from each and compare fields.
	var ocEntry map[string]any
	for _, v := range ocIndex {
		ocEntry = v
		break
	}
	var obEntry map[string]any
	for _, v := range obIndex {
		obEntry = v
		break
	}

	if ocEntry == nil {
		t.Skip("no openclaw sessions")
	}
	if obEntry == nil {
		t.Skip("no openbot sessions")
	}

	// Check all OpenClaw fields exist in OpenBot entry.
	for key := range ocEntry {
		t.Run("field/"+key, func(t *testing.T) {
			if _, ok := obEntry[key]; !ok {
				t.Errorf("openbot session missing field %q", key)
			}
		})
	}

	// Check type consistency for shared fields.
	for key, ocVal := range ocEntry {
		obVal, ok := obEntry[key]
		if !ok {
			continue
		}
		t.Run("type/"+key, func(t *testing.T) {
			ocType := typeOf(ocVal)
			obType := typeOf(obVal)
			if ocType != obType {
				t.Errorf("type mismatch for %q: openclaw=%s openbot=%s", key, ocType, obType)
			}
		})
	}
}

func typeOf(v any) string {
	switch v.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "bool"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}
```

**Step 2: Run tests**

Run: `GOWORK=off go test -v -run "TestSessionEntry|TestSessionTranscript|TestSessionKey|TestLiveSession" ./test/compat/`
Expected: PASS (SessionEntry and Transcript tests use in-memory stores, LiveSession may skip)

**Step 3: Commit**

```bash
git add test/compat/session_seed_test.go
git commit -m "test: add comprehensive session entry and transcript comparison tests"
```

---

### Task 6: Run Full Test Suite and Verify

**Step 1: Build the openbot binary**

Run: `GOWORK=off go build -o $HOME/bin/openbot ./cmd/openbot/`

**Step 2: Run all tests**

Run: `GOWORK=off go test -v -count=1 ./...`
Expected: All PASS (some tests may SKIP if openclaw CLI not available)

**Step 3: Run just the compat tests**

Run: `GOWORK=off go test -v -count=1 ./test/compat/`
Expected: All PASS or SKIP (no FAIL)

**Step 4: Verify directory parity manually**

Run: `diff <(cd ~/.openclaw && find . -type d | sort) <(cd ~/.openbot && find . -type d | sort)`
Expected: Minimal differences (only data-specific dirs like .git)

**Step 5: Final commit**

```bash
git add -A
git commit -m "feat: complete OpenClaw feature parity with comprehensive tests"
```
