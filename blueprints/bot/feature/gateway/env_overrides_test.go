package gateway

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
)

func TestApplySkillEnvOverrides_NoSkills(t *testing.T) {
	cleanup := applySkillEnvOverrides(nil, "")
	cleanup() // should not panic
}

func TestApplySkillEnvOverrides_NoConfig(t *testing.T) {
	skills := []*skill.Skill{
		{Name: "test", Ready: true, PrimaryEnv: "TEST_KEY"},
	}
	// No config file exists at default path; should return a no-op cleanup.
	cleanup := applySkillEnvOverrides(skills, "/nonexistent")
	cleanup()
}

func TestApplySkillEnvOverrides_SetsEnvFromConfig(t *testing.T) {
	// Create a temporary config directory with openbot.json.
	dir := t.TempDir()
	configDir := filepath.Join(dir, ".openbot")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := map[string]any{
		"skills": map[string]any{
			"entries": map[string]any{
				"my-skill": map[string]any{
					"env": map[string]any{
						"MY_CUSTOM_VAR": "custom_value",
					},
					"apiKey": "sk-test-key-123",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(filepath.Join(configDir, "openbot.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Override HOME to point to our temp dir so DefaultConfigPath resolves.
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	// Clear any pre-existing values.
	os.Unsetenv("MY_CUSTOM_VAR")
	os.Unsetenv("MY_PRIMARY_ENV")

	skills := []*skill.Skill{
		{
			Name:       "my-skill",
			Ready:      true,
			PrimaryEnv: "MY_PRIMARY_ENV",
		},
	}

	cleanup := applySkillEnvOverrides(skills, dir)

	// Verify env vars were set.
	if got := os.Getenv("MY_CUSTOM_VAR"); got != "custom_value" {
		t.Errorf("MY_CUSTOM_VAR = %q, want %q", got, "custom_value")
	}
	if got := os.Getenv("MY_PRIMARY_ENV"); got != "sk-test-key-123" {
		t.Errorf("MY_PRIMARY_ENV = %q, want %q", got, "sk-test-key-123")
	}

	// Cleanup should restore original values.
	cleanup()

	if got := os.Getenv("MY_CUSTOM_VAR"); got != "" {
		t.Errorf("after cleanup MY_CUSTOM_VAR = %q, want empty", got)
	}
	if got := os.Getenv("MY_PRIMARY_ENV"); got != "" {
		t.Errorf("after cleanup MY_PRIMARY_ENV = %q, want empty", got)
	}
}

func TestApplySkillEnvOverrides_PreservesExistingValues(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, ".openbot")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := map[string]any{
		"skills": map[string]any{
			"entries": map[string]any{
				"my-skill": map[string]any{
					"env": map[string]any{
						"EXISTING_VAR": "override_value",
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(filepath.Join(configDir, "openbot.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	// Set an existing value that should be preserved on cleanup.
	os.Setenv("EXISTING_VAR", "original_value")

	skills := []*skill.Skill{
		{Name: "my-skill", Ready: true},
	}

	cleanup := applySkillEnvOverrides(skills, dir)

	if got := os.Getenv("EXISTING_VAR"); got != "override_value" {
		t.Errorf("EXISTING_VAR = %q, want %q", got, "override_value")
	}

	cleanup()

	if got := os.Getenv("EXISTING_VAR"); got != "original_value" {
		t.Errorf("after cleanup EXISTING_VAR = %q, want %q", got, "original_value")
	}
}

func TestApplySkillEnvOverrides_SkipsNotReadySkills(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, ".openbot")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := map[string]any{
		"skills": map[string]any{
			"entries": map[string]any{
				"not-ready": map[string]any{
					"env": map[string]any{
						"SHOULD_NOT_SET": "bad",
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(filepath.Join(configDir, "openbot.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	os.Unsetenv("SHOULD_NOT_SET")

	skills := []*skill.Skill{
		{Name: "not-ready", Ready: false},
	}

	cleanup := applySkillEnvOverrides(skills, dir)
	defer cleanup()

	if got := os.Getenv("SHOULD_NOT_SET"); got != "" {
		t.Errorf("SHOULD_NOT_SET = %q, want empty (skill not ready)", got)
	}
}

func TestApplySkillEnvOverrides_UsesSkillKey(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, ".openbot")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := map[string]any{
		"skills": map[string]any{
			"entries": map[string]any{
				"custom-key": map[string]any{
					"apiKey": "sk-via-skillkey",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(filepath.Join(configDir, "openbot.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	os.Unsetenv("PRIMARY_KEY")

	skills := []*skill.Skill{
		{
			Name:       "my-skill",
			SkillKey:   "custom-key",
			Ready:      true,
			PrimaryEnv: "PRIMARY_KEY",
		},
	}

	cleanup := applySkillEnvOverrides(skills, dir)

	if got := os.Getenv("PRIMARY_KEY"); got != "sk-via-skillkey" {
		t.Errorf("PRIMARY_KEY = %q, want %q", got, "sk-via-skillkey")
	}

	cleanup()

	if got := os.Getenv("PRIMARY_KEY"); got != "" {
		t.Errorf("after cleanup PRIMARY_KEY = %q, want empty", got)
	}
}
