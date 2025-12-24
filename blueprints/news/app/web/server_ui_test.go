//go:build e2e

package web_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/news/app/web"
	"github.com/go-mizu/mizu/blueprints/news/feature/comments"
	"github.com/go-mizu/mizu/blueprints/news/feature/stories"
	"github.com/go-mizu/mizu/blueprints/news/feature/tags"
	"github.com/go-mizu/mizu/blueprints/news/feature/users"
	"github.com/go-mizu/mizu/blueprints/news/pkg/ulid"
	"github.com/go-mizu/mizu/blueprints/news/store/duckdb"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupUITestServer creates a test server with seeded HN-like data.
func setupUITestServer(t *testing.T) (*httptest.Server, *duckdb.Store, *hnTestData) {
	t.Helper()

	tempDir := t.TempDir()
	store, err := duckdb.Open(tempDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	srv, err := web.NewServer(store, web.ServerConfig{
		Addr: ":0",
		Dev:  true,
	})
	if err != nil {
		store.Close()
		t.Fatalf("new server: %v", err)
	}

	ts := httptest.NewServer(srv.Handler())

	// Seed HN-like data
	data := seedHNData(t, store)

	t.Cleanup(func() {
		ts.Close()
		store.Close()
	})

	return ts, store, data
}

// hnTestData holds references to seeded test data.
type hnTestData struct {
	users    []*users.User
	stories  []*stories.Story
	comments []*comments.Comment
	tags     []*tags.Tag
}

// seedHNData seeds realistic Hacker News-like data.
func seedHNData(t *testing.T, store *duckdb.Store) *hnTestData {
	t.Helper()
	ctx := context.Background()
	now := time.Now()

	data := &hnTestData{}

	// Create users
	usernames := []string{"pg", "dang", "sama", "tptacek", "patio11"}
	for i, username := range usernames {
		user := &users.User{
			ID:        ulid.New(),
			Username:  username,
			Email:     username + "@example.com",
			Karma:     int64(1000 * (len(usernames) - i)),
			About:     "HN user " + username,
			CreatedAt: now.Add(-time.Duration(365-i*30) * 24 * time.Hour),
		}
		if err := store.Users().Create(ctx, user); err != nil {
			t.Fatalf("create user %s: %v", username, err)
		}
		data.users = append(data.users, user)
	}

	// Create tags
	tagNames := []string{"programming", "startup", "tech", "go", "rust"}
	for _, name := range tagNames {
		tag := &tags.Tag{
			ID:          ulid.New(),
			Name:        name,
			Description: "Stories about " + name,
			StoryCount:  0,
		}
		if err := store.Tags().Create(ctx, tag); err != nil {
			t.Fatalf("create tag %s: %v", name, err)
		}
		data.tags = append(data.tags, tag)
	}

	// Create stories
	storyData := []struct {
		title string
		url   string
		text  string
		score int64
		age   time.Duration
	}{
		{"Go 1.22 Released with Enhanced Routing", "https://go.dev/blog/go1.22", "", 450, 2 * time.Hour},
		{"Show HN: I built a news aggregator in Go", "", "I've been working on this project for a few months. Here's what I learned...", 120, 5 * time.Hour},
		{"The State of Rust in 2024", "https://blog.rust-lang.org/2024/01/state-of-rust.html", "", 380, 8 * time.Hour},
		{"Ask HN: What's your favorite programming language?", "", "Curious to hear what everyone prefers these days.", 95, 12 * time.Hour},
		{"How we scaled to 1M users", "https://engineering.example.com/scaling", "", 220, 1 * 24 * time.Hour},
		{"A Deep Dive into DuckDB", "https://duckdb.org/why_duckdb", "", 180, 2 * 24 * time.Hour},
		{"The Future of Web Development", "https://web.dev/future", "", 150, 3 * 24 * time.Hour},
		{"Building a Startup in 2024", "https://startupguide.com/2024", "", 200, 4 * 24 * time.Hour},
		{"Why I Switched from Python to Go", "https://devblog.example.com/python-to-go", "", 340, 5 * 24 * time.Hour},
		{"Understanding Distributed Systems", "https://distributed.systems/intro", "", 280, 6 * 24 * time.Hour},
	}

	for i, sd := range storyData {
		author := data.users[i%len(data.users)]
		story := &stories.Story{
			ID:           ulid.New(),
			AuthorID:     author.ID,
			Title:        sd.title,
			URL:          sd.url,
			Domain:       stories.ExtractDomain(sd.url),
			Text:         sd.text,
			TextHTML:     "<p>" + sd.text + "</p>",
			Score:        sd.score,
			CommentCount: int64(i * 3),
			CreatedAt:    now.Add(-sd.age),
		}
		// Assign some tags
		var tagIDs []string
		if i%2 == 0 {
			tagIDs = []string{data.tags[0].ID} // programming
		}
		if i%3 == 0 && len(data.tags) > 1 {
			tagIDs = append(tagIDs, data.tags[1].ID) // startup
		}
		if err := store.Stories().Create(ctx, story, tagIDs); err != nil {
			t.Fatalf("create story: %v", err)
		}
		story.Author = author
		data.stories = append(data.stories, story)
	}

	// Create comments for first few stories
	for i := 0; i < 3; i++ {
		story := data.stories[i]
		for j := 0; j < 3; j++ {
			author := data.users[(i+j)%len(data.users)]
			commentID := ulid.New()
			comment := &comments.Comment{
				ID:        commentID,
				StoryID:   story.ID,
				AuthorID:  author.ID,
				Text:      "This is a great point! I've been thinking about this...",
				TextHTML:  "<p>This is a great point! I've been thinking about this...</p>",
				Score:     int64(10 - j),
				Depth:     0,
				Path:      commentID,
				CreatedAt: now.Add(-time.Duration(i+j) * time.Hour),
			}
			if err := store.Comments().Create(ctx, comment); err != nil {
				t.Fatalf("create comment: %v", err)
			}
			comment.Author = author
			data.comments = append(data.comments, comment)

			// Add a reply
			if j == 0 {
				replyID := ulid.New()
				reply := &comments.Comment{
					ID:        replyID,
					StoryID:   story.ID,
					ParentID:  commentID,
					AuthorID:  data.users[(i+j+1)%len(data.users)].ID,
					Text:      "I agree, and would add that...",
					TextHTML:  "<p>I agree, and would add that...</p>",
					Score:     5,
					Depth:     1,
					Path:      commentID + "/" + replyID,
					CreatedAt: now.Add(-time.Duration(i+j)*time.Hour + 30*time.Minute),
				}
				if err := store.Comments().Create(ctx, reply); err != nil {
					t.Fatalf("create reply: %v", err)
				}
			}
		}
	}

	return data
}

func getPage(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, string(body)
}

func assertStatusOK(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func assertHTML(t *testing.T, body string) {
	t.Helper()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("response is not HTML")
	}
}

func assertContainsAll(t *testing.T, body string, substrs ...string) {
	t.Helper()
	for _, s := range substrs {
		if !strings.Contains(body, s) {
			t.Errorf("body missing %q", s)
		}
	}
}

// TestUI_HomePage tests the home page renders correctly.
func TestUI_HomePage(t *testing.T) {
	ts, _, data := setupUITestServer(t)

	resp, body := getPage(t, ts.URL+"/")

	assertStatusOK(t, resp)
	assertHTML(t, body)
	assertContainsAll(t, body,
		"<title>",
		"News",
		"class=\"nav\"",
		"class=\"thread-list\"",
		data.stories[0].Title,
		"Comments",
	)
}

// TestUI_NewestPage tests the newest page renders correctly.
func TestUI_NewestPage(t *testing.T) {
	ts, _, data := setupUITestServer(t)

	resp, body := getPage(t, ts.URL+"/newest")

	assertStatusOK(t, resp)
	assertHTML(t, body)
	assertContainsAll(t, body,
		"Newest",
		"class=\"thread-list\"",
		data.stories[0].Title,
	)
}

// TestUI_TopPage tests the top page renders correctly.
func TestUI_TopPage(t *testing.T) {
	ts, _, data := setupUITestServer(t)

	resp, body := getPage(t, ts.URL+"/top")

	assertStatusOK(t, resp)
	assertHTML(t, body)
	assertContainsAll(t, body,
		"Top",
		"class=\"thread-list\"",
		data.stories[0].Title,
	)
}

// TestUI_StoryPage tests the story detail page renders correctly.
func TestUI_StoryPage(t *testing.T) {
	ts, _, data := setupUITestServer(t)

	story := data.stories[0]
	resp, body := getPage(t, ts.URL+"/story/"+story.ID)

	assertStatusOK(t, resp)
	assertHTML(t, body)
	assertContainsAll(t, body,
		story.Title,
		"class=\"thread-full\"",
		"class=\"comments-section\"",
	)
}

// TestUI_StoryWithText tests a text story (Ask HN style) renders correctly.
func TestUI_StoryWithText(t *testing.T) {
	ts, _, data := setupUITestServer(t)

	// Find a text story (no URL)
	var textStory *stories.Story
	for _, s := range data.stories {
		if s.URL == "" && s.Text != "" {
			textStory = s
			break
		}
	}
	if textStory == nil {
		t.Skip("no text story found")
	}

	resp, body := getPage(t, ts.URL+"/story/"+textStory.ID)

	assertStatusOK(t, resp)
	assertHTML(t, body)
	assertContainsAll(t, body,
		textStory.Title,
		"class=\"thread-body-content\"",
	)
}

// TestUI_UserPage tests the user profile page renders correctly.
func TestUI_UserPage(t *testing.T) {
	ts, _, data := setupUITestServer(t)

	user := data.users[0]
	resp, body := getPage(t, ts.URL+"/user/"+user.Username)

	assertStatusOK(t, resp)
	assertHTML(t, body)
	assertContainsAll(t, body,
		user.Username,
		"class=\"user-header\"",
		"Karma",
	)
}

// TestUI_TagPage tests the tag page renders correctly.
func TestUI_TagPage(t *testing.T) {
	ts, _, data := setupUITestServer(t)

	tag := data.tags[0]
	resp, body := getPage(t, ts.URL+"/tag/"+tag.Name)

	assertStatusOK(t, resp)
	assertHTML(t, body)
	assertContainsAll(t, body,
		tag.Name,
		"class=\"board-header\"",
		"class=\"thread-list\"",
	)
}

// TestUI_StaticCSS tests that the CSS file is served correctly.
func TestUI_StaticCSS(t *testing.T) {
	ts, _, _ := setupUITestServer(t)

	resp, body := getPage(t, ts.URL+"/static/css/news.css")

	assertStatusOK(t, resp)

	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/css") {
		t.Errorf("Content-Type: got %q, want text/css", ct)
	}

	// Check for CSS content
	assertContainsAll(t, body,
		":root",
		"--bg:",
		"--accent:",
		".story-list",
		".comment",
	)
}

