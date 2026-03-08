package jina

import (
	"fmt"
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

// API key: jina_ + hex(32) + mixed alphanumeric with _ and -
var jinaKeyRe = regexp.MustCompile(`jina_[a-f0-9]{32}[a-zA-Z0-9_-]+`)

// Register gets a free Jina API key with 1M tokens.
// Flow:
// 1. Open jina.ai dashboard in rod browser
// 2. Dismiss cookies, wait for Turnstile to auto-solve
// 3. Extract Turnstile token, POST to keygen.jina.ai/trial (1M tokens)
// 4. If that fails, click "Create API Key" and intercept/capture
func (r *registrar) Register(email, password string, verbose bool) (string, error) {
	l := launcher.New().Headless(false)
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	page := stealth.MustPage(browser)
	page = page.Timeout(90 * time.Second)

	if verbose {
		fmt.Println("  navigating to jina.ai API dashboard...")
	}

	page.MustNavigate("https://jina.ai/api-dashboard/")
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("page load: %w", err)
	}
	time.Sleep(5 * time.Second)

	// Clear any cached keys from previous runs
	page.Eval(`() => {
		localStorage.clear();
		sessionStorage.clear();
	}`)

	// Dismiss cookie banner
	page.Eval(`() => {
		for (const b of document.querySelectorAll('button, a')) {
			const t = (b.textContent || '').toLowerCase().trim();
			if (t.includes('deny') || t.includes('reject') || t.includes('decline') || t.includes('necessary')) {
				b.click(); return;
			}
		}
	}`)
	time.Sleep(2 * time.Second)

	// Install comprehensive interceptor: capture ALL keygen requests and redirect /empty→/trial
	if verbose {
		fmt.Println("  installing fetch+XHR interceptor...")
	}
	page.MustEval(`() => {
		window.__jinaApiKey = "";
		window.__jinaCalls = [];

		// Intercept fetch
		const origFetch = window.fetch;
		window.fetch = async function(input, opts) {
			const url = typeof input === 'string' ? input : (input.url || '');
			window.__jinaCalls.push('fetch:' + url);
			let targetUrl = input;
			if (typeof input === 'string' && input.includes('keygen.jina.ai')) {
				if (input.includes('/empty')) {
					targetUrl = input.replace('/empty', '/trial');
					window.__jinaCalls.push('REDIRECTED:' + targetUrl);
				}
			}
			const resp = await origFetch.call(this, targetUrl, opts);
			if (typeof url === 'string' && url.includes('keygen.jina.ai')) {
				try {
					const clone = resp.clone();
					const data = await clone.json();
					if (data.api_key) {
						window.__jinaApiKey = data.api_key;
						window.__jinaCalls.push('CAPTURED:' + data.api_key.substring(0, 12));
					} else {
						window.__jinaCalls.push('RESP_NO_KEY:' + JSON.stringify(data).substring(0, 200));
					}
				} catch(e) {
					window.__jinaCalls.push('RESP_ERR:' + e.message);
				}
			}
			return resp;
		};

		// Intercept XMLHttpRequest
		const origOpen = XMLHttpRequest.prototype.open;
		const origSend = XMLHttpRequest.prototype.send;
		XMLHttpRequest.prototype.open = function(method, url, ...rest) {
			this.__url = url;
			if (typeof url === 'string' && url.includes('keygen.jina.ai/empty')) {
				url = url.replace('/empty', '/trial');
				window.__jinaCalls.push('XHR_REDIRECTED:' + url);
			}
			return origOpen.call(this, method, url, ...rest);
		};
		XMLHttpRequest.prototype.send = function(body) {
			if (this.__url && this.__url.includes('keygen.jina.ai')) {
				this.addEventListener('load', () => {
					try {
						const data = JSON.parse(this.responseText);
						if (data.api_key) {
							window.__jinaApiKey = data.api_key;
							window.__jinaCalls.push('XHR_CAPTURED:' + data.api_key.substring(0, 12));
						}
					} catch(e) {}
				});
			}
			return origSend.call(this, body);
		};
	}`)

	// Click "Create API Key"
	if verbose {
		fmt.Println("  clicking 'Create API Key'...")
	}
	page.Eval(`() => {
		for (const el of document.querySelectorAll('button, a, div[role="button"], span')) {
			if ((el.textContent || '').trim().includes('Create API Key')) {
				el.click(); return 'clicked';
			}
		}
	}`)

	// Wait for key to appear via interceptor
	for i := 0; i < 30; i++ {
		// Check intercepted key
		val, err := page.Eval(`() => window.__jinaApiKey || ""`)
		if err == nil {
			s := fmt.Sprint(val.Value)
			if strings.HasPrefix(s, "jina_") && jinaKeyRe.MatchString(s) {
				if verbose {
					fmt.Printf("  got key via interceptor: %s...%s\n", s[:12], s[len(s)-4:])
				}
				return s, nil
			}
		}

		// Log intercepted calls for debugging
		if verbose && (i == 5 || i == 10 || i == 15) {
			callsVal, _ := page.Eval(`() => JSON.stringify(window.__jinaCalls || [])`)
			if callsVal != nil {
				fmt.Printf("  intercepted calls: %s\n", fmt.Sprint(callsVal.Value))
			}
		}

		if i == 8 || i == 16 {
			// Retry: reload page, re-install interceptor, click again
			if verbose {
				fmt.Printf("  retrying (attempt %d)...\n", i/8+1)
			}
			page.MustNavigate("https://jina.ai/api-dashboard/")
			time.Sleep(4 * time.Second)
			page.Eval(`() => { localStorage.clear(); sessionStorage.clear(); }`)
			page.MustEval(`() => {
				window.__jinaApiKey = "";
				window.__jinaCalls = [];
				const origFetch = window.fetch;
				window.fetch = async function(input, opts) {
					const url = typeof input === 'string' ? input : (input.url || '');
					window.__jinaCalls.push('fetch:' + url);
					let targetUrl = input;
					if (typeof input === 'string' && input.includes('keygen.jina.ai')) {
						if (input.includes('/empty')) {
							targetUrl = input.replace('/empty', '/trial');
						}
					}
					const resp = await origFetch.call(this, targetUrl, opts);
					if (typeof url === 'string' && url.includes('keygen.jina.ai')) {
						try {
							const clone = resp.clone();
							const data = await clone.json();
							if (data.api_key) {
								window.__jinaApiKey = data.api_key;
							}
						} catch(e) {}
					}
					return resp;
				};
				const origOpen = XMLHttpRequest.prototype.open;
				XMLHttpRequest.prototype.open = function(method, url, ...rest) {
					if (typeof url === 'string' && url.includes('keygen.jina.ai/empty')) {
						url = url.replace('/empty', '/trial');
					}
					return origOpen.call(this, method, url, ...rest);
				};
			}`)
			time.Sleep(2 * time.Second)
			// Click create button
			page.Eval(`() => {
				for (const el of document.querySelectorAll('button, a, div[role="button"], span')) {
					if ((el.textContent || '').trim().includes('Create API Key')) {
						el.click(); return;
					}
				}
			}`)
		}

		if verbose && i%5 == 0 {
			fmt.Printf("  waiting for key... (%ds)\n", i*2)
		}
		time.Sleep(2 * time.Second)
	}

	// Approach 2: Try extracting Turnstile token and calling /trial directly
	if verbose {
		fmt.Println("  trying direct /trial call with Turnstile token...")
	}
	directKey, err := tryDirectTrialCall(page, verbose)
	if err == nil && directKey != "" {
		return directKey, nil
	}
	if verbose && err != nil {
		fmt.Printf("  direct /trial failed: %v\n", err)
	}

	// Last resort: scan page for any key (may have 0 balance)
	if k := findKey(page); k != "" {
		if verbose {
			fmt.Printf("  found key in page (may have 0 balance): %s...%s\n", k[:12], k[len(k)-4:])
		}
		return k, nil
	}

	return "", fmt.Errorf("could not get jina API key")
}

