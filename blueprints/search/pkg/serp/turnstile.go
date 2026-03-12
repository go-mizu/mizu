package serp

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
)

// turnstileHTML is a minimal page that loads the Turnstile widget.
// Based on Theyka/Turnstile-Solver technique: load just the widget on the target domain,
// let Turnstile auto-solve in a stealth browser, extract the token.
const turnstileHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Loading...</title>
  <script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async></script>
</head>
<body>
  <div class="cf-turnstile" data-sitekey="SITEKEY"></div>
  <input type="hidden" name="cf-turnstile-response" />
</body>
</html>`

// SolveTurnstile solves a Cloudflare Turnstile challenge for the given URL and sitekey.
// Uses the Theyka/Turnstile-Solver approach:
// 1. Launch a stealth browser (go-rod/stealth for evasion)
// 2. Route the target URL to a local page containing only the Turnstile widget
// 3. Turnstile auto-solves when the browser appears legitimate
// 4. Extract and return the cf-turnstile-response token
func SolveTurnstile(targetURL, sitekey string, verbose bool) (string, error) {
	l := launcher.New().Headless(false)
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	// Create page with stealth evasions
	page := stealth.MustPage(browser)
	page = page.Timeout(90 * time.Second)

	// Prepare the HTML with the sitekey
	html := strings.Replace(turnstileHTML, "SITEKEY", sitekey, 1)

	// Ensure URL ends with /
	if !strings.HasSuffix(targetURL, "/") {
		targetURL += "/"
	}

	// Route the target URL to our local page
	router := page.HijackRequests()
	router.MustAdd(targetURL+"*", func(ctx *rod.Hijack) {
		ctx.Response.SetBody(html)
	})
	go router.Run()

	if verbose {
		fmt.Printf("  Turnstile solver: navigating to %s (sitekey=%s)\n", targetURL, sitekey[:8]+"...")
	}

	page.MustNavigate(targetURL)
	time.Sleep(3 * time.Second)

	// Poll for the Turnstile response token
	for i := 0; i < 30; i++ { // up to 60s
		val, err := page.Eval(`() => {
			const el = document.querySelector('input[name="cf-turnstile-response"]');
			if (!el) return "";
			return el.value || "";
		}`)
		if err == nil {
			token := fmt.Sprint(val.Value)
			if token != "" && token != "<nil>" {
				if verbose {
					fmt.Printf("  Turnstile solved! token=%s...\n", token[:20])
				}
				return token, nil
			}
		}

		// Try clicking the Turnstile widget every few seconds
		if i%3 == 2 {
			page.Eval(`() => {
				const div = document.querySelector('.cf-turnstile');
				if (div) div.click();
			}`)
		}

		if verbose && i%10 == 0 && i > 0 {
			fmt.Printf("  Turnstile solving... (%ds)\n", i*2)
		}
		time.Sleep(2 * time.Second)
	}

	return "", fmt.Errorf("Turnstile solve timed out (60s)")
}

// ExtractTurnstileSitekey extracts the Turnstile sitekey from page HTML.
func ExtractTurnstileSitekey(html string) string {
	// Look for data-sitekey attribute
	idx := strings.Index(html, "data-sitekey=\"")
	if idx < 0 {
		// Try cf-turnstile render config
		idx = strings.Index(html, "sitekey")
		if idx < 0 {
			return ""
		}
	}
	start := idx + len("data-sitekey=\"")
	if start >= len(html) {
		return ""
	}
	end := strings.Index(html[start:], "\"")
	if end < 0 {
		return ""
	}
	return html[start : start+end]
}
