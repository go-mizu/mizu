package perplexity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const inboxKittenAPI = "https://inboxkitten.com/api/v1"

var bodyRegex = regexp.MustCompile(`(?s)<body[^>]*>(.*?)</body>`)

// InboxKittenClient uses inboxkitten.com for disposable emails.
// No auth required — public inbox based on recipient name.
type InboxKittenClient struct {
	httpClient *http.Client
	email      string
	user       string
}

type inboxKittenMessage struct {
	Key     string `json:"key"`
	Subject string `json:"subject"`
}

// NewInboxKittenClient creates a new inboxkitten.com client.
// No API call needed — just generate a random address.
func NewInboxKittenClient(ctx context.Context) (*InboxKittenClient, error) {
	user := fmt.Sprintf("pplx%d%s", time.Now().Unix()%100000, randomString(5))
	return &InboxKittenClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		email:      user + "@inboxkitten.com",
		user:       user,
	}, nil
}

func (c *InboxKittenClient) Email() string {
	return c.email
}

func (c *InboxKittenClient) WaitForMessage(ctx context.Context, matchSubject string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		messages, err := c.listMessages(ctx)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		for _, msg := range messages {
			if strings.Contains(msg.Subject, matchSubject) || msg.Subject == matchSubject {
				body, err := c.readMessage(ctx, msg.Key)
				if err != nil {
					continue
				}
				return body, nil
			}
		}

		time.Sleep(3 * time.Second)
	}

	return "", fmt.Errorf("timeout waiting for email with subject %q", matchSubject)
}

func (c *InboxKittenClient) listMessages(ctx context.Context) ([]inboxKittenMessage, error) {
	url := fmt.Sprintf("%s/mail/list?recipient=%s", inboxKittenAPI, c.user)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("list HTTP %d", resp.StatusCode)
	}

	var messages []inboxKittenMessage
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (c *InboxKittenClient) readMessage(ctx context.Context, key string) (string, error) {
	url := fmt.Sprintf("%s/mail/getMail?key=%s", inboxKittenAPI, key)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Try to extract HTML body
	body := string(raw)
	if matches := bodyRegex.FindStringSubmatch(body); len(matches) > 1 {
		return matches[1], nil
	}
	return body, nil
}
