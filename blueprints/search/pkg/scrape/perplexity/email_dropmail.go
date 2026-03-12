package perplexity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const dropMailAPI = "https://dropmail.me/api/graphql/web-test-2"

// DropMailClient uses dropmail.me's GraphQL API for disposable emails.
type DropMailClient struct {
	httpClient *http.Client
	email      string
	sessionID  string
}

type dropMailGraphQLResp struct {
	Data json.RawMessage `json:"data"`
}

type dropMailSessionResp struct {
	IntroduceSession struct {
		ID        string `json:"id"`
		Addresses []struct {
			Address string `json:"address"`
		} `json:"addresses"`
	} `json:"introduceSession"`
}

type dropMailInboxResp struct {
	Session struct {
		Mails []dropMailMessage `json:"mails"`
	} `json:"session"`
}

type dropMailMessage struct {
	RawSize       int    `json:"rawSize"`
	HeaderSubject string `json:"headerSubject"`
	Text          string `json:"text"`
	HTML          string `json:"html"`
}

// NewDropMailClient creates a new dropmail.me client via GraphQL.
func NewDropMailClient(ctx context.Context) (*DropMailClient, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	query := `mutation { introduceSession { id, expiresAt, addresses { address } } }`
	body, _ := json.Marshal(map[string]string{"query": query})

	req, err := http.NewRequestWithContext(ctx, "POST", dropMailAPI, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("introduce session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("introduce HTTP %d", resp.StatusCode)
	}

	var gqlResp dropMailGraphQLResp
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	var sessResp dropMailSessionResp
	if err := json.Unmarshal(gqlResp.Data, &sessResp); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}

	if sessResp.IntroduceSession.ID == "" || len(sessResp.IntroduceSession.Addresses) == 0 {
		return nil, fmt.Errorf("empty session or addresses")
	}

	return &DropMailClient{
		httpClient: client,
		email:      sessResp.IntroduceSession.Addresses[0].Address,
		sessionID:  sessResp.IntroduceSession.ID,
	}, nil
}

func (c *DropMailClient) Email() string {
	return c.email
}

func (c *DropMailClient) WaitForMessage(ctx context.Context, matchSubject string, timeout time.Duration) (string, error) {
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
			if strings.Contains(msg.HeaderSubject, matchSubject) || msg.HeaderSubject == matchSubject {
				if msg.HTML != "" {
					return msg.HTML, nil
				}
				return msg.Text, nil
			}
		}

		time.Sleep(3 * time.Second)
	}

	return "", fmt.Errorf("timeout waiting for email with subject %q", matchSubject)
}

func (c *DropMailClient) checkInbox(ctx context.Context) ([]dropMailMessage, error) {
	query := fmt.Sprintf(`query { session(id: "%s") { mails { rawSize, headerSubject, text, html } } }`, c.sessionID)
	body, _ := json.Marshal(map[string]string{"query": query})

	req, err := http.NewRequestWithContext(ctx, "POST", dropMailAPI, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("inbox HTTP %d", resp.StatusCode)
	}

	var gqlResp dropMailGraphQLResp
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, err
	}

	var inboxResp dropMailInboxResp
	if err := json.Unmarshal(gqlResp.Data, &inboxResp); err != nil {
		return nil, err
	}

	return inboxResp.Session.Mails, nil
}
