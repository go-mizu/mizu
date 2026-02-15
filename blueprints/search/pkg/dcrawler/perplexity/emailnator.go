package perplexity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

// TempEmailClient provides disposable email addresses for registration.
// Implementations: MailTMClient (primary), GuerrillaClient (fallback).
type TempEmailClient interface {
	Email() string
	WaitForMessage(ctx context.Context, matchSubject string, timeout time.Duration) (string, error) // returns message body
}

// --- mail.tm implementation ---

const (
	mailTMAPI = "https://api.mail.tm"
)

// MailTMClient uses the mail.tm REST API for disposable emails.
type MailTMClient struct {
	httpClient *http.Client
	email      string
	password   string
	token      string
	accountID  string
}

// mailTMDomainResp is the response from GET /domains.
type mailTMDomainResp struct {
	Member []struct {
		Domain   string `json:"domain"`
		IsActive bool   `json:"isActive"`
	} `json:"hydra:member"`
}

// mailTMAccountResp is the response from POST /accounts.
type mailTMAccountResp struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// mailTMTokenResp is the response from POST /token.
type mailTMTokenResp struct {
	Token string `json:"token"`
}

// mailTMMessagesResp is the response from GET /messages.
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

// mailTMFullMessage is the response from GET /messages/{id}.
type mailTMFullMessage struct {
	ID   string `json:"id"`
	HTML []string `json:"html"`
	Text string `json:"text"`
}

// NewMailTMClient creates a new mail.tm client with a random email address.
func NewMailTMClient(ctx context.Context) (*MailTMClient, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	mc := &MailTMClient{httpClient: client}

	// Step 1: Get available domain
	domain, err := mc.getDomain(ctx)
	if err != nil {
		return nil, fmt.Errorf("get mail.tm domain: %w", err)
	}

	// Step 2: Create account with random address
	user := fmt.Sprintf("pplx%d%s", time.Now().Unix()%100000, randomString(5))
	mc.email = user + "@" + domain
	mc.password = randomString(16)

	if err := mc.createAccount(ctx); err != nil {
		return nil, fmt.Errorf("create mail.tm account: %w", err)
	}

	// Step 3: Get JWT token
	if err := mc.getToken(ctx); err != nil {
		return nil, fmt.Errorf("get mail.tm token: %w", err)
	}

	return mc, nil
}

func (mc *MailTMClient) getDomain(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mailTMAPI+"/domains", nil)
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

func (mc *MailTMClient) createAccount(ctx context.Context) error {
	body, _ := json.Marshal(map[string]string{
		"address":  mc.email,
		"password": mc.password,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", mailTMAPI+"/accounts", bytes.NewReader(body))
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
		return fmt.Errorf("mail.tm create account HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var acctResp mailTMAccountResp
	if err := json.NewDecoder(resp.Body).Decode(&acctResp); err != nil {
		return err
	}
	mc.accountID = acctResp.ID
	return nil
}

func (mc *MailTMClient) getToken(ctx context.Context) error {
	body, _ := json.Marshal(map[string]string{
		"address":  mc.email,
		"password": mc.password,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", mailTMAPI+"/token", bytes.NewReader(body))
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
		return fmt.Errorf("mail.tm token HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var tokenResp mailTMTokenResp
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}
	mc.token = tokenResp.Token
	return nil
}

// Email returns the disposable email address.
func (mc *MailTMClient) Email() string {
	return mc.email
}

// WaitForMessage polls the inbox for a message matching the subject.
// Returns the message body (HTML content).
func (mc *MailTMClient) WaitForMessage(ctx context.Context, matchSubject string, timeout time.Duration) (string, error) {
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
				// Fetch full message content
				return mc.readMessage(ctx, msg.ID)
			}
		}

		time.Sleep(3 * time.Second)
	}

	return "", fmt.Errorf("timeout waiting for email with subject %q", matchSubject)
}

func (mc *MailTMClient) listMessages(ctx context.Context) ([]mailTMMessage, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mailTMAPI+"/messages", nil)
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
		return nil, fmt.Errorf("mail.tm messages HTTP %d", resp.StatusCode)
	}

	var msgsResp mailTMMessagesResp
	if err := json.NewDecoder(resp.Body).Decode(&msgsResp); err != nil {
		return nil, err
	}

	return msgsResp.Member, nil
}

func (mc *MailTMClient) readMessage(ctx context.Context, messageID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mailTMAPI+"/messages/"+messageID, nil)
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

	// Prefer HTML content (contains magic link), fall back to text
	if len(msg.HTML) > 0 {
		return strings.Join(msg.HTML, ""), nil
	}
	return msg.Text, nil
}

// --- Guerrilla Mail implementation (fallback) ---

const (
	guerrillaAPI = "https://api.guerrillamail.com/ajax.php"
)

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
		return nil, fmt.Errorf("guerrillamail get address: %w", err)
	}
	defer resp.Body.Close()

	var emailResp guerrillaEmailResp
	if err := json.NewDecoder(resp.Body).Decode(&emailResp); err != nil {
		return nil, fmt.Errorf("parse guerrillamail response: %w", err)
	}

	gc.email = emailResp.EmailAddr
	gc.sidToken = emailResp.SIDToken

	return gc, nil
}

// Email returns the disposable email address.
func (gc *GuerrillaClient) Email() string {
	return gc.email
}

// WaitForMessage polls the inbox for a message matching the subject.
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
				// Fetch full email body
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
		// Maybe raw body
		body, _ := io.ReadAll(resp.Body)
		return string(body), nil
	}

	return msg.MailBody, nil
}

// --- Provider factory ---

// NewTempEmailClient creates a disposable email client.
// Tries mail.tm first, falls back to Guerrilla Mail.
func NewTempEmailClient(ctx context.Context) (TempEmailClient, error) {
	// Try mail.tm first (private inbox, JWT auth)
	mc, err := NewMailTMClient(ctx)
	if err == nil {
		return mc, nil
	}
	fmt.Printf("mail.tm unavailable (%v), trying guerrillamail...\n", err)

	// Fallback to Guerrilla Mail
	gc, err2 := NewGuerrillaClient(ctx)
	if err2 == nil {
		return gc, nil
	}

	return nil, fmt.Errorf("all temp email providers failed: mail.tm: %v, guerrillamail: %v", err, err2)
}

// randomString generates a random lowercase string of given length.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.IntN(len(letters))]
	}
	return string(b)
}
