package serp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const mailTMBase = "https://api.mail.tm"

type MailTMClient struct {
	hc *http.Client
}

func NewMailTMClient() *MailTMClient {
	return &MailTMClient{hc: &http.Client{Timeout: 15 * time.Second}}
}

type mailTMDomain struct {
	Domain string `json:"domain"`
}

// PickDomain returns the first available mail.tm domain.
func (c *MailTMClient) PickDomain() (string, error) {
	resp, err := c.hc.Get(mailTMBase + "/domains")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		HydraMember []mailTMDomain `json:"hydra:member"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.HydraMember) == 0 {
		return "", fmt.Errorf("no domains available from mail.tm")
	}
	return result.HydraMember[0].Domain, nil
}

// CreateAccount creates a mail.tm account and returns the token.
func (c *MailTMClient) CreateAccount(address, password string) (token string, err error) {
	body, _ := json.Marshal(map[string]string{"address": address, "password": password})
	resp, err := c.hc.Post(mailTMBase+"/accounts", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("mail.tm create account: status %d", resp.StatusCode)
	}
	return c.GetToken(address, password)
}

// GetToken authenticates and returns a bearer token.
func (c *MailTMClient) GetToken(address, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{"address": address, "password": password})
	resp, err := c.hc.Post(mailTMBase+"/token", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Token == "" {
		return "", fmt.Errorf("mail.tm: empty token in response")
	}
	return result.Token, nil
}

type MailTMMessage struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
}

// ListMessages returns inbox messages.
func (c *MailTMClient) ListMessages(token string) ([]MailTMMessage, error) {
	req, _ := http.NewRequest("GET", mailTMBase+"/messages", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		HydraMember []MailTMMessage `json:"hydra:member"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.HydraMember, nil
}

// GetMessageBody fetches the full text/html of a message.
func (c *MailTMClient) GetMessageBody(token, id string) (string, error) {
	req, _ := http.NewRequest("GET", mailTMBase+"/messages/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		HTML []string `json:"html"`
		Text string   `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.HTML) > 0 {
		return result.HTML[0], nil
	}
	return result.Text, nil
}

// PollForMessage polls inbox until a message arrives (or timeout).
func (c *MailTMClient) PollForMessage(token string, timeout time.Duration) (*MailTMMessage, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		msgs, err := c.ListMessages(token)
		if err == nil && len(msgs) > 0 {
			return &msgs[0], nil
		}
		time.Sleep(3 * time.Second)
	}
	return nil, fmt.Errorf("timed out waiting for email (%.0fs)", timeout.Seconds())
}
