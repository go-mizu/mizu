package perplexity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
