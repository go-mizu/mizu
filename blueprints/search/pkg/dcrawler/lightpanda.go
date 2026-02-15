package dcrawler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/proto"
)

// recycleInterval is the number of pages after which a Lightpanda process
// is proactively recycled to prevent memory leaks from causing crashes.
const recycleInterval = 50

// lightpandaProcess manages a single Lightpanda browser process and its rod connection.
type lightpandaProcess struct {
	cmd        *exec.Cmd
	browser    *rod.Browser
	ws         *cdp.WebSocket
	port       int
	pageCount  int // pages processed since last restart
}

// lightpandaPool manages multiple Lightpanda processes (one per worker).
type lightpandaPool struct {
	mu        sync.Mutex
	processes []*lightpandaProcess
	config    Config
	proxy     *stealthProxy
}

func newLightpandaPool(cfg Config) (*lightpandaPool, error) {
	workers := cfg.RodWorkers
	if workers <= 0 {
		workers = 8
	}

	// Start stealth proxy for Chrome header emulation + polyfill injection
	proxy, err := newStealthProxy()
	if err != nil {
		return nil, fmt.Errorf("stealth proxy: %w", err)
	}

	pool := &lightpandaPool{config: cfg, proxy: proxy}

	// Launch one process per worker
	for i := range workers {
		port := 19222 + i
		proc, err := launchLightpanda(cfg, port)
		if err != nil {
			pool.close()
			return nil, fmt.Errorf("lightpanda worker %d: %w", i, err)
		}
		pool.processes = append(pool.processes, proc)
	}

	return pool, nil
}

func launchLightpanda(cfg Config, port int) (*lightpandaProcess, error) {
	// Find lightpanda binary
	binPath, err := exec.LookPath("lightpanda")
	if err != nil {
		return nil, fmt.Errorf("lightpanda not found in PATH: %w (install from https://github.com/lightpanda-io/browser)", err)
	}

	args := []string{
		"serve",
		"--host", "127.0.0.1",
		"--port", fmt.Sprintf("%d", port),
	}

	cmd := exec.Command(binPath, args...)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start lightpanda: %w", err)
	}

	// Wait for CDP endpoint to become available
	wsURL := fmt.Sprintf("ws://127.0.0.1:%d", port)
	browser, ws, err := connectRodToLightpanda(wsURL, 5*time.Second)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return nil, fmt.Errorf("connect to lightpanda on port %d: %w", port, err)
	}

	return &lightpandaProcess{
		cmd:     cmd,
		browser: browser,
		ws:      ws,
		port:    port,
	}, nil
}

// connectRodToLightpanda connects rod to a Lightpanda CDP endpoint with retry.
// Lightpanda uses gorilla/websocket which requires a valid base64 Sec-WebSocket-Key.
// Rod's default WebSocket sends "nil" as the key, so we must inject a valid one.
func connectRodToLightpanda(wsURL string, timeout time.Duration) (*rod.Browser, *cdp.WebSocket, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Generate valid WebSocket key (16 random bytes, base64-encoded)
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			return nil, nil, fmt.Errorf("generate ws key: %w", err)
		}
		key := base64.StdEncoding.EncodeToString(buf)

		ws := &cdp.WebSocket{}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := ws.Connect(ctx, wsURL, http.Header{
			"Sec-WebSocket-Key": {key},
		})
		cancel()

		if err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		client := cdp.New()
		client.Start(ws)

		browser := rod.New().Client(client)
		if err := browser.Connect(); err != nil {
			ws.Close()
			time.Sleep(200 * time.Millisecond)
			continue
		}

		return browser, ws, nil
	}

	return nil, nil, fmt.Errorf("timeout connecting to %s after %s", wsURL, timeout)
}

func (lp *lightpandaPool) close() {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	for _, proc := range lp.processes {
		if proc.browser != nil {
			proc.browser.Close()
		}
		if proc.ws != nil {
			proc.ws.Close()
		}
		if proc.cmd != nil && proc.cmd.Process != nil {
			proc.cmd.Process.Kill()
			cmd := proc.cmd
			go cmd.Wait()
		}
	}
	lp.processes = nil
	if lp.proxy != nil {
		lp.proxy.close()
	}
}

// restart kills and relaunches a single Lightpanda process.
func (lp *lightpandaPool) restart(idx int) error {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	if idx >= len(lp.processes) {
		return fmt.Errorf("invalid process index %d", idx)
	}

	proc := lp.processes[idx]

	// Kill old process
	if proc.browser != nil {
		proc.browser.Close()
	}
	if proc.ws != nil {
		proc.ws.Close()
	}
	if proc.cmd != nil && proc.cmd.Process != nil {
		proc.cmd.Process.Kill()
		proc.cmd.Wait()
	}

	// Wait for port to be freed
	time.Sleep(500 * time.Millisecond)

	// Launch new process on same port
	newProc, err := launchLightpanda(lp.config, proc.port)
	if err != nil {
		return err
	}

	lp.processes[idx] = newProc
	return nil
}

// getPage creates a new page from the process at the given index.
func (lp *lightpandaPool) getPage(idx int) (*rod.Page, error) {
	lp.mu.Lock()
	proc := lp.processes[idx]
	b := proc.browser
	lp.mu.Unlock()

	page, err := b.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, err
	}

	if lp.config.UserAgent != "" {
		page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: lp.config.UserAgent,
		})
	}

	return page, nil
}

// shouldRecycle returns true if the process at idx has served enough pages
// and should be proactively recycled to prevent memory leaks.
func (lp *lightpandaPool) shouldRecycle(idx int) bool {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	if idx >= len(lp.processes) {
		return false
	}
	return lp.processes[idx].pageCount >= recycleInterval
}

