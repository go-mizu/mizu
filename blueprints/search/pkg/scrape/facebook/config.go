package facebook

import (
	"os"
	"path/filepath"
	"time"
)

const (
	BaseURL         = "https://www.facebook.com"
	MBasicURL       = "https://mbasic.facebook.com"
	DefaultDelay    = 2 * time.Second
	DefaultWorkers  = 2
	DefaultTimeout  = 30 * time.Second
	DefaultMaxPages = 3
)

const (
	EntityPage    = "page"
	EntityProfile = "profile"
	EntityGroup   = "group"
	EntityPost    = "post"
	EntitySearch  = "search"
)

var userAgents = []string{
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
}

type Config struct {
	DataDir      string
	DBPath       string
	StatePath    string
	Workers      int
	Delay        time.Duration
	Timeout      time.Duration
	MaxPages     int
	MaxComments  int
	Cookies      string
	CookiesFile  string
	Entities     []string
	PreferMBasic bool
}

func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "facebook")
	return Config{
		DataDir:      dataDir,
		DBPath:       filepath.Join(dataDir, "facebook.duckdb"),
		StatePath:    filepath.Join(dataDir, "state.duckdb"),
		Workers:      DefaultWorkers,
		Delay:        DefaultDelay,
		Timeout:      DefaultTimeout,
		MaxPages:     DefaultMaxPages,
		MaxComments:  100,
		Entities:     []string{EntityPage, EntityProfile, EntityGroup, EntityPost},
		PreferMBasic: true,
	}
}
