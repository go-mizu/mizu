package perplexity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- DB Tests ---

func tempDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := OpenDB(filepath.Join(dir, "test.duckdb"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenDB(t *testing.T) {
	db := tempDB(t)
	count, err := db.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 searches, got %d", count)
	}
}

func TestSaveAndCountSearches(t *testing.T) {
	db := tempDB(t)

	r := &SearchResult{
		Query:      "test query",
		Answer:     "test answer",
		Mode:       ModeAuto,
		Model:      "turbo",
		Source:     "sse",
		SearchedAt: time.Now(),
		Citations: []Citation{
			{URL: "https://example.com", Title: "Example", Domain: "example.com"},
		},
		WebResults: []WebResult{
			{Name: "Example", URL: "https://example.com", Snippet: "A snippet"},
		},
	}

	if err := db.SaveSearch(r); err != nil {
		t.Fatalf("save search: %v", err)
	}

	count, err := db.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 search, got %d", count)
	}
}

func TestRecentSearches(t *testing.T) {
	db := tempDB(t)

	for i := 0; i < 3; i++ {
		r := &SearchResult{
			Query:      fmt.Sprintf("query %d", i),
			Answer:     fmt.Sprintf("answer %d", i),
			Source:     "sse",
			SearchedAt: time.Now().Add(time.Duration(i) * time.Second),
		}
		if err := db.SaveSearch(r); err != nil {
			t.Fatalf("save search %d: %v", i, err)
		}
	}

	results, err := db.RecentSearches(2)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Query != "query 2" {
		t.Errorf("expected most recent first, got %q", results[0].Query)
	}
}

func TestSearchWithAccountAndAPIKey(t *testing.T) {
	db := tempDB(t)

	r := &SearchResult{
		Query:      "api test",
		Answer:     "api answer",
		Source:     "api",
		SearchedAt: time.Now(),
		AccountID:  42,
		APIKeyID:   7,
		TokensUsed: 150,
		DurationMs: 1234,
	}

	if err := db.SaveSearch(r); err != nil {
		t.Fatalf("save: %v", err)
	}

	count, err := db.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

// --- Account Tests ---

func TestAccountCRUD(t *testing.T) {
	db := tempDB(t)

	// Create
	a := &Account{
		Email:       "test@example.com",
		Source:      "emailnator",
		SessionData: `{"cookies":[]}`,
		ProQueries:  5,
		FileUploads: 10,
		Status:      AccountActive,
	}
	if err := db.SaveAccount(a); err != nil {
		t.Fatalf("save account: %v", err)
	}
	if a.ID == 0 {
		t.Error("expected non-zero ID")
	}

	// List all
	accounts, err := db.ListAccounts("")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}
	if accounts[0].Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", accounts[0].Email)
	}

	// List by status
	accounts, err = db.ListAccounts(AccountActive)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(accounts) != 1 {
		t.Errorf("expected 1 active, got %d", len(accounts))
	}

	// Get session
	sess, err := db.GetAccountSession(a.ID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sess != `{"cookies":[]}` {
		t.Errorf("unexpected session: %s", sess)
	}

	// Update status
	if err := db.UpdateAccountStatus(a.ID, AccountFailed, "cloudflare block"); err != nil {
		t.Fatalf("update status: %v", err)
	}

	accounts, err = db.ListAccounts(AccountActive)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(accounts) != 0 {
		t.Errorf("expected 0 active after fail, got %d", len(accounts))
	}

	// Update usage
	if err := db.UpdateAccountUsage(a.ID, 3); err != nil {
		t.Fatalf("update usage: %v", err)
	}

	// Count
	total, active, err := db.CountAccounts()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 total, got %d", total)
	}
	if active != 0 {
		t.Errorf("expected 0 active, got %d", active)
	}

	// Delete
	if err := db.DeleteAccount(a.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	total, _, err = db.CountAccounts()
	if err != nil {
		t.Fatalf("count after delete: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 after delete, got %d", total)
	}
}

// --- API Key Tests ---

func TestAPIKeyCRUD(t *testing.T) {
	db := tempDB(t)

	k := &APIKey{
		Key:    "pplx-test-key-123",
		Name:   "test-key",
		Status: KeyActive,
		Tier:   "tier0",
	}
	if err := db.SaveAPIKey(k); err != nil {
		t.Fatalf("save key: %v", err)
	}
	if k.ID == 0 {
		t.Error("expected non-zero ID")
	}

	// List
	keys, err := db.ListAPIKeys()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].Key != "pplx-test-key-123" {
		t.Errorf("unexpected key: %s", keys[0].Key)
	}
	if keys[0].Name != "test-key" {
		t.Errorf("unexpected name: %s", keys[0].Name)
	}

	// Update status
	if err := db.UpdateAPIKeyStatus(k.ID, KeyRateLimited, "429 too many requests"); err != nil {
		t.Fatalf("update status: %v", err)
	}

	// Update usage
	if err := db.UpdateAPIKeyUsage(k.ID, 500); err != nil {
		t.Fatalf("update usage: %v", err)
	}

	keys, err = db.ListAPIKeys()
	if err != nil {
		t.Fatalf("list after update: %v", err)
	}
	if keys[0].Status != KeyRateLimited {
		t.Errorf("expected rate_limited, got %s", keys[0].Status)
	}
	if keys[0].TotalTokens != 500 {
		t.Errorf("expected 500 tokens, got %d", keys[0].TotalTokens)
	}
	if keys[0].UseCount != 1 {
		t.Errorf("expected use_count=1, got %d", keys[0].UseCount)
	}

	// Count
	total, active, err := db.CountAPIKeys()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 total, got %d", total)
	}
	if active != 0 {
		t.Errorf("expected 0 active (rate limited), got %d", active)
	}

	// Delete
	if err := db.DeleteAPIKey(k.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	total, _, err = db.CountAPIKeys()
	if err != nil {
		t.Fatalf("count after delete: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 after delete, got %d", total)
	}
}

