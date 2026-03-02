package recrawler

import (
	"time"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

// FailedDB is deprecated. Use store.FailedDB directly.
type FailedDB = store.FailedDB

func NewFailedDB(path string) (*FailedDB, error)  { return store.NewFailedDB(path) }
func OpenFailedDB(path string) (*FailedDB, error) { return store.OpenFailedDB(path) }

func LoadRetryURLs(dbPath string) ([]crawl.SeedURL, error) {
	return store.LoadRetryURLs(dbPath)
}
func LoadRetryURLsSince(dbPath string, since time.Time) ([]crawl.SeedURL, error) {
	return store.LoadRetryURLsSince(dbPath, since)
}
func LoadFailedDomains(dbPath string) ([]crawl.FailedDomain, error) {
	return store.LoadFailedDomains(dbPath)
}
func FailedDomainSummary(dbPath string) (map[string]int, int, error) {
	return store.FailedDomainSummary(dbPath)
}
func FailedURLSummary(dbPath string) (map[string]int, int, error) {
	return store.FailedURLSummary(dbPath)
}
