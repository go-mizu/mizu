package insta

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	home, _ := os.UserHomeDir()
	expectedDataDir := filepath.Join(home, "data", "instagram")

	if cfg.DataDir != expectedDataDir {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, expectedDataDir)
	}
	if cfg.Delay <= 0 {
		t.Error("Delay should be positive")
	}
	if cfg.MaxRetry <= 0 {
		t.Error("MaxRetry should be positive")
	}
	if cfg.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}
	if cfg.UserAgent == "" {
		t.Error("UserAgent should not be empty")
	}
	if cfg.Workers <= 0 {
		t.Error("Workers should be positive")
	}
}

func TestConfigPaths(t *testing.T) {
	cfg := Config{DataDir: "/tmp/instagram-test"}

	tests := []struct {
		name   string
		fn     func() string
		expect string
	}{
		{"UserDir", func() string { return cfg.UserDir("nasa") }, "/tmp/instagram-test/nasa"},
		{"UserMediaDir", func() string { return cfg.UserMediaDir("nasa") }, "/tmp/instagram-test/nasa/media"},
		{"UserDBPath", func() string { return cfg.UserDBPath("nasa") }, "/tmp/instagram-test/nasa/posts.duckdb"},
		{"ProfilePath", func() string { return cfg.ProfilePath("nasa") }, "/tmp/instagram-test/nasa/profile.json"},
		{"HashtagDir", func() string { return cfg.HashtagDir("sunset") }, "/tmp/instagram-test/hashtag/sunset"},
		{"HashtagDBPath", func() string { return cfg.HashtagDBPath("sunset") }, "/tmp/instagram-test/hashtag/sunset/posts.duckdb"},
		{"LocationDir", func() string { return cfg.LocationDir("123") }, "/tmp/instagram-test/location/123"},
		{"LocationDBPath", func() string { return cfg.LocationDBPath("123") }, "/tmp/instagram-test/location/123/posts.duckdb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.expect {
				t.Errorf("got %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestSessionPath(t *testing.T) {
	cfg := Config{SessionDir: "/tmp/sessions"}

	path := cfg.SessionPath("myuser")
	if path != "/tmp/sessions/myuser.json" {
		t.Errorf("SessionPath = %q, want /tmp/sessions/myuser.json", path)
	}

	// SessionFile override
	cfg.SessionFile = "/custom/path/session.json"
	path = cfg.SessionPath("myuser")
	if path != "/custom/path/session.json" {
		t.Errorf("SessionPath with override = %q, want /custom/path/session.json", path)
	}
}

func TestConstants(t *testing.T) {
	if WebAppID == "" {
		t.Error("WebAppID should not be empty")
	}
	if !strings.HasPrefix(GraphQLURL, "https://") {
		t.Errorf("GraphQLURL = %q, should start with https://", GraphQLURL)
	}
	if !strings.HasPrefix(LoginURL, "https://") {
		t.Errorf("LoginURL = %q, should start with https://", LoginURL)
	}
	if !strings.HasPrefix(WebProfileURL, "https://") {
		t.Errorf("WebProfileURL = %q, should start with https://", WebProfileURL)
	}
	if PostsPerPage <= 0 {
		t.Error("PostsPerPage should be positive")
	}
	if CommentsPerPage <= 0 {
		t.Error("CommentsPerPage should be positive")
	}
}
