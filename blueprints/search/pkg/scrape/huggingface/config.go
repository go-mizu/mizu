package huggingface

import (
	"os"
	"path/filepath"
	"time"
)

const (
	BaseURL         = "https://huggingface.co"
	DefaultDelay    = 250 * time.Millisecond
	DefaultWorkers  = 4
	DefaultTimeout  = 30 * time.Second
	DefaultPageSize = 100
)

const (
	EntityModel      = "model"
	EntityDataset    = "dataset"
	EntitySpace      = "space"
	EntityCollection = "collection"
	EntityPaper      = "paper"
)

var userAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
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
	dataDir := filepath.Join(home, "data", "huggingface")
	return Config{
		DataDir:   dataDir,
		DBPath:    filepath.Join(dataDir, "huggingface.duckdb"),
		StatePath: filepath.Join(dataDir, "state.duckdb"),
		Workers:   DefaultWorkers,
		Delay:     DefaultDelay,
		Timeout:   DefaultTimeout,
		PageSize:  DefaultPageSize,
		Types:     []string{EntityModel, EntityDataset, EntitySpace, EntityCollection, EntityPaper},
	}
}
