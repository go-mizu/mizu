package multitenancy

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(c *mizu.Ctx) (*Tenant, error) {
		return &Tenant{ID: "test", Name: "Test Tenant"}, nil
	}))

	var capturedTenant *Tenant
	app.Get("/", func(c *mizu.Ctx) error {
		capturedTenant = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedTenant == nil {
		t.Fatal("expected tenant")
	}
	if capturedTenant.ID != "test" {
		t.Errorf("expected ID 'test', got %q", capturedTenant.ID)
	}
}

func TestWithOptions_Required(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Resolver: func(c *mizu.Ctx) (*Tenant, error) {
			return nil, ErrTenantNotFound
		},
		Required: true,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWithOptions_NotRequired(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Resolver: func(c *mizu.Ctx) (*Tenant, error) {
			return nil, ErrTenantNotFound
		},
		Required: false,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		tenant := Get(c)
		if tenant != nil {
			return c.Text(http.StatusOK, tenant.ID)
		}
		return c.Text(http.StatusOK, "no tenant")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "no tenant" {
		t.Errorf("expected 'no tenant', got %q", rec.Body.String())
	}
}

func TestSubdomainResolver(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(SubdomainResolver()))

	var capturedTenant *Tenant
	app.Get("/", func(c *mizu.Ctx) error {
		capturedTenant = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "http://acme.example.com/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedTenant == nil {
		t.Fatal("expected tenant")
	}
	if capturedTenant.ID != "acme" {
		t.Errorf("expected 'acme', got %q", capturedTenant.ID)
	}
}

func TestHeaderResolver(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(HeaderResolver("X-Tenant-Id")))

	var capturedTenant *Tenant
	app.Get("/", func(c *mizu.Ctx) error {
		capturedTenant = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-Id", "tenant-123")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedTenant == nil {
		t.Fatal("expected tenant")
	}
	if capturedTenant.ID != "tenant-123" {
		t.Errorf("expected 'tenant-123', got %q", capturedTenant.ID)
	}
}

func TestPathResolver(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(PathResolver()))

	var capturedTenant *Tenant
	var capturedPath string
	app.Get("/api/users", func(c *mizu.Ctx) error {
		capturedTenant = Get(c)
		capturedPath = c.Request().URL.Path
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/acme/api/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedTenant == nil {
		t.Fatal("expected tenant")
	}
	if capturedTenant.ID != "acme" {
		t.Errorf("expected 'acme', got %q", capturedTenant.ID)
	}
	if capturedPath != "/api/users" {
		t.Errorf("expected '/api/users', got %q", capturedPath)
	}
}

func TestQueryResolver(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(QueryResolver("tenant")))

	var capturedTenant *Tenant
	app.Get("/", func(c *mizu.Ctx) error {
		capturedTenant = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?tenant=my-tenant", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedTenant == nil {
		t.Fatal("expected tenant")
	}
	if capturedTenant.ID != "my-tenant" {
		t.Errorf("expected 'my-tenant', got %q", capturedTenant.ID)
	}
}

func TestLookupResolver(t *testing.T) {
	tenants := map[string]*Tenant{
		"acme": {ID: "acme", Name: "Acme Corp", Metadata: map[string]any{"plan": "premium"}},
	}

	app := mizu.NewRouter()
	app.Use(New(LookupResolver(HeaderResolver("X-Tenant-Id"), func(id string) (*Tenant, error) {
		if tenant, ok := tenants[id]; ok {
			return tenant, nil
		}
		return nil, ErrTenantNotFound
	})))

	var capturedTenant *Tenant
	app.Get("/", func(c *mizu.Ctx) error {
		capturedTenant = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-Id", "acme")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedTenant == nil {
		t.Fatal("expected tenant")
	}
	if capturedTenant.Name != "Acme Corp" {
		t.Errorf("expected 'Acme Corp', got %q", capturedTenant.Name)
	}
	if capturedTenant.Metadata["plan"] != "premium" {
		t.Error("expected metadata")
	}
}

func TestChainResolver(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(ChainResolver(
		HeaderResolver("X-Tenant-Id"),
		QueryResolver("tenant"),
		SubdomainResolver(),
	)))

	var capturedTenant *Tenant
	app.Get("/", func(c *mizu.Ctx) error {
		capturedTenant = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	// Should use query param
	req := httptest.NewRequest(http.MethodGet, "/?tenant=query-tenant", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedTenant == nil {
		t.Fatal("expected tenant")
	}
	if capturedTenant.ID != "query-tenant" {
		t.Errorf("expected 'query-tenant', got %q", capturedTenant.ID)
	}
}

func TestFromContext(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(c *mizu.Ctx) (*Tenant, error) {
		return &Tenant{ID: "test"}, nil
	}))

	var t1, t2 *Tenant
	app.Get("/", func(c *mizu.Ctx) error {
		t1 = Get(c)
		t2 = FromContext(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if t1 != t2 {
		t.Error("Get and FromContext should return same tenant")
	}
}

func TestMustGet(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(c *mizu.Ctx) (*Tenant, error) {
		return &Tenant{ID: "test"}, nil
	}))

	var tenant *Tenant
	app.Get("/", func(c *mizu.Ctx) error {
		tenant = MustGet(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if tenant == nil {
		t.Error("expected tenant from MustGet")
	}
}

func TestMustGet_Panic(t *testing.T) {
	// The router catches panics, so we test that the panic message is correct
	// by using the router with no multitenancy middleware
	app := mizu.NewRouter()

	var panicValue any
	app.Use(func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) (err error) {
			defer func() {
				if r := recover(); r != nil {
					panicValue = r
					panic(r) // Re-panic to let router handle it
				}
			}()
			return next(c)
		}
	})

	app.Get("/", func(c *mizu.Ctx) error {
		_ = MustGet(c) // Should panic
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if panicValue == nil {
		t.Error("expected panic")
	}
	if !strings.Contains(fmt.Sprint(panicValue), "tenant not found") {
		t.Errorf("expected panic message about tenant not found, got: %v", panicValue)
	}
}

func TestErrors(t *testing.T) {
	if ErrTenantNotFound.Error() != "tenant not found" {
		t.Error("unexpected error message")
	}
	if ErrTenantInvalid.Error() != "tenant invalid" {
		t.Error("unexpected error message")
	}
}

func TestWithOptions_ErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Resolver: func(c *mizu.Ctx) (*Tenant, error) {
			return nil, errors.New("custom error")
		},
		Required: true,
		ErrorHandler: func(c *mizu.Ctx, err error) error {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
