package soundcloud

import (
	"os"
	"path/filepath"
	"time"
)

const (
	BaseURL         = "https://soundcloud.com"
	APIBaseURL      = "https://api-v2.soundcloud.com"
	DefaultDelay    = 2 * time.Second
	DefaultWorkers  = 2
	DefaultTimeout  = 30 * time.Second
	DefaultPageSize = 25
)

const (
	EntityTrack    = "track"
	EntityUser     = "user"
	EntityPlaylist = "playlist"
	EntitySearch   = "search"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.3 Safari/605.1.15",
}

type Config struct {
	DataDir   string
	DBPath    string
	StatePath string
	Workers   int
	Delay     time.Duration
	Timeout   time.Duration
	PageSize  int
}

func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "soundcloud")
	return Config{
		DataDir:   dataDir,
		DBPath:    filepath.Join(dataDir, "soundcloud.duckdb"),
		StatePath: filepath.Join(dataDir, "state.duckdb"),
		Workers:   DefaultWorkers,
		Delay:     DefaultDelay,
		Timeout:   DefaultTimeout,
		PageSize:  DefaultPageSize,
	}
}
