// Package-level HuggingFace types. The actual HTTP implementation that calls
// the HF API lives in the cli layer (cli/cc_publish_hf.go) to keep this
// package free of HTTP dependencies.
package arctic

import "context"

type HFOp struct {
	LocalPath  string
	PathInRepo string
	Delete     bool
}

type CommitFn func(ctx context.Context, ops []HFOp, message string) (string, error)
