package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_FromFile(t *testing.T) {
	t.Setenv("TELEGRAM_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")

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

	expected := filepath.Join(dir, "workspace")
	if cfg.Workspace != expected {
		t.Errorf("expected default workspace %s, got %s", expected, cfg.Workspace)
	}
}

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
