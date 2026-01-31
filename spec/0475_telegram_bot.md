# OpenBot: Standalone Telegram Bot Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a standalone Telegram bot binary (`openbot`) that reads OpenClaw-compatible config from `~/.openbot`, connects to Telegram via long-polling, and routes messages through Claude with workspace context, skills, and memory -- matching OpenClaw's system prompt architecture.

**Architecture:** Single Go binary, no HTTP server. Telegram long-poll loop receives messages, checks DM policy (allowlist), manages per-peer sessions in SQLite, builds an enriched system prompt (workspace bootstrap + skills + memory search), calls Claude API, and sends the reply back via Telegram. Config is cloned from `~/.openclaw` on first run.

**Tech Stack:** Go 1.25, existing `blueprints/bot` packages (channel/telegram, memory, skill, workspace, compact, llm, store/sqlite), Telegram Bot API (long-poll), Anthropic Claude API, SQLite (WAL mode).

---

## Background

### What is OpenClaw?

OpenClaw is a personal AI assistant platform (TypeScript) that runs locally and connects to messaging channels (Telegram, Discord, Slack, etc.). Its key innovation is the **workspace context system**: a set of markdown files (SOUL.md, AGENTS.md, TOOLS.md, IDENTITY.md, USER.md, HEARTBEAT.md) that define the bot's personality, behavior rules, and user context. These files are injected into the system prompt on every LLM call.

### What we're building

A lightweight Go reimplementation of OpenClaw's Telegram channel as a standalone binary. It:

1. Reads config from `~/.openbot/openbot.json` (100% compatible with `~/.openclaw/openclaw.json`)
2. Copies workspace bootstrap files from `~/.openclaw/workspace` on first run
3. Connects to Telegram via Bot API long-polling
4. Enforces DM allowlist policy
5. Manages conversation sessions per peer (SQLite)
6. Builds OpenClaw-compatible system prompts with workspace + skills + memory
7. Calls Claude API and returns responses via Telegram

### Config compatibility

The `openbot.json` config file uses the same schema as OpenClaw:

```json
{
  "auth": {
    "profiles": {
      "anthropic:default": { "provider": "anthropic", "mode": "api_key" }
    }
  },
  "agents": {
    "defaults": {
      "workspace": "/Users/apple/.openbot/workspace"
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "botToken": "...",
      "dmPolicy": "allowlist",
      "allowFrom": ["1994676962"],
      "groupPolicy": "allowlist",
      "streamMode": "partial"
    }
  }
}
```

Environment variable overrides: `$TELEGRAM_API_KEY` overrides `channels.telegram.botToken`, `$ANTHROPIC_API_KEY` is used for Claude API.

### Directory layout

```
$HOME/.openbot/
  openbot.json            # Config (cloned from ~/.openclaw/openclaw.json)
  workspace/              # Bootstrap files (cloned from ~/.openclaw/workspace)
    SOUL.md
    AGENTS.md
    TOOLS.md
    IDENTITY.md
    USER.md
    HEARTBEAT.md
    BOOTSTRAP.md
    skills/               # Optional skill directories
  data/
    bot.db                # SQLite database (sessions, messages)
  telegram/
    offset.json           # Last processed Telegram update_id
```

### Code layout (new files in blueprints/bot/)

```
cmd/openbot/main.go       # Entry point
pkg/config/config.go       # Config loading + env var overrides
pkg/config/init.go         # First-run: clone ~/.openclaw -> ~/.openbot
pkg/bot/bot.go             # Core bot loop (wire Telegram -> gateway -> reply)
pkg/bot/prompt.go          # System prompt builder (OpenClaw-compatible)
pkg/bot/bot_test.go        # Tests for bot loop
pkg/config/config_test.go  # Tests for config loading
```

### Reused packages (no changes needed)

- `pkg/channel/telegram/telegram.go` -- Telegram long-poll driver
- `pkg/memory/` -- FTS5 + vector hybrid search
- `pkg/skill/` -- SKILL.md loading and prompt building
- `pkg/workspace/` -- Bootstrap file loading and filtering
- `pkg/compact/` -- Token estimation and context pruning
- `pkg/llm/llm.go` -- Claude and Echo LLM providers
- `store/sqlite/` -- SQLite persistence (sessions, messages)
- `store/store.go` -- Store interface
- `types/types.go` -- Data structures

---

## Tasks

### Task 1: Config package -- loading and env var overrides

**Files:**
- Create: `blueprints/bot/pkg/config/config.go`
- Create: `blueprints/bot/pkg/config/config_test.go`

**Step 1: Write failing test for config loading**

