package youtube

import (
	"os"
	"path/filepath"
	"time"
)

const (
	BaseURL            = "https://www.youtube.com"
	DefaultDelay       = 1500 * time.Millisecond
	DefaultWorkers     = 2
	DefaultTimeout     = 30 * time.Second
	DefaultMaxResults  = 30
	DefaultMaxPlaylist = 200
	EntityVideo        = "video"
	EntityChannel      = "channel"
	EntityPlaylist     = "playlist"
	EntitySearch       = "search"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
}

type Config struct {
	DataDir     string
	DBPath      string
	StatePath   string
	Workers     int
	Delay       time.Duration
	Timeout     time.Duration
	MaxResults  int
	MaxPlaylist int
}

func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "youtube")
	return Config{
		DataDir:     dataDir,
		DBPath:      filepath.Join(dataDir, "youtube.duckdb"),
		StatePath:   filepath.Join(dataDir, "state.duckdb"),
		Workers:     DefaultWorkers,
		Delay:       DefaultDelay,
		Timeout:     DefaultTimeout,
		MaxResults:  DefaultMaxResults,
		MaxPlaylist: DefaultMaxPlaylist,
	}
}
