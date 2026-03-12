package amazon

import (
	"os"
	"path/filepath"
	"time"
)

// userAgents is a pool of browser User-Agent strings used in rotation.
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.6; rv:132.0) Gecko/20100101 Firefox/132.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.6778.135 Mobile Safari/537.36",
	"Mozilla/5.0 (Android 14; Mobile; rv:132.0) Gecko/132.0 Firefox/132.0",
}

// Config holds all configuration for the Amazon scraper.
type Config struct {
	DataDir   string
	DBPath    string
	StatePath string
	Workers   int
	Delay     time.Duration
	Timeout   time.Duration
	MaxPages  int  // 0 = unlimited
	ForceRod  bool // force rod for all fetches (default: only on HTTP failure)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "amazon")
	return Config{
		DataDir:   dataDir,
		DBPath:    filepath.Join(dataDir, "amazon.duckdb"),
		StatePath: filepath.Join(dataDir, "state.duckdb"),
		Workers:   DefaultWorkers,
		Delay:     DefaultDelay,
		Timeout:   DefaultTimeout,
		MaxPages:  0,
	}
}
