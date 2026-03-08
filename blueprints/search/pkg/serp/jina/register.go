package jina

import (
	"bytes"
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
	serp.RegisterRegistrar("jina", &registrar{})
}

type registrar struct{}

var jinaKeyRe = regexp.MustCompile(`jina_[a-f0-9]{32}[a-zA-Z0-9_-]+`)

const firebaseAPIKey = "AIzaSyAwnOlP9TIbmc672C695yWwtiLhK1rTAKY"

// Register creates a Firebase account, then logs in to jina.ai via the
// "Continue with your Email" flow, and extracts the API key.
func (r *registrar) Register(email, password string, verbose bool) (string, error) {
	// Step 1: Create Firebase account via REST API
	if verbose {
		fmt.Println("  creating Firebase account...")
	}
	fbResp, err := firebaseSignUp(email, password)
	if err != nil {
		return "", fmt.Errorf("firebase signup: %w", err)
	}
	if verbose {
		fmt.Printf("  Firebase account created (uid: %s)\n", fbResp.LocalID)
	}

	// Step 2: Launch browser
	l := launcher.New().
		Headless(false).
		Set("disable-blink-features", "AutomationControlled").
		Set("window-size", "1920,1080").
		Delete("enable-automation")
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	page := stealth.MustPage(browser)
	page = page.Timeout(120 * time.Second)
	page.MustEvalOnNewDocument(`() => {
		Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
	}`)

	// Step 3: Navigate to jina.ai/api-dashboard
	if verbose {
		fmt.Println("  navigating to jina.ai/api-dashboard...")
	}
	page.MustNavigate("https://jina.ai/api-dashboard/")
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("page load: %w", err)
	}
	time.Sleep(3 * time.Second)

	// Dismiss cookie banner
	page.Eval(`() => {
		for (const b of document.querySelectorAll('button, a')) {
			const t = (b.textContent || '').toLowerCase().trim();
			if (t.includes('deny') || t.includes('reject') || t.includes('decline') || t.includes('necessary')) {
				b.click(); return;
			}
		}
	}`)
	time.Sleep(1 * time.Second)

	// Step 4: Click the "person" / login icon to open dialog
	if verbose {
		fmt.Println("  clicking login icon...")
	}
	page.Eval(`() => {
		for (const el of document.querySelectorAll('*')) {
			const t = (el.textContent || '').trim();
			if ((t === 'login' || t === 'person') && el.children.length <= 2) {
				el.click(); return;
			}
		}
	}`)
	time.Sleep(2 * time.Second)

	// Step 5: First dialog: click "Log in" (not "Create API Key")
	if verbose {
		fmt.Println("  clicking 'Log in' in dialog...")
	}
	page.Eval(`() => {
		const dialogs = document.querySelectorAll('.q-dialog, [role="dialog"]');
		for (const d of dialogs) {
			for (const el of d.querySelectorAll('button, a, div')) {
				const t = (el.textContent || '').trim();
				if (t === 'Log in' || t === 'loginLog in') {
					el.click(); return 'clicked_login';
				}
			}
		}
	}`)
	time.Sleep(2 * time.Second)

	// Step 6: Check the "agree to Terms" checkbox first
	if verbose {
		fmt.Println("  checking Terms checkbox...")
	}
	page.Eval(`() => {
		const dialogs = document.querySelectorAll('.q-dialog, [role="dialog"]');
		for (const d of dialogs) {
			// Click checkbox or its label
			const checkboxes = d.querySelectorAll('input[type="checkbox"], .q-checkbox, [role="checkbox"]');
			for (const cb of checkboxes) {
				cb.click();
				return 'clicked_checkbox';
			}
			// Also try clicking the label text
			for (const el of d.querySelectorAll('*')) {
				const t = (el.textContent || '').trim();
				if (t.includes('agree') || t.includes('Terms')) {
					if (el.tagName === 'LABEL' || el.closest('label') || el.querySelector('input')) {
						el.click(); return 'clicked_label';
					}
				}
			}
		}
	}`)
	time.Sleep(1 * time.Second)

	// Click "Continue with your Email" — must match Email specifically, not Google/GitHub
	if verbose {
		fmt.Println("  clicking 'Continue with your Email'...")
	}
	page.Eval(`() => {
		const dialogs = document.querySelectorAll('.q-dialog, [role="dialog"]');
		for (const d of dialogs) {
			const btns = d.querySelectorAll('button, a, div[role="button"]');
			for (const el of btns) {
				const t = (el.textContent || '').trim().toLowerCase();
				// Must contain "email" but NOT "google" or "github"
				if (t.includes('email') && !t.includes('google') && !t.includes('github')) {
					el.click(); return 'clicked_email';
				}
			}
		}
	}`)
	time.Sleep(3 * time.Second)

	// Check for new popup/page
	pages, _ := browser.Pages()
	if verbose {
		fmt.Printf("  open pages: %d\n", len(pages))
		for i, p := range pages {
			info, _ := p.Info()
			if info != nil {
				fmt.Printf("    page %d: %s\n", i, info.URL)
			}
		}
	}

	// If a popup opened, switch to it
	if len(pages) > 1 {
		for _, p := range pages {
			info, _ := p.Info()
			if info != nil && !strings.Contains(info.URL, "jina.ai") {
				if verbose {
					fmt.Printf("  switching to popup: %s\n", info.URL)
				}
				page = p
				time.Sleep(2 * time.Second)
				break
			}
		}
	}

	// Step 7: Find and fill the email/password form
	if verbose {
		val, _ := page.Eval(`() => {
			const dialogs = document.querySelectorAll('.q-dialog, [role="dialog"]');
			const result = [];
			dialogs.forEach(d => {
				const inputs = [];
				d.querySelectorAll('input').forEach(inp => {
					inputs.push({ type: inp.type, name: inp.name, placeholder: inp.placeholder, visible: inp.offsetParent !== null });
				});
				const btns = [];
				d.querySelectorAll('button').forEach(b => btns.push(b.textContent.trim()));
				result.push({ text: d.innerText.substring(0, 500).replace(/\n/g, ' | '), inputs: inputs, buttons: btns });
			});
			// Also check for inputs outside dialogs (in case form appeared on page)
			const allInputs = [];
			document.querySelectorAll('input[type="email"], input[type="password"], input[name="email"]').forEach(inp => {
				allInputs.push({ type: inp.type, name: inp.name, placeholder: inp.placeholder, visible: inp.offsetParent !== null, inDialog: !!inp.closest('.q-dialog') });
			});
			return JSON.stringify({ dialogs: result, authInputs: allInputs, url: location.href });
		}`)
		if val != nil {
			fmt.Printf("  form state: %s\n", fmt.Sprint(val.Value))
		}
	}

	// Fill email using rod's native Input() for proper Vue/Quasar reactivity
	if verbose {
		fmt.Println("  filling email + password via keyboard input...")
	}

	// Use rod's element selectors within the dialog
	emailEl, err := page.Element(".q-dialog input[type='text']")
	if err != nil {
		return "", fmt.Errorf("email input not found: %w", err)
	}
	emailEl.MustSelectAllText().MustInput(email)
	time.Sleep(500 * time.Millisecond)

	pwdEl, err := page.Element(".q-dialog input[type='password']")
	if err != nil {
		return "", fmt.Errorf("password input not found: %w", err)
	}
	pwdEl.MustSelectAllText().MustInput(password)
	time.Sleep(500 * time.Millisecond)

	if verbose {
		// Verify inputs were filled
		val, _ := page.Eval(`() => {
			const d = document.querySelector('.q-dialog');
			if (!d) return 'no dialog';
			const t = d.querySelector('input[type="text"]');
			const p = d.querySelector('input[type="password"]');
			return JSON.stringify({email: t ? t.value : 'nil', pwdLen: p ? p.value.length : -1});
		}`)
		if val != nil {
			fmt.Printf("  input values: %s\n", fmt.Sprint(val.Value))
		}
	}

	// Click "Log in" button in the email login dialog
	if verbose {
		fmt.Println("  clicking 'Log in' button...")
	}
	loginBtn, err := page.Element(".q-dialog button")
	if err == nil {
		// Find the right button — skip "chevron_left", find "Log in"
		buttons, _ := page.Elements(".q-dialog button")
		for _, btn := range buttons {
			text, _ := btn.Text()
			text = strings.TrimSpace(text)
			if text == "Log in" {
				// Check this button is in the "Log in with Email" dialog
				inEmailDialog, _ := btn.Eval(`() => {
					const dialog = this.closest('.q-dialog');
					return dialog && dialog.innerText.includes('Log in with Email');
				}`)
				if inEmailDialog != nil && inEmailDialog.Value.Bool() {
					btn.MustClick()
					if verbose {
						fmt.Println("  clicked 'Log in' button")
					}
					break
				}
			}
		}
	} else {
		// Fallback to JS click
		page.Eval(`() => {
			const dialogs = document.querySelectorAll('.q-dialog');
			for (const d of dialogs) {
				if (!d.innerText.includes('Log in with Email')) continue;
				for (const b of d.querySelectorAll('button')) {
					if (b.textContent.trim() === 'Log in') { b.click(); return; }
				}
			}
		}`)
	}
	_ = loginBtn
	time.Sleep(8 * time.Second)

	// Step 8: Check if logged in
	if verbose {
		val, _ := page.Eval(`() => {
			return JSON.stringify({
				url: location.href,
				bodyText: document.body.innerText.substring(0, 500).replace(/\n/g, ' | ')
			});
		}`)
		if val != nil {
			fmt.Printf("  after sign in: %s\n", fmt.Sprint(val.Value))
		}
	}

	// Step 9: Navigate to jina.ai/?newKey as authenticated user to generate key
	if verbose {
		fmt.Println("  navigating to jina.ai/?newKey to generate key...")
	}
	page.MustNavigate("https://jina.ai/?newKey")
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("newKey page load: %w", err)
	}
	time.Sleep(5 * time.Second)

	// Scan for API key on the ?newKey page
	for i := 0; i < 30; i++ {
		val, err := page.Eval(`() => {
			// Check visible text
			const text = document.body.innerText;
			const m = text.match(/jina_[a-f0-9]{32}[a-zA-Z0-9_-]+/);
			if (m) return JSON.stringify({key: m[0], source: 'innerText'});
			// Check input values, code blocks, spans
			for (const el of document.querySelectorAll('input, code, pre, span, div')) {
				const v = el.value || el.textContent || '';
				const m2 = v.match(/jina_[a-f0-9]{32}[a-zA-Z0-9_-]+/);
				if (m2) {
					// Skip keys embedded in JS source code (minified scripts)
					const parent = el.closest('script');
					if (parent) continue;
					// Check if this is visible content
					if (el.offsetParent !== null || el.tagName === 'INPUT') {
						return JSON.stringify({key: m2[0], source: el.tagName + ':' + (el.className || '').substring(0, 50)});
					}
				}
			}
			return '';
		}`)
		if err == nil {
			s := fmt.Sprint(val.Value)
			if strings.Contains(s, "jina_") {
				var found struct {
					Key    string `json:"key"`
					Source string `json:"source"`
				}
				if json.Unmarshal([]byte(s), &found) == nil && jinaKeyRe.MatchString(found.Key) {
					if verbose {
						fmt.Printf("  found key: %s...%s (from %s)\n", found.Key[:12], found.Key[len(found.Key)-4:], found.Source)
					}
					return found.Key, nil
				}
			}
		}

		if verbose && i%5 == 0 {
			// Log page state for debugging
			val2, _ := page.Eval(`() => {
				return JSON.stringify({
					url: location.href,
					text: document.body.innerText.substring(0, 300).replace(/\n/g, ' | '),
					hasKeyReady: document.body.innerText.includes('API key is ready'),
					hasTurnstile: !!document.querySelector('[data-turnstile-callback], iframe[src*="turnstile"]')
				});
			}`)
			if val2 != nil {
				fmt.Printf("  scan %d: %s\n", i, fmt.Sprint(val2.Value))
			}
		}
		time.Sleep(2 * time.Second)
	}

	return "", fmt.Errorf("could not find jina API key after login")
}

type firebaseResponse struct {
	IDToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	LocalID      string `json:"localId"`
	Email        string `json:"email"`
}

func firebaseSignUp(email, password string) (*firebaseResponse, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"email":             email,
		"password":          password,
		"returnSecureToken": true,
	})
	resp, err := http.Post(
		"https://identitytoolkit.googleapis.com/v1/accounts:signUp?key="+firebaseAPIKey,
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var result firebaseResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.IDToken == "" {
		return nil, fmt.Errorf("no idToken in response")
	}
	return &result, nil
}

func (r *registrar) VerifyAndGetKey(email, password, emailBody string, verbose bool) (string, error) {
	return "", fmt.Errorf("jina does not require email verification")
}
