package dcrawler

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/DataDog/zstd"
	"github.com/cespare/xxhash/v2"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

// networkInterceptJS is injected before page scripts to capture URLs from XHR/fetch responses.
// Catches article URLs loaded via API calls (news feeds, infinite scroll, AJAX pagination).
// URLs are stored in window.__xhrURLs Set for later retrieval.
const networkInterceptJS = `(function(){
var S=new Set();window.__xhrURLs=S;
var R=/https?:\/\/[^\s"'<>]+/g;
var Q=/["'](\/[a-zA-Z][^"'\s<>]{3,300})["']/g;
var skip=/\.(js|css|png|jpg|jpeg|gif|svg|ico|woff2?|ttf|eot|map|webp|avif|mp[34]|wav)$/i;
var skipP=/^\/_|^\/static\/|^\/assets\/|^\/webpack\/|^\/chunks\//;
function X(t){if(!t||t.length>1000000)return;t=t.replace(/\\\//g,'/');R.lastIndex=0;for(var m;(m=R.exec(t))!==null;){var u=m[0].replace(/[),;.:!?'"]+$/,'');if(u.length>10&&u.length<2000)S.add(u)}Q.lastIndex=0;for(var m2;(m2=Q.exec(t))!==null;){var p=m2[1];if(!skip.test(p)&&!skipP.test(p))S.add(location.origin+p)}}
var F=window.fetch;if(F){window.fetch=function(){return F.apply(this,arguments).then(function(r){try{var c=r.headers.get('content-type')||'';if(c.includes('json')||c.includes('html')||c.includes('text'))r.clone().text().then(X).catch(function(){})}catch(e){}return r})}}
var P=XMLHttpRequest.prototype,O=P.send;P.send=function(){this.addEventListener('load',function(){try{var c=this.getResponseHeader('content-type')||'';if(c.includes('json')||c.includes('html')||c.includes('text'))X(this.responseText)}catch(e){}});return O.apply(this,arguments)}
})()`

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
	if c.config.MaxPages > 0 && c.stats.success.Load() >= int64(c.config.MaxPages) {
		return
	}
	// Adaptive filter: skip URLs from classes with >85% block rate
	if c.isURLClassBlocked(item.URL) {
		c.stats.RecordSkipped()
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

	// Hard deadline: force-close page if worker exceeds total timeout.
	// Chrome operations can hang beyond context cancellation (blocking system calls),
	// so we need a goroutine that kills the page from outside.
	hardDeadline := timeout + 45*time.Second // 15s beyond fetchCtx for cleanup
	pageClosed := make(chan struct{})
	var forceClosePage func()

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

	// Start hard-deadline watchdog: if this page is still alive after hardDeadline,
	// force-close it. This unblocks any goroutine stuck in Chrome operations.
	forceClosePage = func() { page.Close() }
	go func() {
		select {
		case <-time.After(hardDeadline):
			page.Close() // force-kill: unblocks any stuck Chrome call
		case <-pageClosed:
			// Normal cleanup completed before hard deadline
		}
	}()

	// Context-bound page: ALL operations respect the global deadline.
	p := page.Context(fetchCtx)
	defer func() {
		// Reset page to about:blank to free JS memory (critical for heavy SPA sites).
		// This is both a cleanup step AND a browser health check.
		if err := page.Timeout(2 * time.Second).Navigate("about:blank"); err != nil {
			if forceClosePage != nil {
				forceClosePage()
			}
			if isBrowserDead(err) {
				browserDead = true
			}
		} else {
			rp.putPage(page) // page is healthy, recycle it
			browserDead = false
		}
		close(pageClosed) // signal watchdog to stop
	}()

	// Inject XHR/fetch interceptor to capture URLs from API responses (news feeds, AJAX pagination).
	scriptRes, _ := proto.PageAddScriptToEvaluateOnNewDocument{Source: networkInterceptJS}.Call(p)

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

	// WAF/bot challenge detection — Cloudflare, CloudFront, and Tencent Edge One.
	// Chrome can solve these automatically if given time to execute the challenge JS.
	if fetchCtx.Err() == nil {
		isChallenge := false
		if info, ie := p.Info(); ie == nil && info != nil {
			titleLower := strings.ToLower(info.Title)
			if info.Title == "Just a moment..." {
				isChallenge = true // Cloudflare
			} else if strings.Contains(titleLower, "the request could not be satisfied") {
				// CloudFront WAF — not solvable via JS, record as blocked immediately
				fetchMs := time.Since(start).Milliseconds()
				c.recordBlocked(item, "CloudFront WAF block", fetchMs)
				return
			}
		}
		// Tencent WAF: page contains "EO_Bot_Ssid" (cookie challenge) or redirects to waf.tencent.com.
		if !isChallenge {
			if wafCheck, wErr := p.Timeout(1 * time.Second).Eval(
				`() => document.documentElement.innerHTML.includes('EO_Bot_Ssid') || document.documentElement.innerHTML.includes('waf.tencent.com')`); wErr == nil && wafCheck != nil && wafCheck.Value.Bool() {
				isChallenge = true
			}
		}
		if isChallenge {
			c.stats.SetRodPhase(workerID, "cf-check")
			challengeEnd := time.Now().Add(8 * time.Second)
			for time.Now().Before(challengeEnd) && fetchCtx.Err() == nil {
				select {
				case <-fetchCtx.Done():
				case <-time.After(500 * time.Millisecond):
				}
				// Check if challenge resolved: page grew beyond challenge size.
				if htmlLen, hErr := p.Timeout(1 * time.Second).Eval(`() => document.documentElement.outerHTML.length`); hErr == nil && htmlLen != nil {
					if htmlLen.Value.Int() > 5000 {
						time.Sleep(500 * time.Millisecond)
						break
					}
				}
			}
		}
	}

	// Early blocked-page detection: check title BEFORE scrolling to avoid wasting
	// 10-15s scrolling a blocked page. Real pages have descriptive titles after render;
	// anti-bot pages keep the URL path as title.
	if fetchCtx.Err() == nil {
		if info, ie := p.Info(); ie == nil && info != nil && info.Title != "" {
			if blocked, reason := isBlockedPage(nil, info.Title, item.URL); blocked {
				fetchMs := time.Since(start).Milliseconds()
				c.recordBlocked(item, reason, fetchMs)
				return
			}
		}
	}

	// Scroll for infinite scroll pages (Pinterest, news feeds, etc.)
	// Uses early termination: stops when BOTH page height AND XHR URL count
	// stop growing. Checking XHR count is critical for sites like news.qq.com
	// where API responses load article URLs without changing visible page height.
	if c.config.ScrollCount > 0 && fetchCtx.Err() == nil {
		c.stats.SetRodPhase(workerID, "scroll")
		var lastHeight, lastXHRCount int
		noGrowth := 0
		for range c.config.ScrollCount {
			if fetchCtx.Err() != nil {
				break
			}
			_, _ = p.Eval(`() => window.scrollTo(0, document.body.scrollHeight)`)
			p.Timeout(1500 * time.Millisecond).WaitRequestIdle(200*time.Millisecond, nil, nil, nil)()
			time.Sleep(100 * time.Millisecond)
			// Check if scroll produced new content (height) or new URLs (XHR interceptor)
			heightGrew, xhrGrew := false, false
			if heightRes, hErr := p.Timeout(1 * time.Second).Eval(`() => document.body.scrollHeight`); hErr == nil && heightRes != nil {
				h := heightRes.Value.Int()
				if h > 0 && h != lastHeight {
					heightGrew = true
				}
				lastHeight = h
			}
			if xhrRes, xErr := p.Timeout(1 * time.Second).Eval(`() => (window.__xhrURLs || {size:0}).size`); xErr == nil && xhrRes != nil {
				xc := xhrRes.Value.Int()
				if xc > lastXHRCount {
					xhrGrew = true
				}
				lastXHRCount = xc
			}
			if heightGrew || xhrGrew {
				noGrowth = 0
			} else {
				noGrowth++
				if noGrowth >= 3 {
					break // Neither height nor XHR URLs growing — done
				}
			}
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

	// Detect blocked/anti-bot responses that return HTTP 200 but no real content.
	// Common patterns: very small pages (<500 bytes), "Access Restricted" pages,
	// pages whose title is just the URL path (QQ anti-bot placeholder).
	if blocked, reason := isBlockedPage(body, pageTitle, item.URL); blocked {
		c.recordBlocked(item, reason, fetchMs)
		return
	}

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
	if c.config.StoreBody {
		if compressed, err := zstd.Compress(nil, body); err == nil {
			result.BodyCompressed = compressed
		}
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

	// Collect URLs discovered from XHR/fetch response bodies (injected interceptor).
	if extractCtx.Err() == nil {
		if xhrRes, xhrErr := ep.Timeout(2 * time.Second).Eval(`() => Array.from(window.__xhrURLs || [])`); xhrErr == nil {
			var xhrURLs []string
			if xhrRes.Value.Unmarshal(&xhrURLs) == nil {
				for _, rawURL := range xhrURLs {
					resolved := resolveURL(rawURL, baseURL)
					if resolved != "" {
						meta.Links = append(meta.Links, Link{
							TargetURL:  resolved,
							Rel:        "xhr",
							IsInternal: isInternalURL(resolved, c.config.Domain),
						})
					}
				}
			}
		}
	}

	// Remove injected script to prevent accumulation across page reuses.
	if scriptRes != nil {
		proto.PageRemoveScriptToEvaluateOnNewDocument{Identifier: scriptRes.Identifier}.Call(page)
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

	hasAliases := len(c.config.DomainAliases) > 0
	if c.config.MaxDepth == 0 || item.Depth < c.config.MaxDepth {
		for _, link := range meta.Links {
			// With domain aliases, also try non-internal links — the frontier's
			// isSameDomain check handles alias matching correctly.
			if link.IsInternal || hasAliases {
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
	c.recordURLClass(item.URL, false)
	return
}

// domLinkResult is the JSON structure returned by the DOM link extraction script.
type domLinkResult struct {
	URL  string `json:"url"`
	Text string `json:"text"`
	Rel  string `json:"rel"`
}

// isBlockedPage detects anti-bot responses that return HTTP 200 but contain no real content.
// Returns true and the reason if the page appears to be blocked.
func isBlockedPage(body []byte, title, requestURL string) (bool, string) {
	bodyLen := len(body)

	// Very small pages (< 500 bytes) with HTTP 200 are almost certainly anti-bot placeholders.
	// Real pages are at least a few KB. Skip when body is nil (early title-only check).
	if body != nil && bodyLen < 500 {
		return true, fmt.Sprintf("empty response (%d bytes)", bodyLen)
	}

	// QQ/Tencent anti-bot: title matches the URL path or hostname+path.
	// Browser mode returns titles like "news.qq.com/rain/a/20260218A0xxxx" (hostname+path),
	// while raw HTML has titles like "/rain/a/20260218A0xxxx" (path only).
	// Normal pages have descriptive titles like "Article Title_腾讯新闻".
	if title != "" {
		// Extract path from request URL for comparison
		if u, err := url.Parse(requestURL); err == nil && u.Path != "" && u.Path != "/" {
			urlPath := u.Path                       // e.g., "/rain/a/20260218A0xxxx"
			hostPath := u.Host + u.Path              // e.g., "news.qq.com/rain/a/20260218A0xxxx"
			if title == urlPath || title == urlPath[1:] || title == hostPath {
				return true, "title matches URL path (anti-bot placeholder)"
			}
		}
	}

	// Check for known WAF/anti-bot signatures in small pages (< 10KB).
	// Don't scan large pages — they're likely real content.
	if bodyLen < 10_000 {
		content := strings.ToLower(string(body))
		wafSignatures := []string{
			"access restricted",                     // Generic access block
			"限制访问",                                  // Chinese: access restricted
			"访问受限",                                  // Chinese: access limited
			"eo_bot_ssid",                           // Tencent Edge One WAF
			"waf.tencent.com",                       // Tencent WAF redirect
			"captcha-delivery",                      // Generic CAPTCHA
			"cf-browser-verification",               // Cloudflare
			"generated by cloudfront",               // AWS CloudFront WAF
			"the request could not be satisfied",    // CloudFront block page
			"request blocked",                       // CloudFront / generic WAF
		}
		for _, sig := range wafSignatures {
			if strings.Contains(content, sig) {
				return true, fmt.Sprintf("WAF signature: %s", sig)
			}
		}
	}

	// Title-based WAF detection (works even without body, e.g. early title check)
	if title != "" {
		titleLower := strings.ToLower(title)
		wafTitles := []string{
			"the request could not be satisfied", // CloudFront
			"access denied",                      // Generic WAF
			"403 forbidden",                      // Generic WAF
			"attention required",                 // Cloudflare
		}
		for _, wt := range wafTitles {
			if strings.Contains(titleLower, wt) {
				return true, fmt.Sprintf("WAF title: %s", title)
			}
		}
	}

	return false, ""
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

		// Performance resource entries: discover URLs from XHR/fetch API calls
		try {
			performance.getEntriesByType('resource').forEach(e => {
				if (e.initiatorType === 'xmlhttprequest' || e.initiatorType === 'fetch') {
					add(e.name, '', 'perf-' + e.initiatorType);
				}
			});
		} catch(ex) {}

		// onclick handlers with URL patterns (common in Chinese news sites)
		document.querySelectorAll('[onclick]').forEach(el => {
			const oc = el.getAttribute('onclick') || '';
			const m = oc.match(/(?:location\.href|window\.open|location\.replace)\s*[=(]\s*['"]([^'"]+)['"]/);
			if (m && m[1]) add(m[1], el.textContent, 'onclick');
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
