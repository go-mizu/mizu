//go:build !chdb

package cc

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	duckdbdrv "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/duckdb"
	indexpack "github.com/go-mizu/mizu/blueprints/search/pkg/index/pack"
)

func runPipelineFromDuckDBRaw(ctx context.Context, eng index.Engine, packFile string, batchSize int, progress indexpack.PackProgressFunc) (*indexpack.PipelineStats, error) {
	return duckdbdrv.RunPipelineFromDuckDBRaw(ctx, eng, packFile, batchSize, progress)
}
