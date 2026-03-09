//go:build chdb

package cc

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	indexpack "github.com/go-mizu/mizu/blueprints/search/pkg/index/pack"
)

func runPipelineFromDuckDBRaw(_ context.Context, _ index.Engine, _ string, _ int, _ indexpack.PackProgressFunc) (*indexpack.PipelineStats, error) {
	return nil, fmt.Errorf("duckdb source unavailable in chdb build")
}
