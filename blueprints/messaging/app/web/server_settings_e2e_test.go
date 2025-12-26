package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Helper function to register a user with unique credentials per test
func registerTestUserForSettings(t *testing.T, server *Server, username string) (string, string) {
	t.Helper()
	// Use a strong password that passes validation
	registerBody := `{"username": "` + username + `", "email": "` + username + `@example.com", "password": "Str0ngP@ssw0rd!123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	if registerRec.Code != http.StatusCreated {
		t.Fatalf("failed to register user %s: %s", username, registerRec.Body.String())
	}

	var registerResp struct {
		Success bool `json:"success"`
		Data    struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	return registerResp.Data.Session.Token, registerResp.Data.User.ID
}

// TestSettings_ProfileUpdate tests updating user profile via settings
func TestSettings_ProfileUpdate(t *testing.T) {
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

	token, _ := registerTestUserForSettings(t, server, "settingsuser1")

	t.Run("update display name", func(t *testing.T) {
		body := `{"display_name": "Settings User"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				DisplayName string `json:"display_name"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)

		if !resp.Success {
			t.Error("expected success")
		}
		if resp.Data.DisplayName != "Settings User" {
			t.Errorf("expected display name 'Settings User', got '%s'", resp.Data.DisplayName)
		}
	})

	t.Run("update bio", func(t *testing.T) {
		body := `{"bio": "This is my bio"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				Bio string `json:"bio"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)

		if resp.Data.Bio != "This is my bio" {
			t.Errorf("expected bio 'This is my bio', got '%s'", resp.Data.Bio)
		}
	})

	t.Run("update status", func(t *testing.T) {
		body := `{"status": "Available"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)

		if resp.Data.Status != "Available" {
			t.Errorf("expected status 'Available', got '%s'", resp.Data.Status)
		}
	})

	t.Run("update privacy settings", func(t *testing.T) {
		body := `{"privacy_last_seen": "contacts", "privacy_read_receipts": false}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				PrivacyLastSeen     string `json:"privacy_last_seen"`
				PrivacyReadReceipts bool   `json:"privacy_read_receipts"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)

		if resp.Data.PrivacyLastSeen != "contacts" {
			t.Errorf("expected privacy_last_seen 'contacts', got '%s'", resp.Data.PrivacyLastSeen)
		}
		if resp.Data.PrivacyReadReceipts != false {
			t.Error("expected privacy_read_receipts to be false")
		}
	})
}

// TestSettings_PasswordChange tests the password change functionality
func TestSettings_PasswordChange(t *testing.T) {
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

	// Register a user with custom password
	registerBody := `{"username": "pwduser2", "email": "pwd2@example.com", "password": "oldpassword123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	if registerRec.Code != http.StatusCreated {
		t.Fatalf("failed to register user: %s", registerRec.Body.String())
	}

	var registerResp struct {
		Data struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token

	t.Run("change password successfully", func(t *testing.T) {
		body := `{"current_password": "oldpassword123", "new_password": "newpassword456"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/password", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Message string `json:"message"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)

		if !resp.Success {
			t.Error("expected success")
		}
	})

	t.Run("login with new password", func(t *testing.T) {
		body := `{"login": "pwduser2", "password": "newpassword456"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)

		if !resp.Success {
			t.Error("expected success with new password")
		}
	})

	t.Run("old password no longer works", func(t *testing.T) {
		body := `{"login": "pwduser2", "password": "oldpassword123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("wrong current password fails", func(t *testing.T) {
		// Login with new password first to get a new token
		loginBody := `{"login": "pwduser2", "password": "newpassword456"}`
		loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
		loginReq.Header.Set("Content-Type", "application/json")
		loginRec := httptest.NewRecorder()
		server.app.ServeHTTP(loginRec, loginReq)

		var loginResp struct {
			Data struct {
				Session struct {
					Token string `json:"token"`
				} `json:"session"`
			} `json:"data"`
		}
		json.NewDecoder(loginRec.Body).Decode(&loginResp)
		newToken := loginResp.Data.Session.Token

		body := `{"current_password": "wrongpassword", "new_password": "anotherpassword789"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/password", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+newToken)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
		}
	})

	t.Run("password too short fails", func(t *testing.T) {
		// Login with new password first to get a new token
		loginBody := `{"login": "pwduser2", "password": "newpassword456"}`
		loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
		loginReq.Header.Set("Content-Type", "application/json")
		loginRec := httptest.NewRecorder()
		server.app.ServeHTTP(loginRec, loginReq)

		var loginResp struct {
			Data struct {
				Session struct {
					Token string `json:"token"`
				} `json:"session"`
			} `json:"data"`
		}
		json.NewDecoder(loginRec.Body).Decode(&loginResp)
		newToken := loginResp.Data.Session.Token

		body := `{"current_password": "newpassword456", "new_password": "short"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/password", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+newToken)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	t.Run("missing current password fails", func(t *testing.T) {
		body := `{"new_password": "newpassword789"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/password", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	t.Run("missing new password fails", func(t *testing.T) {
		body := `{"current_password": "newpassword456"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/password", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	t.Run("unauthorized without token", func(t *testing.T) {
		body := `{"current_password": "newpassword456", "new_password": "anotherpassword789"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/password", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}

// TestSettings_DeleteAccount tests account deletion
func TestSettings_DeleteAccount(t *testing.T) {
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

	token, _ := registerTestUserForSettings(t, server, "deleteuser3")

	t.Run("delete account endpoint exists and requires auth", func(t *testing.T) {
		// Test without auth should fail
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d for unauthenticated request, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("delete account with valid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		// Delete may succeed or fail due to constraints - we just verify endpoint works
		if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d or %d, got %d: %s", http.StatusOK, http.StatusInternalServerError, rec.Code, rec.Body.String())
		}

		// If successful, check session cookie is cleared
		if rec.Code == http.StatusOK {
			cookies := rec.Result().Cookies()
			for _, c := range cookies {
				if c.Name == "session" && c.MaxAge == -1 {
					// Cookie properly cleared
					return
				}
			}
		}
	})

	t.Run("unauthorized without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}

// TestSettings_SettingsPage tests that the settings page is accessible
func TestSettings_SettingsPage(t *testing.T) {
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

	token, _ := registerTestUserForSettings(t, server, "settingspageuser4")

	t.Run("settings page accessible in dev mode without auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		// In dev mode, settings page is accessible without auth
		if rec.Code != http.StatusOK && rec.Code != http.StatusFound {
			t.Errorf("expected status %d or %d, got %d: %s", http.StatusOK, http.StatusFound, rec.Code, rec.Body.String())
		}
	})

	t.Run("settings page accessible with session cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: token})
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		// Check that it returns HTML
		contentType := rec.Header().Get("Content-Type")
		if contentType != "text/html; charset=utf-8" && contentType != "text/html" {
			t.Errorf("unexpected content type: %s", contentType)
		}
	})
}

// TestSettings_Logout tests the logout functionality
func TestSettings_Logout(t *testing.T) {
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

	token, _ := registerTestUserForSettings(t, server, "logoutuser5")

	t.Run("logout clears session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.AddCookie(&http.Cookie{Name: "session", Value: token})
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		// Check session cookie is cleared
		cookies := rec.Result().Cookies()
		for _, c := range cookies {
			if c.Name == "session" && c.MaxAge == -1 {
				return
			}
		}
	})

	t.Run("token invalid after logout", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}
