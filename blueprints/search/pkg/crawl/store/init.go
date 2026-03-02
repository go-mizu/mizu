package store

import (
	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
)

func init() {
	// Register factory functions so pkg/crawl/swarm_drone.go can create store objects
	// without importing store directly (which would create a pkg/crawl → store → pkg/crawl cycle).
	// These vars are set before RunDrone is called because any CLI binary that uses the swarm
	// engine also imports cli/recrawl.go or cli/hn.go, which import this package.
	crawl.DroneResultDBFactory = func(dir string, shardCount, batchSize, duckMemPerShardMB int) (crawl.ResultWriter, error) {
		return NewResultDB(dir, shardCount, batchSize, duckMemPerShardMB)
	}
	crawl.DroneFailedDBFactory = func(path string) (crawl.FailureWriter, error) {
		return OpenFailedDB(path)
	}
}
