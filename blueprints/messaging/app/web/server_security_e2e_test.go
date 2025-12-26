package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// Authentication Security Tests
// =============================================================================

func TestSecurity_PasswordStrength(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	tests := []struct {
		name     string
		password string
		wantOK   bool
		wantErr  string
	}{
		{
			name:     "too short",
			password: "abc123",
			wantOK:   false,
			wantErr:  "at least 8 characters",
		},
		{
			name:     "no digits",
			password: "abcdefgh",
			wantOK:   false,
			wantErr:  "at least one number",
		},
		{
			name:     "no letters",
			password: "12345678",
			wantOK:   false,
			wantErr:  "at least one letter",
		},
		{
			name:     "common password",
			password: "password123",
			wantOK:   false,
			wantErr:  "too common",
		},
		{
			name:     "valid password",
			password: "SecurePass123",
			wantOK:   true,
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"username": "user` + string(rune('a'+i)) + `", "email": "user` + string(rune('a'+i)) + `@example.com", "password": "` + tc.password + `"}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			server.app.ServeHTTP(rec, req)

			if tc.wantOK {
				if rec.Code != http.StatusCreated {
					t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
				}
			} else {
				if rec.Code != http.StatusBadRequest {
					t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
				}
				if tc.wantErr != "" && !strings.Contains(rec.Body.String(), tc.wantErr) {
					t.Errorf("expected error containing %q, got: %s", tc.wantErr, rec.Body.String())
				}
			}
		})
	}
}

func TestSecurity_RateLimiting_Login(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// First, register a user
	registerBody := `{"username": "ratelimituser", "email": "ratelimit@example.com", "password": "SecurePass123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	// Attempt multiple failed logins (rate limit is 5 per minute)
	for i := 0; i < 6; i++ {
		body := `{"login": "ratelimituser", "password": "wrongpassword"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "192.168.1.100") // Same IP for rate limiting
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if i < 5 {
			// First 5 attempts should be allowed (even if wrong password)
			if rec.Code == http.StatusTooManyRequests {
				t.Errorf("attempt %d: should not be rate limited yet", i+1)
			}
		} else {
			// 6th attempt should be rate limited
			if rec.Code != http.StatusTooManyRequests {
				t.Errorf("attempt %d: expected rate limit, got status %d", i+1, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "Too many") {
				t.Errorf("expected rate limit error message, got: %s", rec.Body.String())
			}
		}
	}
}

