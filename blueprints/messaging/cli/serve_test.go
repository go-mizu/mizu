package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/messaging/app/web"
)

func TestServeCommand(t *testing.T) {
	cmd := NewServe()

	if cmd.Use != "serve" {
		t.Errorf("expected Use to be 'serve', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestModeString(t *testing.T) {
	tests := []struct {
		dev      bool
		expected string
	}{
		{true, "development"},
		{false, "production"},
	}

	for _, tc := range tests {
		result := modeString(tc.dev)
		if result != tc.expected {
			t.Errorf("modeString(%v) = %s, expected %s", tc.dev, result, tc.expected)
		}
	}
}

func TestServerIntegration(t *testing.T) {
	// Test that a server can be created and handle requests
	tmpDir := t.TempDir()

	cfg := web.Config{
		Addr:    ":0", // Use any available port
		DataDir: tmpDir,
		Dev:     true,
	}

	server, err := web.New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer server.Close()

	// Test various endpoints using httptest
	handler := server.Handler()

	t.Run("health check - home page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("login page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/login", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("register page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/register", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("api auth endpoints exist", func(t *testing.T) {
		endpoints := []struct {
			method string
			path   string
		}{
			{http.MethodPost, "/api/v1/auth/register"},
			{http.MethodPost, "/api/v1/auth/login"},
		}

		for _, ep := range endpoints {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			// 400 means the endpoint exists but we didn't provide valid data
			// 404 would mean the route doesn't exist
			if rec.Code == http.StatusNotFound {
				t.Errorf("endpoint %s %s should exist", ep.method, ep.path)
			}
		}
	})
}

func TestServerConfiguration(t *testing.T) {
	t.Run("creates data directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := web.Config{
			Addr:    ":0",
			DataDir: tmpDir + "/subdir/data",
			Dev:     true,
		}

		server, err := web.New(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer server.Close()
	})

	t.Run("dev mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := web.Config{
			Addr:    ":0",
			DataDir: tmpDir,
			Dev:     true,
		}

		server, err := web.New(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer server.Close()
	})

	t.Run("production mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := web.Config{
			Addr:    ":0",
			DataDir: tmpDir,
			Dev:     false,
		}

		server, err := web.New(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer server.Close()
	})
}

func TestServerGracefulShutdown(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := web.Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	}

	server, err := web.New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Close should not error
	if err := server.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestServerConcurrentRequests(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := web.Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	}

	server, err := web.New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer server.Close()

	handler := server.Handler()

	// Send concurrent requests
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			done <- rec.Code == http.StatusOK
		}()
	}

	// Wait for all requests with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	successCount := 0
	for i := 0; i < 10; i++ {
		select {
		case success := <-done:
			if success {
				successCount++
			}
		case <-ctx.Done():
			t.Fatal("timeout waiting for concurrent requests")
		}
	}

	if successCount != 10 {
		t.Errorf("expected 10 successful requests, got %d", successCount)
	}
}

func TestDefaultDataDir(t *testing.T) {
	dir := defaultDataDir()

	if dir == "" {
		t.Error("defaultDataDir() should not return empty string")
	}
}

func TestGlobalFlags(t *testing.T) {
	// Verify global flags have reasonable defaults
	if addr == "" {
		// addr is set during command execution, just verify it's accessible
		t.Log("addr is empty (expected before command execution)")
	}

	// dataDir default is set in Execute, verify defaultDataDir works
	defaultDir := defaultDataDir()
	if defaultDir == "" {
		t.Error("defaultDataDir should not be empty")
	}
}
