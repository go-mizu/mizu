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
