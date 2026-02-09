package x

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDBInsertAndQuery(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	// Insert tweets
	tweets := []Tweet{
		{
			ID:        "1001",
			Text:      "Hello from Go! #golang",
			Username:  "karpathy",
			UserID:    "12345",
			Name:      "Andrej Karpathy",
			Likes:     5000,
			Retweets:  800,
			Replies:   200,
			Views:     100000,
			Hashtags:  []string{"golang"},
			Photos:    []string{"https://pbs.twimg.com/photo1.jpg"},
			PostedAt:  time.Now().Add(-24 * time.Hour),
			FetchedAt: time.Now(),
		},
		{
			ID:        "1002",
			Text:      "Another tweet about AI",
			Username:  "karpathy",
			UserID:    "12345",
			Name:      "Andrej Karpathy",
			Likes:     10000,
			Retweets:  2000,
			Replies:   500,
			Views:     500000,
			IsRetweet: false,
			PostedAt:  time.Now().Add(-48 * time.Hour),
			FetchedAt: time.Now(),
		},
	}

	if err := db.InsertTweets(tweets); err != nil {
		t.Fatalf("InsertTweets: %v", err)
	}

	// Get stats
	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.Tweets != 2 {
		t.Errorf("expected 2 tweets, got %d", stats.Tweets)
	}

	// Top tweets
	top, err := db.TopTweets(5)
	if err != nil {
		t.Fatalf("TopTweets: %v", err)
	}
	if len(top) != 2 {
		t.Errorf("expected 2 top tweets, got %d", len(top))
	}
	if top[0].Likes != 10000 {
		t.Errorf("expected top tweet with 10000 likes, got %d", top[0].Likes)
	}

	// Check JSON arrays roundtrip
	if len(top[1].Hashtags) != 1 || top[1].Hashtags[0] != "golang" {
		t.Errorf("expected hashtags [golang], got %v", top[1].Hashtags)
	}
	if len(top[1].Photos) != 1 {
		t.Errorf("expected 1 photo, got %d", len(top[1].Photos))
	}

	// Insert user
	profile := &Profile{
		ID:             "12345",
		Username:       "karpathy",
		Name:           "Andrej Karpathy",
		Biography:      "Building AI",
		FollowersCount: 1000000,
		FollowingCount: 500,
		TweetsCount:    3000,
		IsVerified:     true,
		FetchedAt:      time.Now(),
	}

	if err := db.InsertUser(profile); err != nil {
		t.Fatalf("InsertUser: %v", err)
	}

	stats, _ = db.GetStats()
	if stats.Users != 1 {
		t.Errorf("expected 1 user, got %d", stats.Users)
	}

	// Insert follow users
	followUsers := []FollowUser{
		{ID: "100", Username: "user1", Name: "User 1", FollowersCount: 100},
		{ID: "101", Username: "user2", Name: "User 2", FollowersCount: 200},
	}
	if err := db.InsertFollowUsers(followUsers); err != nil {
		t.Fatalf("InsertFollowUsers: %v", err)
	}

	stats, _ = db.GetStats()
	if stats.Users != 3 {
		t.Errorf("expected 3 users, got %d", stats.Users)
	}

	// Verify DB size
	if stats.DBSize == 0 {
		t.Error("expected non-zero DB size")
	}

	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected DB file to exist")
	}
}

func TestDBUpsert(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	// Insert tweet
	tweets := []Tweet{
		{ID: "1001", Text: "v1", Username: "user", Likes: 100, FetchedAt: time.Now()},
	}
	if err := db.InsertTweets(tweets); err != nil {
		t.Fatalf("InsertTweets: %v", err)
	}

	// Update same tweet with more likes
	tweets[0].Likes = 500
	tweets[0].Text = "v2"
	if err := db.InsertTweets(tweets); err != nil {
		t.Fatalf("InsertTweets upsert: %v", err)
	}

	// Verify only 1 tweet and updated
	top, _ := db.TopTweets(10)
	if len(top) != 1 {
		t.Errorf("expected 1 tweet, got %d", len(top))
	}
	if top[0].Likes != 500 {
		t.Errorf("expected 500 likes after upsert, got %d", top[0].Likes)
	}
	if top[0].Text != "v2" {
		t.Errorf("expected text v2, got %s", top[0].Text)
	}
}

func TestSessionSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions", "testuser.json")

	// Save
	err := SaveSession(path, "testuser", "test_auth_token", "test_ct0", nil)
	if err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	// Load
	sess, err := LoadSession(path)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if sess.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", sess.Username)
	}
}

func TestProfileSaveLoad(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{DataDir: dir}

	profile := &Profile{
		ID:             "12345",
		Username:       "karpathy",
		Name:           "Andrej Karpathy",
		FollowersCount: 1000000,
		FetchedAt:      time.Now(),
	}

	if err := SaveProfile(cfg, profile); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	loaded, err := LoadProfile(cfg, "karpathy")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if loaded.Username != "karpathy" {
		t.Errorf("expected username karpathy, got %s", loaded.Username)
	}
	if loaded.FollowersCount != 1000000 {
		t.Errorf("expected 1M followers, got %d", loaded.FollowersCount)
	}
}

func TestDBNewFields(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	// Insert tweet with new fields
	tweets := []Tweet{
		{
			ID:          "2001",
			Text:        "Testing new fields",
			Username:    "testuser",
			Likes:       100,
			Bookmarks:   50,
			Quotes:      25,
			Language:    "en",
			Source:      "Twitter Web App",
			Place:       "San Francisco, CA",
			ReplyToUser: "otheruser",
			IsEdited:    true,
			FetchedAt:   time.Now(),
		},
	}
	if err := db.InsertTweets(tweets); err != nil {
		t.Fatalf("InsertTweets: %v", err)
	}

	// Query back
	top, err := db.TopTweets(1)
	if err != nil {
		t.Fatalf("TopTweets: %v", err)
	}
	if len(top) != 1 {
		t.Fatalf("expected 1 tweet, got %d", len(top))
	}
	tw := top[0]
	if tw.Bookmarks != 50 {
		t.Errorf("bookmarks: got %d, want 50", tw.Bookmarks)
	}
	if tw.Quotes != 25 {
		t.Errorf("quotes: got %d, want 25", tw.Quotes)
	}
	if tw.Language != "en" {
		t.Errorf("language: got %q, want en", tw.Language)
	}
	if tw.Source != "Twitter Web App" {
		t.Errorf("source: got %q, want Twitter Web App", tw.Source)
	}
	if tw.Place != "San Francisco, CA" {
		t.Errorf("place: got %q, want San Francisco, CA", tw.Place)
	}
	if tw.ReplyToUser != "otheruser" {
		t.Errorf("reply_to_user: got %q, want otheruser", tw.ReplyToUser)
	}
	if !tw.IsEdited {
		t.Error("is_edited: got false, want true")
	}

	// Insert user with new fields
	profile := &Profile{
		ID:                   "99999",
		Username:             "testpro",
		Name:                 "Test Professional",
		URL:                  "https://t.co/abc",
		PinnedTweetIDs:       []string{"1001", "1002"},
		ProfessionalType:     "Business",
		ProfessionalCategory: "Technology",
		CanDM:                true,
		DefaultProfile:       false,
		DefaultAvatar:        false,
		DescriptionURLs:      []string{"https://example.com"},
		FetchedAt:            time.Now(),
	}
	if err := db.InsertUser(profile); err != nil {
		t.Fatalf("InsertUser: %v", err)
	}

	stats, _ := db.GetStats()
	if stats.Users != 1 {
		t.Errorf("expected 1 user, got %d", stats.Users)
	}
}

func TestSanitizeDirName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"golang", "golang"},
		{"hello world", "hello_world"},
		{"#golang", "_golang"},
		{"from:karpathy", "from_karpathy"},
		{"", "_"},
		{"a/b\\c", "a_b_c"},
	}
	for _, tc := range tests {
		got := sanitizeDirName(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizeDirName(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
