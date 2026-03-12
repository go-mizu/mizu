package serp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

func init() {
	RegisterRegistrar("serper", &serperRegistrar{})
}

type serperRegistrar struct{}

func (r *serperRegistrar) Register(email, password string, verbose bool) (string, error) {
	return r.registerRod(email, password, verbose)
}

func (r *serperRegistrar) registerRod(email, password string, verbose bool) (string, error) {
	l := launcher.New().Headless(false)
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	// Use stealth page to avoid Turnstile bot detection
	page := stealth.MustPage(browser)
	page = page.Timeout(180 * time.Second)

	page.MustNavigate("https://serper.dev/signup")
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("page load: %w", err)
	}
	time.Sleep(3 * time.Second)

	// Dismiss cookie banner
	buttons, _ := page.Elements(`button`)
	for _, btn := range buttons {
		text, _ := btn.Text()
		tl := strings.ToLower(text)
		if strings.Contains(tl, "accept") || strings.Contains(tl, "agree") || strings.Contains(tl, "got it") {
			_ = btn.Click(proto.InputMouseButtonLeft, 1)
			time.Sleep(500 * time.Millisecond)
			break
		}
	}
	time.Sleep(1 * time.Second)

	// Install fetch interceptor to capture API responses
	page.MustEval(`() => {
		window.__serpResp = "";
		const origFetch = window.fetch;
		window.fetch = async function(...args) {
			const resp = await origFetch.apply(this, args);
			try {
				const clone = resp.clone();
				const text = await clone.text();
				if (text.includes("apiKey") || text.includes("api_key")) {
					window.__serpResp = text;
				}
			} catch(e) {}
			return resp;
		};
	}`)

	// Fill form using native React setter
	fillInput := func(name, value string) {
		page.MustEval(`(name, value) => {
			const el = document.querySelector('input[name="' + name + '"]');
			if (el) {
				el.focus();
				const nativeSet = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
				nativeSet.call(el, value);
				el.dispatchEvent(new Event('input', {bubbles:true}));
				el.dispatchEvent(new Event('change', {bubbles:true}));
			}
		}`, name, value)
		time.Sleep(300 * time.Millisecond)
	}

	fillInput("firstName", email[:6])
	fillInput("lastName", "User")
	fillInput("email", email)
	fillInput("password", password)

	if verbose {
		fmt.Println("  form filled, waiting for Turnstile (stealth mode)...")
	}

	// Wait for Turnstile to auto-solve.
	// With stealth evasions, Turnstile should auto-solve in non-headless mode.
	// Also try clicking the widget periodically.
	turnstileSolved := false
	for i := 0; i < 45; i++ { // up to 90s
		val, err := page.Eval(`() => {
			const el = document.querySelector('input[name="cf-turnstile-response"]');
			return el ? el.value : "";
		}`)
		if err == nil {
			s := fmt.Sprint(val.Value)
			if s != "" && s != "<nil>" {
				if verbose {
					fmt.Printf("  Turnstile solved! (%ds)\n", i*2)
				}
				turnstileSolved = true
				break
			}
		}

		// Try clicking the Turnstile widget via mouse simulation (more realistic than JS click)
		if i%4 == 2 {
			// Find Turnstile iframe and click its center using rod mouse
			frames, _ := page.Elements(`iframe[src*="turnstile"], iframe[src*="challenges.cloudflare"]`)
			for _, frame := range frames {
				shape, err := frame.Shape()
				if err == nil && len(shape.Quads) > 0 {
					// Click center of iframe
					pt := shape.OnePointInside()
					page.Mouse.MustMoveTo(pt.X, pt.Y)
					time.Sleep(100 * time.Millisecond)
					page.Mouse.MustClick(proto.InputMouseButtonLeft)
					if verbose {
						fmt.Printf("  mouse-clicked Turnstile iframe at (%.0f,%.0f)\n", pt.X, pt.Y)
					}
				}
			}
			// Also try clicking .cf-turnstile div
			if div, err := page.Element(`.cf-turnstile`); err == nil {
				shape, err := div.Shape()
				if err == nil && len(shape.Quads) > 0 {
					pt := shape.OnePointInside()
					page.Mouse.MustMoveTo(pt.X, pt.Y)
					time.Sleep(100 * time.Millisecond)
					page.Mouse.MustClick(proto.InputMouseButtonLeft)
				}
			}
		}

		if verbose && i%10 == 0 && i > 0 {
			fmt.Printf("  waiting for Turnstile... (%ds)\n", i*2)
		}
		time.Sleep(2 * time.Second)
	}

	if !turnstileSolved {
		if verbose {
			fmt.Println("  Turnstile not solved after 90s — trying submit anyway")
		}
	}

	time.Sleep(500 * time.Millisecond)

	// Click submit
	submitted := false
	buttons, _ = page.Elements(`button`)
	for _, btn := range buttons {
		text, _ := btn.Text()
		tl := strings.ToLower(text)
		if strings.Contains(tl, "create") || strings.Contains(tl, "sign up") || strings.Contains(tl, "register") {
			_ = btn.Click(proto.InputMouseButtonLeft, 1)
			submitted = true
			if verbose {
				fmt.Printf("  clicked button: %s\n", strings.TrimSpace(text))
			}
			break
		}
	}
	if !submitted {
		if btn, err := page.Element(`button[type="submit"]`); err == nil {
			_ = btn.Click(proto.InputMouseButtonLeft, 1)
			submitted = true
		}
	}
	if !submitted {
		return "", fmt.Errorf("submit button not found")
	}

	if verbose {
		fmt.Println("  form submitted, waiting for response...")
	}

	// Wait for result
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)

		// Check fetch interceptor
		if val, err := page.Eval(`() => window.__serpResp || ""`); err == nil {
			s := fmt.Sprint(val.Value)
			if s != "" && s != "<nil>" {
				var resp map[string]any
				if json.Unmarshal([]byte(s), &resp) == nil {
					if key, ok := resp["apiKey"].(string); ok && key != "" {
						return key, nil
					}
				}
				if verbose {
					if len(s) > 200 {
						s = s[:200] + "..."
					}
					fmt.Printf("  intercepted response: %s\n", s)
				}
			}
		}

		info, err := page.Info()
		if err != nil {
			break
		}
		currentURL := info.URL
		if verbose && i%5 == 0 {
			fmt.Printf("  current URL: %s\n", currentURL)
		}

		if strings.Contains(currentURL, "dashboard") || strings.Contains(currentURL, "api-key") {
			if verbose {
				fmt.Println("  redirected to dashboard!")
			}
			time.Sleep(3 * time.Second)
			html, _ := page.HTML()
			if key := extractAPIKeyFromHTML(html); key != "" {
				return key, nil
			}
			page.MustNavigate("https://serper.dev/api-key")
			time.Sleep(3 * time.Second)
			html, _ = page.HTML()
			if key := extractAPIKeyFromHTML(html); key != "" {
				return key, nil
			}
			return "", nil
		}

		html, _ := page.HTML()
		if strings.Contains(html, "verify") || strings.Contains(html, "check your email") || strings.Contains(html, "confirmation") {
			if verbose {
				fmt.Println("  email verification required")
			}
			return "", nil
		}
		if strings.Contains(html, "already") || strings.Contains(html, "exists") {
			return "", fmt.Errorf("email already registered")
		}
	}

	return "", fmt.Errorf("registration timed out")
}

