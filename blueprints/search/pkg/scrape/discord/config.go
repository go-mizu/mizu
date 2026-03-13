package discord

import (
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultDelay   = 500 * time.Millisecond
	DefaultWorkers = 2
	DefaultTimeout = 30 * time.Second
)

// Config holds configuration for the Discord scraper.
type Config struct {
	Token     string
	DataDir   string
	DBPath    string
	StatePath string
	Workers   int
	Delay     time.Duration
	Timeout   time.Duration
}

// DefaultConfig returns a Config with default values.
// Token is read from DISCORD_TOKEN env var if set.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "discord")
	return Config{
		Token:     os.Getenv("DISCORD_TOKEN"),
		DataDir:   dataDir,
		DBPath:    filepath.Join(dataDir, "discord.duckdb"),
		StatePath: filepath.Join(dataDir, "state.duckdb"),
		Workers:   DefaultWorkers,
		Delay:     DefaultDelay,
		Timeout:   DefaultTimeout,
	}
}
