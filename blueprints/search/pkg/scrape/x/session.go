package x

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Session holds serializable auth data for X/Twitter.
type Session struct {
	Username  string         `json:"username"`
	AuthToken string         `json:"auth_token,omitempty"`
	CT0       string         `json:"ct0,omitempty"`
	Cookies   []*http.Cookie `json:"cookies,omitempty"`
	SavedAt   time.Time      `json:"saved_at"`
}

// SaveSession saves session data to a JSON file.
func SaveSession(path string, username string, authToken, ct0 string, cookies []*http.Cookie) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}

	sess := Session{
		Username:  username,
		AuthToken: authToken,
		CT0:       ct0,
		Cookies:   cookies,
		SavedAt:   time.Now(),
	}

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return os.WriteFile(path, data, 0o600)
}

// LoadSession loads session data from a JSON file.
func LoadSession(path string) (*Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}

	return &sess, nil
}

// SaveProfile saves a user profile to a JSON file.
func SaveProfile(cfg Config, profile *Profile) error {
	path := cfg.ProfilePath(profile.Username)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profile: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// LoadProfile loads a user profile from a JSON file.
func LoadProfile(cfg Config, username string) (*Profile, error) {
	path := cfg.ProfilePath(username)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
