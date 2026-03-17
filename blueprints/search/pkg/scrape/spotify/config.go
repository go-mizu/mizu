package spotify

import (
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultDelay   = 250 * time.Millisecond
	DefaultWorkers = 2
	DefaultTimeout = 30 * time.Second
)

var userAgents = []string{
	// Spotify serves richer anonymous SSR HTML, including initialState,
	// for the generic mobile-web-player UA.
	"Mozilla/5.0",
}

type Config struct {
	DataDir   string
	DBPath    string
	StatePath string
	Workers   int
	Delay     time.Duration
	Timeout   time.Duration
}

func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "spotify")
	return Config{
		DataDir:   dataDir,
		DBPath:    filepath.Join(dataDir, "spotify.duckdb"),
		StatePath: filepath.Join(dataDir, "state.duckdb"),
		Workers:   DefaultWorkers,
		Delay:     DefaultDelay,
		Timeout:   DefaultTimeout,
	}
}
