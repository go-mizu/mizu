//go:build chdb

package cli

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	indexpack "github.com/go-mizu/mizu/blueprints/search/pkg/index/pack"
)

func packDuckDBRaw(_ context.Context, _, _ string, _, _ int, _ indexpack.ProgressFunc) (*indexpack.PipelineStats, error) {
	return nil, fmt.Errorf("duckdb source unavailable in chdb build")
}

func runPipelineFromDuckDBRaw(_ context.Context, _ index.Engine, _ string, _ int, _ indexpack.PackProgressFunc) (*indexpack.PipelineStats, error) {
	return nil, fmt.Errorf("duckdb source unavailable in chdb build")
}
