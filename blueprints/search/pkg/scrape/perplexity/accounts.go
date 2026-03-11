package perplexity

import (
	"encoding/json"
	"fmt"
	"sync"
)

// AccountManager handles multi-account rotation with failure detection.
type AccountManager struct {
	db           *DB
	accountIdx   int // round-robin index for scraped accounts
	keyIdx       int // round-robin index for API keys
	mu           sync.Mutex
}

// NewAccountManager creates a new account manager.
func NewAccountManager(db *DB) *AccountManager {
	return &AccountManager{db: db}
}

// NextAccount returns the next healthy scraped account using round-robin.
// Skips exhausted/failed/banned accounts.
func (am *AccountManager) NextAccount() (*Account, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	accounts, err := am.db.ListAccounts(AccountActive)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no active accounts; register with 'search perplexity register'")
	}

	// Round-robin
	am.accountIdx = am.accountIdx % len(accounts)
	account := accounts[am.accountIdx]
	am.accountIdx++

	return &account, nil
}

// NextAPIKey returns the next healthy API key using round-robin.
func (am *AccountManager) NextAPIKey() (*APIKey, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	keys, err := am.db.ListAPIKeys()
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}

	// Filter active keys
	var active []APIKey
	for _, k := range keys {
		if k.Status == KeyActive {
			active = append(active, k)
		}
	}
	if len(active) == 0 {
		return nil, fmt.Errorf("no active API keys; add with 'search perplexity accounts add-key'")
	}

	am.keyIdx = am.keyIdx % len(active)
	key := active[am.keyIdx]
	am.keyIdx++

	return &key, nil
}

// AddAccount stores a new registered account with its session.
func (am *AccountManager) AddAccount(email string, sess *sessionData) error {
	sessionJSON, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	account := &Account{
		Email:       email,
		Source:      "emailnator",
		SessionData: string(sessionJSON),
		ProQueries:  sess.CopilotQueries,
		FileUploads: sess.FileUploads,
		Status:      AccountActive,
	}
	return am.db.SaveAccount(account)
}

// AddAPIKey stores a new API key.
func (am *AccountManager) AddAPIKey(key, name string) error {
	k := &APIKey{
		Key:    key,
		Name:   name,
		Status: KeyActive,
		Tier:   "tier0",
	}
	return am.db.SaveAPIKey(k)
}

// MarkAccountFailed marks an account as failed and records the error.
func (am *AccountManager) MarkAccountFailed(id int, errMsg string) error {
	return am.db.UpdateAccountStatus(id, AccountFailed, errMsg)
}

// MarkAccountExhausted marks an account as having 0 remaining pro queries.
func (am *AccountManager) MarkAccountExhausted(id int) error {
	return am.db.UpdateAccountStatus(id, AccountExhausted, "pro queries exhausted")
}

// MarkAccountBanned marks an account as banned.
func (am *AccountManager) MarkAccountBanned(id int, errMsg string) error {
	return am.db.UpdateAccountStatus(id, AccountBanned, errMsg)
}

// MarkKeyFailed marks an API key as failed.
func (am *AccountManager) MarkKeyFailed(id int, errMsg string) error {
	return am.db.UpdateAPIKeyStatus(id, KeyInvalid, errMsg)
}

// MarkKeyRateLimited marks an API key as rate limited.
func (am *AccountManager) MarkKeyRateLimited(id int, errMsg string) error {
	return am.db.UpdateAPIKeyStatus(id, KeyRateLimited, errMsg)
}

// RecordAccountUsage updates account usage after a successful query.
func (am *AccountManager) RecordAccountUsage(id int, proQueriesLeft int) error {
	if proQueriesLeft <= 0 {
		am.db.UpdateAccountUsage(id, 0)
		return am.MarkAccountExhausted(id)
	}
	return am.db.UpdateAccountUsage(id, proQueriesLeft)
}

// RecordKeyUsage updates API key usage after a successful query.
func (am *AccountManager) RecordKeyUsage(id, tokens int) error {
	return am.db.UpdateAPIKeyUsage(id, tokens)
}

// LoadAccountSession loads session data from an account and applies it to a client.
func (am *AccountManager) LoadAccountSession(client *Client, account *Account) error {
	sessionStr, err := am.db.GetAccountSession(account.ID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	if sessionStr == "" {
		return fmt.Errorf("account %d has no session data", account.ID)
	}

	var sess sessionData
	if err := json.Unmarshal([]byte(sessionStr), &sess); err != nil {
		return fmt.Errorf("parse session: %w", err)
	}

	// Apply session to client
	client.applySession(&sess)
	return nil
}

// applySession applies session data to a client.
func (c *Client) applySession(sess *sessionData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.copilotQueries = sess.CopilotQueries
	c.fileUploads = sess.FileUploads
	c.authenticated = true

	// Apply cookies
	if len(sess.Cookies) > 0 {
		c.loadSessionCookies(sess.Cookies)
	}
}
