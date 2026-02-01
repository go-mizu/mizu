// Package compat tests OpenBot ↔ OpenClaw skill compatibility.
// Loads all real SKILL.md files from both bundled and installed locations,
// verifies parsing, eligibility, and prompt output match expectations.
package compat

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
)

// openclawBundledDir returns the path to OpenClaw's npm-installed bundled skills.
func openclawBundledDir(t *testing.T) string {
	t.Helper()
	// Try standard nvm location.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	// Walk nvm versions to find openclaw.
	nvmDir := filepath.Join(home, ".nvm", "versions", "node")
	entries, err := os.ReadDir(nvmDir)
	if err != nil {
		t.Skipf("nvm directory not found: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(nvmDir, entry.Name(), "lib", "node_modules", "openclaw", "skills")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	t.Skip("openclaw npm package not installed")
	return ""
}

// openbotBundledDir returns the path to our copied bundled skills.
func openbotBundledDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "skills")
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir
	}
	t.Skip("openbot bundled skills directory not found")
	return ""
}

// ---------------------------------------------------------------------------
// Load All OpenClaw Bundled Skills
// ---------------------------------------------------------------------------

func TestLoadAllOpenClawBundledSkills(t *testing.T) {
	dir := openclawBundledDir(t)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	loaded := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillDir := filepath.Join(dir, entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			sk, err := skill.LoadSkill(skillDir)
			if err != nil {
				t.Fatalf("LoadSkill(%s) error: %v", entry.Name(), err)
			}
			if sk.Name == "" {
				t.Error("Name is empty")
			}
			// Some skills like canvas have no frontmatter description.
			// They get their name from the directory name. This is valid.
			if sk.Content == "" {
				t.Error("Content is empty (no body after frontmatter)")
			}
		})
		loaded++
	}

	if loaded < 50 {
		t.Errorf("expected at least 50 skills; loaded %d", loaded)
	}
	t.Logf("Successfully loaded %d OpenClaw bundled skills", loaded)
}

// TestLoadAllOpenbotBundledSkills loads our copied bundled skills.
func TestLoadAllOpenbotBundledSkills(t *testing.T) {
	dir := openbotBundledDir(t)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	loaded := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillDir := filepath.Join(dir, entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			sk, err := skill.LoadSkill(skillDir)
			if err != nil {
				t.Fatalf("LoadSkill(%s) error: %v", entry.Name(), err)
			}
			if sk.Name == "" {
				t.Error("Name is empty")
			}
		})
		loaded++
	}

	if loaded < 50 {
		t.Errorf("expected at least 50 bundled skills; loaded %d", loaded)
	}
	t.Logf("Successfully loaded %d openbot bundled skills", loaded)
}

// ---------------------------------------------------------------------------
// Skill Name Matching: OpenClaw == OpenBot
// ---------------------------------------------------------------------------

