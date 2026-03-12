package serp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// ProviderRegisterOptions controls provider auto-registration.
type ProviderRegisterOptions struct {
	Provider string
	Verbose  bool
}

// RegisterProvider creates a mail.tm email, signs up on the given provider,
// verifies email if needed, and returns an Account with API key.
func RegisterProvider(opts ProviderRegisterOptions) (*Account, error) {
	reg, ok := providerRegistrars[opts.Provider]
	if !ok {
		return nil, fmt.Errorf("no auto-registration for provider %q", opts.Provider)
	}

	mtc := NewMailTMClient()

	// 1. Pick domain
	if opts.Verbose {
		fmt.Println("  [1] picking mail.tm domain...")
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
	password := "Pass" + randHex(8) + "!1"

	if opts.Verbose {
		fmt.Printf("  [2] email: %s\n", email)
	}

	// 3. Create mail.tm account
	if opts.Verbose {
		fmt.Println("  [3] creating mail.tm account...")
	}
	token, err := mtc.CreateAccount(email, password)
	if err != nil {
		return nil, fmt.Errorf("mail.tm create account: %w", err)
	}

	// 4. Register on provider
	if opts.Verbose {
		fmt.Printf("  [4] registering on %s...\n", opts.Provider)
	}
	apiKey, err := reg.Register(email, password, opts.Verbose)
	if err != nil {
		return nil, fmt.Errorf("%s registration: %w", opts.Provider, err)
	}

	// If we got an API key directly (no email verification needed), we're done
	if apiKey != "" {
		if opts.Verbose {
			fmt.Printf("  [5] got API key directly: %s...\n", apiKey[:8])
		}
		return &Account{
			Email:        email,
			Password:     password,
			APIKey:       apiKey,
			Provider:     opts.Provider,
			RegisteredAt: time.Now(),
			SearchesLeft: 9999,
			LastChecked:  time.Now(),
		}, nil
	}

	// 5. Poll for verification email
	if opts.Verbose {
		fmt.Println("  [5] waiting for verification email...")
	}
	msg, err := mtc.PollForMessage(token, 90*time.Second)
	if err != nil {
		return nil, fmt.Errorf("verification email: %w", err)
	}

	body, err := mtc.GetMessageBody(token, msg.ID)
	if err != nil {
		return nil, fmt.Errorf("get message body: %w", err)
	}

	// 6. Provider-specific verification + API key extraction
	if opts.Verbose {
		fmt.Println("  [6] verifying email + extracting API key...")
	}
	apiKey, err = reg.VerifyAndGetKey(email, password, body, opts.Verbose)
	if err != nil {
		// If the error indicates onboarding is needed (not a hard failure),
		// return account without key so the user can complete manually
		if strings.Contains(err.Error(), "credit card") || strings.Contains(err.Error(), "onboarding") {
			if opts.Verbose {
				fmt.Printf("  NOTE: %v\n", err)
			}
			return &Account{
				Email:        email,
				Password:     password,
				Provider:     opts.Provider,
				RegisteredAt: time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("%s verify: %w", opts.Provider, err)
	}

	return &Account{
		Email:        email,
		Password:     password,
		APIKey:       apiKey,
		Provider:     opts.Provider,
		RegisteredAt: time.Now(),
		SearchesLeft: 9999,
		LastChecked:  time.Now(),
	}, nil
}

// ProviderRegistrar handles signup for a specific provider.
type ProviderRegistrar interface {
	// Register signs up. Returns apiKey if available immediately, or "" if email verification needed.
	Register(email, password string, verbose bool) (apiKey string, err error)
	// VerifyAndGetKey verifies the email and returns the API key.
	VerifyAndGetKey(email, password, emailBody string, verbose bool) (apiKey string, err error)
}

var providerRegistrars = map[string]ProviderRegistrar{}

// RegisterRegistrar registers a provider registrar.
func RegisterRegistrar(name string, r ProviderRegistrar) {
	providerRegistrars[name] = r
}

// HasRegistrar returns true if auto-registration is available for the provider.
func HasRegistrar(name string) bool {
	_, ok := providerRegistrars[name]
	return ok
}
