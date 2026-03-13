package amazon

import (
	"os"
	"path/filepath"
	"time"
)

const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

type Config struct {
	DataDir   string
	Market    string
	Workers   int
	Timeout   time.Duration
	RateLimit float64
	MaxPages  int
	Resume    bool
	UserAgent string
	SortBy    string
}

func DefaultConfig() Config {
	return Config{
		DataDir:   defaultDataDir(),
		Market:    "www.amazon.com",
		Workers:   4,
		Timeout:   20 * time.Second,
		RateLimit: 1.5,
		MaxPages:  5,
		UserAgent: DefaultUserAgent,
	}
}

func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "amazon-search")
}

func (c Config) DBPath() string {
	return filepath.Join(c.DataDir, "amazon.duckdb")
}
