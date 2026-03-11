package perplexity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
)

// Client wraps HTTP with Chrome TLS fingerprinting for Perplexity access.
type Client struct {
	http           *http.Client
	cfg            Config
	cookies        *cookiejar.Jar
	csrfToken      string
	mu             sync.Mutex
	copilotQueries int
	fileUploads    int
	authenticated  bool
}

// NewClient creates a new Perplexity client with Chrome TLS fingerprint.
func NewClient(cfg Config) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	transport := &http.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialTLSChrome(ctx, network, addr)
		},
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		MaxIdleConnsPerHost: 10,
	}

	c := &Client{
		http: &http.Client{
			Transport: transport,
			Jar:       jar,
			Timeout:   cfg.Timeout,
		},
		cfg:     cfg,
		cookies: jar,
	}

	return c, nil
}

// dialTLSChrome establishes a TLS connection with Chrome's JA3 fingerprint.
// Forces HTTP/1.1 ALPN since Go's http.Transport doesn't support h2 via custom DialTLS.
func dialTLSChrome(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	dialer := &net.Dialer{Timeout: 15 * time.Second}
	rawConn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	// Get Chrome spec and modify ALPN to force HTTP/1.1
	spec, err := utls.UTLSIdToSpec(utls.HelloChrome_Auto)
	if err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("get chrome spec: %w", err)
	}

	// Replace ALPN extension to only include http/1.1
	for i, ext := range spec.Extensions {
		if alpn, ok := ext.(*utls.ALPNExtension); ok {
			spec.Extensions[i] = &utls.ALPNExtension{AlpnProtocols: []string{"http/1.1"}}
			_ = alpn
			break
		}
	}

	tlsConn := utls.UClient(rawConn, &utls.Config{
		ServerName: host,
	}, utls.HelloCustom)

	if err := tlsConn.ApplyPreset(&spec); err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("apply chrome spec: %w", err)
	}

	if err := tlsConn.HandshakeContext(ctx); err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("utls handshake: %w", err)
	}

	return tlsConn, nil
}

// InitSession establishes a session with Perplexity.
// Fetches /api/auth/session for Cloudflare cookies, then /api/auth/csrf for the CSRF token.
func (c *Client) InitSession(ctx context.Context) error {
	// Step 1: Hit session endpoint to establish Cloudflare cookies
	req, err := http.NewRequestWithContext(ctx, "GET", endpointAuthSession, nil)
	if err != nil {
		return err
	}
	setHeaders(req, defaultHeaders())

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("init session: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	// Step 2: Fetch CSRF token from dedicated endpoint
	csrfReq, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/auth/csrf", nil)
	if err != nil {
		return err
	}
	setHeaders(csrfReq, defaultHeaders())

	csrfResp, err := c.http.Do(csrfReq)
	if err != nil {
		return fmt.Errorf("get csrf: %w", err)
	}
	defer csrfResp.Body.Close()

	var csrfData struct {
		CsrfToken string `json:"csrfToken"`
	}
	if err := json.NewDecoder(csrfResp.Body).Decode(&csrfData); err == nil && csrfData.CsrfToken != "" {
		c.csrfToken = csrfData.CsrfToken
		return nil
	}

	// Fallback: extract CSRF token from cookies (older behavior)
	u, _ := url.Parse(baseURL)
	for _, cookie := range c.cookies.Cookies(u) {
		if cookie.Name == "next-auth.csrf-token" {
			parts := strings.SplitN(cookie.Value, "%", 2)
			c.csrfToken = parts[0]
			break
		}
	}

	if c.csrfToken == "" {
		for _, cookie := range c.cookies.Cookies(u) {
			if cookie.Name == "next-auth.csrf-token" {
				decoded, _ := url.QueryUnescape(cookie.Value)
				parts := strings.SplitN(decoded, "|", 2)
				c.csrfToken = parts[0]
				break
			}
		}
	}

	return nil
}

// LoadSession loads a saved session from disk.
func (c *Client) LoadSession() error {
	path := c.cfg.SessionPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var sess sessionData
	if err := json.Unmarshal(data, &sess); err != nil {
		return err
	}

	u, _ := url.Parse(baseURL)
	var cookies []*http.Cookie
	for _, cd := range sess.Cookies {
		cookies = append(cookies, &http.Cookie{
			Name:   cd.Name,
			Value:  cd.Value,
			Domain: cd.Domain,
			Path:   cd.Path,
		})
	}
	c.cookies.SetCookies(u, cookies)
	c.copilotQueries = sess.CopilotQueries
	c.fileUploads = sess.FileUploads
	c.authenticated = true

	// Re-extract CSRF token
	for _, cookie := range c.cookies.Cookies(u) {
		if cookie.Name == "next-auth.csrf-token" {
			parts := strings.SplitN(cookie.Value, "%", 2)
			c.csrfToken = parts[0]
			break
		}
	}

	return nil
}

// SaveSession saves the current session to disk.
func (c *Client) SaveSession() error {
	if err := os.MkdirAll(filepath.Dir(c.cfg.SessionPath()), 0o755); err != nil {
		return err
	}

	u, _ := url.Parse(baseURL)
	var cookies []*cookieData
	for _, cookie := range c.cookies.Cookies(u) {
		cookies = append(cookies, &cookieData{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Domain: cookie.Domain,
			Path:   cookie.Path,
		})
	}

	sess := sessionData{
		Cookies:        cookies,
		CopilotQueries: c.copilotQueries,
		FileUploads:    c.fileUploads,
		CreatedAt:      time.Now(),
	}

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.cfg.SessionPath(), data, 0o600)
}

// IsAuthenticated returns true if the client has an active session.
func (c *Client) IsAuthenticated() bool {
	return c.authenticated
}

// CopilotQueries returns remaining pro query count.
func (c *Client) CopilotQueries() int {
	return c.copilotQueries
}

// doRequest executes an HTTP request with Chrome headers.
func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	setHeaders(req, defaultHeaders())
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.http.Do(req)
}

// setHeaders copies headers to a request without overwriting Host.
func setHeaders(req *http.Request, headers http.Header) {
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
}

// loadSessionCookies applies cookie data to the client jar.
func (c *Client) loadSessionCookies(cookies []*cookieData) {
	u, _ := url.Parse(baseURL)
	var httpCookies []*http.Cookie
	for _, cd := range cookies {
		httpCookies = append(httpCookies, &http.Cookie{
			Name:   cd.Name,
			Value:  cd.Value,
			Domain: cd.Domain,
			Path:   cd.Path,
		})
	}
	c.cookies.SetCookies(u, httpCookies)

	// Re-extract CSRF token
	for _, cookie := range c.cookies.Cookies(u) {
		if cookie.Name == "next-auth.csrf-token" {
			parts := strings.SplitN(cookie.Value, "%", 2)
			c.csrfToken = parts[0]
			break
		}
	}
}

// parseBaseURL returns the parsed base URL.
func parseBaseURL() (*url.URL, error) {
	return url.Parse(baseURL)
}
