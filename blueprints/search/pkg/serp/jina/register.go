package jina

import (
	"fmt"
	"net"
	"os"
	"os/exec"
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

// Register navigates to jina.ai/?newKey in a visible browser.
// The page auto-generates a key via Cloudflare Turnstile.
// In headless mode, Turnstile may not solve automatically —
// the browser is shown so the user can click the checkbox if needed.
func (r *registrar) Register(email, password string, verbose bool) (string, error) {
	// Try launching Chrome manually (bypasses rod launcher detection)
	browser, cleanup, err := launchCleanChrome(verbose)
	if err != nil {
		return "", fmt.Errorf("browser launch: %w", err)
	}
	defer cleanup()

	page := stealth.MustPage(browser)
	page = page.Timeout(300 * time.Second)
	page.MustEvalOnNewDocument(`() => {
		Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
	}`)

	// Navigate to jina.ai/api-dashboard and click "Create API Key"
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

	// Click person/login icon to open dialog
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

	// Click "Create API Key" in the dialog
	if verbose {
		fmt.Println("  clicking 'Create API Key'...")
	}
	page.Eval(`() => {
		const dialogs = document.querySelectorAll('.q-dialog, [role="dialog"]');
		for (const d of dialogs) {
			for (const el of d.querySelectorAll('button, a, div')) {
				const t = (el.textContent || '').trim();
				if (t.includes('Create API Key') || t.includes('Create') || t.includes('Get API Key')) {
					el.click(); return 'clicked_create';
				}
			}
		}
	}`)
	time.Sleep(3 * time.Second)

	if verbose {
		val, _ := page.Eval(`() => JSON.stringify({
			url: location.href,
			text: document.body.innerText.substring(0, 500).replace(/\n/g, ' | '),
			hasTurnstile: !!document.querySelector('iframe[src*="turnstile"], [data-turnstile-callback]'),
			hasKey: /jina_[a-f0-9]{32}/.test(document.body.innerText)
		})`)
		if val != nil {
			fmt.Printf("  after create click: %s\n", fmt.Sprint(val.Value))
		}
	}

	// Also try navigating to ?newKey if we're still on the dashboard
	info, _ := page.Info()
	if info != nil && strings.Contains(info.URL, "api-dashboard") {
		if verbose {
			fmt.Println("  navigating to jina.ai/?newKey...")
		}
		page.MustNavigate("https://jina.ai/?newKey")
		if err := page.WaitLoad(); err != nil {
			return "", fmt.Errorf("newKey page load: %w", err)
		}
		time.Sleep(5 * time.Second)
	}

	// Scan for API key (wait up to ~2 minutes for Turnstile to solve)
	for i := 0; i < 60; i++ {
		key := extractJinaKey(page)
		if key != "" {
			if verbose {
				fmt.Printf("  found key: %s...%s\n", key[:12], key[len(key)-4:])
			}
			return key, nil
		}

		if verbose && i%10 == 0 {
			val, _ := page.Eval(`() => JSON.stringify({
				url: location.href,
				hasKeyReady: document.body.innerText.includes('key is ready') || document.body.innerText.includes('Free API Key'),
				hasTurnstile: !!document.querySelector('iframe[src*="turnstile"]'),
				hasError: document.body.innerText.includes('cannot generate') || document.body.innerText.includes('verify'),
				text: document.body.innerText.substring(0, 400).replace(/\n/g, ' | ')
			})`)
			if val != nil {
				fmt.Printf("  scan %d: %s\n", i, fmt.Sprint(val.Value))
			}
		}
		time.Sleep(2 * time.Second)
	}

	return "", fmt.Errorf("could not find jina API key (Turnstile may not have solved)")
}

// extractJinaKey looks for a jina API key in visible page content.
func extractJinaKey(page *rod.Page) string {
	val, err := page.Eval(`() => {
		// Check visible text only (not script source)
		const text = document.body.innerText;
		const m = text.match(/jina_[a-f0-9]{32}[a-zA-Z0-9_-]+/);
		if (m) return m[0];
		// Check input values
		for (const el of document.querySelectorAll('input')) {
			const v = el.value || '';
			const m2 = v.match(/jina_[a-f0-9]{32}[a-zA-Z0-9_-]+/);
			if (m2) return m2[0];
		}
		// Check code blocks and pre elements
		for (const el of document.querySelectorAll('code, pre')) {
			const v = el.textContent || '';
			const m2 = v.match(/jina_[a-f0-9]{32}[a-zA-Z0-9_-]+/);
			if (m2 && el.offsetParent !== null) return m2[0];
		}
		return '';
	}`)
	if err != nil {
		return ""
	}
	s := fmt.Sprint(val.Value)
	if strings.HasPrefix(s, "jina_") && jinaKeyRe.MatchString(s) {
		return s
	}
	return ""
}

// launchCleanChrome launches Chrome directly via os/exec with a fresh profile,
// avoiding rod's launcher which adds detectable automation flags.
func launchCleanChrome(verbose bool) (*rod.Browser, func(), error) {
	// Find Chrome binary
	chromePath := launcher.NewBrowser().MustGet()

	// Find a free port for debugging
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Create temp profile dir
	tmpDir, err := os.MkdirTemp("", "jina-chrome-*")
	if err != nil {
		return nil, nil, err
	}

	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", port),
		fmt.Sprintf("--user-data-dir=%s", tmpDir),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-default-apps",
		"--disable-extensions",
		"--disable-sync",
		"--disable-translate",
		"--metrics-recording-only",
		"--disable-blink-features=AutomationControlled",
		"--window-size=1920,1080",
		"about:blank",
	}

	cmd := exec.Command(chromePath, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, nil, fmt.Errorf("chrome start: %w", err)
	}

	if verbose {
		fmt.Printf("  Chrome launched (pid=%d, port=%d)\n", cmd.Process.Pid, port)
	}

	// Wait for Chrome to be ready
	wsURL := ""
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)
		u, err := launcher.ResolveURL(fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			wsURL = u
			break
		}
	}
	if wsURL == "" {
		cmd.Process.Kill()
		os.RemoveAll(tmpDir)
		return nil, nil, fmt.Errorf("chrome did not start (port %d)", port)
	}

	browser := rod.New().ControlURL(wsURL).MustConnect()
	cleanup := func() {
		browser.MustClose()
		cmd.Process.Kill()
		cmd.Wait()
		os.RemoveAll(tmpDir)
	}

	return browser, cleanup, nil
}

func (r *registrar) VerifyAndGetKey(email, password, emailBody string, verbose bool) (string, error) {
	return "", fmt.Errorf("jina does not require email verification")
}

// keyInfo checks the balance of a Jina API key.
func keyInfo(apiKey string) (map[string]interface{}, error) {
	// TODO: implement via dash.jina.ai/api/v1/api_key/fe_user?api_key=<key>
	_ = apiKey
	return nil, fmt.Errorf("not implemented")
}
