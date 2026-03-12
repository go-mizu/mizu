package parallel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
	"github.com/go-mizu/mizu/blueprints/search/pkg/serp"
)

func init() {
	serp.RegisterRegistrar("parallel", &registrar{})
}

type registrar struct{}

// Register on parallel.ai via email magic link.
// 1. POST magic link request to platform.parallel.ai
// 2. User receives email with login link
// 3. Click link → logged in → extract API key from dashboard
func (r *registrar) Register(email, password string, verbose bool) (string, error) {
	if verbose {
		fmt.Println("  requesting magic link from parallel.ai...")
	}

	// Try to find the Clerk/auth API endpoint for magic link
	// Parallel uses Clerk for auth — try to send magic link via their auth flow
	l := launcher.New().Headless(true)
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	page := stealth.MustPage(browser)
	page = page.Timeout(60 * time.Second)

	page.MustNavigate("https://platform.parallel.ai")
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("page load: %w", err)
	}
	time.Sleep(3 * time.Second)

	// Look for "Sign in with Email" or "Sign up" and click
	page.Eval(`() => {
		const els = document.querySelectorAll('button, a');
		for (const el of els) {
			const t = el.textContent.toLowerCase();
			if (t.includes('sign up') || t.includes('email')) {
				el.click();
				break;
			}
		}
	}`)
	time.Sleep(2 * time.Second)

	// Fill email input
	page.Eval(`(email) => {
		const inputs = document.querySelectorAll('input[type="email"], input[name="email"], input[placeholder*="email" i]');
		for (const el of inputs) {
			el.focus();
			const nativeSet = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
			nativeSet.call(el, email);
			el.dispatchEvent(new Event('input', {bubbles:true}));
			el.dispatchEvent(new Event('change', {bubbles:true}));
		}
	}`, email)
	time.Sleep(1 * time.Second)

	// Click continue/submit
	page.Eval(`() => {
		const btns = document.querySelectorAll('button');
		for (const b of btns) {
			const t = b.textContent.toLowerCase();
			if (t.includes('continue') || t.includes('sign in') || t.includes('submit') || t.includes('send')) {
				b.click();
				break;
			}
		}
	}`)
	time.Sleep(2 * time.Second)

	if verbose {
		html, _ := page.HTML()
		if strings.Contains(html, "check") || strings.Contains(html, "email") || strings.Contains(html, "link") {
			fmt.Println("  magic link sent! waiting for verification email...")
		} else {
			fmt.Println("  submitted signup form, waiting for email...")
		}
	}

	// Return empty key — email verification needed
	return "", nil
}

func (r *registrar) VerifyAndGetKey(email, password, emailBody string, verbose bool) (string, error) {
	// Find the magic link in the email
	linkRe := regexp.MustCompile(`https://[^\s"<']*parallel\.ai[^\s"<']*`)
	links := linkRe.FindAllString(emailBody, -1)

	var magicLink string
	for _, link := range links {
		if strings.Contains(link, "sign") || strings.Contains(link, "verify") ||
			strings.Contains(link, "callback") || strings.Contains(link, "ticket") ||
			strings.Contains(link, "token") {
			magicLink = link
			break
		}
	}
	if magicLink == "" && len(links) > 0 {
		magicLink = links[0] // Use first parallel.ai link
	}
	// Also check for clerk.* links (Clerk auth)
	if magicLink == "" {
		clerkRe := regexp.MustCompile(`https://[^\s"<']*clerk[^\s"<']*`)
		clerkLinks := clerkRe.FindAllString(emailBody, -1)
		if len(clerkLinks) > 0 {
			magicLink = clerkLinks[0]
		}
	}
	if magicLink == "" {
		// Try any https link in the email
		anyRe := regexp.MustCompile(`https://[^\s"<']+`)
		allLinks := anyRe.FindAllString(emailBody, -1)
		for _, l := range allLinks {
			if !strings.Contains(l, "unsubscribe") && !strings.Contains(l, "privacy") {
				magicLink = l
				break
			}
		}
	}

	if magicLink == "" {
		return "", fmt.Errorf("no magic link found in email")
	}

	if verbose {
		fmt.Printf("  magic link: %s\n", magicLink)
	}

	// Open magic link in rod browser to complete auth
	l := launcher.New().Headless(true)
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	page := stealth.MustPage(browser)
	page = page.Timeout(60 * time.Second)

	page.MustNavigate(magicLink)
	time.Sleep(5 * time.Second)

	// Should redirect to dashboard — look for API key
	for i := 0; i < 15; i++ {
		info, err := page.Info()
		if err != nil {
			break
		}
		if verbose && i%5 == 0 {
			fmt.Printf("  current URL: %s\n", info.URL)
		}

		// Check if we're on the dashboard/API keys page
		if strings.Contains(info.URL, "platform.parallel.ai") {
			// Try navigating to API keys page
			if !strings.Contains(info.URL, "key") {
				page.MustNavigate("https://platform.parallel.ai/api-keys")
				time.Sleep(3 * time.Second)
			}

			html, _ := page.HTML()
			// Look for API key patterns
			keyRe := regexp.MustCompile(`[a-zA-Z0-9]{32,64}`)
			if m := keyRe.FindString(html); m != "" {
				return m, nil
			}

			// Try extracting from JS
			val, err := page.Eval(`() => {
				const text = document.body.innerText;
				const m = text.match(/[a-zA-Z0-9_-]{32,}/);
				return m ? m[0] : '';
			}`)
			if err == nil {
				s := fmt.Sprint(val.Value)
				if len(s) >= 32 {
					return s, nil
				}
			}
		}

		time.Sleep(2 * time.Second)
	}

	// Try the API directly to see if we can get key via Clerk session
	return "", fmt.Errorf("could not extract API key from parallel.ai dashboard")
}

// tryHTTPMagicLink tries to trigger a magic link via HTTP (Clerk API).
func tryHTTPMagicLink(email string, verbose bool) error {
	// Clerk's magic link endpoint (if discoverable)
	payload, _ := json.Marshal(map[string]string{
		"email_address": email,
	})
	req, _ := http.NewRequest("POST", "https://platform.parallel.ai/api/auth/magic-link", strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if verbose {
		fmt.Printf("  magic link API: %d %s\n", resp.StatusCode, string(body)[:min(len(body), 200)])
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
