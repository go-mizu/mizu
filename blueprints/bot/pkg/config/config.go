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
	ConfigDir    string // base config directory (~/.openbot)
	AnthropicKey string

	// Full OpenClaw-compatible config sections.
	Meta     MetaConfig     `json:"meta"`
	Wizard   WizardConfig   `json:"wizard"`
	Auth     AuthConfig     `json:"auth"`
	Agents   AgentsConfig   `json:"agents"`
	Messages MessagesConfig `json:"messages"`
	Commands CommandsConfig `json:"commands"`
	Gateway  GatewayConfig  `json:"gateway"`
	Plugins  PluginsConfig  `json:"plugins"`
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

// MetaConfig tracks version metadata.
type MetaConfig struct {
	LastTouchedVersion string `json:"lastTouchedVersion,omitempty"`
	LastTouchedAt      string `json:"lastTouchedAt,omitempty"`
}

// WizardConfig tracks setup wizard state.
type WizardConfig struct {
	LastRunAt      string `json:"lastRunAt,omitempty"`
	LastRunVersion string `json:"lastRunVersion,omitempty"`
	LastRunCommand string `json:"lastRunCommand,omitempty"`
	LastRunMode    string `json:"lastRunMode,omitempty"`
}

// AuthProfile represents a single auth profile.
type AuthProfile struct {
	Provider string `json:"provider"`
	Mode     string `json:"mode"`
}

// AuthConfig holds auth profiles.
type AuthConfig struct {
	Profiles map[string]AuthProfile `json:"profiles,omitempty"`
}

// ContextPruningConfig controls context pruning behavior.
type ContextPruningConfig struct {
	Mode string `json:"mode,omitempty"` // cache-ttl, none
	TTL  string `json:"ttl,omitempty"`  // e.g. "1h"
}

// CompactionConfig controls conversation compaction.
type CompactionConfig struct {
	Mode string `json:"mode,omitempty"` // safeguard, aggressive, off
}

// HeartbeatConfig controls periodic heartbeat messages.
type HeartbeatConfig struct {
	Every string `json:"every,omitempty"` // e.g. "30m"
}

// SubagentsConfig controls subagent concurrency.
type SubagentsConfig struct {
	MaxConcurrent int `json:"maxConcurrent,omitempty"`
}

// AgentDefaults holds default settings for all agents.
type AgentDefaults struct {
	Workspace      string               `json:"workspace,omitempty"`
	ContextPruning ContextPruningConfig `json:"contextPruning,omitempty"`
	Compaction     CompactionConfig     `json:"compaction,omitempty"`
	Heartbeat      HeartbeatConfig      `json:"heartbeat,omitempty"`
	MaxConcurrent  int                  `json:"maxConcurrent,omitempty"`
	Subagents      SubagentsConfig      `json:"subagents,omitempty"`
}

// AgentsConfig holds agent configuration.
type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
}

// MessagesConfig holds messaging behavior settings.
type MessagesConfig struct {
	AckReactionScope string `json:"ackReactionScope,omitempty"` // group-mentions, all, none
}

// CommandsConfig holds slash command behavior.
type CommandsConfig struct {
	Native       string `json:"native,omitempty"`       // auto, on, off
	NativeSkills string `json:"nativeSkills,omitempty"` // auto, on, off
}

// GatewayAuthConfig holds gateway authentication settings.
type GatewayAuthConfig struct {
	Mode           string `json:"mode,omitempty"`     // token, password
	Token          string `json:"token,omitempty"`
	Password       string `json:"password,omitempty"`
	AllowTailscale bool   `json:"allowTailscale,omitempty"`
}

// TailscaleConfig holds Tailscale integration settings.
type TailscaleConfig struct {
	Mode        string `json:"mode,omitempty"` // off, serve, funnel
	ResetOnExit bool   `json:"resetOnExit,omitempty"`
}

// GatewayConfig holds gateway settings.
type GatewayConfig struct {
	Port      int               `json:"port,omitempty"`
	Mode      string            `json:"mode,omitempty"` // local
	Bind      string            `json:"bind,omitempty"` // loopback, lan, tailnet, auto
	Auth      GatewayAuthConfig `json:"auth,omitempty"`
	Tailscale TailscaleConfig   `json:"tailscale,omitempty"`
}

// PluginEntry represents a single plugin configuration.
type PluginEntry struct {
	Enabled bool `json:"enabled"`
}

