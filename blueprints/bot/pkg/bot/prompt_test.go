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
