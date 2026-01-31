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
