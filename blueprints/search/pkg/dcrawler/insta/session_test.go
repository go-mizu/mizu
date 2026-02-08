package insta

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionSaveLoad(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.json")

	sess := &Session{
		Username: "testuser",
		UserID:   "12345",
		Cookies: map[string]string{
			"sessionid": "abc123",
			"csrftoken": "xyz789",
			"ds_user_id": "12345",
		},
		SavedAt: time.Now().Truncate(time.Second),
	}

	// Save
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Load
	loadData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var loaded Session
	if err := json.Unmarshal(loadData, &loaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if loaded.Username != sess.Username {
		t.Errorf("Username = %q, want %q", loaded.Username, sess.Username)
	}
	if loaded.UserID != sess.UserID {
		t.Errorf("UserID = %q, want %q", loaded.UserID, sess.UserID)
	}
	if loaded.Cookies["sessionid"] != "abc123" {
		t.Errorf("sessionid cookie = %q, want abc123", loaded.Cookies["sessionid"])
	}
	if loaded.Cookies["csrftoken"] != "xyz789" {
		t.Errorf("csrftoken cookie = %q, want xyz789", loaded.Cookies["csrftoken"])
	}
}

func TestSessionFilePermissions(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "sessions", "test.json")

	// Use Client.SaveSession to test dir creation + file permissions
	cfg := DefaultConfig()
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	client.username = "testuser"
	client.userID = "12345"

	if err := client.SaveSession(path); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	// Check file exists
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	// Check permissions (should be 0600 - owner read/write only)
	perm := fi.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("permissions = %o, want 600", perm)
	}
}

func TestTwoFactorError(t *testing.T) {
	err := &TwoFactorError{Identifier: "abc123"}

	if err.Error() != "two-factor authentication required" {
		t.Errorf("Error() = %q", err.Error())
	}
	if err.Identifier != "abc123" {
		t.Errorf("Identifier = %q, want abc123", err.Identifier)
	}
}

func TestClientIsLoggedIn(t *testing.T) {
	cfg := DefaultConfig()
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if client.IsLoggedIn() {
		t.Error("new client should not be logged in")
	}
	if client.Username() != "" {
		t.Errorf("Username = %q, want empty", client.Username())
	}

	client.loggedIn = true
	client.username = "test"

	if !client.IsLoggedIn() {
		t.Error("client should be logged in after setting flag")
	}
	if client.Username() != "test" {
		t.Errorf("Username = %q, want test", client.Username())
	}
}

func TestApplySession(t *testing.T) {
	cfg := DefaultConfig()
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	sess := &Session{
		Username: "restored",
		UserID:   "54321",
		Cookies: map[string]string{
			"sessionid": "sess_abc",
			"csrftoken": "csrf_xyz",
		},
	}

	if err := client.ApplySession(sess); err != nil {
		t.Fatalf("ApplySession: %v", err)
	}

	if !client.IsLoggedIn() {
		t.Error("should be logged in after applying session")
	}
	if client.Username() != "restored" {
		t.Errorf("Username = %q, want restored", client.Username())
	}
	if client.csrfToken != "csrf_xyz" {
		t.Errorf("csrfToken = %q, want csrf_xyz", client.csrfToken)
	}
}
