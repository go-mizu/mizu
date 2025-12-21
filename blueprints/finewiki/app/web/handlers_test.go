package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/blueprints/finewiki/feature/view"
)

// mockSearchAPI implements search.API for testing.
type mockSearchAPI struct {
	results []search.Result
	err     error
}

func (m *mockSearchAPI) Search(ctx context.Context, q search.Query) ([]search.Result, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

// mockViewAPI implements view.API for testing.
type mockViewAPI struct {
	page *view.Page
	err  error
}

func (m *mockViewAPI) ByID(ctx context.Context, id string) (*view.Page, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.page, nil
}

func (m *mockViewAPI) ByTitle(ctx context.Context, wikiname, title string) (*view.Page, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.page, nil
}

// mockTemplates implements Templates for testing.
type mockTemplates struct {
	rendered string
	err      error
}

func (m *mockTemplates) Render(w any, name string, data any) error {
	if m.err != nil {
		return m.err
	}
	if ww, ok := w.(io.Writer); ok {
		ww.Write([]byte(m.rendered))
	}
	return nil
}

func TestServer_Home(t *testing.T) {
	srv := New(
		&mockViewAPI{},
		&mockSearchAPI{},
		&mockTemplates{rendered: "<html>Home</html>"},
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if !strings.Contains(rec.Body.String(), "Home") {
		t.Errorf("body = %q, want to contain 'Home'", rec.Body.String())
	}
}

func TestServer_Search(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		searchAPI  *mockSearchAPI
		wantStatus int
	}{
		{
			name:  "search with results",
			query: "?q=test",
			searchAPI: &mockSearchAPI{
				results: []search.Result{
					{ID: "1", Title: "Test Result"},
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "search with no results",
			query:      "?q=xyz",
			searchAPI:  &mockSearchAPI{results: []search.Result{}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "search error",
			query:      "?q=test",
			searchAPI:  &mockSearchAPI{err: errors.New("db error")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New(
				&mockViewAPI{},
				tt.searchAPI,
				&mockTemplates{rendered: "<html>Search</html>"},
			)

			req := httptest.NewRequest(http.MethodGet, "/search"+tt.query, nil)
			rec := httptest.NewRecorder()

			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestServer_Page_ByID(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		viewAPI    *mockViewAPI
		wantStatus int
	}{
		{
			name:  "page found by id",
			query: "?id=enwiki/1",
			viewAPI: &mockViewAPI{
				page: &view.Page{ID: "enwiki/1", Title: "Test Page"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "page not found",
			query:      "?id=nonexistent",
			viewAPI:    &mockViewAPI{err: errors.New("not found")},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New(
				tt.viewAPI,
				&mockSearchAPI{},
				&mockTemplates{rendered: "<html>Page</html>"},
			)

			req := httptest.NewRequest(http.MethodGet, "/page"+tt.query, nil)
			rec := httptest.NewRecorder()

			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestServer_Page_ByTitle(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		viewAPI    *mockViewAPI
		wantStatus int
	}{
		{
			name:  "page found by title",
			query: "?wiki=enwiki&title=Test",
			viewAPI: &mockViewAPI{
				page: &view.Page{ID: "enwiki/1", WikiName: "enwiki", Title: "Test"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "page not found",
			query:      "?wiki=enwiki&title=Nonexistent",
			viewAPI:    &mockViewAPI{err: errors.New("not found")},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New(
				tt.viewAPI,
				&mockSearchAPI{},
				&mockTemplates{rendered: "<html>Page</html>"},
			)

			req := httptest.NewRequest(http.MethodGet, "/page"+tt.query, nil)
			rec := httptest.NewRecorder()

			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestServer_Page_MissingParams(t *testing.T) {
	srv := New(
		&mockViewAPI{},
		&mockSearchAPI{},
		&mockTemplates{rendered: "<html>Page</html>"},
	)

	req := httptest.NewRequest(http.MethodGet, "/page", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServer_Healthz(t *testing.T) {
	srv := New(
		&mockViewAPI{},
		&mockSearchAPI{},
		&mockTemplates{},
	)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "ok")
	}
}

// Note: 404 handling for unknown routes is delegated to the mizu framework.
// If custom 404 handling is needed, register a not-found handler.
