package perplexity

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var magicLinkRegex = regexp.MustCompile(`"(https://www\.perplexity\.ai/api/auth/callback/email\?callbackUrl=.*?)"`)

// Register creates a new Perplexity account using emailnator.
// Requires emailnator cookies (XSRF-TOKEN and laravel_session from emailnator.com).
// After successful registration, the client will have 5 pro queries and 10 file uploads.
func (c *Client) Register(ctx context.Context, emailCookies EmailnatorCookies) error {
	// Initialize session if not already done
	if c.csrfToken == "" {
		if err := c.InitSession(ctx); err != nil {
			return fmt.Errorf("init session: %w", err)
		}
	}

	// Generate disposable email
	emailClient, err := NewEmailnatorClient(ctx, emailCookies)
	if err != nil {
		return fmt.Errorf("create emailnator client: %w", err)
	}

	email := emailClient.Email()
	fmt.Printf("Generated email: %s\n", email)

	// Request magic link
	formData := fmt.Sprintf("email=%s&csrfToken=%s&callbackUrl=%s&json=true",
		email,
		c.csrfToken,
		"https://www.perplexity.ai/",
	)

	resp, err := c.doRequest(ctx, "POST", endpointAuthSignin,
		strings.NewReader(formData),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return fmt.Errorf("request signin: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("signin request failed: HTTP %d", resp.StatusCode)
	}

	fmt.Println("Sign-in email requested, waiting for magic link...")

	// Wait for the sign-in email
	msg, err := emailClient.WaitForMessage(ctx, signinSubject, accountTimeout)
	if err != nil {
		return fmt.Errorf("wait for email: %w", err)
	}

	// Open the email and extract the magic link
	content, err := emailClient.OpenMessage(ctx, msg.MessageID)
	if err != nil {
		return fmt.Errorf("open email: %w", err)
	}

	matches := magicLinkRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return fmt.Errorf("magic link not found in email")
	}
	magicLink := matches[1]

	// Complete registration by visiting the magic link
	authResp, err := c.doRequest(ctx, "GET", magicLink, nil, "")
	if err != nil {
		return fmt.Errorf("complete auth: %w", err)
	}
	defer authResp.Body.Close()
	io.Copy(io.Discard, authResp.Body)

	// Update account state
	c.mu.Lock()
	c.copilotQueries = defaultCopilotQueries
	c.fileUploads = defaultFileUploads
	c.authenticated = true
	c.mu.Unlock()

	// Save session
	if err := c.SaveSession(); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	fmt.Printf("Account created! Pro queries: %d, File uploads: %d\n",
		defaultCopilotQueries, defaultFileUploads)

	return nil
}