var apiKeyPattern = regexp.MustCompile(`[a-f0-9]{32,64}`)

func extractAPIKeyFromHTML(html string) string {
	matches := apiKeyPattern.FindAllString(html, -1)
	for _, m := range matches {
		if len(m) >= 32 && len(m) <= 64 {
			return m
		}
	}
	return ""
}

func (r *serperRegistrar) VerifyAndGetKey(email, password, emailBody string, verbose bool) (string, error) {
	// Find verification link
	linkRe := regexp.MustCompile(`https://serper\.dev[^\s"<']+verify[^\s"<']*|https://serper\.dev[^\s"<']+confirm[^\s"<']*`)
	link := linkRe.FindString(emailBody)
	if link == "" {
		genericRe := regexp.MustCompile(`https://serper\.dev/[^\s"<']+`)
		link = genericRe.FindString(emailBody)
	}

	if link != "" {
		if verbose {
			fmt.Printf("  clicking verification link: %s\n", link)
		}
		hc := &http.Client{Timeout: 20 * time.Second}
		resp, err := hc.Get(link)
		if err != nil {
			return "", fmt.Errorf("verify GET: %w", err)
		}
		resp.Body.Close()
	}

	return r.loginAndGetKey(email, password, verbose)
}

func (r *serperRegistrar) loginAndGetKey(email, password string, verbose bool) (string, error) {
	l := launcher.New().Headless(false)
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	page := stealth.MustPage(browser)
	page = page.Timeout(60 * time.Second)

	page.MustNavigate("https://serper.dev/login")
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("login page load: %w", err)
	}
	time.Sleep(3 * time.Second)

	// Fill login via JS
	page.MustEval(`(email, pass) => {
		const setVal = (name, v) => {
			const el = document.querySelector('input[name="' + name + '"]') || document.querySelector('input[type="' + name + '"]');
			if (el) {
				el.focus();
				const nativeSet = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
				nativeSet.call(el, v);
				el.dispatchEvent(new Event('input', {bubbles:true}));
				el.dispatchEvent(new Event('change', {bubbles:true}));
			}
		};
		setVal('email', email);
		setVal('password', pass);
	}`, email, password)
	time.Sleep(1 * time.Second)

	// Submit
	buttons, _ := page.Elements(`button`)
	for _, btn := range buttons {
		text, _ := btn.Text()
		tl := strings.ToLower(text)
		if strings.Contains(tl, "log in") || strings.Contains(tl, "login") || strings.Contains(tl, "sign in") {
			_ = btn.Click(proto.InputMouseButtonLeft, 1)
			break
		}
	}
	time.Sleep(5 * time.Second)

	// Navigate to API key page
	page.MustNavigate("https://serper.dev/api-key")
	time.Sleep(3 * time.Second)

	html, _ := page.HTML()
	if key := extractAPIKeyFromHTML(html); key != "" {
		return key, nil
	}

	if val, err := page.Eval(`() => document.body.innerText`); err == nil {
		if key := extractAPIKeyFromHTML(fmt.Sprint(val.Value)); key != "" {
			return key, nil
		}
	}

	return "", fmt.Errorf("could not find API key on dashboard")
}
