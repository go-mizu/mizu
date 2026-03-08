package apify

import (
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultAlgoliaAppID      = "OW0O5I3QO7"
	DefaultAlgoliaAPIKey     = "0ecccd09f50396a4dbbe5dbfb17f4525"
	DefaultAlgoliaIndexName  = "prod_PUBLIC_STORE"
	DefaultHitsPerPage       = 1000
	DefaultStoreURL          = "https://apify.com/store/categories"
	DefaultActorAPIBaseURL   = "https://api.apify.com"
	DefaultAlgoliaDSNBaseURL = "https://OW0O5I3QO7-dsn.algolia.net"
)

// Config controls Apify Store crawl behavior.
type Config struct {
	DataDir           string
	DBPath            string
	StoreURL          string
	AlgoliaDSNBaseURL string
	ActorAPIBaseURL   string
	AlgoliaAppID      string
	AlgoliaAPIKey     string
	AlgoliaIndexName  string
	HitsPerPage       int
	Workers           int
	QPS               float64
	Timeout           time.Duration
	MaxRetries        int
	MaxDetails        int // 0 = unlimited
	RefreshDetails    bool
	IndexOnly         bool
	DetailOnly        bool
	InitialPage       int // mostly for debugging; default 0
	EnrichVersions    bool
	EnrichLatestBuild bool
}

func DefaultConfig() Config {
	dataDir := defaultDataDir()
	return Config{
		DataDir:           dataDir,
		DBPath:            filepath.Join(dataDir, "apify.duckdb"),
		StoreURL:          DefaultStoreURL,
		AlgoliaDSNBaseURL: DefaultAlgoliaDSNBaseURL,
		ActorAPIBaseURL:   DefaultActorAPIBaseURL,
		AlgoliaAppID:      DefaultAlgoliaAppID,
		AlgoliaAPIKey:     DefaultAlgoliaAPIKey,
		AlgoliaIndexName:  DefaultAlgoliaIndexName,
		HitsPerPage:       DefaultHitsPerPage,
		Workers:           16,
		QPS:               25,
		Timeout:           30 * time.Second,
		MaxRetries:        3,
		InitialPage:       0,
		EnrichVersions:    true,
		EnrichLatestBuild: true,
	}
}

func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "apify")
}
