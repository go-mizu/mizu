package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/feature/notifications"
	"github.com/go-mizu/blueprints/microblog/feature/posts"
	"github.com/go-mizu/blueprints/microblog/feature/relationships"
	"github.com/go-mizu/blueprints/microblog/feature/search"
	"github.com/go-mizu/blueprints/microblog/feature/trending"
)

// Mock implementations

type mockAccountsAPI struct {
	accounts map[string]*accounts.Account
	sessions map[string]*accounts.Session
}

func newMockAccountsAPI() *mockAccountsAPI {
	return &mockAccountsAPI{
		accounts: make(map[string]*accounts.Account),
		sessions: make(map[string]*accounts.Session),
	}
}

func (m *mockAccountsAPI) Create(ctx context.Context, in *accounts.CreateIn) (*accounts.Account, error) {
	a := &accounts.Account{
		ID:          "test-id-123",
		Username:    in.Username,
		DisplayName: in.DisplayName,
		Email:       in.Email,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.accounts[a.ID] = a
	return a, nil
}

func (m *mockAccountsAPI) GetByID(ctx context.Context, id string) (*accounts.Account, error) {
	if a, ok := m.accounts[id]; ok {
		return a, nil
	}
	return nil, accounts.ErrNotFound
}

func (m *mockAccountsAPI) GetByUsername(ctx context.Context, username string) (*accounts.Account, error) {
	for _, a := range m.accounts {
		if a.Username == username {
			return a, nil
		}
	}
	return nil, accounts.ErrNotFound
}

func (m *mockAccountsAPI) GetByEmail(ctx context.Context, email string) (*accounts.Account, error) {
	for _, a := range m.accounts {
		if a.Email == email {
			return a, nil
		}
	}
	return nil, accounts.ErrNotFound
}

func (m *mockAccountsAPI) Update(ctx context.Context, id string, in *accounts.UpdateIn) (*accounts.Account, error) {
	a, ok := m.accounts[id]
	if !ok {
		return nil, accounts.ErrNotFound
	}
	if in.DisplayName != nil {
		a.DisplayName = *in.DisplayName
	}
	if in.Bio != nil {
		a.Bio = *in.Bio
	}
	return a, nil
}

func (m *mockAccountsAPI) Login(ctx context.Context, in *accounts.LoginIn) (*accounts.Session, error) {
	for _, a := range m.accounts {
		if a.Username == in.Username {
			return m.CreateSession(ctx, a.ID)
		}
	}
	return nil, accounts.ErrNotFound
}

func (m *mockAccountsAPI) CreateSession(ctx context.Context, accountID string) (*accounts.Session, error) {
	s := &accounts.Session{
		ID:        "session-123",
		AccountID: accountID,
		Token:     "test-token-abc",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	m.sessions[s.Token] = s
	return s, nil
}

func (m *mockAccountsAPI) GetSession(ctx context.Context, token string) (*accounts.Session, error) {
	if s, ok := m.sessions[token]; ok {
		return s, nil
	}
	return nil, accounts.ErrInvalidSession
}

func (m *mockAccountsAPI) DeleteSession(ctx context.Context, token string) error {
	delete(m.sessions, token)
	return nil
}

func (m *mockAccountsAPI) Verify(ctx context.Context, id string, verified bool) error { return nil }
func (m *mockAccountsAPI) Suspend(ctx context.Context, id string, suspended bool) error { return nil }
func (m *mockAccountsAPI) SetAdmin(ctx context.Context, id string, admin bool) error   { return nil }
func (m *mockAccountsAPI) List(ctx context.Context, limit, offset int) (*accounts.AccountList, error) {
	return &accounts.AccountList{}, nil
}
func (m *mockAccountsAPI) Search(ctx context.Context, query string, limit int) ([]*accounts.Account, error) {
	return nil, nil
}

type mockPostsAPI struct {
	posts map[string]*posts.Post
}

func newMockPostsAPI() *mockPostsAPI {
	return &mockPostsAPI{posts: make(map[string]*posts.Post)}
}

func (m *mockPostsAPI) Create(ctx context.Context, accountID string, in *posts.CreateIn) (*posts.Post, error) {
	p := &posts.Post{
		ID:         "post-123",
		AccountID:  accountID,
		Content:    in.Content,
		Visibility: in.Visibility,
		CreatedAt:  time.Now(),
	}
	if p.Visibility == "" {
		p.Visibility = posts.VisibilityPublic
	}
	m.posts[p.ID] = p
	return p, nil
}

func (m *mockPostsAPI) GetByID(ctx context.Context, id, viewerID string) (*posts.Post, error) {
	if p, ok := m.posts[id]; ok {
		return p, nil
	}
	return nil, posts.ErrNotFound
}

func (m *mockPostsAPI) Update(ctx context.Context, id, accountID string, in *posts.UpdateIn) (*posts.Post, error) {
	p, ok := m.posts[id]
	if !ok {
		return nil, posts.ErrNotFound
	}
	if in.Content != nil {
		p.Content = *in.Content
	}
	return p, nil
}

func (m *mockPostsAPI) Delete(ctx context.Context, id, accountID string) error {
	delete(m.posts, id)
	return nil
}

func (m *mockPostsAPI) GetThread(ctx context.Context, id, viewerID string) (*posts.ThreadContext, error) {
	p, ok := m.posts[id]
	if !ok {
		return nil, posts.ErrNotFound
	}
	return &posts.ThreadContext{Post: p}, nil
}

type mockTimelinesAPI struct{}

func (m *mockTimelinesAPI) Home(ctx context.Context, accountID string, limit int, maxID, sinceID string) ([]*posts.Post, error) {
	return nil, nil
}
func (m *mockTimelinesAPI) Local(ctx context.Context, viewerID string, limit int, maxID, sinceID string) ([]*posts.Post, error) {
	return nil, nil
}
func (m *mockTimelinesAPI) Hashtag(ctx context.Context, tag, viewerID string, limit int, maxID, sinceID string) ([]*posts.Post, error) {
	return nil, nil
}
func (m *mockTimelinesAPI) Account(ctx context.Context, accountID, viewerID string, limit int, maxID string, onlyMedia, excludeReplies bool) ([]*posts.Post, error) {
	return nil, nil
}
func (m *mockTimelinesAPI) List(ctx context.Context, listID, viewerID string, limit int, maxID string) ([]*posts.Post, error) {
	return nil, nil
}
func (m *mockTimelinesAPI) Bookmarks(ctx context.Context, accountID string, limit int, maxID string) ([]*posts.Post, error) {
	return nil, nil
}

type mockInteractionsAPI struct{}

func (m *mockInteractionsAPI) Like(ctx context.Context, accountID, postID string) error       { return nil }
func (m *mockInteractionsAPI) Unlike(ctx context.Context, accountID, postID string) error     { return nil }
func (m *mockInteractionsAPI) Repost(ctx context.Context, accountID, postID string) error     { return nil }
func (m *mockInteractionsAPI) Unrepost(ctx context.Context, accountID, postID string) error   { return nil }
func (m *mockInteractionsAPI) Bookmark(ctx context.Context, accountID, postID string) error   { return nil }
func (m *mockInteractionsAPI) Unbookmark(ctx context.Context, accountID, postID string) error { return nil }
func (m *mockInteractionsAPI) VotePoll(ctx context.Context, accountID, pollID string, choices []int) error {
	return nil
}
func (m *mockInteractionsAPI) GetLikedBy(ctx context.Context, postID string, limit, offset int) ([]string, error) {
	return nil, nil
}
func (m *mockInteractionsAPI) GetRepostedBy(ctx context.Context, postID string, limit, offset int) ([]string, error) {
	return nil, nil
}

type mockRelationshipsAPI struct{}

func (m *mockRelationshipsAPI) Get(ctx context.Context, accountID, targetID string) (*relationships.Relationship, error) {
	return &relationships.Relationship{ID: targetID}, nil
}
func (m *mockRelationshipsAPI) Follow(ctx context.Context, accountID, targetID string) error {
	return nil
}
func (m *mockRelationshipsAPI) Unfollow(ctx context.Context, accountID, targetID string) error {
	return nil
}
func (m *mockRelationshipsAPI) Block(ctx context.Context, accountID, targetID string) error {
	return nil
}
func (m *mockRelationshipsAPI) Unblock(ctx context.Context, accountID, targetID string) error {
	return nil
}
func (m *mockRelationshipsAPI) Mute(ctx context.Context, accountID, targetID string, hideNotifications bool, duration *time.Duration) error {
	return nil
}
func (m *mockRelationshipsAPI) Unmute(ctx context.Context, accountID, targetID string) error {
	return nil
}
func (m *mockRelationshipsAPI) GetFollowers(ctx context.Context, targetID string, limit, offset int) ([]string, error) {
	return nil, nil
}
func (m *mockRelationshipsAPI) GetFollowing(ctx context.Context, targetID string, limit, offset int) ([]string, error) {
	return nil, nil
}
func (m *mockRelationshipsAPI) CountFollowers(ctx context.Context, accountID string) (int, error) {
	return 0, nil
}
func (m *mockRelationshipsAPI) CountFollowing(ctx context.Context, accountID string) (int, error) {
	return 0, nil
}
func (m *mockRelationshipsAPI) GetBlocked(ctx context.Context, accountID string, limit, offset int) ([]string, error) {
	return nil, nil
}
func (m *mockRelationshipsAPI) GetMuted(ctx context.Context, accountID string, limit, offset int) ([]string, error) {
	return nil, nil
}
func (m *mockRelationshipsAPI) IsBlocked(ctx context.Context, accountID, targetID string) (bool, error) {
	return false, nil
}
func (m *mockRelationshipsAPI) IsMuted(ctx context.Context, accountID, targetID string) (bool, error) {
	return false, nil
}

type mockNotificationsAPI struct{}

func (m *mockNotificationsAPI) List(ctx context.Context, accountID string, types []notifications.NotificationType, limit int, maxID, sinceID string, excludeTypes []notifications.NotificationType) ([]*notifications.Notification, error) {
	return nil, nil
}
func (m *mockNotificationsAPI) Get(ctx context.Context, id, accountID string) (*notifications.Notification, error) {
	return nil, nil
}
func (m *mockNotificationsAPI) MarkAsRead(ctx context.Context, id, accountID string) error { return nil }
func (m *mockNotificationsAPI) MarkAllAsRead(ctx context.Context, accountID string) error  { return nil }
func (m *mockNotificationsAPI) Dismiss(ctx context.Context, id, accountID string) error    { return nil }
func (m *mockNotificationsAPI) DismissAll(ctx context.Context, accountID string) error     { return nil }
func (m *mockNotificationsAPI) CountUnread(ctx context.Context, accountID string) (int, error) {
	return 0, nil
}
func (m *mockNotificationsAPI) CleanOld(ctx context.Context, olderThan time.Duration) (int64, error) {
	return 0, nil
}

type mockSearchAPI struct{}

func (m *mockSearchAPI) Search(ctx context.Context, query string, types []search.ResultType, limit int, viewerID string) ([]*search.Result, error) {
	return nil, nil
}
func (m *mockSearchAPI) SearchPosts(ctx context.Context, query string, limit int, maxID, sinceID, viewerID string) ([]string, error) {
	return nil, nil
}
func (m *mockSearchAPI) SearchAccounts(ctx context.Context, query string, limit int) ([]string, error) {
	return nil, nil
}

type mockTrendingAPI struct{}

func (m *mockTrendingAPI) Tags(ctx context.Context, limit int) ([]*trending.TrendingTag, error) {
	return nil, nil
}
func (m *mockTrendingAPI) Posts(ctx context.Context, limit int) ([]string, error) { return nil, nil }
func (m *mockTrendingAPI) SuggestedAccounts(ctx context.Context, accountID string, limit int) ([]string, error) {
	return nil, nil
}

// Helper to create test server

func newTestServer() *Server {
	return &Server{
		app:           mizu.New(),
		accounts:      newMockAccountsAPI(),
		posts:         newMockPostsAPI(),
		timelines:     &mockTimelinesAPI{},
		interactions:  &mockInteractionsAPI{},
		relationships: &mockRelationshipsAPI{},
		notifications: &mockNotificationsAPI{},
		search:        &mockSearchAPI{},
		trending:      &mockTrendingAPI{},
	}
}

// Tests

func TestHandleRegister(t *testing.T) {
	s := newTestServer()

	app := mizu.New()
	app.Post("/api/v1/auth/register", s.handleRegister)

	body := `{"username":"testuser","email":"test@example.com","password":"secret123"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data in response")
	}

	if _, ok := data["token"]; !ok {
		t.Error("expected token in response")
	}

	account, ok := data["account"].(map[string]any)
	if !ok {
		t.Fatal("expected account in response")
	}

	if account["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got %v", account["username"])
	}
}

func TestHandleLogin(t *testing.T) {
	s := newTestServer()
	mockAccounts := s.accounts.(*mockAccountsAPI)

	// Create a test account first
	mockAccounts.accounts["test-id"] = &accounts.Account{
		ID:       "test-id",
		Username: "testuser",
		Email:    "test@example.com",
	}

	app := mizu.New()
	app.Post("/api/v1/auth/login", s.handleLogin)

	body := `{"username":"testuser","password":"secret123"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetAccount(t *testing.T) {
	s := newTestServer()
	mockAccounts := s.accounts.(*mockAccountsAPI)

	// Create a test account
	mockAccounts.accounts["test-id-123456789012345678901234"] = &accounts.Account{
		ID:          "test-id-123456789012345678901234",
		Username:    "testuser",
		DisplayName: "Test User",
	}

	app := mizu.New()
	app.Get("/api/v1/accounts/{id}", s.handleGetAccount)

	req := httptest.NewRequest("GET", "/api/v1/accounts/test-id-123456789012345678901234", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data in response")
	}

	if data["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got %v", data["username"])
	}
}

func TestHandleCreatePost(t *testing.T) {
	s := newTestServer()
	mockAccounts := s.accounts.(*mockAccountsAPI)

	// Create a test session
	mockAccounts.accounts["test-id"] = &accounts.Account{ID: "test-id", Username: "testuser"}
	mockAccounts.sessions["test-token"] = &accounts.Session{
		ID:        "session-123",
		AccountID: "test-id",
		Token:     "test-token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	app := mizu.New()
	app.Post("/api/v1/posts", s.authRequired(s.handleCreatePost))

	body := `{"content":"Hello, world!"}`
	req := httptest.NewRequest("POST", "/api/v1/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data in response")
	}

	if data["content"] != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got %v", data["content"])
	}
}

func TestHandleGetPost(t *testing.T) {
	s := newTestServer()
	mockPosts := s.posts.(*mockPostsAPI)

	// Create a test post
	mockPosts.posts["post-123"] = &posts.Post{
		ID:        "post-123",
		AccountID: "test-id",
		Content:   "Test content",
	}

	app := mizu.New()
	app.Get("/api/v1/posts/{id}", s.handleGetPost)

	req := httptest.NewRequest("GET", "/api/v1/posts/post-123", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data in response")
	}

	if data["content"] != "Test content" {
		t.Errorf("expected content 'Test content', got %v", data["content"])
	}
}

func TestHandleGetPostNotFound(t *testing.T) {
	s := newTestServer()

	app := mizu.New()
	app.Get("/api/v1/posts/{id}", s.handleGetPost)

	req := httptest.NewRequest("GET", "/api/v1/posts/nonexistent", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestAuthRequiredMiddleware(t *testing.T) {
	s := newTestServer()

	app := mizu.New()
	app.Get("/api/v1/protected", s.authRequired(func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]any{"success": true})
	}))

	// Without auth header
	req := httptest.NewRequest("GET", "/api/v1/protected", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}
