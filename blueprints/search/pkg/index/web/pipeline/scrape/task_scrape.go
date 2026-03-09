package scrape

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler"
)

// Compile-time check.
var _ core.Task[ScrapeState, ScrapeMetric] = (*ScrapeTask)(nil)

// ScrapeState is emitted periodically during a domain scrape.
type ScrapeState struct {
	Domain      string  `json:"domain"`
	Pages       int64   `json:"pages"`
	Success     int64   `json:"success"`
	Failed      int64   `json:"failed"`
	Timeout     int64   `json:"timeout"`
	Blocked     int64   `json:"blocked"`
	Skipped     int64   `json:"skipped"`
	Frontier    int     `json:"frontier"`
	InFlight    int64   `json:"in_flight"`
	BytesRecv   int64   `json:"bytes_recv"`
	LinksFound  int64   `json:"links_found"`
	PagesPerSec float64 `json:"pages_per_sec"`
	BytesPerSec float64 `json:"bytes_per_sec"`
	PeakSpeed   float64 `json:"peak_speed"`
	AvgFetchMs  float64 `json:"avg_fetch_ms"`
	RetryQueue  int     `json:"retry_queue"`
	ElapsedMs   int64   `json:"elapsed_ms"`
	Progress    float64 `json:"progress"`
}

// ScrapeMetric is the final result of a domain scrape.
type ScrapeMetric struct {
	Domain  string
	Pages   int64
	Success int64
	Failed  int64
	Bytes   int64
	Links   int64
	Elapsed time.Duration
}

// ScrapeTask wraps dcrawler.Crawler as a core.Task[ScrapeState, ScrapeMetric].
type ScrapeTask struct {
	cfg dcrawler.Config
}

// NewScrapeTask creates a ScrapeTask from the given dcrawler config.
func NewScrapeTask(cfg dcrawler.Config) *ScrapeTask {
	return &ScrapeTask{cfg: cfg}
}

// Run starts the domain crawl, polls stats, and emits progress updates.
func (t *ScrapeTask) Run(ctx context.Context, emit func(*ScrapeState)) (ScrapeMetric, error) {
	crawler, err := dcrawler.New(t.cfg)
	if err != nil {
		return ScrapeMetric{}, err
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- crawler.Run(ctx)
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-errCh:
			emit(snapshot(t.cfg, crawler))
			return metric(t.cfg, crawler), err
		case <-ticker.C:
			emit(snapshot(t.cfg, crawler))
		}
	}
}

func snapshot(cfg dcrawler.Config, c *dcrawler.Crawler) *ScrapeState {
	stats := c.Stats()
	speed := stats.Speed() // must call before ByteSpeed/PeakSpeed (updates rolling window)
	progress := float64(0)
	if cfg.MaxPages > 0 {
		progress = float64(stats.Done()) / float64(cfg.MaxPages)
		if progress > 1 {
			progress = 1
		}
	}
	return &ScrapeState{
		Domain:      cfg.Domain,
		Pages:       stats.Done(),
		Success:     stats.Success(),
		Failed:      stats.Failed(),
		Timeout:     stats.Timeout(),
		Blocked:     stats.Blocked(),
		Skipped:     stats.Skipped(),
		Frontier:    stats.FrontierLen(),
		InFlight:    stats.InFlight(),
		BytesRecv:   stats.Bytes(),
		LinksFound:  stats.LinksFound(),
		PagesPerSec: speed,
		BytesPerSec: stats.ByteSpeed(),
		PeakSpeed:   stats.PeakSpeed(),
		AvgFetchMs:  stats.AvgFetchMs(),
		RetryQueue:  stats.RetryQLen(),
		ElapsedMs:   stats.Elapsed().Milliseconds(),
		Progress:    progress,
	}
}

func metric(cfg dcrawler.Config, c *dcrawler.Crawler) ScrapeMetric {
	stats := c.Stats()
	stats.Freeze()
	return ScrapeMetric{
		Domain:  cfg.Domain,
		Pages:   stats.Done(),
		Success: stats.Success(),
		Failed:  stats.Failed(),
		Bytes:   stats.Bytes(),
		Links:   stats.LinksFound(),
		Elapsed: stats.Elapsed(),
	}
}