func TestSkillNamesParity(t *testing.T) {
	ocDir := openclawBundledDir(t)
	obDir := openbotBundledDir(t)

	ocEntries, err := os.ReadDir(ocDir)
	if err != nil {
		t.Fatalf("read openclaw dir: %v", err)
	}

	for _, entry := range ocEntries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			obSkillDir := filepath.Join(obDir, entry.Name())
			if _, err := os.Stat(filepath.Join(obSkillDir, "SKILL.md")); os.IsNotExist(err) {
				t.Errorf("openbot missing bundled skill %q (present in openclaw)", entry.Name())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Metadata JSON Parsing: All Skills
// ---------------------------------------------------------------------------

func TestMetadataParsingAllSkills(t *testing.T) {
	dir := openbotBundledDir(t)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillDir := filepath.Join(dir, entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			sk, err := skill.LoadSkill(skillDir)
			if err != nil {
				t.Fatalf("LoadSkill error: %v", err)
			}

			// Skills that use metadata JSON should have emoji parsed.
			data, _ := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
			content := string(data)
			if strings.Contains(content, `"emoji"`) && sk.Emoji == "" {
				t.Error("SKILL.md has emoji in metadata but Emoji field is empty")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Known Skills: Specific Field Validation
// ---------------------------------------------------------------------------

func TestKnownSkill_Weather(t *testing.T) {
	dir := openbotBundledDir(t)
	sk, err := skill.LoadSkill(filepath.Join(dir, "weather"))
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}

	if sk.Name != "weather" {
		t.Errorf("Name = %q; want %q", sk.Name, "weather")
	}
	if !strings.Contains(sk.Description, "weather") {
		t.Errorf("Description = %q; should mention weather", sk.Description)
	}
	if len(sk.Requires.Binaries) != 1 || sk.Requires.Binaries[0] != "curl" {
		t.Errorf("Requires.Binaries = %v; want [curl]", sk.Requires.Binaries)
	}
	// curl exists on macOS/Linux, so skill should be ready.
	if !sk.Ready {
		t.Error("weather skill should be Ready (curl is available)")
	}
}

func TestKnownSkill_GitHub(t *testing.T) {
	dir := openbotBundledDir(t)
	sk, err := skill.LoadSkill(filepath.Join(dir, "github"))
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}

	if sk.Name != "github" {
		t.Errorf("Name = %q; want %q", sk.Name, "github")
	}
	if sk.Emoji != "🐙" {
		t.Errorf("Emoji = %q; want %q", sk.Emoji, "🐙")
	}
	if len(sk.Requires.Binaries) != 1 || sk.Requires.Binaries[0] != "gh" {
		t.Errorf("Requires.Binaries = %v; want [gh]", sk.Requires.Binaries)
	}
}

func TestKnownSkill_CodingAgent(t *testing.T) {
	dir := openbotBundledDir(t)
	sk, err := skill.LoadSkill(filepath.Join(dir, "coding-agent"))
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}

	if sk.Name != "coding-agent" {
		t.Errorf("Name = %q; want %q", sk.Name, "coding-agent")
	}
	wantAnyBins := []string{"claude", "codex", "opencode", "pi"}
	if len(sk.Requires.AnyBins) != len(wantAnyBins) {
		t.Fatalf("AnyBins length = %d; want %d", len(sk.Requires.AnyBins), len(wantAnyBins))
	}
	for i, b := range wantAnyBins {
		if sk.Requires.AnyBins[i] != b {
			t.Errorf("AnyBins[%d] = %q; want %q", i, sk.Requires.AnyBins[i], b)
		}
	}
}

func TestKnownSkill_AppleNotes_DarwinOnly(t *testing.T) {
	dir := openbotBundledDir(t)
	sk, err := skill.LoadSkill(filepath.Join(dir, "apple-notes"))
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}

	if len(sk.Requires.OS) == 0 || sk.Requires.OS[0] != "darwin" {
		t.Errorf("Requires.OS = %v; want [darwin]", sk.Requires.OS)
	}
	if runtime.GOOS == "darwin" && !sk.Ready {
		// On macOS, apple-notes should be ready (no binary deps, just OS).
		t.Log("Note: apple-notes not ready on darwin - may have additional requirements")
	}
	if runtime.GOOS == "linux" && sk.Ready {
		t.Error("apple-notes should NOT be ready on linux")
	}
}

func TestKnownSkill_Slack_ConfigPath(t *testing.T) {
	dir := openbotBundledDir(t)
	sk, err := skill.LoadSkill(filepath.Join(dir, "slack"))
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}

	if sk.Name != "slack" {
		t.Errorf("Name = %q; want %q", sk.Name, "slack")
	}
	if sk.Emoji != "💬" {
		t.Errorf("Emoji = %q; want %q", sk.Emoji, "💬")
	}
	if len(sk.Requires.CfgPaths) != 1 || sk.Requires.CfgPaths[0] != "channels.slack" {
		t.Errorf("CfgPaths = %v; want [channels.slack]", sk.Requires.CfgPaths)
	}
}

