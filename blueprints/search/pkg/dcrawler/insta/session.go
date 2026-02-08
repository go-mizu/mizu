package insta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Session holds serializable session data.
type Session struct {
	Username string            `json:"username"`
	UserID   string            `json:"user_id"`
	Cookies  map[string]string `json:"cookies"`
	SavedAt  time.Time         `json:"saved_at"`
}

// Login authenticates with Instagram using username and password.
// On success, the client's session cookies are set for authenticated requests.
func (c *Client) Login(ctx context.Context, username, password string) error {
	// Step 1: Init to get CSRF token
	if err := c.Init(ctx); err != nil {
		return fmt.Errorf("init before login: %w", err)
	}

	// Step 2: POST login
	ts := fmt.Sprintf("%d", time.Now().Unix())
	encPassword := fmt.Sprintf("#PWD_INSTAGRAM_BROWSER:0:%s:%s", ts, password)

	form := url.Values{}
	form.Set("enc_password", encPassword)
	form.Set("username", username)

	// doPost may return body + error for non-200; we still need the body for login
	data, httpErr := c.doPost(ctx, LoginURL, form.Encode())
	if data == nil && httpErr != nil {
		return fmt.Errorf("login request: %w", httpErr)
	}

	var resp struct {
		Authenticated     bool   `json:"authenticated"`
		User              bool   `json:"user"`
		UserID            string `json:"userId"`
		Status            string `json:"status"`
		TwoFactorRequired bool   `json:"two_factor_required"`
		TwoFactorInfo     *struct {
			Identifier string `json:"two_factor_identifier"`
		} `json:"two_factor_info"`
		CheckpointURL string `json:"checkpoint_url"`
		Message       string `json:"message"`
		ErrorType     string `json:"error_type"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		if httpErr != nil {
			return fmt.Errorf("login request: %w", httpErr)
		}
		return fmt.Errorf("parse login response: %w", err)
	}

	if resp.TwoFactorRequired {
		identifier := ""
		if resp.TwoFactorInfo != nil {
			identifier = resp.TwoFactorInfo.Identifier
		}
		return &TwoFactorError{Identifier: identifier}
	}

	if resp.CheckpointURL != "" || resp.ErrorType == "ChallengeRequired" {
		url := resp.CheckpointURL
		if url == "" {
			url = "(verify account in browser)"
		}
		return &CheckpointError{URL: url}
	}

	if !resp.Authenticated {
		if !resp.User {
			return fmt.Errorf("user %q not found", username)
		}
		return fmt.Errorf("wrong password for %q", username)
	}

	c.username = username
	c.userID = resp.UserID
	c.loggedIn = true

	// Refresh CSRF token from cookies
	c.extractCSRF()

	return nil
}

// Login2FA completes a two-factor authentication login.
func (c *Client) Login2FA(ctx context.Context, username, code, identifier string) error {
	form := url.Values{}
	form.Set("username", username)
	form.Set("verificationCode", code)
	form.Set("identifier", identifier)

	data, err := c.doPost(ctx, TwoFactorURL, form.Encode())
	if err != nil {
		return fmt.Errorf("2fa request: %w", err)
	}

	var resp struct {
		Authenticated bool   `json:"authenticated"`
		UserID        string `json:"userId"`
		Status        string `json:"status"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parse 2fa response: %w", err)
	}

	if !resp.Authenticated {
		return fmt.Errorf("2fa authentication failed")
	}

	c.username = username
	c.userID = resp.UserID
	c.loggedIn = true
	c.extractCSRF()

	return nil
}

// SaveSession saves the current session cookies to a JSON file.
func (c *Client) SaveSession(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	cookies := make(map[string]string)
	u, _ := url.Parse("https://www.instagram.com/")
	for _, cookie := range c.http.Jar.Cookies(u) {
		cookies[cookie.Name] = cookie.Value
	}

	sess := Session{
		Username: c.username,
		UserID:   c.userID,
		Cookies:  cookies,
		SavedAt:  time.Now(),
	}

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600) // restricted permissions for credentials
}

// LoadSessionFile loads a session from a JSON file and applies it to the client.
func (c *Client) LoadSessionFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return fmt.Errorf("parse session: %w", err)
	}

	return c.ApplySession(&sess)
}

// ApplySession sets the client's cookies from a session.
func (c *Client) ApplySession(sess *Session) error {
	u, _ := url.Parse("https://www.instagram.com/")

	var cookies []*http.Cookie
	for name, value := range sess.Cookies {
		cookies = append(cookies, &http.Cookie{
			Name:   name,
			Value:  value,
			Domain: ".instagram.com",
			Path:   "/",
		})
	}
	c.http.Jar.SetCookies(u, cookies)

	c.username = sess.Username
	c.userID = sess.UserID
	c.loggedIn = true
	c.extractCSRF()

	return nil
}

