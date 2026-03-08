package serp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"
)

// RegisterOptions controls registration behavior.
type RegisterOptions struct {
	Verbose bool
}

// RegisterAccount creates a mail.tm email, registers on SerpAPI via rod browser,
// verifies email, signs in, and returns an Account with API key.
func RegisterAccount(opts RegisterOptions) (*Account, error) {
	mtc := NewMailTMClient()

	// 1. Pick domain
	if opts.Verbose {
		fmt.Println("  [1/7] picking mail.tm domain...")
	}
	domain, err := mtc.PickDomain()
	if err != nil {
		return nil, fmt.Errorf("mail.tm pick domain: %w", err)
	}

	// 2. Generate random email+password
	randHex := func(n int) string {
		b := make([]byte, n)
		rand.Read(b)
		return hex.EncodeToString(b)
	}
	email := randHex(6) + "@" + domain
	password := "Pass" + randHex(8) + "!"

	if opts.Verbose {
		fmt.Printf("  [2/7] email: %s\n", email)
	}

	// 3. Create mail.tm account
	if opts.Verbose {
		fmt.Println("  [3/7] creating mail.tm account...")
	}
	token, err := mtc.CreateAccount(email, password)
	if err != nil {
		return nil, fmt.Errorf("mail.tm create account: %w", err)
	}

	// 4. Register on SerpAPI via rod browser
	// HTTP registration doesn't work (reCAPTCHA blocks it), so go straight to rod.
	// If no proxy is set, try to auto-fetch one to avoid IP blocking.
	if os.Getenv("SERP_PROXY") == "" {
		if opts.Verbose {
			fmt.Println("  [4/7] fetching free proxy...")
		}
		if proxy, err := FetchFreeProxy(); err == nil {
			os.Setenv("SERP_PROXY", proxy)
			if opts.Verbose {
				fmt.Printf("  using proxy: %s\n", proxy)
			}
		} else if opts.Verbose {
			fmt.Printf("  no free proxy found (%v), using direct connection\n", err)
		}
	}
	if opts.Verbose {
		fmt.Println("  [4/7] registering on SerpAPI via browser...")
	}
	if err := RegisterWithRod(email, password); err != nil {
		return nil, fmt.Errorf("rod registration: %w", err)
	}

	// 5. Poll for verification email
	if opts.Verbose {
		fmt.Println("  [5/7] waiting for verification email...")
	}
	msg, err := mtc.PollForMessage(token, 90*time.Second)
	if err != nil {
		return nil, fmt.Errorf("verification email: %w", err)
	}

	// 6. Get message body and verify
	body, err := mtc.GetMessageBody(token, msg.ID)
	if err != nil {
		return nil, fmt.Errorf("get message body: %w", err)
	}

	if opts.Verbose {
		fmt.Println("  [6/7] verifying email...")
	}
	serpClient := NewSerpAPIClient()
	if err := serpClient.VerifyEmail(body); err != nil {
		return nil, fmt.Errorf("email verification: %w", err)
	}

	// 7. Sign in and get API key
	if opts.Verbose {
		fmt.Println("  [7/7] signing in to get API key...")
	}
	apiKey, err := serpClient.SignIn(email, password)
	if err != nil {
		return nil, fmt.Errorf("sign in: %w", err)
	}

	// Get initial searches_left
	info, err := serpClient.GetAccount(apiKey)
	if err != nil {
		info = &AccountInfo{TotalSearchesLeft: 100}
	}

	return &Account{
		Email:        email,
		Password:     password,
		APIKey:       apiKey,
		RegisteredAt: time.Now(),
		SearchesLeft: info.TotalSearchesLeft,
		LastChecked:  time.Now(),
	}, nil
}
