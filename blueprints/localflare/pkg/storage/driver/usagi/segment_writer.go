package usagi

import (
	"os"
	"sync"
)

type segmentWriter struct {
	shard int
	mu    sync.Mutex
	file  *os.File
	id    int64
	size  int64
}