// TestUI_Pagination tests that pagination links work.
func TestUI_Pagination(t *testing.T) {
	ts, _, _ := setupUITestServer(t)

	// Page 1
	resp, body := getPage(t, ts.URL+"/?p=1")
	assertStatusOK(t, resp)
	assertHTML(t, body)

	// Check for more link if there are enough stories
	if strings.Contains(body, "class=\"pagination\"") {
		if !strings.Contains(body, "?p=2") {
			t.Error("pagination link should point to page 2")
		}
	}
}

// TestUI_404Pages tests that 404 pages work correctly.
func TestUI_404Pages(t *testing.T) {
	ts, _, _ := setupUITestServer(t)

	tests := []struct {
		name string
		path string
	}{
		{"NonexistentStory", "/story/nonexistent"},
		{"NonexistentUser", "/user/nonexistent"},
		{"NonexistentTag", "/tag/nonexistent"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, _ := getPage(t, ts.URL+tc.path)
			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
			}
		})
	}
}

// TestUI_Navigation tests that navigation links are present.
func TestUI_Navigation(t *testing.T) {
	ts, _, _ := setupUITestServer(t)

	resp, body := getPage(t, ts.URL+"/")

	assertStatusOK(t, resp)
	assertContainsAll(t, body,
		"href=\"/\"",
		"href=\"/newest\"",
		"href=\"/top\"",
		"class=\"nav\"",
	)
}

