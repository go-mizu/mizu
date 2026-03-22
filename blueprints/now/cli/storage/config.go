package storage

import (
	"bufio"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultEndpoint = "https://storage.liteio.dev"

// Config holds resolved CLI configuration.
type Config struct {
	Endpoint string
	Token    string
}

// Deps holds shared dependencies for all commands.
type Deps struct {
	Config *Config
	Client *Client
	Out    *Output
}

// ConfigDir returns the storage config directory.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "storage")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "storage")
}

// TokenFile returns the path to the token file.
func TokenFile() string {
	return filepath.Join(ConfigDir(), "token")
}

// LoadConfig loads configuration from file, env, and flags.
// Resolution order: config file < env vars < token file < flags.
func LoadConfig(flagToken, flagEndpoint string) *Config {
	cfg := &Config{
		Endpoint: DefaultEndpoint,
	}

	// 1. Config file (lowest priority)
	configPath := filepath.Join(ConfigDir(), "config")
	if f, err := os.Open(configPath); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k == "endpoint" {
				cfg.Endpoint = v
			}
		}
		f.Close()
	}

	// 2. Environment variables
	if v := os.Getenv("STORAGE_ENDPOINT"); v != "" {
		cfg.Endpoint = v
	}
	if v := os.Getenv("STORAGE_TOKEN"); v != "" {
		cfg.Token = v
	}

	// 3. Token file (if no env token)
	if cfg.Token == "" {
		if data, err := os.ReadFile(TokenFile()); err == nil {
			cfg.Token = strings.TrimSpace(string(data))
		}
	}

	// 4. Flags (highest priority)
	if flagEndpoint != "" {
		cfg.Endpoint = flagEndpoint
	}
	if flagToken != "" {
		cfg.Token = flagToken
	}

	return cfg
}

// NewClient creates an HTTP client from config.
func NewClient(cfg *Config) *Client {
	return &Client{
		Endpoint: cfg.Endpoint,
		Token:    cfg.Token,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// SaveToken writes a token to the token file with 0600 permissions.
func SaveToken(token string) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(TokenFile(), []byte(token), 0o600)
}

// RemoveToken deletes the token file.
func RemoveToken() error {
	return os.Remove(TokenFile())
}

// RequireToken returns an error if no token is configured.
func RequireToken(cfg *Config) error {
	if cfg.Token == "" {
		return &CLIError{
			Code: ExitAuth,
			Msg:  "not authenticated",
			Hint: "Run 'storage login' to authenticate\nOr set STORAGE_TOKEN environment variable",
		}
	}
	return nil
}
