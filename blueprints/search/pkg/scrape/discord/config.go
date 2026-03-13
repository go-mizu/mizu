package discord

import (
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultDelay      = 500 * time.Millisecond
	DefaultJitter     = 300 * time.Millisecond // ±150ms random jitter
	DefaultWorkers    = 2
	DefaultTimeout    = 30 * time.Second
	DefaultMaxRetries = 5
)

// Config holds configuration for the Discord scraper.
type Config struct {
	Token      string
	DataDir    string
	DBPath     string
	StatePath  string
	Workers    int
	Delay      time.Duration
	Jitter     time.Duration // random jitter range added to delay (±Jitter/2)
	Timeout    time.Duration
	MaxRetries int // max retries on 429 rate limit (default 5)
}

// DefaultConfig returns a Config with default values.
// Token is read from DISCORD_TOKEN env var if set.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "discord")
	return Config{
		Token:      os.Getenv("DISCORD_TOKEN"),
		DataDir:    dataDir,
		DBPath:     filepath.Join(dataDir, "discord.duckdb"),
		StatePath:  filepath.Join(dataDir, "state.duckdb"),
		Workers:    DefaultWorkers,
		Delay:      DefaultDelay,
		Jitter:     DefaultJitter,
		Timeout:    DefaultTimeout,
		MaxRetries: DefaultMaxRetries,
	}
}