// --- Error Log Tests ---

func TestErrorLogging(t *testing.T) {
	db := tempDB(t)

	e := &ErrorLog{
		Source:     "sse",
		Operation:  "search",
		Query:     "test query",
		ErrorType:  ErrHTTP,
		ErrorMsg:  "HTTP 403 Forbidden",
		HTTPStatus: 403,
	}
	if err := db.LogError(e); err != nil {
		t.Fatalf("log error: %v", err)
	}

	e2 := &ErrorLog{
		Source:     "api",
		Operation:  "api_chat",
		Query:     "api query",
		ErrorType:  ErrRateLimit,
		ErrorMsg:  "429 Too Many Requests",
		HTTPStatus: 429,
		APIKeyID:   1,
	}
	if err := db.LogError(e2); err != nil {
		t.Fatalf("log error 2: %v", err)
	}

	// Count
	count, err := db.CountErrors()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 errors, got %d", count)
	}

	// Recent errors (all)
	errors, err := db.RecentErrors(10, "", "")
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(errors) != 2 {
		t.Fatalf("expected 2, got %d", len(errors))
	}

	// Filter by source
	errors, err = db.RecentErrors(10, "sse", "")
	if err != nil {
		t.Fatalf("filter source: %v", err)
	}
	if len(errors) != 1 {
		t.Errorf("expected 1 sse error, got %d", len(errors))
	}

	// Filter by type
	errors, err = db.RecentErrors(10, "", ErrRateLimit)
	if err != nil {
		t.Fatalf("filter type: %v", err)
	}
	if len(errors) != 1 {
		t.Errorf("expected 1 rate_limit error, got %d", len(errors))
	}
}

func TestErrorBodyTruncation(t *testing.T) {
	db := tempDB(t)

	// Create a very long response body
	longBody := make([]byte, maxErrorBodyLen+1000)
	for i := range longBody {
		longBody[i] = 'x'
	}

	e := &ErrorLog{
		Source:       "api",
		Operation:    "api_chat",
		ErrorType:    ErrHTTP,
		ErrorMsg:     "error",
		ResponseBody: string(longBody),
	}
	if err := db.LogError(e); err != nil {
		t.Fatalf("log: %v", err)
	}

	errors, err := db.RecentErrors(1, "", "")
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(errors) != 1 {
		t.Fatalf("expected 1, got %d", len(errors))
	}
	if len(errors[0].ResponseBody) > maxErrorBodyLen {
		t.Errorf("expected body truncated to %d, got %d", maxErrorBodyLen, len(errors[0].ResponseBody))
	}
}

// --- AccountManager Tests ---

func TestAccountManagerRoundRobin(t *testing.T) {
	db := tempDB(t)

	// Add 3 accounts
	for i := 0; i < 3; i++ {
		a := &Account{
			Email:       fmt.Sprintf("user%d@test.com", i),
			Source:      "emailnator",
			SessionData: "{}",
			ProQueries:  5,
			FileUploads: 10,
			Status:      AccountActive,
		}
		if err := db.SaveAccount(a); err != nil {
			t.Fatalf("save account %d: %v", i, err)
		}
	}

	am := NewAccountManager(db)

	// Verify round-robin
	emails := make([]string, 3)
	for i := 0; i < 3; i++ {
		a, err := am.NextAccount()
		if err != nil {
			t.Fatalf("next account %d: %v", i, err)
		}
		emails[i] = a.Email
	}

	if emails[0] != "user0@test.com" || emails[1] != "user1@test.com" || emails[2] != "user2@test.com" {
		t.Errorf("unexpected round-robin order: %v", emails)
	}

	// Should wrap around
	a, err := am.NextAccount()
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	if a.Email != "user0@test.com" {
		t.Errorf("expected wrap to user0, got %s", a.Email)
	}
}

func TestAccountManagerSkipsInactive(t *testing.T) {
	db := tempDB(t)

	// Add 2 accounts, one failed
	a1 := &Account{Email: "active@test.com", Source: "emailnator", SessionData: "{}", ProQueries: 5, Status: AccountActive}
	a2 := &Account{Email: "failed@test.com", Source: "emailnator", SessionData: "{}", ProQueries: 0, Status: AccountFailed}
	db.SaveAccount(a1)
	db.SaveAccount(a2)

	am := NewAccountManager(db)
	acc, err := am.NextAccount()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	// NextAccount only returns active accounts
	if acc.Email != "active@test.com" {
		t.Errorf("expected active account, got %s", acc.Email)
	}
}

func TestAccountManagerNoAccounts(t *testing.T) {
	db := tempDB(t)
	am := NewAccountManager(db)

	_, err := am.NextAccount()
	if err == nil {
		t.Error("expected error for no accounts")
	}
}

func TestAccountManagerAPIKeyRotation(t *testing.T) {
	db := tempDB(t)

	// Add 2 API keys
	k1 := &APIKey{Key: "pplx-key1", Name: "key1", Status: KeyActive, Tier: "tier0"}
	k2 := &APIKey{Key: "pplx-key2", Name: "key2", Status: KeyActive, Tier: "tier0"}
	db.SaveAPIKey(k1)
	db.SaveAPIKey(k2)

	am := NewAccountManager(db)

	key1, _ := am.NextAPIKey()
	key2, _ := am.NextAPIKey()
	key3, _ := am.NextAPIKey()

	if key1.Key != "pplx-key1" {
		t.Errorf("expected key1 first, got %s", key1.Key)
	}
	if key2.Key != "pplx-key2" {
		t.Errorf("expected key2 second, got %s", key2.Key)
	}
	if key3.Key != "pplx-key1" {
		t.Errorf("expected key1 again (wrap), got %s", key3.Key)
	}
}

