package serp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// RegisterOptions controls registration behavior.
type RegisterOptions struct {
	UseRod  bool // force rod browser (skip HTTP attempt)
	Verbose bool
}

// RegisterAccount creates a mail.tm email, registers on SerpAPI, verifies, and returns an Account.
func RegisterAccount(opts RegisterOptions) (*Account, error) {
	mtc := NewMailTMClient()

	// 1. Pick domain
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
	password := randHex(12)

	if opts.Verbose {
		fmt.Printf("  email: %s\n", email)
	}

	// 3. Create mail.tm account
	token, err := mtc.CreateAccount(email, password)
	if err != nil {
		return nil, fmt.Errorf("mail.tm create account: %w", err)
	}

	// 4. Register on SerpAPI (HTTP first, rod fallback)
	serpClient := NewSerpAPIClient()
	if !opts.UseRod {
		httpErr := serpClient.RegisterHTTP(email, password)
		if httpErr != nil {
			if opts.Verbose {
				fmt.Printf("  HTTP registration failed (%v), trying rod browser...\n", httpErr)
			}
			if err := RegisterWithRod(email, password); err != nil {
				return nil, fmt.Errorf("rod registration failed: %w", err)
			}
		}
	} else {
		if err := RegisterWithRod(email, password); err != nil {
			return nil, fmt.Errorf("rod registration failed: %w", err)
		}
	}

	// 5. Poll for verification email
	if opts.Verbose {
		fmt.Println("  waiting for verification email...")
	}
	msg, err := mtc.PollForMessage(token, 90*time.Second)
	if err != nil {
		return nil, fmt.Errorf("verification email: %w", err)
	}

	// 6. Get message body
	body, err := mtc.GetMessageBody(token, msg.ID)
	if err != nil {
		return nil, fmt.Errorf("get message body: %w", err)
	}

	// 7. Verify email
	if err := serpClient.VerifyEmail(body); err != nil {
		return nil, fmt.Errorf("email verification: %w", err)
	}

	// 8. Sign in and get API key
	if opts.Verbose {
		fmt.Println("  signing in to get API key...")
	}
	apiKey, err := serpClient.SignIn(email, password)
	if err != nil {
		return nil, fmt.Errorf("sign in: %w", err)
	}

	// 9. Get initial searches_left
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
