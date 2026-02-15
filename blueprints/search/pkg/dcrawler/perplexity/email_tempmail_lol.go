package perplexity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const tempMailLolAPI = "https://api.tempmail.lol"

// TempMailLolClient uses tempmail.lol for disposable emails.
type TempMailLolClient struct {
	httpClient *http.Client
	email      string
	token      string
}

type tempMailLolGenerateResp struct {
	Address string `json:"address"`
	Token   string `json:"token"`
}

type tempMailLolInboxResp struct {
	Email []tempMailLolMessage `json:"email"`
}

type tempMailLolMessage struct {
	From    string `json:"from"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	HTML    string `json:"html"`
	Date    string `json:"date"`
}

// NewTempMailLolClient creates a new tempmail.lol client.
func NewTempMailLolClient(ctx context.Context) (*TempMailLolClient, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", tempMailLolAPI+"/generate", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("generate HTTP %d", resp.StatusCode)
	}

	var gen tempMailLolGenerateResp
	if err := json.NewDecoder(resp.Body).Decode(&gen); err != nil {
		return nil, fmt.Errorf("parse generate: %w", err)
	}

	if gen.Address == "" || gen.Token == "" {
		return nil, fmt.Errorf("empty address or token")
	}

	return &TempMailLolClient{
		httpClient: client,
		email:      gen.Address,
		token:      gen.Token,
	}, nil
}

func (c *TempMailLolClient) Email() string {
	return c.email
}

func (c *TempMailLolClient) WaitForMessage(ctx context.Context, matchSubject string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		messages, err := c.checkInbox(ctx)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		for _, msg := range messages {
			if strings.Contains(msg.Subject, matchSubject) || msg.Subject == matchSubject {
				if msg.HTML != "" {
					return msg.HTML, nil
				}
				return msg.Body, nil
			}
		}

		time.Sleep(3 * time.Second)
	}

	return "", fmt.Errorf("timeout waiting for email with subject %q", matchSubject)
}

func (c *TempMailLolClient) checkInbox(ctx context.Context) ([]tempMailLolMessage, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tempMailLolAPI+"/auth/"+c.token, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("inbox HTTP %d", resp.StatusCode)
	}

	var inbox tempMailLolInboxResp
	if err := json.NewDecoder(resp.Body).Decode(&inbox); err != nil {
		return nil, err
	}

	return inbox.Email, nil
}