func TestAccountManagerSkipsInactiveKeys(t *testing.T) {
	db := tempDB(t)

	k1 := &APIKey{Key: "pplx-active", Name: "active", Status: KeyActive, Tier: "tier0"}
	k2 := &APIKey{Key: "pplx-invalid", Name: "invalid", Status: KeyInvalid, Tier: "tier0"}
	db.SaveAPIKey(k1)
	db.SaveAPIKey(k2)

	am := NewAccountManager(db)
	key, err := am.NextAPIKey()
	if err != nil {
		t.Fatalf("next key: %v", err)
	}
	if key.Key != "pplx-active" {
		t.Errorf("expected active key, got %s", key.Key)
	}
}

func TestAccountManagerMarkFailed(t *testing.T) {
	db := tempDB(t)

	a := &Account{Email: "fail@test.com", Source: "emailnator", SessionData: "{}", ProQueries: 5, Status: AccountActive}
	db.SaveAccount(a)

	am := NewAccountManager(db)
	if err := am.MarkAccountFailed(a.ID, "cloudflare block"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	accounts, _ := db.ListAccounts(AccountActive)
	if len(accounts) != 0 {
		t.Errorf("expected 0 active after mark failed, got %d", len(accounts))
	}
}

func TestAccountManagerRecordUsage(t *testing.T) {
	db := tempDB(t)

	a := &Account{Email: "use@test.com", Source: "emailnator", SessionData: "{}", ProQueries: 5, Status: AccountActive}
	db.SaveAccount(a)

	am := NewAccountManager(db)

	// Record usage with remaining queries
	if err := am.RecordAccountUsage(a.ID, 4); err != nil {
		t.Fatalf("record: %v", err)
	}

	accounts, _ := db.ListAccounts(AccountActive)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 active, got %d", len(accounts))
	}
	if accounts[0].ProQueries != 4 {
		t.Errorf("expected 4 pro queries, got %d", accounts[0].ProQueries)
	}

	// Exhaust queries
	if err := am.RecordAccountUsage(a.ID, 0); err != nil {
		t.Fatalf("exhaust: %v", err)
	}

	accounts, _ = db.ListAccounts(AccountActive)
	if len(accounts) != 0 {
		t.Errorf("expected 0 active after exhaust, got %d", len(accounts))
	}
}

// --- API Client Tests ---

func TestAPIClientToSearchResult(t *testing.T) {
	client := NewAPIClient("test-key", 1)

	resp := &ChatResponse{
		ID:    "test-id",
		Model: "sonar",
		Choices: []ChatChoice{{
			Message: &ChatMessage{
				Role:    "assistant",
				Content: "Test answer",
			},
		}},
		Citations: []string{"https://example.com", "https://test.org"},
		SearchResults: []APISearchResult{
			{Title: "Example", URL: "https://example.com", Snippet: "A snippet"},
			{Title: "Test", URL: "https://new.com", Snippet: "New site"},
		},
		Usage: &ChatUsage{TotalTokens: 100},
	}

	result := client.ToSearchResult(resp, "test query", 500)

	if result.Query != "test query" {
		t.Errorf("unexpected query: %s", result.Query)
	}
	if result.Answer != "Test answer" {
		t.Errorf("unexpected answer: %s", result.Answer)
	}
	if result.Source != "api" {
		t.Errorf("unexpected source: %s", result.Source)
	}
	if result.Model != "sonar" {
		t.Errorf("unexpected model: %s", result.Model)
	}
	if result.TokensUsed != 100 {
		t.Errorf("unexpected tokens: %d", result.TokensUsed)
	}
	if result.DurationMs != 500 {
		t.Errorf("unexpected duration: %d", result.DurationMs)
	}
	if result.APIKeyID != 1 {
		t.Errorf("unexpected key ID: %d", result.APIKeyID)
	}

	// Citations should include both explicit citations and search results
	if len(result.Citations) < 2 {
		t.Errorf("expected at least 2 citations, got %d", len(result.Citations))
	}

	// Web results
	if len(result.WebResults) != 2 {
		t.Errorf("expected 2 web results, got %d", len(result.WebResults))
	}
}

func TestAPIError(t *testing.T) {
	e := &APIError{StatusCode: 429, Body: "rate limited"}
	if !e.IsRateLimit() {
		t.Error("expected rate limit")
	}
	if e.IsAuth() {
		t.Error("unexpected auth error")
	}

	e2 := &APIError{StatusCode: 401, Body: "unauthorized"}
	if e2.IsRateLimit() {
		t.Error("unexpected rate limit")
	}
	if !e2.IsAuth() {
		t.Error("expected auth error")
	}

	e3 := &APIError{StatusCode: 403, Body: "forbidden"}
	if !e3.IsAuth() {
		t.Error("expected auth error for 403")
	}

	// Test Error() formatting
	e4 := &APIError{StatusCode: 500, Body: "internal server error"}
	msg := e4.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}

func TestTruncate(t *testing.T) {
	if truncate("short", 10) != "short" {
		t.Error("short string should not be truncated")
	}
	if truncate("a long string here", 6) != "a long..." {
		t.Errorf("unexpected truncation: %s", truncate("a long string here", 6))
	}
	if truncate("", 5) != "" {
		t.Error("empty string should stay empty")
	}
}

// --- Config Tests ---

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Timeout != defaultTimeout {
		t.Errorf("unexpected timeout: %v", cfg.Timeout)
	}
	if cfg.Language != "en-US" {
		t.Errorf("unexpected language: %s", cfg.Language)
	}

	home, _ := os.UserHomeDir()
	expectedDir := filepath.Join(home, "data", "perplexity")
	if cfg.DataDir != expectedDir {
		t.Errorf("unexpected data dir: %s", cfg.DataDir)
	}
}

