package pipeline

import (
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/util"
)

// Re-export helpers from util so existing consumers keep working via the
// pipeline package without importing the lower-level util package.

// ParseFileSelector parses a file selector string into a list of indices.
func ParseFileSelector(s string, total int) ([]int, error) {
	return util.ParseFileSelector(s, total)
}

// WARCFileIndex extracts the zero-padded 5-digit WARC file index from a path.
func WARCFileIndex(warcPath string, fallback int) string {
	return util.WARCFileIndex(warcPath, fallback)
}

// PackPath returns the expected pack file path for the given format and WARC index.
func PackPath(packDir, format, warcIdx string) (string, error) {
	return util.PackPath(packDir, format, warcIdx)
}

// PhaseProgress returns fractional progress clamped to [0, 1].
func PhaseProgress(done, total int64) float64 {
	return util.PhaseProgress(done, total)
}

// PhaseRate returns done/elapsed in items per second.
func PhaseRate(done int64, elapsed time.Duration) float64 {
	return util.PhaseRate(done, elapsed)
}

// MBPerSec converts bytes and elapsed time to MB/s.
func MBPerSec(bytes int64, elapsed time.Duration) float64 {
	return util.MBPerSec(bytes, elapsed)
}

// FileProgress computes overall progress across a multi-file loop.
func FileProgress(fileIdx, fileTotal int, fileFraction float64) float64 {
	return util.FileProgress(fileIdx, fileTotal, fileFraction)
}

// FileExists reports whether the given path exists.
func FileExists(path string) bool {
	return util.FileExists(path)
}

// NonBlockingEmit wraps an emit callback with a buffered channel so that slow
// consumers never block the task goroutine. Intermediate states are dropped
// when the channel is full — only the latest state matters for progress.
func NonBlockingEmit[S any](fn func(*S)) func(*S) {
	ch := make(chan *S, 64)
	go func() {
		for s := range ch {
			fn(s)
		}
	}()
	return func(s *S) {
		if s == nil {
			return
		}
		select {
		case ch <- s:
		default:
			// Drop intermediate state — consumer is behind.
		}
	}
}

// crawlSizeCache is an async-cached directory size helper.
type crawlSizeCache struct {
	mu   sync.Mutex
	size int64
	done bool
}

func newCrawlSizeCache(dir string) *crawlSizeCache {
	c := &crawlSizeCache{}
	go func() {
		sz := dirSize(dir)
		c.mu.Lock()
		c.size = sz
		c.done = true
		c.mu.Unlock()
	}()
	return c
}

// Get returns the cached directory size, or 0 if not yet computed.
func (c *crawlSizeCache) Get() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.size
}

// Refresh triggers a background recalculation.
func (c *crawlSizeCache) Refresh(dir string) {
	go func() {
		sz := dirSize(dir)
		c.mu.Lock()
		c.size = sz
		c.done = true
		c.mu.Unlock()
	}()
}