```go
// pkg/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_FromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "openbot.json")
	content := `{
		"channels": {
			"telegram": {
				"enabled": true,
				"botToken": "file-token-123",
				"dmPolicy": "allowlist",
				"allowFrom": ["111", "222"]
			}
		},
		"agents": {
			"defaults": {
				"workspace": "/tmp/ws"
			}
		}
	}`
	os.WriteFile(cfgPath, []byte(content), 0o644)

	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if cfg.Telegram.BotToken != "file-token-123" {
		t.Errorf("expected file-token-123, got %s", cfg.Telegram.BotToken)
	}
	if cfg.Telegram.DMPolicy != "allowlist" {
		t.Errorf("expected allowlist, got %s", cfg.Telegram.DMPolicy)
	}
	if len(cfg.Telegram.AllowFrom) != 2 {
		t.Errorf("expected 2 allowFrom entries, got %d", len(cfg.Telegram.AllowFrom))
	}
	if cfg.Workspace != "/tmp/ws" {
		t.Errorf("expected /tmp/ws, got %s", cfg.Workspace)
	}
}

func TestLoadConfig_EnvVarOverrides(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "openbot.json")
	content := `{
		"channels": {
			"telegram": {
				"enabled": true,
				"botToken": "file-token"
			}
		}
	}`
	os.WriteFile(cfgPath, []byte(content), 0o644)

	t.Setenv("TELEGRAM_API_KEY", "env-token-override")
	t.Setenv("ANTHROPIC_API_KEY", "env-anthropic-key")

	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	// Env var should override file value.
	if cfg.Telegram.BotToken != "env-token-override" {
		t.Errorf("expected env override, got %s", cfg.Telegram.BotToken)
	}
	if cfg.AnthropicKey != "env-anthropic-key" {
		t.Errorf("expected env anthropic key, got %s", cfg.AnthropicKey)
	}
}

func TestLoadConfig_TelegramDisabled(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "openbot.json")
	content := `{
		"channels": {
			"telegram": {
				"enabled": false,
				"botToken": "token"
			}
		}
	}`
	os.WriteFile(cfgPath, []byte(content), 0o644)

	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if cfg.Telegram.Enabled {
		t.Error("expected telegram to be disabled")
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/openbot.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadConfig_DefaultWorkspace(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "openbot.json")
	content := `{
		"channels": {
			"telegram": {
				"enabled": true,
				"botToken": "tok"
			}
		}
	}`
	os.WriteFile(cfgPath, []byte(content), 0o644)

	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	// When workspace is empty, it should default to <configDir>/workspace.
	expected := filepath.Join(dir, "workspace")
	if cfg.Workspace != expected {
		t.Errorf("expected default workspace %s, got %s", expected, cfg.Workspace)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/bot && GOWORK=off go test ./pkg/config/... -v`
