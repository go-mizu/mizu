package xml

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

type User struct {
	XMLName xml.Name `xml:"user"`
	ID      int      `xml:"id"`
	Name    string   `xml:"name"`
}

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestResponse(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/user", func(c *mizu.Ctx) error {
		user := User{ID: 1, Name: "John"}
		return Response(c, http.StatusOK, user)
	})

	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Header().Get("Content-Type"), "application/xml") {
		t.Errorf("expected XML content type, got %q", rec.Header().Get("Content-Type"))
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<user>") {
		t.Errorf("expected XML user element, got %q", body)
	}
	if !strings.Contains(body, "<id>1</id>") {
		t.Errorf("expected XML id element, got %q", body)
	}
}

func TestBind(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{AutoParse: true}))

	app.Post("/user", func(c *mizu.Ctx) error {
		var user User
		if err := Bind(c, &user); err != nil {
			return c.Text(http.StatusBadRequest, err.Error())
		}
		return c.Text(http.StatusOK, user.Name)
	})

	xmlBody := `<?xml version="1.0"?><user><id>1</id><name>Jane</name></user>`
	req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(xmlBody))
	req.Header.Set("Content-Type", "application/xml")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "Jane" {
		t.Errorf("expected Jane, got %q", rec.Body.String())
	}
}

func TestBody(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{AutoParse: true}))

	app.Post("/", func(c *mizu.Ctx) error {
		body := Body(c)
		return c.Text(http.StatusOK, string(body))
	})

	xmlBody := `<data>test</data>`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(xmlBody))
	req.Header.Set("Content-Type", "application/xml")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "test") {
		t.Errorf("expected body to contain test, got %q", rec.Body.String())
	}
}

func TestSendError(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/error", func(c *mizu.Ctx) error {
		return SendError(c, http.StatusNotFound, "resource not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<error>") {
		t.Error("expected error element")
	}
	if !strings.Contains(body, "<code>404</code>") {
		t.Error("expected code element")
	}
	if !strings.Contains(body, "<message>resource not found</message>") {
		t.Error("expected message element")
	}
}

func TestPretty(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Pretty("  "))

	app.Get("/user", func(c *mizu.Ctx) error {
		user := User{ID: 1, Name: "John"}
		return Response(c, http.StatusOK, user)
	})

	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "\n") {
		t.Error("expected pretty printed XML with newlines")
	}
}

func TestContentNegotiation(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(ContentNegotiation())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, PreferredFormat(c))
	})

	t.Run("prefers XML", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept", "application/xml")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "xml" {
			t.Errorf("expected xml, got %q", rec.Body.String())
		}
	})

	t.Run("prefers JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "json" {
			t.Errorf("expected json, got %q", rec.Body.String())
		}
	})

	t.Run("defaults to JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "json" {
			t.Errorf("expected json, got %q", rec.Body.String())
		}
	})
}

func TestRespond(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())
	app.Use(ContentNegotiation())

	type Data struct {
		XMLName xml.Name `xml:"data" json:"-"`
		Value   string   `xml:"value" json:"value"`
	}

	app.Get("/", func(c *mizu.Ctx) error {
		return Respond(c, http.StatusOK, Data{Value: "test"})
	})

	t.Run("responds with XML", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept", "application/xml")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if !strings.Contains(rec.Header().Get("Content-Type"), "xml") {
			t.Errorf("expected XML content type, got %q", rec.Header().Get("Content-Type"))
		}
	})

	t.Run("responds with JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if !strings.Contains(rec.Header().Get("Content-Type"), "json") {
			t.Errorf("expected JSON content type, got %q", rec.Header().Get("Content-Type"))
		}
	})
}

func TestXMLDeclaration(t *testing.T) {
	t.Run("with declaration", func(t *testing.T) {
		app := mizu.NewRouter()
		app.Use(WithOptions(Options{XMLDeclaration: true}))

		app.Get("/", func(c *mizu.Ctx) error {
			return Response(c, http.StatusOK, User{ID: 1, Name: "Test"})
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		body := rec.Body.String()
		if !strings.HasPrefix(body, "<?xml") {
			t.Error("expected XML declaration")
		}
	})

	t.Run("without declaration", func(t *testing.T) {
		app := mizu.NewRouter()
		app.Use(WithOptions(Options{XMLDeclaration: false}))

		app.Get("/", func(c *mizu.Ctx) error {
			return Response(c, http.StatusOK, User{ID: 1, Name: "Test"})
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		body := rec.Body.String()
		if strings.HasPrefix(body, "<?xml") {
			t.Error("expected no XML declaration")
		}
	})
}

func TestWrap(t *testing.T) {
	users := []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	wrapped := Wrap("users", users)
	data, err := xml.Marshal(wrapped)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	result := string(data)
	if !strings.Contains(result, "<users>") {
		t.Error("expected users wrapper element")
	}
}
