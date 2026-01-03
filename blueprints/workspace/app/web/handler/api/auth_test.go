package api_test

import (
	"net/http"
	"testing"

	"github.com/go-mizu/blueprints/workspace/feature/users"
)

func TestAuthRegister(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
	}{
		{
			name: "valid registration",
			body: map[string]string{
				"email":    "alice@example.com",
				"name":     "Alice",
				"password": "password123",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "missing email",
			body: map[string]string{
				"name":     "Bob",
				"password": "password123",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing name uses email",
			body: map[string]string{
				"email":    "bob@example.com",
				"password": "password123",
			},
			wantStatus: http.StatusCreated, // Name defaults to email
		},
		{
			name: "missing password",
			body: map[string]string{
				"email": "bob@example.com",
				"name":  "Bob",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "short password",
			body: map[string]string{
				"email":    "bob@example.com",
				"name":     "Bob",
				"password": "123",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("POST", "/api/v1/auth/register", tt.body)
			ts.ExpectStatus(resp, tt.wantStatus)

			if tt.wantStatus == http.StatusCreated {
				var user users.User
				ts.ParseJSON(resp, &user)

				if user.Email != tt.body["email"] {
					t.Errorf("email = %q, want %q", user.Email, tt.body["email"])
				}
				// When name is empty, API uses email as name
				expectedName := tt.body["name"]
				if expectedName == "" {
					expectedName = tt.body["email"]
				}
				if user.Name != expectedName {
					t.Errorf("name = %q, want %q", user.Name, expectedName)
				}
				if user.ID == "" {
					t.Error("user ID should not be empty")
				}

				// Check session cookie
				var hasCookie bool
				for _, c := range resp.Cookies() {
					if c.Name == "workspace_session" {
						hasCookie = true
						break
					}
				}
				if !hasCookie {
					t.Error("missing session cookie")
				}
			}
			resp.Body.Close()
		})
	}
}

func TestAuthRegisterDuplicateEmail(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register first user
	ts.Register("duplicate@example.com", "First", "password123")

	// Try to register with same email
	resp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email":    "duplicate@example.com",
		"name":     "Second",
		"password": "password123",
	})
	ts.ExpectStatus(resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestAuthLogin(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register a user first
	ts.Register("logintest@example.com", "Login Test", "password123")

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
	}{
		{
			name: "valid login",
			body: map[string]string{
				"email":    "logintest@example.com",
				"password": "password123",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "wrong password",
			body: map[string]string{
				"email":    "logintest@example.com",
				"password": "wrongpassword",
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "non-existent user",
			body: map[string]string{
				"email":    "nonexistent@example.com",
				"password": "password123",
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "empty body",
			body:       map[string]string{},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("POST", "/api/v1/auth/login", tt.body)
			ts.ExpectStatus(resp, tt.wantStatus)

			if tt.wantStatus == http.StatusOK {
				var user users.User
				ts.ParseJSON(resp, &user)

				if user.Email != tt.body["email"] {
					t.Errorf("email = %q, want %q", user.Email, tt.body["email"])
				}

				// Check session cookie
				var hasCookie bool
				for _, c := range resp.Cookies() {
					if c.Name == "workspace_session" {
						hasCookie = true
						break
					}
				}
				if !hasCookie {
					t.Error("missing session cookie")
				}
			}
			resp.Body.Close()
		})
	}
}

func TestAuthLogout(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register and get session
	_, cookie := ts.Register("logout@example.com", "Logout Test", "password123")

	t.Run("authenticated logout", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/auth/logout", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		// Check that cookie is cleared
		for _, c := range resp.Cookies() {
			if c.Name == "workspace_session" && c.MaxAge < 0 {
				// Cookie is being cleared
				break
			}
		}
		resp.Body.Close()
	})

	t.Run("unauthenticated logout", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/auth/logout", nil)
		ts.ExpectStatus(resp, http.StatusUnauthorized)
		resp.Body.Close()
	})
}

func TestAuthMe(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register and get session
	registeredUser, cookie := ts.Register("me@example.com", "Me Test", "password123")

	t.Run("authenticated", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/auth/me", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var user users.User
		ts.ParseJSON(resp, &user)

		if user.ID != registeredUser.ID {
			t.Errorf("user ID = %q, want %q", user.ID, registeredUser.ID)
		}
		if user.Email != registeredUser.Email {
			t.Errorf("email = %q, want %q", user.Email, registeredUser.Email)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/auth/me", nil)
		ts.ExpectStatus(resp, http.StatusUnauthorized)
		resp.Body.Close()
	})

	t.Run("invalid session", func(t *testing.T) {
		invalidCookie := &http.Cookie{
			Name:  "workspace_session",
			Value: "invalid-session-id",
		}
		resp := ts.Request("GET", "/api/v1/auth/me", nil, invalidCookie)
		ts.ExpectStatus(resp, http.StatusUnauthorized)
		resp.Body.Close()
	})
}

func TestAuthSessionPersistence(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register user
	_, cookie := ts.Register("session@example.com", "Session Test", "password123")

	// Make multiple authenticated requests
	for i := 0; i < 3; i++ {
		resp := ts.Request("GET", "/api/v1/auth/me", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	}
}