// tryDirectTrialCall extracts the Turnstile token from the page and POSTs to /trial directly.
func tryDirectTrialCall(page *rod.Page, verbose bool) (string, error) {
	val, err := page.Eval(`() => {
		// Try to find Turnstile response token
		// Method 1: turnstile.getResponse()
		if (typeof turnstile !== 'undefined') {
			const widgets = document.querySelectorAll('[data-sitekey]');
			for (const w of widgets) {
				const id = w.id || w.getAttribute('data-turnstile-id');
				if (id) {
					try { const t = turnstile.getResponse(id); if (t) return t; } catch(e) {}
				}
			}
			try { const t = turnstile.getResponse(); if (t) return t; } catch(e) {}
		}
		// Method 2: hidden input
		for (const inp of document.querySelectorAll('input[type="hidden"]')) {
			const v = inp.value;
			if (v && v.length > 100 && v.startsWith('0.')) return v;
		}
		// Method 3: cf-turnstile response
		const cfEl = document.querySelector('.cf-turnstile [name="cf-turnstile-response"]');
		if (cfEl && cfEl.value) return cfEl.value;
		// Method 4: iframe postMessage result
		for (const inp of document.querySelectorAll('[name*="turnstile"], [name*="cf-"]')) {
			if (inp.value && inp.value.length > 50) return inp.value;
		}
		return '';
	}`)
	if err != nil {
		return "", fmt.Errorf("eval turnstile: %w", err)
	}
	token := fmt.Sprint(val.Value)
	if token == "" || len(token) < 50 {
		return "", fmt.Errorf("no turnstile token found")
	}
	if verbose {
		fmt.Printf("  got Turnstile token (%d chars)\n", len(token))
	}

	// POST to /trial with the token
	result, err := page.Eval(`(token) => {
		return fetch('https://keygen.jina.ai/trial', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ turnstile_token: token })
		}).then(r => r.json()).then(d => JSON.stringify(d));
	}`, token)
	if err != nil {
		return "", fmt.Errorf("trial POST: %w", err)
	}
	resp := fmt.Sprint(result.Value)
	if verbose {
		fmt.Printf("  /trial response: %s\n", resp)
	}

	// Extract api_key from response
	if m := jinaKeyRe.FindString(resp); m != "" {
		return m, nil
	}
	return "", fmt.Errorf("/trial response has no key: %s", resp)
}

func (r *registrar) VerifyAndGetKey(email, password, emailBody string, verbose bool) (string, error) {
	return "", fmt.Errorf("jina does not require email verification")
}

func findKey(page *rod.Page) string {
	val, err := page.Eval(`() => {
		const re = /jina_[a-f0-9]{32}[a-zA-Z0-9_-]+/;
		const m = document.body.innerText.match(re);
		if (m) return m[0];
		for (const el of document.querySelectorAll('input, textarea, code, pre, span')) {
			const v = el.value || el.textContent || '';
			const m2 = v.match(re);
			if (m2) return m2[0];
		}
		return '';
	}`)
	if err == nil {
		s := fmt.Sprint(val.Value)
		if strings.HasPrefix(s, "jina_") {
			return s
		}
	}

	html, _ := page.HTML()
	if m := jinaKeyRe.FindString(html); m != "" {
		return m
	}

	return ""
}
