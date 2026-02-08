package cc

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// SiteMode defines the extraction mode.
type SiteMode int

const (
	SiteModeURLs  SiteMode = iota // URLs + metadata only (CDX API, no WARC)
	SiteModeLinks                 // URLs + links (WARC fetch for HTML)
	SiteModeFull                  // URLs + links + full body
)

// SiteConfig configures site extraction.
type SiteConfig struct {
	Domain          string
	CrawlID         string
	Mode            SiteMode
	Workers         int
	Timeout         time.Duration
	MaxBodySize     int
	TransportShards int
	Resume          bool
	BatchSize       int
}

// DefaultSiteConfig returns sensible defaults.
func DefaultSiteConfig(domain string) SiteConfig {
	return SiteConfig{
		Domain:          domain,
		CrawlID:         "CC-MAIN-2026-04",
		Mode:            SiteModeURLs,
		Workers:         500,
		Timeout:         30 * time.Second,
		MaxBodySize:     512 * 1024,
		TransportShards: 32,
		BatchSize:       1000,
	}
}

// SiteStats tracks extraction progress.
type SiteStats struct {
	TotalPages int

	fetched  atomic.Int64
	success  atomic.Int64
	failed   atomic.Int64
	skipped  atomic.Int64
	bytesFetched   atomic.Int64
	bytesExtracted atomic.Int64
	linksFound     atomic.Int64

	startTime time.Time
	peakSpeed float64

	speedMu    sync.Mutex
	speedTicks []siteSpeedTick

	frozen   bool
	frozenAt time.Duration
}

type siteSpeedTick struct {
	time  time.Time
	count int64
}

// NewSiteStats creates a new stats tracker.
func NewSiteStats(totalPages int) *SiteStats {
	return &SiteStats{
		TotalPages: totalPages,
		startTime:  time.Now(),
	}
}

func (s *SiteStats) recordSuccess(fetchedBytes, extractedBytes int64, links int) {
	s.fetched.Add(1)
	s.success.Add(1)
	s.bytesFetched.Add(fetchedBytes)
	s.bytesExtracted.Add(extractedBytes)
	s.linksFound.Add(int64(links))
}

func (s *SiteStats) recordFailure() {
	s.fetched.Add(1)
	s.failed.Add(1)
}

func (s *SiteStats) recordSkip() {
	s.skipped.Add(1)
}

// Done returns total processed.
func (s *SiteStats) Done() int64 {
	return s.fetched.Load() + s.skipped.Load()
}

// Speed returns rolling pages/sec.
func (s *SiteStats) Speed() float64 {
	done := s.fetched.Load()
	now := time.Now()

	s.speedMu.Lock()
	s.speedTicks = append(s.speedTicks, siteSpeedTick{time: now, count: done})

	cutoff := now.Add(-10 * time.Second)
	start := 0
	for start < len(s.speedTicks) && s.speedTicks[start].time.Before(cutoff) {
		start++
	}
	if start > 0 && start < len(s.speedTicks) {
		s.speedTicks = s.speedTicks[start:]
	}

	var speed float64
	if len(s.speedTicks) >= 2 {
		first := s.speedTicks[0]
		last := s.speedTicks[len(s.speedTicks)-1]
		dt := last.time.Sub(first.time).Seconds()
		if dt > 0 {
			speed = float64(last.count-first.count) / dt
		}
	}
	s.speedMu.Unlock()

	if speed > s.peakSpeed {
		s.peakSpeed = speed
	}
	return speed
}

// Freeze locks elapsed time for final display.
func (s *SiteStats) Freeze() {
	if s.frozen {
		return
	}
	s.frozen = true
	s.frozenAt = time.Since(s.startTime)
}

// Elapsed returns elapsed time.
func (s *SiteStats) Elapsed() time.Duration {
	if s.frozen {
		return s.frozenAt
	}
	return time.Since(s.startTime)
}