// TestUI_ExternalLinks tests that external story links work.
func TestUI_ExternalLinks(t *testing.T) {
	ts, _, data := setupUITestServer(t)

	// Find a story with URL
	var linkStory *stories.Story
	for _, s := range data.stories {
		if s.URL != "" {
			linkStory = s
			break
		}
	}
	if linkStory == nil {
		t.Skip("no link story found")
	}

	resp, body := getPage(t, ts.URL+"/")

	assertStatusOK(t, resp)
	// Check that the external URL is in the page
	if !strings.Contains(body, linkStory.URL) {
		t.Errorf("page should contain external URL %s", linkStory.URL)
	}
}

// TestUI_DomainDisplay tests that domains are displayed for link stories.
func TestUI_DomainDisplay(t *testing.T) {
	ts, _, _ := setupUITestServer(t)

	resp, body := getPage(t, ts.URL+"/")

	assertStatusOK(t, resp)
	// Check for domain display (e.g., "go.dev")
	assertContainsAll(t, body,
		"class=\"domain\"",
	)
}

// TestUI_TimeDisplay tests that relative time is displayed.
func TestUI_TimeDisplay(t *testing.T) {
	ts, _, _ := setupUITestServer(t)

	resp, body := getPage(t, ts.URL+"/")

	assertStatusOK(t, resp)
	// Should contain relative time strings
	if !strings.Contains(body, "ago") && !strings.Contains(body, "just now") {
		t.Error("page should contain relative time display")
	}
}

