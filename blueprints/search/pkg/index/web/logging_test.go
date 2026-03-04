package web

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSkipRequestLog(t *testing.T) {
	cases := []struct {
		path string
		skip bool
	}{
		{path: "/ws", skip: true},
		{path: "/api/overview", skip: true},
		{path: "/api/meta/status", skip: true},
		{path: "/api/meta/refresh", skip: true},
		{path: "/api/jobs", skip: false},
	}

	for _, tc := range cases {
		if got := skipRequestLog(tc.path); got != tc.skip {
			t.Fatalf("skipRequestLog(%q) = %v, want %v", tc.path, got, tc.skip)
		}
	}
}

func TestWithRequestLogging_SkipsNoisyMetaEndpoints(t *testing.T) {
	var buf bytes.Buffer
	prev := dashboardLogger
	dashboardLogger = log.New(&buf, "", 0)
	t.Cleanup(func() { dashboardLogger = prev })

	handler := withRequestLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))

	for _, path := range []string{"/api/overview", "/api/meta/status", "/api/meta/refresh"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	if got := buf.String(); strings.Contains(got, "/api/meta/") || strings.Contains(got, "/api/overview") {
		t.Fatalf("expected no noisy endpoint logs, got %q", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if got := buf.String(); !strings.Contains(got, "path=/api/jobs") {
		t.Fatalf("expected normal api log for /api/jobs, got %q", got)
	}
}