// Render returns formatted stats.
func (s *SiteStats) Render() string {
	done := s.Done()
	total := int64(s.TotalPages)
	succ := s.success.Load()
	fail := s.failed.Load()
	skip := s.skipped.Load()
	links := s.linksFound.Load()
	speed := s.Speed()
	elapsed := s.Elapsed()

	pct := float64(0)
	if total > 0 {
		pct = float64(done) / float64(total) * 100
	}

	eta := "---"
	if elapsed.Seconds() > 2 && done > 0 {
		avgSpeed := float64(done) / elapsed.Seconds()
		remaining := total - done
		if remaining > 0 {
			etaDur := time.Duration(float64(remaining)/avgSpeed) * time.Second
			eta = formatDuration(etaDur)
		} else {
			eta = "0s"
		}
	}

	barWidth := 40
	filled := int(pct / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s  %5.1f%%  %s/%s\n",
		bar, pct, fmtInt64(done), fmtInt(s.TotalPages)))
	b.WriteString(fmt.Sprintf("  Speed %s/s  │  Peak %s/s  │  Elapsed %s  │  ETA %s\n",
		fmtInt64(int64(speed)), fmtInt64(int64(s.peakSpeed)), formatDuration(elapsed), eta))
	b.WriteString(fmt.Sprintf("  OK %s  │  Failed %s  │  Skipped %s  │  Links %s\n",
		fmtInt64(succ), fmtInt64(fail), fmtInt64(skip), fmtInt64(links)))

	return b.String()
}

// SiteExtractor extracts pages from Common Crawl for a single domain.
type SiteExtractor struct {
	config SiteConfig
	client *Client
	sdb    *SiteDB
	stats  *SiteStats
}

// NewSiteExtractor creates a new site extractor.
func NewSiteExtractor(cfg SiteConfig, client *Client, sdb *SiteDB, stats *SiteStats) *SiteExtractor {
	return &SiteExtractor{
		config: cfg,
		client: client,
		sdb:    sdb,
		stats:  stats,
	}
}

// ExtractURLsOnly stores CDX entries directly as pages (no WARC fetching).
func (se *SiteExtractor) ExtractURLsOnly(entries []CDXJEntry) {
	for _, e := range entries {
		ptr, err := CDXJToWARCPointer(e, se.config.Domain)
		if err != nil {
			continue
		}

		ts, _ := time.Parse("20060102150405", e.Timestamp)

		se.sdb.AddPage(SitePage{
			URL:          e.URL,
			StatusCode:   ptr.FetchStatus,
			ContentType:  e.Mime,
			Language:     e.Languages,
			CrawlID:      se.config.CrawlID,
			CrawledAt:    ts,
			WARCFilename: e.Filename,
			WARCOffset:   ptr.RecordOffset,
			WARCLength:   ptr.RecordLength,
		})
		se.stats.recordSuccess(0, 0, 0)
	}
}

// ExtractWithWARC fetches WARC records and extracts content/links.
func (se *SiteExtractor) ExtractWithWARC(ctx context.Context, pointers []WARCPointer, skip map[string]bool) error {
	var live []WARCPointer
	for _, p := range pointers {
		if skip != nil && skip[p.URL] {
			se.stats.recordSkip()
			continue
		}
		live = append(live, p)
	}

	if len(live) == 0 {
		return nil
	}

	// Shuffle for load distribution
	rand.Shuffle(len(live), func(i, j int) {
		live[i], live[j] = live[j], live[i]
	})

	ptrCh := make(chan WARCPointer, min(len(live), 10000))
	go func() {
		defer close(ptrCh)
		for _, p := range live {
			select {
			case ptrCh <- p:
			case <-ctx.Done():
				return
			}
		}
	}()

	workers := se.config.Workers
	if workers <= 0 {
		workers = 500
	}
	if workers > len(live) {
		workers = len(live)
	}

	g, gctx := errgroup.WithContext(ctx)
	for i := range workers {
		id := i
		g.Go(func() error {
			se.worker(gctx, id, ptrCh)
			return nil
		})
	}

	return g.Wait()
}

