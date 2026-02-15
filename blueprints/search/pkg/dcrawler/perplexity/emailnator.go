package perplexity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

// EmailnatorClient generates disposable emails and reads messages.
type EmailnatorClient struct {
	httpClient *http.Client
	headers    http.Header
	email      string
	adIDs      map[string]bool // advertisement message IDs to skip
	cookies    EmailnatorCookies
}

// EmailnatorCookies holds the cookies needed for emailnator.com.
type EmailnatorCookies struct {
	XSRFToken      string `json:"xsrf_token"`
	LaravelSession string `json:"laravel_session"`
}

// FetchEmailnatorCookies visits emailnator.com to automatically obtain XSRF-TOKEN
// and laravel_session cookies. Uses Chrome TLS fingerprinting to bypass protection.
func FetchEmailnatorCookies(ctx context.Context) (*EmailnatorCookies, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	transport := &http.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialTLSChrome(ctx, network, addr)
		},
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  false,
	}

	client := &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   defaultTimeout,
	}

	// Visit the main page to get cookies set
	req, err := http.NewRequestWithContext(ctx, "GET", emailnatorBase, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", chromeUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch emailnator page: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("emailnator returned HTTP %d", resp.StatusCode)
	}

	// Extract cookies
	u, _ := url.Parse(emailnatorBase)
	var xsrf, laravel string
	for _, cookie := range jar.Cookies(u) {
		switch cookie.Name {
		case "XSRF-TOKEN":
			xsrf = cookie.Value
		case "laravel_session":
			laravel = cookie.Value
		}
	}

	if xsrf == "" || laravel == "" {
		return nil, fmt.Errorf("emailnator cookies not found (XSRF=%v, laravel=%v)", xsrf != "", laravel != "")
	}

	return &EmailnatorCookies{
		XSRFToken:      xsrf,
		LaravelSession: laravel,
	}, nil
}

// NewEmailnatorClient creates a client and generates a disposable email.
func NewEmailnatorClient(ctx context.Context, cookies EmailnatorCookies) (*EmailnatorClient, error) {
	// URL-decode XSRF token for the header
	xsrfDecoded, _ := url.QueryUnescape(cookies.XSRFToken)

	ec := &EmailnatorClient{
		httpClient: &http.Client{Timeout: defaultTimeout},
		headers:    emailnatorHeaders(xsrfDecoded),
		adIDs:      make(map[string]bool),
		cookies:    cookies,
	}

	// Generate email
	body, _ := json.Marshal(map[string][]string{"email": {"googleMail"}})
	req, err := http.NewRequestWithContext(ctx, "POST", emailnatorGenerate, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	ec.setRequestHeaders(req)

	resp, err := ec.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("generate email: %w", err)
	}
	defer resp.Body.Close()

	var genResp emailnatorResp
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("parse email response: %w", err)
	}
	if len(genResp.Email) == 0 {
		return nil, fmt.Errorf("no email generated")
	}
	ec.email = genResp.Email[0]

	// Load initial ads
	msgs, err := ec.listMessages(ctx)
	if err == nil {
		for _, m := range msgs {
			ec.adIDs[m.MessageID] = true
		}
	}

	return ec, nil
}

// Email returns the generated disposable email address.
func (ec *EmailnatorClient) Email() string {
	return ec.email
}

// setRequestHeaders sets headers and cookies on a request.
func (ec *EmailnatorClient) setRequestHeaders(req *http.Request) {
	for k, vs := range ec.headers {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	req.AddCookie(&http.Cookie{Name: "XSRF-TOKEN", Value: ec.cookies.XSRFToken})
	req.AddCookie(&http.Cookie{Name: "laravel_session", Value: ec.cookies.LaravelSession})
}

// listMessages fetches the inbox.
func (ec *EmailnatorClient) listMessages(ctx context.Context) ([]emailnatorMessage, error) {
	body, _ := json.Marshal(map[string]string{"email": ec.email})
	req, err := http.NewRequestWithContext(ctx, "POST", emailnatorMessages, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	ec.setRequestHeaders(req)

	resp, err := ec.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var msgList emailnatorMessageList
	if err := json.NewDecoder(resp.Body).Decode(&msgList); err != nil {
		return nil, err
	}

	return msgList.MessageData, nil
}

// WaitForMessage polls for a message matching the subject.
func (ec *EmailnatorClient) WaitForMessage(ctx context.Context, matchSubject string, timeout time.Duration) (*emailnatorMessage, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		msgs, err := ec.listMessages(ctx)
		if err != nil {
			time.Sleep(emailRetryDelay)
			continue
		}

		for i := range msgs {
			if ec.adIDs[msgs[i].MessageID] {
				continue
			}
			if msgs[i].Subject == matchSubject {
				return &msgs[i], nil
			}
		}

		time.Sleep(emailRetryDelay)
	}

	return nil, fmt.Errorf("timeout waiting for email with subject %q", matchSubject)
}

// OpenMessage reads the content of a specific message.
func (ec *EmailnatorClient) OpenMessage(ctx context.Context, messageID string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"email":     ec.email,
		"messageID": messageID,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", emailnatorMessages, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	ec.setRequestHeaders(req)

	resp, err := ec.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