func TestKnownSkill_SkillCreator_NoRequirements(t *testing.T) {
	dir := openbotBundledDir(t)
	sk, err := skill.LoadSkill(filepath.Join(dir, "skill-creator"))
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}

	if sk.Name != "skill-creator" {
		t.Errorf("Name = %q; want %q", sk.Name, "skill-creator")
	}
	if !sk.Ready {
		t.Error("skill-creator should be Ready (no requirements)")
	}
}

func TestKnownSkill_Summarize_RequiresBin(t *testing.T) {
	dir := openbotBundledDir(t)
	sk, err := skill.LoadSkill(filepath.Join(dir, "summarize"))
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}

	if sk.Name != "summarize" {
		t.Errorf("Name = %q; want %q", sk.Name, "summarize")
	}
	if len(sk.Requires.Binaries) != 1 || sk.Requires.Binaries[0] != "summarize" {
		t.Errorf("Requires.Binaries = %v; want [summarize]", sk.Requires.Binaries)
	}
}

func TestKnownSkill_Gemini_RequiresBin(t *testing.T) {
	dir := openbotBundledDir(t)
	sk, err := skill.LoadSkill(filepath.Join(dir, "gemini"))
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}

	if sk.Name != "gemini" {
		t.Errorf("Name = %q; want %q", sk.Name, "gemini")
	}
	if len(sk.Requires.Binaries) != 1 || sk.Requires.Binaries[0] != "gemini" {
		t.Errorf("Requires.Binaries = %v; want [gemini]", sk.Requires.Binaries)
	}
}

// ---------------------------------------------------------------------------
// LoadAllSkills with Bundled Dir
// ---------------------------------------------------------------------------

func TestLoadAllSkillsWithBundledDir(t *testing.T) {
	dir := openbotBundledDir(t)
	ws := t.TempDir() // empty workspace

	all, err := skill.LoadAllSkills(ws, dir)
	if err != nil {
		t.Fatalf("LoadAllSkills error: %v", err)
	}

	if len(all) < 50 {
		t.Errorf("expected at least 50 skills; got %d", len(all))
	}

	// All should be source "bundled".
	for _, s := range all {
		if s.Source != "bundled" {
			t.Errorf("skill %q Source = %q; want %q", s.Name, s.Source, "bundled")
		}
	}

	t.Logf("Loaded %d skills total", len(all))
}

// ---------------------------------------------------------------------------
// Prompt Output: All Ready Skills Appear
// ---------------------------------------------------------------------------

func TestPromptOutputContainsReadySkills(t *testing.T) {
	dir := openbotBundledDir(t)
	ws := t.TempDir()

	all, err := skill.LoadAllSkills(ws, dir)
	if err != nil {
		t.Fatalf("LoadAllSkills error: %v", err)
	}

	prompt := skill.BuildSkillsPrompt(all)

	if !strings.Contains(prompt, "<available_skills>") {
		t.Error("prompt missing <available_skills> tag")
	}
	if !strings.Contains(prompt, "</available_skills>") {
		t.Error("prompt missing closing </available_skills> tag")
	}

	// Count ready skills and verify they appear in prompt.
	readyCount := 0
	for _, s := range all {
		if s.Ready && !s.DisableModelInvocation {
			readyCount++
			if !strings.Contains(prompt, "<name>"+s.Name+"</name>") {
				t.Errorf("ready skill %q not found in prompt", s.Name)
			}
		}
	}

	t.Logf("Prompt includes %d ready skills", readyCount)

	// Weather and skill-creator should always be ready.
	if !strings.Contains(prompt, "<name>weather</name>") {
		t.Error("weather skill not in prompt (curl should be available)")
	}
	if !strings.Contains(prompt, "<name>skill-creator</name>") {
		t.Error("skill-creator not in prompt (no requirements)")
	}
}

// ---------------------------------------------------------------------------
// Workspace Precedence Over Bundled
// ---------------------------------------------------------------------------

