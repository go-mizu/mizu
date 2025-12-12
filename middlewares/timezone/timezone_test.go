package timezone

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should default to UTC
	if info.Name != "UTC" {
		t.Errorf("expected UTC default, got %q", info.Name)
	}
}

func TestWithOptions_FromHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Timezone", "America/New_York")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.Name != "America/New_York" {
		t.Errorf("expected America/New_York, got %q", info.Name)
	}
}

func TestWithOptions_FromCookie(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "timezone", Value: "Europe/London"})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.Name != "Europe/London" {
		t.Errorf("expected Europe/London, got %q", info.Name)
	}
}

func TestWithOptions_FromQuery(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?tz=Asia/Tokyo", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.Name != "Asia/Tokyo" {
		t.Errorf("expected Asia/Tokyo, got %q", info.Name)
	}
}

func TestWithOptions_SetCookie(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{SetCookie: true}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Timezone", "America/Chicago")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	var found bool
	for _, cookie := range cookies {
		if cookie.Name == "timezone" && cookie.Value == "America/Chicago" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected timezone cookie to be set")
	}
}

func TestWithOptions_InvalidTimezone(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Timezone", "Invalid/Timezone")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should fall back to default
	if info.Name != "UTC" {
		t.Errorf("expected UTC fallback, got %q", info.Name)
	}
}

func TestWithOptions_CustomDefault(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{Default: "America/Los_Angeles"}))

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.Name != "America/Los_Angeles" {
		t.Errorf("expected America/Los_Angeles, got %q", info.Name)
	}
}

func TestLocation(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var loc *time.Location
	app.Get("/", func(c *mizu.Ctx) error {
		loc = Location(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Timezone", "Europe/Paris")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if loc == nil {
		t.Fatal("expected location to be set")
	}
	if loc.String() != "Europe/Paris" {
		t.Errorf("expected Europe/Paris, got %q", loc.String())
	}
}

func TestName(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var name string
	app.Get("/", func(c *mizu.Ctx) error {
		name = Name(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Timezone", "Australia/Sydney")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if name != "Australia/Sydney" {
		t.Errorf("expected Australia/Sydney, got %q", name)
	}
}

func TestOffset(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var offset int
	app.Get("/", func(c *mizu.Ctx) error {
		offset = Offset(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// UTC has 0 offset
	if offset != 0 {
		t.Errorf("expected 0 offset for UTC, got %d", offset)
	}
}

func TestNow(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var now time.Time
	app.Get("/", func(c *mizu.Ctx) error {
		now = Now(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Timezone", "UTC")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if now.IsZero() {
		t.Error("expected non-zero time")
	}
	if now.Location().String() != "UTC" {
		t.Errorf("expected UTC location, got %q", now.Location().String())
	}
}

func TestFromHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(FromHeader("X-User-TZ"))

	var name string
	app.Get("/", func(c *mizu.Ctx) error {
		name = Name(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-TZ", "Pacific/Auckland")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if name != "Pacific/Auckland" {
		t.Errorf("expected Pacific/Auckland, got %q", name)
	}
}

func TestFromCookie(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(FromCookie("user_tz"))

	var name string
	app.Get("/", func(c *mizu.Ctx) error {
		name = Name(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "user_tz", Value: "America/Denver"})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if name != "America/Denver" {
		t.Errorf("expected America/Denver, got %q", name)
	}
}

func TestWithDefault(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithDefault("Asia/Singapore"))

	var name string
	app.Get("/", func(c *mizu.Ctx) error {
		name = Name(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if name != "Asia/Singapore" {
		t.Errorf("expected Asia/Singapore, got %q", name)
	}
}

func TestLookupPrecedence(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New()) // Default: header, cookie, query

	var name string
	app.Get("/", func(c *mizu.Ctx) error {
		name = Name(c)
		return c.Text(http.StatusOK, "ok")
	})

	// Header should take precedence
	req := httptest.NewRequest(http.MethodGet, "/?tz=Asia/Tokyo", nil)
	req.Header.Set("X-Timezone", "Europe/Berlin")
	req.AddCookie(&http.Cookie{Name: "timezone", Value: "America/Chicago"})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if name != "Europe/Berlin" {
		t.Errorf("expected header to take precedence, got %q", name)
	}
}
