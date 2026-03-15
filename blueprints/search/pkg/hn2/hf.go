package hn2

import "context"

// HFOp describes a single file operation within a Hugging Face dataset commit.
// Set Delete=true to remove a file from the repository; otherwise the file at
// LocalPath is uploaded to PathInRepo.
type HFOp struct {
	LocalPath  string
	PathInRepo string
	Delete     bool
}

// CommitFn is the function signature for committing a set of HFOps to Hugging Face.
// It returns the commit URL on success.
type CommitFn func(ctx context.Context, ops []HFOp, message string) (string, error)
