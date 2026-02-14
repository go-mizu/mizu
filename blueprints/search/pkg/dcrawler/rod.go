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
		workers = 20
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

	timeout := c.config.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	// Retry navigation on transient Chrome errors (ERR_NETWORK_CHANGED, etc.)
	const maxRetries = 3
	var page *rod.Page
	var navErr error
	for attempt := range maxRetries {
		if ctx.Err() != nil {
			return
		}
		var err error
		page, err = rp.getPage()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.recordError(item, err, 0)
			return
		}

		err = page.Timeout(timeout).Navigate(item.URL)
		if err == nil {
			navErr = nil
			break
		}
		navErr = err
		rp.putPage(page)
		page = nil
		if ctx.Err() != nil {
			return
		}
		if attempt < maxRetries-1 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(attempt+1) * time.Second):
			}
		}
	}
	if navErr != nil {
		fetchMs := time.Since(start).Milliseconds()
		c.recordError(item, fmt.Errorf("navigate (after %d retries): %w", maxRetries, navErr), fetchMs)
		return
	}
	defer rp.putPage(page)

	// Wait for DOM to be ready (DOMContentLoaded), NOT window.load which waits for
	// all resources (ads, trackers, images). Ad-heavy sites never fire load in time.
	_, _ = page.Timeout(timeout).Eval(`() => new Promise(r => {
		if (document.readyState !== 'loading') r();
		else document.addEventListener('DOMContentLoaded', r);
	})`)

	// Wait for Cloudflare challenge to resolve (title changes from "Just a moment...")
	// Poll title for up to 8 seconds (CF challenges resolve in 2-5s)
	cfDeadline := time.Now().Add(8 * time.Second)
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
		case <-time.After(200 * time.Millisecond):
		}
	}

	// Brief wait for JS rendering after challenge resolves
	page.Timeout(5 * time.Second).WaitRequestIdle(500*time.Millisecond, nil, nil, nil)()

	// Scroll for infinite scroll pages (Pinterest, etc.)
	if c.config.ScrollCount > 0 {
		for range c.config.ScrollCount {
			if ctx.Err() != nil {
				return
			}
			_, _ = page.Eval(`() => window.scrollTo(0, document.body.scrollHeight)`)
			page.Timeout(5 * time.Second).WaitRequestIdle(500*time.Millisecond, nil, nil, nil)()
			time.Sleep(300 * time.Millisecond)
		}
	}

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

	// HTML tokenizer extraction (catches __NEXT_DATA__, JSON-LD, meta tags, inline JS)
	meta := ExtractLinksAndMeta(body, baseURL, c.config.Domain, c.config.ExtractImages)

	// DOM-based JS extraction (catches dynamically-rendered links, data-href, prefetch)
	domLinks := c.extractDOMLinks(page, baseURL)
	meta.Links = append(meta.Links, domLinks...)

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

// domLinkResult is the JSON structure returned by the DOM link extraction script.
type domLinkResult struct {
	URL  string `json:"url"`
	Text string `json:"text"`
	Rel  string `json:"rel"`
}

// extractDOMLinks runs JavaScript in the browser to extract links from the rendered DOM.
// This catches dynamically-generated links that don't exist in the raw HTML source.
func (c *Crawler) extractDOMLinks(page *rod.Page, baseURL *url.URL) []Link {
	result, err := page.Timeout(5 * time.Second).Eval(`() => {
		const links = [];
		const seen = new Set();
		// All anchor hrefs from rendered DOM
		document.querySelectorAll('a[href]').forEach(a => {
			if (a.href && !seen.has(a.href)) {
				seen.add(a.href);
				links.push({url: a.href, text: (a.textContent || '').trim().slice(0, 200), rel: a.rel || ''});
			}
		});
		// data-href / data-url attributes (React/Vue/Angular patterns)
		document.querySelectorAll('[data-href],[data-url],[data-link]').forEach(el => {
			const u = el.dataset.href || el.dataset.url || el.dataset.link;
			if (u && !seen.has(u)) {
				seen.add(u);
				links.push({url: u, text: '', rel: 'data-attr'});
			}
		});
		// Next.js client-side navigation links
		document.querySelectorAll('link[rel="prefetch"][href],link[rel="preload"][href][as="fetch"]').forEach(l => {
			if (l.href && !seen.has(l.href)) {
				seen.add(l.href);
				links.push({url: l.href, text: '', rel: l.rel});
			}
		});
		return links;
	}`)
	if err != nil {
		return nil
	}

	var domLinks []domLinkResult
	if err := result.Value.Unmarshal(&domLinks); err != nil {
		return nil
	}

	var links []Link
	for _, dl := range domLinks {
		if dl.URL == "" {
			continue
		}
		resolved := resolveURL(dl.URL, baseURL)
		if resolved == "" {
			continue
		}
		links = append(links, Link{
			TargetURL:  resolved,
			AnchorText: truncate(normalizeText(dl.Text), 200),
			Rel:        "dom-" + dl.Rel,
			IsInternal: isInternalURL(resolved, c.config.Domain),
		})
	}
	return links
}
