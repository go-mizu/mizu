package perplexity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const tempMailPlusAPI = "https://tempmail.plus/api"

// TempMailPlusClient uses tempmail.plus for disposable emails.
// No auth required — public inbox based on email address.
type TempMailPlusClient struct {
	httpClient *http.Client
	email      string
	user       string
}

type tempMailPlusInboxResp struct {
	MailList []tempMailPlusMessage `json:"mail_list"`
}

type tempMailPlusMessage struct {
	MailID  int    `json:"mail_id"`
	Subject string `json:"subject"`
	From    string `json:"from"`
}

type tempMailPlusFullMessage struct {
	Text string `json:"text"`
	HTML string `json:"html"`
}

// NewTempMailPlusClient creates a new tempmail.plus client.
// No API call needed — just generate a random address.
func NewTempMailPlusClient(ctx context.Context) (*TempMailPlusClient, error) {
	user := fmt.Sprintf("pplx%d%s", time.Now().Unix()%100000, randomString(5))
	return &TempMailPlusClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		email:      user + "@mailto.plus",
		user:       user,
	}, nil
}

func (c *TempMailPlusClient) Email() string {
	return c.email
}

func (c *TempMailPlusClient) WaitForMessage(ctx context.Context, matchSubject string, timeout time.Duration) (string, error) {
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
				body, err := c.readMessage(ctx, msg.MailID)
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

func (c *TempMailPlusClient) checkInbox(ctx context.Context) ([]tempMailPlusMessage, error) {
	url := fmt.Sprintf("%s/mails?email=%s&epin=", tempMailPlusAPI, c.email)
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
		return nil, fmt.Errorf("inbox HTTP %d", resp.StatusCode)
	}

	var inbox tempMailPlusInboxResp
	if err := json.NewDecoder(resp.Body).Decode(&inbox); err != nil {
		return nil, err
	}

	return inbox.MailList, nil
}

func (c *TempMailPlusClient) readMessage(ctx context.Context, mailID int) (string, error) {
	url := fmt.Sprintf("%s/mails/%d?email=%s&epin=", tempMailPlusAPI, mailID, c.email)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var msg tempMailPlusFullMessage
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return "", err
	}

	if msg.HTML != "" {
		return msg.HTML, nil
	}
	return msg.Text, nil
}
