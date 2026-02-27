package crawl

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/archived/recrawler"
)

// DevNullResultWriter implements ResultWriter by discarding all results.
// Use with --writer devnull for benchmarking pure crawl throughput without I/O overhead.
type DevNullResultWriter struct{}

func (w *DevNullResultWriter) Add(_ recrawler.Result)        {}
func (w *DevNullResultWriter) Flush(_ context.Context) error { return nil }
func (w *DevNullResultWriter) Close() error                  { return nil }

// DevNullFailureWriter implements FailureWriter by discarding all failures.
// Used alongside DevNullResultWriter for benchmark runs.
type DevNullFailureWriter struct{}

func (w *DevNullFailureWriter) AddURL(_ recrawler.FailedURL) {}
func (w *DevNullFailureWriter) Close() error                 { return nil }