// PluginsConfig holds plugin settings.
type PluginsConfig struct {
	Entries map[string]PluginEntry `json:"entries,omitempty"`
}

// ChannelsConfig holds all channel configurations.
type ChannelsConfig struct {
	Telegram TelegramConfig `json:"telegram"`
}

// configFile represents the raw JSON structure of openbot.json (OpenClaw-compatible).
type configFile struct {
	Meta     MetaConfig     `json:"meta"`
	Wizard   WizardConfig   `json:"wizard"`
	Auth     AuthConfig     `json:"auth"`
	Agents   AgentsConfig   `json:"agents"`
	Messages MessagesConfig `json:"messages"`
	Commands CommandsConfig `json:"commands"`
	Channels ChannelsConfig `json:"channels"`
	Gateway  GatewayConfig  `json:"gateway"`
	Plugins  PluginsConfig  `json:"plugins"`
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

	configDir := filepath.Dir(path)

	cfg := &Config{
		Telegram:  raw.Channels.Telegram,
		Workspace: raw.Agents.Defaults.Workspace,
		ConfigDir: configDir,
		Meta:      raw.Meta,
		Wizard:    raw.Wizard,
		Auth:      raw.Auth,
		Agents:    raw.Agents,
		Messages:  raw.Messages,
		Commands:  raw.Commands,
		Gateway:   raw.Gateway,
		Plugins:   raw.Plugins,
	}

	// Default workspace to <configDir>/workspace if not specified.
	if cfg.Workspace == "" {
		cfg.Workspace = filepath.Join(configDir, "workspace")
	}

	// Default data dir to <configDir> (matching OpenClaw layout where
	// agents/memory/etc. live directly under ~/.openbot/).
	cfg.DataDir = configDir

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

// SaveToFile writes the config to disk with backup rotation (5 deep).
func SaveToFile(path string, cfg *Config) error {
	// Rotate backups before writing.
	rotateBackups(path, 5)

	raw := configFile{
		Meta:     cfg.Meta,
		Wizard:   cfg.Wizard,
		Auth:     cfg.Auth,
		Agents:   cfg.Agents,
		Messages: cfg.Messages,
		Commands: cfg.Commands,
		Channels: ChannelsConfig{Telegram: cfg.Telegram},
		Gateway:  cfg.Gateway,
		Plugins:  cfg.Plugins,
	}

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// rotateBackups rotates .bak files: .bak.4 → deleted, .bak.3 → .bak.4, etc.
func rotateBackups(path string, maxBackups int) {
	// Remove oldest backup.
	oldest := fmt.Sprintf("%s.bak.%d", path, maxBackups-1)
	os.Remove(oldest)

	// Shift existing backups up.
	for i := maxBackups - 2; i >= 1; i-- {
		src := fmt.Sprintf("%s.bak.%d", path, i)
		dst := fmt.Sprintf("%s.bak.%d", path, i+1)
		os.Rename(src, dst)
	}

	// .bak → .bak.1
	os.Rename(path+".bak", path+".bak.1")

	// current → .bak
	os.Rename(path, path+".bak")
}

// ConfigGet retrieves a config value by dot-separated path.
func ConfigGet(path string, data map[string]any) (any, bool) {
	parts := splitDotPath(path)
	current := any(data)
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// ConfigSet sets a config value by dot-separated path.
func ConfigSet(path string, value any, data map[string]any) {
	parts := splitDotPath(path)
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}
		next, ok := current[part].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[part] = next
		}
		current = next
	}
}

// ConfigUnset removes a config value by dot-separated path.
func ConfigUnset(path string, data map[string]any) {
	parts := splitDotPath(path)
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			delete(current, part)
			return
		}
		next, ok := current[part].(map[string]any)
		if !ok {
			return
		}
		current = next
	}
}

// LoadRawConfig reads the config file as a generic map for get/set/unset operations.
func LoadRawConfig(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return raw, nil
}

// SaveRawConfig writes a generic map back to the config file with backup rotation.
func SaveRawConfig(path string, data map[string]any) error {
	rotateBackups(path, 5)
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, append(out, '\n'), 0o600)
}

func splitDotPath(path string) []string {
	var parts []string
	for _, p := range filepath.SplitList(path) {
		parts = append(parts, p)
	}
	// filepath.SplitList uses OS path separator; we need dot splitting.
	parts = nil
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '.' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	return parts
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
