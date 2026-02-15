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
		if rp.config.RodBlockResources {
			setupResourceBlocking(p)
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

// setupResourceBlocking configures Chrome to block heavy resources (images, fonts, CSS, etc.)
// for faster page loads. Only documents, scripts, and data requests are allowed through.
// This dramatically reduces page load time and Chrome resource usage.
func setupResourceBlocking(page *rod.Page) {
	router := page.HijackRequests()
	block := func(ctx *rod.Hijack) {
		ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
	}
	_ = router.Add("*", proto.NetworkResourceTypeImage, block)
	_ = router.Add("*", proto.NetworkResourceTypeFont, block)
	_ = router.Add("*", proto.NetworkResourceTypeStylesheet, block)
	_ = router.Add("*", proto.NetworkResourceTypeMedia, block)
	_ = router.Add("*", proto.NetworkResourceTypeWebSocket, block)
	_ = router.Add("*", proto.NetworkResourceTypePrefetch, block)
	go router.Run()
}

// isPermanentNavError returns true for Chrome navigation errors that will never succeed on retry.
func isPermanentNavError(errorText string) bool {
	return strings.Contains(errorText, "ERR_NAME_NOT_RESOLVED") ||
		strings.Contains(errorText, "ERR_CONNECTION_REFUSED") ||
		strings.Contains(errorText, "ERR_CERT_") ||
		strings.Contains(errorText, "ERR_SSL_") ||
		strings.Contains(errorText, "ERR_INVALID_URL") ||
		strings.Contains(errorText, "ERR_TOO_MANY_REDIRECTS") ||
		strings.Contains(errorText, "ERR_BLOCKED_BY_RESPONSE")
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
		timeout = 30 * time.Second
	}

	// Global deadline: navigate timeout + buffer for render wait + extraction.
	fetchCtx, fetchCancel := context.WithTimeout(ctx, timeout+30*time.Second)
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

	// Context-bound page: ALL operations respect the global deadline.
	p := page.Context(fetchCtx)
	defer func() {
		// Reset page to about:blank to free JS memory (critical for heavy SPA sites).
		// This is both a cleanup step AND a browser health check.
		if err := page.Timeout(2 * time.Second).Navigate("about:blank"); err != nil {
			page.Close()
			if isBrowserDead(err) {
				browserDead = true
			}
		} else {
			rp.putPage(page) // page is healthy, recycle it
			browserDead = false
		}
	}()

	// Phase: navigate using Chrome's native DOMContentLoaded event.
	// Previous approach: polling readyState with Eval every 150ms.
	// Problem: each Eval forces Chrome to context-switch, stealing CPU from page rendering
	// and actually CAUSING timeouts (8 tabs × Eval every 150ms = 53 Eval/s overhead).
	// New approach: listen for DOMContentLoaded event (zero CPU overhead, Chrome notifies us).
	c.stats.SetRodPhase(workerID, "nav")

	// Set up DOMContentLoaded listener BEFORE sending navigate command.
	// This ensures we never miss the event even if the page loads instantly.
	domReady := false
	dclCh := make(chan struct{}, 1)
	go func() {
		defer func() { recover() }() // safety: don't crash if Chrome disconnects
		p.EachEvent(func(e *proto.PageDomContentEventFired) (stop bool) {
			return true
		})()
		select {
		case dclCh <- struct{}{}:
		default:
		}
	}()

	// Send the navigate command — Chrome starts loading immediately.
	navRes, navErr := proto.PageNavigate{URL: item.URL}.Call(p)
	if navErr != nil {
		if isBrowserDead(navErr) {
			browserDead = true
			return
		}
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, fmt.Errorf("navigate: %w", navErr), time.Since(start).Milliseconds())
		return
	}
	navErrorText := ""
	if navRes.ErrorText != "" {
		navErrorText = navRes.ErrorText
		if isPermanentNavError(navErrorText) {
			c.recordError(item, fmt.Errorf("navigate: %s", navErrorText), time.Since(start).Milliseconds())
			return
		}
	}

	// Wait for DOMContentLoaded event — zero CPU overhead, Chrome does all the work.
	select {
	case <-dclCh:
		domReady = true
	case <-time.After(timeout):
		// Event didn't fire within timeout. Do one final readyState check —
		// maybe we missed the event or Chrome is being slow.
		if rs, evalErr := p.Timeout(2 * time.Second).Eval(
			`() => document.readyState`); evalErr == nil && rs != nil {
			state := rs.Value.Str()
			if state == "interactive" || state == "complete" || state == "loading" {
				// Page has navigated (even if still loading) — accept it.
				domReady = true
			}
		}
	case <-fetchCtx.Done():
	}

	if !domReady {
		if ctx.Err() != nil {
			return
		}
		// Last resort: try to extract whatever HTML Chrome has.
		partialHTML, htmlErr := p.Timeout(5 * time.Second).HTML()
		if htmlErr != nil || len(partialHTML) < 100 {
			errMsg := "navigate: timeout waiting for DOM ready"
			if navErrorText != "" {
				errMsg = "navigate: " + navErrorText
			}
			c.recordError(item, fmt.Errorf("%s", errMsg), time.Since(start).Milliseconds())
			return
		}
		// Substantial server-rendered content — proceed with partial extraction.
	}

	// Phase: wait for DOM to stabilize (React/Next.js hydration + render).
	// Polls document.body.innerHTML.length: stable for 600ms = hydration complete.
	c.stats.SetRodPhase(workerID, "render")
	_, _ = p.Timeout(5 * time.Second).Eval(`() => new Promise((resolve) => {
		const afterDOM = () => {
			let lastLen = document.body ? document.body.innerHTML.length : 0;
			let stable = 0;
			const check = () => {
				const len = document.body ? document.body.innerHTML.length : 0;
				if (len === lastLen) {
					stable++;
					if (stable >= 3) { resolve(); return; }
				} else {
					stable = 0;
					lastLen = len;
				}
				setTimeout(check, 200);
			};
			setTimeout(check, 300);
		};
		if (document.readyState !== 'loading') afterDOM();
		else document.addEventListener('DOMContentLoaded', afterDOM);
	})`)

	// Render wait timeout is NOT fatal — the DOM is already "interactive".
	// Just skip optional post-render steps (CF check, scroll) if deadline expired.

	// Cloudflare challenge check — only poll if CF challenge detected and deadline not expired.
	if fetchCtx.Err() == nil {
		if info, ie := p.Info(); ie == nil && info.Title == "Just a moment..." {
			c.stats.SetRodPhase(workerID, "cf-check")
			cfEnd := time.Now().Add(3 * time.Second)
			for time.Now().Before(cfEnd) && fetchCtx.Err() == nil {
				select {
				case <-fetchCtx.Done():
				case <-time.After(200 * time.Millisecond):
				}
				if info, ie := p.Info(); ie != nil || info.Title != "Just a moment..." {
					break
				}
			}
		}
	}

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

	// Phase: extract page content.
	// Use a fresh 10s context for extraction — the global fetchCtx may have expired
	// during render wait, but the page content is still in Chrome's memory.
	c.stats.SetRodPhase(workerID, "extract")
	fetchMs := time.Since(start).Milliseconds()

	extractCtx, extractCancel := context.WithTimeout(ctx, 10*time.Second)
	defer extractCancel()
	ep := page.Context(extractCtx)

	// Page info: fallback to empty title if it fails (don't abandon the page).
	var pageTitle, pageURL string
	if pageInfo, err := ep.Info(); err == nil && pageInfo != nil {
		pageTitle = pageInfo.Title
		pageURL = pageInfo.URL
	}

	// HTML extraction with fallback: try p.HTML() first, then Eval as backup.
	htmlContent, err := ep.HTML()
	if err != nil {
		// Fallback: extract via JavaScript evaluation
		if rs, evalErr := ep.Timeout(5 * time.Second).Eval(
			`() => document.documentElement.outerHTML`); evalErr == nil && rs != nil {
			htmlContent = rs.Value.Str()
		}
	}
	if htmlContent == "" {
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, fmt.Errorf("get html: empty content"), fetchMs)
		return
	}
	body := []byte(htmlContent)

	finalURL := item.URL
	if pageURL != "" {
		finalURL = pageURL
	}

	result := Result{
		URL:           item.URL,
		URLHash:       xxhash.Sum64String(item.URL),
		Depth:         item.Depth,
		StatusCode:    200,
		ContentType:   "text/html",
		ContentLength: int64(len(body)),
		BodyHash:      xxhash.Sum64(body),
		Title:         pageTitle,
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
	// Uses extractCtx (fresh 10s deadline) — fetchCtx may have expired during render wait.
	if extractCtx.Err() == nil {
		domLinks := c.extractDOMLinks(ep, baseURL)
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
// Enhanced for Next.js/React SPAs: extracts from rendered anchors, ARIA roles, data attrs,
// Next.js __NEXT_DATA__ props, form actions, and link preloads.
func (c *Crawler) extractDOMLinks(page *rod.Page, baseURL *url.URL) []Link {
	result, err := page.Timeout(3 * time.Second).Eval(`() => {
		const links = [];
		const seen = new Set();
		const add = (url, text, rel) => {
			if (url && !seen.has(url)) {
				seen.add(url);
				links.push({url, text: (text || '').trim().slice(0, 200), rel: rel || ''});
			}
		};

		// All anchor hrefs from rendered DOM (covers Next.js <Link>, React Router <Link>, etc.)
		document.querySelectorAll('a[href]').forEach(a => {
			add(a.href, a.textContent, a.rel);
		});

		// data-href / data-url attributes (React/Vue/Angular patterns)
		document.querySelectorAll('[data-href],[data-url],[data-link]').forEach(el => {
			add(el.dataset.href || el.dataset.url || el.dataset.link, '', 'data-attr');
		});

		// ARIA role=link elements (React sometimes uses these for navigable non-anchor elements)
		document.querySelectorAll('[role="link"]').forEach(el => {
			const u = el.getAttribute('href') || el.dataset.href || el.dataset.url;
			if (u) add(u, el.textContent, 'role-link');
		});

		// Next.js prefetch/preload hints (client-side navigation)
		document.querySelectorAll('link[rel="prefetch"][href],link[rel="preload"][href][as="fetch"]').forEach(l => {
			add(l.href, '', l.rel);
		});

		// Alternate/hreflang links (localization)
		document.querySelectorAll('link[rel="alternate"][href]').forEach(l => {
			add(l.href, '', 'alternate');
		});

		// Form actions
		document.querySelectorAll('form[action]').forEach(f => {
			if (f.action && f.action !== location.href) add(f.action, '', 'form');
		});

		// Next.js __NEXT_DATA__: walk props for internal URL paths
		const nd = document.getElementById('__NEXT_DATA__');
		if (nd) {
			try {
				const data = JSON.parse(nd.textContent);
				const walk = (obj, depth) => {
					if (depth > 8 || !obj) return;
					if (typeof obj === 'string') {
						if (obj.length > 1 && obj.length < 300 && obj.startsWith('/') &&
							/^\/[a-zA-Z]/.test(obj) &&
							!/\.(js|css|png|jpg|svg|woff|map)$/i.test(obj) &&
							!obj.startsWith('/_next/') && !obj.startsWith('/_nuxt/')) {
							add(location.origin + obj, '', 'next-data');
						}
					} else if (Array.isArray(obj)) {
						for (const item of obj) walk(item, depth + 1);
					} else if (typeof obj === 'object') {
						for (const val of Object.values(obj)) walk(val, depth + 1);
					}
				};
				walk(data.props, 0);
				// Extract page route itself
				if (data.page && data.page !== '/') add(location.origin + data.page, '', 'next-page');
			} catch(e) {}
		}

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
