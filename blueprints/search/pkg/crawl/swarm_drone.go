package crawl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/archived/recrawler"
)

// RunDrone is called by the hidden "cc recrawl-drone" CLI subcommand.
// It reads SeedURL JSON lines from stdin, crawls with KeepAliveEngine,
// and writes droneStats JSON lines to stdout periodically.
func RunDrone(ctx context.Context, cfg Config) error {
	var seeds []recrawler.SeedURL
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		var s recrawler.SeedURL
		if err := json.Unmarshal(scanner.Bytes(), &s); err == nil {
			seeds = append(seeds, s)
		}
	}

	// Count results for periodic reporting
	var okCount, failCount, timeoutCount int64
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	reportCh := make(chan droneStats, 32)

	// Reporting goroutine
	go func() {
		for {
			select {
			case ds := <-reportCh:
				enc := json.NewEncoder(os.Stdout)
				enc.Encode(ds) //nolint:errcheck
			case <-ctx.Done():
				return
			}
		}
	}()

	rw := &droneResultWriter{
		reportCh:     reportCh,
		okCount:      &okCount,
		failCount:    &failCount,
		timeoutCount: &timeoutCount,
		ticker:       ticker,
	}
	fw := &droneFailureWriter{failCount: &failCount, timeoutCount: &timeoutCount}

	_, err := (&KeepAliveEngine{}).Run(ctx, seeds, &NoopDNS{}, cfg, rw, fw)

	// Final report
	fmt.Fprintf(os.Stdout, `{"ok":%d,"failed":%d,"timeout":%d,"total":%d,"rps":0}`+"\n",
		okCount, failCount, timeoutCount, okCount+failCount+timeoutCount)
	return err
}

// droneResultWriter reports stats periodically while forwarding results.
type droneResultWriter struct {
	reportCh     chan droneStats
	okCount      *int64
	failCount    *int64
	timeoutCount *int64
	ticker       *time.Ticker
	lastOK       int64
}

func (d *droneResultWriter) Add(r recrawler.Result) {
	if r.Error == "" {
		*d.okCount++
	}
	// Periodic report check
	select {
	case <-d.ticker.C:
		d.reportCh <- droneStats{
			OK:      *d.okCount,
			Failed:  *d.failCount,
			Timeout: *d.timeoutCount,
			Total:   *d.okCount + *d.failCount + *d.timeoutCount,
		}
	default:
	}
}

func (d *droneResultWriter) Flush(_ context.Context) error { return nil }
func (d *droneResultWriter) Close() error                  { return nil }

type droneFailureWriter struct {
	failCount    *int64
	timeoutCount *int64
}

func (f *droneFailureWriter) AddURL(u recrawler.FailedURL) {
	if u.Reason == "http_timeout" {
		*f.timeoutCount++
	} else {
		*f.failCount++
	}
}
func (f *droneFailureWriter) Close() error { return nil }
