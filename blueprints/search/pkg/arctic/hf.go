package arctic

import "context"

type HFOp struct {
	LocalPath  string
	PathInRepo string
	Delete     bool
}

type CommitFn func(ctx context.Context, ops []HFOp, message string) (string, error)
