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
	RegisterRegistrar("serpstack", &serpstackRegistrar{})
}

type serpstackRegistrar struct{}

// Register on serpstack.com (apilayer) via rod.
func (r *serpstackRegistrar) Register(email, password string, verbose bool) (string, error) {
	l := launcher.New().Headless(false)
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	// SerpStack signup is via apilayer
	page := stealth.MustPage(browser)
	page = page.Timeout(90 * time.Second)
	page.MustNavigate("https://serpstack.com/signup/free")
	page = page.Timeout(60 * time.Second)

	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("page load: %w", err)
	}
	time.Sleep(2 * time.Second)

	// Fill form fields — try common patterns
	for _, sel := range []string{
		`input[name="first_name"], input[placeholder*="first" i]`,
	} {
		if el, err := page.Element(sel); err == nil {
			el.MustInput(email[:6])
			time.Sleep(200 * time.Millisecond)
			break
		}
	}
	for _, sel := range []string{
		`input[name="last_name"], input[placeholder*="last" i]`,
	} {
		if el, err := page.Element(sel); err == nil {
			el.MustInput("User")
			time.Sleep(200 * time.Millisecond)
			break
		}
	}

	if emailEl, err := page.Element(`input[name="email"], input[type="email"]`); err == nil {
		emailEl.MustInput(email)
		time.Sleep(300 * time.Millisecond)
	} else {
		return "", fmt.Errorf("email field not found")
	}

	if passEl, err := page.Element(`input[name="password"], input[type="password"]`); err == nil {
		passEl.MustInput(password)
		time.Sleep(300 * time.Millisecond)
	} else {
		return "", fmt.Errorf("password field not found")
	}

	// Check any checkboxes (terms)
	checkboxes, _ := page.Elements(`input[type="checkbox"]`)
	for _, cb := range checkboxes {
		cb.MustClick()
		time.Sleep(200 * time.Millisecond)
	}

	// Submit
	if btn, err := page.Element(`button[type="submit"], input[type="submit"], button.signup`); err == nil {
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

		// Dashboard/quickstart page
		if strings.Contains(url, "dashboard") || strings.Contains(url, "quickstart") {
			time.Sleep(3 * time.Second)
			html, _ := page.HTML()
			if key := extractSerpStackKey(html); key != "" {
				return key, nil
			}
			// Navigate to dashboard
			page.MustNavigate("https://serpstack.com/dashboard")
			time.Sleep(3 * time.Second)
			html, _ = page.HTML()
			if key := extractSerpStackKey(html); key != "" {
				return key, nil
			}
			return "", nil
		}

		html, _ := page.HTML()
		if strings.Contains(html, "verify") || strings.Contains(html, "confirmation") {
			return "", nil
		}
		if strings.Contains(html, "already") {
			return "", fmt.Errorf("email already registered")
		}
	}

	return "", fmt.Errorf("registration timed out")
}

var serpstackKeyRe = regexp.MustCompile(`[a-f0-9]{32}`)

func extractSerpStackKey(html string) string {
	matches := serpstackKeyRe.FindAllString(html, -1)
	for _, m := range matches {
		if len(m) == 32 {
			return m
		}
	}
	return ""
}

func (r *serpstackRegistrar) VerifyAndGetKey(email, password, emailBody string, verbose bool) (string, error) {
	// Find verification link
	linkRe := regexp.MustCompile(`https://[^\s"<']*serpstack[^\s"<']*confirm[^\s"<']*|https://[^\s"<']*apilayer[^\s"<']*confirm[^\s"<']*`)
	link := linkRe.FindString(emailBody)
	if link == "" {
		linkRe = regexp.MustCompile(`https://[^\s"<']+`)
		for _, m := range linkRe.FindAllString(emailBody, -1) {
			if strings.Contains(m, "serpstack") || strings.Contains(m, "apilayer") {
				link = m
				break
			}
		}
	}

	l := launcher.New().Headless(false)
	controlURL := l.MustLaunch()
	browser := rod.New().ControlURL(controlURL).MustConnect()
	defer browser.MustClose()

	if link != "" {
		if verbose {
			fmt.Printf("  verification link: %s\n", link)
		}
		page := stealth.MustPage(browser)
		page = page.Timeout(90 * time.Second)
		page.MustNavigate(link)
		page.Timeout(30 * time.Second)
		time.Sleep(5 * time.Second)
	}

	// Navigate to dashboard
	page := stealth.MustPage(browser)
	page = page.Timeout(90 * time.Second)
	page.MustNavigate("https://serpstack.com/dashboard")
	page = page.Timeout(30 * time.Second)
	time.Sleep(5 * time.Second)

	html, _ := page.HTML()
	if key := extractSerpStackKey(html); key != "" {
		return key, nil
	}

	return "", fmt.Errorf("could not find API key")
}
