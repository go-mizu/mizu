package insta

import (
	"context"
	"os"
	"testing"
	"time"
)

// Integration tests that hit the real Instagram API.
// Set INSTA_EMAIL and INSTA_PWD environment variables.
// Run with: go test -v -run TestIntegration -count=1

func skipIfNoCredentials(t *testing.T) (string, string) {
	email := os.Getenv("INSTA_EMAIL")
	pwd := os.Getenv("INSTA_PWD")
	if email == "" || pwd == "" {
		t.Skip("INSTA_EMAIL and INSTA_PWD not set")
	}
	return email, pwd
}

func skipIfNoSession(t *testing.T) *Client {
	email, pwd := skipIfNoCredentials(t)

	cfg := DefaultConfig()
	cfg.Delay = 2 * time.Second
	cfg.Timeout = 30 * time.Second

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	if err := client.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Try loading saved session first
	sessionPath := cfg.SessionPath(email)
	if err := client.LoadSessionFile(sessionPath); err == nil {
		t.Logf("Loaded session from %s", sessionPath)
		return client
	}

	// Login fresh
	t.Logf("Logging in as %s...", email)
	err = client.Login(ctx, email, pwd)
	if err != nil {
		var tfa *TwoFactorError
		if ok := func() bool {
			if e, ok := err.(*TwoFactorError); ok {
				tfa = e
				return true
			}
			return false
		}(); ok {
			t.Skipf("2FA required (identifier: %s) - login manually first", tfa.Identifier)
		}
		t.Fatalf("Login: %v", err)
	}

	// Save session for future tests
	if err := client.SaveSession(sessionPath); err != nil {
		t.Logf("Warning: save session: %v", err)
	} else {
		t.Logf("Session saved to %s", sessionPath)
	}

	return client
}

func TestIntegration_Login(t *testing.T) {
	email, pwd := skipIfNoCredentials(t)

	cfg := DefaultConfig()
	cfg.Timeout = 30 * time.Second

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	if err := client.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	t.Logf("Logging in as %s...", email)
	err = client.Login(ctx, email, pwd)
	if err != nil {
		var tfa *TwoFactorError
		if e, ok := err.(*TwoFactorError); ok {
			tfa = e
			t.Logf("2FA required, identifier: %s", tfa.Identifier)
			t.Skip("2FA required - cannot complete automated login")
			return
		}
		t.Fatalf("Login failed: %v", err)
	}

	if !client.IsLoggedIn() {
		t.Fatal("client should be logged in")
	}
	t.Logf("Logged in as %s (ID: %s)", client.Username(), client.userID)

	// Save session
	sessionPath := cfg.SessionPath(email)
	if err := client.SaveSession(sessionPath); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	t.Logf("Session saved to %s", sessionPath)

	// Verify session file exists
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		t.Fatal("session file not created")
	}
}

func TestIntegration_Profile(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Timeout = 30 * time.Second

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	if err := client.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	profile, err := client.GetProfile(ctx, "nasa")
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}

	if profile.Username != "nasa" {
		t.Errorf("Username = %q, want nasa", profile.Username)
	}
	if profile.FollowerCount < 1_000_000 {
		t.Errorf("FollowerCount = %d, expected > 1M", profile.FollowerCount)
	}
	if profile.ID == "" {
		t.Error("ID should not be empty")
	}
	t.Logf("Profile: @%s (%s), %d followers, %d posts",
		profile.Username, profile.FullName, profile.FollowerCount, profile.PostCount)
}

func TestIntegration_ProfileWithPosts(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Timeout = 30 * time.Second

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	if err := client.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	result, err := client.GetProfileWithPosts(ctx, "nasa")
	if err != nil {
		t.Fatalf("GetProfileWithPosts: %v", err)
	}

	if result.Profile.Username != "nasa" {
		t.Errorf("Username = %q, want nasa", result.Profile.Username)
	}
	if len(result.Posts) == 0 {
		t.Fatal("expected at least 1 post")
	}
	if len(result.Posts) > 12 {
		t.Errorf("expected at most 12 posts from profile, got %d", len(result.Posts))
	}

	t.Logf("Got %d posts (hasMore=%v, cursor=%q)", len(result.Posts), result.HasMore, truncate(result.Cursor, 20))

	// Verify posts have data
	for i, p := range result.Posts {
		if p.ID == "" {
			t.Errorf("post[%d] ID is empty", i)
		}
		if p.Shortcode == "" {
			t.Errorf("post[%d] Shortcode is empty", i)
		}
		if p.DisplayURL == "" {
			t.Errorf("post[%d] DisplayURL is empty", i)
		}
	}
}

func TestIntegration_Search(t *testing.T) {
	client := skipIfNoSession(t)

	result, err := client.Search(context.Background(), "golang programming", 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	total := len(result.Users) + len(result.Hashtags) + len(result.Places)
	t.Logf("Search results: %d users, %d hashtags, %d places",
		len(result.Users), len(result.Hashtags), len(result.Places))

	if total == 0 {
		t.Error("expected at least 1 search result")
	}

	for _, u := range result.Users {
		if u.Username == "" {
			t.Error("user has empty username")
		}
	}
}

func TestIntegration_PostsWithAuth(t *testing.T) {
	client := skipIfNoSession(t)

	ctx := context.Background()
	posts, err := client.GetUserPosts(ctx, "golang", 24, func(p Progress) {
		t.Logf("Progress: %s %d/%d", p.Phase, p.Current, p.Total)
	})
	if err != nil {
		t.Logf("Warning: %v", err)
	}

	t.Logf("Got %d posts for @golang", len(posts))

	if len(posts) == 0 {
		t.Fatal("expected at least 1 post")
	}

	// With auth, we should get more than 12 if the account has > 12 posts
	// golang may not have many posts, so just check we got something
	for i, p := range posts {
		if p.ID == "" {
			t.Errorf("post[%d] ID is empty", i)
		}
	}
}

func TestIntegration_Comments(t *testing.T) {
	client := skipIfNoSession(t)

	// First get a post to use
	result, err := client.GetProfileWithPosts(context.Background(), "nasa")
	if err != nil {
		t.Fatalf("GetProfileWithPosts: %v", err)
	}
	if len(result.Posts) == 0 {
		t.Skip("no posts to test comments on")
	}

	shortcode := result.Posts[0].Shortcode
	t.Logf("Fetching comments for post %s", shortcode)

	comments, err := client.GetComments(context.Background(), shortcode, 10, func(p Progress) {
		t.Logf("Comments: %d/%d", p.Current, p.Total)
	})
	if err != nil {
		t.Fatalf("GetComments: %v", err)
	}

	t.Logf("Got %d comments", len(comments))

	for i, c := range comments {
		if c.ID == "" {
			t.Errorf("comment[%d] ID is empty", i)
		}
		if c.Text == "" {
			t.Errorf("comment[%d] Text is empty", i)
		}
	}
}

func TestIntegration_Hashtag(t *testing.T) {
	client := skipIfNoSession(t)

	posts, err := client.GetHashtagPosts(context.Background(), "golang", 12, func(p Progress) {
		t.Logf("Hashtag: %d/%d", p.Current, p.Total)
	})
	if err != nil {
		t.Fatalf("GetHashtagPosts: %v", err)
	}

	t.Logf("Got %d posts for #golang", len(posts))

	if len(posts) == 0 {
		t.Error("expected at least 1 post for #golang")
	}
}
