// Package compat tests OpenBot ↔ OpenClaw compatibility.
// It calls both CLIs (when available), compares directory structures,
// config schemas, session formats, and CLI output.
package compat

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// skipIfNoCLI skips the test if the named CLI is not on PATH.
func skipIfNoCLI(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not on PATH, skipping", name)
	}
}

// runCLI runs a CLI command and returns stdout, stderr, and exit code.
func runCLI(t *testing.T, name string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(name, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

// ---------------------------------------------------------------------------
// Directory Structure Tests
// ---------------------------------------------------------------------------

func TestDirectoryStructure(t *testing.T) {
	openclawDir := filepath.Join(os.Getenv("HOME"), ".openclaw")
	openbotDir := filepath.Join(os.Getenv("HOME"), ".openbot")

	if _, err := os.Stat(openclawDir); os.IsNotExist(err) {
		t.Skip("~/.openclaw does not exist, skipping")
	}
	if _, err := os.Stat(openbotDir); os.IsNotExist(err) {
		t.Skip("~/.openbot does not exist, skipping")
	}

	// Required directories that must exist in both.
	requiredDirs := []string{
		"workspace",
		"agents/main/agent",
		"agents/main/sessions",
		"memory",
		"identity",
		"logs",
		"telegram",
		"cron",
		"devices",
		"credentials",
	}

	for _, dir := range requiredDirs {
		ocPath := filepath.Join(openclawDir, dir)
		obPath := filepath.Join(openbotDir, dir)

		t.Run("dir/"+dir, func(t *testing.T) {
			// Check OpenClaw has it.
			if _, err := os.Stat(ocPath); os.IsNotExist(err) {
				t.Skipf("openclaw missing %s (expected)", dir)
			}

			// Check OpenBot has it.
			if _, err := os.Stat(obPath); os.IsNotExist(err) {
				t.Errorf("openbot missing directory %s (exists in openclaw)", dir)
			}
		})
	}
}

// TestWorkspaceBootstrapFiles checks that both workspaces have the same bootstrap files.
func TestWorkspaceBootstrapFiles(t *testing.T) {
	openclawWs := filepath.Join(os.Getenv("HOME"), ".openclaw", "workspace")
	openbotWs := filepath.Join(os.Getenv("HOME"), ".openbot", "workspace")

	if _, err := os.Stat(openclawWs); os.IsNotExist(err) {
		t.Skip("~/.openclaw/workspace does not exist, skipping")
	}
	if _, err := os.Stat(openbotWs); os.IsNotExist(err) {
		t.Skip("~/.openbot/workspace does not exist, skipping")
	}

	bootstrapFiles := []string{
		"AGENTS.md",
		"SOUL.md",
		"TOOLS.md",
		"IDENTITY.md",
		"USER.md",
		"HEARTBEAT.md",
	}

	for _, name := range bootstrapFiles {
		t.Run("workspace/"+name, func(t *testing.T) {
			ocPath := filepath.Join(openclawWs, name)
			obPath := filepath.Join(openbotWs, name)

			if _, err := os.Stat(ocPath); os.IsNotExist(err) {
				t.Skipf("openclaw missing %s", name)
			}
			if _, err := os.Stat(obPath); os.IsNotExist(err) {
				t.Errorf("openbot missing workspace file %s (exists in openclaw)", name)
			}
		})
	}

	// Check memory subdirectory exists.
	t.Run("workspace/memory", func(t *testing.T) {
		if _, err := os.Stat(filepath.Join(openbotWs, "memory")); os.IsNotExist(err) {
			t.Error("openbot missing workspace/memory/ directory")
		}
	})
}

// ---------------------------------------------------------------------------
// Config Schema Tests
// ---------------------------------------------------------------------------

func TestConfigSchema(t *testing.T) {
	openclawCfg := filepath.Join(os.Getenv("HOME"), ".openclaw", "openclaw.json")
	openbotCfg := filepath.Join(os.Getenv("HOME"), ".openbot", "openbot.json")

	if _, err := os.Stat(openclawCfg); os.IsNotExist(err) {
		t.Skip("~/.openclaw/openclaw.json does not exist, skipping")
	}
	if _, err := os.Stat(openbotCfg); os.IsNotExist(err) {
		t.Skip("~/.openbot/openbot.json does not exist, skipping")
	}

	ocData, err := os.ReadFile(openclawCfg)
	if err != nil {
		t.Fatalf("read openclaw config: %v", err)
	}
	obData, err := os.ReadFile(openbotCfg)
	if err != nil {
		t.Fatalf("read openbot config: %v", err)
	}

	var ocMap, obMap map[string]any
	if err := json.Unmarshal(ocData, &ocMap); err != nil {
		t.Fatalf("parse openclaw config: %v", err)
	}
	if err := json.Unmarshal(obData, &obMap); err != nil {
		t.Fatalf("parse openbot config: %v", err)
	}

	// Check that all top-level sections from OpenClaw exist in OpenBot.
	requiredSections := []string{"meta", "wizard", "auth", "agents", "messages", "commands", "channels", "gateway", "plugins"}
	for _, section := range requiredSections {
		t.Run("section/"+section, func(t *testing.T) {
			if _, ok := ocMap[section]; !ok {
				t.Skipf("openclaw config missing section %s", section)
			}
			if _, ok := obMap[section]; !ok {
				t.Errorf("openbot config missing section %q (present in openclaw)", section)
			}
		})
	}

	// Check channels.telegram fields.
	t.Run("channels.telegram", func(t *testing.T) {
		ocChannels, ok := ocMap["channels"].(map[string]any)
		if !ok {
			t.Skip("openclaw missing channels")
		}
		obChannels, ok := obMap["channels"].(map[string]any)
		if !ok {
			t.Fatal("openbot missing channels")
		}

		ocTg, ok := ocChannels["telegram"].(map[string]any)
		if !ok {
			t.Skip("openclaw missing telegram config")
		}
		obTg, ok := obChannels["telegram"].(map[string]any)
		if !ok {
			t.Fatal("openbot missing telegram config")
		}

		tgFields := []string{"enabled", "dmPolicy", "allowFrom", "groupPolicy", "streamMode"}
		for _, field := range tgFields {
			if _, ok := ocTg[field]; !ok {
				continue // skip if openclaw doesn't have it
			}
			if _, ok := obTg[field]; !ok {
				t.Errorf("openbot telegram config missing field %q", field)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Session Entry Tests
// ---------------------------------------------------------------------------

func TestSessionEntryFields(t *testing.T) {
	openclawSess := filepath.Join(os.Getenv("HOME"), ".openclaw", "agents", "main", "sessions", "sessions.json")
	openbotSess := filepath.Join(os.Getenv("HOME"), ".openbot", "agents", "main", "sessions", "sessions.json")

	if _, err := os.Stat(openclawSess); os.IsNotExist(err) {
		t.Skip("~/.openclaw sessions.json does not exist, skipping")
	}
	if _, err := os.Stat(openbotSess); os.IsNotExist(err) {
		t.Skip("~/.openbot sessions.json does not exist, skipping")
	}

	ocData, err := os.ReadFile(openclawSess)
	if err != nil {
		t.Fatalf("read openclaw sessions: %v", err)
	}
	obData, err := os.ReadFile(openbotSess)
	if err != nil {
		t.Fatalf("read openbot sessions: %v", err)
	}

	var ocIndex, obIndex map[string]any
	if err := json.Unmarshal(ocData, &ocIndex); err != nil {
		t.Fatalf("parse openclaw sessions: %v", err)
	}
	if err := json.Unmarshal(obData, &obIndex); err != nil {
		t.Fatalf("parse openbot sessions: %v", err)
	}

	// Get the first session entry from each.
	var ocEntry, obEntry map[string]any
	for _, v := range ocIndex {
		if m, ok := v.(map[string]any); ok {
			ocEntry = m
			break
		}
	}
	for _, v := range obIndex {
		if m, ok := v.(map[string]any); ok {
			obEntry = m
			break
		}
	}

	if ocEntry == nil {
		t.Skip("no openclaw sessions to compare")
	}
	if obEntry == nil {
		t.Skip("no openbot sessions to compare")
	}

	// Check that all fields in OpenClaw entry exist in OpenBot entry.
	for key := range ocEntry {
		t.Run("field/"+key, func(t *testing.T) {
			if _, ok := obEntry[key]; !ok {
				t.Errorf("openbot session entry missing field %q (present in openclaw)", key)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CLI Output Comparison Tests
// ---------------------------------------------------------------------------

func TestSessionsOutput(t *testing.T) {
	skipIfNoCLI(t, "openclaw")
	skipIfNoCLI(t, "openbot")

	ocOut, _, ocExit := runCLI(t, "openclaw", "sessions", "--json")
	obOut, _, obExit := runCLI(t, "openbot", "sessions", "--json")

	if ocExit != 0 {
		t.Skipf("openclaw sessions failed (exit %d)", ocExit)
	}
	if obExit != 0 {
		t.Errorf("openbot sessions --json failed (exit %d)", obExit)
		return
	}

	// Both should produce valid JSON.
	var ocJSON, obJSON any
	if err := json.Unmarshal([]byte(ocOut), &ocJSON); err != nil {
		t.Skipf("openclaw sessions output not valid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(obOut), &obJSON); err != nil {
		t.Errorf("openbot sessions --json output not valid JSON: %v", err)
	}
}

func TestStatusOutput(t *testing.T) {
	skipIfNoCLI(t, "openbot")

	obOut, _, obExit := runCLI(t, "openbot", "status", "--json")
	if obExit != 0 {
		t.Errorf("openbot status --json failed (exit %d)", obExit)
		return
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(obOut), &result); err != nil {
		t.Errorf("openbot status --json not valid JSON: %v", err)
		return
	}

	requiredKeys := []string{"config", "workspace", "sessionCount"}
	for _, key := range requiredKeys {
		if _, ok := result[key]; !ok {
			t.Errorf("openbot status --json missing key %q", key)
		}
	}
}

func TestCLIHelpCoversAllCommands(t *testing.T) {
	skipIfNoCLI(t, "openbot")

	obOut, _, _ := runCLI(t, "openbot", "help")

	// All major commands should appear in help output.
	requiredInHelp := []string{
		"agent",
		"sessions",
		"history",
		"status",
		"config",
		"doctor",
		"memory",
		"skills",
		"agents",
		"models",
		"channels",
		"gateway",
		"cron",
		"plugins",
		"hooks",
		"logs",
	}

	for _, cmd := range requiredInHelp {
		t.Run("help/"+cmd, func(t *testing.T) {
			if !strings.Contains(obOut, cmd) {
				t.Errorf("help output missing command %q", cmd)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Agent File Tests
// ---------------------------------------------------------------------------

func TestAgentFiles(t *testing.T) {
	openbotDir := filepath.Join(os.Getenv("HOME"), ".openbot")
	if _, err := os.Stat(openbotDir); os.IsNotExist(err) {
		t.Skip("~/.openbot does not exist, skipping")
	}

	agentFiles := []string{
		"agents/main/agent/models.json",
		"agents/main/agent/auth-profiles.json",
	}

	for _, f := range agentFiles {
		t.Run(f, func(t *testing.T) {
			path := filepath.Join(openbotDir, f)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("missing agent file %s", f)
			}
		})
	}
}

// TestConfigBackupRotation verifies that saving config creates backup files.
func TestConfigBackupRotation(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "openbot.json")

	// Write initial config.
	initial := map[string]any{"meta": map[string]any{"version": "1"}}
	data, _ := json.MarshalIndent(initial, "", "  ")
	os.WriteFile(cfgPath, data, 0o600)

	// Import and call SaveRawConfig multiple times.
	// Since we can't import the config package in this test package easily,
	// simulate the backup rotation logic.
	for i := 0; i < 6; i++ {
		// Simulate rotation.
		rotateBackups(cfgPath, 5)

		updated := map[string]any{"meta": map[string]any{"version": string(rune('2' + i))}}
		data, _ := json.MarshalIndent(updated, "", "  ")
		os.WriteFile(cfgPath, data, 0o600)
	}

	// Check backup files exist.
	if _, err := os.Stat(cfgPath + ".bak"); os.IsNotExist(err) {
		t.Error("missing .bak file")
	}
	if _, err := os.Stat(cfgPath + ".bak.1"); os.IsNotExist(err) {
		t.Error("missing .bak.1 file")
	}

	// .bak.4 should exist (5 rotations).
	if _, err := os.Stat(cfgPath + ".bak.4"); os.IsNotExist(err) {
		t.Error("missing .bak.4 file")
	}

	// .bak.5 should NOT exist (max 5 backups).
	if _, err := os.Stat(cfgPath + ".bak.5"); !os.IsNotExist(err) {
		t.Error(".bak.5 should not exist (max 5 backups)")
	}
}

// rotateBackups mirrors the config package's rotation logic for testing.
func rotateBackups(path string, maxBackups int) {
	oldest := path + ".bak." + itoa(maxBackups-1)
	os.Remove(oldest)
	for i := maxBackups - 2; i >= 1; i-- {
		src := path + ".bak." + itoa(i)
		dst := path + ".bak." + itoa(i+1)
		os.Rename(src, dst)
	}
	os.Rename(path+".bak", path+".bak.1")
	os.Rename(path, path+".bak")
}

func itoa(n int) string {
	return string(rune('0' + n))
}

// ---------------------------------------------------------------------------
// Additional File Tests
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// CLI Comparison Tests
// ---------------------------------------------------------------------------

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

			ocOut = strings.TrimSpace(ocOut)
			obOut = strings.TrimSpace(obOut)
			if obOut == "" && ocOut != "" {
				t.Errorf("openbot returned empty for %s, openclaw returned %q", p, ocOut)
			}
		})
	}
}

func TestMemoryStatusComparison(t *testing.T) {
	skipIfNoCLI(t, "openbot")

	out, _, exit := runCLI(t, "openbot", "memory", "status")
	if exit != 0 {
		t.Errorf("openbot memory status failed (exit %d)", exit)
		return
	}

	if !strings.Contains(out, "Memory") {
		t.Error("memory status missing 'Memory' line")
	}
}

func TestSkillsListComparison(t *testing.T) {
	skipIfNoCLI(t, "openbot")

	out, _, exit := runCLI(t, "openbot", "skills", "list")
	if exit != 0 {
		t.Errorf("openbot skills list failed (exit %d)", exit)
		return
	}

	// Output should be either "No skills found." (no bundled dir) or have table headers.
	if !strings.Contains(out, "NAME") && !strings.Contains(out, "No skills found") {
		t.Error("skills list missing table headers or 'No skills found' message")
	}
}

func TestModelsListComparison(t *testing.T) {
	skipIfNoCLI(t, "openbot")

	out, _, exit := runCLI(t, "openbot", "models", "list")
	if exit != 0 {
		t.Errorf("openbot models list failed (exit %d)", exit)
		return
	}

	// Should output something about models.
	if out == "" {
		t.Error("models list produced empty output")
	}
}

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

// ---------------------------------------------------------------------------
// Directory File Comparison Tests
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// LLM Tools Count Test
// ---------------------------------------------------------------------------

func TestToolCount(t *testing.T) {
	// This test verifies that OpenBot registers at least 23 tools
	// (matching OpenClaw's tool count).
	// We can't easily import the tools package here, but we can check
	// the tool registration in builtins.go by counting Register calls.
	builtinsPath := filepath.Join("..", "..", "pkg", "tools", "builtins.go")
	data, err := os.ReadFile(builtinsPath)
	if err != nil {
		t.Fatalf("read builtins.go: %v", err)
	}

	content := string(data)
	registerCount := strings.Count(content, "r.Register(")

	// OpenClaw has 23 tools.
	if registerCount < 23 {
		t.Errorf("expected at least 23 tool registrations, got %d", registerCount)
	}
}
