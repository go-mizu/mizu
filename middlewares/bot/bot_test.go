package bot

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("detect googlebot", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if !info.IsBot {
			t.Error("expected bot detection")
		}
		if info.BotName != "googlebot" {
			t.Errorf("expected 'googlebot', got %q", info.BotName)
		}
		if info.Category != "search" {
			t.Errorf("expected category 'search', got %q", info.Category)
		}
	})

	t.Run("detect curl", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("User-Agent", "curl/7.68.0")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if !info.IsBot {
			t.Error("expected bot detection for curl")
		}
		if info.Category != "tool" {
			t.Errorf("expected category 'tool', got %q", info.Category)
		}
	})

	t.Run("normal browser", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if info.IsBot {
			t.Error("expected non-bot for normal browser")
		}
	})
}

func TestWithOptions_BlockBots(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		BlockBots: true,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("block bot", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("User-Agent", "curl/7.68.0")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("allow browser", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestWithOptions_AllowedBots(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		BlockBots:   true,
		AllowedBots: []string{"googlebot"},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("allowed bot", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("User-Agent", "Googlebot/2.1")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d for allowed bot, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("non-allowed bot", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("User-Agent", "curl/7.68.0")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected %d for non-allowed bot, got %d", http.StatusForbidden, rec.Code)
		}
	})
}

func TestWithOptions_BlockedBots(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		BlockBots:   true,
		BlockedBots: []string{"semrush"},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("blocked bot", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("User-Agent", "SemrushBot")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected %d for blocked bot, got %d", http.StatusForbidden, rec.Code)
		}
	})
}

func TestWithOptions_CustomPatterns(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		CustomPatterns: []string{"mycustombot"},
	}))

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "MyCustomBot/1.0")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !info.IsBot {
		t.Error("expected custom bot detection")
	}
}

func TestWithOptions_ErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		BlockBots: true,
		ErrorHandler: func(c *mizu.Ctx, info *Info) error {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "bot blocked",
				"bot":   info.BotName,
			})
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "curl/7.68.0")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" && contentType != "application/json; charset=utf-8" {
		t.Errorf("expected JSON response from custom handler, got %q", contentType)
	}
}

func TestIsBot(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var isBot bool
	app.Get("/", func(c *mizu.Ctx) error {
		isBot = IsBot(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "Googlebot")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !isBot {
		t.Error("expected IsBot to return true")
	}
}

func TestBotName(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var name string
	app.Get("/", func(c *mizu.Ctx) error {
		name = BotName(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "Bingbot/2.0")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if name != "bingbot" {
		t.Errorf("expected 'bingbot', got %q", name)
	}
}

func TestBlock(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Block())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "Crawler")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestAllow(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Allow("googlebot", "bingbot"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "Googlebot")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestDeny(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Deny("semrush", "ahrefs"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "Semrush")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}
