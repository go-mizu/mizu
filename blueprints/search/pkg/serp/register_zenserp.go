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
	RegisterRegistrar("zenserp", &zenserpRegistrar{})
}

type zenserpRegistrar struct{}

// Register on zenserp.com via rod. Has reCAPTCHA v3 (invisible).
func (r *zenserpRegistrar) Register(email, password string, verbose bool) (string, error) {
	l := launcher.New().Headless(false)
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	page := stealth.MustPage(browser)
	page = page.Timeout(90 * time.Second)
	page.MustNavigate("https://app.zenserp.com/register")
	page = page.Timeout(60 * time.Second)

	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("page load: %w", err)
	}
	time.Sleep(3 * time.Second)

	// Fill name
	if el, err := page.Element(`input[name="name"], input[placeholder*="name" i]`); err == nil {
		el.MustInput(email[:8])
		time.Sleep(300 * time.Millisecond)
	}

	// Fill email
	if el, err := page.Element(`input[name="email"], input[type="email"]`); err == nil {
		el.MustInput(email)
		time.Sleep(300 * time.Millisecond)
	} else {
		return "", fmt.Errorf("email field not found")
	}

	// Fill password
	passEls, _ := page.Elements(`input[type="password"]`)
	if len(passEls) >= 1 {
		passEls[0].MustInput(password)
		time.Sleep(300 * time.Millisecond)
	}
	if len(passEls) >= 2 {
		passEls[1].MustInput(password) // confirmation
		time.Sleep(200 * time.Millisecond)
	}

	// Check checkboxes (terms, marketing)
	checkboxes, _ := page.Elements(`input[type="checkbox"]`)
	for _, cb := range checkboxes {
		cb.MustClick()
		time.Sleep(200 * time.Millisecond)
	}

	// Submit
	if btn, err := page.Element(`button[type="submit"], input[type="submit"]`); err == nil {
		btn.MustClick()
	} else {
		return "", fmt.Errorf("submit button not found")
	}

	if verbose {
		fmt.Println("  form submitted, waiting...")
	}

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

		if strings.Contains(url, "dashboard") || strings.Contains(url, "api-key") {
			time.Sleep(3 * time.Second)
			html, _ := page.HTML()
			if key := extractZenserpKey(html); key != "" {
				return key, nil
			}
			// Try API keys page
			page.MustNavigate("https://app.zenserp.com/api-key")
			time.Sleep(3 * time.Second)
			html, _ = page.HTML()
			if key := extractZenserpKey(html); key != "" {
				return key, nil
			}
			return "", nil
		}

		html, _ := page.HTML()
		if strings.Contains(html, "verify") || strings.Contains(html, "confirm") || strings.Contains(html, "check your email") {
			return "", nil
		}
		if strings.Contains(html, "already") || strings.Contains(html, "taken") {
			return "", fmt.Errorf("email already registered")
		}
	}

	return "", fmt.Errorf("registration timed out")
}

var zenserpKeyRe = regexp.MustCompile(`[a-f0-9-]{32,48}`)

func extractZenserpKey(html string) string {
	matches := zenserpKeyRe.FindAllString(html, -1)
	for _, m := range matches {
		if len(m) >= 32 {
			return m
		}
	}
	return ""
}

func (r *zenserpRegistrar) VerifyAndGetKey(email, password, emailBody string, verbose bool) (string, error) {
	linkRe := regexp.MustCompile(`https://[^\s"<']*zenserp[^\s"<']*`)
	links := linkRe.FindAllString(emailBody, -1)

	l := launcher.New().Headless(false)
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	for _, link := range links {
		if strings.Contains(link, "confirm") || strings.Contains(link, "verify") {
			if verbose {
				fmt.Printf("  verification link: %s\n", link)
			}
			page := stealth.MustPage(browser)
			page = page.Timeout(90 * time.Second)
			page.MustNavigate(link)
			page.Timeout(30 * time.Second)
			time.Sleep(5 * time.Second)
			break
		}
	}

	// Login
	page := stealth.MustPage(browser)
	page = page.Timeout(90 * time.Second)
	page.MustNavigate("https://app.zenserp.com/login")
	page = page.Timeout(30 * time.Second)
	if err := page.WaitLoad(); err != nil {
		return "", err
	}
	time.Sleep(2 * time.Second)

	if el, err := page.Element(`input[type="email"], input[name="email"]`); err == nil {
		el.MustInput(email)
		time.Sleep(300 * time.Millisecond)
	}
	if el, err := page.Element(`input[type="password"]`); err == nil {
		el.MustInput(password)
		time.Sleep(300 * time.Millisecond)
	}
	if btn, err := page.Element(`button[type="submit"], input[type="submit"]`); err == nil {
		btn.MustClick()
	}
	time.Sleep(5 * time.Second)

	// Get API key
	page.MustNavigate("https://app.zenserp.com/api-key")
	time.Sleep(3 * time.Second)
	html, _ := page.HTML()
	if key := extractZenserpKey(html); key != "" {
		return key, nil
	}

	return "", fmt.Errorf("could not find API key")
}
