package graphql

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Post("/graphql", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	body := `{"query": "{ users { id name } }"}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_MaxDepth(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{MaxDepth: 2}))

	app.Post("/graphql", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Query with depth 4
	body := `{"query": "{ users { posts { comments { author { name } } } } }"}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d for too deep query, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWithOptions_MaxComplexity(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{MaxComplexity: 2}))

	app.Post("/graphql", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Complex query
	body := `{"query": "{ users { id name email } posts { id title content } comments { id text } }"}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d for too complex query, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWithOptions_DisableIntrospection(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{DisableIntrospection: true}))

	app.Post("/graphql", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	body := `{"query": "{ __schema { types { name } } }"}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d for introspection, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWithOptions_BlockedFields(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{BlockedFields: []string{"password", "secret"}}))

	app.Post("/graphql", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	body := `{"query": "{ users { password } }"}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d for blocked field, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSkipNonPOST(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/graphql", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected GET to pass through, got %d", rec.Code)
	}
}

func TestMaxDepth(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(MaxDepth(3))

	app.Post("/graphql", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Depth 2 should pass
	body := `{"query": "{ users { name } }"}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestMaxComplexity(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(MaxComplexity(10))

	app.Post("/graphql", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	body := `{"query": "{ users { id } }"}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestNoIntrospection(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(NoIntrospection())

	app.Post("/graphql", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Normal query should pass
	body := `{"query": "{ users { id } }"}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	// __type introspection should fail
	body = `{"query": "{ __type(name: \"User\") { name } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected introspection to be blocked, got %d", rec.Code)
	}
}

func TestProduction(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Production())

	app.Post("/graphql", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Simple query should pass
	body := `{"query": "{ users { id } }"}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestBlockFields(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(BlockFields("deleteMutation", "adminAccess"))

	app.Post("/graphql", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	body := `{"query": "mutation { deleteMutation { success } }"}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected blocked field rejection, got %d", rec.Code)
	}
}

func TestCustomErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		MaxDepth: 1,
		ErrorHandler: func(c *mizu.Ctx, err error) error {
			return c.JSON(http.StatusForbidden, map[string]string{
				"custom_error": err.Error(),
			})
		},
	}))

	app.Post("/graphql", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	body := `{"query": "{ users { posts { id } } }"}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected custom error code, got %d", rec.Code)
	}
}
