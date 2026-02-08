package dcrawler

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

// rodPool manages a headless Chrome browser and a pool of pages.
type rodPool struct {
	browser *rod.Browser
	pool    rod.Pool[rod.Page]
	config  Config
}

func newRodPool(cfg Config) (*rodPool, error) {
	l := launcher.New().
		Headless(cfg.RodHeadless).
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-features", "IsolateOrigins,site-per-process")
	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("rod launcher: %w", err)
	}
	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("rod connect: %w", err)
	}

	workers := cfg.RodWorkers
	if workers <= 0 {
		workers = 8
	}
	pool := rod.NewPagePool(workers)

	return &rodPool{
		browser: browser,
		pool:    pool,
		config:  cfg,
	}, nil
}

func (rp *rodPool) getPage() (*rod.Page, error) {
	p, err := rp.pool.Get(func() (*rod.Page, error) {
		p, err := stealth.Page(rp.browser)
		if err != nil {
			return nil, err
		}
		if rp.config.UserAgent != "" {
			p.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
				UserAgent: rp.config.UserAgent,
			})
		}
		return p, nil
	})
	return p, err
}

func (rp *rodPool) putPage(p *rod.Page) {
	rp.pool.Put(p)
}

func (rp *rodPool) close() {
	rp.pool.Cleanup(func(p *rod.Page) { p.Close() })
	rp.browser.Close()
}

// rodWorker fetches pages using headless Chrome.
func (c *Crawler) rodWorker(ctx context.Context, rp *rodPool) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-c.frontier.ch:
			if c.limiter != nil {
				if err := c.limiter.Wait(ctx); err != nil {
					return
				}
			}
			c.rodFetchAndProcess(ctx, rp, item)
		}
	}
}

func (c *Crawler) rodFetchAndProcess(ctx context.Context, rp *rodPool, item CrawlItem) {
	if c.config.MaxPages > 0 && c.claimed.Add(1) > int64(c.config.MaxPages) {
		return
	}
	c.stats.inFlight.Add(1)
	defer c.stats.inFlight.Add(-1)

	start := time.Now()

	page, err := rp.getPage()
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, err, 0)
		return
	}
	defer rp.putPage(page)

	timeout := c.config.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	err = page.Timeout(timeout).Navigate(item.URL)
	if err != nil {
		fetchMs := time.Since(start).Milliseconds()
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, fmt.Errorf("navigate: %w", err), fetchMs)
		return
	}

	err = page.Timeout(timeout).WaitLoad()
	if err != nil {
		fetchMs := time.Since(start).Milliseconds()
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, fmt.Errorf("wait load: %w", err), fetchMs)
		return
	}

	// Wait for Cloudflare challenge to resolve (title changes from "Just a moment...")
	// Poll title for up to 15 seconds
	cfDeadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(cfDeadline) {
		info, ie := page.Info()
		if ie != nil {
			break
		}
		if info.Title != "Just a moment..." {
			break
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
	}

	// Brief wait for JS rendering after challenge resolves
	page.Timeout(3 * time.Second).WaitRequestIdle(300*time.Millisecond, nil, nil, nil)()

	fetchMs := time.Since(start).Milliseconds()

	pageInfo, err := page.Info()
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, fmt.Errorf("page info: %w", err), fetchMs)
		return
	}

	htmlContent, err := page.HTML()
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, fmt.Errorf("get html: %w", err), fetchMs)
		return
	}
	body := []byte(htmlContent)

	finalURL := item.URL
	if pageInfo != nil && pageInfo.URL != "" {
		finalURL = pageInfo.URL
	}

	result := Result{
		URL:           item.URL,
		URLHash:       xxhash.Sum64String(item.URL),
		Depth:         item.Depth,
		StatusCode:    200,
		ContentType:   "text/html",
		ContentLength: int64(len(body)),
		BodyHash:      xxhash.Sum64(body),
		Title:         pageInfo.Title,
		FetchTimeMs:   fetchMs,
		CrawledAt:     time.Now(),
	}
	if finalURL != item.URL {
		result.RedirectURL = finalURL
	}

	baseURL, _ := url.Parse(finalURL)
	if baseURL == nil {
		baseURL, _ = url.Parse(item.URL)
	}

	meta := ExtractLinksAndMeta(body, baseURL, c.config.Domain)
	if meta.Description != "" {
		result.Description = meta.Description
	}
	if meta.Language != "" {
		result.Language = meta.Language
	}
	if meta.Canonical != "" {
		result.Canonical = meta.Canonical
	}
	result.LinkCount = len(meta.Links)
	c.stats.RecordLinks(len(meta.Links))

	if c.config.MaxDepth == 0 || item.Depth < c.config.MaxDepth {
		for _, link := range meta.Links {
			if link.IsInternal {
				c.frontier.TryAdd(link.TargetURL, item.Depth+1)
			}
		}
	}
	if c.config.StoreLinks && len(meta.Links) > 0 {
		c.resultDB.AddLinks(result.URLHash, meta.Links)
	}

	c.resultDB.AddPage(result)
	c.stats.RecordSuccess(result.StatusCode, int64(len(body)), fetchMs)
	c.stats.RecordDepth(item.Depth)
}
