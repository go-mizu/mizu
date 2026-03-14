package goodread

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"golang.org/x/time/rate"
)

// fingerprintOverrideJS patches headless-Chrome-detectable properties.
const fingerprintOverrideJS = `(function(){
try{Object.defineProperty(screen,'width',{get:()=>1920});
Object.defineProperty(screen,'height',{get:()=>1080});
Object.defineProperty(screen,'availWidth',{get:()=>1920});
Object.defineProperty(screen,'availHeight',{get:()=>1040});
Object.defineProperty(screen,'colorDepth',{get:()=>24});
Object.defineProperty(screen,'pixelDepth',{get:()=>24});}catch(e){}
try{Object.defineProperty(window,'outerWidth',{get:()=>1920});
Object.defineProperty(window,'outerHeight',{get:()=>1080});}catch(e){}
try{Object.defineProperty(window,'devicePixelRatio',{get:()=>1});}catch(e){}
})()`

// detectChromeBin finds a Chrome/Chromium binary: $CHROME_BIN → ~/bin/chromium → system paths.
func detectChromeBin() string {
	if p := os.Getenv("CHROME_BIN"); p != "" {
		return p
	}
	candidates := []string{
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append([]string{
			home + "/bin/chromium",
			home + "/.local/bin/chromium",
		}, candidates...)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// RodClient fetches Goodreads pages using a headless Chrome browser (go-rod).
// It bypasses bot-detection that slows plain HTTP responses.
type RodClient struct {
	browser *rod.Browser
	limiter *rate.Limiter
	timeout time.Duration
}

// NewRodClient launches a headless Chrome instance ready for Goodreads scraping.
func NewRodClient(cfg Config) (*RodClient, error) {
	l := launcher.New().
		Headless(true).
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-dev-shm-usage", "").
		Set("no-sandbox", "").
		Set("window-size", "1920,1080").
		Set("lang", "en-US").
		Set("disable-gpu", "").
		Set("disable-background-networking", "").
		Set("disable-extensions", "").
		Set("disable-sync", "")

	if bin := detectChromeBin(); bin != "" {
		l = l.Bin(bin)
	}

	url, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch chrome: %w", err)
	}

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("connect to chrome: %w", err)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &RodClient{
		browser: browser,
		limiter: newLimiter(cfg),
		timeout: timeout,
	}, nil
}

// FetchHTMLTimed is an alias for FetchHTML for use in benchmarks.
func (c *RodClient) FetchHTMLTimed(ctx context.Context, url string) (*goquery.Document, int, error) {
	return c.FetchHTML(ctx, url)
}

// Close shuts down the headless browser.
func (c *RodClient) Close() {
	if c.browser != nil {
		c.browser.Close()
	}
}

// FetchHTML fetches a URL using headless Chrome and returns a parsed goquery document.
func (c *RodClient) FetchHTML(ctx context.Context, url string) (*goquery.Document, int, error) {
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, 0, err
		}
	}

	pageCtx, cancel := context.WithTimeout(ctx, c.timeout+10*time.Second)
	defer cancel()

	page, err := stealth.Page(c.browser)
	if err != nil {
		return nil, 0, fmt.Errorf("rod new page: %w", err)
	}
	defer page.Close()

	// Inject fingerprint overrides before any page scripts run.
	_, injectErr := proto.PageAddScriptToEvaluateOnNewDocument{Source: fingerprintOverrideJS}.Call(page)
	if injectErr != nil {
		return nil, 0, fmt.Errorf("inject fingerprint js: %w", injectErr)
	}

	p := page.Context(pageCtx)

	// Navigate and wait for DOM content.
	if err := p.Timeout(c.timeout).Navigate(url); err != nil {
		return nil, 0, fmt.Errorf("navigate: %w", err)
	}
	if err := p.Timeout(c.timeout).WaitLoad(); err != nil {
		// Non-fatal: page might be usable even if load event didn't fire cleanly.
		_ = err
	}

	// Check for login redirect.
	currentURL, _ := p.Info()
	if currentURL != nil {
		if strings.Contains(currentURL.URL, "/sign_in") || strings.Contains(currentURL.URL, "/user/login") {
			return nil, 401, fmt.Errorf("login required: redirected to %s", currentURL.URL)
		}
	}

	html, err := p.HTML()
	if err != nil {
		return nil, 0, fmt.Errorf("get html: %w", err)
	}

	// Detect soft-redirect in rendered HTML.
	if strings.Contains(html, "You are being") {
		if redir := extractHTMLRedirect(html); redir != "" && redir != url {
			if strings.Contains(redir, "/sign_in") || strings.Contains(redir, "/login") {
				return nil, 401, fmt.Errorf("login required (soft redirect)")
			}
			// Follow the redirect in the same page.
			if err := p.Timeout(c.timeout).Navigate(redir); err != nil {
				return nil, 0, fmt.Errorf("follow soft redirect: %w", err)
			}
			if err := p.Timeout(c.timeout).WaitLoad(); err != nil {
				_ = err
			}
			html, err = p.HTML()
			if err != nil {
				return nil, 0, fmt.Errorf("get html after redirect: %w", err)
			}
		}
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, 0, fmt.Errorf("parse html: %w", err)
	}

	return doc, 200, nil
}