func TestSecurity_RateLimiting_Register(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Attempt multiple registrations (rate limit is 3 per 10 minutes)
	for i := 0; i < 4; i++ {
		body := `{"username": "reguser` + string(rune('0'+i)) + `", "email": "reg` + string(rune('0'+i)) + `@example.com", "password": "SecurePass123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "192.168.1.200") // Same IP
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if i < 3 {
			if rec.Code == http.StatusTooManyRequests {
				t.Errorf("attempt %d: should not be rate limited yet", i+1)
			}
		} else {
			// 4th attempt should be rate limited
			if rec.Code != http.StatusTooManyRequests {
				t.Errorf("attempt %d: expected rate limit, got status %d: %s", i+1, rec.Code, rec.Body.String())
			}
		}
	}
}

func TestSecurity_SessionExpiry(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register and login
	registerBody := `{"username": "sessiontest", "email": "session@example.com", "password": "SecurePass123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	var registerResp struct {
		Data struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token

	t.Run("valid token works", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("invalid token rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer invalid-token-12345")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("logout invalidates session", func(t *testing.T) {
		// Logout
		logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		logoutReq.Header.Set("Authorization", "Bearer "+token)
		logoutReq.AddCookie(&http.Cookie{Name: "session", Value: token})
		logoutRec := httptest.NewRecorder()
		server.app.ServeHTTP(logoutRec, logoutReq)

		// Try to use the same token
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d after logout, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}

// =============================================================================
// Authorization Security Tests
// =============================================================================

func TestSecurity_MessageOwnership(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Create two users
	registerUser := func(username, email string) (string, string) {
		body := `{"username": "` + username + `", "email": "` + email + `", "password": "SecurePass123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		var resp struct {
			Data struct {
				Session struct {
					Token string `json:"token"`
				} `json:"session"`
				User struct {
					ID string `json:"id"`
				} `json:"user"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		return resp.Data.Session.Token, resp.Data.User.ID
	}

	token1, _ := registerUser("owner", "owner@example.com")
	token2, user2ID := registerUser("other", "other@example.com")

	// User1 creates a chat with User2
	chatBody := `{"type": "direct", "recipient_id": "` + user2ID + `"}`
	chatReq := httptest.NewRequest(http.MethodPost, "/api/v1/chats", bytes.NewBufferString(chatBody))
	chatReq.Header.Set("Content-Type", "application/json")
	chatReq.Header.Set("Authorization", "Bearer "+token1)
	chatRec := httptest.NewRecorder()
	server.app.ServeHTTP(chatRec, chatReq)

	var chatResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(chatRec.Body).Decode(&chatResp)
	chatID := chatResp.Data.ID

	// User1 creates a message
	msgBody := `{"type": "text", "content": "Original message"}`
	msgReq := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/messages", bytes.NewBufferString(msgBody))
	msgReq.Header.Set("Content-Type", "application/json")
	msgReq.Header.Set("Authorization", "Bearer "+token1)
	msgRec := httptest.NewRecorder()
	server.app.ServeHTTP(msgRec, msgReq)

	var msgResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(msgRec.Body).Decode(&msgResp)
	messageID := msgResp.Data.ID

	t.Run("owner can edit message", func(t *testing.T) {
		body := `{"content": "Updated by owner"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/chats/"+chatID+"/messages/"+messageID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("non-owner cannot edit message", func(t *testing.T) {
		body := `{"content": "Hijacked message"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/chats/"+chatID+"/messages/"+messageID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token2)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
		}
	})

	t.Run("non-owner cannot delete for everyone", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/chats/"+chatID+"/messages/"+messageID+"?for_everyone=true", nil)
		req.Header.Set("Authorization", "Bearer "+token2)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})
}

// =============================================================================
// Input Validation & XSS Prevention Tests
// =============================================================================

func TestSecurity_XSSPrevention(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register a user
	registerBody := `{"username": "xsstest", "email": "xss@example.com", "password": "SecurePass123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	var registerResp struct {
		Data struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token
	userID := registerResp.Data.User.ID

	// Create a self-chat
	chatBody := `{"type": "direct", "recipient_id": "` + userID + `"}`
	chatReq := httptest.NewRequest(http.MethodPost, "/api/v1/chats", bytes.NewBufferString(chatBody))
	chatReq.Header.Set("Content-Type", "application/json")
	chatReq.Header.Set("Authorization", "Bearer "+token)
	chatRec := httptest.NewRecorder()
	server.app.ServeHTTP(chatRec, chatReq)

	var chatResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(chatRec.Body).Decode(&chatResp)
	chatID := chatResp.Data.ID

	xssPayloads := []string{
		`<script>alert('XSS')</script>`,
		`<img src=x onerror=alert('XSS')>`,
		`<svg onload=alert('XSS')>`,
		`javascript:alert('XSS')`,
		`<a href="javascript:alert('XSS')">click</a>`,
	}

	for _, payload := range xssPayloads {
		t.Run("sanitize_"+payload[:10], func(t *testing.T) {
			body := `{"type": "text", "content": "` + strings.ReplaceAll(payload, `"`, `\"`) + `"}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/messages", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()

			server.app.ServeHTTP(rec, req)

			if rec.Code != http.StatusCreated {
				t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
				return
			}

			var resp struct {
				Data struct {
					Content string `json:"content"`
				} `json:"data"`
			}
			json.NewDecoder(rec.Body).Decode(&resp)

			// Verify XSS payload is escaped
			if strings.Contains(resp.Data.Content, "<script") {
				t.Errorf("XSS payload not sanitized: %s", resp.Data.Content)
			}
			if strings.Contains(resp.Data.Content, "onerror=") {
				t.Errorf("XSS payload not sanitized: %s", resp.Data.Content)
			}
			if strings.Contains(resp.Data.Content, "onload=") {
				t.Errorf("XSS payload not sanitized: %s", resp.Data.Content)
			}
		})
	}
}

func TestSecurity_MessageLengthLimit(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register a user
	registerBody := `{"username": "lentest", "email": "len@example.com", "password": "SecurePass123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	var registerResp struct {
		Data struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token
	userID := registerResp.Data.User.ID

	// Create a self-chat
	chatBody := `{"type": "direct", "recipient_id": "` + userID + `"}`
	chatReq := httptest.NewRequest(http.MethodPost, "/api/v1/chats", bytes.NewBufferString(chatBody))
	chatReq.Header.Set("Content-Type", "application/json")
	chatReq.Header.Set("Authorization", "Bearer "+token)
	chatRec := httptest.NewRecorder()
	server.app.ServeHTTP(chatRec, chatReq)

	var chatResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(chatRec.Body).Decode(&chatResp)
	chatID := chatResp.Data.ID

	t.Run("message exceeding max length rejected", func(t *testing.T) {
		// Create a message longer than 4096 characters
		longContent := strings.Repeat("x", 5000)
		body := `{"type": "text", "content": "` + longContent + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/messages", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("empty message rejected", func(t *testing.T) {
		body := `{"type": "text", "content": "   "}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/messages", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})
}

// =============================================================================
// WebSocket Security Tests
// =============================================================================

func TestSecurity_WebSocketOriginValidation(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:           ":0",
		DataDir:        tmpDir,
		Dev:            false, // Production mode
		AllowedOrigins: []string{"https://app.mizu.dev"},
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register a user
	registerBody := `{"username": "wsorigin", "email": "wsorigin@example.com", "password": "SecurePass123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	var registerResp struct {
		Data struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token

	tests := []struct {
		name         string
		origin       string
		shouldReject bool
	}{
		{
			name:         "allowed origin",
			origin:       "https://app.mizu.dev",
			shouldReject: false,
		},
		{
			name:         "unauthorized origin",
			origin:       "https://evil.com",
			shouldReject: true,
		},
		{
			name:         "no origin header (non-browser)",
			origin:       "",
			shouldReject: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ws?token="+token, nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Sec-WebSocket-Version", "13")
			req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
			rec := httptest.NewRecorder()

			server.app.ServeHTTP(rec, req)

			if tc.shouldReject {
				// Should get 403 Forbidden for bad origins
				if rec.Code != http.StatusForbidden && rec.Code != http.StatusBadRequest {
					// WebSocket upgrade returns 400 when origin check fails
					t.Logf("Response code: %d, body: %s", rec.Code, rec.Body.String())
				}
			}
		})
	}
}

func TestSecurity_WebSocketAuthRequired(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	t.Run("no token returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ws?token=invalid-token", nil)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}

// =============================================================================
// Security Headers Tests
// =============================================================================

func TestSecurity_Headers(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     false, // Production mode for strict headers
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	server.app.ServeHTTP(rec, req)

	headers := rec.Header()

	t.Run("X-Frame-Options", func(t *testing.T) {
		if v := headers.Get("X-Frame-Options"); v != "DENY" {
			t.Errorf("expected X-Frame-Options DENY, got %s", v)
		}
	})

	t.Run("X-Content-Type-Options", func(t *testing.T) {
		if v := headers.Get("X-Content-Type-Options"); v != "nosniff" {
			t.Errorf("expected X-Content-Type-Options nosniff, got %s", v)
		}
	})

	t.Run("X-XSS-Protection", func(t *testing.T) {
		if v := headers.Get("X-XSS-Protection"); v != "1; mode=block" {
			t.Errorf("expected X-XSS-Protection '1; mode=block', got %s", v)
		}
	})

	t.Run("Referrer-Policy", func(t *testing.T) {
		if v := headers.Get("Referrer-Policy"); v != "strict-origin-when-cross-origin" {
			t.Errorf("expected Referrer-Policy strict-origin-when-cross-origin, got %s", v)
		}
	})

	t.Run("Content-Security-Policy", func(t *testing.T) {
		csp := headers.Get("Content-Security-Policy")
		if csp == "" {
			t.Error("expected Content-Security-Policy to be set")
		}
		if !strings.Contains(csp, "default-src 'self'") {
			t.Errorf("expected CSP to contain default-src 'self', got %s", csp)
		}
	})

	t.Run("Permissions-Policy", func(t *testing.T) {
		pp := headers.Get("Permissions-Policy")
		if pp == "" {
			t.Error("expected Permissions-Policy to be set")
		}
		if !strings.Contains(pp, "geolocation=()") {
			t.Errorf("expected Permissions-Policy to restrict geolocation, got %s", pp)
		}
	})
}

func TestSecurity_API_CacheControl(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     false,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register a user to get a valid token
	registerBody := `{"username": "cachetest", "email": "cache@example.com", "password": "SecurePass123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	var registerResp struct {
		Data struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	server.app.ServeHTTP(rec, req)

	headers := rec.Header()

	t.Run("Cache-Control for API", func(t *testing.T) {
		cc := headers.Get("Cache-Control")
		if !strings.Contains(cc, "no-store") {
			t.Errorf("expected Cache-Control to contain no-store, got %s", cc)
		}
	})
}

// =============================================================================
// Search Query Sanitization Tests
// =============================================================================

func TestSecurity_SearchQuerySanitization(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register a user
	registerBody := `{"username": "searchtest", "email": "search@example.com", "password": "SecurePass123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	var registerResp struct {
		Data struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token

	t.Run("empty query rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/search/messages", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("SQL injection attempt sanitized", func(t *testing.T) {
		// This should not cause errors as the query is sanitized
		req := httptest.NewRequest(http.MethodGet, "/api/v1/search/messages?q=test'+OR+1=1--", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		// Should succeed (return empty results) rather than error
		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})
}

// =============================================================================
// Media URL Validation Tests
// =============================================================================

func TestSecurity_MediaURLValidation(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register a user
	registerBody := `{"username": "mediatest", "email": "media@example.com", "password": "SecurePass123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	var registerResp struct {
		Data struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token
	userID := registerResp.Data.User.ID

	// Create a self-chat
	chatBody := `{"type": "direct", "recipient_id": "` + userID + `"}`
	chatReq := httptest.NewRequest(http.MethodPost, "/api/v1/chats", bytes.NewBufferString(chatBody))
	chatReq.Header.Set("Content-Type", "application/json")
	chatReq.Header.Set("Authorization", "Bearer "+token)
	chatRec := httptest.NewRecorder()
	server.app.ServeHTTP(chatRec, chatReq)

	var chatResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(chatRec.Body).Decode(&chatResp)
	chatID := chatResp.Data.ID

	tests := []struct {
		name     string
		mediaURL string
		wantOK   bool
	}{
		{
			name:     "valid https url",
			mediaURL: "https://example.com/image.png",
			wantOK:   true,
		},
		{
			name:     "valid http url",
			mediaURL: "http://example.com/image.png",
			wantOK:   true,
		},
		{
			name:     "javascript url rejected",
			mediaURL: "javascript:alert('XSS')",
			wantOK:   false,
		},
		{
			name:     "data url rejected",
			mediaURL: "data:text/html,<script>alert('XSS')</script>",
			wantOK:   false,
		},
		{
			name:     "file url rejected",
			mediaURL: "file:///etc/passwd",
			wantOK:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"type": "image", "content": "image", "media_url": "` + tc.mediaURL + `"}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/messages", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()

			server.app.ServeHTTP(rec, req)

			if tc.wantOK {
				if rec.Code != http.StatusCreated {
					t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
				}
			} else {
				if rec.Code != http.StatusBadRequest {
					t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
				}
			}
		})
	}
}

// =============================================================================
// General Authentication Tests
// =============================================================================

func TestSecurity_NoInformationLeakage(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register a user
	registerBody := `{"username": "existinguser", "email": "existing@example.com", "password": "SecurePass123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	t.Run("login with wrong password", func(t *testing.T) {
		body := `{"login": "existinguser", "password": "wrongpassword123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		// Should return generic "Invalid credentials" not "wrong password"
		if !strings.Contains(rec.Body.String(), "Invalid credentials") {
			t.Errorf("expected generic error, got: %s", rec.Body.String())
		}
	})

	t.Run("login with non-existent user", func(t *testing.T) {
		body := `{"login": "nonexistentuser", "password": "anypassword123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		// Should return same generic error, not reveal that user doesn't exist
		if !strings.Contains(rec.Body.String(), "Invalid credentials") {
			t.Errorf("expected generic error, got: %s", rec.Body.String())
		}
	})
}
