package fw1

import (
	"os"
	"path/filepath"
	"time"
)

// Config holds downloader configuration.
type Config struct {
	DataDir     string        // Base directory for downloads (default: $HOME/data/fineweb-1)
	Concurrency int           // Concurrent downloads (default: 3)
	Timeout     time.Duration // Per-file timeout (default: 30 minutes)
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		DataDir:     filepath.Join(home, "data", "fineweb-1"),
		Concurrency: 3,
		Timeout:     30 * time.Minute,
	}
}
