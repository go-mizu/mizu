package kaggle

import (
	"os"
	"path/filepath"
	"time"
)

const (
	BaseURL           = "https://www.kaggle.com"
	DefaultDelay      = 1500 * time.Millisecond
	DefaultWorkers    = 2
	DefaultTimeout    = 30 * time.Second
	DefaultMaxPages   = 5
	DefaultPageSize   = 20
	EntityDataset     = "dataset"
	EntityModel       = "model"
	EntityCompetition = "competition"
	EntityNotebook    = "notebook"
	EntityProfile     = "profile"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) Gecko/20100101 Firefox/135.0",
}

type Config struct {
	DataDir   string
	DBPath    string
	StatePath string
	Workers   int
	Delay     time.Duration
	Timeout   time.Duration
	MaxPages  int
	PageSize  int
	Types     []string
}

func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "kaggle")
	return Config{
		DataDir:   dataDir,
		DBPath:    filepath.Join(dataDir, "kaggle.duckdb"),
		StatePath: filepath.Join(dataDir, "state.duckdb"),
		Workers:   DefaultWorkers,
		Delay:     DefaultDelay,
		Timeout:   DefaultTimeout,
		MaxPages:  DefaultMaxPages,
		PageSize:  DefaultPageSize,
		Types:     []string{EntityDataset, EntityModel},
	}
}
