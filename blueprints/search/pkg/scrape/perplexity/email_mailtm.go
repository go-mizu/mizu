package perplexity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// mailTMAPIClient is a shared implementation for mail.tm and mail.gw (identical Hydra API).
type mailTMAPIClient struct {
	httpClient *http.Client
	baseURL    string
	email      string
	password   string
	token      string
	accountID  string
}

type mailTMDomainResp struct {
	Member []struct {
		Domain   string `json:"domain"`
		IsActive bool   `json:"isActive"`
	} `json:"hydra:member"`
}

type mailTMAccountResp struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

type mailTMTokenResp struct {
	Token string `json:"token"`
}

type mailTMMessagesResp struct {
	Member []mailTMMessage `json:"hydra:member"`
}

type mailTMMessage struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	From    struct {
		Address string `json:"address"`
	} `json:"from"`
	Intro     string `json:"intro"`
	CreatedAt string `json:"createdAt"`
}

type mailTMFullMessage struct {
	ID   string   `json:"id"`
	HTML []string `json:"html"`
	Text string   `json:"text"`
}

// newMailTMAPIClient creates a mail.tm/mail.gw client with the given base URL.
func newMailTMAPIClient(ctx context.Context, baseURL string) (*mailTMAPIClient, error) {
	mc := &mailTMAPIClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
	}

	// Step 1: Get available domain
	domain, err := mc.getDomain(ctx)
	if err != nil {
		return nil, fmt.Errorf("get domain: %w", err)
	}

	// Step 2: Create account with random address
	user := fmt.Sprintf("pplx%d%s", time.Now().Unix()%100000, randomString(5))
	mc.email = user + "@" + domain
	mc.password = randomString(16)

	if err := mc.createAccount(ctx); err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	// Step 3: Get JWT token
	if err := mc.getToken(ctx); err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	return mc, nil
}

// NewMailTMClient creates a new mail.tm client.
func NewMailTMClient(ctx context.Context) (*mailTMAPIClient, error) {
	return newMailTMAPIClient(ctx, "https://api.mail.tm")
}

// NewMailGWClient creates a new mail.gw client (identical API, different domains).
func NewMailGWClient(ctx context.Context) (*mailTMAPIClient, error) {
	return newMailTMAPIClient(ctx, "https://api.mail.gw")
}

func (mc *mailTMAPIClient) getDomain(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mc.baseURL+"/domains", nil)
	if err != nil {
		return "", err
	}

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var domResp mailTMDomainResp
	if err := json.NewDecoder(resp.Body).Decode(&domResp); err != nil {
		return "", err
	}

	for _, d := range domResp.Member {
		if d.IsActive {
			return d.Domain, nil
		}
	}
	return "", fmt.Errorf("no active domains available")
}

func (mc *mailTMAPIClient) createAccount(ctx context.Context) error {
	body, _ := json.Marshal(map[string]string{
		"address":  mc.email,
		"password": mc.password,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", mc.baseURL+"/accounts", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var acctResp mailTMAccountResp
	if err := json.NewDecoder(resp.Body).Decode(&acctResp); err != nil {
		return err
	}
	mc.accountID = acctResp.ID
	return nil
}

func (mc *mailTMAPIClient) getToken(ctx context.Context) error {
	body, _ := json.Marshal(map[string]string{
		"address":  mc.email,
		"password": mc.password,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", mc.baseURL+"/token", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var tokenResp mailTMTokenResp
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}
	mc.token = tokenResp.Token
	return nil
}

func (mc *mailTMAPIClient) Email() string {
	return mc.email
}

func (mc *mailTMAPIClient) WaitForMessage(ctx context.Context, matchSubject string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		messages, err := mc.listMessages(ctx)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		for _, msg := range messages {
			if strings.Contains(msg.Subject, matchSubject) || msg.Subject == matchSubject {
				return mc.readMessage(ctx, msg.ID)
			}
		}

		time.Sleep(3 * time.Second)
	}

	return "", fmt.Errorf("timeout waiting for email with subject %q", matchSubject)
}

func (mc *mailTMAPIClient) listMessages(ctx context.Context) ([]mailTMMessage, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mc.baseURL+"/messages", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+mc.token)

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("messages HTTP %d", resp.StatusCode)
	}

	var msgsResp mailTMMessagesResp
	if err := json.NewDecoder(resp.Body).Decode(&msgsResp); err != nil {
		return nil, err
	}

	return msgsResp.Member, nil
}

func (mc *mailTMAPIClient) readMessage(ctx context.Context, messageID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mc.baseURL+"/messages/"+messageID, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+mc.token)

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var msg mailTMFullMessage
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return "", err
	}

	if len(msg.HTML) > 0 {
		return strings.Join(msg.HTML, ""), nil
	}
	return msg.Text, nil
}
