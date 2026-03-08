package jina

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
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

// Register gets a free Jina API key via their website.
// Jina auto-generates a key with 1M free tokens per visitor.
func (r *registrar) Register(email, password string, verbose bool) (string, error) {
	l := launcher.New().Headless(false)
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	page := stealth.MustPage(browser)
	page = page.Timeout(60 * time.Second)

	// Intercept network responses for jina_ keys
	var mu sync.Mutex
	var foundKey string
	keyRe := regexp.MustCompile(`jina_[a-zA-Z0-9_-]{20,}`)

	router := page.HijackRequests()
	router.MustAdd("*api*", func(ctx *rod.Hijack) {
		ctx.MustLoadResponse()
		body := ctx.Response.Body()
		if m := keyRe.FindString(body); m != "" {
			mu.Lock()
			foundKey = m
			mu.Unlock()
		}
	})
	go router.Run()

	if verbose {
		fmt.Println("  navigating to jina.ai (main page with API widget)...")
	}

	// Visit main page — the API key widget is here
	page.MustNavigate("https://jina.ai/?sui=apikey")
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("page load: %w", err)
	}
	time.Sleep(5 * time.Second)

	checkKey := func(label string) string {
		mu.Lock()
		k := foundKey
		mu.Unlock()
		if k != "" {
			if verbose {
				fmt.Printf("  found key via %s: %s...%s\n", label, k[:12], k[len(k)-4:])
			}
			return k
		}
		return ""
	}

	scanPage := func() string {
		// Check innerText
		val, err := page.Eval(`() => {
			const text = document.body.innerText;
			const m = text.match(/jina_[a-zA-Z0-9_-]{20,}/);
			return m ? m[0] : '';
		}`)
		if err == nil {
			s := fmt.Sprint(val.Value)
			if strings.HasPrefix(s, "jina_") {
				return s
			}
		}

		// Check all input values
		val, err = page.Eval(`() => {
			const inputs = document.querySelectorAll('input, textarea, code, pre');
			for (const el of inputs) {
				const v = el.value || el.textContent || '';
				const m = v.match(/jina_[a-zA-Z0-9_-]{20,}/);
				if (m) return m[0];
			}
			return '';
		}`)
		if err == nil {
			s := fmt.Sprint(val.Value)
			if strings.HasPrefix(s, "jina_") {
				return s
			}
		}
		return ""
	}

	for i := 0; i < 30; i++ {
		if k := checkKey("network"); k != "" {
			return k, nil
		}
		if k := scanPage(); k != "" {
			if verbose {
				fmt.Printf("  found key in page: %s...%s\n", k[:12], k[len(k)-4:])
			}
			return k, nil
		}

		// Dump localStorage every 10s for debugging
		if verbose && i%5 == 0 {
			val, _ := page.Eval(`() => {
				const items = {};
				for (let i = 0; i < localStorage.length; i++) {
					const k = localStorage.key(i);
					items[k] = localStorage.getItem(k).substring(0, 100);
				}
				return JSON.stringify(items);
			}`)
			fmt.Printf("  [%ds] localStorage: %v\n", i*2, val.Value)

			// Also print page URL + visible text snippet
			info, _ := page.Info()
			if info != nil {
				fmt.Printf("  [%ds] URL: %s\n", i*2, info.URL)
			}
		}

		// Click elements that might reveal the key
		if i == 2 {
			// Click "API" nav button
			page.Eval(`() => {
				const els = document.querySelectorAll('a, button, div[role="tab"], div[class*="tab"], nav *');
				for (const el of els) {
					const t = (el.textContent || '').trim();
					if (t === 'API' || t === 'API Key' || t === 'Get API Key') {
						el.click();
						return 'clicked: ' + t;
					}
				}
			}`)
		}
		if i == 5 {
			// Click "API KEY & BILLING"
			page.Eval(`() => {
				const els = document.querySelectorAll('*');
				for (const el of els) {
					const t = (el.textContent || '').trim().toLowerCase();
					if (t.includes('api key') && t.includes('billing') && t.length < 30) {
						el.click();
						return 'clicked billing';
					}
				}
			}`)
		}
		if i == 8 {
			// Try the embed widget approach
			page.Eval(`() => {
				const iframes = document.querySelectorAll('iframe');
				for (const f of iframes) {
					if (f.src && f.src.includes('jina')) return 'iframe: ' + f.src;
				}
				// Click any "copy" buttons
				const btns = document.querySelectorAll('button');
				for (const b of btns) {
					const t = (b.textContent || '').toLowerCase();
					if (t.includes('copy') || b.title?.toLowerCase().includes('copy')) {
						b.click();
						return 'clicked copy';
					}
				}
			}`)
		}

		time.Sleep(2 * time.Second)
	}

	return "", fmt.Errorf("could not find jina API key — try visiting https://jina.ai/?sui=apikey manually and copy the key")
}

func (r *registrar) VerifyAndGetKey(email, password, emailBody string, verbose bool) (string, error) {
	return "", fmt.Errorf("jina does not require email verification")
}
