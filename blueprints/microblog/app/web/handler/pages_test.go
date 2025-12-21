package handler_test

import (
	"context"
	"database/sql"
	"html/template"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/app/web/handler"
	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/feature/notifications"
	"github.com/go-mizu/blueprints/microblog/feature/posts"
	"github.com/go-mizu/blueprints/microblog/feature/relationships"
	"github.com/go-mizu/blueprints/microblog/feature/timelines"
	"github.com/go-mizu/blueprints/microblog/feature/trending"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

func setupPagesTestEnv(t *testing.T) (*sql.DB, *template.Template, accounts.API, posts.API, timelines.API, relationships.API, notifications.API, trending.API, func()) {
	t.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	store, err := duckdb.New(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Close()
		t.Fatalf("failed to initialize schema: %v", err)
	}

	accountsStore := duckdb.NewAccountsStore(db)
	postsStore := duckdb.NewPostsStore(db)
	timelinesStore := duckdb.NewTimelinesStore(db)
	relationshipsStore := duckdb.NewRelationshipsStore(db)
	notificationsStore := duckdb.NewNotificationsStore(db)
	trendingStore := duckdb.NewTrendingStore(db)

	accountsSvc := accounts.NewService(accountsStore)
	postsSvc := posts.NewService(postsStore, accountsSvc)
	timelinesSvc := timelines.NewService(timelinesStore, accountsSvc)
	relationshipsSvc := relationships.NewService(relationshipsStore)
	notificationsSvc := notifications.NewService(notificationsStore, accountsSvc)
	trendingSvc := trending.NewService(trendingStore)

	// Create minimal templates for testing
	tmpl := template.New("test")
	tmpl.New("home").Parse(`<!DOCTYPE html><html><body>Home</body></html>`)
	tmpl.New("login").Parse(`<!DOCTYPE html><html><body>Login</body></html>`)
	tmpl.New("register").Parse(`<!DOCTYPE html><html><body>Register</body></html>`)
	tmpl.New("profile").Parse(`<!DOCTYPE html><html><body>Profile</body></html>`)
	tmpl.New("post").Parse(`<!DOCTYPE html><html><body>Post</body></html>`)
	tmpl.New("tag").Parse(`<!DOCTYPE html><html><body>Tag</body></html>`)
	tmpl.New("explore").Parse(`<!DOCTYPE html><html><body>Explore</body></html>`)
	tmpl.New("notifications").Parse(`<!DOCTYPE html><html><body>Notifications</body></html>`)
	tmpl.New("bookmarks").Parse(`<!DOCTYPE html><html><body>Bookmarks</body></html>`)
	tmpl.New("search").Parse(`<!DOCTYPE html><html><body>Search</body></html>`)
	tmpl.New("settings").Parse(`<!DOCTYPE html><html><body>Settings</body></html>`)
	tmpl.New("follow_list").Parse(`<!DOCTYPE html><html><body>Follow List</body></html>`)

	cleanup := func() {
		db.Close()
	}

	return db, tmpl, accountsSvc, postsSvc, timelinesSvc, relationshipsSvc, notificationsSvc, trendingSvc, cleanup
}

func TestPage_Home(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/", nil, "")

	if err := h.Home(ctx); err != nil {
		t.Fatalf("Home() error = %v", err)
	}

	// Should render HTML
	if rec.Code != 200 && rec.Code != 0 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPage_HomeAuthenticated(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	optionalAuth := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/", nil, account.ID)

	if err := h.Home(ctx); err != nil {
		t.Fatalf("Home() error = %v", err)
	}

	if rec.Code != 200 && rec.Code != 0 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPage_Login(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/login", nil, "")

	if err := h.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if rec.Code != 200 && rec.Code != 0 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPage_Register(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/register", nil, "")

	if err := h.Register(ctx); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if rec.Code != 200 && rec.Code != 0 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPage_Profile(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	_, _ = accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/u/testuser", nil, "")
	ctx.Request().SetPathValue("username", "testuser")

	if err := h.Profile(ctx); err != nil {
		t.Fatalf("Profile() error = %v", err)
	}

	if rec.Code != 200 && rec.Code != 0 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPage_ProfileNotFound(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/u/nonexistent", nil, "")
	ctx.Request().SetPathValue("username", "nonexistent")

	if err := h.Profile(ctx); err != nil {
		t.Fatalf("Profile() error = %v", err)
	}

	if rec.Code != 404 && rec.Code != 0 {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestPage_Post(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	post, _ := postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Test post",
	})

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/u/testuser/post/"+post.ID, nil, "")
	ctx.Request().SetPathValue("id", post.ID)

	if err := h.Post(ctx); err != nil {
		t.Fatalf("Post() error = %v", err)
	}

	if rec.Code != 200 && rec.Code != 0 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPage_Tag(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/tags/golang", nil, "")
	ctx.Request().SetPathValue("tag", "golang")

	if err := h.Tag(ctx); err != nil {
		t.Fatalf("Tag() error = %v", err)
	}

	if rec.Code != 200 && rec.Code != 0 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPage_Explore(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/explore", nil, "")

	if err := h.Explore(ctx); err != nil {
		t.Fatalf("Explore() error = %v", err)
	}

	if rec.Code != 200 && rec.Code != 0 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPage_Notifications(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	optionalAuth := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/notifications", nil, account.ID)

	if err := h.Notifications(ctx); err != nil {
		t.Fatalf("Notifications() error = %v", err)
	}

	if rec.Code != 200 && rec.Code != 0 && rec.Code != 302 {
		t.Errorf("expected status 200 or 302, got %d", rec.Code)
	}
}

func TestPage_NotificationsUnauthenticated(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/notifications", nil, "")

	if err := h.Notifications(ctx); err != nil {
		t.Fatalf("Notifications() error = %v", err)
	}

	// Should redirect to login
	if rec.Code != 302 && rec.Code != 0 {
		t.Errorf("expected status 302 (redirect), got %d", rec.Code)
	}
}

func TestPage_Bookmarks(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	optionalAuth := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/bookmarks", nil, account.ID)

	if err := h.Bookmarks(ctx); err != nil {
		t.Fatalf("Bookmarks() error = %v", err)
	}

	if rec.Code != 200 && rec.Code != 0 && rec.Code != 302 {
		t.Errorf("expected status 200 or 302, got %d", rec.Code)
	}
}

func TestPage_Search(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/search?q=test", nil, "")
	ctx.Request().URL.RawQuery = "q=test"

	if err := h.Search(ctx); err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if rec.Code != 200 && rec.Code != 0 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPage_Settings(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	optionalAuth := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/settings", nil, account.ID)

	if err := h.Settings(ctx); err != nil {
		t.Fatalf("Settings() error = %v", err)
	}

	if rec.Code != 200 && rec.Code != 0 && rec.Code != 302 {
		t.Errorf("expected status 200 or 302, got %d", rec.Code)
	}
}

func TestPage_FollowList(t *testing.T) {
	_, tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, cleanup := setupPagesTestEnv(t)
	defer cleanup()

	_, _ = accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relSvc, notifSvc, trendingSvc, optionalAuth, true)

	rec, ctx := testRequest("GET", "/u/testuser/followers", nil, "")
	ctx.Request().SetPathValue("username", "testuser")
	ctx.Request().SetPathValue("type", "followers")

	if err := h.FollowList(ctx); err != nil {
		t.Fatalf("FollowList() error = %v", err)
	}

	if rec.Code != 200 && rec.Code != 0 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
