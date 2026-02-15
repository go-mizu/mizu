package perplexity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const guerrillaAPI = "https://api.guerrillamail.com/ajax.php"

// GuerrillaClient uses the Guerrilla Mail API for disposable emails.
type GuerrillaClient struct {
	httpClient *http.Client
	email      string
	sidToken   string
}

type guerrillaEmailResp struct {
	EmailAddr string `json:"email_addr"`
	SIDToken  string `json:"sid_token"`
}

type guerrillaInboxResp struct {
	List []guerrillaMsg `json:"list"`
}

type guerrillaMsg struct {
	MailID      json.Number `json:"mail_id"`
	MailSubject string      `json:"mail_subject"`
	MailFrom    string      `json:"mail_from"`
	MailBody    string      `json:"mail_body"`
}

// NewGuerrillaClient creates a new Guerrilla Mail client with a random email.
func NewGuerrillaClient(ctx context.Context) (*GuerrillaClient, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	gc := &GuerrillaClient{httpClient: client}

	req, err := http.NewRequestWithContext(ctx, "GET", guerrillaAPI+"?f=get_email_address", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", chromeUA)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get address: %w", err)
	}
	defer resp.Body.Close()

	var emailResp guerrillaEmailResp
	if err := json.NewDecoder(resp.Body).Decode(&emailResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	gc.email = emailResp.EmailAddr
	gc.sidToken = emailResp.SIDToken

	return gc, nil
}

func (gc *GuerrillaClient) Email() string {
	return gc.email
}

func (gc *GuerrillaClient) WaitForMessage(ctx context.Context, matchSubject string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		messages, err := gc.checkInbox(ctx)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		for _, msg := range messages {
			if strings.Contains(msg.MailSubject, matchSubject) || msg.MailSubject == matchSubject {
				body, err := gc.fetchEmail(ctx, msg.MailID.String())
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

func (gc *GuerrillaClient) checkInbox(ctx context.Context) ([]guerrillaMsg, error) {
	url := fmt.Sprintf("%s?f=check_email&sid_token=%s&seq=0", guerrillaAPI, gc.sidToken)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", chromeUA)

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var inbox guerrillaInboxResp
	if err := json.NewDecoder(resp.Body).Decode(&inbox); err != nil {
		return nil, err
	}

	return inbox.List, nil
}

func (gc *GuerrillaClient) fetchEmail(ctx context.Context, emailID string) (string, error) {
	url := fmt.Sprintf("%s?f=fetch_email&sid_token=%s&email_id=%s", guerrillaAPI, gc.sidToken, emailID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", chromeUA)

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var msg guerrillaMsg
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		body, _ := io.ReadAll(resp.Body)
		return string(body), nil
	}

	return msg.MailBody, nil
}
