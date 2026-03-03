package warc_md

// This file contains deprecated stubs retained only for CLI compile
// compatibility until Task 10 rewrites cli/cc_warc_markdown.go.
// All symbols here will be removed in Task 10.

import (
	"context"
	"fmt"
)

// CompressConfig is a deprecated stub. Phase 3 (compress) has been removed.
//
// Deprecated: will be deleted in Task 10.
type CompressConfig struct {
	InputDir  string
	OutputDir string
	Workers   int
	Force     bool
}

// RunCompress is a deprecated stub. Phase 3 (compress) has been removed.
//
// Deprecated: will be deleted in Task 10.
func RunCompress(_ context.Context, _ CompressConfig, _ ProgressFunc) (*PhaseStats, error) {
	return nil, fmt.Errorf("RunCompress: phase 3 (compress) has been removed")
}

// RunInMemoryPipeline is a deprecated stub. The in-memory pipeline has been
// removed along with phase 3.
//
// Deprecated: will be deleted in Task 10.
func RunInMemoryPipeline(_ context.Context, _ Config, _ []string, _ ProgressFunc) (*PipelineResult, error) {
	return nil, fmt.Errorf("RunInMemoryPipeline: in-memory pipeline has been removed")
}