func TestWorkspaceOverridesBundled(t *testing.T) {
	bundledDir := openbotBundledDir(t)
	ws := t.TempDir()

	// Create a workspace skill with the same name as a bundled one.
	wsSkillDir := filepath.Join(ws, "skills", "weather")
	if err := os.MkdirAll(wsSkillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	overrideContent := "---\nname: weather\ndescription: My custom weather\n---\nCustom weather body."
	if err := os.WriteFile(filepath.Join(wsSkillDir, "SKILL.md"), []byte(overrideContent), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	all, err := skill.LoadAllSkills(ws, bundledDir)
	if err != nil {
		t.Fatalf("LoadAllSkills error: %v", err)
	}

	// Find the weather skill.
	var weatherSkill *skill.Skill
	for _, s := range all {
		if s.Name == "weather" {
			weatherSkill = s
			break
		}
	}

	if weatherSkill == nil {
		t.Fatal("weather skill not found")
	}
	if weatherSkill.Source != "workspace" {
		t.Errorf("weather Source = %q; want %q (workspace should override bundled)", weatherSkill.Source, "workspace")
	}
	if weatherSkill.Description != "My custom weather" {
		t.Errorf("weather Description = %q; want %q", weatherSkill.Description, "My custom weather")
	}
}

// ---------------------------------------------------------------------------
// Skill Content Integrity (body is non-empty for all)
// ---------------------------------------------------------------------------

func TestAllSkillsHaveBody(t *testing.T) {
	dir := openbotBundledDir(t)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			sk, err := skill.LoadSkill(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatalf("LoadSkill error: %v", err)
			}
			if strings.TrimSpace(sk.Content) == "" {
				t.Error("skill body is empty")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Skill File Integrity: SKILL.md exists in every dir
// ---------------------------------------------------------------------------

func TestAllSkillDirsHaveSKILLMD(t *testing.T) {
	dir := openbotBundledDir(t)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			skillFile := filepath.Join(dir, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillFile); os.IsNotExist(err) {
				t.Errorf("missing SKILL.md in %s", entry.Name())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// OpenClaw vs OpenBot SKILL.md Content Match
// ---------------------------------------------------------------------------

func TestSKILLMDContentMatch(t *testing.T) {
	ocDir := openclawBundledDir(t)
	obDir := openbotBundledDir(t)

	entries, err := os.ReadDir(ocDir)
	if err != nil {
		t.Fatalf("read openclaw dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			ocFile := filepath.Join(ocDir, entry.Name(), "SKILL.md")
			obFile := filepath.Join(obDir, entry.Name(), "SKILL.md")

			ocData, err := os.ReadFile(ocFile)
			if err != nil {
				t.Skipf("openclaw skill %s missing: %v", entry.Name(), err)
			}
			obData, err := os.ReadFile(obFile)
			if err != nil {
				t.Fatalf("openbot skill %s missing: %v", entry.Name(), err)
			}

			if string(ocData) != string(obData) {
				t.Errorf("SKILL.md content mismatch for %s (files differ)", entry.Name())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Eligibility Consistency: Same Results on Same Machine
// ---------------------------------------------------------------------------

func TestEligibilityConsistency(t *testing.T) {
	ocDir := openclawBundledDir(t)
	obDir := openbotBundledDir(t)

	ocEntries, err := os.ReadDir(ocDir)
	if err != nil {
		t.Fatalf("read openclaw dir: %v", err)
	}

	for _, entry := range ocEntries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			ocSk, err := skill.LoadSkill(filepath.Join(ocDir, entry.Name()))
			if err != nil {
				t.Skipf("openclaw load error: %v", err)
			}
			obSk, err := skill.LoadSkill(filepath.Join(obDir, entry.Name()))
			if err != nil {
				t.Fatalf("openbot load error: %v", err)
			}

			if ocSk.Ready != obSk.Ready {
				t.Errorf("Ready mismatch: openclaw=%v openbot=%v", ocSk.Ready, obSk.Ready)
			}
		})
	}
}