// incrementPageCount increments the page counter for a process.
func (lp *lightpandaPool) incrementPageCount(idx int) {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	if idx < len(lp.processes) {
		lp.processes[idx].pageCount++
	}
}

// isLightpandaDead returns true if the error indicates the Lightpanda process crashed.
func isLightpandaDead(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "use of closed network connection") ||
		strings.Contains(s, "connection reset by peer") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "EOF") ||
		strings.Contains(s, "connection refused")
}

// lightpandaWorker fetches pages using Lightpanda browser.
// Each worker owns a dedicated Lightpanda process (1:1 mapping).
func (c *Crawler) lightpandaWorker(ctx context.Context, lp *lightpandaPool, workerID int) {
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
			dead := c.lightpandaFetchAndProcess(ctx, lp, item, workerID)
			if dead {
				// Restart immediately on process death — don't wait for
				// 3 consecutive errors, as each wasted attempt loses a page.
				c.stats.SetRodPhase(workerID, "restart")
				if err := lp.restart(workerID); err == nil {
					c.stats.rodRestarts.Add(1)
				}
				time.Sleep(500 * time.Millisecond)
			} else if lp.shouldRecycle(workerID) {
				// Proactive recycling: restart process after N pages to prevent
				// memory leaks from accumulating and causing crashes.
				c.stats.SetRodPhase(workerID, "recycle")
				if err := lp.restart(workerID); err == nil {
					c.stats.rodRestarts.Add(1)
				}
			}
		}
	}
}

// lightpandaFetchAndProcess fetches a page using Lightpanda.
// Returns true if the process appears dead — caller should restart.
func (c *Crawler) lightpandaFetchAndProcess(ctx context.Context, lp *lightpandaPool, item CrawlItem, workerID int) (processDead bool) {
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

	// Global deadline for this fetch
	fetchCtx, fetchCancel := context.WithTimeout(ctx, timeout+15*time.Second)
	defer fetchCancel()

	// Phase: get page from process
	c.stats.SetRodPhase(workerID, "pool")
	page, err := lp.getPage(workerID)
	if err != nil {
		if isLightpandaDead(err) {
			processDead = true
		}
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, fmt.Errorf("lightpanda page: %w", err), 0)
		return
	}

	p := page.Context(fetchCtx)
	defer func() {
		page.Close()
	}()

	// Phase: navigate
	c.stats.SetRodPhase(workerID, "nav")

	// Set up DOMContentLoaded listener BEFORE navigate
	domReady := false
	dclCh := make(chan struct{}, 1)
	go func() {
		defer func() { recover() }()
		p.EachEvent(func(e *proto.PageDomContentEventFired) (stop bool) {
			return true
		})()
		select {
		case dclCh <- struct{}{}:
		default:
		}
	}()

	// Route through stealth proxy for Chrome header emulation + polyfill injection
	navigateURL := item.URL
	if lp.proxy != nil {
		navigateURL = lp.proxy.rewriteURL(item.URL)
	}

	navRes, navErr := proto.PageNavigate{URL: navigateURL}.Call(p)
	if navErr != nil {
		if isLightpandaDead(navErr) {
			processDead = true
		}
		if ctx.Err() != nil {
			return
		}
		c.recordError(item, fmt.Errorf("navigate: %w", navErr), time.Since(start).Milliseconds())
		return
	}
	if navRes.ErrorText != "" {
		if isPermanentNavError(navRes.ErrorText) {
			c.recordError(item, fmt.Errorf("navigate: %s", navRes.ErrorText), time.Since(start).Milliseconds())
			return
		}
	}

	// Wait for DOMContentLoaded
	select {
	case <-dclCh:
		domReady = true
	case <-time.After(timeout):
		// Fallback: check readyState
		if rs, evalErr := p.Timeout(2 * time.Second).Eval(
			`() => document.readyState`); evalErr == nil && rs != nil {
			state := rs.Value.Str()
			if state == "interactive" || state == "complete" || state == "loading" {
				domReady = true
			}
		}
	case <-fetchCtx.Done():
	}

	if !domReady {
		if ctx.Err() != nil {
			return
		}
		// Try to extract partial content
		partialHTML, htmlErr := p.Timeout(5 * time.Second).HTML()
		if htmlErr != nil || len(partialHTML) < 100 {
			c.recordError(item, fmt.Errorf("navigate: timeout waiting for DOM ready"), time.Since(start).Milliseconds())
			return
		}
	}

	// No render stabilization phase — Lightpanda has no CSS rendering pipeline
	// No Cloudflare check — Lightpanda can't solve CF challenges
	// No scroll — Lightpanda doesn't support WaitRequestIdle

	// Phase: extract page content
	c.stats.SetRodPhase(workerID, "extract")
	fetchMs := time.Since(start).Milliseconds()

	extractCtx, extractCancel := context.WithTimeout(ctx, 10*time.Second)
	defer extractCancel()
	ep := page.Context(extractCtx)

	var pageTitle, pageURL string
	if pageInfo, err := ep.Info(); err == nil && pageInfo != nil {
		pageTitle = pageInfo.Title
		pageURL = pageInfo.URL
		// Extract real URL from proxy URL
		if lp.proxy != nil && pageURL != "" {
			pageURL = lp.proxy.extractRealURL(pageURL)
		}
	}

	htmlContent, err := ep.HTML()
	if err != nil {
		// Fallback: JavaScript evaluation
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

	// HTML tokenizer extraction
	meta := ExtractLinksAndMeta(body, baseURL, c.config.Domain, c.config.ExtractImages)

	// DOM-based JS extraction (V8 — works in Lightpanda)
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
	lp.incrementPageCount(workerID)
	return
}