func TestConfigPaths(t *testing.T) {
	cfg := Config{DataDir: "/tmp/test-perplexity"}
	if cfg.DBPath() != "/tmp/test-perplexity/perplexity.duckdb" {
		t.Errorf("unexpected db path: %s", cfg.DBPath())
	}
	if cfg.SessionPath() != "/tmp/test-perplexity/.session.json" {
		t.Errorf("unexpected session path: %s", cfg.SessionPath())
	}
}

// --- Types Tests ---

func TestDefaultSearchOptions(t *testing.T) {
	opts := DefaultSearchOptions()
	if opts.Mode != ModeAuto {
		t.Errorf("unexpected mode: %s", opts.Mode)
	}
	if len(opts.Sources) != 1 || opts.Sources[0] != SourceWeb {
		t.Errorf("unexpected sources: %v", opts.Sources)
	}
	if opts.Language != "en-US" {
		t.Errorf("unexpected language: %s", opts.Language)
	}
}

func TestNilIfZero(t *testing.T) {
	if nilIfZero(0) != nil {
		t.Error("expected nil for 0")
	}
	if nilIfZero(42) != 42 {
		t.Error("expected 42 for 42")
	}
}

// --- Client Tests ---

func TestNewClient(t *testing.T) {
	cfg := Config{
		DataDir: t.TempDir(),
		Timeout: 10 * time.Second,
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if client.IsAuthenticated() {
		t.Error("new client should not be authenticated")
	}
	if client.CopilotQueries() != 0 {
		t.Errorf("expected 0 copilot queries, got %d", client.CopilotQueries())
	}
}

func TestClientApplySession(t *testing.T) {
	cfg := Config{DataDir: t.TempDir(), Timeout: 10 * time.Second}
	client, _ := NewClient(cfg)

	sess := &sessionData{
		CopilotQueries: 5,
		FileUploads:    10,
		Cookies: []*cookieData{
			{Name: "test", Value: "value", Domain: ".perplexity.ai", Path: "/"},
		},
	}

	client.applySession(sess)

	if !client.IsAuthenticated() {
		t.Error("expected authenticated after apply session")
	}
	if client.CopilotQueries() != 5 {
		t.Errorf("expected 5 copilot queries, got %d", client.CopilotQueries())
	}
}

func TestClientSaveLoadSession(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{DataDir: dir, Timeout: 10 * time.Second}
	client, _ := NewClient(cfg)

	// Manually set some state
	client.copilotQueries = 3
	client.fileUploads = 8
	client.authenticated = true

	if err := client.SaveSession(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(cfg.SessionPath()); err != nil {
		t.Fatalf("session file not found: %v", err)
	}

	// Load into new client
	client2, _ := NewClient(cfg)
	if err := client2.LoadSession(); err != nil {
		t.Fatalf("load: %v", err)
	}
	if !client2.IsAuthenticated() {
		t.Error("expected authenticated after load")
	}
	if client2.CopilotQueries() != 3 {
		t.Errorf("expected 3, got %d", client2.CopilotQueries())
	}
}

func TestExportSession(t *testing.T) {
	cfg := Config{DataDir: t.TempDir(), Timeout: 10 * time.Second}
	client, _ := NewClient(cfg)
	client.copilotQueries = 5
	client.fileUploads = 10

	sess := client.exportSession()
	if sess.CopilotQueries != 5 {
		t.Errorf("expected 5, got %d", sess.CopilotQueries)
	}
	if sess.FileUploads != 10 {
		t.Errorf("expected 10, got %d", sess.FileUploads)
	}
}

// --- Integration: AccountManager + Client ---

func TestAccountManagerLoadSession(t *testing.T) {
	db := tempDB(t)

	// Create an account with session data
	sess := &sessionData{
		CopilotQueries: 4,
		FileUploads:    8,
		Cookies:        []*cookieData{{Name: "sid", Value: "abc", Domain: ".perplexity.ai", Path: "/"}},
	}
	sessJSON, _ := json.Marshal(sess)

	a := &Account{
		Email:       "loadtest@test.com",
		Source:      "emailnator",
		SessionData: string(sessJSON),
		ProQueries:  4,
		FileUploads: 8,
		Status:      AccountActive,
	}
	db.SaveAccount(a)

	am := NewAccountManager(db)

	cfg := Config{DataDir: t.TempDir(), Timeout: 10 * time.Second}
	client, _ := NewClient(cfg)

	if err := am.LoadAccountSession(client, a); err != nil {
		t.Fatalf("load session: %v", err)
	}

	if !client.IsAuthenticated() {
		t.Error("expected authenticated")
	}
	if client.CopilotQueries() != 4 {
		t.Errorf("expected 4, got %d", client.CopilotQueries())
	}
}

// --- extractDomain Test ---

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url    string
		domain string
	}{
		{"https://www.example.com/path", "www.example.com"},
		{"https://test.org", "test.org"},
		{"invalid-url", ""},
	}

	for _, tt := range tests {
		got := extractDomain(tt.url)
		if got != tt.domain {
			t.Errorf("extractDomain(%q) = %q, want %q", tt.url, got, tt.domain)
		}
	}
}

// --- Chat Request JSON marshaling ---

