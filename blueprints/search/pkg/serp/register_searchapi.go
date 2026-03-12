package serp

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
)

func init() {
	RegisterRegistrar("searchapi", &searchapiRegistrar{})
}

type searchapiRegistrar struct{}

// Register on searchapi.io via rod. Has reCAPTCHA v3 (invisible).
func (r *searchapiRegistrar) Register(email, password string, verbose bool) (string, error) {
	l := launcher.New().Headless(false)
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	page := stealth.MustPage(browser)
	page = page.Timeout(90 * time.Second)
	page.MustNavigate("https://www.searchapi.io/users/sign_up")
	page = page.Timeout(60 * time.Second)

	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("page load: %w", err)
	}
	time.Sleep(2 * time.Second)

	// Fill full name
	nameEl, err := page.Element(`input[name="user[full_name]"], input[placeholder*="name" i]`)
	if err == nil {
		nameEl.MustInput(email[:8])
		time.Sleep(300 * time.Millisecond)
	}

	// Fill email
	emailEl, err := page.Element(`input[name="user[email]"], input[type="email"]`)
	if err != nil {
		return "", fmt.Errorf("email field not found: %w", err)
	}
	emailEl.MustInput(email)
	time.Sleep(300 * time.Millisecond)

	// Fill password
	passEl, err := page.Element(`input[name="user[password]"], input[type="password"]`)
	if err != nil {
		return "", fmt.Errorf("password field not found: %w", err)
	}
	passEl.MustInput(password)
	time.Sleep(500 * time.Millisecond)

	// Check terms checkbox if present
	if checkbox, err := page.Element(`input[type="checkbox"]`); err == nil {
		checkbox.MustClick()
		time.Sleep(300 * time.Millisecond)
	}

	// Click submit
	submitEl, err := page.Element(`button[type="submit"], input[type="submit"]`)
	if err != nil {
		return "", fmt.Errorf("submit not found: %w", err)
	}
	submitEl.MustClick()

	if verbose {
		fmt.Println("  form submitted, waiting for redirect...")
	}

	// Wait for result
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)
		info, err := page.Info()
		if err != nil {
			break
		}
		url := info.URL
		if verbose && i%5 == 0 {
			fmt.Printf("  current URL: %s\n", url)
		}

		// Dashboard means we're in
		if strings.Contains(url, "dashboard") || strings.Contains(url, "api_key") {
			time.Sleep(2 * time.Second)
			html, _ := page.HTML()
			if key := extractSearchAPIKey(html); key != "" {
				return key, nil
			}
			return "", nil // Need email verification
		}

		// Check for verification needed
		html, _ := page.HTML()
		if strings.Contains(html, "confirm") || strings.Contains(html, "verify") || strings.Contains(html, "check your email") {
			if verbose {
				fmt.Println("  email verification required")
			}
			return "", nil
		}

		// Errors
		if strings.Contains(html, "has already been taken") {
			return "", fmt.Errorf("email already registered")
		}
	}

	return "", fmt.Errorf("registration timed out")
}

var searchapiKeyRe = regexp.MustCompile(`[a-zA-Z0-9]{20,}`)

func extractSearchAPIKey(html string) string {
	matches := searchapiKeyRe.FindAllString(html, -1)
	for _, m := range matches {
		if len(m) >= 20 && len(m) <= 80 {
			return m
		}
	}
	return ""
}

func (r *searchapiRegistrar) VerifyAndGetKey(email, password, emailBody string, verbose bool) (string, error) {
	// Find verification link
	linkRe := regexp.MustCompile(`https://www\.searchapi\.io[^\s"<']+confirm[^\s"<']*`)
	link := linkRe.FindString(emailBody)
	if link == "" {
		linkRe = regexp.MustCompile(`https://www\.searchapi\.io/[^\s"<']+`)
		link = linkRe.FindString(emailBody)
	}

	if link != "" {
		if verbose {
			fmt.Printf("  verification link: %s\n", link)
		}
	}

	// Use rod to verify + get key from dashboard
	l := launcher.New().Headless(false)
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	if link != "" {
		page := stealth.MustPage(browser)
		page = page.Timeout(90 * time.Second)
		page.MustNavigate(link)
		page.Timeout(30 * time.Second)
		time.Sleep(5 * time.Second)
	}

	// Login
	page := stealth.MustPage(browser)
	page = page.Timeout(90 * time.Second)
	page.MustNavigate("https://www.searchapi.io/users/sign_in")
	page = page.Timeout(30 * time.Second)
	if err := page.WaitLoad(); err != nil {
		return "", err
	}
	time.Sleep(2 * time.Second)

	if emailEl, err := page.Element(`input[type="email"], input[name*="email"]`); err == nil {
		emailEl.MustInput(email)
		time.Sleep(300 * time.Millisecond)
	}
	if passEl, err := page.Element(`input[type="password"]`); err == nil {
		passEl.MustInput(password)
		time.Sleep(300 * time.Millisecond)
	}
	if btn, err := page.Element(`button[type="submit"], input[type="submit"]`); err == nil {
		btn.MustClick()
	}
	time.Sleep(5 * time.Second)

	// Go to dashboard/API key page
	page.MustNavigate("https://www.searchapi.io/dashboard")
	time.Sleep(3 * time.Second)

	html, _ := page.HTML()
	if key := extractSearchAPIKey(html); key != "" {
		return key, nil
	}

	return "", fmt.Errorf("could not find API key")
}