func (se *SiteExtractor) worker(ctx context.Context, id int, ptrCh <-chan WARCPointer) {
	maxBody := se.config.MaxBodySize
	if maxBody <= 0 {
		maxBody = 512 * 1024
	}
	storeBody := se.config.Mode == SiteModeFull
	extractLinks := se.config.Mode == SiteModeLinks || se.config.Mode == SiteModeFull

	for p := range ptrCh {
		if ctx.Err() != nil {
			return
		}

		// Fetch with retry + exponential backoff for 403/429/503
		var data []byte
		var err error
		start := time.Now()
		const maxRetries = 4
		for attempt := range maxRetries {
			data, err = se.client.FetchWARCRecord(ctx, id, p)
			if err == nil {
				break
			}
			// Only retry on rate-limit / server errors
			errMsg := err.Error()
			if !strings.Contains(errMsg, "HTTP 403") &&
				!strings.Contains(errMsg, "HTTP 429") &&
				!strings.Contains(errMsg, "HTTP 503") {
				break
			}
			if attempt < maxRetries-1 {
				backoff := time.Duration(2<<uint(attempt)) * time.Second // 2s, 4s, 8s
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return
				}
			}
		}
		fetchMs := time.Since(start).Milliseconds()

		if err != nil {
			se.stats.recordFailure()
			se.sdb.AddPage(SitePage{
				URL:          p.URL,
				CrawlID:      se.config.CrawlID,
				WARCFilename: p.WARCFilename,
				WARCOffset:   p.RecordOffset,
				WARCLength:   p.RecordLength,
				FetchTimeMs:  fetchMs,
				Error:        err.Error(),
			})
			continue
		}

		resp, err := ParseWARCRecord(data)
		if err != nil {
			se.stats.recordFailure()
			se.sdb.AddPage(SitePage{
				URL:          p.URL,
				CrawlID:      se.config.CrawlID,
				WARCFilename: p.WARCFilename,
				WARCOffset:   p.RecordOffset,
				WARCLength:   p.RecordLength,
				FetchTimeMs:  fetchMs,
				Error:        fmt.Sprintf("parse: %v", err),
			})
			continue
		}

		body := resp.Body
		contentType := p.ContentType
		if ct, ok := resp.HTTPHeaders["Content-Type"]; ok {
			contentType = ct
		}

		var title, description string
		var linkCount int

		if isHTML(contentType) && len(body) > 0 {
			title, description = ExtractPageInfo(body)

			if extractLinks {
				rawLinks := ExtractLinks(body)
				siteLinks := ResolveLinks(rawLinks, p.URL, se.config.Domain)
				if len(siteLinks) > 0 {
					se.sdb.AddLinks(siteLinks)
					linkCount = len(siteLinks)
				}
			}
		}

		bodyStr := ""
		if storeBody && isHTML(contentType) {
			if len(body) > maxBody {
				body = body[:maxBody]
			}
			bodyStr = string(body)
		}

		se.stats.recordSuccess(int64(len(data)), int64(len(resp.Body)), linkCount)

		se.sdb.AddPage(SitePage{
			URL:           p.URL,
			StatusCode:    resp.HTTPStatus,
			ContentType:   contentType,
			ContentLength: int64(len(resp.Body)),
			Title:         title,
			Description:   description,
			Language:      p.Language,
			CrawlID:       se.config.CrawlID,
			CrawledAt:     resp.Date,
			WARCFilename:  p.WARCFilename,
			WARCOffset:    p.RecordOffset,
			WARCLength:    p.RecordLength,
			Body:          bodyStr,
			FetchTimeMs:   fetchMs,
		})
	}
}

// RunSiteWithDisplay runs the site extractor with a live progress display.
func RunSiteWithDisplay(ctx context.Context, se *SiteExtractor, pointers []WARCPointer, skip map[string]bool, stats *SiteStats) error {
	displayCtx, displayCancel := context.WithCancel(ctx)
	var displayWg sync.WaitGroup
	displayWg.Add(1)
	go func() {
		defer displayWg.Done()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		var lines int
		for {
			select {
			case <-ticker.C:
				if lines > 0 {
					fmt.Printf("\033[%dA\033[J", lines)
				}
				output := stats.Render()
				fmt.Print(output)
				lines = strings.Count(output, "\n")
			case <-displayCtx.Done():
				return
			}
		}
	}()

	err := se.ExtractWithWARC(ctx, pointers, skip)

	stats.Freeze()
	displayCancel()
	displayWg.Wait()

	fmt.Print(stats.Render())
	fmt.Println()

	return err
}
