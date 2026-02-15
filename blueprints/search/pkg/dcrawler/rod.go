package dcrawler

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

// rodPool manages a headless Chrome browser and a pool of pages.
type rodPool struct {
	mu          sync.Mutex
	browser     *rod.Browser
	pool        rod.Pool[rod.Page]
	config      Config
	lastRestart time.Time
	restarts    int
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
		workers = 40
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
		// Mutex-protect browser access: tryRestart may replace rp.browser concurrently
		rp.mu.Lock()
		b := rp.browser
		rp.mu.Unlock()

		p, err := stealth.Page(b)
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
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.pool.Cleanup(func(p *rod.Page) { p.Close() })
	rp.browser.Close()
}

// tryRestart kills Chrome and relaunches it. Safe for concurrent calls:
// uses a mutex and skips if already restarted within 5s.
func (rp *rodPool) tryRestart() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if time.Since(rp.lastRestart) < 5*time.Second {
		return nil // another worker already restarted
	}

	// Close old browser + pool
	rp.pool.Cleanup(func(p *rod.Page) { p.Close() })
	rp.browser.Close()

	// Launch new Chrome
	l := launcher.New().
		Headless(rp.config.RodHeadless).
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-features", "IsolateOrigins,site-per-process")
	controlURL, err := l.Launch()
	if err != nil {
		return fmt.Errorf("rod launcher: %w", err)
	}
	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("rod connect: %w", err)
	}

	workers := rp.config.RodWorkers
	if workers <= 0 {
		workers = 40
	}
	rp.browser = browser
	rp.pool = rod.NewPagePool(workers)
	rp.lastRestart = time.Now()
	rp.restarts++
	return nil
}

// isBrowserDead returns true if the error indicates the Chrome CDP connection is broken.
func isBrowserDead(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "use of closed network connection") ||
		strings.Contains(s, "connection reset by peer") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "ERR_INTERNET_DISCONNECTED")
}

// getPageCtx gets a page from the pool, respecting the context deadline.
// If ctx expires before a page is available (e.g. Chrome is unresponsive),
// returns ctx.Err() instead of blocking forever.
func (rp *rodPool) getPageCtx(ctx context.Context) (*rod.Page, error) {
	type result struct {
		page *rod.Page
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		p, err := rp.getPage()
		ch <- result{p, err}
	}()
	select {
	case <-ctx.Done():
		// Clean up if the goroutine eventually completes
		go func() {
			if r := <-ch; r.page != nil {
				r.page.Close()
			}
		}()
		return nil, ctx.Err()
	case r := <-ch:
		return r.page, r.err
	}
}

// rodWorker fetches pages using headless Chrome.
func (c *Crawler) rodWorker(ctx context.Context, rp *rodPool, workerID int) {
	consecutiveErrors := 0
	for {
		select {
		case <-ctx.Done():
			c.stats.SetRodPhase(workerID, "")
			return
		case item := <-c.frontier.ch:
			c.stats.SetRodWorkerItem(workerID, item.URL)
			if c.limiter != nil {
				c.stats.SetRodPhase(workerID, "rate-limit")
				if err := c.limiter.Wait(ctx); err != nil {
					c.stats.SetRodPhase(workerID, "")
					return
				}
			}
			dead := c.rodFetchAndProcess(ctx, rp, item, workerID)
			if dead {
				consecutiveErrors++
				if consecutiveErrors >= 3 {
					c.stats.SetRodPhase(workerID, "restart")
					if err := rp.tryRestart(); err == nil {
						c.stats.rodRestarts.Add(1)
					}
					consecutiveErrors = 0
					time.Sleep(time.Second) // let new browser settle
				}
			} else {
				consecutiveErrors = 0
			}
		}
	}
}

