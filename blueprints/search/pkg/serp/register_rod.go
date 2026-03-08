package serp

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// RegisterWithRod uses a headless browser to sign up on SerpAPI.
func RegisterWithRod(email, password string) error {
	l := launcher.New().Headless(true).MustLaunch()
	browser := rod.New().ControlURL(l).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://serpapi.com/users/sign_up")
	page = page.Timeout(30 * time.Second)

	if err := page.WaitLoad(); err != nil {
		return fmt.Errorf("page load: %w", err)
	}

	if err := page.MustElement(`input[name="user[email]"]`).Input(email); err != nil {
		return fmt.Errorf("fill email: %w", err)
	}
	if err := page.MustElement(`input[name="user[password]"]`).Input(password); err != nil {
		return fmt.Errorf("fill password: %w", err)
	}
	if err := page.MustElement(`input[name="user[password_confirmation]"]`).Input(password); err != nil {
		return fmt.Errorf("fill confirmation: %w", err)
	}
	page.MustElement(`input[type="submit"]`).MustClick()
	time.Sleep(2 * time.Second)
	return nil
}
