package serp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const recaptchaSiteKey = "6Le-e4MbAAAAABeyLUPW1KL30Hp1RM35bCJ-ShF1"

// RegisterWithRod uses a visible browser to sign up on SerpAPI.
// The form has reCAPTCHA v2 (invisible) that triggers an image challenge.
//
// Env vars:
//   - TWOCAPTCHA_KEY: if set, uses 2captcha API to auto-solve reCAPTCHA
//   - SERP_PROXY: if set, routes browser through this proxy (e.g. socks5://host:port)
//
// Without TWOCAPTCHA_KEY, opens a visible browser for manual captcha solving.
func RegisterWithRod(email, password string) error {
	twoCaptchaKey := os.Getenv("TWOCAPTCHA_KEY")
	proxyURL := os.Getenv("SERP_PROXY")

	l := launcher.New().Headless(false)
	if proxyURL != "" {
		l = l.Proxy(proxyURL)
		fmt.Printf("  using proxy: %s\n", proxyURL)
	}
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://serpapi.com/users/sign_up")
	page = page.Timeout(120 * time.Second)

	if err := page.WaitLoad(); err != nil {
		return fmt.Errorf("page load: %w", err)
	}
	time.Sleep(2 * time.Second)

	// Bring window to front
	_ = proto.PageBringToFront{}.Call(page)

	// Fill form fields with human-like delays
	nameFromEmail := strings.Split(email, "@")[0]
	page.MustElement(`input[name="user[full_name]"]`).MustInput(nameFromEmail)
	time.Sleep(300 * time.Millisecond)
	page.MustElement(`input[name="user[email]"]`).MustInput(email)
	time.Sleep(300 * time.Millisecond)
	page.MustElement(`input[name="user[password]"]`).MustInput(password)
	time.Sleep(300 * time.Millisecond)
	page.MustElement(`input[name="user[password_confirmation]"]`).MustInput(password)
	time.Sleep(500 * time.Millisecond)

	if twoCaptchaKey != "" {
		// Auto-solve reCAPTCHA via 2captcha
		fmt.Println("  solving reCAPTCHA via 2captcha...")
		token, err := solveRecaptchaV2(twoCaptchaKey, "https://serpapi.com/users/sign_up", recaptchaSiteKey)
		if err != nil {
			fmt.Printf("  2captcha failed: %v — falling back to manual\n", err)
		} else {
			// Inject via JS callback (the way the form's JS works)
			page.MustEval(`(token) => {
				if (window.ReCaptchaFormSuccess) {
					window.ReCaptchaFormSuccess(token);
				}
			}`, token)
			time.Sleep(2 * time.Second)
		}
	}

	// Click submit — this triggers invisible reCAPTCHA
	page.MustElement(`input[type="submit"]`).MustClick()

	if twoCaptchaKey == "" {
		fmt.Println("  solve the reCAPTCHA in the browser window...")
	}

	// Wait for page to navigate away from sign_up
	for i := 0; i < 24; i++ { // up to 120s
		time.Sleep(5 * time.Second)

		info, err := page.Info()
		if err != nil {
			// Page/target gone — likely navigated to a new page (success)
			fmt.Println("  page navigated away (target closed) — assuming success")
			return nil
		}
		currentURL := info.URL

		html, err := page.HTML()
		if err != nil {
			fmt.Println("  page navigated away — assuming success")
			return nil
		}

		// Success: confirmation message
		if strings.Contains(html, "A message with a confirmation link has been sent") ||
			(strings.Contains(html, "confirmation link") && !strings.Contains(html, "sign_up")) {
			fmt.Println("  registration successful — confirmation email sent!")
			return nil
		}
		// Success: redirected to dashboard
		if strings.Contains(currentURL, "dashboard") {
			fmt.Println("  registration successful — redirected to dashboard!")
			return nil
		}
		// Success: redirected away from sign_up
		if !strings.Contains(currentURL, "sign_up") && !strings.Contains(currentURL, "users") {
			fmt.Printf("  navigated to: %s\n", currentURL)
			return nil
		}

		// Error: IP blocked
		if strings.Contains(html, "couldn't create your account") ||
			strings.Contains(html, "activity on your network") {
			return fmt.Errorf("blocked by SerpAPI (IP/network detection). Use --proxy socks5://host:port")
		}
		// Error: reCAPTCHA token rejected
		if strings.Contains(html, "could not verify your browser") {
			return fmt.Errorf("reCAPTCHA verification failed — token was rejected")
		}
		// Error: other
		if errEl, err := page.Element(".alert-danger"); err == nil {
			if text, err := errEl.Text(); err == nil && strings.TrimSpace(text) != "" {
				return fmt.Errorf("signup error: %s", strings.TrimSpace(text))
			}
		}
	}

	return fmt.Errorf("registration timed out (120s)")
}

// solveRecaptchaV2 uses the 2captcha service to solve a reCAPTCHA v2 challenge.
func solveRecaptchaV2(apiKey, pageURL, siteKey string) (string, error) {
	submitURL := fmt.Sprintf(
		"https://2captcha.com/in.php?key=%s&method=userrecaptcha&googlekey=%s&pageurl=%s&json=1&invisible=1",
		url.QueryEscape(apiKey), url.QueryEscape(siteKey), url.QueryEscape(pageURL))

	resp, err := http.Get(submitURL)
	if err != nil {
		return "", fmt.Errorf("submit: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var submitResp struct {
		Status  int    `json:"status"`
		Request string `json:"request"`
	}
	if err := json.Unmarshal(body, &submitResp); err != nil {
		return "", fmt.Errorf("parse: %w (body: %s)", err, string(body))
	}
	if submitResp.Status != 1 {
		return "", fmt.Errorf("2captcha: %s", submitResp.Request)
	}
	taskID := submitResp.Request
	fmt.Printf("  2captcha task: %s\n", taskID)

	resultURL := fmt.Sprintf(
		"https://2captcha.com/res.php?key=%s&action=get&id=%s&json=1",
		url.QueryEscape(apiKey), url.QueryEscape(taskID))

	for i := 0; i < 30; i++ {
		time.Sleep(5 * time.Second)
		resp, err := http.Get(resultURL)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var r struct {
			Status  int    `json:"status"`
			Request string `json:"request"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			continue
		}
		if r.Status == 1 {
			fmt.Println("  2captcha solved!")
			return r.Request, nil
		}
		if r.Request != "CAPCHA_NOT_READY" {
			return "", fmt.Errorf("2captcha: %s", r.Request)
		}
	}
	return "", fmt.Errorf("2captcha timed out (150s)")
}
