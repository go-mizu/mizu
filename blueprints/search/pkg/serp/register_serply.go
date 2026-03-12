package serp

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func init() {
	RegisterRegistrar("serply", &serplyRegistrar{})
}

type serplyRegistrar struct{}

// Register on serply.io via HTTP — NO captcha, standard Rails form.
func (r *serplyRegistrar) Register(email, password string, verbose bool) (string, error) {
	jar, _ := cookiejar.New(nil)
	hc := &http.Client{Timeout: 30 * time.Second, Jar: jar}

	// Step 1: GET signup page for CSRF token + spinner
	if verbose {
		fmt.Println("  fetching signup page...")
	}
	resp, err := hc.Get("https://app.serply.io/users/sign_up")
	if err != nil {
		return "", fmt.Errorf("GET signup: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	pageHTML := string(body)

	csrfRe := regexp.MustCompile(`name="authenticity_token"\s+value="([^"]+)"`)
	csrfMatch := csrfRe.FindStringSubmatch(pageHTML)
	if len(csrfMatch) < 2 {
		return "", fmt.Errorf("no CSRF token found")
	}
	csrfToken := csrfMatch[1]

	// Extract spinner (anti-bot hash)
	spinnerRe := regexp.MustCompile(`name="spinner"\s+value="([^"]+)"`)
	spinnerMatch := spinnerRe.FindStringSubmatch(pageHTML)
	spinner := ""
	if len(spinnerMatch) >= 2 {
		spinner = spinnerMatch[1]
	}

	// Extract honeypot field name (hidden input with tabindex="-1" and "If you are a human" label)
	honeypotRe := regexp.MustCompile(`If you are a human, ignore this field</label><input[^>]+name="([^"]+)"`)
	honeypotMatch := honeypotRe.FindStringSubmatch(pageHTML)
	honeypotName := ""
	if len(honeypotMatch) >= 2 {
		honeypotName = honeypotMatch[1]
	}

	if verbose {
		fmt.Printf("  CSRF: %s...  honeypot: %s\n", csrfToken[:16], honeypotName)
	}

	// Step 2: Wait for invisible_captcha timestamp threshold (default 4s, use 5s to be safe)
	if verbose {
		fmt.Println("  waiting 5s (invisible_captcha timestamp threshold)...")
	}
	time.Sleep(5 * time.Second)

	// Step 3: POST signup
	formData := url.Values{
		"authenticity_token":     {csrfToken},
		"user[name]":             {email[:8] + " User"},
		"user[email]":            {email},
		"user[password]":         {password},
		"user[terms_of_service]": {"1"},
		"commit":                 {"Sign up"},
		"spinner":                {spinner},
	}
	if honeypotName != "" {
		formData.Set(honeypotName, "") // must be empty
	}

	req, _ := http.NewRequest("POST", "https://app.serply.io/users", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://app.serply.io/users/sign_up")
	req.Header.Set("Origin", "https://app.serply.io")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err = hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("POST signup: %w", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	respHTML := string(respBody)

	if verbose {
		fmt.Printf("  POST status: %d\n", resp.StatusCode)
	}

	if strings.Contains(respHTML, "has already been taken") {
		return "", fmt.Errorf("email already registered")
	}

	// Success — email confirmation required
	if strings.Contains(respHTML, "confirmation") || strings.Contains(respHTML, "verify") ||
		resp.StatusCode == 302 || resp.StatusCode == 303 || resp.StatusCode == 200 {
		if verbose {
			fmt.Println("  signup successful — email confirmation needed")
		}
		return "", nil
	}

	return "", nil
}

func (r *serplyRegistrar) VerifyAndGetKey(email, password, emailBody string, verbose bool) (string, error) {
	// Find verification link
	linkRe := regexp.MustCompile(`https://app\.serply\.io/users/confirmation[^\s"<']*`)
	link := linkRe.FindString(emailBody)
	if link == "" {
		allRe := regexp.MustCompile(`https://[^\s"<']+`)
		for _, m := range allRe.FindAllString(emailBody, -1) {
			if strings.Contains(m, "serply") && strings.Contains(m, "confirm") {
				link = m
				break
			}
		}
	}

	if link != "" {
		if verbose {
			fmt.Printf("  verification link: %s\n", link)
		}
		hc := &http.Client{Timeout: 20 * time.Second}
		resp, err := hc.Get(link)
		if err != nil {
			return "", fmt.Errorf("verify: %w", err)
		}
		resp.Body.Close()
		if verbose {
			fmt.Printf("  verification status: %d\n", resp.StatusCode)
		}
	} else {
		return "", fmt.Errorf("no verification link found in email")
	}

	// Login and get API key
	return r.loginAndGetKey(email, password, verbose)
}

func (r *serplyRegistrar) loginAndGetKey(email, password string, verbose bool) (string, error) {
	jar, _ := cookiejar.New(nil)
	hc := &http.Client{Timeout: 30 * time.Second, Jar: jar}

	// GET login page for CSRF
	resp, err := hc.Get("https://app.serply.io/users/sign_in")
	if err != nil {
		return "", err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	csrfRe := regexp.MustCompile(`name="authenticity_token"\s+value="([^"]+)"`)
	csrfMatch := csrfRe.FindStringSubmatch(string(body))
	if len(csrfMatch) < 2 {
		return "", fmt.Errorf("no CSRF token on login page")
	}

	// POST login
	formData := url.Values{
		"authenticity_token": {csrfMatch[1]},
		"user[email]":        {email},
		"user[password]":     {password},
		"commit":             {"Log in"},
	}
	req, _ := http.NewRequest("POST", "https://app.serply.io/users/sign_in", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://app.serply.io/users/sign_in")

	resp, err = hc.Do(req)
	if err != nil {
		return "", err
	}
	io.ReadAll(resp.Body)
	resp.Body.Close()

	if verbose {
		fmt.Printf("  login status: %d\n", resp.StatusCode)
	}

	// Navigate to dashboard pages looking for API key
	for _, path := range []string{"/", "/dashboard", "/api-keys", "/settings", "/api_keys"} {
		resp, err := hc.Get("https://app.serply.io" + path)
		if err != nil {
			continue
		}
		pageBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		html := string(pageBody)

		if verbose {
			fmt.Printf("  page %s: status=%d len=%d\n", path, resp.StatusCode, len(html))
		}

		// Detect Stripe checkout onboarding wall
		if strings.Contains(html, "checkout.stripe.com") || strings.Contains(html, "Continue to Checkout") {
			return "", fmt.Errorf("serply requires credit card verification (Stripe checkout) before API access — signup succeeded, use 'serp add-key serply <key>' after completing onboarding at https://app.serply.io")
		}

		// Look for API key patterns in the page
		keyPatterns := []*regexp.Regexp{
			regexp.MustCompile(`[Aa][Pp][Ii].?[Kk]ey[^a-zA-Z0-9]*([a-zA-Z0-9]{20,})`),
			regexp.MustCompile(`data-api-key="([^"]+)"`),
			regexp.MustCompile(`value="([a-zA-Z0-9]{20,64})"`),
			regexp.MustCompile(`"api_key"\s*:\s*"([^"]+)"`),
			regexp.MustCompile(`"apiKey"\s*:\s*"([^"]+)"`),
			regexp.MustCompile(`API Key[^<]*<[^>]*>([a-zA-Z0-9]{16,})`),
		}
		for _, re := range keyPatterns {
			if m := re.FindStringSubmatch(html); len(m) >= 2 {
				return m[1], nil
			}
		}
	}

	return "", fmt.Errorf("could not find API key after login")
}