// rodFetchAndProcess fetches a page using headless Chrome.
// Returns true if the browser appears dead (CDP connection broken) — caller should restart.
func (c *Crawler) rodFetchAndProcess(ctx context.Context, rp *rodPool, item CrawlItem, workerID int) (browserDead bool) {
	if c.config.MaxPages > 0 && c.claimed.Add(1) > int64(c.config.MaxPages) {
		return
	}
	c.stats.inFlight.Add(1)
	defer c.stats.inFlight.Add(-1)
	defer c.stats.SetRodPhase(workerID, "")

	start := time.Now()
	timeout := c.config.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	// Hard deadline: 30s context for the ENTIRE fetch cycle.
	fetchCtx, fetchCancel := context.WithTimeout(ctx, 30*time.Second)
	defer fetchCancel()

	// Phase: get page from pool
	c.stats.SetRodPhase(workerID, "pool")
	page, err := rp.getPageCtx(fetchCtx)
	if err != nil {
		if isBrowserDead(err) {
			browserDead = true
		}
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, fmt.Errorf("rod pool: %w", err), 0)
		return
	}

	// Context-bound page: ALL operations respect the 30s deadline.
	p := page.Context(fetchCtx)
	defer func() {
		// Reset page to about:blank to free JS memory (critical for heavy SPA sites).
		// This is both a cleanup step AND a browser health check.
		if err := page.Timeout(3 * time.Second).Navigate("about:blank"); err != nil {
			page.Close()
			if isBrowserDead(err) {
				browserDead = true
			}
		} else {
			rp.putPage(page) // page is healthy, recycle it
			browserDead = false
		}
	}()

	// Phase: navigate
	c.stats.SetRodPhase(workerID, "nav")
	if err := p.Timeout(timeout).Navigate(item.URL); err != nil {
		if isBrowserDead(err) {
			browserDead = true
		}
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, fmt.Errorf("navigate: %w", err), time.Since(start).Milliseconds())
		return
	}

	// Phase: wait for DOMContentLoaded (NOT window.load — ads never fire load)
	c.stats.SetRodPhase(workerID, "dom")
	_, _ = p.Timeout(timeout).Eval(`() => new Promise(r => {
		if (document.readyState !== 'loading') r();
		else document.addEventListener('DOMContentLoaded', r);
	})`)

	if fetchCtx.Err() != nil {
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, fmt.Errorf("rod: deadline exceeded after DOM wait"), time.Since(start).Milliseconds())
		return
	}

	// Phase: Cloudflare challenge check (poll title for up to 5s)
	c.stats.SetRodPhase(workerID, "cf-check")
	cfEnd := time.Now().Add(5 * time.Second)
	for time.Now().Before(cfEnd) && fetchCtx.Err() == nil {
		info, ie := p.Info()
		if ie != nil {
			break
		}
		if info.Title != "Just a moment..." {
			break
		}
		select {
		case <-fetchCtx.Done():
		case <-time.After(200 * time.Millisecond):
		}
	}

	if fetchCtx.Err() != nil {
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, fmt.Errorf("rod: deadline exceeded after CF check"), time.Since(start).Milliseconds())
		return
	}

	// Phase: wait for JS rendering (short idle wait, max 3s)
	c.stats.SetRodPhase(workerID, "idle")
	p.Timeout(3 * time.Second).WaitRequestIdle(300*time.Millisecond, nil, nil, nil)()

	// Scroll for infinite scroll pages (Pinterest, etc.)
	if c.config.ScrollCount > 0 && fetchCtx.Err() == nil {
		c.stats.SetRodPhase(workerID, "scroll")
		for range c.config.ScrollCount {
			if fetchCtx.Err() != nil {
				break
			}
			_, _ = p.Eval(`() => window.scrollTo(0, document.body.scrollHeight)`)
			p.Timeout(3 * time.Second).WaitRequestIdle(300*time.Millisecond, nil, nil, nil)()
			time.Sleep(200 * time.Millisecond)
		}
	}

	// Phase: extract page content
	c.stats.SetRodPhase(workerID, "extract")
	fetchMs := time.Since(start).Milliseconds()

	pageInfo, err := p.Info()
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, fmt.Errorf("page info: %w", err), fetchMs)
		return
	}

	htmlContent, err := p.HTML()
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
	if fetchCtx.Err() == nil {
		domLinks := c.extractDOMLinks(p, baseURL)
		meta.Links = append(meta.Links, domLinks...)
	}

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
	return
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
