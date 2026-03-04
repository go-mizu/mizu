//go:build chdb

package cli

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func packDuckDBRaw(_ context.Context, _, _ string, _, _ int, _ index.ProgressFunc) (*index.PipelineStats, error) {
	return nil, fmt.Errorf("duckdb source unavailable in chdb build")
}

func runPipelineFromDuckDBRaw(_ context.Context, _ index.Engine, _ string, _ int, _ index.PackProgressFunc) (*index.PipelineStats, error) {
	return nil, fmt.Errorf("duckdb source unavailable in chdb build")
}