func TestChatRequestJSON(t *testing.T) {
	req := ChatRequest{
		Model: APISonar,
		Messages: []ChatMessage{
			{Role: "system", Content: "Be helpful"},
			{Role: "user", Content: "What is Go?"},
		},
		MaxTokens:           1024,
		ReturnImages:        true,
		ReturnRelated:       true,
		SearchRecencyFilter: "week",
		WebSearchOptions:    &WebSearchOptions{SearchContextSize: "high"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	json.Unmarshal(data, &decoded)

	if decoded["model"] != "sonar" {
		t.Errorf("unexpected model: %v", decoded["model"])
	}
	if decoded["return_images"] != true {
		t.Errorf("expected return_images=true")
	}
	if decoded["search_recency_filter"] != "week" {
		t.Errorf("unexpected recency: %v", decoded["search_recency_filter"])
	}

	msgs := decoded["messages"].([]any)
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}

	wsOpts := decoded["web_search_options"].(map[string]any)
	if wsOpts["search_context_size"] != "high" {
		t.Errorf("unexpected context size: %v", wsOpts["search_context_size"])
	}
}

// --- ChatResponse JSON roundtrip ---

func TestChatResponseJSON(t *testing.T) {
	raw := `{
		"id": "resp-123",
		"model": "sonar-pro",
		"created": 1700000000,
		"choices": [{"index": 0, "finish_reason": "stop", "message": {"role": "assistant", "content": "Go is great"}}],
		"citations": ["https://go.dev", "https://golang.org"],
		"search_results": [{"title": "Go Dev", "url": "https://go.dev", "snippet": "The Go programming language"}],
		"usage": {"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30}
	}`

	var resp ChatResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.ID != "resp-123" {
		t.Errorf("unexpected id: %s", resp.ID)
	}
	if resp.Model != "sonar-pro" {
		t.Errorf("unexpected model: %s", resp.Model)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Go is great" {
		t.Errorf("unexpected content: %s", resp.Choices[0].Message.Content)
	}
	if len(resp.Citations) != 2 {
		t.Errorf("expected 2 citations, got %d", len(resp.Citations))
	}
	if len(resp.SearchResults) != 1 {
		t.Errorf("expected 1 search result, got %d", len(resp.SearchResults))
	}
	if resp.Usage == nil {
		t.Fatal("expected usage")
	}
	if resp.Usage.TotalTokens != 30 {
		t.Errorf("expected 30 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

// --- Multiple accounts registration + DB integration ---

func TestMultipleAccountsDB(t *testing.T) {
	db := tempDB(t)
	am := NewAccountManager(db)

	// Add multiple accounts
	for i := 0; i < 5; i++ {
		sess := &sessionData{
			CopilotQueries: 5,
			FileUploads:    10,
		}
		if err := am.AddAccount(fmt.Sprintf("user%d@test.com", i), sess); err != nil {
			t.Fatalf("add account %d: %v", i, err)
		}
	}

	total, active, err := db.CountAccounts()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 5 {
		t.Errorf("expected 5 total, got %d", total)
	}
	if active != 5 {
		t.Errorf("expected 5 active, got %d", active)
	}

	// Exhaust first account
	accounts, _ := db.ListAccounts(AccountActive)
	am.RecordAccountUsage(accounts[0].ID, 0)

	total, active, err = db.CountAccounts()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 5 {
		t.Errorf("expected 5 total, got %d", total)
	}
	if active != 4 {
		t.Errorf("expected 4 active, got %d", active)
	}

	// Ban second account
	am.MarkAccountBanned(accounts[1].ID, "IP banned")

	total, active, err = db.CountAccounts()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if active != 3 {
		t.Errorf("expected 3 active, got %d", active)
	}
}

// --- Multiple API keys + rotation ---

func TestMultipleAPIKeysDB(t *testing.T) {
	db := tempDB(t)
	am := NewAccountManager(db)

	for i := 0; i < 3; i++ {
		if err := am.AddAPIKey(fmt.Sprintf("pplx-key%d", i), fmt.Sprintf("key%d", i)); err != nil {
			t.Fatalf("add key %d: %v", i, err)
		}
	}

	total, active, _ := db.CountAPIKeys()
	if total != 3 || active != 3 {
		t.Errorf("expected 3/3, got %d/%d", total, active)
	}

	// Record usage on all keys
	keys, _ := db.ListAPIKeys()
	for _, k := range keys {
		am.RecordKeyUsage(k.ID, 100)
	}

	// Mark one rate limited
	am.MarkKeyRateLimited(keys[0].ID, "429")
	total, active, _ = db.CountAPIKeys()
	if active != 2 {
		t.Errorf("expected 2 active, got %d", active)
	}

	// NextAPIKey should skip rate limited key
	key, _ := am.NextAPIKey()
	if key.Key == "pplx-key0" {
		t.Error("should skip rate limited key")
	}
}

// --- Schema Migration Test ---

func TestSchemaMigration(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "migrate.duckdb")

	// Open first time (creates tables)
	db1, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open 1: %v", err)
	}
	db1.Close()

	// Open second time (should not fail - migration idempotent)
	db2, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open 2 (reopen): %v", err)
	}
	defer db2.Close()

	// Verify tables exist
	count, err := db2.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	total, _, err := db2.CountAccounts()
	if err != nil {
		t.Fatalf("count accounts: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 accounts, got %d", total)
	}
}

// --- Context cancellation ---

func TestDBOperationsWithCancelledContext(t *testing.T) {
	db := tempDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel
	_ = ctx  // DB operations don't use context directly

	// DB operations should still work (they use database/sql which handles contexts internally)
	a := &Account{Email: "ctx@test.com", Source: "emailnator", SessionData: "{}", ProQueries: 5, Status: AccountActive}
	if err := db.SaveAccount(a); err != nil {
		t.Fatalf("save: %v", err)
	}
}

// --- Full workflow: register flow simulation ---

func TestFullWorkflowSimulation(t *testing.T) {
	db := tempDB(t)
	am := NewAccountManager(db)

	// Step 1: Register multiple accounts
	for i := 0; i < 3; i++ {
		sess := &sessionData{
			CopilotQueries: defaultCopilotQueries,
			FileUploads:    defaultFileUploads,
			Cookies: []*cookieData{
				{Name: "session_id", Value: fmt.Sprintf("sess_%d", i), Domain: ".perplexity.ai", Path: "/"},
			},
		}
		if err := am.AddAccount(fmt.Sprintf("user%d@emailnator.com", i), sess); err != nil {
			t.Fatalf("register %d: %v", i, err)
		}
	}

	// Step 2: Add API keys
	if err := am.AddAPIKey("pplx-key-prod", "production"); err != nil {
		t.Fatalf("add key: %v", err)
	}
	if err := am.AddAPIKey("pplx-key-dev", "development"); err != nil {
		t.Fatalf("add key: %v", err)
	}

	// Step 3: Rotate through accounts for scrape queries
	cfg := Config{DataDir: t.TempDir(), Timeout: 10 * time.Second}
	for i := 0; i < 5; i++ {
		account, err := am.NextAccount()
		if err != nil {
			t.Fatalf("next account: %v", err)
		}

		client, _ := NewClient(cfg)
		if err := am.LoadAccountSession(client, account); err != nil {
			t.Fatalf("load session: %v", err)
		}

		if !client.IsAuthenticated() {
			t.Error("expected authenticated")
		}

		// Simulate query usage
		am.RecordAccountUsage(account.ID, defaultCopilotQueries-1)
	}

	// Step 4: Use API keys for API queries
	for i := 0; i < 4; i++ {
		key, err := am.NextAPIKey()
		if err != nil {
			t.Fatalf("next key: %v", err)
		}

		apiClient := NewAPIClient(key.Key, key.ID)
		resp := &ChatResponse{
			Model: APISonar,
			Choices: []ChatChoice{{
				Message: &ChatMessage{Role: "assistant", Content: fmt.Sprintf("Answer %d", i)},
			}},
			Usage: &ChatUsage{TotalTokens: 50 + i*10},
		}

		// Convert and save result
		result := apiClient.ToSearchResult(resp, fmt.Sprintf("query %d", i), int64(100+i*50))
		if err := db.SaveSearch(result); err != nil {
			t.Fatalf("save search: %v", err)
		}

		am.RecordKeyUsage(key.ID, resp.Usage.TotalTokens)
	}

	// Step 5: Simulate failures
	accounts, _ := db.ListAccounts(AccountActive)
	if len(accounts) > 0 {
		am.MarkAccountExhausted(accounts[0].ID)
	}

	keys, _ := db.ListAPIKeys()
	if len(keys) > 0 {
		am.MarkKeyRateLimited(keys[0].ID, "HTTP 429")
	}

	// Step 6: Log errors
	db.LogError(&ErrorLog{
		Source:     "sse",
		Operation:  "search",
		Query:     "test query",
		ErrorType:  ErrCloudflare,
		ErrorMsg:  "403 Cloudflare challenge",
		HTTPStatus: 403,
	})
	db.LogError(&ErrorLog{
		APIKeyID:   keys[0].ID,
		Source:     "api",
		Operation:  "api_chat",
		Query:     "api query",
		ErrorType:  ErrRateLimit,
		ErrorMsg:  "429 Too Many Requests",
		HTTPStatus: 429,
	})

	// Step 7: Verify final state
	searchCount, _ := db.Count()
	if searchCount != 4 {
		t.Errorf("expected 4 searches, got %d", searchCount)
	}

	totalAccounts, activeAccounts, _ := db.CountAccounts()
	if totalAccounts != 3 {
		t.Errorf("expected 3 total accounts, got %d", totalAccounts)
	}
	if activeAccounts != 2 {
		t.Errorf("expected 2 active accounts, got %d", activeAccounts)
	}

	totalKeys, activeKeys, _ := db.CountAPIKeys()
	if totalKeys != 2 {
		t.Errorf("expected 2 total keys, got %d", totalKeys)
	}
	if activeKeys != 1 {
		t.Errorf("expected 1 active key, got %d", activeKeys)
	}

	errorCount, _ := db.CountErrors()
	if errorCount != 2 {
		t.Errorf("expected 2 errors, got %d", errorCount)
	}

	// Filtered errors
	sseErrors, _ := db.RecentErrors(10, "sse", "")
	if len(sseErrors) != 1 {
		t.Errorf("expected 1 sse error, got %d", len(sseErrors))
	}

	rateLimitErrors, _ := db.RecentErrors(10, "", ErrRateLimit)
	if len(rateLimitErrors) != 1 {
		t.Errorf("expected 1 rate_limit error, got %d", len(rateLimitErrors))
	}
}

// --- Thread Tests ---

func TestThreadCRUD(t *testing.T) {
	db := tempDB(t)

	// Create thread
	thread := &Thread{
		Title:  "Test conversation",
		Mode:   ModeAuto,
		Model:  "",
		Source: "sse",
	}
	if err := db.CreateThread(thread); err != nil {
		t.Fatalf("create thread: %v", err)
	}
	if thread.ID == 0 {
		t.Error("expected non-zero thread ID")
	}

	// Get thread
	got, err := db.GetThread(thread.ID)
	if err != nil {
		t.Fatalf("get thread: %v", err)
	}
	if got.Title != "Test conversation" {
		t.Errorf("expected title 'Test conversation', got %q", got.Title)
	}
	if got.MessageCount != 0 {
		t.Errorf("expected 0 messages, got %d", got.MessageCount)
	}

	// List threads
	threads, err := db.ListThreads(10)
	if err != nil {
		t.Fatalf("list threads: %v", err)
	}
	if len(threads) != 1 {
		t.Fatalf("expected 1 thread, got %d", len(threads))
	}

	// Count
	count, err := db.CountThreads()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	// Delete
	if err := db.DeleteThread(thread.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	count, _ = db.CountThreads()
	if count != 0 {
		t.Errorf("expected 0 after delete, got %d", count)
	}
}

func TestThreadMessages(t *testing.T) {
	db := tempDB(t)

	thread := &Thread{Title: "Message test", Mode: ModeAuto, Source: "sse"}
	db.CreateThread(thread)

	// Add user message
	userMsg := &ThreadMessage{
		ThreadID: thread.ID,
		Role:     "user",
		Content:  "What is Go?",
	}
	if err := db.AddThreadMessage(userMsg); err != nil {
		t.Fatalf("add user msg: %v", err)
	}
	if userMsg.ID == 0 {
		t.Error("expected non-zero message ID")
	}

	// Add assistant message
	assistantMsg := &ThreadMessage{
		ThreadID:    thread.ID,
		Role:        "assistant",
		Content:     "Go is a programming language...",
		BackendUUID: "uuid-123",
		Citations:   []Citation{{URL: "https://go.dev", Title: "Go Dev"}},
		WebResults:  []WebResult{{Name: "Go", URL: "https://go.dev", Snippet: "Go programming"}},
		RelatedQ:    []string{"What is goroutine?", "Go vs Rust"},
		DurationMs:  500,
	}
	if err := db.AddThreadMessage(assistantMsg); err != nil {
		t.Fatalf("add assistant msg: %v", err)
	}

	// Verify thread message count updated
	got, _ := db.GetThread(thread.ID)
	if got.MessageCount != 2 {
		t.Errorf("expected 2 messages, got %d", got.MessageCount)
	}

	// Get messages
	messages, err := db.GetThreadMessages(thread.ID)
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Role != "user" {
		t.Errorf("expected user first, got %s", messages[0].Role)
	}
	if messages[1].Role != "assistant" {
		t.Errorf("expected assistant second, got %s", messages[1].Role)
	}
	if messages[1].BackendUUID != "uuid-123" {
		t.Errorf("expected uuid-123, got %s", messages[1].BackendUUID)
	}
	if len(messages[1].Citations) != 1 {
		t.Errorf("expected 1 citation, got %d", len(messages[1].Citations))
	}
	if len(messages[1].RelatedQ) != 2 {
		t.Errorf("expected 2 related, got %d", len(messages[1].RelatedQ))
	}

	// Get last backend UUID
	uuid, err := db.GetLastBackendUUID(thread.ID)
	if err != nil {
		t.Fatalf("get last uuid: %v", err)
	}
	if uuid != "uuid-123" {
		t.Errorf("expected uuid-123, got %s", uuid)
	}
}

func TestThreadDeleteCascade(t *testing.T) {
	db := tempDB(t)

	thread := &Thread{Title: "Delete test", Mode: ModeAuto, Source: "sse"}
	db.CreateThread(thread)

	// Add messages
	for i := 0; i < 3; i++ {
		db.AddThreadMessage(&ThreadMessage{
			ThreadID: thread.ID,
			Role:     "user",
			Content:  fmt.Sprintf("message %d", i),
		})
	}

	// Verify messages exist
	msgs, _ := db.GetThreadMessages(thread.ID)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}

	// Delete thread should cascade to messages
	if err := db.DeleteThread(thread.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	msgs, _ = db.GetThreadMessages(thread.ID)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after delete, got %d", len(msgs))
	}
}

func TestGetLastBackendUUIDEmpty(t *testing.T) {
	db := tempDB(t)

	thread := &Thread{Title: "Empty UUID test", Mode: ModeAuto, Source: "sse"}
	db.CreateThread(thread)

	// No messages — should return empty
	uuid, err := db.GetLastBackendUUID(thread.ID)
	if err == nil && uuid != "" {
		t.Errorf("expected empty uuid for thread with no messages, got %q", uuid)
	}
}

func TestMultipleThreads(t *testing.T) {
	db := tempDB(t)

	for i := 0; i < 5; i++ {
		thread := &Thread{Title: fmt.Sprintf("Thread %d", i), Mode: ModeAuto, Source: "sse"}
		db.CreateThread(thread)
		db.AddThreadMessage(&ThreadMessage{
			ThreadID: thread.ID,
			Role:     "user",
			Content:  fmt.Sprintf("Query %d", i),
		})
	}

	count, _ := db.CountThreads()
	if count != 5 {
		t.Errorf("expected 5, got %d", count)
	}

	threads, _ := db.ListThreads(3)
	if len(threads) != 3 {
		t.Errorf("expected 3 (limited), got %d", len(threads))
	}
}

func TestFormatThread(t *testing.T) {
	thread := &Thread{
		ID:           1,
		Title:        "Test thread",
		Mode:         ModeAuto,
		Model:        "turbo",
		Source:       "sse",
		MessageCount: 2,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	messages := []ThreadMessage{
		{Role: "user", Content: "What is Go?"},
		{
			Role:       "assistant",
			Content:    "Go is a programming language.",
			DurationMs: 1500,
			Citations:  []Citation{{URL: "https://go.dev", Title: "Go Dev"}},
		},
	}

	output := FormatThread(thread, messages)
	if output == "" {
		t.Error("expected non-empty output")
	}
	if !strings.Contains(output, "Thread #1") {
		t.Error("expected thread ID in output")
	}
	if !strings.Contains(output, "What is Go?") {
		t.Error("expected user message in output")
	}
	if !strings.Contains(output, "Go is a programming language") {
		t.Error("expected assistant message in output")
	}
}

// --- Live: Perplexity SSE search ---

func TestLiveSSESearch(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("set LIVE_TEST=1 to run live tests")
	}

	cfg := Config{DataDir: t.TempDir(), Timeout: defaultTimeout}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := client.Search(ctx, "what is golang", DefaultSearchOptions())
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if result.Answer == "" {
		t.Error("expected non-empty answer")
	}
	if result.Query != "what is golang" {
		t.Errorf("unexpected query: %s", result.Query)
	}
	t.Logf("Answer preview: %.100s", result.Answer)
	t.Logf("Citations: %d, WebResults: %d", len(result.Citations), len(result.WebResults))

	// Save to DB
	db := tempDB(t)
	if err := db.SaveSearch(result); err != nil {
		t.Fatalf("save: %v", err)
	}
	count, _ := db.Count()
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

// --- Live: Perplexity Labs ---

func TestLiveLabsSearch(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("set LIVE_TEST=1 to run live tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	lc, err := NewLabsClient(ctx)
	if err != nil {
		t.Fatalf("labs connect: %v", err)
	}
	defer lc.Close()

	result, err := lc.Ask(ctx, "what is rust programming language", ModelR1)
	if err != nil {
		t.Fatalf("labs ask: %v", err)
	}

	if result.Output == "" {
		t.Error("expected non-empty output")
	}
	t.Logf("Output preview: %.100s", result.Output)
}

// --- Email Provider Registry Tests ---

func TestEmailProviderRegistry(t *testing.T) {
	if len(allProviders) != 7 {
		t.Fatalf("expected 7 providers, got %d", len(allProviders))
	}

	expectedTiers := map[string]ProviderTier{
		"mail.tm":      TierPrivate,
		"mail.gw":      TierPrivate,
		"tempmail.lol": TierPrivate,
		"guerrillamail": TierSession,
		"dropmail":      TierSession,
		"tempmail.plus": TierPublic,
		"inboxkitten":   TierPublic,
	}

	for _, p := range allProviders {
		expected, ok := expectedTiers[p.Name]
		if !ok {
			t.Errorf("unexpected provider %q", p.Name)
			continue
		}
		if p.Tier != expected {
			t.Errorf("provider %q: expected tier %d, got %d", p.Name, expected, p.Tier)
		}
		if p.NewFunc == nil {
			t.Errorf("provider %q has nil NewFunc", p.Name)
		}
	}
}

func TestShuffledByTier(t *testing.T) {
	ordered := shuffledByTier(allProviders)
	if len(ordered) != 7 {
		t.Fatalf("expected 7 providers, got %d", len(ordered))
	}

	// Verify tier ordering is maintained: all Private before Session before Public
	lastTier := TierPrivate
	for _, p := range ordered {
		if p.Tier < lastTier {
			t.Errorf("provider %q (tier %d) appeared after tier %d — tier ordering violated", p.Name, p.Tier, lastTier)
		}
		lastTier = p.Tier
	}

	// Verify correct counts per tier
	tierCounts := map[ProviderTier]int{}
	for _, p := range ordered {
		tierCounts[p.Tier]++
	}
	if tierCounts[TierPrivate] != 3 {
		t.Errorf("expected 3 private, got %d", tierCounts[TierPrivate])
	}
	if tierCounts[TierSession] != 2 {
		t.Errorf("expected 2 session, got %d", tierCounts[TierSession])
	}
	if tierCounts[TierPublic] != 2 {
		t.Errorf("expected 2 public, got %d", tierCounts[TierPublic])
	}
}

func TestRandomString(t *testing.T) {
	s1 := randomString(10)
	s2 := randomString(10)
	if len(s1) != 10 {
		t.Errorf("expected length 10, got %d", len(s1))
	}
	if s1 == s2 {
		t.Error("expected different random strings")
	}
}

// --- Live: Temp email provider ---

func TestLiveTempEmail(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("set LIVE_TEST=1 to run live tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewTempEmailClient(ctx)
	if err != nil {
		t.Fatalf("create temp email: %v", err)
	}

	email := client.Email()
	if email == "" {
		t.Error("expected non-empty email address")
	}
	t.Logf("Generated email: %s", email)
}

// --- Live: Individual provider tests ---

func TestLiveMailTM(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("set LIVE_TEST=1 to run live tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewMailTMClient(ctx)
	if err != nil {
		t.Fatalf("mail.tm: %v", err)
	}
	t.Logf("mail.tm email: %s", client.Email())
}

func TestLiveMailGW(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("set LIVE_TEST=1 to run live tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewMailGWClient(ctx)
	if err != nil {
		t.Fatalf("mail.gw: %v", err)
	}
	t.Logf("mail.gw email: %s", client.Email())
}

func TestLiveTempMailLol(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("set LIVE_TEST=1 to run live tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewTempMailLolClient(ctx)
	if err != nil {
		t.Fatalf("tempmail.lol: %v", err)
	}
	t.Logf("tempmail.lol email: %s", client.Email())
}

func TestLiveGuerrillaMail(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("set LIVE_TEST=1 to run live tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewGuerrillaClient(ctx)
	if err != nil {
		t.Fatalf("guerrillamail: %v", err)
	}
	t.Logf("guerrillamail email: %s", client.Email())
}

func TestLiveDropMail(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("set LIVE_TEST=1 to run live tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewDropMailClient(ctx)
	if err != nil {
		t.Fatalf("dropmail: %v", err)
	}
	t.Logf("dropmail email: %s", client.Email())
}

func TestLiveTempMailPlus(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("set LIVE_TEST=1 to run live tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewTempMailPlusClient(ctx)
	if err != nil {
		t.Fatalf("tempmail.plus: %v", err)
	}
	t.Logf("tempmail.plus email: %s", client.Email())
}

func TestLiveInboxKitten(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("set LIVE_TEST=1 to run live tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewInboxKittenClient(ctx)
	if err != nil {
		t.Fatalf("inboxkitten: %v", err)
	}
	t.Logf("inboxkitten email: %s", client.Email())
}

// --- Live: Full register workflow ---

func TestLiveAutoRegister(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("set LIVE_TEST=1 to run live tests")
	}

	db := tempDB(t)
	cfg := Config{DataDir: t.TempDir(), Timeout: defaultTimeout}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := client.AutoRegister(ctx, db); err != nil {
		t.Fatalf("auto register: %v", err)
	}

	if !client.IsAuthenticated() {
		t.Error("expected authenticated after register")
	}
	if client.CopilotQueries() != defaultCopilotQueries {
		t.Errorf("expected %d pro queries, got %d", defaultCopilotQueries, client.CopilotQueries())
	}

	// Verify account saved to DB
	total, active, _ := db.CountAccounts()
	if total != 1 || active != 1 {
		t.Errorf("expected 1/1 accounts, got %d/%d", total, active)
	}
}