Expected: FAIL (package doesn't exist yet)

**Step 3: Implement config loading**

```go
// pkg/config/config.go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the resolved bot configuration.
type Config struct {
	Telegram     TelegramConfig
	Workspace    string
	DataDir      string
	AnthropicKey string
}

// TelegramConfig holds Telegram channel settings (matches OpenClaw schema).
type TelegramConfig struct {
	Enabled     bool     `json:"enabled"`
	BotToken    string   `json:"botToken"`
	DMPolicy    string   `json:"dmPolicy"`
	AllowFrom   []string `json:"allowFrom"`
	GroupPolicy string   `json:"groupPolicy"`
	StreamMode  string   `json:"streamMode"`
}

// configFile represents the raw JSON structure of openbot.json (OpenClaw-compatible).
type configFile struct {
	Channels struct {
		Telegram TelegramConfig `json:"telegram"`
	} `json:"channels"`
	Agents struct {
		Defaults struct {
			Workspace string `json:"workspace"`
		} `json:"defaults"`
	} `json:"agents"`
}

// LoadFromFile reads and parses an OpenClaw-compatible config file.
// Environment variables override file values:
//   - TELEGRAM_API_KEY overrides channels.telegram.botToken
//   - ANTHROPIC_API_KEY sets the Anthropic API key
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var raw configFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg := &Config{
		Telegram:  raw.Channels.Telegram,
		Workspace: raw.Agents.Defaults.Workspace,
	}

	// Default workspace to <configDir>/workspace if not specified.
	if cfg.Workspace == "" {
		cfg.Workspace = filepath.Join(filepath.Dir(path), "workspace")
	}

	// Default data dir to <configDir>/data.
	cfg.DataDir = filepath.Join(filepath.Dir(path), "data")

	// Default DM policy.
	if cfg.Telegram.DMPolicy == "" {
		cfg.Telegram.DMPolicy = "allowlist"
	}

	// Env var overrides.
	if envToken := os.Getenv("TELEGRAM_API_KEY"); envToken != "" {
		cfg.Telegram.BotToken = envToken
	}
	cfg.AnthropicKey = os.Getenv("ANTHROPIC_API_KEY")

	return cfg, nil
}

// DefaultConfigDir returns the default openbot config directory path.
func DefaultConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openbot")
}

// DefaultConfigPath returns the default openbot.json path.
func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir(), "openbot.json")
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/bot && GOWORK=off go test ./pkg/config/... -v`
Expected: PASS (all 5 tests)

**Step 5: Commit**

```bash
git add blueprints/bot/pkg/config/
git commit -m "feat(openbot): add config package with OpenClaw-compatible loading"
```

---

### Task 2: Config init -- clone from OpenClaw on first run

**Files:**
- Create: `blueprints/bot/pkg/config/init.go`
- Add tests to: `blueprints/bot/pkg/config/config_test.go`

**Step 1: Write failing test for init logic**

Add to `config_test.go`:

```go
func TestEnsureConfig_ClonesFromOpenClaw(t *testing.T) {
	// Simulate ~/.openclaw with config and workspace.
	openclawDir := t.TempDir()
	openclawCfg := `{
		"channels": {
			"telegram": {
				"enabled": true,
				"botToken": "original-token",
				"allowFrom": ["999"]
			}
		},
		"agents": {
			"defaults": {
				"workspace": "` + openclawDir + `/workspace"
			}
		}
	}`
	os.WriteFile(filepath.Join(openclawDir, "openclaw.json"), []byte(openclawCfg), 0o644)

	// Create workspace files.
	wsDir := filepath.Join(openclawDir, "workspace")
	os.MkdirAll(wsDir, 0o755)
	os.WriteFile(filepath.Join(wsDir, "SOUL.md"), []byte("# Soul\nTest soul"), 0o644)
	os.WriteFile(filepath.Join(wsDir, "AGENTS.md"), []byte("# Agents\nTest agents"), 0o644)

	// Target openbot dir (should not exist yet).
	openbotDir := filepath.Join(t.TempDir(), ".openbot")

	err := EnsureConfig(openbotDir, openclawDir)
	if err != nil {
		t.Fatalf("EnsureConfig: %v", err)
	}

	// Config file should exist.
	if _, err := os.Stat(filepath.Join(openbotDir, "openbot.json")); err != nil {
		t.Errorf("expected openbot.json to exist: %v", err)
	}

	// Workspace files should be copied.
	soul, err := os.ReadFile(filepath.Join(openbotDir, "workspace", "SOUL.md"))
	if err != nil {
		t.Fatalf("read SOUL.md: %v", err)
	}
	if string(soul) != "# Soul\nTest soul" {
		t.Errorf("SOUL.md content mismatch: %s", soul)
	}
}

func TestEnsureConfig_SkipsIfExists(t *testing.T) {
	openbotDir := t.TempDir()
	// Pre-create config.
	os.WriteFile(filepath.Join(openbotDir, "openbot.json"), []byte(`{"channels":{}}`), 0o644)

	// Should not error even if openclaw dir doesn't exist.
	err := EnsureConfig(openbotDir, "/nonexistent/openclaw")
	if err != nil {
		t.Fatalf("EnsureConfig should skip existing: %v", err)
	}
}

func TestEnsureConfig_NoOpenClaw(t *testing.T) {
	openbotDir := filepath.Join(t.TempDir(), ".openbot")

	err := EnsureConfig(openbotDir, "/nonexistent/openclaw")
	if err == nil {
		t.Error("expected error when no source exists")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/bot && GOWORK=off go test ./pkg/config/... -v -run TestEnsureConfig`
Expected: FAIL (function doesn't exist)

**Step 3: Implement init logic**

```go
// pkg/config/init.go
package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// EnsureConfig ensures ~/.openbot exists with a valid config.
// If openbotDir doesn't exist, it clones from openclawDir.
// If openbotDir already has openbot.json, it does nothing.
func EnsureConfig(openbotDir, openclawDir string) error {
	cfgPath := filepath.Join(openbotDir, "openbot.json")

	// If config already exists, nothing to do.
	if _, err := os.Stat(cfgPath); err == nil {
		return nil
	}

	// Check that source (openclaw) exists.
	srcCfgPath := filepath.Join(openclawDir, "openclaw.json")
	if _, err := os.Stat(srcCfgPath); err != nil {
		return fmt.Errorf("no existing config and no openclaw source at %s: %w", openclawDir, err)
	}

	// Create openbot directory.
	if err := os.MkdirAll(openbotDir, 0o700); err != nil {
		return fmt.Errorf("create openbot dir: %w", err)
	}

	// Copy and rewrite config.
	if err := cloneConfig(srcCfgPath, cfgPath, openbotDir); err != nil {
		return fmt.Errorf("clone config: %w", err)
	}

	// Copy workspace files.
	srcWs := filepath.Join(openclawDir, "workspace")
	dstWs := filepath.Join(openbotDir, "workspace")
	if err := copyDir(srcWs, dstWs); err != nil {
		return fmt.Errorf("clone workspace: %w", err)
	}

	// Create data directory.
	if err := os.MkdirAll(filepath.Join(openbotDir, "data"), 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	return nil
}

// cloneConfig copies the openclaw config and rewrites workspace paths.
func cloneConfig(src, dst, openbotDir string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Parse, rewrite workspace path, and re-serialize.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Rewrite agents.defaults.workspace to point to openbot workspace.
	if agents, ok := raw["agents"].(map[string]any); ok {
		if defaults, ok := agents["defaults"].(map[string]any); ok {
			if ws, ok := defaults["workspace"].(string); ok && ws != "" {
				if strings.Contains(ws, "openclaw") {
					defaults["workspace"] = filepath.Join(openbotDir, "workspace")
				}
			}
		}
	}

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(dst, out, 0o600)
}

// copyDir copies a directory tree, skipping .git.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // source doesn't exist, skip
		}
		return err
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}

		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/bot && GOWORK=off go test ./pkg/config/... -v`
Expected: PASS (all tests)

**Step 5: Commit**

```bash
git add blueprints/bot/pkg/config/init.go
git commit -m "feat(openbot): add first-run config cloning from OpenClaw"
```

---

### Task 3: Bot prompt builder -- OpenClaw-compatible system prompt

**Files:**
- Create: `blueprints/bot/pkg/bot/prompt.go`
- Create: `blueprints/bot/pkg/bot/prompt_test.go`

**Step 1: Write failing test**

```go
// pkg/bot/prompt_test.go
package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/workspace"
)

func setupTestWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	workspace.EnsureWorkspace(dir)

	// Write richer SOUL.md.
	os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte("# SOUL.md\nBe helpful and concise.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "IDENTITY.md"), []byte("# IDENTITY.md\n- Name: TestBot\n"), 0o644)

	return dir
}

func TestBuildSystemPrompt_ContainsIdentity(t *testing.T) {
	ws := setupTestWorkspace(t)
	pb := NewPromptBuilder(ws)

	prompt := pb.Build("dm", "")
	if !strings.Contains(prompt, "You are a personal assistant") {
		t.Error("prompt should contain identity section")
	}
}

func TestBuildSystemPrompt_ContainsWorkspaceFiles(t *testing.T) {
	ws := setupTestWorkspace(t)
	pb := NewPromptBuilder(ws)

	prompt := pb.Build("dm", "")
	if !strings.Contains(prompt, "SOUL.md") {
		t.Error("prompt should contain SOUL.md section")
	}
	if !strings.Contains(prompt, "Be helpful and concise") {
		t.Error("prompt should contain SOUL.md content")
	}
}

func TestBuildSystemPrompt_ContainsRuntimeInfo(t *testing.T) {
	ws := setupTestWorkspace(t)
	pb := NewPromptBuilder(ws)

	prompt := pb.Build("dm", "")
	if !strings.Contains(prompt, "Current Date") {
		t.Error("prompt should contain current date section")
	}
}

func TestBuildSystemPrompt_DMOrigin_IncludesAllFiles(t *testing.T) {
	ws := setupTestWorkspace(t)
	pb := NewPromptBuilder(ws)

	prompt := pb.Build("dm", "")
	if !strings.Contains(prompt, "AGENTS.md") {
		t.Error("dm prompt should contain AGENTS.md")
	}
	if !strings.Contains(prompt, "SOUL.md") {
		t.Error("dm prompt should contain SOUL.md")
	}
}

func TestBuildSystemPrompt_EmptyWorkspace(t *testing.T) {
	pb := NewPromptBuilder("")

	prompt := pb.Build("dm", "")
	// Should still contain identity section even without workspace.
	if !strings.Contains(prompt, "You are a personal assistant") {
		t.Error("prompt should contain identity even without workspace")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/bot && GOWORK=off go test ./pkg/bot/... -v`
Expected: FAIL (package doesn't exist)

**Step 3: Implement prompt builder**

```go
// pkg/bot/prompt.go
package bot

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/workspace"
)

// PromptBuilder constructs OpenClaw-compatible system prompts.
type PromptBuilder struct {
	workspaceDir string
}

// NewPromptBuilder creates a prompt builder for the given workspace.
func NewPromptBuilder(workspaceDir string) *PromptBuilder {
	return &PromptBuilder{workspaceDir: workspaceDir}
}

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

	// 3. Skills.
	if pb.workspaceDir != "" {
		if sk := pb.skillsSection(); sk != "" {
			sections = append(sections, sk)
		}
	}

	// 4. Runtime info.
	sections = append(sections, pb.runtimeSection())

	return strings.Join(sections, "\n\n")
}

// identitySection returns the base identity prompt matching OpenClaw's style.
func (pb *PromptBuilder) identitySection() string {
	return `You are a personal assistant running inside OpenBot.
You help your human through Telegram.
Be genuinely helpful, not performatively helpful.
Have opinions. Be resourceful before asking.`
}

// workspaceSection loads and formats bootstrap files.
func (pb *PromptBuilder) workspaceSection(origin string) string {
	files, err := workspace.LoadBootstrapFiles(pb.workspaceDir)
	if err != nil {
		return ""
	}

	var filtered []workspace.BootstrapFile
	if origin == "subagent" {
		filtered = workspace.FilterForSubagent(files)
	} else {
		filtered = workspace.FilterForMain(files)
	}

	if len(filtered) == 0 {
		return ""
	}

	return workspace.BuildContextPrompt(filtered)
}

// skillsSection loads skills and formats the prompt.
func (pb *PromptBuilder) skillsSection() string {
	skills, err := skill.LoadAllSkills(pb.workspaceDir)
	if err != nil || len(skills) == 0 {
		return ""
	}

	hasReady := false
	for _, s := range skills {
		if s.Ready {
			hasReady = true
			break
		}
	}
	if !hasReady {
		return ""
	}

	return skill.BuildSkillsPrompt(skills)
}

// runtimeSection returns current date/time and runtime info.
func (pb *PromptBuilder) runtimeSection() string {
	now := time.Now()
	return fmt.Sprintf(`## Current Date & Time
%s

## Runtime
host=%s os=%s arch=%s`,
		now.Format("Monday, January 2, 2006 3:04 PM MST"),
		"openbot",
		runtime.GOOS,
		runtime.GOARCH,
	)
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/bot && GOWORK=off go test ./pkg/bot/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add blueprints/bot/pkg/bot/
git commit -m "feat(openbot): add OpenClaw-compatible system prompt builder"
```

---

### Task 4: Bot core -- message loop wiring Telegram to Claude

**Files:**
- Create: `blueprints/bot/pkg/bot/bot.go`
- Create: `blueprints/bot/pkg/bot/bot_test.go`

**Step 1: Write failing test for bot core**

```go
// pkg/bot/bot_test.go
package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func TestBot_HandleMessage_BasicFlow(t *testing.T) {
	ws := setupTestWorkspace(t)
	cfg := &config.Config{
		Workspace: ws,
		Telegram: config.TelegramConfig{
			Enabled:  true,
			BotToken: "test-token",
			DMPolicy: "open",
		},
	}

	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	msg := &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		PeerID:      "user-1",
		PeerName:    "Test",
		Content:     "Hello bot!",
		Origin:      "dm",
	}

	resp, err := b.HandleMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	if !strings.Contains(resp, "[Echo]") {
		t.Errorf("expected echo response, got: %s", resp)
	}
	if !strings.Contains(resp, "Hello bot!") {
		t.Errorf("expected message echoed, got: %s", resp)
	}
}

func TestBot_HandleMessage_Allowlist_Allowed(t *testing.T) {
	ws := setupTestWorkspace(t)
	cfg := &config.Config{
		Workspace: ws,
		Telegram: config.TelegramConfig{
			Enabled:   true,
			BotToken:  "test-token",
			DMPolicy:  "allowlist",
			AllowFrom: []string{"user-1"},
		},
	}

	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	msg := &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		PeerID:      "user-1",
		PeerName:    "Allowed",
		Content:     "Hi",
		Origin:      "dm",
	}

	_, err = b.HandleMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("allowed user should succeed: %v", err)
	}
}

func TestBot_HandleMessage_Allowlist_Blocked(t *testing.T) {
	ws := setupTestWorkspace(t)
	cfg := &config.Config{
		Workspace: ws,
		Telegram: config.TelegramConfig{
			Enabled:   true,
			BotToken:  "test-token",
			DMPolicy:  "allowlist",
			AllowFrom: []string{"user-1"},
		},
	}

	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	msg := &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		PeerID:      "user-999",
		PeerName:    "Stranger",
		Content:     "Hi",
		Origin:      "dm",
	}

	_, err = b.HandleMessage(context.Background(), msg)
	if err == nil {
		t.Error("expected error for blocked user")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("expected 'not allowed' error, got: %v", err)
	}
}

func TestBot_HandleMessage_SessionPersistence(t *testing.T) {
	ws := setupTestWorkspace(t)
	cfg := &config.Config{
		Workspace: ws,
		Telegram: config.TelegramConfig{
			Enabled:  true,
			BotToken: "test-token",
			DMPolicy: "open",
		},
	}

	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	msg := &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		PeerID:      "user-1",
		PeerName:    "Test",
		Content:     "First",
		Origin:      "dm",
	}

	// Send two messages -- should reuse session.
	_, err = b.HandleMessage(ctx, msg)
	if err != nil {
		t.Fatalf("first: %v", err)
	}

	msg.Content = "Second"
	_, err = b.HandleMessage(ctx, msg)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
}

func TestBot_HandleMessage_SlashCommand(t *testing.T) {
	ws := setupTestWorkspace(t)
	cfg := &config.Config{
		Workspace: ws,
		Telegram: config.TelegramConfig{
			Enabled:  true,
			BotToken: "test-token",
			DMPolicy: "open",
		},
	}

	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	msg := &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		PeerID:      "user-1",
		PeerName:    "Test",
		Content:     "/help",
		Origin:      "dm",
	}

	resp, err := b.HandleMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	if !strings.Contains(resp, "Available commands") {
		t.Errorf("expected help listing, got: %s", resp)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/bot && GOWORK=off go test ./pkg/bot/... -v`
Expected: FAIL (Bot type doesn't exist yet)

**Step 3: Implement bot core**

```go
// pkg/bot/bot.go
package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-mizu/mizu/blueprints/bot/feature/command"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/compact"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/memory"
	"github.com/go-mizu/mizu/blueprints/bot/store"
	"github.com/go-mizu/mizu/blueprints/bot/store/sqlite"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// DefaultContextWindow is the context window budget for Claude.
const DefaultContextWindow = 200000

// Bot is the standalone Telegram bot engine.
// It wires message handling, session management, context building, and LLM calls.
type Bot struct {
	cfg      *config.Config
	store    store.Store
	llm      llm.Provider
	commands *command.Service
	prompt   *PromptBuilder
	memory   *memory.MemoryManager
	allowSet map[string]bool
}

// New creates a Bot using an in-memory SQLite store for the given config.
func New(cfg *config.Config, provider llm.Provider) (*Bot, error) {
	// Ensure data dir exists.
	if cfg.DataDir == "" {
		cfg.DataDir = filepath.Join(config.DefaultConfigDir(), "data")
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(cfg.DataDir, "bot.db")
	s, err := sqlite.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := s.Ensure(context.Background()); err != nil {
		s.Close()
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	// Create a default agent if none exists.
	if err := ensureDefaultAgent(s, cfg); err != nil {
		s.Close()
		return nil, fmt.Errorf("ensure agent: %w", err)
	}

	// Build allowlist set.
	allowSet := make(map[string]bool, len(cfg.Telegram.AllowFrom))
	for _, id := range cfg.Telegram.AllowFrom {
		allowSet[id] = true
	}

	// Initialize memory manager if workspace exists.
	var mem *memory.MemoryManager
	if cfg.Workspace != "" {
		if _, err := os.Stat(cfg.Workspace); err == nil {
			mem, _ = memory.New(cfg.Workspace, memory.DefaultMemoryConfig())
		}
	}

	return &Bot{
		cfg:      cfg,
		store:    s,
		llm:      provider,
		commands: command.NewService(),
		prompt:   NewPromptBuilder(cfg.Workspace),
		memory:   mem,
		allowSet: allowSet,
	}, nil
}

// Close releases all resources.
func (b *Bot) Close() {
	if b.memory != nil {
		b.memory.Close()
	}
	if b.store != nil {
		b.store.Close()
	}
}

// HandleMessage processes an inbound message end-to-end.
func (b *Bot) HandleMessage(ctx context.Context, msg *types.InboundMessage) (string, error) {
	// 1. Check DM policy.
	if err := b.checkPolicy(msg); err != nil {
		return "", err
	}

	// 2. Resolve agent (always the default agent).
	agent, err := b.store.ResolveAgent(ctx, string(msg.ChannelType), msg.ChannelID, msg.PeerID)
	if err != nil {
		return "", fmt.Errorf("resolve agent: %w", err)
	}

	// 3. Get or create session.
	session, err := b.store.GetOrCreateSession(ctx,
		agent.ID, msg.ChannelID, string(msg.ChannelType),
		msg.PeerID, msg.PeerName, msg.Origin)
	if err != nil {
		return "", fmt.Errorf("get/create session: %w", err)
	}

	// 4. Check for slash commands.
	cmd, args, isCommand := b.commands.Parse(msg.Content)
	if isCommand {
		return b.handleCommand(ctx, cmd, args, agent, session), nil
	}

	// 5. Store user message.
	userMsg := &types.Message{
		SessionID: session.ID,
		AgentID:   agent.ID,
		ChannelID: msg.ChannelID,
		PeerID:    msg.PeerID,
		Role:      types.RoleUser,
		Content:   msg.Content,
	}
	if err := b.store.CreateMessage(ctx, userMsg); err != nil {
		return "", fmt.Errorf("store user message: %w", err)
	}

	// 6. Build conversation history.
	history, err := b.store.ListMessages(ctx, session.ID, 50)
	if err != nil {
		return "", fmt.Errorf("list messages: %w", err)
	}

	llmMessages := make([]types.LLMMsg, len(history))
	for i, m := range history {
		llmMessages[i] = types.LLMMsg{Role: m.Role, Content: m.Content}
	}

	// 7. Apply context pruning.
	totalTokens := compact.EstimateMessagesTokens(llmMessages)
	llmMessages = compact.PruneMessages(llmMessages, totalTokens, DefaultContextWindow, compact.DefaultPruneConfig())

	totalTokens = compact.EstimateMessagesTokens(llmMessages)
	historyBudget := float64(DefaultContextWindow-compact.DefaultReserveTokensFloor) / float64(DefaultContextWindow)
	pruneResult := compact.PruneHistoryForContextShare(llmMessages, DefaultContextWindow, historyBudget)
	llmMessages = pruneResult.Messages

	// 8. Build system prompt.
	systemPrompt := b.prompt.Build(msg.Origin, msg.Content)

	// 9. Add memory search results if available.
	if b.memory != nil && msg.Content != "" {
		results, err := b.memory.Search(ctx, msg.Content, 0, 0)
		if err == nil && len(results) > 0 {
			systemPrompt += "\n\n" + formatMemoryResults(results)
		}
	}

	// 10. Call LLM.
	llmReq := &types.LLMRequest{
		Model:        agent.Model,
		SystemPrompt: systemPrompt,
		Messages:     llmMessages,
		MaxTokens:    agent.MaxTokens,
		Temperature:  agent.Temperature,
	}

	llmResp, err := b.llm.Chat(ctx, llmReq)
	if err != nil {
		log.Printf("LLM error: %v", err)
		return "", fmt.Errorf("LLM chat: %w", err)
	}

	// 11. Store assistant response.
	assistantMsg := &types.Message{
		SessionID: session.ID,
		AgentID:   agent.ID,
		ChannelID: msg.ChannelID,
		PeerID:    msg.PeerID,
		Role:      types.RoleAssistant,
		Content:   llmResp.Content,
	}
	if err := b.store.CreateMessage(ctx, assistantMsg); err != nil {
		return "", fmt.Errorf("store assistant message: %w", err)
	}

	return llmResp.Content, nil
}

// checkPolicy enforces DM allowlist.
func (b *Bot) checkPolicy(msg *types.InboundMessage) error {
	if b.cfg.Telegram.DMPolicy != "allowlist" {
		return nil
	}
	if len(b.allowSet) == 0 {
		return nil // empty allowlist = allow all (match OpenClaw behavior)
	}
	if !b.allowSet[msg.PeerID] {
		return fmt.Errorf("peer %s not allowed by DM policy", msg.PeerID)
	}
	return nil
}

func (b *Bot) handleCommand(ctx context.Context, cmd, args string, agent *types.Agent, session *types.Session) string {
	switch cmd {
	case "/new", "/reset":
		session.Status = "expired"
		b.store.UpdateSession(ctx, session)
		return b.commands.Execute(cmd, args, agent)
	case "/context":
		prompt := b.prompt.Build(session.Origin, "")
		if prompt == "" {
			return "No system prompt configured."
		}
		return fmt.Sprintf("System prompt:\n%s", prompt)
	default:
		return b.commands.Execute(cmd, args, agent)
	}
}

// ensureDefaultAgent creates a default agent with wildcard binding if none exists.
func ensureDefaultAgent(s store.Store, cfg *config.Config) error {
	ctx := context.Background()
	agents, err := s.ListAgents(ctx)
	if err != nil {
		return err
	}
	if len(agents) > 0 {
		return nil // agent already exists
	}

	agent := &types.Agent{
		ID:           "default",
		Name:         "OpenBot",
		Model:        "claude-sonnet-4-20250514",
		SystemPrompt: "",
		Workspace:    cfg.Workspace,
		MaxTokens:    4096,
		Temperature:  0.7,
		Status:       "active",
	}
	if err := s.CreateAgent(ctx, agent); err != nil {
		return err
	}

	// Create wildcard binding so all messages route to this agent.
	binding := &types.Binding{
		ID:          "default-binding",
		AgentID:     "default",
		ChannelType: "*",
		ChannelID:   "*",
		PeerID:      "*",
		Priority:    0,
	}
	return s.CreateBinding(ctx, binding)
}

// formatMemoryResults formats search results for system prompt injection.
func formatMemoryResults(results []memory.SearchResult) string {
	var b strings.Builder
	b.WriteString("# Relevant Context\n\n")
	for i, r := range results {
		b.WriteString(fmt.Sprintf("## [%d] %s (lines %d-%d, score: %.2f)\n",
			i+1, r.Path, r.StartLine, r.EndLine, r.Score))
		b.WriteString("```\n")
		b.WriteString(r.Snippet)
		if !strings.HasSuffix(r.Snippet, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
	}
	return b.String()
}
```

Note: bot.go needs an import of `"strings"` for `formatMemoryResults`. The full import list should include it.

**Step 4: Run tests to verify they pass**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/bot && GOWORK=off go test ./pkg/bot/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add blueprints/bot/pkg/bot/bot.go blueprints/bot/pkg/bot/bot_test.go
git commit -m "feat(openbot): add core bot engine with session management and LLM integration"
```

---

### Task 5: Entry point -- cmd/openbot/main.go

**Files:**
- Create: `blueprints/bot/cmd/openbot/main.go`

**Step 1: Implement entry point**

```go
// cmd/openbot/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/bot"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/channel/telegram"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 1. Ensure config exists (clone from OpenClaw if needed).
	openbotDir := config.DefaultConfigDir()
	openclawDir := filepath.Join(os.Getenv("HOME"), ".openclaw")

	if err := config.EnsureConfig(openbotDir, openclawDir); err != nil {
		log.Printf("Config init: %v", err)
		log.Printf("Create %s/openbot.json manually or install OpenClaw first.", openbotDir)
		os.Exit(1)
	}

	// 2. Load config.
	cfg, err := config.LoadFromFile(config.DefaultConfigPath())
	if err != nil {
		log.Fatalf("Load config: %v", err)
	}

	if !cfg.Telegram.Enabled {
		log.Fatal("Telegram channel is disabled in config")
	}
	if cfg.Telegram.BotToken == "" {
		log.Fatal("No Telegram bot token. Set TELEGRAM_API_KEY or configure channels.telegram.botToken")
	}

	// 3. Create LLM provider.
	provider := llm.NewClaude()

	// 4. Create bot engine.
	b, err := bot.New(cfg, provider)
	if err != nil {
		log.Fatalf("Create bot: %v", err)
	}
	defer b.Close()

	// 5. Create Telegram driver.
	telegramCfg := types.TelegramConfig{
		BotToken: cfg.Telegram.BotToken,
	}
	cfgJSON, _ := json.Marshal(telegramCfg)

	handler := func(ctx context.Context, msg *types.InboundMessage) error {
		resp, err := b.HandleMessage(ctx, msg)
		if err != nil {
			log.Printf("Handle message from %s: %v", msg.PeerName, err)
			return nil // don't crash on message errors
		}

		// Send response back via Telegram.
		outMsg := &types.OutboundMessage{
			ChannelType: types.ChannelTelegram,
			PeerID:      msg.PeerID,
			Content:     resp,
		}
		if err := tgDriver.Send(ctx, outMsg); err != nil {
			log.Printf("Send to %s: %v", msg.PeerName, err)
		}
		return nil
	}

	drv, err := telegram.NewDriver(string(cfgJSON), handler)
	if err != nil {
		log.Fatalf("Create Telegram driver: %v", err)
	}
	tgDriver = drv

	// 6. Start polling.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("OpenBot starting...")
	fmt.Printf("  Workspace: %s\n", cfg.Workspace)
	fmt.Printf("  DM Policy: %s\n", cfg.Telegram.DMPolicy)
	if len(cfg.Telegram.AllowFrom) > 0 {
		fmt.Printf("  Allow From: %v\n", cfg.Telegram.AllowFrom)
	}
	fmt.Println("  Connecting to Telegram...")

	if err := drv.Connect(ctx); err != nil {
		log.Fatalf("Connect Telegram: %v", err)
	}
	fmt.Println("  Connected! Listening for messages...")

	// 7. Wait for shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh
	fmt.Println("\nShutting down...")
	cancel()
	drv.Disconnect(context.Background())
	fmt.Println("OpenBot stopped.")
}

// tgDriver is set after creation so the handler closure can reference it.
var tgDriver interface {
	Send(ctx context.Context, msg *types.OutboundMessage) error
}
```

Note: The Telegram driver currently uses `init()` to register in the channel registry. For the standalone binary, we need a direct constructor. We will need to **export a constructor** from the telegram package. See Task 6.

**Step 2: Commit**

```bash
git add blueprints/bot/cmd/openbot/
git commit -m "feat(openbot): add standalone entry point"
```

---

### Task 6: Export Telegram driver constructor

The existing `telegram.Driver` is only created through the `channel.Register` init pattern. For direct use in `cmd/openbot`, we need an exported constructor.

**Files:**
- Modify: `blueprints/bot/pkg/channel/telegram/telegram.go`

**Step 1: Add exported constructor**

Add after the `init()` function:

```go
// NewDriver creates a Telegram driver directly (for standalone use).
func NewDriver(config string, handler channel.MessageHandler) (*Driver, error) {
	var cfg types.TelegramConfig
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		return nil, fmt.Errorf("parse telegram config: %w", err)
	}
	return &Driver{config: cfg, handler: handler, client: &http.Client{Timeout: 30 * time.Second}}, nil
}
```

**Step 2: Run existing tests to verify no regressions**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/bot && GOWORK=off go test ./... -v`
Expected: All existing tests PASS

**Step 3: Commit**

```bash
git add blueprints/bot/pkg/channel/telegram/telegram.go
git commit -m "feat(openbot): export Telegram driver constructor for standalone use"
```

---

### Task 7: Update Makefile for openbot binary

**Files:**
- Modify: `blueprints/bot/Makefile`

**Step 1: Add openbot targets**

Add these targets to the Makefile:

```makefile
OPENBOT_BINARY ?= $(HOME)/bin/openbot
OPENBOT_CMD    ?= ./cmd/openbot

.PHONY: openbot
openbot: ## Build openbot binary to $$HOME/bin/openbot
	@mkdir -p $(dir $(OPENBOT_BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(OPENBOT_BINARY) $(OPENBOT_CMD)
	@echo "Built: $(OPENBOT_BINARY)"

.PHONY: run-openbot
run-openbot: ## Run openbot (standalone Telegram bot)
	@GOWORK=off $(GO) run $(OPENBOT_CMD)
```

**Step 2: Commit**

```bash
git add blueprints/bot/Makefile
git commit -m "feat(openbot): add Makefile targets for openbot binary"
```

---

### Task 8: Integration test -- full message round-trip

**Files:**
- Add to: `blueprints/bot/pkg/bot/bot_test.go`

**Step 1: Write integration test for full round-trip**

```go
func TestBot_FullRoundTrip_WithMemoryAndContext(t *testing.T) {
	ws := setupTestWorkspace(t)

	// Add a skill.
	skillDir := filepath.Join(ws, "skills", "greet")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: greet
description: Greeting skill
---
# Greet Skill
Say hello nicely.
`), 0o644)

	// Add indexable content.
	os.WriteFile(filepath.Join(ws, "notes.md"), []byte("# Notes\nThe project uses Go and SQLite.\n"), 0o644)

	cfg := &config.Config{
		Workspace: ws,
		Telegram: config.TelegramConfig{
			Enabled:  true,
			BotToken: "test-token",
			DMPolicy: "open",
		},
	}

	var capturedPrompt string
	captureLLM := &capturingLLM{
		onChat: func(req *types.LLMRequest) {
			capturedPrompt = req.SystemPrompt
		},
	}

	b, err := New(cfg, captureLLM)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	msg := &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		PeerID:      "user-1",
		PeerName:    "Tester",
		Content:     "Tell me about Go and SQLite",
		Origin:      "dm",
	}

	resp, err := b.HandleMessage(ctx, msg)
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	// Response should exist.
	if resp == "" {
		t.Error("expected non-empty response")
	}

	// System prompt should contain identity.
	if !strings.Contains(capturedPrompt, "personal assistant") {
		t.Error("prompt should contain identity")
	}

	// System prompt should contain workspace context.
	if !strings.Contains(capturedPrompt, "SOUL.md") {
		t.Error("prompt should contain SOUL.md")
	}

	// System prompt should contain runtime info.
	if !strings.Contains(capturedPrompt, "Current Date") {
		t.Error("prompt should contain runtime date")
	}
}

// capturingLLM captures the request and delegates to Echo.
type capturingLLM struct {
	onChat func(req *types.LLMRequest)
}

func (c *capturingLLM) Chat(ctx context.Context, req *types.LLMRequest) (*types.LLMResponse, error) {
	if c.onChat != nil {
		c.onChat(req)
	}
	return (&llm.Echo{}).Chat(ctx, req)
}
```

**Step 2: Run all tests**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/bot && GOWORK=off go test ./... -v`
Expected: All tests PASS

**Step 3: Commit**

```bash
git add blueprints/bot/pkg/bot/bot_test.go
git commit -m "test(openbot): add integration test for full message round-trip"
```

---

### Task 9: Manual Telegram test

**Step 1: Build the binary**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/bot && make openbot
```

**Step 2: Run with real Telegram token**

```bash
TELEGRAM_API_KEY="<token-from-openclaw-config>" ANTHROPIC_API_KEY="<key>" openbot
```

**Step 3: Send a test message in Telegram**

- Open Telegram, find your bot
- Send "Hello"
- Verify you get a Claude response
- Send "/help" and verify you get the commands list
- Send "/new" and verify session reset

**Step 4: Verify workspace context**

- Send "/context" and verify the system prompt includes SOUL.md, AGENTS.md content

---

## Summary

| Task | What | Files |
|------|------|-------|
| 1 | Config loading + env overrides | `pkg/config/config.go`, `pkg/config/config_test.go` |
| 2 | First-run clone from OpenClaw | `pkg/config/init.go`, tests in `config_test.go` |
| 3 | OpenClaw-compatible prompt builder | `pkg/bot/prompt.go`, `pkg/bot/prompt_test.go` |
| 4 | Core bot engine | `pkg/bot/bot.go`, `pkg/bot/bot_test.go` |
| 5 | Entry point binary | `cmd/openbot/main.go` |
| 6 | Export Telegram constructor | `pkg/channel/telegram/telegram.go` (modify) |
| 7 | Makefile targets | `Makefile` (modify) |
| 8 | Integration test | `pkg/bot/bot_test.go` (add) |
| 9 | Manual Telegram test | (manual) |