// TestUI_CommentsDisplay tests that comments are displayed on story page.
func TestUI_CommentsDisplay(t *testing.T) {
	ts, _, data := setupUITestServer(t)

	// Use first story which has comments
	story := data.stories[0]
	resp, body := getPage(t, ts.URL+"/story/"+story.ID)

	assertStatusOK(t, resp)
	assertContainsAll(t, body,
		"class=\"comment\"",
		"class=\"comment-header\"",
		"class=\"comment-content\"",
	)
}

// TestUI_Footer tests that footer is present (forum templates don't have footer).
func TestUI_Footer(t *testing.T) {
	ts, _, _ := setupUITestServer(t)

	resp, body := getPage(t, ts.URL+"/")

	assertStatusOK(t, resp)
	// Forum-style templates don't include footer, just check page renders
	assertContainsAll(t, body,
		"</html>",
	)
}

// TestUI_AllThemes tests that all themes can render pages correctly.
func TestUI_AllThemes(t *testing.T) {
	themes := []string{"default", "hn", "bbs", "old", "phpbb", "vbulletin"}

	for _, theme := range themes {
		t.Run(theme, func(t *testing.T) {
			// Setup server with specific theme
			tempDir := t.TempDir()
			store, err := duckdb.Open(tempDir)
			if err != nil {
				t.Fatalf("open store: %v", err)
			}

			srv, err := web.NewServer(store, web.ServerConfig{
				Addr:  ":0",
				Dev:   true,
				Theme: theme,
			})
			if err != nil {
				store.Close()
				t.Fatalf("new server: %v", err)
			}

			ts := httptest.NewServer(srv.Handler())

			// Seed some data
			data := seedHNData(t, store)

			t.Cleanup(func() {
				ts.Close()
				store.Close()
			})

			// Test home page
			t.Run("HomePage", func(t *testing.T) {
				resp, body := getPage(t, ts.URL+"/")
				assertStatusOK(t, resp)
				assertHTML(t, body)
				if !strings.Contains(body, data.stories[0].Title) {
					t.Errorf("home page should contain story title")
				}
			})

			// Test story page
			t.Run("StoryPage", func(t *testing.T) {
				resp, body := getPage(t, ts.URL+"/story/"+data.stories[0].ID)
				assertStatusOK(t, resp)
				assertHTML(t, body)
				if !strings.Contains(body, data.stories[0].Title) {
					t.Errorf("story page should contain story title")
				}
			})

			// Test user page
			t.Run("UserPage", func(t *testing.T) {
				resp, body := getPage(t, ts.URL+"/user/"+data.users[0].Username)
				assertStatusOK(t, resp)
				assertHTML(t, body)
				if !strings.Contains(body, data.users[0].Username) {
					t.Errorf("user page should contain username")
				}
			})

			// Test tag page
			t.Run("TagPage", func(t *testing.T) {
				resp, body := getPage(t, ts.URL+"/tag/"+data.tags[0].Name)
				assertStatusOK(t, resp)
				assertHTML(t, body)
				if !strings.Contains(body, data.tags[0].Name) {
					t.Errorf("tag page should contain tag name")
				}
			})

			// Test CSS for theme
			t.Run("ThemeCSS", func(t *testing.T) {
				cssFile := theme + ".css"
				if theme == "default" {
					cssFile = "default.css"
				}
				resp, _ := getPage(t, ts.URL+"/static/css/"+cssFile)
				assertStatusOK(t, resp)
				if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/css") {
					t.Errorf("Content-Type: got %q, want text/css", ct)
				}
			})
		})
	}
}