// TestSession checks if the current session is valid by making a lightweight GraphQL query.
// Returns the authenticated username or an error.
func (c *Client) TestSession(ctx context.Context) (string, error) {
	data, err := c.graphQL(ctx, HashSessionTest, map[string]any{})
	if err != nil {
		return "", fmt.Errorf("session test: %w", err)
	}

	var resp struct {
		Data struct {
			User *struct {
				Username string `json:"username"`
			} `json:"user"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}

	if resp.Data.User == nil || resp.Data.User.Username == "" {
		return "", fmt.Errorf("session invalid or expired")
	}

	return resp.Data.User.Username, nil
}

// IsLoggedIn returns whether the client has an active session.
func (c *Client) IsLoggedIn() bool {
	return c.loggedIn
}

// Username returns the logged-in username.
func (c *Client) Username() string {
	return c.username
}

// SetUsername sets the username (useful after challenge verification).
func (c *Client) SetUsername(username string) {
	c.username = username
}

// extractCSRF reads the csrftoken cookie and stores it for request headers.
func (c *Client) extractCSRF() {
	u, _ := url.Parse("https://www.instagram.com/")
	for _, cookie := range c.http.Jar.Cookies(u) {
		if cookie.Name == "csrftoken" {
			c.csrfToken = cookie.Value
			break
		}
	}
}

// TwoFactorError indicates that 2FA is required to complete login.
type TwoFactorError struct {
	Identifier string
}

func (e *TwoFactorError) Error() string {
	return "two-factor authentication required"
}

// CheckpointError indicates that Instagram requires identity verification.
type CheckpointError struct {
	URL string
}

func (e *CheckpointError) Error() string {
	return fmt.Sprintf("checkpoint required: verify account at https://www.instagram.com%s", e.URL)
}

// ChallengeStart begins the checkpoint challenge flow by requesting a verification code.
// choice: 1 = email, 0 = SMS
func (c *Client) ChallengeStart(ctx context.Context, checkpointURL string, choice int) error {
	fullURL := "https://www.instagram.com" + checkpointURL

	// Step 1: GET the challenge page to get the form
	if _, err := c.doGet(ctx, fullURL); err != nil {
		// May return non-200, that's OK - we just need cookies set
	}

	// Step 2: POST the choice (email=1, sms=0) to request code
	form := url.Values{}
	form.Set("choice", fmt.Sprintf("%d", choice))
	if _, err := c.doPost(ctx, fullURL, form.Encode()); err != nil {
		return fmt.Errorf("challenge start: %w", err)
	}

	return nil
}

// ChallengeVerify submits the verification code for a checkpoint challenge.
func (c *Client) ChallengeVerify(ctx context.Context, checkpointURL string, code string) error {
	fullURL := "https://www.instagram.com" + checkpointURL

	form := url.Values{}
	form.Set("security_code", code)

	data, err := c.doPost(ctx, fullURL, form.Encode())
	if err != nil {
		// Try to parse JSON even on error
		if data == nil {
			return fmt.Errorf("challenge verify: %w", err)
		}
	}

	// Check if challenge was passed
	var resp struct {
		Status        string `json:"status"`
		Message       string `json:"message"`
		Authenticated bool   `json:"authenticated"`
		UserID        string `json:"userId"`
	}
	if err := json.Unmarshal(data, &resp); err == nil {
		if resp.Authenticated {
			c.loggedIn = true
			if resp.UserID != "" {
				c.userID = resp.UserID
			}
			c.extractCSRF()
			return nil
		}
		if resp.Message != "" {
			return fmt.Errorf("challenge: %s", resp.Message)
		}
	}

	// If we got cookies with sessionid, we're likely authenticated
	u, _ := url.Parse("https://www.instagram.com/")
	for _, cookie := range c.http.Jar.Cookies(u) {
		if cookie.Name == "sessionid" && cookie.Value != "" {
			c.loggedIn = true
			c.extractCSRF()
			return nil
		}
	}

	return fmt.Errorf("challenge verification failed (code may be wrong)")
}

// doPost performs a POST request with standard headers.
func (c *Client) doPost(ctx context.Context, rawURL string, body string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", rawURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setWebHeaders(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := readBody(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return respBody, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	return respBody, nil
}

// docIDQuery performs a doc_id-based GraphQL POST query with rate limiting.
func (c *Client) docIDQuery(ctx context.Context, docID string, variables map[string]any) ([]byte, error) {
	if err := c.waitRate(ctx, queryTypeGraphQL, docID); err != nil {
		return nil, err
	}

	varsJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("variables", string(varsJSON))
	form.Set("doc_id", docID)
	form.Set("server_timestamps", "true")

	return c.doPost(ctx, GraphQLURL, form.Encode())
}
