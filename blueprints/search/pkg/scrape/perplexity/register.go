package perplexity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

var magicLinkRegex = regexp.MustCompile(`"(https://www\.perplexity\.ai/api/auth/callback/email\?callbackUrl=.*?)"`)

// AutoRegister creates a new Perplexity account using a disposable email.
// Uses mail.tm (primary) or Guerrilla Mail (fallback) for the temp email.
// If db is non-nil, the account is saved to the database.
func (c *Client) AutoRegister(ctx context.Context, db *DB) error {
	// Initialize session if not already done
	if c.csrfToken == "" {
		if err := c.InitSession(ctx); err != nil {
			return fmt.Errorf("init session: %w", err)
		}
	}

	// Create disposable email
	fmt.Println("Creating disposable email...")
	emailClient, err := NewTempEmailClient(ctx)
	if err != nil {
		return fmt.Errorf("create temp email: %w", err)
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

	// Wait for the sign-in email — new interface returns body directly
	content, err := emailClient.WaitForMessage(ctx, signinSubject, accountTimeout)
	if err != nil {
		return fmt.Errorf("wait for email: %w", err)
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

	// Save session file (backward compat)
	if err := c.SaveSession(); err != nil {
		fmt.Printf("Warning: could not save session file: %v\n", err)
	}

	// Save to database if available
	if db != nil {
		sess := c.exportSession()
		sessionJSON, _ := json.Marshal(sess)
		account := &Account{
			Email:       email,
			Source:      "temp_email",
			SessionData: string(sessionJSON),
			ProQueries:  defaultCopilotQueries,
			FileUploads: defaultFileUploads,
			Status:      AccountActive,
			CreatedAt:   time.Now(),
		}
		if err := db.SaveAccount(account); err != nil {
			fmt.Printf("Warning: could not save account to DB: %v\n", err)
		} else {
			fmt.Printf("Account saved to DB with ID %d\n", account.ID)
		}
	}

	fmt.Printf("Account created! Pro queries: %d, File uploads: %d\n",
		defaultCopilotQueries, defaultFileUploads)

	return nil
}

// exportSession exports the current session as a sessionData.
func (c *Client) exportSession() *sessionData {
	sess := &sessionData{
		CopilotQueries: c.copilotQueries,
		FileUploads:    c.fileUploads,
		CreatedAt:      time.Now(),
	}

	// Export cookies
	u, _ := parseBaseURL()
	for _, cookie := range c.cookies.Cookies(u) {
		sess.Cookies = append(sess.Cookies, &cookieData{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Domain: cookie.Domain,
			Path:   cookie.Path,
		})
	}

	return sess
}
